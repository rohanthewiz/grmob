package comps

import (
	"strings"
	"testing"
	"time"

	"github.com/rohanthewiz/grmob/core"
)

// The span every test here picks: September 14–20, 2026, a Monday to a Sunday.
// September 1 is a Tuesday, so in a Sunday-start grid day n of the shown month
// sits at index n+1 and cellFor's `first` is 2 — the same offset the DatePicker
// tests use.
var (
	rangeFrom  = time.Date(2026, time.September, 14, 12, 0, 0, 0, time.UTC)
	rangeTo    = time.Date(2026, time.September, 20, 12, 0, 0, 0, time.UTC)
	rangeToday = time.Date(2026, time.September, 12, 12, 0, 0, 0, time.UTC)
)

// rangeSeptember is the Calendar template every harness here passes. Today is
// the only field in it that matters: the picker clears Selected and holds its
// browsed Month at zero until an arrow is tapped, so with an empty span there
// would be nothing left in Calendar's anchor chain and the grid would render
// nothing at all — the same thing a DatePicker with no Selected and no Today
// does, and a reminder that the clock is the caller's to supply.
func rangeSeptember() Calendar { return Calendar{Today: rangeToday} }

// rangeHarness re-renders the picker against one context, the way a Manager
// does, and writes a completed range back into the fields the next pass reads
// — so a test sees what a real caller would see. The picker owns three pieces
// of state, so its tests cannot call Render bare.
//
// Every pass runs in debug mode and fails on any concern, which is how the
// suite asserts that the ordinary uses below raise none.
type rangeHarness struct {
	t     *testing.T
	ctx   *core.Context
	build func(*rangeHarness) DateRangePicker
	node  *core.Node

	start, end time.Time
	changes    int
	cleared    int
}

func newRangeHarness(t *testing.T, build func(*rangeHarness) DateRangePicker) *rangeHarness {
	t.Helper()
	core.SetDebugMode(true)
	core.ClearConcerns()
	t.Cleanup(func() { core.SetDebugMode(false); core.ClearConcerns() })
	h := &rangeHarness{t: t, ctx: core.NewContext(), build: build}
	h.render()
	return h
}

func (h *rangeHarness) render() {
	h.t.Helper()
	h.ctx.BeginRenderPass()
	h.ctx.Reset()
	h.node = h.build(h).Render(h.ctx)
	h.ctx.EndRenderPass()
	// The whole-tree audit, as render.Manager runs it; see renderDebug.
	core.AuditTree(h.node)
	if dump := core.DumpConcerns(); dump != "" {
		h.t.Errorf("the picker raised concerns:\n%s", dump)
	}
}

// record is the OnChange every harness-built picker hands over: it stores the
// pair and counts the reports, so "fires once, on the completing tap" is a
// number a test can read rather than a flag.
func (h *rangeHarness) record(start, end time.Time) {
	h.start, h.end = start, end
	h.changes++
}

func (h *rangeHarness) open() {
	h.t.Helper()
	h.ctx.TriggerCallback(pickerTrigger(h.t, h.node).Props["onClick"].(string))
	h.render()
}

// tapDay taps day n of September 2026 in the open sheet.
//
// The cell is found by its spoken name rather than by grid index, because a
// WeekStart the caller changed moves every index and the date does not. Names
// are unique across the 42 cells — two of them can show the same numeral but
// never the same date — and an endpoint's name carries a suffix, hence the
// prefix match.
func (h *rangeHarness) tapDay(n int) {
	h.t.Helper()
	want := time.Date(2026, time.September, n, 12, 0, 0, 0, time.UTC).Format("Monday, January 2, 2006")
	cell := findFirst(pickerCalendar(h.t, h.node), func(x *core.Node) bool {
		return x.Style != nil && x.Style.AccessibilityRole == core.RoleGridCell &&
			strings.HasPrefix(x.Style.AccessibilityLabel, want)
	})
	if cell == nil {
		h.t.Fatalf("no cell named %q in the sheet's grid", want)
	}
	id, ok := cell.Props["onClick"].(string)
	if !ok {
		h.t.Fatalf("the cell for day %d carries no onClick", n)
	}
	h.ctx.TriggerCallback(id)
	h.render()
}

func (h *rangeHarness) tapButton(label string) {
	h.t.Helper()
	b := findButton(h.node, label)
	if b == nil {
		h.t.Fatalf("no button captioned %q", label)
	}
	h.ctx.TriggerCallback(b.Props["onClick"].(string))
	h.render()
}

