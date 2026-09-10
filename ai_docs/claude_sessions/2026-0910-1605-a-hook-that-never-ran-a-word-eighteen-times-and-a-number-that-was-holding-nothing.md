# Session: a hook that never ran, a word eighteen times, and a number that was holding nothing

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-10 (follows "a-grep-that-became-an-arm-a-drain-nothing-bounded-and-a-record-that-outgrew-its-file")

## Ask

"Do all items in the Next list." Five items, all age 0, all raised by the
previous session.

The previous session's shape was **checks that were only true once**. This one
is the sequel that shape produces: **checks nobody had checked**. Four of the
five items were the previous session's own work looking back at itself — a
hook that was verified against everything except whether it runs, an arm
verified against Go and nothing else, a heuristic verified by there being only
one of the thing it guesses about, a bound verified by an argument.

| item | where | shape |
|---|---|---|
| 1 | .claude/ | a hook that never ran |
| 2 | wasm/verify | a word eighteen times, one of them a call |
| 3 | wasm/verify | a guess given something to stand on |
| 4 | internal/themehistory | an argument that turned out to be right |
| 5 | wasm/verify | a number that was holding nothing |

---

## Item 1 · a hook that never ran

The item expected to prove the hook works. It proved the opposite first.

`.claude/settings.json` was:

    {"matcher": "Bash",
     "hooks": [{"type": "command", "if": "Bash(git commit:*)", …}]}

**Both halves are wrong.** A PreToolUse `matcher` is a regex over the TOOL NAME
— `Bash`, `Write|Edit`. `Bash(git commit:*)` is permission-rule spelling and
matches no tool. And `if` is not a field the hook schema has, so the narrowing
everybody believed was happening was a key in a JSON object being ignored.

The file parsed. `jq -e` said so, which is why the previous session's
verification passed: **JSON validity is not schema validity**, and the check
that was run could only ever have found a syntax error. The schema was read
back out of the hooks reference embedded in the Claude Code binary.

Two probes settled the live state: a plain Bash call and a `git commit
--dry-run`, both with a non-doc file staged, both silent. The hook was firing
on nothing.

**The fix moves the filter into the script.** The matcher can only say "a Bash
call", so `session-doc-check.sh` now reads `.tool_input.command` off stdin —
reversing the old comment's "stdin is deliberately not read". The command line
answers *is this a commit*; the INDEX still answers *does it owe a doc*, which
is the better question and unchanged.

The filter is loose in one direction and says so: `git`, then anything that is
not a command separator, then `commit`. That catches `git -C dir commit` and
`cd x && git commit`, and also `git log --grep commit`, whose cost is one `git
diff --cached`. A jq that is missing falls back to scanning the raw payload —
looser again, and a hook that vanishes when a tool is missing is the failure
mode this whole file is about.

**`wasm/verify/hookconfig_test.go`** holds the settings file to the schema:
every key is one the schema has, every matcher is a tool name and not a
permission rule, every event name is an event, and a command naming a script in
this tree names one that exists and is executable.

**Verified end to end, in a fresh process**, which is the thing the item asked
for and which this session's own process could not do. `claude -p` with a
staged non-doc file and one allowed command came back:

> Nothing was committed — this was a dry run. If this session did real work,
> its `## Next` list lives only in context right now: `/sess-wrap` … or
> `/sess-save` … Want me to run one before you commit?

That is the script's sentence, arriving in another model's context and being
acted on.

**Break-tests — 3 run, 3 fired**: the `if` key, the matcher as a permission
rule, a command that is not there. Plus 9 pipe-tests over the script's six rule
rows, the jq-absent fallback and an empty payload.

---

## Item 2 · a word eighteen times, one of them a call

The `-z` arm parses Go and cannot see a shell script — and the file the rule
was written FOR is a shell script. `session-doc-check.sh` runs `git diff
--cached --name-only -z` and the only thing holding it was a comment.

