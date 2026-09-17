package comps

import (
	"math"
	"strconv"

	"github.com/rohanthewiz/grmob/core"
)

// SliderRow is the settings-screen row for a number in a range: the title and
// the current reading on one line, the track across the width under them.
//
//	┌────────────────────────────────────────────┐
//	│ 🔆  Brightness                        72%  │
//	│     ▬▬▬▬▬▬▬▬▬▬▬▬▬▬●─────────               │
//	└────────────────────────────────────────────┘
//
//	comps.SliderRow{
//	    Title: "Brightness", Min: 0, Max: 100, Step: 1,
//	    Value:    level.Get(),
//	    OnChange: level.Set,
//	    Format:   func(v float64) string { return fmt.Sprintf("%.0f%%", v) },
//	}
//
// It completes the settings-row family: SwitchRow and CheckboxRow for a
// boolean, SelectRow for one of a few named values, and this for a number.
//
// # Two callbacks, and why OnChange is the one that fires last
//
// core.Slider reports twice over: continuously under the finger, and once more
// when the finger lifts. This row wires the caller's OnChange to the *second*
// of those, and leaves the first unwired unless OnDrag is set.
//
// That is not only about cost at the bridge. A Set on any core.State requests
// a render of the whole tree, so a row that fed its caller's state on every
// tick would put a full render pass between each pixel of the drag — for a
// value the person has not finished choosing. A settings slider is adjusted
// and let go; what is downstream of it is usually a write to disk, a device
// call or a request, and none of those wants sixty of itself per second.
//
// The visible cost is that the readout does not follow the finger: it shows
// what was last committed until the drag ends. The thumb *does* follow it —
// every renderer draws the finger's position while dragging and Go's value
// otherwise (see core.Slider) — so the control is never sluggish, and only the
// number beside the title lags. A caller who wants the number live opts into
// the cost explicitly, which is what OnDrag is for:
//
//	draft := core.NewState(ctx, -1.0) // -1: not dragging
//	shown := level.Get()
//	if draft.Get() >= 0 {
//	    shown = draft.Get()
//	}
//	comps.SliderRow{
//	    Title: "Brightness", Max: 100, Value: shown,
//	    OnDrag:   func(v float64) { draft.Set(v) },
//	    OnChange: func(v float64) { draft.Set(-1); level.Set(v) },
//	}
//
// The draft is the caller's rather than the row's on purpose: holding it here
// would make SliderRow a hook-owning widget — rendered unconditionally, in a
// stable position, every pass, as DatePicker and SelectRow must be — and would
// charge every row in the framework the render-per-tick it was written to
// avoid, to make one of them look livelier. This row takes no hooks at all and
// is free to be conditional.
//
// # The row is not a tap target
//
// SwitchRow makes the whole row tappable because a switch has one other state
// to go to and a row-sized target is easier to hit than a switch-sized one. A
// slider has no such "other" value: a tap on the row would have to invent one,
// and the only honest candidate — the value under the tap — cannot be computed
// in Go, which sees no coordinates. So the row carries no OnTap and no role,
// and the slider is the control a reader is looking for, which is the rule
// ListRow states for every row with a control in it.
//
// # Range and step
//
// Max at or below Min is turned into Min..Min+1 with the value pinned to the
// start, which is exactly what core.Slider does with a degenerate range — done
// here as well so the readout and the thumb cannot disagree about what a
// zero-value SliderRow is showing. Step snaps the thumb to multiples of it
// from Min, and also decides how many decimals the default readout writes.
//
// # Theme roles read
//
// Everything ListRow reads, plus Colors.TextSecondary for the readout.
type SliderRow struct {
	// Title is the setting's name, and the slider's accessible label.
	Title string

	// Subtitle is the quieter second line under the title, and the slider's
	// accessibility hint.
	Subtitle string

	// Leading is an optional icon or avatar before the text, as in ListRow.
	Leading core.View

	// Value is the caller's current number, clamped into [Min, Max].
	Value float64

	// Min and Max bound the track. Both zero gives 0..1; see "Range and step".
	Min float64
	Max float64

	// Step snaps the thumb to multiples of it from Min. Zero is continuous.
	Step float64

	// OnChange receives the final value, once, when the drag ends. It is a
	// setter: apply the value you are given. Nil leaves a track that can be
	// dragged and always springs back, which is a read-only meter drawn as a
	// control — prefer ProgressBar or Gauge for that, and Disabled for a
	// setting that is temporarily unavailable.
	OnChange func(float64)

	// OnDrag receives every value under the finger. Nil — the default — means
	// nothing is reported until the drag ends. Setting it costs a render pass
	// per tick; see the type comment for the one shape that is worth it.
	OnDrag func(float64)

	// Format writes the readout beside the title. Nil writes the number at the
	// precision Step is written at (0.5 gives one decimal, 0.25 two), or, with
	// no Step, at the precision the range's span suggests — see
	// sliderRowDecimals for both tables. Returning "" draws no readout at all,
	// which is how a row whose number means nothing to a reader ("Contrast")
	// hides it.
	Format func(float64) string

	// Disabled greys the slider and drops its reports.
	Disabled bool

	// Style is passed to the underlying ListRow and so beats its defaults.
	Style []core.StyleProp
}

