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

// With Floating, the content becomes the base layer of a ZStack that grows
// to fill the safe area, and Floating is placed over its bottom-end corner.
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
| `Floating` | drawn over the content at its bottom-end corner, above the `Footer`; the slot for a `comps.FAB` |

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
A region that sizes to its content is not that case: a `CodeEditor` with no
`Height` in a scrolled `Screen` lays out at its full height on every host and
pans only sideways. (On Compose that took a guard — a bare `verticalScroll`
under an unbounded height throws, and every tutorial lesson crashed on it until
the renderer capped the viewport at the content; see `core.Scroll`. SwiftUI
sizes a nested scroll view to its content without one, which a simulator pass
over lessons 4.3 and 4.6 confirmed.)
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

**`Floating` is the slot for a `FAB`.** The content (the `Scroll`, or the
column when there is none) becomes the base layer of a `core.ZStack` that
grows to fill the safe area, and `Floating` is the top layer, placed
`core.StackAlignBottomEnd` with a `Spacing.LG` margin. The `Footer` stays
below the stack, so the button floats above a `BottomBar` rather than on it.

```go
comps.Screen{
    Scroll:   true,
    Children: []core.View{notes},
    Floating: comps.FAB{Icon: "+", AccessibilityLabel: "New note", OnTap: create},
    Footer:   comps.BottomBar{Items: tabs, Selected: tab.Get()},
}
```

The base layer states `Width("100%")` and `Height("100%")`, because a stack
centres and hugs any layer that says nothing: without them the content would
sit in the middle of the screen as wide as its widest row. The stack, not the
column, grows, since a layer of a stack has no axis to grow along. The
scaffold does not pad the content for the button, because it cannot see a
list's own inset; give the last row room. A nil `Floating` builds no stack.

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

**`FocusRef` names the button for `core.Focus`.** A handler elsewhere can move
focus onto it, as `Drawer`'s opener does with the drawer's ✕ and its
`OnDismiss` does with the ☰. Nil names nothing.

## CopyButton

A button that puts a fixed string on the clipboard and confirms it.

```go
comps.CopyButton{
    Text:               inviteCode,
    Label:              "Copy code",
    AccessibilityLabel: "Copy invite code",
    Emphasis:           comps.EmphasisOutlined,
}
// tap → core.WriteClipboard(Text), core.Haptic(HapticLight), core.ShowToast("Copied")
```

- **The toast confirms, not the caption.** Flipping the caption to "Copied ✓"
  and back needs a timer, a timer is a hook, and a hook would forbid rendering
  the button inside an `if`. CopyButton takes no hook slot; every code block in
  the tutorial carries one for that reason.
- **No `Text`, no copy.** An empty `Text` disables the button — a link still
  loading is a real state — and a tap that arrives anyway writes nothing:
  `WriteClipboard("")` would clear the clipboard.
- Several on one screen would all be "Copy", so `AccessibilityLabel` says what
  is copied. The copied text is not read out.
- Over a `CodeEditor` in a `ZStack`, give the button `core.ZIndex(1)`: the web
  draws the editor as a positioned `<pre>`, which paints over a later sibling
  that is not positioned.

## Link

A line of text that goes somewhere.

```go
comps.Link{Text: "Privacy policy", URL: "https://example.com/privacy"}
comps.Link{Text: "Forgot password?", OnTap: showReset}
```

- `RoleLink`, not a button: it leaves the screen, and a reader deciding
  whether to follow it needs to know that.
- `OnTap` wins; otherwise a tap is `core.OpenURL(URL)`. With neither, it reports
  `ConcernLinkInert`.
- Drawn in `Primary`'s on-light tone, hugging its text (`AlignSelf(start)`) so
  the empty width beside it is not a target.
- **Not underlined** on its own line, where being a line of link colour is what
  says it is a link. **Inline**, inside a sentence, use `Link.Span(ctx)`: a run
  of a `core.Paragraph`, in the same colour, underlined (there the colour is the
  only other signal), with the same tap.

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

comps.CheckboxRow{Title: "Also delete attachments",
    Checked: purge.Get(), OnToggle: purge.Set}
```

Both are a `ListRow` with the trailing slot fixed to `core.Switch` or
`core.Checkbox` and `OnTap` wired to the same setter. Pick the switch for a
setting that takes effect on the tap and the checkbox for a value a form
collects later; see `core.Switch` for why those are different controls.

**Which edge the checkbox is on.** `CheckboxRow` is for a row that *is an
option* — a setting, a filter, "also delete attachments" — and puts the
checkbox trailing, in the same column as a `SwitchRow`'s switch. A row that is
*the thing being marked* — a task done, a sentence agreed to — leads with the
checkbox instead, because the mark is read before the content; build that with
`ListRow{Leading: core.Checkbox(…)}`, as the todo list above and the terms box
in the forms section do.

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

## SelectRow

The settings row for a value chosen from a short list: the title on the
leading edge, the current choice on the trailing edge, and a sheet of the
alternatives behind a tap on the row.

```go
comps.SelectRow{
    Title: "Theme",
    Options: []core.SelectOption{
        core.Option("system", "System"),
        core.Option("light", "Light"),
        core.Option("dark", "Dark"),
    },
    Value:    theme.Get(),
    OnChange: theme.Set,
}
```

```
┌──────────────────────────────────┐        ┌─────────────────────┐
│ Theme                    Dark ›  │  tap → │ Theme               │
└──────────────────────────────────┘        │   System            │
                                            │   Light             │
                                            │ ✓ Dark              │
                                            │   Cancel            │
                                            └─────────────────────┘
