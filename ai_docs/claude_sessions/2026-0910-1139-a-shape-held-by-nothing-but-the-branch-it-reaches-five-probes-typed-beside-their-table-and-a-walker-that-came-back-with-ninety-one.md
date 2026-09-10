# Session: a shape held by nothing but the branch it reaches, five probes typed beside their table, and a walker that came back with ninety-one

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-10 (follows "a-premise-the-stdlib-does-not-carry-four-names-this-file-owned-and-a-direction-the-struct-has-never-gone")

## Ask

"Do all items in the Next list." Six items, all age 0 except the declined one
— one medium, four low, one a non-goal. Two files edited, one package added.
The item that produced the most was the one that looked like bookkeeping:
moving a throwaway script into the repository found a bug in the method the
prose had been describing for a session.

| item | where | shape |
|---|---|---|
| 1 | themenearmiss_test.go | four fixtures held to a branch and not to a shape |
| 2 | themenearmiss_test.go | five probes typed beside the table they probe |
| 3 | themenearmiss_test.go | a process counter read as "this run" |
| 4 | inkglyph_test.go | seventeen of thirty-one, and no record of the fourteen |
| 5 | internal/themehistory | a measurement that only existed as prose |
| 6 | — | a sound chain bound at k = 3, declined again |

---

## Item 1 · the shape, in numbers, beside the sentence

`affordedEndingWitness` carried a `shape` string — "five siblings each one
edit from the other four", "one parent holding two leaves" — and nothing
re-derived it. The arms asserted which ending each set reaches and nothing
else, so a fixture that drifted into reaching the right ending by a different
route went on passing with the prose beside it describing a set that no longer
exists.

Each witness now carries the countable half:

    perParent   leaves under each parent, most first
    closestD    the closest sibling pair, and
    farthestD   the farthest — equal when the sentence says every pair is one
                distance apart, apart when it bounds from one side only
    crowd*      the reading the sentence names: within `crowdWithin` the most
                siblings any leaf has is `crowdSiblings`, at `crowdAt` —
                every leaf tied at the maximum, because "Palette.Echo has two
                within four" is a claim that Echo is the ONLY one

`affordedWitnessShapeOf` re-derives all of it, and is deliberately a **second
implementation** of the grouping and the crowd search `themeLeafSetOf` already
has. That is the objection the threshold test states one screen up and
recomputes its own readings for: the witness's numbers are a claim about the
FIXTURE, and reading them off the derivation would make every arm a comparison
of `themeLeafSetOf` with itself. One join is kept — both walks must agree on
`closestD` — so the separation cannot degrade into the two being handed
different lists.

**Break-test — 1 run, fired.** `Weight.Aae` → `Weight.Qqq`: perParent
unchanged at [5], **the branch arms stayed green**, and the two shape arms
fired (farthest 3 against a recorded 1; crowd 3 at four leaves against 4 at
five). That is exactly the drift the item described.

### And one witness where the branch does imply the shape

Ending 1 is "no width crowds these names at all", and a parent of three
ALWAYS crowds once the width reaches its farthest pair — so a three-leaf
`Duo.` cannot reach ending 1 at all. The first break-test attempted was
exactly that edit and it moved the set to branch 2. The perParent arm is worth
having for the other three; for this one it is a second reading of something
the classification already forces, and the note says so.

---

## Item 2 · five probes that named a parent nothing had to have

`themeNearMiss(sparse, "Palette.Alfa")` and its four siblings were literals
typed into the test while the populations they probe come from
`affordedEndingWitness`. `themeNearMiss` compares within a parent only, so a
witness renamed out of `Palette.` left every probe naming a parent no set has
— and the arms would have reported "themeNearMiss found nothing" and meant
"this probe is not about this set".

`affordedWitnessProbe(t, e, leaf)` builds the path under the witness's own
parent and returns the distance to the nearest leaf and which leaf that is. It
is Fatal on the two structural failures (a witness spanning two parents; a
probe that names a leaf the set has), because either makes every arm
downstream a reading of nothing.

