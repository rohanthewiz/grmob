# Session: a blue that was two decisions, a corner nobody could name, and a heading over a run

Session: https://claude.ai/code/session_01YVcrPiEYG2vnpUePWiK5T5
Date: 2026-09-06 (follows "a-parameter-list-and-three-missing-sides")

## Ask

"Take the three oldest items in the Next list."

All three were age 4 — items 1, 2 and 3 — and they had sat for the reason
age-4 items sit: the first was labelled a *decision* rather than a patch, the
second was blocked on "one consumer is not a vocabulary", and the third was
rated low.

What the three had in common turned out to be that **each one's stated blocker
had already dissolved and nobody had noticed**. The theme decision had exactly
one defensible answer. The vocabulary argument was answerable by a fact about
the three platforms nobody had checked. And the "low" item was low because the
work looked like four renderers, which it was, and the four renderers each took
six lines.

## Phase 1 — the blue, and why only the role could be moved

### The entry named the fix and the previous session named the cost

> `inkOn` returns the theme's declared white on `#007AFF`, below AA for body
> text, and every filled `Button` plus the calendar's selected day spends it.

`Colors.Primary` is now `#0040DD`, Apple's accessible blue — the hex
`PrimaryOnLight` already carried. White over it is 7.56:1 rather than 4.02:1.

### The half-move is the interesting wrong answer

Darkening `Components.Button.Background` alone and leaving the role at
systemBlue looks narrower and is actually *broken*, and the reason is the
mechanism the previous two sessions built. `declaredInk` reads the button base
as "the ink for **this fill**". Move one without the other and `Primary`
becomes a fill the theme has paired nothing with — so `Calendar`'s selected day
falls back to measurement, which on a mid-tone blue picks **black**. That is
the exact disagreement `inkOn` was written to end, reintroduced by the fix for
it.

So the two must move together, and nothing in the type system says so:

    TestBundledButtonFillsAreThePrimaryRole   core/theme_test.go

on the field-frame pin's pattern, and for the same reason — a `Style` is a
value, so a component default cannot call a resolver.

### The rule, stated from the other side

Five of the eight role/theme pairs failed AA as ink; four still do. Primary
left the list, and it is worth saying *why that is not a contradiction* of the
on-light tones existing at all:

> Darken the **role** when the role is spent as a fill and the ink declared
> over it is the problem. Add a **tone** when the role is a perfectly good fill
> and only fails as ink.

Success and Warning are the second case — DefaultTheme's green and orange carry
black at ~9.5:1. Primary turned out to be the first, because a *pairing* has no
second tone to reach for.

### Five tests failed, and four of them were tripwires

This is the part worth recording. `go test ./core/ ./components/` after the
two-line theme edit:

    TestOnLightResolvesAColourToItsRolesTone           a literal "#007aff"
    TestInkOnReadsTheThemeDeclarationAndMeasures...    a literal "#007aff"
    TestVariantDefaultKeepsTheThemePairing             "the contrast rule now
                                                       agrees with the theme
                                                       pairing on Primary; this
                                                       test no longer proves the
                                                       declaration is consulted"
    TestButtonOutlinedAndGhostAreTransparent...        "fixture no longer
                                                       exercises the split"
    TestChipProminenceLoudIsAnOutlineNotAFill          "fixture no longer
                                                       exercises the split"

Three of those five are self-guards written by earlier sessions, and every one
of them said something true: **DefaultTheme's Primary was the fixture for two
different properties, and the fix destroyed both.**

    the declaration splits from the measurement
        white 4.02 vs black 5.23 — so declaredInk and contrastInk disagree

    the role splits from its on-light tone
        #007AFF cannot be read as ink on white and #0040DD can

Both bundled themes now have Primary ink-weight on its own, so neither can
demonstrate either property. An implementation that deleted `declaredInk` and
only measured would paint identical pixels under both of them.

The answer was `midTonePrimaryTheme()` in `components/variant_test.go` — **the
framework's own former default, kept as a fixture precisely because it stopped
being the default**. systemBlue, white declared over it, accessible blue as the
separate tone. It carries both splits, and the guards now point at it rather
than at DefaultTheme.

