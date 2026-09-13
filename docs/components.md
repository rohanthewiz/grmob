# Widget Library — the `comps` package

`comps` is GrMob's higher-level widget library, built **entirely on the
public core API** — a deliberate dogfooding discipline: if a widget can't be
built out here, that is a gap in core's primitives, not a reason to reach
inside.

```go
import "github.com/rohanthewiz/grmob/comps"
```

## The struct-widget idiom

Every widget is a struct implementing `core.View`, configured through named
fields:

```go
comps.Card{
    Title:  "Account",
    Body:   balanceSummary,
    Footer: comps.Badge{Text: "verified"},
}
```

Structs — rather than more constructor functions in core — for two reasons:

- **Named fields scale.** A widget can grow an optional knob without
  breaking a single call site, where positional arguments cannot.
- **A `core.View`-typed field is a composition slot.** `Card.Header`,
  `FormField.Input`, `Accordion.Content` accept any view. Where a widget
  offers a simple path *and* a slot (`Card.Title` vs `Card.Header`), the
  slot wins when both are set.

All widgets take their look from `ctx.Theme()` — palette colors, spacing and
typography scales, never hard-coded values — and accept `Style` overrides
for per-use adjustment.

## Screen

The root scaffold: safe-area inset, an optional scroll region, and the
vertical column that holds the screen's content.

```
SafeArea
  ├─ Scroll            (only when Scroll is true; KeyboardAware lands here)
  │    └─ Column       ← Gap / Fill / Style land here
  │                      (and KeyboardAware, when there is no Scroll)
  │         ├─ Children[0]
  │         └─ …
  └─ Footer            (only when Footer is set; pinned, never scrolls)
```

```go
// The zero value: SafeArea(Column(children...)), nothing else.
comps.Screen{
    Children: []core.View{header, body, composer},
}

// A screen that scrolls as a whole.
comps.Screen{
    Scroll:   true,
    Children: []core.View{hero, section1, section2},
}

// A screen whose list fills the space and pushes a footer down.
comps.Screen{
    Fill: true,
    Gap:  12,
    Children: []core.View{title, entryRow, list, footer},
}
```

| field | effect |
|---|---|
| `Children` | laid out top to bottom in the column; a `nil` entry is skipped |
| `Scroll` | wraps the column in `core.Scroll` |
| `KeyboardAware` | the region yields to the software keyboard instead of being covered |
| `Gap` | uniform vertical spacing, in points; zero means *unset* |
| `Fill` | `FlexGrow(1)` on the column — claim the full safe-area height |
| `Style` | applied to the column **after** `Gap` and `Fill`, so it overrides both |
| `Footer` | pinned below the content, outside the scroll region; the content grows to push it to the bottom edge |

**Every field defaults to contributing nothing**, so the zero value renders
the bare scaffold with no style props at all and the theme's `Column` base
carries through untouched. That is what let all five migrations stay
byte-identical. It matters specifically for `Gap`: style props *set* rather
than merge, so an unconditional `core.Gap(0)` would overwrite a gap the
theme's `Column` base had set. Unset and zero therefore have to mean the same
thing — the absence of a gap, not the imposition of one.

**`Scroll` is for screens with no scrolling region inside them.** Leave it
false when the screen already contains a `core.List` or its own `Scroll` — a
scroll view nested in a scroll view fights for the same drag on both natives.
Of the nine app packages in `examples/`, two scroll at the root
(`fintechapp`, `signup`), and the `tutorial`'s `lesson_screen` and chapter 6
scroll as a whole. The rest do not: `chat` scrolls its message list, `todoapp`
scrolls a virtualized `core.List`, the `tutorial`'s contents screen is one,
and `mobileapp` and `layout` are short enough to need neither.

**A scrolling child is the page, and is inset once.** `Screen`'s column and
`core.List` are both built on the theme's `Components.Column`, whose only entry
in every bundled theme is the standard 12/16 inset. So a screen whose whole
content is a `List` used to be inset *twice* — and invisibly, because neither
inset is written anywhere:

```
SafeArea
  └─ Column   padding 12/16   ← the theme's, via Screen
       └─ List padding 12/16   ← the theme's, again
```

Every row drew 16 points further in than the same content in a scrolled
`Column`, which is the shape this page recommends moving *away* from. So the
scaffold drops its own inset when its content is a single scrolling page:

```go
// Inset once, by the List. No Padding(0) needed.
comps.Screen{Fill: true, Children: []core.View{core.List(rows...)}}
```

Three things about the rule:

- **"Only child" is counted after `nil` entries are skipped**, so the
  conditional-slot idiom below still qualifies — an absent banner beside a list
  is a single-child screen, exactly as the tree the reconciler walks is. The
  decision is taken on the *rendered* child, so `comps.GroupedList` counts
  too: it is a `List` once it renders.
- **The set is node types that scroll and arrive pre-inset**, which today is
  `core.List` alone. `core.Scroll` is deliberately outside it: it carries no
  theme base, so its content is inset once — by the column — and dropping that
  would move the page rather than unstack it. A `List` with siblings is a block
  within the page rather than the page, and the column keeps its inset.
- **`Style` still wins.** The cleared padding is applied ahead of the caller's
  `Style`, so `core.Padding(24)` around a list gives 24. Nothing but the inset
  is dropped — `Gap`, `Fill` and a background all survive.

