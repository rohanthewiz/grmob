# Session: the last three items on a plan, and a map that needed no timing flag

Session: https://claude.ai/code/session_01AtctQFkHpNuADXAJrdFDJA
Date: 2026-09-11 · Previous:
`2026-0911-0725-the-loop-seven-iterations-and-a-next-list-that-emptied.md`

## Ask

"Is the plan in `ai_docs/plans/components-datatable-compass-map.md` complete?"
— then, on the answer: update the status header and build what is left (**B5**,
**D0**, **D1**).

## The audit, which was the question

The plan's header said "everything from Tier C on remains proposed". Six days
stale. Read against the tree rather than against the header:

    B1 Horizontal        core/layout.go:410        landed
    B2 OnEndReached      core/list.go:81           landed
    B3 StickyHeader      core/list.go:26           landed
    B4 Rotate            core/style.go:73          landed 2026-09-05
    Tier C entire        core/heading.go + 3 hosts landed 2026-09-05
    A1's a11y follow-up  core/role.go              landed
    A3's ghost border    borderResetTypes          fixed
    B5 core.Switch       —                         not built
    D0 StaticMap         —                         not built
    D1 MapView           —                         not built

So the answer was "no, three items", and two of the three had been explicitly
gated on demand by the plan's own suggested order. The header was rewritten to
say what landed; the three items were then built.

## B5 — `core.Switch`

`core/switch.go`, four renderers, and three decisions the plan's one-line table
cell did not contain.

**The state crosses the wire as `checked`, not `on`.** Go says `on` because a
switch is on; the wire says what the DOM says. That is not tidiness: both web
renderers already carry a `checked` prop on the create path *and* in
update-props, so the Go-side spelling `on` would have meant four new branches
whose only job is to mean what an existing branch already means — and the update
half is the one that gets forgotten, because a switch drawn correctly on the
first render and frozen thereafter looks like a working widget until somebody
changes its value from Go.

**HTML has no switch element, and the web cell needed two attributes rather than
one.** `role="switch"` is what the plan wrote and it is the announcement half;
the drawing half is HTML's own `switch` boolean attribute, which Safari renders
as a track and a thumb and other engines ignore — degrading to a checkbox, which
is the same bool in the same state.

**The role is written from the node type.** That made
`htmlout.CarriesOwnRole` a table (`ownRoles`) where it had been a comparison
against `"Modal"`, added `OwnRoleFor` so the runtime's copy could be pinned
(`TestRuntimeOwnRolesMatchGo`), and put `switch` into `aria/spec.NearMisses`
beside `dialog` — roles this framework emits and does not name. The fixture was
regenerated with `go run ./aria/gen` from the committed spec copy, which said
something nobody had asked: ARIA gives `switch` an `aria-expanded`.

No `RoleSwitch`, and the reason is mechanical rather than aesthetic: every
`core.Role` obliges all four renderers to grow an arm (`core.Roles()` is held
against each native dispatch in `mobile/verify`), and a Material `Switch` and a
SwiftUI `Toggle` announce themselves already.

It is a node type and not a flag on `Checkbox` because **a changed type is a
replace**, and a replace is how one platform control is exchanged for another. A
bool prop would have had update-props swapping a Compose `Checkbox` for a Compose
`Switch` inside one node's patch.

### The test that should have existed first

`mobile/verify/nodetypes_test.go`: both native dispatches held against
`htmlout.Tags()` plus `TransparentTypes()`. Written because three of B5's four
arms had nothing checking them — the natives' dispatch ends in a catch-all, so a
forgotten arm is silent *by construction*: a childless control draws as an empty
box on the phone and correctly on both web targets.

Break-tested by deleting the Kotlin `"Switch"` arm; it named the type and the
consequence. It then earned itself twice over, catching `MapView` and `Marker`
the moment the node types existed.

## D0 — `components.StaticMap`

An image and a hand-off, no renderer work. The field list survived; every open
question turned out to be a **policy** rather than a value.

- **The provider is a seam.** `StaticMapProvider` is one function over a
  `StaticMapArea` that arrives already defaulted and already clamped, so no
  provider repeats the same four lines and none can disagree about what a zero
  means. `OSMStaticMap` is keyless and is a volunteer-run service with a
  low-volume policy — said in the doc rather than discovered in production —
  with `GoogleStaticMap(key)` beside it.
- **The hand-off cannot be `geo:`.** Nothing in this framework knows its
  platform, deliberately (`core.OpenURL` promises only the portable part), so the
  default is the one https URL all three resolve and which reaches the installed
  maps app on both phones. The label is deliberately **not** in it: `query` is a
  search, and a search for "St Mary's" lands on whichever St Mary's the geocoder
  liked. The `label` argument on `MapHandoff` exists for the platform-specific
  one-liner an app that knows its platform can write.
