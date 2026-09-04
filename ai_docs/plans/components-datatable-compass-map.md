# Next components: DataTable, a widget bundle, Compass, Map

**Status:** A1 landed 2026-09-04 (`components/grouping.go`, `paging.go`,
`grouped_list.go`, `data_table.go`, tutorial lesson 4.6). The church sermons
switch and everything from A2 on remain proposed.
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

### A3. `Calendar` / `DatePicker` (pure Go, medium)

Month grid built from `time`: 7-column header, 6 `core.Row`s of day cells, controlled
`Selected time.Time` + `OnSelect`, `Min/Max`, `Marked func(time.Time) bool` for event
dots. The events screen wants this; a range variant is a later field, not a new widget.

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

---

## Suggested order

1. **A1** GroupedList + DataTable + Pagination/LoadMore, then switch the church sermons
   screen to `GroupedList` grouped by month (proves the API against a real consumer).
2. **A2** the lifted widget bundle (AppBar, Banner, EmptyState, SearchField, ChipStrip).
3. **B1 + B2 + B3** in one renderer pass (horizontal scroll, end-reached, sticky headers)
   — three small props, one trip through four renderers; then flip `LoadMore` to
   auto-load and the chip strip to horizontal.
4. **A3** Calendar/DatePicker.
5. **C** heading plumbing + `Rotate` + Compass.
6. **D0** StaticMap for the church events screen.
7. **B4/B5** and **D1** as demand appears.

Tutorial: each landed widget gets a lesson in `examples/tutorial` and a ROADMAP line.
