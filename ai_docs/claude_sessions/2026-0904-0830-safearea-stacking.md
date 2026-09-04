# Session: SafeArea stacks its children

Session: https://claude.ai/code/session_01Mv1iFwxr7pCAdxCFMKT8Pk
Date: 2026-09-04 (follows "box-overlay-divergence" in this directory, same
session — that doc left SafeArea outstanding with a reason; this is it done)

## Ask

"Now fix SafeArea too."

## Why it was left out of the Box change

The Box fix deferred this deliberately, and the deferral turned out to be
about the right thing: converting SafeArea does not only stop two children
overlapping, it changes how a *single* child is sized — and every screen in
every app has exactly one child under its SafeArea.

## The two halves

- **The overlap.** Android built `Box(...)`, iOS `ZStack(alignment:
  .topLeading)`; both DOM targets stack (SafeArea is in the WASM runtime's
  `STACK_CONTAINERS`). Two children drew on top of each other on device.
  Latent — the corpus has no multi-child SafeArea.
- **The stretch.** This is the half that was live on all 43 nodes. An overlay
  container lets its child size to its own content, so a screen's whole
  content column **hugged its widest child instead of filling the screen**,
  while `align-items: stretch` fills it on the web. A screen whose column
  carries a background would not have spanned the width.

The second half is the same correction `isColumnStretch` already documents for
Column — "an Input in a Column runs the full width of the screen in the
browser, and on a phone it used to hug its placeholder" — so this lands on an
established precedent in this repo rather than inventing one. That is what
resolved the hesitation recorded in the Box doc: the rendering change is a
correction toward the reference, not a coin flip.

## Keeping the chrome

SafeArea keeps its own dispatch arm rather than joining Box on Column's. It
carries three things a Column has no business knowing about:

- the window insets (`WindowInsets.safeDrawing.exclude(WindowInsets.ime)` —
  the IME is deliberately excluded, see the arm's own comment),
- an edge-to-edge background that must paint *under* the bars,
- on Android, the system-bar icon appearance (`SystemBarIcons`).

Only the stacking underneath moved.

### The ordering is load-bearing

`boxModifier` puts its `extra` argument at the **head** of the chain:

```
extra → semantics → margin → dimensions → shadow/clip → background
      → border → gestures → padding
```

So an inset passed as `extra` would shrink the box *before* it is painted, and
the light band under the status bar that SafeArea's doc comment exists to
prevent would come straight back. The old code got this right by applying the
inset after the whole chain (`style.boxModifier(extra).windowInsetsPadding(…)`).

`GrMobColumn` therefore gained an `outer: Modifier = Modifier` slot appended
*after* `boxModifier`, with that reasoning written beside it. One caller.

```kotlin
"SafeArea" -> {
    style?.background?.let { SystemBarIcons(it) }
    GrMobColumn(
        node, extra,
        outer = Modifier.windowInsetsPadding(
            WindowInsets.safeDrawing.exclude(WindowInsets.ime)
        ),
    )
}
```

On iOS the background still rides outside GrMobColumn's own box, where
`ignoresSafeArea` can carry it under the bars:

```swift
GrMobColumn(node: node, grow: grow)
    .background((node.style?.background ?? Color.clear).ignoresSafeArea())
```

### One incidental change, checked

Routing through GrMobColumn also runs `gestureModifier(node)`, which the old
SafeArea arm did not. Verified inert for the existing nodes: it returns a bare
`Modifier` when neither onClick nor onLongPress is set (Renderer.kt:444), so
all 43 SafeArea nodes get exactly the chain they had plus the stacking change.
Where the props *are* set, a SafeArea is now clickable — which both DOM
targets have always honored through the generic listener path.

## The align-fallback gate

SafeArea joins `alignFallbackAxes` (`htmlout/crossaxis.go`) and the runtime's
`alignFallbackAxisFor`, for the reason Box did one commit earlier: that
table's stated invariant is "exactly the containers the natives read the
fallback for", and GrMobColumn is where that read happens. The gate is now
Column, Card, Box, SafeArea, List.

## A test that failed on its own explanation

The first run of the new pin failed like this:

```
the case "SafeArea": arm builds a ZStack — its children draw on top of each other …
```

— on an arm that no longer builds one. The comment I had just written says "A
vertical stack, not a ZStack", and the check was reading the prose.

`declSource`'s doc comment already flags this hazard from the other side: it
can afford to keep comments because "the substrings held against these regions
are expression fragments (`ifEmpty { s.align }`), which prose does not
accidentally spell." A bare type name is not one of those. So `dispatchArm`
strips `//`-to-end-of-line before the negative check, with that reasoning
recorded — including the one accepted limitation (a `//` inside a string
literal would go too; neither dispatch has one, and the positive half of every
check would fail loudly if a strip ever ate real code).

## Changes

- `android/…/Renderer.kt` — `GrMobColumn` gains `outer`; the SafeArea arm
  routes through it. The old `Box(…) { RenderChildren(node) }` is gone.
- `ios/…/Renderer.swift` — the SafeArea `ZStack` becomes `GrMobColumn`, with
  the edge-to-edge background still applied outside it.
- `htmlout/crossaxis.go`, `wasm/grmob-runtime.js` — `SafeArea: "column"` in
  the align-fallback gate.
- `htmlout/crossaxis_test.go` — expected set grown to five, same
  visible-decision mechanism as last time.
- `core/layout.go`, `docs/concepts/views.md` — SafeArea documented as a Column
  below the inset, on every target.
- `mobile/verify/stacking_test.go` (new, replaces `box_test.go`) — one test
  over Box *and* SafeArea, since it is one rule. It pins routing **to**
  GrMobColumn, not merely the absence of an overlay: routing is what supplies
  the stretch, and "is not a ZStack" would pass for a hand-rolled second stack
  that no alignment or gap check reaches. Plus `dispatchArm` /
  `stripLineComments`, and generic next-arm regexps so reordering the dispatch
  cannot silently widen what a check reads.
- `mobile/verify/gap_test.go` — `readNative`'s comment notes its second caller.

Confirmed to fail against the pre-fix sources via
`git stash push -- ios android` (only the SafeArea rows failed, since Box was
already committed — which is itself a check that the two halves are pinned
independently).

## Still outstanding

- **`htmlout` renders every stack container as block flow.** Unchanged from
  the two previous docs. The WASM runtime plants `display:flex` on every
  `STACK_CONTAINERS` type whether or not the Style asks; htmlout has no such
  default, so a Row or Column with no flex props runs its `<span>` children
  inline. Affects SafeArea no more than the rest — and note that both
  containers fixed here are now *more* consistent with the WASM runtime than
  htmlout is.

  **Done next, same session** — see
  `2026-0904-0903-block-flow-and-syntax-highlighting.md`. The axis and the
  membership turned out to be one fact, so the fix is a sixth shared table
  (`htmlout/stack.go`) rather than a flag, and the runtime's bare Set became a
  pinnable lookup in the process.
- **CameraView stays an overlay** on both natives, which is what it is for.
  Modal is one on Android for the same reason. Both are why the new pin checks
  the dispatch arm rather than the file.

## Verification

`go test ./...`, `gofmt`, `go vet`, `sh wasm/verify/run.sh`,
`sh ios/verify/run.sh`, and `:app:compileDebugKotlin` (re-run with
`--rerun-tasks`, no warnings) — all clean.
