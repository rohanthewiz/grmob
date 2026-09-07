# Session: two numberings of one sequence, three controls nobody could delete safely, and a refusal that had stopped applying

Session: https://claude.ai/code/session_017k47JkhG1WBZR5nP29c7Ge
Date: 2026-09-07 (follows "a-pin-nobody-executed-two-siblings-nobody-compared-and-a-skip-nobody-saw")

## Ask

"take the oldest 5 items in the Next list".

Five items, one commit. Four of them lived in `wasm/verify/browser.mjs`; the
fifth turned out not to need the fix its own entry proposed.

| # | age·value | item |
|---|---|---|
| 1 | 4·low | The two shrink controls in the band check cannot guard their own deletion |
| 2 | 4·low | The band check mounts only definite offers and `max-content` |
| 3 | 4·low | `browser.mjs`'s check numbering is not a sequence |
| 4 | 2·low | `gobindCarriesUnused` refuses six types whose parameter spellings are legible |
| 5 | 2·low | The rendered-band grid mounts nine trees and reads no pixels |

## Item 3 — it was two numberings, not a broken one

The entry said the header lists eight and the body has two sections numbered 6.
Reading both lists, it was worse and more interesting than that: they were
**different numberings of the same eleven checks**.

```
   header   1 tabindex (with the toolbar walk folded into it) … 10 fixed-size
   body     1 tabindex, 2 disabled, 3 ArrowDown, 4 toolbar walk … 6 sticky,
            6 widget palette, 7 ARIA … 10 fixed-size
```

So `check 8` resolved to the band arithmetic or to the ARIA value range
depending on which list the reader had in front of them — and twelve comments
across four languages cite these by number.

The opening sentence turned out to be the one thing that was already right:
"four about the keyboard, two about paint, four about layout, and one about an
accessibility value" sums to eleven, which is what the body actually runs. The
numbered list had lost the toolbar walk; "Eight claims sit in that blind spot"
was stale by three.

Renumbered into **execution order**, so the number a reader sees in the pass and
the number in the list are the same fact. `checknumbering_test.go` is what makes
it one:

```
   both lists are 1..N        contiguous, in order, no duplicates
   entry N opens with         a prefix match, so renaming a check in one
   marker N's own words       place is a failure rather than two names
   both prose tallies = N     the by-kind sentence and "N claims sit…"
   every `check K` resolves   across all six files that cite one
```

Resolving each citation was the slow half and had to be done by intent rather
than by arithmetic. One (`the pinned divergence in check 7`) was ambiguous until
`valuerange.mjs` turned out to be where "pinned divergence" is a term of art —
which settled it as the ARIA check.

## Item 1 — a control cannot guard its own deletion

Three checks open by proving they have a subject. Each control is code in the
same file as the fixture it is about, so deleting *both* is invisible to the
pass — and the band case is not hypothetical: the subject is one
`MinWidth: "0"`, and without it both arrangements sit at their natural width and
agree by never having divided anything.

Nothing inside the file can close that, which is the whole content of the item.
Pinned from Go — and the useful discovery was that **two of the three subjects
are numbers the file states**, so they became arithmetic rather than substrings:

```
   check 3   the page's 4000px filler   >  --window-size=800,600
   check 6   40px band + 6 x 60px rows  >  the Scroll's 160px port
   check 9   MinWidth: "0", twice          — a substring, and named as one
```

The arithmetic pins fail on a fixture edited *to fit*, not only on one deleted.
The remaining substrings are the controls' own failure sentences, joined across
their `+` line breaks first — otherwise the pin would be whichever fragment
happened to fit on one line, breaking on rewording and holding nothing against
the edit that mattered.

Totality is not claimed: a fourth control is unpinned until somebody adds a row,
and the file says so rather than buying a convention the checks would have to
obey.

## Item 2 — measure first, then assert

The entry proposed that reading `bandfixture`'s negative offer as `max-content`
is a judgement held to nothing, with `min-content` as the other candidate. Rather
than pick a side, the fixture was mounted at all three intrinsic keywords with a
temporary probe:

```
   max-content = 164   min-content = 164   fit-content = 164   natural = 164
```

