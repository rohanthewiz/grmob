package comps

import (
	"time"

	"github.com/rohanthewiz/grmob/core"
)

// ConcernDateRangePickerInert is raised, in debug builds only, when a
// DateRangePicker has no OnChange. The sheet still opens and the days still
// take taps, and the range they complete is handed to nobody — so the grid
// reopens on the old range every time and the field never changes. On screen
// that is indistinguishable from a picker nobody has finished using, which is
// the bar PINInput's own inert case is reported against.
const ConcernDateRangePickerInert = "date-range-picker-inert"

// dateRangeSeparator joins the two dates in the trigger's summary when the
// caller names none. An en dash with air on both sides, which is what a range
// of dates is set with; a hyphen is for compounds and reads as one.
const dateRangeSeparator = " – "

// DateRangePicker is a two-date field: a tappable summary of the chosen span
// that opens a Calendar in a modal sheet and closes again on the tap that
// completes the range.
//
//	┌────────────────────────────────┐        ┌───────────────────────────┐
//	│ Sep 14, 2026 – Sep 20, 2026 📅 │  tap → │ Stay dates      Clear  ✕  │
//	└────────────────────────────────┘        │  ‹   September 2026    ›  │
//	                                          │  Su Mo Tu We Th Fr Sa     │
//	                                          │  … [14]▓15▓▓16▓ … [20] …  │
//	                                          └───────────────────────────┘
//
//	comps.FormField{
//	    Label: "Stay dates",
//	    Input: comps.DateRangePicker{
//	        Start:    stay.Get().from,
//	        End:      stay.Get().to,
//	        OnChange: func(from, to time.Time) { stay.Set(span{from, to}) },
//	        Calendar: comps.Calendar{Today: today, Min: today},
//	    },
//	}
//
// It is DatePicker's shape with one date more and one rule more, and both
// halves of that are deliberate: the trigger, the sheet, the two ways out and
// the Calendar-as-template are the same, so a form holding one of each looks
// like a form rather than like two widgets.
//
// # One rule makes the whole protocol
//
// The sheet's grid reports one tapped day at a time, as Calendar always does.
// What turns that into a range is a single piece of state the widget owns — a
// *pending* start — and one rule over it:
//
//	no pending start   this tap becomes the pending start
//	a pending start    this tap is the other end; the range is reported
//
// Everything a range picker is usually specified with falls out of those two
// lines and needs no case of its own:
//
//   - **The third tap starts a new range.** Completing a range clears the
//     pending start, so the next tap finds none and begins again. There is no
//     "is this nearer the start or the end" arithmetic, and no tap that means
//     something different depending on where it lands.
//   - **Tapping the same day twice is a one-day range**, start and end on one
//     cell, because the second tap is the other end wherever it falls.
//   - **The second tap may be the earlier one.** The pair is ordered before it
//     is reported, so OnChange always receives start ≤ end. The alternative —
//     treating an earlier second tap as a restart — throws away a tap the
//     reader made on purpose, and "the other end" is not a claim about which
//     end; a reader who wanted to restart has the third tap for it.
//
// # Why the pending start is the widget's and the range is not
//
// This is the package's third state-owning widget, after Accordion and
// DatePicker, and it owns one piece more than DatePicker: the sheet is open,
// the month being browsed, and the half-made range.
//
// That third one is state no application wants, which is the test SliderRow
// states and this passes: a form's field is a span of days or it is nothing,
// and "from the 14th, no end yet" is a value it would have to invent a way to
// hold and a way to draw. Worse, holding it in the caller would make closing
// the sheet mid-pick *destructive* — the first tap would already have
// overwritten the range the reader opened the sheet to look at, and the
// backdrop, the ✕ and the back gesture would all be traps. Owning it keeps the
// promise DatePicker's sheet makes: a reader who opens this to check which
// week they booked can leave having changed nothing.
//
// So OnChange fires once per completed range, never mid-pick, and the picker
// inherits the hook rule with it — render it unconditionally, in a stable
// position, every pass.
//
// # What the sheet shows while the range is half made
//
// The pending start, alone, as one lit day with no band: Calendar draws
// exactly that for a RangeStart with no RangeEnd. The caller's own range is
// out of the grid from the first tap, which is the feedback that a new one has
// begun — there is no moment where the old span and the new start are both lit
// and the reader has to work out which is which.
//
// # Picking closes it, and there is no Done
//
// The tap that completes the range is the tap that finishes, as DatePicker's
// single tap is. What the sheet carries instead are the ways *out* — the
// backdrop, the ✕, and Clear when the field is clearable — and all three
// discard the pending start.
//
// # Theme roles read
//
//	Trigger   Components.Input — the same frame every other field in the form has
//	Summary   Typography.Body over TextPrimary, or TextSecondary for the placeholder
//	Sheet     Card, over the Modal's own scrim
//	Band      Colors.Primary, thinned — see Calendar's "A range of days is a band"
type DateRangePicker struct {
	// Start and End are the chosen span, inclusive; both zero shows
	// Placeholder. They are the caller's to hold, and the widget only ever
	// hands them back as a pair through OnChange.
	//
	// A Start with no End is drawn as that one day and summarized as that one
	// date. It is not a state this widget produces — an open-ended range is
	// not something a month grid can draw — and a caller modelling "from the
	// 14th onwards" wants a date field and a rule, not this.
	Start time.Time
	End   time.Time

	// OnChange fires once per completed range, with start ≤ end, and the
	// sheet closes. Both values are midday in the calendar's location — see
	// Calendar's "Dates in, dates out". Nil reports
	// ConcernDateRangePickerInert: the taps are still taken and the range they
	// complete goes nowhere.
	OnChange func(start, end time.Time)

	// OnClear puts a "Clear" button in the sheet that empties the field and
	// closes it. Nil renders no such button: whether a span is optional is the
	// form's question, not the picker's.
	OnClear func()

	// Placeholder is the summary's text when nothing is chosen. Empty leaves
	// the trigger blank but still tappable.
	Placeholder string

	// Format is the time layout each half of the summary is written in; empty
	// gives DatePicker's "Jan 2, 2006", so a range field and a date field on
	// one screen spell a date the same way.
	//
	// Go's time package names months in English only, so a layout carrying a
	// name is a layout in English; "2006-01-02" reads the same everywhere.
	Format string

	// Separator joins the two halves of the summary; empty gives " – ". It is
	// a field because the punctuation a range is set with is a typographic
	// convention that differs by locale, and because a numeric Format often
	// wants "/" or "→" rather than a dash that could be read as a minus.
	Separator string

	// Calendar is the template the sheet's grid is rendered from, exactly as
	// DatePicker uses it. Today, Min, Max, Marked, WeekStart, the three label
	// functions, Header and Style all apply. Month, OnMonthChange, RangeStart
	// and RangeEnd are overwritten, since those are what the picker drives;
	// Selected is cleared and Deselectable forced off, because a range picker
	// has no single selection and a tap here always means one of the two ends.
	Calendar Calendar

	// Title names the sheet. Empty leaves the heading row to the buttons
	// alone, which is right when the FormField label above the trigger has
	// already said what is being picked.
	Title string

	// ClearLabel and CloseLabel caption the sheet's two ways out; empty gives
	// "Clear" and a ✕ glyph.
	ClearLabel string
	CloseLabel string

	// Disabled marks the trigger inert: it neither opens nor announces itself
	// as actionable. The sheet's own grid is disabled with it, so a tap racing
	// the patch cannot land in an open picker.
	Disabled bool

	// AccessibilityLabel names the trigger; empty announces the summary text,
	// which is the span or the placeholder. AccessibilityHint describes what
	// tapping does.
	AccessibilityLabel string
	AccessibilityHint  string

	// Style is applied to the trigger row after its defaults. The sheet is
	// styled through Calendar.Style and the theme.
	Style []core.StyleProp
}

