# GrMob Roadmap

> A native UI runtime for Go — declarative, composable, and portable.

---

## ✅ Done

### 🎯 Core Engine
- [x] Declarative component system in idiomatic Go
- [x] `View` interface & `ComponentFunc` for composable UIs
- [x] Reconciler with diffing and patching system
- [x] `NewState` for local stateful logic
- [x] `If`, `Match`, `For` for conditional and iterative rendering
- [x] Style system with reusable `StyleProps`
- [x] `Box`, `Text`, `Image`, `Button`, `Input`, `Spacer`, etc.
- [x] Hooks: `UseInterval`, `UseTimeout`, `UseEffect`, `UseMemo`,
      `UseReducer`, `UseChildContext`
- [x] `ErrorBoundary` / `SafeRender`, plus the driver's pass and event-handler
      panic guards — a panicking component costs its subtree, not the process
- [x] WASM runtime (`grmob-runtime.js`) with event bridge

### 🧪 Layout & Styling
- [x] `Row`, `Column`, `Gap`, `Align`, `Justify`, `ZIndex`
- [x] `PositionSticky`, `Absolute`, `Relative`, etc.
- [x] Responsive layouts via style merging
- [x] Shadow, border, radius, hover styles
- [x] Proportional flex weights on every target — `GrMobFlexStack`, a custom
      SwiftUI `Layout`, brought iOS in line with Compose's `Modifier.weight`
      (and made `justify-content` exact rather than Spacer-emulated)
- [x] `AlignItems: "stretch"` on both native renderers
- [x] A `ContentMode` prop on `Image` (`Fit` / `Fill` / `Stretch` / `Center`)
- [x] A native disabled state — `core.Disabled` maps onto the platform's own,
      subtree-propagating, and announced by the screen reader

### 🧠 Developer Experience
- [x] Internal path-based rendering IDs for patches (`reconcile.Patch.TargetID`)
- [x] Logging and inspection of patches
- [x] Debug mode: cursor-drift, duplicate-key, unknown-item, render-panic
      and handler-panic concerns
- [x] `htmlout` HTML export for tests and tooling
- [x] Snapshot testing for views — the migration tests pin a widget's
      exported markup to the hand-rolled original byte for byte

### 🌐 Web Support
- [x] WASM compilation with `main.wasm`
- [x] HTML + JS runtime to mount and patch views
- [x] WebAssembly event bridge to Go via `window.GoInvokeCallback`

### 🎨 Theming
- [x] `Theme{}` with `ColorPalette`, `Typography`, `SpacingScale`, component defaults
- [x] Two bundled themes (`DefaultTheme`, `MaterialTheme`) — an Apple-like
      and a Google-like design system, which was the "real-world design
      system demo" this file used to track
- [x] Semantic status roles: `Success`, `Warning`, `Error`, `Border`

### 📱 Native Runtime Bridges
- [x] **Android Runtime** (Go → JSON → Jetpack Compose renderer)
- [x] **iOS Runtime** (Go → JSON → SwiftUI renderer)
- [x] gomobile bridge with a four-channel event surface (`mobile/bridge.go`)

### 🧩 Widget Library (`components`)
- [x] `Screen`, `Button`, `InputRow`, `SegmentedControl`, `Card`, `ListRow`
- [x] `Badge`, `Chip`, `Separator`, `Avatar`, `ProgressBar`
- [x] `FormField`, `Accordion`, `Tabs`
- [x] `Variant` × `Emphasis` color axes with contrast-picked ink

### 🧬 Extensions
- [x] Animations & transitions (`Transition`, easing curves)
- [x] Accessibility labels, hints and announced selection state
- [x] Navigation (`Navigator`, `Push`, `Pop`, `Replace`, `PopToRoot`, `Reset`,
      per-frame state) and `core.Modal` / toasts
- [x] Forms with validation (`forms`) — a rule vocabulary, cross-field checks,
      four reveal policies so a form explains itself only once the user claims
      to be done, and server-side errors; `FormField`'s `Error` slot finally
      has a source (`examples/signup`)
- [x] Focus and blur events (`core.OnFocus`, `core.OnBlur`) — the input
      builders now take behavior props like the containers do, and
      `forms.RevealOnBlur` reveals a field's error when the user leaves it
      rather than on their second keystroke (`examples/signup`)
- [x] Keyboard-aware regions (`core.KeyboardAware`,
      `components.Screen.KeyboardAware`) — a scrolling region shortens its
      viewport, a fixed one lifts whole, so a docked composer stays reachable
      (`examples/signup`, `examples/chat`)
- [x] Camera: `CameraView`, capture event
- [x] Persistence via `bytdb` (see `examples/todoapp`)

---

## 🧩 In Progress

### 📦 Packaging
- [ ] `grmob build --target=wasm`
- [ ] `grmob build --target=android`
- [ ] `grmob build --target=ios`

---

## 🔜 Planned


### Native Bridge (Planned for Android/iOS)

- [ ] Keystore (Secure): `Keystore.Save()`, `Keystore.Get()`
- [ ] Device Storage (Plain): `DeviceStorage.Set()`, `DeviceStorage.Get()`
- [ ] Bluetooth: `Scan`, `Connect`, `Send`
- [ ] Location / GPS
- [ ] FaceID / Biometric authentication
- [ ] Contacts access

### 📱 Native Runtime Bridges

- [ ] **Still thinking** for desktop

### 🛠️ DevTools
- [ ] State inspector overlay (similar to React DevTools)
- [ ] Visual patch viewer
- [ ] Hot module replacement

### 🧪 Testing & Perf
- [ ] Benchmark diff/patch engine
- [ ] Latency profiling in runtime patching

### 🧬 Extensions
- [ ] Router-style navigation for web
- [ ] Keyboard navigation (the accessibility *labels* are done, and focus is
      now *observable* via `OnFocus`/`OnBlur`; programmatic focus — moving it,
      or dismissing the keyboard on a tap outside — is not)

---

## 🌍 Vision

> Build **native experiences** using only Go.

---

## 💬 Contribute

Open to collaborators and contributors.
Join the discussion or check the GitHub repo.

---

