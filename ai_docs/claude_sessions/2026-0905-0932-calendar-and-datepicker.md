# Session: Calendar and DatePicker (plan A3)

Session: https://claude.ai/code/session_01AM7rDTyaXrFQ4X6rPDdUWp
Date: 2026-09-05 (follows "b1-b3-adopted-downstream")

## Ask

"Let's do plan A3: Calendar / DatePicker" — item 1 of the previous session's
Next list, and step 4 of `ai_docs/plans/components-datatable-compass-map.md`.

## What landed

    components/calendar.go          Calendar + the date math it needs
    components/calendar_test.go     22 tests, including the DST regression
    components/date_picker.go       DatePicker
    components/date_picker_test.go  12 tests
    components/doc.go               the hook note now has two widgets in it
    examples/tutorial/chapter4.go   lesson 4.9
    examples/tutorial/chapter4_test.go  three demo-liveness tests
    examples/tutorial/app_test.go   nodeStyle grew three fields
    ROADMAP.md, the plan doc

Pure Go, no renderer work, no new node type — the whole point of Tier A.
`go build`, `go vet`, `gofmt`, `go test ./...` and `-race` all clean.

## The four decisions that carried the design

The plan's sketch was one paragraph and it survived intact. What it did not
say is where the work was.

### 1. Midday, not midnight

This is the one that would have shipped as a bug. Every cell is built at
**12:00 in the calendar's location**, and midday is the value handed to
`Marked` and `OnSelect`.

Midnight is a local time that does not exist on every calendar day. Chile
springs forward at 24:00, so 2026-09-06 in `America/Santiago` begins at 01:00
and Go resolves `time.Date(2026, 9, 6, 0,0,0,0, santiago)` to **2026-09-05
23:00** — the previous day. A grid built at midnight therefore emits two cells
that both read as the 5th, and the 6th is unselectable, in exactly the zones
nobody testing in UTC will ever look at.

Verified empirically before writing a line of the widget, and pinned by
`TestCalendarSurvivesADaySkippedMidnight`, which skips itself if a future
tzdata moves the transition rather than quietly passing on a fixture that no
longer tests anything.

Midday is skipped by no transition in the tz database. The cost is that the
value is an instant *inside* the day rather than at its edge, which the doc
says loudly, with the `y, m, d := d.Date()` line a caller actually wants.

### 2. `Today` is a field, and there is no `time.Now()` anywhere

`grep -rn "time.Now()" components core hooks` returned nothing before this
session and returns nothing after it. Calendar was the first widget with a
reason to want one, and three arguments said no: a render that reads the clock
is not a function of its inputs (the snapshot drifts every midnight); "today"
is a time-zone question the widget cannot answer and the caller can; and a
zero `Today` drawing no ring is the honest picture of a calendar nobody has
told what day it is.

The tutorial demo pins its own `tutorialToday` for the same reason, which
makes the lesson demonstrate the argument rather than just state it.

### 3. Six rows always, and the padding days are inert

A grid that sized itself to its month would change height between February and
August and shove the screen below it about on every arrow tap. Fixed 6×7 also
means a month change patches 42 numbers and touches no structure — which is
why the cells are deliberately **not** `Keyed`: they never reorder, so
positional matching is exactly right, and keying by date would turn every
patch into a wholesale replacement.

The same fixed shape is why the `Marked` dot is always in the tree and merely
goes transparent: the day numbers keep one baseline whether or not their day
has something on it, and toggling a mark is a color patch rather than a child
insertion in the middle of 42 cells.

Adjacent days are drawn dimmed and **inert**. A controlled calendar cannot
move its own month, so a tap on the trailing "2" under March would either
select a day the grid no longer highlights or fire two callbacks in an order
the caller has to guess. They are there so six rows read as six rows.

### 4. `DatePicker` owns two states and nothing else

It is the package's second stateful widget after Accordion, and it owns
exactly the two pieces no application ever wants: is the sheet open, which
month is being browsed. `components/doc.go`'s hook section now names both and,
more usefully, names the counter-example — Calendar's visible month *looks*
like private view state and is not, because a screen opening on the month of
its next event has to be able to say so.

Two shapes it took from existing package precedent rather than inventing:

- **`Calendar` is a template field**, the `SegmentedControl.Segment` move.
  `Today`, `Min`, `Max`, `Marked`, `WeekStart`, the three label functions,
  `Header` and `Style` all pass through; `Month`, `OnMonthChange`, `Selected`
  and `OnSelect` are what the picker is computing.
