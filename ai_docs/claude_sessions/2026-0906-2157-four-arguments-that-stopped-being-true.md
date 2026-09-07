# Session: a shortfall that was cheaper to fix than to defend, and three rules nobody could see working

Session: https://claude.ai/code/session_01YVcrPiEYG2vnpUePWiK5T5
Date: 2026-09-06 (follows "the-four-oldest-again")

## Ask

"Work on the four oldest items in the Next list."

| # | age·value | item |
|---|---|---|
| 1 | 5·low | `Colors.ControlBorder` is still 2.92:1 against `Colors.Surface` |
| 2 | 4·medium | `Margin`/`Padding` cannot say "clear this side" ahead of an axis prop |
| 3 | 4·low | `rowsSpec` carries two single-widget knobs and a test that fails at three |
| 4 | 3·medium | `declaredInk` is unobservable under both bundled themes |

All four closed. Three of them are the same shape as last session's — an
argument that had stopped being true — but the *way* they stopped differs, and
item 1's way is the interesting one: **the argument was still sound; what
changed is that the alternative got cheap.**

Item 1 was a look decision, so it was put to the user rather than made here.

## Phase 1 — the census retired the shortfall it was built to record

`Colors.ControlBorder` is iOS systemGray `#8E8E93`: 3.26:1 on `DefaultTheme`'s
page, 2.92:1 on its `Surface`, which is the quiet chip's own fill. Last
session built the census that measures every (tone, fill) pair and recorded
this one in `knownBoundaryShortfalls` with six lines of argument.

The argument was right. The edge that identifies a pill is the outer one — the
fill is 1.12:1 against the page and identifies nothing — so the inner pair is
a boundary between two parts of one control, and the outer pair always cleared.

What retired it is not a better argument. **It is that defending the pair used
to be less work than fixing it.** Evaluating a candidate tone meant going and
finding every fill a boundary lands on; that is precisely what the census now
is, so the question became one test run. `#89898E` — five steps darker,
1.068:1 against systemGray, which is to say indistinguishable — clears all
four backdrops at 3.12:1 and above.

Asked, and answered: darken. The retint moves four values that a test already
pins together (`Colors.ControlBorder`, `FallbackControlBorder`, and the `Input`
and `TextArea` frames), and it is the one value in that palette that leaves
its published Apple source, which the theme now says out loud.

Three consequences worth recording:

- **`knownBoundaryShortfalls` is empty, and that is its resting state.** An
  empty exemption table is what "no pair is under the floor" looks like. It is
  also the state it is easiest to add a bad entry to — no neighbour to match,
  and both sibling tests pass vacuously over nothing — so
  `TestKnownBoundaryShortfallsIsEmptyOrJustified` states what an entry has to
  look like: a reason long enough to be a sentence, a ratio actually under the
  floor, and a pointer to where the argument is made in full.
- **The chip's own test was shaped around the value it must not fail on.** Its
  comment said asserting the inner backdrop "would fail on a value that is
  Apple's own systemGray and correct" — which is a test that has stopped being
  able to find anything. It measures both backdrops now.
- The sibling test that deletes a fixed exemption is the one that fired.

## Phase 2 — the hole that must not be closed

Item 2. `PaddingLeft(0)` then `PaddingHorizontal(16)` gives 16 on both sides,
and there is no way to say "left stays zero" once the axis prop runs.

The answer is that **there must not be one.** A prop whose effect outlives the
props written after it would be the only `StyleProp` in the framework that is
not last-one-wins, and it would be invisible at the call site of whatever it
defeats. What the item was actually missing is that the pair *is* expressible
— in the other order — and that nothing said so.

So the rule, stated: the three widths (`Padding(all)`, the two axes, the four
sides) are asymmetric in exactly one way. A narrower prop can override or
clear a wider one, because it settles the axis first. A wider prop cannot
preserve a narrower one, because it writes every side it covers. **The wider
brush goes first.**

`TestTheWiderBrushGoesFirst` pins all three ordered pairs on both families and
both axes, both directions — eight rows where the two existing tests had one
pair each on one axis. Beside it, `TestTheTwoOrdersAreDistinguishable` is the
one carrying the content: if an axis prop were ever changed to *merge* rather
than overwrite — which is exactly what the Next item asked for — every row's
two orders would converge and the table above would still pass, because both
expectations would be rewritten to the same value by whoever made the change.

### The guarantee the advice rests on

"Just reorder" is only available to a caller whose props are applied last, and
every widget promises that on its `Style` field in prose:

