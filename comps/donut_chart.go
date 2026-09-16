package comps

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/rohanthewiz/grmob/core"
)

// ChartSlice is one share of a DonutChart or PieChart.
type ChartSlice struct {
	Label string
	Value float64

	// Color overrides the slice's palette colour.
	Color string
}

// DonutChart draws shares of a whole as a ring, with a legend underneath.
//
//	comps.DonutChart{
//	    Subject:     "Budget",
//	    Slices:      []comps.ChartSlice{{Label: "Rent", Value: 1200}, {Label: "Food", Value: 450}},
//	    CenterValue: "$1,650",
//	    CenterLabel: "per month",
//	}
//
// # Geometry
//
// Slices start at twelve o'clock and go clockwise, in the order given — the
// caller's order is kept rather than sorted, because a budget's categories
// have an order of their own that a legend should not reshuffle. Each slice is
// a core.Sector in a 100 × 100 viewBox drawn with CanvasFit, so the ring stays
// round in any box. A Gap of a degree or so is taken out of each slice's
// sweep, split evenly between its two ends, so neighbours of similar colour
// stay distinguishable; a slice too thin to lose that much keeps its full
// sweep instead of vanishing.
//
// # Legend and percentages
//
// The legend lists each slice with its percentage, rounded by largest
// remainder so the column adds up to exactly 100 — independently rounded
// shares of 33⅓ each would read 33, 33, 33.
//
// # Accessibility
//
// One element: "Budget: Rent 1200, 73%; Food 450, 27%". The centre text is
// included when it is set, since it is usually the total.
type DonutChart struct {
	// Slices are the shares, in drawing order. Zero, negative and NaN values
	// are skipped: a share of a whole cannot be negative.
	Slices []ChartSlice

	// Subject says what the whole is. It leads the spoken summary and is not
	// drawn.
	Subject string

	// Size is the ring's diameter in px; 0 means 160.
	Size float64

	// Thickness is the ring's width as a fraction of its radius; 0 means 0.35.
	// PieChart ignores it.
	Thickness float64

	// Gap is the space between slices in degrees; 0 means 1.5. A negative Gap
	// means none.
	Gap float64

	// CenterValue and CenterLabel are drawn in the hole, value over label —
	// typically the total. PieChart has no hole and ignores them.
	CenterValue string
	CenterLabel string

	// HideLegend drops the legend, for a chart whose slices are already named
	// beside it.
	HideLegend bool

	// Colors overrides the theme palette for slices without their own Color.
	Colors []string

	// Format writes a slice's value in the spoken summary.
	Format func(float64) string

	// Style is applied last, to the outer column.
	Style []core.StyleProp

	// AccessibilityLabel replaces the generated summary.
	AccessibilityLabel string

	// pie is set by PieChart: no hole, no centre text, no gap.
	pie bool
}

// PieChart is a DonutChart without the hole: every slice is a wedge from the
// centre. Thickness and the centre text are ignored, and Gap defaults to none,
// since a gap in a pie reads as a crack rather than a separator.
type PieChart DonutChart

func (p PieChart) Render(ctx *core.Context) *core.Node {
	d := DonutChart(p)
	d.pie = true
	return d.Render(ctx)
}

func (c DonutChart) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	size := c.Size
	if size <= 0 {
		size = 160
	}
	thickness := c.Thickness
	if thickness <= 0 || thickness > 1 {
		thickness = 0.35
	}
	gap := c.Gap
	switch {
	case gap < 0:
		gap = 0
	case gap == 0 && !c.pie:
		gap = 1.5
	}

	palette := c.Colors
	if len(palette) == 0 {
		palette = chartPalette(t)
	}

	const r = chartView / 2
	inner := r * (1 - thickness)
	if c.pie {
		inner = 0
	}

	values, total := sliceValues(c.Slices)
	shapes := make([]core.Shape, 0, len(c.Slices)+1)
	// The track: the whole ring in the divider colour, drawn only when there
	// is nothing to show, so an empty chart still says where its data would
	// go. Under slices it would show through the gaps as grey seams. The slot
	// is kept either way, so data arriving patches it rather than shifting
	// every slice.
	track := core.Shape{}
	if total <= 0 {
		track = core.Shape{Path: core.Sector(r, r, inner, r, -90, 360), Fill: t.Colors.BorderColor()}
	}
	shapes = append(shapes, track)

	colors := make([]string, len(c.Slices))
	start := -90.0
	visible := 0
	for _, v := range values {
		if v > 0 {
			visible++
		}
	}
	for i, s := range c.Slices {
		colors[i] = seriesColor(palette, s.Color, i)
		v := values[i]
		if v <= 0 || total <= 0 {
			// An empty shape keeps later slices in their child slots.
			shapes = append(shapes, core.Shape{})
			continue
		}
		sweep := v / total * 360
		from, span := start, sweep
		// See "Geometry": the gap comes out of both ends, unless the slice is
		// alone (a full ring has no neighbour) or too thin to spare it.
		if visible > 1 && sweep > gap*2 {
			from, span = start+gap/2, sweep-gap
		}
		shapes = append(shapes, core.Shape{
			Path: core.Sector(r, r, inner, r, from, span),
			Fill: colors[i],
		})
		start += sweep
	}

	dim := px(size)
	ring := []core.PropsAndChildren{
		core.Width(dim),
		core.Height(dim),
		core.AccessibilityHidden(),
		core.Canvas(chartView, chartView, shapes, core.Width(dim), core.Height(dim)),
	}
	if !c.pie && (c.CenterValue != "" || c.CenterLabel != "") {
		centre := []core.PropsAndChildren{
			core.Padding(0),
			core.Gap(0),
			core.AlignItemsProp(core.AlignItemsCenter),
			// Inside the hole, so a long label wraps rather than running over
			// the ring.
			core.MaxWidth(px(size * (1 - thickness) * 0.8)),
		}
		if c.CenterValue != "" {
			centre = append(centre, core.Text(c.CenterValue,
				core.FontSize(max(14, size*0.13)),
				core.FontWeight(core.Bold),
				core.TextColor(t.Colors.TextPrimary),
				core.Align(core.AlignCenter),
			))
		}
		if c.CenterLabel != "" {
			centre = append(centre, core.Text(c.CenterLabel,
				core.FontSize(chartLabelSize),
				core.TextColor(t.Colors.TextSecondary),
				core.Align(core.AlignCenter),
			))
		}
		ring = append(ring, core.Column(centre...))
	}

	items := make([]core.PropsAndChildren, 0, 8+len(c.Style))
	items = append(items,
		core.Padding(0),
		core.Gap(float64(t.Spacing.MD)),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.AccessibilityRole(core.RoleImg),
		core.AccessibilityLabel(c.label(values, total)),
	)
	for _, sp := range c.Style {
		items = append(items, sp)
	}
	items = append(items, core.ZStack(ring...))
	if !c.HideLegend && len(c.Slices) > 0 {
		items = append(items, c.legend(t, colors, values, total, size))
	}
	return core.Column(items...).Render(ctx)
}

