# Session: Holding the alignment switches to core

**Session ID:** session_01LZZTYuh1UXxGjKfkomQRM6
**Date:** 2026-09-01, ~00:51
**Branch:** master
**Follows:** `2026-0901-0029-native-contentmode-coverage.md`

The backlog item said the alignment and justify-content switches were blocked:
"unlike `ContentMode` they have no named Go type to be checked against. Giving
them one is the same 'core needs an enumeration first' problem the tag census
has."

That was wrong, and usefully so. `core.Alignment`, `core.JustifyContent` and
`core.AlignItems` have been named types with declared constants since
`style.go` was written. Nothing was blocked. What was missing was the
enumeration functions and the pins — and, it turned out, three live bugs that
had been sitting behind the missing checks the whole time.

---

# Part 1 — What the survey found before any code was written

Three types, four renderers, and a very uneven picture.

**`JustifyContent` and `AlignItems` were already correct everywhere.** Not by
luck: `htmlout` and the WASM runtime emit them **verbatim**, because core's
spellings *are* the CSS ones. A renderer cannot be wrong about a value it never
interprets. All of the drift risk was native, and pinning them is a regression
guard rather than a repair.

**`Alignment` diverged four ways.**

| `Align(…)` | htmlout | WASM runtime | Swift | Kotlin |
|---|---|---|---|---|
| start/center/end | `text-align` | **dropped** | ✓ | ✓ |
| justify | *nothing* | **dropped** | `.leading` | **`TextAlign.Justify`** |
| baseline | *nothing* | **dropped** | `.leading` | `Start` |
| stretch (cross-axis) | n/a | **dropped** | stretches | **does not stretch** |

`wasm/grmob-runtime.js` never read `style.Align` in **any** form. Not a mapping
gap — an absent prop. Every `core.Align` on the web target was silently thrown
away for the entire life of that runtime.

## The decision that shaped the rest

`Alignment` carries two roles: text alignment (`start`/`center`/`end`/`justify`)
and cross-axis placement (`stretch`/`baseline`, plus the first three again). No
single dispatch serves both, so a flat "every value needs an arm" rule would
have forced a `"stretch"` arm into a text-align switch that could never mean
anything.

Chosen: **split the enumeration, keep the type.** All six constants stay
(nothing removed, no caller breaks); core gains `Alignments()` as the census and
`TextAlignments()` as the subset a text dispatch owes. Rejected: a separate
`CrossAlign` type — cleanest model, but a breaking change for anyone already
passing `AlignStretch` to `Align()`, and the split is real only at the point of
*use*, since `Style` has one `Align` field whose role depends on the node it
lands on.

# Part 2 — `core/alignment.go`, and generalizing the pin

Four enumerations: `Alignments()`, `TextAlignments()`, `JustifyContents()`,
`AlignItemsValues()` (named for its values because the type name is taken — the
same collision `AlignItemsProp` worked around).

They live in their own file rather than beside the constants. `style.go` is the
`Style` struct and ~40 prop helpers; four enumerations plus the paragraphs
explaining what each obliges a renderer to do would be the longest and least
related thing in it. The cost is real and is stated in the test: list and
declaration are no longer on the same screen, so a constant added to `style.go`
is *more* likely to be forgotten — which argues for the pin, not against the
arrangement.

`contentmode_enum_test.go`'s AST reader became `core/enum_pin_test.go`:
`declaredConstants(file, typeName)`, plus `requireExactEnum` and
`requireSubsetEnum`. The ContentMode test shrank from 105 lines to 18.

One thing the generalization had to get right: `style.go`'s const block holds
`AlignItems`, `FlexDirection` **and** the untyped `DisplayFlex = "flex"` at
once. Reading the type off each spec rather than off the block is what keeps
`AlignItemsValues()` from being handed `"row"`.

`TextAlignments()` gets `requireSubsetEnum` instead, which checks only the
direction a subset can be checked in — every listed value must be a declared
constant. That the *omissions* are the right omissions is a judgment about what
text alignment means, argued in the doc comment rather than asserted in a test,
with `TestTextAlignsOmitTheCrossAxisOnlyAlignments` guarding the specific
mistake someone would make (adding `text-align:stretch`, which is not a
keyword).

# Part 3 — `htmlout/textalign.go`, the fourth DOM table

Same arrangement as `objectfit.go`: Go holds the authority, the runtime holds a
copy because it is the side assigning the property, `wasm/verify` keeps them
equal under `go test ./...`. `htmlout`'s inline three-arm switch became a table
lookup and gained `justify` on the way.

