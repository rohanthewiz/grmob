# Session: sixteen of eighteen seconds, a tower the compiler could not substitute, and three ways one arm was wrong

Session: https://claude.ai/code/session_01AtctQFkHpNuADXAJrdFDJA
Date: 2026-09-11 · Previous:
`2026-0911-1516-the-floor-css-has-three-ways-a-sensor-stays-dead-and-eighteen-seconds-attributed.md`

## Ask

> Work the Next list items in reasonable batches, but do fix the Release
> compiler crash first

Items 1, 2 and 6 are done. Items 3, 4, 5 and 7 were decisions rather than
work and are now in `ai_docs/plans/non_goals.md`. Item 8 is still a key that
is not this repository's to obtain.

The three sentences worth keeping if the rest is lost:

1. **The opaque-type tower that crashed the Release compiler was also costing
   ten seconds of every launch.** One fix, two findings, and nobody was
   looking for the second.
2. **Optimisation buys nothing here and laziness buys everything**: Release
   and Debug are the same reading, and the contents screen as a `core.List`
   is 1.7s where the scrolled Column was 7.4s — back to the 1.8s floor
   `examples/mobileapp` set.
3. **The location arm recorded as "unverified" was not merely unrun, it was
   wrong in three places** — and the path the device-wide switch actually
   takes is not the one the arm was written for.

## Item 1 — the Release build, and the file the project did not know about

### A different failure first

Reproducing the crash produced a different error: `cannot find
'GrMobMinContent' in scope`. `GrMobApp.xcodeproj` is generated from
`project.yml` and stays untracked, and the copy on this machine predated last
session's new file — so the Release build was failing on a stale project
rather than on the compiler bug. `xcodegen generate`, and the real abort
appeared:

    1.  Apple Swift version 6.3.3
    3.  While evaluating request ASTLoweringRequest(Lowering AST to SIL …)
    4.  While silgen emitFunction SIL function "$s8GrMobApp0aB7MapViewV4bodyQrvg"
    5.  Abort: substOpaqueTypesWithUnderlyingTypes at SubstitutionMap.cpp:651
        Possible non-terminating type substitution detected

### `GrMobMapView` was innocent; it was merely first

The body getter named in the crash is four lines. What it calls is not:
`grMobBox` chains about eleven `some View` extensions, most of them
`@ViewBuilder`, and each conditional roughly doubles the size of the type
produced. Every view in `Renderer.swift` carries the same tower — SILGen
simply reaches `GrMobMapView.swift` first.

**Why Debug never noticed** is the whole shape of the bug: a file-by-file
build leaves an opaque type declared in ANOTHER file abstract, and
`grMobBox` lives in `GrMobStyle.swift`. Whole-module optimisation
substitutes it. So the app could be run, tested and demonstrated for months
while being impossible to ship, with every check green.

`GrMobBoxModifier` is the fix: the chain moved into one non-generic
`ViewModifier`, so the tower is built once over a single fixed `Content` type
and each call site gets the concrete `ModifiedContent<Self,
GrMobBoxModifier>`. The modifier order is the chain verbatim; nothing about
the rendering changed.

### The check that keeps it fixed

`ios/verify/run.sh` compiles the runtime a second time with `-O -wmo`. Seven
seconds, nothing xcodebuild needs — the Command Line Tools' own compiler
reproduces the abort exactly, verified by removing the fix (SIGABRT, 134).
The object file is thrown away; the exit code is the result.

    OK: view layer survives whole-module optimisation (the Release build)

## Item 2 — the measurement that answered its own question twice

Instrument: install, launch, screenshot on a fixed cadence, take the first
frame whose title band matches the settled one. Two false starts are worth
recording, because both would have produced confident wrong numbers —
comparing forward from frame 0 measures SpringBoard leaving rather than the
app arriving, and a run whose last frame is still the blank launch screen has
no reference at all and reports the launch screen's own arrival (0.6s).

| build | three cold launches |
|---|---|
| Debug, `grMobBox` as a chain | 17.45 · 17.68 · 17.62 |
| Debug, `grMobBox` as `GrMobBoxModifier` | 7.40 · 7.32 · 7.52 |
| **Release**, same | 7.50 · 7.19 · 7.29 |

The first row reproduces last session's 17.4–18.1s, which is what makes the
instrument trustworthy.

**Ten seconds were the tower.** A distinct tower type per view type means
generic metadata instantiated per node on the way up, and collapsing it cut
58% of the launch. The Release build was fixed for shipping and turned out to
be a performance fix.

**Optimisation buys nothing.** Release and Debug are the same reading to
within the spread of either, so last session's conjecture — "Debug Swift is
slow enough that the whole finding may evaporate" — is refuted. The remaining
7.4s is SwiftUI building a view per node, and the only lever left is building
fewer of them.

### So the lazy question was measured rather than argued

