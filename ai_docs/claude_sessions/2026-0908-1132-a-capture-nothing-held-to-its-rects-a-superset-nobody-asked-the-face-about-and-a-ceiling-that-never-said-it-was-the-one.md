# Session: a capture nothing held to its rects, a superset nobody asked the face about, and a ceiling that never said it was the one

Session: https://claude.ai/code/session_01MQcEV7mAeEV4h1LywCdQhe
Date: 2026-09-08 (follows "a-tree-two-readers-disagreed-about-a-claim-about-the-words-and-a-threshold-that-travelled")

## Ask

"Let's continue working the Next list items" — all seven, every one raised last
session, worked in order.

| # | age·value | item |
|---|---|---|
| 1 | 1·medium | The fingerprint joins the two readers on identity and nothing joins them on LAYOUT |
| 2 | 1·low | `inkGlyphPerCharacter` refuses a superset and the superset is not measured |
| 3 | 1·low | `inkDeclarationGuard` pairs reads by a name suffix |
| 4 | 1·low | The tail recites `targets` and nothing joins it to the three the census counts |
| 5 | 1·low | The blind-spot note names the lines and does not say whether the phrase is on one |
| 6 | 1·low | `citationExemptInputs` is tested and `citingFiles` is what fills both its arguments |
| 7 | 1·low | `themeNearMissReach` is the one judgement left and no real struct is near it |

## Item 1 — the other side of the window the tree fingerprint opened

Last session's fingerprint joins the protocol's face reads to the page the
rects came from, and says why it leaves geometry out: a scroll or a resize
moves every rect and changes no fact about which element is which, so folding
layout into that hash would turn a benign relayout into a failure about fonts.

The gap that leaves is on the other side of the same window. Between the
evaluate that returns the rects and the capture that returns the pixels sat
`Page.captureScreenshot` and a `devicePixelRatio` read, and EVERY ink assertion
in the grid samples at `rect.x * dpr`. The rects say where a box is; the
capture says what colour is at a coordinate. A relayout in between moves the
pixels under coordinates the rects already reported, and the tree fingerprint
reports the page unchanged throughout — because it is unchanged. Nothing about
identity moved.

`LAYOUT_FINGERPRINT_JS` is the same shape of expression over the same elements'
geometry, bracketing the shutter rather than the font reads. The two windows
nest and neither subsumes the other. It hashes each element's path and full-
precision rect — no rounding, because rounding would be a tolerance and the
tolerance that matters is a device pixel, which is what the sample points are
rounded to — plus the four page-wide numbers a rect cannot carry, so the
message can say WHICH thing moved rather than only that the hash did.

    the page scrolled from (0, 0) to (0, 7), and the same 128 elements are at
    different rects (c9499856 then 9d04300c)

The dpr rides back inside it, so the number every sample point is multiplied by
and the fingerprint holding those coordinates to the capture are one read. That
retires a round trip rather than adding one.

`pixelsHeld` is the single gate the four sampling sites now ask, and it is two
independent questions with one answer: the grid past the fold has no pixels for
these boxes, and a page that relaid out has pixels for boxes that have moved.
Neither touches a layout assertion — the rects are the first side of the pair
that came apart.

`FNV1A_JS` is the mixing, defined once and interpolated into both expressions:
the two hashes are only ever compared with each other, so a drift between two
copies would not read as a wrong number, it would read as two readers that
never agree.

Break-tests: the layout fingerprint dropped from the evaluate; a `padding-top`
applied between the rects and the capture (the tree fingerprint stays silent at
128 elements, which is the gap); a scroll (named as a scroll).

## Item 2 — a claim about faces in general, put to the face

`inkGlyphPerCharacter` refuses any fixture string carrying one of
`inkLigatureSeeds`' nine pairs, deliberately as a superset: being wrong towards
refusing an innocent pair costs a fixture author one word. That argument is
about faces in general, and it was the whole of what stood behind the list —
while `CSS.getPlatformFontsForNode`, which returns a glyph count, was already on
the line for every run in the grid.

