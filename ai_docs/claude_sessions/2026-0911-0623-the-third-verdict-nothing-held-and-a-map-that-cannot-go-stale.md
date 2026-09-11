# Session: the third verdict nothing held, and a map that cannot go stale

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-11 · Previous:
`2026-0911-0614-a-sentence-saying-two-numbers-are-one-number-beside-a-second-number.md`

## Ask

`/loop` — work the Next list. **Iteration 7**, working both live items: the
oldest one (age 14, the other record shapes) and the one raised last iteration
(the band machinery's map).

## The oldest item, measured once and declined

Fourteen sessions on the list. Measured by parsing the four records' literals:

    affordedMeasuredOn            4 fields,  0 strings, 0 wall-clock bands
    foldMeasuredOn               15 fields,  0 strings, 0 wall-clock bands
    verifyTimingsTakenOn         11 fields,  8 strings, wall-clock bands in 6
    themehistoryTimingsTakenOn    9 fields,  5 strings, wall-clock bands in 4

**Not one wall clock between the two other records.** Every part of the timings
machinery exists for the property a wall clock has and a count does not: it
cannot be re-derived. `affordedMeasuredOn`'s four ending counts are re-walked
over its own eighty names and asserted exactly; `foldMeasuredOn`'s are
recomputed against the Unicode tables, and it already carries its own equivalent
of the machine under its own name — `build foldBuild`, naming the Unicode, ICU
and node versions its counts are a reading of.

So there is nothing to extend, and the item is in `ai_docs/plans/non_goals.md`
as the fifteenth entry, with the condition that would revive it: a wall clock
arriving in one of those records, or a count in one that stops being re-derived.

## The map, which is a log line rather than a paragraph

The band machinery grew to sixteen held declarations over four iterations and
the only way to find the set was to read two lists and the census that reads
them. The obvious answer was a paragraph naming the members — and a paragraph
naming declarations is a thing that goes stale, which is what most of these
censuses exist to catch.

So the census names the set instead of counting it, built out of the lists:

    16 shape(s) kept in two copies — 11 function(s) (packageBase, isMajorVersion,
    importedAs, dotImportsIn, qualifiersFor, unquote, forgetDotImportsReported,
    recordedBand, bandPlacement, againstBandGiven, bandVerdictWanted) and
    5 value declaration(s) (dotImportsReported, recordedBandForm, bandVerdictEnv,
    bandVerdictAsked, packageReadingEnv) — 2 package(s) declaring the whole set…

    21 name(s) declared in both …: 16 held identical, 5 recorded as differing
    with a reason (the two arms, coresAttribution, main, recordMachineDiffers)

Between the two lines a reader under `-v` has the whole of what these packages
share and which half is which. Break-tested by dropping a shape from the list:
the map follows it, and the inversion reports the unregistration in the same
run — the map cannot go stale because it IS the list, and leaving the list is
itself a finding.

## What that map immediately exposed

**A third verdict, with the false UNDER still in it.**

Iteration 4 found both records reporting UNDER on readings that round to their
floors, and fixed the two copies of `againstBandGiven` that the census holds
identical. `wholeFileVerdict` — the TestMain line that prints on every asked run
of `wasm/verify` — is a **third** implementation of the same decision, with its
own wording and its own rounding, and nothing held it in step with the other
two. It was still comparing raw clocks two iterations later.

It no longer decides anything. It supplies the local half — which reading, and
the sentence it belongs to — and the comparison is `againstBandGiven`:

    this package took 2.63s.

    In the band verifyTimingsTakenOn.wholeFileInProcess records (2.55s–2.78s),
    on the machine it names — 34% up a band 230ms wide, so it reaches neither
    end. An end no reading has reached is evidence of nothing…

What that buys beyond the fix: the boundary case — a reading half a step under
the floor, which is AT the floor — is asserted by
`TestTheSentenceAroundAPlacementSaysWhichOfTheFourCasesItIs`, and no wall clock
can be made to land there on demand. Delegating moved this line under a test
that could not otherwise cover it. Its own `round` closure and its bespoke
UNDER/OVER wording are gone, which is 78 lines down to 34.

## Verification

Eleven paths, all green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      rc 0
    GOOS=js GOARCH=wasm go build ./...

**Break-tests: two, both read.** A shape dropped from the list (the map follows,
the inversion fires). And the in-process band moved up so real readings fall
below its floor — UNDER by 78ms, 51ms and 14ms across four runs, in band on the
fourth, from the delegated sentence.

**Figures, each package alone:**

    wasm/verify wholeFile      2.818–2.954s, all in band (2.76–2.97s)
    internal/themehistory      2.877–3.052s, in band (2.82–3.22s)
    wholePackage
    …wholeRun, …batchRetire    both in band

Three files, +102 −56.

## Next

1. **(age 28 · value low, and this is the last of it) Every Next list in this
   loop was written by the session that would not work it.** Twenty-six
   iterations across two runs. The tally, which has not changed and is now
   worth stopping rather than extending: **every instruction paid and no
   conclusion survived intact.** This run added three new failure shapes — an
   item whose measurement was wrong rather than its conclusion (iteration 6, two
   of three constants already held), an item that proposed replacing something
   that cannot be replaced (iteration 5), and an item that named the cheap half
   as the blocker (iteration 2). The instruction that paid in all seven
   iterations is the same one: **measure before writing, and read the findings
   rather than counting them.**
2. **(declined, non-goal)** Fifteen entries. See `ai_docs/plans/non_goals.md`.

There is no open work item. Both live items were closed this iteration — one
declined with its measurement, one built — and the third-verdict defect they
exposed is fixed. The next session either finds something by taking figures,
which is how three of this run's defects were found, or there is nothing to do.
