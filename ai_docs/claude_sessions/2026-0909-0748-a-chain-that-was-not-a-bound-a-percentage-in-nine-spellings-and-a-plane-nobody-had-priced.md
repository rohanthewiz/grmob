# Session: a chain that was not a bound, a percentage in nine spellings, and a plane nobody had priced

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-09 (follows "a-list-read-off-the-lexer-a-record-that-carried-its-names-and-a-band-that-was-not-twice-the-other")

## Ask

"Do all items in the Next list." Six items, all age 0 — two medium, three low,
and one that was listed as a deliberate non-goal. Two files, no new ones.

| item | where | shape |
|---|---|---|
| 1 | themenearmiss_test.go | a composition that would reach past k=2 |
| 2 | themenearmiss_test.go | a census blind to arithmetic over two variables |
| 3 | themenearmiss_test.go | five seconds somebody guessed the cause of |
| 4 | themenearmiss_test.go | two records, two conventions, one population |
| 5 | inkglyph_test.go | a million code points nobody had priced |
| 6 | themenearmiss_test.go | eighty names a human re-types (declined) |

Four of the six came back with an answer different from the one the item
predicted. That is most of what this session is.

---

## Item 1 · the chain composes exactly and does not bound

The item's construction: dropping k leaves is k one-leaf drops, the residuals
compose, so `∏(1 + b_i) − 1` is a bound costing k×80 walks instead of C(80,k).

**The identity is exact and is now written down.** With c_i the ending's count
over the i-th population of a chain and s_i that step's scale,
`1 + r_i = c_(i+1)/(c_i·s_i)`, and the scales telescope to S(n−k)/S(n) — which
is the number `affordedScale` divides by. The intermediate counts cancel:

    1 + r_total = ∏ (1 + r_i)

The only empirical part is whether the two spellings of the k-step scale agree
in floating point. They are one ulp apart, and that is now an arm
(`affordedScaleCompose`, a part in 1e12) rather than a premise nobody checked —
the chain bound and the band it is compared against would otherwise be dividing
by two roundings of one number, which is the hazard this file has already been
caught by once.

**The bound is not a bound, and the number says so.** `affordedChainBoundOf`
walks one representative chain (these names with the first i dropped, a rule
rather than a choice) and composes. Against the measured two-leaf band:

                                            chain    measured
    its own crowding stopped ...  losing    56.13%     46.77%   covers
                                 gaining    77.55%     87.88%   UNDER
    no width crowds these ...     losing     0.04%      0.04%   ties
                                 gaining     0.04%      0.04%   covers
    the ceiling cost this set ..  losing     5.97%      5.10%   covers
                                 gaining     5.80%      4.85%   covers
    the ceiling and the crowd ..  losing    47.92%     38.02%   covers
                                 gaining    62.79%     61.35%   covers

One of the eight is under, and it is the same ending and direction `band × k`
fails on: a chain bound would have called an honest two-field edit a re-sort.
Seven cover and are loose by up to 26%, because a product of maxima is the case
where every step is simultaneously at its worst.

**Why it fails is not the looseness.** The b_i have to bound the steps of every
chain, and a band measured over ONE representative population of size n−1 does
not bound a step out of the other 79. The sound version takes the largest
one-leaf residual out of ANY population of that size — 80 walks at n−1 plus the
3160 at n−2 — which is *more expensive than the direct measurement it was meant
to replace*, and still looser. So the composition is not cheaper, not tighter
and not sound, and all three are now recorded rather than argued.

The comparison is asserted in the direction that keeps it honest: the arm fires
when the composition covers all eight, because covering is what would make it
usable and the log line would otherwise carry a finding nobody re-derives.

Comparing rounded on both sides, deliberately: the small ending's two numbers
agree to twelve places and differ in the last bit, and calling that "under"
would be reporting a rounding as a finding.

**Break-tests — 4 run, every one fired.** A drifted chain record. The steps
inflated so the composition covers everything (the covers-all-eight arm, with
the record arm beside it). A chain entry for an ending that does not exist. And
the composed scale multiplied by 1.0000001.

