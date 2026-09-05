package components

import (
	"fmt"
	"time"

	"github.com/rohanthewiz/grmob/core"
)

// Calendar is a controlled month grid: seven weekday captions over six rows of
// day cells, with a selected day, an optional "today" ring, optional dots for
// the days that have something on them, and arrows that ask the caller to
// change month.
//
//	┌─────────────────────────────────────────┐
//	│  ‹        March 2026              ›     │  <- Header (arrows only if OnMonthChange)
//	│  Su  Mo  Tu  We  Th  Fr  Sa             │  <- weekday captions
//	│  ·1   2   3   4   5   6   7             │  <- ·n = Marked(n) is true
//	│   8   9  10  11 [12] 13  14             │  <- [n] = Selected
//	│  15  16  17 ·18  19  20  21             │
//	│  22  23  24  25  26  27  28             │
//	│  29  30  31   1   2   3   4             │  <- adjacent days, dimmed and inert
//	│   5   6   7   8   9  10  11             │
//	└─────────────────────────────────────────┘
//
//	month := core.NewState(ctx, someDate)
//	components.Calendar{
//	    Month:         month.Get(),
//	    OnMonthChange: month.Set,
//	    Selected:      picked.Get(),
//	    OnSelect:      picked.Set,
//	    Today:         today,                 // see "The widget never asks what time it is"
//	    Marked:        func(d time.Time) bool { return len(eventsOn(d)) > 0 },
//	}
//
// # Everything is the caller's, including which month is on screen
//
// The widget holds no state and calls no hook, so it may be rendered
// conditionally — the same contract every widget here but Accordion keeps.
// That extends to the *visible month*, which looks like private view state and
// is not: an events screen wants to open on the month of the next event, a
// booking form wants to jump to the month a search result lands in, and a
// screen with two calendars side by side wants them to move together. A
// widget that owned its month could serve none of those.
//
// The cost is one piece of state at the call site, as above. DatePicker is the
// packaging of exactly that state for the form case; reach for this when the
// grid is part of the screen rather than behind a field.
//
// A nil OnMonthChange draws no arrows. That is the static case — a month with
// its events dotted, printed into a page — not a broken one.
//
// # The widget never asks what time it is
//
// There is no time.Now() in here, and Today is a field rather than something
// the widget works out. Three reasons, in ascending order of how much they
// bite:
//
//   - A render that reads the clock is not a pure function of its inputs, so
//     a snapshot test of "the March 2026 grid" would drift into a different
//     picture every midnight.
//   - "Today" is a question about a time zone, and the widget cannot know
//     whether it should answer in the device's zone, the congregation's, or
//     the one the data is stamped in. The caller can.
//   - A zero Today draws no ring, which is the honest rendering of a calendar
//     that has not been told what day it is.
//
// # The month is always six rows
//
// A month spans four to six weeks depending on its length and the weekday it
// starts on, and a grid that grew and shrank with it would change height as
// the reader pages through the year — pushing whatever sits below the
// calendar up and down on every arrow tap. So the grid is always 6×7 = 42
// cells, padded at both ends with the adjacent months' days.
//
// The fixed shape pays a second time on the reconciler: changing month patches
// 42 text contents and their styles and touches no structure at all, where a
// variable grid would add and remove whole rows.
//
// Those adjacent days are drawn dimmed and are *inert*. They are there so the
// grid reads as a grid — six full rows, no ragged hole — not as targets. A
// controlled calendar cannot move its own month, so a tap on the trailing "2"
// under March would either select a date the visible grid no longer highlights
// or fire two callbacks the caller has to sequence; the arrows are the one
// deliberate way the month changes.
//
// # Dates in, dates out: midday, not midnight
//
// Every cell is built at **12:00 in the calendar's location**, and that is the
// value handed to Marked and OnSelect. It looks like it should be midnight and
// must not be.
//
// Midnight does not exist on every calendar day. Chile springs forward at
// 24:00, so 2026-09-06 in America/Santiago begins at 01:00 and Go resolves
// time.Date(2026, 9, 6, 0, 0, 0, 0, santiago) to *2026-09-05 23:00* — the
// previous day. A grid built at midnight would therefore emit two cells that
// both read as September 5, and September 6 would be unselectable, in exactly
// the zones nobody testing in UTC ever looks at. Midday is skipped by no
// transition in the tz database.
//
// So the value is an instant *inside* the intended day rather than at its
// edge. A caller who wants the day itself takes the triple that a date
// actually is:
//
//	OnSelect: func(d time.Time) { y, m, day := d.Date(); … }
//
// # Which location
//
// The calendar works in the location of its anchor — Month if set, else
// Selected, else Today — and converts Selected, Min, Max and Today into it
// before reducing them to a calendar day. An event stamped in UTC therefore
// lands on the day it happened *locally*, which is what a reader comparing
// the grid to their own week expects; a caller who means otherwise passes a
// Month already in the location they mean.
//
// With all three of Month, Selected and Today zero there is no anchor and no
// clock to fall back on, and the widget renders nothing.
//
// # Localization
//
// Go's time package formats in English only, so MonthLabel, WeekdayLabel and
// DayLabel are the seams for everything a reader sees as a word. The day
// *numbers* are numerals and are not routed through anything.
type Calendar struct {
	// Month is any instant within the month to draw; only its year and month
	// (and its location) are read. Zero falls back to Selected, then to
	// Today; with all three zero the widget renders nothing.
	Month time.Time

	// OnMonthChange is asked to move a month back or forward, and receives
	// midday on the first of the new month. Nil draws no arrows.
	OnMonthChange func(time.Time)

	// Selected is the highlighted day; zero highlights none. Compared by
	// calendar day in the calendar's location, so any instant during the day
	// selects it.
	Selected time.Time

	// OnSelect fires with the tapped day at midday in the calendar's location
	// (see the type comment). Nil renders an inert grid — a month display
	// rather than a picker.
	OnSelect func(time.Time)

	// Today rings the current day without selecting it, so "today" and "the
	// day I picked" can be two different cells and both be visible. Zero
	// draws no ring; the widget does not consult the clock.
	Today time.Time

	// Min and Max bound the selectable range, inclusive and compared by
	// calendar day — a Max of "today at 15:04" still includes today. Days
	// outside are drawn like adjacent days and are inert, and an arrow whose
	// whole target month lies outside the range is disabled.
	//
	// Zero on either side is unbounded.
	Min time.Time
	Max time.Time

	// Marked puts a dot under a day: the days with an event, a deadline, a
	// service. It is called once per visible cell — 42 times per render,
	// adjacent months included — so it should be a lookup, not a query.
	Marked func(time.Time) bool

	// WeekStart is the weekday the grid's leftmost column is. The zero value
	// is time.Sunday, which is also the intended default.
	WeekStart time.Weekday

	// MonthLabel names the month in the header; nil gives "January 2006".
	// WeekdayLabel captions a column; nil gives the first two letters of the
	// English name ("Su", "Mo", …). DayLabel is the *spoken* name of a cell
	// for a screen reader; nil gives "Monday, January 2, 2006", to which the
	// widget appends ", selected" and ", today" as they apply.
	MonthLabel   func(time.Time) string
	WeekdayLabel func(time.Weekday) string
	DayLabel     func(time.Time) string

	// Header replaces the default month row, arrows and all — for a screen
	// whose own chrome already carries the month, or one that navigates by
	// year. Navigation is then entirely yours: OnMonthChange is not called
	// from anywhere else.
	//
	// To drop the header without replacing it, pass core.Fragment().
	Header core.View

	// Style is applied to the outer column after its defaults.
	Style []core.StyleProp
}

