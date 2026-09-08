# Session: a tree two readers disagreed about, a claim about the words, and a threshold that travelled

Session: https://claude.ai/code/session_01MQcEV7mAeEV4h1LywCdQhe
Date: 2026-09-08 (follows "a-path-doing-two-jobs-a-face-nobody-asked-and-a-citation-in-a-comment")

## Ask

"Let's continue working the Next list items" — all seven, every one raised last
session, worked in order.

| # | age·value | item |
|---|---|---|
| 1 | 1·medium | The face check reads over the protocol and nothing holds it to the page the rects came from |
| 2 | 1·medium | `pinCodeOnly` blanks JavaScript's regex literal by not lexing it, and the floor is what stands behind that |
| 3 | 1·low | The subject guard holds three paths and the file has more paths than that |
| 4 | 1·low | The glyph-per-character claim is a property of these fixture strings |
| 5 | 1·low | `asked.targets` counts a conjunction and the tail recites it as one number |
| 6 | 1·low | `citationExemptVerdict` is tested and the walk that calls it is still the untested part |
| 7 | 1·low | `themeNearMiss`'s threshold is derived from core.Theme and the function takes any leaf map |

## Item 1 — the page the two readers had to be describing

Every other measurement in the band grid comes back from ONE `Runtime.evaluate`:
rects, run metrics, ancestry sweeps and computed styles are taken inside a
single expression, so they are of one layout by construction and a mount landing
between two of them is not a thing that can happen. The face read is not that
shape. `CSS.getPlatformFontsForNode` takes a node id, the id comes from
`DOM.querySelector`, and the selector is asked against a root taken once before
any of them — 51 round trips over a document that could in principle change
under them.

Neither read could say so. A face is a family name and a glyph count and both
are entirely plausible for the wrong element; what it would produce downstream
is `inkFaceFault`'s third arm, a run whose glyph count is not its character
count, reported as a font substitution that did not happen.

`TREE_FINGERPRINT_JS` is one expression evaluated on both sides, and the first
one rides back INSIDE the rects' own evaluate — which is what makes it the
rects' page rather than a third reading of a third moment. It is the mounted
element count and an FNV-1a over each element's path, tag, child count and text
length in document order.

Deliberately not the geometry, with the reason written down: a scroll, a resize
or a font finishing loading moves every rect and changes no fact about which
element is which, and the claim is about identity — that the node id the
protocol resolved for a path names the element the page read that path at.

    128 mounted elements, hash deee4254, on both sides

On a disagreement `treeMovedFault` is one message and every face arm is
suppressed through a single `faceFault` wrapper, because an unheld claim is not
evidence.

Break-tests: the fingerprint dropped from the evaluate, an element appended
between the two reads, and a text node appended (the same count, a different
tree — the message says so).

## Item 2 — the small version of the damage, which no percentage can see

`pinCodeOnly` declines to lex JavaScript's regex literal, and a quote inside one
opens a string that was never opened. The floor is what stood behind that: 4.0%
of a file surviving is a lexer that lost its place, and the two consumers
measure 16.8% and 17.3%. That is a real bound with a measurement on each side
and it is a bound on CATASTROPHE.

The small version is one line. A regex with an odd number of quotes blanks the
rest of its own line; if an assertion is spelled there, its citation reports as
DELETED. Two hundred bytes out of a hundred thousand moves no percentage, and
what comes out is one false failure in a table of forty, pointing at a harness
that is fine.

So `pinCodeOnly` returns the lines it ended INSIDE a single-line string. That is
not a heuristic about regexes — it is the complete set of places the damage can
start, because an unterminated single-line string is a syntax error in both
these languages. A regex with an EVEN number of quotes of one kind pairs them
among themselves and blanks only its own body.

Both consumers report none, so the claim has a measurement behind it. And the
deletion report now separates three things instead of asserting one:

    gone from the file          the assertion is gone
    in the file, not in code,   the sentence above a deleted assertion outlived
    no open-quote lines         it — the rot the lexer was added to catch
    in the file, not in code,   the lexer's own blind spot, with the lines
    open-quote lines            named

