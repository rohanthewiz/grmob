# Session: Global Mutable State Consolidation (Gap 4)

- **Session ID**: `9f7d2094-de22-4bbe-be49-fb959587a6e6`
- **Session link**: https://claude.ai/code/session_01WYkTemAUD23zXtGj8SVGjW
- **Date**: 2026-08-29
- **Branch**: master
- **Previous session**: `2026-0829-2324-ios-simulator-pass.md` (attack-order step 5 completion)

## Goal

The attack order's five numbered steps in
`ai_docs/plans/grmob-mobile-feasibility-analysis.md` were all complete, so
this session took the next outstanding item from the gaps list: **gap 4,
global mutable state** — "consolidate onto Context, one render mutex, marshal
mutations onto one goroutine."

**Outcome: complete, all green** (`go build ./...`, `go vet`,
`go test -race ./...`, wasm target build, `ios/verify/run.sh` conformance
pass). The gomobile bind surface is byte-identical, so the Android/iOS shells
need no changes and no xcframework/AAR rebuild was required for verification.

## What changed

### Callback registry → per-context (`core/event.go`, rewritten)

- The four package-level handler maps, ID counters, and liveness marks became
  a `callbackRegistry` struct, created once per `NewContext` root and shared
  by pointer with every derived context (`NewChildContext`, `Scope`,
  `WithTheme`/`WithConfig` copies) — same pattern `renderManager` already
  used.
- Registration stays internal (`ctx.registerCallback` etc. — every component
  builder already had `ctx` in scope); dispatch and pass boundaries are now
  exported *methods*: `ctx.TriggerCallback/Text/Bool/Int`,
  `ctx.BeginRenderPass()`, `ctx.PurgeUnusedCallbacks()`,
  `ctx.ReceiveEventPayload()`. The old package-level functions are gone.
- Handlers are invoked **outside** the registry lock (lookup marks liveness
  under the lock, returns the fn). This fixes a latent deadlock: a handler
  that programmatically dispatches another callback used to deadlock on the
  global `callbackMux`.
- Removed the vestigial `Context.callbackMap` / `callbackCounter` /
  `usedCallbacks` fields — written but never read (the analysis flagged
  exactly this: "Context has its own callbackMap that is not the one actually
  used"). Verified unused by grep before removal.

### Navigation stack → per-context (`core/navigation.go`, rewritten)

- `navigatorStack` (unsynchronized package global) became a mutex-guarded
  `navigatorState` on the context tree. `Navigator`/`Push`/`Pop`/`Replace`/
  `Reset` keep their signatures; Pop/Replace preserve the original
  "mark dirty only if something changed" semantics.

### Event dispatch marshaled under the render mutex (`render/manager.go`)

- New `Dispatch{Callback,TextCallback,BoolCallback,IntCallback}(id, val)`
  methods run the handler *and* the follow-up diff render inside one `mu`
  hold (via extracted `renderAgainLocked`). A handler can never interleave
  with a pump render pass; its `State.Set` → `RequestRender` path is async
  (buffered-channel nudge + goroutine hop) so nothing re-enters the mutex.
- `mobile/bridge.go` `Trigger*` functions now delegate to `mgr.Dispatch*` —
  **exported names and signatures unchanged**, so the bind surface and the
  Kotlin/Swift shells are untouched.

### Ripple updates

- `BehaviorProp.Apply(*Node)` → `Apply(*Context, *Node)` (registration needs
  the registry); `layout.go` Column apply-site updated. (Pre-existing quirk
  noted, not fixed: Column applies BehaviorProps into a props map that never
  reaches the returned Node.)
- `wasm/main.go`: dispatches via `ctx.ReceiveEventPayload`.
- Root `main.go` (build-tagged template) and `examples/components.go` moved
  to the Dispatch path.
- `examples/runtime/main.go`: was **already broken at HEAD** (called
  `core.NewRuntime`, which doesn't exist anywhere) and blocked
  `go build ./...`; ported to drive the pass boundary by hand
  (`ctx.BeginRenderPass()` + `core.Render`) with `htmlout`. Verified by
  running it: input event round-trips ("Olá, Ismael").

## Tests

- `core/event_test.go` ported to context-based registry, plus new:
  - `TestContextsHaveIsolatedRegistries` — two apps mint identical
    pass-sequenced IDs yet dispatch only their own handlers; one app's purge
    can't evict the other's.
  - `TestDerivedContextsShareOneRegistry` — scoped/themed subtrees register
    into the root's registry.
  - `TestNestedDispatchDoesNotDeadlock` — the old-global deadlock class.
  - `TestNavigationStackIsPerContext` — Push in app A doesn't navigate app B.
- `render/push_test.go`: async-shape tests now trigger via a kept `ctx`
  handle (dispatch-without-render); new
  `TestConcurrentDispatchAndTimerPushesDoNotRace` storms `DispatchCallback`
  from 3 goroutines against a 5ms interval under `-race`.
- `render/manager_test.go`: event-path tests use `m.DispatchCallback`
  directly (it returns the patches).

## Known follow-up (deliberately out of scope)

The `hooks` package still holds globals: `UseEffect`'s deps map is
unsynchronized and its index never resets per pass (`ResetEffects` has no
callers), and the interval/timeout stores key by cursor only — two apps using
`UseInterval` at the same cursor collide, and `ClearIntervals` stops every
app's tickers. Fixing it properly needs a lifecycle-ownership decision (who
stops tickers on `Manager.Close`), so it should be its own pass.

## Next step

Gap 4 is done. Remaining from the analysis: the hooks follow-up above and the
long-tail renderer-surface breadth (gap 5's widget/style coverage — images,
list virtualization, gestures, accessibility) on the Compose/SwiftUI
renderers.
