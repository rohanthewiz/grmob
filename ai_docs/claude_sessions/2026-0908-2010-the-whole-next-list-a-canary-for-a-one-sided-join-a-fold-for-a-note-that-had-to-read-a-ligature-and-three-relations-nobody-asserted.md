# Session: the whole Next list — a canary for a one-sided join, a fold for a note that had to read a ligature, and three relations nobody asserted

Session: https://claude.ai/code/session_01MQcEV7mAeEV4h1LywCdQhe
Date: 2026-09-08 (follows "a-fourth-reader-nobody-joined-a-name-a-canvas-may-refuse-and-a-message-that-named-the-layout")

## Ask

"Continue with the remaining Next list items. Do them in a few stages if needed."
Nine items — six at age 1, three raised last session. All nine done, in three
stages, each stage verified and broken on purpose before moving on.

| stage | items | where |
|---|---|---|
| A | 7, 8, 9 | the canvas join in browser.mjs |
| B | 1, 2, 3 | the ligature note, the shape floor, the tap-target census |
| C | 4, 5, 6 | pinfixture, checknumbering, themenearmiss |

---

## Stage A — the canvas join

### Item 7 · the canary (the one medium of the nine)

The join names `CSS.getPlatformFontsForNode`'s family at the HEAD of the
element's own list, and prepending is what keeps an unreachable platform family
from reporting every run — a face's name need not be a family CSS can reach.
The cost is that a head which resolves to nothing is SKIPPED: the list resolves
as it did, the two advances are equal, and "the face was named and matched" and
"the name reached nothing" come back as one number. `asked.canvasFaces` counted
28 either way and the tail recited it as though the face had been named.

Nothing already in hand separates them — the element's own list is the fallback
in both cases, so `asIs` is the same number either way, and a substitution is
exactly what the join must not do.

So a second pair of measurements, on the same canvas, in the same evaluate, of
the same string: the advance under `INK_CANARY_FALLBACK` alone, and the advance
under the platform family in front of it. The generic always resolves, so the
second falls back to the first EXACTLY when the name in front of it reaches
nothing.

    monospace, because the separation wanted is metric: this grid's faces are
    proportional and a fixed-advance face is the one whose widths for these
    strings are nothing like theirs. Being wrong the only way this can be wrong
    — a platform face whose advance happens to equal monospace's — costs a run
    its confirmation and never claims a face was named when it was not.

`INK_FONT_SHORTHAND_JS` grew a third parameter, `instead`, which REPLACES the
family list: substituting is the wrong shape for the join and the right shape
for the canary, because there the fallback is the answer. `inkCanvasNameReached`
is split out because two readers need the same verdict — the arm and the counter
— and a second spelling would let them disagree about which runs were confirmed.

The tail now recites both: 28 asked, 28 reached.

### Item 8 · the missing-box arm, decided

`!m.asked` without `unspellable` is the page's `querySelector` finding no
element at a path the rects were read at. It is **unreachable through a moved
tree**, and now says why: `TREE_FINGERPRINT_JS` hashes exactly this population —
every `[data-node-path]`, the path itself among the fields fed — and its second
reading is taken in that same evaluate. A path that stopped resolving moves the
hash, `treeMovedFault` fires, `facesHeld` drops the consultation first.

What is left is one evaluate's `querySelectorAll` and `querySelector`
disagreeing about the same attribute. The arm is kept as the reader that would
fire if the fingerprint's population ever stopped being the population these
lookups draw from, and its silence on every run is the evidence it has not. The
page also distinguishes the two ways of not asking now (`missing` vs no single
family), so the arm cannot be reached by the case `inkFaceFault` already owns.

### Item 9 · the bound's population

`LAYOUT_UNIT` is the tolerance because two `measureText` calls on one canvas, in
one evaluate, for one string, under two requests resolving to one face return
THE SAME DOUBLE. That is a claim about the derivation, and nothing recorded what
the population does under it — the shape `INK_ROW_ROUNDING` has a census on both
sides of it for.

`asked.canvasWidest` is the widest the two advances ever came apart, and
anything at all inside the bound is reported: zero is what the premise predicts,
so the reporting threshold is the premise itself rather than a fraction of the
margin. The other bracket is not hypothetical — a canvas on another face costs
about 18px on these strings, three orders above the bound.

    named again to the canvas the ink band and the run advance are measured on in
    28 runs — a name a canary finds reaches a real face in
    28 of them, so the agreement is a face's and not a skipped
    lookup's — which gave the same width both ways, the widest of those
    28 differences being 0.000000px against
    a bound of 0.015625,

### Break-tests — 4 run, every one failed as it had to

    the platform family replaced by a name nothing resolves    28 canary reports
    the same, with the canary arm dropped                      PASS, silently
    one run's second advance moved by 0.001px                  the bound census
    the after-faces lookup pointed at a path that is not there the missing arm

