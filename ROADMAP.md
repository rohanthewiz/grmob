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
      `FlexBasis` — **web targets only**
      (WASM DOM and `htmlout`). Compose and SwiftUI have no direct equivalent
      for out-of-flow placement; a layout that depends on these will not look
      the same on device. The one exception is `Position: sticky` on a `List`
      child, which both natives now honour as a pinned header — see
      `core.StickyHeader()` below. `FlexShrink` left this list in two steps and
      is on all four targets now: `GrMobFlexSolver` implements CSS's
      scaled-base rule, and Compose honours the one value it can express — see
      `core.ShrinkNone` below.
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
- [x] `StaticMap` — "where is this": a map image from a tile provider, which
      hands off to the platform's own maps app on a tap. A `core.Image` plus a
      `core.OpenURL`, so it works on all four targets with no renderer behind
      it, which is also the argument for building it before a live map view —
      an app asking "where is the church" wants a picture and then directions,
      and directions belong to the maps app with the user's own home address in
      it. Three decisions carry the widget: the provider is a *policy* (the
      keyless default is a volunteer service, named as such, with
      `GoogleStaticMap(key)` and a one-function seam beside it), the hand-off is
      one https URL because nothing here knows its platform, and a tappable map
      is `RoleLink` rather than `RoleButton` because the tap leaves the app.
      Latitude clamps at Web Mercator's limit and longitude wraps — two rules,
      two geographic facts (tutorial lesson 4.11)

### 🧬 Extensions
- [x] **The ARIA fixture is generated** — `aria/verify/testdata/aria.json` is
      produced by `aria/gen` from the W3C specification's own machine-readable
      role definitions (`aria/spec` reads `bind`-shaped feature cells out of the
      published HTML with no dependency). It was hand-transcribed for two
      sessions and `doc.go` said at the time that a transcription can be wrong
      in exactly the way the prose it replaces could be wrong. It was, in four
      places — `list` requiring `group`, implicit orientations on `radiogroup`
      and `treegrid` that ARIA 1.2 does not state, and `term`/`time` listed as
      name-prohibited — and all four sat in near-miss rows, the entries that
      exist so a guard has something to argue with and are therefore the ones
      least likely to be argued with. `sh aria/fetch.sh` is the one thing here
      that touches the network and nothing on a verification path depends on it:
      the fixture is committed and the test that holds it to the spec skips
      when no download is present
- [x] **A toolbar has a keyboard** — the third composite, and the first whose
      members ARIA does not name. `COMPOSITE_MEMBERS` maps a container role to a
      member role, which is the whole rule for a listbox and a tablist and no
      rule at all for a toolbar; the runtime has a second one now
      (`COMPOSITE_FOCUSABLE`, `focusableMembers`): every focusable control not
      inside a nested composite. `components.ChipStrip` is a `Row` of
      `core.Button`s, so a twelve-chip filter bar was twelve stops in the page's
      tab order where ARIA promises one. A nested composite stops the walk and
      keeps its own stop, which is two stops rather than one and is the honest
      outcome of a rule that will not guess. Checked in a real Chrome
      (`browser.mjs`), where the claim is that the *walk* found the right
      elements rather than that `tabindex` works
- [x] **Selection follows focus** —
      `core.AccessibilitySelectionFollowsFocus()` on a composite container makes
      an arrow, `Home`, `End` or a typeahead match invoke the member's own
      `OnTap`, which is ARIA's recommendation for a tab strip over cheap panels
      and its warning for anything expensive. The standing argument against it
      was that the framework could not make the choice because `aria-selected`
      is rendered from Go state; the premise was the wrong half, since `Enter`
      on a member has reached Go since the day the pattern was written.
      Web-only, unreachable from the patch pass (a selection fired there would
      call into Go, produce a patch, and fire again)
- [x] **`components.CollapseBand`** — the disclosure the default band builds, on
      its own, for a `GroupedList` `Header` override. `Collapse` reaches past an
      override for the row emission and stops at it for the control, which left
      the override author rebuilding a button, an `aria-expanded` and a heading
      wrapper from an argument that lives in an unexported type
- [x] **The three debug guards are measured** — each `IsDebugMode` guard is
      unobservable (`upsertConcern` has its own backstop), so what they buy is
      cost and nothing measured it. `testing.AllocsPerRun` is the assertable
      form: `AuditTree` and `EndRenderPass` allocate exactly zero with debug
      mode off, and `renderAll`'s guard saves exactly the duplicate-key check's
      own allocations
