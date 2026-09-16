package comps

import (
	"fmt"
	"math"
	"strconv"

	"github.com/rohanthewiz/grmob/core"
)

// ChartPoint is one (x, y) observation in a ScatterChart. A NaN or infinite
// coordinate leaves the point out.
type ChartPoint struct {
	X, Y float64
}

// ScatterSeries is one colour of points in a ScatterChart.
type ScatterSeries struct {
	// Name labels the series in the legend (drawn when there is more than one
	// series) and in the spoken summary.
	Name string

	Points []ChartPoint

	// Color overrides the series' palette colour.
	Color string
}

// ScatterChart plots points against two value axes.
//
//	comps.ScatterChart{
//	    Subject: "Height and weight",
//	    Series:  []comps.ScatterSeries{{Name: "Players", Points: points}},
//	    XFormat: func(v float64) string { return fmt.Sprintf("%.0f cm", v) },
//	}
//
// # Two value axes, one layout
//
// The y axis is LineChart's: nice ticks down the left, gridlines behind the
// data. The x axis is nice ticks too, and they run edge to edge exactly as a
// line chart's points do, so the tick labels under the plot are placed by the
// same point arithmetic (pointLabels, in chart.go): the first label aligned to
// the start edge, the last to the end, the rest centred on their ticks. No
// width is measured.
//
//	20 ┤ ·    ·         ·
//	10 ┤    ·     ·  ·
//	 0 ┼──────────────────
//	   0     50     100
//
// Neither axis is pulled to zero unless asked (XZeroBased, ZeroBased): a
// scatter's question is how two quantities move together, and an origin far
// from the data squeezes the cloud into a corner.
//
// # Dots
//
// Each point is a zero-length stroke with round caps, LineChart's dot, for the
// same reason: under CanvasStretch a circle path would come out as an ellipse,
// and a stroke's width is never scaled. Every point of a series is one
// subpath of one shape, so a thousand points are one canvas node.
type ScatterChart struct {
	// Series are the point sets, drawn in order (a later series paints over an
	// earlier one).
	Series []ScatterSeries

	// Subject says what the chart shows. It leads the spoken summary and is
	// not drawn.
	Subject string

	// Height is the plot's height in px, excluding labels; 0 means 160.
	Height float64

	// DotSize is each point's diameter in px; 0 means 6.
	DotSize float64

	// ZeroBased and XZeroBased include zero in the y and x axes.
	ZeroBased, XZeroBased bool

	// Colors overrides the theme palette for series without their own Color.
	Colors []string

	// Format writes a y tick label and a spoken y value; XFormat the same for
	// x. Nil uses the tick's own precision on an axis and up to two decimals
	// when spoken.
	Format, XFormat func(float64) string

	// Style is applied last, to the outer column.
	Style []core.StyleProp

	// AccessibilityLabel replaces the generated summary.
	AccessibilityLabel string
}

func (c ScatterChart) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	h := c.Height
	if h <= 0 {
		h = chartHeight
	}
	dot := c.DotSize
	if dot <= 0 {
		dot = 6
	}

	xlo, xhi, ylo, yhi := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
	for _, s := range c.Series {
		for _, p := range s.Points {
			if !finitePoint(p) {
				continue
			}
			xlo, xhi = math.Min(xlo, p.X), math.Max(xhi, p.X)
			ylo, yhi = math.Min(ylo, p.Y), math.Max(yhi, p.Y)
		}
	}
	// niceScale widens an empty range (Inf, −Inf) to 0..1 itself.
	yScale := niceScale(ylo, yhi, chartMaxTicks(h), c.ZeroBased)
	// Five, for the reason BarChart.Horizontal gives: every tick is labelled.
	xScale := niceScale(xlo, xhi, chartMaxXLabels, c.XZeroBased)

	palette := c.Colors
	if len(palette) == 0 {
		palette = chartPalette(t)
	}

	shapes := append(gridShapes(t, yScale), verticalGridShapes(t, xScale)...)
	colors := make([]string, len(c.Series))
	names := make([]string, len(c.Series))
	for i, s := range c.Series {
		color := seriesColor(palette, s.Color, i)
		colors[i], names[i] = color, s.Name
		p := core.NewPath()
		for _, pt := range s.Points {
			if !finitePoint(pt) {
				continue
			}
			x, y := chartView-xScale.y(pt.X), yScale.y(pt.Y)
			p.MoveTo(x, y).LineTo(x, y)
		}
		shapes = append(shapes, core.Shape{Path: p, Stroke: color, StrokeWidth: dot, Cap: core.CapRound})
	}

	canvas := core.Canvas(chartView, chartView, shapes, core.CanvasStretch, core.Height(px(h)))

	ticks := xScale.ticks()
	extent := math.Max(math.Abs(xScale.lo), math.Abs(xScale.hi))
	tickText := make([]string, len(ticks))
	for i, v := range ticks {
		tickText[i] = formatTick(v, xScale.step, extent)
		if c.XFormat != nil {
			tickText[i] = c.XFormat(v)
		}
	}

	var legendView core.View
	if len(c.Series) > 1 {
		legendView = legend(t, names, colors)
	}
	return cartesianFrame(ctx, yScale, h, c.Format, canvas, pointLabels(t, tickText, len(ticks)),
		legendView, c.label(), c.Style)
}

func finitePoint(p ChartPoint) bool {
	return !math.IsNaN(p.X) && !math.IsNaN(p.Y) && !math.IsInf(p.X, 0) && !math.IsInf(p.Y, 0)
}

// label is the spoken summary: per series, how many points and the range each
// coordinate covers. Individual points are not read — a cloud is its extent
// and its count, and a listener cannot hold forty pairs.
func (c ScatterChart) label() string {
	if c.AccessibilityLabel != "" {
		return c.AccessibilityLabel
	}
	fx, fy := c.XFormat, c.Format
	if fx == nil {
		fx = formatValue
	}
	if fy == nil {
		fy = formatValue
	}
	var parts []string
	for _, s := range c.Series {
		name := s.Name
		if name != "" {
			name += ", "
		}
		count := 0
		xlo, xhi, ylo, yhi := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
		for _, p := range s.Points {
			if !finitePoint(p) {
				continue
			}
			count++
			xlo, xhi = math.Min(xlo, p.X), math.Max(xhi, p.X)
			ylo, yhi = math.Min(ylo, p.Y), math.Max(yhi, p.Y)
		}
		switch count {
		case 0:
			parts = append(parts, name+"no data")
		case 1:
			parts = append(parts, fmt.Sprintf("%s1 point at %s, %s", name, fx(xlo), fy(ylo)))
		default:
			parts = append(parts, fmt.Sprintf("%s%s points, x from %s to %s, y from %s to %s",
				name, strconv.Itoa(count), fx(xlo), fx(xhi), fy(ylo), fy(yhi)))
		}
	}
	if len(parts) == 0 {
		parts = append(parts, "no data")
	}
	return summaryPrefix(c.Subject) + joinSentences(parts)
}
