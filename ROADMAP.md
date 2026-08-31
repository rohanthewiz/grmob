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
- [x] Navigation (`Navigator`, `Push`, `Pop`) and `core.Modal` / toasts
- [x] Camera: `CameraView`, capture event
- [x] Persistence via `bytdb` (see `examples/todoapp`)

---

## 🧩 In Progress

### 🔧 Core Abstractions
- [ ] `Reset` for the navigation stack (`Push` / `Pop` are done)

### 🧰 UI DSL
- [ ] Forms with validation — `FormField` renders an `Error`, but nothing
      validates
- [ ] Keyboard-aware scroll area for mobile
- [ ] Proportional flex weights on iOS (needs a custom SwiftUI `Layout`)
- [ ] `AlignItems: "stretch"` on both native renderers
- [ ] A `ContentMode` prop on `Image`
- [ ] A native disabled state — no renderer carries one

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
- [ ] Keyboard navigation (the accessibility *labels* are done; focus
      traversal is not)

---

## 🌍 Vision

> Build **native experiences** using only Go.

---

## 💬 Contribute

Open to collaborators and contributors.
Join the discussion or check the GitHub repo.

---

