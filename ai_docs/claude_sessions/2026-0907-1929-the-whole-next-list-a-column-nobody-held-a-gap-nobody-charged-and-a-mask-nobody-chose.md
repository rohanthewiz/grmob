# Session: the whole Next list — a column nobody held, a gap nobody charged, and a mask nobody chose

Session: https://claude.ai/code/session_017k47JkhG1WBZR5nP29c7Ge
Date: 2026-09-07 (follows "one-scanner-one-harness-one-version-four-unread-lines-and-a-row-nobody-had-mounted")

## Ask

"Do the remaining items in the Next list" — all thirteen.

They fall into five groups that turned out to share a shape: **a claim with no
holder**. A CSS column printed beside two mechanisms that produce it and
compared with neither. A line of a measure policy implemented in three places
and measured in none. A guard nothing has executed. A reading settled by a
compiler nobody runs. A scanner with two levels and no call site that chooses.

| # | age·value | item |
|---|---|---|
| 1 | 2·medium | The pin census table's CSS column is unheld |
| 2 | 2·medium | Which mask a check gets is decided per file, not per question |
| 3 | 2·low | `GRMOB_COMPOSE_SOURCES=required` is exercised by no pass |
| 4 | 2·medium | The importer reading runs only where Swift does |
| 5 | 2·medium | `[]byte`'s refusal is now answerable and still stands |
| 6 | 2·low | `checknumbering_test.go` scans a hand-listed set of files |
| 7 | 2·low | The band grid samples one pixel per band |
| 8 | 2·low | The band grid's fold guard cannot be reached |
| 9 | 0·medium | Nothing measures the gap collapse |
| 10 | 0·low | The gate harness's self-test never runs under `go test ./...` |
| 11 | 0·low | `regionOf` cuts `browser.mjs` between two anchors |
| 12 | 0·low | The claims table's windows are byte counts |
| 13 | 0·low | `composeLayoutVersion` re-derives what gradle resolves |

## Items 1 and 9 — the fixture states both columns, and a fifth row

`internal/pinfixture` had one executable column and one written one. `Case.CSS`
is a field now — still stated, never computed here, for the reason it was always
stated: a flex line transcribed into Go would be a third spelling after
`GrMobFlexSolver` and a browser, and every comparison would become a question
about whether two transcriptions agree.

What changed is that a claim is a value, so two things can be held to it:

```
   internal/pinfixture   states     24, 80, 16
   ios/verify/pin.swift  solves     GrMobFlexSolver, whole column, every row
   wasm/verify check 12  measures   a real Chrome, whole column, every row
```

The control row is the case that made it worth doing. Three of the four rows
were pinned to their exact numbers by claims that already existed — the pin
keeps its base, the extents match Compose where the fixture says they do, a
child's width does not depend on where it sits. `24, 80, 16` was determined by
none of them.

And a third statement joined the two columns: `MainsAgreeWithCSS` is derived
from where the pin sits, and the fixture's own test now asserts that the derived
flag, the stated column and the transcription all say the same thing. Two of
three agreeing was never enough — the third is what the harnesses read.

### The fifth row is about spacing and nothing else

`Measured.Gaps` records what the Row actually inserted after each child.
`spaceAfterLastNoWeight` is `min(spacing, what is left)` — a sentence three
documents repeat, implemented in `MeasureCompose`, and measured by nothing,
because every case carried `gap: 0`.

The new case is the pin-middle row again with an 8px gap, and its numbers are
chosen so that **neither column of extents moves**:

```
   CSS      deficit grows by 16, both shrinkable children were already
            clamped to zero                          0, 200, 0   unchanged
   Compose  the first child was offered the whole Row and the pin ignores
            what it is offered                      60, 200, 0   unchanged
   gaps     8, 8   against   8, 0                                the content
```

So the only thing it can be measuring is the gap. A browser has no node to ask,
so check 12 reads child positions and subtracts; the solver charges `spacing` in
`natural()` before it distributes anything, so its answer is the gap repeated.
`GapsAgreeWithCSS` is asserted in both directions on both targets, for the reason
the other flag is.

## Items 7 and 8 — three colours, and a guard with arms

The band grid mounted nine real `components.GroupHeader`s and read **one pixel
each**: the Row's own fill. A band is a fill, a run of words and a count, and two
of the three were unread — a band painting its fill over an invisible label
passed every rect in the check.

