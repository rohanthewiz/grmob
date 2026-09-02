package render_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/render"
)

// Dispatch is the host-event path: run some Go under the render mutex, then
// diff. A state write inside fn must show up in the returned patches.
func TestDispatchRendersTheStateItsFuncWrote(t *testing.T) {
	var count core.State[int]
	app := func(ctx *core.Context) core.View {
		return core.ComponentFunc(func(ctx *core.Context) *core.Node {
			count = core.NewState(ctx, 0)
			return core.Text("count: " + itoa(count.Get())).Render(ctx)
		})
	}
	m := render.New(core.NewContext(), app)

	// Before the mount there is no tree to diff against: fn still runs and
	// the empty patch list says so.
	if out := m.Dispatch("pre-mount", func() {}); out != "[]" {
		t.Fatalf("pre-mount Dispatch = %q, want []", out)
	}
	m.RenderInitial()

	out := m.Dispatch("bump", func() { count.Set(count.Get() + 1) })
	if !strings.Contains(out, "count: 1") {
		t.Errorf("Dispatch did not render the write: %s", out)
	}
	if out := m.Dispatch("noop", func() {}); out != "[]" {
		t.Errorf("no-op Dispatch = %q, want []", out)
	}

	// A panicking fn is guarded like a handler: the process survives and
	// the pass still renders.
	if out := m.Dispatch("boom", func() { panic("boom") }); out != "[]" {
		t.Errorf("panicking Dispatch = %q", out)
	}
}

func itoa(n int) string { return strconv.Itoa(n) }
