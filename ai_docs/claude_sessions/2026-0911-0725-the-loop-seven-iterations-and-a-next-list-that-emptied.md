# Session: the loop, seven iterations and a Next list that emptied

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-11 · wraps the run that began with
"every-band-field-holds-its-taking-command" and ended with
"the-third-verdict-nothing-held-and-a-map-that-cannot-go-stale"

## Ask

`/loop` — work the Next list, nine iterations or until the list is empty. Each
opened with `/sl` and closed with `/sess-wrap`. **Seven did work and then the
list was empty**, which is the stopping condition rather than the count. Every
iteration has its own doc, committed and pushed.

    d5f67bc   every band field holds the command it was taken by
    26f3451   a band says three things
    f7c6c68   the figure go test prints is placed in its band
    173d694   four levers nothing compared
    92d98b0   the two-copy question asked the other way round
    af3f529   a sentence saying two numbers are one number
    e8ce1a0   the third verdict nothing held

Seventeen files, +3334 −221, across eight Go files in two packages. No new Go
files: every change landed in the band and record machinery that already
existed.

---

## What was built

    a taking-command census   every band field held to having, in its record's
                              own doc comment, the command it was taken by.
                              Both directions, plus an entry naming a field
                              whose value does not open with a band
    a band's third property   recordedBand returns the PRECISION its ends are
                              written at. An end is a reading rounded outward
                              to two decimals, so "does this reading reach the
                              floor" has an exact answer at that precision and
                              none without it
    bandPlacement             where in the band a reading fell, how wide the
                              band is, and which end — if either — the reading
                              is evidence FOR, carrying the rule that an
                              unreached end is not a reason to move one
    againstBandGiven          the verdict sentence as a function of four
                              arguments. The form that can be ASSERTED, and the
                              one all three verdicts now build
    a placement arm           the figure `go test` itself prints — which no test
                              can see — handed back through
                              GRMOB_PACKAGE_READING to the code that knows the
                              band. One per package, two lines in each taking
                              table
    const as a held shape     the two-copy census read `token.VAR` only, so
                              registering a constant reported it MISSING. Three
                              GRMOB_ levers and their reader are held now
    the question inverted     every name both record packages declare must be
                              in a two-copy list or in twoCopyNamesThatDiffer
                              with its reason, and the exemptions held three
                              ways
    a map that is a log line  the census names the sixteen shapes it holds and
                              the five it excuses, built out of the lists, so
                              there is no paragraph to go stale

Two new assertions, which are the first things in this machinery that could be
asserted at all: seven placements and six verdict sentences, over four durations
each rather than over a clock.

## What it found

**Three false band verdicts, from the machinery's own inconsistency.** Both
records reported UNDER on readings of 2.755s and 2.816s against floors of 2.76s
and 2.82s. Both were wrong: at the precision those bands are written to, those
readings ARE the floors. `bandPlacement` already judged "reaches an end" that
way and the in-band decision did not, so one sentence could call a reading
outside a band and, in its next clause, at its floor. Fixed in the two copies a
census holds — and found two iterations later still living in the third, which
nothing held in step.

**Two stale floors, reported rather than noticed.**

    verifyTimingsTakenOn.wholeFile         2.78 → 2.76, on 2.762s and 2.765s
    themehistoryTimingsTakenOn.wholePackage  2.86 → 2.82, on 2.824s and 2.843s

Nothing was made faster; both packages gained tests in the same session. The
three previous widenings of the first were each found by a person running the
command a dozen times and comparing two ranges by eye. These were found by an
arm that was an hour old.

**Four levers nothing compared.** `bandVerdictEnv`, `bandVerdictAsked`,
`bandVerdictWanted`, `packageReadingEnv` — identical in both packages, held by
nothing, three of them for several sessions with the argument for holding them
already written down twice. The failure they allowed is silence: a misspelled
`GRMOB_BAND_VERDICT` in one package compiles, passes, and leaves a reader
following the documented command with a verdict from one record and nothing from
the other.

**Three stale width copies, all derivable from numbers in the same file.**
`3% wide` of a band that was 6.8%, `300ms wide` of one that was 360ms, and
`130ms wide against 130ms` for a comparison that had reversed. Then the sentence
that corrected the first went stale one iteration later, when the floor moved
again — so the number is gone rather than corrected twice.

**Plus:** a band reader behind `//go:build !race` that a census needed (the
second instance of that exact fault); the four machine comparisons written out
twice in one package; two more copies of a band, stale at a value from before
its first widening, one of them in the table that exists to price the `-short`
lever; a numerator counted over a different set from its denominator; and a
count in two consecutive session docs that was one low.

## What was declined, with numbers

