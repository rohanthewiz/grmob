# Session: three fractions that can be one row, three arms that can be four, and a walk the last commit never carried

Session: https://claude.ai/code/session_017k47JkhG1WBZR5nP29c7Ge
Date: 2026-09-07 (follows "a-row-of-pixels-a-min-with-three-answers-and-a-tree-git-knew-better-than-the-disk")

## Ask

"Do the age 0 items in the Next list" — the eight raised last session.

They share a shape, and it is the one immediately below last session's. That
one was **a number chosen against something that can move**. This one is **an
enumeration that describes today's members rather than the space they come
from**: three arms because three is how many there were the day somebody wrote
them, one spelling because one row happened to refuse one thing, three colours
because three is what a band declares, three fractions that are three rows only
while the box is tall enough, four literal forms because the fourth language
uses three of them here.

The fix is the same move each time, in one of two directions: close the space
and enumerate it, or stop asking the declaration and ask the capture.

| # | age·value | item |
|---|---|---|
| 1 | 0·medium | `INK_ROWS` is three fractions of a box, not three rows of a glyph |
| 2 | 0·medium | `confusableInk` compares with three colours the band declares |
| 3 | 0·medium | `proseOf`'s block rule is about blank lines, and KDoc is one comment |
| 4 | 0·low | The apostrophe arm cannot see a Groovy dollar-slashy string |
| 5 | 0·low | The three convention arms are declared on one protocol and read by one file |
| 6 | 0·low | `absence` is one spelling per row |
| 7 | 0·low | The receipt verdict fires once per run |
| 8 | 0·low | `repositoryFiles` treats a git failure and an empty listing alike |

## Item 8 first — the work it refines was not in the tree

`repositoryFiles` does not exist. Last session's doc describes moving the
citation walk from the disk to `git ls-files --cached --others
--exclude-standard`, with three break-tests and a paragraph about what the
change buys; commit `7c77214` touched sixteen files and
`wasm/verify/checknumbering_test.go` was not one of them. `citingFiles` was
still `filepath.WalkDir` minus five prefixes.

So item 8 is both halves. The enumeration is git's now, the fallback walk is
kept and announced by name — a source tarball, a container built by copying the
tree in, and `go test` run from an export all reach it — and `citingFiles`
reports which one answered, because the two cover different sets and a failure
naming a vendored file means something different depending on which produced it.

The age-0 refinement is the third answer, which is the one that had been folded
into the first:

```
    git could not answer          the walk, announced with git's error
    git answered with nothing     a repository containing no files, which
                                  this one is not — the enumeration is being
                                  run somewhere other than the working tree
    git answered                  the set a person here writes
```

Falling back on the second tells the reader the first story about the second
situation, and hands them a passing check while doing it.

The break-tests are the specification, and two of them are controls:

    a vendored dep git ignores       invisible
    the same file, not ignored       loud, by name — node_modules/leftpad/
                                     CHANGELOG.md cites check 99
    git absent                       the walk, announced, with what it cannot do
    git empty                        the fault named as its own

## Items 1 and 2 — the band grid's other two sentences

**Three fractions are three rows only sometimes.** `INK_ROWS` is `[0.4, 0.5,
0.6]` of the label's line box, and the argument for the spacing — "far enough
apart to sit in different rows of a glyph's bitmap at caption sizes" — was a
claim about a font and a display in a file that chooses neither. The fractions
are a tenth of the box apart, so they are three rows exactly while a tenth of
the box is at least one device pixel.

That is closer than it reads. MaterialTheme's label is 14 device-independent
pixels tall at dpr 1, and the three rows come out at **181, 182, 183** —
consecutive. `inkRows(top, height, dpr)` resolves them and the scan refuses a
box where two land together, because a scan reading one row twice has one
chance and reports three.

The break-test is `[0.40, 0.42, 0.44]`: every band fails by name, and with the
guard removed the same fractions pass silently on rows `63, 63, 63`.

