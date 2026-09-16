package comps

import (
	"math"
	"strconv"
	"strings"

	"github.com/rohanthewiz/grmob/core"
)

// This file holds what the chart widgets share: the series type, the value
// scale and its "nice" ticks, the palette drawn from theme roles, and the axis
// and legend pieces laid out as Text around a core.Canvas.
//
// # How a chart is put together
//
// A cartesian chart (LineChart, AreaChart, BarChart) is three kinds of node:
//
//	┌────┬──────────────────────────────┐
//	│ 40 │  ── spacer, half a label ──  │
//	│    │┌────────────────────────────┐│
//	│ 20 │├────── gridline ────────────┤│  core.Canvas, CanvasStretch,
//	│    ││    ╱╲      ╱               ││  viewBox 100 × 100, Height H px
//	│  0 │└────────────────────────────┘│
//	│    │  ── spacer, half a label ──  │
//	│    │ Jan      Apr       Jul   Oct │  x labels, a Row of weighted boxes
//	└────┴──────────────────────────────┘
//	  y labels: a Column of fixed-height boxes spaced by an exact Gap
//
// Text stays out of the canvas (core.Canvas's v1 has none, and platform text
// in a drawing is where the targets disagree most), so every label is placed
// by flex arithmetic that puts its centre on the value it names. That needs
// no measurement of the drawn box, which no target reports back to Go.
//
// # Why the y labels line up
//
// The domain is widened to whole ticks (niceScale), so the N+1 labels are
// evenly spaced from the canvas's top edge to its bottom edge. Each label sits
// in a box one label-line (lh) tall, and the column is H + lh tall with a
// Gap of (H − N·lh)/N, so label i's box starts at i·H/N and its centre at
// i·H/N + lh/2. The canvas is pushed down by a spacer lh/2 tall, which puts
// gridline i at the same i·H/N + lh/2. No negative margins, which not every
// target honours the same way.
//
// # One element, one sentence
//
// A chart is announced once, as RoleImg with a summary sentence, and every
// label and legend entry under it is hidden — the same shape DigitalClock and
// Compass take. A reader walking twenty tick labels learns less than
// "Visits: 12 points, from 12 in Jan to 45 in Dec; low 12, high 45".

// ChartSeries is one named run of values: a line, an area, or one colour of
// bars in a grouped BarChart.
type ChartSeries struct {
	// Name labels the series in the legend (drawn when a chart has more than
	// one series) and in the spoken summary.
	Name string

	// Values are the y values in x order. NaN is a missing value: a line or
	// area breaks around it, and a bar is not drawn.
	Values []float64

	// Color overrides the series' palette colour. A "#rrggbb" hex is best:
	// an area's translucent fill is made by appending an alpha byte to it.
	Color string
}

// Geometry shared by the cartesian charts.
const (
	// chartView is the side of the square viewBox every chart draws in. With
	// CanvasStretch the square becomes whatever box the layout gives, and 100
	// keeps coordinates at two decimal places on the wire.
	chartView = 100.0

	// chartHeight is a cartesian chart's plot height in px when unset.
	chartHeight = 160.0

	// chartLabelSize and chartLabelLine are the axis labels' font size and the
	// height of the box each sits in. The box is a little taller than the
	// text's line so descenders never collide with the next label, and it is
	// what the tick spacing arithmetic is written in.
	chartLabelSize = 11.0
	chartLabelLine = 14.0

	// chartMaxXLabels caps the labels under a chart. Beyond about five a
	// phone-width axis crowds, and a label wider than its slot would widen it
	// and pull its neighbours off their points.
	chartMaxXLabels = 5
)

