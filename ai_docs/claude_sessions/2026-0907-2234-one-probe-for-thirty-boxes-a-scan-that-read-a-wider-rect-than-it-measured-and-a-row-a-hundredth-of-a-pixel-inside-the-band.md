# Session: one probe for thirty boxes, a scan that read a wider rect than it measured, and a row a hundredth of a pixel inside the band

Session: https://claude.ai/code/session_017k47JkhG1WBZR5nP29c7Ge
Date: 2026-09-07 (follows "a-row-that-was-outside-the-band-a-count-nobody-read-as-digits-and-two-tolerances-for-one-fixture")

## Ask

"Do all in the next list" — all nine, the age-3 item included.

The last three sessions have been walking the same move outward. Two sessions
ago it was **an enumeration describing today's members rather than the space
they come from**; last session it was **a number held to nothing on the other
side**. This one is **a measurement standing in for measurements it was not
taken of**: one antialiasing probe licensing thirty boxes, a vertical band
measured for a run and a horizontal sweep taken of the element, a floor derived
over every number in a fixture rather than over the ones a check reads, three
palettes that vary a colour and never a metric.

| # | age·value | item |
|---|---|---|
| 1 | 0·medium | The probe answers for the page and the scan trusts it for every band |
| 2 | 0·medium | The band is measured for the element and the scan reads its whole width |
| 3 | 0·medium | `Case.Resolution` is the closest two numbers, not the closest two a check tells apart |
| 4 | 0·low | The convention arms range over results and the probe parameters are one list |
| 5 | 0·low | The indented shape moves the label and nothing states what it moves it to |
| 6 | 0·low | `citationExempt` classifies files and nothing asks git whether the path exists |
| 7 | 0·low | The two senses reach a failure message and no assertion |
| 8 | 0·low | `SUBPIXEL_EPSILON` is 1 and the probe measures 0 |
| 9 | 3·low | Five shapes and every theme paints them the same way |

## Item 9 first, because it found the live one

The three bundled themes all set `Typography.Caption` to a normal weight at
twelve or thirteen points with no line height of their own. So twenty bands over
five shapes were still one type scale, and the ink scan — three rows placed
relative to a font's metrics inside a line box — had only ever been asked one
question about metrics.

The grid grew a fourth palette: a bundled theme with its caption tier changed
and nothing else, both size and leading, because they move the band by different
mechanisms (a size scales the glyphs and the x-height with them; leading moves
the box around a band it does not touch). gen.go refuses a derived theme that
has stopped differing in either.

It found this, on the first run:

```
    face                     box      x-height band, as fractions of the box
    AmberTheme, Default      15px     0.401 to 0.800
    MaterialTheme            14px     0.391 to 0.786
    Default + tall caption   29px     0.388 to 0.690   ← INK_ROWS' 0.7 is outside
```

It passed. 0.7 of 29px is 20.3, the band ends at 20.01, the device row is 20 —
**inside by a hundredth of a pixel**, which is the same margin, on the same
check, that last session found for 0.4 and moved the fractions to fix.

Twice is a coordinate system, not a coincidence. The rows have to be inside the
band the run has ink in; the band's position inside a line box is a relationship
between leading and a font's metrics; a fraction of the BOX can only ever be
checked against that afterwards. So the fractions are now of the measured band —
`[0.25, 0.5, 0.75]` of baseline-to-x-height, read off the element's own inline
text box and the canvas metrics for its resolved face. Inside by construction,
at every line-height and in every face.

`inkBandFault` goes with the coordinate system that made it possible. What it
checked is now impossible; what remains is the band being unusable, and
`inkBandRows` returns `{rows}` or `{problem}` for the four ways it can be: the
metrics never arrived, the canvas would not take the resolved font, the run
wrapped, or the band is too short for three fractions to be three device rows.

And one new guard, because the guarantee is now structural and a structural
guarantee is only as good as the constant that defines it: `inkRowsFault` holds
INK_ROWS to being strictly inside (0, 1), strictly increasing, and more than one
entry. A fraction of 1.3 puts a row below the baseline and **nothing downstream
would say so** — the scan's three verdicts are ORs across the rows, so a row
reading backdrop costs the redundancy the three exist for and fails nothing.

