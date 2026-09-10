# Session: a quoting rule that dropped a file in silence, a directory fixed at birth, and one sentence for four shapes

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-10 (follows "a-next-list-that-was-already-done-a-choice-made-by-sort-order-and-forty-four-hundred-processes-for-sixteen-rows")

## Ask

"Do all items in the Next list." Six items, all age 0 — the previous session
rebuilt its list from code because `8d14384` shipped without a doc, so nothing
was carried in and nothing has lapsed.

Then two corrections from the user, both about scope rather than code: the
`/commit` change belonged to this project and not to every project, and
`.claude/` should be tracked with local settings held out.

The shape this session found is one level up again. The last two were about
arms that never run and decisions made in passing. This one is about
**premises that were true when they were written**: a guard looking for a
newline in a spelling that no longer had one, a working directory that was
benign because of the order two tests happen to run in, and a sentence that
described the only case anybody had ever produced.

| item | where | shape |
|---|---|---|
| 1 | `.claude/commands/commit.md` | a list that is lost at the moment it is not written |
| 2 | wasm/verify | seven numbers and no computer |
| 3 | internal/themehistory | a reader with no way to die |
| 4 | internal/themehistory | a directory fixed at birth |
| 5 | internal/themehistory | a quoting rule that dropped a file in silence |
| 6 | internal/themehistory | one sentence for four shapes |

Items 3, 4 and 5 turned out to be one mechanism, and 5 turned out to be a live
bug rather than the dead branch the list described.

---

## Item 5 · a quoting rule that dropped a file in silence

The item said: `blob`'s newline fallback has never run, the state is makeable,
write the recipe. Making the state was the finding.

`git ls-tree -r --name-only` **C-quotes** a path it cannot write literally —
and the rule is not the one this file already knew about:

    on disk           core/a<LF>b.go    core/q"x.go      core/ä.go     core/a b.go
    ls-tree           "core/a\nb.go"    "core/q\"x.go"   "core/\303\244.go"  core/a b.go
    status --porcelain                                                  "core/a b.go"
    either, with -z   core/a<LF>b.go    core/q"x.go      core/ä.go     core/a b.go

`main_test.go` had argued this at length for `git status --porcelain -z` — a
whole paragraph about why a quoted path "matches nothing coming out of ls-tree
or off disk". The argument stopped at the one command. The other two readings
of a path in the package kept the default, and the two commands do not even
quote the same things: `status` quotes a space and `ls-tree` does not.

**What that cost.** A quoted name ends `.go"` and not `.go`, so it fails
leavesAt's suffix test and the file is **dropped with nothing on stderr**.
Demonstrated in a throwaway clone with `SpacingScale` moved into
`core/a<LF>b.go`:

    new walker   80 leaves, stderr empty
    old walker   76 leaves — "17 of the 17 added leaves and 1 removed any"

A commit that removed four leaves, in a table whose headline finding is that
nothing ever has. Silently, in the one direction this whole file exists to
notice.

**And the guard the item was actually about.** `blob` sends a path holding a
newline to a `cat-file -p` of its own. `"core/a\nb.go"` holds a backslash and
an `n`. The guard was never wrong; it was being handed a spelling with nothing
left in it to guard against.

So: `treePaths`, one function, `-z`, split on NUL, used by all three call
sites. And the guard is reachable now — which matters, because asking the
batch for such a path is not a failed request but a **desynchronised stream**:

    written:  HEAD:core/a<LF>b.go<LF>
    read:     HEAD:core/a missing<LF>     ← returned as the error
              b.go missing<LF>            ← nobody asked; nobody reads it

The second line answers the next request. `TestAPathWithANewlineInItGoesRound-
TheBatch` asserts both halves — the bytes come back, and the reader that was
NOT used is still usable — because a run where the fallback silently stopped
firing would pass the first half on the very request that broke the second.

**Break-tests — 2 run, 2 fired.** The fallback removed (the fetch fails); and
the fallback removed with the first assertion made non-fatal, which reproduced
the desync exactly: `HEAD:core/theme.go` answered by the leftover `b.go
missing`.

---

## Item 4 · a directory fixed at birth

`git cat-file --batch` resolves `<rev>:<path>` from the top of the tree, so
only repository DISCOVERY depends on cwd — and discovery is which repository.
The batch process's directory was whatever the first fetch in the run happened
to see, and every test `t.Chdir`s to the root before fetching. `t.Chdir`
restores; the git process keeps what it was born in.

