# Next list: exact alarms, re-arming after a reboot, and two device sweeps

**Session:** 997261ed-9e24-48f1-b5b0-2e26e1a1a630
**Date:** 2026-09-16 12:29 (follows "next-list-charts-on-devices-tier-e-alarms-timezone")
**Branch:** master (4302fee → this commit)

## 1. The ask

"Keep going through the Next list doing what we can." Taken as every item that
code, the emulator (`Medium_Phone_API_36.1`) or the booted iPhone 17 Pro
simulator could close, skipping items that need hardware, a person listening,
or an API decision.

## 2. What landed

### Item 41: the Compose gap fix, swept

- **The census:** grep for `Justify(Center|End|Between|Around|Evenly)` in
  `comps` and `examples` gives 34 sites. Only these also carry a non-zero
  `Gap`:
  - `comps/dialog.go` footer (JustifyEnd)
  - `comps/rich_text_editor.go` link prompt (JustifyEnd)
  - `comps/paging.go` Pagination (Between) and LoadMore (Center)
  - lesson 1.4's Justify demo
  - 4.19 and 4.20 (already seen)
- **LoadMore can't be affected.** Every state renders one child, except Err,
  where a `FlexGrow(1)` box takes the free space.
- **On the emulator:**
  - **6.6 Dialog:** Keep and Delete sit at the trailing edge with the SM gap.
  - **4.6 Pagination:** "‹ Newer" at the start, the label between, "Older ›"
    at the end.
  - **1.4, the boxes checked by arithmetic.** Boxes are padded 10/18/26dp
    (76.5/117.5/160.5 px), the gap is 21 px and the inner width 792 px.
    - Evenly: CSS predicts the letters at 268.9, 486.5 and 744.7; the emulator
      measured 269, 487 and 745.
    - Center: CSS predicts A at 367.75; the emulator measured 369.
- **Found on the way (new item):** 4.6's DataTable rows drift by a few px. The
  Date column has no `Weight`, so its cell hugs "Mar 1" vs "Mar 22". That is
  the documented semantics and the same on the web.

### Item 23: the min-content floor, seen

- **1.1:** "Following" stays on one line.
- **8.2:** "Repair (set B = A)" wraps only between words.
- **Also visible in 8.2 (new item):** "Advance both counters" keeps one line
  while Repair takes three. That is the documented no-proportional-shrink
  divergence (`Renderer.kt`, pinMainAxis doc), not the floor.

### Item 46: the iOS canvas outset and the miter limit

- SwiftUI's `StrokeStyle.miterLimit` defaults to **10**. SVG's
  `stroke-miterlimit` and Compose's `Stroke.DefaultMiter` are **4**, so iOS
  drew sharp-corner spikes up to 2.5× longer than the other targets.
- **`Renderer.swift`:**
  - New `grMobCanvasMiterLimit = 4`, passed to every stroke.
  - The outset is now `w·limit/2 = 2w` for any stroked shape with a miter
    join, and `w` for round or bevel.
  - The stale doc line "SwiftUI's Canvas draws unclipped by default" was
    corrected.
- **`core/canvas.go`:** the `LineJoin` doc states the limit of 4 on every
  target.
- The line chart's end dot on 4.20 is still whole on the simulator.

### Item 29: the `ANDROID_HOME` trap and SKIP-as-pass

- **`sources.sh` already defaulted the SDK.** The remaining trap was
  `android/build.sh` followed by `./gradlew` in the caller's shell.
- **`build.sh`** now writes `android/local.properties`
  (`sdk.dir=$ANDROID_HOME`) when the file is absent and the SDK exists. The
  file is gitignored and is AGP's own per-checkout answer.
- **Seen:** `env -u ANDROID_HOME ./gradlew --offline assembleDebug` succeeded
  after it.
- **`android/verify/gate.sh`** gained `skipped()`. It prints the SKIP line and,
  under `GRMOB_VERIFY_STRICT=1`, exits **3** ("could not check", distinct from
  1 = "found a fault").
  - `run.sh` (both census SKIPs, both JVM-harness SKIPs) and `sources.sh`
    (compile skip, com.grmob.app skip) use it.
  - In `sources.sh`, gradle's error output is printed before the skip, so
    strict mode doesn't swallow it.