`examples/tutorial`'s Home was `components.Screen{Scroll: true}` over a
Column of six chapter Cards holding 49 rows. As a `core.List` whose children
are the title, the progress card and the six cards:

    1.85  1.69  1.69      and after the padding fix: 1.72  1.71  1.85

Which is `examples/mobileapp`'s 1.8s. The gap is closed, not narrowed.

**The padding wart is worth knowing.** `core.List` carries the theme's
`Components.Column` base (12/16) and `Screen`'s own column adds another 16,
so the page was inset twice — every child shifted 16 points inward. Caught by
measuring ink columns in the screenshots rather than by eye:

    Scroll (before)         title ink  x =  32.7 .. 265.3 pt
    List, padding doubled   title ink  x =  48.7 .. 281.3 pt
    List, padding fixed     title ink  x =  32.7 .. 265.3 pt

`Screen.Style: []core.StyleProp{core.Padding(0)}` — the scaffold's column is
a safe-area frame here and the List is the page. Any screen that follows
`Screen.Scroll`'s own advice ("leave it false when the screen has its own
scrolling region inside it — a core.List") meets the same doubling.

Verified on a simulator (geometry restored to the pixel) and on an emulator
(renders, scrolls through all six chapters, a lesson opens). **Android's
launch time was not measured** — `core.List` is a LazyColumn there and should
help, but that is an expectation and not a reading.

## Two test bugs, neither of them this session's

### The map swipe that gestured at nothing

`testEchoGuardOnMapKit` was **already failing** before anything was touched —
the control build on `ccd8bf8` fails identically, so last session's "both
passed" describes a different simulator. Instrumented:

    window = (0, 0, 402, 874)
    map    = (46, -188, 310, 260)       centre y = -58   ← off-window

Steps 1 and 2 scroll down to the readout and the button, which puts the map
mostly above the top of the screen; `swipeLeft()` starts at the element's
CENTRE, so the gesture landed outside the window and the map never moved.
Device-dependent — a taller screen leaves both on at once.
`scrollFullyIntoView` brings the whole map on screen first and asserts it got
there.

### The flick that was four points short

`testLocationPanelSaysWhichStateItIsIn` then failed, but only when run
*second*. The target sits 2989 points down a window 874 points tall, and
twelve `app.swipeUp()` flicks came to within a few points of enough — so
whether it arrived depended on how tall the rows above happened to be, and an
"opened" badge from the previous test was the difference.

`pageDrag` replaces both flicks: a drag across three quarters of the screen,
from a start point chosen clear of every map on screen. The second half
matters as much as the first — a flick starts at the centre, the centre of
these lessons is sometimes a MapView, and a gesture that began on the map
would pan it and manufacture the very region report the assertion three lines
down says should not exist. Only the start point matters; UIKit hands the
whole gesture to the view under the initial touch.

Two tests, run repeatedly, and the second one dropped from 58s to 34s.

## Item 6 — the arm nothing had run, and what running it found

Last session: "Whether iOS delivers `locationManagerDidChangeAuthorization`
for the *global* switch has not been observed… the arm costs one Bool and
cannot make the dead state deader." Driving the taps refuted both halves.

### Driving the switch at all

Two things had to be right before the readings meant anything, and both took
a run to find. Settings restores its last page, so the navigation stack has to
be walked back to the root first. And `element.tap()` on the switch ROW does
nothing — the control sits at the right-hand end of a 330-point element, so
the tap has to be by coordinate, after two seconds of settling or it lands
during the push animation. Until that was right, every reading was of a switch
that had not moved.

### The A/B, against the same sequence on the unfixed build

                       before                     after
    services ON    kCLErrorDomain error 1     Waiting for the first fix
    services OFF   No position, none on way   No position, none on way
    services ON    kCLErrorDomain error 1     Waiting for the first fix
    then relaunch  Waiting for the first fix  Waiting for the first fix

The last row is what makes the third a finding rather than a broken
simulator: the platform was ready the whole time and nothing asked it.

### Three things were wrong, not one

1. **The arm was never set.** The global switch kills a *running* sensor
   through `didFailWithError(.denied)` — not through either of the two paths
   that refuse a *start*, which is where `armed` was written. That handler
   cleared `running` and armed nothing, so the sensor never asked again.
2. **The message was NSError's fallback.** `error.localizedDescription` for
   `CLError.denied` is "The operation couldn't be completed. (kCLErrorDomain
   error 1.)", and it was reaching the screen verbatim. The reason is now read
   off the AUTHORIZATION — still authorized but refused means the switch
   outside the app — because `CLLocationManager.locationServicesEnabled()`
   was observed **returning true with Location Services switched off**, so
   `beginUpdates()`'s own guard cannot name this case.
