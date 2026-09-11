# Next components: DataTable, a widget bundle, Compass, Map

**Status:** A1 landed 2026-09-04 (`components/grouping.go`, `paging.go`,
`grouped_list.go`, `data_table.go`, tutorial lesson 4.6), and the church
sermons screen has adopted it. A2 landed 2026-09-04 as all seven widgets —
`app_bar.go`, `banner.go`, `empty_state.go`, `search_field.go`,
`chip_strip.go`, `skeleton.go`, `stat_tile.go`, tutorial lesson 4.7 — plus
`hooks/debounce.go`, which the plan's "debounced OnChange via UseTimeout" line
turned out to require: `UseTimeout` arms once per mount and stays fired, so it
cannot debounce. B1–B4 landed 2026-09-05 across the four renderers and were
adopted downstream the same day. A3 landed 2026-09-05 as `components/calendar.go`
and `components/date_picker.go` plus tutorial lesson 4.9. Tier C landed
2026-09-05 entire — `core/heading.go`, `hooks/heading.go`, `permission.Location`,
`components/compass.go`, and the three host sensors
(`android/.../app/HeadingSensor.kt`, `ios/GrMob/App/HeadingSensor.swift`, the
browser's `deviceorientationabsolute`/`webkitCompassHeading` arm in
`wasm/grmob-runtime.js`), plus tutorial lesson 4.10. Both follow-ups the tiers
above left hanging closed with it: the A1 accessibility role prop is
`core.AccessibilityRole` (`core/role.go`), and the `EmphasisGhost` border the A3
note found is reset by `borderResetTypes` (`htmlout/tag.go`, restated as
`BORDER_RESET_TYPES` in the runtime).

**B5, D0 and D1 landed 2026-09-11, which closes the plan.** B5 is `core.Switch`
across the four renderers (tutorial lesson 2.6); D0 is `components.StaticMap`
(lesson 4.11); D1 is `core.MapView` with `core.Marker` children plus the whole
location fix — `core/location.go`, `hooks/location.go`, `LocationSensor` on both
shells — on MapKit, osmdroid and Leaflet (lesson 4.12). What each of the three
turned out to need beyond its sketch is recorded at the tier. The one gap worth
naming here: **the Compose half of D1 is compiled by nothing in this repository.**
`android/verify` builds only the two Kotlin files that import nothing, and
`GrMobMapView.kt`/`LocationSensor.kt` import Compose, the Android SDK and
osmdroid — so they are held to the contract textually (`mobile/verify/
mapview_test.go`, `sensor_test.go`) and have never been through `kotlinc`. The
iOS half type-checks against the real SDK through `ios/verify`, and the web half
runs against a fake Leaflet in `wasm/verify/mapview_test.mjs`.
**Date:** 2026-09-04
**Driver:** `../church/church_mobile` (sermons list wants grouping + paging; events want
a "where" affordance), plus general widget-library gaps.

## What exists today (constraints the plan works within)

- `components` is a pure-Go layer: struct widgets implementing `core.View`, themed from
  `ctx.Theme()`, no renderer changes. Anything built here works on all four targets
  (DOM, htmlout, Compose, SwiftUI) for free.
- `core.List` is the virtualized column (LazyColumn / LazyVStack). Rows must be `Keyed`.
- `core.Scroll` is vertical only. No scroll offset or end-reached signal reaches Go, so
  the church app pages with an explicit "Load more" tail (`app/ui.go: pagedList`).
- Host events (`core.OnHostEvent` / `mobile.ReportHostEvent` / `GrMobWASM.HostEvent`)
  and system events (`core.SendSystemEvent`) are the sensor channel; audio and lifecycle
  use it, and the comment in `core/host_events.go` already reserves "a location fix".
- `permission/` covers camera only. No `Rotate`/transform style prop exists.
- A new **node type** costs four renderers: `htmlout/export.go`, `wasm/grmob-runtime.js`,
  `android/.../Renderer.kt`, `ios/GrMob/Runtime/Renderer.swift` (Slider is the template).

