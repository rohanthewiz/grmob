# Session: the loop, nine iterations about numbers that were not readings

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-11 · wraps the run that began with
"a-convention-already-being-followed-by-hand…" and ended with
"ten-commands-that-did-not-run…"

## Ask

`/loop 10` — work the Next list, ten iterations. Each opened with `/sl` and
closed with `/sess-wrap`. Nine did work; this is the tenth and it is the
wrap. Every iteration has its own doc, committed and pushed.

    10225f1   a convention already being followed by hand
    5f099ab   six figures under their floors
    bd0ad4e   a clock that measured a different thing
    27fe865   a set named for its first subject
    c965a78   the fall that was a spread
    52d6a00   a prediction refuted then un-refuted
    9846b19   three rules about ends
    e032b81   a lever put where it is read
    b12b1e6   ten commands that did not run

Twenty-one files, +3023 −183, three of them new Go files —
`quotedprose_test.go`, `band_test.go`, `wholefileband_test.go`.

---

## What was built

    a quoting convention     a token in backquotes is quoted, not claimed.
                             It replaced renamedTestsStillNamed outright,
                             and it was already being followed by hand —
                             of 4 backquoted range figures in Go prose, 3
                             were exactly the class it names
    a band verdict           the line that prints a reading now prints
                             where that reading fell in the band the record
                             carries. Three sites in internal/themehistory
    an in-process reading    wasm/verify had none of its own headline
                             figure, which is why it was the record that
                             drifted. A TestMain clocks m.Run()
    a census renamed to      checkTwoCopyDecls holds ten shapes identical
    what it covers           across the two packages, up from eight

## What it found

**Every band in this repository was stale.** Not one, not the one that
prompted the run — all of them, on the first run of the thing that checked:

    wholeFile        floor 2.88s,  read from 2.840s
    wholeRun         floor 1.56s,  read from 1.406s
    perObjectRun     floor 30.53s, read from 28.65s
    its three terms  409ms/0.87s/0.125s, read 396ms/835ms/124ms
    walkParse        floor 0.20s,  read 0.19s in 21 of 21 runs

Plus, outside the bands: two prose copies of an `ls-tree` count that grows,
a figure a hundredth out of step with its field, the last by-hand figure
copy on a record's re-taking list, a `func`-sample skip present in one code
path and not its twin, five of ten re-taking commands that did not run, and
one band field — added by this run — with no taking method at all.

## What was declined, with numbers

Four entries added to `ai_docs/plans/non_goals.md`:

    typed bands instead of a parser    the copy does not go away: two
                                       `package main` programs cannot import
                                       each other's tests
    a sentence naming a field must     corpus of 1 at sentence granularity,
    quote that field's range           0 defects. Empty by construction
    a GRMOB_ name is an env var        4 of 11 are Kotlin `const val`
    a command table line is runnable   2 lines, both real, both now fixed

## The five things this run actually learned

**1. A band's ends are readings, not choices.** Widen, almost never narrow.
An unreached end is evidence of nothing. A band that looks too wide is cheap
and a band that looks right is expensive.

That cost five re-takings to arrive at, and the proof is that the rule was
**broken two iterations after it was written, by the same hand, on the same
record**: `walkParse`'s `0.20–0.24s` was re-centred on `0.19–0.20s` after 21
runs found no reading above 0.20 — and was falsified five runs later by a
0.22s, then by a 0.24s. The original ceiling was right all along. Whoever
first wrote it had seen the tail; three sittings of re-measuring had not.

**2. Consecutive runs are correlated, and the fix for that is not more
runs.** Sixteen runs back to back are closer to one sample of a sitting than
to sixteen samples of a figure. But the sittings rule does not generalise
either — 21 runs of `walkParse` across three sittings gave two values. What
varies is a property of what is being timed, and the tail is found by luck
rather than by method.

**3. A control is only a control if it is tighter than the difference.**
Iteration 1 ruled the machine out by reading the untouched sibling record.
The sibling consulted was a whole-package figure with a 300ms band and the
difference being ruled out was 140ms. The answer came back "no difference"
because it could not have come back anything else.

