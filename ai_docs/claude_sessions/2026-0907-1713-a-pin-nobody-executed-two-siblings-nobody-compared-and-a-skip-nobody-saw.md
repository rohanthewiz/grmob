# Session: a pin nobody executed, two siblings nobody compared, and a skip nobody saw

Session: https://claude.ai/code/session_017k47JkhG1WBZR5nP29c7Ge
Date: 2026-09-07 (follows "a-refusal-compose-could-express-two-arms-nobody-could-tell-apart-and-a-version-nobody-had-read")

## Ask

"Show me what's in the Next list", then "take all the high and medium items".

Four items, one commit. The one high and the three mediums — a modifier held up
by a compile, a divergence written down three times and measured nowhere, a
check that skipped where nobody looks, and a coarse cut that had been handing
seventeen call sites a paragraph of English.

| # | age·value | item |
|---|---|---|
| 8 | 0·high | Nothing executes `pinMainAxis` |
| 9 | 0·medium | The Compose sibling distribution after a pin is documented and unmeasured |
| 10 | 0·medium | The source-level half of the Compose census skips on every machine that has not fetched |
| 11 | 0·medium | Only three checks in `mobile/verify` mask their comments |

## Items 8 and 9 — androidx cannot be run, and that is a finding

The Next entry said the arithmetic "could be checked the way `GrMobFlexSolver`
is — if a Compose measure could be run anywhere in this repository, which today
it cannot." That premise was worth testing rather than repeating, so it was:
the 1.6.8 `.aar`s are in the cache on any machine that has built the app, and
`android/verify` already assembles a Kotlin compiler out of the same cache.

It still cannot, and the reason is sharper than "needs the Android runtime":

```
   RowColumnMeasurementHelper   measures Measurables into Placeables
   Placeable                    abstract class with `internal abstract` members
   internal, across modules     cannot be implemented at all — runtime or no
                                runtime, from Kotlin or from Java
```

Also `rowColumnMeasurePolicy` is `internal`, so even the policy cannot be
obtained without a composition. The census's existing sentence is now in
`native.md` with the `Placeable` half added, because "needs the Android runtime"
invites somebody to try again.

### What was built instead

`internal/pinfixture`: a **transcription** of foundation-layout's zero-weight
measure loop, labelled one everywhere it appears, executed over one overflowing
Row. Three children, the pin moved through all three positions, and the control
with none.

```
                        CSS              Compose
    no pin              24, 80, 16       60, 60,  0
    pin first          200,  0,  0      200,  0,  0
    pin middle           0,200,  0       60,200,  0
    pin last             0,  0,200       60, 40,200
```

`ios/verify/pin.swift` solves each row through `GrMobFlexSolver` — this
repository's CSS arithmetic, already split out so it runs without a simulator —
and compares. Three sentences come out, and **each needs a different row to be
visible**:

- **The declaration agrees.** The pinned child is 200 in every pinned row, on
  both targets, wherever it sits. That is what `core.ShrinkNone` says and what
  the fourth target had no way to express.
- **The pin is load-bearing.** The control row is the same three children one
  flag apart, and that child is 80 rather than 200.
- **The siblings diverge.** CSS shares the deficit over everything that can
  shrink, so a flex line's sizes do not depend on the order; a Compose Row hands
  out what is left, so its answer does.

### The numbers were forced, not chosen

The fixture went through three attempts. A pinned base *below* the offer leaves
the pin refusing nothing when it comes first; a pinned base *above* it makes CSS
clamp every sibling to zero, because `deficit < shrinkable ⟺ pinnedBase <
offer`. That is not a fixture problem — it is the reason the pin-first row is
the one where the two targets agree, and the agreement is **two different rules
arriving at one answer**: CSS has nothing left to give, and Compose offered
nothing. Asserting it as an agreement is `band.swift`'s unbadged-band
discipline; saying it is a coincidence is what stops it reading as a shared law.

### Two things the census had never stated

Both fall out of the same lines the rest rests on:

```
   mainAxisLayoutSize = max(fixedSpace + weightedSpace, mainAxisMin)
```

No upper bound — and `Size.kt`'s node reports `layout(placeable.width, …)`
unclamped, verified in the sources jar. So a fixed-width Row whose children
overflow is *measured* wider than it was told to be (200, 260, 300 above) rather
than clipping, which is what makes the pin an overflow rather than a clip. And
`spaceAfterLastNoWeight = min(spacing, what is left)`, so the gap after the
overflowing child collapses with it.