The tiers below are ordered by value per renderer touched: Tier A touches none.

---

## Tier A — pure-Go widgets (no renderer work)

### A1. Paged, grouped collections: `GroupedList[T]` and `DataTable[T]`

One engine, two facades. Both are generic structs; a generic struct satisfying `core.View`
is fine in Go and keeps rows typed at the call site.

**Shared engine (`components/paging.go`, `components/grouping.go`):**

```go
// Pagination is the numbered-page footer (client-side or server-side pages).
type Pagination struct {
    Page, PageCount int      // 0-based page, total pages (0 = unknown → prev/next only)
    OnChange        func(page int)
}

// LoadMore is the append-style tail the church app hand-rolls in pagerFooter.
type LoadMore struct {
    HasMore, Loading bool
    Err              error
    OnLoadMore, OnRetry func()
}

// Group is what GroupBy yields; the header renders once per run of equal keys.
type Group struct{ Key, Label string; Count int }
```

`GroupBy` runs over the *display order* and emits a header whenever the key changes
(run-length, not a map), so a list that arrives sorted stays in one pass and an
append-only pager (sermons in date-desc order) can only ever grow the last group —
no header jumps on "Load more".

**`GroupedList[T]`** — the sermon shape:

```go
components.GroupedList[api.Sermon]{
    Items:   pager.Items,
    Key:     func(s api.Sermon) string { return rowKey("sermon", s.ID) },
    GroupBy: func(s api.Sermon) components.Group {
        m := s.DateTaught.Format("2006-01")
        return components.Group{Key: m, Label: s.DateTaught.Format("January 2006")}
    },
    Row:     func(s api.Sermon) core.View { return sermonRow(ctx, s) },
    Header:  nil,                       // optional override of the default group header
    Empty:   emptyNote(ctx, "No sermons found."),
    Footer:  components.LoadMore{HasMore: pager.HasMore, Loading: pager.Loading,
             Err: pager.Err, OnLoadMore: pager.LoadMore},
}
```

Renders `core.List` with keyed rows and `Keyed("group:"+key, header)` headers. Default
header: theme `Surface` band, `TextSecondary` label, count badge. Headers are not sticky
in v1 (see B3).

**`DataTable[T]`** — columns on top of the same engine:

```go
type Column[T any] struct {
    Title  string
    Cell   func(T) core.View        // or Text func(T) string for the simple path
    Weight float64                  // FlexGrow share; 0 = content width
    Align  core.Alignment
    Narrow bool                     // drop this column when the table is in compact mode
    Less   func(a, b T) bool        // enables client-side sort on header tap
}
type DataTable[T any] struct {
    Columns  []Column[T]
    Rows     []T
    Key      func(T) string
    GroupBy  func(T) Group           // optional; group header spans the row
    Sort     Sort; OnSort func(Sort) // controlled: {Column int, Desc bool}
    OnRowTap func(T)
    Selected func(T) bool            // row tint, same convention as ListRow
    Compact  bool                    // phone mode: hides Narrow columns
    Pagination *Pagination           // numbered pages, client-side slicing when PageCount==0
    Footer   core.View               // e.g. LoadMore
    Empty, Loading core.View
}
```

Header row is a `core.Row` of tappable header cells (sort glyph ▲/▼ on the active one);
body is `core.List` of keyed `core.Row`s, cells sized by `Weight`. Client-side paging
slices `Rows` when `Pagination.PageCount == 0` and `OnChange` is nil; otherwise the
caller owns the page. Tests: htmlout snapshots for grouped/paged/sorted/empty states, the
run-length grouping unit test, and a compact-mode column-drop test.

Accessibility note: there is no `AccessibilityRole` prop, so htmlout emits divs, not
`<table>`. Adding a `Role` behavior prop (`table/row/columnheader/cell`, mapped to ARIA
on web and `Modifier.semantics` / `accessibilityAddTraits` natively) is a small core
follow-up worth doing alongside.

### A2. Widgets to lift out of the church app (each is a hand-rolled helper there)

