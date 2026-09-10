# Session: a count that was wrong when it was typed, a census that could not see its own request, and a lever tied to the wrong scope

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-10 (follows "an-off-by-one-the-program-writes-on-purpose-a-comparison-no-machine-reverses-and-a-go-mod-that-was-not-one-line")

## Ask

"Do all items in the Next list." Seven items, all age 0, all raised by the
previous session — two medium, five low.

The previous session's shape was **a price nobody had decided was worth paying
beside an assertion nobody had checked could fail**. This one is what happens
when those checks are asked about THEMSELVES: **a census that cannot see the
change it asks for, a scope that answers a looser question than the one
written above it, and a number kept by hand in a comment that was wrong on the
day it was typed**. Three of the seven turned on a premise that did not
survive being measured — three repository walks that are seven, an estimate of
0.83s that is 0.90s, and a `-short` check that was asking about the enclosing
function rather than the branch.

Nothing outside tests changed. `git diff` touches five test files and adds two.

| item | where | shape |
|---|---|---|
| 1 | internal/themehistory | a paragraph holding up two claims |
| 2 | wasm/verify/versionorder | a census blind to its own request |
| 3 | internal/themehistory | a figure that was a reading of a core count |
| 4 | wasm/verify/shortlever | a lever tied to the wrong scope |
| 5 | wasm/verify/repowalks | a count that was wrong when it was typed |
| 6 | internal/themehistory | a remainder that got smaller than its noise |
| 7 | wasm/verify/gitquoting | a row for a non-reason |

---

## Item 1 · a paragraph holding up two claims

`enumWorkers` runs eight goroutines and two claims elsewhere rest on nothing
here being parallel: `blobsMu`'s comment says the program is single-threaded,
and the whole-walk arm swaps `os.Stdout` for the duration of `run()`. The
argument that the pool is fine — runs before the walk, touches no package
state, joins first — was a paragraph, and a paragraph is not a check.

`TestTheGoroutinesInThisPackageAreTheOnesDecidedOn` is the arm. Three `go`
statements, one row each saying what joins it, `goroutineBudget = 3`.

The part worth having is not the census, it is the **call graph**. A goroutine
body is three lines and a call:

    go func(){ blobs.read(…) }()      the shape nobody writes
    go func(){ leavesAt(sha) }()      the same bug, spelled the way somebody
                                      would — one hop to blob, two to `blobs`

So the walk follows this package's own declarations, transitively, out of each
goroutine, and holds them against four things: `blobs`, `os.Stdout`,
`fmt.Print*` (which reads `os.Stdout` at CALL TIME, so it is the redirect
wearing another name), and the `Fatal`/`FailNow`/`SkipNow` family (which calls
`runtime.Goexit` and ends the WORKER, leaving a `WaitGroup` nobody finishes).
`batchesStarted` is deliberately not guarded and the table says why: an
`atomic.Int64` is the one piece of package state that means the same thing from
two goroutines.

A row is per FUNCTION and the budget per STATEMENT, which is not a mismatch:
what a reader wants named is the place that decided to be concurrent, and a
second `go` inside a listed function is over budget rather than unlisted.