**The tolerance's palette is the capture's now, not the declaration's.**
`confusableInk` clears a case by measuring the composited ink against the three
colours a band declares — its fill, its pill, the page — and a screenshot holds
whatever was painted. A chevron's tint, a focus ring, a pressed state: none is
in the fixture and each is a colour a label pixel could be, so the guarantee
"this scan can tell the ink from everything else in the box" was a claim about a
list rather than about a rect.

`offSegment` asks the rect. Antialiasing is exact linear blending — a glyph
pixel at coverage `c` is `fill + c·(want − fill)`, and a translucent ink's alpha
multiplies into `c` and comes out on the same line — so every legitimate pixel
in the label's box lies ON the segment between the fill and the composited ink.
A pixel off it is a third colour drawn inside the rect, which is a colour
`confusableInk` was never given the chance to rule out.

```
    the worst real deviation, nine bands over three themes     1.2 channels
    the floor                                                    3
```

Three is `INK_EPSILON` itself rather than a number of its own, and that is the
argument for it: a pixel within the ink tolerance of the composite is accepted
AS the ink two lines below, so the distance at which "off the line" starts
mattering is exactly the distance at which the scan stops being able to tell.

The sharpest break-test of the session is a magenta `::first-letter` injected
into the page before the screenshot — a real third colour, painted by a real
browser, **97.8 channels off the line**.

## Item 3 — a rule about blankness, applied to a construct that spans it

The defect was live rather than latent, and it fails in the silent direction.
`proseSourceOf` walks back over "comment-only lines" and stops at the first
blank one, which is right for `///` runs and wrong for the one construct that
contains blank lines: `/* … */`. A KDoc or a Swift block comment with an empty
line between its paragraphs had its note **cut in half**, and the region did not
merely start late — it started four characters into a comment:

```
    /**                                          ← outside the region
     * Go's core.SelectedState as a trait.       ← outside the region
                                                 ← the walk stopped here
     * SwiftUI has no word for the off state.    ← the whole region
```

A `strings.Contains` over that still returns a string. A phrase in the half that
was cut off reads as a phrase that was deleted.

Blankness cannot answer this, because a line that is empty inside a comment is
blank in the mask and so is one that separates two paragraphs. So the scanner
says where its comments were — the same shape `unterminated` is, one more thing
the scan already knows and a mask cannot carry, rather than a second pass. A
line inside a span continues the note whatever it looks like; a blank line
outside one still ends it, which is the rule the paragraph always stated.

**And a row that could only pass.** The existing "the next declaration's note
stays out" case read `Nothing here has a word for the off state either` and
refused `no word for the off state`, which is not a substring of it. The
assertion held whatever the cut did, including when the cut carried the whole
neighbour along. Both neighbour rows use the neighbour's own wording now and
both fail when the end walk stops trimming.

## Items 4 and 5 — two enumerations that were complete on the day

**Groovy's third literal form.** `$/…/$` is multi-line by construction — it
exists so a regexp or a path can carry backslashes unescaped — so it is bounded
at `/$` rather than at the newline, and an unterminated one gets a name of its
own rather than sharing `multi-line string`, because a reader told the wrong
delimiter goes looking for the wrong character.

Nothing in this repository writes one, which is precisely the state `'...'` was
in until somebody looked, and the argument for adding the arm before rather than
after a file starts depending on it.

The escapes are what make the arm more than a delimiter. Content holding the two
characters `/$` is written `$/$$` — the slash escaped as `$/`, the dollar as
`$$` — and the middle two characters of that are a terminator to anything not
reading the dollars. The fixture carries exactly that, and the arm without its
escape handling closes four bytes in and leaves the hidden brace standing.

**The convention's arms are enumerable, and now they are enumerated.** Every
SPELLING in the two type tables is held to a line of importer.swift in both
directions. The three arms of `swiftResults`' method branch had the compiler and
not the pairing: which arms exist was a fact about a Go function that nothing
compared with the file settling them, so a fourth arm compiles and importer.swift
goes on settling three.

An arm is three booleans read off a rendered declaration — the NSError** became a
`throws`, or survived as a parameter, and the method still returns something or
does not:

