# Session: a premise the spec did not support, and a role that had waited for its second spender

Session: https://claude.ai/code/session_01YVcrPiEYG2vnpUePWiK5T5
Date: 2026-09-06 (follows "a-name-a-reference-and-a-transcript")

## Ask

"Now do the oldest 2 items on the Next list."

The two oldest were items 1 and 2: `core` with no `aria-expanded` (age 3,
medium) and `Colors.Border` still spent as a control boundary by `Chip`
(age 3, low).

One of them **began with a wrong premise and had to be re-derived**, and the
other was a decision that had already been made and was waiting for a condition
to expire.

Ordered smallest first: the border, then the disclosure.

## Phase 0 — the premise that did not survive being checked

The Next entry for `aria-expanded` said, in its own words:

> The header now has a legal role to carry it (`group` supports
> `aria-expanded`), which removes the objection that stood in front of it.

That is false. `aria-expanded` is used in `application`, `button`, `checkbox`,
`combobox`, `gridcell`, `link`, `listbox`, `menuitem`, `row`, `rowheader`, `tab`
and `treeitem`, and inherits into `columnheader`, `menuitemcheckbox`,
`menuitemradio` and `switch`. `group` is in neither list.

Checked against MDN before any of the design was written, because the whole
item was resting on it. What it changed is the shape of the work: the objection
that stood in front of the field did not go away, it moved — and answering it
turned out to require restructuring `components.Accordion` rather than adding a
prop to it.

## Phase 1 — `ColorPalette.ControlBorder`

### The decision was made a session ago and left one condition open

The field-frame session split "a rule between things" from "the edge that
identifies a control" and put a 3:1 floor (WCAG 1.4.11) under the second. It
then declined to add a palette role, and wrote down exactly why:

> ColorPalette carries no role for the boundary because nothing outside those
> two component defaults spends it.

`components.Chip` is what made that false, and it had been false the whole
time — a `ProminenceQuiet` chip is a Surface fill inside a `Colors.Border`
hairline, which is two channels both spent at hairline weight:

                        Default          Material
    fill vs page        1.12:1           1.09:1
    old ring vs page    1.26:1           1.32:1
    new ring vs page    3.26:1           4.61:1

A filter row is *meant* to recede — that is what `ProminenceQuiet` means — but
receding is about how loud the fill and the ink are, not about whether the
control can be found.

So: `ControlBorder`, `ControlBorderColor()`, `FallbackControlBorder`, both
bundled themes declaring it, following the late-role pattern `Border`,
`Success` and `Warning` established exactly.

### The resolver that must not be the obvious one

`ControlBorderColor()` does **not** fall back to `BorderColor()`. The tempting
version — a theme that has one probably meant the other — returns precisely the
1.26:1 hairline the roles were split apart to stop a control being drawn in, and
it would look like it was working. `TestTheControlBoundaryDoesNotFallBackToThe
Divider` uses a palette with a *stated* `Border` and no `ControlBorder`, which
is the case that would hide it.

### Three copies of one decision, and the type system cannot hold them

`Components.Input` and `Components.TextArea` state their frame as a literal hex
and still do: a component default is a `Style` **value**, so it cannot call a
resolver. The role and the two literals are therefore three copies of one
decision, pinned by `TestBundledFieldFramesAreTheControlBorderRole`. The failure
guarded is a retint — somebody darkens a theme's fields, leaves the role alone,
and every chip in that theme keeps the old edge while every field beside it
moves, with nothing failing anywhere.

`TestTheDividerAndTheBoundaryAreDifferentTones` guards the other direction: a
theme that tints both the same has silently un-split them.

### The number under the floor, stated rather than rounded off

A chip has two backdrops and DefaultTheme's boundary clears one of them:

    #8E8E93 vs #FFFFFF (the page)          3.26:1   ✅
    #8E8E93 vs #F2F2F7 (the chip's fill)   2.92:1   ✗

