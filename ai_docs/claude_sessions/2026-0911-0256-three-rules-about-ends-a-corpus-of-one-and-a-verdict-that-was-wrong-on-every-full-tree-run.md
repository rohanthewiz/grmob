# Session: three rules about ends, a corpus of one, and a verdict that was wrong on every full-tree run

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-11 · iteration 7 of a ten-iteration `/loop`
Previous: `2026-0911-0243-a-prediction-refuted-then-un-refuted…`

## Ask

Next item 1 (age 0, value medium): an unreached end is evidence of nothing —
record that, and leave `walkEnumerate`'s unreached floor alone.

---

## The bands thread, closed

Five re-takings across four iterations, and every one that went wrong went
wrong the same way. The three rules are now stated once, in
`verifyTimingsTakenOn`'s header where they govern every field:

    widen, almost never narrow    a ceiling nothing has reached costs a
                                  reader nothing. A tightened one costs a
                                  false verdict the first time the tail
                                  shows up, and walkParse's did, five runs
                                  after it was tightened
    an unreached end is           walkEnumerate's floor is 0.08s and seven
    evidence of NOTHING           runs read 0.09–0.10s. walkParse's ceiling
                                  looked exactly that way for twenty-six
                                  runs before one landed on it
    an end is a READING,          if an end is not a number somebody took it
    not a choice                  is a guess, and a guess at an end is where
                                  a false verdict comes from

With the corollary that was got wrong here: **a band that looks too wide is
cheap and a band that looks right is expensive.** Re-taking is for a reading
that falls outside. A reading comfortably inside is not an invitation.

One stale claim went with it — the header said the three walk depths "say
they are reproducible", which thirty-two runs took back last iteration.

## A check measured to nothing, and the reason is the result

The defect class fixed by hand all session: a sentence quoting a record's
numbers while the record said something else. The mechanical version is *a
sentence naming `record.field` must quote that field's range*.

At the comment group it reads nothing — 5 groups, 25 figures, **0 matching**.
Not 25 defects: the check is meaningless at that granularity, because a group
here is forty lines discussing a figure's history and the sibling package.
The same reading that killed the wall-clock rule.

At the sentence, **the corpus is one** — a history sentence with the field
backquoted. Zero defects.

The corpus is empty *by construction*: the doctrine "name the field, do not
restate the number" already won. The measurement is worth more than the rule
would be, because it says the doctrine is complete in the Go prose rather
than believed. Declined in `ai_docs/plans/non_goals.md`.

## The defect: both verdicts were wrong on `go test ./...`

Found by asking what the iteration-3 guards did not cover. They handle
`-short`, `-run`, `-count`, `-bench`, `-race` and the machine. They do not
handle the package not being alone.

**`go test` runs package binaries in parallel.** The bands were taken with
one package running by itself, which is what their method lines say.
Measured:

    wasm/verify            2.5–2.7s alone     2.98–3.10s in `go test ./...`
    themehistory wholeRun  1.4–1.6s alone     1.70s there

Both report OVER. On a path that runs in **every session's verification**. A
verdict wrong every time somebody runs the whole tree is the noise the file's
own header says argues for deleting the record, and it would teach a reader
to skip the line in the one case it is right.

A test binary cannot know what else `go test` is running, so the verdict is
now **asked for**: `GRMOB_BAND_VERDICT=required`, spelled out the way
`GRMOB_PER_OBJECT_FETCH` is and for the same reason — a typo should be a
setting that names itself rather than one that silently did nothing. Silent
by default. The reporting arm, which prints on every `-v` run, says how to
ask and states the condition.

**And `-race` caught the fix.** The constants were put in
`wholefileband_test.go`, which is `//go:build !race`, so the reporting arm
that names them stopped compiling under `-race`. Moved to a file that is
always built. That is the verification path doing the job the build tag made
necessary.

---

## Verification

Eleven paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      rc 0
    GOOS=js GOARCH=wasm go build ./...

**Break-tests, read:**

    default, `go test -v ./...`      0 verdict lines, where there were 3
    asked, one package alone         correct verdicts, both packages
    asked, `go test -v ./...`        still OVER — documented, not a bug: the
                                     lever says "this package alone" and the
                                     message repeats the condition
    GRMOB_BAND_VERDICT=yes           silent. Not truthy, by design
    -race before the constants moved undefined: bandVerdictEnv

Figures at the end, all in band:

    wasm/verify             2.859–2.884s   recorded 2.78–2.97s
    internal/themehistory   3.045–3.147s   recorded 2.86–3.22s
    …wholeRun, …batchRetire in band when asked

## Next

1. **(age 0 · value medium) The verdict is now opt-in, so nothing checks the
   bands on an ordinary run.** That is the right trade and it leaves a gap
   the last five iterations were about: five stale bands were found by a
   verdict that now has to be asked for. The cheap closing move is to put
   `GRMOB_BAND_VERDICT=required` into the verification the session doc
   already runs — one line in `wasm/verify/run.sh` or the session's own
   check-list — so the figures at the end of a session are compared by the
   machine rather than by eye, which is where this whole thread started.
2. **(age 1 · value low) Record the widest reading ever seen per field**
   rather than the method. Made sharper by the three rules: if an end is
   always a reading, then the ends ARE the extremes ever seen, and a field
   whose ends are not readings is the thing to find. Eight fields are in
   that state or unknown.
3. **(age 5 · value low) `…TimingsTakenOn` is not the only record shape** —
   and its value dropped this iteration. It exists to unblock the wall-clock
   rule, which iteration 1 measured at zero real defects in its residue.
   Unblocking a rule that finds nothing is poor value; worth doing only if
   the two structures are being edited anyway.
4. **(age 18 · value low) Every Next list in this loop was written by the
   session that would not work it.** Seven for seven. This one was right and
   small, and the iteration's real finding came from asking a question the
   item did not pose — what the guards did not cover.
5. **(declined, non-goal)** Two entries added tonight. See
   `ai_docs/plans/non_goals.md`.
