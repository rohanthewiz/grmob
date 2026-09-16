package comps

import (
	"fmt"
	"math"
	"strconv"

	"github.com/rohanthewiz/grmob/core"
)

// LineChart draws one or more series as lines over a value axis, with the
// points evenly spaced from the left edge to the right.
//
//	comps.LineChart{
//	    Subject: "Weekly visits",
//	    Labels:  []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"},
//	    Series:  []comps.ChartSeries{{Name: "Visits", Values: visits}},
//	}
//
// The layout — y labels, gridlines, x labels, legend — is described at the top
// of chart.go. The plot is a core.Canvas with CanvasStretch, so it fills the
// width it is given and its lines stay StrokeWidth px thick at any width.
//
// # Missing values
//
// A NaN value leaves a gap: the line stops at the point before it and starts
// again at the point after. A value on its own between two gaps is drawn as a
// dot, since a line of one point has no length.
//
// # Points and dots
//
// Points (and the lone-value dot above) are zero-length strokes with round
// caps, which every target draws as a circle StrokeWidth wide. A Circle path
// would not do: under CanvasStretch the viewBox is scaled differently on each
// axis, so a circle drawn in it comes out as an ellipse, while a stroke's
// width is never scaled.
type LineChart struct {
	// Series are the lines, drawn in order (a later series paints over an
	// earlier one). Every series is read against the same x positions:
	// value i of each is at point i.
	Series []ChartSeries

	// Labels name the x positions ("Mon", "Jan", "09:00"). Up to five are
	// drawn, evenly spaced and always including the first; all of them feed
	// the spoken summary. Nil draws no x axis.
	Labels []string

	// Subject says what the chart measures ("Weekly visits"). It leads the
	// spoken summary and is not drawn: a chart usually sits under a heading
	// that already says it.
	Subject string

	// Height is the plot's height in px, excluding labels; 0 means 160.
	Height float64

	// StrokeWidth is the line's width in px; 0 means 2.
	StrokeWidth float64

	// Area fills under each line, down to zero (or the nearest edge of the
	// axis when zero is off it), with a translucent tint of the line's colour.
	// AreaChart is this with the field set.
	Area bool

	// ZeroBased starts the value axis at zero even when every value is far
	// above it. Off by default for a line — its shape is the point, and a
	// temperature between 18 and 24 drawn against 0 is a flat line — and
	// always on for an area, whose filled height would otherwise misstate the
	// value.
	ZeroBased bool

	// Points marks each value with a dot.
	Points bool

	// Colors overrides the theme palette for series without their own Color.
	Colors []string

	// Format writes a y tick label and a value in the spoken summary. Nil uses
	// the tick's own precision on the axis and up to two decimals when spoken.
	Format func(float64) string

	// Style is applied last, to the outer column.
	Style []core.StyleProp

	// AccessibilityLabel replaces the generated summary.
	AccessibilityLabel string
}

// AreaChart is a LineChart with Area set: each series filled down to zero.
// Several series overlap translucently rather than stacking.
type AreaChart LineChart

func (a AreaChart) Render(ctx *core.Context) *core.Node {
	l := LineChart(a)
	l.Area = true
	return l.Render(ctx)
}

func (c LineChart) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	h := c.Height
	if h <= 0 {
		h = chartHeight
	}
	sw := c.StrokeWidth
	if sw <= 0 {
		sw = 2
	}

	n := 0
	for _, s := range c.Series {
		n = max(n, len(s.Values))
	}
	lo, hi, _ := dataRange(c.Series)
	scale := niceScale(lo, hi, chartMaxTicks(h), c.ZeroBased || c.Area)

	palette := c.Colors
	if len(palette) == 0 {
		palette = chartPalette(t)
	}

	shapes := gridShapes(t, scale)
	colors := make([]string, len(c.Series))
	names := make([]string, len(c.Series))
	// All fills go under all lines, so a later series' area never hides an
	// earlier series' line.
	var lines []core.Shape
	for i, s := range c.Series {
		color := seriesColor(palette, s.Color, i)
		colors[i], names[i] = color, s.Name
		line, area, dots := linePaths(s.Values, n, scale, scale.y(scale.base()))
		if c.Area {
			shapes = append(shapes, core.Shape{Path: area, Fill: withAlpha(color, "33")})
		}
		lines = append(lines, core.Shape{
			Path: line, Stroke: color, StrokeWidth: sw,
			Cap: core.CapRound, Join: core.JoinRound,
		})
		if c.Points {
			dots = allPoints(s.Values, n, scale)
		}
		// Always emitted, possibly empty, so toggling Points or a gap
		// appearing does not shift the slots of the shapes after it.
		lines = append(lines, core.Shape{
			Path: dots, Stroke: color, StrokeWidth: sw * 2.5, Cap: core.CapRound,
		})
	}
	shapes = append(shapes, lines...)

	canvas := core.Canvas(chartView, chartView, shapes, core.CanvasStretch, core.Height(px(h)))

	var legendView core.View
	if len(c.Series) > 1 {
		legendView = legend(t, names, colors)
	}

	return cartesianFrame(ctx, scale, h, c.Format, canvas, pointLabels(t, c.Labels, n),
		legendView, c.label(), c.Style)
}

