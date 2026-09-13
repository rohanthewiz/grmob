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

## The sibling file

`need_hardware.md` is this file's opposite number: items that are **not**
declined, are designed and written, and cannot be checked without a physical
device. They carry in Next lists forever for a reason that is not indecision,
and they were being read as deferred work. An item belongs there rather than
here when running it would settle something; here when running it would not.

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
`twoCopyValueShapes` (named `twoCopyStateShapes` when that entry was
written), and the shared parse now fails if they diverge.

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

---

## A sentence naming a record field is not held to quoting that field's range

*Raised: 2026-09-11 · Moved here: 2026-09-11 · Code: the two
`…TimingsTakenOn` records*

**What was proposed.** The defect class this repository kept fixing by hand
for several sessions was a sentence quoting a record's numbers while the
record said something else — `main.go` carried `30.14–30.42s against
399–401ms` attributed to a record that had moved. The mechanical version: a
sentence that names `record.field` AND quotes a range must quote the range
that field carries.

**Measured at the comment group, it reads nothing.** 5 groups name a field
and quote a range; 25 figures in them, 0 matching the named field. That is
not 25 defects — it is the check being meaningless at that granularity. A
comment group here is routinely forty lines and discusses a figure's history,
the sibling package, and three takings that are deliberately different
numbers. Proximity within a group says nothing, which is exactly the reading
that killed the wall-clock rule one entry above.

**Measured at the sentence, the corpus is one.** Over every Go comment
outside `ai_docs`: **one sentence** in the repository names a record field and
quotes a range in the same sentence, and it is a history sentence with the
field backquoted under the quoting convention. Zero defects.

**And the reason is the useful part.** The corpus is empty by construction.
This repository's own doctrine, arrived at over several sessions and now
stated in both records, is *name the field, do not restate the number*. A
rule to catch "named the field and restated the number" has nothing to catch
because the doctrine already won. The measurement is worth more than the rule
would be: it says the doctrine is complete in the Go prose rather than merely
believed.

**What would change this.** A sentence appearing that does restate a field's
range. The scan is forty lines of Python and can be re-run; at the sentence
granularity it is cheap and it is the granularity that means anything.

---

## A `GRMOB_` name in prose is not held to being an environment variable

*Raised: 2026-09-11 · Moved here: 2026-09-11 · Code: the four env vars Go
reads, and `android/app/src/main/java/com/grmob/runtime/GrMobProgress.kt`,
which is why this does not work*

**What was proposed.** `GRMOB_[A-Z0-9_]+` looked like the narrow shape that
made the test-name rule work: one prefix this repository owns, screaming
case, nothing in English like it. The rule would be that a `GRMOB_` name
written in prose — a comment, a doc page, a shell script, a `.mjs` — is a
variable something here actually reads.

**The shape has two meanings, which is the test it fails.** Eleven distinct
names match it:

    7 environment variables   GRMOB_BAND_VERDICT, GRMOB_COMPOSE_SOURCES,
                              GRMOB_IMPORTER, GRMOB_PER_OBJECT_FETCH read by
                              Go; GRMOB_TRANSCRIPT and GRMOB_CHROME by the
                              browser pass's JavaScript; GRMOB_AUDIO_OUT by
                              a Swift UI test
    4 Kotlin constants        GRMOB_PROGRESS_DETERMINATE, …_INDETERMINATE,
                              …_UNSTATED and …_EMPTY_RANGE are `const val`
                              declarations holding "determinate",
                              "indeterminate", "unstated", "empty-range"

Four of eleven — 36% of the corpus — are not environment variables. That is
the same failure as the backquoted-name rule two entries up, and worse: a
backquoted identifier is ambiguous across languages, and this one is
ambiguous inside a single directory of one language.

**And there is nothing to find.** 91 mentions across the repository, and
every one resolves to a name something reads or declares, in whichever of
the five languages owns it. **Zero stale.** A first scan said 32 were
unresolved; all 32 were the scan looking only at Go string literals in a
polyglot repository, which is a fault in the measurement and not a finding.

