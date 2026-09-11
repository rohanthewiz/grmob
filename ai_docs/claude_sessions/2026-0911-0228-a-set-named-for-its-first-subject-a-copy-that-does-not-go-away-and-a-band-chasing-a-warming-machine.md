# Session: a set named for its first subject, a copy that does not go away, and a band chasing a warming machine

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-11 · iteration 4 of a ten-iteration `/loop`
Previous: `2026-0911-0216-a-clock-that-measured-a-different-thing…`

## Ask

Next item 1 (age 0, value medium): `recordedBand` is two copies and nothing
holds them identical. The item proposed typed bands — the records carrying
`lo`/`hi` as durations and rendering the prose — and claimed it would leave
"nothing to parse, no regexp, no copy", mechanically, deleting more than it
added.

---

## The item's claim was wrong, and the measurement says where

**The copy does not go away.** These are two separate `package main` programs
and neither can import the other's tests. That is *why* the parser is
duplicated, and a renderer is duplicated for exactly the same reason and at
about the same size — roughly 30 lines becomes roughly 25.

**And the rendering costs more than the parsing.** Over the ten figure fields
in the two records:

    3 fields    a bare range — "0.08–0.10s"
    5 fields    a range then a note
    2 fields    a range, a note, and MORE ranges inside the note.
                perObjectRun carries three further bands and a ratio range

A renderer has to reproduce each field's chosen unit and precision, because
the records write `0.08–0.10s` and `0.18–0.29ms` where
`time.Duration.String()` gives `80ms` and `290µs`. So the struct needs a unit
and a decimal count beside the two durations — four pieces of data to say
what the string `"0.08–0.10s"` already says exactly and more readably. The
two fields with embedded ranges do not fit the shape at all.

Declined, with the numbers, in `ai_docs/plans/non_goals.md`. **What would
change it: a third package needing the same reader** — which is the point at
which two copies become a shape kept in step by whoever remembers to, the
argument `twoCopyPackages` already makes for its own value.

## What was done instead

Held the two copies identical, using the machinery that already exists for
exactly that. The obstacle was never the mechanism — it was the name.

`checkImportResolverCopies` compares declarations by name across the two
directories with doc comments stripped, off a list. Nothing in it is about
import resolution except what it is called. So the set is renamed for what
unifies it rather than for its first subject, which is the move this
repository already made once when `copies_test.go` became
`sharedparse_test.go`:

    importResolverDecl            → twoCopyDecl
    importResolverDeclarationsIn  → twoCopyDeclarationsIn
    importResolverShapes          → twoCopyFunctionShapes
    importResolverStateShapes     → twoCopyStateShapes
    importResolverAllShapes       → allTwoCopyShapes
    importResolverCopies          → twoCopyPackages
    checkImportResolverCopies     → checkTwoCopyDecls

Thirty-four identifier occurrences and thirteen prose mentions; the subtest
is now "the declarations kept in two copies". `recordedBand` joins the
function list and `recordedBandForm` the state list — state for the same
reason `dotImportsReported` is: the compiler insists both packages have it
and has no opinion about whether the two spell the same range.

**Ten shapes, two copies each, held identical.** A band reader that took a
hyphen in one package and an en dash in the other would have compiled,
passed, and silently stopped comparing half of one record.

## The third re-take in one evening, which is itself the finding

The band set two hours earlier, `2.65–2.78s` from sixteen runs, read 2.608s
with nothing changed. Over this session the in-process figure went from
2.767s at its widest down to 2.593s — **about 6%, one direction, no code.**

At the third re-take, the re-taking is the finding: **a reading of this
package falls over the course of a session.** The build cache warms, the file
cache warms, the same binary runs for the tenth time. A figure taken in one
sitting is a figure about that sitting.

Two rules came out of it, both written into the record:

- **Widen to hold what has been SEEN, then leave it alone** until something
  is known to have changed. Chasing each reading down produces a band always
  correct about the last run and never about the next.
- **Outward rounding handles sampling noise and not a systematic drift.** A
  floor outward-rounded to 2.60s from a 2.608s low reported UNDER within four
  runs, because the fall had not finished. The floor now carries a margin
  *measured* from the drift — the fall was 160ms end to end and the floor
  sits about a third of that below the lowest reading. It is the one number
  in the record that is not a reading, and it says so.

Three bands widened to hold both ends of the session:

    wholeFileInProcess   2.55–2.78s over thirty-four runs
    wholeFile            2.80–2.97s over forty runs in two sessions
    wholePackage         2.86–3.22s over sixty-three runs in three sessions

---

## Verification

Eleven paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      rc 0
    GOOS=js GOARCH=wasm go build ./...

**Break-tests, read:**

    a hyphen for an en dash in one     the census names recordedBandForm and
    copy of the pattern                both files
    one copy's body changed            the census prints both declarations
                                       and says which files
    eight consecutive runs against     all in band
    the new floor

Figures at the end, all in band:

    wasm/verify             2.800–2.822s   recorded 2.80–2.97s
    …wholeFileInProcess     2.599–2.703s   recorded 2.55–2.78s
    internal/themehistory   2.917–2.967s   recorded 2.86–3.22s
    …wholeRun, …batchRetire in band

## Next

1. **(age 0 · value medium) The within-session fall is unexplained and is now
   costing 230ms of band width.** It was measured — 6% over one session, one
   direction — and attributed to caches warming without anybody checking
   which cache. The experiment is cheap and mechanical: read the figure cold
   (`go clean -cache` then one run), then at ten runs, then at thirty, and
   see which of the build cache, the file cache or the binary's own warmth
   accounts for it. A margin that is understood can be taken back out, and
   the band goes from 230ms to something near the sampling noise.
2. **(age 2 · value medium) Nothing says how many runs a band needs.** Now
   partly answered — the count is not the thing, the SPREAD OF OCCASIONS is,
   and two fields carry a method line saying so while eight do not. The cheap
   form is to put the method in every field that has a band.
3. **(age 2 · value low) The verdict is invisible on a quiet green run.**
   Confirmed twice now. Closing it means an assertion, which both records
   argue against.
4. **(age 3 · value low) `…TimingsTakenOn` is not the only record shape** —
   the walk census's `costs:` field and the GOMAXPROCS table hold readings
   outside any record, which blocks the wall-clock rule.
5. **(age 15 · value low) Every Next list in this loop was written by the
   session that would not work it.** Four for four. This item named the right
   problem and the wrong fix, and the fix it named would have added a
   duplicate of its own.
6. **(declined, non-goal)** Typed bands, plus the five standing entries. See
   `ai_docs/plans/non_goals.md`.