- **Seen:** a strict run with `PATH=/usr/bin:/bin` exited 3; the same run
  without strict exited 0. A normal run still shows every stage OK.

### Item 40: `permission.ExactAlarms`

- **`permission/permission.go`:** `ExactAlarms Permission = "exact_alarms"`,
  added to `Permissions()`.
  - The spelling uses an underscore because the browser test requires an
    unquoted JS key (`exact_alarms: `).
  - The doc gives each host's answer and why this isn't folded into
    Notifications: two switches, and either can be off.
  - It points at `hooks.UsePermissionLive`, not `UsePermission`.
- **Android, `Permissions.kt`:**
  - Map entry `"exact_alarms" to emptyArray()`.
  - `exactAlarmStatus`: below 31 granted; from 31 `canScheduleExactAlarms()`
    decides granted or denied, and the answer is never prompt.
  - `request` opens `Settings.ACTION_REQUEST_SCHEDULE_EXACT_ALARM` with
    `package:` and sends the current status at once. The grant is read by the
    foreground re-check.
- **iOS, `Permissions.swift`:** check and request both send "granted".
- **Web, `grmob-runtime.js`:** `DESCRIPTORS.exact_alarms: null`, and `query`
  answers "granted" for it.
- **Lesson 4.19:**
  - `exactStatus := hooks.UsePermissionLive(ctx, permission.ExactAlarms)`.
  - An outlined "Allow exact alarms" button with the caption "Without it
    Android may ring a minute late.", shown only when Denied (so Unknown never
    flashes it).
- **The first attempt used `UsePermission`.** Coming back from Settings left
  the button up, because that hook checks only on mount. A cold start showed
  the grant, which pinned the cause.
- **Seen on the emulator:**
  - The button tapped through to Settings (`com.android.settings/.spa.SpaActivity`,
    "Allow setting alarms and reminders" for GrMob).
  - After switching it on and pressing back, the button was gone.
  - "Ring at the next minute" then scheduled with `window=0
    exactAllowReason=permission` and posted at 12:18:00.050.
- **Seen on the simulator:** the new UI test asserts the button never appears
  on iOS.

### Item 42: scheduled notifications survive a reboot, update or force stop

```
schedule ──▶ store[id] = {title, body, at}   fire / cancel / post-now ──▶ remove
BOOT_COMPLETED ─┐
package update ─┼─▶ rearm: at > now ─▶ schedule again (same PendingIntent)
attach (launch) ┘         at ≤ now ─▶ post now, remove ("late, not never")
```

- **`Notifications.kt`:**
  - The store is its own SharedPreferences file,
    `grmob-scheduled-notifications`, keyed by the Go id with a JSON value.
  - `remember` runs before the alarm is set. `forget` runs on cancel, on fire
    (`postScheduled`) and on an immediate post.
  - `rearm(context)` is internal and is also called from `attach`.
- **A behaviour fix along the way:** an immediate post under an id now cancels
  a still-pending alarm under that id. Before, it could fire a second banner
  later, which contradicts core's "posting again under the same ID replaces
  the pending request" and iOS's same-identifier add.
- **New `NotificationBootReceiver`:** exported, and filters BOOT_COMPLETED and
  MY_PACKAGE_REPLACED, checking the action.
- **Manifest:** `RECEIVE_BOOT_COMPLETED` and the receiver.
- **`core/notifications.go`:** the Android row now describes the re-arming,
  replacing "Pending alarms do not survive a reboot or a force stop".
- **Seen on the emulator:**
  - "Wake up · Weekdays" was enabled and "Ring at the next minute" set; Home
    left 6 entries in the store and 6 alarms.
  - `adb reboot` at 12:18:49; booted by 12:19:41.
  - The boot broadcast arrived ~2 minutes later (~12:22). Then five 06:30
    alarms were back (`window=0`), and soon-2 (due 12:19) was posted late and
    removed from the store.
  - Force stop cleared the alarms to 0; launching the app brought back 5.
- **Clean-up:** the store file was deleted with `run-as` and the app
  force-stopped, leaving 0 grmob alarms.
- **New item:** alarms from a previous process can't be cancelled by a new one
  (see Next).

### Items 49 and 47: iOS

