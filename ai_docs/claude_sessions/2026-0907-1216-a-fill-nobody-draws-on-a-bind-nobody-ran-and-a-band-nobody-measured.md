# Session: a fill nobody draws on, a bind nobody ran, and a band nobody measured

Session: https://claude.ai/code/session_017k47JkhG1WBZR5nP29c7Ge
Date: 2026-09-07 (follows "a-command-nobody-ran-two-columns-nobody-reached-and-a-browser-nobody-asked")

## Ask

"Do the oldest 5 Next list items."

| # | age·value | item |
|---|---|---|
| 1 | 3·low | A `Header` override cannot be told the edge was withheld |
| 2 | 3·low | `internal/palette` reflects and `core` cannot tell it a fill is reachable |
| 3 | 2·low | The palette check paints one widget per theme |
| 4 | 2·low | The two-result gobind arm is transcribed nowhere |
| 5 | 1·medium | The band's insets are on the control and only the web has been asked |

All five closed. Three of them (2, 4, 5) turned out to share a shape the last
session's five did not: **a claim that was true and had never been checked
against the thing it was about**. The exclusion tag was a claim about geometry
with no complement, the two-result refusal was a claim about Clang with no
Clang, and the band's insets were a claim about flex with only one flex engine
asked. Each wanted a different instrument — a second tag, a real
`gomobile bind`, and a solver that was already sitting there.

## Phase 1 — two facts a band was never handed (item 1)

`GroupedList.AutoLoadWithheld()` answers the caller, who owns the footer. A
`Header` override is a different reader in a different place: it is handed a
`Group` and nothing else.

`Group` grew two fields, and the second one was not the item's:

```
   Trailing           this is the last run, the one an append pager extends
   AutoLoadWithheld   this run being shut is why the list has no edge sensor
```

`Trailing` is the one that turned up on the way. `HideTrailingCount`'s own doc
says an override "owns the decision itself" — and the decision is *do not
publish an open run's count*, which needs to know which run is open. Nothing
handed to the override said. Deriving it meant re-walking `Items` with the same
`GroupBy` the widget had just walked.

Stamping it in `groupRuns` (and in `trailingRun`, which is the other walk)
also removed a duplicated derivation: the band's count rule was `ri ==
len(runs)-1` and the sensor's was a separate walk back from the end, forty
lines apart in one function. One statement now.

The composite is the reason the widget states `AutoLoadWithheld` rather than
letting a band derive it:

```
   Trailing  &&  Collapse.hides(run)  &&  OnEndReached != nil
                 └─ needs OnToggle as well as IsCollapsed
```

A caller with a predicate and no handler hides nothing, so a band deriving it
from `shut[g.Key]` alone would announce a pause the list is not taking.

`rowsSpec`'s census refused the new field until it was classified, which is
what it is for. The admission test asks whether a knob can be done to
`appendRows`' *returned slice* instead; the two existing single-owner fields
fail it because one needs the item and one needs rows not to be produced. This
is a third reason, and it is the only one not about rows at all: **it needs to
reach a band's input.** The `Group` is consumed while the children are built,
and the children come back opaque.

## Phase 2 — the fill nobody draws on (item 2)

`notbackdrop` said a fill is never drawn on. The absence of one said nothing —
so the census was the boundary tone crossed with whatever fills the struct
happened to carry, which is a set with no claim behind it and two costs:

```
   a pair nothing builds is measured beside a pair three widgets build,
   at the same weight, and a shortfall in either reads identically

   a notbackdrop tag can be DELETED with no consequence but a pair
   quietly joining the census
```

The second is the sharper one and was already on the Next list (item 11, age
0). Dropping `Camera`'s tag adds a pair that clears 6:1: the run stays green
while a geometry claim has been thrown away. `Button`'s is load-bearing by
accident — `Primary` as a fill is 2.17:1 and the census fails — so half the
exclusions were defended by their own numbers and half by nothing.

The fix is a second tag, exclusive and total. `palette.Untagged()` reports a
field carrying neither *or both*, and the census refuses to run in that state,
so deleting either tag is now the same event as deleting a field's name. Both
values are required to be arguments, and the reachability claim travels on into
the census's failure message, which is the difference between

```
   ControlBorder is 2.9:1 on Card fill
   ...and a FormField's frame inside a Card is that pair
