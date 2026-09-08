# Session: a copy that was not a computed value, an extent that trusted its neighbour, and a permission wider than the claim

Session: https://claude.ai/code/session_01MQcEV7mAeEV4h1LywCdQhe
Date: 2026-09-07 (follows "a-probe-that-could-not-see-its-own-ancestors-a-rect-nothing-stood-behind-and-a-quarter-nobody-measured")

## Ask

"We were working items in the Next list when the app was accidentally quit.
Please continue" — all seven, every one of them raised last session.

The previous run died mid-break-test. `wasm/verify/browser.mjs` was the only
dirty file, carrying items 1–4 implemented and four injections still in it: a
`boxShadow` painted into a label's box, `runW: rects[0].width + 6`,
`INK_ROW_ROUNDING = 0`, and an `if (true)` where the finished code says `else`.
Those were restored first — the constant to 0.15, the value its own comment
argues for — and then everything was break-tested from scratch, because an
injection nobody remembers running is a check nobody has seen fire.

| # | age·value | item |
|---|---|---|
| 1 | 1·medium | The ancestry is neutral on both sides and the two elements are assumed equal |
| 2 | 1·medium | `inkExtent` calls the not-fill columns "the words" because `surplusInk` ran first |
| 3 | 1·low | `INK_EDGE_CLEARANCE` is 1 because two would fail the grid |
| 4 | 1·low | Both fold guards ask about the box they are about to sample; nothing asks about the grid |
| 5 | 1·low | A reworded assertion and a deleted one are the same failure |
| 6 | 1·low | `citationSense.Fails` is a field with one value and an arm no fixture holds |
| 7 | 1·low | `themeDifferences` permits a prefix, and the caption tier is a whole `core.Style` |

## Item 1 — a Style travels; a computed value does not

`inkProbeFor` copies the scanned node's `core.Style`, and last session held every
strict ancestor of both sides to `INK_PATH_PROPS`. What that argument still
assumed is the pair itself: a `Rotate` copies because it is in the Style, and a
`-webkit-font-smoothing` or a `will-change` arriving through a stylesheet, a UA
default or a future runtime mapping sits on the ELEMENT and in no Style at all.
It would land on one of the two and not the other with every ancestor neutral,
and nothing compared them.

So both elements' own computed values are read for the same twelve properties
and compared leaf by leaf. `inkOwnFault` names the first that differs, both
sides' values, and how many more.

Break-test: `webkitFontSmoothing = 'antialiased'` set on band 0's label and count
only — fires on both boxes, naming the property, the box's `antialiased` and the
probe's `auto`, and listing the probes those declarations are shared with.

## Item 2 — the check below is not asked of this box

`inkExtent` reads the columns of the label's rect that are not the fill and calls
the first and last of them the ends of the words. That is only the words while
nothing else is painted there on those rows — which is exactly what `surplusInk`
has just held. The pair was sound today and sound by the ORDER of two checks,
with nothing saying so.

It is an `else` now, and the surplus failure carries the sentence: whatever this
pixel belongs to would move the extent, and the comparison would be about a
different quantity.

The control is the sharper half of this item. With the rect widened 6px AND a
`inset -3px 0 0 0 #000000` shadow painted at the right edge of one label's box:

```
    else (shipped)   19 extent failures. The surplused band reports the surplus
                     and says the run-rect question was not asked of it.
    if (true)        19 extent failures. The surplused band's rect is 6px too
                     wide, the shadow has dragged its extent's `to` out to
                     x=279, and the check that exists for that defect passes in
                     silence.
```

Same count, opposite meanings: the ordering accident does not make the extent
fire wrongly, it makes it MISS. That is the false negative the dependency was
hiding, and it is only visible with both injections in at once.

Break-tests: `runW + 6` alone (all 20 fire — the else arm still runs with no
surplus), `runW + 6` with the shadow (19, the surplused band suppressed by name),
and the same pair with the dependency removed (19, the defect silently passed).

## Item 3 — the rounding, measured, and a bound with a number on each side

`INK_EDGE_CLEARANCE` is 1 because 2 fails fifteen of the twenty-eight boxes.
That is a measurement of the fixture and it was never a measurement of the thing
the clearance is about — how much ink a device row of movement costs.

`inkRowCoverage` reads it: the fraction of a scanned row's own window that is not
the backdrop, asked one device row either side of every row the scan reads.

```
    the row in question             coverage one device row out, at worst
    the three the scan reads        0.239   (DefaultTheme, the long title)
    a row on the BASELINE           0.100   — and 0.000 in every count pill
    a row on the X-HEIGHT line      0.348   — and 0.000 in two count pills
```

`INK_ROW_ROUNDING = 0.15` sits in the gap, with the wider margin on the side that
has measurements from more than one face. Stated plainly in the comment: this
separates the baseline end and separates NOTHING at the x-height end, where
ascenders put the outward neighbour anywhere between 0.000 and 0.348. What rules
that row out is the structural argument, not this floor.

