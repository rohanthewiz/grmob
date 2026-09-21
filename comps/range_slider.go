package comps

import (
	"github.com/rohanthewiz/grmob/core"
)

// ConcernRangeSliderInert is raised, in debug builds only, when a RangeSlider
// has no OnChange and is not Disabled: two tracks that can be dragged and
// always spring back, which SliderRow's doc calls a meter drawn as a control.
const ConcernRangeSliderInert = "range-slider-inert"

// ConcernRangeSliderInverted is raised, in debug builds only, when Low is
// above High. The widget draws the two the right way round, so the screen is
// correct and the caller's state is not: the first drag would report a pair
// the caller never held. It is reported so the state gets fixed at its source.
const ConcernRangeSliderInverted = "range-slider-inverted"

// RangeSlider chooses a minimum and a maximum that cannot cross: a price
// filter, an age bracket, the hours a shop is open.
//
//	comps.RangeSlider{
//	    Title: "Price", Min: 0, Max: 200, Step: 5,
//	    Low: low.Get(), High: high.Get(),
//	    OnChange: func(l, h float64) { low.Set(l); high.Set(h) },
//	    Format:   func(v float64) string { return fmt.Sprintf("$%.0f", v) },
//	}
//
//	┌────────────────────────────────────────────┐
//	│ Price                           $20 – $80  │  the range, on the title line
//	│ Minimum                               $20  │
//	│ ▬▬▬●────────────────────────               │
//	│ Maximum                               $80  │
//	│ ▬▬▬▬▬▬▬▬▬▬▬●────────────────               │
//	└────────────────────────────────────────────┘
//
// # It is two sliders, on purpose
//
// One track with two thumbs is a node type no target here has, so it is not
// something a widget can compose (it is on the round-four blocked list). This
// is two SliderRows, and for a screen-reader user that is not a stopgap: it
// is the form the control takes on every platform. VoiceOver and TalkBack
// adjust one value per stop, and a native two-thumb slider is presented to
// them as two adjustable elements, which is exactly this. What the
// composition gives up is for the eye alone: the two values are not drawn on
// one line, so the title line states the range in words to make up for it.
//
// # The thumbs push each other
//
// Dragging Low past High carries High along with it, and the reverse:
//
//	Low 20  High 80     drag Low to 90   →   OnChange(90, 90)
//	Low 20  High 80     drag High to 10  →   OnChange(10, 10)
//
// The alternative, stopping a thumb at the other one, makes a range
// impossible to move as a whole: to shift 20–30 up to 60–70 the reader would
// have to know to move the maximum first. Pushing lets either order work. A
// caller who needs a minimum width (at least 10 apart) widens the pair in
// OnChange, where the rule belongs.
//
// # Whose values
//
// The caller's, both of them, as with SliderRow: the widget takes no hooks
// and may be rendered conditionally. It also inherits SliderRow's reporting:
// OnChange fires once when a drag ends, not under the finger, for the reasons
// that doc gives. The range on the title line therefore follows the last
// commit, while each thumb follows the finger.
//
// # Accessibility
//
// The whole is a core.RoleGroup named by Title. Each slider is named by its
// own label ("Minimum", "Maximum"; Labels replaces them), so a reader moving
// through hears "Price, group; Minimum, slider, 20".
//
// # Theme roles read
//
// Everything SliderRow reads, for each of the two rows and the title line.
type RangeSlider struct {
	// Title names the range. It is drawn on the first line, beside the range
	// in words, and is the group's accessible name. Empty draws no title line.
	Title string

	// Min and Max bound both tracks. Both zero gives 0..1, as SliderRow.
	Min float64
	Max float64

	// Step snaps both thumbs to multiples of it from Min. Zero is continuous.
	Step float64

	// Low and High are the caller's pair, each clamped into [Min, Max].
	Low  float64
	High float64

	// OnChange receives the whole pair, once, when either drag ends. It is
	// always ordered (low <= high): see "The thumbs push each other".
	OnChange func(low, high float64)

	// Format writes every number drawn: both readouts and the two ends of the
	// range on the title line. Nil gives SliderRow's default precision.
	Format func(float64) string

	// Labels are the two sliders' titles. An empty entry means "Minimum" or
	// "Maximum".
	Labels [2]string

	// Disabled greys both sliders and drops their reports.
	Disabled bool

	// Style is applied to the outer column after its defaults.
	Style []core.StyleProp
}

