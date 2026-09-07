# Session: a band too narrow to ship, and a frame that was the idiom and the bug

Session: https://claude.ai/code/session_017k47JkhG1WBZR5nP29c7Ge
Date: 2026-09-06 (follows "four-arguments-that-stopped-being-true")

## Ask

"Work on the four oldest items in the Next list."

| # | age·value | item |
|---|---|---|
| 1 | 4·medium | `declaredInk` and `OnLight`'s Primary arm have no bundled witness |
| 2 | 4·low | The single-owner census cannot check a shared field's claim |
| 3 | 3·medium | An unsized `ZStack` with a placed layer diverges on iOS |
| 4 | 3·low | `StackAlign` is inert outside a `ZStack` and nothing says so |

All four closed. Item 1 was a look decision and was put to the user, but the
question it was put as is not the one the Next list wrote down — the arithmetic
turned out to *forbid* the obvious answer, and finding that out is most of what
this session was.

## Phase 1 — the item asked for a theme; the arithmetic said which theme

The Next item said the gap closes by "shipping a third bundled theme whose
house button is a mid-tone brand colour". That reading was wrong, and the
reason is a two-line derivation nobody had done.

`contrastInk` returns whichever of the theme's two ink roles scores higher
against the fill. So a *declaration* that disagrees with it is by construction
the loser. Write both ratios against a fill of luminance `L`:

    light ratio  =  (Lb + 0.05) / (L  + 0.05)
    dark  ratio  =  (L  + 0.05) / (Lt + 0.05)
    product      =  (Lb + 0.05) / (Lt + 0.05)      — the fill cancels

**The product is a constant of the palette**, at most 21 for pure white over
pure black. Two numbers whose product is fixed: the smaller is at most the
geometric mean, √21 = 4.5826. Meanwhile
`TestVariantInkIsLegibleOnEveryThemeAndVariant` asks 4.5:1 of every variant's
ink, `VariantDefault` included. So a bundled theme witnessing the rule the way
the fixture does would have to place its button label inside **[4.50, 4.58] —
1.8% of a scale that runs to 21, at its very bottom.**

And that band only exists at all when the two inks are near the poles.
Feasibility is `(Lt + 0.05) < (Lb + 0.05) / 20.25`, which against a white page
means a near-black darker than `#101010`. Both a soft MD3 near-black
(`#1C1B1F`) and Material's `#212121` are outside it: **for those palettes no
brand colour whatever could host such a fill.**

So the mid-tone theme the item asked for was unshippable, and the census row it
was meant to fill was hiding two different rules.

### The escape

The squeeze binds only when the declared ink is one of the two *poles being
measured*. `declaredInk` reads `Components.Button.TextColor`, which can be any
colour — and a brand ink that is neither pole differs from the measurement's
answer with no marginality at all.