The edge that *identifies* the pill is the outer one — the fill is 1.12:1
against the page and identifies nothing, so what a reader picks the control out
by is the ring against the page. The inner edge is a boundary between two parts
of one control. Closing the last 0.08 would mean darkening past Apple's own
systemGray, which is the repaint-the-theme's-choice move the on-light tones
exist to avoid. The contrast test measures the page alone and says so.

### The reverse lookup, and why it is not `Components.Input.BorderColor`

Reading the Input base would have got the right pixels today — both themes hold
the same hex in both places. It would also have tied a chip's edge to a text
field's, so a theme restyling its fields would silently restyle its chips. The
role is the thing both of them name; a widget that wants to *look like* a field
still reads the base (that is how `DatePicker` gets radius, fill and edge in one
prop), and this role is for a widget that wants only the edge.

## Phase 2 — `core.ExpandedState` and `AccessibilityExpanded`

### Why it is a second type and not `SelectedState` reused

Identical values, and reusing the type would have compiled, exported and
rendered. Two grounds, and the second is the one that bites:

**They are independent facts about one node.** A menu button can be both the
current tab and showing its submenu, so they are two fields — and two fields
typed the same are two fields a caller can transpose.
`AccessibilitySelected(ExpandedOpen)` would type-check.

**They are scoped to different roles**, and the lists overlap without matching:

    role                       selection        disclosure
    button                     aria-pressed     yes
    tab, row, columnheader     aria-selected    yes
    option                     aria-selected    no
    link, listbox              no               yes

An `option` is a leaf choice; the thing that expands is the `listbox` around it.
A shared guard would be wrong at four roles and wrong silently, which is why
there are two switches.

`TestTheTwoStateTypesShareValuesAndNotAType` pins the premise from both ends:
the two must stay equal as strings and unequal as interface values, so a future
`type ExpandedState = SelectedState` fails by name.

### Three values again, arriving from the other end

`SelectedState`'s third value separates "off" from "not selectable". This one
separates "closed" from "not a disclosure", and losing it is worse: a collapsed
section that answers nothing is announced as an *ordinary button*, so a reader
is told they can press it and not that there is anything behind it. "Collapsed"
is the whole of what invites the press.

### The near miss that is deliberately not covered

`aria-expanded` says the content is here, in the page, and can be shown or
hidden. A trigger that opens a **modal** is a different relationship — ARIA
spells that `aria-haspopup`, which this vocabulary does not carry — so
`components.DatePicker`'s trigger, which looks exactly like a disclosure and
even flips a glyph, states nothing. It was the obvious second consumer and it
is the wrong one.

### The natives disagree, which is the reverse of the usual split

Everywhere else the two phones agree and the web is strict because ARIA scopes
its attributes. Here:

    web        aria-expanded, six roles plus the Button node type
    Compose    expand()/collapse() semantics *actions*
    SwiftUI    nothing at all

**Compose's is an action, not a property**, which is why it is the one
accessibility mapping that does not live in `GrMobStyle.boxModifier`'s semantics
lambda beside `grMobRole` and `grMobSelected`. An action has to *do* something,
and the only thing that can open the section is the callback the node's tap
already runs — which lives on the node, not the style. So `grMobDisclosure` is
in `Renderer.kt`'s `gestureModifier`, the one place holding both halves.

The consequence is stated rather than hidden: **a node with a state and no
`OnClick` gets neither action.** An expand action TalkBack can invoke and
Compose cannot perform is worse than a disclosure that is merely quiet.

Which action goes with which state is the inverse of the state and is this
mapping's one invisible bug — both arms compile, both wire the same callback,
and both toggle correctly on activation; only the announcement is wrong. The
test matches the state literal and its action as *one string*, because two
`Contains` checks for `expand` and `collapse` pass on the swapped version.

**SwiftUI's near miss is sharper than the IDREF pair's was.**
`accessibilityIdentifier` was a test selector wearing an accessibility name;
`accessibilityValue` is a real accessibility channel that SwiftUI's own
`DisclosureGroup` genuinely uses for this. What makes it wrong for a *framework*
is that the renderer would have to supply "expanded" / "collapsed" in English,
for every app in every locale, in a slot the app may want for a real value. It
is the move `components.Chip`'s `", selected"` name suffix was deleted for.