// Render draws the title line and the two rows.
func (r RangeSlider) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	if core.IsDebugMode() {
		if r.OnChange == nil && !r.Disabled {
			core.ReportConcern(ConcernRangeSliderInert,
				"RangeSlider has no OnChange and is not Disabled, so both thumbs spring back")
		}
		if r.Low > r.High {
			core.ReportConcern(ConcernRangeSliderInverted,
				"RangeSlider's Low is above its High; it is drawn the right way round, but the caller's state is inverted")
		}
	}

	low, high := r.ordered()

	items := make([]core.PropsAndChildren, 0, len(r.Style)+6)
	items = append(items,
		core.Padding(0),
		core.Gap(0),
		core.Width("100%"),
		core.AccessibilityRole(core.RoleGroup),
	)
	if r.Title != "" {
		items = append(items, core.AccessibilityLabel(r.Title))
	}
	items = append(items, asProps(r.Style)...)

	if r.Title != "" {
		items = append(items, ListRow{
			Title: r.Title,
			Trailing: core.Text(r.rangeText(low, high),
				core.UseStyle(t.Typography.Body),
				core.TextColor(t.Colors.TextSecondary),
				// The words repeat what the two sliders state as values, so a
				// screen reader is spared the third telling.
				core.AccessibilityHidden(),
			),
		})
	}

	var onLow, onHigh func(float64)
	if r.OnChange != nil {
		onLow = func(v float64) { r.OnChange(pushLow(v, high)) }
		onHigh = func(v float64) { r.OnChange(pushHigh(low, v)) }
	}
	items = append(items,
		r.slider(orDefault(r.Labels[0], "Minimum"), low, onLow),
		r.slider(orDefault(r.Labels[1], "Maximum"), high, onHigh),
	)
	return core.Column(items...).Render(ctx)
}

// slider is one of the two rows.
func (r RangeSlider) slider(title string, value float64, onChange func(float64)) core.View {
	return SliderRow{
		Title: title, Value: value,
		Min: r.Min, Max: r.Max, Step: r.Step,
		OnChange: onChange,
		Format:   r.Format,
		Disabled: r.Disabled,
	}
}

// ordered is the pair as drawn: each clamped into the range by SliderRow's
// own rule, and swapped if the caller handed them over inverted.
func (r RangeSlider) ordered() (low, high float64) {
	probe := SliderRow{Min: r.Min, Max: r.Max}
	probe.Value = r.Low
	_, _, low = probe.bounds()
	probe.Value = r.High
	_, _, high = probe.bounds()
	if low > high {
		low, high = high, low
	}
	return low, high
}

// rangeText is the title line's "20 – 80", in the numbers' own format.
func (r RangeSlider) rangeText(low, high float64) string {
	f := SliderRow{Min: r.Min, Max: r.Max, Step: r.Step, Format: r.Format}
	lo, hi := f.format(low), f.format(high)
	if lo == "" && hi == "" {
		return ""
	}
	// An en dash with spaces: "$20–$80" reads as a minus sign beside a
	// negative bound, and a range of negatives is a legitimate one.
	return lo + " – " + hi
}

// pushLow is the pair after Low is dragged to v: High is carried up with it
// when v passes it.
func pushLow(v, high float64) (float64, float64) {
	if v > high {
		return v, v
	}
	return v, high
}

// pushHigh is the pair after High is dragged to v: Low is carried down.
func pushHigh(low, v float64) (float64, float64) {
	if v < low {
		return v, v
	}
	return low, v
}
