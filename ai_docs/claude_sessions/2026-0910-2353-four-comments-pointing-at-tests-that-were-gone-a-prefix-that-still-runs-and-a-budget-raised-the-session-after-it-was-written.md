# Session: four comments pointing at tests that were gone, a prefix that still runs, and a budget raised the session after it was written

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-10 (follows "a-name-that-had-stopped-being-true-a-reason-that-always-wins-and-five-references-to-a-test-that-was-gone")

## Ask

`/loop` iteration 5 of 9. The list's top item was the one the previous
iteration created: a rename had left five sentences naming a function that no
longer existed, through eleven green verification paths. The item said to
measure the false positive rate before writing anything, because that is what
killed the doubled-word rule three iterations ago.

| item | value | outcome |
|---|---|---|
| 1 | high | measured, written, and it found four real defects on the way in |

---

## The measurement, in four passes

The general rule — an identifier in prose resolves to something declared —
was never written. Prose is full of identifier-shaped words that are not
declarations, and of qualified names from the standard library that this
repository does not declare and should not have to.

A Go TEST name is the narrow case: `Test` followed by an upper-case letter is
a shape nothing in English has. Measured over every Go comment and string
constant outside `ai_docs`:

    336 mentions
     25 unresolved   raw
     20 unresolved   after joining each comment group and un-hyphenating the
                     line-splits — `TestEveryGitListingAsksForNul-` / `Separated
                     Paths` is one name to a reader and two to a line scanner
      6 unresolved   after accepting a PREFIX

**Four of the six were real**, in four packages, none of them the package
being worked on:

    TestGroupHeaderLabelIsASecondLevelHeading   the test is `…IsAHeading`
    TestNativeBoxIsAVerticalStackNotAnOverlay   renamed to `TestNativeContainers
                                                StackTheirChildrenAndDoNotOverlay`
    TestNoFloatThisFileComparesIsDerivedTwice   `TestNoFloatAComparisonRestsOn…`
    TestTheGobindPinIsTheOneInGoMod             `TestTheGobindReadingsArePinned…`

Every one is a sentence saying "see TestX for the other half" — an
instruction, pointing nowhere. The other two are deliberate: records that tell
the history of a rename, which is a sentence ABOUT the name having changed.

That is the opposite result from the doubled-word rule, which scored nine hits
and no defects and was declined on the number. Same method, opposite answer.

## Why a prefix resolves, which is not a concession

`go test -run X` matches every test whose name begins with X. So a prefix
mention still takes a reader to the test — which is what a test name in a
comment is FOR, and this repository writes `-run` command lines into comments
for exactly that use.

It also makes a name wrapped across two lines with NO hyphen readable without
reassembling prose nobody hyphenated: the fragment is a prefix of the whole.
That single rule took the residue from 20 to 6, and 14 of those 14 were wraps
or `-run` prefixes.

What it costs: renaming `TestFoo` to `TestFooAndBar` leaves every mention of
`TestFoo` resolving. That is the right answer rather than a missed one —
those mentions still run the test they name.

## The hyphen join went into `prose`

Rather than a second joiner. `prose.add` now treats a piece beginning where
the last ended in `-` as a word split: hyphen dropped, no space. Measured
across every Go comment in the repository — **twelve lines end in a hyphen and
every one of them is a word split across lines.** No counterexample to exempt.

The rule is applied at a PIECE boundary and not between words, because within
a line a trailing `-` is punctuation somebody typed.

---

## The budget fired one session after it was written

`walkQuestionBudget` was set to five in the previous iteration, with a doc
naming the two honest answers for the sixth question. The sixth question
arrived immediately, which is the most useful thing that could have happened
to it: the argument had to be made in writing rather than assumed.

    it needs both halves of one parse   what tests EXIST is the declarations,
                                        what prose POINTS AT is the comments,
                                        and no other walk has both
    a walk of its own is unavailable    repositoryWalkBudget is 7 against 7
    it is not the comment question      that one is two rules about characters
                                        in a line; this one's reaching arm is
                                        that the DECLARATIONS were found,
                                        which is a different and much louder
                                        failure

The last row is the one that decided it. Folding this into the comment
question would have kept the number at five — and a number kept at five by
arranging the questions to suit it is worth nothing at all.

## The check caught its own header, twice

