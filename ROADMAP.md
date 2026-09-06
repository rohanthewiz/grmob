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
- [x] `core.ZStack` — the z-axis container, and the one thing `Box`
      deliberately is not: every child drawn in the same box, in tree order,
      last on top. A SwiftUI `ZStack`, a Compose `Box` and a single-cell CSS
      grid, all told to *centre* rather than left to their own defaults. Not
      built from `Position`/`ZIndex` below, which only two of the four targets
      read — naming the container is what lets each renderer reach for its own
      construct. Sizes to its largest child; a layer that wants to sit
      elsewhere states its own box rather than reaching for a per-child
      alignment prop that does not exist. First consumer: `Compass`
- [x] `Position` (`Sticky`/`Absolute`/`Relative`/`Fixed`) with `Top`/`Right`/
      `Bottom`/`Left`/`ZIndex`, plus `MinWidth`/`MaxWidth`/`MinHeight`/
      `MaxHeight`, `Overflow`, `WhiteSpace`, `AlignSelf`,
      `FlexBasis`, `FlexShrink` — **web targets only**
      (WASM DOM and `htmlout`). Compose and SwiftUI have no direct equivalent
      for out-of-flow placement; a layout that depends on these will not look
      the same on device. The one exception is `Position: sticky` on a `List`
      child, which both natives now honour as a pinned header — see
      `core.StickyHeader()` below.
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
- [x] `core.Horizontal()` on `Scroll` — the sideways strip (chips, tabs, a
      card carousel) on all four targets. It spells itself in `FlexDirection`
      + `Overflow`, so both web targets already drew it and the natives learnt
      to read the axis (`horizontalScroll`, `ScrollView(.horizontal)`)
- [x] `core.StickyHeader()` on a `List` child — pinned group bands on all four
      targets. The marker is `Position: sticky`, which the web has always
      emitted; the natives now honour it with `stickyHeader {}` and
      `pinnedViews: .sectionHeaders`

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
- [x] An **on-light tone** per status role (`PrimaryOnLight`, `SuccessOnLight`,
      `WarningOnLight`, `ErrorOnLight`) — the second value a role needs when it
      is spent as *ink* on a light surface rather than as a fill with an ink
      picked over it. Five of the eight bundled role colours failed the 4.5:1
      body-text floor that way; all eight tones clear it. Read through the four
      resolvers, or through `Colors.OnLight(colour)` for a widget holding a hex
      it has no name for. An unset tone falls back to its own role, so a theme
      written before these renders exactly as it did. Spent by `Button`'s
      outlined and ghost treatments, `Chip`'s loud prominence, `Banner`'s
      hairline and glyph, and `StatTile`'s delta — every one of which
      previously documented the gap it could not close from where it sat

- [x] The **ink over a fill**, resolved by asking the theme before measuring
      (`components.Variant.Ink`). `Components.Button` is the one place a
      palette states a fill and an ink together, so a fill matching it takes
      the ink it was declared with; everything else falls to the higher WCAG
      contrast of the theme's two ink roles. Pure measurement picks *black* on
      `DefaultTheme`'s `#007AFF` (5.23:1 against white's 4.02:1), which is how
      `Calendar` came to draw a black numeral on iOS system blue while every
      filled `Button` beside it painted white. No third ink role could have
      fixed it — nothing outscores black on a mid-tone — so the question
      changed rather than the candidate list

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
- [x] `GroupedList` and `DataTable` — keyed, virtualized collections over
      `core.List` with run-length group headers, controlled sort, compact
      mode and client- or server-side paging; `Pagination` and `LoadMore`
      footers (tutorial lesson 4.6; plan in `ai_docs/plans/components-datatable-compass-map.md`)
- [x] Screen furniture — `AppBar` (automatic back off `core.CanPop`),
      `Banner` (variant on the edges, never the fill), `EmptyState` (empty,
      busy and failed in one shape), `SearchField`, `ChipStrip`, `Skeleton`
      and `StatTile`, all stateless and controlled; plus `hooks.UseDebounce`,
      the re-arming timer `UseTimeout` deliberately is not (tutorial lesson
      4.7)
- [x] Infinite feeds — `core.OnEndReached` on `core.List` (debounced in Go by
      the row count at the last fire, so a slow fetch cannot double-load),
      wired through `GroupedList.OnEndReached`; `StickyHeaders` on
      `GroupedList` and `DataTable`, and `ChipStrip.Scrollable`
