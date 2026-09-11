# Session: the fall that was a spread, a cold cache that changed nothing, and a count that was the wrong statistic

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-11 · iteration 5 of a ten-iteration `/loop`
Previous: `2026-0911-0228-a-set-named-for-its-first-subject…`

## Ask

Next item 1 (age 0, value medium): **the within-session fall is unexplained
and is costing 230ms of band width.** The previous iteration measured a 6%
one-directional drop across a session, attributed it to caches warming
without checking which, and carried a margin for it. The item proposed
reading the figure cold, at ten runs, and at thirty.

---

## The experiment, run with a temporary cache rather than by clearing theirs

`go clean -cache` would have wiped the user's whole Go build cache
machine-wide to answer a question about one package. `GOCACHE=$(mktemp -d)`
answers the same question in isolation and removes nothing, so that is what
was used.

**The build cache is ruled out.** A run under a completely empty GOCACHE —
rebuilding the standard library and everything else — produced an in-process
figure of **2.643s**, indistinguishable from the warm runs either side of it.
Obvious in hindsight, since the clock starts after the binary is built, and
worth measuring precisely because the written explanation had named a warming
cache.

**And compilation is ruled out of `wholeFile` too**, which corrects something
this record said two iterations ago. That cold run took **6.57s of wall
clock** and `go test` still **reported 2.849s**. The reported figure excludes
compiling. The ~190ms between the two fields is the process starting,
package initialisation and the build *check* — not the build.

## The previous iteration's conclusion was wrong

It said: *a reading of this package falls over the course of a session.* That
was drawn from a sequence that happened to descend — 2.767s early, 2.593s
hours later, one direction.

A later sitting of twelve consecutive runs went **the other way**: 2.555s up
to 2.733s, no trend. Across the whole session the figure runs **2.555–2.767s,
a spread of 210ms**, and any one sitting of eight or twelve samples about
half of that and looks tight doing it.

So it is not a fall. It is a spread, and both accounts are now in the record
— the wrong one kept beside the right one, because the wrong one is what the
evidence honestly looked like.

## Which explains all three re-takings, and answers an older item

**Consecutive runs are correlated.** Sixteen runs back to back are not
sixteen samples of the figure; they are closer to one sample of a *sitting*,
and the next sitting lands somewhere else in a distribution twice as wide.

That is why three bands were re-taken in one evening and why each re-taking
was contradicted by the next sitting. Every one of them was set from runs
taken in a row.

It also answers the standing item "nothing says how many runs a band needs":
**the count of runs is the wrong statistic.** Ten runs at three different
times beats fifty in a row. The rule is now stated once in
`verifyTimingsTakenOn`'s header, where it governs every field, and the fields
that carry only a count are marked as older than the lesson.

Bands now name their sittings:

    wholeFileInProcess   2.55–2.78s over about sixty runs in five sittings
    wholeFile            2.78–2.97s over forty-six runs in four sittings
                         across two sessions

`wholeFile` was widened once more during this iteration, on a 2.798s reading
against a 2.80s floor. That is the rule being applied — widen to hold what
has been *seen* — and not the chasing the rule warns about, which would be
re-centring the band on the latest sitting.

## What is left, and what it costs

The machine: the file cache, and whatever else about an eight-core laptop
differs between one ten-minute stretch and another. Not measurable here
without a reboot or a cache purge. It is now **bounded rather than
explained** — 210ms, which is what the band holds. The band is 230ms wide for
a figure whose observed spread is 210ms, which is the honest width and not a
padded one.

---

## Verification

Eleven paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      rc 0
    GOOS=js GOARCH=wasm go build ./...

Readings taken this iteration: 3 warm baseline, 1 under a cold GOCACHE, 3
under that cache once warm, 12 consecutive, 5 confirming the widened floor —
24 in five sittings.

Figures at the end, all in band:

    wasm/verify             2.799–2.892s   recorded 2.78–2.97s
    …wholeFileInProcess     2.636s         recorded 2.55–2.78s
    internal/themehistory   2.887–2.905s   recorded 2.86–3.22s
    …wholeRun, …batchRetire in band

## Next

1. **(age 0 · value medium) Eight of the ten band fields still carry a run
   count and no sitting count.** The lesson is stated in the record header
   and applied to two fields. The other eight — `walkEnumerate`,
   `walkRead`, `walkParse`, `foldWalk`, `wholePackage`, `wholeRun`,
   `perObjectRun`, `batchRetire` — say "over seven runs" or nothing at all,
   and by tonight's finding that means their spreads are probably understated
   the same way. The cheap form is not to re-take all eight: it is to take
   ONE of them across three sittings and see whether the spread widens as
   predicted. If it does, the others are known to be narrow and can be
   widened on paper with the argument; if it does not, tonight's finding is
   about this package and not about bands.
2. **(age 3 · value low) The verdict is invisible on a quiet green run.**
   Confirmed twice. Closing it means an assertion, which both records argue
   against.
3. **(age 4 · value low) `…TimingsTakenOn` is not the only record shape** —
   the walk census's `costs:` field and the GOMAXPROCS table hold readings
   outside any record, which blocks the wall-clock rule.
4. **(age 16 · value low) Every Next list in this loop was written by the
   session that would not work it.** Five for five, and this is the clearest
   case yet: the item asked for an experiment, the experiment ran, and it
   refuted the previous iteration's stated conclusion rather than refining
   it. The instruction paid; the conclusion did not survive.
5. **(declined, non-goal)** Typed bands, plus the five standing entries. See
   `ai_docs/plans/non_goals.md`.