3. **No callback arrives, so the recovery is the foreground.**
   `locationManagerDidChangeAuthorization` carries this app's authorization
   faithfully and does not carry the global switch. The foreground is the
   right substitute because of where the switch lives: several taps inside
   Settings, so a user who changes it has necessarily left and come back.
   Self-limiting — `armed` is only true after a start was refused or killed.

### And a fourth, found by the fix

With the retry in place the sensor restarted and then sat on "waiting for the
first fix" forever. `startUpdatingLocation` on a manager already in a delivery
session is a no-op on top of that session's state, so a manager that has just
been refused stays refused. `beginUpdates()` now stops before it starts —
**the same shape as last session's Android fix**, where `requestLocationUpdates`
on a live registration kept the platform's distance filter and delivered
nothing. Both hosts now tear down before they build up.

Android needs none of the rest: `onProviderEnabled` fires for the equivalent
switch, which is the asymmetry `LocationSensor.kt`'s `registered` flag records
from the other side.

## Items 3, 4, 5, 7 — moved to `ai_docs/plans/non_goals.md`

Each was a decision already argued in writing rather than work outstanding,
and an item that will never be done costs a re-reading and a re-declining
every session it survives:

    non-text leaves floor at zero      four different measurements, no shared
                                       routine, and the under-estimate is the
                                       behaviour every one of them already had
    no min-height floor                a text's min-content height is a
                                       function of the width it wraps at,
                                       which is the layout's output
    declared size floors at 0          the host cannot see content behind the
                                       .frame a declaration becomes; 0 is the
                                       safe end of the two possible errors
    UseHeadingWhen does not exist      the compass costs almost nothing to
                                       leave running, which is why location
                                       got the flag first

## Verification

    go test ./...                    clean, every package
    gofmt -l .                       clean
    ios/verify/run.sh                clean, with the new WMO check
    wasm/verify/run.sh               clean
    android/verify/run.sh            clean
    xcodebuild -configuration Debug  BUILD SUCCEEDED
    xcodebuild -configuration Release BUILD SUCCEEDED   ← first time ever
    gradlew :app:assembleDebug       BUILD SUCCESSFUL
    LiveMapUITests                   2 passed, run four times over
    a simulator                      nine cold-launch timings across three
                                     builds, the contents geometry measured in
                                     pixels, and the location A/B
    an emulator                      the contents screen renders, scrolls
                                     through all six chapters, opens a lesson

## Next

1. **(new · value medium) The `core.List` launch win is unmeasured on
   Android.** iOS went 7.4s → 1.7s because a LazyVStack materializes only the
   rows on screen; Compose's LazyColumn should do the same for the same
   reason, and the emulator confirms the screen renders and scrolls but
   nothing timed it. The instrument is the harder half: the iOS reading comes
   from `simctl io screenshot` on a fixed cadence, and `adb exec-out screencap`
   is the equivalent. Worth having because it would say whether the finding is
   about SwiftUI or about tree size, which is a fact about the framework
   rather than about one host.
2. **(new · value medium) `components.Screen` insets a `core.List` child
   twice.** `Screen`'s column carries the theme's `Components.Column` padding
   and so does `core.List`, so the documented pattern — `Scroll: false` with a
   List inside, which `Screen.Scroll`'s own doc recommends — shifts every
   child 16 points inward. `examples/tutorial/home.go` works around it with
   `Style: []core.StyleProp{core.Padding(0)}`. The fix belongs in `Screen`
   rather than in each caller, and the question is which rule: drop the
   column's padding when its only child is a scrolling container, or give
   `Screen` a field that says the child is the page. The second is a smaller
   change and a worse API; the first needs a definition of "only child" that
   survives a nil entry.
3. **(new · value low) Nothing measures how long `ios/verify`'s WMO pass
   takes to fail.** It costs about seven seconds when it passes. A future
   change that makes the substitution merely slow rather than non-terminating
   would show up as a verify run that takes minutes, with no message saying
   why. A timeout around that one command would name it.
4. **(new · value low) `HeadingSensor` has no foreground retry.** The compass
   shares CoreLocation's authorization and therefore shares the device-wide
   switch, and `LocationSensor.retryIfArmed` is now the pattern one file over.
   Nothing has reported the symptom — the compass is used on one lesson and
   the sequence that would show it is the same several taps inside Settings —
   so this is stated rather than done, on the same grounds `UseHeadingWhen`
   was declined.
5. **(carried · value medium) `church_mobile` has no map provider configured,
   so its event *images* are blank.** Nothing left in code: the README's Maps
   section carries the yaml, the provider name and the instruction to restrict
   the key to the Maps Static API and this app's bundle ids. It needs a Google
   Maps Static key, which is yours to obtain. The live events map works
   without one and is reachable from the detail screen.
6. **(declined, non-goal)** Twenty-four entries. See
   `ai_docs/plans/non_goals.md` — four of them moved there this session (the
   non-text min-content floors, the missing min-height half, the declared-size
   floor, and `UseHeadingWhen`).
