package comps

import (
	"math"
	"strings"

	"github.com/rohanthewiz/grmob/core"
)

// BarChart draws a value per category as a vertical bar, with several series
// grouped side by side within each category.
//
//	comps.BarChart{
//	    Subject: "Steps",
//	    Labels:  []string{"Mon", "Tue", "Wed"},
//	    Series:  []comps.ChartSeries{{Name: "Steps", Values: []float64{8200, 10400, 6100}}},
//	}
//
// # Slots
//
// The width is divided into one equal slot per category, and a category's
// bars share the middle of its slot:
//
//	│  slot 0   │  slot 1   │  slot 2   │
//	│   ▐█▌▐█▌  │   ▐█▌▐█▌  │   ▐█▌▐█▌  │    group = BarFill of the slot
//	│   Mon     │   Tue     │   Wed     │    labels in equal boxes, centred
//
// Equal slots are what make the labels exact: the label row is the same
// number of equal flex boxes, so each label's centre is its slot's centre on
// every target, without Go knowing how wide the chart is drawn.
//
// # Zero and negatives
//
// The axis always includes zero, and bars grow from it — up for positive
// values, down for negative ones. A bar's length is only proportional to its
// value when it starts at zero, which is why this is not optional as it is on
// LineChart.
//
// # Stacked
//
// Stacked puts a category's series in one bar, end to end, instead of side by
// side. Positive values stack up from zero and negative ones down from it,
// each in its own running total, so a negative part never hides a positive
// one (the convention spreadsheet charts use). The axis spans the totals.
//
// # Horizontal
//
// Horizontal lays the categories top to bottom and the bars along the value
// axis, which is the arrangement for category names too long to sit under a
// bar:
//
//	┌──────────┬────────────────────────────────┐
//	│  Rent    │██████████████████ 1200         │  one band per category,
//	│  Food    │█████████ 450                   │  Height/n px each
//	│ Transpo… │████ 200                        │
//	└──────────┼────────────────────────────────┘
//	           0      500     1000   1500         ticks on a point axis
//	 names: MaxLines(1), at most LabelWidth wide; values: ShowValues
//
// The name column is sized by its widest name, up to LabelWidth, and a longer
// name is cut with an ellipsis (core.MaxLines). So is a tick label wider than
// its box. Every tick's box is two thirds of an interval wide (see
// pointLabels; the end ones cannot extend past the plot's edges), so a Format
// that writes "$1500" where "$1.5k" would do can still be cut on a narrow
// plot. The bands are fixed px boxes rather than flex weights, since Go knows
// the plot's height exactly; the tick labels use LineChart's point
// arithmetic, because ticks, like points, run edge to edge.
//
// # Values
//
// ShowValues writes each bar's value at its tip: just above a vertical bar
// (below one that hangs negative), just past a horizontal one's end. A stacked
// chart shows each category's total at the end of its stack, and a grouped
// one a value per bar.
//
// Text cannot be placed inside core.Canvas, so the values are a layer of
// ordinary Text over it (core.ZStack), and each is placed by arithmetic Go can
// do without knowing the drawn size of the plot:
//
//	vertical     across: the bar's share of the width, as flex weights —
//	             the cells barValueCells centres on the bars
//	             along:  px, because the plot is Height px tall and a
//	             stretched viewBox maps y linearly onto it
//	horizontal   along:  flex weights again, a spacer as long as the bar's
//	             share of the width before the label
//	             across: px bands, as the names column has
//
// A vertical chart keeps a label line of headroom above the plot (and one
// below, when a value is negative), so a bar reaching the end of the axis
// still has room for its label. A horizontal one cannot reserve room it
// cannot measure, so a bar leaving less than a quarter of the plot past its
// tip (barValueRoom) carries its value inside, against its end, in whichever
// of the theme's inks contrasts with the bar.
type BarChart struct {
	// Series are the groups' members, in order within each group. Value i of
	// each series belongs to category i.
	Series []ChartSeries

	// Labels name the categories. Up to five are drawn (every other one, or
	// every third, past that); all of them feed the spoken summary.
	Labels []string

	// Subject says what the chart measures. It leads the spoken summary and is
	// not drawn.
	Subject string

	// Height is the plot's height in px, excluding labels; 0 means 160.
	Height float64

	// BarFill is the fraction of each slot the group of bars takes; 0 means
	// 0.7, leaving the rest as space between categories.
	BarFill float64

	// Colors overrides the theme palette for series without their own Color.
	Colors []string

	// Stacked draws a category's series as one bar, end to end. See
	// "Stacked" above.
	Stacked bool

	// Horizontal lays the bars along the value axis, categories top to
	// bottom. See "Horizontal" above. Height is then the whole plot's height,
	// and 0 means 28 px a category (at least 56).
	Horizontal bool

	// LabelWidth caps the category name column of a horizontal chart, in px;
	// 0 means 96. Unused by vertical bars.
	LabelWidth float64

	// ShowValues writes each bar's value (a stack's total) at the bar's tip.
	// See "Values" above.
	ShowValues bool

	// Format writes a tick label, a shown value and a spoken value.
	Format func(float64) string

	// Style is applied last, to the outer column.
	Style []core.StyleProp

	// AccessibilityLabel replaces the generated summary.
	AccessibilityLabel string
}