- [x] **The gobind type table reads gobind** — `gobindSwiftTypes` and the two
      refusals beside it are read off `bind/genobjc.go` at the version `go.mod`
      pins, rather than off a header a `gomobile bind` once produced. A returned
      bound interface is no longer refused (`objcParamType` special-cases
      `String` alone, so a protocol is `_Nullable` in both positions), and the
      multi-result refusal now names which of gobind's three arms applies —
      including that three or more results gobind refuses outright, which makes
      it a Go signature to change rather than a row to add
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
- [x] **`wasm/verify/browser.mjs`** — the keyboard facts a shimmed DOM
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
- [x] **The palette on a screenshot** (`wasm/verify/palette.mjs`,
      `wasm/verify/browser.mjs`) — the fifth browser check, and the first that
      is not about the keyboard. `ControlBorder` has WCAG 1.4.11's 3:1 floor
      under it and the census in `components/variant_test.go` measures every
      pair as arithmetic over hex strings — which proves the number and cannot
      prove either colour reaches a screen. Two retints and a third palette
      later, nothing had ever looked. Now one swatch per pair is painted,
      screenshotted, decoded (an eighty-line PNG reader on `node:zlib`, because
      `run.sh` promises Go and Node and nothing else) and compared **as a hex**
      — the ratio travels with the table from Go, since a second WCAG
      implementation is what a contrast floor least survives. The table is
      pinned to `core.BundledThemes()` in both directions
- [x] **The boundary census derives its backdrops** (`internal/palette`) — the
      list of fills a control can be drawn on was five names and one exclusion,
      and the set is a consequence of `core.ComponentDefaults` rather than a
      decision: a new field carrying a `Background` was a pair nothing would
      measure. Reflected over the struct now, with `Camera` and `Button` kept
      out as data carrying their arguments. Deriving it found two pairs the
      hand-written list had never measured
- [x] **The gomobile stub's header comment is checked**
      (`ios/verify/gomobile_stub.swift`) — the declarations were pinned
      character-for-character and the comment describing the rules that pinned
      them was prose. The load-bearing half is rows in a delimited block now,
      and `gomobilestub_test.go` holds each to what it describes: the version
      to `go.mod`, the names to the checker's own, each type row to
      `swiftType`, each result row to what `swiftResult` does with that shape
- [x] **A band's insets are its tap target** (`components.bandInsets`,
      `GroupHeader.ControlStyle`, `CollapseBand.ControlStyle`) — the chrome was
      padding on the row that held the button, so a press in the 16px before
      the chevron did nothing. It is padding on the control now: identical
      pixels, a different node. The row keeps its fill, its centering,
      `StickyHeader` and the badge's own trailing inset, which is the one past
      the control's edge
- [x] **A shut trailing group withholds `OnEndReached`** (`components.GroupedList`)
      — an append pager extends the last run and a shut run emits nothing, so a
      page fetched into a collapsed bottom group landed nowhere and closed the
      guard behind it: one page spent, every fire after it refused, the feed
      reading as exhausted. Withheld while that group is shut, restored when it
      opens; the `Footer` stays as the deliberate way to ask
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
- [x] `core.Switch` — the instant-effect boolean on all four targets
      (Material's `Switch`, SwiftUI's `Toggle`, and an
      `<input type="checkbox" switch role="switch">` on the web), beside the
      `Checkbox` that means the other thing: a value something else will act on
      rather than a setting the tap has already changed. It is a node type and
      not a flag, because a changed type is a *replace* and a replace is how one
      platform control is exchanged for another. Three things it did not need:
      a prop of its own (the state crosses as `checked`, so both web renderers'
      existing create-and-patch handling works untouched), a theme field (it
      reads `Components.CheckBox`; a control a platform draws spends no palette
      entry), and a `core.Role` — `switch` is written from the node type, making
      `htmlout.ownRoles` a table where it had been a comparison against
      `"Modal"`, and joining `dialog` in `aria/spec.NearMisses` as a role this
      framework emits and does not name
- [x] `core.MapView` — a live map on all four targets: MapKit on iOS,
      osmdroid over OpenStreetMap on Android (no key, no Play Services
      dependency), Leaflet on the web, and a placeholder that keeps its region
      in `data-lat`/`data-lng`/`data-zoom` in the static export. Markers are
      **keyed child nodes** (`core.Marker`), the TextGrid trick, so a marker that
      moved is one patch rather than a rebuilt annotation layer — which on every
      platform is a visible flicker and on two of them loses the open callout.
      Three decisions carry it: the region is **applied only when it changes**
      (Go moving the map is an instruction, Go re-rendering is not — without it
      any unrelated render snaps the map out from under a finger), the same
      comparison read the other way is what keeps a host's own recentring from
      arriving back as a gesture, and every host throttles its region stream on
      its own side because a pan is a frame-by-frame event and each one would be
      a full render pass. The web half is checked against a fake Leaflet
      (`wasm/verify/leaflet.mjs`), which is what found the first version of that
      guard: a flag set around `setView` only closes the synchronous case, and
      Leaflet promises nothing about when `moveend` arrives
