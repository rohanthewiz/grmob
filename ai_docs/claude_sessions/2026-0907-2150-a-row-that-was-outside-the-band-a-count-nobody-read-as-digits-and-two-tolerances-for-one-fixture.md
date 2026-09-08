# Session: a row that was outside the band, a count nobody read as digits, and two tolerances for one fixture

Session: https://claude.ai/code/session_017k47JkhG1WBZR5nP29c7Ge
Date: 2026-09-07 (follows "three-fractions-that-can-be-one-row-three-arms-that-can-be-four-and-a-walk-the-last-commit-never-carried")

## Ask

"Do all in the next list" — all ten, the two age-3 items included.

They share a shape with the last two sessions and it is one step further out.
Last session's was **an enumeration that describes today's members rather than
the space they come from**. This one is **a number or a claim held to nothing on
the other side**: a tolerance argued for on its own page with no statement of
what it had to be true of, a fraction of a box whose reason was a font's
metrics, a scan that read a pill and never the digits in it, a rendering mode
the whole ink scan rests on and no line names.

The fix is the same move each time: find the thing on the OTHER side of the
comparison — the fixture's own resolution, the face the browser resolved, git's
own exclude rules, the compiler's annotation — and hold the number to it.

| # | age·value | item |
|---|---|---|
| 1 | 3·low | `bandRenderBuilders` mounts three shapes and the scan's rect is the same rect nine times |
| 2 | 3·low | `pinEpsilon` and `pinSame` are two tolerances for one fixture |
| 3 | 0·medium | `offSegment`'s linearity is grayscale antialiasing's, and nothing says so |
| 4 | 0·medium | The other half of `INK_ROWS` is still a sentence |
| 5 | 0·medium | `errorConventionArms` names three symbols and nothing holds a symbol to its arm |
| 6 | 0·low | A band's count is read as a fill and never as digits |
| 7 | 0·low | The four redundant skip prefixes are redundant by assumption |
| 8 | 0·low | `repositoryFiles` returns tracked and untracked files as one set |
| 9 | 0·low | `offSegment` mixes two metrics |
| 10 | 0·low | The convention enumeration probes results and never parameters |

## Item 4 — the fractions were outside the band they were chosen for

The sharpest finding of the session, and it was live rather than latent.

`INK_ROWS` was `[0.4, 0.5, 0.6]` of the label's line box, and the reason the
rows read ink at all was one sentence: "they are all inside the x-height band —
an ascender or a descender would read backdrop where a lowercase word has
none." That is a relationship between a line box's height and a font's vertical
metrics, and this file chooses neither.

Measured against the caption face Chrome resolves here — bold 13px Times, box
15px, ascent 12, x-height 5.986 — the band runs from **39.01 to 45.00** in page
coordinates and the topmost row lands at **39.00**. Outside, by a hundredth of a
pixel, since the day it was written.

It read ink anyway, because this label carries capitals and digits. A band
titled in lower case would have been scanned by two rows and a row of backdrop,
and the failure would have said the words were not there.

So the band is measured rather than assumed, off the browser and not derived:
the inline text box is the font's content area, so its top is the baseline less
the ascent, and the canvas metrics for a probe glyph in the element's own
resolved font give the height. Two probes, because the two things scanned are
different kinds of content:

```
    a word    "x" — the shortest thing a word can be made of, so the band is
              the x-height and every letter beside it has ink across it
    a count   "0" — digits are lining figures, all of one height, so there is
              no shorter member to spoil the band
```

And the fractions moved. The x-height band is not centred on a line box's
middle — it runs from the baseline UP, and the baseline sits well below centre —
so `[0.5, 0.6, 0.7]` is the same tenth-of-a-box spacing centred on the band
instead of on the box: 0.40 to 0.80 of the box for this face, a tenth of margin
at each end.

The break-tests are the old fractions (every band fails naming 0.01px), a
line-height of 2.6 (3.00px below the band), and a title long enough to wrap —
which reports **two line boxes** by name, because a range over the text returns
one rect per line and three fractions of a two-line box are three rows of
nothing.

## Item 3 — the arithmetic under the arithmetic

`offSegment` holds every pixel in a text box to the line between the fill and
the ink, on the argument that a glyph pixel at coverage `c` is
`fill + c·(want − fill)` exactly. That is GRAYSCALE antialiasing's arithmetic.
LCD subpixel antialiasing gives each channel its own coverage — that is the
whole point of it — so an ordinary glyph edge gets a coloured fringe and lands
well off the line, and `INK_EPSILON`'s exact match on a stem's interior rests on
the same thing.

