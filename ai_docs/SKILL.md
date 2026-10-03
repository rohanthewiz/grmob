---
name: grmob-native-mobile-go
description: Build native mobile apps (Android, iOS), browser apps and HTML exports in pure Go with the GrMob framework — declarative views, positional hook state, and a tree diff that every host applies as patches. Use when starting a GrMob app (`grmob new`), writing screens, state, forms, navigation or tests for one, or building it for the browser, Android or iOS. For authoring a reusable widget, see the grmob-component skill.
---

# GrMob — AI Agent Documentation

> Native mobile UI in pure Go. No templates, no XML, no Swift, no Kotlin, no JS.

```go
import "github.com/rohanthewiz/grmob/core"
```

## The mental model

An app is a **function from a context to a view**, called again on every pass.
It returns a description of the UI; it never touches the screen.

```
App(ctx) ──▶ View.Render(ctx) ──▶ *core.Node tree
                                        │
                              reconcile.Diff(old, new)
                                        │
                                   []reconcile.Patch
                          ┌─────────────┼─────────────┬──────────────┐
                      Compose        SwiftUI     JS runtime      htmlout
                      (Android)       (iOS)      (browser)     (HTML file)
```

Four consequences worth holding on to:

1. **The same app function runs on every target**, including inside a plain Go
   test. There is nothing platform-specific in application code.
2. **Nodes are immutable once rendered.** The diff treats pointer equality as
   proof a subtree is unchanged, so mutating a node after its pass has returned
   makes the change invisible rather than merely late.
3. **State lives in the context, addressed by call order** — not by name. This
   is the single biggest source of bugs; see the rules below.
4. **`State.Set` is the whole update path.** Write state, and the next pass's
   diff reaches the screen. Never try to "refresh" anything by hand.

## Starting an app

An app lives in **its own module**, made by the `grmob` command. Do not copy
this repository's `build.sh`, `wasm/` or shells into an app: they assume the
grmob module root and go stale silently.

```sh
go run github.com/rohanthewiz/grmob/cmd/grmob@latest new myapp \
    -name "My App" -id com.example.myapp     # both optional; derived from the dir
cd myapp
./dev.sh          # http://localhost:8080 — rebuilds and hot-swaps on every save
go test ./app     # the fastest loop: no browser, no device
```

`new` flags: `-module` (default: the dir name), `-name` (launcher label),
`-id` (Android applicationId and iOS bundle ID — letters and digits per
segment, at least two segments), `-grmob <version|master>`,
`-replace <path to a grmob checkout>` (a `replace` directive, for working
against local framework changes), `-no-build`. The target directory must be
empty (a `.git` is allowed).

What it writes:

```
app/app.go        root view + init registration + AppName (every target mounts this)
app/app_test.go   render.Manager test with debug mode and tap/shows helpers
wasm/main.go      browser host: webhost.Run(nil, app.App)
wasm/index.html   host page (rendered once; the app may edit it)
build.sh          browser build; re-copies grmob-runtime.js from go.mod's grmob
dev.sh            go run github.com/rohanthewiz/grmob/serve -dev
grmob.json        {"name", "id"} for the native shells
```

The rule the scaffold keeps: **anything grmob owns comes from the grmob
version in `go.mod`, at build time.** The runtime JS is copied by every build
and never committed or edited — a runtime bug is a grmob bug; fix it upstream
and `go get github.com/rohanthewiz/grmob@<version> && go mod tidy`. The two
exceptions are copied into the app once, because an app edits them: the host
page (`grmob web` warns when it differs from go.mod's page; `-refresh`
re-renders it) and the native shells (a build warns when go.mod has moved past
the version they came from; `-refresh` re-copies). Commit before `-refresh` —
it overwrites edits.

Every later command runs from anywhere inside the app (it walks up to
`grmob.json`):

```sh
go run github.com/rohanthewiz/grmob/cmd/grmob doctor          # which targets this machine can build
go run github.com/rohanthewiz/grmob/cmd/grmob web             # one-off browser build (+ host-page check)
go run github.com/rohanthewiz/grmob/cmd/grmob android -install # APK; Android SDK + NDK, JDK 17+
go run github.com/rohanthewiz/grmob/cmd/grmob ios -run        # simulator; Xcode + xcodegen (-open: Xcode)
```

