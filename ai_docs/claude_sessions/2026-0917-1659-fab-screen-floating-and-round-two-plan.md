# FAB, Screen.Floating, and the round-two low-hanging-fruit plan

**Session:** 4d1397ea-c325-43fd-a019-012dc146d1f2
**Date:** 2026-09-17 16:59 (follows "foldables-window-record-and-two-pane")
**Branch:** master (08cdf15 → this commit)

## 1. The ask

1. "What other low-hanging fruit in terms of components can be added to grmob"
2. "create a new plan in ai_docs/plans/ and start the first comp"

## 2. The survey

The first-round plan (`ai_docs/plans/comps-low-hanging-fruit.md`) had landed
entire on 2026-09-12, so the obvious widgets (Dialog, Snackbar, RadioGroup,
Drawer, Menu, SearchableSelect…) were done. The survey walked `comps/`
(75 struct widgets), `core/` (the node vocabulary), `hooks/`, `richtext/`,
and the four renderers to find what had become newly cheap since:

- `core.StackAlign` on `core.ZStack` landed this week, on all four targets —
  anything that floats over content is now a stack with an aligned layer.
- `core.Canvas` with `Rect`/`Circle`/`Sector`/paths/gradients — anything that
  is a picture computed in Go is cheap (QR code, heatmap).
- `hooks.UseNow`, `UseIntervalWhile`, `UseTimeoutWhile` — time-driven widgets
  no longer re-render the app per tick.
- `core.FocusNext`/`FocusRef` — a widget can move focus between its own inputs
  (PIN input).
- Percent `Width`/`Height` resolve on every target (`GrMobStyle.kt` maps them
  to `fillMaxWidth`/`fillMaxHeight`; SwiftUI against the proposal; the DOM
  against the grid track) — a ZStack layer can be told to fill the stack.

Gaps confirmed: no inline text spans in core (so a read-only rich-text view
would flatten bold on the natives), no clipboard bridge, no scroll offset
(carousel still blocked), no QR encoder anywhere in the module tree.

## 3. The plan: `ai_docs/plans/comps-low-hanging-fruit-2.md`

Same rule as round one: pure composition, no file under `htmlout/`, `wasm/`,
`android/` or `ios/` touched.

| Tier | Widgets |
|---|---|
| D (hours each, one decision) | D1 `FAB` + `Screen.Floating`; D2 `QRCode`; D3 `Countdown`/`Stopwatch`; D4 `SelectRow`/`SliderRow`; D5 `PINInput`; D6 `DateRangePicker`; D7 `TimePicker` |
| E (an hour, no decisions) | `KeyValueList`, `Breadcrumb`, `AvatarStack`, `LabeledSeparator`, `PasswordField`, `BarItem.Badge`, `Lightbox` |
| F (a prerequisite first) | `Heatmap` (needs a sequential palette), `Histogram` (numeric axis), `RichTextView` (needs inline spans) |
| Blocked on a renderer | carousel, pull-to-refresh, swipe actions, tooltip/popover, native time wheel, clipboard copy |

Suggested order: D1, D2 (the roadmap's `cats://pair` flow has nothing to
show), D4, D3, then D5–D7, then E as one bundle with one lesson.

## 4. D1 landed: `comps.FAB` and `Screen.Floating`

### `comps/fab.go`

A `comps.Button` in a circle and nothing more: Variant × Emphasis, disabled
dimming and the Style order are Button's, so a FAB and a Button on one screen
agree about primary.