Headless Chrome disables it. No line of the file said so, and the failure would
have been fifteen bands each reporting a third colour inside a label — fifteen
messages about a palette, for a cause that is not a palette.

`SUBPIXEL_PROBE` is black words on a white page, mounted with the bands and read
off the same screenshot. Under grayscale antialiasing every blend of `#000000`
and `#FFFFFF` is a grey, so the three channels of every pixel in that box are
equal; a pixel whose channels differ is a subpixel-rendered one. `illlim illlim`
rather than a word: nothing but vertical strokes, which is the most edge per
pixel scanned.

One probe, one verdict, read before any band. The break-test is a magenta
`::first-letter` in the probe's box — **192 channels apart**.

## Item 9 — one metric, and it used to be two

The projection was Euclidean and the distance reported off it was max-channel,
which is `channelDistance`'s metric and the one every tolerance in the file is
expressed in. So the `t` it picked minimised a quantity nothing compares against,
and the number handed to `OFF_SEGMENT_EPSILON` was not the number the projection
had minimised.

Each channel's signed error is affine in `t`, the max of their absolute values is
convex and piecewise linear, and the minimum of such a function over `[0, 1]` is
at an endpoint or where two of its six pieces cross — seventeen candidates,
enumerated exactly, rather than a search.

Checked against a 20,000-step scan over 200,000 random triples: **zero
mismatches**. And the two metrics really do differ:

```
    worst disagreement over random triples          56.9 channels
    pixels the old metric would fail at ɛ=3 whose
    true max-channel distance is under it           found in the first 500k
    the fifteen bands' worst, old / new             1.2 / 0.9
```

## Item 6 — a count read as a colour and never as digits

"A band is a fill, a run of words and a count" is the argument the ink scan was
added for, and only two thirds of it was ever read. The label got a scan; the
count got the same single point sample the band's fill gets, taken inside the
pill's own leading padding — which is a run of fill with no digit in it by
construction. **A pill that painted itself perfectly and rendered no number
passed**, which is exactly the state the label was in before any of this existed.

The digits are scanned the way the words are, and the pill's own two paddings
say where. The pill's radius is 999, so its leading and trailing edges are the
apexes of a curve and every pixel there is a blend with the BAND behind it —
a third colour to the segment test, and the one place in this scan where that
would be correct painting rather than a fault. Between the two paddings is the
digits' own box, and it is nothing but fill and digit.

gen.go carries `BadgeInk` and `BadgePadRight` and refuses the cases that would
make the scan meaningless: a pill with no declared ink, an ink equal to its own
fill, a trailing padding too small to leave a window.

```
    the digits made transparent      "the number is not there"
    the digits in another ink        the pixel furthest from the pill named
    the window widened past the
    paddings                         180.3 channels off the line — the curve
```

## Items 1 and 5 — two pairings that were about existence

**The grid mounted three shapes and the scan read one rect nine times.** The
count-hidden band existed on the disclosure branch only, so the parity
comparison — "a caller who adds `OnToggle` gets a control and not a relayout" —
was a claim about badged bands, and the other shape's two branches were never
compared. Five shapes now, fifteen bands, and the pairing is a field rather than
an inference from two booleans:

```
    a plain band                               pair "badged"
    a disclosure band                          pair "badged"
    a plain band, count hidden                 pair "unbadged"     new
    a disclosure band, count hidden            pair "unbadged"
    a plain band, indented, long title, no
    count                                      no pair             new
```

The last exists to put the label's rect where no other shape puts it — a
caller's own indent through `ControlStyle` (which must not move the tap target),
no badge, and a title long enough that the words are most of the band. gen.go
refuses a pair with one half; the browser refuses a pair that did not run.
Break-tests: a pair with one branch (gen.go fatals by name), and a trailing
inset on the plain count-hidden band, which fires the **unbadged** pair in every
theme — the comparison that did not exist this morning.

**And three symbols nobody held to an arm.** Both directions of
`errorConventionArms`' pairing were about EXISTENCE: a row with no function
failed and a function with no row failed. Swap the names of two rows and both
still pass — three rows, three functions, every one present — while the compiler
settles the arms under the wrong descriptions.

So the arm is read off the annotation, which meant giving all three arms one
form. `importer.swift` had two: a method reference for two of them, and a call
with a value annotation for the third, on the argument that annotating the
function's own return proves nothing because Swift widens `Data` into `Data?` on
the way out. That argument is about a RETURN annotation. A function-type
annotation is not one, and `swiftc` confirms it:

```
    let arm: () throws -> Data = p.dataOrError        the arm, type-checks
    ... from a () throws -> Data?                     "cannot convert value of
                                                       type '() throws -> Data?'"
```