```

**Why a sheet and not `core.Select` in the trailing slot.** `core.Select` is
the platform's own picker and is right inside a `FormField`, beside text
inputs. In a *row* it is wrong twice over: a settings list is tapped anywhere
along its width, and a picker in the trailing slot is only as wide as its
longest label; and on the web a click on the control bubbles to the row, so a
row holding both would open two things. That is the same double dispatch
`SwitchRow` works through — there the two handlers converge on one value and
the guard drops the second, but two *openings* have nothing to converge on.

**The row owns one piece of state** — whether the sheet is open, exactly as
`DatePicker` does. So the hook rule applies in full: render a `SelectRow`
unconditionally, in a stable position, every pass. A loop over a list of
settings is fine; a row that appears only when another switch is on is not,
and wants `core.When` around a whole screen section instead.

Other notes:

- `Options` is `[]core.SelectOption`, the type `core.Select` and
  `SearchableSelect` take, so a list moves between the three unchanged. An
  empty `Label` falls back to `Value`; `Disabled` and `GroupDisabled` grey an
  action and drop its taps, read per option.
- `Group` is **not** drawn: a sheet action is a button with one line and has
  no section construct. A list long enough to want headings wants
  `core.Select` (an `<optgroup>`) or `SearchableSelect` (the row's subtitle).
- Picking the value already chosen closes the sheet and calls nothing —
  `OnChange` is a setter, like `SwitchRow.OnToggle`.
- The row takes `RoleButton` and `PopupDialog`, as `DatePicker`'s trigger
  does, and is named by its own text ("Theme, Dark"). The chosen action is
  marked with `SheetAction.Checked`.
- `Value` matching no option shows `Placeholder` and, in debug builds,
  reports `comps.ConcernSelectRowValueNotAnOption` — the mismatch looks
  exactly like an unset row on screen. An empty `Value` is quiet.
- `SheetTitle` overrides the sheet's heading ("Sort" → "Sort by");
  `CancelLabel` captions its way out.

## SliderRow

The settings row for a number in a range: the title and the current reading
on one line, the track across the width under them.

```go
comps.SliderRow{
    Title: "Brightness", Min: 0, Max: 100, Step: 1,
    Value:    level.Get(),
    OnChange: level.Set,
    Format:   func(v float64) string { return fmt.Sprintf("%.0f%%", v) },
}
```

```
┌────────────────────────────────────────────┐
│ 🔆  Brightness                        72%  │
│     ▬▬▬▬▬▬▬▬▬▬▬▬▬▬●─────────               │
└────────────────────────────────────────────┘
```

The title, the reading and the track share the row's growing middle column,
which is what aligns them: the track starts where the title starts whatever
the leading icon's width, with no arithmetic against the row's own padding.

**`OnChange` fires once, when the drag ends.** `core.Slider` reports twice
over — continuously under the finger and once more when it lifts — and this
row wires `OnChange` to the second. A `Set` on any `core.State` requests a
render of the whole tree, so a row feeding its caller on every tick would put
a full render pass between each pixel of a drag, for a value nobody has
finished choosing; and what is downstream of a settings slider is usually a
write to disk, a device call or a request.

The thumb still follows the finger — every renderer draws the dragged
position and Go's value otherwise — so only the number beside the title lags.
A caller who wants it live opts into the cost through `OnDrag`, and holds the
draft itself:

```go
draft := core.NewState(ctx, -1.0) // -1: not dragging
shown := level.Get()
if draft.Get() >= 0 {
    shown = draft.Get()
}
comps.SliderRow{
    Title: "Brightness", Max: 100, Value: shown,
    OnDrag:   func(v float64) { draft.Set(v) },
    OnChange: func(v float64) { draft.Set(-1); level.Set(v) },
}
```

The draft stays the caller's on purpose: holding it in the widget would make
`SliderRow` hook-owning — unconditional, stable position, every pass — and
would charge every row the render-per-tick it was written to avoid. As
written the row takes no hooks and is free to be conditional.

Other notes:

- The row carries **no** `OnTap` and no role. A switch has one other state a
  row-sized target can reach; a slider's "other value" is the one under the
  tap, and Go sees no coordinates. The slider is the control a reader is
  looking for, and it is named by `Title` and hinted by `Subtitle`.
- `Max` at or below `Min` becomes `Min..Min+1` with the value pinned to the
  start — `core.Slider`'s own rule, repeated here so the reading and the
  thumb cannot disagree.
- `Format` nil writes the number at the precision `Step` is written at (0.5
  gives one decimal, 0.25 two), or, with no step, at the precision the span
  suggests: none over 10, one over 1, two at or below. Returning `""` draws
  no reading at all.
- `Disabled` greys the track and registers neither callback.

## KeyValueList

The label-and-value table of an order summary, a profile or an about screen.

```go
comps.KeyValueList{
    Label:    "Order details",
    Dividers: true,
    Rows: []comps.KeyValue{
        {Key: "Placed", Value: "14 Mar 2026"},
        {Key: "Total",  Value: "$42.10"},
    },
}
// Placed                 14 Mar 2026
// ──────────────────────────────────
// Total                       $42.10
```

- Each row is a [`ListRow`](#listrow): the key is the leading slot, the value
  the trailing one, and the empty middle grows between them, so the value is
  pinned to the edge and nothing new solves layout. The key is pinned at its
  width (`FlexShrink(0)`, ListRow's own advice for text there), so a long value
  wraps and a key never does.
- The key is in the body ink and the value in the secondary one — the iOS
  "value" cell. A long value wraps against the trailing edge.
- The column is a `RoleList` of listitems, which the widget can claim because
  it owns both halves. Each row is named "Key, Value", because the widget knows
  both strings and a row is then one stop for a reader rather than two
  unrelated ones.
- `Dividers` puts a hidden `Separator` between rows, never above the first or
  below the last.

## BulletList

Short points behind a marker, bulleted or numbered.

```go
comps.BulletList{Items: []string{"Free delivery", "Cancel any time"}}
comps.BulletList{Items: steps, Ordered: true}   // 1. 2. 3. …; Start moves the first
```

- The marker is pinned and the text grows, so a long item wraps under its own
  first word (a hanging indent). Ordered markers are right-aligned in one
  column sized for the widest number, so "9." and "10." end at the same x.
- A `RoleList` of listitems, each named by its text; the marker is hidden,
  because a screen reader states the position itself.
- Not `core.List`: a bullet list is short, and static children need no keys.
  The tutorial's key points are built with it.

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

`Variant` is a package-level type, not Badge's own, so [Banner](#banner) —
the inline alert — and Button resolve the same four roles the same way. `Variant.Color(theme)` and
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

## RadioGroup

A vertical set of mutually exclusive options, each a row with a ring on the
leading edge.

```go
comps.RadioGroup{
    Label: "Shipping",
    Options: []comps.RadioOption{
        {Value: "std", Label: "Standard", Subtitle: "3–5 days"},
        {Value: "exp", Label: "Express", Subtitle: "Next day"},
    },
    Value:    ship.Get(),
    OnChange: ship.Set,
}
```

**Which choice widget.**

| Widget | Use it for |
|---|---|
| `core.Select` | a compact field whose options stay hidden until opened |
| `SegmentedControl` | two to four short labels side by side |
| `RadioGroup` | every option visible, stacked, each with room for a subtitle |

**One target per row.** The ring is drawn, not a platform control, so the
row's tap is the only handler. There is no second dispatch to guard against,
unlike `SwitchRow`. `OnChange` fires only when the tapped option differs from
`Value` and is enabled.

**Radio roles.** The group is a `RoleRadioGroup` and each row a `RoleRadio`.
The choice is `AccessibilitySelected`, which the web writes as `aria-checked`
on a radio, so a reader hears "radio button, checked". The browser runtime
supplies the radio group keyboard: one tab stop on the checked radio, and the
arrow keys move the check. `OnChange` therefore fires as a user arrows through
the options. On Android the group and rows map to `selectableGroup()` and
`Role.RadioButton`; iOS has no radio trait and announces the checked row as
selected.

Other notes:

- Set `Label`. A radio group with no name is announced as a bare run of radio
  buttons.
- No row takes a background tint, because the ring already shows the choice.
- `Disabled` on the group or on one `RadioOption` greys it and drops its taps.
- A nil `OnChange` draws a display-only group with no handlers.

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

**`Value` is a float.** It rounds to whole glyphs unless `Halves` is set, so a
caller storing an average never changed type when half-glyphs arrived.
`OnChange` reports whole positions because a tap lands on one glyph. Tapping
the glyph that is already the value does nothing.

**`Halves` draws, it does not type.** With `Halves` the value rounds to the
nearest half ("3.5 of 5") and every star is a small `Canvas`. The half is the
star's exact left-half polygon, not a clipped "★": SwiftUI truncates a Text
squeezed below its width to "…" rather than letting it be cut. `Glyph` and
`EmptyGlyph` do not apply to a `Halves` rating. Without `Halves` the tree is
unchanged.

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

## LabeledSeparator

The rule with a word in it — the "or" between two ways to sign in.

```go
comps.LabeledSeparator{Label: "or"}
// ──────────────── or ────────────────
```

Two [`Separator`](#separator)s that grow equally, with the label between them,
so the word stays centred at any width and the rules are the theme's own
hairline. The rules are hidden; the label is read, because "or" between two
sign-in buttons is part of what the screen says. The label never shrinks —
the rules give way first. An empty `Label` draws one unbroken rule.

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

## AvatarStack

The overlapping row of faces: who is in a thread, who is going.

```go
comps.AvatarStack{Avatars: attendees, Max: 4}
// (AL)(GH)(KJ)(+3)      six people, Max 4
```

**A `ZStack`, because a `Row` cannot overlap.** A negative margin or gap is not
portable, so every face is a stack layer placed `core.StackAlignStart` and
pushed right by a `MarginLeft` one step larger than the last — the same
margin-on-a-layer that holds [`FAB`](#fab) off its corner. The stack's box is
pinned: width `ring + step·(n-1)`, height `ring`.

**The ring is a layer, not a border.** Each face sits on a disc of the theme's
`Background`, drawn as its own layer before the face. When it was built, a
border was sized differently across targets: inside the box on both natives,
and outside it in the static export, which was content-box then. The overlap
was off by the difference. Exports are border-box now. The layer stays,
because a plain `Box` with a size and a fill needs no target to agree about
box models.
`RingColor` matches it to a `Card` or `Surface` panel; `RingWidth: -1` drops it.

Faces overlap by 20% of `Size` by default (`Overlap`): at 25–30% the next disc
cut into a two-letter pair of initials.

**`Max` counts the surplus disc**, so `Max: 4` is four discs wide however long
the list: six people draw three faces and "+3". Later faces overlap earlier
ones, and the "+N" disc is drawn last so nothing covers its count.

**One picture, one name.** The stack is `RoleImg` named "Ada Lovelace, Grace
Hopper and 3 others", with every face hidden behind it. The count takes in the
surplus and any unnamed face. `Label` replaces the sentence ("6 attendees", or
another language). An empty stack is a hidden, sizeless `Box`.

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

## StepIndicator

The "step 2 of 4" header of a multi-screen flow. Done steps are ticked, the
current step is filled, and the rest are outlined, joined by short rules.

```go
comps.StepIndicator{
    Steps:   []string{"Account", "Address", "Payment", "Review"},
    Current: step.Get(),
    OnTap:   step.Set, // done steps only
}
```

**Only done steps are tappable.** Going back to fix an address is normal.
Jumping ahead past a step that has not been validated is not, so the widget
never offers it, and no caller has to guard against it. The current step is
not tappable either, because tapping it would do nothing.

**A long flow scrolls.** The strip is a `core.Scroll` with `core.Horizontal`.
Collapsing to "2 / 4" past some number of steps would need the screen width,
which Go does not have, so any threshold would be wrong on some screen.

Other notes:

- The strip is `RoleNavigation` when `OnTap` is set and `RoleGroup` when it is
  not. Its name states the position, "Step 2 of 4: Address", and `Label`
  prefixes it ("Checkout, step 2 of 4: Address").
- Each step is named "Step 1: Account, done", "Step 2: Address" or
  "Step 3: Payment", and the current step states `core.CurrentStep`
  (`aria-current="step"`, selected on the natives). ", done" stays a suffix
  because no platform has a completed state. A tappable step is a button. The circles and rules are
  hidden from assistive technology.
- `Current` is clamped into the steps, so an index past the end shows every
  step before the last as done.
- Done steps and the rules after them use the Success colour; the current step
  uses Primary. Both discs pick a contrasting ink.

## Wizard

A multi-step flow on one screen: a `StepIndicator`, the current step's title
and body, and a Back / Next footer.

```go
name := core.NewState(ctx, "")   // every step's state, above the wizard
step := core.NewState(ctx, 0)

comps.Wizard{
    Label: "Order",
    Steps: []comps.WizardStep{
        {Title: "Your name", Body: nameField(name), Blocked: name.Get() == ""},
        {Title: "Gift note", Body: noteField(note), Optional: true, Blocked: note.Get() == ""},
        {Title: "Review",    Body: summary(name, note)},
    },
    Current:  step.Get(),
    OnChange: step.Set,
    OnFinish: placeOrder,
}
```

**A step's `Body` must not own hooks.** Only the current step's body is
rendered, so a hook inside one is a conditional hook, and every hook after it
shifts slots when the step changes. Hold each step's state above the wizard and
hand the bodies values. That is also what keeps a step's input alive across
Back and Next. A body that cannot avoid hooks can take a `ctx.Scope` of its
own. The `Wizard` itself holds no hook, so it may be rendered conditionally.

**The moves it offers:**

| Button | Does | When |
|---|---|---|
| Back | `OnChange(Current-1)` | absent on the first step, not disabled |
| Next | `OnChange(Current+1)` | disabled while the step is `Blocked` |
| Skip | `OnChange(Current+1)` | replaces Next while an `Optional` step is `Blocked` |
| Finish | `OnFinish()` | the last step; disabled when `OnFinish` is nil |
| a done step | `OnChange(i)` | in the indicator; later steps are never tappable |

`Blocked` is the field, not `CanAdvance`, so that the zero value advances: a
step that is only read needs nothing set.

**The footer can be lifted out.** It is drawn at the end of the wizard's own
column. For a form long enough to scroll, set `DetachFooter` and place
`Footer()` yourself:

```go
w := comps.Wizard{…, DetachFooter: true}
comps.Screen{Scroll: true, KeyboardAware: true,
    Children: []core.View{w}, Footer: w.Footer()}
```

Other notes:

- The column is a `RoleGroup` named by `Label`. Under the indicator is a
  `RoleStatus` line, "Step 2 of 3, optional", so a step change is announced
  where live regions are. The title is a level-2 heading.
- Focus does not move to the heading on a step change: `core.Focus` reaches
  fields and Buttons, and no target focuses a Text. VoiceOver announces no live
  region, so on iOS a step change is silent until the reader moves.
- The body is keyed by step index, so a change of step replaces it and does not
  morph one form into the next.
- `NextLabel`, `BackLabel`, `FinishLabel`, `SkipLabel` and `PositionLabel` put
  every word in the app's language.
- Debug concerns: `ConcernWizardNoSteps`, and `ConcernWizardInert` for more
  than one step with no `OnChange`.

## TreeView

An indented, expandable hierarchy: a file browser, an outline, a category
picker.

```go
open := core.NewState(ctx, map[string]bool{"docs": true})

