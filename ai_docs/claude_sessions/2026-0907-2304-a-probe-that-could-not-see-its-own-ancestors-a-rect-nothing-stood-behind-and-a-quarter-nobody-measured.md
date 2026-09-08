# Session: a probe that could not see its own ancestors, a rect nothing stood behind, and a quarter nobody measured

Session: https://claude.ai/code/session_017k47JkhG1WBZR5nP29c7Ge
Date: 2026-09-07 (follows "one-probe-for-thirty-boxes-a-scan-that-read-a-wider-rect-than-it-measured-and-a-row-a-hundredth-of-a-pixel-inside-the-band")

## Ask

"Do all in the next list" — all seven, every one of them raised last session.

Last session's move was **a measurement standing in for measurements it was not
taken of**. This one is the same move asked of the things that replaced it: the
per-declaration probes copy a declaration and cannot copy an ancestor; the run
rect is the source of three readings and the subject of none; the fractions are
checked against their own definition and never against a distance; the probe Row
inherited every guard the bands have except the one that matters.

| # | age·value | item |
|---|---|---|
| 1 | 0·medium | A probe reproduces a declaration and cannot reproduce an ancestor |
| 2 | 0·medium | The run rect is where three readings come from and what none of them is about |
| 3 | 0·low | The margin at each end of the band is a quarter by assertion |
| 4 | 0·low | The probe Row's own fold is the grid's, and only the bands are checked against it |
| 5 | 0·low | `caseNumbers` names the assertion that reads each number in prose |
| 6 | 0·low | `citationSenses` decides three senses the same way and the sameness is the decision |
| 7 | 0·low | The derived theme changes a caption tier and nothing holds it to changing only that |

## Item 1 — the half of the question a copy cannot carry

`inkProbeFor` copies the scanned node's whole `core.Style`, so every declaration
an element makes about itself travels. What no copy carries is what sits above
it: a transform, an opacity under one, a filter, a `will-change` anywhere up the
chain composites the subtree, and a browser draws text into a composited layer
by a different route. The probe stays on the old route and goes on reporting a
grey box for a band that has moved off it.

Holding one chain to the other is not available — a label sits inside a control
inside a Row inside a page box, a probe sits in the probe Row, and the two
structures have no reason to match. What both CAN be held to is the thing the
probe's argument actually needs:

```
    INK_PATH_PROPS   transform, perspective, filter, backdrop-filter,
                     will-change, opacity, mix-blend-mode, isolation,
                     transform-style, contain, -webkit-font-smoothing,
                     text-rendering — with the value each reads as when it is
                     doing nothing
```

With every strict ancestor neutral on all twelve, the declaration is the whole
of what decides the rendering path, and copying the declaration copies the
question. Both sides are held to it: a band whose ancestry stopped being neutral
suppresses its own scan, a probe's suppresses every box it answers for.

Break-tests: `Rotate: 0.0001` on the probe Row (all eleven probes fire, naming
`root/20` and the matrix, and every band is suppressed), and the same on one
band's page box (that band's label and count fire, naming `root/0`, and no other
band is touched). The angle is a ten-thousandth of a degree because at half a
degree the WIDTH check fires first and reports a squeeze — a real ancestor
transform, detected by the check that exists for it.

## Item 2 — three readings from a rect, and nothing about the rect

`r.labelBand.runX/runW` is the browser's own answer to where the glyphs are. The
three ink rows are fractions of a band measured off that rect's top; `scanInk`
sweeps it; `surplusInk` holds everything outside it to the backdrop. All three
are taken FROM the rect, so none of them says the words are in it.

Half of that gap was already closed and nobody had noticed: a rect that shrank
onto part of the words leaves ink in the surplus, and surplusInk fires. It was
tried — narrowing `runW` by 6px reports "something is painted in the label's
rect and outside its words", correctly. What nothing saw is the other direction:
a rect WIDER than the words, or displaced into the empty half of a stretched
box, whose extra columns are backdrop and answer nothing either way.

`inkExtent` is the screenshot's own answer — the first and last column of the
element's box that is not the band's fill, on the rows already being scanned —
and the rect is held to reaching within a gutter of each end of it.

```
    the paint and the client rect, over twenty bands
    outward (paint past the rect)     at most 0.52px   AmberTheme, plain
    inward  (rect past the paint)     at most 0.71px   tall caption, long title
```

Both signs are one quantity, so they are one constant: `INK_RUN_GUTTER` is now
"how far the paint and the rect may part company at either end", measured at
0.71 worst, one device pixel clearing everything, two with a pixel of margin.

