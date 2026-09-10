# Session: an expansion checked by one diff, a width two sentences never named, and thirteen reasons nothing read

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-10 (follows "a-shape-held-by-nothing-but-the-branch-it-reaches-five-probes-typed-beside-their-table-and-a-walker-that-came-back-with-ninety-one")

## Ask

"Do all items in the Next list." Six items, five of them raised in the previous
session and one a standing non-goal. Three files edited, one package added.
Every item was the same shape from a different side: a claim that was true,
that nothing re-derived, and that would go on reading as true after the thing
it describes had moved.

| item | where | shape |
|---|---|---|
| 1 | internal/themeleaves | two expansions agreeing, checked once by hand |
| 2 | themenearmiss_test.go | a width typed for a sentence that names none |
| 3 | inkglyph_test.go | a parser of this file's own formatter |
| 4 | inkglyph_test.go | thirteen reasons the arm never looked at |
| 5 | themenearmiss_test.go | a baseline that says THAT, not WHICH |
| 6 | — | a sound chain bound at k = 3, declined again |

---

## Item 1 · the half of the reading that never needed git

`internal/themehistory` expands core.Theme by parsing sources at a revision;
`affordedLeafNames` expands it over reflect. The whole edit-size table under
`affordedBandSteps` is the parsed population read at sixteen revisions, and it
is a fact about the population this file measures only while the two agree.
That was verified once, by hand, with a diff — the same shape as a number
written into a note.

The expansion moved into **`internal/themeleaves`**, so it can be imported:

    themeleaves.Of(sources, root)     path → source text, no git anywhere
    themeleaves.InDir(dir, root)      the same over a working tree

`themehistory` keeps the git half — `ls-tree`, `cat-file` — and calls `Of` for
the parse. `TestTheHistoryWalkersExpansionIsTheOneThisFileMeasures` holds
`InDir("../../core", "Theme")` against `affordedLeafNames()` name for name on
every run, sorted, because affordedLeafNames' order is the sorted PATHS' order
and themeleaves has no paths to sort by.

The two directions are not symmetric and the failure says so:

    only the source walk has it   a field whose type it could not resolve to a
                                  struct declared in core/ — it stopped and
                                  kept the FIELD's name
    only reflect has it           the children behind exactly that stop

`Expansion` also carries `Unparsed` and `Found`. Both are ordinary in a history
— a revision caught mid-refactor, a revision predating the struct — and
themehistory now says so on stderr rather than letting either read as a fact
about the struct; in a working tree both are Fatal.

**Break-test — 1 run, fired.** The ninety-one-name bug re-introduced (guard
reassigned inside the field loop): 92 names against 80, naming the twelve —
Body, Button, Camera, Caption, Card, CheckBox, Column, Input, Margin, Row,
Subtitle, TextArea. "only reflect has: nothing", because every one of those
children is supplied by some other parent.

---

## Item 2 · a reading at the width the names put the last pair at

`crowdWithin` was one literal per witness. Two of the four sentences name a
width — "Palette.Echo has two within four", "six edits apart" — and two are
claims about EVERY width: "no width above the floor keeps an answer to a pair",
"cannot produce a crowd at any distance". Those two had a literal that
*happened* to equal the farthest pair, with the reasoning in a comment.

Witnesses 0 and 1 now carry `crowdAtFarthest` and no number, and the reading is
taken at that run's own `farthestD`. A literal beside the flag is its own
failure: it would be read by nothing.

And the argument that lets one reading stand for all of them — past the
farthest pair every sibling is already inside the threshold, so nothing moves —
is an arm rather than a comment:

    crowd(farthestD)  ==  crowd(farthestD + Σ len(names))

The second width is wider than any two of these names can be apart, so the two
readings are over the same comparisons. It cannot fail while the crowd search
is monotone in its width, which is the point: that property is what every
"past here nothing changes" sentence in the file rests on.

**Break-tests — 2 run, 2 fired.** A `crowdWithin` typed beside the flag; a
crowd search given a ceiling — both saturation arms fired, and the message
names the finding as the search having stopped being monotone.

---

## Item 3 · the words come back beside the sentence

`foldReachedWords` recovered the kind words by splitting a recorded region
description on "/" and matching the longest trailing word. It worked — the
join is scripts "/"-joined, a space, kinds "/"-joined, so the seam piece reads
"Latin letters" and gives up "letters" — and it made the function a parser of
`foldRegionsNamed`'s output, two formatters with an undeclared contract
between them.

`foldRegionsNamed` returns `[]foldRegionName` now: the text, and the kind words
that went into it. Both come out of one `foldWordList`, so the sentence and the
column cannot be two different traversals of the same map. The record carries
both columns:

    {"U+2071..U+217C Common/Latin letters/modifier letters/numerals/symbols",
      []string{"letters", "modifier letters", "numerals", "symbols"}},

and the arm holds both with `slices.EqualFunc` — a run where only the kinds
moved is a naming that has started saying something else about the same range.
The astral U+1CCD7..U+1CCE9 entry has an empty column, which is the same fact
its sentence states: Go's 15.0.0 has no category for a single code point in it.

`foldRegionTexts` and `foldRegionLines` keep the four call sites and the
failure output honest. The reached/unreached answer is unchanged — letters,
modifier letters, numerals, symbols printed; **marks** and **punctuation**
still never have been.

**Break-test — 1 run, fired.** One kinds cell edited to `marks`: the arm fails
on a region whose text it still agrees with.

---

## Item 4 · thirteen reasons, and four of them are now claims

`foldUnwordedCategories` mapped a category to a sentence and the arm only ever
read the KEY. The three groups were a paragraph in a doc comment, so a
mis-filed entry, or a reason that had stopped being true, was invisible.

