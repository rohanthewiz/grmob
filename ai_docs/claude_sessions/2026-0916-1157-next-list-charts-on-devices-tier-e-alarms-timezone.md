# Next list: charts on devices, Tier E alarms, and Go's time zone on Android

**Session:** c28dcc41-506d-49d8-a6d2-9349c4baed02
**Date:** 2026-09-16 11:57 (follows "charts-on-canvas")
**Branch:** master (2a97a83 → this commit)

## 1. The ask

"Do the latest items in the Next list. Use the emulator / simulator as
necessary." I read that as the age-0 and age-1 items of the previous doc
(30–43), minus the ones marked non-goal.

## 2. What landed

### Item 30: `core.Canvas`, 4.19 and 4.20 on the emulator and simulator

Both lessons ran on `Medium_Phone_API_36.1` and the booted iPhone 17 Pro
simulator (iOS 26.5).

- **What matched Go's geometry on both:**
  - Y labels sit on their gridlines, with a 14px label box on both platforms.
  - The x-label weights split correctly: Jan/Apr/Jul/Oct under a line, and
    Mon/Wed/Fri/Sun under bars.
  - Round-capped zero-length strokes draw as dots.
  - The `#rrggbb33` area tint, the donut gaps and the gauge caps all draw.
  - The analog clock's hands pivot at the centre.
  - Shift and "Use 12%" redraw through update-props patches (the gauge went
    72% → 60%).
- **Bug 1, iOS: SwiftUI's `Canvas` clips to its frame.** htmlout
  (`overflow:visible`) and Compose (no clip) don't. The last point's dot on a
  line chart and the sparkline's end dot were cut in half.
  - Fix in `GrMobCanvas` (`ios/GrMob/Runtime/Renderer.swift`): the Canvas is
    laid out larger by `outset` on every side (negative padding) and the
    drawing is translated back.
  - `outset` is the widest stroke width among the shapes.
  - The drawing has `allowsHitTesting(false)`. The outer box's
    `contentShape(Rectangle())` still takes taps.
- **Bug 2, Android: Compose dropped `Gap` under any non-start `Justify`.** It
  was a documented "known divergence, not attempted because this repo cannot
  build Android". In 4.20 the donut and gauge touched on Android but were
  16px apart on the web and iOS.
  - Fix in `horizontalArrangement` / `verticalArrangement` (`Renderer.kt`).
  - center and flex-end use `Arrangement.spacedBy(gap, alignment)`.
  - The space-* values use a new `GappedDistribution`, an
    `Arrangement.HorizontalOrVertical`. It lays down the gap first, then
    shares the free space as CSS does. RTL mirrors the placement, and
    `spacing` reports the gap for FlowRow.
  - Each arm stays one line, so `TestKotlinArrangementsCoverEveryJustifyContent`
    still parses.
- **Bug 3, Android: Go ran in UTC.** There's no `$TZ` and no `/etc/localtime`
  in an app process. The clocks read 16:47 on a device showing 11:47, and the
  alarm notification said "4:47 PM". Absolute instants were right, which is
  why the alarm still fired on time.
  - New `mobile.SetTimeZone(name, offsetSeconds)` in `mobile/timezone.go`. It
    uses `time.LoadLocation`, falling back to `time.FixedZone`.
  - Called once, before rendering, from both apps' `GomobileBridge` init.
  - Once only because `time.Local` is an unsynchronised global.
  - After the fix, 4.19's clocks matched the status bar.
- **New `ios/GrMobUITests/TutorialChartsUITests.swift`:**
  - Deep-links to 4.20.
  - Checks that each chart is one element labelled with its summary
    ("Traffic: …", "Steps this week: …", "Budget: …", "Battery, 72%").
  - Shift must change the line summary, and "Use 12%" must make the gauge
    read "Battery, 60%".
  - With `TEST_RUNNER_GRMOB_SHOTS_DIR` in xcodebuild's *environment*, it writes
    PNGs. As a `NAME=value` argument it becomes a build setting and never
    arrives.

### Item 37: the TRY IT badge

`demoPanel` gives the badge `core.FlexShrink(0)`.

- **Chrome, 360px wide:** with the fix, all three 4.19 badges are 23px tall.
  Setting `flex-shrink` back to 1 in devtools brings back the bug the item
  described: 42px tall, "TRY IT" on two lines.
