# Session: iOS Simulator Pass (Attack-Order Step 5 — completion)

- **Session ID**: `47218cd6-2cf8-47cc-af4b-1c4cd8a563db`
- **Session link**: https://claude.ai/code/session_01WYkTemAUD23zXtGj8SVGjW
- **Date**: 2026-08-29
- **Branch**: master
- **Previous session**: `2026-0829-2207-swiftui-ios-renderer.md` (step 5, pre-Xcode)

## Goal

Close out the Xcode-dependent tail of step 5: full Xcode was installed since the
last session, so run the pending xcframework build + simulator pass and exercise
every event kind on a real simulator.

**Outcome: complete, all green.** Step 5 of
`ai_docs/plans/grmob-mobile-feasibility-analysis.md` is now fully verified end
to end. One real layout bug found and fixed.

## What ran

Toolchain now: Xcode 26.6 (iOS 26.5 SDK + simulators), selected via
xcode-select. gomobile at the pinned rev (@188f512ec823) binds fine under it.

1. `ios/build.sh` → `GrMob.xcframework` (device + simulator slices). The app
   then compiled against the *real* gobind surface — the stub header from last
   session matched exactly.
2. `xcodegen generate` → build → install → launch on "iPhone 17 Pro"
   (iOS 26.5). Initial render correct; push channel visibly ticking.
3. New **XCUITest smoke test** (see below) — passes:
   - push channel: "App running for Ns" advances with no native event in flight
   - void: 3 Increment taps → "Count: 3" (three distinct round-trips)
   - int: tab switch reaches Go, Form subtree appears
   - text: typing "Ada" round-trips per keystroke → "Hello, Ada!"
   - bool: Toggle flip → "Subscribed"
   - state retention: back to Counter tab → still "Count: 3"

## Bug found & fixed

The whole tree rendered vertically centered (tab bar mid-screen): `GrMobRoot`
handed the tree to the window bare, and SwiftUI centers a smaller-than-window
view by default. Fix in `ios/GrMob/Runtime/Renderer.swift` (GrMobRoot):
`.frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)` —
matches the Android root's top-flowing layout. Verified by screenshot + test
re-run.

## New/changed files

- `ios/GrMobUITests/GrMobUITests.swift` — **new**: simulator analog of
  `examples/mobileapp/app_test.go`; drives all four event kinds + push channel
  + tab-state retention via real taps. Tab labels are addressed as *buttons*
  (the hand-rolled tab bar wraps each label in a SwiftUI Button).
- `ios/project.yml` — added the `GrMobUITests` bundle.ui-testing target
  (needs `GENERATE_INFOPLIST_FILE: YES` or codesign fails) and an explicit
  `GrMobApp` scheme including it. Run with:
  `xcodebuild test -project GrMobApp.xcodeproj -scheme GrMobApp -destination 'platform=iOS Simulator,name=iPhone 17 Pro'`
- `ios/GrMob/Runtime/Renderer.swift` — the root-alignment fix above.

## Gotchas recorded

- No touch-injection CLI on this machine (no idb/cliclick); XCUITest is the
  repeatable way to drive the simulator.
- Memory `ios-toolchain` updated to reflect installed Xcode + these commands.

## Remaining known gap

The demo app has no Modal node, so `.sheet` behavior is exercised only by the
data-layer conformance harness (`ios/verify/`), not on-screen. Check it when a
Go app actually uses Modal (medium/large detents).

## Next step

Step 5 is done. Continue with the next step in the attack order in
`ai_docs/plans/grmob-mobile-feasibility-analysis.md`.