`TestEveryGitListingInAScriptAsksForNulSeparatedPaths` lexes rather than greps,
and that file is why: **it holds the word `git` eighteen times and exactly one
of them is an invocation.** The other seventeen are the prose explaining why
the one asks for `-z`. A grep reports eighteen findings or none.

    sh   quotes, backslash escapes, redirections, heredocs, and `#` as a
         comment only where a comment can start — not inside a string, not
         mid-word
    js   // and /* */ removed, then every string and template literal lexed AS
         a shell command, because that is what a git call is from JavaScript

`git` must be in **command position**, after leading assignments and `env`/
`sudo`/`exec`. Quoted regions are lexed twice — once as part of the word, once
on their own — so `sh -c 'git ls-files --name-only'` is seen as the call it is.

**47 scripts, 1 invocation.** Three looseness concessions are written down
rather than left to be found: heredoc bodies are skipped, a line inside a quote
reports the line the quote opened on, and a string that happens to begin with
`git` is reported.

**Break-tests — 4 run, 4 fired**: `-z` dropped from the hook's own call (named
`:104`), a listing added to `run.sh`, a listing inside `sh -c "…"` — which is
the one that proves the quoted recursion — and the real call commented out,
which trips the walk-reaching arm. A fifth finding came out of BT1: the
reported command read `git diff --name-only 2 /dev/null`, so the lexer now
drops redirection targets and their file descriptors.

---

## Item 3 · a guess given something to stand on

`gitArgs` read a call to a bare `git` identifier as a git invocation wherever
it appeared. The defence was that the loose direction is the safe one.

That is true and it is not the whole cost. **A check that says something
confident and wrong is a check people stop reading**, and this one's value is
that it is believed when it fires — what it reports (a file silently missing
from a listing) is invisible by construction, so a reader has nothing else to
weigh a finding against.

Now two passes. The first finds every `func git` in the repository, holds each
to containing `exec.Command("git", …)`, and notes which directory it is in. The
second reads a bare `git(…)` as an invocation **only in a package that declares
one**. The heuristic is a premise with a check under it.

Still 16 invocations — the tightening lost no real call.

**Break-tests — 3 run, 3 fired.** A `func git` that runs `hg`. A bare `git()`
listing call in the package that has the helper (fires, correctly). And a bare
`git()` in a package with no helper — which **does not compile**, and that is
itself the answer: the whole worry class requires a declaration, and a
declaration is now named.

---

## Item 4 · an argument that turned out to be right

`batchRetireGrace` is five seconds because "the healthy case is microseconds to
low milliseconds, four orders of magnitude under this". Sound reasoning about a
pipe drain and a process exit, **never timed**.

It is **0.18–0.30ms** over four sets of seven — about 1/17000 of the grace. The
a priori figure was right to the order, and saying so is the point: a bound
justified entirely a priori is one nobody can tell has stopped being true.

**What is asserted is not the clock.** A healthy `git cat-file --batch` exits 0
because it noticed the EOF; a git the deadline killed does not exit 0 at all.
So the arm fires exactly when "no run that is working can reach it" stops being
true, and never because a machine was busy — a slow retire is still a 0 exit
right up to the deadline. The number is recorded, the outcome is asserted.

`internal/themehistory/timings_test.go` is the sibling record the item said did
not exist: `wholePackage` 1.15–1.27s, `wholeRun` 1.52–1.70s from a **built
binary** rather than `go run`, which folds a build into the reading.

A duplicate of `verifyTimingsTakenOn` rather than a shared package, and the
comment says why: two separate `package main` programs, and what has to stay in
step is a shape, not a value.

**One thing the record records is a gap it cannot close.** The previous session
wrote 1.87s for the same command with three fewer arms in it. Nothing was made
faster. That 60% is what a wall clock with no record beside it looks like from
the other side.

**Break-tests — 2 run, 2 fired**: the grace lowered to 1ns (the healthy git
comes back `signal: killed`), and an empty field in the record.

---

## Item 5 · a number that was holding nothing

The item asked whether the `128ms` earns its place, given that what it settles
survives the figure drifting by half.