- **Android:** stays one line beside 4.7's four-line hint.
- The hints that 4.19 and 4.20 had shortened were left short.

### Item 31: Tier E, alarms that ring with the app closed

Delivery:

```
UseAlarms{Notify} ── lifecycle "background" ──▶ LocalNotification{At} × ≤60
                                                  │ "notification" {…, "at": ms}
      Android  AlarmManager ─▶ NotificationAlarmReceiver ─▶ NotificationCompat
      iOS      UNCalendarNotificationTrigger (local wall clock)
      Browser  per-id setTimeout while the tab is open (capped 2^31−1 steps)
UseAlarms ◀── lifecycle "active" ── cancel all; last = now; OnRing(rang while away)
```

- **`core/notifications.go`:**
  - `LocalNotification.At`: a zero or past time posts now.
  - Wire key `"at"` (Unix milliseconds), sent only when in the future.
  - New section "Scheduled notifications" with each host's promise.
  - Test: `TestPostNotificationSendsAFutureAtOnly`.
- **`hooks/alarms.go`:**
  - `AlarmOptions.Notify`, and `NotifyText` (defaults to Label or "Alarm",
    plus the 12-hour time).
  - The record gains `away`, `scheduled` and `awaySince`.
  - `setAway` / `awayChanged`:
    - **Going away:** silences any ringing alarm, clears the queue, and
      schedules snoozes plus each enabled alarm's occurrences in the next 7
      days. One-time alarms are scheduled once; at most 60 in all; ids are
      `grmob.alarm.<ID>.<unix>`.
    - **Coming back:** cancels those, sets `last = now`, drops past snoozes,
      and returns the alarms that came due while away for `OnRing`.
  - While away with Notify on, `check` neither rings nor queues. This matters
    on Android, where the goroutine keeps running and would otherwise ring
    twice.
  - The lifecycle subscription is always taken, because opts is re-read every
    render.
  - The "In-app only" doc became "Off screen".
  - Tests: `TestAlarmsNotifyScheduleWhileAwayAndCancelOnReturn` and
    `TestAlarmsWithoutNotifyIgnoreTheBackground`, run under `-race`.
- **Android:**
  - `Notifications.kt` schedules through `AlarmManager`.
    - Exact (`setExactAndAllowWhileIdle`) below API 31 or when
      `canScheduleExactAlarms()`; otherwise `setAndAllowWhileIdle`.
    - The PendingIntent is keyed by `id.hashCode()`, with the id as data so
      colliding hashes stay distinct.
    - cancel removes the alarm too.
  - `post()` now takes a Context and creates the channel itself, because the
    receiver may run in a new process with no `attach`.
  - Manifest: `SCHEDULE_EXACT_ALARM`, and a non-exported
    `NotificationAlarmReceiver`.
- **iOS, `Notifications.swift`:** a calendar trigger (year…second) when `at`
  is in the future; a re-post that schedules removes the banner already
  showing.
- **Web, `grmob-runtime.js`:**
  - A timers map. Any command under an id clears its timer.
  - A long wait is taken in capped steps, since `setTimeout` overflows past
    2^31−1 ms.
  - Permission is read when the banner is due.
  - New `wasm/verify/notifications_test.mjs` (5 tests) drives it through the
    loader's timer queue with a shadowed `Date`.
- **Lesson 4.19:** `Notify: true`, `hooks.UsePermission(ctx,
  permission.Notifications)`, an outlined "Allow notifications" button while
  not granted, new prose, and two key points in place of one.
- **Seen working:**
  - **Android:** set "Ring at the next minute" at 11:46:12, pressed Home, ran
    `am kill`, and the process was gone. `dumpsys alarm` showed the
    `NotificationAlarmReceiver` alarm at 11:47:00 with a +35s window
    (inexact). The notification posted at 11:47:36 from a new process, reading
    "Try it · 4:47 PM" (before the time-zone fix).
  - **iOS:** new `TutorialAlarmNotifyUITests` passes in 65s. It taps the
    button, presses Home, waits 3s, terminates the app, then finds a
    SpringBoard element containing "Try it" no earlier than the alarm's
    minute.

### Item 33: a snooze coming back, watched live

