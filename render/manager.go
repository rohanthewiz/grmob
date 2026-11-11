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
// ApplyPatches is invoked from a background goroutine (the pump), or from the
// thread that called a Dispatch* method, inside that call: with a listener
// attached, a dispatch delivers its pass here rather than returning it (see
// deliverLocked). Either way the render lock is held. Native implementations
// must hop to their UI thread before touching views (runOnUiThread /
// DispatchQueue.main) and return, without waiting on the hop; the payload is
// the same patch-array JSON RenderAgain returns, and an empty patch set is
// never pushed.
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
	m.applyLocked(l, out)
}

// applyLocked hands one pass's diff to the listener; callers must hold mu, and
// that is the whole point (see renderAndPush and deliverLocked). An empty diff
// is never pushed, per PatchListener's contract.
func (m *Manager) applyLocked(l PatchListener, out string) {
	if out == "[]" {
		return
	}
	// Guarded: ApplyPatches crosses into host code (a JS throw over
	// syscall/js, a Java exception surfacing through gomobile), and an
	// unrecovered panic here would kill the pump goroutine outright. The
	// pump is the only consumer of renderRequests, so its death is silent
	// and total — every later State.Set fills the one-slot buffer and is
	// dropped, while taps keep working because they render on their own
	// thread. That reads as "async updates stopped" with no crash to chase.
	// On the dispatch path the same panic would unwind through the bridge
	// call and take the process with it.
	if rerr := core.Guard(func() { l.ApplyPatches(out) }); rerr != nil {
		logRenderPanic("patch listener", rerr)
	}
}

