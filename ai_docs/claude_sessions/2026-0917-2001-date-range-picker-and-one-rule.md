# D6: DateRangePicker, and a decision that turned out to be a consequence

**Session:** adbf17b6-e412-487e-a6c7-5562ac1df3e1
**Date:** 2026-09-17 20:01 (follows "pin-input-and-a-code-with-no-gaps")
**Branch:** master (7ecb271 → this commit)

## 1. The ask

"start D6" — `ai_docs/plans/comps-low-hanging-fruit-2.md`, sixth in the plan's
order and the second of the three it said could be taken in any order.

## 2. Two halves

`Calendar` gained `RangeStart` and `RangeEnd time.Time` and draws the fill
between them. `comps.DateRangePicker` is `DatePicker`'s trigger, sheet, two
ways out and Calendar-as-template, with the two-tap protocol on top.

The fields on `Calendar` are **display, not input**. The grid still reports one
tapped day at a time, as it always has; whoever holds the pair decides what a
tap means to it. A screen showing a booking's nights sets the pair and leaves
`OnSelect` nil, and the lesson's demo puts exactly that under the picker so the
split is visible rather than asserted.

## 3. The plan's decision was a consequence, not a rule

The sketch carried one decision already — "the third tap, with a range already
set, starts a new range at that day rather than moving the nearer end." The
build did not implement it. It implemented something smaller and got it for
free.

The widget owns one piece of state, a **pending start**, and the whole protocol
is one line over it:

```
no pending start    this tap becomes the pending start
a pending start     this tap is the other end; the range is reported
```

Everything a range picker is usually specified with falls out of those two
lines with no case of its own:

- **The third tap starts a new range** — completing one clears the pending
  start, so the next tap finds none. There is no "is this nearer the start or
  the end" arithmetic anywhere in the file.
- **Tapping one day twice is a one-day range**, start and end on one cell,
  because the second tap is the other end wherever it falls.
- **The sheet closing on completion** makes the "third tap" almost always the
  first tap of a *reopening*, and the rule reads identically either way.

`TestDateRangePickerThirdTapStartsANewRange` pins it end to end, including that
the old span leaves the grid on that tap so the two are never both lit.

## 4. The second tap may be the earlier one

The one case the sketch did not ask about. Tapping the 20th and then the 13th
either orders the pair or throws the second tap away as a restart.

It orders them, and `OnChange` promises `start ≤ end`. "The other end" is not a
claim about which end, and a reader who meant to start over already has the
third tap for it — restarting would discard a tap made on purpose to serve a
gesture that is already available. Material restarts; this does not, and the
type doc says so rather than leaving it to be discovered.

## 5. The pending start is the widget's, which is `SliderRow`'s test run the
other way

`SliderRow` refused to hold the in-flight drag value. Same test here, opposite
answer, and the pair is now the clearest statement of it in the package: **hold
state in the widget only when no application wants it.**

A form's field is a span of days or it is nothing. "From the 14th, no end yet"
is a value a caller would have to invent a way to hold and a way to draw.

But the decisive half is not ergonomics, it is damage:

> Holding the half-made range in the caller would make closing the sheet
> *destructive*. The first tap would already have overwritten the range the
> reader opened the sheet to look at, and the backdrop, the ✕ and the back
> gesture would all be traps.

So this is the package's third hook-owning widget — open, browsed month,
pending start, to `DatePicker`'s first two — and `closeSheet` clears the
pending one, which every way out of the sheet runs through.
`TestDateRangePickerLeavingMidPickChangesNothing` taps a stray day, leaves by
the ✕, and asserts the field still holds what it opened with *and* that the
discarded start does not survive to the next opening.

## 6. The corner radius is what makes the band a band

Four decisions in the drawing, and the interesting one is not a colour.

| cell | fill | radius |
| --- | --- | --- |
| endpoint, or `Selected` | `Primary` | the cell's own `Spacing.SM` |
| interior | `Primary` at 20% | **0** |
| everything else | none | `Spacing.SM` |

Dropping the radius is the whole of what stops a run of interior cells reading
as a row of abutting pills. `core` has one `BorderRadius` and not four, so the
rounded endpoint meets the square band with a small notch; the alternative was
a second `Box` inside all 42 cells — a structural change to every calendar in
the tree to round two corners. Material's range picker draws the same notch on
purpose, the endpoint being a circle over a rectangle.

Three more, each written down rather than left to be found:

- **The ring came out of the fill switch.** Today inside the band used to be
  unreachable as a fallthrough arm. It is now `if isToday && !filled`, so the
  ring survives on the band (going square with it) and is suppressed only under
  a fill, where it would be Primary on Primary. A range covering today is the
  common case, not the odd one.
- **The band runs through the adjacent days.** Cutting it at the 1st would stop
  it somewhere the reader can see no reason for. They stay dimmed and inert.
- **`Selected` and an endpoint are deliberately indistinguishable.** One "this
  day is chosen" look in the grid, so a caller setting both gets no third case;
  `DateRangePicker`, having no single selection, clears `Selected` on the
  template rather than leaving a stale one to light a day that means nothing.

