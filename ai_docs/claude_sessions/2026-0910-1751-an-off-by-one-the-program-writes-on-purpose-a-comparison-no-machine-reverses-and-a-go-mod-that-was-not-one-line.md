# Session: an off-by-one the program writes on purpose, a comparison no machine reverses, and a go.mod that was not one line

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-10 (follows "a-comparison-that-could-not-say-which-way-a-floor-that-was-a-third-of-one-reading-and-a-number-nothing-could-re-take")

## Ask

"Do all items in the Next list." Seven items, all age 0, all raised by the
previous session — three medium, four low.

The previous session's shape was **records that answer a question with a
number instead of a comparison**. This one is what those records leave behind
once each has been paid for: **a price nobody had decided was worth paying, and
an assertion nobody had checked could fail**. Three of the seven turned out to
rest on a premise that was simply wrong — an off-by-one nobody expected (the
program writes one on purpose), a ratio that would be a wall clock (it is the
one quantity that is not), a go.mod with one line in it (it has several, and
one of them is the dependency the comment says it does not have).

| item | where | shape |
|---|---|---|
| 1 | internal/themehistory | a price with no decision behind it |
| 2 | internal/themehistory | a comparison no machine reverses |
| 3 | wasm/verify/versionorder | a go.mod that was not one line |
| 4 | wasm/verify/gitquoting | a census that trusted the function's shape |
| 5 | wasm/verify/shortlever | a lever with no rule |
| 6 | internal/themehistory | a subtraction nobody had done |
| 7 | internal/themehistory | a walk arriving by a different door |

---

## Item 1 · a price with no decision behind it

The whole-walk arm holds `batchReader.reads` to an EQUALITY against what
`themeSourcesAt` names across the history, and that expectation is 88
`git ls-tree` processes — **857ms, 35% of the arm**, paid on every green run
and again under `-race`. The cheap alternative was a floor scaled off HEAD, one
process. What was missing was not an argument, it was the question: **is an
off-by-one in the fetch path a failure mode anybody expects?**

It is, and the program writes one itself. `blob` sends a path containing a
newline to a `cat-file -p` of its own — the batch protocol is line-terminated,
so asking anyway does not fail the request, it DESYNCHRONISES the stream — and
that fetch never reaches `batchReader.read`:

    a path with a newline   `reads` short by exactly one per such path per
                            revision, with ONE process started, a live reader,
                            and a correct table printed
    a desync mid-run        the reader is retired and replaced, so `reads`
                            starts again from nought. Caught by the process
                            count too

The first is the case only the equality sees. Every other assertion in the arm
passes through it unchanged, and a floor of a thousand would never notice a run
that quietly stopped batching a file. So the trade is made and written down,
and the price is paid DOWN rather than accepted: the enumeration now runs in a
bounded pool (`enumWorkers`), which is the one concurrent thing in this
package and says at length what it deliberately leaves serial.

    0.83s → 0.22s          the enumeration, 8 workers
    ~2.5s → ~1.8s          the arm, on a plain run
    3.77–3.93s → 3.01–3.07s   the package

**Break-tests — 2 run, 2 fired**: `blob`'s guard widened so 88 real paths went
round the batch (`short by 88` — one process, live reader, table printed, and a
floor would have passed), and the pool made to drop half the commits it was
handed (`1463 over`).

---

## Item 2 · a comparison no machine reverses

`perTook <= batchTook` over 2906 objects is 30s against 400ms. No machine
reverses it, so the assertion was decoration and the content was the two
numbers logged beside it — the same shape this package keeps writing arms
against, one level up.

The objection to a floor was that **a multiple is a wall-clock assertion
wearing a ratio's clothes**. That is the wrong reading of what these two
numbers are: they are taken in ONE RUN over the SAME objects in the same order,
so everything about the machine that scales both — a slower disk, a busy core,
a cold page cache, a `-race` binary — divides out of the quotient. What is left
is exactly the quantity `blob`'s comment is made of, fork cost against pipe
cost. A number that survives the machine changing is not a reading of the
machine.

`perObjectSlowdownFloor = 5`, against a measured **76×**. Deliberately nowhere
near it: a machine whose forks are fifteen times cheaper relative to its pipes
than this one's still passes, and what it fails is the trade having actually
gone.

**Break-test — 1 run, 1 fired**: the per-object clock divided by 20, which
fired at 3.8× — and is a case the old direction check passes.

---

## Item 3 · a go.mod that was not one line

