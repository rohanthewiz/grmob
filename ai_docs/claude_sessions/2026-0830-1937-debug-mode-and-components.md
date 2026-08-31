# Session: Debug mode + components package (Workstreams 4 and 3)

**Session ID:** session_016rSW3A4ZS7PwB6dSrGWLEc
**Date:** 2026-08-30, ~19:37
**Branch:** master
**Commits:** `9da66c8` (debug mode), `d8e25be` (components package)

## Goal

Finish the element-lessons adoption plan (`ai_docs/plans/element-lessons-adoption.md`).
Workstreams 1 (htmlout on element) and 2 (core.Cached + Diff fast path) were done in
prior sessions; this session landed the remaining two in the plan's sequenced order:
Workstream 4 (debug mode) first because it unblocks safe growth of Workstream 3
(components package). The plan is now **complete** and its doc updated with DONE notes.

## Workstream 4 — Debug mode (`9da66c8`)

### What landed

- **`core/debug.go`** — `SetDebugMode(bool)` / `IsDebugMode()` on a process-wide
  `atomic.Bool` (mirrors element's DebugSet/IsDebugMode; comment documents the
  move-to-Context escape path if two apps ever need different settings). A
  deduplicating concerns collector keyed by (Kind, Detail) with a per-finding
  Count — checks run every pass, so a persistent bug bumps a count instead of
  flooding. API: `Concerns()` (sorted snapshot for tests), `ClearConcerns()`,
  `DumpConcerns()` (human-readable, `×N` counts). Concern kinds:
  `ConcernCursorDrift`, `ConcernDuplicateKey`, `ConcernCachedHooks`,
  `ConcernCachedCallbacks`.

- **Cursor drift (check 1)** — `Context.EndRenderPass()`, called by
  `render.Manager` after the tree render in all three pass methods
  (`RenderInitial`, `renderAgainLocked`, `RenderAndGetPatches`), so every host
  (WASM included — it drives through Manager) gets it. `auditCursor` recurses
  the context tree (children, slot-held child contexts, scopes — Reset's
  traversal, same lock discipline for the slots snapshot) with two comparisons:
  - `0 < Cursor < len(slots)` → a hook that allocated trailing slots on an
    earlier pass was skipped this pass (slots only grow, so cursor can never
    exceed them).
  - `Cursor != debugLastCursor`, both non-zero → hook count varies between
    passes; catches the growth direction, where the appended slot makes
    `cursor == len(slots)` line up.
  - The non-zero guards are what stop false positives from navigated-away
    `Scope`s (legitimately 0-cursor passes); `debugLastCursor` is never
    overwritten with 0, so a returning scope is judged against its last
    *rendered* pass. Two new fields on Context (`debugLastCursor`,
    `debugPassSeen`), unlocked because EndRenderPass runs under the manager's
    pass serialization.

- **Duplicate keys (check 2)** — `checkDuplicateKeys` runs in `renderAll`,
  which is now the single child-render choke point: its signature gained a
  `parentType string` (all call sites updated: containerNode passes `typ`,
  Fragment/For/Modal/TabView/CameraView pass their names) and Scroll's
  hand-rolled render loop was folded into it. Empty keys never collide.

- **Cached debug bypass (check from Workstream 2's hook point)** —
  `cachedView.Render` bypasses the cache under debug mode and `debugRender`
  *measures* the two constraint violations by sampling around the fresh
  render: parent `ctx.Cursor` advance → `ConcernCachedHooks`; registry
  per-pass counter advance (new `callbackRegistry.registrationCount()`) →
  `ConcernCachedCallbacks`. Comment notes the deliberate debug/production
  behavior difference: under the bypass the drift is *reported*, not
  *exhibited* — "an app that only misbehaves with debug off has likely
  tripped exactly these concerns."

- **Provenance (check 3)** — sketch only, per plan: a comment block in
  debug.go describes stamping `Props["debugSource"]` via
  `runtime.FuncForPC`, and why it's deferred (anonymous ComponentFuncs all
  name as `core.Row.func1`, so it's only useful once named user components
  are common).