### What holds it up, and the one link that does not

Stated in the package header rather than implied:

```
   the transcription   every branch names the source line it mirrors        ← unchecked
   those lines         read out of the sources jar by mobile/verify
   that version        derived from the BOM's own pom
   the renderer's half shrink_test.go pins the two lines pinMainAxis is
```

`shrink_test.go` gained a paragraph saying it is now that chain's link, and
`native.md`'s table is held to the fixture's Compose column by a test — a doc
table with nothing compiling against it is exactly what rots.

## Item 10 — a skip is a value, not a control-flow accident

`go test` prints a skip only under `-v`, so the check most likely to catch an
androidx release was the one least likely to be noticed missing.
`composeSourcesVerdict(cached, setting)` is now a function of values with all
five states tested, in the shape `gate.sh`'s `jvm_harness_verdict` and
`browser.mjs`'s `startupVerdict` established. Three consumers:

```
   go test ./...                    skips, with the fetch command in the reason
   android/verify/run.sh            runs it by name, prints OK or SKIP beside
                                    the pass's other skips
   GRMOB_COMPOSE_SOURCES=required   turns the skip into a failure
```

A spelling the variable does not take (`1`, `yes`) is refused rather than read
as "not required" — the two spellings a reader reaches for would otherwise
disable the thing they were typed to enable.

Not a fetch wired into the pass. Everything else `android/verify` touches is
already in a cache the app build filled, and a verify script that reached the
network would stop being runnable in the `--offline` position the rest of it is
written for.

## Item 11 — the mask, and what it must not take

The break-test from last session found one instance: `pinMainAxis`'s doc comment
sits below `RowChildren`, `declSource` runs to the next declaration, and a
`strings.Contains(body, "shrinkPinned")` meant to find a call was satisfied by
the paragraph explaining it. Seventeen other call sites had never been looked at.

The obvious fix — `codeOf` everywhere — is wrong, and running it proves it:
**seven existing checks fail**, because `maskSwiftNonCode` blanks string
literals and several checks read a value out of one (`s?.align ?? ""`,
`position == "sticky"`). Two different questions want opposite things:

```
   does it CALL this        a doc comment satisfies it, and so would a literal
   does it LIST this VALUE  the literals ARE the subject
```

So one scanner with a flag. `declSource` masks comments for every caller *and
before the anchor search*, so an anchor surviving only in prose is now "not
found" — `swiftDeclIndices`' rule reaching the coarse cut. `codeOf` keeps the
stronger mask for the file that asks the first question. `labels()` gets the
comment mask too, so a commented-out `case "x":` cannot be counted as an arm or
a mentioned `default:` stand in for a deleted catch-all.

`declSourceOf` is the pure decision, split out for the reason `swiftDeclIndices`
was: the interesting inputs are otherwise reachable only by owning a renderer
with the fault in it.

## The break-tests

**15 run, 15 caught.** Two more were defects in the session's own tests, found by
running them and fixed rather than recorded: a "roomy" Row keeps its definite
width (`mainAxisMin` is the offer, so `RowMain` reads the offers, not the total),
and the agreement rule needs the clamp premise alongside the position — the
control row has no pin and fails a premise written only about pinned ones.

    items 8/9  the flag claims agreement everywhere, the transcription stops
               pinning, the solver is never told the factors, the Row fits its
               children, the spacing never collapses, the Row clamps what it
               reports, every child is offered the whole container, the
               agreeing row is dropped, the control row is dropped, the census
               table drifts, the census row is deleted                       11
    item 10    absent and required, an unrecognised spelling                  2
    item 11    the mask is removed, the mask takes the literals too           2

## Verification

Eight paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh           gate + flex + stack + 3 bands + 4 pinned Rows
                                + 15 menus + replay + view + app
    android/verify/run.sh       gate + the Compose census's source half, named
                                + 15 picker menus + 22 value ranges
    wasm/verify/run.sh          replay + 19 mjs suites + BROWSER PASS
    android ./gradlew compileDebugKotlin --offline
    GOOS=js GOARCH=wasm go build ./...

