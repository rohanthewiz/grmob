# Session: two scans of one parse, a finding counted twice, and a floor that was being chased

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-10 (follows "a-row-that-could-not-be-wrong-a-note-nothing-could-read-and-a-saving-that-was-a-thousandth")

## Ask

"Work the items in the Next list." Five items — two medium, three low — every
one raised by the previous session, three iterations into the same loop.

The previous session's shape was **what a written-down claim is worth**. This
one is about the loop itself producing the thing it removes:

    two scans      a census added one iteration ago walks the same trees a
                   census added two iterations ago walks, for the same
                   construct
    one fact       counted per record where it is a property of a package
    one floor      set from three readings, contradicted by the fourth, set
                   again, contradicted again

Nothing outside tests changed. `git diff` touches four test files.

| item | value | where | shape |
|---|---|---|---|
| 1 | medium | copies_test.go | a read the failing run pays twice for |
| 2 | medium | repowalks_test.go | two scans of one parse |
| 3 | low | repowalks_test.go | a figure with no machine behind it |
| 4 | low | copies_test.go | one fact, two findings |
| 5 | low | both records | a shape enforced from another package |

---

## Item 1 · a read the failing run pays twice for

`identifiersIn` took a directory and a term list, read the directory, read
every file that might hold a term, parsed it, and threw all of it away. The
cores-note check asks twice — once for each package's own terms, then over the
other packages for whatever the first pass could not find — so a repository
where a note is WRONG did the `os.ReadDir`, the `os.ReadFile` and the
`parser.ParseFile` a second time, over trees that had been in memory a
microsecond earlier.

It cost nothing while the second pass did not run, which is every green run,
and that is exactly the argument that stops being true at the moment something
breaks: **the run already failing is the one that pays twice**. A census whose
cost depends on whether it is about to report is one nobody measures under the
conditions it matters in.

`packageSource` caches the listing, the bytes and the trees, keyed by path,
each parse attempted once. The ANSWER is not cached — a second question about
a different term list re-inspects the trees it already has, which is not what
was expensive. Errors are reported once per file for the same reason: a file
that cannot be opened is one finding about that file, not one per asking.

One thing this turned up. Making `identifiersIn` a METHOD broke the `besides`
arm from last iteration, which finds a call the way every census here finds
one — a bare identifier, because `x.Fn(…)` is a method call and that is not
what an unqualified call resolves to. The row would have said nothing calls it
and been right. So it stays a function, and the header says why: that is the
watched code being shaped by the watcher, which is worth stating rather than
leaving as a puzzle for whoever tries to tidy it.

---

## Item 2 · two scans of one parse

`callsTo` and `declaredAndCalledBy` both walked every function in this
directory looking for bare calls to a named function. One priced the loops
around each site and one did not, and they were kept apart by an argument about
what the ANSWER means rather than about the walk.

The argument is real and it belongs to the pricing, not to the finding:

    callSitesOf    every bare call to a name, grouped by the function it is
                   in, and whether this directory declares it
    priceCalls     what those sites cost, multiplying in the loop bounds,
                   because the budget is made of that number

The walk census takes the sites and prices them. The `besides` pass takes the
same sites and reads off the names, because a read the budgets do not govern
needs to be known to happen and not to be priced — and deciding what a
`besides` number would mean when the read is in a loop is a question nothing
has yet had to ask.

Both behaviours re-checked after the split, because a refactor that changes an
answer is worse than the duplication it removed: the priced loop still prices
at ×3 naming its constant, the shadowed one is still refused, and the `besides`
row still reports a function renamed away.

---

## Item 3 · a figure with no machine behind it

`besides.through` got an arm last iteration. `besides.costs` did not, and the
NUMBER cannot have one — a wall clock is a reading of a machine, which is why
this repository keeps records instead of asserting timings.

What can be held is the attribution, and every other wall-clock figure in this
package's prose already does it: "where verifyTimingsTakenOn was taken", "on
verifyTimingsTakenOn's machine". That was a convention followed by nothing.
It is a check now — `costs` has to name the record — and the figure was edited
to say so.

So the field has an arm over both halves, of the two different kinds this
repository has: the function is held to existing, and the number is held to
being attributed. Neither holds the number, and nothing can.

---

## Item 4 · one fact, two findings

The arm and the note are declarations of a DIRECTORY — `arms` and `notes` are
keyed by one, and a second record in the same package does not need a second
arm — and they were being asked once per RECORD. A package with two records
reported each missing thing twice: two findings, one fact, one edit that
answers both, which is the wall this repository writes one-per-row rules
against everywhere else.

Nothing forbids a second record in one package below `timingsRecordCopies`, so
this is reachable rather than hypothetical. The questions are per directory
now, and the finding names every record in it, because what a reader wants is
which numbers are going unattributed and the answer is all of them.

Demonstrated with a second record in `wasm/verify` and the arm renamed away:
one finding, naming both records.

---

## Item 5 · a shape enforced from another package