**4. Measure the measurement.** Three scans produced numbers that were
artefacts, and each would have shipped a rule if believed: 25 "disagreements"
that were a comment group being forty lines long, 32 "unresolved" env names
that were a Go-only scan of a five-language repository, and 15 wall-clock
"findings" that were a crude regexp over tables the rule was never about.
Every one was caught by reading the list instead of the count.

**5. A verdict that fires wrongly is worse than no verdict.** Both band
verdicts reported OVER inside `go test ./...`, because `go test` runs package
binaries in parallel and the bands are of a package alone. On a path that
runs in every session's verification. They are opt-in now.

## What the run is evidence of, beyond the code

Every iteration's Next list was written by the iteration that would not work
it, and **nine for nine the instruction paid and the conclusion did not
survive**:

    iteration 2   "a marker makes both rules writable" — it made one
    iteration 4   "typed bands remove the copy" — they duplicate it
    iteration 5   "a reading falls over a session" — it is a spread
    iteration 6   both branches named in advance, and it got BOTH
    iteration 8   named a file that could not host its own suggestion
    iteration 9   "the tables are already covered" — a prefix resolves by
                  design, so an elided command is invisible

The instruction that paid every time was the same one as the last run:
**measure before writing, and read the findings rather than counting them.**

And a smaller one, new this run: when the evidence changes, keep the wrong
account beside the right one. Three records now carry both — what it looked
like, and what it was.

---

## Verification

Eleven paths, green at the end of every one of the nine iterations:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      rc 0
    GOOS=js GOARCH=wasm go build ./...

Plus, this iteration, every command in both re-taking tables run as written —
ten of ten.

**Break-tests across the run: thirty-one, every one read rather than
counted.** Four of them found defects the code review had not: `by 0s`
against a microsecond band, the same rounding wrong again from the other
side, `bandVerdictEnv` undefined under `-race`, and the census catching a
`NumCPU` reader the cores note did not name.

**Figures at the end**, both packages in band:

    wasm/verify             2.868–2.960s   recorded 2.78–2.97s
    …wholeFileInProcess     2.769s         recorded 2.55–2.78s
    internal/themehistory   3.066–3.170s   recorded 2.86–3.22s
    …wholeRun, …batchRetire in band

**Found in this wrap, in the run's own session docs**: one filename missing
its date component entirely, two more whose timestamps misordered the run
against the commits that recorded them, and two `Previous:` references
pointing at files that did not exist. Renamed to the commit times; the chain
now resolves end to end. The run spent nine iterations on figures that had
stopped matching their records and left four of the same defect in its own
filing.

## Next

1. **(age 1 · value medium) Every band field should be held to having a
   taking command.** Measured at 1 real of 10 on its first run — the field
   this run itself added. Fields off the AST, labels off the record's doc
   comment, one shape and one meaning, and `checkTimingsRecordCopies`
   already visits both records on the shared parse so it needs no second
   copy. The cost is what any new question on that walk costs: a
   `walkQuestionBudget` raise with its argument, and the registrations three
   censuses will ask for.
2. **(age 4 · value low) Record which band ends have actually been
   reached.** Follows directly from rule 1 above: if an end is always a
   reading, a field whose ends are not readings is the thing to find.
3. **(age 8 · value low) `…TimingsTakenOn` is not the only record shape.**
   Value has fallen twice: it exists to unblock the wall-clock rule, which
   was measured at zero real defects in its residue.
4. **(age 21 · value low) Every Next list in this loop was written by the
   session that would not work it.** Nineteen iterations of evidence across
   two runs now. The useful form has not changed: an item's instruction is
   worth more than its conclusion, and the conclusion is worth writing down
   anyway because it can be checked.
5. **(declined, non-goal)** Ten entries now. See
   `ai_docs/plans/non_goals.md`. (Said "nine" when written; the file held ten
   at that commit. Corrected by the session that found the same count one low
   in its own doc.)
