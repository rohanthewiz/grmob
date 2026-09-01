package render

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/reconcile"
)

// PatchListener is the Go→native push channel: the native shell implements it
// and registers via Manager.SetListener, and Go calls ApplyPatches whenever
// state changes outside a native event — timer ticks, network responses, any
// goroutine calling State.Set. Without it the bridge is strictly
// request/response and async updates never reach the screen (WASM worked
// around this by polling IsDirty).
//
// The single string-parameter method is deliberate: gomobile bind maps this
// interface onto Java/Kotlin and Objective-C/Swift directly, so an Android
// Activity or iOS view controller can implement it without JNI glue.
//
// ApplyPatches is invoked from a background goroutine. Native implementations
// must hop to their UI thread before touching views (runOnUiThread /
// DispatchQueue.main); the payload is the same patch-array JSON RenderAgain
// returns, and an empty patch set is never pushed.
type PatchListener interface {
	ApplyPatches(patches string)
}

type Manager struct {
	// mu serializes every render pass AND every event dispatch (Dispatch*
	// below). Render passes mutate shared state that cannot tolerate
	// interleaving — the context's hook cursor, its callback registry
	// (BeginRenderPass resets the ID counters), and currentTree — and passes
	// are started from two directions: the native event path (Dispatch*) and
	// the push pump below. Folding dispatch under the same mutex is what
	// marshals app mutations: an event handler can never run in the middle of
	// a pump render pass, so handlers observe settled trees and their writes
	// are rendered atomically by the pass that follows within the same lock
	// hold.
	mu          sync.Mutex
	currentTree *core.Node
	context     *core.Context
	renderFunc  func(*core.Context) core.View

	listenerMu sync.Mutex
	listener   PatchListener

	// renderRequests carries coalesced "state changed" nudges to the pump.
	//
	//   State.Set ──┐                       ┌──> RenderAgain ──> listener
	//   timer tick ─┼─> requestRender ──▷──┤      (mu held)
	//   State.Set ──┘   (buffer of 1,       └──> nothing pending: park
	//                    extra nudges
	//                    dropped)
	//
	// The buffer size of 1 is the coalescing mechanism: a burst of N rapid
	// state writes leaves at most one pending token, and the single pump
	// render that consumes it sees the final state — one diff, one push,
	// instead of N. A nudge arriving mid-render lands in the buffer and
	// triggers one follow-up pass, so the last write is never lost.
	renderRequests chan struct{}
	stop           chan struct{}
	stopOnce       sync.Once
}

func New(ctx *core.Context, rootView func(*core.Context) core.View) *Manager {
	if ctx.Theme() == nil {
		ctx = ctx.WithTheme(core.DefaultTheme)
	}
	m := &Manager{
		context:        ctx,
		renderFunc:     rootView,
		renderRequests: make(chan struct{}, 1),
		stop:           make(chan struct{}),
	}
	// Complete the notification circuit: State.Set / RequestRender fire the
	// context's default render target, which now nudges this manager's pump.
	ctx.OnStateChange(m.requestRender)
	go m.pump()
	return m
}

// Close stops the push pump and closes the app's context tree, which stops
// the background resources hooks registered on it (interval tickers, pending
// timeouts). The Manager is the app-lifetime owner, so its Close is the one
// shutdown entry point: hosts that replace an app (mobile.Register, the WASM
// runtime re-mounting) close the old Manager and thereby cannot leak tickers
// rendering into a dead tree. Normally an app-lifetime singleton, so this
// mainly matters for tests and hot-reload hosts.
func (m *Manager) Close() {
	m.stopOnce.Do(func() {
		close(m.stop)
		m.context.Close()
	})
}

// SetListener attaches (or replaces) the native push target. Any state change
// that happened before attachment is flushed immediately so the listener
// never starts out behind.
func (m *Manager) SetListener(l PatchListener) {
	m.listenerMu.Lock()
	m.listener = l
	m.listenerMu.Unlock()
	m.requestRender()
}

func (m *Manager) getListener() PatchListener {
	m.listenerMu.Lock()
	defer m.listenerMu.Unlock()
	return m.listener
}

// requestRender is the cheap, non-blocking nudge described on renderRequests.
// Safe to call from any goroutine, including from within a render pass.
func (m *Manager) requestRender() {
	select {
	case m.renderRequests <- struct{}{}:
	default: // a render is already pending; this write will be included in it
	}
}

// pump is the single consumer of renderRequests: it turns state-change nudges
// into render passes and pushes non-empty diffs to the listener. One goroutine
// per Manager, started in New, stopped by Close.
func (m *Manager) pump() {
	for {
		select {
		case <-m.stop:
			return
		case <-m.renderRequests:
			// Nothing is mounted before the initial render, and with no
			// listener a pump render would consume the diff and discard it —
			// leaving a polling runtime (which calls RenderAgain itself) with
			// an empty diff and a stale screen. In both cases leave the dirty
			// flag standing; SetListener re-nudges, so nothing is lost.
			if m.getListener() == nil || !m.hasInitialRender() {
				continue
			}
			m.renderAndPush()
		}
	}
}

