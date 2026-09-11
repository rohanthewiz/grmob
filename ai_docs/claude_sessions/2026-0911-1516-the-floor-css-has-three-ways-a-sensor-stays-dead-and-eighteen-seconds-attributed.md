# Session: the floor CSS has, three ways a sensor stays dead, and eighteen seconds attributed

Session: https://claude.ai/code/session_01AtctQFkHpNuADXAJrdFDJA
Date: 2026-09-11 · Previous:
`2026-0911-1313-the-whole-next-list-two-devices-and-the-shape-four-docs-recommended-that-the-framework-forbids.md`

## Ask

> Work the Next list items in reasonable batches. Move any non-goals to
> @ai_docs/plans/non_goals.md

Eleven open items. Six are done, five are decided in writing, and one is
blocked on a key that is not this repository's to obtain.

    141178f  item 1        the min-content floor on iOS
    ce83ee4  items 2,3,4   the location sensor, three ways
    d50ab13  item 6 + 5,8,9,10,11   the launch time measured; five moved to non_goals

The three sentences worth keeping if the rest is lost:

1. **SwiftUI's documented way of asking a view for its minimum does not work
   on the one view that needed it** — a `Text` answers `ProposedViewSize.zero`
   with `0.0` — so the CSS floor is computed from the node tree instead.
2. **The state the previous session said `core.Location` does not carry was
   already expressible and merely unreachable**: Active with Received false.
   What was missing was a way back *into* it after a refusal.
3. **The iOS app's 18 seconds to first frame is none of the two suspects**:
   1.8s of it is the framework, 11ms is Go and the parse, and ~17.7s is
   SwiftUI mounting a 424KB tree in a Debug build.

## Item 1 — `min-width: auto` on iOS, and the probe that returned zero

The bug was last session's: `components.ListRow`'s `core.Text("4.12")` beside
a FlexGrow centre column, compressed to one glyph and wrapped down the side of
each row as `4 / . / 1 / 2`. CSS floors every flex item at its own min-content
size, so an overflowing row overflows; `GrMobFlexSolver` had no floor.

### The measurement that redirected the whole change

The obvious implementation is to ask the view: SwiftUI documents
`ProposedViewSize.zero` as a request for a subview's minimum. That was built
first, instrumented on a simulator, and refuted:

    [GRMOBMIN] axis=horizontal i=0 zero=0.0 base=21.0

A `Text` **accepts any width it is proposed** and wraps to fit, so it has no
minimum to report. There is nothing in the view layer to read. So
`GrMobMinContent` computes the floor from the NODE — the widest unbreakable
run of the string, measured with CoreText rather than UIKit, which is what lets
`ios/verify` check the whole thing on a macOS host rather than on trust.

The font ladder became one function answering in two vocabularies
(`grMobFontWeightPair`): a floor measured at the regular face for text that is
drawn bold is quietly too small, and quietly-too-small is the failure mode
nothing visible catches.

### The loop, and the case that proves it is one

A child that stops at its floor is no longer absorbing its share, and the share
it refused is owed by the rest — which can push another onto its own floor. So
`resolve` runs CSS 9.7's freeze loop rather than one division. `ios/verify` has
the case that tells them apart:

    main 170, bases [100,100], mins [90,0]
      one pass   [90, 85]   overflows by 5 that nothing accounts for
      the loop   [90, 80]   adds up to exactly the 170 it was given

Verified by breaking it: both new fixtures fail on a single pass.

### Every unknown resolves downwards, and the badge is why one did not

A floor that is too low leaves a child as crushable as before; one that is too
high overflows a line a browser would have fitted. So a declared width floors
at **0** (CSS takes the *smaller* of the declared and content suggestions, and
this host cannot see the content behind the frame a declaration becomes — which
is also the column `internal/pinfixture` records and `wasm/verify` has watched
Chrome produce), and every leaf that is not text floors at 0.

