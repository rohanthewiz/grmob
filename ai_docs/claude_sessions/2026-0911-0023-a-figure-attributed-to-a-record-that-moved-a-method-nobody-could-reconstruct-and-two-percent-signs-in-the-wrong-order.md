# Session: a figure attributed to a record that moved, a method nobody could reconstruct, and two percent-signs in the wrong order

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-11 (follows "a-rule-declined-on-fifteen-findings-three-stale-names-outside-go-and-a-band-that-had-to-be-re-taken")

## Ask

`/loop` iteration 7 of 9. The item asked whether a record that re-takes its own
rows on a rule should have a rule for the figures that explain those rows, and
asserted that `0.09s`, `0.16s`, `0.18s` and `0.010s` were stale by
construction because none was re-taken when the package's cost moved.

The premise was half right, and the half that was wrong is the interesting one.

---

## Two of the three still hold

Re-taken, seven runs apiece:

    walkEnumerate   0.08–0.10s   recorded 0.09s   holds
    walkRead        0.15–0.17s   recorded 0.16s   holds
    walkParse       0.20–0.24s   recorded 0.18s   moved

They are not stale by construction, and the reason matters: these measure the
WALK, and what moved tonight was the questions riding on it. The tree grew by
one per cent and the parse figure by eleven, which is the tree plus the
machine, not the session's work.

## The method was not recoverable, and guessing it was off by five times

The first re-taking reconstructed the work by hand — `git ls-files`, a stat per
file, a read, a parse — and came back at **0.012 / 0.020 / 0.058s**, roughly a
fifth of the recorded figures. That reconstruction was wrong, not the record.

What the recorded numbers include is the whole wall clock of a test that walks
at that depth: `citingFiles` scans every file's text for citations on the way
past, and the test then asks its own question of what came back. **The depth is
the shape of the walk; the figure is the whole of the cost.**

Nothing said so. `internal/themehistory`'s record carries a `# Re-taking it`
table of exact commands and this one had two lines in its own — for `wholeFile`
and `foldWalk` — and nothing for the three depths. They have one now, with the
`-v -run` command for each, and the paragraph says what a reconstruction gets
wrong and by how much.

---

## The real finding: attribution by reference is a pointer that moves

Every wall clock in this package's prose says *"where verifyTimingsTakenOn was
taken"*. That is the discipline this record exists for, and it is not
sufficient.

When `wholeFile` was re-taken last iteration — 2.78–2.93s to 2.88–2.97s —
**every sentence citing this record silently began claiming to be from a taking
it was not from.** One of the three figures had moved with it, two had not, and
nothing anywhere said which. The reference is to a record, and the record is
mutable.

A wall clock still cannot be an arm. What it can be is a FIELD:

    walkEnumerate, walkRead, walkParse   now on verifyTimingsTakenOn. The two
                                         failure messages in repowalks_test.go
                                         read them, so a re-taking is one edit
    gitquoting_test.go's copy            "against the 0.18s a repository-wide
                                         parse costs" became "at twenty times
                                         that", naming the field. A ratio does
                                         not need re-taking
    repowalks_test.go's header           "1.18–1.40s of a 2.88–2.97s package",
                                         summed from the fields at both ends.
                                         Prose cannot read a field, so this is
                                         the one left that is by hand

The record now carries a `# What a re-taking also moves` list, because the list
is what there is when an arm is not available. The general lesson is the one
this repository keeps arriving at from different directions: **a figure quoted
in two places is a copy, and attributing the copy to the original does not make
it one thing. Moving it into the original does.**

---

## Two percent-signs in the wrong order

Feeding the record's field into the parse-budget message put the new `%s`
ahead of the existing one in the format string and the arguments behind it.
`go vet` is clean on that — both are strings — and so is every test, because
the message only renders when the check fails.

Caught by breaking the budget and reading the output, which is the same
discipline that has caught every message this session and is the only thing
that can: a failure message is code that runs exactly when nobody is watching
it in a green run.

## And three more stale figures, found by sweeping

    "about a second of a 2.7-second package"    → 1.18–1.40s of 2.88–2.97s
    "eight of thirty-seven where this record    → ten of forty-one
     was taken"
    "spreads over 2.49–2.59s ... a 2.7s run"    kept, and marked as the
                                                illustration it is: what it
                                                demonstrates is the WIDTH of
                                                one machine's spread, which
                                                has not changed

The last is the distinction worth having. A figure used to show a SHAPE is not
stale when the magnitude moves; a figure used to price something is. The
sentence now says which it is.

---

## Verification

Eleven paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      rc 0
    GOOS=js GOARCH=wasm go build ./...

**Two break-tests, both read rather than counted:**

    repositoryParseBudget = 3   renders "Each parse is 0.20–0.24s where
                                verifyTimingsTakenOn was taken" — the field,
                                in the right slot
    a row's depth changed       renders "0.08–0.10s, 0.15–0.17s and
                                0.20–0.24s respectively", all three from the
                                record

**Figures.** Twelve runs at **2.862–2.969s** against the 2.88–2.97s recorded
last iteration. The low end is a fiftieth below the floor, which is the record
telling the truth about itself: its own doc says these ranges are narrower than
the truth by roughly a twentieth at each end, and that a reader landing just
outside one has been told not to judge by it. Not re-taken for that.

## What this iteration is evidence of

The item's premise was wrong in a way only measurement could show, and right
about something it had not identified. "These figures are stale" was false for
two of three. "A record that re-takes its rows should have a rule for the
figures explaining them" was true, and the rule turned out not to be about
re-taking at all — it is that a figure explaining a record should not be a
copy of one.

The five-times error is the part worth carrying. A figure with no method beside
it is not merely hard to re-take; it is easy to re-take WRONGLY, and the wrong
answer looks like a finding.

## Next

Sorted by **value**, highest first; age breaks ties. Age is how many saved
sessions ago the item was first raised, counted in `ai_docs/claude_sessions/`
and measured from this doc, so `age 0` means it was raised here.

1. **(age 0 · value low) The other record has not been read for the same
   fault.** `internal/themehistory/timings_test.go` carries the same shape —
   figures in prose attributed to a record that gets re-taken — and this
   session only proved the fault exists in wasm/verify's. Its `wholePackage`
   row was read tonight and is in band, so nothing is known to be stale; what
   is not known is whether any of its prose figures are copies that should be
   fields. The work is one sweep of that file, and the thing to look for is a
   number quoted in a message rather than read from the record.
2. **(age 2 · value low) `renamedTestsStillNamed` is a graveyard with no
   pruning.** Two entries, both current. An entry whose sentence has since
   been rewritten is dead weight reading as a guard, which is the failure
   `citationExempt`'s own assertion catches one file over. Not worth an arm at
   two; written down in the table's doc.
3. **(age 8 · value low) Every Next list in this loop was written by the
   session that would not work it.** Seven iterations. Tonight is the clearest
   case yet of the failure mode being specific rather than general: the item
   named four figures as stale and two were fine, because the session that
   wrote it had just re-taken a different number and reasoned by association.
4. **(declined, non-goal) A backquoted name in prose is not held to being a
   declaration.** See `ai_docs/plans/non_goals.md`.
5. **(declined, non-goal) A `t.Run` inside a loop counts as one question.**
   See `ai_docs/plans/non_goals.md`.
6. **(declined, non-goal) `packageLevelCallsTo` cannot tell an initializer
   from a stored function value.** See `ai_docs/plans/non_goals.md`.
7. **(declined, non-goal) The diff-and-print remainder stays unmeasured.**
   See `ai_docs/plans/non_goals.md`.
