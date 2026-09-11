# Session: a rule declined on fifteen findings, three stale names outside Go, and a band that had to be re-taken

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-11 (follows "four-comments-pointing-at-tests-that-were-gone-a-prefix-that-still-runs-and-a-budget-raised-the-session-after-it-was-written")

## Ask

`/loop` iteration 6 of 9. The list's top item was the generalisation of the
test-name rule: the same check for helpers, recognised by backquotes, with the
instruction to measure the residue first and the number to beat stated — four
real defects in six findings.

| item | value | outcome |
|---|---|---|
| 1 | medium | measured and **declined**; moved to non-goals with the numbers |
| — | — | the measurement's own gap: three stale names outside Go, fixed and armed |
| — | — | `verifyTimingsTakenOn` re-taken, and the core table with it |

---

## The rule, declined on its own numbers

Over every Go comment outside `ai_docs`, taking each backquoted span that is a
single Go identifier:

    896   mentions
    786   unresolved against package-level declarations
    334   adding methods, struct fields and interface methods
    304   adding every imported package name
    205   adding every local variable and parameter in the tree

Narrowing to lowerCamelCase with at least one hump — which cuts the
all-lowercase vocabulary in one move — gets it to 69 mentions and **15
unresolved**. Every one was read, and **none is a defect**:

    accessibilityIdentifier, accessibilityValue,
    simultaneousGesture, measureWithoutPlacing      Swift and Compose
    getOrNull                                       Kotlin's stdlib
    compileDebugKotlin, fetchComposeLayoutSources   Gradle tasks
    flexShrink, shrinkFactor                        CSS
    measureText                                     a canvas API
    shrinkPinned                                    a native property, read out
                                                    of generated source with
                                                    strings.Contains
    inkCanaryAgreement                              a JavaScript function in
                                                    browser.mjs
    dottedVersionParsers                            "this used to be…"
    enumWorkersPool                                 a hypothetical in a
                                                    sentence about a failure

**The backquote convention here does not mean "a Go declaration".** It means
"a literal token of some language" — and this repository's subject is the
agreement between four of them, plus CSS and ARIA vocabularies and a build
system's task names. The exemption table would be a list of other languages'
identifiers, maintained in Go, to keep a Go rule quiet. That shape is the tell.

Why the test-name rule was not the same bet: `Test` followed by an upper-case
letter is a shape only a Go test has. Nothing in Swift, Kotlin, JavaScript,
CSS or ARIA is spelled that way, and nothing in English is. A lowerCamelCase
identifier is a shape every language here has, which is exactly why the
residue is what it is.

In `ai_docs/plans/non_goals.md`, with the numbers and what would change it.

---

## The gap the measurement exposed

The previous session's scan had also listed five Test-shaped names in tracked
files that are not Go. Three were stale, in three different kinds of file:

    .claude/hooks/session-doc-check.sh   TestEveryGitListingInAScriptAsksForZ
                                         → …AsksForNulSeparatedPaths
    docs/platforms/wasm.md               TestNoRoleCollidesWithTheTabPanelWiring
                                         → TestAPageThatRolesItselfATabPanel
                                           IsStillNotWired
    wasm/verify/browser.mjs              TestTheNativesFixedSizeArmsAreTheOnes
                                         TheCensusDescribes → the two tests it
                                         became, in fixedsize_test.go

So the rule now reads them. Those files are read rather than parsed — a test
name in a shell comment is a word in a file — and the cost is 136 files and
2.5 MB, **measured at 0.003s**, which is why this is a read here rather than a
change to what `citingFiles` hands back to its six callers. Isolated by taking
the scan out and putting it back, it is not distinguishable from noise.

**One shape is skipped: a name written as `func TestX(`.** The documentation
contains example tests a reader is meant to write — `func TestCounter(t
*testing.T)` in the getting-started page, `func TestNoHookDrift` in the
debug-mode one — and those are declarations in a sample rather than citations.
A citation in prose is never spelled with `func` in front of it. Break-tested
both ways.

Mentions went from 337 to 441.

---

## The band had moved, and the machine was ruled out

The package read 2.876–2.958s against a recorded 2.78–2.93s. Each of this
session's additions had measured at between nothing and ten milliseconds, so
the first suspicion was the machine, which has been running tests for hours.

**The other record settles it.** `internal/themehistory` records 2.92–3.22s
and came back at 3.01–3.16s at the same moment, on the same machine, against a
package this session has not touched — the middle of its own range. A slow
machine moves both. This moved one.