## Phase 3 — `components.Accordion` becomes ARIA's accordion

### The third arrangement of one header, and why the second looked right

    v1  Row, unroled + named     name on `generic`: prohibited, pruned by every
                                 browser, read out perfectly by both phones
    v2  Row, group + heading     name legal, heading on the words, and no way
        on the words             at all to say the section is open
    v3  Box(heading) > Row(button + expanded)

v2 was a deliberate design decision with a written argument (`role="button"`
makes children presentational, so the heading inside would stop being one) and
a test asserting `role="group"` and the *absence* of `role="button"`. It was
right about the mechanism and wrong about the outcome: `group` is not one of
the six roles `aria-expanded` is defined for, so the choice was between a header
that announces its name and one that announces its state — and a `group` is not
something a reader is told they can press at all.

### The objection that dissolved

The widget had already considered ARIA's own nesting and turned it down:

> a heading takes its name from its content, so a heading around this row is
> named "▸ What is a hook"

True of a heading with no name of its own. `core.AccessibilityLabel` overrides
content-derived naming, and the `Title` is right there. So:

    Box  role=heading  aria-level=3  aria-label="What is a hook"
      Row  role=button  aria-expanded="false"  aria-label="What is a hook"
        "▸"  "What is a hook"        presentational, inside the button

A reader hears the question twice — once as an outline entry to jump to, once
as a control that says collapsed or expanded. That is what every accessible
accordion on the web does, for the same reason: the two facts belong to two
elements.

The chevron was previously left deliberately audible, because it was the only
thing on screen that said which way the disclosure pointed. It is presentational
now — by construction rather than by choice, being inside a button.

A `Header` slot gets the button and its state and **no** heading, which is
exactly what it got before, on `Card.Title`/`Card.Header`'s division.

### The package convention that grew an exception

`components/heading_test.go`'s helper said headings live "on the words, never on
the row around them". That is still true of every widget but this one, and the
helper now checks for an explicitly-named heading node first and falls through
to the Text node — so the exception is stated where the rule is, rather than the
rule quietly becoming false.

## Verification

All six paths, green:

    gofmt / go build / go vet / go test ./... / go test -race ./...
    wasm/verify/run.sh          replay + 12 mjs suites (a11y now 33 tests)
    ios/verify/run.sh           data + view + app layers
    android ./gradlew compileDebugKotlin --offline --rerun-tasks
    GOOS=js GOARCH=wasm go build ./...

New and changed tests:

    core/expanded_enum_test.go          the census, the zero value, ARIA's
                                        spellings, ExpandedWhen's false half,
                                        and the two types sharing values and
                                        not a type
    core/theme_test.go                  the field frames pinned to the role,
                                        the divider and the boundary held
                                        apart, and the fallback that must not
                                        be BorderColor()
    components/chip_test.go             the quiet ring is the boundary role and
                                        not the divider, and it clears 3:1
                                        against the page under both themes
    components/accordion_test.go        the three facts on the two elements as
                                        whole substrings, the heading's name
                                        without the chevron, exactly one
                                        heading, both halves of the state, and
                                        a custom header keeping the disclosure
    components/heading_test.go          the helper's new first lookup
    htmlout/export_test.go              the role census with the four
                                        divergences called out, the Button node
                                        type, the group that is *not* a rescue
                                        here, hidden winning, and a selection
                                        and a disclosure on one node
    wasm/verify/a11y_test.mjs           the same live, plus the patch sequence
                                        a static export cannot have
    wasm/verify/a11y_test.go            ariaExpanded's arms and the
                                        unconditional write
    mobile/verify/expanded_test.go      Compose parsed/mapped/called/covered
                                        and the state-action pairing as one
                                        string; iOS's absent parse and both
                                        halves of its note
    examples/tutorial/chapter4_test.go  three headers seeded open/shut/shut and
                                        the state following a tap
    examples/todoapp/chip_migration_test.go  the legacy reference's ring

