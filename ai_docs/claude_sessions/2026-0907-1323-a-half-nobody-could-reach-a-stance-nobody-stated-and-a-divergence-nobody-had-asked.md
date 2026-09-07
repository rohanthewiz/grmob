# Session: a half nobody could reach, a stance nobody stated, and a divergence nobody had asked

Session: https://claude.ai/code/session_017k47JkhG1WBZR5nP29c7Ge
Date: 2026-09-07 (follows "a-void-that-held-things-a-walk-that-did-not-stop-and-a-brace-nobody-counted")

## Ask

"Take on the oldest 5 items."

| # | age·value | item |
|---|---|---|
| 1 | 2·low | The browser's value-range divergence half cannot fail on this machine |
| 2 | 2·low | `TestTheWidgetSwatchRingIsThePaletteRole` cannot see the swap it is named for |
| 3 | 2·low | `browser.mjs` requires the transcript and skips on no Chrome |
| 4 | 2·low | The widget swatches mount one tree per case and screenshot each |
| 5 | 1·medium | Nothing has asked a browser whether the band's insets survive overflow |

All five closed. Every one of them lives in `wasm/verify/browser.mjs`, which is
what makes the session read as one thing rather than five: **the browser pass
had grown a set of claims it could not exercise from a keyboard.**

Three of the five share a shape: *a decision whose only route in was a machine
in a particular state.* A divergence that fails only if Chrome changes its mind,
a stance that fires only on a machine missing a tool, a provenance that has a
pixel only under a theme nobody ships. Each was made into a function taking
values, and then handed the state no machine here can be in.

**The fifth was a prediction with nobody to ask.** It is now measured, and it
was right.

## Phase 1 — a half nobody could reach (item 1)

`browser.mjs` asserts the value-range table in two opposite directions: a row
whose numbers parse must agree with `core.Progress`, a row carrying an
unreadable number must **not**.

Only the first can fail. The second is a pinned divergence — Chrome reads
`aria-valuenow="half"` as 0 and pins the bar at the start where Go and Compose
read it as absent — so the branch that reports "the browser now agrees" had
never executed on any machine, and a dropped `!` was invisible to every pass.

The fix is the shape `localCopyGate` took last session: the decision's inputs
are two plain objects, so it moved out of the round trip and into
`valuerange.mjs` beside the table, as `valueRangeProblem(row, ax)`.
`valuerange_test.mjs` hands it all four answers.

**Writing the "agreeing browser" fixture is what found the sharp edge.** The
first version built it from the row's own resolved numbers:

```
   determinate      { value: now, min, max }               correct
   indeterminate    { value: 0, min: 0, max: 0 }           WRONG — those zeros
                    are core.Progress saying "not a position", and the arm
                    agrees on the *absence* of one. So the fixture handed the
                    check Chrome's disagreement and asserted it was agreement.
```

The test failed on its first run for exactly that reason, which is the only way
that distinction gets written down.

**One break-test missed and had to be sharpened.** Adding `ax.min === row.min`
to the indeterminate arm agrees with every row in the fixture by coincidence —
every indeterminate row's min is 0 and so is Chrome's. Bounds nothing would ever
report (`min: 42, max: 99`) are what make the omission observable.

`TestEveryReadingIsOneTheBrowserChecks` now reads `valuerange.mjs`. A substring
search of `browser.mjs` would find the four readings nowhere and report four
failures — or worse, find them in a comment and report none.

## Phase 2 — a provenance with no pixel (item 2)

`TestTheWidgetSwatchRingIsThePaletteRole` states in its own doc comment what it
cannot see, and that was the item: `Components.Input.BorderColor` holds the same
hex as `Colors.ControlBorder` in all three bundled themes, so a chip that had
started reading the field base — the exact drift `chipRing`'s argument was
written to prevent — renders an identical tone and passes everything.

**Both directions are invisible, and the reverse is the likelier one.** "The
role is the thing both of them name" reads like an instruction to use it
everywhere, and a field frame that took it would be equally undetectable.

The item's own suggestion was a fourth bundled theme, and rejected it in the
same sentence: a palette added to the framework for a test. A **throwaway**
theme does the same work — a copy of `DefaultTheme` with the role and the field
base set to two hexes nothing ships — because provenance has a pixel exactly
when the two authorities differ.

For that to be a statement about the widgets the browser mounts rather than
about a second pair built in a test, `gen.go`'s `widgets` slice became a
package-level `widgetBuilders` and its render loop became
`renderWidgetCase(name, theme, w)`. It returns an error where it used to call
`fatal`: `os.Exit` stops a `go run` correctly and takes the whole test binary
down doing the same job in a test.