So the nine pairs are mounted (`inkLigatureRow`, in the first probe's own
declaration, in a Row of their own so the probes' declared widths are not
squeezed by a question about faces) and asked. On this machine:

    2 of those 9 pairs drawn as one glyph by Times (fi, fl)

Neither answer is a failure. A pair drawn as one glyph is a refusal this build
earns; a pair drawn as two is a refusal carried for a build that is not this
one, which is a fair thing to carry and a different thing to say — and the tail
now says which instead of reciting the list's size.

What IS a failure is the row answering about the wrong face, which is the same
shape of mistake the tree fingerprint stops one level up: a census over a family
no run in the grid resolves is a confident statement about somebody else's
ligatures.

The measurement also reaches back into `inkFaceFault`'s third arm, which is the
arm the refusal exists to keep quiet. A run that arrives there anyway used to
be a bare "the face is substituting"; it can now say whether the string holds a
pair this face was measured to join.

    This face was measured to draw fi as one glyph, and this string contains
    it — so the refusal that was supposed to keep this string out of the grid
    did not fire

Break-tests: an empty ligature table; the row put on `monospace` through an
injected stylesheet ("drawn by Menlo and no run this grid scans is"); and the
seed check disabled with a label carrying "fi", which reaches the third arm and
names the pair.

## Item 3 — the read says what it asked

`inkDeclarationGuard` took every key ending in `Own` or `Ancestry` and looked
for the same stem plus `Subject`. That is the convention the `declarations`
table produces, not a property of it, and it left two holes with one shape: a
read named for what it measures rather than for how — `ownStyle`, `labelBox` —
is outside the pairing by spelling, and a MISSING `Own` beside a `Subject` that
IS there is not a mismatch at all, because there is no key to notice.

So the read carries a manifest (`out.reads`) built from the same table it
derives the reads from, and the guard is held to that: every declared stem has
all three answers, `probeBox` — the deliberate complement — has its subject and
must not acquire the other two, and every value in the read with the SHAPE of
one of the three must sit at a stem the manifest names.

Shape rather than spelling is what closes the second hole. `own()` returns a
computed style whatever key it is stored under; `subject()` returns its four
fields; an ancestry is an array of `{at, prop, got, neutral}`. An empty array is
called a chain, and the ambiguity is resolved towards reporting — being wrong
that way costs a line in a table nobody had to write.

Break-tests: `out.ownStyle = own(paths.label)` (invisible to the old guard, and
the item's own example); `out.controlOwn = own(paths.control)` (last session's
control, still caught); and the `Own` read deleted from the table's loop, which
the old guard could not see at all — "label is in the manifest and labelOwn is
not in the read".

## Item 4 — the conjunction derived from the parts

`targetWhole` was a boolean set false at three sites while three counters were
incremented at three others. They agreed by construction and nothing said they
had to: a fourth assertion added to the tap-target claim could clear the
conjunction the tail recites while the census went on reporting three full
populations underneath it.

`BAND_TARGET_PARTS` names the three once. The band loop writes a verdict per
part into a ledger; `bandTargetTally` counts each part at its own counter and
DERIVES the conjunction from the ledger, so there is no boolean an assertion can
reach. `declared` takes each part's census population from the same rows, so a
part cannot be added to the claim without arriving with a counter and a
population.

Three ways the ledger can drift, all reported: a part with no verdict, a verdict
on a part this branch does not make (the stretch's population is the eight bands
with a wrapper), and a verdict under a name the table does not have.

    the tap-target ledger came back with a verdict for "taller" and
    BAND_TARGET_PARTS has no row for it

Break-tests: that fourth verdict, and the trailing-edge assertion no longer
writing one — "nothing decided the control's trailing edge against the band's
content, which is one of the 3 assertions the tap-target claim is made of".

## Item 5 — the verdict the list was standing in for

The blind-spot note named the lines the lexer had lost its place on and asked
the reader to check them. Both values needed for the verdict were in hand: the
phrase's offset in the raw file is one match away and the line it falls on is
one count away.