`TestChipProminenceLoudIsAnOutlineNotAFill` moved onto the fixture wholesale.
`TestButtonOutlinedAndGhost...` became a theme × variant matrix with a
post-loop census: every variant tested must split under at least one theme in
it.

### The payoff, and it is the exemption disappearing

`TestButtonFilledStatusVariantsAreLegibleOnEveryTheme` carried a paragraph
explaining why `VariantDefault` had to be excluded — failing on it would report
a palette decision as a widget defect, and "fixing" it in the widget would
repaint every button in every tree. The palette paid it. Both censuses now run
all four variants on both themes, so a later retint of either bundled button is
caught by number rather than by eye.

### The second copy of the hex

`core.PrimaryColor()` returned a hardcoded `#007AFF` and `examples/chat` paints
white on it. Fixing the theme and leaving that standing would have been half a
job, so both legacy helpers read `DefaultTheme` now. `wasm/main.go`'s demo
pill goes through the same function.

## Phase 2 — the corner, and the fact that settled the vocabulary argument

### The entry's blocker

> An `AlignSelf`-shaped prop is the obvious shape and all three constructs have
> a spelling for it, but one consumer is not a vocabulary.

`AlignSelf` is the wrong shape, and checking why is what unblocked it.
`AlignSelf` is flexbox's: one axis, whose identity depends on the container's
direction, honoured by the two DOM targets and neither native. A layer needs
both axes at once, and a stack has no main axis for the other to be the cross
of. Reusing the field would have given it two meanings dispatched on the
parent's node type — the shape `Box` and `ZStack` were split apart to stop.

What made it a vocabulary was arithmetic, not a second consumer:

    SwiftUI   Alignment            9 members (.topLeading … .bottomTrailing)
    Compose   Alignment            9 members (TopStart … BottomEnd)
    CSS grid  justify-self/align-self   3 × 3

The same nine values, three times. So `core.StackAlignment` is a 3×3 grid with
the centre unspelled:

        top-start      top      top-end
            start    (center)     end
     bottom-start   bottom    bottom-end

`StackAlignCenter` is `""`, so an unset `Style.StackAlign` *is* it and no tree
that predates the property moves. `StackAlignments()` omits it — the census is
the placements that ask for something — which is `SelectedStates()`' rule.

### The centre is in htmlout's table and not in the census, and that is the point

`stackPlacements` has nine rows where `StackAlignments()` has eight, and the
distinction is between a **dispatch** and a **total statement**:

- The WASM runtime restates every property it manages on every pass, so a layer
  that *loses* its placement has to be written back to the middle.
- `align-self` is a property a layer can already set for itself through
  `Style.AlignSelf`, and a grid item honours it. A stack that wrote nothing on
  its unplaced layers would let that flex prop move one — on the two DOM
  targets only, in flat contradiction of `ZStack`'s contract.

So imposing the centre **closed a pre-existing leak** rather than costing two
declarations for nothing. `TestALayersOwnAlignSelfDoesNotPlaceIt` pins it in
htmlout and there is an mjs twin.

### Imposed, not written

The web half travels through htmlout's `imposed` channel — and it is the first
thing that channel has carried that **differs from sibling to sibling**;
everything before it was one string for the whole set. The alternative (the
layer writing its own `justify-self`/`align-self`) would re-place a `Row`'s
children the moment somebody wrote the prop on the wrong node, silently and on
the web alone. `TestAPlacementOutsideAStackReachesNothing` is the pin.