**Twenty-four break-tests, all caught.** The chip's ring back to the divider;
`ControlBorderColor` falling back to `BorderColor`; DefaultTheme's Input frame
drifting off the role; MaterialTheme un-splitting the two roles; `option`
wrongly taking `aria-expanded`; the Button node-type arm removed; the attribute
never written at all; `ExpandedState` aliased onto `SelectedState`; the runtime
losing its `link`/`listbox` arms, guarding the write, and never reading the
field; Kotlin with expand and collapse transposed, with the expand arm gone,
with the mapping never called, and with the parse dropped; Swift losing the
note, un-backticking `accessibilityValue`, and reading the key it says it does
not; the accordion dropping the state, stating only the open half, losing the
wrapper's explicit name, going back to a group, losing the wrapper entirely, and
stamping a heading over a custom header.

Break-tests were done with per-file `cp` snapshots, per the previous session's
lesson. No `git checkout` was run.

## Docs

`core/theme.go`: `ControlBorder`'s field doc with the two-step history and the
measured table, `Border`'s claim about having no palette role rewritten now that
it does, `FallbackControlBorder`, and both bundled palettes with both backdrops
measured.
`core/expanded.go`: the type, the two grounds for it being a second type, and
the three-values argument arriving from the other end.
`core/style.go`: the field doc with the three-way role table, the dialog near
miss, and the two natives' asymmetry.
`core/style_props.go`: the setter, naming the four roles the two guards
disagree at.

`htmlout/export.go`: `ariaExpanded` with the full ARIA list, the overlap stated
as a table, and why there is no `group`-shaped rescue.
`wasm/grmob-runtime.js`: the twin, and why it returns a string where
`ariaSelected` returns a pair.
`Renderer.kt`: `grMobDisclosure` — why it is not in `boxModifier`, what a node
with no handler gets, and which action pairs with which state.
`GrMobStyle.swift`: the gap, `accessibilityValue` named as the near miss, and
why an English literal is the wrong answer.

`components/chip.go`: the quiet treatment's numbers, the role-not-the-base
argument, and the 2.92:1 stated with what it costs.
`components/accordion.go`: the three arrangements, the objection that dissolved,
and the shape drawn out.

`docs/concepts/styling-and-theming.md`: `ControlBorder` in the role table and in
its own "`Border` is not `ControlBorder`" block with the measured table; a new
`AccessibilityExpanded` section with the role comparison, the dialog near miss,
and the per-target table.
`docs/components.md`: `Chip`'s "neither answer is invisible" note, and
`Accordion`'s new "ARIA's accordion shape" subsection.
`docs/platforms/exporters.md`: one bullet covering all three state attributes
and their three role lists.
`docs/platforms/wasm.md`: "Three state attributes, two style fields" — why one
returns a pair and one a string, and why both are written unconditionally.
`docs/platforms/native.md`: `AccessibilityExpanded` — Compose's action mapping
and where it had to live, and SwiftUI's near miss.

`ROADMAP.md`: the disclosure entry, and `ControlBorder` appended to the field
frame entry.

Tutorial: **4.4 grew a disclosure half** — why the heading is the package's one
exception, the exported shape, `ExpandedWhen`'s two halves, and the native
divergence. **7.x** gained the `ControlBorder` swatch and the "a second spender
is what a role is for" paragraph.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here. Value is the payoff of doing it, not the effort: `high` means
something is being worked around today or a second consumer has arrived,
`medium` means it blocks one named thing, `low` means it is a gap nobody has
bumped into yet.

1. **(age 4 · value low) `appendRows` takes twelve positional arguments.** The
   band knobs travel together through both collections; a struct would make the
   next one an addition rather than a transposition risk.
2. **(age 4 · value low) `core` has no single-side padding props except
   `PaddingTop`.** An indent goes through a whole `EdgeInsets` in a `UseStyle`.