**What would change this.** A prefix that means one thing — if the Kotlin
constants were renamed, the shape would carry one meaning and the rule
becomes writable. It would then have zero findings on today's corpus, which
is the second reason not to write it. The scan is thirty lines of Python and
can be re-run.

---

## A command in a "Re-taking it" table is not held to being runnable

*Raised: 2026-09-11 · Moved here: 2026-09-11 · Code: the two
`…TimingsTakenOn` records' doc comments*

**What the defect was.** Both records' re-taking tables carried commands that
did not run. `internal/themehistory` wrote `./internal/…` on three of four
lines, and `wasm/verify` wrote its three walk depths as fragments — no
`go test`, no package path, and two of the three test names cut off at an
ellipsis. All shortened to keep a column aligned, in a repository with no
line-width census and 195-character comment lines elsewhere. All ten
commands are now written out and every one was run.

That is the second time this defect class has been found here; a session in
September found "a command nobody ran" for the same reason.

**Why the rule that should have caught it could not.** The prose-names rule
holds every `Test`-shaped name in prose to being a test this repository has,
and it reads these tables. It passed on
`TestEveryGitListingInAScriptAsksForNul…` — because the regexp stops at the
ellipsis, leaving `…AsksForNul`, which is a PREFIX of the real test, and a
prefix resolves **by design**: `go test -run X` runs everything X begins. The
rule's own header argues for that and is right to. It means an elided command
is invisible to it.

**The narrow shape, and why it is not written.** A tab-indented comment line
containing `go test` and a typographic `…` is one shape with one meaning —
`./...` is three ASCII dots and Go's own wildcard, so there is no collision.
Measured: **2 lines, both real defects.** Both are now fixed, so the rule
would find nothing, and it catches only one of the two ways these commands
were broken — the other three lines were fragments with no ellipsis in them
at all. A rule that scores zero on today's corpus and covers half the fault
is the shape this file exists for.

**What would change this.** A third instance. Two is a coincidence a person
fixed; three is a habit, and the argument for a five-line check is then
already written above.

---

## A declaration a census reads is not held to living outside a build tag

*Raised: 2026-09-11 · Moved here: 2026-09-11 · Code:
`wasm/verify/wholefileband_test.go`'s `//go:build !race`*

**What the defect was, twice.** `wasm/verify/wholefileband_test.go` is
`//go:build !race`, because the TestMain clock in it measures a figure that a
race build is not a reading of. Two iterations running, something a census
needed was declared inside it and the package stopped compiling under `-race`:
first `bandVerdictEnv`, read by the reporting arm, and then `recordedBand` and
`recordedBandForm`, read by `checkEveryBandFieldHasATakingCommand`. Both were
moved into `timings_test.go`, which carries no tag.

The fault is easy to make and invisible while writing: the tag is at the top of
a 270-line file, the declarations are ordinary Go, and `go test ./...` — the
command a session runs first — is green either way.

**Why no rule is written.** The Go toolchain is the arm, and it is already in
the verification path: `go test -race ./...` reports `undefined: recordedBand`
with the file and line of every reader, which is a better finding than any
parse of build tags would produce. It caught both instances, in the session
that introduced each. A check over build constraints would be a second,
weaker implementation of something the compiler does exactly.

What made the first instance cost anything was running `-race` late. That is an
ordering habit, not a missing rule, and the habit this repository already has —
eleven verification paths, run together — is what fixed it both times.

**What would change this.** A third instance found *after* a commit, rather
than by the race suite in the same session. That would be evidence the
verification path is not running where it needs to, and the answer then is
about when `-race` runs rather than about a new census.

---

## A band field does not carry a mark saying which of its ends have been reached

*Raised: 2026-09-11 · Moved here: 2026-09-11 · Code:
the two `…TimingsTakenOn` records, and `bandPlacement`*

**What was asked for.** "Record which band ends have actually been reached."
It follows from the rule this repository paid five re-takings for — an end is a
reading, not a choice — and it names a real gap: an end no reading has ever
landed on is indistinguishable, in the source, from one twenty runs have
landed on. Both are two decimals in a struct literal.

**What was built instead.** The verdict line says it, per reading:
`bandPlacement` reports where in the band the reading fell and whether it
reaches an end at the precision the end is written to. So the evidence arrives
beside the figure, in the line a person reads when they take figures at the end
of a session.

