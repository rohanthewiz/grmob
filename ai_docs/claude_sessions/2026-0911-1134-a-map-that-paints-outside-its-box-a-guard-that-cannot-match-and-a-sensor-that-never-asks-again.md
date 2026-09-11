# Session: a map that paints outside its box, a guard that cannot match, and a sensor that never asks again

Session: https://claude.ai/code/session_01AtctQFkHpNuADXAJrdFDJA
Date: 2026-09-11 · Previous:
`2026-0911-1044-the-default-that-expired-and-an-echo-guard-that-was-the-bug-it-named.md`

## Ask

Item **1** of the previous session's Next list: *nobody has run the Android app,
only compiled it.*

The item's own argument for doing it was that the browser session had found a
bug invisible to every test in the repository. The emulator run found three
more, and none of them is reachable from `go test`, `wasm/verify`,
`mobile/verify` or `assembleDebug`.

**No repository file changed this session.** `git status` was clean at the end
as it was at the start; the AAR under `android/app/libs/` is gitignored. This
doc is the deliverable.

## The setup, which is worth writing down

Emulator `Medium_Phone_API_36.1` — Android 16, arm64-v8a on an M3, 1080x2400 at
density 420 (2.625x). The whole check is four commands:

    android/build.sh ./examples/tutorial
    cd android && ./gradlew :app:assembleDebug
    adb install -r app/build/outputs/apk/debug/app-debug.apk
    adb shell am start -n com.grmob.app/.MainActivity

The tutorial rather than `examples/mobileapp`, because `core.MapView` has
exactly one consumer in the repository and it is lesson 4.12.

### Driving it by text, not by pixels

The Compose runtime exposes every text node's semantics, so `uiautomator dump`
carries both the string and its bounds. A helper in the scratchpad taps by
substring, which is what made the run repeatable — and one detail mattered
enough to record: **`adb shell input swipe` with a short duration is a fling**.
Three fast swipes travelled from lesson 4.1 to 7.3. A 900ms drag stays below the
fling threshold and moves a predictable amount.

### And a gap that cost a detour

The tutorial's deep links are a *host event* the web page sends
(`deeplink.go`); the native shells drop the unknown system event by
construction, and `AndroidManifest.xml` has no `VIEW` intent filter. So there is
no way to open lesson 4.12 on a device except to scroll to it. Correct by
design, and it means every future device run pays the same scroll.

## Finding 1: the map paints outside its box

The headline is that **osmdroid draws**. Tiles, all three seed markers in the
right places in Lisbon, the correct opening region, `libgojni.so` loaded, no
crash. That is the first time that sentence has been true of a run rather than
of a compile.

Then the two readout captions turned out to be painted *over* by the map. The
layout is right and the drawing is not:

                       layout slot          painted
        x            121 .. 959           87 .. 992
        y            918 .. 1601         703 .. 1726
                     683px = 260dp       1023px

683 is exactly the `core.Height("260px")` the lesson asks for, at density 2.625.
So the slot is correct and the tiles bleed 215px above it and 125px below —
across both `caption(...)` rows, which are themselves laid out correctly
beneath at y=1633 and y=1705.

The measurement was taken twice, and then settled by the platform rather than by
either of my numbers: `adb shell setprop debug.layout true` draws Android's own
view bounds, and the rectangle it drew around the osmdroid MapView is the 260dp
slot. **The View is measured correctly and paints past itself.**

Which narrows the cause to clipping. Compose does not clip a child's drawing
unless a modifier says to — `Modifier.width/height` set the layout size and
nothing else, the same fixed-size fact `docs/platforms/native.md`'s census
already states from the other side — and `GrMobMapView`'s `AndroidView` passes
`marginAndSize(node.style, extra)` with no `clipToBounds`. The consequence is
not limited to the tutorial: **anything laid out under a `core.MapView` on
Android is being painted over.**

Not fixed here. Where to clip is a decision with a cost — `clipToBounds` on
every `AndroidView` or only this one, and osmdroid over-draws deliberately for
scroll smoothness — and item 1 asked for the run.

## Finding 2: the echo guard holds, and reports a move nobody made

The repro that broke in the browser last session passes on real osmdroid:

    pan east            Reported -9.1394 → -9.0684      reported once ✓
    drop a pin          Reported stays -9.0684          map did NOT snap back ✓

That is the fix confirmed on the host it was written blind for. The other half
of the contract holds too, including the part that looks like a bug and is not:

    "Back to the centre" after a pan     no move    ✓ the documented consequence
    "Show Belém"                         moves      ✓
    then "Back to the centre"            moves      ✓ `settled` is not stale

