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
      construct. Sizes to its largest child. First consumer: `Compass`
- [x] `core.StackAlign` — the per-layer opt-out from that centre, on all four
      targets: nine placements, the centre unspelled so it stays the zero
      value. A layer used to state its own box instead (a full-height column
      justifying its child to one end), which is what `Compass` did and what
      kept the prop out while `Compass` was the only consumer. What made it a
      vocabulary is that a SwiftUI `Alignment`, a Compose `Alignment` and a
      CSS grid item's `justify-self`/`align-self` are the same nine values, so
      the prop is portable where the flexbox `AlignSelf` above is web-only.
      The web half is *imposed by the stack* rather than written by the layer,
      because `align-self` means something else to a flex child — which also
      closes a leak: a layer's own `AlignSelf` used to move it on the two DOM
      targets and nowhere else. iOS carried the one divergence for two
      releases — a filling frame is greedy — and no longer does; see the
      overlay `Layout` entry below
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
- [x] Per-side padding props — `PaddingTop`/`Bottom`/`Left`/`Right` set one
      inset without restating the other three through a whole `EdgeInsets`.
      Each dissolves its axis's shorthand first, so a zero really clears;
      no renderer changed. `Margin` has the same six props now, and the
      three-width ordering rule is in docs/concepts/styling-and-theming.md.
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
      iOS systemBlue `#007AFF` (5.23:1 against white's 4.02:1), which is how
      `Calendar` came to draw a black numeral on system blue while every
      filled `Button` beside it painted white. No third ink role could have
      fixed it — nothing outscores black on a mid-tone — so the question
      changed rather than the candidate list

- [x] `DefaultTheme.Colors.Primary` darkened to Apple's accessible blue
      `#0040DD`, the hex `PrimaryOnLight` already carried. The declared pair
      was white on systemBlue at 4.02:1, under WCAG AA for body text, and it
      was spent by every filled `Button`, `Badge`, `Avatar` and `ProgressBar`
      fill plus `Calendar`'s selected day — a pairing, which no second tone
      can reach. `Components.Button.Background` moves with the role and is
      pinned to it (`TestBundledButtonFillsAreThePrimaryRole`): move one
      without the other and every *other* Primary fill loses its declaration
      and falls back to measurement. The payoff is that `VariantDefault` joins
      the two filled-legibility censuses it had to be exempted from; the cost
      is that neither bundled theme can still demonstrate the declaration
      beating the measurement, which a test fixture now carries

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
- [x] **A permission re-checked on the way back in** (`permission.WatchForeground`,
      `hooks.UsePermissionLive`) — no platform announces a permission changed
      in Settings, so a screen that sent the user there was still drawing its
      `Denied` state when they came back: the button that fixed the problem
      was the thing still saying it was broken. The signal every platform does
      send is the foreground transition, and this turns it into a `Check`.
      What kept it open for five sessions was "which screen owns the
      re-check", and the answer is that no screen does — there is one device
      with one camera, so the *permission* owns it, reference-counted by kind
      the way `core.StartHeading` counts the sensor. Five screens watching the
      camera are one check per resume between them, an app with no live
      watcher takes no lifecycle subscription at all, and a resume that
      changed nothing stops at the record because it notifies only on a change
- [x] **The Android asked-before flag survives a restart** — the shell
      reconstructs `Prompt` from `Denied` by remembering whether this install
      has asked, because `shouldShowRequestPermissionRationale` is false both
      for "never asked" and for "don't ask again". That flag was in memory, so
      every cold start reported a permanent refusal as `Prompt` and the first
      thing the user saw was a button that did nothing. It is in
      `SharedPreferences` now: the fact is the platform's own bookkeeping,
      which Android keeps and will not answer, so it belongs in the one file
      that knows about it. The request path stopped short-circuiting on
      `Denied` in the same pass — Android 11+ auto-reset clears
      don't-ask-again without clearing this flag, and the launcher is the only
      thing that can say so
- [x] **A browser request reads before it opens the device** — a granted
      camera request used to call `getUserMedia` to confirm a permission the
      browser had already written down, lighting the recording indicator to
      answer a question nobody asked. Every request now queries first and only
      reaches for the device in the one state where a request has something to
      do. The refusal is read back too: `NotAllowedError` cannot say whether
      the user pressed Block or dismissed the prompt, and the Permissions API
      can — a Block is recorded `denied`, a dismissal leaves `prompt` — so the
      two reach Go as different words, and only a browser with no descriptor
      for the kind still collapses them
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
- [x] **The keyboard half of the listbox and tablist patterns** (WASM runtime)
      — the two pairs above name real ARIA *controls*, and a control's pattern
      is behaviour as well as attributes: one tab stop for the widget, arrow
      keys between its members, a roving `tabindex` saying which member holds
      it. Three shipped screens claimed a pattern nothing implemented —
      `examples/mobileapp`'s article list was a listbox of `<div>`s no keyboard
      could reach at all, `examples/social`'s bottom bar and tutorial 4.5 were
      strips a keyboard could cross only by tabbing through every member. The
      entry said it needed a focus concept `core` does not have; it did not.
      Everything the pattern wants was already on the wire — the roles say what
      contains what, `aria-selected` says where a keyboard enters, the
      container's own axis says which arrows move, the author's `onClick` says
      what activation means — so the runtime reads it and no screen, widget or
      `core` type changed a line. `core.TabView`'s own bar gets it for free.
      `htmlout` deliberately writes none of it and that is the one intended
      difference between the two DOM targets: a roving tab stop with nothing to
      move it takes every member but one out of the tab order and reaches none
      of them, so `tabindex` is behaviour rather than semantics and a static
      export must not carry it (`wasm/verify/keynav_test.go` holds both
      directions). Both phones never had the gap — VoiceOver and TalkBack
      navigate a collection by swipe
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
- [x] **A disclosure has a second consumer** (`components.Collapse`) — a
      `GroupedList` band can be shut, and `Accordion` stopped being the only
      widget in the package that says whether something is open. What the
      second consumer bought is the shared shape: ARIA's disclosure
      arrangement — a heading wrapping a button carrying `aria-expanded`, named
      explicitly so the chevron never reaches the outline entry — took three
      attempts to land, and both rejected versions looked correct in an export.
      It is one value now (`components.disclosure`) and the two widgets are
      rendered side by side and compared. The collapse state is the *caller's*,
      which keeps `GroupedList` hook-free and is the right owner anyway: which
      months are shut is screen state that wants to survive a pager reload. A
      shut run emits no rows at all rather than hidden ones, and the band keeps
      its key across the toggle, so re-opening patches rather than remounts
- [x] **`Margin` has the four sides and the two axes**
      (`core.MarginTop`/`Bottom`/`Left`/`Right`/`Horizontal`/`Vertical`) —
      padding got its per-side props a session earlier and margin was left with
      `Margin(all)` alone, so every single-sided gap went through a whole
      `EdgeInsets` in a `UseStyle`. That is worse on this field than it was on
      padding: `UseStyle` replaces `Margin` outright and a margin's other three
      sides are usually zero, so the struct looks like it set one gap while
      clearing the rest. Nothing in any renderer changed — the same
      `EdgeInsets`, the same two settle helpers, the same resolution — and both
      live workarounds (`components.Separator`'s inset, `examples/chat`'s
      bubble gap) are one prop each
- [x] **The bridge stand-in is pinned by signature, not just by name**
      (`mobile/verify`) — `ios/verify` type-checks the iOS app layer against a
      hand-written stand-in for the `gomobile bind` module, and the stand-in
      was checked only for *which* symbols it declared. The argument for
      stopping there — a wrong signature fails the Swift type-check the moment
      the shell calls it — assumed every declaration has a call site, and three
      do not. Each declaration is now compared character-for-character against
      what gobind would emit for the Go signature behind it, off a three-row
      type table that is all this deliberately narrow bridge can need. It keeps
      gobind's nullability asymmetry, which is the point of copying it: a stub
      taking `String` everywhere would accept shell code the framework rejects
- [x] **The last control boundary under 3:1 was retinted, and the census is
      what made that cheap** (`core.DefaultTheme`) — `Colors.ControlBorder` was
      iOS `systemGray` `#8E8E93`, which clears against that theme's page at
      3.26:1 and falls to 2.92:1 against its `Surface`, the quiet chip's own
      fill. It stood exempt for three sessions with a sound argument attached
      (the edge that identifies a pill is the outer one; the inner pair is a
      boundary between two parts of one control). What retired it was not a
      better argument but a cheaper alternative: while evaluating a candidate
      tone meant auditing every fill a boundary lands on, defending the pair
      was less work than fixing it, and once the census *is* that list it costs
      one test run. `#89898E` clears all four backdrops and differs by 5/255
      per channel. `knownBoundaryShortfalls` is empty now, which is its resting
      state — it exists so the next pair has to be fixed or defended in writing
- [x] **The inset props' width lattice is stated and pinned**
      (`core/inset_width_order_test.go`) — `Padding(all)` over the two axis
      props over the four sides, on both families, and one asymmetry: a
      narrower prop can clear a wider one because it settles the axis first, a
      wider prop cannot preserve a narrower one because it writes every side it
      covers. So the wider brush goes first, every combination is expressible,
      and there is deliberately *no* prop for "this side survives the next axis
      prop" — it would be the one `StyleProp` whose effect outlives what is
      written after it. What makes the rule reachable is a separate guarantee
      the twenty-odd widget `Style` fields promise in prose and nothing
      checked: a widget applies its own insets before the caller's `Style`,
      never after. Six widgets now assert it from the outside
- [x] **A single-widget knob in `rowsSpec` has an admission test**
      (`components/rows_spec_test.go`) — the census counted the two
      widget-specific fields in the shared parameter list and said a third was
      "worth asking about", which is a prompt without an answer. The answer is
      a property of `appendRows`' output rather than a matter of taste: every
      child it emits is a `core.Keyed` closure, so a band, a separator and a
      row are one dynamic type with the key sealed inside until render time and
      the item captured out of reach. A knob that can be applied to that slice
      belongs in a wrapper the widget applies itself; one that cannot belongs
      in the spec. Both current fields fail the wrapper test, for the only two
      available reasons — `Wrap` needs the item, `Collapse` needs rows *not to
      be produced*. The owner column is now derived from the widgets rather
      than trusted as a string
- [x] **Every palette rule has a recorded witness**
      (`components/palette_witness_test.go`) — two rules were invisible under
      both bundled themes: `inkOn`'s first step (read the theme's declared
      pair before measuring) and `Colors.OnLight`'s `Primary` arm. An
      implementation that deleted either would paint identical pixels and pass
      everything. The census records, per rule and per theme, which themes can
      still *show* each rule working, so losing the last witness is a failure
      rather than a silence — which is exactly what happened once before: the
      two-step ink rule's witness *was* `DefaultTheme`, and a straightforward
      retint took it away with nothing to report it
