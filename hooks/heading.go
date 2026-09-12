package hooks

import (
	"sync"

	"github.com/rohanthewiz/grmob/core"
)

// headingRecord is the per-hook-slot memory of UseHeading: whether this slot
// currently holds a subscription and a sensor reference. It lives in the
// context's hook-slot array (via core.NewState) for the same reasons
// audioRecord does — identity by slot position, no cross-talk between
// components at the same cursor, and nothing package-level to reset. The mutex
// covers the one write that happens off the render goroutine: the close path
// clearing started.
type headingRecord struct {
	mu      sync.Mutex
	started bool
}

// UseHeading turns the compass on for as long as this component is mounted,
// and re-renders on every meaningful change to the reading:
//
//	h := hooks.UseHeading(ctx)
//	if !h.Available && h.Received {
//	    return comps.EmptyState{Hint: "This device has no compass"}
//	}
//	return comps.Compass{Heading: h.Magnetic, ShowDegrees: true}
//
// # What it owns
//
// Two things, taken together on the first render and released together on
// close: a core.OnHeading subscription, and one reference on the sensor
// (core.StartHeading). The reference is the half that matters — the sensor
// costs battery while it runs, so a screen that shows a compass must be the
// thing that turns it off, and "while this component is mounted" is the only
// scope that reliably describes when nobody is looking at it any more.
//
// Because core refcounts Start/Stop, two components may each call UseHeading
// and each release independently; the magnetometer stops when the second one
// lets go, not the first.
//
// # The mount/unmount limit, and why it is worse here
//
// The subscription is taken on the hook's first render and released when the
// context tree is closed (ctx.Close, normally via render.Manager.Close), not
// when the component leaves the view tree — hooks have no unmount signal
// today, the same limit UseAudio and UseInterval carry.
//
// For those two the cost is a redundant render that diffs to nothing. Here it
// is a sensor left running, which is a battery cost a user can measure, so it
// is worth saying plainly what the workaround is: put a compass on its own
// navigation route. A route's frame has its own cleanup registry (see
// core/cleanup.go) and popping it closes that registry, which releases the
// reference for real.
//
// # One redundant render at mount
//
// The subscription is taken *before* core.StartHeading, so the hook hears the
// sensor coming on and asks for a render it does not strictly need: the pass
// that mounted it already read the post-Start value. That extra pass diffs to
// nothing, and the ordering it buys is worth more than it costs.
//
// The alternative — start, then subscribe — drops any reading that lands in
// the gap, and the reading most likely to land there is the one that matters
// most: a host with no compass answers `available: false` immediately and
// then never speaks again. Missing that single event leaves a screen waiting
// on a reading that is never coming, with nothing in any log. Missing a
// heading tick costs 66 milliseconds of staleness that the next tick repairs.
//
// # Why the reading is not stored in the slot
//
// core keeps one record for the process (core.CurrentHeading) and reading it
// at render time is what makes every subscriber see the same tick. The slot
// only remembers that this component already started the sensor, so re-renders
// do not stack references — which, with refcounting, would be a leak that
// never reaches zero.
func UseHeading(ctx *core.Context) core.Heading {
	slot := core.NewState(ctx, &headingRecord{})
	rec := slot.Get()

	rec.mu.Lock()
	already := rec.started
	rec.started = true
	rec.mu.Unlock()

	if !already {
		cancel := core.OnHeading(func(core.Heading) {
			// RequestRender rather than MarkDirty: a sensor reading is an
			// async source with no native event in flight, exactly like a
			// timer tick or an audio status change, so it needs the push
			// channel to reach the screen.
			ctx.RequestRender()
		})
		core.StartHeading()
		ctx.OnClose(func() {
			cancel()
			core.StopHeading()
			// Same drain semantics as UseInterval and UseAudio: after a Close
			// the tree stays renderable and a re-mount must be able to
			// subscribe again, so the slot forgets that it did.
			rec.mu.Lock()
			rec.started = false
			rec.mu.Unlock()
		})
	}
	return core.CurrentHeading()
}