// legend is one row per slice: swatch, label, percentage. It is at least as
// wide as the ring so the percentages line up in a column under it.
func (c DonutChart) legend(t *core.Theme, colors []string, values []float64, total, size float64) core.View {
	pcts := percentages(values, total)
	items := make([]core.PropsAndChildren, 0, len(c.Slices)+4)
	items = append(items,
		core.Padding(0),
		core.Gap(float64(t.Spacing.XS)),
		core.MinWidth(px(size)),
		core.AccessibilityHidden(),
	)
	for i, s := range c.Slices {
		items = append(items, core.Row(
			core.Padding(0),
			core.Gap(float64(t.Spacing.SM)),
			core.AlignItemsProp(core.AlignItemsCenter),
			swatch(colors[i]),
			core.Column(core.Padding(0), core.FlexGrow(1), core.FlexBasis("0"),
				core.Text(s.Label, core.FontSize(13), core.TextColor(t.Colors.TextPrimary))),
			core.Text(strconv.Itoa(pcts[i])+"%", core.FontSize(13), core.TextColor(t.Colors.TextSecondary)),
		))
	}
	return core.Column(items...)
}

// label is "Subject: Rent 1200, 73%; Food 450, 27%", followed by the centre
// text when there is some.
func (c DonutChart) label(values []float64, total float64) string {
	if c.AccessibilityLabel != "" {
		return c.AccessibilityLabel
	}
	format := c.Format
	if format == nil {
		format = formatValue
	}
	pcts := percentages(values, total)
	var parts []string
	for i, s := range c.Slices {
		if values[i] <= 0 {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s %s, %d%%", s.Label, format(values[i]), pcts[i]))
	}
	if len(parts) == 0 {
		parts = append(parts, "no data")
	}
	out := summaryPrefix(c.Subject) + joinSentences(parts)
	if !c.pie {
		if centre := strings.TrimSpace(c.CenterValue + " " + c.CenterLabel); centre != "" {
			out += " " + centre + "."
		}
	}
	return out
}

// sliceValues is each slice's drawable value (non-positive and NaN read as 0)
// and their total.
func sliceValues(slices []ChartSlice) ([]float64, float64) {
	values := make([]float64, len(slices))
	total := 0.0
	for i, s := range slices {
		v := s.Value
		if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
			v = 0
		}
		values[i] = v
		total += v
	}
	return values, total
}

// percentages rounds each value's share of total to a whole percent so the
// results add up to exactly 100, by the largest remainder method: floor every
// share, then give the missing points to the shares that lost the most in
// flooring. Ties go to the earlier slice, so the result is stable.
func percentages(values []float64, total float64) []int {
	out := make([]int, len(values))
	if total <= 0 {
		return out
	}
	type rem struct {
		i int
		r float64
	}
	rems := make([]rem, 0, len(values))
	sum := 0
	for i, v := range values {
		exact := v / total * 100
		out[i] = int(math.Floor(exact))
		sum += out[i]
		rems = append(rems, rem{i, exact - math.Floor(exact)})
	}
	sort.SliceStable(rems, func(a, b int) bool { return rems[a].r > rems[b].r })
	for k := 0; sum < 100 && k < len(rems); k++ {
		if values[rems[k].i] <= 0 {
			continue
		}
		out[rems[k].i]++
		sum++
	}
	return out
}
