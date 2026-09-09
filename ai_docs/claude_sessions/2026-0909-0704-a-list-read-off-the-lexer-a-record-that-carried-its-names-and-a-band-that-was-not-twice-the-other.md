# Session: a list read off the lexer, a record that carried its names, and a band that was not twice the other

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-09 (follows "a-band-that-was-three-times-wrong-a-grid-complete-in-the-dimension-it-counts-and-ten-assignments-nobody-read")

## Ask

"Do all items in the Next list." Six items, all age 0 — three medium, three
low. Four files plus browser.mjs, no new ones.

| item | where | shape |
|---|---|---|
| 1 | pinfixture_test.go | a fix for a hand-written list, itself hand-written |
| 2 | inkcanary_test.go | the third canary reading, asked by nobody |
| 3 | themenearmiss_test.go | a removal with no family to be a member of |
| 4 | themenearmiss_test.go | a step of two leaves with no band at all |
| 5 | inkglyph_test.go | a plane every walk stopped at without saying so |
| 6 | themenearmiss_test.go | a hazard fixed at one site, looked for nowhere |

Items 3 and 4 came out as one piece of work in two steps: recording the
record's leaf names is what gives a removal a family, and the same move one
level out is what makes a two-leaf step measurable instead of scaled.

---

## Item 1 · the branch set, derived rather than listed

`pinLexBranches` was last session's fix for a pair census complete in the
dimension it counted — and it was a hand-written list of eleven checked for
COVERAGE. A twelfth branch in `pinCodeOnly` left the census reporting 11 of 11
and the grid reporting 20 of 20 with the new route reached by nothing: the same
complaint one level up for the fourth time in that file.

`pinLexDecisionSites` reads the decisions off the function's **syntax tree** —
every `if` condition, `case` expression, `for` condition and boolean assignment
in the body, each split on its top-level `||`, because a disjunction is the
lexer offering separate routes to one arm and the three quote characters are
exactly that. 26 distinct sites. Each is claimed by a row's new `sites` column
or excused in writing by `pinLexNotAConstruct`:

    16  decide which construct the lexer is in — claimed by the eleven rows
    10  extents (how far an already-decided construct runs) and the
        line-number pass's bookkeeping, each with its reason

Identity is the text `go/printer` produces, which is stable under a reformat
and not under a reword — the intended cost, since a condition spelled
differently is one somebody edited and the row above it is prose about the old
one. Assignments are identified by the whole statement, because `single` and
`tick` are both `ext != ".swift"`.

What is still a judgement is the SPLIT between a construct decision and an
extent, and it is now written out per condition rather than exercised by
omission.

**Break-tests — 4 run, every one fired.** A twelfth branch (a raw-string
prefix with its own flag): both the case and the flag named as unclaimed. A
reworded escape arm: an orphaned claim AND an unclaimed decision, both. A stale
exclusion. And a site both excused and claimed.

---

## Item 2 · the reading nothing asked, and the sentence it found

`inkCanaryFaceSpread` was stubbed by the wiring test and asked by nothing else.
It is now lifted and run over eight probe populations this Chrome does not
give — a probe that did not answer, one that answered with no face, a run where
nothing answered, a family list answering per request, the same family at two
glyph counts, the same two faces in the other order.

Asked beside `inkCanaryFaceList` over the SAME probes, because the two walk one
population with the filter `p.faces && p.faces.length > 0` written twice and
the recital takes its CLAUSE from the spread's `answers` and its NOUN from the
list. Held structurally: a list is null exactly when nothing was read, and takes
its per-request form exactly when the answers differed.

### The sentence the probes found

The recital branched two ways on `canvasGenericAnswers === 1`. There is a third
state: a run where every probe came back empty answers **zero**, takes the
second branch, and recites *"0 different faces across them, which is a family
list answering per request and the reason one probe is not enough"* over a run
that read nothing at all. Nothing on this machine produces it, which is why it
stood — the browser pass runs the live population and the live population has
never been empty.

The clause is now `inkCanarySpreadPhrase` in browser.mjs, three branches, one
spelling shared by the sentence and the reading — the same shape
`inkCanvasGenericGap` is split out for.

