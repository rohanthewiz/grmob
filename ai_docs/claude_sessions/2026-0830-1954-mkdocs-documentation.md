# Session: MkDocs documentation set

**Session ID:** session_016rSW3A4ZS7PwB6dSrGWLEc (continuation)
**Date:** 2026-08-30, ~19:54
**Branch:** master

## Goal

Create a new MkDocs-style documentation set for grmob, keeping
`docs/tutorial-todo.md` untouched. The user views docs with
rohanthewiz/gkdocs but they must remain compatible with any mkdocs reader.
Mermaid diagrams instead of ASCII charts.

## What landed

### New structure

- **`mkdocs.yml`** at repo root — material theme config, mermaid via the
  canonical `pymdownx.superfences` custom-fence config, full nav.
- **`docs/index.md`** — landing page: pitch, minimal counter example,
  pipeline overview diagram, feature list, "where to go next" table.
- **`docs/getting-started.md`** — install, first app (mobile.Register +
  AppName bindable-symbol note), the three feedback loops in speed order
  (test-drive via render.Manager → wasm preview → device builds), render
  loop sequence diagram.
- **`docs/concepts/`** (8 pages):
  - `architecture.md` — package map, pass boundary sequence
    (BeginRenderPass → Reset → render → EndRenderPass → Diff → purge),
    Node immutability contract, the two delivery paths (sync event return
    vs async push with coalescing), positional identity, lifecycle/Close.
  - `views.md` — View/ComponentFunc, container argument contract,
    leaf-widget table, conditionals (with the "conditionals hide views,
    not hooks" warning), For/Keyed, Navigator (incl. core.Reset vs
    ctx.Reset disambiguation), composition patterns.
  - `state-and-hooks.md` — slot/cursor model, rules of hooks with a
    drift diagram, state-lives-high + immutable updates, Scope vs
    UseChildContext, hooks pkg (UseEffect deps semantics, UseInterval
    latest-closure note, UseTimeout), OnClose/Close, persistence pointer
    (bytdb + SetDataDir).
  - `events.md` — four callback kinds/ID namespaces, ID lifecycle
    sequence diagram (beginPass → register → purge → dispatch), stability
    granularity note, dispatch paths per host, Cached constraint.
  - `styling-and-theming.md` — StyleProps, accessibility props, UseStyle
    **subset-merge caveat** (can't zero fields; Padding(0) assigns),
    Theme structure, resolution order, theme-base override pattern,
    Transition.
  - `reconciliation.md` — Patch type, ordering rules
    (highest-index-first removals), full Diff decision flowchart, pointer
    fast path, DeepEqual rationale for props/styles, keyed-slot semantics
    and the no-move-patches-yet honesty, cost-model table.
  - `caching.md` — Cached usage, construct-once rule, the four
    constraints, debug enforcement, "profiling response not default" tip.
  - `debug-mode.md` — concerns API, check table, cursor-drift decision
    flowchart, Scope non-flagging rationale, Cached bypass
    (reported-not-exhibited), test pattern asserting zero concerns.
- **`docs/components.md`** — struct-widget idiom, all six widgets with
  usage, Accordion hook warning, Tabs wrap-don't-supersede rationale,
  "writing your own" checklist including htmlout equivalence pinning.
- **`docs/platforms/`** (3 pages):
  - `native.md` — bridge contract sequence diagram, delivery guarantee,
    bridge function table, Android/iOS build steps (build.sh args,
    Xcode/NDK prereqs), SetDataDir timing (init runs before it → lazy
    store open), test-first workflow.
  - `wasm.md` — build command, GrMobWASM function table,
    GrMobApplyPatches push vs IsDirty polling, data-onclick attribute
    wiring, GrMobRequestPermission, same-engine note.
  - `exporters.md` — htmlout (escaping guarantees, determinism,
    equivalence-pinning test pattern), jsonout, which-to-use table.

### Removed (superseded)

- `docs/reconciliation.md` and `docs/ui-architecture.md` — 2025-era
  paper-style docs that had drifted from the code (`ctx.SetTheme`,
  `StyleContext`, int-typed Style fields, none of which exist). Their
  accurate content was folded into `concepts/reconciliation.md` and
  `concepts/styling-and-theming.md`. Git history preserves the originals.

### Untouched

- `docs/tutorial-todo.md` — kept verbatim per instruction; wired into nav
  as "Tutorial — Todo App".

## Verification

- All internal `.md` links across the new pages resolve (checked with a
  walk script; one false positive from Go generics syntax in a code fence).
- Mermaid diagrams use standard flowchart/sequenceDiagram syntax.

## Notes / risks

- `mkdocs.yml` uses the canonical mkdocs-material mermaid config, which
  includes the `!!python/name:` YAML tag; a strict non-Python YAML parser
  could choke on that one line. If gkdocs does, drop the `format:` line —
  gkdocs likely renders ```mermaid fences natively anyway.
- Docs were written against the current API surface verified in-source
  this session (reconcile/patch.go, mobile/bridge.go, hooks, wasm/main.go,
  build scripts) — including the UseStyle subset-merge caveat and the
  navigation Reset naming collision, both easy future doc-rot spots.