The runtime does the same split through a dataset stamp: `applyStyle` parks the
value on the element (`data-stack-align`, `data-tab-selected`'s channel) and
`syncOverlay` reads it back and writes the pair.

### The one divergence, named rather than hidden

SwiftUI has no per-child ZStack alignment — a stack's `alignment:` is the
stack's — so a placed layer is wrapped in
`.frame(maxWidth: .infinity, maxHeight: .infinity, alignment:)`. That frame is
greedy, so an **unsized** stack with an aligned layer grows to its parent's
proposal on iOS where a Compose `Box` and a CSS grid track stay the size of
their largest child.

The frame is applied *only* to a layer that asks for a placement, which keeps
every existing tree untouched, and `ZStack`'s doc already asks a stack to pin
its dimensions. A stack that does is identical on all four. Written into
`core/stack_align.go`, `core/layout.go`, `Renderer.swift`, and the views and
native docs — four places, because it is the sort of thing that gets
rediscovered.

### The consumer that had been the blocker

`Compass`'s index mark was a full-height `Column` justifying its glyph to the
start. One prop replaced a height, a main-axis rule and a cross-axis rule —
three props, none of which says where the glyph goes — and removed a second
copy of the stack's size, so a `Size` change no longer has to be made twice or
the mark drifts off the rim.

## Phase 3 — a heading over a run

`SelectOption` gained `Group` and `Disabled`. Four renderers, six lines each,
and two decisions worth recording.

### Runs, not a gather

Consecutive options sharing a heading are one section, **in the order written**.
The same heading either side of a different one is two sections.

The alternative — gather every option with a given heading — silently reorders
the caller's list, which is what a person sees and what the keyboard walks.
Sorting a list into its sections is a line of Go at the call site; un-sorting
one is not. So the widget does the thing it can undo nothing of.

Each target draws a run its own way, and this is where the two menus stop
looking alike:

    web       <optgroup>, closed when the next option's heading differs
    iOS       Section — a *container*, so the runs are computed ahead of the
              ForEach and the buttons moved out of GrMobSelect into
              grMobMenuItems so a run can be handed over whole
    Android   DropdownMenu has no section, so a heading is an ordinary
              DropdownMenuItem with enabled = false and an empty onClick

The Swift disabling is on the **Button**, not the Section: on the Section it
would take the whole run with it.

### The keys are written only when they say something

The WASM runtime decides whether to rebuild a picker's `<option>` elements by
comparing the option list's JSON, and rebuilding closes an open drop-down
mid-choice. A `group` key present on every option with an empty value would
have changed every picker's signature the first time this shipped — rebuilding
every list in every app, for nothing.

`disabled` crosses as the string `"true"`, `core.SelectedState`'s spelling: the
map is `[]map[string]string`, one flat shape all four renderers already read,
and `"true"` is what the DOM writes so the web half needs no translation.

### The existing Swift pin caught the move

`TestNativeSelectDispatchesTheOptionValue` failed the moment the buttons left
`GrMobSelect` — "does not call textChanged(". The anchor was repointed at
`grMobMenuItems` rather than the code being kept in one place to suit the
anchor. That is the check working, and it is recorded in the test's own doc.

## The break-test that got through

**Twenty-six break-tests, twenty-five caught.**

The one that passed: deleting htmlout's close of a **trailing** `<optgroup>`.
The output was a `</select>` inside an open group — malformed markup a browser
repairs and a parser does not — and every assertion in the file passed.

The reason is the shape of the fixture. `TestSelectGroupsConsecutiveOptions...`
ends its list with an *ungrouped* option, so the last run is closed by the next
option's heading changing and the explicit close after the loop is never
reached. `strings.Count(out, "</optgroup>") != 2` therefore agreed with the bug.

`TestSelectClosesAGroupThatEndsTheList` checks the **nesting** and not the tag
count — `</optgroup>` before `</select>`, both options inside it — because a
stray close in the wrong place balances a count and is still wrong.

The general shape, worth keeping: *a loop with an end-of-loop flush needs a
fixture whose last element reaches it.* The same hole would exist in any
run-length encoder.

The full list of break-tests, by phase:

**Theme (6, all caught).** Role back to systemBlue with the button dark
(`TestBundledButtonFillsAreThePrimaryRole`); both back to systemBlue (both
legibility censuses, by number); the fixture theme's primary darkened (all
three split-guards); `declaredInk` made case-sensitive; `inkOn` stopped asking
the theme; the loud chip spending the accent instead of its tone.

**StackAlign (11, all caught).** `applyTo` dropping the field (the reflective
merge census named it); a ninth constant with no census entry; htmlout dropping
the centre row (three tests, in two packages); Swift losing an arm; Kotlin
gaining an arm for the centre (both coverage checks); the Swift parser not
reading the key; Kotlin reverting to `RenderChildren`; the runtime made
non-total; a wrong row in the runtime table; the compass reverting to its
wrapper; `IsOverlay` forced true.