**Why the mark itself is not.** It would be a hand-kept claim with nothing able
to check it. Every other claim in these records is held by something: the
machine fields by the reporting arm, the cores note by two censuses, the taking
table by `checkEveryBandFieldHasATakingCommand`. A "floor reached" mark has no
such backstop, because the evidence for it is a wall clock — the one thing in
this repository that cannot be asserted. A mark a session writes by hand, with
no arm over it, is exactly the state the taking table was in for six iterations
and the cores note was in before it had a census.

The alternative — the test writing the mark back into the source — is declined
under its own entry above: a record a test can rewrite is a record that
re-baselines an accident.

**What would change this.** A mark that something can check. If a future
session finds a way for a reading to be recorded by the run that took it,
without that run being able to edit the claim it is evidence for, the gap this
describes is worth closing and the argument above stops holding.

---

## A width stated in prose is not held to quoting the range it comes from

*Raised: 2026-09-11 · Moved here: 2026-09-11 · Code: the two
`…TimingsTakenOn` records' comments*

**What the defect was.** Five sentences in these two records state a band's
width as a number. **Three of the five were wrong**: `3% wide` of a band that
is 6.8%, `300ms wide` of one that is 360ms, and `130ms wide against 130ms` for
a comparison that has since reversed — this band is now the wider of the two.
Every one of the three is derivable from two numbers in the same file. All
three are fixed: two now read in the past tense as the instrument that was
used, and the third is struck.

**The form separates them exactly.** The two correct claims both quote the
range they are computed from in the same sentence (`2.49–2.59s … roughly 4%
wide`, `1.40–1.67s is a band 19% wide`). None of the three wrong ones names a
range or a field — they say "that package's whole-package figure" and "the two".
So a form rule would have scored **3 of 3 on the defects and 0 of 2 false
positives**, which is a better separation than any rule this file has declined.

**Why it is not written anyway.** Two reasons, and the second is the one that
decides.

The residue is zero: the three are fixed and the remaining instances are either
self-contained or quotations. That alone would only postpone it.

What decides is that the rule's own explanation breaks it. The sentence that
strikes the `130ms` claim quotes the claim, and `bandPlacement`'s doc comment
lists all three as the reason the width is printed at all. Both are prose about
a width with no range beside it, and both are correct. Three of the six
questions on the shared parse already carry a self-exemption, and that file's
header says the fourth is the point at which the exemption should become a
convention rather than a list. This would be the fourth — and the convention it
needs (a quoted span is not a claim) already exists for test names in
`quotedprose_test.go`, so the honest cost is extending that convention to
double-quoted prose and teaching a census to respect it, for a rule whose
corpus is five sentences.

**What would change this.** A fourth wrong width, or the same class appearing
in a record that is not one of these two. Either makes the corpus big enough
that extending the quoting convention is paid for by more than one rule.

---

## A constant whose doc names another constant is not held to being defined as it

*Raised: 2026-09-11 · Moved here: 2026-09-11 · Code: the integer constants in
`wasm/verify` and `internal/themehistory`*

**What the defect was.** `twoCopyPackages` was `2`, and its own doc comment said
"it is the same constant as `timingsRecordCopies`" — a sentence claiming two
numbers are one number, beside a second number. `coresNoteScanDirs` had got this
right two files over and written the argument down: "a bare 2 here would be a
second number to keep in step by hand." Fixed: `twoCopyPackages` is now defined
as `timingsRecordCopies`.

**Why the rule is not written.** Measured: **27 integer constants across the two
packages, 10 of them literals whose doc comment names another constant, and 1 of
those 10 was a copy.** The other nine name a constant to point at a *pattern* —
`gitWrapperAcceptRules` says `timingsRecordCopies` "is the same shape of
constant for the same kind of reason", about acceptance rules in a git-wrapper
census, and it is `2` by coincidence — and eight of the nine have a different
value from the constant they name, so even a value comparison would not separate
them.

Telling "is the same constant as X" from "is the same shape of constant as X"
is reading English, which is the thing the cores-note census learned not to do:
"Every other way of deciding is this arm guessing at English." A rule that
cannot make that distinction either reports nine false findings or waits for a
convention nobody is following yet.

