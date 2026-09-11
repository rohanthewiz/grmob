# Session: two copies nothing compared, a name taken off the wrong segment, and a note held to existing

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-10 (follows "six-censuses-that-could-not-read-an-import-block-a-limit-cheaper-to-close-than-to-describe-and-a-share-taken-on-somebody-elses-computer")

## Ask

"Do all items in the Next list." Eight items — one high, three medium, four
low — all raised by the previous session, which had itself written the thing
most of them are about: `importedAs`, the twenty lines that stopped six
censuses resolving a package by its conventional spelling.

The previous session's shape was **a qualifier nobody resolved**. This one is
the same fault read one level further out: not the IDENTIFIER a call is
written with, but the thing the census is holding the identifier up against.
A copy nothing compared. A package name taken off the wrong path segment. A
standing sentence about core counts that was held to EXISTING rather than to
being true. In each case the failure is silent and the evidence was already in
the room — the other copy, go.mod, the package's own declarations.

Three items came apart into two findings that had been reported as one: a loop
whose length is written down against one that is not, and the goroutine
census's two passes, one of which had been fixed and one of which had not.

Nothing outside tests changed. `git diff` touches seven test files, adds one,
and renames one.

| item | where | shape |
|---|---|---|
| 1 | two packages, one new file | a copy nothing compared |
| 2 | the shared resolver | one fact, five findings |
| 3 | the shared resolver, go.mod | a name off the wrong segment |
| 4 | both timings records | a note held to existing |
| 5 | wasm/verify/repowalks | a bound the source was holding |
| 6 | wasm/verify/gitquoting | a skip in the unsafe direction |
| 7 | internal/themehistory | the half the last fix did not reach |
| 8 | internal/themehistory | four files formatted for a message nothing prints |

---

## The rename, which items 1, 3 and 4 all needed

`wasm/verify/timingsrecords_test.go` → **`copies_test.go`**, and
`TestEveryTimingsRecordIsTheSameShape` →
**`TestTheShapesThisRepositoryKeepsTwoCopiesOfAreInStep`**.

Three of the eight items wanted a repository-wide parse and there was no room
for one: `repositoryParseBudget` is 4, it is at 4, and its own comment says a
fifth is where a shared parse becomes the cheaper of two bad options. That
constant exists to force a decision, and spending it on the arrangement of a
file rather than on a question would have been the worst way to reach it.

So the three questions were folded into the walk that already parses every Go
file, and the file was renamed to what it now is. Each question is a reading of
declarations the walk has already built; none of them adds a walk. The
`repositoryWalks` row was rewritten to say all three, and twelve prose
references across five files were followed.

---

## Item 1 · a copy nothing compared

The previous session put `importedAs` in two packages and wrote down, in the
copy's own header, that there was no arm over it and that naming that was
better than implying there was. That was honest about the state and wrong
about what could be done. A timings record can be COUNTED because it is a name;
the argument that a helper is only a shape was true of finding one and false of
comparing two.

The divergence had already happened, in both halves the item predicted:

    wasm/verify      path.Base, and a dot import reported through `t`
    themehistory     an open-coded LastIndex, and a bool the caller reported
                     in its own words

Both are now one generated set of six — `packageBase`, `isMajorVersion`,
`importedAs`, `dotImportsIn`, `qualifiersFor`, `unquote` — in
`wasm/verify/importnames_test.go` and a new
`internal/themehistory/importnames_test.go`. `concurrency_test.go` calls
`qualifiersFor` and its inline dot report is gone.

`checkImportResolverCopies` holds three things:

    identical      each declaration printed with go/printer, doc comment
                   removed. The two headers are SUPPOSED to differ — one says
                   what the function is, the other says why there is a second
                   — and the printer normalises everything else, so what is
                   compared is exactly what a copy has to keep in step
    the whole set  a package declaring some of them and not the others is a
                   copy coming apart, which is the state that precedes two
                   packages answering differently
    two of them    `importResolverCopies`, the same constant as
                   timingsRecordCopies and for the same reason