**Select (9, one gap).** Core writing both keys unconditionally; htmlout
gathering instead of running; **htmlout never closing the trailing run — the
one that got through**; htmlout dropping `disabled`; the runtime appending
every option at the top level; the runtime ignoring `disabled`; Swift dropping
the Section; Kotlin dropping `enabled`; the tutorial's seat picker losing a
group.

## A process note that cost three reverts

A `restore()` helper written as `for f in $FILES` — an unquoted scalar — does
not word-split under zsh. Three break-test reverts silently did nothing, and
the next batch ran against a repo still carrying the previous mutation. It was
recoverable (the three edits were known and `git checkout` was never run, which
would have discarded the whole session), and snapshots use a zsh **array** now.

This is the second session running in which the break-test harness itself was
the thing that broke; the previous one was a `cd android` that leaked into the
next batch.

## Verification

All six paths, green:

    gofmt / go build / go vet / go test ./... / go test -race ./...
    wasm/verify/run.sh          replay + 12 mjs suites
    ios/verify/run.sh           flex solver + replay + view + app layers
    android ./gradlew compileDebugKotlin --offline --rerun-tasks
    GOOS=js GOARCH=wasm go build ./...

New files:

    core/stack_align.go             the type, nine constants, the census, the prop
    core/stack_align_enum_test.go   the const-block pin, the zero-value rule, and
                                    the 3x3 decomposition check
    mobile/verify/stackalign_test.go both natives' coverage, the centre held out
                                    of both dispatches, the two style parsers,
                                    and the per-layer read in each stack renderer

New and changed tests elsewhere:

    core/theme_test.go              the Button fill pinned to the Primary role;
                                    the OnLight case check repointed at Error,
                                    whose tone is still a different hex
    components/variant_test.go      midTonePrimaryTheme, the fixture carrying
                                    both splits DefaultTheme lost; VariantDefault
                                    added to the ink census
    components/button_test.go       the outlined/ghost check as a theme x variant
                                    matrix with a per-variant split census;
                                    VariantDefault added to the filled census
    components/chip_test.go         the loud outline moved onto the fixture
    components/compass_test.go      the mark is the Text, placed, not a box
    core/select_test.go             the two keys written only when set; a
                                    disabled option keeping its index; the label
                                    default surviving the grouped path
    htmlout/overlay_test.go         per-layer placement; every placement distinct
                                    and reaching the markup; inert outside a
                                    stack; a layer's own AlignSelf not moving it
    htmlout/select_test.go          optgroup runs; a split group not gathered; a
                                    trailing group closed (the break-test's
                                    answer); disabled written; the label escaped
    wasm/verify/overlay_test.go     STACK_PLACEMENTS against Go's table; the
                                    totality of the pass, both ends
    wasm/verify/overlay_test.mjs    four live cases, including a layer that loses
                                    its placement returning to the centre
    wasm/verify/select_test.mjs     five live cases, including that a group or a
                                    disabled flag changing rebuilds the list
    mobile/verify/select_test.go    groups and disabled on both natives; the
                                    value-dispatch anchor repointed at
                                    grMobMenuItems
    examples/tutorial/chapter5_test.go  the seat picker's runs read off the
                                    rendered tree, with two options sharing a
                                    label

## Docs

`core/theme.go`: the on-light table kept as the state the fields were
introduced for, plus the darken-the-role / add-a-tone rule; `Primary` carrying
the whole argument for the move including why the button base could not be
moved alone; `PrimaryOnLight` restated as an identity on MaterialTheme's
pattern.
`components/variant.go`: `inkOn`'s "honest cost" section became "the cost the
theme then paid", plus a new section on what the rule is left with — unchanged,
not historical, but with its evidence now in a fixture.
`components/button.go`, `components/chip.go`, `components/calendar.go`: the
numbers and the "[was 4.02]" brackets brought current; the calendar's comment
now says the two rules agree under this theme and the call still has to be
`inkOn`.
`core/style.go`: `PrimaryColor`/`DangerColor` documented as the theme-blind
legacy pair and pointed at `DefaultTheme`.

`core/stack_align.go`: the grid drawn, why not `AlignSelf`, what each target
does, and the SwiftUI divergence with the way to avoid it.
`core/layout.go`: `ZStack`'s alignment contract rewritten as "centred by
default"; the sizing note grew the second reason to pin a stack's box.
`htmlout/stack.go`: the placement table, why the centre is in it, and why grid
spellings rather than flexbox ones.
`Renderer.swift` / `Renderer.kt`: the frame and the modifier, each with why the
*other* platform's approach is not available.

