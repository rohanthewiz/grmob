package comps

import (
	"math"

	"github.com/rohanthewiz/grmob/core"
)

// Sparkline is a word-sized line chart: no axes, no labels, just the shape of
// a series, for a table cell, a list row or beside a StatTile's number.
//
//	core.Row(
//	    core.Text("Visits"),
//	    comps.Sparkline{Values: visits, Subject: "Visits this week", Style: []core.StyleProp{core.Width("80px")}},
//	)
//
// It fills its parent's width unless Style gives one, and is Height px tall.
// The domain is the data's own low to high, not rounded to ticks: with no axis
// to read against, the full height is spent on the shape.
//
// NaN values break the line, as on LineChart. ShowLast marks the latest value
// with a dot, which is where a reader's eye goes first; the dot is a round-
// capped zero-length stroke so it stays round when the line is stretched (see
// LineChart's "Points and dots").
type Sparkline struct {
	Values []float64

	// Subject names the series for the spoken summary ("Visits this week").
	Subject string

	// Height is in px; 0 means 32.
	Height float64

	// StrokeWidth is in px; 0 means 1.5.
	StrokeWidth float64

	// Color is the line; empty uses the theme's Primary.
	Color string

	// Area fills under the line with a tint of Color, down to the bottom edge.
	// Not down to zero, as AreaChart's does: a sparkline's domain rarely
	// includes zero, and with no axis the tint is texture, not a measurement.
	Area bool

	// ShowLast draws a dot on the last value.
	ShowLast bool

	// Format writes a value in the spoken summary.
	Format func(float64) string

	// Style is applied last, to the canvas.
	Style []core.StyleProp

	// AccessibilityLabel replaces the generated summary.
	AccessibilityLabel string
}

func (s Sparkline) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	h := s.Height
	if h <= 0 {
		h = 32
	}
	sw := s.StrokeWidth
	if sw <= 0 {
		sw = 1.5
	}
	color := s.Color
	if color == "" {
		color = t.Colors.Primary
	}

	n := len(s.Values)
	lo, hi, _ := dataRange([]ChartSeries{{Values: s.Values}})
	scale := sparkScale(lo, hi, h, sw)
	line, area, dots := linePaths(s.Values, n, scale, chartView)

	// Four shapes, always, so toggling Area or ShowLast patches a slot rather
	// than shifting the ones after it.
	areaShape := core.Shape{}
	if s.Area {
		areaShape = core.Shape{Path: area, Fill: withAlpha(color, "33")}
	}
	last := core.NewPath()
	if s.ShowLast {
		for i := n - 1; i >= 0; i-- {
			if v := s.Values[i]; !math.IsNaN(v) && !math.IsInf(v, 0) {
				x, y := pointX(i, n), scale.y(v)
				last.MoveTo(x, y).LineTo(x, y)
				break
			}
		}
	}
	shapes := []core.Shape{
		areaShape,
		{Path: line, Stroke: color, StrokeWidth: sw, Cap: core.CapRound, Join: core.JoinRound},
		{Path: dots, Stroke: color, StrokeWidth: sw * 2, Cap: core.CapRound},
		{Path: last, Stroke: color, StrokeWidth: sw * 3, Cap: core.CapRound},
	}

	props := make([]core.PropsAndChildren, 0, 4+len(s.Style))
	props = append(props, core.CanvasStretch, core.Height(px(h)), core.AccessibilityLabel(s.label()))
	for _, sp := range s.Style {
		props = append(props, sp)
	}
	return core.Canvas(chartView, chartView, shapes, props...).Render(ctx)
}

// sparkScale is the data's own range, widened so the line's stroke and the
// last-value dot (three strokes wide, so half of it is 1.5 strokes) sit inside
// the box at the top and bottom. The box's height in px is known, so the
// margin can be written as a fraction of it: m px of an h px box is m/h.
//
//	value span (hi − lo) occupies 1 − 2·margin of the box
//	so the whole box spans (hi − lo) / (1 − 2·margin) in value units
func sparkScale(lo, hi, h, sw float64) valueScale {
	if math.IsInf(lo, 0) || math.IsInf(hi, 0) || lo > hi {
		lo, hi = 0, 1
	}
	if lo == hi {
		// A flat series draws along the middle.
		lo, hi = lo-1, hi+1
	}
	margin := math.Min(0.4, sw*1.5/h)
	pad := (hi - lo) / (1 - 2*margin) * margin
	return valueScale{lo: lo - pad, hi: hi + pad, step: 1}
}

// label is "Visits this week: 7 points, from 12 to 45, low 12, high 45." — the
// same sentence LineChart speaks for one unnamed series.
func (s Sparkline) label() string {
	if s.AccessibilityLabel != "" {
		return s.AccessibilityLabel
	}
	return LineChart{Subject: s.Subject, Series: []ChartSeries{{Values: s.Values}}, Format: s.Format}.label()
}
