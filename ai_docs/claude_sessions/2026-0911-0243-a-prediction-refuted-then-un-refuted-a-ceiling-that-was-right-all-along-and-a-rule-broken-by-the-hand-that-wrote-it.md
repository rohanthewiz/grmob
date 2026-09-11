# Session: a prediction refuted then un-refuted, a ceiling that was right all along, and a rule broken by the hand that wrote it

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-11 · iteration 6 of a ten-iteration `/loop`
Previous: `2026-0911-0235-the-fall-that-was-a-spread…`

## Ask

Next item 1 (age 0, value medium): take **one** of the eight band fields that
carry a run count and no sitting count, read it across three sittings, and
see whether its spread widens as the previous iteration's finding predicted.
"If it does, the others can be widened on paper with the argument; if it does
not, tonight's finding is about this package and not about bands."

A falsifiable prediction with both branches written down in advance. It got
both answers.

---

## Three sittings said the prediction was wrong

`walkParse` — one test, one repository-wide parse, read off `-v`'s own
`--- PASS:` line, the method the record already documents. Twenty-one runs in
three sittings, with the whole verification suite run in between to move the
machine:

    sitting 1   0.20 0.19 0.19 0.20 0.20 0.20 0.19
    sitting 2   0.20 0.20 0.19 0.20 0.20 0.20 0.19
    sitting 3   0.20 0.20 0.20 0.20 0.19 0.19 0.20

Two values, in all three sittings, where 0.01s is the resolution of the line
they are read off. Seven runs each of the other two depths gave the same
answer — 0.09–0.10s and 0.15–0.16s. **No spread to widen.**

So the write-up said the three depths were reproducible, that variance is a
property of what is being timed rather than of having a band, and the
sittings rule was narrowed to the aggregate figures. `walkParse` was re-taken
at 0.19–0.20s, since 0.20–0.24s had a floor three of twenty-one runs fell
below.

## Then five more runs said it was right

    sitting 4   0.19 0.19 0.20 0.20 0.22

One reading in twenty-six, outside a band set on the other twenty-five, in
the sitting immediately after the one that set it. And the sitting after
**that**:

    sitting 5   0.20 0.19 0.19 0.24 0.19 0.19

**0.24s — the original ceiling, exactly.** Thirty-two runs: twenty-nine at
0.19 or 0.20, one at 0.22, one at 0.24. Whoever first wrote `0.20–0.24s` had
seen the tail and put the ceiling on it, and three sittings of re-measuring
had not reached it.

## What that settles, which is neither of the things written tonight

Not "these are reproducible" and not "bands need sittings". Both were written
from samples that had not finished. What thirty-two runs say is narrower:
**the middle of this figure is tight and its tail is not** — which is what a
wall clock on a shared machine is — and three sittings of seven did not find
the tail.

The band is now `0.19–0.24s`: the original, with its floor let out one
resolution step. The recorded band was wrong only at the floor.

## The rule broken by the hand that wrote it

Iteration 4 wrote: *widen to hold what has been SEEN, then leave it alone
until something is known to have changed. Chasing each reading down produces
a band always correct about the last run and never about the next.*

Re-taking `0.20–0.24s` as `0.19–0.20s` is not widening. It is re-centring on
the latest sitting, which is precisely what the rule forbids, and it was
falsified within five runs by a reading the old band would have held. Two
iterations between writing the rule and breaking it, same record, same hand.

Which is the argument for a band being **widened and almost never narrowed**.
A ceiling nothing has reached in twenty-six runs costs a reader nothing. A
ceiling somebody tightened costs a false verdict the first time the tail
shows up.

## One constraint found on the way

The three walk depths are the only figures in either record **interpolated
into a sentence** — `repowalks_test.go` reads them into "%s, %s and %s
respectively where verifyTimingsTakenOn was taken". A first attempt at
writing the sitting count into their values turned that list into nonsense.
The method lives in the comment and the values stay bare; the rendered list
was forced and read to confirm it.

---

## Verification

Eleven paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      rc 0
    GOOS=js GOARCH=wasm go build ./...

Readings: 32 of `walkParse` in five sittings, 7 each of `walkEnumerate` and
`walkRead`, plus the package figures. The failure message that interpolates
the three depths was forced and read.

Figures at the end:

    wasm/verify             2.796–2.820s   recorded 2.78–2.97s
    …wholeFileInProcess     2.701s         recorded 2.55–2.78s
    internal/themehistory   2.861–2.951s   recorded 2.86–3.22s
    walkParse               0.19–0.24s     recorded 0.19–0.24s

## Next

1. **(age 0 · value medium) Two bands were narrowed tonight and one of them
   was wrong within five runs. The other has not been checked.**
   `wholeFileInProcess` went 2.65–2.78 → 2.60–2.78 → 2.55–2.78, which is
   widening and therefore fine, but `wholeRun` was REPLACED — 1.56–1.67s
   became 1.40–1.67s, holding both, and `perObjectRun` went 30.53–30.84s to
   28.65–30.84s, also holding both. Those two are safe by the rule. The one
   to check is `walkEnumerate` at 0.08–0.10s, whose floor nothing reached in
   seven runs: by tonight's finding that is a tail nobody has sampled, not a
   floor to raise. Leave it, and say so in the record — an unreached end is
   evidence of nothing.
2. **(age 1 · value low) Eight fields still carry no sitting count**, and the
   value of adding one is now smaller than it looked: the sittings are how
   you find a tail, but the tail was already in the original ceiling. The
   useful form is to record, per field, **the widest reading ever seen**
   rather than the method.
3. **(age 4 · value low) The verdict is invisible on a quiet green run.**
4. **(age 5 · value low) `…TimingsTakenOn` is not the only record shape.**
5. **(age 17 · value low) Every Next list in this loop was written by the
   session that would not work it.** Six for six — and this is the first one
   whose item was written well enough to be wrong usefully: it named both
   branches in advance, which is why the reversal was legible instead of
   just confusing.
6. **(declined, non-goal)** See `ai_docs/plans/non_goals.md`.
