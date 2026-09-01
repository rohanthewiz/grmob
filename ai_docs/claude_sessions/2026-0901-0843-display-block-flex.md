# Session: Stopping Display: block from killing htmlout's flex container

**Session ID:** session_01HFQHJembyEppoKJCujVjga
**Date:** 2026-09-01, ~08:43
**Branch:** master
**Follows:** `2026-0901-0833-dom-align-fallback.md`

Worked the next unblocked backlog item, the one the previous session
surfaced: **"A themed Card's `Display: block` overrides its own flex
container in htmlout — affects explicit `AlignItems` too."** htmlout-only,
no Android toolchain needed, and it directly undermined the alignment work
just finished. Committed as `cd21a90`.

---

## The gap

`styleValue` emitted `Display` *after* the flex declarations, deliberately,
so an "explicit Display (set by the author)" would win the browser's
last-declaration-wins parse. But the merge in `containerNode` erases who set
what, and `DefaultTheme`'s Card style carries `Display: block` — so every
themed Card's *own theme* was killing the `align-items` its author asked
for, whether set explicitly or through the `Align` cross-axis fallback.

One target out of four disagreed: the natives read `Display` only to honor
`"none"` (Renderer.swift:59 and Renderer.kt:118 both bail out before any
layout), the WASM runtime deliberately emits no `Display` at all
(`styleFromGrMob`'s comment explains the el.style overwrite hazard), and
htmlout alone let a valid `"block"` beat the container.

## The fix

All in `styleValue` (htmlout/export.go): `Display` is now *resolved against
the flex container* rather than simply emitted last. On a flex container:

- `"none"` still lands after `display:flex` and wins — hiding beats layout
  on every target that reads Display at all.
- `"block"` is not emitted — a block-level flex container is exactly
  `display:flex`; the mode's whole meaning is already stated.
- `"inline"` folds into the container as `inline-flex`, the one CSS
  spelling that keeps both the inline level and the flex layout.
- `"visible"`/`"hidden"` are not CSS display keywords; the browser was
  dropping them as invalid after the flex declaration anyway, so the dead
  declaration is no longer written.

A node that is not a flex container keeps the verbatim, last-position
emission unchanged. The stale aside in `wasm/grmob-runtime.js`'s
Display comment (which described htmlout's old invalid-declaration behavior)
was rewritten to describe the sibling problem and its resolution; no runtime
*code* changed, so the verify pins were untouched.

## Tests

Five new pins in a "Display against the flex container" section of
`htmlout/export_test.go`:

- **The themed Card**, built from `core.DefaultTheme.Components.Card` with
  `AlignItems` on top (the file's convention is Node-in/HTML-out, no
  Context), with a *premise guard*: the test fatals with "premise gone" if
  the theme ever stops setting `Display: block`, because the test's story is
  only true while it does.
- **The Align-fallback variant** — the fallback computes align-items later
  than the explicit prop, so it gets its own pin.
- **none-beats-flex pinned by declaration order** (index compare), not just
  presence — the browser honors whichever valid declaration comes last.
- **The inline-flex fold**, with a `strings.Count(out, "display:") == 1`
  assertion so a stray second declaration cannot reopen the fight. (Note
  `"display:inline"` is a substring of `"display:inline-flex"` — the count
  is what makes this test sound.)
- **Verbatim emission on non-flex nodes** — the resolution is strictly about
  the conflict, not a new opinion on Display.

## Gate

`gofmt` clean, `go vet ./...` clean, full Go suite green,
`wasm/verify/run.sh` green. Test cache cleared and re-run fresh after the
final comment edit. Mutation-checked at three points, each failing by name:
unconditional Display emission (caught by the themed-Card, fallback, *and*
inline tests), a gutted inline-flex fold, and a dropped `"none"` escape.

**Process stumble worth remembering:** mutation 1 was restored with
`git checkout htmlout/export.go`, which reverted to HEAD and wiped the
*uncommitted fix itself*. Caught immediately, both edits re-applied
verbatim; mutations 2 and 3 were restored with reverse perl edits instead.
When mutation-testing uncommitted work, restore by reversing the mutation,
never by checkout.

## The residual worth knowing

- **`Display: none` is now the last Display disagreement across targets:**
  it hides a node on both natives and in htmlout, but the live WASM runtime
  ignores it entirely (emits no Display, for the documented overwrite
  hazard). Fixing it there needs care on the patch path — the guarded
  `if (style.X)` assignment convention means un-hiding (Display removed from
  the Style) would not reset the property. Falls under the existing "runtime
  style mapping thinner than htmlout's" item, but is now the sharpest edge
  of it.
- `DisplayFlex = "flex"` (core/style.go) is an untyped string constant
  sitting in the FlexDirection const block — harmless with the new
  resolution (a redundant trailing `display:flex` is skipped on flex
  containers), but an oddity someone may trip on.
- The `visible`/`hidden` DisplayModes still export as invalid CSS
  (`display:visible`) on non-flex nodes, verbatim, dropped by the browser.
  Pre-existing, unchanged, unpinned.

## Backlog

In Progress still holds only Packaging (`grmob build --target=…`).

Closed this session: **the themed Card `Display: block` vs flex container
conflict in htmlout** (both the explicit-AlignItems and Align-fallback
paths, pinned and mutation-checked).

New this session:

- **The WASM runtime ignores `Display: none`** while the other three targets
  hide (see residuals — the sharpest edge of the thin-style-mapping item).

Still open from earlier sessions (unchanged this session):

- **Compose drops `Style.Gap` when `JustifyContent` is set** (blocked on
  having no Android toolchain).
- **`declStart` and `matchingBrace` are two bounds on the same parse**; a
  third consumer was predicted to argue for a tiny language-aware scanner.
- **The four SwiftUI/Compose dispatch *values* remain uncheckable.**
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
- **A `Cached` subtree silently swallows focus commands** and order
  membership.
- **An app-drawn keyboard toolbar has no worked example.**
- **`imeAction` is a third prop that must not vanish.**
- **`gen.go`'s transcripts still emit no `add`, `remove` or `add-child`.**
- **The demo scenario's tab switches produce only `update-props`.**
- **Nothing runs either verify harness automatically** (the Go-side checks
  run under `go test ./...` regardless).
- **No example app uses `core.TextArea`** (nor `core.Image` / `CameraView`).
- **`docs/platforms/native.md` restates the ContentMode and alignment tables
  in prose**, unpinned.
- **The Kotlin edits from previous sessions remain unbuilt** (no Android
  toolchain here; none were made this session).
- **`examples/todoapp/store.go` fails a repo-wide `gofmt -l`** (pre-existing
  import ordering from the rebrand commit, still untouched).
