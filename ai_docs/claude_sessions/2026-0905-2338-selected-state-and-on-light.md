# Session: three words a widget could not say — "tab", "on", and "dark enough to read"

Session: https://claude.ai/code/session_01AM7rDTyaXrFQ4X6rPDdUWp
Date: 2026-09-05 (follows "dated-list-and-tier-c")

## Ask

"Now do the oldest 3 items on the Next list — do phases if necessary."

The three oldest were items 1, 2 and 3 of the list the previous session built:
the missing `tab`/`tablist` roles (age 5, high), ARIA's `log` (age 5, low), and
an "on-light" tone per palette role (age 4, high). Item 1 carried a rider from
the same list — item 8, "a widget cannot say *this control is on*", explicitly
marked "same slot as item 1, they land together or not at all" — so the first
phase closed both.

Three phases, in that order, because each is a vocabulary the next one spends.

## Phase 1 — the vocabulary and the state

### Three roles, and one deliberately absent

`RoleTab`, `RoleTabList`, `RoleLog`. Twenty roles now, still nine of them inert
on both natives.

**The tab pair is the best argument the file has for the vocabulary being
ARIA's.** Compose has `Role.Tab` for the control and nothing for the strip;
SwiftUI has `.isTabBar` for the strip and nothing for the control. Neither
platform's own vocabulary could have supplied the pair. A caller sets both and
each platform takes the half it knows.

**`RoleLog` is not a politeness level.** It and `status` are both polite, and
on Compose they are literally the same call. The difference is the shape of the
content — a status is *replaced*, a log is *appended to* and read back in order
— which is why a chat transcript marked `status` announces correctly and reads
back as one region that has just changed entirely. `examples/chat`'s message
column is the consumer; the item had none when it was raised.

**There is no `RoleTabPanel`, and the absence is load-bearing twice over.**

The design reason: a panel is not really a role, it is one end of a
*relationship*, and the half that carries the announcement is `aria-controls` /
`aria-labelledby` — IDREFs, which a `Style` cannot carry (the same limit that
makes a hint `aria-description` rather than `aria-describedby`). `core.TabView`
already owns both ends on all four targets. Exactly `RoleDialog`'s shape.

The mechanical reason, which the verify-first habit turned up before a line was
written: the WASM runtime's `canBeTabPanel` tells its own wiring apart from an
author's role *by the value* — an element carrying `"tabpanel"` can only have
got it from `wireTabPanel`, because no `core.Role` spells it — and uses that to
avoid unwiring and rewiring a panel on alternate syncs. htmlout's
`TestNoRoleCollidesWithTheTabPanelWiring` already pinned it as a hypothetical;
it is now a decision, and the test comment says so.

### `core.SelectedState` — three values, not a bool

    SelectedUnset  ""       a Box, a heading. No claim. The zero value.
    SelectedOn     "true"
    SelectedOff    "false"  a control that *could* be on and is not

The third is the one a bool loses, and losing it is not cosmetic: a tablist in
which only the live tab carries a state announces the other four as plain tabs,
so the strip reads as one tab and four pieces of furniture. `SelectedOff` is a
value with a job. `core.SelectedWhen(bool)` exists because the tempting
hand-rolled conversion sets `SelectedOn` and leaves the rest silent.

The values are ARIA's own spellings for the same reason `core.Role`'s are — the
two DOM targets write them into the attribute verbatim — and both natives
compare against those literals, so the strings are a four-way contract.
`core/selected_enum_test.go` pins them.

### One field, two attributes — `ariaLevel` in a mirror

The level pair resolves *two Go fields onto one attribute* by switching on the
role. This is the same switch running the other way:

    tab, row, columnheader     ->  aria-selected
    button (or a Button node)  ->  aria-pressed
    anything else              ->  nothing at all

ARIA has two words because it draws a real distinction. Selection is *one of
these* — choosing one unchooses the rest. Pressed is *this one, on or off*. A
filter chip is pressed; a tab is selected; a widget that says the wrong one is
announced as a member of a set that does not exist.

The role list is ARIA's own scoping, and the two near misses are worth naming
because both look like they belong: a `cell` is not a `gridcell`, and a
`listitem` is not an `option`. Neither takes the attribute.

