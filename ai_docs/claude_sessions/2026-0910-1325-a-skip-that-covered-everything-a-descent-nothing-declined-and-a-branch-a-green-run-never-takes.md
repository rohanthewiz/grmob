# Session: a skip that covered everything, a descent nothing declined, and a branch a green run never takes

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-10 (follows "a-field-spelling-that-produced-no-name-a-parse-error-nothing-reported-and-five-categories-inside-a-word-that-was-reached")

## Ask

"Do all items in the Next list." Seven items, six raised in the previous
session and one a standing non-goal. Six files edited, no new files.

The previous session's shape was "a mechanism whose only test is one answer".
This one's is the state of the arms those answers live in: **an arm that does
not run is not an arm, whatever it reports.** Three of the six items are one
arm each that was off — off whenever anybody was working, off for every
revision but one, off on every green run — and the fix in each case was to
make it run on less rather than to leave it covering nothing.

| item | where | shape |
|---|---|---|
| 1 | internal/themehistory | a skip that covered everything, made into a hole |
| 2 | internal/themehistory | a descent nothing declined and nothing reported |
| 3 | main_test.go | a key that turns a difference into a nameless one |
| 4 | inkglyph_test.go | a loop order argued from its own reason |
| 5 | themenearmiss_test.go | a branch a green run never takes |
| 6 | three files | a rule in three copies, none of them saying so |
| 7 | — | a sound chain bound at k = 3, declined again |

---

## Item 1 · a skip that covered everything

`TestTheRevisionsFileSetIsTheOneTheWorkingTreeWalkReads` compares the two file
filters at HEAD, and it skipped whenever `core/` differed from HEAD. That skip
is honest — the two readings would be over two different trees — and it is
all-or-nothing, so the arm ran on a clean checkout and in CI and never for the
person editing `core/`, who is the person who would move a filter.

`git status --porcelain` already names the paths that differ, so it does not
have to be all-or-nothing:

    core/  theme.go   colors.go   type.go   sizes.go
                       modified               new, untracked
           └─ compared ─┘        └─ compared ─┘
                       └──── held out, and named ────┘

`-z` and not the default output, and this is the part worth writing down: plain
porcelain **quotes** a path holding a space, a quote or a non-ASCII byte —
`"core/a b.go"`, with the quotes as part of the line — and such a path would
match nothing coming out of `ls-tree` or off disk. It would be read as clean,
which is the one direction this must not fail in: it puts a file the two walks
disagree about back INTO the comparison and reports the disagreement as a
drifted filter.

Only one arm depends on the hole being empty rather than small. `Names` is a
reading of a whole file set and cannot be taken over part of one, so it says so
and stands down; `nested`, `Found` and `Unparsed` read HEAD alone and are
unaffected by anything on disk. The hole is stated in the failure message and
in the log line alike.

**Two runs demonstrating it**, both previously a total skip: `core/theme.go`
edited — **48 of 49 compared**, `theme.go` named; an untracked `core/zz.go` —
**49 of 49 compared**, the new file named.

---

## Item 2 · a descent nothing declined

`ls-tree -r` descends into subdirectories of `core/` and `themeleaves.InDir`
does not. So a revision that split `core/` into subpackages was parsed WITH
those files, produced a row in the edit-size table, and every arm that checks
this expansion runs against `InDir` at HEAD and had nothing to notice.

    revision:      ls-tree -r ──> core/theme.go, core/sub/theme.go
    working tree:  ReadDir    ──> core/theme.go
                                  └─ one package, one directory

`leavesAt` takes `InDir`'s rule now and **names what it drops**: a `revision`
type embedding `themeleaves.Expansion` plus `nested`, reported on stderr per
revision the way `Unparsed` is, and an arm at HEAD.

`-r` is still asked for, deliberately. A non-recursive listing reports a
subdirectory as one tree object and never mentions the `.go` files inside it,
so the paths this walk declines would be paths it could not name. `-r` is how
they are seen; `revision.nested` is where they go.

The nested arm is its own arm and not part of the file-set comparison, which is
the point: both walks decline these files identically, so the two readings
agree — about a population neither of them is over.

The table is **byte-identical** before and after, with nothing on stderr.