// barSummaryLimit is the most categories whose values are all read out. Past
// it the summary gives the range instead, as LineChart's does.
const barSummaryLimit = 8

func (c BarChart) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	fill := c.BarFill
	if fill <= 0 || fill > 1 {
		fill = 0.7
	}

	n := len(c.Labels)
	for _, s := range c.Series {
		n = max(n, len(s.Values))
	}
	h := c.Height
	if h <= 0 {
		h = chartHeight
		if c.Horizontal {
			h = math.Max(56, 28*float64(n))
		}
	}

	lo, hi, _ := dataRange(c.Series)
	if c.Stacked {
		lo, hi = stackRange(c.Series, n)
	}
	maxTicks := chartMaxTicks(h)
	if c.Horizontal {
		// The ticks share a width Go does not know with labels as wide as
		// "15k"; five fit a phone and match what pointLabels draws in full.
		maxTicks = chartMaxXLabels
	}
	scale := niceScale(lo, hi, maxTicks, true)

	palette := c.Colors
	if len(palette) == 0 {
		palette = chartPalette(t)
	}
	colors := make([]string, len(c.Series))
	names := make([]string, len(c.Series))
	for j, s := range c.Series {
		colors[j], names[j] = seriesColor(palette, s.Color, j), s.Name
	}

	bars := c.barRects(n, fill, scale)
	var shapes []core.Shape
	if c.Horizontal {
		shapes = verticalGridShapes(t, scale)
	} else {
		shapes = gridShapes(t, scale)
	}
	for j := range c.Series {
		// One path per series holding every bar as its own closed subpath:
		// n rectangles cost one node, and a series whose values change
		// patches one prop.
		p := core.NewPath()
		for _, r := range bars[j] {
			x, y, w, bh := r.a0, r.b0, r.a1-r.a0, r.b1-r.b0
			if c.Horizontal {
				// The rect was computed with the band along a and the value
				// along b; horizontal bars swap the axes.
				x, y, w, bh = r.b0, r.a0, r.b1-r.b0, r.a1-r.a0
			}
			p.MoveTo(x, y).LineTo(x+w, y).LineTo(x+w, y+bh).LineTo(x, y+bh).Close()
		}
		shapes = append(shapes, core.Shape{Path: p, Fill: colors[j]})
	}
	// The zero line over the bars' feet, a step darker than the gridlines so
	// negative bars read as hanging from it.
	zero := core.Line(0, scale.y(0), chartView, scale.y(0))
	if c.Horizontal {
		zx := chartView - scale.y(0)
		zero = core.Line(zx, 0, zx, chartView)
	}
	shapes = append(shapes, core.Shape{
		Path:   zero,
		Stroke: t.Colors.ControlBorderColor(), StrokeWidth: 1,
	})

	canvas := core.Canvas(chartView, chartView, shapes, core.CanvasStretch, core.Height(px(h)))

	var legendView core.View
	if len(c.Series) > 1 {
		legendView = legend(t, names, colors)
	}

	format := c.Format
	if format == nil {
		format = formatValue
	}
	if c.Horizontal {
		return c.horizontalFrame(ctx, scale, h, n, fill, canvas, legendView, colors, format)
	}
	var values core.View
	bottomExtra := 0.0
	if c.ShowValues {
		values, bottomExtra = c.valueLayer(t, n, fill, scale, h, colors, format)
	}
	return cartesianFrameWithValues(ctx, scale, h, c.Format, canvas, bandLabels(t, c.Labels, n),
		legendView, c.label(n), c.Style, values, bottomExtra)
}

