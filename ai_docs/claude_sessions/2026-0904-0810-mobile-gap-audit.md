# Session: mobile gap audit

Session: https://claude.ai/code/session_01Mv1iFwxr7pCAdxCFMKT8Pk
Date: 2026-09-04 (follows "tutorial-layout-widget-audit" in this directory)

## Ask

"Now let's audit the mobile targets for the same gap bug" — the previous
session found `core.Gap` rendering as nothing on the web (a CSS
shorthand/longhand collision in `styleFromGrMob`).

## Method

The web bug was found by driving a browser. The natives cannot be driven from
here, so the audit ran in two layers:

- **A Go sweep tool** (temporary, `gapsweep/`, deleted after) that rendered
  every importable example app plus all 40 tutorial lessons through
  `render.Manager` and tallied, per node type, how many nodes carry `Gap`,
  `RowGap` or `ColumnGap`. Lessons were walked by firing the same
  `core.ReceiveHostEvent("route", …)` the browser's hash handler sends —
  after one `RenderInitial()`, because the subscription is taken during App's
  first pass and an event fired before it logs "no consumer".
- **A read of every container path** in `Renderer.kt` and `Renderer.swift`
  against that tally, so "this drops Gap" could be graded as live or latent.

Both natives *do* build on this machine, which the previous session's notes
had written off:

- `sh ios/verify/run.sh` — replays a Go transcript through the real runtime
  files and then `swiftc -typecheck`s the view layer.
- `ANDROID_HOME="$HOME/Library/Android/sdk" ./gradlew :app:compileDebugKotlin
  --offline` — gradle does not read the env var's default, so the SDK path
  must be exported or `android/local.properties` written, or it fails with
  "SDK location not found".

## What the sweep measured

Across every example app and all 40 lessons:

| node type | Gap | RowGap | ColumnGap | total |
|---|---|---|---|---|
| Row | 383 | 0 | 0 | 430 |
| Column | 306 | 0 | 0 | 329 |
| Card | 4 | 0 | 0 | 5 |
| Box | 1 | 0 | 0 | 136 |
| Scroll / SafeArea / List / TabView / Modal / TextGrid | 0 | 0 | 0 | — |

Two more counts that decided what to touch:

- **Gap + a distributing JustifyContent: 0.** This is the combination
  Compose's `Arrangement.Center` and friends cannot express, already
  documented as a deliberate Android divergence. Nothing hits it.
- **Multi-child overlay containers: 1.** One `Box` with two children, in
  tutorial ch.1 — and it is the same node carrying the one `Box` Gap.

## Finding 1: the gap longhands were never parsed

`core.RowGap` / `core.ColumnGap` have existed in `core.Style` and been honored
by both web targets since the style-parity pass. Neither native parser
mentioned the keys — they crossed the bridge in the JSON and were dropped.

Not the web bug's mechanism (no shorthand to collide with) but its class: a
declared, documented prop that is silently inert on half the targets, which no
compiler can see because an unread JSON key is not a type error in either
language.

They were *documented* as web-only (`ROADMAP.md`, the support matrix in
`docs/concepts/styling-and-theming.md`), so this was a declared limitation
rather than a bug — except the stated reason does not hold. That row is "CSS
the natives have no direct equivalent for", and a Compose
`Arrangement.spacedBy` / a SwiftUI stack `spacing` is exactly the equivalent.
The row was already stale in the same way for `FlexWrap`, which both
renderers implement.

## Finding 2: Scroll dropped Gap on both natives

```kotlin
Column(Modifier.verticalScroll(rememberScrollState())) { … }   // no arrangement
```
```swift
VStack(alignment: .leading, spacing: 0) { … }                   // hard zero
```

A Scroll *is* a flex column on both web targets — it is in the WASM runtime's
`STACK_CONTAINERS` and `htmlout.styleValue` emits `gap` for it (the iOS source
comment even says so, one function above the hard-coded zero) — so `core.Gap`
on a Scroll spaced children in the browser and drew flush on a phone.

## Finding 3 (web, found by the audit): htmlout would not promote for the longhands

`htmlout`'s `isFlex` gate listed `Gap` but not its two longhands, with a
comment explaining that flex-wrap and the axis gaps "only have an effect once
the box is already a flex container". True for `flex-wrap`; false for the gaps,
which ask for the same spacing `Gap` asks for. So

```go
core.Column(core.RowGap(8), a, b)
```

emitted `row-gap:8px` into a **block-flow** div — inert — while now spacing on
both natives. `TestSecondaryFlexPropsDoNotCreateAFlexContainer` pinned the old
behavior and had to be split.

## The fix

Both natives gained the two fields plus a pair of derived accessors:

```
verticalGap   = RowGap    ?: Gap     // a vertical stack's spacing
horizontalGap = ColumnGap ?: Gap     // a horizontal stack's spacing
```