**Break-tests — 5 run, every one fired.** The third branch removed: both
empty-population rows, naming the sentence they got instead. The spread's
filter drifted. `answers` counted over family names rather than the rendered
list: the glyph-count and ordering rows. The LIST's filter drifted while the
counts stayed right: the null-join arm. And a list that collapses when it
should not: three rows.

---

## Item 3 · a record that carries its names

`leaves: 80` was the whole of what `affordedMeasuredOn` said about its
population, and three readings were arguing about set MEMBERSHIP from that one
integer:

    the exact arm     ran whenever this run had 80 leaves, on a premise about
                      the names. A leaf RENAMED keeps the count and changes the
                      strings the walk is over.
    the added-leaf    is a proof because the record's population is this run's
    arm               names with the new leaf dropped — true when the record's
                      names are a SUBSET, assumed from the count being one lower.
    the removed-leaf  had no arm, because a run one leaf short cannot walk the
    reading           record's population from its own names.

The 80 names are now recorded. The walk is a pure function of them, so:

**The record's own census is re-derived on every run.** Not gated on anything —
it is not about core.Theme at all, it is about `themeLeafSetOf`, and the record
is 80 strings and 4 counts. That is the strongest reading in the file and it is
the one that used to go silent exactly when somebody was editing the struct.
The one-leaf band record got the same treatment (its guard was the same
count-not-names premise, and it fired that way the first time a rename was
tried).

**A leaf REMOVED is now asserted.** `affordedOneLeafBand` over the RECORD's
names walks the 80 populations of 79 — and this run, being those names with one
dropped, is one of them. Same family argument as the addition, in the direction
that had none.

**Both premises are checked by name.** `affordedMissingLeaves` returns the
subset difference; the counting argument (n distinct names covering n of n+1)
is what makes it a subset test, and it is written down.

**Break-tests — 9 run, 8 fired, 1 passed.** Two record counts swapped with the
total unchanged: the re-derivation, naming both. A rename that moves the
classification: the new code stays green and says *"one leaf from the record by
COUNT and not by name"*, while the old guard produced two false "themeLeafSetOf
re-sorted" failures — the bug, demonstrated. A two-leaf record whose names and
`leaves` disagree; a repeat in the names; the band record's stated population
diverging from the census record's; a removal WITH a re-sort (the arm that did
not exist); a +1 with no subset going quiet rather than asserting against the
wrong family. And an HONEST removal: **passed**, naming the dropped leaf — the
family argument holding empirically, which is why the arm can be an assertion.

---

## Item 4 · two leaves are not twice one leaf

The item's guess was that the drift grows about linearly in k, making `band × k`
a bound with evidence. It was measured over all 3160 pairs and it is not:

                                            one leaf   two leaves   ×
    its own crowding stopped ...  losing      24.96%      46.77%   1.87
                                 gaining      33.26%      87.88%   2.64
    no width crowds these ...     losing       0.02%       0.04%   2.24
                                 gaining       0.02%       0.04%   2.24
    the ceiling cost this set ..  losing       2.92%       5.10%   1.75
                                 gaining       2.84%       4.85%   1.71
    the ceiling and the crowd ..  losing      21.63%      38.02%   1.76
                                 gaining      27.60%      61.35%   2.22

Five of the eight ratios are ABOVE two. Twice the one-leaf gaining band for the
crowding ending is 66.5% and two leaves are worth 87.9%, so `band × k` would
have called an honest two-field edit a re-sort — the same failure the stated
tenth had, arriving in the fix for it. Gaining divides by the SMALLER
population's count, so the same step read backwards is worth more.

So `affordedKLeafBand` generalises the walk (`affordedOneLeafBand` is now a call
into it at k=1) and two leaves are MEASURED. The family is still enumerable at
two — 3160 populations, no case anybody chose — so the two-leaf arms are
assertions on the same footing as the one-leaf pair, and
`affordedTwoLeafBandMeasuredOn` is re-derived over the record's names on every
run, which is the widest reading on `themeLeafSetOf` in the file by a long way.

Where it stops is stated: C(80,k) is 80, 3160, 82160, 1.6M. Three leaves is
minutes. The log line's sentence stands past two with a reason attached rather
than an absence.

