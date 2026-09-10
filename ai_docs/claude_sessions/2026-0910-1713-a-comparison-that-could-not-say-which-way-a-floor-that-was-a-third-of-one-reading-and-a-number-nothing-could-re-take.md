# Session: a comparison that could not say which way, a floor that was a third of one reading, and a number nothing could re-take

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-10 (follows "a-table-with-no-version-a-heredoc-nothing-ran-and-a-count-that-was-a-bound")

## Ask

"Do all items in the Next list." Seven items, all age 0, all raised by the
previous session — three medium, four low.

The previous session's shape was **premises standing on one instance**. This
one is what that leaves behind once each premise has a record beside it:
**records that answer a question with a number instead of a comparison**. A
version equality where the question was order. A floor where the question was
how many. A figure carried in prose where the question was a ratio. And, under
the four low items, one shape repeated — **a decision taken and then left as a
mechanism**, so the next reader meets the mechanism and has to re-derive the
decision.

| item | where | shape |
|---|---|---|
| 1 | wasm/verify/hookconfig | a comparison that could not say which way |
| 2 | internal/themehistory | a floor that was a third of one reading |
| 3 | internal/themehistory | a number nothing could re-take |
| 4 | internal/themehistory | a cost with no lever |
| 5 | wasm/verify/timingsrecords | a direction nobody had named |
| 6 | internal/themehistory | a global with one reader |
| 7 | wasm/verify/gitquoting | an approximation on its first rung |

---

## Item 1 · a comparison that could not say which way

`hookSchemaNote` held `claude --version` against the recorded build with
string equality, so `2.1.300` and `2.0.9` against tables read from `2.1.267`
produced the same sentence: "a different build". **They are opposite
findings** — one means a release may have ADDED the key and the config is
correct, the other means nothing later than the tables is installed and the
key cannot be new. The note told the reader to check the version, which is the
right instruction and is also the check declining to do the comparison it had
just made.

`compareVersions` orders two dotted versions: numeric field by field (`2.1.267`
against `2.1.30` is 267 > 30, which string comparison gets backwards), a
missing field is 0 so `2.1` and `2.1.0` are one release, a pre-release tail is
behind its release, and **anything it cannot read returns `ok=false` rather
than a side**. Twenty-odd lines, no dependency.

The note is now four shapes, and the two new ones are the point:

    a NEWER claude    the bad direction is live — the key may be one a
                      release after the tables added
    an OLDER claude   the key CANNOT be a later addition; either a mistake,
                      or a key this build has and the tables' build dropped

The report arm splits the same way, and the unorderable case is reported on a
green run because it is the one state where a finding cannot name a direction.
`TestOrderingTwoClaudeVersions` is 22 pairs, chosen as the ones string
comparison gets backwards and the ones a naive split answers confidently.

**Break-tests — 4 run, 4 fired**: tables attributed to `2.0.9` (NEWER branch
under a real finding), to `2.9.0` (OLDER branch), to a version whose field
overflows an int (the unorderable branch), and the comparator replaced by
string comparison — which fired 18 of the table's 22 cases, including the pair
the item is about.

---

## Item 2 · a floor that was a third of one reading

`wholeWalkReadsFloor = 1000`, in a run that fetches 2906, in a file whose whole
subject is numbers taken once. What it guarded was real — "one process served
eleven objects" passes a process count while saying nothing — but if `core/`
halved it would silently stop discriminating, and a walk that broke while still
fetching 1200 passed it.

The expectation now comes out of the repository on every run. `leavesAt`'s
four-line filter became **`themeSourcesAt`**, which is the walk's answer to
*which objects will be fetched* — and the arm sums it across the history and
holds `batchReader.reads` to **exactly** that number. A floor became an
EQUALITY, which fails in both directions: a revision the walk skipped, and an
object fetched twice.

Extracted rather than re-spelled in the test on purpose: a test carrying its
own copy of the suffix rule and the directory rule would be asserting one copy
against another, and both can be wrong together.

The enumeration costs one `ls-tree` per commit — **857ms**, which is recorded
rather than absorbed (see item 4). `go run ./internal/themehistory` is
**byte-identical to HEAD** with nothing on stderr, compared against a scratch
worktree.

**Break-tests — 3 run, 3 fired**: every source fetched twice (`2906 over`), two
sources per revision skipped (`short by 176`), and the enumeration made to
decline everything — which fires the arm that keeps an expectation of nought
from passing a walk that fetched nothing.

---

## Item 3 · a number nothing could re-take

"Thirty-one seconds" was taken once, in the session that deleted the code
which cost it. The previous session got one step closer — a break-test broke
`blob` into retiring its reader per fetch, measured 31.8s, and recorded a
sentence saying so — which is the same shape one step along: **a reading
believed because a past session says it took it.**