// The grid is always this shape; see "The month is always six rows".
const (
	calendarRows = 6
	calendarCols = 7
)

// calendarDotSize is the diameter of the Marked dot, in px. Small enough to
// sit under a day number without changing the cell's rhythm, large enough to
// survive a phone's pixel grid.
const calendarDotSize = 5

func (c Calendar) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	anchor, ok := c.anchor()
	if !ok {
		// No month, no selection, no today, and no clock to fall back on.
		// Rendering an arbitrary month (the zero time's January of year 1)
		// would be worse than rendering nothing: it looks like data.
		return core.Box().Render(ctx)
	}
	loc := anchor.Location()
	first := monthFirst(anchor, loc)

	items := make([]core.PropsAndChildren, 0, len(c.Style)+calendarRows+4)
	// Shed the theme Column's inset and set the grid's own rhythm: a hairline
	// of air between the week rows, nothing else. The cells carry their own
	// padding, so the column must not add a second one.
	items = append(items, core.Padding(0), core.Gap(float64(t.Spacing.XS)))
	for _, sp := range c.Style {
		items = append(items, sp)
	}

	if c.Header != nil {
		items = append(items, c.Header)
	} else {
		items = append(items, c.monthHeader(ctx, first, loc))
	}
	items = append(items, c.weekdayRow(ctx))

	// Column-major offset of the first cell: how many days of the previous
	// month have to precede the 1st for it to land under its own weekday.
	// The +7 keeps the modulo non-negative for any WeekStart.
	lead := int(first.Weekday()-c.WeekStart+7) % 7
	year, month, _ := first.Date()

	for row := range calendarRows {
		cells := make([]core.PropsAndChildren, 0, calendarCols+1)
		// Gap(0) between cells: the day pills are the grid's visible
		// structure and they should tile, not float. Padding(0) sheds the
		// theme Row's inset for the same reason the column above does.
		cells = append(cells, core.Padding(0), core.Gap(0))
		for col := range calendarCols {
			// time.Date normalizes an out-of-range day, so day 0 is the last
			// of the previous month and day 32 the 1st of the next — the
			// whole 42-cell window falls out of one expression with no
			// day-by-day addition to drift across a DST boundary. Midday, not
			// midnight; see the type comment.
			day := 1 - lead + row*calendarCols + col
			cells = append(cells, c.dayCell(ctx, time.Date(year, month, day, 12, 0, 0, 0, loc), month))
		}
		items = append(items, core.Row(cells...))
	}

	return core.Column(items...).Render(ctx)
}

