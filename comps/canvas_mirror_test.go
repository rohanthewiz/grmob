package comps

import (
	"testing"
	"time"

	"github.com/rohanthewiz/grmob/core"
)

// Which widgets mirror their drawing under RTL (core.CanvasMirrorsRTL), and
// which must not. The prop is opt-in, so this census is the only thing that
// says a new chart remembered it, or that a picture of the physical world
// did not pick it up.
//
// A widget mirrors when its drawing has a reading direction that its own
// labels, a Row of stars or a seek slider beside it already follow: time or
// category along x, a value axis that grows from the leading edge, a
// radar's clockwise spokes. It stays fixed when the drawing has no
// direction (a donut, a gauge, a symmetric funnel) or when its geometry is a
// fact about the world (a QR code must decode).
func TestCanvasMirrorCensus(t *testing.T) {
	series := []ChartSeries{{Name: "a", Values: []float64{1, 3, 2}}}
	labels := []string{"x", "y", "z"}
	mirrored := map[string]core.View{
		"LineChart":            LineChart{Labels: labels, Series: series},
		"AreaChart":            AreaChart{Labels: labels, Series: series},
		"BarChart":             BarChart{Labels: labels, Series: series},
		"BarChart{Horizontal}": BarChart{Labels: labels, Series: series, Horizontal: true},
		"Histogram":            Histogram{Values: []float64{1, 2, 2, 3, 5}},
		"ScatterChart":         ScatterChart{Series: []ScatterSeries{{Name: "a", Points: []ChartPoint{{1, 2}, {3, 4}}}}},
		"CandlestickChart":     CandlestickChart{Candles: []Candle{{1, 3, 0.5, 2}, {2, 4, 1, 3}}, Labels: []string{"1", "2"}},
		"RadarChart":           RadarChart{Axes: []string{"a", "b", "c"}, Series: series},
		"Heatmap":              Heatmap{Values: [][]float64{{1, 2}, {3, 4}}},
		"Sparkline":            Sparkline{Values: []float64{1, 3, 2}},
		"Waveform":             Waveform{Peaks: []float64{0.2, 0.8, 0.5}, Progress: 0.5},
		"Rating{Halves}":       Rating{Value: 2.5, Halves: true, ReadOnly: true},
	}
	fixed := map[string]core.View{
		"DonutChart": DonutChart{Slices: []ChartSlice{{Label: "a", Value: 1}, {Label: "b", Value: 2}}},
		"PieChart":   PieChart{Slices: []ChartSlice{{Label: "a", Value: 1}, {Label: "b", Value: 2}}},
		"Gauge":      Gauge{Value: 0.4, Label: "g"},
		"FunnelChart": FunnelChart{Stages: []FunnelStage{
			{Label: "a", Value: 10}, {Label: "b", Value: 5}}},
		"QRCode": QRCode{Data: "https://example.com"},
	}
	canvases := func(n *core.Node) []*core.Node {
		var out []*core.Node
		var walk func(*core.Node)
		walk = func(n *core.Node) {
			if n.Type == "Canvas" {
				out = append(out, n)
			}
			for _, c := range n.Children {
				walk(c)
			}
		}
		walk(n)
		return out
	}
	for name, v := range mirrored {
		cs := canvases(renderView(t, v))
		if len(cs) == 0 {
			t.Errorf("%s drew no Canvas; the census is stale", name)
		}
		for _, c := range cs {
			if c.Props["mirror"] != true {
				t.Errorf("%s has a Canvas without CanvasMirrorsRTL: under RTL its labels mirror and its drawing would not", name)
			}
		}
	}
	for name, v := range fixed {
		cs := canvases(renderView(t, v))
		if len(cs) == 0 {
			t.Errorf("%s drew no Canvas; the census is stale", name)
		}
		for _, c := range cs {
			if _, ok := c.Props["mirror"]; ok {
				t.Errorf("%s's Canvas mirrors; its drawing has no reading direction to follow", name)
			}
		}
	}
	// AnalogClock is not a Canvas at all (it is boxes turned by Rotate), so a
	// clock face cannot pick the prop up by accident. Checked so that a
	// rewrite onto a Canvas has to come through this census.
	if cs := canvases(renderView(t, AnalogClock{Time: time.Date(2026, 9, 21, 10, 10, 0, 0, time.UTC)})); len(cs) != 0 {
		for _, c := range cs {
			if _, ok := c.Props["mirror"]; ok {
				t.Error("AnalogClock's Canvas mirrors: a clock face never does")
			}
		}
	}
}
