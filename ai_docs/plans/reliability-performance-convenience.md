# Reliability, performance and convenience plan

*Analysis date: 2026-11-11, against master at 5abb646 (v0.8.1 plus two
commits). Produced by a four-way review: the engine (`core`, `render`,
`reconcile`), the native hosts and bridge, the browser target with the CLI and
dev server, and the library layer with the test suite, CI and repo hygiene.*

*How to read the evidence labels. **repro**: a throwaway probe outside the
repo reproduced it. **measured**: a benchmark or profile produced the number
quoted. **read**: traced end to end by reading the code, not executed.
**plausible**: inferred, not fully traced. The engine findings and the items
marked ✔ were verified a second time while writing this plan; the rest were
verified once, by the review that found them.*

*Nothing here duplicates an Open, Validate or Non-goal entry of
`ai_docs/todo/next-list.md`, or an entry of `non_goals.md` or
`need_hardware.md`. Where an item touches one of those, it says so. The
plan's items are tracked on the next list as N-131 to N-144, grouped by the
sections below.*

## Baseline

All green at 5abb646: `go vet ./...` (5 s), `go test ./... -count=1` (20 s
wall, 3,180 results, 0 failures, 6 deliberate skips), `go test -race` on
`hooks forms render core mobile` (no races), `go mod tidy -diff` (clean).

Per-pass cost on an Apple M3, list-shaped screen of keyed
`Row(Text, Text, Button)`:

| Nodes | Build | Diff, unchanged tree | Whole no-op pass | Allocated per pass |
|---|---|---|---|---|
| 1,001 | 0.43 ms | 0.97 ms | 1.46 ms | 1.75 MB, 11.0k allocs |
| 10,001 | 4.67 ms | 9.82 ms | 14.5 ms | 17.6 MB, 114k allocs |

The tutorial's home screen (105 nodes) passes in 0.18 ms. Typical screens are
cheap; the numbers matter for long lists, grids and phones, which run this
several times slower.

Three facts frame most of the performance section:

- **The diff costs more than twice the build**, and 60% of it is
  `reflect.DeepEqual` (profile).
- **`core.Style` is 1,024 bytes** and every node allocates one. The build
  benchmark spends about 60% of its CPU in the allocator and GC.
- **Every event renders twice** (see P1).

## The ten that matter most

| # | Item | Kind | Effort |
|---|---|---|---|
| 1 | R1 — editor toolbar commands deadlock the Go runtime on the web | reliability | S–M |
| 2 | R2 — a tap's returned patches can arrive after a later pushed pass | reliability | S |
| 3 | R3 + R4 — an unencodable pass is dropped silently; no host can resync | reliability | M |
| 4 | R5 — Android rebuilds its whole runtime on every rotation | reliability | M |
| 5 | P1 + P2 — two render passes per event, and a diff 10× slower than needed | performance | S + M |
| 6 | P7 — the dev watcher runs `go list -deps` four times a second | performance | S |
| 7 | C1 — a `grmobtest` package to replace 16 hand-rolled test drivers | convenience | M |
| 8 | C2 — `hooks.UseAsync`, with cancellation and stale-result dropping | convenience | M |
| 9 | C3 + C4 — `go tool grmob`, and a native bind that builds one ABI | convenience | S |
| 10 | C5 — framework-owned shell sources sync on every build | convenience | M |

---

## Part A — Reliability

### R1. Editor toolbar commands kill the Go runtime in the browser — **repro**

- **Where:** `render/manager.go:172-194` holds `Manager.mu` across
  `ApplyPatches`. On the web that call runs `applyBatch` synchronously, which
  reaches `runCodeCommand` → `emitCodeChange` → `window.GoInvokeCallback`
  (`wasm/grmob-runtime.js:5313` ✔) → `RenderAgain` (`wasm/index.html:660-664`
  ✔) → `mu.Lock()` on a mutex the pump already holds. The rich-text editor
  takes the same route through `richSendDoc` (`:6762`).
- **Effect:** clicking CodeEditor "Indent" (lesson 4.13) or RichTextEditor
  "Bold" (lesson 4.14) prints `fatal error: all goroutines are asleep -
  deadlock!`. The page stays on screen and never responds again.
- **Why the existing guard misses it:** `applyingTree` protects only
  `dispatchFromElement` (`:5426`). There are 23 direct `GoInvokeCallback`
  call sites, and the comment at `:8854-8856` ("mount and patch do not nest
  today") is no longer true.
- **Fix, in two layers:**
  1. JS: one `dispatchToGo(id, payload)` that queues while `applyingTree > 0`
     and flushes when the outermost `withTreeApplied` exits. Route all 23
     sites through it.
  2. Go: `webhost` sets a plain bool around the listener call (js/wasm is
     single-threaded). `renderAgain` and `receiveEvent` entered while it is
     set log a loud line and defer the work instead of blocking. This turns
     any site layer 1 misses into a log line, not a dead page.
- **Related hazard:** `windowMetrics.track()` → `HostEvent` (`:10441`) is also
  a nested Go call inside a batch. It is safe only because `hostEvent` takes
  no mutex; moving the web host onto `Manager.Dispatch` would deadlock it.
- **Pin:** a live-build check in `wasm/verify/browser.mjs` that clicks both
  toolbars and then dispatches one more event.

### R2. A tap's returned patches can be applied after a later pass — **repro** ✔

- **Where:** `render/manager.go:345-350`. `Dispatch*` releases `mu` when it
  returns, and the host enqueues the returned JSON afterwards
  (`GrMobRuntime.kt:286-291`, `GrMobRuntime.swift:180-186`). Between those two
  moments the pump can render pass N+1 and its listener can post first.
- **Effect:** paths are positional, so an inversion writes to the wrong node
  or leaves a stale prop, silently.
- **Evidence:** a pure-Go probe with the hosts' shape (one FIFO fed by both
  paths; a handler that sets state once synchronously and once from a
  goroutine) gave 1–4 out-of-order arrivals per 20,000 taps on each of three
  runs. The JNI string copy widens the window on a device.
