# Session: Element Lessons — Workstream 2 (core.Cached + Diff fast path)

**Session:** https://claude.ai/code/session_016rSW3A4ZS7PwB6dSrGWLEc
**Date:** 2026-08-30
**Commit:** bcfe717

## What this session did

Implemented **Workstream 2** of `ai_docs/plans/element-lessons-adoption.md`:
`core.Cached` plus the pointer-equality fast path in `reconcile.Diff`. Committed as
`bcfe717`; plan doc updated with a DONE status block.

## Implementation

- **`core/cached.go` (new):** `Cached(view View) View` wraps a view with `sync.Once`
  and replays the same `*Node` every pass. Doc comment carries the constraints:
  1. No hooks (`NewState`/`UseChildContext`/`hooks.*`) — positional slot-cursor drift.
  2. No callbacks/BehaviorProps — callbackRegistry uses per-pass sequential IDs +
     purge; a cached subtree skipping registration breaks its own handlers AND shifts
     IDs for every component registered after it.
  3. No per-pass value dependence (theme/config are baked in at first render).
  4. Node immutability (framework-wide contract; Cached raises the stakes).
  Debug-mode bypass left as a comment hook in `cachedView.Render` for the debug
  workstream (`if IsDebugMode() { return c.view.Render(ctx) }`).
- **`reconcile/patch.go`:** `if old == new { return nil }` at the top of `Diff`. It
  subsumes the previous both-nil guard (removed).
- **`core/node.go`:** `Node` doc comment now states the frozen-after-render contract:
  reconciler only reads, renderers must only read; pointer equality is the
  reconciler's "unchanged" evidence.
- **`examples/mobileapp/app.go`:** package-level
  `var appHeader = core.Cached(core.Text("GrMob Demo", …))` above the TabView
  (App restructured: `SafeArea(Column(appHeader, tabs(tab)))` with TabView extracted
  to `tabs(tab core.State[int])`). Living documentation; deliberately inert.

## Key discovery beyond the plan

**`render.Manager` calls the root function every pass** (`r.renderFunc(r.context)` in
`renderAgainLocked`), so a `Cached` constructed inside a render body is a fresh
wrapper each pass and caches nothing. Cached views must be constructed once —
package-level var or equivalent. This construct-once constraint is documented
prominently on `Cached` and in the mobileapp example comment.

## Testing technique worth remembering

To prove the identity guard actually fires (rather than the deep walk merely finding
equality): plant a **func-valued prop as a sentinel** — `reflect.DeepEqual` reports
non-nil funcs unequal *even against themselves*, so without the guard `Diff(n, n)`
emits a spurious update-props. Both new reconcile tests were verified to fail with
the guard stashed (`git stash push reconcile/patch.go`) and pass restored.

Tests added:
- `core/cached_test.go`: renders-exactly-once across passes (same pointer each time);
  16-goroutine concurrent `Render` under `-race` (uses `atomic.Int32` and
  `wg.Go` — lint prefers these over `atomic.AddInt32`/manual `wg.Add`).
- `reconcile/patch_test.go`: `TestDiffSamePointerShortCircuits` (sentinel),
  `TestDiffCachedSubtreeEmitsNoPatches` (shared subtree pointer inside two
  fresh-allocated parent trees — the exact consecutive-render shape — emits only the
  dynamic sibling's patch).

## Verification

- `go build ./...` clean; `go test -race ./...` green across the repo.
- mobileapp bridge tests unaffected (they find nodes by type, not positional path).

## State / next steps

- Workstreams 1 (htmlout on element) and 2 (this) DONE in the plan doc.
- **Next per the plan's sequencing table: debug mode** (plan's Workstream 4, ordered
  3rd) — `core.SetDebugMode`, cursor-drift + duplicate-sibling-key concern checks,
  concerns collector, and wiring the Cached debug bypass. Then the components
  package (largest).

## Files touched

- `core/cached.go`, `core/cached_test.go` (new)
- `core/node.go` (Node immutability doc)
- `reconcile/patch.go` (fast path), `reconcile/patch_test.go` (guard-proving tests)
- `examples/mobileapp/app.go` (cached header)
- `ai_docs/plans/element-lessons-adoption.md` (WS2 status)
