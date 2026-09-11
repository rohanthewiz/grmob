# Session: two metrics that printed confident nonsense, and a win that was 29% rather than all of it

Session: https://claude.ai/code/session_01AtctQFkHpNuADXAJrdFDJA
Date: 2026-09-11 · Previous:
`2026-0911-1725-sixteen-of-eighteen-seconds-a-tower-the-compiler-could-not-substitute-and-three-ways-one-arm-was-wrong.md`

## Ask

> fix the Screen padding doubling properly then work the Next list items in
> reasonable batches

Items 1, 2, 3 and 4 are done. Item 5 was moved to
`ai_docs/plans/non_goals.md` on the user's instruction.

The three sentences worth keeping if the rest is lost:

1. **The `core.List` win reproduces on Compose and is 29% of the screen's
   cost, not all of it** — so last session's finding was about SwiftUI, not
   about tree size, which is exactly what the item asked. The other 71% is
   423KB of JSON crossing the bridge, paid for all 49 rows whether or not
   Compose composes them.
2. **Two screen-comparison metrics printed confident wrong columns before the
   third worked**, both defeated by the same fact: a page of text on white is
   mostly white. The instrument took longer than the measurement.
3. **`components.Screen` now decides its inset on the *rendered* child**,
   which is what makes "only child" survive a nil entry and what makes
   `components.GroupedList` count — and it costs one assumption about callback
   ordering that is now a test rather than a comment.

## Item 2 — the padding doubling, fixed in the scaffold

### What was doubled and why it was invisible

`Screen`'s column and `core.List` are both built on the theme's
`Components.Column`, whose only entry in every bundled theme is the 12/16
inset. A screen whose whole content is a List was inset twice, and neither
inset is written in any source file:

    SafeArea
      └─ Column   padding 12/16   ← the theme's, via Screen
           └─ List padding 12/16   ← the theme's, again

`examples/tutorial/home.go` worked around it with
`Style: []core.StyleProp{core.Padding(0)}`. That workaround is gone.

### Three decisions, each of which could have gone the other way

**The rule fires on the rendered child, not on the Go value.** That is what
makes "only child" mean what `core.Column` means by it — nil entries skipped,
counted after the skip — and it is what makes `components.GroupedList` count,
since that widget is a `List` only once it renders. A rule written against the
Go type would have missed the widget most screens reach for when the whole
screen is the list. There is a test for each half.

**The set is "scrolls AND arrives pre-inset", which is `core.List` alone.**
It is the only container besides `core.Column` itself built on
`Components.Column`, and that is not a coincidence — it is the pair of
properties that makes the scaffold's inset a duplicate rather than the only
one. `core.Scroll` is deliberately outside the set: it carries no theme base,
so its content is inset once, by this column, and dropping that would move the
page rather than unstack it.

**`Style` still wins**, because the cleared padding is applied *ahead* of the
caller's props and `containerNode` applies style props in argument order. That
ordering is what forced the shape of the implementation: to put a prop before
the caller's, the decision has to be made before any prop is chosen, so
`Screen` renders its sole child itself and hands the node on wrapped.

### The assumption that pre-render rests on, now checked

Rendering the child early is only safe while nothing on `Screen` registers a
callback — `containerNode`'s contract is that a container's callback IDs
precede its children's, and a `Screen` that started registering one would
break it silently: the IDs would still be assigned, in the other order, and the
only symptom would be a handler wired to the wrong node after a diff.

Every prop `Screen` adds is a style prop or `KeyboardAware`, which writes a
flag and asks nothing of the context. `TestScreenRegistersNoCallbacksOfItsOwn`
pins that: the sole child's handler takes `cb_0`, the first ID of the pass.

### Verified in pixels, not just in trees

The unit tests pin the column's inset at both ends, and the app-level
`TestHomeIsInsetOnce` in `examples/tutorial` pins the consequence — both fail
against a build with the rule disabled. On a simulator, the title's ink
columns:

    measured this session     x = 33.0 .. 265.3 pt
    recorded, single inset    x = 32.7 .. 265.3
    recorded, doubled         x = 48.7 .. 281.3

The 0.3pt on the left edge is this session's threshold (`<100` on all
channels) against last session's tooling; the right edge agrees exactly, and
the doubled geometry is 16 points away in both.

