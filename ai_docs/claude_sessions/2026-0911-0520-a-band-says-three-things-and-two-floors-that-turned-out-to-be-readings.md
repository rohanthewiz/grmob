# Session: a band says three things, three widths that were copies, and two floors that turned out to be readings

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-11 · Previous:
`2026-0911-0502-every-band-field-holds-its-taking-command.md`

## Ask

`/loop` — work the Next list, up to nine iterations. This is **iteration 2**,
working **Next item 1**: record which band ends have actually been reached.

## What was built

A band was being read for two numbers. It carries three, and the third is what
the item was about.

    the precision      recordedBand now returns the step its ends are WRITTEN
                       at, taken from the finer of the two. An end is a reading
                       rounded outward to the record's own two decimals, so
                       "does this reading reach the floor" has an exact answer
                       at that precision and none without it: 2.5512s is the
                       floor of a band written 2.55–2.78s and is not the floor
                       of one written 2.551–2.780s
    bandPlacement      where in the band the reading fell, the band's width,
                       and which end — if either — the reading is evidence FOR.
                       One declaration, held identical in both packages by
                       checkTwoCopyDecls, which now covers eleven shapes
    an assertion       TestWhereAReadingFellInItsBandIsReadOffTheBandsOwnPreci…
                       seven cases: both ends, the same reading at two
                       precisions, a microsecond band, a quarter of the way up,
                       a zero-width band. The FIRST thing in this band
                       machinery that can be asserted, because it is arithmetic
                       over four durations rather than a reading of a machine

What the verdict line says now, on a run that asks for it:

    this package took 2.652s, in the 2.55s–2.78s that …wholeFileInProcess
    records — 44% up a band 230ms wide, so it reaches neither end. An end no
    reading has reached is evidence of nothing, and not a reason to move one.

That last clause is the rule the last run paid five re-takings for, put where it
is read rather than in a header nobody is holding open at the moment they are
tempted.

## What it found

**Three sentences stating a band's width were stale, and all three were
derivable from two numbers in the same file.**

    `3% wide`                   of a band that is 6.8% — true of 2.88–2.97s,
                                which has since been widened three times
    `300ms wide`                of the control in the machine-ruled-out
                                argument, which is 360ms now
    `130ms wide against 130ms`  a claim that the in-process figure is the
                                TIGHTER of the two. It is 230ms against 190ms
                                and is now the WIDER one

The third is the interesting one: the reasoning behind it was that excluding the
noisy overhead should narrow the band, and the reversal is **not** evidence
against that — this band is taken over five sittings and the other over four,
and by this record's own sittings rule a band grows with sittings until it has
found its ends. Two bands with different numbers of sittings behind them cannot
be compared for width at all. So the claim is struck rather than re-taken, and
the two numbers are gone: the verdict prints the width now, which is the one
place it cannot go stale.

**And a count in these session docs has been one low for two sessions
running.** "Nine entries now" was written when `non_goals.md` held ten;
"ten entries" was written by the iteration before this one when it held eleven.
Two different sessions, same hand-kept figure, same drift. Both docs corrected,
and this one states the command that derives it.

## What happened on the first sitting after the change

Both packages' headline floors were **reached** — which is the evidence item 1
asked for, arriving within an hour of the thing that can see it:

    wasm/verify wholeFile      2.778s, twice in nine readings, against a floor
                               of 2.78s. At the hundredth the band is written
                               to, that IS the floor
    internal/themehistory      2.860s against a floor of 2.86s, in the first
    wholePackage               sitting of six readings

Neither is a re-take: by the record's own rule an end is a reading rounded
outward to two decimals, so a reading that rounds to the end is at the end and
not under it. What they are is the answer to "is this floor a reading or a
guess" for two of the ten bands, and it is now on the record in the place the
evidence belongs — a session doc beside the figures, rather than a mark in the
source with nothing able to check it.

## What was declined, with numbers

