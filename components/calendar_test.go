package components

import (
	"testing"
	"time"

	"github.com/rohanthewiz/grmob/core"
)

// renderCalendar renders c against a fresh context and returns the root node.
func renderCalendar(t *testing.T, c Calendar) *core.Node {
	t.Helper()
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	return c.Render(ctx)
}

// dayCells flattens the six week rows into the 42 cells in reading order. The
// first two children of the root are the month header and the weekday
// captions; everything after is a week.
func dayCells(t *testing.T, n *core.Node) []*core.Node {
	t.Helper()
	if len(n.Children) != calendarRows+2 {
		t.Fatalf("root children = %d, want %d (header + captions + %d weeks)",
			len(n.Children), calendarRows+2, calendarRows)
	}
	cells := make([]*core.Node, 0, calendarRows*calendarCols)
	for row := range calendarRows {
		week := n.Children[row+2]
		if len(week.Children) != calendarCols {
			t.Fatalf("week %d has %d cells, want %d", row, len(week.Children), calendarCols)
		}
		cells = append(cells, week.Children...)
	}
	return cells
}

// dayNumber reads the numeral a cell displays.
func dayNumber(t *testing.T, cell *core.Node) string {
	t.Helper()
	if len(cell.Children) == 0 || cell.Children[0].Type != "Text" {
		t.Fatalf("cell's first child is not the day number: %+v", cell)
	}
	return cell.Children[0].Props["content"].(string)
}

// cellFor returns the cell displaying day n of the *shown* month, i.e. the
// first cell numbered n that is not part of the leading run of adjacent days.
// Adjacent days are found positionally instead.
func cellFor(t *testing.T, cells []*core.Node, first, n int) *core.Node {
	t.Helper()
	return cells[first+n-1]
}

// sep 2026: September 1 is a Tuesday, so a Sunday-start grid leads with two
// August cells and the 1st sits at index 2. That offset is what makes it a
// better fixture than a month that happens to start on the week's first day.
var sep2026 = time.Date(2026, time.September, 15, 9, 30, 0, 0, time.UTC)

func TestCalendarGridIsAlwaysSixWeeks(t *testing.T) {
	// February 2027 is 28 days beginning on a Monday: in a Monday-start grid
	// it fills exactly four rows, which is the month most likely to expose a
	// grid that sizes itself to its content.
	short := Calendar{Month: time.Date(2027, time.February, 10, 0, 0, 0, 0, time.UTC), WeekStart: time.Monday}
	long := Calendar{Month: time.Date(2026, time.May, 10, 0, 0, 0, 0, time.UTC)} // 31 days from a Friday: six rows

	for name, c := range map[string]Calendar{"short": short, "long": long} {
		n := renderCalendar(t, c)
		cells := dayCells(t, n)
		if len(cells) != calendarRows*calendarCols {
			t.Errorf("%s month: %d cells, want %d — the grid must not size itself to the month",
				name, len(cells), calendarRows*calendarCols)
		}
	}
}

func TestCalendarLaysDaysUnderTheirWeekday(t *testing.T) {
	n := renderCalendar(t, Calendar{Month: sep2026})
	cells := dayCells(t, n)

	// Sunday start: Aug 30 (Sun), Aug 31 (Mon), then Sep 1 on Tuesday.
	for i, want := range []string{"30", "31", "1", "2", "3", "4", "5"} {
		if got := dayNumber(t, cells[i]); got != want {
			t.Errorf("cell %d = %q, want %q", i, got, want)
		}
	}
	// The tail runs into October: Sep 30 sits at index 2+29 = 31, so the grid
	// carries ten trailing days.
	if got := dayNumber(t, cells[31]); got != "30" {
		t.Errorf("cell 31 = %q, want the 30th", got)
	}
	if got := dayNumber(t, cells[32]); got != "1" {
		t.Errorf("cell 32 = %q, want October's 1st", got)
	}
}