- **History:** the September audit's item 1.2 fixed the opposite direction (a
  push overtaken by a later dispatch). `render/push_order_test.go:89` pins
  only that one.
- **Fix:** when a listener is attached, `Dispatch*` hands its patches to the
  listener inside the same lock hold and returns `"[]"`. Every pass then
  leaves through one ordered door. Hosts need no change: both already apply
  an empty return as a no-op. Without a listener (tests, polling hosts) the
  return value stays as it is.
- **Pin:** the converse of `push_order_test.go`: interleave dispatches with
  goroutine sets and assert arrivals are monotonic.

### R3. One unencodable value drops a whole pass and desyncs the host — **repro** ✔

- **Where:** `render/manager.go:321-322` sets `currentTree = newTree` before
  `renderJSON`, which returns `{"error":"failed to encode JSON"}` when
  `json.Marshal` fails (`:435-441`) and logs nothing.
- **Effect:** `core.FlexGrow`, `FontSize`, `Gap`, `BorderRadius` or a
  `core.Slider` value holding a NaN (a `done/total` with `total == 0`) turns
  the entire pass into that error object. Every other patch of the pass is
  lost and never resent, because Go already believes the host has it. The
  natives log and drop the payload; the web runtime throws
  `patches.forEach is not a function`. On the initial mount the screen is
  blank.
- **Already guarded:** `core.Opacity` and the chart data path sanitise
  non-finite values. Nothing else does.
- **Fix:** treat an encode failure like the render-panic path at
  `manager.go:280-310`: keep the last good tree, skip the purge, return
  `"[]"`, and log the marshal error. Under debug mode, walk the new tree and
  name the first node path and field holding a non-finite float, as a
  concern.
- **Pin:** a NaN in a style float and in a prop, each alongside an unrelated
  text change; assert the text change arrives on the next good pass.

### R4. No host can detect or recover from a desync — **read**

- **Where:** `TreeStore.kt:161-165` and `TreeStore.swift:121-125` say "the
  next full replace re-synchronizes", but nothing ever requests one. In
  `applyBatch` an unresolvable path returns silently
  (`grmob-runtime.js:8930` ✔), the `switch` has no default, and a throw in
  one patch aborts the batch and skips every post-pass (`:9245-9279`).
- **Fix:** a resync request that needs no wire change.
  - Go: `Manager.Resync()` pushes
    `[{"Type":"replace","TargetID":"root","Changes":<currentTree>}]` through
    the listener, in order, without re-rendering. Every host already applies
    a root replace, because navigation emits one.
  - Hosts: on a dangling path, a non-array payload or a throw, log the patch,
    set a "resync pending" flag and call `mobile.Resync()` /
    `GrMobWASM.Resync()`. The flag clears when a root replace lands, so a
    queue of stale patches cannot start a storm.
  - JS: wrap each batch in try/catch so the post-passes still run.
- **Depends on:** R2, so that the replace cannot be overtaken.
- **Pin:** each host's verify harness replays a transcript with one patch
  deleted and asserts the tree matches after the resync.

### R5. Android builds a new runtime for every Activity — **read** ✔

- **Where:** `MainActivity.kt:58-59,83` constructs `GomobileBridge`,
  `GrMobRuntime` and its `TreeStore` in `onCreate`. The manifest claims no
  `configChanges`, so rotation, a dark-mode switch and a fold all recreate.
  There is no `onDestroy`.
- **Effect, four ways:**
  - **Lost pushes.** `start()` renders, spends the parse time on the main
    thread, and only then replaces the listener (`GrMobRuntime.kt:117-126`).
    A pump pass in that window is diffed against the new tree and delivered
    to the dead store.
  - **Edit sequence reset.** `lastEditSeq` restarts at 0 while Go's per-field
    ledger keeps its high-water mark (`core/text_edit.go:160-162`), so a
    rewrite can rebase from an edit Go has not applied.
  - **Time zone rewritten.** `setTimeZone` runs on every recreation, which
    `mobile/timezone.go:26-31` calls a data race. Next-list N-013 assumes it
    runs once.
  - **Cost.** Each rotation pays the full bridge, parse and build on the main
    thread (see P5).
- **Fix:** hold the bridge, runtime and store in a process-scoped owner
  (`GrMobApplication`); the Activity only calls `setContent` and reattaches
  what genuinely needs an Activity (permissions, window metrics).
