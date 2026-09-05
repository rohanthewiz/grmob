package components

import (
	"testing"
	"time"

	"github.com/rohanthewiz/grmob/core"
)

var pickerDay = time.Date(2026, time.September, 12, 0, 0, 0, 0, time.UTC)

// The picker's tree: a trigger row and the sheet's Modal, side by side under
// one Box. These readers name the parts so a test reads as intent rather than
// as index arithmetic.
func pickerTrigger(t *testing.T, n *core.Node) *core.Node {
	t.Helper()
	if len(n.Children) != 2 {
		t.Fatalf("picker children = %d, want the trigger and the sheet", len(n.Children))
	}
	return n.Children[0]
}

func pickerModal(t *testing.T, n *core.Node) *core.Node {
	t.Helper()
	m := n.Children[1]
	if m.Type != "Modal" {
		t.Fatalf("second child is %q, want Modal", m.Type)
	}
	return m
}

// pickerCalendar digs out the grid inside the sheet: Modal → Card → [head, grid].
func pickerCalendar(t *testing.T, n *core.Node) *core.Node {
	t.Helper()
	card := pickerModal(t, n).Children[0]
	if len(card.Children) != 2 {
		t.Fatalf("sheet children = %d, want the heading row and the grid", len(card.Children))
	}
	return card.Children[1]
}

func TestDatePickerTriggerSummarizesTheSelection(t *testing.T) {
	theme := core.DefaultTheme
	ctx := core.NewContext()
	n := renderPass(ctx, DatePicker{Selected: pickerDay, OnSelect: func(time.Time) {}})

	if findText(n, "Sep 12, 2026") == nil {
		t.Error("the trigger should show the selection in the default layout")
	}
	summary := pickerTrigger(t, n).Children[0]
	if summary.Style.TextColor != theme.Colors.TextPrimary {
		t.Errorf("a filled field's ink = %q, want TextPrimary", summary.Style.TextColor)
	}
}

func TestDatePickerPlaceholderWhenEmpty(t *testing.T) {
	theme := core.DefaultTheme
	ctx := core.NewContext()
	n := renderPass(ctx, DatePicker{Placeholder: "Choose a date", OnSelect: func(time.Time) {}})

	summary := pickerTrigger(t, n).Children[0]
	if summary.Props["content"] != "Choose a date" {
		t.Errorf("summary = %v, want the placeholder", summary.Props["content"])
	}
	// The placeholder is quieter than a value, the way a placeholder in a text
	// field is: it names what is missing rather than reporting what is there.
	if summary.Style.TextColor != theme.Colors.TextSecondary {
		t.Errorf("placeholder ink = %q, want TextSecondary", summary.Style.TextColor)
	}
}

func TestDatePickerCustomFormat(t *testing.T) {
	ctx := core.NewContext()
	n := renderPass(ctx, DatePicker{Selected: pickerDay, Format: "2006-01-02", OnSelect: func(time.Time) {}})
	if findText(n, "2026-09-12") == nil {
		t.Error("Format should be the layout the summary is written in")
	}
}

func TestDatePickerOpensAndCloses(t *testing.T) {
	ctx := core.NewContext()
	p := DatePicker{Selected: pickerDay, OnSelect: func(time.Time) {}}

	n := renderPass(ctx, p)
	if pickerModal(t, n).Props["visible"] != false {
		t.Fatal("the sheet should start closed")
	}

	ctx.TriggerCallback(pickerTrigger(t, n).Props["onClick"].(string))
	n = renderPass(ctx, p)
	if pickerModal(t, n).Props["visible"] != true {
		t.Fatal("tapping the trigger should open the sheet")
	}

	// The ✕ is one of the ways out; the backdrop is the other and rides the
	// Modal's own OnDismiss.
	closeBtn := findButton(pickerModal(t, n), "✕")
	if closeBtn == nil {
		t.Fatal("the sheet should carry a close button")
	}
	ctx.TriggerCallback(closeBtn.Props["onClick"].(string))
	n = renderPass(ctx, p)
	if pickerModal(t, n).Props["visible"] != false {
		t.Error("the close button should shut the sheet")
	}
	if pickerModal(t, n).Props["onDismiss"] == nil {
		t.Error("the backdrop should be a way out too")
	}
}