One form, one regexp, three booleans read out of the parameter list, the
`throws` and the result. Break-tests: two rows' symbols swapped (both fail,
each naming the arm its declaration actually settles), and an annotation the
parser cannot read.

## Items 7 and 8 — git asked, rather than assumed

**Four prefixes were redundant by assumption.** `citationSkipDirs` was
documented as four entries git already excludes plus two load-bearing ones, and
nothing checked the four. A `.gitignore` is a file somebody edits, and an entry
that moved from decorative to load-bearing reads exactly the same.

Each entry carries its relationship now, and git is asked about it —
`check-ignore` for the rules, the listing for what those rules produce:

```
    gitNeverLists   .git. No rule excludes it; ls-files does not enumerate it
    gitIgnores      the four build directories, named by the platform
                    .gitignore files, kept for the walk
    gitWouldList    ai_docs (tracked) and docs/site (mkdocs output nothing
                    ignores) — the prefix is the only thing keeping them out
```

The break-test is the one the item named: commenting `app/build/` **and**
`build/` out of android/.gitignore — the first alone changes nothing, because
`build/` matches at any depth — makes two entries fail by name, saying the
prefix is now the only guard on every machine.

**And two senses of "this repository's file" were one set.** `--cached` is a
file the history has; `--others --exclude-standard` is a file somebody wrote and
has not added. A citation failure naming the first is a commit to fix and one
naming the second is an edit on the desk of whoever is running the test, and the
enumeration knew which and threw it away. Two invocations now, because one
command's output cannot say which flag produced a path, and `--others` already
excludes anything tracked so the lists are disjoint. The walk carries a third
sense, because a filesystem cannot answer the question at all.

```
    docs/zz-break-test.md (written here and not committed yet) cites check 99
    docs/zz-break-test.md (tracked) cites check 99
```

## Item 2 — two tolerances, one fixture, and now one floor

`pinEpsilon` is 0.0001 because it compares GrMobFlexSolver's doubles against
integers; `PIN_EPSILON` is 0.05 because it compares a real browser's LayoutUnits
against the same integers. Four hundred apart, each argued for on its own page,
and no statement anywhere of what either had to be true of.

They are not supposed to be equal — a tolerance bounds the machinery on ONE side
of a comparison, and the two sides are a Swift solver and Chrome. What they
share is the other side, and that side can state what it requires.
`Case.Resolution` is the smallest distance apart any two of a case's numbers
are, derived by `internal/pinfixture` and read by both:

```
    no pin / pin first / pin middle / pin last      8, 20, 20, 20
    pin middle, with spacing                         8
    pin last, with partial spacing                   4   ← the floor
```