So the shape is re-created rather than described, and not by mutating the
program (a test that breaks the code to measure it is measuring the mutation).
`TestOneProcessPerObjectIsSlowerThanOneProcessForAllOfThem` fetches every
object the walk names **both ways in one run**, off unless
`GRMOB_PER_OBJECT_FETCH=required` — the `GRMOB_COMPOSE_SOURCES` idiom, where a
typo is a failure that names itself rather than a switch that silently did
nothing.

    one `cat-file -p` each   30.14–30.42s   2906 processes
    one `cat-file --batch`   399–401ms      1 process
                             ─────────────  75–76× on the fetches alone

Asserted: **the same bytes** through both routes at every revision the table is
built from (`TestTheBatchedFetchReadsWhatCatFileDoes` makes that comparison at
HEAD only), **one process** for all of them, and **the direction**. Not the
multiple — 75× is a reading of this machine's fork cost against its pipe cost.

The thirty-one seconds turns out to have been right, and it is no longer what
the argument rests on. `blob`'s comment now carries the ratio, and both halves
of it are re-takeable by somebody with one environment variable.

**Break-tests — 3 run, 3 fired**: `GRMOB_PER_OBJECT_FETCH=1` (refused by
name), the batched bytes trimmed (50 of 50 objects reported, first named), and
the per-object clock divided so forking looks free.

---

## Item 4 · a cost with no lever

The whole-walk arm is the one test here that runs the real program over the
real history, it roughly doubles the package, and it is paid again under
`-race` on every `go test ./...`. The previous session named that and left it.

It stays on by default — a claim nobody checks on a green run is the state it
was written to end — and it now has a lever and a price list:

    plain    3.77–3.93s   default      1.31–1.40s   -short
    -race    7.36–7.60s   default      2.53–2.55s   -race -short

So the arm is ~2.5s of a plain run and ~4.8s of a `-race` one, and `-short`
skips it and nothing else in the package. The skip message says what is not
asserted on that run and that no clock was taken.

The other half of the item is written down rather than fixed, because it is
correct: the arm needs fifty commits touching `core/` and skips below that, so
**the figure in `wholeRun` can only ever be re-taken on a full checkout**.

**Measured, not broken**: the four figures above are three runs each.

---

## Item 5 · a direction nobody had named

`timingsRecordCopies = 2` fails a correct third record — the same failure
direction item 1 apologises for. It is deliberate, and "deliberate" was
sitting in the mechanism rather than in a sentence.

Now it is a sentence, and the contrast is the content:

    the hook tables    the failure is a MISTAKE about another program's
                       schema. Nothing here can tell it from a release that
                       moved, and softening it would let `if` sit in
                       settings.json again — so the finding stays and arrives
                       with two versions, which is all that side can offer
    this constant      the failure IS the prompt. The third record is the
                       moment the trade changes, the change wanted is in THIS
                       repository, and the message asks for an extraction or a
                       written reason

Same cost — somebody reads a message and edits a file. The difference is that
one of them is asking for a decision it can name.

**Break-test — 1 run, 1 fired**: a third `…TimingsTakenOn` in
`internal/themeleaves`, which produced both the count and the request for a
reason rather than a digit.

---

## Item 6 · a global with one reader

`batchesStarted` is a package-level atomic counting for the lifetime of the
test binary, read by one test that snapshots before and after. Trivial today;
the kind of global that is only trivial while there is one reader.

`batchesStartedSince()` is the discipline written down once — a closure over
the value at entry — and the counter's comment says why **nothing resets it**:
a reset would lose the retire-and-replace the counter exists to notice, and it
would silently subtract from any delta another test had open.

**Break-test — 1 run, 1 fired**: the delta replaced by an absolute read, in a
full-package run — the walk reported **14** `git cat-file --batch` processes,
which is every reader the package's other tests had started. The delta is
load-bearing and now says so.

---

## Item 7 · an approximation on its first rung

`whyNotAGitWrapper` answers a dataflow question syntactically because this
package parses FILES rather than loading packages — a good reason, written down
once, and now carrying two approximations rather than one. The other is
`themenearmiss`'s float census, which declines the type checker in the same
words and has since grown four name-keyed tables and a count of the collisions
it knows it cannot resolve. **That is what this one looks like after several
sessions of obviously-correct additions.**

So the rules are counted rather than remembered.
`TestTheGitWrapperTaintWalkIsTheSizeItsReasonCovers` parses this file, counts
the `reaches = true` in that function — an acceptance rule is a place it
decides the premise holds — and holds three things:

    a rule per row      gitWrapperTaintRules names each rule and the case in
                        TestWhatCountsAsAGitWrapper that exercises it
    one way out         exactly one `return ""` in the body, so an acceptance
                        cannot be spelled around the counter
    two, and the third  gitWrapperAcceptRules = 2 — the number the written
                        reason covers. The third fails, and the message asks
                        for go/types or a written reason

