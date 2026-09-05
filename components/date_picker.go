package components

import (
	"time"

	"github.com/rohanthewiz/grmob/core"
)

// DatePicker is a date field: a tappable summary of the chosen day that opens
// a Calendar in a modal sheet and closes again on the tap that picks.
//
//	┌──────────────────────────┐        ┌───────────────────────────┐
//	│ Jan 2, 2026          📅  │  tap → │ Event date      Clear  ✕  │
//	└──────────────────────────┘        │  ‹    January 2026     ›  │
//	                                    │  Su Mo Tu We Th Fr Sa     │
//	                                    │   …      [2]     …        │
//	                                    └───────────────────────────┘
//
//	components.FormField{
//	    Label: "Event date",
//	    Input: components.DatePicker{
//	        Selected: date.Get(),
//	        OnSelect: date.Set,
//	        Calendar: components.Calendar{Today: today, Min: today},
//	    },
//	}
//
// # It is the input, not the field
//
// There is no Label, Hint, Error or Required here, because FormField already
// owns all four and any input can sit in its slot. A picker that grew its own
// label would be a second way to write a form, worded and spaced slightly
// differently from every other field on the screen.
//
// # Two states, both of them the widget's
//
// This is the second widget in the package that owns state (Accordion is the
// other), and it owns exactly the two pieces no application ever wants: is the
// sheet open, and which month is being browsed inside it. Owning them means
// inheriting the hook rules — render a DatePicker unconditionally, in a stable
// position, every pass, the same obligation calling core.NewState directly
// carries. Calendar itself takes no hooks, so the grid on a screen is still
// free to be conditional; it is this packaging that is not.
//
// The browsed month is held as a *zero* time.Time until an arrow is tapped,
// rather than being seeded from the selection when the picker mounts. Seeding
// would go stale the moment the caller set a date from somewhere else — a
// "next Sunday" shortcut, a form loading a saved draft — and the picker would
// open on the month the screen first rendered in. A zero Month is exactly what
// Calendar's own anchor fallback reads as "follow Selected, then Today", so
// the two states cost one line between them and no re-derivation. Opening the
// sheet resets it, so the picker always opens on the month it is showing.
//
// # Picking closes it
//
// A single date has nothing to confirm: the tap that chooses is the tap that
// finishes, so there is no Done button standing between the two. What the
// sheet does carry is the ways *out* — the backdrop, the ✕, and Clear when the
// field is clearable — because a reader who opened it to look at March needs
// to leave without having changed anything.
type DatePicker struct {
	// Selected is the chosen day; zero shows Placeholder instead.
	Selected time.Time

	// OnSelect fires with the tapped day and the sheet closes. The value is
	// midday in the calendar's location — see Calendar's "Dates in, dates
	// out". Nil renders a summary that opens a sheet nothing can be picked
	// from, which is Calendar's inert display behind a field; use Disabled
	// for a field that should not open at all.
	OnSelect func(time.Time)

	// OnClear puts a "Clear" button in the sheet that empties the field and
	// closes it. Nil renders no such button: whether a date is optional is
	// the form's question, not the picker's, and a widget that always offered
	// to clear a required field would be offering an invalid state.
	OnClear func()

	// Placeholder is the summary's text when nothing is selected. Empty
	// leaves the trigger blank but still tappable.
	Placeholder string

	// Format is the time layout the summary is written in; empty gives
	// "Jan 2, 2006".
	//
	// Go's time package names months and weekdays in English only, so a
	// layout carrying a name is a layout in English. Where that matters, use
	// a numeric layout — "2006-01-02" reads the same in every language and
	// has none of the day/month ambiguity of "02/01/2006".
	Format string

	// Calendar is the template the sheet's grid is rendered from, exactly as
	// SegmentedControl.Segment is the template for its chips. Today, Min,
	// Max, Marked, WeekStart, the three label functions, Header and Style all
	// apply; Month, OnMonthChange, Selected and OnSelect are overwritten,
	// since those four are what the picker is for.
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
	// as actionable. The sheet's own contents are disabled with it, so a tap
	// racing the patch cannot land in an open picker.
	Disabled bool

	// AccessibilityLabel names the trigger; empty announces the summary text,
	// which is the date or the placeholder. AccessibilityHint describes what
	// tapping does.
	AccessibilityLabel string
	AccessibilityHint  string

	// Style is applied to the trigger row after its defaults. The sheet is
	// styled through Calendar.Style and the theme.
	Style []core.StyleProp
}

