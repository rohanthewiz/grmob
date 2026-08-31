# Session: Running the todo example on the iOS simulator

**Session ID:** session_014UQGD2NBjK6NrbCqdUmmdn
**Date:** 2026-08-30, ~22:34
**Branch:** master

## Goal

Run `examples/todoapp` on the iOS simulator. No code changes were requested or
made — this was an end-to-end verification that the documented iOS ship path
(`docs/tutorial-todo.md` §"Shipping", `ios/build.sh` + `ios/project.yml`) still
works against the current tree and a current Xcode.

## Environment as found

| Tool | Version / state |
|---|---|
| Xcode | 26.6 (build 17F113), `xcode-select -p` → `/Applications/Xcode.app/Contents/Developer` |
| Go | 1.26.1 darwin/arm64 |
| gomobile | present at `~/go/bin/gomobile` (not on PATH by default — `build.sh` prepends `$(go env GOPATH)/bin`) |
| xcodegen | `/opt/homebrew/bin/xcodegen` |
| Simulator | iPhone 17 Pro, iOS 26.5, already booted |

Both prerequisites `ios/build.sh` guards on (full Xcode, gomobile) were
satisfied, so no fallback path was exercised.

## The four steps

```sh
# 1. Bind the bridge + the app package into the xcframework.
./ios/build.sh ./examples/todoapp

# 2. Regenerate the project from the xcodegen spec (untracked .xcodeproj).
cd ios && xcodegen generate

# 3. Build for the simulator slice.
xcodebuild -project GrMobApp.xcodeproj -scheme GrMobApp \
  -destination 'platform=iOS Simulator,name=iPhone 17 Pro' \
  -derivedDataPath <scratch>/dd build

# 4. Install and launch.
xcrun simctl install booted <scratch>/dd/Build/Products/Debug-iphonesimulator/GrMobApp.app
xcrun simctl launch booted com.grmob.demo
```

Step 3 ended in `** BUILD SUCCEEDED **`; step 4 returned
`com.grmob.demo: 81978`.

`-derivedDataPath` was pointed at a scratch directory rather than the default
`~/Library/Developer/Xcode/DerivedData` purely to keep the build products
addressable for `simctl install` in the next step — the app bundle path is then
deterministic instead of containing Xcode's hashed project directory name.

## Verification

Two checks, one static and one visual.

**The binding actually linked.** The `AppName` quirk documented in the tutorial
(gobind drops a bound package that exports no bindable symbol, taking its
registering `init` with it) is the failure mode that produces a nil manager at
runtime rather than a build error, so it is worth confirming from the headers
before launching:

```sh
grep -o 'Todoapp[A-Za-z]*' \
  ios/Frameworks/GrMob.xcframework/ios-arm64_x86_64-simulator/GrMob.framework/Headers/*.h
# → Todoapp, TodoappAppName, TodoappTodo   (+ a new Todoapp.objc.h)
```

`Todoapp.objc.h` existing at all is the signal: gobind emitted a header for the
package, so `todoapp.init` → `mobile.Register` is in the binary.

**The tree renders natively.** `xcrun simctl io booted screenshot` shows the
whole Go-defined view: the "Todos" header, the "What needs doing?" input with
its Add button, the All/Active/Done filter chips (All in the selected style),
the empty-state line "No tasks yet — add one above.", and "0 items left".
That is the full root of `examples/todoapp/app.go` — header, `InputWithSubmit`
row, chip row, list region, footer count — so styling, layout and the initial
render pass all crossed the bridge intact. The empty state also confirms the
bytdb store opened cleanly on a fresh container (snapshot `nil, 1`) rather than
erroring.

## Notes for next time

- Nothing in the working tree changed: `ios/.gitignore` already covers the
  xcframework, the generated `GrMobApp.xcodeproj` and Xcode build state, so
  `git status` stayed clean throughout.
- Re-running after a Go change needs steps 1 and 3 only. `xcodegen generate` is
  needed again only when `ios/project.yml` changes.
- Relaunching without any rebuild: `xcrun simctl launch booted com.grmob.demo`.
- Interaction was not driven in this session. The XCUITest suite
  (`ios/GrMobUITests/TodoAppUITests.swift`) is the mechanism for that:
  `xcodebuild test -project GrMobApp.xcodeproj -scheme GrMobApp -destination
  'platform=iOS Simulator,name=iPhone 17 Pro'`.
