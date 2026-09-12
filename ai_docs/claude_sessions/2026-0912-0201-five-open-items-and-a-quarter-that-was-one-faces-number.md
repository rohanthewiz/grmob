# Session: five open items, and a quarter that was one face's number

Session: https://claude.ai/code/session_018HHEppckHrYoZNwpi2jgha
Date: 2026-09-12 · Previous:
`2026-0912-0117-two-flags-a-face-and-the-first-green-master.md`

## Ask

> how many items are in the Next list?

> Do the 5 open items

All five are closed. Six commits, `ee9d43a` through `201de11`, and the
result that matters is one line of a Linux runner's log:

```
20 of them their words in their own ink and 8 their counts as digits inside
their own pills — the thinnest stem among them reaching 90.7% …
```

no `SKIP:` above it. The band grid's ink scan has not read a glyph on a
Linux machine since it started skipping, and the previous session's Next
list said the way out was "a measurement session, not a fix".

It was a measurement session. What it measured was that the constant
everybody suspected was innocent and the one nobody was looking at —
`INK_ROWS` — was a quarter because Times has serif feet.

The four sentences worth keeping:

1. **A skip that asks for a measurement should take it.** The remedy was
   "re-measure three constants on a machine that has the face", addressed to
   readers of a CI log who cannot attach to that machine. Two sessions met it
   and did nothing. The skip now prints the readings.
2. **The instrument was checked against the face already measured, and that
   is what made it usable.** Forced onto Times, it reproduced
   `INK_ROW_ROUNDING`'s recorded bracket — 0.100 and 0.239 — to three
   decimals. Then it did not reproduce `INK_ASCENDER_SEPARATION`'s, and the
   difference was real: that table had gone stale in silence.
3. **Three numbers in this file were arrived at by hand and three were
   wrong.** A constant that never existed, an ascender bracket that had
   drifted, and a stem reading computed off one printed pixel (0.915, where
   the machine says 0.907). Each was caught by something printing the
   measurement instead.
4. **Two of the five items closed by being carried out rather than built.**
   The Chrome-start mystery recurred and the instrument named it; the
   windowing device number was already absorbed into `need_hardware.md` and
   only needed the Next list to stop carrying it.

---

## Item 1 — the ink scan is unread on every Linux machine · closed

### The instrument, and the drill that validated it

`inkBandProfile` sweeps every band and two rows past each end, through the
same coverage predicate the verdicts use, and `inkCalibrationReport` prints
the two brackets the constants sit in plus the row-by-row profile. Only on
the skip path, so a calibrated run pays nothing.

Checked by forcing the calibrated name to one nothing resolves, which runs
the whole apparatus against Times:

```
INK_ROW_ROUNDING is 0.15. Here: over 10.0% and at or under 23.9% — a bracket
```

0.100 and 0.239 are what the constant's own comment records. The first
pooling bug was caught the same way: the ascender bracket came back 33.3% on
both sides, which is two digits of a six-column count pill in a statistic
that is about lower-case words.

### What the profile showed, which was not the floor

```
Times               |45:27/29     the baseline row, 27% glyph
Liberation Serif    |45:3/4       the same row, 3%
```

Identical bands, identical rows, one face's outlines reaching its own metric
baseline and the other's stopping short of it. A quarter of a six-row band
puts the bottom scanned row one row above that, so the rounding
`INK_EDGE_CLEARANCE` exists to survive would have taken it off the glyphs —
on a grid where every word was drawn correctly.

`INK_ROWS` is three tenths now, and Times is not compromised to accommodate
the second face:

```
                  Times    Liberation Serif
a quarter         0.239    0.027
three tenths      0.333    0.333        floor 0.15
```

### The half of that which was wrong, and how it was found

Raising `INK_EDGE_CLEARANCE` to 2 alongside it looked free. It is not: the
band's edges are fractional (`y=62.51 to 68.50`) and the rows round
independently of them, so four of the twenty-eight boxes clear one row and
not two. The arithmetic that predicted otherwise was done on rounded edges —
a model of a rounding rather than a rounding — and the grid said so the
moment the constant moved. It stays 1, with that written where it is.

### Then the scan ran, and found the next premise

