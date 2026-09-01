# Session: Closing the ContentMode gap in the native renderers

**Session ID:** session_01LZZTYuh1UXxGjKfkomQRM6
**Date:** 2026-09-01, ~00:29
**Branch:** master
**Follows:** `2026-0901-0016-objectfit-dedupe.md`

The gap the last session *opened* rather than closed. `core.ContentModes()`
existed by the end of it, and exactly one renderer was held to it.

---

# Part 1 — Why the natives needed a different kind of check

The DOM pair could be deduped because both sides hold literally the same table:
`htmlout` and the WASM runtime each map a mode onto a CSS keyword, so the
values compare. The natives map onto vocabularies with no CSS in them:

```
mode     | CSS object-fit | SwiftUI                         | Compose ContentScale
---------+----------------+---------------------------------+---------------------
fit      | contain        | resizable + scaledToFit         | Fit
fill     | cover          | resizable + scaledToFill + clip | Crop
stretch  | fill           | resizable, no aspect ratio      | FillBounds
center   | none           | not resizable                   | None
```

Three value columns with nothing in common. Only the **key set** is shared by
all four, so the only question a test can ask the natives is *coverage*: does
every mode core declares have an arm of its own?

That turns out to be the half that bites. Both natives fold the unrecognized
case into fit — SwiftUI and Compose have no "unset" to fall back to the way CSS
does — so a fifth `ContentMode` would have drawn as fit on iOS and Android
while both DOM targets fell back to the browser default. Four renderers, two
behaviors, no error anywhere. Not a crash: a design that is subtly wrong on
half the platforms it ships to.

# Part 2 — Making the coverage readable from outside

Both mappings now list **every** mode explicitly, including the one the
catch-all would have handled:

```kotlin
"fit" -> ContentScale.Fit        // redundant with `else` below, deliberately
"fill" -> ContentScale.Crop
"stretch" -> ContentScale.FillBounds
"center" -> ContentScale.None
else -> ContentScale.Fit          // absent, or a mode this build predates
```

Swift is the same shape, and there the redundancy is a duplicated body
(`image.resizable().scaledToFit()` in both `case "fit":` and `default:`). Both
carry a comment saying it must not be folded away, and naming the test that
reads it.

**Rejected on the way there:** a Swift enum (`GrMobScaling`) between the string
switch and the view builder, so the Swift compiler would enforce exhaustiveness
on the drawing. It buys less than it looks: you cannot add a `case` with no
body anyway, so the compiler was only going to re-ask a question the author was
already answering. Dropped as a layer for its own sake.

# Part 3 — `mobile/verify`, a third verify home

`wasm/verify` holds the JS pin; `ios/verify` replays a bridge transcript. But
this rule has to hold in **both** natives at once, and the two candidate homes
were both wrong: duplicating the parse into `ios/verify` plus a new
`android/verify` meant ~30 lines of near-identical machinery in two packages,
and neither belongs to a rule about the pair.

`mobile/verify` because **`mobile` is the bridge surface both native shells are
written against** — "true of every native renderer" is a statement about that
package's consumers. `doc.go` exists only so the test-only directory is a
package `go build ./...` and `go vet ./...` will name rather than skip.

- **`switchlabels_test.go`** — one parser for Swift's
  `switch mode { case "…": }` and Kotlin's `when (mode) { "…" -> }`. The
  languages differ only in punctuation, so the difference is a
  `dispatchSyntax` value (arm regexp, catch-all regexp, its spelling) and
  everything downstream is written once.
- **`contentmode_test.go`** — the two tests, each comparing in both
  directions. A mode with no arm is the gap. An arm with no mode is dead code
  that reads as deliberate support: a typo (`"centre"`) that has quietly never
  matched, or a mode removed from core and left behind.

## Why parse and not compile