- [x] **The location fix** (`core/location.go`, `hooks/location.go`,
      `ios/GrMob/App/LocationSensor.swift`,
      `android/.../app/LocationSensor.kt`) — the second sensor, on the
      arrangement Tier C's compass established: one `"sensor"` system event with
      a `kind`, a refcounted start/stop, a record in between, and `Received`
      beside `Available` so "no fix yet" and "this device will never tell you"
      are different screens. Two things the compass did not need: `Accuracy` in
      metres, which is the field that separates a 5-metre GPS fix from a
      2000-metre guess that looks identical to any code reading only the
      coordinates, and `core.DistanceMeters` — a haversine, because the flat
      approximation breaks at the antimeridian and the notification filter is a
      consumer. The two shells differ on who prompts, which is the platform's
      difference rather than this framework's: `CLLocationManager` can ask from
      anywhere and there is no fix without it, Android needs the Activity that
      `Permissions.kt` holds
- [x] **Both native renderers dispatch every node type**
      (`mobile/verify/nodetypes_test.go`) — the census the natives were missing.
      htmlout's tag table is documented as one and the WASM runtime's copy is
      pinned to it, so three of the four targets already failed when a node type
      was forgotten. The natives' dispatch ends in a catch-all, which makes a
      missing arm silent by construction: a childless control with no arm draws
      as an empty box on the phone and correctly on both web targets. Written
      the session `core.Switch` landed, which is the session it would have
      caught three arms in had it existed first
- [x] `core.TextGrid` — a monospace grid of styled runs on all four targets,
      rows as children so a terminal diff patches one row, not the grid
- [x] **The overlay `Layout`'s three decisions are run off-device**
      (`ios/GrMob/Runtime/GrMobStack.swift`, `ios/verify/stack.swift`) — the
      arithmetic was already checkable; what stayed inside the SwiftUI `Layout`
      was which proposal each layer is measured with, whether the container may
      be clamped to it, and what a layer is offered at placement. None is
      arithmetic, all three are load-bearing, and a type-check was the only
      thing reading them. A `LayoutSubview` is an opaque proxy no test can
      construct, but the two things the layout asks one are a two-method
      protocol — so the decisions moved next door and now run against a fake
      that records every offer. What still needs a simulator is SwiftUI's own
      behaviour, not this framework's
- [x] **`core.ValueRange.Progress` — the numeric reading has an authority**
      (`core/value.go`, `internal/valuefixture`,
      `android/app/src/main/java/com/grmob/runtime/GrMobProgress.kt`) — the
      three-way branch on a range's numbers existed only in Kotlin, inside an
      extension on `SemanticsPropertyReceiver`, and was checked by searching
      the file for the word `Indeterminate`. The rules are ARIA's and were
      already written in prose on `core.ValueRange`'s fields; naming them makes
      them executable. Four readings, not three — an empty range is a claim
      Compose cannot hold and has to be told apart from "nothing was stated"
      before the property is assigned — and `android/verify` now runs a
      second import-free Kotlin file against Go over a shared table
- [x] **A `Spacer`'s own `Style` outranks its size prop**
      (`wasm/grmob-runtime.js`, `htmlout/export.go`) — the three declarations
      were written after the style pass on one DOM target and instead of it on
      the other, which made a `Spacer` the one node type where a type default
      beat an author. Both now state the chassis underneath the author's style,
      as `modalChassis` already did; the runtime records which of the three the
      author claimed rather than reading the live property back, since a size
      change arrives with no `Style` beside it. Moving `htmlout`'s out of an
      early return gave a `Spacer` back its accessibility attributes, its
      callback IDs, its children and its own style — four things the other DOM
      renderer had been giving the same node all along
- [x] **A picker heading with nothing under it, answered rather than left
      open** (`core/input.go`, `core/select_menu.go`) — `Group` is a field on
      an option, so an empty section is unwritable, and that is now stated
      where a caller reads it, pinned as a property of `SelectMenuSections`,
      and answered: a disabled placeholder row says *why* the category is
      empty, which an empty `<optgroup>` cannot. The shape is in
      `internal/menufixture`, so all four picker menus are held to it
- [x] **The browser pass asks its first layout question**
      (`wasm/verify/browser.mjs`) — five checks in, everything it asked could
      have been asked of a page with no geometry. `core.StickyHeader()`'s three
      declarations are exactly the kind a shimmed DOM can only restate, and the
      box around them is what defeats a pin in practice. The band is scrolled
      and asked twice: through the rects the browser reports, and through the
      pixels at a point an unpinned band would have left a row behind