> Style is applied last, so every default above is overridable.

Twenty-odd promises, checked nowhere. `TestACallerStylePropOutranksAWidgets-
OwnInsets` checks it on the six widgets where it can bite — the ones whose
*root node* carries a non-zero inset default — across the two shapes that
differ:

    explicit sides     Card (both families), ListRow
    a live shorthand   GroupHeader (both axes), SearchField, Separator

The shorthand cases are the ones with teeth. A zero side sitting beside a
non-zero `Horizontal` is resolved straight back by every renderer, so each case
asserts the axis field ended at zero as well — the difference between the prop
working and the prop appearing to work. Each case also carries a guard: a
widget whose default on that side has become zero fails, rather than passing on
a no-op.

## Phase 3 — the third knob has an admission test now

Item 3. `rowsSpec` is shared by two widgets and two of its eleven fields are
one widget's. The census counted them and said a third was "worth asking
about", which is a prompt with no answer attached.

The answer is a property of `appendRows`' output rather than a matter of taste.
**Every child it emits is a `core.Keyed` closure**, so a band, a separator and
a row come back as one dynamic type, with the key assigned during `Render` and
the item captured out of reach. A wrapper around that slice cannot pick a row
out of it, cannot read a key without rendering a child the container is about
to render itself, and has no route back to the item.

So: *can this knob be done to `appendRows`' result?* No means it belongs in the
spec. Yes means it belongs in a wrapper the widget applies itself, because the
wrapper needs nothing `appendRows` has.

Both current fields fail the wrapper test, for the only two available reasons:

    Wrap      needs the item — both halves of DataTable's decoration
              (the tap handler, the selection tint) are functions of the row
    Collapse  needs rows not to be produced — no pass over produced children
              can undo their production

`TestAppendRowsOutputIsOpaqueToItsCaller` is that argument as an assertion.

And the owner column is derived from the widgets now rather than trusted as a
string: the census names the fields that *supply* each single-owner knob, and
the test computes which widget has them. "Wrap is DataTable's" stops being true
the moment `GroupedList` grows an `OnRowTap`, and the ordinary way that happens
is a caller asking for it and nobody revisiting the file.

## Phase 4 — a rule with no witness, recorded as such

Item 4. Two rules are invisible under both bundled themes:

    inkOn's first step       reads the theme's declared fill/ink pair before
                             measuring. Both bundled themes pair white with a
                             fill dark enough that measurement picks white
                             anyway.
    OnLight's Primary arm    returns the role's ink-weight tone. Both bundled
                             themes set PrimaryOnLight equal to Primary, so the
                             arm is an identity.

Delete either and no bundled pixel moves. Their only witness is
`midTonePrimaryTheme`, a test fixture.

**This does not fix that, and says so.** Adding a witness means shipping a
third bundled theme, which is a look decision. What it fixes is the silent
part: `TestEveryPaletteRuleStillHasAWitness` records, per rule and per theme,
which themes can show each rule doing something —

                                     Default   Material   midTonePrimary
    the declaration outranks the
    measurement                        -          -            ✓
    OnLight moves Primary              -          -            ✓
    OnLight moves Error                ✓          -            ✓
    OnLight moves Success              ✓          -            -
    OnLight moves Warning              ✓          ✓            -

— and fails if the last witness disappears. It also fails if a *new* one
appears, because a rule that has become observable under a bundled theme is one
whose fixture may have stopped being load-bearing, and several comments would
then be wrong.

That failure is not hypothetical, which is the whole reason to build it. The
two-step ink rule's witness *used to be* `DefaultTheme`, and it was lost to the
retint that moved `Colors.Primary` to Apple's accessible blue — a
straightforward improvement that quietly cost an assertion its teeth, with
nothing anywhere to report it. Item 1 above is the same move again, one
property over.

The four `OnLight` rows are written out rather than looped, because the four
roles are independent and a merged row would report "three of four" and hide
exactly the fact the census exists for.

## The break-tests

**Twenty-five run, twenty-four caught. Zero skips, zero restore failures.**

Two real gaps on the first pass, both closed and re-run.

**Gap 1 — a single-widget knob could be silently downgraded.** Flipping the
census's `Wrap` row from `"DataTable"` to `"both"` (and dropping its `feeds`)
passed the whole suite: the single-widget count came *down* by one, the
derived-owner check skips shared rows by design, and the shared-forwarding test
asserts a tap target on the table only. The fix is small and precise — a spec
field that neither widget spells the same way must name what supplies it, since
there is otherwise nothing left to check its owner column against. `Wrap` is
the only such field.

