# Session: Todo app example, input-submit event, focused-field fix, tutorial

Session ID: `418cae93-4576-45c1-af90-c4e6ede0bb86`
(Claude session link: https://claude.ai/code/session_01GKeRK9N45Cd4xWNZ2YX7ms)
Date: 2026-08-30 (early morning through ~08:00)
Branch: `master` — all work committed; tree clean at wrap.

## What happened, in order

1. **Built `examples/todoapp`** (`521abf5`) — a complete todo app on the
   mobile bridge: controlled input + Add, All/Active/Done filters, virtualized
   keyed `List` rows (checkbox toggle, per-row ✕ delete), remaining count,
   conditional "Clear completed", a11y labels/hints, native transitions.
   Bridge-level lifecycle test (`app_test.go`) drives the whole journey
   headlessly via `mobile.Trigger*Callback`.
2. **Ran it on the iOS simulator** — parameterized `ios/build.sh` (optional
   first arg = app package, default `./examples/mobileapp`), bound todoapp,
   xcodegen + xcodebuild, installed/launched on the booted iPhone 17 Pro as
   `com.grmob.demo`. First screenshot exposed an invisible white label on
   the selected filter chip → fixed by overriding text+background together
   (`13f242f`).
3. **User bug report: input not cleared after Add.** Root cause was in both
   native renderers, not the app: Go emitted the `"value":""` patch, but the
   focused text field's local buffer unconditionally swallowed upstream
   writes. Fix (`68155fc`): queue every value the field sends upstream; an
   upstream change matching a queued entry is an echo (ignored), one matching
   nothing is a deliberate Go rewrite and lands mid-focus. Applied to
   `ios/GrMob/Runtime/Renderer.swift` and `android/.../Renderer.kt`.
   Pinned by new `ios/GrMobUITests/TodoAppUITests.swift` (real typing +
   tap; passed on simulator).
4. **Ran the Android fix on the emulator** — parameterized `android/build.sh`
   the same way (`a7ef8cc`), built APK with cached Gradle 8.14, drove the
   flow via adb (tap/`input text`/uiautomator dump). Verified: rows added,
   placeholder back while keyboard still up. Detour: first Gradle run failed
   silently ("SDK location not found" — Gradle needs `ANDROID_HOME` exported;
   `tail` masked the non-zero exit) and adb installed a stale APK that showed
   the old demo app. Toolchain facts saved to auto-memory.
5. **Enter-to-submit** (`0c94667`) — new `core.InputWithSubmit(value,
   placeholder, onChange, onSubmit, ...styles)`; onSubmit rides the existing
   void-callback channel (no bridge change). iOS: `.onSubmit` +
   `.submitLabel(.done)`; Android: `ImeAction.Done` + `KeyboardActions`.
   Todoapp adopted it; verified at all three levels (Go test, XCUITest with
   `"\n"`, adb keyevent 66).
6. **Destructive-button contrast** (`48bd1d5`) — ✕ and "Clear completed" were
   red-on-theme-blue; now white label on `#B3261E` background (override the
   color pair, not one half). Screenshot-verified on emulator.
7. **In-depth tutorial** (`863755f`) — `docs/tutorial-todo.md`, walking the
   todoapp source concept by concept (mental model + round-trip diagram,
   bindable-package contract, rules of hooks, copy-on-write state, controlled
   inputs/echo-rewrite contract, List/keys/reconciler, theming pitfalls,
   a11y, three-level test pyramid, shipping both platforms). Linked from a
   new README "Tutorial" section.

## Commits this session (oldest first)

- `521abf5` feat: todo app example — full CRUD flow on the mobile bridge
- `13f242f` fix: todoapp chip contrast; make ios/build.sh app package selectable
- `68155fc` fix: land Go-initiated input rewrites in focused text fields
- `a7ef8cc` chore: make android/build.sh app package selectable like ios/build.sh
- `0c94667` feat: submit event for text inputs — Enter commits
- `48bd1d5` fix: todoapp destructive buttons — white glyph on danger red
- `863755f` docs: in-depth todo tutorial — the current framework surface end to end

## Key knowledge worth carrying forward

- **App-package contract:** `init(){ mobile.Register(core.NewContext(), App) }`
  plus one bindable exported symbol (`AppName()`), or gobind drops the package.
- **Both build scripts** now take the app package as arg 1:
  `ios/build.sh ./examples/todoapp`, `android/build.sh ./examples/todoapp`.
  Rebuilding with no arg restores the mobileapp demo (same bundle/app id:
  `com.grmob.demo` on iOS, `com.grmob.app/.MainActivity` on Android).
- **Text-field contract (renderers):** locally-owned while focused; echoes of
  the field's own sends are dropped via a pending-values queue; any other
  upstream value is an authoritative Go rewrite and applies immediately.
  App code just calls `state.Set(...)`.
- **Hooks discipline:** positional slots on Context — no `NewState` in list
  rows; parent owns collection state; bind `NewState` result to a variable
  (pointer receivers); address items by ID, never index.
- **Reconciler:** children matched by index; differing keys ⇒ slot replace
  (no moves) ⇒ keyed rows stay correct but lose transient native state.
- **Testing pyramid:** Go bridge tests (inner loop, sub-second) →
  `TodoAppUITests` on iOS (runs alone via
  `-only-testing:GrMobUITests/TodoAppUITests`; the default GrMobUITests
  suite targets the mobileapp demo and needs the default framework) →
  adb-driven checks on Android (uiautomator dump; placeholder visible ⟺ field
  empty).
- **Android build trap:** export `ANDROID_HOME` for Gradle (build.sh sets it
  only for gomobile); don't pipe gradle through `tail` without checking
  `PIPESTATUS`; confirm APK mtime before `adb install` (stale APKs install
  fine). Gradle 8.14 lives unpacked in `~/.gradle/wrapper/dists/`.
- Both simulators were left running the final todo app build.

## Possible next steps

- Progressive tutorial checkpoints (step-1/step-2 layouts or git tags).
- Android instrumented UI test to mirror TodoAppUITests.
- Persistence for the todo list via the mutation-helper choke point
  (bytdb per user preference).
- `htmlout` has no case for `List`/`TabView`/`Modal` (falls through to div);
  wasm runtime hardcodes its root view — both are gaps if web preview matters.