func TestCalendarWeekStartShiftsColumnsAndCaptions(t *testing.T) {
	n := renderCalendar(t, Calendar{Month: sep2026, WeekStart: time.Monday})

	captions := n.Children[1]
	if len(captions.Children) != calendarCols {
		t.Fatalf("captions = %d, want %d", len(captions.Children), calendarCols)
	}
	for i, want := range []string{"Mo", "Tu", "We", "Th", "Fr", "Sa", "Su"} {
		got := captions.Children[i].Children[0].Props["content"]
		if got != want {
			t.Errorf("caption %d = %v, want %q", i, got, want)
		}
	}
	// September 1 is a Tuesday, so a Monday-start grid leads with one day.
	cells := dayCells(t, n)
	if got := dayNumber(t, cells[0]); got != "31" {
		t.Errorf("first cell = %q, want August's 31st", got)
	}
	if got := dayNumber(t, cells[1]); got != "1" {
		t.Errorf("second cell = %q, want the 1st", got)
	}
}

// The captions repeat what every cell's spoken name already carries, so they
// are hidden rather than read a second time per row.
func TestCalendarWeekdayCaptionsAreDecorative(t *testing.T) {
	n := renderCalendar(t, Calendar{Month: sep2026})
	for i, cap := range n.Children[1].Children {
		if !cap.Children[0].Style.AccessibilityHidden && !cap.Style.AccessibilityHidden {
			t.Errorf("caption %d is announced; it should be hidden from assistive technology", i)
		}
	}
}

func TestCalendarSelectionFillsAndPicksItsInk(t *testing.T) {
	theme := core.DefaultTheme
	n := renderCalendar(t, Calendar{
		Month:    sep2026,
		Selected: time.Date(2026, time.September, 12, 18, 0, 0, 0, time.UTC),
		OnSelect: func(time.Time) {},
	})
	cells := dayCells(t, n)
	sel := cellFor(t, cells, 2, 12)

	if sel.Style.Background != theme.Colors.Primary {
		t.Errorf("selected background = %q, want Primary %q", sel.Style.Background, theme.Colors.Primary)
	}
	// The ink is chosen against the fill rather than assumed: Primary is a
	// light blue in one bundled theme and a dark indigo in another.
	want := contrastInk(theme.Colors.Primary, theme.Colors.Background, theme.Colors.TextPrimary)
	if got := sel.Children[0].Style.TextColor; got != want {
		t.Errorf("selected ink = %q, want the contrast-picked %q", got, want)
	}
	// Any instant during the day selects it — 18:00 above, midday in the grid.
	if dayNumber(t, sel) != "12" {
		t.Errorf("selected cell shows %q, want 12", dayNumber(t, sel))
	}
}

func TestCalendarTodayIsARingAndSelectionWins(t *testing.T) {
	theme := core.DefaultTheme
	day12 := time.Date(2026, time.September, 12, 0, 0, 0, 0, time.UTC)
	day18 := time.Date(2026, time.September, 18, 0, 0, 0, 0, time.UTC)

	n := renderCalendar(t, Calendar{Month: sep2026, Today: day12, Selected: day18, OnSelect: func(time.Time) {}})
	cells := dayCells(t, n)

	today := cellFor(t, cells, 2, 12)
	if today.Style.BorderWidth != 1 || today.Style.BorderColor != theme.Colors.Primary {
		t.Errorf("today = border %v/%q, want a 1px Primary ring", today.Style.BorderWidth, today.Style.BorderColor)
	}
	if today.Style.Background != "" {
		t.Errorf("today background = %q, want none — the ring is what keeps it distinct from the selection", today.Style.Background)
	}

	// Both on one day: the fill wins and the ring is dropped, because a
	// Primary ring on a Primary fill is invisible anyway.
	both := renderCalendar(t, Calendar{Month: sep2026, Today: day12, Selected: day12, OnSelect: func(time.Time) {}})
	cell := cellFor(t, dayCells(t, both), 2, 12)
	if cell.Style.Background != theme.Colors.Primary {
		t.Error("a day that is both today and selected should be filled")
	}
	if cell.Style.BorderWidth != 0 {
		t.Errorf("border width = %v, want 0 — the ring under a fill is invisible and only costs a patch", cell.Style.BorderWidth)
	}
}

