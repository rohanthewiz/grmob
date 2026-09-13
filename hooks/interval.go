package hooks

import (
	"reflect"
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
// mu guards fn and started. fn is refreshed by the render goroutine every
// pass and read by the ticker goroutine on every tick. started is read during
// render passes (which the manager serializes) but *cleared* from the
// cleanup path, which runs on whichever goroutine called Close — so it needs
// the lock too.
type intervalRecord struct {
	mu      sync.Mutex
	fn      func()
	started bool
	// paused is the latest render's answer to "should a tick do anything?".
	// Always false for UseInterval; UseIntervalWhile writes !active every
	// pass. Guarded by mu for the same reason fn is: written by the render
	// goroutine, read by the ticker goroutine.
	paused bool
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
	useInterval(ctx, fn, interval, true)
}

// UseIntervalWhile is UseInterval with an off switch the render controls: a
// tick calls fn and requests a render only while the most recent render passed
// active = true. A paused tick does nothing at all — no fn, no render pass.
//
//	hooks.UseIntervalWhile(ctx, loading, func() { angle.Set(angle.Get() + 30) }, 80*time.Millisecond)
//
// # Why a second hook rather than a no-op fn
//
// UseInterval requests a render after every tick, because it cannot know
// whether fn changed anything. That is right for a clock and wrong for a
// widget that is only sometimes animating: a spinner written on UseInterval
// with an `if !active { return }` inside fn still costs the whole app a render
// pass every tick for as long as the process lives (hooks have no unmount
// signal, so the ticker outlives the spinner's visibility). Here the pause is
// read on the ticker goroutine before either call, so an idle widget costs one
// goroutine wake per interval and nothing else.
//
// # What it does not change
//
// The hook still occupies one slot and must be called unconditionally in a
// stable position; the ticker still starts on the first render (even a paused
// one) and stops only on ctx.Close. The duration is fixed by the first render,
// as with UseInterval. Resuming is not immediate: the first effective tick
// arrives on the ticker's next beat after a render passes active = true.
func UseIntervalWhile(ctx *core.Context, active bool, fn func(), interval time.Duration) {
	useInterval(ctx, fn, interval, active)
}

// useInterval is the shared body. active is stored every pass so the ticker
// goroutine always sees the latest render's decision.
func useInterval(ctx *core.Context, fn func(), interval time.Duration, active bool) {
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
	rec.paused = !active
	alreadyRunning := rec.started
	rec.started = true
	rec.mu.Unlock()

	if alreadyRunning {
		return
	}

	ticker := time.NewTicker(interval)
	done := make(chan struct{})
	ctx.OnClose(func() {
		// Stop() ends the ticks but never closes ticker.C, so without done
		// the goroutine below would park on the channel forever.
		ticker.Stop()
		close(done)
		// Clearing started is what makes the hook survive a close-and-remount
		// over the same context — the shape both hosts use (wasm/main.go's
		// renderInitial and mobile.Register both call Manager.Close() and
		// then render again on the same ctx). The record lives in a hook slot
		// on that context, so it outlives the Close; leaving started set made
		// the re-mounted hook take the "already running" early return above
		// and never start a replacement ticker, and the app's timers were
		// silently dead for the rest of the process.
		//
		// This is the same drain (not terminal) semantics cleanupRegistry
		// documents: after a Close the tree stays renderable and its hooks
		// must be able to register fresh resources.
		rec.mu.Lock()
		rec.started = false
		rec.mu.Unlock()
	})

	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				rec.mu.Lock()
				f, paused := rec.fn, rec.paused
				rec.mu.Unlock()
				// A paused tick is dropped whole: skipping only f would still
				// request the render pass this hook exists to avoid.
				if paused {
					continue
				}
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
	alreadyScheduled := rec.scheduled
	rec.scheduled = true
	rec.mu.Unlock()

	if alreadyScheduled {
		return
	}

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
		// See UseInterval's OnClose: clearing the flag is what lets a
		// re-mount over the same context arm a fresh timer instead of taking
		// the "already scheduled" early return forever. A timeout that
		// already fired is re-armed by the re-mount, which is the correct
		// reading of a mount — the new tree has not seen it fire.
		rec.mu.Lock()
		rec.scheduled = false
		rec.mu.Unlock()
	})
}