**What would change this.** A second real instance, or a convention that marks
the claim — a constant that says `= <other>` in its doc the way a taking table
says its command. One defect in ten candidates is not enough to ask anybody to
follow a new convention.

---

## The other measurement records do not grow the timings machinery

*Raised: 2026-09-10 · Moved here: 2026-09-11 · Code:
`affordedMeasuredOn` in `wasm/verify/themenearmiss_test.go`, `foldMeasuredOn`
in `wasm/verify/inkglyph_test.go`*

**What was proposed.** `…TimingsTakenOn` is not the only record in this
repository. `affordedMeasuredOn` and `foldMeasuredOn` are records too, and they
have none of what the timings records have: no five machine fields, no
reporting arm, no cores note, no band verdict, no taking command. Carried on
the Next list for fourteen sessions, value falling.

**What the measurement says.** Counted by parsing the four records' literals:

    affordedMeasuredOn            4 fields,  0 strings, 0 wall-clock bands
    foldMeasuredOn               15 fields,  0 strings, 0 wall-clock bands
    verifyTimingsTakenOn         11 fields,  8 strings, wall-clock bands in 6
    themehistoryTimingsTakenOn    9 fields,  5 strings, wall-clock bands in 4

**Not one wall clock between them.** Every piece of the timings machinery
exists for the one property a wall clock has and a count does not: it cannot be
re-derived. The machine fields say which computer, because the same code on
another machine gives another number. The band and its verdict exist because
the figure moves between sittings on one machine. The taking command exists
because nothing in the repository can produce the figure on demand.

A count is a reading of DATA, and both of these records are re-derived by the
run that reads them — `affordedMeasuredOn`'s four ending counts are re-walked
over its own eighty names and asserted exactly, and `foldMeasuredOn`'s are
recomputed against the Unicode tables. A number that moved there is a finding
about the data in the commit that moves it, which is strictly better than
anything attribution could offer.

And each already carries its own equivalent of the machine, for the same
reason and under its own name: `foldMeasuredOn.build` is a `foldBuild` naming
the Unicode, ICU and node versions its counts are a reading of, and
`affordedMeasuredOn` carries the eighty names themselves — the data is in the
record.

So there is nothing to extend. The two families answer different questions and
the shapes they have are the right shapes for each.

**What would change this.** A wall clock arriving in one of these records, or a
count in one of them that stops being re-derived. Either makes it the same kind
of record as the timings ones, and the machinery is then worth copying rather
than discussing.

---

## The iOS cold deep-link path is not driven end to end by a script

*Raised: 2026-09-11 · Moved here: 2026-09-11 · Code: `ios/GrMobUITests/`,
and `mobile/verify`'s manifest pins*

**What was declined.** `core.OnDeepLink` was verified on iOS by a throwaway
XCUITest that attached to the running app and tapped through a system prompt.
The obvious next step is the Android arrangement — one command, repeatable,
checked in: `xcrun simctl openurl booted grmob://lesson/4.12`.

**The argument is a platform's, not this repository's.** iOS puts "Open in
GrMobApp?" in front of a custom-scheme link from an unknown source, and that
prompt belongs to SpringBoard rather than to the app. A script cannot dismiss
it; only something driving the device can, which means either Safari automation
or XCUITest — and an XCUITest that exists to tap one system alert is a test
whose whole subject is the alert.

The alternative the platform actually intends is a **Universal Link**, which
raises no prompt because the association is verified. That needs a domain
serving `apple-app-site-association` over HTTPS at a path Apple fetches, which
is an operational commitment rather than a piece of code, and this repository
has no domain.

**What is checked instead.** `mobile/verify` pins the manifest half on both
platforms — `CFBundleURLTypes` and the intent filter — which is the part
nothing else can check, since the OS reads those at install time with no
compiler holding an opinion. A missing scheme is not a crash: the OS simply
never offers the app the link. And the Android side IS driven end to end, cold
and warm, by one `adb` command, so the *Go* half of `core.OnDeepLink` has a
repeatable runner; what iOS adds on top of it is one shell's `.onOpenURL`.

