# N-135 and N-136: a hook goroutine's panic is recovered, and typed JSON stays text

Session: `24f5253b-6388-4010-a453-fc852c9ea0ef`
**Date:** 2026-11-11 10:57 · **Branch:** master (db1d84a, 4138d42, plus this commit)

## Goal

`/work-next-list N-135 N-136 N-137, plus the R9o and D1 slices`: the rest of
the plan's Wave 1 (`ai_docs/plans/reliability-performance-convenience.md`,
"Suggested order"). R1–R4 were N-131 to N-133. This doc covers the first two
items. N-137 (R9 b–l), N-144's R9o slice and N-143's D1 slice follow in this
run.

## N-135 (R7): `core.Safely`

The only `recover()` in core, hooks, render and mobile was `core.Guard`
(`core/error_boundary.go`), used for render passes and dispatched handlers.
Host events (lifecycle, window, location) arrive through
`render.Manager.Dispatch`, so they were already guarded. The unguarded app
code was on the hooks' own goroutines:

| Site | Label in the log and the concern |
|---|---|
| `hooks/effect.go`, `go effect()` | `UseEffect` |
| `hooks/interval.go`, the ticker loop's `f()` | `UseInterval tick` |
| `hooks/interval.go`, UseTimeout's `AfterFunc` | `UseTimeout` |
| `hooks/interval.go`, UseTimeoutWhile's `AfterFunc` | `UseTimeoutWhile` |
| `hooks/debounce.go`, the `AfterFunc` | `Debouncer call` |
| `hooks/alarms.go`, `OnRing` from the ticker | `UseAlarms OnRing` (not in the plan; same shape) |

`hooks/now.go`, which the plan named, runs only `ctx.RequestRender()` on its
goroutine, so there is nothing to guard. `core.RenderManager.TriggerRender`'s
`go fn()` runs the render driver's nudge, not app code. Both were left alone.

`core.Safely(where, fn) bool` (`core/safely.go`) is `Guard` plus a log line
in `logRenderPanic`'s shape ("grmob: recovered panic during UseInterval
tick: …" and the stack), plus `ConcernHandlerPanic` under debug mode with the
place in the detail. A new concern kind was not added; the plan named
handler-panic, and the blast radius is the same (a callback abandoned
partway). Each hook still requests the render it would have after a normal
return. It is exported so app goroutines can use it
(`go core.Safely("load profile", …)`).

The apidoc generator refused the new file until it was listed in a topic
(`internal/apidoc/packages.go`, the views topic beside `error_boundary.go`).

Checks: `hooks/panic_test.go` (6) and `core/safely_test.go` (4). Restoring
the bare `f()` in the ticker crashes the test binary ("panic: first tick
fails"), which is how such a regression would show.

## N-136 (R8): route by the ID

`ReceiveEventPayload` sniffed the value's type and first tried every string
as a nested `{"value": …}` envelope. Every sender was read before the change:
webhost, `wasm/main.go` and the shots host each JSON-parse `{"value": …}` and
pass `payload["value"]`. The runtime's senders (`extractEventPayload`, the
code editor's text, the rich editor's document and selection JSON, tabs,
maps, swipes, shape taps) each send one plain value. Nothing depended on the
unwrap.

Callback IDs are `prefix + scope path + counter` (`nextIDLocked`), so the
prefix always starts the ID, even under a keyed scope (`txt_cb_r2/e0/1`). The
five prefixes are now constants (`voidIDPrefix` … `intIDPrefix`) used by
registration and dispatch alike. Dispatch picks the kind from the prefix and
checks the value against it. A mismatch is logged ("dropping event for X:
want a string, got float64") rather than missing its map silently, and a void
or back ID ignores a stray value. No coercion: a string for an int ID is
dropped, not parsed.

Checks: four new tests in `core/event_test.go`. The old `event.go`, swapped
back in, delivered `x` for `{"value":"x"}` and lost `{"value":true}` and
`{"value":7}`. `wasm/verify/run.sh` passed, browser pass included. The page
itself was not typed into.

## Housekeeping

- The tracked-Go-file census moved 743 → 746 (`wasm/verify/repowalks_test.go`,
  `timings_test.go`). N-137 and D1 will move it again.
- Commits: `db1d84a` (N-135), `4138d42` (N-136).

## Lessons

- "Route by X" needs every sender read first. The unwrap had no sender, so
  removing it was safe, but that was a finding, not an assumption.
- gopls diagnostics in this tree lag the files (it flagged `encode_test.go`
  and `nonfinite.go`, both fine under `go vet`). Trust `go build` and
  `go vet`.

## Next

Closed: N-135, N-136. Declined: None. Raised: None.
Deferred: None. Promoted: None. Moved: None.
Updated: None. Full list: `ai_docs/todo/next-list.md`.
