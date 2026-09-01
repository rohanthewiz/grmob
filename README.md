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
- **Timers & Effects** – Hooks like `UseInterval`, `UseTimeout`, and soon `UseEffect`
- **WebAssembly Support** – Works in browser environments via Go + WASM

---

## 📦 Example

A **simple social network profile screen**, broken into components:

### `main.go`
```go
package main

import (
    "fmt"
    "github.com/rohanthewiz/grmob/core"
)

func main() {
    ctx := core.NewContext().WithConfig(&core.AppConfig{
        Name: "LetsBe Social",
        Author: "Your Name",
        Version: "0.1.0",
        Locale: "en-MZ"
    })

    App(ctx).Render(ctx)
    
}
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
                core.Button("Like"),
                core.Spacer(4),
                core.Button("Comment"),
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
    core.Default(core.Text("ℹ️ Idle")),
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

- `core/` – core abstractions: Node, View, Context, State, Style
- `hooks/` – reactive utilities like `UseInterval`, `UseTimeout`, `UseEffect` (coming soon)
- `forms/` – form state and validation: rules, cross-field checks, reveal policy
- `render/` – render manager, patching logic, and JSON tree generation
- `android/` – native renderer for Android (Kotlin)
- `ios/` – native renderer for iOS (Swift or Kotlin Multiplatform)
- `examples/` – declarative UI demos in Go
- `wasm/` – WebAssembly runtime and JS bridge for testing in browser

---

## 📱 Renderers

Renderers are responsible for turning the abstract `Node` tree into real UI elements.

- Native Android using `FrameLayout`, `TextView`, `Button`, etc.
- Native iOS via `UIView`, `UILabel`, etc. (coming soon)
- HTML (optional, for export and dev tools)

## 🏗 Building the Android Module

1. Install `gomobile` and initialize it:
   ```bash
   go install golang.org/x/mobile/cmd/gomobile@latest
   gomobile init
   ```
2. Generate the Android library from the Go code:
   ```bash
   gomobile bind -target=android -tags grmob -o android/app/libs/grmob.aar ./
   ```
3. Open the `android/` folder in Android Studio and run the `app` module on a device or emulator.

---

## 🛠 Dev Experience

- Hot reload (planned)
- Custom DSLs and style tokens
- Testing helpers for views and events
- Code generation for component scaffolds (planned)
- Smart diff-based rendering with `IsDirty()` loop in JS runtime
- Patch minimization to avoid unnecessary DOM updates

---

## 🧠 Hooks (State & Side Effects)

- `NewState[T]` – basic reactive state
- `UseInterval(ctx, fn, interval)` – run `fn` on an interval
- `UseTimeout(ctx, fn, delay)` – run `fn` once after a delay
- `UseEffect(ctx, fn)` – run once after mount (coming soon)

---

## 📃 License

MIT License © 2026 Rohan Allison · © 2025 Ismael Matsinhe
