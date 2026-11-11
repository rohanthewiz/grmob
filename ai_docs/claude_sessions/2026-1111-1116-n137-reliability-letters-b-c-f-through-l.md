# N-137: nine of the plan's small reliability items (R9 b, c, f–l)

Session: `24f5253b-6388-4010-a453-fc852c9ea0ef`
**Date:** 2026-11-11 11:16 · **Branch:** master (d068733 … 0536147, plus this commit)

## Goal

The middle of `/work-next-list N-135 N-136 N-137, plus the R9o and D1
slices`. N-135 and N-136 are in
`2026-1111-1057-n135-n136-hook-goroutine-panics-and-typed-json`. N-137 is
the plan's R9 table, b–l
(`ai_docs/plans/reliability-performance-convenience.md`). Each letter was
committed on its own and struck in the item as it landed. d and e are left
for the rest of this run.

## What landed

| Letter | Commit | Change |
|---|---|---|
| f | d068733 | `Debouncer`'s fire clears `d.timer` under the lock, before the call, only if it is still its own timer. A stale fire cannot clear a newer `Call` |
| b | 4e266a5 | Every `mobile` entry point, `ReportHostEvent` included, goes through `manager()`. Before `Register`, `RenderInitial` returns a one-Text tree naming the mistake, the rest return `[]`, and it is logged once |
| c | 35b302b | `renderAll` and `WithTheme`'s loop drop a nil child; `core.Keyed` over a nil returns nil |
| i, j | 005b31f | `serve -dev` takes its root from `go env GOMOD` (`locateModule`), and watches every package dir except `GOMODCACHE` (`watchesPackageDir`) |
| g, k, l | e84da4d | `installShell` stages copy, patch and record beside the shell. A failed or interrupted `grmob new` removes what it wrote. The display name is escaped per context in the templates |
| h | 0536147 | `GrMob.instantiateWasm`: stream only on exactly `application/wasm`, buffer otherwise, never retry |

## Decisions worth knowing

- **b:** the plan suggested returning `{"error": …}`. Hosts refuse that
  payload (N-133's finding), so `RenderInitial` returns a real tree that
  every host already draws.
- **c:** compaction costs the ordinary positional shift when a nil-returning
  child comes and goes. `core.If` was never affected: it renders an empty
  Fragment, not nil. Before, both natives skipped JSON null while parsing,
  and the web's `renderNode` would have thrown on it.
- **g:** a `-refresh` merges the staged tree over the existing shell rather
  than renaming, so the app's own files there (signing, `local.properties`)
  survive as before. The record is copied last.
- **k:** cleanup on failure, not a `-force` flag. Ctrl-C is caught for the
  scaffold's length: the running go command gets the terminal's interrupt,
  fails, and that error takes the cleanup path. A failed first build is
  kept.
- **l:** `[[html .Name]]`, `[[js .Name]]` and `[[printf "%q" .Name]]` are each
  the identity on an ordinary name. So existing pages still match
  `grmob web`'s comparison, though a name with an apostrophe will now read as
  "differs". `validateName` refuses control characters, the only thing that
  breaks the unescaped comment and README uses.
- **h:** a first version stripped MIME parameters and lowered the case. Node
  refused both: the spec matches the whole value ignoring case, Node matches
  it case-sensitively, and both refuse parameters. Exact `application/wasm`
  is the one value every engine streams.

## Checks

- Each letter has a test that fails without its fix (mutation-checked for b,
  c, f, k, l, and h's old shape kept as a test). g's tests encode the plan's
  failure; there was no `installShell` to run them against before.
- `go test ./...` before each commit. wasm/verify (browser pass) after c and
  h. ios/verify after c.
- The census moved 746 → 747 → 748 → 749.
- Not run live: `serve -dev` with a browser attached inside a workspace, and
  a `grmob new` interrupted by hand.

## Lessons

- A test of the stale-fire race that only stress-tested was useless: the
  last call fired anyway. Holding the debouncer's own lock (an internal
  test) made the race exact.
- `go list -m` in a workspace lists every module; `go env GOMOD` is the
  containing one. Checked in a scratch workspace before relying on it.
- The repo's `mjsnames` census reads ALL_CAPS words in `.mjs` comments as
  constants. Quote a platform's message rather than its error code.

## Next

Closed: None. Declined: None. Raised: None.
Deferred: None. Promoted: None. Moved: None.
Updated: N-137 (b, c, f, g, h, i, j, k, l struck, with a line each on how;
d and e remain). Full list: `ai_docs/todo/next-list.md`.