**What would change this.** A domain. With a verified Universal Link there is
no prompt, `xcrun simctl openurl` reaches the app directly, and the script is
worth writing that afternoon. Failing that, a second iOS deep-link defect —
the first one would be evidence that the shell's half needs a runner of its
own, and the XCUITest that taps the alert is then paying for itself.

---

## `comps.MapPanel` is not rearranged until it has a second consumer

*Raised: 2026-09-09 · Moved here: 2026-09-11 · Code: `comps/map_panel.go`*

**What was declined.** `MapPanel` was extracted from one screen and still has
exactly one consumer. The standing question is whether its arrangement — which
props it takes, what it decides for the caller — is right, and the obvious move
is to keep revisiting it.

**The argument.** The extraction was made with the condition written down: a
component with one consumer is a component whose shape is a guess, and the
second consumer is what turns the guess into a measurement — it is the one that
either fits or says exactly which prop is wrong. Rearranging it before then is
designing against an imagined caller, and the cost of being wrong is paid twice
(once now, once when the real second caller arrives and disagrees).

The part that WAS worth improving has been: `FitRegion`, the arithmetic
underneath, which crosses the antimeridian now and is held to it by tests. That
is a fact about geometry rather than about an arrangement, and it is right or
wrong independently of how many screens call it.

**What would change this.** A second consumer. Not a hypothetical one — a
screen somebody is actually writing. Until then the entry exists so that
"nothing has happened to MapPanel" reads as a decision rather than as neglect.

---

## `prefs` and `session` stay two bytdb files

*Raised: 2026-09-08 · Moved here: 2026-09-11 · Code:
`internal/prefs/prefs.go`'s package comment, in `church_mobile`*

**What was declined.** Two bytdb files means two locks and two write-ahead
logs for what is, from a distance, one application's local state. Merging them
into one store with two tables is the obvious simplification.

**The argument is already written where it belongs.** `prefs.go`'s package comment opens with
"Why a second bytdb file and not a second table in the session store", and the
reason is a lifetime difference rather than a schema one: a session is
disposable and is cleared on sign-out, preferences outlive every session and
must survive exactly that clearing. One file with two tables makes "throw the
session away" a selective delete that somebody has to get right, where two
files make it a file the app removes.

**What would change this.** The cost being paid for rather than merely counted:
a measurement showing the second lock or the second WAL costing something on a
real device. Two locks nothing contends for and two WALs nothing flushes under
load are a cost on paper. This entry exists because the simplification looks
obvious from outside the file and the answer is inside it.

---

## The OpenStreetMap tile sources stay as they are

*Raised: 2026-09-09 · Moved here: 2026-09-11 · Code:
`android/.../runtime/GrMobMapView.kt`, `wasm/grmob-runtime.js`, and
`comps/map_panel.go`'s own doc*

**What was declined.** Both the Android host (osmdroid) and the browser host
(Leaflet) draw from `tile.openstreetmap.org`, which is the OpenStreetMap
project's own infrastructure, donated and rate-limited. Pointing them at a paid
provider is the responsible-looking move.

**The argument is about who is running this.** The usage policy is written into
every host that draws those tiles — three files, each beside the URL — along
with the User-Agent the policy requires and the instruction to change the
source before shipping to a real user base. What this repository is is a
framework and a tutorial: a demo app, an emulator, and whoever is reading the
lesson. That traffic is what the OSM policy contemplates, and it was confirmed
serving normally as recently as this session.

Choosing a provider *for* a downstream app would also be choosing its billing
relationship, which is not a framework's decision to make. The keyless default
is what makes the map node work on arrival, which is the same argument
`LocationSensor.kt` makes for LocationManager over the fused provider and
`GrMobMapView.kt` makes for osmdroid over Google Maps.

**What would change this.** A real user base on a build that ships from here,
or OSM's policy changing. The note beside each URL is where somebody in that
position will read it, which is the condition this file asks entries to state.

---

## `android/device` has no instrumented-test runner

*Raised: 2026-09-11 · Moved here: 2026-09-11 · Code: `android/device/README.md`*

