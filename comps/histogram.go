package comps

import (
	"math"

	"github.com/rohanthewiz/grmob/core"
)

// Histogram counts raw values into bins and draws the counts as touching bars
// over a numeric axis — how response times, ages or scores are spread.
//
//	comps.Histogram{
//	    Subject: "Response time (ms)",
//	    Values:  samples,          // the raw values, not counts
//	}
//
//	 12 ┤      ██
//	    │   ██ ██ ██
//	  6 ┤   ██ ██ ██ ██
//	    │██ ██ ██ ██ ██ ██
//	  0 ┼──┴──┴──┴──┴──┴──┤
//	    0    100   200   300              labels on the bin *edges*
//
// # The axis is numeric, and that was the catch
//
// A histogram looks like a BarChart, and the binning is a few lines of Go —
// the plan's catch was that BarChart's categories are strings, centred under
// their bars, while a bin is a *range* whose edges are what the axis names.
// "100–150" under a bar is a category chart of ranges; "100" and "150" at the
// bars' shoulders is a histogram, and the difference is what a reader
// measures a value against.
//
// It settled without a new axis. n bins have n+1 edges evenly spaced from the
// plot's left edge to its right, which is exactly the spacing LineChart gives
// its points — so the edge labels are pointLabels, the row LineChart and the
// horizontal BarChart already use for axes that run edge to edge. The bars
// are BarChart's single-series slots with the gap between them nearly closed:
// a histogram's bars touch, because the bins do.
//
// # Nice edges
//
// The edges come from the charts' own niceScale, so bins are 10 or 25 or 0.5
// wide and start on a multiple of that — "0, 50, 100", never "3.7, 51.2". Bins
// is therefore a target, and the drawn count is at most it: 12 asked over
// 0–95 may come out as 10 bins of 10. Zero Bins takes Sturges' rule,
// ⌈log₂ n⌉ + 1, the usual default for a sample of unknown shape. A value on
// an interior edge belongs to the bin it opens (the half-open [a, b) every
// statistics package uses), and the largest value, on the last edge, to the
// last bin.
//
// # Counts are whole
//
// The count axis never ticks at a fraction: a step below 1 is raised to 1,
// because "half a sample" is a gridline pointing at nothing.
//
// # One element, one sentence
//
// "Response time (ms): 120 values in 8 bins of 50 from 0 to 400; most, 34,
// between 100 and 150."
//
// # Theme roles read
//
//	Bars       Chart slot 1 (Colors.ChartColors), or Color
//	Grid       Colors.BorderColor; the zero line Colors.ControlBorderColor
//	Labels     Colors.TextSecondary at the charts' label size
type Histogram struct {
	// Values are the raw observations. NaN and infinities are skipped.
	Values []float64

	// Bins is the most bins drawn; 0 means Sturges' rule. See "Nice edges".
	Bins int

	// Subject says what was measured. It leads the spoken summary and is not
	// drawn.
	Subject string

	// Height is the plot's height in px, excluding labels; 0 means 160.
	Height float64

	// Color overrides the bars' colour.
	Color string

	// Format writes an edge label and the edges in the summary; nil writes
	// them as the axis would, with the decimals the bin width needs.
	Format func(float64) string

	// Style is applied last, to the outer column.
	Style []core.StyleProp

	// AccessibilityLabel replaces the generated summary.
	AccessibilityLabel string
}

// histogramBarFill is the share of each bin its bar takes: nearly all, so the
// bars read as one shape cut into bins rather than as separate categories,
// with a hairline between them so neighbouring counts can still be told apart.
const histogramBarFill = 0.94

