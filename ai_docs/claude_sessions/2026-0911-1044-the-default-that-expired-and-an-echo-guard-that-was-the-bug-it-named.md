# Session: the default that expired, and an echo guard that was the bug it named

Session: https://claude.ai/code/session_01AtctQFkHpNuADXAJrdFDJA
Date: 2026-09-11 · Previous:
`2026-0911-0945-a-map-with-nowhere-to-point-and-a-brand-that-reached-half-the-buttons.md`

## Ask

Items **1–9** of the previous session's Next list, all of them.

Two of the nine turned out to rest on a false premise, and finding that out is
most of what the session was worth.

## The premise that was false: item 1

Item 1 asked for device-pixel-ratio handling in `components.StaticMap`, which
requests a 320px image into a 320 *logical*-px frame. Before writing it, the
host it requests from was checked:

    nslookup staticmap.openstreetmap.de   →  NXDOMAIN
    OSM wiki, StaticMapLite               →  "This service has been discontinued"

The widget's **default provider points at a host that does not exist**, and
`church_mobile` shipped it on the event detail screen last session. A reader
looking at an event has been seeing a grey rectangle.

So item 9 — "the provider decision belongs before a downstream app ships, not
after a block" — was not a future decision. It was overdue, and forced.

### There is no keyless replacement

Every static-map service the OpenStreetMap wiki still lists takes an account.
The two keyless entries on that page are a web form and an HTML-embed
generator, neither of which is a URL an image node can fetch.
`maps.wikimedia.org` serves one and answers a library's image loader with 403.

The obvious escape — compose the map out of `tile.openstreetmap.org` raster
tiles, which are keyless, alive, and already what `core.MapView` draws through
on all three hosts — is **not available**. A tile grid centred on an arbitrary
point needs its tiles at pixel offsets inside a clipped box, which is absolute
positioning, and `absolute` is the one `Style.Position` value with no Compose
analog. `GrMobStyle.kt` says so in as many words. A widget that laid out
correctly on web and iOS and drifted on Android is worse than one asking for a
key.

So: **`StaticMap` has no default provider any more.** `OSMStaticMap` is kept
and deprecated (a provider returning a URL to a dead host is easier to diagnose
than a symbol that went away), a nil `Provider` renders the frame and no image,
and `ConcernNoMapProvider` names it in debug mode. A build that has not chosen
should look unfinished and say so.

### And then the DPR fix, which the field separation is

`Scale` on `StaticMap` and `StaticMapArea`. The point is that `Width`/`Height`
stay **logical** and feed the box, while `Scale` feeds only the URL — one
number could not answer both questions, which is why this was "not expressible
from the call site". `GoogleStaticMap` spends it (`scale=1|2`, size limit
applied *before* scaling, so a 640px box at 2x is a legal 1280px request);
`OSMStaticMap` ignores it and says so.

Deliberately not a multiply on `Width` before the provider sees it, and the
reason is the interesting half: asking a map service for twice the pixels at
the same zoom returns twice as much *map*, and asking at zoom+1 returns the
same ground drawn for a deeper zoom, whose labels land at half the physical
size they were drawn for. Neither is a sharper picture of the same thing. It is
a provider capability or it is nothing.

Nothing reads the ratio off the device because nothing in grmob reports screen
metrics on any renderer. `church_mobile` states `MapScale = 2` as a constant,
with the cost written down: four times the bytes, for one card-sized map on a
screen the reader opened deliberately.

## The bug: item 3, and it was in all three hosts

Item 3 was "the Leaflet path has never run in a browser; the fake proves the
bookkeeping and says nothing about Leaflet's own promises."

It ran. Tiles, markers, attribution, `OnMapTap`, `OnRegionChange` — all live,
all correct. And then:

    user pans              map at 38.7250,-9.1500 z14, reported to Go ✓
    a pin is dropped       map snaps back to 38.7139,-9.1394 z13 ✗

The lesson's own prose, on the page it was refuted on: *"The map stays where
you left it anyway, and that is the one rule that makes a controlled map
usable."*

### One slot serving two facts

`record.applied` was documented as "the region this code last handed to
Leaflet" and was written by **both** paths. A pan wrote the *user's* region
into it, so Go's **unchanged** region then read as a change, and the next patch
to reach the map — a dropped pin, a marker moving, any unrelated re-render —
re-applied it. That is the exact failure the guard is named for, described
verbatim in the comment above the code that had it.

The fix is two memories, because they are two facts:

    apply    want !== applied    Go changed its mind — an instruction
             want === settled    the map is already there — Go echoing the pan
                                 back, which must not land a frame late
    report   next !== applied    not the moveend our own setView caused
             next !== settled    not a second event for a map that has not moved