**What was declined.** `ui.sh` and `paint.py` drive a real device and found
three bugs in one week that every automated check in this repository passes
either way. iOS's equivalent is a real XCUITest, wired to a scheme and runnable
by one command. Android's would be an `androidTest` source set and a
connected-check task, and the two scripts are its content.

**The argument, which the README already makes.** These are run by a person on
purpose. A harness wired to a runner that cannot run — no device attached, no
emulator booted, a CI machine with neither — is a red build nobody can fix, and
a red build nobody can fix is one every reader learns to ignore. The iOS side
went the other way only because XCUITest is a test framework with a simulator
under it: the runner already exists and already knows how to skip.

The asymmetry is therefore the platforms' rather than a decision that half the
work was done. Adding `androidTest` is a build-system change — a second source
set, a second dependency tree, a `connectedDebugAndroidTest` task — for scripts
that are a shell function and a Python file.

**What would change this.** Something needing Android's equivalent of
`LiveMapUITests`: an assertion about a running Android app that has to hold on
every commit rather than when somebody looks. That is the day the build-system
change pays for itself, and these two files are what it starts from.

---

## `GrMobMinContent` floors every leaf that is not text at zero

*Raised: 2026-09-11 · Moved here: 2026-09-11 · Code:
`ios/GrMob/Runtime/GrMobMinContent.swift`*

**What was declined.** A Button, an Input, an Image and a MapView each have a
min-content size in CSS, and this host gives all four a floor of zero. Only
text is measured. The obvious completion is one measurement per leaf kind.

**The argument.** Each of those four would need a *different* measurement —
none of them is a function of a string, so there is no shared routine to
extend, only four new ones. And the under-estimate is safe by construction: a
floor that is too low leaves a child exactly as crushable as it was before the
floor existed, which is the behaviour every one of these nodes already had and
which nothing has complained about. Text is where the divergence was found and
where it bites, being the only leaf whose whole business is to be narrower than
it wants to be — a run of words is the one thing a flex line can legitimately
squeeze, right up until it cannot.

**What would change this.** A screen where a button, a field or an image is
visibly crushed on iOS and is not on the web. That is a reproduction, and the
measurement it needs is the one leaf kind it names rather than all four.

---

## There is no min-height half of the min-content floor

*Raised: 2026-09-11 · Moved here: 2026-09-11 · Code:
`ios/GrMob/Runtime/GrMobMinContent.swift`*

**What was declined.** CSS's automatic minimum applies on both axes. This host
implements the main-axis width case and leaves Columns floorless.

**The argument.** A text's min-content *height* is a function of the width it
wraps at, and a tree walk does not know that width — it is the output of the
layout the floor is an input to. Closing it honestly means a second pass, and
the thing it would buy is a Column that overflows its height rather than
compressing its children, which is a larger behavioural change than the defect
that prompted the width floor. Columns keep the behaviour they have always had.

**What would change this.** A vertical analogue of the row that started it: a
Column whose children are compressed below their content on iOS and are not on
the web, with a reproduction. The design question — overflow or compress — has
to be answered before the code is written.

---

## A declared main size floors at 0 rather than at `min(declared, min-content)`

*Raised: 2026-09-11 · Moved here: 2026-09-11 · Code:
`ios/GrMob/Runtime/GrMobMinContent.swift`, `internal/pinfixture`*

**What was declined.** CSS takes the *smaller* of the declared and the content
suggestions. This host, for a node that declares a width, takes 0.

**The argument.** A declaration becomes a `.frame` on this target, and the host
cannot see the content behind a frame — so the content suggestion is not
available to compare against. Zero is the safe end of the two possible errors:
right for the empty sized boxes `internal/pinfixture` mounts and an
under-estimate for a sized Text, where an over-estimate would overflow a line a
browser fits. Closing it means measuring a node's content *before* its own size
is applied, which is a second walk asking a different question of the same tree.

**What would change this.** A sized Text crushed below its content on iOS and
not on the web — which is the case the under-estimate is wrong for, and the only
one.

---

## `hooks.UseHeadingWhen` does not exist

*Raised: 2026-09-11 · Moved here: 2026-09-11 · Code: `hooks/location.go`
(`UseLocationWhen`), `hooks/heading.go`*

