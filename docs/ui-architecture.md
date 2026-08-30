# **Styling Architecture and Theme System for GrMob**

**Ismael GraHms**  
*April 2025*

---

## **Abstract**

This paper proposes a comprehensive styling and theming architecture for a declarative UI Domain-Specific Language (DSL) written in the Go programming language. Rooted in the principles of functional UI composition, the architecture draws inspiration from frameworks such as React Native, SwiftUI, and Flutter, yet remains fully idiomatic to Go. The styling system supports fine-grained component styling, reuse, dynamic composition, and thematic consistency. Additionally, it introduces a centralized theming context to facilitate scalable design systems and adaptive visual identity.

---

## **1. Introduction**

In declarative user interface development, styling is not merely a cosmetic concern; it is a central part of compositional clarity, maintainability, and aesthetic coherence. Unlike imperative UI models where layout and style are entangled in mutable views, declarative UIs treat styling as pure configuration — separate from logic but deeply influential on perception.

The architecture herein presented offers:

- **Composability** — Styles and themes are reusable and easily merged.
- **Go-idiomatic syntax** — Style declarations follow the native constructs of Go: functions, structs, and interfaces.
- **Component hierarchy awareness** — Styles inherit and override within view trees.
- **Platform-agnostic abstractions** — Styling is neutral, enabling backend-specific renderers (e.g., SwiftUI, Compose).

---

## **2. Style Representation**

### 2.1 The `Style` Struct

The foundation of the styling system is a plain Go struct capturing common UI attributes.

```go
type Style struct {
    FontSize      int
    FontWeight    FontWeight
    TextColor     string
    Background    string
    Padding       EdgeInsets
    Margin        EdgeInsets
    BorderRadius  int
    Shadow        int
    Align         Alignment
    Display       DisplayMode
}
```

Each field is optional; unset fields take on defaults or inherit from parent styles.

### 2.2 Supporting Primitives

```go
type EdgeInsets struct {
    Top, Right, Bottom, Left int
    Horizontal, Vertical     int
}

type FontWeight int

const (
    Light  FontWeight = 200
    Normal FontWeight = 400
    Bold   FontWeight = 700
)
```

Partial edge insets are resolved using helper logic — for example, `Horizontal` overrides both `Left` and `Right`.

---

## **3. Declarative Styling in Go**

### 3.1 Inline Style Functions

The most idiomatic way to apply styling is via inline declarative functions returning `StyleProp`:

```go
Text("Welcome",
    FontSize(18),
    TextColor("#333"),
    Padding(12),
)
```

Each function implements:

```go
type StyleProp interface {
    Apply(*Style)
}

type styleFunc func(*Style)

func (f styleFunc) Apply(s *Style) { f(s) }
```

This model allows DSL components to accumulate and apply all style modifiers functionally.

---

## **4. Reuse and Composition**

### 4.1 Reusable Styles

Developers can define base styles as structs and apply them declaratively:

```go
var HeaderStyle = Style{
    FontSize:   22,
    FontWeight: Bold,
    TextColor:  "#111",
}
```

Usage:

```go
Text("Dashboard", UseStyle(HeaderStyle))
```

### 4.2 Merging Styles

The `With(...)` method allows the creation of composed styles:

```go
PrimaryButtonStyle := Style{
    Background: "#0099FF",
    TextColor:  "#FFFFFF",
}

Rounded := Style{ BorderRadius: 8 }

UseStyle(PrimaryButtonStyle.With(Rounded))
```

The merge is shallow and resolves only non-zero values, preserving base defaults unless explicitly overridden.

---

## **5. Hierarchical Style Propagation**

Within component trees, container components may define base styling that applies to children. This permits implicit design cohesion:

```go
Card(
    UseStyle(CardBaseStyle),
    Text("Section Header"),
    Button("Continue"),
)
```

Here, padding or background from `CardBaseStyle` can affect layout or visual context for its descendants.

