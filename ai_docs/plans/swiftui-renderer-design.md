# SwiftUI-Based iOS Renderer (Attack-Order Step 5)

*Design date: 2026-08-29. Companion to `compose-renderer-design.md` — the same
architecture ported one-to-one; this doc records only the iOS-specific
decisions and deltas.*

## The design, unchanged

Patches are applied to a **Swift data tree, never to views**. `TreeStore`
resolves positional paths against the live tree at apply time; SwiftUI views
read the tree and SwiftUI's own diffing handles view identity, recycling,
animation, and accessibility downstream. Same bridge (`mobile/bridge.go`),
same two delivery paths, same arrival-order contract.

```
Go side                     bridge                Swift side
───────                     ──────                ──────────
core.Node tree ─ diff ─▶ patch JSON ─▶ TreeStore.applyPatches
                                          │ mutates @Observable state
                                          ▼
                                    GrMobNode tree ─ read ─▶ SwiftUI Views
                                                                 │ (SwiftUI diffing:
                                                                 ▼  identity, recycling, a11y)
                                                              iOS UI
```

## Platform translations (Compose concept → SwiftUI)

| Compose (Android runtime)             | SwiftUI (iOS runtime)                          |
|---------------------------------------|------------------------------------------------|
| snapshot state (`mutableStateOf`)     | `@Observable` per-property tracking (iOS 17 floor) |
| `SnapshotStateList` children          | plain `[GrMobNode]` var on an `@Observable` class |
| `key(child.key or index)`             | ForEach id = explicit key, else **object identity** (see below) |
| `CompositionLocal` runtime            | `@Environment(\.grMobRuntime)`               |
| `Modifier` chain (outermost-first)    | modifier chain (innermost-first) — same CSS order, reversed in source |
| `Modifier.weight` (FlexGrow)          | infinity frame on the main axis (`GrMobGrow`), parent-computed |
| `Arrangement` (justify-content, gap)  | stack `spacing` + flexible-Spacer emulation    |
| single main `Handler` funnel          | `DispatchQueue.main.async` funnel (FIFO — not `Task`, which orders nothing) |
| events single-thread executor         | serial `DispatchQueue("grmob-events")`       |
| material3 Button/Checkbox/TabRow      | styled `Button`, `Toggle` (switch — iOS has no checkbox), hand-rolled top tab bar |
| Coil `AsyncImage`                     | built-in `AsyncImage`                          |
| `Dialog`                              | `.sheet` (the iOS idiom for Modal)             |

**Identity via object identity**: Compose needed positional `key()`; SwiftUI
gets something better for free. `update-props`/`update-style` mutate a node
in place (same instance → identity and sibling view state survive), while
`replace` swaps in a fresh instance (new `ObjectIdentifier` → SwiftUI resets
the subtree) — exactly the Go reconciler's semantics, with no index math.

**Flex emulation caveats** (SwiftUI stacks have no justify-content): CSS
values are emulated with flexible Spacers, which is faithful whenever the
stack has free main-axis space; `space-around` approximates as
`space-evenly`; multiple FlexGrow siblings split space equally regardless of
weights (proportional weights need a custom `Layout` — future work).

**Controlled inputs**: same compromise as Android, SwiftUI mechanics — the
binding serves a local `@State` buffer while `@FocusState` reports focus
(seeded from the Go value when focus arrives), and serves the Go value
directly when unfocused.

## File map

- `ios/GrMob/Runtime/` — app-agnostic runtime, file-for-file mirror of
  `android/.../runtime/`: `GrMobNode.swift`, `GrMobStyle.swift`
  (+ `grMobBox` in CSS box-model order), `TreeStore.swift`,
  `GrMobRuntime.swift` (+ `GrMobBridge` protocol), `Renderer.swift`.
- `ios/GrMob/App/` — shell: `GrMobApp.swift`, `GomobileBridge.swift` over
  the gomobile-generated C functions (`MobileRenderInitial`,
  `MobileTrigger*Callback`, `MobileSetListener` /
  `MobilePatchListenerProtocol` — names verified against gobind's actual
  generated header, Go `int` ↔ `long` ↔ Swift `Int`).
- `ios/build.sh` — `gomobile bind -target=ios,iossimulator` →
  `ios/Frameworks/GrMob.xcframework` (same package pair as Android:
  `./mobile ./examples/mobileapp`; the gobind linkage trap and its
  `mobileapp.AppName` fix apply identically).
- `ios/project.yml` — xcodegen spec (source of truth; the generated
  `GrMobApp.xcodeproj` is untracked). iOS 17 deployment floor.
- `ios/verify/` — data-layer conformance harness, see below.

## Verification without Xcode (this machine has only the CLT)

Two layers, both green:

1. **Typecheck**: the whole runtime + shell compiles under
   `swiftc -typecheck -target arm64-apple-macos14.0` (SwiftUI is
   cross-platform; macOS 14 ≙ iOS 17 API-wise; the shell checked against a
   stub reproducing gobind's generated header verbatim).
2. **Transcript replay** (`ios/verify/run.sh`): `gen.go` drives the real
   bridge + demo app through every event kind, recording each patch batch in
   arrival order plus the final full tree; a compiled macOS harness replays
   the batches through the real `GrMobNode`/`GrMobStyle`/`TreeStore`
   files and deep-compares trees. This is the iOS analog of
   `examples/mobileapp/app_test.go` and the fast feedback loop for store or
   parser changes.

What still needs full Xcode: running `ios/build.sh` (gomobile drives
xcodebuild), building the app, and exercising it in a simulator — the
on-device verification pass that step 4 got on Android.

## Known limits / next

- Everything renderer-level from the Android list applies: no list
  virtualization (`Scroll` is an eager VStack in a ScrollView), keyed
  reorders still arrive as `replace`, CameraView is a styled stub.
- Proportional FlexGrow and exact `space-around` need a custom `Layout`.
- `Modal` uses `.sheet` with medium/large detents; a centered-dialog variant
  (Android parity) would be an overlay ZStack if ever needed.
- Line height maps to `lineSpacing` (height − font size) — an approximation.
- On-device pass pending Xcode: install Xcode, `sudo xcode-select -s
  /Applications/Xcode.app`, run `ios/build.sh`, `cd ios && xcodegen
  generate && open GrMobApp.xcodeproj`, run on an iOS 17+ simulator.
