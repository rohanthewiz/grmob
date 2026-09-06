# Session: two node types the list was waiting on, and an ink the theme already knew

Session: https://claude.ai/code/session_01YVcrPiEYG2vnpUePWiK5T5
Date: 2026-09-06 (follows "field-frames-and-two-outlines")

## Ask

"Now do the oldest 3 items on the Next list."

The three oldest were items 1, 2 and 3 of the previous session's list: the
`<select>` border decision (age 4, low), `contrastInk` picking black on
`#007AFF` (age 3, medium), and `core` having no z-stacking primitive (age 2,
medium).

Two of the three were **contingent on a node type that did not exist**, which
is what made them cheap to leave on the list and what had to change to close
them. So the session is one small colour fix and two new node types across all
four targets.

Ordered smallest first: the ink, then `ZStack`, then `Select`.

## Phase 1 — the ink over a fill

### The bug report was right and its diagnosis was not

A selected calendar cell drew `#000000` on the DefaultTheme blue. The item
guessed the fix was "probably a third candidate rather than a new field", and
that turns out to be arithmetically impossible: `contrastInk` returns the
*maximum* contrast, and against a mid-tone nothing a theme could name outscores
black. Any fix expressed as another candidate loses the same comparison. What
had to change is the question.

    white on #007AFF   4.02:1
    black on #007AFF   5.23:1   ← the maximum, and the wrong answer

Black is the higher ratio and it is the one that reads as a rendering fault.
More to the point, the framework was already painting white on that exact blue
in every filled `components.Button` — because **the theme states the pair
outright**, in `Components.Button` (`Background: #007AFF`, `TextColor:
#FFFFFF`). One colour, two answers, one screen.

### Ask the theme, then measure

`inkOn(theme, fill)` reads the declaration first and falls through to
`contrastInk` for a fill nobody has said anything about. `Components.Button` is
the one place a palette states a fill and an ink *together* — a filled button
is the control a theme cannot describe without answering the question — which
is what makes it a declaration rather than two colours sitting near each other.
It is the same reverse lookup `ColorPalette.OnLight` performs one property
over, and the same reason `components.Chip` already read its accent off the
Button base rather than off `Colors.Primary`.

The principle is the one the on-light tones landed under a session ago: *the
widget has the number and not the authority.*

### The variant switch disappeared, and that fixed a second thing

`Variant.Ink` had an arm exempting `VariantDefault`, returning the theme's
`Background` **whatever bg was**. That is now `inkOn(t, bg)` with no switch at
all, and both bundled themes land on the identical hex because their Button
base declares Primary → `#FFFFFF` = their Background.

Two things fall out:

- `Badge{Color: "#FFF9C4"}` with no variant used to get white ink on pale
  yellow. Badge's own doc already promised the opposite ("resolved against bg,
  so an explicit Color still gets a legible ink picked for it"); it is true now.
- A theme with no `Components.Button` block at all — `examples/fintechapp`
  shipped as one for months — has declared no pairing, so its default variant
  is measured like any other. That is the one case whose pixels move, and
  towards the more legible ink.

### The cost, stated rather than hidden

White on `#007AFF` is 4.02:1, below AA for body text, and `inkOn` now returns
it where the old rule returned a passing black. That is not a contrast
regression waved through: it is the number every filled Button has always
painted, because it is the pair DefaultTheme declares. Raising it is the
*theme's* move — a darker Button base, or Apple's accessible `#0040DD` this
palette already carries as `PrimaryOnLight` — and it would lift the buttons and
the calendar together. One widget quietly disagreeing with the theme fixed
nothing and hid the question.

## Phase 2 — `core.ZStack`

### Centred, and only centred

`Compass` had wanted this for three sessions and its note named the widget that
would use it first. The contract is deliberately small: **every child drawn in
the same box, in tree order, centred on both axes, keeping its own size.** That
is the one arrangement all three constructs agree on without argument —

    SwiftUI    ZStack(alignment: .center)          (restates its default)
    Compose    Box(contentAlignment = .Center)     (overrides TopStart)
    CSS        display:grid + align/justify-items  (children in cell 1/1)

— and the alignment is *stated* in all three rather than inherited anywhere,
because an inherited default is invisible from the other two renderers.

There is no per-child alignment prop. A layer that wants to be elsewhere says
so **with its own box**: give it the stack's dimensions and lay its content out
inside it. The Compass does exactly that — the index mark is a full-height
Column justifying its glyph to the start, so the mark lands top-centre while
the Column itself is centred like everything else. One consumer is not a
vocabulary; the prop is the obvious next step if a second one wants it.

### A grid, not absolute positioning