// barRect is one bar in band-and-value coordinates: a runs across the
// category's band (viewBox x for vertical bars, y for horizontal ones) and b
// along the value axis, in the direction the drawing runs (y down for
// vertical bars, x rightwards for horizontal ones).
type barRect struct{ a0, a1, b0, b1 float64 }

// barRects lays out every bar, per series, in barRect coordinates.
//
//	grouped:  │ pad │ s0 │ s1 │ s2 │ pad │   each series its own sub-slot,
//	                                        a sliver apart
//	stacked:  │ pad │   one bar    │ pad │   series end to end within it,
//	                                        positives up from 0, negatives down
func (c BarChart) barRects(n int, fill float64, scale valueScale) [][]barRect {
	out := make([][]barRect, len(c.Series))
	if n == 0 {
		return out
	}
	// The value axis in drawing direction: vertical bars grow towards smaller
	// y, horizontal ones towards larger x, which is the y scale mirrored.
	at := scale.y
	if c.Horizontal {
		at = func(v float64) float64 { return chartView - scale.y(v) }
	}
	span := func(from, to float64) (float64, float64) {
		a, b := at(from), at(to)
		return math.Min(a, b), math.Max(a, b)
	}

	slot := chartView / float64(n)
	group := slot * fill
	m := max(1, len(c.Series))
	if c.Stacked {
		up := make([]float64, n)
		down := make([]float64, n)
		for j, s := range c.Series {
			for i, v := range s.Values {
				if i >= n || math.IsNaN(v) || math.IsInf(v, 0) || v == 0 {
					continue
				}
				total := &up[i]
				if v < 0 {
					total = &down[i]
				}
				b0, b1 := span(*total, *total+v)
				*total += v
				a0 := float64(i)*slot + (slot-group)/2
				out[j] = append(out[j], barRect{a0, a0 + group, b0, b1})
			}
		}
		return out
	}
	bar := group / float64(m)
	// A sliver between bars of one group so neighbours read as two bars, not
	// one striped one. None for a single series: the slot gap does that job.
	inner := 0.0
	if m > 1 {
		inner = bar * 0.12
	}
	for j, s := range c.Series {
		for i, v := range s.Values {
			if i >= n || math.IsNaN(v) || math.IsInf(v, 0) || v == 0 {
				continue
			}
			a0 := float64(i)*slot + (slot-group)/2 + float64(j)*bar + inner/2
			b0, b1 := span(0, v)
			out[j] = append(out[j], barRect{a0, a0 + bar - inner, b0, b1})
		}
	}
	return out
}

// stackRange is the lowest negative total and highest positive total across
// categories, and zero when there are none of either.
func stackRange(series []ChartSeries, n int) (lo, hi float64) {
	for i := range n {
		up, down := 0.0, 0.0
		for _, s := range series {
			if i >= len(s.Values) || math.IsNaN(s.Values[i]) || math.IsInf(s.Values[i], 0) {
				continue
			}
			if v := s.Values[i]; v > 0 {
				up += v
			} else {
				down += v
			}
		}
		lo, hi = math.Min(lo, down), math.Max(hi, up)
	}
	return lo, hi
}

