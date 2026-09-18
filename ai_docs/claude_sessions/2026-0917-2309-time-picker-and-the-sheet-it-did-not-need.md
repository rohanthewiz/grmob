# D7: TimePicker, and the sheet it did not need

**Session:** 7b60a2a8-06a1-4066-879c-38473b021548
**Date:** 2026-09-17 23:09 (follows "date-range-picker-and-one-rule")
**Branch:** master (2673f68 → this commit)

## 1. The ask

"start D7" — `ai_docs/plans/comps-low-hanging-fruit-2.md`, the last of Tier D.
The plan's sketch: "Hour and minute `Stepper`s, or two `Select`s, in the sheet
`DatePicker` uses", with `TimePicker{Value, OnChange, TwentyFourHour,
MinuteStep, Label, Style}` and `TwentyFourHour` matching `DigitalClock`.

## 2. D6's question, answered the other way

D6 made its range picker own a *pending start* because a range has an invalid
intermediate — "from the 14th, no end yet" — that somebody must hold. D7 asked
the same question, and for a time the answer is no:

> Changing only the hour of 9:30 gives 10:30, which is a real time and exactly
> the one asked for. There is no "hour chosen, minute pending".

Everything else followed from that:

- **No sheet.** `DatePicker` has one because a month grid does not fit in a
  field, and needs no Done because one tap is the whole choice. A sheet around
  two or three choices would need a Done over a held draft, or a live value
  the ✕ cannot take back — D6's trap.
- **No hooks.** With no draft there is no state, so `TimePicker` is stateless
  like `Stepper` and `SliderRow`: renderable conditionally, testable with a
  bare `Render`. The opposite of its two date siblings, by the same test —
  *hold state in the widget only when no application wants it*, and here there
  is none.
- **Every pick reports at once.** One pick, one `OnChange`, a complete time.

## 3. Selects, not Steppers

The plan left it open. `Select`s won on four counts:

- each part is the platform's own picker (`<select>`, SwiftUI `Menu`,
  Material dropdown) — and a sheet of those would be a popup opening popups;
- 9:00 → 17:30 is two picks, where a `Stepper` wants eight taps on the hour;
- a clock wraps 23 → 0 where `Stepper` clamps;
- `core.Select` already reads `Components.Input`, so the parts wear the field
  frame without a line of chrome here.

The period is a third `Select` rather than a `SegmentedControl`: one tap more,
but the same height and frame as its neighbours in a row.

## 4. `TwentyFourHour` became `Hour24`

"Matches `DigitalClock`" was taken as the *word*, not only the behaviour:
`DigitalClock`'s field is `Hour24`, and the field that sets a clock is
configured with the clock's word. The padding rule came along — `09` on a
24-hour picker, `9` on a 12-hour one — and the group's spoken value is built
with `clockDigits`, so the field and the clock read identically.

## 5. The shape of the value

- `OnChange` gets `Value` with hour and minute replaced, **on `Value`'s date
  and in its location**. That is what lets a `DatePicker` and a `TimePicker`
  edit one `time.Time`, each its own half; lesson 4.26 is that composition,
  with one line of glue on the DatePicker side (it reports midday, so the
  clock is put back).
- Seconds and nanoseconds are zeroed: a reported value should be one the field
  would draw.
- A zero `Value` is midnight. No blank option, no placeholder — a blank would
  be the half-made time back again. An optional time is a switch beside it.
- A re-pick of the shown option reports nothing (`Stepper`'s rule). `report`
  compares with `Equal` after building the new time.
- DST gaps: `time.Date` normalizes a nonexistent wall time; documented, not
  special-cased.

## 6. 12-hour arithmetic without a case for noon

The hour picker's *values* are 0–11 and its *labels* are 12, 1 … 11. Keeping
the two apart makes the 24-hour hour `h12 + 12·pm` with no branch for 12 AM or
12 PM. Flipping AM/PM keeps `hour % 12`. The period options' values are the
identities `"am"`/`"pm"`, so a localised `AMLabel`/`PMLabel` never changes what
`OnChange` is computed from. `TestTimePickerTwelveHourArithmetic` walks
9 PM → noon → midnight → 7 AM → 7 PM.

## 7. An off-step minute is kept, not rounded

9:07 on a 15-minute picker gets `:07` inserted into the list in order
(`slices.BinarySearch` + `slices.Insert`). Rounding for display lies about the
value; rounding through `OnChange` is a write the reader never made. The test
asserts that rendering writes nothing and that the odd option leaves after the
next pick. `MinuteStep` ≤ 0 means every minute; ≥ 60 means only `:00` (plus
an off-step current minute).

## 8. Accessibility

`RoleGroup` named by `Label`, with the whole time as `AccessibilityValue` —
Stepper's shape — so a native reader hears "Start time, 9:30 AM" once. Each
`Select` is named `Hour` / `Minute` / `AM/PM` (`HourLabel`, `MinuteLabel`,
`PeriodLabel` localise), because a picker's own text is only its current
option. The colon is `AccessibilityHidden`.

## 9. One concern

`ConcernTimePickerInert` — no `OnChange`. Every pick goes nowhere; the field
snaps back or, on a target that keeps the picked option until patched, shows a
time the app does not hold. Same bar as the last four sessions'.

## 10. Lesson 4.26

`lessonTimePicker()` appended to `examples/tutorial/chapter4.go` ("Picking a
time of day"). The demo puts a `DatePicker` and a quarter-hour `TimePicker`
over one `time.Time`, a "24-hour clock" `SwitchRow`, and a `DigitalClock`
reading the same value under the same `Hour24`, with a caption of what is held.

`TestTimePickerLessonReportsEachPickAndKeepsTheDate`: opens on 09:30, two
picks land 16:30 on the same day, a date pick moves to the 20th and keeps
16:30, and the switch drops the period picker and the clock's `PM` — with the
`PM` asserted present first so its absence is not vacuous.

A cross-reference in the lesson prose first said "1.x's Stepper"; the Stepper
lesson is 4.15 and it says so now.

## 11. Fallout

- `docs/components.md`: `## TimePicker` between DateRangePicker and Accordion;
  the `core.Select` link points at `api/core-controls.md#func-select`.
