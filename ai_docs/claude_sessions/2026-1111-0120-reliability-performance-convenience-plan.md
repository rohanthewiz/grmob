# A reliability, performance and convenience plan for GrMob

Session: `c54148c2-b113-4800-954e-9b47caf89d66`
**Date:** 2026-11-11 01:20 · **Branch:** master (5abb646, plus this commit)

## Goal

The user asked for a deep look at GrMob, with every opportunity to increase
reliability, performance and convenience written into a plan under
`ai_docs/plans/`. Mid-session they added: run `/sess-wrap` when completely
done.

No product code changed. The session produced one plan, fourteen next-list
items, and this doc.

## What was produced

- **`ai_docs/plans/reliability-performance-convenience.md`** — the plan. Four
  parts (Reliability R1–R9, Performance P0–P10, Convenience C1–C10,
  Housekeeping), a suggested order in five waves, a list of what was checked
  and found sound, what was deliberately not proposed, and how to reproduce
  each measurement.
- **`ai_docs/todo/next-list.md`** — N-131 to N-144 raised from the plan;
  N-013's text corrected (see below); Next ID is now N-145.

## How the review was run

Four slices, in parallel:

| Slice | Who | Scope |
|---|---|---|
| Engine | this session | `core`, `render`, `reconcile`, with benchmarks and probes |
| Native hosts | background agent | `mobile/`, the Compose and SwiftUI runtimes and shells, the build scripts |
| Web and tooling | background agent | `wasm/grmob-runtime.js`, `webhost`, `serve`, `htmlout`, `cmd/grmob` |
| Libraries and hygiene | background agent | `hooks`, `forms`, `comps` (sampled), the small packages, the test suite, CI, the module zip |

Each agent was told to read `next-list.md`, `non_goals.md`,
`need_hardware.md` and the September audit (`fable-pre-0.1.0-analysis.md`)
first, to report only what those do not already track, and to label every
finding confirmed or plausible. The agents were read-only in the repo;
`git status` was clean until the plan was written.

The headline claims of each agent's report were checked again here before
they went into the plan: the ordering-race probe was re-run (1–4 inversions
per 20,000 taps on three runs), and the Android lifecycle, the web
`GoInvokeCallback` path, the tracked binaries, the debouncer and the dev
watcher were re-read at the cited lines. The plan marks those ✔. The rest
carry the agent's own label.

## Baseline at 5abb646

`go vet ./...` clean. `go test ./... -count=1`: 20 s wall, 3,180 results, no
failures, six deliberate skips. `go test -race` on `hooks forms render core
mobile`: no races. `go mod tidy -diff`: clean. The latest tag is v0.8.1, not
v0.8.0 as the session's starting notes assumed.

## What the engine slice found (verified first-hand)

All on an Apple M3, with throwaway code in a scratch module that `replace`d
the repo.

- **Every event renders the whole app twice.** A handler's `State.Set` leaves
  a token for the pump; `Dispatch*` renders under the lock; the pump then
  renders again to produce `"[]"`. Twenty taps ran the root function forty
  times. `manager.go:341-344` describes the second pass as intended.
- **The diff costs more than the build, and is 10× slower than it needs to
  be.** On 1,001 nodes: build 0.43 ms, diff of an unchanged tree 0.97 ms.
  60% of the diff is `reflect.DeepEqual`. A generated field-by-field
  `Style` equality plus a scalar fast path for props took 126 µs; with child
  paths built lazily, 97 µs and no allocations.
- **`core.Style` is 1,024 bytes.** A pass allocates 1.75 KB per node and the
  build benchmark spends about 60% of its CPU in allocation and GC.
- **One NaN drops a whole pass.** `FlexGrow`, `FontSize`, `Gap`,
  `BorderRadius` or a Slider value holding a NaN makes `renderJSON` return
  `{"error":"failed to encode JSON"}` after `currentTree` has advanced, with
  no log. `core.Opacity` and the chart data path already sanitise; nothing
  else does.
- **Typed text that parses as JSON is rewritten on the web.**
  `ReceiveEventPayload` unwraps a string that parses to an object with a
  `value` key: typing `{"value":"x"}` delivers `x`; `{"value":true}` calls no
  handler. This is the September audit's item 2.3, still open.

## What the three agents found (headlines)