## Item 3 — the WMO pass is on a clock

`ios/verify/run.sh` compiles the runtime with `-O -wmo` to keep the Release
build buildable. The fault it guards against is a type substitution that does
not terminate, and the compiler's detector only fires on shapes it recognises:
a chain that grows in some other direction can make the substitution merely
*expensive*, which has no message at all — the verify run stops printing and
the reader's first theory is a slow machine.

    WMO_TIMEOUT="${WMO_TIMEOUT:-120}"          about seven seconds when it passes

The clock is `perl -e 'alarm shift; exec @ARGV'` rather than `timeout(1)`,
which macOS does not ship — coreutils' `gtimeout` is a Homebrew package and
this script's whole premise is that it needs only Go and the Command Line
Tools. SIGALRM leaves the shell's 128+signal, so 142 is the timeout arm and
anything else is the old failure. Both arms exercised.

## Item 4 — the compass's foreground retry

`HeadingSensor` got the pattern `LocationSensor` arrived at last session: an
`armed` flag, a `retryIfArmed` on the same `scenePhase` hook, the
stop-before-start teardown in `beginUpdates`, and the `.denied` message read
off the sensor rather than from `error.localizedDescription` — which for that
code is NSError's untranslated fallback, "The operation couldn't be completed.
(kCLErrorDomain error 1.)".

Two asymmetries with location, documented rather than copied:

    no authorization to change   The compass asks for nothing, so a `.denied`
                                 it receives can only be the device-wide
                                 switch — and there is no authorization
                                 callback it could have used instead of the
                                 foreground. Location had one and it did not
                                 carry the global switch; here there was never
                                 one to begin with.
    no `acquiring` event         A cold GPS fix is tens of seconds away, so Go
                                 has to be told a refusal is withdrawn before
                                 the first reading can say so. A magnetic
                                 bearing arrives in a frame or two and
                                 core.ReceiveHeading clears Error on any
                                 available reading.

`headingAvailable()` is false on the simulator, so `start` never gets past its
own guard and there is no session for the switch to kill. **This arm is not
measured**, and the file says so where the flag is declared rather than
implying otherwise — which is the same thing last session's doc had to correct
about the location arm.

## Item 1 — the Android launch measurement

### The instrument took longer than the measurement

`android/device/launch.sh` and `arrival.py`, checked in on the same argument
as `ui.sh` and `paint.py`: run by a person, nothing in any suite.

It reports `am start -W`'s TotalTime rather than a screenshot cadence. Android
times its own launches, which the iOS side cannot do — and the pulls are
brutal here: a launch `am start -W` reports at 4.6s alone reads 9.5s while
frames are being captured.

**`--frames` exists because the obvious endpoint is wrong.** TotalTime ends at
the activity's first frame, and this app puts an Android 12 splash window up
within a few hundred milliseconds of a five-second launch. If TotalTime ended
there it would be a constant and the A/B would read zero no matter what the
tree did. So `--frames` prints the frame column and the system's `Displayed`
line for the *same* launch and lets the question answer itself:

    splash held to ~10s          and Displayed at 9.84s
                                 → TotalTime is the content

### Two metrics that printed confident wrong columns

Both are written up in `arrival.py`, because each looked correct:

    ink per frame          A blank window and the settled contents screen
                           differ by 4.7 percentage points, because 4.7% is
                           all the ink there is. The launcher wallpaper scored
                           35% and everything else scored 4.7% — a reading
                           that separates the launcher from the app and
                           nothing from anything.
    whole-frame match      A blank white window reproduces every white pixel
                           of the settled screen, which is 95% of them. It
                           scored 89% against 100%, on a column whose whole
                           job is to tell those two apart.

The third works: the denominator is the *reference's ink* — the sampled pixels
that differ from the reference's own commonest colour. A blank window
reproduces none of them, the settled screen all of them. It is what iOS does
one platform over, where "the title band" is that instrument's way of pointing
at the ink.

### The reading

Four arms, means of five cold launches, one emulator
(`sdk_gphone64_arm64`, 1080x2400), the tutorial app unless stated:

| home screen | cold launch |
|---|---|
| title + progress card only, same binary | 2529 ms |
| the whole contents as a `core.List` | 5045 ms |
| the whole contents as a scrolled Column | 6062 ms |
| `examples/mobileapp`, for scale | 3458 ms |