`compareVersions` is twenty lines with a reason attached: "no dependency,
against a repository whose go.mod has one line in it". **Both halves were
wrong.** go.mod has several requirements and one of them is
`golang.org/x/mod`, held indirect by the `tool` block that pins gomobile — so
the import moves a line rather than adding a download. And `semver` answers
every rule the comparator has, which was checked rather than assumed:

    semver.Compare("v2.1.267", "v2.1.30")       +1   numeric, not lexical
    semver.Compare("v2.1", "v2.1.0")             0   a missing field is 0
    semver.Compare("v2.2.0-nightly", "v2.2.0")  -1   a pre-release is behind
    semver.IsValid on anything else           false   the unordered case

What survives is smaller and is about ONE caller: semver wants a leading `v`
and `claude --version` prints `2.1.267`, so something wraps it either way.

`TestTheDottedVersionParsersAreTheOnesTheReasonCovers` is the census, and the
syntactic handle is the PARSER rather than the comparator — an orderer cannot
be recognised by shape and a parser can, because ordering dotted versions means
having the fields as numbers. One hit across 381 Go files (`versionFields`),
budget 1, with `versionReaders` listing the four places that read a version and
do NOT order one.

**Break-tests — 2 run, 2 fired**: a second hand-rolled parser, and a
`versionReaders` row naming a file that had moved.

---

## Item 4 · a census that trusted the function's shape

`gitWrapperAcceptRules` counts `reaches = true` and holds the body to one
`return ""`. Two ways round that, both the same move — an acceptance decided
somewhere the census does not read — and both are now checks:

    reaches = reachesVia(…)   an assignment whose right-hand side is not the
                              literal `true`. The count does not move and the
                              rule is live. Every assignment to `reaches` is
                              held to `= true` (with `reaches := false`, the
                              declaration, told apart by its token)
    if acceptsVia(…) { … }    a helper deciding the rule and leaving
                              `reaches = true` as its punctuation. The callees
                              are held to `gitWrapperTaintHelpers`

What remains is one level further out and is not closable by a parse of this
function: the three helpers could themselves grow. They are listed by name,
which is the point of listing them.

