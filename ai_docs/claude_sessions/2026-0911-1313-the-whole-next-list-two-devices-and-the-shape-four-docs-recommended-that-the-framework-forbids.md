# Session: the whole Next list, two devices, and the shape four docs recommended that the framework forbids

Session: https://claude.ai/code/session_01AtctQFkHpNuADXAJrdFDJA
Date: 2026-09-11 · Previous:
`2026-0911-1134-a-map-that-paints-outside-its-box-a-guard-that-cannot-match-and-a-sensor-that-never-asks-again.md`

## Ask

> Work the Next list items in reasonable batches. Commit and push each when
> completed. When the whole list is done do /sess-wrap

Fourteen open items, one declined. All fourteen are now closed or explicitly
accounted for, in five commits to `grmob` and one to `church_mobile`.

    dd1c7a0  items 1, 2, 4   the map clip, the pixel tolerance, the sensor re-arm
    233072c  item 5          the hook that had no consumer
    aaa25df  item 3          the iOS app, run for the first time
    c41238e  items 7, 9, 11  the antimeridian, deep links, the device harness
    01a6b14  —               a precision claim corrected
    2eea33b  items 8, 14     church_mobile (separate repo, asked before starting)

The three sentences worth keeping if the rest is lost:

1. **MapKit has the echo-guard bug too**, proven by one line changed and the
   same test run twice.
2. **The shape four doc comments recommended for `hooks.UseLocation` is one the
   framework's own debug audit reports as a bug** — which is why that hook had
   no consumer anywhere in the repository.
3. **The first screen of the iOS app was broken** and nobody could have known,
   because nobody had ever run it.

## Item 1 — `core.MapView` paints inside its box now

`GrMobMapView.kt` clips. The decision the last session left open — this node or
`marginAndSize` for every `AndroidView` — went to this node: that chain is also
every material3 control's, and those draw their elevation shadow *inside*
themselves, after the modifier runs, so a `clipToBounds` there would flatten
every raised control in the vocabulary to fix one node that wraps a View with
an over-draw policy of its own.

A corner radius is honored too, because leaflet.css already puts
`overflow: hidden` on Leaflet's container — a `MapView` with a radius had round
corners on the web and square ones here.

Measured after the fix, against the platform's own bounds:

    ViewFactoryHolder   x 121..959   y 231..914     683px = 260dp exactly
    12px above top        0/838 off-ground    page
    just inside top     775/838 off-ground    TILES
    just inside bottom  796/838 off-ground    TILES
    6px below bottom      0/838 off-ground    page
    12px left of edge     0/603 off-ground    page
    12px right of edge    0/603 off-ground    page

Zero pixels outside the box on all four edges, against 215px above and 125px
below before.

## Item 4 — the echo guard compares in pixels, and the width was measured

Half a pixel was the first guess, on the argument that rounding to the nearest
pixel cannot be off by more than half of one. The emulator refused it. Logging
the deltas on the same "Show Belém" tap, at zoom 15 and 38.7°N:

    asked for   38.697,             -9.2065
    got back    38.69703256527164,  -9.206500053405762
    error       3.26e-5°            5.34e-8°
                0.97 pixels         0.0012 pixels

The two axes are not alike. Longitude comes back essentially intact; **latitude
loses nearly a whole pixel**, which is the signature of a Mercator y truncated
to an integer pixel rather than rounded to one. Truncation bounds the error
below one pixel, not below half.

So `GRMOB_MAP_TOLERANCE_PX` is 1.5 — the measured bound with a little room —
and the room cannot hide a gesture, because Android's touch slop is 8dp (21
pixels at density 2.625) before a drag is a drag at all.

The tolerance is a *pixel count converted to degrees at the zoom*, not an
epsilon in degrees: one pixel is 1.7e-4° at zoom 12 and 1.3e-6° at zoom 19, a
factor of 128, so no constant can serve both. Latitude is scaled by cos(lat),
clamped at ±85° — Web Mercator's own limit, and what keeps the cosine off zero.

Exact equality stays on the one comparison where both numbers came from Go. A
tolerance there would swallow a deliberate small nudge — following a location,
stepping a marker along a path — and never reach the map. The browser keeps
exact equality on **both** paths and now has a comment saying why: Leaflet
caches the centre it was given (`_lastCenter`) and hands it straight back.

