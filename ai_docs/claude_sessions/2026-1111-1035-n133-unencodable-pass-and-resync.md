# N-133: an unencodable pass is dropped, and a drifted host asks for Go's tree

Session: `2ece2445-d8d5-41c3-8952-63656028d745`
**Date:** 2026-11-11 10:35 · **Branch:** master (84d8ff5, plus this commit)

## Goal

The last of `/work-next-list N-131 N-132 N-133`. N-131 and N-132 are in
`2026-1111-1020-n131-n132-web-push-deadlock-and-dispatch-order`. N-133 is the
plan's R3 (one unencodable value drops a whole pass, silently) and R4 (no host
can detect or recover from a desync), and R4 depended on N-132's ordering.

## R3: encode before committing

`renderAgainLocked` used to set `currentTree = newTree` and then call
`renderJSON`, which answered a failed marshal with `{"error":"failed to encode
JSON"}`. Go then believed the host had the pass, and the host refused it. One
NaN froze every unrelated change made with it.

Now the patches are encoded (`encodePass`, which returns the error) before
anything is committed. A failure is handled like the panic path just above
it: currentTree is kept, the purge skipped (the host's callback IDs are the
old pass's), the dirty flag cleared, `[]` returned. The next pass is diffed
against what the host holds, so everything waits and then arrives together.
While a NaN stands, every pass drops: `reflect.DeepEqual(NaN, NaN)` is false,
so a NaN node is always in the diff.

The log names the first non-finite number (`render/nonfinite.go`, a reflective
walk of Style and Props in tree order, map keys sorted): "root/1 (Slider)
Props.value = NaN". Under debug mode it is also `core.ConcernUnencodablePass`
(`unencodable-pass`), added to debug-mode.md's table and the API pages. An
unencodable mount stands in the panic placeholder, and the first encodable pass
replaces root.

A Slider clamps its value into [min, max], so −Inf becomes 0 before the node
sees it; NaN slips through the clamp. The tests use NaN for the prop case and
an infinity in FlexGrow for the style case.

## R4: Resync

`render.Manager.Resync` sends currentTree as `[{"Type":"replace","TargetID":
"root","Changes":...}]` through `deliverLocked` (N-132's door). It re-renders
nothing, so a pending write is not in it, and after a dropped pass it is the
last good tree. Bridged as `mobile.Resync` (gomobile stub, both bridge
implementations, native.md's table) and `GrMobWASM.Resync` (webhost.Bindings,
all three browser hosts, deferred under PushGuard because it takes the lock).

Every host asks once per drift, behind a "resync pending" flag that a root
replace clears:

- **Web.** The `goCalls` queue now holds calls rather than (id, payload), so
  the request waits for the batch and for Go's push like a dispatch.
  `applyBatch` asks on a non-list payload, on an `add` with no parent, on a
  lookup miss, and on a throw (the forEach is wrapped, re-indented, so the
  post-passes still run). A root replace with no root on the page is answered
  by mounting into the mount point. The flag also drops when the Resync call
  returns, so a Go module without the binding cannot leave the page unable to
  ask again.
- **Swift / Kotlin.** `TreeStore.onDrift` (main-actor in Swift), set by each
  GrMobRuntime to dispatch `bridge.resync()` on the events queue. The Kotlin
  patch loop is wrapped in a try/catch. `update-props` and `update-style`
  used optional chaining on `resolve(path)`, so a miss on the commonest patch
  was a silent no-op; both now warn, and warn is what reports drift.

## Checks

- `render/encode_test.go` (5), `render/resync_test.go` (4).
- `wasm/verify/resync_test.mjs` (6), including a replay that drops a whole
  batch and ends on Go's final tree. The replay suite now asserts a faithful
  replay warns of no drift.
- ios/verify `checkResync`: a faithful replay never asks. Losing a whole
  batch, or any single patch, left nothing dangling on that transcript (every
  later patch resolved, onto the wrong node), so the drift is a node-sized
  hole removed after mount. `root/0`'s hole asked once, and the answer matched
  Go's final tree.
- Live, with scratch CDP probes: all 87 lessons routed with up to 12 buttons
  pressed in each gave no drift warning and no Resync. Removing lesson 2.2's
  event-log text node and then pressing its button gave one Resync, the node
  back with "1 · button", Go running, and nothing deferred.
- Android: the AAR rebuilt (`android/build.sh ./examples/tutorial`),
  `gradlew compileDebugKotlin` clean, `android/verify/run.sh` green. No test
  executes the Kotlin TreeStore (that is R9m), so N-134 now says its replay
  should cover the drift path, with `checkResync` as the model.
- `go vet`, gofmt, `go test ./...`, `-race` on the touched packages,
  `wasm/verify/run.sh` (browser pass, all 25 checks), `ios/verify/run.sh`.
- The tracked-Go-file census moved again, 740 → 743.

## Lessons

- A drift that leaves every path resolvable is invisible to path-based
  detection. On ios/verify's transcript no lost batch or lost patch dangles,
  because `add` clamps and later paths land on the wrong nodes. Resync
  repairs what is noticed; it is not an integrity check.
- Kotlin's `resolve(path)?.props = …` and Swift's optional chaining were the
  silent half of "the next full replace re-synchronizes". Check that a miss
  is reported before relying on the warn path.

## Next

Closed: N-133. Declined: None. Raised: None.
Deferred: None. Promoted: None. Moved: None.
Updated: N-134 (its R9m replay should cover the Kotlin drift path).
Full list: `ai_docs/todo/next-list.md`.
