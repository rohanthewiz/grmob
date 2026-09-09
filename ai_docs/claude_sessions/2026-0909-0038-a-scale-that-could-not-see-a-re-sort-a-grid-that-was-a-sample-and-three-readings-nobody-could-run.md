# Session: a scale that could not see a re-sort, a grid that was a sample, and three readings nobody could run

Session: https://claude.ai/code/session_01Txqr9n7cviHkU6RQz5gVJq
Date: 2026-09-09 (follows "two-edges-for-a-third-a-copy-in-more-than-one-construct-and-a-probe-that-earned-its-round-trips")

## Ask

"Do all items in the Next list", then "a second pass to pick up any new Next
list items and do those". Six items closed in pass one; the five follow-ons
pass one raised, closed in pass two. Six files, one of them new.

| pass | item | where | shape |
|---|---|---|---|
| 1 | 1 | themenearmiss_test.go | one ratio for four endings |
| 1 | 2 | pinfixture_test.go | a reachability nobody walked |
| 1 | 3+5 | browser.mjs | a skip nobody counted, a read nobody bounded |
| 1 | 4 | inkglyph_test.go | a census with no build behind it |
| 1 | 6 | checknumbering_test.go | a sort with no population under it |
| 2 | A | themenearmiss_test.go | the residual, reported and never asserted |
| 2 | B | pinfixture_test.go | 13 pairs of 20, called a grid |
| 2 | C+D | inkcanary_test.go (new) | readings only a browser could run |
| 2 | E | checknumbering_test.go | a sort no run had exercised |

---

## Item 1 · where the sets went

`affordedScale` is one ratio over four endings. That is right when the walk
changes and wrong for the change that threatens this census: a rewrite of
`themeLeafSetOf` re-sorts which ending a set reaches, the population is
identical, the scale is 1.00, and "this ending should be where it was" is the
constant argument in a measurement's clothes.

The residual is what separates them, and it is available because both censuses
are partitions of one walk:

    affordedRecordTotal   the record's four counts against affordedSetCount of
                          the population it says it was taken over. Equal by
                          construction, and a mistyped re-measure is invisible
                          from the record because the wrong number still looks
                          like a count
    census == len(sets)   the same on this run — the switch has a default arm
    affordedTakers        endings ABOVE their own prediction, largest first.
                          The four residuals sum to zero, so a shortfall has a
                          partner and the message can name it

`affordedShortfallCause` now decides among three findings instead of two. The
third — an arm gone with no taker — is unreachable while the partition holds,
and is spelled out rather than left as a fallthrough.

    50 of the 930 sets ended with "the ceiling cost this set a wider
    threshold", against a floor of 125 … The population is the one the record
    was taken over — same leaves, same window, same 930 sets — so this is not
    the walk at all: this ending predicts about 377 and 50 arrived, and "the
    ceiling and the crowding stop in the same place" is at 380 against a
    prediction of 53, +327. That is themeLeafSetOf reshaping WHICH ending a set
    reaches rather than an arm going.

**Break-tests — 4 run, 3 fired.** A record mistyped by one: the partition arm.
A fifth ending: 925 of 930 accounted for. A shortfall with a taker: the message
above. And a re-sort that stayed above every floor: **passed** — which is the
finding, and is item A below.

---

## Item 2 · which transitions can straddle at all

`pinBlankedAs` reads every construct in a span and both fixtures opened with
`//`. The note said why: two transitions end at a newline, "everything else
needs a delimiter in the raw file and a delimiter breaks the match".

The second half is wrong. A delimiter breaks the match when the phrase is
spelled without it, and a citation that quotes a call carries its own quotes:

    log(pinSame("x"))    code the lexer kept, then a string literal, and not a
                         newline anywhere in it

So the rule is about the bytes at the boundary, not about the constructs on
either side: **whitespace the strip removes, or bytes the citation spells.**
`TestWhichConstructTransitionsAStraddleCanBeBuiltFrom` walks it, asserting both
directions per row — the kinds run the copy actually meets, and the same source
declining the same phrase with the delimiter taken out.