- **Also:** `Permissions.kt:154,195` keeps a static strong reference to the
  Activity; clear it in `onDestroy`.

### R6. A deep link is replayed on every recreation — **read** ✔

- **Where:** `MainActivity.kt:90` calls `reportDeepLink(intent)` unguarded.
  Five lines later the notification tap is guarded with
  `savedInstanceState == null` for exactly this reason, and the comment at
  `:174-178` claims the deep link is safe too.
- **Effect:** open a `grmob://` link, navigate elsewhere, rotate: the app's
  `OnDeepLink` handler runs again and navigates back.
- **Fix:** a process-level "already reported this intent" record. It falls
  out of R5 for free.

### R7. A panic in a hook's goroutine ends the process — **read** ✔

- **Where:** the only `recover()` in `core`, `render`, `hooks` and `mobile`
  is `core/error_boundary.go:65`, which covers render passes and dispatched
  handlers. Unprotected: `go effect()` (`hooks/effect.go:53`), the ticker body
  (`interval.go:145`), and the `AfterFunc` bodies in `interval.go`,
  `debounce.go:136` and `now.go`.
- **Effect:** a nil dereference inside a `UseInterval` callback is an app
  crash on a phone, and a dead page on the web.
- **Fix:** one shared `safely(ctx, label, fn)` built on `core.Guard`, logging
  as `logRenderPanic` does and reporting `ConcernHandlerPanic` under debug
  mode.
- **Pin:** a panicking interval, effect and debounce; assert the next tick
  still fires.

### R8. Typed text that is valid JSON is rewritten on the web — **repro** ✔

- **Where:** `core/event.go:524-545`. A string value is `json.Unmarshal`ed,
  and if it parses to an object with a `value` key, that inner value is
  delivered instead. `webhost/host.go:163` feeds it raw field text.
- **Effect:** typing `{"value":"x"}` into an input delivers `x`. Typing
  `{"value":true}` or `{"value":7}` calls no handler at all, so a controlled
  field snaps back. It is the September audit's item 2.3, and the cases most
  exposed are `CodeEditor` and any field holding JSON.
- **Fix:** route by the callback ID's kind prefix (`txt_cb_`, `bool_cb_`,
  `int_cb_`, `cb_`, `back_cb_`), which is already unambiguous, and never
  parse a text payload.

### R9. Smaller reliability items

| # | Item | Where | Fix | Evidence |
|---|---|---|---|---|
| a | A dead Go program shows nothing | `wasm/index.html:631,655-658`; scaffold template `:126-139` | `exited.then` shows a "stopped — reload" banner unless a deliberate `Shutdown` is in flight | read |
| b | `mobile` functions dereference a nil manager | `mobile/bridge.go:75-123` | Return `{"error":"mobile.Register was never called"}`; hosts draw the error text instead of a blank screen | read |
| c | Nil from a `ComponentFunc` shifts sibling paths | `GrMobNode.kt:55-57`, `GrMobNode.swift:188-190`; `core.Keyed` dereferences it | Compact nils in `renderAll` (September item 2.1, still open) | read |
| d | Duplicate keys across two `For` groups reach `LazyColumn` | `Renderer.kt:3691-3711,3894-3905` | Suffix repeats when flattening, as `idScope.seen` does | plausible |
| e | Nothing is torn down when a subtree leaves the DOM | `grmob-runtime.js:8723,8749,4310` | One `disposeSubtree(el)`; prune on `!el.isConnected`. A `ShowUser` map's GPS watch otherwise outlives its screen | repro |
| f | `Debouncer.Pending()` stays true after firing | `hooks/debounce.go:136-141` ✔ | Clear `d.timer` in the `AfterFunc` closure | repro |
| g | `vendorShell` is not atomic | `cmd/grmob/native.go:533-551` | Build in a sibling temp dir and rename. An interrupted copy is otherwise built unpatched as `com.grmob.app` | read |
| h | The instantiate fallback hides the real error | `wasm/index.html:625-628` | Fall back only on a wrong content type | repro |
| i | `serve -dev` cannot start inside a Go workspace | `serve/dev.go:99-106` ✔ | Take the root from `go env GOMOD` | repro |
| j | The watcher ignores locally replaced modules | `serve/dev.go:384` | Watch every dir outside `GOMODCACHE`. Every `grmob new -replace` app has this layout | read |
| k | `grmob new` cannot be resumed after a failure | `cmd/grmob/new.go:106-147,181-195` | Remove the dir this run created, or add `-force` | read |
| l | The app name is injected unescaped into the host page | `index.html.tmpl:6,138` | Escape for HTML and for a JS string | read |
| m | Android's patch applier is never executed by a test | `android/verify/run.sh` | Split the apply logic from the Compose state holders so the JVM harness can replay the Go transcript, as `ios/verify` does | read |
| n | Tests depend on execution order | `hooks/location_test.go:48`, `examples/todoapp/app_test.go:128` | Fix both; expose one reset (see C1); add `-shuffle=on` to CI. Seeds 2 and 3 fail today | repro |
| o | CI flake in the Android remedy drill | `android/verify/remedy.sh:67` | `rm -rf "$cold" || true` in the trap, or `gradlew --stop` first. Run 40020908087 failed after printing its own `OK:` | repro |