`stringLiteralValue` decides what a note IS — literals joined by `+`, and
nothing else — and it lives beside the check rather than beside the note. The
two packages that write a note had nothing telling them the shape, and getting
it wrong is now a finding (the previous iteration's work) raised in another
package's test.

Both `coresAttribution` declarations carry it: literals and `+`, terms in
backquotes, and what happens if not. One comment each, where somebody about to
break it is looking.

---

## Verification

Eleven paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      rc 0
    android ./gradlew compileDebugKotlin --offline
    android ./gradlew :app:fetchComposeLayoutSources --offline
    GOOS=js GOARCH=wasm go build ./...

    wasm/verify/copies_test.go        items 1, 4
    wasm/verify/repowalks_test.go     items 2, 3
    wasm/verify/timings_test.go       item 5
    internal/themehistory/timings_test.go   item 5, and the widening below

**Two break-tests for the new arms, both fired**, and four re-run to prove the
two refactors changed nothing:

    the unattributed cost    the record's name taken out of `costs`: "that
                             figure does not name verifyTimingsTakenOn"
    two records, one arm     a second record in wasm/verify with the arm
                             renamed: ONE finding, naming both
    the priced loop          still ×3 with its constant named, after the
                             callSitesOf/priceCalls split
    the shadow               still refused, after the split
    the besides row          still reports its function renamed away, after
                             the split
    the unreadable note      still reported, after packageSource

**Figures.** `wasm/verify` **2.814–2.902s** over twelve, inside its
2.78–2.93s.

`internal/themehistory` is the interesting one, and it is a lesson about the
measuring rather than about the code. Its floor was set from three readings
last iteration, contradicted by the next run, set again, and contradicted
again — a floor being **chased** rather than measured. Three readings find a
range the fourth leaves.

So the plain row is sixteen readings now, **2.920–3.057s**, and the floor comes
from the set rather than from whichever run was last. Combined with every
earlier taking the row is **2.92–3.22s over fifty-seven runs in three
sessions**.

And every row that moved this time moved DOWN, by about what the taking an hour
earlier had moved it up:

    plain           2.920–3.057s    an hour before: 3.06–3.19s
    plain -short    1.181–1.189s    1.205–1.304s
    -race           6.466–6.609s    6.549–6.697s
    -race -short    2.391–2.436s    2.406–2.422s

The afternoon that read a tenth dear was not a machine that had got slower. It
was one end of this machine's own spread across a session — and an hour of that
spread is worth as much as every code change this package has ever seen. The
other three rows are three readings apiece and are therefore NARROWER THAN THE
TRUTH by roughly what the plain row gained, which is written into the record
rather than fixed: fixing it is thirty runs to learn what the plain row has
already said.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

1. **(age 0 · value medium) The `besides` arm quietly forbids a method, and
   says so only in a comment on the one function it watches.** `identifiersIn`
   had to stay a free function because `callSitesOf` finds bare identifiers,
   and the next person to write a `besides` row will point `through` at a
   method, get "nothing in this directory calls it", and have no way to know
   the finding is about the spelling rather than about the code. The message
   is the place to say it — one line naming the limit — or `callSitesOf`
   learns `x.Fn(…)` for a receiver whose type this package declares.
2. **(age 0 · value medium) `packageSource` caches trees for the life of the
   check and nothing bounds it.** Two directories today, every `.go` file in
   each that holds a term, held until `checkCoresNoteNamesNothingThatIsGone`
   returns. That is right at this size and it is the shape
   repositoryParseBudget exists to govern one level up — a cache with no
   stated limit is a cache nobody notices growing, and the thing that would
   grow it is a third package carrying a timings record, which
   `timingsRecordCopies` explicitly contemplates.
3. **(age 0 · value low) The three-readings-is-chasing lesson is in one
   record and applies to every figure in the repository.** internal/theme-
   history's plain row now says why sixteen readings and not three, and
   wasm/verify's `wholeFile`, its GOMAXPROCS table, its `foldWalk`, and every
   inline figure in both packages were set from three or seven. None of them
   is wrong; all of them are narrower than the truth by the same argument, and
   the argument is currently written in the one place that happened to trip
   over it.
4. **(age 0 · value low) `recordList` and `walkList` and `repositoryWalkList`
   and `enumerationList` are four functions that join a slice of things into a
   message.** Each sorts, each formats slightly differently, and three of them
   live in repowalks_test.go. That is the cheapest kind of duplication and the
   easiest to keep adding to — this session added the fourth.
5. **(age 0 · value low) The `besides` pass runs inside the repository-walk
   test and reports through it, but it is not about a repository walk.** It
   sits in the same function as the budgets, shares their log line, and its
   findings arrive under a test named
   TestTheRepositoryWideWalksInThisPackageAreTheOnesDecidedOn — which is the
   one thing a `besides` row is defined as not being. A subtest would give it
   its own name and its own failure boundary, which is the same move
   copies_test.go made two sessions ago and for the same reason.
6. **(declined, non-goal) The diff-and-print remainder stays unmeasured.** See
   `ai_docs/plans/non_goals.md` — what was declined, the argument, and the two
   conditions that would change it.
