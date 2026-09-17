package hooks

import (
	"sync"

	"github.com/rohanthewiz/grmob/core"
)

// windowRecord is the per-hook-slot memory of UseWindow: whether this slot's
// subscription is live. Same shape and reasons as lifecycleRecord.
type windowRecord struct {
	mu         sync.Mutex
	subscribed bool
}

// UseWindow returns the app window's size and fold, and re-renders the app
// whenever either changes — a foldable unfolding, the hinge bending into
// tabletop, an iPad window resized in Split View, a browser window dragged
// wider:
//
//	win := hooks.UseWindow(ctx)
//	if win.WidthClass() == core.SizeCompact {
//	    return listScreen
//	}
//	return comps.TwoPane{First: list, Second: detail}
//
// Branch on WidthClass or Posture rather than on raw Width wherever a bucket
// will do: the report changes on every pixel of a window drag, but the tree
// only needs to change when a bucket does, and a component that renders from
// the bucket diffs to nothing in between.
//
// The subscription is taken on the hook's first render and released when the
// context tree closes, with the same no-unmount-signal limit and the same
// harmless cost as UseLifecycle: a stale subscription asks for one render that
// diffs to nothing.
//
// Like UseLifecycle the value is not copied into the slot; core keeps one
// record for the process (core.CurrentWindow) and reading it at render time is
// what makes every subscriber see the same report.
func UseWindow(ctx *core.Context) core.Window {
	slot := core.NewState(ctx, &windowRecord{})
	rec := slot.Get()

	rec.mu.Lock()
	already := rec.subscribed
	rec.subscribed = true
	rec.mu.Unlock()

	if !already {
		cancel := core.OnWindow(func(core.Window) {
			// RequestRender rather than MarkDirty: the report arrives from
			// the host with no native event in flight.
			ctx.RequestRender()
		})
		ctx.OnClose(func() {
			cancel()
			// A re-mount over a closed tree must be able to subscribe again.
			rec.mu.Lock()
			rec.subscribed = false
			rec.mu.Unlock()
		})
	}
	return core.CurrentWindow()
}