**`KeyboardAware` lands on the scroll region, or on the column when there is
none** — the two halves of what it means. A scrolling screen wants its
*viewport* shortened, so the platform's scroll-the-focused-field-into-view has
somewhere visible to put the field; a fixed screen wants the whole column
lifted, so whatever is docked at its bottom — a chat composer, a checkout bar,
the thing the keyboard actually covers — stays reachable. Either way the
`SafeArea` itself stays put, so a header does not slide off the top. See
[`core.KeyboardAware`](concepts/views.md#the-software-keyboard) for what each
platform does with it.

```go
// A form: the viewport ends where the keyboard begins.
comps.Screen{Scroll: true, KeyboardAware: true, Children: fields}

// A chat: no scroll here, so the column lifts and the composer rides up.
comps.Screen{KeyboardAware: true, Children: []core.View{header, thread, composer}}
```

**`Footer` is the slot for a bar that must not scroll.** A `comps.BottomBar`,
a checkout bar or a chat composer placed among `Children` scrolls away with
the content. In `Footer` it becomes the `SafeArea`'s second child, and the
content region takes the leftover height so the footer sits on the bottom edge
even under short content. With `Scroll` the `Scroll` grows and the column
inside it does not; without `Scroll` the column grows, as if `Fill` were set.

```go
comps.Screen{
    Scroll:   true,
    Children: []core.View{feed},
    Footer:   comps.BottomBar{Items: tabs, Selected: tab.Get()},
}
```

The footer takes no inset or style from the scaffold. It is the caller's
widget and draws its own background and padding. A nil `Footer` leaves the
tree exactly as it was.

**`Fill` is load-bearing wherever a child grows.** A `FlexGrow` child can only
grow inside a parent that has height to give, so a screen whose list should
expand into the leftover space needs `Fill: true` on the scaffold —
`examples/todoapp` is the worked example. `Fill` with `Scroll` is legal but
unusual: it makes the scrolled content at least as tall as the viewport, which
is how you bottom-anchor a footer on a short page. It does not make a scroll
view fill anything.

Because `Children` flows into `core.Column`'s argument list, which skips nil
items, the conditional-region idiom costs the tree nothing when the condition
is false:

```go
var banner core.View
if offline {
    banner = OfflineBanner()
}
comps.Screen{Children: []core.View{banner, body}}
```

That is the same contract behind [`core.MaybeProp`](concepts/views.md) — no
node, no flex slot, no stray `Gap` — where a `core.If` would leave an empty
`Fragment` for the column to space against.

## Button

A themed action button with **two orthogonal color axes** and no per-call hex.

```go
comps.Button{Label: "Save",   OnTap: save}                                   // theme Button base
comps.Button{Label: "Delete", OnTap: rm,   Variant: comps.VariantError}
comps.Button{Label: "Cancel", OnTap: back, Emphasis: comps.EmphasisOutlined}
comps.Button{Label: "Skip",   OnTap: skip, Emphasis: comps.EmphasisGhost}
```

`Variant` says **which** color (the meaning) and is the same enum
[Badge](#variants) uses, so a danger button and an error badge are the same red
by construction. `Emphasis` says **how much** of it:

| emphasis | fill | label | rule |
|---|---|---|---|
| `EmphasisFilled` (zero value) | the variant's color | contrast-picked against the fill | — |
| `EmphasisOutlined` | transparent | the variant's color | 1px, the variant's color |
| `EmphasisGhost` | transparent | the variant's color | — |

!!! note "Why two fields instead of one `Primary | Secondary | Danger | Ghost`"
    A flat enum conflates the two questions, so it cannot express an outlined
    destructive button — the ordinary shape of a "Delete" confirmation —
    without a fifth value, and then a ghost destructive needs a sixth.

The rule in the last column is the only thing separating outlined from ghost,
and for a while it was the one property no target agreed on. Neither native
drew the outlined rule — a Button draws its own container, so each renderer
hands it a style with the box-drawing fields stripped, and the border was the
one of them never fed back through the platform control's own slot. Both web
targets drew a ghost rule — a `<button>` carries the user agent's border, and
emitting no declaration is what leaves it in charge, so `core.BorderWidth(0)`
could not remove it. Both halves are fixed: the same guard (a width **and** a
color) now decides the border on all four targets.

**The zero value applies nothing.** `Button{Label: l, OnTap: f}` renders
exactly `core.Button(l, f)` — the theme's own `Components.Button` carries the
look through untouched, rather than being re-derived from the palette. A theme
whose Button base deliberately differs from `Colors.Primary` keeps that choice.

The flip side: a theme with *no* `Components.Button` gets a button with no
styling. That is not new — `core.Button` has always behaved that way — but the
widget makes it visible, which is how `examples/fintechapp` turned out to have
no `Components` block at all.

### Secondary is deliberately not a variant

`Variant` carries *meaning*; `Secondary` is a brand slot a theme may set to
anything. `Style` is the escape hatch for the brand case:

```go
comps.Button{
    Label: "Recharge", Emphasis: comps.EmphasisOutlined,
    Style: []core.StyleProp{
        core.TextColor(t.Colors.Secondary),
        core.BorderColor(t.Colors.Secondary),
    },
}
```

### Contrast, and what the widget can promise

`EmphasisFilled` owns both the fill and the label, so it picks the label by
contrast and is tested to clear WCAG AA under both bundled themes.

Outlined and Ghost own neither — the fill is transparent, so the label's real
backdrop is whatever you placed the button on. Their label and rule are
therefore the role's
[**on-light tone**](concepts/styling-and-theming.md#the-on-light-tones), the
palette's second value per role, rather than the fill color. Measured against
each theme's own `Background` (both `#FFFFFF`), with the value each replaced:

| variant | DefaultTheme | MaterialTheme | AmberTheme |
|---|---|---|---|
| default | **7.56:1** (needs no second tone) | 7.63:1 (needed no second tone) | **5.78:1** (was 2.04) |
| success | **5.40:1** (was 2.22) | 5.13:1 (needed no second tone) | 5.13:1 (needed no second tone) |
| warning | **5.28:1** (was 2.20) | **5.60:1** (was 3.08) | **5.60:1** (was 3.08) |
| error | **5.38:1** (was 3.55) | 7.33:1 (needed no second tone) | 7.33:1 (needed no second tone) |

All twelve clear WCAG AA (4.5:1); five of them did not before their tone
existed. `AmberTheme`'s `default` row is the widest gap in the table and the
reason that palette was written: MD amber 700 is a fine fill and cannot be ink,
which is the case neither of the other two still makes for `Primary`.

`DefaultTheme`'s `default` row read "was 4.02" until its `Primary` role was
[darkened to the accessible blue its tone already carried](concepts/styling-and-theming.md#the-on-light-tones),
which the *filled* treatment needed and this one did not. The number here is
unchanged by that; the role and its tone simply became one colour.

The promise is still narrower than `EmphasisFilled`'s: these numbers hold
against a theme's `Background`, and a button placed on some other surface — a
tinted card, a photo — is measured against that instead, which nothing here can
know. What changed is that the *default* case is legible rather than documented
as illegible. A theme that declares no on-light tones falls back to the role
color, i.e. to the numbers in brackets, and to exactly the pixels this widget
painted before.

Darkening the role color *here* was rejected then and still is — it would
repaint DefaultTheme's own brand blue. Declaring the second value is the
theme's call; spending it is the widget's.

### FullWidth and Disabled

`FullWidth` sets both a `100%` width and a **block** display: the bundled
themes give Button an inline display, and width does nothing to an inline box.

`Disabled` does three separate things, and all three are needed. It renders
the palette's muted pair (Surface fill, TextSecondary ink); it replaces the
handler with a no-op (dropping it would leave a nil func in the callback
registry for a tap that races the disabling patch to panic on); and it sets
[`core.Disabled`](concepts/styling-and-theming.md#disabled), which is what
makes the *platform* refuse to dispatch and announce the state to a screen
reader.

That last part is why the accessibility label is left alone. The widget used
to append `", disabled"` to it — the `", selected"` convention `ListRow` still
uses — because no renderer carried a disabled state. Now that they all do,
appending it as well would announce the state twice. `Chip` and `Calendar`
followed the same path once
[`core.AccessibilitySelected`](concepts/styling-and-theming.md#accessibilityselected)
existed; `ListRow` did not, and its own entry says why.

## InputRow

The composer: a text field that fills the row, and an optional trailing button
that commits it.

```
Row (Gap)
  ├─ Input   ← FlexGrow(1), value / placeholder / onChange / onSubmit
  └─ Button  ← only when Button.Label is set; OnTap defaults to OnSubmit
```

```go
// A field and a Send button, one commit action.
comps.InputRow{
    Value:       draft.Get(),
    Placeholder: "What needs doing?",
    OnChange:    func(v string) { draft.Set(v) },
    OnSubmit:    addTodo,
    Button:      comps.Button{Label: "Add"},
}

// A search field with no button: the return key commits it.
comps.InputRow{
    Value:       query.Get(),
    Placeholder: "Search",
    OnChange:    func(v string) { query.Set(v) },
    OnSubmit:    runSearch,
}

// A docked composer, with the bar treatment this widget has no opinion about.
comps.InputRow{
    Value: draft.Get(), Placeholder: "Mensagem…",
    OnChange: func(v string) { draft.Set(v) },
    OnSubmit: send,
    Button:   comps.Button{Label: "Enviar"},
    Style: []core.StyleProp{
        core.BackgroundColor("#FFFFFF"),
        core.Padding(12),
    },
}
```

| field | effect |
|---|---|
| `Value` | the field's text — fully controlled, so `OnChange` is the only way it changes |
| `Placeholder` | empty-state text; the field's only label |
| `OnChange` | fires on every keystroke |
| `OnSubmit` | the commit action: keyboard return/done **and** the button's tap |
| `Button` | trailing commit button; the zero value renders **no node at all** |
| `Gap` | horizontal spacing, in points; zero means the theme's `SM` step |
| `Style` | applied to the row **after** `Gap`, so it overrides it |

**One commit action, three paths.** The button *is* the field's submit,
rendered as a tap target for the case where the keyboard's return key is not
obvious or not reachable — so `OnSubmit` drives both and the two cannot drift
apart. Both hand-written call sites this replaced named the same helper twice.
Setting `Button.OnTap` explicitly still wins, but it has to be said out loud.

**`Gap` defaults to the theme's step, unlike `Screen`'s.** `Screen` treats a
zero `Gap` as *unset*, because the spacing between a screen's sections is the
app's decision. The gap here is the opposite kind of thing — it is the widget's
internal layout, and the field and the button must not touch — so `InputRow`
owns it the way `FormField` owns the spacing between its label and its input.
Zero means the theme's `SM` step (8pt in both bundled themes). To ask for no
gap at all, say it through `Style`:

```go
Style: []core.StyleProp{core.Gap(0)}
```

**A nil `OnSubmit` builds the field without a submit path**, rather than with
one wired to a no-op: the renderers read the `onSubmit` prop to decide whether
to show a submit affordance on the keyboard, and a registered no-op would
advertise an action the row ignores.

**The input is owned, not slotted.** Unlike [`FormField`](#formfield), which
takes whatever input it is given, `InputRow` builds its own — the wiring *is*
the widget, and a slot would hand it back to the caller. So the field itself
takes no per-call styling; a composer that needs to restyle its input has
outgrown this and should go back to `core.Row` + `core.InputWithSubmit`. Wrap
the row in a `FormField` when it needs a caption or an error line.

## Card

A surface with optional header, body, and footer regions, on the theme's
Card base (background, padding, radius, shadow).

```go
comps.Card{
    Title: "Recent activity",        // simple path: themed bold subtitle
    Body:  activityList,
    Footer: core.Text("Updated 2m ago"),
}

comps.Card{
    Header: customHeaderRow,          // escape hatch — overrides Title
    Body:   content,
    Style:  []core.StyleProp{core.Margin(0)},
}
```

`Title` is a `RoleHeading` at **level 2** — a card is a section of a screen,
the same tier as a `GroupedList` band and one below an `AppBar`'s title.
`HeadingLevel` moves it when the card is not where the default assumes; a card
inside a section says 3. It applies to `Title` alone: a `Header` replaces the
line entirely and the view in it is yours to describe, since the widget cannot
know whether it was handed a heading, a row of controls or an avatar. See
[the outline](concepts/styling-and-theming.md#the-packages-heading-outline).

## ListRow

The leading-control / flexible-title / trailing-action shape every list
needs: a checkbox and a task with a delete button, an avatar and a name with
a chevron, a label and an amount.

```go
comps.ListRow{
    Leading:  core.Checkbox(t.Done, func(v bool) { setDone(id, v) }),
    Title:    t.Title,
    Subtitle: "Due today",
    Trailing: core.Button("✕", func() { remove(id) }),
    OnTap:    func() { open(id) },

    Selected:           id == selectedID,
    AccessibilityLabel: t.Title,
    AccessibilityHint:  "Opens the task",
}
```

**How the trailing slot gets pinned.** The examples that hand-rolled this
shape disagreed: some used `Justify(JustifyBetween)` on the row, others
`FlexGrow(1)` on the middle text. They are not equivalent —
`JustifyBetween` distributes slack between *every* pair of children, so a
row with no trailing slot pushes leading and title apart. ListRow settles it
on `FlexGrow`: a middle column takes all the slack, so leading and trailing
sit hard against the row's edges in every configuration.

```
┌ Row ─────────────────────────────────────────────────────────┐
│ [Leading]  ┌ Column FlexGrow(1) ────────────┐    [Trailing]  │
│            │ Title                          │                │
│            │ Subtitle                       │                │
│            └────────────────────────────────┘                │
└──────────────────────────────────────────────────────────────┘
            └──────── takes all the slack ────┘
```

The middle column is rendered **even when empty** — unlike `Card`, which
omits empty regions. Here the middle is structure, not content: making it
conditional would make the pinning conditional too.

Other notes:

- `Content` is the escape hatch for the middle, overriding `Title`/
  `Subtitle` — the same simple-path-plus-slot idiom as `Card.Header`.
- `OnTap` / `OnLongPress` make the whole row a target and are wired only
  when non-nil, so a presentational row registers no callback at all. Both
  may be set at once; the renderers wire them as one gesture recognizer, so
  a long press never also fires the tap.
- `Selected` tints the row with the theme's `Surface` (override with
  `SelectedStyle`). How the state is *announced* depends on `Selectable`.
- `Selectable` makes the row one choice in a listbox: it takes `RoleOption` and
  states
  [`core.AccessibilitySelected`](concepts/styling-and-theming.md#accessibilityselected)
  for **both** values of `Selected`, so a reader says "selected" on the chosen
  row and "not selected" on the rest rather than passing over them silently.

    ```go
    core.List(
        core.AccessibilityRole(core.RoleListBox),       // the caller's half
        comps.ListRow{Title: "Weekly",  Selectable: true, Selected: plan == weekly},
        comps.ListRow{Title: "Monthly", Selectable: true, Selected: plan == monthly},
    )
    ```

    Opt-in for the same reason `NestingLevel` is: an `option` is owned by a
    `listbox`, a row cannot see its own container, and an orphan `option` names
    a structure that is not there.

    Without it the row falls back to appending `", selected"` to the
    accessibility label — which is what this widget did for three versions,
    because until `core.RoleOption` existed no role it could take would carry
    the state. That suffix now reaches a browser at all: an unroled row is
    given `role="group"` by both web exporters, which is what makes the name it
    rides on audible — see
    [`RoleGroup`](concepts/styling-and-theming.md#rolegroup-and-the-one-role-you-get-without-asking). `RoleButton` is true only of a tappable row and would make it a
    *foreign child* of any `role="list"` it sits in; `RoleListItem` is the
    honest description of a row and ARIA defines no selection state for it. The
    suffix says the true thing in the weaker place: once, inside a name that is
    meant to be stable.

    The two web targets write `role="option"` and `aria-selected`. Neither
    native names a listbox or an option, but both announce the *state* on any
    node, so the row still reads as chosen on device.

    The container's role also buys the **keyboard**: under the WASM runtime a
    `RoleListBox` becomes one stop in the page's tab order, Up and Down move
    between its rows, `Home` and `End` reach the ends, and `Enter` or `Space`
    runs the focused row's `OnTap`. That arrives with the container's role and
    not with this flag — a `Selectable` row in an unroled Box is an `option`
    with nothing to be an option of. A static `htmlout` export writes no tab
    stops, deliberately; see
    [WASM — Composite widgets are operable](platforms/wasm.md#composite-widgets-are-operable).
- `NestingLevel` is the same table's other answer, and it lands. A depth's
  role is `listitem` — one of the three ARIA defines `aria-level` for, and the
  honest description of a row — so the row states its depth and becomes the
  first consumer
  [`core.AccessibilityNestingLevel`](concepts/styling-and-theming.md#accessibilitynestinglevel)
  has ever had. It is what a flattened outline cannot say any other way: a list
  is a flat run of siblings, so an indent is pixels a screen reader never sees.

    ```go
    core.List(
        core.AccessibilityRole(core.RoleList),          // the caller's half
        comps.ListRow{Title: "Gospels", NestingLevel: 2},
        comps.ListRow{Title: "Matthew", NestingLevel: 3, Style: indent},
    )
    ```

    Opt-in, because a `listitem` is owned by a `list` and a row cannot see its
    own container: role the list yourself, or leave the field at zero and the
    row is the unroled box it has always been. The cost is that a row inside a
    `role="list"` must not also be a `role="button"`, so a tappable row in an
    outline announces as an item at a depth rather than as a control — the same
    foreign-child rule as above, from the other side.

    `NestingLevel` and `Selectable` ask for different roles and a node has one,
    so a row is a depth *or* a choice. `Selectable` wins when both are set: an
    `option` carries `aria-selected` and no `aria-level`, a `listitem` the
    reverse, and the state is what the tap changes. ARIA's role for an item
    that is both is `treeitem` inside a `tree`, which `core.Role` does not
    carry.
- No label is synthesized from `Title`. A row is a compound control whose
  slots carry meaning the widget cannot see, and labelling the container
  overrides how its children are announced, so naming the row is the
  caller's call.

All five hand-rolled instances in the examples are now built on it:
`todoapp`'s task row and footer, `mobileapp`'s feed row and subscribe
toggle, and `fintechapp`'s transaction row. Two of them lost workarounds in
the process — `mobileapp`'s feed row no longer appends to a
`[]core.PropsAndChildren` to add a conditional style prop (`SelectedStyle`
is that conditional, declared), and `todoapp`'s footer no longer wraps its
bulk-clear button in `core.If` (a nil `Trailing` emits no node at all).

## SwitchRow & CheckboxRow

The settings-screen row: a title, an optional subtitle, the control on the
trailing edge, and the **whole row** tappable rather than only the control.

```go
comps.SwitchRow{Title: "Notifications", Subtitle: "Push and email",
    On: notify.Get(), OnToggle: notify.Set}

comps.CheckboxRow{Title: "I agree to the terms",
    Checked: agreed.Get(), OnToggle: agreed.Set}
```

Both are a `ListRow` with the trailing slot fixed to `core.Switch` or
`core.Checkbox` and `OnTap` wired to the same setter. Pick the switch for a
setting that takes effect on the tap and the checkbox for a value a form
collects later; see `core.Switch` for why those are different controls.

**One tap, one change.** The row and the control are both tappable, and the
targets disagree about what a tap on the control reaches:

| Target | Tap on the control | Handlers that fire |
|---|---|---|
| Compose | the Switch consumes the press | control only |
| SwiftUI | the Toggle's gesture wins | control only |
| Web | native toggle, then the click bubbles to the row | row, then control |

Go cannot tell where a web click landed, so the widget makes the double
dispatch harmless instead: both handlers **set** a value through one guard
that drops a report of the value already rendered. The row sets `!On`; the
control sets whatever the platform reports. After the row's dispatch the app
re-renders, and the control's `change` then reports the value that is already
current, so it is dropped.

That is why `OnToggle` is a setter. Apply the value it hands you; do not
invert your own state inside it.

Rendering the control `Disabled` so only the row handles taps was rejected.
`Disabled` is the platform's disabled state, so every target would draw the
switch greyed and every screen reader would call it dimmed.

Other notes:

- The control is named by the row: `Title` becomes its accessibility label
  and `Subtitle` its hint, so a reader hears "Notifications, switch, on".
  The row itself takes no role.
- `Disabled` disables both the row and the control. The row's handler stays
  registered, as `core.Style.Disabled` requires.
- `Leading` takes an icon or avatar, and `Style` reaches the underlying
  `ListRow`.

## Badge

A small **non-interactive** status pill — a count, a "verified" mark, a
state label. Defaults: theme Primary background, theme Background ink,
caption-sized, stadium-shaped.

```go
comps.Badge{Text: "3"}
comps.Badge{Text: "beta", Color: "#7B1FA2"}
```

### Variants

`Variant` names what the badge *means* and takes its colors from the
palette's status roles, so a status pill carries no literal hex:

```go
comps.Badge{Text: "Paid",     Variant: comps.VariantSuccess}
comps.Badge{Text: "Expiring", Variant: comps.VariantWarning}
comps.Badge{Text: "Failed",   Variant: comps.VariantError}
```

| variant | fill |
|---|---|
| `VariantDefault` (zero value) | `Primary` |
| `VariantSuccess` | `Success` |
| `VariantWarning` | `Warning` |
| `VariantError` | `Error` |

The zero value is the look every badge had before the field existed, so
adding it restyles nothing.

**The ink is computed, not fixed.** The palette pairs no ink with a status
role, and the right answer flips between themes — `DefaultTheme`'s Success is
a *light* green wanting dark ink, `MaterialTheme`'s is a dark one wanting
light ink. So for a status variant the label takes whichever of the theme's
two ink roles has more contrast against the fill. Reusing the default
variant's white ink would render DefaultTheme's Success and Warning badges at
~2.2:1 — unreadable. `VariantDefault` is exempt and keeps the theme's
Primary/Background pairing, which is what makes the zero value a no-op.

Explicit `Color` beats `Variant`, and the ink is still resolved against
whichever fill won, so an override cannot silently produce an illegible pill.
Explicit `TextColor` beats the computed ink.

**A variant reinforces the text, it does not replace it.** Nothing announces
"warning" to a screen reader, and a reader who cannot tell the tints apart
sees only the label — so the label has to say it ("Overdue", not "!").

`Variant` is a package-level type, not Badge's own, so a future Alert or
banner resolves the same four roles the same way. `Variant.Color(theme)` and
`Variant.Ink(theme, bg)` are exported for building your own status surface.

For a *selectable* pill, use Chip.

## Chip

A selectable pill — filter toggles, tag pickers. **Controlled**: the chip
holds no state; it renders `Selected` and reports taps through `OnTap`, so a
chip group is one piece of parent state plus a loop.

```go
comps.Chip{
    Label:    label,
    Selected: i == active,
    OnTap:    func() { onSelect(i) },
    AccessibilityLabel: "Show " + strings.ToLower(label) + " tasks",
    AccessibilityHint:  "Filters the task list",
}
```

- Selected is the loud state: the theme's Button base, untouched (plus a ring
  painted in the fill, so both states hold the same box). Unselected is the
  quiet one: Surface fill, `TextPrimary` ink, a 1px ring in
  `Colors.ControlBorder`.
- `Prominence` tunes *how* quiet the unselected state is — see below.
- `SelectedStyle` and `UnselectedStyle` replace their state's default. Both
  read `nil` as "use the default" and an allocated-but-empty slice as "apply
  nothing", which is how you drop a default instead of overriding it.
- `Style` applies to both states and the state wins where they collide —
  otherwise one `Style` shared across a strip would flatten the distinction
  the strip is drawing.
- Every chip states
  [`core.AccessibilitySelected`](concepts/styling-and-theming.md#accessibilityselected),
  selected or not, so a reader announces "pressed" / "not pressed" alongside
  the name. It used to be a `", selected"` suffix on the accessibility label,
  which announced nothing at all for a chip that had no label — most of them —
  and which changed the control's *name* on every tap.

The two state defaults used to be the other way round — the selected chip was
the quiet one — which read as an inverted filter row and is the one thing that
has changed here. `SelectedStyle: {Surface fill, Primary ink}` plus
`UnselectedStyle: []core.StyleProp{}` restores it.

### Prominence: quiet or loud

Which state is louder is settled — the selected one — and that is not what
this field touches. *How much* quieter the other one is has two right answers:

| | |
|---|---|
| `ProminenceQuiet` (zero) | Surface fill, `TextPrimary` ink, a ring in `Colors.ControlBorder`. Right for a **filter** row, which is chrome above the content it filters: a loud row of years competes with the archive it is filtering. |
| `ProminenceLoud` | The chip's accent, in its on-light tone, as ink and as a 1px rule over a transparent fill — the outlined treatment. Right for a row of **suggestions** the reader is meant to reach into: grey pills over an empty amount field do not read as "tap one of these". |

```go
comps.Chip{Label: "$25", Prominence: comps.ProminenceLoud,
    Selected: cents == 2500, OnTap: func() { set(2500) }}
```

Material draws the same distinction (filter chip vs. suggestion chip) with a
different default prominence for each.

**Neither answer is "invisible".** Both treatments draw their ring at
control-boundary weight, because WCAG 1.4.11 puts a 3:1 floor under the edge
that identifies a control and a chip's fill clears it in neither state
(`Surface` is 1.12:1 against the page under `DefaultTheme`). The quiet ring
used to be the palette's `Border` hairline — 1.26:1 — which made a filter row
that receded out of sight rather than into the background; it is
[`Colors.ControlBorder`](concepts/styling-and-theming.md#color-roles) now, and
that role exists because of this chip. Quiet is about the fill and the ink.

Loud is **not** the pre-inversion look. That one gave every unselected chip a
solid fill and left the chosen one pale; here the fill is transparent, so the
selected chip is still the only solid pill in the row.

The accent is the theme's own `Components.Button` background — the fill the
selected chip paints — so the outline and what it becomes when tapped are the
same hue on any theme. A theme with no Button fill falls back to
`Colors.Primary`. Whichever it lands on is then resolved through
[`Colors.OnLight`](concepts/styling-and-theming.md#the-on-light-tones) — a
lookup by *colour* rather than by role, because the accent is a hex the widget
read off the Button base and has no name for — so the outline is drawn at ink
weight. The numbers are the outlined [`Button`](#button)'s `default` row, since
it is the same colour on the same backdrop: **7.56:1** under `DefaultTheme` and
**7.63:1** under `MaterialTheme`. Neither bundled role needs a second tone
today, so under both of them this lookup is currently an identity; a theme that
declares none falls back to the accent itself, which is what this painted
before, and a theme whose brand colour is a mid-tone is where the lookup still
moves the pixels.

The outline and the fill it becomes when tapped are now two weights of one hue
rather than the same value — still the same hue by construction, which is what
kept the two from drifting apart on a theme whose buttons are not
primary-coloured.

`UnselectedStyle` wins where both are set — it replaces the treatment,
`Prominence` picks between them. And because `SegmentedControl.Segment` is a
whole `Chip`, `Segment: comps.Chip{Prominence: comps.ProminenceLoud}`
carries it to every segment.

The todoapp filter bar is built on Chip; `examples/todoapp/chip_migration_test.go`
pins its rendered HTML against the same bar written out by hand — the pattern
to copy when extracting your own widgets.

## SegmentedControl

A controlled single-select rendered as a row of chips — a filter bar, a mode
switcher, a scope picker.

```
Row (Gap)
  ├─ Chip "All"     ← Selected == 0
  ├─ Chip "Active"
  └─ Chip "Done"
```

```go
comps.SegmentedControl{
    Labels:   []string{"All", "Active", "Done"},
    Selected: filter.Get(),
    OnSelect: func(i int) { filter.Set(i) },
}
```

| field | effect |
|---|---|
| `Labels` | segment captions, left to right; `Selected` indexes this slice |
| `Selected` | index of the active segment; out of range selects **nothing** |
| `OnSelect` | fires with the tapped segment's index |
| `Segment` | the `Chip` template every segment is rendered from |
| `SegmentLabel` | derives a segment's accessibility name from its caption |
| `KeyPrefix` | prepended to each segment's reconciler key (default: the caption) |
| `Gap` | spacing between segments; zero means the theme's `SM` step |
| `Style` | applied to the row **after** `Gap`, so it overrides it |

**Selection is an index and the caller owns it.** The control holds no state:
it renders `Selected` and reports taps — the same contract [`Chip`](#chip) has,
one level up. That is what lets the selected index *be* the app's own filter
enum; `examples/todoapp` declares `filterAll`/`filterActive`/`filterDone` as
indices into its label slice, so there is no mapping in between.

An out-of-range `Selected` selects nothing. That is a legal state, not a
defensive check: a scope picker that starts with no scope chosen says so with
`-1` rather than by growing a fourth "none" segment.

**`Segment` is a template, not a set of pass-through fields.** Everything a
`Chip` can do — `Style`, `SelectedStyle`, `AccessibilityHint` — is set once and
applies to all of them:

```go
comps.SegmentedControl{
    Labels:    filterLabels,
    Selected:  active,
    OnSelect:  onSelect,
    KeyPrefix: "filter-",
    Segment: comps.Chip{
        Style:             []core.StyleProp{core.FontSize(13)},
        SelectedStyle:     []core.StyleProp{core.BackgroundColor(colorAccent)},
        AccessibilityHint: "Filters the task list",
    },
    SegmentLabel: func(label string, _ int) string {
        return "Show " + strings.ToLower(label) + " tasks"
    },
}
```

`Label`, `Selected` and `OnTap` on the template are ignored — those three are
exactly what the control computes. The alternative was re-exporting Chip's
surface as `SegmentStyle`, `SelectedSegmentStyle`, `SegmentHint` and so on,
which grows a field every time `Chip` does. It is the same move
[`InputRow`](#inputrow) makes with `Button`.

**`SegmentLabel` is a function because the name is the one thing that varies
per segment and is not derivable from the caption** — todoapp announces "Show
active tasks" for a chip captioned "Active". A parallel `[]string` would have
to be kept in step with `Labels` by hand. Which segment is live is announced
separately, as a control state, so return the name only — and the name then
does not change when the selection moves. A nil `SegmentLabel` leaves `Chip` to
announce the caption itself.

**It becomes a tab strip with two props and no new field.** As built it is a
group of toggle buttons, which is what a filter bar is. Give the row
`RoleTabList` and the segment template `RoleTab` and the state each `Chip`
already sets goes out as `aria-selected` instead of `aria-pressed`, because the
web exporters pick the attribute from the role:

```go
comps.SegmentedControl{
    Labels:   []string{"Sermons", "Articles"},
    Selected: tab.Get(), OnSelect: func(i int) { tab.Set(i) },
    Style:   []core.StyleProp{core.AccessibilityRole(core.RoleTabList)},
    Segment: comps.Chip{Style: []core.StyleProp{core.AccessibilityRole(core.RoleTab)}},
}
```

It buys the keyboard with it under the WASM runtime: the strip becomes one tab
stop, Left and Right move between segments, and `Home` and `End` reach the
ends. Nothing about the widget changed to get that — the roles and
`aria-selected` are all the runtime needs. See
[WASM — Composite widgets are operable](platforms/wasm.md#composite-widgets-are-operable).

One thing it does not buy, and it is ARIA's rule rather than the widget's: a
tablist claims its children are tabs, so a row that also holds a count or an
add button is not one.

The **panel** is a third prop rather than a limit. `core.AccessibilityID` names
the region the strip switches and `core.AccessibilityControls` points each
segment at it, which is the pair of references `core.Style` grew for exactly
this shape:

```go
comps.SegmentedControl{
    Labels:   []string{"Sermons", "Articles"},
    Selected: tab.Get(), OnSelect: func(i int) { tab.Set(i) },
    Style:   []core.StyleProp{core.AccessibilityRole(core.RoleTabList)},
    Segment: comps.Chip{Style: []core.StyleProp{
        core.AccessibilityRole(core.RoleTab),
        core.AccessibilityControls("library-panel"),
    }},
}
core.Box(
    core.AccessibilityID("library-panel"),
    core.AccessibilityLabel(titles[tab.Get()]),
    page,
)
```

Every segment points at the one region, which is right: there is one panel and
its contents change, so a per-segment id would be naming three regions only one
of which exists. A fully wired tab strip — where the pages are real, separate
elements and each tab names its own — is [`core.TabView`](#tabs), which mints
the ids and writes both ends from the node type. `examples/social` is the
worked example of the hand-built form.

**Segments are keyed**, by `KeyPrefix` + caption. Keys never appear in exported
HTML but they drive reconciler matching and native view recycling, so captions
are assumed distinct — two identical captions collide, which debug mode reports
rather than silently mismatching segments.

## Stepper

A number with a − and a + beside it, for small ranges where two taps beat
opening a keyboard: a quantity, a guest count, a font size.

```go
comps.ListRow{
    Title:    "Guests",
    Trailing: comps.Stepper{Value: guests.Get(), Min: 1, Max: 8,
        OnChange: guests.Set, Label: "Guests"},
}
```

**Clamped in the widget, reported only on change.** A tap computes the next
value, clamps it into `Min`..`Max` and calls `OnChange` only when the result
differs, so the handler is a plain setter. At a bound the button that would
leave the range is disabled.

**Bounds are opt-in.** They apply when `Max > Min`. A zero-value pair leaves
the stepper unbounded, because `Min: 0, Max: 0` is not a range anyone means.

Other notes:

- `Label` is the group's accessible name and is not drawn. Put the stepper in
  a `ListRow`'s `Trailing` or a `FormField` for a visible label.
- The buttons are outlined rather than ghost. A ghost "−" has no visible edge
  and reads as text on a phone.
- The row is `RoleGroup` with the value stated as an accessibility value. The
  natives read its text; the web scopes value attributes to progress bars and
  reads the visible number instead.
- The buttons are announced "Decrease" and "Increase". `DecreaseLabel` and
  `IncreaseLabel` localise them, and `Format` changes how the value is drawn
  and announced.

## Rating

A row of stars, or any glyph, read-only or tappable.

```go
comps.Rating{Value: stars.Get(), OnChange: func(v int) { stars.Set(float64(v)) }}
comps.Rating{Value: 4.5, ReadOnly: true, Label: "Average score"}
```

**`Value` is a float.** Version 1 rounds to whole glyphs, so a caller can store
an average today and gain half-glyphs later without a type change. `OnChange`
reports whole positions because a tap lands on one glyph. Tapping the glyph
that is already the value does nothing.

**Interactive glyphs are buttons, read-only glyphs are decoration.** Each
tappable glyph is a button named "3 of 5". A read-only rating registers no
callbacks and hides its glyphs, so the group's "4 of 5" is the one
announcement rather than five "black star" readings.

Other notes:

- `Max` sets the glyph count and defaults to 5. `Glyph` and `EmptyGlyph`
  default to ★ and ☆.
- Filled glyphs use the warning role's on-light tone, which holds contrast on
  a light surface where the raw warning colour does not.

## Separator

The hairline rule between rows and between sections. The zero value is the
common case:

```go
comps.Separator{}
comps.Separator{Inset: 56}          // starts under the text, not the avatar
comps.Separator{Thickness: 0.5}     // sub-pixel hairline on a 2x display
```

- Always hidden from assistive technology. A rule carries no information,
  and one between every pair of rows turns a 20-row feed into 39
  utterances.
- No forced margin, which is what makes it usable inside a list —
  `core.Divider` force-applies `Margin(8)`, and that is why neither example
  that wanted a rule used it.
- `Inset` is applied as left/right **margin**, not `EdgeInsets.Horizontal`:
  the HTML exporter reads only the four per-side fields.

**Horizontal only, for now.** A vertical rule has to stretch to its row's
height, which is cross-axis stretch, and that used to be the blocker: neither
renderer mapped `AlignItems: "stretch"`, so the rule would have collapsed to
zero height on both. Both map it today — Compose pins a stretched `Row` to
`IntrinsicSize.Max` and gives each child `fillMaxHeight()`, and SwiftUI's
`GrMobFlexStack` proposes the full cross extent to a stretched child — so
adding a `Vertical` field is now a widget change rather than a renderer one.
It has simply not been added. Note the one asymmetry if you hand-roll it
meanwhile: a `Row` reads `AlignItems` only, never the simpler `Align`
fallback (`Align` is a text-alignment concept and has never applied to a
row's vertical axis), so the containing row needs `AlignItems: "stretch"`
spelled out.

The default tint is the theme's `Border` role, read through
`ColorPalette.BorderColor()` rather than off the field, so a theme written
before that role existed falls back to `core.FallbackBorder` (`#E5E5EA`)
instead of rendering an invisible rule. `Color` still overrides per instance.

## Avatar

The circular portrait: a remote image when there is one, initials on a
colored disc when there is not.

```go
comps.Avatar{Src: user.PhotoURL, Name: user.Name}  // image, labelled
comps.Avatar{Name: "Ada Lovelace"}                 // "AL" on a disc
comps.Avatar{Name: "Ada Lovelace", Size: 64}
```

- Both branches are the same square with `BorderRadius = Size/2`, so `Size`
  is the single knob. (An oversized fixed radius would also give a circle,
  but would silently keep the old geometry when `Size` changed.)
- Initials derive from the **first and last** words of `Name` — "Ada King
  Lovelace" is AL, not AK — uppercased, rune-based so non-Latin names keep
  whole characters. `Initials` overrides when the rule gets it wrong.
- The disc is a `Row` with `JustifyCenter` + `AlignItemsCenter`, because
  `Box` is pinned to the top-leading corner on both platforms and cannot
  centre a child.
- Image avatars default to a Surface background: `core.Image`'s theme base
  is `Components.Camera`, whose background is solid black — right behind a
  viewfinder, wrong behind a portrait that has not downloaded yet.

**Accessibility**, unlike `ListRow`, *is* synthesized here, because an
avatar has exactly one meaning and `Name` is it:

| state | result |
|---|---|
| `AccessibilityLabel` set | used verbatim |
| `Name` set | used as the label |
| neither | the node is hidden from assistive tech |

The last row is the important one: an unnamed avatar is decoration beside
text that already names the person, and unlabeled it is announced as
"image" or read out as its URL.

*Non-square images letterbox* inside the circle — the renderers scale with
`.scaledToFit` / Compose's Fit default. `Avatar` does not expose a way to
change that, though the underlying prop now exists:
`core.ImageWithMode(src, core.ContentModeFill, ...)` covers the four modes
(`Fit`, `Fill`, `Stretch`, `Center`) on every renderer. Threading it through
`Avatar` is a widget change waiting for a caller that wants it.

## ProgressBar

The determinate track-and-fill bar.

```go
comps.ProgressBar{Value: 0.45, AccessibilityLabel: "Upload"}
comps.ProgressBar{Value: done / total, Thickness: 10, Color: "#34C759"}
```

- `Value` is clamped to 0–1 rather than rejected (NaN reads as 0): a bar fed
  a live ratio should pin at full and keep rendering.
- The bar takes `core.RoleProgressBar` and states its position through
  `core.AccessibilityValue`, so the name is "Upload" and the value is
  announced separately as it moves. With no label the bar is hidden entirely
  — an unlabeled bar announces a bare number with nothing to attach it to.
- `ValueText` is the spoken form of the value, and is empty by default on
  purpose. Both web targets and Compose localize the percentage themselves
  from the numbers; SwiftUI has no numeric accessibility value at all, so an
  iOS bar announces its name alone unless an app supplies words. Supplying
  them costs the other three their localization, because ARIA and Compose
  announce the text *instead of* the number.
- The percentage used to be appended to the accessible label, because no
  renderer had a progress semantic to carry it. Three of the four do now, and
  a name was the wrong channel for a value that changes: a name is meant to be
  stable, so a bar ticking from 44 to 45 re-announced the whole string rather
  than the part that changed — the same reason `Chip`'s old `", selected"`
  suffix was deleted.
- The fill renders at every value, zero-width included. A constant child
  count keeps advancing progress a *style patch* on one node instead of an
  insert/remove, which is also what lets a `Transition` animate it.

**Why a percentage width and not two flex weights.** The obvious build is
two boxes weighted `FlexGrow(v)` / `FlexGrow(1-v)`. That was exact on
Android, where `FlexGrow` maps onto Compose's `Modifier.weight`, and
silently wrong on iOS, where it mapped onto `frame(maxWidth: .infinity)`:
SwiftUI stacks have no weight, so two growers split free space *equally
regardless of their values*. Every bar would have sat at 50% on iOS. A
percentage width is proportional on all three targets instead:

| target | mapping | accuracy |
|---|---|---|
| Android | `fillMaxWidth(fraction)` | exact |
| HTML | `width:<pct>%` | exact |
| iOS | `containerRelativeFrame` | proportional; measured against the nearest *container*, so a bar that spans its container is exact and one inset in a narrow card reads wide |

Proportional weights have since landed on iOS: `GrMobFlexStack` is that
custom `Layout`, and `GrMobFlexSolver` resolves `FlexGrow` by value on all
three targets. The bar has not been migrated, so the caveat above still
describes what it does today — but the blocker is gone, and moving it to flex
would remove the caveat, since a flex child is measured against its immediate
parent rather than the nearest container.

## FormField

The label / input / hint-or-error frame around any input:

```go
comps.FormField{
    Label: "Email",
    Hint:  "We never share it",
    Input: core.Input(email.Get(), "you@example.com", func(v string) { email.Set(v) }),
}
```

`Error`, when non-empty, **replaces** `Hint` (a field shows one line of
feedback; an error outranks guidance) and inks with the theme's Error color.
The `Input` slot keeps it agnostic to what is wrapped — `Input`, `TextArea`,
`NumericInput`, a custom picker.

The widget renders feedback; it does not produce any. What fills `Error` is
[`forms`](concepts/forms.md), which also decides *when* a message should be
visible:

```go
comps.FormField{
    Label:    "Email",
    Required: form.Required("email"),
    Hint:     "We never share it",
    Error:    form.Error("email"),
    Input:    form.Input("email", "you@example.com"),
}
```

`Required` draws the conventional asterisk after the label, inked in the theme's
Error color and announced to screen readers as "required" — the marker takes
`core.RoleImg`, which is what says the label replaces the glyph rather than
sitting beside it. (`img` is the role for a node whose meaning is carried by
what it looks like; the `group` a named node is otherwise given invites a reader
to announce the label *and* the asterisk it was standing in for.) It is
annotation
only — the widget still validates nothing — which is why it is worth asking
[`form.Required(name)`](concepts/forms.md#the-required-marker) rather than
writing `true`: the form derives its answer from the field's own rules, so the
marker cannot outlive the rule that justified it. It is ignored when `Label` is
empty, there being nothing to mark.

Because the `Input` slot takes any view, wrapping is also how a control with
no error line of its own gets one — a checkbox row, for instance, with the
`ListRow` title standing in for the label:

```go
comps.FormField{
    Error: form.Error("terms"),
    Input: comps.ListRow{
        Leading: form.Checkbox("terms"),
        Title:   "I accept the terms of service",
    },
}
```

## Accordion

A collapsible section — tappable chevron header, content shown while
expanded.

```go
comps.Accordion{
    Title:   "Advanced options",
    Content: advancedPanel,
    InitiallyExpanded: false,
}
```

!!! warning "Accordion owns state"
    Accordion is the one widget in the package that calls `NewState`, so the
    [rules of hooks](concepts/state-and-hooks.md#the-rules-of-hooks) apply to
    it: render it unconditionally, in a stable position, every pass. And
    because `Content` only renders while expanded, it must be **hook-free**
    (interactive, hook-free content is fine — its callbacks re-register on
    every visible pass). [Debug mode](concepts/debug-mode.md) flags
    violations as cursor drift.

`Header` replaces the default title text (the tap target and toggle stay
with the widget); `InitiallyExpanded` seeds the first pass only.

`Title` is a `RoleHeading` at **level 3** — a disclosure sits inside a section,
one tier below a `Card` title or a band — and `HeadingLevel` moves it: a screen
built entirely of accordions under an `AppBar` says 2. As with `Card`, it
applies to the default header only.

### The header is ARIA's accordion shape

This is the one widget in the package whose heading does not ride the words,
and the reason is the other half of what a disclosure has to announce.

```
Box  role=heading  aria-level=3  aria-label="Advanced options"
  Row  role=button  aria-expanded="false"  aria-label="Advanced options"
    "▸"  "Advanced options"        presentational, inside the button
```

The row is the tap target, so the row is the control: it states `RoleButton`
and [`AccessibilityExpanded`](concepts/styling-and-theming.md#accessibilityexpanded),
and `aria-expanded` is defined for a button and **not** for the `group` role a
named row would otherwise be supplied. Without that, a header announces what it
is called and never that it can be pressed or whether it is open.

A button's children are presentational, so the tier cannot stay on the title
inside it. It moves to a `Box` wrapped around the row, **named explicitly** with
the `Title` — which is what stops the heading from being called "▸ Advanced
options", since a heading with no name of its own takes one from its content.
That was the objection this widget raised against the wrapping shape for two
releases, and `AccessibilityLabel` is the answer to it.

A reader hears the question twice: once as an outline entry to jump to, once as
a control that says collapsed or expanded. That is what every accessible
accordion on the web does.

The chevron used to be deliberately left audible, because it was the only thing
on screen that said which way the disclosure pointed. It is presentational now
— by construction rather than by choice, being inside a button — and the state
says it in a channel that does not depend on a reader pronouncing "▸".

A `Header` slot gets the button and its state and **no** heading, on the same
division `Card.Title` / `Card.Header` draws: you replaced the content, so the
widget will not stamp an outline entry named by a `Title` that is not on screen.

### It is now a shared shape

The arrangement above is `comps.disclosure`, and the collapsible
`GroupedList` band is built out of the same value. It moved there when the
second consumer arrived: the argument took three attempts and both rejected
ones looked correct in an export, so a hand-copied second version would have
been checked only against its own expectations. `TestBothDisclosuresBuildTheSameShape`
renders the two side by side and compares the tier, both names, the state and
the chevron in both directions.

The type also makes the pairing structural. `Expanded` and `OnToggle` are
fields of one struct because a stated expansion with no handler is announced
on both web targets and is silently nothing on Android — Compose wires its
expand/collapse actions to the node's own click callback and offers neither
without one. [Debug mode](concepts/debug-mode.md) reports that as an inert
disclosure.

## Tabs

The named-field facade over `core.TabView`:

```go
comps.Tabs{
    Items:    []core.TabItem{core.Tab("Home", "🏠"), core.Tab("Search", "🔍")},
    Selected: tab.Get(),
    OnChange: func(i int) { tab.Set(i) },
    Content:  []core.View{homePage, searchPage},
}
```

Tabs *wraps* rather than supersedes `TabView`: the `"TabView"` node type is a
wire contract the native renderers consume, and node-type contracts live in
core — this struct only supplies the field names. All pages are children of
the node; the native side shows the selected one. With no `OnChange`, no
callback is registered, keeping static tab strips diff-stable.

All four targets draw a bar above the selected page. The two DOM ones hide the
other pages rather than dropping them, and wire each tab to its page as an ARIA
tab set — `role="tabpanel"`, `aria-controls`, `aria-labelledby` — so a screen
reader on the web knows which region each tab governs (see
[WASM — Tab views](platforms/wasm.md#tab-views)); the natives compose only the
selected one, and give it a fresh identity on each switch, so per-tab view state
is dropped there. `Selected` is controlled state on every target: a switch
arrives as a prop change, never as a rebuilt subtree. The `Icon` of a
`core.TabItem` is drawn by no target.

## Collections — GroupedList & DataTable

Keyed, lazily-composed collections over `core.List`, with run-length group
headers, controlled sort, compact mode and client- or server-side paging.
`Pagination` (numbered pages) and `LoadMore` (the four-state tail: nothing /
Load more / Loading… / error + Retry) are the footers.

```go
comps.DataTable[Entry]{
    Columns: []comps.Column[Entry]{
        {Title: "Title", Weight: 2, Text: title, Less: byTitle},
        {Title: "Speaker", Narrow: true, Text: speaker},
    },
    Rows: entries, Key: entryKey,
    Sort: sortBy.Get(), OnSort: func(s comps.Sort) { sortBy.Set(&s) },
    Pagination: &comps.Pagination{Page: page.Get(), PageSize: 20, OnChange: page.Set},
}
```

Both are hook-free and fully controlled. Rows are sorted, then paged, then
grouped. Give a column `Less` only when `Rows` is the whole set: `Less` sorts
all of `Rows` and only `Rows`, so sorting an accumulated window yields the
first rows *of the window* under a header claiming the first rows of the
table. When the server pages, set `Sortable` without `Less` and put the sort
in the query — [debug mode](concepts/debug-mode.md) reports the detectable
half of that as a `partial-sort` concern. `HideTrailingCount` suppresses the
last group's badge while a pager still has pages to fetch, since a closed
group's count is final and an open one's is about to change.

### Sticky bands and infinite feeds

`StickyHeaders` pins each group band to the top of the viewport while its run
scrolls underneath (both widgets); `OnEndReached` fires when the reader gets
within a few rows of the bottom, so the next page arrives without a tap
(`GroupedList`).

```go
comps.GroupedList[Entry]{
    Items:         pager.Items,
    GroupBy:       byMonth,
    StickyHeaders: true,
    OnEndReached:  pager.LoadNext,   // the scroll
    Footer: comps.LoadMore{     // and the tap, and the states
        HasMore: pager.HasMore, Loading: pager.Loading,
        Err: pager.Err, OnLoadMore: pager.LoadNext},
}
```

**Keep the footer.** Auto-loading replaces the tap, not the tail: `LoadMore`
is still where "Loading…" and a failed page's Retry live, and it is the
manual fallback wherever the edge cannot be reported (a static export, a
browser with no `IntersectionObserver`). Handing the same load function to
both is the intended shape — `core.OnEndReached` will not re-ask until the row
count changes, so a tap and a scroll cannot double-load.

**A shut trailing group withholds the edge.** `Collapse` and `OnEndReached` are
the one pair of features here that are in tension. An append pager can only
extend the *last* run, and a shut run emits no rows — so a page fetched while
the bottom group is collapsed lands nowhere, and the guard above (which watches
the row count) then refuses every fire after it. The feed reads as exhausted
while the pager's offset has quietly moved on. So the widget withholds the prop
while that group is shut and restores it the moment the reader opens it; the
`Footer` stays reachable, which is the other reason to keep it. A shut group
*above* the last one changes nothing — the pager was never going to extend it.

**Ask `AutoLoadWithheld()` if the footer is conditional.** Withholding is
silent by construction: a feed that stopped fetching because the last run is
shut and a feed that has genuinely run out produce the same tree. A screen
whose `Footer` is always a `LoadMore` is fine — the button is there and calls
the same function — and the shape that is not is a footer hidden on the
strength of auto-loading doing the work. The answer is composed from `Items`,
`GroupBy` and `Collapse`, three fields none of which means anything alone, so
the widget is the one that can give it:

```go
list := comps.GroupedList[Sermon]{
    Items: pager.Items, GroupBy: byMonth, Collapse: shut,
    OnEndReached: pager.LoadMore,
}
// Shown when there is more to fetch *and* nothing is fetching it.
if pager.HasMore && list.AutoLoadWithheld() {
    list.Footer = comps.LoadMore{HasMore: true, OnLoadMore: pager.LoadMore}
}
```

It answers `false` when `OnEndReached` is nil — there is no sensor to withhold
on a manual pager. `Collapse.IsCollapsed` is the question to ask about the run
itself.

**A `Header` override is told the same two things on the `Group`.** The method
answers the *caller*, who owns the footer. An override is a different reader in
a different place — it is handed a `Group` and nothing else — so `Group` carries
the two facts only the widget knows:

| field | what it says |
|---|---|
| `Trailing` | this is the last run, the one an append pager extends |
| `AutoLoadWithheld` | this run being shut is why the list has no edge sensor |

```go
Header: func(g comps.Group) core.View {
    band := core.Row(comps.CollapseBand{Collapse: shut, Group: g})
    if g.AutoLoadWithheld {
        band = core.Row(band, comps.Badge{Text: "paused"})
    }
    return band
},
```

`Trailing` is what makes `HideTrailingCount`'s rule implementable in an
override: the rule is *do not publish an open run's count*, and until the field
existed nothing handed to the override said which run was open — deriving it
meant re-walking `Items` with the same `GroupBy` the widget had just walked.
Both are filled in by the widget like `Count`, so a value a `GroupBy` callback
sets is overwritten, and both are stamped before *anything* reads the `Group` —
the `Collapse` predicate, the override, the default band, `OnToggle` — so every
reader sees one shape.

`AutoLoadWithheld` is true on at most one group of a list and always a
`Trailing` one, false throughout a list with no `OnEndReached`, and false on
every band of a `DataTable`, which has no edge sensor at all. The widget states
it rather than letting a band derive it because the composite is easy to get
subtly wrong: it is `Trailing` *and* the run is hidden *and* a sensor was given,
and "the run is hidden" needs `OnToggle` as well as `IsCollapsed` — a caller
with a predicate and no handler hides nothing, so their bands would announce a
pause the list is not taking.

The default `GroupHeader` ignores it. What a band says about a paused feed is a
wording decision, and the default band's vocabulary is a label and a count.

`StickyHeaders` pins the *default* `GroupHeader`. A `Header` override builds
its own view, which the widget cannot reach into; such a header pins itself
with `core.StickyHeader()` in its own `Style`. On `DataTable` the flag pins
the group bands and not the column header — the column header is a sibling of
the body list rather than a row inside it, which is what keeps it on screen in
the first place.

A band's label is a `RoleHeading` at **level 2** — a section of the screen
whose name `AppBar`'s title carries at level 1. Without the tier the two
announce as peers and a reader navigating by heading cannot tell the screen
from the month inside it. The role sits on the label rather than on the band,
so the heading's name is "March" and not "March, 12"; the count badge is still
announced, as the separate thing it is.

`HeadingLevel` on either collection moves the default bands: a banded list
inside a `Card` says 3, and a feed on a screen with **no bar at all** says 1 —
which used to be unsayable, so a barless feed started its outline at 2 with no
1 above it. It reaches `GroupHeader`, so it does nothing without `GroupBy` and
nothing under a `Header` override, the same division `StickyHeaders` draws. The
*column* header is unaffected: its cells carry `RoleColumnHeader` and no tier,
because `aria-level` is defined for `heading`, `listitem` and `row` and
pointedly not for `columnheader`.

`DataTable` states its structure as well as drawing it:
`RoleTable` on the table, `RoleRowGroup` on the body list, `RoleRow` on the
header and body rows, `RoleColumnHeader` and `RoleCell` on their cells. The
rowgroup is load-bearing rather than decorative — ARIA reads the rows a table
*owns*, and the body list is a container between the two, so without it the
other four describe a table with no rows. A busy or empty table withholds the
rowgroup, since what the body holds then is one placeholder and not rows.

### Collapsible bands

`GroupedList.Collapse` turns the bands into disclosures whose runs the reader
can shut. The state is the caller's:

```go
shut := core.NewState(ctx, map[string]bool{})

comps.GroupedList[Entry]{
    Items: entries, GroupBy: byMonth,
    Collapse: comps.Collapse{
        IsCollapsed: func(g comps.Group) bool { return shut.Get()[g.Key] },
        OnToggle: func(g comps.Group) {
            next := maps.Clone(shut.Get())
            next[g.Key] = !next[g.Key]
            shut.Set(next)
        },
    },
}
```

**Why the caller holds it.** `GroupedList` calls no hook, which is what lets
it be rendered conditionally — inside a `core.IfElse` against a pager's loaded
flag — without disturbing your hook cursor. Owning collapse state would end
that, and the widget is the wrong place for it anyway: which months are shut
is screen state, it usually wants to survive a pager reload, and a screen that
wants "collapse all" has no way to reach inside a widget's `NewState`.
`Accordion` is the other answer to the same question and stays the right one
for a single section.

The two functions are one type because they are useless apart: a predicate
with no handler would hide rows behind a band nobody can open, so it hides
nothing. The zero `Collapse` is the list exactly as it was.

A collapsed run emits **no rows at all** — not hidden ones — so a shut month
costs the reconciler nothing and reaches no document. The band keeps its key
across the toggle, so re-opening patches the rows back rather than remounting
the band.

The band becomes [the disclosure shape](#it-is-now-a-shared-shape) `Accordion`
uses: a heading wrapping a button that carries `aria-expanded`, with the
chevron inside the control. The count badge stays **outside** the button — a
button's children are presentational, and the count is content rather than
chrome — which also keeps the heading named "March" rather than "March 12".
The cost is that the badge is not part of the tap target.

The band's own insets **are**. They live on the control rather than on the row
that holds it, so the 16px before the chevron and the 4px above and below are
part of what a finger hits:

```
 Row ────────────────────────────────────      Row ───────────────────────────
│      ┌──────────────────┐     ┌───┐    │    │┌──────────────────────┐ ┌───┐ │
│ 16px │ ▸ January 2026   │ 8px │ 3 │ 16 │    ││  ▸ January 2026      │ │ 3 │ │
│      └──────────────────┘     └───┘    │    │└──────────────────────┘ └───┘ │
 ────────────────────────────────────────      ───────────────────────────────
 the insets are dead space                     the control owns them
```

Nothing moves: padding on a stretched child fills exactly the space the same
padding on its parent held. The band `Row` keeps its fill, its cross-axis
centering and `core.StickyHeader` — it has to, since that is the node the list
sees as its child — plus the badge's own trailing inset, which is the one that
is past the control's edge. `GroupHeader.Style` therefore no longer reaches the
padding; **`ControlStyle` does**, and it is where a caller's own chrome belongs.

Unlike `StickyHeaders` and `HeadingLevel`, `Collapse` is *not* ignored under a
`Header` override. The override owns the band; this owns whether the rows
under it are emitted, which is not something a view you built can reach.

That division left the override author holding three things at once: a button,
an `aria-expanded` stated on every pass open or shut, and a heading wrapper
whose nesting order is four paragraphs of argument in an unexported type — and
the shape most likely to come out of that is one of the two `Accordion` tried
and discarded, both of which look correct in an export.

So there are two ways to get a correct control. `GroupHeader` is the answer when
the whole default band will do; it is exported, takes `Expanded` and `OnToggle`,
and builds all of it. `CollapseBand` is the answer when it will not — the
disclosure alone, with no `Surface`, no padding and no count badge, for placing
in a row of your own:

```go
Header: func(g comps.Group) core.View {
    return core.Row(
        comps.CollapseBand{
            Collapse: shut, Group: g,
            // Your chrome, on the control — not on the Row, where a press
            // would do nothing. This is the same move the default band makes.
            ControlStyle: []core.StyleProp{
                core.PaddingLeft(16), core.PaddingRight(8)},
        },
        comps.Avatar{Name: leader[g.Key]},
        comps.Badge{Text: strconv.Itoa(g.Count)},
        core.PaddingRight(16),
    )
},
```

It takes your own `Collapse` — the same value handed to the list — which is what
keeps the control and the row hiding answering to one state. `Content` replaces
the words inside the button and never the announced name, because a button's
children are presentational and the name comes from `Group.Label`. An inactive
`Collapse` builds a plain heading rather than a control with nothing behind it:
a stated expansion with no handler is what `core.AuditTree` reports as
`ConcernInertDisclosure`. `ControlStyle` lands on that stand-in too, so a band
does not change size on the day it gets a handler.

`DataTable` does not take it. A table's band sits inside the body's rowgroup,
where ARIA has no reading for it even as a plain heading — making it a button
would be a second claim on a structure that is already the wrong shape. The
fix is per-band rowgroups, which is a change to the row emission both widgets
share.

Lesson 4.6 of the [interactive tutorial](tutorial-interactive.md) now builds
one, which it did not before: `CollapseBand`'s only readers were its own tests,
and the field whose whole justification is how a *real* custom band is assembled
(`ControlStyle`) had never been assembled into one. The demo's bands all start
shut, which is the half a `Header` override does not own — the run is withheld
by the widget, not by anything in the header — and the badge sits outside the
button, which is the half it does.

See lessons 4.6 and 4.8 of the [interactive tutorial](tutorial-interactive.md)
and the godoc for the full field list.

## AppBar

The title strip at the top of a screen: an optional back affordance, the
screen's name, and trailing actions.

```go
comps.AppBar{
    Title:    "Sermon",
    Subtitle: "22 March 2026",
    Actions:  []core.View{shareButton},
}
```

- **The back control appears only when there is somewhere to go.** With no
  `Leading` and no `HideBack`, the arrow is drawn exactly when
  `core.CanPop(ctx)` is true — so a tab root gets none without asking and a
  pushed screen gets one without wiring. `OnBack` *replaces* `core.Pop`
  rather than running before it, which is what makes a confirm-before-leaving
  handler possible.
- `Leading` replaces the automatic control entirely — a close button on a
  modally presented screen, where `CanPop` is false. `Content` replaces the
  Title/Subtitle stack.
- The title takes the theme's **Subtitle** size with the primary ink and a
  bold weight. `Typography.Title` is the screen's large heading (28pt under
  `DefaultTheme`) and does not fit in a bar.
- A hairline rule is drawn under the bar unless `HideSeparator` is set,
  because an unstyled bar sits on the same Background as the content below it.

The bar carries `RoleBanner` and its `Title` carries `RoleHeading` at **level
1**, so a reader navigating by landmark or by heading lands where they expect.
Level 1 because an `AppBar` is the screen's own bar: there is nothing above it
for it to be a section of, and the tier is what keeps it from announcing as a
peer of `GroupedList`'s band labels, which take 2. The banner sits on the bar
row (not on the box the separator adds around it) so `Style` can override it —
a second bar on a screen that already has a banner should say something else.

It is an ordinary `Row`, not a platform navigation bar: nothing floats,
collapses on scroll, or claims the status bar. `Screen`'s `SafeArea` is what
keeps it clear of the notch.

## BottomBar

The strip pinned to the bottom of a screen: two to five destinations or
actions, each an icon over a label. It belongs in `Screen.Footer`.

```go
comps.Screen{
    Children: []core.View{content},
    Footer: comps.BottomBar{
        Items: []comps.BarItem{
            {Icon: "🏠", Label: "Home",   OnTap: func() { tab.Set(0) }},
            {Icon: "🔍", Label: "Search", OnTap: func() { tab.Set(1) }},
            {Icon: "👤", Label: "Me",     OnTap: func() { tab.Set(2) }},
        },
        Selected: tab.Get(),
    },
}
```

**The role follows `Selected`.**

| `Selected` | Role | Meaning |
|---|---|---|
| 0 or more | `RoleNavigation` | destinations, one of them current |
| negative | `RoleToolbar` | actions, none of them current |

The zero value selects the first item, as `SegmentedControl` and `Tabs` do, so
an action strip says `Selected: -1`.

**Items share the width.** Each cell grows equally, so the tap targets tile the
bar with no dead gaps between them.

Other notes:

- The current item is drawn bold in the primary on-light tone, and its
  accessible name gains ", selected". Core has no current-page state, and a tab
  role would claim a panel the bar does not control.
- The icon is decoration and hidden from assistive technology.
- `BarItem.AccessibilityLabel` replaces an abbreviated label as the spoken name.

## Banner

The inline strip that tells the user something about the screen they are on:
a failed refresh over content that is still good, an offline notice, a
"Reconnecting…".

```go
comps.Banner{
    Text:        "Could not refresh. Showing a saved copy.",
    Variant:     comps.VariantWarning,
    ActionLabel: "Retry", OnAction: reload,
    OnDismiss:   func() { notice.Set(false) },
}
```

**The variant is a tint, not a fill.** A hairline border and the leading glyph
take the role's
[on-light tone](concepts/styling-and-theming.md#the-on-light-tones); the strip
keeps the theme's Surface and the primary ink. A saturated Error red across the
width of a screen reads as a failure of the app rather than of one fetch, and
the palette carries no muted *container* tone to fill with instead — an
on-light tone is the opposite end of the range, ink for a light surface rather
than a wash to sit behind one. The upshot is that a banner's contrast does not
depend on which variant it is.

The glyph is the mark that most needed the second tone: under `DefaultTheme` a
warning's ⚠ in `Colors.Warning` on Surface is about 2:1. The tones are stated
against a theme's `Background` and drawn here on its `Surface`, which costs
roughly 7% — all four still clear AA.

Default glyphs are `ⓘ ✓ ⚠ ⊗` for the four roles, overridable with `Glyph` and
droppable with `NoGlyph`. They are **decoration** and are hidden from
assistive technology, so `Text` has to carry the meaning — "Could not
refresh", not "Something went wrong" beside a red edge.

**It announces itself when it appears.** A banner shows up because something
changed, usually while the reader is elsewhere on the screen, so the strip is
a live region: `RoleAlert` when the variant is Error, `RoleStatus` otherwise —
error interrupts, everything else waits for a pause. Override it through
`Style` for a strip that is really static content.

It is not a toast: `core.ShowToast` disappears on a timer, a Banner stays
until the state that produced it changes. For an edge-to-edge strip with no
frame, pass `core.BorderWidth(0)` and `core.BorderRadius(0)` in `Style`.

## Dialog

The "Delete this note?" moment: a title, a sentence, and at most two buttons
over the screen, built on `core.Modal` and `comps.Card`.

```go
comps.Dialog{
    Visible:   confirming.Get(),
    Title:     "Delete note?",
    Message:   "This cannot be undone.",
    Confirm:   comps.DialogAction{Label: "Delete", Variant: comps.VariantError, OnTap: del},
    Cancel:    comps.DialogAction{Label: "Keep"},        // nil OnTap uses OnDismiss
    OnDismiss: func() { confirming.Set(false) },
}
```

**One struct, three shapes.** Which actions carry a `Label` decides the shape.

| Confirm | Cancel | Shape |
|---|---|---|
| set | set | confirm, the two-button "are you sure?" |
| set | empty | alert, one acknowledgement button |
| empty | empty | sheet, title and `Body` only |

**Button order is fixed.** Cancel sits on the leading side and is drawn ghost.
Confirm sits on the trailing side, filled with its own `Variant`, so
`VariantError` gets the theme's Error fill. The pair is packed to the trailing
edge. Material 3 and Apple's guidelines agree on this order, and there is no
knob, because an order field is exactly the disagreement the widget removes.

**Controlled, like `core.Modal`.** The dialog never closes itself.

- A scrim tap, the Android back gesture and an iOS swipe-down all report
  through `OnDismiss`. Leave it nil for a dialog that must be answered.
- A `Cancel` with a nil `OnTap` calls `OnDismiss`, so "Keep" and a scrim tap
  cannot drift apart.
- `Confirm.OnTap` does not close the dialog. The handler sets `Visible` false
  once the work it started has an outcome.

Other notes:

- `Body` replaces `Message` with any view: a field, a checkbox row, a list.
- `DialogAction.Disabled` holds a Confirm back until a `Body` field is valid.
- The Modal chassis already writes `role="dialog"` on the web and the natives
  present a platform dialog. The widget adds only the name: the card carries
  `AccessibilityLabel(Title)` and the title text is a heading.
- `Style` lands on the card after the widget's own props, so a caller can
  cap the width with `core.MaxWidth` or replace the label.
- iOS presents a `core.Modal` as a sheet rather than a centred card. That is
  the host's rendering of the chassis, not something the widget chooses.

## ActionSheet

A short list of actions on the bottom edge of the screen, such as Share, Copy
link and Delete, with a separate Cancel. It is built on `core.Modal` and
`comps.Card`.

```go
comps.ActionSheet{
    Visible: open.Get(),
    Title:   "Note",
    Actions: []comps.SheetAction{
        {Label: "Share", OnTap: share},
        {Label: "Delete", Variant: comps.VariantError, OnTap: del},
    },
    Cancel:    "Cancel",
    OnDismiss: func() { open.Set(false) },
}
```

**Picking an action closes the sheet.** A tap runs the action's `OnTap` and
then `OnDismiss`, so no handler closes the sheet itself. This is the one way it
differs from `Dialog`, whose Confirm never closes. A sheet is a menu, and a menu
closes on selection. An action that needs a follow-up question opens a `Dialog`
from its `OnTap`.

**How it reaches the bottom edge.** `core.Modal` has no placement prop. It
centres its content on the web and on Android, and iOS presents it as a bottom
sheet. The widget puts a growing, invisible filler above its card:

| Target | What the filler does |
|---|---|
| web | grows along the overlay's column and pushes the card to the bottom |
| Android | is weighted, so the dialog window's column fills and the card is last |
| iOS | has no height, because the sheet is already at the bottom |

The filler also reports a tap above the panel as a dismiss. On the web and
Android that tap lands inside the Modal's content rather than on its scrim.

Other notes:

- Actions are full-width ghost buttons. `VariantError` gives a destructive
  action the error ink, and `Disabled` holds one back. A disabled action does
  not dismiss the sheet.
- Actions are buttons, not listbox options. They are commands, and an option
  would be announced "not selected".
- `Cancel`, a tap above the panel and the scrim all call `OnDismiss`. With
  `OnDismiss` nil, only `Visible` closes the sheet.
- The card carries `AccessibilityLabel(Title)` and the title is a heading. The
  Modal chassis supplies the dialog semantics.
- `Style` lands on the card last. `core.MaxWidth` keeps the panel from spanning
  a wide browser window.
- The Android and iOS placement comes from reading the renderers. It has not
  been checked on a device.

## Snackbar

The "Note deleted · Undo" strip: one line of text and at most one action,
shown for a few seconds.

```go
comps.Snackbar{
    Visible:   undo.Get() != nil,
    Message:   "Note deleted",
    Action:    "Undo",
    OnAction:  restore,
    OnTimeout: func() { undo.Set(nil) },
}
```

**Why not `core.ShowToast`.** A toast is drawn and removed by the host and
cannot carry a button. An action callback on the toast would be a change to
all four renderers. A widget needs none of them.

**Controlled, and it holds one hook.** The caller owns `Visible`. `OnTimeout`
fires once, `Duration` after `Visible` turns true, and `OnAction` fires when
the action is tapped. Neither hides the strip. The timer is
`hooks.UseTimeoutWhile`, keyed on `Message`:

- hiding the snackbar cancels a pending timeout
- a new `Message` while it is up restarts the timer

Like `Spinner`, render it on every pass and drive `Visible`. Leaving it out of
the tree moves its hook slot.

**Where it goes.** It is a strip, not an overlay, so the caller places it.

| Placement | Behaviour |
|---|---|
| `Screen.Footer: core.Column(snackbar, bottomBar)` | pinned above the bar; the scroll region shrinks while it is up, so it never covers a row |
| a `core.ZStack` layer with `core.StackAlign(core.StackAlignBottom)` | floats over content that fills the stack |

Other notes:

- `Duration` zero means `SnackbarDuration` (4 seconds). A negative value, or a
  nil `OnTimeout`, never times out.
- The strip is a `RoleStatus` live region, read at the next pause.
  `VariantError` makes it `RoleAlert`, which interrupts. It carries no
  accessible name, because a label would replace the message.
- The default look is the page inverted: `TextPrimary` fill, `Background` ink.
  A `Variant` fills with its colour and picks a contrasting ink.
- An `Action` with a nil `OnAction` draws no button.

## EmptyState

The centered placeholder for content a screen does not have — and for the
other two moments with the same shape:

```go
empty   comps.EmptyState{Glyph: "📭", Title: "No messages yet"}
busy    comps.EmptyState{Title: "Loading sermons…"}
failed  comps.EmptyState{Glyph: "☁", Title: "Could not reach the server.",
            ActionLabel: "Retry", OnAction: reload}
```

Three states, one widget, so their wording and spacing cannot drift apart.
The busy case is a line of text rather than a spinner because core has no
indeterminate progress node — and naming what is loading is more useful than
an animation anyway.

A wait renders in two places, and the words move between them. On a screen of
its own it goes in `Title`, as above. Under the last row of a paged list — the
footer saying the next page is coming — body-sized primary ink reads as one
more row, so there it goes in `Hint` with `Title` left empty, usually with the
padding brought in too:

```go
tail  comps.EmptyState{Hint: "Loading more…",
          Style: []core.StyleProp{core.Padding(t.Spacing.SM)}}
```

The rule underneath is *errors and empties speak in the primary line; a wait
speaks there only when it is the whole screen*. The widget cannot apply it — it
is handed a slot and never learns whether that slot is a screen's middle or a
list's end — so it is the caller's, written down so two screens do not answer
it differently.

The column sets `Width: 100%`, which looks redundant and is not: on both
natives a column hugs its widest child, so without it the block sits at the
leading edge with its children centered inside a box only as wide as the
longest line. The DOM targets fill the line already, so the bug is invisible
on the target you are most likely to be looking at.

The built action is **outlined**, not filled: an empty state is a dead end,
and a solid Primary button in the middle of an empty screen is the loudest
thing on it.

## SearchField

A text field dressed as a search box: a leading magnifier, a flexible input,
and a clear button that appears once there is something to clear.

```go
d := hooks.UseDebounce(ctx, 250*time.Millisecond)

comps.SearchField{
    Value: query.Get(),
    OnChange: func(s string) {
        query.Set(s)                      // now: the field is controlled
        d.Call(func() { runSearch(s) })   // in 250ms, if the typing stopped
    },
    OnSubmit: func() { d.Cancel(); runSearch(query.Get()) },
}
```

**It holds no state and calls no hook**, which is what lets a search box live
in a header that appears and disappears — a hook-slot consumer could not.
That is also why it cannot debounce its own `OnChange`: a controlled field's
value has to reach state on the keystroke or the characters do not appear.
What wants delaying is the *reaction*, and that lives in the caller — see
[`hooks.UseDebounce`](concepts/state-and-hooks.md#hooksusedebouncectx-delay).

The row paints the theme's Surface at the theme's own field radius and the
input inside it is flattened (transparent, no radius, no padding, no border),
so there is one box rather than two. The border half of that is newer than the
rest: the theme's `Input` base now states a field frame, so without it the
second box would be drawn deliberately and on all four targets. `AccessibilityLabel` falls back to the resolved
placeholder, because a placeholder is not a label on any platform — it
vanishes on the first keystroke.

The row carries `RoleSearch` — the landmark is the whole region, so a reader
jumping to it arrives before the clear button rather than past it. The input
keeps its own name; a role says what a region is and a label says what a
control is called.

## ChipStrip

A run of `Chip`s that wraps onto as many lines as it needs — a filter bar,
tags on an article, quick amounts on a form.

```go
comps.ChipStrip{Chips: []comps.Chip{
    {Label: "All",      Selected: f == "",        OnTap: func() { filter.Set("") }},
    {Label: "Sermons",  Selected: f == "sermon",  OnTap: func() { filter.Set("sermon") }},
}}
```

The field is `[]Chip` rather than a parallel vocabulary of labels and
callbacks, so a chip in a strip is configured exactly like a chip anywhere
else. `Children` is the escape hatch for a strip mixing chips with something
else.

**ChipStrip is not SegmentedControl.** The segmented control is one-of-N: a
fixed, exhaustive set drawn as one joined control. ChipStrip is the loose
case — any number selected including none, a set that comes from data.

`Scrollable` makes the strip one line that pans sideways instead of a block
that wraps:

```go
comps.ChipStrip{Scrollable: true, Chips: years}
```

Wrapping is right for a set the reader should see all of — the tags on an
article, the references on a sermon. Panning is right for a long filter bar,
where a strip growing to three lines pushes the content it filters off the
screen and the chips past the fold read as "there is more" rather than as a
queue. The two are exclusive: a scrolling strip is one line, so there is
nothing to wrap.

Under the hood it becomes a `core.Scroll` carrying `core.Horizontal()`, not a
`Row` with an overflow — the natives implement sideways panning in their
scroll composites alone, so the node type has to change. `Style` still lands
on the strip itself either way.

The strip claims **no** role of its own. A filter bar is a toolbar and can say
so with `Style: []core.StyleProp{core.AccessibilityRole(core.RoleToolbar)}`,
but the tags on an article are not one, and the widget cannot tell them
apart — nor does it implement the roving focus a toolbar implies.

## Skeleton

The grey placeholder that holds a screen's shape while its content loads.

```go
comps.Skeleton{}                                        // one line
comps.Skeleton{Lines: 3}                                // a paragraph
comps.Skeleton{Width: "44px", Height: 44, Radius: 999}  // an avatar
```

A stack's last bar is short (`LastLineWidth`, 60% by default), which is what
makes it read as a paragraph — applied only when `Lines` is 2 or more, since
on a single bar the last line is the only line.

The bars take the palette's **Border** role, not Surface: Surface is a
*panel's* fill, so a Surface bar inside a card disappears — the same trap
`Separator` documents. They are hidden from assistive technology and the
container carries the label (`"Loading"` by default) under `core.RoleStatus`.

That role is what makes the wait *announce*. A named container with no role is
given `group` by both web targets, which makes the name legal and stops there:
a reader says "Loading" only if the user happens to walk onto the block. A
skeleton is not a group of things — the bars stand in for content that is not
here yet — and `status` is ARIA's word for one advisory that is replaced, which
is the region's whole contract. It is a live region on the two web targets and
on Android (Compose's polite live region); SwiftUI has no live-region property,
so on iOS this is still a labelled container that VoiceOver reads on arrival
without interrupting for.

**No shimmer.** A moving highlight is a repeating keyframe animation, and
`core.Transition` animates a property between two declared values. Looping it
from Go would push a render pass and a bridge patch per frame of a
decoration, which is the one thing the "declare in Go, animate natively" model
exists to avoid.

Skeleton and EmptyState answer different questions: a skeleton says content is
coming and will look roughly like this (worth saying when the layout is
known); an empty state says there is nothing here and why.

## Spinner

The "something is happening, shape unknown" indicator. `Skeleton` covers a
known layout, `ProgressBar` covers measurable work, and `Spinner` covers the
rest.

```go
comps.Spinner{Hidden: !loading.Get()}
comps.Spinner{Size: comps.SpinnerLarge, Label: "Uploading"}
```

**It holds hooks, so render it unconditionally.** No renderer draws a looping
animation on its own, so the spin is stepped from Go: 30 degrees every 80
milliseconds, through a state slot and `hooks.UseIntervalWhile`. Like
`Accordion` and `DatePicker`, it must be rendered in a stable position on every
pass. Set `Hidden` to stop showing it; leaving it out of the tree moves its
hook slots.

**`Hidden` is also what makes it free.** A visible spinner costs a render pass
per step. `Hidden` hides the node and pauses the interval, and a paused
`UseIntervalWhile` tick requests no render. A spinner built on plain
`UseInterval` would re-render the whole app on every tick for the life of the
process.

Other notes:

- The ring carries an orbiting dot, because a uniform ring turned about its
  centre draws the same pixels at every angle.
- The outer box is a `RoleStatus` live region named by `Label`, which defaults
  to "Loading". The ring is hidden from assistive technology so its steps are
  never read.
- `SpinnerSmall`, the default medium and `SpinnerLarge` read the theme's
  `Spacing.MD`, `LG` and `XL`.

## StatTile

One figure with its name and, optionally, its movement.

```go
core.Row(core.Gap(12),
    comps.StatTile{Label: "Attendance", Value: "412", Fill: true,
        Delta: "+18 vs last week", DeltaVariant: comps.VariantSuccess},
    comps.StatTile{Label: "Giving", Value: "MZN 42,750", Fill: true},
)
```

**It has no frame.** "Tile" names the content, not a card: the widget paints
and insets nothing, which is what lets three tiles share one `core.Card` or
take one each, rather than a `Framed` bool that is wrong half the time.

**The delta's zero variant is neutral, not Primary** — the one place in this
package where `VariantDefault` is not the theme's brand color. A delta is a
measurement, and whether a number going up is good is the caller's domain:
attendance up is a success, spend up is not, latency up is an incident. So
the default says nothing. A colored delta takes the role's
[on-light tone](concepts/styling-and-theming.md#the-on-light-tones) rather than
its fill color, for the reason an outlined [`Button`](#contrast-and-what-the-widget-can-promise)'s
label does: the line is *read*, on whatever the tile was dropped into, and the
tile paints no background to pick an ink against. Under `DefaultTheme` a
Success delta was 2.22:1 and a Warning delta 2.20:1 before the palette had a
second value per role.

`Fill` sets `FlexGrow` **and** a zero `FlexBasis`, which is what makes the
four targets agree: Compose and SwiftUI divide the whole axis by weight, CSS
divides only the leftover space. The natives ignore `FlexBasis`, so the prop
that is inert on two targets is exactly the one that converges the other two.

## Compass

A bearing drawn as a compass rose.

```go
h := hooks.UseHeading(ctx)   // starts the device compass, releases it on unmount

switch {
case !h.Received:  return comps.Skeleton{}            // no reading yet
case !h.Available: return comps.EmptyState{Hint: h.Error}
default:           return comps.Compass{Heading: h.Magnetic, ShowDegrees: true}
}
```

**The rose turns, not a needle.** A magnetic compass has a fixed card and a
needle that swings to north; a navigation compass — every phone — turns the
whole card under a fixed mark at twelve o'clock. This is the second, because
the question a phone user is asking is "which way am I facing", which is read
off the top. So the rose is drawn with `core.Rotate(-Heading)`: turning the
device clockwise must turn the rose counter-clockwise by the same amount for
it to keep pointing at the same piece of the world.

**The mark sits on the rose's rim, drawn over it.** For a long time it could
not: `Box` stacks vertically on all four targets and absolute positioning is
web-only, so a mark drawn *over* the rose would have been a web-only widget
wearing a portable name, and it was parked in the row above instead.
[`core.ZStack`](concepts/views.md#containers) is the container that fixed it,
and this is the widget it was added for. The dial is two layers — the rose,
then the mark, which asks for the top with
[`core.StackAlign`](concepts/views.md#containers) because a `ZStack` centres
every layer that says nothing. The rose's inset grew from half a letter to a
whole one to make room; at heading zero the mark and the N deliberately
coincide, since an index pointing at N is what facing north looks like.

The mark used to be wrapped in a full-height column justifying its child to the
start — the escape a `ZStack` documented while it had no per-child alignment,
and this widget being its only consumer is what kept the prop out. The wrapper
cost a node per frame and restated the stack's height in a second place, so a
`Size` change had to be made twice or the mark drifted off the rim.

**`Heading` is a float, not a `core.Heading`.** The widget draws any bearing:
the direction of a route leg, a wind reading, the way a photograph was taken.
`hooks.UseHeading` supplies the sensor's; nothing else has to know about it.

**It announces once, as a sentence.** Four letters whose *positions* carry the
meaning are exactly what a screen reader cannot convey — read in tree order
the rose is "N W E S" whatever the bearing — so the whole widget speaks
("Heading 312 degrees, northwest") and every part inside it is hidden.
`AccessibilityLabel` overrides the sentence.

`Size` is the only geometry knob: the circle, its padding and the lettering
all derive from it. Letters are an eighth of the diameter with a 10px floor,
so a deliberately small compass stays readable.

## StaticMap

A map image of one point, which hands off to the platform's own maps app when
it is tapped.

```go
comps.StaticMap{
    Lat: 38.7223, Lng: -9.1393,
    Label:  "Lisbon Baptist Church",
    Marker: true,
}
```

**A picture and a hand-off, not a map engine.** The widget builds a URL, gives
it to [`core.Image`](concepts/views.md#leaves), and opens
[`core.OpenURL`](platforms/native.md) on a tap — so it works on all four
targets today with no renderer behind it. That is also the shape of what it
answers: "where is this" wants a picture that says *there* and then directions,
which the platform's maps app does better than any embedded view, with the
user's own home address and transport preferences. A live panning map is a node
type with MapKit, osmdroid and Leaflet behind it, and it is the thing to build
when an app needs to *interact* with a map.

**The provider is required, and there is no default.** `Provider` is one
function — `func(comps.StaticMapArea) string` — and a widget with none
renders its frame, no image, and `comps.ConcernNoMapProvider` in debug
mode. It used to default to `OSMStaticMap`, the OpenStreetMap community's
keyless service, on the argument that a widget nobody can render without first
buying something is a widget nobody evaluates. That service has been
discontinued and its host no longer resolves, and no keyless replacement
exists: every static-map service the OpenStreetMap wiki still lists takes a
key. So the choice is `GoogleStaticMap(key)` or a provider of your own, and the
widget says so instead of drawing a map of a host it cannot reach.

A provider sees values already defaulted and already clamped — no zero `Zoom`,
no 4000px `Width` — so every provider is spared the same four lines and none of
them can disagree about what a zero means. `StaticMap.Area()` is that
resolution, exported so a caller can ask what will be requested rather than
re-deriving it.

**`Width`/`Height` size the box; `Scale` sharpens the picture.** The first two
are logical pixels and the third is the device pixel ratio — `Scale: 2` leaves
the widget 320×180 on screen and asks the provider for 640×360 actual pixels.
Two numbers because one number cannot answer both questions, which is what made
every map before this field a 1x asset upscaled on a retina phone. Nothing
reads the ratio off the device: no renderer here reports screen metrics, so a
caller who has the number states it and a caller who says nothing gets 1x.
Whether it can be spent is the provider's business — Google's API has a `scale`
parameter (1 or 2, so a 3x device gets the 2x image); a provider without one
ignores the field, which is why this is not a multiply applied to `Width`
before the provider sees it.

**One hand-off URL for three platforms.** Nothing in this framework knows which
platform it is on — `core.OpenURL` promises only the portable part — so the
default `Handoff` is an https maps URL all three resolve, and which on both
phones reaches the installed app. An app that *does* know its platform returns
a `geo:` or `maps://` URL in one line, and that is what the `label` argument on
a `MapHandoff` is for: the cross-platform URL deliberately carries the
coordinates and not the name, because a name is a search and a search can land
on a different St Mary's in another country. Returning `""` says there is no
hand-off at all, and the widget renders a picture with no link role and no
callback.

**Tappable is a link; untappable is an image.** A tap leaves the app entirely,
which is what [`core.RoleLink`](concepts/styling-and-theming.md#accessibility)
says and `RoleButton` does not — a button does something *here*. With no
hand-off the widget is a `core.RoleImg`, a picture standing in for one fact,
the same argument `Compass` makes. Either way the image inside is hidden, so
the widget announces once instead of reading out a provider URL.

**Latitude clamps, longitude wraps.** Two rules because they are two geographic
facts. Latitude is held to the Web Mercator limit (±85.0511°), past which every
tile service returns an error image; longitude wraps, because 190°E is 170°W
and clamping it to 180 would move the point rather than name it.

**Lat 0, Lng 0 is the Gulf of Guinea.** There is no unset coordinate — a
`float64` pair has no third state — so a screen whose location has not loaded
yet renders a `Skeleton` rather than this widget.

## MapPanel

A live map over a set of points, opened at a view that contains all of them.

```go
comps.MapPanel{
    Pins: []comps.MapPin{
        {ID: "hall", Lat: 38.7223, Lng: -9.1393, Title: "The hall"},
        {ID: "annex", Lat: 38.7251, Lng: -9.1402, Title: "The annex"},
    },
    OnPinTap: func(id string) { open(id) },
    Caption:  comps.PlaceCount(2),
}
```

**One thing it adds to [`core.MapView`](concepts/views.md#leaves), and that is
its whole case.** The *opening region*. `core.MapView` takes a `Region` and
applies it only when it changes, which is correct and leaves the caller holding
a question — what region shows all my points? — whose answer is a bounding box,
a projection correction and a logarithm. `comps.FitRegion(pins)` is that
arithmetic, exported for a caller who wants it without the widget. Everything
else here is arrangement: the pins as keyed children, the empty state for a set
with nothing in it, an optional caption.

**The longitude spread is scaled by the cosine of the centre latitude.** A
degree of longitude narrows towards the poles and a degree of latitude does
not, so a fit computed from raw degrees is too tight in Reykjavík and about
right in Quito. Whichever span needs the wider view decides. A single pin has a
spread of zero and no logarithm, so the spread is *floored* rather than
branched on — one rule, and the floor is the view a single pin wants anyway.

**Two sets it does not fit, and says so rather than pretending.** A set spanning
more than half the globe is clamped to `MinFitZoom` (a view 180° wide), because
`core.Region` reads a zero `Zoom` as "unstated" and would substitute the
neighbourhood default. And a set straddling the antimeridian is measured the
long way round — Tokyo and Honolulu read as 298° apart rather than 62 — which
needs a circular mean and a different contract for the centre. Both are pinned
by tests, so fixing either is a change to a line rather than a surprise.

**An empty set has no region and renders no map.** `FitRegion` returns `false`,
and the panel draws its `Empty` state. The tempting answer — 0,0 — is a real
place in the Gulf of Guinea that a map will happily draw, which is the same
rule `StaticMap` states under "Lat 0, Lng 0".

**No region state, no recentre control, no echo.** The region is computed once
from the pins and handed over; where the reader takes the map from there is the
reader's. An app that wants to follow the map reaches for `core.MapView`
directly and holds the `Region` itself. Changing the *pins* does move the map,
because a new set is a new fitted region — right when the set is the subject,
and the reason a caller whose pins update every few seconds wants `core.MapView`
instead.

**No key, unlike `StaticMap`.** Each host's own engine draws the map with its
own tiles, so nothing here asks the app for a credential — the odd asymmetry
that the richer widget is the free one. The tile usage policy in
[`core.MapView`](concepts/views.md#leaves) still applies.

## CodeEditor

A programmer's editor: a monospace buffer with syntax colour, an optional
line-number gutter, and an optional toolbar of the editing commands a code
surface needs.

```go
ref := core.UseEditorRef(ctx)      // the toolbar's address

comps.CodeEditor{
    Value:       src.Get(),
    OnChange:    src.Set,
    Language:    "go",             // or Highlighter: a highlight.Highlighter
    LineNumbers: true,
    Toolbar:     ref,              // no ref, no toolbar
    Height:      "240px",
}
```

**Read-only, with no toolbar, is the display half** — a code block in a
document, a payload in a log viewer, the snippet a tutorial is teaching. That
is deliberately *not* `Disabled`: a disabled control is inert and greyed and is
skipped by assistive technology, while a read-only one is content the reader is
meant to select and copy. On the web the runtime lays a transparent
`<textarea>` over the rows so the caret is real, and that element leaves the tab
order when the editor is read-only — so a page of code blocks gains no tab stops.

**The ref is yours, and that is what keeps this widget hook-free.** A ref has to
be stable across passes, which means `core.UseEditorRef`, which is a hook —
and anything holding a hook slot must be rendered unconditionally on every
pass. That is a fine obligation for an editor with a toolbar and a bad one for a
code block, which is exactly the thing rendered inside an `if`, inside a loop,
inside a lesson body. So the toolbar *names* the ref and an editor without one
touches no hook. (`Accordion` and `DatePicker` are the two widgets that do own
slots; this is not one of them.)

**The highlighter runs on every pass, with no memoization.** `go/scanner` over a
thousand lines is well under a millisecond. The alternative is `hooks.UseMemo`,
which is the hook obligation above. A buffer big enough to change that arithmetic
wants a caller-supplied `Highlighter` that caches, not a widget that starts
consuming slots.

**`Language` picks the lexer by name and the comment marker with it.** `"go"`
and `"json"` in v1; anything else, including `""`, is uncoloured rather than an
error, so a screen whose editor mis-spells its language still renders. JSON has
no line comment, so the toolbar drops its comment button rather than showing a
permanently inert one.

**The zero `Scheme` is derived from the theme's own `Background`.** A code
surface has to be a deliberate colour — `highlight.Darcula` on a dark app,
`highlight.Light` on a light one — because no palette role means "the background
of a code listing", and an editor that took the app's Surface would be legible
by accident.

`OnSelectionChange` reports the caret as byte offsets into `Value`, parsed in
core from the `"start:end"` the hosts send. Bytes, because that is the one unit
all four hosts can agree on: Android and the browser count UTF-16 and iOS counts
`String.Index`.

### The node underneath

[`core.CodeEditor`](concepts/views.md#leaves) is the primitive, and it is a node
type rather than a composition for one reason: `core.Style` has no font family,
so a transparent `core.TextArea` in a `ZStack` over a `core.TextGrid` cannot be
pitch-matched to the grid under it from outside. The overlay has to be built by
something that owns both elements, which is the renderer.

Three rules every host implements, and they are what make a host-owned buffer
controllable from Go:

1. **Echo guard**, unchanged from `core.TextArea`. The buffer is the host's
   while focused and Go's otherwise, and Go's echo of the host's own last
   `onChange` never moves the caret. A value Go sends that the host never sent
   is a deliberate rewrite and lands even mid-typing.
2. **Decoration is advisory and per line.** A host applies row *N*'s styling
   only if that row's text equals the host's current line *N*. Go is a keystroke
   behind for a few milliseconds after every keypress, so the line being typed
   goes plain for one frame and no other line does. Never the other way round:
   decoration never rewrites the buffer.
3. **Commands are epoch-stamped props.** `core.RunEditorCommand(ref, cmd)` bumps
   a counter; the host acts once when the counter changes, on its own selection.
   A counter rather than a flag, because indenting twice is two commands with
   the same string and two identical prop maps produce no patch. Unlike a focus
   command, an editor that mounts under a standing epoch *adopts* it without
   running it — a command names a moment, and an editor that was not there
   missed it.

The commands in v1 are `core.EditIndent`, `core.EditOutdent`,
`core.EditCommentLine` and `core.EditSelectAll`. Not in v1: autocomplete,
folding, find-and-replace, and a host-side grammar.

### The lexers

`highlight` is a package of its own at the module root, beside `hooks` and
`permission`, because it is a model with no view:

```go
rows := highlight.Go().Rows(src, highlight.Darcula)   // []core.GridRow
core.TextGrid(rows, core.BackgroundColor(highlight.Darcula.Bg))
```

`highlight.Go()` is `go/scanner`, not a set of regexps — so a `//` inside a
string is a string without anything being told so, and the token set cannot
drift as the language grows. `highlight.JSON()` is a small hand lexer, written
to be tolerant rather than correct: a buffer mid-edit is invalid JSON most of
the time, and `encoding/json` would answer "invalid" for all of it.
`highlight.Plain()` colours nothing, and `highlight.ForLanguage(name)` is the
table.

Every lexer obeys one contract: `len(Rows(src, scheme)) ==
strings.Count(src, "\n") + 1`. The rows are line-for-line with the input,
because both consumers address them by line — `core.TextGrid` pairs them by
index, and the stale-line rule above compares row *N* with line *N*.

## RichTextEditor

Formatted text — bold, italics, headings, lists, quotes, links — whose value is
a document rather than a string.

```go
bar := comps.UseRichToolbar(ctx)

comps.RichTextEditor{
    Doc:         note.Get(),
    OnChange:    note.Set,
    Placeholder: "Write something…",
    Toolbar:     bar,
    MinHeight:   "160px",
}
```

**A read-only editor with no toolbar is the display half** — a comment, a note,
a description, the body of a card in a list. There is no separate
"RichTextView" node because there does not need to be one: the renderer's own
text engine draws the document either way, and `ReadOnly` is the difference
between reading it and writing it.

**The toolbar is the caller's, and that is what keeps the display half
hook-free.** A toolbar needs three things that must survive a render pass — the
`core.EditorRef` its buttons command, the last reported selection (so the bold
button can look pressed), and whether the link prompt is open. All three are
hooks. `UseRichToolbar(ctx)` is where they live, so an editor without one
touches nothing and can be rendered inside an `if`, inside a loop, inside a list
of comments. `CodeEditor` makes the same split for the same reason.

**Which buttons look pressed is the host's answer, not Go's.** Go owns the
document and the host owns the caret, so "is the text under the cursor bold" is
a question only the platform can answer. It comes back through
`OnRichSelectionChange` as a `core.RichSelection` and the toolbar draws itself
from the last one — widget-private state of the same kind `DatePicker`'s open
sheet is: presentation only, nothing an app would want to read.

`RichToolbarDefault` is the list of buttons; `bar.Items` is a copy of it, so
append to it, reorder it or replace it per screen. A `RichToolItem`'s `Command`
is a core `Edit*` constant or one of the two builders, plus one sentinel:
`comps.RichToolLink` opens the link prompt, because a URL has to be typed
before there is a command to send.

The strip carries no `core.RoleToolbar`, and that is deliberate: a widget in
this package may not declare a keyboard composite's container role, because it
would make a *nested* composite reachable by ordinary composition. Declare it on
your own box if you want the landmark.

### The document

[`richtext.Doc`](https://pkg.go.dev/github.com/rohanthewiz/grmob/richtext) is a
package at the module root, beside `hooks` and `highlight`, because it is a
model with no view:

```go
type Doc struct{ Blocks []Block }
type Block struct { Kind BlockKind; Runs []Run }
type Run struct {
    Text                                  string
    Bold, Italic, Underline, Strike, Code bool
    Link                                  string  // "" = not a link
}
```

Seven block kinds (paragraph, three headings, two lists, quote) plus a code
block, and six marks. No tables, no images, no nested lists, no colours — each
is a real feature with a real driver behind it and none has one yet.

**JSON is the wire and the storage.** `Doc.JSON()` is what crosses on every
keystroke and what `bytdb` takes as-is; short keys for the reason
`core.GridRun`'s are short. `Markdown()` / `FromMarkdown()` are the import and
export door, not the value: Markdown cannot represent a selection-preserving
edit, and making it the wire would put a Markdown parser in four hosts. `HTML()`
is what htmlout writes and what anything outside this system wants.

Markdown loses exactly two things, and both are pinned by tests rather than
discovered: an **empty paragraph** (a blank line is Markdown's block separator
and nothing else) and the **other marks on a code span** (a code span's content
is literal by definition). The JSON keeps both.

### The node underneath

[`core.RichTextEditor`](concepts/views.md#leaves) takes the document and a
`func(richtext.Doc)`, and shares two of `CodeEditor`'s three rules: the echo
guard (compared on the doc's JSON, since Go marshals with a fixed key order) and
the epoch-stamped command props. The *stale-line* rule is absent and does not
need to be there — here the doc **is** the styled buffer, so there is no second
fact to disagree with the first.

The commands are `core.EditBold`, `EditItalic`, `EditUnderline`, `EditStrike`,
`EditCode`, `EditUnlink`, `EditUndo`, `EditRedo`, plus `core.EditLink(url)` and
`core.EditBlock(kind)` — the two that carry an argument, which rides in the
command string because the channel is one prop and everything after the first
colon is the argument.

Each host does genuinely different work, and it is worth knowing which:

| Target | Construction |
|---|---|
| htmlout | `richtext.Doc.HTML()` inside the node's box, read-only. A snapshot has no caret. |
| WASM | A `contenteditable` `<div>`. Every command is a **pure transformation of the document** — the selection is only read and restored, never operated on, because a `Range` under `contenteditable` is the least predictable surface on the web. No `execCommand`. Paste is intercepted and re-done as a text insertion, so foreign markup dies at the edge. Undo is the runtime's own stack of Docs. |
| iOS | `UITextView`. Marks are attributes edited straight into the `textStorage`, which preserves the caret and gives "press bold, then type" free through `typingAttributes`; block kinds take the long way round (read out, transform, rebuild, restore), because a prefix and an indent have no in-place spelling. |
| Android | `EditText` + `Spannable` through `AndroidView` — the one deliberate reach past Compose in the renderer. `Spannable` has had a span type for every mark and every paragraph treatment for a decade; building block structure into one `AnnotatedString` rebuilt per keystroke is the riskiest thing this design could ask for, and the classic-view route removes it. |

## Writing your own

The package doc (`comps/doc.go`) is the reference for the idiom. In
short:

1. A struct with named fields; `core.View`-typed fields for slots.
2. `Render(ctx)` builds on core containers/widgets and returns their node.
3. Read the theme; accept `Style []core.StyleProp` for overrides.
4. If the widget owns state, document its hook obligations (see Accordion).
5. Give it a focused test — and if it's extracted from app code, pin the
   rendered output against the original with `htmlout.ExportHTML`.
