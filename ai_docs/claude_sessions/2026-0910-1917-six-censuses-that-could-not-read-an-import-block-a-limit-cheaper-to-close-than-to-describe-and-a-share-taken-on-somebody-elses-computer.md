# Session: six censuses that could not read an import block, a limit cheaper to close than to describe, and a share taken on somebody else's computer

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-10 (follows "a-count-that-was-wrong-when-it-was-typed-a-census-that-could-not-see-its-own-request-and-a-lever-tied-to-the-wrong-scope")

## Ask

"Do all items in the Next list except for the non-goal. Move it to
ai_docs/plans/non_goals.md." Eight items — one high, three medium, four low —
all raised by the previous session.

The previous session's shape was **a census that cannot see the change it asks
for**. This one is the same fault read one level down: not the NOUN a census
counts, but the IDENTIFIER it counts it by. Six files across two packages
answered "is this a call into package P" by comparing the qualifier to P's
conventional spelling, and every one of them had written down the same defence
— matching a bare name is loose in the direction of REPORTING, which is safe.
That argument is about the false positive. **The false-negative half was
nobody's, and it is the silent one.**

Three items turned out to be about something being cheaper than the sentence
explaining why it had not been done: a package-wide parse measured at 0.01s, a
share the running machine can take for itself, and a `for` loop that a parse
can find even if it cannot count.

Nothing outside tests changed. `git diff` touches eight test files, adds one,
and appends to `ai_docs/plans/non_goals.md`.

| item | where | shape |
|---|---|---|
| 1 | six files, two packages | a qualifier nobody resolved |
| 2 | internal/themehistory | a share taken on somebody else's computer |
| 3 | wasm/verify/gitquoting | a limit cheaper to close than to describe |
| 4 | both timings records | one attribution and one silence |
| 5 | internal/themehistory | a wall one size too small |
| 6 | internal/themehistory | a route nothing takes |
| 7 | internal/themehistory | a bucket that read as a name |
| 8 | wasm/verify/repowalks | a site that is not a call |
| 9 | — | moved to non_goals.md |

---

## Item 1 · a qualifier nobody resolved

Six files, one assumption:

    pkg.Name == "semver"    versionorder_test.go, the comparator detector
    pkg.Name == "parser"    repowalks_test.go, the walk-depth reader
    pkg.Name == "testing"   shortlever_test.go, the lever walk
    pkg.Name == "exec"      gitquoting_test.go, the git-wrapper taint
    pkg.Name == "strings"   versionorder_test.go, the split-and-parse half
    pkg.Name == "os"/"fmt"  internal/themehistory/concurrency_test.go

Each header argued the loose direction was the safe one. That covers a local
`semver` that exports a Compare being REPORTED and dismissed. It says nothing
about `import sv "golang.org/x/mod/semver"` — which is a second comparator
arriving in the exact spelling versionorder's own failure message asks somebody
to write, and which produces no output at all.

`wasm/verify/importnames_test.go` is the new file. `importedAs(file, path)` is
the qualifier set that file binds; `qualifiersFor(t, …)` is the same with the
one thing an import block cannot answer turned into a finding — a **dot
import** puts the package's names in file scope, where a census reading
qualifiers sees a bare `Compare(a, b)` and nothing to resolve. That gets
reported rather than read as clean.

It resolves in BOTH directions, which is the part the old argument was missing
from: a file that does not import x/mod/semver has no call into it, whatever
its identifiers are named.

themehistory gets a twenty-line copy, for the reason its timings record is a
copy — two separate `package main` programs, and a package existing so that one
function could be one function. The copy says so, and says the thing the
timings record has and this does not: there is no arm over it, because a record
is a NAME and a helper is a shape.

**Break-tests — 5 run, 3 fired, 2 correctly silent**, plus two regression
demonstrations:

    sv.Compare in a tracked file          fired: 2 comparators, both named
    semver.Compare, no such import        silent — the false positive, closed
    gotesting.Short() with a shrink       fired: unlisted lever, and the
                                          branch held to *gotesting.T's Skip
    goparser.ParseFile in a repo walk     silent (correct: still 4 parse walks)
      …with the name check put back       fired — depth reported as `reads
                                          every tracked file`, 3 parses not 4
    osexec.Command in a git wrapper       silent (correct: it is a wrapper)
      …with the name check put back       fired — "contains no
                                          exec.Command", a rejection of a
                                          CORRECT file, which the case table
                                          calls the one noise it cannot afford
    stdio.Println in a worker             fired: "names stdio.Println"