The obvious HTML overlay is `position:relative` + `position:absolute`, and it
costs the one thing an overlay still needs. An absolutely positioned child is
out of flow, contributes nothing to its parent's size, and an unsized stack
collapses — while a SwiftUI ZStack and a Compose Box both size to their largest
child. A single-cell grid keeps the layers in flow: the track sizes to the
biggest, the rest draw in the same cell.

`core.Style` already carries `Position`, `Top/Right/Bottom/Left` and `ZIndex`,
and **only the two DOM targets read any of them** — which is exactly why the
container is a node type rather than a recipe. Anything built from those props
is a web-only widget wearing a portable name, which is the sentence
`Compass` had been carrying in its doc.

### The declaration goes on the children, which is the interesting half

A layer has no idea it is a layer, so the container stamps it.

    htmlout    the `imposed` channel — its second caller after the TabView
               pages, and for the identical reason
    runtime    syncOverlay after the children exist, plus syncTouchedOverlays
               after every patch batch, walking up as the TabView pass does

The patch pass is the half that would have shipped broken: an `add` lands a new
layer under a `ZStack` that was not itself touched, and an unplaced layer is
not invisible — CSS auto-places it into its own implicit row, *below* the
stack, which reads as a layout quirk rather than a missing declaration.

A `Fragment` layer forwards the cell to its own children rather than absorbing
it, so a `core.For` inside a stack overlays what it generated.

### The flex test had to be skipped, not extended

A `ZStack` carrying a `Gap` must not be promoted to a flex container: the props
are merely *inert* on an overlay, and answering to one would cost it the
overlay outright — a total loss of the node type's behaviour, caused by a prop
that means nothing to it. Both DOM renderers put the overlay branch **ahead**
of the flex test rather than beside it, and the runtime's ordering is pinned by
a test that compares the two string offsets.

### What the Compass got

The mark moved from the row above the rose onto its rim, drawn over it. The
rose's inset went from half a letter to a whole one to make room — the glyph is
three quarters of a letter tall — and at heading zero the mark and the N
deliberately coincide, because an index pointing at N is what facing north
looks like.

## Phase 3 — `core.Select`, and the border decision under it

### The item was an answer waiting for a question

Item 1 had been on the list for four sessions holding a decision nobody could
apply: *join `borderResetTypes` if the style is meant to own the frame, stay
out if the browser draws the control.* Closing it meant landing the picker.

### The natives decided the web's question

`<select>` joins the reset, and the deciding fact is what the **other three
targets** do. Neither native builds the picker from its platform control:

    SwiftUI    Menu + grMobBox        not .pickerStyle(.menu)
    Compose    Box + DropdownMenu     not ExposedDropdownMenuBox

Both of those idiomatic controls draw a frame, a container colour and an
indicator of their own that no Go style can remove. Taking them would have made
the picker the one control in the vocabulary whose edge came from the platform
on two targets and from the theme on the other two — the exact divergence the
reset set exists to prevent. So each native draws the style's own box and the
platform supplies only the behaviour, the frame comes from `Components.Input`
on three targets out of four, and the web was again the one place a second one
was being drawn underneath.

Only the frame is reset. A `<select>`'s drop-down indicator is not chrome
around the control, it is the thing that says the control is a picker — the
checkbox's own argument one row down — and `border` does not touch it.

### It is a field, so it reads the field's base

`Components.Input`, which is how it inherits the frame, the radius, the fill and
the padding, and why it needs no palette role. That inheritance is what makes
the reset safe rather than levelling-down: there is something to reset *to*.
Same move `components.DatePicker` makes.

### Options are a prop; the open state is nobody's

Options travel flattened as `[]map[string]string`, like `TabView`'s tabs, and
each DOM renderer builds the `<option>` elements itself. They are **chrome**:
no `data-node-path`, no patch addressed to one, and marked `data-grmob-chrome`
so the conformance replay skips them — an unmarked one shows up there as three
extra nodes Go never rendered, which is how that marking got tested.

`applySelectOptions` rebuilds only when the list itself changed, keyed on a JSON
signature (a length comparison misses a relabel). Not a performance note:
replacing a `<select>`'s options resets the control, so rebuilding on every
props patch would close an open drop-down mid-choice — and a controlled picker
gets a props patch on exactly the pass where someone has just opened it. The
value is assigned every time and always *after* the options, because a
`<select>` silently ignores a value matching none of its current ones.

Whether the menu is open is the renderer's own state on both natives. There is
no prop for it and no patch describes it; putting the flag in the tree is what
would make a picker close on every unrelated re-render.

### Two absences, both deliberate