// chartPalette returns the series colours a theme offers, in order. They are
// theme roles so a theme swap recolours every chart, and so a dark theme's
// roles (which are tuned for its background) come along for free.
//
// Primary, Secondary, Warning, Success and Error are the hue-carrying roles
// every bundled theme sets. Duplicates are dropped — DefaultTheme gives
// Secondary and Success the same green — because two series in one colour are
// worse than a shorter cycle. Error is last because a red series reads as a
// problem, so it is reached only by a chart with four or more series.
//
// Past the roles the cycle repeats each colour at 60% alpha: a tint of the
// same hue over the page, which stays distinct from its solid twin and never
// invents a hue the theme did not choose. A role that already carries alpha
// (or is not "#rrggbb") repeats unchanged.
func chartPalette(t *core.Theme) []string {
	roles := []string{
		t.Colors.Primary,
		t.Colors.Secondary,
		t.Colors.WarningColor(),
		t.Colors.SuccessColor(),
		t.Colors.Error,
	}
	seen := map[string]bool{}
	base := make([]string, 0, len(roles))
	for _, c := range roles {
		k := strings.ToLower(c)
		if c == "" || seen[k] {
			continue
		}
		seen[k] = true
		base = append(base, c)
	}
	out := append([]string(nil), base...)
	for _, c := range base {
		out = append(out, withAlpha(c, "99"))
	}
	return out
}

// seriesColor is the colour of series i: its own Color, else the palette's.
func seriesColor(palette []string, own string, i int) string {
	if own != "" {
		return own
	}
	if len(palette) == 0 {
		return "#888888"
	}
	return palette[i%len(palette)]
}

// withAlpha appends an alpha byte to a "#rrggbb" or "#rgb" colour. Any other
// form is returned unchanged, because appending to a colour that already has
// alpha, or to a name, would produce something no target parses.
func withAlpha(c, aa string) string {
	if len(c) == 4 && c[0] == '#' {
		c = "#" + strings.Repeat(c[1:2], 2) + strings.Repeat(c[2:3], 2) + strings.Repeat(c[3:4], 2)
	}
	if len(c) == 7 && c[0] == '#' {
		return c + aa
	}
	return c
}

// valueScale maps data values onto the viewBox's y axis, with 0 units at the
// top (hi) and chartView at the bottom (lo), as screen coordinates run.
type valueScale struct {
	lo, hi, step float64
}

// y is the viewBox y of value v.
func (s valueScale) y(v float64) float64 {
	return chartView - (v-s.lo)/(s.hi-s.lo)*chartView
}

// ticks returns the tick values from lo to hi inclusive, each rounded to the
// step's precision so accumulated float error never prints as 0.30000000004.
func (s valueScale) ticks() []float64 {
	n := int(math.Round((s.hi - s.lo) / s.step))
	d := stepDecimals(s.step)
	out := make([]float64, 0, n+1)
	for i := 0; i <= n; i++ {
		out = append(out, roundTo(s.lo+float64(i)*s.step, d))
	}
	return out
}

// base is where bars and areas grow from: zero when zero is in the domain,
// otherwise the domain edge nearest to it.
func (s valueScale) base() float64 {
	return math.Max(s.lo, math.Min(s.hi, 0))
}

// niceScale widens [lo, hi] to whole multiples of a 1, 2 or 5 × 10ⁿ step, with
// at most maxTicks ticks. This is Heckbert's "nice numbers for graph labels"
// (Graphics Gems, 1990): round the range to a nice number, divide it into
// maxTicks−1 intervals, round the interval to a nice number, then snap the ends
// outwards onto it. Axis labels come out as 0, 20, 40 rather than 0, 17.3,
// 34.6, and the data always lies within the ticks.
//
// zero forces 0 into the domain, which bars and areas need so that their
// lengths are proportional to their values. An empty or single-valued range is
// widened first, since a zero range has no scale.
func niceScale(lo, hi float64, maxTicks int, zero bool) valueScale {
	if math.IsInf(lo, 0) || math.IsInf(hi, 0) || math.IsNaN(lo) || math.IsNaN(hi) || lo > hi {
		lo, hi = 0, 1
	}
	if zero {
		lo, hi = math.Min(lo, 0), math.Max(hi, 0)
	}
	if lo == hi {
		// A flat series sits mid-chart rather than on an edge. Ten percent
		// either side of a non-zero value, or 0..1 for all zeros.
		pad := math.Abs(lo) * 0.1
		if pad == 0 {
			pad = 1
		}
		if zero && lo == 0 {
			hi = pad
		} else {
			lo, hi = lo-pad, hi+pad
		}
		if zero {
			lo, hi = math.Min(lo, 0), math.Max(hi, 0)
		}
	}
	maxTicks = max(maxTicks, 2)
	span := niceNum(hi-lo, false)
	step := niceNum(span/float64(maxTicks-1), true)
	s := valueScale{
		lo:   math.Floor(lo/step) * step,
		hi:   math.Ceil(hi/step) * step,
		step: step,
	}
	// Rounding to nice numbers can overshoot the tick budget by one or two
	// (a range of 0..95 with 5 ticks rounds to step 20 and ends at 100 — six
	// ticks). Doubling the step until it fits keeps the labels from crowding.
	for int(math.Round((s.hi-s.lo)/s.step))+1 > maxTicks {
		s.step = niceNum(s.step*2, true)
		s.lo = math.Floor(lo/s.step) * s.step
		s.hi = math.Ceil(hi/s.step) * s.step
	}
	return s
}

