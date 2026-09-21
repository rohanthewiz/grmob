package comps

import (
	"math"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

var weekOfCandles = []Candle{
	{Open: 102, High: 108, Low: 101, Close: 107},
	{Open: 107, High: 109, Low: 98, Close: 99},
	{Open: 99, High: 104, Low: 97, Close: 103},
}

// shapeWith finds the canvas child carrying key = value: a candle kind's
// wicks by "stroke", its bodies by "fill".
func shapeWith(canvas *core.Node, key, value string) *core.Node {
	for _, s := range canvas.Children {
		if s.Props[key] == value {
			return s
		}
	}
	return nil
}

// pathYs are the y coordinates of a shape's moves and lines, in order.
func pathYs(shape *core.Node) []float64 {
	d, _ := shape.Props["d"].([]float64)
	var out []float64
	for i := 0; i < len(d); {
		switch d[i] {
		case core.PathMove, core.PathLine:
			out = append(out, d[i+2])
			i += 3
		case core.PathCubic:
			i += 7
		default:
			i++
		}
	}
	return out
}

func TestCandlestickChartIsOneElementWithASummary(t *testing.T) {
	_, n := renderDebug(t, CandlestickChart{
		Subject: "ACME",
		Labels:  []string{"Mon", "Tue", "Wed"},
		Candles: weekOfCandles,
	})
	if n.Style.AccessibilityRole != core.RoleImg {
		t.Errorf("role = %q, want img", n.Style.AccessibilityRole)
	}
	want := "ACME: Mon open 102, close 107; Tue open 107, close 99; Wed open 99, close 103; " +
		"low 97 in Wed, high 109 in Tue; 2 up, 1 down."
	if got := n.Style.AccessibilityLabel; got != want {
		t.Errorf("summary = %q\nwant      %q", got, want)
	}
	for _, s := range []string{"Mon", "Tue", "Wed"} {
		if findText(n, s) == nil {
			t.Errorf("period label %q missing", s)
		}
	}
}

// A price axis from zero would flatten every candle into a sliver: the ticks
// bracket the data and zero is not among them.
func TestCandlestickChartAxisDoesNotStartAtZero(t *testing.T) {
	_, n := renderDebug(t, CandlestickChart{Candles: weekOfCandles})
	if findText(n, "0") != nil {
		t.Error("the price axis should not include zero for prices near 100")
	}
	if findText(n, "110") == nil {
		t.Error("want a tick at 110, the first nice value above the high of 109")
	}
}

// Rising and falling candles are split by colour into one wick path and one
// body path each, with the theme's Success and Error by default, and both
// colours are the caller's to swap.
func TestCandlestickChartColoursByDirection(t *testing.T) {
	_, n := renderDebug(t, CandlestickChart{Candles: weekOfCandles})
	canvas := canvasOf(n)
	up, down := core.DefaultTheme.Colors.Success, core.DefaultTheme.Colors.Error
	for _, c := range []struct {
		key, color string
		want       int
	}{{"fill", up, 2}, {"fill", down, 1}, {"stroke", up, 2}, {"stroke", down, 1}} {
		s := shapeWith(canvas, c.key, c.color)
		if s == nil {
			t.Fatalf("no shape with %s %s", c.key, c.color)
		}
		if got := subpaths(s); got != c.want {
			t.Errorf("%s %s holds %d candles, want %d", c.key, c.color, got, c.want)
		}
	}

	_, n = renderDebug(t, CandlestickChart{Candles: weekOfCandles, UpColor: "#cc0000", DownColor: "#00aa00"})
	if s := shapeWith(canvasOf(n), "fill", "#cc0000"); s == nil || subpaths(s) != 2 {
		t.Error("UpColor should ink the two rising bodies")
	}
}

// A doji's body is one px tall: chartView/Height viewBox units, since the
// stretched canvas maps y linearly onto Height px.
func TestCandlestickChartDojiKeepsAOnePxBody(t *testing.T) {
	_, n := renderDebug(t, CandlestickChart{
		Height:  200,
		Candles: []Candle{{Open: 50, High: 60, Low: 40, Close: 50}},
	})
	body := shapeWith(canvasOf(n), "fill", core.DefaultTheme.Colors.Success)
	if body == nil {
		t.Fatal("a doji counts as rising and should have a body")
	}
	ys := pathYs(body)
	if len(ys) != 4 {
		t.Fatalf("body has %d corners, want 4", len(ys))
	}
	if got, want := ys[2]-ys[0], chartView/200; math.Abs(got-want) > 0.011 {
		t.Errorf("doji body is %.3f units tall, want one px = %.3f", got, want)
	}
}

// A candle with a NaN keeps its slot and draws nothing, and a chart with no
// candles at all still renders.
func TestCandlestickChartSkipsInvalidCandles(t *testing.T) {
	_, n := renderDebug(t, CandlestickChart{
		Labels:  []string{"a", "b"},
		Candles: []Candle{{Open: math.NaN(), High: 2, Low: 1, Close: 2}, {Open: 1, High: 3, Low: 1, Close: 2}},
	})
	if got := subpaths(shapeWith(canvasOf(n), "fill", core.DefaultTheme.Colors.Success)); got != 1 {
		t.Errorf("bodies = %d, want only the valid candle", got)
	}
	want := "b open 1, close 2; low 1 in b, high 3 in b; 1 up, 0 down."
	if n.Style.AccessibilityLabel != want {
		t.Errorf("summary = %q\nwant      %q", n.Style.AccessibilityLabel, want)
	}

	_, n = renderDebug(t, CandlestickChart{Subject: "ACME"})
	if n.Style.AccessibilityLabel != "ACME: no data." {
		t.Errorf("empty summary = %q", n.Style.AccessibilityLabel)
	}
}

// Past candleSummaryLimit periods the summary gives the span, not every one.
func TestCandlestickChartLongSummary(t *testing.T) {
	candles := make([]Candle, 8)
	for i := range candles {
		o := float64(10 + i)
		candles[i] = Candle{Open: o, High: o + 2, Low: o - 1, Close: o + 1}
	}
	_, n := renderDebug(t, CandlestickChart{Candles: candles, ShowLegend: true})
	want := "8 periods, from open 10 in period 1 to close 18 in period 8; " +
		"low 9 in period 1, high 19 in period 8; 8 up, 0 down."
	if n.Style.AccessibilityLabel != want {
		t.Errorf("summary = %q\nwant      %q", n.Style.AccessibilityLabel, want)
	}
	if findText(n, "Up") == nil || findText(n, "Down") == nil {
		t.Error("ShowLegend should key both colours")
	}
}
