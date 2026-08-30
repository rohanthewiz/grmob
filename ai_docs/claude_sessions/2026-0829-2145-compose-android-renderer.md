# Session: Compose-Based Android Renderer (Attack-Order Step 4)

- **Session ID**: `a808f2b6-5c5b-4849-8933-19bce08327bb`
- **Session link**: https://claude.ai/code/session_01WYkTemAUD23zXtGj8SVGjW
- **Date**: 2026-08-29
- **Branch**: master
- **Previous session**: `2026-0829-2108-push-channel.md` (steps 1–3)

## Goal

Step 4 of the attack order in `ai_docs/plans/grmob-mobile-feasibility-analysis.md`:
replace the View-based `PatchRenderer.kt` with a Jetpack Compose renderer. Full design
rationale lives in `ai_docs/plans/compose-renderer-design.md` (written this session).

**Outcome: complete and verified live on an API 36 emulator.** Tabs, button events,
IME text input, checkbox toggles, and the interval-driven push channel all work
on-device through the new renderer.

## The design in one paragraph

Patches are applied to a **Kotlin data tree backed by Compose snapshot state, never to
views**. `TreeStore` resolves positional paths against the live tree at apply time (no
stale path→View cache); `update-props` mutates one node's `mutableStateOf` and
recomposes only its readers; structural patches mutate a `SnapshotStateList` and
recompose only the parent's keyed children loop. Compose's reconciler handles view
identity, recycling, and a11y downstream — the strategic call from the feasibility
analysis. Threading: bridge `Trigger*` calls run on a single-thread executor; both
delivery paths (sync event returns, async Go pushes) funnel into one main-thread
Handler, which *is* the arrival-order contract.

## New files

- `android/app/src/main/java/com/grmob/runtime/` — app-agnostic runtime:
  - `GrMobNode.kt` — snapshot-state node tree + JSON parsing (Go-cap keys: `Type`,
    `Props`, `Style`, `Children`; props keys lowercase)
  - `TreeStore.kt` — patch application; non-JSON initial payload logged, not crashed on
  - `GrMobStyle.kt` — Go `Style` → Modifier, CSS box-model order (margin → size →
    shadow → clip → background → border → padding); colors accept `#RRGGBBAA`
  - `Renderer.kt` — all 20 node types; material3 for Button/Checkbox/TabRow,
    foundation elsewhere; FlexGrow → scope `weight` passed down as the child's
    `extra` modifier; controlled inputs locally-owned while focused, Go-owned when not
  - `GrMobRuntime.kt` — `GrMobBridge` interface + threading funnel
- `android/app/src/main/java/com/grmob/app/GomobileBridge.kt` — impl over generated
  `mobile.Mobile` (verified 1:1 against the gomobile sources jar; Go int → Java long)
- `android/app/src/main/AndroidManifest.xml` (was missing), `android/gradle.properties`
  (`android.useAndroidX=true`), Compose gradle config (Kotlin 1.9.24 ↔ compose-compiler
  1.5.14, BOM 2024.06.00, material3, coil), `android/.gitignore`
- `examples/mobileapp/` — demo app bound into the AAR (counter + interval clock +
  input/checkbox form in a TabView) with `app_test.go` replaying the exact Kotlin call
  sequence
- Deleted: `PatchRenderer.kt`, old JNI `GrMobBridge.kt`; `MainActivity` rewritten;
  `build.sh` now binds `./mobile ./examples/mobileapp` and defaults ANDROID_HOME/NDK

## Bugs found & fixed (Go side)

1. **gobind linkage trap** — the on-device startup crash (SIGABRT, nil manager): a
   bound package with zero *bindable* exported symbols gets a Java class but is never
   imported by the gobind glue, so its `init()` (the `mobile.Register` call!) never
   runs. `mobileapp.AppName()` exists solely to force linkage; comment explains.
   Diagnosed by temporarily wrapping `RenderInitial` in recover → returning the Go
   stack to Kotlin (gomobile's logcat copier loses panic output to the abort race).
2. **`hooks.UseInterval`/`UseTimeout` cross-wired state slots** — they did a bare
   `Cursor++` without appending a slot, so every later `NewState` in the pass landed
   one index short (checkbox bool in the input's string slot → type-assert panic).
   Now reserve through `core.NewState`.
3. **Int callbacks (TabView) outside the render-pass ID system** — counter never
   reset (ID churn every render), never purged, and `mobile.TriggerIntCallback`
   missing. Moved registry into `event.go`; bridge function added.
4. **State-slot data race** (step-3 caveat) — slot get/set/append and `Reset`'s slot
   scan now guarded by `ctx.lock` (unlock before notify: RequestRender → MarkDirty
   re-takes it). New `TestConcurrentStateWritesDoNotRace`; found the `Reset` race.
5. `htmlout` `%d`-on-float vet errors (pre-existing) fixed.

## Toolchain (see also memory: android-toolchain)

- `go.mod`: added `golang.org/x/mobile` pinned at `188f512ec823` (Oct 2025); Go floor
  1.23 → 1.24. **Do not `go get x/mobile@latest`** — it force-bumps the floor to 1.26.
  gomobile/gobind installed at the same commit.
- No Gradle/Studio on machine: Gradle 8.5 zip in session scratchpad (ephemeral —
  re-download to build again), JDK 17 via Homebrew, SDK `~/Library/Android/sdk`,
  NDK 28.2. `android/build.sh` → AAR; `gradle assembleDebug` in `android/` → APK.
- `android/.gradle` junk untracked (`git rm -r --cached`), `.gitignore` added
  (`.gradle/`, `app/build/`, `app/libs/`, `local.properties`).

## Test status at wrap

`go test -race ./core/ ./render/ ./reconcile/ ./mobile/ ./examples/mobileapp/` — all
pass (38 tests). Vet clean incl. hooks + htmlout. WASM target compiles. On-device:
demo APK exercised end-to-end (void/text/bool/int events + push channel), no crashes.

## Known caveats / open items

- Lists are non-virtualized (`Scroll` = eager Column); LazyColumn node + windowing
  protocol is the next renderer milestone.
- Keyed reorders still `replace` (positional identity); Compose `key()` already in
  place to benefit when move patches land.
- `hooks.UseEffect` still uses a process-global never-resetting index — same bug class
  as the interval one; untouched.
- CameraView is a styled stub (needs a CameraX pass). Modal backdrop prop unused
  (Dialog scrims itself).
- Pre-existing breakage untouched: `examples/runtime/main.go` (`core.NewRuntime`
  undefined).

## Next step

**Step 5: iOS via SwiftUI** — same design one-to-one: `TreeStore` ↔ ObservableObject
tree, `boxModifier` ↔ ViewModifiers, same `mobile` bind surface (gomobile
xcframework). The gobind linkage trap applies there too.