// datePickerFormat is the summary's layout when none is given: month, day and
// a four-digit year, which is unambiguous where a numeric "02/01/2026" is not.
const datePickerFormat = "Jan 2, 2006"

// datePickerSheet bounds the modal's width. A floor is not decoration: the
// grid's cells are flex-basis-0, so inside an overlay that shrinks to fit its
// content they would divide a width of nothing. The ceiling keeps a phone
// sheet from becoming a desktop-wide band.
const (
	datePickerMinWidth = "300px"
	datePickerMaxWidth = "360px"
)

func (p DatePicker) Render(ctx *core.Context) *core.Node {
	// Two hooks, in a fixed order, on every pass. See the type comment.
	open := core.NewState(ctx, false)
	// Zero means "whatever month Calendar's anchor resolves to", which is the
	// selection's, then Today's. Only an arrow tap ever makes it concrete.
	month := core.NewState(ctx, time.Time{})

	closeSheet := func() { open.Set(false) }

	items := make([]core.PropsAndChildren, 0, len(p.Style)+4)
	items = append(items,
		p.trigger(ctx, open, month),
		p.sheet(ctx, open, month, closeSheet),
	)

	// Box, not Column: the picker is one control in somebody's form, and a
	// Column would arrive with the theme's screen inset wrapped around it.
	// The Modal is a sibling of the trigger rather than a child of it, since
	// an overlay nested inside a bordered row would inherit the row's clip on
	// any target that honours overflow.
	return core.Box(items...).Render(ctx)
}

// trigger is the summary the field shows when the sheet is closed.
func (p DatePicker) trigger(ctx *core.Context, open core.State[bool], month core.State[time.Time]) core.View {
	t := ctx.Theme()

	summary := p.Placeholder
	ink := t.Colors.TextSecondary
	if !p.Selected.IsZero() {
		layout := p.Format
		if layout == "" {
			layout = datePickerFormat
		}
		summary = p.Selected.Format(layout)
		ink = t.Colors.TextPrimary
	}

	name := p.AccessibilityLabel
	if name == "" {
		name = summary
	}

	items := make([]core.PropsAndChildren, 0, len(p.Style)+10)
	items = append(items,
		// The theme's own field chrome, so a picker sitting between two text
		// inputs is the same height and the same fill as they are.
		core.UseStyle(t.Components.Input),
		// Plus the hairline the theme's Input entry does not carry. A text
		// field can get away with no edge because a caret and a keyboard
		// announce it the moment it is touched; a summary that only reports a
		// value has to look like a control before it is touched, and on
		// DefaultTheme its fill is the page's own white.
		core.BorderWidth(1),
		core.BorderColor(t.Colors.BorderColor()),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.Gap(float64(t.Spacing.SM)),
	)
	for _, sp := range p.Style {
		items = append(items, sp)
	}
	// After the caller's styles, as Button does it: whether the control is
	// inert is not a look, so a Style override must not re-enable it.
	if p.Disabled {
		items = append(items, core.Disabled(true))
	}
	items = append(items,
		core.AccessibilityLabel(name),
		core.AccessibilityHint(p.AccessibilityHint),
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
		// somebody paged to.
		month.Set(time.Time{})
		open.Set(true)
	}))

	items = append(items,
		core.Text(summary,
			core.UseStyle(t.Typography.Body),
			core.TextColor(ink),
			core.FlexGrow(1),
		),
		// Decoration: the trigger's spoken name already carries the value,
		// and "calendar" read out after it is noise.
		core.Text("📅", core.AccessibilityHidden()),
	)

	return core.Row(items...)
}

// sheet is the modal the trigger opens: a heading row of exits over the grid.
func (p DatePicker) sheet(ctx *core.Context, open core.State[bool], month core.State[time.Time], closeSheet func()) core.View {
	t := ctx.Theme()

	cal := p.Calendar
	cal.Month = month.Get()
	cal.OnMonthChange = month.Set
	cal.Selected = p.Selected
	if p.OnSelect != nil && !p.Disabled {
		cal.OnSelect = func(d time.Time) {
			p.OnSelect(d)
			// The tap that chooses is the tap that finishes; see the type
			// comment on why there is no Done button.
			closeSheet()
		}
	} else {
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