`settled` is where the map rests, however it got there — so `applyRegion` sets
it after a programmatic move too, or a later instruction back to the user's old
view would be skipped as "already there" while the map sat elsewhere.

Fixed identically in `wasm/grmob-runtime.js`, `GrMobMapView.kt` and
`GrMobMapView.swift`, and verified live in the browser afterwards: the pin
drops, the map stays.

### Why the fake never saw it

`wasm/verify/mapview_test.mjs` had both halves and never composed them. Its pan
test sets `maps[0].center` **without firing moveend**, so the report path never
ran and never corrupted the slot — an arrangement real Leaflet never produces,
because a pan always fires moveend. Three tests added; the one that names the
bug fails on revert, the other two hold invariants the fix must not break.

`mobile/verify/mapview_test.go` now pins six strings per native host instead of
three, and its statement of the contract was rewritten — it had been asserting
that each host "must record what it applied on **both paths**", which is the
bug spelled as a requirement.

### The consequence, stated rather than hidden

Re-rendering the *same* Region is now never a re-centre. An app that pans away
and wants the opening view back cannot get it by handing the same numbers over
again — from a host's side that is indistinguishable from the unrelated
re-render the guard exists to ignore. The remedy is the one `core.MapView`
already recommends (echo `OnRegionChange` into state); no imperative recentre
command was added, because that is a host feature in three languages and the
echo costs one line. Tutorial lesson 4.12 now says this, since its own "Back to
the centre" button is the case.

### A caveat about the browser check

`document.hidden` is true for an automated tab, so `requestAnimationFrame` never
fires, and `syncMap`'s deferral means a MapView created while hidden stays
uncreated until the tab is shown. That is the harness, not a defect — but it
cost twenty minutes of chasing a map that was not there, and a future browser
check should screenshot first.

## Item 2, which took fifteen seconds

The Android app compiles. `android/build.sh && ./gradlew :app:assembleDebug` →
BUILD SUCCESSFUL, a 21MB APK, `GrMobMapView.class` and `LocationSensor.class`
in the dex. The Next item's claim that "the repository has no path that would
compile them" was stale: both files are in `app/src/main/java` and the gradle
project has been there all along. `mobile/verify`'s header said the same thing
and now carries the command instead.

Re-run clean at the end of the session, after the Kotlin echo-guard change.

## The church app, which is items 4, 5, 6, 7 and 9's downstream half

### Per-event coordinates (item 4), as a sidecar table

`models/events.go` is SQLBoiler v1 output marked DO NOT EDIT, and the repo
already has the answer for that: `event_recurrences` and `sermon_cache_access`
are hand-written SQL in 1:1 tables precisely so the generated models stay
untouched. `event_locations` follows them.

The 1:1 shape also states a rule two nullable columns could only check: **a
coordinate is a pair**. One number without the other is a half-typed form, and
here that is unrepresentable rather than rejected — a row exists or it does not.
Range `CHECK`s are typo checks, not projections, for the reason the session
before last settled: clamping moves the church to the pole and passing it
through draws somebody else's map.

Wire form `EventLocationAPI{configured, latitude, longitude}`, the second place
the `configured` convention applies, which is what makes it the contract's rule
rather than one DTO's habit. The list path loads points for the whole window in
one query — including the base event of every rule, because a series anchored
before the window still produces occurrences inside it.

The contract tests were the interesting part. They **passed without the new
expectations**: `EventPoints`' error is logged and not fatal, so sqlmock's
refusal of an unexpected query arrived as a logged failure and every event came
back with no location. A test that passes for the wrong reason again. Both probe
queries are now ordered expectations, and two new tests cover a point on the
detail endpoint and a point surviving recurrence expansion.

### `eventPlace`, the precedence (items 4 and 7)

    the event's own coordinates   an exact answer somebody typed for THIS event
    the church's coordinates      when the location text names the church
    nothing                       everything else

The event's own pair skips the alias list **entirely**, and the argument is that
the alias list only ever answered a question a coordinate answers better. It
also fixes the one case no refinement of string matching can reach: a
"Fellowship Hall" at a second campus matches the site's alias exactly and is a
mile from the configured point.

Item 7 falls out of it. The 📍 row is tappable whenever a point is known —
`GoogleMapsHandoff` needs no provider and no key — and it carries coordinates,
not the name, which is why it had to wait for a point at all. A maps *search* on
"Fellowship Hall" is the failure that function's doc warns about.

### The provider (item 9), as configuration

`mobile.maps.{provider,static_key}` → an `enabled` boolean on the wire, because
the client should not re-derive a two-part decision and get one half right. Both
halves required and neither inferred from the other: a key with no provider is a
site that pasted a credential and stopped; a provider with no key is a site that
chose a service and has not signed up.

