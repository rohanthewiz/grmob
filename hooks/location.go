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
//
// It carries the subscription itself because UseLocationWhen can let go of the
// sensor without the component going away, which UseLocation never could: when
// the flag goes false there is no close path to cancel on, so the cancel has to
// be somewhere the next render can find it.
type locationRecord struct {
	mu      sync.Mutex
	started bool
	cancel  func()
	// closing says the release path is registered with the context. Once, for
	// the life of the slot, however many times the sensor is started and
	// stopped in between — ctx.OnClose has no unregister, so re-registering per
	// start would leave a handler per flip.
	closing bool
}

// acquire subscribes and starts the sensor, unless this slot already has. The
// subscribe-before-start ordering is UseHeading's argument: the event most
// likely to land in the gap is the one that matters most — a host with no
// location answers available:false immediately and then never speaks again,
// and missing it leaves a screen waiting forever on a fix that is not coming.
func (r *locationRecord) acquire(ctx *core.Context) {
	r.mu.Lock()
	if r.started {
		r.mu.Unlock()
		return
	}
	r.started = true
	r.mu.Unlock()

	cancel := core.OnLocation(func(core.Location) { ctx.RequestRender() })
	r.mu.Lock()
	r.cancel = cancel
	r.mu.Unlock()
	core.StartLocation()
}

// release drops this slot's subscription and its sensor reference, if it has
// them. Idempotent, because it is reached from two places that do not know
// about each other: the flag going false, and the context closing.
//
// The slot forgets that it started, which is the same drain semantics the other
// hooks have: after a Close the tree stays renderable and a re-mount must be
// able to subscribe again.
func (r *locationRecord) release() {
	r.mu.Lock()
	if !r.started {
		r.mu.Unlock()
		return
	}
	r.started = false
	cancel := r.cancel
	r.cancel = nil
	r.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	core.StopLocation()
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
// A screen that cannot be its own route — a settings pane with a map preview
// beside the permission toggle — wants UseLocationWhen instead: the same one
// slot on every pass, with the sensor following a flag the caller owns.
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
// needed — see LocationSensor.kt's awaitingPermission. The re-armed start says
// `acquiring: true` before any fix exists (core.LocationAcquiring), so the
// record goes back to Active-with-nothing-received rather than carrying the
// old reason through the whole acquisition window: the example above draws its
// Skeleton arm, which is what it always meant to.
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
	return UseLocationWhen(ctx, true)
}

// UseLocationWhen is UseLocation with a switch: the sensor runs while `want` is
// true and is released when it goes false, and the hook occupies ONE slot
// either way.
//
//	status := hooks.UsePermissionLive(ctx, permission.Location)
//	loc := hooks.UseLocationWhen(ctx, status == permission.Granted)
//
// # What it is for, which is not saving a Start call
//
// UseLocation's own doc has a paragraph about mounting a screen being what puts
// the system dialog on screen, and an instruction: put such a screen behind a
// navigation route, because the route frame is what releases the reference.
// That is honest and it costs a screen — a settings pane that shows a map
// *preview* beside a permission toggle cannot be its own route, and today it
// either prompts on mount or does not use the hook.
//
// The alternative a caller reaches for is the one thing the hook rules forbid:
//
//	if granted {                       // WRONG
//	    loc = hooks.UseLocation(ctx)
//	}
//
// Hook slots are handed out by call position (core.NewState), so an arm turning
// on or off moves every slot after it and the component starts reading somebody
// else's state — core/debug.go's cursor audit reports exactly this. The flag is
// the same intent with the call site held still: one NewState, every pass,
// whatever the answer.
//
// # It is still not an authorization policy
//
// The flag is the caller's, and it does not have to be a permission — "the map
// tab is the visible one", "the user asked to be followed", "the app is in the
// foreground" are all it. This hook has no opinion about what makes `want`
// true, which is the same refusal UseLocation makes for the same reason: a hook
// that read a permission itself would be a second authorization policy living
// in the wrong package, and wrong for the app that wants the OS to ask at that
// moment.
//
// # Flipping it is cheap, and stopping is real
//
// A false flag releases this slot's reference through the same path a Close
// takes, so the GPS goes off when the last holder lets go — which is the
// release UseLocation's own doc says a mounted component cannot do without a
// route change. A true flag re-subscribes and starts again; core.StartLocation
// is reference-counted, so a sibling still holding it keeps the sensor up and
// this slot merely rejoins.
//
// What does not come back is anything the sensor said while the flag was false:
// core keeps the last fix, so the record a re-enabled hook reads is the last
// position the device reported, which may be old. Location.Received and
// Location.Active are what say whether it is being refreshed.
func UseLocationWhen(ctx *core.Context, want bool) core.Location {
	slot := core.NewState(ctx, &locationRecord{})
	rec := slot.Get()

	// Registered on the first pass rather than on the first start: the flag can
	// flip any number of times and ctx.OnClose has no unregister, so a
	// handler per start would be a handler per flip. Release is idempotent, so
	// a close after the flag has already gone false costs nothing.
	rec.mu.Lock()
	register := !rec.closing
	rec.closing = true
	rec.mu.Unlock()
	if register {
		ctx.OnClose(func() {
			rec.release()
			rec.mu.Lock()
			rec.closing = false
			rec.mu.Unlock()
		})
	}

	if want {
		rec.acquire(ctx)
	} else {
		rec.release()
	}
	return core.CurrentLocation()
}