3. **(age 3 · value medium) `DefaultTheme` paints 4.02:1 on its own primary.**
   `inkOn` returns the theme's declared white on `#007AFF`, below AA for body
   text, and every filled `Button` plus the calendar's selected day spends it.
   The fix is the theme's — most likely Apple's accessible `#0040DD`, which the
   palette already carries as `PrimaryOnLight` — and it would move every filled
   button in every app, which is why it is a decision rather than a patch.
4. **(age 3 · value medium) `ZStack` has no per-child alignment.** Every layer
   centres, and a layer that wants a corner wraps itself in a box sized to the
   stack. An `AlignSelf`-shaped prop is the obvious shape and all three
   constructs have a spelling for it, but one consumer is not a vocabulary.
5. **(age 3 · value low) A `Select` cannot be grouped or disabled per option.**
   `<optgroup>` and a disabled `<option>` have no place in `SelectOption`, and
   neither native menu would read one without work.
6. **(age 3 · value low) A picker's open menu is invisible to Go.**
   Deliberate, but it means no test can drive a native picker's list, and the
   two `mobile/verify` pins read source text rather than behaviour.
7. **(age 3 · value low) A styleless node skips the border reset in the WASM
   runtime.** `applyStyle` only runs when `node.Style` is non-nil. No node
   `core` builds is ever in that state, so this is a hand-assembled-tree gap.
8. **(age 2 · value high) Neither a `listbox` nor a `tablist` has keyboard
   navigation on the web, and there are two consumers now.** The roles state
   semantics ARIA's patterns pair with behaviour: the container takes focus,
   the arrow keys move the active item, and a roving tabindex or
   `aria-activedescendant` says which. `examples/mobileapp`'s article list is
   the listbox; `examples/social`'s bottom bar and tutorial 4.5 are the
   tablists, and all of them hand a keyboard user a role that claims more than
   the widget does. Needs a focus concept `core` does not have —
   `core/focus.go` is about putting the cursor in a named field. Both phones
   navigate by swipe and lose nothing. **An accordion header is now a third
   consumer of the same gap in a different form**: it is a real `button` on the
   web and therefore already keyboard-operable, which is the half this item is
   about — so it *narrows* rather than widens.
9. **(age 2 · value medium) Nothing re-checks a permission on foreground.**
   A user can grant one in Settings and come back, and no platform says so.
   `hooks.UsePermission` deliberately does not (it cannot see whether its
   screen is still on top, and five screens would each fire a check per
   resume), so every consumer writes the same `UseLifecycle` pairing by hand.
   A `hooks.UsePermissionLive` — or a `RecheckOnForeground` option — is the
   obvious shape; what is missing is a rule for which screen owns it.
10. **(age 2 · value low) The Android "asked before" flag does not survive a
    process restart.** It is what separates a permanent refusal from "never
    asked", so after a restart a permanently-denied permission reads as
    `Prompt` until the next request proves otherwise — one dead button press.
11. **(age 2 · value low) A browser `Request` opens the device to answer.**
    There is no request API, so `getUserMedia` is the only thing that prompts,
    and a granted camera check has genuinely opened the camera for a moment. A
    refused one also reads as `denied` whether the user pressed Block or
    dismissed the prompt, because `NotAllowedError` does not say which.
12. **(age 2 · value low) The gomobile stub's types are unchecked.** The pin
    holds the *names* — every bindable symbol declared, nothing extra — because
    copying gobind's type mapping into a test would be reimplementing gobind. A
    wrong signature fails the Swift typecheck the moment the shell calls it, so
    what is genuinely unguarded is a stub declaration the shell never touches.
13. **(age 1 · value medium) A hand-built tab strip's panel still cannot say it
    is one.** `AccessibilityControls` closes the pointing and `RoleTabPanel` is
    now defensible. What stands in the way is mechanical: the WASM runtime tells
    its own wiring apart from an author's role by `"tabpanel"` not being a
    `core.Role`, held there by `TestNoRoleCollidesWithTheTabPanelWiring`.
    Replacing that discriminator with a `data-grmob-panel` marker (which both
    web targets would write, as they already do `data-grmob-chrome`) is the
    right fix and unblocks the constant. Two native arms and a large doc block
    go with it.
