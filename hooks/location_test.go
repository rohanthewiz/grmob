package hooks_test

import (
	"testing"
	"time"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/hooks"
)

// recordLocationCommands captures the start/stop traffic UseLocation generates.
// It filters on the sensor *kind*, which recordSensorCommands next door does not
// have to: both sensors send the same event name, so a test that counted every
// "sensor" command would pass while starting the wrong one.
func recordLocationCommands(t *testing.T) *[]string {
	t.Helper()
	var cmds []string
	core.SetSystemEventHandler(func(name string, data map[string]any) {
		if name != "sensor" || data["kind"] != "location" {
			return
		}
		if c, ok := data["command"].(string); ok {
			cmds = append(cmds, c)
		}
	})
	t.Cleanup(func() { core.SetSystemEventHandler(nil) })
	return &cmds
}

// UseLocation must start the sensor once, re-render on fixes, and — the half
// that costs a measurable amount of battery if it is wrong — release it on
// close.
func TestUseLocationStartsOnceRendersOnFixesAndStopsOnClose(t *testing.T) {
	cmds := recordLocationCommands(t)
	ctx := core.NewContext()
	renders := make(chan struct{}, 16)
	ctx.OnStateChange(func() { renders <- struct{}{} })

	var last core.Location
	body := func(ctx *core.Context) { last = hooks.UseLocation(ctx) }

	renderPass(ctx, body)
	renderPass(ctx, body) // a second render must not stack a reference
	if got := *cmds; len(got) != 1 || got[0] != "start" {
		t.Fatalf("sensor commands = %v, want exactly one start", got)
	}
	if last.Received {
		t.Errorf("hook reported a fix before any arrived: %+v", last)
	}
	if !last.Active {
		t.Errorf("hook returned Active=false on the pass that started the sensor: %+v", last)
	}

	// Mounting requests one render, because the hook subscribes before it
	// starts and so hears its own Active transition. See "One redundant render
	// at mount" on UseHeading, whose ordering this shares.
	awaitSignal(t, renders, "render request from the sensor coming on")
	assertQuiet(t, renders, 30*time.Millisecond, "extra render at mount (reference stacked)")

	core.ReceiveHostEvent("location", map[string]any{"lat": 38.7223, "lng": -9.1393, "accuracy": 8.0})
	awaitSignal(t, renders, "render request after a fix")
	assertQuiet(t, renders, 30*time.Millisecond, "second render request (subscription stacked)")

	renderPass(ctx, body)
	if last.Lat != 38.7223 || !last.Received || !last.Active {
		t.Errorf("hook returned %+v after the fix", last)
	}

	ctx.Close()
	if got := *cmds; len(got) != 2 || got[1] != "stop" {
		t.Fatalf("sensor commands = %v, want a stop on close", got)
	}
	core.ReceiveHostEvent("location", map[string]any{"lat": 40.0, "lng": -8.0})
	assertQuiet(t, renders, 30*time.Millisecond, "render request after Close")

	// Re-mount over the same context (the WASM host's shape) takes the sensor
	// again rather than staying dead.
	renderPass(ctx, body)
	if got := *cmds; len(got) != 3 || got[2] != "start" {
		t.Fatalf("sensor commands = %v, want a restart after re-mount", got)
	}
	ctx.Close()
}

// Two components holding the GPS is the case the reference count exists for,
// and the one a plain on/off flag gets wrong silently.
func TestTwoConsumersEachHoldTheLocationSensor(t *testing.T) {
	cmds := recordLocationCommands(t)

	first := core.NewContext()
	second := core.NewContext()
	body := func(ctx *core.Context) { hooks.UseLocation(ctx) }

	renderPass(first, body)
	renderPass(second, body)
	if got := *cmds; len(got) != 1 || got[0] != "start" {
		t.Fatalf("sensor commands = %v, want one start for two consumers", got)
	}

	first.Close()
	if got := *cmds; len(got) != 1 {
		t.Fatalf("sensor commands = %v; the first close stopped the GPS while a "+
			"second consumer was still watching", got)
	}
	if !core.LocationActive() {
		t.Error("sensor inactive while one consumer remains")
	}

	second.Close()
	if got := *cmds; len(got) != 2 || got[1] != "stop" {
		t.Errorf("sensor commands = %v, want a stop once the last consumer closed", got)
	}
}

// The two sensors are independent through the hooks as well as in core: a
// screen with a compass on it must not be holding the GPS, which is the
// expensive half of that mistake.
func TestTheLocationHookDoesNotStartTheCompass(t *testing.T) {
	core.SetSystemEventHandler(func(string, map[string]any) {})
	t.Cleanup(func() { core.SetSystemEventHandler(nil) })

	ctx := core.NewContext()
	renderPass(ctx, func(ctx *core.Context) { hooks.UseLocation(ctx) })
	defer ctx.Close()

	if core.HeadingActive() {
		t.Error("UseLocation started the magnetometer")
	}
	if !core.LocationActive() {
		t.Error("UseLocation did not start the GPS")
	}
}