- [x] **The wrapper test is asked about the slice callers really get**
      (`components/rows_spec_test.go`) — `appendRows` appends into the
      container's own argument list and returns it, so the admission rule was
      being answered against a fixture that held children alone. It now takes
      the prop prefix both widgets pass, and the first leg is that a core prop
      is a closure too: `reflect.Kind` cannot tell one from a keyed child, so
      the only sound filter is the `core.View` assertion, checked in both
      directions before the children are looked at
- [x] **The witness census is read down its columns as well as across its
      rows** (`components/palette_witness_test.go`) — a row now states what its
      witnesses amount to (a bundled theme, the fixture alone, or nothing) and
      the test derives the same value and compares, which caught the file's own
      prose already claiming two fixture-only rules that `AmberTheme` had
      witnessed for a release. A new palette adds a column, and its two
      extremes — witnessing nothing, witnessing everything — look alike in a
      row diff and mean opposite things, so each costs a named entry with a
      reason, the road `internal/palette`'s backdrop exclusions took
- [x] **The tutorial's theme chapter offers every bundled palette**
      (`examples/tutorial/chapter7.go`) — both lessons wrote their list out by
      hand, so `AmberTheme` shipped and the chapter whose whole subject is
      theming went on offering two. A hand-written list in a tutorial is worse
      than one in a test: the gap is not a missed assertion, it is a palette
      the reader is never told exists
- [x] **A `Spacer`'s `Style` is honoured on all four targets** — its size
      arrives as a *prop*, which is what made it the one node type on both
      natives whose `Style` went missing whole: each arm was one expression
      built from the prop, with no call to `grMobBox` or `boxModifier` anywhere
      in it, so a hand-assembled Spacer's `Background`, `Margin`,
      `AccessibilityLabel` and `OnTap` were dropped on a phone and honoured in
      a browser. The chassis now goes underneath the author's declarations on
      both, per axis, by opposite mechanisms — SwiftUI's outer frame wins so
      the chassis is written inside it and omitted on a claimed axis; Compose's
      constraints flow outside-in so `boxModifier` first is the whole of it
- [x] **The overlay `Layout`'s adapter is executed rather than compiled**
      (`ios/GrMob/Runtime/GrMobStackBridge.swift`) — converting between
      `ProposedViewSize` and `GrMobProposal` was three field-copying
      expressions in `Renderer.swift`, "three lines with no decision in them",
      which was true and still left them on the half of the target that nothing
      runs. A swapped axis compiles and draws. The obstacle was never SwiftUI —
      it was `LayoutSubview` in particular, and a `ProposedViewSize` is an
      ordinary struct `ios/verify` can construct
- [x] **A stale ARIA download is named as one** — the spec is not committed and
      W3C keeps every revision at its own URL forever, so a 1.1 copy parses
      cleanly to ~94 roles and regenerates a fixture differing on exactly the
      four facts 1.2 changed. Left to the diff that reads as a broken fixture.
      `spec.Version` and the document's own title heading make it one line
      about the fetch instead; `aria/fetch.sh`'s URL is held to the constant
- [x] **`components.CollapseBand` is built by an example** — its only readers
      were its own tests, and `ControlStyle`'s whole justification is how a
      *real* custom band is assembled. Lesson 4.6 assembles one, with every
      band starting shut so the half a `Header` override does not own — the
      widget withholding the run — is visible rather than described
- [x] **A keyboard contract on a widget with no keyboard is reported**
      (`inert-follows-focus`) — the WASM runtime writes
      `data-grmob-selection-follows-focus` for any node that asks, deliberately,
      since consulting the composite tables where attributes are written would
      put them in two places. `core.KeyboardComposites()` is that list in Go,
      held to the runtime's two tables by `wasm/verify`, and `core.AuditTree`
      is where the claim-about-nothing is reported
- [x] **A composite inside a composite is reported** (`nested-composite`) —
      both keep their own roving `tabindex`, so the pair is two tab stops where
      ARIA describes one. It was stated in three comments and a documentation
      section, all of them in the WASM target, where an author writing Go does
      not read. ARIA's version needs two widgets writing `tabindex` onto one
      element and an owner rule for when they disagree, which this framework
      does not have — so the divergence stands and the author is told
- [x] **A hand-assembled `Spacer`'s children reach all four targets** — a
      Compose `Spacer` and a SwiftUI `Color.clear` are leaves, where both DOM
      renderers emit them like any other element's, so a subtree rendered in a
      browser and vanished on a phone. It was left open as "unreachable from
      Go", which was true of `core.Spacer(n)` and not of `*core.Node`; the DOM
      side is also the side that cannot move, because the runtime addresses
      patches by walking `node.Children`. Both natives stack them now, and the
      axis is `htmlout`'s `stackAxes` rather than each renderer's guess — a
      `Spacer` with children is a `Box` with a fixed size, and the row costs a
      childless one nothing