**Break-tests — 4 run, 4 fired**: `leavesAt` called from a worker ("calls
leavesAt, which reaches blobs"), a `t.Fatalf` in a worker, an `fmt.Printf` in a
worker, and a fourth goroutine feeding the channel.

One thing the census caught about itself: its own trailing `Logf` said "None of
them reaches …" on a failing run, which is the claim the errors above had just
contradicted — the exact fault `shortlever_test.go`'s header warns about. It
now says "Held against …, see the finding(s) above" when `t.Failed()`.

---

## Item 2 · a census blind to its own request

`TestTheDottedVersionParsers…` counted HAND-ROLLED parsers — a body that splits
on `"."` and runs the pieces through `strconv`. Its header called the blind
spot correct behaviour: an orderer built on `x/mod/semver` never splits, and a
semver-based comparator IS the change the arm asks for.

It is, right up until somebody adds one and leaves `versionFields` where it is.
The repository then holds two comparators that agree until the day they do not,
and the count still reads one. The zero-found case covered both of them going;
nothing covered one of them ARRIVING.

The handle was right and the NOUN was wrong. The budget is now about
comparators and the construction is a column:

    1 hand-rolled, 0 semver   today
    1 semver, 0 hand-rolled   the change this arm asks for, having been made.
                              Passes, and is meant to
    2 of anything             two comparators. The failure, whichever way each
                              is built — and one of each is the WORST case,
                              because the two disagree exactly where semver has
                              opinions (build metadata, a bare `v`, a
                              pre-release tail) and each looks obviously correct
                              beside its own caller

**Break-tests — 2 run, 1 fired, 1 correctly silent**: a `semver.Compare`-based
orderer added to a tracked file (fired at 2, naming both kinds), and the same
function with the receiver renamed to `sortpkg` (silent — which is the check
not matching every `.Compare` in the language).

---

## Item 3 · a figure that was a reading of a core count

`enumWorkers` is `min(NumCPU, 8)`, so the arm's largest term scales with the
machine in a way nothing else in either record does. The item ESTIMATED 0.45s
at two cores and 0.83s at one. Measured instead, by fixing the worker count and
re-running, three runs apiece:

    workers    the enumeration    the whole package
    1          0.90–0.92s         3.69–3.82s
    2          0.51s              3.30–3.31s
    4          0.29–0.30s         3.07–3.16s
    8          0.21–0.23s         3.05–3.14s

The one-core row is the figure this package had BEFORE the pool, which is what
says the pool is the only thing that moved. So `3.01–3.11s` is not a number
about this code, it is a number about this code **on eight cores**, and the
record says so in three places: the field, `enumWorkers`, and the whole-walk
arm's cost paragraph.

The `-short` half moves the OTHER way and that is written down too: a short run
skips the enumeration entirely, so the lever is worth the whole 0.90s on one
core against 0.22s on eight — **strongest on the machines least likely to have
been measured**.

The reporting arm now names the term when the core counts differ, rather than
reporting "8 cores against 4" and leaving the reader to find out which number
that moves.

---

## Item 4 · a lever tied to the wrong scope

`callsSkip` asked whether anything in the enclosing FUNCTION skips. That is a
question about the wrong scope, and it passes the exact shape the rule is
against:

    if runtime.GOOS == "js" { t.Skip(…) }      the function skips…
    samples := 1000
    if testing.Short() { samples = 10 }        …and `-short` shrinks a loop

A lever is a CONDITION. `shortLeverCall` now finds the innermost `if` whose
COND contains the call — by position arithmetic, so `a && testing.Short()`,
`!testing.Short()` and a parenthesised call all answer the same way — and the
guarded branch is what must skip. Four distinct failures, each its own
half-sentence: no `*testing.T` in the signature, the call not in any condition,
the branch not skipping, and the receiver being someone else's.

The receiver check came free with it. It was documented as deliberately absent
— the dataflow question `gitquoting_test.go` declines — but the parameter is
right there in the signature, and comparing two identifiers is not dataflow.

**Break-tests — 3 run, 3 fired**: a shrinking `-short` inside a test that skips
for an unrelated reason (**which the old check passed**), `tt := t; tt.Skip(…)`,
and `short := testing.Short(); if short { … }`.

---

## Item 5 · a count that was wrong when it was typed

The item said three repository-wide walks. The record said three, naming
`TestEveryTimingsRecordIsTheSameShape` as the last.

There are **seven**, and four of them parse.
`TestEveryGitListingAsksForNulSeparatedPaths` has parsed the whole tree since
before any of the other three existed and was simply never counted. And one
walk is not a test at all: `checkCitationsResolve` is a HELPER, so it walks
once per caller — a unit a census counting declarations would price at one
however many tests called it.

`TestTheRepositoryWideWalksInThisPackageAreTheOnesDecidedOn` reads it out of
the source, with a depth column, because the depths are not decoration:

    asks git for the file list    ~0.09s   one git process and a stat per file
    reads every tracked file      ~0.16s   citingFiles
    parses every Go file          ~0.18s   go/parser over all 381, on top

Budgets are split: `repositoryWalkBudget = 7` and `repositoryParseBudget = 4`
under it, because the parse count is the one that decides when a shared parse
stops being a fixture with lifetime rules nobody can reason about.

**It costs 0.01s and is not itself a repository walk.** It reads ONE directory
— which is where every caller of the unexported `citingFiles` has to be — and
parses only the files whose bytes name an enumeration, seven of thirty-five.
The byte scan cannot produce a false negative (a call contains the name), and
the parse is still what DECIDES, so this file's own prose naming `citingFiles`
a dozen times is read and discarded rather than counted, which is the failure a
grep would have.

**Break-tests — 2 run, 2 fired**: a `citingFiles` call added to the
list-only walk (depth mismatch), and a second call to `checkCitationsResolve`
(`×2`, and 8 over a budget of 7).

---

## Item 6 · a remainder that got smaller than its noise

`wholeRun` was 0.40s of fetches plus 0.90s of trees plus about 0.25s of "parse,
diff and printing", named as a remainder. The parse was the largest unmeasured
term, and the reason nobody had timed it was that timing `themeleaves.Of` means
holding every source at every revision in memory — 2906 blobs. The per-object
arm already has them.

So it is the SAME call the walk makes, over one revision's direct sources, once
per revision, over the bytes the batch just returned. Sources are collected
first and timed after, so map-building is not counted as parse.

    one `cat-file --batch`   398–405ms
    one `cat-file -p` each   30.40–30.52s   (2906 processes)
    ratio                    75.1–76.4×     (floor 5×)
    the trees, serially      0.87–0.92s     (88 `ls-tree`)
    themeleaves.Of           118–120ms      (88 revisions, 5201 leaf names)
    ─────
    together                 1.39–1.44s     against a wholeRun of 1.52–1.60s

What is left — the diff between revisions, and the printing — is **0.1–0.2s,
the same size as the disagreement between three separate readings of this
machine**. So it stays a remainder, and the record now says the condition
rather than the intention: not worth a clock until it is bigger than the noise
around it.

The clock has an arm beside it that is not a clock: `themeleaves.Of` must name
at least one leaf across the history. A parse that read nothing would produce a
perfectly plausible 120ms.

**Break-test — 1 run, 1 fired**: `themeleaves.Of(r.sources, "NoSuchTypeHere")`
— 0 leaves in 123ms, reported as the cost of failing to parse.

---

## Item 7 · a row for a non-reason

`gitWrapperTaintHelpers` had a `len` row, and the row said only that it is
spelled the way a local helper is. That is a fact about the walk collecting
bare names, not a reason anybody could act on — and an `append` or a `make`
arriving in the body would have earned a row on the same non-reason, at which
point the table stops being what it is FOR: the list of places a rule could be
hiding.

Nothing predeclared is such a place. It has no body in this repository. So the
class goes, asked of `go/types.Universe` rather than of a hand-written list —
`min` and `max` were not predeclared before Go 1.21, and a list here would be a
second copy of a language definition that changes. Builtins and predeclared
TYPES both, because `string(b)` is a conversion and an acceptance rule can hide
in exactly as much of either.

`shadowsAPredeclaredName` is what makes that a fact rather than a guess: this
file's top level and the function's own scope, which is where a redeclaration
would have to be to be worth reading. The rest of the package is a named limit
beside the other three.

`go/types` is imported as `gotypes` — this package already declares a `types`
function in `timingsrecords_test.go`, and the alias is one import's problem
rather than a rename in a file where the name is load-bearing.

**Break-tests — 2 run, 1 fired, 1 correctly silent**: `make(map[string]bool,
len(append([]string(nil), "x")))` in the body (silent, which is the whole
point), and a package-level `func imag` called from the body (fired, with the
shadow explained).

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

    internal/themehistory/concurrency_test.go   item 1 (new)
    internal/themehistory/timings_test.go       items 1, 3, 6
    wasm/verify/versionorder_test.go            item 2
    wasm/verify/shortlever_test.go              item 4
    wasm/verify/repowalks_test.go               item 5 (new)
    wasm/verify/timings_test.go                 items 3, 5 (record)
    wasm/verify/gitquoting_test.go              item 7

**14 break-tests run, 12 fired, 2 correctly silent** — the two silent ones are
cases where NOT firing is the finding: builtins in a body needing no row, and a
`.Compare` on something that is not `semver`.

Figures re-taken: `internal/themehistory` **3.01–3.11s** over seven runs (and
3.69–3.82s on one core, measured); `wasm/verify` 2.70–2.80s → **2.73–2.84s**,
which is one machine's spread rather than the new arm — that arm is 0.01s.

    -short lever          default        -short
    plain                 3.01–3.11s     1.22–1.27s
    -race                 6.52–6.64s     2.41s

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

1. **(age 0 · value high) Every census in this repository resolves a package
   by its conventional NAME, and an alias defeats all of them.** `semver` in
   the new comparator detector, `parser` in the walk-depth reader, `os` and
   `fmt` in the goroutine census, `testing` in the short-lever walk, `strings`
   and `strconv` in the version parser, `exec` in the git-wrapper taint. Six
   files, one assumption. `import sv "golang.org/x/mod/semver"` then
   `sv.Compare(…)` is invisible to item 2's new arm — which is the same blind
   spot item 2 was written to close, reintroduced one construction along. Each
   file states its own version of the reasoning ("the safe direction"), and
   that is true of the FALSE-POSITIVE half only; the false-negative half is
   nobody's. What would settle it is reading each file's import block, which
   every one of these walks already has parsed and none of them looks at.
2. **(age 0 · value medium) The whole-walk equality was priced when its cost
   was one number.** 88 `ls-tree` buys an off-by-one nobody else would see,
   and the previous session made that trade against 0.22s. It is 0.90s on one
   core, and the arm runs on every green run and again under `-race`. The
   decision is not obviously wrong at 0.90s; it is that it was never made at
   0.90s, and the record now carries the range that would prompt it.
3. **(age 0 · value medium) `shadowsAPredeclaredName` reads one file, and the
   census that could read the package now exists.** It was left at this file's
   top level plus the function's scope because a package-wide parse looked
   expensive. `repowalks_test.go` then measured that shape at 0.01s — read a
   directory, byte-scan for the name, parse only the hits. The limit is now
   closable at a cost that has been taken, and it is still written down as a
   limit.
4. **(age 0 · value medium) The two timings records now disagree about
   whether a figure is attributed to cores.** `internal/themehistory` says its
   number is a reading of a core count and carries the table; `wasm/verify`'s
   four parse walks are single-threaded and their 0.18s each does NOT move
   with cores — which is true, useful, and said nowhere. A reader holding the
   two records has one that attributes and one that is silent, and silence
   here reads as "not measured" rather than "does not vary".
   `TestEveryTimingsRecordIsTheSameShape` holds the five machine fields; it has
   no opinion about this.
5. **(age 0 · value low) The goroutine census reports the first guarded thing
   per goroutine, not all of them.** A worker with both a `t.Fatalf` and an
   `fmt.Printf` fires once, and the second is found on the next run after the
   first is fixed. That is the deliberate "three thousand identical failures is
   a wall" shape one size too small: two findings in one goroutine are two
   findings.
6. **(age 0 · value low) The goroutine census's call graph is keyed by NAME
   with no receiver.** A method `read` on `batchReader` and a package function
   `read` would be one node, which over-approximates — the safe direction —
   but the message would then blame a callee the goroutine never called. Named
   in the header; nothing distinguishes the two cases in the output.
7. **(age 0 · value low) `touchesGuarded` reports a BUCKET and the message
   reads as a name.** Four methods come back as `t.Fatal` and three as
   `fmt.Print`, so a `t.Fatalf` is reported as "names t.Fatal". The reason
   text is right for all of them and the line number is exact; the noun is not
   what the source says.
8. **(age 0 · value low) The repository-walk census counts call SITES, not
   calls.** A walk inside a loop or driven from a subtable is one site and N
   walks, and `runs` would say 1. There is no such site today. The same shape
   as the helper case it does close, one construct along.
9. **(age 0 · deliberate non-goal) The diff-and-print remainder stays
   unmeasured.** 0.1–0.2s, which is the size of the disagreement between the
   three readings around it, so a clock on it would report noise with a name.
   The record states the condition under which that changes rather than the
   intention, which is the difference between a decision and a deferral. Not
   an open question until the remainder grows.
