# Session: a field spelling that produced no name, a parse error nothing reported, and five categories inside a word that was reached

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-10 (follows "an-expansion-checked-by-one-diff-a-width-two-sentences-never-named-and-thirteen-reasons-nothing-read")

## Ask

"Do all items in the Next list." Six items, five raised in the previous session
and one a standing non-goal. Four files edited, two test files added.

The previous session's shape was "a claim nothing re-derives". This one's is
narrower and came out of the work rather than being aimed at: **a mechanism
whose only test is one answer is a mechanism tested as far as that answer goes.**
Two live bugs fell out of writing the first such test, and a third reading
turned out to be half a claim.

| item | where | shape |
|---|---|---|
| 1 | internal/themehistory | a filter in two copies, one of them upstream of every arm |
| 2 | internal/themeleaves | a walker held to one struct's worth of its own rules |
| 3 | themenearmiss_test.go | an attribution that was a stack convention |
| 4 | inkglyph_test.go | a column only a machine with node could check |
| 5 | inkglyph_test.go | six words standing in for seventeen categories |
| 6 | — | a sound chain bound at k = 3, declined again |

---

## Item 1 · the half above themeleaves

`themeleaves.InDir` is held against reflect at HEAD on every run. What that
arm cannot see is the half above it: `leavesAt` picks a revision's files with
`ls-tree` and its own `.go` / `_test.go` test, and `themeleaves.Of` picks again
with a second copy of the same rule. Two copies, and the arm is downstream of
both — it hands `InDir` a directory and never goes through `leavesAt` at all.

    go run ./internal/themehistory
              │
              ├── leavesAt(sha) ── ls-tree ── cat-file ── filter  ← only here
              │                                             │
              └───────────────────────────────────── themeleaves.Of
                                                            │
       wasm/verify ── themeleaves.InDir ── ReadDir ── filter ┘  ← and here

`Expansion` carries **`Files`** now — every path that got as far as go/parser,
sorted, with `Unparsed` a subset of it — so each reading says what it read.
`internal/themehistory/main_test.go` compares the two at HEAD, by base name,
and then the names those files gave.

It skips in exactly the two cases where the readings would be over different
trees: no git or no repository, and a `core/` that differs from HEAD. That is
the one place in this repository a skipping git test is honest, because the git
half **is** the subject — and the skip says which case it is rather than
reporting a pass.

**Break-test — 1 run, fired.** `leavesAt` given a drifted suffix test: 42 files
against 49, naming the seven, and the run returns before the downstream arms
echo it.

---

## Item 2 · the walker's own rules, and two bugs behind them

The arm in wasm/verify asks one question: does this package agree with reflect
about `core.Theme`. `core.Theme` holds no pointer to a struct, no embedded
field, no generic and no type from another package — so four of `walk`'s rules
were asserted by nothing, at three lines of source text each.

`themeleaves_test.go` declares the shapes as **real Go types** and hands
**this file's own source** to `Of`. The declaration reflect walks and the
declaration go/parser walks are then one text, with no fixture between them to
go stale.

Nine shapes where the two must agree — siblings of one type, a pointer beside a
value, an embedded value and an embedded pointer, a shared leaf name, an
anonymous struct, an empty struct, two names on one field, three levels — and
three where they must **not**, which is the documented edge asserted from both
sides: an embedded generic, an alias, a struct from another package. Each of
those asserts the field name the parse keeps, the name behind the stop that
only reflect reaches, and that neither list holds the other's.

### The two bugs

**An embedded field spelled anything but a bare identifier produced no name at
all.** Not a leaf under another name — nothing, and no descent either.

    Theme ──┬── *Palette        ← reflect: leaf "Palette";  this walk: nothing
            ├── image.Rectangle ← reflect: Min, Max;        this walk: nothing
            └── Sized[float64]  ← reflect: its underlying;  this walk: nothing

That is the one way this walker could come back with **fewer** names than
reflect rather than more, and it is the direction wasm/verify's failure message
is least specific about: a name only reflect has reads as "Theme is assembled
from more than core/" when it could equally have been an embedded pointer
sitting in it. `embeddedName` reads all five spellings the spec allows.

**`Unparsed` almost never fired.** It tested `f == nil`, and go/parser
*recovers*: a file with a syntax error comes back non-nil holding whatever was
readable. So the one case the field exists for — a revision caught
mid-refactor — was the case that passed through silently with a short
population. It reports on the **error** now, and the partial declarations are
still used, which is what its two callers were always told they were getting.