A `StyleContext` is propagated top-down and accessed during component rendering.

---

## **6. Component Integration**

Each component accepts arbitrary styling via props:

```go
func PrimaryButton(label string, onClick func()) View {
    return Button(label,
        UseStyle(PrimaryButtonStyle),
        OnClick(onClick),
    )
}
```

The engine resolves styles from (in priority order):
- Inline style props
- Component defaults
- Thematic defaults
- Inherited context

---

## **7. Planned Extensions**

Future features include:
- **Responsive styling**, e.g., `When(ScreenWidth().LessThan(400), SmallStyle)`
- **Style tokens**, e.g., `Spacing.LG` or `Typography.Caption`
- **Conditional application**, e.g., `If(isError, TextColor("red"))`
- **Animation hooks**, e.g., `FadeIn(duration)`

---

## **8. Theme Support**

### 8.1 Purpose of Themes

Themes allow centralized definition of design systems. They:
- Ensure brand consistency
- Enable dark/light mode switching
- Encapsulate spacing, color, and typography standards
- Simplify refactoring

---

### 8.2 Theme Structure

```go
type Theme struct {
    Colors     ColorPalette
    Typography Typography
    Spacing    SpacingScale
    Components ComponentDefaults
}

type ColorPalette struct {
    Primary        string
    Secondary      string
    Background     string
    Surface        string
    TextPrimary    string
    TextSecondary  string
    Error          string
}

type Typography struct {
    Title    Style
    Subtitle Style
    Body     Style
    Caption  Style
}

type SpacingScale struct {
    XS, SM, MD, LG, XL int
}

type ComponentDefaults struct {
    Button Style
    Card   Style
    Input  Style
}
```

Themes can be extended by composition or partial overriding.

---

### 8.3 Applying a Theme

A theme is injected at the top level via context:

```go
App(
    WithTheme(DefaultLightTheme,
        Column(
            Text("Welcome"),  // Uses Typography.Body by default
            Button("Click"),  // Uses ComponentDefaults.Button
        ),
    ),
)
```

The context can be queried within components:

```go
func WelcomeHeader(name string, ctx Context) View {
    return Text("Hello, " + name,
        UseStyle(ctx.Theme().Typography.Title),
    )
}
```

---

### 8.4 Theme Switching

At runtime, the theme may be switched dynamically:

```go
ctx.SetTheme(DarkTheme)
```

The render engine re-evaluates the component tree accordingly, enabling dark mode toggles or user-defined themes.

---

### 8.5 Nesting and Partial Overrides

Themes can be scoped to a sub-tree and partially overridden:

```go
WithTheme(DefaultTheme.With(ColorPalette{
    Primary: "#FF0000",
}), ... )
```

Unspecified fields fall back to parent or default values.

---

### 8.6 Serialization and Tooling

Themes can be serialized to and from JSON, TOML, or YAML:

```json
{
  "colors": {
    "primary": "#0044CC",
    "background": "#FAFAFA"
  },
  "typography": {
    "title": {
      "fontSize": 24,
      "fontWeight": 700
    }
  }
}
```

This supports remote configuration, admin interfaces, or design-token synchronization.

---

## **9. Lifecycle of Styling**

The application of styles follows this sequence:

1. **Tree Construction:** Functional components yield a `View` tree.
2. **Style Resolution:** Each node composes style props from inline definitions, component defaults, and the theme context.
3. **Theme Application:** Theme tokens are injected if referenced or inherited.
4. **Rendering:** Final style structs are passed to platform-specific renderers.

---

## **10. Conclusion**

The styling and theme system presented herein constitutes the visual grammar of the UI DSL — a language for expressing aesthetic, identity, and layout with elegance and precision. It is declarative, idiomatic, and modular, respecting Go's syntactic integrity while enabling powerful abstractions for mobile-native interface development.

As the DSL evolves into a full-featured framework, this styling layer shall remain its spine: neutral, extensible, and deeply expressive.