The two put-back runs are the finding, not the check: they are what the six
files were doing before this.

---

## Item 2 · a share taken on somebody else's computer

The whole-walk arm holds the batch reader's fetch count to an EQUALITY, and
that costs 88 `git ls-tree`. The previous session settled the trade against
0.22s — "a seventh of this arm" — and 0.22s is the eight-core end of a term
that is 0.90s at one worker.

Two things, and only the second is new code:

The decision is now taken at 0.90s, in writing, and comes out the same way: it
buys a fetch-path off-by-one this program has an actual mechanism for (the
newline path, which goes round the batch and is invisible to every other
assertion in the arm); the alternative is the HEAD-scaled floor it replaced,
which cannot see that; and `-short` is the escape, worth most exactly where the
pool is worth least.

And the share stopped being quoted. The log line takes it from the run:

    8 workers    enumerated in 220ms — 13% of this arm's 1.719s
    1 worker     enumerated in 867ms — 36% of this arm's 2.376s

A seventh and a third, read off the machine in front of the reader rather than
off a comment written on a different one. The 0.83s the enumWorkers header had
been quoting was an estimate; the measured figure is 0.87–0.92s, and that line
now says which it is.

---

## Item 3 · a limit cheaper to close than to describe

`shadowsAPredeclaredName` read this file's top level and the function's own
scope. The rest of the package was a written-down limit — a `func len` two
files over is what a bare `len(…)` resolves to, and the taint census would have
skipped the call as predeclared.

It was left open because a package-wide parse looked expensive. Then
repowalks_test.go measured that exact shape for itself: read one directory,
byte-scan for the name, parse only the hits. **0.01s.** The limit was closable
at a price the package had already paid, and the paragraph describing it was
longer than the code that ends it.

The scan is over the CANDIDATES rather than the language: only predeclared
names this body actually calls, which today is one. A file whose bytes do not
contain `len` cannot declare it, so the byte scan has no false negative, and
the parse is what decides — this file's own prose, which says `len` a dozen
times, is read and discarded.

Two limits replace it, both real: a dot-imported package's names would be in
file scope where no declaration scan finds them, and the directory scan reads
build-excluded files too — which is the ASKING direction, the safe one.

**Break-tests — 2 run, 1 fired, 1 correctly silent**: `func imag` in another
file of the package with a call added to the body (fired, with the shadow
explained), and the same call with no such declaration (silent, which is the
whole point of dropping the class).

---

## Item 4 · one attribution and one silence

`cores` is the only machine field that moves a recorded number by a term
somebody can name, and the two records said different amounts about it:
internal/themehistory carried a measured table, wasm/verify said nothing. A
reader holding both found an attribution beside a gap, and **silence reads as
"nobody measured that" rather than "that is not where the difference is"**.

wasm/verify's answer is not "nothing", which is what made it worth taking
rather than asserting:

    GOMAXPROCS    the whole package
    1             3.08–3.21s
    2             2.78–2.84s
    4             2.74–2.76s
    8             2.75–2.84s

The four repository-wide parse walks are single-threaded — 0.18s each, the same
on any machine — and foldWalk is a node process. What moves is the `afforded*`
band family, which splits across GOMAXPROCS. So this package is **flat from two
cores upwards**, which is the opposite SHAPE from themehistory's, where the
term is 88 git processes and the improvement runs all the way to eight. Neither
is guessable from the other, which is why having one and not the other was
worse than having neither.

Both packages now declare `coresAttribution` and print it when the counts
differ, and `TestEveryTimingsRecordIsTheSameShape` holds every record to having
one — the same pass, and the same directory-keyed mechanism, as the reporting
arm it already required.

