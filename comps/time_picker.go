package comps

import (
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/rohanthewiz/grmob/core"
)

// ConcernTimePickerInert is raised, in debug builds only, when a TimePicker
// has no OnChange. The pickers still open and still take a choice, and the
// choice goes nowhere: the field goes on reporting the time it was handed, so
// it either snaps back or, on a target that keeps the picked option until a
// patch says otherwise, shows a time the application does not hold. Neither
// looks broken on screen, which is the bar DateRangePicker's and PINInput's
// inert cases are reported against.
const ConcernTimePickerInert = "time-picker-inert"

// TimePicker is a time-of-day field: an hour, a minute and — on a 12-hour
// clock — an AM/PM, each a core.Select, in one row.
//
//	┌────────┐   ┌────────┐ ┌────────┐
//	│  9   ▾ │ : │ 30   ▾ │ │ AM   ▾ │    12-hour (the default)
//	└────────┘   └────────┘ └────────┘
//	┌────────┐   ┌────────┐
//	│ 09   ▾ │ : │ 30   ▾ │               Hour24
//	└────────┘   └────────┘
//
//	comps.FormField{
//	    Label: "Start time",
//	    Input: comps.TimePicker{
//	        Value:      start.Get(),
//	        OnChange:   start.Set,
//	        MinuteStep: 15,
//	        Label:      "Start time",
//	    },
//	}
//
// # No sheet, because nothing is ever half made
//
// The plan put this in the sheet DatePicker uses. The build did not, and the
// reason is the question DateRangePicker's pending start answered the other
// way.
//
// DatePicker needs a sheet because a month grid does not fit in a field, and
// its sheet needs no Done button because one tap is the whole choice. A time
// is two or three choices, so a sheet around them would need either a Done
// button and a draft held in the widget until it is pressed, or a live value
// that the ✕ cannot take back — the trap DateRangePicker's doc describes.
//
// Neither is needed, because a time has no invalid intermediate. Changing
// only the hour of 9:30 gives 10:30, which is a real time and exactly what the
// reader asked for; there is no "hour chosen, minute pending" state in the
// way "from the 14th, no end yet" is a state. Every change is a complete
// value, so every change is reported at once and there is nothing to confirm
// and nothing to discard.
//
// With no draft there is no hook, which makes this the stateless kind of
// widget, like Stepper and SliderRow: a TimePicker may be rendered
// conditionally, and a test may call Render bare.
//
// A sheet of core.Selects would also be a popup opening popups, since each
// Select is already the platform's own picker (a <select>, a SwiftUI Menu, a
// Material dropdown) — which is also why this is built from Selects rather
// than Steppers. Getting from 9:00 to 17:30 is two picks of two taps each,
// where a Stepper wants eight taps on the hour alone, and a Stepper clamps at
// its bounds where a clock has to wrap from 23 to 0.
//
// # What changes, and what does not
//
// OnChange receives Value with its hour and minute replaced, in Value's own
// location and on Value's own date — so a TimePicker can set the clock on the
// day a DatePicker chose, and the two fields compose into one time.Time. The
// seconds and nanoseconds are zeroed, because the field does not show them
// and a reported value should be one the field would draw.
//
// A zero Value is drawn as midnight. There is no "no time yet" state and no
// placeholder: a Select always shows one of its options, and inventing a blank
// one would put back the half-made value this design exists to avoid. A form
// where the time is optional puts a switch beside the field.
//
// In a location with daylight saving, a wall time that does not exist on
// Value's date (2:30 on the spring-forward night) is normalized by time.Date
// to the one that does, and the field then shows that.
//
// # A minute that is not on the step is kept, not rounded
//
// MinuteStep thins the minute list — 15 gives :00, :15, :30, :45. A Value
// whose minute is off the step (9:07 on a 15-minute picker, loaded from
// somewhere else) gets its own minute added to the list in order, so the field
// shows 9:07 rather than a time the application does not hold. Rounding it
// for display would be a lie about the value; rounding it through OnChange
// would be a write the reader never made. The extra option leaves the list as
// soon as another minute is chosen.
//
// # Accessibility
//
// The row is RoleGroup named by Label, with the whole time as its value, the
// shape Stepper has: a native reader arriving at the group hears "Start time,
// 9:30 AM" once, and the three pickers inside are for changing it. Each picker
// is named — "Hour", "Minute", "AM/PM" by default — because a picker's own
// text is only its current option, and "9, pop-up button" says nothing about
// which part of which time it is. The colon is decoration and hidden.
//
// # Theme roles read
//
//	Pickers    Components.Input, through core.Select — the frame of every field
//	Colon      Typography.Body over TextSecondary
//	Gaps       Spacing.XS inside the time, Spacing.SM before AM/PM
type TimePicker struct {
	// Value is the time shown. Only its hour and minute are read; its date
	// and location are carried through to OnChange. Zero shows midnight.
	Value time.Time

	// OnChange receives the new time on every pick — see "What changes, and
	// what does not". Nil reports ConcernTimePickerInert.
	OnChange func(time.Time)

	// Hour24 draws 00–23 and no AM/PM, as DigitalClock.Hour24 does; the
	// plan's TwentyFourHour became this name so the clock and the field that
	// sets one are configured with the same word. The hours are zero-padded
	// on a 24-hour picker and not on a 12-hour one, which is DigitalClock's
	// rule too.
	Hour24 bool

	// MinuteStep is the gap between the minutes offered; zero or negative
	// means every minute, and 60 or more means only :00. 5 and 15 are the
	// usual values for an appointment. See "A minute that is not on the step
	// is kept".
	MinuteStep int

	// Label names the group for assistive technology ("Start time"). It is
	// not drawn: a FormField around the picker is the visible label, as it is
	// for Stepper.
	Label string

	// HourLabel, MinuteLabel and PeriodLabel name the three pickers; empty
	// gives "Hour", "Minute" and "AM/PM".
	HourLabel, MinuteLabel, PeriodLabel string

	// AMLabel and PMLabel caption the two periods; empty gives "AM" and "PM".
	// They are fields because Go's time package spells the period in English
	// only, and a 12-hour clock is written differently in the languages that
	// use one.
	AMLabel, PMLabel string

	// Disabled disables all three pickers.
	Disabled bool

	// Style is applied to the row after its defaults.
	Style []core.StyleProp
}