And the floor is held from the other side too, once per run: if the row
immediately below the band clears `INK_ROW_ROUNDING` in all 28 boxes, the first
bracket has gone and the rows would pass wherever they were put.

Break-tests: `INK_ROW_ROUNDING = 0` (the grid-wide guard fires — "the rows would
pass wherever they were put"), and `INK_EDGE_CLEARANCE = 0` with `INK_ROWS` at
`[0.25, 0.5, 0.999]` (fires on the baseline row, 8.0% for a label and 0.0% for a
count pill).

## Item 4 — one message, because the grid is one Column

Every band guards its own bottom and every probe guards its own; the grid they
are all children of was asked about by nobody. A window short enough to clip the
last band clips a run of them at once, and the reader gets a wall.

`root` is read with the rects now, `foldVerdict` is asked of it first, and
`gridClipped` suppresses every pixel read below. The layout assertions still run,
because a rect is the same whether or not the capture reached it.

```
    the guard on    1 message, naming the knob
    the guard off  15 messages, the first fourteen naming probes
```

Break-tests: `--window-size=800,665` (one message), and the same window with the
grid guard disabled (fifteen).

## Item 5 — two edits, two failures

`caseNumbers` quotes an assertion per consumer per number, and a phrase that has
gone missing has two quite different causes: the check was DELETED (the row here
is stale and `resolution` is deriving a floor over a distinction nobody makes),
or it was REWORDED (nothing is stale and the fix is to requote). Both arrived as
one failure.

They separate on the field: deleting an assertion takes the fixture's field with
it, rewording one almost never does. So `pinCite` carries both, the field is
asked only after the phrase — it is the weaker claim, because browser.mjs mounts
`internal/bandfixture` whose cases spell `c.offer` too — and `Field` must appear
in `Phrase`, or the second question would be answered by a token the row has no
claim on.

pin.swift's eight citations were bare fields, on the argument that the file is
the pin check and nothing else. That made the second question vacuous on that
half: a phrase that IS the field cannot go missing while the field stays. All
eight now quote the `abs(…) > pinEpsilon` they are actually spelled with.

That broke `pinStripSpace`, and finding out why was the real work of this item.
It removed every space, which glues neighbouring tokens: `where abs(css[i]`
became `…countwhereabs(css[i]`, and `pinSpells`' leading `\b` — there to stop
`pinSame(…)` matching inside `myPinSame(…)` — then rejected a phrase that was
present. Every phrase the table happened to carry began just after a bracket or
a `!`, so the boundary had been holding by luck. A space now survives only
between two word characters, which is exactly where it carries meaning.

Break-tests: `pinSame(total, c.offer)` renamed in browser.mjs (REWORDED, "it does
still spell c.offer" — and that token is bandfixture's, which is the point),
`c.compose.mains` renamed everywhere in browser.mjs (DELETED, "contains neither"),
a field moved outside its phrase (the substring guard), and pin.swift's
`abs(css[i] - c.css[i]) > pinEpsilon` wrapped over four lines (control: passes).

## Item 6 — an arm whose only fixture was somebody's uncommitted edit

`citationSenses` decides whether a stale `check N` is an error or something the
reader is merely told. Every row says error, so the `t.Logf` arm had never run,
and the way it was tested was by editing a row.

`citationVerdict` is that decision as a function of values — the sentence, and
whether reporting it fails — and the loop's only remaining job is choosing which
of the testing package's two reporters carries it.
`TestCitationVerdictsAreDecidedByValues` holds both arms of it and both arms of
`citationSenseVerdict`, including the states no repository can produce: a sense
with no row, and no senses at all. Same move as fold.mjs's `foldVerdict`.

It asserts what the code would do if the table had a false in it, not that the
shipped table has one. That is the half that stops being true silently.

Break-tests: `act.Fails` replaced by `true` in the verdict (the reported-only row
fires), and `len(can) > 0` replaced by `len(present) > 0` in
`citationSenseVerdict` (the unclassified-sense row fires).

A note on the harness, not the code: the first attempt backed the file up as
`checknumbering_test.go.bak` beside the original, and the citation walk read
every `check N` in the copy. The backup lives outside the repository now.

## Item 7 — a permission as narrow as the claim

`bandRenderTallVaries` was `"Typography.Caption."`, matched with `HasPrefix`
against paths the same file produces — so the two agreed by construction and a
typo failed loudly. What it could not say is which caption fields may move.
`core.Theme`'s `Caption` is a `core.Style`: FontSize, FontWeight, TextColor,
Background, Padding, BorderRadius and twenty more. The theme moves two. A third
changed alongside them satisfied the prefix perfectly while the comment beside
the derivation went on naming two.

It is the exact set of leaves now, held in both directions: nothing outside it
may differ, and everything in it must. The second direction is its own fault — a
permission nothing uses is a list with slack in it, and "differs only where
permitted" is satisfied by differing nowhere.

Break-tests: `Caption.FontWeight` also changed (fires by name — the case a prefix
cannot see), a third path added to the list that nothing moves (fires as an
unused permission), and `Colors.Surface` also changed (control: the case the
prefix already caught, still caught).

## The break-tests

**18 run: 15 that had to fail and did, 3 controls.**

    item 1   font-smoothing on the scanned boxes and not their probes         1
    item 2   the run rect 6px wide; 6px wide with a surplus in the box        2
             the same pair with the dependency removed (control: the
             defect passes in silence — the false negative it was hiding)     1
    item 3   INK_ROW_ROUNDING at 0; a scanned row on the baseline             2
    item 4   the window at 665; the same window, grid guard off               2
    item 5   the offer comparison renamed; c.compose.mains renamed away;
             a field outside its phrase                                       3
             pin.swift's css comparison wrapped over four lines (control)      1
    item 6   act.Fails forced true; the guard asked of len(present)           2
    item 7   Caption.FontWeight also changed; an unused permission            2
             Colors.Surface also changed (control: the prefix's own case)      1

Item 2's control is the unusual one: it does not turn a failure into a pass by
removing a guard, it turns a REPORTED defect into an unreported one. The count of
failures is identical on both sides and only the surplused band's message tells
them apart.

## Verification

Ten paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      replay + 20 mjs suites + BROWSER PASS (12 checks,
                            20 real bands over five shapes and four palettes,
                            on three rows far enough off both edges of their own
                            measured ink band that a device row of rounding
                            leaves them on the ink, over a run rect the paint
                            reaches the ends of, behind 11 antialiasing probes
                            resolving the same twelve rendering-path properties
                            as the boxes they answer for)
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

1. **(age 0 · value medium) `INK_PATH_PROPS` is twelve properties chosen by
   argument, and the comparison is now exact.** Both sides' computed values are
   read and compared leaf by leaf, so the check is only as wide as the list —
   and the list was assembled from what is known to composite a subtree or
   change a glyph's rendering mode. A property added to CSS, or one already
   there that nobody wrote down (`font-synthesis`, `font-variation-settings`,
   `text-size-adjust`), sits on one side and not the other and this says
   nothing. What would settle it is the opposite question: read EVERY computed
   property on both elements, and list the ones allowed to differ — which is
   item 7's move, applied to a much larger struct that a browser owns rather
   than this repository.
2. **(age 0 · value medium) Item 2's control is the shape of a whole class of
   failure nobody looks for.** A guard removed usually turns a pass into a
   failure, and the break-test that proves it fires is easy. The surplus/extent
   pair fails the other way: the defect is present, the check runs, and it
   passes because something else moved the quantity it measures. Nothing in
   this harness systematically asks "which of these checks can be made to pass
   BY a defect", and the answer was found here by injecting two things at once
   on a hunch.
3. **(age 0 · value low) `INK_ROW_ROUNDING` separates one end of the band and
   the comment says so.** 0.15 is above what a baseline row scores and below
   what the scan's rows score, and at the x-height end an ascender puts the
   outward neighbour anywhere in [0.000, 0.348] — so the floor is not a bound
   there at all. The row above the x-height line is ruled out by a structural
   argument that no number is attached to: it misses every round letter beside
   the ascender. That is measurable — the coverage of a row above the x-height
   over the SUBSET of columns that are not ascenders — and nobody has measured
   it.
4. **(age 0 · value low) `gridClipped` suppresses every pixel read and the
   layout assertions still run.** That is deliberate and stated: a rect is the
   same whether or not the capture reached it. What it means in practice is
   that a clipped run reports one message and a PASS for everything rect-shaped
   in the same breath, and the OK line at the end still recites twenty bands.
   A reader who skims the tail would not learn that half the pass was not
   asked.
5. **(age 0 · value low) `pinStripSpace` now keeps a space between two word
   characters, and that is a lexer's job done by a rule of thumb.** It is
   correct for Swift and JavaScript operators and it would be wrong for a
   language where whitespace inside an expression is significant in some other
   way, or for a phrase quoting a string literal with meaningful runs of
   spaces. No row does either today. The honest bound on this claim is that it
   canonicalises two specific files, and `pinConsumers` is the closed set that
   makes that true — nothing ties the normaliser to that set.
6. **(age 0 · value low) `citationVerdict` is a function of values and its
   caller still chooses the reporter.** `switch { case say == "": ; case fails:
   t.Error; default: t.Log }` is three lines in the loop and they are not
   covered by the fixture — a `default` that called `t.Error` would pass every
   test in this file, because no shipped row reaches it. The same gap the item
   closed, one level up, and it cannot be closed the same way without a fake
   `testing.TB`.
7. **(age 0 · value low) `bandRenderTallVaries` is two leaf paths and the
   derivation beside it is two assignments.** They agree today and they are two
   separate statements of one fact: adding a third assignment means remembering
   to add a third path, and the failure for forgetting is the good one (it
   fires by name). What it cannot do is derive one from the other — the list
   could be produced by diffing before and after the assignments, at which
   point it would agree by construction and stop being a claim, which is
   exactly the trade item 5's field-versus-phrase argument settles the other
   way.