"Benign because of the order two tests happen to run in" was the item. The
break-test says it is worse than that. With the comparison removed, a fetch of
`core/theme.go` from a scratch repository holding 50 bytes came back as
**45,395 bytes of grmob's real `core/theme.go`** — not an error, the wrong
file.

`cmd.Dir` is stated rather than inherited, compared on every fetch, and a
caller that has moved gets a fresh reader. One `getcwd` per fetch against a
pipe round trip is not measurable at 4400 fetches.

**Break-test — 1 run, 1 fired**, with the 45KB in the message.

---

## Item 3 · a reader with no way to die

Every error but `missing` left the stream at an unknown offset and the reader
was a package-level value that lived for the run. Now:

    errMissing   git's complete one-line answer. Costs the caller its file and
                 the next caller nothing. The reachable one.
    errDesync    a header that is not three fields, a size that will not parse,
                 a body that ends early, a pipe that closed, a request that
                 could not be written. Sets `dead`; blob retires the process
                 and starts another.

The trade is one process against a table assembled out of misaligned bytes, and
it is only affordable because a dead reader can now be replaced — which is also
why git's `ambiguous` is treated as desynchronising rather than recognised:
being wrong in that direction costs one process, and being wrong in the other
costs every response after it.

`readResponse` is a function of a `*bufio.Reader`, so the five shapes are a
table over hand-built streams rather than five states of a git nobody can make
misbehave. What is asserted is the classification, not the wording.

The one shape that IS makeable is the process dying, and
`TestAKilledBatchIsReplacedRatherThanReadFrom` runs the whole path: a `missing`
first (which must NOT cost a process — asserted by pid), then a kill, then the
desync error, then `dead`, then a correct fetch from a different pid.

**Break-tests — 3 run, 3 fired.** Never mark dead; mark dead on every error
including `missing`; classify `missing` as a desync.

---

## Item 6 · one sentence for four shapes

`Shadowed` records a bare name parsed twice **wherever** the declarations were.
Both readers of it — the command's stderr line and `reportNested`'s sentence —
said "declared both in core/ and below it". That is the union probe's case, and
it is the only one this repository would ever produce, which is exactly why the
other three went unnoticed: nobody has read the sentence and found it naming
the wrong directory.

`shadowKind` in `main.go`, shared by both readers — one rule, one copy, which
is the lesson the previous session's item 5 wrote into this same file:

    core/theme.go + core/zsub/x.go   both directly in core/ and below it
    core/a/x.go   + core/b/x.go      2 different directories below core/, none
                                     of them core/ itself
    core/x.go     + core/y.go        directly in core/ — one package, one name
                                     twice. Does not compile; reachable at a
                                     revision caught mid-refactor
    core/x.go     + core/x.go        in ONE file. go/parser reads it; the
                                     compiler does not accept it

The last two are why `Shadow.Files` is recorded as parsed rather than
deduplicated: collapsing it makes the bottom row print as the one above.

Demonstrated end to end in a throwaway clone, all three reachable ways:

    SpacingScale is declared 2 time(s) directly in core/, which is one package
    declaring one name twice (declared in core/dup.go, core/theme.go; …)

    OnlyBelow is declared in 2 different directories below core/, none of them
    core/ itself (declared in core/asub/x.go, core/bsub/x.go; …)

**Break-tests — 2 run, 2 fired.** The old one-sentence-for-everything restored,
which failed on exactly the four shapes it got wrong; and the one-file branch
removed.

---

## Item 2 · seven numbers and no computer

The item said the recorded 1.88s is from another machine. Chasing it, `1.88`
appears in no file — it is a number the session docs recorded. What IS in
`themenearmiss_test.go` is seven or so wall-clock numbers in prose (`685ms`,
`112ms`, `2530.0us`, `5.5s`, `1.8s`, `0.78s`) and two provenance phrases that
name nothing checkable: "on an M3" and "on this machine".

`affordedTimingsTakenOn` — `machine` for a reader, four fields the runtime
answers for itself, and the whole-file range. Both phrases now cite it.

The honest number is the one worth writing down: `go test ./wasm/verify`
spreads **1.81–1.92s over seven runs on one idle machine**, which is 6% wide by
itself. So a 2.0s run held against a 1.88s written down somewhere distinguishes
nothing — one machine's own spread, a regression, or a different computer, and
the number alone cannot say which. The spread is why the record is a RANGE; the
machine is why it is a record.

The arm is a `Logf` and the file says why: a timing cannot be an assertion, and
neither can the machine, because that would fail on every computer that is not
this one. The one assertion is that the record has no empty field — a fact
about the source, not the machine.

