package hooks

import (
	"sync"

	"github.com/rohanthewiz/grmob/core"
)

// reducerRecord is the per-hook-slot state of one UseReducer. The state lives
// behind the record's own mutex rather than directly in the hook slot,
// because dispatch is a read-modify-write and core.State only offers atomic
// Get and atomic Set:
//
//	slot-backed State                 reducerRecord
//	 dispatch: s.Set(r(s.Get(), a))    dispatch: [mu] state = r(state, a)
//	           ^^^^^^^^^^^^^^^^        two concurrent dispatches serialize,
//	 two concurrent dispatches can     so every action is applied exactly
//	 both read the old state and       once, in some order
//	 one update is lost
//
// Sequencing is the entire reason to reach for a reducer over NewState, so
// losing an action under concurrency would defeat the hook. The record is
// reached by pointer from the slot, which means the slot value itself never
// changes and dispatch must ask for the re-render itself (see below).
type reducerRecord[S any] struct {
	mu    sync.Mutex
	state S
}

// UseReducer holds state that evolves through named actions instead of raw
// writes. It returns the current state for this render and a dispatch
// function; dispatch applies reducer to the live state and requests a render,
// exactly as State.Set does.
//
//	type action int
//	const (increment action = iota; reset)
//
//	count, dispatch := hooks.UseReducer(ctx, func(s int, a action) int {
//	    switch a {
//	    case increment:
//	        return s + 1
//	    case reset:
//	        return 0
//	    }
//	    return s
//	}, 0)
//
//	core.Button("+1", func() { dispatch(increment) })
//
// Rules the implementation depends on:
//
//   - reducer must be pure and must return a *new* state value rather than
//     mutating the one it is given — earlier renders still hold the old value,
//     and the reconciler diffs against the tree they produced.
//   - reducer must not dispatch. It runs while the record's mutex is held, so
//     a re-entrant dispatch deadlocks. Chain actions from an event handler or
//     a hooks.UseEffect instead.
//   - initial is evaluated on every render but only the first render's value
//     is kept, the same as core.NewState.
//
// dispatch is safe to call from any goroutine — timers, network handlers, a
// native event thread — and is stable enough to hand to child views for the
// life of the app: every render returns a fresh closure, but all of them
// write through the same record.
func UseReducer[S any, A any](ctx *core.Context, reducer func(S, A) S, initial S) (S, func(A)) {
	// See UseMemo for why the record goes through NewState: slot identity is
	// what makes the hook per-call-site, and NewState is what keeps the slot
	// array and the cursor in step.
	slot := core.NewState(ctx, &reducerRecord[S]{state: initial})
	rec := slot.Get()

	rec.mu.Lock()
	current := rec.state
	rec.mu.Unlock()

	dispatch := func(action A) {
		rec.mu.Lock()
		rec.state = reducer(rec.state, action)
		rec.mu.Unlock()

		// RequestRender rather than a bare MarkDirty, and explicitly rather
		// than implicitly: because the state hangs off a pointer the slot
		// value never changes, so State.Set — the usual carrier of the render
		// request — is never called on this slot. Without this line a
		// dispatch would update the state invisibly until some unrelated
		// event happened to trigger the next pass.
		ctx.RequestRender()
	}

	return current, dispatch
}
