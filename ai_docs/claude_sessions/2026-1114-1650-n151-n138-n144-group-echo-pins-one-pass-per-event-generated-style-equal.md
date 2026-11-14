# N-151, N-138, N-144: group-echo pins, one pass per event, a generated Style.Equal

Session: `94ba3d17-9a13-41bc-96af-d1aadbbafc95`

An unattended `/work-next-list` run. This doc covers the first three items.

## 1. Triage

Doable here: N-151 (red test), N-144 (CI watching), N-138 (P1–P3), N-139
(P7), N-143 (D5, D6, apidiff in CI), N-141 (CLI), N-146 (emulator TalkBack),
N-149 and N-150 (alarm snooze and missed-alarm record), N-134 (Android
process owner, large), and N-147 (re-reading the XCUITest tree).

Skipped because they need the user:
- N-015 and N-142 are marked as the user's call.
- N-098 is an API decision.
- N-024, N-047, N-113, N-129 and N-130 are contingent or have no consumer.
- N-045 is by design, and N-051 needs the Mi Max 3.
- N-099 needs a real iPad keyboard.
- Most of Validate needs hardware or a person listening.

## 2. N-151: the group-echo test pinned replaced lines

762584b (N-101) changed `Renderer.kt`'s RenderNode (`ownStop`, the `said` and
`says` pair) and `spokenByLabel` (the ", " phrase split).
`mobile/verify/group_echo_test.go` still pinned the old strings, so the Go
job was red on 762584b and fdb0a2b.

The test now pins:
- `ownStop` and its three conditions;
- the new `said`/`says` lines;
- `spokenByLabel`'s `val label = …`, `mutableSetOf(label)` and the split line.

**Gotcha:** `codeOf` masks string literals (`maskSwiftNonCode`). The gesture
names `"onClick"` and `"onLongPress"` therefore cannot be pinned through it;
that pin reads `valuesOf` instead. Commit `5ba0f27`.

## 3. N-138: the engine's per-pass cost

### P1: one pass per event (`ccae046`)

- **The dirty flag is a generation.** `core/context.go`'s `dirtyFlag` holds
  `gen` and `clean`. New `DirtyGeneration()` and `ClearDirtyThrough(gen)`;
  `ClearDirty` is kept.
  - `renderAgainLocked` and `renderInitialLocked` read gen before the build
    and clear through it when the pass commits or is abandoned. A write made
    during the pass therefore stays dirty and gets a pass of its own.
  - The mount now clears too, which saves one empty pass after every mount.
- **The pump** (`renderAndPush`) returns when `!IsDirty()`, asked under `mu`.
- **`TriggerRender`** copies the handler out under its lock and calls it
  inline. It used to run `go fn()`. The OnStateChange doc now says the
  handler runs synchronously and must not block.
  - The hooks tests register buffered channels (4–64 slots). They all pass.
- **`State.Set`** of an unchanged built-in scalar (`unchangedScalar`, a type
  switch over unnamed kinds) requests nothing. Named scalars and composites
  still always render.
- **The web:**
  - New `render.Manager.RenderIfDirty`.
  - The `RenderAgain` binding in `wasm/main.go`, `webhost/host.go` and
    `wasm/shots/host/main.go` now calls it. The pages still call
    `RenderAgain` after `ReceiveEvent`, which is now cheap.
  - `grmob-runtime.js`'s `checkLoop` (the per-frame `IsDirty` poll) was
    deleted. `GrMobApplyPatches` is defined in that same file, so the
    listener is always attached and the poll never found work.
- **Pins:**
  - `TestATapRendersTheRootOnce` (was `…TwiceToday`; `perTap` 2 → 1).
  - `TestAWriteDuringATapsPassStillGetsItsOwnPass`. It was mutation-checked:
    with a plain `ClearDirty` it renders once and fails.
  - `core/dirty_test.go`: the generation semantics and the scalar skip.

### P2: the diff (`96956e0`)

- **`core.Style.Equal`** is generated into `core/style_equal_gen.go` by
  `go run ./internal/styleeq/gen`.
  - The generator parses core's source and skips the generated file. That
    keeps a removed field from breaking the generator's own compile.
  - Fields are compared one of four ways:
    - `==` for pointer-free comparables;
    - `.Equal` for `*Style`;
    - `equalStyleMap` for `map[string]Style`, where nil and empty differ, as
      in DeepEqual;
    - a `reflect.DeepEqual` fallback for any other kind.
  - `internal/styleeq`'s test fails on a stale file. It also fuzzes 20,000
    pairs against DeepEqual, with NaN excluded: DeepEqual treats the same
    map object as equal without looking inside it.
- **`reconcile`:**
  - Diff now walks with a `differ` that keeps child indices in a `[32]int`
    buffer and spells a path only when it emits a patch.
  - `sameProp` compares string, bool, int and float64 without reflection.
- **Results on an M3:**

  | Benchmark | Before | After |
  |---|---|---|
  | DiffUnchanged, 1,001 nodes | 1.08 ms, 1,150 allocs | 120 µs, 1 alloc (the `differ`, 320 B) |
  | DiffUnchanged, 10,001 nodes | 11.7 ms | 1.27 ms |
  | ManagerNoopPass, 250 rows | 1.59 ms | ~0.8 ms |

### P3: the registry (`b08be3f`)

`purge` deletes unmarked entries in place and `clear`s `used`. `beginPass`
clears instead of calling `make`. The trade, written in the comment: map
buckets are never given back. ManagerNoopPass at 250 rows is now ~0.58 ms and
8,314 allocs.

### Collateral each time

- `wasm/verify`'s tracked-Go-file count (in `repowalks_test.go` and
  `timings_test.go`, five sentences) went 768 → 769 → 773. It counts
  untracked files too.
- `docs/api` was regenerated three times (`go run ./internal/apidoc/gen`).
- `wasm/verify/run.sh`, which includes headless Chrome, passed after P1 and
  after P2.

## 4. N-144: CI watched, closed

- The Android remedy drill was green in all seven runs from 43070809214 to
  43649142856.
- The only red runs were N-151's.
- `-race -shuffle=on` over the product packages and the separate census step
  are green.
- The site's `workflow_run` deploys of f53119e, 5ba0f27 and ccae046
  succeeded, and the log shows `git checkout --force <head_sha>`.

## Next

Closed: N-151, N-138, N-144. Declined: None. Raised: None. Deferred: None.
Promoted: None. Moved: None. Updated: None. Full list:
`ai_docs/todo/next-list.md`.