// anchor resolves the month the grid is drawn around: Month, else Selected,
// else Today. The bool is false when all three are zero, which is the one
// state with no answer — there is no clock in here to ask.
func (c Calendar) anchor() (time.Time, bool) {
	for _, t := range []time.Time{c.Month, c.Selected, c.Today} {
		if !t.IsZero() {
			return t, true
		}
	}
	return time.Time{}, false
}

// monthHeader is the default month row: a back arrow, the month's name, a
// forward arrow. The arrows appear only when there is somewhere to report a
// change to, and each is disabled when its whole target month lies outside
// [Min, Max] — a bounded picker should not offer a month with nothing
// choosable in it.
func (c Calendar) monthHeader(ctx *core.Context, first time.Time, loc *time.Location) core.View {
	t := ctx.Theme()

	label := first.Format("January 2006")
	if c.MonthLabel != nil {
		label = c.MonthLabel(first)
	}

	items := make([]core.PropsAndChildren, 0, 5)
	items = append(items,
		core.Padding(0),
		core.PaddingVertical(t.Spacing.XS),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.Gap(float64(t.Spacing.SM)),
	)

	if c.OnMonthChange != nil {
		items = append(items, c.monthArrow(t, first, loc, -1, "‹", "Previous month"))
	}
	// The label grows so the two arrows sit hard against the edges — the
	// FlexGrow-not-JustifyBetween pinning GroupHeader and ListRow settled on,
	// which keeps working when one arrow is absent.
	items = append(items, core.Box(
		core.FlexGrow(1),
		core.Text(label,
			core.UseStyle(t.Typography.Body),
			core.FontWeight(core.Bold),
			core.Align(core.AlignCenter),
		),
	))
	if c.OnMonthChange != nil {
		items = append(items, c.monthArrow(t, first, loc, +1, "›", "Next month"))
	}

	return core.Row(items...)
}