**The `Button` node type is the one arm that is not ARIA's**, and without it the
widget that most wants the attribute would be the single node that could not
have it: `components.Chip` renders as a `core.Button` and sets no role. Both web
exporters already took `nodeType` for the Modal chassis; this is the second
caller.

**Both attributes are written on every runtime call**, not only the one the role
asks for. A patch can turn a tab into a button, and writing one would leave the
other standing — a node announced as a selected tab *and* a pressed button at
once. The `.mjs` suite has that case.

### The natives need no switch, and are not scoped

Compose sets `selected = true/false`; SwiftUI adds `.isSelected`. One spelling
each where ARIA has two attributes, so the switch lives in the two web
exporters and not in core.

Two asymmetries, both documented at the point of temptation rather than
smoothed over:

- **SwiftUI has no word for *off*.** An unselected control carries no trait,
  which is the same view an unstated one produces — so that platform cannot
  distinguish "off" from "not selectable" and does not try. Compose can, which
  is half of why the type has three values.
- **Neither native scopes the state by role.** A state on an unroled Box reaches
  both natives and neither web target. The web is strict because ARIA is: an
  attribute on a generic element is dropped by screen readers, the same failure
  `RoleImg` was added to close. Each platform says the truest thing it can,
  which is the rule the nine unmapped roles already follow.

`mobile/verify/selected_test.go` pins the parse, the mapping, the call, and —
the one that would have shipped broken — that Compose's semantics lambda is
*entered* for a state alone. `grMobSelected` can be correct, called, and never
reached, because a chip with a state and no label, hint, role or disabled flag
would take the `else` branch and get no semantics at all.

## Phase 2 — adoption, and one deliberate refusal

`Chip` and `Calendar` both dropped the `", selected"` suffix they had been
spelling into the accessible *name* for want of a slot. That is the reasoning
`Button.Disabled` already gives for dropping its own `", disabled"`: two
announcements of one fact, and the name is the wrong half to keep — a name is
meant to be stable, so a reader re-announcing the control after a tap read out
the whole altered name rather than the one thing that changed.

The suffix also announced *nothing at all* for a chip with no accessibility
label, which is most chips: the state rode on the label, and the label was only
emitted when non-empty. That half had no test and no caller to notice.

- **`Chip`** states the selection on every chip, selected or not. It needs no
  role: the node type is a `<button>`.
- **`SegmentedControl` and `ChipStrip`** inherit it through Chip.
- **`Calendar`** states it on all forty-two cells, which already carry
  `RoleButton`. ARIA's own date-picker pattern would say this differently — a
  `gridcell` with `aria-selected` — and `core.Role` has no gridcell
  deliberately: the role would oblige the whole grid scaffold and roving focus
  around it, and a lone gridcell inside plain divs is the "table with no rows"
  failure `role.go` calls worse than no role. `Deselectable` now reads better
  than it did: a pressed toggle that un-presses when activated is exactly what
  it is, and without it the cell is a toggle that only turns on.

**The payoff of putting the state on `Style` and the attribute choice in the
exporters:** `SegmentedControl` becomes a tab strip with two props and no new
field. Give the row `RoleTabList` and the segment template `RoleTab`, and the
same `Selected` bool comes out as `aria-selected` instead of `aria-pressed`.
Neither widget knows which arrangement it is in. Pinned end to end, through the
widgets in Go and out through htmlout, because the two halves live in different
packages.

**`ListRow` deliberately kept its suffix**, and the reason is that a row is not
a control. Adopting the field means giving every row a role, and both candidates
are wrong: `RoleButton` is true only of a tappable row and would make it a
*foreign child* of any `role="list"` it sits in — one widget's announcement
costing the enclosing list its shape — while `RoleListItem` is the honest
description and ARIA defines neither state attribute for it. A selectable item
in a collection is an `option` in a `listbox`, and core carries neither, nor the
roving focus a listbox promises. Written into the widget's doc and raised below.

## Phase 3 — the on-light tones

A palette role is one hex, and one hex cannot do both jobs a role is asked to
do. Spent as a **fill** with an ink chosen over it, a mid-tone works —
`Variant.Ink` picks the more legible of the theme's two inks. Spent as **ink
itself**, the backdrop is whatever the widget was placed on, and a mid-tone
loses. Five of the eight bundled role colours failed the 4.5:1 body-text floor
that way, including the *default* case.

