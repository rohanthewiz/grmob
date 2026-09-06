# Session: a frame the browser was drawing for us, and two outlines nobody could see

Session: https://claude.ai/code/session_01YVcrPiEYG2vnpUePWiK5T5
Date: 2026-09-06 (follows "selected-state-and-on-light")

## Ask

"Now do the oldest 3 items on the Next list."

The three oldest were items 1, 2 and 3 of the previous session's list: heading
levels 3-6 reachable and unexercised (age 4, low), the themes giving
`Components.Input` no border (age 3, medium), and `AccessibilityNestingLevel`
with no consumer (age 3, low).

Three phases, the border first because it is the one with a cross-target
mechanism under it and the other two are both about `aria-level`.

## Phase 1 — the field frame

### The item was a deadlock, and the theme was the half that had to move

`borderResetTags` writes `border:none` for the tags a browser draws one on,
because on three targets "no border in the style" means no border on screen and
on the web it meant "whatever the user agent draws". It held `button` alone.
`<input>` and `<textarea>` were excluded with a note: neither bundled theme gave
`Components.Input` a border, so resetting theirs would have left every web text
field an unmarked rectangle.

**The phones were already right and looked wrong.** Compose composes a
`BasicTextField` through the ordinary `boxModifier` and SwiftUI a `.plain`
`TextField` through `grMobBox` — both draw exactly what the style asks for, and
what it asked for was nothing. So the missing frame was a *theme* bug that the
browser had been papering over on one target out of four. Verified before
touching either renderer; neither needed a line.

### The tone is a boundary, not a hairline

The obvious fill was `Colors.Border`, and it is wrong by a number. Chrome draws
an input border at `rgb(118,118,118)` — 4.54:1 — and the palette's hairline is
`#E5E5EA` (1.26:1) or `#E0E0E0` (1.32:1). Replacing the first with the second
is the levelling-down the original note was written to prevent, in a different
direction.

WCAG **1.4.11 Non-text Contrast** is the line: a rule *between* things is
decoration and a pale one is a legitimate choice, while the edge that says
*this rectangle is a field you can type in* is the only thing identifying a
control, and that carries a 3:1 floor. One hex cannot be both — the same shape
as the on-light tones' argument one property over.

    DefaultTheme    #8E8E93  iOS systemGray   3.26:1 on its own white
    MaterialTheme   #757575  MD grey 600      4.61:1 on white and on #FAFAFA

Both are measured against **two** backdrops, because a field has two: the page
behind it and its own fill. Under DefaultTheme those are the same white and the
border is the entire control; under MaterialTheme the fill is a shade off the
page.

**No new palette role.** The frames live in `Components.Input` and
`Components.TextArea`, and `ColorPalette.Border`'s doc now says it is a divider
and not a field's edge. Nothing outside those two component defaults spends the
boundary: a widget that wants to look like a text field reads the `Input` base
itself, which is also how it gets the radius and the fill.

### The set became node-type-keyed, which the old comment had predicted

`borderResetTags` said "a set rather than a `tag == "button"`, because the
question it answers is per-tag and the answer will change when the theme
question above is settled." It changed further than that: **five node types
share `<input>` and only three of them want the reset.**

    Input, InputPassword, NumericInput   a frame the style should own
    TextArea                             the same, one tag over
    Checkbox, Slider                     the user agent draws the *control*

A checkbox's border is not chrome around the control, it is the box; a range
track has none. Both draw through `appearance: auto`, where a browser ignores
the property — so a tag-keyed set would have been harmless *today* and wrong on
the day someone reaches for `appearance:none`. `borderResetTypes`,
`ResetsUABorder(nodeType)`, `BorderResetTypes()`, and `BORDER_RESET_TYPES` in
the runtime, whose call loses its `tagForType(...)` in the process.

### Two widgets had to react, in opposite directions

- **`SearchField` flattens harder.** The row is the frame; the field inside it
  already dropped its fill, corners and inset and now drops its width too. It
  was previously wearing the *browser's* border inside the row, which the fill
  mostly hid and which no `BorderWidth(0)` could have removed anyway.
- **`DatePicker` stops restating.** Its trigger used to add a hairline in
  `Colors.Border` on top of the `Input` base, because the base carried none.
  Restating it would now have quietly *paled* the picker's edge below the
  fields beside it — so the four props became one `UseStyle`, and the test
  asserts against the base rather than a literal.