**Break-test — 1 run, 1 fired**: `coresAttribution` renamed in wasm/verify
(fired, naming the package and what the note has to say).

---

## Items 5, 6, 7 · what a finding says

Three separate looseness in one census, all about the message rather than the
verdict.

**5 · a wall one size too small.** `touchesGuarded` returned the first hit. A
worker with a `t.Fatalf` and an `fmt.Printf` fired once, and the second was
found on the next run after the first was fixed. What justifies stopping early
elsewhere here is a wall of IDENTICAL findings; two different reasons are not
that. It now returns all of them, **one hit per ROW** — four prints in one
worker is still one thing to fix, and four lines saying so is the wall from the
other side. The call-graph check also runs now even when the direct pass found
something, and only skips a name the direct pass already reported.

**6 · a route nothing takes.** The graph is keyed by name with no receiver, so
a method `read` and a function `read` are one node. Over-approximating is the
safe direction for whether a finding is REPORTED and says nothing about what it
SAYS: the chain named can run through a declaration the goroutine never called.
Telling them apart needs the receiver's type at the call site, which is the
dataflow question these censuses decline. What can be done without it is to say
so, and `declarationOf` makes the collision visible from the declarations
alone.

**7 · a bucket that read as a name.** Four selectors come back under the row
`t.Fatal` and three under `fmt.Print`, so a `t.Fatalf` was reported as "names
t.Fatal" — an exact line number beside a noun the source does not contain.
`guardedHit` carries both: the row is what the REASON is keyed by, the spelling
is what a reader greps for.

**Break-tests — 2 run, 2 fired**: a worker with both a `t.Fatalf` and an
`fmt.Printf` (two findings, one run, `names t.Fatalf` and `names fmt.Printf`
with their own lines), and a package function `rd` beside a method
`(*batchReader).rd` reached from a goroutine (fired, naming both declarations
and saying the chain may run through one it never touched).

---

## Item 8 · a site that is not a call

The repository-walk census counts call SITES. That is the same number as walks
exactly while each site is reached once, and a walk inside a `for` — or inside
a subtest closure a `range` drives — is one site and as many walks as the loop
is long. `runs: 1` beside it reads exactly like a row that is right.

A parse cannot price it: the loop's length is a run-time fact. So it is
REPORTED, which is the honest version of the same finding. Loops are collected
as source ranges and calls tested against them by POSITION, for the reason
`containsPos` gives one file over — one parse, one FileSet, and nothing has to
know what the loop is made of, which is also why descending through a `FuncLit`
costs nothing.

**Break-tests — 2 run, 2 fired**: `for range 2 { checkCitationsResolve(…) }`,
and the same call inside `t.Run(n, func(t *testing.T){…})` inside a `range`.
Both name the file, the line and the enclosing function, and both say that the
row's count understates by the loop's length.

---

## Item 9 · moved

`ai_docs/plans/non_goals.md` gains **"The diff-and-print remainder stays
unmeasured"** — what was declined, where the shape lives, the argument, and
what would change it. The two conditions are written out: the remainder growing
past the spread around it, or the readings around it getting tighter, since
what makes it unmeasurable is the RATIO between the term and the noise rather
than the term's own size.

The code comment stays where it is and gains a pointer, which is what
themenearmiss_test.go does for the k = 3 entry — the comment is where somebody
reading that function looks, the file is where somebody planning work looks.

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

    wasm/verify/importnames_test.go             item 1 (new)
    wasm/verify/versionorder_test.go            item 1
    wasm/verify/repowalks_test.go               items 1, 8
    wasm/verify/shortlever_test.go              item 1
    wasm/verify/gitquoting_test.go              items 1, 3
    wasm/verify/timings_test.go                 item 4
    wasm/verify/timingsrecords_test.go          item 4
    internal/themehistory/concurrency_test.go   items 1, 5, 6, 7
    internal/themehistory/timings_test.go       items 2, 4, 9
    ai_docs/plans/non_goals.md                  item 9