New files:

    internal/pinfixture/pinfixture.go       the transcription and the four rows
    internal/pinfixture/pinfixture_test.go  the table, the guards, the doc pin
    ios/verify/pin.swift                    the same Rows through GrMobFlexSolver

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

1. **(age 4 · value low) The two shrink controls in the band check cannot guard
   their own deletion.** Dropping the control *and* `min-width: 0` together
   leaves both arrangements at their natural width, agreeing by never reaching
   the arithmetic. The same is true of the scroll control in check 3 and the
   scroller in check 6 — either an argument that this is what a control is, or
   an argument for one source-level pin over the three.
2. **(age 4 · value low) The band check mounts only definite offers and
   `max-content`.** Reading `bandfixture`'s negative offer as the CSS
   `max-content` is a judgement stated in a comment and held to nothing;
   `min-content` is the other candidate and a different question.
3. **(age 4 · value low) `browser.mjs`'s check numbering is not a sequence.**
   The header lists eight and the body has two sections numbered 6, plus 9 and
   10 added to a broken sequence.
4. **(age 2 · value low) `gobindCarriesUnused` refuses six types whose
   parameter spellings are legible.** Only the `(value, error)` out-pointer
   spelling is unreadable, so a bridge function taking an `int8` and returning
   nothing is refused for a reason that does not apply to it. Splitting the
   refusal by position would admit those.
5. **(age 2 · value low) The rendered-band grid mounts nine trees and reads no
   pixels.** Every other browser check that mounts real widgets samples a
   colour; this one measures rects only, so a band that laid out correctly and
   painted nothing would pass.
6. **(age 2 · value low) `maskNonCode` and `matchingBrace` answer the same
   question twice.** Now sharper than when it was raised: the mask became ONE
   scanner with a flag precisely because two copies would be a bug apiece, and
   `matchingBrace` is still the third copy — same arms, same order, brace
   counting instead of blanking. A comment asks them to agree; nothing makes
   them.
7. **(age 2 · value low) The two `gate_test.sh` scripts share a shape and no
   code.** Each has its own `expect` helper, and a third harness's gate would
   write a third.
8. **(age 1 · value low) `ext.composeLayoutVersion` is a second spelling of a
   derived fact.** It exists because the resolution that would derive it cannot
   be asked for a sources jar without dragging in the whole Compose graph. A
   test holds it to the BOM, which is the right shape — but a reader of
   `build.gradle` alone still sees two versions and no mechanism.
9. **(age 0 · value high) The transcription is the one link in the pin's chain
   that nothing checks.** `pinfixture`'s branches each name the foundation-layout
   line they mirror, and only one of those lines (`mainAxisMax - fixedSpace`) is
   actually read out of the sources jar. The others — `coerceAtLeast(0)`,
   `spaceAfterLastNoWeight`'s `min`, `mainAxisLayoutSize`'s missing upper bound,
   `SizeNode`'s unclamped `layout(...)` — are quoted in a Go comment and held to
   nothing. `composelayout_test.go` already has the machinery; the claims table
   just does not list them.
10. **(age 0 · value medium) The pinned Row is not mounted in a browser.** The
    CSS column is `GrMobFlexSolver`'s alone, and the band census records that
    the solver and a real Chrome diverge under overflow when a child has padding
    of its own. These children have none, so the two *should* agree — which is a
    sentence with nothing behind it, and the reason the fixture carries no
    padding rather than carrying some and hand-waving the difference.
11. **(age 0 · value medium) The census table's CSS column is unheld.**
    `TestTheCensusTableIsTheFixture` compares only the Compose column, because
    the CSS numbers are Swift's to produce and transcribing them into Go would
    be writing them a third time. So half of the table a reader trusts is still
    prose — and it is the half that is easier to check, since `ios/verify`
    already computes it.
12. **(age 0 · value medium) Which mask a check gets is decided per file, not
    per question.** `codeOf` is used by `shrink_test.go` and nowhere else, yet
    "does the renderer call this" is what most of the package asks — those
    checks now have their comments masked and their literals intact, so a string
    literal naming a call still satisfies them. The two questions are named in
    `maskNonCode`'s header; the call sites do not choose between them.
13. **(age 0 · value low) `GRMOB_COMPOSE_SOURCES=required` is exercised by no
    pass.** It exists so a machine set up to have the sources hears about it
    when they go missing, and nothing in the repository sets it — so the arm
    that turns a skip into a failure runs only in its own unit test.
