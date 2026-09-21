# Package comps — Charts

```go
import "github.com/rohanthewiz/grmob/comps"
```

Sparklines, line, area, bar and scatter charts, histograms, heatmaps and calendar heatmaps, donuts and pies, gauges, candlesticks, funnels, radars and audio waveforms, drawn on core.Canvas.

One of 7 topic pages of [package comps](comps.md), which has the package overview and an index of every topic. This page documents the declarations in `comps/chart.go`, `comps/sparkline.go`, `comps/line_chart.go`, `comps/bar_chart.go`, `comps/histogram.go`, `comps/scatter_chart.go`, `comps/heatmap.go`, `comps/donut_chart.go`, `comps/gauge.go`, `comps/candlestick_chart.go`, `comps/funnel_chart.go`, `comps/radar_chart.go`, `comps/waveform.go`.

## Index

- [Constants](#constants) — `ConcernFunnelChartStageGrows`, `ConcernRadarChartTooFewAxes`
- [`type AreaChart`](#type-areachart)
    - [`func (AreaChart) Render`](#func-areachart-render)
- [`type BarChart`](#type-barchart)
    - [`func (BarChart) Render`](#func-barchart-render)
- [`type CalendarHeatmap`](#type-calendarheatmap)
    - [`func (CalendarHeatmap) Render`](#func-calendarheatmap-render)
    - [`func (CalendarHeatmap) WeeksFor`](#func-calendarheatmap-weeksfor)
- [`type Candle`](#type-candle)
- [`type CandlestickChart`](#type-candlestickchart)
    - [`func (CandlestickChart) Render`](#func-candlestickchart-render)
- [`type ChartPoint`](#type-chartpoint)
- [`type ChartSeries`](#type-chartseries)
- [`type ChartSlice`](#type-chartslice)
- [`type DayValue`](#type-dayvalue)
- [`type DonutChart`](#type-donutchart)
    - [`func (DonutChart) Render`](#func-donutchart-render)
- [`type FunnelChart`](#type-funnelchart)
    - [`func (FunnelChart) Render`](#func-funnelchart-render)
- [`type FunnelStage`](#type-funnelstage)
- [`type Gauge`](#type-gauge)
    - [`func (Gauge) Render`](#func-gauge-render)
- [`type Heatmap`](#type-heatmap)
    - [`func (Heatmap) Render`](#func-heatmap-render)
- [`type Histogram`](#type-histogram)
    - [`func (Histogram) Render`](#func-histogram-render)
- [`type LineChart`](#type-linechart)
    - [`func (LineChart) Render`](#func-linechart-render)
- [`type PieChart`](#type-piechart)
    - [`func (PieChart) Render`](#func-piechart-render)
- [`type RadarChart`](#type-radarchart)
    - [`func (RadarChart) Render`](#func-radarchart-render)
- [`type ScatterChart`](#type-scatterchart)
    - [`func (ScatterChart) Render`](#func-scatterchart-render)
- [`type ScatterSeries`](#type-scatterseries)
- [`type Sparkline`](#type-sparkline)
    - [`func (Sparkline) Render`](#func-sparkline-render)
- [`type Waveform`](#type-waveform)
    - [`func (Waveform) BarsFor`](#func-waveform-barsfor)
    - [`func (Waveform) Render`](#func-waveform-render)

## Constants

ConcernFunnelChartStageGrows is raised, in debug builds only, when a FunnelChart stage is larger than the one before it and AllowIncrease is not set. The stage is drawn as it is either way (see "A stage that grows"); the concern is that the usual cause is stages passed out of order, which draws a plausible funnel that says the wrong thing.

```go
const ConcernFunnelChartStageGrows = "funnel-chart-stage-grows"
```

<small>[comps/funnel_chart.go:15](https://github.com/rohanthewiz/grmob/blob/master/comps/funnel_chart.go#L15)</small>

ConcernRadarChartTooFewAxes is raised, in debug builds only, when a RadarChart has fewer than three Axes. Two axes span a line and one a point, so there is no polygon to draw and the chart shows its empty rim.

```go
const ConcernRadarChartTooFewAxes = "radar-chart-too-few-axes"
```

<small>[comps/radar_chart.go:14](https://github.com/rohanthewiz/grmob/blob/master/comps/radar_chart.go#L14)</small>

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

### type CalendarHeatmap

```go
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
```

CalendarHeatmap is the contribution calendar: a column per week, a row per weekday, and each day painted by its count — commits, workouts, words written.

	comps.CalendarHeatmap{
	    Subject: "Workouts",
	    Days:    workouts,          // []comps.DayValue
	    End:     today,
	}

	      Jan         Feb         Mar
	Mon  ░ ▒ ░ · ▓ ░ ▒ ░ · ░ ▓ █ ░ ▒ ░ ▒ ░
	Wed  ▒ ░ · ░ ░ ▓ ░ · ▒ ░ ░ ▒ ▓ ░ ░ ·
	Fri  ░ · ▒ ░ ▒ ░ · ░ ░ ▒ ░ ░ ▒ ░ ▓
	      0 ■■■■■ 4

It is a Heatmap — the same grid, scale, legend and one-sentence summary — with the calendar arithmetic done here:

  - The last column is the week holding End, and the grid runs Weeks columns back from it, each week starting on WeekStart (Sunday, the zero value, as Calendar's does).
  - Days after End are not drawn at all; they have not happened, which is different from having happened with nothing in them.
  - A day with no entries, or entries summing to zero, is "no data" and painted Surface — the contribution calendar's convention, where an empty day is the ground and every step is some activity. See "No data is not zero" on Heatmap for why this is a choice made here rather than there.
  - Days are matched by calendar date in End's location, and several entries on one day are summed.
  - Rows are labelled on alternate weekdays (Mon, Wed, Fri under a Sunday start), and a column is labelled with its month's name when it holds the 1st — but not the first column, if the next month's name is fewer than three columns away, where the two would collide.

The summary is the calendar's own: "Workouts: 42 over 17 weeks, on 23 days; most on Tue 3 Mar 2026, 4." When several days share the highest total, the sentence names the earliest of them. A tie has no right answer, so it only needs to be stable: the same days always name the same day. Earliest is simply the order the grid is walked in, and a reader told "most on" a date can find it by reading forward from there. The sentence does not mention the tie, because counting how many days share the peak would be a second statistic that the grid itself does not show.

<small>[comps/heatmap.go:463](https://github.com/rohanthewiz/grmob/blob/master/comps/heatmap.go#L463)</small>

#### func (CalendarHeatmap) Render

```go
func (c CalendarHeatmap) Render(ctx *core.Context) *core.Node
```

<small>[comps/heatmap.go:546](https://github.com/rohanthewiz/grmob/blob/master/comps/heatmap.go#L546)</small>

#### func (CalendarHeatmap) WeeksFor

```go
func (c CalendarHeatmap) WeeksFor(width float64) int
```

WeeksFor is how many week columns fit width px with square cells — each column as wide as a row is tall (CellHeight, 14 by default) — clamped to between 1 and 53, a year.

	win := hooks.UseWindow(ctx)
	weeks := comps.CalendarHeatmap{}.WeeksFor(win.Width - 2*16) // less the screen's padding
	comps.CalendarHeatmap{Days: days, Weeks: weeks}

The grid stretches its columns to whatever width it is given, so Weeks is not what makes it fit — it always fits. What Weeks decides is the cell's shape: 53 weeks across a 360px phone is a column under six pixels wide under a row fourteen tall, a grid of slivers. This returns the count at which the cells come out square.

It is arithmetic on a width the caller supplies, not a measurement: no host reports a rendered width, and the caller knows its own padding where this widget does not. It reads no hook, so CalendarHeatmap stays safe to render conditionally; the caller's hooks.UseWindow is what makes a rotation or a fold re-render with a new count.

<small>[comps/heatmap.go:528](https://github.com/rohanthewiz/grmob/blob/master/comps/heatmap.go#L528)</small>

### type Candle

```go
type Candle struct {
	Open, High, Low, Close float64
}
```

Candle is one period of a CandlestickChart: where the price opened and closed, and the highest and lowest it reached in between.

<small>[comps/candlestick_chart.go:11](https://github.com/rohanthewiz/grmob/blob/master/comps/candlestick_chart.go#L11)</small>

### type CandlestickChart

```go
type CandlestickChart struct {
	// Candles are the periods, oldest first. A candle holding a NaN or an
	// infinity is not drawn and keeps its slot, as a gap in the row.
	Candles []Candle

	// Labels name the periods. Up to five are drawn; all feed the summary.
	Labels []string

	// Subject says what is priced. It leads the spoken summary and is not
	// drawn.
	Subject string

	// Height is the plot's height in px, excluding labels; 0 means 160.
	Height float64

	// UpColor and DownColor ink a period that rose (close at or above open)
	// and one that fell; empty means the theme's Success and Error. See
	// "Which colour is up".
	UpColor   string
	DownColor string

	// BodyFill is the fraction of each slot the body takes; 0 means 0.6.
	BodyFill float64

	// ShowLegend draws a key for the two colours under the plot, reading
	// UpLabel and DownLabel ("Up" and "Down" when empty).
	ShowLegend bool
	UpLabel    string
	DownLabel  string

	// Format writes a tick label and a spoken price.
	Format func(float64) string

	// Style is applied last, to the outer column.
	Style []core.StyleProp

	// AccessibilityLabel replaces the generated summary.
	AccessibilityLabel string
}
```

CandlestickChart draws a price per period as a candle: a thin wick from the period's low to its high, and a body from its open to its close, coloured by which way the period went.

	comps.CandlestickChart{
	    Subject: "ACME",
	    Labels:  []string{"Mon", "Tue", "Wed"},
	    Candles: []comps.Candle{{Open: 102, High: 108, Low: 101, Close: 107}, ...},
	}

#### Slots

The width is BarChart's: one equal slot per period, so the label row under the plot is bandLabels unchanged and each label sits under its candle on every target without Go knowing the drawn width.

	│  slot 0  │  slot 1  │  slot 2  │
	│    │     │          │    │     │   wick: a 1 px line, low to high,
	│   ┌┴┐    │    │     │   ┌┴┐    │         at the slot's centre
	│   │ │    │   ┌┴┐    │   │ │    │   body: BodyFill of the slot wide,
	│   └┬┘    │   └┬┘    │   └┬┘    │         open to close
	│   Mon    │   Tue    │   Wed    │

#### The axis does not start at zero

A bar's length is its value, so BarChart's axis must include zero. A candle states a range, not a length, and a share trading between 101 and 108 drawn on an axis from 0 would be a row of identical slivers. The axis is niceScale over the lows and highs, as LineChart's is by default.

#### Which colour is up

A period that closed at or above its open takes UpColor, the theme's Success, and one that closed below takes DownColor, its Error. That is the Western convention. Markets in China, Japan and Korea print it the other way round (red rises), so both are fields, and a chart for those readers swaps them. ShowLegend keys the two colours for a page whose readers may be used to either.

Colour is the only drawn difference between the two, so the spoken summary counts the periods that rose and fell.

#### A doji

A period that closed where it opened has a body of no height, which would leave a bare wick that looks like missing data. Its body is drawn one px tall. The plot is Height px tall and CanvasStretch maps the viewBox's y linearly onto it, so one px is exactly chartView/Height units: no measurement needed.

#### One shape per kind, not per candle

Four shapes carry every candle: the rising wicks, the falling wicks, the rising bodies, the falling bodies, each as one path of subpaths. Thirty candles cost four nodes and not sixty, and the slots stay put while the data changes, so a new period patches props and moves nothing.

#### Theme roles read

	Rising     Colors.Success
	Falling    Colors.Error
	Gridlines  Colors.BorderColor()
	Labels     Colors.TextSecondary

<small>[comps/candlestick_chart.go:78](https://github.com/rohanthewiz/grmob/blob/master/comps/candlestick_chart.go#L78)</small>

#### func (CandlestickChart) Render

```go
func (c CandlestickChart) Render(ctx *core.Context) *core.Node
```

<small>[comps/candlestick_chart.go:123](https://github.com/rohanthewiz/grmob/blob/master/comps/candlestick_chart.go#L123)</small>

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

### type DayValue

```go
type DayValue struct {
	Day   time.Time
	Value float64
}
```

DayValue is one dated value of a CalendarHeatmap.

<small>[comps/heatmap.go:498](https://github.com/rohanthewiz/grmob/blob/master/comps/heatmap.go#L498)</small>

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

### type FunnelChart

```go
type FunnelChart struct {
	// Stages are the steps, first to last. A negative, NaN or infinite value
	// draws as zero.
	Stages []FunnelStage

	// Subject says what is being funnelled. It leads the spoken summary and
	// is not drawn.
	Subject string

	// Height is the drawing's height in px; 0 means 40 a stage (at least 80).
	Height float64

	// ShowRates adds the step conversion between each pair of stages: the
	// later stage as a percentage of the earlier one. A step from a stage of
	// zero has no rate and shows none.
	ShowRates bool

	// LabelWidth caps the name column, in px; 0 means 96. A longer name is
	// cut with an ellipsis.
	LabelWidth float64

	// Colors overrides the fading single hue with a colour per stage, cycled.
	Colors []string

	// AllowIncrease says a stage larger than the one before it is intended.
	// See "A stage that grows".
	AllowIncrease bool

	// Format writes a stage's value, drawn and spoken.
	Format func(float64) string

	// Style is applied last, to the outer row.
	Style []core.StyleProp

	// AccessibilityLabel replaces the generated summary.
	AccessibilityLabel string
}
```

FunnelChart draws how many of a population survive each step of a process: visitors, sign-ups, purchases. Each stage is a centred band whose width is its value, tapering into the next.

	comps.FunnelChart{
	    Subject:   "Checkout",
	    ShowRates: true,
	    Stages: []comps.FunnelStage{
	        {Label: "Visited", Value: 1200},
	        {Label: "Signed up", Value: 744},
	        {Label: "Paid", Value: 93},
	    },
	}

#### Layout

	┌───────────┬──────────────────────┬──────┬─────┐
	│  Visited  │ ████████████████████ │ 1200 │     │  one band per stage,
	│           │  ╲████████████████╱  │      │ 62% │  Height/n px each
	│ Signed up │   ██████████████     │  744 │     │
	│           │     ╲████████╱       │      │ 13% │  a rate sits on the
	│      Paid │       ██             │   93 │     │  line between two bands
	└───────────┴──────────────────────┴──────┴─────┘
	  names        core.Canvas, stretched  values  ShowRates

Canvas has no text, so the names, the values and the rates are columns of Text beside it, the arrangement a horizontal BarChart uses. Every column is Height px tall and cut into px bands by the same arithmetic as the drawing, so a name is level with its band on every target. The rates column is the same bands shifted down by half of one, which puts each rate level with the boundary it describes:

	values   │ band 0 │ band 1 │ band 2 │
	rates    │ ½ │ 0→1    │ 1→2    │ ½ │

#### The shape of a band

A stage's band is a trapezoid: as wide at the top as its own value and at the bottom as the next stage's, so the slope between two stages is the drop between them. The last stage has no next and is a rectangle. Widths are shares of the largest stage, which is usually the first.

#### One colour, fading

The stages are one population at successive moments, not categories, so they are one hue: the first chart colour, stepping down in alpha from solid to funnelMinAlpha. The categorical palette would say the stages are different kinds of thing. A stage's Color, or Colors, overrides it.

#### A stage that grows

A stage larger than the one before it is drawn as it is: its band is wider than the one above, and the band above flares out to meet it. It is not clamped, because real funnels have them (people who re-enter at a later step, a stage counted over a longer window) and a chart that hid the growth would misreport the data. Its rate reads over 100%.

Far more often it is a mistake, stages passed out of order, so debug builds report ConcernFunnelChartStageGrows. AllowIncrease says the data is meant and silences it.

#### Accessibility

One element: "Checkout: Visited 1200; Signed up 744, 62% of the step before; Paid 93, 13% of the step before; 8% overall."

#### Theme roles read

	Bands    Colors.ChartColors()[0], fading in alpha
	Names    Colors.TextSecondary
	Values   Colors.TextPrimary
	Rates    Colors.TextSecondary

<small>[comps/funnel_chart.go:98](https://github.com/rohanthewiz/grmob/blob/master/comps/funnel_chart.go#L98)</small>

#### func (FunnelChart) Render

```go
func (c FunnelChart) Render(ctx *core.Context) *core.Node
```

<small>[comps/funnel_chart.go:146](https://github.com/rohanthewiz/grmob/blob/master/comps/funnel_chart.go#L146)</small>

### type FunnelStage

```go
type FunnelStage struct {
	Label string
	Value float64

	// Color overrides the stage's colour.
	Color string
}
```

FunnelStage is one step of a FunnelChart.

<small>[comps/funnel_chart.go:18](https://github.com/rohanthewiz/grmob/blob/master/comps/funnel_chart.go#L18)</small>

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

### type Heatmap

```go
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
```

Heatmap draws a grid of values as a grid of colours: a row per thing, a column per slot, and each cell painted by how much it holds.

	comps.Heatmap{
	    Subject:      "Orders by hour",
	    RowLabels:    []string{"Mon", "Tue", "Wed"},
	    ColumnLabels: []string{"9", "12", "15", "18"},
	    Values: [][]float64{
	        {2, 8, 5, 1},
	        {3, 9, 7, 2},
	        {1, 4, math.NaN(), 0},
	    },
	}

	┌─────┬───────────────────────────────┐
	│ Mon │ ░░░ ▓▓▓ ▒▒▒ ░░░               │  core.Canvas, CanvasStretch:
	│ Tue │ ░░░ ███ ▓▓▓ ░░░               │  CellHeight px a row, the
	│ Wed │ ░░░ ▒▒▒ ··· ░░░               │  width shared by the columns
	│     │  9   12  15  18               │  column labels, equal slots
	└─────┴───────────────────────────────┘
	  0 ■■■■■ 9                            legend: the steps, low to high

#### The colours are the theme's Sequential role

A heatmap paints a \*quantity\*, and the palette's categorical Chart role cannot do that: its slots are ordered to stay apart, not to read as more. So the cells take core.ColorPalette.SequentialColors — a checked run of lightness in one hue, least to most, added for this widget; the case against deriving one from Primary is written on that role. Colors overrides it per chart.

#### Steps, not a gradient

The value range is cut into as many equal steps as the scale has colours (five on the bundled themes), and a cell takes its step's colour flat. A continuous blend would claim a precision the eye cannot read back from a colour, and steps are what a legend can key: five swatches say exactly which colours mean what, where a gradient bar can only say "towards here". One value everywhere is the top step — everything present is the most there is.

#### No data is not zero

A NaN cell is "no data" and is painted in the theme's Surface, apart from every step (the scale's lightest entry was checked against each bundled Surface for exactly this pair). Zero is a value like any other and takes the step it falls in; a caller whose zeros mean "nothing happened" (a contribution calendar) passes NaN for them, which is what CalendarHeatmap does.

#### Drawn as one shape per colour

Every cell of a step is a subpath of that step's one path, as BarChart draws a series, so a 7 × 53 calendar is six shapes rather than 371 and a cell changing step patches two paths. The cells are drawn with a gap a tenth of a cell wide, which is what makes a grid of equal colours read as cells rather than as a band.

#### Labels

Row labels sit in a column of boxes each exactly CellHeight tall, so each centres on its row by construction. Column labels are the equal slots BarChart's categories use, with one addition: an empty label gives its slot to the label before it, so "Mar" over four unlabelled weeks has four columns of room and starts at its first — how a calendar names its months. A label with a slot of its own is centred on it.

#### One element, one sentence

As every chart here, the grid is one RoleImg with a summary sentence and everything under it hidden: "Orders by hour: 3 rows by 4 columns; low 0 at Wed, 18; high 9 at Tue, 12; 1 cell with no data."

#### Theme roles read

	Cells      Colors.SequentialColors (or Colors), Colors.Surface for no data
	Labels     Colors.TextSecondary at the charts' label size

<small>[comps/heatmap.go:88](https://github.com/rohanthewiz/grmob/blob/master/comps/heatmap.go#L88)</small>

#### func (Heatmap) Render

```go
func (h Heatmap) Render(ctx *core.Context) *core.Node
```

<small>[comps/heatmap.go:121](https://github.com/rohanthewiz/grmob/blob/master/comps/heatmap.go#L121)</small>

### type Histogram

```go
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
```

Histogram counts raw values into bins and draws the counts as touching bars over a numeric axis — how response times, ages or scores are spread.

	comps.Histogram{
	    Subject: "Response time (ms)",
	    Values:  samples,          // the raw values, not counts
	}

	 12 ┤      ██
	    │   ██ ██ ██
	  6 ┤   ██ ██ ██ ██
	    │██ ██ ██ ██ ██ ██
	  0 ┼──┴──┴──┴──┴──┴──┤
	    0    100   200   300              labels on the bin *edges*

#### The axis is numeric, and that was the catch

A histogram looks like a BarChart, and the binning is a few lines of Go — the plan's catch was that BarChart's categories are strings, centred under their bars, while a bin is a \*range\* whose edges are what the axis names. "100–150" under a bar is a category chart of ranges; "100" and "150" at the bars' shoulders is a histogram, and the difference is what a reader measures a value against.

It settled without a new axis. n bins have n+1 edges evenly spaced from the plot's left edge to its right, which is exactly the spacing LineChart gives its points — so the edge labels are pointLabels, the row LineChart and the horizontal BarChart already use for axes that run edge to edge. The bars are BarChart's single-series slots with the gap between them nearly closed: a histogram's bars touch, because the bins do.

#### Nice edges

The edges come from the charts' own niceScale, so bins are 10 or 25 or 0.5 wide and start on a multiple of that — "0, 50, 100", never "3.7, 51.2". Bins is therefore a target, and the drawn count is at most it: 12 asked over 0–95 may come out as 10 bins of 10. Zero Bins takes Sturges' rule, ⌈log₂ n⌉ + 1, the usual default for a sample of unknown shape. A value on an interior edge belongs to the bin it opens (the half-open \[a, b) every statistics package uses), and the largest value, on the last edge, to the last bin.

#### Counts are whole

The count axis never ticks at a fraction: a step below 1 is raised to 1, because "half a sample" is a gridline pointing at nothing.

#### One element, one sentence

"Response time (ms): 120 values in 8 bins of 50 from 0 to 400; most, 34, between 100 and 150."

#### Theme roles read

	Bars       Chart slot 1 (Colors.ChartColors), or Color
	Grid       Colors.BorderColor; the zero line Colors.ControlBorderColor
	Labels     Colors.TextSecondary at the charts' label size

<small>[comps/histogram.go:66](https://github.com/rohanthewiz/grmob/blob/master/comps/histogram.go#L66)</small>

#### func (Histogram) Render

```go
func (c Histogram) Render(ctx *core.Context) *core.Node
```

<small>[comps/histogram.go:99](https://github.com/rohanthewiz/grmob/blob/master/comps/histogram.go#L99)</small>

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

### type RadarChart

```go
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
```

RadarChart draws several measures of one subject as a polygon on spokes around a centre, one spoke per measure, so that a profile reads as a shape: a player's speed, power and stamina; a product scored on five criteria.

	comps.RadarChart{
	    Subject: "Player",
	    Axes:    []string{"Speed", "Power", "Stamina", "Skill", "Vision"},
	    Series:  []comps.ChartSeries{{Name: "Ade", Values: []float64{8, 6, 7, 9, 5}}},
	    Max:     10,
	    Filled:  true,
	}

#### Geometry

Axis k of n points at −90° + k·360°/n: the first straight up, the rest clockwise, the order a clock face and DonutChart both read in. Value v sits at v/Max of the way out. The grid is Rings concentric polygons with the same corners and a spoke to each, so a gridline is a straight run between two spokes and a value on a spoke reads against it exactly (a circular grid would only agree with the polygon on the spokes themselves).

#### The labels, and why they are not in the drawing

core.Canvas has no text, so every label is a Text laid over the drawing in a core.ZStack. A ZStack gives nine named places (core.StackAlign), which is Compass's four letters and no more; five or seven labels round a rim need arbitrary ones. They get them from core.Translate. Each label is a box the stack centres, as it centres any layer that says nothing, moved by px offsets from the same trigonometry as the drawing:

	            Speed                  box of LabelWidth × a label line,
	        ┌─────────┐                centred, then translated so that the
	Vision ╱     │     ╲ Power         edge nearest the rim touches the point
	      │      ┼      │              radarLabelGap px outside the spoke's
	      ╲     ╱ ╲     ╱              end: start-aligned on the right,
	  Skill ───       ─── Stamina      end-aligned on the left, centred at
	                                   the top and bottom

Translate's px are exact because Size is: the drawing is a Size px square under CanvasFit, so a viewBox unit is Size/100 px on both axes and Go knows where every spoke ends without measuring anything. The stack is pinned to the drawing plus a label's width either side and a label's height above and below, so no label leaves it.

The scale's values are written the same way, just beside the upward spoke at each ring, because a radar with unlabelled rings shows shape and no magnitude.

#### Right to left

Translate's x is leading-relative, so in a right-to-left layout the labels mirror across the vertical spoke while the drawing, like every Canvas, does not. Axis k's label then sits on axis n−k's spoke. That is the same disagreement LineChart's label row has with its line under RTL, and it wants the same answer (a Canvas that mirrors, or a direction Go can read), which is a renderer's to give.

#### Values outside the scale

A value above Max is drawn at Max and a negative one at the centre: the rim is the chart's edge and a polygon crossing it would run under the labels. The spoken summary reads the value as given. NaN is a missing value; it draws at the centre and is left out of the summary.

#### Accessibility

One element. Up to radarSummaryLimit axes are read in full, per series ("Ade, Speed 8, Power 6, …"); past that, each series' lowest and highest.

#### Theme roles read

	Series   Colors.ChartColors(), a fill of the same hue at radarFillAlpha
	Grid     Colors.BorderColor()
	Labels   Colors.TextSecondary

<small>[comps/radar_chart.go:90](https://github.com/rohanthewiz/grmob/blob/master/comps/radar_chart.go#L90)</small>

#### func (RadarChart) Render

```go
func (c RadarChart) Render(ctx *core.Context) *core.Node
```

<small>[comps/radar_chart.go:160](https://github.com/rohanthewiz/grmob/blob/master/comps/radar_chart.go#L160)</small>

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

### type Waveform

```go
type Waveform struct {
	// Peaks are the loudness samples in time order, 0 (silence) to 1 (full
	// scale). Values outside that are clamped; NaN reads as silence.
	Peaks []float64

	// Progress is how much has played, 0 to 1. A bar is played once its
	// centre is behind the playhead.
	Progress float64

	// Bars is how many bars to draw; 0 means one per peak, up to 56. See
	// "More peaks than bars" and BarsFor.
	Bars int

	// Height is the strip's height in px; 0 means 48.
	Height float64

	// BarWidth is each bar's thickness in px (0 means 3), and Gap the space
	// BarsFor leaves between bars (0 means 2). Gap is only BarsFor's: drawn
	// bars are spread evenly over the width the layout gives.
	BarWidth float64
	Gap      float64

	// PlayedColor and RestColor ink the bars behind and ahead of the
	// playhead; empty means the theme's Primary and its control border.
	PlayedColor string
	RestColor   string

	// Decorative hides the strip from screen readers, for a waveform beside
	// a control that already speaks the position.
	Decorative bool

	// Style is applied last, to the canvas.
	Style []core.StyleProp

	// AccessibilityLabel replaces the generated sentence.
	AccessibilityLabel string
}
```

Waveform draws a recording's loudness over time as a row of bars mirrored about a centre line, with the part already played in the accent colour: the strip a voice message or a podcast player shows.

The peaks come from the caller. No host decodes audio for Go, so a server or a build step computes them (one number per slice of the recording, its loudest sample, scaled to 0–1) and ships them beside the file. Without peaks there is no waveform to draw, and this widget does not invent one.

	comps.Waveform{Peaks: msg.Peaks, Progress: position / duration}

#### Bars are strokes, and that is what keeps them round

	    ╷   ┃ ╷                 one vertical line per bar, stroked BarWidth
	  ╷ ┃ ╷ ┃ ┃ ╷   ╷           px wide with round caps, mirrored about the
	──┃─┃─┃─┃─┃─┃─╷─┃─╷──       centre: from mid − peak·reach to mid + peak·reach
	  ╵ ┃ ╵ ┃ ┃ ╵   ╵
	    ╵   ┃ ╵
	 ◀─ played ─▶◀─ rest ─▶

The drawing is stretched across whatever width the layout gives (core.CanvasStretch), and a rounded rectangle drawn as a path would stretch with it, its corners turning to ellipses. A Canvas stroke is never scaled: its width is layout px on every target (see core.Canvas). So a bar is a line, its thickness is the stroke's, and its rounded ends are the stroke's round caps, all exactly BarWidth px however wide the strip is drawn. Only the bars' spacing stretches, which is the part that should.

The viewBox is Height units tall and the canvas Height px, so y is one to one and the caps' half a BarWidth of overshoot can be taken off the line's reach exactly: no bar is cut by the canvas's edge.

Two shapes carry everything, the played bars and the rest, each one path. Progress moving only shifts subpaths from one to the other.

#### More peaks than bars

Bars says how many bars to draw. With more peaks than that, each bar takes the maximum of its bucket, never the mean: a clap in a quiet passage is one sample wide, and averaging would erase the one feature a listener scans the strip for. With fewer peaks than Bars, there is a bar per peak.

Go cannot measure the strip, so how many bars suit it is the caller's arithmetic, from the width it knows (hooks.UseWindow less its own insets). BarsFor does it, as CalendarHeatmap.WeeksFor does for weeks:

	w := comps.Waveform{Peaks: peaks, Progress: p}
	w.Bars = w.BarsFor(win.Width - 32)

#### Display only

Tapping a waveform to seek needs the tap's x position, which no event carries to Go. So this is a picture of progress and not a control, and AudioPlayer, which offers it through its Waveform field, keeps its slider for seeking.

#### Accessibility

One image: "Audio waveform, 40 percent played". AccessibilityLabel replaces the sentence. Beside a seek bar that already speaks the position, as in AudioPlayer, it is decoration, and Decorative hides it.

#### Theme roles read

	Played   Colors.Primary
	Rest     Colors.ControlBorderColor()

<small>[comps/waveform.go:76](https://github.com/rohanthewiz/grmob/blob/master/comps/waveform.go#L76)</small>

#### func (Waveform) BarsFor

```go
func (w Waveform) BarsFor(width float64) int
```

BarsFor is how many bars fit width px at BarWidth and Gap, at least 1. n bars take n·BarWidth + (n−1)·Gap.

<small>[comps/waveform.go:134](https://github.com/rohanthewiz/grmob/blob/master/comps/waveform.go#L134)</small>

#### func (Waveform) Render

```go
func (w Waveform) Render(ctx *core.Context) *core.Node
```

<small>[comps/waveform.go:142](https://github.com/rohanthewiz/grmob/blob/master/comps/waveform.go#L142)</small>