---

## Part B — Performance

### P0. Land the benchmarks first

`reconcile` has no benchmark, and ROADMAP's "Benchmark diff/patch engine" is
still open. Commit four, so every item below has a number to beat:

- `BenchmarkDiffUnchanged` and `BenchmarkDiffOneChange` in `reconcile`, at
  1k and 10k nodes;
- `BenchmarkManagerNoopPass` and `BenchmarkManagerTap` in `render`;
- a root-render counter test asserting one pass per tap (the pin for P1).

Alongside them, add pass statistics to debug mode: handler, build, diff and
encode durations, patch count and payload bytes per pass, readable through
`core` and logged behind a switch. This is ROADMAP's "patch logging" and
"latency profiling", and C9 builds on it.

### P1. Every event renders the whole app twice — **measured** ✔

- **Where:** a handler's `State.Set` calls `RequestRender`, which leaves a
  token in the pump's channel. `Dispatch*` then renders under the lock, and
  the pump wakes and renders again to produce `"[]"`. The comment at
  `manager.go:341-344` describes this as intended.
- **Evidence:** 20 human-paced taps ran the root function 40 times. On the
  web the shape differs but the waste is the same: the patches arrive through
  the push channel inside `ReceiveEvent`, and the explicit `RenderAgain` that
  follows returned `"[]"` on every measured tap, costing 1.2–6.0 ms.
- **Fix:**
  - A generation counter in place of the dirty bool. `MarkDirty` increments
    it; a pass records the value it started from; the pump skips when nothing
    has moved since. A set that lands mid-render still gets its follow-up
    pass, which is what the token guaranteed.
  - Web: stop calling `RenderAgain` after `ReceiveEvent` when the push
    listener is installed, and delete the per-frame `IsDirty` poll
    (`grmob-runtime.js:10651-10670`), which never finds work and throws once
    when Go exits.
  - `core/render_manager.go:33` spawns a goroutine per notification to call a
    function that is already a non-blocking channel send. Call it inline.
- **Small companion:** `State.Set` always requests a render. For the scalar
  kinds (`string`, `bool`, the integer and float types) compare first and
  return when nothing changed. `open.Set(false)` on an already-closed menu
  then costs nothing.

### P2. The diff is 10× slower than it needs to be — **measured** ✔

- **Where:** `reconcile/patch.go:159` and `:212` use `reflect.DeepEqual` for
  every prop value and for the 70-field `Style` of every node, and `:113`
  builds each child's path string whether or not a patch needs it.
- **Evidence:** a prototype with a generated field-by-field `Style` equality
  and a scalar fast path for props, on the same 1,001-node tree:

  | Variant | Time | Allocations |
  |---|---|---|
  | Today | 975 µs | 1,150 |
  | Generated equality, eager paths | 126 µs | 1,150 |
  | Generated equality, lazy paths | 97 µs | 0 |

- **Fix:**
  - `Style.Equal`, generated by a small `go generate` program from the struct
    itself (`!=` per comparable field; recursion for `HoverStyle`,
    `FocusStyle` and `PseudoStates`). A census test fails when a field is
    added and the generated file is stale, which is how `docs/api` is already
    kept honest.
  - Props: switch on `string`, `bool`, `int`, `float64` before falling back
    to `DeepEqual`.
  - Paths: pass the parent path and index down, and build the string only
    when a patch is emitted.
- **Pin:** the existing `reconcile` suite plus a fuzz test asserting
  `Style.Equal` agrees with `reflect.DeepEqual` on random styles.

### P3. The registry rebuilds four maps and re-spells every ID each pass — **read**

- **Where:** `core/event.go:379-418` allocates four new maps and a new `used`
  map per pass; `beginPass` allocates `used` again. Survivors are nearly
  always 100%.
- **Fix:** delete the unmarked entries in place and `clear(r.used)`.
  September's Phase 4 item 3, unchanged. Measure with P0 before and after;
  expect a small win, taken because it is a ten-line change.

### P4. `core.Style` is 1 KB per node — **measured**, fix unproven

- **Evidence:** `unsafe.Sizeof(core.Style{})` is 1,024. A pass allocates
  1.75 KB per node, and the build benchmark spends about 60% of its CPU in
  allocation and GC. After P2, the build is what remains of a pass.
- **Experiment, not a commitment:** move the rare field groups (the 15
  accessibility fields, the transform and animation group, the position
  group) behind embedded struct pointers. `encoding/json` flattens embedded
  pointers, so the wire shape stays as it is.
- **Risk:** every direct field read in `comps` and the hosts' Go-side helpers
  must tolerate a nil group. It is mechanical but wide.
- **Gate:** keep it only if `BenchmarkManagerNoopPass` improves by 30% or
  more. Otherwise record the result in `non_goals.md` and stop.

### P5. Android decodes every payload on the main thread, twice — **read**, figures from the repo

- **Where:** `TreeStore.kt:87` parses with `JSONArray`, then
  `GrMobNode.parse` copies it again into maps (`GrMobNode.kt:71-87`).
  `Startup.kt:48-54` records about 249 ms of parse and 185 ms of build for
  53 KB on the emulator, against 6 ms on iOS.
