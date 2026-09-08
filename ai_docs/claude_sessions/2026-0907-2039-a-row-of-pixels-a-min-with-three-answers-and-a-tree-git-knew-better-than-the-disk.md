# Session: a row of pixels, a min with three answers, and a tree git knew better than the disk

Session: https://claude.ai/code/session_017k47JkhG1WBZR5nP29c7Ge
Date: 2026-09-07 (follows "the-whole-next-list-a-column-nobody-held-a-gap-nobody-charged-and-a-mask-nobody-chose")

## Ask

"Do the remaining items in the Next list" — all ten.

They turned out to share a shape too, and it is one level down from last
session's. Last time the recurring fault was **a claim with no holder**. This
time it is **a number chosen against something that can move**: a scan row
picked because it was "most likely" to cross a stem, a sample five pixels into a
pill whose padding is eight, a byte floor derived from one row's subject and
applied to five, a tolerance whose bound was a sentence, a tree defined as the
disk minus a list.

| # | age·value | item |
|---|---|---|
| 1 | 0·medium | The label-ink scan reads one row of pixels |
| 2 | 0·medium | The composite is this file's arithmetic |
| 3 | 0·medium | `maskNonCode` does not know Groovy's single quotes |
| 4 | 0·low | The five readers are named for three questions, and `proseIn` has no `Of` |
| 5 | 0·low | The importer's protocol is one shape of one convention |
| 6 | 0·low | The receipt is only written by a fetch |
| 7 | 0·low | `windowSlack` is one number for five rows |
| 8 | 0·low | The citation walk reads every text file in the tree |
| 9 | 2·low | The band grid samples the badge at a fixed 5px |
| 10 | 2·low | The pin fixture's spacing row is one gap |

## Items 1, 2 and 9 — three numbers the band grid had picked

**One row became three.** The label was scanned across its vertical middle,
which the comment called "the row most likely to cross a stem" — and likely is
not something a check rests on. A face whose x-height band fell between two
stems at that exact y reports a label with no ink in it. `INK_ROWS` is
`[0.4, 0.5, 0.6]` now, three passes of the same arithmetic over one rect.

The break-test is the argument: pointed at `[0.05]` — an ascender row, above the
x-height — every one of the nine bands fails with "the words are not there".
Each of the three chosen rows finds ink on its own, so the redundancy is real
rather than nominal.

**The tolerance's bound is a check.** `INK_EPSILON` is 3 because two roundings
have to agree, and "three is far below the distance to any other colour in a
band" was the whole justification — a sentence about a distance nothing
measured. `confusableInk` measures it, against the three colours a pixel inside
a band can legitimately be (the fill, the pill, the page), at `INK_MARGIN` times
the tolerance.

    the closest pair anything bundled paints    109 channels
    the floor                                    12

DefaultTheme's translucent secondary ink over its own band fill is the tight
one, and 109 is now written down rather than assumed. A case that cannot
discriminate reports that instead of passing.

**The badge sample moved with the widget.** Five device-independent pixels in,
against a pill whose leading padding is eight — a number measured off a widget's
current insets, which is the shape items 12 and 13 were about one level up.
`gen.go` carries `BadgePadLeft` off the rendered node and refuses a padding too
small to sample inside; the browser samples half of it.

The sharpest break-test of the session: shrink `components.Badge`'s leading
padding from 8 to 4 and the derived point still passes, while the old literal 5
fails on `#CD8109` — an antialiased digit edge. The defect was not hypothetical,
it was one inset change away.

## Item 10 — a min has three answers

`spaceAfterLastNoWeight` is `min(spacing, what is left)`, and the fixture's one
gapped row reached only its ENDS: 8 charged after the lead child, 0 after the
pin. Both are real arms and both are also what a different, simpler, wrong rule
produces — *charge the gap unless the Row has overflowed*. Those two rules agree
everywhere the fixture looked.

`partialGap` is 16 and the row is `[A,B,P]`, which is the pin-last row again
with a gap:

```
   gaps   16, 16   against   16, 4        16 asked for, 4 left, 4 charged
   CSS     0, 0, 200                      unchanged from its twin
   Compose 60, 40, 200                    unchanged from its twin
```

16 is bounded on both sides by arithmetic and the comment states it: at 10 or
below the spacing is charged in full, at 20 or above it collapses, and only
11..19 reaches the middle. So the row is the same kind of statement the 8px row
is — everything about it is its twin's, and the only thing it can be measuring
is the gap.

Both harnesses now refuse a transcript that has lost it (`sawPartialGap`), for
the reason each already refuses one that cannot overflow. The break-test that
matters is the second one: replacing `min` with "charge unless overflowed"
changes no number in the fixture except this row's 4.

## Items 3 and 4 — the scanner's fourth language, and the cut that went the wrong way

**`maskNonCode` knows Groovy now.** Three of the four languages it serves spell
their literals identically; Groovy has `'...'` and `'''...'''` and the scan knew
neither. That was found by accident last session, and the direction of the gap
is worth recording: the risk was never a check that failed. It was a check whose
subject is a string literal in a language the scan believed had none — a
coordinate quoted in a comment satisfying a question asked under a reader named
for code.