The control: put `if (/["]/.test(String(c.what)) || ` in front of
`!pinSame(total, c.offer)` in browser.mjs. The lexer reports line 6167, and
every deletion message that follows carries "check those first — this failure
is pinCodeOnly's and not the harness's". Before this session the same injection
produced six confident deletion reports and nothing else.

Break-tests: that control, the assertion commented out (which produces the
prose arm and explicitly rules the blind spot out), and four rows in the lexer
table — an odd quote reported, paired quotes not reported, a genuinely
unterminated string, and Swift's multi-line string, which is closed by its
delimiter and must not be.

## Item 3 — the shape of the mistake rather than the paths that make it

`inkSubjectFault` was asked at the three paths a declaration is read at, which
is a fact about the current file rather than a property of it: `r.band`,
`r.control`, `r.wrapper`, `r.leading` and `r.chevron` are read as rects and none
is read as a declaration, so the confusion is not available at any of them
today. What makes that true is that nobody has added a declaration read to one —
the same sentence that was true of the badge until `components.Badge` was asked
to grow.

Two halves. The evaluate's three reads — ancestry, computed style, subject
guard — are derived from one `declarations` table, so they go together by
construction. And `inkDeclarationGuard` holds what came BACK to the same
pairing: every `*Own` and `*Ancestry` key in the read must have its `*Subject`
beside it, asked of every band, every probe and the grid.

The first makes it easy to do right; the second makes doing it wrong a failure.

Break-test: `out.controlOwn = own(paths.control)` written around the table — 20
messages, one per band, naming the key.

## Item 4 — the claim that was about the words and read as a claim about the face

`inkFaceFault` holds every scanned run to one glyph per character. That holds on
this grid because of what this grid says, and not because a face draws a glyph
per character in general: a face is entitled to draw one glyph for "fi" and the
common ones do. The first fixture string with an f-ligature in it makes the
check fire for a reason that is correct rendering, and it fires as a font
substitution in the middle of an antialiasing scan.

`inkGlyphPerCharacter` decides it where the strings are. A scanned run may carry
no f-ligature pair (`ff fi fl ft fb fh fj fk`, and the discretionary `st`) and
nothing outside printable ASCII — the second for the other direction, because a
composed form or a combining mark is any number of glyphs for any number of code
points and the two counts stop being comparable before a face has an opinion.

The seed list is deliberately a superset: what it has to rule out is a fixture
string whose glyph count is a question about which optional ligatures a build
shipped with, and being wrong towards refusing an innocent pair costs a fixture
author one word.

Asked at the rendered node rather than at `bandRenderGroup`, because what the
browser counts glyphs for is what the component put in the DOM. Three subjects:
the label, the count badge's digits, and the probe's own run.

    "January 2026 final"    fails at the fixture, naming "fi"
    "illłim"                fails, naming 'ł' and the code point/glyph split

Break-tests: those two. `inkFaceFault`'s message now says where the decision was
made, so what is left for it is the substitution itself — a face doing it to a
string that check passed.

## Item 5 — three assertions counted as one number

`asked.targets` counted the conjunction of the leading edge, the trailing edge
and the disclosure branch's wrapper stretch. That is right about the tail's
sentence and wrong about the census: a band that failed the stretch and a band
whose leading edge moved decremented one number, so the line said how many bands
lost the claim without saying which of the three cost them — and could not tell
one band losing all three from three bands losing one each.

The three are counted at their own success paths and the census reports those;
`targets` stays as the number the tail recites and is not censused itself,
because a shortfall in it is a shortfall in one of the three and the three say
which.

