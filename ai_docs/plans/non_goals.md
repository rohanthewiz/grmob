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

---

## A backquoted name in prose is not held to being a declaration

*Raised: 2026-09-10 · Moved here: 2026-09-10 · Code:
`wasm/verify/prosenames_test.go`, which holds the narrow case that does work*

**What was declined.** The generalisation of the test-name rule. That one
holds every Go test named in a comment to being a test this repository has,
and it found four real defects the day it was written. The obvious next step
is the same rule for everything else this repository names in prose — helpers,
types, constants — recognised by the convention already used for them: a name
in backquotes.

**The argument, which is a measurement.** Over every Go comment outside
`ai_docs`, taking each backquoted span that is a single Go identifier and
excluding keywords and predeclared names:

    896   backquoted single-identifier mentions
    786   unresolved against package-level declarations
    334   unresolved, adding methods, struct fields and interface methods
    304   unresolved, adding every imported package name
    205   unresolved, adding every local variable and parameter in the tree

205 findings in 93 distinct names, against a rule whose whole value is that
its findings are few and real.

Narrowing to lowerCamelCase with at least one hump — which removes the
all-lowercase vocabulary in one cut — gets it to a size worth reading:

    69    mentions
    15    unresolved, 14 distinct names

**And none of the fifteen is a defect.** Every one was read:

    accessibilityIdentifier, accessibilityValue, simultaneousGesture,
    measureWithoutPlacing                        Swift and Compose API names
    getOrNull                                    Kotlin's standard library
    compileDebugKotlin, fetchComposeLayoutSources  Gradle task names
    flexShrink, shrinkFactor                     CSS
    measureText                                  a canvas API
    shrinkPinned                                 a native property name, read
                                                 out of generated source by
                                                 `strings.Contains`
    inkCanaryAgreement                           a JavaScript function in
                                                 browser.mjs, named from Go
    dottedVersionParsers                         a deliberate historical
                                                 mention: "this used to be…"
    enumWorkersPool                              a hypothetical, in a sentence
                                                 explaining a failure mode

**What the numbers say.** The backquote convention in this repository does not
mean "a Go declaration". It means "a literal token of some language" — and
this repository's subject is the agreement between four of them, plus CSS and
ARIA vocabularies and a build system's task names. A rule asking whether a
backquoted word is a Go declaration is asking a question the convention does
not answer, and its exemption table would be a list of other languages'
identifiers, maintained in Go, to keep a Go rule quiet. That shape is the tell.

**Why the test-name rule is not the same bet.** `Test` followed by an
upper-case letter is a shape only a Go test has. Nothing in Swift, Kotlin,
JavaScript, CSS or ARIA is spelled that way, and nothing in English is. A
lowerCamelCase identifier is a shape every language in this repository has,
which is exactly why the residue is what it is.

**What would change this.** A convention that separates the two — a distinct
marker for "a declaration in this module", as opposed to a token of whatever
language is under discussion. That is a repository-wide editing convention
rather than a check, it would have to be applied to 896 existing mentions
before any arm over it could run, and the thing it would buy is a rule that
scored zero on today's corpus. If such a convention ever arrives for its own
reasons, the arm is then cheap and this is worth re-reading.

It would also change if the residue started containing defects. The scan above
is fifty lines and can be re-run; what makes this a non-goal is the measured
ratio, not the idea.

**A later session gave backquotes a meaning, and it is the opposite one.**
`wasm/verify/quotedprose_test.go` states that a token in backquotes is quoted
rather than claimed — a name the sentence is ABOUT rather than pointing at.
That does not collide with the finding here; it is the same finding written
from the other side. A backquoted token is precisely the one this repository
declines to resolve, which is why a rule reading backquotes as "a declaration
in this module" scored zero, and why the marker that DID prove writable was
the one that means "do not resolve this".

---

## A wall clock in prose is not held to living in a record

*Raised: 2026-09-11 · Moved here: 2026-09-11 · Code: the two
`…TimingsTakenOn` records, and `wasm/verify/prosefigures_test.go`, which holds
the one figure in prose that CAN be checked*

**What was declined.** Two sessions established that a reading copied into
prose goes out of step with the record it names, silently, and found six such
copies by hand. The obvious arm is the mechanical version: a wall clock in a
comment must live inside a timings record's declaration.

**The discriminator, which does work.** A duration in a comment is one of two
things and they are spelled identically — `250ms` is a debounce this code
performs, and `250ms` is also something somebody timed. The records' own
doctrine separates them: a reading of a machine is a SPREAD, because the spread
is why recorded figures are ranges, and a specified duration is one number
because the code specifies one. Measured over every Go comment outside
`ai_docs`: **215 duration figures, 140 of them single values**, and every
single value sampled is a duration the code performs — a CSS transition, a
debounce, a poll period, a long-press threshold. The 75 ranges are the
readings. That part of the rule is sound.

**What killed it was the escape, in both settings.** A figure is plainly fine
when the paragraph it stands in is *about* a record, so the first form let a
comment group naming a record through. Tested against the tree as it stood
before the copies were fixed by hand, it found **none of them** — because every
one of those paragraphs named the record it was out of step with. main.go's
table said "Three runs on the machine themehistoryTimingsTakenOn names"
directly above numbers that record did not carry.