Break-tests: `[0.25, 0.5, 1.30]`, `[0.25, 0.25, 0.75]`, `[0.5]`, and a caption
of 4px (the band comes back 1.84px tall and two rows land on the same pixel).

## Item 1 — one probe, thirty boxes

`SUBPIXEL_PROBE` was black words on a white page: one size, one weight, one box.
What it licensed was a linearity claim about every label and every count in the
grid — colours it did not paint, weights it did not set, whatever alpha a
theme's ink carries. Chrome picks a text rendering path per element.

gen.go now emits one probe per distinct text DECLARATION the scan reads: the
scanned node's own `core.Style`, copied whole so a field nothing here has
thought about travels with it, over a white box, with the ink replaced by black
at the same alpha. Black at any alpha over white is still a grey, so the verdict
stays the one a screenshot can state without begging the question — asking the
probe in the band's own colours would be asking `offSegment`, which is the thing
this exists to license.

```
    28 scanned boxes (20 labels, 8 counts)  →  11 probes
    a coloured probe suppresses the boxes that share its declaration,
    and no others
```

They ride in one Row rather than one per line, because the grid is a fold budget
and eleven boxes would cost eleven bands' worth of height. Each declares its
width with `core.ShrinkNone` and the browser holds the rendered rect to it: a
Row that ran out of room would squeeze them into slivers, and a probe read
across two device pixels reports a grey page for want of pixels.

Break-tests: the probe painted in `#FF0000` (each probe fires, naming its own
declarations, and the bands it answers for are suppressed rather than reporting
third colours), and a band whose transcript names no probe.

## Item 2 — the vertical measurement and the horizontal sweep were two rects

`inkBandFault` took the inline text box's own top and the resolved face's
x-height — the RUN. The scan swept `label.x` to `label.x + label.w`, which on
the stretched branches is three times the width of the words. Every column past
the run is backdrop: `notFill` false, `t = 0` on the segment, nothing either way.

Narrowing to the run alone would have been the other fault — measuring where the
glyphs are with a number taken from where the glyphs are. So the box is not
dropped, it is SAID: the run is scanned for the three verdicts, and the surplus
on either side is held to being nothing but the band's own fill. That is a claim
neither half made before, and it is the one that catches a second thing painted
in the label's rect but outside its words.

`INK_RUN_GUTTER` is measured rather than assumed, which is the same discipline
item 8 is about:

```
    gutter 0    five of the twenty bands report the pixel at the run's own
                trailing edge — Amber's plain band ends its words at x=99.48
                and the device pixel spanning 99 to 100 is half a stem and half
                backdrop, belonging to neither window
    gutter 1    every band clear
    gutter 2    that, with a pixel of margin, against a surplus 180px wide
```

Break-tests: the gutter at 0 (fires, naming the boundary pixel), and the keep
window narrowed 6px inside the run (fires on a glyph at x=26.00).

## Item 3 — a floor derived over a superset

`resolution` ranged over every number in a case: offer, gap, both columns, every
base, every measured extent. The argument was a sentence — "every number in the
case is a number some assertion on some target compares a measurement against" —
and it is a claim about two files in two other languages, made in a package that
reads neither. True today, and true by nobody's decision tomorrow.

`caseNumbers` names each number and the assertion that holds a measurement to
it. The floor ranges over the read ones, so an unread number does not tighten a
bound two harnesses share. Today every number is read and the floor is the same
4 it was; what changed is that it is derived from a statement.

The coverage is held to the struct reflectively, so a field added to `Case`,
`Child` or `Measured` fails until somebody decides. Both directions: a field
with no reading, and a reading with no field.

Break-tests: the `Compose.Gaps` reading dropped (fires by value), a reading
invented from nothing (fires, and the resolution test fires with it), and a new
`Natural int` on `Case` (fires by field name).