- [x] **The nested-composite finding says which of two things the outer arrows
      do** — it reported "steps over the inner one whole" for all nine ordered
      pairs, which is right for four. `compositeMembers` deliberately descends
      through a composite of the other kind ("an option below a tablist is
      still the listbox's option"), so for the other five the outer widget's
      arrows can land *inside* the nested one. `core.CompositeWalkStopsAt` is
      the rule, read by the audit's sentence and pinned to the runtime's two
      walks; `keynav_test.mjs` runs a descending pair in a real DOM
- [x] **The tappable-container census has an authority** — every `core.Role` is
      decided against it and the reasons were prose top to bottom, with a
      comment arguing that a reason string cannot be checked. Four of the seven
      kinds are derivable — ARIA's Required Owned Elements,
      `core.KeyboardComposites()`, `core.CompositeMemberRole()` and the
      attribute list that gives a role a value range — so the table moved to
      `aria/verify`, where the fixture is, and a role filed under the wrong
      kind now fails instead of reading perfectly
- [x] **`swiftTypeBody` cuts a Swift type where its braces close** — it ended a
      declaration at the first `}` in column one, which is a claim about how
      this repository indents rather than about Swift, and a short cut still
      returns a string: every `strings.Contains` below it would have passed by
      reading nothing. It uses the comment- and literal-skipping brace scanner
      the Kotlin dispatch parse already had, which grew a `"""` arm for the
      case that has no formatting fix
- [x] **The stale-download skip is a branch a test can reach** — `aria/verify`
      turns a 1.1 copy into a SKIP rather than a fixture diff, and that guard
      had never executed on any machine: its subject (`spec.Parse` refusing
      1.1) was covered, but "turn that failure into a skip" is a different
      claim. Extracted as `localCopyGate`, whose three answers are now
      exercised directly — and exercising it turned up a fourth: a truncated
      download or an error page has no heading at all, and was being reported
      as being of edition `""`
- [x] **`CONTROL_ROLES` is a property rather than a list**
      (`core.TappableContainerRoles()`) — the two roles that make a plain
      container a control were three copies of one fact, and a *new* role would
      have left all three untouched: a future `RoleCheckbox` is one by exactly
      the argument `core.Role` makes for `RoleButton`, and would have shipped
      as a role a toolbar steps over. `core/role_control_test.go` holds every
      role in the vocabulary to one side of the question, with a reason
- [x] **The refusals table's shape is half derived** — `Blocked` was computed
      from `core.Roles()` and checked; `Shape` sat beside it looking the same
      and was a sentence nothing could contradict. ARIA's Required Owned
      Elements row is the authority for one piece of it: whether a pattern's
      members are the container's own (`listbox` owns `option`) or somebody
      else's (`grid` owns `row`, and a `gridcell` is a row's), which is exactly
      the difference between a walk this runtime does and the one it lacks.
      That piece is `Nesting`, declared and derived; the rest is labelled prose
- [x] **`aria/gen` runs on a verification path** — the command's own half (the
      walk to the module root, the two paths, the write, and the error a
      missing download produces) was reached by nothing. `aria/gen/main_test.go`
      runs `run` against a synthetic specification in a temp module, from three
      working directories, and holds what it writes to `spec.Scope` and
      `Fixture.Render`
- [x] **The palette census's two degenerate columns are exercised** — the
      `quietThemes`/`universalThemes` tables are empty and have been since they
      were written, so five of the six arms that read them had never run. The
      classification is a function now (`landingComplaints`), driven over
      constructed theme sets, and the real tables stay empty
- [x] **The browser's sticky fixture is pinned to `core.StickyHeader()`** —
      `STICKY_DECLARATIONS` in `browser.mjs`, held to the *set of fields* the
      prop touches rather than to three names typed on each side
- [x] **`core.ValueRange.Progress` has a Go consumer** — `core.AuditTree`
      reports `unusable-value-range`: a stated position or bound that is not a
      number, or a range whose `Max` is at or below its `Min`. Every target
      resolves those and no two of them the same way, while the bar on screen
      goes on drawing the caller's own float. `ValueRange.Unparsed` is the new
      half the reading alone could not give
- [x] **`internal/valuefixture` is compared on the web too** — `browser.mjs`
      mounts one `progressbar` per case in a real Chrome and reads the answer
      out of the browser's accessibility tree. Cases whose numbers all parse
      must agree with `core.Progress`; cases with an unparseable field must
      *not*, which pins a real divergence (Chrome reads `aria-valuemax="lots"`
      as 0 and clamps a bar at 45% into announcing as complete)
