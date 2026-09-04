# Views & Composition

Everything on screen is a `View`:

```go
type View interface {
    Render(ctx *Context) *Node
}

type ComponentFunc func(ctx *Context) *Node // adapts a function to a View
```

This is the `http.HandlerFunc` pattern applied to rendering: a component is
either a function (`ComponentFunc`) or any struct implementing `Render` (the
idiom the [widget library](../components.md) uses). Components compose by
calling each other; there is no registration step.

## Containers

The flex-style containers — `Row`, `Column`, `Card`, `Box`, `List` — share
one argument contract: a mixed variadic list of **style props**, **behavior
props**, and **child views**, in any order.

```go
core.Row(
    core.Gap(8),                       // style prop
    core.OnClick(func() { ... }),      // behavior prop → applies to the Row
    core.Text("left"),                 // child
    core.Text("right", core.FlexGrow(1)),
)
```

- `Row` / `Column` — horizontal / vertical flex stacks, themed base padding.
- `Card` — a themed surface (background, radius, shadow).
- `Box` — the unopinionated container: a `Column` with no theme base. It
  stacks its children vertically like one, on all four targets; it is not an
  overlay. (It drew as a Compose `Box` / SwiftUI `ZStack` on the natives until
  that was fixed, so two children landed on top of each other on device and
  down the page in a browser.) Reach for it when you want a styled container
  and none of the theme's opinions.
- `List` — the virtualized sibling of `Column`: children are laid out lazily
  by the native renderer (Compose `LazyColumn`, SwiftUI `LazyVStack`), so a
  thousand-row feed composes only what is on screen. Use `Column` + `Scroll`
  for short content, `List` for long data-driven collections — and give
  every row a stable identity with `Keyed`.
- `Scroll` — a vertically scrolling region, taking the same argument list as
  the other containers (it used to take a bare `...View`; existing calls are
  unaffected). Like `Box` it has no theme base, so wrapping a screen in one
  does not inset it.
- `SafeArea`, `Spacer(px)`, `Divider(height, color)`, `Fragment`.
  (`Divider` force-applies `Margin(8)`; for a rule inside a list use
  `components.Separator`, which leaves spacing to the caller and defaults
  its own hairline tint.)

Behavior props (`OnClick`, `OnTouch`, `OnLongPress`, `OnFocus`, `OnBlur`, or
the generic `On(event, fn)`) register their callbacks in argument order,
before any child renders — a container's callback IDs always precede its
children's.

The input family (`Input`, `InputWithSubmit`, `InputPassword`, `NumericInput`,
`TextArea`, `Checkbox`) takes the same mixed argument list, minus children —
which is what makes [`OnFocus`/`OnBlur`](events.md) reachable on the nodes
that actually receive focus. A builder's own callbacks (its `onChange`, an
`onSubmit`) always take the lower IDs, so no argument a caller writes can move
them. Passing a `View` to one of these is a [debug-mode](debug-mode.md)
concern: a leaf has nowhere to put a child.

### The software keyboard

`core.KeyboardAware()` is a prop, not a container: the node it is applied to
ends where the software keyboard begins, for as long as the keyboard is up.

```go
core.Scroll(core.KeyboardAware(), form)   // the viewport shortens
```

On a **scrolling** node (`Scroll`, `List`) that shortens the viewport. The
content does not move by itself — but with the viewport now ending above the
keyboard, the platform's own scroll-the-focused-field-into-view has somewhere
visible to put the field, which it cannot do while the viewport still claims
the rows the keyboard is sitting on. This is the form case, and
`components.Screen{Scroll: true, KeyboardAware: true}` is the short way to say
it (`examples/signup`).

On **any other** node it lifts that subtree whole — the case for a screen with
something docked at the bottom, a chat composer or a checkout bar, which sits
outside any scrolling region by construction and is the one thing the keyboard
covers (`examples/chat`).

| | |
|---|---|
| Android | `Modifier.imePadding()`, injected at the one funnel every node passes through. Needs the window to have stopped fitting the system windows itself — the demo activity calls `enableEdgeToEdge()` and the manifest sets `windowSoftInputMode="adjustResize"`. Without both, the platform resizes the whole window instead and the prop reads a consumed, zero-height inset. |
| iOS | SwiftUI treats the keyboard as its own safe-area region and insets a `ScrollView` for it *by itself*, so the shrink is the platform default with or without the flag. What the flag adds is `.scrollDismissesKeyboard(.interactively)` on the two scrolling node types. |
| HTML / WASM | Nothing — a browser has no software keyboard to inset for. |

