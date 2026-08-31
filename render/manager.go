package render

import (
	"encoding/json"
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
			out := m.RenderAgain()
			if out == "[]" {
				continue
			}
			if l := m.getListener(); l != nil {
				l.ApplyPatches(out)
			}
		}
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
	r.context.BeginRenderPass()
	r.context.Reset()
	r.currentTree = r.renderFunc(r.context).Render(r.context)
	// Close the debug pass boundary (no-op unless core.SetDebugMode is on):
	// the initial pass records each context's baseline hook count for the
	// cursor-drift check on later passes.
	r.context.EndRenderPass()
	return renderJSON(r.currentTree)
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
	newTree := r.renderFunc(r.context).Render(r.context)
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
	m.context.TriggerCallback(id)
	return m.renderAgainLocked()
}

// DispatchTextCallback is DispatchCallback for string-carrying events.
func (m *Manager) DispatchTextCallback(id string, value string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.context.TriggerTextCallback(id, value)
	return m.renderAgainLocked()
}

// DispatchBoolCallback is DispatchCallback for bool-carrying events.
func (m *Manager) DispatchBoolCallback(id string, value bool) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.context.TriggerBoolCallback(id, value)
	return m.renderAgainLocked()
}

// DispatchIntCallback is DispatchCallback for int-carrying events.
func (m *Manager) DispatchIntCallback(id string, value int) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.context.TriggerIntCallback(id, value)
	return m.renderAgainLocked()
}

// JSON encoder
func renderJSON[T any](v T) string {
	data, err := json.Marshal(v)
	if err != nil {
		return `{"error":"failed to encode JSON"}`
	}
	return string(data)
}

func (r *Manager) RenderAndGetPatches() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.context.BeginRenderPass()
	r.context.Cursor = 0
	newTree := r.renderFunc(r.context).Render(r.context)
	r.context.EndRenderPass()

	if r.currentTree == nil {
		r.currentTree = newTree
		return render(newTree)
	}

	patches := reconcile.Diff(r.currentTree, newTree, "root")
	r.currentTree = newTree
	if patches == nil {
		patches = []reconcile.Patch{}
	}
	return render(patches)
}

func render[T any](tree T) string {
	data, err := json.Marshal(tree)
	if err != nil {
		return `{"error":"failed to encode render tree"}`
	}
	return string(data)
}