// barValueCells lists what ShowValues writes, in band order, with each cell's
// share of the band axis: per category a leading pad, then one cell per bar
// (a stack's total, or each series' value when grouped), then a trailing pad.
// The shares are the same arithmetic barRects uses, so a cell centres on its
// bar exactly.
func (c BarChart) barValueCells(n int, fill float64, format func(float64) string) (texts []string, weights []float64) {
	if n == 0 {
		return nil, nil
	}
	m := max(1, len(c.Series))
	pad := (1 - fill) / 2
	value := func(j, i int) (float64, bool) {
		if j >= len(c.Series) || i >= len(c.Series[j].Values) {
			return 0, false
		}
		v := c.Series[j].Values[i]
		return v, !math.IsNaN(v) && !math.IsInf(v, 0)
	}
	for i := range n {
		texts, weights = append(texts, ""), append(weights, pad)
		if c.Stacked {
			total, any := 0.0, false
			for j := range c.Series {
				if v, ok := value(j, i); ok {
					total, any = total+v, true
				}
			}
			text := ""
			if any {
				text = format(total)
			}
			texts, weights = append(texts, text), append(weights, fill)
		} else {
			for j := range m {
				text := ""
				if v, ok := value(j, i); ok {
					text = format(v)
				}
				texts, weights = append(texts, text), append(weights, fill/float64(m))
			}
		}
		texts, weights = append(texts, ""), append(weights, pad)
	}
	return texts, weights
}

// barValueRoom is the least share of a horizontal plot's width a bar must
// leave past its tip for its value to be written outside it. A quarter of a
// phone's plot is some 55 px, room for "$1.5k" or "10400" at the label size
// with the gap before it; a bar longer than that carries its value inside.
const barValueRoom = 0.25

// barTip is where a ShowValues cell's label goes, along the value axis in
// viewBox drawing coordinates (y down for vertical bars, x rightwards for
// horizontal ones): the bar's far end, its zero end, whether the bar runs
// towards negative values, and the colour of the bar the label ends. ok is
// false for the pad cells and for a missing value.
type barTip struct {
	ok       bool
	at, zero float64
	negative bool
	color    string
}

// barValueTips is barValueCells's geometry, cell for cell in the same order:
// where each value's bar ends. A stack ends at its positive total when it has
// one and at its negative total otherwise, and its colour is the outermost
// segment's, the one the label touches.
func (c BarChart) barValueTips(n int, scale valueScale, colors []string) []barTip {
	if n == 0 {
		return nil
	}
	at := scale.y
	if c.Horizontal {
		at = func(v float64) float64 { return chartView - scale.y(v) }
	}
	m := max(1, len(c.Series))
	value := func(j, i int) (float64, bool) {
		if j >= len(c.Series) || i >= len(c.Series[j].Values) {
			return 0, false
		}
		v := c.Series[j].Values[i]
		return v, !math.IsNaN(v) && !math.IsInf(v, 0)
	}
	color := func(j int) string {
		if j < len(colors) {
			return colors[j]
		}
		return ""
	}
	var out []barTip
	for i := range n {
		out = append(out, barTip{})
		if c.Stacked {
			up, down, upColor, downColor, any := 0.0, 0.0, "", "", false
			for j := range c.Series {
				v, ok := value(j, i)
				if !ok {
					continue
				}
				any = true
				if v > 0 {
					up, upColor = up+v, color(j)
				} else if v < 0 {
					down, downColor = down+v, color(j)
				}
			}
			tip := barTip{ok: any, at: at(up), zero: at(0), color: upColor}
			if up == 0 && down < 0 {
				tip = barTip{ok: true, at: at(down), zero: at(0), negative: true, color: downColor}
			}
			out = append(out, tip)
		} else {
			for j := range m {
				v, ok := value(j, i)
				out = append(out, barTip{ok: ok, at: at(v), zero: at(0), negative: v < 0, color: color(j)})
			}
		}
		out = append(out, barTip{})
	}
	return out
}