comps.TreeView{
    Label:    "Project files",
    Nodes:    files, // []comps.TreeNode{ID, Label, Leading, Children, Branch}
    Expanded: open.Get(),
    OnToggle: func(id string) { // copy, flip, Set
        next := maps.Clone(open.Get())
        next[id] = !next[id]
        open.Set(next)
    },
    Selected: chosen.Get(),
    OnSelect: chosen.Set,
}
```

**The caller owns what is open.** `Expanded` is your `map[string]bool` and
`OnToggle` only reports an ID. Open state is navigation state: "collapse all"
is an empty map, and revealing a search hit is a map of its ancestors. The
widget holds no hook.

**A branch toggles, a leaf selects.** One row is one target. A picker whose
categories can themselves be chosen gives the branch a first child that stands
for it ("All of Fiction"). A node with no `Children` is a leaf unless it sets
`Branch`: an empty folder, or one whose children load when it first opens.

**IDs are unique in the whole tree,** not only among siblings, because the map
has one key space. Paths work. A duplicate reports
`ConcernTreeViewDuplicateID`, even inside a shut branch.

Other notes:

- A shut branch's children are not rendered, so a large tree costs what its
  open part costs.
- The structure is nested lists: `RoleList`, `RoleListItem` with
  `core.AccessibilityNestingLevel`, and a button inside each item. A branch's
  button states expanded or collapsed; the chosen leaf states
  `core.CurrentTrue`. ARIA's `tree` role promises arrow-key navigation that no
  target supplies, so the widget does not claim it; every row is a Tab stop.
  The roles are not in the API, so they can change later without one.
- The nesting level reaches the web only. Neither native has a depth property,
  so on a phone the indent is what says depth.
- The indent is a spacer leading a Row, not a left padding, so it follows the
  reading direction under RTL. `Indent` is in whole points and defaults to the
  theme's `Spacing.LG`.
- A leaf keeps the 20pt chevron column empty, so labels line up. `Leading` is
  decoration and is hidden from accessibility.
- With no `OnSelect`, leaves are plain text. Branches with no `OnToggle` report
  `ConcernTreeViewInert`.

## Timeline

A vertical list of events joined by a line down the leading edge, a dot per
event: order tracking, an activity feed, a changelog.

```go
comps.Timeline{
    Label: "Order history",
    Events: []comps.TimelineEvent{
        {Time: "09:12", Title: "Order placed"},
        {Time: "11:40", Title: "Packed", Subtitle: "Warehouse 3"},
        {Time: "14:05", Title: "Out for delivery", Variant: comps.VariantSuccess},
    },
}
```

**Each row draws its own piece of the line.** No renderer draws a line across
siblings. Each event is a Row with `AlignItems` stretch, so its leading rail
is as tall as the event's text. The rail has three parts:

| Part | Size | Paints |
|---|---|---|
| top segment | fixed, centres the dot on the first line of text | the line, except on the first event |
| dot | 12 points | the event's `Variant` colour, Primary by default |
| bottom segment | grows to the row's height | the line, except on the last event |

The space below an event is bottom padding inside its body, not a gap between
rows. The bottom segment runs through it and meets the next row's top segment
with no break.

**Not built on `ListRow`.** `ListRow` carries the theme's row padding outside
its slots, which would break the line between every pair of rows. The rows
here are plain Rows with no padding.

Other notes:

- `Time` is a caption above the title. The dot lines up with whichever comes
  first.
- `Content` puts any view under the text: a thumbnail, a quote, a button.
- The Timeline is a `RoleList` named by `Label`, and each event is a
  `RoleListItem`. The rail is hidden from assistive technology.
- Row stretch on Android and iOS comes from reading their renderers. It has not
  been checked on a device.

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

## PasswordField

A password input with a reveal toggle trailing it.

```go
comps.FormField{
    Label: "Password",
    Error: form.Error("password"),
    Input: comps.PasswordField{
        Value:    pw.Get(),
        OnChange: pw.Set,
        Label:    "Password",
    },
}
// [ ••••••••••            ]  Show
```

- **It is the input, not the field.** The plan sketched a `FormField` whose
  input swaps; it goes *in* a `FormField` instead, by `DatePicker`'s rule that
  a control growing its own label is a second way to write a form.
- **The swap.** Hidden it is a `core.InputPassword`, revealed a `core.Input`.
  They are different node types, so the element is replaced rather than
  patched — harmless, because focus is on the toggle just pressed, and the text
  is `Value`, the caller's, drawn into whichever element is up.
- **One hook, the reveal.** Nothing else in a form wants to know whether the
  password was visible while typed, so the widget holds it — the test
  [`TimePicker`](#timepicker) states. It inherits the hook rules: render it
  unconditionally, in a stable position.
- **A stable name with a pressed state.** The caption flips between Show and
  Hide, but the toggle's accessible name stays "Show password", with
  `core.AccessibilitySelected` saying whether it is pressed (`aria-pressed` on
  the web). A name that flipped would read as two different buttons.
- No `OnChange` raises `ConcernPasswordFieldInert`: a password that never
  reaches the app looks, at sign-in, exactly like a wrong one.

## PINInput

The boxed one-character-per-box field a one-time code is typed into: a row of
boxes drawn over **one field that holds the whole code**.

```go
comps.PINInput{
    Length:     6,
    Value:      code.Get(),
    OnChange:   code.Set,
    OnComplete: func(c string) { verify(c) },
}
```

```
┌───┐ ┌───┐ ┌───┐ ┌───┐ ┏━━━┓ ┌───┐
│ 4 │ │ 1 │ │ 7 │ │ 2 │ ┃   ┃ │   │
└───┘ └───┘ └───┘ └───┘ ┗━━━┛ └───┘
                          ▲ the next box to fill, marked while focused
```

A tap anywhere on the row puts the caret in the field (`core.Focus`). Every
key, paste, backspace and SMS autofill is an ordinary edit of one string, and
the boxes redraw from it. The field is a `core.Input` (`core.InputPassword`
when `Secure`), one point square with no frame, fill or ink, in a `ZStack`
layer over the boxes: present, focusable, not seen.

**Why one field.** It used to be six fields, one per box, with the caret moved
from each to the next as a character landed. On the Android emulator, keys
typed about 130 ms apart were lost two ways. A key typed while the caret was on
its way between boxes reached no field. And a box that had already been left
could not replay a key onto Go's rewrite of it. One field has no caret moves,
and its code is one value under the text-edit
protocol (`core/text_edit.go`) like any other field's. It is also what the
platforms' own OTP fields are, and what autofill fills.

The rules fall out of the field:

- `Value` is the field's text, capped at `Length`: a longer paste keeps what
  fits.
- Backspace deletes the last character, wherever the reader tapped.
- **`OnComplete` fires on every edit that leaves the code full**, including an
  edit to a code that was already full. This is deliberately not
  [`Countdown`](#countdown--stopwatch)'s once-per-crossing reading:
  `OnComplete` means "submit this", and a corrected code that stays silent is a
  field that will not submit. It fires from the change handler, so a screen
  restored with a complete code does not resubmit itself. A change producing
  the value already held is an echo: no `OnChange`, no `OnComplete`.

**It holds hooks** (the field's `FocusRef` and whether it has focus), so render
it in a stable position on every pass rather than inside a `core.If`, as with
[`Accordion`](#accordion).

The boxes wear the theme's field frame (`Components.Input`), and the next box
to fill takes a `Colors.Primary` border while the field has focus. They divide
the row with `core.FlexGrow` and a zero `core.FlexBasis`, the pair
`comps.Calendar`'s day cells use. The row fills the width it is given; cap it
with `Style`.

The field asks for the **number pad** with `core.Keyboard(core.KeyboardDigits)`,
which on iOS also marks it as a one-time code field, so the system offers a
code from a text message above the keyboard. The pad is a hint: a hardware
keyboard or a paste can still put letters in.

`Label` is the accessible name only: the row is a `core.RoleGroup` named by it,
and the field is named "*Label*, N of M entered". The boxes are hidden from
screen readers, being a picture of what the field holds. There is no visible
caption; wrap it in a [`FormField`](#formfield) when one is wanted.

In debug builds, a `PINInput` with no `OnChange` raises
`comps.ConcernPINInputInert` (it is read-only in practice and looks exactly
like an empty field), and a `Value` longer than the field raises
`comps.ConcernPINValueTooLong` (the extra characters are never drawn and can
never be typed away).

## TagInput

A set of short strings typed one at a time — recipients, labels, interests.

```go
comps.FormField{
    Label: "Labels",
    Input: comps.TagInput{
        Tags:        tags.Get(),
        OnChange:    tags.Set,     // the whole new set, after each commit or removal
        Label:       "Labels",
        Placeholder: "Add a label",
        Max:         6,
    },
}
// ( design ✕ ) ( urgent ✕ )
// [ Add a label                ]
```

- **Return or a separator commits.** `Separators` defaults to ",". Everything
  before the last separator is committed and what follows stays in the input,
  so a pasted "a, b, c" commits "a" and "b" and leaves " c". Pieces are
  trimmed; empties and duplicates are dropped.
- **The ✕ is its own button, the tag's text is not.** A ✕ inside a Chip would
  be a button inside a button. Each ✕ is named "Remove *tag*"
  (`RemoveLabel` changes the prefix).
- **The draft is the widget's**, so TagInput holds hooks (the draft, and a
  focus ref). Render it unconditionally. The set is the caller's.
- **The input keeps the keyboard across a return.** Return asks for focus
  back, because the next thing typed is the next tag. It changes nothing on
  the web and Compose, which keep focus anyway. On iOS, SwiftUI drops focus on
  submit, so without it every tag needed a tap first.
- `Max` stops a paste part-way and disables the input once reached.
- No "backspace on empty removes the last tag": the input reports text, not
  keys, and an empty input's text does not change.
- A `RoleList` of listitems. The items are deliberately unnamed, so the ✕
  inside each stays reachable on targets that merge a labelled container.
  `ConcernTagInputInert` for a missing `OnChange`.

## NumberPad

An on-screen keypad: ten digits, a backspace, and one corner key you choose.

```go
comps.NumberPad{
    OnKey:       func(k string) { pin.Set(pin.Get() + k) },
    OnBackspace: func() { pin.Set(dropLast(pin.Get())) },
    Extra:       ".", ExtraLabel: "Decimal point",
}
```

- **Use the system pad when there is a field.**
  `core.Keyboard(core.KeyboardDigits)` is the keyboard the reader knows, and on
  iOS it is what SMS autofill fills. `NumberPad` is for where that pad cannot
  go: a lock or payment screen with no field to focus, a kiosk where the system
  keyboard must never appear, and a static export or desktop page where an
  `inputmode` hint does nothing.
- **It holds no value.** It reports keys and the caller builds the string, so
  one widget serves a PIN, an amount and a dialler; the rule for what a key
  does is a line of Go in `OnKey`. It takes no hooks and may be rendered
  conditionally.
- `Extra` is the bottom-left key, reported through `OnKey` like a digit. Empty
  leaves the cell blank (an unpainted, disabled, hidden key, so the zero stays
  exactly under the eight).
- Backspace reports through `OnBackspace`, never through `OnKey`. With no
  `OnBackspace` that key alone is disabled.
- Every reporting key gives `core.HapticLight`. A `Disabled` pad is silent,
  and also drops a tap that races the disabling patch.
- Keys are equal shares at least 56 points tall, in the telephone layout. Cap
  the width on a tablet with `core.MaxWidth` in `Style`.
- The pad is a `RoleGroup` named `Label` ("Number pad"); backspace is named
  `BackspaceLabel` ("Delete").
- No `OnKey` and not `Disabled` raises `ConcernNumberPadInert` in debug builds.

## ColorSwatchPicker

One colour from a set, as a radiogroup of swatches.

```go
comps.ColorSwatchPicker{
    Label: "Label colour",
    Colors: []comps.Swatch{
        {Hex: "#2A78D6", Name: "Brand blue"},
        {Hex: "#FFDD00", Name: "Sunshine"},
    },
    Value:       colour.Get(),
    OnChange:    colour.Set, // receives "#RRGGBB"
    AllowCustom: true,
}
```

- **No `Colors` means the theme's chart colours**, which are already chosen to
  be told apart and validated against the theme's surface. They are named from
  their hues ("blue", "orange").
- **Selected is a ring and a check**, never the colour alone. The check's ink
  is black or white by contrast with its swatch. Every swatch carries the
  ring's border (transparent when unselected), so selecting moves nothing.
- `Value` is compared without regard to case or the short form: `"#fd0"`
  selects a `#FFDD00` swatch. A colour listed twice is offered once.