The runtime gained `textAlignFor` in the shape the shared JS parser requires,
and — the half that actually mattered — one line in `styleFromGrMob` that calls
it. `TestRuntimeStyleAppliesTextAlign` exists because a table nothing reads
would pass the conformance test and change nothing on screen.

**Physical keywords kept, deliberately.** `start` → `left`, not CSS's
direction-aware `start`. That is what the exporter has always emitted and
changing it is a rendering change, not a table change. It is *also* a
divergence — both natives use the direction-aware spelling, so an RTL locale
would left-align on the web and trailing-align on the natives from the same
`AlignStart` — and no test here can see it, because the two DOM copies agree
with each other exactly. Written into the table's doc comment for that reason.

# Part 4 — Twelve native dispatches, plus one array

Every dispatch now lists every value explicitly, including the ones the
catch-all would have produced. Same argument as the `ContentMode` session: a
value that falls through is indistinguishable from one nobody considered.

Two shapes needed more than a redundant arm:

- **The iOS solver answers `justify-content` with two dispatches**, `leading`
  (offset before the run) and `gap` (space between), each returning 0 for the
  half the other owns. They are checked **separately**. A union of their arms
  would pass if each half answered for values it has no business answering
  for — and the arrangement that makes the solver readable is precisely that
  each states its own complete opinion.
- **Compose's `flex-start` arm could not be a bare `Arrangement`**, because
  that is the one path where `Style.Gap` survives. Extracted as
  `packedHorizontally`/`packedVertically` so the explicit arm and `else` share
  one body rather than two chances to change only one of them.

`justifyClaimsFreeSpace` is a fifth copy of the justify list and the one that
is not a switch, so it is read as an array literal (`stringArray`). Its check is
a **classification**: every `JustifyContent` must be in the array or be
`flex-start`, the single value that cannot spend leftover space. A seventh value
then has to be classified rather than defaulting into "does not claim", which
would make the container hug when it should fill — a layout wrong only when
there is space to spare, which is to say on some screens.

## `coverage`, and why `allowed` exists

Several cross-axis dispatches serve two vocabularies at once: `Style.Align`
doubles as the fallback when `AlignItems` is unset, so a Column's switch
legitimately carries `"start"`/`"end"` arms that are `Alignment` values, not
`AlignItems` ones. Those are **permitted, not required**.

`GrMobRow` gets no `allowed` set at all, and that asymmetry is the encoded
claim: `Align` is a text concept and has never been read for a Row's vertical
axis. `Renderer.swift` draws the same line. A future change to either native
without the other now fails.

---

# Testing the tests

**21 mutations, 21 caught** — but, again, not on the first run.

## The one that got away

`swift leading: switches on something other than justify` went **uncaught**.
The header search ran forward from the anchor and, finding no `switch justify {`
in `leading`, happily walked into `gap` and read *its* arms — which cover the
same six values. The test reported coverage `leading` no longer had.

This is the same defect as last session's `matchingBrace` bug, one level up.
`matchingBrace` bounds the *arms* once the dispatch is found; nothing bounded
the search for the dispatch *header*. Both files are full of lookalikes by
construction — two `switch justify {` in `GrMobFlex.swift`, two identical
`when (s?.alignItems?.ifEmpty { s.align }) {` in `Renderer.kt` — and the anchor
is the only thing telling each pair apart.

Fixed with `declStart`: an intervening **function** declaration between anchor
and header is a fatal. Only functions — `val`/`var` lines are everywhere inside
these bodies (`GrMobRow` opens with `val s = animatedStyle(node.style)`), and
treating them as boundaries would reject every dispatch that is not the first
statement of its function.

The other adjustment: Swift's arm regexp required the body on the *next* line.
That was the test dictating style — `grMobTextAlignment` is a four-line
expression switch whose arms are one value each — so the `$` anchor came off,
making Swift and Kotlin symmetric. The guarantee that arms are pure string
literals lives in `parseLabelList`, where it always did.