The second is the item. With the arm gone the run PASSES while every one of the
28 joins bit nothing — and the tail, which now carries the reading either way,
says `0 of them`.

---

## Stage B

### Item 1 · a note that had to read a ligature

`inkLigatureNote` asked `text.toLowerCase().includes(pair)`. That is right about
every string that has ever reached it, and right BECAUSE of gen.go's refusal
rather than independently of it — and this note is the message that fires when
that refusal did not do its job. Deciding what such a string contains by an
ASCII-shaped test is deciding it in the one world where the ASCII assumption is
known to have failed.

`inkFold`: NFKD, then the format characters, then lowercase.

    NFKD and not NFD, because an f-ligature is a COMPATIBILITY equivalence —
    the form that would be right for accents is the wrong form for exactly the
    characters this note is about. The zero-width and bidi controls carry no
    advance and no glyph, so a pair split by one is a pair the shaper still
    joins. Lowercase last, because the two steps above produce letters needing it.

The message says when the fold changed something, so a reader is not told a
string "contains fi" while looking at `ﬁ`.

**Break-test.** A label carrying U+FB01 forced into the glyph-count arm:

    folded    This face was measured to draw fi as one glyph, and this string
              contains it (asked of "january files", ...) — so the refusal ...
              did not fire ... a fixture that got through
    lowercase The pairs this face was measured to join are fi, fl, and none of
              them is in this string. So ... the list is short of this face
              rather than the fixture being short of the list

The wrong suspect, confidently.

### Item 2 · the floor's other edge

`INK_OWN_SHAPE_FLOOR` is a quarter of one enumeration, and the run already had
the upper bracket (`inkOwnRead` must stay above it). The lower one is
`INK_SUBJECT_KEYS` — the narrowest object the shape test must REJECT, now named
rather than written inline. `Math.floor(props / 4)` is under four the moment
`props` is, and both edges are reported as one bracket. The tail states the
bracket instead of the fraction: `476 properties on this build against the 476
Chrome/152.0.7977.83 enumerated, both above the 119 it takes to be shaped like a
computed style and that above the 4 keys of a subject read`.

**Break-test.** `props: 16` → floor 4 → "It has come down to the reject
population ... this is a bracket that has closed while every message stayed
silent, and the next key added to a subject read opens it."

### Item 3 · one flag, three readers

`everyBand` decided three things: which bands `bandTargetRead` wants a verdict
from (in bandtarget.mjs), how many bands the census divides by, and what the
census line CALLS that set (both in browser.mjs, spelled again). Three spellings
of one predicate is a join nothing makes.

`bandTargetCensus(bands, hasWrapper)` now returns one row per part — counter,
declared count, population phrase — derived with the same expression
`bandTargetRead` applies as `wanted`. The band-record shape stays browser.mjs's
and arrives as the predicate. Three tests in bandtarget_test.mjs hold the join,
including one that recomputes each population from the reading's own answer.

**Break-test.** `stretch` given `everyBand: true`:

    before  12 of the 20 bands in this grid did not get the button held to the
            wrapper it is stretched across — 8 did.        (and nothing else)
    after   twelve messages naming the part — "nothing decided the button
            stretched across the heading wrapper" — and then that line

---

## Stage C

### Item 4 · which of the two copies

The deletion report could say "the phrase, spelled 2 times in the file, is on
[40 42]" — a count, when the note's own argument turns on WHICH: prose outliving
its assertion is the rot, and a copy in live code is the lexer's blind spot.

`pinCodeOnly` decided, and decided in place — a line taken for prose entirely
comes back whitespace. `pinCopyNote` is a lookup into what that pass produced,
which is what keeps it from disagreeing with the search that reported the phrase
gone. Extracted as a function of two values, because its arms are reachable only
from a state neither consumer has ever been in — the move bandtarget.mjs and
`startupVerdict` were split out for. Five cases in
`TestPinCopyNoteNamesTheCopyThatSurvivesInCode`, each with what the sentence must
say and what it must not.

### Item 5 · a separating case from the wrong population

`silent[0]` was `.github/workflows/ci.yml` — a file nothing would ever cite. It
satisfies "reached and not citing" and is evidence for nothing:
`citationExemptVerdict`'s silent arm is about a file somebody wrote an EXEMPTION
for. A silent set that was all workflow files would read exactly like this one.

The population is measured rather than listed: the extensions the citing files
have, UNIONED with the extensions `citationExempt`'s own rows have — the second
matters alone, because an exemption's file has stopped citing by hypothesis. The
assertion is over the union; the EXAMPLE is drawn from the tighter set.

    before  486 of the 499 files opened cite nothing (e.g. .github/workflows/ci.yml)
    after   486 ... 430 of them a kind this check is about and 357 a kind
            citationExempt itself names (e.g. android/verify/gen.go)

**Break-test.** The kinds emptied: "not one of them is a kind anything in this
repository cites or is exempted for ... the walk has shown it can open a file,
which was never in doubt."