That asymmetry is why this is a flag rather than something `Scroll` always
does, and it is also why `SafeArea` does not carry it: Android's
`WindowInsets.safeDrawing` bundles the keyboard in with the system bars, so
applying it there would resize every screen whole *and* consume the inset
before an inner region could ask for it. The renderer subtracts the IME from
that set deliberately — the same split SwiftUI makes.

Nothing here dismisses the keyboard on a tap outside — that is a separate
request, [`core.DismissKeyboard`](events.md#setting-focus), and keeping the
two apart is the point: this prop decides which region yields the space, and
a chat composer that wants the inset emphatically does not want a stray tap
closing the keyboard between messages. The iOS drag-to-dismiss above is the
platform's own gesture, not a handler of ours.

## Leaves

| Widget | Signature (abridged) |
|---|---|
| `Text` | `Text(content, styleProps...)` |
| `Button` | `Button(label, onClick, props...)` — also `ButtonWithEvent(label, event, fn, ...)` |
| `Input` | `Input(value, placeholder, onChange, ...)` — also `InputWithSubmit`, `InputPassword`, `NumericInput`, `TextArea` |
| `Checkbox` | `Checkbox(checked, onToggle, ...)` |
| `Slider` | `Slider(value, min, max, onChange, ...)` with `OnSliderChangeEnd(fn)` (fires once on release — the one a seek bar acts on) and `SliderStep(s)` |
| `Image` | `Image(src, styleProps...)` |
| `TextGrid` | `TextGrid(rows []GridRow, props...)` — a monospace grid of styled runs (a terminal pane, a log tail); each `GridRun` has `Text`, `Fg`, `Bg` and `Attr` bits (`GridBold`, `GridDim`, `GridItalic`, `GridUnderline`, `GridStrike`). Rows are children, so a changed row is one patch |
| `CameraView` | `CameraView(props...)` with `OnCapture`, `WithOverlay`, `SetFacing`, ... |
| `Modal` | `Modal(Visible(b), OnDismiss(fn), Backdrop(color), ...)` |
| `TabView` | native tab bar — prefer the [`components.Tabs`](../components.md#tabs) facade |

Inputs are **controlled**: you pass the current value in and receive changes
through `onChange`; the value on screen is whatever your state says it is.
The [tutorial](../tutorial-todo.md) covers the echo/rewrite contract in depth.

## Conditional rendering

Render logic stays declarative with the conditional helpers:

```go
core.If(user != "", core.Text("Welcome, "+user))

core.IfElse(loading.Get(),
    core.Text("Loading…"),
    core.Text("Ready"))

core.Match(status.Get(),
    core.Case("ok",    core.Text("✅")),
    core.Case("error", core.Text("❌")),
    core.Default[string](core.Text("…")))

core.MatchBool(
    core.When(isGuest, guestBanner),
    core.When(isAdmin, adminPanel),
    core.Otherwise(memberView))
```

### Fragments are transparent

`Fragment` — and the `Theme` node `WithTheme` produces — are **grouping nodes**:
they hold children and carry no style of their own. `For` wraps everything it
generates in one, so a loop inside a container arrives as a single child:

```go
core.Row(
    core.Gap(8),
    core.For(labels, func(l string, i int) core.View { return chip(l, i) }),
)
```

All three targets inline that grouping node into the parent's layout, so the
three chips are what the `Row` spaces — not the `Fragment`. On iOS it is a
SwiftUI `Group`, on Android the children are emitted straight into the parent's
scope, and the HTML exporter writes the children with no wrapper element.

That last one was not always true. The exporter used to emit a `<div>` for every
grouping node, and a styleless `div` inside a flex container is a real flex item:
it became the container's only child, so `gap`, `flex-direction` and alignment
stopped at the wrapper instead of reaching the children. The HTML therefore
disagreed with both natives — visibly, in `examples/todoapp`, whose filter bar
lost its 8pt gap in the browser while keeping it on device. Grouping nodes now
emit their children directly.

The node still exists in the tree even though it draws nothing, which is the
distinction the next section turns on.

### One optional item: `MaybeProp`

`If` returns a view, and a false `If` returns an **empty `Fragment`** — a real
child node. The reconciler walks and diffs it on every pass, and it occupies a
child index, so anything addressing children by position counts it. And `If` is
`View → View`, so there was no expression form at all for an optional *style*
or *behavior* prop.

The empty `Fragment` does not *draw* anything: a grouping node with no children
renders no box on any of the three targets. (It briefly appeared to, because the
HTML exporter wrapped every grouping node in a `div` — a real flex item that
swallowed the parent's `gap`. That is fixed; see
[Fragments are transparent](#fragments-are-transparent) below.) The cost `MaybeProp`
removes is a node, not a gap.

`MaybeProp` covers both. It returns its argument when the condition holds and
`nil` otherwise, and the container builders skip a `nil` item outright — no
node, no slot, no style:

```go
core.Column(
    core.UseStyle(bubble),
    core.MaybeProp(!mine, core.Text(from)),          // an optional child
    core.MaybeProp(selected, core.Padding(12)),      // an optional style prop
    core.MaybeProp(onTap != nil, core.OnClick(onTap)), // an optional handler
    core.Text(body),
)
```

Reach for it instead of accumulating a `[]core.PropsAndChildren` by hand. Two
limits:

- The argument is **evaluated eagerly**, like any Go argument. That is free for
  the prop constructors (`core.Text` only builds a closure), but `MaybeProp` is
  not a substitute for an `if` around work that would be expensive or panic on
  the false path.
- It returns `PropsAndChildren` (i.e. `any`), so it only fits the container
  builders — `Row`, `Column`, `Card`, `Box`, `List`. `Text` and `Button` take
  `...StyleProp` and will not accept it.

Use `If` where the alternative is a whole branch of the tree; use `MaybeProp`
for a single optional item among siblings.

!!! tip "Debug mode catches the silent drop"
    Because `PropsAndChildren` is `any`, a container accepts *anything* at
    compile time and quietly ignores what it cannot classify — a bare
    `core.Style` where `core.UseStyle(style)` was meant is the classic case.
    [Debug mode](debug-mode.md) reports it as an
    `unknown-container-item` concern naming the container and the dropped type.
    `MaybeProp`'s `nil` is exempt: that is the contract, not a mistake.

!!! warning "Conditionals hide views, not hooks"
    `If`/`Match` decide which **view** renders — but a view that calls
    `NewState` and renders only sometimes violates the
    [rules of hooks](state-and-hooks.md#the-rules-of-hooks). Keep hook calls
    unconditional; let conditionals choose between hook-free views, or give
    each branch its own `ctx.Scope`.

## Lists and keys

`For` renders a slice into a sibling list:

```go
core.List(
    core.For(todos, func(t Todo, i int) core.View {
        return core.Keyed(fmt.Sprintf("todo-%d", t.ID), todoRow(t))
    }),
)
```

`Keyed(key, child)` stamps an identity onto the rendered node. The
reconciler uses it to detect when the occupant of a list slot changed —
diffing across different keys would leak state between logically distinct
rows (the classic un-keyed-list bug). Keys must be **unique among siblings**;
[debug mode](debug-mode.md) flags duplicates.

Derive keys from data identity (an ID), never from the index — deletion
shifts indices.

## Navigation

`Navigator` renders a route stack; the stack lives on the context tree, so
one app has exactly one:

```go
core.Navigator(HomeScreen)          // root view

core.Push(ctx, DetailsScreen)       // push a route
core.Pop(ctx)                       // back
core.Replace(ctx, OtherScreen)      // swap the top
core.PopToRoot(ctx)                 // unwind, keeping the root's state
core.Reset(ctx, HomeScreen)         // clear the stack to one fresh route
```

Routes are `func(*core.Context) core.View` — plain view functions. Each stack
frame renders into its own hook namespace, so routes may use hooks freely and
a frame's state is discarded when it leaves the stack. See
[Navigation](navigation.md) for the full picture.

!!! note
    `core.Reset(ctx, route)` (navigation) is unrelated to `ctx.Reset()`
    (the internal hook-cursor reset performed at each pass boundary).

## Composition patterns

Build screens out of small functions of their inputs:

```go
func todoRow(t Todo, setDone func(int, bool)) core.View {
    return core.Keyed(fmt.Sprintf("todo-%d", t.ID), core.Row(
        core.Checkbox(t.Done, func(v bool) { setDone(t.ID, v) }),
        core.Text(t.Title, core.FlexGrow(1)),
    ))
}
```

Rows like this are **pure functions of their data** — no hooks — which keeps
them safe to create and discard freely inside `For`. State lives higher up
and flows down as values + closures; see
[State & Hooks](state-and-hooks.md#where-state-should-live).

For reusable widgets with many optional knobs, prefer the struct idiom of the
[widget library](../components.md) — named fields scale where positional
arguments do not, and `View`-typed fields make natural slots.
