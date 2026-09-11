# Session: a rule that caught none of what it was written for, and the one defect it found anyway

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-11 (follows "three-copies-of-two-numbers-two-of-them-wrong-and-a-count-the-commit-that-records-it-invalidates")

## Ask

`/loop` iteration 9 of 9, the last. The item: both records now say in writing
that a reading belongs in the record and an order of magnitude belongs in
prose, and nothing checks it. Measure before writing — the last two rules
measured this way went four-real-of-six and zero-of-fifteen.

This one went **one-real-of-six**, and the way it got there is the whole
iteration.

---

## The discriminator works

A duration in a comment is one of two things, spelled identically. `250ms` is
a debounce this code performs; `250ms` is also something somebody timed.

The records' own doctrine separates them: *the spread is why the recorded
figures are RANGES*. A specified duration is one number because the code
specifies one. Measured over every Go comment outside `ai_docs`:

    215   duration figures
    140   single values — every one sampled is a duration the code performs:
          a CSS transition, a debounce, a poll period, a long-press threshold
     75   ranges — the readings

And of the 75, every one lives inside a timings record except a single
`5–20ms` in hooks_test.go, which is what that test's own producers are
configured to do. The convention is already followed almost perfectly.

## And then the rule caught nothing it was written for

A figure is plainly fine when the paragraph it stands in is ABOUT a record, so
the first form let a comment group naming a record through.

Run against the tree as it stood two commits ago — where five copies had just
been found by hand — it found **none of them.** Every one of those paragraphs
named the record it was out of step with. main.go's table said *"Three runs on
the machine themehistoryTimingsTakenOn names"* directly above numbers that
record did not carry.

I had written the opposite into the rule's own header, as a claim, before
testing it: *"this rule catches those because they are outside the
declaration."* It does not. The header was wrong for the same reason the
figures were — asserted rather than taken.

## Without the escape it fires, and five of six are right

    main.go:919      0.18–0.30ms against a recorded 0.18–0.29ms      REAL
    main.go:661 ×2   "this table said 30.14–30.42s against 399–401ms"
                     — a sentence about what was wrong, written last session
    timings_test.go  "0.87–0.92s measured, which is what the 0.83s this
                     line used to quote"
    repowalks_test.go:49 ×2   "1.18–1.40s of a 2.88–2.97s package", a sum
                     of record fields, named in the record's own re-taking
                     list

Two classes, both of which recur by construction. **A sentence quoting what a
figure used to say** — every re-taking adds one. **A figure derived from record
fields** — which is the arrangement iteration 7 deliberately built. Neither
carries a marker distinguishing it from a copy, and an exemption table growing
by one entry per re-taking is a table documenting the record's history inside a
checker.

Declined, in `ai_docs/plans/non_goals.md`, with the numbers and with the
condition: a marker for "this is a quotation of a past reading" — which is the
same missing convention `renamedTestsStillNamed` is a table instead of. If one
ever arrives, both rules become writable at once.

## The one defect it found anyway

`main.go:919` said the healthy retire costs **0.18–0.30ms** "on the machine in
themehistoryTimingsTakenOn". The record says **0.18–0.29ms**. A copy, out of
step by a hundredth, attributed in the same sentence.

The hand sweep one iteration earlier missed it, because that sweep grepped for
the record's exact strings and a rounded copy is not one of them. main.go names
`themehistoryTimingsTakenOn.batchRetire` now instead of restating it.

A rule that does not survive its own measurement can still pay for the
measurement.

---

## Verification

Eleven paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      rc 0
    GOOS=js GOARCH=wasm go build ./...

**Figures.** Both packages in band at the end:

    wasm/verify             2.866–2.946s   recorded 2.88–2.97s
    internal/themehistory   2.999–3.053s   recorded 2.92–3.22s

## What the nine iterations are evidence of

Five rules were measured before being written. Two were built and three were
declined, and the deciding number was the same each time: how many of the
findings are real.

    a test named in prose resolves          4 real of 6      built
    the two comment-text rules              0 of 0, zero exemptions   built
    a doubled word                          0 real of 9      declined
    a backquoted name is a declaration      0 real of 15     declined
    a wall clock lives in a record          1 real of 6      declined

The two that were built share a property the three that were declined do not:
the shape they look for belongs to one language and one meaning. `Test[A-Z]` is
a Go test and nothing else; a comment line repeated on itself is a botched
edit and nothing else. A doubled word is English, a backquoted identifier is
any of four languages, and a wall clock is either a reading or a duration the
code performs.

The other thing five measurements say, which no single one of them does: the
cost of measuring first is about an hour across nine iterations, and it
prevented three checks that would each have needed an exemption table on the
day it was written.

## Next

Sorted by **value**, highest first; age breaks ties. Age is how many saved
sessions ago the item was first raised, counted in `ai_docs/claude_sessions/`
and measured from this doc, so `age 0` means it was raised here.

1. **(age 0 · value medium) There is no way to mark a figure as a quotation of
   a past reading.** Two declined rules failed on the same gap, one session
   apart: the wall-clock rule cannot tell "30.14–30.42s, which is what this
   said before" from a live copy, and the test-name rule needed
   `renamedTestsStillNamed` as a table for exactly the same reason. A marker —
   a convention, not a mechanism — would make both writable, and the second
   one could then drop its table. The work is deciding the marker and applying
   it, and the corpus is small: two entries in that table and about four
   sentences found tonight. Worth doing when somebody is editing this prose
   anyway rather than as its own errand.
2. **(age 4 · value low) `renamedTestsStillNamed` is a graveyard with no
   pruning.** Two entries, both current. Subsumed by item 1 if that is ever
   done.
3. **(age 10 · value low) Every Next list in this loop was written by the
   session that would not work it.** Nine iterations, and the pattern held to
   the end: tonight's item said "measure before writing" and was right, and
   said nothing about the rule's escape clause, which is what actually decided
   it.
4. **(declined, non-goal) A wall clock in prose is not held to living in a
   record.** See `ai_docs/plans/non_goals.md`.
5. **(declined, non-goal) A backquoted name in prose is not held to being a
   declaration.** See `ai_docs/plans/non_goals.md`.
6. **(declined, non-goal) A `t.Run` inside a loop counts as one question.**
   See `ai_docs/plans/non_goals.md`.
7. **(declined, non-goal) `packageLevelCallsTo` cannot tell an initializer
   from a stored function value.** See `ai_docs/plans/non_goals.md`.
8. **(declined, non-goal) The diff-and-print remainder stays unmeasured.**
   See `ai_docs/plans/non_goals.md`.