Walked across the machine (`runtime.GOMAXPROCS`), reduced by (value, lowest
combination index) so the recorded widest pair does not depend on how the work
was split. It is the one reading with a cost worth naming — about five seconds
— and `-short` gives it up, with the log line saying so.

**Break-tests — 3 run, all as intended.** A recorded two-leaf number drifted by
0.0003. An HONEST two-leaf removal: passed, naming both dropped leaves. And a
two-leaf removal with a re-sort: the arm that had no band at all.

---

## Item 5 · the plane every walk stopped at

`for cp := rune(0); cp <= 0xFFFF` in three places and nothing said so as a
bound. The row census had to name "a key outside the BMP" as one of two
possible causes because nothing could tell it from the other.

The own-fold walk is lifted into `foldSpreadOver(lo, hi)` and run twice — the
BMP and the sixteen planes above it, counted APART because they are two
different facts. Above the BMP the fold does nothing but lowercase:

    astralIgnorable   0   inkGlyphIgnorable's ranges all end below U+FFFF, and
                          INK_FOLD_IGNORABLE is a character class in \uXXXX
                          escapes, which cannot name a code point up there at
                          all. Both are silent about U+E0000's tag characters
                          and U+1D173's musical format controls.
    astralTable       0   every row of inkLigatureForms is in U+FB00's block
    astralLowered   260   Go's tables, so gated as the BMP's 1173 is. Deseret
                          at U+10400 first.

The row census now counts astral keys and self-mapping rows separately and
names the cause it found. And `TestTheTwoFoldsDropTheSameCharacters` walks all
1,114,112 code points rather than the BMP — licensed by a new arm asserting the
class carries no `\u{...}` escape, since "absent above U+FFFF" is a claim about
the SPELLING and the parser reads only `\uXXXX`.

**Break-tests — 5 run, every one fired.** A tag-character range added to
`inkGlyphIgnorable`: the two-folds walk names U+E0000 and says the disagreement
can only go one way, and the census names 128 against 0. A table row keyed above
the BMP. A row mapping its character to itself: the row census naming which of
its two causes. A braced escape in the class. And a drifted astral lowercase
count.

---

## Item 6 · the hazard, looked for

Last session's FMA fix gave the residual one definition and prevented nothing.
Looking found three more of the same shape:

    float64(measured) * scale   FOUR spellings. Two of them DECIDED a floor, in
                                opposite directions, at two sites; two printed
                                it beside a residual taken from a third.
    math.Abs(scale - 1)         two spellings, both compared against
                                affordedScaleSame — one choosing which of three
                                findings a failure names, the other choosing
                                which sentence a green run prints.
    float64(part)/float64(whole)
                                the band's step scale and affordedScale's, which
                                the k-leaf assertions require to be the same
                                number: the residual is a member of the band's
                                family only while the two divisions agree.

Each is now one function — `affordedPredictedFrom`, `affordedPopulationUnchanged`,
`affordedRatioOf` — and `affordedResidualOf` returns the prediction it divided
by, so a message naming the number and an arm comparing against it cannot be
reading two evaluations of one product.

`TestNoFloatAComparisonRestsOnIsDerivedTwice` is what stops a fifth spelling.
It reads every `* / + -` and every `math.` call out of the package's test
sources — string concatenation excluded by looking for a string literal inside
— and holds the 13 derivations to being spelled in exactly one function each,
against a table saying what each is and whether a comparison rests on it (7 do).
Three arms: spelled twice, in the source and not the table, in the table and not
the source.

The reading is syntactic and says so. Two limits are named rather than papered
over: arithmetic over two float variables with no conversion and no literal is
invisible to it, and the same recipe under different variable names reads as two
entries — `math.Round(palette.Ratio(edge, lum)*100)/100` and the same over
`la, lb` are one rounding in two files, which the table can say and the
arithmetic cannot.

**Break-tests — 4 run, every one fired.** The prediction re-spelled inline: the
no-entry arm, naming the exact expression and function. A textual duplicate
across two functions: the spelled-twice arm. A new derivation (three, counting
sub-expressions). And an entry for arithmetic that is gone.

---

## Verification

