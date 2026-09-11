# Session: a row that could not be wrong, a note nothing could read, and a saving that was a thousandth

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-10 (follows "a-stat-that-said-yes-to-an-empty-room-a-rule-that-could-not-see-lowercase-and-a-read-nothing-counted")

## Ask

"Work the items in the Next list." Six items — three medium, three low — every
one raised by the previous session about what it had just built, two iterations
into the same loop.

The previous session's shape was **a fix that leaves a smaller one behind**.
This one is about what a written-down claim is worth:

    the row          `besides` said what a read costs, and a sentence about a
                     read is a claim nothing can be wrong about
    the note         a `coresAttribution` this could not evaluate passed the
                     record check and was exempt from both arms over it, in
                     silence
    the saving       the structural argument predicted half the scan and the
                     clock says a thousandth, and the honest thing is to
                     write down which

Nothing outside tests changed. `git diff` touches four test files.

| item | value | where | shape |
|---|---|---|---|
| 1 | medium | repowalks_test.go | a row that could not be wrong |
| 2 | medium | copies_test.go | a saving that was a thousandth |
| 3 | medium | copies_test.go | a note nothing could read |
| 4 | low | copies_test.go | two parameters carrying one fact |
| 5 | low | importnames_test.go | the same directory read seven times |
| 6 | low | wasm/verify record | the other half of a table |

---

## Item 1 · a row that could not be wrong

`besides` was a sentence, and `asks` is not — that is the whole difference. A
row in `asks` describes a walk the census FINDS, so a row for a walk that has
gone is a finding and an unlisted walk is another. A sentence about a read is a
claim nothing can be wrong about: the call deleted, the function renamed, the
cost changed by a factor, and the row reads exactly as it did.

`besidesRow` names the function the read goes through:

    through    the handle. Held to being declared in this package and to being
               called by something
    reads      what it reads, for a reader deciding whether it should have
               been a walk
    costs      the figure, and how it was measured

and `declaredAndCalledBy` is the pass over it — which is what `runs` already
does for a helper, applied to a read instead of a walk. The COST is still a
written number nobody re-takes automatically, like every other figure in this
repository's prose; what is no longer possible is the row outliving the read.

It is a separate scan from `callsTo` and the header says why: `callsTo` prices
a call, reading the loops around each site and multiplying the bounds in,
because what it counts is repository walks and the budget is made of that
number. This asks the smaller question for a read the budgets deliberately do
not govern, and pricing it would mean deciding what a `besides` number means
when the read is in a loop — a question nothing has had to ask.

Both failures demonstrated: the function renamed away, and the function
declared with nothing calling it.

---

## Item 2 · a saving that was a thousandth

`identifiersIn` scanned both record packages for the UNION of every note's
terms, which pays the cross-reference case on every green run — and that case
has never happened. Each note names its own package's terms; that is what a
note is for.

So the first pass asks each package about its own note and the second runs only
over what the first could not find. On a repository where the notes are right,
the second pass does not happen.

The measurement is the interesting part. Predicted: half the scan. Measured,
over seven takings of sixty runs:

    before    0.011s    12.11–12.30s against 11.38–11.56s
    after     0.010s    12.19–12.28s against 11.57–11.99s

A thousandth, at the edge of what the measurement can see. The reason is inside
`identifiersIn` — it stops looking for a term once it has found one, and the
terms the two notes share are words like `runtime` and `min`, which the first
file answers. Scanning the wrong package for them was never expensive; it was
work for a case that has not happened.

Kept, and the header says what it bought, because a structural argument that
predicts a saving and delivers a thousandth is one somebody should be able to
check.

The cross-reference still resolves — demonstrated by putting
`` `enumWorkers` `` in wasm/verify's note, which the first pass cannot find and
the second reports as belonging to the package it compares itself with.

---

## Item 3 · a note nothing could read

The record check holds each package to DECLARING a `coresAttribution`.
`stringLiteralValue` reads a string constant written as literals joined by `+`,
which is what a sentence that has to fit in a column of source looks like — and
returns "" for anything else. A note assembled by a function, or built out of
other constants, is therefore declared and unreadable: it passes the record
check, comes back empty, and both directions then skip it in silence.

That is a note claiming an attribution with nothing whatever holding it, which
is the state the note itself was in two sessions ago — the same fault, arriving
one level inside the thing that fixed it.

The arm is in `checkTimingsRecordCopies`, once per record directory, and says
which of the two things went wrong: the package has no note (the record check's
finding) or has one whose text cannot be read (this one). Demonstrated by
declaring `const coresAttribution = coresNoteBody`, which compiles, reads
identically to a person, and was invisible.

---

## Item 4 · two parameters carrying one fact

The two checks over a note took `noteText` and `noteTerms`, and the terms are
derived from the text. Two parameters carrying one fact is how a caller comes
to pass a stale one.

`coresNote` is one value — the text, which is what says whether the note could
be read at all, and the terms, which are what it names — derived once where the
text is found. There is no arrangement in which the two disagree.