They agree, on every case and both arrangements. So the judgement **was never
load-bearing**, and that is now a measured fact rather than a comment — and it
is a fact about the fixture, not about CSS: every child is a box with a declared
size, so there is nothing to wrap and the three sizings coincide. The check
mounts all three and holds each to the band's natural width, so it fails on the
day a fixture grows real text, which is exactly the day the choice starts
mattering.

## Item 5 — nine real widgets, no pixels

Every other browser check that mounts something real samples a colour; this one
measured rects, so a band that laid out perfectly and painted nothing would pass.
A band's own fill is the whole reason it may span its container edge to edge
rather than being inset like a row, and it is the one thing in the recipe that
geometry is blind to.

`gen.go` now carries the band Row's fill and the page behind it, **read off the
rendered nodes** rather than from the theme, and refuses a theme where the two
are equal — a pixel that agrees with either answer is a check that cannot fail.
The browser samples six pixels into the band, where the Row's leading inset is 0
(that is the move) and the control's own inset is a run of fill with no ink.

The screenshot brings the widget grid's fold rule with it, and the header comment
that said "nothing here reads a pixel, so there is no fold to stay above" is now
wrong and says so.

## Item 4 — the proposed fix was not the fix

The entry said: only the `(value, error)` out-pointer spelling is unreadable, so
splitting the refusal by position would admit the six scalars.

The first half is exactly right — the refusal was type-level where the gap was
position-level, so `func F(int8)` was refused for a reason about results. The
second half is a smaller fix than the situation deserved, because the real gap
was that **nothing here could ask the Swift importer anything**:

```
   gobind emits         an Objective-C header    read: genobjc.go, the goldens
   the shell writes     Swift                    read: nothing
   between them         the importer             ← the unread step
```

That is why two rows of `gobindErrorOutPointer` were marked "read off a real
bind" and every type needing a third was told to go and run `gomobile bind` on a
Mac — an instruction whose own text warned that guessing was the one thing that
must not happen.

`ios/verify/importer.h` declares gobind's C in gobind's own spelling;
`importer.swift` annotates each symbol with what this repository says Swift
imports it as; `swiftc -typecheck -import-objc-header` settles it. **Its control
is that it reproduces the two spellings already read off a real bind** —
including `UnsafeMutablePointer<ObjCBool>`, which is not what a plain `BOOL`
parameter gives and which no rule stated in Go would have produced. If the
arrangement were answering a different question, that is the line that would say
so.

So the seven (six scalars plus `rune`) are spelled now in *every* position, the
refusal is gone rather than narrowed, and `mobile/verify` holds both tables to
the readings in both directions — a row with no reading is the old problem back,
a reading with no row is a stale control. What stays refused, `uint8` and its
alias and `[]byte`, is refused about the **C** rather than about the Swift, which
this arrangement cannot settle and does not pretend to.

## The break-tests

**18 run, 18 caught.**

    item 3  a duplicate number, a check renamed in one place, a stale tally,
            a citation to a check that does not exist, a lost entry           5
    item 1  the filler no longer clears the viewport, the sticky fixture is
            compressed to fit, the cleared minimum is dropped, two control
            messages deleted                                                 5
    item 2  bandTree stops hugging, the natural width moves                   2
    item 5  gen.go reads the page's fill as the band's, the band stops
            declaring one, the declared fill is not what it paints            3
    item 4  a table row drifts, an out-pointer row written by pattern, the
            reading is deleted, the reading is wrong (caught by swiftc)       4

Two of item 5's three were caught by `gen.go`'s new guards rather than by the
pixel read, so a third mutation was written specifically to reach the comparison
— a check that only ever fires in its own preflight is a check nobody has
tested.

## Verification

Eight paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       gate + flex + stack + 3 bands + 4 pinned Rows
                            + 15 menus + replay + view + THE IMPORTER + app
    android/verify/run.sh   gate + the Compose census's source half
                            + 15 picker menus + 22 value ranges
    wasm/verify/run.sh      replay + 19 mjs suites + BROWSER PASS
    android ./gradlew compileDebugKotlin --offline
    GOOS=js GOARCH=wasm go build ./...

New files:

    wasm/verify/checknumbering_test.go  the sequence, and every citation of it
    wasm/verify/controls_test.go        the three controls, from outside the file
    ios/verify/importer.h               gobind's C, in gobind's spelling
    ios/verify/importer.swift           what Swift imports it as

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

