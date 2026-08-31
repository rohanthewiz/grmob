# Session: `UseMemo` and `UseReducer`

**Session ID:** session_01Ej9aSqY1yefk9am5LZG367
**Date:** 2026-08-31, ~17:06
**Branch:** master
**Follows:** `2026-0831-1648-roadmap-audit.md`

## Goal

Build the two hooks the previous session's audit moved from "listed as done"
to **In Progress**. This closes the backlog item that had ridden along for
eight sessions.

## `UseMemo[T](ctx, compute, deps...) T` — `hooks/memo.go`

Slot-backed `memoRecord[T]{computed, deps, value}`, seeded through
`core.NewState` exactly as `effectRecord` is. Slot position is the hook's
identity, so the same call site reads the same record every pass and two
components (or two apps) at the same cursor cannot see each other's cache.
No mutex — the record is touched only during render passes, which
`render.Manager` serializes.

Semantics match `UseEffect`'s: recompute when `deps` differ by
`reflect.DeepEqual`, and with **no** deps compute exactly once for the slot's
lifetime.

One deliberate divergence: `compute` runs **inline on the render goroutine**,
not on its own goroutine like `UseEffect`. The result is needed to build this
pass's view, so there is nothing to defer to.

Two things the doc comment records as non-goals:

- **Not a correctness tool.** The same value is handed back on every cache
  hit, so a caller that mutates it corrupts later renders.
- **No `UseCallback` counterpart.** Memoizing a closure only pays off in a
  framework that skips subtrees on unchanged prop identity; here the
  reconciler diffs the rendered tree, so a stable closure buys nothing.

## `UseReducer[S,A](ctx, reducer, initial) (S, func(A))` — `hooks/reducer.go`

The state lives behind the record's **own mutex** rather than directly in the
hook slot. That is the whole design decision, and the reason is that
`core.State` offers atomic `Get` and atomic `Set` but no atomic
read-modify-write:

```
slot-backed State                  reducerRecord
 dispatch: s.Set(r(s.Get(), a))     dispatch: [mu] state = r(state, a)
 two concurrent dispatches can      two concurrent dispatches serialize,
 both read the old state and        so every action is applied exactly
 one update is lost                 once, in some order
```

Sequencing is the entire reason to reach for a reducer over `NewState`, so
the naive form would defeat the hook.

Consequences that fell out of that choice, each documented at the call site:

- The reducer runs **under the lock**, so it must not dispatch — a re-entrant
  dispatch deadlocks on the non-reentrant mutex.
- Because the state hangs off a pointer, the **slot value never changes**, so
  `State.Set` — the usual carrier of the render request — is never called.
  `dispatch` therefore calls `ctx.RequestRender()` itself. Without that line a
  dispatch would update state invisibly until some unrelated event triggered
  the next pass.
- `initial` is evaluated every render but only the mount value is kept, same
  as `NewState`.

## The deps-aliasing bug, found while writing `UseMemo`

`UseEffect` stored the caller's variadic `deps` slice directly. Normally the
compiler allocates a fresh slice per call, so it never bit — but a caller who
spreads a slice they own (`UseEffect(ctx, fn, args...)`) hands over an alias.
A later mutation then rewrites the *stored* deps in lockstep with the new
ones, `DeepEqual` reports "unchanged", and **the effect never runs again**.

`UseMemo` was written with `rec.deps = append([]any(nil), deps...)` from the
start; on the user's go-ahead the same one-liner went into `UseEffect`, since
an inconsistency between two hooks in one package is worse than the fix.

## Tests — `hooks/memo_reducer_test.go` (7) + 1 in `hooks_test.go`

`UseMemo`: recompute-on-change including the **change-back** case (the record
holds one generation of deps, not a history); depless compute-once; slot
independence across two memos in one component and a second app; the
deps-aliasing guard.

`UseReducer`: dispatch updates state, marks the tree dirty, and ignores a
changed `initial` on re-render; **200 concurrent dispatches lose nothing**;
slot independence across apps.

`UseEffect`: `TestUseEffectDoesNotAliasCallerDeps`, mirroring the memo's.

### Mutation-checked, not just green

Each of the three subtle tests was confirmed to fail against a deliberately
broken implementation before being kept:

| break | failure |
|---|---|
| `rec.deps = deps` in `UseMemo` | `value 1 after 1 computes, want 2 after 2` |
| dispatch split into unlocked read → compute → locked write | `state after 200 concurrent increments = 199, want 200` |
| `rec.deps = deps` in `UseEffect` | timed out waiting for the re-run |

A test that cannot fail is not evidence. These three assert the exact
properties that motivated the implementation choices, so they are the ones
worth proving.

## Files touched

- `hooks/memo.go` — new
- `hooks/reducer.go` — new
- `hooks/memo_reducer_test.go` — new
- `hooks/effect.go` — deps copy
- `hooks/hooks_test.go` — aliasing regression test
- `docs/concepts/state-and-hooks.md` — two new hook sections; the section
  preamble now reads "time, side effects, and derived state"
- `ROADMAP.md` — both hooks joined the Done hooks line; the In Progress entry
  removed

Gate: `gofmt` clean, `go vet ./...` clean, full suite passes, `go test -race
./hooks` clean. (`examples/todoapp/store.go` is unformatted but was already
so — untouched.)

## Backlog

- **`Reset` for the navigation stack** and **error boundaries** are now the
  only two open Core Abstractions items.
- Theme contrast, `Variant`'s third consumer, and the four renderer gaps
  (iOS flex weights, `AlignItems: "stretch"`, `Image` `ContentMode`, native
  disabled state) — all tracked in `ROADMAP.md` since the last session.