The widget grid does not have that shape (a fill *and* both boundary edges), and
the same argument applies. `gen.go` carries `LabelInk` and `BadgeFill` now, read
off the rendered nodes rather than off the theme, and refuses to emit a case
where either equals the band behind it — a pixel that agrees with both answers
is a check that cannot fail.

**The finding.** `DefaultTheme.TextSecondary` is `#3C3C4399` — eight digits. The
words are drawn at 60% over the band, so a screenshot holds the composite and
not the declared colour. Comparing against the declaration would have been
comparing against a colour nothing paints; the check composites, and does the
same arithmetic the browser does.

The label is scanned rather than sampled, because a glyph is a few stems in a
field of backdrop, and it asks two questions that fail differently: *is there
ink* (some pixel is not the fill) and *is it THIS ink* (some pixel is the
composite, within three per channel).

The fold guard was the same four lines in both grids and had never fired — nine
bands fit an 800px window and the only way to trip it is to bundle more themes.
It is `fold.mjs` now, one answer to one question, and `fold_test.mjs` reaches its
arms by handing it two numbers. Its zero-height case is the interesting one: a
capture that came back empty must not read as "everything fits".

## Items 4 and 5 — the last refusal, and the compiler that settles it

`[]byte` was refused for two releases. Its C spelling was never the problem —
`NSData* _Nullable` in both positions, on one line of gobind's own golden. The
refusal named the two-result split, and **the split is one line of
`bind/types.go`**: `isNullableType` is true for a slice because `nil` is
assignable to it. So the value stays the return, exactly as a string's does.

That left two questions, and both are a compiler's:

```
   what NSData* is called      NSData* _Nullable GrMobImportData(NSData* _Nullable);
                               let grMobImportData: (Data?) -> Data? = …
   what the convention does    @protocol GrMobImportErrorConvention
   to a nullable OBJECT        - (nullable NSData *)dataOrError:(NSError**)error;
   returned by a METHOD        let value: Data = try p.dataOrError()
```

The second had been prose since somebody ran a bind once. It is also the arm
`(Iface, error)` uses, so settling it strengthened a row that was already there.

The code had been keying that arm on `ifaces[name]`, which was the same set
while `[]byte` was refused and was never the same claim: the convention turns on
gobind's **annotation**, not on the Go type's kind. `nullableObjectResult` says
so, and `goTypeKey` fixed the latent nil dereference beside it — `swiftResults`
read `first.Name` off an `*ast.Ident` assertion that is nil for every slice.

**The second finding.** The first spelling of the error-convention claim
annotated the wrapper's own return, and `throws -> Data?` type-checked: Swift
widens on the way out, so the annotation proved nothing in the direction that
mattered. Moved onto the value (`let value: Data`), and confirmed by making the
protocol's return `_Nonnull` — at which point the convention declines to rewrite
the method, the error parameter stays, and the line fails.

### And the reading now runs where Go does

`mobile/verify` held two type tables to what `importer.swift` *says*. `swiftc`
is what holds that file to the truth, and it ran in `ios/verify` — a Mac. On
every other machine the pairing was verified, the reading behind it was not, and
a green `go test ./...` could not tell you which.

`TestTheImporterReadingIsSettledByACompiler` runs the typecheck itself.
`importerVerdict` is a function of values with the same five states
`composeSourcesVerdict` has, `GRMOB_IMPORTER=required` turns the skip into a
failure, and `ios/verify/run.sh` **calls the test** rather than spelling the
command a second time.

## Items 3, 12 and 13 — the Compose census

**The required arm had never run.** `composeSourcesVerdict`'s five states are
covered by a table; whether the *check acts on them* was covered by nothing, and
a `t.Skipf` written where a `t.Fatalf` belongs passes that table and leaves the
switch inert. `TestTheRequiredArmFailsAPassRatherThanSkipping` builds a gradle
cache holding the BOM's pom and no sources jar, and runs the census check
against it with the switch and without. Three states, and the pair is what makes
each mean anything. The version derivation's own skip honours the switch now
too — it was out of reach of the one flag that exists to refuse exactly that.

**The windows are measurements of somebody else's file.** 1500, 200, 220, 1600,
each derived by measuring `foundation-layout` as it is shipped. Drift is loud in
one direction and silent in the other: a `want` that falls off the end fails, and
a `notWant` whose window no longer reaches the thing it refuses passes. Every row
must now leave `windowSlack` (40) bytes past its furthest positive claim — 38 is
`).coerceAtMost(constraints.mainAxisMax)`, the shortest plausible spelling of the
absence. The five rows have 1358, 151, 79, 88 and 82.