```

**Classifying `Text` cost the census two rows, and that is the payoff.**
`core.Text` is a leaf — content and style props, never children — so nothing
can be nested inside one and no boundary can land on its fill. Two bundled
themes give it a white `Background` and the census was dutifully measuring
against it. `grep` for `Components.Text` finds no reader anywhere: it is a
component default nothing in the framework reads, and it was contributing
pairs.

One test was retired with its argument. `TestTheBackdropExclusionsNameRealFills`
required a `notbackdrop` field to state a fill in some theme, on the grounds
that an exclusion for a fill-less field would be inherited by the next theme to
fill it. That does not survive totality: a tag on a fill-less field is now a
decision made *for that field* and is supposed to survive the field gaining a
fill — which is exactly what `Column` and `Row`'s `backdrop` tags are.

## Phase 3 — the other half of the tone's job (item 3)

The premise turned out to be half wrong, which changed what the check is for.
A quiet `components.Chip` is a tappable control and exports as a **`<button>`**
— the first member `borderResetTypes` ever had. So "a real widget on a tag the
user agent draws a border on" was already covered.

What a `core.Input` adds is two things a chip cannot say:

```
   a second authority   chipRing reads Colors.ControlBorderColor (the role);
                        core.Input reads Components.Input.BorderColor (a
                        literal, pinned to the role separately, because a
                        core.Style is a value and cannot call a resolver)
   a second tag         <button> is given `outset`, <input> `inset`, and an
                        <input> arrives with a fill and padding of its own
```

`widgetCase.RingFrom` names the authority per case, so a case whose spelling
the Go test does not know is a failure rather than a silent borrow of its
neighbour's.

And **both** horizontal edges are scanned now, not one. The break-tests are
what separated the two reasons:

```
   border-style dropped (UA `inset` in force)   caught by either edge alone
   border drawn on three sides                  caught by neither, unless
                                                both are read
```

One edge in the declared tone says the tone survived. It says nothing about
whether the box was closed.

## Phase 4 — somebody ran the bind (item 4)

The old refusal was right about what it did not know and right that guessing
was the one thing that must not happen. It was also the whole problem: the next
bridge function of that shape was blocked on somebody having a Mac.

This machine has one. A package with one function and one interface method of
every result shape was bound with `gomobile bind -target=ios`, and the module
read back through `swiftc`. Every row is that:

```
   Go results        package func                          interface method
   error             (_ error:) -> Bool                    throws
   (string, error)   (_ error:) -> String                  (error:) -> String
   (Iface, error)    (_ error:) -> MobileXProtocol?        throws -> MobileXProtocol
   (int, error)      (_ ret0:, _ error:) -> Bool           (ret0_:) throws
   (bool, error)     (_ ret0:, _ error:) -> Bool           (ret0_:) throws
```

**The prediction was correct.** No package function throws: gobind emits every
one as a C function and Clang's error convention is the Objective-C *method*
one.

**And a method throws — except once.** The convention needs a return it can
signal failure through: `BOOL`, or a nullable object. A `(string, error)`
method returns `NSString* _Nonnull`, which is neither, so it keeps its explicit
`error:` parameter and does not throw. A `(Iface, error)` method throws *and
loses the optional*, because `nil` is now the error signal and can no longer
also be a value. Nothing that could be read anywhere would have produced that.

Two shapes are still refused, and both refusals are gobind's, verified by
running it: `too many result values`, and `second result value must be of type
error`.

**The bind also found a hole.** `error` was in neither `bindableGoTypes` nor
`gobindSwiftTypes`, on the reasonable-looking grounds that nothing in `mobile`
returns one. gobind binds it. So a bridge function that grew an `error` would
have been judged unbindable here, gone undeclared in the stub, and taken
`GomobileBridge.swift`'s type-check with it — the exact hole the whole file
exists to close, one level down. The two tables answer for each other now.

The stub's checked rows changed shape with the arms: they used to be a result
*count* and a phrase, which was the right shape while everything was refused.
An error result changes the **parameter list** as well as the return, so a row
that named only the tail would be silent about the `NSErrorPointer` — the half
a shell author actually types. A row is now a whole declaration, built by the
same two functions that build the stub's own.

## Phase 5 — asking the solver whether the move was free (item 5)

`ios/verify` carries `GrMobFlexSolver`, split out of the SwiftUI `Layout`
precisely so it can be exercised without mounting anything. That makes it the
one thing here that can answer *did this layout change move any pixels?* — and
the band's insets are the change it was never asked about.

`internal/bandfixture` reads the real `GroupHeader`'s geometry off a rendered
node (insets, gap, flex-grow, alignment) and derives the arrangement it came
from by a stated rule. Three fixture bugs were found by the check failing, and
each was a real asymmetry rather than a typo:

```
   the trailing inset   is the gap before the badge when there is one, and the
                        band's own trailing inset when there is not
   the inter-item gap   resolve() returns what justify-content distributed;
                        the arrangement's own spacing is consumed before growth
                        and is not in it
   the control's box    is SUPPOSED to differ — its leading edge is 0 in one
                        arrangement and 16 in the other, which is the move
