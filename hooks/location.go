package hooks

import (
	"sync"

	"github.com/rohanthewiz/grmob/core"
)

// locationRecord is the per-hook-slot memory of UseLocation: whether this slot
// currently holds a subscription and a sensor reference. It lives in the
// context's hook-slot array (via core.NewState) for the reasons headingRecord
// does — identity by slot position, no cross-talk between components at the
// same cursor, and nothing package-level to reset. The mutex covers the one
// write that happens off the render goroutine: the close path clearing started.
type locationRecord struct {
	mu      sync.Mutex
	started bool
}

// UseLocation turns the device's positioning on for as long as this component
// is mounted, and re-renders when the fix meaningfully changes:
//
//	loc := hooks.UseLocation(ctx)
//	switch {
//	case !loc.Received:  return components.Skeleton{}          // acquiring
//	case !loc.Available: return components.EmptyState{Hint: loc.Error}
//	default:             return components.StaticMap{Lat: loc.Lat, Lng: loc.Lng}
//	}
//
// It is UseHeading with a different sensor, and every argument in that hook's
// doc applies here — the refcounted Start/Stop, the subscribe-before-start
// ordering, why the reading is not kept in the slot. What follows is only what
// differs, and the two differences are the two expensive things about location.
//
// # Ask for the permission first
//
// Every platform gates location, and the dialog is the expected path rather
// than an exception. This hook does not ask: it starts the sensor, the host
// asks the OS if it must, and a refusal comes back as Available: false with a
// reason. Which means that on an *undecided* permission, mounting a component
// that calls this hook is what puts the system dialog on screen — a side
// effect of a screen appearing, which is the surest way to spend an app's one
// chance at a yes.
//
// So a screen that can be reached cold draws the permission's state beside the
// fix, and asks from a gesture:
//
//	status := hooks.UsePermissionLive(ctx, permission.Location)
//	loc := UseLocation(ctx)          // unconditionally — see below
//
//	switch status {
//	case permission.Granted:     return readout(loc)
//	case permission.Prompt:      return askButton()      // permission.Request
//	case permission.Denied:      return settingsHint()
//	case permission.Unavailable: return nil
//	default:                     return components.Skeleton{}
//	}
//
// # Both hooks run on every pass, and the branch is about drawing
//
// This example used to read `case permission.Granted: return mapScreen(ctx)`
// with the hook inside mapScreen, which is a hook inside a conditional. Hook
// slots are handed out by call position (core.NewState), so an arm turning on
// or off moves every slot after it and the component starts reading somebody
// else's state. core/debug.go's cursor audit names that failure, and
// examples/tutorial's TestMain turns the audit on — which is how the shape got
// corrected: lesson 4.12 could not be written the way this doc described it.
//
// The consequence is that mounting the screen starts the sensor. What gates
// that is the *route*: this hook's reference is released when the route frame
// is popped, so a screen the user navigated to is a screen the user asked for.
// A feature that must not ask until a tap goes behind a button that navigates.
//
// The hook is deliberately not the thing that enforces a permission check. A
// hook that refused to start until a permission was granted would be a second
// authorization policy living in the wrong package, and it would be wrong for
// the app that genuinely wants the OS to ask at that moment — a "find my
// nearest branch" button, where the dialog *is* the response to the tap.
//
// # A refusal is not the end of it
//
// The grant usually arrives after this hook has already run, because the tap
// that asks for it is on this screen. Both native hosts keep a refused start
// open and re-arm it when the permission answer changes, so no remount is
// needed — see LocationSensor.kt's awaitingPermission. What stays stale for
// the acquisition window is Error, which still holds the refusal until the
// first fix lands; a screen that prints it will print the old reason while the
// GPS is genuinely working on one.
//
// # Leaving the screen has to actually release it
//
// The mount/unmount limit UseHeading describes is the same here and the cost is
// higher: a hook's subscription is released when the context tree is closed
// rather than when the component leaves the view tree, so a GPS left running is
// a battery cost a user can measure and attribute to this app.
//
// The workaround is the same and the reason to take it seriously is not: put a
// location-reading screen on its own navigation route. A route's frame has its
// own cleanup registry (core/cleanup.go), and popping it releases the reference
// for real.
func UseLocation(ctx *core.Context) core.Location {
	slot := core.NewState(ctx, &locationRecord{})
	rec := slot.Get()

	rec.mu.Lock()
	already := rec.started
	rec.started = true
	rec.mu.Unlock()

	if !already {
		// Subscribe before starting, the ordering UseHeading argues for: the
		// event most likely to land in the gap is the one that matters most —
		// a host with no location answers available:false immediately and then
		// never speaks again, and missing it leaves a screen waiting forever on
		// a fix that is not coming.
		cancel := core.OnLocation(func(core.Location) {
			ctx.RequestRender()
		})
		core.StartLocation()
		ctx.OnClose(func() {
			cancel()
			core.StopLocation()
			// Same drain semantics as the other hooks: after a Close the tree
			// stays renderable and a re-mount must be able to subscribe again,
			// so the slot forgets that it did.
			rec.mu.Lock()
			rec.started = false
			rec.mu.Unlock()
		})
	}
	return core.CurrentLocation()
}