```
    (error)  /  (scalar, error)   throws, no error param, no return
    (Iface, error) / ([]byte, …)  throws, no error param, a return
    (string, error)               no throws, the error param, a return
```

gobind's `ret0_` out-pointer is deliberately not a fourth field: that is a
question about nullability decided before the convention sees the method, and
already settled by `gobindErrorOutPointer` against its own importer lines. It is
why `(error)` and `(int, error)` share a row here and share a declaration over
there.

What makes this a check rather than the same claim written twice is that the
INPUT space is closed. gobind refuses three or more results and refuses a second
result that is not an error, so a bound method's results are one of `()`, `(T)`,
`(error)` or `(T, error)`, and T ranges over `bindableGoTypes` and the bound
interfaces — both enumerable here, with a synthetic protocol name for the arm no
bound function reaches yet. An arm added to `swiftResults` is an arm the loop
produces.

## Items 6 and 7 — one spelling, and one place to say it

**`absence` is a list.** A single spelling per row made the floor a fact about
the row, which is right, and a fact about ONE of the things the row refuses,
which is not. A second `notWant` is a second subject refused in a different
number of characters, and what the row needs is the longest of them. The
single-string version would have accepted one string holding both marks — a
spelling that is not how either thing would be written, sizing the window
against a sentence nobody would put in the file.

The pairing is per entry now, and the second direction is new with the list:

```
    a mark in no spelling      sized against one thing, looking for another
    a spelling with no mark    a floor that refuses nothing — the strictest
                               row in the table and the emptiest
```

The break-test that matters is the last one: a second spelling of 106 bytes
against a window with 88 of room fires, and taking the floor as the SHORTEST
instead of the longest passes.

**The receipt's report was reachable from one place, behind a skip.**
`receiptVerdict` was called only from inside `checkResolutionAgrees`, which the
census test reaches after deriving a version out of the BOM's pom — and that
derivation skips on a machine with no gradle cache. So the machine the report is
most about, the one that has never run `./gradlew`, was the one machine that
could not hear it, and `GRMOB_COMPOSE_SOURCES=required` could not reach it
there. A switch whose sentence is "this machine settles these claims" was false
for exactly the machine that needed it.

`TestTheFetchWroteDownWhatItResolved` holds one file's presence and nothing else
— no cache, no version, no jar — which is what makes it run everywhere. The
census's call is a comparison with nothing to compare rather than a report.

## The break-tests

**25 run: 19 that had to fail and did, 6 controls that had to hold.**

    item 1      the fractions collapsed to [0.40, 0.42, 0.44]                 1
                and the same fractions with the guard removed (control)       1
    item 2      a magenta ::first-letter painted into the label's box, the
                tolerance dropped below what real antialiasing produces,
                the line drawn to the page instead of the band's fill         3
    item 3      the walk reading blankness alone, inComment always true,
                the end walk no longer trimming the next note                 3
    item 4      the dollar-slashy arm removed, its escapes dropped            2
    item 5      a fourth arm added in Go, the declaration renamed away, a
                declaration no row names, a row deleted                       4
    item 6      a second spelling past the window's room, an absence no
                mark would recognise, a mark no absence carries               3
                and the floor taken as the shortest (control)                 1
    item 7      no receipt with the switch set                                1
                no receipt unset, and no gradle cache at all (controls)       2
    item 8      the vendored file not ignored, git answering with nothing     2
                the vendored file ignored, git absent (controls)              2

Two corrected the work rather than confirming it. The first escape break-test
PASSED, because the fixture's `$/b` is a dollar-then-slash and the terminator is
slash-then-dollar — they never collide, so the row was measuring nothing; the
fixture is `$/$$` now. And the proseOf neighbour row could only pass, for the
reason above.

Two more break-tests passed for good reasons and are recorded as controls rather
than as misses: a second absence of 71 bytes against 88 of room is a row that
genuinely has the space, and the guard-removed row is what makes item 1's guard
load-bearing rather than decorative.

## Verification

