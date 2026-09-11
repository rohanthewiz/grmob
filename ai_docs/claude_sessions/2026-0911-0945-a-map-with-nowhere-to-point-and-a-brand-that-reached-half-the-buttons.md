# Session: a map with nowhere to point, and a brand that reached half the buttons

Session: https://claude.ai/code/session_01AtctQFkHpNuADXAJrdFDJA
Date: 2026-09-11 · Previous:
`2026-0911-0913-the-last-three-items-on-a-plan-and-a-map-that-needed-no-timing-flag.md`

## Ask

Next item **3** from the previous session: adopt one of B5/D0/D1 in a real
downstream app. Chosen: `components.StaticMap` in `../church/church_mobile`'s
events screen, which the plan had named as StaticMap's stated driver.

## The blocking fact, found before writing anything

Neither repo held a coordinate.

    api.Event / resource/event/api_rweb.go   event_location, free text
    models/events.go, db/bytdb_schema.go     no lat/lng column
    module_event_form.go                     no lat/lng admin field
    AppConfig / resource/apiv1/appconfig.go  name, theme, logo, giving, features

So `StaticMap{Lat, Lng}` had nothing to render from, and the task was not
"call the widget" but "decide where a point comes from". A compiled-in
constant was ruled out by the server's own stated architecture — `appconfig.go`
says the flags exist "so one app binary can serve any church on this platform"
— which makes a per-build coordinate a contradiction rather than a shortcut.

Asked. Answer: **church-level location in app-config now, per-event later**;
map on the **event detail screen, under the location row**.

## Zero is a place, three times over

The same fact shaped every layer, and it is `components.StaticMap`'s own:
`Lat 0, Lng 0` is the Gulf of Guinea. A `float64` pair has no third state, so
"nobody configured this" has to be carried by something beside the numbers.

- **In YAML** by the pointer being nil. `config.MobileLocation` holds
  `*float64`, because latitude 0 crosses Ecuador, Kenya and Indonesia and
  longitude 0 crosses Ghana and, by definition, Greenwich. A church at 0,0 is
  a church, and `latitude: 0` in a config file means it.
- **On the wire** by an explicit `configured` boolean rather than a nullable
  object. The contract's rule is that every key is present and non-null (the
  client maps it straight into a struct with no optional fields), and the thing
  the client must decide cannot be read off the coordinates — so the server
  that knows says so outright, once.
- **In the app** by `ChurchLocation.Valid()`, which folds the flag and a range
  re-check together so no call site reads the coordinates without the question.

Both coordinates are required together: one without the other is a half-typed
config, not a point. Out of range is **refused, not clamped** — a latitude of
300 is a mistyped 30, and clamping would move the church to the pole while
serving it through would draw somebody else's map. Longitude is refused rather
than wrapped for the same reason, though wrapping is the mathematically correct
operation and is what the widget does with the value it finally gets: wrapping
is right for a number that arrived from a computation, refusing is right for a
number somebody typed.

## `eventAtChurch`, which is the whole of the screen work

`event_location` is one text field doing two jobs — rooms inside the building
("Fellowship Hall", "Sanctuary") and venues across town ("Zilker Park") — and
nothing about either string says which it is. The app has exactly one
coordinate. So the rule never guesses: the site lists the strings that mean
*here* (`mobile.location.aliases`), the site's own name counts without being
listed, and everything else gets no map.

**Matching is exact**, after trim → lowercase → collapse internal whitespace.
Containment was the tempting version and is wrong in both directions:

    "Meet at Fellowship Hall then drive to the lake"   ← alias inside a sentence
                                                          about going elsewhere
    "Sanctuary Baptist, across town"                   ← short alias inside
                                                          another organization

Both would have put the reader at the wrong address. The cost is misses — "Room
204" gets no map — and the asymmetry is the argument: a missing map is a row
that is not there, a wrong map is a reader driving somewhere else. The remedy
for a miss is one config line, which the site owns; the remedy for a wrong map
is a phone call.