// niceNum returns a 1, 2, 5 or 10 × 10ⁿ near x. round picks the nearest one;
// otherwise the smallest one at or above x.
func niceNum(x float64, round bool) float64 {
	if x <= 0 {
		return 1
	}
	exp := math.Floor(math.Log10(x))
	f := x / math.Pow(10, exp)
	var nf float64
	if round {
		switch {
		case f < 1.5:
			nf = 1
		case f < 3:
			nf = 2
		case f < 7:
			nf = 5
		default:
			nf = 10
		}
	} else {
		switch {
		case f <= 1:
			nf = 1
		case f <= 2:
			nf = 2
		case f <= 5:
			nf = 5
		default:
			nf = 10
		}
	}
	return nf * math.Pow(10, exp)
}

// stepDecimals is how many decimal places distinguish ticks step apart: none
// for a step of 5 or 20, one for 0.5, two for 0.25 (which niceNum never makes,
// but a caller's own step might).
func stepDecimals(step float64) int {
	if step <= 0 {
		return 0
	}
	d := 0
	for d < 10 && math.Abs(step*math.Pow(10, float64(d))-math.Round(step*math.Pow(10, float64(d)))) > 1e-9 {
		d++
	}
	return d
}

func roundTo(v float64, places int) float64 {
	p := math.Pow(10, float64(places))
	return math.Round(v*p) / p
}

// formatTick writes a tick value with exactly the decimals its step needs, so
// an axis reads 0.5, 1.0, 1.5 rather than 0.5, 1, 1.5.
//
// Large axes are shortened to k or M, chosen once from extent (the largest
// tick's magnitude) so every label on an axis shares a unit: 0, 500k, 1.0M
// would make a reader compare across units. The thresholds are 10 000 and
// 1 000 000 because "2500" is as short as "2.5k" and reads more plainly, while
// a y axis column wide enough for "1500000" takes a third of a phone's chart.
func formatTick(v, step, extent float64) string {
	unit, suffix := 1.0, ""
	switch {
	case extent >= 1e6:
		unit, suffix = 1e6, "M"
	case extent >= 1e4:
		unit, suffix = 1e3, "k"
	}
	if v == 0 {
		return "0"
	}
	return trimFloat(v/unit, stepDecimals(step/unit)) + suffix
}

// trimFloat formats v with places decimals, and writes negative zero as "0".
func trimFloat(v float64, places int) string {
	s := strconv.FormatFloat(roundTo(v, places), 'f', places, 64)
	if strings.Trim(s, "-0.") == "" {
		return strconv.FormatFloat(0, 'f', places, 64)
	}
	return s
}

// formatValue is the default spoken form of a data value: up to two decimals,
// with trailing zeros dropped (12, 12.5, 12.34).
func formatValue(v float64) string {
	return strconv.FormatFloat(roundTo(v, 2), 'f', -1, 64)
}

// dataRange is the smallest and largest non-NaN value across series. ok is
// false when there is no value at all.
func dataRange(series []ChartSeries) (lo, hi float64, ok bool) {
	lo, hi = math.Inf(1), math.Inf(-1)
	for _, s := range series {
		for _, v := range s.Values {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				continue
			}
			lo, hi, ok = math.Min(lo, v), math.Max(hi, v), true
		}
	}
	return lo, hi, ok
}

// chartMaxTicks is how many y ticks fit a plot h px tall: one label line of
// clear space between neighbours, and never more than six, past which the
// gridlines stop helping and start competing with the data.
func chartMaxTicks(h float64) int {
	return max(2, min(6, int(h/(chartLabelLine*2))+1))
}

