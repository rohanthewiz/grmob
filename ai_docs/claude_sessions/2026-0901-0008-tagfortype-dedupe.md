# Session: Deduping the node-type → tag table

**Session ID:** session_01LZZTYuh1UXxGjKfkomQRM6
**Date:** 2026-09-01, ~00:08
**Branch:** master
**Follows:** `2026-0831-2357-inputtype-dedupe.md`

The previous session closed the node-type → **input-type** duplication and
noticed, without acting on it:

> **`tagForType` is duplicated the same way** — once in `htmlout/export.go`,
> once in `grmob-runtime.js` … The runtime's copy of `tagForType` is the
> larger of the two and covers node types `htmlout`'s `default` branch handles
> implicitly, so the two are not currently line-for-line comparable — a shared
> authority would have to settle that first.

This session settled it, and then did the dedupe. The settling was the
interesting half.

---

# Part 1 — What made the two copies incomparable

Not a scatter of small differences. **One axis: `Fragment` and `Theme`.**

`htmlout` renders both transparently — their children land directly in the
parent, no box — which is what `core.For` and `core.WithTheme` need and what
both natives already do (SwiftUI `Group`, Compose `RenderChildren`). The WASM
runtime boxes both in a `<div>`.

That is not an oversight, and it is not fixable by editing a table. Patches
are addressed **positionally**: `reconcile.Patch.TargetID` is `"root/1/0"`,
built by walking `node.Children` (`reconcile/patch.go:96`), and the runtime
resolves it against the `data-node-path` attributes it wrote while walking the
same indices (`grmob-runtime.js:9,18`). **Drop the element for a Fragment and
every patch beneath it goes to the wrong node.** `htmlout` is a static
snapshot with no patch stream to keep addressable, so flattening costs it
nothing.

So the two renderers answer *two different questions*, and the old shared
name `tagForType` conflated them:

| question | answered by |
|---|---|
| *which* element does this node type become? | one table, shared |
| *whether* an element at all | per-renderer, and they genuinely differ |

The dedupe only works once those are separate.

## The authority

`htmlout/tag.go` holds both halves:

```go
var tags = map[string]string{...}          // 19 rows
func TagFor(nodeType string) string        // query; defaults to "div"
func Tags() map[string]string              // enumerate, as a copy

var transparentTypes = map[string]bool{"Fragment": true, "Theme": true}
func IsTransparent(nodeType string) bool
func TransparentTypes() []string           // sorted, for the conformance test
```

`TransparentTypes` is exported for exactly the reason `InputTypes` was: the
wasm conformance test has to know which rows are exempt, and a hand-written
list over there would be precisely the untracked second copy this file exists
to delete.

**The table is a census, not a list of exceptions.** All fourteen node types
that become a plain `<div>` are spelled out even though the default produces
the same answer. The point is that adding a node type to `core` and forgetting
the renderers shows up as a **gap in a visible list** rather than as silence —
a default can never fail.

`export.go` lost its `tagForType` switch; `renderContainer` calls `TagFor` and
`renderNode` asks `IsTransparent` instead of comparing two string literals.
The long Fragment/Theme comment moved to `tag.go`, where the fact now lives.

# Part 2 — Pinning the JavaScript copy

The runtime's `tagForType` was a `switch` with fallthrough cases; `inputTypeFor`
was a flat object literal. **Reshaped the switch into the same literal form**,
with `|| "div"` where the `default` was, so that:

- both runtime tables are parsed by **one** helper
  (`wasm/verify/jstable_test.go`), instead of two near-identical 40-line
  parsers, and
- the shape constraint the parse imposes is *one* documented shape, stated
  twice in the runtime and once in the docs.

Behavior is unchanged: the six types that previously reached `div` through
`default` (`Box`, `List`, `Modal`, `TabView`, `CameraView`, `Theme`) now say
`div` explicitly.

The parser now **folds the fallback check in**:

```go
parseRuntimeTable(t, src, funcName, fallback) map[string]string
```