The wiring comment said, in writing: *"No exemption: this question has nothing
to say about itself."* It had seven things to say about itself — the four dead
names quoted as evidence, and `TestSomething` / `TestFooAndBar` used to
explain the prefix rule.

So `proseNameRulesFile` joins `commentRulesFile` and `proseFigureRulesFile`.
Three of the six questions now carry a self-exemption and it is the same shape
every time: **a rule worth writing down is worth showing an example of, and an
example of a rule is a thing the rule catches.** Noted in the exemption list,
with the point at which it should become a convention rather than a list.

This one is narrower than the other two — the file declares no test, so what
is exempt is its PROSE and not the file, and the other five questions read it
normally.

---

## The cost, which took two measurements to get right

First reading after wiring it in: **2.914–3.001s**, eleven of twelve above the
recorded band. Not noise this time.

    0.07s   running the compiled pattern over every comment line — 2.5% of
            the package, for a question with 337 answers
    0.01s   with `strings.Contains(s, "Test")` in front of it

The pattern begins with the literal `Test`, so the substring test is the
necessary condition rather than an approximation of it: no false negative is
possible. The string-literal half was narrowed the same way, to one literal at
a time rather than the `+` chain — which also makes the line attribution exact
and gives up only a name split across a concatenation, the limit the figures
rule states one file over.

**Figures.** Package **2.830–2.913s** over twelve, inside the recorded
2.78–2.93s. Two earlier takings were discarded rather than recorded: one at
2.914–3.001 which was the unfiltered scan and a real cost, and one at
2.829–3.065 taken straight after it, whose top three runs were the machine
rather than the code. The rule that keeps working: take the reading quiet.

## Verification

Eleven paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      rc 0
    GOOS=js GOARCH=wasm go build ./...

**Three break-tests:**

    a stale name elsewhere    core/layout.go pointed back at the dead name.
                              Named, with its file, line and kind
    a prefix                  the same sentence cut to `TestNativeContainers
                              Stack`, which is a prefix of a real test. No
                              finding, which is the rule working
    walkQuestionBudget = 5    "answers 6 question(s) and the budget is 5"

## What this iteration is evidence of

The item said measure first, and measuring is what produced the rule's actual
shape. The prefix rule was not in the plan — it came out of looking at what
the 20 residual names WERE, and it turned an unusable check into a precise
one. Neither was the hyphen join, nor the `Contains` gate that made it
affordable.

And the thing the measurement bought that nothing else could: four defects
found before the rule was written, which is what made it obvious the rule was
worth the exemption it needed.

## Next

Sorted by **value**, highest first; age breaks ties. Age is how many saved
sessions ago the item was first raised, counted in `ai_docs/claude_sessions/`
and measured from this doc, so `age 0` means it was raised here.

1. **(age 0 · value medium) The same rule for helpers, and what it would
   cost.** `checkCitationsResolve`, `priceCalls`, `subtestSitesIn` — this
   repository names its unexported helpers in prose constantly, and nothing
   holds those either. The reason this session did not do it is the one that
   stopped the general rule: a helper name is lowerCamel and so is most of
   English inside backquotes. The measurable version is: a backquoted word
   that is a single identifier, matched against every declaration in the
   repository including methods and fields. Measure the residue before
   writing it — the number to beat is four real defects in six findings.
2. **(age 0 · value low) `renamedTestsStillNamed` is a graveyard with no
   pruning.** Two entries, both current. An entry whose sentence has since
   been rewritten is dead weight reading as a guard, which is the failure
   `citationExempt`'s own assertion catches one file over — the same check
   would work here and is not worth writing at two entries. Written down in
   the table's doc rather than left to be noticed.
3. **(age 6 · value low) Every Next list in this loop was written by the
   session that would not work it.** Five iterations. This one is the first
   where the item's own instruction — measure before writing — was the thing
   that made the iteration work, which is the pattern arguing against itself.
4. **(declined, non-goal) A `t.Run` inside a loop counts as one question.**
   See `ai_docs/plans/non_goals.md`.
5. **(declined, non-goal) `packageLevelCallsTo` cannot tell an initializer
   from a stored function value.** See `ai_docs/plans/non_goals.md`.
6. **(declined, non-goal) The diff-and-print remainder stays unmeasured.**
   See `ai_docs/plans/non_goals.md`.