// The tap that chooses is the tap that finishes: no Done button stands between
// picking a day and the sheet closing.
func TestDatePickerPickingSelectsAndCloses(t *testing.T) {
	ctx := core.NewContext()
	var got time.Time
	p := DatePicker{Selected: pickerDay, OnSelect: func(d time.Time) { got = d }}

	n := renderPass(ctx, p)
	ctx.TriggerCallback(pickerTrigger(t, n).Props["onClick"].(string))
	n = renderPass(ctx, p)

	// September 2026 starts on a Tuesday, so in a Sunday-start grid the 18th
	// is at index 2 + 17.
	cell := cellFor(t, dayCells(t, pickerCalendar(t, n)), 2, 18)
	ctx.TriggerCallback(cell.Props["onClick"].(string))

	if y, m, d := got.Date(); y != 2026 || m != time.September || d != 18 {
		t.Errorf("selected %v, want 2026-09-18", got)
	}
	n = renderPass(ctx, p)
	if pickerModal(t, n).Props["visible"] != false {
		t.Error("picking a day should close the sheet")
	}
}

func TestDatePickerClearOnlyWhenClearable(t *testing.T) {
	ctx := core.NewContext()
	n := renderPass(ctx, DatePicker{Selected: pickerDay, OnSelect: func(time.Time) {}})
	if findButton(n, "Clear") != nil {
		t.Error("a picker with no OnClear should not offer to empty a field the form may require")
	}

	ctx2 := core.NewContext()
	cleared := false
	p := DatePicker{Selected: pickerDay, OnSelect: func(time.Time) {}, OnClear: func() { cleared = true }}
	n = renderPass(ctx2, p)
	ctx2.TriggerCallback(pickerTrigger(t, n).Props["onClick"].(string))
	n = renderPass(ctx2, p)

	clear := findButton(n, "Clear")
	if clear == nil {
		t.Fatal("OnClear should put a Clear button in the sheet")
	}
	ctx2.TriggerCallback(clear.Props["onClick"].(string))
	if !cleared {
		t.Error("Clear should call OnClear")
	}
	n = renderPass(ctx2, p)
	if pickerModal(t, n).Props["visible"] != false {
		t.Error("Clear should close the sheet too")
	}
}

func TestDatePickerDisabledNeitherOpensNorPicks(t *testing.T) {
	ctx := core.NewContext()
	picked := false
	p := DatePicker{Selected: pickerDay, Disabled: true, OnSelect: func(time.Time) { picked = true }}

	n := renderPass(ctx, p)
	trigger := pickerTrigger(t, n)
	if !trigger.Style.Disabled {
		t.Error("a disabled picker's trigger should be inert")
	}
	// The handler is registered anyway, so a tap already in flight when the
	// disabling patch lands finds something to call.
	id, ok := trigger.Props["onClick"].(string)
	if !ok {
		t.Fatal("the handler should still be registered while disabled")
	}
	ctx.TriggerCallback(id)
	n = renderPass(ctx, p)
	if pickerModal(t, n).Props["visible"] != false {
		t.Error("a disabled picker must not open")
	}

	// And the grid behind it is inert, so nothing can be chosen out of a
	// sheet that raced its way open.
	cell := cellFor(t, dayCells(t, pickerCalendar(t, n)), 2, 18)
	ctx.TriggerCallback(cell.Props["onClick"].(string))
	if picked {
		t.Error("a disabled picker's grid must not select")
	}
}

// The template is the SegmentedControl.Segment move: everything but the four
// fields the picker is computing goes through untouched.
func TestDatePickerCalendarTemplatePassesThrough(t *testing.T) {
	ctx := core.NewContext()
	n := renderPass(ctx, DatePicker{
		Selected: pickerDay,
		OnSelect: func(time.Time) {},
		Calendar: Calendar{
			// Month and Selected here are overwritten by the picker; the rest
			// applies.
			Month:      time.Date(2001, time.January, 1, 0, 0, 0, 0, time.UTC),
			Selected:   time.Date(2001, time.January, 1, 0, 0, 0, 0, time.UTC),
			WeekStart:  time.Monday,
			Min:        time.Date(2026, time.September, 10, 0, 0, 0, 0, time.UTC),
			Marked:     func(d time.Time) bool { return d.Day() == 18 },
			MonthLabel: func(m time.Time) string { return "Setembro de 2026" },
		},
	})
	cal := pickerCalendar(t, n)

	if findText(cal, "Setembro de 2026") == nil {
		t.Error("the template's MonthLabel should reach the grid — and its Month should not")
	}
	if got := cal.Children[1].Children[0].Children[0].Props["content"]; got != "Mo" {
		t.Errorf("first caption = %v, want the template's Monday start", got)
	}
	// Monday start, September 1 a Tuesday: one leading day, so the 9th sits at
	// index 1 + 8 and the 10th just after it.
	cells := dayCells(t, cal)
	if !cells[9].Style.Disabled || cells[10].Style.Disabled {
		t.Error("the template's Min should bound the grid")
	}
	if cells[18].Children[1].Style.Background != core.DefaultTheme.Colors.Primary {
		t.Error("the template's Marked should dot the 18th")
	}
	// The 12th is at index 1 + 11 under a Monday start; it is the picker's own
	// Selected, which must win over the template's.
	if cells[12].Style.Background != core.DefaultTheme.Colors.Primary {
		t.Error("the picker's Selected should win over the template's")
	}
}

