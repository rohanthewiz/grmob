# Session: a table with no version, a heredoc nothing ran, and a count that was a bound

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-10 (follows "a-hook-that-never-ran-a-word-eighteen-times-and-a-number-that-was-holding-nothing")

## Ask

"Do all items in the Next list." Five items, all age 0, all raised by the
previous session.

The previous session's shape was **checks nobody had checked**. This one is
what that produces once every check has been checked: **premises standing on
one instance**. A schema table true of one build. A heredoc rule written for
the one heredoc in the tree. A wrapper premise held up by there being one
wrapper. A duplication argument written for two copies. And, running underneath
three of them, a second shape — **a number sitting in the position a
measurement goes, that was never one**.

| item | where | shape |
|---|---|---|
| 1 | wasm/verify/hookconfig | a table with no version |
| 2 | wasm/verify/gitscript | a heredoc nothing ran |
| 3 | wasm/verify/gitquoting | a premise with one instance |
| 4 | wasm/verify (new) | an argument written for two |
| 5 | internal/themehistory | a recipe where a command goes |

---

## Item 1 · a table with no version

`hookEvents`, `hookGroupKeys`, `hookEntryKeys` and `hookTypes` were copied out
of the hooks reference embedded in Claude Code 2.1.267 and nothing recorded
that. **The failure direction is the bad one**: a key a later release ADDS is a
key the table has never heard of, so a config that is correct fails a check
whose whole purpose is to catch one that is wrong.

That direction cannot be fixed and the check should not be softened — this
repository's settings.json is written by hand, and `if` sat in it for a session
doing nothing. What was missing is not tolerance, it is **the two facts that
tell a reader which of the two things a finding is**.

`hookSchemaReadFrom` carries the build, the date and where in it. `claude
--version` is read once — `sync.OnceValue`, `LookPath`, a five-second deadline,
and a missing binary is an ANSWER rather than a failure, because CI has no
Claude Code and the config check is still worth running there. `hookSchemaNote()`
ends all four messages whose authority is a table, in three shapes:

    no claude here      either a mistake or a key a release added; ask a
                        session for its hooks reference
    same build          "This is not a version difference" — the schema has
                        not moved under this file, so the finding is about
                        the file
    a different build   "Check the version before changing the file"

The header of that file says this package has no business launching an agent.
`--version` is on the other side of that line and the comment says why: it
prints one line and exits, starts no session, reads nothing of this
repository's. **The version difference is reported and never asserted**, for
the reason the timings records give about a machine — an arm over it fails on
every computer whose install has moved on, which is every computer eventually.

**Break-tests — 4 run, 4 fired**: a version that is a word (`latest`), a date
that is not one, an emptied key table, and `if` put back into settings.json —
which showed the note arriving under the finding with the right branch chosen.

---

## Item 2 · a heredoc nothing ran

`skipHeredoc` walked past the body on the grounds that a heredoc is data. True
of the one heredoc in this repository, which is a JSON literal, and not true of
the shape:

    ssh host <<EOF
    git ls-tree --name-only HEAD
    EOF

**The body is already delimited, so lexing it costs nothing but the decision of
when** — and skipping always is an answer to that decision, not the absence of
one.

The discriminator is the command, because that is what the question actually
is: *does anything run this text*. `shellFromStdin` is eight names matched on a
word's BASE, and **every word of the command is scanned rather than the first**
— `docker exec -i c bash <<EOF` and `sudo -u x ssh host <<EOF` are both the
shape and neither has the shell in command position. A body fed to `cat` or
`jq` stays data, which is what keeps a JSON literal from being lexed as
commands.

`skipHeredoc` became **`heredocBody`**, returning the body, the line it starts
on and the last byte consumed. The line matters as much as the text: a finding
inside a body has to report its position in the enclosing FILE.

**And the lexer's rules were exercised by nothing.** This repository has one
git call in a script, so command position, quoted recursion, redirection
targets, `#`-that-is-not-a-comment and now heredocs were all checked by hand
once, by editing scripts and reverting them.
`TestTheScriptLexerFindsAGitCallInTheShapesThatHideIt` is **15 shapes**,
including both directions of the new rule, and `TestALineNumberSurvivesAHeredoc`
holds the counting across a body.

**Break-tests — 3 run, 3 fired**: a listing inside an `ssh` heredoc added to
`build.sh` (reported at build.sh:3), the discriminator forced to always-true
(the JSON case fires), and the line counter dropped from the body walk.

---

## Item 3 · a premise with one instance

`runsGit` asked whether `exec.Command("git", …)` appeared anywhere in a `func
git` body. That is the right FIRST question and not the whole one:

    func git(args ...string) (string, error) {
        root, _ := exec.Command("git", "rev-parse", "--show-toplevel").Output()
        return run(filepath.Join(string(root), args[0]), args[1:]...)
    }