`gitWrapperTaintLimits` lists the three loosenesses beside them, all in the
direction of accepting a helper, because a third rule would be arriving on top
of those and that is the question at that point.

**Break-tests — 4 run, 4 fired**: a third acceptance rule (both the census
mismatch and the budget), a case renamed out of the table, a second
`return ""`, and the function itself renamed.

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
    internal/themehistory/main.go           items 2, 3, 6
    internal/themehistory/timings_test.go   items 2, 3, 4, 6
    internal/themehistory/main_test.go      items 2, 3
    wasm/verify/timingsrecords_test.go      item 5
    wasm/verify/gitquoting_test.go          item 7
    wasm/verify/timings_test.go             re-taken

`go run ./internal/themehistory` is **byte-identical to HEAD** with **nothing
on stderr**, compared against a detached worktree at HEAD rather than a stash,
because the extraction is in the fetch path. **16 break-tests run, 16 fired.**

Both packages' figures re-taken over seven runs each: `wasm/verify` 2.51–2.58s
→ **2.49–2.59s** (item 1's `claude --version` fork and item 7's parse are
inside the old spread), `internal/themehistory` 2.84–3.08s → **3.77–3.93s**,
which is the old figure plus the whole-walk arm's 1.5s plus item 2's 857ms
enumeration — accounted for in the record, and kept apart from the unexplained
60% gap sitting in the same comment.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

1. **(age 0 · value medium) The equality costs 857ms to assert 2906.**
   Item 2 replaced a constant with a derived number and the derivation is 88
   `git ls-tree` processes — 35% of the arm's wall clock, paid on every green
   run and again under `-race`, to turn a floor into an equality. The cheap
   alternative was a floor scaled off the files at HEAD (one `ls-tree`), which
   adapts if `core/` halves and does not catch an off-by-one. The equality is
   worth more and the price is now the largest single thing `-short` skips;
   what would settle it is knowing whether an off-by-one in the fetch path is
   a failure mode anybody expects, and nobody has asked.
2. **(age 0 · value medium) The per-object arm asserts a direction that
   cannot fail.** `perTook <= batchTook` over 2906 objects is 30s against
   400ms; no machine reverses it, so the assertion is decoration and the real
   content is the two numbers in the log. The honest arm is the RATIO with a
   floor — "at least 5×" would fail on a machine where fork is nearly free
   and would still be a claim about the argument rather than about a clock —
   and it was left out because a multiple is a wall-clock assertion wearing a
   ratio's clothes. One of those two readings is wrong.
3. **(age 0 · value medium) `compareVersions` is a fourth version comparator
   in this repository's neighbourhood.** `mobile/verify` derives a Compose
   release out of a BOM pom, `hookconfig` now orders dotted versions,
   `timings` compares Go versions by string prefix, and `golang.org/x/mod`
   does all three properly and is not a dependency this go.mod has. Twenty
   lines was the right call for one caller; the third caller is where it stops
   being, and the place to notice is the same shape item 7 just wrote an arm
   for.
4. **(age 0 · value low) `gitWrapperAcceptRules` counts `reaches = true` and
   trusts the shape of the function.** The arm holds the body to one
   `return ""`, which closes the obvious way around the counter, and it cannot
   see an acceptance smuggled into a helper the function calls — a
   `reachesVia(...)` that returns a bool would be a rule the census never
   hears about. That is the same class of gap as the taint walk's own, one
   level up: a syntactic count of a syntactic thing.
5. **(age 0 · value low) `-short` is now load-bearing and is asserted
   nowhere.** One test reads `testing.Short()` and nothing checks that the
   short run still exercises what the package claims — a second arm added
   under the same skip, or a `-short` that starts skipping the wrong thing,
   would show up as a fast green run. The repository has no `-short`
   convention beyond this one call, so either it stays a single lever with a
   comment or it becomes a rule with an arm; today it is neither.
6. **(age 0 · value low) The batched figure in `perObjectRun` is a third
   reading of the same fetches.** `wholeRun` is 1.49–1.61s for the walk,
   `perObjectRun` carries 399–401ms for the fetches alone, and the difference
   is the `ls-tree` processes plus every revision's parse — which is stated in
   a sentence in the record and asserted by nothing. It is the one number pair
   in either record that invites a subtraction, and a reader who does it gets
   a figure nobody has measured.
7. **(age 0 · value low) `themeSourcesAt` is now called twice per run in the
   tests and once in the program.** The walk calls it per revision, the
   whole-walk arm calls it again for the expectation, and the per-object arm a
   third time. That is intentional — the expectation has to be independent of
   the fetch — and it means a slow `ls-tree` shows up three times in the
   package's wall clock. Worth a note only because the previous session removed
   a "walk taken twice in one run" and this is one arriving by a different
   door, for a better reason.