Four fields, four resolvers, and `Colors.OnLight(colour)` — a reverse lookup for
a widget that holds a hex and no name for it, which is a commoner position than
it sounds: `Chip`'s accent is read off `Components.Button.Background`
deliberately, so a theme whose buttons are not primary-coloured keeps its own
look.

    DefaultTheme                        MaterialTheme
    primary  #0040DD  4.02 -> 7.56      #6200EE  7.63  (already ink)
    success  #1E7A34  2.22 -> 5.40      #2E7D32  5.13  (already ink)
    warning  #C93400  2.20 -> 5.28      #BF360C  3.08 -> 5.60
    error    #D70015  3.55 -> 5.38      #B00020  7.33  (already ink)

Three of DefaultTheme's four are Apple's own published accessible light-mode
variants, which is the provenance the rest of that palette has. **The green is
not, and that is a finding worth keeping:** Apple's accessible green `#248A3D`
measures 4.40:1 — under the AA floor, for a tone whose whole job is to be read
as body text — so systemGreen is darkened past it instead. It is the one value
without a published source and the one that would otherwise have shipped a
number that looks official and fails.

Material's three passing roles **state their own colour** rather than being left
to the fallback: "this role needs no second tone" is a measurement, and a blank
field cannot be told apart from "nobody has looked".

**The fallback is softer than `Border`/`Success`/`Warning`'s, and deliberately.**
Those degrade to a visible constant because an empty colour is *no* colour; an
absent on-light tone has a perfectly good, merely paler, answer sitting beside
it — the role itself, which is exactly what every widget spent before. So a
theme written before these fields renders as it always did.

**"Light" is the theme's own `Background`**, not a global assumption. A dark
theme's role colours are usually already legible on its dark ground, so it
leaves these empty — which is why these are four extra fields rather than a
second palette every theme fills in twice.

Spent by four widgets, every one of which had documented the gap it could not
close from where it sat:

    Button      outlined + ghost: the label and the rule
    Chip        ProminenceLoud's outline, via the colour lookup
    Banner      the hairline and the leading glyph — the ⚠ was about 2:1
    StatTile    the delta line, which is a sentence someone reads

`Badge`'s background and `Button`'s filled fill still use `Variant.Color`. The
split is now clean: `Color` for fills, `OnLight` for ink.

`StatTile`'s `VariantDefault` stays `TextSecondary` and does not go through the
lookup — that arm is the deliberate *neutral* the widget gives its zero variant,
and `Variant.OnLight` would resolve it to the brand tone the tile is
specifically avoiding.

The contrast test lives in `components`, not `core`: the palette *declares* the
tones, but the WCAG arithmetic that says whether a declaration is any good is
`relativeLuminance`/`contrastRatio`, which live beside `Variant.Ink`. Its
companion checks that the *role* colours still mostly fail — written as a census
rather than four assertions, because if a later retint made every role
ink-weight on its own, eight palette fields would have nothing left to do and
something should say so.

## Verification

All six paths, green:

    go build / go vet / gofmt / go test ./... / go test -race ./...
    wasm/verify/run.sh          replay + 9 mjs suites
    ios/verify/run.sh           data + view + app layers
    android ./gradlew compileDebugKotlin --offline --rerun-tasks
    GOOS=js GOARCH=wasm go build ./...

New tests:

    core/selected_enum_test.go         the census, the zero value, the ARIA
                                       spellings, and SelectedWhen's false half
    core/theme_test.go                 the four fallbacks, the role-resolver
                                       chain, the reverse lookup, and that
                                       Secondary is not one of the toned roles
    components/variant_test.go         every bundled tone clears AA; and the
                                       census that says why they exist
    htmlout/export_test.go             every role x every state against ARIA's
                                       scoping, the Button node type, the zero
                                       value, aria-hidden winning
    wasm/verify/a11y_test.mjs          seven live cases, including the role
                                       change that has to swap the attribute
    wasm/verify/a11y_test.go           the source pins for ariaSelected
    mobile/verify/selected_test.go     both parsers, both mappings, both calls,
                                       Compose's semantics guard, the coverage
                                       census, and SwiftUI's documented gap
    components/calendar_test.go        every cell states a selection; exactly
                                       one says yes
    components/segmented_control_test.go  the tab-strip arrangement, end to end
    examples/tutorial/chapter4_test.go the same, through the live app