Padding and margin are **added**, and that was found on the simulator rather
than reasoned: they are in the flex base size the floor is compared against, so
the tutorial's "TRY IT" badge — a Text with 8 points of padding a side — was
still drawn as `TR / Y / IT` until they were. With them it reads `TRY / IT`,
which is what a browser draws at that width.

### The experiment

`examples/tutorial`'s two `core.FlexShrink(0)` declarations were removed and
the app rebuilt. The numbers and the chevrons render whole without them. They
are kept, because they say something stronger and true on every host: the floor
promises "no narrower than the content", the pin promises "no narrower than the
ideal size", and a row number is the second.

**Compose needed nothing, and the reason is recorded**: its `Row` has no
proportional shrink at all — an unweighted child is offered whatever the ones
before it left — so there is no arm for a floor to bound. That is the same fact
`internal/pinfixture` records as the sibling divergence.

## Items 2, 3, 4 — the location sensor, three ways it stayed dead

### The refusal outlived the run it belonged to

Both natives keep a refused start armed. When the re-arm worked, Go's last
event was still the refusal — tens of seconds of a screen printing "location
permission not granted" about a sensor that was working. The previous session
recorded this as needing a state `core.Location` does not carry.

It did not. **Active with Received false** is a sensor that is running and has
said nothing, which is exactly "acquiring"; what was missing was a way back
into it. `core.LocationAcquiring` is that, `acquiring: true` on the wire, and
both hosts send it the moment updates are actually registered.

It resets the report and leaves `Available` true, which is what makes a screen
that has never heard of it improve without being touched: the arm that printed
the reason no longer matches and the arm that draws a spinner does. A new run
does not inherit a previous one's refusal either; a *fix* survives one, because
a place does not stop being true.

It is not a field on `Location`, and the reason is that it would be a redundant
one — `Active && !Received` already says it.

### Location services off and back on

Documented last session as a gap rather than fixed, because the recovery signal
(`onProviderEnabled`) fires only for a registered listener and the sensor
registered only with providers that were **already enabled** — so the one case
that needed the callback was the one case with nothing to deliver it. It
registers with both regardless now and decides separately whether any can
answer. `stop()` consults a new `registered` flag rather than `running`, since
the two came apart the moment a listener could be registered and not running.

**The emulator then found the half reasoning would have missed.** The recovery
re-armed, reported, and delivered no fix. `requestLocationUpdates` on top of a
live registration keeps the platform's own 5m distance filter with the old fix
as its reference, so a device that has not *moved* gets no callback at all. A
stop-and-start in the same session produced one instantly, and the only
difference between them was `removeUpdates`. `start()` tears the registration
down before rebuilding it.

### Verified, against the same sequence on the previous commit

    running          38.72072, -9.14525 — accurate to about 2000 m
    location off     No fix: location is switched off
    location on      Waiting for the first fix        <- the acquiring state
    +4s              38.72072, -9.14525 — accurate to about 2000 m

    the same sequence on 141178f:
    location on      No fix: location is switched off  (forever)

That middle line is `core.LocationAcquiring` end to end, on a device.

### `hooks.UseLocationWhen`

The gate for a screen that cannot be its own route. `UseLocation`'s own doc
says put such a screen behind a route, because a route frame is what releases
the reference — and that costs a screen; a settings pane with a map preview
beside a permission toggle has nowhere to navigate to. The flag is the same
intent with the call site held still: one `core.NewState`, every pass, whatever
the answer, where the `if` a caller reaches for moves every slot after it.
`UseLocation` is now this hook with the flag nailed true.

Its consumer is lesson 4.12, whose panel gains a switch — flick it and the GPS
is released. Default true, so mounting behaves exactly as it did and the
previous session's emulator verification still describes this screen. Flicked
on the emulator, and the readout says so. The test drives the rendered tree and
asserts a hook slot **below** the call site is undisturbed, which is the shape
that actually breaks when a hook goes inside an `if` — a start/stop count would
pass on the broken version.