- **Why it is not the declined windowing item:** that cost recurs on every
  navigation `replace` and, until R5 lands, on every rotation. It is not only
  a cold-launch figure. `Startup.kt:42-44` already names "parsing straight
  into nodes" as the lever.
- **Fix:** decode off the main thread on a serial stage that preserves
  arrival order, streaming with `android.util.JsonReader` straight into
  `GrMobNode` and `GrMobStyle`.
- **Caveat:** every figure is one emulator's, as `need_hardware.md` records.

### P6. Web runtime hot spots — **measured** (CDP profiles of the tutorial)

| # | Item | Where | Cost seen | Fix |
|---|---|---|---|---|
| a | `applyStyle` writes all 48 managed properties on every created node | `grmob-runtime.js:2821-2823`, `:7401` | 86 of 603 ms busy over 30 navigations; the largest single JS function | On the create path assign only non-empty values; keep the update path total |
| b | Six post-batch passes each re-walk every touched element's ancestors | `:1096`, `:1108`, `:3753`, `:3849`, `:4343`, `:5812`, `:2724` | 67 of 313 ms over 20 theme flips | One combined walk with a visited set |
| c | `applySwipe` runs on every node with props | `:3576-3593` | 16–19 of 603 ms | Return early when no swipe prop or mark is present |
| d | `querySelector` per patch | `:8929`, `:8944` | 3–8 µs each, 2.7% of a theme flip | Leave it. September's Phase 4 item 5 is real but small; revisit only if a profile says so |

### P7. The dev watcher runs `go list -deps` four times a second — **measured**

- **Where:** `serve/dev.go:346-349` ✔. `stamps()` calls `goSources()` on
  every 250 ms poll and again inside `settle()`. The comment at `:54-56`
  says the build graph is re-read after every build.
- **Cost:** 0.115 s per call, so about half a core while idle, and roughly
  0.6 s of fixed latency on every save. Save-to-swap is an estimated 1.8 s
  for a leaf edit and 2.9 s for a `core` edit; compilation is 0.9 s and
  2.0 s of that.
- **Fix:** cache the file list; re-list only after a build or when `go.mod`
  or `go.sum` changes; poll with `stat` only.

### P8. Library hot spots — **measured**

| # | Item | Where | Cost | Fix |
|---|---|---|---|---|
| a | `CodeEditor` re-tokenises the whole buffer every pass | `comps/code_editor.go:139` | 0.19 ms and 465 KB at 500 lines; 0.92 ms and 2.7 MB at 1,700. Paid on any pass of the screen, not only on edits | A small LRU in `highlight` keyed by language, scheme and value |
| b | `DataTable` sorts every row every pass | `comps/data_table.go:377-395` | 4.67 ms at 10,000 rows, to show a page of 20 | Sort only on a change of rows or sort order |
| c | Form error reads are O(fields²) | `forms/form.go:505-535,442-472,189-195` | 314 µs and 655 KB per pass at 40 fields | Memoise `derived()` on the per-pass `Form`; allocate the record lazily. September's Phase 4 item 4 |
| d | The tutorial's `codeBlock` re-highlights static source | `examples/tutorial/widgets.go:124` | 19% of lesson build CPU | Falls out of (a) |

### P9. iOS

| # | Item | Where | Fix | Evidence |
|---|---|---|---|---|
| a | Min-content text is re-measured for every Row on every body evaluation | `Renderer.swift:987-990`, `GrMobMinContent.swift:103-190,422-431` | Memoise `measure` by run, size and weight; cache the `CTFont`. One text patch otherwise re-typesets every word under every ancestor Row | read |
| b | Images have no decoded cache and no downsampling | `Renderer.swift:424-453` | An `NSCache` keyed by src; `CGImageSourceCreateThumbnailAtIndex` at the box size, off the main thread. A failed load also spins forever with no log | read |

### P10. Build, test and CI time

| # | Item | Evidence | Fix |
|---|---|---|---|
| a | CI runs the race detector over source-census packages | `wasm/verify` takes 5.7 s plain and 54.9 s under `-race` locally; 68 s in CI | `-race` on product packages only; plain `go test` for `wasm/verify`, `internal/...`, `mobile/verify`, `aria/...`. Expect 30–60 s off the longest job (estimate) |
| b | Timer tests sleep real time | About 7 of `examples/tutorial`'s 10 s; `hooks/debounce_test.go` has 11 sleeps | `testing/synctest`. A probe ran `UseDebounce`, `UseInterval`, `UseTimeout` and `UseNow` through a real Manager: 10 fake seconds in 0.57 ms |
| c | The site workflow repeats CI | `site.yml`'s `verify` job re-runs the tutorial tests and `wasm/verify/run.sh` (108 s) | Depend on the CI run instead |
| d | Nothing compresses the 13.1 MB module | gzip gives 2.98 MB, brotli 2.07 MB; `serve` never sets `Content-Encoding` | Precompress in `build.sh` and serve the `.br`/`.gz` when accepted. Check what GitHub Pages does for `.wasm`; not verified |

