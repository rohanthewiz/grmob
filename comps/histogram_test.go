package comps

import (
	"math"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// Twenty values over 0–95 with Bins 10: nice edges every 10, the half-open
// rule at an interior edge, and the maximum in the last bin.
func TestHistogramBinsOnNiceEdges(t *testing.T) {
	values := []float64{0, 3, 9.9, 10, 10, 12, 25, 31, 33, 38, 41, 47, 52, 58, 60, 66, 71, 83, 90, 95, math.NaN()}
	b := Histogram{Values: values, Bins: 10}.bin()

	if b.lo != 0 || b.step != 10 || len(b.counts) != 10 {
		t.Fatalf("bins = from %v by %v × %d, want from 0 by 10 × 10", b.lo, b.step, len(b.counts))
	}
	want := []int{3, 3, 1, 3, 2, 2, 2, 1, 1, 2}
	for i, n := range want {
		if b.counts[i] != n {
			t.Errorf("bin %d [%v, %v) = %d, want %d", i, b.edge(i), b.edge(i+1), b.counts[i], n)
		}
	}
	if b.total != 20 {
		t.Errorf("total = %d, want the 20 finite values", b.total)
	}
}

// Sturges' rule when Bins is zero: 20 values → ⌈log₂ 20⌉ + 1 = 6 at most.
func TestHistogramDefaultsToSturges(t *testing.T) {
	values := make([]float64, 20)
	for i := range values {
		values[i] = float64(i * 5)
	}
	if nb := len(Histogram{Values: values}.bin().counts); nb > 6 || nb < 3 {
		t.Errorf("bins = %d, want Sturges' 6 or the nice count under it", nb)
	}
}

// One element with a sentence, bars touching on whole-count ticks, and the
// edges under the plot on the point axis.
func TestHistogramIsOneElementOverANumericAxis(t *testing.T) {
	_, n := renderDebug(t, Histogram{
		Subject: "Response time (ms)",
		Values:  []float64{100, 110, 120, 130, 150, 160, 210, 350},
		Bins:    4,
	})
	if n.Style.AccessibilityRole != core.RoleImg {
		t.Errorf("role = %q, want img", n.Style.AccessibilityRole)
	}
	want := "Response time (ms): 8 values in 3 bins of 100 from 100 to 400; most, 6, between 100 and 200."
	if got := n.Style.AccessibilityLabel; got != want {
		t.Errorf("summary = %q\nwant      %q", got, want)
	}
	// n bins have n+1 edges, each drawn under the plot.
	for _, s := range []string{"100", "200", "300", "400"} {
		if findText(n, s) == nil {
			t.Errorf("edge label %q missing", s)
		}
	}
	// A peak of 6 ticks in whole counts, never 0.5.
	if findText(n, "0.5") != nil || findText(n, "1.5") != nil {
		t.Error("the count axis should tick in whole counts")
	}

	canvas := canvasOf(n)
	var bars *core.Node
	for _, s := range canvas.Children {
		if s.Props["fill"] == core.DefaultTheme.Colors.ChartColors()[0] {
			bars = s
		}
	}
	if bars == nil {
		t.Fatal("no bar shape in chart slot 1's colour")
	}
	if got := subpaths(bars); got != 3 {
		t.Errorf("bars = %d, want one per non-empty bin", got)
	}
}

func TestHistogramWithNoValuesSaysSo(t *testing.T) {
	_, n := renderDebug(t, Histogram{Subject: "Scores", Values: []float64{math.NaN()}})
	if got := n.Style.AccessibilityLabel; got != "Scores: no values." {
		t.Errorf("summary = %q", got)
	}
}
