package comps

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
//	│  ·1   2   3   4   5   6 ··7             │  <- ·n = one mark, ··n = two
//	│   8   9  10  11 [12] 13  14             │  <- [n] = Selected
//	│  15  16  17 ·18  19  20  21             │
//	│  22  23  24  25  26  27  28             │
//	│  29  30  31   1   2   3   4             │  <- adjacent days, dimmed and inert
//	│   5   6   7   8   9  10  11             │
//	└─────────────────────────────────────────┘
//
//	month := core.NewState(ctx, someDate)
//	comps.Calendar{
//	    Month:         month.Get(),
//	    OnMonthChange: month.Set,
//	    Selected:      picked.Get(),
//	    OnSelect:      picked.Set,
//	    Today:         today,                 // see "The widget never asks what time it is"
//	    Marked:        func(d time.Time) int { return len(eventsOn(d)) },
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
// # The zero time goes both ways
//
// Selected already spells "nothing is chosen" as the zero time. With
// Deselectable set, OnSelect reports that same zero when the reader taps the
// day that is already selected — so a grid used as a *filter* is cleared from
// the grid itself, the value making a round trip through the caller's state
// with no second callback to wire and nothing beside the calendar to build.
//
// It is off by default, and the default is the interesting half. A picker
// asking which day the appointment is has no "no day" to offer, and there a
// stray second tap that quietly emptied the field would lose an answer the
// reader never asked to lose. So the widget does not guess which of the two
// it is in: a filter opts in, a picker leaves it alone, and DatePicker forces
// it off and puts the way out on its own Clear button.
//
// There is deliberately no OnDeselect. Two callbacks setting the same piece
// of state is two things for every consumer to keep in step, and the day a
// screen wants to tell them apart it can test for the zero it was handed.
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

	// Deselectable makes a tap on the already-selected day report the zero
	// time through OnSelect instead of the day, so a calendar standing in for
	// a filter can be un-set without a "Show all" button beside it. Off by
	// default; see "The zero time goes both ways" for why the default is that
	// way round.
	//
	// It changes what a tap *reports* and nothing about how the cell is drawn.
	// What it does change is how well the announcement fits: every cell states
	// core.AccessibilitySelected, which reaches the web as aria-pressed, and a
	// pressed toggle button that un-presses when you activate it is exactly
	// what a deselectable day is. Without this field the cell is a toggle that
	// only turns on, which is the honest report of a grid where the selection
	// can move but not clear.
	//
	// That state used to be a ", selected" suffix on the spoken name, because
	// core.Style had no slot for a state. It has one now, and the suffix is
	// gone — see dayLabel.
	Deselectable bool

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

	// Marked counts what a day has on it — events, deadlines, services — and
	// the cell draws that many dots under its number, capped at
	// calendarMaxDots. Zero draws none, and so does a negative; nil is the
	// same as a function that always answers zero.
	//
	// A count rather than a bool because two services on one Sunday and one
	// service on one Sunday are different facts about the day, and a reader
	// scanning a month for its busy weeks is asking exactly that question. A
	// caller holding only a yes/no writes it as a count and loses nothing:
	//
	//	Marked: func(d time.Time) int { if hasEvent(d) { return 1 }; return 0 }
	//
	// It is called once per visible cell — 42 times per render, adjacent
	// months included — so it should be a lookup, not a query.
	//
	// The dots are decoration and hidden from assistive technology. The widget
	// knows how many things a day holds and nothing about what any of them is,
	// so there is nothing it could truthfully announce; a count worth speaking
	// goes into the spoken name through DayLabel, which is the seam that
	// knows.
	Marked func(time.Time) int

	// WeekStart is the weekday the grid's leftmost column is. The zero value
	// is time.Sunday, which is also the intended default.
	WeekStart time.Weekday

	// MonthLabel names the month in the header; nil gives "January 2006".
	// WeekdayLabel captions a column; nil gives the first two letters of the
	// English name ("Su", "Mo", …). DayLabel is the *spoken* name of a cell
	// for a screen reader; nil gives "Monday, January 2, 2006", to which the
	// widget appends ", today" when it applies. The selection is not part of
	// the name — it is announced as the control state it is; see dayLabel.
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

