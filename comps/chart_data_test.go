package comps

import (
	"encoding/json"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/rohanthewiz/grmob/core"
)

// Every chart that speaks a summary also carries its numbers
// (core.AccessibilityChart, N-010), on the same node that carries the summary
// and RoleImg — the element the web's table lives in and iOS's descriptor
// describes.

// chartDataOf renders v and returns its root's chart data, failing when the
// root is not the labelled RoleImg element or carries none.
func chartDataOf(t *testing.T, v core.View) core.ChartData {
	t.Helper()
	n := v.Render(core.NewContext())
	if n.Style == nil || n.Style.AccessibilityRole != core.RoleImg || n.Style.AccessibilityLabel == "" {
		t.Fatalf("%T: the root is not the labelled RoleImg element", v)
	}
	d, ok := n.Props["chartData"].(core.ChartData)
	if !ok {
		t.Fatalf("%T: the root carries no chartData (props %v)", v, n.Props)
	}
	// Whatever a chart puts there has to cross the bridge: encoding/json
	// refuses NaN and ±Inf.
	if _, err := json.Marshal(n.Props); err != nil {
		t.Fatalf("%T: the chart data does not marshal: %v", v, err)
	}
	return d
}

// texts is one series' (category, text) pairs.
func texts(s core.ChartDataSeries) [][2]string {
	var out [][2]string
	for _, p := range s.Points {
		out = append(out, [2]string{p.X, p.Text})
	}
	return out
}

func TestEveryChartCarriesItsData(t *testing.T) {
	day := time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)
	for _, v := range []core.View{
		LineChart{Labels: []string{"a"}, Series: []ChartSeries{{Values: []float64{1}}}},
		AreaChart{Labels: []string{"a"}, Series: []ChartSeries{{Values: []float64{1}}}},
		BarChart{Labels: []string{"a"}, Series: []ChartSeries{{Values: []float64{1}}}},
		BarChart{Horizontal: true, Labels: []string{"a"}, Series: []ChartSeries{{Values: []float64{1}}}},
		CandlestickChart{Candles: []Candle{{1, 3, 0.5, 2}}},
		Histogram{Values: []float64{1, 2, 2, 3}},
		ScatterChart{Series: []ScatterSeries{{Points: []ChartPoint{{1, 2}}}}},
		DonutChart{Slices: []ChartSlice{{Label: "a", Value: 1}}},
		FunnelChart{Stages: []FunnelStage{{Label: "a", Value: 1}}},
		Heatmap{Values: [][]float64{{1}}},
		CalendarHeatmap{End: day, Weeks: 2, Days: []DayValue{{Day: day, Value: 3}}},
		RadarChart{Axes: []string{"a", "b", "c"}, Series: []ChartSeries{{Values: []float64{1, 2, 3}}}},
	} {
		chartDataOf(t, v)
	}
}

// A stacked chart draws running totals; its table reads the input, which is
// what a reader asks for and what the summary already speaks.
func TestAStackedChartsDataIsItsInput(t *testing.T) {
	d := chartDataOf(t, LineChart{
		Labels: []string{"Mon", "Tue"}, Stacked: true, Area: true,
		Series: []ChartSeries{{Name: "Web", Values: []float64{3, 4}}, {Name: "App", Values: []float64{1, 2}}},
	})
	if got := texts(d.Series[1]); !reflect.DeepEqual(got, [][2]string{{"Mon", "1"}, {"Tue", "2"}}) {
		t.Errorf("App's points = %v, want its own values, not Web plus App", got)
	}
	if !d.Series[0].Continuous {
		t.Error("a line chart's series are continuous")
	}
}

// A missing value is a point left out — never NaN, which the wire refuses —
// and the chart's own Format spells the rest; an unlabelled row is named by
// its position, a lone unnamed series by the subject.
func TestChartDataSkipsGapsAndKeepsTheChartsFormat(t *testing.T) {
	money := func(v float64) string { return "$" + formatValue(v) }
	d := chartDataOf(t, BarChart{
		Subject: "Revenue", Labels: []string{"Jan"}, Format: money,
		Series: []ChartSeries{{Values: []float64{120, math.NaN(), 180}}},
	})
	if want := []string{"Jan", "2", "3"}; !reflect.DeepEqual(d.X.Categories, want) {
		t.Errorf("categories = %v, want %v", d.X.Categories, want)
	}
	if d.Series[0].Name != "Revenue" {
		t.Errorf("a lone unnamed series is named %q, want the subject", d.Series[0].Name)
	}
	if got := texts(d.Series[0]); !reflect.DeepEqual(got, [][2]string{{"Jan", "$120"}, {"3", "$180"}}) {
		t.Errorf("points = %v", got)
	}
	if d.Series[0].Continuous {
		t.Error("bars are separate marks, not a continuous line")
	}
	if d.Y.Min > 0 || d.Y.Max < 180 {
		t.Errorf("the value axis %v..%v does not cover the data", d.Y.Min, d.Y.Max)
	}
}

// A candle is a row with four prices; an invalid one is missing from all four.
func TestCandlestickDataIsFourPrices(t *testing.T) {
	d := chartDataOf(t, CandlestickChart{
		Labels:  []string{"Mon", "Tue"},
		Candles: []Candle{{1, 3, 0.5, 2}, {math.NaN(), 1, 1, 1}},
	})
	var names []string
	for _, s := range d.Series {
		names = append(names, s.Name)
		if len(s.Points) != 1 || s.Points[0].X != "Mon" {
			t.Errorf("%s holds %v, want Monday's price alone", s.Name, s.Points)
		}
	}
	if want := []string{"Open", "High", "Low", "Close"}; !reflect.DeepEqual(names, want) {
		t.Errorf("series = %v, want %v", names, want)
	}
}

// A histogram's rows are its bins, named by their edges, holding counts.
func TestHistogramDataIsItsBins(t *testing.T) {
	d := chartDataOf(t, Histogram{Values: []float64{1, 1, 2, 9}, Bins: 2})
	total := 0.0
	for _, p := range d.Series[0].Points {
		total += p.Y
	}
	if total != 4 || d.Series[0].Name != "Count" {
		t.Errorf("series %q counts %v observations, want Count of 4", d.Series[0].Name, total)
	}
	for _, c := range d.X.Categories {
		if !containsRune(c, '–') {
			t.Errorf("bin %q is not named by its two edges", c)
		}
	}
}

// The scatter is the numeric case: no categories, each point its own x.
func TestScatterDataIsNumeric(t *testing.T) {
	d := chartDataOf(t, ScatterChart{Series: []ScatterSeries{{Name: "A", Points: []ChartPoint{{1.5, 2}}}}})
	if len(d.X.Categories) != 0 || d.X.Title == "" || d.Y.Title == "" {
		t.Fatalf("x axis = %+v, want numeric with titles for the table's heads", d.X)
	}
	if p := d.Series[0].Points[0]; p.XValue != 1.5 || p.XText != "1.5" || p.Y != 2 {
		t.Errorf("point = %+v", p)
	}
}

func containsRune(s string, r rune) bool {
	for _, c := range s {
		if c == r {
			return true
		}
	}
	return false
}