The first arm is the control that makes the rest mean anything: same APK, same
Go binary, same 49 lessons of data, only the chapter cards taken off the
screen.

    List vs Column          1017 ms   the composition of the rows that are not
                                      on screen, the only thing that differs
    List vs near-empty      2516 ms   everything else that scales with the
                                      tree, which laziness does not touch

**So the lazy container wins, in the predicted direction and for the predicted
reason, and it wins 29% of the screen's cost where on iOS it won all of it.**
The question the item asked was whether the finding is about SwiftUI or about
tree size. It is about SwiftUI.

### Where the other 2516ms goes

A Go screen reaches Compose as JSON, and this one is **423,472 bytes** of it
(12,382 without the cards). Go builds and serialises all of it in 0.9ms on a
desktop — 0.05ms for the small one — so the Go side is not the cost.
Everything that happens to those bytes afterwards is paid for all 49 rows
whether or not Compose composes them, because the whole tree crosses either
way.

That is the lever Android has and iOS did not need: send fewer nodes, not
compose fewer views. Nothing does that yet. `TestHomeTreeSize` prints the byte
count and fails only at an order of magnitude — the number is a fact about 49
lessons of prose, which is edited, so a new chapter must not fail a test and a
screen that suddenly sends four megabytes must.

One emulator, one device shape, Debug builds of both halves. The A/B is sound
— every arm ran on the same machine within the same hour — but the absolute
numbers are an emulator's.

## Item 5 — moved to `ai_docs/plans/non_goals.md`

`church_mobile`'s static maps, carried across four sessions and re-declined in
each. The missing piece is a Google Maps Static API key, which is a billing
relationship, and the call site is in a downstream repository. Everything this
side owes is done: an unconfigured provider draws an empty box rather than
Google's error tile, `ConcernNoMapProvider` names it in debug mode, and the
yaml and key restrictions are written down. A waiting item in a work list is
work done repeatedly.

## Verification

    go test ./...                    clean, every package
    gofmt -l .                       clean
    ios/verify/run.sh                clean, both arms of the new timeout
    wasm/verify/run.sh               clean
    android/verify/run.sh            clean
    xcodebuild -configuration Debug  BUILD SUCCEEDED
    xcodebuild -configuration Release BUILD SUCCEEDED
    gradlew :app:assembleDebug       BUILD SUCCESSFUL
    LiveMapUITests                   2 passed
    a simulator                      the contents screen's title ink measured
                                     in pixels against both recorded geometries
    an emulator                      20 cold launches across four arms, plus
                                     the frame column and the Displayed line
                                     for one of them

## Next

1. **(new · value medium) Android's lever is the tree, and nothing pulls it.**
   `core.List` saves the composition of off-screen rows and nothing else: the
   contents screen still ships 423,472 bytes of JSON across the bridge on
   every initial render, and 2516ms of a 5045ms launch scales with it. The
   shape of a fix is windowing on the Go side — a `core.List` that renders a
   slice of its children and asks for more as the host scrolls — which is a
   protocol change (the host has to report a visible range) and not a widget
   change. Worth sizing before building: the same 423KB crosses on every
   re-render too, so the win may be larger than the launch number suggests.
2. **(new · value low) `launch.sh`'s numbers are one emulator's.** The A/B is
   sound because every arm ran on one machine in one hour, but nothing says
   whether a physical device shows the same 29%/71% split — a phone's JNI and
   parse are much faster than an emulator's, which would move the ratio and
   possibly the conclusion about which lever matters. One device run would
   settle it.
3. **(carried · value low) `HeadingSensor.retryIfArmed` is unmeasured.** The
   compass shares CoreLocation's device-wide switch and now has the arm, the
   teardown and the named message that `LocationSensor` needed. None of it has
   been observed: `headingAvailable()` is false on the simulator, so the
   experiment that verified the location arm cannot be repeated. It needs a
   physical device, one lesson, and the same several taps inside Settings.
4. **(declined, non-goal)** Twenty-five entries. See
   `ai_docs/plans/non_goals.md` — `church_mobile`'s static maps moved there
   this session, which empties the last carried item off this list.