The apostrophe arm is bounded at the newline, and that bound is what makes it
safe to add: an apostrophe with no partner blanks nothing rather than swallowing
the file. Both new rows of the mask/counter agreement table fire when their arm
is removed, and the `it's fine` row holds the arm's refusal to over-reach.

The vocabulary had to move with it. What this package called a "single-quoted
literal" was `"..."` — one quote character — which is now a *double-quoted*
literal, because there are real single quotes in the scanner.

**`proseOf`.** Prose had an `In` and no `Of`, and that was a gap in the CUT
rather than in the naming: `declSource` blanks comments before it looks and then
runs from the declaration line DOWN, and a declaration's note is written above
it. So two checks about one function's note were checks about a thousand-line
renderer.

`proseSourceOf` walks the comment block in both directions — back from the
anchor to the first blank or code line, forward to the next declaration minus
ITS block — with offsets off the mask and text off the source.

The break-test is the whole case for it. Move the phrase out of
`grMobSelectedTrait`'s note and into a neighbouring declaration's:
`proseOf` fails, and `proseIn` passes.

## Item 5 — the other two arms of one convention

`swiftResults` sends a bound method's `(value, error)` result down one of three
arms, chosen by gobind's nullability annotation. One was settled by a compiler
last session, because it is the arm `([]byte, error)` reaches. The other two
were the bind somebody ran once.

```
   nullable object   NSData*  _Nullable    () throws -> Data
   _Nonnull object   NSString* _Nonnull    (NSErrorPointer) -> String
   BOOL              BOOL                  () throws -> Void
```

Two methods on the same protocol, two annotated method references. The
_Nonnull arm is spelled as a reference rather than a call because half of what
it claims is the parameter that SURVIVED, and each annotation discriminates
against the one plausible alternative: `() throws -> String` (the convention
applying after all) does not convert to the second row in either direction, and
`(NSErrorPointer) -> Bool` (the convention declining) is not a throwing nullary
function.

All three break-tests are header mutations — the nonnull return made nullable,
the BOOL return made an object, the error parameter removed — and each fails at
the line naming both types.

## Items 6 and 7 — the census's two silences

**A receipt nobody wrote was compared with nothing.** `checkResolutionAgrees`
returned silently when `android/app/build/composeLayoutSources.txt` was absent,
which is the state of every machine that has run `./gradlew` and not the fetch —
so the loud answer arrived only for people who had already done the thing the
check is about. `receiptVerdict` names it and puts it under
`GRMOB_COMPOSE_SOURCES`, the way every other absence in that file is. `./gradlew
clean` removes the receipt and leaves the jar, which is why this is a verdict
rather than an expectation.

**The window floor is the row's own.** `windowSlack` was 40, which was 38
rounded up, which was the length of `).coerceAtMost(constraints.mainAxisMax)` —
one row's subject, applied to five rows, with no way to stay right if a second
`notWant` about something else appeared. Each row states its `absence` instead:
the shortest plausible spelling of what it refuses, whose length IS its floor.

The pair is held both ways, and each direction is a way it rots: a `notWant`
with no absence is a claim with no floor, an absence with no `notWant` is a
floor for a spelling the row cannot see, and a mark the absence would not
contain means the two have come apart. The one row that states an absence has 88
bytes where it needs 39.

Rows with only `want`s get no floor, and that is the honest answer rather than a
narrowing: a window too short for a positive claim drops the substring off its
end and fails by name.

## Item 8 — the tree is git's answer, not the disk's

The citation walk read every file on disk minus five prefixes minus anything
with a NUL in it, and that was the whole of what stood between it and a
`node_modules` somebody adds. `git ls-files --cached --others --exclude-standard`
is every tracked file plus every untracked one git would offer to add — which is
exactly the set a person here writes, and keeps the property the walk was
adopted for, since `--others` covers a file written five minutes ago.

Four of the five skip prefixes are now reached only by the fallback: git
excludes the build directories itself, because the two platform `.gitignore`s
already name them. `docs/site` is the one that is load-bearing in both paths,
and `ai_docs` is a decision no mechanism could make.

The three break-tests are the specification:

    a vendored dep git ignores       invisible
    the same file, not ignored       loud, by name — and the fix is a skip entry
    git absent                       the walk, announced, with what it cannot do

What remains is a dependency vendored by COMMITTING it, which is this
repository's file by every mechanical test there is. That failure is loud, and
saying so is better than pretending the enumeration covers it.

## The break-tests

