# GrMob

**GrMob** is a fully idiomatic Go framework for building native mobile apps using a declarative, functional DSL. Designed entirely in Go — GrMob offers a new approach to mobile development where UI, logic, and state management are written in pure Go, and rendered natively on Android and iOS.

---

## ✨ Features

- **Declarative Syntax** – Compose views with pure functions and fluent props
- **Native Rendering** – Output native components on Android/iOS
- **Component-Based** – Build custom views by composing smaller ones
- **Styling System** – Functional styling with support for themes and inheritance
- **State Management** – Built-in state system inspired by hooks (`NewState`, `UseInterval`, `UseTimeout`, etc.)
- **Event Handling** – Built-in callback registry for interactions
- **Theming & Tokens** – Define centralized visual identity and reusable design primitives
- **Bridge-Free Events** – Events and hardware calls require no manual bridge setup
- **App Config Injection** – Provide global config for name, author, version, locale
- **Reactive Runtime** – Smart diffing engine with `patch` and `mount`, dirty flag detection
- **Timers & Effects** – `UseInterval`, `UseTimeout`, `UseEffect`, `UseMemo`, `UseReducer`
- **Forms & Navigation** – Validation rules with reveal policies, and a `Navigator` with modals and toasts
- **Widget Library** – `components`: buttons, cards, chips, tabs, accordions, form fields and more
- **Robustness** – Error boundaries, a zero-cost debug mode, and cached subtrees
- **WebAssembly Support** – The same app runs in the browser via Go + WASM

---

## 📦 Example

A **simple social network profile screen**, broken into components:

### `register.go`
```go
package social

import (
    "github.com/rohanthewiz/grmob/core"
    "github.com/rohanthewiz/grmob/mobile"
)

// Registering the root view is the whole integration contract: the native
// shells and the WASM host call the mobile bridge, and it drives App.
func init() {
    ctx := core.NewContext().WithConfig(&core.AppConfig{
        Name:    "LetsBe Social",
        Author:  "Your Name",
        Version: "0.1.0",
        Locale:  "en-MZ",
    })
    mobile.Register(ctx, App)
}

// AppName gives gomobile a bindable symbol so this package links in.
func AppName() string { return "LetsBe Social" }
```

### `app.go`
```go
import "github.com/rohanthewiz/grmob/core"

func App(ctx *core.Context) core.View {
    return core.SafeArea(
        core.Scroll(
            core.Column(
                ProfileHeader(),
                core.Spacer(16),
                ProfileStats(),
                core.Spacer(12),
                PostList(),
            ),
        ),
    )
}
```

### `profile.go`
```go
import "github.com/rohanthewiz/grmob/core"

func ProfileHeader() core.View {
    return core.Column(
        core.Image("https://example.com/avatar.jpg", core.UseStyle(core.Style{BorderRadius: 40})),
        core.Text("Jane Doe", core.FontSize(20), core.FontWeight(core.Bold)),
        core.Text("Software Engineer • Maputo"),
    )
}

func ProfileStats() core.View {
    return core.Row(
        Stat("Posts", "128"),
        core.Spacer(12),
        Stat("Followers", "1.2k"),
        core.Spacer(12),
        Stat("Following", "180"),
    )
}

func Stat(label, value string) core.View {
    return core.Column(
        core.Text(value, core.FontWeight(core.Bold)),
        core.Text(label, core.TextColor("#888")),
    )
}
```

### `posts.go`
```go
import "github.com/rohanthewiz/grmob/core"

func PostList() core.View {
    return core.Column(
        Post("Enjoying the GrMob project! 🚀"),
        core.Spacer(8),
        Post("Working on UI DSLs in Go is pure joy.", "#golang #ux #native"),
    )
}

func Post(content string, tags ...string) core.View {
    full := content
    if len(tags) > 0 {
        full += "\n" + tags[0]
    }
    return core.Card(
        core.Column(
            core.Text(full),
            core.Spacer(4),
            core.Row(
                core.Button("Like", func() { /* like it */ }),
                core.Spacer(4),
                core.Button("Comment", func() { /* open composer */ }),
            ),
        ),
    )
}
```
## 🧠 Conditional Components

GrMob offers expressive helpers like `If`, `IfElse`, `Match`, and `When` to enable clear and composable **conditional rendering**, plus `MaybeProp` for a single optional child, style prop or handler inside a container's argument list.

This eliminates verbose control flow scattered across functions and allows you to describe UI variations naturally and declaratively.

### ✅ Benefits
- Write cleaner, more declarative code
- Avoid nested `if` statements in render logic
- Make the UI adapt reactively to state changes
- Encapsulate complex flows (like onboarding, permissions, login states)

### ✨ Examples

#### Simple `If`
```go
core.If(user.Get() != "", core.Text("Welcome, "+user.Get()))
```

#### With fallback
```go
core.IfElse(isLoading.Get(),
    core.Text("Loading..."),
    core.Text("Ready"),
)
```

#### Match enum
```go
core.Match(status.Get(),
    core.Case("success", core.Text("✅ Success")),
    core.Case("error", core.Text("❌ Error")),
    core.Default[string](core.Text("ℹ️ Idle")),
)
```

