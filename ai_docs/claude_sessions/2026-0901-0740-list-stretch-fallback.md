# Session: Closing the List Align-stretch fallback gap

**Session ID:** session_01HFQHJembyEppoKJCujVjga
**Date:** 2026-09-01, ~07:40
**Branch:** master
**Follows:** `2026-0901-0223-rtl-textalign.md`

Worked the next backlog item: **"Neither native's `List` reads the `Align`
stretch fallback; the two agree with each other, but may both be wrong."**
They were both wrong, in the same way, and the agreement was exactly what kept
it invisible.

---

## The bug

On both iOS and Android, `Align(AlignStretch)` with `AlignItems` unset did
nothing on a `List`. Each List's *placement* dispatch reads `alignItems` with
the `Style.Align` fallback, and its `"stretch"` arm says "the fill modifier
handles this" — but each *fill* binding tested `alignItems` alone
(Renderer.swift's `(s?.alignItems ?? "") == "stretch"`, Renderer.kt's
`isStretch`, the **Row** spelling). The value took the arm's word for a fill
that never happened: rows placed at the start edge, stretched nowhere, while a
Column with the identical style stretched on both platforms.

The DOM needed nothing: htmlout never maps `Align` to `align-items`, but List
rows fill the cross axis there anyway (block flow, or the flex
`align-items: stretch` default once Gap makes the div a flex column). So the
natives' Columns and both DOM targets already agreed that stretch should
stretch; only the two List fill bindings missed it.

## The fix

- **`ios/GrMob/Runtime/Renderer.swift`** — extracted `crossAxisValue(_:)` as
  the one authority on the `alignItems → align` fallback read. Three readers
  now: `crossAlignmentH` (kept as `let v = crossAxisValue(s)` + `switch v {`
  so the verify anchor still matches), `GrMobList`'s stretch binding, and
  `GrMobFlexStack`'s `crossAlign` (vertical axis only — the Row-axis
  asymmetry stays at the call site, semantics unchanged).
- **`android/.../Renderer.kt`** — `GrMobList`'s item loop calls
  `isColumnStretch` instead of `isStretch`: a List's cross axis is horizontal
  like a Column's, so the column spelling — the fallback-aware one — was the
  right helper all along. The `isStretch` doc block records the episode.
- **`mobile/verify/alignment_test.go`** — new
  `TestListStretchFillReadsTheAlignFallback`. The coverage checks only reach
  switches and this bug lived in an equality, so the pin is a two-level
  substring hold: the fill binding must read the fallback-aware helper
  (`crossAxisValue(s)` / `isColumnStretch(s)`), and the helper must still
  contain the `align` expression fragment (`s?.align ?? ""` /
  `ifEmpty { s.align }`). Expression fragments rather than names so a comment
  cannot satisfy the pin.
- **`mobile/verify/switchlabels_test.go`** — new `declSource` helper: cuts a
  declaration's source at the next function declaration rather than by
  braces, because `isColumnStretch` is expression-bodied and has no block to
  bound.
- **`docs/platforms/native.md`** — "What this found" gains the List
  paragraph, following the existing `isColumnStretch` Column story.

## Gate

`gofmt` clean, `go vet` clean, full Go suite green, `ios/verify/run.sh` green
(replay + flex solver + Swift typecheck, which covers the `crossAxisValue`
extraction and the `GrMobFlexStack` rewrite). Mutation-checked the new pin at
all four points: reverting either platform's fill binding, or gutting either
helper's fallback read, each fails with its named message; all restored and
re-verified green.

The Kotlin edit is unbuilt as before (no Android toolchain here), but it is a
swap to an existing helper with the same signature the old call used.

## The residual worth knowing

- The pin guards **stretch only**. The non-stretch fallback values
  (`Align: center`/`end` on a List) were already correct on the natives — but
  the whole cross-axis fallback remains native-only: the DOM never maps
  `Align` to `align-items`, so `Align: center` on a List centers rows on
  device and only centers *text* on the web. Part of the documented
  asymmetry, not this item, but adjacent to it.
- `declSource`'s cut is coarse: a declaration's doc comment rides along with
  the source below it. The pin substrings are expression fragments precisely
  so prose cannot satisfy them; keep that property if the pin grows.

## Backlog

In Progress still holds only Packaging (`grmob build --target=…`).

Closed this session: **the List `Align` stretch fallback** (both natives).

Still open from earlier sessions (unchanged this session):

- **Compose drops `Style.Gap` when `JustifyContent` is set** (noted in
  `Renderer.kt` beside the dispatch; blocked on having no Android toolchain).
- **The DOM does not read the `Align` cross-axis fallback at all** — promoted
  from a footnote above; centering/end-packing a List via `Align` works on
  device only. May be deliberate (documented as native-only), but nothing
  pins the choice.
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
- **The Kotlin edits from this and previous sessions remain unbuilt** (no
  Android toolchain here).