- **Tappable is `core.RoleLink`, not `RoleButton`** — the tap leaves the app
  entirely, which is exactly the distinction that role's doc draws. With no
  hand-off it is `RoleImg`, the argument `Compass` already makes, and either way
  the image inside is hidden so the widget announces once instead of reading out
  a provider URL.

Two smaller facts: latitude clamps at Web Mercator's limit and longitude
*wraps*, because those are two different facts about a sphere; and `Lat 0, Lng 0`
is the Gulf of Guinea, so there is no unset coordinate to detect and a screen
with no location yet renders a `Skeleton`.

`StaticMap.Area()` is exported because the tutorial wanted to print the URL the
widget would request — and building it by hand there would have been the second
answer the resolution exists to prevent.

## D1 — `core.MapView` and the location fix

The plan's prop list and host choices all survived. What it did not have is the
rule the node rests on.

**The region is applied only when it changes.** A map is the one widget whose
value the user changes continuously by touching it, so a controlled region
re-asserted on every patch snaps the map out from under the finger on the next
unrelated render — and an app that echoed `OnRegionChange` into state would fight
its own round trip, because the echo arrives a frame late. Each host remembers
the region it last applied and compares.

**The same comparison, read the other way, is what suppresses the gesture a
host's own recentring would report.** The web half had a boolean flag set around
`setView` first, and `wasm/verify/mapview_test.mjs` is what said that was wrong:
the flag only closes the case where Leaflet fires `moveend` synchronously from
inside `setView`, which it happens to do with animation off and promises nowhere.
One comparison replaced two mechanisms, and all three hosts now do it that way.

Other departures:

- **`Center{Lat,Lng}` + `Zoom` became one `core.Region`**, the same type in both
  directions, so an app echoing a pan stores the type it renders from.
- **iOS is `MKMapView` behind a `UIViewRepresentable`, not SwiftUI's `Map`.** The
  short spelling cannot do three of the four promises: no tap-location API at all
  (`OnMapTap` unimplementable), a `MapMarker` that is not tappable, and a region
  binding that writes back on every frame of a drag. It also needs a projection —
  MapKit speaks `MKCoordinateSpan` in degrees where every other engine speaks the
  slippy zoom level — so the conversion goes through Web Mercator's own
  definition with the width read off a `GeometryReader`: exact in longitude,
  which is the axis the zoom level is defined by.
- **Leaflet is the host page's dependency**, not the runtime's. `wasm/index.html`
  adds the script and the stylesheet; the runtime uses `window.L` when present and
  draws a placeholder otherwise. The placeholder adds **no child element**, which
  is a constraint rather than a preference: a MapView's children are Markers
  addressed positionally, so chrome inside one would have to be counted by
  `chromeOffset`. It keeps the region in `data-lat`/`data-lng`/`data-zoom` — the
  same attributes htmlout exports — which makes the static export *upgradeable*
  rather than merely degraded.
- **The location sensor reuses Tier C's shape entirely**: one `"sensor"` event
  with a `kind`, refcounted start/stop, a record in between, `Received` beside
  `Available`. What it added is `Accuracy` in metres (5m is a GPS fix, 2000m is a
  guess from the cell tower, and they look identical to code reading only the
  coordinates) and `core.DistanceMeters` — a haversine, because the flat
  approximation breaks at the antimeridian and the notification filter is a
  consumer.
- **The two shells differ on who prompts**, and that is the platform's
  difference: `CLLocationManager` asks from anywhere and there is no fix without
  it, so the iOS sensor asks; Android needs the Activity `Permissions.kt` holds,
  so its sensor reports `available: false` with a reason instead.

## Verification

    go test ./...              green (408 tracked Go files; five count sentences updated)
    wasm/verify/run.sh         green, including the headless-Chrome pass
    ios/verify/run.sh          green, including the iOS-SDK typecheck of the new Swift
    android/verify/run.sh      green — but see below
    GOOS=js GOARCH=wasm build  green
    go vet ./...               clean

New tests, counted off the files rather than remembered: 4 in
`core/switch_test.go`, 7 in `htmlout/switch_test.go`, 1 in
`wasm/verify/ownrole_test.go`, 12 in `components/static_map_test.go`, 12 in
`core/location_test.go`, 3 in `hooks/location_test.go`, 10 in
`core/mapview_test.go`, 5 in `htmlout/mapview_test.go`, 13 in
`wasm/verify/mapview_test.mjs` (against a fake Leaflet in `leaflet.mjs`), 6 in
`mobile/verify/mapview_test.go`, 5 added to `mobile/verify/sensor_test.go`, 1
census in `mobile/verify/nodetypes_test.go`, and 3 tutorial liveness tests.