- [x] **A `GroupedList` can be asked whether its edge was withheld** —
      `AutoLoadWithheld()`. The withholding is right and silent, and a feed
      that stopped fetching looks exactly like one that ran out; a screen whose
      footer is conditional had no way to tell
- [x] **The browser pass paints a real widget, not only swatches** — `gen.go`
      renders a quiet `components.Chip` per bundled theme and reads its three
      colours off the rendered node; `widget_test.go` holds the ring to
      `Colors.ControlBorderColor()` and both backdrops to `internal/palette`'s
      derived list
- [x] **The backdrop exclusions live on the field** — a `notbackdrop` struct
      tag on `core.ComponentDefaults`, whose value is the argument.
      `palette.NotABackdrop()` is the reading of it, so an entry can no longer
      name a field that does not exist or drift from the one it names
- [x] **gobind's three result arms are read off the pinned generator** —
      `TestTheResultArmsAreReadOffThePinnedGobind` holds `swiftResult`'s
      refusals to `bind/genobjc.go` in the module cache. Reading it through
      also corrected them: every bound symbol here is a package-level func,
      emitted as a C function, and Clang's `throws` convention is the
      Objective-C *method* one
- [x] **A `Header` override is told what the widget knows** — `Group` carries
      `Trailing` (this is the run an append pager extends) and
      `AutoLoadWithheld` (this run being shut is why the list has no edge
      sensor). The first is what makes `HideTrailingCount`'s rule
      implementable in an override at all; both are stamped before anything
      reads the `Group`, so the predicate, the band and `OnToggle` see one
      shape
- [x] **Every `ComponentDefaults` field says whether a control is drawn on it**
      — a `backdrop` tag beside the `notbackdrop` one, exclusive and total.
      With only the exclusion, everything else was measured because it was left
      over, and a `notbackdrop` tag could be *deleted* with no consequence but
      a pair quietly joining the census. `palette.Untagged()` now refuses that
      state, and the reachability claim travels into the census's failure
      message. Classifying `Text` — a leaf nothing can be nested inside, whose
      component default nothing even reads — removed two pairs the census had
      been measuring against nothing
- [x] **The browser paints an `Input` frame as well as a chip ring** — the
      tone's two spenders read it from two different Go values, so each case
      names its own authority (`widgetCase.RingFrom`). The field is also a
      second tag with a second user-agent rule, and both horizontal edges are
      scanned: one edge says the tone survived, the pair says the box was
      closed
- [x] **gobind's result arms are transcribed, from a real bind** — a package
      with one function and one interface method of every result shape was
      bound with `gomobile bind -target=ios` and read back through `swiftc`.
      No package function ever throws (they are C functions); an interface
      method does, except when its return is `NSString* _Nonnull`, which gives
      Clang's error convention nothing to signal with. The bind also found a
      hole: `error` was bindable and in neither type table, so a bridge
      function that grew one would have gone undeclared and taken the app
      layer's type-check with it