On Android, driven through `uiautomator dump` and `input tap`:

- Set at 11:49:28, rang at 11:50:02, snoozed at 11:50:04.
- The panel read "11:50 AM is snoozed until 11:51:02."
- It rang again at 11:51:06.
- The one-time "Try it" row switched itself off through `OnRing`.

## 3. Knock-on edits the repo's tests required

- **Go-file count 541 → 543** (`mobile/timezone.go` and its test):
  `wasm/verify/repowalks_test.go` lines 30, 532, 730 and `timings_test.go`
  lines 823, 892.
- **`ios/verify/gomobile_stub.swift`:** declares
  `MobileSetTimeZone(_:_:)` (`TestTheGomobileStubMatchesTheBoundGoSurface`).
- **`mobile/verify/notifications_test.go`:** the shells must spell `"at"`.
- **`go run ./internal/apidoc/gen`** regenerated `mobile.md`,
  `core-device.md`, `hooks.md` and `index.md`.
- **Plan doc** `ai_docs/plans/clocks-canvas-charts.md`: the status now records
  Tier E and the device findings.

## 4. Verification

- `gofmt -l` is clean, `go vet ./...` is clean, and `go test ./...` passes.
- `node --test wasm/verify/*_test.mjs`: 553 pass.
- `ios/verify/run.sh`: view layer and app layer type-check, WMO, and importer
  all OK.
- `android/verify/run.sh`: compile OK, plus menus, ranges and 7 canvas
  drawings.
- **XCUITests:** `TutorialChartsUITests` and `TutorialAlarmNotifyUITests`
  pass.
- The iOS build with `SetTimeZone` succeeded (`build-for-testing`), but the
  simulator app was not re-run after that change.
- Android `assembleDebug` passed after every change. The final APK is
  installed on the emulator.
- **Chrome:** a scratch WASM build (served from the scratchpad; the tracked
  `wasm/main.wasm` is untouched) was used for the badge measurement only. The
  web scheduling path was tested in Node, not in a real browser.

## 5. Harness notes

- **Gradle needs `ANDROID_HOME`.** `android/build.sh` defaults it for gomobile
  only, so `./gradlew` in the same shell failed with "SDK location not found".
  `echo exit $?` after a subshell hid that on the first try; the APK installed
  then was stale.
- **No touch-injection tool for the simulator** (no idb or cliclick). XCUITest
  was the way to scroll and tap. `simctl terminate` before a first launch
  errors harmlessly.
- **Android UI driving:** `uiautomator dump` plus `grep` on `text=…bounds=` to
  find controls, then `input tap` at the centre. A slow `input swipe` (800 to
  1500 ms) scrolls without flinging.
- **Emulator time zone:** `adb shell getprop persist.sys.timezone` gave
  `America/Chicago`, which is what exposed the UTC bug.
- **Left running:** the emulator (background task), and the simulator stays
  booted. The Chrome tab was closed and the http.server stopped.

## Next

**Legend:** *age* is how many session docs ago the item was first raised,
counted from this doc (0 = raised here; `≥ k` = older than the last rebuild's
window). *value*: **high** = worked around today or a second consumer has
arrived; **medium** = blocks one named thing or is a visible defect; **low** =
nobody has hit the gap yet. Sorted by age, oldest first; new items last.

**Closed or changed this session** (previous numbering):

- **30** (Canvas and charts on the emulator and simulator) is done. It found
  and fixed three bugs: the iOS canvas edge clip, Compose dropping the gap,
  and Go in UTC on Android.
- **31** (Tier E) is done as `LocalNotification.At` + `AlarmOptions.Notify`.
  AlarmKit and `AlarmClock` were not used (see new item 39).
- **33** (snooze coming back watched live) is done on Android.
- **37** (TRY IT badge wraps) is done: `FlexShrink(0)`, checked in Chrome and
  on Android.
- **32** (alarm sound and haptics unheard) is still open. Nothing here can
  hear or feel them.

