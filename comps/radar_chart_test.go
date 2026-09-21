package comps

import (
	"math"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

var playerAxes = []string{"Speed", "Power", "Stamina", "Skill", "Vision"}

// pxValue reads a "12.5px" dimension.
func pxValue(t *testing.T, s string) float64 {
	t.Helper()
	if s == "" {
		return 0
	}
	return pxOf(t, s)
}

// layerOf finds the stack layer holding text: the label's translated box.
func layerOf(n *core.Node, text string) *core.Node {
	return findFirst(n, func(n *core.Node) bool {
		return n.Type == "Column" && n.Style != nil && (n.Style.TranslateX != "" || n.Style.TranslateY != "") &&
			len(n.Children) == 1 && n.Children[0].Props["content"] == text
	})
}

func TestRadarChartIsOneElementWithASummary(t *testing.T) {
	_, n := renderDebug(t, RadarChart{
		Subject: "Player",
		Axes:    playerAxes,
		Max:     10,
		Series:  []ChartSeries{{Name: "Ade", Values: []float64{8, 6, 7, 9, 5}}},
	})
	if n.Style.AccessibilityRole != core.RoleImg {
		t.Errorf("role = %q, want img", n.Style.AccessibilityRole)
	}
	want := "Player: Ade, Speed 8, Power 6, Stamina 7, Skill 9, Vision 5."
	if got := n.Style.AccessibilityLabel; got != want {
		t.Errorf("summary = %q\nwant      %q", got, want)
	}
	for _, s := range playerAxes {
		if findText(n, s) == nil {
			t.Errorf("axis label %q missing", s)
		}
	}
	// One series needs no legend; the scale's inner rings are written.
	for _, s := range []string{"2.5", "5.0", "7.5"} {
		if findText(n, s) == nil {
			t.Errorf("ring value %q missing", s)
		}
	}
}

// The polygon's corners sit value/Max of the way out along each spoke, the
// first spoke straight up and the rest clockwise.
func TestRadarChartPolygonFollowsTheSpokes(t *testing.T) {
	_, n := renderDebug(t, RadarChart{
		Axes:   []string{"N", "E", "S", "W"},
		Max:    10,
		Series: []ChartSeries{{Values: []float64{10, 5, 0, 20}}},
	})
	canvas := canvasOf(n)
	if len(canvas.Children) != 3 {
		t.Fatalf("shapes = %d, want the grid, a polygon and its markers", len(canvas.Children))
	}
	poly := canvas.Children[1]
	xs, ys := pathXs(poly), pathYs(poly)
	want := [][2]float64{
		{50, 50 - radarRim},   // N at the rim
		{50 + radarRim/2, 50}, // E half way out
		{50, 50},              // S at the centre
		{50 - radarRim, 50},   // W over Max, held at the rim
	}
	for k, w := range want {
		if math.Abs(xs[k]-w[0]) > 0.01 || math.Abs(ys[k]-w[1]) > 0.01 {
			t.Errorf("corner %d = (%v, %v), want (%v, %v)", k, xs[k], ys[k], w[0], w[1])
		}
	}
	if poly.Props["fill"] != nil {
		t.Error("an outline until Filled")
	}
	if got := subpaths(canvas.Children[2]); got != 4 {
		t.Errorf("markers = %d, want one per axis", got)
	}
}

// Rim labels are centred layers moved by px: the top one straight up and
// centred, a right-hand one start-aligned with its near edge outside the rim.
func TestRadarChartLabelsArePlacedByTranslate(t *testing.T) {
	const size, width = 200.0, 60.0
	_, n := renderDebug(t, RadarChart{
		Axes: []string{"N", "E", "S", "W"}, Size: size, LabelWidth: width, Max: 1,
		Series: []ChartSeries{{Values: []float64{1, 1, 1, 1}}},
	})
	rim := radarRim * size / chartView

	north := layerOf(n, "N")
	if north == nil {
		t.Fatal("no layer for N")
	}
	if x := pxValue(t, north.Style.TranslateX); x != 0 {
		t.Errorf("N is moved %v sideways, want 0", x)
	}
	if y, want := pxValue(t, north.Style.TranslateY), -(rim + radarLabelGap + chartLabelLine/2); math.Abs(y-want) > 0.06 {
		t.Errorf("N is moved %v, want %v: its bottom edge a gap above the rim", y, want)
	}
	if a := north.Children[0].Style.Align; a != core.AlignCenter {
		t.Errorf("N aligns %q, want centre", a)
	}

	east := layerOf(n, "E")
	if x, want := pxValue(t, east.Style.TranslateX), rim+radarLabelGap+width/2; math.Abs(x-want) > 0.06 {
		t.Errorf("E is moved %v, want %v: its start edge a gap outside the rim", x, want)
	}
	if a := east.Children[0].Style.Align; a != core.AlignStart {
		t.Errorf("E aligns %q, want start", a)
	}
	if a := layerOf(n, "W").Children[0].Style.Align; a != core.AlignEnd {
		t.Errorf("W aligns %q, want end", a)
	}

	// The stack is pinned wide and tall enough for every label.
	stack := findFirst(n, func(n *core.Node) bool { return n.Type == "ZStack" })
	if w := pxValue(t, stack.Style.Width); w != size+2*(width+radarLabelGap) {
		t.Errorf("stack width = %v", w)
	}
}

func TestRadarChartFilledLegendAndAutoMax(t *testing.T) {
	_, n := renderDebug(t, RadarChart{
		Axes:   playerAxes,
		Filled: true,
		Series: []ChartSeries{
			{Name: "Ade", Values: []float64{8, 6, 7, 9, 5}},
			{Name: "Bo", Values: []float64{5, 9, 6, 4, 8}},
		},
	})
	first := core.DefaultTheme.Colors.ChartColors()[0]
	if got := canvasOf(n).Children[1].Props["fill"]; got != first+radarFillAlpha {
		t.Errorf("fill = %v, want %s", got, first+radarFillAlpha)
	}
	if findText(n, "Ade") == nil || findText(n, "Bo") == nil {
		t.Error("two series want a legend")
	}
	// A high of 9 over 4 rings widens to 10: rings at 2.5, 5, 7.5.
	for _, s := range []string{"2.5", "5.0", "7.5"} {
		if findText(n, s) == nil {
			t.Errorf("ring value %q missing", s)
		}
	}
}

func TestRadarStep(t *testing.T) {
	for _, c := range [][2]float64{{2.25, 2.5}, {0.3, 0.5}, {1, 1}, {2.6, 5}, {7, 10}, {120, 200}, {0, 1}, {0.25, 0.25}} {
		if got := radarStep(c[0]); math.Abs(got-c[1]) > 1e-12 {
			t.Errorf("radarStep(%v) = %v, want %v", c[0], got, c[1])
		}
	}
}

func TestRadarChartTooFewAxes(t *testing.T) {
	core.SetDebugMode(true)
	core.ClearConcerns()
	t.Cleanup(func() { core.SetDebugMode(false); core.ClearConcerns() })
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := RadarChart{Axes: []string{"a", "b"}, Series: []ChartSeries{{Values: []float64{1, 2}}}}.Render(ctx)
	ctx.EndRenderPass()
	if dump := core.DumpConcerns(); !strings.Contains(dump, ConcernRadarChartTooFewAxes) {
		t.Errorf("want %s, got:\n%s", ConcernRadarChartTooFewAxes, dump)
	}
	// The rim, and the series' two slots kept empty.
	if got := len(canvasOf(n).Children); got != 3 {
		t.Errorf("shapes = %d, want 3", got)
	}
	if findText(n, "a") != nil {
		t.Error("no spokes, so no rim labels")
	}
}

func TestRadarChartLongAndMissingSummaries(t *testing.T) {
	axes := make([]string, 10)
	vals := make([]float64, 10)
	for i := range axes {
		axes[i], vals[i] = string(rune('a'+i)), float64(i+1)
	}
	vals[3] = math.NaN()
	_, n := renderDebug(t, RadarChart{Axes: axes, Series: []ChartSeries{{Values: vals}, {Name: "none"}}})
	want := "9 axes, low 1 in a, high 10 in j; none, no data."
	if n.Style.AccessibilityLabel != want {
		t.Errorf("summary = %q\nwant      %q", n.Style.AccessibilityLabel, want)
	}
}
