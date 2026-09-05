# Session: Calendar, seen from the consumer

Session: https://claude.ai/code/session_01AM7rDTyaXrFQ4X6rPDdUWp
Date: 2026-09-05 (follows "calendar-and-datepicker")

## Ask

"Adopt Calendar downstream in church_mobile" — item 1 of the previous
session's Next list.

**No grmob code changed.** The whole diff is in `../church/church_mobile`
(branch `roh/use-grmob`, commit `b983ebc`, 8 files, +505/−23), whose own doc
`ai_docs/claude_sessions/2026-0905-1003-adopt-calendar-events.md` carries the
detail. What belongs here is what the widget looked like from the other side.

## The scorecard

    Calendar.Today          -> forced the app to grow a clock seam
    Calendar.Month          -> anchored on an event's date, not on today
    Calendar.Min/Max        -> "this is all the pager has fetched"
    Calendar.Marked         -> a map lookup over the loaded days
    Calendar.OnSelect       -> a day filter on the events list
    DatePicker              -> no consumer; nothing in the app takes a date

The events tab gained a `🗓` toggle over a month grid, a dot per event day,
and a tap that narrows the list to one day.

## The three fields that earned their design

- **`Today` as a field, not a `time.Now()`.** This is the one that paid out
  hardest, and not in the way the widget's doc predicts. The doc argues
  purity and time zones; what actually happened downstream is that the app
  could not answer the question at all without a clock seam, and once it had
  one, *two existing bugs fell out with it* — `yearFilter`'s chips have been
  pinning "2021…2026" into a golden that would have broken by itself next
  January, and `chatTimestamp` was comparing a local calendar day against
  `time.Now()` with no way for a test to fix either.

  A widget that had helpfully called `time.Now()` would have left both in
  place and added a third. The refusal pushed the question up to the only
  layer that can answer it, which is the entire argument, arrived at from the
  consumer's side.

- **`Min`/`Max` as the honest bound on a paged feed.** `Marked` can only know
  the pages fetched so far, and an unbounded grid lets a reader page into a
  month that renders blank — which reads as "no events", not "not loaded".
  Bounding to the loaded span fixes it, and the bound turns out to be *exact*
  rather than conservative because the feed is date-ordered and pages append:
  the loaded set is contiguous in time, so there is no gap in the middle for a
  dot to be missing from. That same fact is what licensed the filtered list to
  drop `OnEndReached` entirely.

  Worth noting the shape this produces on a one-event fixture: exactly one
  selectable cell, both arrows dead. Correct, and startling enough that the
  behavioral tests needed a two-month fixture to exercise anything.

- **The location rule.** "The calendar works in the location of its anchor"
  read as pedantry when written; downstream it is load-bearing. The server
  sends bare `2006-01-02` dates, which decode to UTC midnight, and `Marked`
  compares by a `"2006-01-02"` key. A grid anchored on `time.Now()` would sit
  in the device's zone and hand midday-local cells to that lookup — every dot
  gone in any negative-offset zone. Anchoring on an event's own date is one
  line and makes the whole thing agree.

## Friction found

- **No way to say "nothing is selected".** `Selected` is one `time.Time` and
  `OnSelect` always reports a day, so a consumer using the calendar as a
  *filter* rather than as a *value* has to build un-choosing itself — here, a
  second tap on the selected day plus a "Show all" strip. That is probably the
  right division, since a widget inventing a deselect gesture would be
  guessing, but every filter consumer will build the same two things. A
  `Deselectable bool` that lets a second tap report the zero time is the
  smallest thing that would settle it, and is worth considering before a
  second consumer hand-rolls it differently.

- **`Marked func(time.Time) bool` cannot count.** Two services on one Sunday
  look exactly like one service. The bool answers "is there something", which
  is what this screen asks, but the very next want is a number or an
  intensity, and the signature forecloses both. `Marked func(time.Time) int`
  would have been the same cost on the day and is not a change that can be
  made now.

- **42 more controls with no role.** The day cells are tappable `Box`es with
  accessibility labels and no button semantics — the app's test harness
  addresses them by label, which works precisely because there is nothing else
  to address them by. The `Role`/`AccessibilityRole` gap is now three
  consumers deep: `DataTable`'s `<table>`, the screen-furniture bundle, and
  this.

- **`DatePicker` has no consumer here.** No screen in the app takes a date as
  input. It was built because A3 named it and because a calendar behind a
  field is the other half of the shape, but it is unproven against a real
  form — the giving and prayer-wall screens are the two that could want one
  and neither does today.

## Downstream verification

`go build`, `go vet`, `gofmt`, `go test ./...`, `-race` — clean. Three event
goldens moved by one header button and its callback ids; `sermons.html` did
not move at all, which is the check that pinning the test clock to 2026 landed
on the same year that recorded it.

Two new goldens cover the states behind a tap (`events_calendar.html`,
`events_day.html`) through a new `snapshotAfter` helper — the first time that
repo has pinned a screen state reachable only by interaction. Four behavioral
tests drive the toggle, the filter, the empty day and the range bound.

## Next

1. `components.Chip`'s selected look — Surface fill with Primary ink is the
   *less* prominent treatment on the *selected* chip. Fourth session running.
2. A `Role`/`AccessibilityRole` prop in core, now serving `DataTable`, the
   screen-furniture bundle and the calendar's day cells.
3. Consider `Calendar.Deselectable` and a counted `Marked` — both found by
   this adoption, both cheap now and awkward later.
4. The `<button>` user-agent border divergence on the two DOM renderers
   (found last session; `EmphasisGhost` draws a rule on the web and not on the
   natives).
5. Small: say "emit `OnEndReached` before the children" in its doc, for
   hand-written `core.List` callers (carried over).
6. Small: add the mid-list busy case to `EmptyState`'s doc (carried over).
7. Then Tier C: heading plumbing + `Rotate` + Compass.
