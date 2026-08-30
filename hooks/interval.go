package hooks

import (
	"sync"
	"time"

	"github.com/rohanthewiz/grmob/core"
)

// intervalRecord is the per-hook-slot state of one UseInterval: the liveness
// of its ticker goroutine and the freshest callback closure. Storing it in
// the context's hook-slot array (via core.NewState) replaces the former
// package-global store keyed by cursor alone, which had two real collisions:
// two components at the same cursor position — even within one app — shared a
// key, so the second interval silently never started; and the global
// ClearIntervals stopped every app's tickers in the process.
//
//	render pass (serialized       ticker goroutine
//	 by render.Manager)                 │
//	   │ started? ──▶ start once        │ tick
//	   │ fn = latest closure ──[mu]──▶  │ read fn, call it
//	   ▼                                ▼ ctx.RequestRender()
//
// mu guards only fn: the render goroutine refreshes it every pass while the
// ticker goroutine reads it on every tick. started needs no lock — it is
// touched exclusively during render passes, which the manager serializes.
type intervalRecord struct {
	mu      sync.Mutex
	fn      func()
	started bool
}

// UseInterval invokes fn every interval for as long as the app lives. The
// ticker starts on the hook's first render; later renders only refresh the
// callback closure, so ticks always run the latest one (with the current
// render's state captures) rather than the closure from the mount render.
// The interval duration itself is fixed by the first render — a changed
// duration on a re-render is ignored, matching the original behavior.
//
// The ticker is owned by the context tree: it stops when the tree is closed
// (ctx.Close, normally reached via render.Manager.Close), not when the
// component leaves the view tree — hooks have no unmount signal today.
func UseInterval(ctx *core.Context, fn func(), interval time.Duration) {
	// The record doubles as this hook's cursor slot. Reserving the slot
	// through NewState (rather than a bare Cursor++) keeps the slot array and
	// the cursor aligned: a bare increment leaves no slot behind, so every
	// NewState after this hook in the same render would append its backing
	// slot one index short of its cursor, and later hooks would read each
	// other's slots (e.g. a bool Checkbox state landing where a string Input
	// state is read, which panics on the type assertion).
	slot := core.NewState(ctx, &intervalRecord{})
	rec := slot.Get()

	rec.mu.Lock()
	rec.fn = fn
	rec.mu.Unlock()

	if rec.started {
		return
	}
	rec.started = true

	ticker := time.NewTicker(interval)
	done := make(chan struct{})
	ctx.OnClose(func() {
		// Stop() ends the ticks but never closes ticker.C, so without done
		// the goroutine below would park on the channel forever.
		ticker.Stop()
		close(done)
	})

	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				rec.mu.Lock()
				f := rec.fn
				rec.mu.Unlock()
				f()
				// RequestRender (not just MarkDirty) so a timer tick reaches
				// the screen through the push channel even when no native
				// event is in flight to piggyback a re-render on.
				ctx.RequestRender()
			}
		}
	}()
}

// timeoutRecord mirrors intervalRecord for the one-shot case; see that type
// for the slot-storage rationale and the locking picture (here the timer
// goroutine reads fn exactly once, when it fires).
type timeoutRecord struct {
	mu        sync.Mutex
	fn        func()
	scheduled bool
}

// UseTimeout invokes fn once, delay after the hook's first render. Renders
// while the timer is pending only refresh the closure (the fire runs the
// latest one); renders after it has fired do nothing. That last part is a
// deliberate change from the store-keyed version, which forgot a fired
// timeout and therefore re-armed it on whichever render happened to come
// next — a repeat schedule driven by unrelated render timing.
//
// A pending timer is cancelled when the context tree is closed, so a Manager
// shutdown cannot leak a late fn call into a dead app.
func UseTimeout(ctx *core.Context, fn func(), delay time.Duration) {
	// Same slot-reservation rationale as UseInterval above.
	slot := core.NewState(ctx, &timeoutRecord{})
	rec := slot.Get()

	rec.mu.Lock()
	rec.fn = fn
	rec.mu.Unlock()

	if rec.scheduled {
		return
	}
	rec.scheduled = true

	timer := time.AfterFunc(delay, func() {
		rec.mu.Lock()
		f := rec.fn
		rec.mu.Unlock()
		f()
		// RequestRender rather than the bare MarkDirty this used to do: a
		// timeout firing with no native event in flight needs the push
		// channel to reach the screen, exactly like an interval tick.
		ctx.RequestRender()
	})
	ctx.OnClose(func() {
		timer.Stop()
	})
}
