package comps

import (
	"math"
	"strconv"
	"time"

	"github.com/rohanthewiz/grmob/core"
)

// Heatmap draws a grid of values as a grid of colours: a row per thing, a
// column per slot, and each cell painted by how much it holds.
//
//	comps.Heatmap{
//	    Subject:      "Orders by hour",
//	    RowLabels:    []string{"Mon", "Tue", "Wed"},
//	    ColumnLabels: []string{"9", "12", "15", "18"},
//	    Values: [][]float64{
//	        {2, 8, 5, 1},
//	        {3, 9, 7, 2},
//	        {1, 4, math.NaN(), 0},
//	    },
//	}
//
//	┌─────┬───────────────────────────────┐
//	│ Mon │ ░░░ ▓▓▓ ▒▒▒ ░░░               │  core.Canvas, CanvasStretch:
//	│ Tue │ ░░░ ███ ▓▓▓ ░░░               │  CellHeight px a row, the
//	│ Wed │ ░░░ ▒▒▒ ··· ░░░               │  width shared by the columns
//	│     │  9   12  15  18               │  column labels, equal slots
//	└─────┴───────────────────────────────┘
//	  0 ■■■■■ 9                            legend: the steps, low to high
//
// # The colours are the theme's Sequential role
//
// A heatmap paints a *quantity*, and the palette's categorical Chart role
// cannot do that: its slots are ordered to stay apart, not to read as more.
// So the cells take core.ColorPalette.SequentialColors — a checked run of
// lightness in one hue, least to most, added for this widget; the case
// against deriving one from Primary is written on that role. Colors
// overrides it per chart.
//
// # Steps, not a gradient
//
// The value range is cut into as many equal steps as the scale has colours
// (five on the bundled themes), and a cell takes its step's colour flat. A
// continuous blend would claim a precision the eye cannot read back from a
// colour, and steps are what a legend can key: five swatches say exactly
// which colours mean what, where a gradient bar can only say "towards here".
// One value everywhere is the top step — everything present is the most
// there is.
//
// # No data is not zero
//
// A NaN cell is "no data" and is painted in the theme's Surface, apart from
// every step (the scale's lightest entry was checked against each bundled
// Surface for exactly this pair). Zero is a value like any other and takes
// the step it falls in; a caller whose zeros mean "nothing happened" (a
// contribution calendar) passes NaN for them, which is what CalendarHeatmap
// does.
//
// # Drawn as one shape per colour
//
// Every cell of a step is a subpath of that step's one path, as BarChart
// draws a series, so a 7 × 53 calendar is six shapes rather than 371 and a
// cell changing step patches two paths. The cells are drawn with a gap a
// tenth of a cell wide, which is what makes a grid of equal colours read as
// cells rather than as a band.
//
// # Labels
//
// Row labels sit in a column of boxes each exactly CellHeight tall, so each
// centres on its row by construction. Column labels are the equal slots
// BarChart's categories use, with one addition: an empty label gives its slot
// to the label before it, so "Mar" over four unlabelled weeks has four
// columns of room and starts at its first — how a calendar names its months.
// A label with a slot of its own is centred on it.
//
// # One element, one sentence
//
// As every chart here, the grid is one RoleImg with a summary sentence and
// everything under it hidden: "Orders by hour: 3 rows by 4 columns; low 0
// at Wed, 18; high 9 at Tue, 12; 1 cell with no data."
//
// # Theme roles read
//
//	Cells      Colors.SequentialColors (or Colors), Colors.Surface for no data
//	Labels     Colors.TextSecondary at the charts' label size
type Heatmap struct {
	// Values are the cells, a row at a time; rows may differ in length, and
	// the grid is as wide as the longest. NaN is no data.
	Values [][]float64

	// RowLabels name the rows, drawn to their left. ColumnLabels name the
	// columns, drawn under them; an empty entry lends its slot to the label
	// before it (see "Labels"). Both feed the spoken summary.
	RowLabels, ColumnLabels []string

	// Subject says what the grid measures. It leads the spoken summary and is
	// not drawn.
	Subject string

	// CellHeight is each row's height in px; 0 means 20.
	CellHeight float64

	// Colors overrides the theme's sequential scale, least to most.
	Colors []string

	// Format writes a value in the legend and the summary.
	Format func(float64) string

	// Style is applied last, to the outer column.
	Style []core.StyleProp

	// AccessibilityLabel replaces the generated summary.
	AccessibilityLabel string
}