**24 run, 24 caught.**

    item 1      the scan pointed at one row above the x-height, and each of
                the three chosen rows alone                                  4
    item 2      the margin raised until a real theme trips the guard          1
    item 9      the badge sampled outside the pill, a pill padding too small
                to sample inside, and the old literal 5 against a 4px pill    3
    item 10     partialGap collapsed to 8, MeasureCompose's min replaced by
                "charge unless overflowed", the row removed (browser, Swift)  4
    item 3      the apostrophe arm removed, the Groovy multi-line arm
                removed, the gradle read moved to the code-level reader       3
    item 4      an anchor pointing at a neighbour, the note reworded, the
                note moved to another declaration in the same file            3
    item 5      the nonnull return made nullable, the BOOL return made an
                object, the error parameter removed                           3
    item 6      the receipt renamed away, with the switch and without          2
    item 7      the absence removed, a notWant the absence would not catch,
                a window shrunk below its own floor                           3
    item 8      a vendored file ignored and the same file not ignored, git
                made unavailable                                             3

Two corrected the work rather than confirming it. The first slack break-test
passed because the row's positive claims end 132 bytes in and not 142, which the
old comment had wrong; the number is measured and written down now. And the
`INK_MARGIN` probe is what produced 109 — the sentence about the tolerance's
bound now has a figure in it.

## Verification

Ten paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       gate harness + gate + flex + stack + 3 bands
                            + 6 PINNED ROWS (both columns, both spacings)
                            + 15 menus + replay + view + importer + app
    android/verify/run.sh   gate harness + gate + the Compose census's source
                            half + 15 picker menus + 22 value ranges
    wasm/verify/run.sh      replay + 20 mjs suites + BROWSER PASS (12 checks,
                            9 bands scanned across three rows, 6 pinned Rows)
    android ./gradlew compileDebugKotlin --offline
    android ./gradlew :app:fetchComposeLayoutSources --offline
    GOOS=js GOARCH=wasm go build ./...

No new files. Fifteen changed.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

1. **(age 0 · value medium) `INK_ROWS` is three fractions of a box, not three
   rows of a glyph.** 0.4/0.5/0.6 of the label's line box is inside the x-height
   band for the faces Chrome picks today, and the argument for the spacing —
   "far enough apart to sit in different rows of a glyph's bitmap at caption
   sizes" — is a claim about a font this repository does not choose. The check
   that would settle it is the one the fold guard got: a function of the box's
   height and the fractions, refusing a configuration where two rows land on the
   same device pixel.
2. **(age 0 · value medium) `confusableInk` compares the composite with three
   colours the band declares, and a screenshot holds more than three.** The
   chevron's tint, a focus ring, a pressed state: none is in the band fixture
   today and any of them is a colour a label pixel could be. The guard is a
   claim about the palette a case DECLARES rather than about what the capture
   contains.
3. **(age 0 · value medium) `proseOf`'s block rule is about blank lines, and
   Kotlin's KDoc is one comment.** `/** … */` spanning fifteen lines is a single
   construct the mask blanks whole, so the walk back over "comment-only lines"
   happens to work. A doc comment with a blank `*` line inside it — legal, and
   common in this repository's Swift — would end the walk early and cut the note
   in half, silently.
4. **(age 0 · value low) The apostrophe arm cannot see a Groovy dollar-slashy
   string.** `$/…/$` is Groovy's third literal form. Nothing in
   `android/app/build.gradle` uses one, which is exactly the state `'...'` was in
   until somebody looked.
5. **(age 0 · value low) The three convention arms are declared on one protocol
   and read by one file.** `swiftResults` picks between them with
   `nullableObjectResult` and `gobindErrorOutPointer`, and nothing pairs the Go
   arms with the Swift declarations the way `importerLine` pairs the type tables
   with their annotations. A fourth arm added in Go would compile, and
   importer.swift would go on settling three.
6. **(age 0 · value low) `absence` is one spelling per row.** The floor is the
   shortest plausible way to write what the row refuses, and a row refusing two
   different things — which is what a second `notWant` about another subject
   means — wants the longer of two floors. The pair check would accept it
   today: a single absence containing both marks.
7. **(age 0 · value low) The receipt verdict fires once per run and the census
   check runs once.** A machine with no receipt is told so by exactly one test,
   and that test skips entirely when the sources are absent — so the state the
   verdict is about is reachable only on a machine that has fetched the jar and
   then lost the build directory.
8. **(age 0 · value low) `repositoryFiles` treats a git failure and an empty
   listing alike.** Both fall back to the walk, which is right for the first and
   arguable for the second: an empty listing from a command that SUCCEEDED is a
   repository with no files in it, which is a different fault from having no
   git, and the fallback tells the reader the wrong one.
9. **(age 2 · value low) `bandRenderBuilders` mounts three shapes and every
   theme paints them the same way.** The nine bands are three shapes times three
   themes, and what the ink scan reads back differs only in the palette — so a
   shape whose label is positioned differently (a band with no badge and a long
   title, say) is not in the grid, and the scan's rect is the same rect nine
   times.
10. **(age 2 · value low) `pinEpsilon` and `pinSame` are two tolerances for one
    fixture.** The Swift side compares CGFloat against integers at 0.0001 and
    the browser side has its own; the fixture's numbers are integers and both
    are reading the same six Rows. Nothing states that the two agree about what
    counts as the same pixel.