- **Web:** editor toolbar commands deadlock the Go runtime (the pump holds
  `Manager.mu` across `ApplyPatches`, and `runCodeCommand` re-enters Go
  through `GoInvokeCallback`); reproduced on lessons 4.13 and 4.14. A dead Go
  program shows nothing. `applyBatch` has no failure isolation. The dev
  watcher runs `go list -deps` on every 250 ms poll. The wasm module is
  13.1 MB and nothing compresses it (gzip 2.98 MB, brotli 2.07 MB); `-s -w`
  saves 3.8%, not the 40% `build.sh` claims.
- **Native:** a tap's returned patches can be enqueued after a later pushed
  pass (the converse of the September audit's item 1.2). Android builds a new
  runtime, store and bridge in every `MainActivity.onCreate`, which loses
  pushes, resets the text-edit sequence, rewrites the time zone and replays
  deep links. No host can detect or recover from a desync. Android decodes
  every payload on the main thread, twice. The native bind builds four ABIs
  for a debug loop.
- **Libraries and hygiene:** a panic in a hook's goroutine ends the process.
  `Debouncer.Pending()` stays true after firing. Tests fail under
  `-shuffle` on seeds 2 and 3. Two Mach-O binaries are tracked (`chat` is 21%
  of every module download). The test driver is hand-written 16 times,
  including in the scaffold. There is no async-resource hook. A consumer
  importing `core`, `comps` and `render` links none of bytdb, element or
  rweb, so splitting the module is not worth doing.

## Decisions made while writing the plan

- **No sequence numbers on patch payloads.** Delivering every pass through
  the listener inside the lock hold makes ordering correct by construction,
  and a resync can be a root `replace` pushed through the same door, which
  every host already applies. The wire shape and the three verify transcripts
  stay as they are.
- **The `core.Style` size reduction is an experiment with a gate** (keep it
  only for a 30% gain on the no-op pass benchmark), not a commitment.
- **A subtree memo with callbacks is not proposed.** Keyed callback scopes
  make it feasible, but it must replay focus stamps, text-edit stamps and the
  end-reached ledger, and after the diff fix the remaining cost is
  allocation.
- **Windowing `core.List` stays declined**, as `non_goals.md` records. The
  Android decode item attacks the same parse cost from the decoder's side.
- **Next-list items are grouped, not one per finding.** The plan has about
  sixty items; fourteen list entries point into it by plan ID, so the list
  stays readable and `/work-next-list` gets units it can close.

## One existing item corrected

N-013 (a non-goal) said `mobile.SetTimeZone` runs once at startup. On Android
it runs on every Activity recreation. The decision stands; the item's text
now says the premise was wrong and names N-134 as what restores it.

## Things that went wrong in the session

- Two agents and this session wrote into the same scratch directory
  (`bench/`), and one overwrote the engine benchmark file. The numbers had
  already been taken; the prototype was rebuilt in its own directory. Worth
  remembering: give each agent its own named scratch directory in the brief.
- The next-list edit script's run-twice guard matched the `Next ID: N-131`
  line and refused the first run. Fixed to match the item header.
- The first draft of the plan said there were 29 direct `GoInvokeCallback`
  call sites (a `grep -c` that counted comments) and stated as fact that
  `.gitattributes export-ignore` does not affect Go module zips. The count is
  23, and the second claim was not checked; both were corrected before the
  commit.

## Not verified

- Every Android figure is one emulator's, as `need_hardware.md` already
  records.
- The save-to-swap latency (about 1.8 s for a leaf edit) is summed from
  measured components, not timed end to end.
- The native bind loop was not timed.
- Whether GitHub Pages compresses `.wasm` was not checked.
- Google Play's current target-API floor should be confirmed before choosing
  a `targetSdk`.
- The agents' findings beyond the ones marked ✔ in the plan were verified
  once, by the agent that found them.

## Files

- `ai_docs/plans/reliability-performance-convenience.md` (new)
- `ai_docs/todo/next-list.md` (N-131 to N-144 raised; N-013 corrected; Next
  ID N-145)
- `ai_docs/claude_sessions/2026-1111-0120-reliability-performance-convenience-plan.md`
  (this doc)

## Next

Closed: None. Declined: None. Raised: N-131, N-132, N-133, N-134, N-135,
N-136, N-137, N-138, N-139, N-140, N-141, N-142, N-143 (Open); N-144
(Validate). Deferred: None. Promoted: None. Moved: None. Updated: N-013.
Full list: `ai_docs/todo/next-list.md`.
