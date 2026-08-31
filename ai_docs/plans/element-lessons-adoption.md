# Adopting Lessons from the Element Library

**Status:** Complete — all four workstreams landed 2026-08-30
**Date:** 2026-08-30
**Source:** Comparative review of `github.com/rohanthewiz/element` (../element) against grmob's view layer.

## Background

grmob's `View` / `ComponentFunc` pair (core/view.go) is structurally the same design as
element's `Component` / `CompFunc` — the `http.HandlerFunc` adapter pattern applied to
rendering. The review found four ideas in element that grmob has not yet absorbed, plus
one latent bug (unescaped HTML output) that adopting element directly fixes.

What we are deliberately **not** copying: element's streaming model (writing bytes
immediately, using Go argument-evaluation order as tree traversal). grmob must retain a
`*Node` tree so the reconciler can diff it — build-then-diff stays. Element's inline
`func() (x any) {…}()` conditional idiom is also worse than grmob's `If`/`For`/`Match`,
and builder pooling is premature until per-frame allocation shows up in a profile.

The four workstreams below are ordered by value-to-effort. Each is independently
shippable; none depends on another except where noted.

---

## Workstream 1 — Rebuild `htmlout` on element (fixes an escaping bug)

**Priority: highest.** This is a correctness fix, not just a refactor.

**Status: DONE (2026-08-30).** Escaping tests written first and confirmed failing
against the Sprintf exporter (live attribute injection via input value/placeholder/img
src, live markup via Text content). Rewritten on element v0.7.0 (which ships TE() and
attribute quote-escaping); pretty output chosen since only example apps consume it.
Note: `b.Html()` emits the doctype itself — don't write one manually.

### Problem

`htmlout/export.go` hand-assembles HTML with `fmt.Sprintf` and writes text content and
attribute values (`content`, `value`, `placeholder`, `src`, button labels) completely
unescaped. `core.Text("<script>…")` or a double-quote in an input value produces broken
or injectable HTML. Element already solves escaping (quote-escaping in attribute values
automatically; full `html.EscapeString` for text via `TE()`), deterministic attribute
order, single-pass buffered output, and pretty-printing.

### Plan

1. Add the dependency: `go get github.com/rohanthewiz/element` (zero transitive deps,
   so it does not bloat go.mod). Do NOT let `go mod tidy` drop the x/mobile pins —
   go.mod comments this constraint.
2. Rewrite `renderNode` as a tree-walk over an `element.Builder`:
   - Keep the existing `tagForType` mapping and the per-type special cases
     (Input/TextArea/Checkbox/Image/Text/Button/Spacer/CameraView) — the switch
     structure is fine; only the output mechanism changes.
   - Text content and labels: `b.Span(...).TE(content)` — `TE`, not `T`, so user
     content is entity-escaped.
   - Attributes: pass as attr pairs to the builder (`b.Ele("input", "type", "text",
     "value", val, ...)`); element quote-escapes attribute values.
   - Keep emitting the `data-onclick` / `data-onchange` / `data-ontoggle` callback-ID
     attributes so the WASM runtime contract is unchanged.
   - `styleAttr` stays as-is (it only serializes our own Style struct — trusted
     values) but feed its result through the builder as a `style` attribute rather
     than string-concatenating into the tag.
3. Indented output: element writes compact HTML; use `b.Pretty()` if human-readable
   output should be preserved. Decide per call site — compact is correct for the WASM
   runtime, pretty for the export/demo path.
4. Tests:
   - Escaping tests: `Text("<b>&\"")`, input value with quotes, image src with quotes —
     assert entity-escaped output. These fail against the current implementation;
     write them first.
   - Golden-ish structural tests for each node type (don't byte-compare against the
     old output — whitespace will differ; assert via substring/parse checks).

### Acceptance

- `go test ./htmlout/...` green, including new escaping tests.
- `go build ./...` and existing example apps still export sensible HTML.

---

## Workstream 2 — `core.Cached` + pointer-equality fast path in the reconciler

**Priority: high, small surface.** Element's `Cached()` (component_cached.go) wraps a
component with `sync.Once` and replays bytes. grmob's version is *more* valuable
because the reconciler can exploit it.

**Status: DONE (2026-08-30).** `core/cached.go` + the `old == new` guard at the top of
`reconcile.Diff` (it subsumes the old both-nil check). Node immutability contract
documented on `core.Node`. Guard proven live, not just passing: the fast-path tests use
a func-valued prop as a sentinel (`reflect.DeepEqual` reports non-nil funcs unequal
even against themselves), so they fail if Diff falls through to the deep walk —
verified by stashing the guard. Living documentation: `appHeader` in
examples/mobileapp, a package-level `core.Cached` var (constructed once — a Cached
built inside a render body is a fresh wrapper each pass and caches nothing; this
usage constraint is documented on Cached alongside the no-hooks/no-callbacks ones).
Debug bypass left as a comment hook in cachedView.Render for Workstream 4.