---

## Item 2 · the census that could not see a product

`affordedLooksFloat` is a text test — `float64(`, `math.`, or a decimal literal
— and `a * b` over two float variables carries none of the three. It is now the
first of two tests. The second is a small inference over the syntax
(`affordedFloatSource`, `affordedFloatLocals`): which package functions return
floats and at which positions, which struct fields are declared as floats, and
which locals hold one — taken to a fixpoint, because Go does not require the
assignment to precede the use. The text test is still asked first, so nothing
the census used to count can stop being counted.

It is not go/types and says which three things it cannot reach: methods,
anything in another package, and block scoping (an identifier that is a float
anywhere in a function marks every expression mentioning the name). Each miss
costs an entry somebody has to write rather than arithmetic nobody sees.

### What it found

**Nine expressions in eighteen places, all one multiply** — a fraction printed
as a percentage. Two of the nine were the SAME expression in two functions,
`b.losing * 100` in `affordedBandFor` and in the log line, which is precisely
the shape this census reports and about which it could say nothing at all. They
are `affordedPercent` now. `two.losing / one.losing` and its partner became
`affordedStepRatio` the same way.

**Two were the inner halves of entries already here** — `scale - 1` under its
own `math.Abs`, `v * 10000` under its own `math.Round` — invisible alone
because the float-ness lived in the wrapper.

**And a container of floats reaches it through the type**: `map[string]float64`
marks the name, so `losing[ending] - 1` — the last step of the chain
composition — is arithmetic and not a map lookup. Without that the composition
this session added would have been censused nowhere.

13 derivations before, 25 now, 10 of them carrying a comparison.

**Break-tests — 5 run, every one fired.** A percentage re-spelled inline (the
no-entry arm — the duplicate detector is textual, so a re-spelling under other
variable names arrives as an unentered derivation, which is the documented
limit). A product of two float locals. A subtraction of two float struct
fields. An entry for arithmetic that is gone. And one inferred derivation
copied into a second function, which is the spelled-twice arm.

---

## Item 3 · the five seconds are not where the item said

`affordedEachSet` replaces the per-window pair of slices. The prefixes are a
function of the NAME and not of the window, so three prefixed lists are built
per population and every set is a sub-slice of one of them — including the
two-parent arrangement, where "Left." on the window's even positions is the
absolute position's parity XORed with the window start's, so two lists cover
both cases. `affordedSets` is a collector around it: one loop, not two.

Measured over one 78-name population on an M3:

    the loop this replaced   96.7us   4057 allocations
    this walk                 7.4us    243 allocations
    the whole census        2530.0us  60556 allocations

So the sets were 3.8% of the census's time and 6.7% of its allocations, and end
to end the band barely moved: 5.5s/6.0s/6.8s before against 5.5s/5.6s/5.7s
after — a few percent, mostly the spread narrowing. **The five seconds are
`themeLeafSetOf`**, which allocates two maps and a distance matrix per set and
is called 2.9 million times, and that is the derivation under test rather than
test-support code.

The walk is kept anyway — thirteen million allocations for four numbers is not
a thing to leave behind — and what the measurement settles is that `-short` is
not carrying a cost this could have removed.

### And a hole the rewrite opened, which nothing could see

Breaking the parity produced no failure. The mirror set is the same partition
of names under swapped parent labels, so every distance, every ending, every
band and every record in the file is blind to it: a walk that lost the
alternation would measure a population nothing here could tell from this one.
So the arrangement is asserted directly — the two-parent sets alternate and
begin with Left — and that arm is the only thing in the file that would say so.

**Break-tests — 3 run, all fired after the arm was added.** The parity dropped
(silent before the arm, named after it). The one-parent prefix list built from
a stale name. And the record's own census re-derivation catching both.

---

## Item 4 · one record, one population

