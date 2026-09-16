package htmlout

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

func TestPathDataSpellsEachOpcode(t *testing.T) {
	got := PathData([]float64{0, 0, 1.5, 1, 10, 20, 2, 1, 2, 3, 4, 5, 6, 3})
	if want := "M0 1.5L10 20C1 2 3 4 5 6Z"; got != want {
		t.Errorf("PathData = %q, want %q", got, want)
	}
	// A truncated operation ends the path rather than failing it.
	if got := PathData([]float64{0, 1, 2, 1, 3}); got != "M1 2" {
		t.Errorf("truncated PathData = %q", got)
	}
	if got := PathData([]float64{0, 1, 2, 9, 1, 1}); got != "M1 2" {
		t.Errorf("unknown opcode PathData = %q", got)
	}
}

// The JSON-decoded form of the props ([]any, not []float64) draws the same.
func TestCanvasShapeAttrsAcceptDecodedProps(t *testing.T) {
	typed := CanvasShapeAttrs(map[string]any{"d": []float64{0, 1, 2}, "stroke": "#000", "strokeWidth": 2.0, "dash": []float64{4, 2}})
	decoded := CanvasShapeAttrs(map[string]any{"d": []any{0.0, 1.0, 2.0}, "stroke": "#000", "strokeWidth": 2.0, "dash": []any{4.0, 2.0}})
	if strings.Join(typed, "|") != strings.Join(decoded, "|") {
		t.Errorf("typed %v != decoded %v", typed, decoded)
	}
}

func TestCanvasExport(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	node := core.Canvas(100, 50, []core.Shape{
		{Path: core.Rect(0, 0, 100, 50), Fill: "#eee"},
		{Path: core.Line(0, 50, 100, 0), Stroke: "#123456", StrokeWidth: 2, Cap: core.CapRound},
	}, core.CanvasStretch, core.Height("120px"), core.AccessibilityLabel("Rising")).Render(ctx)

	html := ExportHTML(node)
	for _, want := range []string{
		`<svg`,
		`viewBox="0 0 100 50"`,
		`preserveAspectRatio="none"`,
		`display:block; width:100%; overflow:visible; aspect-ratio:100 / 50; height:120px`,
		`role="img"`,
		`aria-label="Rising"`,
		`<path d="M0 0L100 0L100 50L0 50Z" fill="#eee"`,
		`<path d="M0 50L100 0" fill="none" stroke="#123456" stroke-width="2" vector-effect="non-scaling-stroke" stroke-linecap="round"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("export missing %s\n%s", want, html)
		}
	}
}

func TestUnlabelledCanvasExportsHidden(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	html := ExportHTML(core.Canvas(10, 10, nil).Render(ctx))
	if !strings.Contains(html, `aria-hidden="true"`) || !strings.Contains(html, `preserveAspectRatio="xMidYMid meet"`) {
		t.Errorf("unlabelled canvas export:\n%s", html)
	}
}
