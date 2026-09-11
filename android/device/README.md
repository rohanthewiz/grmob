# The Android device harness

Two scripts for driving a grmob app on a real device or emulator, and one
answer they exist to give: **is what the app draws the same as what the app
laid out?**

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
```

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