## Item 4 — the sharpest control of the session

`conventionProbeParams` was `a int, b string, c []byte`, "one of each kind" —
which is the same shape as the three arms that test exists to close. Now both
halves range over the closed space: every `bindableGoTypes` row and every bound
interface alone, plus all of them at once (a rule that fires on a COMBINATION is
invisible to any list of singletons).

The break-test is `swiftResults` growing a rule that reads an `int8` parameter,
and the control is the same broken rule under the old hand-picked triple:

```
    the enumeration     15 result lists × 16 parameter lists — fires by name,
                        on `(p12 int8)` alone
    the old triple      ok  github.com/rohanthewiz/grmob/mobile/verify  0.193s
```

## Item 5 — a shape that moved the label, and nothing said where to

The indented band was in the grid to give the ink scan a rect the other four
never produce, and every assertion over it was one they also make. A
`ControlStyle` that stopped reaching the control would render the theme's own
inset, and the shape would pass while exercising nothing.

Two halves, on the two sides:

```
    gen.go     holds the rendered control padding to the number the BUILDER
               declared (reading it back would be two readings of one side,
               agreeing), and refuses an indent equal to what the other shapes
               render — at 16 the "somewhere else" is the same rect
    browser    holds the control's FIRST child's leading edge to that padding.
               The first child, not the label: the disclosure puts a chevron in
               front of the words, and `label.x - control.x` there is a padding
               plus a glyph plus a gap, only one of which is declared anywhere
```

The tap target's own leading edge is still asserted separately and must not
move. That pair is what makes `ControlStyle` safe to hand a caller.

Break-tests: `bandRenderIndent = 16` (gen.go refuses), the `ControlStyle` prop
deleted (gen.go refuses, naming the declaration), and `Leading` pointed at the
wrong child (fires in every theme, naming which of the two insets it is).

## Items 6, 7 and 8 — three small ones

**An exemption that outlived its path.** `citationExempt` was held to still
carrying a citation and never to still existing. A renamed file failed with "no
longer cites a check", which sends the reader to look at an exemption when what
happened was a rename. `citingFiles` now returns what it considered, and the two
failures are distinct — confirmed both ways: a renamed key gets the rename
message, and a real file that stopped citing gets the old one.

**A sense that reached a message and no assertion.** Every sense the enumeration
produces must have a row in `citationSenses`, and the row's sentence is what a
failure carries. The decision written down is that all three fail — an untracked
citation is one about to be committed, and the walk's third sense is held to the
same rule where git could not answer. Break-test: a fourth sense
(`"in the index"`) fails by name.

**A tolerance whose whole value was margin.** `SUBPIXEL_EPSILON` is 1 to absorb
"the browser's eight-bit rounding", and the three channels of a grey enter that
rounding as one value and leave as one. `SUBPIXEL_FLOOR` is 0, asserted
separately: above the ceiling is subpixel antialiasing, and between the two is a
page that is still grey and an explanation that is not. The control —
`SUBPIXEL_FLOOR = -1` — reports every probe at `#FFFFFF`, **0 channels apart**.

## The window

The grid went from 15 bands to 20 and grew a probe row, and `foldVerdict` named
the knob: `--window-size=800,600` → `800,900`. Raised rather than the grid
trimmed — the grid holds the shapes and palettes the ink scan has to be right
about, and the viewport is the arbitrary half of that pair. The scroll control
is unaffected: its subject is a 4000px filler.

## The break-tests

**21 run: 19 that had to fail and did, 2 controls.**

    item 1      the probe repainted in #FF0000                                1
                a band whose transcript names no probe                        1
    item 2      INK_RUN_GUTTER 0, and the keep window inside the words        2
    item 3      a dropped reading, an invented reading, a new Case field      3
    item 4      swiftResults reading an int8 parameter                        1
                the same rule under the old fixed triple (control)            1
    item 5      indent = the theme's inset, ControlStyle deleted,
                Leading pointed at the wrong child                            3
    item 6      a renamed exempt path                                         1
                an exempt file that exists and stopped citing (control)       1
    item 7      a fourth sense; an untracked citation carrying its sentence   2
    item 8      SUBPIXEL_FLOOR = -1 — every probe measures exactly 0          1
    item 9      three bad INK_ROWS, a 4px caption, a tall theme that
                stopped differing                                             5