Both directions break-tested, and the messages name the wrong authority by its
spelling rather than by its hex:

```
   quiet Chip says its boundary comes from Colors.ControlBorder and it drew
   Components.Input.BorderColor
```

## Phase 3 — a stance nobody stated (item 3)

Two preconditions, two opposite answers:

```
   no Chrome, no WebSocket    a fact about the MACHINE     SKIP, exit 0
   no GRMOB_TRANSCRIPT        a fact about the INVOCATION  FAIL, exit 1
```

Both right, and both were inline guards whose arms could only be reached by
arranging a machine that had the fault. On a laptop the FAIL arms never ran; in
a container without Chrome the SKIP arms did and the others did not. **The
mutation that matters is a swapped stance**, and it is the one that leaves the
pass green.

Extracted as `startupVerdict(...)` in `startup.mjs`, taking three booleans and a
string.

**The ordering turned out to be a third claim, and it had never been stated.**
The two faults can be true at once, and on a machine with no Chrome the wrong
order prints SKIP and exits 0 — so the person who forgot the variable never
learns they did. Deciding the invocation fault first is now asserted directly.

Item 5 gave the gate a third precondition (an empty band table) on the way past,
which is what a stated rule buys over two inline guards.

## Phase 4 — six screenshots that became one (item 4)

Each widget swatch was a mount, a rects call, a screenshot and a PNG decode.
Six of them, and the Input swatch had doubled that a session ago.

`gen.go`'s trees are already pages — a Box in the theme's own `Colors.Background`
with 24px of padding and a 240px width — so they lay out as a grid without
being touched, which is the whole point: a tree this harness had edited would be
a different widget.

**The cost the item named is real and it is smaller than it sounds.** A theme's
page fill is now a sibling's fill rather than the document's. No sample was ever
reading the document: each is taken inside a rect the browser reported for the
node that declares the colour.

**One thing the single-mount version never had to think about.**
`captureBeyondViewport: false` means a swatch below the fold has a rect and no
pixels, and `pixelAt` answers `null` for every sample — three failures naming
colours, none of them saying the swatch was never on screen. Said once, in its
own guard, and break-tested by setting the grid one wide:

```
   MaterialTheme/Input frame: the swatch grid runs past the bottom of the
   viewport (this one ends at 551px of a 513px screenshot)
```

Three per row is 720px of an 800px window. The whole `run.sh` went from 3.7s to
2.9s.

## Phase 5 — a divergence nobody had asked (item 5)

`ios/verify/band.swift` records that `components.GroupHeader`'s two inset
arrangements divide an overflow deficit differently, and ends with a sentence
about somebody else's software:

> CSS distributes shrink over the inner flex base size rather than the outer
> one, so a browser may well agree where this does not, and nothing in this
> repository has asked one.

Now one has. `internal/bandfixture` rides in `wasm/verify`'s transcript, and
`browser.mjs` lays out every case in both arrangements at every offer — one
mount, one round trip of rects — and compares the band's width, the label's
leading edge and room, the badge's leading edge, and the band's height.

**The prediction was right, to the LayoutUnit.**

```
   120px offered, a 100px label and a 24px badge     the label's room

     GrMobFlexSolver, insets on the control              63.14
     GrMobFlexSolver, insets on the Row                  64.52
     Chrome, both arrangements                           64.52
```

So the difference `ios/verify` records is a genuine cross-target divergence —
one renderer's shrink proportion counts a child's padding and the other's does
not — rather than an artefact of either. Both targets now assert their own
answer in their own direction.

**There is no separate overflow arm in the browser check, and that is the
finding.** The identity that holds with slack goes on holding without it.

### Three things the first version got wrong, each caught by the pass

```
   the offer as a width      a Row is a content box (this runtime writes no
                             box-sizing), so declaring Width: 120px hands the
                             arrangement with padding on the Row a band 32px
                             wider than the other. The offer has to become an
                             inner extent first — which is band.swift's own
                             `inner = offer - chrome`, the proposal a SwiftUI
                             Layout receives after its container's padding is
                             removed. 34 problems reported.

   min-width: auto           CSS will not shrink a flex item below its own
                             min-content width, and the fixture's label is a
                             box with a DECLARED width. Nothing shrank, both
                             arrangements sat at their natural width, and they
                             agreed by never reaching the arithmetic. A real
                             band's label is text, whose min-content is its
                             longest word, so clearing the minimum is the
                             faithful model rather than a convenience.

   an unobservable align     AlignItems was carried from the fixture and
                             nothing could see it: switching it to `stretch`
                             changed no measurement. It is observable in the
                             control's HEIGHT — a stretched control is as tall
                             as the line, a centred one as tall as its own
                             content plus insets — and the fixture's
                             deliberately oversized badge is the case that can
                             tell the difference.
```

