package render_test

import (
	"encoding/json"
	"io"
	"log"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/render"
)

// quietLogs silences the driver's panic reports for the duration of a test.
// The reporting itself is asserted in TestManagerLogsAnUncaughtRenderPanic;
// everywhere else it is noise that makes a passing run look like a failing one.
func quietLogs(t *testing.T) {
	t.Helper()
	prev := log.Writer()
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(prev) })
}

// flakyApp is a root view whose render panics whenever *fail is true. The flag
// is read at render time, so a test can flip an app between healthy and
// panicking between passes.
func flakyApp(fail *bool, label string) func(*core.Context) core.View {
	return func(ctx *core.Context) core.View {
		return core.ComponentFunc(func(c *core.Context) *core.Node {
			if *fail {
				panic("root render exploded")
			}
			return core.Text(label).Render(c)
		})
	}
}

func TestManagerKeepsTheLastGoodTreeWhenAPassPanics(t *testing.T) {
	// The top-level safety net. A panic no ErrorBoundary caught must not
	// unwind into the native bridge, and the screen must keep showing the last
	// complete tree rather than a half-built one.
	quietLogs(t)
	fail := false
	m := render.New(core.NewContext(), flakyApp(&fail, "healthy"))
	defer m.Close()

	if out := m.RenderInitial(); !strings.Contains(out, "healthy") {
		t.Fatalf("initial render = %s", out)
	}

	fail = true
	if out := m.RenderAgain(); out != "[]" {
		t.Errorf("panicking pass emitted patches %s, want []", out)
	}

	// And the manager is still usable: the next healthy pass renders normally
	// against the tree that was kept.
	fail = false
	if out := m.RenderAgain(); out != "[]" {
		t.Errorf("recovered pass = %s, want [] (the tree is unchanged from the last good one)", out)
	}
}

func TestManagerSurvivesAPanicInTheInitialRender(t *testing.T) {
	// The initial pass has no last-good tree to keep, so it stands in a
	// placeholder. That matters beyond looks: a nil currentTree reads as "not
	// mounted", which parks the push pump permanently.
	quietLogs(t)
	fail := true
	m := render.New(core.NewContext(), flakyApp(&fail, "healthy"))
	defer m.Close()

	out := m.RenderInitial()
	var node jsonNode
	if err := json.Unmarshal([]byte(out), &node); err != nil {
		t.Fatalf("initial render did not produce a node: %v (%s)", err, out)
	}
	if node.Type != "Text" {
		t.Errorf("placeholder node type = %q, want Text", node.Type)
	}

	// A later healthy pass must be able to replace the placeholder.
	fail = false
	if out := m.RenderAgain(); !strings.Contains(out, "healthy") {
		t.Errorf("recovery pass = %s, want patches introducing the real tree", out)
	}
}

func TestManagerSurvivesAPanicInAnEventHandler(t *testing.T) {
	// Handlers run between passes, so no ErrorBoundary in the tree can see
	// them; without the dispatch guard a nil dereference in an onClick unwinds
	// straight out through the native bridge and kills the process.
	quietLogs(t)
	clicks := 0
	ctx := core.NewContext()
	m := render.New(ctx, func(c *core.Context) core.View {
		return core.ComponentFunc(func(cc *core.Context) *core.Node {
			return core.Column(
				core.Button("boom", func() { panic("handler exploded") }),
				core.Button("fine", func() { clicks++ }),
			).Render(cc)
		})
	})
	defer m.Close()
	m.RenderInitial()

	if out := m.DispatchCallback("cb_0"); out != "[]" {
		t.Errorf("dispatch after a panicking handler = %s, want []", out)
	}
	// The app is still live: the sibling handler still dispatches.
	m.DispatchCallback("cb_1")
	if clicks != 1 {
		t.Errorf("sibling handler ran %d times, want 1 — the panic took the app with it", clicks)
	}
}

func TestManagerKeepsHandlersAliveAcrossAPanickingPass(t *testing.T) {
	// The subtle half of abandoning a pass: a partial pass marked only some
	// callback IDs live, so running the usual post-diff purge against that
	// partial set would delete the handlers of the tree still on screen and
	// leave every button on it dead.
	quietLogs(t)
	fail := false
	clicks := 0
	m := render.New(core.NewContext(), func(c *core.Context) core.View {
		return core.ComponentFunc(func(cc *core.Context) *core.Node {
			if fail {
				panic("root render exploded")
			}
			return core.Button("tap", func() { clicks++ }).Render(cc)
		})
	})
	defer m.Close()
	m.RenderInitial()

	fail = true
	m.RenderAgain()

	// The button is still on screen, so its handler must still fire. (The
	// dispatch triggers a render that panics again; only the click matters.)
	m.DispatchCallback("cb_0")
	if clicks != 1 {
		t.Fatalf("handler ran %d times after a panicking pass, want 1 — the purge ate a live handler", clicks)
	}
}

func TestManagerLogsAnUncaughtRenderPanic(t *testing.T) {
	// core.ErrorBoundary deliberately stays silent and lets the app's fallback
	// decide; a panic that reaches the driver has no such owner, and recovering
	// it without a word would turn a crash into a screen that quietly stops
	// updating.
	var buf strings.Builder
	prev := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(prev)

	fail := true
	m := render.New(core.NewContext(), flakyApp(&fail, "healthy"))
	defer m.Close()
	m.RenderInitial()

	got := buf.String()
	if !strings.Contains(got, "root render exploded") {
		t.Errorf("log does not name the panic value:\n%s", got)
	}
	if !strings.Contains(got, "flakyApp") {
		t.Errorf("log does not carry a stack naming the panicking component:\n%s", got)
	}
}

func TestErrorBoundaryKeepsTheRestOfTheAppRendering(t *testing.T) {
	// End to end, through the real driver: one panicking panel is replaced by
	// its fallback while its siblings keep updating normally.
	quietLogs(t)
	count := 0
	m := render.New(core.NewContext(), func(c *core.Context) core.View {
		return core.ComponentFunc(func(cc *core.Context) *core.Node {
			n := core.NewState(cc, 0)
			count = n.Get()
			return core.Column(
				core.ErrorBoundary(
					core.ComponentFunc(func(*core.Context) *core.Node { panic("panel failed") }),
					func(error) core.View { return core.Text("panel unavailable") },
				),
				core.Text("live: "+string(rune('a'+n.Get()))),
				core.Button("bump", func() { n.Set(n.Get() + 1) }),
			).Render(cc)
		})
	})
	defer m.Close()

	out := m.RenderInitial()
	if !strings.Contains(out, "panel unavailable") {
		t.Fatalf("fallback missing from the initial tree: %s", out)
	}
	if !strings.Contains(out, "live: a") {
		t.Fatalf("sibling missing from the initial tree: %s", out)
	}

	patches := m.DispatchCallback("cb_0")
	if !strings.Contains(patches, "live: b") {
		t.Errorf("sibling did not update alongside a failing panel: %s (state = %d)", patches, count)
	}
}
