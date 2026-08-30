# Session: Todo persistence with bytdb — data dir bridge, store, verified on both platforms

Session ID: `7f43a056-b2e2-4a2f-817e-b0380dcee5cc`
(Claude session link: https://claude.ai/code/session_01GKeRK9N45Cd4xWNZ2YX7ms)
Date: 2026-08-30 (~08:45–09:20)
Branch: `master` — continues from `2026-0830-0802-todoapp-and-tutorial.md`.

## What happened

Asked: "Would it be possible to add todo persistence with bytdb?" — the
next-step item from the previous session. Implemented end to end and
verified on both simulators.

1. **Toolchain gate first.** bytdb (all versions) requires go 1.26.1; local
   toolchain was 1.25.4. `GOTOOLCHAIN=auto` downloaded 1.26.1 transparently
   on `go get github.com/rohanthewiz/bytdb@v0.11.0`; gomobile bind and both
   native builds work fine under the switched toolchain. `go.mod` now says
   `go 1.26.1`. Also confirmed bytdb + btypedb are pure Go (no cgo/mmap) and
   cross-compile for `ios/arm64` and `android/arm64`.
2. **Bridge: `mobile.SetDataDir` / `DataDir`** (`mobile/bridge.go`) — Go
   can't discover the app sandbox path, so the shell registers it before
   `renderInitial`. iOS: `GomobileBridge.swift` init registers Application
   Support (created first; it doesn't exist by default — and it's the right
   home for app-managed data, not Documents). Android: `GomobileBridge(dataDir)`
   constructor arg, `MainActivity` passes `filesDir.absolutePath` (only a
   Context can name it; the bridge class deliberately holds no Context).
3. **`examples/todoapp/store.go`** — bytdb store, one table:
   `todos (id int PRIMARY KEY, title text, done boolean)`. Opens lazily on
   the first render pass (`openStore` is cheap after open: mutex + path
   compare); snapshot read at open seeds `NewState` initial values
   *synchronously* so persisted rows are in the initial tree — no
   empty-flash (a `hooks.UseEffect` load runs on a goroutine and would patch
   rows in a frame later). `nextID` resumes from `max(id)+1`. Every mutation
   helper in `app.go` write-throughs with the matching row op on the event
   path (µs into an fsync'd WAL — hard kill after a tap loses nothing).
   Store errors are logged and swallowed; nil store (no data dir: web
   preview, bare tests) = all methods nil-receiver-safe, app runs in-memory
   unchanged.
4. **Tests, all three levels.**
   - Go: `TestTodoPersistence` — relaunch simulated by `closeStore()` +
     fresh `mobile.Register(core.NewContext(), App)`; covers restore of
     rows/done/count, ID continuity (a post-relaunch add must not collide
     with a persisted row), delete and clear-completed durability. `TestMain`
     sets a temp data dir so the whole suite exercises the write-through path.
   - iOS: new `testTodosSurviveRelaunch` XCUITest — real typing,
     `app.terminate()` (hard kill), relaunch, row restored, deleted by its
     accessibility label. Both TodoAppUITests passed on iPhone 17 Pro sim.
   - Android: adb-driven on the emulator — add via `input text` + keyevent
     66, `am force-stop`, relaunch: row back; toggle done, force-stop again:
     checkbox + "Clear completed" restored. Screenshot-verified.
5. **Docs.** `docs/tutorial-todo.md`: new §5 "Persistence: write-through to
   an embedded database" (later sections renumbered 6/7/8), §3 snippets
   updated to the store-seeded state, stale "persistence out of scope"
   package doc rewritten, done next-step bullet removed.

## Key knowledge worth carrying forward

- **Go toolchain:** repo now requires go ≥ 1.26.1 (via bytdb);
  `GOTOOLCHAIN=auto` handles it, first build after a Go upgrade downloads
  the toolchain once.
- **Never `go mod tidy` here:** the `x/mobile`/`x/mod`/`x/sync`/`x/tools`
  requires are imported by no Go file — they pin the gomobile toolchain and
  tidy would drop them. A comment in go.mod now says so.
- **Data-dir contract:** shell calls `SetDataDir` before `RenderInitial`;
  app packages open stores lazily on first render (init runs too early).
  No data dir ⇒ nil store ⇒ in-memory app. Store singleton is process-wide
  (bytdb holds an exclusive file lock); a changed dir (tests) closes and
  reopens; `closeStore()` + re-Register = simulated relaunch.
- **Hydration pattern (no effect hooks needed):** open lazily at top of the
  root fn, seed `NewState` initial from the snapshot — synchronous, so the
  first mounted tree already has the data. `NewState` ignores the initial
  after the first pass, so the per-render snapshot cost is one slice copy.
- **UI tests must reset state now** — rows survive suite runs. iOS clears
  via delete buttons matched on `label BEGINSWITH 'Delete '` (the a11y
  label; the ✕ glyph is not the accessible name).
- **bytdb quick facts:** `bytdb.Open(path)` + `bsql.New(engine)`;
  `CREATE TABLE IF NOT EXISTS` works; int columns come back as `int64`;
  bool as `bool`; `$1` placeholders; per-statement autocommit is fine here.
- **Gradle invocation:** no `gradlew` in android/ — use
  `~/.gradle/wrapper/dists/gradle-8.14-all/<hash>/gradle-8.14/bin/gradle
  assembleDebug` with `ANDROID_HOME` exported.
- Both simulators left running the persistent todoapp build (iOS
  `com.grmob.demo`, Android `com.grmob.app`; Android has one done row
  "Persist on Android").

## Possible next steps

- Android instrumented UI test mirroring `testTodosSurviveRelaunch`.
- Progressive tutorial checkpoints (step-1/step-2 layouts or git tags).
- `htmlout` has no case for `List`/`TabView`/`Modal` (falls through to div);
  wasm runtime hardcodes its root view — both are gaps if web preview
  matters.