And while tidying: the `sortedKeys` helper this session added was a duplicate of
`keysOf`, which is already generic over the map's value type and already sorts.
Deleted, and the four call sites use `keysOf` — which is a small thing and
exactly the class the copies census exists for, arriving inside the file that
census lives in.

---

## Item 5 · the same directory read seven times

`holdsAPackage` ran once per ASKING, and the loop is over askings rather than
paths — six censuses asking about `os/exec` is six times round it. The finding
stays per asking, because each call site is a place somebody would fix. The
READ does not: what is in a directory does not change between two questions
about it in one run.

Memoised on the directory. The `cmd` break-test still reports, which is what a
cache has to prove.

---

## Item 6 · the other half of a table

The previous session widened `wholeFile` to 2.78–2.93s and left the GOMAXPROCS
table under it from an earlier afternoon — the same fault it had just closed in
the other record, still open in this one. A reader comparing a one-core run
against the number above would have been comparing two days.

Re-taken whole, three runs a row:

    GOMAXPROCS    now             was
    1             3.179–3.247s    3.08–3.27s
    2             2.833–2.877s    2.78–2.88s
    4             2.799–2.857s    2.74–2.79s
    8             2.805–2.849s    2.75–2.85s

Only the four-core row moved, by seven hundredths at the top end, and it is
widened to 2.74–2.86s. The other three came back inside the ranges they already
had, which is the useful half of re-taking a table nothing has changed. The
shape did not move: one core is a tenth dearer and two is where it stops
improving.

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

    wasm/verify/copies_test.go        items 2, 3, 4
    wasm/verify/repowalks_test.go     item 1
    wasm/verify/importnames_test.go   item 5
    wasm/verify/timings_test.go       item 6

**Five break-tests run, all five fired**, and one re-run to prove a cache
changed nothing:

    the row's function renamed   "nothing in this directory declares a `func
                                 identifiersIn`"
    the row's function uncalled  "nothing in this directory calls
                                 `identifiersIn`" — the opposite failure, and
                                 as invisible
    an unreadable note           `const coresAttribution = coresNoteBody`:
                                 declared, compiles, reads the same to a
                                 person, and was exempt from both arms
    the cross-reference          `` `enumWorkers` `` in wasm/verify's note:
                                 the lazy second pass finds it in the other
                                 record package and logs it rather than
                                 reporting it
    the memoised read            `cmd` still reports "holds no Go file"

**Figures.** `wasm/verify` **2.798–2.917s**, inside its 2.78–2.93s.
`internal/themehistory` **3.024–3.147s**, inside its 3.01–3.22s. Neither record
needed touching, which is the first time in three iterations that has been
true.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

1. **(age 0 · value medium) The second pass of the cores-note scan re-reads
   and re-parses files the first pass already read.** `identifiersIn` takes a
   directory and a term list, reads the directory, and throws everything away
   — so the unresolved-term pass over the same two directories does the
   `os.ReadDir`, the `os.ReadFile` and the `parser.ParseFile` again, for files
   whose trees were in memory a microsecond earlier. It costs nothing today
   because the second pass does not run on a green repository, which is
   exactly the argument that stops being true the first time a note is wrong
   — the run that is already failing is the one that pays twice.
2. **(age 0 · value medium) `declaredAndCalledBy` and `callsTo` are two scans
   of one parse asking overlapping questions.** Both walk every function in
   this directory looking for bare calls to a named function; one prices the
   loops around the site and one does not. The header says why they are
   separate and that argument is about what the ANSWER means, not about the
   walk — the walk is the same walk, run twice over the same trees, once for
   seven walk names and once for one `through`. What would merge them is a
   single pass that collects call sites and lets each caller decide whether to
   price them.
3. **(age 0 · value low) The `costs` figure in a `besides` row is still a
   number nobody re-takes.** `through` is now held to being declared and
   called, which is what item 1 was for, and the measurement beside it is
   prose: 0.010s, seven takings, two ranges. If the read grows a second
   directory or the term list doubles, the row goes on saying 0.010s and
   nothing anywhere disagrees. That is true of every wall-clock figure in this
   repository and it is worth saying once, here, that the field which now has
   an arm over half of itself has none over the other half.
4. **(age 0 · value low) The unreadable-note arm fires per record directory
   and the record check fires per RECORD.** A package with two
   `…TimingsTakenOn` records in it — which nothing forbids below
   `timingsRecordCopies` — gets the missing-note finding twice and the
   unreadable-note finding once. The two arms are about the same package-level
   fact and they count it differently, which is the one-per-row rule this
   repository writes everywhere else.
5. **(age 0 · value low) `stringLiteralValue` is the only thing deciding what
   a note is, and it lives beside the check rather than beside the note.** It
   reads literals joined by `+` and returns "" for everything else, and what
   "everything else" covers is now a finding rather than a silence — but the
   two packages that WRITE a note have nothing telling them the shape. A
   comment on `coresAttribution` in each record saying "literals and `+`,
   because copies_test.go reads this without running it" is one line and is
   where somebody about to break it is looking.
6. **(declined, non-goal) The diff-and-print remainder stays unmeasured.** See
   `ai_docs/plans/non_goals.md` — what was declined, the argument, and the two
   conditions that would change it.