Continuous events are a possible eleventh item: a slider dispatches one full
pass per pointer move into an unbounded queue (`Renderer.kt:1811-1815`,
`Renderer.swift:2816-2820`). Keeping only the latest pending value per
callback ID is a small change, but the lag it would cure is plausible, not
observed. Measure with P0's statistics on a device before doing it.

---

## Part C — Convenience

### C1. A `grmobtest` package

- **Today:** `type node struct`, a `tree()` that unmarshals `RenderInitial`,
  `find`, `tap` and `assertNoConcerns` are written out in six example test
  files (16 copies) and in the scaffold
  (`cmd/grmob/templates/app/app_test.go.tmpl:43-83`), so every `grmob new`
  app begins with about 45 lines of it. `ai_docs/SKILL.md:612-635` teaches it
  by hand.
- **Shape:** `grmobtest.New(t, App)` returning a driver with `Tree`, `Find`,
  `Tap(label)`, `Type(placeholder, text)`, `Toggle`, `HasText`, `AwaitText`
  and `NoConcerns`, plus `Reset()` over the eleven unexported `…ForTest`
  resets that downstream tests cannot reach today.
- **Unlocks:** R9n (shuffle-safe tests) and P10b (synctest) for app authors,
  not only for this repository.

### C2. `hooks.UseAsync`

- **Today:** `comps` widgets take caller-driven `Loading`, `Error` and
  `OnRetry` fields (`bible_verse.go`, `data_table.go`, `message_thread.go`,
  `paging.go`), and `bible_verse.go:55` tells the caller to "fetch in a
  goroutine or an effect". `UseEffect` runs a bare goroutine with no cleanup,
  ordering or cancellation (`hooks/effect.go:37-54`), so the tutorial tags
  results with an id to discard stale ones (`chapter3.go:207-213`).
- **Shape:**
  `UseAsync[T](ctx, func(context.Context) (T, error), deps...)` returning
  `{Data, Loading, Err, Reload}`. It cancels the context on a dependency
  change and on `Close`, drops results from a superseded generation, and
  runs under R7's guard.
- **Also closes:** September's item 2.4, without changing `UseEffect`'s
  signature.

### C3. The CLI as a `go.mod` tool

- **Today:** every printed and documented command is
  `go run github.com/rohanthewiz/grmob/cmd/grmob …`. The scaffold registers
  only `serve` as a tool (`cmd/grmob/new.go:137`).
- **Fix:** `go mod edit -tool=…/cmd/grmob` in the scaffold, so the command is
  `go tool grmob android`, always at `go.mod`'s version. `cmd/grmob` imports
  only the standard library, so it adds nothing to the app's graph. It also
  removes the CLI-versus-`go.mod` skew that triggers R9g.
- **Add while there:** `grmob version`; `grmob dev` (so `dev.sh` and its `sh`
  dependency are optional); doctor checks for `platforms/android-34`,
  accepted licences, a connected device, shell and host-page drift, and a
  `go.work`.

### C4. A native loop that builds one ABI

- **Today:** `-target=android` binds four ABIs (`cmd/grmob/native.go:320` ✔)
  and Gradle's `abiFilters` then discards one. iOS always binds
  `ios,iossimulator` (`:393` ✔) although the command builds only for the
  simulator. `ensureGomobile` runs `go get -tool` and `go build` on every
  invocation.
- **Fix:** a `-fast` flag (arm64 only; simulator-arm64 only), and skip the
  bind when the Go build ID has not changed. Preflight a device before the
  slow steps on Android, as iOS already does, and accept `-s <serial>`.
- **Not measured:** the loop time itself. Time it before and after.

### C5. Shell upgrades that do not overwrite the app's edits

- **Today:** `vendorShell` copies the whole shell once. A version mismatch
  only warns, and the one remedy, `-refresh`, overwrites everything: the
  manifest, `build.gradle` with its `versionCode` and signing, `project.yml`
  and the icons. `copyTree` never deletes, so a file renamed upstream is left
  behind and fails to compile as a redeclaration. Under a `replace` directive
  the stamp is a constant, so staleness is never detected.
- **Fix:**
  - Split ownership. The runtime sources nobody should edit
    (`com/grmob/runtime/`, `GrMob/Runtime/`) sync from the module on every
    build, as `grmob-runtime.js` already does. App-owned files are copied
    once.
  - Keep a pristine copy of what was vendored, so `grmob upgrade --diff` can
    show a three-way difference for the app-owned files.
  - Stamp a content hash, not a version string.
  - Delete files the new shell no longer has.

### C6. Release builds

- **Today:** the CLI runs `assembleDebug` and `-configuration Debug` only.
  `minifyEnabled` is false, there is no signing config, `grmob.json` has no
  version field, and `docs/platforms/native.md` has no release section.
  `targetSdk` is 34; Google Play has required API 35 or higher since
  August 2025 (confirm the current floor before choosing a number).
- **Fix:** `grmob android -release` (signed AAB) and `grmob ios -archive`,
  version fields in `grmob.json`, a raised `targetSdk`, and the docs section.
  ROADMAP's Packaging entry predates `cmd/grmob`; this is what remains of it.

### C7. A production web output