// Render allocates the three states and draws the trigger beside the sheet.
func (p DateRangePicker) Render(ctx *core.Context) *core.Node {
	// Three hooks, in a fixed order, on every pass. See "Why the pending start
	// is the widget's and the range is not".
	open := core.NewState(ctx, false)
	// Zero means "whatever month Calendar's anchor resolves to", which is now
	// the range's own start before it is Today. Only an arrow tap ever makes
	// it concrete, and opening the sheet resets it, so the picker always opens
	// on the month it is showing.
	month := core.NewState(ctx, time.Time{})
	// The first of the two taps, held until the second lands. Zero is "no
	// range is being made", which is also what the sheet opens in.
	pending := core.NewState(ctx, time.Time{})

	if core.IsDebugMode() && p.OnChange == nil {
		core.ReportConcern(ConcernDateRangePickerInert,
			"DateRangePicker has no OnChange, so a completed range is reported to nobody and the field can never change")
	}

	closeSheet := func() {
		open.Set(false)
		// A half-made range dies with the sheet. This is the whole of the
		// "leaving changes nothing" promise: the pending start is the only
		// thing a first tap touched, and every way out runs through here.
		pending.Set(time.Time{})
	}

	items := make([]core.PropsAndChildren, 0, 2)
	items = append(items,
		p.trigger(ctx, open, month),
		p.sheet(ctx, open, month, pending, closeSheet),
	)

	// Box, not Column, and the Modal a sibling of the trigger rather than a
	// child of it, both for DatePicker's reasons: a Column would arrive with
	// the theme's screen inset around one control in somebody's form, and an
	// overlay nested inside a bordered row would inherit the row's clip on any
	// target that honours overflow.
	return core.Box(items...).Render(ctx)
}