The key ships in an unauthenticated payload and that is stated plainly in three
places: a static map is an image URL the phone fetches, so the key is inside it
by construction — as public as the Stripe publishable key beside it, reached
from the other end. What it is not is unmetered. Restricting it is the Google
console's job and the config file cannot do it.

### The theme audit (item 6), which found a coincidence

Reflection over `DefaultTheme` for every field carrying a branded role's value:

    Colors.Primary (#0040DD)     Colors.Primary, Colors.PrimaryOnLight,
                                 Components.Button.Background
    Colors.Secondary (#34C759)   Colors.Secondary, Colors.Success
    Colors.Surface (#F2F2F7)     Colors.Surface

So Secondary and Surface carry nothing that needs releasing — there is no
`SecondaryOnLight` and no Surface-derived slot — and `Colors.Success` shares
Secondary's hex **by accident**, because "resolved" is green and DefaultTheme's
accent happens to be the same green. This app deliberately does not brand
Success, so that pairing is a coincidence to record, not a bug to fix.

Which is the shape the guard had to take: sharing a value and being paired with
a role cannot be told apart by looking at values. `theme_test.go` is a census
with an acknowledged set — every co-valued field is recorded as *released* or
*kept*, a field grmob adds later joins the set on its own and fails by name, and
a released field must not still hold the default after branding. Break-tested by
removing the `PrimaryOnLight` release.

### `core.Switch` gets a consumer (item 5), and confirms the doc

A Settings screen with one real switch: *Load map images*, persisted in
`internal/prefs` — a second bytdb file rather than a second table, because the
session store's `clear()` runs on sign-out and settings must survive that.

Item 5 asked whether `Components.CheckBox` is the right theme base for Switch.
The golden answers it: the base contributes `background:#FFFFFF` and
`border-radius:6px` to an `<input type=checkbox switch role=switch>`, and both
are ignored — by the browser, which draws the control itself, and by both
natives, which read only margin and size off a control's style. Which is exactly
what `core.Switch`'s doc already argued. **The downstream consumer confirmed the
reasoning rather than refuting it**, which is the less interesting outcome and
worth recording as such.

One switch and not three. Almost nothing about this app is the member's choice —
branding, features, amounts and the map provider are all the site's — and a
screen padded with invented toggles would look like a settings screen and behave
like a menu of ways to break the app.

### `core.MapView` gets a consumer, and then a facade (items 5 and 8)

An events map: every located event as a pin, tapping one opens it. The right
widget for the right question — one point wants a picture and a hand-off, a set
wants something to pan.

Item 8 said the facade was worth it "once a second screen wants the same
arrangement", and there is still one screen. What justified it anyway was
written first *in the app*: a bounding box, a cosine correction and a logarithm
are not facts about church events. `components.FitRegion` is exported beside
`components.MapPanel` for callers who drive `core.MapView` themselves, and the
app was converted onto both — the same "prove downstream" discipline as last
session, and it immediately found that `MapPanel` was inheriting `core.Column`'s
theme padding and putting a gutter round a full-screen map.

Two things `FitRegion` cannot do are stated rather than papered over, and pinned
by tests so fixing either is a change to a line:

- **More than half the globe** clamps to `MinFitZoom`, which is 1 and not 0
  because `core.Region` reads a zero zoom as *unstated* and substitutes the
  neighbourhood default — returning 0 would hand back a street view of a set
  spanning continents.
- **The antimeridian** is measured the long way. Tokyo and Honolulu are 62
  degrees apart going east and read here as 298.

## Verification

    go test ./...              green in all three repos
    go vet ./...               clean in all three
    GOOS=js GOARCH=wasm build  green (church_mobile)
    wasm/verify/run.sh         green (380 JS assertions + the transcript replay)
    ios/verify/run.sh          green (view layer type-checks)
    gradlew :app:clean :app:assembleDebug   BUILD SUCCESSFUL
    a real browser             lesson 4.12, before and after the fix

**Three break-tests.** Folding the two map memories back into one slot fails
exactly the new Leaflet test and nothing else. Reversing `eventPlace`'s
precedence names both the screen test and two rows of the table. Removing the
`PrimaryOnLight` release names the field and quotes why.

New tests: 5 in `components/static_map_test.go`, 9 in
`components/map_panel_test.go`, 3 in `wasm/verify/mapview_test.mjs`, 3 more
pinned strings per host in `mobile/verify/mapview_test.go`, 5 in
`resource/apiv1/appconfig_test.go`, 2 in `resource/event/api_contract_test.go`,
4 in `app/app_test.go`, 6 in `app/events_map_test.go`, 3 in
`app/settings_test.go`, 3 in `app/theme_test.go`. Two new goldens (seventeen
now), five re-recorded.

The aggregate golden diff was audited rather than accepted. Everything in it is
explained: the Google URL with `scale=2`, the 📍 row becoming a link, the
Settings row, the second fixture event — and the calendar's day cells turning
from disabled to selectable, which is `Calendar{Min: loaded.first, Max:
loaded.last}` doing what it documents now that the fixture spans two dates
rather than one.

A census fired on its own account: `wasm/verify` holds five sentences quoting
the tracked-Go-file count, which went 408 → 410 with `map_panel.go` and its
test. All five updated.

## Files

**grmob** — `components/static_map.go` (no default provider, `Scale`,
`ConcernNoMapProvider`), `components/map_panel.go` + test (new),
`components/static_map_test.go`, `core/mapview.go` (the two-memory contract),
`wasm/grmob-runtime.js`, `android/.../GrMobMapView.kt`,
`ios/.../GrMobMapView.swift`, `wasm/verify/mapview_test.mjs`,
`mobile/verify/mapview_test.go`, `examples/tutorial/chapter4.go`,
`docs/components.md`, `ROADMAP.md`, `wasm/verify/{repowalks,timings}_test.go`.

**church** (server) — `db/migrate/20260911160000_CreateEventLocationsTable.sql`
(new), `db/bytdb_schema.go`, `resource/event/location_queries.go` (new),
`resource/event/{get_presenter,queries,module_event_form,api_rweb}.go`,
`event_controller/event_controller_rweb.go`, `config/config.go` (`MobileMaps`),
`resource/apiv1/appconfig.go`, two test files.

**church_mobile** — `internal/prefs/prefs.go` (new),
`internal/api/models_account.go` (`MapsConfig`), `internal/api/models_content.go`
(`EventPoint`), `app/services.go` (`MapProvider`, `MapScale`), `app/events.go`
(`eventPlace`, the tappable row, the header action), `app/events_map.go` (new),
`app/settings.go` (new), `app/more.go`, `app/theme.go` (doc), `README.md`, five
test files, seventeen goldens.

## Next

1. **(new · value high) Nobody has run the Android app, only compiled it.**
   `assembleDebug` says the calls exist; it says nothing about whether the
   osmdroid map draws, whether the echo-guard fix behaves on a real MapView, or
   whether `LocationSensor` gets a fix. `adb install` and an emulator is the
   whole of the check, and the browser session is the argument for doing it: the
   bug found there was invisible to every test in the repository.
2. **(new · value high) The iOS map path has never run either.**
   `ios/verify` type-checks the view layer and replays a transcript; MapKit's
   `regionDidChangeAnimated` and the `applying` flag are untested against the
   real delegate. Same fix, same shape, a simulator instead of an emulator.
3. **(new · value medium) `church_mobile` has no map provider configured, so
   its event maps are blank.** The code is right and the deployment is not: a
   site has to get a Google Maps Static key and restrict it. Until then the
   event screen shows the location row and no picture — which is the intended
   behaviour and is not what anybody wants to ship.
4. **(new · value medium) `FitRegion` does not handle the antimeridian.**
   Documented and pinned, and it is a circular mean rather than an average —
   which changes the contract for the centre, since the midpoint of two
   longitudes has two answers. Worth it when a caller has a set that crosses it.
5. **(new · value medium) The events map screen is not reachable from the map
   image.** A reader looking at one event's static map has no way to ask "what
   else is near this". The tap already leaves for the maps app, which is the
   right default; a second affordance would need somewhere to put it.
6. **(new · value low) `prefs` and `session` are two bytdb files.** The reason
   is real (sign-out clears the session store wholesale) and the cost is two
   locks and two WALs. One engine with two tables and a narrower `clear()` is
   the alternative, and it trades a documented boundary for a careful DELETE
   somebody has to keep careful.
7. **(carried · value low) The tile sources are the OpenStreetMap project's
   own**, on `tile.openstreetmap.org`, in osmdroid and Leaflet and now under a
   real app's events map. osmdroid sets a User-Agent and caches to the app's own
   directory, which is the policy's own requirement; Leaflet's attribution
   control is on. The static-map half of this item is closed — that host is gone
   and the app pays Google now.
8. **(new · value low) `components.MapPanel` has one consumer**, which is the
   condition item 8 said to wait for. `FitRegion` is the part that earned
   extraction; the arrangement around it is still a guess at what a second
   screen will want.
9. **(carried · value low) The church app's giving and chat screens are
   unaudited for the theme census.** `theme_test.go` now names every field
   sharing a branded role's colour, which is a claim about the *palette*. What
   it does not check is a screen reading `DefaultTheme` directly instead of the
   context's theme — a different mistake with the same symptom.
10. **(declined, non-goal)** Fifteen entries. See `ai_docs/plans/non_goals.md`.