- **No focus stamp.** `focusableLeafTypes` is about the *keyboard*, and a menu
  is not a keyboard target on either phone. The web is the outlier — a browser
  focuses a `<select>` happily, and `htmlout` exports the `autofocus` — because
  focus there is a document concept rather than a keyboard one.
- **No blur binding on `form.Select`.** A choice is a commit, not a draft, so
  there is no "still working on it" state for leaving one to end. Same reason
  `form.Checkbox` has none.

`onChange` carries the option's **Value**, never its label or its index: the
index is the one identity that changes when a list is reordered, and a label is
written to be read. So every rule reads the string unchanged and
`forms.Required` rejects an unchosen picker exactly as it rejects an empty
field.

## Verification

All six paths, green:

    go build / go vet / gofmt / go test ./... / go test -race ./...
    wasm/verify/run.sh          replay + 11 mjs suites
    ios/verify/run.sh           data + view + app layers
    android ./gradlew compileDebugKotlin --offline --rerun-tasks
    GOOS=js GOARCH=wasm go build ./...

New and changed tests:

    components/variant_test.go       the declaration read on both themes, the
                                     case-insensitive match, the undeclared
                                     fills that stay measured, both halves of
                                     a declaration required, and a theme whose
                                     button is not primary-coloured
    components/calendar_test.go      the selected ink as a *literal* #FFFFFF
    components/compass_test.go       the mark and rose as indexed layers of one
                                     ZStack, and the inset clearing the glyph
    core/zstack_test.go              the mixed argument list, paint order, the
                                     behavior prop, and no theme base
    core/select_test.go              the text callback channel, the label
                                     default, the Input base, no focus stamp
    htmlout/overlay_test.go          grid + cell, the unstyled stack, the six
                                     props that must not promote it, Display
                                     resolution, a Fragment layer, and that no
                                     other container imposes anything
    htmlout/select_test.go           options, the marked choice, an unmatched
                                     value, escaping, disabled, both border
                                     directions, and a themed picker end to end
    wasm/verify/overlay_test.go      OVERLAY_TYPES vs Go, the child area, the
                                     branch ordering, both stamp paths
    wasm/verify/overlay_test.mjs     the live grid, the styleless stack, the
                                     patch paths, and an ordinary container
                                     left alone
    wasm/verify/select_test.mjs      option building, no rebuild on a value
                                     change, rebuild on a relabel, the dispatch
                                     payload, the border reset
    mobile/verify/stacking_test.go   both ZStack arms: the construct and the
                                     stated centring
    mobile/verify/select_test.go     both dispatch arms, the forbidden platform
                                     controls, and the value key in the dispatch
    examples/tutorial/chapter5_test.go  the two picker declarations live, the
                                     absent blur binding, and the shared frame

Every new assertion was confirmed to bite by breaking the thing it guards:
`declaredInk` mutated in both directions, the runtime's `justifyItems` and
`syncTouchedOverlays` removed, the Swift alignment dropped, the Kotlin picker
swapped for `ExposedDropdownMenuBox`, the Swift dispatch pointed at `"label"`,
and the option chrome marker deleted (which fails the replay, not a unit test).

## Docs

`docs/concepts/views.md`: `ZStack` in the container list with the layer recipe
and the inert-props warning; `Select` in the leaves table; `Box` now points at
where its overlay behaviour went.

`docs/concepts/styling-and-theming.md`: a new "the ink over a fill" section —
ask the theme, then measure, with the `#007AFF` arithmetic and the honest cost;
the border-reset paragraph gains `<select>` and the reason the natives decided
it. `docs/concepts/forms.md`: `form.Select`, the two declaration shapes, and
the blur-binding note now covering both commit controls.

`docs/platforms/wasm.md`: "The overlay" (why a grid, where the child
declaration comes from, both stamp paths) and "Pickers and their options" (the
chrome marking and the rebuild signature). `docs/platforms/native.md`: two new
sections — why the picker is not a platform picker and where the open state
lives, and what an overlay is on device. `docs/platforms/exporters.md`: the
`ZStack` and `Select` bullets, plus the stale claim that `<input>`/`<textarea>`
are "deliberately excluded" from the border reset, which stopped being true a
session ago.

`docs/components.md`: the Compass's mark is on the rim now. ROADMAP gained
`core.ZStack`, `core.Select` and the ink rule, and its Compass entry lost the
limitation it was describing.

Tutorial: 4.10 teaches the overlay it now uses; **5.6 is new** — "Pickers: one
value from a list", with both declaration shapes on one screen and the frame
argument. `examples/signup` grew a plan picker, which puts a `Select` through
the WASM replay transcript end to end.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here. Value is the payoff of doing it, not the effort: `high` means
something is being worked around today or a second consumer has arrived,
`medium` means it blocks one named thing, `low` means it is a gap nobody has
bumped into yet.