// axisText is one hidden axis or legend label.
func axisText(t *core.Theme, s string, align core.Alignment) core.View {
	return core.Text(s,
		core.FontSize(chartLabelSize),
		core.TextColor(t.Colors.TextSecondary),
		core.Align(align),
		core.AccessibilityHidden(),
	)
}

// yAxis is the tick label column beside a plot h px tall. See "Why the y
// labels line up" at the top of this file for the arithmetic.
func yAxis(t *core.Theme, s valueScale, h float64, format func(float64) string) core.View {
	ticks := s.ticks()
	intervals := max(1, len(ticks)-1)
	items := make([]core.PropsAndChildren, 0, len(ticks)+6)
	items = append(items,
		core.Padding(0),
		core.Height(px(h+chartLabelLine)),
		core.Gap((h-float64(intervals)*chartLabelLine)/float64(intervals)),
		core.AlignItemsProp(core.AlignItemsEnd),
		core.AccessibilityHidden(),
	)
	// Top to bottom is high to low.
	for i := len(ticks) - 1; i >= 0; i-- {
		label := formatTick(ticks[i], s.step, math.Max(math.Abs(s.lo), math.Abs(s.hi)))
		if format != nil {
			label = format(ticks[i])
		}
		items = append(items, core.Column(
			core.Padding(0),
			core.Height(px(chartLabelLine)),
			core.Justify(core.JustifyCenter),
			axisText(t, label, core.AlignEnd),
		))
	}
	return core.Column(items...)
}

// gridShapes are the horizontal rules at each tick, drawn under the data.
// Hairlines in the theme's divider role: they are reading aids, not data.
func gridShapes(t *core.Theme, s valueScale) []core.Shape {
	ticks := s.ticks()
	out := make([]core.Shape, 0, len(ticks))
	for _, v := range ticks {
		y := s.y(v)
		out = append(out, core.Shape{
			Path:        core.Line(0, y, chartView, y),
			Stroke:      t.Colors.BorderColor(),
			StrokeWidth: 1,
		})
	}
	return out
}

// weightedLabel is one label box in an x-axis row: FlexGrow(weight) with a zero
// basis, so the row is divided in proportion to the weights on every target
// (see StatTile.Fill for why both props are needed).
func weightedLabel(t *core.Theme, text string, weight float64, align core.Alignment) core.View {
	return core.Column(
		core.Padding(0),
		core.FlexGrow(weight),
		core.FlexBasis("0"),
		axisText(t, text, align),
	)
}

// pointLabels is the x label row under a line or area chart, whose n points sit
// at x = i/(n−1) of the width, edge to edge.
//
// A label is centred on its point by giving it a box that extends equally
// either side of the point. With labels every s points, interior boxes span
// [i − s/2, i + s/2], which tile the row exactly. The two end points cannot
// have symmetric boxes without leaving the row, so the first label is aligned
// to the start edge and a label on the last point to the end edge — the usual
// convention for a time axis. A final labelled point that is not the last
// point keeps a centred box only if that box fits; the rest of the row is an
// empty filler.
//
//	points   0 . . 3 . . 6 . . 9 . 11          s = 3
//	boxes   [0 ][  3  ][  6  ][  9  ][ ]       weights 1.5, 3, 3, 3, 0.5
//	         ↑start     centred       filler
func pointLabels(t *core.Theme, labels []string, n int) core.View {
	if len(labels) == 0 || n == 0 {
		return nil
	}
	label := func(i int) string {
		if i < len(labels) {
			return labels[i]
		}
		return ""
	}
	items := []core.PropsAndChildren{core.Padding(0), core.Gap(0), core.AccessibilityHidden()}
	if n == 1 {
		items = append(items, weightedLabel(t, label(0), 1, core.AlignCenter))
		return core.Row(items...)
	}

	last := n - 1
	s := (last + chartMaxXLabels - 2) / (chartMaxXLabels - 1) // ceil(last / (max−1))
	s = max(s, 1)
	half := float64(s) / 2

	items = append(items, weightedLabel(t, label(0), half, core.AlignStart))
	used := half
	for i := s; i <= last; i += s {
		if i == last {
			items = append(items, weightedLabel(t, label(i), half, core.AlignEnd))
			used += half
			break
		}
		if float64(i)+half > float64(last) {
			break
		}
		items = append(items, weightedLabel(t, label(i), float64(s), core.AlignCenter))
		used += float64(s)
	}
	if rest := float64(last) - used; rest > 1e-9 {
		items = append(items, weightedLabel(t, "", rest, core.AlignCenter))
	}
	return core.Row(items...)
}