The item 4 control is the one that made the case: the same defect, under the
list somebody picked by hand, is a clean pass.

## Verification

Ten paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      replay + 20 mjs suites + BROWSER PASS (12 checks,
                            20 real bands over five shapes and four palettes,
                            their words and their counts' digits scanned on
                            three rows taken as fractions of each element's own
                            measured ink band, over the run's own rect with the
                            rest of the box held to the backdrop, behind 11
                            antialiasing probes — one per text declaration any
                            of it reads)
    android ./gradlew compileDebugKotlin --offline
    android ./gradlew :app:fetchComposeLayoutSources --offline
    GOOS=js GOARCH=wasm go build ./...

No new files. Seven changed.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

1. **(age 0 · value medium) A probe reproduces a declaration and cannot
   reproduce an ancestor.** `inkProbeFor` copies the scanned node's own Style
   onto a white box, which covers everything the element declares about itself.
   What decides a text rendering path is not only that: a transform, an opacity,
   a `will-change` or a stacking context ANYWHERE above the band moves its
   glyphs onto another path and leaves the probe on the old one. The gap is
   smaller than one probe for the page and it is the same shape, one level up —
   and nothing in the grid states that the bands' ancestry and the probes' are
   the same.
2. **(age 0 · value medium) The surplus is held to the fill and the run is held
   to the segment, and neither is held to the run being where the words are.**
   `surplusInk` asserts the box outside the run is backdrop and `scanInk`
   asserts the run holds ink; a run rect that reported the wrong PLACE — half
   the words, or a box beside them — satisfies both, because the ink it finds
   and the backdrop it finds are each consistent with the other rect. The
   client rect is the browser's own answer and there is nothing behind it here.
3. **(age 0 · value low) `INK_ROWS` is three fractions and the margin at each
   end of the band is a quarter by assertion, not by measurement.** The
   fractions are inside (0, 1) and increasing, which is checked. Whether a
   quarter of an x-height is enough clearance from the baseline and the
   x-height line — where a rounding lands on a horizontal edge of every glyph
   at once — is the old "most likely" argument in a new coordinate system. The
   grid could measure how far the outermost rows actually sit from the two
   edges, the way the gutter now does.
4. **(age 0 · value low) The probe Row's own fold is the grid's, and only the
   bands are checked against it.** `foldVerdict` runs per band; the probe row
   mounts after all twenty and is checked only for width. A viewport that fit
   every band and clipped the probes would report eleven probes "not laid out"
   — which is the right failure with the wrong cause attached, and the fold
   message is the one that names the knob.
5. **(age 0 · value low) `caseNumbers` names the assertion that reads each
   number in prose.** The strings are what a reader gets and nothing holds them
   to the harnesses they describe: an assertion deleted from pin.swift or
   browser.mjs leaves a row here saying it is still read, and the floor stays
   tightened for a distinction nobody makes any more. The coverage between the
   table and the STRUCT is checked in both directions; the coverage between the
   table and the two consumers is prose.
6. **(age 0 · value low) `citationSenses` decides three senses the same way and
   the sameness is the decision.** All three fail, which is written down and
   argued. Nothing distinguishes them downstream, so the table's real content
   is its coverage — a fourth sense must be classified — and a row that changed
   its mind (an untracked citation as a warning, say) has nowhere to say so
   without a second field nothing reads.
7. **(age 0 · value low) The derived theme changes a caption tier and every
   other tier is still one scale.** `bandRenderThemes` varies
   `Typography.Caption` because the band's label and count are captions. A
   theme with a different Body or Title changes nothing in this grid, which is
   correct — and it means "four palettes" is three palettes and one caption,
   and the widget grid beside it still sees three of everything.
