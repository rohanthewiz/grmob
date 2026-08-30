# Session: Hooks Per-Context State + Lifecycle Ownership

- **Session ID**: `c83e0672-cbd6-4277-b997-27cc36311e32`
- **Session link**: https://claude.ai/code/session_01WYkTemAUD23zXtGj8SVGjW
- **Date**: 2026-08-29
- **Branch**: master
- **Previous session**: `2026-0829-2344-global-state-consolidation.md` (gap 4)

## Goal

The follow-up deliberately left out of gap 4: the `hooks` package still held
package-level globals — `UseEffect`'s unsynchronized deps map with a
never-reset index, and interval/timeout stores keyed by cursor only (two apps
at the same cursor collided; `ClearIntervals` stopped every app's tickers).
It needed a lifecycle-ownership decision: who stops tickers on
`Manager.Close`.

**Outcome: complete, all green** (`go build ./...`, `go vet`,
`go test -race ./...` plus 4x repeat runs of hooks/render/core, wasm target
build, `ios/verify/run.sh` conformance pass). The gomobile bind surface is
untouched (`mobile/bridge.go` unchanged), so the shells need no rebuild.

## Design

Two decisions, both following the gap-4 per-root-pointer pattern:

1. **Hook state lives in the context's own hook slots.** Each hook stores a
   record pointer (`*effectRecord` / `*intervalRecord` / `*timeoutRecord`) in
   the slot it was already reserving via `core.NewState` — identity comes
   from slot position, so per-app AND per-component isolation fall out for
   free and every global map disappears. The records are mutated through the
   pointer, never via `State.Set`, so hook bookkeeping never triggers render
   nudges.
2. **Lifecycle: the Manager owns the app's lifetime.** New per-root
   `cleanupRegistry` on `Context` (`core/cleanup.go`), shared by pointer with
   every derived context like `renderManager`/`registry`/`nav`. Hooks hand
   their stop functions to `ctx.OnClose(fn)`; `render.Manager.Close()` calls
   `m.context.Close()`. `Close` has **drain semantics, not terminal**: it
   runs-and-forgets what's registered so far but stays usable, because the
   WASM host re-mounts over the same shared ctx (RenderInitial called twice)
   and the re-render's hooks must be able to register fresh resources.

## What changed

### `hooks/effect.go` (rewritten)

- Old code was broken outright, not just racy: the global `effectIndex`
  never reset (`ResetEffects` had no callers), so every render minted fresh
  indices and every effect re-ran regardless of deps — the deps comparison
  was dead code. Now: run on mount, re-run on `reflect.DeepEqual` deps
  change, once-ever with no deps. Still `go effect()` on its own goroutine.
- No mutex on the record: it is only touched during render passes, which
  render.Manager serializes. `ResetEffects` deleted.
- Note: `UseEffect` now consumes a cursor slot (it didn't before) — no
  callers outside the repo's own tests existed.

### `hooks/interval.go` (rewritten)

- `UseInterval`: ticker starts on the hook's first render; later renders
  refresh `rec.fn` under the record mutex, so ticks run the **latest
  closure** (current state captures) instead of the mount render's stale
  one. Interval duration is fixed by the first render (documented, matches
  old behavior). Goroutine selects on a `done` channel because
  `Ticker.Stop` never closes `ticker.C` — without it the goroutine parks
  forever. Stop func (`ticker.Stop(); close(done)`) registered via
  `ctx.OnClose`.
- `UseTimeout`: `time.AfterFunc`, fires once per slot lifetime. Two
  deliberate behavior changes, both documented in code:
  - no re-arm after firing (the old store deleted the fired key, so the
    next render re-scheduled it — repeats driven by render cadence);
  - `RequestRender` instead of bare `MarkDirty` on fire, so a timeout with
    no native event in flight reaches the screen via the push channel,
    exactly like interval ticks.
- `ClearIntervals` deleted; global `intervalStore`/`timeoutStore` gone.
- The old cursor-collision bug fixed by slot storage: two components each
  calling `UseInterval` at their own cursor 0 shared the global key
  `"interval-0"` even within ONE app — the second ticker silently never
  started.

### `core/cleanup.go` (new) + `core/context.go`

- `cleanupRegistry` (mutex + fn slice), `ctx.OnClose(fn)`, `ctx.Close()`.
  Cleanups run outside the registry lock (same rule as callback dispatch).
- `cleanup *cleanupRegistry` added to Context's shared-per-root pointers;
  wired in `NewContext`, `NewChildContext`, `WithConfig`, `WithTheme`
  (Scope derives via NewChildContext).

### `render/manager.go`

- `Close()` now also calls `m.context.Close()` inside the `stopOnce`. This
  makes `mobile.Register`'s existing close-on-replace stop the old app's
  tickers with zero bridge changes.

### `wasm/main.go`

- `renderInitial` closes the previous manager (if any) before
  `render.New` — replaces the deleted `hooks.ClearIntervals()` sweep; hooks
  import dropped.

## Tests

- `hooks/hooks_test.go` (new, 8 tests): deps semantics
  (mount/unchanged/changed, no-deps once), four-slot independence across
  two apps, tick-and-stop-on-Close, the two cursor-collision regressions
  (two apps; two scoped components in one app), latest-closure ticks,
  timeout no-re-arm, timeout cancelled by Close. Channel-based sync with
  generous await timeouts + quiet-window assertions.
- `core/cleanup_test.go` (new, 4 tests): run-once + drain, registry shared
  across child/scope/themed/configured contexts, per-root isolation,
  register-after-Close is stopped by the next Close (the re-mount shape).
- `render/push_test.go`: the two `defer hooks.ClearIntervals()` lines
  removed (Manager.Close covers it); new
  `TestManagerCloseStopsHookIntervals` proves the ownership contract
  (drain-until-quiet then require sustained silence, tolerating one
  in-flight tick).

## Gotchas learned

- `core.State[T]` methods have pointer receivers — `core.NewState(...).Get()`
  doesn't compile; bind the return to a variable first.
- `GOOS=js GOARCH=wasm go build ./wasm/` fails with "build output wasm
  already exists and is a directory" — pass `-o <scratch>/grmob.wasm`.

## Next step

All numbered attack-order steps and gaps 1–4 plus the hooks follow-up are
done. Remaining: gap 5's renderer-surface breadth on the Compose/SwiftUI
renderers — images, list virtualization, gestures, accessibility.