- [x] `Calendar` and `DatePicker` — a controlled month grid (fixed six rows,
      dimmed inert adjacent days, a `Today` the caller supplies rather than a
      `time.Now()` the widget reads, `Min`/`Max` by calendar day, `Marked`
      dots) and the field that opens one in a sheet, owning only the two view
      states no application wants. Cells are built at *midday*, because
      midnight is a local time that does not exist on every day in every zone
      (tutorial lesson 4.9)
- [x] `Calendar.Marked` counts (`func(time.Time) int`) rather than answering
      yes or no: one dot per thing on the day, capped at three, in a cluster
      that always holds at least one box so the numerals keep one baseline and
      the nothing-to-one case stays a colour patch
- [x] `Calendar.Deselectable` — a second tap on the chosen day reports the
      zero time through `OnSelect`, which is what `Selected` already means by
      it, so a grid used as a filter clears itself. Off by default and forced
      off by `DatePicker`: a form's date setter must not receive a clear it
      cannot tell from a pick
- [x] `Compass` — a bearing as a rose that turns under a fixed index mark,
      drawn over the rim on a `core.ZStack` (it sat in the row *above* the
      circle until core had a z-axis container) and announcing itself as one
      spoken sentence because four letters whose positions carry the meaning
      are exactly what a screen reader cannot convey (tutorial lesson 4.10)

### 🧬 Extensions
- [x] Animations & transitions (`Transition`, easing curves)
- [x] `core.Rotate(deg)` — a paint transform on all four targets
      (`transform: rotate()`, `Modifier.rotate`, `.rotationEffect`), about the
      node's own centre, with the winding deliberately left unnormalised so an
      animated bearing does not unwind a full turn every time it passes north.
      Layer order is pinned in both natives by `mobile/verify/rotate_test.go`:
      a rotation applied below the background turns the content inside a box
      that stays square, which compiles and animates and is still wrong
- [x] Heading sensor — `core.StartHeading`/`StopHeading` (refcounted, so two
      screens can each hold the compass), `core.CurrentHeading`/`OnHeading`
      and `hooks.UseHeading`, over one `"sensor"` system event carrying a
      `kind` so location and motion reuse the channel. `SensorManager`'s fused
      rotation vector on Android, `CLLocationManager` on iOS,
      `deviceorientationabsolute`/`webkitCompassHeading` in the browser, and
      an honest `available: false` on a desktop — which is a different answer
      from "no reading yet", and a screen draws different things for the two
- [x] **Permissions** (`permission`) — `Check` never prompts and `Request`
      does, which is the whole reason they are two functions: a check that
      prompted would put the OS dialog up as a side effect of a screen
      mounting, and a request that only checked would leave a button that does
      nothing. Shaped like the compass minus the refcounting — `Check`/`Request`
      out over the `"permission"` system event, the answer back over the
      `"permission"` host event into a record a screen reads with
      `permission.Current` or `hooks.UsePermission` — so there is no callback
      to leak and no request id to correlate. Four statuses, and the fourth is
      the one usually left out: `Denied` is fixable in the system settings and
      `Unavailable` is not (no camera, an iOS restriction, an undeclared
      Android permission, no host attached at all), so a screen offering "Open
      Settings" for both sends someone to a page with no switch on it. The
      spellings are the W3C Permissions API's, for the reason the roles are
      ARIA's. It is deliberately *not* a second way to do what
      `core.StartHeading` already does — it exists for the rationale shown
      before a dialog and for the status read back to draw a settings row,
      the second of which the compass named a session before it landed
      (`Heading.HasTrue` needs location authorization and the compass host
      prompts for nothing on purpose)
- [x] Accessibility labels, hints and announced selection state
- [x] Accessibility *roles* (`core.AccessibilityRole`, twenty-three ARIA-spelled
      values) — `role=` on both web targets, traits on SwiftUI and semantics
      on Compose where those vocabularies reach, and an explicit no-op arm
      where they do not; pinned in both natives by `mobile/verify/role_test.go`.
      Adopted by `DataTable` (table/rowgroup/row/columnheader/cell), `AppBar`
      (banner + heading), `Banner` (status/alert by variant), `SearchField`
      (search) and `Calendar`'s day cells (button). `RoleImg` joined for
      `Compass`, and is the case that showed the vocabulary was doing more
      than naming things: ARIA forbids an accessible name on a generic
      element, so a labelled container had no announced name on either web
      target while both natives read it fine. `RoleTab`/`RoleTabList` and
      `RoleLog` complete the vocabulary for a hand-built tab strip and a chat
      transcript (`examples/chat`); the tab pair is the one row where the two
      natives disagree about which half they can say — Compose has `Role.Tab`
      and no strip, SwiftUI `.isTabBar` and no tab. There is deliberately no
      `RoleTabPanel`: a panel is one end of a relationship whose other half is
      an IDREF a `Style` cannot carry, and `core.TabView` owns both ends.
      `RoleListBox`/`RoleOption` unblocked a widget rather than described one:
      `aria-selected` is scoped to `option` and not to `listitem`, so a
      selectable row had no role that could carry its state — see
      `ListRow.Selectable` below
