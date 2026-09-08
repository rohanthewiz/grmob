# Session: a probe read at the wrong element, a rect two answers disagree about, and a recitation nobody counted

Session: https://claude.ai/code/session_01MQcEV7mAeEV4h1LywCdQhe
Date: 2026-09-08 (follows "a-copy-that-was-not-a-computed-value-an-extent-that-trusted-its-neighbour-and-a-permission-wider-than-the-claim")

## Ask

"Let's continue working the Next list items" — all seven, every one of them
raised last session, worked in order.

| # | age·value | item |
|---|---|---|
| 1 | 1·medium | `INK_PATH_PROPS` is twelve properties chosen by argument |
| 2 | 1·medium | Nothing asks which of these checks can be made to pass BY a defect |
| 3 | 1·low | The row above the x-height is ruled out by an argument with no number |
| 4 | 1·low | A clipped run reports a PASS for everything rect-shaped in the same breath |
| 5 | 1·low | `pinStripSpace` is a lexer's job done by a rule of thumb |
| 6 | 1·low | `citationVerdict`'s caller still chooses the reporter |
| 7 | 1·low | `bandRenderTallVaries` is two paths and the derivation is two assignments |

## Item 1 — the list stopped being a list, and the probe was read at the wrong element

The plan was to widen the comparison from INK_PATH_PROPS' twelve to every
computed property with a table of exceptions. Measuring first turned up
something else.

`inkProbePath(i)` is `root/N/i` — gen.go's probe is a white **Box** with one
**Text** child carrying the scanned node's Style, and the path names the Box.
That is the rect the screenshot scan reads, and it is not the element drawing
glyphs. `probeOwn` and `probeAncestry` were both read there.

    read against            properties read    properties that differ
    the probe's Box                     476                        47
    the probe's Text node               476                        22

Twenty-five of the forty-seven were the mix-up: display, flex-direction,
flex-shrink, font-size (16px against the label's 13), font-weight (400 against
700), line-height, height, block-size, all eight paddings, all eight radii. The
twelve-property comparison could not see any of it because all twelve happen to
be neutral on both. Read at the Text node those all agree — which is the
statement the probe's argument always wanted to make and had never made.