func TestCalendarAdjacentDaysAreDimmedAndInert(t *testing.T) {
	theme := core.DefaultTheme
	tapped := time.Time{}
	n := renderCalendar(t, Calendar{Month: sep2026, OnSelect: func(d time.Time) { tapped = d }})
	cells := dayCells(t, n)

	lead := cells[0] // August 30
	if !lead.Style.Disabled {
		t.Error("a leading adjacent day should be inert")
	}
	if lead.Children[0].Style.TextColor != theme.Colors.TextSecondary {
		t.Errorf("adjacent ink = %q, want TextSecondary", lead.Children[0].Style.TextColor)
	}
	// The handler is registered as a no-op rather than dropped, so a tap
	// already in flight when the patch lands finds something to call. The
	// platform is what refuses to dispatch.
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	fresh := Calendar{Month: sep2026, OnSelect: func(d time.Time) { tapped = d }}.Render(ctx)
	id, ok := dayCells(t, fresh)[0].Props["onClick"].(string)
	if !ok {
		t.Fatal("an inert cell should still register a handler")
	}
	ctx.TriggerCallback(id)
	if !tapped.IsZero() {
		t.Errorf("tapping an adjacent day selected %v; it should do nothing", tapped)
	}
}

func TestCalendarSelectReportsMiddayInTheCalendarsLocation(t *testing.T) {
	// A location whose offset is large enough that midnight UTC and midnight
	// local are different calendar days, so a widget that quietly worked in
	// UTC would hand back the wrong date.
	kiritimati := time.FixedZone("LINT", 14*3600)
	var got time.Time

	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := Calendar{
		Month:    time.Date(2026, time.September, 15, 0, 0, 0, 0, kiritimati),
		OnSelect: func(d time.Time) { got = d },
	}.Render(ctx)

	cell := cellFor(t, dayCells(t, n), 2, 12)
	ctx.TriggerCallback(cell.Props["onClick"].(string))

	if got.Location() != kiritimati {
		t.Errorf("location = %v, want the calendar's %v", got.Location(), kiritimati)
	}
	y, m, d := got.Date()
	if y != 2026 || m != time.September || d != 12 {
		t.Errorf("date = %d-%02d-%02d, want 2026-09-12", y, int(m), d)
	}
	if got.Hour() != 12 {
		t.Errorf("hour = %d, want 12 — cells are built at midday so no DST transition can move them off their day", got.Hour())
	}
}

// The reason cells are built at midday rather than midnight. Chile springs
// forward at 24:00, so 2026-09-06 has no 00:00 and Go resolves a request for
// it to 2026-09-05 23:00. A midnight grid emits two cells that both read as
// the 5th and never offers the 6th.
func TestCalendarSurvivesADaySkippedMidnight(t *testing.T) {
	loc, err := time.LoadLocation("America/Santiago")
	if err != nil {
		t.Skip("tzdata unavailable:", err)
	}
	// Precondition: this is still a day whose midnight does not exist.
	if d := time.Date(2026, time.September, 6, 0, 0, 0, 0, loc).Day(); d == 6 {
		t.Skip("2026-09-06 no longer skips midnight in this tzdata; the regression needs a new fixture")
	}

	n := renderCalendar(t, Calendar{Month: time.Date(2026, time.September, 15, 12, 0, 0, 0, loc)})
	cells := dayCells(t, n)

	// September 2026 starts on a Tuesday, so with a Sunday-start grid the
	// month's own run is cells 2..31. Built at midnight, the 6th's cell would
	// resolve back to the 5th and every day after it would be one short.
	if got := dayNumber(t, cellFor(t, cells, 2, 6)); got != "6" {
		t.Errorf("the cell where the 6th belongs shows %q — a skipped midnight has shifted the grid", got)
	}
	for day := 1; day <= 30; day++ {
		if got := dayNumber(t, cells[day+1]); got != itoa(day) {
			t.Fatalf("cell %d = %q, want %d — the month's run must be contiguous", day+1, got, day)
		}
	}
}