- [x] **`core.AmberTheme` — a third bundled palette, written to be the witness**
      (`core/theme.go`) — MD amber 700 as the brand, which is an excellent
      *fill* and 2.04:1 as ink on white, so its `Primary` role and its on-light
      tone are genuinely different colours; and a `Components.Button` declaring
      MD brown 900 over that amber where measurement would pick the page's
      near-black at 8.39:1. Two rules that rested on a test fixture now rest on
      a shipped palette. Material provenance throughout bar one value, which is
      amber 700 scaled to 56% because the family carries no swatch dark enough
      to be read as ink
- [x] **Why the *strong* form of that rule cannot be shipped, as arithmetic**
      (`TestThePoleFlipBandIsTooNarrowToShip`) — the dramatic case is a
      declaration choosing the opposite ink *pole* from the measurement, and it
      is unshippable rather than merely absent. A declaration that loses the
      measurement is the lower-contrast of the two inks by definition, and the
      two ratios against any fill multiply to a palette constant of at most 21,
      so the loser can never exceed √21 = 4.58:1 while the AA check asks 4.5:1.
      The band is 1.8% wide — and it only exists at all when the palette's inks
      are near pure black and white, so two of the three bundled palettes could
      not host such a fill at *any* brand colour. That half stays with the
      fixture, provably rather than accidentally