The difference from the record's check is the point of the two constants
sitting beside each other: a record's VALUES are different numbers about
different machines and were never meant to match, so it is held to a shape.
These are code, so they are held to being identical — the stronger thing, and
the cheaper one, because there is nothing to decide about whether a difference
was meant.

---

## Item 2 · one fact, five findings

The dot-import report fired per asking. Every census that resolves qualifiers
asks of every file it walks, and four of them walk the whole repository, so one
line of somebody's import block is one fact and as many findings as there are
askers — the wall this repository writes one-per-row rules against everywhere
else.

`qualifiersFor` now records `(file, path)` in `dotImportsReported` and reports
once, and the finding names every path that file dot-imports rather than the
one being asked about, so a file with two of them is one visit rather than two
runs.

The one visible edge is written down rather than left to be discovered:
`-count=2` makes the second iteration silent about a file the first reported.
The run still fails, and the fact recorded is a fact about a file on disk,
which does not change between two iterations of one binary.

---

## Item 3 · a name off the wrong segment

`path.Base` is exact for every path any census names and wrong for two
families a module graph actually contains — `github.com/x/y/v2` and
`gopkg.in/yaml.v2` — and wrong in SILENCE: the qualifier resolves to a name
nothing binds, every call is invisible, and the file reads as clean.

`packageBase` follows the two conventions the toolchain itself follows. The
remaining guess is the one nobody can close without loading the package, which
is the cost every walk here declines.

The other half is `checkImportPathsAreImportable`, which reads go.mod and holds
every path a census names to being one this module could import:

    stdlib      a first element with no dot, which is the module-path rule read
                backwards and is exact
    a module    this module or something go.mod requires, by prefix
    the name    packageBase's result not still looking like a version

go.mod is LEXED rather than loaded, and the reason is this file's own subject:
`golang.org/x/mod/modfile` would parse it properly and is in the graph, but it
is held there indirectly by the `tool` block, and importing from it would make
it direct — a change to the thing versionorder_test.go's whole census is about,
made in order to read it.

The limit is written down: a typo in the PACKAGE half of a path under a
required module (`golang.org/x/mod/semvr`) passes, because only a build settles
whether a directory exists. The module half is exact, which is where the
version suffixes and the interesting names are.

---

## Item 4 · a note held to existing

`coresAttribution` was held to being declared. Both packages had a measured
table; nothing re-derived either. A note saying "nothing here moves with cores",
left standing after somebody adds a pool, passed exactly as well as one that
was right — which is the state the record itself was in before it had an arm,
arriving one level along inside the thing the arm asks for.

The numbers cannot be checked from here and were never going to be: a wall
clock is a reading of a machine, which is why these are records rather than
assertions. What a parse CAN settle is the INVENTORY, and the inventory is
exactly the set a note about core counts is a claim about. So every declaration
in a record's package that reads `runtime.NumCPU` or `runtime.GOMAXPROCS` has
to be NAMED in that package's note — the reporting arm excepted, because its
NumCPU is the comparison against the record rather than a term.

It fired on two terms wasm/verify's note had never mentioned. The note said
"the afforded* band family in themenearmiss_test.go"; the declarations are
`affordedKLeafBandWalk` and `affordedTwoStepBands`, and both are now written
out. themehistory's already named `enumWorkers` and passed unchanged.

What this does not cover is a measurement drifting. What it does is make the
note go stale LOUDLY in the one way that is somebody's fault: a term arriving.

---

## Item 5 · a bound the source was holding

The repository-walk census reported every loop identically — one finding doing
the work of two. `for range 2 { … }` is two walks and the two is IN THE SOURCE;
`for _, c := range cases` is len(cases), which is a run-time fact. The first
was a cost the arm declined to compute rather than one it could not, and a row
forced to say `runs: 1` beside it is a row that cannot be right.

