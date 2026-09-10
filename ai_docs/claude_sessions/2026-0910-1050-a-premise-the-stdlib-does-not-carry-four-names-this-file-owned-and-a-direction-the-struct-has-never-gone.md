# Session: a premise the stdlib does not carry, four names this file owned, and a direction the struct has never gone

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-10 (follows "a-lever-that-was-not-where-the-note-said-a-walk-taken-twice-in-one-run-and-five-collisions-nobody-had-counted")

## Ask

"Do all items in the Next list." Six items, all age 0 — one medium, four low,
one a declared non-goal. Two files, no new ones. Three of the five actionable
items were raised with a premise attached and two of those premises were
wrong, which is again most of what this session found.

| item | where | shape |
|---|---|---|
| 1 | themenearmiss_test.go | a coupling held by position and nothing else |
| 2 | themenearmiss_test.go | one walk taken four times in a run |
| 3 | themenearmiss_test.go | five conflated names, four of them ours |
| 4 | themenearmiss_test.go | a judgement the git history could price |
| 5 | inkglyph_test.go | a block table Go does not have |
| 6 | — | a sound chain bound at k = 3, declined again |

---

## Item 1 · the second copy that is the assertion

`affordedEndingAt` returns 0, 1, 2, 3 as literals and `affordedEndingNames`
lists four sentences in that order. The item said a reordering "silently
re-labels every census, band and record in the file, and every arm goes on
passing" — **that half is not quite right**: the record's census is keyed by
the sentence and re-walked on every run, so the counts arrive under the wrong
headings and that arm fails.

What it does not survive is a **re-take**. A person who reorders the array,
sees the census arm fail and pastes a fresh record has a file that is
internally consistent, states the wrong sentence for every population it
counts, and has no arm left that would say so.

So `affordedEndingWitness` writes the four sentences down a second time,
beside a set that reaches each — and that second copy is the assertion rather
than a duplicate to be removed, because it is the one copy
`affordedRetakeSource` does not print.

Two arms per witness, because they are two findings:

    index      the switch's branches still stand where the array's entries are
    sentence   the array has not been reordered underneath them

Neither implies the other. `affordedEndingOf` is
`affordedEndingNames[affordedEndingAt(set)]`, so a reorder of the array alone
leaves every index right and every sentence wrong, and an edit to the switch's
literals leaves them both wrong the other way.

The four fixtures `TestTheNearMissThresholdMovesWithTheNamesItIsMeasuredOver`
built by hand were the same four populations, so the names moved into the
table and that test reads them from it. Two copies of a fixture list is how
this file came to have two copies of the ending list.

**Break-tests — 2 run, 2 fired.** The array reordered (the sentence arm, plus
the census arm three hundred lines away). Two of the switch's return literals
swapped (the index arm, in the other direction).

---

## Item 2 · the walk taken four times, and the key that would not have been caught

The one-leaf band over the record's names is a pure function of a list of
strings, and a green run asked for it four times: the steps loop, the
composition's b₁, `affordedChainBoundOf`'s first step (`names[0:]`, the same
list), and — while core.Theme still has the record's population — this run's
own band.

    a walk of 80 names     13ms
    a reuse                under 3µs
    the test               0.78s → 0.72s

Memoised on `(names, k)` with the answer copied out, because handing the same
map to two callers makes a caller that writes into it edit an answer somebody
else is about to read — a bug that would look exactly like a re-sort. Walks
and reuses are counted and both are in the log line: a `reused` that drops is
a caller asking about a **different population**, which is a change in what is
being read rather than in what it costs.