| mutation | caught by |
|---|---|
| swift `leading`: arm dropped (`space-around`) | missing-value error |
| swift `gap`: label typo (`center` → `centre`) | unknown-value error |
| swift `gap`: function renamed | named fatal |
| **swift `leading`: switches on something else** | named fatal — **missed until `declStart`** |
| swift `leading`: `default:` deleted | named fatal |
| swift `claimsFreeSpace`: drop `space-evenly` | classification error |
| swift `claimsFreeSpace`: `flex-start` added | classification error (hug-vs-fill) |
| swift `claimsFreeSpace`: non-literal element | named fatal |
| swift `crossOffset`: drop `stretch` | missing-value error |
| swift `crossOffset`: arm for a non-value | unknown-value error |
| swift `crossAlignmentH`: drop `stretch` | missing-value error |
| swift `crossAlignmentH`: duplicate arm | unreachable-arm error |
| swift `grMobTextAlignment`: drop `justify` | missing-value error |
| swift `grMobTextAlignment`: arm below `default` | missing-value error |
| kotlin `verticalArrangement`: drop `space-evenly` | missing-value error |
| kotlin `horizontalArrangement`: unreadable arm | named fatal |
| kotlin `horizontalArrangement`: `else ->` deleted | named fatal |
| kotlin `GrMobColumn`: whens on a different value | named fatal |
| kotlin `textStyle`: whens on a different value | named fatal |
| kotlin `GrMobRow`: drop `stretch` | missing-value error |
| **kotlin `GrMobRow`: adds the `align` fallback it has not got** | unknown-value error |
| kotlin `GrMobList`: drop `center` | missing-value error |
| kotlin `GrMobColumn`: drop `flex-end` | missing-value error |
| kotlin `textStyle`: drop `justify` | missing-value error |
| kotlin `textStyle`: `stretch` arm | unknown-value error |
| core: 7th `JustifyContent` | both solver halves + both Compose arrangements |
| core: 5th `AlignItems` | five dispatches, by name |
| core: `stretch` promoted into `TextAlignments()` | both natives **and** htmlout's census |
| js: runtime table drops `justify` | conformance |
| js: `end` maps to a different value | conformance |
| **js: table kept, but nothing reads it** | `TestRuntimeStyleAppliesTextAlign` |
| js: `textAlignFor` renamed | named fatal |
| go: htmlout table drops `center` | census |
| core: 5 enum-pin mutations (M1–M5) | all, by name |

**5 benign edits, 5 tolerated.** A brace in a line comment, in a block comment,
and in a string literal, inside a dispatch; the word `fun` in prose between
anchor and header; arms reordered with all still above the catch-all. The
matching non-benign edit — the same arm moved *below* the catch-all — is caught.

Two mutations in the first battery were themselves broken (a pattern that
missed a closing brace; a pattern that missed an intervening comment and so
inserted a duplicate arm instead of moving one). Both were re-run correctly and
both are caught. Worth recording: an "uncaught" result is a claim about the
mutation as much as about the test.

Mutations applied and restored from a scratchpad snapshot, not `git checkout`.

## Files touched

**New** (1,023 lines): `core/alignment.go` (the four enumerations and the
two-roles table), `core/enum_pin_test.go` (the generalized AST reader,
`requireExactEnum`, `requireSubsetEnum`), `core/alignment_enum_test.go`,
`htmlout/textalign.go`, `htmlout/textalign_test.go`,
`mobile/verify/alignment_test.go` (the `coverage` helper and ten checks),
`wasm/verify/textalign_test.go`.

**Modified** (+498/−125): `core/contentmode_enum_test.go` (105 → 18 lines, onto
the shared pin), `htmlout/export.go` (inline switch → table lookup),
`wasm/grmob-runtime.js` (`textAlignFor` + the line that reads it),
`ios/GrMob/Runtime/GrMobFlex.swift` (three dispatches, plus the array's shape
constraint), `ios/GrMob/Runtime/Renderer.swift` (two dispatches),
`android/…/runtime/Renderer.kt` (five dispatches, `packedHorizontally`/
`packedVertically`, `isColumnStretch`), `mobile/verify/switchlabels_test.go`
(`declStart`, `stringArray`, shared path helpers, relaxed Swift arm regexp),
`mobile/verify/contentmode_test.go`, and three docs.

Gate: `gofmt` clean, `go vet` clean, full Go suite, `GOOS=js GOARCH=wasm`
build, `node --check`, `wasm/verify/run.sh`, `ios/verify/run.sh` (flex solver +
9 patch batches + Swift typecheck) — all green.
(`examples/todoapp/store.go` remains unformatted — pre-existing, untouched.)

## Behavior actually changed

Three renderers, all in the direction of agreement:

1. **`htmlout` now emits `text-align:justify`** for `Align(AlignJustify)`. It
   emitted nothing before.
2. **The WASM runtime reads `Style.Align` at all**, for the first time. This is
   the largest change in the session and the one with the widest blast radius:
   any app that set `Align` on a web-rendered node was getting nothing and is
   now getting the alignment it asked for. `wasm/verify/run.sh`'s replay passes,
   so no existing transcript disagrees.
