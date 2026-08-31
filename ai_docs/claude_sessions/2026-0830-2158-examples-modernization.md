# Session: Examples modernization

**Session ID:** session_014UQGD2NBjK6NrbCqdUmmdn
**Date:** 2026-08-30, ~21:58
**Branch:** master

## Goal

Execute `ai_docs/plans/examples-modernization.md` — align the seven examples
under `examples/` with the contracts the recent workstreams (Cached, debug
mode, components) and the MkDocs set now teach. All seven compiled and passed
before this session; the gap was pedagogical, not functional: the five older
examples (`chat`, `fintechapp`, `layout`, `runtime`, `social`) contradicted or
ignored documented contracts, and examples are documentation that runs.

Executed workstreams 1 → 7 in order, then a follow-up change to `mobileapp`
that the plan had marked out of scope.

## What landed

### Workstream 1 — `examples/runtime`: the complete pass boundary

`renderTree` previously modelled half the boundary (`BeginRenderPass` +
`core.Render`). It now runs the full pairing prescribed by `core/debug.go`:

```
BeginRenderPass()      — callback ID counters restart
Reset()                — hook cursors restart (core.Render does this)
root.Render(ctx)       — components consume slots
EndRenderPass()        — debug: cursors audited (no-op in production)
PurgeUnusedCallbacks() — drops handlers not re-registered this pass
```

`main` turns on `core.SetDebugMode(true)` and prints a `Concerns:` section
after the second render, labelling the empty case `(none)` explicitly since
`DumpConcerns()` returns `""` when clean. This makes runtime the one example
demonstrating the debug API end to end. Output verified byte-identical apart
from the new concerns line.

### Workstream 2 — documented render entry in the one-shot exporters

`layout` and `fintechapp` moved from `view.Render(ctx)` to
`core.Render(ctx, view)`, each with a comment noting that `core.Render` resets
the hook cursor so the entry point stays correct when called on every pass.
`chat` got the same treatment as part of its rewrite (WS5). Deliberately did
NOT add `BeginRenderPass`/`EndRenderPass` to these exporters — they render
once, and the runtime example owns the pass-loop lesson.

### Workstream 3 — `Spacer` interleaving → `Gap`

Migrated uniform spacing runs to `core.Gap(n)` on the container; kept `Spacer`
where gaps genuinely differ, with a justification comment at each site.

- `layout/main.go`: root Column (uniform 8) and `BodySection`'s inner Column.
- `fintechapp/main.go`: `BalanceCard` (8), `ActionsSection` (12),
  `TransactionList`'s row list (12 — moved out of `TransactionItem`, which had
  been carrying its own trailing `Spacer(12)`, the exact pattern Gap replaces).
  `App` (24/24/28) and `HeaderSection` (12/4) keep Spacers + comments.
- `social/pages.go`: `HomePage` (12), `DetailsPage` (10); deleted the stray
  adjacent `Spacer(10)` + `Spacer(8)`.

Note: `core.Scroll` does NOT accept style props (`core/layout.go:59` — children
only), contrary to a plan assumption; spacing goes on a Column inside it.

### Workstream 4 — `examples/social` self-contained + Scope

- **New `examples/social/app.go`** with exported `App(ctx) core.View` owning
  the Navigator + tab bar that previously lived in `wasm/main.go:110-127`.
  `wasm/main.go` now uses `social.App` through its existing dot-import and is
  pure host wiring for the app tree.
- **`DetailsPage` counter revived under a scope**: `sc := ctx.Scope("details")`
  + `NewState(sc, 0)` + an increment button. The doc comment reuses the
  phrasing from `docs/concepts/state-and-hooks.md` so example and doc reinforce
  each other, and names debug mode's cursor-drift check as the tripwire.
- **New `examples/social/app_test.go`** — two tests driving
  `render.Manager` (push Details → increment ×2 → pop → re-push), asserting the
  counter survives, the tab bar still works after the round trip, and zero
  debug concerns.

