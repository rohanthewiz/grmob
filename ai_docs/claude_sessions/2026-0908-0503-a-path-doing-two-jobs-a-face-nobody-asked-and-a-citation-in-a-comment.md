# Session: a path doing two jobs, a face nobody asked, and a citation living in a comment

Session: https://claude.ai/code/session_01MQcEV7mAeEV4h1LywCdQhe
Date: 2026-09-08 (follows "a-probe-read-at-the-wrong-element-a-rect-two-answers-disagree-about-and-a-recitation-nobody-counted")

## Ask

"Let's continue working the Next list items" — all seven, every one raised last
session, worked in order.

| # | age·value | item |
|---|---|---|
| 1 | 1·medium | The probe was read at the wrong element and nothing distinguishes the two jobs a path does |
| 2 | 1·medium | `INK_OWN_MAY_DIFFER` is measured against one browser and nothing records which |
| 3 | 1·low | The advance check's two answers are the same engine |
| 4 | 1·low | `asked` counts two of the grid's claims and the tail recites five |
| 5 | 1·low | `pinFreeForm` classifies a language by its file extension |
| 6 | 1·low | `citationReporter` is two methods and the loop above it uses `t` for more |
| 7 | 1·low | `themeNearMiss` calls anything past a case slip "a name that was never right" |

## Item 1 — the two jobs a path does, named and held

A path here names an element, and the checks ask two different things of one:
WHERE the pixels are, and WHOSE declaration is in question. For a box that
draws its own glyphs those are the same element and the distinction never
surfaces; for a box with the glyphs inside it they are not. That is the fault
the probes shipped with, fixed last session at the probe and nowhere else.

Measured across the grid, every element read for a declaration:

    the label       20 spans, no element children, direct text
    the count        8 spans, no element children, direct text
    the probe text  11 spans, no element children, direct text
    the probe Box   11 divs, one element child, no text of its own

The count is the interesting one, and the Next item's premise about it was
wrong in a way worth recording: it reads as a pill with digits inside it and it
is not. `components.Badge` is `core.Text` with a fill, a radius and paddings on
it, so the pill and the digits are ONE element and `badgeOwn` at the pill IS at
the digits. True by accident — a badge that grew an icon beside its number
becomes a Box with a Text in it, the path goes on naming the Box, and the
comparison quietly starts holding a container to a probe built from the digits'
Style.

So `inkSubjectFault` holds every declaration subject to being a glyph-drawing
leaf, asked in `inkUnreadable` before the ancestry sweep and the computed-style
comparison because a wrong element makes both of those answers wrong quietly.
And the complement at the one place the file has a path for each job:
`inkProbeBoxFault` holds the probe's sample rect to being a box with exactly one
child and no text of its own, which is what makes `inkProbeTextPath`'s `/0` a
derivation from what `inkProbeFor` builds rather than an index that lands right.

Break-tests: the count's declaration read at the control, the probe's sample
rect read at its text node, the probe's declaration read at its Box.

## Item 2 — the build the table of exceptions was written against

`INK_OWN_MAY_DIFFER` is a list of exceptions to one build's enumeration and
nothing recorded which. The widening is the design — a property that ships in a
later Chrome joins the comparison with nobody editing the file — and it is also,
on the first run on a newer browser, a failure with a property name nobody
recognises, differing for a reason nobody has thought about, in the middle of a
check about antialiasing.

`INK_OWN_MEASURED_ON` is `Chrome/152.0.7977.83` and 476, read at run time off
`/json/version`, and `inkOwnBuildNote` adds a paragraph to both messages that
can use it — the own-comparison's failure and the unused-permission census —
saying which build the table was measured against and how far this one is from
it, in whichever direction. The count and not the list: the list would be four
hundred names regenerated on every Chrome, and the delta is what separates
"the two declarations came apart" from "this browser grew a property".

The dpr is deliberately not recorded, with the reason written down: a computed
style is in CSS pixels, so nothing in this comparison moves with it.