// pointX is the viewBox x of point i of n: evenly spaced, edge to edge, and
// centred when there is only one.
func pointX(i, n int) float64 {
	if n <= 1 {
		return chartView / 2
	}
	return float64(i) * chartView / float64(n-1)
}

// linePaths builds a series' line, its area, and the dots for values standing
// alone between gaps. Each run of consecutive values is one subpath of the
// line and one closed subpath of the area, closed down to the viewBox y base.
func linePaths(values []float64, n int, s valueScale, base float64) (line, area, dots *core.Path) {
	line, area, dots = core.NewPath(), core.NewPath(), core.NewPath()
	runStart := -1
	flush := func(end int) { // end is exclusive
		if runStart < 0 {
			return
		}
		if end-runStart == 1 {
			x, y := pointX(runStart, n), s.y(values[runStart])
			dots.MoveTo(x, y).LineTo(x, y)
		} else {
			area.MoveTo(pointX(runStart, n), base)
			for i := runStart; i < end; i++ {
				x, y := pointX(i, n), s.y(values[i])
				if i == runStart {
					line.MoveTo(x, y)
				} else {
					line.LineTo(x, y)
				}
				area.LineTo(x, y)
			}
			area.LineTo(pointX(end-1, n), base).Close()
		}
		runStart = -1
	}
	for i, v := range values {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			flush(i)
			continue
		}
		if runStart < 0 {
			runStart = i
		}
	}
	flush(len(values))
	return line, area, dots
}

// allPoints is a dot on every value.
func allPoints(values []float64, n int, s valueScale) *core.Path {
	p := core.NewPath()
	for i, v := range values {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			continue
		}
		x, y := pointX(i, n), s.y(v)
		p.MoveTo(x, y).LineTo(x, y)
	}
	return p
}

// label is the spoken summary: for each series, how many points, where it
// starts and ends, and its low and high, each with its x label when there is
// one. Every value is not read out — past a handful a listener cannot hold
// them — but the shape of a trend is in those four.
func (c LineChart) label() string {
	if c.AccessibilityLabel != "" {
		return c.AccessibilityLabel
	}
	format := c.Format
	if format == nil {
		format = formatValue
	}
	at := func(i int) string {
		if i < len(c.Labels) && c.Labels[i] != "" {
			return " in " + c.Labels[i]
		}
		return ""
	}
	var parts []string
	for _, s := range c.Series {
		first, last, low, high := -1, -1, -1, -1
		count := 0
		for i, v := range s.Values {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				continue
			}
			count++
			if first < 0 {
				first = i
			}
			last = i
			if low < 0 || v < s.Values[low] {
				low = i
			}
			if high < 0 || v > s.Values[high] {
				high = i
			}
		}
		// A lone series' name usually repeats the Subject ("Visits: Visits,
		// ..."), so it is spoken only when there is no Subject or when it
		// tells the series apart.
		name := s.Name
		if len(c.Series) == 1 && c.Subject != "" {
			name = ""
		}
		if name != "" {
			name += ", "
		}
		if count == 0 {
			parts = append(parts, name+"no data")
			continue
		}
		if count == 1 {
			parts = append(parts, fmt.Sprintf("%s1 point, %s%s", name, format(s.Values[first]), at(first)))
			continue
		}
		parts = append(parts, fmt.Sprintf("%s%s points, from %s%s to %s%s, low %s%s, high %s%s",
			name, strconv.Itoa(count),
			format(s.Values[first]), at(first), format(s.Values[last]), at(last),
			format(s.Values[low]), at(low), format(s.Values[high]), at(high)))
	}
	if len(parts) == 0 {
		parts = append(parts, "no data")
	}
	return summaryPrefix(c.Subject) + joinSentences(parts)
}