Left in place (not mentioned by the plan, and the global "don't remove code
unless sure" rule): `wasm/main.go`'s unused `TabsComponent`, `HomeScreen`,
`DetailsScreen` demo leftovers. Candidate for a follow-up cleanup.

### Workstream 5 — `examples/chat` rewritten as a real chat app

The file was a second copy of the social feed ("GrMobGram", writing
`social.html`). Replaced with a message thread, ~200 lines, teaching exactly
three things:

- `core.For` + `core.Keyed("msg-"+id, …)` for data-driven rows with stable
  identity.
- `core.UseStyle` carrying a whole visual role (sent vs received bubble),
  with `core.Justify` for main-axis placement.
- One mutation choke point (`send`) reached by both `InputWithSubmit`'s
  onSubmit and the send button.

Stateful, so it drives two passes by hand like runtime and prints before/after
HTML. Dot-import dropped in favour of explicit `core.`.

Two deliberate details worth remembering:

- Row spacing rides on the bubble (a bottom margin), not a `Gap` on the list
  Column: `core.For` groups its output in a single **Fragment** child, so a Gap
  on the container spaces the Fragment against its siblings, not the messages
  inside it.
- The optional sender label is appended to a slice rather than wrapped in
  `core.If` — a false `core.If` still returns a Fragment, and an empty Fragment
  is a real child that takes a slot in the bubble's flex layout.

### Workstream 6 — debug-mode assertions in the flagship tests

Added to `mobileapp`, `todoapp`, and the new `social` test:

- `TestMain` calling `core.SetDebugMode(true)` (process-wide, one switch per
  package). `todoapp`'s existing TestMain gained the call ahead of
  `SetDataDir`; `mobileapp` got a new one.
- `core.ClearConcerns()` at the start of each render-driving test (the
  collector is process-wide too).
- `defer assertNoConcerns(t)` — fails with the full `DumpConcerns()` block.

The known caveat held: `mobileapp.appHeader` is `core.Cached` and deliberately
inert, and did NOT trip `cached-hooks`/`cached-callbacks`.

### Workstream 7 — `fintechapp` onto `components` (smallest slice)

- `TransactionItem` rebuilt as a Row with the amount as a `components.Badge`
  (label left, badge pinned right via `Justify(JustifyBetween)`). Signature
  gained an `ink` parameter: Badge defaults its label colour to the theme's
  Background (chosen to read on Primary), which is illegible on a light teal
  credit — `Badge.TextColor` is the slot for exactly that.
- `MaterialButton` **left hand-rolled**, with a comment saying why: Chip is a
  *selection* affordance carrying a `Selected` flag and a pressed-in variant
  (the filter-bar pattern it was extracted from). Transfer/Recharge are
  one-shot primary actions.
- `BalanceCard` **stays on `core.Card`**, with a comment: `components.Card`
  renders its Title in the theme's *Subtitle* role, visibly wrong for what is a
  caption above a figure.

### Follow-up (user-requested) — `mobileapp` structure

The plan put mobileapp structure out of scope; the user asked for it after the
final report flagged the remaining sites.

- `formTab`: uniform 8 between all three children → `core.Gap(8)`, both Spacers
  removed. Verified in the emitted JSON tree (`"Gap":8`, Spacer nodes gone).
- `counterTab`: kept its `Spacer(16)` — gaps there are 0 then 16 (count and
  button are one block; the 16 separates that block from the timer line), which
  `Gap` cannot express. Now carries a justification comment cross-referencing
  `formTab`, so the two sites read as one rule.

## Two bugs found and fixed along the way

### 1. `Context.WithTheme`/`WithConfig` dropped the `scopes` map

`core/context.go` — both methods build a fresh `Context` literal and omitted
`scopes`, leaving it nil. Any `ctx.Scope(key)` on such a context panicked with
*assignment to entry in nil map*. This is not an obscure path:

- `render.New` calls `ctx.WithTheme(core.DefaultTheme)` when the context has no
  theme (`render/manager.go:69`),
- `wasm/main.go:14` is `core.NewContext().WithTheme(core.DefaultTheme)`,
- `core.WithTheme(theme, children...)` derives a new context per render
  (`core/theme.go:45`).

So the documented `Scope` idiom crashed in the shipped WASM host. Workstream 4
is unbuildable without the fix. Fixed by sharing the parent's scope table
(`scopes: ctx.scopes`) rather than making a fresh map — also the correct
semantics: a themed copy is the same context wearing a different theme, so a
scope reached through it must be the same scope.

### 2. `htmlout` silently dropped `Style.Gap`

`htmlout.styleValue` never serialized `Gap`, though both native renderers honor
it (Compose `Arrangement.spacedBy`, SwiftUI stack spacing). The plan's WS3
verification step ("`gap` style present, spacer divs gone") assumed otherwise,
and the migration would have *deleted* the exporters' visible spacing.

Extended `styleValue` to emit the flex **container** properties — `gap`,
`justify-content`, `align-items`, `flex-direction` — plus item-level
`flex-grow`. Key detail: a plain `<div>` is block flow and ignores all of them,
so the container must be made flex for any to take effect. Only nodes that
actually set one of these become flex containers; everything else keeps the
block-flow output the exporter has always produced. Main axis defaults to the
node's own stacking direction (Row horizontal, everything else vertical), with
an explicit `FlexDirection` overriding. `styleValue` gained a `nodeType`
parameter for this. Emitted before `Display` so an author's explicit `Display`
wins the browser's last-declaration-wins parse.

## Verification performed

- `go build ./...`, `go vet ./...`, `go test ./...` — all pass.
- `GOOS=js GOARCH=wasm go build -o <scratch>/main.wasm ./wasm` — passes. (Note:
  a bare `go build ./wasm` fails with *build output "wasm" already exists and
  is a directory*; `-o` is required.)
- Each exporter re-run and its HTML inspected: `layout`, `fintechapp`, `chat`,
  `runtime`.
- **Negative tests, to prove the new assertions earn their keep:**
  - Reverting `sc := ctx.Scope("details")` to bare `ctx` fails the social test
    with `interface conversion: interface {} is string, not int` — the counter
    reads the tab state's string, exactly as the comment describes.
  - Forcing a constant key in `todoapp`'s row builder produces
    `[duplicate-key] ×6 For has multiple children with key "todo-dup"` and
    fails both todoapp tests.
- Final plan check: `grep -rn "Spacer(" examples/` shows only sites carrying a
  non-uniform-spacing justification comment —
  `mobileapp/app.go:97`, `fintechapp/main.go:41,43,45` (comment on the
  enclosing `core.Column`), `:58,60` (comment on `HeaderSection`), `:112`.

## Files touched

**Modified:** `core/context.go`, `htmlout/export.go`, `wasm/main.go`,
`examples/runtime/main.go`, `examples/layout/main.go`,
`examples/fintechapp/main.go`, `examples/chat/main.go`,
`examples/social/pages.go`, `examples/mobileapp/app.go`,
`examples/mobileapp/app_test.go`, `examples/todoapp/app_test.go`

**Added:** `examples/social/app.go`, `examples/social/app_test.go`,
`ai_docs/plans/examples-modernization.md`

## Open items / follow-ups

- `wasm/main.go` still carries unused demo leftovers (`TabsComponent`,
  `HomeScreen`, `DetailsScreen`) that duplicate the social pages. Removing them
  would finish the "pure host wiring" intent of Workstream 4.
- `gofmt -l` flags `components/badge.go` and `examples/todoapp/store.go` —
  pre-existing drift in files this session did not touch.
- `docs/` was left alone per the plan. Nothing surfaced that contradicts the
  MkDocs set, but the `Scope`-under-Navigator case is now demonstrated in code
  (`examples/social/pages.go`) and could be cross-linked from
  `docs/concepts/state-and-hooks.md`.
- The two core/htmlout fixes are behaviour changes outside `examples/`; the
  `htmlout` flex support in particular is worth a line in the docs if the HTML
  exporter is ever described as a faithful preview.