// valueLayer is ShowValues for vertical bars: a row of cells as wide as the
// ones barValueCells weighs, laid over the plot's headroom, the plot and its
// foot, each dropping its label to its bar's tip by a px spacer.
//
//	┌ headroom: a label line, and the half line every plot has ┐
//	│        1200                                              │
//	│        ▐██▌   450                                        │  label top =
//	│        ▐██▌  ▐██▌                                        │  headroom + tip·h/100 − line
//	├────────────────────── zero ──────────────────────────────┤
//	│                     ▐██▌                                 │  a negative bar's label
//	│                      −80                                 │  starts at its tip
//	└ foot: the half line, and another half when a value is negative ┘
//
// bottomExtra is that other half line, which cartesianFrameWithValues adds
// under the canvas so the lowest label clears the category names.
func (c BarChart) valueLayer(t *core.Theme, n int, fill float64, scale valueScale, h float64,
	colors []string, format func(float64) string) (layer core.View, bottomExtra float64) {
	texts, weights := c.barValueCells(n, fill, format)
	tips := c.barValueTips(n, scale, colors)
	if len(texts) == 0 || len(tips) != len(texts) {
		return nil, 0
	}
	for k, tip := range tips {
		if tip.ok && tip.negative && texts[k] != "" {
			bottomExtra = chartLabelLine / 2
		}
	}
	headroom := chartLabelLine * 1.5
	total := headroom + h + chartLabelLine/2 + bottomExtra

	items := make([]core.PropsAndChildren, 0, len(texts)+5)
	items = append(items, core.Padding(0), core.Gap(0), core.Width("100%"), core.Height(px(total)),
		core.AccessibilityHidden())
	for k, text := range texts {
		top := 0.0
		if tip := tips[k]; tip.ok && text != "" {
			top = headroom + tip.at/chartView*h
			if !tip.negative {
				top -= chartLabelLine
			}
		} else {
			text = ""
		}
		items = append(items, core.Column(
			core.Padding(0),
			core.Gap(0),
			core.FlexGrow(weights[k]),
			core.FlexBasis("0"),
			core.MinWidth("0px"),
			core.Box(core.Padding(0), core.Height(px(math.Max(0, top)))),
			core.Column(
				core.Padding(0),
				core.Height(px(chartLabelLine)),
				core.Justify(core.JustifyCenter),
				chartLabelText(t, text, core.AlignCenter, t.Colors.TextSecondary),
			),
		))
	}
	return core.Row(items...), bottomExtra
}

