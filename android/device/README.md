# The Android device harness

Scripts for driving a grmob app on a real device or emulator, and two answers
they exist to give: **is what the app draws the same as what the app laid
out?**, and **did a change to a screen's tree make the app start faster?**

```sh
android/build.sh ./examples/tutorial
(cd android && ./gradlew :app:assembleDebug)
adb install -r android/app/build/outputs/apk/debug/app-debug.apk
adb shell am start -a android.intent.action.VIEW \
  -d "grmob://lesson/4.12"                          # the deep link, not a scroll

android/device/ui.sh texts                          # what is on screen
android/device/ui.sh tap "Show Belém"               # drive it by that text
android/device/ui.sh drag 540 1900 540 900          # scroll the map into view
android/device/ui.sh paint                          # does it paint outside its box?

android/device/launch.sh 5                          # five cold launches, timed
android/device/launch.sh 1 --frames                 # ...and the check on that number
```

## The launch instrument

`launch.sh` reports `am start -W`'s TotalTime, which ActivityTaskManager
measures itself — no screenshots stealing CPU from the thing being timed, which
is what the iOS side has to do. `--frames` is the check on the one assumption
under that number, and it is worth reading once: TotalTime ends at the
activity's first frame, and this app has an Android 12 splash window that goes
up within a few hundred milliseconds of a five-second launch. The frame column
and the Displayed line are printed for the *same* launch so the question
answers itself.

Two metrics were tried and thrown away before the one in `arrival.py` worked,
both defeated by the same fact — a page of text on white is mostly white — and
both are written up in that file, because each looked correct and printed a
confident wrong column.

It was built to answer whether iOS's `core.List` win (7.4s → 1.7s) reproduced
on Compose. It does, in the same direction and for the same reason, and it is
29% of the screen's cost rather than all of it: the four arms and what they
attribute are in `launch.sh`'s header.

Then it was used to find the other 71%, which is the more useful thing it has
done. The runtime times its own mount now — `adb shell setprop
log.tag.GrMobStartup DEBUG`, then read `GrMobStartup:D` — and the split said
the cost was neither Go nor the JNI crossing but **org.json**, parsing a
payload that was 92.4% zero-valued `core.Style` fields. The tags on that struct
are `,omitzero` as a result:

```
                     bytes on the wire    parse     cold launch
before                       423,472     1666 ms       4850 ms
after                         53,408      249 ms       3530 ms
```

The lesson worth carrying to the next screen is the one the instrument made
cheap: **measure the stages before choosing a lever.** The obvious fix here was
windowing `core.List` over the bridge, which is a protocol change; the
measurement said the parse was the cost, a probe said a streaming parser would
not help (`android.util.JsonReader` is no faster on the same string), and the
actual fix was a struct tag.

## Why these are checked in

Both were written in a scratchpad during a session that found three bugs with
them, and all three were invisible to every automated check in the repository:

- **`core.MapView` painted 389dp of tiles through a 260dp slot**, over the
  captions laid out beneath it. A paint fact, not a tree fact — `go test`,
  `mobile/verify`, `wasm/verify` and `assembleDebug` all pass either way.
  `paint.py` is what measured it, against the platform's own view bounds.
- **The echo guard's exact-equality comparison could never match**, because
  osmdroid quantises its centre to integer pixels. Found by driving the lesson
  and reading its two readouts — `ui.sh texts`.
- **`LocationSensor` never retried after a late permission grant.** Found by
  granting it through the app and watching the readout not change.

The next device run would have rebuilt all of this from scratch, including the
one non-obvious detail: `adb shell input swipe` with a short duration is a
*fling*, and a run driven by flings is not repeatable.

`launch.sh` is checked in on the same argument. Its numbers are one emulator's
and will not survive a different machine, but the *method* is what took the
session — a cold launch that is actually cold, a splash window that makes the
obvious endpoint the wrong one, and two screen-comparison metrics that printed
confident nonsense before the third one worked.

## Why nothing runs them automatically

They need a device. A harness wired into a suite that cannot run it is a third
thing to keep true and a red build nobody can fix, which is the argument that
kept these out of the tree until the bugs above made the case the other way.
So they are run by a person, and `go test ./...` neither knows nor cares that
they exist.

The iOS equivalent went the other way and is a real test —
`ios/GrMobUITests/LiveMapUITests.swift` — because XCUITest is a test framework
with a simulator under it, so there is a runner to hang it on. Android's
equivalent would be Espresso or UI Automator as an instrumented test, which is
a build-system change (a `androidTest` source set, a connected-check task) and
is worth doing the day something needs it. These two files are what that day
would start from.
