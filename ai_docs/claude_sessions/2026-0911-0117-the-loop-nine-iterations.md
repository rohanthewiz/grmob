# Session: the loop, nine iterations

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-11 · wraps the run that began with
"a-figure-in-prose-is-a-copy…" and ended with
"a-rule-that-caught-none-of-what-it-was-written-for…"

## Ask

`/loop` — work the Next list, five iterations, raised to nine mid-run. Each
iteration opened with `/sl` and closed with `/sess-wrap`. This is the wrap of
the run rather than of an iteration; every iteration has its own doc and every
one is committed and pushed.

    9de1a01   a figure in prose is a copy
    d1969d2   a row that said three
    fed41aa   a quarter of what the rule was about
    8b6d39e   a name that had stopped being true
    ea187ce   four comments pointing at tests that were gone
    e6fbbc7   a rule declined on fifteen findings
    eb382b9   a figure attributed to a record that moved
    8680cff   three copies of two numbers
    4f6b7a2   a rule that caught none of what it was written for

Thirty files, +3933 −328, four of them new — `sharedparse_test.go`,
`prosefigures_test.go`, `commenttext_test.go`, `prosenames_test.go`.

---

## What was built

Three arms, all riding one repository-wide parse that already existed:

    the file counts in prose   every sentence pricing something against how
                               many Go files the tree holds, held to the count
                               the walk just made. Fired four times during the
                               run, each time on a commit that added a file —
                               including the commits that added these arms
    two comment-text rules     a line that is one comment written twice, and a
                               tab anywhere but the leading indent. Zero
                               exemptions repository-wide on the day written
    a test named in prose      held to being a test this repository has, in Go
                               comments, string constants, and every tracked
                               file that is not Go — docs, scripts, a .mjs

And three constants that make a decision visible instead of a default:
`walkQuestionBudget` (what stops one walk absorbing every question),
`coresNoteScanDirs`, and the per-question exemptions that replaced a
walk-level skip.

## What was found

**Sixteen things that were wrong before the run started**, in seven places:

    7 test names pointing at nothing   components/app_bar_test.go,
                                       core/layout.go, themenearmiss_test.go,
                                       gomobilestub_test.go, a hook script,
                                       docs/platforms/wasm.md, browser.mjs
    9 stale or duplicated figures      the parse depth in three places, "a
                                       2.7-second package", "eight of
                                       thirty-seven", main.go's table,
                                       main_test.go's pair, its wholeRun
                                       restatement, perObjectRun's own doc,
                                       and 0.18–0.30ms against 0.18–0.29ms

**And three the run introduced and caught itself**: a row that said three
questions when the walk had four, five references to a test the run had just
renamed, and two `%s` in the wrong order — that last one invisible to `go vet`,
because both were strings, and to every green run, because a failure message
only renders when the check fails.

## What was declined

Three rules, each on its own numbers, each written up in
`ai_docs/plans/non_goals.md` with what would change it:

    a doubled word                     0 real of 9
    a backquoted name is a declaration 0 real of 15
    a wall clock lives in a record     1 real of 6

Plus two limits carried for two sessions each, moved there with their
arguments: a `t.Run` inside a loop counted as one question, and
`packageLevelCallsTo` not telling an initializer from a stored function value.

## The five measurements, which are the actual result

    a test named in prose resolves          4 real of 6      built
    the two comment-text rules              0 of 0, zero exemptions   built
    a doubled word                          0 real of 9      declined
    a backquoted name is a declaration      0 real of 15     declined
    a wall clock lives in a record          1 real of 6      declined

The two built share a property the three declined do not: **the shape belongs
to one language and one meaning.** `Test` followed by a capital is a Go test
and nothing else — not Swift, not Kotlin, not CSS, not English. A comment line
repeated on itself is a botched edit and nothing else. Whereas a doubled word
is English, a backquoted identifier is any of four languages this repository
holds in agreement, and a wall clock is either a reading or a duration the code
performs.

Measuring first cost about an hour across nine iterations and prevented three
checks that would each have arrived with an exemption table.

## The records

Both were re-taken, and the second one twice over.

`verifyTimingsTakenOn.wholeFile` went 2.78–2.93s → 2.88–2.97s, and the core
table with it, which is that record's own stated rule. The machine was ruled
out by reading the untouched sibling at the same moment: a slow afternoon moves
both records, and only one moved.

`themehistoryTimingsTakenOn` had three copies of two numbers in two other
files, two of them wrong and all three attributed. And a second fault the
other record does not have: `2906 objects` and `88 ls-tree` are readings of
this repository's own history, which grows — they were 2955 and 89 by the time
the paragraph describing them was written, in the same session.

The generalisation both records now carry: **a figure quoted in two places is a
copy, and attributing the copy does not make it one thing.** Moving it into the
original does — which is why three walk-depth figures are fields now, and two
failure messages read them.

## What the run is evidence of, beyond the code

Every iteration's Next list was written by the iteration that would not work
it, and the item was wrong in a specific way each time:

    iteration 3   all three of the item's numbers were wrong, and the
                  corrections all pointed the same way
    iteration 4   the item posed "rename OR cap" and the answer was both
    iteration 6   "measure first" was right and the conclusion was opposite to
                  the last time that instruction was followed
    iteration 7   named four figures as stale; two were fine
    iteration 9   said nothing about the escape clause that decided the rule

None of that is an argument against writing the list. It is an argument for
the thing this repository already does everywhere else: the list is a claim,
and a claim is worth having because it can be checked.

---

## Verification

Eleven paths, green at the end of every one of the nine iterations:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      rc 0
    GOOS=js GOARCH=wasm go build ./...

Fifteen break-tests across the run, every one read rather than counted — a
failure message is code that runs exactly when nobody is watching a green run.

**Figures at the end.** Both packages in band:

    wasm/verify             2.866–2.946s   recorded 2.88–2.97s
    internal/themehistory   2.999–3.053s   recorded 2.92–3.22s

## Next

Sorted by **value**, highest first; age breaks ties. Age is how many saved
sessions ago the item was first raised, counted in `ai_docs/claude_sessions/`
and measured from this doc, so `age 0` means it was raised here.

1. **(age 1 · value medium) There is no way to mark a figure as a quotation of
   a past reading.** Two declined rules failed on the same gap: the wall-clock
   rule cannot tell "30.14–30.42s, which is what this said before" from a live
   copy, and the test-name rule needed `renamedTestsStillNamed` as a table for
   exactly the same reason. A marker — a convention, not a mechanism — makes
   both writable, and the second could then drop its table. The corpus is
   small: two table entries and about four sentences. Worth doing when
   somebody is editing this prose anyway rather than as its own errand.
2. **(age 5 · value low) `renamedTestsStillNamed` is a graveyard with no
   pruning.** Two entries, both current. An entry whose sentence has since been
   rewritten is dead weight reading as a guard, which is the failure
   `citationExempt`'s own assertion catches one file over. Subsumed by item 1
   if that is ever done.
3. **(age 11 · value low) Every Next list in this loop was written by the
   session that would not work it.** Nine iterations of evidence, summarised
   above. The useful form of the item is not "stop writing lists" — it is that
   a list item's CONCLUSION is worth less than its instruction, and the
   instruction that paid every time was "measure before writing".
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