// Render builds the two-line row described in the type doc.
func (r SliderRow) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()
	min, max, value := r.bounds()

	style := make([]core.StyleProp, 0, len(r.Style)+1)
	style = append(style, r.Style...)
	// After the caller's styles, as every row in this family orders it: being
	// inert is not a look, so a Style override must not re-enable the row.
	if r.Disabled {
		style = append(style, core.Disabled(true))
	}

	return ListRow{
		Leading: r.Leading,
		// The title, the readout and the track go into the row's growing
		// middle column rather than beside it, and that is what aligns them:
		// the track then starts where the title starts and ends where the
		// readout ends, on any theme, with no arithmetic against the row's own
		// padding. The Box is the one node this costs — ListRow.Content is a
		// single view and there are two things to put in it.
		Content: core.Box(
			core.Padding(0),
			core.Gap(float64(t.Spacing.XS)),
			r.header(t, value),
			r.track(min, max, value),
		),
		Style: style,
	}.Render(ctx)
}

// header is the title line: a nested ListRow, so the typography and the
// trailing pin come from the same place every other row in the package gets
// them. Padding(0) drops the theme Row base's inset, which this row is already
// inside one copy of.
func (r SliderRow) header(t *core.Theme, value float64) core.View {
	return ListRow{
		Title:    r.Title,
		Subtitle: r.Subtitle,
		Trailing: r.readout(t, value),
		Style:    []core.StyleProp{core.Padding(0)},
	}
}

// readout is the number beside the title, or nil when there is none to draw.
func (r SliderRow) readout(t *core.Theme, value float64) core.View {
	text := r.format(value)
	if text == "" {
		return nil
	}
	return core.Text(text,
		core.UseStyle(t.Typography.Body),
		core.TextColor(t.Colors.TextSecondary),
	)
}

// track is the control itself.
func (r SliderRow) track(min, max, value float64) core.View {
	// Width is stated rather than left to the container's stretch: a Compose
	// child sizes to its content unless it is told to fill, so a bare slider
	// in a column is a short one on Android and a full-width one everywhere
	// else. examples/mobileapp's seek bar states it for the same reason.
	props := []core.PropsAndChildren{core.Width("100%")}
	props = append(props, controlProps(r.Title, r.Subtitle, r.Disabled)...)
	if r.Step > 0 {
		props = append(props, core.SliderStep(r.Step))
	}
	if r.OnChange != nil && !r.Disabled {
		props = append(props, core.OnSliderChangeEnd(r.OnChange))
	}

	// The continuous handler is registered only when OnDrag asks for it, so
	// an ordinary row puts no per-tick callback in the registry and the
	// renderers advertise no continuous channel for it.
	var onDrag func(float64)
	if r.OnDrag != nil && !r.Disabled {
		onDrag = r.OnDrag
	}
	return core.Slider(value, min, max, onDrag, props...)
}

// bounds resolves the range and clamps the value, mirroring core.Slider's own
// rule so the readout cannot report a number the thumb is not on.
func (r SliderRow) bounds() (min, max, value float64) {
	min, max, value = r.Min, r.Max, r.Value
	if max <= min {
		max = min + 1
		value = min
		return min, max, value
	}
	if value < min {
		value = min
	}
	if value > max {
		value = max
	}
	return min, max, value
}

// format writes the readout. The caller's Format wins outright, including when
// it returns "".
func (r SliderRow) format(value float64) string {
	if r.Format != nil {
		return r.Format(value)
	}
	min, max, _ := r.bounds()
	return strconv.FormatFloat(value, 'f', sliderRowDecimals(r.Step, max-min), 64)
}

// sliderRowDecimals picks how many decimals the default readout writes.
//
// The step decides it when there is one, because the step is exactly the
// smallest difference the control can express: a row stepping by 0.5 that
// rounded to whole numbers would show the same reading for two adjacent
// positions, and one stepping by 0.25 would show the same reading for four.
// So the answer is not a bracket the step falls into but the step's own
// precision — how many decimals it takes to write the step itself.
//
// With no step there is no such evidence and the span stands in for it, as the
// only other thing known about the scale:
//
//	span          decimals   a reading
//	───────────   ────────   ─────────
//	  ≥ 10           0       72
//	  > 1            1       3.5
//	  ≤ 1            2       0.72
//
// The bottom row is why the brackets are not the obvious ≥1 / ≥0.1: a 0..1
// range is the common continuous one (a volume, an opacity), and one decimal
// gives it eleven distinct readings — so the number would sit still across a
// tenth of the thumb's travel, which reads as a control that has stopped
// responding rather than as a rounded one.
func sliderRowDecimals(step, span float64) int {
	if step > 0 {
		return sliderRowStepDecimals(step)
	}
	switch {
	case span >= 10:
		return 0
	case span > 1:
		return 1
	default:
		return 2
	}
}

// sliderRowStepDecimals is the smallest d for which step·10^d is a whole
// number — the precision the step is written at.
//
// Capped at three, which is both where a readout stops being read and where
// binary floating point stops answering the question reliably: a step like
// 0.001 is not exactly representable, so the comparison is against a tolerance
// rather than for equality, and a loop that kept multiplying by ten would
// eventually find a "whole" number that is only rounding error.
func sliderRowStepDecimals(step float64) int {
	const maxDecimals = 3
	scaled := step
	for d := 0; d < maxDecimals; d++ {
		if math.Abs(scaled-math.Round(scaled)) < 1e-9 {
			return d
		}
		scaled *= 10
	}
	return maxDecimals
}