- [x] **The band's insets are asked of SwiftUI** — `internal/bandfixture`
      reads the real `GroupHeader`'s geometry and `ios/verify/band.swift`
      solves both arrangements through `GrMobFlexSolver`. They place the same
      pixels at every offer with slack and at an indefinite proposal, and
      diverge in two recorded places: under overflow (shrink is proportional
      to a base that includes the child's own padding) and for a badge taller
      than the control, which no real band has
- [x] **The band's insets are asked of a browser too, and the divergence is
      real** — `internal/bandfixture` rides in `wasm/verify`'s transcript and
      `browser.mjs` lays out both arrangements at every offer in a real Chrome.
      They agree everywhere, overflow included: CSS distributes shrink over the
      *inner* flex base size, where `GrMobFlexSolver`'s base includes the
      child's own padding. So the `ios/verify` difference is a cross-target
      divergence rather than an artefact, and both targets assert their own
      answer. The overflow arm carries its own control — a flex item's automatic
      minimum size would otherwise leave both arrangements at their natural
      width, agreeing by never reaching the arithmetic
- [x] **The value-range verdict is reachable from a unit test** — half of it
      could never fail on a machine with a shipping browser: the
      `parses: false` rows are a *pinned divergence*, reported only if a
      browser starts applying ARIA's defaults to a value that is not a number.
      `valueRangeProblem(row, ax)` moved beside the table in `valuerange.mjs`,
      so `valuerange_test.mjs` hands it all four answers — including the one no
      browser gives
- [x] **`browser.mjs`'s two preconditions state their two stances once** —
      `startup.mjs`. A missing Chrome is a fact about the machine (SKIP); a
      missing `GRMOB_TRANSCRIPT` is a fact about the invocation (FAIL), decided
      first, because the other order turns a forgotten variable into a green run
      on a machine with no Chrome. Inline guards could only be reached by
      arranging a machine that had the fault; this one takes three booleans and
      a string
- [x] **A widget's boundary *provenance* has a pixel** —
      `TestEachWidgetReadsTheAuthorityItNames` renders `gen.go`'s own widget
      builders through a throwaway theme whose `Colors.ControlBorder` and
      `Components.Input.BorderColor` are deliberately different hexes. In every
      bundled theme the two hold the same value, so a chip that had started
      reading the field base — or a field that had started reading the role —
      passed every pixel and every hex comparison in the repository
- [x] **The widget swatches are one page and one screenshot** — six mounts,
      six screenshots and six PNG decodes became one of each, laid out as a
      grid. The trees go in unmodified; all that changes is that a theme's page
      fill is a sibling's rather than the document's, which no sample was ever
      reading
- [x] **The disclosure band's tap target spans the band, measured** — the
      collapsible branch puts the insets on a button one level inside the Row's
      growing heading wrapper, so whether a press lands on the whole band is a
      *cross-axis* question and `GrMobFlexSolver` is a main-axis distributor.
      `gen.go` renders real `components.GroupHeader`s through every bundled
      theme and `browser.mjs` measures the rects: the button does fill the
      wrapper, and it does so by a cross-axis default rather than by anything
      the widget declares. The picture in `components.bandInsets` had been
      assuming it
- [x] **The two band branches are the same chrome and not the same height** —
      measured against pixels, `GroupHeader.ControlStyle`'s "a control and not a
      relayout" is exact for the leading and trailing edges and off by a point
      for the height, in every bundled theme: the disclosure's button holds a
      chevron the plain band does not, and that glyph's line box exceeds the
      caption's. Checked as the equation it is — the difference must equal the
      chevron's overhang over the words — so chrome drifting between the
      branches still fails
- [x] **A bold caption is no shorter than a plain one, checked** — every
      `SameHeight` case in `internal/bandfixture` rests on the padded control
      being the band's tallest child, and half the reason was a claim about
      glyphs that no Go test can take. The rendered bands measure it, in three
      themes, with real text
- [x] **What a fixed-size container does with an oversized child, on four
      targets** — nothing anywhere had asked. A browser's answer turns out to be
      per-axis rather than the blanket "spills" that was assumed: the child is
      squeezed along the container's main axis (a flex item's shrink factor
      defaults to 1, and an empty box has no automatic minimum to stop at) and
      spills across the cross one. `htmlout` emits the same declarations and
      inherits the answer; the two natives are derived from the platform call
      each renderer makes and pinned at those call sites. Compose is the odd one
      out — `Modifier.width`/`height` set the child's minimum *and* maximum, so
      it squeezes on both axes
- [x] **Why the band's cross-target census has three rows and not four** — the
      web and the SwiftUI solver are executable because the arithmetic is ours
      or the browser is a browser; Compose's `Row` is androidx's code and needs
      the Android runtime to measure anything, and its sources are not cached at
      the version this build pins. The derived answer (Compose agrees with the
      web, for a third reason: no proportional shrink at all) is recorded, and
      what is *tested* is the premise — that Android still delegates its
      distribution, so a renderer that stopped would put the answer back within
      reach
- [x] **`core.ComponentDefaults.Text` is gone, and the next inert default
      fails** — `core.Text` builds its Style from its own props and never
      touched the theme, so two bundled themes described a widget that does not
      exist and a theme author filling it in saw nothing happen. A run of words
      already has an authority (`Typography`), so the field was removed rather
      than wired. `TestEveryComponentDefaultReachesAWidget` renders a witness per
      field through a theme carrying a marker no widget sets for itself: a field
      nothing merges now has nowhere for the marker to arrive
- [x] **`bindableGoTypes` is total over what gobind carries** — `error` was left
      out of it on the grounds that no bridge function returns one, and the
      omission was invisible: gobind binds it, so such a function would have
      produced a symbol, gone undeclared in the stub, and taken
      `GomobileBridge.swift`'s type-check with it. The carried set is now derived
      from gobind's own `isSupported` in the pinned module cache and every member
      must be classified — spelled, or refused with a reason — so a bridge
      function using an unspelled one fails by name instead of dropping out of
      the check
