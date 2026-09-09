# Session: a fourth reader nobody joined, a name a canvas may refuse, and a message that named the layout

Session: https://claude.ai/code/session_01MQcEV7mAeEV4h1LywCdQhe
Date: 2026-09-08 (follows "a-window-between-the-shutter-and-the-faces-a-floor-nobody-measured-and-a-width-the-ceiling-was-costing")

## Ask

"How many items are in the Next list?" — seven, all raised the session before,
all age 0. Then "let's do item 1", the one medium of them.

| # | age·value | item |
|---|---|---|
| 1 | 0·medium | The three windows are closed and the run metrics are inside none of them |

## Item 1 — the fourth reader

Three readers of the band grid were already held to one another. The rects, the
computed styles and the run metrics come back from one evaluate and are of one
layout by construction; the capture is bracketed by `LAYOUT_FINGERPRINT_JS`; the
platform-font reads are bracketed by that and `TREE_FINGERPRINT_JS` together,
which is what `facesHeld` names.

The fourth is a canvas. `band()` opens one per scanned box and takes three
numbers off it — the ascent that puts the baseline under the inline text box's
top, the ink ascent the three sampled rows are fractions of, and
`measureText(textContent)`, which is one of `inkRunRectFault`'s two width
answers. Those ride back with the rects, so no relayout can get between them and
the boxes they are about; that window was closed last session. What nothing held
is WHICH FACE they are about.

The canvas is handed a shorthand this file assembles out of four computed
longhands, and an element's font request is not four longhands. `font-stretch`
selects a width face, `font-size-adjust` changes the size the face is asked for,
`font-variant-caps` can reach a small-caps face or have one synthesised, and
`font-variation-settings` and `font-optical-sizing` pick an instance of a
variable font. None of them travels through the shorthand.

### Why the two width answers could not see it

The note above `inkFaceList` says why `inkRunRectFault`'s pair is blind to the
resolution: canvas measurement and layout go through one shaper, so a family
list that resolved to something unexpected moves both numbers together and they
agree about the wrong face. That argument holds while both sides are ASKED the
same thing — which an assembled shorthand is not. When the requests differ the
pair does fire, and it fires with the wrong story: its message is about
arithmetic done to a rect after shaping, and it sends a reader to the layout for
a face the canvas picked. That paragraph is now in the file, next to the
argument it qualifies.

### The join

The same measurement, with `CSS.getPlatformFontsForNode`'s own family at the
head of the element's own list. Naming a face the canvas has already resolved to
changes nothing, so a difference is the canvas on some other face.

    AmberTheme/a plain band: the label's words are drawn by Times, and the
    canvas the ink band and the run advance are measured on makes them
    93.6152px wide against 75.4800px for Times itself — 18.1353px wider for the
    same string ("January 2026"), against a LayoutUnit of 0.015625

Prepended rather than substituted, and that is the whole of what makes it a
one-sided test. A platform font name is a FACE's name and need not be a family
CSS can reach — `.AppleSystemUIFont` is not — so a substitution would measure
the canvas default and report every run in the grid. At the head of the list a
name that resolves to nothing is skipped and the list resolves as it did: equal
advances, and the check goes quiet rather than wrong. What is left to report is
the one refusal the browser states out loud, the assignment itself not taking,
which is the `unspellable` arm.

Bounded by a LayoutUnit rather than asked as an equality: two measurements of
one face, on one canvas, in one evaluate, of one string are the same number, and
the bound is there so the check is about a face and not about a double's last
bits.

Taken in the after-faces evaluate, because the family it names comes back over
the protocol and there is nowhere earlier to know it. It costs no round trip —
the two fingerprints were being taken there anyway — and it is covered by both
of them, which is the same suppression every other face consultation gets.

`INK_FONT_SHORTHAND_JS` is the shorthand spelled once, carried as source the way
`FNV1A_JS` is, because two spellings would leave the join holding a shorthand
nothing ever measured with. Asked last inside `inkUnreadable`, after the arm
that establishes there is one face to name, and it SUPPRESSES the scan rather
than reporting beside it: a canvas on another face makes the three rows
fractions of another face's band.

## The break-tests

**4 run, every one failed as it had to.**

    canvas resolved to Courier while layout is on Times          28 messages
    the same, with the join dropped                              20 messages
    the family assignment forced to refuse                       28 messages
    the read dropped from the after-faces evaluate               28 messages

The second is the item. With the join gone the same defect is reported by
`inkRunRectFault` as

    the browser's layout makes the run of words 75.4844px wide and the same
    face's own advance for the same string is 93.6152px — 18.1309px narrower

— "a rect that is not the width of these words", pointing at the layout. The
eight badge runs go unreported entirely, because nothing holds a digit's rect to
an advance, and six census lines fire saying how much of the grid went unread.
The defect was visible; it was visible as a confident finding about the wrong
thing.

