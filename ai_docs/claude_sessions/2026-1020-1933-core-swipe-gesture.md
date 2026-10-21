# core.OnSwipe: a swipe gesture on every host (N-112)

Session: `63d90e29-12aa-4968-ad4c-82996c004fdf`
**Date:** 2026-10-20 19:33 · **Branch:** master (57fbcb1 → 5e93a45, plus this commit)

## Goal

The user pasted N-112 from the next-list: core had no swipe or drag gesture.
`comps.Gallery` could step only by its tap zones and buttons, and a horizontal
`core.Scroll` could not stand in for a swipe, because a scroll reports no
position. A swipe needed a new core prop wired on every host.

## The API: `core/swipe.go`

```go
core.Box(core.OnSwipe(func(d core.SwipeDirection) {
    if d == core.SwipeLeading { next() } else { prev() }
}), card)
```

- `OnSwipe` (prop `onSwipe`) is horizontal and reports `SwipeLeading` or
  `SwipeTrailing`. `OnVerticalSwipe` (prop `onVerticalSwipe`) reports
  `SwipeUp` or `SwipeDown`. Both are text callbacks whose payload is the
  direction's value. Any other payload is dropped.
- **Logical, not physical.** Leading means the finger moved toward the
  leading edge: left in LTR, right in RTL. The host resolves it, as canvas
  taps are reflected on the host, because Go does not know the layout
  direction. A carousel then needs no RTL check of its own.
- **One rule on every host, at release:** at least `SwipeMinDistance` (40
  layout units) along the axis, and strictly further along it than across. A
  tie is neither axis. There is no velocity or fling, because each platform's
  fling detector differs and porting one would give three different gestures.
- **Discrete.** The gesture fires once at release and sends no per-move
  deltas. Deltas would cost a bridge round trip and a render per pointer
  move, for content-tracking that core cannot animate anyway.
- **One axis claimed per prop**, so a swipeable card in a vertical Scroll
  still scrolls. A node carrying both props claims both.
- A swipe that fires is never also a tap or a long press on a child.
- **Disabled drops it in Go.** The closure checks `n.Style.Disabled` at
  dispatch time, so a `Disabled` written after `OnSwipe` still counts.
- A nil handler writes no prop, so it claims no axis.

## Hosts

- **Web** (`wasm/grmob-runtime.js`, "Swipes"): `applySwipe(el, props)` runs
  on both the create path and the update-props path, and is total, like
  `applyPressKeepsFocus`. The generic `on*` arm skips both keys.
  - **Tracking:** one pointer from down to up, primary button only. A second
    pointer is ignored. Past a 10px slop (`SWIPE_SLOP`) the element takes
    pointer capture, so a release outside the card still lands.
  - **Axes:** `touch-action` is `pan-y pinch-zoom`, `pan-x pinch-zoom`, or
    `pinch-zoom` for both. It is only cleared if the runtime wrote it.
  - **The click after a swipe:** it is swallowed for the swiping element and
    its descendants (`swipeClaimsClick`, checked in `eventQualifies` and in
    `attachShapeTap`), and the guard clears on a 0ms timer.
  - **Native drag:** `dragstart` is prevented, so a drag that starts on an
    `<img>` does not become a native drag-and-drop.
  - **Long press:** `attachLongPress` now also disarms once a press travels
    past the slop, which matches the native recognizers.
- **Compose** (`GrMobSwipe.kt`, applied in `RenderNode`'s `mods` before
  `boxModifier`):
  - `awaitFirstDown(requireUnconsumed = false)`, because a child's clickable
    consumes its down and Gallery's stage is covered by clickable zones.
  - The slop detector depends on what is claimed: horizontal, vertical, or
    either.
  - Every move after the slop is consumed, which cancels a child's tap.
  - The callback IDs and the RTL flag are read through `rememberUpdatedState`.
    The detector is keyed only on which axes are claimed.