// timeoutWhileRecord is the per-slot state of one UseTimeoutWhile. It differs
// from timeoutRecord in holding the pending timer, because this hook cancels
// and re-arms where UseTimeout only ever fires once.
//
// gen is what makes a cancel safe against a timer that has already fired and
// is waiting on mu: every disarm bumps it, and a fire that finds its captured
// generation stale returns without calling fn. time.Timer.Stop alone cannot
// promise that, since Stop reports false once the timer's goroutine has
// started.
//
// mu guards every field. The render goroutine arms and disarms; the timer
// goroutine reads fn and gen; the close path disarms from whichever goroutine
// called Close.
type timeoutWhileRecord struct {
	mu    sync.Mutex
	fn    func()
	timer *time.Timer
	gen   uint64
	// armed is true from the render that armed the timer until a render passes
	// active = false. It stays true after the timer fires, which is what makes
	// the timeout fire once per activation rather than once per render.
	armed bool
	deps  []any
	// closeHooked records that an OnClose cleanup is registered for this
	// slot, so a render does not add one per pass.
	closeHooked bool
}

// disarm stops any pending fire and ends the current activation. The caller
// holds mu.
func (r *timeoutWhileRecord) disarm() {
	if r.timer != nil {
		r.timer.Stop()
		r.timer = nil
	}
	r.gen++
	r.armed = false
}

// UseTimeoutWhile calls fn once, delay after a render first passes
// active = true. A render that passes active = false cancels a pending call,
// and the next render that passes true arms a fresh one. While active, a
// change in deps (compared with reflect.DeepEqual, as UseEffect does) restarts
// the delay.
//
//	hooks.UseTimeoutWhile(ctx, visible, onTimeout, 4*time.Second, message)
//
//	render:  active=false   true ──── true ──── true(deps changed) ── false
//	timer:        ·         arm ───────────────  re-arm ───────────── cancel
//	fn:           ·                  (fires once)          (fires once)
//
// # Why UseTimeout could not do this
//
// UseTimeout arms on the first render and never again, which suits a splash
// screen and not a widget that is shown, hidden and shown again: a snackbar
// built on it would time out once for the life of the app. This hook is to
// UseTimeout what UseIntervalWhile is to UseInterval, with deps added because
// a snackbar whose message is replaced while it is up should get its full
// delay for the new message.
//
// # What it shares with the other timer hooks
//
// It takes one slot and must be called unconditionally in a stable position.
// fn is refreshed every render and the fire runs the latest closure, then
// requests a render so the change reaches the screen with no native event in
// flight. A pending timer is cancelled when the context tree is closed.
func UseTimeoutWhile(ctx *core.Context, active bool, fn func(), delay time.Duration, deps ...any) {
	slot := core.NewState(ctx, &timeoutWhileRecord{})
	rec := slot.Get()

	rec.mu.Lock()
	rec.fn = fn
	switch {
	case !active:
		if rec.armed {
			rec.disarm()
		}
	case rec.armed && reflect.DeepEqual(rec.deps, deps):
		// Same activation, same deps: pending or already fired, nothing to do.
	default:
		rec.disarm()
		rec.armed = true
		// Copied for the reason UseEffect copies: a caller spreading a slice
		// it later mutates would otherwise change the stored deps in step.
		rec.deps = append([]any(nil), deps...)
		gen := rec.gen
		rec.timer = time.AfterFunc(delay, func() {
			rec.mu.Lock()
			if rec.gen != gen {
				rec.mu.Unlock()
				return
			}
			rec.timer = nil
			f := rec.fn
			rec.mu.Unlock()
			f()
			ctx.RequestRender()
		})
	}
	needHook := !rec.closeHooked
	rec.closeHooked = true
	rec.mu.Unlock()

	// Registered outside mu: Close runs cleanups and each one takes mu, so
	// holding mu while reaching into the context's cleanup registry would
	// order the two locks one way here and the other way there.
	if needHook {
		ctx.OnClose(func() {
			rec.mu.Lock()
			rec.disarm()
			// Cleared so a re-mount over the same context registers again,
			// the drain-not-terminal rule UseInterval's OnClose describes.
			rec.closeHooked = false
			rec.mu.Unlock()
		})
	}
}