**Break-tests — 2 run, 2 fired.** A zeroed field (fails); a wrong GOARCH
(reports "GOARCH arm64 against amd64" and passes, which is the design).

---

## Item 1 · a list lost at the moment it is not written

Not a code item. `/sess-wrap` makes the doc and the commit one action; nothing
noticed a `/commit` that happened without one.

Written first into `~/.claude/commands/commit.md`, and **that was the wrong
place** — the user's correction, and a fair one: a check about
`ai_docs/claude_sessions/` is about this repository. Restored to its original
one line and moved to `.claude/commands/commit.md`, where it also gets to name
`8d14384` as the concrete case, which the global version could not.

`.claude/` is tracked. `.gitignore` gained a note in the file's own voice: the
directory holds this repository's slash commands and is as much part of how it
is worked on as `build.sh`; `settings.local.json` is where permission grants
land as they are approved, so it differs per person and would conflict on every
pull. Proved with a real probe file rather than by reading the rule —
`git check-ignore -v` names `.gitignore:39`.

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

Three source files, +1017 −64, plus `.claude/commands/commit.md` and a
`.gitignore` stanza.

    internal/themehistory/main.go              items 3, 4, 5, 6
    internal/themehistory/main_test.go         items 3, 4, 5, 6
    wasm/verify/themenearmiss_test.go          item 2
    .claude/commands/commit.md                 item 1 (new)
    .gitignore                                 item 1

`go run ./internal/themehistory` reproduces the recorded histogram
**byte-identically** with **nothing on stderr**, in 1.71–2.00s.
`internal/themehistory`'s own tests are 1.00–1.02s with four new arms, against
0.83s with none. **8 break-tests run, 8 fired**, plus four end-to-end
demonstrations in throwaway clones: a top-level duplicate declaration, two
subpackages sharing a name, a name declared only below `core/`, and a leaf type
moved into a path git will not write literally.

`go test ./wasm/verify` is **1.81–1.92s over seven runs**, which is now written
down with the machine attached rather than in a session doc.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

1. **(age 0 · value medium) The quoting finding stopped at this package.**
   `treePaths` fixes `ls-tree` in `internal/themehistory`, and the reasoning
   that produced it — two git commands, two quoting rules, and `-z` as the only
   spelling both agree on — applies to every place in this repository that
   parses git output. Nothing has checked whether there are others. The cheap
   form is a grep for `ls-tree`, `ls-files`, `diff --name-only` and `status`
   outside this package; the honest one is that a path read out of git without
   `-z` is a path with a rule attached to it, and the rule is not the one the
   reader is thinking of.
2. **(age 0 · value low) `retire` can block on a git that will not exit.** It
   closes stdin, drains stdout and Waits, which terminates for a
   `cat-file --batch` because that process exits on EOF. A git wedged for any
   other reason — a filesystem that will not answer, a pack being rewritten
   underneath it — would hold the drain open and the whole run with it. A
   context with a deadline and a Kill behind it is three lines; what is not
   obvious is what timeout is honest, since the drain is bounded by one
   response and the exit is not bounded by anything.
3. **(age 0 · value low) The `ambiguous` response is classified by a decision
   nobody has tested.** git answers `<oid> ambiguous` for a short oid matching
   two objects, and it is a complete one-line response like `missing` — so
   treating it as desynchronising is deliberately conservative and costs one
   process if it ever arrives. Nothing here asks for a short oid, so it cannot
   arrive today. If something ever does, the row belongs in
   `TestEveryBatchResponseShapeIsToldApart` with the decision made on purpose
   rather than by default.
4. **(age 0 · value low) The timings record covers one file.** Every wall-clock
   number in `themenearmiss_test.go` now cites `affordedTimingsTakenOn`, and
   the other timing prose in `wasm/verify` — `inkcanary_test.go`'s browser
   numbers, `checknumbering_test.go`'s walk costs — carries no machine. The
   record is package-level and could be cited by all of them; what stopped this
   session was that only one file's numbers had been re-measured today, and a
   record attributing numbers nobody re-took is the same unattributed number
   with a name on it.
5. **(age 0 · value low) `.claude/commands/commit.md` is a check nothing
   enforces.** It asks the model to notice a commit that owes a session doc,
   which is a habit and not a mechanism — the same class of thing as the note
   the previous session declined to make into a hook. A `PreToolUse` hook on
   `git commit` could see the staged paths and answer the question outright.
   The reason not to is that a hook that refuses a commit is a hook people
   disable; the reason to is that a note in a file is exactly what did not stop
   `8d14384`.