- **New `ios/GrMobUITests/TutorialClockLocalTimeUITests.swift`:**
  - Opens 4.19 and finds the DigitalClock by its label regex
    `^h:mm:ss (AM|PM)`.
  - Compares hour and minute to the simulator's `Calendar.current`, with ±1
    minute tolerance.
  - Scrolls to the alarm demo and asserts there is no "Allow exact alarms"
    button.
  - Writes screenshots under `TEST_RUNNER_GRMOB_SHOTS_DIR`.
  - Passed (17 s), alongside `TutorialChartsUITests` (19 s).
- **47:** `simctl uninstall booted com.grmob.demo`, then
  `TutorialAlarmNotifyUITests` passed in 47 s. The xcresult activities (parsed
  from `xcresulttool get test-results activities` with python) show Tap "Allow
  notifications", then Tap SpringBoard "Allow", then Tap "Ring at the next
  minute". So the Allow branch is live on a clean install.
- `ios/build.sh ./examples/tutorial` and `xcodegen generate` came first; the
  pbxproj is untracked.

### Knock-on

- `docs/api/{permission,core-device,core-controls}.md` were regenerated
  (`go run ./internal/apidoc/gen`).
- **Plan doc** `ai_docs/plans/clocks-canvas-charts.md`: the Tier E bullet
  records ExactAlarms and the re-arm, with what was seen.

## 3. Verification

- `gofmt -l` clean on the changed files; `go vet ./...` clean;
  `go test ./...` passes.
- `node --test wasm/verify/*_test.mjs`: 553 pass, 0 fail.
- `ios/verify/run.sh`: all OK, including the view layer, WMO and the app
  layer.
- `android/verify/run.sh`: all OK, including com.grmob.runtime and
  com.grmob.app compiled with the Compose plugin; the strict and lenient exits
  were checked as above.
- `android/build.sh ./examples/tutorial` plus `assembleDebug`, installed; a
  final APK with the Live hook is on the emulator.
- **XCUITests:** `TutorialChartsUITests`, `TutorialClockLocalTimeUITests` and
  `TutorialAlarmNotifyUITests` (clean install) all pass.

## 4. Harness notes

- **`scratchpad/ad.sh`** is an adb helper: `dump`, `find TEXT`, `tap TEXT`
  (centre of the first matching bounds), `swipe DY` (a 1 s swipe), `shot NAME`,
  `lesson ID` (the `grmob://lesson/ID` VIEW intent). uiautomator bounds are
  quoted (`"[x,y][x,y]"`), so a regex anchored with `]$` never matches.
- **Revoking an app op needs both modes:** `appops set --uid com.grmob.app
  SCHEDULE_EXACT_ALARM default` as well as the package mode. The package mode
  alone left "Uid mode: allow".
- **BOOT_COMPLETED on the API 36 emulator arrived ~2 minutes after
  `sys.boot_completed=1`.** Poll with an `until` loop; chained sleeps are
  blocked by the harness.
- **zsh has no `PIPESTATUS`**, so check the APK mtime (`stat -f %Sm`) to
  confirm a fresh build.
- **No PIL in python3**, so screenshots can't be cropped; read the full PNG.
- **`xcresulttool` activity titles** contain escaped quotes; parse the JSON
  rather than grep it.
- **Left running:** the emulator and the booted simulator. The Android APK and
  the simulator app both carry this session's build.

## Next

**Legend:** *age* is how many session docs ago the item was first raised,
counted from this doc (0 = raised here; `≥ k` = older than the last rebuild's
window). *value*: **high** = worked around today or a second consumer has
arrived; **medium** = blocks one named thing or is a visible defect; **low** =
nobody has hit the gap yet. Sorted by age, oldest first; new items last.

**Closed this session** (previous numbering):

- **23** (min-content floor on a device) is seen on 1.1 and 8.2.
- **29** (`ANDROID_HOME` / SKIP reads like a pass): `build.sh` writes
  `local.properties`, and `GRMOB_VERIFY_STRICT` makes a skip exit 3.
- **40** (Android alarms inexact unless granted) is done as
  `permission.ExactAlarms` plus 4.19's Live-checked button.
- **41** (Compose gap fix unswept) is swept: six sites, all corrections toward
  the web.
- **42** (lost on reboot or force stop) is done: a store, a boot receiver and a
  re-arm at attach.