**What was declined.** `UseLocationWhen` is the gate for a screen that cannot
be its own route: one `core.NewState`, every pass, whatever the answer, where
the `if` a caller reaches for would move every hook slot after it. The compass
has the same shape and the same problem one file over, and the symmetry is one
function.

**The argument.** Nothing has asked for it, and the two sensors are not
symmetrical in the thing that matters — cost. Location got the flag because a
GPS left running is a battery bill and a privacy indicator; the compass costs
almost nothing to leave on, which is precisely why nobody has wanted to switch
it off. A hook added for symmetry rather than for a caller is API surface with
no consumer to keep it honest, and `UseLocationWhen`'s own test drives a
rendered tree to prove a slot below the call site is undisturbed — the version
for the compass would be that test copied with a different sensor in it.

**What would change this.** A screen that wants the compass released while
staying mounted. The day one exists this is a short function, and
`UseLocationWhen` is the whole template.

---

## `church_mobile`'s static maps stay unconfigured here

*Raised: 2026-09-11 · Moved here: 2026-09-11 · Code:
`comps/static_map.go` (`GoogleStaticMap`, `ConcernNoMapProvider`),
consumer in `../church/church_mobile`*

**What was declined.** `comps.StaticMap` draws an empty box in
`church_mobile`'s event detail screen, because no `Provider` is configured. The
obvious completion is to configure one — the widget takes a single function,
`GoogleStaticMap(key)` is written and tested, and the app is one line from
drawing maps.

**The argument.** The missing piece is not code and is not this repository's to
supply. It is a Google Maps Static API key, which is a billing relationship
belonging to whoever ships the app; `GoogleStaticMap` is a constructor
specifically so the key stays configuration rather than a package variable this
tree could hold. And `church_mobile` is downstream — a separate repository —
so even the call site is not here.

Everything this side owes is done and was re-confirmed twice. The widget
renders an unconfigured provider as a box with no image rather than as Google's
"not authorized" error tile, so a misconfigured build looks unfinished instead
of broken; `ConcernNoMapProvider` names it in debug mode; and the yaml, the
provider name and the instruction to restrict the key to the Maps Static API
and the app's bundle ids are written down for whoever holds the key. The live
events map needs none of it and works today, reachable from the same screen.

Carried as a Next item across four sessions, where it was re-read and
re-declined each time for the same reason. That is the shape this file exists
for: it is not a thing to do, it is a thing that is waiting on somebody, and a
waiting item in a work list is work done repeatedly.

**What would change this.** A key. The change is one argument to one
constructor in a repository that is not this one, and nothing here has to move
for it.

---

## Windowing `core.List` over the bridge

*Raised: 2026-09-10 · Moved here: 2026-09-11 · Code:
`examples/tutorial/app_test.go`, `TestWhatWindowingWouldSave` (the profile);
`core/host_events.go` (the channel it would have used)*

**What was declined.** Teaching `core.List` to send only the children near the
viewport: the host reports a visible range, the app renders a window of
children plus placeholders for the rest, and the payload for a long list stops
being paid all at once on the first frame.

**The shape, which is fully worked out.** It needs no new bridge surface —
`core.OnHostEvent` / `mobile.ReportHostEvent` already carries a name and a JSON
payload, already returns the following pass's patches on the event path, and is
already serialized with render passes by the manager. A visible range is one
more event name. What it does need is a bootstrap guess (a cold launch has no
visible range, because the host cannot lay out what it has not received) and
**placeholder children** rather than absent ones — a LazyColumn sent three
children believes there are three, so the scroll extent is wrong and the scroll
stops short until more arrive. Placeholders keep the extent right at ~40 bytes
each.

**The argument.** It was sized twice, and the second sizing is why it is here.

```
                          payload    a window of 4-5 children removes
before the chapter cards
  learned to collapse      51,242    33,800 - 38,900 bytes   (66-76%)
today                      17,366     7,254 -  8,681 bytes   (42-50%)
```