The stretch's population is the bands that HAVE a wrapper — eight, not twenty. A
claim counted over twenty bands when eight can make it is a census line that
fails on every run.

    the leading edge broken on the disclosure branch
      8 of the 20 bands ... did not get their tap target's leading edge
      held to the band's — 12 did

    the wrapper stretch broken
      8 of the 8 bands mounted behind a heading wrapper did not get the
      button held to the wrapper it is stretched across — 0 did

Break-tests: those two.

## Item 6 — the line the whole pair rested on

`citationExemptVerdict` tells a renamed exemption from a silent one and a
fixture holds it to all four inputs. What FED it was two expressions in the
walk: `reached` from `considered`, every path the enumeration opened, and
`cited` from a set built while walking `files`, the paths that turned out to have
citations. Two lists, two different questions, and the verdict function cannot
tell them apart — hand it the wrong pair and it reports the wrong failure
confidently.

The interesting swap is `reached`, and it is the natural mistake: `files` is the
list the loop walks and the only one in scope where an exemption is first
noticed. Taken from there, an exemption whose file is still sitting on disk and
has simply stopped citing anything reads as NOT REACHED, and the reader is sent
to look for a rename that never happened — the exact confusion the ORDER of the
two arms inside `citationExemptVerdict` exists to prevent, arriving one level up
where the ordering cannot see it.

`citationExemptInputs(path, considered, files)` is a function of the two
enumerations, and the fixture hands it the case that separates them: a path the
walk opened and found no citation in.

Break-test: `reached` derived from `files` — two rows fail, the separating one
and the one defining the function over both maps independently.

## Item 7 — the threshold that travels with the names

`themeNearMissEdits` is 1 because of three readings of core.Theme's own field
names, and `themeNearMiss` took a bare `map[string]bool` and read the constant.
The only caller passes core.Theme's leaves, so the derivation held for the one
use — and nothing in the signature or the constant said which struct the number
was about. A second caller would have got core.Theme's number measured against
somebody else's names, silently, with the message still citing Spacing.XS and
Spacing.XL as its reason for hedging about a struct that has neither.

`themeLeafSet` carries the leaves with their own measurements: the threshold,
the closest sibling pair and how far apart it is, and the two crowd readings
that bound the threshold from each side. `themeLeaves` does it once per struct;
`themeNearMiss` reads `edits` off the set it is handed and the ambiguity
sentence names the closest pair THAT set has.

The threshold is the largest distance at which an answer is still at most a pair
of names, searched upward from a floor of 1 — below which there is no threshold
at all, only exact match. `themeNearMissReach = 3` is the ceiling and is the one
judgement left here, written down as one: a sparse struct has no crowding at any
distance, and that is not a licence to report a name five edits away as probably
what was meant.

    core.Theme                          50 parents, closest pair 1 apart,
                                        within 1 at most 1 sibling, within 2
                                        at most 4 → threshold 1
    four names no two within 3          threshold 3, stopped by the ceiling
    five names each 1 from the others   threshold 1, the floor

The test's job inverted: it used to compute the three readings, and now it
recomputes them independently and holds the DERIVATION to them — plus a second
test over two contrived structs, because one struct gives one answer and a
constant with a walk in front of it would give the same one.

Break-test: `themeNearMissReach` moved to 1 — core.Theme's threshold is then
capped by the ceiling rather than by the struct, the sparse set stops widening,
and the message over it stops naming a leaf two edits away.

## The break-tests

**12 run: 11 that had to fail and did, and item 2's control.**

    item 1   the fingerprint dropped; an element appended between the
             two reads; a text node appended (same count, new hash)      3
    item 2   the assertion commented out — the prose arm, with the
             blind spot explicitly ruled out                             1
             a regex literal's odd quote on the asserted line — the
             lines named, and every deletion report says so              1
    item 3   a declaration read written around the table                 1
    item 4   a label carrying "fi"; a probe run with 'ł' in it           2
    item 5   the leading edge off on the disclosure branch; the
             wrapper stretch lost                                        2
    item 6   `reached` derived from the citing files                     1
    item 7   the reach ceiling moved to 1                                1

