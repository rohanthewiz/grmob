# Session: a census that corrects instead of walking, a bound that turned out to be one, and a paste nobody had read

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-09 (follows "a-chain-that-was-not-a-bound-a-percentage-in-nine-spellings-and-a-plane-nobody-had-priced")

## Ask

"Do all items in the Next list." Six items, all age 0 — two medium, four low.
Two files, no new ones. The first item paid for the second, and the third found
a duplicate in code written the same afternoon.

| item | where | shape |
|---|---|---|
| 1 | themenearmiss_test.go | 2.9M classifications, most of them re-asked |
| 2 | themenearmiss_test.go | the chain bound nobody had taken soundly |
| 3 | themenearmiss_test.go | three gaps in a syntactic reading |
| 4 | themenearmiss_test.go | a tolerance with one machine behind it |
| 5 | inkglyph_test.go | 257 characters and no idea where |
| 6 | themenearmiss_test.go | a paste printed and never parsed |

---

## Item 1 · the census that corrects instead of walking

The two-leaf band walked 3160 populations of 78 names and censused each one:
2.9 million classifications, five seconds. Last session's allocation fix found
the sets were 3.8% of it and `themeLeafSetOf` the rest — which is the
derivation under test and not this file's to make cheaper.

What IS this file's is how often it asks. Windows are runs of consecutive
names, so dropping a leaf leaves almost every window where it was. `affordedWhole`
classifies the whole population once, per window, and a sub-population is that
census with two corrections:

    take out   every window of the whole that holds a dropped name
    put in     every window of the smaller population whose names are NOT
               consecutive in the whole

and the two are exactly complementary, because a window holding no dropped
name is a run of consecutive kept names. About thirty of the 906 sets per
population are questions nobody has already answered.

    k = 1        8ms
    k = 2      463ms   (was 5.3s)
    k = 3     12.4s
    k = 4     about four minutes

The test went 5.7s → 1.78s and the package 6.5s → 2.8s.

**And `-short` is gone from that test.** The gate existed for a five-second
reading; it is now half a second, and leaving it would have meant a log line
explaining that a green short run had skipped three thousand populations to
save that.

### What holds it

Not an argument about index arithmetic. Three readings, none of them chosen:

    the record's own 80        every one-drop censused BOTH ways — corrected
    one-drops                  and walked — and compared. 225ms.
    every drop of one to       over the first fourteen of this run's names.
    four names                 The bookkeeping is a function of the positions
                               and the window bound, not of the names, which
                               is why a shorter list is evidence about the
                               longer one: C(14,4) is 1001 and C(80,4) is 1.6
                               million. 503ms.
    the window accounting      out − in must equal W(n) − W(n−k) whatever the
                               drop was. Asserted on every population of every
                               walk, and carried out through the walk's own
                               report rather than raised inside it.

Beyond those: `affordedBandMeasuredOn.step[2]`'s eight numbers were measured by
the direct walk before any of this existed, and are re-derived through the
corrected census on every run.

### And the mounting nothing could see

The windows put back are gathered by index rather than sub-sliced, which is a
second way of applying one rule. Breaking its parity produced **no failure** —
the mirror set is the same two groups under swapped parent names, so every
ending, band and record here is blind to it. `affordedParent` is that rule in
one place now, and a new arm mounts every index list of one to six positions
drawn from fourteen names — 6475 of them — and checks each position against it.

**Break-tests — 4 run, 4 fired.** A window range off by one. The contiguity
test inverted. The gather's parity (silent until the arm was added — that is
the finding). And the prefixed list's own parity rule.

---

## Item 2 · the sound chain bound, which covers

Last session found that composing the one-leaf band along ONE representative
chain is not a bound: one of eight comes in under the measured two-leaf band.
That failure had two possible causes and could not tell them apart — the
representative population was unlucky, or the product is the wrong shape.

`affordedSoundTwoStepBound` separates them. b₁ is the largest one-leaf residual
out of ANY population of n−1: every ordered pair of drops, each of the 3160
two-drops read against both of the populations it can be reached from. 557ms,
because a drop census is a correction now.

