# Session: Error Boundaries

**Session ID:** session_01YWwH2ECtdanGaihAf4wtrE
**Date:** 2026-08-31, ~17:33
**Branch:** master
**Follows:** `2026-0831-1706-usememo-and-usereducer.md`

## Goal

Close the second-to-last Core Abstractions item: *"Error boundaries and safe
rendering fallback — nothing recovers a panicking `Render` today."*

The stakes are platform-specific and worth stating plainly, because they
justify the whole design: a render pass runs on the native event thread with a
bridge call in flight, so a panic unwinds straight out through JNI (Android) or
cgo (iOS) and **kills the process**. One bad slice index in one panel takes the
app down.

## `core.ErrorBoundary(child, fallback)` — `core/error_boundary.go`

Renders `child`; on a panic escaping its `Render`, renders `fallback(err)`
instead. Companions: `SafeRender(child)` (the same with the built-in fallback),
`DefaultErrorFallback`, and `Guard(fn) *RenderError` — the bare recover,
exported because the render driver in another package needs the same primitive
around a whole pass.

`RenderError{Value any, Stack []byte}` carries the raw panic value rather than
a message, with `Unwrap` so a fallback can `errors.Is` a panicked error instead
of string-matching it. `Stack` is captured *inside* the deferred recover, while
the panicking frames are still on the goroutine stack, so it names the guilty
component rather than the recover site.

### The two repairs, and why they differ

A panic partway through a render leaves two pieces of per-pass bookkeeping
half-advanced. Both are **positional**, so leaving them where they fell would
corrupt components with nothing to do with the failure:

| bookkeeping | left alone | symptom |
|---|---|---|
| hook slots | parent `Cursor` stranded mid-hooks | later siblings read the wrong slots — state visibly swaps |
| callback IDs | counters past the child's partial registrations | later siblings' IDs shift — taps hit the wrong handler |

**Hook slots are solved structurally, not by rollback.** The boundary claims
*two* child contexts via `UseChildContext` — one for the child, one for the
fallback — and renders into those:

```
parent ctx slots:  [ ... | childCtx | fallbackCtx | ... ]
                            ^ a panic can only strand a cursor in here
```

The parent footprint is then fixed at two slots whether the child succeeds,
fails early, or fails late. Rollback could not have achieved this: there is no
single "correct" cursor to rewind to that also accounts for the fallback's own
hooks.

**Callback IDs are a genuine rollback** (`snapshotCounters` /
`rollbackCounters`, new in `core/event.go`). Two halves, fixing different
problems:

- *Rewinding the counters* keeps ID assignment positional. A panicking subtree
  registers a handler count that depends on how far it got — which can vary
  with data between passes — so without the rewind every later component's IDs
  shift whenever the failure point moves.
- *Un-marking the IDs in between* is what makes `purge` collect the abandoned
  handlers. Purge keeps everything marked `used` since `beginPass`; the
  abandoned subtree marked its own, and those nodes are not on screen, so
  without this they stay dispatchable for as long as the failure persists.

The map entries themselves are deliberately left for `purge` — duplicating its
job here buys nothing.

### It does not latch

React's boundaries stay in the fallback until reset, because there the failed
subtree's *instances* are unrecoverable. Nothing of the sort is true here — the
tree is rebuilt from scratch every pass, so a child that panicked on a stale
index renders normally next pass once state settles. The boundary retries every
pass and heals on its own; latching would turn a one-frame glitch into a
permanently dead panel.

The cost is that a deterministic panic re-panics every frame, which is why the
doc comment warns that a fallback logging unconditionally logs at frame rate.

### Smaller decisions, each documented at the call site

- **The fallback is the notification hook.** `ErrorBoundary` logs nothing — the
  fallback gets the full `*RenderError` and the app decides. The *driver* does
  log, because a panic reaching it has no such owner.
- **The fallback is built AND rendered inside the guard.** `fallback(err)` is
  app code too; a crash while formatting the error must not defeat the boundary
  that exists to prevent crashes. A third-level failure returns a hand-built
  `Text` node — no builders, no hooks, no theme lookup, nothing that could
  panic again.
- **`ctx.Cursor = 0` on recovery.** A truncated child context is the exact
  signature of the conditional-hook bug the debug drift audit hunts, and zero
  is that audit's documented "rendered nothing this pass" value. Without it a
  recovered panic is reported twice: once as a panic, once as phantom drift.
- **`DefaultErrorFallback` gates its detail line on debug mode.** A panic
  message identifies the bug in a debug build and leaks internals into a user's
  screenshot in a release one.
- **The child gets its own hook namespace** — a real consequence, documented:
  a component reaching for `ctx.Scope("x")` expecting to share with something
  *outside* the boundary gets a different scope. Shared app state (nav,
  callbacks, theme, config) rides on pointers and is unaffected.

## Driver safety nets — `render/manager.go`

A boundary only covers what it wraps.

**Render passes** (`RenderInitial`, `renderAgainLocked`, `RenderAndGetPatches`)
run under `core.Guard`. On a panic the pass is abandoned, and three things
deliberately do *not* run afterwards:

- `currentTree` is not replaced — the last complete tree stays mounted, `[]`
  goes out.