// renderAndPush runs one pump pass and hands its diff to the listener *inside*
// the render critical section.
//
// Why the delivery is inside the lock, and not after it: the native runtimes
// funnel both patch paths (a Trigger* call's return value and a listener push)
// into one FIFO queue and apply them in arrival order, which is only correct
// if arrival order matches emission order — the contract stated in
// mobile/bridge.go. Patches carry positional paths ("root/2"), so applying
// pass N+1 against the pre-N tree addresses the wrong nodes.
//
// Releasing mu between the render and the push broke exactly that:
//
//	pump (goroutine)            event thread (DispatchCallback)
//	------------------------    -------------------------------
//	lock; render pass N
//	unlock
//	                            lock; handler; render pass N+1; unlock
//	                            queue N+1   <-- arrives first
//	queue N                                 <-- arrives second, stale
//
// Holding mu across the ApplyPatches call makes "produce the diff" and "hand
// it over" one indivisible step, so a dispatch cannot slip a newer pass in
// front of an older one. The listener contract already requires ApplyPatches
// to be cheap and non-reentrant (hop to the UI thread and return), so the
// widened hold does not extend the lock in practice.
func (m *Manager) renderAndPush() {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := m.renderAgainLocked()
	if out == "[]" {
		return
	}
	l := m.getListener()
	if l == nil {
		return
	}
	// Guarded: ApplyPatches crosses into host code (a JS throw over
	// syscall/js, a Java exception surfacing through gomobile), and an
	// unrecovered panic here would kill the pump goroutine outright. The
	// pump is the only consumer of renderRequests, so its death is silent
	// and total — every later State.Set fills the one-slot buffer and is
	// dropped, while taps keep working because they render on their own
	// thread. That reads as "async updates stopped" with no crash to chase.
	if rerr := core.Guard(func() { l.ApplyPatches(out) }); rerr != nil {
		logRenderPanic("patch listener", rerr)
	}
}

func (m *Manager) hasInitialRender() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.currentTree != nil
}

func (r *Manager) RenderInitial() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.renderInitialLocked()
}

// renderInitialLocked is the mount pass; callers must hold mu.
func (r *Manager) renderInitialLocked() string {
	r.context.BeginRenderPass()
	r.context.Reset()

	var tree *core.Node
	if rerr := core.Guard(func() {
		tree = r.renderFunc(r.context).Render(r.context)
	}); rerr != nil {
		// Nothing was mounted, so unlike a later pass there is no last-good
		// tree to fall back to — the screen would be blank and, worse,
		// currentTree would stay nil, which the pump reads as "not mounted
		// yet" and refuses to render forever. Standing in a placeholder tree
		// keeps the app alive and lets a later state change try again.
		logRenderPanic("initial render", rerr)
		tree = panicPlaceholder()
	}
	r.currentTree = tree

	// Close the debug pass boundary (no-op unless core.SetDebugMode is on):
	// the initial pass records each context's baseline hook count for the
	// cursor-drift check on later passes.
	r.context.EndRenderPass()
	return renderJSON(r.currentTree)
}

// panicPlaceholder is the tree a failed pass stands in when there is no
// last-good tree to keep. Built by hand rather than through core's builders so
// it cannot itself run app code or consume hook slots on a context whose pass
// has already gone wrong.
func panicPlaceholder() *core.Node {
	return &core.Node{
		Type:  "Text",
		Props: map[string]any{"content": "Something went wrong."},
		Style: &core.Style{},
	}
}

// logRenderPanic reports a panic that no ErrorBoundary caught.
//
// The Manager logs where core.ErrorBoundary deliberately does not: a boundary
// hands the error to a fallback the app wrote, so the app decides what to do
// with it, but a panic that reaches here has no such owner. Recovering it
// silently would convert a crash — loud, with a stack, reported by every
// crash reporter on the platform — into a screen that quietly stops updating,
// which is a far worse thing to debug.
func logRenderPanic(where string, rerr *core.RenderError) {
	log.Printf("grmob: recovered panic during %s: %v\n%s", where, rerr.Value, rerr.Stack)
}

// RenderAgain ReRender Used after an event (input/click/state change) to get diff
func (r *Manager) RenderAgain() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.renderAgainLocked()
}

