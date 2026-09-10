# Session: a Next list that was already done, a choice made by sort order, and forty-four hundred processes for sixteen rows

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-10 (follows "a-skip-that-covered-everything-a-descent-nothing-declined-and-a-branch-a-green-run-never-takes")

## Ask

"Do all items in the Next list." **There was no list.** The newest saved doc's
seven items were all done by commit `8d14384`, which saved no session doc of
its own — so the follow-ups that session generated exist nowhere, and the only
record of the work is its commit message.

So the list was rebuilt from the code that session left, which is what
`/next-list` asks for in the general case and what it had to do literally here.
Five items, four files edited, no new files.

The shape this session found is one level up from the last one's. That one was
about arms that do not run. This one is about **decisions made in passing**: a
resolution taken by string comparison and never mentioned, a cost paid four
thousand times because nobody added it up, and a switch whose branches were
prose because only a broken repository could reach them.

| item | where | shape |
|---|---|---|
| 1 | internal/themeleaves | a choice made by sort order and never said |
| 2 | internal/themehistory | 31 seconds, none of it git working |
| 3 | main_test.go | three branches only a broken tree could run |
| 4 | main_test.go | a shape the table claimed to hold and did not |
| 5 | main_test.go | one rule, two copies, twenty lines apart |

---

## The list itself, which is the first finding

`/next-list` over the last six docs found no lapsed items: every session's Ask
confirms it consumed the previous list whole, and the chain is clean from
`1022` through `1325`. What it found instead is a leak of a different kind.

    doc 1325  ── Next: 7 items ──┐
                                 ├── 8d14384  does all 7. Saves no doc.
    (no doc)                     │            Its own follow-ups: nowhere.
                                 ▼
    this session   ── rebuilds the list by reading the code

A list carried forward loses items one at a time. A session that ships without
a doc loses the whole list at once, and it does not read as a gap: the last
doc's Next section is still there, still looks live, and every item in it is
done. The five below came out of reading `8d14384`'s code rather than any
document.

---

## Item 1 · a choice made by sort order and never said

`themeleaves.Of` keys every struct it parses by its BARE NAME in one flat map,
so a second declaration of a name overwrites the first and the winner is
whichever path sorted last. The comment excusing that said a package declaring
one name twice does not compile.

**That premise stopped being true when the previous session wrote
`unionWithNested`,** which hands `Of` the top level of `core/` and a
subdirectory of it — two packages, on purpose. And it was never true of the
history walk, which parses all 88 commits that touched `core/`, including any
caught mid-refactor.

The cost of leaving it silent is that the expansion comes back *whole and
plausible over the wrong declaration*. `Expansion.Shadowed` records it:

    Shadow{Name: "SpacingScale",
           Files: ["core/theme.go", "core/zsub/extra.go"]}  ← winner last

`Files` in parse order with the winner last, because that is what a reader with
a moved population needs. A path listed twice is one file declaring the name
twice, recorded as read rather than deduplicated — the count is how many
declarations there were.

**Three consumers.** The command prints it per revision on stderr the way it
prints `Unparsed`. `main_test.go` asserts it is empty at HEAD — where it is
empty for a reason outside this package's control, so a non-empty one is a tree
that would not compile or `Of` having stopped keying on the bare name. And
`reportNested` **answers the question it used to ask**: it printed both possible
causes of a moved population and told the reader it could not tell them apart,
and it now names the collisions or says there are none.

    1 bare name(s) are declared both in core/ and below it: SpacingScale
    (declared in core/theme.go, core/zsub/extra.go; the last of those is
    the one that answered).

**Measured:** none of the 88 revisions declares a struct name twice, so the
record costs the table nothing and the command's stderr stays empty.

**Break-test — 1 run, 1 fired.** The path sort reversed: both rows of the
direction table flipped, naming `sub/` and `zsub/` as the two spellings of the
hazard.

---

## Item 2 · thirty-one seconds, none of it git working

`go run ./internal/themehistory` took **31.3 seconds**. 88 commits touch
`core/`, and each was read with one `ls-tree` plus one `git cat-file -p` per
non-test `.go` file in it — **around 4400 git processes for a table of sixteen
rows**. Almost none of that is git reading anything: it is fork, exec, opening
the repository, and tearing the process down, 4400 times.

    before   ls-tree ── cat-file ── cat-file ── cat-file ── ...   per revision
    after    ls-tree ── ┐
                        └── one `cat-file --batch`, for the whole run