**A blank location gets no map** for the same reason. Most events with no
location really are at the church — that is usually why nobody typed one — but
"most" is not "all", and a blank is equally the signature of a venue nobody has
settled on.

The widget is `StaticMap` and not `MapView`, and the argument is the component's
own: one point, not a set, nothing to pan or zoom *to*, and a hand-off to the
maps app the reader already keeps their home address in. `Marker: true` despite
the point being the image's centre — this map sits mid-scroll in a column of
text rows, where the pin is what says the picture is *of* somewhere.

## Two things found by doing it, which is what "prove downstream" is for

### The app did not compile against grmob HEAD

`components.Calendar.Marked` had become `func(time.Time) int`. `eventDays`
migrated from `map[string]bool` to a tally, which is strictly better here: a
Sunday with a service and a potluck now draws two dots, and the screen had the
number for free. Recurring series arrive pre-expanded, so a plain increment is
the right count.

### A live theming regression, and the test that watched it happen

`themeFor` copies `DefaultTheme` and assigns `Colors.Primary`. The palette
carries a **second value per role** — `PrimaryOnLight`, the ink-weight tone a
control with no fill uses, because its real backdrop is the page. The override
replaced the colour and left the default's *measurement* behind.

    filled buttons, headers      #1b5e20   the site's forest green
    calendar month arrows        #0040DD   iOS blue
    giving suggestion chips      #0040DD
    every outlined/ghost button  #0040DD

`TestSiteBrandingReachesTheTree` passed throughout, and that is the finding
inside the finding: it asserts the site's colour appears *somewhere*, which one
branded pixel satisfies.

Fixed by **clearing** the tone rather than assigning the brand into it. An empty
tone is what the palette's own resolver reads as "no measurement here, use the
role colour" — exactly the claim this app can support. Writing the hex in
renders identically while asserting a contrast check nobody ran. What is not
available is leaving the old number in place.

The guard is `TestNoGoldenPaintsTheDefaultAccent`: **absence** is the assertion
worth making, and the fifteen golden files are the only place it is cheap —
they are every screen in the app, in the states the tests stage, already on
disk. A single screen's tree cannot say it, because a test renders one tab at a
time. Break-tested by reverting the fix and re-recording: it names five files
and their counts.

The foot-gun went into grmob's `ColorPalette` doc under "Overriding a role means
releasing its tone", since the first downstream app to brand a theme walked
straight into it.

## Verification

    go test ./...              green in all three repos
    go vet ./...               clean
    GOOS=js GOARCH=wasm build  green (church_mobile)

