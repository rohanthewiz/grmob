# Session: Element Library Lessons — Plan + Workstream 1 (htmlout on element)

**Session:** https://claude.ai/code/session_016rSW3A4ZS7PwB6dSrGWLEc
**Date:** 2026-08-30

## What this session did

1. Reviewed `~/projs/go/element` against grmob's view layer to extract lessons,
   especially around components.
2. Wrote an adoption plan: `ai_docs/plans/element-lessons-adoption.md`.
3. Implemented **Workstream 1**: rebuilt `htmlout` on the element library, fixing a
   real HTML-escaping/injection bug.

## The comparative review (condensed)

grmob's `View`/`ComponentFunc` pair is structurally element's `Component`/`CompFunc`
(the `http.HandlerFunc` adapter pattern) — convergent design, good sign. Four ideas
worth absorbing, now captured as plan workstreams:

1. **htmlout on element** (DONE this session) — the old exporter wrote user strings
   unescaped; element solves escaping/determinism/pretty-printing already.
2. **`core.Cached` + pointer-equality fast path in `reconcile.Diff`** — element's
   `Cached()` decorator adapted to the retained-tree model; returning the same `*Node`
   lets Diff short-circuit static subtrees. **Verified hard constraint:** the
   callbackRegistry (core/event.go) uses per-pass sequential IDs and purges
   non-re-registered callbacks, so a cached view containing BehaviorProps loses its
   handlers AND shifts every later component's callback IDs. No hooks, no
   BehaviorProps, no post-render mutation inside Cached.
3. **Debug mode** — element's "concerns" tracker, retargeted at hook-framework
   failure modes: cursor drift from conditional `NewState`, duplicate sibling keys,
   callback registration escaping through a cached render.
4. **`components` package** — struct widgets with `View`-typed slot fields
   (`Card{Header, Body, Footer View}`), themed via `ctx.Theme()`, built outside core
   on the public API only (element's dogfooding discipline).

**Not copying:** element's streaming model (grmob must retain a Node tree to diff);
element's inline `func() (x any) {…}()` conditionals (grmob's `If`/`For`/`Match` are
better); builder pooling (premature without profiles).

## Workstream 1 implementation

Test-first, per the plan: wrote `htmlout/export_test.go` escaping tests and confirmed
them failing against the Sprintf exporter — e.g. an input placeholder of
`" onmouseover="pwn()` rendered as a live attribute; `Text("<script>…")` came out as
live markup.

Then rewrote `htmlout/export.go` as a tree-walk over an `element.Builder`:

- `TE()` for user text (content, labels, textarea values) — entity-escaped.
- Attribute values (input value/placeholder, img src) get element's automatic
  quote-escaping — closes attribute breakout.
- Preserved: `tagForType` mapping, `data-onclick`/`data-onchange`/`data-ontoggle`
  callback-attribute contract, style serialization subset, attribute order.
- Output stays pretty (`b.Pretty()`) — only example apps consume htmlout.

Dependency: `github.com/rohanthewiz/element v0.7.0` (direct). v0.7.0 == the local
element repo HEAD, so `TE()` and attr escaping are in the published tag.

### Gotchas hit (worth remembering)

- **`b.Html()` writes `<!DOCTYPE html>` itself** — writing one manually doubles it.
- Used a slice, not a map, for callback-prop → data-attr mapping: map iteration
  order would make exported attribute order nondeterministic.
- `go mod tidy` must NOT run in this repo (drops the x/mobile toolchain pins —
  go.mod comments this). Edited the stale `// indirect` marker on element by hand.
- IDE "element is not in your go.mod" diagnostics were stale; `go list -m` and a
  full build confirm the module resolves.

### Verification

- `go build ./...` clean; `go test -race ./...` green across the repo.
- `examples/layout` export eyeballed: single doctype, escaping visible
  (`Here&#39;s your dashboard`), styles/structure intact.

## State / next steps

- Plan doc: `ai_docs/plans/element-lessons-adoption.md` (Workstream 1 marked DONE
  with notes; sequencing table at the bottom).
- **Next up: Workstream 2** — `core/cached.go` + the `old == new` guard at the top
  of `reconcile.Diff`, with the constraints doc'd on the wrapper; then debug mode
  (WS3), then the components package (WS4, largest).
- Open questions parked in the plan: pretty-vs-compact per call site (resolved
  pretty for now); `components.Tabs` wrap-or-supersede `core/tabview.go`.

## Files touched

- `ai_docs/plans/element-lessons-adoption.md` (new)
- `htmlout/export.go` (rewritten on element)
- `htmlout/export_test.go` (new — escaping + structural tests)
- `go.mod`, `go.sum` (element v0.7.0 direct dependency)