// bandLabels is the x label row under a bar chart, whose n categories each own
// an equal slot with the bars centred in it. Equal boxes line the labels up
// exactly; past chartMaxXLabels only every s-th box has text, but every box is
// kept so the rest stay on their bars.
func bandLabels(t *core.Theme, labels []string, n int) core.View {
	if len(labels) == 0 || n == 0 {
		return nil
	}
	s := max(1, (n+chartMaxXLabels-1)/chartMaxXLabels)
	items := make([]core.PropsAndChildren, 0, n+3)
	items = append(items, core.Padding(0), core.Gap(0), core.AccessibilityHidden())
	for i := range n {
		text := ""
		if i%s == 0 && i < len(labels) {
			text = labels[i]
		}
		items = append(items, weightedLabel(t, text, 1, core.AlignCenter))
	}
	return core.Row(items...)
}

// legend is a wrapping row of colour swatches and names, hidden like the axes.
func legend(t *core.Theme, names, colors []string) core.View {
	items := make([]core.PropsAndChildren, 0, len(names)+5)
	items = append(items,
		core.Padding(0),
		core.Gap(float64(t.Spacing.SM)+4),
		core.RowGap(float64(t.Spacing.XS)),
		core.FlexWrap(true),
		core.AccessibilityHidden(),
	)
	for i, name := range names {
		items = append(items, core.Row(
			core.Padding(0),
			core.Gap(float64(t.Spacing.XS)+2),
			core.AlignItemsProp(core.AlignItemsCenter),
			swatch(colors[i]),
			axisText(t, name, core.AlignStart),
		))
	}
	return core.Row(items...)
}

// swatch is the small rounded square that keys a colour to a name.
func swatch(color string) core.View {
	return core.Box(
		core.Padding(0),
		core.Width("10px"),
		core.Height("10px"),
		core.BorderRadius(2),
		core.BackgroundColor(color),
	)
}

// cartesianFrame assembles the y axis, the canvas and the x labels (see the
// diagram at the top of this file) and the legend under them, as one element
// announcing label. xLabels may be nil.
func cartesianFrame(ctx *core.Context, s valueScale, h float64, format func(float64) string,
	canvas core.View, xLabels core.View, legendView core.View, label string, style []core.StyleProp) *core.Node {
	t := ctx.Theme()

	plot := []core.PropsAndChildren{
		core.Padding(0),
		core.Gap(0),
		core.FlexGrow(1),
		core.FlexBasis("0"),
		// Half a label line above and below, so the top and bottom gridlines
		// meet the centres of the top and bottom labels.
		core.Box(core.Padding(0), core.Height(px(chartLabelLine/2))),
		canvas,
		core.Box(core.Padding(0), core.Height(px(chartLabelLine/2))),
	}
	if xLabels != nil {
		plot = append(plot, xLabels)
	}

	items := make([]core.PropsAndChildren, 0, 8+len(style))
	items = append(items,
		core.Padding(0),
		core.Gap(float64(t.Spacing.SM)),
		core.AccessibilityRole(core.RoleImg),
		core.AccessibilityLabel(label),
	)
	for _, sp := range style {
		items = append(items, sp)
	}
	items = append(items, core.Row(
		core.Padding(0),
		core.Gap(6),
		core.AlignItemsProp(core.AlignItemsStart),
		yAxis(t, s, h, format),
		core.Column(plot...),
	))
	if legendView != nil {
		items = append(items, legendView)
	}
	return core.Column(items...).Render(ctx)
}

// summaryPrefix starts a spoken summary with the chart's subject, when given.
func summaryPrefix(subject string) string {
	if subject == "" {
		return ""
	}
	return subject + ": "
}

// joinSentences joins summary clauses with "; " and ends with a full stop.
func joinSentences(parts []string) string {
	s := strings.Join(parts, "; ")
	if s != "" && !strings.HasSuffix(s, ".") {
		s += "."
	}
	return s
}