**31.3s → 1.5s**, stdout byte-identical, nothing on stderr. 89 processes
instead of 4400. `--batch` and `-p` both write a blob raw, with no filters and
no line-ending conversion, which is what makes this a change in how the text is
fetched and not in what it says.

What it trades is a process boundary for a **parse**, and that is the part worth
writing down. `-p` hands back a process's whole stdout and cannot return the
wrong thing. `--batch` is a stream:

    <oid> SP <type> SP <size> LF   header
    <size> bytes                   the object, raw
    LF                             a terminator NOT counted in the size

The body is read by count, and the trailing LF is consumed separately. A reader
that leaves that byte unread is one byte into the next header, and **every file
after it in the run is wrong** — which would not look like a failure: go/parser
would be handed text beginning mid-file, and a short expansion reads as a commit
that removed leaves.

So the two fetches are held against each other over all 49 files the walk reads
at HEAD, in walk order, because the failure is positional and a spot check of
one file is the one check that cannot see it. A second arm pins the only error
the reader can carry on from: **`missing` is a complete response** — one line,
no body — so a request at the wrong revision costs an error and leaves the
stream sitting at the next header.

One shape goes back to a process of its own: a path containing a newline cannot
be asked for down a newline-terminated protocol, so it falls back to
`cat-file -p`. Nothing here has one and git will track one, and the file-set arm
is specifically about paths that need quoting.

`expansionsOver` and `unionWithNested` were moved onto the same reader, so
fetching is one mechanism rather than three.

**Break-tests — 2 run, 2 fired.** The terminator read dropped, caught on the
second file; and `missing` treated as an empty blob rather than an error.

---

## Item 3 · three branches only a broken tree could run

`expansionsOver`'s four outcomes were a switch inline in the arm, and only the
last case ever ran. The other three need a fetch that fails between one
`ls-tree` and the next read, a working tree whose `theme.go` is dirty, or a
`git cat-file` that disagrees with the disk about a file `git status` calls
clean.

Every input the decision uses is a value — two expansions, an error, two path
lists — so `readSubset` is a pure function and the four outcomes are a table.
What stays behind in the arm is the FETCH, which is the part that genuinely
needs a repository and a dirty tree.

    err != nil       a failure. The paths were all read moments ago
    neither Found    a note. theme.go itself is held out, so both
                     re-readings are empty and equal for no reason
    one Found        a failure, and a strange one: same bytes, two answers
    both Found       the comparison the caller wanted, over a SUBSET

What is asserted is the decision — failure, note, or nothing, and whether there
are two lists worth comparing — plus the facts each message must carry. Not the
wording: a test that pinned the prose would fail on every rephrasing and would
be asserting that nobody had edited a paragraph.

**Break-test — 1 run, 1 fired.** The "one reading found it" case made
non-fatal.

---

## Item 4 · a shape the table claimed to hold and did not

`TestARenameUnderThePackageHoldsOutBothOfItsPaths` says its table holds "every
shape one record can have". It did not hold the untracked DIRECTORY:
`git status --porcelain` collapses one to a single record with a trailing
slash rather than listing what is in it.

    ?? core/zsub/          one record, whatever is inside

Added, and it produced a small correction on the way in. The key comes out as
`zsub` and not `zsub/`, because `relToPkg` goes through `filepath.Rel`, which
cleans its result. Either spelling is inert — the dirty set is only ever
consulted for paths one of the two walks RETURNED, both return `.go` files, and
neither descends — so nothing turns on which it is, and "inert" is now
something this file has checked rather than assumed.

---

## Item 5 · one rule, two copies, twenty lines apart

`underPkg` keyed the walks' file lists relative to `core/`, and `dirtyPaths`
keyed `git status`'s paths the same way with its own inline closure — the same
five lines twice, in one file, at the top of a test whose whole subject is a
rule existing in two copies.

Two spellings would not fail loudly: a dirty file keyed one way and looked up
the other is simply never found, so it would be compared rather than held out,
and the arm would report an uncommitted edit as the two file filters having come
apart. Which is the exact failure this test exists to tell apart from a real
one, arriving through its own keying.

