# Session: three copies of two numbers, two of them wrong, and a count the commit that records it invalidates

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-11 (follows "a-figure-attributed-to-a-record-that-moved-a-method-nobody-could-reconstruct-and-two-percent-signs-in-the-wrong-order")

## Ask

`/loop` iteration 8 of 9. The item was a sweep: the previous iteration proved
that figures attributed to a mutable record go stale silently, and proved it in
wasm/verify's record only. `internal/themehistory` carries the same shape and
had not been read for it.

It has the same fault, worse — and a second one that is not the same problem at
all.

---

## The first fault: three copies of two numbers, two of them wrong

    the record       30.40–30.52s against 398–405ms, 75.1–76.4×
    main.go          30.14–30.42s against 399–401ms, 75–76×
    main_test.go     30.14–30.42s against 399–401ms

    the record       wholeRun 1.52–1.60s
    main_test.go     "themehistoryTimingsTakenOn.wholeRun: 1.49–1.61s"

All three attributed. main.go's table says *"Three runs on the machine
themehistoryTimingsTakenOn names"* directly above numbers that record does not
carry, and main_test.go names the field and then restates its value wrongly in
the same sentence.

And a fourth value inside the record itself: `perObjectRun`'s own doc table
says *"the fetches 395–405ms, the batched half above"* where the field above
says 398–405ms.

**The fix is not to re-take the copies.** main.go — production code, which
cannot reference a test's record in anything but prose — now states the orders
of magnitude its argument rests on and names the field for the readings:

    one `cat-file -p` each   one process per object, thirty seconds
    one `cat-file --batch`   one process, about four hundred milliseconds
                             seventy-odd × on the fetches alone

A ratio and an order of magnitude do not drift; a reading does. main_test.go
names the field instead of restating it.

## The second fault: a count the commit that records it invalidates

`2906 objects` and `88 ls-tree` are not readings of a machine. They are
readings of **this repository's own history**, and it grows — including by the
commits that write them down.

Re-taken tonight: **2955 objects, 89 commits.** They moved during the session
that found them.

This is the shape iteration 1 armed for `tracked Go files`, arriving in a
package where it was quoted fourteen times across two packages. The counts stay
in the record, where they say what the timings are a reading OVER; everywhere
else now says `one ls-tree per commit` and `every object in the history`, which
is what those sentences were always about. The arm prints the live number on
every run, which is the only place a count like this can be right.

## What was re-taken

    wholeRun       1.52–1.60s → 1.56–1.67s, 2906 → 2955 objects
                   (seven runs)
    perObjectRun   30.40–30.52s → 30.53–30.84s; 398–405ms → 409–412ms;
                   75.1–76.4× → 74.7–75.1×; the trees 0.87–0.92s → 0.87–0.91s;
                   the parse 0.118–0.120s → 0.125–0.145s; together
                   1.39–1.44s → 1.41–1.47s (three runs, under
                   GRMOB_PER_OBJECT_FETCH=required)

`wholePackage` was read and left alone: 2.977–3.114s against a recorded
2.92–3.22s.

The record carries a `# What a re-taking also moves` section now, as
wasm/verify's does — and this one has to explain both faults, because they have
different fixes. A figure that drifts with the machine gets re-taken. A figure
that drifts with the repository gets deleted from prose.

---

## Verification

Eleven paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      rc 0
    GOOS=js GOARCH=wasm go build ./...

**Figures.** Both packages read against their own bands at the end:

    internal/themehistory   2.977–3.114s   recorded 2.92–3.22s
    wasm/verify             2.862–2.941s   recorded 2.88–2.97s

The wasm/verify low end is a fiftieth under its floor for the second iteration
running, which its own record predicts of itself in writing and says not to
judge by.

**No break-tests this iteration.** Nothing here is a new arm — it is a record
re-taken and fourteen copies removed — and the arms that already exist over
this record (`checkTimingsRecordCopies`, `checkCoresNoteNamesEveryScaledTerm`,
`checkCoresNoteNamesNothingThatIsGone`) ran green throughout, which is what
says the note still names every term this package has after the prose moved.

## What this iteration is evidence of

The previous iteration found a fault and fixed it where it was found. Sweeping
the sibling for the same fault found it three times over — and found a second
one underneath that the first package does not have, because wasm/verify's
figures are about a tree that changes slowly and this one's are about a commit
history that changes every time anybody writes any of this down.

The generalisation both packages now carry: **a figure quoted in two places is
a copy, and attributing the copy does not make it one thing.** The new half is
that some figures cannot be quoted anywhere at all, because the act of quoting
them is the act that makes them wrong.

## Next

Sorted by **value**, highest first; age breaks ties. Age is how many saved
sessions ago the item was first raised, counted in `ai_docs/claude_sessions/`
and measured from this doc, so `age 0` means it was raised here.

1. **(age 3 · value low) `renamedTestsStillNamed` is a graveyard with no
   pruning.** Two entries, both current. An entry whose sentence has since
   been rewritten is dead weight reading as a guard, which is the failure
   `citationExempt`'s own assertion catches one file over. Not worth an arm at
   two; written down in the table's doc.
2. **(age 0 · value low) Nothing holds a figure to being attributed.** Both
   records now say, in writing, that a reading belongs in the record and an
   order of magnitude belongs in prose — and nothing checks it. The obvious
   arm is the one iteration 1 built for file counts, generalised: a decimal
   followed by `s` or `ms` in a comment, held to being in a record or
   explicitly marked as an order of magnitude. Measure before writing it: the
   corpus is every timing figure in both packages, and the last two rules
   measured this way went four-real-of-six and zero-of-fifteen.
3. **(age 9 · value low) Every Next list in this loop was written by the
   session that would not work it.** Eight iterations. Tonight's item was
   exactly right and understated: it asked for a sweep and expected the same
   fault, and the sweep found a second one with a different fix.
4. **(declined, non-goal) A backquoted name in prose is not held to being a
   declaration.** See `ai_docs/plans/non_goals.md`.
5. **(declined, non-goal) A `t.Run` inside a loop counts as one question.**
   See `ai_docs/plans/non_goals.md`.
6. **(declined, non-goal) `packageLevelCallsTo` cannot tell an initializer
   from a stored function value.** See `ai_docs/plans/non_goals.md`.
7. **(declined, non-goal) The diff-and-print remainder stays unmeasured.**
   See `ai_docs/plans/non_goals.md`.