`affordedBandMeasuredOn` and `affordedTwoLeafBandMeasuredOn` are one record
keyed by step, plus the chain bound:

    step[1]  eighty populations, each of the record's names dropped in turn
    step[2]  3160, every pair
    chain    what the one-leaf band composes to over a two-leaf step

The leaves-and-window fields are gone. They were a second copy of
`affordedMeasuredOn`'s, asserted equal to it, and what makes them unnecessary
rather than merely redundant is that every band here is re-derived over the
record's own names on every run. The population is stated once.

The three arms per step are one loop over `affordedBandSteps`, with the varying
prose in three small functions (`affordedStepWord`, `affordedStepWider`,
`affordedStepMissing`, `affordedStepRetake`). Two new arms: a step measured
with nothing recording it, and a step recorded that nothing measures.

**A defect found while unifying.** The two-leaf arms called
`affordedBandFor(..., len(names), ...)` and its sentence said "a single leaf" —
so both cited a two-leaf band as a one-leaf one, over a population of 80 where
the family has 3160 members. The assertions were against the right numbers and
described them wrongly, which is this file's own complaint in the other
direction. `affordedBandFor` now takes the step.

**And a stale number in a note.** The log line's ×-column divided the
measurement by the ROUNDED record, so the small ending's ratio printed 2.24×
when the two measurements are 2.03× apart. Both ends are live numbers now, and
`affordedKLeafBand`'s table is corrected — including "five of the eight are
above two", which was four in the numbers it was written beside.

**Break-tests — 5 run, every one fired.** A record for a step nothing measures.
A step measured with no record. A drifted one-leaf number (naming the step). An
ending missing from the two-leaf step. An ending in the record the derivation
does not have.

---

## Item 5 · the plane nobody had priced

`foldMeasuredOn`'s census walked the BMP through node and nothing said what
NFKD does above U+FFFF. The reason given was that nobody had priced a walk of a
million code points through a child process.

**It is 128ms and 30KB of JSON.** The bound was not a cost; it was a walk
nobody had run. The script now walks all 1,114,112 code points and the census
is counted per plane-set — two different facts, the way `foldOwnMeasuredOn`
counts gen.go's own width:

    astralChanges   2307   what browser.mjs's fold changes above the BMP
    astralAgreed     260   which is EVERY code point gen.go's fold touches up
                           there (foldOwnMeasuredOn.astralLowered, the same
                           number reached from the other side)
    astralGap       2047   entirely NFKD reaching where the narrow fold does not
    astralBearing    257   and turning it into a letter a seed is spelled with

The seed-bearing population above the BMP is not the BMP's made smaller: NFKD
decomposes U+1CCD7 to "b" and the mathematical alphanumerics to bare letters,
so 257 characters complete a pair for `inkLigatureNote` and complete none for
the refusal. What holds every one of them out is the printable-ASCII arm again
— and for a different reason, since nothing up there can be inside two ASCII
bytes at all. The witness is built and asked on both sides of U+FFFF.

The "gen.go may not be WIDER" edge now runs over every plane too, which is what
the astral half buys that the gen.go-only walk could not: whether node
lowercases the same characters Go does is a fact about two vendors' Unicode
versions, and until this walk went past U+FFFF nothing compared them.

**Break-tests — 3 run, every one fired.** The walk cut back to the BMP (three
arms, including gen.go reading as wider on Deseret). A character gen.go folds
and NFKD does not, above the BMP. And a drifted astral seed-bearing count.

---

## Item 6 · the transcription, without the flag

Listed as a deliberate non-goal: re-taking the record means pasting eighty
names and four counts, and a `-update` flag is the one thing it cannot have,
because a record a test can rewrite re-baselines an accident.

The decline stands and the pain is gone. A run that FAILED prints the whole
record in source shape — names wrapped as the literal wraps them, endings and
bands sorted so two runs can be diffed — measured over THIS run's names, which
is what the record would become. What is automated is the copying; what stays
with a reader is the decision, and the header says so.

It costs the two-leaf walk a second time and is paid only by a run that has
already failed, which is the run where somebody was about to do it by hand.

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