// bandValueLayer is ShowValues for horizontal bars: a column h px tall of the
// bands barValueCells weighs, each a row that puts its label past its bar's
// tip, or inside the bar against its end when too little of the plot is left
// (see barValueRoom). The three weights of a row always sum to the plot's
// width, so a label's edge is the bar's end on every target.
//
//	outside, positive   [ spacer: tip ][ label → ]
//	inside,  positive   [ spacer: zero ][ ← label: zero..tip ][ rest ]
//	outside, negative   [ ← label: tip ][ spacer: rest ]
//	inside,  negative   [ spacer: tip ][ label → : tip..zero ][ rest ]
func (c BarChart) bandValueLayer(t *core.Theme, n int, fill float64, scale valueScale, h float64,
	colors []string, format func(float64) string) core.View {
	texts, weights := c.barValueCells(n, fill, format)
	tips := c.barValueTips(n, scale, colors)
	total := 0.0
	for _, w := range weights {
		total += w
	}
	if len(texts) == 0 || len(tips) != len(texts) || total <= 0 {
		return nil
	}
	const gap = 4
	segment := func(weight float64, text string, align core.Alignment, ink string, side core.StyleProp) core.View {
		if weight <= 1e-9 {
			return nil
		}
		props := []core.PropsAndChildren{
			core.Padding(0),
			core.FlexGrow(weight),
			core.FlexBasis("0"),
			core.MinWidth("0px"),
		}
		if side != nil {
			props = append(props, side)
		}
		if text != "" {
			props = append(props, chartLabelText(t, text, align, ink))
		}
		return core.Column(props...)
	}

	items := make([]core.PropsAndChildren, 0, len(texts)+5)
	items = append(items, core.Padding(0), core.Gap(0), core.Width("100%"), core.Height(px(h)),
		core.AccessibilityHidden())
	for k, text := range texts {
		row := []core.PropsAndChildren{
			core.Padding(0),
			core.Gap(0),
			core.Height(px(h * weights[k] / total)),
			core.AlignItemsProp(core.AlignItemsCenter),
		}
		tip := tips[k]
		if !tip.ok || text == "" {
			items = append(items, core.Row(row...))
			continue
		}
		p, z := tip.at/chartView, tip.zero/chartView
		outside, inside := t.Colors.TextSecondary, contrastInk(tip.color, t.Colors.TextPrimary, t.Colors.Background)
		switch {
		case !tip.negative && (1-p >= barValueRoom || p-z < barValueRoom):
			row = append(row,
				segment(p, "", core.AlignStart, "", nil),
				segment(1-p, text, core.AlignStart, outside, core.PaddingLeft(gap)))
		case !tip.negative:
			row = append(row,
				segment(z, "", core.AlignStart, "", nil),
				segment(p-z, text, core.AlignEnd, inside, core.PaddingRight(gap)),
				segment(1-p, "", core.AlignStart, "", nil))
		case p >= barValueRoom || z-p < barValueRoom:
			row = append(row,
				segment(p, text, core.AlignEnd, outside, core.PaddingRight(gap)),
				segment(1-p, "", core.AlignStart, "", nil))
		default:
			row = append(row,
				segment(p, "", core.AlignStart, "", nil),
				segment(z-p, text, core.AlignStart, inside, core.PaddingLeft(gap)),
				segment(1-z, "", core.AlignStart, "", nil))
		}
		items = append(items, core.Row(row...))
	}
	return core.Column(items...)
}

// chartLabelText is one line of chart label text: the label size, cut with an
// ellipsis when its box is narrower, and hidden from readers, who get the
// summary instead.
func chartLabelText(t *core.Theme, text string, align core.Alignment, ink string) core.View {
	return core.Text(text,
		core.FontSize(chartLabelSize),
		core.TextColor(ink),
		core.Align(align),
		core.MaxLines(1),
		core.AccessibilityHidden(),
	)
}

// horizontalFrame lays out a horizontal bar chart: the name column, the plot
// over its tick labels, and the value column. See "Horizontal" above.
func (c BarChart) horizontalFrame(ctx *core.Context, scale valueScale, h float64, n int, fill float64,
	canvas, legendView core.View, colors []string, format func(float64) string) *core.Node {
	t := ctx.Theme()

	labelWidth := c.LabelWidth
	if labelWidth <= 0 {
		labelWidth = 96
	}
	names := make([]string, n)
	for i := range names {
		if i < len(c.Labels) {
			names[i] = c.Labels[i]
		}
	}

	ticks := scale.ticks()
	extent := math.Max(math.Abs(scale.lo), math.Abs(scale.hi))
	tickText := make([]string, len(ticks))
	for i, v := range ticks {
		tickText[i] = formatTick(v, scale.step, extent)
		if c.Format != nil {
			tickText[i] = c.Format(v)
		}
	}

	// The plot, with the values laid over it when shown. The stack is pinned
	// to the canvas's own size so neither layer decides it (see core.ZStack).
	plot := canvas
	if c.ShowValues {
		if layer := c.bandValueLayer(t, n, fill, scale, h, colors, format); layer != nil {
			plot = core.ZStack(core.Padding(0), core.Width("100%"), core.Height(px(h)), canvas, layer)
		}
	}
	row := []core.PropsAndChildren{
		core.Padding(0),
		core.Gap(6),
		core.AlignItemsProp(core.AlignItemsStart),
		bandColumn(t, names, repeatWeight(1, n), h, core.AlignEnd,
			core.MaxWidth(px(labelWidth))),
		core.Column(
			core.Padding(0),
			core.Gap(0),
			core.FlexGrow(1),
			core.FlexBasis("0"),
			core.MinWidth("0px"),
			plot,
			pointLabels(t, tickText, len(ticks)),
		),
	}

	items := make([]core.PropsAndChildren, 0, 8+len(c.Style))
	items = append(items,
		core.Padding(0),
		core.Gap(float64(t.Spacing.SM)),
		core.AccessibilityRole(core.RoleImg),
		core.AccessibilityLabel(c.label(n)),
	)
	for _, sp := range c.Style {
		items = append(items, sp)
	}
	items = append(items, core.Row(row...))
	if legendView != nil {
		items = append(items, legendView)
	}
	return core.Column(items...).Render(ctx)
}

