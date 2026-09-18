# D5: PINInput, and a code that cannot hold a gap

**Session:** 2214f9ac-de3a-4418-8b09-f8c1bfcf5570
**Date:** 2026-09-17 19:35 (follows "countdown-and-stopwatch")
**Branch:** master (62797f8 → this commit)

## 1. The ask

"start D5" — the next item in `ai_docs/plans/comps-low-hanging-fruit-2.md`,
fifth in the plan's order and the first of the three the plan said could be
taken in any order.

## 2. What the widget is for

Six boxes, one character each, for a one-time code. The reason it is a widget
rather than six fields in a `core.Row` is the cursor: typing a character
should move it on, and nothing in `comps` had ever asked core's focus system
for anything.

So `PINInput` is the first widget in the package to drive it — one
`core.FocusRef` per cell, one `core.UseFocusOrder` over them, `core.FocusNext`
after every character written. The order buys a second thing for free that the
sketch did not mention: `stampTraversal` gives every cell but the last the
keyboard's **Next** action, wired to the cell after it, so the field advances
from the soft keyboard as well as from a keystroke.

The struct shipped exactly as sketched — `Length`, `Value`, `OnChange`,
`OnComplete`, `Secure`, `Label`, `Style` — no field added or dropped. Which is
unusual for this plan and worth noting: `Stopwatch` needed a second field last
session and `SelectRow` needed a rethink of who holds state. Here the shape
was right and everything interesting was in the semantics.

## 3. `Value` is one string, and that decides most of the rest

The one real design question. `Value` is the whole code, not a cell array:
cell *i* draws the *i*-th character and the empty cells are the ones past the
end.

A plain string cannot hold a gap. So the cells fill strictly left to right,
and there are exactly two edits with no cases left over:

```
typing at cell i    Value = code[:i] + typed + whatever the run did not cover
clearing cell i     Value = code[:i]     — cell i and everything after it goes
```

**Clearing is the asymmetric one and it is the decision.** A cleared middle
cell has to either shift the tail left — so cells the finger never touched
change under it — or drop the tail. Dropping is the one a person can predict,
because it is what "start again from here" means, and it is what backspacing
through an OTP field amounts to on every platform that has one.

The same invariant answers a question the cells can otherwise raise, and one
the sketch did not ask about: the web lets a click land in any box, so what
happens when a character is typed into a cell *past* the end of the code? It
lands at the end. There is no position for it to occupy, because positions
past the end do not exist.

## 4. A paste and a second character are the same event

The sketch's stated decision was the paste: a full code arrives in the first
cell's `OnChange` as a multi-character string, spread it and jump to the end.
What the build found is that the *other* multi-character case is the same
operation:

| what happened | what the cell reports |
| --- | --- |
| six-digit code pasted into cell 0 | `"417293"` |
| `"2"` typed into a cell holding `"1"` | `"12"` |

The field is controlled, so it reports its entire contents either way. One
rule covers both — **write the incoming string from this cell forward, cursor
after the last cell it filled** — and the second row falls out of it correctly
without a branch: `"1"` rewrites cell 0 with the character already there, `"2"`
fills cell 1, cursor lands on cell 2.

One case it reads wrongly, stated in the doc rather than worked around: a
character inserted *before* an existing one arrives as `"21"` and is written in
that order. Nothing in the event says where the caret was, so no widget here
can tell the two apart.

The sketch's second decision needed no work, only writing down: a backspace in
an *empty* cell changes nothing and is therefore never reported at all. There
are no key events — a field reports its text, not the keys that made it — so
the cursor stays put. That is a limit of the wire and the workaround would be
a renderer change.

## 5. `OnComplete` is deliberately not `OnDone`

Last session's `Countdown.OnDone` fires once per *crossing*, from an effect
keyed on whether the deadline has passed. The obvious move was to mirror it.
It is wrong here, twice:

```
Countdown.OnDone      once per crossing, from an effect
PINInput.OnComplete   on every edit that leaves the code full, from the handler
```

- **Per crossing would not re-submit a correction.** `OnComplete` means
  "submit this code". Somebody who mistypes one digit of a full code, fixes
  it, and gets silence has a field that will not submit.
- **From an effect would resubmit on mount.** An effect's deps run on the
  first pass, so a screen restored with a complete code already in it would
  fire on sight. `Countdown` wants exactly that — a deadline restored from
  disk while the app was closed really has passed — and a code restored from
  state has not just been entered.

An edit producing the value already held is treated as an echo: no
`OnChange`, no cursor move, no `OnComplete`. Both natives can report their own
text back after a Go-side update, and none of the three is worth doing twice.

## 6. The hook count follows the high-water `Length`

One `FocusRef` per cell means one hook slot per cell, and `Length` is a field
a caller can change between passes. A `Length` that *shrank* would retire slots
from the middle of the sequence and drift every cursor after it — including
slots belonging to the component that rendered the widget.