**Break-test — a real revision.** A throwaway clone, `core/sub/extra.go`
committed, arm fires naming it; the file-set arm stays green, as designed.

---

## Item 3 · a difference the base name cannot name

The two file sets were compared by base name — which is what made the
comparison possible at all, since git's paths are repository-relative and
`InDir`'s are joined onto the directory it was handed. It equates
`core/sub/theme.go` with `core/theme.go`, which is exactly the case item 2 is
about.

The payoff turned out to be sharper than "a wrong match". Against a revision
holding `core/sub/theme.go`, with the descent put back:

    relative   50 against 49 — only the revision walk: sub/theme.go
    base name  50 against 49 — only the revision walk: nothing
                               only the working tree:  nothing

Because `theme.go` is then in the revision's list **twice**, and `missing` is a
set difference: every name one side has, the other has too, and the lists are
still different lengths. The failure is real, it is loud, and it names no file.

Keyed on the path relative to `core/` now, which both sides can produce. A path
`filepath.Rel` cannot resolve is kept whole — the loud version of the failure,
rather than a silent fall back onto the base name, which would match one it is
not.

**Break-tests — 4 run, 4 fired.** A filter drift dropping `core/alignment.go`;
the nested arm above; and the two keys against the same nested revision, which
is the pair printed above.

---

## Item 4 · being worded, and being worded first

`foldCategoryOf` tries every entry of `foldRegionKinds` and then falls through
to a walk of `unicode.Categories` for anything two letters long.
`unicode.Categories` holds `"LC"` — Go's union of Lu, Ll and Lt, not a category
a code point is IN — and it is two letters long.

    foldCategoryOf(cp)
      ├── foldRegionKinds   Lu Ll Lt Lo Lm Nd ...   ← 'A' stops here
      └── unicode.Categories, len(name) == 2        ← and "LC" is in here

`TestTheRegionKindsAccountForEveryCategoryGoHas` asserts that Lu, Ll and Lt are
worded, as LC's whole excuse for needing no word. **That is not this claim.**
It says nothing about them being tried FIRST, which is the property the
function rests on — reorder the two loops for tidiness, with all three still
worded, and this file starts recording LC without an arm moving.

So it is walked. First that the hazard is real — LC is in the table, is two
letters, is not `Cn`, and holds `'A'`, so nothing but the first loop answering
stands between a cased letter and `"LC"` — then **all 4095 code points in Go's
LC table**, through the function itself.

**Break-tests — 2 run, 2 fired.** The loops swapped, caught at `U+0041 "A"`;
and `Lt` moved out of `foldRegionKinds` into the unworded list, caught at
`U+01C5 "ǅ"` — a titlecase letter, which is the only kind that could have.

---

## Item 5 · the branch a green run never takes

`affordedHoldBandAttribution` asserts every key in the tally is a test name,
which makes `affordedBandCaller`'s `file:line` fallback the branch no passing
run enters. Three lines, read under pressure by somebody meeting an unfamiliar
key in a failure message, never once run. It is also the whole content of a
claim made in prose beside it.

    affordedBandCaller() called from        frames above it        answers
      the test body                         TestFoo                TestFoo
      a goroutine the test started          TestFoo.funcN, goexit  file:line
      a t.Run closure                       TestFoo.funcN, tRunner file:line

`affordedCallSite` is `runtime.Caller(1)` in the fallback's own spelling, so
the two can be taken on ONE source line —
`got, want := affordedBandCaller(), affordedCallSite()`. Written out separately
the answers would differ by one and the assertion would be approximate, or
would break the day somebody inserted a blank line. All three stacks confirmed;
nothing touches the memo, since a stack walk does not count.

**Break-tests — 2 run, 2 fired.** `runtime.Callers(3, …)`, where the ordinary
case answers `testing.go:2036` — the fallback naming testing's own frame; and
the fallback keeping the full path instead of the base name.

---

## Item 6 · three copies, each saying which it is

    wasm/verify/gen.go                    themeLeafPaths     the walk, as paths
    wasm/verify/themenearmiss_test.go     affordedLeafNames  the dedup on it
    internal/themeleaves/                 reflectLeafNames   both collapsed,
      themeleaves_test.go                                    over reflect

