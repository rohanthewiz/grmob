package hooks

import (
	"sync"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/permission"
)

// permissionRecord is the per-hook-slot memory of UsePermission: whether this
// slot already holds a subscription and has issued its opening check. It lives
// in the context's hook-slot array (via core.NewState) for the same reasons
// headingRecord does — identity by slot position, no cross-talk between
// components at the same cursor, nothing package-level to reset. The mutex
// covers the one write that happens off the render goroutine, the close path
// clearing started.
type permissionRecord struct {
	mu      sync.Mutex
	started bool
}

// UsePermission reports what the platform currently says about p, checking it
// once on mount and re-rendering whenever the answer changes:
//
//	switch hooks.UsePermission(ctx, permission.Location) {
//	case permission.Granted:
//	    return mapView(ctx)
//	case permission.Prompt:
//	    return components.Button{Label: "Use my location",
//	        OnClick: func() { permission.Request(permission.Location) }}
//	case permission.Denied:
//	    return components.EmptyState{Hint: "Location is off — turn it on in Settings"}
//	default: // Unknown while the check is in flight, Unavailable on a device
//	         // that cannot do it at all
//	    return components.Skeleton{}
//	}
//
// # It checks and does not ask
//
// The opening call is permission.Check, never permission.Request, and the
// difference is the whole reason the two are separate functions. A Check
// prompts nothing, so it is safe in a render pass; a Request run from one
// would put the OS dialog on screen as a side effect of drawing, at a moment
// the user did nothing to cause. That is the surest way to a permanent
// refusal, and a permanent refusal is much harder to undo than a permission
// never asked for. Requesting stays the caller's, from a gesture.
//
// # Unknown is a state the caller has to draw
//
// The check is asynchronous, so the first pass returns permission.Unknown and
// a later one returns the platform's answer. That is not a wart to hide: on a
// slow shell the gap is visible, and a screen that treats Unknown as Denied
// flashes an error state before the permission it already has arrives. Draw a
// placeholder for it, as the example does.
//
// A headless run — a Go test, a static export, any host that registered no
// system-event sink — resolves to Unavailable immediately rather than sitting
// on Unknown, so an app under test takes the "cannot do this" branch instead
// of the spinner. See permission.send for why that is the honest answer.
//
// # Coming back from Settings
//
// A user can grant or revoke a permission in the system settings and return to
// the app, and no platform tells the app that happened. This hook does not
// re-check on foreground, because a hook cannot see whether its screen is
// still the one on top and a stack of five screens would each fire a check on
// every app resume. A screen that cares pairs this with hooks.UseLifecycle and
// calls permission.Check itself when the state turns "active".
//
// # The mount/unmount limit
//
// The subscription is taken on the hook's first render and released when the
// context tree is closed (ctx.Close, normally via render.Manager.Close), not
// when the component leaves the view tree — hooks have no unmount signal
// today, the same limit UseAudio, UseInterval and UseHeading carry. The cost
// here is the mild one: a subscription that outlives its screen asks for a
// render that diffs to nothing. Nothing is left running, because a permission
// query is a question and not a sensor.
func UsePermission(ctx *core.Context, p permission.Permission) permission.Status {
	slot := core.NewState(ctx, &permissionRecord{})
	rec := slot.Get()

	rec.mu.Lock()
	already := rec.started
	rec.started = true
	rec.mu.Unlock()

	if !already {
		// Subscribe before checking, for the reason UseHeading subscribes
		// before starting: the answer to this very check can land
		// synchronously — a headless run answers inside the Check call, and a
		// shell with the status already cached can too — and a subscription
		// taken afterwards would miss the one event the screen is waiting on.
		cancel := permission.On(func(changed permission.Permission, _ permission.Status) {
			if changed != p {
				// One process-wide channel carries every permission, so a
				// camera answer reaches a hook watching location. Filtering
				// here rather than re-rendering on any of them is what keeps
				// a screen that asks for two permissions from rendering twice
				// per answer.
				return
			}
			// RequestRender rather than MarkDirty: the answer arrives from an
			// async source with no native event in flight, exactly like a
			// sensor reading or an audio status change, so it needs the push
			// channel to reach the screen.
			ctx.RequestRender()
		})
		permission.Check(p)
		ctx.OnClose(func() {
			cancel()
			// Same drain semantics as UseInterval and UseHeading: after a
			// Close the tree stays renderable and a re-mount must be able to
			// subscribe again, so the slot forgets that it did.
			rec.mu.Lock()
			rec.started = false
			rec.mu.Unlock()
		})
	}
	return permission.Current(p)
}
