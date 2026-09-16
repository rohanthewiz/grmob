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

	// Format writes a y tick label and a spoken value.
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

	h := c.Height
	if h <= 0 {
		h = chartHeight
	}
	fill := c.BarFill
	if fill <= 0 || fill > 1 {
		fill = 0.7
	}

	n := len(c.Labels)
	for _, s := range c.Series {
		n = max(n, len(s.Values))
	}
	lo, hi, _ := dataRange(c.Series)
	scale := niceScale(lo, hi, chartMaxTicks(h), true)

	palette := c.Colors
	if len(palette) == 0 {
		palette = chartPalette(t)
	}

	shapes := gridShapes(t, scale)
	colors := make([]string, len(c.Series))
	names := make([]string, len(c.Series))
	m := max(1, len(c.Series))
	base := scale.y(0)
	if n > 0 {
		slot := chartView / float64(n)
		group := slot * fill
		bar := group / float64(m)
		// A sliver between bars of one group so neighbours read as two bars,
		// not one striped one. None for a single series: the slot gap does
		// that job.
		inner := 0.0
		if m > 1 {
			inner = bar * 0.12
		}
		for j, s := range c.Series {
			color := seriesColor(palette, s.Color, j)
			colors[j], names[j] = color, s.Name
			// One path per series holding every bar as its own closed
			// subpath: n rectangles cost one node, and a series whose values
			// change patches one prop.
			p := core.NewPath()
			for i, v := range s.Values {
				if math.IsNaN(v) || math.IsInf(v, 0) || v == 0 {
					continue
				}
				x := float64(i)*slot + (slot-group)/2 + float64(j)*bar + inner/2
				y := scale.y(v)
				top, height := math.Min(y, base), math.Abs(y-base)
				p.MoveTo(x, top).LineTo(x+bar-inner, top).LineTo(x+bar-inner, top+height).LineTo(x, top+height).Close()
			}
			shapes = append(shapes, core.Shape{Path: p, Fill: color})
		}
	}
	// The zero line over the bars' feet, a step darker than the gridlines so
	// negative bars read as hanging from it.
	shapes = append(shapes, core.Shape{
		Path:   core.Line(0, base, chartView, base),
		Stroke: t.Colors.ControlBorderColor(), StrokeWidth: 1,
	})

	canvas := core.Canvas(chartView, chartView, shapes, core.CanvasStretch, core.Height(px(h)))

	var legendView core.View
	if len(c.Series) > 1 {
		legendView = legend(t, names, colors)
	}

	return cartesianFrame(ctx, scale, h, c.Format, canvas, bandLabels(t, c.Labels, n),
		legendView, c.label(n), c.Style)
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