Lesson 4.12's own prose predicted the first of those three and the device
agreed, which is the cross-host statement the tutorial exists to make.

### The part that is wrong

Tapping "Show Belém" changed the **Reported** readout to exactly the region Go
had just asked for. It should not have been able to.

`applyMapRegion` sets `applied` *and* `settled` to the destination immediately
before moving the map. `reportRegion` returns early on either. So a report
getting through at all proves that what osmdroid handed back is not what it was
handed:

    map.controller.setCenter(GeoPoint(lat, lng))   // exact doubles in
    map.mapCenter                                  // not the same doubles out

osmdroid holds its scroll position in integer pixels at the current zoom, so the
round trip through `setCenter`/`getMapCenter` is lossy by construction, and an
**exact-equality** guard can never match on this host. The same thing happens on
load, which is why "Nothing reported yet — pan or zoom the map." is unreachable
on Android.

It is not the browser bug returning. The echo lands on `settled`, the next pass
compares against it and skips, and the map does not jump — it converges in one
extra round trip. What it costs is honesty: **`OnRegionChange` fires for moves
the user did not make**, and an app that treats the callback as a gesture — a
"search this area" fetch, an analytics event — fires it on every programmatic
recentre and once on arrival.

Worth noting which half is untested: whether there is *also* a report of the
intermediate state between `setZoom` and `setCenter` cannot be told from a
readout that shows only the last one. Counting the callbacks needs a consumer
that counts.

## Finding 3: `LocationSensor` never asks again

Item 1's third question was whether `LocationSensor` gets a fix. It could not be
answered as written, and finding out why is half the value:

**No app in the repository calls `hooks.UseLocation`.** Both mentions in the
tutorial are inside a `prose(...)` string and a `codeBlock(...)` string. The
Android location path has never had a consumer to run — so `LocationSensor.kt`
was in the same position `GrMobMapView.kt` was in before the browser session.

So the question was answered with a throwaway probe: its own module in the
scratchpad with a `replace` directive back to the repo, one screen printing the
permission status beside every field of `core.Location`. Outside the repository
on purpose — it is a diagnostic, not an example. (Binding it needs the same
`tool` block `go.mod` uses to hold `x/mobile`, or `go mod tidy` drops the bind
package and `gomobile` fails on an import it cannot resolve.)

### What it found

With the permission granted at launch, **the sensor works**:

    Available: true   Lat 38.720721   Lng -9.144182   Accuracy 2000.0 m

Not the point fed to `adb emu geo fix`, and that is correct: the grant is
coarse-only, Android fuzzes it, and 2000m is what `Location.Accuracy`'s own
comment predicts for "a guess from the cell tower or the IP address". The
permission dialog asks for *approximate* location, which is
`permission.Location`'s documented narrowing showing up on a real prompt. The
refusal path is right too — `Available:false` with
`Error: "location permission not granted"`, immediately, which is the
"ask me" state the class comment says it exists to make drawable.

And then:

    launch with permission revoked     permission: denied    Available: false
    grant it through the app           permission: granted   Available: false
    12s later, with a fix being fed    permission: granted   Available: false

The permission answer reaches Go — the `permission:` line updates — and the
sensor stays dead. The cause is four lines into `start()`:

```kotlin
if (!hasPermission(ctx)) {
    send(... "location permission not granted")
    return          // running stays false
}
```

Nothing calls `start()` again. `core.StartLocation()` is issued once per hook
slot, guarded by `locationRecord.started`, so the host command that would
re-arm the sensor is never sent a second time. **A screen that calls
`hooks.UseLocation` before the permission is granted is dead for the life of the
context tree**, and the only recovery is a remount — which is the navigation
route the hook's doc already recommends, for the unrelated reason of releasing
the GPS.

Whose job the re-arm is, is a real design question — `Permissions.kt` already
delivers the answer as a host event and could re-invoke `start()`; or
`hooks.UseLocation` could re-issue `StartLocation` on a status transition — so
it is left as a decision rather than guessed at.

## Verification

    gradlew :app:assembleDebug           BUILD SUCCESSFUL (tutorial, then probe,
                                         then tutorial again)
    adb install -r                       Success, three times
    a real emulator                      lesson 4.12, pan/pin/buttons
    adb shell setprop debug.layout       the platform's own view bounds
    the location probe                   granted, revoked, and granted-after

No test suite was run, because no repository file changed. The AAR was restored
to the tutorial build and reinstalled, and the emulator was left running.

## Files

**None in the repository.** Everything built this session lives in the session
scratchpad: the probe module (`locprobe/`), the text-driving helper, a PNG band
measurer and an overlay tool for holding a screenshot against a bounds dump, and
twelve screenshots. All of it dies with the session, which is Next item 11.