// Option values for the period picker. They are identities rather than text,
// so a localised AMLabel does not change what OnChange is computed from.
const (
	timePeriodAM = "am"
	timePeriodPM = "pm"
)

// Render draws the pickers in one row. It takes no hooks; see "No sheet,
// because nothing is ever half made".
func (p TimePicker) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	if core.IsDebugMode() && p.OnChange == nil {
		core.ReportConcern(ConcernTimePickerInert,
			"TimePicker has no OnChange, so a picked hour or minute is reported to nobody and the field can never change")
	}

	hour, minute := p.Value.Hour(), p.Value.Minute()
	pm := hour >= 12

	items := make([]core.PropsAndChildren, 0, len(p.Style)+10)
	items = append(items,
		// The theme Row base pads a free-standing row; this one sits inside a
		// FormField, which already supplies the inset — Stepper's reasoning.
		core.Padding(0),
		core.Gap(float64(t.Spacing.XS)),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.AccessibilityRole(core.RoleGroup),
		core.AccessibilityValue(core.ValueRange{Text: p.spoken()}),
	)
	if p.Label != "" {
		items = append(items, core.AccessibilityLabel(p.Label))
	}
	items = append(items, asProps(p.Style)...)

	// The hour picker's value is the hour within its clock: 0–23 on a 24-hour
	// picker, and 0–11 on a 12-hour one, where 0 is *labelled* 12. Keeping the
	// value arithmetic (0–11) apart from the label (12, 1 … 11) is what makes
	// the 12-hour sum below one line: h24 = h12 + 12·pm, with no special case
	// for noon or midnight.
	hourValue := hour
	if !p.Hour24 {
		hourValue = hour % 12
	}

	items = append(items,
		p.picker(strconv.Itoa(hourValue), p.hourOptions(), orDefault(p.HourLabel, "Hour"),
			func(v int) (int, int) {
				if !p.Hour24 && pm {
					v += 12
				}
				return v, minute
			}),
		core.Text(":",
			core.UseStyle(t.Typography.Body),
			core.TextColor(t.Colors.TextSecondary),
			core.AccessibilityHidden(),
		),
		p.picker(strconv.Itoa(minute), p.minuteOptions(minute), orDefault(p.MinuteLabel, "Minute"),
			func(v int) (int, int) { return hour, v }),
	)

	if !p.Hour24 {
		period := timePeriodAM
		if pm {
			period = timePeriodPM
		}
		items = append(items, core.Select(period,
			[]core.SelectOption{
				core.Option(timePeriodAM, orDefault(p.AMLabel, "AM")),
				core.Option(timePeriodPM, orDefault(p.PMLabel, "PM")),
			},
			func(v string) {
				// The hour within the period is kept, so 9:30 AM becomes
				// 9:30 PM and not some other hour.
				h := hour % 12
				if v == timePeriodPM {
					h += 12
				}
				p.report(h, minute)
			},
			p.pickerProps(orDefault(p.PeriodLabel, "AM/PM"),
				// A little more air before the period than between hour and
				// minute: "9 : 30  AM" groups the digits as the clock does.
				core.MarginLeft(t.Spacing.XS))...,
		))
	}

	return core.Row(items...).Render(ctx)
}