Four is the middle arm of the collapse: the partial-spacing row charges a gap of
4 where the Row declares 16 and a fully collapsed gap would be 0. A tolerance of
4 makes those three the same answer. Both harnesses hold their own number an
order below it (`pinMargin` / `PIN_MARGIN`, four, `INK_MARGIN`'s argument), and
the fixture's own test holds the derivation to every case's numbers.

Break-tests: 1.5 on each side, and each fails on the partial-spacing row, naming
the other harness.

## Item 10 — a probe built as `func()` because the function reads results

`swiftResults` reads a signature's results and nothing else, so building every
probe as `func() (…)` covers every bound method — today. That is a fact about
the function rather than about the space, which is the same shape as the three
arms this test exists to close.

Every result list is probed twice now, once bare and once behind
`(a int, b string, c []byte)` — one of each kind a bound method can take, every
one a `bindableGoTypes` row — and the two rendered declarations must be
identical. `sameResultShape` is written out rather than `reflect.DeepEqual`, so
a field added to `resultShape` has to be considered here.

The break-test makes `swiftResults` return early on any parameter list, and
every result in the enumeration reports two different declarations for one
method.

## The break-tests

**21 run: 19 that had to fail and did, 2 controls.**

    item 1      a parity pair with one branch (gen.go refuses)                1
                a trailing inset on the plain count-hidden band — the new
                unbadged pair fires in every theme                            1
    item 2      1.5 in browser.mjs, 1.5 in pin.swift                          2
    item 3      a magenta ::first-letter painted into the probe               1
    item 4      the old fractions against their own band, a line-height of
                2.6, a title long enough to wrap                              3
    item 5      two rows' symbols swapped, an annotation the parser cannot
                read                                                          2
    item 6      the digits transparent, the digits in another ink, the
                window widened past the pill's paddings                       3
    item 7      an ignored dir claimed load-bearing, a load-bearing dir
                claimed ignored, the .gitignore edit the item names           3
                and that edit with only `app/build/` removed (control)        1
    item 8      an untracked citation, the same file tracked                  2
    item 9      the exact minimiser against a 20,000-step scan over
                200,000 random triples (control)                              1
    item 10     swiftResults reading a parameter                              1

The item 7 control is the one that corrected the work: commenting out
`app/build/` alone leaves `android/app/build` ignored, because `build/` in the
same file matches at any depth. The break-test that means something removes
both.

## Verification

Ten paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines — gate harness + gate + flex + stack
                            + 3 bands + 6 pinned Rows (now holding their own
                            tolerance to the fixture's floor) + 15 menus
                            + replay + view + IMPORTER TYPECHECK + app
    android/verify/run.sh   gate harness + gate + the Compose census's source
                            half + 15 picker menus + 22 value ranges
    wasm/verify/run.sh      replay + 20 mjs suites + BROWSER PASS (12 checks,
                            15 real bands over five shapes, their words AND
                            their counts' digits scanned on rows measured
                            against the resolved face, behind one
                            antialiasing probe)
    android ./gradlew compileDebugKotlin --offline
    android ./gradlew :app:fetchComposeLayoutSources --offline
    GOOS=js GOARCH=wasm go build ./...

No new files. Nine changed.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

1. **(age 0 · value medium) The probe answers for the page and the scan trusts
   it for every band.** `SUBPIXEL_PROBE` is black on white at one size in one
   box, and what it licenses is a linearity claim about nine other colour pairs
   at whatever sizes the themes set. Chrome can and does pick per-element
   rendering paths — a translucent ink, a transform, a compositing layer — so
   "this box is grayscale" is one measurement standing in for fifteen, and the
   fifteen are the ones the tolerance is spent on.
2. **(age 0 · value medium) The x-height band is measured for the element and
   the scan reads the element's whole width.** `inkBandFault` uses the inline
   text box to find the baseline, and then the label scan sweeps
   `r.label.x` to `r.label.x + r.label.w` — which on the stretched branches is
   three times the width of the words. Every column past the run is backdrop
   that answers `notFill: false` and sits at `t = 0` on the segment, so the two
   halves of the scan are looking at different rects and only one of them was
   measured.
3. **(age 0 · value medium) `Case.Resolution` is the closest two numbers come,
   and not the closest two numbers a check must tell apart.** It is derived over
   every number in the case — offers, bases, both columns, the gaps — on the
   argument that some assertion somewhere compares a measurement against each.
   That is a superset, so the floor is the tightest one available and is safe;
   it is also not the real question, and a fixture that added a number no check
   reads would tighten a bound for no reason.
4. **(age 0 · value low) The convention arms are enumerated over results and
   parameters, and the probe parameters are one list.** `conventionProbeParams`
   is three types chosen as "one of each kind", which is the same shape as the
   three arms: a list that describes today's kinds. The input space is closed
   and enumerable — `bindableGoTypes` and the bound interfaces — so the
   parameter half could range over it the way the result half does.
5. **(age 0 · value low) The indented shape moves the label and nothing states
   what it is supposed to move it to.** The band with `PaddingLeft(40)` is in
   the grid so the scan meets a rect it has not met, and every assertion over it
   is one the other four shapes also make. Nothing says the label's rect
   actually moved — the shape would still pass if `ControlStyle` stopped
   reaching the control, which is the declaration it is there to exercise.
6. **(age 0 · value low) `citationSkipDirs` classifies prefixes and
   `citationExempt` classifies files, and only one of them is asked of git.** An
   exempt file is held to still carrying a citation, which is the right check
   for that table; nothing asks whether an exempt PATH still exists. A renamed
   file leaves a row that can never be satisfied, and the failure it produces
   names the exemption rather than the rename.
7. **(age 0 · value low) The two senses reach a failure message and no
   assertion.** `repoFile.sense` tells a reader who has to act, which is worth
   having, and nothing in the check turns on it. A citation in an untracked file
   and one in a committed file are the same failure at the same severity — which
   may be right, and is currently not a decision anybody made.
8. **(age 0 · value low) `SUBPIXEL_EPSILON` is 1 and the probe measures 0.**
   The bound absorbs "the browser's eight-bit rounding", and a blend of two
   greys rounded to eight bits is a grey exactly — the margin between the
   measurement and the floor is the whole tolerance, and nothing records that
   the real answer is zero.
9. **(age 3 · value low) `bandRenderBuilders` mounts five shapes and every theme
   paints them the same way.** The shapes now differ from each other; what a
   theme changes is still only the palette. A theme with a different caption
   tier — a larger `Typography.Caption`, a `LineHeight` — would move every rect
   in the grid, and the three bundled ones do not.