### The iOS half

The same three changes, built and type-checked, and `armed` now covers
Location Services being off device-wide as well as the two authorization
cases. **The re-arm readout was verified on Android only.** The iOS
device-wide arm is stated as unverified in the file rather than claimed:
whether iOS delivers `locationManagerDidChangeAuthorization` for the global
switch has not been observed, and arming it costs one Bool and cannot make the
dead state deader.

## Item 6 — eighteen seconds, and where they go

Noticed rather than measured last session: "a 4-second screenshot caught the
launch screen, a 22-second one caught the app". Measured from the host now —
screenshot at 0.5s, fingerprint the band of the frame the title occupies,
three cold launches — at **18.0–18.1s**, a tighter band than the method
deserves.

Both named suspects are eliminated, and by a control rather than by argument:

    examples/mobileapp, same build and simulator      1.8s
    bridge.renderInitial() (Go, across gomobile)      4ms
    JSON parse + GrMobNode tree, 424603 bytes         6ms
    Go's own render of the same tree, on this Mac     ~1ms
    everything after the mount, i.e. SwiftUI          ~17.7s

The control is the decisive line: the demo app launches in 1.8s with the same
framework and the same gomobile init, so the 16 seconds that separate them are
the tutorial's own tree. And Go renders that tree in about a millisecond, so it
is not Go. It is SwiftUI building a view per node for a contents screen of 49
two-line rows, in a Debug build.

**What a Release build would do is not known, and a new finding is why.** The
Release configuration crashes the Swift compiler:
`Abort: function substOpaqueTypesWithUnderlyingTypes`, in `GrMobMapView`'s body
getter, under whole-module optimisation. Reproduced identically on `175c00c`,
so it predates this session — and nothing in this repository's verification
path has ever built Release, which is why an app that cannot be compiled for
release was not known to be one.

The min-content floor was checked against the same instrument while it was
out: **17.4–18.0s before, 18.0–18.1s after** — a difference smaller than the
spread between two readings of the same build.

## Items 5, 8, 9, 10, 11 — moved to `ai_docs/plans/non_goals.md`

Each with what was declined, where the shape lives, the argument, and what
would reopen it:

    the iOS cold deep link is not scripted    the prompt is SpringBoard's, not
                                              the app's; the fix is a Universal
                                              Link, which needs a domain
    MapPanel is not rearranged                a second consumer is the
                                              measurement, and there is one
    prefs and session stay two files          the lifetime argument is in
                                              prefs.go; the cost is counted
                                              rather than paid
    the OSM tiles stay                        the policy contemplates this
                                              traffic, the note is beside every
                                              URL, and choosing a provider for
                                              a downstream app is choosing its
                                              billing relationship
    android/device has no runner              a harness wired to a runner that
                                              cannot run is a red build nobody
                                              can fix

## Item 7 — still the user's key

`church_mobile` has no map provider configured, so its event *images* are
blank. Nothing left in code: the README's Maps section carries the yaml, the
provider name and the instruction to restrict the key to the Maps Static API
and this app's bundle ids. Re-confirmed this session. It needs a Google Maps
Static key, which is yours to obtain. The live events map works without one.

## Verification

    go test ./...                    clean, every package
    gofmt -l .                       clean
    wasm/verify/run.sh               clean
    ios/verify/run.sh                11 OK, with the min-content checks
    android/verify/run.sh            clean
    gradlew :app:assembleDebug       BUILD SUCCESSFUL
    xcodebuild build                 BUILD SUCCEEDED
    xcodebuild -configuration Release COMPILER CRASH — pre-existing, see item 6
    LiveMapUITests                   2 tests, both passed on a simulator
    a simulator                      the lesson list pinned and unpinned,
                                     lesson 1.4, the launch-time measurement
    an emulator                      the location sequence above, the GPS
                                     switch, and the A/B against 141178f

