# Session: SwiftUI iOS Renderer (Attack-Order Step 5)

- **Session ID**: `d2391761-a029-4023-95d2-62b3a6c09269`
- **Session link**: https://claude.ai/code/session_01WYkTemAUD23zXtGj8SVGjW
- **Date**: 2026-08-29
- **Branch**: master
- **Previous session**: `2026-0829-2145-compose-android-renderer.md` (step 4)

## Goal

Step 5 of the attack order in `ai_docs/plans/grmob-mobile-feasibility-analysis.md`:
iOS via SwiftUI, porting the Compose renderer design one-to-one. Full rationale in
`ai_docs/plans/swiftui-renderer-design.md` (written this session).

**Outcome: complete and verified to the limit of this machine — no full Xcode
installed (CLT only), so the runtime is typechecked and the data layer is proven
against the real Go reconciler, but the xcframework build + simulator pass is
pending an Xcode install.** No Go framework code changed; the bridge is shared
with Android as designed.

## The port in one paragraph

Same architecture, Swift dialect: patches are applied to an `@Observable`
`GrMobNode` data tree (`@Observable` per-property tracking is the SwiftUI
analog of Compose snapshot state — `update-props` re-evaluates only readers of
that node's props; structural patches only the parent's children loop).
`TreeStore` (`@MainActor`) resolves positional paths against the live tree at
apply time. Threading: `Trigger*` calls run on a serial
`DispatchQueue("grmob-events")`; both delivery paths funnel through
`DispatchQueue.main.async` — deliberately not `Task`, which carries no FIFO
guarantee — which *is* the arrival-order contract. One improvement over the
Android port: ForEach identity is the node *instance* (`ObjectIdentifier`,
explicit key when set), so in-place patches preserve sibling view state and
`replace` (fresh instance) resets the subtree with no index math.

## New files

- `ios/GrMob/Runtime/` — app-agnostic runtime, file-for-file mirror of
  `android/.../runtime/`: `GrMobNode.swift`, `GrMobStyle.swift` (grMobBox
  chain in CSS box-model order — reversed in source since SwiftUI chains read
  innermost-first; strokeBorder for inside borders; compositingGroup before
  shadow), `TreeStore.swift`, `GrMobRuntime.swift` (+ `GrMobBridge`
  protocol, Sendable by contract), `Renderer.swift` (all 20 node types).
- `ios/GrMob/App/` — `GrMobApp.swift` (@main; start() in App.init),
  `GomobileBridge.swift` over the generated C surface.
- `ios/build.sh` — `gomobile bind -target=ios,iossimulator` →
  `ios/Frameworks/GrMob.xcframework`; binds the same `./mobile
  ./examples/mobileapp` pair (the gobind linkage trap + `mobileapp.AppName`
  fix apply identically).
- `ios/project.yml` — xcodegen spec (source of truth; generated
  `GrMobApp.xcodeproj` untracked, iOS 17 floor, `-ObjC` ldflag).
- `ios/verify/` — **data-layer conformance harness**: `gen.go` replays every
  event kind through the real bridge + demo app, recording each patch batch in
  arrival order plus the final full tree; `main.swift` (compiled as a macOS
  executable with the real Node/Style/TreeStore files) mounts, applies, and
  deep-compares. `run.sh` ties it together. This is the iOS analog of
  `examples/mobileapp/app_test.go`.
- `ios/.gitignore` — Frameworks/, generated xcodeproj, build state.

## iOS-specific mappings (vs. Android)

Checkbox → `Toggle` (switch; iOS has no checkbox). TabView → hand-rolled top
tab bar (native TabView wants locally-owned selection; GrMob's is a
controlled Go int). Modal → `.sheet` with medium/large detents. Image →
built-in `AsyncImage`. FlexGrow → infinity frame on the parent's main axis
(`GrMobGrow`); justify-content emulated with flexible Spacers
(`space-around` ≈ `space-evenly`); proportional weights need a custom Layout.
Controlled inputs: local `@State` buffer authoritative while `@FocusState`
reports focus (seeded from Go's value at focus arrival), Go-owned otherwise.

## Verification (all green)

1. `swiftc -typecheck -target arm64-apple-macos14.0` clean on runtime + shell
   (macOS 14 ≙ iOS 17 APIs; SwiftUI is cross-platform).
2. gobind's ObjC header generated locally (`gobind -lang=objc -outdir=...`) to
   confirm the real Swift-visible names — `MobileRenderInitial`,
   `MobileTrigger*Callback` (Go int ↔ long ↔ Int),
   `MobilePatchListenerProtocol` — and the shell typechecked against a stub
   reproducing that header verbatim.
3. `ios/verify/run.sh`: **OK — 6 patch batches applied (void, int×2, text×2,
   bool events); Swift tree matches Go's final render exactly.**
4. `go vet`/`go build` clean except pre-existing `examples/runtime` breakage.

## Toolchain (see memory: ios-toolchain, android-toolchain)

No Xcode.app on the machine; only Command Line Tools (Swift 6.3, macOS SDKs).
xcodegen 2.46 installed via Homebrew; `xcodegen generate` works without Xcode.
gomobile pinned @188f512ec823 (do not bump — forces Go 1.26 floor).

## Next step (needs Xcode)

Install Xcode, `sudo xcode-select -s /Applications/Xcode.app`, then:
`ios/build.sh` → `cd ios && xcodegen generate && open GrMobApp.xcodeproj` →
run on an iOS 17+ simulator and exercise all four event kinds + the interval
push channel (the on-device pass step 4 got on Android). Watch for: gomobile
bind actually succeeding at this pin with current Xcode, sheet-Modal behavior,
and keyboard interaction with the focused-input compromise.
