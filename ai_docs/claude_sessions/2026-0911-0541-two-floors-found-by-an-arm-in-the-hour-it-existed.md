# Session: two floors found by an arm in the hour it existed, and a width copy that went stale in one iteration

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-11 · Previous:
`2026-0911-0520-a-band-says-three-things-and-two-floors-that-turned-out-to-be-readings.md`

## Ask

`/loop` — work the Next list. **Iteration 3**, working **Next item 1**:
`wholeFile` and `wholePackage` have no verdict of their own.

## What was built

Each record's headline figure — what `go test` itself prints — is now placed in
its band by the code that knows the band.

    TestTheFigureGoTestPrintedForThisPackageIsPlacedInItsBand
        one per package, gated on GRMOB_PACKAGE_READING holding the figure
        `go test` reported. Two commands, which is what the taking table now
        says: run the package, then hand its own wall clock back

    againstBandGiven
        againstBand split in two. The gate and the machine comparison are the
        caller's; what is left is a function of four arguments returning a
        sentence — which is the form that can be ASSERTED. Held identical
        across both packages, twelve two-copy shapes now

    recordMachineDiffers, in wasm/verify
        the four machine comparisons were written out TWICE in that package —
        in the reporting arm and in the whole-file verdict's guard, with the
        same two subtleties explained in different words. A third caller made
        it a function, which is what internal/themehistory did a run earlier
        for the same reason. The cores census caught the new reader and asked
        for the note to name it, which it now does

    TestTheSentenceAroundAPlacementSaysWhichOfTheFourCasesItIs
        five cases: in band, under, over, a field written the wrong way round,
        and a reading from another machine. The last two are the ones worth
        holding — both return early and a regression in either reads as a
        verdict that passed

## Why the reading comes from the environment, measured first

The obvious design is an opt-in arm that runs `go test` itself and reads the
`ok` line. Measured before writing anything: **three nested runs of wasm/verify
read 2.751–2.902s where nine plain runs read 2.778–2.891s, and one of the three
was under the recorded floor.** A figure taken by a `go test` running underneath
another process is a figure from a different invocation, and "the method is part
of the reading" is this record's most repeated sentence. A nested run would have
added a new measurement in order to check an old one.

So the reading is the one `go test` already printed, and the only new thing is
the arithmetic.

## What it found, within the hour

**Both records' headline floors were under-set, and the arm said so on the first
sitting it existed for.**

    verifyTimingsTakenOn.wholeFile      "UNDER the band, by 15.2ms" — then
                                        2.762s and 2.765s in a sitting of ten.
                                        Floor 2.78 → 2.76, the fourth widening
    themehistoryTimingsTakenOn          "UNDER the band, by 18ms" — then 2.824s
    .wholePackage                       and 2.843s in a sitting of thirteen.
                                        Floor 2.86 → 2.82, the second widening

Nothing was made faster; both packages GAINED tests this session. Both floors
were set from too few sittings, which is what the sittings rule predicts — and
the difference from the three previous widenings of `wholeFile` is who found it.
Those were found by a person running the command a dozen times and comparing two
ranges by eye. These were reported.

**And the widening exposed two more copies of the band it moved**, both already
stale at `2.92–3.22s` — a version from before the FIRST widening:

    a sentence thirty lines above the field, quoting it in the present tense to
    make a point about core counts → now names the field instead

    the `-short` table's "plain default" cell, which is the same quantity as the
    field written out a second time → now says "the band above". The table that
    exists to price the `-short` lever was comparing against a figure the record
    had already replaced, twice

**A width copy I wrote one iteration ago was stale one iteration later.** The
sentence explaining that `3% wide` had gone stale carried the corrected
percentage — and the floor moved in the same session, found by this iteration's
own arm. The number is gone rather than corrected a second time.

## Verification

Eleven paths, all green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      rc 0
    GOOS=js GOARCH=wasm go build ./...

**Break-tests: six, every one read.** Four against the new arm — unset (the
skip, which prints the two-command recipe), a figure with no unit (refused, not
guessed at), a figure under the floor (UNDER by 158.4ms), and a figure taken
from `go test ./...` (OVER by 270ms, which is the misuse the arm cannot detect
and says so). Two against the new assertion — the machine gate removed (three
failures) and the unreadable-field branch removed (three failures).

**Figures after the widenings, each package alone, through the new arm:**

    wasm/verify wholeFile      2.76–2.97s recorded; readings 33% and 62% up
    internal/themehistory      2.82–3.22s recorded; four readings 2.855–2.961s,
    wholePackage               all in band
    …wholeRun, …batchRetire    in band
    …wholeFileInProcess        in band, unchanged

Five files, +567 −75.

## Next

1. **(age 1 · value medium) Four `GRMOB_` levers are duplicated across the two
   packages and nothing holds them in step.** `bandVerdictEnv`,
   `bandVerdictAsked`, `bandVerdictWanted` and now `packageReadingEnv` are
   declared in both records' packages with identical text, and only functions
   can be registered as two-copy shapes today: the walk skips `const` (it reads
   `token.VAR` only), so registering the two string constants would report them
   missing from both packages rather than holding them. Measured: **4 duplicated
   declarations, 0 divergent today.** The failure they allow is the sharp kind —
   a documented command that works in one package and is silently ignored in the
   other, because a lever nothing reads prints nothing and silence reads as "in
   band". The cost is teaching `twoCopyDeclarationsIn` to see consts and
   `declarationText` to render them as `const`, which pays for four declarations
   now and every future lever.
2. **(age 11 · value low) `…TimingsTakenOn` is not the only record shape.**
   `affordedMeasuredOn` and `foldMeasuredOn` carry bands too, and both are
   re-derived by the run that reads them. Unchanged.
3. **(age 24 · value low) Every Next list in this loop was written by the
   session that would not work it.** Twenty-two iterations. This item's own
   guess was that the honest form "may be a verdict the verify SCRIPT prints" —
   wrong in a checkable way: no verify script runs either package's tests, so
   there was no such place to put it. The instruction that paid was the one it
   attached almost in passing: *measure before building*. The nested-run
   measurement is what chose the design.
4. **(declined, non-goal)** Thirteen entries. See
   `ai_docs/plans/non_goals.md`.