- **Tests** (`core/debug_test.go`): skipped-hook drift, grown-hook drift
  (cross-pass check), unvisited-Scope non-drift, duplicate keys via Column
  and via For, Cached bypass (fresh pointers + both escape concerns + clean
  view silent), debug-off records nothing, dedup counts across passes.

## Workstream 3 — `components` package (`d8e25be`)

### What landed

Struct widgets implementing `core.View` with named fields where a View-typed
field is a slot; public core API only; themed via `ctx.Theme()` with Style
overrides; per-widget focused tests. `components/doc.go` carries the idiom +
discipline + hooks-in-widgets rules.

- **Card** — `{Title, Header, Body, Footer, Style}`. Title is the simple path
  (Subtitle typography + Bold — "a card heading is a section heading");
  Header slot wins when both set (mirrors element's Body vs BodyComponent).
  Renders on `core.Card` so the theme base applies.
- **Badge** — non-interactive pill. Defaults: `Colors.Primary` background,
  `Colors.Background` ink (theme-derived, no hard-coded white), Caption font
  size, radius 999 for stadium shape.
- **Chip** — selectable pill, controlled (no internal state). `Style` always,
  `SelectedStyle` on top when selected (nil → theme default
  Surface bg / Primary ink). Appends ", selected" to AccessibilityLabel.
- **FormField** — `{Label, Hint, Error, Input, Style}` on a Column with
  `Padding(0)` (assigns, not merges — this is how the theme Column's screen
  padding gets zeroed) + `Gap(Spacing.XS)`. Label = Caption size + TextPrimary
  + Bold; Error replaces Hint and inks with `Colors.Error`.
- **Accordion** — the one hook-owning widget (`NewState` for expanded).
  Doc comment states the render-unconditionally rule, that Content is
  conditionally rendered so must be hook-free, and that debug mode flags
  violations as cursor drift. Header slot replaces the chevron+Title content;
  tap target stays with the widget.
- **Tabs** — open question resolved as **wrap**, not supersede:
  `core.TabView` is the "TabView" node-type *wire contract* the native
  renderers consume, and node-type contracts belong in core. Tabs is the
  named-field facade delegating to it; omits `onTabChange` when OnChange is
  nil to keep static strips diff-stable.

### Todoapp migration (acceptance proof)

`filterBar` in `examples/todoapp/app.go` now renders `components.Chip` (with
`SelectedStyle` overriding Chip's theme default to keep the app's accent
palette, and Chip owning the ", selected" suffix).
`examples/todoapp/chip_migration_test.go` keeps the pre-extraction bar
verbatim as `legacyFilterBar` and asserts `htmlout.ExportHTML` output is
**byte-identical** for all three active states. Fresh contexts per render
make per-pass callback IDs line up, so any diff would be structural.

## Verification

`go build ./... && go vet ./... && go test -race ./...` fully green after
each workstream; each committed separately per the plan's sequencing rule.

## Key decisions / gotchas worth remembering

- Debug flag is process-wide by design; the collector moves to Context with
  it if that ever changes.
- `renderAll(ctx, parentType, views)` is the intentional single choke point
  for sibling-list checks — new containers should route children through it.
- `core.Padding(0)` *assigns* a zero EdgeInsets (styleFunc), while
  `core.UseStyle` merges only non-zero fields and cannot zero anything out.
- `Context.children` is vestigial (appended nowhere, only iterated in
  Reset/audit); real child contexts live in slots and scopes.
- Byte-identical component migrations are achievable because Style props
  touching disjoint fields commute — application order leaves no trace in
  the final Style struct — and callback IDs are per-pass sequence numbers.

## Possible next steps

- More widgets in `components/` (element's set: pagination, modal facade,
  toast facade); extract other hand-rolled example-app patterns.
- Provenance (debug check 3) once named user components are common.
- Identity-keyed callback IDs (noted in event.go as the eventual fix for
  stale-tree dispatch around structural re-renders).