**It covers all eight.** So the representative was unlucky and the shape is
fine:

                                            sound    measured    ×
    its own crowding stopped ...  losing    86.61%     46.77%   1.85
                                 gaining   163.03%     87.88%   1.86
    no width crowds these ...     losing     0.04%      0.04%   1.00
                                 gaining     0.04%      0.04%   1.00
    the ceiling cost this set ..  losing     6.54%      5.10%   1.28
                                 gaining     6.33%      4.85%   1.30
    the ceiling and the crowd ..  losing    53.63%     38.02%   1.41
                                 gaining    73.16%     61.35%   1.19

Which turns it from a finding into a **check**. The sound bound dominates every
two-drop residual by construction, so it dominates the measured band by
construction — and the two walks share nothing but the classification: one
reads each population of 78 against the whole, the other reads it against the
two populations of 79 it can be reached from. A run where the bound does not
cover is one of the two being wrong. That arm fires in the opposite direction
from the cheap chain's, because one is a heuristic that was tried and the other
is a theorem.

It does not rescue the construction, and now the reason is general rather than
empirical: **bounding the last step over every chain needs a census of every
population of size n−k, which is exactly the family the direct measurement
walks.** There is no k at which the chain gets there first.

**Break-tests — 4 run, 4 fired.** Only one chain into each pair. A drifted
record. The second step's scale taken as the two-step one. And a bound made to
fail the domination it guarantees.

---

## Item 3 · the census reads methods, other packages, variadics and ranges

Three gaps were named last session. Two are closed:

    method   result types by NAME, without the receiver
    from     functions in another package OF THIS REPOSITORY, keyed by the
             name the import is used under — the module root is found by
             walking up for go.mod, and `palette.Ratio(a, b)` is a float
             because internal/palette's own source says so

Two more turned up while closing them: a variadic `...float64` param, and the
value of a range over anything holding floats.

### What it found, in code written the same afternoon

**A duplicate.** The chain composition was spelled in `affordedChainBoundOf`
and again in `affordedSoundTwoStepBound` — `1 + b.losing` in both — which is
the identity two readings rest on, written twice, against bands one of them has
to dominate. It is `affordedComposedOf` now: one place, `∏(1+b_i) − 1`.

**The palette multiply.** `palette.Ratio(edge, lum) * 100` is arithmetic the
old reading could not see; the `math.Round` around it was an entry and the
multiply inside it could not be.

13 derivations before this file learned to infer, 25 last session, 28 now, 13
of them carrying a comparison. Block scope is still the remaining gap and still
errs the safe way — an entry somebody has to write, never arithmetic nobody
sees.

**Break-tests — 2 run, 2 fired.** Arithmetic on another package's float result.
A method returning one.

---

## Item 4 · a tolerance with a population behind it

`affordedScaleCompose` was 1e-12 justified by one number: the difference at
n = 80, on one machine, called "one ulp" in a comment.

`affordedComposeSpread` walks every population size from three to the record's
— 78 divisions, nothing beside the walk that reads them. **The worst is at five
names, not eighty**: the small-population edge is real and it is still one ulp
(1.4803e-16), so what the measurement says is that the size does not matter.
The constant used to assume that.

Three arms now: the spread under the tolerance, the spread within a decade of
the recorded one (a decade rather than exactly — this is the one recorded
number here that is not a count or a rounded band, and the multiply that
produces it is one the compiler may fuse on one machine and not another), and
the tolerance keeping a 1000× margin over the measurement, which it clears by a
factor of seven.

**Break-tests — 3 run, 3 fired.** The recorded spread off by a decade. The
tolerance tightened without re-reading the measurement. A scale taken in a
different order.

---

## Item 5 · where the 257 actually are

The astral seed-bearing count was a number with no shape behind it. 257
characters could be two blocks a reader can look up or 257 singletons scattered
over sixteen planes, and the count cannot tell them apart.

They are **118 runs making 3 regions**:

    U+1CCD7..U+1CCE9   the outlined letters
    U+1D401..U+1D69D   the mathematical alphanumerics
    U+1F111..U+1F190   the enclosed ones

Merged at a gap of 256, and the constant is not choosing the answer: the widest
gap inside a region is 67 code points and the narrowest between two is 1816, so
anything from 68 to 1816 gives the same three. Both gaps are printed, so a
build that moves them says so rather than quietly re-drawing the regions. Runs
and regions are both recorded and asserted on the matching Unicode version.

