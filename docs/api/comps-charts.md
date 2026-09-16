# Package comps — Charts

```go
import "github.com/rohanthewiz/grmob/comps"
```

Sparklines, line, area and bar charts, donuts and pies, and gauges, drawn on core.Canvas.

One of 7 topic pages of [package comps](comps.md), which has the package overview and an index of every topic. This page documents the declarations in `comps/chart.go`, `comps/sparkline.go`, `comps/line_chart.go`, `comps/bar_chart.go`, `comps/donut_chart.go`, `comps/gauge.go`.

## Index

- [`type AreaChart`](#type-areachart)
    - [`func (AreaChart) Render`](#func-areachart-render)
- [`type BarChart`](#type-barchart)
    - [`func (BarChart) Render`](#func-barchart-render)
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
- [`type Sparkline`](#type-sparkline)
    - [`func (Sparkline) Render`](#func-sparkline-render)

## Types

### type AreaChart

```go
type AreaChart LineChart
```

AreaChart is a LineChart with Area set: each series filled down to zero. Several series overlap translucently rather than stacking.

<small>[comps/line_chart.go:90](https://github.com/rohanthewiz/grmob/blob/master/comps/line_chart.go#L90)</small>

#### func (AreaChart) Render

```go
func (a AreaChart) Render(ctx *core.Context) *core.Node
```

<small>[comps/line_chart.go:92](https://github.com/rohanthewiz/grmob/blob/master/comps/line_chart.go#L92)</small>

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

	// Format writes a y tick label and a spoken value.
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

<small>[comps/bar_chart.go:38](https://github.com/rohanthewiz/grmob/blob/master/comps/bar_chart.go#L38)</small>

#### func (BarChart) Render

```go
func (c BarChart) Render(ctx *core.Context) *core.Node
```

<small>[comps/bar_chart.go:75](https://github.com/rohanthewiz/grmob/blob/master/comps/bar_chart.go#L75)</small>

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
	// AreaChart is this with the field set.
	Area bool

	// ZeroBased starts the value axis at zero even when every value is far
	// above it. Off by default for a line — its shape is the point, and a
	// temperature between 18 and 24 drawn against 0 is a flat line — and
	// always on for an area, whose filled height would otherwise misstate the
	// value.
	ZeroBased bool

	// Points marks each value with a dot.
	Points bool

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

#### Points and dots

Points (and the lone-value dot above) are zero-length strokes with round caps, which every target draws as a circle StrokeWidth wide. A Circle path would not do: under CanvasStretch the viewBox is scaled differently on each axis, so a circle drawn in it comes out as an ellipse, while a stroke's width is never scaled.

<small>[comps/line_chart.go:37](https://github.com/rohanthewiz/grmob/blob/master/comps/line_chart.go#L37)</small>

#### func (LineChart) Render

```go
func (c LineChart) Render(ctx *core.Context) *core.Node
```

<small>[comps/line_chart.go:98](https://github.com/rohanthewiz/grmob/blob/master/comps/line_chart.go#L98)</small>

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