**Two controls, because both halves of the overflow arm can pass by having no
subject.** The label must end up with less room than it asked for, and the badge
must have shared the deficit — the second because `band.swift` divides it
between both children, and a badge pinned at its own width would leave the two
passes solving different problems.

## The break-tests

**20 caught, 2 misses, 0 restore failures.**

    item 1   the divergence half's `!`, the indeterminate arm's bounds,
             axRange's role, a missing bar, the no-arm fallthrough        5
    item 2   a chip reading the field base, a field reading the role      2
    item 3   the transcript stance as a skip, the machine decided first,
             an empty band table accepted                                 3
    item 4   the chip's ring dropped (all three themes, both rows), the
             path arithmetic transposed, a grid below the fold            3
    item 5   min-width dropped, the offer not converted, the Row's gap
             zeroed, align-items stretch, the pairing off by one, the
             oversized badge shortened, the badge pinned at its width     7

Two mutations are invisible and both are worth stating rather than fixing:

- **Deleting the shrink control *and* `min-width: 0` together.** A control
  assertion cannot guard its own deletion; this is the same shape as the scroll
  control in check 3 and the scroller control in check 6.
- **Deleting the label-room comparison.** Only invisible because it is logically
  equivalent to the `grew` assertion beside it: the insets are constant per
  arrangement, so a label-room divergence *is* a control-width divergence, and
  a browser that started behaving like the SwiftUI solver fails there instead.

## Verification

Eight paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    wasm/verify/run.sh          replay + 17 mjs suites (15 + the two new) + BROWSER PASS
                                (20 swatches, 6 real widgets on both edges on
                                 one page, a sticky band, 22 value ranges,
                                 3 bands x 2 arrangements x 4 offers)
    ios/verify/run.sh           flex + stack + 3 band arrangements
                                + 15 picker menus + replay + view + app
    android/verify/run.sh       15 picker menus and 22 value ranges, on a JVM
    android ./gradlew compileDebugKotlin --offline
    GOOS=js GOARCH=wasm go build ./...

New files:

    wasm/verify/startup.mjs             the two preconditions and their stances
    wasm/verify/startup_test.mjs        every answer, including the order
    wasm/verify/valuerange_test.mjs     the verdict's four answers

Changed:

    wasm/verify/browser.mjs             the widget grid, the band check, the
                                        gate call, the verdict call
    wasm/verify/valuerange.mjs          axRange, axAgreesWithGo,
                                        valueRangeProblem
    wasm/verify/valuerange_test.go      the reading pin repointed
    wasm/verify/gen.go                  widgetBuilders, renderWidgetCase,
                                        ratioBetween's error, the bands table
    wasm/verify/widget_test.go          the split-theme provenance census
    wasm/verify/run.sh                  the header's list of claims
    internal/bandfixture/bandfixture.go the web's answer, on SharesADeficit
    ios/verify/band.swift               the note that had no browser

## Docs

`docs/platforms/wasm.md`: the two startup stances and their order, the
one-page swatch grid, the provenance test, the extracted value-range verdict,
and a new section on the band's insets under overflow with the three-row
comparison table. `docs/platforms/native.md`: the browser's answer, in the
band's own bullet. `ROADMAP.md`: six entries.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here. Value is the payoff of doing it, not the effort: `high` means
something is being worked around today or a second consumer has arrived,
`medium` means it blocks one named thing, `low` means it is a gap nobody has
bumped into yet.

1. **(age 2 · value medium) The disclosure band's control does not grow.** The
   collapsible band puts `FlexGrow(1)` on the heading wrapper and the insets on
   the *button* inside it, so whether the tap target spans the band depends on
   whether a non-growing child fills a growing parent — a cross-axis question
   `GrMobFlexSolver` does not answer. `browser.mjs` can now answer it: it
   mounts bands and measures rects, and the plain band went in this session.
   The picture in `bandInsets`' doc comment assumes the answer.
2. **(age 2 · value low) `core.ComponentDefaults.Text` is a field nothing
   reads.** `core.Text` builds its Style from its own props and never touches
   the theme, so the two bundled themes that state a `Components.Text` are
   describing a widget that does not exist. What it costs is a theme author who
   fills it in and sees nothing happen.
