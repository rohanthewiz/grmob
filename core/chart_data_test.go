package core

import (
	"encoding/json"
	"math"
	"testing"
)

// AccessibilityChart stores a copy the wire can carry: non-finite points
// dropped, non-finite bounds zeroed, and nothing shared with the caller's
// slices, so reusing them for the next pass cannot rewrite a rendered node.
func TestAccessibilityChartStoresAFiniteCopy(t *testing.T) {
	cats := []string{"a", "b"}
	pts := []ChartDataPoint{{X: "a", Y: 1}, {X: "b", Y: math.NaN()}, {X: "b", Y: math.Inf(1)}}
	d := ChartData{
		X:      ChartAxis{Categories: cats},
		Y:      ChartAxis{Min: math.Inf(-1), Max: 5},
		Series: []ChartDataSeries{{Name: "s", Points: pts}},
	}
	n := Column(AccessibilityChart(d)).Render(NewContext())
	got := n.Props["chartData"].(ChartData)

	if len(got.Series[0].Points) != 1 || got.Series[0].Points[0].X != "a" {
		t.Errorf("points = %+v, want only the finite one", got.Series[0].Points)
	}
	if got.Y.Min != 0 || got.Y.Max != 5 {
		t.Errorf("y axis = %+v, want the infinite bound zeroed", got.Y)
	}
	if _, err := json.Marshal(n.Props); err != nil {
		t.Fatalf("chart data does not marshal: %v", err)
	}

	cats[0], pts[0].Y = "changed", 99
	if got.X.Categories[0] != "a" || got.Series[0].Points[0].Y != 1 {
		t.Error("the stored data shares the caller's slices")
	}
}
