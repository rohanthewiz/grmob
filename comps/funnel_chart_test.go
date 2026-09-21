package comps

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

var checkoutStages = []FunnelStage{
	{Label: "Visited", Value: 1200},
	{Label: "Signed up", Value: 744},
	{Label: "Paid", Value: 93},
}

// pathXs are the x coordinates of a shape's moves and lines, in order.
func pathXs(shape *core.Node) []float64 {
	d, _ := shape.Props["d"].([]float64)
	var out []float64
	for i := 0; i < len(d); {
		switch d[i] {
		case core.PathMove, core.PathLine:
			out = append(out, d[i+1])
			i += 3
		case core.PathCubic:
			i += 7
		default:
			i++
		}
	}
	return out
}

func TestFunnelChartIsOneElementWithASummary(t *testing.T) {
	_, n := renderDebug(t, FunnelChart{Subject: "Checkout", Stages: checkoutStages, ShowRates: true})
	if n.Style.AccessibilityRole != core.RoleImg {
		t.Errorf("role = %q, want img", n.Style.AccessibilityRole)
	}
	want := "Checkout: Visited 1200; Signed up 744, 62% of the step before; " +
		"Paid 93, 13% of the step before; 8% overall."
	if got := n.Style.AccessibilityLabel; got != want {
		t.Errorf("summary = %q\nwant      %q", got, want)
	}
	for _, s := range []string{"Visited", "Signed up", "Paid", "1200", "744", "93", "62%", "13%"} {
		if findText(n, s) == nil {
			t.Errorf("text %q missing", s)
		}
	}
}

func TestFunnelChartRatesAreOptIn(t *testing.T) {
	_, n := renderDebug(t, FunnelChart{Stages: checkoutStages})
	if findText(n, "62%") != nil {
		t.Error("rates drawn without ShowRates")
	}
}

// Each band is as wide at the top as its own value and at the bottom as the
// next one's, centred; the last is a rectangle.
func TestFunnelChartBandsTaperIntoTheNextStage(t *testing.T) {
	_, n := renderDebug(t, FunnelChart{Stages: []FunnelStage{
		{Label: "a", Value: 100}, {Label: "b", Value: 50}, {Label: "c", Value: 20},
	}})
	canvas := canvasOf(n)
	if len(canvas.Children) != 3 {
		t.Fatalf("shapes = %d, want one per stage", len(canvas.Children))
	}
	widths := func(i int) (top, bottom float64) {
		xs := pathXs(canvas.Children[i])
		return xs[1] - xs[0], xs[2] - xs[3]
	}
	for i, want := range [][2]float64{{100, 50}, {50, 20}, {20, 20}} {
		top, bottom := widths(i)
		if top != want[0] || bottom != want[1] {
			t.Errorf("stage %d spans %v → %v, want %v → %v", i, top, bottom, want[0], want[1])
		}
		if xs := pathXs(canvas.Children[i]); xs[0]+xs[1] != chartView {
			t.Errorf("stage %d is not centred: %v..%v", i, xs[0], xs[1])
		}
	}
}

// A stage that grows is drawn as given and reported, unless the caller says
// the data is meant.
func TestFunnelChartStageThatGrows(t *testing.T) {
	grows := []FunnelStage{{Label: "Cart", Value: 40}, {Label: "Checkout", Value: 60}}

	core.SetDebugMode(true)
	core.ClearConcerns()
	t.Cleanup(func() { core.SetDebugMode(false); core.ClearConcerns() })
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := FunnelChart{Stages: grows, ShowRates: true}.Render(ctx)
	ctx.EndRenderPass()
	if dump := core.DumpConcerns(); !strings.Contains(dump, ConcernFunnelChartStageGrows) {
		t.Errorf("want %s, got:\n%s", ConcernFunnelChartStageGrows, dump)
	}
	if findText(n, "150%") == nil {
		t.Error("a growing step's rate should read over 100%, unclamped")
	}
	// Widths are shares of the largest stage, so the second fills the width
	// and the first flares out to meet it.
	if xs := pathXs(canvasOf(n).Children[0]); xs[2]-xs[3] != chartView {
		t.Errorf("first band's foot = %v wide, want the full %v", xs[2]-xs[3], chartView)
	}

	renderDebug(t, FunnelChart{Stages: grows, AllowIncrease: true})
}

// The default fill is one hue fading; a stage's own colour wins.
func TestFunnelChartFadesOneHue(t *testing.T) {
	stages := append([]FunnelStage(nil), checkoutStages...)
	stages[1].Color = "#123456"
	_, n := renderDebug(t, FunnelChart{Stages: stages})
	base := core.DefaultTheme.Colors.ChartColors()[0]
	canvas := canvasOf(n)
	if got := canvas.Children[0].Props["fill"]; got != base+"ff" {
		t.Errorf("first stage = %v, want %sff", got, base)
	}
	if got := canvas.Children[1].Props["fill"]; got != "#123456" {
		t.Errorf("second stage = %v, want its own colour", got)
	}
	if got := canvas.Children[2].Props["fill"]; got != base+"59" {
		t.Errorf("last stage = %v, want %s at funnelMinAlpha (59)", got, base)
	}
}

// Nothing, zeros and a from-zero step all render without a rate or a panic.
func TestFunnelChartDegenerateData(t *testing.T) {
	_, n := renderDebug(t, FunnelChart{Subject: "Checkout"})
	if n.Style.AccessibilityLabel != "Checkout: no data." {
		t.Errorf("empty summary = %q", n.Style.AccessibilityLabel)
	}
	_, n = renderDebug(t, FunnelChart{ShowRates: true, AllowIncrease: true,
		Stages: []FunnelStage{{Label: "a", Value: 0}, {Label: "b", Value: -4}, {Label: "c", Value: 3}}})
	if want := "a 0; b 0; c 3."; n.Style.AccessibilityLabel != want {
		t.Errorf("summary = %q, want %q", n.Style.AccessibilityLabel, want)
	}
}