- [x] **A name on a plain container is announced at all** (`core.RoleGroup`) —
      the general answer to the failure `RoleImg` closed for one widget. ARIA
      prohibits an accessible name on the `generic` role a `<div>` and a
      `<span>` carry and browsers prune it, so an `AccessibilityLabel` on any
      layout node was read out by VoiceOver and TalkBack and by nothing on
      either web target — a two-target silence that hit every labelled
      `ListRow`, every `Accordion` header, every named `StatTile` and
      `Skeleton`. Both web exporters now *supply* `role="group"` to a node with
      a name, no role and a generic tag, so no call site changed. `group` is
      the smallest role that makes a name legal — nameable, not a landmark, no
      required children, children not made presentational — which is what lets
      it be given to a container nothing has looked inside; `region` would put
      six rows in a screen's table of contents and `button` would silence the
      heading inside an `Accordion` header. It is the one arm both natives
      leave empty because they do not *need* it rather than cannot say it
- [x] **A hand-built tab strip can point at its panel**
      (`core.AccessibilityID` / `core.AccessibilityControls`) — the
      vocabulary's only two *references*, where everything else on `Style` is a
      value. `core.TabView` mints its own ids and writes the whole tab/panel
      wiring from the node type, so the wired case never needed them; a strip
      assembled out of chips — which is what you build for a bar that looks
      different — could say `role="tab"` and `role="tablist"` and then had no
      way at all to say which region each tab shows. The line that keeps this
      from becoming a second ARIA: *a reference prop earns its place only when
      what it points at cannot be said as a value* — `aria-labelledby` and
      `aria-describedby` point at text `AccessibilityLabel` and
      `AccessibilityHint` already carry, and `aria-controls` points at another
      element, which no string stands in for. Adopted by `examples/social`'s
      bottom bar, the app the gap was noticed in. Neither native reads either
      key: no such relationship exists in their vocabularies, and the near miss
      (`accessibilityIdentifier` / `testTag`) is a test selector rather than an
      accessibility property
- [x] A widget can say **this disclosure is open** (`core.AccessibilityExpanded`)
      — `aria-expanded` on both web targets, scoped to a *third* ARIA role list
      that is neither `aria-level`'s nor `aria-selected`'s: it drops `option`
      and adds `link` and `listbox`, which is why `core.ExpandedState` is a
      type of its own rather than `SelectedState` reused. Three values for the
      reason that type has three: a collapsed section that answers nothing is
      announced as an ordinary button, and "collapsed" is the whole of what
      invites the press. The two natives disagree here, which is the reverse of
      the usual split — Compose has `expand`/`collapse` semantics *actions* and
      is given the one the state calls for, wired to the node's own click
      callback, while SwiftUI has no expanded trait and its near miss
      (`accessibilityValue`) would mean shipping an English literal to every
      locale. Adopted by `components.Accordion`, whose header row became a
      button inside a heading — ARIA's own accordion shape, reachable once
      `AccessibilityLabel` was noticed to override the content-derived name the
      wrapper had been turned down for
- [x] A widget can say **this control is on** (`core.AccessibilitySelected`) —
      `aria-selected` or `aria-pressed` on both web targets (the role picks,
      which is `aria-level`'s switch running the other way), `selected`
      semantics on Compose, the `.isSelected` trait on SwiftUI. Three values,
      not a bool: a tablist in which only the live tab answers announces the
      other four as furniture, so `SelectedOff` is a value with a job. Adopted
      by `Chip` (and through it `SegmentedControl` and `ChipStrip`) and by
      `Calendar`'s day cells, both of which dropped the `", selected"` suffix
      they had been spelling into the accessible *name* for want of a slot.
      `ListRow` was the holdout for three more sessions and is now the third:
      `Selectable` makes the row an `option`, the caller puts `RoleListBox` on
      the container, and the suffix goes. A row is a choice *or* a depth, never
      both — `option` takes the state and no `aria-level`, `listitem` the
      reverse — and ARIA's role for an item that is both (`treeitem` in a
      `tree`) is deliberately not in the vocabulary
- [x] Accessibility *heading levels* (`core.AccessibilityHeadingLevel`) —
      `aria-level` on both web targets and `.accessibilityHeading` on SwiftUI;
      Compose has no level to map onto and says so. `AppBar`'s title takes 1
      and `GroupedList`'s band labels 2, so a banded screen has an outline
      instead of a flat run of peer headings