### What a theme that predates this gets

A borderless field on the web, exactly as it always had on both phones. That is
the point of the reset rather than a casualty of it, and it is written down in
`borderResetTypes` where the previous exclusion note lived.

## Phase 2 — the outline past 2

`core.AccessibilityHeadingLevel` has run 1..6 since it existed and the framework
only ever said 1 (an `AppBar` title) or 2 (a `GroupedList` band). A `Card` title
and an `Accordion` title both drew at heading weight and announced as prose, so
a reader navigating by heading fell from the section straight to the end of the
screen.

    AppBar.Title      1   fixed by construction
    GroupHeader       2   a band titles a run of rows within a screen
    Card.Title        2   a card is a section of a screen, the same tier
    Accordion.Title   3   a disclosure sits inside one of those

**Three of the four are defaults and one is not, and the split is the design.**
A widget can state its own tier only where its position is fixed. An AppBar's
is: it belongs to the screen it bars. The other three are *usually* where the
table says and can legitimately be anywhere, so they take a `HeadingLevel`
field whose zero value is the tier above. That also retires an acknowledged
wrong — `GroupHeader` used to document a bandless feed on a barless screen as
starting at 2 with no 1 above it, "the lesser of the two wrongs". The caller in
that position now writes 1.

Levels **4 to 6** are a caller's, which is the honest answer: nothing in the
framework is four tiers deep by construction, and pretending otherwise would be
inventing a widget to justify a range.

**A negative level asks for a heading with no tier at all**, and it needs no
case in the resolution. `core` drops rather than clamps an out-of-range level,
so the value survives to the exporters and is written by none of them — which
is exactly the announcement every heading in the package made before it had a
tier. A rule that special-cased it would have had to pick a spelling, and the
range rule already has one.

**Two rules shared by all three.** The field applies to the *default* content
only — a `Card.Header` or an `Accordion.Header` replaces the line and is the
caller's to describe — and the role rides the **words**, never the row, so a
band's heading is "March" and not "March, 12" and an accordion's is "Advanced
options" and not "▸ Advanced options".

That last one is also the answer to ARIA's disclosure pattern, which nests a
heading *around* a button carrying `aria-expanded`. Neither half helps: a
heading takes its name from its content, so wrapping the row brings the chevron
back into the name, and `core` has no vocabulary for an expanded state at all.
Which is why the accordion's chevron is now documented as deliberately *not*
hidden — it is decoration by every other measure in the package, and the only
thing in the row that says which way the disclosure is pointing.

`headingLevel`, the two tier constants and `headingProps` live in a new
`components/heading.go`, which is where the outline is written down.
`TestTheDefaultHeadingOutline` spells the tiers as **literals** rather than
reading the constants back, which was the first version and asserted only that
a constant equals itself.

The tutorial is the live consumer: a lesson screen has no AppBar, so
`lessonHeader` takes level 1 and `keyPoints` takes 2 at their call sites, and
the accordion lesson's FAQ takes 3 from the widget default. All three tiers on
one screen, pinned end to end through the running app — the relationship is the
thing being checked, and it only exists once three widgets from three files are
on one screen.

## Phase 3 — the nesting level's first consumer

The item said to watch whether the first nested list downstream reached for it.
None had, and the reason was structural rather than incidental: the framework
has no nested-collection widget, and the one candidate — `ListRow` — was
carrying a refusal from the previous session.

**That refusal turns out to be about the role, not the widget**, and reading it
the other way is what unblocked this:

    RoleButton    true only of a tappable row, and a foreign child of any
                  role="list" it sits in
    RoleListItem  the honest description of a row, and ARIA defines neither
                  *state* attribute for it

`aria-level` is defined for exactly `heading`, `listitem` and `row`. So the
depth's role is one the row can honestly take and the selection's is not — a
selectable collection item is an `option` in a `listbox`, which `core.Role`
still does not carry. Two fields, one widget, opposite answers, and the
difference is a property of the roles.

`ListRow.NestingLevel` makes the row a `listitem` at that depth. It is what a
flattened outline cannot say any other way: a list is a flat run of siblings —
which is also what makes it virtualizable — so an indent is pixels a screen
reader never sees and the nesting has to travel as data.

