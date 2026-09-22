package comps

import (
	"fmt"
	"math"
	"strings"

	"github.com/rohanthewiz/grmob/core"
)

// ConcernRadarChartTooFewAxes is raised, in debug builds only, when a
// RadarChart has fewer than three Axes. Two axes span a line and one a point,
// so there is no polygon to draw and the chart shows its empty rim.
const ConcernRadarChartTooFewAxes = "radar-chart-too-few-axes"

// RadarChart draws several measures of one subject as a polygon on spokes
// around a centre, one spoke per measure, so that a profile reads as a shape:
// a player's speed, power and stamina; a product scored on five criteria.
//
//	comps.RadarChart{
//	    Subject: "Player",
//	    Axes:    []string{"Speed", "Power", "Stamina", "Skill", "Vision"},
//	    Series:  []comps.ChartSeries{{Name: "Ade", Values: []float64{8, 6, 7, 9, 5}}},
//	    Max:     10,
//	    Filled:  true,
//	}
//
// # Geometry
//
// Axis k of n points at −90° + k·360°/n: the first straight up, the rest
// clockwise, the order a clock face and DonutChart both read in. Value v sits
// at v/Max of the way out. The grid is Rings concentric polygons with the
// same corners and a spoke to each, so a gridline is a straight run between
// two spokes and a value on a spoke reads against it exactly (a circular grid
// would only agree with the polygon on the spokes themselves).
//
// # The labels, and why they are not in the drawing
//
// core.Canvas has no text, so every label is a Text laid over the drawing in
// a core.ZStack. A ZStack gives nine named places (core.StackAlign), which is
// Compass's four letters and no more; five or seven labels round a rim need
// arbitrary ones. They get them from core.Translate. Each label is a box the
// stack centres, as it centres any layer that says nothing, moved by px
// offsets from the same trigonometry as the drawing:
//
//	            Speed                  box of LabelWidth × a label line,
//	        ┌─────────┐                centred, then translated so that the
//	Vision ╱     │     ╲ Power         edge nearest the rim touches the point
//	      │      ┼      │              radarLabelGap px outside the spoke's
//	      ╲     ╱ ╲     ╱              end: start-aligned on the right,
//	  Skill ───       ─── Stamina      end-aligned on the left, centred at
//	                                   the top and bottom
//
// Translate's px are exact because Size is: the drawing is a Size px square
// under CanvasFit, so a viewBox unit is Size/100 px on both axes and Go knows
// where every spoke ends without measuring anything. The stack is pinned to
// the drawing plus a label's width either side and a label's height above
// and below, so no label leaves it.
//
// The scale's values are written the same way, just beside the upward spoke
// at each ring, because a radar with unlabelled rings shows shape and no
// magnitude.
//
// # Right to left
//
// Translate's x is leading-relative, so in a right-to-left layout the labels
// mirror across the vertical spoke while the drawing, like every Canvas, does
// not. Axis k's label then sits on axis n−k's spoke. That is the same
// disagreement LineChart's label row has with its line under RTL, and it
// wants the same answer (a Canvas that mirrors, or a direction Go can read),
// which is a renderer's to give.
//
// # Values outside the scale
//
// A value above Max is drawn at Max and a negative one at the centre: the rim
// is the chart's edge and a polygon crossing it would run under the labels.
// The spoken summary reads the value as given. NaN is a missing value; it
// draws at the centre and is left out of the summary.
//
// # Accessibility
//
// One element. Up to radarSummaryLimit axes are read in full, per series
// ("Ade, Speed 8, Power 6, …"); past that, each series' lowest and highest.
//
// # Theme roles read
//
//	Series   Colors.ChartColors(), a fill of the same hue at radarFillAlpha
//	Grid     Colors.BorderColor()
//	Labels   Colors.TextSecondary
type RadarChart struct {
	// Axes name the measures, clockwise from the top. Their count is the
	// number of spokes; fewer than three reports ConcernRadarChartTooFewAxes.
	Axes []string

	// Series are the profiles. Value k of each belongs to axis k; a series
	// shorter than Axes is missing the rest.
	Series []ChartSeries

	// Subject says what is being profiled. It leads the spoken summary and is
	// not drawn.
	Subject string

	// Max is the value at the rim; 0 means the data's highest, widened to a
	// round number. Every axis shares it, so measures on different scales
	// must be normalised by the caller.
	Max float64

	// Rings is the number of grid polygons, the rim included; 0 means 4.
	Rings int

	// Size is the drawing's diameter in px; 0 means 180. The widget is wider
	// and taller than that by its labels.
	Size float64

	// Filled tints each polygon with its own colour. Off, the polygons are
	// outlines, which is clearer once three or more overlap.
	Filled bool

	// LabelWidth is the width of an axis label's box, in px; 0 means 56. A
	// longer label is cut with an ellipsis.
	LabelWidth float64

	// HideScale drops the ring values beside the upward spoke.
	HideScale bool

	// Colors overrides the theme palette for series without their own Color.
	Colors []string

	// Format writes a ring value and a spoken value.
	Format func(float64) string

	// Style is applied last, to the outer column.
	Style []core.StyleProp

	// AccessibilityLabel replaces the generated summary.
	AccessibilityLabel string
}