- **No `Label`, `Hint`, `Error` or `Required`** — `FormField` owns all four and
  any input drops into its slot. A picker growing its own label would be a
  second way to write a form, worded and spaced slightly differently from
  every other field on the screen.

The browsed month is held as a **zero `time.Time`** rather than seeded from the
selection. Zero is exactly what Calendar's anchor fallback reads as "follow
Selected, then Today", so the two states cost one line between them, and a
selection set from elsewhere between two openings is followed instead of going
stale. Opening resets it.

Picking closes the sheet: a single date has nothing to confirm, so there is no
Done button between the tap that chooses and the tap that finishes. What the
sheet carries instead are the ways *out* — the backdrop, the ✕, and Clear when
`OnClear` is set, which is nil by default because whether a date is optional is
the form's question and a widget that always offered to empty a required field
would be offering an invalid state.

## Smaller things worth remembering

- **Localization is three nil-able functions**, not a locale. Go's `time`
  formats in English only, so `MonthLabel`, `WeekdayLabel` and `DayLabel` are
  the seams; the day numerals are routed through nothing. `DayLabel` names the
  *spoken* cell, and the state suffixes (", today", ", selected") are appended
  to whatever it returns, so a translated calendar still announces its
  selection.
- **`ymd`** collapses an instant to `20260312` for its calendar day in its own
  location. Ordered comparison of that *is* ordered comparison of dates, which
  is what `Min`/`Max` need and what `time.Before` cannot give without first
  normalizing both sides to the same hour.
- **`time.Date` normalization does the grid.** Day `1-lead+i` for i in 0..41,
  handed to `time.Date` once, yields the whole 42-cell window — day 0 is the
  last of the previous month, day 32 the 1st of the next — with no day-by-day
  addition to drift across a transition.
- **Selected ink is contrast-picked**, not white: Primary is a light blue under
  one bundled theme and a dark indigo under another. The `Marked` dot on a
  selected cell takes that same ink, because a Primary dot on a Primary fill is
  not there.
- **Disabled cells register a no-op handler** rather than dropping one — the
  pairing `components.Button` settled on. Worth noting that both DOM renderers
  already honour `Style.Disabled` on a plain container (`pointer-events:none` +
  `aria-disabled`), so the flag really is inert on all four targets.
- **The selection is a rounded rectangle, not a circle.** `core.Style` has no
  aspect-ratio and the cell's width belongs to its container, so a circle would
  need a hard-coded size that stops matching the moment the calendar is not the
  width it was designed at.

## Found, not fixed

**A `<button>` keeps the user agent's default border on both DOM targets.**
`htmlout/export.go` and `wasm/grmob-runtime.js` both emit a `border`
declaration only when `BorderWidth > 0 && BorderColor != ""`, and otherwise
emit nothing — so the browser's own button border stands. Compose and SwiftUI
draw none. `components.Button`'s `EmphasisGhost` is documented as
"EmphasisOutlined without the rule", and on the web it has a rule.

There is no style-level workaround: `core.BorderWidth(0)` emits nothing, which
is the same as saying nothing. So the calendar's month arrows and the picker
sheet's Clear/✕ are boxed on the web and bare on a phone. It is a two-line
renderer fix with wide golden churn behind it — every button on both web
targets changes — which is why it is recorded rather than done here.

Screenshotted both widgets through `htmlout` to confirm the rest: the grid, the
ring, the fill, the dots, the dimmed run, the disabled arrow at the end of a
range, and the open sheet all render as designed.

## Next

1. Adopt Calendar downstream in `../church/church_mobile` — the events screen
   is what asked for this (`app/events.go` groups by month and has an
   `EventDate` per row, so `Marked` is a lookup it already has).
2. `components.Chip`'s selected look — Surface fill with Primary ink is the
   *less* prominent treatment on the *selected* chip. Fourth session running.
3. A `Role`/`AccessibilityRole` prop in core, serving `DataTable`'s `<table>`
   semantics, the screen-furniture bundle, and now the calendar's day cells —
   which are tappable Boxes with names but no button role.
4. The `<button>` border divergence above.
5. Small: say "emit `OnEndReached` before the children" in its doc, for
   hand-written `core.List` callers (carried over).
6. Small: add the mid-list busy case to `EmptyState`'s doc (carried over).
7. Then Tier C: heading plumbing + `Rotate` + Compass.