3. **(age 2 · value low) The two-table coupling in `mobile/verify` is only as
   total as `bindableGoTypes`.** `error` is in it because a bind was run; the
   next type gobind can carry and this bridge does not use is in exactly the
   position `error` was. Nothing enumerates what gobind binds.
4. **(age 2 · value low) `internal/bandfixture`'s content sizes are made up.**
   That is right for the arithmetic and it means the height agreement is
   checked against *insets* rather than against text — the other half, that a
   bold caption is no shorter than a plain one, is stated and unmeasured.
5. **(age 2 · value low) The band check models one renderer.** It now models
   two — the SwiftUI solver and a browser — and Compose is still the third.
   Compose applies padding through a `Modifier` and distributes weight through
   its own `Row`, so the same question has an answer nobody has asked;
   `android/verify` is a JVM pass and could.
6. **(age 1 · value medium) A Spacer's children overflow on three targets and
   are measured into the size on Compose.** `Modifier.size` sets min and max,
   so a child taller than the void is squeezed where CSS and SwiftUI let it
   spill. It is not a Spacer property — it is what every fixed-size container
   does on that target — which makes it the more interesting question: nothing
   anywhere has asked whether the four targets agree about overflow for *any*
   fixed-size box.
7. **(age 1 · value low) The descending nested-composite pairs are reachable
   and have no example.** `keynav_test.mjs` builds a `listbox > tablist >
   option` tree because it is the smallest thing that tells the two rules
   apart, and the doc comment calls it contrived. Whether any real screen can
   produce one is unasked — if none can, the five descending pairs are a
   finding nobody will ever see.
8. **(age 1 · value low) `swiftTypeBody`'s anchor is still a substring.** The
   cut is syntactic now; finding the declaration is `strings.Index`, so an
   anchor matching a mention of the type in a doc comment above it would cut
   from there. Every current anchor starts with `private struct` or `struct`.
9. **(age 1 · value low) `core.CompositeMemberRole` returns "" for two
   different reasons.** A toolbar has a keyboard and no member role; a
   `RoleHeading` has neither. The doc says callers separate them by asking
   `KeyboardComposites` first, which is a contract nothing enforces.
10. **(age 1 · value low) `localCopyGate` is the only gate of its kind that is
    tested.** `startupVerdict` is now the second, and it was extractable for
    the same reason — its inputs are values. `ios/verify` skips on a missing
    iPhoneOS SDK and `android/verify` on a missing Kotlin compiler, both by the
    same argument and both as inline shell conditions no test reaches. Those
    two are `command -v`.
11. **(age 0 · value high) `core.FlexShrink(0)` does nothing on any target.**
    `Style.Merge`, `htmlout.Export` and the WASM runtime all guard on
    `FlexShrink != 0`, so zero means unset — and CSS's default is 1. "Do not
    shrink this item" is unexpressible, which is a real hole in the vocabulary
    rather than a gap in a harness. It was found by a break-test that could not
    break: the mutation `FlexShrink: 1 -> 0` in the band fixture changed
    nothing at all.
12. **(age 0 · value medium) The sticky fixture's `FlexShrink: 0` is inert, and
    its comment says why it matters.** `browser.mjs`'s `STICKY` puts it on the
    `List` with the reasoning that without it there would be nothing to scroll
    and the check would pass by having no subject. Item 11 is why it is inert;
    what makes this its own entry is that removing it changes nothing, so the
    fixture has been overflowing for a different reason than the one recorded,
    and the control assertion beside it is the only thing that has been keeping
    the check honest.
13. **(age 0 · value low) The two shrink controls in the band check cannot
    guard their own deletion.** Dropping the control *and* `min-width: 0`
    together leaves both arrangements at their natural width, agreeing by never
    reaching the arithmetic, and nothing reports it. The same is true of the
    scroll control in check 3 and the scroller control in check 6, which is
    either an argument that this is simply what a control is, or an argument
    for a source-level pin over the three of them.
14. **(age 0 · value low) The band check mounts only definite offers and
    `max-content`.** `bandfixture`'s negative offer is SwiftUI probing for an
    ideal size and `max-content` is read as the CSS analogue, which is a
    judgement stated in a comment and not held to anything. `min-content` is
    the other candidate and would be a different question.
15. **(age 0 · value low) `browser.mjs`'s check numbering is not a sequence.**
    The header lists eight and the body has two sections numbered 6. It has
    been wrong since the widget swatches were added and it is now one more.