// renderAgainLocked is the diff-producing render pass; callers must hold mu.
func (r *Manager) renderAgainLocked() string {
	// BeginRenderPass must precede the render so callback IDs restart from
	// zero and re-registrations line up with the previous pass; the purge
	// after the diff then drops only IDs no longer registered this pass.
	r.context.BeginRenderPass()
	r.context.Reset()

	var newTree *core.Node
	if rerr := core.Guard(func() {
		newTree = r.renderFunc(r.context).Render(r.context)
	}); rerr != nil {
		// Top-level safety net: a panic no core.ErrorBoundary caught. Abandon
		// the pass rather than trying to salvage it — the tree is
		// half-constructed and the only honest thing left on screen is what
		// was there before.
		//
		// Three things are deliberately NOT done here:
		//
		//   currentTree is not replaced, so the last complete tree stays
		//   mounted and the emitted "[]" tells the host nothing changed.
		//
		//   PurgeUnusedCallbacks is not called. Purge keeps only IDs marked
		//   used since BeginRenderPass, and this pass got partway through
		//   marking them; purging on that partial set would delete the
		//   handlers of the tree still on screen and leave every button dead.
		//
		//   EndRenderPass is not called. The debug cursor audit describes a
		//   completed pass; running it over a half-rendered context tree
		//   reports drift that says nothing except that a panic happened,
		//   which the log line above already said.
		//
		// The dirty flag IS cleared: the state change that prompted this pass
		// has been consumed, and a deterministic panic will not render any
		// better on an immediate retry — leaving it set would just make a
		// polling host (WASM) re-panic on every poll.
		logRenderPanic("render pass", rerr)
		r.context.ClearDirty()
		return "[]"
	}

	// Debug-mode audit of the pass that just finished (cursor drift across
	// the context tree); a no-op when debug mode is off.
	r.context.EndRenderPass()
	patches := reconcile.Diff(r.currentTree, newTree, "root")
	r.currentTree = newTree
	r.context.ClearDirty()
	r.context.PurgeUnusedCallbacks()
	if patches == nil {
		// A no-change render must serialize as "[]", not "null": the native
		// runtimes iterate the decoded patch list without a null check.
		patches = []reconcile.Patch{}
	}
	return renderJSON(patches)
}

// DispatchCallback runs the void handler registered under id (a button tap,
// say), then renders and returns the resulting patches — the whole sequence
// under the render mutex, so the handler cannot interleave with a pump render
// pass and its state writes are diffed in the same lock hold. This is the
// event path native bridges should use; the Dispatch*/Trigger split exists
// because dispatching a handler without an immediate render (the async shape)
// is only wanted by hosts that poll or rely purely on the push channel.
//
// Handlers may call State.Set freely: the resulting RequestRender nudge is
// asynchronous (a buffered-channel send plus a goroutine hop), so nothing in
// the handler path re-enters this mutex. The pump's follow-up pass then finds
// an empty diff and pushes nothing.
func (m *Manager) DispatchCallback(id string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.guardHandler(id, func() { m.context.TriggerCallback(id) })
	return m.renderAgainLocked()
}

// DispatchTextCallback is DispatchCallback for string-carrying events.
func (m *Manager) DispatchTextCallback(id string, value string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.guardHandler(id, func() { m.context.TriggerTextCallback(id, value) })
	return m.renderAgainLocked()
}

// DispatchBoolCallback is DispatchCallback for bool-carrying events.
func (m *Manager) DispatchBoolCallback(id string, value bool) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.guardHandler(id, func() { m.context.TriggerBoolCallback(id, value) })
	return m.renderAgainLocked()
}

// DispatchIntCallback is DispatchCallback for int-carrying events.
func (m *Manager) DispatchIntCallback(id string, value int) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.guardHandler(id, func() { m.context.TriggerIntCallback(id, value) })
	return m.renderAgainLocked()
}

// guardHandler runs one event handler under a panic guard.
//
// core.ErrorBoundary covers render; this covers the other half. A handler runs
// on the native event thread with a bridge call in flight, so a panic there
// unwinds straight out through the Go/JNI (or cgo) boundary and kills the
// process — a tap on a button whose handler dereferences a nil is as fatal as
// a panicking Render, and no boundary in the tree can see it, because handlers
// run between passes rather than during one.
//
// Recovery is deliberately partial: the handler's own work is abandoned
// wherever it stopped, so it may have written some of its state and not the
// rest. Nothing here can know what half-applied means for an app, so the pass
// that follows simply renders whatever state actually exists. That is strictly
// better than the alternative — the same half-written state, plus a dead
// process.
func (m *Manager) guardHandler(id string, fn func()) {
	if rerr := core.Guard(fn); rerr != nil {
		logRenderPanic("event handler "+id, rerr)
		if core.IsDebugMode() {
			core.ReportConcern(core.ConcernHandlerPanic,
				fmt.Sprintf("handler %s panicked: %v", id, rerr.Value))
		}
	}
}

// JSON encoder
func renderJSON[T any](v T) string {
	data, err := json.Marshal(v)
	if err != nil {
		return `{"error":"failed to encode JSON"}`
	}
	return string(data)
}

// RenderAndGetPatches renders one pass and returns whichever payload the host
// needs: the full tree when nothing is mounted yet, the diff against the
// mounted tree afterwards. It is the "mount or update, I don't want to know
// which" entry point for a host driving passes by hand; render.Manager's own
// callers use RenderInitial and RenderAgain, which say which they mean.
//
// It delegates rather than re-implementing the two passes, which is the whole
// of the fix here. Its own copy had drifted badly: it reset the root cursor
// with `r.context.Cursor = 0` instead of `Reset()`, so every child scope kept
// its cursor from the previous pass and its slots grew by one per render; and
// it never called PurgeUnusedCallbacks or ClearDirty, so handlers for
// vanished nodes stayed dispatchable and a polling host re-rendered forever.
func (r *Manager) RenderAndGetPatches() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.currentTree == nil {
		return r.renderInitialLocked()
	}
	return r.renderAgainLocked()
}