The first injection was run twice: prepending `Times` to the measuring side
passed, which is how this run learned that all 28 scanned runs resolve to Times
and only the probes reach Helvetica. `Courier` is what separates them.

## Verification

Ten paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      replay + 20 mjs suites + BROWSER PASS (20 real
                            bands over five shapes and four palettes, drawn by
                            one platform face this browser names — one across
                            the whole grid — and named again to the canvas the
                            ink band and the run advance are measured on in 28
                            runs, which gave the same width both ways)
    android ./gradlew compileDebugKotlin --offline
    android ./gradlew :app:fetchComposeLayoutSources --offline
    GOOS=js GOARCH=wasm go build ./...

One file changed: wasm/verify/browser.mjs.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

1. **(age 1 · value low) The ligature census is measured on a face and spent on
   strings.** `inkLigatureNote` asks whether the string in front of it contains
   a pair this face joins, by `text.toLowerCase().includes(pair)`. The seeds
   are ASCII pairs and the fixtures are Latin, so lowercasing is the whole of
   the normalisation — a fixture carrying a composed form, or a pair split by a
   zero-width joiner, would be refused by gen.go and reported by this note as
   not containing what it contains.
2. **(age 1 · value low) `INK_OWN_SHAPE_FLOOR` is a quarter and the quarter is
   the judgement now.** The floor moves with the one enumeration this file has
   counted, which is the thing it was missing, and the fraction is chosen. The
   check that would settle it is the one the census already has the numbers
   for: the widest read this run takes (`inkOwnRead`) and the narrowest object
   the shape test must reject (a subject read's four keys) are both in hand, so
   the floor could be stated as sitting between two measured populations rather
   than as a fraction of one.
3. **(age 1 · value low) `bandtarget.mjs` owns the parts and browser.mjs owns
   the populations.** `declared` derives each part's census population from
   `everyBand`, in browser.mjs, from a table the module exports. That is one
   more join than the module makes: a part added with the wrong `everyBand`
   gets a population and a counter that agree with each other and with nothing
   in the band loop, and the first thing to notice would be a census line
   failing on every run with no message saying which row is wrong.
4. **(age 1 · value low) The phrase count is reported and the copies are not
   distinguished.** "Spelled 2 times in the file" is the number; which of the
   two is the assertion and which is the prose above it is not said, and the
   note's own argument turns on exactly that distinction. `pinCodeOnly` knows —
   it is the pass that blanked one of them — and the note could name the copy
   that survives in code rather than leaving a reader to open both lines.
5. **(age 1 · value low) The silent set is measured and one member is named.**
   `silent[0]` is the alphabetically first path the walk opened and found no
   citation in, which today is a CI workflow — a file nothing would ever cite.
   The separating case that matters to `citationExemptVerdict` is a path an
   exemption could be about, and the population those are drawn from is the
   harnesses, not the tree. A silent set that was all workflow files would read
   as evidence for a property it does not support.
6. **(age 1 · value low) `afforded` is measured for capped sets and the message
   is the only reader.** The number says what the ceiling costs this struct,
   and nothing asserts the relation it rests on: that `afforded >= edits`
   always, and that a set with `affordedOpen` has no parent holding three
   leaves. Both are properties of the derivation rather than of core.Theme, and
   the sets the test contrives are the only place they are visible — one more
   set with a shape nobody chose would be the generated-input move this file
   makes everywhere else.
7. **(age 0 · value medium) The canvas join is one-sided by design and its
   silence is not told apart from its success.** Prepending is what keeps an
   unreachable platform family from reporting every run, and the cost is that
   "the head family resolved and matched" and "the head family resolved to
   nothing and was skipped" come back as the same number. `asked.canvasFaces`
   counts 28 either way, and the tail recites it as though the face had been
   named. The separating measurement is available in the same evaluate: a third
   advance under the platform family SUBSTITUTED for the list says whether that
   name reaches a face at all, and it is a reading rather than an assertion —
   the arm that reports on it is the one that would have to be argued for.
8. **(age 0 · value low) `inkCanvasFaceFault`'s missing-box arm has no reader
   but the suppression above it.** `!m.asked` without `unspellable` needs the
   page's `querySelector` to miss a path the protocol had just resolved a node
   id for — which is a tree that moved, which trips `treeMovedFault`, which
   drops the whole face consultation through `facesHeld` before this arm is
   reached. Either the arm is unreachable and says so, or it is the cheaper
   reader of a condition the fingerprint reports more expensively; nothing in
   the file decides which.
9. **(age 0 · value low) The bound is stated from the derivation and nothing
   records what the population does.** `LAYOUT_UNIT` is the tolerance because
   two measurements of one face are the same double, and on this run all 28
   came back exactly equal — a fact the run does not carry. `INK_ROW_ROUNDING`
   has a census on both sides of it for exactly this reason: a margin that is
   silently spent is a margin nobody notices leaving. The widest difference the
   28 runs report is one number, and the injected face swap costs 18px against
   a bound of a 64th, so both brackets are in hand.
