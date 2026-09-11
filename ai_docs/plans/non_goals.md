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

---

## A t.Run inside a loop is counted as one question

*Raised: 2026-09-10 · Moved here: 2026-09-10 · Code:
`wasm/verify/repowalks_test.go`, `subtestSitesIn`*

**What was declined.** `subtestSitesIn` counts the subtest SITES a walk's body
opens, and `repositoryWalks` holds each row's `asks` list to that number. A
site inside a `for` is one site and as many subtests as the loop is long, so a
table-driven walk test would open one site and ask several questions, and its
row would be held to saying one.

**The argument.** The loop's length is a run-time fact and no parse has it.
This is the same limit `priceCalls` already writes up one pass over for a WALK
inside a loop, and it is resolved there in the only way it can be: a bound
written in the source — `for range 2`, `for i := 0; i < 3; i++` — is read off
the source and multiplied in, and a `range` over something whose length is
decided at run time is REPORTED rather than guessed at.

Doing the same here would mean building the loop-bound reader a second time
for a shape that does not exist: there is no `t.Run` inside a loop anywhere in
this package, and the walks it would apply to are seven functions that each
open between zero and five subtests by hand. The cost is not the code, which
is already written next door; it is a second caller of `loopBound` and
`packageLevelInts` threaded into a census that currently reads nothing but
declarations, to price a shape nobody has written.

**What would change this.** A table-driven repository walk appearing — a walk
test whose questions come out of a slice of cases rather than being spelled
out. At that point the row would be saying one where the body asks several,
which is the count being wrong in the direction that hides work, and the
machinery to fix it is `priceCalls`'s and already exists. Until then the limit
is stated in `subtestSitesIn`'s own doc, which is where somebody writing that
loop will be reading.

---

## `packageLevelCallsTo` does not distinguish an initializer from a stored function value

*Raised: 2026-09-10 · Moved here: 2026-09-10 · Code:
`wasm/verify/repowalks_test.go`, `packageLevelCallsTo`*

**What was declined.** A walk called from a package-level declaration is
reported rather than counted, because a `var` initializer runs when the test
binary starts and nothing attributes it to a caller. `var x = someWalk(root)`
runs at init; `var f = func() { someWalk() }` does not run until something
calls `f`. The census reports both identically and says so.

**The argument.** Telling them apart is data flow. It means knowing whether
the call is evaluated when the declaration is, which for anything but the two
literal cases above needs to follow values through assignments, struct
literals and function returns — and the walks in this package decline type
information on purpose, because every one of them is a syntax census that
stays cheap by not resolving anything.

The finding is also already correct without it. Both shapes are a walk the
budgets are not counting, both need the same fix — move the call into the
function that needs the result and pass it in — and the message says that the
number beside the row is not one this pass computed, which is the honest
statement either way. What the distinction would buy is a more precise
sentence about a case that has never occurred: no package-level declaration in
this directory calls anything either caller asks about.

**What would change this.** The case occurring, and the two kinds needing
different advice. If a package-level `var` ever holds a function value that
walks the repository, and somebody is told to move a call that never ran, the
message is wrong in a way a reader can act on badly — and that is the moment
to decide whether the distinction is worth type information, not before.
