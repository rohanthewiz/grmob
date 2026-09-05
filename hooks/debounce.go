package hooks

import (
	"sync"
	"time"

	"github.com/rohanthewiz/grmob/core"
)

// Debouncer collapses a burst of calls into one: each Call cancels the
// previous pending call and re-arms the delay, so only the last one in a run
// actually fires. It is what a search box, an autosave, or a
// recompute-on-resize wants — an expensive reaction driven by an input that
// changes far faster than the reaction is worth running.
//
// # Why this is not UseTimeout
//
// UseTimeout arms exactly once per mount and then stays fired: a render after
// it has fired does nothing, deliberately, because the store-keyed version it
// replaced used to re-arm itself on whichever unrelated render came next.
// That is the right contract for "do X once, shortly after this appears", and
// it is the wrong one here — a debounce is defined by re-arming. So the two
// hooks are separate rather than one hook with a flag; they differ in the one
// thing the type is about.
//
//	                Call    Call  Call            (a burst of keystrokes)
//	                  │       │     │
//	timer  ───────────x───────x─────┬──── delay ──▶ fn()
//	               (stopped)      (armed)
//
// # Threading
//
// Call may arrive from any goroutine — it normally arrives from a native
// event callback — and the timer fires on its own. mu guards the whole
// record; fn itself runs outside the lock so a slow callback cannot block the
// next Call, and so an fn that re-enters Call (a debounced action that
// schedules another) does not deadlock.
type Debouncer struct {
	mu    sync.Mutex
	timer *time.Timer
	delay time.Duration
	// ctx is refreshed every render so the fire path nudges the app through
	// whichever context is current; RequestRender routes to the hook owner,
	// so a themed copy is as good as the original.
	ctx *core.Context
	// bound records that OnClose has been registered for this record. Same
	// role as intervalRecord.started: it makes the registration happen on the
	// first render only, and clearing it on close is what lets a
	// close-and-remount over the same context register a fresh one.
	bound bool
}

// UseDebounce returns the Debouncer stored in this component's hook slot,
// refreshing its delay from the current render.
//
// Like every hook it must be called unconditionally, in a stable order, on
// every pass: it consumes a cursor slot through core.NewState (see
// UseInterval for why a hook reserves its slot that way rather than by bumping
// the cursor).
//
//	d := hooks.UseDebounce(ctx, 300*time.Millisecond)
//	...
//	components.SearchField{
//	    Value: query.Get(),
//	    OnChange: func(s string) {
//	        query.Set(s)                       // the field is controlled: now
//	        d.Call(func() { runSearch(s) })    // the query is not: in 300ms
//	    },
//	    OnSubmit: func() { d.Cancel(); runSearch(query.Get()) },
//	}
//
// Unlike UseInterval, whose period is fixed by the first render, the delay is
// re-read every pass: it is held on the record rather than baked into a
// running ticker, so a delay driven by state (a "search as I type" preference)
// takes effect on the next Call.
//
// A pending call is dropped when the context tree is closed, so a Manager
// shutdown cannot land a late callback in a dead app.
func UseDebounce(ctx *core.Context, delay time.Duration) *Debouncer {
	slot := core.NewState(ctx, &Debouncer{})
	d := slot.Get()

	d.mu.Lock()
	d.delay = delay
	d.ctx = ctx
	first := !d.bound
	d.bound = true
	d.mu.Unlock()

	if !first {
		return d
	}
	ctx.OnClose(func() {
		d.Cancel()
		// See UseInterval's OnClose: cleanup is a drain, not a terminal
		// state, so the record has to be re-registerable.
		d.mu.Lock()
		d.bound = false
		d.mu.Unlock()
	})
	return d
}

// Call schedules fn to run once the delay has passed with no further Call,
// replacing whatever was pending.
//
// With a delay of zero or less fn runs synchronously, right here. That is the
// honest reading of "debounce by nothing", and it means a caller can turn the
// debounce off from state without a second code path. No render is requested
// in that case: the call is indistinguishable from the caller having invoked
// fn itself, and it is already inside whatever event handling led here. The
// delayed path does request one, because a timer firing with no native event
// in flight has no other way to reach the screen (the same reason UseTimeout
// and UseInterval do it).
//
// A nil fn is a no-op and does *not* cancel a pending call — cancelling is
// Cancel's job, and reading "schedule nothing" as "unschedule everything"
// would make a guard like `d.Call(handlerFor(mode))` silently destructive
// when the handler happens to be nil.
func (d *Debouncer) Call(fn func()) {
	if fn == nil {
		return
	}

	d.mu.Lock()
	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}
	delay, ctx := d.delay, d.ctx
	if delay <= 0 {
		d.mu.Unlock()
		fn()
		return
	}
	d.timer = time.AfterFunc(delay, func() {
		fn()
		if ctx != nil {
			ctx.RequestRender()
		}
	})
	d.mu.Unlock()
}

// Cancel drops a pending call. Use it when something has superseded the
// debounced work outright — a submit that runs the search now, a screen
// leaving, a field cleared.
//
// It makes no promise about a call already in flight: time.Timer.Stop reports
// false once the timer has fired, and by then fn may already be running on the
// timer goroutine. Cancel is therefore "no *further* call", not "undo". Work
// that must not run twice needs its own guard, as it would with any timer.
func (d *Debouncer) Cancel() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}
}

// Pending reports whether a call is scheduled and has not fired yet. It is
// for the UI that wants to say so — a "searching…" hint that appears the
// moment typing stops rather than when the request goes out.
//
// It is a snapshot, not a lock: the timer can fire in the instant after it
// returns true.
func (d *Debouncer) Pending() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.timer != nil
}
