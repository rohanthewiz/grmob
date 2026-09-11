# Session: every band field holds its taking command, and a reader behind a build tag

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-11 · Previous:
`2026-0911-0316-the-loop-nine-iterations-about-numbers-that-were-not-readings.md`

## Ask

`/loop` — work the Next list, up to nine iterations, `/sl` in and `/sess-wrap`
out. This is iteration 1 and it worked **Next item 1**: every band field should
be held to having a taking command.

## What was built

`checkEveryBandFieldHasATakingCommand`, in `wasm/verify/copies_test.go`, read as
part of the records question on the shared repository parse.

    the bands       off the AST. A band field is one whose value opens with a
                    range recordedBand can read — the same test the verdict
                    makes, so a field this asks about is exactly a field
                    something compares a reading against
    the labels      off the record's own doc comment. A label is a field name
                    alone on a line at one tab, the command indented under it,
                    which is the form both tables already had
    both directions a field with no entry, an entry with no command under it,
                    an entry naming a field the record does not have, and an
                    entry naming a field whose value does not open with a band

Two supporting changes: `timingsRecord` now carries the record's doc comment
(`recordDoc`, which prefers a ValueSpec's comment to its GenDecl's so a record
moved into a `var` block is not reported as a record with no table), and
`recordLiteralFields` reads a record literal's field names and, of those, the
ones carrying a band.

## What it found

**Nothing, on its first run: ten band fields across the two records, ten with a
command.** The one real defect — `wholeFileInProcess`, added with a band and no
entry, which stood for six iterations — was found and fixed by the iteration
that raised this item, by hand, reading a table beside a literal.

That is the residue this repository normally declines a rule for, and the
argument for writing it anyway is in the function's own header: a corpus that is
clean because somebody just cleaned it is not a corpus that was never dirty, and
this fault recurs by the one action nobody treats as a change — adding a field.

**What the race suite found, which was a real defect.** `go test -race ./...`
failed to build: `undefined: recordedBand`, `undefined: recordedBandForm`. The
band reader lives in `wasm/verify/wholefileband_test.go`, which is
`//go:build !race` because the TestMain clock in it measures a figure a race
build is not a reading of — and a census that runs on every run cannot read a
declaration that does not exist under `-race`.

**This is the second iteration running with that exact defect**; the previous
one was `bandVerdictEnv`. The reader moved to `timings_test.go`, beside the
lever that moved for the same reason one iteration earlier, and the tagged file
now carries a pointer saying why neither is in it: the tag bounds what must not
RUN outside one command, and is not where a package keeps the things that read
its record.

## What was declined, with numbers

One entry added to `ai_docs/plans/non_goals.md`, which now has eleven
(counted by `grep -c '^## '` less the file's own two section headings — the
two session docs before this one each stated the figure one low):

    a declaration a census reads      the Go toolchain is the arm and it is
    is not held to living outside     already in the verification path. Two
    a build tag                       instances, both caught by `go test -race
                                      ./...` in the session that made them,
                                      with the file and line of every reader —
                                      a better finding than any parse of build
                                      constraints would produce

Also declined, by reference rather than by a new entry: holding a table entry's
command to RUNNING. That was measured one iteration ago at two lines, both real,
both fixed, and is already filed.

## Two predictions from the Next list, both refuted

The item was written by the session that would not work it, and it carried two
conclusions beside its instruction:

    "the cost is a walkQuestionBudget   it is not. A question on that walk is
    raise with its argument"            a subtest and the budget counts
                                        subtests; this is a reading inside the
                                        records question, which already had the
                                        literal and now has the doc comment
                                        beside it. The budget stays at six
    "and the registrations three        zero. No census asked for anything —
    censuses will ask for"              not the walk's own row, not the
                                        two-copy shapes, not the names in
                                        prose. The helpers it uses were
                                        already declared in this package

The instruction — fields off the AST, labels off the record's doc comment, ride
the existing parse — paid exactly as written. Nineteen iterations of this
pattern across two runs, and twenty now.

## Break-tests: six, every one read

    label removed                 names the field, quotes its band, shows the
                                  form to add
    label kept, command removed   "lists the field with nothing under it",
                                  which is the state worse than absent
    label renamed                 BOTH directions fire — the field is
                                  uncovered and the label is dead. Kept, and
                                  said out loud in the header: a rename is
                                  indistinguishable from a deletion plus an
                                  addition until somebody reads both
    label on a non-band field     quotes the value, says the range has to come
                                  first
    doc comment removed           one finding for the record, naming the four
                                  fields wanting an entry, rather than four
    band pattern broken           "no field in any of the 2 records has a value
                                  that opens with a band" — the over-nothing
                                  arm, which also catches the verdicts having
                                  silently stopped comparing

One of them changed the code: the first findings quoted the whole field value,
which for `wholeFile` is a 400-character paragraph. They quote the band prefix
now, and the wrong-way-round message truncates at 120 characters.

## Verification

Eleven paths, all green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      rc 0
    GOOS=js GOARCH=wasm go build ./...

**Figures, each package alone, every one in band — so nothing was re-taken:**

    wasm/verify wholeFile         2.866–2.963s   recorded 2.78–2.97s
    …wholeFileInProcess           2.684–2.775s   recorded 2.55–2.78s
    internal/themehistory         3.029–3.182s   recorded 2.86–3.22s
    …wholeRun, …batchRetire       in the band, verdict printed

Five files, +512 −55.

## Next

1. **(age 5 · value low→medium) Record which band ends have actually been
   reached.** Follows from the rule that a band's ends are readings, not
   choices: a field whose ends are not readings is the thing to find. Its cost
   has dropped this iteration — `recordLiteralFields` already reads every band
   off the AST on the shared parse, so the walk half is built; what is missing
   is where a "reached" mark would live, and a mark a session writes by hand is
   the same class of fault this iteration just closed.
2. **(age 9 · value low) `…TimingsTakenOn` is not the only record shape.**
   `affordedMeasuredOn` and `foldMeasuredOn` carry bands too. Value is still
   low and the reason is now written in code rather than only here: those
   records are re-derived by the run that reads them, so a number that moved is
   a finding about the data, and a taking command for them would be the command
   the reader has already run.
3. **(age 22 · value low) Every Next list in this loop was written by the
   session that would not work it.** Twenty iterations of evidence. Both
   conclusions on this iteration's item were refuted and its instruction paid
   in full, which is the same result as the nineteen before it.
4. **(declined, non-goal)** Eleven entries, including this iteration's. See
   `ai_docs/plans/non_goals.md`.
