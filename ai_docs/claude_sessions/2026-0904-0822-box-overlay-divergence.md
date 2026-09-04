# Session: the Box overlay divergence

Session: https://claude.ai/code/session_01Mv1iFwxr7pCAdxCFMKT8Pk
Date: 2026-09-04 (follows "mobile-gap-audit" in this directory, same session —
that audit found this and left it as a design call; this is the call)

## Ask

"Now fix the Box overlay divergence."

## What it was

One node type, three behaviors:

| target | `core.Box(a, b)` |
|---|---|
| WASM runtime | `display:flex; flex-direction:column` — stacks (Box is in `STACK_CONTAINERS`) |
| Android | `Box(...)` — a Compose Box, children overlap at TopStart |
| iOS | `ZStack(alignment: .topLeading)` — children overlap at topLeading |
| `htmlout` | block flow — see the note at the end, a separate bug |

So a `Box` with two children drew its second child *on top of* the first on
device and *underneath* it in the browser.

## Deciding which side is right

`core.Box`'s own doc comment said only "the unopinionated container", which
settles nothing. Three other places do:

- `docs/concepts/views.md`: "The **flex-style containers** — `Row`, `Column`,
  `Card`, `Box`, `List` — share one argument contract", and the only stated
  difference from Column is "no theme base by design".
- `core.Scroll`'s doc comment: "Like Box, it is the unopinionated container" —
  and Scroll stacks vertically on all four targets.
- The tutorial's own ch.1 prose, shipped and rendered: "Box and Card hold
  children **like the stacks do**, but carry different opinions."

Also worth noting: overlay is not reachable from the DSL's vocabulary anyway.
`core.Style` is CSS-flavored, and CSS's overlay is `Position: absolute`, which
is a declared web-only prop. "Box = ZStack" was never something an app could
have relied on deliberately.

So the natives were the outlier and Box stacks vertically.

## Measuring before touching

A sweep (same tool shape as the gap audit's, temporary, deleted) over every
example app and all 40 tutorial lessons:

```
Box nodes:                        136
  children by count:              map[0:135 2:1]
  with >1 child (would restack):  1
  with Align set, AlignItems not: 0
```

Two things this settled:

- **Box is overwhelmingly a styled void.** 135 of 136 have no children at all
  — `core.Divider`, `components.Separator`, the ProgressBar fill,
  `core.Box(core.FlexGrow(1))` used as slack. The one with children is
  tutorial ch.1's Box-vs-Card demo, i.e. the lesson about what a Box *is* was
  the only thing the bug was visibly breaking.
- **The cross-axis edge case is not live.** Routing Box through the Column
  path makes it read the `Align` fallback, which the web declined for Box.
  That only differs when `Align` is set and `AlignItems` is not — zero
  occurrences — so reusing the Column path was safe rather than merely
  plausible.

## The change

Both renderers give Box Column's dispatch arm:

```kotlin
"Column", "Card", "Box" -> GrMobColumn(node, extra)
```
```swift
case "Column", "Card", "Box": GrMobColumn(node: node, grow: grow)
```

That is the entire fix. Box inherits the arrangement, the cross-axis dispatch,
the stretch default, the children loop's weight/fill handling, and the gesture
wiring already written for Column — `GrMobColumn` attaches onTap/onLongPress
itself, so the old Box arm's explicit `grMobBox(onTap:onLongPress:)` was not
lost, it was absorbed.

Box also joins `alignFallbackAxes` in both DOM targets (`htmlout/crossaxis.go`
and the runtime's `alignFallbackAxisFor`). That table's stated invariant is
"exactly the containers the natives read the fallback for"; the natives read
it for Box now, so leaving Box out would have made the comment false.

## The test that was supposed to fail, and did

`TestAlignFallbackGateMatchesTheNativeContainers` failed on the table change:

```
crossaxis_test.go:90: gate includes "Box", which no native reads the fallback for
```

Its own comment says growing the set is legitimate and "the test makes the
growth a visible decision on both DOM targets at once". So it was updated with
the reasoning rather than routed around — including the observation that this
growth arrived from an unexpected direction: the *type* did not change, the
natives did.

## Changes

- `android/…/Renderer.kt`, `ios/…/Renderer.swift` — Box shares Column's arm;
  the old overlay arms are gone.
- `htmlout/crossaxis.go`, `wasm/grmob-runtime.js` — `Box: "column"` added to
  the align-fallback gate, with the reason in the table's comment.
- `htmlout/crossaxis_test.go` — expected set grown to four, with the "the
  natives changed, not the type" note.
- `core/layout.go` — `Box` gains a real doc comment: a Column with no theme
  base, not an overlay, on any target.
- `docs/concepts/views.md` — same, in the container list.
- `mobile/verify/box_test.go` (new) — Box must share Column's arm on both
  renderers, and that arm must not build the overlay it replaced (`ZStack` /
  `Box(`). Checked on the arm rather than the file, because both names stay in
  use elsewhere — CameraView is a genuine overlay on both natives.
  Confirmed to fail against the pre-fix sources via
  `git stash push -- ios android`.
- `mobile/verify/gap_test.go` → the Box pin was split out into `box_test.go`;
  it came out of the gap sweep but is not about gaps.

## Prose that had gone stale

Three comments asserted the old behavior as a live constraint, and would have
quietly misled the next reader:

- `components/avatar.go`: "Row rather than Box for the disc: Box is a ZStack
  pinned to .topLeading … and cannot centre its child." No longer true. The
  comment now says the constraint is gone and the `Row` stands on its own
  merits (it centres on two props with no theme padding to undo, and the
  markup is pinned by `TestAvatarFallbackDisc`). The implementation was left
  alone — a working choice does not need rewriting because its original
  justification expired. The test's failure message got the same correction.
- `components/progress_bar.go` and its test: "a Compose Box and a SwiftUI
  ZStack both size to their content" → "a stack sizes to its content on both
  natives", which stays true either way.

## Left outstanding, deliberately

- **`SafeArea` has the identical defect** — a Compose `Box` / SwiftUI `ZStack`
  on the natives, a stack on the web. Not fixed here: converting it also puts
  every screen's content column under `ColumnChildren`'s `fillMaxWidth`, so
  all 43 SafeArea nodes change how their *child sizes*, not just how two
  children would overlap. That is a real rendering change across every screen
  and deserves its own pass with its own before/after, not a rider on this one.

  **Done next, same session** — see `2026-0904-0830-safearea-stacking.md`. The
  sizing change turned out to be the same correction `isColumnStretch` already
  documents for Column, which is what settled it. `mobile/verify/box_test.go`
  was replaced by `stacking_test.go`, covering both node types under one rule.
- **`htmlout` renders every stack container as block flow.** Carried over from
  the gap-audit session. The WASM runtime plants `display:flex` on every
  `STACK_CONTAINERS` type whether or not the Style asks; htmlout has no such
  default, so a Row or Column with no flex props runs its `<span>` children
  inline. Confirmed against `examples/layout`'s output, where `BodySection`'s
  Row exports as a bare `<div style="padding:…">`. Affects Box no more than
  the rest.
- **CameraView stays an overlay** on both natives, which is what it is for.

## Verification

`go test ./...`, `gofmt`, `go vet`, `sh wasm/verify/run.sh`,
`sh ios/verify/run.sh`, and `:app:compileDebugKotlin` (re-run with
`--rerun-tasks`, no warnings) — all clean.
