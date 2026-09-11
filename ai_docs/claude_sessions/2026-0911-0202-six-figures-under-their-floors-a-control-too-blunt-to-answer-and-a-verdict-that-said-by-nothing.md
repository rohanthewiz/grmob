# Session: six figures under their floors, a control too blunt to answer, and a verdict that said "by nothing"

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-11 · iteration 2 of a ten-iteration `/loop`
Previous: `2026-0911-0143-a-convention-already-being-followed-by-hand…`

## Ask

Next item 1 (age 0, value medium): **a band is the one check this repository
has no arm for**, and it had just cost a wrong headline figure. The proposed
form was that the test which takes a reading also prints whether the reading
fell in the band the record carries for it.

---

## What was built

`internal/themehistory/band_test.go`. Three pieces:

    recordedBand           reads the range a record field OPENS with. Every
                           field in both records is written that way and
                           always has been, so the band is a prefix and
                           costs one anchored pattern rather than a second
                           copy of every number
    recordMachineDiffers   the four machine comparisons, extracted from the
                           reporting arm rather than copied
    againstBand            the sentence, appended to the line that already
                           prints the reading

Wired into the three sites in `internal/themehistory` that take a live
reading and name a field: `wholeRun`, `perObjectRun`, `batchRetire`.

It **prints and cannot assert** — the argument that a wall clock is a fact
about one computer has not changed, and a band is two such facts rather than
one. What is new is only that the comparison is now made by something that
does not get it wrong. The limit is stated in the file: a `Logf` is invisible
on a quiet green run, which is enough for the occasion it is for — a person
taking figures at the end of a session, who runs `-v` — and not enough to
catch a floor going stale between sessions.

`wasm/verify` got nothing, because nothing there takes an in-process reading
of `wholeFile` to compare. That is why it was the record that drifted.

## What it found, immediately

**Every band in this repository that could be checked was stale.** The
verdict came back UNDER on the first run it made, and then kept coming back.

    verifyTimingsTakenOn.wholeFile   floor 2.88s,   read from 2.840s
    wholeRun                         floor 1.56s,   read from 1.406s
    perObjectRun                     floor 30.53s,  read from 28.65s
    its batched fetches              floor 409ms,   read 396ms
    its serial trees                 floor 0.87s,   read 835ms
    its themeleaves.Of               floor 0.125s,  read 124ms

Sixteen readings of `wholeRun` over whole-package runs: 1.406–1.571s, exactly
one of them inside the old band. Five of `perObjectRun`: 28.65–29.32s.

**One figure did not move**: `batchRetire`, the only reading in either record
measured in microseconds and the only one not dominated by process spawning.

## The call that was wrong, and the one that replaced it

The first instinct was to REPLACE `wholeRun`'s range — two tight clusters
barely touching is not the shape a busy afternoon makes. That was written,
and then the per-object arm was run and every term on its table was under
too, and re-reading the previous iteration's own note, `wholeFile` had been
under as well.

Six independent figures do not get faster in one evening for six reasons.
They get read low for one. So all of it is **widened to hold both takings**,
and the hypothesis — an idle laptop at two in the morning against working
afternoons — is written down as a hypothesis, because nobody verified it and
the alternative is six unexplained code changes that did not happen.

The cost is stated rather than hidden: `1.40–1.67s` is a band 19% wide, and a
band that wide hides a difference. It is wide because nobody knows which
taking is the anomaly, and the way it narrows again is somebody reading the
printed verdict over several sessions — which the record could not ask for
before, because nothing printed a verdict.

## The correction to iteration 1

That iteration ruled the machine out by reading the untouched sibling at the
same moment, and reported that as the method this record always uses. **The
sibling it consulted was that package's whole-package figure, whose band is
300ms wide, and the difference being ruled out was 140ms.** The control
answered "no difference" because it could not have answered anything else.