With Liberation Serif recorded, Linux ran the scan for real. Twenty labels
passed. Six of eight count pills failed one reading, and the reading was
right to fire and wrong to be a boolean:

> `ink` asks whether some pixel IS the declared colour, on a premise written
> beside it: a stem's interior is unblended at any size a caption is set at.

A claim about a face, a size and a rasterizer. Liberation Serif's digits in a
caption-sized pill never fill a pixel, so the whitest pixel in a white number
on a `#0040DD` pill is `#EAEFFC` — a correctly drawn number, every pixel of
it a blend. `INK_STEM_REACH` makes it a fraction: 1.000 on Times, 0.907 at
worst on the runner, floor 0.8. The two tall-caption pills are the same face
two points bigger and pass outright, so it is about size and reaches the face
only through it.

What still refuses a count that lost its declaration: off the pill-to-ink
line, `offSegment` at three channels; on that line but dimmer, this floor —
half strength reaches 0.5.

**The floor has a measurement on one side and none on the other**, so the
tail recites the thinnest stem the run actually found rather than leaving the
margin to be spent in a comment. That is also what caught 0.915 → 0.907.

---

## Item 2 — `site`'s Chrome failed to start once · closed

It recurred, on CI, and `faa0b7c`'s instrument said what a year of the old
message would not have:

```
Chrome never reported a DevTools port: Chrome is still running and has not
written one after 10s.
```

**Alive.** Not a rejected flag, not a crash, not a missing library — a cold
Chrome on a contended runner. Ten seconds was never derived from anything.

It is a minute now, and the paragraph says what kind of number that is: a
bound on "a browser that is going to start has started", not a timeout tuned
to a workload — the distinction `batchRetireGrace` draws in
`internal/themehistory`. Being generous costs only a run that is already
failing, because the wait returns the instant the port appears and the arm
added last session returns the instant Chrome exits.

---

## Item 3 — nothing checks that a gate's remedy sentence works · closed for
## this gate, and the class is now checked

Three parts, of which the middle one is new standing machinery.

**The sentence was wrong, and in the way the item predicted.** The ink skip
told readers to re-measure `INK_ROWS`, `INK_EDGE_CLEARANCE` and
`INK_ROW_GLYPH_FLOOR`. Two of those exist. The third never has — not
renamed, not moved, never declared — printed on every Linux run for a
session, in the one message whose whole job is to say what to go and do.

**`TestTheConstantsNamedInProseResolve`** is the rule for that class: a
`SCREAMING_SNAKE` name in `wasm/verify`'s prose resolves to something the
directory or the runtime it exercises declares, to an environment variable
something reads, or to a platform member listed by name. Measured before it
was written — 108 declarations, 16 unresolved against `browser.mjs` alone,
two left once the siblings and the runtime are in the namespace, and both of
those are `GeolocationPositionError`'s.

It caught two on its first run, both mine. `INK_CALIBRATIONS`, named three
times in prose about where a calibration is recorded when the thing that
records one is `INK_CALIBRATED_ON`. And the repository's existing prose rule
caught this test's own comment citing a test name I had not looked up.

**And the remedy was carried out**, which is the item's literal shape: strip
the precondition (the runner has the wrong face), run the remedy (take the
readings, move the constants, record the face), run the pass, expect
not-a-skip. It is not a skip.

**What is left**, and it is the expensive half: `android/verify`'s and
`ios/verify`'s toolchain remedies are still sentences nothing executes. Those
need a network, a Gradle cache and an SDK to drill.

---

## Item 4 — `retire`'s bound is conditional and stated, not enforced · closed