1. **(age 3 · value low) `maskNonCode` and `matchingBrace` answer the same
   question twice.** The mask became ONE scanner with a flag precisely because
   two copies would be a bug apiece, and `matchingBrace` is still the third copy
   — same arms, same order, brace counting instead of blanking. A comment asks
   them to agree; nothing makes them.
2. **(age 3 · value low) The two `gate_test.sh` scripts share a shape and no
   code.** Each has its own `expect` helper, and a third harness's gate would
   write a third.
3. **(age 2 · value low) `ext.composeLayoutVersion` is a second spelling of a
   derived fact.** A test holds it to the BOM, which is the right shape — but a
   reader of `build.gradle` alone still sees two versions and no mechanism.
4. **(age 1 · value high) The Compose transcription is the one link in the pin's
   chain that nothing checks.** `pinfixture`'s branches each name the
   foundation-layout line they mirror, and only `mainAxisMax - fixedSpace` is
   actually read out of the sources jar. The others — `coerceAtLeast(0)`,
   `spaceAfterLastNoWeight`'s `min`, `mainAxisLayoutSize`'s missing upper bound,
   `SizeNode`'s unclamped `layout(...)` — are quoted in a Go comment and held to
   nothing. `composelayout_test.go` already has the machinery; the claims table
   just does not list them.
5. **(age 1 · value medium) The pinned Row is not mounted in a browser.** The CSS
   column is `GrMobFlexSolver`'s alone, and the band census records that the
   solver and a real Chrome diverge under overflow when a child has padding.
   These children have none, so the two *should* agree — a sentence with nothing
   behind it. Item 2 above is the shape that would settle it: mount the fixture
   and measure, rather than reason about whether the difference applies.
6. **(age 1 · value medium) The pin census table's CSS column is unheld.**
   `TestTheCensusTableIsTheFixture` compares only the Compose column, because the
   CSS numbers are Swift's to produce. So half of a table a reader trusts is
   still prose — and it is the easier half, since `ios/verify` already computes
   it.
7. **(age 1 · value medium) Which mask a check gets is decided per file, not per
   question.** `codeOf` is used by `shrink_test.go` and nowhere else, yet "does
   the renderer call this" is what most of `mobile/verify` asks — so a string
   literal naming a call still satisfies those checks. The two questions are
   named in `maskNonCode`'s header; the call sites do not choose between them.
8. **(age 1 · value low) `GRMOB_COMPOSE_SOURCES=required` is exercised by no
   pass.** The arm that turns a skip into a failure runs only in its own unit
   test.
9. **(age 0 · value medium) The importer reading runs only where Swift does.**
   `mobile/verify` holds the two type tables to what `importer.swift` *says*;
   `swiftc` is what holds that file to the truth, and it runs in `ios/verify` —
   a Mac. On any other machine the pairing is verified and the reading behind it
   is not, silently. That is the shape `android/verify` grew a named step for
   last session, and this one has none.
10. **(age 0 · value medium) `[]byte`'s refusal is now answerable and still
    stands.** Its stated reason is that no golden exercises `([]byte, error)` —
    but `NSData*` is `_Nullable` in gobind's own golden, so it takes the same arm
    `string` does and stays the return rather than moving into an out-pointer.
    That is one reading away from being a row, and the refusal text still sends
    the reader to a Mac.
11. **(age 0 · value low) `checknumbering_test.go` scans a hand-listed set of
    files for citations.** Six paths, written out. A new file citing `check 7`
    is unchecked until somebody adds it, which is the same explicit-table limit
    `controls_test.go` states about controls — and here it could plausibly be
    derived by walking the repository instead.
12. **(age 0 · value low) The band grid samples one pixel per band.** The fill,
    and nothing else. The widget grid reads a fill *and* both boundary edges,
    because either single edge is consistent with a correct frame; a band's label
    ink and its badge are unread, so a band painting its fill over an invisible
    label would pass.
13. **(age 0 · value low) The band grid's fold guard cannot be reached.** Nine
    bands fit the viewport comfortably, and the only way to trip the guard is to
    add themes. It is written out — like `CompositeWalkNotApplicable`'s arm — and
    stated rather than hidden, but nothing has ever executed it.