The remaining twenty-two are `INK_OWN_MAY_DIFFER`, in three families with a
reason each: `color` and the sixteen properties that follow it (the probe
repaints the ink black at the declared alpha), `background-color` (inkProbeFor
clears the run's own fill; only the count pill has one), and four used-size
leaves. Held in both directions — nothing outside the table may differ on any
box, and every entry must be spent by some box in the run.

The ancestry sweep moved with it, and gains the Box as an ancestor it now
measures rather than one it started from and left neutral by construction.

Break-tests: `font-variation-settings` on a label (fires by name — one of the
three properties the Next item predicted the twelve would miss, and the message
reports 476 compared), an unused permission added to the table, the probe read
at its Box again (fires with the six the widened comparison catches), the probe
text node at a path that is not there, and `will-change` on the probe's white
Box (invisible to the old sweep).

## Item 2 — the class named, and the values it is about held

The false negative last session was: widen the run rect by 6px AND paint
something in the extra columns, and `inkExtent` moves with the paint, agrees
with the rect and PASSES. Nineteen failures on both sides of the injection, and
only the surplused band's own message told them apart.

The general shape, written down where the checks are: **a check can be made to
pass BY a defect when the quantity it measures is derived from a value nothing
holds.** Ordering it after a check that happens to constrain that value makes
the PAIR sound and leaves the value unheld.

So the values are held. Every window the ink scan reads is a rect or an offset
that came from somewhere, and each is now a claim of its own made against the
layout, before any pixel:

    the label's run rect   starts at the leading edge of the label's own
                           content box (0.000 in all twenty, measured) and
                           ends inside it
    the same rect's width  against the advance the SAME face gives for the
                           same string, through the canvas already open for
                           the ascent
    the digit window       gen.go's badgePadLeft/Right against the padding
                           the BROWSER resolved on that element — Go's
                           arithmetic on the browser's rect, never compared

The containment bound is slack by construction on the stretched branches (up to
218px between the words and the end of the box), which is why the advance is
the answer that matters. Measured, the two agree to 0.0044–0.0137px, every one
positive and every one exactly `ceil(advance·64)/64 - advance` — the layout
rounding up to a LayoutUnit. So `LAYOUT_UNIT = 1/64` is the bound by
derivation rather than by measurement, held two-sided.

The control is the point of the item. Under the exact injection pair that
produced twenty indistinguishable runs last session:

    before   19 extent failures either way; the shipped run and the
             defect-passes run differ only in one band's message
    now      22 messages: the shadow named by item 1's widened comparison,
             and the 6px widening named on every band before a pixel is read

Break-tests: `runW + 6` (fires on the advance), `runX + 10` (fires on the
leading edge), the widening-plus-shadow pair (the control above), gen.go's
badgePadLeft moved 4px off the browser's, the ltr refusal branch, and a run
with no advance.

## Item 3 — the structural argument, with a number under it

`INK_ROW_ROUNDING`'s own comment says it separates the baseline end and
separates nothing at the x-height end, and that what rules out a row up there is
structural: it misses every round letter beside the ascender. No number.

Partition the run's columns into the ones whose ink reaches above the x-height
and the rest, and ask what fraction of THE REST has ink on a row:

    at the x-height line       0.708 to 0.947 across the twenty bands
    one device row above it    0.011 to 0.154

`INK_ASCENDER_SEPARATION = 0.43` is the midpoint, so the margin is equal on
each side, and it is held in both directions. The second direction is not
decoration: "a row above reads few round letters" is trivially true of a title
that has few, and the whole claim is that the two rows are looking at different
things — the break-tests reach that arm twice and the first arm once.

`INK_ASCENDER_PROBE = 2` because round letters are cut with an optical
overshoot: a partition taken one row up puts the tops of o, e, a and c on the
ascender side, which is exactly the up-to-15% the table above shows.

Break-tests: the partition taken at the line itself, the band top moved onto the
ascenders, the separator dropped to 0.10 (fires on the row-above arm at 12.8%),
and a fill nothing matches so every column reads as ink (the empty-partition
refusal).

## Item 4 — the tail recites what was asked

The OK line said twenty bands paint their words and their counts. Both numbers
were `BAND_RENDERS.length` — the size of gen.go's table — and neither came from
the work.

There was already one silent skip behind it. `if (r.label && ...)` drops the
whole ink scan for a band whose label node the mount does not have, and unlike
the badge and the heading wrapper a step above, nothing compared that against
what gen.go declared.

So: a label parity guard like the badge's; two counters incremented at the end
of each scan's own success path; a census holding them to what the transcript
declared; and a tail that recites the counters. The badge counter declines a
window with no columns, because the three verdicts below it all run and every
one is a statement about no pixels.

    the grid clipped at 800x665
      before   1 message, and a PASS for everything rect-shaped
      now      3: the grid guard, and "20 of the 20 bands that declare a
               label were not scanned for their words in their own ink — 0
               were", and the same for the 8 counts

Break-tests: `--window-size=800,665`, the label node absent from the mount (the
parity guard AND the census, which also reports the count the `continue` cost),
and the digit window collapsed to nothing.

## Item 5 — the normaliser tied to the set it was written for

`pinStripSpace` keeps a space only between two word characters. That is correct
for a free-form C-family syntax and wrong where an indent is a statement
boundary — and the only thing making it true was that `pinConsumers` happens to
hold a `.mjs` and a `.swift`.

`pinFreeForm` is that assumption, as a map from extension to the sentence saying
why the rule holds for that language, and every consumer's extension has to be
in it. A harness in Python fails by name rather than being quietly canonicalised
wrong.

And the second half the item named: within those languages there is one place a
run of spaces still means something — inside a string, character or template
literal. Both sides go through the same normaliser, so a cited phrase never goes
missing; what happens is that two sources differing only inside the literal
canonicalise to one string and either satisfies the row. The claim is quietly
weaker than it reads. No citation quotes a literal today, and the first one that
does is now a decision.

Break-tests: a `.py` consumer, a citation carrying a quoted literal, and a
`pinFreeForm` row with the reason deleted.

## Item 6 — the reporter choice, handed a recorder

`switch { case say == "": ; case fails: t.Error; default: t.Log }` was three
lines in a walk of the repository. Every sense in `citationSenses` fails, so the
Log arm had never run, and a `default` that called Error would have passed every
test in the file.

`citationReport(rep, say, fails)` takes a two-method `citationReporter` —
`*testing.T` satisfies it as it stands, asserted at compile time — and
`citationRecorder` remembers instead of reporting.
`TestCitationReportChoosesItsReporterFromTheVerdict` holds all four inputs,
including the pair the switch's ORDER decides rather than its conditions: an
empty sentence stays silent even where the sense fails.

Break-tests: the default arm turned into an Error (the exact edit the note said
would pass every test in this file — it fails now), and the silent arm dropped.

## Item 7 — two causes that arrived as one message

The list and the derivation stay two statements of one fact; the item's own
argument settles that trade. What was left is a diagnosis.

An unused permission had two causes and one message pointing at the derivation:
the assignments stopped making the change, or the path names no leaf of
`core.Theme` at all — a rename under it, or a typo here — in which case the
derivation is fine and the reader has been sent to the wrong place. Same shape
as pinfixture's deleted-versus-reworded pair.

`themeLeafPaths` is `themeDifferences`' walk with the comparison taken out, so
the two cannot disagree about how a path is spelled. A path that names no leaf
gets its own failure, and `themeNearMiss` narrows the answer rather than
printing the tier's seventy-one leaves — a wall being the thing item 4 is about.

    Typography.Caption.Lineheight     "Typography.Caption.LineHeight differs
                                      from it only in case, and is probably
                                      what was meant"
    Typography.Caption.LetterSpacing  "nothing under Typography.Caption differs
                                      from it only in case, and that prefix has
                                      71 leaves — so this is a name that was
                                      never right rather than one that has
                                      drifted"

Break-tests: both of those. Two earlier injections (a typo in place of a real
path, and the derivation stopping outright) were intercepted by the guards above
this one, which is the right order and is why the reachable case is a THIRD
entry rather than an edited one.

## The break-tests

**25 run: 24 that had to fail and did, and the control that is the point of
item 2.**

    item 1   font-variation-settings on a label; an unused permission;
             the probe read at its Box; its text node at a wrong path;
             will-change on the probe's Box                              5
    item 2   runW + 6; runX + 10; badgePadLeft 4px off; the ltr refusal;
             no advance                                                  5
             runW + 6 WITH the shadow — the injection pair that was
             indistinguishable last session, now named directly          1
    item 3   the partition at the line; the band top on the ascenders;
             the separator at 0.10; every column reading as ink          4
    item 4   the window at 665; the label node missing; the digit
             window collapsed                                            3
    item 5   a .py consumer; a citation quoting a literal; a
             pinFreeForm row with no reason                              3
    item 6   the default arm calling Error; the silent arm dropped       2
    item 7   a misspelt leaf; a name that was never one                  2

Item 2's pair is the unusual one again, and it has changed sides. Last session
it demonstrated the false negative: identical failure counts, opposite
meanings. This session it is the control that the false negative is gone — the
defect is named on every band, before a pixel is read, and the shadow that used
to hide inside the extent count is caught on its own by item 1.

## Verification

Ten paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    ios/verify/run.sh       11 OK lines
    android/verify/run.sh   5 OK lines
    wasm/verify/run.sh      replay + 20 mjs suites + BROWSER PASS (12 checks,
                            20 real bands over five shapes and four palettes,
                            20 of them scanned for their words and 8 for their
                            counts — counted where the scans ran — on three
                            rows inside a band whose own top reads three
                            quarters of the round letters where the row above
                            it reads a tenth, over a run rect held to its own
                            content box and to the advance the resolved face
                            gives for the same string, behind 11 antialiasing
                            probes resolving every computed property their
                            boxes do except the 22 with a reason, each of which
                            some box spends)
    android ./gradlew compileDebugKotlin --offline
    android ./gradlew :app:fetchComposeLayoutSources --offline
    GOOS=js GOARCH=wasm go build ./...

No new files. Five changed: browser.mjs, gen.go, checknumbering_test.go,
pinfixture.go, pinfixture_test.go.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

1. **(age 0 · value medium) The probe was read at the wrong element for as long
   as it has existed, and what found it was a measurement taken for another
   reason.** `inkProbePath` named the Box because that is the rect the
   screenshot scan reads, and the two reads that are about the DECLARATION
   quietly followed it. Nothing in this file distinguishes "the path of the box
   I am sampling" from "the path of the element whose CSS I am asking about",
   and both bands and probes now use paths for both purposes. The same
   confusion is available anywhere else a check reads a rect and a computed
   style from one path — the badge is a pill with digits inside it, and
   `r.badgeOwn` is read at the pill.
2. **(age 0 · value medium) `INK_OWN_MAY_DIFFER` is measured against one
   browser at one dpr on one machine.** 476 is what this Chrome enumerates; a
   newer one enumerates more, and the new properties join the comparison
   automatically, which is the whole point — and it also means the first CI run
   on another Chrome could fail on a property that differs for a reason nobody
   has thought about yet. That is the right failure and it is an unpleasant one
   to meet at a bad moment. Nothing records which Chrome the twenty-two were
   measured on, and the failure message cannot say "this browser exposes N more
   properties than the table was written against".
3. **(age 0 · value low) The advance check compares a canvas measurement with a
   layout measurement and the two are the same engine.** Two answers is better
   than one and it is not two independent answers: Chrome's canvas text
   measurement and Chrome's layout share a shaper, so a shaping bug moves both
   and this agrees. What it catches is arithmetic done to the rect after
   shaping — which is exactly the injected defect and exactly the class item 2
   is about — and what it cannot catch is the face resolving differently from
   what either of them thinks. `INK_PATH_PROPS`' font-smoothing pair is the
   nearest thing to a guard on that, and it is about rendering mode rather than
   about metrics.
4. **(age 0 · value low) `asked` counts two of the grid's claims and the tail
   recites five.** The tap target, the declared inset and the band's own fill
   are still recited off `BAND_RENDERS.length`, and the first two are rect
   assertions that genuinely do run for every band. The third is not: the fill
   is sampled inside `if (!gridClipped)` and skipped with everything else. So
   the census closed the two claims that had a silent skip behind them and left
   a third with the same shape, on the grounds that its skip is loud — which is
   the argument the label's skip also had until it turned out not to.
5. **(age 0 · value low) `pinFreeForm` classifies a language by its file
   extension.** That is a proxy standing in for a property of a grammar, and the
   test says so; what it cannot do is notice a `.mjs` file that is a template, a
   `.swift` file with a multi-line string literal in the region a phrase is
   quoted from, or a language whose extension is shared with one already in the
   table. The literal check catches the second of those at the citation and not
   at the source, so a source with a literal in it is still normalised by a rule
   that is wrong for that span — it just cannot be cited.
6. **(age 0 · value low) `citationReporter` is two methods of `testing.TB` and
   the loop above it uses `t` for more than two.** The extraction moved the arm
   that chooses a reporter and left the walk itself — `t.Errorf` for an
   unclassified sense, for an exemption that outlived its path, for a file that
   stopped citing anything — deciding its own reporter inline. Those are all one
   arm each, so there is nothing to get wrong yet, and the reason this one was
   worth extracting was a three-way switch rather than the principle.
7. **(age 0 · value low) `themeNearMiss` compares case-insensitively and calls
   anything further out "a name that was never right".** A transposition, a
   doubled letter, a singular for a plural are all typos it will report as
   deliberate. An edit distance would cover them and would need a threshold,
   which is another number with a measurement on each side of it — and the
   fixture for it would be a list of misspellings somebody made up, which is the
   shape of fixture this repository has twice decided is not evidence.