- **Today:** `grmob web` builds into `wasm/`, beside `main.go`. There is no
  `dist/`, no content hashing, no precompression, no manifest and no service
  worker. Plain `serve` sends `Last-Modified` only, so a browser may pair a
  cached `main.wasm` with a new runtime (plausible, not observed).
- **Fix:** `grmob web -dist` writing a directory ready to upload: hashed
  asset names, `.br` and `.gz` siblings (P10d), and a PWA manifest. Send
  `Cache-Control: no-cache` from plain `serve`.

### C8. Dev server conveniences

| Item | Today | Fix |
|---|---|---|
| Open on a phone | Binds all interfaces but prints only `localhost`; `internal/qr` is used by `comps.QRCode` alone | Print the LAN URL and a terminal QR |
| Address handling | The log line is `http://localhost%s`, wrong for `-addr host:port`; a taken port gives no hint | Format from the listener; suggest the next free port |
| Runtime errors | The overlay covers compile and boot failures; recovered render and handler panics reach the console only | Forward them to the overlay and, through one POST route, to the terminal |
| State across a hot swap | Scroll survives; the route survives in the tutorial only | An opt-in snapshot: the app reports a JSON blob, the dev client replays it as a host event after boot. Hook-slot replay stays out, as `dev.go:48-49` decided |

### C9. See what Go reports, on the device

- **Today:** concerns live only in memory (`core/debug.go:137-180`) and
  nothing under `mobile/`, `android/` or `ios/` reads them. Recovered panics
  go to `log.Printf`. There is no per-batch patch log.
- **Fix:** a `mobile.SetDebug` switch that logs new concerns per pass, a
  tag-gated patch log (count, bytes, apply time, dangling paths) fed by P0's
  statistics, and a debug overlay drawn by the shell from a system event.

### C10. Larger bets, each needing a design note first

- **Live reload on a real device.** `GrMobBridge` is already a nine-method
  string interface on both hosts, with `GomobileBridge` as its only
  implementation. A debug-only `RemoteBridge` speaking WebSocket to a desktop
  process (an extension of `serve -dev`) would put Go edits on a phone with
  no bind and no reinstall. Effort M–L; the highest-value item in this part
  if native iteration is where the time goes.
- **A hidden-scope signal for hooks.** `hooks/interval.go` says "hooks have no
  unmount signal today", so a ticker inside a tab that is no longer drawn
  still forces a whole-app pass per tick. Hooks are positional within a
  `Context`, so the unit that can be hidden is the scope. A per-pass
  "visited" mark on each context would let the framework pause a scope's
  tickers and cancel its `UseAsync` work when a pass ends without visiting
  it, and resume on the next visit. `UseIntervalWhile` then becomes a special
  case. The same gate, driven by lifecycle, would stop rendering while the
  app is in the background (today the pump has no such gate).
- **Pull-to-refresh.** Absent from `core` and `comps`; needs all three
  renderers.
- **Persisted state.** `examples/todoapp/store.go` is 210 lines of bytdb
  plumbing, and it only logs a failed write. A bytdb-backed `UsePersisted` in
  a sub-package (as `cart/cartdb` is) keeps `core` dependency-free and covers
  ROADMAP's "Device Storage (Plain)".

---

## Part D — Housekeeping

| # | Item | Detail |
|---|---|---|
| 1 | Two build artifacts are tracked ✔ | `chat` (3.9 MB Mach-O at the root) is 21% of every module download; `cmd/docs/docs` (9.6 MB) is in every clone. `git rm --cached` both, ignore them, and add a census check that no tracked file is Mach-O or ELF. History keeps 7.5 MB unless rewritten |
| 2 | `ai_docs/` is 19% of the module zip | 340 files, 322 of them session docs. A decision for the owner, stated here and not recommended either way. Moving them out of the module is the certain lever; whether a `.gitattributes export-ignore` shrinks a Go module zip was not checked |
| 3 | Releases are visible only in tag annotations | No GitHub Releases; `go get` users and pkg.go.dev never see the notes. A tag-triggered `gh release create --notes-from-tag`, and an apidiff job against the previous tag (it is run by hand today). README's top callout still says "Upgrading to v0.3.0?" |
| 4 | CI gaps | staticcheck (25 minor findings today), govulncheck, `go mod tidy -diff`, `-shuffle=on`, `android/verify/lint.sh`. The four XCUITest files run nowhere |
| 5 | Dead fields | `Context.idGen`, `children`, `childrenCursor` (`core/context.go:12,64-65`); `const DEBUG` in the runtime; `core.AppConfig.Locale` is read by nothing |
| 6 | Docs drift | `README.md:160` says 84 lessons (87 is right; the pin in `readme_counts_test.go:52-70` requires only one correct occurrence). `hooks/doc.go:22` ✔ says a dep-less `UseEffect` runs every pass; it runs once. `docs/index.md:56` lists five hooks of nineteen. `build.sh:16` claims `-s -w` saves 40%; measured 3.8%. `wasm/index.html:7` says 51 lessons |
| 7 | ROADMAP drift | Deep links are listed as planned and are done. Packaging is listed as planned with no mention of `cmd/grmob` |
| 8 | In-repo build scripts lag the CLI | `android/build.sh` and `ios/build.sh` use whatever `gomobile` is on PATH; the CLI honours the `go.mod` tool pin. `ios/build.sh` has no `LDFLAGS` pass-through |
| 9 | Stale comments this plan contradicts | `mobile/bridge.go:18-23` (arrival order), `TreeStore.kt:162-163` and `TreeStore.swift:122-123` (resync), `MainActivity.kt:174-178` (deep link), `grmob-runtime.js:8854-8856` and `:8685-8688`, `serve/dev.go:54-56`, `Startup.kt:46-62` (payload sizes). Fix each with its item |

