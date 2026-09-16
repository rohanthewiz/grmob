package comps

import (
	"math"
	"strconv"

	"github.com/rohanthewiz/grmob/core"
)

// Gauge draws one value within a range as an arc filling a track, with the
// value written in the middle.
//
//	comps.Gauge{Value: 72, Max: 100, Label: "Battery", ValueText: "72%"}
//
// # Geometry
//
// The track is an arc of Sweep degrees, opening at the bottom and symmetric
// about twelve o'clock, so a 240° gauge runs from about eight o'clock round to
// four:
//
//	   ╭───────╮
//	 ╱     ▲     ╲       start = 90° + (360° − Sweep)/2
//	│     72%     │      measured clockwise from three o'clock,
//	 ╲  Battery  ╱       as core.Path.Arc takes angles
//	  ↑         ↑
//	start      end
//
// Both arcs are strokes with round caps rather than filled sectors, because a
// stroke's width is in px: Thickness means the same on every target and at
// every Size. The canvas is CanvasFit in a Size × Size box, so the viewBox
// scale is known (Size/100 px per unit) and the radius is pulled in by half the
// stroke so the rounded ends stay inside the box.
//
// The box is square even though a 240° arc leaves its bottom quarter empty,
// because the value text is centred on the arc's centre, and a ZStack centres
// its layers in its box. Trimming the box would move the text off the centre;
// placing it by measured offsets is what the rest of the framework avoids.
//
// # Accessibility
//
// One element, "Battery, 72%": the Label and the ValueText (or the value and
// the range, "72 of 100", when no ValueText is given).
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

func (g Gauge) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	size := g.Size
	if size <= 0 {
		size = 160
	}
	thickness := g.Thickness
	if thickness <= 0 {
		thickness = size / 10
	}
	sweep := g.Sweep
	if sweep <= 0 {
		sweep = 240
	}
	sweep = math.Max(60, math.Min(360, sweep))
	color := g.Color
	if color == "" {
		color = t.Colors.Primary
	}
	track := g.TrackColor
	if track == "" {
		track = t.Colors.BorderColor()
	}

	frac := g.fraction()

	const c = chartView / 2
	// Half the stroke, converted from px to viewBox units, keeps the round
	// caps and the outer edge inside the box.
	r := c - thickness/2*(chartView/size)
	start := 90 + (360-sweep)/2

	var fill *core.Path
	if frac > 0 {
		fill = core.NewPath().Arc(c, c, r, start, sweep*frac)
	}
	ends := core.CapRound
	if sweep >= 360 {
		// A full ring has no ends, and round caps on a partial fill of one
		// would poke past the start mark.
		ends = core.CapButt
	}
	shapes := []core.Shape{
		{Path: core.NewPath().Arc(c, c, r, start, sweep), Stroke: track, StrokeWidth: thickness, Cap: ends},
		// Present even when empty, so the fill keeps its child slot as the
		// value moves through zero.
		{Path: fill, Stroke: color, StrokeWidth: thickness, Cap: ends},
	}

	dim := px(size)
	centre := []core.PropsAndChildren{
		core.Padding(0),
		core.Gap(0),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.MaxWidth(px(size - thickness*2.5)),
		core.AccessibilityHidden(),
		core.Text(g.valueText(),
			core.FontSize(max(14, size*0.18)),
			core.FontWeight(core.Bold),
			core.TextColor(t.Colors.TextPrimary),
			core.Align(core.AlignCenter),
		),
	}
	if g.Label != "" {
		centre = append(centre, core.Text(g.Label,
			core.FontSize(max(chartLabelSize, size*0.08)),
			core.TextColor(t.Colors.TextSecondary),
			core.Align(core.AlignCenter),
		))
	}

	items := make([]core.PropsAndChildren, 0, 8+len(g.Style))
	items = append(items,
		core.Width(dim),
		core.Height(dim),
		core.AccessibilityRole(core.RoleImg),
		core.AccessibilityLabel(g.label()),
	)
	for _, sp := range g.Style {
		items = append(items, sp)
	}
	items = append(items,
		core.Canvas(chartView, chartView, shapes, core.Width(dim), core.Height(dim)),
		core.Column(centre...),
	)
	return core.ZStack(items...).Render(ctx)
}

// bounds is the effective range; see Min and Max.
func (g Gauge) bounds() (lo, hi float64) {
	if g.Max > g.Min {
		return g.Min, g.Max
	}
	return 0, 100
}

// fraction is the value's position in the range, clamped to [0, 1].
func (g Gauge) fraction() float64 {
	if math.IsNaN(g.Value) {
		return 0
	}
	lo, hi := g.bounds()
	return math.Max(0, math.Min(1, (g.Value-lo)/(hi-lo)))
}

func (g Gauge) valueText() string {
	if g.ValueText != "" {
		return g.ValueText
	}
	if math.IsNaN(g.Value) {
		return "–"
	}
	return strconv.FormatFloat(roundTo(g.Value, 1), 'f', -1, 64)
}

func (g Gauge) label() string {
	if g.AccessibilityLabel != "" {
		return g.AccessibilityLabel
	}
	value := g.ValueText
	if value == "" {
		if math.IsNaN(g.Value) {
			value = "no value"
		} else {
			_, hi := g.bounds()
			value = g.valueText() + " of " + formatValue(hi)
		}
	}
	if g.Label == "" {
		return value
	}
	return g.Label + ", " + value
}