Two of those are one mechanism in two pieces. The third could not be: it is
what the PARSE is held against, and a comparison whose two sides share code is
a comparison of a thing with itself. The cost of that argument is a copy nobody
else reads — which is exactly the copy that can drift without a failure — so it
is named at all three sites rather than left to be found as a coincidence.

---

## Item 7 · declined again

A sound chain bound at k = 3. `affordedTwoStepBands` exists only for two.
Bounding the last step over every chain needs a census of every population of
size n−k, which is the family the direct measurement walks — and the direct
measurement is the cheaper of the two at every k this file takes. Kept at
k = 2 for the SHAPE it holds, not as a route to anything.

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

Six files, +630 −77, no new files.

    internal/themehistory/main.go              item 2 (revision, nested)
    internal/themehistory/main_test.go         items 1, 2, 3
    internal/themeleaves/themeleaves_test.go   item 6
    wasm/verify/gen.go                         item 6
    wasm/verify/inkglyph_test.go               item 4
    wasm/verify/themenearmiss_test.go          items 5, 6

`go test ./wasm/verify` is **1.79–1.81s against the recorded 1.88s**.
`go run ./internal/themehistory` reproduces the recorded histogram and the
eighty names exactly, before and after the nested filter, with nothing on
stderr. 8 break-tests run, 8 fired, plus two runs demonstrating item 1.

**One mishap worth recording.** A `git checkout internal/themehistory/main.go`
run to undo a break-test discarded that file's real edits along with it. They
were re-applied from the edit script, and every break-test after that point
restored from a file copy in the scratchpad rather than from git. The final
state is verified by the full run above and not by that recovery.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

1. **(age 0 · value medium) The name-for-name arm goes silent on any dirty file
   the walks reach, and it does not have to.** `Names` is a whole-reading
   quantity, which is why it stands down — but the reading could be re-taken
   over the kept subset on both sides: `themeleaves.Of` is exported and takes a
   sources map, so the test could `cat-file` the kept paths out of HEAD, read
   the same paths off disk, and compare two expansions over one file set. It
   costs a fetch per file and it would stop this arm being off exactly while
   somebody is editing `core/` — which is the same argument item 1 made about
   the skip, one level down.
2. **(age 0 · value medium) The nested arm fails on a state nobody has
   decided.** A tracked `.go` file below `core/` makes it error, and its
   message asks the reader whether those files declare anything `core.Theme`
   reaches. That is a question the arm could answer itself: fetch them, run
   `Of` over the union, compare `Names`. Unchanged, it is a `Logf` naming a new
   directory; changed, it is the finding the message currently guesses at. As
   written, a legitimate `core/internal/` would be a red build with a paragraph
   asking somebody to think about it.
3. **(age 0 · value low) `dirtyPaths`' rename branch is exercised by nothing.**
   Both halves of a rename are held out, which is right — the source is at HEAD
   and gone from disk and the destination is the other way round, so each is a
   path exactly one walk reads — and no break-test produced one. A `git mv`
   inside `core/` is the only shape that puts two paths in one record, and it
   is the branch that decides whether the hole is stated completely.
4. **(age 0 · value low) `underPkg`'s `filepath.Rel` error path has never
   run.** It keeps the whole path on purpose, so the file-set arm names it
   rather than silently matching a file it is not — the loud branch. It is
   loud inside a message about two filters having come apart, which is not what
   an unresolvable path means, and nothing has ever produced one.
5. **(age 0 · value low) The nested revision can only be made by hand.** The
   clone-and-commit that fired the nested arm is written down in this doc and
   nowhere in the repository, so a later reader has the arm, the message, and
   no way to watch it fire. A fixture cannot carry it either — the arm is about
   git's own tree — so the honest fix is a recipe in the test's comment rather
   than a testdata directory.
6. **(age 4 · deliberate non-goal) A sound chain bound at k = 3.** Moved out
   of this list to `ai_docs/plans/non_goals.md`, which is where declined work
   now lives so it is decided once rather than re-declined at the bottom of
   every session. The argument is unchanged and `affordedTwoStepBands` carries
   a pointer to it.
