# Examples Modernization — Aligning examples/ with the Current Architecture

**Status:** Planned
**Date:** 2026-08-30
**Source:** Audit of `examples/` against the pass-boundary contract (core/debug.go:144-157),
debug mode, the components package, and the idioms the new MkDocs set teaches
(session `2026-0830-1954-mkdocs-documentation.md`).

## Background

`examples/mobileapp` and `examples/todoapp` were updated alongside the recent
workstreams (Cached, debug mode, components) and are current. The five older
examples — `chat`, `fintechapp`, `layout`, `runtime`, `social` — predate those
workstreams. All seven compile, `go test ./examples/...` passes, and the
runtime example's event round-trip was verified working (`txt_cb_0` dispatch
still produces the re-rendered greeting). Nothing is broken; the gap is that
the older examples contradict or ignore contracts the docs now teach, and
examples are documentation that runs.

Guiding principle for every change below: **each example should model exactly
one thing well and follow the documented contract while doing it.** Where an
old example demonstrates an anti-pattern the docs warn about (social's
commented-out route hook), convert it into a demonstration of the documented
fix rather than deleting it.

## Workstream 1 — Complete the pass boundary in `examples/runtime`

The example exists to teach the hand-rolled pass loop, but only models half of
it. `core/debug.go:144-157` prescribes the full pairing:

```
BeginRenderPass()  — callback ID counters restart
Reset()            — hook cursors restart (core.Render does this)
root.Render(ctx)   — components consume slots
EndRenderPass()    — debug: cursors audited
PurgeUnusedCallbacks() — handlers not re-registered this pass are dropped
```

Changes to `examples/runtime/main.go`:

1. In `renderTree`, add `defer`-free explicit calls after the render:
   `ctx.EndRenderPass()` then `ctx.PurgeUnusedCallbacks()`. Keep the existing
   comment block and extend it to explain *why* each call exists (End = debug
   cursor audit, no-op in production; Purge = drops handlers not re-registered,
   which matters once passes repeat).
2. Turn on debug mode at the top of `main`: `core.SetDebugMode(true)`, and
   after the second render print `core.DumpConcerns()` (prints nothing when
   clean — say so in the output label, e.g. "Concerns: (none)" when the dump is
   empty). This makes runtime the one example that demonstrates the debug-mode
   API end to end.
3. Verify by `go run ./examples/runtime`: output must be unchanged apart from
   the concerns line, and the concerns dump must be empty.

## Workstream 2 — Use the documented render entry in the one-shot exporters

`chat`, `fintechapp`, and `layout` call `view.Render(ctx)` directly, skipping
`core.Render(ctx, view)` (which performs the `ctx.Reset()` cursor reset).
Harmless on a fresh context rendered once, but it contradicts the contract the
docs and Workstream 1 establish, and these files are what people copy first.

Changes (one line each, plus a short comment):