// repeatWeight is n copies of w.
func repeatWeight(w float64, n int) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = w
	}
	return out
}

// bandColumn is a column h px tall of text cells whose heights are h in
// proportion to weights: a horizontal chart's names or values, each centred
// on its band or bar. Heights are px, not flex weights, since h is known.
// Each text is one line, cut with an ellipsis when the column is narrower.
func bandColumn(t *core.Theme, texts []string, weights []float64, h float64, align core.Alignment, extra ...core.PropsAndChildren) core.View {
	total := 0.0
	for _, w := range weights {
		total += w
	}
	items := make([]core.PropsAndChildren, 0, len(texts)+4+len(extra))
	items = append(items, core.Padding(0), core.Gap(0), core.Height(px(h)), core.AccessibilityHidden())
	items = append(items, extra...)
	if total <= 0 {
		return core.Column(items...)
	}
	for k, text := range texts {
		items = append(items, core.Column(
			core.Padding(0),
			core.Height(px(h*weights[k]/total)),
			core.Justify(core.JustifyCenter),
			core.Text(text,
				core.FontSize(chartLabelSize),
				core.TextColor(t.Colors.TextSecondary),
				core.Align(align),
				core.MaxLines(1),
				core.AccessibilityHidden(),
			),
		))
	}
	return core.Column(items...)
}

// label reads every value when there are at most barSummaryLimit categories
// ("Mon 8200, Tue 10400, Wed 6100"), and the low and high otherwise.
func (c BarChart) label(n int) string {
	if c.AccessibilityLabel != "" {
		return c.AccessibilityLabel
	}
	format := c.Format
	if format == nil {
		format = formatValue
	}
	name := func(i int) string {
		if i < len(c.Labels) && c.Labels[i] != "" {
			return c.Labels[i]
		}
		return "item " + formatValue(float64(i+1))
	}
	var parts []string
	for _, s := range c.Series {
		prefix := ""
		if s.Name != "" && len(c.Series) > 1 {
			prefix = s.Name + ", "
		}
		var vals []string
		low, high := -1, -1
		for i, v := range s.Values {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				continue
			}
			vals = append(vals, name(i)+" "+format(v))
			if low < 0 || v < s.Values[low] {
				low = i
			}
			if high < 0 || v > s.Values[high] {
				high = i
			}
		}
		switch {
		case len(vals) == 0:
			parts = append(parts, prefix+"no data")
		case n <= barSummaryLimit:
			parts = append(parts, prefix+strings.Join(vals, ", "))
		default:
			parts = append(parts, prefix+formatValue(float64(len(vals)))+" bars, low "+
				format(s.Values[low])+" in "+name(low)+", high "+format(s.Values[high])+" in "+name(high))
		}
	}
	if len(parts) == 0 {
		parts = append(parts, "no data")
	}
	return summaryPrefix(c.Subject) + joinSentences(parts)
}