1. **(age ≥19 · value high) Lessons on hardware, and checks still open.**
   - Real devices for everything (deferred by the user).
   - **Screen readers:**
     - radio and StepIndicator done-step semantics
     - combobox active option
     - "pop-up" on Menu/DatePicker triggers
     - aria-current/selected on current items
     - Calendar's grid on the web (needs a person with VoiceOver)
     - CodeEditor toolbar role on Compose
     - SearchableSelect natives reading field + list + status
     - AccessibilityHidden confining TalkBack/VoiceOver behind a Drawer
     - VoiceOver on the calendar cell now that it is one element
   - **iOS:**
     - sheet `Dialog` and ActionSheet filler
     - Drawer slide by eye and under Reduce Motion
     - RTL for Drawer and CodeEditor
     - editing in the sideways code editor
     - `banded` and a drag on 4.6's list
   - **Pinning and Drawer:**
     - `Screen.Footer` pinning
     - 100% layers in a pinned-height ZStack
     - the panel's `Height 100%` in an HStack
     - `core.Focus` on a Button
     - a hardware keyboard reaching a shut panel
   - **Android:**
     - predictive back from contents
     - MaxWidth where it binds (tablet)
     - a capped stretched child in RTL
2. **(age ≥19 · non-goal) Rename `docs/components.md` to `comps.md`.**
3. **(age ≥19 · non-goal) Rewrite `components` in the older plans.**
4. **(age ≥19 · non-goal) Trim the copied Android shell's permissions.**
5. **(age ≥19 · non-goal) Replace the iOS usage strings further.**
6. **(age ≥19 · non-goal) C4 `Carousel`** until the host reports a scroll
   offset.
7. **(age 18 · non-goal) A parent that gains `onBack` after its descendants
   outranks them** on Android.
8. **(age 18 · non-goal) An AppBar outside the Navigator** with a custom
   `OnBack` is outranked by the Navigator's pop on Android.
9. **(age 17 · non-goal) Forward does not re-open a screen left by browser
   back.**
10. **(age 16 · non-goal) Sticky headers, placement animation and
    `OnEndReached` in a List with no viewport** on Compose.
11. **(age 14 · value low) The cost of a paused TimelineView per node** in
    SwiftUI's box chain. Not profiled.
12. **(age 14 · non-goal) MaxWidth with a growing sibling on the natives.**
13. **(age 14 · non-goal) The typed-hash fold's `history.length` fallback.**
14. **(age 14 · non-goal) A page's own `pushState` while a claim is on
    screen.**
15. **(age 14 · non-goal) A Drawer's shut panel is composed on the natives.**
16. **(age 12 · non-goal) A List with no Height in a scrolled page is not lazy**
    on iOS or Android.
17. **(age 6 · value low) Compose's today `stateDescription` not heard** under
    TalkBack on the emulator.
18. **(age 6 · value medium) F-keys on iOS unverified; key delivery to a
    simulator is unreliable.**
19. **(age 6 · value low) Page-global chords on iOS are verified once.**
20. **(age 5 · non-goal) GameController cannot take a key.**
21. **(age 5 · non-goal) Commits already on a remote carrying an unformatted
    file** are noted, not blocked, except at the tip.
22. **(age 4 · value low) The Compose Row hug is unswept in `comps` widgets'
    own internal Rows.**
23. **(age 3 · value medium) The min-content floor has not been seen on a
    device.** Lessons 1.1 and 8.2 are the screens that should change.
    Emulators and simulators ran this session, but those two lessons weren't
    opened.
24. **(age 3 · value low) `content ÷ N%` arithmetic unwatched** for a Row child
    with a percentage MaxWidth.
25. **(age 3 · value low) A plain horizontal strip in a Row now hugs.** Unseen.
26. **(age 3 · value low) The iOS chord gate is unheard.**
27. **(age 3 · value low) Merging a labelled node is unheard under TalkBack.**
28. **(age 3 · value medium) `.claude/settings.json` cannot be edited from a
    session.** Worth an upstream report.
29. **(age 3 · value low) `android/verify/sources.sh` needs `ANDROID_HOME` and
    a filled gradle cache**, and SKIP reads like a pass. `android/build.sh`
    followed by `./gradlew` has the same `ANDROID_HOME` trap.
30. **(age 2 · value low) Alarm sound and haptics are unheard.**
    - The lesson passes no `Sound`.
    - Ringing replaces whatever core's single audio player was playing and
      doesn't resume it (documented).
    - Haptic pulses were not felt on a device.
    - The Notify banner uses the platform's default sound, also unheard.
