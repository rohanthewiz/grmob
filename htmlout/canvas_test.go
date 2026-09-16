package htmlout

import (
	"regexp"
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
	typed := CanvasShapeAttrs(map[string]any{"d": []float64{0, 1, 2}, "stroke": "#000", "strokeWidth": 2.0, "dash": []float64{4, 2}}, "g")
	decoded := CanvasShapeAttrs(map[string]any{"d": []any{0.0, 1.0, 2.0}, "stroke": "#000", "strokeWidth": 2.0, "dash": []any{4.0, 2.0}}, "g")
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

// A gradient shape exports a leading <defs> chrome element holding a
// userSpaceOnUse paint server, and its <path> fills by reference; the JSON-
// decoded prop forms export the same; a flat canvas gets no <defs>.
func TestCanvasGradientExport(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	node := core.Canvas(100, 50, []core.Shape{
		{Path: core.Rect(0, 0, 100, 50), Fill: "#eee"},
		{Path: core.Rect(0, 0, 100, 50), FillGradient: core.LinearGradientFill(0, 0, 0, 50, core.Stop(0, "#2A78D666"), core.Stop(1, "#2A78D600"))},
		{Path: core.Circle(50, 25, 20), FillRule: core.FillEvenOdd, FillGradient: core.RadialGradientFill(50, 25, 20, core.Stop(0, "#ffffff"), core.Stop(1, "#000000"))},
	}).Render(ctx)

	// Compared with the pretty-printer's whitespace between tags removed and
	// case folded: the element builder lowercases tag names, and an HTML
	// parser restores SVG's camel case (linearGradient) in foreign content.
	html := strings.ToLower(regexp.MustCompile(`>\s+<`).ReplaceAllString(ExportHTML(node), "><"))
	for _, want := range []string{
		`<defs data-grmob-chrome="gradients"><linearGradient id="grmob-root-fill-1" gradientUnits="userSpaceOnUse" x1="0" y1="0" x2="0" y2="50"><stop offset="0" stop-color="#2A78D666"></stop><stop offset="1" stop-color="#2A78D600"></stop></linearGradient><radialGradient id="grmob-root-fill-2" gradientUnits="userSpaceOnUse" cx="50" cy="25" r="20">`,
		`fill="#eee"`,
		`fill="url(#grmob-root-fill-1)"`,
		`fill="url(#grmob-root-fill-2)" fill-rule="evenodd"`,
	} {
		if !strings.Contains(html, strings.ToLower(want)) {
			t.Errorf("export missing %s\n%s", want, html)
		}
	}
	if strings.Index(html, "<defs") > strings.Index(html, "<path") {
		t.Error("the <defs> must lead the shapes, where chromeOffset expects chrome")
	}

	decoded := map[string]any{"gradient": "linear", "gradientAt": []any{0.0, 0.0, 0.0, 50.0},
		"gradientStops": []any{0.0, 1.0}, "gradientColors": []any{"#2A78D666", "#2A78D600"}}
	tagT, attrsT, stopsT := CanvasGradient(node.Children[1].Props, "g")
	tagD, attrsD, stopsD := CanvasGradient(decoded, "g")
	if tagT != tagD || strings.Join(attrsT, "|") != strings.Join(attrsD, "|") || len(stopsT) != len(stopsD) {
		t.Errorf("typed %s %v %v != decoded %s %v %v", tagT, attrsT, stopsT, tagD, attrsD, stopsD)
	}
	if tag, _, _ := CanvasGradient(map[string]any{"gradient": "linear", "gradientAt": []float64{0, 0, 1, 1},
		"gradientStops": []float64{0, 1}, "gradientColors": []string{"#000"}}, "g"); tag != "" {
		t.Error("mismatched stops and colours produced a gradient")
	}

	flat := ExportHTML(core.Canvas(10, 10, []core.Shape{{Path: core.Rect(0, 0, 10, 10), Fill: "#000"}}).Render(ctx))
	if strings.Contains(flat, "<defs") {
		t.Errorf("a canvas with no gradient wrote a <defs>: %s", flat)
	}
}