// heatCellGap is the share of a cell left empty between neighbours.
const heatCellGap = 0.1

func (h Heatmap) Render(ctx *core.Context) *core.Node {
	return heatGrid{
		values:     h.Values,
		rowLabels:  h.RowLabels,
		colLabels:  h.ColumnLabels,
		cellHeight: orDefaultFloat(h.CellHeight, 20),
		colors:     h.Colors,
		format:     h.Format,
		style:      h.Style,
		label:      h.AccessibilityLabel,
		summary: func(g heatGrid, format func(float64) string) string {
			return g.matrixSummary(h.Subject, format)
		},
	}.render(ctx)
}

// heatGrid is the drawing both heatmaps share. absent marks cells that are not
// drawn at all — a calendar's days after its last — as distinct from NaN, a
// cell drawn as "no data".
type heatGrid struct {
	values     [][]float64
	absent     [][]bool
	rowLabels  []string
	colLabels  []string
	cellHeight float64
	colors     []string
	format     func(float64) string
	style      []core.StyleProp
	label      string
	summary    func(g heatGrid, format func(float64) string) string
}

// dims is the grid's rows and columns.
func (g heatGrid) dims() (rows, cols int) {
	rows = len(g.values)
	for _, r := range g.values {
		cols = max(cols, len(r))
	}
	return rows, cols
}

// cell is the value at (i, j) and whether it is drawn at all.
func (g heatGrid) cell(i, j int) (v float64, drawn bool) {
	if i < len(g.absent) && j < len(g.absent[i]) && g.absent[i][j] {
		return 0, false
	}
	if j >= len(g.values[i]) {
		return math.NaN(), true
	}
	return g.values[i][j], true
}

// extent is the lowest and highest finite drawn value, and where each is;
// ok is false when no cell has one.
func (g heatGrid) extent() (lo, hi float64, loAt, hiAt [2]int, ok bool) {
	rows, cols := g.dims()
	for i := range rows {
		for j := range cols {
			v, drawn := g.cell(i, j)
			if !drawn || math.IsNaN(v) || math.IsInf(v, 0) {
				continue
			}
			if !ok || v < lo {
				lo, loAt = v, [2]int{i, j}
			}
			if !ok || v > hi {
				hi, hiAt = v, [2]int{i, j}
			}
			ok = true
		}
	}
	return lo, hi, loAt, hiAt, ok
}

// heatStep is the scale step of v among k: equal slices of [lo, hi], the top
// value in the top step, and a flat range entirely in it.
func heatStep(v, lo, hi float64, k int) int {
	if hi <= lo {
		return k - 1
	}
	return min(k-1, max(0, int((v-lo)/(hi-lo)*float64(k))))
}

