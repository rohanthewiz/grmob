# Session: Animations Pass — Declare in Go, Drive Natively

- **Session ID**: `c83e0672-cbd6-4277-b997-27cc36311e32`
- **Session link**: https://claude.ai/code/session_01WYkTemAUD23zXtGj8SVGjW
- **Date**: 2026-08-30
- **Branch**: master
- **Previous session**: `2026-0830-0446-gap5-renderer-surface.md`

## Goal

The feasibility analysis's animation strategy: "declare animations in Go but
drive them natively" — no per-frame patches over the bridge, ever. Scope:
implicit transitions (property changes + list placement), riding the
long-dormant `Style.Transition` field. The keyframe `Style.Animation` field
("bounce 2s infinite") stays web-only/unmapped.

**Outcome: complete, verified to the pixel** — full Go suite green, iOS
XCUITest 2/2 with animations active, and emulator screenshot sampling proving
the Compose fade runs natively with a pure alpha ramp.

## Design

One declaration, four backends:

- `core.Transition(durationMs, easing)` (`core/animation.go`, new) writes the
  canonical `"<ms>ms <easing>"` into `Style.Transition`. `Easing` constants
  are the CSS keywords (linear/ease/ease-in/ease-out/ease-in-out); every
  renderer maps them to the CSS spec's cubic-bezier control points, so the
  curve is identical everywhere. Non-positive duration clears the field.
- Semantics: a Transition on a node animates that node's own property
  changes (update-style patches); a Transition on a **List** animates row
  *placement* (reorder/insert/remove). Two declarations, two scopes — same
  contract on both platforms. `replace` patches swap node identity and
  therefore snap, matching the reconciler's intent.
- Parsers tolerate the CSS longhand ("all 0.3s ease") by skipping unknown
  tokens.

## What changed

- **Android** (`GrMobStyle.kt`, `Renderer.kt`): parsed `transitionMs` +
  `transitionEasing` fields with a `transitionTween<T>()` helper;
  `animateContentSize(tween)` in boxModifier (inserted before the dimension
  modifiers so explicit and content-driven size changes both animate);
  `animatedStyle()` composable animates the background via a hand-driven
  `Animatable` (see gotcha below); GrMobList rows get
  `animateItemPlacement` when the List declares a Transition (foundation
  1.6 under BOM 2024.06 — the 1.7 `animateItem` API isn't available;
  `LazyItemScope`-only, so it's built inside the item lambda).
- **iOS** (`GrMobStyle.swift`, `Renderer.swift`): `transition` field +
  `swiftUIAnimation` computed parse (`.timingCurve` for the CSS beziers);
  `grMobBox` ends with `.animation(anim, value: s)` scoped to the node's
  own style; GrMobList attaches `.animation(anim, value: rows.map(\.viewID))`
  for placement.
- **Web**: `htmlout` styleAttr and `wasm/grmob-runtime.js` emit
  `transition: all <decl>` — the canonical form is valid CSS as-is.
- **Demo** (`examples/mobileapp`): feed rows declare
  `Transition(250, EaseInOut)` so the selection highlight fades.
- Both style files' "intentionally ignored fields" headers updated:
  Transition is now mapped; Animation/Position/ZIndex remain ignored.

## Two real bugs found and fixed

1. **Swift compiler crash** (signal 6, "Possible non-terminating conformance
   substitution", `substOpaqueTypesWithUnderlyingTypes`): one more
   `@ViewBuilder` conditional stacked on grMobBox's opaque-type tower blew
   the compiler's substitution limit. Fix: `grMobTransition` is a plain
   unconditional modifier — `.animation(nil, value:)` is already the
   no-animation case. Comment in code warns the next conditional will
   reintroduce the crash.
2. **Fade-through-gray artifact, caught by pixel sampling**: the first cut
   animated against `Color.Transparent` — transparent BLACK — and Compose
   interpolates non-premultiplied color (Core Animation premultiplies, so
   SwiftUI doesn't have this problem). Measured mid-fade: (169,171,175)
   between white and #E8F0FE. Fix in `animatedStyle`: an appearing color
   first `snapTo(target.copy(alpha = 0))` (fixes hue invisibly) then
   animates in; a disappearing one animates to *its own hue* at alpha 0.

## Verification

- Go: build, vet, `go test -race ./...`, wasm build — green. New
  `core/animation_test.go` (canonical form, clearing, node plumbing).
- iOS: `ios/verify/run.sh` conforms (9 batches — the harness compiles
  GrMobStyle.swift, so it also caught the compiler crash); xcframework
  rebuilt; XCUITest **2/2** on the iPhone 17 Pro simulator with the feed
  transition active.
- Android: AAR + APK rebuilt (scratchpad Gradle 8.5, `ANDROID_HOME` env),
  driven on the emulator with frame-time screenshots via
  `adb shell "input tap ...; sleep 0.04; screencap ..."` and a hand-rolled
  PNG pixel reader (no Pillow on the machine; script at scratchpad
  pixel.py pattern — unfilters rows up to y). Mid-fade samples (234,241,253)
  and (236,242,253): monotonic between #E8F0FE and white, no gray, settling
  at exactly (232,240,254).

## Gotchas learned

- adb tap→capture timing is bimodal: chained `input tap; screencap` lands
  ~0ms into the fade (patch not yet applied) or, with `sleep 0.12`, ~200ms
  in (nearly settled). `sleep 0.04`–`0.07` hits mid-window for a 250ms
  transition. The tapped row's sample is contaminated by the ripple overlay
  (~(173,174,176) gray) — sample the *deselected* row's fade-out instead.
- Compose ripple gray and the transparent-black artifact look similar in
  samples; distinguish by whether the whole frame or only the touched row
  shows it.
- Emulator background task exits 134 (SIGABRT) after `adb emu kill` —
  normal shutdown, not a failure (second occurrence; expected noise).

## Next step

Renderer long tail remaining: keyboard avoidance, CameraX/AVFoundation
capture for CameraView, proportional multi-grower FlexGrow on iOS, and —
if keyframe animations are ever wanted natively — a design for mapping
`Style.Animation` onto Compose/SwiftUI repeatables.