- `Swatch.Name` is what a screen reader says. An unnamed swatch is spoken by a
  name guessed from its hue and raises `ConcernColorSwatchUnnamed`; a `Hex`
  that is not `#rgb` or `#rrggbb` is not drawn and raises
  `ConcernColorSwatchBadHex`.
- `AllowCustom` adds a hex field under the grid. Six digits commit as they are
  typed; the short form commits on return only, because every six-digit colour
  passes through a valid three-digit one on its way. A custom `Value` shows in
  the preview beside the field.
- **It holds a hook** (the half-typed hex, which no application wants), so
  render it unconditionally, in a stable position. The hook is taken whether
  or not `AllowCustom` is set.
- On the web the group's arrow keys are Up and Down, in reading order through
  the rows: the runtime gives a composite one axis, and this one is a column
  of rows.
- It is not a hue and saturation square. That needs a touch position on a
  Canvas, which no event carries.

## RangeSlider

A minimum and a maximum that cannot cross.

```go
comps.RangeSlider{
    Title: "Price", Min: 0, Max: 200, Step: 5,
    Low: low.Get(), High: high.Get(),
    OnChange: func(l, h float64) { low.Set(l); high.Set(h) },
    Format:   func(v float64) string { return fmt.Sprintf("$%.0f", v) },
}
```

- **It is two `SliderRow`s, on purpose.** One track with two thumbs is a node
  type no target has. Two labelled sliders are also the form the control takes
  for VoiceOver and TalkBack on every platform, which adjust one value per
  stop. What the eye loses, the title line gives back: it states the range in
  words ("$20 – $80").
- **The thumbs push each other.** Dragging `Low` past `High` carries `High`
  along, and the reverse, so `OnChange` always reports an ordered pair and a
  range can be moved as a whole from either end. A minimum width is the
  caller's rule, applied in `OnChange`.
- `OnChange` fires once when a drag ends, as `SliderRow`'s does. The caller
  holds both values; the widget takes no hooks.
- `Labels` replaces "Minimum" and "Maximum". `Format` writes every number
  drawn.
- An inverted pair is drawn the right way round and raises
  `ConcernRangeSliderInverted`; no `OnChange` and not `Disabled` raises
  `ConcernRangeSliderInert`.

## MaskedInput

A text field that formats as the reader types.

```go
comps.MaskedInput{
    Mask:     "(###) ###-####",
    Value:    phone.Get(), // "5551234567"
    OnChange: func(raw, formatted string) { phone.Set(raw) },
    Keyboard: core.KeyboardDigits,
    Label:    "Phone",
}
```

- **The mask:** `#` a digit, `A` a letter, `*` either; anything else is a
  literal. A key the next slot refuses is dropped, and so is anything past the
  last slot.
- **`Value` is the raw value**, slot characters only, which is the form an
  application stores and sends. `OnChange` hands over the drawn text as well.
  `OnComplete` fires on every edit that fills the last slot.
- **Literals are written late:** `555` draws `(555`, and the `) ` arrives with
  the fourth digit. The text never ends in a literal, so backspace always
  removes something the reader typed. (Written eagerly, `(555) ` minus one
  character formats straight back to `(555) `.)
- A pasted `555.123.4567` reads as the same ten digits. The text is walked
  against the mask, so a mask with a literal its own slots could hold
  (`+1 (###) ###-####`) reads that literal as the literal; the price is that a
  raw value under such a mask cannot begin with that character.
- `Placeholder` defaults to the mask's shape, `(___) ___-____`.
- It takes no hooks and may be rendered conditionally.
- **Every formatted keystroke is a rewrite** under the text-edit protocol.
  Typing at machine speed loses nothing (measured on the Android emulator).
  The caret travels with its text on all three live hosts. One limit: a key
  typed mid-text at the very end of a group leaves the caret after the
  reflowed digits, since no host reports its caret to Go.