The obstacle was that the search runs over a string with the whitespace
canonicalised out of it, which is what makes a citation survive a reformat and
also what threw the positions away. `pinStripSpaceLines` is that same
canonicalisation with the raw line every surviving byte came from, one entry per
byte; `pinSpellsPattern` is the expression `pinSpells` decides WHETHER with, so
`pinPhraseLines` locates the very match it found rather than a second spelling
of it.

    the phrase is on [6733], which is among them — so this failure is
    pinCodeOnly's and not the harness's: the assertion is there and the lexer
    blanked it

    the phrase is on [6733], which is not among them — so the blind spot is
    ruled out and what survives at that line is prose

Break-tests: last session's control (the odd-quote regex on the asserted line),
and the other arm — an odd-quote regex on an unrelated line with the assertion
commented out, where the note rules its own blind spot out by looking. A fixture
holds `pinPhraseLines` to four shapes, including a phrase broken across a line,
which is reported on both; and to the property no row can show, that the string
the index is taken over is byte-for-byte the string the search runs on.

## Item 6 — two questions, or two names for one set

`citationExemptInputs` reads one lookup out of `considered` and one out of
`files`, and a fixture holds it to all four pairs. What produces those two is
still one loop in `citingFiles` that appends to the second only inside the
branch that has just written the first — correct, and the only thing making them
different questions.

So the property is asked of the real tree, both halves. Every citing path was
opened, because a file that cites without having been reached is a pair
`citationExemptVerdict` has no arm for and would diagnose as a rename. And
something was opened that cites nothing, because that is the separating case:
with the two sets equal it is unreachable from this repository, the fixture goes
on passing over contrived maps, and an exemption that has simply gone quiet
reports as a rename against a file still sitting on disk.

Break-test: `considered[rel]` moved inside the citation branch — "the
enumeration opened 13 files and 13 of them carry citations, so nothing it looked
inside is silent".

## Item 7 — which of the two stopped the search

`themeLeaves` searches upward for the largest threshold that keeps an answer to
a pair of names, floored at 1 and capped at `themeNearMissReach`. Two different
facts came out with one number in front of them and nothing recorded which:
core.Theme's 1 is its own crowding, and a sparse struct's 3 is a judgement about
where a slip stops being one slip.

`cappedByReach` is recorded where it is known — on the loop's two exits — rather
than inferred afterwards, because `crowdBeyond` is measured at `edits+1` for
both and a struct whose crowding stopped the search exactly at the ceiling would
be indistinguishable from one the ceiling stopped.

The message says it on the one arm that invites the reader to wonder whether a
wider threshold would have found something:

    That threshold is themeNearMissReach and not this struct's own crowding —
    these names are far enough apart to carry a wider one, and 3 edits is where
    a slip stops reading as one slip.

The test asserts the flag on all three sets: core.Theme is not capped (the
ceiling is slack over it), the sparse set is, the crowded set is at the floor for
a reason about its names — and the message cites the constant in the first case
and not in the second, so a reader is never sent to raise a number that is not
binding.

Break-tests: the ceiling moved to 1 (both the existing number assertion and the
new flag assertion fire), and the origin sentence removed from the message.

## The break-tests