**gradle writes down what it resolved.** `composeLayoutVersion` derives the
version out of the BOM's pom the way gradle does, and the argument for that was
that a disagreement shows up as a jar that is not there. It does — spelled
`SKIP`, which is the quiet answer for the one thing the arrangement exists to
make loud. `fetchComposeLayoutSources` writes a receipt now, and the BOM
coordinate rides along so a receipt from an older BOM is told apart from a
derivation that is wrong: only one of those is anybody's fault, and only one is a
failure.

The first idea was to glob for any cached sources jar and compare. This machine's
cache holds 1.6.8 *and* 1.10.0, from another project — so that would have been a
false positive on the first run.

## Items 6, 10, 11 — three mechanisms that were nearly right

**The citation list is a walk now.** `checknumbering_test.go` scanned nine
hand-listed files. It was a stated limit, and the limit was real in both
directions: `docs/platforms/native.md` — the census the whole numbering exists to
serve — cited check 12 twice and had never been in it. The list is inverted: it
says which files opt OUT (`core/debug.go`'s own numbered sections, `ai_docs/`'s
record of what was true when written, this file's own examples), each with its
reason, and **every exemption must still carry a citation** — one that stopped
being needed reads as "this file numbers things of its own" long after it does
not.

**`regionOf` ends at a brace.** It cut `browser.mjs` from one anchor to the next,
which is a claim about two declarations being adjacent that the file states
nowhere. It runs to the first column-zero `}` now. Still not a JavaScript
scanner — a rule about formatting rather than about syntax, and honest about
which; what makes it safe is the direction it fails in, since the region can no
longer run long.

**The gate harness's self-test runs under `go test`.** It ran from the two
`run.sh`, which is where its subject is used and also the only place — and those
need a Mac with the CLT or a JDK and a gradle cache. `checknumbering_test.go` is
the precedent for the other answer, and the Go test runs the shell script rather
than transcribing its assertions, which would be the duplication `harness.sh`
was written to end.

## Item 2 — the mask is chosen by the question now

One scanner, two levels, and which level a check got was decided by which helper
somebody reached for — and the helpers were named for what they read:

```
   readNative   the whole file, raw          53 sites
   declSource   one declaration, no comments 21 sites
   codeOf       one declaration, no literals  1 file
```

The strongest reader existed, was correct for a good deal of the package, and
was used by one file out of twenty-five. Five readers now, named for the
question:

```
   codeIn / codeOf       does the renderer DO this
   valuesIn / valuesOf   does it LIST this VALUE      (a dispatch arm is a literal)
   proseIn               does the file SAY this       (a refusal's own wording)
```

**The method is the interesting part.** Every call site was converted to the
code-level reader and the suite was run. Moving to a stronger mask cannot make a
check pass, only fail — so the failures *are* the audit. 43 of them fired, and
each one names which question it was really asking: a check whose subject is a
dispatch arm fails when the arm is blanked, and a check whose subject is a note
fails when the note is. Then a hill-climb tried every site at the strictest
reader that still holds.

```
   codeIn / codeOf       44
   valuesIn / valuesOf   33
   proseIn               11
```

That is not a distribution anybody would have guessed, which is the whole
argument for per-question rather than per-file. `TestEveryNativeReadNamesItsQuestion`
closes the primitives: `readNative` and `declSource` are called by the five
readers and nowhere else, so a new check has to name its question to get a string
at all. It scans the package through `maskNonCode` — the scanner it is about —
because this file's own explanation names both primitives repeatedly.

One accident surfaced on the way: `composelayout_test.go` read a Groovy file
through the code-level mask and worked, because `maskNonCode` does not know
Groovy's single quotes. The subjects there are all coordinates — quoted strings —
so it is `valuesIn` now, and the reason is written down rather than left to hold
by luck.

## The break-tests

**27 run, 27 caught.**

    items 1,9   a control-row CSS literal off by one (Go, solver, browser),
                MeasureCompose stops collapsing the gap (Go, solver),
                the mount stops honouring the fixture's gap                  4
    items 7,8   a label declared in the band's own fill (refused by gen.go),
                the runtime stops applying TextColor, the pill stops
                painting, foldVerdict stops guarding                         4
    items 4,5   []byte spelled Data instead of Data?, the method arm keyed
                on interfaces again, the ObjCBool control written the way a
                rule would have predicted, a protocol return made _Nonnull,
                GRMOB_IMPORTER set to a spelling it does not take            5
    items 3,12  the required arm turned back into a skip in the census
                check, the version derivation stops honouring the switch,
                a window sized to the expression alone                       3
    item 13     the derivation pointed at another artifact, a receipt
                written for an older BOM                                     2
    item 6      a stale citation in a file the old list did not hold,
                an exemption that stopped being needed                       2
    item 11     a neighbour declaration between the two anchors, a cleared
                minimum removed, the two anchors reordered                   3
    item 10     gate_expect stops counting, the script prints no footer      2
    item 2      a check reaching for the primitive, a code question
                satisfied by a string literal naming the call                2

Three of them corrected the work rather than confirming it, and each is recorded
above: the translucent ink, the `Data?` widening, and the two cached
foundation-layout versions.

## Verification

Ten paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       gate harness + gate + flex + stack + 3 bands
                            + 5 PINNED ROWS (both columns, and the spacing)
                            + 15 menus + replay + view + IMPORTER (delegated)
                            + app
    android/verify/run.sh   gate harness + gate + the Compose census's source
                            half + 15 picker menus + 22 value ranges
    wasm/verify/run.sh      replay + 20 mjs suites + BROWSER PASS (12 checks)
    android ./gradlew compileDebugKotlin --offline
    android ./gradlew :app:fetchComposeLayoutSources --offline
    GOOS=js GOARCH=wasm go build ./...

New files:

    internal/gateharness/doc.go             the directory is a package
    internal/gateharness/harness_go_test.go so `go test ./...` runs the shell
    wasm/verify/fold.mjs                    one fold guard, two grids
    wasm/verify/fold_test.mjs               and its arms, reached

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

1. **(age 0 · value medium) The label-ink scan reads one row of pixels.** It
   crosses the label's rect at its vertical middle, which is the row most likely
   to cross a stem — and "most likely" is the whole of the argument. A font
   whose x-height band fell between two stems at that exact y would report a
   label with no ink in it, which is a failure in the safe direction and still a
   failure. Two or three rows would cost nothing.
2. **(age 0 · value medium) The composite is this file's arithmetic.** `over()`
   blends an eight-digit ink over the band in eight bits, and a browser's
   rounding is its own — which is why `INK_EPSILON` is 3 rather than 0. Nothing
   states what the tolerance is bounded by: the distance from the composite to
   any other colour a band paints is far larger, and that is a sentence rather
   than a check.
3. **(age 0 · value medium) `maskNonCode` does not know Groovy's single
   quotes.** It was found by an accident rather than by looking — the gradle
   reads worked under the code-level mask because their subjects survived it.
   The scanner serves four languages now (Swift, Kotlin, Groovy, and Java by
   inheritance) and one of them has a string literal it cannot see.
4. **(age 0 · value low) The five readers are named for three questions, and
   `proseIn` has no `Of`.** A check whose subject is a note *beside one
   declaration* has to read the whole file, because `declSource` blanks comments
   before it cuts. Two sites do exactly that today.
5. **(age 0 · value low) The importer's protocol is one shape of one
   convention.** `GrMobImportErrorConvention` settles the nullable-object return.
   The other two arms — a `_Nonnull` object return keeping its error parameter,
   and a `BOOL` return becoming a plain `throws` — are still the bind somebody
   ran, and both are declarable in the same header.
6. **(age 0 · value low) The receipt is only written by a fetch.** A machine that
   has run `./gradlew` and never the fetch has a pom, derives a version, and is
   compared with nothing. That is the honest state, and it means the loud answer
   arrives only for people who have already done the thing the check is about.
7. **(age 0 · value low) `windowSlack` is one number for five rows.** The floor
   is the shortest spelling of the one absence the table states. A second
   `notWant` about something else would want its own, and nothing would say so.
8. **(age 0 · value low) The citation walk reads every text file in the tree.**
   It skips four build directories and `ai_docs`, by prefix, and sniffs for a NUL
   byte. That is cheap and it is also the whole of what stops it reading a
   vendored dependency somebody adds later.
9. **(age 2 · value low) The band grid samples the badge at a fixed 5px.** The
   pill's radius is 999 and its leading padding is 8, so 5 is inside both — on
   the themes bundled today. It is a number chosen against a widget's current
   insets, which is the shape items 12 and 13 were about one level up.
10. **(age 2 · value low) The pin fixture's spacing row is one gap.** Eight
    points, one arrangement. The collapse is a `min` over two quantities and only
    one of the two is varied — a gap larger than what is left before the pin
    would exercise the other side of it.