#### Multiple conditions with `When`
```go
core.MatchBool(
    core.When(user.Get() == "", core.Text("👋 Welcome Guest")),
    core.When(user.Get() == "admin", core.Text("🛠️ Admin Panel")),
    core.Otherwise(core.Text("Logged in as "+user.Get())),
)
```

This leads to beautiful, logical component trees that **read like prose**.

## 🎯 Event Handlers

You can attach callbacks to any element using the generic `On` helper or the
`ButtonWithEvent` constructor for buttons.

```go
core.Column(
    core.Text("Tap the box"),
    core.On("Click", func() { fmt.Println("Column clicked") }),
)

core.ButtonWithEvent("Hold", "TouchStart", func() {
    fmt.Println("Button touched")
})
```

---

## 📖 Tutorials

New to GrMob? Start with the [interactive tutorial](docs/tutorial-interactive.md)
— a GrMob app that teaches GrMob. Forty lessons across eight chapters, and
every lesson is a live screen: the explanation, the code under discussion,
and a "TRY IT" panel wired to real state and callbacks, from your first
`Column` through theming, navigation, and error boundaries. It lives in
[`examples/tutorial`](examples/tutorial) and runs in the browser with two
commands:

```bash
GOOS=js GOARCH=wasm go build -o wasm/main.wasm ./wasm
(cd wasm && python3 -m http.server 8080)
```

Then [Building a Todo App](docs/tutorial-todo.md) is an in-depth,
start-to-finish walkthrough: state and the rules of hooks, controlled inputs
with Enter-to-submit, the virtualized keyed `List`, theming pitfalls,
accessibility, testing at three levels, and shipping the same Go code to the
iOS simulator and Android emulator. The finished app is
[`examples/todoapp`](examples/todoapp).

---

## 📐 Architecture

- `core/` – Node, View, Context, State, Style, theming, navigation, focus, error boundaries, debug mode
- `hooks/` – `UseInterval`, `UseTimeout`, `UseEffect`, `UseMemo`, `UseReducer`
- `components/` – the widget library built on `core`
- `forms/` – form state and validation: rules, cross-field checks, reveal policies, server errors
- `reconcile/` – the diff engine that turns two trees into a patch list
- `render/` – the render manager: passes, dirty tracking, callback dispatch, patch pumping
- `mobile/` – the gomobile-bindable bridge the native shells talk to
- `android/` – the Jetpack Compose shell and renderer (Kotlin)
- `ios/` – the SwiftUI shell and renderer (Swift)
- `wasm/` – the WebAssembly host and JS runtime for the browser
- `htmlout/`, `jsonout/` – exporters for previews, snapshot tests and tooling
- `examples/` – complete apps, including the interactive tutorial and the todo app

---

## 📱 Renderers

Renderers turn the abstract `Node` tree into real UI, and apply the reconciler's
patches to keep it current.

- **Android** – Jetpack Compose (`android/`)
- **iOS** – SwiftUI (`ios/`)
- **Browser** – the DOM, via WebAssembly and `wasm/grmob-runtime.js`
- **HTML export** – `htmlout`, for previews and byte-for-byte snapshot tests

Every renderer has a verify harness that replays the Go engine's own patch
transcripts against it, so the three targets stay in step.

## 🏗 Building for Android and iOS

Both shells take an app package as their argument. Any package whose `init`
calls `mobile.Register` drops in; the examples all do.

```bash
go install golang.org/x/mobile/cmd/gomobile@latest golang.org/x/mobile/cmd/gobind@latest
gomobile init

android/build.sh ./examples/todoapp   # binds mobile + your app into android/app/libs/grmob.aar
ios/build.sh ./examples/todoapp       # needs full Xcode; produces the xcframework for ios/
```

Then open `android/` in Android Studio, or the Xcode project under `ios/`, and
run. The full walkthrough, including the bridge contract the shells implement,
is in [docs/platforms/native.md](docs/platforms/native.md).

---

## 🛠 Dev Experience

- Drive the whole engine from a plain Go test: `render.New` → `RenderInitial` → `DispatchCallback`
- Snapshot views as HTML with `htmlout` and pin them byte for byte
- Debug mode flags cursor drift, duplicate keys, unknown items and panics, and costs nothing when off
- Inspect any render's patch set directly: `reconcile.Diff` returns plain
  `[]Patch` values, and `RenderAgain` / `TriggerCallback` hand the same set
  back as JSON
- The browser host polls `IsDirty` and applies minimal DOM patches, so nothing re-mounts

---

## 🧠 Hooks (State & Side Effects)

- `NewState[T]` – reactive state local to a component
- `UseInterval(ctx, fn, interval)` – run `fn` on an interval
- `UseTimeout(ctx, fn, delay)` – run `fn` once after a delay
- `UseEffect(ctx, fn, deps...)` – run after mount, and again when a dependency changes
- `UseMemo(ctx, compute, deps...)` – cache a value until a dependency changes
- `UseReducer(ctx, reducer, initial)` – state driven by dispatched actions

---

## 📃 License

MIT License © 2026 Rohan Allison · © 2025 Ismael Matsinhe