On the device, all four contract behaviours:

    load                 Nothing reported yet          ✓ was unreachable before
    Show Belém           map moves, reports nothing     ✓ the fix
    a real pan east      -9.2065 → -9.1899, once        ✓
    drop a pin           report unchanged, no snap-back  ✓
    Centre the map on me moves, reports nothing          ✓ (a third kind of move)

## Item 3 — the iOS app, run for the first time

`ios/verify` type-checks the view layer and replays a transcript, so it proves
the renderer agrees with Go about the tree. The map's correctness is mostly not
in the tree. Building against `examples/tutorial`, generating the project and
launching on an iPhone 17 Pro simulator is the first time this app has been run
at all.

### MapKit has the bug, and the experiment is the evidence

Same test, one line changed, everything else identical:

    isSamePlace   "Nothing reported yet" on load, "Show Belém" moves the map
                  and reports nothing, a swipe reports, a dropped pin does not
                  move the map
    isSame        the readout never says "Nothing reported yet" at all — the
                  map reports a region change on load, before anybody has
                  touched it

So the prediction held and 1.5 pixels is enough here too. The part worth
knowing is **why the existing `applying` flag did not already cover it**: that
flag spans the `setRegion` call and catches the callback MapKit fires
synchronously from inside it, but `regionDidChangeAnimated` fires again after
the next layout pass, outside any flag a caller can hold. The comparison is the
only thing standing there.

`ios/GrMobUITests/LiveMapUITests.swift` is that run, checked in — one file per
bound app, `-only-testing`, following `TodoAppUITests`.

### And the first screen was broken

The lesson list rendered its numbers vertically — `4 / . / 1 / 2` down the side
of each row — and sliced the chevrons in half.

A `ListRow` is a Row with a `FlexGrow(1)` centre column; the two-line titles
overflow a phone, so the deficit is shared among the children that can shrink.
On the web and on Compose the number survives, because **CSS floors every flex
item at its own min-content width via `min-width: auto`**. The iOS solver has no
such floor, so it grinds a four-character number down to one glyph.

The root cause is recorded in `GrMobFlex.swift` rather than fixed: the solver
would have to clamp each child at its min-content size, which it cannot do from
`bases` alone — `GrMobFlexLayout` can get the minima from `sizeThatFits(.zero)`
and pass them in, so the shape is clear, but every Row on this host lays out
through there and `ios/verify`'s fixtures, `internal/pinfixture` and
`wasm/verify` all encode the current model.

What is fixed is the declaration that should always have been there:
`core.FlexShrink(0)` on the row number and the chevron, true of both on every
host. It gives `core.FlexShrink` its first production caller — until now it
existed only in fixtures and verify generators. `components.ListRow`'s
`Leading` field carries the trap for the next caller, with why ListRow cannot
fix it on their behalf.

## Item 2 — `LocationSensor` stays armed for a late grant

The grant arrives *after* the screen that wants it, because the tap that asks
is on that screen. Both hosts abandoned the start, and `core.StartLocation`
emits its event only on the 0→1 transition of its reference count, so no second
command ever came.

The two hosts are wired differently and the platform is why: Android's
`LocationManager` has no authorization-change callback, so `Permissions.kt`
hands every answer it sends to Go to the sensor as well; CoreLocation *does*
have one, so iOS re-arms from its own delegate. iOS also had a second bug of the
same shape that Android did not — the `.denied` arm reported and gave up, so a
user granting it in Settings and coming back hit `guard running else { return }`.

Both disarm on stop, so a screen that unmounts while waiting cannot leave a GPS
to start later with no consumer.

**Verified on the device**, which needed item 5 to exist first:

    revoked        No fix: location permission not granted / Undecided
    granted        38.72072, -9.12217 — accurate to about 2000 m / Granted
    centre on me   Asking for: 38.7207, -9.1222 at zoom 16.00

No remount anywhere in that. 2000m is the coarse grant being fuzzed, as
`permission.Location` documents.

What it does **not** fix: Go's last event is still the refusal until the first
fix lands, so a screen reading `loc.Error` shows a stale reason during
acquisition. Saying otherwise needs a state `core.Location` does not carry.

## Item 5 — the hook that had no consumer, and why

`hooks.UseLocation` appeared twice in `examples/tutorial` and both were inside
strings — one `prose(...)` and one `codeBlock(...)`. `hooks.UsePermissionLive`
had no consumer at all.