Not a shortcut, and worth stating because it looks like one. `default` and
`else` make a string switch **exhaustive by construction**, so "you forgot a
mode" is not a type error in Swift or Kotlin and never will be. `ios/verify`
already type-checks `Renderer.swift` and cannot see this. Only something
holding the arms up against Go's list can notice, which means reading them out
of the source — and doing that in Go is what puts the check inside a plain
`go test ./...`, reachable without Xcode, without the Android SDK, and without
anyone remembering a `run.sh`.

The price is a shape both functions must keep: one arm per line, string
literals first, catch-all last. Stated beside each of them, and every violation
is a named fatal.

---

# Testing the tests

**17 mutations, 17 caught** — but not on the first run.

## The one that got away, and what it exposed

The first version anchored on the function name and then scanned **forward to
end of file**. `Renderer.swift` has two other `case "center":` switches (text
alignment, justify-content) and `Renderer.kt` has its own. So:

- deleting `grMobScaled`'s `default:` arm went **uncaught** — the scan found a
  *later* function's `default:` and cut there;
- deleting Kotlin's `else ->` was caught, but for the wrong reason: it
  complained about arms named `"end"` and `"justify"`, read out of a different
  `when` entirely.

Both are the same defect. The fix is `matchingBrace`: bound the region to the
dispatch's own braces before reading anything out of it. Brace counting has to
skip `//`, `/* */` and string literals, or a `{` in an arm's comment ends the
block early — and an early end reads as missing coverage. Swift and Kotlin
spell all three the same way, so one scanner serves both.

That the mutation battery found this is the point of running one. The test
passed on the real source the whole time.

| mutation | caught by |
|---|---|
| swift: arm dropped (`stretch`) | missing-mode error |
| swift: label typo (`center` → `centre`) | both directions, two errors |
| swift: `grMobScaled` renamed | named fatal |
| swift: switches on something other than `mode` | named fatal |
| **swift: `default:` arm deleted** | named fatal — **missed until `matchingBrace`** |
| swift: arm moved below `default:` (unreachable) | missing-mode error |
| swift: `case "fit", "tile":` | unknown-mode error (multi-label read whole, not truncated) |
| swift: `case "fit", let other where …:` | named fatal, arm unreadable |
| swift: duplicate `case "fit":` | unreachable-arm error |
| kotlin: arm dropped (`center`) | missing-mode error |
| kotlin: `contentScaleFor` renamed | named fatal |
| kotlin: `else ->` deleted | named fatal — **was caught for the wrong reason** before |
| kotlin: whens on `mode.lowercase()` | named fatal |
| kotlin: arm moved below `else ->` | missing-mode error |
| kotlin: `"fit", "tile" ->` | unknown-mode error |
| kotlin: `"fit", in setOf("legacy") ->` | named fatal, arm unreadable |
| **core: a fifth `ContentMode`, unhandled by both natives** | both tests, by name |
| control: unmutated (before, after) | passes |

The `center` mutation needed extra context to be unique — three `case "center":`
in one file — which is itself the argument for anchoring on the function name.

**4 benign edits, 4 tolerated.** A brace in a line comment, in a block comment,
and in a string literal, inside each dispatch. Each would have ended the block
early without the comment/string skipping, so these are the other half of the
`matchingBrace` evidence: it must not fire on braces that are not structure.

Mutations applied and restored from a scratchpad snapshot, not `git checkout`.

## Files touched

`ios/GrMob/Runtime/Renderer.swift` (+15: `case "fit":`, and the comment stating
the shape and why the duplication stays), `android/…/runtime/Renderer.kt` (+13:
same), `mobile/verify/doc.go` (new: why this package exists),
`mobile/verify/switchlabels_test.go` (new: `dispatchSyntax`, `labels`,
`parseLabelList`, `matchingBrace`), `mobile/verify/contentmode_test.go` (new:
the two tests and the coverage comparison), `core/image.go` (+13:
`ContentModes()`'s doc now names all four consumers and their pins, and says
which two are table comparisons and which two are coverage-only),
`docs/platforms/native.md` (+34: the coverage argument under `ContentMode`, and
`mobile/verify` in "Testing without a device"), `docs/platforms/wasm.md` (+12:
the CSS table is no longer the only one held to the list).

