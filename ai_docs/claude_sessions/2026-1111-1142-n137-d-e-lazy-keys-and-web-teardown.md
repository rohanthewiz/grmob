# N-137 d and e, and the R9o and D1 slices: lazy keys, web teardown, a drill's trap, two binaries

Session: `24f5253b-6388-4010-a453-fc852c9ea0ef`
**Date:** 2026-11-11 11:42 · **Branch:** master (4f2dd2f … 7160ec8, plus this commit)

## Goal

The end of `/work-next-list N-135 N-136 N-137, plus the R9o and D1 slices`.
Earlier docs this run: `2026-1111-1057-n135-n136-hook-goroutine-panics-and-typed-json`
and `2026-1111-1116-n137-reliability-letters-b-c-f-through-l`. This one has
N-137's last two letters (d, e), which close it, plus N-144's R9o slice and
N-143's D1 slice.

## N-137 d: a key two For groups share (4f2dd2f)

`GrMobList` flattens every Fragment among a List's children into one lazy
row list, and Go keys are unique per sibling list only. Two For groups keyed
by the same IDs gave `LazyColumn` a repeated key, and Compose throws on
that. `grMobLazyKeys` (new, import-free `GrMobLazyKeys.kt`) keeps a first
occurrence's key, keys a repeat as `Pair(key, n)`, and leaves unkeyed rows
positional by index. android/verify's JVM harness runs it with hand-written
cases (Go has no counterpart), and returning the raw keys fails it. The crash
was confirmed from source, not reproduced on a device.

## N-137 e: teardown on the web (94d7c8c)

A registry in the runtime: `onDetach(el, name, release)` when an element
acquires something outside the DOM, and `disposeDetached()` at
`withTreeApplied`'s outermost exit, which releases every registered element
no longer connected. One prune point covers every way out: a remove, an
ancestor's replace, a new mount. Holders:

- a map: clears its pending report, stops the location watch, calls
  Leaflet's `remove()`;
- a list's end-reached and start-reached IntersectionObservers;
- a canvas's place in the page-long `canvasTextObserver`, a ResizeObserver
  that held every canvas that ever drew text.

Also: a map element removed before its creation frame used to retry
`syncMap` every frame forever. It now retries once.

Harness changes: `dom.mjs` models `isConnected` (it was undefined, which the
map's `=== false` read as connected), `load.mjs` has an opt-in recording
ResizeObserver (`loadRuntime({ resizeObserver: true })`; off by default
because `windowMetrics` changes path when one exists), and the fake Leaflet
counts `remove()`. `teardown_test.mjs` has 7 tests: six fail without the
prune, and the old retry made the seventh hang.

## N-144 R9o: the remedy drills' trap (3a4a49b)

Shown in sh, bash and dash: under `set -e`, a command that fails in an EXIT
trap becomes the script's status. The Android drill's `rm -rf` raced the
gradle daemon that `sources.sh` leaves writing into the drill's
`GRADLE_USER_HOME`. Its cleanup now stops that daemon first (only if a
`daemon/` dir exists, or `--stop` would download a distribution), and then
removes `"${cold:?}"`, with neither step able to fail. INT and TERM `exit`,
so cleanup runs once from EXIT. The iOS drill got the same shape, and
`ios/verify/remedy.sh` passed end to end. The Android drill needs a network,
so CI is its check (noted on N-144).

## N-143 D1: two tracked binaries (7160ec8)

`git rm --cached chat cmd/docs/docs` (both still on disk) and two
`.gitignore` lines. Nothing consumed them: `cmd/docs/Dockerfile` builds from
source. The new `internal/trackedfiles` test fails on any tracked ELF,
Mach-O (either byte order) or PE by its first bytes. It leaves out the fat
header, which a Java class file shares. It named exactly these two before
the untrack. It sits in a package of its own because wasm/verify's shared
walk skips large and binary files and budgets its walks
(`repowalks_test.go`).

## Housekeeping

- Census 749 → 751.
- The safety check refused one shell command because a `sed` in it rewrote
  an `rm -rf "$cold"` line. Nothing ran. The script edits were redone with
  the Edit tool and the removals written as `${var:?}`, and the trap's
  semantics were then exercised with `false` standing in for the rm.
- A mutation check, the old map retry with a frame drain that empties the
  queue, hung the test and left the runtime mutated while it ran. The task
  was stopped and the runtime restored with `cmp`; no node process was left.

## Lessons

- A mutation that restores an infinite loop needs a time limit (macOS has
  no `timeout`). Keep the restore in a step that runs whatever happens.
- A test DOM missing a property the code tests with `=== false` hides every
  later truthiness test of it. Model the property instead.

## Next

Closed: N-137. Declined: None. Raised: None.
Deferred: None. Promoted: None. Moved: None.
Updated: N-143 (D1 struck and done), N-144 (R9o done; whether CI's Android
drill stays green is its check). Full list: `ai_docs/todo/next-list.md`.