`rangeBand` also refuses to thin a `Primary` it cannot parse — `withAlpha`
hands back a name or an `rgba()` unchanged, and an *opaque* brand colour there
would swallow the day numbers the band sits behind. Nine characters beginning
with `#` is what a thinned colour looks like; anything else leaves the span
unfilled rather than unreadable, the endpoints still saying where it is.
`TestEveryBundledThemeCanThinItsPrimary` checks the three bundled themes are
not in that case, because that degradation is silent by design.

## 7. The ends of a range are a suffix, and that needed justifying

`Calendar` spent two rounds *removing* suffixes: ", selected" went out as
`AccessibilitySelected`, ", today" as `AccessibilityCurrent(CurrentDate)`. Both
moves were made because a platform property turned up that said the thing
better. That is the test, and the ends of a range fail it.

- Every day of the span states `AccessibilitySelected` — ARIA's own date-range
  grid marks the whole band, and the nights between the taps are as chosen as
  the days that named them.
- No target has a property for "this is where the span begins", so that state
  cannot tell the two ends from the eleven days between them.
- The alternative to a suffix is therefore *silence*: fourteen identically
  named selected days with no findable edge, which is precisely what a reader
  arrowing across a booking is arrowing across to check.

So `dayLabel` appends ", start of range", ", end of range", or — for a one-day
span — ", start and end of range" in one clause rather than two. It goes after
a caller's `DayLabel`, in the place the natives' ", today" goes.

## 8. Two smaller things the build settled

**The range joined the anchor chain**, ahead of `Today`: `Month`, `Selected`,
`RangeStart`, `RangeEnd`, `Today`. That is what lets the picker open on the
span while clearing `Selected` and holding `Month` at zero until an arrow is
tapped — with nothing else in the chain, the fallback is the only thing
deciding which month the sheet opens on.

**The harness had to find cells by name, not by index.** `tapDay` matches the
cell's spoken date with a prefix (endpoints carry the suffix above), because a
template's `WeekStart` moves every grid index and the date does not — a bug the
template test found by lighting the 15th when it asked for the 14th.

## 9. Two concerns

Same bar as the last three sessions': permanently wrong, and ordinary on
screen.

- `ConcernDateRangePickerInert` — no `OnChange`. Every tap is taken, a range is
  completed, and it is handed to nobody; the grid reopens on the old span every
  time.
- `ConcernCalendarRangeReversed` — a `RangeEnd` before its `RangeStart`. There
  is nothing between them, so the grid draws two lone endpoints and no band,
  which is indistinguishable from two days picked out. It lives on `Calendar`
  because that is where the pair is, and the picker never produces it: it
  orders before reporting.

## 10. Lesson 4.25

`examples/tutorial/chapter4.go`, appended — chapter 4 is the widget library and
4.9 is Calendar and DatePicker, so the span picker belongs there rather than in
the forms chapter. Deep-linked lesson numbers do not move.

The demo puts both halves on one screen: the picker running the protocol, and a
static grid under it wearing the same band with no `OnSelect` at all. The
static grid takes a `DayLabel` so its cells are distinguishable from the
sheet's, which both the demo and the test need — `findNode` is depth-first and
both grids carry a cell for every day of March 2026.

`TestDateRangeLessonMakesASpanInTwoTapsAndRestartsOnTheThird` drives the three
claims a tree can answer: one tap reports nothing, a second (deliberately the
*earlier* day) reports one ordered span and counts its nights, and the band
reaches the grid that takes no taps — endpoints filled with their corners,
interior filled with radius 0, the day outside unfilled, and every cell of it
disabled.

One snag worth recording: the lesson's ASCII band diagram would not lex as Go
and the highlighter's fallback test caught it. It is a `//` comment inside the
code block now.

## 11. Fallout

- `docs/components.md`: `## DateRangePicker` between PINInput and Accordion,
  covering the protocol, the state argument and Calendar's band.
- `docs/api/` regenerated; `date_range_picker.go` registered under "Inputs &
  pickers", whose blurb gained "date ranges".
- "0 of 65 lessons opened" → 66 in README.md, docs/tutorial-interactive.md,
  `examples/tutorial/screenshot_test.go`, `internal/shotclaims/*`;
  `docs/images/tutorial-contents.png` re-taken with `wasm/shots/shoot.sh` and
  read back to confirm it says 66.
- The file census in `wasm/verify` went 568 → 570 across five sentences.
- The plan marked D6 landed, with the four decisions written up.

26 new tests in `comps` (9 on the band, 17 on the picker), one in the tutorial;
full suite, `go vet` and `gofmt` green.

## 12. Not verified, and not done

- **Neither native.** The band is a background colour and a radius on cells
  both renderers already draw, so the risk is low — but an 8-digit hex fill
  across a 42-cell grid has not been seen off the DOM, and neither has the
  three-state picker's sheet. Both natives' colour parsers accept `#RRGGBBAA`
  (checked in `GrMobStyle.kt` and `GrMobStyle.swift`), which is why the tint is
  spelled that way and why no renderer file was touched.
- **The notch is a drawing nobody has looked at on a device.** It is the
  documented consequence of one border radius rather than four, and the fix, if
  a device run says it reads badly, is a per-corner radius in `core` — a
  renderer change, and out of this plan.

## 13. Next

D7 `TimePicker`, the last of Tier D: hour and minute `Stepper`s (or two
`Select`s) in `DatePicker`'s sheet, `TwentyFourHour` matching `DigitalClock`.
The native wheel is a node type and stays out. Then Tier E as one bundle with
one lesson.
