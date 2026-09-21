# core.Opacity on all four targets, and TypingIndicator's dots fade by it

**Session:** a42380e9-22c9-4059-9385-6d053eae4b87
**Date:** 2026-09-21 09:35
**Branch:** master (41e87c9 → this commit)

## The ask

1. `/sl` (loaded `2026-0921-0912-comps-round-four-phase-1-chat-family` and the
   next list).
2. "Go ahead and add Opacity to core Style" — the gap the previous session's
   J1 ran into.
3. "Convert the TypingIndicator to use opacity."
4. `/sw`.

## What landed

| Piece | Files |
|---|---|
| The prop, the sentinel, the reading | `core/opacity.go` (new), `core/style.go` (field, `applyTo` arm, the struct doc's "one field" note is now two), `core/animation.go` (Transition's list) |
| htmlout | `htmlout/export.go` |
| WASM runtime | `wasm/grmob-runtime.js` (`styleFromGrMob`) |
| Compose | `GrMobStyle.kt` (`opacity`, `alphaOf`, `boxModifier`), `Renderer.kt` (`animatedStyle`) |
| SwiftUI | `GrMobStyle.swift` (`opacity`, `alpha`, grMobBox's one `.opacity`) |
| Tests | `core/opacity_test.go`, `htmlout/export_test.go`, `wasm/verify/opacity_test.go` + `.mjs`, `wasm/verify/totality_test.mjs` (FULL_STYLE), `mobile/verify/opacity_test.go` |
| TypingIndicator | `comps/typing_indicator.go`, `_test.go`, lesson 4.34's key point |
| Docs | `docs/concepts/styling-and-theming.md` (an Opacity section, the support table), `docs/components.md`, ROADMAP, `internal/apidoc/packages.go` (opacity.go on the style-props topic), `docs/api/` regenerated |
| Census 636 → 640 tracked Go files | `wasm/verify/repowalks_test.go`, `timings_test.go` |
| Bookkeeping | `ai_docs/todo/next-list.md` (N-064 raised, N-062 updated) |

## The design

### Zero is a sentinel: `core.OpacityClear = -1`

Opacity is the second Style number whose CSS initial value is 1 (flex-shrink
was the first). A plain zero would be lost three times: `omitzero` on the
wire, `applyTo`'s non-zero-wins merge, and every renderer's guard. So
`core.Opacity(0)` stores -1, deliberately `ShrinkNone`'s number, and renderers
read `Style.OpacityFactor() (alpha, declared)`.

- The argument is clamped to [0, 1]; NaN stores unset (opaque).
- `Opacity(1)` stores 1, not unset. Both ends of the range are non-zero, so a
  merged style can take a node to transparent *and back*, which Rotate's
  merge cannot do.
- `OpacityFactor` also clamps a hand-built Style, so no renderer range-checks.

Declined: storing the complement (a `Fade` whose zero is opaque, no sentinel).
The wire's field names are CSS's on purpose, and `1-(1-0.7)` is
`0.7000000000000001` in every renderer that undoes it. The reasoning is in
`core/opacity.go`.

### Per target

| Target | Mapping | Note |
|---|---|---|
| htmlout | `opacity:%g` when declared | undeclared writes nothing; existing exports are byte-identical |
| WASM | `out.opacity`, total | `-1 → "0"`; dropping the prop clears the declaration on the reused element |
| Compose | `Modifier.alpha`, beside `rotate` | outside shadow, fill and border, so the box fades as one picture |
| SwiftUI | folded into the existing `.opacity(hidden ? 0 : alpha)` | no new layer on the chain that has crashed SILGen before |

Compose resolves the alpha **at parse** (`alphaOf`), unlike `flexShrink`,
which keeps its sentinel behind a getter. `animatedStyle` writes each frame's
alpha back with `copy`; stored raw, a frame that reached 0 would read as
"unset" and the node would flash opaque as its fade ended. The animated value
is `coerceIn(0f, 1f)`, because `Modifier.alpha` throws outside the range and
an easing may overshoot.

SwiftUI eases it for free: the node's one `.animation(value: s)` is outside
the `.opacity`.

Documented on the prop: group opacity on every target; paint only (no
reflow); a faded node keeps its semantics and, on the web and Compose, its
taps. SwiftUI stops hit-testing at exactly 0 — stated from SwiftUI's known
behaviour, not measured here.

### What the tests pin

- `wasm/verify/opacity_test.go`: the sentinel's number in the JS, Kotlin and
  Swift runtimes, each pattern spanning the *unset* arm too (an unset Opacity
  read as 0 would draw every node transparent).
- `mobile/verify/opacity_test.go`: both parsers read the key; both renderers
  hand the platform the *reading*, not the raw field; the alpha layer is
  outside the painted box on both natives and inside SwiftUI's
  `grMobTransition`; `animatedStyle` eases and clamps.
- `opacity_test.mjs`: fade to the sentinel and back over patches, ending with
  no declaration.

## TypingIndicator

All three dots are `TextPrimary`; the dark one is `Opacity(1)`, the others
`typingRestAlpha = 0.4`, each with the same 300ms Transition. One patch per
beat, as before.

- 0.4 is fitted to the tone already looked at: black at 0.4 over `#F2F2F7` is
  about `#919194`, against `ControlBorderColor()`'s `#89898E`. (0.3 was the
  first pick and composited visibly lighter, about 2.2:1.)
- The widget no longer needs a second tone that reads on the bubble in every
  theme, which took three tries last session. `TextPrimary` on `Surface` is
  legible by definition.
- The dark dot declares `Opacity(1)` rather than leaving the field unset, so
  neither state depends on what an absent key means to a renderer.
- `darkDot` in the test finds the dot by alpha and requires every dot to be
  `TextPrimary` with a *declared* opacity of 1 or 0.4.

## Verification

- `go test ./...` passes; `wasm/verify/run.sh` exits 0.
- `ios/verify/run.sh`: the view layer type-checks **and survives the Release
  whole-module build**, which is where that modifier chain has crashed.
- `./gradlew :app:compileDebugKotlin --offline` succeeds.
- Lesson 4.34 looked at in headless Chrome (throwaway
  `wasm/shots/scripts/zz-typing.js`, deleted): one dark dot, two greys.
- Not run on any device or simulator.

## Noticed, not touched (in N-064)

Compose's `DisplayHidden` `alpha(0f)` sits at the foot of `boxModifier`,
inside the background and border, so a hidden node with a fill may still draw
the fill. iOS and the web hide the whole box.

## Next

Closed: None. Declined: None. Raised: N-064.
Deferred: None. Promoted: None.
Updated: N-062. Full list: `ai_docs/todo/next-list.md`.