func (c Histogram) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	h := c.Height
	if h <= 0 {
		h = chartHeight
	}

	bins := c.bin()
	// The count axis: zero-based as every bar axis is, and never finer than
	// whole counts.
	peak := 0
	for _, n := range bins.counts {
		peak = max(peak, n)
	}
	yscale := niceScale(0, float64(peak), chartMaxTicks(h), true)
	if yscale.step < 1 {
		yscale.step = 1
		yscale.hi = math.Max(1, math.Ceil(yscale.hi))
	}

	color := c.Color
	if color == "" {
		color = seriesColor(chartPalette(t), "", 0)
	}

	shapes := gridShapes(t, yscale)
	nb := len(bins.counts)
	if nb > 0 {
		slot := chartView / float64(nb)
		inset := slot * (1 - histogramBarFill) / 2
		p := core.NewPath()
		for i, n := range bins.counts {
			if n == 0 {
				continue
			}
			x0, x1 := float64(i)*slot+inset, float64(i+1)*slot-inset
			y0, y1 := yscale.y(float64(n)), yscale.y(0)
			p.MoveTo(x0, y0).LineTo(x1, y0).LineTo(x1, y1).LineTo(x0, y1).Close()
		}
		shapes = append(shapes, core.Shape{Path: p, Fill: color})
	}
	shapes = append(shapes, core.Shape{
		Path:   core.Line(0, yscale.y(0), chartView, yscale.y(0)),
		Stroke: t.Colors.ControlBorderColor(), StrokeWidth: 1,
	})
	canvas := core.Canvas(chartView, chartView, shapes, core.CanvasStretch, core.CanvasMirrorsRTL, core.Height(px(h)))

	format := c.edgeFormat(bins)
	var xLabels core.View
	if nb > 0 {
		edges := make([]string, nb+1)
		for i := range edges {
			edges[i] = format(bins.edge(i))
		}
		xLabels = pointLabels(t, edges, nb+1)
	}

	label := c.AccessibilityLabel
	if label == "" {
		label = c.summary(bins, format)
	}
	return cartesianFrameWithValues(ctx, yscale, h, nil, canvas, xLabels, nil, label, c.Style, nil, 0)
}

// histBins is the binning: the first edge, the bin width and a count per bin.
type histBins struct {
	lo, step float64
	counts   []int
	total    int
}

// edge is the value of edge i, rounded to the step's precision so "0.30000000004"
// never reaches a label.
func (b histBins) edge(i int) float64 {
	return roundTo(b.lo+float64(i)*b.step, stepDecimals(b.step))
}

// bin counts the finite values into nice bins; see "Nice edges".
func (c Histogram) bin() histBins {
	lo, hi := math.Inf(1), math.Inf(-1)
	n := 0
	for _, v := range c.Values {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			continue
		}
		lo, hi = math.Min(lo, v), math.Max(hi, v)
		n++
	}
	if n == 0 {
		return histBins{}
	}

	target := c.Bins
	if target <= 0 {
		target = int(math.Ceil(math.Log2(float64(n)))) + 1
	}
	target = max(1, target)

	// k bins have k+1 edges, and niceScale counts ticks, which are edges.
	s := niceScale(lo, hi, target+1, false)
	nb := max(1, int(math.Round((s.hi-s.lo)/s.step)))
	b := histBins{lo: s.lo, step: s.step, counts: make([]int, nb), total: n}
	for _, v := range c.Values {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			continue
		}
		// Half-open [a, b): floor puts an interior edge in the bin it opens,
		// and the clamp puts the last edge in the last bin.
		i := min(nb-1, max(0, int(math.Floor((v-b.lo)/b.step))))
		b.counts[i]++
	}
	return b
}

// edgeFormat is Format, or the axis's own tick form for these edges.
func (c Histogram) edgeFormat(b histBins) func(float64) string {
	if c.Format != nil {
		return c.Format
	}
	extent := math.Max(math.Abs(b.lo), math.Abs(b.edge(len(b.counts))))
	return func(v float64) string { return formatTick(v, b.step, extent) }
}

// summary is the one sentence the chart is announced as.
func (c Histogram) summary(b histBins, format func(float64) string) string {
	nb := len(b.counts)
	if nb == 0 {
		return joinSentences([]string{summaryPrefix(c.Subject) + "no values"})
	}
	parts := []string{summaryPrefix(c.Subject) + plural2(b.total, "value") + " in " +
		plural2(nb, "bin") + " of " + format(b.step) + " from " + format(b.edge(0)) + " to " + format(b.edge(nb))}
	best := 0
	for i, n := range b.counts {
		if n > b.counts[best] {
			best = i
		}
	}
	parts = append(parts, "most, "+formatValue(float64(b.counts[best]))+", between "+
		format(b.edge(best))+" and "+format(b.edge(best+1)))
	return joinSentences(parts)
}