func (h *rangeHarness) isOpen() bool {
	return pickerModal(h.t, h.node).Props["visible"] == true
}

// litDays returns the day numbers of every cell in the open sheet's grid that
// carries any fill at all, endpoints and band alike, in grid order. It is how
// the tests state what the reader can see without asserting on colours the
// Calendar suite already pins.
func (h *rangeHarness) litDays() []string {
	h.t.Helper()
	var out []string
	for _, cell := range dayCells(h.t, pickerCalendar(h.t, h.node)) {
		if cell.Style != nil && cell.Style.Background != "" {
			out = append(out, dayNumber(h.t, cell))
		}
	}
	return out
}

// plainRange is the picker most of these tests build: a span the harness holds
// and a recorder for what comes back.
func plainRange(h *rangeHarness) DateRangePicker {
	return DateRangePicker{Start: h.start, End: h.end, OnChange: h.record, Calendar: rangeSeptember()}
}

func TestDateRangePickerTriggerSummarizesTheSpan(t *testing.T) {
	theme := core.DefaultTheme
	h := newRangeHarness(t, func(h *rangeHarness) DateRangePicker {
		return DateRangePicker{Start: rangeFrom, End: rangeTo, OnChange: h.record}
	})

	if findText(h.node, "Sep 14, 2026 – Sep 20, 2026") == nil {
		t.Error("the trigger should show both ends in the default layout, joined by an en dash")
	}
	summary := pickerTrigger(t, h.node).Children[0]
	if summary.Style.TextColor != theme.Colors.TextPrimary {
		t.Errorf("a filled field's ink = %q, want TextPrimary", summary.Style.TextColor)
	}
}

func TestDateRangePickerPlaceholderWhenEmpty(t *testing.T) {
	theme := core.DefaultTheme
	h := newRangeHarness(t, func(h *rangeHarness) DateRangePicker {
		return DateRangePicker{Placeholder: "Choose your dates", Start: h.start, End: h.end, OnChange: h.record, Calendar: rangeSeptember()}
	})

	summary := pickerTrigger(t, h.node).Children[0]
	if summary.Props["content"] != "Choose your dates" {
		t.Errorf("summary = %v, want the placeholder", summary.Props["content"])
	}
	if summary.Style.TextColor != theme.Colors.TextSecondary {
		t.Errorf("placeholder ink = %q, want TextSecondary", summary.Style.TextColor)
	}
}

func TestDateRangePickerFormatAndSeparatorAreTheCallers(t *testing.T) {
	h := newRangeHarness(t, func(h *rangeHarness) DateRangePicker {
		return DateRangePicker{
			Start: rangeFrom, End: rangeTo, OnChange: h.record,
			Format: "2006-01-02", Separator: " → ",
		}
	})
	if findText(h.node, "2026-09-14 → 2026-09-20") == nil {
		t.Error("both halves take Format, and Separator joins them")
	}
}

// A Start the caller passed with no End is one date and no separator: a
// trailing dash is a promise of a second date that is not coming.
func TestDateRangePickerHalfSpanSummarizesAsOneDate(t *testing.T) {
	h := newRangeHarness(t, func(h *rangeHarness) DateRangePicker {
		return DateRangePicker{Start: rangeFrom, OnChange: h.record}
	})
	if findText(h.node, "Sep 14, 2026") == nil {
		t.Error("a Start with no End should summarize as that one date")
	}
	if findText(h.node, "Sep 14, 2026 – ") != nil {
		t.Error("a half span must not trail a separator")
	}
}

// The protocol proper: the first tap reports nothing and lights one day, the
// second reports the pair and closes the sheet.
func TestDateRangePickerTwoTapsMakeARange(t *testing.T) {
	h := newRangeHarness(t, plainRange)
	h.open()

	h.tapDay(14)
	if h.changes != 0 {
		t.Fatalf("OnChange fired %d times on the first tap; a half-made range is the widget's own", h.changes)
	}
	if !h.isOpen() {
		t.Error("the first tap does not finish anything, so the sheet stays open")
	}
	if lit := h.litDays(); len(lit) != 1 || lit[0] != "14" {
		t.Errorf("lit days after the first tap = %v, want just the 14th and no band", lit)
	}

	h.tapDay(20)
	if h.changes != 1 {
		t.Fatalf("OnChange fired %d times, want exactly one report for one completed range", h.changes)
	}
	if y, m, d := h.start.Date(); y != 2026 || m != time.September || d != 14 {
		t.Errorf("start = %v, want 2026-09-14", h.start)
	}
	if y, m, d := h.end.Date(); y != 2026 || m != time.September || d != 20 {
		t.Errorf("end = %v, want 2026-09-20", h.end)
	}
	if h.isOpen() {
		t.Error("the tap that completes the range is the tap that finishes")
	}
	// And the field now shows it.
	if findText(h.node, "Sep 14, 2026 – Sep 20, 2026") == nil {
		t.Error("the trigger should summarize the range the picker just reported")
	}
}