14. **(age 1 · value medium) Three widgets take `group` where ARIA has a better
    role.** The supplied fallback made their names audible and stopped there.
    `ProgressBar` wants `progressbar` with `aria-valuenow`/`valuemin`/`valuemax`
    — it is already computing the percentage into its name, which is the
    workaround. `Skeleton` wants `status`, since "Loading" is an advisory that
    is replaced. `FormField`'s required marker is a glyph standing in for a
    word, which is `img`'s shape. Each is a widget change plus, for
    `ProgressBar`, a value vocabulary `core` does not have.
15. **(age 1 · value medium) Nothing catches a dangling `aria-controls` or a
    duplicate `AccessibilityID`.** Both are written verbatim and neither
    exporter can see the whole document at the moment it writes one — but a
    finished tree *can* be walked, which is exactly what `core.SetDebugMode`
    already does for cursor drift and duplicate keys. A debug-mode pass
    reporting an id claimed twice and a reference resolving to nothing would
    catch the one failure mode the pair has, and it is a failure that is
    invisible on every target rather than merely quiet on two.
16. **(age 1 · value low) An `AccessibilityID` is not validated.** An id
    containing a space is invalid HTML; an empty one is written as `id=""`; one
    starting with `grmob-` collides with `core.TabView`'s minted ids and breaks
    a wiring the author never wrote. The prefix is documented as reserved and
    nothing enforces it. A drop would be silent too, so the honest fix is
    probably the debug-mode pass in item 15 rather than a guard in the
    exporters.
17. **(age 0 · value medium) A Next-list item asserted a spec fact that was
    wrong, and nothing could have caught it.** "`group` supports
    `aria-expanded`" was written into the list a session ago, survived three
    re-sorts, and was false. The item it justified was still worth doing and the
    work it implied was different. Every ARIA claim in this framework's docs is
    hand-checked prose, and there are now dozens — three role lists, two name
    prohibitions, four `aria-level` scopes, six `aria-selected` roles. A
    generated table (from the ARIA spec's own machine-readable role definitions,
    checked in as a fixture) would make the claims testable instead of
    reviewable. It is a build-time concern rather than a runtime one, which is
    why it is medium rather than high.
18. **(age 0 · value medium) `Accordion` is the only disclosure, and
    `AccessibilityExpanded` has no second consumer.** One consumer is not a
    vocabulary, and the field's role list carries five arms nothing reaches —
    `link`, `listbox`, `row`, `columnheader` and `tab`. The shapes that would
    reach them are real and absent: a `GroupedList` band that collapses (row),
    a combobox built out of `SearchField` plus a listbox (link/listbox), a
    `DataTable` with collapsible column groups (columnheader). Each is a widget
    rather than a prop, so this is a note about where the next one should look
    rather than a task.
19. **(age 0 · value low) `Colors.ControlBorder` is 2.92:1 against
    `Colors.Surface` under `DefaultTheme`.** The outer edge clears 3:1 and is
    what identifies a quiet chip, which is argued in two places and asserted in
    one — but any future widget that draws a boundary *on* a Surface fill
    inherits the shortfall without inheriting the argument. The honest fixes are
    both theme decisions: darken past Apple's systemGray, or give the quiet chip
    a fill that is not Surface.
20. **(age 0 · value low) An `ExpandedState` on a node with no handler is
    silently inert on Android.** `grMobDisclosure` is in `gestureModifier`,
    which returns early when a node carries neither `onClick` nor `onLongPress`,
    so the state reaches the renderer and buys nothing. That is deliberate — an
    action nothing can perform is worse than none — but it is a rule stated only
    in a comment, where the equivalent web rule (a state on an unroled node is
    dropped) has a test on both targets. `core.SetDebugMode` reporting a
    disclosure with no way to open it would put it where the other two are.

Read by value instead: **high** 8 · **medium** 3, 4, 9, 13, 14, 15, 17, 18 ·
**low** 1, 2, 5, 6, 7, 10, 11, 12, 16, 19, 20.