The first native build vendors grmob's shell into `android/` or `ios/`; after
that the icon, permissions and manifest there are the app's to change.

### The working loop

1. Write views in `app/` — one file per screen is the usual split; screens are
   plain functions taking `ctx` and the state they need.
2. Keep `func init() { mobile.Register(...) }` and `AppName()` in the package
   `grmob android|ios` binds (`./app`); see the next section for why.
3. Test each behaviour through `render.Manager` with debug mode on (the
   scaffold's `app_test.go` has `tree` / `find` / `tap` / `shows` helpers to
   extend), and assert `core.Concerns()` is empty.
4. Check it by eye in `./dev.sh`, then on a native target.
5. Before hand-rolling a card, chip, list row or form field, look in
   [the widget library](#the-widget-library).

## A complete app

This is the shape `grmob new` writes into `app/app.go`.

```go
package counter

import (
    "fmt"

    "github.com/rohanthewiz/grmob/core"
    "github.com/rohanthewiz/grmob/mobile"
)

// Registers with the bridge at link time. This is the whole integration
// contract for the native shells and the browser host.
func init() { mobile.Register(core.NewContext(), App) }

// gomobile cannot bind a func-typed symbol, so without an exported function of
// a bindable shape the linker drops this package — and the init above with it.
// Every app package needs one such symbol.
func AppName() string { return "Counter" }

func App(ctx *core.Context) core.View {
    count := core.NewState(ctx, 0)

    return core.Column(
        core.Padding(24),
        core.Gap(12),
        core.Text(fmt.Sprintf("Count: %d", count.Get()), core.FontSize(28)),
        core.Row(
            // The theme insets every Column and Row (16 either side in
            // DefaultTheme): right for a screen band, wrong for a Row
            // nested in an already padded Column. Padding(0) clears it.
            core.Padding(0),
            core.Gap(8),
            core.Button("−", func() { count.Set(count.Get() - 1) }),
            core.Button("+", func() { count.Set(count.Get() + 1) }),
        ),
    )
}
```

## Views and containers

A view is anything with `Render(ctx *core.Context) *core.Node`. Containers take
a single variadic `...core.PropsAndChildren` (which is `any`) and sort styles
from children at runtime — so **props and children are interleaved freely**:

```go
core.Column(
    core.Padding(16),          // a StyleProp
    core.Text("Title"),        // a View
    core.Gap(8),               // another StyleProp, after a child: fine
    core.Text("Body"),
)
```

| Containers | Leaves | Structure |
|---|---|---|
| `Column` `Row` `Box` `Card` `Scroll` `List` `ZStack` `SafeArea` | `Text` `Image` `Divider` `Spacer` | `Fragment` `Keyed` `For` `If` `IfElse` `Match` |

Leaf widgets take **typed positional arguments first**, then styles:

```go
core.Text("hello", core.FontSize(16), core.TextColor("#333"))
core.Button("Save", onSave)
core.Input(value, "you@example.com", onChange)
core.Checkbox(checked, func(b bool) { /* ... */ })
core.Slider(v, 0, 100, func(f float64) { /* ... */ })
core.Select(value, opts, onChange)
core.Image("logo.png", core.Width("100%"))
```

### Composition

A **component** is a View written in Go out of other Views; a **widget**
is a reusable struct component (every `comps` type); a **primitive** is a
core constructor a host draws itself. The definition and the contract a
component keeps are on `core.View`; building a reusable one is the
**grmob-component** skill (`ai_docs/SKILL-component.md`).

Extract a screen into a function; there is no component registration step.

```go
func header(title string) core.View {
    return core.Row(core.Padding(12), core.Text(title, core.FontWeight(core.Bold)))
}
```

Use `core.ComponentFunc` when a piece needs its **own** render function
value. Its body runs when the tree renders it, not when it is constructed,
so its hooks are claimed at that point in the tree — still in the caller's
positional sequence, since containers render children on the same context:

```go
core.ComponentFunc(func(ctx *core.Context) *core.Node {
    n := core.NewState(ctx, 0)
    return core.Text(fmt.Sprint(n.Get())).Render(ctx)
})
```

## State and the rules of hooks

`core.NewState[T](ctx, initial) State[T]` claims the slot at the context's
current cursor. Slots are **positional**: identified by the order the calls
happen in, so the Nth `NewState` call of a pass is the Nth slot.

```go
name  := core.NewState(ctx, "")   // slot 0
email := core.NewState(ctx, "")   // slot 1
```

**The rules, and what breaks when they are broken:**

```go
// WRONG — the state call is conditional, so every hook after it shifts onto
// a neighbour's slot on the pass where the condition flips.
if expanded {
    detail := core.NewState(ctx, "")
}

// RIGHT — call unconditionally, use conditionally.
detail := core.NewState(ctx, "")
if expanded { /* read detail */ }
```

- Call every hook on **every** pass, in the **same order**.
- Never call a hook inside `if`, `for`, `switch`, or after an early return.
- A conditional **view** is fine (`core.If`) — it is a conditional **hook** that
  is not. Widgets that hold hooks (`comps.Accordion`, `DatePicker`) count
  as hook callers and must be rendered unconditionally too.
- `core.SetDebugMode(true)` detects the drift and reports it; see Debug mode.

**Updating state** — treat values as immutable and replace them:

```go
items := core.NewState(ctx, []Todo{})

// WRONG: mutating the slice in place. The slot still holds the same header,
// so a pointer-equality fast path can conclude nothing changed.
list := items.Get()
list[0].Done = true

// RIGHT: copy, change the copy, Set it.
list := append([]Todo(nil), items.Get()...)
list[0].Done = true
items.Set(list)
```

`Set` is safe from **any goroutine** — a timer, an HTTP response, a channel
reader. It writes the slot and nudges the render loop, which coalesces a burst
of writes into one pass over the settled state.

### State lives high

State that must outlive a screen belongs above the thing that comes and goes.
Routes are closures, so the usual move is to capture the context the
`Navigator` renders into:

```go
func App(ctx *core.Context) core.View {
    cart := core.NewState(ctx, []Item{})       // survives every navigation
    return core.Navigator(func(*core.Context) core.View {
        return productList(ctx, cart)          // capture, don't re-declare
    })
}
```

### Scoping

`ctx.Scope(key)` gives a stable named sub-context (cursor independent of its
parent's), and `core.UseChildContext(ctx)` gives a positional one. Reach for
`Scope` when a subtree's hook count varies and you want it not to disturb
siblings.

## Hooks

```go
import "github.com/rohanthewiz/grmob/hooks"

hooks.UseEffect(ctx, fn)                       // every pass
hooks.UseEffect(ctx, fn, userID)               // when userID changes
hooks.UseInterval(ctx, tick, time.Second)      // self-cancelling on close
hooks.UseTimeout(ctx, fire, 300*time.Millisecond)
v   := hooks.UseMemo(ctx, expensive, dep)
s, dispatch := hooks.UseReducer(ctx, reduce, initial)
d   := hooks.UseDebounce(ctx, 250*time.Millisecond)

loc  := hooks.UseLocation(ctx)                 // read-only host reports
life := hooks.UseLifecycle(ctx)
st   := hooks.UsePermission(ctx, permission.Camera)
```

**Dependency comparison is value equality.** A func value, or a slice built
fresh each pass, compares unequal every time — which turns a conditional effect
into an unconditional one silently, since it still does the right thing, just
far more often than intended.

`UseInterval` / `UseTimeout` re-bind their function every pass, so the callback
that fires sees **current** state, not the state of the pass that started it.

Cleanup: `ctx.OnClose(fn)`. Contexts close when their frame leaves the
navigation stack or when `Manager.Close` runs, which is what makes a ticker
inside a screen safe to walk away from.

## Styling and theming

Individual props, or a whole `Style` value:

```go
core.Box(
    core.Padding(16), core.BorderRadius(8),
    core.BackgroundColor("#fff"), core.Gap(8),
    core.Width("100%"), core.Justify(core.JustifyBetween),
)

core.Text("Body", core.UseStyle(ctx.Theme().Typography.Body))
```

**`UseStyle`'s one trap:** the merge rule is "a set field wins, an unset field
is ignored", and a zero value is indistinguishable from unset. So `UseStyle`
can **add** but never **clear**:

```go
core.UseStyle(core.Style{FontSize: 0})   // does NOT reset the font size
core.FontSize(0)                          // this does
```

Theme a subtree, and read the theme rather than hard-coding colors:

```go
core.WithTheme(core.MaterialTheme, screen())
ctx.Theme().Colors.Primary    // roles, not hex, in widget code
ctx.Theme().Typography.Body   // Title / Subtitle / Body / Caption
ctx.Theme().Spacing           // never a magic number in widget code
```

Bundled: `core.DefaultTheme`, `core.MaterialTheme`, `core.AmberTheme`,
`core.DarkTheme` (all four from `core.BundledThemes()`).
Override by copying a base and editing the copy.

### Accessibility

Accessibility props are ordinary `StyleProp`s, so they compose like any other:

```go
core.Button("Close", onClose, core.AccessibilityLabel("Close dialog"))
```

**Do not restate a role the node type already owns.** `core.Button` exports as
a `<button>` and builds a real control on both natives; `core.Modal` writes
`role="dialog"` itself. `RoleButton` exists for the *other* case — a `Box` or
`Row` with an `OnClick`, which every renderer otherwise draws as inert scenery:

```go
core.Box(core.OnClick(open), core.AccessibilityRole(core.RoleButton),
    core.AccessibilityLabel("Open settings"))
```

Landmark, live-region and content roles (`RoleBanner`, `RoleNavigation`,
`RoleStatus`, `RoleHeading`, `RoleLink`, `RoleImg`, `RoleTab`) describe the node
itself and are always the author's to state. Four role families make claims
about their *children* — the tabular set, both collection pairs, and
`RoleTabList` — so a container carrying one has to actually contain what it
says (a `core.List` with a "Load more" footer inside it is not a list).

## Events

There are three prop families, and they differ in what they need:

| Family | Interface | Example |
|---|---|---|
| `StyleProp` | `Apply(*Style)` | `core.Padding(8)`, `core.AccessibilityLabel("x")` |
| `BehaviorProp` | `Apply(*Context, *Node)` | `core.OnClick(fn)` — needs the context's callback registry |
| child views | `core.View` | `core.Text("hi")` |

```go
core.Box(
    core.OnClick(open),                    // tap
    core.OnLongPress(showMenu),
    core.OnTouch(highlight),
    core.OnFocus(onFocus), core.OnBlur(onBlur),
    core.On("custom-event", handler),      // the general form
    core.Text("Settings"),
)
```

Handlers are registered per pass and get an ID; IDs are purged and reissued
every pass, which is why **a callback inside a `Cached` subtree dangles** and
why nothing outside a render may hold one. The host sends the ID back, the
context dispatches, and the pass that follows carries the result.

Handler callbacks run under the render mutex, so a handler observes a settled
tree and every write it makes is rendered together by the next pass.

## Conditionals and lists

```go
core.If(loading, core.Text("Loading…"))
core.IfElse(signedIn, dashboard(), loginForm())

core.For(todos, func(t Todo, i int) core.View {
    return core.Keyed(t.ID, todoRow(ctx, t))
})

core.Match(status,
    core.Case(StatusOK, core.Text("ok")),
    core.Case(StatusErr, core.Text("failed")),
)
core.MaybeProp(compact, core.Padding(4))   // a conditional prop
```

**Key children of a dynamic list.** Positional diffing means an unkeyed
insertion at the top re-props every row below it. Keys also have a limit worth
knowing: there are no move patches yet, so a key that changes slot is
*replaced*, which is visually right but costs that subtree its transient native
state (focus, scroll offset).

## Navigation

```go
core.Navigator(func(ctx *core.Context) core.View { return home(ctx) })

core.Push(ctx, detail)       // screen underneath keeps its state
core.Pop(ctx)
core.Replace(ctx, other)
core.Reset(ctx, home)        // clear the stack
core.PopToRoot(ctx)
core.CanPop(ctx)             // e.g. whether to draw a back button
```

Each frame renders in its own scope, so routes may use hooks freely, and a
route's state is discarded when its frame leaves the stack.

**Two different `Reset`s — do not confuse them:**

| Call | Meaning |
|---|---|
| `core.Reset(ctx, route)` | navigation: clear the stack, seed with `route` |
| `ctx.Reset()` | rewind the context's hook cursor — **render-driver business.** Calling it from inside a tree corrupts sibling slots mid-pass |

## Forms

```go
import "github.com/rohanthewiz/grmob/forms"

form := forms.UseForm(ctx, forms.Spec{
    Fields: []forms.Field{
        {Name: "email", Rules: []forms.Rule{
            forms.Required("We need an address to reach you"),
            forms.Email(""),                      // "" takes the default message
        }},
        {Name: "password", Rules: []forms.Rule{forms.MinLen(8, "")}},
        {Name: "confirm"},
        {Name: "terms", Rules: []forms.Rule{forms.Accepted("Please accept")}},
    },
    // Cross-field checks see every value at once.
    Validate: func(v forms.Values) map[string]string {
        if v["confirm"] != v["password"] {
            return map[string]string{"confirm": "The two passwords differ"}
        }
        return nil
    },
})

comps.FormField{
    Label: "Email",
    Error: form.Error("email"),                   // "" until the user should see it
    Input: form.Input("email", "you@example.com"),
}
```

`form.Input(name, placeholder)` binds value **and** onChange to the same name in
one call. Also: `form.Checkbox`, `form.InputWithSubmit`, `form.Error(name)`,
`form.Errors()`, `form.Blurred(name)`.

Rules run in order and **the first complaint wins**, because a field shows one
line of feedback — so `Required` belongs first; it is the only rule that speaks
about an empty value. Field rules beat `Validate` messages for the same field,
being the more specific complaint. A `Validate` key that names no field is a
form-level error, readable with `form.Error("form")`. `Field.Initial` seeds a
value once and is not re-applied, so a field the user cleared stays cleared.
Rules and `Validate` must be pure and must not call back into the `Form`.
The full rule set: `Required` `MinLen` `MaxLen` `Email` `Pattern` `Accepted`.

The form — not the widget — decides *when* an error becomes visible. Do not
build that logic yourself.

## The widget library

```go
import "github.com/rohanthewiz/grmob/comps"
```

Struct widgets configured by named field, with `core.View` composition slots:

```go
comps.Screen{
    Children: []core.View{
        comps.AppBar{Title: "Account", Subtitle: "4 cards"},
        comps.Card{
            Title:  "Balance",
            Body:   core.Text("$42.00"),
            Footer: comps.Badge{Text: "verified"},
        },
        comps.SegmentedControl{Labels: tabs, Selected: sel, OnSelect: pick},
    },
}
```

Available, by the topic pages of `docs/api/comps-*.md` (read the page for a
widget's exact fields before using it):

| Topic | Widgets |
|---|---|
| Screens & structure | `Screen` `AppBar` `BottomBar` `FAB` `Tabs` `Drawer` `StepIndicator` `Wizard` `TreeView` `TwoPane` `Card` `Accordion` `Breadcrumb` `Separator` `LabeledSeparator` |
| Lists & tables | `ListRow` `SwitchRow` `CheckboxRow` `SelectRow` `SliderRow` `InputRow` `KeyValueList` `BulletList` `GroupedList[T]` `GroupHeader` `CollapseBand` `DataTable[T]` `EditableGrid` `Pagination` `LoadMore` `Timeline` |
| Inputs & pickers | `FormField` `PasswordField` `PINInput` `TagInput` `MaskedInput` `NumberPad` `ColorSwatchPicker` `RangeSlider` `SearchField` `SearchableSelect` `RadioGroup` `Calendar` `DatePicker` `DateRangePicker` `TimePicker` `CodeEditor` `RichTextEditor` `RichTextView` |
| Buttons & choices | `Button` `CopyButton` `Link` `Chip` `ChipStrip` `SegmentedControl` `Stepper` `Rating` `Badge` |
| Overlays & feedback | `Dialog` `Lightbox` `ActionSheet` `Menu` `Snackbar` `Banner` `ProgressBar` `Spinner` `Skeleton` `EmptyState` |
| Data display & maps | `Avatar` `AvatarStack` `StatTile` `Compass` `AnalogClock` `DigitalClock` `Countdown` `Stopwatch` `AlarmRow` `AlarmRinging` `AudioPlayer` `MessageBubble` `MessageThread` `TypingIndicator` `ReactionBar` `Poll` `ExpandableText` `QRCode` `MapPanel` `StaticMap` |
| Charts | `Sparkline` `LineChart` `AreaChart` `BarChart` `ScatterChart` `Histogram` `Heatmap` `CalendarHeatmap` `DonutChart` `PieChart` `Gauge` `CandlestickChart` `FunnelChart` `RadarChart` `Waveform` |

`Screen` is the scaffold: `Children`, `Scroll`, `KeyboardAware`, `Gap`, `Fill`,
`Style`. Leave `Scroll` false when the screen already contains its own
scrolling region — a scroll view inside a scroll view fights for the same drag
on both natives.

Widgets take their look from `ctx.Theme()` and accept `Style` overrides (a
`[]core.StyleProp` field). When a widget offers both a simple path and a slot
(`Card.Title` vs `Card.Header`), the **slot wins** if both are set.

## Caching

```go
core.Cached(expensiveStaticSubtree())
```

`Cached` renders once and replays the same `*Node` forever, which the diff
short-circuits on pointer identity. The constraints are strict, and debug mode
enforces them:

- Construct it **once** — `core.Cached(x)` inside a render call caches nothing.
- No hooks inside.
- No callbacks inside (their IDs are purged each pass and would dangle).
- Nothing that reads changing state.

Reach for it as a profiling response, not a default.

## Debug mode

```go
core.SetDebugMode(true)   // development builds and tests; costs nothing when off
```

It flags exactly the failures that are otherwise silent: cursor drift, duplicate
sibling keys, dangling callback references, handler panics, misused caches.

```go
core.ClearConcerns()
// ... drive some passes ...
if c := core.DumpConcerns(); c != "" {
    t.Fatalf("debug concerns raised:\n%s", c)
}
```

Asserting the concern list is **empty** is the standard test shape.

## Testing — the fastest loop there is

No simulator, no browser, no device. Drive the real engine from a Go test:

```go
func TestMain(m *testing.M) {
    core.SetDebugMode(true)   // audit every pass this package drives
    m.Run()
}

func TestIncrement(t *testing.T) {
    mgr := render.New(core.NewContext(), App)
    defer mgr.Close()

    tree := mgr.RenderInitial()             // the whole tree, as JSON
    id := findCallbackID(tree, "+")
    patches := mgr.DispatchCallback(id)     // tap it; get the diff back

    if !strings.Contains(patches, "Count: 1") {
        t.Fatalf("got %s", patches)
    }
    if c := core.DumpConcerns(); c != "" {
        t.Fatalf("concerns:\n%s", c)
    }
}
```

`render.Manager` is the app-lifetime owner: `New`, `SetListener`,
`RenderInitial`, `Dispatch(label, fn)`, `DispatchCallback` /
`DispatchTextCallback` / `DispatchBoolCallback` / `DispatchIntCallback`,
`RenderAgain`, `RenderAndGetPatches`, `Close`.

Every pass is serialised under one mutex, so a handler always observes a settled
tree and its writes are rendered together by the pass that follows.

For a snapshot you can pin byte for byte:

```go
html := htmlout.ExportHTML(node)   // standalone HTML document
json := jsonout.Export(node)       // the same shape the bridges send
```

## Targets

| Target | In an app made by `grmob new` | What runs |
|---|---|---|
| Go test | `go test ./app` | `render.New` — start here for everything |
| Browser | `./dev.sh` (hot swap) or `grmob web` / `./build.sh` | `webhost.Run` + `grmob-runtime.js` |
| Android | `grmob android [-install] [-refresh]` | gomobile bind → `.aar`; Compose renderer applies patches |
| iOS | `grmob ios [-run] [-open] [-refresh]` | gomobile bind → xcframework; SwiftUI renderer applies patches |
| HTML | `htmlout.ExportHTML(node)` | a standalone document, for snapshots |

Inside this repository instead, `./build.sh` + `go run ./serve` (or
`go run ./serve -dev`) runs the interactive tutorial, and `android/build.sh
./<pkg>` / `ios/build.sh ./<pkg>` bind an example package into the shells.

The native bridge (`package mobile`) narrows everything to strings, bools and a
one-method interface, because gomobile cannot bind functions, generics or maps:

```go
mobile.Register(core.NewContext(), App)
mobile.SetListener(l)          // Go → native push channel for async updates
mobile.RenderInitial()         // string
mobile.TriggerCallback(id)     // string of patches, synchronously
mobile.SetDataDir(path)        // call before anything opens a store
```

`SetDataDir` timing matters: `init` runs before the host can call it, so open
persistent stores lazily, not in `init`.

### Persistence

Use `github.com/rohanthewiz/bytdb`, opened lazily on the first render pass.
`examples/todoapp/store.go` is the reference shape:

```
first render ──▶ openStore() ──▶ read snapshot ──▶ NewState initial values
tap / submit ──▶ mutation helper ──▶ State.Set (UI)
                               └───▶ store write-through (disk)
```

- `openStore` reads `mobile.DataDir()`; empty (browser preview, bare tests)
  means **run in memory** and return nil. Make every store method
  nil-receiver-safe so the app calls them unconditionally.
- Keep the open engine in a mutex-guarded package singleton: bytdb holds an
  exclusive file lock, so a second `Open` of the same path fails. Reopen when
  the directory changes (tests move to a fresh `t.TempDir()`).
- Seed `NewState` from the snapshot rather than loading in `hooks.UseEffect`:
  effects run on their own goroutine, so the first frame would mount empty and
  the rows would pop in a patch later.
- Write through synchronously in the mutation helpers — the single choke
  point every change goes through — instead of a dirty flag or debounce.
- A failed open logs and degrades to in-memory; losing persistence beats
  losing the UI.
- Tests: call `mobile.SetDataDir(t.TempDir())` before `render.New`.

Patches arrive on two paths — the synchronous return of a `Trigger*` call, and
`PatchListener` pushes for timers and goroutines. Each pass delivers its diff on
exactly one of them, so a renderer applies everything it receives in arrival
order and stays consistent. `ApplyPatches` is called from a background
goroutine: hop to the UI thread before touching views.

## Pitfalls, in the order they bite

1. **A hook inside a conditional.** Shifts every later slot. Turn on debug mode.
2. **Mutating state in place** instead of replacing the value.
3. **Mutating a `*Node` after render.** The diff will skip what you changed.
4. **Unkeyed dynamic lists.** Key rows by a stable ID.
5. **`core.Reset` vs `ctx.Reset`.** The second is not yours to call.
6. **`UseStyle` to clear a field.** It cannot; use the individual setter.
7. **Effect deps that are funcs or fresh slices.** They never compare equal.
8. **`Cached` constructed inside a render**, or containing a callback or a hook.
9. **No bindable exported symbol** in an app package, so the linker drops it.
10. **Opening a data store in `init`**, before the host has called `SetDataDir`.
11. **Double insets on nested containers.** The theme pads every `Column` and
    `Row`; a nested one needs `core.Padding(0)` to line up with its parent's content.

## Where the full documentation is

Narrative guides, and a generated reference of every exported symbol:

```sh
./docs.sh     # serves mkdocs.yml + docs/ with gkdocs at :8000
```

- `docs/getting-started.md`, `docs/tutorial-todo.md`,
  `docs/tutorial-interactive.md`
- `docs/concepts/` — architecture, views, state & hooks, events, forms,
  navigation, styling, reconciliation, caching, error boundaries, debug mode
- `docs/api/` — one page per package, generated from the doc comments by
  `go run ./internal/apidoc/gen` and committed