// "The other end" is not a claim about which end. Tapping the later day first
// orders the pair rather than throwing the earlier tap away.
func TestDateRangePickerSecondTapMayBeTheEarlierOne(t *testing.T) {
	h := newRangeHarness(t, plainRange)
	h.open()
	h.tapDay(20)
	h.tapDay(14)

	if h.changes != 1 {
		t.Fatalf("OnChange fired %d times, want one", h.changes)
	}
	if h.start.After(h.end) {
		t.Fatalf("reported %v – %v: OnChange promises start ≤ end", h.start, h.end)
	}
	if d := h.start.Day(); d != 14 {
		t.Errorf("start day = %d, want the 14th", d)
	}
	if d := h.end.Day(); d != 20 {
		t.Errorf("end day = %d, want the 20th", d)
	}
}

// The second tap is the other end wherever it falls, including on the cell the
// first one landed on. A one-day range needs no case of its own.
func TestDateRangePickerSameDayTwiceIsAOneDayRange(t *testing.T) {
	h := newRangeHarness(t, plainRange)
	h.open()
	h.tapDay(14)
	h.tapDay(14)

	if h.changes != 1 {
		t.Fatalf("OnChange fired %d times, want one", h.changes)
	}
	if !h.start.Equal(h.end) {
		t.Errorf("reported %v – %v, want one day at both ends", h.start, h.end)
	}
	if findText(h.node, "Sep 14, 2026 – Sep 14, 2026") == nil {
		t.Error("a one-day range still summarizes as a range")
	}
}

// The plan's decision, and here it is a consequence rather than a rule:
// completing a range clears the pending start, so the next tap finds none and
// begins again. Nothing is moved to whichever end happens to be nearer.
func TestDateRangePickerThirdTapStartsANewRange(t *testing.T) {
	h := newRangeHarness(t, plainRange)
	h.open()
	h.tapDay(14)
	h.tapDay(20)
	h.open()

	h.tapDay(5)
	if h.changes != 1 {
		t.Fatalf("OnChange fired %d times; the tap after a completed range starts a new one and reports nothing", h.changes)
	}
	if lit := h.litDays(); len(lit) != 1 || lit[0] != "5" {
		t.Errorf("lit days = %v, want only the new start — the old span leaves the grid so the two are never both lit", lit)
	}

	h.tapDay(8)
	if h.changes != 2 {
		t.Fatalf("OnChange fired %d times, want a second report", h.changes)
	}
	if h.start.Day() != 5 || h.end.Day() != 8 {
		t.Errorf("reported %v – %v, want the 5th to the 8th", h.start, h.end)
	}
}

// The promise the pending start exists for: a reader who opens the sheet to
// check which week they booked can leave having changed nothing, even after a
// stray tap. Every way out runs through the same close.
func TestDateRangePickerLeavingMidPickChangesNothing(t *testing.T) {
	h := newRangeHarness(t, plainRange)
	h.start, h.end = rangeFrom, rangeTo
	h.render()

	h.open()
	h.tapDay(3) // a stray first tap
	h.tapButton("✕")

	if h.changes != 0 {
		t.Errorf("OnChange fired %d times on the way out; closing reports nothing", h.changes)
	}
	if !h.start.Equal(rangeFrom) || !h.end.Equal(rangeTo) {
		t.Errorf("the field now holds %v – %v, want the span it opened with", h.start, h.end)
	}

	// And the discarded start does not survive to the next opening.
	h.open()
	lit := h.litDays()
	if len(lit) != 7 || lit[0] != "14" || lit[6] != "20" {
		t.Errorf("reopened on lit days %v, want the seven days of the original span", lit)
	}
}