Each entry carries its group, and the group is **re-derived from the category's
own name** — P, Z, C, with LC the one entry that is not a category at all and
whose first letter says exactly what it is not. Then each group's reason
becomes an arm:

    punctuation shapes  Pc Ps Pe Pi Pf   Pd and Po must still share one word —
                                         that collision IS the reason
    separators          Zs Zl Zp         every code point unicode.IsSpace
    non-graphic         Cc Cf Cs Co      not one unicode.IsGraphic
    not a category      LC               Lu, Ll and Lt all worded above

Every code point rather than a sample: the claims are absolutes, and Co's
137,468 code points cost about ten milliseconds. The log line prints the
partition — 5 punctuation shapes, 3 separators, 4 non-graphic, 1 not a category
— so a heading somebody changed shows up as a count.

**Break-tests — 3 run, 3 fired.** Zs filed under non-graphic (two arms: the
derived group, and U+0020 as the counterexample); Pd reworded to "dashes"; Lt
removed from the worded list (two arms: the partition, and LC's excuse).

---

## Item 5 · the note names the other caller

The baseline said THAT another test had walked a band, not WHICH — and in this
package the interesting version of that finding is "the composition test is
taking its own census" and the uninteresting one is "a fixture warmed the memo".

`affordedBandCaller` walks 32 frames and attributes an asking to the OUTERMOST
`Test` function, because the immediate caller is always this file's own
plumbing; the fallback is file:line, for an asking with no Test frame above it.
The tally moves under the same lock as the totals, so a caller's share and the
total it is part of cannot disagree. Taken outside the lock, so a stack walk is
not something the parallel band walks wait for.

**Break-test — 1 run, fired.** A test that runs first and asks for three bands:

    …process totals are 3 and 6. Per caller:
    TestTmpSomebodyElseWantsABand (2 walk(s), 1 reuse(s)),
    TestTheAffordedWidthHoldsItsTwoRelations (1 walk(s), 5 reuse(s))

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

Three files +866 −204, one new package.

    internal/themeleaves/themeleaves.go   item 1 (new)
    internal/themehistory/main.go         item 1
    wasm/verify/themenearmiss_test.go     items 1, 2, 5
    wasm/verify/inkglyph_test.go          items 3, 4

`go test ./wasm/verify` is **1.88s against 1.96s**. `go run ./internal/themehistory`
reproduces the recorded histogram and the eighty names exactly. 7 break-tests
run, 7 fired.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

1. **(age 0 · value medium) The history table's sixteen revisions are still
   read by a mechanism no arm exercises.** The expansion is held against
   reflect at HEAD now, over a working tree. What that cannot see is the git
   half: `leavesAt` fetches a revision's files with `ls-tree` and `cat-file`
   and hands them to `themeleaves.Of` as a map, and nothing checks that the map
   it builds is the same set of files `InDir` would produce for the same tree.
   A filter that drifted — a `.go` suffix test, the `_test.go` exclusion that
   now exists in both places — would give the table a different population per
   revision while HEAD's arm stayed green. A test could compare
   `leavesAt(HEAD)` with `InDir(".../core")` when a repository is present and
   skip when it is not, which is the one place a skipping git test is honest.
2. **(age 0 · value low) `themeleaves` has no test of its own.** Its arm lives
   in wasm/verify and asks one question: does this package agree with reflect
   about core.Theme. The properties that make it a walker rather than that one
   answer — a self-holding struct terminating, an embedded field taking its
   type's name, a pointer field staying a leaf, two siblings of one type each
   descending — are asserted by nothing, and every one of them is three lines
   of source text in a map literal. The package is importable now, which is
   what makes that cheap.
3. **(age 0 · value low) `affordedBandCaller` names a frame this package's own
   layout supplies.** It takes the outermost function whose name starts with
   "Test", which is right for every caller today and is a convention rather
   than a fact: a band asked for from a `t.Run` closure, a helper that starts
   with Test, or a goroutine gets the file:line fallback silently. Nothing
   asserts the tally's keys are test names at all. The affordance test could
   check its own share is filed under its own name — `t.Name()` is right
   there — which would make the attribution a claim rather than a heuristic.
4. **(age 0 · value low) The kinds column is only asserted where node is.**
   `foldReachedWords` reads the recorded column on every machine, and the arm
   that holds that column against the derivation runs only when both Unicode
   versions match their records. On a machine with no node — or a newer one —
   the column is a literal nothing checks, and it is the half a reader cannot
   eyeball, because a kinds cell that disagrees with its own sentence looks
   like nothing. The words in a recorded sentence and the words in its column
   could be held against each other without any population at all, which is
   the parse this session deleted — worth having back as an ARM over the
   record rather than as the way the words are obtained.
5. **(age 1 · value low) `foldReachedWords` reports words, not categories.**
   It answers which of the seventeen words have been printed, and "marks" and
   "punctuation" being unreached is a claim about six categories (Mn Mc Me, Pd
   Po) collapsed into two words. A build whose NFKD reached Mn alone would
   flip "marks" to reached and say nothing about Mc and Me, which are the
   entries the mapping is actually a guess about.
6. **(age 2 · deliberate non-goal) A sound chain bound at k = 3.**
   `affordedTwoStepBands` exists only for two, and there is no reason to write
   the three-step version: bounding the last step over every chain needs a
   census of every population of size n−k, which is the family the direct
   measurement walks — and the direct measurement is now the cheaper of the two
   at every k this file takes. The composition is kept at k = 2 for the SHAPE
   it holds, not as a route to anything. Listed so it is visibly declined
   rather than quietly missing.