| Widget | Source in church app | Notes |
|---|---|---|
| `AppBar` | `screenHeader` | title, optional back, trailing actions slice |
| `Banner` | `noticeStrip` | text + optional action, `Variant` for error/info |
| `EmptyState` | `emptyNote` / `busyNote` | glyph, title, hint, optional action |
| `SearchField` | — | input with clear button and debounced `OnChange` via `UseTimeout` |
| `ChipStrip` | `chipRow` | wrapping now; horizontal once B1 lands |
| `Skeleton` | — | shimmer-less placeholder blocks (no animation dependency) |
| `StatTile` | social/fintech examples | value + label + delta |

Half a session each; do them as one batch with snapshot tests.

**Landed 2026-09-04.** Departures from the sketch above, each recorded where
it matters:

- `SearchField` is *stateless* and does not debounce itself. A controlled
  field cannot: the value has to reach state on the keystroke or the
  characters do not appear. The debounce moved to the caller as
  `hooks.UseDebounce` (a re-arming `*Debouncer` with `Call`/`Cancel`/`Pending`),
  which is a new hook rather than a flag on `UseTimeout` — the two differ in
  exactly the thing they are about. Keeping the widget hook-free also keeps it
  usable in a header that appears and disappears.
- `Banner` spends its variant on the border and glyph, not on the fill. A
  saturated role color across the width of a screen is too loud, and the
  palette has no muted container tone; the side effect is that a banner's
  contrast is the same whatever it is saying.
- `EmptyState` absorbed `busyNote` and `errorRetry` as well as `emptyNote` —
  three states, one shape.