Break-tests: a differing property under a table measured on a narrower build
(the message names the 16 properties this browser has that it did not), and an
unspent permission under a table measured on a wider one.

## Item 3 — a third party to a question two answers agree on by construction

`inkRunRectFault` holds the run rect to the advance the same face gives for the
same string and calls that two answers. It is two answers to the arithmetic and
one to the shaping: Chrome's canvas measurement and Chrome's layout share a
shaper, so a face that resolved to something other than what was asked for moves
both together. Nothing on the page can see it either — a computed `font-family`
is the REQUEST, and neither the canvas nor `getComputedStyle` reports which
entry of the list was reached.

`CSS.getPlatformFontsForNode`, over the DevTools protocol, is the compositor's
own record: one entry per face with the glyphs that face drew. Read with the
rects and the screenshot, off the same layout.

    every run in the grid    one face, "Times"
    the glyph counts         12 for "January 2026", 26 for the long title,
                             1 for a count's digit, 6 for the probe's own text
                             — exactly the string lengths

So three claims. One face per run, because a run with fallback in it makes the
ascent and the advance averages over faces. One glyph per character, which is
the third answer to "was the string on the page the string that was measured" —
`measureText(textContent)` reads the DOM's characters and this counts the
compositor's glyphs. And the same family as the probe, which neither of the
other guards can see: `font-family` agrees on both sides precisely because it is
the request they share.

Break-tests: a run that fell back to a second face, a face that drew one glyph
fewer than the string's characters, a probe drawn by a different family from the
box it answers for, and no platform font reported at all.

## Item 4 — the third skip, whose argument was that it was loud

The census closed the words and the counts and left the tap target, the declared
inset and the band's own fill reciting `BAND_RENDERS.length`, on the grounds
that their skips are loud. The fill's is not: it is sampled inside
`if (!gridClipped)` and skipped with every other pixel read.

All five are counted now, each at its own check's success path, and the argument
is retired rather than reapplied — it is the argument the label's skip also had
until it turned out not to hold. The tap target is counted as a conjunction of
its three assertions (leading edge, trailing edge, and the wrapper stretch on
the disclosure branch), because any one of them failing means the target did not
span the band.

    the grid clipped at 800x665
      before   2 census lines — the words and the counts
      now      3: the fill's silent skip says so too

Break-tests: the window at 665 (the fill), the indented shape no longer held to
its own inset (4 of 20), and the disclosure branch's tap target moved off the
leading edge (8 of 20).

## Item 5 — the phrase that survives in the sentence explaining the assertion

`pinFreeForm` says whitespace between tokens is insignificant in a consumer's
language, which is the property `pinStripSpace` needs. The item's point is that
the extension is a proxy for a grammar and the literal check watches only the
citation side: a source with a literal in it is still normalised by a rule that
is wrong for that span.

The interesting direction turned out to be the other one. A phrase with no quote
in it can match inside a COMMENT, and both harnesses are written in prose as
much as in code — every assertion is discussed somewhere above itself. A row
whose phrase survives only in that sentence goes on passing after the assertion
is deleted, which is the exact rot the citations exist to catch.

    lexed, code is        16.8% of browser.mjs
                          17.3% of pin.swift

which is the measure of how much of the search space that sentence was competing
with. `pinCodeOnly` blanks comments and literals per language — Swift's block
comments nest and JavaScript's do not, an apostrophe is prose in one and a
string opener in the other, a backtick quotes an identifier in one and opens a
template in the other — and `pinFreeForm` became a two-field struct so the
lexer's assumptions live beside the whitespace one instead of in a switch. What
is deliberately not lexed is JavaScript's regex literal, bounded to a line by
construction and covered by a floor on how much code survives.

The control: comment out `if (!pinSame(total, c.offer)) {` in browser.mjs. Under
the whole-file search the suite passes; under the lexer it reports a DELETED
assertion. That is the failure this half was added to catch, arriving through
the door the citation check was watching from the other side.

Break-tests: that control, a `.py` consumer, a `pinFreeForm` row with an empty
Lexis, a lexer whose block comment never closes (4.0% survives, and the floor
names the cause), and an eleven-row table holding the lexer to each construct in
each language, both directions — what has to survive and what must not.