Now `relToPkg`, and `TestAPathThatCannotBeMadeRelativeIsKeptWhole` covers both
callers.

**Break-tests — 2 run, 2 fired.** A base-name fallback in the shared rule; and
the rename source record left unconsumed, which reproduced the documented
invention exactly — `../e/from.go` and `../e/origin.go` in place of `from.go`
and `origin.go`.

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

Four files, +910 −85, no new files.

    internal/themehistory/main.go              items 1, 2
    internal/themehistory/main_test.go         items 1, 2, 3, 4, 5
    internal/themeleaves/themeleaves.go        item 1
    internal/themeleaves/themeleaves_test.go   item 1

`go run ./internal/themehistory` reproduces the recorded histogram
**byte-identically** with nothing on stderr, in **1.5s against 31.3s**.
`internal/themehistory`'s own tests are 0.83s with two new arms, against 1.24s
with none. 6 break-tests run, 6 fired, plus three end-to-end demonstrations in a
throwaway clone: a benign nested revision, a colliding one (which named
`SpacingScale` and which file answered), and a revision declaring a name twice
at the top level (which produced the new stderr line).

**One number that moved and is not this session's.** `go test ./wasm/verify` is
**1.99–2.07s** on this machine against the 1.88s recorded in the file and the
1.79–1.81s measured last session. Checked by stashing: HEAD alone gives
2.02–2.03s, so it is the machine and not these changes. Nothing in
`wasm/verify` was touched.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

Every item is age 0, and that is a fact about the previous session rather than
about this one: `8d14384` shipped without a doc, so nothing was carried in.

1. **(age 0 · value medium) A session shipped without a doc and its Next list
   is gone.** Not a code item. `8d14384` did five items of real work, wrote a
   long commit message, and saved no session doc — so the follow-ups it
   generated had to be re-derived from its code here, and anything it noticed
   and did not write into the code is lost. `/sess-wrap` exists to make the doc
   and the commit one action; what is missing is anything that notices the
   commit happened without one. The cheapest form is a note in `/commit`'s
   skill, and the honest one is that the two commands are the same command.
2. **(age 0 · value medium) The recorded 1.88s for `wasm/verify` is a number
   from another machine.** It reads 2.0s here at HEAD, which is 6% out, and the
   file re-walks its own census on every run precisely because a number in a
   note is a number that has already moved. Every other measurement in that
   file is re-derived; the timings are the ones still written down. A timing
   cannot be an arm — it would fail on a loaded machine — but it could carry the
   machine it was taken on, which is the difference between "this got slower"
   and "this is a different computer".
3. **(age 0 · value low) The batch reader has no way to die.** `blobs` is a
   package-level reader started on first use and never closed, and every error
   but `missing` leaves the stream at an unknown offset — after which every
   later call in the process reads from the wrong place and nothing says so.
   Marking it dead on any such error and starting a fresh one would recover, at
   the cost of a process; leaving it as it is means one bad response poisons a
   run. Nothing has ever produced one, which is the same sentence this file has
   written about four other branches.
4. **(age 0 · value low) The batch process's working directory is fixed at the
   moment it starts.** Every test that fetches through it happens to `t.Chdir`
   to the repository root first, and `t.Chdir` restores the old directory
   afterwards while the git process keeps the one it was born in. It is benign
   — `<rev>:<path>` resolves from the top of the tree, not from `cwd`, so only
   repository DISCOVERY depends on it — but "benign because of the order two
   tests happen to run in" is load-bearing and unstated.
5. **(age 0 · value low) `blob`'s newline fallback has never run.** A path
   containing a newline cannot go down a newline-terminated protocol, so it
   gets a `cat-file -p` of its own. Git will track such a path and the file-set
   arm is specifically about paths that need quoting, so the two halves of this
   file disagree about how exotic a path can be. The state is makeable — `git
   add` of a file whose name holds a newline — and the recipe is the same shape
   as the two already written into these tests.
6. **(age 0 · value low) `Shadowed` reports collisions anywhere, and the nested
   arm's sentence describes only one kind.** `Of` records any bare name it got
   twice; the message says "declared both in `core/` and below it", which is
   the union probe's case. Two nested directories shadowing each other, or two
   files below `core/` declaring one name, would print that sentence and it
   would be describing something else.