Two files, +1405 −356.

    wasm/verify/themenearmiss_test.go  items 1, 2, 3, 4, 6
    wasm/verify/inkglyph_test.go       item 5

`go test ./wasm/verify` is 6.5s against 6.0s before — the chain bound is two
extra one-leaf bands (~0.35s) and the astral walk ~0.15s. `-short` is 1.29s
against 1.13s, and still gives up the two-leaf band, the chain bound and the
3160-population reading on `themeLeafSetOf`.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

1. **(age 0 · value medium) The two-leaf band re-classifies windows a drop did
   not change.** The allocation measurement found the real cost:
   `themeLeafSetOf` is 96% of the census and is called 2.9 million times. Most
   of those calls are re-asked answers — a k-drop leaves every window that does
   not span the gap identical to one the full population already classified, so
   about thirty of the 906 sets per population are new. An incremental census
   would be 15–30× cheaper and would take the five seconds to under one, making
   `-short` almost unnecessary and putting C(80,3) within reach at last. What
   stops it being obviously right is that it is a SECOND walk to hold against
   this one, and the file's own rule is that two copies of a loop drift. The
   defence that would work is structural rather than statistical: the multiset
   of name-sequences the incremental path accounts for must equal what
   `affordedEachSet` produces for the reduced population, which is checkable for
   every one of the 3160 at about 1% of the cost of classifying them.
2. **(age 0 · value medium) The chain bound is measured along one chain and the
   sound one has never been taken.** `affordedChainBoundOf` walks the
   representative chain (drop the first i names) because that is what makes it
   cheap, and the finding is that it does not cover. What has NOT been measured
   is the sound version — b_1 as the largest one-leaf residual out of ANY
   population of n−1 — which would say how much of the failure is the
   representative being unlucky and how much is the product being the wrong
   shape. It costs 80 walks at n−1 plus the 3160 at n−2 that the direct
   measurement already pays, so it is affordable exactly once, as a reading
   rather than as a bound. If the sound chain DOES cover at k=2 it is evidence
   the construction is worth its cost at k=3, where nothing else can reach.
3. **(age 0 · value low) The float census cannot see a method, a call into
   another package, or a block scope.** The inference reads package-level
   functions' result types, struct field types and local assignments, and each
   of the three gaps is named where it is taken. `palette.Ratio(a, b) * 2` is
   invisible unless the expression carries a text mark, and the two entries
   that use it do — which is the same "true of the hazards found" argument the
   old text-only reading rested on, one level in. go/types is still the honest
   answer and still costs a second toolchain inside this suite.
4. **(age 0 · value low) `affordedScaleCompose` is a tolerance with one
   machine behind it.** A part in 1e12, against a measured one ulp on arm64.
   The number it protects is the premise of the chain composition, and what
   would make the tolerance a measurement rather than a choice is the same
   thing that made the band one: a population. Nobody has asked what the
   composition error looks like across the leaf counts the walk can produce —
   it is 80 divisions and would fit in the run that already takes them.
5. **(age 0 · value low) The astral witness is one string and the BMP's is
   another, and neither names its own block.** The witness arms build from the
   FIRST seed-bearing character found, which is U+1CCD7 today and was chosen by
   the walk's order rather than by anything about the population. 257 astral
   characters complete a pair; the arm asks one. A census of which BLOCKS they
   fall in — the mathematical alphanumerics, the outlined forms, whatever
   Unicode 17 adds — would say whether the printable-ASCII arm is holding one
   family or several, and that is the question a reader of the number would
   actually ask.
6. **(age 0 · value low) The re-take paste is printed and never parsed.**
   `affordedRetakeSource` writes the literal a person pastes, and nothing
   checks that what it writes would compile or that it matches the shape the
   record is actually in — a field renamed in the struct leaves the printer
   emitting a paste that no longer applies, silently, on the one run where
   somebody needs it. The cheap holding is the one this file already uses on
   browser.mjs: parse the printed text back and compare it against the record's
   own declaration in the source.