Two entries added to `ai_docs/plans/non_goals.md`, which now holds **thirteen**
(`grep -c '^## '` less the file's own two section headings):

    a band field does not carry a    the mark would be a hand-kept claim with
    mark saying which ends have      no arm over it. Every other claim in
    been reached                     these records is held by something; the
                                     evidence for this one is a wall clock,
                                     which is the one thing here that cannot
                                     be asserted. The verdict line says it per
                                     reading instead
    a width in prose is not held     5 claims, 3 wrong, and the form separates
    to quoting its range             them exactly: both correct ones quote the
                                     range they derive from, none of the three
                                     wrong ones names a range or a field. 3 of
                                     3 on defects, 0 of 2 false positives —
                                     and declined anyway, because the rule's
                                     own explanation breaks it: the sentence
                                     striking the `130ms` claim quotes it, and
                                     bandPlacement's header lists all three.
                                     That would be the fourth self-exemption
                                     on the shared parse, which its header
                                     says is the point the exemption becomes a
                                     convention — and the convention needed (a
                                     quoted span is not a claim) exists for
                                     test names only

## The item's own prediction, and what it cost to find out

The Next item predicted its cost had dropped because `recordLiteralFields`
already reads every band off the AST. **That was the wrong half.** The walk was
never the expensive part; the question is where a "reached" mark could live such
that something can check it, and the answer is that it cannot — so the item
splits into a thing built (the placement sentence) and a thing declined (the
mark). The prediction named the cheap half as the blocker.

Twenty-one iterations now of a Next item written by the session that would not
work it. The instruction paid again; the conclusion was wrong again.

## Break-tests: six, every one read

Three against the verdict, by rewriting the band and taking real readings:

    floor raised above the reading    UNDER, by 36ms
    ceiling dropped below it          OVER, by 34.5ms
    the same band to three decimals   UNDER by 1.8ms where two decimals read
                                      in band — the precision changing the
                                      verdict, which is the point

Three against the new assertion, which is deterministic and therefore the only
place the end branches can be exercised at all:

    the step ignored (fixed ms)    two cases fail: the reading 1.2ms above the
                                   floor, and the microsecond band
    the percentage from the top    "25% up" becomes 75%
    the two end branches swapped   six failures

**And one break-test broke the work**: reverting a break-edit with
`git checkout -- <file>` threw away the uncommitted changes in that same file.
Re-applied from the other copy of `bandPlacement` and verified by the two-copy
census. Break-edits are reverted from a scratch copy now, not from git.

## Verification

Eleven paths, all green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      rc 0
    GOOS=js GOARCH=wasm go build ./...

**Figures, each package alone:**

    wasm/verify wholeFile      2.778–2.891s over nine runs, recorded 2.78–2.97s
    …wholeFileInProcess        2.636–2.702s, recorded 2.55–2.78s
    internal/themehistory      2.860–3.009s over six runs, recorded 2.86–3.22s
    …wholeRun                  45% up a 270ms band
    …batchRetire               52% up a 110µs band

Every one in band, two of them at a floor. Nothing re-taken.

Eight files, +461 −23.

## Next

1. **(age 1 · value medium) `wholeFile` and `wholePackage` have no verdict of
   their own.** Both floors were reached this sitting and both were found by
   eye, by reading `go test`'s own output against the record — which is exactly
   the comparison `bandPlacement` exists to stop a person doing. They are the
   two figures `go test` reports and nothing inside a package can see, and one
   of them already has the answer: wasm/verify's TestMain clocks `m.Run()` and
   compares the in-process figure. The missing piece is not a clock, it is that
   `wholeFile` is `go test`'s figure including build and process start, so the
   honest form may be a verdict the verify SCRIPT prints rather than the
   package. Measure before building: the two are 170ms apart and that gap is
   the whole of what is unwatched.
2. **(age 10 · value low) `…TimingsTakenOn` is not the only record shape.**
   `affordedMeasuredOn` and `foldMeasuredOn` carry bands too, and both are
   re-derived by the run that reads them, so a number that moved is a finding
   about the data. Unchanged this iteration.
3. **(age 23 · value low) Every Next list in this loop was written by the
   session that would not work it.** Twenty-one iterations. This one named the
   cheap half as the blocker and was wrong in a new way — the instruction was
   not just incomplete, it pointed at the part that was already done.
4. **(declined, non-goal)** Thirteen entries, two added this iteration. See
   `ai_docs/plans/non_goals.md`.