Item 2's is this session's control, and it is the same shape item 5's was last
session: an injection that the previous check reports wrongly and the new one
diagnoses.

## Verification

Ten paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      replay + 20 mjs suites + BROWSER PASS (12 checks,
                            20 real bands over five shapes and four palettes —
                            20 leading edges, 20 trailing edges, 8 wrapper
                            stretches, 20 indents, 20 fills, 20 sets of words
                            and 8 counts, every one counted where its own check
                            ran — drawn by one platform face this browser names
                            over the same tree the rects were read from, a glyph
                            per character of strings gen.go holds to being
                            strings that claim is about, behind 11 probes)
    android ./gradlew compileDebugKotlin --offline
    android ./gradlew :app:fetchComposeLayoutSources --offline
    GOOS=js GOARCH=wasm go build ./...

No new files. Six changed: browser.mjs, gen.go, themenearmiss_test.go,
checknumbering_test.go, pinfixture.go, pinfixture_test.go.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

1. **(age 0 · value medium) The fingerprint joins the two readers on identity
   and nothing joins them on LAYOUT.** That exclusion is deliberate and written
   down — a scroll or a resize moves every rect and changes no fact about which
   element is which — and it leaves a real gap on the other side of the same
   window: `Page.captureScreenshot` and the dpr read both sit between the rects
   and the second fingerprint, and a relayout there moves the pixels under
   coordinates the rects reported. Every ink assertion in the grid samples at
   `rect.x * dpr`, so what that produces is a colour read from the wrong box,
   with the fingerprint reporting the page unchanged because it was.
2. **(age 0 · value low) `inkGlyphPerCharacter` refuses a superset and the
   superset is not measured.** `inkLigatureSeeds` is nine pairs chosen from what
   serif faces commonly carry, and the check that would fire because of one is
   `inkFaceFault`, which reads the platform face this browser actually resolved.
   Nothing asks that face whether it ligates "st" — so the fixture is refused a
   word on a claim about faces in general while a third party who could answer
   for this one is already on the line.
3. **(age 0 · value low) `inkDeclarationGuard` pairs reads by a name suffix.**
   It takes every key ending in `Own` or `Ancestry` and looks for the same stem
   plus `Subject`. That is the convention the `declarations` table produces and
   it is held by string surgery: a read named for what it measures rather than
   for how it is measured — `labelBand`, `probeBoxSubject` — is outside the
   guard by spelling, and a fourth declaration read named `ownStyle` would be
   too.
4. **(age 0 · value low) The tail recites `targets` and nothing joins it to the
   three the census counts.** `targetWhole` is set false at each failing
   assertion and the three counters are incremented at each succeeding one, so
   they agree by construction today. What nothing states is that they must: a
   fourth assertion added to the tap-target claim that cleared `targetWhole`
   without a counter of its own would leave the census reciting three full
   populations under a conjunction that had dropped.
5. **(age 0 · value low) The blind-spot note names the lines and does not say
   whether the phrase is on one.** "Check those first" is a hint where a verdict
   is available: the phrase's offset in the raw file is one search away, and the
   line it falls on is one count away, so the message could say "the assertion
   is on line 6167, which is one of them" instead of asking the reader to.
6. **(age 0 · value low) `citationExemptInputs` is tested and `citingFiles` is
   what fills both its arguments.** The extraction moved the two lookups out and
   a fixture holds them apart. What produces `considered` and `found` is still
   one loop in `citingFiles` that appends to the second only inside the branch
   that wrote the first — correct, and the only thing making the two maps
   different questions rather than two names for one.
7. **(age 0 · value low) `themeNearMissReach` is the one judgement left and no
   real struct is near it.** core.Theme measures 1 against a ceiling of 3, and
   the only set that reaches the ceiling is four contrived names. So the number
   is slack over everything this repository asks about, which is what the test
   asserts — and a struct that landed on 3 would get a threshold set by the
   ceiling with nothing to say whether its names could have afforded 4.