// UseLocationWhen's whole point: the sensor follows the flag, and the hook
// occupies one slot whatever the flag says.
//
// The slot half is asserted with a second hook AFTER it, which is the shape
// that actually breaks when a hook is put inside an `if`: the conditional hook
// moves every slot below it, so the state that comes back belongs to somebody
// else. A test that only counted starts and stops would pass on the broken
// version.
func TestUseLocationWhenFollowsTheFlagWithoutMovingTheSlotsBelowIt(t *testing.T) {
	cmds := recordLocationCommands(t)
	ctx := core.NewContext()

	want := false
	var loc core.Location
	var below core.State[int]
	body := func(ctx *core.Context) {
		loc = hooks.UseLocationWhen(ctx, want)
		below = core.NewState(ctx, 7)
	}

	renderPass(ctx, body)
	if got := *cmds; len(got) != 0 {
		t.Fatalf("sensor commands = %v, want none while the flag is false", got)
	}
	if loc.Active {
		t.Error("the record says Active with the flag false")
	}
	below.Set(41)

	want = true
	renderPass(ctx, body)
	if got := *cmds; len(got) != 1 || got[0] != "start" {
		t.Fatalf("sensor commands = %v, want a start when the flag goes true", got)
	}
	if got := below.Get(); got != 41 {
		t.Errorf("the slot below the hook holds %d, want 41 — the hook moved the cursor", got)
	}

	// Still one reference, not one per pass.
	renderPass(ctx, body)
	if got := *cmds; len(got) != 1 {
		t.Fatalf("sensor commands = %v, want the start not repeated", got)
	}

	want = false
	renderPass(ctx, body)
	if got := *cmds; len(got) != 2 || got[1] != "stop" {
		t.Fatalf("sensor commands = %v, want a stop when the flag goes false", got)
	}
	if core.LocationActive() {
		t.Error("the sensor is still running with the flag false")
	}
	if got := below.Get(); got != 41 {
		t.Errorf("the slot below the hook holds %d after the flag flipped twice", got)
	}

	// And it comes back.
	want = true
	renderPass(ctx, body)
	if got := *cmds; len(got) != 3 || got[2] != "start" {
		t.Fatalf("sensor commands = %v, want a restart", got)
	}
	ctx.Close()
	if got := *cmds; len(got) != 4 || got[3] != "stop" {
		t.Fatalf("sensor commands = %v, want a stop on close", got)
	}
}

// Closing after the flag has already released the sensor must not stop it a
// second time — which on a refcounted sensor would take it away from a sibling
// that is still holding it, and is the reason release is idempotent.
func TestClosingAfterTheFlagWentFalseDoesNotStopTheSensorTwice(t *testing.T) {
	cmds := recordLocationCommands(t)

	holder := core.NewContext()
	renderPass(holder, func(ctx *core.Context) { hooks.UseLocation(ctx) })

	gated := core.NewContext()
	want := true
	body := func(ctx *core.Context) { hooks.UseLocationWhen(ctx, want) }
	renderPass(gated, body)

	want = false
	renderPass(gated, body)
	gated.Close()

	if got := *cmds; len(got) != 1 || got[0] != "start" {
		t.Fatalf("sensor commands = %v, want one start and no stop — the holder is "+
			"still watching", got)
	}
	if !core.LocationActive() {
		t.Error("the sensor stopped while a second consumer was still holding it")
	}

	holder.Close()
	if got := *cmds; len(got) != 2 || got[1] != "stop" {
		t.Fatalf("sensor commands = %v, want the stop when the last holder lets go", got)
	}
}

// UseLocation is UseLocationWhen with the flag nailed true, so the flag hook
// must re-render on a fix exactly as the plain one does. Asserted rather than
// assumed, because the subscription moved into the record when the two were
// merged and a dropped RequestRender is invisible to every count above.
func TestUseLocationWhenRendersOnAFix(t *testing.T) {
	recordLocationCommands(t)
	ctx := core.NewContext()
	renders := make(chan struct{}, 16)
	ctx.OnStateChange(func() { renders <- struct{}{} })
	t.Cleanup(ctx.Close)

	var last core.Location
	body := func(ctx *core.Context) { last = hooks.UseLocationWhen(ctx, true) }
	renderPass(ctx, body)
	awaitSignal(t, renders, "render request from the sensor coming on")

	core.ReceiveHostEvent("location", map[string]any{"lat": 38.7223, "lng": -9.1393})
	awaitSignal(t, renders, "render request after a fix")
	renderPass(ctx, body)
	if last.Lat != 38.7223 || !last.Received {
		t.Errorf("hook returned %+v after the fix", last)
	}
}
