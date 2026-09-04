# Session: GroupedList and DataTable (plan item A1)

Session: https://claude.ai/code/session_01J7qf4UetyH8cQ3D7Egn9dC
Date: 2026-09-04 (follows "wasm-hot-reload")

## Ask

"What other components would be great to add to grmob. How about a datatable
with pagination and row grouping as I could use for a list of sermons in
church_mobile, and any other reasonable-effort components? How about a compass?
How about a map? Make a plan." Then: "Start on A1, GroupedList and DataTable."

## The plan

Written to `ai_docs/plans/components-datatable-compass-map.md`. Tiers by value
per renderer touched:

- **A (pure Go, no renderer work):** A1 GroupedList/DataTable with Pagination
  and LoadMore footers; A2 widgets lifted from the church app (AppBar, Banner,
  EmptyState, SearchField, ChipStrip, Skeleton, StatTile); A3 Calendar/DatePicker.
- **B (one prop each, four renderers):** horizontal Scroll, `OnEndReached` on
  List, sticky group headers, `Rotate` style, Switch node.
- **C Compass:** heading host event (SensorManager / CLLocationManager /
  DeviceOrientation), `hooks.UseHeading`, a pure-Go dial using Rotate.
- **D Map:** D0 StaticMap (tile image + OpenURL hand-off, half a session);
  D1 live MapView node with markers as keyed children (MapKit, osmdroid,
  Leaflet), gated on need.

Constraints that shaped it: `components` is pure Go so it is free on all four
targets; a new node type costs four renderers; Scroll is vertical only and no
scroll position reaches Go; host events already reserve a slot for sensors.

## What landed (A1)

New in `components/`:

- `grouping.go` — `Group{Key, Label, Count}`, the run-length `groupRuns`
  engine, exported `GroupHeader` band (label + count Badge), `itoa`.
- `paging.go` — `Pagination` (three models keyed off `PageCount`/`PageSize`:
  caller-owned pages, widget-sliced pages, open-ended) and `LoadMore` (the
  four-state tail: nothing / Load more / Loading… / error + Retry; Loading wins
  over Err, Err over HasMore; Retry falls back to OnLoadMore).
- `grouped_list.go` — `GroupedList[T]` over `core.List`: keyed rows, optional
  run-length group headers, `Dividers`, `Empty` and `Footer` slots. Holds the
  shared `appendRows` helper DataTable also uses.
- `data_table.go` — `DataTable[T]`: `Column[T]{Title, Text|Cell, Weight,
  Align, Narrow, Less, Sortable}`, `Sort *Sort` + `OnSort`, `Compact`,
  grouping, `OnRowTap`, `Selected`, `Loading`/`Empty`, `Footer`, and a
  `*Pagination` that the table slices itself when it has a PageSize and no
  PageCount. Header row outside the List so it stays put.

Tests: `grouped_list_test.go`, `paging_test.go`, `data_table_test.go`
(grouping runs, key order, dividers, empty+footer, pager edges and callbacks,
LoadMore states, sort toggle/direction/copy-not-in-place, Sortable without
Less, compact keeps caller indices, client and server paging, clamped stale
page, row tap/selection/grouping, htmlout export smoke).

Tutorial: lesson 4.6 "Collections: GroupedList & DataTable" in
`examples/tutorial/chapter4.go` with `TestCollectionsDemoSortsPagesAndLoadsMore`.
README and `docs/tutorial-interactive.md` now say 41 lessons. ROADMAP has the
entry.

## Design decisions

- **Fully controlled, hook-free.** Sort, Page and Compact live in the caller's
  state; the widgets only report intent. So they can sit inside `core.IfElse`
  against a pager's loaded flag without disturbing the hook cursor.
- **Sort, then page, then group.** A client-side page's headers agree with its
  rows; group runs follow the sort. Sorting copies the slice.
- **Grouping is by run, not by bucket.** Sorted input yields one header per
  group; an append-only offset pager can only grow the last group, so nothing
  above the fold moves on Load more. Unsorted input honestly repeats headers.
- **Compact keeps caller indices.** `visibleColumns` carries the original index
  so `Sort.Column`/`OnSort` speak the caller's column list; toggling Compact
  never re-points the active sort.
- **Keys.** Headers are `group:<key>`, dividers `sep:<rowkey>`, positional
  fallback `row:<i>` when Key is nil.
- **Fragments are not flattened** by containers, so child lists are built as
  flat `[]core.PropsAndChildren`, never via `core.For`.
- **`Sort` is a pointer** so "no sort" is nil rather than an ambiguous column 0.

## Gotchas met

- Pagination's default "Next ›" label is the same as the lesson screen's own
  Next button; the curriculum-walk test taps the first Button with that label
  and hit the demo's pager. The demo uses `PrevLabel: "‹ Newer"`,
  `NextLabel: "Older ›"` instead.
- My first expected sort order in a test was wrong, not the sort.
- htmlout emits divs for the table (no `AccessibilityRole` prop exists); an
  ARIA role prop is noted in the plan as a small core follow-up.

## Verification

`go test ./...` passes; `GOOS=js GOARCH=wasm go build ./wasm` builds.

## Next

1. Switch the church sermons screen to `GroupedList` grouped by month with
   `LoadMore` fed from its `usePager` (`../church/church_mobile/app/sermons.go`).
2. Plan A2: the lifted widget bundle.
3. B1–B3 in one renderer pass (horizontal scroll, end-reached, sticky headers).
