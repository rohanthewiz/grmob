# Non-goals

Things this repository has decided **not** to build, with the reason, so that
each one is visibly declined rather than quietly missing.

## Why this file exists

A session's Next list is a list of things to do. An item that will never be
done sits in it forever, re-read at the top of every session and re-declined at
the bottom of it — which is the same work done repeatedly and the same argument
made from memory. Moved here, it is decided once, in writing, and a later
reader who wonders why the obvious next step was never taken finds the answer
instead of the gap.

The bar for an entry is that somebody could reasonably propose it. A non-goal
nobody would suggest is not worth writing down; a non-goal that looks like the
natural next move is exactly what this file is for.

**Each entry states:** what was declined · where in the code the shape lives ·
the argument · and what would have to change for it to be reconsidered. The
last line matters: a non-goal argued from a cost is only a non-goal while the
cost holds.

## Housekeeping

Other non-goals are still carried inline in session docs and in code comments.
As they surface — a Next item marked "deliberate non-goal", a comment
explaining why something was not built — move them here and leave the code
comment in place. The comment is where somebody reading that function will look;
this file is where somebody planning work will look, and neither replaces the
other.

---

## A sound chain bound at k = 3

*Raised: 2026-09-06 · Moved here: 2026-09-10 · Code:
`wasm/verify/themenearmiss_test.go`, `affordedTwoStepBands`*

**What was declined.** `affordedTwoStepBands` produces a sound composed bound
for a chain of two drops — the widest one-leaf residual out of any population
of n−1, composed with the measured one-leaf band. The obvious extension is the
same construction at k = 3, matching `affordedBandSteps`, which runs to three.

**The argument.** Bounding the LAST step over every chain needs a census of
every population of size n−k. That census is the same family the direct
measurement walks, and the direct measurement is the cheaper of the two at
every k this file takes:

    k = 2   the sound bound censuses every population of n−2 (3160 of them),
            which is exactly what affordedKLeafBand(names, 2) already walks
    k = 3   the same identity holds, over C(80,3) = 82160 populations, and
            the direct measurement of the three-leaf band is still one walk
            of them rather than two

So the composition at k = 3 would cost more than the number it is a bound for,
and produce a looser one. It is not an approximation that buys speed; it is a
slower route to a weaker answer.

**Why k = 2 is kept anyway.** Not as a route to k = 3. It holds a SHAPE: it is
the reading that separated the two possible causes of `affordedChainBoundOf`
failing to cover — an unlucky representative population versus a product of the
wrong shape — and it answered the first. Having answered it, what remains is a
check that the two walks agree, which costs nothing because the census is taken
once and read three ways.

**What would change this.** A composition whose per-step bound could be taken
WITHOUT censusing the populations at that size — a closed form, or a bound read
off the ratio structure rather than off the members. Then the cost argument
inverts and the construction is worth having at every k. Nothing in this file
currently suggests one exists.

---

## The diff-and-print remainder stays unmeasured

*Raised: 2026-09-10 · Moved here: 2026-09-10 · Code:
`internal/themehistory/timings_test.go`,
`themehistoryTimingsTakenOn.perObjectRun`*

**What was declined.** `wholeRun` is 1.52–1.60s and three of its terms are now
measured in the same arm that produces them — the batched fetches at
398–405ms, the 88 `ls-tree` the walk pays serially at 0.87–0.92s, and
`themeleaves.Of` over the same sources at 0.118–0.120s, which is 1.39–1.44s
together. What is left is the diff between consecutive revisions and the
printing. The obvious next step is a fourth clock around those two, so that
the decomposition adds up with no remainder in it.

**The argument.** The remainder is 0.1–0.2s, and 0.1–0.2s is the size of the
disagreement between three separate readings of this machine. The parse was
worth a clock because it was the largest unmeasured term AND because the arm
already held every source in memory, so timing it cost nothing but the call;
this one is under the noise floor of the instrument being used to take it. A
clock on it would report a number that moves by its own magnitude between
runs, and a figure like that in a record whose whole subject is attribution is
worse than a stated remainder: it reads as measured.

The other half is that a remainder which says so is not a gap. `perObjectRun`
names the three terms, their sum and the total, and says what the difference
is and why it is not a reading. A person deciding whether this program is
worth optimising has everything they would get from the fourth clock except a
false precision.

**What would change this.** The remainder growing past the spread around it —
a diff that started doing real work per revision, or a table that grew enough
for the printing to matter. The condition is written down in the field's own
comment rather than an intention to get to it later, which is the difference
between a decision and a deferral: this is not an open question until the
number moves.

It would also change if the readings around it got tighter — a quieter
machine, or more runs — since what makes the term unmeasurable is the ratio
between it and the noise rather than its own size. Nothing currently suggests
either is worth arranging for it.