passes, and every finding about that package is then **a git command line
assembled out of arguments git never saw** — not merely loose, about a
different program.

**`whyNotAGitWrapper`** asks the question the caller-side reading actually
rests on: does the helper's VARIADIC PARAMETER reach the call. The variadic
names start tainted, an assignment whose right-hand side mentions a tainted
name taints its left-hand names, and the premise holds if a tainted name
reaches `exec.Command("git", …)` past the program name **or is assigned into a
field of the `*exec.Cmd` that call produced** — `cmd.Args = append(cmd.Args,
args...)` is an ordinary wrapper and reaches git just as surely.

It is a dataflow question, which `go/types` answers and a parse does not; this
package parses FILES rather than loading packages, deliberately, because it
walks revisions and generated trees that need not build. So what is here is the
intraprocedural syntactic approximation, with its looseness written down: no
control flow, no shadowing, both in the direction of ACCEPTING a helper.

**Still 16 invocations** — the tightening lost no real call.
`TestWhatCountsAsAGitWrapper` covers seven shapes, four of which this
repository does not contain and one of which is the item's own example.

**Break-tests — 2 run, 2 fired**: seven table shapes as an arm, and
themehistory's real helper rewritten to run `git status` and ignore its
arguments — which produced both the wrapper finding and, one line later, the
`-z` finding about the wrong call that the old check would have produced in
silence.

---

## Item 4 · an argument written for two

The two timings records are the same struct twice, and the second says why: two
separate `package main` programs, and an `internal/timingrecord` imported by
both would be a package existing so that two structs could be one.

**That is a good reason, and it is a reason about two.** At three it stops being
a trade between a package's cost and one duplicate and becomes a shape kept in
step by whoever remembers to.

`wasm/verify/timingsrecords_test.go` does not unify them. It writes the trigger
down: every `…TimingsTakenOn` in the repository is held to the five machine
fields **at the right types**, and to having
`TestTheTimingsInThisPackageSayWhichMachineTheyCameFrom` in its package — and
the check **fails at the third**, quoting the copy argument back and asking for
either an extraction or a written reason beside it.

The values are deliberately not compared: they are different numbers about
different machines-at-different-moments and were never meant to match. What has
to stay in step is the shape. The record's own arm holds it to being filled in.

`timingsRecordCopies = 2` is not a limit for its own sake — it is the number
the reasoning was written about, which is why raising it is asked to come with
a sentence rather than just a digit.

**Break-tests — 3 run, 3 fired**: a third record, the reporting arm renamed out
of a package, and `goVersion` renamed in one record only.

---

## Item 5 · a recipe where a command goes

`wholeRun` was `go build -o th && ./th`, timed by hand with `date +%s%N`. Every
other number in the two records is produced by something a person can run; this
one had a recipe.

The 128ms in inkglyph_test.go resolved the same shape by establishing it was
holding nothing up. **This one is holding something up** — it is the
after-figure in `blob`'s cost argument, which is the reason this program is
shaped the way it is. So it went where the retire figure went: a test that
produces the number as a by-product of asserting something falsifiable.

`batchReader` counts `reads`; `batchesStarted` counts processes, and lives
outside the reader because a desync RETIRES and replaces one — a per-reader
field would be reset by the event worth noticing.
`TestTheWholeWalkGoesRoundOneBatchProcess` runs the real walk and asserts the
structural half of the argument, which was asserted nowhere:

    one process        batchesStarted moves by exactly one
    it served the run  the survivor's fetch count clears a floor, dead is nil
    the table came out the run's stdout carries the headline

The clock falls out: **1.49–1.61s over seven runs, in process, 2906 objects,
1 process**. Taken in-process rather than from a binary — the difference is one
exec and one dynamic link, 1.48–1.54s for `./th`, inside either spread — and
what it buys is a line in the re-taking table.

### Three numbers, reconciled

**"Around forty-four hundred" was a bound.** Eighty-eight commits times up to
forty-nine files, arithmetic away from the source, written in the position a
count goes. The walk fetches **2906**, and the reader counts them on every run.
The bound was right about the order and it was not a measurement, and the
comment could not tell the two apart.

**"1.8 seconds" is dropped rather than reconciled.** It was in main_test.go for
the same walk the record now says is 1.49–1.61s. An unattributed number and an
attributed one are not two readings that can be held against each other, which
is what both timings records exist to say.