// monthArrow builds one navigation button. delta is in months.
func (c Calendar) monthArrow(t *core.Theme, first time.Time, loc *time.Location, delta int, glyph, name string) core.View {
	year, month, _ := first.Date()
	// Midday again, and on the 1st: the target is a month, and AddDate on a
	// 31st would skid into the following month for the short ones.
	target := time.Date(year, month+time.Month(delta), 1, 12, 0, 0, 0, loc)

	// A month is reachable when any of its days is inside the range. Only the
	// nearest edge has to be tested: moving back, the *last* day of the target
	// month is the closest it gets to Min; moving forward, the first.
	reachable := true
	switch {
	case delta < 0 && !c.Min.IsZero():
		lastOfTarget := time.Date(year, month, 0, 12, 0, 0, 0, loc)
		reachable = ymd(lastOfTarget) >= ymd(c.Min.In(loc))
	case delta > 0 && !c.Max.IsZero():
		reachable = ymd(target) <= ymd(c.Max.In(loc))
	}

	return Button{
		Label:              glyph,
		Emphasis:           EmphasisGhost,
		Disabled:           !reachable,
		AccessibilityLabel: name,
		OnTap:              func() { c.OnMonthChange(target) },
		// The theme's Button base is sized for a word and lit for a raised
		// surface; a chevron needs neither. The horizontal inset comes down to
		// the SM step so two arrows do not eat a third of the header, and the
		// elevation goes to zero because a ghost button that casts a shadow is
		// a rectangle the reader can see but not find the edges of.
		Style: []core.StyleProp{
			core.PaddingHorizontal(t.Spacing.SM),
			core.Shadow(0),
		},
	}
}

// weekdayRow is the caption line above the grid. Its cells are sized exactly
// as the day cells below so the columns line up: equal share of the row, no
// content-driven width.
func (c Calendar) weekdayRow(ctx *core.Context) core.View {
	t := ctx.Theme()

	items := make([]core.PropsAndChildren, 0, calendarCols+2)
	items = append(items, core.Padding(0), core.Gap(0))
	for col := range calendarCols {
		wd := time.Weekday((int(c.WeekStart) + col) % 7)
		items = append(items, core.Box(
			core.FlexGrow(1),
			core.FlexBasis("0"),
			core.PaddingVertical(t.Spacing.XS),
			core.Text(c.weekdayLabel(wd),
				core.UseStyle(t.Typography.Caption),
				core.Align(core.AlignCenter),
			),
			// The captions repeat the information the day cells already carry
			// in their spoken names ("Monday, March 2, 2026"), so a reader
			// walking the grid would hear the weekday twice. They are
			// decoration for the eye.
			core.AccessibilityHidden(),
		))
	}
	return core.Row(items...)
}

func (c Calendar) weekdayLabel(wd time.Weekday) string {
	if c.WeekdayLabel != nil {
		return c.WeekdayLabel(wd)
	}
	// Two letters distinguish all seven English names; one does not (S, T).
	return wd.String()[:2]
}