## Next

1. **(new · value high) `core.MapView` paints outside its layout box on
   Android.** Measured against the platform's own `debug.layout` bounds: the
   View is 260dp and the tiles cover 389dp, over whatever is laid out beneath.
   The fix is a clip on the `AndroidView` in `GrMobMapView.kt`; the decision is
   whether it belongs on that one node or on `marginAndSize` for every
   `AndroidView`, given osmdroid over-draws on purpose. Nothing in
   `mobile/verify` can see this — it is a paint fact, not a tree fact.
2. **(new · value high) `LocationSensor` never retries after the permission is
   granted.** `start()` returns without setting `running`, and
   `core.StartLocation()` is issued once per hook slot, so the re-arm command
   never comes. A screen that asks for location before the grant stays
   `Available:false` for the life of the tree. The open question is whose job
   the re-arm is: `Permissions.kt` already has the answer in hand, and
   `hooks.UseLocation` already has the status. Check iOS for the same shape —
   `LocationSensor.swift` has the same early return to make.
3. **(carried · value high) The iOS map path has never run either.**
   `ios/verify` type-checks the view layer and replays a transcript; MapKit's
   `regionDidChangeAnimated` and the `applying` flag are untested against the
   real delegate. Same fix, same shape, a simulator instead of an emulator —
   and now with three specific things to look for, since findings 1, 2 and 3 all
   have a MapKit/CoreLocation counterpart worth checking rather than assuming.
4. **(new · value medium) The echo guard compares regions with `==` on doubles,
   and osmdroid cannot round-trip them.** Every programmatic move is reported
   back to Go as a region change, including one on load. It converges and does
   not jump, so the cost is a callback that lies about who moved the map. An
   epsilon is the obvious answer and it is not free: it has to be coarse enough
   for a pixel-quantised centre and fine enough that a real one-pixel drag still
   reports. Whether the browser and MapKit have the same lossiness is unchecked.
5. **(new · value medium) Nothing in the repository calls
   `hooks.UseLocation`.** Both tutorial mentions are prose, so the hook, the
   `core.OnLocation` subscription path and both native sensors have no consumer
   any run can exercise. A lesson 4.13 — or a live readout added to 4.12, which
   already owns the `ShowUserLocation`-is-not-`UseLocation` argument — would
   turn the probe into something that ships. `hooks.UseHeading` is worth
   checking for the same gap.
6. **(carried · value medium) `church_mobile` has no map provider configured, so
   its event maps are blank.** The code is right and the deployment is not: a
   site has to get a Google Maps Static key and restrict it.
7. **(carried · value medium) `FitRegion` does not handle the antimeridian.**
   Documented and pinned, and it is a circular mean rather than an average —
   which changes the contract for the centre, since the midpoint of two
   longitudes has two answers.
8. **(carried · value medium) The events map screen is not reachable from the
   map image.** A reader looking at one event's static map has no way to ask
   "what else is near this".
9. **(new · value low) The natives have no deep-link intent filter**, so
   reaching lesson 4.12 on a device is a scroll. Correct by construction — the
   route channel is a host event the web page speaks and the natives drop — but
   every device run pays it, and an `adb shell am start -d` path would make the
   next one a command instead of a search.
10. **(carried · value low) `prefs` and `session` are two bytdb files.** The
    reason is real (sign-out clears the session store wholesale) and the cost is
    two locks and two WALs.
11. **(new · value low) The device-run tooling lives in a scratchpad and dies
    with the session.** Tap-by-text over `uiautomator`, the 900ms anti-fling
    drag, the screenshot band measurer and the bounds overlay are how findings 1
    and 3 were pinned down, and the next device run rebuilds them. Somewhere
    under `mobile/verify` is the obvious home; the argument against is that a
    checked-in harness nobody runs is a third thing to keep true.
12. **(carried · value low) The tile sources are the OpenStreetMap project's
    own**, on `tile.openstreetmap.org`, in osmdroid and Leaflet and now under a
    real app's events map. Confirmed live this session: the tiles served
    normally under the configured User-Agent.
13. **(carried · value low) `components.MapPanel` has one consumer**, which is
    the condition the extraction was told to wait for. `FitRegion` is the part
    that earned it; the arrangement around it is still a guess.
14. **(carried · value low) The church app's giving and chat screens are
    unaudited for the theme census.** `theme_test.go` names every field sharing
    a branded role's colour, which is a claim about the *palette*. What it does
    not check is a screen reading `DefaultTheme` directly.
15. **(declined, non-goal)** Fifteen entries. See `ai_docs/plans/non_goals.md`.