func TestCalendarRangeDisablesDaysAndArrows(t *testing.T) {
	theme := core.DefaultTheme
	n := renderCalendar(t, Calendar{
		Month:    sep2026,
		OnSelect: func(time.Time) {},
		// Inclusive on both ends, and Max carries a time of day to pin that
		// the comparison is by calendar day, not by instant.
		Min:           time.Date(2026, time.September, 10, 0, 0, 0, 0, time.UTC),
		Max:           time.Date(2026, time.September, 20, 15, 4, 0, 0, time.UTC),
		OnMonthChange: func(time.Time) {},
	})
	cells := dayCells(t, n)

	for _, tc := range []struct {
		day  int
		want bool // disabled
	}{{9, true}, {10, false}, {20, false}, {21, true}} {
		cell := cellFor(t, cells, 2, tc.day)
		if cell.Style.Disabled != tc.want {
			t.Errorf("Sept %d disabled = %v, want %v", tc.day, cell.Style.Disabled, tc.want)
		}
		if tc.want && cell.Children[0].Style.TextColor != theme.Colors.TextSecondary {
			t.Errorf("Sept %d should be dimmed like an adjacent day", tc.day)
		}
	}

	// Both arrows still reach a month with selectable days in it (August has
	// none below Min... but the range starts Sept 10, so August is empty and
	// the back arrow must be dead).
	header := n.Children[0]
	back, fwd := header.Children[0], header.Children[2]
	if !back.Style.Disabled {
		t.Error("the back arrow should be disabled: no day in August is inside [Min, Max]")
	}
	if !fwd.Style.Disabled {
		t.Error("the forward arrow should be disabled: no day in October is inside [Min, Max]")
	}
}

func TestCalendarArrowsReachAdjacentMonths(t *testing.T) {
	var got time.Time
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := Calendar{Month: sep2026, OnMonthChange: func(m time.Time) { got = m }}.Render(ctx)

	header := n.Children[0]
	if len(header.Children) != 3 {
		t.Fatalf("header children = %d, want back arrow, label, forward arrow", len(header.Children))
	}

	ctx.TriggerCallback(header.Children[0].Props["onClick"].(string))
	if y, m, d := got.Date(); y != 2026 || m != time.August || d != 1 {
		t.Errorf("back = %v, want the 1st of August", got)
	}
	ctx.TriggerCallback(header.Children[2].Props["onClick"].(string))
	if y, m, d := got.Date(); y != 2026 || m != time.October || d != 1 {
		t.Errorf("forward = %v, want the 1st of October", got)
	}
	// December → January is where a naive month+1 wraps into an invalid month
	// rather than the next year; time.Date normalizes it.
	ctx2 := core.NewContext()
	ctx2.BeginRenderPass()
	dec := Calendar{
		Month:         time.Date(2026, time.December, 20, 0, 0, 0, 0, time.UTC),
		OnMonthChange: func(m time.Time) { got = m },
	}.Render(ctx2)
	ctx2.TriggerCallback(dec.Children[0].Children[2].Props["onClick"].(string))
	if y, m := got.Year(), got.Month(); y != 2027 || m != time.January {
		t.Errorf("December + 1 = %v %v, want January 2027", m, y)
	}
}

func TestCalendarWithoutOnMonthChangeDrawsNoArrows(t *testing.T) {
	n := renderCalendar(t, Calendar{Month: sep2026})
	header := n.Children[0]
	if len(header.Children) != 1 {
		t.Errorf("header children = %d, want the label alone", len(header.Children))
	}
	if findText(header, "September 2026") == nil {
		t.Error("the month should still be named when there is nowhere to report a change")
	}
}