```

The answer is **yes, with two recorded exceptions**, and finding them is what
the pass was worth.

**Under overflow the two arrangements diverge.** The solver shrinks each child
in proportion to its base, and a base includes that child's own padding — so
the control's 32 points are inside the proportion in one arrangement and
outside it in the other:

```
   120pt offered, a 100pt label and a 24pt badge
     insets on the control   base 124 of 148 -> 87.14 -> label 63.14
     insets on the Row       base 100 of 124 -> 64.52 -> label 64.52
```

It needs a *second child* to show: with the count hidden the control is alone
on the line and is clamped to the container either way. So the badged case
diverges and the unbadged one does not, the fixture states which is which, and
both are asserted — an "it diverges" with no case that does not would be a
claim about whatever happened.

CSS distributes shrink over the *inner* flex base size rather than the outer
one, so a browser may well agree where this does not. Nothing here has asked
one; that is stated as an open question rather than as a claim.

**A badge taller than the control would change the band's height.** With
children centred, a Row's height is its tallest child plus its own vertical
padding, so moving that padding onto one child stops it being added to the
other. It cannot happen to the real band — the control carries 4+4 of vertical
padding against the badge's 2+2, and both wrap the same caption type — so it is
a case with a made-up badge, asserted in the opposite direction so that the
agreement of the real ones is not holding for a reason nobody stated.

The disclosure band is deliberately out of scope and is a different question:
its insets are on a **button one level inside** the Row's growing heading
wrapper, so what wants answering there is whether a non-growing child fills a
growing parent, which a main-axis distributor is not what asks.

## The break-tests

**25 caught, 1 miss (a known limit, now in a second place), 0 restore failures.**

    item 1   both stamps, the pause on every band, the forward, HideCount   5
    item 2   both tags, both at once, a bare marker, the roles, the key     6
    item 3   an inset frame, a three-sided frame, RingFrom, a dropped case  4
    item 4   a label, a throws, an optional, a row, the bindable set        6
    item 5   Rewind, the widget's insets, the spacing, the alignment        4

Two of those only became catchable because the break-test said so, and both
gaps were real:

- **`bindableGoTypes` could lose `error` silently.** Nothing in `mobile` uses
  one, so the entry had no exerciser. The fix is not a fixture but a coupling:
  the set that decides which functions need a stub declaration and the table
  that decides how their types are spelled now have to be the same set, and the
  failure message says which direction the hole opens.
- **A band that stopped centring was invisible.** `band.swift`'s height model
  *assumes* centred children — under `stretch` none of it holds — and the
  fixture was not carrying the alignment. It is now, Go pins it, and the Swift
  refuses a case that is not centred rather than answering for one.

The one miss is the provenance limit `TestTheWidgetSwatchRingIsThePaletteRole`
already documented, now demonstrated in the other direction: pointing the field
swatch's `RingFrom` at `Colors.ControlBorder` passes, because
`Components.Input.BorderColor` holds the same hex in all three bundled themes.
The check catches a tone that is *neither*; it cannot catch a case that names
the wrong one of two identical ones.

## Verification

Eight paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    wasm/verify/run.sh          replay + 17 mjs suites + BROWSER PASS
                                (20 swatches, 6 real widgets on both edges,
                                 a sticky band, 22 value ranges)
    ios/verify/run.sh           flex + stack + 3 band arrangements
                                + 15 picker menus + replay + view + app
    android/verify/run.sh       15 picker menus and 22 value ranges, on a JVM
    android ./gradlew compileDebugKotlin --offline
    GOOS=js GOARCH=wasm go build ./...

New files:

    internal/bandfixture/bandfixture.go       the band's geometry, both ways
    internal/bandfixture/bandfixture_test.go  its pins to the widget
    ios/verify/band.swift                     the solver's answer

Changed:

    components/grouping.go             Group.Trailing, .AutoLoadWithheld,
                                       both walks stamping the first
    components/grouped_list{,_test}.go rowsSpec.AutoLoadWithheld, one predicate
                                       call, and the band tests
    components/rows_spec_test.go       the third admission reason
    components/variant_test.go         the totality and reachability guards
    core/theme.go                      the `backdrop` tags and their doc
    internal/palette/palette.go        IsABackdrop, Untagged, Backdrop.Why
    ios/verify/{gen.go,main.swift,run.sh}  the band cases in the transcript
    ios/verify/gomobile_stub.swift     the verified result rows
    mobile/verify/gomobilestub_test.go swiftResults, the error type, the
                                       two-table coupling
    wasm/verify/gen.go                 the Input swatch and RingFrom
    wasm/verify/widget_test.go         per-case authority, both widgets
    wasm/verify/browser.mjs            both edges, and why
    wasm/verify/palette.mjs            the two Text-fill rows, gone

## Docs

`docs/concepts/styling-and-theming.md`: the tag pair, what the second one
bought, and the two widgets. `docs/platforms/wasm.md`: two spenders, two tags,
three sampling lessons. `docs/platforms/native.md`: the verified result table
(replacing a `throws` claim that was wrong), and a new section on asking the
solver whether a change was free. `docs/components.md`: `Group.Trailing` and
`Group.AutoLoadWithheld`. `ROADMAP.md`: five new entries.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here. Value is the payoff of doing it, not the effort: `high` means
something is being worked around today or a second consumer has arrived,
`medium` means it blocks one named thing, `low` means it is a gap nobody has
bumped into yet.

1. **(age 2 · value medium) A hand-assembled `Spacer`'s children are dropped on
   both natives.** Both DOM renderers emit a Spacer's children like any other
   element's; a Compose `Spacer` and a `Color.clear` are leaves.
2. **(age 2 · value medium) The nested-composite finding is reported in Go and
   asserted in JavaScript, and the two are not held together.**
   `core.KeyboardComposites()` is pinned to the runtime's tables; the *walk's
   stopping rule* is not.
3. **(age 2 · value low) `swiftTypeBody`'s cut is "the first closing brace in
   column one".** True of every Swift type in `Renderer.swift` and not true of
   Swift.
4. **(age 2 · value low) `notTappable`'s reasons are prose by construction.**
   The four kinds it sorts roles into are almost a derivable property, the way
   the refusals table's `Nesting` now is.
5. **(age 2 · value low) The stale-download skip is a guard nothing
   exercises.** Its subject — `spec.Parse` refusing 1.1 — is covered; the skip
   itself has never run.
6. **(age 1 · value low) The browser's value-range divergence half cannot fail
   on this machine.** A row with an unparseable number must *not* agree with
   `core.Progress`, and it does not — so the assertion has never fired and a
   mutation of it is invisible.
7. **(age 1 · value low) `TestTheWidgetSwatchRingIsThePaletteRole` cannot see
   the swap it is named for**, and now in both directions: the field swatch can
   claim `Colors.ControlBorder` as its authority and pass, because the role and
   `Components.Input.BorderColor` hold the same hex in all three bundled
   themes. A fourth bundled theme that split them would sharpen the whole
   family — and would be a palette added for a test.
8. **(age 1 · value low) `browser.mjs` requires the transcript and skips on no
   Chrome.** Two preconditions with two stances, both right and neither obvious
   from outside.
9. **(age 1 · value low) The widget swatches mount one tree per case and
   screenshot each.** Six Chrome round trips now rather than three, because the
   Input swatch doubled them — each page paints its own background, so a single
   page carrying all of them would halve the browser pass's slowest section at
   the cost of the per-theme fill being a sibling rather than the document.
10. **(age 0 · value medium) Nothing has asked a browser whether the band's
    insets survive overflow.** `ios/verify` says the two arrangements divide an
    overflow deficit differently, because this solver shrinks in proportion to
    a base that includes the child's own padding. CSS distributes shrink over
    the *inner* flex base size, so a browser is likely to agree where SwiftUI
    does not — which would make it a genuine cross-target divergence rather
    than an artefact. `wasm/verify` already mounts trees and measures rects;
    this is one more.
11. **(age 0 · value medium) The disclosure band's control does not grow.** The
    collapsible band puts `FlexGrow(1)` on the heading wrapper and the insets
    on the *button* inside it, so whether the tap target spans the band depends
    on whether a non-growing child fills a growing parent — a cross-axis
    question `GrMobFlexSolver` does not answer and no target has been asked.
    This is the same argument `bandInsets` makes, one level out, and the
    picture in that doc comment assumes the answer.
12. **(age 0 · value low) `core.ComponentDefaults.Text` is a field nothing
    reads.** `core.Text` builds its Style from its own props and never touches
    the theme, so the two bundled themes that state a `Components.Text`
    (padding 12, radius 6, a white fill — the shape of a `TextArea` base) are
    describing a widget that does not exist. It is now tagged `notbackdrop`, so
    it costs the census nothing; what it still costs is a theme author who
    fills it in and sees nothing happen.
13. **(age 0 · value low) The two-table coupling in `mobile/verify` is only as
    total as `bindableGoTypes`.** `error` is in it now because a bind was run;
    the next type gobind can carry and this bridge does not use is in exactly
    the position `error` was. Nothing enumerates what gobind binds, and the
    only way to find out is to bind something.
14. **(age 0 · value low) `internal/bandfixture`'s content sizes are made up.**
    That is right for the arithmetic and it means the height agreement is
    checked against *insets* rather than against text: the control is padded
    more than the badge, which is half of why it is the taller child, and the
    other half — that a bold caption is no shorter than a plain one — is stated
    and unmeasured.
15. **(age 0 · value low) The band check models one renderer.** Compose applies
    padding through a `Modifier` and distributes weight through its own
    `Row`, so the same question has a second answer nobody has asked;
    `android/verify` is a JVM pass and could.