const (
	// radarRim is the rim's radius in viewBox units. Under the full 50 so
	// the rim's stroke and a marker on it are not cut by the canvas's edge.
	radarRim = 47.0

	// radarLabelGap is the space between a spoke's end and its label, in px.
	radarLabelGap = 6.0

	// radarFillAlpha is a filled polygon's opacity: low enough that three
	// overlapping profiles still show the grid through all of them.
	radarFillAlpha = "33"

	// radarChipAlpha is the opacity of the chip behind a ring value: enough
	// to break a 2 px line under the digits, short of a solid patch that
	// would read as a hole in a filled polygon.
	radarChipAlpha = "d9"

	// radarSummaryLimit is the most axes read out in full per series.
	radarSummaryLimit = 8
)

func (c RadarChart) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	size := c.Size
	if size <= 0 {
		size = 180
	}
	rings := c.Rings
	if rings <= 0 {
		rings = 4
	}
	labelWidth := c.LabelWidth
	if labelWidth <= 0 {
		labelWidth = 56
	}
	n := len(c.Axes)
	if core.IsDebugMode() && n < 3 {
		core.ReportConcern(ConcernRadarChartTooFewAxes, fmt.Sprintf(
			"RadarChart has %d axes and needs at least 3 to enclose an area, so no polygon is drawn", n))
	}

	top := c.Max
	if top <= 0 {
		_, hi, ok := dataRange(c.Series)
		if !ok || hi <= 0 {
			hi = 1
		}
		// The rings divide the scale evenly, so the rim is widened to a
		// value that Rings divides into round steps.
		top = radarStep(hi/float64(rings)) * float64(rings)
	}

	palette := c.Colors
	if len(palette) == 0 {
		palette = chartPalette(t)
	}
	colors := make([]string, len(c.Series))
	names := make([]string, len(c.Series))
	for j, s := range c.Series {
		colors[j], names[j] = seriesColor(palette, s.Color, j), s.Name
	}

	const mid = chartView / 2
	// at is the viewBox point a fraction f of the way out along axis k.
	at := func(k int, f float64) (x, y float64) {
		sin, cos := math.Sincos(radarAngle(k, n))
		return mid + radarRim*f*cos, mid + radarRim*f*sin
	}

	grid := core.NewPath()
	if n >= 3 {
		for r := 1; r <= rings; r++ {
			f := float64(r) / float64(rings)
			for k := range n {
				x, y := at(k, f)
				if k == 0 {
					grid.MoveTo(x, y)
				} else {
					grid.LineTo(x, y)
				}
			}
			grid.Close()
		}
		for k := range n {
			x, y := at(k, 1)
			grid.MoveTo(mid, mid).LineTo(x, y)
		}
	} else {
		// Nothing to enclose: the bare rim, so the widget still says where
		// its data would go (DonutChart's empty track, for the same reason).
		grid = core.Circle(mid, mid, radarRim)
	}
	shapes := make([]core.Shape, 0, 1+2*len(c.Series))
	shapes = append(shapes, core.Shape{Path: grid, Stroke: t.Colors.BorderColor(), StrokeWidth: 1})

	// Two shapes per series, always, so a series keeps its child slots while
	// its values change: the polygon, then its markers as one path of dots.
	for j, s := range c.Series {
		if n < 3 {
			shapes = append(shapes, core.Shape{}, core.Shape{})
			continue
		}
		outline, dots := core.NewPath(), core.NewPath()
		for k := range n {
			f := 0.0
			if k < len(s.Values) {
				f = radarFraction(s.Values[k], top)
			}
			x, y := at(k, f)
			if k == 0 {
				outline.MoveTo(x, y)
			} else {
				outline.LineTo(x, y)
			}
			// CanvasFit scales both axes alike, so a circle stays round.
			// Each is its own closed subpath of the one markers path. The
			// MoveTo is what makes it so: a closed path still has a current
			// point, and Arc alone would join the last dot to this one.
			dots.MoveTo(x+1.6, y).Arc(x, y, 1.6, 0, 360).Close()
		}
		outline.Close()
		poly := core.Shape{Path: outline, Stroke: colors[j], StrokeWidth: 2, Join: core.JoinRound}
		if c.Filled {
			poly.Fill = withAlpha(colors[j], radarFillAlpha)
		}
		shapes = append(shapes, poly, core.Shape{Path: dots, Fill: colors[j]})
	}

	dim := px(size)
	stackW := size + 2*(labelWidth+radarLabelGap)
	stackH := size + 2*(chartLabelLine+radarLabelGap)
	layers := []core.PropsAndChildren{
		core.Padding(0),
		core.Width(px(stackW)),
		core.Height(px(stackH)),
		core.AccessibilityHidden(),
		core.Canvas(chartView, chartView, shapes, core.CanvasMirrorsRTL, core.Width(dim), core.Height(dim)),
	}
	// A viewBox unit in px; see "The labels".
	unit := size / chartView
	if n >= 3 {
		for k, name := range c.Axes {
			layers = append(layers, radarAxisLabel(t, name, radarAngle(k, n), radarRim*unit, labelWidth))
		}
		if !c.HideScale {
			format := c.Format
			if format == nil {
				format = func(v float64) string { return formatTick(v, top/float64(rings), top) }
			}
			// The rim's value is left to the first axis's label, which sits
			// where it would go.
			for r := 1; r < rings; r++ {
				f := float64(r) / float64(rings)
				layers = append(layers, radarScaleLabel(t, format(top*f), radarRim*unit*f))
			}
		}
	}

	items := make([]core.PropsAndChildren, 0, 8+len(c.Style))
	items = append(items,
		core.Padding(0),
		core.Gap(float64(t.Spacing.SM)),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.AccessibilityRole(core.RoleImg),
		core.AccessibilityLabel(c.label()),
	)
	items = append(items, asProps(c.Style)...)
	items = append(items, core.ZStack(layers...))
	if len(c.Series) > 1 {
		items = append(items, legend(t, names, colors))
	}
	return core.Column(items...).Render(ctx)
}