1. **(age 3 · value medium) The `permission` package is dead code.**
   Commented-out Portuguese calling an `InvokeNative` that does not exist.
   Tier D's location work walks straight into it.
2. **(age 3 · value low) Three app-layer Swift files are still checked by
   nothing.** `GomobileBridge`, `GrMobApp` and `AppLifecycle` import the
   generated `Mobile.xcframework`.
3. **(age 2 · value high) `ListRow` still cannot announce its selection.** Its
   state wants `option` inside `listbox` — ARIA's pair for a selectable item in
   a collection, which also promises roving focus. A row that has taken
   `RoleListItem` for its depth cannot also become a `button`, so the listbox
   pair is the only door left.
4. **(age 2 · value low) A hand-built tab strip cannot point at its panel.**
   `aria-controls` and `aria-labelledby` are IDREFs and `Style` carries values.
   `core.TabView` covers the wired case, so this is only a gap for a strip built
   out of chips — no consumer yet.
5. **(age 2 · value low) `RoleLog` has one consumer and no test of its own.**
   `examples/chat` has no test file at all.
6. **(age 1 · value high) An accessible name on an unroled container is dropped
   on both web targets.** `<div aria-label="…">` with no role is `role=generic`,
   where ARIA prohibits the attribute. It hits every `ListRow` with an
   `AccessibilityLabel` and every `Accordion` header. Both natives honour the
   name, so this is a two-target silence, and the fix is a design question:
   which role does a named, tappable, non-button row get?
7. **(age 1 · value medium) `core` has no `aria-expanded`.** `Accordion` is the
   package's one stateful widget and its header cannot say whether it is open.
   The natural shape is `core.AccessibilityExpanded` with `SelectedState`'s
   three values, scoped by role on the web and mapped to Compose's
   `expand`/`collapse` actions.
8. **(age 1 · value medium) `Colors.Border` is still spent as a control
   boundary by `Chip`.** A `ProminenceQuiet` chip is a Surface fill on the
   page's Background (1.09:1) inside a `Border` hairline (1.26:1), so the
   control that filters the screen is close to invisible as a control. Same
   1.4.11 argument as the field frames, one widget over.
9. **(age 1 · value low) `appendRows` takes twelve positional arguments.** The
   band knobs travel together through both collections; a struct would make the
   next one an addition rather than a transposition risk.
10. **(age 1 · value low) `core` has no single-side padding props except
    `PaddingTop`.** An indent goes through a whole `EdgeInsets` in a `UseStyle`.
11. **(age 0 · value medium) `DefaultTheme` paints 4.02:1 on its own primary.**
    Now stated in one place rather than three: `inkOn` returns the theme's
    declared white on `#007AFF`, which is below AA for body text, and every
    filled `Button` plus the calendar's selected day spends it. The fix is the
    theme's — a darker `Button.Background`, most likely Apple's accessible
    `#0040DD`, which the palette already carries as `PrimaryOnLight` — and it
    would move every filled button in every app, which is why it is a decision
    rather than a patch.
12. **(age 0 · value medium) `ZStack` has no per-child alignment.** Every layer
    centres, and a layer that wants a corner wraps itself in a box sized to the
    stack (`Compass` does exactly that, twice over: a Column with an explicit
    height restating the stack's). An `AlignSelf`-shaped prop is the obvious
    shape and all three constructs have a spelling for it —
    `.frame(alignment:)`, `Modifier.align`, `align-self`/`justify-self` — but
    one consumer is not a vocabulary.
13. **(age 0 · value low) A `Select` cannot be grouped or disabled per option.**
    `<optgroup>` and a disabled `<option>` have no place in `SelectOption`, and
    neither native menu would read one without work. Nothing has asked; the
    field to add is obvious when something does.
14. **(age 0 · value low) A picker's open menu is invisible to Go.** Deliberate
    — see the phase note — but it means no test anywhere can drive a native
    picker's list, and the two `mobile/verify` pins read source text rather than
    behaviour. Same limitation the Modal's presentation has.
15. **(age 0 · value low) A styleless node skips the border reset in the WASM
    runtime.** `applyStyle` only runs when `node.Style` is non-nil, so a
    hand-built `Select` or `Button` with no Style keeps the browser's frame
    where `htmlout` writes `border:none` for it. No node core builds is ever in
    that state — every one carries at least a theme base — so this is a
    hand-assembled-tree gap, found while writing `select_test.mjs`.

Read by value instead: **high** 3, 6 · **medium** 1, 7, 8, 11, 12 · **low** 2,
4, 5, 9, 10, 13, 14, 15.