// trigger is the summary the field shows when the sheet is closed.
func (p DateRangePicker) trigger(ctx *core.Context, open core.State[bool], month core.State[time.Time]) core.View {
	t := ctx.Theme()

	summary := p.Placeholder
	ink := t.Colors.TextSecondary
	if span := p.summary(); span != "" {
		summary = span
		ink = t.Colors.TextPrimary
	}

	name := p.AccessibilityLabel
	if name == "" {
		name = summary
	}

	items := make([]core.PropsAndChildren, 0, len(p.Style)+10)
	items = append(items,
		// The theme's own field chrome, inherited rather than restated, so a
		// range field sitting between two text inputs is the same height, the
		// same fill and the same edge as they are. DatePicker's trigger gives
		// the long version of why the frame is not re-declared here.
		core.UseStyle(t.Components.Input),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.Gap(float64(t.Spacing.SM)),
	)
	items = append(items, asProps(p.Style)...)
	// After the caller's styles, as Button does it: whether the control is
	// inert is not a look, so a Style override must not re-enable it.
	if p.Disabled {
		items = append(items, core.Disabled(true))
	}
	items = append(items,
		core.AccessibilityLabel(name),
		core.AccessibilityHint(p.AccessibilityHint),
		// A button that opens a dialog. The Row is scenery on every target
		// until a role says otherwise, and the role is also what gives
		// aria-haspopup somewhere ARIA defines it.
		core.AccessibilityRole(core.RoleButton),
		core.AccessibilityHasPopup(core.PopupDialog),
	)

	// Registered even when disabled, and a no-op then: core.Disabled is what
	// makes every renderer refuse to dispatch, while a handler that is still
	// in the registry is what a tap already in flight lands on.
	items = append(items, core.OnClick(func() {
		if p.Disabled {
			return
		}
		// Reset the browsed month before showing the sheet, so the picker
		// opens on the month it is displaying rather than the last one
		// somebody paged to. The pending start needs no reset here: every way
		// out of the sheet runs through closeSheet, which clears it.
		month.Set(time.Time{})
		open.Set(true)
	}))

	items = append(items,
		core.Text(summary,
			core.UseStyle(t.Typography.Body),
			core.TextColor(ink),
			core.FlexGrow(1),
		),
		// Decoration: the trigger's spoken name already carries the span, and
		// "calendar" read out after it is noise.
		core.Text("📅", core.AccessibilityHidden()),
	)

	return core.Row(items...)
}