31. **(age 2 · non-goal) `DigitalClock` digits can shift width by a pixel.**
    There's no font family in `core.Style`; centred to hide it.
32. **(age 2 · non-goal) `AnalogClock{Smooth}` spins back once if the tick at
    00:00:00 is skipped** (an app suspended over midnight).
33. **(age 2 · non-goal) Canvas v1 omits** text inside the drawing, gradients,
    clipping, per-shape hit-testing, and the even-odd fill rule.
34. **(age 1 · value low) A chart's hidden data table** was not built.
    - The one-sentence summary is all a screen reader gets.
    - A BarChart with more than 8 categories only gives the low and high.
    - Needs a screen-reader-only primitive that core doesn't have.
35. **(age 1 · value low) Chart types not built:** smoothed (monotone cubic)
    lines, stacked areas and bars, horizontal bars, scatter, and value labels
    on bars.
36. **(age 1 · value low) An x label wider than its slot** widens the slot and
    pulls neighbours off their points.
    - A real fix needs a single-line truncating text style on four renderers.
      `WhiteSpace` is web-only, and each target floors a flex slot
      differently: CSS `min-width:auto`, Compose `weight`, and iOS
      `GrMobMinContent`.
37. **(age 1 · non-goal) A 180° `Gauge` leaves its bottom half empty.**
38. **(age 1 · value low) The chart palette has only 3–5 distinct hues** per
    bundled theme.
    - A fix is a new theme role (for example a categorical list on
      `ColorPalette`): a theme-API decision, not a comps change.
39. **(age 1 · value low) Chart summaries are English**, like other widgets'
    spoken strings. `AccessibilityLabel` overrides them.
40. **(age 0 · value medium) Android alarms are inexact unless the user grants
    exact alarms.**
    - API 36 doesn't grant `SCHEDULE_EXACT_ALARM` to a new install; the banner
      came 36s late.
    - An alarm-clock app would declare `USE_EXACT_ALARM` (Play policy), or
      send the user to Settings from a permission flow. The `permission`
      package has no exact-alarm kind.
41. **(age 0 · value medium) The Compose gap fix is unswept across other
    screens.**
    - Every Android Row/Column/FlowRow with a `Gap` and a non-start `Justify`
      now spaces its children (as web and iOS always did).
    - Only 4.20 was looked at. A screen tuned by eye on Android could now
      overflow, for example a `JustifyBetween` header with a Gap.
42. **(age 0 · value low) Scheduled notifications are lost on an Android
    reboot or force stop.** There's no `BOOT_COMPLETED` receiver and no store
    of what was scheduled. `UseAlarms` reschedules only the next time the app
    runs and leaves the screen.
43. **(age 0 · value low) A Notify alarm is a banner, not a ringing screen.**
    - There's no full-screen intent, no looping sound, and no Snooze action on
      the notification.
    - iOS AlarmKit and Android `AlarmClock` / full-screen intents are the
      route to a real system alarm.
44. **(age 0 · value low) Alarms the OS rang are reported to `OnRing` only if
    the app was backgrounded, not killed.** A one-time alarm rung with the
    process dead stays Enabled in an in-memory list. The lesson's list isn't
    persisted anyway.
45. **(age 0 · value low) `mobile.SetTimeZone` runs once at startup.** A
    time-zone change while the app runs is picked up on the next launch,
    because `time.Local` can't be written safely while goroutines read it.
46. **(age 0 · value low) The iOS canvas outset is the widest stroke width.**
    A miter join sharper than about 60° reaches further than that and can
    still clip at the edge. No bundled chart uses a miter at an edge.
47. **(age 0 · value low) `TutorialAlarmNotifyUITests`' "Allow" branch is
    unconfirmed.**
    - It passed, but whether the SpringBoard permission alert was answered by
      the test or had already been granted in the simulator is unknown.
    - A clean simulator (`simctl privacy reset`) run would show it.
48. **(age 0 · value low) The web's scheduled notification is unseen in a real
    browser** (Node tests only). It needs a granted permission in Chrome and
    a tab left open.
49. **(age 0 · value low) The iOS build with `SetTimeZone` wasn't run on the
    simulator.** It built for testing, but the clocks weren't re-checked on
    iOS; iOS already had `/etc/localtime`.
