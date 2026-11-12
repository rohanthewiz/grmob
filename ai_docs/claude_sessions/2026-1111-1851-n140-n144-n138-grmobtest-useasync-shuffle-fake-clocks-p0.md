# N-140, N-144 and N-138 P0: grmobtest, UseAsync, shuffled tests, fake clocks, the benchmarks

Session: `31fda89a-e73d-40d0-9bcd-fd2faedb9766`
**Date:** 2026-11-11 18:51 · **Branch:** master (15f2b4c … 9cda534, plus this commit)

## Goal

Loaded the last session and the Next list, recommended an order, and the
user said to go with it and `/sess-wrap` at the end: N-140 C1 (`grmobtest`),
N-144's R9n and P10, N-140 C2 (`UseAsync`), then N-138 P0. N-134 was named
as worth doing but bigger (it needs the emulator) and was not started.

## N-140 C1: grmobtest (5adce9a)

`grmobtest.New(t, App)` returns a Driver over render.Manager. Every read
renders and decodes the wire tree into `*core.Node`; every verb dispatches
the callback ID found in that tree. Verbs: `Tree`, `Find`/`FindAll`, `Tap`,
`TapWhere`, `Type`, `Toggle` (by accessible name), `HasText` (exact Text),
`Shows` (substring in any word on screen; callback IDs are skipped by their
wire prefixes, so `Shows("3")` cannot match `cb_3`), `AwaitText`,
`NoConcerns` (fails with debug mode off rather than pass vacuously).

`Reset` reaches the eleven unexported `reset…ForTest` seams through a new
`internal/testreset` registry: core, keystore and permission register from
init, so no product package exports a test seam and grmobtest links nothing
an app didn't. A census test fails on an unregistered seam. New resets when
it opens the only live Driver; the last Driver's cleanup resets again. So a
host's boot report goes after New and before the first read (New renders
nothing). The tutorial's fold test had to move its `ReceiveWindow`.

Migrated: the scaffold template (its end-to-end test passes), counter,
signup, social (keeps its wire-named Style map on purpose), mobileapp and
todoapp (bridge-driven, so the free functions), and the tutorial, whose
`node` became an alias of `core.Node` (the hand-written Style mirror went;
18 typed comparisons fixed; about 1,500 call sites rewritten by script).
Docs: SKILL.md, getting-started, debug-mode, components, README, a generated
API page under a new Testing group.

## N-144 R9n: any order (f4d2875)

hooks: a location fix one test delivered was the next test's record. Every
hooks test feeding a device or permission record starts with
`freshProcessState` (`grmobtest.Reset` in and out). todoapp: the lifecycle
journey read the rows the persistence test left; both start with
`freshApp` (close the store, own TempDir, new Register, Reset). CI runs
`-shuffle=on`. Checked under seeds 2, 5, 17, 19, 23, 29, 31, 37 and once
with `-race -shuffle=on`.

## N-144 P10 a–c (34cdb34)

- a: CI's race step covers product packages; `CENSUS_PKGS` (wasm/verify,
  the native verifies, aria, internal) run plain. About 32 s against 127 s
  over that set locally; wasm/verify alone 6.7 s against 57.0 s.
- b: `grmobtest.WithFakeClock` = `synctest.Test` plus a 30 s real-time
  watchdog. hooks 5.3 s → 0.3 s, tutorial 9.4 s → 1.6 s, plus comps'
  TypingIndicator test. The debounce stale-fire test stays on real time: a
  goroutine blocked on a mutex never lets a bubble's clock move.
- c: site.yml deploys after a green CI `workflow_run` of a master push and
  checks out `head_sha`; a dispatch keeps the old gate. Unverified until
  the first push runs it.
- d (serving a compressed module) is a product change: raised as N-145.

## N-140 C2: UseAsync (4addc41)

`UseAsync[T](ctx, fn, deps...)` → `Async{Data, Err, Loading, Reload}`.
fn's context is cancelled on a deps change, Reload and close; each start is
a generation and only the current one writes, so a late answer that ignored
cancellation is still dropped. A deps change clears Data and Err; Reload
keeps Data and clears Err at once (BibleVerse's Error wins over Loading, so
keeping it would hide the retry). A panic becomes `ErrAsyncPanicked`. Four
mutants caught. hooks' doc table had said a dep-less UseEffect runs every
pass (N-143 drift); fixed there and in SKILL.md.

## N-138 P0 (9cda534)

Benchmarks: reconcile's diff at 1,001 nodes is 1.06 ms and 1,150 allocs (the
plan measured 975 µs and 1,150); a whole no-op pass 1.40 ms, a tap 1.42 ms
(250 rows, debug off). P1's pin, `TestATapRendersTheRootTwiceToday`: 20 taps
render the root 40 times, exact on the fake clock. Debug mode records a
`core.PassStat` per pass (handler, build, diff, encode, patches, bytes) in a
512-pass ring, `SetPassLogging` logs each; an A/B shows no cost with debug
mode off. On the fake clock the attribution test checks phases to the
nanosecond. ROADMAP's benchmark item is ticked.

## Housekeeping

- Tracked-Go-file census 751 → 767 across the five commits.
- CI at the start of the session (43070809214, `15f2b4c`) was green
  including the Android remedy drill: the first R9o check, noted on N-144.
- A mutation check (UseInterval's Close not stopping its ticker) inside a
  bubble hung: the test's own drain loop waits for quiet that never comes,
  and fake time makes it a hot loop. The restore was queued behind it; the
  file was restored by hand and the task stopped. That is what led to the
  watchdog, and the same mutant then failed in 30 s naming the test.

## Lessons

- In a synctest bubble an unbounded wait is a hot spin, not a stall; put a
  real-time watchdog around bubbles, and run mutations from a script whose
  restore is an EXIT trap (zsh rejects a function named like an alias).
- A reset-on-open test helper changes where setup goes: state a host reports
  at boot must be sent after the app is built and before its first render.
- `go test -bench 'A.*b'` does not match sub-benchmarks across the slash;
  write `-bench 'A/b$'`.

## Next

Closed: N-140. Declined: None. Raised: N-145.
Deferred: None. Promoted: None. Moved: N-144 → Validate.
Updated: N-138 (P0 done), N-143 (one drift line fixed), N-144 (R9n and
P10 a–c done; the R9o check's first green run). Full list:
`ai_docs/todo/next-list.md`.