// The browsed month is reset every time the sheet opens, so a picker always
// opens on the month it is showing rather than the last one somebody paged to
// — and a Selected set from elsewhere between two openings is followed.
func TestDatePickerReopensOnTheSelectedMonth(t *testing.T) {
	ctx := core.NewContext()
	p := DatePicker{Selected: pickerDay, OnSelect: func(time.Time) {}}

	n := renderPass(ctx, p)
	ctx.TriggerCallback(pickerTrigger(t, n).Props["onClick"].(string))
	n = renderPass(ctx, p)
	if findText(pickerCalendar(t, n), "September 2026") == nil {
		t.Fatal("the sheet should open on the selection's month")
	}

	// Page forward twice, then close.
	for range 2 {
		fwd := pickerCalendar(t, n).Children[0].Children[2]
		ctx.TriggerCallback(fwd.Props["onClick"].(string))
		n = renderPass(ctx, p)
	}
	if findText(pickerCalendar(t, n), "November 2026") == nil {
		t.Fatal("the arrows should page the sheet's grid")
	}
	ctx.TriggerCallback(findButton(pickerModal(t, n), "✕").Props["onClick"].(string))

	// Reopen with a selection the caller has since moved: the picker follows
	// it rather than reopening on November.
	p.Selected = time.Date(2027, time.March, 3, 0, 0, 0, 0, time.UTC)
	n = renderPass(ctx, p)
	ctx.TriggerCallback(pickerTrigger(t, n).Props["onClick"].(string))
	n = renderPass(ctx, p)
	if findText(pickerCalendar(t, n), "March 2027") == nil {
		t.Error("reopening should follow the current selection, not the last month browsed")
	}
}

func TestDatePickerTitleAndCaptionsAreOverridable(t *testing.T) {
	ctx := core.NewContext()
	n := renderPass(ctx, DatePicker{
		Selected:   pickerDay,
		OnSelect:   func(time.Time) {},
		OnClear:    func() {},
		Title:      "Data do evento",
		ClearLabel: "Limpar",
		CloseLabel: "Fechar",
	})
	if findText(n, "Data do evento") == nil {
		t.Error("Title should name the sheet")
	}
	if findButton(n, "Limpar") == nil || findButton(n, "Fechar") == nil {
		t.Error("both exits should take the caller's captions")
	}
}

func TestDatePickerTriggerCarriesTheThemeFieldChrome(t *testing.T) {
	theme := core.DefaultTheme
	ctx := core.NewContext()
	n := renderPass(ctx, DatePicker{Selected: pickerDay, OnSelect: func(time.Time) {}})
	trigger := pickerTrigger(t, n)

	if trigger.Style.Background != theme.Components.Input.Background {
		t.Errorf("trigger fill = %q, want the theme's Input fill %q",
			trigger.Style.Background, theme.Components.Input.Background)
	}
	// Plus the hairline the theme's Input entry does not carry: on
	// DefaultTheme the field's fill is the page's own white, and a summary
	// with no caret has to look like a control before it is touched.
	if trigger.Style.BorderWidth != 1 || trigger.Style.BorderColor != theme.Colors.BorderColor() {
		t.Errorf("trigger border = %v/%q, want a 1px palette hairline",
			trigger.Style.BorderWidth, trigger.Style.BorderColor)
	}
}

func TestDatePickerNamesItselfForScreenReaders(t *testing.T) {
	ctx := core.NewContext()
	n := renderPass(ctx, DatePicker{Selected: pickerDay, OnSelect: func(time.Time) {}})
	if got := pickerTrigger(t, n).Style.AccessibilityLabel; got != "Sep 12, 2026" {
		t.Errorf("trigger name = %q, want the summary it shows", got)
	}
	// The glyph is decoration; the value is already in the trigger's name.
	glyph := pickerTrigger(t, n).Children[1]
	if !glyph.Style.AccessibilityHidden {
		t.Error("the calendar glyph should be hidden from assistive technology")
	}

	ctx2 := core.NewContext()
	n = renderPass(ctx2, DatePicker{
		Selected:           pickerDay,
		OnSelect:           func(time.Time) {},
		AccessibilityLabel: "Event date",
		AccessibilityHint:  "Opens a calendar",
	})
	trigger := pickerTrigger(t, n)
	if trigger.Style.AccessibilityLabel != "Event date" || trigger.Style.AccessibilityHint != "Opens a calendar" {
		t.Errorf("trigger semantics = %q / %q, want the caller's",
			trigger.Style.AccessibilityLabel, trigger.Style.AccessibilityHint)
	}
}
