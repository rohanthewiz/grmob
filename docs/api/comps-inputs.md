# Package comps — Inputs & pickers

```go
import "github.com/rohanthewiz/grmob/comps"
```

Form fields, one-time code fields, search, searchable selects, radio groups, dates and calendars, and the two editors.

One of 7 topic pages of [package comps](comps.md), which has the package overview and an index of every topic. This page documents the declarations in `comps/form_field.go`, `comps/pin_input.go`, `comps/search_field.go`, `comps/searchable_select.go`, `comps/radio_group.go`, `comps/date_picker.go`, `comps/calendar.go`, `comps/code_editor.go`, `comps/rich_text_editor.go`.

## Index

- [Constants](#constants) — `ConcernPINInputInert`, `ConcernPINValueTooLong`, `RichToolLink`
- [Variables](#variables) — `RichToolbarDefault`
- [`type Calendar`](#type-calendar)
    - [`func (Calendar) Render`](#func-calendar-render)
- [`type CodeEditor`](#type-codeeditor)
    - [`func (CodeEditor) Render`](#func-codeeditor-render)
- [`type DatePicker`](#type-datepicker)
    - [`func (DatePicker) Render`](#func-datepicker-render)
- [`type FormField`](#type-formfield)
    - [`func (FormField) Render`](#func-formfield-render)
- [`type PINInput`](#type-pininput)
    - [`func (PINInput) Render`](#func-pininput-render)
- [`type RadioGroup`](#type-radiogroup)
    - [`func (RadioGroup) Render`](#func-radiogroup-render)
- [`type RadioOption`](#type-radiooption)
- [`type RichTextEditor`](#type-richtexteditor)
    - [`func (RichTextEditor) Render`](#func-richtexteditor-render)
- [`type RichToolItem`](#type-richtoolitem)
- [`type RichToolbar`](#type-richtoolbar)
    - [`func UseRichToolbar`](#func-userichtoolbar)
    - [`func (*RichToolbar) Selection`](#func-richtoolbar-selection)
- [`type SearchField`](#type-searchfield)
    - [`func (SearchField) Render`](#func-searchfield-render)
- [`type SearchableSelect`](#type-searchableselect)
    - [`func (SearchableSelect) Render`](#func-searchableselect-render)

## Constants

ConcernPINInputInert is raised, in debug builds only, when a PINInput has no OnChange. The field is then read-only in practice — every keystroke reaches the handler, is discarded, and the next pass paints Value back over it — and on screen an inert PINInput is indistinguishable from one nobody has typed into yet. Disclosure's inert case is reported for the same reason: a widget that cannot do the one thing it exists for should say so somewhere other than in a bug report.

```go
const ConcernPINInputInert = "pin-input-inert"
```

<small>[comps/pin_input.go:16](https://github.com/rohanthewiz/grmob/blob/master/comps/pin_input.go#L16)</small>

ConcernPINValueTooLong is raised, in debug builds only, when Value holds more characters than there are cells. The extra ones are not drawn and can never be typed away, so a field that looks full is carrying a value its caller cannot see — and OnComplete's "the code is as long as the field" test would be met by characters nobody entered.

```go
const ConcernPINValueTooLong = "pin-value-too-long"
```

<small>[comps/pin_input.go:23](https://github.com/rohanthewiz/grmob/blob/master/comps/pin_input.go#L23)</small>

RichToolLink is the sentinel Command that opens the link prompt. Not a core command: core.EditLink needs a URL, and the prompt is where one comes from.

```go
const RichToolLink = "components:link"
```

<small>[comps/rich_text_editor.go:109](https://github.com/rohanthewiz/grmob/blob/master/comps/rich_text_editor.go#L109)</small>

## Variables

RichToolbarDefault is the toolbar most notes want: the five marks, three block kinds, the two lists, a quote, and the link prompt.

A var rather than a func so a caller can take a slice of it, append to it, or reorder it — which is the whole reason the toolbar is a list of items rather than a bool. It is package state, so treat it as read-only; UseRichToolbar copies it.

```go
var RichToolbarDefault = []RichToolItem{
	{Label: "B", Command: core.EditBold, AccessibilityLabel: "Bold"},
	{Label: "I", Command: core.EditItalic, AccessibilityLabel: "Italic"},
	{Label: "U", Command: core.EditUnderline, AccessibilityLabel: "Underline"},
	{Label: "S", Command: core.EditStrike, AccessibilityLabel: "Strikethrough"},
	{Label: "</>", Command: core.EditCode, AccessibilityLabel: "Inline code"},
	{Label: "H1", Command: core.EditBlock(richtext.Heading1), AccessibilityLabel: "Heading 1"},
	{Label: "H2", Command: core.EditBlock(richtext.Heading2), AccessibilityLabel: "Heading 2"},
	{Label: "¶", Command: core.EditBlock(richtext.Paragraph), AccessibilityLabel: "Paragraph"},
	{Label: "•", Command: core.EditBlock(richtext.Bullet), AccessibilityLabel: "Bulleted list"},
	{Label: "1.", Command: core.EditBlock(richtext.Numbered), AccessibilityLabel: "Numbered list"},
	{Label: "❝", Command: core.EditBlock(richtext.Quote), AccessibilityLabel: "Quote"},
	{Label: "🔗", Command: RichToolLink, AccessibilityLabel: "Add link"},
}
```

<small>[comps/rich_text_editor.go:118](https://github.com/rohanthewiz/grmob/blob/master/comps/rich_text_editor.go#L118)</small>

## Types

### type Calendar

```go
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
	// What it does not change is the announcement: every cell states
	// core.AccessibilitySelected, which reaches the web as aria-selected on a
	// gridcell (ARIA's own date-picker spelling) and both natives as their
	// selected state. The difference is only what activating the chosen day
	// does, which a reader discovers by doing it — the selection moves or it
	// clears.
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
	// for a screen reader; nil gives "Monday, January 2, 2006". Neither today
	// nor the selection is part of the name: both are announced as the states
	// they are (core.CurrentDate and core.AccessibilitySelected); see dayLabel.
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
```

Calendar is a controlled month grid: seven weekday captions over six rows of day cells, with a selected day, an optional "today" ring, optional dots for the days that have something on them, and arrows that ask the caller to change month.

	┌─────────────────────────────────────────┐
	│  ‹        March 2026              ›     │  <- Header (arrows only if OnMonthChange)
	│  Su  Mo  Tu  We  Th  Fr  Sa             │  <- weekday captions
	│  ·1   2   3   4   5   6 ··7             │  <- ·n = one mark, ··n = two
	│   8   9  10  11 [12] 13  14             │  <- [n] = Selected
	│  15  16  17 ·18  19  20  21             │
	│  22  23  24  25  26  27  28             │
	│  29  30  31   1   2   3   4             │  <- adjacent days, dimmed and inert
	│   5   6   7   8   9  10  11             │
	└─────────────────────────────────────────┘

	month := core.NewState(ctx, someDate)
	comps.Calendar{
	    Month:         month.Get(),
	    OnMonthChange: month.Set,
	    Selected:      picked.Get(),
	    OnSelect:      picked.Set,
	    Today:         today,                 // see "The widget never asks what time it is"
	    Marked:        func(d time.Time) int { return len(eventsOn(d)) },
	}

#### Everything is the caller's, including which month is on screen

The widget holds no state and calls no hook, so it may be rendered conditionally — the same contract every widget here but Accordion keeps. That extends to the \*visible month\*, which looks like private view state and is not: an events screen wants to open on the month of the next event, a booking form wants to jump to the month a search result lands in, and a screen with two calendars side by side wants them to move together. A widget that owned its month could serve none of those.

The cost is one piece of state at the call site, as above. DatePicker is the packaging of exactly that state for the form case; reach for this when the grid is part of the screen rather than behind a field.

A nil OnMonthChange draws no arrows. That is the static case — a month with its events dotted, printed into a page — not a broken one.

#### The zero time goes both ways

Selected already spells "nothing is chosen" as the zero time. With Deselectable set, OnSelect reports that same zero when the reader taps the day that is already selected — so a grid used as a \*filter\* is cleared from the grid itself, the value making a round trip through the caller's state with no second callback to wire and nothing beside the calendar to build.

It is off by default, and the default is the interesting half. A picker asking which day the appointment is has no "no day" to offer, and there a stray second tap that quietly emptied the field would lose an answer the reader never asked to lose. So the widget does not guess which of the two it is in: a filter opts in, a picker leaves it alone, and DatePicker forces it off and puts the way out on its own Clear button.

There is deliberately no OnDeselect. Two callbacks setting the same piece of state is two things for every consumer to keep in step, and the day a screen wants to tell them apart it can test for the zero it was handed.

#### The widget never asks what time it is

There is no time.Now() in here, and Today is a field rather than something the widget works out. Three reasons, in ascending order of how much they bite:

  - A render that reads the clock is not a pure function of its inputs, so a snapshot test of "the March 2026 grid" would drift into a different picture every midnight.
  - "Today" is a question about a time zone, and the widget cannot know whether it should answer in the device's zone, the congregation's, or the one the data is stamped in. The caller can.
  - A zero Today draws no ring, which is the honest rendering of a calendar that has not been told what day it is.

#### The month is always six rows

A month spans four to six weeks depending on its length and the weekday it starts on, and a grid that grew and shrank with it would change height as the reader pages through the year — pushing whatever sits below the calendar up and down on every arrow tap. So the grid is always 6×7 = 42 cells, padded at both ends with the adjacent months' days.

The fixed shape pays a second time on the reconciler: changing month patches 42 text contents and their styles and touches no structure at all, where a variable grid would add and remove whole rows.

Those adjacent days are drawn dimmed and are \*inert\*. They are there so the grid reads as a grid — six full rows, no ragged hole — not as targets. A controlled calendar cannot move its own month, so a tap on the trailing "2" under March would either select a date the visible grid no longer highlights or fire two callbacks the caller has to sequence; the arrows are the one deliberate way the month changes.

#### Dates in, dates out: midday, not midnight

Every cell is built at \*\*12:00 in the calendar's location\*\*, and that is the value handed to Marked and OnSelect. It looks like it should be midnight and must not be.

Midnight does not exist on every calendar day. Chile springs forward at 24:00, so 2026-09-06 in America/Santiago begins at 01:00 and Go resolves time.Date(2026, 9, 6, 0, 0, 0, 0, santiago) to \*2026-09-05 23:00\* — the previous day. A grid built at midnight would therefore emit two cells that both read as September 5, and September 6 would be unselectable, in exactly the zones nobody testing in UTC ever looks at. Midday is skipped by no transition in the tz database.

So the value is an instant \*inside\* the intended day rather than at its edge. A caller who wants the day itself takes the triple that a date actually is:

	OnSelect: func(d time.Time) { y, m, day := d.Date(); … }

#### Which location

The calendar works in the location of its anchor — Month if set, else Selected, else Today — and converts Selected, Min, Max and Today into it before reducing them to a calendar day. An event stamped in UTC therefore lands on the day it happened \*locally\*, which is what a reader comparing the grid to their own week expects; a caller who means otherwise passes a Month already in the location they mean.

With all three of Month, Selected and Today zero there is no anchor and no clock to fall back on, and the widget renders nothing.

#### Localization

Go's time package formats in English only, so MonthLabel, WeekdayLabel and DayLabel are the seams for everything a reader sees as a word. The day \*numbers\* are numerals and are not routed through anything.

<small>[comps/calendar.go:143](https://github.com/rohanthewiz/grmob/blob/master/comps/calendar.go#L143)</small>

#### func (Calendar) Render

```go
func (c Calendar) Render(ctx *core.Context) *core.Node
```

<small>[comps/calendar.go:274](https://github.com/rohanthewiz/grmob/blob/master/comps/calendar.go#L274)</small>

### type CodeEditor

```go
type CodeEditor struct {
	// Value is the buffer's text. The editor is controlled: it renders what it
	// is given and reports edits through OnChange.
	Value string

	// OnChange receives every edit. A nil OnChange makes the editor read-only
	// in practice — it will render Value and drop keystrokes — so set ReadOnly
	// instead when that is what you mean, which also tells the platform.
	OnChange func(string)

	// Language picks the lexer by name: "go", "json", or "" for no colour. An
	// unknown name is uncoloured rather than an error, so a screen whose editor
	// mis-spells its language still renders. It also picks the comment marker
	// the comment command toggles; see CommentPrefix.
	Language string

	// Highlighter overrides Language with a lexer of your own — anything
	// satisfying highlight.Highlighter, including one that caches.
	Highlighter highlight.Highlighter

	// Scheme is the palette. The zero value picks highlight.Darcula or
	// highlight.Light from the theme's own background, so an editor in a light
	// app is not a dark rectangle in the middle of the screen and one in a dark
	// app is not a white page.
	Scheme highlight.Scheme

	// LineNumbers turns on the gutter. Off by default: a three-line snippet
	// with line numbers reads as a listing rather than as code.
	LineNumbers bool

	// ReadOnly shows a caret and allows selection while refusing edits. This is
	// the display half of the widget — a code block in a document, a payload in
	// a log viewer — and it is deliberately not Disabled: the reader is meant
	// to select the code and copy it.
	ReadOnly bool

	// TabSize is how many spaces one indent is worth. Zero is core's default of
	// four; use TabSize(0) on the node directly for a literal tab.
	TabSize int

	// CommentPrefix overrides the line-comment marker the toolbar's comment
	// button toggles. Empty takes it from Language.
	CommentPrefix string

	// Height fixes the editor's height, so a long buffer scrolls inside it
	// rather than growing the screen. Empty lets it size to its content.
	Height string

	// Toolbar, when set, adds a row of editing commands above the editor and is
	// the ref they are sent to. Build it with core.UseEditorRef(ctx) in the
	// calling component — see the type's doc for why the ref is yours and not
	// this widget's.
	Toolbar *core.EditorRef

	// ToolbarLabel names the toolbar for a screen reader, which announces it
	// as "<label>, toolbar". Empty is "Editing". Ignored without Toolbar.
	ToolbarLabel string

	// OnSelectionChange reports the caret as byte offsets into Value. Useful
	// for a status line ("line 12, column 4") and for a toolbar that has to
	// know whether anything is selected.
	OnSelectionChange func(start, end int)

	// Style is applied to the editor after the widget's own surface, so the
	// fill, the ink, the radius and the padding are all overridable.
	Style []core.StyleProp
}
```

CodeEditor is a programmer's editor: a monospace buffer with syntax colour, an optional line-number gutter, and an optional toolbar of the editing commands a code surface needs.

	comps.CodeEditor{
	    Value:       src.Get(),
	    OnChange:    src.Set,
	    Language:    "go",
	    LineNumbers: true,
	    Height:      "240px",
	}

	┌ Column ──────────────────────────────────────────────────────┐
	│ ┌ Row (the toolbar, only when Toolbar names a ref) ────────┐ │
	│ │ [ ⇥ ]  [ ⇤ ]  [ // ]                                     │ │
	│ └──────────────────────────────────────────────────────────┘ │
	│ ┌ core.CodeEditor ─────────────────────────────────────────┐ │
	│ │  1  func main() {                                        │ │
	│ │  2      println("hi")                                    │ │
	│ │  3  }                                                    │ │
	│ └──────────────────────────────────────────────────────────┘ │
	└──────────────────────────────────────────────────────────────┘

It is the widget over core.CodeEditor and is what application code should reach for: it runs the highlighter, picks a colour scheme that suits the theme, and builds the toolbar. The node underneath is the primitive, and its doc carries the three rules every host implements.

#### It holds no state and calls no hook

Value is the caller's, exactly as SearchField's is, and every keystroke arrives through OnChange.

The hook question is worth spelling out, because the obvious design fails it. A toolbar needs a core.EditorRef, a ref must be stable across passes, and the way to get one is core.UseEditorRef — a hook. Calling it in here would make every CodeEditor a hook-slot consumer, and therefore something that must be rendered unconditionally on every pass, like Accordion and DatePicker. That is a fine obligation for a date field and a bad one for a \*code block\*: a read-only editor with no toolbar is the thing a document renders inside an \`if\`, inside a loop, inside a lesson body. So the ref is the caller's, and it is the Toolbar field itself — a toolbar is exactly "a ref plus some buttons", and naming the ref is how you ask for one:

	ref := core.UseEditorRef(ctx)
	comps.CodeEditor{Value: v, OnChange: set, Toolbar: ref}

An editor with no toolbar touches no hook and can be rendered anywhere.

#### The highlighter runs every pass, deliberately

No memoization. go/scanner over a thousand lines is well under a millisecond — the tutorial has re-lexed every snippet on every render pass since it had snippets — and the alternative is hooks.UseMemo, which is the hook obligation this widget has just been designed out of. If a buffer ever grows past the point where that is true, the answer is for the \*caller\* to memoize and pass a Highlighter that caches, not for this widget to start consuming slots.

<small>[comps/code_editor.go:65](https://github.com/rohanthewiz/grmob/blob/master/comps/code_editor.go#L65)</small>

#### func (CodeEditor) Render

```go
func (c CodeEditor) Render(ctx *core.Context) *core.Node
```

<small>[comps/code_editor.go:133](https://github.com/rohanthewiz/grmob/blob/master/comps/code_editor.go#L133)</small>

### type DatePicker

```go
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
	// since those four are what the picker is for, and Deselectable is forced
	// off — clearing a field is OnClear's job here, for the reason given
	// where the sheet sets it.
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
```

DatePicker is a date field: a tappable summary of the chosen day that opens a Calendar in a modal sheet and closes again on the tap that picks.

	┌──────────────────────────┐        ┌───────────────────────────┐
	│ Jan 2, 2026          📅  │  tap → │ Event date      Clear  ✕  │
	└──────────────────────────┘        │  ‹    January 2026     ›  │
	                                    │  Su Mo Tu We Th Fr Sa     │
	                                    │   …      [2]     …        │
	                                    └───────────────────────────┘

	comps.FormField{
	    Label: "Event date",
	    Input: comps.DatePicker{
	        Selected: date.Get(),
	        OnSelect: date.Set,
	        Calendar: comps.Calendar{Today: today, Min: today},
	    },
	}

#### It is the input, not the field

There is no Label, Hint, Error or Required here, because FormField already owns all four and any input can sit in its slot. A picker that grew its own label would be a second way to write a form, worded and spaced slightly differently from every other field on the screen.

#### Two states, both of them the widget's

This is the second widget in the package that owns state (Accordion is the other), and it owns exactly the two pieces no application ever wants: is the sheet open, and which month is being browsed inside it. Owning them means inheriting the hook rules — render a DatePicker unconditionally, in a stable position, every pass, the same obligation calling core.NewState directly carries. Calendar itself takes no hooks, so the grid on a screen is still free to be conditional; it is this packaging that is not.

The browsed month is held as a \*zero\* time.Time until an arrow is tapped, rather than being seeded from the selection when the picker mounts. Seeding would go stale the moment the caller set a date from somewhere else — a "next Sunday" shortcut, a form loading a saved draft — and the picker would open on the month the screen first rendered in. A zero Month is exactly what Calendar's own anchor fallback reads as "follow Selected, then Today", so the two states cost one line between them and no re-derivation. Opening the sheet resets it, so the picker always opens on the month it is showing.

#### Picking closes it

A single date has nothing to confirm: the tap that chooses is the tap that finishes, so there is no Done button standing between the two. What the sheet does carry is the ways \*out\* — the backdrop, the ✕, and Clear when the field is clearable — because a reader who opened it to look at March needs to leave without having changed anything.

<small>[comps/date_picker.go:61](https://github.com/rohanthewiz/grmob/blob/master/comps/date_picker.go#L61)</small>

#### func (DatePicker) Render

```go
func (p DatePicker) Render(ctx *core.Context) *core.Node
```

<small>[comps/date_picker.go:139](https://github.com/rohanthewiz/grmob/blob/master/comps/date_picker.go#L139)</small>

### type FormField

```go
type FormField struct {
	Label string

	// Required draws the conventional marker after the label. It is
	// annotation only: this widget validates nothing, and setting it does not
	// make an empty field fail — forms.Required is what does that.
	//
	// Which is why it is worth feeding it from forms.Form.Required rather
	// than from a literal true. A hand-set flag is a second, independent
	// claim about the same field, and the two drift the first time a rule is
	// added or dropped: a marked field that submits empty, or an unmarked one
	// the user cannot get past. The form derives its answer from the rules
	// themselves, so the mark cannot outlive the rule that justified it.
	//
	// Ignored when Label is empty — there is nothing to mark. A control whose
	// label lives elsewhere (the checkbox row in examples/signup, whose title
	// belongs to the ListRow) has to carry its own.
	Required bool

	// Hint is the quiet guidance line under the input. Error replaces it
	// when non-empty — a field shows one line of feedback, and an error
	// outranks guidance.
	Hint  string
	Error string
	Input core.View
	// Style is applied to the wrapping column after the field's own layout.
	Style []core.StyleProp
}
```

FormField wraps any input in the label / input / hint-or-error frame — the pattern element's form\_field made its most-used component, transplanted: the wrapper owns the text furniture so every form in an app annotates its inputs the same way, and the Input slot keeps it agnostic to what is being wrapped (Input, TextArea, NumericInput, a custom picker...).

The widget renders feedback; it does not produce any. Error is a string the caller supplies, and the thing that supplies it — including the decision about \*when\* a message should be visible at all — is package forms:

	comps.FormField{
	    Label: "Email",
	    Hint:  "We never share it",
	    Error: form.Error("email"),
	    Input: form.Input("email", "you@example.com"),
	}

That split is why the dependency runs one way and only in the caller: forms produces the strings and the bound controls, this widget frames them, and neither package imports the other. Required follows the same shape — the form knows which fields reject an empty value, the widget only draws the mark:

	Required: form.Required("email"),

<small>[comps/form_field.go:29](https://github.com/rohanthewiz/grmob/blob/master/comps/form_field.go#L29)</small>

#### func (FormField) Render

```go
func (f FormField) Render(ctx *core.Context) *core.Node
```

<small>[comps/form_field.go:64](https://github.com/rohanthewiz/grmob/blob/master/comps/form_field.go#L64)</small>

### type PINInput

```go
type PINInput struct {
	// Length is the number of cells. Zero means six, the one-time code length.
	Length int

	// Value is the code so far, in full. The field is controlled: it draws
	// exactly this, one character per cell from the left, and OnChange is the
	// only way it changes.
	Value string

	// OnChange receives the whole code after every edit, never a single cell.
	// Without it the field is read-only and reports ConcernPINInputInert.
	OnChange func(string)

	// OnComplete receives the code on every edit that leaves it as long as
	// the field — including an edit to a code that was already complete. Nil
	// is a field the caller reads from Value instead.
	OnComplete func(string)

	// Secure masks the characters, as a device PIN rather than an emailed
	// code. The cells become core.InputPassword.
	Secure bool

	// Label is the accessible name of the group and the stem of each cell's
	// name. Empty means "Code". It draws nothing.
	Label string

	// Style is applied to the row, after the gap and the accessibility pair,
	// so a caller can override any of them — or cap the width, which is the
	// common one: MaxWidth stops four cells from spreading across a tablet.
	Style []core.StyleProp
}
```

PINInput is the boxed one-character-per-cell field a one-time code is typed into: N single-character inputs in a row, with the cursor moving itself.

	comps.PINInput{
	    Length:     6,
	    Value:      code.Get(),
	    OnChange:   code.Set,
	    OnComplete: func(c string) { verify(c) },
	}

	┌───┐ ┌───┐ ┌───┐ ┌───┐ ┌───┐ ┌───┐
	│ 4 │ │ 1 │ │ 7 │ │ 2 │ │   │ │   │
	└───┘ └───┘ └───┘ └───┘ └───┘ └───┘
	                          ▲ the cursor, put there by the cell before it

It is the first widget in the package to drive core's focus system, and that is the whole of what it adds over a Row of fields: a character typed into a cell moves the cursor to the next one, so a six-digit code is six keystrokes rather than six keystrokes and six taps.

#### The value is one string, and therefore a prefix

Value is the whole code, not a cell array: cell i draws the i-th character and empty cells are the ones past the end. A plain string cannot hold a gap, so the cells fill strictly left to right and the two edits follow from that with no cases left over:

	typing    Value = code[:i] + typed + whatever was past the typed run
	clearing  Value = code[:i]        — everything from cell i on is dropped

Clearing is the asymmetric one and it is worth being plain about. A cleared middle cell has to either shift the tail left — so cells the finger never touched change under it — or drop the tail. Dropping is the one a person can predict, because it is what "start again from here" means, and it is what backspacing through an OTP field amounts to on every platform that has one.

The same invariant answers a question the cells can otherwise ask: a character typed into a cell past the end of the code (the web lets a click land anywhere) lands at the end instead, because there is no position for it to occupy.

#### A paste and a second character are the same event

A cell whose OnChange arrives with more than one character is a paste — the whole code dropped into the first box — and it is also what typing into an already-full cell looks like, since the field is controlled and reports its entire contents. Both are handled as one rule: \*\*the incoming string is written from this cell forward, and the cursor lands after the last cell it filled.\*\* A six-character paste into cell 0 fills the field; a "2" typed into a cell already holding "1" arrives as "12", rewrites cell 0 with the character that was already there and puts the new one in cell 1. Characters past the last cell are dropped.

The one case it reads wrongly is a character inserted \*before\* an existing one (the caret parked at the left edge of a full cell), which arrives as "21" and is written in that order. Nothing in the event says where the caret was, so no widget here can tell the two apart.

#### Backspace on an empty cell does nothing, and cannot

There are no key events in this framework — a field reports its text, not the keys that produced it — so a backspace in an \*empty\* cell changes nothing and is therefore never reported. The cursor stays where it is, and clearing a run of cells means one backspace per cell with a tap in between, or one backspace in the leftmost filled cell, which drops everything after it by the rule above. Document it to callers rather than working around it: the workaround is a key channel, and that is a renderer change.

#### OnComplete fires on every change that leaves the code full

Not once per crossing, which is what Countdown.OnDone does and is deliberately not what this does. A caller's OnComplete is "submit the code", and a person who mistypes one digit of a full code, corrects it, and gets silence has a field that will not submit. So a complete code re-reports whenever it changes.

It fires from the change handler rather than from an effect, so it never fires for a Value that merely arrived complete — a screen restored with a code already in it does not resubmit itself on mount.

A change that produces the value already held is treated as an echo: no OnChange, no cursor move, no OnComplete. Both natives can report their own text back after a Go-side update, and none of the three is worth doing twice.

#### It holds hooks, so it is not conditional-safe

One FocusRef per cell, and refs must be stable across passes or a focus command aims at last pass's identity. So this is a hook caller with Accordion's rule: render it in a stable position every pass rather than inside a core.If.

The hook count does not follow Length. It follows the largest Length this widget has ever been rendered with, held in one slot of its own, because a Length that shrank between passes would otherwise retire hook slots from the middle of the sequence and drift every cursor after them. Growing is safe — new slots are appended past the ones already bound — and never shrinking is what makes it so. The cost is a handful of FocusRefs that nothing points at, which cost a slice entry each and are never stamped onto a node.

#### What the cells are, and what they are not

Each cell is an ordinary core.Input (core.InputPassword when Secure), so it wears the theme's field frame and matches the text inputs above it in a form. They divide the row equally — core.FlexGrow with a zero core.FlexBasis, the pair Calendar's day cells use, which is what makes the four targets agree on "equal shares" rather than "equal shares of the leftovers". The row therefore fills the width it is given; cap it with Style.

They take the platform's text keyboard, not its number pad. The keyboard type is chosen by node type on both natives — "NumericInput" is the numeric one — and that node carries an int value, which cannot express an empty cell: clearing one would report nothing at all, so backspace would stop working entirely. A digits-only keyboard needs a keyboard-type prop on core.Input, which is a renderer change and not this widget's to make.

#### Accessibility

The row is a core.RoleGroup named by Label, and each cell is named "\<Label>, N of M" so a reader moving between them says which box it is in. Label is the accessible name only — there is no visible caption, as with InputRow; wrap this in a FormField when one is wanted.

#### Theme roles read

	Cells   Components.Input — the same frame every other field in the form has
	Gap     Spacing.SM between cells

<small>[comps/pin_input.go:159](https://github.com/rohanthewiz/grmob/blob/master/comps/pin_input.go#L159)</small>

#### func (PINInput) Render

```go
func (p PINInput) Render(ctx *core.Context) *core.Node
```

Render allocates the refs, declares their order and draws the cells.

<small>[comps/pin_input.go:192](https://github.com/rohanthewiz/grmob/blob/master/comps/pin_input.go#L192)</small>

### type RadioGroup

```go
type RadioGroup struct {
	// Options are drawn top to bottom.
	Options []RadioOption

	// Value is the selected option's Value. A Value matching no option draws
	// every ring empty.
	Value string

	// OnChange receives the tapped option's Value, only when it differs from
	// Value. Nil draws a display-only group with no handlers.
	OnChange func(string)

	// Label is the group's accessible name. A radio group with no name is
	// announced as a bare run of radio buttons, so set it unless a visible
	// heading directly above names the group.
	Label string

	// Disabled disables every option.
	Disabled bool

	// Style is applied to the group column after the widget's own props.
	Style []core.StyleProp
}
```

RadioGroup is a vertical set of mutually exclusive options, each a row with a ring on the leading edge: shipping speed, a plan, a theme.

	comps.RadioGroup{
	    Label: "Shipping",
	    Options: []comps.RadioOption{
	        {Value: "std", Label: "Standard", Subtitle: "3–5 days"},
	        {Value: "exp", Label: "Express", Subtitle: "Next day"},
	    },
	    Value:    ship.Get(),
	    OnChange: ship.Set,
	}

#### Where it sits among the choice widgets

	core.Select         compact; the options are hidden until opened
	SegmentedControl    horizontal, short labels, two to four options
	RadioGroup          vertical, every option visible, room for a subtitle

#### Radio roles, and what they changed

The group is core.RoleRadioGroup and each row core.RoleRadio, with the choice stated through AccessibilitySelected — which the web exporters write as aria-checked on a radio. A reader says "radio button, checked", 2 of 3.

It shipped first as RoleListBox and RoleOption through ListRow.Selectable, because core had no radio pair. That announced "option, selected" and gave the browser's listbox keyboard, where the arrows move a highlight and leave the choice alone. The radio pair is what ARIA calls this control, and its keyboard is the one a radio group should have: one tab stop on the checked radio, and the arrows move the check, so OnChange fires as a user arrows through the options. OnChange is a setter and fires only on a change, so that costs a caller nothing.

The rows are ListRows with the role and state passed through Style rather than through Selectable, which would make them options. Selected stays false, so ListRow adds neither its ", selected" name suffix nor its tint.

On the natives, Compose names both ends (selectableGroup() and Role.RadioButton) and SwiftUI names neither; the checked row is announced through .isSelected there, as the listbox rows were.

#### A closed composite: do not put it inside another

RadioGroup declares its own radiogroup, which makes it one of the widgets in this package that declare a keyboard container role (BottomBar's toolbar is another). It holds no core.View, so nothing can be nested inside it. Placing it inside a container you roled as a listbox, radiogroup, tablist or toolbar yourself is the one way to nest it, and core.AuditTree reports that as a nested composite in debug mode. comps/nested\_composite\_test.go holds the "closed" half of this.

#### The whole row is the target, and there is no second control

Unlike SwitchRow, the ring is drawn, not a platform control: a Column with a border and, when selected, a filled dot, hidden from assistive technology. The row's OnTap is therefore the only handler, and the web's double dispatch that SwitchRow guards against cannot happen here. Tapping the selected option does nothing; OnChange fires only for a change, the rule Stepper and Rating follow.

No row takes a background tint. ListRow's Surface tint is its selection cue for a Selected row; these rows are never Selected (see above), and the ring already shows the choice.

#### Theme roles read

	Ring, selected   Colors.PrimaryOnLightColor() — border and dot
	Ring, other      Colors.ControlBorder (Colors.BorderColor() if unset):
	                 the ring is a control boundary, so it takes the stroke
	                 role that clears 3:1
	Ring, disabled   Colors.TextSecondary
	Row text         everything ListRow reads

<small>[comps/radio_group.go:78](https://github.com/rohanthewiz/grmob/blob/master/comps/radio_group.go#L78)</small>

#### func (RadioGroup) Render

```go
func (g RadioGroup) Render(ctx *core.Context) *core.Node
```

Render builds Column(radiogroup) > ListRow(radio)... as described in the type doc.

<small>[comps/radio_group.go:125](https://github.com/rohanthewiz/grmob/blob/master/comps/radio_group.go#L125)</small>

### type RadioOption

```go
type RadioOption struct {
	// Value identifies the option to OnChange and is compared with
	// RadioGroup.Value.
	Value string

	// Label is the row's title and its accessible name.
	Label string

	// Subtitle is the quieter second line and the row's accessibility hint.
	Subtitle string

	// Disabled greys this option and drops its taps.
	Disabled bool
}
```

RadioOption is one choice in a RadioGroup.

<small>[comps/radio_group.go:103](https://github.com/rohanthewiz/grmob/blob/master/comps/radio_group.go#L103)</small>

### type RichTextEditor

```go
type RichTextEditor struct {
	// Doc is the document. The editor is controlled: it renders what it is
	// given and reports edits through OnChange.
	Doc richtext.Doc

	// OnChange receives every edit as a whole document. A nil OnChange makes
	// the editor read-only in practice; set ReadOnly instead when that is what
	// you mean, which also tells the platform.
	OnChange func(richtext.Doc)

	// Placeholder is the prompt an empty editor shows.
	Placeholder string

	// ReadOnly shows the document with a caret and a selection and refuses
	// edits. Deliberately not Disabled: a note being displayed is content the
	// reader is meant to select and copy.
	ReadOnly bool

	// Toolbar, when set, adds the command row above the editor. Build it with
	// comps.UseRichToolbar(ctx) in the calling component.
	Toolbar *RichToolbar

	// MinHeight gives an empty editor something to be. Without it a document
	// with one line in it is one line tall, which reads as a text field rather
	// than as a place to write.
	//
	// It is core.MinHeight underneath, which every target honours in points
	// (Android also takes a percentage; see the platform table in
	// docs/concepts/styling-and-theming.md).
	MinHeight string

	// Height fixes the editor's height instead, so a long document scrolls
	// inside it rather than growing the screen.
	Height string

	// Style is applied to the editor after the widget's own frame.
	Style []core.StyleProp
}
```

RichTextEditor is a formatted-text editor: bold, italics, headings, lists, quotes, links — with a toolbar whose buttons show what is active under the caret.

	bar := comps.UseRichToolbar(ctx)

	comps.RichTextEditor{
	    Doc:         note.Get(),
	    OnChange:    note.Set,
	    Placeholder: "Write something…",
	    Toolbar:     bar,
	    MinHeight:   "160px",
	}

	┌ Column ──────────────────────────────────────────────────────┐
	│ ┌ Row, wrapping (the toolbar) ─────────────────────────────┐ │
	│ │ [B] [I] [U] [S] [</>] [H1] [H2] [•] [1.] [""] [🔗]       │ │
	│ └──────────────────────────────────────────────────────────┘ │
	│ ┌ core.RichTextEditor ─────────────────────────────────────┐ │
	│ │  The document, edited in place by the platform's own     │ │
	│ │  text engine.                                            │ │
	│ └──────────────────────────────────────────────────────────┘ │
	└──────────────────────────────────────────────────────────────┘

#### A read-only editor with no toolbar is the display half

A comment, a note, a description — anything that shows formatted text the reader did not write. There is no second "RichTextView" node, because there does not need to be one: the renderer's own text engine draws the document either way, and \`ReadOnly\` is the difference between reading it and writing it. That is also why this widget takes no hook of its own; see below.

#### The toolbar is the caller's, and that is what keeps the display half free

A toolbar needs three things that must survive a render pass: a core.EditorRef to send commands to, the last reported selection (so the bold button can look pressed), and whether the link prompt is open. All three are hooks, and a widget that called them would make \*every\* RichTextEditor a hook-slot consumer — something that must be rendered unconditionally on every pass, like Accordion and DatePicker.

That is a fine obligation for an editor with a toolbar and a bad one for a note being \*displayed\*, which is the thing rendered inside an \`if\`, inside a loop, inside a list of comments. So UseRichToolbar is the hook, the caller makes it, and an editor with no toolbar touches nothing. comps.CodeEditor makes the same split for the same reason.

<small>[comps/rich_text_editor.go:54](https://github.com/rohanthewiz/grmob/blob/master/comps/rich_text_editor.go#L54)</small>

#### func (RichTextEditor) Render

```go
func (r RichTextEditor) Render(ctx *core.Context) *core.Node
```

<small>[comps/rich_text_editor.go:189](https://github.com/rohanthewiz/grmob/blob/master/comps/rich_text_editor.go#L189)</small>

### type RichToolItem

```go
type RichToolItem struct {
	Label   string
	Command string
	// AccessibilityLabel names the button for a screen reader. A toolbar is
	// glyphs — "B", "H1", "🔗" — and a glyph is not a name.
	AccessibilityLabel string
}
```

RichToolItem is one button on the toolbar: what it says, and what it sends.

Command is a core Edit\* constant or one of the two builders (core.EditBlock, core.EditLink) — except for RichToolLink, which is this package's own sentinel for "open the link prompt", because a URL has to be typed before there is a command to send.

<small>[comps/rich_text_editor.go:99](https://github.com/rohanthewiz/grmob/blob/master/comps/rich_text_editor.go#L99)</small>

### type RichToolbar

```go
type RichToolbar struct {
	// Items is what the toolbar offers, in order. Defaults to a copy of
	// RichToolbarDefault; assign to it to offer something else.
	Items []RichToolItem

	// Label names the strip for a screen reader. Empty uses "Formatting".
	// Assign to it like Items: a screen with two editors wants two names, and
	// an app in another language wants its own word.
	Label string
	// contains filtered or unexported fields
}
```

RichToolbar is everything a toolbar needs that has to survive a render pass.

Built by UseRichToolbar, which is a hook: the ref must be the same pointer every pass or the buttons would command an editor nobody is listening to, and the selection has to be remembered between the report arriving and the next render drawing the buttons from it.

<small>[comps/rich_text_editor.go:139](https://github.com/rohanthewiz/grmob/blob/master/comps/rich_text_editor.go#L139)</small>

#### func UseRichToolbar

```go
func UseRichToolbar(ctx *core.Context) *RichToolbar
```

UseRichToolbar builds the state a RichTextEditor's toolbar needs.

Four hook slots, in a fixed order, so — like any hook user — it must be called unconditionally on every pass of the component that owns it.

	func NoteScreen(ctx *core.Context) core.View {
	    note := core.NewState(ctx, richtext.Doc{})
	    bar  := comps.UseRichToolbar(ctx)
	    return comps.RichTextEditor{Doc: note.Get(), OnChange: note.Set, Toolbar: bar}
	}

<small>[comps/rich_text_editor.go:165](https://github.com/rohanthewiz/grmob/blob/master/comps/rich_text_editor.go#L165)</small>

#### func (*RichToolbar) Selection

```go
func (b *RichToolbar) Selection() core.RichSelection
```

Selection is the last selection the editor reported, which is what the toolbar draws its pressed state from — and is worth reading directly for a status line, or to decide whether a "Link" action makes sense.

<small>[comps/rich_text_editor.go:187](https://github.com/rohanthewiz/grmob/blob/master/comps/rich_text_editor.go#L187)</small>

### type SearchField

```go
type SearchField struct {
	// Value is the current text. The field is controlled: it renders what it
	// is given and reports edits through OnChange.
	Value string

	// Placeholder is the empty-field prompt. Empty is "Search".
	Placeholder string

	// OnChange receives every edit, including the clear. A nil OnChange makes
	// the field read-only in practice — it will render Value and drop
	// keystrokes — so it is worth setting even on a field you expect not to
	// change.
	OnChange func(string)

	// OnSubmit fires on the keyboard's return / IME done action. When nil the
	// field carries no submit callback at all, which keeps a
	// search-as-you-type box off the InputWithSubmit path entirely.
	OnSubmit func()

	// OnClear replaces what the clear button does. The default is
	// OnChange(""), which is what a clear means for a controlled field; set
	// this when clearing also has to dismiss results, cancel a pending
	// request, or restore a previous view.
	OnClear func()

	// Glyph is the leading mark. Empty is 🔍; NoGlyph drops it, for a field
	// whose surroundings already say what it searches.
	Glyph   string
	NoGlyph bool

	// AccessibilityLabel names the field for screen readers. Empty falls back
	// to the placeholder, which is the resolved one — so the default field
	// announces as "Search" rather than as nothing.
	//
	// The fallback exists because a placeholder is not a label on any
	// platform: it is a value hint that vanishes on the first keystroke, so a
	// field relying on it alone is unnamed for exactly the users who most
	// need the name.
	AccessibilityLabel string

	// FocusRef names the input, so core.Focus can put the cursor in it and
	// core.UseFocusOrder can place it in a form's return-key order. It goes
	// on the input rather than the row because the row is not focusable on
	// any target. Nil leaves the field unnamed.
	FocusRef *core.FocusRef

	// Style is applied to the row after the widget's own frame, so the fill,
	// the radius and the padding are all overridable.
	Style []core.StyleProp

	// InputStyle is applied to the input itself, after its own flattening,
	// where Style reaches only the row around it. It exists for semantics
	// that belong to the element with focus: comps.SearchableSelect puts
	// core.RoleComboBox, the expanded state and aria-controls here, because
	// ARIA 1.2 wants them on the field the caret is in rather than on its
	// frame. Visual overrides still belong in Style.
	InputStyle []core.StyleProp
}
```

SearchField is a text field dressed as a search box: a leading magnifier, a flexible input, and a clear button that appears once there is something to clear.

	comps.SearchField{
	    Value:    query.Get(),
	    OnChange: query.Set,
	    OnSubmit: run,
	}

	┌ Row (Surface, rounded) ──────────────────────────────────────┐
	│ 🔍   ┌ Input FlexGrow(1) ─────────────┐   [✕]                │
	│      └────────────────────────────────┘                      │
	└──────────────────────────────────────────────────────────────┘

#### It holds no state and calls no hook

Value is the caller's, exactly as with Chip's Selected and DataTable's Sort, and every keystroke arrives through OnChange for the caller to store. That is worth stating because the obvious alternative — a field that owns its own text — would make this the second widget in the package with hook obligations, and a search box is a thing screens render conditionally (in a header that appears when a "Search" action is tapped), which is precisely what a hook-slot consumer must not be.

#### Debouncing is deliberately not in here

A controlled field cannot debounce its own OnChange: the value has to reach state on the keystroke or the characters do not appear. What wants delaying is the \*reaction\* — the query, the filter, the fetch — and that lives in the caller. hooks.UseDebounce is the piece for it:

	d := hooks.UseDebounce(ctx, 250*time.Millisecond)
	comps.SearchField{
	    Value: query.Get(),
	    OnChange: func(s string) {
	        query.Set(s)                       // now, so typing looks like typing
	        d.Call(func() { search(s) })       // in 250ms, if the typing stopped
	    },
	    OnSubmit: func() { d.Cancel(); search(query.Get()) },
	}

Splitting it that way is also what makes the two paths honest: Enter should search immediately, and Cancel is how the pending call gets out of the way.

#### The frame

The row paints the theme's Surface at the theme's own field radius and the input inside it is flattened — transparent, no radius, no padding and no border of its own. Without that the theme's Input base (its own background, corners and rule) would draw a second box inside the first, which is what a hand-rolled search row looks like before someone notices.

The border half of that is newer than the rest and the reason is worth keeping: the theme's Input entry used to carry no rule at all, so on the web a search field was quietly wearing the \*browser's\* — which the row's own fill mostly hid, and which no core.BorderWidth(0) could have removed anyway (see borderResetTypes in htmlout/tag.go). Now that both bundled themes state a field frame, the second box would be drawn deliberately and on all four targets, so the flattening has to say so.

<small>[comps/search_field.go:65](https://github.com/rohanthewiz/grmob/blob/master/comps/search_field.go#L65)</small>

#### func (SearchField) Render

```go
func (s SearchField) Render(ctx *core.Context) *core.Node
```

<small>[comps/search_field.go:124](https://github.com/rohanthewiz/grmob/blob/master/comps/search_field.go#L124)</small>

### type SearchableSelect

```go
type SearchableSelect struct {
	// Options are the choices, in the order matches are listed.
	Options []core.SelectOption

	// Value is the chosen option's Value; empty means nothing is chosen.
	Value string

	// OnChange receives the Value of a picked option, and "" when the clear
	// button empties the field. It is not called when the user picks the
	// option already chosen.
	OnChange func(string)

	// Query is the field's text. It is the caller's, as SearchField.Value is.
	Query string

	// OnQueryChange receives every edit, the chosen label after a pick, and
	// "" on clear. Nil leaves a field that drops keystrokes.
	OnQueryChange func(string)

	// Label names the field and, with " suggestions", the list. Empty falls
	// back to the placeholder, as SearchField does.
	Label string

	// Placeholder is the empty field's prompt. Empty is "Search".
	Placeholder string

	// MaxResults caps the rows shown; the status line says how many more
	// matched. Zero is 6, which fits under a field above a phone keyboard.
	MaxResults int

	// Filter decides whether an option matches the query. The query is
	// trimmed first, and an option is only offered when Filter is true. Nil
	// matches a case-insensitive substring of Label.
	Filter func(opt core.SelectOption, query string) bool

	// Count writes the status line from the rows shown and the total that
	// matched. Nil writes "No matches", "1 match", "3 matches" or
	// "6 of 14 matches". Supply it to say it in another language.
	Count func(shown, total int) string

	// FocusRef names the field, so the field can join a core.UseFocusOrder
	// and be the target of core.Focus. See "Focus and the keyboard".
	FocusRef *core.FocusRef

	// ID names the list, so the field's aria-controls can point at it, and
	// prefixes each row's id ("<ID>-option-0", …) for aria-activedescendant.
	// Empty derives "searchable-select-" plus Label's letters and digits,
	// lowercased and dash-joined, so two selects on one screen with the same
	// Label need an ID each; core's audit reports the duplicate if they have
	// none. The ids are read by the web targets alone.
	ID string

	// Style is applied to the outer column after the widget's own props.
	Style []core.StyleProp
}
```

SearchableSelect is a choice from a list too long to scroll: a search field whose typing filters the options into a short list under it, where a tap picks one.

	comps.SearchableSelect{
	    Label:         "Country",
	    Options:       countries,          // []core.SelectOption
	    Value:         country.Get(),
	    OnChange:      country.Set,
	    Query:         query.Get(),
	    OnQueryChange: query.Set,
	}

	┌ Column ────────────────────────────────────────────┐
	│ ┌ SearchField (RoleSearch) ──────────────────────┐ │
	│ │ 🔍  an                                     ✕   │ │  the input is the
	│ └────────────────────────────────────────────────┘ │  RoleComboBox
	│         │ aria-controls, while rows show           │
	│ ┌ Column▼RoleListBox "Country suggestions" ──────┐ │  only while the
	│ │ Argentina        South America     (option)    │ │  query matches and
	│ │ Canada           North America     (option)    │ │  is not the chosen
	│ └────────────────────────────────────────────────┘ │  label; id ID, rows
	│ Text RoleStatus  "2 of 6 matches"                  │  ID-option-N
	└────────────────────────────────────────────────────┘  status hidden while shut

#### When the list shows

While Query is not empty and is not exactly the chosen option's label. Picking an option sets the query to its label, so the list closes, and the field now reads as the choice. Editing that text opens the list again. The widget keeps no open flag: both halves of the condition are the caller's state already, which keeps the widget free of hooks and so safe to render conditionally, as SearchField is.

Nothing shows for an empty query. Listing every option on focus would need a focus flag, held either in a hook or in more caller state, and a list short enough to show in full is one a Select or a RadioGroup already serves better.

#### Focus and the keyboard, which are the design

The shape is a field and a list. What had to be decided is how the two share the keyboard. There are five parts:

 1. The list never takes focus. It appears under a field the user is typing in, and nothing issues a focus command, so typing carries on.
 2. The return key belongs to the form, not to the list. The field has no OnSubmit, so with FocusRef in a core.UseFocusOrder the keyboard shows Next and moves on to the following field. The list is not in the order: core.FocusNext walks declared refs only, and no option is one. "Enter picks the top match" was the alternative. It was rejected because an explicit submit suppresses the Next action (see stampTraversal in core/focus\_order.go), and a field in the middle of a form cannot do both with one action key. On the web ARIA adds one exception: once the arrows have reached an option, Enter picks that option and does not also run Next. With no option reached, Enter is the form's again, so nothing is picked that the user did not arrow to.
 3. On the web, the field is an ARIA combobox and focus never leaves it. ArrowDown and ArrowUp move an active option, which the WASM runtime names through aria-activedescendant and outlines, and Enter picks it. Tab goes on to the next control, past the list: the listbox is the combobox's popup and holds no tab stop. The highlight does not pick, because an arrow key that changed the value would close the list under the user.
 4. Picking dismisses the keyboard. On a phone the choice is made, so the keyboard is in the way of the form. core.DismissKeyboard blurs only a field that has focus, so a web user who picked with the arrow keys keeps focus wherever it was.
 5. On the web a keyboard pick leaves focus in the field. Part 4's dismiss would blur it, and the widget cannot tell a key from a tap, so the runtime tells them apart instead: a pick made with Enter declines the one blur that follows it, and a tap is dismissed as on a phone. Before the combobox pattern the list itself held focus, so a pick removed the focused option with the list and dropped focus onto the page, and returning it with core.Focus would have raised a phone's keyboard again.

#### It is an ARIA combobox

ARIA's pattern for a field that filters a list under it is role="combobox" on the field, with aria-expanded saying whether the list shows, aria-controls naming it, and aria-activedescendant naming the option the arrows reached. The widget states the first two and the runtime writes the third per keystroke (core.RoleComboBox says why that one is behaviour):

	the field (SearchField's input)   RoleComboBox, ExpandedWhen(rows show),
	                                  and AccessibilityControls(ID) while
	                                  they do
	the list                          RoleListBox and AccessibilityID(ID)
	each row                          RoleOption and
	                                  AccessibilityID(ID-option-N)

"Expanded" means the listbox is in the tree, not merely that the query is open: a query with no matches renders no listbox (see Render), and a field saying expanded, with an aria-controls naming nothing, would be the dangling reference core's audit reports.

The rows' ids name slots rather than options, so an option's Value never has to be made into an id. That is safe because the runtime drops the active option on every edit of the field, and editing is what re-fills the slots.

The role goes on the input and not on SearchField's row, because ARIA 1.2 wants the state on the element that has focus; SearchField.InputStyle is the door. The search landmark stays on the row around it.

A polite status line still says how many options match. A combobox is announced as expanded, with no count, and the list appears silently under the field, so the count remains the part a screen reader user most needs.

The status line is hidden while the list is shut, and a live region that becomes visible is not announced reliably. Later keystrokes change its text while it is showing, and those are announced.

#### Options

Options are core.SelectOption, the type core.Select takes, so a list can move from one to the other unchanged. Group becomes the row's subtitle rather than a heading: a listbox owns its options, and a heading among them would be a foreign child. Disabled and GroupDisabled rows are shown and announced, and a tap on one does nothing.

#### Theme roles read

	Field       as SearchField (Surface, Components.Input.BorderRadius)
	List frame  Colors.Background, Colors.Border, the input radius
	Rows        as ListRow, with SelectedStyle for the current value
	Status      Typography.Caption, Colors.TextSecondary

<small>[comps/searchable_select.go:137](https://github.com/rohanthewiz/grmob/blob/master/comps/searchable_select.go#L137)</small>

#### func (SearchableSelect) Render

```go
func (s SearchableSelect) Render(ctx *core.Context) *core.Node
```

Render builds Column(SearchField, listbox?, status) as drawn in the type doc.

<small>[comps/searchable_select.go:200](https://github.com/rohanthewiz/grmob/blob/master/comps/searchable_select.go#L200)</small>