Neither bug moves this repository's history: the table is byte-identical before
and after both fixes, and no revision of `core/` reports a parse error.

**Break-tests — 4 run, 4 fired.** The guard reassigned inside the field loop
(the ninety-one-name bug: `[Caption Size Subtitle Weight]` against
`[Size Weight]`, and the self-holding case caught it too); embedded fields read
as bare identifiers only; a pointer to a struct made to descend; and the
`Unparsed` arm, which fired against the shipped code before the fix.

Also written down, because it is not the only possible rule: a self-reference
**keeps the field's own name**, which is what every other stop in this walk
does.

---

## Item 3 · t.Name() is the same fact from the side that cannot be wrong

`affordedBandCaller` attributes an asking to the outermost `Test` frame on the
stack. That is right for every caller today and it is a convention: a band
asked for inside `t.Run` has no `Test` frame above it — Go runs the closure on
its own goroutine and its name's last segment is `func1` — and so does one
asked for from a goroutine a test started. Nothing said the tally's keys were
test names, and the sentence it produces reads identically either way.

`affordedHoldBandAttribution` holds four things:

    this test is in the tally at all      else the fallback fired for the one
                                          caller there certainly is
    its share == the baseline delta       one number counted by the caller and
                                          by the callee
    Σ per-caller == the memo's totals     the totals are kept separately on
                                          purpose; this is the arm that says
                                          the second way gets the first answer
    every key starts with "Test"          the fallback is deliberate and is
                                          not what the tally is for

**Break-tests — 2 run, 2 fired.** Attribution forced to the file:line fallback;
and a reuse that moves the total without moving a caller's share — which fails
twice, on the share and on the sum.

---

## Item 4 · the record holds its own three columns

The derived arm compares the recorded regions against this build's, and it is
gated on node's Unicode being 16.0 **and** Go's being 15.0.0 — which is right,
since the spans come from one and the naming from the other. It leaves the
record checked by nothing on a machine with no node, or a newer one. The
sentence is fine there; a reader can tell "U+FF22..U+FF54 Latin letters" from a
wrong one. **The columns are not**: a kinds cell reading `marks` under a
sentence that says "letters" looks like nothing at all.

    U+2071..U+217C Common/Latin letters/modifier letters/numerals/symbols
    └─ range ────┘ └ scripts ┘ └────────── kinds, "/"-joined ───────────┘
       cats: Ll Lm Lu Nl Sc So  ── foldWordsFor ──> letters, modifier letters,
                                                    numerals, symbols

`TestTheRecordedRegionNamesHoldTheirOwnColumns` needs no Unicode data and no
child process: every sentence begins with a range, ends with its own kind words
`"/"`-joined, and those words are the recorded categories mapped through
`foldRegionKinds`. An empty column is allowed only under a sentence that says
why — which is the astral `U+1CCD7..U+1CCE9` entry and nothing else.

This is the parse the previous session deleted, back as an **arm** rather than
as the way the words are obtained. Read forward it was a second formatter
nobody would update; read backward against a column written independently it is
a check.

---

## Item 5 · five categories inside a word that was reached

`foldReachedWords` answered in words, and seventeen categories print six words.
An unreached word is sound — "marks" unprinted means neither Mn nor Mc nor Me
has turned up, because nothing else could have printed it. A **reached** word is
not a claim about its categories at all:

    letters   Lu Ll Lt Lo   printed, and Lo never has been
    numerals  Nd Nl No      printed by Nl alone
    symbols   So Sk Sm Sc   printed by So and Sc

So a region records its **categories** and the words are derived from them —
`foldCategoryOf` and `foldWordFor`, with `foldKindOf` now the composition of
the two. The record was re-taken with the third column against this build
(node 22.12.0 / Unicode 16.0, Go 15.0.0), and the sentences are unchanged.

The reading is at category granularity now: **7 of the 17 entries printed
(Lu Ll Lt Lm Nl So Sc), 10 never**, and it names the five that were
indistinguishable from reached at word granularity — **Lo, Nd, No, Sk, Sm**.

`foldCategoryOf`'s two loops are ordered, and the order is load-bearing:
`unicode.Categories` holds `"LC"`, which is two letters long, so the
fall-through would return it for any cased letter that got that far. Nothing
does, because Lu, Ll and Lt are all in `foldRegionKinds` and are tried first —
the same fact `foldUnwordedCategories` gives as LC's whole reason for needing no
word of its own.

