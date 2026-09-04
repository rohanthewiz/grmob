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
- [x] `Row`, `Column`, `Gap`, `RowGap`/`ColumnGap`, `FlexWrap`, `Align`,
      `Justify` — all four targets. The two gap longhands are the axis halves
      of `Gap` and win over it where set, exactly as in CSS.
- [x] `Position` (`Sticky`/`Absolute`/`Relative`/`Fixed`) with `Top`/`Right`/
      `Bottom`/`Left`/`ZIndex`, plus `MinWidth`/`MaxWidth`/`MinHeight`/
      `MaxHeight`, `Overflow`, `WhiteSpace`, `AlignSelf`,
      `FlexBasis`, `FlexShrink` — **web targets only**
      (WASM DOM and `htmlout`). Compose and SwiftUI have no direct equivalent
      for out-of-flow placement; a layout that depends on these will not look
      the same on device.
- [x] `Padding`/`Margin` `Horizontal`/`Vertical` shorthands on all four targets
- [x] Responsive layouts via style merging
- [x] `Shadow`, border, radius on all four targets
- [x] Proportional flex weights on every target — `GrMobFlexStack`, a custom
      SwiftUI `Layout`, brought iOS in line with Compose's `Modifier.weight`
      (and made `justify-content` exact rather than Spacer-emulated)
- [x] `AlignItems: "stretch"` on both native renderers
- [x] A `ContentMode` prop on `Image` (`Fit` / `Fill` / `Stretch` / `Center`)
- [x] A native disabled state — `core.Disabled` maps onto the platform's own,
      subtree-propagating, and announced by the screen reader

### 🧠 Developer Experience
- [x] Internal path-based rendering IDs for patches (`reconcile.Patch.TargetID`)
- [x] Inspectable patch sets — `reconcile.Diff` returns plain `[]Patch`
      values, and `RenderAgain`/`TriggerCallback` hand the same set back as
      JSON for a test or a host to read
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
- [x] gomobile bridge with a four-channel event surface (`mobile/bridge.go`),
      plus system events out and host events in

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
- [x] Programmatic focus (`core.Focus`, `core.DismissKeyboard`,
      `core.UseFocusRef`, `core.FocusTarget`) — a named field can be focused
      from anywhere and the keyboard dismissed on a tap outside; the commands
      ride the render tree as an epoch-stamped prop pair rather than a new
      bridge call. `core.Button` joined the same argument list in the process,
      so every leaf now takes behavior props (`examples/signup`)
- [x] Focus traversal (`core.UseFocusOrder`, `core.FocusNext`,
      `core.FocusPrevious`) — a form declares the order its return key walks
      in one line, and every field but the last advertises the platform's
      "next" action (`ImeAction.Next`, `.submitLabel(.next)`,
      `enterkeyhint`). The action rides the existing `onSubmit` channel, so it
      costs one string prop and no new bridge surface (`examples/signup`)
- [x] Keyboard-aware regions (`core.KeyboardAware`,
      `components.Screen.KeyboardAware`) — a scrolling region shortens its
      viewport, a fixed one lifts whole, so a docked composer stays reachable
      (`examples/signup`, `examples/chat`)
- [x] Camera: `CameraView`, capture event
- [x] Persistence via `bytdb` (see `examples/todoapp`)
- [x] System events on every host — `mobile.SetSystemEventListener` plus the
      Kotlin and Swift sinks behind it. Toasts previously reached only the
      browser: the natives had no sink at all, so `core.ShowToast` on a device
      emitted into a nil handler and vanished
- [x] `core.OpenURL` — hand a URL to the platform's own browser, dialer or mail
      composer (Intent ACTION_VIEW / UIApplication.open / window.open)
- [x] Audio playback with a media session — `core.AudioLoad/Play/Pause/Seek/
      Skip/SetRate/Stop`, `hooks.UseAudio`, one player per process behind
      Media3 + `MediaSessionService` (Android), `AVPlayer` + Now Playing +
      remote commands (iOS), `HTMLAudioElement` + the Media Session API
      (browser); background playback and lock-screen controls on both natives
- [x] Host events — the reverse of system events: `core.ReceiveHostEvent` /
      `core.OnHostEvent`, `mobile.ReportHostEvent`, `GrMobWASM.HostEvent`.
      Audio status was the first traffic and the app lifecycle the second;
      keystore results and location fixes have their channel ready
- [x] App lifecycle events — `core.CurrentLifecycle` / `core.OnLifecycle` /
      `hooks.UseLifecycle` over the `"lifecycle"` host event; active /
      inactive / background from `ProcessLifecycleOwner`, `scenePhase` and
      the Page Visibility API, so a client can reconnect on resume
- [x] `core.Slider` — a range control on all four targets, with a separate
      end-of-drag callback so a seek bar acts once
- [x] `core.TextGrid` — a monospace grid of styled runs on all four targets,
      rows as children so a terminal diff patches one row, not the grid

---

## 🔜 Planned

### 📦 Packaging
- [ ] `grmob build --target=wasm`
- [ ] `grmob build --target=android`
- [ ] `grmob build --target=ios`
      (today: `android/build.sh`, `ios/build.sh`, and `GOOS=js GOARCH=wasm go build ./wasm`)

### Native Bridge (Planned for Android/iOS)

- [ ] Keystore (Secure): `Keystore.Save()`, `Keystore.Get()` — the church app
      keeps its bearer token in bytdb for want of this; see its README
- [ ] Clipboard: read/write — cats-mobile wants paste into its composer
- [ ] URL-scheme deep links (`cats://pair` from a QR lands in the app)
- [ ] Haptics (cats-mobile: a buzz when an agent blocks)
- [ ] Local notifications
- [ ] Device Storage (Plain): `DeviceStorage.Set()`, `DeviceStorage.Get()`
- [ ] Bluetooth: `Scan`, `Connect`, `Send`
- [ ] Location / GPS
- [ ] FaceID / Biometric authentication
- [ ] Contacts access

### 📱 Native Runtime Bridges

- [ ] **Still thinking** for desktop

### 🛠️ DevTools
- [ ] State inspector overlay (similar to React DevTools)
- [ ] Patch logging — a debug-mode trace of what each pass emitted
- [ ] Visual patch viewer
- [ ] Hot module replacement

### 🧪 Testing & Perf
- [ ] Benchmark diff/patch engine
- [ ] Latency profiling in runtime patching

### 🧬 Extensions
- [ ] Router-style navigation for web

### 🎨 Styling gaps
- [ ] `HoverStyle`, `FocusStyle` and `PseudoStates` — the fields exist on
      `core.Style` and merge correctly, but no renderer reads them. Inline
      styles cannot express a pseudo-state, so the web targets need a
      generated stylesheet and class names, not another declaration.
- [ ] `Animation` (`"bounce 2s infinite"`) is emitted by both web targets and
      is inert until the hosting page defines the matching `@keyframes`;
      neither native reads it.
- [ ] `FlexDirection` on the natives — Compose and SwiftUI take the axis from
      the node type (`Row`/`Column`), so an explicit direction is web-only.

---

## 🌍 Vision

> Build **native experiences** using only Go.

---

## 💬 Contribute

Open to collaborators and contributors.
Join the discussion or check the GitHub repo.

---