**It does not, and nothing had to be re-measured to establish that.** The
question is not "how do we re-take it" but "what is it holding up", and the
answer is nothing that needs it:

    the walk is affordable      demonstrated by the test running it on every
                                green run
    the walk still goes all     asserted by foldMeasuredOn's six astral
    the way up                  counts, which a narrowing to the BMP fails
    what it costs while doing   verifyTimingsTakenOn.foldWalk, re-takeable by
    it                          running the test

All three are re-derived. The wall clock is not, and does not have to be. So
the figure stays where it was taken, marked as the reading a past decision was
made on, and the sentences it used to carry now stand on the arms. Pricing it a
second time would have been work to keep something honest that could instead
stop being asked to hold anything.

**And the record's own re-taking table was wrong.** It offered `go test -bench
. -run '^$' ./wasm/verify` for themenearmiss's 96.7us/7.4us/2530.0us. There are
no benchmarks in the package — they were written to take those three numbers
and removed once taken, and the instruction outlived the thing it instructed.
Those three are in exactly the position the `128ms` is, and the table now says
so instead of offering a command that does nothing.

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

    .claude/settings.json                      item 1
    .claude/hooks/session-doc-check.sh         item 1
    wasm/verify/hookconfig_test.go             item 1 (new)
    wasm/verify/gitscript_test.go              item 2 (new)
    wasm/verify/gitquoting_test.go             item 3
    internal/themehistory/timings_test.go      item 4 (new)
    internal/themehistory/main.go              item 4
    wasm/verify/inkglyph_test.go               item 5
    wasm/verify/timings_test.go                item 5

`go run ./internal/themehistory` reproduces the recorded histogram
**byte-identically against a stash of these changes**, with **nothing on
stderr**. **12 break-tests run, 12 fired**, plus 9 pipe-tests over the hook
script and one end-to-end run of a fresh Claude Code process.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

1. **(age 0 · value high) The hook schema is a table read out of a binary.**
   `hookEvents`, `hookEntryKeys` and `hookTypes` in hookconfig_test.go were
   copied from the hooks reference embedded in Claude Code 2.1.267. They are
   now a fact about that version sitting in a repository that will outlive it,
   and the failure direction is the bad one: a key added in a later release
   fails this test on a config that is CORRECT. Nothing in the repository
   records which version the tables came from, which is the same omission
   `verifyTimingsTakenOn` exists to end one file away.
2. **(age 0 · value medium) The lexer cannot see a heredoc body.**
   `skipHeredoc` walks past the body on the grounds that a heredoc is data, and
   for the one in this repository — a JSON literal — that is right. `ssh host
   <<EOF … git ls-tree … EOF` is not data, and it is the shape a deploy script
   takes. The body is already delimited, so lexing it as shell instead of
   skipping it is a small change; what stops it being obvious is that a
   heredoc fed to `cat` or `jq` would then be lexed as commands.
3. **(age 0 · value medium) `runsGit` is satisfied by the wrong shape.** It
   looks for `exec.Command("git", …)` anywhere in a `func git` body, so a
   helper that calls git once for an unrelated reason and does something else
   with its arguments passes. The check that would actually settle it is that
   the helper's variadic parameter reaches that call, which is a dataflow
   question a `go/ast` walk answers badly and `go/types` answers properly —
   and this package parses files rather than loading packages.
4. **(age 0 · value low) The two timings records are a copy.**
   `themehistoryTimingsTakenOn` and `verifyTimingsTakenOn` have the same five
   machine fields and the same reporting arm, and the reasoning for keeping
   them apart is written down in one of them. It is a good reason today. It
   stops being one at the third record, and the third record is what a fourth
   package with a wall clock in a comment produces.
5. **(age 0 · value low) `wholeRun` is a number nothing re-takes.** It is
   1.52–1.70s from a built binary, taken by hand with `date +%s%N` around a
   loop. Every other number in that record has a command in the table beside
   it that a person can run; this one has a recipe. The place it would belong
   is the same place the retire figure went — a test that produces the number
   as a by-product of asserting something falsifiable — and `go run` over the
   whole history inside `go test` is a 1.5s test for a comment.