`core.AmberTheme` is that: MD amber 700 as the brand (a fine fill, 2.04:1 as
ink on white) with MD brown 900 declared over it at 6.77:1, where measurement
picks the page's near-black at 8.39:1. Deleting `declaredInk` repaints every
`Primary` fill in that theme, with three points of headroom on both sides. It
witnesses `OnLight`'s Primary arm for free, since a role that cannot be ink
needs a separate tone — the one value in the palette without a published
source, because the amber family stops at 900 (#FF6F00, 2.79:1) and has nothing
darker.

The user was given the choice between that, the [4.50, 4.58] band, and shipping
no theme at all, and picked the brand ink and the amber.

### The census row split in two

- **"the declaration outranks the measurement"** — AmberTheme + fixture.
- **"the declaration flips the ink pole"** — the fixture alone, and now
  *provably* rather than accidentally. `TestThePoleFlipBandIsTooNarrowToShip`
  scans the luminance range for a counterexample, checks the band's width
  against the 1.8% the prose quotes, and reports per bundled theme whether such
  a fill is reachable under its own inks. It logs:

      DefaultTheme:  reachable   ("#FFFFFF" / "#000000")
      MaterialTheme: unreachable ("#FFFFFF" / "#212121")
      AmberTheme:    unreachable ("#FFFFFF" / "#1C1B1F")

The theme cleared every other census — variant ink legibility, on-light AA,
field frames, the boundary census, the role pins — on the first run. One test
failed, and it was the right one: `TestVariantDefaultKeepsTheThemePairing`
asserted "the ink is the theme's `Background`", which was true of both palettes
for a reason neither states — they spend white on both, so nothing could tell
which of the two the test was checking. It asserts the declaration now, with
"and these two happen to declare their Background" kept as a separate, named
fact.

### `core.BundledThemes()`, and why it is derived

A dozen tests each spelled their own two-entry map of "the bundled themes". A
third palette would not have *failed* any of them; it would have been **absent**
from them. That is the same failure `TestBundledThemesSetEveryColorRole` was
written reflectively to avoid one level down.

So the list is one function — and guarding a shared list with a second
hand-written list would be a poor joke, so `TestBundledThemesListIsExhaustive`
parses `theme.go` for package-level `var X = &Theme{}` and requires each to
appear in the map. The parse is deliberately narrow (this file, top level,
`&Theme{`) so a fixture built inside a function is not swept in.

## Phase 2 — "shared" was the claim with nothing behind it

Item 2. The `rowsSpec` census records an owner per field, and the two columns
are checked by opposite mechanisms — which is worth being explicit about,
because assuming one covers the other is the mistake:

    owner "X"     falsified against the widgets. `feeds` names the fields that
                  supply the knob and the owner is *computed* from them.
    owner "both"  nothing to derive. A shared field is one both widgets
                  forward, which is a fact about two call sites and not about
                  either struct — so the claim is worth exactly as much as an
                  assertion that fails when one of them stops.

The census required a new field to have a *row*. It never required one to have
an *assertion*. A knob added, marked "both", forwarded by one widget and never
asserted passed everything.

The nine shared knobs are a table keyed by field name now, and
`TestEverySharedKnobHasItsOwnEffectAssertion` drives it from the census in both
directions (a shared row with no effect fails; an effect naming a non-shared row
fails; a single-owner row *with* an effect fails, since the runner drives every
entry through both widgets).

Two assertions fell out that the merged pass never had. `Row` and `Header` had
none at all — the row count stood in for `Rows`, `Key` and `Row` together, and
nothing anywhere rendered a `Header` override through this path. `Rows` (the
count) and `Key` (the keys) are separable now, which matters because a dropped
`Key` emits exactly as many children and silently reconciles by position.

`Header` needs its own render: an override owns its band, so it turns off the
default band's three knobs by design. The harness renders either shape on
demand rather than the table carrying two parallel node lists.

Each effect fires on its own field and names it:

    GroupedList drops Key           -> .../GroupedList/Key
    GroupedList drops Dividers      -> .../GroupedList/Dividers
    GroupedList drops Row           -> .../GroupedList/Row
    DataTable drops HeadingLevel    -> .../DataTable/HeadingLevel

## Phase 3 — the idiom was the bug

Item 3. SwiftUI has no per-child `ZStack` alignment, so a placed layer was
wrapped in `.frame(maxWidth: .infinity, maxHeight: .infinity, alignment:)`.
That is SwiftUI's own idiom for the job, it works, and **a filling frame is
greedy** — an unsized stack with an aligned layer grew to its parent's proposal
on iOS where a Compose `Box` and a CSS grid track stay the size of their
largest child.

It was documented in four places, avoidable by pinning the stack's box, and
pinned nowhere: `ios/verify` type-checks and replays a transcript, and neither
measures a size.

A custom `Layout` places by coordinate, so nothing is wrapped and nothing is
greedy. Three things made it worth doing rather than deferring again:

- **The arithmetic can leave SwiftUI.** `GrMobStack.swift` is pure
  CoreGraphics, split out for the reason `GrMobFlexSolver` was: a `Layout`
  needs a view hierarchy to exercise and a function from numbers to numbers
  does not. `ios/verify` now *measures* the overlay — largest child per axis
  independently, all nine anchors against a known box, the bounds' origin being
  added, an oversized layer overhanging rather than clamping.
- **The vocabulary had to move with it.** `grMobStackAlignment` returned a
  SwiftUI `Alignment`, which cannot be converted to a coordinate — it is an
  opaque pair of guides only SwiftUI can resolve. `grMobStackAnchor` returns
  unit fractions instead, which is the same nine placements in the form the
  arithmetic can use, and the form a test can compare. The old mapping could
  only ever be compared against itself.
- **One path, not two.** The `Layout` replaces the `ZStack` for every stack
  rather than only for placed ones. Two paths that have to agree about sizing
  is the worse trade, and what keeps the change honest is that the one path is
  separately measurable.

Two details are load-bearing and both are commented where they are made:

    the anchor rides a LayoutValueKey     a Layout sees opaque subview proxies
                                          and cannot get back to the GrMobNode;
                                          a parallel array mis-aligns the moment
                                          SwiftUI flattens a Group — and a stack
                                          is exactly where that shows, since a
                                          core.For inside one generates layers
    children measured with the incoming   which is what a ZStack does, and what
    proposal                              keeps a greedy layer greedy: a
                                          background stating maxWidth: .infinity
                                          still makes the stack fill

`containerSize` is deliberately *not* clamped to the proposal. A SwiftUI
`ZStack` reports the union of its children whether or not it fits, and this
replaced a `ZStack`: clamping would be a second behaviour change riding along
with the one the file exists for.

`mobile/verify` gained a guard pointed at the return trip:
`TestNativeZStackOverlaysItsChildren` now fails if `maxWidth: .infinity` ever
appears in `GrMobZStack` again. That frame is the obvious thing to reach for
the next time somebody needs a layer placed — it is the platform's own idiom,
and it works, and it quietly makes one target disagree.

## Phase 4 — a fifth arm, and the parent that is not the tree parent

Item 4. `core.AuditTree` is a live walk with four findings in it, and "a
placement on a node whose parent is not an overlay" is a fifth arm.

The one thing that needed care is which parent. A `Fragment` and a `Theme` have
no box and hand their children upward — htmlout forwards the imposed placement
*through* a Fragment precisely so a `core.For` inside a `ZStack` overlays what
it generated. Checking the tree parent would report every generated layer in
the framework's own idiom for generating layers, which is the failure mode that
gets an audit switched off.

    ZStack                  ZStack                  Row
    └── Fragment            └── Row                 └── Text StackAlign(top)
        └── Text  ok            └── Text  reported      ^ reported

The two sets that decide it (`placingContainers`, `groupingContainers`) moved
into `core`, which is the right owner for a fact about `core.ZStack` — an
exporter is a consumer of the node vocabulary, not its author, and core could
not previously say anything about a placement it defines. htmlout's own tables
keep their per-target reasoning and are pinned to core's by
`TestHtmloutAgreesWithCoreOnWhoPlacesAndWhoGroups`, which lives in htmlout
because only that direction compiles.

The finding says what to do instead — move the node into a `ZStack`, or say
what was meant with the container's own props — and that sentence is asserted,
since a message is the entire product of a debug-only check.

The audit's own file doc was widened rather than left to imply the walk is
still only about ARIA. The walk's subject is the failures a finished tree can
be asked about and a renderer cannot; the four were the first family, not the
only one.

## The break-tests

**Twenty-three run, twenty-two caught, zero MISSED, one skip. Zero restore
failures.**

The skip was a bad anchor (indentation), re-aimed and confirmed separately.

Same harness as the last five sessions — snapshot by content, one exact-string
mutation, run the suite, restore, assert the restore by hashing, and an anchor
that does not appear exactly once is a `SKIP` — with one addition: mutations
carry which suite verifies them, so the six aimed at `GrMobStack.swift` run
`ios/verify` instead of `go test`.

Worth recording what caught what.

- **The five iOS mutations were caught by the new solver checks alone.** A
  transposed anchor, sizing to the first child, dropping `bounds.minX`, clamping
  an oversized layer, and an arm for the centre are all changes that type-check
  and replay identically; before this session nothing in the repository could
  see any of them.
- **The `.frame` mutation was caught by `mobile/verify`**, which is the guard
  written for the return trip rather than for the change.
- **Four theme mutations were caught by the witness census**, including
  `BundledThemes()` losing an entry — which is the shape the shared list was
  built to make loud, and which would previously not have failed anything.
- **Two of the rows-spec mutations were caught by pre-existing widget tests**
  as well as by the new effect table, so the effect table was re-run on its own
  to confirm each per-field assertion fires on exactly its own field.

## Verification

All six paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    wasm/verify/run.sh          replay + 15 mjs suites
    ios/verify/run.sh           flex solver + STACK SOLVER + 9 picker menus
                                + replay + view + app
    android ./gradlew compileDebugKotlin --offline --rerun-tasks
    GOOS=js GOARCH=wasm go build ./...

New files:

    core/placement_audit.go               the fifth arm and its argument
    core/placement_audit_test.go          eight cases, including the two
                                          shapes a naive check gets wrong
    core/bundled_themes_test.go           the theme list, derived from the AST
    ios/GrMob/Runtime/GrMobStack.swift    the overlay's arithmetic, pure
    ios/verify/stack.swift                and the checks that measure it

Changed:

    core/theme.go                  AmberTheme; BundledThemes(); the
                                   Border/ControlBorder and OnLight prose
    core/stack_align.go            the two node-type sets; the divergence
                                   recorded as closed
    core/layout.go                 the same, at ZStack's own doc
    core/a11y_audit.go             the placer threaded through the walk
    ios/GrMob/Runtime/Renderer.swift   GrMobStackLayout replaces the ZStack
    ios/GrMob/Runtime/GrMobStyle.swift  grMobStackAlignment moved out
    ios/verify/{run.sh,main.swift}      the new pass, wired
    mobile/verify/stackalign_test.go    re-pointed at the new file
    mobile/verify/stacking_test.go      the new construct, plus the guard
                                        against the filling frame's return
    mobile/verify/switchlabels_test.go  swiftStack
    components/variant.go          inkOn's "what that leaves here"
    components/variant_test.go     the pairing test asserts the declaration
    components/palette_witness_test.go  the split rule and the band theorem
    components/rows_spec_test.go   the per-field effect table
    htmlout/overlay_test.go        the two lists compared

## Docs

`docs/concepts/styling-and-theming.md`: AmberTheme in the two contrast tables,
"three themes ship", and the witness admonition rewritten around the band.
`docs/concepts/debug-mode.md`: `inert-stack-placement` in the table plus a
section on why the *placing* container is not the tree parent.
`docs/platforms/native.md` and `docs/concepts/views.md`: the iOS divergence
recorded as closed, and what `ios/verify` measures now.
`docs/components.md`: the on-light tone table gains a third column, where
AmberTheme's default row is the widest gap in it.
`ROADMAP.md`: seven new entries; the `StackAlign` entry cross-references the
overlay `Layout`.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here. Value is the payoff of doing it, not the effort: `high` means
something is being worked around today or a second consumer has arrived,
`medium` means it blocks one named thing, `low` means it is a gap nobody has
bumped into yet.

1. **(age 3 · value low) A `Select`'s group headings cannot be styled or
   ordered independently.** A heading is a string on an option, so there is no
   way to give one an icon, mark a whole run disabled, or state a heading with
   no options under it. `<optgroup disabled>` exists in HTML and both natives
   could express a disabled section. `core.SelectMenuSections` is the one place
   a section is described, so this would be a change to `SelectMenuSection` and
   its three transliterations rather than to four renderers.
2. **(age 2 · value medium) The Kotlin decomposition has no runner.**
   `GrMobSelectMenu.kt` imports nothing precisely so a JVM harness could
   execute it, and `ios/verify` proves what such a harness buys — twice over
   now, since the stack solver caught five mutations this session that nothing
   else could see. Android has only `compileDebugKotlin`, and adding a
   `src/test` source set means resolving JUnit against a `--offline` gradle
   cache that may not carry it. `grMobValue`'s three-way branch is still
   checked by `strings.Contains`, and the prefix-matching failure that shape
   produces has now appeared in five consecutive sessions' break-tests.
3. **(age 2 · value low) The WASM runtime is the fourth copy of the
   decomposition and does not call the authority.** `applySelectOptions` walks
   the flat list itself. Defensible — a DOM append has no closing step — but a
   change to `SelectMenuSection`'s shape has three consumers to update and one
   to remember. What is missing is the shared fixture: `wasm/verify/gen.go`
   emitting the same `menuCases` table.
4. **(age 2 · value low) `styleFromGrMob` has one exemption from its own
   totality rule and nothing states the rule for adding a second.**
   `delete out.display` for a `Modal` is right, and "abstain by deleting the
   key" is now a technique available to any property. The test pins the line;
   what is unwritten is the *test a candidate has to pass*. **The shape to copy
   now exists three times:** `knownBoundaryShortfalls`' entry-shape test,
   `rowsSpec`'s admission test, and this session's
   `TestEverySharedKnobHasItsOwnEffectAssertion` are all "what a future
   addition must look like", written before it arrives.
5. **(age 2 · value low) The keyboard pattern is verified against a DOM that
   is not a browser.** `wasm/verify/dom.mjs` has no bubbling, no layout, and
   its `focus()` is an assignment. Still unverified: that `tabindex="-1"`
   really removes a `<button>` from the tab order, that a disabled control
   refuses focus, that `preventDefault` on `ArrowDown` stops the scroll. The
   honest answer is a browser-driven pass rather than a wider shim.
6. **(age 2 · value low) `menu`, `tree` and `grid` are patterns `core.Role`
   has no vocabulary for.** Each needs more than arrows: a menu has submenus
   and Escape, a tree has expansion state per node, a grid is two-dimensional.
   `components.disclosure` is the heading-around-button shape a tree's twisty
   needs and `Collapse` is the caller-owned expansion state a tree node would
   want; what is still absent is the recursion and the `treeitem` role, and
   the reason remains that no widget in this repository is one.
7. **(age 1 · value medium) The ARIA fixture is transcribed, not generated.**
   `aria/verify/testdata/aria.json` makes every guard checkable against one
   statement, but the statement is hand-written from the spec and can be wrong
   the way the prose it replaced could be wrong. The W3C publishes the role
   definitions machine-readably. The same shape now exists four times —
   `gobindSwiftTypes`, `wantWitnesses`, `wantRowsSpecFields` and the ARIA
   fixture are all hand-transcribed tables other tests are held to — the
   difference being that the first three are one to eleven rows, where the
   ARIA fixture is a spec.
8. **(age 1 · value low) The audit's debug-mode guard has no test that can
   fail.** `AuditTree` returns early when debug mode is off, and deleting that
   line changes nothing observable, because `upsertConcern` carries its own
   guard. What the guard buys is cost, and nothing measures it — and the walk
   just gained a fifth per-node check, so the cost it buys went up. A benchmark
   asserting "off is free" would cover all three debug checks.
9. **(age 1 · value low) `aria-orientation` is announced for `toolbar` and no
   target gives one a keyboard.** ARIA's toolbar pattern has arrow-key
   navigation, and `components.ChipStrip` is a toolbar of real controls a
   keyboard crosses one tab stop at a time. What stops a third row in
   `COMPOSITE_MEMBERS` is that ARIA does not name a toolbar's members, so the
   member walk needs a second rule — "every focusable descendant not inside a
   nested composite" is probably it.
10. **(age 1 · value low) A typeahead match does not select, only focuses.**
    ARIA's listbox pattern lets a single-select listbox move the selection with
    the focus. This moves focus and leaves selection to the author's `onClick`,
    which is the safe default and also not a choice the framework can make,
    since `aria-selected` is written from Go state a keystroke cannot reach
    without a render pass.
11. **(age 1 · value medium) The gobind type table cannot describe two of the
    shapes it refuses.** `swiftType` rejects a returned bound interface and
    `swiftResult` rejects a multi-result signature, in both cases with a
    message saying to read `Headers/Mobile.objc.h` and add the row. That is the
    right stance for a table that may only hold facts read off a real header —
    and it means the *next* bridge function of either shape is blocked on
    someone having run a `gomobile bind` at least once.
12. **(age 1 · value low) A `Header` override gets the row hiding and has to
    build its own control.** `Collapse` deliberately reaches past an override
    for the row emission, so an override author writes their own button, their
    own `aria-expanded` and their own heading wrapper, with
    `components.disclosure` unexported beside them. Exporting it is one option;
    a `Collapse.Band` helper that returns the default control alone is another.
    This session's `Header` effect assertion is the first thing anywhere that
    renders an override through `appendRows`, so the shape now has a fixture.
13. **(age 1 · value low) A collapsible band's tap target excludes its own
    padding and its badge.** The button fills the space between the band's
    insets and stops where the count begins, which is a consequence of keeping
    the count announceable. The fix is not obvious: moving the chrome onto the
    button would leave the badge without its trailing inset, and the band Row
    is the node `StickyHeader` has to sit on.
14. **(age 1 · value low) A collapsed run is invisible to `OnEndReached`.**
    An infinite feed whose last group is shut has no rows near the bottom, so
    the edge sensor sits on the band and the next page never loads. Neither
    widget knows the two features are in tension. The honest fix is probably a
    footer that stays reachable, which `LoadMore` already is, plus a note; the
    interesting one is whether a shut trailing group should suppress the
    auto-load rather than starve it.
15. **(age 1 · value low) The gomobile stub's *doc comment* is still prose.**
    The declarations are pinned character-for-character now, and the header
    comment above them is a hand-written description of rules that live in
    `gobindSwiftTypes` and the two signature builders. It agrees today because
    it was written from them.
16. **(age 1 · value medium) The `ControlBorder` retint is not visible on any
    screenshot.** Now two retints deep: `AmberTheme` adds a whole palette that
    no verification path has ever *looked* at. Four renderers emit its hexes
    and all four passes are arithmetic, type-checks or DOM shims. A palette is
    the one artefact where the thing being verified and the thing being shipped
    are furthest apart.
17. **(age 1 · value low) `boundaryBackdrops` is a list of fills somebody
    remembered.** The census crosses the tone with `Background`, `Surface`,
    `Card`, `Input` and `TextArea`, named by hand, with `Camera` excluded by
    name. A new `ComponentDefaults` field carrying a `Background` — a `Sheet`,
    a `Popover` — is a backdrop a control can sit on and the census would not
    know. Reflecting over `ComponentDefaults` for non-empty `Background`s would
    close it, at the cost of needing the `Camera` exclusion to survive as data
    rather than as a line in a slice literal.
18. **(age 1 · value low) The wrapper test is a rule about a slice nobody
    returns.** `appendRows` appends into the caller's `[]core.PropsAndChildren`
    and returns it, so "can this knob be done to the result" is answered
    against a slice that also holds the container's own props. The opacity
    assertion measures the children it emitted, which is the right subject, but
    a real wrapper would have to find them among the props first.
19. **(age 0 · value medium) The overlay `Layout` is measured and has never
    been mounted.** `GrMobStackSolver` is checked hard, and `GrMobStackLayout`
    — the part that proposes sizes to subviews and reads the `LayoutValueKey`
    — is only type-checked, exactly like every other SwiftUI view here. Three
    things it does are judgement calls a simulator would settle in a minute and
    no test can: measuring children with the incoming proposal rather than an
    unspecified one, *not* clamping the container to the proposal, and
    re-proposing `bounds.size` at placement. Each is argued in the file; none is
    observed. This is the first change in a while where the arithmetic is
    pinned and the *plumbing* is the risk.
20. **(age 0 · value low) `AmberTheme` is not used by any example app.**
    Three palettes ship and the tutorial's theme chapter still lists two by
    hand (`examples/tutorial/chapter7.go` builds its own slice). Nothing is
    wrong; it is that the new theme's only readers are tests, so the one
    question a bundled theme exists to answer — does a real screen look right
    in it — has not been asked. Switching one example to it would ask it.
21. **(age 0 · value low) `wantWitnesses` is a hand-transcribed table that a
    fourth theme silently widens.** The census now derives its *theme list*
    from `core.BundledThemes()`, so a new palette is asked every rule — and its
    answers land as a diff against a map written by hand, which is the right
    failure. What is not stated anywhere is what a new row should look like when
    somebody is adding a theme rather than debugging one: "witnessed by nobody"
    and "witnessed by everybody" are both legal and mean opposite things.