Two fixtures added to the sentence test with heads that are not `//`: a runaway
string (which needed a per-row `open` allowance, since that construct is
declared by the lexer's unterminated-string report) and a block comment closing
into code and then a literal, with no newline in it.

**Break-tests — 3 run, every one fired.** `pinBlankedAs` returning the first
kind only: all 13 rows. A `bare` that matches after all: the strip arm. Dropping
the runaway from a row's kinds: the record arm *and* the coverage arm.

---

## Items 3 and 4 · a skip nobody counted, a read nobody bounded

`inkCanaryAgreement` passed over a probe whose `faces` had two entries — the
population where a name-versus-metric disagreement is most likely — without a
count. Now `multi`, `unread` and `unjoined` are separate: a comparison
declining on a page state, a read that failed, and what the first cost the join
in rows. The clause is silent when all three are zero, because a clause that
reports zeroes on every healthy run is one nobody reads.

And the bound item 5 asked for. `inkCanaryReqAxes` measures which axes of the
request key the probes actually differ in, which is the only thing that decides
whether a second read could answer differently from the first:

    read at 6 of the 6 distinct requests … answering with one face at every one
    of them, which is this stack having no optical cut across weight (700, 400)
    and size (13px, 19px, 12px), which is 6 points of the request grid and the
    bound on what reading more of them could show — and at one style normal
    throughout, so nothing here is a reading about style

Six reads, six grid points: nothing is being spent twice on this theme, which
is not what last session's item guessed. The blind half is style, and it is now
named.

---

## Item 4 · what the census is a fact about

The 15054 was NFKD as one node's ICU implements it, against a table somebody
wrote down, with nothing recording which node. `foldMeasuredOn` records Unicode
(the version of the DATA, not node's), ICU and node, taken from
`process.versions` in the same process that did the folding.

    the build moved    this run's Unicode is not the record's. Every count is
                       expected to differ; the two edges still hold; re-take.
    the table moved    same Unicode, different numbers. NFKD cannot have
                       changed, so it is gen.go's fold — and a fold that
                       quietly narrows passes both edges, because narrower is
                       the direction they allow.

Also closed a hole the item did not name: the whole witness paragraph ran
inside `if bearingExample >= 0`, so a census finding no seed-bearing character
skipped it in silence and printed a rune nobody found. `bearing == 0` is now a
failure.

Recorded on Unicode 16.0 / ICU 76.1 / node 22.12.0: 15802, 748, 15054, 299.

**Break-tests — 2 run, both fired.** `inkGlyphFold` dropping its lowercase: the
ASCII edge, the witness arm, *and* three census rows. `bearing` stale by 113:
the census arm alone, with the build matching.

---

## Item 6 · what the proximity sort decided

`citationLendersNear` puts same-directory rows first. `citationLenderSpread`
counts what it had to decide — rows, near, directories, and how many kinds in
the whole table have rows in more than one directory — and the line says which
of four states it is in.

The measurement immediately contradicted the guess in last session's item, and
in the more interesting direction:

    nearest this file first (none of the 2 in this file's own directory, across
    2 of them, so every row is equally far and the order below is alphabetical
    — 1 of the 1 kind citationExempt names has rows in more than one directory)

The example is in neither lender's directory. The sort decides nothing here.

---

## Pass two

### A · the residual, asserted

Item 1's break-test that passed. Every floor is one ending at a time, so 40
sets moving between two endings clears all four and the census reads healthy.

On an unmoved population the comparison needs no tolerance at all: same leaves,
same window, same walk, and which ending a set reaches is a function of the
set — so every count is the recorded one, exactly. Asked only in that state;
when the population has moved, how far each ending should move is a question
about which windows crowd, which the scale does not model, and that reading
stays in the log line under `affordedResidualQuiet`.

**Break-test.** 40 sets re-sorted, both floors clear: two failures naming both
sides of the move, against the record.

### B · the grid was a sample

Thirteen ordered pairs of five constructs, called "the grid". Twenty exist.
The seven missing ones are all reachable and all reachable the same way, so
they are now rows — and the pairs are *enumerated* rather than listed, so a
pair with no row is named.

Plus the third lexer. Every row ran one of the two non-nesting paths;
`pinCodeOnly` branches three ways. The Swift row is written so the two paths
disagree about it — with nesting the first `*/` closes the inner comment and
the literal after it is still inside the outer one; without, that literal is a
construct the copy passes through. A row both paths answered alike would have
been a Swift row in name only, and the first one I wrote was exactly that.

**Break-tests — 3 run, every one fired**, including Swift's nesting turned off.

### C and D · readings only a browser could run

`multi`, `unread`, `unjoined` and the axis census are all dead on this
machine's Chrome: every probe names one face, and the grid varies two axes.
Their only evidence was a browser behaving in a way nobody can schedule.

`inkcanary_test.go` lifts the real declarations out of browser.mjs — the
inkglyph_test.go idiom — and runs them under node over five request grids and
four probe answers this browser does not produce: a probe naming two faces, a
probe naming none, a style spelled with a space, and a name agreeing over
advances 12px apart.

And the join item D wanted. The parse takes size and weight off the END of the
key because `fontStyle` can be two words, and nothing held it to
`INK_CANARY_REQ_JS` — the same "a join on two spellings of a key" failure that
expression was extracted to prevent, arriving in the reader. The last fixture
builds keys with `eval(INK_CANARY_REQ_JS)` and asks the parse for the three
fields back.

**Break-tests — 2 run, both fired.** Splitting on the first space:
`"oblique 10deg 400 19px"` parses as style `oblique`, weight `10deg`, size
`400` — a grid of 8 where the fixture is 2. Dropping `unjoined++`: two rows.

### E · a sort no run had exercised

The near group has never been non-empty on this repository, so
`citationLendersNear`'s own arm had never run.
`TestTheLenderOrderIsAskedOfOrderingsThisRepositoryDoesNotHave` builds four
orderings, asserts the order, asserts the output is a permutation of the input
apart from the order (a dropped row satisfies any short-enough `want`), and
asserts the clause. Plus the kind census over three tables.

**Break-tests — 2 run, both fired**: the groups swapped, and the far group
dropped.

---

## Verification

Ten paths, green, after both passes:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      replay + mjs suites + browser pass (rc 0)
    android ./gradlew compileDebugKotlin --offline
    android ./gradlew :app:fetchComposeLayoutSources --offline
    GOOS=js GOARCH=wasm go build ./...

Six files, +1432 −44 plus one new:

    wasm/verify/themenearmiss_test.go       items 1, A
    internal/pinfixture/pinfixture_test.go  items 2, B
    wasm/verify/browser.mjs                 items 3, 5
    wasm/verify/inkglyph_test.go            item 4
    wasm/verify/checknumbering_test.go      items 6, E
    wasm/verify/inkcanary_test.go   (new)   items C, D

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

1. **(age 0 · value medium) The exact census is asserted only where the
   population is unchanged, which is the state a re-measure leaves.** The arm
   that catches a re-sort compares each ending against the record exactly, and
   it runs only when `leaves` and `window` both match `affordedMeasuredOn`. The
   moment core.Theme gains a leaf, that arm goes silent and the only thing left
   is the per-ending floors — which is the state item A was written because
   they cannot see a re-sort. So the strongest reading here is the one that
   stops working exactly when somebody is editing the struct, and nothing says
   so: the log line reports the residual under `affordedResidualQuiet` and a
   green run at 40% off prediction prints a sentence.
2. **(age 0 · value medium) `affordedResidualQuiet` is a tenth nobody
   measured.** Its note argues a leaf added mid-name shifts which windows crowd
   "by a percent or two" and that a tenth is well above that — and neither
   number is in hand. The measurement is cheap and nobody has taken it: walk
   the same census at 79 and 81 leaves, take the largest residual each way, and
   the band is a reading rather than a taste. That is the argument
   INK_OWN_SHAPE_FLOOR's quarter makes about itself, unmade here.
3. **(age 0 · value low) The transition grid asserts 20 pairs through one
   language and a half.** Every pair has a row and 20 of the 22 rows are `.go`;
   Swift gets the nested comment and one line comment, and `.mjs` gets one. The
   completeness arm counts PAIRS, so a lexer path could lose a construct
   entirely — Swift's `"""` multi-line string, which `pinFreeForm` names and no
   row here reaches — and the grid would still report itself complete. The
   census is complete in the dimension it counts and nothing says what the
   other dimension is.
4. **(age 0 · value low) `foldMeasuredOn` asserts nothing on a machine that is
   not this one.** The census bracket runs only where the Unicode version
   matches the record, so a developer on a different node — or CI on a rolling
   one — gets the two edges and no census reading at all, which is the state
   the record was written to replace. The honest shape is probably a bracket
   that survives a build change (the gap as a FRACTION of what the note's fold
   changes moves far less than the counts do), and nothing here knows whether
   that fraction is stable, which makes it a measurement rather than an edit.
5. **(age 0 · value low) The lifted canary test asserts the readings and not
   the wiring.** `inkcanary_test.go` runs `inkCanaryAgreement` and
   `inkCanaryReqAxes` over populations this browser does not produce, and the
   browser pass runs the wiring over the population it does. Neither asks
   whether `asked.canvasGenericMulti` is the number that reading returned —
   the assignments at the join site are three lines nothing reads, and a
   transposed pair there would put `unread`'s count in `multi`'s clause with
   both tests green.