Five entries added to `ai_docs/plans/non_goals.md`, which now holds fifteen:

    a declaration a census reads must     the Go toolchain is that arm and it
    live outside a build tag              is already in the verification path
    a band field carries a "reached"      it would be a hand-kept claim with
    mark                                  nothing able to check it; the verdict
                                          says it per reading instead
    a prose width must quote its range    3 of 3 on defects, 0 of 2 false
                                          positives — declined because the
                                          rule's own explanation breaks it
    a constant whose doc names another    1 defect in 10 candidates, and the
    is defined as it                      distinction needs reading English
    the other records grow this           not one wall clock in either; both
    machinery                             are re-derived by the run that reads
                                          them

## The four things this run learned

**1. A verdict's precision has to be the record's, everywhere.** Three
implementations of one decision, two held identical by a census and one not, and
the unheld one was still wrong two iterations after the other two were fixed.
The fix that lasts is not the correction — it is that the third one now supplies
only the local half and the decision is made in one place.

**2. An arm an hour old beats three sessions of looking.** Two floors had been
wrong for an unknown number of sittings. Both were found on the first sitting
after something compared the numbers. The three earlier widenings of one of them
were each found by eye, and each was set from too few sittings again.

**3. An inclusion list's failure mode is silence, and it cannot be inverted
away.** 4 of 21 shared declarations were identical and unregistered. But only an
inclusion list can say the whole SET is present in both packages — so the
inversion is an addition, not a replacement, and both lists stay.

**4. A figure that is derivable does not belong in prose at all.** Every width
copy in this run was stale, including the one written to explain that width
copies go stale. The durable form was to print it beside the reading and take
the number out of the sentence — and the same move works for a census's own
inventory, where a log line built from the list replaced the paragraph nobody
could keep true.

## What the run is evidence of, beyond the code

Every iteration's Next list was written by the iteration that would not work it,
and **seven for seven the instruction paid and the conclusion did not survive**:

    iteration 1   "the cost is a budget raise and three registrations" — it was
                  neither: no raise, no registrations
    iteration 2   "the cost has dropped, the walk is built" — the walk was
                  never the expensive half
    iteration 3   "the honest form may be a verdict the verify SCRIPT prints" —
                  no script runs either package's tests
    iteration 4   the most accurate item of the run, and it still missed the
                  rename and the defect the iteration spent its last hour on
    iteration 5   proposed replacing a list that cannot be replaced
    iteration 6   the first item whose MEASUREMENT was wrong rather than its
                  conclusion: two of its three constants were already held
    iteration 7   both items closed, and the second exposed a defect neither
                  list had named

Three of this run's defects were found by taking the closing figures rather than
by any item on any list. That is the step no Next item has ever had to name, and
it is where the next session should start if it wants to find something.

---

## Verification

Eleven paths, green at the end of every one of the seven iterations:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      rc 0
    GOOS=js GOARCH=wasm go build ./...

**Break-tests across the run: thirty-two, every one read rather than counted.**
Four found things the design had not: findings that quoted a 400-character field
value instead of its band, a message that said the same fact twice, a numerator
over the wrong set, and `main` — an exempted name every `package main` declares,
which made a new census report the whole repository.

**Two process failures, recorded rather than glossed.** `git checkout --` on a
file carrying uncommitted work destroyed an iteration's changes mid-break-test;
and a scratch copy taken two iterations earlier silently reverted 39 lines of a
committed fix. The habit that survives both: take the copy in the same command
as the break, and read `git status` before trusting a restore.

**Figures at the end**, both packages alone, every one in band:

    wasm/verify wholeFile         2.818–2.954s   recorded 2.76–2.97s
    …wholeFileInProcess           2.63–2.68s     recorded 2.55–2.78s
    internal/themehistory         2.877–3.052s   recorded 2.82–3.22s
    …wholeRun, …batchRetire       in band, placement printed

## Next

1. **(age 28 · value low, and this is the last of it) Every Next list in this
   loop was written by the session that would not work it.** Twenty-six
   iterations across two runs, and the tally has stopped changing: every
   instruction paid, no conclusion survived intact. This run added three new
   failure shapes to the collection — an item whose measurement was wrong rather
   than its conclusion, an item that proposed replacing something that cannot be
   replaced, and an item that named the cheap half as the blocker. The
   instruction that paid in all seven is the same one it has always been:
   **measure before writing, and read the findings rather than counting them.**
   Worth stopping rather than extending: it is now an observation with no
   remaining prediction to test.
2. **(declined, non-goal)** Fifteen entries. See `ai_docs/plans/non_goals.md`.

**There is no open work item.** The loop stopped on an empty list rather than on
its iteration count. Three of this run's defects were found by taking the
closing figures, so a session with nothing on the list still has one honest move:
run both packages alone, place both headline figures, and read what the verdict
says.