// deliverLocked is how a dispatch hands back the pass it rendered; callers
// must hold mu.
//
// With a listener attached the patches go to the listener here, inside the
// lock hold, and the dispatch returns "[]". Without one (tests, a polling
// host, a shell that has not attached yet) they are returned as before.
//
// # Why not return them
//
// renderAndPush closed one direction of the ordering hole: a dispatch can no
// longer slip its pass in front of a push that is still being handed over. The
// return value left the other direction open. The lock was released when the
// dispatch returned, before the host had queued what it returned, and a pump
// pass started in that gap pushed the NEXT pass first:
//
//	event thread                       pump
//	lock; handler; render N; unlock
//	  (host still holding pass N)      lock; render N+1; push N+1; unlock
//	queue N   <-- applied after N+1, against a tree it was not diffed from
//
// A probe with the hosts' shape (one FIFO fed by both paths) measured 1 to 4
// inversions per 20,000 taps, and the JNI string copy widens the gap on a
// device. Paths are positional, so an inversion writes to the wrong node or
// leaves a stale prop, silently.
//
// Handing the pass over inside the lock makes every pass leave through one
// ordered door, and no host needs a change: GrMobRuntime.kt and
// GrMobRuntime.swift apply the "[]" a Trigger* call now returns as the no-op
// it always was, and their listeners already hop to the main thread and
// return, so the lock is held no longer than a push holds it.
//
// The listener is read once, so a SetListener racing this call cannot leave a
// pass with neither path: it goes to the listener read, or is returned.
func (m *Manager) deliverLocked(out string) string {
	l := m.getListener()
	if l == nil {
		return out
	}
	m.applyLocked(l, out)
	return "[]"
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

	// Close the debug pass boundary (no-op unless core.SetDebugMode is on):
	// the initial pass records each context's baseline hook count for the
	// cursor-drift check on later passes.
	r.context.EndRenderPass()
	// And the accessibility audit over the tree that pass produced —
	// duplicate ids, dangling aria-controls, ids that are not ids, and
	// disclosures nothing can open. It runs here rather than in an exporter
	// because every one of those is a fact about the whole document, which
	// neither web target can see from the element it is writing.
	core.AuditTree(tree)

	// A tree that will not encode (a NaN in a style or a prop) is a mount
	// with nothing to mount: the payload used to be {"error": ...}, which
	// every host refused, and the screen stayed blank. The placeholder stands
	// in, as for a panic, and is what the next pass diffs against, so the
	// app comes up the moment its state encodes. See unencodable.
	out, err := encodePass(tree)
	if err != nil {
		r.unencodable("initial render", err, tree)
		tree = panicPlaceholder()
		out, _ = encodePass(tree) // a Text with one string prop always encodes
	}
	r.currentTree = tree
	return out
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
	// the context tree, then the accessibility checks over the new tree); a
	// no-op when debug mode is off.
	//
	// Before the diff rather than after, so the audit sees the tree it was
	// asked about even though nothing between them touches it: the two debug
	// passes belong together and both describe the render, not the patch.
	r.context.EndRenderPass()
	core.AuditTree(newTree)
	patches := reconcile.Diff(r.currentTree, newTree, "root")
	if patches == nil {
		// A no-change render must serialize as "[]", not "null": the native
		// runtimes iterate the decoded patch list without a null check.
		patches = []reconcile.Patch{}
	}

	// Encoded BEFORE the pass is committed. currentTree is Go's record of
	// what the host holds; it used to advance first, and a pass that then
	// failed to encode went out as {"error": ...}, which every host refused.
	// Every patch of that pass was lost and never re-sent, because Go
	// believed the host already had them: one NaN in a progress value froze
	// every unrelated change made in the same pass, for good.
	//
	// Now a pass that will not encode is abandoned exactly as a panicking
	// one is (above, and for the same three reasons): currentTree stays at
	// the last tree the host has, the purge is skipped because the host's
	// callback IDs are still the old pass's, and the dirty flag is cleared.
	// The next pass is diffed against the tree the host really holds, so
	// once the value encodes again everything this pass carried arrives with
	// it.
	out, err := encodePass(patches)
	if err != nil {
		r.unencodable("render pass", err, newTree)
		r.context.ClearDirty()
		return "[]"
	}
	r.currentTree = newTree
	r.context.ClearDirty()
	r.context.PurgeUnusedCallbacks()
	return out
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
//
// With a listener attached the patches are handed to it before the lock is
// released and the return value is "[]"; without one they are returned. Every
// Dispatch* below does the same. See deliverLocked for the ordering hole the
// return value left open.
func (m *Manager) DispatchCallback(id string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.guardHandler(id, func() { m.context.TriggerCallback(id) })
	return m.deliverLocked(m.renderAgainLocked())
}

// DispatchTextCallback is DispatchCallback for string-carrying events.
func (m *Manager) DispatchTextCallback(id string, value string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.guardHandler(id, func() { m.context.TriggerTextCallback(id, value) })
	return m.deliverLocked(m.renderAgainLocked())
}

// DispatchTextEdit is DispatchTextCallback for a native text field's
// keystroke, carrying the host's sequence number and adopted rewrite epoch.
// An edit typed against text Go has since rewritten is acknowledged without
// reaching the handler; see core/text_edit.go for the protocol.
func (m *Manager) DispatchTextEdit(id string, value string, seq, epoch int) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.guardHandler(id, func() { m.context.TriggerTextEdit(id, value, seq, epoch) })
	return m.deliverLocked(m.renderAgainLocked())
}

// DispatchBoolCallback is DispatchCallback for bool-carrying events.
func (m *Manager) DispatchBoolCallback(id string, value bool) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.guardHandler(id, func() { m.context.TriggerBoolCallback(id, value) })
	return m.deliverLocked(m.renderAgainLocked())
}

// DispatchIntCallback is DispatchCallback for int-carrying events.
func (m *Manager) DispatchIntCallback(id string, value int) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.guardHandler(id, func() { m.context.TriggerIntCallback(id, value) })
	return m.deliverLocked(m.renderAgainLocked())
}