Both new coverage checks were confirmed to bite by removing an arm and watching
them fail before restoring it.

## Docs

`docs/concepts/styling-and-theming.md` gained an `AccessibilitySelected`
section and a `#### The on-light tones` section, plus the three new roles in
the vocabulary table. `docs/components.md`: Button's contrast table rewritten
(all eight rows now clear AA), Chip's prominence numbers, the SegmentedControl
tab-strip recipe, Banner's tint paragraph, StatTile's delta, and ListRow's
refusal. Tutorial lesson **4.2** gained the state contract, the two-prop tab
strip, a live demo of it, and two key points. ROADMAP gained two checked items
and the role count moved from seventeen to twenty.

`examples/tutorial/app_test.go` gained `findNodes` and two `nodeStyle` fields —
a tab and a filter chip draw identically, so the style is the only place in the
tree where the difference exists.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here. Value is the payoff of doing it, not the effort: `high` means
something is being worked around today or a second consumer has arrived,
`medium` means it blocks one named thing, `low` means it is a gap nobody has
bumped into yet.

1. **(age 4 · value low) Heading level 6 is reachable and nothing in the
   framework goes past 2.** `AppBar` takes 1 and `GroupedList`'s bands take 2;
   3 through 6 are reachable and unexercised.
2. **(age 3 · value medium) The themes give `Components.Input` no border.**
   Blocks widening `borderResetTags` to `<input>`/`<textarea>`; a palette
   decision, and now a smaller one than it was — the palette has just shown it
   is willing to grow a second value per role.
3. **(age 3 · value low) `AccessibilityNestingLevel` has no consumer.** Watch
   whether the first nested list downstream reaches for it.
4. **(age 3 · value low) A `<select>` will need the border decision** if a
   picker node type lands. Contingent on a node type that does not exist.
5. **(age 2 · value medium) `contrastInk` picks black on `#007AFF`.** A
   selected calendar cell's numeral and dots come out `#000000` on the
   DefaultTheme blue. *Not* fixed by this session: `contrastInk` chooses
   between the theme's two ink roles for a colour used as a fill, which is the
   other half of the pair the on-light tones complete. The fix is probably a
   third candidate rather than a new field.
6. **(age 1 · value medium) `core` has no z-stacking primitive.** `Compass`
   wanted a needle over its rose and could not have one. A `core.Stack` node
   type would be a Compose `Box` / SwiftUI `ZStack` / `position: relative` plus
   absolute children.
7. **(age 1 · value medium) The `permission` package is dead code.**
   Commented-out Portuguese calling an `InvokeNative` that does not exist. Tier
   D's location work walks straight into it.
8. **(age 1 · value low) Three app-layer Swift files are still checked by
   nothing.** `GomobileBridge`, `GrMobApp` and `AppLifecycle` import the
   generated `Mobile.xcframework`.
9. **(age 0 · value high) `ListRow` cannot announce its selection**, and the
   vocabulary is what is missing rather than the plumbing. Its state wants
   `option` inside `listbox` — ARIA's pair for a selectable item in a
   collection, which also promises roving focus. `RoleButton` would make the
   row a foreign child of any `role="list"` around it. The first real consumer
   for a listbox pair, and the reason this session's field stopped one widget
   short.
10. **(age 0 · value low) A hand-built tab strip cannot point at its panel.**
    `aria-controls` and `aria-labelledby` are IDREFs and `Style` carries values.
    `core.TabView` covers the wired case, so this is only a gap for a strip
    built out of chips — no consumer yet, and the fix would be a node-path-
    derived id scheme both DOM targets agree on, which is what TabView already
    has internally.
11. **(age 0 · value low) `RoleLog` has one consumer and no test of its own.**
    `examples/chat` has no test file at all, so the adoption is checked only by
    the vocabulary-wide export test. Cheap to close if the chat example ever
    grows a suite.

Read by value instead: **high** 9 · **medium** 2, 5, 6, 7 · **low** 1, 3, 4, 8,
10, 11.