func TestDateRangePickerIsAButtonThatOpensADialog(t *testing.T) {
	h := newRangeHarness(t, func(h *rangeHarness) DateRangePicker {
		return DateRangePicker{Start: rangeFrom, End: rangeTo, OnChange: h.record}
	})
	trigger := pickerTrigger(t, h.node)

	if trigger.Style.AccessibilityRole != core.RoleButton {
		t.Errorf("trigger role = %q, want button: a Row is scenery until a role says otherwise",
			trigger.Style.AccessibilityRole)
	}
	if trigger.Style.AccessibilityHasPopup != core.PopupDialog {
		t.Errorf("trigger popup = %q, want dialog", trigger.Style.AccessibilityHasPopup)
	}
	if got := trigger.Style.AccessibilityLabel; got != "Sep 14, 2026 – Sep 20, 2026" {
		t.Errorf("trigger name = %q, want the summary it shows", got)
	}
	if !trigger.Children[1].Style.AccessibilityHidden {
		t.Error("the calendar glyph should be hidden: the trigger's name already carries the span")
	}
}

func TestDateRangePickerClearOnlyWhenClearable(t *testing.T) {
	h := newRangeHarness(t, func(h *rangeHarness) DateRangePicker {
		return DateRangePicker{Start: rangeFrom, End: rangeTo, OnChange: h.record}
	})
	h.open()
	if findButton(h.node, "Clear") != nil {
		t.Error("a picker with no OnClear should not offer to empty a field the form may require")
	}

	g := newRangeHarness(t, func(g *rangeHarness) DateRangePicker {
		return DateRangePicker{
			Start: g.start, End: g.end, OnChange: g.record, Calendar: rangeSeptember(),
			OnClear: func() { g.start, g.end = time.Time{}, time.Time{}; g.cleared++ },
		}
	})
	g.start, g.end = rangeFrom, rangeTo
	g.render()
	g.open()
	g.tapButton("Clear")

	if g.cleared != 1 {
		t.Errorf("OnClear fired %d times, want one", g.cleared)
	}
	if g.isOpen() {
		t.Error("Clear should close the sheet too")
	}
	if !g.start.IsZero() || !g.end.IsZero() {
		t.Errorf("the field still holds %v – %v after a clear", g.start, g.end)
	}
}

func TestDateRangePickerDisabledNeitherOpensNorPicks(t *testing.T) {
	h := newRangeHarness(t, func(h *rangeHarness) DateRangePicker {
		return DateRangePicker{Start: rangeFrom, End: rangeTo, Disabled: true, OnChange: h.record}
	})

	trigger := pickerTrigger(t, h.node)
	if !trigger.Style.Disabled {
		t.Error("a disabled picker's trigger should be inert")
	}
	// The handler is registered anyway, so a tap already in flight when the
	// disabling patch lands finds something to call.
	id, ok := trigger.Props["onClick"].(string)
	if !ok {
		t.Fatal("the handler should still be registered while disabled")
	}
	h.ctx.TriggerCallback(id)
	h.render()
	if h.isOpen() {
		t.Error("a disabled picker must not open")
	}

	// And the grid behind it is inert, so nothing can be started out of a
	// sheet that raced its way open.
	cell := cellFor(t, dayCells(t, pickerCalendar(t, h.node)), 2, 18)
	h.ctx.TriggerCallback(cell.Props["onClick"].(string))
	h.render()
	if h.changes != 0 {
		t.Error("a disabled picker's grid must not pick")
	}
}

// The template is DatePicker's move: everything but the fields the picker
// drives goes through untouched, and the ones it drives are not negotiable.
func TestDateRangePickerCalendarTemplateIsDrivenWhereItMustBe(t *testing.T) {
	stale := time.Date(2001, time.January, 1, 12, 0, 0, 0, time.UTC)
	h := newRangeHarness(t, func(h *rangeHarness) DateRangePicker {
		return DateRangePicker{
			Start: rangeFrom, End: rangeTo, OnChange: h.record,
			Calendar: Calendar{
				// Overwritten by the picker, all four.
				Month:        stale,
				Selected:     stale,
				RangeStart:   stale,
				RangeEnd:     stale,
				Deselectable: true,
				// Passed through.
				WeekStart:  time.Monday,
				Min:        time.Date(2026, time.September, 10, 0, 0, 0, 0, time.UTC),
				MonthLabel: func(m time.Time) string { return "Setembro de 2026" },
			},
		}
	})
	h.open()
	cal := pickerCalendar(t, h.node)

	if findText(cal, "Setembro de 2026") == nil {
		t.Error("the template's MonthLabel should reach the grid — and its Month should not")
	}
	if got := captionRow(t, cal).Children[0].Children[0].Props["content"]; got != "Mo" {
		t.Errorf("first caption = %v, want the template's Monday start", got)
	}
	// Monday start, September 1 a Tuesday: one leading day, so the 9th sits at
	// index 1 + 8 and the 10th just after it.
	cells := dayCells(t, cal)
	if !cells[9].Style.Disabled || cells[10].Style.Disabled {
		t.Error("the template's Min should bound the grid")
	}
	// The picker's own span is what is lit, and the template's stale Selected
	// lights nothing: a leftover selection would wear exactly an endpoint's
	// fill and mean nothing at all.
	if lit := h.litDays(); len(lit) != 7 || lit[0] != "14" || lit[6] != "20" {
		t.Errorf("lit days = %v, want the picker's own seven", lit)
	}

	// Deselectable is forced off, so a tap on an endpoint is an ordinary first
	// tap rather than a zero time arriving through the grid.
	h.tapDay(14)
	if h.changes != 0 {
		t.Fatal("a tap on an endpoint should start a new range, not report one")
	}
	if lit := h.litDays(); len(lit) != 1 || lit[0] != "14" {
		t.Errorf("lit days = %v, want the new pending start alone", lit)
	}
}