func TestCalendarMarkDotIsAlwaysPresentAndOnlySometimesInked(t *testing.T) {
	theme := core.DefaultTheme
	n := renderCalendar(t, Calendar{
		Month:  sep2026,
		Marked: func(d time.Time) bool { return d.Day() == 18 },
	})
	cells := dayCells(t, n)

	marked := cellFor(t, cells, 2, 18)
	plain := cellFor(t, cells, 2, 17)
	if len(marked.Children) != len(plain.Children) {
		t.Fatalf("marked cell has %d children, unmarked %d — the dot must not come and go",
			len(marked.Children), len(plain.Children))
	}
	if got := marked.Children[1].Style.Background; got != theme.Colors.Primary {
		t.Errorf("mark = %q, want Primary %q", got, theme.Colors.Primary)
	}
	if got := plain.Children[1].Style.Background; got != ColorTransparent {
		t.Errorf("unmarked dot = %q, want transparent so the day numbers keep one baseline", got)
	}

	// On a filled cell the dot takes the ink the number took: a Primary dot on
	// a Primary fill is not there.
	sel := renderCalendar(t, Calendar{
		Month:    sep2026,
		Selected: time.Date(2026, time.September, 18, 0, 0, 0, 0, time.UTC),
		Marked:   func(d time.Time) bool { return d.Day() == 18 },
		OnSelect: func(time.Time) {},
	})
	cell := cellFor(t, dayCells(t, sel), 2, 18)
	if cell.Children[1].Style.Background == theme.Colors.Primary {
		t.Error("a mark on the selected day must not be drawn in the fill's own color")
	}
	if cell.Children[1].Style.Background != cell.Children[0].Style.TextColor {
		t.Error("a mark on the selected day should take the day number's ink")
	}
}

func TestCalendarWithNoAnchorRendersNothing(t *testing.T) {
	// No Month, no Selected, no Today, and no clock in the widget to fall back
	// on. Drawing January of year 1 would look like data.
	n := renderCalendar(t, Calendar{OnSelect: func(time.Time) {}})
	if len(n.Children) != 0 {
		t.Errorf("children = %d, want none", len(n.Children))
	}
}

func TestCalendarAnchorFallsBackThroughSelectedThenToday(t *testing.T) {
	sel := renderCalendar(t, Calendar{Selected: time.Date(2026, time.March, 4, 0, 0, 0, 0, time.UTC)})
	if findText(sel.Children[0], "March 2026") == nil {
		t.Error("with no Month, the calendar should open on the selected day's month")
	}
	today := renderCalendar(t, Calendar{Today: time.Date(2026, time.July, 4, 0, 0, 0, 0, time.UTC)})
	if findText(today.Children[0], "July 2026") == nil {
		t.Error("with neither Month nor Selected, the calendar should open on Today's month")
	}
}

func TestCalendarWithoutOnSelectIsADisplay(t *testing.T) {
	n := renderCalendar(t, Calendar{Month: sep2026})
	cell := cellFor(t, dayCells(t, n), 2, 15)
	if !cell.Style.Disabled {
		t.Error("with no OnSelect every day should be inert — the grid is a month display, not a picker")
	}
}

func TestCalendarLabelSeamsAreTheLocalizationPoints(t *testing.T) {
	n := renderCalendar(t, Calendar{
		Month:        sep2026,
		Selected:     time.Date(2026, time.September, 12, 0, 0, 0, 0, time.UTC),
		Today:        time.Date(2026, time.September, 12, 0, 0, 0, 0, time.UTC),
		OnSelect:     func(time.Time) {},
		MonthLabel:   func(m time.Time) string { return "Setembro de 2026" },
		WeekdayLabel: func(w time.Weekday) string { return []string{"D", "S", "T", "Q", "Q", "S", "S"}[int(w)] },
		DayLabel:     func(d time.Time) string { return "dia " + itoa(d.Day()) },
	})

	if findText(n.Children[0], "Setembro de 2026") == nil {
		t.Error("MonthLabel should name the header")
	}
	if got := n.Children[1].Children[0].Children[0].Props["content"]; got != "D" {
		t.Errorf("first caption = %v, want the caller's %q", got, "D")
	}
	// The state suffixes are appended to whatever names the day, so a
	// translated calendar still announces its selection.
	cell := cellFor(t, dayCells(t, n), 2, 12)
	if got := cell.Style.AccessibilityLabel; got != "dia 12, today, selected" {
		t.Errorf("spoken name = %q, want the caller's name with the state appended", got)
	}
}