It requires the literal to be followed by `[type] || "<fallback>";` with the
caller's expected value. That is what makes the parse total — the fallback
carries the half of each contract that lives outside the table (`""` is what
leaves a `<textarea>` with no `type` attribute; `"div"` is what an unknown node
type becomes), and a table comparison alone would pass a runtime whose default
had drifted. `TestRuntimeInputTypeFallbackIsEmpty` accordingly split: its JS
half is now enforced at parse time, its Go half survives as
`TestInputTypeForUnlistedTypeIsEmpty`.

Every parse step is still a **named fatal** rather than a short map, for the
reason the last session wrote down: *a check that reads nothing must not be
able to read as a pass.*

## How the divergence is exempted without becoming a hole

`TestRuntimeTagsMatchGo` removes the transparent rows before comparing — but
does not merely skip them:

```
runtime has no row for a transparent type  →  error (it needs an explicit box)
runtime boxes it in something other than div →  error (the known divergence is a div)
```

So the exemption is narrowed to the exact tag the runtime is known to use. If
the runtime ever learns to address elementless nodes, these rows go away and
the loop fails until it is deleted — which is the right kind of failure to get
from a landmark.

## What makes the authority real on the Go side

`renderNode`'s typed element calls (`b.Span`, `b.Button`, `b.Img`,
`b.TextArea`, `b.Input`) still spell their tags themselves — more readable at
the call site than `b.Ele(TagFor(...))` — so only `renderContainer` actually
reads the table. `TestExportedTagsMatchTable` is what ties those spellings
back to it: it exports a node of every type in `Tags()` and compares the first
element inside `<body>`.

Without that test, the wasm conformance test would have been comparing the
runtime against a table **nothing in Go was required to follow**.

One wrinkle it had to encode: an `Image` with no `src` deliberately falls
through to the container path and degrades to a `<div>`, so the test supplies
a `src`. That is stated in the test rather than left to be rediscovered.

---

# Testing the tests

**16 mutations, 16 caught**, with an unmutated control before *and* after.
The last five re-run the previous session's input-type mutations, because the
shared-parser refactor could have weakened them — it did not.

| mutation | result |
|---|---|
| JS tag value drift (`Button: "div"`) | caught |
| JS tag row dropped (`TextArea`) | caught |
| JS tag row Go lacks (`Video: "video"`) | caught |
| Go tag value drift (`Text` → `div`) | caught |
| JS tag fallback changed to `span` | caught, as a named fatal |
| `tagForType` renamed | caught, as a named fatal |
| JS tag table emptied | caught, as a named fatal |
| JS `Fragment` boxed as `span` | caught, by the divergence pin |
| JS `Fragment` row dropped | caught, by the divergence pin |
| Go transparency dropped (`Theme`) | caught |
| Go type both tagged *and* transparent | caught |
| htmlout exporter drift (`b.Span` → `b.Div` for Text) | caught |
| JS input type drift (`Checkbox: "text"`) | caught |
| JS input fallback removed (`}[type];`) | caught |
| `inputTypeFor` renamed | caught |
| control: unmutated (before, after) | passes |

Three of those are also caught by pre-existing tests, so they were re-run
listing *every* failing test, to confirm the new ones fire on their own rather
than riding along:

```
Go transparency dropped (Theme)  → TestRuntimeTagsMatchGo + 2 pre-existing
Go both tagged and transparent   → TestNoTypeIsBothTaggedAndTransparent,
                                   TestExportedTagsMatchTable, TestRuntimeTagsMatchGo
htmlout exporter drift           → TestExportedTagsMatchTable,
                                   TestTransparentTypesEmitNoElementOfTheirOwn + 4
```

Mutations were applied and restored from a scratchpad snapshot, not
`git checkout`.

## Files touched