**Break-tests — 3 run, 3 fired.** A kinds cell edited to `marks` (two arms: the
derivation from cats, and the sentence's tail); an empty-kinds region given
categories; and — on the derived arm — `So` swapped for `Sm`, which prints the
same word, holds the same sentence, and is a different population. That last is
the finding the third column was added to be able to have.

---

## Item 6 · declined again

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

Four files +768 −74, two new test files (764 lines).

    internal/themeleaves/themeleaves.go        items 1, 2 (Files, embeddedName,
                                               Unparsed on error)
    internal/themeleaves/themeleaves_test.go   item 2 (new)
    internal/themehistory/main.go              item 1
    internal/themehistory/main_test.go         item 1 (new)
    wasm/verify/themenearmiss_test.go          item 3
    wasm/verify/inkglyph_test.go               items 4, 5

`go test ./wasm/verify` is **1.85–1.95s against the recorded 1.88s**.
`go run ./internal/themehistory` reproduces the recorded histogram and the
eighty names exactly, before and after the two walker fixes, with nothing on
stderr. 10 break-tests run, 10 fired.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

1. **(age 0 · value medium) The file-set arm skips whenever `core/` is dirty,
   which is most of the time somebody is working in it.** The skip is honest —
   the two readings would be over different trees — but it means the arm
   effectively runs on a clean checkout and in CI and not for the person
   editing, which is the person who would move a filter. It does not have to be
   all-or-nothing: `git status --porcelain` already names the paths that
   differ, so the comparison could be taken over the files that do NOT, and
   report how many it left out. That turns a skip that covers everything into a
   reading with a stated hole.
2. **(age 0 · value medium) `ls-tree -r` descends and `InDir` does not, and only
   HEAD says so.** The new arm catches a divergence at HEAD; every historical
   revision is read by the recursive walk alone. A revision where `core/` held a
   subdirectory of `.go` files would have been parsed with those files included,
   and the row it produced is in the table. Nothing has ever put one there, and
   nothing would say so if a revision had. `leavesAt` could report the paths it
   skipped as not-top-level, the way it reports `Unparsed`.
3. **(age 0 · value low) The file sets are compared by base name.** Which is
   what makes the comparison possible at all — git's paths are
   repository-relative and `InDir`'s are joined onto a directory — and it means
   `core/sub/theme.go` and `core/theme.go` compare equal. That is exactly the
   case item 2 is about, arriving as a silence rather than as a difference. The
   comparison could be over paths made relative to `core/`, which both sides can
   produce.
4. **(age 0 · value low) Nothing asserts `foldCategoryOf` never returns "LC".**
   The whole ordering of its two loops rests on that: `unicode.Categories` holds
   `LC`, it is two letters long, and the fall-through would return it for a
   cased letter that reached it. It is true because Lu, Ll and Lt are worded and
   tried first — which is asserted, from the other end, as LC's reason for
   needing no word. The property this function actually depends on is one walk
   of those three tables away, and it would fail loudly the day somebody
   reorders the loops for tidiness.
5. **(age 0 · value low) `affordedBandCaller`'s fallback branch is exercised by
   nothing.** The attribution arm now asserts the tally holds only test names,
   which means the file:line path is the branch a green run never takes. It is
   three lines and it is the branch that fires on the day the convention breaks,
   so it is the branch that will be read under pressure and has never been run.
   A direct call to the helper from a goroutine, asserting the shape of what
   comes back, would cost nothing.
6. **(age 0 · value low) The rule "recurse on struct, take the last segment,
   keep each name once" now exists in three copies.** `gen.go`'s
   `themeLeafPaths` plus `affordedLeafNames`, and `themeleaves_test.go`'s
   `reflectLeafNames`. The third is deliberate — the whole worth of the
   comparison is that it does not share code with what it compares against — but
   three copies is one more than the argument needs, and the new one is the copy
   with no other reader. Worth a note at each site saying which of the three it
   is, or worth the test importing the wasm/verify pair if that package ever
   exports them.
7. **(age 3 · deliberate non-goal) A sound chain bound at k = 3.**
   `affordedTwoStepBands` exists only for two, and there is no reason to write
   the three-step version: bounding the last step over every chain needs a
   census of every population of size n−k, which is the family the direct
   measurement walks — and the direct measurement is the cheaper of the two at
   every k this file takes. The composition is kept at k = 2 for the SHAPE it
   holds, not as a route to anything. Listed so it is visibly declined rather
   than quietly missing.