Ten paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      replay + mjs suites + browser pass (rc 0)
    android ./gradlew compileDebugKotlin --offline
    android ./gradlew :app:fetchComposeLayoutSources --offline
    GOOS=js GOARCH=wasm go build ./...

Five files, +2326 −212.

    internal/pinfixture/pinfixture_test.go  item 1
    wasm/verify/browser.mjs                 item 2 (the three-way clause)
    wasm/verify/inkcanary_test.go           item 2
    wasm/verify/themenearmiss_test.go       items 3, 4, 6
    wasm/verify/inkglyph_test.go            item 5

`go test ./wasm/verify` is 6.2s, of which 5.3s is the two-leaf band. `-short`
takes it to 0.6s and gives up exactly two things, both named in the log line.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

1. **(age 0 · value medium) The k-leaf family stops at two and the reason is a
   cost, not a shape.** C(80,3) is 82160 walks — minutes even in parallel — so
   three leaves have no band and the log line says so. What has NOT been tried
   is the chain: dropping k leaves is k successive one-leaf steps through
   intermediate populations, and the residuals compose multiplicatively, so
   `∏(1 + b_i) − 1` is a bound if the one-leaf band at each intermediate SIZE is
   in hand. Those bands are 80 walks each at n−1, n−2, …, which is k×80 rather
   than C(80,k). Whether the composition is tight enough to be useful — the
   measured two-leaf gaining band is 2.64× the one-leaf one and the chain bound
   would predict about 2.2× — is checkable against the k=2 measurement already
   recorded, which is the one place the two can be compared.
2. **(age 0 · value medium) The float census cannot see arithmetic over two
   float variables.** `affordedLooksFloat` is a syntactic stand-in — a
   `float64(` conversion, a `math.` call, or a decimal literal — and `a * b`
   where both operands came from somewhere else is a derivation it does not
   count. Every hazard this file has actually taken carries one of the three,
   which is an argument and not a statement, and it is the same shape of
   argument the FMA note exists to replace. The honest fix is go/types over the
   package, and the cost is a second toolchain inside a test suite that imports
   half the repository; the cheaper one is threading float-ness through local
   assignments, which is a small type inference nobody has written.
3. **(age 0 · value low) The two-leaf band is five seconds and `-short` is the
   only lever.** The measurement is 3160 walks over ~900 sets each, and the
   parallel walk gets about 4× on this machine. Most of the cost is
   `affordedSets` allocating a fresh pair of slices per window per population —
   roughly 6M small allocations — and the walk could reuse buffers the way
   `affordedKLeafBand` reuses `smaller`. Whether that is worth doing to
   test-support code, against simply leaving `-short` as the lever, has not been
   argued.
4. **(age 0 · value low) `affordedTwoLeafBandMeasuredOn` carries no population
   of its own and the one-leaf record now carries a redundant one.** Both are
   measurements of `affordedMeasuredOn`'s names; the two-leaf record says so by
   carrying nothing, and the one-leaf record still states `leaves` and `window`
   which are asserted equal to the census record's. Two records, two
   conventions, for one population. The tidy shape is one record with three
   maps, and the reason not to do it now is that the band records are re-taken
   at different moments from the census.
5. **(age 0 · value low) The astral fold is asserted for gen.go and not for
   browser.mjs.** `foldOwnMeasuredOn`'s three astral counts are held on every
   machine because they need no ICU. The other census — `foldMeasuredOn`, which
   runs node — still walks the BMP alone, and what an astral code point does to
   NFKD is unmeasured. The bound is now stated on the gen.go side and the
   agreement above U+FFFF rests on `INK_FOLD_IGNORABLE` being unable to name
   one; that is the ignorable half. The `inkFold` half — whether the two folds
   AGREE on what they turn an astral character into — is a walk of a million
   code points through node, and nobody has priced it.
6. **(age 0 · value low, deliberate non-goal for now) The record's names are 80
   strings a human has to re-take.** Re-taking `affordedMeasuredOn` now means
   pasting a name list as well as four counts, and nothing generates it. A
   `-update` flag that rewrites the literals is the obvious move and is
   deliberately not taken: a record a test can rewrite is a record that
   re-baselines an accident, which is the failure every message in that file
   warns about. Listed so it is visibly declined rather than quietly missing.