func (g heatGrid) render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	scale := g.colors
	if len(scale) == 0 {
		scale = t.Colors.SequentialColors()
	}
	format := g.format
	if format == nil {
		format = formatValue
	}
	rows, cols := g.dims()
	lo, hi, _, _, hasData := g.extent()

	// One path per step, and one for no data, each holding its cells as
	// closed subpaths. See "Drawn as one shape per colour".
	paths := make([]*core.Path, len(scale)+1) // [0] is no data
	cw, ch := chartView/float64(max(1, cols)), chartView/float64(max(1, rows))
	gx, gy := cw*heatCellGap, ch*heatCellGap
	for i := range rows {
		for j := range cols {
			v, drawn := g.cell(i, j)
			if !drawn {
				continue
			}
			slot := 0
			if hasData && !math.IsNaN(v) && !math.IsInf(v, 0) {
				slot = 1 + heatStep(v, lo, hi, len(scale))
			}
			if paths[slot] == nil {
				paths[slot] = core.NewPath()
			}
			x, y := float64(j)*cw+gx/2, float64(i)*ch+gy/2
			paths[slot].MoveTo(x, y).LineTo(x+cw-gx, y).LineTo(x+cw-gx, y+ch-gy).LineTo(x, y+ch-gy).Close()
		}
	}
	// Every slot keeps its place in the list, a nil path included, so a step
	// that empties and refills patches its own child rather than shifting
	// the rest (core.Shape: a shape with nothing to draw holds its slot).
	shapes := make([]core.Shape, 0, len(paths))
	for s, p := range paths {
		fill := t.Colors.Surface
		if s > 0 {
			fill = scale[s-1]
		}
		if p == nil {
			shapes = append(shapes, core.Shape{})
			continue
		}
		shapes = append(shapes, core.Shape{Path: p, Fill: fill})
	}
	height := g.cellHeight * float64(max(1, rows))
	canvas := core.Canvas(chartView, chartView, shapes, core.CanvasStretch, core.Height(px(height)))

	plot := []core.PropsAndChildren{
		core.Padding(0),
		core.Gap(float64(t.Spacing.XS)),
		core.FlexGrow(1),
		core.FlexBasis("0"),
		core.MinWidth("0px"),
		canvas,
	}
	if cl := spanLabels(t, g.colLabels, cols); cl != nil {
		plot = append(plot, cl)
	}

	grid := []core.PropsAndChildren{core.Padding(0), core.Gap(6), core.AlignItemsProp(core.AlignItemsStart)}
	if len(g.rowLabels) > 0 {
		grid = append(grid, g.rowAxis(t, rows))
	}
	grid = append(grid, core.Column(plot...))

	label := g.label
	if label == "" {
		label = g.summary(g, format)
	}
	items := make([]core.PropsAndChildren, 0, len(g.style)+6)
	items = append(items,
		core.Padding(0),
		core.Gap(float64(t.Spacing.SM)),
		core.AccessibilityRole(core.RoleImg),
		core.AccessibilityLabel(label),
	)
	items = append(items, asProps(g.style)...)
	items = append(items, core.Row(grid...))
	if hasData {
		items = append(items, heatLegend(t, scale, format(lo), format(hi)))
	}
	return core.Column(items...).Render(ctx)
}

// rowAxis is the row label column: a box exactly one row tall per row, so
// every label centres on its row with no measurement.
func (g heatGrid) rowAxis(t *core.Theme, rows int) core.View {
	items := []core.PropsAndChildren{core.Padding(0), core.Gap(0), core.AccessibilityHidden()}
	for i := range rows {
		text := ""
		if i < len(g.rowLabels) {
			text = g.rowLabels[i]
		}
		items = append(items, core.Row(
			core.Padding(0),
			core.Height(px(g.cellHeight)),
			core.AlignItemsProp(core.AlignItemsCenter),
			axisText(t, text, core.AlignEnd),
		))
	}
	return core.Column(items...)
}

// spanLabels is a column label row of n equal slots in which each label also
// owns the empty slots after it (see "Labels" on Heatmap). A label alone in
// its slot is centred on it; one spanning several starts at its first.
func spanLabels(t *core.Theme, labels []string, n int) core.View {
	if len(labels) == 0 || n == 0 {
		return nil
	}
	items := []core.PropsAndChildren{core.Padding(0), core.Gap(0), core.AccessibilityHidden()}
	for i := 0; i < n; {
		text := ""
		if i < len(labels) {
			text = labels[i]
		}
		span := 1
		for i+span < n && (i+span >= len(labels) || labels[i+span] == "") {
			span++
		}
		if text == "" {
			// Leading slots before the first label: an empty filler as wide
			// as they are.
			span = 1
		}
		align := core.AlignCenter
		if span > 1 {
			align = core.AlignStart
		}
		items = append(items, weightedLabel(t, text, float64(span), align))
		i += span
	}
	return core.Row(items...)
}

// heatLegend is the key under the grid: the low value, a swatch per step, the
// high value. Hidden, like every chart legend; the summary says the numbers.
func heatLegend(t *core.Theme, scale []string, lo, hi string) core.View {
	items := []core.PropsAndChildren{
		core.Padding(0),
		core.Gap(3),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.AccessibilityHidden(),
		axisText(t, lo, core.AlignEnd),
	}
	for _, c := range scale {
		items = append(items, swatch(c))
	}
	items = append(items, axisText(t, hi, core.AlignStart))
	return core.Row(items...)
}

