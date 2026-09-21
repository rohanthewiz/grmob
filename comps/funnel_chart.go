package comps

import (
	"fmt"
	"math"

	"github.com/rohanthewiz/grmob/core"
)

// ConcernFunnelChartStageGrows is raised, in debug builds only, when a
// FunnelChart stage is larger than the one before it and AllowIncrease is not
// set. The stage is drawn as it is either way (see "A stage that grows"); the
// concern is that the usual cause is stages passed out of order, which draws
// a plausible funnel that says the wrong thing.
const ConcernFunnelChartStageGrows = "funnel-chart-stage-grows"

// FunnelStage is one step of a FunnelChart.
type FunnelStage struct {
	Label string
	Value float64

	// Color overrides the stage's colour.
	Color string
}

// FunnelChart draws how many of a population survive each step of a process:
// visitors, sign-ups, purchases. Each stage is a centred band whose width is
// its value, tapering into the next.
//
//	comps.FunnelChart{
//	    Subject:   "Checkout",
//	    ShowRates: true,
//	    Stages: []comps.FunnelStage{
//	        {Label: "Visited", Value: 1200},
//	        {Label: "Signed up", Value: 744},
//	        {Label: "Paid", Value: 93},
//	    },
//	}
//
// # Layout
//
//	┌───────────┬──────────────────────┬──────┬─────┐
//	│  Visited  │ ████████████████████ │ 1200 │     │  one band per stage,
//	│           │  ╲████████████████╱  │      │ 62% │  Height/n px each
//	│ Signed up │   ██████████████     │  744 │     │
//	│           │     ╲████████╱       │      │ 13% │  a rate sits on the
//	│      Paid │       ██             │   93 │     │  line between two bands
//	└───────────┴──────────────────────┴──────┴─────┘
//	  names        core.Canvas, stretched  values  ShowRates
//
// Canvas has no text, so the names, the values and the rates are columns of
// Text beside it, the arrangement a horizontal BarChart uses. Every column is
// Height px tall and cut into px bands by the same arithmetic as the drawing,
// so a name is level with its band on every target. The rates column is the
// same bands shifted down by half of one, which puts each rate level with
// the boundary it describes:
//
//	values   │ band 0 │ band 1 │ band 2 │
//	rates    │ ½ │ 0→1    │ 1→2    │ ½ │
//
// # The shape of a band
//
// A stage's band is a trapezoid: as wide at the top as its own value and at
// the bottom as the next stage's, so the slope between two stages is the
// drop between them. The last stage has no next and is a rectangle. Widths
// are shares of the largest stage, which is usually the first.
//
// # One colour, fading
//
// The stages are one population at successive moments, not categories, so
// they are one hue: the first chart colour, stepping down in alpha from
// solid to funnelMinAlpha. The categorical palette would say the stages are
// different kinds of thing. A stage's Color, or Colors, overrides it.
//
// # A stage that grows
//
// A stage larger than the one before it is drawn as it is: its band is wider
// than the one above, and the band above flares out to meet it. It is not
// clamped, because real funnels have them (people who re-enter at a later
// step, a stage counted over a longer window) and a chart that hid the
// growth would misreport the data. Its rate reads over 100%.
//
// Far more often it is a mistake, stages passed out of order, so debug
// builds report ConcernFunnelChartStageGrows. AllowIncrease says the data is
// meant and silences it.
//
// # Accessibility
//
// One element: "Checkout: Visited 1200; Signed up 744, 62% of the step
// before; Paid 93, 13% of the step before; 8% overall."
//
// # Theme roles read
//
//	Bands    Colors.ChartColors()[0], fading in alpha
//	Names    Colors.TextSecondary
//	Values   Colors.TextPrimary
//	Rates    Colors.TextSecondary
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

const (
	// funnelMinAlpha is the last stage's opacity when the stages fade. A
	// third keeps the narrowest band, often a sliver, visible on Surface.
	funnelMinAlpha = 0.35

	// funnelSeamPx is the gap between bands, in px: enough to read as steps
	// rather than one tapering blob.
	funnelSeamPx = 2.0
)

