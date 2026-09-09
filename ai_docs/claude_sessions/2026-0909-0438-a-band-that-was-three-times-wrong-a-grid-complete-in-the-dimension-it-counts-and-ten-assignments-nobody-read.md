# Session: a band that was three times wrong, a grid complete in the dimension it counts, and ten assignments nobody read

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-09 (follows "a-scale-that-could-not-see-a-re-sort-a-grid-that-was-a-sample-and-three-readings-nobody-could-run")

## Ask

"Do all items in the Next list." Five items, all age 0 — two medium, three
low. Four files, no new ones.

| item | where | shape |
|---|---|---|
| 1 | themenearmiss_test.go | the strongest reading, silent while somebody edits |
| 2 | themenearmiss_test.go | a tenth nobody measured, and it was wrong |
| 3 | pinfixture_test.go | complete in pairs, blind to routes |
| 4 | inkglyph_test.go | a census that stops at the edge of this machine |
| 5 | inkcanary_test.go | the readings asserted, the wiring not |

Items 1 and 2 came out as one piece of work: the measurement item 2 asked for
is exactly what licenses item 1's new arm to be an assertion rather than a
taste.

---

## Item 2 · the tenth, measured — and wrong in the direction that matters

`affordedResidualQuiet` was a tenth on the argument that a leaf added mid-name
shifts which windows crowd "by a percent or two", and that a tenth was well
above that. Neither number was in hand. `affordedOneLeafBand` walks every
population core.Theme's names are a single leaf away from — each name dropped
in turn, read in both directions, no synthetic name anywhere — and takes the
largest residual per ending:

    no width crowds these names at all             0.02%   (473 sets)
    the ceiling cost this set a wider threshold      2.9%   (377 sets)
    the ceiling and the crowding stop in the same   27.6%   ( 53 sets)
    its own crowding stopped the search             33.3%   ( 27 sets)

A tenth is three times the drift of the two large endings and a third of the
drift of the two small ones. On an honest one-leaf edit the log line would
have called a standing-still walk re-sorted — the reading failing at exactly
the moment somebody is editing the struct, which is the moment it is read.

The shape is not a surprise once it is in front of you: a one-leaf change moves
a handful of SETS between endings, and a handful is a third of 27 and nothing
at all of 473. That is the per-ending-floor complaint arriving one level up for
the third time in the file.

So the band is per ending, re-derived on every run over eighty populations, and
recorded in `affordedBandMeasuredOn` — which is also the wider of the two
readings on `themeLeafSetOf` here: the census is one walk over these names and
the band is eighty walks over eighty populations, so a re-sort that happened to
leave the census where it was still has all of those to get past.
`affordedResidualQuiet` survives as the fallback for an ending the record does
not name, and the arm that uses it says so rather than printing it like a
measurement.

### A hazard the band found on the way in

The log line computed the residual inline while the band computed it in
`affordedResidualOf`, and on arm64 the two disagreed in the last bit — Go may
fuse a multiply and the subtract after it into one rounding, and it did at one
spelling and not the other. Against a band that is the MAXIMUM of a family this
number is a member of, one ulp is the whole distance between "inside its band"
and a green run reporting a re-sort that did not happen. It showed up as a
0.02%-band ending reported 0.0% out. Both now go through the one function, and
the note at the call site says why it is not tidiness.

---

## Item 1 · the arm that is silent exactly when somebody is editing

The exact comparison runs only while `leaves` and `window` both match the
record. That is the state a re-measure leaves and not the state an edit leaves,
so the strongest reading in the file goes quiet the moment core.Theme gains a
leaf — leaving four floors that cannot see a re-sort by construction.

**A leaf ADDED is now asserted, and the band's construction is why it needs no
tolerance.** The band drops each of THIS run's names in turn, so when the run
has one leaf more than the record, the record's own population is a member of
that family — this run's names minus the added leaf. Its residual is therefore
one of the numbers the band is the maximum of. Not "about the same size as":
one of them. A residual over the band is arithmetic that cannot happen while
`themeLeafSetOf` sorts the walk the way it did when the record was taken.

**A leaf REMOVED is deliberately not asserted.** A run one leaf short cannot
walk the record's population at all — the removed name is not there to put back
— so the band measured is the 78↔79 step standing in for the 79↔80 step in
force, and a bound whose family does not contain the case it brackets is a
tolerance wearing a proof's clothes. That reading stays in the log line.

And the log line now says which of the three readings is in force, because they
are not equally strong and only one of them runs on any given population:

    the census against the record within the band a single ADDED leaf is worth
    to each ending, asserted — the record's own population is this run's names
    with the new leaf dropped, so its residual is a member of the family that
    band is the largest of

