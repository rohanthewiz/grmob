# Session: a quarter of what the rule was about, two budgets that were both full, and two limits moved to non-goals

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-10 (follows "a-row-that-said-three-a-rule-declined-on-nine-measurements-and-an-arm-that-fired-on-its-own-new-file")

## Ask

`/loop` iteration 3. The list's top item was the scope decision left open by
iteration 2: whether the two comment rules should read the repository rather
than one directory, and what that costs.

| item | value | outcome |
|---|---|---|
| 1 | medium | widened to the repository, riding the shared parse |
| 3, 4 | low | moved to `ai_docs/plans/non_goals.md`, argued |

---

## The item's own numbers were wrong, in both directions

It said "a sixth repository walk against a budget of six". Neither figure was
right, and the truth made the decision easier rather than harder:

    repositoryWalkBudget    7, against 7 walks
    repositoryParseBudget   4, against 4 parses

**Both are full.** A new walk is not a budget to raise, it is the conversation
those two constants exist to force — and it would be paying its own
`git ls-files` and its own read of every tracked file to reach files that four
existing walks have already parsed. `repositoryParseBudget`'s own doc had
already written the answer: *a fifth is where a shared parse becomes the
cheaper of two bad options.* The shared parse exists. Questions land on it.

### And the scope argument did not survive measurement either

The in-directory version was justified with "this is the package that is
mostly prose". Counted the way the arm counts, with go/parser:

    wasm/verify            12,326   24.6%
    core                    8,619   17.2%
    components              7,269   14.5%
    mobile/verify           4,600    9.2%
    htmlout                 3,248    6.5%
    internal/themehistory   2,831    5.6%
                           ──────
    repository             50,190

A rule scoped to this package read **a quarter** of what it is about. The
sentence was true about the package and irrelevant to the decision — core and
components carry another third between them and are written the same way.

The arm now reads **49,175 comment lines in 385 files**, which is four times
the coverage for about five milliseconds, measured: 0.396–0.462s for that walk
with the scan against 0.393–0.427s without it, five runs apiece. It rides a
parse somebody else was already paying for.

---

## What the fifth question costs the walk's name

The objection iteration 2 raised against widening was real and is not
dismissed: the copies walk is about shapes kept in two copies, and a garbled
comment is not one. What changed is not the objection, it is what the walk was
found to be.

Its unifying principle was never its subject. Its own header says the three
shapes are one arm BECAUSE they are one repository-wide parse — the budget
made that arrangement, not the topic — and **two of its four existing
questions already have their checks in importnames_test.go**, not in the file
the walk is named after. The arrangement is: one parse, and each question's
check owned by the file that owns its subject.

So `checkCommentText` and `commentFindingsIn` live in commenttext_test.go,
beside the rules and the argument for them, and the walk gained a collector, a
subtest and a row entry. What it costs is a file whose name describes its
largest question rather than all of them, which is now written in its header
rather than left to be noticed.

    one walk, five questions, five failure boundaries, five checks owned by
    the files that own their subjects

## The skip moved with it

`commentRulesFile` is `wasm/verify/commenttext_test.go` now rather than a bare
base name, because the walk that applies the rules enumerates the repository.
The file still quotes the incident verbatim and is still a finding about
itself twice over.

---

## Two limits moved to non-goals

Both were "stated in the message rather than resolved", carried for two
sessions each, and both are decisions rather than deferrals — which is what
`ai_docs/plans/non_goals.md` is for.

**A `t.Run` inside a loop counts as one question.** The loop's length is a
run-time fact no parse has. `priceCalls` solves the same problem one pass over
for a walk in a loop, by reading a written bound and reporting an unwritten
one — so the machinery exists, and the cost of reusing it is threading
`loopBound` and `packageLevelInts` into a census that currently reads nothing
but declarations, to price a shape nobody has written. Reconsidered when a
table-driven walk test appears.

**`packageLevelCallsTo` cannot tell an initializer from a stored function
value.** `var x = someWalk(root)` runs at init and `var f = func() {
someWalk() }` does not, and telling them apart is data flow over assignments,
struct literals and returns — which every walk here declines on purpose. The
finding is already correct without the distinction: both are a walk the budgets
are not counting and both need the same fix. Reconsidered when the case occurs
and the two kinds need different advice.

---

## Verification

Eleven paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      rc 0
    GOOS=js GOARCH=wasm go build ./...

**Two break-tests:**

    a garbled comment in core/    a duplicated, tab-joined line appended to
                                  core/style.go. Both rules name it — a file
                                  that was outside the scan an hour earlier
    asks says four                the row's fifth entry removed. Iteration 2's
                                  arm fires: 5 subtests against 4 questions

**Figures.** Package **2.816–2.929s** over twelve, inside the recorded
2.78–2.93s. An earlier taking put the top at 2.999s and was discarded rather
than recorded: it was taken immediately after `go test -race ./...`, and the
isolating measurement above says the scan is worth about five milliseconds,
not seventy. A quiet re-taking came back in band.

## What this iteration is evidence of

The item carried three numbers and every one was wrong — the budget, the walk
count, and the implied value of the scope limit. Two were wrong from memory
and one from an argument that had never been measured.

What made the decision easy was that all three corrections pointed the same
way. Both budgets being full turned "should this be its own walk" from a
judgement into a thing already decided in writing; and the quarter-coverage
figure turned "the incident was here" from a reason into a coincidence.

## Next

Sorted by **value**, highest first; age breaks ties. Age is how many saved
sessions ago the item was first raised, counted in `ai_docs/claude_sessions/`
and measured from this doc, so `age 0` means it was raised here.

1. **(age 0 · value medium) The copies walk is five questions under a name
   that describes one.** Written down rather than fixed, this session, because
   the alternative was renaming a test that `repositoryWalks` names, that six
   session docs cite, and whose file name is the walk's identity in every
   message it prints. The question is whether the walk is now
   `TestTheQuestionsTheOneRepositoryParseAnswers` and `copies_test.go` is
   where its biggest question lives — which is what the arrangement already
   is — or whether five is where a walk should stop collecting questions and
   the parse budget should be re-argued instead.
2. **(age 4 · value low) Every Next list in this loop was written by the
   session that would not work it.** Three iterations of evidence now, and
   they are not the same evidence: iteration 1's cold read found what an
   author cannot see, iteration 2 opened on a regression the previous
   iteration had introduced, and iteration 3 found that all three figures in
   the item it was working were wrong. The pattern is that a list is written
   with the confidence of the session that wrote it and read by one that has
   to check it.
3. **(declined, non-goal) A `t.Run` inside a loop counts as one question.**
   See `ai_docs/plans/non_goals.md`.
4. **(declined, non-goal) `packageLevelCallsTo` cannot tell an initializer
   from a stored function value.** See `ai_docs/plans/non_goals.md`.
5. **(declined, non-goal) The diff-and-print remainder stays unmeasured.**
   See `ai_docs/plans/non_goals.md`.
