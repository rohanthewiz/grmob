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
//	    return comps.Button{Label: "Use my location",
//	        OnClick: func() { permission.Request(permission.Location) }}
//	case permission.Denied:
//	    return comps.EmptyState{Hint: "Location is off — turn it on in Settings"}
//	default: // Unknown while the check is in flight, Unavailable on a device
//	         // that cannot do it at all
//	    return comps.Skeleton{}
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
// # What the branches may not contain
//
// Hooks. The switch above returns different subtrees per status, and a hook
// called inside one of those arms is a hook inside a conditional: slots are
// handed out by call position (core.NewState), so the arm turning on or off
// moves every slot after it and the component reads state that belongs to
// something else. core/debug.go's cursor audit reports it by name.
//
// `mapView(ctx)` above is therefore only safe if mapView draws and does not
// hook. A branch that needs a sensor calls the sensor's hook *beside* this one,
// unconditionally, and uses the status to decide what to draw with it — see
// hooks.UseLocation, whose own doc used to get this wrong, and lesson 4.12 in
// examples/tutorial, which is the pair written out.
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
// re-check on foreground: it reports what the record says and asks once, on
// mount. UsePermissionLive is this hook plus the re-check, and its doc carries
// the argument for why the re-check is owned by the permission rather than by
// the screen — which is the objection that used to be written here, and the
// reason it no longer stands.
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

// UsePermissionLive is UsePermission that also re-checks p every time the app
// returns to the foreground.
//
//	switch hooks.UsePermissionLive(ctx, permission.Camera) {
//	case permission.Granted:
//	    return scanner(ctx)
//	case permission.Denied:
//	    return comps.EmptyState{
//	        Hint:   "Camera is off",
//	        Action: comps.Button{Label: "Open Settings", OnTap: openSettings},
//	    }
//	...
//	}
//
// That Denied branch is the whole reason this exists. It sends the user to
// the system settings, they flip the switch, and they come back — and with
// UsePermission the screen still says the camera is off, because nothing on
// any of the three platforms announces a permission change and the hook's one
// check already happened on mount. The screen is wrong until something else
// re-renders it, and the button that fixed the problem is the thing still
// telling the user it is broken.
//
// # What it costs, and why it is not per-screen
//
// The re-check is permission.WatchForeground, which reference-counts by
// permission rather than by caller: five screens watching the camera produce
// one check per resume between them, and an app with no live watcher takes no
// lifecycle subscription at all. See that function's file for the argument —
// it is the one that used to keep this hook from existing.
//
// A resume that changed nothing costs one system event out and one host event
// back, and stops there: permission.set notifies only on a change, so no
// subscriber runs and no render is requested. Only the resume that actually
// changed something reaches the screen.
//
// # The same rule about branches applies
//
// `scanner(ctx)` above must not call hooks, for the reason UsePermission's
// "What the branches may not contain" gives: a hook in a status arm shifts
// every slot after it. A sensor's hook goes beside this call, not inside a
// branch of it.
//
// # Use this one by default
//
// UsePermission is the narrower tool and stays for the screens that want it —
// a debug readout, a settings row rendered inside a sheet that cannot be left
// without unmounting it — but a screen that draws a Denied state and offers a
// way out of it wants this. The extra cost over UsePermission is one lifecycle
// subscription per process and one round trip per resume per kind.
//
// The watch is released when the context tree is closed, the same mount/unmount
// limit UsePermission carries and for the same reason: hooks have no unmount
// signal. A watch that outlives its screen costs a check per resume that
// diffs to nothing.
func UsePermissionLive(ctx *core.Context, p permission.Permission) permission.Status {
	// Ordered so the opening Check happens first: UsePermission subscribes and
	// checks on its own first render, and registering the watch before that
	// would be asking for a re-check of something nobody has checked yet.
	status := UsePermission(ctx, p)

	slot := core.NewState(ctx, &foregroundRecord{})
	rec := slot.Get()

	rec.mu.Lock()
	already := rec.watching
	rec.watching = true
	rec.mu.Unlock()

	if !already {
		cancel := permission.WatchForeground(p)
		ctx.OnClose(func() {
			cancel()
			// Same drain semantics as every other hook here: after a Close the
			// tree stays renderable and a re-mount must be able to watch
			// again, so the slot forgets that it did.
			rec.mu.Lock()
			rec.watching = false
			rec.mu.Unlock()
		})
	}
	return status
}

// foregroundRecord is the per-hook-slot memory of UsePermissionLive's watch.
//
// A second record rather than a field on permissionRecord, because the two
// hooks do not share a slot: UsePermissionLive calls UsePermission, which
// takes its own core.NewState at the cursor position before this one. Folding
// the flag into permissionRecord would put UsePermission's slot and this one
// at the same index and make the two hooks read each other's memory.
type foregroundRecord struct {
	mu       sync.Mutex
	watching bool
}