`loopBound` reads the two constructions somebody actually writes around a call
they meant to make a fixed number of times — `for range N`, a composite
literal, `for i := 0; i < N; i++` — and nothing else. Nested loops multiply and
one unreadable bound anywhere in the nest makes the product unknown. A bounded
site is priced into `runs` and named in the log; an unbounded one is the
finding that was there before, now saying which loop it could not price.

Two deliberate silences: a string range is unknown, because it goes round once
per RUNE and the literal's length is in bytes; and a call inside a loop whose
FuncLit is never invoked is priced as though it ran, which overstates — the
direction this arm is allowed to be wrong in.

---

## Item 6 · a skip in the unsafe direction

`packageLevelNames` skipped a file it could not read or parse, with the reason
beside it: the build says so first, so the only way there is a file that does
not compile. Sound, and an ASSUMPTION, and loose in the direction of NOT
finding a shadow — a `func len` in an unparseable file is a shadow the census
misses and a call it then drops as predeclared.

It is also an assumption the walk was already holding the evidence for: the
bytes are read before the parse and the byte scan has already said this file
might declare the name. Both are now reported, the parse failure only for a
file the scan kept. The finding is not "this does not compile"; it is that the
census cannot tell whether the class exclusion still holds.

Demonstrated with `//go:build ignore`, which is the case the limits list
already described: a file the build excludes and the directory scan reads.

---

## Item 7 · the half the last fix did not reach

The previous session made the direct pass report every guarded row and left
`reachesFrom` returning the first — the same finding, in the half of the check
it had not touched. A goroutine reaching `blobs` through one callee and
`fmt.Print` through another was told about whichever the breadth-first walk met
first.

`touches` became name → every row, `reachesFrom` returns `[]reachedGuard`, and
a row reached twice is reported once with the shorter chain — breadth-first
order, so a reader gets the least to check.

---

## Item 8 · four files formatted for a message nothing prints

`declarationOf` was called for every function in the package on every run, so
that `ambiguity` could look one up on a collision that has never happened. It
costs nothing at this size and it is backwards in exactly the package the
feature is for, where names collide often and the rendering would be paid for
every name to be read for a handful.

`declaredAs` holds `*ast.FuncDecl` and the formatting is on the message's side
of the call. The collision break-test reports the same two declarations it did
before.

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

    wasm/verify/importnames_test.go             items 1, 2, 3
    internal/themehistory/importnames_test.go   items 1, 2, 3 (new)
    wasm/verify/copies_test.go                  items 1, 3, 4 (renamed)
    wasm/verify/timings_test.go                 item 4
    internal/themehistory/timings_test.go       item 4
    wasm/verify/repowalks_test.go               item 5
    wasm/verify/gitquoting_test.go              item 6
    internal/themehistory/concurrency_test.go   items 1, 7, 8
    wasm/verify/versionorder_test.go            the rename

**15 break-tests run, 12 fired, 3 correctly silent.** Three further runs put
the old code back to show what it had been doing: the last-segment rule
resolving `bytdb/v2` to `v2` and `yaml.v2` to `yaml.v2`; one dot-import fact
reported twice when two censuses asked; and the call graph reporting one of two
rows. The silent three are cases where not firing is the answer — a typo'd
stdlib path that the module rule cannot judge and says so, a bounded loop
priced rather than reported, and `packageBase` over the four families it does
know.

**Figures.** `wasm/verify` **2.78–2.92s** over fourteen runs against a recorded
2.73–2.84s. The new readings were measured by taking them out and putting them
back — **0.18s → 0.19s**, the fourth parse walk costing a hundredth more than
the three beside it — which does not account for the rest, and the record now
says so rather than attributing it. `internal/themehistory` **3.01–3.12s** over
fourteen, which is the recorded range plus a hundredth.