`core/input.go`: `SelectOption.Group` (runs, and why a gather is the wrong
shape) and `.Disabled` (drawn rather than omitted, and what it does not stop);
`Select`'s doc gained a section on both; the flattening comment explains the
signature argument for writing the keys conditionally.

`ROADMAP.md`: the blue as its own entry under the ink rule, `StackAlign` under
`ZStack`, the two option fields under `Select`.
`docs/concepts/styling-and-theming.md`: a note on the primary row being settled
at the role; the ink section rewritten with a warning that both bundled themes
now agree with the measurement; `StackAlign` in the first row of the parity
table with a paragraph on why it and `AlignSelf` are in different rows.
`docs/concepts/views.md`: the 3×3 table, the two admonitions.
`docs/concepts/forms.md`, `docs/components.md`, `docs/platforms/{native,wasm,exporters}.md`
all updated.

Tutorial: **5.6 grew the option list's other two fields** — the seat picker is
grouped by cabin with the exit row disabled, and two seats share the label
"Aisle" and differ in value, which is the case the Value rule exists for.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here. Value is the payoff of doing it, not the effort: `high` means
something is being worked around today or a second consumer has arrived,
`medium` means it blocks one named thing, `low` means it is a gap nobody has
bumped into yet.

1. **(age 5 · value low) A picker's open menu is invisible to Go.**
   Deliberate, but it means no test can drive a native picker's list, and the
   two `mobile/verify` pins read source text rather than behaviour. Now
   slightly larger than it was: the Section headings and the disabled arms
   added this session are pinned the same way, so three facts about the menu
   rest on a parse.
2. **(age 5 · value low) A styleless node skips the border reset in the WASM
   runtime.** `applyStyle` only runs when `node.Style` is non-nil. No node
   `core` builds is ever in that state, so this is a hand-assembled-tree gap.
3. **(age 4 · value high) Neither a `listbox` nor a `tablist` has keyboard
   navigation on the web, and there are three consumers now.** The roles state
   semantics ARIA's patterns pair with behaviour: the container takes focus,
   the arrow keys move the active item, and a roving tabindex or
   `aria-activedescendant` says which. `examples/mobileapp`'s article list is
   the listbox; `examples/social`'s bottom bar and tutorial 4.5 are the
   tablists, and all of them hand a keyboard user a role that claims more than
   the widget does. Needs a focus concept `core` does not have —
   `core/focus.go` is about putting the cursor in a named field. Both phones
   navigate by swipe and lose nothing. An accordion header is a third consumer
   in a different form: a real `button` on the web and therefore already
   keyboard-operable, which *narrows* the item rather than widening it.
4. **(age 4 · value medium) Nothing re-checks a permission on foreground.**
   A user can grant one in Settings and come back, and no platform says so.
   `hooks.UsePermission` deliberately does not (it cannot see whether its
   screen is still on top, and five screens would each fire a check per
   resume), so every consumer writes the same `UseLifecycle` pairing by hand.
   A `hooks.UsePermissionLive` — or a `RecheckOnForeground` option — is the
   obvious shape; what is missing is a rule for which screen owns it.
5. **(age 4 · value low) The Android "asked before" flag does not survive a
   process restart.** It is what separates a permanent refusal from "never
   asked", so after a restart a permanently-denied permission reads as
   `Prompt` until the next request proves otherwise — one dead button press.
6. **(age 4 · value low) A browser `Request` opens the device to answer.**
   There is no request API, so `getUserMedia` is the only thing that prompts,
   and a granted camera check has genuinely opened the camera for a moment. A
   refused one also reads as `denied` whether the user pressed Block or
   dismissed the prompt, because `NotAllowedError` does not say which.
7. **(age 4 · value low) The gomobile stub's types are unchecked.** The pin
   holds the *names* — every bindable symbol declared, nothing extra —
   because copying gobind's type mapping into a test would be reimplementing
   gobind. A wrong signature fails the Swift typecheck the moment the shell
   calls it, so what is genuinely unguarded is a stub declaration the shell
   never touches.
