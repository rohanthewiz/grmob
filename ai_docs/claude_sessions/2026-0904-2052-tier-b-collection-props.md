# Session: B1–B3, one trip through four renderers

Session: https://claude.ai/code/session_01AM7rDTyaXrFQ4X6rPDdUWp
Date: 2026-09-04 (follows "bundle-adopted-downstream")

## Ask

"Let's do B1-B3 in one renderer pass" — item 1 of the previous session's Next
list, and item 3 of the suggested order in
`ai_docs/plans/components-datatable-compass-map.md`.

Three small props, four renderers each, plus the widget-level knobs the plan
said to flip afterwards ("then flip `LoadMore` to auto-load and the chip strip
to horizontal").

## What landed

| # | Go | web | Compose | SwiftUI |
|---|---|---|---|---|
| B1 | `core.Horizontal()` | *nothing* | `Row` + `horizontalScroll` | `ScrollView(.horizontal)` |
| B2 | `core.OnEndReached(fn)` | `IntersectionObserver`; `data-onendreached` | `snapshotFlow` over last visible index | `.onAppear` on the last row |
| B3 | `core.StickyHeader()` | *nothing* | `stickyHeader {}` | `pinnedViews` + `Section` |

Widgets: `ChipStrip.Scrollable`, `GroupedList.StickyHeaders`,
`GroupedList.OnEndReached`, `DataTable.StickyHeaders`. Tutorial lesson 4.8.

## The design call that made two of the three free

Both `Horizontal` and `StickyHeader` are **StyleProps that set fields
`core.Style` already carried**, not new props:

    Horizontal()    -> FlexDirection: row, Overflow: auto
    StickyHeader()  -> Position: sticky, Top: "0", ZIndex: 1

That is why the htmlout and WASM columns above say "nothing". Both web targets
already emit every one of those declarations, and `htmlout`'s `styleValue` and
the runtime's `styleFromGrMob` already let an explicit `FlexDirection` override
the node type's stacking axis. The entire web half of B1 and B3 was written
before this session started; only the natives needed code.

The alternative — a `Props["horizontal"]` flag, which is what `keyboardAware`
does — was rejected for a concrete reason, not a stylistic one. **The runtime's
style path is total.** An `update-style` patch carries the whole new `Style`
and `styleFromGrMob` reassigns every property it manages, so a flag living in
`Props` would have had to be mirrored onto the element and re-read on every
style patch, or the first unrelated re-render would quietly stand the strip
back up on end. The style channel was built for exactly this and needs no
mirror.

The natives' side of that bargain: `flexDirection` is read by `GrMobScroll`
alone and `position` by `GrMobList` alone. Both are documented as such in the
style mirrors — the precedent is already there, verbatim, on `flexWrap`
("Read by GrMobRow only").

`GrMobStyle`'s class comment used to say Position had "no Compose analog at
this layer". That is still true of `relative`/`absolute`/`fixed` and was never
true of `sticky`, which is precisely what a pinned lazy-list header is. The
comment now says which of the four it maps and why.

## B2 is the one with real work behind it

Every platform reports the same bottom more than once: an observer re-fires on
resize, `.onAppear` re-fires when a row is recycled, a snapshot flow emits per
visible-index change. A slow fetch therefore sees two or three calls before its
first page lands, and an offset pager loads page 2 twice.

**The debounce is one line of state, in Go**: remember how many rows the list
held when the handler last ran, refuse to run again until that number changes.
The four renderers each get to be as naive as their platform makes convenient
and none has to agree with the others about what "again" means.

Where the state lives: `endReachedState`, a pointer field on `Context` beside
`registry`/`nav`/`focus`, shared by every derived context — so two Managers in
one test binary cannot silence each other's feeds. Keyed by callback ID, and
trimmed from `PurgeUnusedCallbacks` against the registry's own survivors, so a
list leaving the tree does not leave a guard behind. It inherits the registry's
documented stale-ID window and nothing here can close it earlier; the two lists
are indistinguishable from this side.

Consequences worth knowing:

- A page that returns **nothing** leaves the guard shut. Scrolling at the
  bottom of an exhausted (or failed) feed does not re-ask forever. The retry is
  `LoadMore`'s error arm, which is why the widget docs say *keep the footer*.
- An **empty** list never reports the edge. The first page is the app's to ask
  for. Web and iOS get this free (no observation target, nothing to appear);
  Compose needed an explicit `rowCount == 0` guard.
- Handing the same load function to `OnEndReached` and `LoadMore.OnLoadMore` is
  the intended shape — a tap and a scroll cannot double-load.

## The runtime's sentinel that isn't

The plan said "IntersectionObserver on a sentinel row". The runtime cannot have
one: patches are addressed positionally (`data-node-path`, and `add-child`
derives the new index from `el.children.length`), so an extra element inside a
list would shift every sibling index after it. The TabView bar is the one piece
of chrome allowed inside a node's box and it pays for the privilege with
`chromeOffset`.

So it observes the **last child** instead, which costs the tree nothing. The
price is that the target moves with every appended page, so the observation is
re-pointed from the same two places `syncTabView` is called: after `renderNode`
builds the children, and after a patch batch lands. The post-batch pass walks
ancestor chains, because a patch aimed at a *row* is still a change to the
list's last child.

`onEndReached` also needed a branch **before** the generic `on*` one in both the
create and update paths — the third prop to need it, after `onDismiss`,
`onTabChange` and `onLongPress`. `mapEventName` would otherwise derive
"endreached", attach a listener for an event that does not exist, and mark the
slot taken so the real wiring could never be installed.

## What the verify harnesses caught

Two existing pins failed, both correctly, and both because code moved rather
than broke:

1. `TestListStretchFillReadsTheAlignFallback` — extracting the stretch equality
   into helpers (`rowFill` in Kotlin, `rowView` in Swift) moved it out of the
   declaration the pin anchors on. **The pin was right and the refactor was
   wrong**: that test exists because this read and the placement dispatch once
   came apart silently on both natives at once. The equality moved back inside
   `GrMobList` / `struct GrMobList`, with a comment saying why it lives there.
2. `TestNativeContainersSpaceAlongTheirOwnAxis` — Swift's `GrMobScroll` now
   reads `horizontalGap`, which the pin listed as the wrong axis. A Scroll has
   two axes now, so it joins `GrMobFlexStack` and `GrMobRow` as a container
   that must mention both and can reject neither.

New pins in `mobile/verify/collections_test.go`: both style parsers read
`FlexDirection`/`Position`, and each native's Scroll and List name both the
read *and* the platform primitive. Reading a key and doing nothing with it is
the same outcome as not reading it.

## Testing

- `core/list_props_test.go` — the three prop→Style mappings, the "supplied not
  forced" defaults in both argument orders, and five tests over the debounce
  (once per row count, quiet when a page adds nothing, purged with its
  callback, not shared between apps, shared with derived contexts).
- `htmlout/collection_props_test.go` — the claim that two of three needed no
  exporter code, stated as a test so it fails if the export stops carrying them.
- `wasm/verify/endreached_test.mjs` — 8 tests. The harness gained a
  controllable `IntersectionObserver` (`observedBy`, `intersect`) beside its
  existing frame and timer queues. The interesting ones are the re-pointing:
  an appended page, a patch aimed at a row, a list that loses the prop.
- `mobile/verify/collections_test.go` — as above.
- Widget tests for all four new fields; tutorial tests for 4.8.
- `go test ./... -race`, `go vet`, `gofmt`, `ios/verify` (replay + typecheck),
  `wasm/verify` (replay + unit), `gradlew compileDebugKotlin` — all clean.

## One risk deliberately removed

SwiftUI's sectioned list is only built when a child actually asked to be
pinned; `listSections` returns `nil` otherwise and the flat `LazyVStack` is
rendered exactly as before. A `Section` changes what the lazy stack's items
*are*, the iOS harness type-checks but cannot see layout, and every existing
List would have gone through the new shape for no reason. Compose needed no
such guard: its hand-written loop emits `item(key:contentType:)` per row, which
is what `itemsIndexed` was sugar for.

## Deliberately not done

- **A horizontal `List`.** The lazy containers' cross-axis and `FlexGrow`
  contracts are written for a vertical main axis on both natives, and nothing
  asks for a lazily-materialized carousel. A strip of chips is short by
  construction, which is what `Scroll` is for. Documented on `Horizontal`.
- **Pinning `DataTable`'s column header.** It is a sibling of the body list,
  not a row inside it — which is what keeps it on screen already — and both
  natives pin only inside their lazy container. Documented and tested as *not*
  pinned, so it cannot become a web-only behavior sold as a cross-platform one.

## Next

1. Plan A3: Calendar / DatePicker.
2. `components.Chip`'s selected look (Surface bg, Primary ink) reads as the
   *less* prominent state, so a filter row looks inverted. Now more visible
   than ever: `ChipStrip.Scrollable` puts a long row of them on screen.
3. A `Role`/`AccessibilityRole` prop in core, serving the screen-furniture
   bundle and `DataTable`'s `<table>` semantics together.
4. Small: add the mid-list busy case to `EmptyState`'s doc (carried over).
5. Adopt B1–B3 downstream in church_mobile: the sermons screen's month bands
   want `StickyHeaders`, its pager wants `OnEndReached`, and its year filter
   wants `ChipStrip{Scrollable: true}`.