// The browsed month is reset every time the sheet opens, so the picker always
// opens on the span it is showing rather than the last month somebody paged to.
func TestDateRangePickerReopensOnTheSpansMonth(t *testing.T) {
	h := newRangeHarness(t, func(h *rangeHarness) DateRangePicker {
		return DateRangePicker{Start: rangeFrom, End: rangeTo, OnChange: h.record}
	})
	h.open()
	if findText(pickerCalendar(t, h.node), "September 2026") == nil {
		t.Fatal("the sheet should open on the span's month, with no Selected and no Month to anchor it")
	}

	for range 2 {
		fwd := pickerCalendar(t, h.node).Children[0].Children[2]
		h.ctx.TriggerCallback(fwd.Props["onClick"].(string))
		h.render()
	}
	if findText(pickerCalendar(t, h.node), "November 2026") == nil {
		t.Fatal("the arrows should page the sheet's grid")
	}

	h.tapButton("✕")
	h.open()
	if findText(pickerCalendar(t, h.node), "September 2026") == nil {
		t.Error("reopening should follow the span, not the last month browsed")
	}
}

func TestDateRangePickerTitleAndCaptionsAreOverridable(t *testing.T) {
	h := newRangeHarness(t, func(h *rangeHarness) DateRangePicker {
		return DateRangePicker{
			Start: rangeFrom, End: rangeTo, OnChange: h.record, OnClear: func() {},
			Title: "Datas da estadia", ClearLabel: "Limpar", CloseLabel: "Fechar",
		}
	})
	if findText(h.node, "Datas da estadia") == nil {
		t.Error("Title should name the sheet")
	}
	if findButton(h.node, "Limpar") == nil || findButton(h.node, "Fechar") == nil {
		t.Error("both exits should take the caller's captions")
	}
}

func TestDateRangePickerTriggerCarriesTheThemeFieldChrome(t *testing.T) {
	theme := core.DefaultTheme
	h := newRangeHarness(t, func(h *rangeHarness) DateRangePicker {
		return DateRangePicker{Start: rangeFrom, End: rangeTo, OnChange: h.record}
	})
	trigger := pickerTrigger(t, h.node)

	if trigger.Style.Background != theme.Components.Input.Background {
		t.Errorf("trigger fill = %q, want the theme's Input fill %q",
			trigger.Style.Background, theme.Components.Input.Background)
	}
	if trigger.Style.BorderWidth != theme.Components.Input.BorderWidth ||
		trigger.Style.BorderColor != theme.Components.Input.BorderColor {
		t.Errorf("trigger border = %v/%q, want the theme's Input frame %v/%q",
			trigger.Style.BorderWidth, trigger.Style.BorderColor,
			theme.Components.Input.BorderWidth, theme.Components.Input.BorderColor)
	}
}

// A picker with no OnChange takes every tap, completes a range and hands it to
// nobody — which looks exactly like one nobody has finished using.
func TestDateRangePickerWithNoOnChangeReportsAConcern(t *testing.T) {
	core.SetDebugMode(true)
	core.ClearConcerns()
	defer func() { core.SetDebugMode(false); core.ClearConcerns() }()

	ctx := core.NewContext()
	renderPass(ctx, DateRangePicker{Start: rangeFrom, End: rangeTo})

	if !strings.Contains(core.DumpConcerns(), ConcernDateRangePickerInert) {
		t.Errorf("want %s, got:\n%s", ConcernDateRangePickerInert, core.DumpConcerns())
	}
}