## Item 6 — the decision the walk still made inline

The item was skeptical of extracting more from the walk, and rightly: `t.Fatalf`
for an empty enumeration, `t.Logf` for which set was searched, `t.Errorf` for an
unclassified sense are all one arm each. The exemptions are not. They are a
three-way decision — not reached, reached and silent, doing its job — and every
row of `citationExempt` is in the third arm, so the two sentences that tell a
rename from a dead exemption were prose nobody could ask a question of.

`citationExemptVerdict(path, why, from, reached, cited)` returns the sentence or
"", and the test holds all four inputs, including the one the ORDER decides
rather than the conditions: not reached AND not citing has to report the rename,
because a walk that never opened a file cannot have seen a citation in it.

Break-tests: the two arms swapped, and the silent arm dropped.

## Item 7 — the threshold the struct decides

`themeNearMiss` compared case-insensitively and called everything further out "a
name that was never right" — a confident diagnosis of a deliberate choice, made
about a transposition, a doubled letter, a singular for a plural. The item said
an edit distance would need a threshold and that a fixture of made-up
misspellings is not evidence. Both halves come off the struct instead.

    core.Theme                            872 leaves, 50 parents
    closest two sibling leaves            1 apart — Spacing.XS and Spacing.XL
    most siblings within 1 of any leaf    1, so an answer is at most two names
    most siblings within 2                4, so the next threshold out is five

The first reading settles the message's SHAPE rather than the number: this
struct HAS a pair one edit apart, so at any threshold a path can be one edit
from two real names, and "this is what was meant" is a claim the function is not
in a position to make. It names every candidate. The second and third make 1 the
threshold, held from both sides — at 1 the answer stays a pair, at 2 it becomes
five, and a wall of names is what the function exists not to print.

The distance is optimal string alignment rather than Levenshtein, because an
adjacent swap is the most common typo there is and costs two edits without it —
which would have needed the threshold of 2 and the five-name answer.

And the fixture is generated, not listed. Every one of the 872 leaves is
perturbed by each edit the distance is defined over — a doubled letter, a
dropped one, a pair swapped — and `themeNearMiss` has to name the leaf it came
from. 2616 slips, every one traced back, and not one of them a name anybody
chose.

    Typography.Caption.LineHeigth   "Typography.Caption.LineHeight is one edit
                                    from it … and is probably what was meant"
    Spacing.XT                      "Spacing.XL and Spacing.XS are each one edit
                                    from it, and which was meant is not something
                                    this can say"
    Typography.Caption.LetterSpacing "nothing under Typography.Caption is within
                                    one edit of it … so a slip of that kind is
                                    ruled out. Two independent slips are not, and
                                    neither is a name that was never right"

Break-tests: a case slip, a transposition, a doubled letter, a name one edit
from two real siblings, a name that was never one, and the threshold moved to 0
and to 2 (each fails against a different one of the three measurements).

## The break-tests

**25 run: 24 that had to fail and did, and item 5's control.**

    item 1   the count's declaration read at the control; the probe's rect
             read at its text node; its declaration read at its Box        3
    item 2   a differing property on a wider browser; an unspent
             permission on a narrower one                                  2
    item 3   a second face in a run; a glyph fewer than characters; a
             probe drawn by another family; no platform font at all        4
    item 4   the window at 665; the indent unheld; the disclosure
             target off its leading edge                                   3
    item 5   a .py consumer; an empty Lexis; a block comment that never
             closes                                                        3
             the assertion commented out — passes the whole-file search
             and fails the lexer                                           1
    item 6   the two arms swapped; the silent arm dropped                  2
    item 7   a case slip; a transposition; a doubled letter; one edit
             from two siblings; a name that was never one; the
             threshold at 0; the threshold at 2                            7

Item 5's is this session's control, and it is the same shape item 2's was last
session: an injection that the old check passes and the new one names.

## Verification

