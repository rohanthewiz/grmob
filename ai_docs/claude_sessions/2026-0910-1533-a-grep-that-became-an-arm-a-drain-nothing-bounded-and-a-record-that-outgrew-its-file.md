# Session: a grep that became an arm, a drain nothing bounded, and a record that outgrew its file

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-10 (follows "a-quoting-rule-that-dropped-a-file-in-silence-a-directory-fixed-at-birth-and-one-sentence-for-four-shapes")

## Ask

"Do all items in the Next list." Five items, all age 0, all raised by the
previous session in this same conversation.

The shape this session found is **checks that were only true once**. The
previous session's headline finding came out of a grep somebody ran on an
afternoon; the deadline-less drain was correct for the one process it was
written for; the timings record was accurate about the file it lived in on the
day it was written. Each item is the same move: take a fact that happened to
hold and make something hold it.

| item | where | shape |
|---|---|---|
| 1 | wasm/verify | a grep that became an arm |
| 2 | internal/themehistory | a drain nothing bounded |
| 3 | internal/themehistory | a classification made by default |
| 4 | wasm/verify | a record that outgrew its file |
| 5 | .claude/ | a habit that became a mechanism |

---

## Item 1 · a grep that became an arm

The item feared a second `-z` gap somewhere in the repository. **There is
none.** The only other git readers are `checknumbering_test.go`'s two
`ls-files -z` — already right, with a comment saying why — and a
`check-ignore -q` that parses an exit status and no output.

So the deliverable is not a fix. `TestEveryGitListingAsksForNulSeparatedPaths`
parses every Go file the repository enumerates, finds every `exec.Command("git",
…)` and every call to a package-local `git(…)` helper, and requires `-z` of any
whose arguments make git write paths:

    the subcommand   ls-files, ls-tree, status
    the flag         --name-only, --name-status, --porcelain

Everything else is left alone on purpose — `rev-parse` answers with an oid,
`cat-file` with an object, `log --format=%H` with whatever the format says.

**Parsed and not grepped**, because the question is about one call's argument
list: a grep for `ls-tree` finds the prose, the comment above the call, and the
call, and cannot tell which of them has a `-z` in it.

**It first reported two invocations**, both inside the file that does the
enumerating. `citingFiles` returns the files that carry a CITATION, not every
file — and a git call in a file with no "check N" in it is precisely the one
that would be missed. Switching to its `considered` map took it to **16**.

**Break-tests — 2 run, 2 fired.** `-z` dropped from `ls-tree` and from `git
status`, each naming the exact file and line.

---

## Item 2 · a drain nothing bounded

`retire` closed stdin, drained stdout and Waited. Both steps terminate for a
`git cat-file --batch`, and neither is guaranteed to: the drain ends when the
pipe reaches EOF, which is when the process exits, and the process exits when it
notices the EOF on its stdin. A git wedged for any other reason holds the drain
— **and the drain holds `blobsMu`**, which is the whole run.

`batchRetireGrace` is five seconds, and the comment is careful that this is not
a tuned timeout: it is a bound on "a healthy git has already gone", and the
healthy case is four orders of magnitude under it. No working run can reach it.
A variable rather than a const so the arm can lower it to 150ms — a test that
waited out the real one would be a five-second test asserting a constant.

The arm builds a `batchReader` by hand around `sh -c 'sleep 60'`, which is the
shape retire has to survive and which git will not be talked into: a child that
keeps its stdout open and ignores its stdin. It asserts the premise first
(`Signal(0)` after closing stdin), then that retire returns, then that
`cmd.ProcessState` is set — because Kill signals a process and only Wait reaps
it.

**Break-tests — 2 run, 2 fired.** The old retire held for the full 10s the test
waits; a `return` after the Kill left a zombie and the arm named it.

---

## Item 3 · a classification made by default

git answers `<name> ambiguous` for an object name matching more than one
object, and it is a complete one-line response exactly as `missing` is. It was
being classified as desynchronising by falling through `len(fields) != 3` —
correct, and by accident.

Now a row in the table with the two costs written down:

    read as desynchronising     one git process, once, and a run that carries
                                on with correct bytes
    read as complete when it    a body that was never there is skipped, and
    is not                      every response after it belongs to the request
                                before it

And why it is free rather than merely cheap: every name asked for here is
`<full sha>:<path>` built from `ls-tree`'s own output, so nothing in this walk
can produce an ambiguous one.

**Break-test — 1 run, 1 fired.** `ambiguous` recognised alongside `missing`.

---

## Item 4 · a record that outgrew its file

`affordedTimingsTakenOn` was written yesterday-in-session-time describing
"every wall-clock number in this file". That was **already untrue when it was
written**: `inkglyph_test.go` states the astral fold walk's cost twice.

Renamed `verifyTimingsTakenOn` and moved to its own `timings_test.go`, because
a record cited by two files that lives inside one of them reads as that file's
property and the next person adding a timing to a third does not find it. Six
citation sites updated; the two `128ms` mentions now cite it.