**Break-tests — 3 run, 2 fired.** A key that drops the step (conflates k = 1
and k = 3 over the record's names). A key that drops the names (conflates the
chain's two steps).

The third did not fire and is the finding: a key of `(step, len(names))`
survives this population, because the three walks a green run takes are 80 at
k = 1, 80 at k = 3 and 79 at k = 1 — no two of them the same size. It would be
wrong the first time a leaf was **renamed**, which is the same
premise-by-count the exact census arm carried before it learned to read the
record's own strings. Written into the key's note rather than left as a green
break-test nobody explained.

---

## Item 3 · four of the five were ours to rename

The item said renaming the walk's pair takes the census from five conflations
to three, "and what is left is `chain`, `got` and `measured`, which are
collisions between unrelated structs". **Two of those three were also this
file's own** — local comparison tables inside the test, not distant structs:

    losing    affordedBand's float, and the k-leaf walk's [4]best
    gaining   the same
    chain     affordedBandMeasuredOn's map, and a float in the arm asking
              whether the composition covers
    measured  affordedBracket's int, and a float in two comparison tables
    got       affordedTaker's int, and a float64 in widget_test.go

The census reports and does not resolve, which is right for the general case —
telling two same-named declarations apart is what go/types is for — and is not
right for a collision where both declarations are in this package and one of
them is a local table nobody reads by name. Four renames, no number moved:
`widestLosing`/`widestGaining` in the walk, `band` and `composed` in the two
tables.

**Five conflations → one.** What is left is `got`, an int here and a `float64`
in widget_test.go: two files with nothing to do with each other, where neither
name is wrong and no local edit separates them. That one stays counted, and
the log line now says what a number there MEANS — a name two files disagree
about, which is the case this census cannot resolve without go/types.

**Break-test — 1 run, counted not failed** (which is the arm's design): the
old names put back, and the count reads 3 rather than 1.

---

## Item 4 · what the git history says about four leaves

Read by parsing `core/` at every commit that touched it and expanding `Theme`
to distinct leaf names syntactically — the reading reproduces
`affordedMeasuredOn`'s eighty exactly at HEAD, which is what says it is
measuring the same population. Sixteen commits moved it:

    leaves moved   commits   what they were
               1         8   a role, a state, a flag: one field, one decision
               2         2
               3         2   three Accessibility fields; Border/Success/Warning
               4         1   Max/Min/Now/Text — a slider's value range
               5         1   a second tone per palette role
              28         1   the style-props surface arriving at once

Both halves of the old note's guess were wrong. It said the fourth
simultaneous edit "is a rarer thing than the third by about the margin the
cost says" — the margin is **two to one** against a cost margin of six. And
there is no k at which the family becomes complete: a five-leaf edit happened
as often as a four-leaf one, and edits arrive **feature-sized** —
Max/Min/Now/Text is one slider, the five is one decision about palette roles,
and a feature's size is not bounded by what this test can afford to walk.

So the list stays at {1, 2, 3}, and now for a reason with a distribution under
it: k ≤ 3 covers 12 of the 14 non-bootstrap edits, k = 4 would cover 13 for
685ms — six times the rest of the test — and would still leave the "outside
the band is the absence of a finding" sentence in place for the fourteenth.

The reading ends where `affordedMeasuredOn`'s population begins, which is how
a reader tells whether it is current: while the record says 80 leaves, the
table covers every edit there has been.

### And the direction nothing has ever gone

**Every one of those sixteen commits ADDED leaves.** Not one removed or
renamed a leaf name in the whole history of the struct. So `affordedBand`'s
`losing` number — a population read against a LARGER one — is a band over a
direction core.Theme has never taken, and every arm that has ever fired
against a real edit fired against `gaining`. Still measured, because the day
somebody removes a field is the day it has to be there; written down because a
reader weighing what these numbers have caught should know the two halves have
not had equal exposure.

---

## Item 5 · the block table Go does not have

The item said "Go's `unicode` tables carry block ranges, so the block a region
falls in is derivable rather than typed." **They do not.** `unicode` carries
Categories, Scripts and Properties and no Blocks table at all, and adding
`golang.org/x/text` to a repository with three dependencies for a log line's
prose is not a trade worth making.

So a region is named by what the stdlib does carry — the scripts its code
points are in and the categories they belong to, one word per category:

    U+00CC..U+02E2   Latin letters/modifier letters
    U+1D2E..U+1ECB   Latin letters/modifier letters
    U+2071..U+217C   Common/Latin letters/modifier letters/numerals/symbols
    U+249D..U+24E3   Common symbols
    U+2C7C           Latin modifier letters
    U+3250..U+33FF   Common symbols
    U+A7F3           Latin modifier letters
    U+FF22..U+FF54   Latin letters

Less evocative than "the enclosed CJK squares", and derived, which the
sentence was not. Members only, not the whole span: a region is runs merged
across gaps of up to `foldBearingGap`, and reading the gaps drags in
Cyrillic, Greek and Hangul (the second break-test).

### And what the derivation found on the other plane

    U+1CCD7..U+1CCE9  assigned in neither script nor category by Go's
                      Unicode 15.0.0, so newer than it

The prose called that one "the outlined letters" — a claim about a Unicode
16.0 block **this toolchain does not know exists**. The regions come from
node's NFKD at 16.0 and the naming from Go's tables at 15.0.0, and the two
versions disagreeing now shows up in the log line instead of being papered
over by a sentence.

Both namings are recorded and asserted, gated on BOTH versions matching their
records. That closes a gap the counts could not see: 96 runs in 8 regions says
nothing about WHICH code points they hold, so a build whose NFKD re-drew every
region would have printed eight different spans under the same two numbers and
the same sentence.

**Break-tests — 2 run, 2 fired.** The regions named off the wrong plane's runs
(which also turned up an empty-region corner that read as "newer than Go's
tables" and now says what it is). The naming taken over each region's whole
span rather than the runs inside it.

---

## Item 6 · declined again

A sound chain bound at k = 3. `affordedTwoStepBands` exists only for two and
there is no reason to write the three-step version: bounding the last step
over every chain needs a census of every population of size n−k, which is the
family the direct measurement walks — and the direct measurement is the
cheaper of the two at every k this file takes. Kept at k = 2 for the SHAPE it
holds, not as a route to anything.

---

## Verification

Ten paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      rc 0
    android ./gradlew compileDebugKotlin --offline
    android ./gradlew :app:fetchComposeLayoutSources --offline
    GOOS=js GOARCH=wasm go build ./...

Two files, +673 −68.

    wasm/verify/themenearmiss_test.go  items 1, 2, 3, 4
    wasm/verify/inkglyph_test.go       item 5

`go test ./wasm/verify` is **1.80s against 1.84s**. The float census's field
conflations are **1, down from 5**. 8 break-tests run; 7 fired, and the eighth
not firing is written into the code as a finding rather than deleted.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

1. **(age 0 · value medium) The witness table's four sets are held to their
   BRANCHES and not to their shapes.** `affordedEndingWitness` carries a
   `shape` string — "five siblings each one edit from the other four", "one
   parent holding two leaves" — and nothing re-derives it. The arm asserts
   which ending each set reaches, so a fixture that drifted into reaching the
   right ending for the WRONG reason still passes: `Duo.Alpha`/`Duo.Zulu`
   edited to three names under one parent would go on reaching ending 1 by a
   different route, and the prose beside it would be describing a set that no
   longer exists. Two of the four shapes are countable directly — leaves per
   parent, and the closest sibling distance — and the set carries both
   (`closestD`, and `byParent` is what the open-flag arm already computes one
   test over.)
2. **(age 0 · value low) The threshold test's probe names are literals beside
   a table they are not read from.** `themeNearMiss(sparse, "Palette.Alfa")`
   and its four siblings — `Palette.Zulu`, `Ramp.Zzzzzz`, `Duo.Mike`,
   `Weight.Zzz` — are typed into the test while the populations they probe now
   come from `affordedEndingWitness`. A witness whose parent was renamed from
   `Palette.` leaves every probe naming a parent no set has, and the arms would
   report "themeNearMiss found nothing" rather than "the probe is not about
   this set". The parent is derivable from the witness's first name.
3. **(age 0 · value low) The band memo's counters are process-global and the
   sentence reading them says "this run".** `affordedBandMemo` accumulates for
   the life of the test binary, and `walks`/`reused` are read once, by the
   affordance test's log line. The numbers happen to be that test's own —
   because it is the only caller of the band walks in the package, which is
   something nobody asserted and which the next test to want a band would
   quietly end. Either the counters are snapshotted at the top of the test that
   prints them, or the sentence says it is counting the process.
4. **(age 0 · value low) `foldRegionKinds` maps seventeen categories and Go's
   tables carry thirty-one.** The fall-through prints the two-letter category for
   anything unmapped, which is honest, but the mapping was written by looking
   at what these two populations happen to contain. A build whose NFKD reached
   into, say, `Nd` in a region would print `numerals` and one that reached `Cf`
   would print `Cf`, and the difference between "a category nobody thought
   about" and "a category deliberately left as its code" is invisible. Go's
   `unicode.Categories` has the full list and the gap is enumerable.
5. **(age 0 · value low) The git-history reading is prose and dated, and the
   tool that took it is gone.** Item 4's table was produced by a throwaway
   AST walker in a scratch directory; what is left in the file is the numbers,
   the method in a paragraph, and the commit it was taken to. That is this
   file's convention for a measurement, and it is weaker than the convention
   the rest of the file has moved to — the record's census is re-walked on
   every run precisely because a number in a note is a number that has already
   moved. A test cannot shell out to git reliably, but the walker itself is
   forty lines and could live in the repository so the next reading is a
   command rather than a re-derivation.
6. **(age 1 · deliberate non-goal) A sound chain bound at k = 3.**
   `affordedTwoStepBands` exists only for two, and there is no reason to write
   the three-step version: bounding the last step over every chain needs a
   census of every population of size n−k, which is the family the direct
   measurement walks — and the direct measurement is now the cheaper of the two
   at every k this file takes. The composition is kept at k = 2 for the SHAPE
   it holds, not as a route to anything. Listed so it is visibly declined
   rather than quietly missing.