- `PurgeUnusedCallbacks` is skipped. The partial pass marked only *some* IDs
  live; purging against that set would delete the handlers of the tree still on
  screen and leave every button dead. (This one is subtle enough that it got
  its own mutation-checked test.)
- `EndRenderPass` is skipped — the debug audit describes a *completed* pass.

The dirty flag *is* cleared: the state change has been consumed, and a
deterministic panic will not render better on an immediate retry — leaving it
set would make a polling host (WASM) re-panic every poll.

The initial pass has no last-good tree, so it stands in a placeholder node. Not
cosmetic: a nil `currentTree` reads as "not mounted" to `hasInitialRender`,
which parks the push pump **forever**.

**Event handlers.** Handlers run *between* passes, so no boundary in the tree
can see them — a nil deref in an `OnClick` is as fatal as a panicking `Render`.
`guardHandler` wraps all four `Dispatch*`. Recovery is deliberately partial and
documented as such: the handler's work is abandoned wherever it stopped, so
state may be half-updated. Nothing can know what half-applied means for an app,
so the next pass renders whatever state exists — strictly better than the same
half-written state *plus* a dead process.

**WASM host** (`wasm/main.go`) dispatches via `ctx.ReceiveEventPayload`
directly, bypassing the Manager entirely — found while checking the other hosts
route through the guarded paths. A handler panic there unwinds into the
`js.Func` callback and aborts the Go runtime, so it got the same guard.
(`mobile/bridge.go` already routes through `Dispatch*`, so it was covered.)

## Debug mode

Two new concerns in `core/debug.go`:

- `render-panic` — a boundary caught a panic. The app kept running, which is
  *why* it needs reporting: a boundary high in the tree can hide a component
  dead for weeks behind a plausible "unavailable" panel.
- `handler-panic` — distinct kind, because the phase and blast radius differ:
  a render panic costs a frame, a handler panic can leave state half-applied.

Also exported `core.ReportConcern(kind, detail)` so the render package can file
concerns; callers gate on `IsDebugMode` themselves since detail strings cost a
Sprintf.

## Tests — 12 in `core/error_boundary_test.go`, 6 in `render/panic_test.go`

Coverage: fallback substitution and the stack naming the panicking frame;
`errors.Is` through `Unwrap`; sibling hook-slot stability across passes where
the child's hook count *varies* (3 → 1 → 5 → 0); callback-ID stability with a
varying pre-panic handler count, plus the abandoned handler being purged;
no-latching; a panicking fallback (both while building and while rendering);
the debug detail gate; the concern; absence of phantom drift. Driver side:
last-good-tree retention, initial-render placeholder + recovery, handler-panic
survival, handler liveness across an abandoned pass, the log line, and one
end-to-end test where a failing panel's siblings keep updating through the real
Manager.

### Mutation-checked, not just green

Eight tests were confirmed to fail against a deliberately broken
implementation before being kept:

| break | failure |
|---|---|
| child renders into parent ctx | `parent allocated 9 slots, want 4` |
| no counter rollback | `sibling callback ID moved to "cb_0" (was "cb_2")` |
| no `Cursor = 0` | two phantom cursor-drift concerns |
| unguarded fallback | panic escapes the test binary |
| purge on the abandoned pass | `handler ran 0 times, want 1 — the purge ate a live handler` |
| no dispatch guard | `panic: handler exploded` |
| no initial placeholder | `placeholder node type = ""` |
| no pass guard | `panic: root render exploded` |

Each backup was diffed against the original afterwards to confirm restoration.

Two test-authoring slips worth noting, both caught by the tests themselves:
the stack assertion looked for `grmob/core.boom` but Go inlines the helper into
the test's symbol name (`...boom.func2`), and the abandoned-handler check
probed `cb_n` — an ID that never existed — instead of `cb_1`.

## Files touched

- `core/error_boundary.go` — new
- `core/error_boundary_test.go` — new
- `render/panic_test.go` — new
- `core/event.go` — `counterSnapshot`, `snapshotCounters`, `rollbackCounters`
- `core/debug.go` — `ConcernRenderPanic`, `ConcernHandlerPanic`, `ReportConcern`
- `render/manager.go` — pass guards, `guardHandler`, `panicPlaceholder`,
  `logRenderPanic`
- `wasm/main.go` — guarded event dispatch
- `docs/concepts/error-boundaries.md` — new; `mkdocs.yml` nav entry
- `docs/concepts/debug-mode.md` — two concern rows
- `docs/concepts/architecture.md` — the pass sequence now notes the guard
- `ROADMAP.md` — item moved to Done; debug-mode line updated

Gate: `gofmt` clean, `go vet ./...` clean, full suite passes, `go test -race
./core ./render ./hooks` clean, `GOOS=js GOARCH=wasm go build ./wasm/` clean.
(`examples/todoapp/store.go` remains unformatted — pre-existing, untouched.)

## Backlog

- **`Reset` for the navigation stack** is now the *only* open Core Abstractions
  item.
- Unchanged from last session: theme contrast, `Variant`'s third consumer, and
  the four renderer gaps (iOS flex weights, `AlignItems: "stretch"`, `Image`
  `ContentMode`, native disabled state).