- `StatTile` has no frame (the card is the caller's) and its delta's zero
  variant is neutral rather than Primary, because whether a number going up is
  good is the caller's domain.
- `ChipStrip` takes `[]Chip` rather than a parallel labels/selection
  vocabulary, and ships no `Scrollable` field waiting on B1.

### A3. `Calendar` / `DatePicker` (pure Go, medium)

Month grid built from `time`: 7-column header, 6 `core.Row`s of day cells, controlled
`Selected time.Time` + `OnSelect`, `Min/Max`, `Marked func(time.Time) bool` for event
dots. The events screen wants this; a range variant is a later field, not a new widget.

**Landed 2026-09-05.** The sketch survived; four things it did not say turned
out to carry the design:

- **The grid is always six rows**, padded with the adjacent months' days drawn
  dimmed and *inert*. A grid that sized itself to its month changes height
  between February and August and shoves the screen below it about on every
  arrow tap; the fixed shape also makes a month change a pure prop patch over
  42 cells rather than an add/remove of whole rows. The adjacent days are
  inert because a controlled calendar cannot move its own month — a tap on
  one would either select a day the grid no longer highlights or fire two
  callbacks in an order the caller has to guess.
- **Cells are built at midday, and midday is what `OnSelect` hands back.**
  Midnight is a local time that does not exist on every calendar day: Chile
  springs forward at 24:00, so `time.Date(2026, 9, 6, 0,0,0,0, Santiago)`
  resolves to *2026-09-05 23:00*, and a midnight grid emits two cells reading
  as the 5th while the 6th becomes unselectable — in exactly the zones nobody
  testing in UTC ever looks at. `calendar_test.go` pins the case.
- **`Today` is a field, not a `time.Now()`.** A render that reads the clock is
  not a function of its inputs (the snapshot drifts every midnight), and
  "today" is a time-zone question the widget cannot answer and the caller can.
  There is no `time.Now()` anywhere in `components`, `core` or `hooks`, and
  this widget was the first one with a reason to want one.
- **`DatePicker` is the packaging, not a second calendar.** It owns exactly the
  two states no application wants — is the sheet open, which month is being
  browsed — which makes it the package's second stateful widget after
  Accordion, and it takes a `Calendar` as a *template* the way
  `SegmentedControl` takes a `Chip`. It carries no label, hint or error,
  because `FormField` already owns all three and any input drops into its slot.
  The browsed month is held as a zero `time.Time` rather than seeded from the
  selection, since zero is exactly what `Calendar`'s anchor fallback reads as
  "follow Selected, then Today" — so a selection set from elsewhere between two
  openings is followed instead of going stale.

Found while building it, not fixed here: a `<button>` keeps the user agent's
default border in both DOM renderers (`htmlout/export.go` and
`wasm/grmob-runtime.js` emit `border` only when `BorderWidth > 0 &&
BorderColor != ""`), while Compose and SwiftUI draw none. So `EmphasisGhost` —
documented as "outlined without the rule" — draws a rule on the web and not on
the natives, and there is no style-level workaround, since `BorderWidth(0)`
emits nothing either. It is a two-line renderer fix with wide golden churn
behind it: every button on both web targets changes.

---

## Tier B — small core additions (4 renderers each, all one-prop changes)

| # | Addition | DOM / htmlout | Compose | SwiftUI | Unblocks |
|---|---|---|---|---|---|
| B1 | `core.Horizontal` on `Scroll` | `overflow-x:auto; flex-direction:row` | `Row(Modifier.horizontalScroll)` | `ScrollView(.horizontal)` | chip strips, tab strips, card carousels |
| B2 | `core.OnEndReached(fn)` on `List` | `IntersectionObserver` on a sentinel row | `LazyListState` last-visible ≥ n-3 | `.onAppear` on last row | infinite scroll; `LoadMore` becomes automatic with a manual fallback |
| B3 | `core.StickyHeader` marker on a List child | `position:sticky` (already supported on web) | `stickyHeader {}` | `Section(header:)` + `pinnedViews` | pinned group headers in A1 |
| B4 | `core.Rotate(deg)` style | `transform: rotate()` | `Modifier.rotate` | `.rotationEffect` | Compass needle (C1), spinners |
| B5 | `core.Switch` node | `<input type=checkbox role=switch>` | `Switch` | `Toggle` | settings screens; today Checkbox stands in |

Fire an end-reached event at most once per data length (debounce on the Go side by
remembering `len(children)` at last fire), so a slow fetch cannot double-load.

**B5 landed 2026-09-11.** The table's web cell was half the answer: HTML has no
switch element, so a `core.Switch` is an `<input type="checkbox">` carrying *two*
things — `role="switch"`, which the sketch has, and HTML's own `switch`
attribute, which Safari draws as a track and a thumb and other engines ignore.
Three things the sketch did not say:

- **The state crosses the wire as `checked`, not `on`.** Go says `on` because a
  switch is on; the wire says what the DOM says, so both web renderers' existing
  `checked` handling — create *and* update-props — works untouched. Naming it `on`
  would have meant four new branches meaning what an existing branch already
  means, and the update half is the one that would have been forgotten: a switch
  drawn right on the first render and frozen after looks like a working widget.
- **It is a node type rather than a flag on Checkbox**, and the reason is the
  reconciler: a changed type is a *replace*, and a replace is how one platform
  control is exchanged for another. A bool prop would have had update-props
  swapping a Compose Checkbox for a Compose Switch inside one node.
- **The role is written from the node type**, which made `htmlout.CarriesOwnRole`
  a table (`ownRoles`) where it had been a comparison against `"Modal"`, and put
  `switch` in `aria/spec.NearMisses` beside `dialog` — roles this framework emits
  and does not name. No `RoleSwitch`: a Role obliges all four renderers to grow
  an arm, and a Material Switch and a SwiftUI Toggle announce themselves.

It also left a test behind that should have existed first:
`mobile/verify/nodetypes_test.go` holds both native dispatches against the tag
table. The natives end in a catch-all, so a forgotten arm is silent by
construction — a childless control draws as an empty box on the phone and
correctly on both web targets — and three of B5's four arms would have been
caught by it.

---

## Tier C — Compass (sensor plumbing + one pure-Go widget)

Value beyond the dial: the start/stop + host-event + permission pattern is exactly what
location (and later, motion) reuse.

1. **Wire contract.** System events out: `sensor.start {"kind":"heading"}` /
   `sensor.stop`. Host event in: `"heading"` with `{"magnetic": 123.4, "true": 130.1,
   "accuracy": 2, "ts": ms}` (`true` omitted when unavailable).
2. **Hosts.**
   - Android: `SensorManager` rotation-vector → `getOrientation` azimuth, low-pass
     smoothed; throttle to ~15 Hz. No permission.
   - iOS: `CLLocationManager.startUpdatingHeading` (magnetic needs no authorization;
     `trueHeading` needs location when-in-use, so report it only when authorized).
   - Browser: `deviceorientationabsolute` (Android Chrome) / `webkitCompassHeading`
     (iOS Safari, needs `DeviceOrientationEvent.requestPermission()` from a user
     gesture — expose that as the start call's job). Desktop reports `available:false`.
3. **Go API.** `core.StartHeading()/StopHeading()`, `hooks.UseHeading(ctx) Heading`
   which starts on mount and stops on cleanup (`core.cleanup.go` pattern), and
   `permission.Heading()` for the browser prompt.
4. **Widget.** `components.Compass{Heading float64, Size int, ShowDegrees bool}` — a
   round `Box` with cardinal labels, a `Rotate(-heading)` dial (B4), and a fixed needle.
   Snapshot-tested with fixed headings; the hook is tested by feeding
   `core.ReceiveHostEvent("heading", …)` directly.

Effort: ~1 session plumbing (three hosts), half a session widget + hook.

---

## Tier D — Map

Two steps, the first nearly free and probably enough for the church app.

### D0. Static map + hand-off (no renderer work)

`components.StaticMap{Lat, Lng, Zoom, Width, Height, Marker bool, Provider}` renders a
`core.Image` from a static-tile URL (OSM static tile endpoint by default, Google Static
Maps when a key is configured) and, on tap, `core.OpenURL` of a `geo:`/`maps://`/
`https://maps.google.com/?q=` link so the platform's own maps app gives directions.
Covers "where is the church / this event". Half a session.

**Landed 2026-09-11.** The sketch's field list survived; what it left open was
every question that turned out to matter, and all three answers are *policies*
rather than values:

- **The provider is a seam, not a URL.** `StaticMapProvider` is one function over
  a `StaticMapArea` — which arrives already defaulted and already clamped, so no
  provider repeats the same four lines and none can disagree about what a zero
  means. `OSMStaticMap` is the keyless default and is a volunteer-run service with
  a low-volume policy, named as such in the doc: right for a church's address
  card, wrong for a screen every user opens ten times a day, and
  `GoogleStaticMap(key)` is beside it.
- **The hand-off cannot be `geo:`.** Nothing in this framework knows which
  platform it is on — deliberately, see `core.OpenURL` — so the default is the one
  https URL all three resolve, and which on both phones reaches the installed
  maps app. The label is deliberately *not* in it: `query` is a search, and a
  search for "St Mary's" lands on whichever St Mary's the geocoder liked. A
  platform-specific hand-off is a one-line func, and the `label` argument on
  `MapHandoff` exists for it.
- **A tappable map is `RoleLink`, not `RoleButton`.** The tap leaves the app
  entirely, which is what that distinction is for; with no hand-off the widget is
  `RoleImg`, the same argument `Compass` makes, and either way the image inside is
  hidden so the widget announces once instead of reading out a provider URL.

Two smaller ones: latitude clamps at Web Mercator's limit and longitude *wraps*,
because they are two different facts about a sphere — and `Lat 0, Lng 0` is the
Gulf of Guinea, so there is no "unset" coordinate to detect and a screen with no
location yet renders a `Skeleton` instead.

### D1. Live `core.MapView` node (large; gate on a real need)

- **Props:** `Center{Lat,Lng}`, `Zoom`, `ShowUserLocation`, `OnRegionChange`,
  `OnMarkerTap(id)`, `OnMapTap(lat,lng)`. **Markers as keyed child nodes**
  (`core.Marker(id, lat, lng, title, ...)`) — the TextGrid trick — so the reconciler diffs
  one marker, not the marker set, and hosts add/remove annotations from ordinary child
  patches.
- **Hosts:** iOS MapKit `Map` (free, no key); Android **osmdroid** first (no key, no Play
  Services dependency) with Google Maps Compose as a later provider; web **Leaflet** with
  OSM tiles (loaded by the host page, so the serve/index.html gains a script tag).
- **Location fix** ships with it: `"location"` host event, `hooks.UseLocation`,
  `permission.Location()` — same plumbing as Tier C.
- Effort: 3–4 sessions. Region-change events need throttling on every host; test with
  the existing verify harnesses (`wasm/verify`, `ios/verify`, `mobile/verify`).

**Landed 2026-09-11.** The prop list and the host choices all survived. What the
sketch did not have is the rule the whole node rests on, plus three host facts:

- **The region is applied only when it changes.** A map is the one widget whose
  value the user changes continuously by touching it, so a controlled region
  re-asserted on every patch snaps the map out from under the finger on the next
  unrelated render — and an app that echoed `OnRegionChange` into state would
  fight its own round trip, because the echo arrives a frame late. Each host
  remembers the region it last applied and compares. Read the other way, the same
  comparison is what stops a host's own recentring from arriving back in Go as a
  gesture: the first version of the web half used a flag set around `setView`
  instead, and `wasm/verify/mapview_test.mjs` is what said that only closes the
  case where Leaflet fires `moveend` synchronously — which it happens to do with
  animation off and promises nowhere.
- **`Center{Lat,Lng}` and `Zoom` became one `core.Region`**, the same type in both
  directions, so an app echoing a pan stores the type it renders from. Two types
  would have differed in nothing and converted at every seam.
- **iOS is `MKMapView` behind a `UIViewRepresentable`, not SwiftUI's `Map`.** The
  short spelling cannot do three of the four promises: no tap-location API at all
  (so `OnMapTap` would be unimplementable), a `MapMarker` that is not tappable,
  and a region binding that writes back on every frame of a drag. It also needs a
  projection: MapKit speaks `MKCoordinateSpan` in degrees where every other engine
  speaks the slippy zoom level, so the conversion goes through Web Mercator's own
  definition with the width read off a `GeometryReader` — exact in longitude,
  which is the axis zoom is defined by.
- **Leaflet is the host page's dependency.** `wasm/index.html` adds the script and
  the stylesheet; the runtime uses `window.L` when it is there and draws a
  placeholder otherwise. The placeholder adds *no child element* — a MapView's
  children are Markers addressed positionally — so it keeps the region in
  `data-lat`/`data-lng`/`data-zoom`, which is exactly what htmlout exports for the
  same node. That makes the static export upgradeable rather than merely degraded.

Not closed: `GrMobMapView.kt` and `LocationSensor.kt` have never been compiled.
`android/verify` builds only the Kotlin that imports nothing, so the Compose half
is held to its contract by `mobile/verify/mapview_test.go` reading the source —
the strongest instrument available for "all three hosts implement the same rule",
and weaker than a compiler. A gradle build is the next thing to run against it.

---

## Suggested order

1. **A1** GroupedList + DataTable + Pagination/LoadMore, then switch the church sermons
   screen to `GroupedList` grouped by month (proves the API against a real consumer).
2. **A2** the lifted widget bundle (AppBar, Banner, EmptyState, SearchField, ChipStrip).
3. **B1 + B2 + B3** in one renderer pass (horizontal scroll, end-reached, sticky headers)
   — three small props, one trip through four renderers; then flip `LoadMore` to
   auto-load and the chip strip to horizontal.
4. **A3** Calendar/DatePicker. *(landed 2026-09-05)*
5. **C** heading plumbing + `Rotate` + Compass.
6. **D0** StaticMap for the church events screen. *(landed 2026-09-11)*
7. **B4/B5** and **D1** as demand appears. *(B4 landed 2026-09-05; B5 and D1
   landed 2026-09-11 — which is the whole of this plan done, so the next one
   starts from the downstream apps rather than from here.)*

Tutorial: each landed widget gets a lesson in `examples/tutorial` and a ROADMAP line.