### Design

```go
// core/cached.go
type cachedView struct {
    view View
    once sync.Once
    node *Node
}

func Cached(view View) View { return &cachedView{view: view} }

func (c *cachedView) Render(ctx *Context) *Node {
    c.once.Do(func() { c.node = c.view.Render(ctx) })
    return c.node
}
```

Returning the **same `*Node` pointer** every pass lets `reconcile.Diff` short-circuit:

```go
// at the top of Diff, after the nil checks
if old == new { // same pointer: subtree is cached and unchanged
    return nil
}
```

This turns a static nav bar / footer from "re-render + re-diff every pass" into a no-op.

### Constraints (must be documented on `Cached` and enforced where cheap)

These come from differences between grmob and element that element never had to face:

1. **No hooks inside a cached view.** `NewState`/`UseChildContext` advance the parent
   context's cursor; a view that renders on pass 1 but not pass 2 would shift every
   later slot. Cached views must be pure functions of their construction arguments.
2. **No BehaviorProps inside a cached view — hard constraint, verified.** The
   callbackRegistry (core/event.go) assigns IDs from per-pass sequential counters and
   purges any callback not re-registered in the latest pass. A cached subtree with
   callbacks breaks twice over: its pass-1 IDs are purged after the first pass it
   skips, and by no longer consuming counter slots it shifts the callback IDs of
   every component registered after it — invalidating handlers well outside the
   cached subtree. Debug mode (Workstream 4) should detect a callback registration
   escaping through a cached render and flag it as a concern.
3. **Nodes are mutable.** The contract is: nothing mutates a node after render.
   Reconcile only reads; renderers must also only read. State this in the Node doc
   comment as part of this workstream.
4. **Debug bypass:** element re-renders fresh in debug mode so concerns are tracked.
   grmob has no debug mode yet — leave a hook point (`if debugMode { return
   c.view.Render(ctx) }`) wired to Workstream 4's flag.

### Steps

1. Add `core/cached.go` with the wrapper and a thorough doc comment covering the
   constraints above.
2. Add the pointer-equality early return in `reconcile.Diff` (one guard + comment).
3. Tests:
   - Cached view renders exactly once across passes (counter in the wrapped view).
   - `Diff(same, same, path)` returns no patches and does not recurse (cover via a
     deep cached subtree + patch count).
   - Concurrency: two goroutines calling Render race-free (`go test -race`).
4. Use it in one example app (e.g. a static header in examples/mobileapp) as living
   documentation.

### Acceptance

- `go test -race ./core/... ./reconcile/...` green.
- Diff of a tree whose root children include a cached subtree emits zero patches for
  that subtree.

---

## Workstream 3 — A `components` package of struct widgets with View slots

**Priority: medium, largest surface.** Element's components library (accordion, card,
tabs, modal, pagination, form fields…) shows the idiom: a struct with named config
fields, where a `Component`-typed field is a *slot* (`Card{BodyComponent: …}`).