// dayCell renders one day of the 42. day is midday in the calendar's location;
// month is the month being displayed, which is what makes a cell "adjacent".
func (c Calendar) dayCell(ctx *core.Context, day time.Time, month time.Month) core.View {
	t := ctx.Theme()

	_, dayMonth, dayNum := day.Date()
	adjacent := dayMonth != month
	inRange := c.inRange(day)
	// Adjacent days are context, not targets (see the type comment); an
	// out-of-range day is not a target either. Both read as "not available"
	// and are drawn the same way, because to the reader they are the same
	// thing.
	selectable := !adjacent && inRange && c.OnSelect != nil

	selected := !c.Selected.IsZero() && sameDay(day, c.Selected)
	isToday := !c.Today.IsZero() && sameDay(day, c.Today)

	ink := t.Colors.TextPrimary
	if adjacent || !inRange {
		ink = t.Colors.TextSecondary
	}

	items := make([]core.PropsAndChildren, 0, 12)
	items = append(items,
		core.FlexGrow(1),
		core.FlexBasis("0"),
		core.PaddingVertical(t.Spacing.SM),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.Gap(0),
		core.BorderRadius(float64(t.Spacing.SM)),
	)

	dot := t.Colors.Primary
	switch {
	case selected:
		// The fill is the strongest thing in the grid, so the ink is picked
		// against it rather than assumed — Primary is a light blue in one
		// bundled theme and a dark indigo in another, and a hard-coded white
		// is unreadable on the first.
		ink = contrastInk(t.Colors.Primary, t.Colors.Background, t.Colors.TextPrimary)
		dot = ink
		items = append(items, core.BackgroundColor(t.Colors.Primary))
	case isToday:
		// A ring rather than a fill, so today and the selected day are two
		// distinguishable cells. Under a selection the ring would be Primary
		// on Primary — invisible — which is why this arm is the fallthrough.
		items = append(items,
			core.BorderWidth(1),
			core.BorderColor(t.Colors.Primary),
		)
	}

	if selectable {
		d := day // captured per cell; the closure outlives this pass
		items = append(items, core.OnClick(func() { c.OnSelect(d) }))
	} else {
		// Disabled *and* a no-op handler, the pairing components.Button
		// settled on: core.Disabled is what makes every renderer refuse to
		// dispatch (pointer-events on the web, enabled=false in Compose,
		// .disabled in SwiftUI) and announce the state, while a registered
		// no-op closes the window in which a tap already in flight arrives at
		// a callback ID the patch has just emptied.
		items = append(items,
			core.Disabled(true),
			core.OnClick(func() {}),
		)
	}
	items = append(items,
		core.AccessibilityLabel(c.dayLabel(day, selected, isToday)),
	)

	items = append(items, core.Text(itoa(dayNum),
		core.UseStyle(t.Typography.Body),
		core.TextColor(ink),
		core.Align(core.AlignCenter),
	))

	// The dot is always in the tree and goes transparent when the day is not
	// marked, rather than appearing and disappearing. Two things follow: the
	// day numbers keep the same baseline whether or not their day has
	// something on it, and toggling a mark is a color patch rather than a
	// child insertion in the middle of a 42-cell grid.
	marked := c.Marked != nil && c.Marked(day)
	if !marked {
		dot = ColorTransparent
	}
	items = append(items, core.Box(
		core.Width(fmt.Sprintf("%dpx", calendarDotSize)),
		core.Height(fmt.Sprintf("%dpx", calendarDotSize)),
		core.BorderRadius(calendarDotSize),
		core.BackgroundColor(dot),
		// The mark's meaning belongs to the cell's spoken name, not to a
		// nameless box a reader would otherwise stop on.
		core.AccessibilityHidden(),
	))

	// Box, not Column: a Column would arrive with the theme's screen inset,
	// and 16px of horizontal padding inside a cell one seventh of a row wide
	// leaves no room for a two-digit day.
	//
	// Deliberately not Keyed either. The grid's 42 cells never reorder — the
	// shape is fixed — so positional matching is exactly right, and keying
	// them by date would make every month change a wholesale replacement of
	// the children it is supposed to be a patch of.
	return core.Box(items...)
}

// dayLabel is the cell's spoken name. The state suffixes are appended to
// whatever names the day — a caller's DayLabel included — so a translated
// calendar still announces its selection.
func (c Calendar) dayLabel(day time.Time, selected, isToday bool) string {
	label := day.Format("Monday, January 2, 2006")
	if c.DayLabel != nil {
		label = c.DayLabel(day)
	}
	if isToday {
		label += ", today"
	}
	if selected {
		label += ", selected"
	}
	return label
}

// inRange reports whether day falls inside [Min, Max], compared by calendar
// day in day's own location so a Max stamped mid-afternoon still includes its
// own day.
func (c Calendar) inRange(day time.Time) bool {
	d := ymd(day)
	if !c.Min.IsZero() && d < ymd(c.Min.In(day.Location())) {
		return false
	}
	if !c.Max.IsZero() && d > ymd(c.Max.In(day.Location())) {
		return false
	}
	return true
}

// monthFirst is midday on the first of t's month, in loc. Midday for the
// reason the type comment gives; the first because the grid is laid out from
// the weekday the month starts on.
func monthFirst(t time.Time, loc *time.Location) time.Time {
	y, m, _ := t.In(loc).Date()
	return time.Date(y, m, 1, 12, 0, 0, 0, loc)
}

// sameDay compares two instants as calendar days, reading b in a's location.
// a is always a grid cell, so this asks the question the reader is asking:
// does the thing b names fall on the day this square is?
func sameDay(a, b time.Time) bool {
	return ymd(a) == ymd(b.In(a.Location()))
}

// ymd collapses an instant to a single comparable number for its calendar
// day in its own location: 2026-03-12 becomes 20260312. Ordered comparison of
// these is ordered comparison of dates, which is what Min/Max need and what
// time.Before cannot give without first normalizing both sides to the same
// hour.
func ymd(t time.Time) int {
	y, m, d := t.Date()
	return y*10000 + int(m)*100 + d
}