**16 run: 15 that had to fail and did, and item 5's control.**

    item 1   the layout fingerprint dropped; a padding applied between the
             rects and the capture; a scroll                                3
    item 2   an empty ligature table; the row put on another family; the
             seed check disabled with a label carrying "fi"                 3
    item 3   `out.ownStyle` written around the table; `out.controlOwn`
             (last session's control); the Own read deleted from the loop   3
    item 4   a fourth verdict in the ledger; the trailing edge writing none  2
    item 5   an odd-quote regex on the asserted line (this session's
             control); one on an unrelated line, assertion commented out    2
    item 6   `considered` written only for citing files                     1
    item 7   the reach ceiling moved to 1; the origin sentence removed       2

Item 5's is this session's control, and it is the same injection last session
used: the difference is what the message says about it now.

## Verification

Ten paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      replay + 19 mjs suites + BROWSER PASS (20 real
                            bands over five shapes and four palettes — 20
                            leading edges, 20 trailing edges, 8 wrapper
                            stretches counted off one ledger the recited
                            conjunction is derived from, 20 indents, 20 fills,
                            20 sets of words and 8 counts, every one counted
                            where its own check ran — drawn by one platform
                            face this browser names over the same tree the
                            rects were read from, in a capture held to the same
                            layout those rects describe, a glyph per character
                            of strings gen.go refuses a ligature pair in, 2 of
                            those 9 pairs measured as one glyph on this face,
                            behind 11 probes)
    android ./gradlew compileDebugKotlin --offline
    android ./gradlew :app:fetchComposeLayoutSources --offline
    GOOS=js GOARCH=wasm go build ./...

No new files. Five changed: browser.mjs, gen.go, checknumbering_test.go,
themenearmiss_test.go, pinfixture_test.go.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

1. **(age 0 · value medium) The layout fingerprint brackets the capture and
   nothing brackets the run metrics against it.** The capture and the rects are
   now held to one layout, and the ink scan's other input is not a rect: the run
   widths come from a `measureText` taken inside the rects' evaluate, and the
   rows are fractions of an ink band measured there too. Those ride with the
   rects and are safe. What does not is the FACE — the platform-font reads run
   after the capture, under the tree fingerprint only, so a font that finished
   loading between the shutter and the last face read would leave the glyph
   counts describing a rendering the pixels do not show, with both fingerprints
   silent because identity held and the layout question had already been asked.
2. **(age 0 · value low) The ligature row is drawn in one declaration and the
   grid has eleven.** `inkLigatureRow` takes the first probe's Style, and the
   census is honest about it: it fails when that family is not one the grid's
   runs resolve. But "one of them" is not "all of them" — eleven probes are
   eleven declarations, and two of them differing in weight are two faces that
   may ligate differently. The check that makes the single row sufficient today
   is `inkFaceFault`'s last arm holding each box to its own probe's family, and
   that is a per-pair claim; nothing states the grid-wide one the census rests
   on.
3. **(age 0 · value low) `inkReadShape` calls a computed style anything with
   more than fifty string values.** The threshold is a fact about how many
   longhands a browser exposes — hundreds — and it is written as a bare number
   with the argument in a comment. `INK_OWN_MEASURED_ON` already records the
   build the exception table was measured against and `inkOwnRead` already
   counts what this build enumerates, so the guard could be told the same
   number the census reads instead of carrying a constant nobody measures.
4. **(age 0 · value low) `bandTargetTally` counts and reduces, and its side
   effect is the census.** It takes `asked` and increments it, which is what
   makes the conjunction and the three counts one reading — and it means the
   function cannot be asked what a ledger comes to without also spending the
   counters. A caller that wanted to check a ledger twice, or a test that wanted
   to put a contrived one in front of it, would move the census on.
5. **(age 0 · value low) `pinPhraseLines` reports the FIRST match and the note
   speaks as though there is one.** A phrase spelled twice in a file — an
   assertion and the comment above it — gives two line ranges, and the verdict
   is taken over whichever came first. That is the right answer whenever one of
   them is the surviving copy and the wrong one when both survive and only the
   later is on a blanked line.
6. **(age 0 · value low) The two enumerations are held apart by a count.**
   `len(considered) > len(files)` says a silent file exists somewhere in the
   repository; it does not say one exists that any exemption is about. The
   fixture's separating case is a path in `considered` and not in `files`, and
   what makes it reachable from a real walk is the same inequality — so the
   check is over the right property and the population it is measured on is the
   whole tree rather than the paths `citationExempt` names.
7. **(age 0 · value low) `cappedByReach` is measured and `themeNearMissReach`
   is still the number.** The flag says which of the two stopped the search and
   the message says it out loud, so a reader of a capped set knows the ceiling
   is what they are looking at. What nothing says is what the struct would have
   afforded: `crowdBeyond` is measured at `edits + 1`, so a capped set knows it
   is not crowded at 4 and stops — and "these names could carry a wider
   threshold" is stated without the width they could carry.