- A mask with no slot raises `ConcernMaskedInputNoSlots` (usually a mask in
  another library's alphabet, `999-999`); no `OnChange` and not `Disabled`
  raises `ConcernMaskedInputInert`.


## DateRangePicker

A two-date field: a tappable summary of the chosen span that opens a
[`Calendar`](api/comps-inputs.md#type-calendar) in a modal sheet and closes again on
the tap that completes the range.

```go
comps.FormField{
    Label: "Stay dates",
    Input: comps.DateRangePicker{
        Start:    from.Get(),
        End:      to.Get(),
        OnChange: func(a, b time.Time) { from.Set(a); to.Set(b) },
        Calendar: comps.Calendar{Today: today, Min: today},   // the template
    },
}
```

It is `DatePicker`'s shape with one date more and one rule more. The trigger,
the sheet, the two ways out and the Calendar-as-template are the same, so a
form holding one of each looks like a form rather than like two widgets.

### One rule makes the whole protocol

The sheet's grid reports one tapped day at a time, as `Calendar` always does.
What turns that into a range is a single piece of state the widget owns — a
*pending start* — and one rule over it:

| state | what a tap means |
| --- | --- |
| no pending start | this tap becomes the pending start |
| a pending start | this tap is the other end; the range is reported |

Everything a range picker is usually specified with falls out of those two
lines:

- **The third tap starts a new range.** Completing one clears the pending
  start, so the next tap finds none and begins again. There is no "is this
  nearer the start or the end" arithmetic.
- **Tapping one day twice is a one-day range**, start and end on one cell.
- **The second tap may be the earlier one.** The pair is ordered before it is
  reported, so `OnChange` always receives start ≤ end. Treating an earlier
  second tap as a restart would throw away a tap the reader made on purpose,
  and "the other end" is not a claim about which end.

### The pending start is the widget's; the range is not

This is the package's third state-owning widget, after `Accordion` and
`DatePicker`, and it owns one piece more than `DatePicker`: the sheet is open,
the month being browsed, and the half-made range. So render it unconditionally,
in a stable position, every pass.

The third one is state no application wants — the test [`SliderRow`](#sliderrow)
states. A form's field is a span of days or it is nothing, and "from the 14th,
no end yet" is a value it would have to invent a way to hold and a way to draw.
Worse, holding it in the caller would make closing the sheet mid-pick
*destructive*: the first tap would already have overwritten the range the
reader opened the sheet to look at, and the backdrop, the ✕ and the back
gesture would all be traps. `OnChange` therefore fires once per completed
range, never mid-pick, and every way out of the sheet discards the pending
start.

While the range is half made the sheet shows the pending start alone, as one
lit day with no band, and the caller's own span leaves the grid — so the old
range and the new start are never both lit.

`Start` and `End` are the caller's to hold. A `Start` with no `End` is drawn as
that one day and summarized as that one date; it is not a state this widget
produces. `Format` writes each half (default `"Jan 2, 2006"`, `DatePicker`'s)
and `Separator` joins them (default `" – "`). `OnClear` puts a Clear button in
the sheet, as it does on `DatePicker`.

In debug builds, a `DateRangePicker` with no `OnChange` raises
`comps.ConcernDateRangePickerInert`: it takes every tap, completes a range and
hands it to nobody, which looks exactly like a picker nobody has finished
using.

### Calendar's band

The drawing is `Calendar`'s, through two fields the picker drives and any grid
may set for itself:

```go
comps.Calendar{RangeStart: from, RangeEnd: to, Today: today}   // no OnSelect: a picture

// │ 15  16 [17]▓18▓▓19▓▓20▓[21] 22 │   [n] endpoint, ▓ interior
```

The two endpoints wear the selected day's fill — there is one "this day is
chosen" look in the grid and it stays one look — and the days between wear the
same colour thinned to 20% and give up their corner radius, which is the whole
of what makes a run of them read as one shape rather than as a row of pills.
The rounded endpoint meets the square band with a small notch, because `core`
has one border radius and not four; Material's range picker draws the same
notch on purpose.

Three edges worth knowing:

- The band **runs through the leading and trailing adjacent days** rather than
  stopping at the 1st. They stay dimmed and inert; cutting the band where the
  month happens to end would stop it somewhere the reader can see no reason
  for.
- It **breaks at the end of each week row**, which is where a calendar breaks.
- **Today's ring survives inside it** and goes square with the cells it sits
  in. A range covering today is the common case, and losing the ring there
  would be the one place the grid stopped saying what day it is.

For assistive technology every day of the span is announced as selected, not
only its two ends — ARIA's own date-range grid marks the whole band. The two
ends are then named in the cell's label, `", start of range"` and `", end of
range"` (or `", start and end of range"` for a one-day span). That is a suffix
rather than a state for the one reason this calendar accepts a suffix at all:
no target has a property for it, so the alternative is fourteen identically
named selected days with no findable edge.

A `RangeEnd` before its `RangeStart` has nothing between the two to fill, so
the grid shows two lone endpoints and raises
`comps.ConcernCalendarRangeReversed` in debug builds.

## TimePicker

A time-of-day field: an hour, a minute and — on a 12-hour clock — an AM/PM,
each a [`core.Select`](api/core-controls.md#func-select), in one row.

```go
comps.FormField{
    Label: "Start time",
    Input: comps.TimePicker{
        Value:      start.Get(),
        OnChange:   start.Set,   // on every pick
        MinuteStep: 15,
        Label:      "Start time",
    },
}

// [ 9 ▾] : [30 ▾]  [AM ▾]      12-hour, the default
// [09 ▾] : [30 ▾]              Hour24
```

### No sheet, because nothing is ever half made

`DatePicker` needs a sheet because a month grid does not fit in a field, and
its sheet needs no Done button because one tap is the whole choice. A time is
two or three choices, so a sheet around them would need either a Done button
and a draft held until it is pressed, or a live value the ✕ cannot take back —
the trap [`DateRangePicker`](#daterangepicker) exists to avoid.

Neither is needed, because a time has no invalid intermediate. Changing only
the hour of 9:30 gives 10:30, which is a real time and exactly the one asked
for; there is no "hour chosen, minute pending" in the way "from the 14th, no
end yet" is a state. Every pick is a complete value, so every pick is reported
at once. With no draft there is no hook: `TimePicker` is stateless like
`Stepper`, and may be rendered conditionally.

It is built from `Select`s rather than `Stepper`s because each part is then
the platform's own picker, and because 9:00 to 17:30 is two picks where a
stepper wants eight taps on the hour alone — and clamps at 23 where a clock
wraps. A sheet of `Select`s would have been a popup opening popups.

### What changes, and what does not

`OnChange` receives `Value` with its hour and minute replaced, **on `Value`'s
own date and in its own location**, so a `DatePicker` and a `TimePicker` can
edit one `time.Time` between them — each its own half. The seconds and
nanoseconds are zeroed, because the field does not show them. A pick of the
option already shown is not reported.

A zero `Value` is drawn as midnight. There is no blank option and no
placeholder: a blank would bring back the half-made time. A form where the
time is optional puts a switch beside the field.

`Hour24` is `DigitalClock.Hour24`'s word, so the clock and the field that sets
one are configured alike, and it follows the clock's padding rule: `09` on a
24-hour picker, `9` on a 12-hour one. On a 12-hour picker the hours are listed
12, 1 … 11, and flipping AM/PM keeps the hour within the period. `AMLabel` and
`PMLabel` localise the period, since Go spells it in English only.

### A minute off the step is kept, not rounded

`MinuteStep` thins the minute list — 15 gives :00, :15, :30, :45; zero offers
every minute. A `Value` whose minute is off the step (9:07, loaded from
somewhere else) has its own minute slotted into the list in order, so the field
shows 9:07. Rounding for display would be a lie about the value, and rounding
through `OnChange` would be a write the reader never made. The extra option
leaves as soon as another minute is picked.

### Accessibility

The row is a group named by `Label`, with the whole time as its value —
`Stepper`'s shape — so a native reader arriving at it hears "Start time, 9:30
AM" once. Each picker is named `"Hour"`, `"Minute"` and `"AM/PM"` by default
(`HourLabel`, `MinuteLabel`, `PeriodLabel` localise them), because a picker's
own text is only its current option. The colon is hidden.

In debug builds, a `TimePicker` with no `OnChange` raises
`comps.ConcernTimePickerInert`: every pick goes nowhere, which on screen looks
like a field that simply has not been changed.

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

## Breadcrumb

The trail from the root of a hierarchy to the current page.

```go
comps.Breadcrumb{
    Items: []string{"Files", "Photos", "2026"},
    OnTap: func(i int) { nav.PopTo(i) },
}
// Files › Photos › 2026
```

- A `RoleNavigation` named "Breadcrumb" — ARIA's breadcrumb pattern. Every
  ancestor is a ghost [`Button`](#button); the chevrons are hidden.
- **The last item is not a button.** It is where the reader already is, so it
  is text stating `core.CurrentPage`, and not a dead tab stop at the end of the
  trail.
- With `OnTap` nil the whole trail is text — a location label, not navigation —
  which is why a nil `OnTap` is not a concern.
- A long trail wraps rather than truncating; collapsing the middle into "…" is
  a caller's shortened `Items`.

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

- The current item is drawn bold in the primary on-light tone and states
  `core.CurrentPage`: `aria-current="page"` on the web, selected on Compose and
  SwiftUI. Its name stays the label. A tab role would claim a panel the bar does
  not control.
- The icon is decoration and hidden from assistive technology.
- `BarItem.AccessibilityLabel` replaces an abbreviated label as the spoken name.

### Badges

`BarItem.Badge` puts a count on an item as a [`Badge`](#badge) over the icon's
top-end corner. The icon becomes a two-layer `ZStack` — the glyph, centred, and
the badge placed `core.StackAlignTopEnd`. The glyph keeps a symmetric
horizontal margin so the badge overlaps its corner rather than all of it, and
no top margin, so a badged icon stays level with its neighbours. An item with
no badge renders the tree it always did. The badge is the compact pill both
platforms' bars use, two points under a free-standing `Badge`, so it covers
the corner and not the glyph.

```go
{Icon: "✉", Label: "Inbox", Badge: "3", BadgeLabel: "3 unread"}
```

The count joins the item's name — "Inbox, 3 unread" — because the cell is one
button with one name; `BadgeLabel` is the spoken form, `Badge` itself when
empty. The badge is `VariantError`, the colour both platforms give a count.
With no icon, the label wears the badge.

## FAB

The floating action button: a screen's one primary action as a raised disc
over the content. It is a `comps.Button` in a circle, so its fill, ink,
disabled treatment and `Style` order are Button's, and belongs in
`Screen.Floating`.

```go
comps.FAB{Icon: "+", AccessibilityLabel: "New note", OnTap: create}
comps.FAB{Icon: "✎", Label: "Compose", OnTap: compose}   // extended
comps.FAB{Icon: "↑", Size: comps.FABSmall, AccessibilityLabel: "Top"}
```

**Two shapes at one height.** Icon-only is a disc: 56 points across (40 for
`FABSmall`), no padding so the glyph centres, the glyph sized to the disc.
With `Label` it is the *extended* form, a pill as wide as its glyph and word
with `Spacing.MD` at each side. Both are the same height, so a label can be
added without the button moving.

**It does not place itself.** A floating thing needs a layer to float on, and
`core.ZStack` sizes to its largest layer, so the screen that already fills the
safe area is what hosts it: `Screen.Floating` places it bottom-end above the
`Footer`. Anywhere else a `FAB` is an ordinary round button in the flow.

Other notes:

- Icon-only needs `AccessibilityLabel`. A `+` is a glyph, not a name, and the
  widget invents nothing to say instead.
- The sizes are fixed points, not theme spacing steps: a FAB is a touch target
  first, and 56 and 40 are the platform norms.
- `core.Shadow(6)` raises it above cards (which sit at 2). `Style` lands after
  the widget's own props, so `core.Shadow(0)` flattens it.
- The zero `Variant` is the theme's own Button pairing, so a FAB and a Button
  on one screen agree about what primary looks like.

## Drawer

Side navigation: a panel of destinations pinned to the leading edge over the
screen. A ☰ button opens it, and picking a destination, the ✕ or a tap on the
scrim closes it.

```go
open := core.NewState(ctx, false)
section := core.NewState(ctx, 0)
closeRef := core.UseFocusRef(ctx)

comps.Drawer{
    Open:      open.Get(),
    OnDismiss: func() { open.Set(false) },
    Title:     "Notebook",
    Items: []comps.DrawerItem{
        {Icon: "📥", Label: "Inbox",   OnTap: func() { section.Set(0) }},
        {Icon: "⭐", Label: "Starred", OnTap: func() { section.Set(1) }},
    },
    Selected: section.Get(),
    CloseRef: closeRef,
    Content: comps.Screen{Children: []core.View{
        comps.AppBar{Title: "Inbox", Leading: comps.Button{
            Label: "☰", AccessibilityLabel: "Open navigation",
            Emphasis: comps.EmphasisGhost,
            OnTap: func() { open.Set(true); core.Focus(closeRef) },
        }},
        body,
    }},
}
```

**It is a layer, not a Modal.** The drawer is a `ZStack` of two layers: the
screen, and a Row holding the panel and the scrim. A Modal would be a centred
Dialog window on Compose and a bottom sheet on SwiftUI, so only the web could
pin it to an edge. A ZStack layer at 100% × 100% fills the stack on all four
targets.

What a Modal would have supplied, and what the drawer does instead:

| Modal gives | Drawer |
|---|---|
| Screen-reader confinement | The screen layer is `AccessibilityHidden` while open |
| Focus inside the dialog | `CloseRef` names the ✕; the opener calls `core.Focus` on it |
| Tab stays inside (web) | Not contained: core has no `inert`, and `aria-hidden` does not stop Tab |
| Android back closes it | `core.OnBack(OnDismiss)` on the panel layer while open |

**Give it a box to cover.** The drawer covers its ZStack, which is as big as its
largest layer. At the app root or in a bounded parent it covers that. Inside a
scrolling column nothing bounds the height, so pin one with
`Style: []core.StyleProp{core.Height("360px")}`.

**Shut is hidden, not removed.** The panel layer is rendered every pass with
`Display none` while shut. Opening is a style patch, and hooks inside `Body` keep
their slots.

Other notes:

- Picking a destination runs its `OnTap` and then `OnDismiss`.
- `Selected` follows `BottomBar`: the zero value selects the first item and a
  negative value selects none. The current row states `core.CurrentPage`
  through `ListRow.Current`.
- The panel is a `RoleNavigation` landmark named by `Title`.
- `Body` replaces the rows with any content. Its handlers close nothing unless
  they call `OnDismiss`.
- `Width` defaults to `280px`. Cap a percentage with `core.MaxWidth` through
  `PanelStyle`; all four targets honour it.
- No `OnDismiss` draws no ✕ and leaves the scrim inert.
- Drawer holds no hooks.

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
- A `Body` taller than the screen scrolls inside the dialog on every target,
  so a long form on a landscape phone is reachable rather than cut off.

## Lightbox

One image, large, over everything, on a dark ground.

```go
comps.Lightbox{
    Src:       photo.URL,
    Alt:       photo.Description,
    Open:      viewing.Get(),
    OnDismiss: func() { viewing.Set(false) },
}
```

- **Dialog's plumbing, not Dialog.** A `core.Modal`, controlled by `Open`, every
  way out reported through `OnDismiss` — with an image where [`Dialog`](#dialog)'s
  card would be.
- **Fit.** `core.ContentModeFit`, never cropped: the reason to open a thumbnail
  is to see what its crop cut off. `Height` (400 by default) and the panel's
  width bound it.
- **Dark on every target, in every theme.** The scrim is near-black and the
  panel black, with white ink. The panel is filled rather than transparent
  because SwiftUI presents a `Modal` as a sheet on the system background, not
  over the scrim.
- The image is named by `Alt`, or hidden when there is none (the `Caption`, if
  any, is then what is read). The ✕ is named "Close".
- No `OnDismiss` raises `ConcernLightboxInescapable`: the close button and the
  scrim would close nothing.

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
- `Checked` marks the current choice with a leading ✓ and states
  `core.CurrentTrue` (`aria-current="true"`, selected on the natives). It is how
  a `Menu` becomes a picker.
- Actions are buttons, not listbox options. They are commands, and an option
  would be announced "not selected".
- `Cancel`, a tap above the panel and the scrim all call `OnDismiss`. With
  `OnDismiss` nil, only `Visible` closes the sheet.
- The card carries `AccessibilityLabel(Title)` and the title is a heading. The
  Modal chassis supplies the dialog semantics.
- `Style` lands on the card last. `core.MaxWidth` keeps the panel from spanning
  a wide browser window or a tablet.
- The Android and iOS placement comes from reading the renderers. It has not
  been checked on a device.

## Menu

A button that opens a short list, such as the "⋯" on a row or a "Sort by"
picker. It is an `ActionSheet` with a trigger.

```go
open := core.NewState(ctx, "") // which row's menu is open

comps.Menu{
    Trigger: comps.Button{Label: "⋯", AccessibilityLabel: "Actions for " + note.Title,
        Emphasis: comps.EmphasisGhost},
    Open:      open.Get() == note.ID,
    OnOpen:    func() { open.Set(note.ID) },
    OnDismiss: func() { open.Set("") },
    Title:     note.Title,
    Items: []comps.SheetAction{
        {Label: "Pin to top", OnTap: pin},
        {Label: "Delete", Variant: comps.VariantError, OnTap: del},
    },
    Cancel: "Cancel",
}
```

**There is no popover.** A list anchored under its button needs the button's
position, and no host sends that to Go. Every target can present a Modal, so the
list is the bottom-edge sheet from `ActionSheet`, with its rules. Picking an
item runs its `OnTap` and then `OnDismiss`, and Cancel, the scrim and a tap above
the panel all dismiss.

**Open is the caller's state.** A menu on every row of a list is the common
case. A widget that kept its open flag in a hook would take one slot per row,
and those slots drift when the row count changes. With `Open` controlled, one
state names the open menu and each row compares against it.

**A picker is a menu with a checked item.** `SheetAction.Checked` puts a ✓
before the label and marks the item as the current one (`core.CurrentTrue`). Each item sets the
value in its own `OnTap`. Put the current value in the trigger's label.

```go
comps.SheetAction{
    Label:   "Newest",
    Checked: order.Get() == "newest",
    OnTap:   func() { order.Set("newest") },
}
```

Other notes:

- `Trigger` is a `comps.Button` template. Its label, variant, emphasis,
  `Disabled`, `Style` and names apply, and `OnOpen` replaces its `OnTap`. A
  widget cannot attach a tap to a View it did not build, so the slot is a
  Button. An icon trigger needs `AccessibilityLabel`.
- The trigger states `aria-haspopup="dialog"` (`core.AccessibilityHasPopup`)
  and no expanded state. A control that opens a dialog is not a disclosure. The
  value is `dialog`, not `menu`: the sheet is a dialog of buttons, not an ARIA
  menu with a menu's keyboard.
- `Checked` is a name suffix, not a selected state. On a button that state
  becomes `aria-pressed`, which announces a toggle.
- `Style` lands on the sheet's card, as `ActionSheet.Style` does.

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

Like `Accordion`, render it on every pass and drive `Visible`. Leaving it out of
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

`FocusRef` names the input, so `core.Focus` can target it and
`core.UseFocusOrder` can include it in a form's return-key order.

## SearchableSelect

A choice from a list too long to scroll, such as countries. Typing into a search
field filters the options into a short list under it, and a tap picks one.

```go
comps.SearchableSelect{
    Label:         "Country",
    Options:       countries, // []core.SelectOption
    Value:         country.Get(),
    OnChange:      country.Set,
    Query:         query.Get(),
    OnQueryChange: query.Set,
    FocusRef:      countryRef,
}
```

**When the list shows.** The list shows while `Query` is not empty and is not
the chosen option's label. A pick reports the option's `Value` through
`OnChange`, writes its label through `OnQueryChange` (which closes the list),
and dismisses the keyboard. Editing the text opens the list again. Clear
empties both the text and the choice. The widget holds no hook, so it can be
rendered conditionally.

**Focus and the keyboard.**

| Question | Answer |
|---|---|
| Does the list take focus when it appears? | No. Typing carries on in the field. |
| What does the return key do? | It belongs to the form. The field has no submit, so with `FocusRef` in `core.UseFocusOrder` it shows Next and moves to the next field. On the web, once the arrows have reached an option, Enter picks it instead. |
| Does Next stop on the list? | No. The order walks declared refs, and no option is a field. |
| Does Enter pick the top match? | No. An explicit submit would suppress Next, and one key cannot do both. |
| Keyboard on the web? | The field is an ARIA combobox and keeps focus: the arrows move an active option (outlined, and named by `aria-activedescendant`), Enter picks it, Escape clears it. Tab moves past the list. The arrows do not select. |
| Where does focus go after a pick? | On a phone the keyboard is dismissed. On the web a pick made with Enter leaves focus in the field: the runtime declines the dismiss that follows it. A tap dismisses as on a phone. |

Other notes:

- The matches are a `RoleListBox` named "<Label> suggestions", and each row is a
  selectable `ListRow`. A status line (`RoleStatus`) says "3 matches",
  "5 of 6 matches" or "No matches". It is hidden while the list is shut.
- The field is `core.RoleComboBox`, with `aria-expanded` while matches show and
  `aria-controls` naming the list. The list's id is `ID`, or one derived from
  `Label` (`searchable-select-country`), so two selects with the same label need
  an `ID` each. Rows are `<ID>-option-N`.
- `Options` are `core.SelectOption`, the type `core.Select` takes. `Group`
  becomes the row's subtitle, because a heading inside a listbox would be a
  foreign child. `Disabled` and `GroupDisabled` rows are listed but cannot be
  picked.
- `MaxResults` caps the rows (default 6). `Filter` replaces the
  case-insensitive label match, and `Count` replaces the English status text.

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
apart. Once you declare the role, the WASM runtime supplies the toolbar
keyboard: one tab stop for the strip and the arrow keys between its chips.

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

**The platform turns it.** The ring carries `core.Spin(1000)`, one revolution
a second, so each renderer's own frame clock drives it: CSS keyframes on the
web, a graphics layer on Compose, a `TimelineView` on SwiftUI. Go sends nothing
while it spins, so a visible spinner costs no patches and no render passes.

**It holds no hooks.** Leaving it out of the tree is safe. `Hidden` is still
the better switch where the spinner has a fixed place in a layout: it keeps
the tree the same shape, so showing it is a style patch. A hidden spinner is
`Display` none and draws no frames on any target.

It used to be stepped from Go, 30 degrees every 80 milliseconds through a state
slot and `hooks.UseIntervalWhile`. That cost a render pass per step and gave it
two hook slots, which is why older code renders it unconditionally.

Other notes:

- The ring carries an orbiting dot, because a uniform ring turned about its
  centre draws the same pixels at every angle.
- The outer box is a `RoleStatus` live region named by `Label`, which defaults
  to "Loading". The ring is hidden from assistive technology.
- No target slows the spin for a reduce-motion setting yet; core has no signal
  for it.
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

## Histogram

Raw values counted into bins, drawn as touching bars over a numeric axis.

```go
comps.Histogram{Subject: "Response time (ms)", Values: samples}
// 12 ┤      ██
//  6 ┤   ██ ██ ██ ██
//  0 ┼██─██─██─██─██─██┤
//    0    100   200   300      labels on the bin *edges*
```

- **The axis names edges.** A bin is a range, and what a reader measures
  against is where it starts and stops. n bins have n+1 edges evenly spaced
  edge to edge — `LineChart`'s point spacing — so the labels use that axis
  rather than `BarChart`'s centred categories, and the bars touch.
- **Nice edges.** The edges come from the charts' own nice-number scale, so a
  bin is 10, 25 or 0.5 wide and starts on a multiple of that. `Bins` is
  therefore a *target*; zero takes Sturges' rule, ⌈log₂ n⌉ + 1. Bins are
  half-open, `[a, b)`, with the maximum in the last one.
- The count axis ticks in whole counts only.
- One `RoleImg`, one sentence: "Response time (ms): 120 values in 8 bins of 50
  from 0 to 400; most, 34, between 100 and 150."

## Heatmap & CalendarHeatmap

A grid of values as a grid of colours, and the contribution calendar built
on it.

```go
comps.Heatmap{
    Subject:      "Orders by hour",
    RowLabels:    []string{"Mon", "Tue", "Wed"},
    ColumnLabels: []string{"9", "12", "15", "18"},
    Values:       [][]float64{{2, 8, 5, 1}, {3, 9, 7, 2}, {1, 4, math.NaN(), 0}},
}

comps.CalendarHeatmap{Subject: "Workouts", Days: workouts, End: today}
```

- **The colours are the theme's `Sequential` role** — five steps of one hue
  in even lightness, added for this widget — not the categorical chart hues.
  See [Color roles](concepts/styling-and-theming.md#color-roles). `Colors`
  overrides it per chart.
- **Steps, not a gradient.** The range is cut into as many equal steps as the
  scale has colours, and the legend keys each one: low value, the swatches,
  high value.
- **No data is not zero.** `NaN` is painted `Surface` — the scale's first step
  was checked against every bundled `Surface` for exactly this pair. Zero is a
  value. `CalendarHeatmap` turns a day with nothing into no data, the
  contribution calendar's convention, and does not draw days after `End` at
  all.
- **One shape per colour.** Every cell of a step is a subpath of one path, so
  a 7 × 53 calendar is six shapes, not 371.
- **Labels.** Row labels sit in boxes exactly one row tall. An empty column
  label lends its slot to the label before it, so a month name spans its
  weeks; the calendar names a column when it holds the 1st.
- The calendar is `Weeks` columns (17 by default) ending on the week holding
  `End`, each starting on `WeekStart` (Sunday); entries are summed per date in
  `End`'s location.
- **The grid always fits; `Weeks` decides the cells' shape.** Columns stretch,
  so a year across a phone is a grid of slivers. `WeeksFor(width)` returns the
  count at which cells come out square, for a width you supply — usually
  `hooks.UseWindow(ctx).Width` less your padding. It is arithmetic, not a
  measurement, and reads no hook.
- One `RoleImg`, one sentence: "Workouts: 42 over 17 weeks, on 23 days; most
  on Tue 3 Mar 2026, 4." **A tie for the most names the earliest day.** It
  is stable (the same data always names the same date), and the sentence
  does not say that other days tied.

## CandlestickChart

A price per period: a wick from low to high, a body from open to close.

```go
comps.CandlestickChart{
    Subject: "ACME",
    Labels:  days,
    Candles: []comps.Candle{{Open: 102, High: 108, Low: 101, Close: 107}, ...},
}
```

- **The axis does not start at zero.** A candle states a range, not a length,
  so the scale brackets the lows and highs, as `LineChart`'s does.
- **`UpColor` and `DownColor` are fields** because the convention is regional:
  the defaults are the theme's `Success` for a close at or above the open and
  `Error` below it, and markets in China, Japan and Korea print it the other
  way round. `ShowLegend` keys the two (`UpLabel`, `DownLabel`).
- **A doji keeps a 1px body**, so it does not read as missing data. Exact
  without measuring: the plot is `Height` px and the stretched canvas maps y
  linearly.
- Four shapes carry every candle (rising and falling wicks and bodies). A
  candle holding a NaN keeps its slot and draws nothing.
- One `RoleImg`, one sentence. Colour is the only drawn difference between a
  rise and a fall, so it ends "…; low 97 in Wed, high 109 in Tue; 2 up, 1
  down." Up to five periods are read in full.

## FunnelChart

How many survive each step of a process.

```go
comps.FunnelChart{
    Subject:   "Checkout",
    ShowRates: true,
    Stages: []comps.FunnelStage{{Label: "Visited", Value: 1200}, {Label: "Signed up", Value: 744}, {Label: "Paid", Value: 93}},
}
//   Visited  ████████████████  1200
//             ╲████████████╱         62%     a rate sits on the boundary
// Signed up    ██████████       744          it describes
//               ╲██████╱             13%
//      Paid       ██             93
```

- A band is as wide at the top as its own value and at the bottom as the next
  stage's. Names, values and rates are columns of Text cut into the same px
  bands as the drawing; the rates column is shifted half a band.
- **A stage larger than the one before is drawn as given, never clamped**:
  real funnels have them (re-entry), and its rate reads over 100%. The usual
  cause is stages out of order, so debug builds report
  `ConcernFunnelChartStageGrows` until `AllowIncrease` says it is meant.
- **One hue, fading**, not the categorical palette: the stages are one
  population at successive moments. `FunnelStage.Color` or `Colors` overrides.
- One `RoleImg`: "Checkout: Visited 1200; Signed up 744, 62% of the step
  before; Paid 93, 13% of the step before; 8% overall."

## RadarChart

Several measures of one subject as a polygon on spokes.

```go
comps.RadarChart{
    Subject: "Player",
    Axes:    []string{"Speed", "Power", "Stamina", "Skill", "Vision"},
    Series:  []comps.ChartSeries{{Name: "Ade", Values: []float64{8, 6, 7, 9, 5}}},
    Max:     10,
    Filled:  true,
}
```

- The first axis points up and the rest run clockwise. The grid is polygons,
  so a gridline between two spokes is as straight as the data's edge.
- **Rim labels are ZStack layers moved by `core.Translate` in px.** A ZStack's
  nine named places cover four axes and no more. The px are exact because
  `Size` is: the canvas is a `Size` px square. Labels on the right are
  start-aligned and on the left end-aligned; `LabelWidth` (56) is each one's
  box, and the widget is wider than `Size` by two of them.
- Ring values sit beside the upward spoke on a translucent chip, since a
  polygon can pass through them anywhere inside the rim. `HideScale` drops
  them.
- `Max` is shared by every axis; 0 widens the data's high to a value `Rings`
  divides roundly. A value over `Max` is held at the rim, a negative or NaN one
  at the centre.
- **Under RTL the labels mirror and the drawing does not**, which is every
  chart's label row's disagreement with its Canvas.
- Fewer than three axes: `ConcernRadarChartTooFewAxes`, and an empty rim.
- One `RoleImg`: "Player: Ade, Speed 8, Power 6, …". Past eight axes, each
  series' low and high.

## Waveform

A recording's loudness as mirrored bars, the played part in the accent.

```go
w := comps.Waveform{Peaks: msg.Peaks, Progress: position / duration}
w.Bars = w.BarsFor(win.Width - insets)
```

- **The peaks are the caller's**, 0 to 1 in time order. No host decodes audio
  for Go; a server or a build step computes them.
- **Bars are round-capped strokes, not rectangles.** The drawing is stretched
  to the width the layout gives, which would turn a rounded rectangle's
  corners into ellipses; a Canvas stroke is never scaled, so a bar's
  thickness and its round ends are exact px on every target.
- **More peaks than `Bars`: each bar keeps its bucket's maximum**, never the
  mean, so a one-sample transient survives. `Bars` of 0 is one per peak, up to
  56. `BarsFor(width)` is the count that fits a width you supply, as
  `CalendarHeatmap.WeeksFor` is for weeks. Too many bars for the real width
  close the gaps between them, so err low.
- **Display only.** Seeking by tap needs the tap's x, which no event carries.
  [AudioPlayer](#audioplayer) offers it as its `Waveform` field, above the
  seek bar, and keeps the slider.
- One `RoleImg`: "Audio waveform, 40 percent played". `Decorative` hides it
  beside a control that already speaks the position.

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

## Countdown & Stopwatch

The two widgets that watch a duration: one counting toward a deadline, one
counting away from a start.

```go
comps.Countdown{Until: expiresAt, OnDone: func() { code.Set("") }}
comps.Stopwatch{Since: startedAt.Get(), Elapsed: banked.Get(), Running: running.Get()}
```

**They own a tick, so they are not conditional-safe.** `DigitalClock` takes a
`time.Time` and holds nothing, which is what lets it be rendered
conditionally and tested at a fixed instant. A countdown has to know what
"now" is, so it holds a hook — and with it the rule
[`Accordion`](#accordion) states: render it in a stable position on every
pass and drive `Hidden`, rather than wrapping it in a `core.If`. A hidden
timer is `Display none` and costs no pixels.

**The tick is `hooks.UseIntervalWhile` with an empty callback.** Both widgets
read the clock in their own `Render`, so all a tick has to do is bring the
render back. It runs only while there is a reason for it:

| state | ticking |
| --- | --- |
| counting, visible | yes |
| counting, `Hidden` | only if `OnDone` is set |
| finished | no |
| `Hidden` with no `OnDone` | no |

The second row is the one worth stating. Hiding a countdown removes the
reason to draw but not the reason to count — somebody is still waiting to be
told it ran out — so a hidden countdown that owes an `OnDone` keeps ticking,
and one that owes nothing stops dead. A `Stopwatch` has no such exception,
because it owes nobody a callback.

**Not `hooks.UseNow`.** That hook aligns its ticks to the wall clock, so a
`DigitalClock` changes its seconds digit when the phone's status bar does. A
countdown has no such phase to share: its own boundaries fall at `Until` minus
a whole number of seconds, which nothing else on the screen is on. There being
nothing to align to, the cheaper hook wins — and `UseNow` cannot be paused,
which the table above needs.

**`OnDone` comes from an effect, not from the render pass.** A render may run
more than once for one state and runs while the tree is being built, so a
handler called from inside it would fire twice or re-enter the renderer.
`OnDone` is a `hooks.UseEffect` keyed on whether the deadline has passed, so
it fires once per *crossing*:

```
remaining  5s ──── 4s ──── … ──── 1s ──── 0 ──── 0 ──── 0
deps       false   false         false   true   true   true
OnDone      ·       ·             ·      fire    ·      ·
```

Two things follow from "per crossing" rather than "per widget". A `Countdown`
whose `Until` is already past on its first pass fires immediately — the right
reading of a deadline restored from disk while the app was closed, and why a
zero `Until` raises `comps.ConcernCountdownUntilUnset` rather than passing
quietly. And moving `Until` forward re-arms it, so a restart is
`Until: time.Now().Add(d)` and nothing else.

`OnDone` is a display-grade signal, not a scheduler: it only fires while the
widget is rendered and the app is running, and it is late by up to one tick.
Something that must happen whether or not anyone is looking belongs in the
[`alarm`](api/alarm.md) package and `hooks.UseAlarms`.

**The stopwatch's two numbers belong to the caller.** A stopwatch that knew
only when it started could not be paused — the instant the finger lifts is
recorded nowhere. So the state is the pair every stopwatch keeps, and the
reading is their sum, `Elapsed + (Running ? now − Since : 0)`, which makes the
four moves plain assignments:

```go
start   since.Set(time.Now());                    running.Set(true)
pause   banked.Set(banked.Get() + time.Since(since.Get())); running.Set(false)
resume  since.Set(time.Now());                    running.Set(true)
reset   banked.Set(0);                            running.Set(false)
```

Held here instead, it would be state the app cannot save, restore or show
elsewhere — and a running stopwatch is exactly what an app wants to keep
across a screen change. The same reasoning keeps [`SliderRow`](#sliderrow)'s
drag draft with its caller.

**The two roundings go opposite ways, and both are conservative.** The
countdown rounds *up*, so it never says you have less time left than you do;
the stopwatch *truncates*, so it never claims more elapsed time than has
passed and its first second reads `0:00`. Neither shows hundredths: a
`core.State` change requests a render of the whole tree, and two animated
digits are not worth a hundred passes a second.

**The reading is the phone timer's format** — `M:SS` under an hour, `H:MM:SS`
at or over one, with days folded into hours (`26:00:00`). `Format` overrides
it and receives the duration already rounded the way the widget rounds it, so
a custom format cannot disagree with the widget about which second is on
screen.

**They announce a sentence, not punctuation.** The digits are one element with
`RoleImg` and a spoken label — "4 minutes 12 seconds remaining", "Time is
up", "1 minute 15 seconds elapsed" — for `DigitalClock`'s reason: read as
text, "4:12" is punctuation. The label is not a live region, so nothing is
announced every second; a caller who wants that puts the timer beside its own
`core.RoleStatus` text.

## AudioPlayer

The transport for one track on the app's one player.

```go
comps.AudioPlayer{
    Track:    core.AudioTrack{URL: sermon.URL, Title: sermon.Title, Artist: sermon.Speaker},
    Rates:    []float64{1, 1.25, 1.5, 2},   // adds a speed button that cycles
    ShowStop: true,
}
// Sunday, 14 March
// Pastor Ade
// ●━━━━━━━━━○───────────────
// 12:04                41:30
//    [−15s] [Pause] [+15s]
//     [Speed 1.25×] [Stop]
```

- **A view of the singleton, not a player.** core's audio is one stream for
  the whole app. The widget asks whether the loaded track is its own (same
  URL). If it is, the controls drive it. If not, it shows its track idle,
  Play loads it, and the rest is disabled. One per episode screen is fine.
- **The scrub reading is the widget's.** The elapsed time follows the finger
  and the seek is sent once, on release. This is the opposite of
  [SliderRow](#sliderrow), because the drafted value is the host's position,
  not the app's, and the reading is the point of scrubbing.
- The second line is the Artist, except "Loading…" and "Couldn't play: …"
  for this track.
- A `RoleGroup` named by the title. The seek bar is "Position", with its value
  spoken as "12:04 of 41:30" once a duration is known. It states no range
  before then, because a 0-to-0 range is one Compose cannot express.
- `Waveform []float64` draws the track's peaks above the seek bar as a
  [Waveform](#waveform) filled to the position (and to the finger while
  scrubbing). It is hidden from screen readers and is not a control.
- `SkipSeconds` is 15 by default; a negative value drops the skip buttons.
  It holds hooks: render it unconditionally. `ConcernAudioPlayerNoTrack` for a
  missing URL.

## MessageBubble

One message in a conversation, held to one side of the row.

```go
comps.MessageBubble{Text: "Did you see?", Sender: "Ana", Time: "10:41"}
comps.MessageBubble{Text: "Not yet", Mine: true, Time: "10:42"}
```

- **Mine** is on the trailing side in `Primary`, with the ink chosen by
  contrast. **Theirs** is on the leading side in `Surface` with a `Border`
  hairline, because the palette has no muted container tone (Banner's answer
  to the same gap). Bubbles are capped at 80% of the row.
- `Sender` is drawn on theirs only. Set `Continued` on the second and later of
  a run from one sender: the line is not drawn, and the name is still spoken.
  `Time` is a string you format.
- One spoken stop per message, named who-what-when ("Ana, Did you see?,
  10:41"). The reader's own messages are named "You, …", and `MineLabel`
  localizes that. Put the bubbles under a `core.RoleLog` container.
- `Style` goes on the outer row, where a gap between messages belongs.
- **The tail** is the bottom corner on the sender's side, nearly square
  (`core.CornerRadii`): bottom-right on the reader's own, bottom-left on
  theirs.
- **Not a thread**: a whole conversation is `MessageThread`, below.

## MessageThread

A conversation: bubbles in a `core.List` that opens on the newest message,
loads older ones when the reader scrolls back to the top, and keeps the
reader's place while they land.

```go
comps.MessageThread{
    Messages:    msgs,        // []comps.ThreadMessage, oldest first
    OnLoadOlder: loadOlder,   // nil once there is nothing older
    Loading:     fetching,
}
```

- Every `ThreadMessage` needs a stable `Key` (a server ID, not an index). The
  key is how each host finds the row the reader was looking at after a
  prepend: the reconciler pairs rows by position, so without keys a prepend is
  new text in every row.
- `OnLoadOlder` runs when the reader reaches the top, once per row count
  (`core.OnStartReached`'s guard), so repeated reports of the same top load
  one page, and a page that comes back empty leaves the guard shut.
- The loading and start captions are a line above the list, never a row in
  it, so the list's first row is always a message.
- A message that arrives while the reader is at the end is shown; one that
  arrives while they have scrolled back leaves them where they are.
- The list is an unnamed `core.RoleLog`. A name on a container folds its
  children into one stop on iOS.
- Under it are two `core.List` props any list can use: `core.StartAtEnd()`
  and `core.OnStartReached(fn)`. See their doc for what each host does.

## TypingIndicator

Three dots in a theirs-coloured bubble, darkening one after another while
somebody is writing.

```go
comps.TypingIndicator{Visible: anaTyping, Who: "Ana"}
```

- **Always render it, and switch it with `Visible`.** The widget owns two hook
  slots (which dot is dark, and the interval that moves it), so a `core.If`
  around it is a conditional hook. Hidden is `Display none` with the interval
  paused (`hooks.UseIntervalWhile`), which costs no render passes. The zero
  value is hidden.
- The dots fade by **opacity**: all three are `TextPrimary`, the dark one at
  `core.Opacity(1)` and the others at 0.4, each with a `core.Transition`, so
  no second tone has to read on the bubble in every theme. Go steps the phase
  every 400ms and the platform draws the frames.
- One `RoleStatus` stop named by `Label`: `Who + " is typing"` by default, or
  "Typing". Set `Label` to localize, or for a group ("Ana and Rui are
  typing"). `Caption` draws the same text beside the dots.
- Put it after the transcript, not inside the `RoleLog`: nested, the dots
  would be recorded as a message.
- Under Reduce Motion each host drops the transition, and the dots change
  colour without the ease. Not yet looked at on a device with the setting on.

## ReactionBar

Emoji chips with counts under a message. A tap toggles the reader's own.

```go
comps.ReactionBar{
    Reactions: []comps.Reaction{
        {Emoji: "👍", Count: 3, Mine: true, Label: "thumbs up"},
        {Emoji: "🎉", Count: 1, Label: "party popper"},
    },
    OnToggle: func(emoji string) { toggleReaction(msgID, emoji) },
}
```

- **The caller holds the counts.** A reaction is server state, so the widget
  is stateless, like `Stepper`: it draws `Reactions` and reports the tapped
  emoji. Whether the tap adds or removes is the caller's to decide.
- It is a `ChipStrip` of `Chip`s: `Mine` is the selected chip, and the look
  and the selected-state announcement are `Chip`'s.
- `Reaction.Label` is the emoji's spoken name, because no platform names an
  emoji reliably: "thumbs up, 3 reactions". The reader's own is the chip's
  selected state, not words in the name.
- A `Count` of zero is not drawn, so a caller can pass its whole emoji table.
  Zero with `Mine` set is a caller's bug and is drawn, so it gets seen. A bar
  with nothing to draw is `Display none`.
- `Trailing` is the slot for a caller's own "+" chip. There is no emoji
  picker: that is an anchored popover, which is blocked on a renderer.
- `Disabled` draws the chips inert. No `OnToggle` and not `Disabled` raises
  `ConcernReactionBarInert` in debug builds.

## Poll

A question whose options turn into labelled result bars after a vote.

```go
comps.Poll{
    Question: "Tabs or spaces?",
    Options: []comps.PollOption{
        {Label: "Tabs", Votes: 5, Mine: true},
        {Label: "Spaces", Votes: 3},
    },
    OnVote: func(i int) { castVote(pollID, i) },
}
```

- **Asking** until an option is `Mine`: full-width outlined buttons, and
  `OnVote` reports the tapped index. The caller records the vote and passes
  the option back with `Mine` set, for `ReactionBar`'s reason. The vote is a
  bool per option and not an index on the poll, because an `int` field would
  default to 0 and open the poll already voted for its first option.
- **Showing results** after that, or with `ShowResults` (a closed poll): each
  option is its label, its share and a `ProgressBar`, with the reader's choice
  checked and in the on-light `Primary`, and the total underneath.
- The shares **always total 100**, by the largest-remainder method: three
  options at one vote each read 34 / 33 / 33, not 33 / 33 / 33. With no votes
  every share is 0%. The bars use the true fractions.
- Each result is one spoken stop: "Tabs, 63 percent, 5 votes, your choice".
  `ChoiceLabel` localizes the last part.
- Options keep the caller's order; they are not sorted by share.
- Asking with no `OnVote`, and neither `ShowResults` nor `Disabled`, raises
  `ConcernPollInert` in debug builds.

## ExpandableText

Body text capped at a few lines, with a Read more that opens it in place.

```go
comps.ExpandableText{Text: episode.Summary, Lines: 3}
```

- **The toggle shows past a length, not a measurement.** Whether the cap cut
  anything is a rendered height, which no host reports. The toggle shows
  when the text has more than `ToggleAfter` runes: `Lines` × 40 by default,
  negative to always show it.
- **A cap only comes with a toggle.** Short text is drawn in full, because a
  cap with no way to lift it would hide the end of the text for good.
- The open state is the widget's (one hook): render it unconditionally.
- The text node always carries the whole string. The toggle stays named "Read
  more", with `aria-expanded` saying which way it is.

## QRCode

A string drawn as a QR Code, encoded in Go.

```go
comps.QRCode{
    Data:  "cats://pair?t=9f2c1a&host=studio.local",
    Label: "Scan to pair this device",
}
```

| Field | Meaning |
|---|---|
| `Data` | encoded in byte mode, so any string is legal |
| `Size` | the box's side in px, quiet zone included; `0` means 160 |
| `Level` | `ECLow`/`ECMedium`/`ECQuartile`/`ECHigh`; the zero value is `ECMedium` |
| `Quiet` | the light margin in modules; `0` means the standard's four, a negative value means none |
| `Label` | names the code to assistive tech; empty says "QR code" |

**The encoder is ours, in `internal/qr`.** `github.com/skip2/go-qrcode` is MIT
and correct, and it would still have been the first third-party dependency any
widget in this framework required — `go.mod` holds nothing outside this
author's own packages and the gomobile toolchain. A QR encoder is a closed,
fully specified algorithm with published test vectors, which is the kind of
thing that is cheaper to own than to track: byte mode, versions 1–40, all four
levels, the eight masks with the standard penalty scoring, and no `image/png`
or bitmap model that a `Canvas` would only have to undo. The package's tests
check it against the standard's own printed format and version bit strings,
its published byte capacities and alignment coordinates, the defining
Reed-Solomon property (every codeword vanishes at the generator's roots), and
a round trip back out of the finished grid for all forty versions.

**One path, not a rectangle per module.** A version-10 symbol has some three
thousand modules. Three thousand child nodes is the reconciler's worst case
for a drawing that is either identical between passes or wholly different —
and separate shapes are antialiased against each other, which leaves hairlines
between adjacent modules that a decoder's binarizer can read as light. A single
filled path has no interior seams. Within a row, consecutive dark modules merge
into one rectangle, which costs one comparison per module and typically halves
the path.

**The quiet zone is inside the box.** `Size` is the whole square, so the symbol
is `Size × n/(n+2·Quiet)` across. Putting the margin outside would make the
widget's footprint depend on how long the data turned out to be; a caller lays
out a 160px square and gets one.

**Colour is not themed, and that is the point.** A QR code is read by a camera,
and every decoder's binarizer assumes dark modules on a light field. The widget
uses the theme's own ink and surface when those *are* dark-on-light with room
to spare — 0.6 luminance and 0.15, so a light theme's code sits in the page
rather than on a hard white patch — and otherwise falls back to black on white.
In a dark theme that means a white square, which is what every banking and
payment app shows, for this reason. There is no `Foreground` or `Background`
field: every colour a caller could pass is either the pair already chosen or a
worse one, and an unscannable code fails silently — it looks exactly like a
working one.

**Data too long draws nothing.** The limit is a version-40 symbol: 2953 bytes
at `ECLow`, 1273 at `ECHigh`. Past it the widget keeps the box, so the screen
does not reflow when the data is fixed, draws no symbol, and in debug builds
reports a `comps.ConcernQRDataTooLong` concern. There is no half of a QR code
worth showing: a truncated one still scans, just to the wrong thing.

**The default level is `ECMedium`, not `ECHigh`.** The failure a code on a
screen actually faces is not damage — the glass is pristine — but module size.
A higher level spends more of the symbol on redundancy, so at a fixed drawn
width it means a larger symbol and smaller modules, which is what a phone
camera struggles with. Raise it when a logo will be laid over the middle, or
when the code will be printed and handled.

**The data is never spoken.** A reader announcing a 300-character URL one
character at a time helps nobody, and a person who needs the link needs it as
a link. `Label` should say what scanning it will *do*; put the underlying
action on screen as well where you can.

Encoding runs on every render pass — version choice, block layout, then scoring
all eight masks — which is about 0.2 ms for a link-sized payload. That is under
a hundredth of a frame and not worth caching for a screen that shows a code and
waits. For a code inside a tree that re-renders every frame, `core.Cached` fits:
`QRCode` holds no hooks and registers no callbacks.

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
which is what [`core.RoleLink`](concepts/styling-and-theming.md#accessibility-props)
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

**The toolbar is a toolbar.** The row carries `core.RoleToolbar` and is named by
`ToolbarLabel`, which defaults to "Editing". In the browser it is one tab stop,
and the arrow keys move between its buttons. It holds only the buttons the
widget builds, which is what lets it declare a keyboard container role without
making nested composites reachable, so do not wrap the editor in a toolbar of
your own.

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

The strip is a toolbar. It carries `core.RoleToolbar` and is named by
`bar.Label`, which defaults to "Formatting", so a reader hears "Formatting,
toolbar". In the browser it is one tab stop, and the arrow keys move between its
buttons. The strip holds only buttons built from `RichToolItem` data, which is
what lets a widget declare a keyboard container role without making nested
composites reachable. Do not wrap the editor in a toolbar of your own: the strip
already is one, and a toolbar inside a toolbar is a nested composite. A
read-only editor disables every button, so its strip has no tab stop.

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

## RichTextView

A `richtext.Doc` drawn to be read: the document `RichTextEditor` edits, with no
editor.

```go
comps.RichTextView{Doc: note.Get()}
comps.RichTextView{Doc: doc, OnLink: func(url string) { open(url) }}
```

Each block is one [`core.Paragraph`](api/core-views.md#func-paragraph): the block's runs as
runs of one flow of text, so a sentence with a bold word and a link in it wraps
as one sentence. Headings are bold at the editor's own scale (1.6, 1.35 and
1.15 of Body) and announced as headings of their level. Consecutive list items
are one list to a screen reader, numbered from 1 per list. A quote sits beside
a rule in the secondary ink, and a code block is monospace in a Surface box.

A run's link is a tappable run in `comps.Link`'s colour, underlined. `OnLink`
receives the URL; nil opens it with `core.OpenURL`.

**When to use it rather than a read-only editor.** A `RichTextEditor` with
`ReadOnly` and no toolbar also displays a document, but it hosts a platform
editor per instance (`UITextView`, `EditText`, a `contenteditable`). A list of
fifty notes wants fifty paragraphs of text, which is what this draws. It holds
no hooks, so it can be rendered inside a `core.If`.

`comps.Link` has the inline form too: `comps.Link{Text: "terms", URL: u}.Span(ctx)`
is a run for a `core.Paragraph`, in the link's colour, underlined, with the
same tap.

## Writing your own

The package doc (`comps/doc.go`) is the reference for the idiom. In
short:

1. A struct with named fields; `core.View`-typed fields for slots.
2. `Render(ctx)` builds on core containers/widgets and returns their node.
3. Read the theme; accept `Style []core.StyleProp` for overrides.
4. If the widget owns state, document its hook obligations (see Accordion).
5. Give it a focused test — and if it's extracted from app code, pin the
   rendered output against the original with `htmlout.ExportHTML`.