So `wholeFile` is **2.88–2.97s over twenty-one runs**, the same sample size as
the figure it replaces. What moved it: three questions arrived on the shared
parse, and a per-question exemption replaced a walk-level skip, so a file that
used to be invisible is now parsed and read by every rule.

### And the core-count table with it, which is the record's own rule

> Widening `wholeFile` and leaving these four would have made a record where
> the figure at the top was from one afternoon and the table explaining it was
> from another, with nothing on either saying which.

    GOMAXPROCS    was           now
    1             3.08–3.27s    3.35–3.49s
    2             2.78–2.88s    2.92–3.00s
    4             2.74–2.86s    2.87–2.91s
    8             2.75–2.85s    2.88–2.92s

Every row moved by about the same amount, which is the useful half of
re-taking a whole table: the added work is single-threaded, so it lands on
every core count equally and none of the movement is about concurrency.

**This taking replaces the ranges rather than widening them**, which departs
from how the three takings before it were recorded and is written down as a
departure. Those three each held every taking's range together because each
was of the same program — the row above had not moved. This one is not:
unioning would give a one-core figure of 3.08–3.49s, most of which no version
of this package has ever taken, and a range that wide hides a difference.

The shape is unchanged and its size is not: one core is about a sixth dearer
than four rather than a tenth, because the serial half of the package grew.
`coresAttribution` says so, and still names the same two declarations.

---

## Verification

Eleven paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      rc 0
    GOOS=js GOARCH=wasm go build ./...

**Three break-tests:**

    a stale name in a doc     `TestSomeCheckThatWasDeleted` in wasm.md and
    and in a shell script     `TestAnotherOneLongGone` in the hook. Both named,
                              with file, line and "in text"
    a doc sample              `func TestSomeExampleAReaderWouldWrite` added to
                              getting-started.md. No finding, which is the skip
                              working — and `TestNoHookDrift` proves it is the
                              skip and not prefix resolution, since nothing
                              declares a test beginning that way

**Figures.** Twenty-one runs at 2.879–2.971s; the table at three runs a row.

## What this iteration is evidence of

The item said measure first and the answer was don't build it, which is the
same instruction producing the opposite outcome from last time. The two
measurements were the same shape and differed only in the number: four real of
six, against zero real of fifteen.

The more useful thing is what the measurement found while failing. Looking at
205 unresolved names to decide whether a rule was worth writing is also
looking at 205 names, and three of them — in a different corpus entirely —
were the defects the session actually fixed.

## Next

Sorted by **value**, highest first; age breaks ties. Age is how many saved
sessions ago the item was first raised, counted in `ai_docs/claude_sessions/`
and measured from this doc, so `age 0` means it was raised here.

1. **(age 0 · value medium) Every inline figure in this package's prose is
   still narrower than the truth, and now some are stale by construction.**
   `wholeFile` says so about itself — three or seven readings where sixteen
   were needed — and the same is true of `0.09s`, `0.16s`, `0.18s` and the
   0.010s in the walk table, none of which was re-taken when the package's
   cost moved by five per cent tonight. They are not wrong in a way that
   misleads about a decision, which is why nothing was done about them under a
   re-taking that was already three tables deep. The decision is whether a
   record that re-takes its own rows on a rule should have a rule for the
   figures that explain those rows.
2. **(age 1 · value low) `renamedTestsStillNamed` is a graveyard with no
   pruning.** Two entries, both current. An entry whose sentence has since
   been rewritten is dead weight reading as a guard, which is the failure
   `citationExempt`'s own assertion catches one file over. Not worth an arm at
   two; written down in the table's doc.
3. **(age 7 · value low) Every Next list in this loop was written by the
   session that would not work it.** Six iterations. The new evidence is that
   the item's own instruction can be right and its conclusion wrong: "measure
   before writing" was correct both times and pointed opposite ways.
4. **(declined, non-goal) A backquoted name in prose is not held to being a
   declaration.** See `ai_docs/plans/non_goals.md`.
5. **(declined, non-goal) A `t.Run` inside a loop counts as one question.**
   See `ai_docs/plans/non_goals.md`.
6. **(declined, non-goal) `packageLevelCallsTo` cannot tell an initializer
   from a stored function value.** See `ai_docs/plans/non_goals.md`.
7. **(declined, non-goal) The diff-and-print remainder stays unmeasured.**
   See `ai_docs/plans/non_goals.md`.
