# Comps round four, phase 1: the chat family (TypingIndicator, ReactionBar, Poll)

**Session:** 0950b526-be6a-4eb2-a57f-14a47f95580e
**Date:** 2026-09-21 09:12
**Branch:** master (ba30444 → this commit)

## The ask

1. "Start the first phase of this plan" — `ai_docs/plans/comps-low-hanging-fruit-4.md`,
   Phase 1: J1 `TypingIndicator`, J2 `ReactionBar`, J3 `Poll`, one lesson,
   and `examples/chat` as the first consumer.
2. `/sw`.

## What landed

| Piece | Files |
|---|---|
| J1 `TypingIndicator` | `comps/typing_indicator.go`, `_test.go` |
| J2 `ReactionBar` | `comps/reaction_bar.go`, `_test.go` |
| J3 `Poll` | `comps/poll.go`, `_test.go` |
| First consumer | `examples/chat/main.go`, `main_test.go` |
| Lesson 4.34 "Typing, reactions and polls" | `examples/tutorial/chapter4.go`, `chapter4_test.go` |
| Docs | `docs/components.md`, `docs/api/` regenerated, `internal/apidoc/packages.go` (the three files are on the "display" topic) |
| Lesson count 75 → 76 | README.md, docs/tutorial-interactive.md, `wasm/index.html`, `internal/shotclaims`, two tutorial test comments, `docs/images/tutorial-contents.png` re-taken |
| Census 630 → 636 tracked Go files | `wasm/verify/repowalks_test.go`, `timings_test.go` |
| Bookkeeping | the plan (status, three "What the build changed" blocks, order row 1 struck), `ai_docs/todo/next-list.md` (N-062) |

`go test ./...` passes. No file under `htmlout/`, `android/` or `ios/`
changed. Under `wasm/`, only the page's lesson count and the census figures.

## Where the build left the sketch

Each is also written into the plan, under its item.

### J1 — the dots fade by colour, because core has no opacity

The plan had `core.Transition` easing each dot's *opacity*. `core.Style` has
no opacity (checked: no prop in core, no handling in the Compose runtime), and
adding one is renderer work on three targets, which this round rules out. What
`Transition` does animate everywhere is background colour. So:

- two hooks, both before any branch: `phase` (0 to 2) and
  `hooks.UseIntervalWhile(ctx, Visible, step, 400ms)`;
- the dark dot is `TextPrimary`, the others rest at `ControlBorderColor()`,
  each with `Transition(300, EaseInOut)`.

The first choice for the resting tone was `TextSecondary` with an alpha byte
(`withAlpha`). DefaultTheme's `TextSecondary` is `#3C3C4399`, which already
has one, so `withAlpha` returns it unchanged and the dots would not have
moved at all. `Border` was the second choice and is 1.26:1 on the page, so
invisible. `ControlBorderColor()` is the tone that works in all three themes.

`Spinner`'s doc argues against stepping an animation from Go. The widget's doc
answers it: there is no native loop to hand this to (`core.Spin` declines to
generalise), the rate is 2.5 passes a second, and it runs only while
`Visible`.

Hidden is `Display none` (`Spinner.Hidden`'s answer), not the plan's
zero-height box. `Visible`'s zero value is hidden on purpose.

### J2 — the state is not in the name

The plan's spoken name ended "including yours". `Chip` already sends `Mine` as
`core.AccessibilitySelected`, and its doc argues that a state in a name is
announced twice. So the name is "thumbs up, 3 reactions" (`countNoun` spells
the singular). Added: `Trailing core.View` (the plan pointed at
`ChipStrip.Children`, which the bar does not expose) and `GroupLabel`. An
empty bar is `Display none`. Chips are `Keyed` by emoji.

### J3 — `Voted int` became `PollOption.Mine bool`

An `int` field defaults to 0, so `Poll{Question, Options, OnVote}` would have
opened already voted for its first option, with its buttons gone. A bool per
option has the right zero and matches `Reaction.Mine`. The concern is now
"asking, no `OnVote`, and neither `ShowResults` nor `Disabled`". Options are
outlined buttons, not ghost (`Stepper`'s finding). `pollPercents` is
largest-remainder in integer arithmetic with a stable sort, table-tested.

## What the headless render found

The plan's "render the lesson and look at it" step found two defects in
`Poll` that every test had passed over:

1. Each result row was indented from the question and the total: the result
   `Column` kept the theme's Column inset. Fixed with `core.Padding(0)`.
2. The question was in `Typography.Subtitle`'s grey, and over a column of
   buttons it read as disabled. Fixed with `TextColor(TextPrimary)`.

Both are pinned in `TestPollResultsAreOneStopPerOption`. That is three rounds
running (G1, G3, J3) where looking found what asserting did not.

How the look was taken, for next time: two throwaway scripts in
`wasm/shots/scripts/` (`tap("Chapter 4")`, `tap(<lesson title>)`,
`scrollTo(...)`, frame height 1500), run with
`GRMOB_SHOTS_OUT=<scratch dir> ./shoot.sh <name>`, then deleted. `shoot.sh`
only accepts names under `scripts/`.

## `examples/chat`

- `Message.Reactions`, a `react` choke point beside `send` (it copies the
  thread *and* the one message's `Reactions` slice; a test proves the old
  tree is not mutated), `MessageRow` = bubble + bar.
- `TypingIndicator` sits after the `RoleLog` column, inside the scroller, and
  is always rendered. `send` raises it.
- **The hard-coded `cb_1` is gone.** Reaction chips register callbacks before
  the composer, so the send button's ID moved. `buttonCallback(tree, label)`
  reads it off the tree, in `main` and the tests.
- `TestTheTranscriptIsALogAndNotAStatus` asserted no status *anywhere*. The
  indicator is a legitimate status, so the test now asserts the log holds
  none, which is what it was about.

## Slips

- `go build ./examples/chat` from the repo root overwrote the tracked `chat`
  binary there. Restored with `git checkout -- chat`. Use `go vet` or
  `go build -o /dev/null` for a compile check of an example.
- The lesson first used a `heading(...)` helper that does not exist; lessons
  are prose, code, demo panels and key points only.

## Noticed, not touched (N-063)

- README's chapter table says chapter 4 has 14 lessons. It has 34.
- The plan's "still blocked" list still carries "Per-corner radius, and so
  bubble tails". `core.CornerRadii` exists and `MessageBubble` uses it.

## Next

Closed: None. Declined: None. Raised: N-062, N-063.
Deferred: None. Promoted: None.
Updated: None. Full list: `ai_docs/todo/next-list.md`.