**Opt-in, and that is the ownership rule rather than caution.** A `listitem` is
owned by a `list`, and a row cannot see its container, so the two halves are set
by different people: the caller roles the list, the widget states the depth. A
row asked for neither is the unroled Box it has always been, which is what keeps
every existing list in every app from quietly growing orphan roles.

One cost, stated because it is real: a row inside a `role="list"` must not also
be a `role="button"`, so a tappable row in an outline announces as an item at a
depth rather than as a control. Nothing is lost — a `ListRow` has never carried
`RoleButton` — but it means this field and a future tappable-row role are in
tension.

Tutorial 4.3 gained the demo: six rows, one `core.List` carrying `RoleList`,
three depths, and an indent derived from the depth rather than stored beside it,
which is the whole argument in one line. The end-to-end test checks the pairing
(a list above the items), the depths, and that there are at least three distinct
ones — a version that read the expected depths out of the same slice would have
kept passing if every row said 1.

## Verification

All six paths, green:

    go build / go vet / gofmt / go test ./... / go test -race ./...
    wasm/verify/run.sh          replay + 9 mjs suites
    ios/verify/run.sh           data + view + app layers
    android ./gradlew compileDebugKotlin --offline --rerun-tasks
    GOOS=js GOARCH=wasm go build ./...

New and changed tests:

    components/variant_test.go       both field frames clear 1.4.11's 3:1 on
                                     both backdrops; and the census that the
                                     divider role still does not, which is why
                                     the frame is separate
    htmlout/export_test.go           the reset set spelled out as a list; the
                                     checkbox and slider that stay out; a
                                     themed field keeping its own frame
    wasm/verify/border_test.go       the Go/JS set comparison, renamed; plus
                                     that every member is a real node type
    wasm/verify/border_test.mjs      three field types reset live, a themed
                                     field keeps its frame, checkbox and
                                     slider keep the browser's
    components/heading_test.go       the outline as a table, the overrides,
                                     the negative level, the two Header slots,
                                     and the band level through both hosts
    components/list_row_test.go      the depth, the zero value, and that the
                                     depth does not drag the selection in
    components/search_field_test.go  the flattened width, plus a guard that
                                     the base had a frame to flatten
    components/date_picker_test.go   the trigger inherits rather than restates
    examples/tutorial/chapter4_test.go  the 1-2-3 lesson outline and the
                                     flattened-tree demo, both live

Every new assertion was confirmed to bite by breaking the thing it guards and
watching it fail. Two were vacuous on the first attempt and were rewritten: the
outline table read its own constants, and the border walk agreed with the set
whatever the set said.

## Docs

`docs/concepts/styling-and-theming.md`: the palette table's `Border` row is now
"a divider, not a control boundary", with the 1.4.11 argument beside the
`Surface` and `Success` distinctions; the border-reset paragraph names node
types; `AccessibilityHeadingLevel` gained "The package's heading outline" as a
table plus the negative-level escape; `AccessibilityNestingLevel`'s "nothing in
the framework sets one" became the `ListRow` recipe and the ownership rule.

`docs/components.md`: Card and Accordion gained their tiers, GroupedList gained
`HeadingLevel`, ListRow gained `NestingLevel` with the code sample, SearchField
mentions the fourth thing it flattens. `docs/platforms/wasm.md`: the set,
renamed and explained. `docs/platforms/native.md`: two new sections — the text
field's frame (nothing to do, and why that is the finding) and what a
flattened outline does on device (the role lands on an empty arm, the depth is
not parsed). ROADMAP gained three checked items. Tutorial 4.3 teaches the
depth, 4.4 the outline, 7.3 the divider/boundary split.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here. Value is the payoff of doing it, not the effort: `high` means
something is being worked around today or a second consumer has arrived,
`medium` means it blocks one named thing, `low` means it is a gap nobody has
bumped into yet.

1. **(age 4 · value low) A `<select>` will need the border decision** if a
   picker node type lands. It now has an answer waiting rather than an open
   question — join `borderResetTypes` if the style is meant to own the frame,
   stay out if the browser draws the control — but it is still contingent on a
   node type that does not exist.