- [x] **`core.BundledThemes()` — one list, derived from the source**
      (`core/bundled_themes_test.go`) — a dozen tests each spelled their own
      two-entry map of "the bundled themes", so a third palette would simply
      have been absent from every census rather than failing one. The list is
      now one function, and `TestBundledThemesListIsExhaustive` parses
      `theme.go` for package-level `var X = &Theme{}` declarations and requires
      each to appear in it — guarding the shared list with a *derived* one
      rather than a second hand-written one
- [x] **A `StackAlign` outside a stack is reported** (`inert-stack-placement`,
      `core/placement_audit.go`) — the prop is inert on a `Column`'s or a
      `Row`'s child deliberately, and was silent about it on all four targets:
      it compiles, merges, crosses the bridge and is read by nobody. A fifth
      arm on `core.AuditTree`'s walk, which is where a fact about a node *and
      its container* can be checked at all — an exporter writing one element
      has no index of the document. It walks by the *placing* container rather
      than the tree parent, so a `core.For` inside a `ZStack` is fine and a
      stack's grandchild is not
- [x] **The iOS overlay places without filling** (`ios/GrMob/Runtime/GrMobStack.swift`)
      — SwiftUI has no per-child `ZStack` alignment, so a placed layer was
      wrapped in `.frame(maxWidth: .infinity, maxHeight: .infinity,
      alignment:)`. That is SwiftUI's own idiom and it is *greedy*: an unsized
      stack with an aligned layer grew to its parent's proposal on iOS while a
      Compose `Box` and a CSS grid track stayed the size of their largest
      child. Documented in four places, avoidable by pinning the box, and
      pinned nowhere. A custom `Layout` replaces it, and the arithmetic is a
      pure CoreGraphics file for the reason `GrMobFlexSolver` is one — so
      `ios/verify` now *measures* the overlay (largest child per axis, all nine
      anchors, the bounds' origin, an oversized layer overhanging) where before
      it could only check that the file compiled
- [x] **Every shared `rowsSpec` knob has an effect assertion of its own**
      (`components/rows_spec_test.go`) — the census required a new field to have
      a *row*; nothing required it to have an *assertion*, so a knob marked
      "both", forwarded by one widget and never asserted passed everything. The
      nine shared knobs are now a table keyed by field name, driven from the
      census in both directions, and run against both widgets — which also
      bought `Row` and `Header` the assertions they never had, and separated
      `Rows` (the row count) from `Key` (the row keys)
- [x] **Every control boundary is measured, not just the two named ones**
      (`components/variant_test.go`) — `Colors.ControlBorder` clears WCAG
      1.4.11's 3:1 floor against a page and falls 0.08 short against
      `DefaultTheme`'s `Surface`, which is the quiet chip's own fill. That was
      argued in two prose blocks and asserted nowhere, so a *second* widget
      drawing a boundary on `Surface` would have inherited the shortfall
      without inheriting the argument. A census now crosses each bundled tone
      with every fill a control can sit on and requires each pair to clear or
      to be recorded, with its number and its reason, in one exemption table —
      and a sibling test deletes the exemption if a retint ever closes the gap
- [x] **A composite announces the way it runs** (`aria-orientation`, both web
      targets) — the runtime read a container's resolved `flex-direction` to
      pick the arrow pair and nothing wrote the answer down, so a
      `role="tablist"` laid out as a `Column` took Up/Down while telling a
      reader in browse mode that it ran the other way (ARIA's default for a
      tablist is horizontal; a `listbox` had the same gap in mirror). Both
      targets now write the attribute from the node's own layout axis, resolved
      exactly as the CSS declaration is, and the runtime's keyboard *reads it
      back* — so which way a widget runs is one statement rather than a
      behaviour and an announcement free to drift. `toolbar` takes the
      announcement and no keyboard, since ARIA does not say what a toolbar owns
- [x] **A progress bar is a progress bar** (`core.RoleProgressBar` +
      `core.ValueRange`) — the fourth accessibility state vocabulary, and the
      first that is four attributes at once. `components.ProgressBar` had
      nowhere to put its percentage but the accessible *name* ("Upload, 45
      percent"), which is the channel `Chip`'s `", selected"` suffix was
      deleted from and for the same reason: a name is meant to be stable, so a
      bar ticking from 44 to 45 re-announced the whole string. The three
      numbers are one field because they are one fact in three parts, and they
      are *strings* because a bar at the start of an upload is a stated `0`
      that a float field could not tell from an unstated one. Compose gets the
      better half — `progressBarRangeInfo`, which TalkBack localizes itself —
      and SwiftUI, which has no numeric value slot at all, gets the words if
      the app supplies any. `Skeleton` took `status` and `FormField`'s required
      marker took `img` in the same pass, both of them `group`s until now
- [x] **A hand-built tab strip can name its panels** (`core.RoleTabPanel`) —
      the last piece the reference pair was missing, and it had been blocked by
      its own absence: the WASM runtime told a panel it wired from one an
      author roled by the *value*, since `"tabpanel"` was a string no
      `core.Role` spelled. A `data-grmob-panel` marker both web targets write
      says the same thing about the element rather than about the vocabulary,
      in the channel `data-grmob-chrome` already uses. It also closed a bug the
      old shape hid: the unwire path cleared the panel id unconditionally, so a
      page carrying its own `AccessibilityID` — exactly the page the wiring
      stands down for — was stood down for by having that id deleted
- [x] **A listbox answers the keyboard by name** (type-to-jump) — the other
      half of ARIA's listbox pattern, and the half that makes a long one usable
      at all. A repeated character cycles and a growing string refines, which is
      one rule rather than two. It is the only *state* the keyboard section
      owns; everything else there is derived from the DOM on demand, which is
      what makes the rest survive every patch for nothing. One buffer rather
      than one per widget (only one thing has focus), and a timestamp rather
      than a timer (a widget removed by a patch has no unmount hook to cancel
      one from)
- [x] **The failures nothing could see are reported** (`core.SetDebugMode`) — a
      walk of the finished tree flags a duplicate `AccessibilityID`, an
      `AccessibilityControls` nothing answers to, an id that is not a usable
      HTML id or that lands in the reserved `grmob-` namespace, and an
      `AccessibilityExpanded` on a node with no handler. Every one of them is
      invisible on all four targets, and none is catchable where it is written:
      an export is one document with no index of itself and a patch is one
      element with none at all — a finished *tree*, though, can be walked, which
      is what the cursor audit and the duplicate-key check already do
- [x] **Every ARIA claim is checkable** (`aria/verify`) — the role lists were
      hand-checked prose in a dozen files, which agreed with each other because
      somebody had read them all. A Next-list entry once asserted that `group`
      supports `aria-expanded`; it survived three re-sorts and was false, and no
      test in the suite could have said so. One fixture now states each role's
      attributes, its ARIA default orientation and its required children, and
      the four state guards, the orientation table and the composite keyboard
      table are all held to it. It is transcribed rather than generated, which
      is the weaker half — what it buys regardless is that the fact is stated
      *once*
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
      draws one on (so ghost buttons no longer keep the user agent's). The WASM
      runtime reached that reset for every node once its create path stopped
      styling only the nodes that carried a `Style` — the patch path had always
      been total, so a styleless `<button>` used to keep the browser's rule
      until something restyled it and took the rule away
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
- [x] `SelectOption.Group` and `SelectOption.Disabled` — a heading over a run
      of options and a choice that is drawn, announced and unchoosable, on all
      four targets: an `<optgroup>` and a `disabled` `<option>`, a SwiftUI
      `Section` and a disabled `Button`, an unclickable heading item and a
      disabled item in the Compose dropdown. Grouping is by *runs* — the same
      heading either side of a different one is two sections, in the order
      written — because the list's order is the caller's and a gather would
      silently reorder it. Both keys are written to the wire only when they
      say something, so an ordinary option's JSON signature (which is what
      decides whether the WASM runtime rebuilds an open drop-down) is
      unchanged
- [x] `core.SelectMenuSections` — the one statement of how a flat option list
      becomes the menu a person sees. `htmlout` calls it; the other three
      renderers carry transliterations (`GrMobSelectMenu.swift` / `.kt`,
      `selectMenuSections` in `grmob-runtime.js`) because none of them can call
      into Go while drawing. Four copies of one rule were three too many, and
      the edge each copy had to remember on its own — that a run ending the
      list has nothing following it to close it — is exactly the one `htmlout`
      shipped wrong for a release
- [x] `SelectOption.GroupDisabled` — a whole run marked unavailable, on all
      four targets. **Any** option in the run states it, because a declaration
      written on the second entry and quietly doing nothing would have no way
      to be noticed: a menu is drawn behind a tap and there is no error channel
      there. That reading is what makes a run's state unknowable until the run
      is *closed*, so closing one is two steps now — the flush, and the walk
      back over the items collected before the state was known. What it buys
      over disabling each option by hand is the heading, which greys with its
      rows (`<optgroup disabled>`, a disabled SwiftUI `Section`), and the
      per-item propagation is what carries the refusal to the two targets with
      no section construct to disable. A heading still cannot take an icon, and
      that is a decision: an `<optgroup>`'s label is an attribute, so the web
      can hold text and nothing else
- [x] **One fixture, three transliterations** (`internal/menufixture`) — the
      picker-menu case table, compiled into all three harnesses, with every
      expected answer computed by `core.SelectMenuSections` rather than written
      down. `ios/verify` runs the Swift function, `android/verify` runs the
      Kotlin one on a JVM, `wasm/verify` mounts a picker and rebuilds the
      sections back out of the DOM. The table has an admission test of its own:
      the properties it exists to exercise are named with the predicate that
      finds a case covering each, so a case cannot disappear while the count
      stays plausible
- [x] **`android/verify`** — the Kotlin decomposition, executed. `GrMobSelectMenu.kt`
      imports nothing precisely so a plain JVM could run it, and nothing did:
      the Android build's only check was `compileDebugKotlin`, which proves the
      file parses. The obvious shape — a gradle test source set with JUnit — is
      the one it avoids, because this repository's Android build runs
      `--offline` and a check that needs a populated dependency cache is a
      check nobody runs. It compiles two files with `kotlinc` or, failing that,
      the compiler jars the gradle cache already holds, and skips when neither
      is there. The case table crosses as **Kotlin source**, not JSON: Kotlin's
      standard library has no JSON parser and neither does the JDK
- [x] **`wasm/verify/browser.mjs`** — the three keyboard facts a shimmed DOM
      cannot check, checked in a headless Chrome over the DevTools protocol
      with Node's built-in `WebSocket` (no npm, no network). That `tabindex="-1"`
      really removes a `<button>` from the tab order, that a disabled control
      refuses focus, and that `preventDefault` on `ArrowDown` really stops the
      page scrolling. Widening `dom.mjs` could only ever have restated them —
      there, `tabindex` is a string nobody reads, `focus()` is an assignment and
      `defaultPrevented` is a flag the shim set itself. Keys go through
      `Input.dispatchKeyEvent`, so the tab order is walked by the browser's own
      focus algorithm and a scroll is a real scroll; the pass skips when there
      is no Chrome to launch
- [x] **The rule for a second totality exemption** (`wasm/verify/totality_test.mjs`)
      — `styleFromGrMob` deletes a `Modal`'s `display` rather than assigning
      it, because the `visible` prop owns that property, and "abstain by
      deleting the key" is now a technique available to any property. Three
      conditions, stated and checked: a *prop* owns it, that owner writes it in
      **every** state (the one easy to miss — an owner that assigns only when
      truthy leaves the stale declaration totality exists to prevent, moved one
      channel over), and the exemption is keyed on the node type. The source is
      scanned for `delete out.X` and held to the table, because an abstention
      deletes the key before the declarations reach the element and is
      therefore invisible to any test that does not drive the owning prop
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