---

## Suggested order

1. **Wave 1 — correctness, mostly small.** R1, R2, R3, R7, R8, R9a, R9b,
   R9f, R9i, R9o, D1. Then R4, which depends on R2.
2. **Wave 2 — measured performance.** P0 first, then P1, P2, P3, P7, P10a,
   P6a–c, P8.
3. **Wave 3 — Android.** R5 with R6, then P5 and R9m. These three share the
   same files and the same harness work.
4. **Wave 4 — convenience.** C1 and C2 first (they change how every later
   test and widget is written), then C3, C4, C5, C8, C9.
5. **Wave 5 — bets.** C6, C7, C10 and P4, each behind its own design note or
   gate.

Dependencies worth keeping in mind: R4 needs R2. P1's web half removes one
leg of R1 but does not replace its fix. C9 needs P0. R9n needs C1's `Reset`.
R5 makes R6 trivial and halves P5's exposure.

## Checked and found sound

So these are not re-audited:

- No Go call runs on a UI thread except the mount and the non-blocking
  listener setters; both natives dispatch through one serial worker.
- Render and handler panics are recovered; a panicking listener no longer
  kills the pump.
- Native path resolution is O(depth) per patch. Style is parsed once per
  patch, not per recomposition. Lists are lazy and keyed on both natives.
- Hot swap really exits the old instance, serialises swaps and falls back to
  a page reload. Builds are serialised; SSE reconnects and resyncs.
- Static serving is contained by `os.Root`.
- Timer hooks survive close-and-remount. `cart.Store` calls subscribers
  outside its lock.
- No hook is taken after a branch anywhere in `comps`; no goroutine is
  started there.
- A consumer importing `core`, `comps` and `render` links none of bytdb,
  element or rweb, and its `go.mod` requires only grmob. Splitting examples
  or `serve` into nested modules would save about 4% of the zip and is not
  worth doing.
- The scaffold is compiled, vetted and tested against the current tree in
  CI. Doc snippets are compiled and run.

## Deliberately not proposed

- **Windowing `core.List` over the bridge.** Declined in `non_goals.md` with
  a stated reopening condition (a genuinely long list). P5 attacks the same
  parse cost from the decoder's side and does not need the protocol change.
- **A subtree memo with callbacks** (`core.Memo(key, deps, view)`). Keyed
  callback scopes make it feasible now, but it must also replay focus stamps,
  text-edit stamps and the end-reached ledger. After P2 the remaining pass
  cost is allocation, which P4 addresses more simply. Revisit only if P0's
  numbers on a device say the build still dominates.
- **Sequence numbers on patch payloads.** R2 makes ordering correct by
  construction and R4 recovers through an existing patch type, so the wire
  shape and the three verify transcripts can stay as they are.
- **Bringing back a CHANGELOG.** D3 publishes the tag notes that already
  exist.

## Reproducing the measurements

The probes were throwaway code outside the repository. Each is small enough
to rewrite from its description, and P0 replaces the first three with
committed benchmarks.

- **Per-pass costs:** `n` keyed `Row(Text, Text, Button)` children in a
  `Column`; `BeginRenderPass` + `Reset` + render + `PurgeUnusedCallbacks` for
  the build, `reconcile.Diff` on two renders for the diff,
  `Manager.RenderAgain` for the pass.
- **Diff prototype:** a `reflect`-driven generator that prints `a.F != b.F`
  for each comparable field of `core.Style` and recurses on the two pointers
  and the map.
- **Passes per tap:** an atomic counter in the root function, a no-op
  listener, twenty `DispatchCallback` calls 10 ms apart.
- **Ordering race:** a handler that sets a monotonic counter once inline and
  once from a goroutine; one FIFO fed by both the dispatch return and the
  listener; count arrivals smaller than their predecessor.
- **NaN:** set a `float64` state to `math.NaN()` and feed it to `FlexGrow`,
  `FontSize`, `Gap`, `BorderRadius` and `Slider`.
- **JSON sniff:** `ReceiveEventPayload` with the string `{"value":"x"}` for a
  `txt_cb_` id.
- **Web profiles:** headless Chrome over CDP against the tutorial build,
  with navigation, no-op and theme-flip workloads.

## Verification checklist per wave

```
gofmt -l .                        # empty
go vet ./...
go test -race ./...               # then -shuffle=on once R9n lands
go test -bench . ./reconcile ./render    # after P0; compare against the table above
GOOS=js GOARCH=wasm go build -o /tmp/main.wasm ./wasm
wasm/verify/run.sh
ios/verify/run.sh
android/verify/run.sh
```