// matrixSummary is Heatmap's sentence: the grid's shape, where its low and
// high are, and how many cells hold nothing.
func (g heatGrid) matrixSummary(subject string, format func(float64) string) string {
	rows, cols := g.dims()
	parts := []string{summaryPrefix(subject) + plural2(rows, "row") + " by " + plural2(cols, "column")}
	lo, hi, loAt, hiAt, ok := g.extent()
	if !ok {
		parts = append(parts, "no data")
		return joinSentences(parts)
	}
	parts = append(parts,
		"low "+format(lo)+" at "+g.cellName(loAt),
		"high "+format(hi)+" at "+g.cellName(hiAt),
	)
	empty := 0
	for i := range rows {
		for j := range cols {
			if v, drawn := g.cell(i, j); drawn && math.IsNaN(v) {
				empty++
			}
		}
	}
	if empty > 0 {
		parts = append(parts, plural2(empty, "cell")+" with no data")
	}
	return joinSentences(parts)
}

// cellName says where a cell is, in the labels' words when there are any:
// "Tue, 12", else "row 2, column 3".
func (g heatGrid) cellName(at [2]int) string {
	name := func(labels []string, i int, word string) string {
		if i < len(labels) && labels[i] != "" {
			return labels[i]
		}
		return word + " " + strconv.Itoa(i+1)
	}
	return name(g.rowLabels, at[0], "row") + ", " + name(g.colLabels, at[1], "column")
}

// plural2 is "1 row" / "3 rows".
func plural2(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return strconv.Itoa(n) + " " + word + "s"
}

// orDefaultFloat returns v, or def when v is zero or less.
func orDefaultFloat(v, def float64) float64 {
	if v <= 0 {
		return def
	}
	return v
}

// CalendarHeatmap is the contribution calendar: a column per week, a row per
// weekday, and each day painted by its count — commits, workouts, words
// written.
//
//	comps.CalendarHeatmap{
//	    Subject: "Workouts",
//	    Days:    workouts,          // []comps.DayValue
//	    End:     today,
//	}
//
//	      Jan         Feb         Mar
//	Mon  ░ ▒ ░ · ▓ ░ ▒ ░ · ░ ▓ █ ░ ▒ ░ ▒ ░
//	Wed  ▒ ░ · ░ ░ ▓ ░ · ▒ ░ ░ ▒ ▓ ░ ░ ·
//	Fri  ░ · ▒ ░ ▒ ░ · ░ ░ ▒ ░ ░ ▒ ░ ▓
//	      0 ■■■■■ 4
//
// It is a Heatmap — the same grid, scale, legend and one-sentence summary —
// with the calendar arithmetic done here:
//
//   - The last column is the week holding End, and the grid runs Weeks
//     columns back from it, each week starting on WeekStart (Sunday, the zero
//     value, as Calendar's does).
//   - Days after End are not drawn at all; they have not happened, which is
//     different from having happened with nothing in them.
//   - A day with no entries, or entries summing to zero, is "no data" and
//     painted Surface — the contribution calendar's convention, where an
//     empty day is the ground and every step is some activity. See "No data
//     is not zero" on Heatmap for why this is a choice made here rather than
//     there.
//   - Days are matched by calendar date in End's location, and several
//     entries on one day are summed.
//   - Rows are labelled on alternate weekdays (Mon, Wed, Fri under a Sunday
//     start), and a column is labelled with its month's name when it holds the
//     1st — but not the first column, if the next month's name is fewer than
//     three columns away, where the two would collide.
//
// The summary is the calendar's own: "Workouts: 42 over 17 weeks, on 23 days;
// most on Tue 3 Mar 2026, 4."
type CalendarHeatmap struct {
	// Days are the dated values. Several on one date are summed.
	Days []DayValue

	// End is the last day drawn; zero means today. Its location decides which
	// date each entry falls on.
	End time.Time

	// Weeks is how many week columns are drawn; 0 means 17, about four months,
	// which fits a phone at a legible cell width.
	Weeks int

	// WeekStart is the first row's weekday; the zero value is Sunday.
	WeekStart time.Weekday

	// Subject says what is counted. It leads the spoken summary.
	Subject string

	// CellHeight is each weekday row's height in px; 0 means 14.
	CellHeight float64

	// Colors overrides the theme's sequential scale, least to most.
	Colors []string

	// Format writes a value in the legend and the summary.
	Format func(float64) string

	// Style is applied last, to the outer column.
	Style []core.StyleProp

	// AccessibilityLabel replaces the generated summary.
	AccessibilityLabel string
}