Gate: `gofmt` clean, `go vet` clean, full Go suite, `GOOS=js GOARCH=wasm`
build, `wasm/verify/run.sh`, `ios/verify/run.sh` (flex solver + 9 patch batches
+ Swift typecheck — the new `case "fit":` type-checks) — all green.
(`examples/todoapp/store.go` remains unformatted — pre-existing, untouched.)

## Not verified here

**No behavior changed on either native.** The new `fit` arms produce exactly
what the catch-all produced; that is the whole reason they could be added
safely.

**The Kotlin edit is unbuilt.** Android still has no toolchain here, so the
only evidence for it is that it is a one-line `when` arm and a comment. This is
the one asymmetry between the two halves of the session: Swift type-checks,
Kotlin does not.

**Still no browser. Android still unbuilt. iOS still type-checks without
running.**

## Backlog

In Progress still holds only Packaging (`grmob build --target=…`).

Closed this session: **all four `ContentMode` copies are now held to
`core.ContentModes()`**, and every one of those checks runs under a plain
`go test ./...`. The chain has no unpinned link: `image.go`'s `const` block
→ (`go/ast`) `ContentModes()` → (census) `htmlout.objectFits` → (parse) the
WASM runtime, and → (parse) `Renderer.swift`, `Renderer.kt`.

Opened this session:

- **`mobile/verify` has exactly one check in it.** The machinery is general —
  any prop dispatched on by string in both natives could use it — but nothing
  else uses it yet. The obvious next candidates are the alignment and
  justify-content switches, which have the same shape and are *also* restated
  four times, but unlike `ContentMode` they have no named Go type to be
  checked against. Giving them one is the same "core needs an enumeration
  first" problem the tag census has.
- **Two native switches now carry a machine-read shape constraint**, and a
  reviewer's instinct on both is to simplify it away — the Swift `case "fit":`
  duplicates the `default:` body verbatim. Comments say not to. Nothing
  enforces that a future rewrite reads them, though a rewrite does at least
  fail loudly.

Still open from earlier sessions:

- **The four SwiftUI/Compose *values* remain uncheckable.** Coverage says every
  mode has an arm; nothing says the arm draws the right thing. That needs a
  rendering test on a device or simulator, which is the same wall everything
  native hits here.
- **The WASM runtime boxes `Fragment` and `Theme` in a `<div>`** where every
  other renderer treats them as transparent.
- **The tag census can still go stale in the one direction nothing checks.**
  Node types are string literals at ~21 construction sites, not constants of a
  named type; making the `ContentMode` treatment transfer means giving core a
  node-type enumeration first.
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
  sticky sentinel.
- **`gen.go`'s transcripts still emit no `add`, `remove` or `add-child`.**
- **The demo scenario's tab switches produce only `update-props`.** Still not
  understood.
- **Nothing runs either verify harness automatically.** All the cross-language
  table and coverage checks now run under `go test ./...` regardless — that is
  four of them, and the gap this note describes keeps shrinking.
- **No example app uses `core.TextArea`** (nor `core.Image` / `CameraView`).

Noticed this session, not acted on:

- **`core/image.go`'s doc comments restate the CSS mapping in prose**, and
  `docs/platforms/native.md` restates the whole four-column table. Both are
  unpinned copies. The doc table is arguably the contract the four
  implementations serve, which is the same argument that keeps
  `replay_test.mjs`'s copy independent — but it is a copy, and it is the one a
  reader is most likely to trust.
- **`matchingBrace` is a small language-agnostic scanner living in a test
  file.** If a third native check ever needs it elsewhere it will want a real
  home; today it does not.