- `internal/apidoc/packages.go`: `time_picker.go` under "Inputs & pickers",
  blurb gained "times"; `docs/api/` regenerated.
- 66 → 67 lessons in README.md, docs/tutorial-interactive.md,
  `examples/tutorial/screenshot_test.go`, `internal/shotclaims/shotclaims.go`;
  `docs/images/tutorial-contents.png` re-taken with
  `wasm/shots/shoot.sh tutorial-contents` and read back — it says 67.
- `wasm/verify` census 570 → 572 across the five sentences.
- The plan: D7 marked landed with the three sketch-left decisions, the status
  line says Tier D is complete, and the ordering table strikes D7.

12 new tests in `comps`, one in the tutorial; full suite, `go vet ./...`,
`gofmt` and `wasm/verify` green.

## 12. Verified, and not

- **Seen in a browser:** `htmlout.ExportHTML` of a 12-hour, a 24-hour and a
  DatePicker field, screenshotted with headless Chrome (scratch files only,
  nothing left in the repo). Three content-width pickers, a quiet colon, the
  off-step `07` shown as itself. The pickers are content-width rather than
  filling the field's width, which reads right for a time.
- **Neither native.** Only `Select` nodes, which both shells already draw as a
  `Menu` / `DropdownMenu`, so no renderer file changed — but three menus in one
  row has not been seen on a device, nor has a 60-item minute menu.

## Next

- **Tier E as one bundle with one lesson** — `KeyValueList`, `Breadcrumb`,
  `AvatarStack`, `LabeledSeparator`, `PasswordField`, `BarItem.Badge`,
  `Lightbox` (plan's order step 6).
- **Tier F once its prerequisites land** — `Heatmap` needs a sequential
  palette decision, `Histogram` a numeric axis, `RichTextView` inline spans.
- **A device pass over Tier D's pickers**, first raised in D6: the range band's
  8-digit hex fill and endpoint notch, and now `TimePicker`'s three menus in a
  row and the 60-item minute list on `MinuteStep: 0`. If the notch reads badly
  the fix is a per-corner radius in `core` — a renderer change outside this
  plan.
- *Non-goal, declined:* a native time wheel. It is a node type, and the plan
  keeps it out.
- *Non-goal, declined:* a sheet or Done button on `TimePicker` — see §2; a time
  has no half-made value to confirm.
