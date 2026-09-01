# Session: Teaching the DOM pair the Align cross-axis fallback

**Session ID:** session_01HFQHJembyEppoKJCujVjga
**Date:** 2026-09-01, ~08:33
**Branch:** master
**Follows:** `2026-0901-0740-list-stretch-fallback.md`

Worked the next unblocked backlog item: **"The DOM does not read the `Align`
cross-axis fallback at all — centering/end-packing a List via `Align` works on
device only. May be deliberate (documented as native-only), but nothing pins
the choice."** Resolved it as a real gap, not a deliberate choice: every prior
session treated "one value, different behaviors across targets" as the bug
class this project exists to kill, and this was the last alignment behavior
the DOM pair did not share with the natives.

---

## The gap

Both natives have read `Style.Align` as the cross-axis fallback since they
existed (`crossAxisValue` in Renderer.swift, the
`alignItems.ifEmpty { align }` reads in Renderer.kt); neither DOM target read
it for placement at all. So `Align: center` on a Column centered the children
on iOS and Android and only the *text* on the web (htmlout's `text-align`
inherits), and `Align: stretch` filled rows on device while the web agreed
only wherever block flow or the flex `align-items: stretch` default happened
to produce the same picture.

## The semantics, copied exactly from the natives

- `AlignItems` wins when set; the fallback is consulted only when it is unset.
- The fallback applies only to the vertical-stacking containers the natives
  read it for — `Column`, `Card` (= Column with the themed look), `List` —
  and pointedly never `Row`: `Align` has never been read for a vertical cross
  axis on any target.
- `start`/`center`/`end` place, `stretch` fills; `justify`/`baseline` fall
  through to nothing (no native cross-axis dispatch answers for either —
  `baseline` falls through to start-packing there — so a row would move two
  targets out of four, even though CSS `align-items` genuinely has a
  `baseline` keyword someone could "complete" the table with).
- A `FlexDirection` flipped to a row declines the fallback (the prefix test
  admits `column-reverse`, whose cross axis is horizontal all the same).

## The fix

- **`htmlout/crossaxis.go`** (new) — the authority, in the textalign.go mold.
  `crossAxisAligns` maps the four cross-axis Alignments onto the *AlignItems
  spellings* (`start` → `flex-start`), because the fallback means "behave as
  if that AlignItems had been set" — one semantic, one CSS spelling, whichever
  prop stated it. `alignFallbackAxes` is the type gate (`Column`/`Card`/`List`
  → `"column"`, the axis that makes each cross axis horizontal); a gate is
  needed because `styleValue` serializes every node, and without it a Text
  carrying `Align` in its ordinary text role would become a flex container.
- **`htmlout/export.go`** — `styleValue` computes the effective cross-axis
  value (AlignItems, else the gated fallback) and lets it trigger the
  `display:flex` block the way an explicit AlignItems does.
- **`wasm/grmob-runtime.js`** — `crossAxisAlignFor` and `alignFallbackAxisFor`
  restatements (flat-literal shape for `parseRuntimeTable`) plus the gated
  read in `styleFromGrMob`. Safe on the patch path: an `update-style` patch
  carries the whole new Style (reconcile/patch.go), so an absent AlignItems
  means unset, not unmentioned.
- **`htmlout/crossaxis_test.go`** — the census is a *values* census: there is
  no `core.CrossAxisAlignments()` list, but the table's contract is that it
  translates one-to-one onto `core.AlignItemsValues()` and invents nothing,
  so the values must biject onto that list and every key must be a declared
  Alignment. Plus: gate pin, behavior tests per gated type (ranging over
  `AlignFallbackTypes()`), stretch end-to-end, AlignItems precedence, the
  four declines (Row, Text, flipped Column, text-only Align), copy tests.
- **`wasm/verify/crossaxis_test.go`** — both tables pinned table-against-table
  via the existing `parseRuntimeTable`, and the call site pinned with
  expression fragments (`dir.startsWith("column") && alignFallbackAxisFor(nodeType)`,
  `alignItems = crossAxisAlignFor(style.Align || "")`) so a comment cannot
  satisfy the pin.
