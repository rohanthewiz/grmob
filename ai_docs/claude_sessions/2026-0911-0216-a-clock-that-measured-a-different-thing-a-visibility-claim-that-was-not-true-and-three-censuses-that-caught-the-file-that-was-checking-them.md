# Session: a clock that measured a different thing, a visibility claim that was not true, and three censuses that caught the file that was checking them

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-11 · iteration 3 of a ten-iteration `/loop`
Previous: `2026-0911-0202-six-figures-under-their-floors…`

## Ask

Next item 1 (age 0, value medium): **`wasm/verify` has no in-process reading
of its own headline figure.** It is the record that drifted, twice, and the
one package where the previous iteration's band verdict could not be wired,
because `wholeFile` is what `go test` reports and nothing inside the package
measures it. The proposed fix was a `TestMain` timing `m.Run()`.

---

## What was built

`wasm/verify/wholefileband_test.go`, `//go:build !race`. A `TestMain` that
clocks `m.Run()` and prints the reading against the band, plus a copy of
`recordedBand` and the four guards that decide whether this invocation is the
command the band is about.

The build tag is the `-race` guard: under `-race` the figure is a different
one recorded separately, so rather than detect the race detector the whole
file is excluded and there is no `TestMain` at all. The other four
suppressions are read off the flag package after `m.Run()` returns —
`-short`, `-run`, `-count`, `-bench` — and off the machine the record names
it says nothing.

## Two claims in that file that were false, and were checked

**It does not measure `wholeFile`.** The first version compared the reading
against `verifyTimingsTakenOn.wholeFile` and reported UNDER by 170ms on a run
`go test` called 2.958s. `go test`'s figure includes the build check, the
process starting and package initialisation; a clock inside the process sees
none of it. Comparing them reports a constant overhead as drift — the exact
mistake the record exists to stop.

So the record grew `wholeFileInProcess`, **2.65–2.78s over sixteen runs**,
sitting beside `wholeFile` the way `wholeRun` sits beside `wholePackage` in
`internal/themehistory`. That record grew the pair for this reason two
sessions ago; this one arrived at it from the other direction.

**It is not more visible than a Logf.** The file was written claiming that
stderr from `TestMain` reaches a plain `go test` where a `t.Logf` only
reaches `-v`. It does not — `go test` prints a passing package's output under
`-v` only. The line was seen on a plain run once, on a run where the package
was **failing**, which is the other occasion `band_test.go` already names.

What the file earns is therefore smaller than it was written to be, and still
worth having: it is the only in-process reading this package has, where
before there was none. Both corrections are in its header.

## Three censuses caught the file that was checking the records

Every one of them fired on the first full run, and two were real design
objections rather than registration chores:

    the cores note        wholeFileRunIsTheRecordedOne reads NumCPU, and
                          wasm/verify's coresAttribution is held to naming
                          every reader of the count
    the -short census     this repository holds `-short` to being ONE lever,
                          and "a `-short` read outside a test is a lever on
                          something else". Correct: nothing here skips
                          anything. The guard now reads the FLAG, like its
                          three neighbours, which is both what it meant and
                          what the census asked for
    the float census      `f * float64(unit)` needed an entry in
                          affordedFloatDerivations saying whether anything
                          compares it. Nothing does — it becomes a Duration
                          immediately and every comparison after it is
                          integer

## Two defects in the rounding, one of them a repeat

**`by 0s`, again.** The previous iteration fixed this by scaling the rounding
step to the band's width. That fixed the wrong half: a difference SMALLER
than one step still rounds to zero, and "outside the band by nothing" is the
same useless sentence reached from the other side. A difference that rounds
away is now reported at microsecond resolution — whatever it is, it is not
nothing, because the arm only runs when the reading is outside. Fixed in both
copies. Forced and read: a band of 2.67–9.00s makes the step 63ms, and a
reading 12ms under the floor printed `by 11.994ms` instead of `by 0s`.

**A floor pinned to the exact minimum.** The band was first written as the
observed ends, 2.666–2.767s, and the next run came in a fraction under and
reported itself outside a band it had helped set. A range over N runs is a
sample; its ends are the two most extreme readings in it, not limits. Bands
are rounded outward to the two decimals the records already write —
2.65–2.78s. This is the cheap version of the lesson `wholeRun` learned by
having its floor set three times in one evening, and it is written into the
field's doc.

---

## Verification

Eleven paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      rc 0
    GOOS=js GOARCH=wasm go build ./...

**Break-tests, every one read:**

    a floor 1s above the reading     UNDER, by 978.9ms
    a step larger than the gap       by 11.994ms, not "by 0s"
    -short                           no verdict
    -run <no match>                  no verdict
    -count=2                         no verdict
    -race                            no verdict (file not compiled)
    the file-count arm               fired on the commit adding the file
    the cores note census            fired on the new NumCPU reader
    the -short lever census          fired on testing.Short()
    the float census                 fired on f * float64(unit)

Figures at the end, all in band:

    wasm/verify                2.856–2.948s   recorded 2.84–2.97s
    …wholeFileInProcess        2.694s         recorded 2.65–2.78s
    internal/themehistory      3.046–3.114s   recorded 2.92–3.22s
    …wholeRun, …batchRetire    in band

## Next

1. **(age 0 · value medium) `recordedBand` is now two copies and nothing
   holds them identical.** `checkImportResolverCopies` is exactly the
   machinery for this, and its set is named for import resolution — adding an
   unrelated pair would give that census a name that had stopped being true,
   which this repository has chased twice. The way out is not a bigger
   census: it is for the records to carry their bands as **durations** and
   render the prose from them, at which point there is nothing to parse, no
   regexp, no "the field must open with the range" fragility, and no copy.
   Both records have about a dozen fields; the change is mechanical and it
   deletes more than it adds.
2. **(age 1 · value medium) Nothing says how many runs a band needs.**
   Unchanged and now better evidenced: `wholeRun`'s floor was set three times
   in one evening, and `wholeFileInProcess`'s once before the outward
   rounding. Sixteen runs plus outward rounding has now held twice. The
   method belongs beside the figure — two fields carry it and the rest do
   not.
3. **(age 1 · value low) The verdict is invisible on a quiet green run.**
   Now confirmed rather than assumed: `TestMain` on stderr is no better than
   a `Logf`. Closing it means an assertion, which both records argue against.
4. **(age 2 · value low) `…TimingsTakenOn` is not the only record shape** —
   the walk census's `costs:` field and the GOMAXPROCS table hold readings
   outside any record, which is what blocks the wall-clock rule.
5. **(age 14 · value low) Every Next list in this loop was written by the
   session that would not work it.** This item was right about the mechanism
   and wrong about what it would measure: the clock it proposed reads a
   different quantity than the figure it was meant to check, which is why
   the record has one more field than the item expected.
6. **(declined, non-goal)** See `ai_docs/plans/non_goals.md`.