8. **(age 3 · value medium) A hand-built tab strip's panel still cannot say it
   is one.** `AccessibilityControls` closes the pointing and `RoleTabPanel` is
   now defensible. What stands in the way is mechanical: the WASM runtime
   tells its own wiring apart from an author's role by `"tabpanel"` not being
   a `core.Role`, held there by `TestNoRoleCollidesWithTheTabPanelWiring`.
   Replacing that discriminator with a `data-grmob-panel` marker (which both
   web targets would write, as they already do `data-grmob-chrome`) is the
   right fix and unblocks the constant. Two native arms and a large doc block
   go with it.
9. **(age 3 · value medium) Three widgets take `group` where ARIA has a better
   role.** The supplied fallback made their names audible and stopped there.
   `ProgressBar` wants `progressbar` with `aria-valuenow`/`valuemin`/`valuemax`
   — it is already computing the percentage into its name, which is the
   workaround. `Skeleton` wants `status`, since "Loading" is an advisory that
   is replaced. `FormField`'s required marker is a glyph standing in for a
   word, which is `img`'s shape. Each is a widget change plus, for
   `ProgressBar`, a value vocabulary `core` does not have.
10. **(age 3 · value medium) Nothing catches a dangling `aria-controls` or a
    duplicate `AccessibilityID`.** Both are written verbatim and neither
    exporter can see the whole document at the moment it writes one — but a
    finished tree *can* be walked, which is exactly what `core.SetDebugMode`
    already does for cursor drift and duplicate keys. A debug-mode pass
    reporting an id claimed twice and a reference resolving to nothing would
    catch the one failure mode the pair has, and it is a failure that is
    invisible on every target rather than merely quiet on two.
11. **(age 3 · value low) An `AccessibilityID` is not validated.** An id
    containing a space is invalid HTML; an empty one is written as `id=""`; one
    starting with `grmob-` collides with `core.TabView`'s minted ids and breaks
    a wiring the author never wrote. The prefix is documented as reserved and
    nothing enforces it. A drop would be silent too, so the honest fix is
    probably the debug-mode pass in item 10 rather than a guard in the
    exporters.
12. **(age 2 · value medium) Every ARIA claim in the docs is hand-checked
    prose.** A Next-list item once asserted "`group` supports `aria-expanded`",
    which survived three re-sorts and was false. There are dozens of such
    claims now — three role lists, two name prohibitions, four `aria-level`
    scopes, six `aria-selected` roles. A generated table (from the ARIA spec's
    own machine-readable role definitions, checked in as a fixture) would make
    them testable instead of reviewable. Build-time rather than runtime, which
    is why it is medium.
13. **(age 2 · value medium) `Accordion` is the only disclosure, and
    `AccessibilityExpanded` has no second consumer.** One consumer is not a
    vocabulary, and the field's role list carries five arms nothing reaches —
    `link`, `listbox`, `row`, `columnheader` and `tab`. The shapes that would
    reach them are real and absent: a `GroupedList` band that collapses (row),
    a combobox built out of `SearchField` plus a listbox (link/listbox), a
    `DataTable` with collapsible column groups (columnheader). **Note the
    parallel to `StackAlign` this session:** that item was also blocked on
    "one consumer is not a vocabulary", and what unblocked it was not a second
    consumer but a fact about the three platforms. Worth asking the same
    question here before waiting for a widget.
14. **(age 2 · value low) `Colors.ControlBorder` is 2.92:1 against
    `Colors.Surface` under `DefaultTheme`.** The outer edge clears 3:1 and is
    what identifies a quiet chip, which is argued in two places and asserted in
    one — but any future widget that draws a boundary *on* a Surface fill
    inherits the shortfall without inheriting the argument. The honest fixes
    are both theme decisions: darken past Apple's systemGray, or give the quiet
    chip a fill that is not Surface. **Cheaper than it was:** this session
    established that a role's spelling is movable when the palette is the
    authority, and moved one.
15. **(age 2 · value low) An `ExpandedState` on a node with no handler is
    silently inert on Android.** `grMobDisclosure` is in `gestureModifier`,
    which returns early when a node carries neither `onClick` nor `onLongPress`,
    so the state reaches the renderer and buys nothing. That is deliberate — an
    action nothing can perform is worse than none — but it is a rule stated
    only in a comment, where the equivalent web rule (a state on an unroled
    node is dropped) has a test on both targets. `core.SetDebugMode` reporting
    a disclosure with no way to open it would put it where the other two are.