- **Docs** — new "The cross-axis fallback" section in
  `docs/platforms/wasm.md` (and the "four tables"/"newest table"/"entirely
  native" claims corrected); `docs/platforms/native.md`'s "permitted, not
  required" bullet now says all four targets draw the Row line in the same
  place; `core/alignment.go` (the who-is-held table and the Alignments() doc),
  `htmlout/textalign.go`, `htmlout/textalign_test.go`, and the runtime's
  textAlignFor comment no longer call the fallback native-only.

## Gate

`gofmt` clean on touched packages, `go vet` clean, full Go suite green,
`wasm/verify/run.sh` green (transcript replay + Node unit tests, re-run after
the final JS edit). Mutation-checked at five points: a deleted Go table row
(caught by the census, the stretch behavior test, *and* the wasm conformance
test), a gutted Go fallback read, a corrupted JS table row, a gutted JS call
site, and a `List` dropped from the Go gate — each fails with a named
message; all restored and re-verified green.

(`examples/todoapp/store.go` shows up in a repo-wide `gofmt -l` — pre-existing
import ordering from the rebrand commit, untouched this session.)

## The residual worth knowing

- **A themed `Card`'s `Display: block` beats the flex container in htmlout's
  static export.** The default theme sets it, `styleValue` deliberately emits
  Display last so an explicit one wins, and `display:block` kills
  `align-items`. Pre-existing (explicit `AlignItems` on a themed Card has the
  same problem) and htmlout-only — the WASM runtime never emits Display.
  Nothing pins that interaction.
- **text-align inheritance remains a one-target behavior:** `Align` on a
  container still aligns descendant *text* on the DOM only (CSS inheritance;
  the natives read `Align` per-Text). Now documented as adjacent in
  crossaxis.go's story, but unpinned.
- The default theme's `Align: center` lands only on Button — a leaf outside
  the gate — so no theme style trips the new flex trigger today. A theme
  style that later puts `Align` on Column/Card/List will.

## Backlog

In Progress still holds only Packaging (`grmob build --target=…`).

Closed this session: **the DOM `Align` cross-axis fallback** (both DOM
targets, pinned to the natives' semantics).

New this session:

- **A themed Card's `Display: block` overrides its own flex container in
  htmlout** (see residuals above) — affects explicit `AlignItems` too.

Still open from earlier sessions (unchanged this session):

- **Compose drops `Style.Gap` when `JustifyContent` is set** (noted in
  `Renderer.kt` beside the dispatch; blocked on having no Android toolchain).
- **`declStart` and `matchingBrace` are two bounds on the same parse**; with
  `declSource` there are now three consumers — a third bound was predicted to
  argue for a tiny language-aware scanner.
- **The four SwiftUI/Compose dispatch *values* remain uncheckable** — coverage
  says every value has an arm, not that the arm draws the right thing.
- **The tag census is the only table left without a named core type.**
- **The WASM runtime boxes `Fragment` and `Theme` in a `<div>`.**
- Theme contrast, and `Variant`'s third consumer.
- **Hooks have no unmount signal.**
- **`docs/concepts/architecture.md` does not mention frames.**
- **`core.Image` bases its style on `Theme.Components.Camera`.**
- The WASM runtime's style mapping is still thinner than `htmlout`'s.
- **A bottom-docked bar cannot ask for the keyboard on its own.**
- **`htmlout` renders `Scroll` as a plain div** with no `overflow`.
- **A second imperative API would justify the bridge command channel.**
- **`core.SendSystemEvent` is a dead stub.**
- **A `Cached` subtree silently swallows focus commands** and order membership.
- **An app-drawn keyboard toolbar has no worked example.**
- **`imeAction` is a third prop that must not vanish.**
- **`gen.go`'s transcripts still emit no `add`, `remove` or `add-child`.**
- **The demo scenario's tab switches produce only `update-props`.**
- **Nothing runs either verify harness automatically** (the Go-side checks run
  under `go test ./...` regardless).
- **No example app uses `core.TextArea`** (nor `core.Image` / `CameraView`).
- **`docs/platforms/native.md` restates the ContentMode and alignment tables
  in prose**, unpinned.
- **The Kotlin edits from previous sessions remain unbuilt** (no Android
  toolchain here; none were made this session).