// radarStep is the smallest round ring spacing at or above x: 1, 2, 2.5, 5 or
// 10 times a power of ten.
//
// niceNum's ladder has no 2.5, and here that matters. An axis picks its step
// first and lets the tick count follow; a radar's ring count is fixed, so the
// step alone absorbs the rounding, and from 2 the next rung at 5 can more
// than double the scale: a high of 9 over four rings would put the rim at 20
// and the data in the inner half of the chart. With 2.5 the rim is 10.
func radarStep(x float64) float64 {
	if x <= 0 || math.IsNaN(x) || math.IsInf(x, 0) {
		return 1
	}
	pow := math.Pow(10, math.Floor(math.Log10(x)))
	for _, m := range [...]float64{1, 2, 2.5, 5, 10} {
		// The tolerance keeps a float a hair over a rung (0.30000000000000004)
		// on it rather than bumping it to the next.
		if x <= m*pow*(1+1e-9) {
			return m * pow
		}
	}
	return 10 * pow
}

// radarAngle is axis k of n's direction in radians, in screen coordinates (y
// down): straight up for k = 0, then clockwise.
func radarAngle(k, n int) float64 {
	if n <= 0 {
		return -math.Pi / 2
	}
	return -math.Pi/2 + 2*math.Pi*float64(k)/float64(n)
}

// radarFraction is how far out value v sits, 0 at the centre to 1 at the
// rim. See "Values outside the scale".
func radarFraction(v, top float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) || top <= 0 {
		return 0
	}
	return math.Max(0, math.Min(1, v/top))
}

