# Session: Deduping the ContentMode → object-fit table

**Session ID:** session_01LZZTYuh1UXxGjKfkomQRM6
**Date:** 2026-09-01, ~00:16
**Branch:** master
**Follows:** `2026-0901-0008-tagfortype-dedupe.md`

The last of the three lookalike duplications the input-type session noticed.
Same treatment, one new wrinkle in the table itself, and one thing this table
could do that neither of the other two could.

---

# Part 1 — The wrinkle: the two copies stored different shapes

`htmlout`'s `objectFit` returned `"object-fit:contain"`. The runtime's
`objectFitFor` returned `"contain"`. Same mapping, different amount of it —
because `htmlout` is assembling a semicolon-joined declaration list and the
runtime is assigning `el.style.objectFit`.

So the shared table holds **the value, not the declaration**:

```go
var objectFits = map[core.ContentMode]string{
	core.ContentModeFit:     "contain",
	core.ContentModeFill:    "cover",
	core.ContentModeStretch: "fill",
	core.ContentModeCenter:  "none",
}
func ObjectFitFor(mode string) string   // "" for absent/unrecognized
func ObjectFits() map[string]string     // enumerate, as a copy
```

and `export.go`'s `objectFit` became `objectFitDecl`, which owns exactly one
thing the runtime does not have: the `"object-fit:"` prefix. That prefix is
therefore the one part of the mapping the conformance test cannot see, so it
is asserted separately (`TestObjectFitDeclPrefixesTheProperty`, values written
out rather than read from the table).

## Two ways of saying nothing

`""` means the same thing on both sides and is *implemented* differently, and
the difference is load-bearing:

| | absent/unknown mode |
|---|---|
| `htmlout` | emits no declaration at all |
| runtime | assigns `""`, which **clears** the property |

Clearing is what the patch path needs: an Image whose `contentMode` prop is
removed has to fall back to the browser's default rather than keep the last
mode it was handed. Written down in both files now.

Also worth stating, because it is easy to assume otherwise: `""` is not
`ContentModeFit`. `core.imageNode` omits the prop entirely rather than writing
`"fit"`, so the mode-less case exports exactly as it did before ContentMode
existed. The natives are the ones that fold a missing prop into Fit, because
SwiftUI and Compose have no "unset" to fall back to.

# Part 2 — Pinning the JavaScript copy, and a better guard

**The runtime needed no reshaping.** `objectFitFor` was already a flat literal
with an `""` fallback — it is where that shape came from in the first place —
so only its comment changed, to name the authority and the test.

But it subscripts by `mode`, not `type`, and the parser hard-coded `[type]`.
The guard fired on the first run, exactly as designed:

```
found: [mode] || "";
--- FAIL: TestRuntimeObjectFitsMatchGo
```

The fix was **not** to rename the JS parameter to suit its test. The parser now
captures the parameter name off the signature and requires the subscript to
match it:

```go
start := regexp.MustCompile(`function\s+` + funcName + `\s*\(\s*(\w+)\s*\)\s*\{`)
param  := ...
wantTail := "[" + param + `] || "` + fallback + `";`
```

That is strictly stronger than what it replaced: it no longer just checks that
*a* subscript follows the braces, it checks the table is keyed by the function's
own argument. `mode` here and `type` there are both fine; a subscript that is
neither is a fatal.

# Part 3 — The thing this table could do that the other two could not

The tag census can go stale in the one direction nothing checks — that was the
open note from last session, and the reason given was that node types are
string literals at ~21 construction sites with no list to check against.

**`ContentMode` is different.** It is a named type with four declared
constants, so coverage can be a test instead of a hope. Three links, each
pinned to something other than another list:

```
core/image.go const block  ←(go/ast)→  core.ContentModes()
                           ←(census)→  htmlout.objectFits
                           ←(parse)→   grmob-runtime.js
```

- **`core.ContentModes()`** is new. Go cannot enumerate a named string type's
  constants at run time, so the set has to be written out a second time —
  which is the thing that goes stale.