### Item 6 · the two relations `afforded` rests on

`afforded` and `affordedOpen` reach one reader — `themeNearMiss`'s `why` clause
— and were held over four sets somebody wrote down. The relations were not held
anywhere:

    afforded >= edits, always, and == edits when the crowding stopped the search
    affordedOpen exactly when no parent holds three leaves

930 sets generated from core.Theme's own leaf names, re-parented into shapes it
does not have: windows of one to six consecutive names under one synthetic
parent, and the same window split across two. Real names, and every shape an
accident of where the window fell.

And a census of which of the four endings each set reached, because both
relations are biconditionals and a relation asserted over a population that
never produces the state it is about passes for free:

    its own crowding stopped the search                    27
    no width crowds these names at all                    473
    the ceiling and the crowding stop in the same place    53
    the ceiling cost this set a wider threshold           377

**Break-tests.** `afforded = edits - 1` → the first relation. The upward search
never finding its crowd → the second, in the "reports open on a crowdable set"
direction, AND the ending census, which noticed the population had lost an arm.
`affordedOpen = false` → the other direction.

One injection did not fire and was not a defect: `afforded >= span` instead of
`> span`. `afforded == span` is unreachable — every pair under a parent is
within `span`, so a parent of three crowds AT span and the loop stops below it.

---

## Verification

Ten paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      replay + 20 mjs suites + BROWSER PASS
    android ./gradlew compileDebugKotlin --offline
    android ./gradlew :app:fetchComposeLayoutSources --offline
    GOOS=js GOARCH=wasm go build ./...

Six files changed, +936 −49:

    wasm/verify/browser.mjs                items 1, 2, 7, 8, 9
    wasm/verify/bandtarget.mjs             item 3
    wasm/verify/bandtarget_test.mjs        item 3
    internal/pinfixture/pinfixture_test.go item 4
    wasm/verify/checknumbering_test.go     item 5
    wasm/verify/themenearmiss_test.go      item 6

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

1. **(age 0 · value medium) The canary is a reading of the browser and its own
   premise is unmeasured.** `inkCanvasNameReached` rests on monospace's advance
   for a string differing from the platform face's by more than a LayoutUnit,
   and on this run it does — but by how much is not carried. If some face ever
   sits within a 64th of monospace for a badge's single digit, every run of that
   box reads as a skipped join and the message sends a reader to look for an
   unreachable family name that is reachable. `m.base` and `m.canary` are both
   in hand at the counter, so the narrowest separation the 28 runs achieve is
   one `Math.min` away — the same bracket `asked.canvasWidest` now carries for
   the join's own bound.
2. **(age 0 · value low) `INK_CANARY_FALLBACK` is a generic and the argument is
   about metrics.** The note says monospace is chosen because a fixed-advance
   face is unlike this grid's proportional ones. Nothing asks the browser what
   `monospace` actually resolved to, and a build that mapped it onto the same
   family the text is set in would make the canary compare a face with itself —
   silently, in the direction that reports every run as skipped. The face is one
   `CSS.getPlatformFontsForNode` away on a node already being read.
3. **(age 0 · value low) `inkFold` is applied to the string and not to the
   seeds.** `inkLigatureSeeds`' pairs are ASCII by construction today, so
   folding one side is enough. That is a property of the seed list rather than
   of the fold, and gen.go is where the list is written — a pair added there in
   a composed form would be looked for, unfolded, inside a folded string and
   never found. The fold is in browser.mjs and the seeds arrive over the
   transcript, so the two are one line apart and nothing joins them.
4. **(age 0 · value low) The census of `afforded`'s four endings is a Logf and
   its populations are not bracketed.** The check fails when an ending is
   reached by NO set, which is the direction that matters, and 27 of 930 is a
   thin arm that would still pass at 1. The generated population is a choice —
   windows of one to six — and a change to it could take an ending down to a
   single set without a message, which is the shape `INK_ROW_ROUNDING`'s census
   exists to refuse one directory over.
5. **(age 0 · value low) `kindList` sorts for the message and the assertion
   reads the map.** The failure names the kinds in a stable order and the
   decision — `kinds[filepath.Ext(path)]` — is a lookup, so the two cannot
   disagree today. What is unstated is why the population is EXTENSIONS: an
   exemption is about a file's content, `core/debug.go` and a generated
   `zz_gen.go` share a kind, and the tighter population the example is drawn
   from has exactly two rows behind it. A third exemption of some other kind
   would widen it silently.
6. **(age 0 · value low) `pinCopyNote` places copies by line and the lexer
   blanks by span.** A phrase inside a string literal on a line that also
   carries code comes back as "live", which is right about the line and not
   about the copy — the note then tells a reader to open a line where what
   survives is the literal, not the assertion. pinCodeOnly knows the span it
   blanked; pinPhraseLines knows the span the copy occupies; nothing compares
   them, and the two are one offset apart.