func (c FunnelChart) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	n := len(c.Stages)
	h := c.Height
	if h <= 0 {
		h = math.Max(80, 40*float64(n))
	}
	format := c.Format
	if format == nil {
		format = formatValue
	}

	values := make([]float64, n)
	top := 0.0
	for i, s := range c.Stages {
		values[i] = funnelValue(s.Value)
		top = math.Max(top, values[i])
	}
	if core.IsDebugMode() && !c.AllowIncrease {
		for i := 1; i < n; i++ {
			if values[i] > values[i-1] {
				core.ReportConcern(ConcernFunnelChartStageGrows, fmt.Sprintf(
					"FunnelChart stage %d (%q, %s) is larger than stage %d (%q, %s): it is drawn "+
						"as given, but stages out of order are the usual cause. Set AllowIncrease "+
						"if the data is meant",
					i+1, c.Stages[i].Label, formatValue(values[i]),
					i, c.Stages[i-1].Label, formatValue(values[i-1])))
				break
			}
		}
	}

	colors := c.stageColors(t)
	shapes := make([]core.Shape, 0, n)
	if n > 0 {
		band := chartView / float64(n)
		// The seam comes off the bottom of every band but the last. The
		// canvas is h px tall and stretched, so a px is chartView/h units.
		seam := funnelSeamPx * chartView / h
		width := func(i int) float64 {
			if top <= 0 {
				return 0
			}
			return values[i] / top * chartView
		}
		for i := range c.Stages {
			// One shape per stage, drawn or not, so a stage's child slot
			// never moves while values come and go.
			w0, w1 := width(i), width(i)
			y0, y1 := float64(i)*band, float64(i+1)*band
			if i < n-1 {
				w1 = width(i + 1)
				y1 -= seam
			}
			if w0 <= 0 && w1 <= 0 {
				shapes = append(shapes, core.Shape{})
				continue
			}
			mid := chartView / 2
			p := core.NewPath().
				MoveTo(mid-w0/2, y0).LineTo(mid+w0/2, y0).
				LineTo(mid+w1/2, y1).LineTo(mid-w1/2, y1).Close()
			shapes = append(shapes, core.Shape{Path: p, Fill: colors[i]})
		}
	}
	canvas := core.Canvas(chartView, chartView, shapes, core.CanvasStretch, core.Height(px(h)))

	labelWidth := c.LabelWidth
	if labelWidth <= 0 {
		labelWidth = 96
	}
	names := make([]string, n)
	shown := make([]string, n)
	for i, s := range c.Stages {
		names[i], shown[i] = s.Label, format(values[i])
	}

	row := make([]core.PropsAndChildren, 0, 10+len(c.Style))
	row = append(row,
		core.Padding(0),
		core.Gap(8),
		core.AlignItemsProp(core.AlignItemsStart),
		core.AccessibilityRole(core.RoleImg),
		core.AccessibilityLabel(c.label(values, format)),
	)
	row = append(row, asProps(c.Style)...)
	row = append(row,
		bandColumn(t, names, repeatWeight(1, n), h, core.AlignEnd, core.MaxWidth(px(labelWidth))),
		core.Column(
			core.Padding(0),
			core.FlexGrow(1),
			core.FlexBasis("0"),
			core.MinWidth("0px"),
			core.AccessibilityHidden(),
			canvas,
		),
		funnelColumn(t, shown, repeatWeight(1, n), h, t.Colors.TextPrimary),
	)
	if c.ShowRates && n > 1 {
		// Half a band, a band per boundary, half a band: see "Layout".
		texts := make([]string, 0, n+1)
		weights := make([]float64, 0, n+1)
		texts, weights = append(texts, ""), append(weights, 0.5)
		for i := 1; i < n; i++ {
			texts, weights = append(texts, funnelRate(values[i-1], values[i])), append(weights, 1)
		}
		texts, weights = append(texts, ""), append(weights, 0.5)
		row = append(row, funnelColumn(t, texts, weights, h, t.Colors.TextSecondary))
	}
	return core.Row(row...).Render(ctx)
}

// funnelValue is a stage's drawable value: what cannot be a head count reads
// as zero.
func funnelValue(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
		return 0
	}
	return v
}

// funnelRate is the later stage as a whole percentage of the earlier one, or
// "" when the earlier one is zero and the rate has no meaning.
func funnelRate(from, to float64) string {
	if from <= 0 {
		return ""
	}
	return formatValue(math.Round(to/from*100)) + "%"
}

// stageColors is each stage's fill. See "One colour, fading".
func (c FunnelChart) stageColors(t *core.Theme) []string {
	n := len(c.Stages)
	out := make([]string, n)
	base := seriesColor(chartPalette(t), "", 0)
	for i, s := range c.Stages {
		switch {
		case s.Color != "":
			out[i] = s.Color
		case len(c.Colors) > 0:
			out[i] = c.Colors[i%len(c.Colors)]
		default:
			// Linear from solid to funnelMinAlpha across the stages. One
			// stage alone is solid.
			alpha := 1.0
			if n > 1 {
				alpha = 1 - (1-funnelMinAlpha)*float64(i)/float64(n-1)
			}
			out[i] = withAlpha(base, fmt.Sprintf("%02x", int(math.Round(alpha*255))))
		}
	}
	return out
}

// funnelColumn is bandColumn in a given ink and end-aligned: the values and
// the rates beside the drawing. It exists because bandColumn's ink is fixed
// at TextSecondary, and a funnel's values are its data, not its axis.
func funnelColumn(t *core.Theme, texts []string, weights []float64, h float64, ink string) core.View {
	total := 0.0
	for _, w := range weights {
		total += w
	}
	items := make([]core.PropsAndChildren, 0, len(texts)+4)
	items = append(items, core.Padding(0), core.Gap(0), core.Height(px(h)), core.AccessibilityHidden())
	if total <= 0 {
		return core.Column(items...)
	}
	for k, text := range texts {
		items = append(items, core.Column(
			core.Padding(0),
			core.Height(px(h*weights[k]/total)),
			core.Justify(core.JustifyCenter),
			chartLabelText(t, text, core.AlignEnd, ink),
		))
	}
	return core.Column(items...)
}

// label is each stage with its step rate, then the overall rate from the
// first stage to the last.
func (c FunnelChart) label(values []float64, format func(float64) string) string {
	if c.AccessibilityLabel != "" {
		return c.AccessibilityLabel
	}
	var parts []string
	for i, s := range c.Stages {
		name := s.Label
		if name == "" {
			name = "stage " + formatValue(float64(i+1))
		}
		part := name + " " + format(values[i])
		if i > 0 {
			if rate := funnelRate(values[i-1], values[i]); rate != "" {
				part += ", " + rate + " of the step before"
			}
		}
		parts = append(parts, part)
	}
	if len(parts) == 0 {
		parts = append(parts, "no data")
	} else if n := len(values); n > 2 {
		// With two stages the overall rate is the one step rate again.
		if rate := funnelRate(values[0], values[n-1]); rate != "" {
			parts = append(parts, rate+" overall")
		}
	}
	return summaryPrefix(c.Subject) + joinSentences(parts)
}