The distance comes back because **every arm had a premise about it and none of
them stated it**:

    Palette.Alfa    2 edits, inside sparse's 3 and outside core.Theme's 1 —
                    both halves, since a probe inside core.Theme's threshold
                    would be found either way and prove nothing
    Palette.Zulu    4, outside 3
    Ramp.Zzzzzz     6, outside 3
    Duo.Mike        4, outside 3
    Weight.Zzz      3, outside 1

**Break-tests — 2 run, 1 fired, and both are the finding.** Renaming the
sparse witness's parent `Palette.` → `Hue.` is now GREEN, which is the fix:
before, the probe would have named a parent no set had. Widening the crowded
witness to `Weight.Zz*` so `Zzz` becomes a hit fires the premise arm — and
that arm's assertion (`!Contains(crowdedMiss, "themeNearMissReach")`) had been
passing vacuously, because a found message contains no such sentence.

---

## Item 3 · a counter for the process, read into a sentence about the run

`affordedBandMemo.walks`/`reused` accumulate for the life of the test binary
and were read once, by the affordance test's log line. The numbers happened to
be that test's own — because it is the only caller of the band walks in the
package, which nobody had asserted and which the next test to want a band
would quietly end.

`affordedBandMemoRead` takes both under the lock (a walk finishing between two
separate reads would make the sentence describe a state the memo was never
in). `affordedBandMemoSince` returns the delta from a baseline the test takes
at its top, plus a note when the baseline is not zero.

Not asserted to be zero: a second caller is a perfectly good thing to be, and
a rule about what may be written next door is the wrong shape for this. The
log line states it instead.

**Break-test — 1 run, fired.** A test that runs first and asks for two bands:
this test's share reads 2 walks and 4 reuses against process totals of 3 and 5,
where the old line would have printed 3 and 5 as "this run".

---

## Item 4 · the other fourteen, written down

`foldRegionKinds` words 17 categories and Go's tables carry 31. The
fall-through printed the two-letter code for anything unmapped, which is
honest and which made two very different things identical in the output: a
category deliberately left as its code, and one nobody had thought about.

`foldUnwordedCategories` is the complement, 13 entries with a reason each, in
three groups:

    Pc Ps Pe Pi Pf   "punctuation" is what Pd and Po get, and a region of
                     Ps/Pe is brackets — a different sentence. No one-word
                     English for "initial quotation mark".
    Zs Zl Zp         a separator is not a kind of glyph
    Cc Cf Cs Co LC   no appearance to describe; and LC is not a category but
                     Go's union of Lu/Ll/Lt, all three worded above, so it is
                     unreachable rather than undescribed

Cn is in neither on purpose, and the arm asserts that too: unassigned is the
absence of a kind, and `foldRegionsNamed` reports it a level up as two Unicode
versions disagreeing. 17 + 13 + Cn = 31.

`TestTheRegionKindsAccountForEveryCategoryGoHas` holds the partition against
`unicode.Categories` in both directions — a category Go has that neither list
mentions, a name in both, a name Go does not have.

**Break-tests — 2 run, 2 fired.** `Cf` removed from the record; `Zs` added to
both lists.

### And which words nothing has ever printed

`foldReachedWords` reads the recorded region descriptions and reports that of
the six distinct words, four are printed and **`marks` and `punctuation` never
have been**. Those are words written for categories these two populations have
not reached — a guess about what a region of them would be called rather than
a translation of one that turned up. Not wrong, and now visible, which is the
difference between a mapping and a list of what happened to be there.

---

## Item 5 · the walker, and the ninety-one

`internal/themehistory` is the AST walker that produced the git-history table,
in the repository rather than in a scratch directory — so `go vet`, `go build`
and `go test ./...` cover it and the next reading is a command:

    go run ./internal/themehistory
    go run ./internal/themehistory -names

It reproduces the recorded histogram exactly (1:8, 2:2, 3:2, 4:1, 5:1, 28:1,
plus the 25-leaf bootstrap; 16 of 16 commits added leaves and none removed
any), and prints HEAD's population so a reader can check it against
`affordedMeasuredOn`.