- **46** (iOS miter outset) is done: miter limit pinned to 4, outset 2w.
- **47** (the "Allow" branch unconfirmed) is confirmed on a clean install.
- **49** (iOS SetTimeZone unchecked) is done by
  `TutorialClockLocalTimeUITests`.

1. **(age ≥20 · value high) Lessons on hardware, and checks still open.**
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
2. **(age ≥20 · non-goal) Rename `docs/components.md` to `comps.md`.**
3. **(age ≥20 · non-goal) Rewrite `components` in the older plans.**
4. **(age ≥20 · non-goal) Trim the copied Android shell's permissions.**
5. **(age ≥20 · non-goal) Replace the iOS usage strings further.**
6. **(age ≥20 · non-goal) C4 `Carousel`** until the host reports a scroll
   offset.
7. **(age 19 · non-goal) A parent that gains `onBack` after its descendants
   outranks them** on Android.
8. **(age 19 · non-goal) An AppBar outside the Navigator** with a custom
   `OnBack` is outranked by the Navigator's pop on Android.
9. **(age 18 · non-goal) Forward does not re-open a screen left by browser
   back.**
10. **(age 17 · non-goal) Sticky headers, placement animation and
    `OnEndReached` in a List with no viewport** on Compose.
11. **(age 15 · value low) The cost of a paused TimelineView per node** in
    SwiftUI's box chain. Not profiled.
12. **(age 15 · non-goal) MaxWidth with a growing sibling on the natives.**
13. **(age 15 · non-goal) The typed-hash fold's `history.length` fallback.**
14. **(age 15 · non-goal) A page's own `pushState` while a claim is on
    screen.**
15. **(age 15 · non-goal) A Drawer's shut panel is composed on the natives.**
16. **(age 13 · non-goal) A List with no Height in a scrolled page is not lazy**
    on iOS or Android.
17. **(age 7 · value low) Compose's today `stateDescription` not heard** under
    TalkBack on the emulator.
18. **(age 7 · value medium) F-keys on iOS unverified; key delivery to a
    simulator is unreliable.**
19. **(age 7 · value low) Page-global chords on iOS are verified once.**
20. **(age 6 · non-goal) GameController cannot take a key.**
21. **(age 6 · non-goal) Commits already on a remote carrying an unformatted
    file** are noted, not blocked, except at the tip.
22. **(age 5 · value low) The Compose Row hug is unswept in `comps` widgets'
    own internal Rows.** This session's gap sweep looked at the Justify+Gap
    rows only, not the hug.
23. **(age 4 · value low) `content ÷ N%` arithmetic unwatched** for a Row child
    with a percentage MaxWidth.
24. **(age 4 · value low) A plain horizontal strip in a Row now hugs.** Unseen.
25. **(age 4 · value low) The iOS chord gate is unheard.**
26. **(age 4 · value low) Merging a labelled node is unheard under TalkBack.**
27. **(age 4 · value medium) `.claude/settings.json` cannot be edited from a
    session.** Worth an upstream report.
28. **(age 3 · value low) Alarm sound and haptics are unheard.**
    - The lesson passes no `Sound`.
    - Ringing replaces whatever core's single audio player was playing and
      doesn't resume it (documented).
    - Haptic pulses were not felt on a device.
    - The Notify banner uses the platform's default sound, also unheard.
29. **(age 3 · non-goal) `DigitalClock` digits can shift width by a pixel.**
    There's no font family in `core.Style`; centred to hide it.
30. **(age 3 · non-goal) `AnalogClock{Smooth}` spins back once if the tick at
    00:00:00 is skipped** (an app suspended over midnight).
31. **(age 3 · non-goal) Canvas v1 omits** text inside the drawing, gradients,
    clipping, per-shape hit-testing, and the even-odd fill rule.
32. **(age 2 · value low) A chart's hidden data table** was not built.
    - The one-sentence summary is all a screen reader gets.
    - A BarChart with more than 8 categories only gives the low and high.
    - Needs a screen-reader-only primitive that core doesn't have.
33. **(age 2 · value low) Chart types not built:** smoothed (monotone cubic)
    lines, stacked areas and bars, horizontal bars, scatter, and value labels
    on bars.