- Icon-only: a disc, `Width == Height` (56pt; 40pt for `FABSmall`),
  `BorderRadius` half, `Padding(0)` (the theme's Button base carries a
  horizontal inset that would push a fixed-width disc's glyph off centre),
  `FontSize` = 24/56 of the diameter.
- `Label` set: the extended pill, same height, `PaddingHorizontal(Spacing.MD)`,
  width from content, label = `Icon + "  " + Label`. **No FontSize override**
  — see §6.
- `core.Shadow(6)`, above a Card's 2. Every target reads `Style.Shadow`.
- Sizes are fixed points, not spacing steps: a FAB is a touch target first and
  56/40 sit on no 4/8/16/24/32 scale.
- Icon-only needs `AccessibilityLabel`; the widget invents nothing to say.
- Registered in `internal/apidoc/packages.go` under "Screens & structure"
  (the generator refuses a file no topic lists).

### `Screen.Floating core.View`

The decision the plan settles: the slot is on `Screen`, not the FAB, because
a ZStack sizes to its largest layer and centres every layer that says
nothing — a FAB wrapping its own screen would have to know the screen's
size; the screen already fills the safe area.

```
SafeArea
  ├─ ZStack FlexGrow(1)
  │    ├─ Scroll / Column   Width 100%, Height 100%   ← the content
  │    └─ Box               StackAlignBottomEnd, Margin(Spacing.LG)
  │         └─ Floating
  └─ Footer
```

- The base layer states percent size or the stack would hug and centre it.
- The *stack* grows, not the column: a layer has no main axis to grow along.
- The placement and margin ride on a `core.Box` because a caller's View
  cannot be handed props, and Box carries no theme base.
- Footer stays below the stack: the FAB floats above a BottomBar, not on it.
- Nil builds no stack; `TestScreenZeroValueIsExactlySafeAreaColumn` still holds.

### Tests, docs, lesson

- `comps/fab_test.go` (5 tests) and four new `Screen` tests in
  `comps/screen_test.go` (`floatingStackOf` helper).
- `docs/components.md`: a `## FAB` section after BottomBar; a Floating row,
  tree line and paragraph in the Screen section. `docs/api/` regenerated.
- Tutorial lesson 4.22 "The floating action button" (`lessonFAB`, appended so
  deep-linked numbers do not move). A lesson is itself a scrolling screen, so
  the demo floats the FAB over a fixed-height ZStack — the same shape
  `Screen.Floating` builds — and prints the Screen form as code. Driving
  test `TestFABLessonFloatsTheDiscAndAddsNotes` (the tutorial test's
  `nodeStyle` mirrors only Width among the sizes, so the placement is left to
  comps' own tests).

## 5. Fallout of one more lesson

- "0 of 60 lessons opened" → 61 in README.md, docs/tutorial-interactive.md,
  `examples/tutorial/screenshot_test.go`, `internal/shotclaims/*`.
- `docs/images/tutorial-contents.png` re-taken with
  `wasm/shots/shoot.sh tutorial-contents`.
- `wasm/verify`'s file census counts *untracked* Go files too, so the five
  "554 tracked Go files" sentences in `repowalks_test.go` and
  `timings_test.go` now say 556.

## 6. What a real render caught

A temporary shot script (not committed; `GRMOB_SHOTS_OUT` pointed at the
scratchpad) rendered lesson 4.22 in Chrome. The disc floated bottom-end over
the filled base layer with its shadow, as the tests said. What the tests
could not say: the extended pill's word was set at the disc's 24pt glyph size
and read oversized. The `FontSize` now applies to the disc alone; the pill
keeps the theme's button text, asserted by comparing against a plain
`Button`'s FontSize (the theme base carries one, so "zero" was the wrong
expectation).

## 7. Not verified

The Floating stack on the two natives. `ios/verify` measures a stack's size
and anchors and `mobile/verify` pins the Compose overlay arm, so the two facts
this depends on are pinned separately; their composition on a full-height
screen has not been run on a device. If the base layer hugs its content there,
the fix is in the layer's props, not the renderers.

## 8. Next

D2 `QRCode` per the plan's order: decide vendor (`skip2/go-qrcode`, MIT) vs.
own encoder, then a `Canvas` of `Rect`s, `RoleImage`, dark-on-surface only.
