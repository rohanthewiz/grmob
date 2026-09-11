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