16. **(age 1 · value medium) `Margin` has no per-side props at all.** Padding
    got its four last session; margin still has only `Margin(all)`, so every
    single-side margin goes through a whole `EdgeInsets` in a `UseStyle` — and
    that is worse for margin than it was for padding, since there is no
    `MarginHorizontal` or `MarginTop` either. Two live workarounds today:
    `components/separator.go` writes `Margin: EdgeInsets{Horizontal: s.Inset}`
    for its inset rule, and `examples/chat`'s bubble writes
    `Margin: EdgeInsets{Bottom: 8}` for the gap between messages. The work is
    mechanical — the same `settleHorizontal`/`settleVertical` helpers apply
    unchanged, and no renderer would move — which is why this is medium and not
    high: it is scope, not difficulty.
17. **(age 1 · value low) A side prop cannot express "clear this side" on a
    node whose axis a later prop will set.** `PaddingLeft(0)` then
    `PaddingHorizontal(16)` gives 16 on both sides, which is correct
    last-one-wins and is also the only way to write the pair. Recorded because
    the settle makes every *other* zero work, and the one remaining hole should
    be written down rather than rediscovered.
18. **(age 1 · value low) `rowsSpec` is shared by exactly two widgets and one
    of its ten fields by exactly one.** `Wrap` exists so DataTable can add a
    tap target and a selection tint that GroupedList has no use for, which
    makes it a widget-specific knob living in a shared type. That is the right
    trade at two callers; at three it would be worth asking whether the
    decoration belongs in the spec or in a wrapper around `appendRows`' output.
19. **(age 0 · value medium) `declaredInk` is now unobservable under both
    bundled themes.** Each pairs white with a fill dark enough that measurement
    picks white too, so an implementation that deleted the first step and only
    measured would paint identical pixels under DefaultTheme *and*
    MaterialTheme. The rule is right and still applies to any theme whose house
    button is a mid-tone — but its whole evidence is now one test fixture
    (`midTonePrimaryTheme`), which is a thinner thread than a bundled theme.
    The honest options are to leave it and accept the fixture, or to give one
    bundled theme a mid-tone button base on purpose, which is a look decision.
    Same shape one property over: `PrimaryOnLight` is an identity on both
    bundled themes now, so the *reverse lookup* has no live consumer either.
20. **(age 0 · value medium) An unsized `ZStack` with a placed layer diverges
    on iOS.** SwiftUI has no per-child stack alignment, so a placed layer is
    wrapped in a filling frame, and a filling frame grows an otherwise unsized
    stack to its parent's proposal — where a Compose `Box` and a CSS grid track
    stay the size of their largest child. Documented in four places and
    avoidable by pinning the stack's box, which `ZStack` already asks for. What
    would actually close it is a SwiftUI layout that places without filling
    (a custom `Layout`, or an `alignmentGuide` scheme that can see the
    container's size), and neither is a small piece of work. Nothing pins the
    divergence today either — `ios/verify` type-checks and replays, it does not
    measure.
21. **(age 0 · value low) `StackAlign` is inert outside a `ZStack` and nothing
    says so at the call site.** It is inert deliberately and by construction on
    the web (the stack imposes it, so it never reaches a flex child), and by
    omission on the natives (only the stack renderers read the field). But a
    `core.StackAlign` on a `Column`'s child compiles, merges, crosses the wire
    and does nothing on all four targets with no diagnostic. `core.SetDebugMode`
    already walks a finished tree for cursor drift and duplicate keys; a
    placement on a node whose parent is not an overlay is the same kind of
    finding, and it would join items 10 and 15 in the same pass.
22. **(age 0 · value low) A `Select`'s group headings cannot be styled or
    ordered independently.** A heading is a string on an option, so there is no
    way to give one an icon, mark a whole run disabled, or state a heading that
    has no options under it. `<optgroup disabled>` exists in HTML and both
    natives could express a disabled section; nothing in `SelectOption` can ask
    for it. Recorded rather than done because a heading with fields of its own
    is a second type, and the flat string map is what keeps all four renderers
    reading one shape.