with the `distance == -1` and `|distance| > 1` arms saying, in as many words,
that an ending outside the band is then the absence of a finding rather than
one.

**Break-tests — 4 run, 3 fired.** A band record drifted by 0.003: the band arm.
A reworded ending: both key-set arms. A record one leaf behind with 40 sets
re-sorted: two failures naming both endings. And a record one leaf behind with
an HONEST gain: **passed** — which is the family argument holding empirically,
and is why that arm can be an assertion at all.

---

## Item 3 · complete in the dimension it counts

The pair census walks 20 ordered pairs and holds itself to a row for each. The
completeness arm counts PAIRS, so a lexer route could vanish and the grid would
go on reporting itself complete.

`pinLexBranches` is the other dimension: the eleven places `pinCodeOnly`
decides a construct. **Nine of the eleven produce, or decline to produce, `a
string literal`** — so none of them adds a pair and the census above cannot see
any of them go. The grid reached five. Six rows added:

    '        a single-quoted literal (.go)
    `        a template literal (.mjs)
    '        an apostrophe that is CODE (.swift), which .go reads as a quote
             opening a literal that runs to the newline
    `        a backtick that is CODE (.swift), which .mjs reads as a template
             literal swallowing the rest of the file
    """      a literal that carries past the newline instead of running away
    \        an escape carrying the quote after it

Coverage is read off what the lexer recorded, never off a label: a `via` column
would be a row asserting its own coverage. The four flag branches ask the
stronger question — whether the OTHER path records something different about
the same source, which is the only evidence a flag did anything.

One correction the probes made to the file's prose: `pinCodeOnly` calls the
three-quote run "Swift's multi-line string" and the branch has no extension
guard on it. A `"""` in a `.go` harness is lexed as a literal that carries past
newlines, on every path.

**Break-tests — 3 run, every one fired.** A row deleted: the branch named.
`single := false` in the lexer: two branches, both directions of the flag. And
the first break-test caught the recital claiming "all 11 branches are reached"
while printing 0 rows for one — now a count, not a claim.

---

## Item 4 · a census that stops at the edge of this machine

`foldMeasuredOn`'s bracket runs only where this run's Unicode version matches
the record's. Right for what it holds — NFKD is the ICU's data — but it leaves
a developer on a different node, or on no node at all (where the whole test
skips), with the two edges and no census reading whatsoever. The gate closes
exactly where somebody is most likely to be reading it.

The item's guess was a bracket that survives as a FRACTION. That is a claim
about two ICU versions and there is one on this machine — and it turns out not
to be needed, because the census's gen.go half can be taken with no ICU in it
at all. `inkGlyphFold` is three steps and every code point it changes is
changed by exactly one of them:

    ignorable   24    inkGlyphIgnorable drops it — gen.go's own predicate
    table        8    inkLigatureForms rewrites it — gen.go's own map
    lowered   1173    strings.ToLower, and nothing else did — Go's tables
    ————————————————
    changed   1205    asserted as a partition; a fourth cause is a step
                      somebody added and the three counts are then shares of
                      a population that is not the fold's width

The first two are held on any machine that can build the package. The third is
gated on `unicode.Version` the way the other census is gated on node's, which
is the same reading about a different vendor's copy of the same data — and
without it a table edit and a toolchain upgrade would arrive at the total as
the same number. `table` is also read off `len(inkLigatureForms)`, so a row
that maps a character to itself is a row in the table and not in the fold.

`foldBuildNote` now tells an off-build reader which test still has the answer.

**Break-tests — 3 run, all fired.** The ignorable predicate narrowed by one
code point: 23 against 24. A table row mapping to itself: the record arm AND
the map-length arm, which is the one that says which of the two it is. And a
record taken on another Go Unicode version: the lowercase arm skipped, the two
gen.go counts still asserted, the note saying so.

---

## Item 5 · the readings asserted, the wiring not

Ten assignments carry the canary readings into `asked`, three of them off one
returned object, and nothing read them. A transposed pair costs nothing anybody
can see: the clause reads "N of the probes naming more than one face" over the
count of the probes that came back with none, and both tests stay green — one
never looks at `asked` and the other has never had a nonzero to put in it.

The block is lifted out of browser.mjs by the same regex move the declarations
use, compiled with `new Function` so its calls bind to parameters rather than
to the real declarations, and handed stubs whose every field is a different
number. What comes back is `asked` itself, and each field has to hold the
sentinel belonging to the reading it is named after — a permutation check, not
a type check, and the message names which reading's number landed where.

The guard is asked the same way: the five inside `if (facesHeld)` have to be
ABSENT on a run with no faces, not zero — the difference between a clause that
stays silent and one that recites a measurement nobody took. The two outside it
(the spread, the axes) have to survive, because they are the reading that is
still there on a run with no comparison in it.

**Break-tests — 3 run, 2 fired, 1 taught something.** `multi` and `unread`
transposed: two failures naming both halves. The block's tail rewritten with
braces: the lift's Fatal. And a comment appended to the block's first line:
**passed**, because the regex spans it — which is the anchoring behaving as
intended rather than a hole.

---

## Verification

Ten paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      replay + mjs suites + browser pass (rc 0)
    android ./gradlew compileDebugKotlin --offline
    android ./gradlew :app:fetchComposeLayoutSources --offline
    GOOS=js GOARCH=wasm go build ./...

Four files, +1175 −50. browser.mjs is unchanged — item 5 reads it rather than
editing it.

    wasm/verify/themenearmiss_test.go       items 1, 2
    internal/pinfixture/pinfixture_test.go  item 3
    wasm/verify/inkglyph_test.go            item 4
    wasm/verify/inkcanary_test.go           item 5

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

1. **(age 0 · value medium) The band census names eleven branches and nothing
   holds that list to the lexer.** `pinLexBranches` is the fix for a census
   complete in the dimension it counts — and it is itself a hand-written list
   of eleven, checked for coverage and not for completeness. A twelfth branch
   in `pinCodeOnly` (a fourth extension, a new delimiter, a raw-string prefix)
   leaves the census reporting 11 of 11 and the grid reporting 20 of 20, with
   the new route reached by nothing. That is the same complaint one level up
   for the fourth time in that file, and the honest shape is probably a count
   read off the lexer rather than a list written beside it — the switch has an
   arm per construct and the flags are three booleans off one string, so the
   branch set is derivable in principle and nobody has tried.
2. **(age 0 · value medium) `inkCanaryFaceSpread` is the one canary reading
   nothing asks.** The wiring test stubs it, and that is its only appearance in
   any test: `inkCanaryAgreement` and `inkCanaryReqAxes` are both lifted and
   run over populations this browser does not produce, and the third reading —
   whose `mounted`, `read` and `answers` are three of the ten fields the tail
   prints — has never been asked a question. It is the reading that says what
   asking per request bought, which is the sentence a reader uses to decide
   whether the six reads were worth their round trips.
3. **(age 0 · value medium) A leaf REMOVED still has no assertion, and the
   reason is a family that cannot be enumerated.** The added-leaf arm is a
   proof because the record's population is one of the populations the band
   walks. Removing a leaf inverts that: the record's population is this run's
   names PLUS one unknown name, which is an unbounded family, so the band in
   force is the neighbouring step and the log line says so. The gap is real and
   the shape that would close it is not obvious — the honest options are
   recording the leaf NAMES alongside the counts (so a removal can be walked
   from the record rather than from the run) or accepting that removals are
   rare enough to leave as a reading. Neither has been argued.
4. **(age 0 · value low) A move of more than one leaf has no band at all.** The
   log line says which reading is in force and, past ±1, says an ending outside
   the band is the absence of a finding rather than one — which is honest and
   is not a measurement. Whether the drift grows about linearly in the number
   of leaves moved is checkable (drop k names for k = 2..5 and take the worst
   each way), and if it does, `band × k` is a bound with evidence behind it
   instead of a sentence explaining why there is none. The cost is the choice
   of WHICH k names, which is the "cases somebody thought of" problem the
   one-leaf family avoids by enumerating.
5. **(age 0 · value low) Both fold censuses walk the BMP and neither says so as
   a bound.** `foldOwnMeasuredOn` and `foldMeasuredOn` stop at U+FFFF because
   that is the plane browser.mjs's lifted loop walks, and the surrogates are
   skipped on both sides for a stated reason. Nothing anywhere says what an
   astral code point would do to either fold — `inkLigatureForms` could grow a
   key above the BMP and the new `table != len(inkLigatureForms)` arm would
   report it as a row that changes nothing, which is a true sentence about the
   wrong cause.
6. **(age 0 · value low) The FMA hazard is fixed at one site and prevented
   nowhere.** The residual now has one definition and a note saying why, and
   nothing stops a third spelling appearing next to a band comparison — the
   failure mode is silent, arrives as a green run's sentence rather than a
   failure, and took a break-test's odd output to notice. Whether any other
   comparison in these files is a float derived two ways has not been looked
   for.