**13 break-tests run, 10 fired, 3 correctly silent** — the silent three are
cases where NOT firing is the finding: a `semver.Compare` in a file that
imports no semver, an aliased `go/parser` still read at the right depth, and an
aliased `os/exec` wrapper still accepted. Two further runs put the old name
checks back to show what they had been doing; both fired, one understating a
walk's depth and one rejecting a correct file.

Figures re-taken and unmoved: `wasm/verify` **2.74–2.84s** over seven runs on
an idle machine, against a recorded 2.73–2.84s; the four parse walks
0.18–0.19s each, unchanged, so the import-block reads cost nothing measurable.
`internal/themehistory` **3.02–3.09s** over seven against a recorded 3.01–3.11s.
The one figure widened is the `-race` row, 6.52–6.64s → **6.46–6.64s**, six
readings rather than three, held as a range rather than replaced.

    -short lever          default        -short
    plain                 3.01–3.11s     1.22–1.24s
    -race                 6.46–6.64s     2.41s

The new arms' own cost: the package-wide shadow scan is **0.01s**, which is the
shape repowalks measured and the reason item 3 was closable at all.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

1. **(age 0 · value high) `importedAs` is now in two packages and nothing
   holds the two together.** The timings record's copy is held to two by
   `TestEveryTimingsRecordIsTheSameShape`, which can count copies because a
   record is a NAME — `…TimingsTakenOn`, readable off a parse. A helper is a
   shape, and the copy's own header says there is no arm over it. That is
   honest and it is the exact state the timings record was in before somebody
   wrote the arm: two files kept in step by whoever remembers to. The two
   differ already — wasm/verify's reports the dot import through `t` and
   themehistory's returns a bool the caller reports — which is a divergence
   nothing would catch.
2. **(age 0 · value medium) The dot-import report fires per FILE, and the
   walks call it per file per question.** versionorder asks three times per Go
   file in the repository; a single dot-imported `strings` would produce one
   finding per question rather than one per file. Nothing dot-imports anything
   today, so the multiplication has never happened, and the shape is the
   "three thousand identical failures is a wall" one that item 5 was about.
3. **(age 0 · value medium) The package name is the last path segment, and
   that is a guess this repository could check.** `path.Base` is exact for
   every path any census names and wrong for `gopkg.in/yaml.v2` and its
   family. The failure is silent in the same direction item 1 was about: a
   call into such a package resolves to nothing. go.mod lists what this module
   requires, and a census over import paths whose base is not the package name
   is a walk these files already have the parse for.
4. **(age 0 · value medium) `coresAttribution` is held to EXISTING, not to
   being true.** The census checks that each package declares one. Both
   contain measured tables today; nothing re-derives them, and a record whose
   note says "nothing here moves with cores" after somebody adds a pool passes
   exactly as well as one that is right. The whole-walk arm shows the shape
   that would fix it — it reports its own share per run — and neither record's
   note is produced that way.
5. **(age 0 · value low) The loop report cannot tell a bounded loop from an
   unbounded one.** `for range 2 { … }` is two walks and a parse can read the
   two; `for _, c := range cases` is len(cases), which it cannot. Both are
   reported identically, as "as many walks as the loop has iterations". The
   first is a number this arm declines to compute rather than one it cannot.
6. **(age 0 · value low) `packageLevelNames` skips a file it cannot parse, and
   that is the unsafe direction.** A file go/parser rejects is dropped, so a
   `func len` inside it is a shadow the census does not find and a call it
   then skips. The header says so and argues the only route there is a file
   that does not compile — true, and the arm could say it instead of assuming
   it, since it has already read the bytes.
7. **(age 0 · value low) The goroutine census's direct pass and its call graph
   report different amounts.** The direct pass names every row a goroutine
   reaches; `reachesFrom` still returns the first. A goroutine that reaches
   `blobs` through one callee and `fmt.Print` through another gets one of the
   two, which is item 5's finding one construct along — in the half of the
   check that item 5 did not touch.
8. **(age 0 · value low) `declarationOf` is built for every function in the
   package and read only on a collision.** Four files' worth of strings
   formatted so that `ambiguity` can look one up, on a run where there is no
   collision and never has been. It costs nothing measurable at this size; it
   is a shape that would matter in a package where names collide often, which
   is the case the feature is for.