The GOMAXPROCS table and both lever tables were widened to hold two takings
rather than replaced, which is what the `-race` row already does:

    GOMAXPROCS    the whole package        -short lever   default      -short
    1             3.08–3.27s               plain          3.01–3.12s   1.20–1.27s
    2             2.78–2.88s               -race          6.46–6.64s   2.41–2.44s
    4             2.74–2.79s
    8             2.75–2.85s

The shape did not move: one core is a tenth dearer and two is where wasm/verify
stops improving, which is the opposite of themehistory's, where the term is 88
git processes.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

1. **(age 0 · value high) The copies check compares functions and the two
   files are not only functions.** Both `importnames_test.go` declare a
   package-level `dotImportsReported`, which `qualifiersFor` reads. The
   compiler holds that it EXISTS — a missing one will not build — and nothing
   holds it to being the same kind of thing. A plain `map` in one and a
   `sync.Map` in the other compiles in both places and differs under
   `t.Parallel()`, which is the state the two `importedAs` bodies were in
   before this session. `importResolverShapes` is a list of function names and
   the walk only ever looks at `FuncDecl`s.
2. **(age 0 · value high) The copies census is now three questions in one
   test, and one `t.Fatalf` ends all three.** The record's reaching-anything
   arm is a Fatalf; the two new checks run before it for that reason, but
   `checkRecordShape` and everything after it are in the same function. A
   repository where the records had been renamed reports that and stops, and
   the import resolver's two copies go unchecked on that run. The previous
   arrangement had one question per test and this has three, which is what
   folding them into one walk bought and what it cost.
3. **(age 0 · value medium) `checkImportPathsAreImportable` cannot judge a
   standard-library path at all.** `os/exex` has no dot in its first element,
   so it passes as stdlib, and the census asking about it reports nothing
   forever. The module half is exact and this half is a written limit. What
   would close it is the list `go list std` prints, which is a second copy of
   something that changes every release — or a build, which is the cost every
   walk here declines. It is worth naming what the check is actually worth:
   five of the eight paths asked about today are stdlib.
4. **(age 0 · value medium) The cores note is held to naming its terms and not
   to the terms being real.** A note that names `enumWorkers` passes whether
   or not the sentence about it is true, and it also passes if it names a
   function that no longer exists — the check reads the note for a substring
   and never asks the other direction. A term deleted leaves its name in the
   note and nothing says so, which is the same silence one step round from the
   one this session closed.
5. **(age 0 · value medium) `loopBound` reads a literal and not a constant.**
   `for i := 0; i < enumWorkers; i++` is a bound this repository has written
   down — a package-level `var` or `const` with an integer initialiser — and
   it comes back unknown, so the site is reported rather than priced. The
   declaration is in the same directory the walk already reads, which is how
   `packageLevelNames` answers a question of exactly this shape for one
   hundredth of a second.
6. **(age 0 · value low) The dot-import registry is keyed by file and path and
   lives for the binary, and nothing clears it.** That is right for a run and
   wrong for a test that wants to assert the finding: a break-test for
   `qualifiersFor` cannot run twice in one binary, and there is no way to ask
   what it has recorded. Every other census here can be broken and re-run in
   place.
7. **(age 0 · value low) `declarationText` reports a printer error as the
   text.** A declaration go/printer cannot render comes back as `unprintable:
   …`, which compares unequal to the other copy and produces a drift finding
   naming a file — right by accident. The real finding is that the comparison
   could not be made, and it is one line to say so.
8. **(age 0 · value low) The repository-walk row for the copies census now
   describes three questions in one `asks` string.** That field was written to
   say what ONE walk asks the repository, so that a reader deciding whether to
   share a parse knows what each would then be sharing. A row that is three
   sentences is the first sign the unit has stopped being a question.
9. **(declined, non-goal) The diff-and-print remainder stays unmeasured.** See
   `ai_docs/plans/non_goals.md` — what was declined, the argument, and the two
   conditions that would change it.
