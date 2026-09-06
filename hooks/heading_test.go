package hooks_test

import (
	"testing"
	"time"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/hooks"
)

// recordSensorCommands captures the start/stop traffic UseHeading generates,
// and detaches the handler afterwards so one test cannot see the next one's.
func recordSensorCommands(t *testing.T) *[]string {
	t.Helper()
	var cmds []string
	core.SetSystemEventHandler(func(name string, data map[string]any) {
		if name != "sensor" {
			return
		}
		if c, ok := data["command"].(string); ok {
			cmds = append(cmds, c)
		}
	})
	t.Cleanup(func() { core.SetSystemEventHandler(nil) })
	return &cmds
}

// UseHeading must start the sensor once, re-render on readings, and — the
// half that costs battery if it is wrong — release the sensor on close.
func TestUseHeadingStartsOnceRendersOnReadingsAndStopsOnClose(t *testing.T) {
	cmds := recordSensorCommands(t)
	ctx := core.NewContext()
	renders := make(chan struct{}, 16)
	ctx.OnStateChange(func() { renders <- struct{}{} })

	var last core.Heading
	body := func(ctx *core.Context) { last = hooks.UseHeading(ctx) }

	renderPass(ctx, body)
	renderPass(ctx, body) // a second render must not stack a reference
	if got := *cmds; len(got) != 1 || got[0] != "start" {
		t.Fatalf("sensor commands = %v, want exactly one start", got)
	}
	if last.Received {
		t.Errorf("hook reported a reading before any arrived: %+v", last)
	}
	if !last.Active {
		t.Errorf("hook returned Active=false on the pass that started the sensor: %+v", last)
	}

	// Mounting itself requests one render: the hook subscribes before it
	// starts the sensor, so it hears its own Active transition. That ordering
	// is deliberate — see "One redundant render at mount" on UseHeading — and
	// the pass it causes diffs to nothing.
	awaitSignal(t, renders, "render request from the sensor coming on")
	assertQuiet(t, renders, 30*time.Millisecond, "extra render at mount (reference stacked)")

	core.ReceiveHostEvent("heading", map[string]any{"magnetic": 42.0})
	awaitSignal(t, renders, "render request after a reading")
	assertQuiet(t, renders, 30*time.Millisecond, "second render request (subscription stacked)")

	renderPass(ctx, body)
	if last.Magnetic != 42 || !last.Received || !last.Active {
		t.Errorf("hook returned %+v after the reading", last)
	}

	ctx.Close()
	if got := *cmds; len(got) != 2 || got[1] != "stop" {
		t.Fatalf("sensor commands = %v, want a stop on close", got)
	}
	core.ReceiveHostEvent("heading", map[string]any{"magnetic": 200.0})
	assertQuiet(t, renders, 30*time.Millisecond, "render request after Close")

	// Re-mount over the same context (the WASM host's shape) takes the sensor
	// again rather than staying dead.
	renderPass(ctx, body)
	if got := *cmds; len(got) != 3 || got[2] != "start" {
		t.Fatalf("sensor commands = %v, want a restart after re-mount", got)
	}
	ctx.Close()
}

// Two components holding the compass is the case the reference count exists
// for: the first one to let go must not blind the second.
func TestTwoConsumersEachHoldTheSensor(t *testing.T) {
	cmds := recordSensorCommands(t)

	first := core.NewContext()
	second := core.NewContext()
	body := func(ctx *core.Context) { hooks.UseHeading(ctx) }

	renderPass(first, body)
	renderPass(second, body)
	if got := *cmds; len(got) != 1 || got[0] != "start" {
		t.Fatalf("sensor commands = %v, want one start for two consumers", got)
	}

	first.Close()
	if got := *cmds; len(got) != 1 {
		t.Fatalf("sensor commands = %v; the first close stopped the sensor "+
			"while a second consumer was still watching", got)
	}
	if !core.HeadingActive() {
		t.Error("sensor inactive while one consumer remains")
	}

	second.Close()
	if got := *cmds; len(got) != 2 || got[1] != "stop" {
		t.Errorf("sensor commands = %v, want a stop once the last consumer closed", got)
	}
}