// calendarDotSize is the diameter of one mark dot in px, calendarDotGap the
// air between two. Small enough to sit under a day number without changing the
// cell's rhythm, large enough to survive a phone's pixel grid.
//
// Neither is a theme spacing step, and that is deliberate: the cluster is
// glyph-scale furniture *inside* a cell rather than part of the screen's
// layout rhythm, and the theme's smallest step (XS, 4px) is already most of a
// dot. Tying them to the scale would mean a theme that loosened its spacing
// pushed three dots wider than the cell that holds them.
const (
	calendarDotSize = 5
	calendarDotGap  = 3
)

// calendarMaxDots caps the cluster.
//
// Three is the most a cell one seventh of a row wide can carry and still read
// as a count rather than a smudge: 3 dots and 2 gaps is 21px, against roughly
// 45px of cell on a narrow phone. It is also about where counting stops being
// what the reader does — past three the answer they take away is "several",
// which is exactly what a capped cluster says. An exact number that matters
// belongs in DayLabel, where a screen reader can read it out.
const calendarMaxDots = 3

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
		// The fill is the strongest thing in the grid, so the ink is resolved
		// against it rather than hard-coded — Primary is a light blue in one
		// bundled theme and a dark indigo in another, and one literal cannot
		// be read on both.
		//
		// Resolved, not measured. This used to call contrastInk directly and
		// so returned the higher ratio, which on DefaultTheme's Primary of the
		// day (iOS systemBlue) was *black* — a black numeral on system blue,
		// disagreeing with every filled comps.Button on the same screen,
		// which paints the white the theme's own Button base declares. inkOn
		// reads that declaration first and falls back to the measurement for a
		// fill the theme has said nothing about; see its doc for why a third
		// ink role could not have settled this.
		//
		// DefaultTheme's Primary has since been darkened to Apple's accessible
		// blue, so under it the two rules now agree — but the call still has
		// to be inkOn rather than contrastInk, because they part company again
		// under any theme whose brand colour is a mid-tone.
		ink = inkOn(t, t.Colors.Primary)
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
		// A second tap on the day already chosen reports "nothing", which is
		// the value Selected itself uses to mean that. Resolved here rather
		// than inside the closure so the cell captures a settled value: the
		// closure outlives the pass, and `selected` will not be true of this
		// cell forever. See "The zero time goes both ways".
		if c.Deselectable && selected {
			d = time.Time{}
		}
		items = append(items, core.OnClick(func() { c.OnSelect(d) }))
	} else {
		// Disabled *and* a no-op handler, the pairing comps.Button
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
		core.AccessibilityLabel(c.dayLabel(day, isToday)),
		// Which day is chosen, as a state rather than as part of the name.
		// Paired with the role below: a cell is a button in this vocabulary,
		// so "on" is spelled aria-pressed on the web and the platform's own
		// selected property on the two natives.
		//
		// ARIA's own date-picker pattern would say this differently — a grid
		// of role="gridcell" carrying aria-selected — and core.Role has no
		// value for a gridcell, deliberately: the role would oblige the whole
		// scaffold around it (a grid, rows, and the roving focus a grid
		// promises) and a lone gridcell inside plain divs describes a table
		// with no table, which role.go's structural rule calls worse than no
		// role at all. A pressed toggle button is the true thing this widget
		// can say about itself as it is actually built.
		//
		// Stated on every cell in the grid, including the adjacent and
		// out-of-range ones. They are already announced as disabled buttons;
		// a cell that said nothing about its state would be the one square
		// the reader could not place, and "not pressed" is exactly what an
		// unselectable day is.
		core.AccessibilitySelected(core.SelectedWhen(selected)),
		// A day cell is a Box with a tap handler, which every renderer draws
		// as scenery and every screen reader announces as text — the label
		// above names it and nothing said it could be activated. The role is
		// what makes forty-two of them controls rather than a paragraph.
		//
		// Set on every cell, including the unselectable ones: a disabled
		// button is still a button, and the renderers announce the disabled
		// state separately (core.Disabled, applied just above). A cell that
		// dropped the role when it went out of range would change kind as the
		// reader paged, which is stranger than a dimmed control.
		core.AccessibilityRole(core.RoleButton),
	)

	items = append(items, core.Text(itoa(dayNum),
		core.UseStyle(t.Typography.Body),
		core.TextColor(ink),
		core.Align(core.AlignCenter),
	))

	// One dot per thing on the day, capped, in a row under the number.
	//
	// The cluster is always in the tree and always holds at least one box,
	// drawn transparent when the day has nothing on it. That keeps both
	// properties the single dot this replaced was there for: the day numbers
	// sit on one baseline whether or not their day is marked, and the common
	// transition — nothing to one thing and back, which is every cell on a
	// yes/no calendar and most cells on any other — stays a color patch
	// rather than a child insertion in the middle of a 42-cell grid.
	//
	// Only the second and third dots are structural, and only on the cells
	// whose count actually reaches them. That is the price of counting and it
	// is charged to the cells doing the counting.
	count := 0
	if c.Marked != nil {
		count = c.Marked(day)
	}
	// A negative answer means the same as none. Clamping rather than trusting
	// keeps a caller's `len(x) - 1` slip from asking for a negative number of
	// children.
	count = min(max(count, 0), calendarMaxDots)

	dots := make([]core.PropsAndChildren, 0, calendarMaxDots+3)
	dots = append(dots,
		// Shed the theme Row's inset, as the week rows above do: this is a
		// cluster of marks, not a band of content. The cell's own
		// AlignItemsCenter is what centers it, so there is no Justify here —
		// the row hugs its dots on every target.
		core.Padding(0),
		core.Gap(calendarDotGap),
		// The marks' meaning belongs to the cell's spoken name — DayLabel is
		// the seam that knows what they are — not to a row of nameless boxes a
		// reader would otherwise stop on. Hidden on the row, which takes the
		// subtree with it.
		core.AccessibilityHidden(),
	)
	for i := range max(count, 1) {
		fill := dot
		if i >= count {
			// The one placeholder dot on an unmarked day; see above.
			fill = ColorTransparent
		}
		dots = append(dots, core.Box(
			core.Width(fmt.Sprintf("%dpx", calendarDotSize)),
			core.Height(fmt.Sprintf("%dpx", calendarDotSize)),
			core.BorderRadius(calendarDotSize),
			core.BackgroundColor(fill),
		))
	}
	items = append(items, core.Row(dots...))

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

// dayLabel is the cell's spoken name. The "today" suffix is appended to
// whatever names the day — a caller's DayLabel included — so a translated
// calendar still announces which square is today.
//
// The selection used to be a second suffix here and is not any more: it goes
// out as core.AccessibilitySelected, which every renderer announces as a
// control's state. Two announcements of one fact is the reasoning
// comps.Button gives for dropping its own ", disabled" suffix, and the
// state is the better half to keep — a name is meant to be stable, so a
// reader re-announcing the cell after a tap read out the whole altered name
// rather than the one thing that changed.
//
// "Today" stays a suffix because it is not a state a control can be in. There
// is no platform property for "this is the current date"; it is a fact about
// the day the cell names, which is what a name is for.
func (c Calendar) dayLabel(day time.Time, isToday bool) string {
	label := day.Format("Monday, January 2, 2006")
	if c.DayLabel != nil {
		label = c.DayLabel(day)
	}
	if isToday {
		label += ", today"
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