So the count only ever grows. One slot of its own, ahead of the refs, holding
a `*int` that is bumped during the pass:

```go
slot := core.NewState(ctx, new(int))
high := slot.Get()
if n > *high { *high = n }
refs := make([]*core.FocusRef, *high)
for i := range refs { refs[i] = core.UseFocusRef(ctx) }
core.UseFocusOrder(ctx, refs[:n]...)
```

A pointer rather than the slot's value because `State.Set` would request a
render of the whole tree for a number no tree reads. Only the *live* cells go
into the order, so the keyboard's Next key stops at the end of the field
rather than at the high-water mark. The cost is a handful of refs nothing
points at, a slice entry each, never stamped onto a node.

`TestPINInputKeepsItsHookSlotsWhenLengthShrinks` pins it where it can actually
fail: it renders at `Length: 6`, then at `Length: 4`, with a
`core.NewState(ctx, "kept")` allocated *after* the widget. Without the
high-water mark that sentinel lands on a `*FocusRef`'s slot and the typed
accessor panics.

## 7. Two smaller things the build settled

**The cells divide the row with `FlexGrow(1)` and `FlexBasis("0")`** —
`Calendar`'s day-cell pair. Compose and SwiftUI divide the whole axis by
weight and ignore the basis; CSS divides only the leftovers and needs the zero
to start from. Without it a filled cell is a hair wider than an empty one and
the boxes shuffle as the code is typed.

**They take the text keyboard, not the number pad.** The numeric keyboard is
chosen by node type on both natives — `"NumericInput"` is the numeric one —
and that node carries an `int` value, which cannot express an empty cell:
clearing one would report nothing at all and backspace would stop working
entirely. A digits-only keyboard needs a keyboard-type prop on `core.Input`,
which is a renderer change and stayed out of this plan.

## 8. Two concerns

Same bar as the last two sessions': permanently wrong, and ordinary on screen.

- `ConcernPINInputInert` — no `OnChange`. Every keystroke reaches the handler,
  is discarded, and the next pass paints `Value` back over it. An inert
  `PINInput` is indistinguishable from one nobody has typed into.
- `ConcernPINValueTooLong` — a `Value` longer than the field. The extra
  characters are never drawn and can never be typed away, and `OnComplete`'s
  "the code is as long as the field" test would be met by characters nobody
  entered.

## 9. Lesson 5.7

`examples/tutorial/chapter5.go`, appended — chapter 5 is Forms & Validation
and 5.6 is the pickers, so the code field belongs there rather than in the
widget-library chapter.

`TestPINLessonSpreadsAPasteAndDropsTheTailOnAClear` drives the three claims a
tree can answer: one character advances the cursor (`focusAction == "focus"`
on exactly one cell is the visible stamp), a paste in the first box fills the
field and completes it once, and clearing a middle box drops the tail.

The demo's caption counts completions out loud, so it got a `time`/`times`
branch rather than reading "fired 1 times" after every first success.

## 10. Fallout

- `docs/components.md`: `## PINInput` between FormField and Accordion.
  `docs/api/` regenerated; `pin_input.go` registered under "Inputs & pickers",
  whose blurb gained "one-time code fields".
- "0 of 64 lessons opened" → 65 in README.md, docs/tutorial-interactive.md,
  `examples/tutorial/screenshot_test.go`, `internal/shotclaims/*`;
  `docs/images/tutorial-contents.png` re-taken with `wasm/shots/shoot.sh` and
  read back to confirm it says 65.
- The file census in `wasm/verify` went 566 → 568 across five sentences.
- The plan marked D5 landed, with the four decisions written up.

16 new tests in `comps`, one in the tutorial; full suite, `go vet` and `gofmt`
green.

## 11. Not verified, and not done

- **Neither native.** The cells are ordinary `core.Input` nodes in a Row, a
  shape both renderers have had for a long time, so the risk is in the focus
  walk rather than in the drawing: six `FocusTarget`s in one order, with a
  command issued from a text callback, has not been seen off the DOM. The
  parts are pinned separately — `mobile/verify` and `ios/verify` both cover
  the focus stamps — the composition is not.
- **`examples/signup` was left alone.** The plan's "the signup example wants
  it" is motivation, not a deliverable: wiring it in would change that
  example's screenshot and its shotclaims, which is beyond the definition of
  done for a widget.

## 12. Next

D6 `DateRangePicker` and D7 `TimePicker`, in either order. D6 carries the
sketch's one decision already — the third tap, with a range already set,
starts a new range at that day rather than moving the nearer end — and needs
`Calendar` to grow `RangeStart`/`RangeEnd` and draw the fill between them. D7
is hour and minute steppers in `DatePicker`'s sheet; the native wheel is a
node type and stays out.