`retire` terminates because the Kill behind its deadline reaches whatever
holds the pipe's write end — true of a child that does not fork, and `git
cat-file --batch` does not. That was a paragraph, and every sentence in it is
about WHICH COMMAND.

`TestTheProcessBehindARetiredPipeDoesNotFork` is a census over the package's
source: one place attaches a process to a `batchReader`, and the process
there is that argv and nothing else. Drilled on all three arms, each of which
fails it with the remedy `retire`'s own comment names:

```
a shell behind the pipes      newBatchReader starts [[sh -c git cat-file --batch]]
--filters                     … [[git cat-file --batch --filters]]
a second construction site    attaches a process … in [aSecondReaderSomebodyAdded …]
```

The test files are out of it on purpose: `main_test.go` wraps a `batchReader`
around `sleep` deliberately, and that is the bound's other arm rather than a
counterexample.

---

## Item 5 — windowing's device number · closed, as already absorbed

No code. The device pass this was waiting for is `need_hardware.md`'s
"`launch.sh`'s numbers are one emulator's", which says in its own words that
it no longer gates windowing, and `non_goals.md` carries the declined
proposal. The two documents now point at each other: `non_goals.md` says
where the device number is tracked and which direction it could move the
answer (smaller, not larger). The item was being carried in the Next list
because nothing said it had already landed somewhere.

---

## Found on the way

- **`INK_ASCENDER_SEPARATION`'s recorded brackets had drifted.** 0.708–0.947
  and 0.011–0.154 when it was set; 0.658–0.867 and 0.009–0.128 today, every
  extreme belonging to a tall-caption band the grid grew afterwards. Nothing
  failed and nothing would have — 0.43 still separates — which is the point.
  Both columns are recorded now and the margins are said plainly (0.302
  below, 0.228 above, where the comment claimed symmetry).
- **The tracked-Go-file count moved twice**, 439 → 441, and the repository's
  own census caught both on the run that added the file.

## What is checked, and what is not

- `go test ./...` and `gofmt` — green. `wasm/verify/run.sh` — green here, on
  Times, both face arms of the calibration drill driven.
- **Both workflows green on `201de11`**, with the ink scan reading all 28
  boxes on the Linux runner.
- **Not checked:** whether a minute is enough for the slowest cold Chrome a
  GitHub runner produces. One observation says ten seconds was not.
- **Not checked:** anything needing a real caret, keyboard, IME or device.
  See `ai_docs/plans/need_hardware.md`.

## Next

1. **(new · value medium) `INK_STEM_REACH` is a floor with one side
   measured.** 0.8 sits under a single reading of 0.907 and over nothing at
   all. The other bracket would come from painting a count in a dimmer
   version of its own declared ink and reading what the scan says — a
   fixture, not a grid change — and until that exists the number is an
   argument with one measurement under it. The tail recites the thinnest stem
   on every run, so the margin cannot be spent silently in the meantime.
2. **(new · value low) A third face is still a skip, and now a cheaper
   one.** Any machine resolving something other than Times or Liberation
   Serif gets the skip plus the table it needs. What nobody has is a face
   where the readings do NOT clear — the report has an arm for it ("WHICH
   DOES NOT SEPARATE") that has fired exactly once, on Liberation Serif under
   the old fractions, and is now unreachable by any face this project has
   met.
3. **(carried · value medium) The toolchain gates' remedy sentences are
   still unexecuted.** `android/verify`'s Gradle-cache remedies and
   `ios/verify`'s SDK one. The ink gate's was drilled this session and the
   shape generalises — strip the precondition, run the remedy, run the pass,
   expect not-a-skip — but those two need a network, a Gradle cache and an
   SDK, so it is a CI job rather than a test.
4. **(carried · value low) Windowing as a proposal.** Declined on the
   payload in `non_goals.md`; `TestWhatWindowingWouldSave` asserts its own
   share and is the profile to re-take if a forty-lesson chapter ever ships.
   The device number it was once waiting on is `need_hardware.md`'s, and the
   two files now say so to each other. Not a Next item any more except as
   this line.
5. **(closed) The ink scan is unread on every Linux machine.** Done — and
   the way out was neither of the two the last session named. It was not
   installing the face and not a per-face table; it was one constant that had
   been a quarter because Times has serif feet.
6. **(closed) `site`'s Chrome failed to start once.** Diagnosed by the
   instrument on its recurrence: alive and slow. The bound is a minute.
7. **(closed) `retire`'s bound is conditional and stated, not enforced.**
   Enforced, and drilled on three arms.
8. **(moved) The four hardware items.** `ai_docs/plans/need_hardware.md`.
9. **(declined, non-goal)** Twenty-seven entries, unchanged. See
   `ai_docs/plans/non_goals.md`.