Ten paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      replay + 20 mjs suites + BROWSER PASS (12 checks,
                            20 real bands over five shapes and four palettes —
                            20 tap targets, 20 indents, 20 fills, 20 sets of
                            words and 8 counts, every one counted where its own
                            check ran — read at the element that draws the
                            glyphs rather than at the box around it, drawn by
                            one platform face this browser names, a glyph per
                            character, the same face the probe was drawn with,
                            behind 11 probes resolving every computed property
                            their text nodes do except the 22 with a reason,
                            measured against Chrome/152.0.7977.83)
    android ./gradlew compileDebugKotlin --offline
    android ./gradlew :app:fetchComposeLayoutSources --offline
    GOOS=js GOARCH=wasm go build ./...

One new file: wasm/verify/themenearmiss_test.go. Five changed: browser.mjs,
gen.go, checknumbering_test.go, pinfixture.go, pinfixture_test.go.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

1. **(age 0 · value medium) The face check reads over the DevTools protocol and
   every other guard in this file reads through the page.** That is what makes
   it a third party and it also makes it the one claim whose failure mode
   nothing else covers: `DOM.querySelector` is asked once per subject against a
   root node id taken before the reads, and a mount that moved the tree between
   them would return a node id for a box that is no longer there. The page reads
   are one `Runtime.evaluate` over a snapshot of the same layout; these are
   fifty-seven round trips over a document that could in principle change under
   them, and nothing holds the two sets of reads to describing one page.
2. **(age 0 · value medium) `pinCodeOnly` blanks JavaScript's regex literal by
   not lexing it, and the floor is what stands behind that.** The floor
   discriminates a lexer that lost its place (4.0% survives) from two files that
   are mostly prose (16.8% and 17.3%), which is a real bound with a measurement
   on each side. What it cannot see is a regex whose quote opens a string that
   swallows the rest of ONE line — the line an assertion happens to be on. That
   is a single false "deleted assertion" in a table of forty-odd citations, and
   the message it produces sends a reader to look at a harness that is fine.
3. **(age 0 · value low) The subject guard holds three paths and the file has
   more paths than that.** `r.band`, `r.control`, `r.wrapper`, `r.leading`,
   `r.chevron` are all read as rects and none of them is read as a declaration,
   so the confusion item 1 is about is not available at any of them today. What
   makes that true is that nobody has added a declaration read to one — the same
   sentence that was true of the badge until `components.Badge` was asked to
   grow, and the guard covers the three that already ask rather than the shape
   of the mistake.
4. **(age 0 · value low) The glyph-per-character claim is a property of these
   fixture strings.** "January 2026" and a single digit have no ligature in any
   face; the first title with an "fi" in it makes this fire for a reason that is
   correct rendering. The note says so and says the fixture is the thing to
   change — which is a decision deferred to whoever hits it, in a check that
   will present itself as a font substitution rather than as a fixture choice.
5. **(age 0 · value low) `asked.targets` counts a conjunction and the tail
   recites it as one number.** A band that failed the wrapper stretch and a band
   whose leading edge moved both decrement the same counter, and the census line
   says how many bands lost the claim without saying which of the three
   assertions cost them. The three failures above it name themselves, so nothing
   is lost today; what the count cannot do is distinguish a run where one band
   lost all three from a run where three bands lost one each.
6. **(age 0 · value low) `citationExemptVerdict` is tested and the walk that
   calls it is still the untested part.** The extraction moved a three-way
   decision out; what stayed is a loop over `citationExempt` that decides which
   paths to ask about, and the map lookup that feeds `reached` is the thing that
   would have to be wrong for a rename to go unreported. That is one line and it
   is the line the whole pair rests on.
7. **(age 0 · value low) `themeNearMiss`'s threshold is derived from
   core.Theme and the function takes any leaf map.** The measurements in the
   test are of `core.DefaultTheme`, and the only caller passes exactly that — so
   the derivation holds for the one use. A second caller with a different struct
   would get a threshold measured against somebody else's field names, and
   nothing in the signature or the constant says which struct the number is
   about.
