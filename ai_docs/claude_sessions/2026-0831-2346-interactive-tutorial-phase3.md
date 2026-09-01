# Interactive Tutorial — Phase 3: Chapter 3 (Hooks & Effects)

Session: https://claude.ai/code/session_01Nm3azgkKS6SWsFoqJZEdUh

## Goal

Phase 3 of the eight-phase interactive tutorial: add Chapter 3
"Hooks & Effects" to `examples/tutorial`. As in Phase 2, the framework
needed exactly one change — appending `chapter3()` to `Chapters` in
lesson.go — and IDs 3.x, home rows, progress, and the Next/Prev walk
(now 15 lessons) picked the chapter up automatically.

## What was built

- **chapter3.go** — five lessons, each with a live demo. Through-line:
  render functions stay pure; the hooks package is where time, side
  effects, derived values, and multi-step updates live — all slot-backed,
  so the rules of hooks apply to every one of them.
  - **3.1 The clock: UseInterval** — 1s ticker counting elapsed seconds
    ("N s"), pause checkbox that works purely because ticks run the
    latest closure. Leaving a lesson *stops* the clock: the Navigator
    drops the frame's disposable scope, which closes it (verified in
    core/navigation.go `dropScope` → `cleanup.close()`).
  - **3.2 Once, later: UseTimeout** — "⏰ Right on time" card appears once,
    2.5s after mount (`timeoutRevealDelay`); a "Poke a render" button
    proves re-renders never re-arm a fired slot. Reopening the lesson
    re-arms it — fresh stack frame, fresh slot.
  - **3.3 Effects: UseEffect** — simulated 350ms fetch
    (`effectFetchDelay`) keyed to a gopher SegmentedControl (deps
    re-run), plus a no-deps mount-once effect. Two idioms called out:
    capture dep values at render time (the effect runs later), and derive
    loading state (`got.id != sel`) instead of clearing it in the handler.
  - **3.4 Caching: UseMemo** — filter over a 12-word corpus with a
    compute-call meter and a "Re-render (changes no dep)" ghost button.
    The meter mutates through a stable pointer (`computeMeter`) instead
    of Set — the demo's one deliberate impurity, documented as write-only
    instrumentation that cannot request renders.
  - **3.5 Actions: UseReducer** — score keeper with struct state
    `{score, moves}` stepping both fields per action atomically — the
    smallest honest case for a reducer over two Sets that could tear.
- **chapter3_test.go** — five liveness tests plus one new primitive:
  `awaitText(t, mgr, sub)` polls the rendered tree (25ms cadence, 5s
  deadline that only bounds a hang) until text appears. Polling
  re-renders keeps every pass under debug mode's audit. Reuses
  `typeInto`/`tap`/`toggleCheckbox` unchanged. The effect test asserts
  the negative too: re-selecting the same gopher re-renders without
  re-fetching (settle-sleep of `effectFetchDelay` + margin, then check
  the count held).
- **lesson.go** — `chapter3(),` appended to `Chapters`.

## Verification

- `go test ./...` fully green; gofmt and vet clean; TestMain debug mode
  audits every pass.
- `go test -race -count=2 ./examples/tutorial/ ./hooks/` clean — matters
  because the effect demo writes state from concurrent goroutines.
- `GOOS=js GOARCH=wasm go build -o <file> ./wasm` compiles (bare
  `go build ./wasm` still fails on the wasm/ directory name).

## Facts learned/confirmed this phase

- `render.Manager.requestRender` is a non-blocking nudge (buffer-of-1,
  extras dropped), so background timers from visited lessons can never
  stall tests or the app.
- Navigator frames use `ctx.disposableScope(routeScopeKey(id))`; Pop/
  Replace/Reset retire the frame and the next render pass calls
  `dropScope`, which closes the sub-registry — tickers stop, pending
  timeouts cancel, hook state is genuinely discarded. This is why lesson
  demos reset per visit "by construction".
- Effects/timeouts firing after their scope is dropped are harmless:
  their Sets land in orphaned slots and the RequestRender renders the
  current tree.
- Every `tree(t, mgr)` call in tests is itself a full audited render
  pass — which is precisely what makes the UseMemo cache-hit assertions
  meaningful.

## Next session: Phase 4

Per the eight-phase plan the next chapter is navigation/composition
territory — check the phase list in the Phase 1 session doc
(2026-0831-2047) for the exact scope. Likely `chapter4()` covering
Navigator (Push/Pop/Replace, per-frame scopes — the tutorial itself is
the demo), Scope/UseChildContext, and possibly theming. Source of truth:
docs/concepts/navigation.md and styling-and-theming.md.