## Next

1. **(new · value medium) The iOS app cannot be built for Release.** The Swift
   compiler aborts in `GrMobMapView`'s body getter under whole-module
   optimisation — `substOpaqueTypesWithUnderlyingTypes`, "Possible
   non-terminating type substitution" — on Swift 6.3.3. Reproduced on
   `175c00c` as well, so it is not new; what is new is that anybody has run
   `xcodebuild -configuration Release`. It is a compiler bug rather than an
   error in this code, so the fix is a workaround: break the opaque-type chain
   in that `body` (an `AnyView`, or hoisting the modifier stack into a named
   `ViewModifier`), or disable WMO for the target. Until then nothing here can
   ship, and the launch-time question below cannot be answered.
2. **(new · value medium) 17.7 of the iOS app's 18 seconds are SwiftUI, and
   nobody has seen the release number.** The Debug measurement and its
   attribution are in `LiveMapUITests`. The single most informative next
   reading is the same measurement on a Release build, which item 1 blocks —
   Debug Swift is slow enough that the whole finding may evaporate. If it does
   not, the question is whether a 49-row contents screen should be a
   `core.List` (lazy) rather than a `Scroll` of rows, which would make the
   mount proportional to what is on screen rather than to the chapter.
3. **(new · value low) `GrMobMinContent` floors every leaf that is not text at
   zero.** A Button, an Input, an Image and a MapView each have a min-content
   size in CSS and none of them is a function of a string, so each would need
   its own measurement. Text is where the divergence was found and where it
   bites, being the only leaf whose whole business is to be narrower than it
   wants to be. The under-estimate is safe by construction; what would change
   this is a screen where a button is crushed.
4. **(new · value low) There is no min-height half of the min-content floor.**
   A text's min-content *height* is a function of the width it wraps at, which
   a tree walk does not know, and a Column that overflows its height rather
   than compressing is a bigger behavioural change than the defect that was
   fixed. Columns keep the floorless behaviour they have always had. Stated in
   `GrMobMinContent`'s own doc.
5. **(new · value low) A declared main size floors at 0 rather than at
   `min(declared, min-content)`.** CSS's automatic minimum takes the smaller of
   the two; this host cannot see the content behind the `.frame` a declaration
   becomes, so it takes 0 — right for the empty sized boxes
   `internal/pinfixture` mounts and an under-estimate for a sized Text. Closing
   it means measuring a node's content *before* its own size is applied, which
   is a second walk with a different question.
6. **(new · value low) The iOS location sensor's device-wide arm has never
   run.** Location Services switched off for the whole device is armed on the
   same argument as the two authorization cases, and the signal that would
   finish it is `locationManagerDidChangeAuthorization` — CoreLocation's only
   channel for "the world outside changed". Whether iOS delivers that callback
   for the *global* switch, as opposed to for this app's own authorization,
   has not been observed: `simctl privacy` drives the second and the first is
   several taps inside Settings. The arm costs one Bool and cannot make the
   dead state deader, so it is stated as unverified rather than claimed.
7. **(new · value low) `UseHeadingWhen` does not exist.** `UseLocationWhen` is
   the gate for a screen that cannot be its own route, and the compass has the
   same shape and the same problem one file over. Nothing has asked for it —
   the compass costs almost nothing to leave running, which is the whole reason
   location got the flag first — so this is a symmetry worth having the day
   something wants it rather than now.
8. **(carried · value medium) `church_mobile` has no map provider configured,
   so its event *images* are blank.** Nothing left in code: the README's Maps
   section carries the yaml, the provider name and the instruction to restrict
   the key. It needs a Google Maps Static key, which is yours to obtain. The
   live events map works without one and is reachable from the detail screen.
9. **(declined, non-goal)** Twenty entries. See
   `ai_docs/plans/non_goals.md` — five of them moved there this session.