// Dispatch runs fn under the render mutex — the same serialization the
// callback dispatches above get — then renders and returns the resulting
// patches. It is the event path for host events (mobile.ReportHostEvent):
// traffic that reaches Go from the shell without a callback ID, such as the
// audio player's status ticks. label names the work in the panic log.
//
// Before the initial mount there is no tree to diff against, so fn runs and
// the state it wrote is simply part of the RenderInitial that follows; the
// "[]" tells the host nothing needs applying. That case is rare (a status
// tick cannot precede the Load that caused it, and Load needs a rendered
// button) but a host that sent it would otherwise receive a diff against
// nothing, which no renderer can apply.
func (m *Manager) Dispatch(label string, fn func()) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.guardHandler(label, fn)
	if m.currentTree == nil {
		return "[]"
	}
	return m.deliverLocked(m.renderAgainLocked())
}

// Resync sends the host the whole tree Go believes it holds, as one replace of
// the root, so a host whose tree has drifted from Go's can start again from
// Go's (N-133, the plan's R4).
//
// # Why a host needs it
//
// Patches are positional and incremental: each is correct only against the
// exact tree the previous one produced. A host that misses one, or applies one
// wrongly, stays wrong, and every later patch compounds it. Every host already
// noticed the symptom (a patch whose path resolves to nothing) and logged it;
// the TreeStores said "the next full replace re-synchronizes", but nothing
// ever asked for one. A host now asks, through mobile.Resync or
// GrMobWASM.Resync, when a path dangles, a payload is not an array, or a
// patch throws.
//
// # What it sends
//
// currentTree, which is the tree the last committed pass left the host with
// (an unencodable or panicking pass is not committed; see renderAgainLocked).
// Nothing is re-rendered: the question is what the host should be holding,
// and a render would answer a different one. The callback IDs in that tree
// are the ones still registered, for the same reason. It uses no new wire
// shape: every host already applies a replace of "root", because a
// navigation emits one.
//
// It goes out like a dispatch (deliverLocked): through the listener inside the
// lock when one is attached, so it cannot be overtaken by a pass and cannot
// overtake one, and returned otherwise. Before the mount there is nothing to
// send, and the answer is "[]".
func (m *Manager) Resync() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.currentTree == nil {
		return "[]"
	}
	out, err := encodePass([]reconcile.Patch{{Type: "replace", TargetID: "root", Changes: m.currentTree}})
	if err != nil {
		// currentTree only ever holds a tree that encoded, so this is a
		// should-not-happen; said rather than sent as a payload no host can
		// apply.
		log.Printf("grmob: resync could not encode the current tree: %v", err)
		return "[]"
	}
	return m.deliverLocked(out)
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

// encodePass marshals one pass's payload, the full tree or its patch list.
//
// It returns the error rather than a stand-in payload. Its predecessor,
// renderJSON, answered a failed marshal with {"error":"failed to encode
// JSON"}, which travelled the ordinary delivery path to hosts that wanted an
// array, and the TreeStores' shape guards (mobile/verify's
// payloadguard_test.go) exist because of it. The guards stay: a host should
// refuse a payload it cannot parse whoever sent it.
func encodePass[T any](v T) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// unencodable reports a pass that would not encode. Always logged, because the
// screen has just stopped following the app's state and nothing else will say
// so; under debug mode also a ConcernUnencodablePass, so a test can assert on
// it. Both name the first non-finite number in the tree when there is one,
// which is nearly always the cause, and fall back to the marshal error.
func (r *Manager) unencodable(where string, err error, tree *core.Node) {
	what := err.Error()
	if found, ok := firstNonFinite(tree, "root"); ok {
		what = found
	}
	log.Printf("grmob: dropped the %s because its payload will not encode as JSON (%v): %s. "+
		"The screen keeps the last tree that did, and the next pass is diffed against it.",
		where, err, what)
	if core.IsDebugMode() {
		core.ReportConcern(core.ConcernUnencodablePass, what)
	}
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