`htmlout/tag.go` (new, +146: the table, `TagFor`, `Tags`, `transparentTypes`,
`IsTransparent`, `TransparentTypes`), `htmlout/export.go` (switch deleted;
`renderContainer` → `TagFor`, `renderNode` → `IsTransparent`),
`htmlout/tag_test.go` (new, +115: five tests — exporter/table agreement,
transparency as a set, disjointness, the `div` default, the copy contract),
`wasm/grmob-runtime.js` (`tagForType` switch → flat literal + census +
divergence comment), `wasm/verify/jstable_test.go` (new, +105: `runtimeSource`
and the shared `parseRuntimeTable`), `wasm/verify/inputtype_test.go` (rewritten
onto the shared parser), `wasm/verify/tagtype_test.go` (new, +59),
`docs/platforms/wasm.md` (+28: a "Node types and tags" section and the
divergence, in prose).

Gate: `gofmt` clean, `go vet` clean, full Go suite, `GOOS=js GOARCH=wasm`
build, `wasm/verify/run.sh` (34 tests, unchanged) and `ios/verify/run.sh`
(flex solver + 9 patch batches + Swift typecheck) — all green.
(`examples/todoapp/store.go` remains unformatted — pre-existing, untouched.)

## Not verified here

**No behavior changed.** `htmlout`'s output is byte-identical and the
runtime's JavaScript produces the same element for every node type it did
before — the six `default` types now say `div` explicitly. The existing suites
passing unchanged is the evidence, and is the only evidence available.

**Still no browser.** **Android still unbuilt.** **iOS still type-checks
without running.**

## Backlog

In Progress still holds only Packaging (`grmob build --target=…`).

Closed this session: the node-type → tag duplication between `htmlout` and the
WASM runtime.

**Promoted from a comment to a named, pinned divergence** (not closed):

- **The WASM runtime boxes `Fragment` and `Theme` in a `<div>`** where every
  other renderer treats them as transparent. Inside a flex parent that div
  becomes the single flex item and swallows the gap, `flex-direction` and
  alignment meant for the children — the same bug `htmlout` fixed in
  `2026-0831-1623`. Closing it means teaching the positional patch addressing
  about nodes that have no element, not editing a table.

Still open from earlier sessions:

- **`objectFit` / `objectFitFor` is the last of the three lookalike
  duplications** and would take the same treatment. It is smaller than either
  of the two now done, and the parse helper already exists — though the JS
  copy would have to be reshaped into the flat-literal form first, as
  `tagForType` was.
- Theme contrast, and `Variant`'s third consumer.
- **Hooks have no unmount signal.**
- **`docs/concepts/architecture.md` does not mention frames.**
- **`core.Image` bases its style on `Theme.Components.Camera`.**
- The WASM runtime's style mapping is still much thinner than `htmlout`'s.
- **A bottom-docked bar has no way to ask for the keyboard on its own.**
- **`htmlout` renders `Scroll` as a plain div** with no `overflow`.
- **A second imperative API would justify the bridge command channel.**
- **`core.SendSystemEvent` is a dead stub** — `core/toast.go` is its only
  caller, so `ShowToast` currently reaches nothing.
- **A `Cached` subtree silently swallows focus commands** and order membership.
- **An app-drawn keyboard toolbar has no worked example.**
- **`imeAction` is a third prop that must not vanish**, guarded by a third
  sticky sentinel; a single helper could state the rule once.
- **`gen.go`'s transcripts still emit no `add`, `remove` or `add-child`.**
- **The demo scenario's tab switches produce only `update-props`.** Still not
  understood.
- **Nothing runs either verify harness automatically.** Two of the three
  cross-language tables are now checked by `go test ./...` regardless.
- **No example app uses `core.TextArea`** (nor `core.Image` / `CameraView`).

Noticed this session, not acted on:

- **The tag census can go stale in the one direction nothing checks.** Adding
  a node type to `core` and to neither table still produces a working `<div>`
  in both renderers, silently. A test that enumerated `core`'s node types and
  required each to appear in `tags` or `transparentTypes` would close it, but
  `core` has no such enumeration to read — the type strings are literals at
  ~21 construction sites.
- **The parse's shape constraint is now load-bearing for two functions.**
  Both must stay a flat object literal in a named function followed by
  `[type] || "<fallback>"`. Written down in three places, and every violation
  is a named fatal, but it is still a constraint on how those functions may
  be rewritten.