// summary writes the span the way the trigger shows it, or "" when there is no
// span at all.
//
// Both halves are written in full rather than sharing a year or a month —
// "Sep 14 – 20, 2026" is what a person would type, and it cannot be built from
// an arbitrary Format without taking the layout apart, which is a parser this
// widget is not. The long form is also the unambiguous one, which matters most
// in the place it is read: a field reporting what somebody is about to be
// charged for.
func (p DateRangePicker) summary() string {
	layout := p.Format
	if layout == "" {
		layout = datePickerFormat
	}
	sep := p.Separator
	if sep == "" {
		sep = dateRangeSeparator
	}

	switch {
	case !p.Start.IsZero() && !p.End.IsZero():
		return p.Start.Format(layout) + sep + p.End.Format(layout)
	case !p.Start.IsZero():
		// A half range the caller passed in; see the Start field's doc. One
		// date and no separator, because a trailing dash is a promise of a
		// second date that is not coming.
		return p.Start.Format(layout)
	case !p.End.IsZero():
		return p.End.Format(layout)
	}
	return ""
}

// sheet is the modal the trigger opens: a heading row of exits over the grid.
func (p DateRangePicker) sheet(ctx *core.Context, open core.State[bool], month, pending core.State[time.Time], closeSheet func()) core.View {
	t := ctx.Theme()

	cal := p.Calendar
	cal.Month = month.Get()
	cal.OnMonthChange = month.Set
	// A range picker has no single selection, and a leftover Selected on the
	// template would light a day that means nothing here — the grid draws it
	// with exactly the fill an endpoint has. Cleared rather than documented
	// away, for the reason DatePicker forces Deselectable off: the template is
	// a convenience, and the fields the picker drives are not negotiable.
	cal.Selected = time.Time{}
	cal.Deselectable = false

	cal.RangeStart, cal.RangeEnd = p.Start, p.End
	if start := pending.Get(); !start.IsZero() {
		// Half made: one lit day and no band, and the caller's own range out
		// of the grid. See "What the sheet shows while the range is half
		// made".
		cal.RangeStart, cal.RangeEnd = start, time.Time{}
	}

	if p.OnChange != nil && !p.Disabled {
		cal.OnSelect = func(d time.Time) {
			start := pending.Get()
			if start.IsZero() {
				// The first of the two taps. Nothing is reported yet, and
				// nothing the caller holds has changed.
				pending.Set(d)
				return
			}
			// The other end, wherever it fell. Ordered rather than rejected;
			// see "One rule makes the whole protocol". Both values are midday
			// in the same location — they are two cells of one grid — so
			// Before is a comparison of days here and not of clock times.
			from, to := start, d
			if to.Before(from) {
				from, to = to, from
			}
			pending.Set(time.Time{})
			p.OnChange(from, to)
			// The tap that completes is the tap that finishes.
			closeSheet()
		}
	} else {
		// An inert grid: the days are drawn, dimmed by Calendar's own rule for
		// a nil OnSelect, and nothing can be picked from them.
		cal.OnSelect = nil
	}

	head := make([]core.PropsAndChildren, 0, 6)
	head = append(head,
		core.Padding(0),
		core.PaddingVertical(t.Spacing.XS),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.Gap(float64(t.Spacing.SM)),
		// The title grows so the exits stay pinned to the trailing edge
		// whether or not there is a title to push them there.
		core.Box(
			core.FlexGrow(1),
			core.Text(p.Title,
				core.UseStyle(t.Typography.Body),
				core.FontWeight(core.Bold),
			),
		),
	)
	if p.OnClear != nil {
		label := p.ClearLabel
		if label == "" {
			label = "Clear"
		}
		head = append(head, Button{
			Label:    label,
			Emphasis: EmphasisGhost,
			OnTap: func() {
				p.OnClear()
				closeSheet()
			},
		})
	}
	closeLabel := p.CloseLabel
	if closeLabel == "" {
		closeLabel = "✕"
	}
	head = append(head, Button{
		Label:              closeLabel,
		Emphasis:           EmphasisGhost,
		AccessibilityLabel: "Close",
		OnTap:              closeSheet,
	})

	// The same width bounds DatePicker's sheet takes, and for the same reason:
	// the grid's cells are flex-basis-0, so inside an overlay that shrinks to
	// fit its content they would divide a width of nothing.
	return core.Modal(
		core.Visible(open.Get()),
		core.OnDismiss(closeSheet),
		core.ModalContent(core.Card(
			core.MinWidth(datePickerMinWidth),
			core.MaxWidth(datePickerMaxWidth),
			core.Row(head...),
			cal,
		)),
	)
}
