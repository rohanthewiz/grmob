# Package comps — Charts

```go
import "github.com/rohanthewiz/grmob/comps"
```

Sparklines, line, area, bar and scatter charts, donuts and pies, and gauges, drawn on core.Canvas.

One of 7 topic pages of [package comps](comps.md), which has the package overview and an index of every topic. This page documents the declarations in `comps/chart.go`, `comps/sparkline.go`, `comps/line_chart.go`, `comps/bar_chart.go`, `comps/scatter_chart.go`, `comps/donut_chart.go`, `comps/gauge.go`.

## Index

- [`type AreaChart`](#type-areachart)
    - [`func (AreaChart) Render`](#func-areachart-render)
- [`type BarChart`](#type-barchart)
    - [`func (BarChart) Render`](#func-barchart-render)
- [`type ChartPoint`](#type-chartpoint)
- [`type ChartSeries`](#type-chartseries)
- [`type ChartSlice`](#type-chartslice)
- [`type DonutChart`](#type-donutchart)
    - [`func (DonutChart) Render`](#func-donutchart-render)
- [`type Gauge`](#type-gauge)
    - [`func (Gauge) Render`](#func-gauge-render)
- [`type LineChart`](#type-linechart)
    - [`func (LineChart) Render`](#func-linechart-render)
- [`type PieChart`](#type-piechart)
    - [`func (PieChart) Render`](#func-piechart-render)
- [`type ScatterChart`](#type-scatterchart)
    - [`func (ScatterChart) Render`](#func-scatterchart-render)
- [`type ScatterSeries`](#type-scatterseries)
- [`type Sparkline`](#type-sparkline)
    - [`func (Sparkline) Render`](#func-sparkline-render)

## Types

### type AreaChart

```go
type AreaChart LineChart
```

AreaChart is a LineChart with Area set: each series filled down to zero. Several series overlap translucently rather than stacking.

<small>[comps/line_chart.go:132](https://github.com/rohanthewiz/grmob/blob/master/comps/line_chart.go#L132)</small>

#### func (AreaChart) Render

```go
func (a AreaChart) Render(ctx *core.Context) *core.Node
```

<small>[comps/line_chart.go:134](https://github.com/rohanthewiz/grmob/blob/master/comps/line_chart.go#L134)</small>

### type BarChart

```go
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
```

BarChart draws a value per category as a vertical bar, with several series grouped side by side within each category.

	comps.BarChart{
	    Subject: "Steps",
	    Labels:  []string{"Mon", "Tue", "Wed"},
	    Series:  []comps.ChartSeries{{Name: "Steps", Values: []float64{8200, 10400, 6100}}},
	}

#### Slots

The width is divided into one equal slot per category, and a category's bars share the middle of its slot:

	│  slot 0   │  slot 1   │  slot 2   │
	│   ▐█▌▐█▌  │   ▐█▌▐█▌  │   ▐█▌▐█▌  │    group = BarFill of the slot
	│   Mon     │   Tue     │   Wed     │    labels in equal boxes, centred

Equal slots are what make the labels exact: the label row is the same number of equal flex boxes, so each label's centre is its slot's centre on every target, without Go knowing how wide the chart is drawn.

#### Zero and negatives

The axis always includes zero, and bars grow from it — up for positive values, down for negative ones. A bar's length is only proportional to its value when it starts at zero, which is why this is not optional as it is on LineChart.

#### Stacked

Stacked puts a category's series in one bar, end to end, instead of side by side. Positive values stack up from zero and negative ones down from it, each in its own running total, so a negative part never hides a positive one (the convention spreadsheet charts use). The axis spans the totals.

#### Horizontal

Horizontal lays the categories top to bottom and the bars along the value axis, which is the arrangement for category names too long to sit under a bar:

	┌──────────┬────────────────────────────────┐
	│  Rent    │██████████████████ 1200         │  one band per category,
	│  Food    │█████████ 450                   │  Height/n px each
	│ Transpo… │████ 200                        │
	└──────────┼────────────────────────────────┘
	           0      500     1000   1500         ticks on a point axis
	 names: MaxLines(1), at most LabelWidth wide; values: ShowValues

The name column is sized by its widest name, up to LabelWidth, and a longer name is cut with an ellipsis (core.MaxLines). So is a tick label wider than its box. Every tick's box is two thirds of an interval wide (see pointLabels; the end ones cannot extend past the plot's edges), so a Format that writes "$1500" where "$1.5k" would do can still be cut on a narrow plot. The bands are fixed px boxes rather than flex weights, since Go knows the plot's height exactly; the tick labels use LineChart's point arithmetic, because ticks, like points, run edge to edge.

#### Values

ShowValues writes each bar's value at its tip: just above a vertical bar (below one that hangs negative), just past a horizontal one's end. A stacked chart shows each category's total at the end of its stack, and a grouped one a value per bar.

Text cannot be placed inside core.Canvas, so the values are a layer of ordinary Text over it (core.ZStack), and each is placed by arithmetic Go can do without knowing the drawn size of the plot:

	vertical     across: the bar's share of the width, as flex weights —
	             the cells barValueCells centres on the bars
	             along:  px, because the plot is Height px tall and a
	             stretched viewBox maps y linearly onto it
	horizontal   along:  flex weights again, a spacer as long as the bar's
	             share of the width before the label
	             across: px bands, as the names column has

A vertical chart keeps a label line of headroom above the plot (and one below, when a value is negative), so a bar reaching the end of the axis still has room for its label. A horizontal one cannot reserve room it cannot measure, so a bar leaving too little of the plot past its tip for its label (barValueRoom estimates that per label, from its length) carries its value inside, against its end, in whichever of the theme's inks contrasts with the bar.

<small>[comps/bar_chart.go:96](https://github.com/rohanthewiz/grmob/blob/master/comps/bar_chart.go#L96)</small>

#### func (BarChart) Render

```go
func (c BarChart) Render(ctx *core.Context) *core.Node
```

<small>[comps/bar_chart.go:150](https://github.com/rohanthewiz/grmob/blob/master/comps/bar_chart.go#L150)</small>

### type ChartPoint

```go
type ChartPoint struct {
	X, Y float64
}
```

ChartPoint is one (x, y) observation in a ScatterChart. A NaN or infinite coordinate leaves the point out.

<small>[comps/scatter_chart.go:13](https://github.com/rohanthewiz/grmob/blob/master/comps/scatter_chart.go#L13)</small>

### type ChartSeries

```go
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
```

ChartSeries is one named run of values: a line, an area, or one colour of bars in a grouped BarChart.

<small>[comps/chart.go:54](https://github.com/rohanthewiz/grmob/blob/master/comps/chart.go#L54)</small>

### type ChartSlice

```go
type ChartSlice struct {
	Label string
	Value float64

	// Color overrides the slice's palette colour.
	Color string
}
```

ChartSlice is one share of a DonutChart or PieChart.

<small>[comps/donut_chart.go:14](https://github.com/rohanthewiz/grmob/blob/master/comps/donut_chart.go#L14)</small>

### type DonutChart

```go
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
	// contains filtered or unexported fields
}
```

DonutChart draws shares of a whole as a ring, with a legend underneath.

	comps.DonutChart{
	    Subject:     "Budget",
	    Slices:      []comps.ChartSlice{{Label: "Rent", Value: 1200}, {Label: "Food", Value: 450}},
	    CenterValue: "$1,650",
	    CenterLabel: "per month",
	}

#### Geometry

Slices start at twelve o'clock and go clockwise, in the order given — the caller's order is kept rather than sorted, because a budget's categories have an order of their own that a legend should not reshuffle. Each slice is a core.Sector in a 100 × 100 viewBox drawn with CanvasFit, so the ring stays round in any box. A Gap of a degree or so is taken out of each slice's sweep, split evenly between its two ends, so neighbours of similar colour stay distinguishable; a slice too thin to lose that much keeps its full sweep instead of vanishing.

#### Legend and percentages

The legend lists each slice with its percentage, rounded by largest remainder so the column adds up to exactly 100 — independently rounded shares of 33⅓ each would read 33, 33, 33.

#### Accessibility

One element: "Budget: Rent 1200, 73%; Food 450, 27%". The centre text is included when it is set, since it is usually the total.

<small>[comps/donut_chart.go:52](https://github.com/rohanthewiz/grmob/blob/master/comps/donut_chart.go#L52)</small>

#### func (DonutChart) Render

```go
func (c DonutChart) Render(ctx *core.Context) *core.Node
```

<small>[comps/donut_chart.go:108](https://github.com/rohanthewiz/grmob/blob/master/comps/donut_chart.go#L108)</small>

### type Gauge

```go
type Gauge struct {
	// Value is where the arc fills to. It is clamped into [Min, Max]; NaN
	// draws an empty track.
	Value float64

	// Min and Max bound the range. When Max is not above Min the range is
	// 0 to 100.
	Min, Max float64

	// Label is a caption under the value ("Battery"), and names the gauge to
	// assistive tech.
	Label string

	// ValueText replaces the drawn and spoken value ("72%", "3.2 GB").
	// Empty draws the value with up to one decimal.
	ValueText string

	// Size is the box's side in px; 0 means 160.
	Size float64

	// Thickness is the arc's stroke width in px; 0 means a tenth of Size.
	Thickness float64

	// Sweep is the track's extent in degrees, 60 to 360; 0 means 240.
	Sweep float64

	// Color fills the arc; empty uses the theme's Primary. TrackColor is the
	// unfilled remainder; empty uses the theme's divider role.
	Color      string
	TrackColor string

	// Style is applied last, to the stack.
	Style []core.StyleProp

	// AccessibilityLabel replaces the generated sentence.
	AccessibilityLabel string
}
```

Gauge draws one value within a range as an arc filling a track, with the value written in the middle.

	comps.Gauge{Value: 72, Max: 100, Label: "Battery", ValueText: "72%"}

#### Geometry

The track is an arc of Sweep degrees, opening at the bottom and symmetric about twelve o'clock, so a 240° gauge runs from about eight o'clock round to four:

	   ╭───────╮
	 ╱     ▲     ╲       start = 90° + (360° − Sweep)/2
	│     72%     │      measured clockwise from three o'clock,
	 ╲  Battery  ╱       as core.Path.Arc takes angles
	  ↑         ↑
	start      end

Both arcs are strokes with round caps rather than filled sectors, because a stroke's width is in px: Thickness means the same on every target and at every Size. The canvas is CanvasFit in a Size × Size box, so the viewBox scale is known (Size/100 px per unit) and the radius is pulled in by half the stroke so the rounded ends stay inside the box.

The box is square even though a 240° arc leaves its bottom quarter empty, because the value text is centred on the arc's centre, and a ZStack centres its layers in its box. Trimming the box would move the text off the centre; placing it by measured offsets is what the rest of the framework avoids.

#### Accessibility

One element, "Battery, 72%": the Label and the ValueText (or the value and the range, "72 of 100", when no ValueText is given).

<small>[comps/gauge.go:43](https://github.com/rohanthewiz/grmob/blob/master/comps/gauge.go#L43)</small>

#### func (Gauge) Render

```go
func (g Gauge) Render(ctx *core.Context) *core.Node
```

<small>[comps/gauge.go:81](https://github.com/rohanthewiz/grmob/blob/master/comps/gauge.go#L81)</small>

### type LineChart

```go
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
```

LineChart draws one or more series as lines over a value axis, with the points evenly spaced from the left edge to the right.

	comps.LineChart{
	    Subject: "Weekly visits",
	    Labels:  []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"},
	    Series:  []comps.ChartSeries{{Name: "Visits", Values: visits}},
	}

The layout — y labels, gridlines, x labels, legend — is described at the top of chart.go. The plot is a core.Canvas with CanvasStretch, so it fills the width it is given and its lines stay StrokeWidth px thick at any width.

#### Missing values

A NaN value leaves a gap: the line stops at the point before it and starts again at the point after. A value on its own between two gaps is drawn as a dot, since a line of one point has no length.

#### Smooth

Smooth draws each run of values as a monotone cubic (Fritsch and Carlson, "Monotone Piecewise Cubic Interpolation", 1980) instead of straight segments. Monotone is the property that matters for data: the curve never rises above the higher of two neighbouring values or dips below the lower, so a smoothed line invents no peak the data does not have, which a Catmull-Rom or plain cubic spline does at every sharp turn. It survives the stretch, too: CanvasStretch scales each axis by its own factor, and a curve monotone between its points stays monotone under any per-axis scale.

#### Stacked

Stacked (an area chart's, usually) draws each series on top of the ones before it: series i's line is the running total of series 0..i, and its area fills from the previous total up to its own, so the top line is the whole and each band a part of it.

	total ─────╮        series 2's band: between total₁ and total₂
	      ░░░░░╰──╮
	total₁ ────────╰─   series 1's band
	      ▒▒▒▒▒▒▒▒▒▒▒
	total₀ ───────────  series 0's band, down to zero

A missing value counts as zero in a stack — a gap in one band would tear a hole through every band above it — and the axis spans the totals. The spoken summary still reads each series' own values, not the running totals, since those are what a listener asked about. Negative values stack like any other, downwards through the band below; a stack is for parts of a whole, which are not negative.

#### Points and dots

Points (and the lone-value dot above) are zero-length strokes with round caps, which every target draws as a circle StrokeWidth wide. A Circle path would not do: under CanvasStretch the viewBox is scaled differently on each axis, so a circle drawn in it comes out as an ellipse, while a stroke's width is never scaled.

<small>[comps/line_chart.go:68](https://github.com/rohanthewiz/grmob/blob/master/comps/line_chart.go#L68)</small>

#### func (LineChart) Render

```go
func (c LineChart) Render(ctx *core.Context) *core.Node
```

<small>[comps/line_chart.go:140](https://github.com/rohanthewiz/grmob/blob/master/comps/line_chart.go#L140)</small>

### type PieChart

```go
type PieChart DonutChart
```

PieChart is a DonutChart without the hole: every slice is a wedge from the centre. Thickness and the centre text are ignored, and Gap defaults to none, since a gap in a pie reads as a crack rather than a separator.

<small>[comps/donut_chart.go:100](https://github.com/rohanthewiz/grmob/blob/master/comps/donut_chart.go#L100)</small>

#### func (PieChart) Render

```go
func (p PieChart) Render(ctx *core.Context) *core.Node
```

<small>[comps/donut_chart.go:102](https://github.com/rohanthewiz/grmob/blob/master/comps/donut_chart.go#L102)</small>

### type ScatterChart

```go
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
```

ScatterChart plots points against two value axes.

	comps.ScatterChart{
	    Subject: "Height and weight",
	    Series:  []comps.ScatterSeries{{Name: "Players", Points: points}},
	    XFormat: func(v float64) string { return fmt.Sprintf("%.0f cm", v) },
	}

#### Two value axes, one layout

The y axis is LineChart's: nice ticks down the left, gridlines behind the data. The x axis is nice ticks too, and they run edge to edge exactly as a line chart's points do, so the tick labels under the plot are placed by the same point arithmetic (pointLabels, in chart.go): the first label aligned to the start edge, the last to the end, the rest centred on their ticks. No width is measured.

	20 ┤ ·    ·         ·
	10 ┤    ·     ·  ·
	 0 ┼──────────────────
	   0     50     100

Neither axis is pulled to zero unless asked (XZeroBased, ZeroBased): a scatter's question is how two quantities move together, and an origin far from the data squeezes the cloud into a corner.

#### Dots

Each point is a zero-length stroke with round caps, LineChart's dot, for the same reason: under CanvasStretch a circle path would come out as an ellipse, and a stroke's width is never scaled. Every point of a series is one subpath of one shape, so a thousand points are one canvas node.

#### Square dots past three series

From four series up, every second series (the 2nd, 4th, ...) draws square dots, and its legend swatch is square while the others' are round. Colour alone does not hold that many series apart: DefaultChartColors' slots 3 to 5 are under 3:1 on a light page, and with four or more series some pair of dots fails an all-pairs distinctness check, so a second, non-colour cue is what lets a reader match a cloud to its legend entry (WCAG 1.4.1). Up to three series every dot stays round, as it always was; the first three slots pass the check against each other.

A square dot is a stroke with square caps along a segment a hundredth of a viewBox unit long, not a zero-length one: SwiftUI draws nothing for a zero-length subpath with square caps (round caps it draws). Under CanvasStretch the segment scales to a few hundredths of a pixel, so the dot is square to the eye on every target, and like the round dot its size never scales.

<small>[comps/scatter_chart.go:79](https://github.com/rohanthewiz/grmob/blob/master/comps/scatter_chart.go#L79)</small>

#### func (ScatterChart) Render

```go
func (c ScatterChart) Render(ctx *core.Context) *core.Node
```

<small>[comps/scatter_chart.go:112](https://github.com/rohanthewiz/grmob/blob/master/comps/scatter_chart.go#L112)</small>

### type ScatterSeries

```go
type ScatterSeries struct {
	// Name labels the series in the legend (drawn when there is more than one
	// series) and in the spoken summary.
	Name string

	Points []ChartPoint

	// Color overrides the series' palette colour.
	Color string
}
```

ScatterSeries is one colour of points in a ScatterChart.

<small>[comps/scatter_chart.go:18](https://github.com/rohanthewiz/grmob/blob/master/comps/scatter_chart.go#L18)</small>

### type Sparkline

```go
type Sparkline struct {
	Values []float64

	// Subject names the series for the spoken summary ("Visits this week").
	Subject string

	// Height is in px; 0 means 32.
	Height float64

	// StrokeWidth is in px; 0 means 1.5.
	StrokeWidth float64

	// Color is the line; empty uses the theme's Primary.
	Color string

	// Area fills under the line with a tint of Color, down to the bottom edge.
	// Not down to zero, as AreaChart's does: a sparkline's domain rarely
	// includes zero, and with no axis the tint is texture, not a measurement.
	Area bool

	// ShowLast draws a dot on the last value.
	ShowLast bool

	// Format writes a value in the spoken summary.
	Format func(float64) string

	// Style is applied last, to the canvas.
	Style []core.StyleProp

	// AccessibilityLabel replaces the generated summary.
	AccessibilityLabel string
}
```

Sparkline is a word-sized line chart: no axes, no labels, just the shape of a series, for a table cell, a list row or beside a StatTile's number.

	core.Row(
	    core.Text("Visits"),
	    comps.Sparkline{Values: visits, Subject: "Visits this week", Style: []core.StyleProp{core.Width("80px")}},
	)

It fills its parent's width unless Style gives one, and is Height px tall. The domain is the data's own low to high, not rounded to ticks: with no axis to read against, the full height is spent on the shape.

NaN values break the line, as on LineChart. ShowLast marks the latest value with a dot, which is where a reader's eye goes first; the dot is a round- capped zero-length stroke so it stays round when the line is stretched (see LineChart's "Points and dots").

<small>[comps/sparkline.go:25](https://github.com/rohanthewiz/grmob/blob/master/comps/sparkline.go#L25)</small>

#### func (Sparkline) Render

```go
func (s Sparkline) Render(ctx *core.Context) *core.Node
```

<small>[comps/sparkline.go:58](https://github.com/rohanthewiz/grmob/blob/master/comps/sparkline.go#L58)</small>