**Break-tests — 2 run, 2 fired.** The population scattered rather than
clustered. The regions merged by too wide a gap.

---

## Item 6 · the paste, read back

`affordedRetakeParts` writes the records in source shape so a re-take is a
paste. Nothing read it, and the failure it predicted had **already happened**:
`sound` was added to the band record last hour and the printer did not write
it. The new test caught it on its first run.

The paste is not a record of anything, so there is nothing to hold it still.
What there is instead is the declaration, read out of this file's own source —
the same move the two-folds test makes about browser.mjs. Every key the printer
writes has to be a field of the struct, and every field has to be one the
printer writes. Both directions, because the two failures are different: a
paste that will not compile, and a paste that quietly leaves a field at
whatever it was. Then the census paste's values are checked against this run's
walk, including the ORDER of the names, because the walk takes windows of
consecutive names and a re-ordered list is a different population.

**Break-tests — 4 run, 4 fired.** A field the record has and the printer does
not write. A field renamed under the printer. A paste that does not parse. The
names printed sorted rather than in walk order.

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

Two files, +1664 −195.

    wasm/verify/themenearmiss_test.go  items 1, 2, 3, 4, 6
    wasm/verify/inkglyph_test.go       item 5

`go test ./wasm/verify` is 2.8s against 6.5s, and `-short` is the same as a
full run because the lever it needed is gone. 19 break-tests run; every one
fired, and the one that did not fire the first time is the reason the mounting
arm exists.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

1. **(age 0 · value medium) Three leaves is now twelve seconds, which is a
   decision rather than a wall.** C(80,3) is 82160 populations and the
   corrected census takes them at about 150µs each. That is too much for every
   green run and it is no longer "minutes": a k=3 band would make the two-leaf
   arms' companion for three-field edits an assertion on the same footing as
   the rest, and it is the first time the family has been enumerable at that
   size. What has not been costed is the cheaper census underneath it: the
   corrections key their counts by the ending's own sentence, so every one is a
   map lookup on a long string, and an ending INDEX would take the per-population
   cost to something like 80µs — putting k=3 near six seconds, which is the
   range where a lever becomes worth having again.
2. **(age 0 · value low) The BMP's seed-bearing population has no shape
   either.** The astral half is now 118 runs in 3 regions, named and asserted;
   the BMP's 299 are counted and their runs are computed and thrown away. They
   are the population the printable-ASCII arm actually holds shut on the plane
   every fixture lives in, so if any census here deserves a shape it is that
   one. The clustering is written and the constant is measured; what is missing
   is the two recorded numbers and a sentence about which blocks they fall in.
3. **(age 0 · value low) The band paste's values are checked by nobody.** The
   paste test compares KEYS in both directions for both records and values only
   for the census one, because the bands it prints are the recorded ones and
   comparing those against themselves says nothing. The honest version measures
   the bands in that test and compares — which is two seconds for a printer —
   or has the failing run's own paste checked against the numbers that run
   measured, which is where the values actually matter and where nothing looks
   at them today.
4. **(age 0 · value low) `affordedWhole` is held against the direct walk over
   the record's names and never over this run's.** The base census it corrects
   from is compared with `affordedCensusOf` for the record's population and for
   the fourteen-name family; the whole this run's own one-leaf band builds is
   compared with nothing, even though the test has already walked those sets
   itself into `reached`. It is one map comparison and it would close the last
   population where a corrected census runs unheld.
5. **(age 0 · value low) The sound bound costs 557ms to confirm something that
   cannot change.** It is a theorem — the product of per-step maxima dominates
   — so the arm can only fire when one of the two walks is wrong, which is
   exactly what makes it worth running and also what makes it the most
   expensive tautology in the file. Whether a cheaper version exists has not
   been asked: the second step's band is a maximum over 6320 ordered drops, and
   most of them cannot be the maximum for reasons the one-leaf band already
   knows.
6. **(age 0 · value low) The float census still cannot see a block scope, and
   now has four tables instead of one.** `returns`, `method`, `from` and
   `field` are four name-keyed maps, three of which conflate anything sharing a
   name. Nothing collides today and nothing checks that nothing collides — a
   `changed()` returning a float on one type and an int on another would make
   every `x.changed()` read as a float, quietly. The arm that would catch it is
   cheap: a collision census over the four tables, reported rather than
   resolved.