Removing the escape makes the rule fire, and on the current tree it finds six:

    main.go:919      0.18–0.30ms against a recorded 0.18–0.29ms   REAL
    main.go:661 ×2   "this table said 30.14–30.42s against
                     399–401ms" — a sentence about what was wrong
    timings_test.go  "0.87–0.92s measured, which is what the
                     0.83s this line used to quote"
    repowalks_test.go:49 ×2   "1.18–1.40s of a 2.88–2.97s package",
                     a sum of record fields, named in the record's own
                     re-taking list

One real in six, against the test-name rule's four in six. And the five are two
classes that will both recur: **a sentence quoting what a figure used to say**,
which every re-taking adds one of, and **a figure derived from record fields**,
which is the arrangement the previous session deliberately built. Neither
carries a marker distinguishing it from a copy, and an exemption table that
grows by one entry per re-taking is a table that documents the record's own
history in a checker.

**The one real defect was fixed on the way past** — main.go now names
`themehistoryTimingsTakenOn.batchRetire` instead of restating it, and the hand
sweep had missed it because it grepped for the record's exact strings and a
rounded copy is not one.

**The marker arrived, and it was not enough.** The session after this was
written introduced the convention the paragraph below used to ask for: a token
in backquotes is quoted rather than claimed, stated in
`wasm/verify/quotedprose_test.go`. It did for the test-name rule exactly what
was predicted — `renamedTestsStillNamed` is gone, and the two sentences it
existed for say what they mean in the prose. Re-running the measurement above
against the marked tree, the five sentences that were RIGHT are all resolved:

    main.go:661 ×2            backquoted. The sentence is unchanged in
                              substance; the marker alone did it
    timings_test.go           the live figure now names the record field, and
                              the two figures the line used to carry are
                              backquoted as what it used to say. The marker
                              did not remove this finding — naming the field
                              did — but it is why the history survived the fix
    repowalks_test.go:49 ×2   states the PROPORTION and names the fields. This
                              was the last figure on `verifyTimingsTakenOn`'s
                              re-taking list that a re-taking moved by hand

So the quotation class is closed and the rule is still declined, for a reason
that is sharper than the one it was declined on. **"Lives in a record" is not a
span anything can define.** Readings legitimately live in three shapes, and
only the first is a `…TimingsTakenOn` declaration:

    the record itself         76 range figures, and the rule protects these
    a table in the prose      the GOMAXPROCS table in
    beside it                 wasm/verify/timings_test.go, and the string
                              constant that restates it for a failure
                              message — 7 figures, every one a reading with
                              its method written beside it
    a `costs:` field of       repowalks_test.go's walk census carries
    another structure         "12.19–12.28s against 11.57–11.99s" with its own
                              taking method, in a structure that is a record
                              in everything but name

And one figure that is not a reading at all: `hooks/hooks_test.go:26` says test
producers run at `5–20ms` periods. That is the discriminator's own
counterexample — a SPECIFIED duration written as a range — measured at one
across the repository, which is small but is not zero, and the discriminator
was the part of this rule that was sound.

**What would change this now.** Not a marker. Either a way to say "this
paragraph is a record" that is narrower than a comment group naming one — the
escape that already failed, because every drifted sentence named the record it
had drifted from — or the second and third shapes above becoming records
proper, at which point "inside a declaration" is a span again and the rule is
one function. The residue to re-read by hand is ten figures and the scan is
sixty lines.

---

## The timings records do not carry their bands as durations

*Raised: 2026-09-11 · Moved here: 2026-09-11 · Code:
`wasm/verify/wholefileband_test.go` and `internal/themehistory/band_test.go`,
which hold the two copies of the parser this would have removed*

**What was proposed.** Both records write each figure as prose opening with a
range — `"2.65–2.78s over sixteen runs, the clock TestMain puts around
m.Run()"` — and two packages parse that prefix back out with the same
25-line function, duplicated. The proposal was to hold `lo` and `hi` as
`time.Duration` and render the prose from them: nothing to parse, no regexp,
no "the field must open with the range" fragility, and, it was claimed, no
copy.

**The copy does not go away, which is the whole of it.** These are two
separate `package main` programs and neither can import the other's tests.
That is why the parser is duplicated, and a renderer would be duplicated for
exactly the same reason and at about the same size. The duplicated surface
goes from roughly 30 lines to roughly 25. Nothing is solved that holding the
two copies identical does not solve better, which is what was done instead —
`recordedBand` and `recordedBandForm` are in `twoCopyFunctionShapes` and
`twoCopyStateShapes`, and the shared parse now fails if they diverge.

**And the rendering costs more than the parsing.** Measured over the ten
figure fields in the two records:

    3 fields    a bare range — "0.08–0.10s"
    5 fields    a range then a note
    2 fields    a range, a note, and MORE ranges inside the note:
                perObjectRun carries three further bands and a ratio range,
                wholePackage carries a second band for a single core

A renderer has to reproduce each field's chosen unit and precision exactly,
because the record writes `0.08–0.10s` and `0.18–0.29ms` and
`time.Duration.String()` renders those as `80ms` and `290µs`. So the struct
needs the unit and the decimal count beside the two durations — four pieces
of data to express what the string `"0.08–0.10s"` already says exactly, and
says more readably. The two fields with embedded ranges do not fit the shape
at all without a second mechanism for the ranges inside the note.

**What would change this.** A third package needing the same reader, which is
the point at which two copies become the shape kept in step by whoever
remembers to — the argument `twoCopyPackages` already makes for its own
value. At three, a real shared package earns its keep and the durations come
with it. At two, the census is the cheaper answer and it is in place.