- [x] The **outline goes past 2**: `Card.Title` is a heading at 2 and
      `Accordion.Title` one at 3, which were both drawing at heading weight
      and announcing as prose. Only an `AppBar`'s tier is fixed by
      construction, so the other three take a `HeadingLevel` field whose zero
      value is the default tier — a grouped list inside a card says 3, a feed
      on a barless screen says 1 (previously unsayable), and 4 through 6 are a
      caller's to reach. A negative level asks for a heading with no tier,
      which the drop-don't-clamp rule already spells
- [x] `core.Modal` announces as a dialog — `role="dialog"` + `aria-modal` from
      the Modal chassis on both DOM targets, which the SwiftUI sheet and the
      Compose `Dialog` already provide on device; deliberately not a
      `RoleDialog` an author has to set
- [x] Accessibility *nesting levels* (`core.AccessibilityNestingLevel`) —
      `aria-level`'s other two roles, `listitem` and `row`, on both web
      targets; neither native has a nesting-depth property and both say so. A
      second field rather than a widened heading one, because a heading's tier
      stops at 6 and a depth has no ceiling, and one exporter switch on the
      role keeps the two from ever contending for the attribute
- [x] …and its **first consumer**: `ListRow.NestingLevel` makes a row a
      `listitem` at a depth, which is the one thing a flattened outline cannot
      say any other way — a list is a flat run of siblings, so an indent is
      pixels a screen reader never sees. Opt-in, because a `listitem` is owned
      by a `list` and a row cannot see its container: the caller roles the
      list. It lands where the selected state could not, and the difference is
      the role rather than the widget — a depth's is `listitem`, a selection's
      is `option`, which `core.Role` still does not carry
- [x] `components.Button`'s border means the same thing on all four targets —
      both natives now feed `BorderColor`/`BorderWidth` into the platform
      control's own slot (they were stripped with the rest of the box-drawing
      fields and never fed back, so outlined buttons had no rule on device),
      and both DOM renderers write `border:none` for the node types a browser
      draws one on (so ghost buttons no longer keep the user agent's)
- [x] A **text field's frame** is the theme's on all four targets — both
      bundled themes give `Components.Input` and `Components.TextArea` a
      border at WCAG 1.4.11's 3:1 control-boundary floor, which is what let
      `<input>` and `<textarea>` join the user-agent border reset. That set is
      keyed by node type now rather than by tag, because five node types share
      `<input>` and a `Checkbox` and a `Slider` are drawn by the browser in
      their entirety. `core.Select` joined the set on the same test once it
      existed, and its drop-down indicator is untouched for the checkbox's own
      reason: it is the thing that says the control is a picker. `Colors.Border` keeps the dividers and no longer names
      field edges; `DatePicker`'s trigger inherits the whole frame off the
      `Input` base instead of restating a paler one. `Colors.ControlBorder` is
      the palette role that arrived a session later, when `Chip` turned out to
      be a second spender: a quiet chip's ring is not a rule between things, it
      is the only edge a filter control has, and it was drawing it out of the
      1.26:1 divider. The two component bases still state their frame as a
      literal — a `Style` is a value and cannot call a resolver — and a test
      pins them to the role
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
- [x] `core.Select` — the picker, on all four targets: a `<select>` whose
      options are built from a prop, a SwiftUI `Menu`, a Compose
      `DropdownMenu`. `onChange` carries the option's *value*, never its label
      or its index, so a rule reads it unchanged and `forms.Required` rejects
      an unchosen one; `form.Select` is the bound builder. Deliberately not
      built from either platform's own picker control — `.pickerStyle(.menu)`
      and `ExposedDropdownMenuBox` each draw a frame no Go style can remove —
      so the frame is the theme's `Components.Input` base everywhere, which is
      also what lets `<select>` join the user-agent border reset above
      (tutorial lesson 5.6, `examples/signup`)
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
- [x] Hot module replacement — WASM only (`go run ./serve -dev`: watch,
      rebuild, swap the module in place; the lesson and scroll survive, Go
      state does not). Native has no equivalent; see docs/platforms/wasm.md

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
      the node type (`Row`/`Column`), so an explicit direction is still
      web-only everywhere except `Scroll`, which reads it to choose between a
      vertical and a horizontal scrolling region (`core.Horizontal()`).

---

## 🌍 Vision

> Build **native experiences** using only Go.

---

## 💬 Contribute

Open to collaborators and contributors.
Join the discussion or check the GitHub repo.

---