Ten paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       gate harness + gate + flex + stack + 3 bands
                            + 6 pinned Rows + 15 menus + replay + view
                            + IMPORTER TYPECHECK + app
    android/verify/run.sh   gate harness + gate + the Compose census's source
                            half + 15 picker menus + 22 value ranges
    wasm/verify/run.sh      replay + 20 mjs suites + BROWSER PASS (12 checks,
                            9 bands scanned across three resolved rows and
                            held to their own two colours, 6 pinned Rows)
    android ./gradlew compileDebugKotlin --offline
    android ./gradlew :app:fetchComposeLayoutSources --offline
    GOOS=js GOARCH=wasm go build ./...

No new files. Eight changed.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

1. **(age 3 · value low) `bandRenderBuilders` mounts three shapes and every
   theme paints them the same way.** The nine bands are three shapes times three
   themes, and what the ink scan reads back differs only in the palette — so a
   shape whose label is positioned differently (a band with no badge and a long
   title, say) is not in the grid, and the scan's rect is the same rect nine
   times.
2. **(age 3 · value low) `pinEpsilon` and `pinSame` are two tolerances for one
   fixture.** The Swift side compares CGFloat against integers at 0.0001 and the
   browser side has its own; the fixture's numbers are integers and both are
   reading the same six Rows. Nothing states that the two agree about what
   counts as the same pixel.
3. **(age 0 · value medium) `offSegment`'s linearity is grayscale
   antialiasing's, and nothing says so.** A glyph pixel is on the segment
   because coverage is one scalar; LCD subpixel antialiasing gives each channel
   its own coverage and puts ordinary text off the line. Headless Chrome
   disables it, and `INK_EPSILON`'s exact match already depends on that — so the
   whole band scan rests on a rendering mode no line in the file names, and the
   failure would be nine bands reporting a colour.
4. **(age 0 · value medium) The other half of `INK_ROWS` is still a sentence.**
   The rows are now held to being three DISTINCT rows, and "all inside the
   x-height band" — the reason they read ink at all rather than backdrop — is
   the claim that put the fractions at 0.4–0.6, and it is about a line box's
   relationship to a font's metrics. A tall line-height puts 0.4 on an ascender,
   and every band fails saying the words are not there.
5. **(age 0 · value medium) `errorConventionArms` names three symbols and
   nothing holds a symbol to its arm.** Both directions of the pairing are about
   EXISTENCE: a row with no function fails and a function with no row fails.
   Swap the names of two rows and both checks still pass, while the compiler
   goes on settling the arms under the wrong descriptions — which is exactly the
   readable-and-wrong state `importerLine` avoids by parsing the annotation.
6. **(age 0 · value low) A band's count is read as a fill and never as
   digits.** "A band is a fill, a run of words and a count" is the argument the
   ink scan was added for, and the count pill is still sampled at one point
   inside its own padding. A pill that painted itself and rendered no number
   passes, which is the state the LABEL was in before this scan existed.
7. **(age 0 · value low) The four redundant skip prefixes are redundant by
   assumption.** `citationSkipDirs` is documented as four entries git already
   excludes plus two that are load-bearing, and nothing checks the four. A
   `.gitignore` edit that stopped ignoring `android/app/build` would make one
   load-bearing again silently — the enumeration would still be right, for a
   reason that had moved.
8. **(age 0 · value low) `repositoryFiles` returns tracked and untracked files
   as one set.** `--cached` and `--others` are two different senses of "this
   repository's file", and a citation failure naming an untracked path is a
   different conversation from one naming a committed path. The listing knows
   which and the caller throws it away.
9. **(age 0 · value low) `offSegment` mixes two metrics.** The projection is
   Euclidean and the distance reported off it is max-channel, which is
   `channelDistance`'s metric and the one every tolerance in the file is
   expressed in. The two agree about zero and disagree about everything else, so
   the number compared with `OFF_SEGMENT_EPSILON` is not the number the
   projection minimised.
10. **(age 0 · value low) The convention enumeration probes results and never
    parameters.** `swiftResults` reads only a signature's results, so building
    every probe as `func() (…)` is sound today and is an assumption about the
    function rather than about the space — a rule that looked at a parameter
    would be enumerated against nothing.