Named for the **axis they space along**, not for the CSS property they come
from. `row-gap` spaces items *vertically* — it is the gap between rows — and
reading the field name as the direction is the one-character mistake the pair
exists to make impossible. Every container now reads the accessor for its own
axis:

```
Column / List / Scroll   ──▶ verticalGap
Row                      ──▶ horizontalGap
wrapping Row (FlowRow /  ──▶ both: items along horizontalGap,
GrMobWrapLayout)             lines apart by verticalGap
```

`GrMobWrapLayout` needed splitting into `spacing` + `lineSpacing`: only the
in-line spacing reaches `GrMobWrapSolver`, because it is the only one line
breaking depends on.

## Changes

- `android/…/GrMobStyle.kt` — `rowGap`/`columnGap` fields + parse, and the
  `verticalGap`/`horizontalGap` accessors.
- `android/…/Renderer.kt` — `packedHorizontally`/`packedVertically` read the
  axis accessors (they are the single point where Gap survives at all, so this
  covers Row, Column, List and FlowRow's two axes at once); `GrMobScroll`
  passes `verticalArrangement = packedVertically(node.style)`. No
  justify-content dispatch on a Scroll: a scrolling axis has no leftover space
  to distribute.
- `ios/…/GrMobStyle.swift` — the same two fields and two accessors.
- `ios/…/Renderer.swift` — `GrMobFlexStack` picks its spacing off its own
  axis; `GrMobList` and `GrMobScroll` take `verticalGap`; `GrMobWrapLayout`
  splits `spacing`/`lineSpacing`.
- `htmlout/export.go` — `RowGap`/`ColumnGap` join the `isFlex` decision;
  `FlexWrap` explicitly stays out.
- `wasm/grmob-runtime.js` — the same promotion in `styleFromGrMob`'s flex
  gate. Only reachable for a node type outside `STACK_CONTAINERS`, but the
  rule is now stated identically in both web targets.
- `docs/concepts/styling-and-theming.md`, `ROADMAP.md` — `RowGap`/`ColumnGap`
  and `FlexWrap` moved from the web-only row to the all-four-targets row, with
  a note on why they belong there and the out-of-flow props do not.

## Tests

- `mobile/verify/gap_test.go` (new, 3 tests, 8 pins). Source-parsing like the
  rest of that package, for the reason `switchlabels_test.go` gives: neither
  native runs under `go test ./...`.
  - both parsers read the two JSON keys;
  - each container reads the accessor for *its* axis, and — where only one
    axis is in play — does **not** mention the other one. That negative half is
    the point: a container that reverted to the isotropic `gap` would still
    compile and still look right in every app that sets `Gap` alone.
  - Scroll does not hard-code zero spacing (`spacing: 0` on iOS, a missing
    `verticalArrangement` on Android).
- `htmlout/export_test.go` — the old combined pin split into
  `TestFlexWrapAloneDoesNotCreateAFlexContainer` (unchanged rule) and
  `TestAxisGapsCreateAFlexContainer` (inverted), plus
  `TestAxisGapIsWrittenAfterTheGapShorthand` — the longhands must be emitted
  after the shorthand so the cascade lets the axis value win. That ordering is
  the declaration-list equivalent of the CSSOM fix the previous session made.
- `wasm/verify/runtime_test.mjs` — the matching split.

All 8 native pins were confirmed to **fail** against the pre-fix sources via
`git stash push -- ios android` (`gap_test.go` is untracked, so it survives the
stash), then `git stash pop`.

## Reported, not changed

- **`Box` overlays its children on both natives, stacks them on the web.**
  `Box` is in the runtime's `STACK_CONTAINERS` (flex column) but is a Compose
  `Box` / SwiftUI `ZStack` on device. The sweep found exactly one multi-child
  Box in the corpus — tutorial ch.1's Box-vs-Card demo,
  `core.Box(Gap(4), Text, Text)` — so those two labels currently draw on top
  of each other on a phone. A container-semantics divergence rather than a gap
  bug; picking a side is a design call.
- **Android's five distributing arrangements still drop the gap.** Documented
  as deliberate, 0 live sites.
- **`SafeArea` / `TabView` / `Modal` / `TextGrid` drop Gap on the natives.**
  All latent (0 occurrences) and each defensible: `TextGrid` is a `<pre>` on
  the web where making it flex would break the grid, `core.Modal` carries no
  Style at all, `SafeArea` is single-child everywhere in the corpus.

## Gotcha: htmlout does not stack a plain Column

Noticed while checking the promotion rule, out of scope, worth knowing:

```html
<div style="padding:12px 16px"><span>a</span><span>b</span></div>
```

The WASM runtime plants `display:flex` on every `STACK_CONTAINERS` type
whether or not the Style asks; `htmlout` has no such default, so a Column with
no flex props is block flow and its `Text` children (spans) run together on one
line. Separate from the gap work and not touched.

## Verification

`go test ./...`, `gofmt`, `go vet`, `sh wasm/verify/run.sh`,
`sh ios/verify/run.sh`, and `:app:compileDebugKotlin` — all clean.