Break-tests: `runW + 6` (fires, "the paint ends 5.48px inside the rect"),
`runX - 6, runW + 6` (fires on the leading arm, "the paint starts 6.00px
inside").

## Item 3 — a quarter is a fraction; a rounding is a device pixel

`INK_ROWS` is `[0.25, 0.5, 0.75]` and the case for the outer two was that a
quarter of the band keeps them off the baseline and the x-height line, where a
rounding lands on a horizontal edge of every glyph at once. A quarter of a band
is a fraction; what a rounding has to survive is device pixels; the two are
joined by a font size and a display, and the tall-caption theme is in this grid
precisely because that join moves.

`inkBandRows` now measures it. Over the twenty bands and eight pills at dpr 1:

```
    band              height   clear of the two edges
    Material label    5.53px   1 and 2, or 2 and 1, by branch
    Default label     5.99px   1 and 2, or 2 and 1, by branch
    Default count     8.73px   2 and 2, or 2 and 3
    tall label        8.75px   2 and 2
    tall count       12.77px   3 and 4
```

`INK_EDGE_CLEARANCE` is 1 — a row must be OFF both edges — and deliberately not
2, which would fail two of the three bundled palettes and be a number chosen
against nothing.

Break-tests: the floor at 2 (fires on 15 of the 28 scanned boxes — the control
that says 1 is the measurement and not a preference), and `INK_ROWS` at
`[0.02, 0.5, 0.75]` (fires at 0 clear of the x-height line).

## Item 4 — the guard the probes did not inherit, and what it was hiding

The bands each run `foldVerdict` before they are sampled. The probe Row mounts
after all twenty of them and was checked only for its width — and a box past the
bottom of the window is not the "not laid out" case, because it has a perfectly
good rect. It was the pixel loop reading null for every sample, leaving `worst`
at 0 and `fringe` at null, and **passing**.

The control is the sharp one. At `--window-size=800,665` every band fits and
four probes are clipped:

```
    the guard on    4 probes fire, naming the knob, and every band that rests
                    on them is suppressed
    the guard off   PASS
```

So: the same `foldVerdict` the bands get, with the same knob named, plus a
second guard for the other way a probe can have a rect and no pixels — `fringe
=== null` after a clean fold and width check is the capture falling short, and
"no coloured fringe" about no pixels is a statement about nothing.

Break-tests: the window at 665 (four probes past the fold), and at 585 with the
fold guard disabled (the probes report "not one of its 56×15 device pixels is in
the screenshot").

## Item 5 — a phrase, because a field name is not evidence

`caseNumbers` names each number and the assertion that reads it, and `resolution`
derives the floor two harnesses hold their tolerances to over the read ones. The
coverage between that table and the STRUCT is reflective and runs both ways. The
coverage between the table and its two CONSUMERS was prose: an assertion deleted
from pin.swift or browser.mjs leaves a row still saying the number is read, and
the floor stays tightened for a distinction nobody makes — a failure that never
arrives, because a floor that is too tight is a floor nothing violates.

Each row now carries `By map[string]string` — the consumer, and the phrase its
assertion is spelled with — against a closed `pinConsumers` set.

The first attempt used the FIELD's name, which is the obvious thing and is
wrong. browser.mjs mounts internal/bandfixture too, whose cases have an `Offer`
of their own read as `c.offer` two thousand lines before the pin check does the
same. Rewriting the pin check's comparison out of existence left the token
behind and the test passed. So the entry is the comparison —
`pinSame(total, c.offer)` — which belongs to one assertion and goes when it
goes. Both sides are stripped of whitespace before the search, so a reformat
keeps the claim and a rewrite breaks it.

Break-tests: browser.mjs's offer comparison rewritten (fires — the case the
field-name token missed), `c.compose.offered` renamed in pin.swift (fires, and
needs the word boundary: `offeredX` contains `offered`), and the same comparison
re-wrapped over four lines (control: passes).

What it is worth, stated: the harness still SPELLS that comparison, not that the
comparison still holds a measurement to the number — the claim a search can
support, and the same one wasm/verify's citation walk makes about a `check N`.

## Item 6 — a decision, rather than the absence of a branch

`citationSenses` held three sentences the check read only to paste into a
failure it was going to produce anyway. "All three senses fail" was therefore
not a row anybody could point at; it was the absence of an `if`. A row that
changed its mind had nowhere to say so.

`citationSense{Fails, Act}` is that branch as data. Every row is still `true`,
for the reason argued there, and the check consults it. And a guard, because a
field that can say "do not fail" needs one: `citationSenseVerdict` asks whether
any sense THIS RUN actually met can fail. Asked of the enumeration rather than
of the table, which is the sharper question — on a machine with no git every
file comes back `senseOnDisk`, so one row flipped to false would empty this
check on exactly the machines with the weakest enumeration, while the table went
on looking like two thirds of a check.

Break-tests: all three rows flipped to false (fires — "every file this
enumeration reached came back as one of [tracked], and citationSenses marks none
of those as failing"), and a sense with no row (the classification arm, still
firing by name).

## Item 7 — one palette at two type scales, held to being that

`bandRenderThemes` derives a theme from `DefaultTheme` with its caption tier
changed "and nothing else". The size and the leading were both held to
differing; nothing held the rest to not. The bands read captions, so a theme
that also moved a colour would vary two things through checks that cannot tell
them apart, and every failure would name whichever of the two the check happened
to be looking at.

`themeDifferences` walks both themes reflectively — structs recursed, everything
else compared whole — and gen.go refuses a difference outside
`Typography.Caption`. A tier or a role added to `core.Theme` is covered without
anybody remembering, which is the point: a field this file has never heard of is
exactly the kind that gets copied silently.

Break-tests: `Colors.Surface` also changed (refused by name), `Typography.Body`
also changed (refused by name), and the walk stubbed to return nothing (refused
by the guard that the departure it permits was actually found — "the walk that
says this theme varies nothing but its caption tier is reporting that it varies
nothing, which is not the same claim").

## The break-tests

**18 run: 15 that had to fail and did, 3 controls.**

    item 1   a transform on the probe Row; a transform on one band          2
    item 2   the run rect 6px wide; the run rect 6px early                  2
             the run rect 6px short (control: surplusInk's arm fires)       1
    item 3   INK_ROWS at 0.02; the floor at 2 (control for the floor)       2
    item 4   the window at 665; at 585 with the fold guard off              2
             the window at 665 with the guard off (control: PASS)           1
    item 5   the offer comparison rewritten; compose.offered renamed        2
             the gap comparison re-wrapped (control)                        1
    item 6   three senses flipped to non-failing; a sense with no row       2
    item 7   a colour also changed; a Body tier also changed;
             themeDifferences stubbed out                                   3

The item 4 control is this session's item 4 control from last session: the
defect, under the guard that was missing, is a clean pass.

## Verification

Ten paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      replay + 20 mjs suites + BROWSER PASS (12 checks,
                            20 real bands over five shapes and four palettes, on
                            three rows taken as fractions of each element's own
                            measured ink band and clear of both its edges, over
                            a run rect the paint reaches the ends of, behind 11
                            antialiasing probes — every one of them and every
                            box they answer for under an ancestry that decides
                            nothing about how a glyph is drawn)
    android ./gradlew compileDebugKotlin --offline
    android ./gradlew :app:fetchComposeLayoutSources --offline
    GOOS=js GOARCH=wasm go build ./...

No new files. Five changed.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

1. **(age 0 · value medium) The ancestry is held to being neutral and the
   probe's own box is held to nothing.** `INK_PATH_PROPS` sweeps strict
   ancestors on both sides, on the argument that the element's own declaration
   travels with the Style copy. It travels as `core.Style` — and what a browser
   resolves is CSS. A `Rotate` copies; a `will-change` or a
   `-webkit-font-smoothing` that arrived through a stylesheet, a UA default or
   a future runtime mapping would sit on the element itself, on both sides or
   on neither, and nothing compares the probe's computed values with the
   scanned box's. The chain above is measured; the two elements are assumed
   equal because one was built from the other's fields.
2. **(age 0 · value medium) `inkExtent` reads the columns that are not the
   fill, and calls them the words.** The extent is what bounds the run rect,
   and it is taken over the label's whole box on the same three rows — so
   anything else painted in that box at those rows moves it. `surplusInk`
   fires on exactly that, which is why the pair is sound today; the ordering is
   an accident of two checks rather than a stated dependency, and a surplus
   that was ever downgraded to a warning would leave this one measuring a
   different thing without saying so.
3. **(age 0 · value low) `INK_EDGE_CLEARANCE` is 1 because two would fail the
   grid.** That is a measurement and it is also a floor chosen by what the
   fixture happens to contain — the argument for 1 is "a row must be off the
   edge", which is a statement about a rounding nobody has measured the
   magnitude of. What would settle it is the thing item 3 did for the
   fractions: read the pixel at the band's own edge and see how much of it is
   glyph.
4. **(age 0 · value low) The probe Row's fold is checked per probe and the
   grid's own bottom is checked per band.** Both guards ask about the thing
   they are about to sample, which is right, and nothing asks about the grid as
   a whole. A viewport that clipped the LAST band would fire twenty-one
   messages — one per probe that band's declarations are shared with, plus its
   own — and the reader would be looking at the probes.
5. **(age 0 · value low) `pinConsumers` names two harnesses and the phrases are
   one per consumer per number.** Sixteen phrases for eight numbers, each an
   exact quotation of a comparison, and the failure for a reworded assertion is
   the same failure as for a deleted one. The two are distinguishable — a
   deleted assertion leaves the field name behind too, a reworded one usually
   does not move it — and nothing here tells them apart.
6. **(age 0 · value low) `citationSense.Fails` is a field with one value.** All
   three rows are `true` and `citationSenseVerdict` refuses a run in which none
   of the senses present can fail. Both arms of the branch exist and only one
   has ever executed; the `t.Logf` arm is reachable by editing a row, which is
   how it was tested, and no fixture holds it.
7. **(age 0 · value low) `themeDifferences` permits a prefix and the prefix is
   a string.** `bandRenderTallVaries` is `"Typography.Caption."`, matched
   against paths the same function produces — so the two agree by construction
   and a typo in the constant makes every caption difference "off" and fails
   loudly. What it cannot say is which FIELDS of the caption tier are allowed
   to move: the theme varies `FontSize` and `LineHeight`, and a third caption
   field changed would pass here while the comment beside it goes on naming
   two.