3. **`Align(AlignStretch)` on a Compose Column stretches**, matching iOS.

The Swift `"justify"` arm changes nothing — it produces what `default` produced
— because SwiftUI cannot justify text at all.

## Not verified here

**The Kotlin edits are unbuilt.** Android still has no toolchain here. This is
the largest unbuilt Kotlin change so far — five dispatches, two new functions,
one renamed call site — and it is the session's main risk. Every edit is either
an added `when` arm producing the value `else` already produced, or a mechanical
extraction; `isColumnStretch` is the one that changes behavior.

**Swift type-checks but does not run.** The new arms compile; nothing here draws
a pixel.

**No browser.** `run.sh` replays through a DOM shim, not Chrome.

## Backlog

In Progress still holds only Packaging (`grmob build --target=…`).

Closed this session: **all three alignment types are now held to core across
every renderer that interprets them**, and the `Align` prop renders the same on
all four targets for the first time. `mobile/verify` went from one check to
twelve.

Opened this session:

- **Compose drops `Style.Gap` when `JustifyContent` is set.** CSS treats gap as
  a minimum that justify-content adds to, and the iOS solver does the same;
  Compose's five distributing `Arrangement`s take no spacing argument.
  `Arrangement.spacedBy(gap, alignment)` fixes the three packing values;
  nothing expresses gap-plus-distribution for the `space-*` three without a
  custom `Arrangement`. Not attempted — a rendering change on the one target
  this repo cannot build. Written into `Renderer.kt` beside the dispatch.
- **`AlignStart` is `left`, not `start`, on both DOM targets.** Both natives use
  the direction-aware spelling, so an RTL locale disagrees. Invisible to every
  test here, because the two DOM copies agree with each other exactly.
- **Neither native's `List` reads the `Align` stretch fallback**, though both
  read it for the list's *alignment*. The two agree with each other, which is
  why it was left alone; they may both be wrong.
- **`declStart` and `matchingBrace` are now two bounds on the same parse.** A
  third would suggest the anchor model is the wrong shape and the checks want a
  real (if tiny) language-aware scanner.

Still open from earlier sessions:

- **The four SwiftUI/Compose *values* remain uncheckable.** Coverage says every
  value has an arm; nothing says the arm draws the right thing. Now true of
  twelve dispatches rather than two.
- **The WASM runtime boxes `Fragment` and `Theme` in a `<div>`.**
- **The tag census can still go stale in the one direction nothing checks.**
  Node types are string literals at ~21 construction sites. This session
  *removed* the excuse for it: the alignment types show the pattern transfers
  the moment core has a named type, and the tag census is now the only table
  without one.
- Theme contrast, and `Variant`'s third consumer.
- **Hooks have no unmount signal.**
- **`docs/concepts/architecture.md` does not mention frames.**
- **`core.Image` bases its style on `Theme.Components.Camera`.**
- The WASM runtime's style mapping is still thinner than `htmlout`'s — though
  `text-align` closed one of the gaps.
- **A bottom-docked bar has no way to ask for the keyboard on its own.**
- **`htmlout` renders `Scroll` as a plain div** with no `overflow`.
- **A second imperative API would justify the bridge command channel.**
- **`core.SendSystemEvent` is a dead stub.**
- **A `Cached` subtree silently swallows focus commands** and order membership.
- **An app-drawn keyboard toolbar has no worked example.**
- **`imeAction` is a third prop that must not vanish.**
- **`gen.go`'s transcripts still emit no `add`, `remove` or `add-child`.**
- **The demo scenario's tab switches produce only `update-props`.**
- **Nothing runs either verify harness automatically.** Sixteen cross-language
  table and coverage checks now run under `go test ./...` regardless.
- **No example app uses `core.TextArea`** (nor `core.Image` / `CameraView`).

Noticed this session, not acted on:

- **`docs/platforms/native.md` now restates two four-column tables in prose**
  (`ContentMode`'s and the alignment one). Both are unpinned copies, and the
  doc table is the contract the implementations serve — the same argument that
  keeps `replay_test.mjs`'s copy independent. It remains the copy a reader is
  most likely to trust.
- **`core.Alignment` is still one type doing two jobs.** The enumeration split
  makes that legible and testable without breaking anyone, but every consumer
  still has to know which role it is in. A `CrossAlign` type is the honest
  model whenever a breaking change is affordable.