2. **(age 3 · value medium) `contrastInk` picks black on `#007AFF`.** A
   selected calendar cell's numeral and dots come out `#000000` on the
   DefaultTheme blue. `contrastInk` chooses between the theme's two ink roles
   for a colour used as a fill, which is the other half of the pair the
   on-light tones complete. The fix is probably a third candidate rather than
   a new field.
3. **(age 2 · value medium) `core` has no z-stacking primitive.** `Compass`
   wanted a needle over its rose and could not have one. A `core.Stack` node
   type would be a Compose `Box` / SwiftUI `ZStack` / `position: relative`
   plus absolute children.
4. **(age 2 · value medium) The `permission` package is dead code.**
   Commented-out Portuguese calling an `InvokeNative` that does not exist.
   Tier D's location work walks straight into it.
5. **(age 2 · value low) Three app-layer Swift files are still checked by
   nothing.** `GomobileBridge`, `GrMobApp` and `AppLifecycle` import the
   generated `Mobile.xcframework`.
6. **(age 1 · value high) `ListRow` still cannot announce its selection.**
   Unchanged by this session, which took the *depth* half of the same table.
   Its state wants `option` inside `listbox` — ARIA's pair for a selectable
   item in a collection, which also promises roving focus. Now slightly
   harder, and honestly so: a row that has taken `RoleListItem` for its depth
   cannot also become a `button`, so the listbox pair is the only door left.
7. **(age 1 · value low) A hand-built tab strip cannot point at its panel.**
   `aria-controls` and `aria-labelledby` are IDREFs and `Style` carries
   values. `core.TabView` covers the wired case, so this is only a gap for a
   strip built out of chips — no consumer yet, and the fix would be a
   node-path-derived id scheme both DOM targets agree on, which is what
   TabView already has internally.
8. **(age 1 · value low) `RoleLog` has one consumer and no test of its own.**
   `examples/chat` has no test file at all, so the adoption is checked only by
   the vocabulary-wide export test.
9. **(age 0 · value high) An accessible name on an unroled container is
   dropped on both web targets.** `<div aria-label="…">` with no role is a
   `role="generic"` element, where ARIA prohibits the attribute and readers
   ignore it — the exact rule `RoleImg` was added to close for images, and the
   one `AccessibilitySelected` and `aria-level` are already scoped by. Verified
   against the exporter. It hits every `ListRow` with an `AccessibilityLabel`
   (including the `", selected"` suffix the last session deliberately kept
   there) and every `Accordion` header. Both natives honour the name, so this
   is a two-target silence, and the fix is a design question rather than a
   patch: which role does a named, tappable, non-button row get?
10. **(age 0 · value medium) `core` has no `aria-expanded`.** `Accordion` is
    the package's one stateful widget and its header cannot say whether it is
    open; the chevron glyph is currently the only signal, which is why it is
    documented as deliberately unhidden. The natural shape is
    `core.AccessibilityExpanded` with `SelectedState`'s three values, scoped
    by role on the web and mapped to Compose's `expand`/`collapse` actions —
    the second consumer would be a disclosure-style `ListRow`.
11. **(age 0 · value medium) `Colors.Border` is still spent as a control
    boundary by `Chip`.** This session split divider from boundary for text
    fields and stopped there. A `ProminenceQuiet` chip is a Surface fill on
    the page's Background (1.09:1) inside a `Border` hairline (1.26:1), so the
    control that filters the screen is close to invisible as a control. Same
    1.4.11 argument, one widget over; `DataTable`, `Compass`, `Separator` and
    `Skeleton` outline containers and rules and are right as they are.
12. **(age 0 · value low) `appendRows` takes twelve positional arguments.**
    The band knobs — header, hideTrailingCount, sticky, headingLevel — are
    four of them and travel together through both collections. A struct would
    make the next one an addition rather than a transposition risk; the
    heading-level test exists partly because that risk is real.
13. **(age 0 · value low) `core` has no single-side padding props except
    `PaddingTop`.** The set is `Padding`, `PaddingTop`, `PaddingHorizontal`,
    `PaddingVertical`, so an indent — the tutorial's outline demo wants one —
    goes through a whole `EdgeInsets` in a `UseStyle`.

Read by value instead: **high** 6, 9 · **medium** 2, 3, 4, 10, 11 · **low** 1,
5, 7, 8, 12, 13.