**"Thirty-one seconds" survived.** It is the one figure here that cannot be
re-taken by running anything — the code that cost it is gone — but BT12 broke
`blob` into retiring its reader on every fetch, which is one process per
object, and that run took **31.8s** on the machine the record names. A number
carried since the session that deleted the shape it measures, standing up to
being re-created on purpose.

**Break-tests — 2 run, 2 fired**: the reader retired per fetch (2906 processes,
31.8s, and the survivor serving 1), and the table's headline changed.

---

## Verification

Eleven paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      rc 0
    android ./gradlew compileDebugKotlin --offline
    android ./gradlew :app:fetchComposeLayoutSources --offline
    GOOS=js GOARCH=wasm go build ./...

    wasm/verify/hookconfig_test.go          item 1
    wasm/verify/gitscript_test.go           item 2
    wasm/verify/gitquoting_test.go          item 3
    wasm/verify/timingsrecords_test.go      item 4 (new)
    internal/themehistory/timings_test.go   items 4, 5
    internal/themehistory/main.go           item 5
    internal/themehistory/main_test.go      item 5
    wasm/verify/timings_test.go             item 5

`go run ./internal/themehistory` is **byte-identical to HEAD** with **nothing on
stderr** — compared against a scratch worktree rather than a stash, because the
counters touch the fetch path. **14 break-tests run, 14 fired.**

Test time grew and is recorded rather than absorbed: `wasm/verify` 1.81–1.92s →
**2.51–2.58s**, `internal/themehistory` 1.15–1.27s → **2.84–3.08s**. The second
is accounted for in the record — the old figure plus the whole-walk arm — and
kept apart from the unexplained 60% gap sitting in the same comment, because a
gap with an explanation and a gap without one are different findings.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

1. **(age 0 · value medium) The version comparison cannot say which way.**
   `hookSchemaNote` compares `claude --version` against the recorded build with
   string equality, so `2.1.267` against `2.1.300` and against `2.0.9` are the
   same answer: "a different build". They are opposite findings — one means a
   release may have added the key, the other means the reader is on an install
   older than the tables and the key cannot be new. The note tells them to
   check the version, which is the right instruction and is also the check
   declining to do the comparison it just made. Ordering two dotted versions is
   twenty lines and no dependency.
2. **(age 0 · value medium) `wholeWalkReadsFloor` is a fact about today.**
   1000, chosen against a walk that fetches 2906. What it is guarding is real —
   "one process served eleven objects" would pass the process count while
   saying nothing — but the number is a third of a reading taken once, in a
   file whose entire subject is numbers taken once. If `core/` halves, the floor
   silently stops discriminating; if the walk breaks in a way that still fetches
   1200, it passes. The shape that would settle it is a floor derived from the
   commit count the test already reads.
3. **(age 0 · value medium) The thirty-one seconds is corroborated by
   something that is not in the repository.** BT12 re-created the per-object
   shape and got 31.8s, and what records that is a sentence in a comment saying
   so. That is better than the unattributed number it replaced and it is
   exactly the shape five items were spent removing: a reading nothing can
   re-take, believed because a past session says it took it. The honest options
   are a skipped-by-default arm that can re-create the shape on demand, or
   deleting the figure — and the second is worth considering, since what the
   comment needs is the RATIO and the ratio is now two measured numbers.
4. **(age 0 · value low) The whole-walk arm is 1.5s on every green run.**
   It doubles this package and is paid again under `-race`, on every `go test
   ./...` anybody runs, to assert a claim that changes about once a year. It
   also needs a real history and skips below fifty commits, so on a shallow
   clone the assertion is not made at all and the wall clock is not taken —
   which is correct and means the number in the record can only ever be re-taken
   on a full checkout.
5. **(age 0 · value low) `timingsRecordCopies` fails a correct third record.**
   The same failure direction item 1 is written against: a legitimate addition
   fails a check, and the resolution is a human reading the message and raising
   a constant. It is deliberate here — the failure IS the prompt, and the
   message asks for a reason rather than a digit — but it is worth naming as a
   decision rather than leaving it to be rediscovered as an inconsistency.
6. **(age 0 · value low) `batchesStarted` never resets.**
   A package-level atomic counting for the lifetime of the test binary. The one
   test that reads it snapshots before and after, which is right, and a second
   test wanting a process count would have to do the same or read this one's
   work. Trivial today; the kind of global that is only trivial while there is
   one reader.
7. **(age 0 · value low) The taint walk is a hundred lines of `go/types`
   answered badly.** `whyNotAGitWrapper` approximates a dataflow question
   because this package parses files rather than loading packages. That reason
   is written down once and is now load-bearing for two checks. It is the right
   trade — loading packages would mean every revision and generated tree has to
   build — but the approximation will keep growing a case at a time, and the
   place to notice that is the third one, not the sixth.