34. **(age 2 · value low) An x label wider than its slot** widens the slot and
    pulls neighbours off their points.
    - A real fix needs a single-line truncating text style on four renderers.
      `WhiteSpace` is web-only, and each target floors a flex slot
      differently: CSS `min-width:auto`, Compose `weight`, and iOS
      `GrMobMinContent`.
35. **(age 2 · non-goal) A 180° `Gauge` leaves its bottom half empty.**
36. **(age 2 · value low) The chart palette has only 3–5 distinct hues** per
    bundled theme.
    - A fix is a new theme role (for example a categorical list on
      `ColorPalette`): a theme-API decision, not a comps change.
37. **(age 2 · value low) Chart summaries are English**, like other widgets'
    spoken strings. `AccessibilityLabel` overrides them.
38. **(age 1 · value low) A Notify alarm is a banner, not a ringing screen.**
    - There's no full-screen intent, no looping sound, and no Snooze action on
      the notification.
    - iOS AlarmKit and Android `AlarmClock` / full-screen intents are the
      route to a real system alarm.
39. **(age 1 · value low) Alarms the OS rang are reported to `OnRing` only if
    the app was backgrounded, not killed.** A one-time alarm rung with the
    process dead stays Enabled in an in-memory list. The lesson's list isn't
    persisted anyway. See item 43, the other half of the same process-death
    gap.
40. **(age 1 · value low) `mobile.SetTimeZone` runs once at startup.** A
    time-zone change while the app runs is picked up on the next launch,
    because `time.Local` can't be written safely while goroutines read it.
41. **(age 1 · value low) The web's scheduled notification is unseen in a real
    browser** (Node tests only).
    - It needs a granted permission in Chrome and a tab left open.
    - Chrome's permission prompt is browser UI the automation can't click, so
      a person has to grant it once.
42. **(age 1 · value low) The 4.19 "Allow notifications" button uses
    `UsePermission`**, so a user who grants notifications in Settings rather
    than through the dialog comes back to a stale button.
    - Only the in-app dialog path refreshes it.
    - `UsePermissionLive` is the fix if that path matters; the exact-alarm
      button already uses it.
    - Raised by this session's diagnosis, but it has been true since 4.19
      gained the button.
43. **(age 0 · value medium) OS alarms from a previous process can't be
    cancelled.**
    - `UseAlarms` cancels only the ids in its in-memory `scheduled` list. After
      the process dies, a relaunched app never cancels the old
      `grmob.alarm.<ID>.<unix>` notifications, even for an alarm now off or
      deleted.
    - Android's store now re-arms them after a reboot or force stop, so they
      last longer than before; iOS keeps pending requests natively.
    - The fix is a "cancel by id prefix" command on each host, or a persisted
      list of scheduled ids.
44. **(age 0 · value low) Compose Rows don't shrink children in proportion.**
    - 8.2's "Advance both counters" keeps one line and Repair takes three;
      CSS squeezes both.
    - Documented in `Renderer.kt` (pinMainAxis) and pinned by
      `internal/pinfixture`.
    - Recorded here because it now shows on a lesson screen.
45. **(age 0 · value low) DataTable columns without a `Weight` drift from row
    to row** (4.6's Date column: "Mar 1" vs "Mar 22").
    - Same on every target: a zero-weight column hugs its content by design.
    - Aligning it needs a width or min-width option on `Column`, which is an
      API decision.
46. **(age 0 · value low) `NotificationBootReceiver`'s re-arm can post a missed
    banner twice.**
    - If `attach` re-arms (and posts) a just-due entry while its alarm is also
      being delivered, both paths post.
    - The shared tag makes the second replace the first, but it may alert
      again.
    - Not seen; the window is the moment of launch.
47. **(age 0 · value low) Android's exact-alarm grant is only read on
    foreground.**
    - Revoking "Alarms & reminders" kills the app process (platform behaviour),
      so no in-process update is needed for that direction.
    - Granting from Settings without returning through the app is picked up at
      the next launch or resume.
    - `ACTION_SCHEDULE_EXACT_ALARM_PERMISSION_STATE_CHANGED` isn't listened
      for.
48. **(age 0 · value low) `GRMOB_VERIFY_STRICT` exists only in
    `android/verify`.** `ios/verify` has the same skip-means-pass stance and
    no strict switch.