// picker is the hour or the minute Select. next turns the picked option's
// number into the 24-hour hour and the minute to report, so the one closure
// that differs between the two is the only thing each call spells out.
func (p TimePicker) picker(value string, options []core.SelectOption, name string, next func(int) (int, int)) core.View {
	return core.Select(value, options, func(v string) {
		n, err := strconv.Atoi(v)
		if err != nil {
			// Every option value here is written by strconv.Itoa above, so a
			// value that does not parse did not come from this picker's own
			// list, and there is nothing honest to report for it.
			return
		}
		p.report(next(n))
	}, p.pickerProps(name)...)
}

// pickerProps are the props every one of the three Selects carries.
func (p TimePicker) pickerProps(name string, extra ...core.PropsAndChildren) []core.PropsAndChildren {
	props := make([]core.PropsAndChildren, 0, len(extra)+2)
	props = append(props, core.AccessibilityLabel(name))
	props = append(props, extra...)
	if p.Disabled {
		props = append(props, core.Disabled(true))
	}
	return props
}

// report hands the caller Value with its clock replaced. See "What changes,
// and what does not" for why the date and location stay and the seconds go.
func (p TimePicker) report(hour, minute int) {
	if p.OnChange == nil || p.Disabled {
		return
	}
	v := p.Value
	next := time.Date(v.Year(), v.Month(), v.Day(), hour, minute, 0, 0, v.Location())
	// A pick of the option already shown still arrives on some targets (a
	// native menu re-selecting its current item); it is not a change, and a
	// caller's handler should never see one — Stepper's rule.
	if next.Equal(v) {
		return
	}
	p.OnChange(next)
}

// hourOptions lists the hours of the clock in reading order: 0–23 on a
// 24-hour picker, labelled "00"…"23", and 12, 1 … 11 on a 12-hour one.
func (p TimePicker) hourOptions() []core.SelectOption {
	if p.Hour24 {
		opts := make([]core.SelectOption, 24)
		for h := range 24 {
			opts[h] = core.Option(strconv.Itoa(h), fmt.Sprintf("%02d", h))
		}
		return opts
	}
	opts := make([]core.SelectOption, 12)
	for h := range 12 {
		label := strconv.Itoa(h)
		if h == 0 {
			label = "12"
		}
		opts[h] = core.Option(strconv.Itoa(h), label)
	}
	return opts
}

// minuteOptions lists the minutes on the step, plus current when it is off the
// step — see "A minute that is not on the step is kept, not rounded".
func (p TimePicker) minuteOptions(current int) []core.SelectOption {
	step := p.MinuteStep
	if step <= 0 {
		step = 1
	}
	minutes := make([]int, 0, 60/step+1)
	for m := 0; m < 60; m += step {
		minutes = append(minutes, m)
	}
	if current%step != 0 {
		// Inserted in order rather than appended, so the list still reads
		// top to bottom as a clock does and the odd one out sits where the
		// reader would look for it.
		i, _ := slices.BinarySearch(minutes, current)
		minutes = slices.Insert(minutes, i, current)
	}

	opts := make([]core.SelectOption, len(minutes))
	for i, m := range minutes {
		opts[i] = core.Option(strconv.Itoa(m), fmt.Sprintf("%02d", m))
	}
	return opts
}

// spoken is the whole time as the group's value announces it, written the way
// DigitalClock writes it so the clock and the field read the same.
func (p TimePicker) spoken() string {
	digits, marker := clockDigits(p.Value, p.Hour24, false)
	if marker == "" {
		return digits
	}
	if p.Value.Hour() >= 12 {
		marker = orDefault(p.PMLabel, marker)
	} else {
		marker = orDefault(p.AMLabel, marker)
	}
	return digits + " " + marker
}