`wasm/verify/timings_test.go` now carries that correction beside the original
paragraph. The treatment it chose — widen, not replace — was right; only the
reason was wrong. **A sibling is a control only if its band is tighter than
the difference being ruled out.**

## Two defects the break-tests found

**`by 0s`.** The OVER verdict rounded its difference to the millisecond, and
`batchRetire`'s band is 180µs–290µs — so the first real OVER printed "outside
the band, by nothing". Differences are now rounded to a hundredth of the
band's own width, which is fine enough to be true at either record's scale.

**A `NumCPU` reader the cores note did not name.** `recordMachineDiffers`
reads the core count, and `internal/themehistory`'s `coresAttribution` is
held to naming every reader of it. The census caught it on the first full
run. The note now names it, and says why it is named — it reads the count
without scaling with it, which is a stricter bar than the note's headline
suggests.

---

## Verification

Eleven paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      rc 0
    GOOS=js GOARCH=wasm go build ./...

**Six break-tests, every one read. Two found defects:**

    a field whose range is not first      "No band was read out of …"
    a machine that is not the record's    "Not compared against …"
    the UNDER arm                         fired on its own, twice, on two
                                          different records
    the OVER arm                          fired on its own — and printed
                                          "by 0s", which was the defect
    the file-count arm                    fired on the commit adding
                                          band_test.go, as it does every time
    the cores-note census                 fired on recordMachineDiffers

Figures at the end, all in band:

    wasm/verify                2.840–2.950s   recorded 2.84–2.97s
    internal/themehistory      3.024–3.131s   recorded 2.92–3.22s
    …wholeRun                  in band        recorded 1.40–1.67s
    …batchRetire               in band        recorded 0.18–0.29ms

## Next

Sorted by **value**, highest first; age breaks ties.

1. **(age 0 · value medium) `wasm/verify` still has no in-process reading of
   its own headline figure.** It is the record that drifted, and it is the
   one package where the comparison could not be wired, because `wholeFile`
   comes from `go test`'s own total and nothing inside the package measures
   it. A `TestMain` that times `m.Run()` would produce exactly that figure.
   The complication is real and is why this was not done here: a filtered or
   `-short` run is not the whole package, so the verdict has to suppress
   itself unless `test.run` is empty and `-short` is unset. That is readable
   off the flag package. Perhaps thirty lines, and it closes the gap the last
   two iterations were both about.
2. **(age 0 · value medium) The bands were all set from too few runs, and
   nothing says how many is enough.** `wholeRun`'s floor was set three times
   in one session — seven runs, then sixteen — and was under-read twice
   before the spread stopped widening. The records say "over seven runs" as
   though seven were a convention; it is the number the first one happened to
   use. What a reader needs is not a bigger number but the METHOD beside the
   figure, which `wholeRun` now carries and no other field does.
3. **(age 0 · value low) The verdict is invisible on a quiet green run.**
   Stated as a limit in band_test.go rather than worked around. Closing it
   means the band becomes an assertion, which the whole record argues against.
   Worth revisiting only if a band goes stale again despite the print.
4. **(age 1 · value low) `…TimingsTakenOn` is not the only record shape.**
   Unchanged from the last iteration: the walk census's `costs:` field and
   the GOMAXPROCS table hold readings outside any record, which is what
   blocks the wall-clock rule. `againstBand` would reach them too if they
   were records proper.
5. **(age 13 · value low) Every Next list in this loop was written by the
   session that would not work it.** This one's item was RIGHT — the arm it
   proposed found six stale figures on its first run — and its conclusion
   about what the arm would cost was understated rather than wrong.
6. **(declined, non-goal)** A wall clock in prose is not held to living in a
   record; a backquoted name is not held to being a declaration; a `t.Run` in
   a loop counts as one question; `packageLevelCallsTo` cannot tell an
   initializer from a stored function value; the diff-and-print remainder
   stays unmeasured. See `ai_docs/plans/non_goals.md`.
