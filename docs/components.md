# Widget Library — the `components` package

`components` is GrMob's higher-level widget library, built **entirely on the
public core API** — a deliberate dogfooding discipline: if a widget can't be
built out here, that is a gap in core's primitives, not a reason to reach
inside.

```go
import "github.com/rohanthewiz/grmob/components"
```

## The struct-widget idiom

Every widget is a struct implementing `core.View`, configured through named
fields:

```go
components.Card{
    Title:  "Account",
    Body:   balanceSummary,
    Footer: components.Badge{Text: "verified"},
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
  └─ Scroll            (only when Scroll is true; KeyboardAware lands here)
       └─ Column       ← Gap / Fill / Style land here
                         (and KeyboardAware, when there is no Scroll)
            ├─ Children[0]
            └─ …
```

```go
// The zero value: SafeArea(Column(children...)), nothing else.
components.Screen{
    Children: []core.View{header, body, composer},
}

// A screen that scrolls as a whole.
components.Screen{
    Scroll:   true,
    Children: []core.View{hero, section1, section2},
}

// A screen whose list fills the space and pushes a footer down.
components.Screen{
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
(`fintechapp`, `signup`), and every screen the `tutorial`'s navigator pushes
(`home`, `lesson_screen`, chapter 6) scrolls as a whole. The rest do not:
`chat` scrolls its message list, `todoapp` scrolls a virtualized `core.List`,
and `mobileapp` and `layout` are short enough to need neither.

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
components.Screen{Scroll: true, KeyboardAware: true, Children: fields}

// A chat: no scroll here, so the column lifts and the composer rides up.
components.Screen{KeyboardAware: true, Children: []core.View{header, thread, composer}}
```

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
components.Screen{Children: []core.View{banner, body}}
```

That is the same contract behind [`core.MaybeProp`](concepts/views.md) — no
node, no flex slot, no stray `Gap` — where a `core.If` would leave an empty
`Fragment` for the column to space against.

## Button

A themed action button with **two orthogonal color axes** and no per-call hex.

```go
components.Button{Label: "Save",   OnTap: save}                                   // theme Button base
components.Button{Label: "Delete", OnTap: rm,   Variant: components.VariantError}
components.Button{Label: "Cancel", OnTap: back, Emphasis: components.EmphasisOutlined}
components.Button{Label: "Skip",   OnTap: skip, Emphasis: components.EmphasisGhost}
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
components.Button{
    Label: "Recharge", Emphasis: components.EmphasisOutlined,
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
components.InputRow{
    Value:       draft.Get(),
    Placeholder: "What needs doing?",
    OnChange:    func(v string) { draft.Set(v) },
    OnSubmit:    addTodo,
    Button:      components.Button{Label: "Add"},
}

// A search field with no button: the return key commits it.
components.InputRow{
    Value:       query.Get(),
    Placeholder: "Search",
    OnChange:    func(v string) { query.Set(v) },
    OnSubmit:    runSearch,
}

// A docked composer, with the bar treatment this widget has no opinion about.
components.InputRow{
    Value: draft.Get(), Placeholder: "Mensagem…",
    OnChange: func(v string) { draft.Set(v) },
    OnSubmit: send,
    Button:   components.Button{Label: "Enviar"},
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
components.Card{
    Title: "Recent activity",        // simple path: themed bold subtitle
    Body:  activityList,
    Footer: core.Text("Updated 2m ago"),
}

components.Card{
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
components.ListRow{
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
        components.ListRow{Title: "Weekly",  Selectable: true, Selected: plan == weekly},
        components.ListRow{Title: "Monthly", Selectable: true, Selected: plan == monthly},
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
        components.ListRow{Title: "Gospels", NestingLevel: 2},
        components.ListRow{Title: "Matthew", NestingLevel: 3, Style: indent},
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

## Badge

A small **non-interactive** status pill — a count, a "verified" mark, a
state label. Defaults: theme Primary background, theme Background ink,
caption-sized, stadium-shaped.

```go
components.Badge{Text: "3"}
components.Badge{Text: "beta", Color: "#7B1FA2"}
```

### Variants

`Variant` names what the badge *means* and takes its colors from the
palette's status roles, so a status pill carries no literal hex:

```go
components.Badge{Text: "Paid",     Variant: components.VariantSuccess}
components.Badge{Text: "Expiring", Variant: components.VariantWarning}
components.Badge{Text: "Failed",   Variant: components.VariantError}
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
components.Chip{
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
components.Chip{Label: "$25", Prominence: components.ProminenceLoud,
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
whole `Chip`, `Segment: components.Chip{Prominence: components.ProminenceLoud}`
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
components.SegmentedControl{
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
components.SegmentedControl{
    Labels:    filterLabels,
    Selected:  active,
    OnSelect:  onSelect,
    KeyPrefix: "filter-",
    Segment: components.Chip{
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
components.SegmentedControl{
    Labels:   []string{"Sermons", "Articles"},
    Selected: tab.Get(), OnSelect: func(i int) { tab.Set(i) },
    Style:   []core.StyleProp{core.AccessibilityRole(core.RoleTabList)},
    Segment: components.Chip{Style: []core.StyleProp{core.AccessibilityRole(core.RoleTab)}},
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
components.SegmentedControl{
    Labels:   []string{"Sermons", "Articles"},
    Selected: tab.Get(), OnSelect: func(i int) { tab.Set(i) },
    Style:   []core.StyleProp{core.AccessibilityRole(core.RoleTabList)},
    Segment: components.Chip{Style: []core.StyleProp{
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

## Separator

The hairline rule between rows and between sections. The zero value is the
common case:

```go
components.Separator{}
components.Separator{Inset: 56}          // starts under the text, not the avatar
components.Separator{Thickness: 0.5}     // sub-pixel hairline on a 2x display
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
components.Avatar{Src: user.PhotoURL, Name: user.Name}  // image, labelled
components.Avatar{Name: "Ada Lovelace"}                 // "AL" on a disc
components.Avatar{Name: "Ada Lovelace", Size: 64}
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
components.ProgressBar{Value: 0.45, AccessibilityLabel: "Upload"}
components.ProgressBar{Value: done / total, Thickness: 10, Color: "#34C759"}
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
components.FormField{
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
components.FormField{
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
components.FormField{
    Error: form.Error("terms"),
    Input: components.ListRow{
        Leading: form.Checkbox("terms"),
        Title:   "I accept the terms of service",
    },
}
```

## Accordion

A collapsible section — tappable chevron header, content shown while
expanded.

```go
components.Accordion{
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

The arrangement above is `components.disclosure`, and the collapsible
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
components.Tabs{
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
components.DataTable[Entry]{
    Columns: []components.Column[Entry]{
        {Title: "Title", Weight: 2, Text: title, Less: byTitle},
        {Title: "Speaker", Narrow: true, Text: speaker},
    },
    Rows: entries, Key: entryKey,
    Sort: sortBy.Get(), OnSort: func(s components.Sort) { sortBy.Set(&s) },
    Pagination: &components.Pagination{Page: page.Get(), PageSize: 20, OnChange: page.Set},
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
components.GroupedList[Entry]{
    Items:         pager.Items,
    GroupBy:       byMonth,
    StickyHeaders: true,
    OnEndReached:  pager.LoadNext,   // the scroll
    Footer: components.LoadMore{     // and the tap, and the states
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

components.GroupedList[Entry]{
    Items: entries, GroupBy: byMonth,
    Collapse: components.Collapse{
        IsCollapsed: func(g components.Group) bool { return shut.Get()[g.Key] },
        OnToggle: func(g components.Group) {
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
Header: func(g components.Group) core.View {
    return core.Row(
        components.CollapseBand{
            Collapse: shut, Group: g,
            // Your chrome, on the control — not on the Row, where a press
            // would do nothing. This is the same move the default band makes.
            ControlStyle: []core.StyleProp{
                core.PaddingLeft(16), core.PaddingRight(8)},
        },
        components.Avatar{Name: leader[g.Key]},
        components.Badge{Text: strconv.Itoa(g.Count)},
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
components.AppBar{
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

## Banner

The inline strip that tells the user something about the screen they are on:
a failed refresh over content that is still good, an offline notice, a
"Reconnecting…".

```go
components.Banner{
    Text:        "Could not refresh. Showing a saved copy.",
    Variant:     components.VariantWarning,
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

## EmptyState

The centered placeholder for content a screen does not have — and for the
other two moments with the same shape:

```go
empty   components.EmptyState{Glyph: "📭", Title: "No messages yet"}
busy    components.EmptyState{Title: "Loading sermons…"}
failed  components.EmptyState{Glyph: "☁", Title: "Could not reach the server.",
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
tail  components.EmptyState{Hint: "Loading more…",
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

components.SearchField{
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
components.ChipStrip{Chips: []components.Chip{
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
components.ChipStrip{Scrollable: true, Chips: years}
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
components.Skeleton{}                                        // one line
components.Skeleton{Lines: 3}                                // a paragraph
components.Skeleton{Width: "44px", Height: 44, Radius: 999}  // an avatar
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

## StatTile

One figure with its name and, optionally, its movement.

```go
core.Row(core.Gap(12),
    components.StatTile{Label: "Attendance", Value: "412", Fill: true,
        Delta: "+18 vs last week", DeltaVariant: components.VariantSuccess},
    components.StatTile{Label: "Giving", Value: "MZN 42,750", Fill: true},
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
case !h.Received:  return components.Skeleton{}            // no reading yet
case !h.Available: return components.EmptyState{Hint: h.Error}
default:           return components.Compass{Heading: h.Magnetic, ShowDegrees: true}
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

## Writing your own

The package doc (`components/doc.go`) is the reference for the idiom. In
short:

1. A struct with named fields; `core.View`-typed fields for slots.
2. `Render(ctx)` builds on core containers/widgets and returns their node.
3. Read the theme; accept `Style []core.StyleProp` for overrides.
4. If the widget owns state, document its hook obligations (see Accordion).
5. Give it a focused test — and if it's extracted from app code, pin the
   rendered output against the original with `htmlout.ExportHTML`.