- **SwiftUI** (`GrMobSwipe` in `Renderer.swift`, applied in `RenderNode` next
  to `GrMobEscapeClaim`):
  - It uses
    `simultaneousGesture(DragGesture(minimumDistance: 10, coordinateSpace: .global), including:)`.
  - `including: .subviews` turns it off without changing the view's type, so
    the node keeps its identity when the prop comes or goes.
  - The axis is decided only at release, so iOS never takes the pan from a
    ScrollView.
- **htmlout** records `data-onswipe` and `data-onverticalswipe`, as it does
  `onShapeTap`.

## comps.Gallery

- The stage `ZStack` carries `OnSwipe` when there is more than one card.
  `SwipeLeading` steps forward and `SwipeTrailing` steps back.
- It sits on the stage rather than the tap-zone Row, for three reasons:
  - it also works while a card is flipped, since a step shows the next
    card's front;
  - on iOS it hit-tests on the pictures, which always draw;
  - the back face's vertical Scroll keeps its vertical drags.
- The widget doc and `docs/components.md` were updated. The old "A swipe is
  not among them" paragraph is replaced.

## Registration and docs

- `internal/apidoc/packages.go`: `swipe.go` joins the `events` topic, whose
  blurb now says "Event props and swipes". `docs/api/` is regenerated.
- `docs/concepts/events.md`: a new `## Swipes` section.
- `wasm/verify/repowalks_test.go` and `timings_test.go`: the Go file count
  goes from 711 to 713.

## Verification

- **Go.** `gofmt -l .` prints nothing, `go vet` is clean on `core`, `comps`
  and `htmlout`, and `go test ./...` passes. New tests:
  - `core/swipe_test.go`: each axis's key and payload filter, both axes on
    one node, the Disabled guard, and a nil handler.
  - `TestGallerySwipeSteps`: trailing at the start is a no-op; leading,
    leading, flip, then trailing gives indexes `[1 2 2 1]` and the front up;
    a one-card gallery claims no swipe.
- **DOM shim.** `wasm/verify/swipe_test.mjs` has 15 tests:
  - LTR and RTL directions, the minimum distance exactly, short and diagonal
    drags, and ties;
  - vertical swipes, touch-action per axis, and both axes on one node;
  - the swallowed click and its one-shot clear, and a tap that stays a tap;
  - pointercancel, a second pointer, and a secondary button;
  - prop gain, renumber and loss through patches, and a style patch keeping
    touch-action;
  - the long-press slop disarm.
- **`wasm/verify/run.sh`** passes in full (exit 0).
- **Real Chrome.** A scratch copy of `wasm/shots/shot.mjs` sent real CDP
  mouse drags to lesson 4.41's Gallery. Nothing was written under `wasm/`.
  - Before: "1 of 5", with `touch-action: pan-y pinch-zoom`.
  - A drag left across the middle zone gave "2 of 5", not flipped.
  - A drag right went back to "1 of 5", not flipped.
  - A 5px press flipped the card (`Details=true`).
- **iOS.** `ios/verify/run.sh` passes. The view layer typechecks and
  survives whole-module optimisation. The app layer was skipped: there is no
  iPhoneOS SDK, only Command Line Tools. No simulator was run.
- **Android.**
  - `./gradlew compileDebugKotlin` cannot build here. Offline, dependencies
    are missing from the cache. Online, `android/app/libs/grmob.aar` is
    missing, and gomobile needs an NDK the SDK lacks.
  - `GrMobSwipe.kt` instead passed kotlinc's resolution and type check
    against the cached Compose 1.7.6 classes jars, with a stub runtime.
    Codegen then stopped only because the Compose compiler plugin is not
    cached.
  - The temporary `android/local.properties` was removed afterwards.
- Native behaviour is raised as N-122.

## Next

Closed: N-112. Declined: None. Raised: N-122.
Deferred: None. Promoted: None. Moved: None. Updated: None.
Full list: `ai_docs/todo/next-list.md`.