// DayValue is one dated value of a CalendarHeatmap.
type DayValue struct {
	Day   time.Time
	Value float64
}

// dayKey is a date with the clock and location stripped, for matching.
type dayKey struct {
	y int
	m time.Month
	d int
}

func keyOf(t time.Time) dayKey {
	y, m, d := t.Date()
	return dayKey{y, m, d}
}

func (c CalendarHeatmap) Render(ctx *core.Context) *core.Node {
	end := c.End
	if end.IsZero() {
		end = time.Now()
	}
	loc := end.Location()
	weeks := c.Weeks
	if weeks <= 0 {
		weeks = 17
	}

	sums := map[dayKey]float64{}
	for _, dv := range c.Days {
		if math.IsNaN(dv.Value) || math.IsInf(dv.Value, 0) {
			continue
		}
		sums[keyOf(dv.Day.In(loc))] += dv.Value
	}

	// The first day drawn: back from End to its week's start, then back the
	// remaining whole weeks. Built with time.Date so a daylight-saving change
	// inside the range moves no day across midnight.
	endDay := time.Date(end.Year(), end.Month(), end.Day(), 12, 0, 0, 0, loc)
	back := (int(endDay.Weekday()) - int(c.WeekStart) + 7) % 7
	first := endDay.AddDate(0, 0, -back-7*(weeks-1))

	values := make([][]float64, 7)
	absent := make([][]bool, 7)
	for r := range 7 {
		values[r] = make([]float64, weeks)
		absent[r] = make([]bool, weeks)
	}
	colLabels := make([]string, weeks)
	var total float64
	active := 0
	var best time.Time
	bestV := math.Inf(-1)
	for w := range weeks {
		for r := range 7 {
			day := first.AddDate(0, 0, 7*w+r)
			if day.Day() == 1 {
				colLabels[w] = day.Month().String()[:3]
			}
			if day.After(endDay) {
				absent[r][w] = true
				continue
			}
			v := sums[keyOf(day)]
			if v == 0 {
				values[r][w] = math.NaN()
				continue
			}
			values[r][w] = v
			total += v
			active++
			if v > bestV {
				best, bestV = day, v
			}
		}
	}
	// The first column is named for the month it opens in, unless the next
	// name is too close for both to be read.
	if colLabels[0] == "" {
		next := weeks
		for w := 1; w < weeks; w++ {
			if colLabels[w] != "" {
				next = w
				break
			}
		}
		if next >= 3 {
			colLabels[0] = first.Month().String()[:3]
		}
	}

	rowLabels := make([]string, 7)
	for r := 1; r < 7; r += 2 {
		rowLabels[r] = time.Weekday((int(c.WeekStart) + r) % 7).String()[:3]
	}

	return heatGrid{
		values:     values,
		absent:     absent,
		rowLabels:  rowLabels,
		colLabels:  colLabels,
		cellHeight: orDefaultFloat(c.CellHeight, 14),
		colors:     c.Colors,
		format:     c.Format,
		style:      c.Style,
		label:      c.AccessibilityLabel,
		summary: func(_ heatGrid, format func(float64) string) string {
			parts := []string{summaryPrefix(c.Subject) + format(total) + " over " + plural2(weeks, "week")}
			if active == 0 {
				parts[0] += ", none on any day"
				return joinSentences(parts)
			}
			parts[0] += ", on " + plural2(active, "day")
			parts = append(parts, "most on "+best.Format("Mon 2 Jan 2006")+", "+format(bestV))
			return joinSentences(parts)
		},
	}.render(ctx)
}
