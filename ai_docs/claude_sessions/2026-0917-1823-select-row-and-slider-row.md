# D4: SelectRow and SliderRow, and two answers to "who holds the state"

**Session:** cb8632ac-d2e2-46bb-81a4-88d2178998ae
**Date:** 2026-09-17 18:23 (follows "qrcode-comp-and-an-encoder-of-our-own")
**Branch:** master (96ba6fd → this commit)

## 1. The ask

"start D4, SelectRow and SliderRow" — the next item in
`ai_docs/plans/comps-low-hanging-fruit-2.md`, and the one the plan's own order
put third because it "completes a family, zero decisions".

It did complete the family. It did not have zero decisions.

## 2. What the family looked like before

`comps/settings_row.go` holds `SwitchRow` and `CheckboxRow`: a `ListRow` with
the control fixed to the trailing edge, the whole row tappable, and a long doc
about the web dispatching one tap twice. The two missing members are the value
that is *one of a few names* and the value that is *a number*, both of which
were hand-rolled wherever they were wanted.

## 3. `SelectRow`

```go
comps.SelectRow{Title: "Theme", Options: themeChoices,
    Value: theme.Get(), OnChange: theme.Set}
```

A `ListRow` with the current choice trailing, and a `comps.ActionSheet` of the
alternatives behind a tap on the row, `Checked` on the current one.

**Not `core.Select` in the trailing slot**, and the reason is the one
`SwitchRow`'s doc already works through from the other side. On the web a click
on a control bubbles to the row, so a row holding both would hear one tap
twice. `SwitchRow` survives that because both handlers *set* a value and the
second is dropped by a guard; two *openings* have nothing to converge on. The
lesser reason is the target: a settings list is tapped anywhere along its
width, and a picker in the trailing slot is only as wide as its longest label.

So the row is the control — `RoleButton` plus `PopupDialog`, as `DatePicker`'s
trigger — and it owns the one state nobody else wants, which is whether the
sheet is open. That carries `DatePicker`'s hook obligation with it: rendered
unconditionally, stable position, every pass.

**`core.SelectOption.Group` is the field that does not survive the trip.** A
sheet action is a button with one line and no section construct, and inventing
one here would be a third answer to a question `core.Select` (an `<optgroup>`)
and `SearchableSelect` (the row's subtitle) have each already answered.
Documented as a limit that points at both. `Disabled` and `GroupDisabled` are
read per option, which is what `SearchableSelect` does with the same pair.

**A `Value` no option carries** shows the placeholder and raises
`comps.ConcernSelectRowValueNotAnOption` in debug builds. On screen that
mistake is indistinguishable from a row nobody has set, so nothing else would
ever find it. An empty `Value` is quiet — "not chosen yet" is a state a
settings row legitimately starts in.

## 4. `SliderRow`, and the decision that went the other way

```go
comps.SliderRow{Title: "Text size", Min: 12, Max: 24, Step: 1,
    Value: size.Get(), OnChange: size.Set}
```

**It owns nothing at all**, and that is the interesting half.

The tempting mirror of `SelectRow` is to hold the in-flight drag value so the
reading beside the title follows the finger. `core.State.Set` requests a render
of the *whole tree*, so a widget holding that draft would charge every
`SliderRow` in the framework a full render pass per drag tick — which is
precisely the cost the plan named this item to avoid — to make one of them look
livelier. `OnChange` therefore rides on `OnSliderChangeEnd`, and `OnDrag` (nil
by default) hands the live case to the caller, who holds the draft:

```go
draft := core.NewState(ctx, -1.0) // -1: not dragging
shown := size.Get()
if draft.Get() >= 0 { shown = draft.Get() }
```

The visible cost is small and worth stating: the *thumb* still follows the
finger on every target (core.Slider's own compromise), so only the number
lags. And the row stays hook-free, which means conditional-safe — the exact
opposite of `SelectRow`, one file over. The pair is now the clearest statement
of that trade in the package, which is why they share a tutorial lesson.

**The row is not a tap target.** A switch has one other state a row-sized
target can reach. A slider's "other value" is the one under the tap, and Go
sees no coordinates — so there is no `OnTap` and no role, and the slider is the
control a reader is looking for.

**Where the track goes.** Title and reading on one line, track beneath, and
both inside the row's *growing middle column* rather than beside it. That is
what aligns the track's start with the title's whatever the leading icon's
width, with no arithmetic against the theme's row padding — which a sibling
`Column` would have needed and which no theme guarantees. It costs one `Box`,
since `ListRow.Content` is a single view and there are two things to put in it.
The header is itself a nested `ListRow` with `Padding(0)`, so the typography
and the trailing pin come from the same place every other row gets them.

**The default readout.** Not a bracket the step falls into but the step's own
precision: the smallest *d* for which step·10^d is whole, capped at three. A
row stepping by 0.25 that rounded to one decimal would show the same reading
for four adjacent positions. With no step the span stands in — none over 10,
one over 1, two at or below. The bottom bracket is deliberate: one decimal over
a 0..1 range gives eleven readings, so the number would sit still across a
tenth of the thumb's travel.

## 5. Lesson 6.8

`examples/tutorial/chapter6.go`, appended for the reason 4.15, 4.22 and 4.23
were — lesson numbers in deep links do not move. Chapter 6 rather than 4
because that is where 6.6 teaches the family and 6.7 teaches the sheet this
opens.

The demo holds the slider's draft itself, so the lesson can show the two
numbers disagreeing mid-drag, and
`TestMoreSettingsRowsLessonCommitsOnlyWhenTheDragEnds` asserts exactly that:
after an `onChange` tick the reading says 22 pt and the committed caption still
says 16, and only `onChangeEnd` moves it. The disagreement *is* the claim, so
the test pins it rather than the settled state.

## 6. Fallout

- `docs/components.md`: `## SelectRow` and `## SliderRow` after the SwitchRow
  section. `docs/api/` regenerated; both files registered under "Lists &
  tables" in `internal/apidoc/packages.go` (the generator refuses a file no
  topic lists — `ConcernSelectRowValueNotAnOption` is what tripped it).
- "0 of 62 lessons opened" → 63 in README.md, docs/tutorial-interactive.md,
  `examples/tutorial/screenshot_test.go`, `internal/shotclaims/*`;
  `docs/images/tutorial-contents.png` re-taken with `wasm/shots/shoot.sh`.
- The file census in `wasm/verify` went 560 → 564 across five sentences, the
  four new `comps` files.
- The plan marked D4 landed, with the three decisions written up.

21 new tests in `comps`; full suite and `go vet` green.

## 7. Not verified

The two natives. The track states `Width("100%")` because a Compose child hugs
its content otherwise, but the nested shape — a `ListRow` inside another row's
middle column — has only been seen on the DOM. If the header's trailing reading
drifts off the track's right edge there, the fix is in `SliderRow.header`, not
in the renderers.

## 8. Next

D3 per the plan's order: `Countdown` and `Stopwatch`. Alarms exist and cannot
show time left, and the one decision — `OnDone` fires from the hook's effect,
once, on the tick that crosses zero, never from the render pass — is already
settled in the plan.
