# Session: Gap 5 — Renderer Surface Breadth (List, Gestures, Accessibility)

- **Session ID**: `c83e0672-cbd6-4277-b997-27cc36311e32`
- **Session link**: https://claude.ai/code/session_01WYkTemAUD23zXtGj8SVGjW
- **Date**: 2026-08-30
- **Branch**: master
- **Previous session**: `2026-0829-2355-hooks-per-context-lifecycle.md`

## Goal

Gap 5's long tail from `ai_docs/plans/grmob-mobile-feasibility-analysis.md`:
list virtualization, gestures, and accessibility on the Compose/SwiftUI
renderers (images were already covered by both).

**Outcome: complete, verified live on both platforms** — iOS XCUITest suite
2/2 on the iPhone 17 Pro simulator, and an adb-driven pass on the Android
emulator (Medium_Phone_API_36.1) confirming tap/long-press round trips,
TalkBack descriptions, and real LazyColumn virtualization + recycling.

## What was built

### Go core

- **`core.List`** (`core/list.go`, new): virtualized sibling of Column,
  node type `"List"`, Column's theme base. Children get stable identity via
  the existing `core.Keyed`; unkeyed rows fall back to positional identity.
- **Container behavior props fixed + `OnLongPress`**: `core/layout.go` now
  routes Row/Column/Card/Box (and List) through a shared `containerNode`
  builder — BehaviorProps apply to the node actually returned. This fixes
  the long-standing quirk (flagged two sessions ago) where Column applied
  them to a throwaway props map and Row/Box/Card ignored them entirely.
  Ordering contract: container callbacks register in argument order BEFORE
  any child renders, so container IDs precede children's in every pass.
  `core.OnLongPress` added beside OnClick/OnTouch (`props["onLongPress"]`).
- **Accessibility rides `Style`**: `AccessibilityLabel` / `AccessibilityHint`
  / `AccessibilityHidden` fields plus same-named StyleProp helpers. On Style
  (not Props) so every builder that takes StyleProps supports them with zero
  signature changes, and changes travel as ordinary update-style patches.

### Android renderer (Compose)

- `"List"` → `LazyColumn` with `itemsIndexed(key = row key, contentType =
  row type)`. Go's `For` wraps rows in a Fragment node; `flattenFragments`
  inlines Fragment/Theme wrappers so each row is an individually recycled
  lazy item.
- Gestures: `gestureModifier(node)` builds `combinedClickable` (click +
  optional long-click, TalkBack actions included) for nodes that don't draw
  their own control. `boxModifier` gained a `gestures` parameter inserted
  after background/border and before padding — touch target = visible box,
  padding included, margin excluded, ripple clipped to the shape. Wired for
  Text, Row, Column/Card, Box, Image, List.
- A11y in `boxModifier`: label+hint fold into one `contentDescription`
  (TalkBack has no hint slot); hidden → `clearAndSetSemantics { }` (prunes
  the subtree). Image prefers the style label over the legacy `alt` prop and
  strips it from the style handed to boxModifier so the description isn't
  double-announced.

### iOS renderer (SwiftUI)

- `"List"` → `ScrollView { LazyVStack }` (deliberately NOT SwiftUI `List`,
  which drags in UITableView chrome). Same Fragment flattening; row identity
  is the existing `viewID` (key else object identity).
- Gestures live in `grMobBox(onTap:onLongPress:)` at the same box layer as
  Android, via a `GrMobGestures` ViewModifier: `contentShape(Rectangle())`
  + tap/long-press gestures + VoiceOver activate action and a named
  "Long press" action. **Harness constraint solved**: `ios/verify` compiles
  GrMobStyle/Node/TreeStore *without* Renderer.swift or the runtime, so
  the modifier dispatches through a new runtime-free environment closure
  (`grMobDispatch: (String) -> Void`) that GrMobRoot fills from the live
  runtime — an EnvironmentKey referencing GrMobRuntime would have broken
  the harness build.
- A11y in `grMobBox`: hidden wins; a non-empty label applies
  `accessibilityElement(children: .combine)` + label + hint — one swipe stop
  per labeled row. Image `alt` fallback only applies when no style label is
  set (`grMobAltLabel`; the old unconditional modifier would override with
  an empty string).

### Demo app + transcripts

- `examples/mobileapp`: third tab **Feed** — status line, a decorative
  divider with `AccessibilityHidden()`, and a `List(FlexGrow(1))` of 30
  `Keyed` rows; tap selects (row restyles + label gains ", selected"),
  long-press stars (title gains " ★"). New bridge-level test
  `TestFeedTabListGestures` (restores tab 0 so test order stays irrelevant).
- `ios/verify/gen.go`: transcript now switches to Feed and drives a row's
  onClick and onLongPress (9 patch batches, up from 6).
- `ios/GrMobUITests`: new `testFeedListGesturesAndVirtualization` — rows
  addressed BY accessibility label + button trait (lookup itself proves the
  a11y wiring), virtualization probed via `isHittable`, long-press via
  `press(forDuration:)`, then scroll until row 30 is hittable.

## Verification

- Go: `go build ./...`, `go vet`, `go test -race ./...` green; wasm builds.
  JS runtime and htmlout both default unknown types to `div`, so `List`
  degrades gracefully there.
- iOS data-layer conformance (`ios/verify/run.sh`): 9 batches, tree matches.
- iOS simulator: full XCUITest suite **2/2** after xcframework rebuild
  (`ios/build.sh` → `xcodegen generate` → `xcodebuild test`).
- Android: AAR rebuilt (`android/build.sh`), Kotlin compiles with the
  scratchpad Gradle 8.5 (needs `ANDROID_HOME=~/Library/Android/sdk` env —
  no local.properties in repo), APK driven on the emulator: selection,
  starring, content-descs, and virtualization (Article 30 absent at top,
  Article 1 recycled away after scrolling to the tail) all confirmed.

## Gotchas learned

- **XCUITest's accessibility snapshot realizes offscreen LazyVStack rows**,
  so `.exists` cannot probe virtualization on iOS — `isHittable`
  (viewport-dependent) can. Android's uiautomator dump, by contrast, shows
  only composed rows, so absence-from-hierarchy IS the virtualization probe
  there.
- A row combined into one accessibility element (`children: .combine`)
  hides its inner `Text`s from XCUITest queries — assert derived state via
  the status line, not the row's inner text.
- Compose Rows/LazyColumns hug content width by default: adb taps/swipes
  must land inside actual row bounds (x≈180, not screen center). The demo's
  feed rows would want `Width("100%")` for real full-bleed touch targets —
  left as-is to avoid another bind cycle; do it whenever the demo is next
  rebuilt anyway.
- The emulator background task reports exit 134 (SIGABRT) after
  `adb emu kill` — that's normal shutdown, not a failure.

## Next step

Gap 5's named items are done. Remaining renderer long tail beyond the
analysis: animations, keyboard avoidance, CameraX/AVFoundation capture for
`CameraView`, and proportional multi-grower FlexGrow on iOS (currently
equal-split; needs a custom Layout).
