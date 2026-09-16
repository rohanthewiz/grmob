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
// # Smooth
//
// Smooth draws each run of values as a monotone cubic (Fritsch and Carlson,
// "Monotone Piecewise Cubic Interpolation", 1980) instead of straight
// segments. Monotone is the property that matters for data: the curve never
// rises above the higher of two neighbouring values or dips below the lower,
// so a smoothed line invents no peak the data does not have, which a
// Catmull-Rom or plain cubic spline does at every sharp turn. It survives the
// stretch, too: CanvasStretch scales each axis by its own factor, and a curve
// monotone between its points stays monotone under any per-axis scale.
//
// # Stacked
//
// Stacked (an area chart's, usually) draws each series on top of the ones
// before it: series i's line is the running total of series 0..i, and its
// area fills from the previous total up to its own, so the top line is the
// whole and each band a part of it.
//
//	total ─────╮        series 2's band: between total₁ and total₂
//	      ░░░░░╰──╮
//	total₁ ────────╰─   series 1's band
//	      ▒▒▒▒▒▒▒▒▒▒▒
//	total₀ ───────────  series 0's band, down to zero
//
// A missing value counts as zero in a stack — a gap in one band would tear a
// hole through every band above it — and the axis spans the totals. The
// spoken summary still reads each series' own values, not the running totals,
// since those are what a listener asked about. Negative values stack like any
// other, downwards through the band below; a stack is for parts of a whole,
// which are not negative.
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
	// An unstacked area's tint fades from 30% under the series' extreme value
	// to 4% at the base line (see areaShape); a stacked band stays a flat 40%
	// so each band reads as its own colour. AreaChart is this with the field
	// set.
	Area bool

	// ZeroBased starts the value axis at zero even when every value is far
	// above it. Off by default for a line — its shape is the point, and a
	// temperature between 18 and 24 drawn against 0 is a flat line — and
	// always on for an area, whose filled height would otherwise misstate the
	// value.
	ZeroBased bool

	// Points marks each value with a dot.
	Points bool

	// Smooth draws each line as a monotone cubic curve through its points
	// rather than straight segments. See "Smooth" above.
	Smooth bool

	// Stacked draws each series on top of the running total of the ones
	// before it. See "Stacked" above. With Area it is a stacked area chart.
	Stacked bool

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
	// What is drawn: each series' own values, or its running total when
	// stacked. The summary reads c.Series, never these.
	drawn := c.Series
	if c.Stacked {
		drawn = stackSeries(c.Series, n)
	}
	lo, hi, _ := dataRange(drawn)
	// A stack is zero-based for the reason an area is: its bands' heights are
	// the values.
	scale := niceScale(lo, hi, chartMaxTicks(h), c.ZeroBased || c.Area || c.Stacked)

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
	for i, s := range drawn {
		color := seriesColor(palette, s.Color, i)
		colors[i], names[i] = color, s.Name
		// A stacked band's floor is the total below it; the first band's, and
		// every unstacked area's, is the base line.
		var floor []float64
		if c.Stacked && i > 0 {
			floor = drawn[i-1].Values
		}
		line, area, dots := seriesPaths(s.Values, floor, n, scale, scale.y(scale.base()), c.Smooth)
		if c.Area {
			// A stacked band is opaque enough to read as its own colour
			// rather than a blend of every band under it; overlapping areas
			// stay translucent so each one's line shows through the others.
			if c.Stacked {
				shapes = append(shapes, core.Shape{Path: area, Fill: withAlpha(color, "66")})
			} else {
				shapes = append(shapes, areaShape(area, color, s.Values, scale))
			}
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

// The two ends of an unstacked area's fade, as alpha bytes: 30% at the
// series' furthest point from the base line, 4% at the base line. Their mean
// over a typical area is close to the flat 20% ("33") the fill had before, so
// overlapping areas stay about as legible through each other, while the ink
// gathers under the line — where the eye reads the value — and thins toward
// the axis, where a flat tint only said "filled".
const (
	areaFadeTop  = "4D"
	areaFadeBase = "0A"
)

// areaShape is an unstacked series' area: a vertical fade in its own colour
// from its extreme value down to the base line.
//
// The gradient runs from the drawn point furthest from the base (the peak of
// a positive series, the trough of a negative one) to the base, in viewBox
// units. That puts full strength exactly under the extreme, whatever the
// scale's headroom above it, and keeps the direction right for values below
// zero: the fade always runs toward the axis. A series crossing zero fades
// toward the axis from its larger side, and its smaller side takes the faint
// end colour (the gradient pads past its ends).
//
// Geometry is in viewBox units under CanvasStretch; every target resolves a
// userSpaceOnUse gradient in that space before the stretch, so the fade
// stretches with the drawing and still ends on the base line.
//
// It falls back to the flat fill the chart has always drawn when a fade
// cannot be made: a colour that is not "#rrggbb" / "#rgb" (withAlpha leaves
// such a colour unchanged, and a gradient between two copies of it is no
// fade), or a series with no finite value off the base line (no length to
// fade along).
func areaShape(area *core.Path, color string, values []float64, scale valueScale) core.Shape {
	top, faint := withAlpha(color, areaFadeTop), withAlpha(color, areaFadeBase)
	flat := core.Shape{Path: area, Fill: withAlpha(color, "33")}
	if top == color {
		return flat
	}
	base := scale.y(scale.base())
	extreme, far := base, 0.0
	for _, v := range values {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			continue
		}
		if y := scale.y(v); math.Abs(y-base) > far {
			extreme, far = y, math.Abs(y-base)
		}
	}
	if far == 0 {
		return flat
	}
	return core.Shape{Path: area, FillGradient: core.LinearGradientFill(0, extreme, 0, base,
		core.Stop(0, top), core.Stop(1, faint))}
}

// pointX is the viewBox x of point i of n: evenly spaced, edge to edge, and
// centred when there is only one.
func pointX(i, n int) float64 {
	if n <= 1 {
		return chartView / 2
	}
	return float64(i) * chartView / float64(n-1)
}

// stackSeries returns each series with its values replaced by the running
// total of it and every series before it, over n points. A missing value adds
// nothing (see "Stacked"), so every total is a number.
func stackSeries(series []ChartSeries, n int) []ChartSeries {
	out := make([]ChartSeries, len(series))
	total := make([]float64, n)
	for i, s := range series {
		for k := range n {
			if k < len(s.Values) && !math.IsNaN(s.Values[k]) && !math.IsInf(s.Values[k], 0) {
				total[k] += s.Values[k]
			}
		}
		out[i] = ChartSeries{Name: s.Name, Color: s.Color, Values: append([]float64(nil), total...)}
	}
	return out
}

// linePaths builds a series' straight line, its area down to the viewBox y
// base, and the dots for values standing alone between gaps. It is
// seriesPaths with no floor and no smoothing.
func linePaths(values []float64, n int, s valueScale, base float64) (line, area, dots *core.Path) {
	return seriesPaths(values, nil, n, s, base, false)
}

// seriesPaths builds a series' line, its area, and the dots for values
// standing alone between gaps. Each run of consecutive values is one subpath
// of the line and one closed subpath of the area.
//
// The area runs along the line and back along its floor: the viewBox y base
// when floor is nil, otherwise floor's values over the same points (a stacked
// band's lower edge), traversed backwards. With smooth, both edges are
// monotone cubics, and the floor's is the same curve its own line was drawn
// with, reversed, so a band meets the band below it without a seam.
//
//	→ line:  p₀ ──c── p₁ ──c── p₂
//	                           │
//	← floor: f₀ ──c── f₁ ──c── f₂    (the curve through f, walked backwards)
func seriesPaths(values, floor []float64, n int, s valueScale, base float64, smooth bool) (line, area, dots *core.Path) {
	line, area, dots = core.NewPath(), core.NewPath(), core.NewPath()
	runStart := -1
	flush := func(end int) { // end is exclusive
		if runStart < 0 {
			return
		}
		if end-runStart == 1 {
			x, y := pointX(runStart, n), s.y(values[runStart])
			dots.MoveTo(x, y).LineTo(x, y)
			runStart = -1
			return
		}
		xs := make([]float64, 0, end-runStart)
		ys := make([]float64, 0, end-runStart)
		for i := runStart; i < end; i++ {
			xs = append(xs, pointX(i, n))
			ys = append(ys, s.y(values[i]))
		}
		line.MoveTo(xs[0], ys[0])
		traceRun(line, xs, ys, smooth, false)

		last := len(xs) - 1
		area.MoveTo(xs[0], ys[0])
		traceRun(area, xs, ys, smooth, false)
		if floor == nil {
			area.LineTo(xs[last], base).LineTo(xs[0], base)
		} else {
			fys := make([]float64, len(xs))
			for k := range xs {
				fys[k] = s.y(floor[runStart+k])
			}
			area.LineTo(xs[last], fys[last])
			traceRun(area, xs, fys, smooth, true)
		}
		area.Close()
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

// traceRun continues p through the points (xs[k], ys[k]) from the first to
// the last, or from the last to the first when reverse is set, assuming the
// current point is already the starting one. Straight segments, or the
// monotone cubic of "Smooth" when smooth is set.
func traceRun(p *core.Path, xs, ys []float64, smooth, reverse bool) {
	last := len(xs) - 1
	if !smooth || last < 2 {
		// Two points have one segment, and its monotone cubic is the straight
		// line between them, so it is drawn as one.
		for k := 1; k <= last; k++ {
			i := k
			if reverse {
				i = last - k
			}
			p.LineTo(xs[i], ys[i])
		}
		return
	}
	m := monotoneTangents(xs, ys)
	for k := 0; k < last; k++ {
		i, j := k, k+1 // the segment from point i to point j
		if reverse {
			i, j = last-k, last-k-1
		}
		// Cubic Hermite to Bézier: each control point is a third of the way
		// along its end's tangent. Reversing swaps the ends, and with them
		// which control comes first, which is why i and j carry the whole
		// segment rather than k.
		h := (xs[j] - xs[i]) / 3
		p.CubicTo(xs[i]+h, ys[i]+m[i]*h, xs[j]-h, ys[j]-m[j]*h, xs[j], ys[j])
	}
}

// monotoneTangents returns the slope at each point of a Fritsch–Carlson
// monotone cubic through (xs, ys), xs strictly increasing and len ≥ 3.
//
//  1. d[k], the secant slope of segment k.
//  2. m[k] at an interior point is the mean of its two secants, or 0 where
//     they differ in sign (a local peak or trough: a flat tangent is what
//     keeps the curve from overshooting it). The ends take their one secant.
//  3. On each segment, α = m[k]/d[k] and β = m[k+1]/d[k]. When α² + β² > 9
//     the pair is scaled onto that circle, which is Fritsch and Carlson's
//     sufficient condition for the segment to stay monotone. A flat segment
//     (d = 0) flattens both its tangents.
func monotoneTangents(xs, ys []float64) []float64 {
	n := len(xs)
	d := make([]float64, n-1)
	for k := range d {
		d[k] = (ys[k+1] - ys[k]) / (xs[k+1] - xs[k])
	}
	m := make([]float64, n)
	m[0], m[n-1] = d[0], d[n-2]
	for k := 1; k < n-1; k++ {
		if d[k-1]*d[k] <= 0 {
			m[k] = 0
		} else {
			m[k] = (d[k-1] + d[k]) / 2
		}
	}
	for k := range d {
		if d[k] == 0 {
			m[k], m[k+1] = 0, 0
			continue
		}
		a, b := m[k]/d[k], m[k+1]/d[k]
		if r := a*a + b*b; r > 9 {
			t := 3 / math.Sqrt(r)
			m[k], m[k+1] = t*a*d[k], t*b*d[k]
		}
	}
	return m
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