**Status: DONE (2026-08-30).** `components/` ships Card (Title simple path,
Header slot wins when both set), Badge, Chip, FormField (Error replaces Hint),
Accordion (the one hook-owning widget — its doc comment carries the render-
unconditionally rule and notes debug mode catches violations), and Tabs. The
tabview open question resolved as **wrap**: core.TabView is the "TabView"
node-type wire contract the native renderers consume, so components.Tabs is a
named-field facade delegating to it — one implementation to keep in sync.
All widgets are theme-styled (no hard-coded colors; Badge inks with
Colors.Background on Primary, Chip's selected default is Surface/Primary) and
accept Style/SelectedStyle overrides. Ergonomics + acceptance proven by the
todoapp filter bar rewritten on components.Chip: chip_migration_test.go keeps
the pre-extraction bar verbatim and asserts the Workstream-1 exporter emits
byte-identical HTML for all three active states (fresh contexts per render, so
per-pass callback IDs line up and any diff is structural). Chip owns appending
", selected" to the accessibility label. Per-widget focused tests throughout;
the package compiles against public core API only.

### Why structs, not more funcs in core

- Named fields scale to many optional knobs where positional args do not.
- `View`-typed fields give natural composition slots (`Card{Header: …, Body: …}`).
- Building the package **outside core, on the public API only**, is element's
  dogfooding discipline: it proves the core primitives are sufficient, and keeps core
  from growing widget-by-widget (modal.go, tabview.go, toast.go are already in core —
  leave them; new widgets go here).

### Plan

1. Create `components/` package. Every widget is a struct implementing `core.View`,
   styled through `ctx.Theme()` (never hard-coded colors — follow the theme-base
   pattern `containerNode` uses).
2. First tranche, ported/adapted from element's set where they make sense on mobile:
   - `Card{Header, Body, Footer core.View; Title string; Style …}` — Title as the
     simple path, Header slot as the escape hatch (mirrors element's Body vs
     BodyComponent).
   - `Badge`, `Chip` (the todoapp filter bar is a hand-rolled chip — extract it).
   - `FormField{Label, Hint, Error string; Input core.View}` — element's form_field
     is its most-used component; the label/hint/error wrapper pattern transfers
     directly.
   - `Accordion`, `Tabs` (evaluate against core/tabview.go first — wrap or supersede,
     don't duplicate).
3. Each widget gets a focused test (element does this per-component — keep that).
4. Rewrite one example app section (todoapp filter bar → `Chip`) to prove ergonomics.

### Acceptance

- `components/` compiles against public core API only (no internal access).
- Per-widget tests green; todoapp still renders identically (diff its exported HTML
  before/after, using the Workstream-1 exporter).

---

## Workstream 4 — Debug mode: catch the hook-framework failure modes

**Priority: medium, research-flavored.** Element tracks "concerns" (odd attribute
pairs, unbalanced elements) and tags debug output with `data-ele-id`. grmob's
equivalent class of silent bugs is worse because hooks are positional:

**Status: DONE (2026-08-30), checks 1–2 + Cached bypass; check 3 sketched.**
`core/debug.go`: `SetDebugMode`/`IsDebugMode` (process-wide atomic, mirroring
element), a deduplicating concerns collector (`Concerns()` for tests,
`DumpConcerns()` for humans, count-per-finding so per-pass repeats don't flood),
and `Context.EndRenderPass()` — called from all three of render.Manager's pass
methods — which audits every context (children, slot-held contexts, scopes).
Cursor drift is two comparisons: `0 < cursor < len(slots)` catches a skipped
hook; cursor-vs-last-rendered-pass (both nonzero) catches the growth direction,
where the appended slot makes the counts line up. The nonzero guards are what
keep navigated-away Scopes (legitimately 0-cursor passes) from false-positiving
— covered by an explicit test. Duplicate keys are checked in `renderAll`, now
the single choke point (Scroll's hand loop folded into it; all call sites pass
the container type so findings are locatable). The Cached debug bypass renders
fresh and *measures* the two constraint violations by sampling the parent
cursor and the registry's per-pass counters (new `registrationCount()`) around
the render — so hook/callback escapes are reported directly rather than
exhibited as downstream corruption. Provenance (check 3) is sketched in a
comment in debug.go: deferred because anonymous ComponentFuncs all name as
`core.Row.func1`, so stamping is only useful once named user components are
common.

1. **Cursor drift:** a `NewState` called conditionally or in a loop shifts every later
   slot — today this manifests as mysterious state bleed. Detect: record slot count
   per context after each pass; a pass that ends with a different cursor than the
   slot length (or than last pass) is a hard warning.
2. **Duplicate keys:** two siblings with the same `Key` defeat keyed reconciliation.
   Detect: in debug mode, validate key uniqueness per sibling list during
   `renderAll`/`containerNode`.
3. **Provenance:** annotate nodes with the component that produced them (a
   `debugSource` field or Props entry), surfaced in htmlout as a data attribute —
   element's `data-ele-id` idea.

### Plan

1. `core.SetDebugMode(bool)` + `IsDebugMode()` (mirror element's naming; process-wide
   flag is fine to start).
2. Implement checks 1 and 2; route findings through a concerns collector like
   element's (`concerns.UpsertConcern`) so tests can assert on them, with a
   `DumpConcerns()` for humans.
3. Wire the `Cached` debug bypass from Workstream 2.
4. Provenance (check 3) is a stretch goal — sketch only, implement if cheap.

### Acceptance

- A test that renders a view with a conditional `NewState` sees a cursor-drift
  concern; a test with duplicate sibling keys sees a duplicate-key concern.
- Zero overhead when debug mode is off (guard checks behind the flag; benchmark if
  in doubt).

---

## Suggested sequencing

| Order | Workstream | Size | Why here |
|-------|-----------|------|----------|
| 1 | htmlout on element | S | Fixes a real escaping bug; independent |
| 2 | Cached + diff fast path | S | High leverage for the reconciler; independent |
| 3 | Debug mode (checks 1–2) | M | Unblocks safe growth of 3; wires Cached bypass |
| 4 | components package | L | Biggest surface; benefits from 1 (HTML diffing in tests) and 3 (key validation) |

Each workstream ends with `go build ./... && go test -race ./...` and a commit.

## Open questions (resolve during implementation, not before)

- **Pretty vs compact htmlout** per call site (Workstream 1, step 3).
- **tabview.go vs components.Tabs** (Workstream 3): wrap or supersede.