**Gap 2 — the skipped anchor.** One mutation had a placeholder path and
reported `SKIP`, which is the harness working: an anchor that does not appear
exactly once never silently no-ops. Re-aimed at `core/style.go` and it caught
the change the Next item had literally asked for — `PaddingHorizontal`
preserving an already-explicit left — in `TestTheWiderBrushGoesFirst`.

**The one MISSED is a deleted assertion with an independent oracle.** Removing
the chip test's inner backdrop passes, as any deleted assertion does. Checked
rather than assumed: with that assertion gone *and* the palette reverted, the
boundary census still reports it. Coverage survives the deletion.

Same Python harness as the last four sessions: snapshot by content, one
exact-string mutation, run `go test ./...`, restore, assert the restore by
hashing, and an anchor that does not appear exactly once is a `SKIP`.

Worth noting which tests caught what. The palette census caught the two
mutations aimed at the fixture (`DefaultTheme`'s on-light blue diverging; the
fixture's button base losing its ink) that nothing else in the repository
would have. `TestACallerStylePropOutranksAWidgetsOwnInsets` caught four,
including both "apply the default after the caller's Style" mutations, which
were the ones it was written for.

## Verification

All six paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    wasm/verify/run.sh          replay + 15 mjs suites
    ios/verify/run.sh           flex solver + 9 picker menus + replay + view + app
    android ./gradlew compileDebugKotlin --offline --rerun-tasks
    GOOS=js GOARCH=wasm go build ./...

New files:

    core/inset_width_order_test.go        the three-width lattice, both
                                          families, both directions
    components/caller_style_insets_test.go  the guarantee the ordering rule
                                          rests on, through six widgets
    components/palette_witness_test.go    which themes can still show each
                                          palette rule working

Changed:

    core/theme.go                  the retint and why it is not systemGray;
                                   OnLight's three identity arms
    core/padding_sides.go          "the wider brush goes first", and why no
                                   prop closes the asymmetry
    core/margin_sides.go           the same lattice, one family over
    components/chip.go             chipRing: both backdrops clear, and what
                                   the retired argument was
    components/chip_test.go        the inner backdrop, asserted
    components/variant.go          inkOn points at the witness census
    components/variant_test.go     the exemption deleted; the shape a new one
                                   must have; the fixture points at its guard
    components/rows_spec_test.go   the admission test, the opacity assertion,
                                   and a derived owner column

## Docs

`docs/concepts/styling-and-theming.md`: the boundary table with every pair
clearing and the retint's reasoning; "The wider brush goes first" as its own
subsection with the two-line example; the witness census beside the existing
warning about the bundled themes agreeing with the measurement.
`ROADMAP.md`: four new entries, and a stale line saying margin has no per-side
props.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here. Value is the payoff of doing it, not the effort: `high` means
something is being worked around today or a second consumer has arrived,
`medium` means it blocks one named thing, `low` means it is a gap nobody has
bumped into yet.

1. **(age 4 · value medium) `declaredInk` and `OnLight`'s Primary arm still
   have no bundled witness.** The census records it and fails if the last
   fixture-borne witness goes, which is what was buildable without making a
   look decision. What would actually close it is a third bundled theme whose
   house button is a mid-tone brand colour — the ordinary case for a real app,
   and the case neither bundled palette is. That is a design exercise (a full
   palette, four on-light tones, a `Components` block) rather than a fix, and
   it would pay for itself twice: it is also the only way to test that a theme
   *author* can hit every rule the framework has.
2. **(age 4 · value low) The single-owner census cannot check a shared
   field's claim.** `feeds` makes "this knob is DataTable's" falsifiable, and
   the `"both"` rows are still only covered by an effect test that asserts a
   tap target on one widget. A field wrongly marked shared is now caught for
   `Wrap` (nothing names it) and would not be for a future knob that both
   widgets happen to spell. The honest fix is per-field effect assertions in
   `TestBothWidgetsForwardEveryRowsSpecKnob`, which already exists and asserts
   nine knobs in one pass rather than nine passes.
3. **(age 3 · value medium) An unsized `ZStack` with a placed layer diverges
   on iOS.** SwiftUI has no per-child stack alignment, so a placed layer is
   wrapped in a filling frame, and a filling frame grows an otherwise unsized
   stack to its parent's proposal — where a Compose `Box` and a CSS grid track
   stay the size of their largest child. Documented in four places and
   avoidable by pinning the stack's box, which `ZStack` already asks for. What
   would close it is a SwiftUI layout that places without filling (a custom
   `Layout`, or an `alignmentGuide` scheme that can see the container's size),
   and neither is small. Nothing pins the divergence either — `ios/verify`
   type-checks and replays, it does not measure.
4. **(age 3 · value low) `StackAlign` is inert outside a `ZStack` and nothing
   says so at the call site.** Inert deliberately and by construction on the
   web, and by omission on the natives. A `core.StackAlign` on a `Column`'s
   child compiles, merges, crosses the wire and does nothing on all four
   targets with no diagnostic. **Still the cheapest unclaimed item in the
   list:** `core.AuditTree` is a live tree walk with four findings in it, and
   "a placement on a node whose parent is not an overlay" is a fifth arm rather
   than a new mechanism.
5. **(age 3 · value low) A `Select`'s group headings cannot be styled or
   ordered independently.** A heading is a string on an option, so there is no
   way to give one an icon, mark a whole run disabled, or state a heading with
   no options under it. `<optgroup disabled>` exists in HTML and both natives
   could express a disabled section. `core.SelectMenuSections` is the one place
   a section is described, so this would be a change to `SelectMenuSection` and
   its three transliterations rather than to four renderers.
6. **(age 2 · value medium) The Kotlin decomposition has no runner.**
   `GrMobSelectMenu.kt` imports nothing precisely so a JVM harness could
   execute it, and `ios/verify` proves what such a harness buys. Android has
   only `compileDebugKotlin`, and adding a `src/test` source set means
   resolving JUnit against a `--offline` gradle cache that may not carry it.
   The pile resting on source-text pins alone did not grow this session either,
   but `grMobValue`'s three-way branch is still checked by `strings.Contains`,
   and the prefix-matching failure that shape produces has now appeared in four
   consecutive sessions' break-tests.
7. **(age 2 · value low) The WASM runtime is the fourth copy of the
   decomposition and does not call the authority.** `applySelectOptions` walks
   the flat list itself. Defensible — a DOM append has no closing step — but a
   change to `SelectMenuSection`'s shape has three consumers to update and one
   to remember. What is missing is the shared fixture: `wasm/verify/gen.go`
   emitting the same `menuCases` table.
8. **(age 2 · value low) `styleFromGrMob` has one exemption from its own
   totality rule and nothing states the rule for adding a second.**
   `delete out.display` for a `Modal` is right, and "abstain by deleting the
   key" is now a technique available to any property. The test pins the line;
   what is unwritten is the *test a candidate has to pass*. **The shape to copy
   now exists twice:** `knownBoundaryShortfalls`' new entry-shape test and
   `rowsSpec`'s admission test are both "what a future exception must look
   like", written before the exception arrives.
9. **(age 2 · value low) The keyboard pattern is verified against a DOM that
   is not a browser.** `wasm/verify/dom.mjs` has no bubbling, no layout, and
   its `focus()` is an assignment. Still unverified: that `tabindex="-1"`
   really removes a `<button>` from the tab order, that a disabled control
   refuses focus, that `preventDefault` on `ArrowDown` stops the scroll. The
   honest answer is a browser-driven pass rather than a wider shim.
10. **(age 2 · value low) `menu`, `tree` and `grid` are patterns `core.Role`
    has no vocabulary for.** Each needs more than arrows: a menu has submenus
    and Escape, a tree has expansion state per node, a grid is two-dimensional.
    `components.disclosure` is the heading-around-button shape a tree's twisty
    needs and `Collapse` is the caller-owned expansion state a tree node would
    want; what is still absent is the recursion and the `treeitem` role, and
    the reason remains that no widget in this repository is one.
11. **(age 1 · value medium) The ARIA fixture is transcribed, not generated.**
    `aria/verify/testdata/aria.json` makes every guard checkable against one
    statement, but the statement is hand-written from the spec and can be wrong
    the way the prose it replaced could be wrong. The W3C publishes the role
    definitions machine-readably. The same shape now exists three times —
    `gobindSwiftTypes` and `wantWitnesses` are both hand-transcribed tables
    other tests are held to — the difference being that those are one and five
    rows, where the ARIA fixture is a spec.
12. **(age 1 · value low) The audit's debug-mode guard has no test that can
    fail.** `AuditTree` returns early when debug mode is off, and deleting that
    line changes nothing observable, because `upsertConcern` carries its own
    guard. What the guard buys is cost, and nothing measures it. A benchmark
    asserting "off is free" would cover all three debug checks.
13. **(age 1 · value low) `aria-orientation` is announced for `toolbar` and no
    target gives one a keyboard.** ARIA's toolbar pattern has arrow-key
    navigation, and `components.ChipStrip` is a toolbar of real controls a
    keyboard crosses one tab stop at a time. What stops a third row in
    `COMPOSITE_MEMBERS` is that ARIA does not name a toolbar's members, so the
    member walk needs a second rule — "every focusable descendant not inside a
    nested composite" is probably it.
14. **(age 1 · value low) A typeahead match does not select, only focuses.**
    ARIA's listbox pattern lets a single-select listbox move the selection with
    the focus. This moves focus and leaves selection to the author's `onClick`,
    which is the safe default and also not a choice the framework can make,
    since `aria-selected` is written from Go state a keystroke cannot reach
    without a render pass.
15. **(age 1 · value medium) The gobind type table cannot describe two of the
    shapes it refuses.** `swiftType` rejects a returned bound interface and
    `swiftResult` rejects a multi-result signature, in both cases with a
    message saying to read `Headers/Mobile.objc.h` and add the row. That is the
    right stance for a table that may only hold facts read off a real header —
    and it means the *next* bridge function of either shape is blocked on
    someone having run a `gomobile bind` at least once.
16. **(age 1 · value low) A `Header` override gets the row hiding and has to
    build its own control.** `Collapse` deliberately reaches past an override
    for the row emission, so an override author writes their own button, their
    own `aria-expanded` and their own heading wrapper, with
    `components.disclosure` unexported beside them. Exporting it is one option;
    a `Collapse.Band` helper that returns the default control alone is another.
17. **(age 1 · value low) A collapsible band's tap target excludes its own
    padding and its badge.** The button fills the space between the band's
    insets and stops where the count begins, which is a consequence of keeping
    the count announceable. The fix is not obvious: moving the chrome onto the
    button would leave the badge without its trailing inset, and the band Row
    is the node `StickyHeader` has to sit on.
18. **(age 1 · value low) A collapsed run is invisible to `OnEndReached`.**
    An infinite feed whose last group is shut has no rows near the bottom, so
    the edge sensor sits on the band and the next page never loads. Neither
    widget knows the two features are in tension. The honest fix is probably a
    footer that stays reachable, which `LoadMore` already is, plus a note; the
    interesting one is whether a shut trailing group should suppress the
    auto-load rather than starve it.
19. **(age 1 · value low) The gomobile stub's *doc comment* is still prose.**
    The declarations are pinned character-for-character now, and the header
    comment above them is a hand-written description of rules that live in
    `gobindSwiftTypes` and the two signature builders. It agrees today because
    it was written from them.
20. **(age 0 · value medium) The retint is not visible on any screenshot.**
    `Colors.ControlBorder` moved and every field frame, every quiet chip and
    both `Fallback` paths moved with it, verified entirely by arithmetic. Four
    renderers emit the hex and none of the four verification paths *looks* at
    anything — `ios/verify` type-checks and replays, `wasm/verify` asserts
    against a DOM shim, Android compiles. A 5/255 change is exactly the size
    that arithmetic settles and an eye cannot, which is the argument for it and
    also the reason nobody has seen it.
21. **(age 0 · value low) `boundaryBackdrops` is a list of fills somebody
    remembered.** The census crosses the tone with `Background`, `Surface`,
    `Card`, `Input` and `TextArea`, named by hand, with `Camera` excluded by
    name. A new `ComponentDefaults` field carrying a `Background` — a `Sheet`,
    a `Popover` — is a backdrop a control can sit on and the census would not
    know. Reflecting over `ComponentDefaults` for non-empty `Background`s would
    close it, at the cost of needing the `Camera` exclusion to survive as data
    rather than as a line in a slice literal.
22. **(age 0 · value low) The wrapper test is a rule about a slice nobody
    returns.** `appendRows` appends into the caller's `[]core.PropsAndChildren`
    and returns it, so "can this knob be done to the result" is answered
    against a slice that also holds the container's own props — `Padding(0)`,
    `Gap(0)`, the caller's `Style`. The opacity assertion measures the children
    it emitted, which is the right subject, but a real wrapper would have to
    find them among the props first. That does not change the answer; it does
    mean the rule is stated one level cleaner than the code is.
