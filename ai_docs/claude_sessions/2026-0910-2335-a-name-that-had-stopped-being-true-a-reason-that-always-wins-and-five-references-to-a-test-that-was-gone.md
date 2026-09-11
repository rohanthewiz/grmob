# Session: a name that had stopped being true, a reason that always wins, and five references to a test that was gone

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-10 (follows "a-quarter-of-what-the-rule-was-about-two-budgets-that-were-both-full-and-two-limits-moved-to-non-goals")

## Ask

`/loop` iteration 4 of 9 (the count was raised from 5 mid-run). The list's top
item posed two options and asked which: rename the walk to what it is, or cap
it at five questions and re-argue the parse budget instead.

**Both.** They are not alternatives — one is about the name being true, the
other about what happens when the next question arrives — and doing only one
leaves the other half of the problem.

| item | value | outcome |
|---|---|---|
| 1 | medium | the walk has its own file and name; a question budget holds it |
| — | — | five prose references to the renamed test, none of them held |

---

## The skip was over-broad, and measuring it made the split safe

Before moving anything: the walk skipped `wasm/verify/copies_test.go` entirely
— not parsed, invisible to all five questions — on the grounds that the file
"names the fields and the arm in its own prose and declares neither".

Taken out and run, the only findings that file produces are **four stale
figures against the file-count rule**, from counts quoted as examples. The
record question, the two import questions and the cores-note question all read
it clean. The skip was written for the record question's sake and had been
protecting a different one for some time.

So the exemption belongs to the RULE and not to the walk, which is what
`commentRulesFile` already did for the comment rules:

    proseFigureRulesFile   quotes counts as examples of the form it holds
    commentRulesFile       quotes the incident verbatim, breaking both of its
                           own rules

A question that needs no exemption no longer inherits one. The immediate gain
is that copies_test.go's 1,100-odd comment lines are now read by the comment
rules, which is break-tested below.

## The name

The walk is `sharedparse_test.go` now, and
`TestTheQuestionsOnTheSharedRepositoryParseAreTheOnesDecidedOn`.

What never changed is the thing that actually unifies its questions, and it
was written down from the first day: **they are one arm BECAUSE they are one
repository-wide parse.** That is a fact about cost, not about subject. A file
named for the subject of its first question was going to keep being wrong as
questions arrived, and two arrived in one session.

Each question's check is owned by the file that owns its subject, which was
already true of three of the five before this file existed:

    sharedparse_test.go    the walk, the questions, the exemptions
    copies_test.go         the records, the cores note, packageSource
    importnames_test.go    the resolver copies and the paths asked about
    prosefigures_test.go   the tracked-Go-file figures and the form rule
    commenttext_test.go    the two comment rules

copies_test.go went from 1,808 lines to 1,235, and what is left in it is what
its name was always about.

## The reason that always wins

`walkQuestionBudget = 5`, armed by the census in repowalks_test.go beside
`repositoryWalkBudget` and `repositoryParseBudget`.

The argument for it is the shape of the argument FOR each new question:
`repositoryParseBudget` is full, so a question needing a repository-wide parse
has exactly one cheap home — the walk that already has one. That is correct,
and it is correct **every time**. The parse budget is full whoever is asking;
the shared walk always has the parse; the question always needs no enumeration
of its own.

A reason that always wins is not a reason. Two questions arrived on that walk
in a single session, each with it, each right, and nothing in the repository
would have objected at nine — which is the thing every row in the walk table
exists to stop.

The three budgets now bound the three ways this cost grows: how many walks,
how many of them parse, and how much any one of them is carrying. The last had
no number at all.

---

## Five references to a test that was gone

The rename left five sentences naming a function that no longer exists — two
in timings_test.go, two in versionorder_test.go (one of them inside a string
constant another census prints), one in the moved figures code. `gofmt`,
`go vet`, `go build`, `go test ./...`, `-race` and all four verify scripts
were green with every one of them in place.

All five are fixed. None was found by a check; they were found by grepping for
the old name after the rename, which is the thing this package writes arms
instead of. See the Next list — the shape is one this repository already holds
in one narrow place (`termsNamedIn` holds the backquoted terms in
`coresAttribution` to being terms that exist, in both directions), and this is
the general case of it.

---

## Verification

Eleven paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      rc 0
    GOOS=js GOARCH=wasm go build ./...

**Two break-tests:**

    walkQuestionBudget = 4    "sharedparse_test.go:229 answers 5 question(s)
                              and the budget is 4", with both honest answers
                              spelled out
    a garbled comment in      a duplicated, tab-joined line appended to
    copies_test.go            copies_test.go — a file the comment rules could
                              not see before the walk-level skip was split.
                              Named

**The file-count arm fired twice more**, as designed: the two new files took
the tree from 387 to 389 and it named all five sentences and the number to
write, once when the split landed and once for the count.

**Figures.** Package **2.824–2.907s** over twelve, inside the recorded
2.78–2.93s. Nothing about this iteration was a cost change: the same parse,
the same questions, in different files.

## What this iteration is evidence of

The move was the risky half and measurement made it cheap. Taking the skip out
and reading what actually fired turned "this file is exempt for reasons" into
"this file trips exactly one rule", which is what allowed the split at all.

And the thing the iteration did NOT plan for is the one worth carrying: a
rename is a repository-wide edit that nothing in eleven verification paths can
see, because a name in a comment is text and nothing reads comments as text
except two rules about tabs.

## Next

Sorted by **value**, highest first; age breaks ties. Age is how many saved
sessions ago the item was first raised, counted in `ai_docs/claude_sessions/`
and measured from this doc, so `age 0` means it was raised here.

1. **(age 0 · value high) A declaration named in prose is not held to
   existing.** This session renamed one test and left five sentences naming
   the old one, through every verification path clean. The repository already
   does this in one place and in both directions — `termsNamedIn` holds
   `coresAttribution`'s backquoted terms to being things its package still has
   — so the shape is known and the question is scope and form. The obvious
   rule is: an identifier-shaped word in backquotes, in a comment, resolves to
   something this repository declares. The work is measuring the false
   positive rate before writing it, which is what killed the doubled-word rule
   two iterations ago: prose is full of backquoted things that are not
   declarations (`git ls-files`, `-run NoSuchTest`, `//go:build ignore`,
   `for range 2`). Measure first.
2. **(age 5 · value low) Every Next list in this loop was written by the
   session that would not work it.** Four iterations of evidence. This one is
   the cleanest: the item said "rename OR cap", and the answer was both, which
   a list written by the session that had just argued itself into a corner
   could not have contained.
3. **(declined, non-goal) A `t.Run` inside a loop counts as one question.**
   See `ai_docs/plans/non_goals.md`.
4. **(declined, non-goal) `packageLevelCallsTo` cannot tell an initializer
   from a stored function value.** See `ai_docs/plans/non_goals.md`.
5. **(declined, non-goal) The diff-and-print remainder stays unmeasured.**
   See `ai_docs/plans/non_goals.md`.
