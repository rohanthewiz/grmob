package hooks

import (
	"sync"
	"time"

	"github.com/rohanthewiz/grmob/core"
)

// nowRecord is the per-slot state of one UseNow: the most recent tick's
// instant and the liveness of the goroutine producing ticks. It lives in the
// hook-slot array for the reasons intervalRecord documents.
//
// mu guards both fields. now is written by the tick goroutine and read by
// the render pass; started is cleared from the close path, which runs on
// whichever goroutine called Close.
type nowRecord struct {
	mu      sync.Mutex
	now     time.Time
	started bool
}

// UseNow returns the current time and re-renders every time it crosses a
// boundary of every — each second for time.Second, the top of each minute for
// time.Minute:
//
//	now := hooks.UseNow(ctx, time.Second)
//	comps.AnalogClock{Time: now, ShowSeconds: true}
//
// # Why the clock widgets do not call this themselves
//
// comps.DigitalClock and comps.AnalogClock take a time.Time and hold no hooks,
// the same split as comps.Compass and UseHeading. A widget that ticked itself
// would be a hook caller (render it unconditionally, or its slots shift), could
// not be tested at a fixed instant, and could not show anything but "now here" —
// a world clock, a stopwatch or a replay all want to hand over a different time.
// The app also gets to decide the render rate: a clock without a second hand
// needs one pass a minute, not sixty.
//
// # Why not UseInterval
//
// UseInterval's ticker starts at mount, so a one-second ticker is some fixed
// fraction of a second out of phase with the wall clock for the life of the
// app. A clock driven that way changes its seconds digit up to 999 ms after
// the phone's status bar does, which is visible when both are on screen.
// UseNow instead sleeps until the next boundary every time:
//
//	wall clock   ──|────────|────────|────────|──   boundaries of every
//	mount           ▲
//	UseInterval     ·────────·────────·────────·    fixed phase from mount
//	UseNow                  ·────────·────────·    re-aimed at each boundary
//
// Recomputing the delay on every tick (rather than one ticker started at the
// first boundary) also absorbs timer lateness and wall-clock adjustments, so
// the phase cannot drift over a long-running session.
//
// Boundaries are those of time.Truncate, i.e. multiples of every since the
// zero time in UTC. Seconds and minutes are therefore aligned in every time
// zone; an hour is aligned only in zones whose offset is a whole number of
// hours.
//
// # Limits shared with UseInterval
//
// every <= 0 means one second. The duration is fixed by the first render. The
// goroutine stops when the context tree is closed, not when the component
// leaves the view (hooks have no unmount signal), and restarts on a re-mount
// over the same context.
func UseNow(ctx *core.Context, every time.Duration) time.Time {
	if every <= 0 {
		every = time.Second
	}

	// Same slot-reservation rationale as useInterval.
	slot := core.NewState(ctx, &nowRecord{})
	rec := slot.Get()

	rec.mu.Lock()
	if rec.started {
		now := rec.now
		rec.mu.Unlock()
		return now
	}
	rec.started = true
	// The mount pass reads the clock directly: there is no tick yet, and a
	// zero time.Time would draw 00:00 for up to one interval.
	rec.now = time.Now()
	now := rec.now
	rec.mu.Unlock()

	done := make(chan struct{})
	ctx.OnClose(func() {
		close(done)
		// Cleared so a re-mount over the same context starts a fresh
		// goroutine; see the matching comment in useInterval.
		rec.mu.Lock()
		rec.started = false
		rec.mu.Unlock()
	})

	go func() {
		timer := time.NewTimer(untilNextBoundary(time.Now(), every))
		defer timer.Stop()
		for {
			select {
			case <-done:
				return
			case <-timer.C:
				t := time.Now()
				rec.mu.Lock()
				rec.now = t
				rec.mu.Unlock()
				ctx.RequestRender()
				// Re-aimed from the time just read, not from the previous
				// target, so a late wake-up does not push every later tick
				// late with it.
				timer.Reset(untilNextBoundary(t, every))
			}
		}
	}()

	return now
}

// untilNextBoundary is how long after t the next multiple of every falls. A t
// exactly on a boundary waits a whole interval: it has just been rendered, and
// a zero delay would tick twice for the same instant.
func untilNextBoundary(t time.Time, every time.Duration) time.Duration {
	return t.Truncate(every).Add(every).Sub(t)
}