The fake Leaflet is the instrument the real one cannot be: the runtime's map code
is not rendering, it is bookkeeping — when to call `setView` and when not to,
which marker to move, which id to report at fire time — and a fake that records
calls is exactly the right thing to hold it to. It is the same move `ios/verify`
makes with its fake `LayoutSubview`, and it is what found the echo guard defect.

**The one gap, stated rather than glossed:** `GrMobMapView.kt` and
`LocationSensor.kt` have never been through `kotlinc`. `android/verify` compiles
only the two Kotlin files that import nothing — it is a JVM harness, not an
Android build — and these import Compose, the Android SDK and osmdroid. They are
held to their contract by `mobile/verify/mapview_test.go` reading the source
(echo guard on both paths, the 120 ms throttle, markers by id, marker-tap
suppression, no follow-the-user), which is the strongest instrument available for
"all three hosts implement the same rule" and weaker than a compiler.

## Files

New: `core/switch.go`, `core/location.go`, `core/mapview.go`,
`hooks/location.go`, `components/static_map.go`,
`ios/GrMob/Runtime/GrMobMapView.swift`, `ios/GrMob/App/LocationSensor.swift`,
`android/.../runtime/GrMobMapView.kt`, `android/.../app/LocationSensor.kt`,
`wasm/verify/leaflet.mjs`, plus ten test files.

Touched: the four renderers, `htmlout/tag.go` + `inputtype.go` + `export.go`,
`aria/spec/fixture.go` + the regenerated fixture, `core/host_events.go`,
`forms/inputs.go` (the absent `Switch` builder, recorded as a decision: a switch
means *now* and a form means *later*), `wasm/index.html`, `android/app/build.gradle`
(osmdroid), `AndroidManifest.xml`, six doc pages, `ROADMAP.md` (three entries;
"Location / GPS" retired from Planned), and three tutorial chapters.

Tutorial: lessons **2.6** (two booleans), **4.11** (StaticMap), **4.12** (live
maps and the echo guard). The lesson-count prose was stale in three places —
"Forty", "40", "42" against an actual 46 before this session — and is now 49.

## Next

1. **(new · value high) Build the Android app once.** `GrMobMapView.kt` and
   `LocationSensor.kt` have never been compiled, and the repository has no path
   that would compile them: `android/verify` is a JVM harness over the two
   Kotlin files that import nothing. `./gradlew :app:assembleDebug` on a machine
   with the Android SDK is the whole of the check, and it is the only thing
   standing between "written carefully against the osmdroid 6.1 API" and "known
   to compile". Everything else in D1 is verified — the Swift type-checks against
   the real iOS SDK, the web half runs against a fake Leaflet.
2. **(new · value medium) The Leaflet path has never run in a browser.** The
   fake proves the bookkeeping and says nothing about Leaflet's own promises —
   that `moveend` fires once per gesture, that `circleMarker` draws a circle,
   that `locate({watch:true})` does what the dot needs. `wasm/verify/browser.mjs`
   already drives a headless Chrome; a map pass there would need the CDN (the
   harness is offline by design), so the honest version is a manual check
   against `go run ./serve` with lesson 4.12 open.
3. **(new · value medium) Nothing downstream has adopted any of the three.**
   A1/A2 were proved against `../church/church_mobile` the day they landed;
   B5, D0 and D1 have only the tutorial behind them. The church events screen is
   `StaticMap`'s stated driver and is the next real consumer — and a settings
   screen is where `core.Switch` finds out whether `Components.CheckBox` is the
   right theme base for it.
4. **(new · value low) `core.MapView` has no `components` facade.** `Tabs` wraps
   `TabView` and `Compass` wraps a rotation; a map is still three positional
   floats in a struct literal at the call site. Worth it only once a second
   screen wants the same arrangement — a facade written for one consumer is a
   guess about the second.
5. **(new · value low) The tile sources are the OpenStreetMap project's own**,
   on two hosts and in `components.StaticMap`'s default provider. Every one of
   them is documented as low-volume-only with a note that a real user base wants
   a paid provider, and nothing enforces that. If a downstream app ships a map
   to users, the provider decision is the thing to make before it does, not
   after a block.
6. **(declined, non-goal)** Fifteen entries, plus the loop-instruction
   observation closed last session. See `ai_docs/plans/non_goals.md`.

`ai_docs/plans/components-datatable-compass-map.md` is **done** — every tier
landed, each with its departures recorded at the tier. The next plan starts from
the downstream apps rather than from that file.