**The consumer could not be written in the shape four doc comments
recommended.** `core.StartLocation`, `hooks.UseLocation` and both permission
hooks all showed the sensor's hook inside a `case permission.Granted:` arm — a
hook inside a conditional. `core.NewState` hands out slots by call position, so
that arm turning on or off moves every slot after it; `core/debug.go`'s cursor
audit reports exactly this, and `examples/tutorial`'s `TestMain` turns the
audit on. The documented pattern would have failed the suite.

All four docs now say what is safe: both hooks unconditionally, side by side,
and the branch on what gets **drawn**. What gates the dialog is the *route* —
a hook's reference is released when the route frame is popped, so a screen the
user navigated to is a screen the user asked for.

Lesson 4.12 gained the panel that runs its own closing `codeBlock`. The order
of its readout's arms is load-bearing at both ends, and both were wrong once:

1. A fix in hand wins outright — a position that arrived is ground truth and
   the permission record is bookkeeping.
2. Then the permission, *before* `Received` — with `Received` first, a browser
   preview and a Go test, neither of which can ever be granted anything, would
   sit on "waiting for the first fix" forever, which is a spinner telling a lie.

The new test is behavioural rather than a grep, for the reason the gap was
invisible: a name check is satisfied by a string.

## Item 7 — `FitRegion` crosses the antimeridian

Longitude has no ends. `longitudeSpan` is the smallest arc containing every
point, found as the complement of the widest gap between adjacent points — and
a set that does not straddle falls out of the same rule rather than being
detected, so there is no branch.

The centre is the middle of the arc and **not** a circular mean: a mean of
directions is pulled by clusters, so nineteen pins in Tokyo and one in Honolulu
would centre on Tokyo and leave the twentieth off screen.

Two things the tests had to change with it. The antimeridian row was marked
`clamped` with a note saying it was pinned so fixing it would be a change to
that line — it was. And the containment check had to stop *subtracting*
longitudes: it said Honolulu was 329 degrees from a Pacific centre, which is
the same mistake the code was making, so it could not have told a fixed
`FitRegion` from a broken one.

The claim "nothing that worked before moves" was corrected in a follow-up
commit. The *place* does not move; the float does — `-97.75800000000001` became
`-97.758`, one subtraction fewer, fourteen significant figures of agreement,
and a re-recorded snapshot downstream.

## Item 9 — the inbound half of `core.OpenURL`

`core.OpenURL` hands an address out; there was no way back in, so an app opened
by a link opened at its front door with the address thrown away.

`core.OnDeepLink` is the API and the whole of core's involvement: the shells
forward the URL verbatim, because what `grmob://lesson/4.12` means is the app's
business and a shell that knew the word "lesson" would only serve the tutorial.
Android needs an intent filter, `singleTop` and `onNewIntent`; iOS needs
`CFBundleURLTypes` and one `.onOpenURL` covering cold and warm together.

Verified on both. Android, cold and warm:

    grmob://lesson/4.12   opens 4.12
    grmob://lesson/7.3    delivered to the running instance, opens 7.3
    grmob://lesson/2      opens 2.1, the chapter shorthand
    grmob://lesson/       the contents screen
    grmob://settings      nothing moves

iOS the same, through the platform's own gate: a custom scheme from an unknown
source raises "Open in GrMobApp?", and tapping Open lands on the lesson the URL
names. That prompt is why a scheme is a convenience and not a trust boundary.

`mobile/verify` pins both halves, and the manifest half is the part nothing else
can check: XML and YAML read by the OS at install time, with no compiler having
an opinion about whether the entry is there. A missing scheme is not a crash —
the OS simply never offers the app the link.

**A detail worth keeping:** `adb shell am start -d <uri>` without
`-a android.intent.action.VIEW` is matched by component and reaches
`MainActivity` with a null action, which `reportDeepLink` drops. The app opens
at its front door and the link looks ignored. All three places the command is
written down spell the action out.

## Item 11 — `android/device`, checked in