The collapse and the window were aimed at the same 45 lesson rows, and the
collapse got there first — with no protocol change at all. What is left is
about 8KB, which on the one emulator anybody has measured is **54-64ms** of
parse-and-build: still seven times that instrument's ~8ms noise floor, so the
effect is real and would be visible. It is 1.7% of a 3,200ms launch, where the
first sizing made it 7.7%.

And that emulator is the machine most favourable to the argument. Its
`org.json` spends ~200ms on 51KB where iOS's parser spends 6ms on the same
tree, so a byte is worth more there than anywhere else this runtime ships. A
physical phone would make the prize smaller, not larger.

That last sentence is the only part of this that is still owed a measurement,
and it is owed one whichever way this decision goes: every startup figure this
project quotes is one emulator's. It is tracked as an instrument problem in
`ai_docs/plans/need_hardware.md`, under "`launch.sh`'s numbers are one
emulator's", which says in its own words that it no longer gates this item. So
a reader who arrives here wondering whether the device number would reopen the
question has an answer and a place to watch: it would have to make the prize
*larger*, and the direction of the only evidence anybody has is the other way.

So: a protocol change, a bootstrap guess, and placeholder children in four
renderers, for 1.7% of one host's launch on one screen.

**What would have to change.** A long list. Forty lessons in one open chapter
puts forty rows back on the screen, and any app built on this framework with a
genuinely long `core.List` is in the same position — windowing was always a
framework answer rather than a tutorial one, and what changed is only that the
tutorial stopped being the screen that argued for it. The profile to re-take is
`TestWhatWindowingWouldSave`, which asserts its own share now: it printed a
table nothing checked, and went on quoting 66-76% for a session after the
collapse made that false.

---

## A block-level patch for the rich-text editor's DOM

*Raised: 2026-09-11 · Moved here: 2026-09-11 · Code:
`wasm/grmob-runtime.js`, `richTextToDOM`*

**What was declined.** Rewriting only the blocks an edit touched, instead of
recreating the whole document on every write. The sketch is a group-level diff
where a group is a maximal run of blocks sharing a container, with a full
rebuild whenever the group shape changes.

**The cost it would remove, measured.** Every write comes through one function
— a pending mark, a paste, undo/redo, any toolbar command, any rewrite arriving
from Go — and each recreates the document entire:

```
blocks x runs      elements destroyed and recreated
   1 x 1                          2
 100 x 6                      1,000
2000 x 6                     20,000
```

A patch would put ~10 in every row. The ratio is real and it is why this entry
exists rather than a shrug.

**The argument, which is about correctness and not effort.**

*A block is not an element.* Consecutive list blocks share one `<ul>`, so
"replace block N's element" is not a well-defined operation. The replaceable
unit is a maximal run of blocks sharing a container, and computing that run
correctly on every edit is the whole of the work.

*And the one that decided it:* leaving an element in place is only safe if the
element still describes its block, and between two calls the **browser** has
been editing this DOM. Typing under `contenteditable` splits text nodes and
inserts elements of its own. A full rebuild normalises all of that away every
time; a partial one would normalise the blocks it rewrote and leave the rest in
whatever shape the browser last left them. That is a drift between the model
and the screen, in an editor — and nothing in this repository can test for it.
The harness DOM has no Selection API and no `contenteditable` behaviour at all,
which is the same limit that made the whole command vocabulary a pure document
transformation in the first place.

So the trade is a rebuild proportional to the document, on an action the user
initiated, against a normalisation hole only a browser can show.

**What was taken instead.** The half that carries none of the risk: the tree is
built into a `DocumentFragment` and attached with one `appendChild`. Every
element is still new, so the normalisation argument above is untouched word for
word; what changed is that insertions into the *live* `contenteditable` subtree
went from n+1 to one, so the browser's editing machinery and any
MutationObserver see a single transition rather than ~n states of a document
mid-edit. Pinned at 1, 100 and 2000 blocks in `wasm/verify/richtext_test.mjs`.

**What would have to change.** A profile from a real document that says the
rebuild is felt — and, before any of it ships, somewhere to run the
normalisation check. That means a browser, not this harness: the pass would
have to type into a `contenteditable`, apply a partial rewrite, and compare the
resulting DOM with what a full rebuild produces from the same document.
`wasm/verify` already drives a real browser for other claims; that is where it
would go.