func TestCalendarHeaderOverrideReplacesTheMonthRow(t *testing.T) {
	n := renderCalendar(t, Calendar{
		Month:         sep2026,
		OnMonthChange: func(time.Time) {},
		Header:        core.Text("Pick a service"),
	})
	if findText(n, "September 2026") != nil {
		t.Error("an override replaces the default row, arrows and label together")
	}
	if findText(n, "Pick a service") == nil {
		t.Error("the override should be rendered in the header's place")
	}
	// The grid itself is untouched by the override.
	if len(dayCells(t, n)) != calendarRows*calendarCols {
		t.Error("the override should not disturb the grid")
	}
}

func TestCalendarStyleOverridesTheDefaults(t *testing.T) {
	n := renderCalendar(t, Calendar{Month: sep2026, Style: []core.StyleProp{
		core.Padding(10), core.Gap(0),
	}})
	if n.Style.Padding.Top != 10 {
		t.Errorf("padding = %+v, want the caller's override", n.Style.Padding)
	}
	if n.Style.Gap != 0 {
		t.Errorf("gap = %v, want the caller's 0", n.Style.Gap)
	}
}

// Cells divide the row evenly rather than sizing to their content, so a "9"
// and a "30" occupy the same column width and the captions above line up.
// FlexGrow alone converges only the two DOM targets; the zero basis is what
// makes CSS divide the whole axis the way Compose and SwiftUI already do.
func TestCalendarCellsShareTheRowEvenly(t *testing.T) {
	n := renderCalendar(t, Calendar{Month: sep2026})
	for i, cell := range dayCells(t, n) {
		if cell.Style.FlexGrow != 1 || cell.Style.FlexBasis != "0" {
			t.Fatalf("cell %d: grow %v basis %q, want 1 and \"0\"", i, cell.Style.FlexGrow, cell.Style.FlexBasis)
		}
	}
	for i, cap := range n.Children[1].Children {
		if cap.Style.FlexGrow != 1 || cap.Style.FlexBasis != "0" {
			t.Fatalf("caption %d is sized differently from the cells below it", i)
		}
	}
}

func TestYMDOrdersDates(t *testing.T) {
	d := func(y int, m time.Month, day int) time.Time {
		return time.Date(y, m, day, 12, 0, 0, 0, time.UTC)
	}
	if ymd(d(2026, time.March, 12)) != 20260312 {
		t.Errorf("ymd = %d, want 20260312", ymd(d(2026, time.March, 12)))
	}
	// Ordered comparison of the collapsed form is ordered comparison of dates,
	// across month and year boundaries where a naive day-of-month is not.
	if !(ymd(d(2026, time.January, 31)) < ymd(d(2026, time.February, 1))) {
		t.Error("Jan 31 should order before Feb 1")
	}
	if !(ymd(d(2025, time.December, 31)) < ymd(d(2026, time.January, 1))) {
		t.Error("the year boundary should order correctly")
	}
}

func TestSameDayComparesInTheCellsLocation(t *testing.T) {
	kiritimati := time.FixedZone("LINT", 14*3600)
	cell := time.Date(2026, time.September, 12, 12, 0, 0, 0, kiritimati)

	// 2026-09-11 23:00 UTC is 2026-09-12 13:00 in +14: the same day as the
	// cell, which is the question a reader comparing the grid to their own
	// week is asking.
	if !sameDay(cell, time.Date(2026, time.September, 11, 23, 0, 0, 0, time.UTC)) {
		t.Error("an instant should land on the day it falls on in the calendar's location")
	}
	if sameDay(cell, time.Date(2026, time.September, 12, 23, 0, 0, 0, time.UTC)) {
		t.Error("2026-09-12 23:00 UTC is the 13th in +14 and must not match the 12th")
	}
}