**Break-tests — 2 run, 2 fired**: `reaches = acceptsVia(…)` (both new checks,
plus the pre-existing count), and the item's own example — `if acceptsVia(…) {
reaches = true }`, which leaves the count at 2 and fires the callee check
alone.

---

## Item 5 · a lever with no rule

One `testing.Short()` in the repository, load-bearing, asserted nowhere. Two
things go wrong quietly from there and both look like a fast green run: a
second arm under the same skip, or the lever moving.

`TestTheShortLeversAreTheOnesThisRepositoryHasDecidedOn` makes it a rule.

    where       every testing.Short() is one of the rows, and every row is
                still there — both directions
    how         each one SKIPS. A `-short` that shrinks a loop or lowers a
                sample count is a different thing wearing the same name: the
                test passes, its name is in the output, and what it asserted
                is not what it asserts on a full run
    how many    shortLeverBudget = 1. One lever is a lever; a second is a
                convention, and `-short` then means whatever the set of tests
                reading it happens to skip

The `stops` column is the part worth having: a row naming only the test says
where the lever is, and what a reader of a green `-short` run needs is which
claims did not get made.

**Break-tests — 2 run, 2 fired**: a second lever that SHRINKS rather than skips
(all three arms fired), and the guarded test renamed (both the unlisted-lever
and missing-row directions).

---

## Item 6 · a subtraction nobody had done

`wholeRun` is 1.5s for the walk, `perObjectRun` carried 400ms for the fetches
alone, and the difference was a sentence. It was the one number pair in either
record that invited a subtraction, and a reader who did it got a figure nobody
had measured.

The missing term is now taken — the 88 `ls-tree` run SERIALLY, which is what
the walk itself pays for trees, and which is why the per-object arm passes
`workers = 1` where the whole-walk arm passes `enumWorkers`. All four in one
log:

    one `cat-file --batch`   395–400ms
    one `cat-file -p` each   30.36–30.39s   (2906 processes)
    ratio                    76×            (floor 5×)
    the trees, serially      0.90–0.95s     (88 `ls-tree` processes)

So `wholeRun`'s 1.52–1.60s is 0.40s of fetches plus 0.90s of trees plus about
0.25s of parse, diff and printing — **named in the log as a REMAINDER** rather
than printed as though it had been timed. Three readings of one machine taken
in different runs do not decompose exactly, and saying so is the point of the
record.

The whole-walk arm's own log now carries its worker count and says the split is
elsewhere, because a pooled reading of the trees is not the walk's tree cost
and must not be subtracted as though it were.

---

## Item 7 · a walk arriving by a different door

A previous session removed a walk taken twice in one run. `themeSourcesAt` is
one arriving by a different door, for a better reason, and it is now named in
that function's header rather than left to be rediscovered as the same finding:

    leavesAt              once per revision, in the program. The fetch itself
    themeSourcesAcross    once per revision, in the whole-walk arm
    themeSourcesAcross    once per revision again, in the per-object arm —
                          off unless GRMOB_PER_OBJECT_FETCH is set, so a green
                          run pays two of the three

The duplication is the point: an expectation derived from the walk's own fetch
would be the fetch checking itself.

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

    internal/themehistory/timings_test.go   items 1, 2, 6
    internal/themehistory/main.go           item 7 (comment-only)
    wasm/verify/versionorder_test.go        item 3 (new)
    wasm/verify/hookconfig_test.go          item 3
    wasm/verify/gitquoting_test.go          item 4
    wasm/verify/shortlever_test.go          item 5 (new)
    wasm/verify/timings_test.go             re-taken

`internal/themehistory/main.go` is **comment-only** — `git diff` with the
comment lines filtered out is empty — so the program's output is unchanged by
construction and no worktree comparison was needed. **10 break-tests run, 10
fired.**

Figures re-taken: `internal/themehistory` 3.77–3.93s → **3.01–3.07s** over
seven runs, which is the pooled enumeration and nothing else; `wasm/verify`
2.49–2.59s → **2.70–2.80s**, for two repository-wide censuses that report 0.18s
each. Two arms at 0.18s against a total that moved 0.21s do not add up, and the
record says so rather than rounding it away.

    -short lever          default        -short
    plain                 3.01–3.07s     1.20–1.24s
    -race                 6.46–6.57s     2.38–2.40s

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

1. **(age 0 · value medium) The pool is the one concurrent thing in a package
   whose invariant is that nothing is.** `blobsMu`'s comment says the program
   is single-threaded, and the whole-walk arm's `os.Stdout` redirect is safe
   only because nothing here is parallel. `themeSourcesAcross` now runs eight
   goroutines, and the argument that this is fine — it touches no package
   state, runs before the walk, joins first — is a paragraph rather than a
   check. A later edit that reached `blobs` from inside the pool would be a
   race `-race` catches only if a fetch happened to overlap, and the arm that
   would notice does not exist. The same file already writes censuses for
   exactly this shape.
2. **(age 0 · value medium) The dotted-version census cannot see the change it
   asks for.** It counts hand-rolled parsers, so an orderer built on
   `x/mod/semver` is invisible to it — stated in its header as correct
   behaviour, and it is, right up until somebody adds one and leaves
   `versionFields` in place. The repository then has two comparators and the
   count still reads 1. The zero-found message covers the case where both go;
   nothing covers the case where one arrives. What would settle it is deciding
   whether the budget is about hand-rolling or about comparators, which are
   the same number today and will not be.
3. **(age 0 · value low) `enumWorkers` makes the arm's cost a property of the
   machine, and the record does not say so.** 0.22s is eight cores; a two-core
   runner gets nearer 0.45s and a one-core one gets the old 0.83s back. The
   record carries "over 8 workers", which is the honest spelling, but the
   `wholePackage` table and the value of the `-short` lever are both quoted as
   if they were one number — and `-short`'s whole justification is a figure
   that now varies by a factor of four across machines nobody has measured.
4. **(age 0 · value low) The `-short` census ties a lever to its enclosing
   FUNCTION, not to the `if` it is in.** `callsSkip` asks whether anything in
   the function skips, so a `testing.Short()` used to shrink a loop inside a
   test that also skips for an unrelated reason would pass. It also matches any
   `.Skip`/`.Skipf`/`.SkipNow` selector without checking the receiver, which is
   written down as deliberate — the same dataflow question `gitquoting_test.go`
   declines — and is the same class of gap one level up.
5. **(age 0 · value low) Three repository-wide walks in `wasm/verify`, each
   parsing all 381 Go files.** 0.18s each, measured the same alone as together,
   so 0.54s of a 2.8s package. Each is independent by design and that is the
   right call at three; the fourth is where a shared parse becomes the
   question, and there is no arm counting them — which is the shape this
   repository writes arms for, not written here.
6. **(age 0 · value low) The remainder in `wholeRun`'s decomposition is still
   a remainder.** ~0.25s of parse, diff and printing, named as unmeasured
   rather than presented as a reading. What would measure it is timing
   `themeleaves.Of` over the same sources or instrumenting `run()`, and both
   were left out because the decomposition's point was to stop the subtraction
   being invited, not to complete it. Whether the last term is worth a clock is
   undecided rather than declined.
7. **(age 0 · value low) `gitWrapperTaintHelpers` has a builtin in it for a
   non-reason.** `len` is a row because it is spelled the way a local helper is
   — bare — and the walk only collects unqualified names. Nothing distinguishes
   it from the two real helpers, so an `append` or a `make` arriving in the
   body would need a row for the same non-reason, and the table stops being a
   list of places a rule could hide.