**What the item warned about is the part worth recording.** "A record
attributing numbers nobody re-took is the same unattributed number with a name
on it." So the survey came first, and it was narrower than expected:
`foldMeasuredOn` already records what moves ITS numbers (the Unicode/ICU
build), which leaves exactly **one** machine-dependent number outside
`themenearmiss_test.go`.

`foldWalk` is `0.40–0.52s over seven runs, node v22.12.0` — the enclosing test,
re-measured. The `128ms` is the astral walk alone and is **explicitly not
re-taken**, with the reason in the field's own comment: isolating it again
means either instrumenting a test to print a number for a comment or running a
second copy of the walk outside it, and both are worse than an honest note.
What is re-derivable without either is the bracket, so that is what the record
vouches for.

---

## Item 5 · a habit that became a mechanism

The item weighed both ways and did not decide: "a hook that refuses a commit is
a hook people disable; a note in a file is exactly what did not stop
`8d14384`."

The resolution is the option that answers both. `.claude/hooks/session-doc-
check.sh` is a `PreToolUse` hook on `Bash(git commit:*)` that emits
`additionalContext` — a sentence into the model's context before the commit
runs — and **exits 0 on every path**, including the ones where git itself
failed. It never blocks, so there is nothing to disable.

It reads the INDEX rather than the command line, which is the better answer:
`git commit -m "..."` says nothing about what is in it.

    nothing staged             silent. -a and <path> stage inside the commit
    a doc is staged            silent. The commit carries its own doc
    only session docs staged   silent. A doc-only commit IS the doc
    otherwise                  say so

**One honest lossy step**, and it is this session's own subject arriving in a
shell script. The hook asks for `-z` and then turns NUL records into lines,
because POSIX sh has no NUL-safe read — so a path with a real newline splits in
two. The consequence is bounded: each half is still classified, so the worst
case is a warning nobody needed. A hook that only prints a sentence can afford
that; the walker whose same mistake dropped a file from a table in silence
could not. Written into the comment rather than left for somebody to find.

**Pipe-tested against all five index states**, plus a path with a space, plus
`python3 -m json.tool` on the warning. `jq -e` confirms the settings schema.

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

    internal/themehistory/main.go              items 2, 3
    internal/themehistory/main_test.go         items 2, 3
    wasm/verify/gitquoting_test.go             item 1 (new)
    wasm/verify/timings_test.go                item 4 (new)
    wasm/verify/themenearmiss_test.go          item 4 (record moved out)
    wasm/verify/inkglyph_test.go               item 4
    .claude/hooks/session-doc-check.sh         item 5 (new)
    .claude/settings.json                      item 5 (new)

`go run ./internal/themehistory` reproduces the recorded histogram
**byte-identically** with **nothing on stderr**. `wasm/verify` is 1.86–1.99s
against the 1.81–1.92s now written into `verifyTimingsTakenOn`;
`internal/themehistory`'s own tests are 1.87s with three new arms.
**8 break-tests run, 8 fired.**

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

1. **(age 0 · value medium) The hook has never fired.** `.claude/` held no
   settings file when this session started, so the settings watcher is not
   watching it and the hook will not run until `/hooks` is opened once or the
   session restarts. The script is verified against all five index states by
   pipe-test and `jq -e` confirms the schema — what has not happened is the
   end-to-end proof that Claude Code loads it, matches `Bash(git commit:*)` and
   puts the sentence into context. That is one commit away and this doc is
   being written before it.
2. **(age 0 · value medium) The `-z` arm cannot see anything but Go.**
   `session-doc-check.sh` is a git listing in a shell script and follows the
   rule by hand; `run.sh` in three platform directories, `build.sh`, and any
   `.mjs` that ever shells out are all outside what a go/parser walk reaches. A
   grep-based second pass over non-Go files would be loose in a way the parse
   is not — it cannot tell a comment from a call — but "loose" and "absent" are
   not the same, and the file the rule was written for is now one of the
   unchecked ones.
3. **(age 0 · value low) `gitArgs` matches a bare `git` identifier.** Any
   function called `git` reads as a git invocation, wherever it is and whatever
   it takes. It is the safe direction — the worst case is a message naming a
   call somebody can look at — and it is a heuristic standing in for resolving
   identifiers across packages. If a second `git`-named helper ever appears
   with different semantics, the arm will say something wrong about it
   confidently.
4. **(age 0 · value low) `batchRetireGrace` is five seconds of judgement.** The
   argument is that no healthy shutdown is within four orders of magnitude of
   it, which is true and unmeasured — the healthy path has never been timed,
   only reasoned about. A run that recorded how long retire actually takes
   would turn the bound into a measurement, and the place for that number is
   `verifyTimingsTakenOn`'s sibling in `internal/themehistory`, which does not
   exist.
5. **(age 0 · value low) The `128ms` is bracketed, not re-taken.**
   `verifyTimingsTakenOn.foldWalk` records the enclosing test and says plainly
   that the sub-figure inside it was not re-measured. That is honest and it is
   still a number from an unknown day. The only ways to fix it are to
   instrument the test or to run a second copy of the walk, both of which this
   session declined for good reasons — so what is left is deciding whether the
   number earns its place at all, since what it settles ("the bound was
   affordable") survives the figure drifting by half.