// radarAxisLabel is one rim label: a box the stack centres, translated to
// just outside the end of the spoke at angle, rim px from the centre.
//
// Which edge of the box meets the spoke depends on the side it is on, so text
// always runs away from the drawing and never over it:
//
//	cos > +0.3   right of centre   box's start edge on the point, start-aligned
//	cos < −0.3   left of centre    box's end edge on the point, end-aligned
//	otherwise    top or bottom     centred on the point
//
// and the same three cases on sin decide whether the box sits above, below or
// level. The threshold is wide enough that a seven-axis chart's near-vertical
// labels stay centred, where an edge-anchored box would hang off to one side.
func radarAxisLabel(t *core.Theme, text string, angle, rim, width float64) core.View {
	sin, cos := math.Sincos(angle)
	x, y := (rim+radarLabelGap)*cos, (rim+radarLabelGap)*sin

	align := core.AlignCenter
	switch {
	case cos > 0.3:
		x, align = x+width/2, core.AlignStart
	case cos < -0.3:
		x, align = x-width/2, core.AlignEnd
	}
	switch {
	case sin > 0.3:
		y += chartLabelLine / 2
	case sin < -0.3:
		y -= chartLabelLine / 2
	}
	return core.Column(
		core.Padding(0),
		core.Width(px(width)),
		core.Height(px(chartLabelLine)),
		core.Justify(core.JustifyCenter),
		core.Translate(px(roundTo(x, 1)), px(roundTo(y, 1))),
		chartLabelText(t, text, align, t.Colors.TextSecondary),
	)
}

// radarScaleLabel is one ring's value, out px up the vertical spoke and just
// to its trailing side, start-aligned so the digits run away from the spoke.
//
// The value sits on a chip of the page's Background at radarChipAlpha. A ring
// value is inside the drawing, where a polygon's edge or a marker can pass
// straight through it: the first render of this widget had a profile's 8
// striking out the "7.5" beside it. Nowhere inside the rim is safe from every
// dataset, so the label is made legible over whatever is under it and not
// moved. The chip hugs its text (the fixed-width box around it is only what
// Translate positions), so it hides a few px of line and no more.
func radarScaleLabel(t *core.Theme, text string, out float64) core.View {
	const width = 40.0
	return core.Column(
		core.Padding(0),
		core.Width(px(width)),
		core.Height(px(chartLabelLine)),
		core.Justify(core.JustifyCenter),
		core.AlignItemsProp(core.AlignItemsStart),
		core.Translate(px(width/2+2), px(roundTo(-out, 1))),
		core.Box(
			core.Padding(0),
			core.PaddingHorizontal(2),
			core.BorderRadius(2),
			core.BackgroundColor(withAlpha(t.Colors.Background, radarChipAlpha)),
			chartLabelText(t, text, core.AlignStart, t.Colors.TextSecondary),
		),
	)
}

// label reads each series axis by axis, or its low and high past
// radarSummaryLimit axes.
func (c RadarChart) label() string {
	if c.AccessibilityLabel != "" {
		return c.AccessibilityLabel
	}
	format := c.Format
	if format == nil {
		format = formatValue
	}
	n := len(c.Axes)
	name := func(k int) string {
		if c.Axes[k] != "" {
			return c.Axes[k]
		}
		return "axis " + formatValue(float64(k+1))
	}
	var parts []string
	for _, s := range c.Series {
		prefix := ""
		if s.Name != "" {
			prefix = s.Name + ", "
		}
		var vals []string
		low, high := -1, -1
		for k := 0; k < n && k < len(s.Values); k++ {
			v := s.Values[k]
			if math.IsNaN(v) || math.IsInf(v, 0) {
				continue
			}
			vals = append(vals, name(k)+" "+format(v))
			if low < 0 || v < s.Values[low] {
				low = k
			}
			if high < 0 || v > s.Values[high] {
				high = k
			}
		}
		switch {
		case len(vals) == 0:
			parts = append(parts, prefix+"no data")
		case n <= radarSummaryLimit:
			parts = append(parts, prefix+strings.Join(vals, ", "))
		default:
			parts = append(parts, prefix+formatValue(float64(len(vals)))+" axes, low "+
				format(s.Values[low])+" in "+name(low)+", high "+format(s.Values[high])+" in "+name(high))
		}
	}
	if len(parts) == 0 {
		parts = append(parts, "no data")
	}
	return summaryPrefix(c.Subject) + joinSentences(parts)
}