New tests: 5 in `resource/apiv1/appconfig_test.go` (contract read back through
literal wire keys, not the server's own type), 3 in `internal/api/models_test.go`
(one a 7-case table on `Valid`), 4 in `app/app_test.go` (one a 12-case table on
`eventAtChurch`), 1 census in `app/snapshot_test.go`.

**Three break-tests, and the third is the one worth recording.** Containment
instead of exact match named both wrong-address cases. Deleting the `configured`
guard drew the Gulf of Guinea. But
`TestEventDetailHasNoMapWithoutAConfiguredLocation` initially **passed** with
the guard removed — the map was being suppressed by `eventAtChurch` finding no
alias to match, not by the flag. The fixture now carries aliases with no
coordinates, which is the shape `resolveLocation` actually serves for a
half-filled config, so the flag is the only thing standing in the way. A test
that passes for the wrong reason is indistinguishable from one that works until
the day it is needed.

All fifteen goldens re-recorded. The aggregate diff was audited rather than
accepted: `aria-level`, `aria-pressed`, explicit `border:none`, the new dot
rows, sticky headers and `role="group"` on labelled spans — every one of them a
grmob feature from the sessions since the port was last recorded, nothing
unexplained. A stray `bytdb` 0.8.0→0.9.1 bump that `go build` introduced into
the server's `go.mod` was reverted; it was not part of this work.

## Files

**grmob** — `core/theme.go` (doc only).

**church** (server) — `config/config.go` (`MobileLocation`, `mobile.location`),
`resource/apiv1/appconfig.go` (`Location` DTO + `resolveLocation`),
`resource/apiv1/appconfig_test.go`.

**church_mobile** — `internal/api/models_account.go` (`ChurchLocation`,
`Valid`), `app/services.go` (`ChurchLocation()`), `app/events.go`
(`eventAtChurch`, `normalizeLocation`, `eventLocationMap`, the `eventDays`
tally), `app/theme.go` (the tone release), `README.md` (a configuration
section), three test files, fifteen goldens.

## Next

1. **(new · value high) `components.StaticMap` has no device-pixel-ratio
   handling.** It requests a 320px image and sets a 320 *logical*-px frame, so
   on a 2x or 3x phone it is a 1x asset scaled up — visibly soft, and not
   expressible from the call site, because one size feeds both the URL and the
   box. Google's static API has `scale=2`; the OSM service does not, which is
   what makes this a `StaticMapArea` field and a per-provider decision rather
   than a multiply. First thing a real device will show.
2. **(carried · value high) Build the Android app once.** `GrMobMapView.kt` and
   `LocationSensor.kt` have never been compiled and the repository has no path
   that would compile them — `android/verify` is a JVM harness over the two
   Kotlin files that import nothing. `./gradlew :app:assembleDebug` on a machine
   with the Android SDK is the whole of the check.
3. **(carried · value medium) The Leaflet path has never run in a browser.**
   The fake proves the bookkeeping and says nothing about Leaflet's own
   promises. `wasm/verify/browser.mjs` is offline by design, so the honest
   version is a manual check against `go run ./serve` with lesson 4.12 open.
4. **(new · value medium) Per-event coordinates.** The exact answer, and the
   half this session deliberately deferred: nullable `latitude`/`longitude` on
   `events`, the two admin form fields, the presenter, `api_rweb.go` and its
   contract test. `eventAtChurch` in `app/events.go` is the one place the rule
   lives and the place that changes — an event with its own pair skips the
   alias list entirely. Worth it when a site runs events off-site regularly;
   until then it is a schema migration plus a pair of numbers typed per event.
5. **(new · value medium) `core.Switch` and `core.MapView` still have no
   downstream consumer.** D0 now does. B5's stated next home is a settings
   screen, which is where it finds out whether `Components.CheckBox` is the
   right theme base for it; `church_mobile` has no settings screen yet.
6. **(new · value low) The church app's other theme overrides are unaudited.**
   `themeFor` pushes a branded primary into `Colors.Primary` and
   `Components.Button.Background` and now releases `PrimaryOnLight`. Secondary
   and Surface are assigned with no equivalent check, and grmob's palette may
   grow a third primary-derived slot. `TestNoGoldenPaintsTheDefaultAccent`
   catches the primary case on the next re-record and says nothing about the
   other two.
7. **(new · value low) An off-site event's location is not tappable.** Every
   other actionable row on the detail screen hands a scheme to the platform
   (`tel:`, `mailto:`, `https:`); the 📍 row does nothing for an event the map
   cannot cover. A maps *search* on the free text is the natural completion and
   was left out on purpose — it is the "whichever St Mary's the geocoder liked"
   failure `GoogleMapsHandoff`'s doc warns about, and "Fellowship Hall" is
   exactly the string it goes wrong on.
8. **(carried · value low) `core.MapView` has no `components` facade.** Worth
   it once a second screen wants the same arrangement.
9. **(carried · value low) The tile sources are the OpenStreetMap project's
   own**, on two hosts and in `components.StaticMap`'s default provider — now
   including a real app's event detail screen, which is one step closer to
   users than it was. Every one is documented low-volume-only. The provider
   decision belongs before a downstream app ships, not after a block.
10. **(declined, non-goal)** Fifteen entries. See `ai_docs/plans/non_goals.md`.