`ui.sh` (drive by accessibility text, with the 900ms anti-fling drag) and
`paint.py` (painted pixels against the View's own layout bounds), plus a README
saying why nothing runs them automatically.

The case changed because these found three bugs this week that every automated
check in the repository passes either way. `ui.sh paint` is one command rather
than three because the three have to pass four numbers between them, and an
unquoted `$box` is four arguments in bash and one in **zsh** — found the hard
way; `paint.py` now accepts either.

The iOS side went the other way (a real XCUITest) only because XCUITest is a
test framework with a simulator under it. Android's equivalent is an
instrumented test, which is a build-system change worth making the day
something needs it.

## Items 8 and 14 — `church_mobile`, a separate repository

Asked before starting, since these are a different repo (`../church/church_mobile`,
branch `roh/use-grmob`).

**Item 14, the theme census's blind spot.** The audit asked whether the giving
and chat screens bypass the theme. Neither reads `core.DefaultTheme`, and
giving is thoroughly branded. The finding is one line up from where the item was
looking: `colorMyBubble = "#E8F0FE"` tinted the member's own chat bubbles, so a
forest-green church had green everything — and pale iOS blue on every message
the reader had sent, the most repeated element on the busiest screen. The
census could not see it, correctly: it walks `core.DefaultTheme` for fields
carrying a branded role's value, and a constant in `app/` is not a field of
anything.

Both quiet tones are now derived — `myBubbleTint` from `Colors.Primary`,
`answeredTint` from `Colors.Success`, the latter precisely so that "does not
follow the brand" is said by construction rather than by a hex that said only
"green". Legibility became a property of the ratio: the worst possible brand is
pure black, which washes to `#E6E6E6` and gives black text 16.8:1 against AAA's
7:1. 10% is also where the two constants already were (8% of the default blue,
12% of the default green — the same intent recorded twice and rounded
differently).

`TestNoScreenSpendsAColourTheBrandCannotReach` closes the blind spot rather
than the instance: no hex literal anywhere in `app/`, comments masked first
because the package quotes the hexes it explains. Verified to fail on a
`"#BADA55"` dropped into `chat.go`.

**Item 8, the question the picture could not answer.** A 🗺 "What else is near
here" row on the event detail, gated on the app knowing the point and *not* on
the map provider — the live map needs no key, so the control is available where
the picture above it is not.

`eventsMapScreenLoading` is a second entry point rather than a default
argument, because `eventsMapScreen` takes its events by value on purpose.

And `navRow`, which is where the existing tests earned their keep: `infoRow`'s
doc states that every tap it gets is a `core.OpenURL` and therefore announces
as a link, so using it made two assertions count three "link rows" where they
meant two. `offSiteLinkRows` is named for the fact that link rows *leave*, so
the fix was not to bump the number — it was a sibling with `RoleButton`. Both
tests pass unchanged.

**Item 6 needs nothing from code.** The README's Maps section already carries
the exact yaml, the provider name and the instruction to restrict the key to
the Maps Static API and this app's bundle ids before deploying it. It needs a
Google Maps Static key, which is yours to obtain.

## Items that needed no change, and why

- **10** (`prefs` and `session` are two bytdb files) — already decided and
  documented; `prefs.go`'s header opens with "Why a second bytdb file and not a
  second table in the session store".
- **12** (the OSM tile sources) — the usage-policy note is already written in
  `GrMobMapView.kt`, `grmob-runtime.js` and `MapPanel`'s own doc, on every host
  that draws OSM tiles. iOS uses MapKit's own. Re-confirmed live on the
  emulator: tiles served normally under the configured User-Agent.
- **13** (`components.MapPanel` has one consumer) — still exactly one, which is
  the condition the extraction was told to wait for. Item 7 strengthened
  `FitRegion`, the part that earned it.

## Verification

    go test ./...                    clean, every package
    gofmt -l .                       clean
    wasm/verify/run.sh               2 OK
    ios/verify/run.sh                11 OK
    gradlew :app:assembleDebug       BUILD SUCCESSFUL
    xcodebuild build                 BUILD SUCCEEDED
    LiveMapUITests                   2 tests, both passed on a simulator
    a real emulator                  lesson 4.12, the location panel, 5 deep links
    android/device/ui.sh paint       CLEAN, exit 0
    church_mobile: go test ./...     clean, 3 snapshots re-recorded

`wasm/verify`'s tracked-Go-file census caught its own five sentences drifting
when this session added two files, and named the lines to edit. 410 → 412.

## Next

1. **(new · value high) The iOS flex solver has no min-content floor.** CSS
   gives every flex item `min-width: auto` and will not compress one below its
   min-content width; `GrMobFlexSolver`'s shrink arm will, which is how the
   tutorial's lesson numbers rendered as `4 / . / 1 / 2` down the side of each
   row. The fix is `GrMobFlexLayout` reading each subview's minimum
   (`sizeThatFits(.zero)`) and the solver clamping to it — a small change in a
   place with a large blast radius, since every Row on the host lays out
   through there and `ios/verify`'s flex fixtures, `internal/pinfixture` and
   `wasm/verify` all encode the current model. Compose wants the same question
   asked of it. `core.FlexShrink(0)` is the portable workaround and is now in
   the tutorial's two row slots.
2. **(new · value medium) `Location.Error` is stale for the whole acquisition
   window.** Both hosts now re-arm a refused start, but Go's last event is
   still the refusal until the first fix lands — tens of seconds on cold GPS —
   so a screen written to the documented shape prints "location permission not
   granted" while the sensor is genuinely working. Saying otherwise needs a
   state `core.Location` does not carry: `available:false` means the device
   cannot produce a fix, and an event with no coordinates decodes to 0,0. A
   fifth field, or an `Acquiring` bool, is a four-host contract change.
3. **(new · value medium) A gated sensor hook, for a screen that must not
   prompt on mount.** Calling `hooks.UseLocation` unconditionally is what the
   hook rules require, and on iOS it means mounting the screen shows the
   dialog. The route is the gate today, which is honest and costs a screen. The
   alternative is `UseLocationWhen(ctx, want bool)` — always one slot, starting
   and stopping the sensor as the flag flips — which would let a screen hold
   the hook and the permission branch in the same pass without a conditional.
   Same shape would suit `UseHeading`.
4. **(new · value medium) `LocationSensor` does not recover from location
   services being switched off and back on.** The permission path re-arms; the
   `!any` providers path sets `running = false` and stops there. The signal
   that would recover it is `onProviderEnabled`, which fires only while updates
   are still registered, and nothing has ever run that path — so it is a
   documented gap rather than an untested recovery. iOS has the same shape in
   its `locationServicesEnabled` guard.
5. **(new · value low) `xcrun simctl openurl` cannot be driven end to end
   without a tap.** iOS puts "Open in GrMobApp?" in front of a custom-scheme
   link from an unknown source, so the cold deep-link path was verified with a
   throwaway XCUITest that attached to the running app and tapped through.
   `LiveMapUITests` still reaches lesson 4.12 by scrolling. Driving it by link
   needs either Safari automation or a verified Universal Link, and the second
   needs a domain.
6. **(new · value low) The iOS app takes 15–20 seconds to first frame on a
   simulator.** Noticed rather than measured: a 4-second screenshot caught the
   launch screen, a 22-second one caught the app. The tutorial's 49-lesson
   first render and gomobile's init are the obvious suspects and neither has
   been timed. `LiveMapUITests` waits 20 seconds for the title for this reason.
7. **(carried · value medium) `church_mobile` has no map provider configured,
   so its event *images* are blank.** Nothing left in code: the README's Maps
   section carries the yaml, the provider name and the instruction to restrict
   the key. It needs a Google Maps Static key. The live events map works
   without one and is now reachable from the detail screen, so the gap is
   narrower than it was.
8. **(carried · value low) `components.MapPanel` still has one consumer**,
   which is the condition the extraction was told to wait for. `FitRegion` is
   the part that earned it and got better this session; the arrangement around
   it is still a guess.
9. **(carried · value low) `prefs` and `session` are two bytdb files.** The
   reason is real and is written at the top of `prefs.go`; the cost is two
   locks and two WALs. Listed so it stays visible, not because anything is
   owed.
10. **(carried · value low) The tile sources are the OpenStreetMap project's
    own**, on `tile.openstreetmap.org`, in osmdroid and Leaflet and under a
    real app's events map. Documented on every host that uses them and
    confirmed serving normally again this session. A build with a real user
    base should point at a provider it pays for.
11. **(new · value low) `android/device` has no instrumented-test runner.** The
    two scripts are run by a person on purpose — a harness wired to a runner
    that cannot run is a red build nobody can fix — but the day something needs
    Android's equivalent of `LiveMapUITests`, an `androidTest` source set and a
    connected-check task is what it starts from, and these two files are the
    content.
12. **(declined, non-goal)** Twelve entries. See `ai_docs/plans/non_goals.md`.