- [x] **`core.FlexShrink(0)` means something now** — every optional number in a
      `core.Style` means "unset" by being zero, and flex-shrink is the one whose
      CSS initial value is not zero, so `Style.Merge`, `htmlout.Export` and the
      WASM runtime each discarded "do not shrink" as nothing having been said.
      The prop compiled, applied, serialised and did nothing, and it was found
      by a break-test that could not break. `core.ShrinkNone` is the sentinel
      (-1, which CSS forbids, so no author can produce one), `Style.ShrinkFactor`
      is the single reading, and it is honoured by both DOM targets and by
      `GrMobFlexSolver` — whose shrink arm is now CSS's scaled-base rule rather
      than the one factor the Go side used to be able to express. Compose has no
      proportional shrink to honour it with
- [x] **The sticky fixture's shrink factor, and what it was really resting on**
      — the same inert `FlexShrink: 0`, in a mounted tree. It works now, and the
      List measures 400px in a 160px port either way: a flex item's automatic
      minimum size is content-based and those rows carry text, so the
      declaration never was the reason and could not be. The check asserts the
      arrangement itself instead of one of the mechanisms that could produce it
- [x] **Whether any real screen can produce a nested composite** — the five
      descending pairs `core.CompositeWalkStopsAt` sorts had an example nobody
      called realistic. Nothing in `components` declares a composite *container*
      role: `RoleOption` and `RoleTab` are member roles, and every listbox,
      tablist and toolbar is a container the caller built and roled themselves.
      So the pair needs two deliberate declarations by one author — reachable,
      and not something composition falls into — and a widget that started
      declaring one now fails the test that says so
- [x] **`swiftTypeBody`'s anchor is syntactic at both ends** — the cut was, and
      finding the declaration was `strings.Index`. Every declaration in these
      renderers carries a doc comment and several name their neighbours, so an
      anchor could match a mention and hand every check a paragraph of English
      to search. The anchor must now match at the start of a line, in code, and
      two matches are refused rather than resolved silently
- [x] **`core.CompositeMemberRole` no longer returns "" for two reasons** — a
      toolbar has a keyboard and no member role ARIA names; a `RoleHeading` has
      neither, and both got the same empty answer. The doc said callers separate
      them by asking `KeyboardComposites` first, which a doc cannot enforce and
      which the one caller inside core got right by never being handed a
      non-composite. It is comma-ok now, and the flag is held to
      `KeyboardComposites()` in both directions
- [x] **The two shell gates are functions with their own tests** —
      `ios/verify` skipping on a missing iPhoneOS SDK and `android/verify` on a
      missing Kotlin compiler were inline conditions whose arms needed a machine
      with the fault. Extracting the Android one found the order was wrong:
      `kotlinc` is a JVM application, so a machine with a compiler and no JDK
      ran it and failed under `set -e` instead of skipping
- [x] **`core.FlexShrink(0)` means something on Compose too** — the argument
      for leaving Android out was real and covered the wrong half. A Compose
      `Row` has no proportional shrink at all: an unweighted child is measured
      against the main-axis space the ones before it did not take, so there is
      nothing for a *fractional* factor to scale. Zero is not a proportion but
      a refusal, and a refusal is expressible — `Modifier.pinMainAxis` measures
      the child unbounded and reports the size it measured, so the child keeps
      its extent and the row overflows around it, which is what the other three
      targets do. The reporting half is the one that fails silently: clamping
      the size on the way out compiles, looks better behaved, and draws the
      child spilling out of a box its parent still believes it fits inside
- [x] **`CompositeWalkStopsAt`'s two stopping arms are two values now** — a
      container with no keyboard and a toolbar whose walk really does stop both
      returned `true`, so the distinction was written into the code with a
      paragraph on each arm and thrown away in the return. `core.CompositeWalk`
      is the three answers — not applicable, stops, descends — the bool is a
      reading of it rather than a second implementation, and the audit's finding
      is built from the value so the one case it must not describe cannot be
      described by accident
- [x] **The fixed-size census's SwiftUI main axis is measured** — its cross axis
      is a SwiftUI fact and stays a reading of the call site, but the main-axis
      squeeze is `GrMobFlexSolver`'s, which is ours and which `ios/verify`
      executes. It now runs the census's own box, on `browser.mjs`'s numbers,
      with the two harnesses pinned to one fixture: two passes agreeing about
      different boxes is a weaker statement than the census makes and neither
      pass could tell
- [x] **The census's Compose row is read from the version this build resolves**
      — it had been read from whichever `foundation-layout` a gradle cache
      happened to hold, two minor versions off what the BOM pins. The version is
      derived now (the BOM's own pom, out of the cache, on any machine that has
      ever built the app), a `composeLayoutSources` configuration fetches the
      matching sources once, and the two claims — that `Modifier.width` sets a
      maximum, and that the zero-weight branch offers a child what is left — are
      read out of `Size.kt` and `RowColumnMeasurementHelper.kt` themselves.
      Machines that have never fetched skip that half with the command in the
      message

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