- `examples/chat/main.go:11` → `node := core.Render(ctx, SocialApp())`
  (adjusted for the dot-import; see Workstream 5 for this file's bigger issue).
- `examples/fintechapp/main.go:19` → `node := core.Render(ctx, App(ctx))`.
- `examples/layout/main.go:54` → `node := core.Render(ctx, AppLayoutExample())`.

Comment to add at the first converted call site in each file: one sentence
noting that `core.Render` resets the hook cursor so the entry point is safe to
call on every pass, not just the first — mirroring what the runtime example
spells out in full. Do NOT add `BeginRenderPass`/`EndRenderPass` to these
exporters: they render exactly once, and pretending otherwise would blur what
each example teaches. The runtime example owns the pass-loop lesson.

## Workstream 3 — Migrate old examples from `Spacer` interleaving to `Gap`

The updated examples set `core.Gap(n)` on the container; the older five
interleave `core.Spacer(n)` between every child. Migrate where the spacing is
uniform; keep `Spacer` where gaps genuinely differ between siblings (it still
has a legitimate role — say so in a comment at one representative site).

- `chat/main.go`: `FeedPost`'s inner `Column` (uniform 8) and the action `Row`
  (uniform 8) → `Gap(8)`. The top-level `Scroll` children (uniform 12) →
  `Gap(12)` on the Scroll's Column if Scroll accepts style props the same way —
  verify against core/list.go before assuming.
- `fintechapp/main.go`: `App`'s Column mixes 24/24/28 — leave as Spacer or
  normalize to `Gap(24)` + one extra `Spacer(4)`; prefer leaving it and adding
  the "non-uniform ⇒ Spacer" comment here. `HeaderSection` (12/4) stays
  Spacer. `TransactionList` (16 then per-item trailing 12) → move the
  per-item `Spacer(12)` out of `TransactionItem` and put `Gap(12)` on the list
  Column; the item keeping its own trailing spacer is exactly the pattern Gap
  exists to replace.
- `layout/main.go`: root Column (uniform 8) → `Gap(8)`.
- `social/pages.go`, `wasm/main.go` app views: uniform runs → Gap.
- Delete the stray adjacent `Spacer(10)` + `Spacer(8)` at
  `social/pages.go:22-23` (leftover from a removed line).

Verification: re-run each exporter and eyeball the emitted HTML (`gap` style
present, spacer divs gone); `go vet ./examples/...`.

## Workstream 4 — Make `examples/social` self-contained and teach Scope

Two problems: the package ships `Push`/`Pop` routes with no Navigator root
(the wiring hides in `wasm/main.go`), and `DetailsPage` carries a
commented-out `NewState` (pages.go:18) — which is precisely the slot-aliasing
trap the state docs cover: Navigator routes render into the same context, so
route-level hooks alias each other's slots positionally.

1. Add `examples/social/app.go` with an exported `App(ctx *core.Context)
   core.View` that owns the Navigator + tab bar currently living in
   `wasm/main.go` (`App`, lines 110-127). `wasm/main.go` then imports and uses
   `social.App`, deleting its local copy. This makes the social example
   runnable-in-principle from its own package and shrinks wasm/main.go to pure
   host wiring. Keep `TabButton` in tab.go; it is now called by app.go in the
   same package.
2. Revive the DetailsPage counter **under a scope**:
   `sc := ctx.Scope("details"); count := core.NewState(sc, 0)` plus an
   increment button. Comment must state the why: without the scope this slot
   would positionally alias the tab-state slot allocated by the initial route,
   and the debug checker's cursor-drift concern is exactly the tripwire for
   getting this wrong. (Verify the exact `Scope` signature and idiom against
   `core/context.go:249` and `docs/concepts/state-and-hooks.md` before
   writing — the docs' phrasing should be reused so example and doc reinforce
   each other.)
3. Housekeeping in pages.go: remove the dead commented line; Gap migration per
   Workstream 3.
4. Verification: wasm build still compiles —
   `GOOS=js GOARCH=wasm go build ./wasm` — plus a new
   `examples/social/app_test.go` driving two passes through `render.Manager`
   (push Details, increment, pop, re-render) asserting the counter survives
   and, per Workstream 6, zero concerns.

## Workstream 5 — Give `examples/chat` its own identity

`chat/main.go` is a second copy of the social feed ("GrMobGram", writes
`social.html`) — it duplicates `examples/social` and demonstrates nothing
chat-shaped. **Decision taken: write a real chat example** (the user asked for
a plan to update the examples, and a rename-only fix would leave two social
apps and zero chat apps).

Shape of the new `examples/chat/main.go` (keep it export-driven like the other
one-shot examples, ~100 lines):

- A message list: `core.Scroll` + `core.For` over a `[]Message` slice with
  `core.Keyed("msg-"+id, …)` rows — the For/Keyed idiom currently only shown
  inside the two big apps.
- Sent-vs-received styling via `core.UseStyle` and alignment, exercising the
  styling docs' patterns.
- A composer row: `core.InputWithSubmit` + send button appending to the slice
  through a single mutation helper (echoing todoapp's "one choke point for
  writes" comment style, in miniature).
- Because it's stateful, drive it like runtime does: two passes with a
  simulated send event, printing before/after HTML. This also gives the
  For/Keyed path a second executable exercise.
- Drop the dot-import; use `core.` explicitly like every other example except
  social (dot-import is a pre-existing inconsistency; chat is being rewritten
  anyway, social keeps its style).

The old feed content is not lost — `examples/social` already covers it; note
this in the commit message rather than keeping a dead file.

## Workstream 6 — Debug-mode assertions in the flagship example tests

`docs/concepts/debug-mode.md` advertises the "assert zero concerns" test
pattern; no example test uses it. Add to both `examples/mobileapp/app_test.go`
and `examples/todoapp/app_test.go` (and the new social test from Workstream 4):

- A `TestMain` that calls `core.SetDebugMode(true)` before `m.Run()` — debug
  mode is process-wide, so one switch covers every test in the package.
- `core.ClearConcerns()` at the start of each test that drives renders (the
  collector is also process-wide; without clearing, a concern from one test
  bleeds into another's assertion).
- At the end of each render-driving test:
  `if cs := core.Concerns(); len(cs) != 0 { t.Fatalf("debug concerns:\n%s", core.DumpConcerns()) }`.
- Check first whether existing tests share state in ways TestMain would
  disturb (todoapp's store tests use temp dirs — confirm no ordering
  assumptions).

Note the known caveat from the debug workstream: Cached bypass is
reported-not-exhibited — mobileapp's `appHeader` is deliberately inert, so it
must NOT trip `cached-hooks`/`cached-callbacks`; if it does, that's a real
finding, not test noise.

## Workstream 7 — `fintechapp` onto the components package (smallest slice)

`todoapp` is the only consumer of `components`. Give it a second one, but keep
the diff small — fintechapp's job is showing theming, not the component
catalog:

- Replace the hand-rolled `TransactionItem` with a row built on
  `components.Badge` for the amount (verify Badge's actual API in
  `components/badge.go` first — color/label slots may differ from assumption).
- Replace `MaterialButton` with `components.Chip` **only if** Chip's semantics
  fit a primary action button (it likely doesn't — Chip is a filter/selection
  affordance in todoapp). If it doesn't fit, leave `MaterialButton` alone and
  say why in a comment; forcing a component where it doesn't belong teaches
  the wrong lesson.
- `BalanceCard` stays on `core.Card` unless `components.Card` adds something
  visible (check `components/card.go`); if switched, the htmlout output should
  be compared before/after.

This workstream is the most judgment-dependent; it runs last and any piece of
it may be dropped after reading the component sources.

## Explicitly out of scope

- **`core.Cached` in the one-shot exporters** — they render once; caching
  there would be pedagogically misleading. `mobileapp.appHeader` already
  demonstrates it where re-renders exist.
- **Rewriting `mobileapp`/`todoapp` structure** — they are current; they only
  gain the Workstream 6 test assertions.
- **`docs/` changes** — the MkDocs set was verified against source this
  session. If Workstream 4's Scope example surfaces a docs gap, note it, don't
  fix it here.

## Order and verification

Execute 1 → 2 → 3 → 4 → 5 → 6 → 7. Workstreams 1-3 are mechanical; 4-5 create
files; 6 depends on 4; 7 is optional-per-piece.

After each workstream: `go build ./... && go vet ./examples/... && go test ./examples/...`.
After Workstream 4: `GOOS=js GOARCH=wasm go build ./wasm`.
After Workstreams 2/3/5: run each affected exporter (`go run ./examples/<name>`)
and inspect the emitted HTML; for fintechapp Workstream 7, diff the HTML
before/after and confirm only intended changes.

Final check: `grep -rn "Spacer(" examples/` should only show sites carrying
the "non-uniform spacing" justification comment.