- **`TestContentModesMatchTheDeclaredConstants`** reads those constants out of
  `image.go`'s **syntax tree**. `go/ast` rather than a regexp because the
  question is syntactic ("which constants of type ContentMode does this file
  declare?") and a regexp over source would also match the same words in a doc
  comment. The parse is source-only, so it costs a millisecond and needs
  nothing built.
- **`TestObjectFitsCoversEveryContentMode`** then holds the table to that list.

Why it earns its keep: a mode missing from the table exports no declaration and
clears the property in the runtime, so a new `ContentMode` would render as the
browser default on *both* DOM targets while the natives honored it, silently.

Every parse step is a named fatal, per the standing rule: *a check that reads
nothing must not be able to read as a pass.*

---

# Testing the tests

**18 mutations, 18 caught**, unmutated control before and after. The last three
re-run the tag and input-type tables, because the parser signature change could
have weakened them — it did not.

| mutation | caught by |
|---|---|
| JS value drift (`fill: "contain"`) | conformance |
| JS row dropped (`stretch`) | conformance |
| JS row Go lacks (`tile`) | conformance |
| Go value drift (`center`) | conformance + decl test + export test |
| Go row dropped (`stretch`) | conformance + census + 2 more |
| JS fallback changed | conformance, as a named fatal |
| **JS subscript wrong (`[type]` for a `mode` param)** | conformance, as a named fatal |
| `objectFitFor` renamed | conformance, as a named fatal |
| JS table emptied | conformance, as a named fatal |
| Go `objectFitDecl` drops the prefix | decl test + 2 export tests |
| Go `objectFitDecl` emits a bare `object-fit:` | drop test + 2 export tests |
| core: constant added, not listed | `TestContentModesMatchTheDeclaredConstants` |
| core: listed, no constant | **the compiler** (`undefined: ContentModeTile`) |
| core: new mode, listed, no table row | `TestObjectFitsCoversEveryContentMode` |
| regression: JS tag fallback changed | tag conformance |
| regression: `inputTypeFor` renamed | input conformance |
| regression: JS input value drift | input conformance |
| control: unmutated (before, after) | passes |

One of those is caught by the Go compiler rather than a test, which is the
better outcome but leaves a question: is the test's reverse-direction loop dead
code? Two more mutations answered it — `ContentMode("tile")` in the list, and a
duplicate entry — and both fire with their own messages. Neither branch is
dead.

Mutations were applied and restored from a scratchpad snapshot, not
`git checkout`.

## Files touched

`core/image.go` (+27: `ContentModes`), `core/contentmode_enum_test.go` (new,
+97: the `go/ast` pin), `htmlout/objectfit.go` (new, +69: the table,
`ObjectFitFor`, `ObjectFits`), `htmlout/export.go` (`objectFit` →
`objectFitDecl`, now prefix-only), `htmlout/objectfit_test.go` (new, +82: the
census, the prefix, the dropped-mode contract, the copy contract),
`wasm/grmob-runtime.js` (comment only: names the authority, the test, the
shape, and why `""` clears), `wasm/verify/jstable_test.go` (subscript read off
the signature), `wasm/verify/objectfit_test.go` (new, +38),
`docs/platforms/wasm.md` (+26: an "Image content modes" section, and the
three-table shape note).

Gate: `gofmt` clean, `go vet` clean, full Go suite, `GOOS=js GOARCH=wasm`
build, `wasm/verify/run.sh` (34 tests) and `ios/verify/run.sh` (flex solver +
9 patch batches + Swift typecheck) — all green.
(`examples/todoapp/store.go` remains unformatted — pre-existing, untouched.)

## Not verified here

**No behavior changed.** `htmlout`'s output is byte-identical and the runtime's
literal was not touched, only its comment. The existing suites passing
unchanged is the evidence, and is the only evidence available.

**Still no browser.** **Android still unbuilt.** **iOS still type-checks
without running.**

## Backlog

In Progress still holds only Packaging (`grmob build --target=…`).

Closed this session: the `ContentMode` → `object-fit` duplication. **All three
lookalike cross-language tables now have one Go authority and a parsed pin**,
and all three checks run under a plain `go test ./...`.

Opened this session:

- **There are four copies of the ContentMode mapping, not two.**
  `Renderer.swift` maps the modes onto SwiftUI scaling and `Renderer.kt` onto
  Compose's `ContentScale`, so they share the *key set* with the DOM pair but
  not the values — a table comparison cannot reach them. Worse, both fold the
  unknown case into `else -> Fit`, so a fifth `ContentMode` would render as Fit
  on device while both DOM targets showed the browser default, and nothing
  would fail. `core.ContentModes()` now exists as the list they would be
  checked against; the check itself is a coverage rule in two more languages,
  run by the native harnesses rather than by `go test`.

Still open from earlier sessions:

- **The WASM runtime boxes `Fragment` and `Theme` in a `<div>`** where every
  other renderer treats them as transparent. Closing it means teaching the
  positional patch addressing about nodes that have no element.
- **The tag census can still go stale in the one direction nothing checks.**
  What this session did for `ContentMode` does not transfer: node types are
  string literals at ~21 construction sites, not constants of a named type.
  Making it transfer would mean giving core a node-type enumeration first.
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
- **Nothing runs either verify harness automatically.** All three
  cross-language tables are now checked by `go test ./...` regardless.
- **No example app uses `core.TextArea`** (nor `core.Image` / `CameraView`).

Noticed this session, not acted on:

- **`core/image.go`'s doc comments restate the CSS mapping in prose** ("CSS
  `object-fit: contain`", once per constant). That is a fifth statement of the
  rule, and unlike the others it is unpinned. It reads as the contract the
  table implements, which is the same argument that keeps `replay_test.mjs`'s
  copy independent — so it is probably right to leave it, but it is a copy.
- **The parse's shape constraint now binds three functions.** All must stay a
  flat object literal in a named single-argument function, subscripted by that
  argument, with a string fallback. The subscript half is now checked against
  the signature rather than hard-coded, which loosened it in the one place it
  was accidentally tight.