**And writing it found a bug in the method the prose had been describing.**
The first version came back with **91 leaf names, not 80**: the recursion
guard was reassigned inside the field loop, so it descended into
`Typography.Body`, marked `TextStyle` seen, and then treated `Caption` and
`Subtitle` — the same type — as leaves. Eleven names too many, and the same
mistake in the original throwaway would have moved every row of the table.

Which is the argument for the item: the reading existed only as prose and a
number, and a number in a note is a number that has already moved.

The check the tool documents is `-names` and not the count: two different sets
of eighty both print `80`. Name-for-name identical to `affordedLeafNames`
today, verified by diff.

Why it is not a test: a test that shells out to git fails in a shallow clone,
a source tarball or a build container. It cannot become an arm; it can stop
being a re-derivation.

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

Two files +702 −12, one new package (354 lines).

    wasm/verify/themenearmiss_test.go  items 1, 2, 3, and item 5's prose
    wasm/verify/inkglyph_test.go       item 4
    internal/themehistory/main.go      item 5

`go test ./wasm/verify` is **1.75s against 1.80s**. 6 break-tests run; 5 fired,
and the sixth not firing is the fix for item 2 working.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

1. **(age 0 · value medium) The history walker's population is checked by a
   reader and not by anything.** `internal/themehistory` and
   `affordedLeafNames` expand core.Theme by two different mechanisms —
   syntactic and reflective — and the whole worth of the table rests on them
   producing the same eighty names. That was verified once, by hand, with a
   diff. The walker cannot be a test because it needs git, but its EXPANSION
   does not: `leavesAt` given a working tree rather than a revision is a pure
   parse of `core/`, and a test in the wasm/verify package could hold that
   against `affordedLeafNames()` on every run. The git half stays a command;
   the half that can be an arm is the half that already went wrong once.
2. **(age 0 · value low) The witness crowd readings are one width each, and
   two of the four shapes are claims about every width.** `crowdWithin` is a
   single distance per witness, chosen because the shape sentence names it.
   For witness 1 the sentence is "cannot produce a crowd at ANY distance" and
   the reading is taken at the farthest pair — which is sound, because past
   the farthest pair nothing moves, but the reasoning is in a comment rather
   than in the code. A reading taken at `farthestD` derived from the names,
   rather than at a recorded literal that happens to equal it, would carry its
   own argument.
3. **(age 0 · value low) `foldReachedWords` parses the naming back out of the
   strings that produced it.** It splits a recorded region description on "/"
   and matches the longest trailing word, which works because
   `foldRegionsNamed` joins scripts and kinds with a space and each half with
   "/". Two formatters, one of them a parser of the other's output, held
   together by nothing but both being in this file. The alternative is for
   `foldRegionsNamed` to return the word set alongside the string, which is a
   signature change touching three call sites and the record's shape.
4. **(age 0 · value low) The unworded categories' reasons are prose nothing
   reads.** `foldUnwordedCategories` maps a category to a sentence, and the
   arm only ever looks at the KEY. A reason that became wrong — "no
   appearance to describe" beside a category that acquired one — would sit
   there indefinitely. The three groups in the doc comment are the real
   structure and they are not in the data: a `group` field with three values
   would at least let the log line say "five punctuation shapes, three
   separators, five non-graphic" and make a mis-grouped entry visible.
5. **(age 0 · value low) The band memo's baseline is taken in one test and the
   note is only printed there.** Any other test that starts calling
   `affordedKLeafBand` gets no such sentence, and the affordance test's note
   will say a caller exists without saying which. `runtime.Caller` at the walk
   site would name it, at the cost of a string per miss — or the memo could
   count per-caller and the log line list them.
6. **(age 1 · deliberate non-goal) A sound chain bound at k = 3.**
   `affordedTwoStepBands` exists only for two, and there is no reason to write
   the three-step version: bounding the last step over every chain needs a
   census of every population of size n−k, which is the family the direct
   measurement walks — and the direct measurement is now the cheaper of the two
   at every k this file takes. The composition is kept at k = 2 for the SHAPE
   it holds, not as a route to anything. Listed so it is visibly declined
   rather than quietly missing.
