# A Mi Max 3 on Android 10: force dark, twice, and a first-screen sweep

**Session:** e2755825-d484-4675-b023-d64fa86c62c5
**Date:** 2026-09-18 23:10 (follows "android-device-pass-fold6-secure-field-weight")
**Branch:** master (e2b6296 → 79a505b, then this doc)

## The ask

1. `/sl`: load the 22:03 doc.
2. "I have an android device connected. Pls try to comm with it."
3. Build and install; fix the broken colours; commit and push.
4. "Do the steps that need an Android device," then "go ahead without
   input" once MIUI's input lock turned out to need a SIM.
5. `@RequiresApi` on the Haptics helpers; commit and push.
6. Keep the phone awake. `/sw`.

**The device:**
- Xiaomi Mi Max 3 (`nitrogen`), MIUI on Android 10 (API 29), serial
  `420edad5`. First real device below API 33.
- 1080×2160 px at 440 dpi: 393×785 dp.
- No SIM ("Emergency calls only"). Wi-Fi only.
- Charges as **AC**, not USB, which matters for `svc power stayon usb`.
- The emulator `emulator-5554` (API 36) was attached too, so every command
  set `ANDROID_SERIAL`.
- It first showed as `unauthorized`; the user accepted the USB-debugging
  prompt.

## 1. Force dark, first fix (6b8760d)

**Symptom.** The tutorial's home screen drew a dark page with light-grey
Cards, grey text lightened almost to invisible, and the lesson titles
("Hello, GrMob") gone entirely.

**Cause.** MIUI's system dark mode sets `debug.hwui.force_dark=true`. The
app's theme was `@android:style/Theme.Material.Light.NoActionBar` and the
runtime has no night-mode handling at all, so HWUI's force dark re-tinted
Compose's draw calls by heuristic.
- Proved by `setprop debug.hwui.force_dark false`, relaunch, screenshot,
  then `setprop … true` again: with it off the app drew correctly.
- A first capture right after `am start -W` was blank: `-W` returns at the
  splash window, and the mount takes ~3.9s on this Snapdragon 636. Waits of
  7s+ went into `run_in_background` scripts.

**Fix.** `MainActivity.onCreate`: `window.decorView.isForceDarkAllowed =
false` on API 29+, with a comment.

## 2. Force dark, the real fix (79a505b)

**Symptom.** The first sweep (§4) showed every lesson from 4.13 on with
faded "‹ Contents" and near-invisible text inside Cards (4.21's readouts:
"393 × 785", "compact", …). Stable over 12s, so not a transition.

**Isolation.**
- The emulator drew 4.21 correctly.
- A cold start straight into 4.21 drew correctly.
- Cold → 4.11 → 4.21: correct. Cold → 4.12 → 4.21: **faded.** 4.12 is the
  live map: osmdroid's MapView inside an `AndroidView`.
- Cold → 4.12 → 4.21 with `debug.hwui.force_dark false`: correct. So it was
  force dark leaking back past the decor-view flag.

**Why the flag leaked (not fully diagnosed).** `View.setForceDarkAllowed`
marks a render node; HWUI honours it while walking the render node tree.
Once the AndroidView was attached, that walk stopped honouring it. The theme
attribute is read by `ViewRootImpl.updateForceDarkMode` to decide whether
the window's renderer does force dark at all, so there is no tree walk to
escape.

**Fix.**
- New `android/app/src/main/res/values/themes.xml`: `Theme.GrMob`, the same
  platform light Material parent plus `android:forceDarkAllowed=false`
  (`tools:targetApi="q"`). The first file under `res/values`.
- The manifest names `@style/Theme.GrMob`, with a comment.
- The decor-view line stays (no-removal rule). Its comment now says the
  theme is the primary switch and why the flag alone was not enough.
- Verified: force dark on, cold → 4.12 → 4.21 draws correctly, and the
  re-sweep of all 75 lessons is clean.

## 3. Lint: Haptics false positives (6347a02)

- `./gradlew :app:lintDebug` fails. It reported 11 `NewApi` errors and 8
  `InlinedApi` warnings, all in `Haptics.kt`.
- They are false positives. `handle()` dispatches on `SDK_INT` before
  calling `predefined` (29+), `shaped` (26+) and `legacy`, but lint follows a
  guard only inside the function that holds it.
- `@RequiresApi(Q)` on `predefined` and `@RequiresApi(O)` on `shaped` and
  `waveform`, with a comment. `androidx.annotation` resolves transitively.
- After the change Haptics has 0 lint issues. `lintDebug` still fails on 4
  errors that were already there (Next, item 66).

## 4. The first-screen sweep on Android 10

**MIUI locks.** Without "USB debugging (Security settings)", MIUI refuses:
- Injected input: `SecurityException: Injecting to another application
  requires INJECT_EVENTS permission`. A swipe landing off-screen raises no
  error, which briefly looked like success.
- `settings put secure|global …` (`WRITE_SECURE_SETTINGS`), so TalkBack
  can't be enabled and `debug.force_rtl` can't be set.
- `wm size` (same permission).
- `settings put system …` works, which is how rotation was locked.

The user turned the switch on, but `persist.security.adbinput` stayed empty
and injection stayed refused, even after `adb reconnect`. MIUI needs a SIM
(and a Mi account) for that switch, and the phone has none.

**What works without input:** `am start` deep links, `screencap`,
uiautomator dumps (visible nodes only), logcat, dumpsys, force-stop and
`setprop debug.*`.

**The sweep.** `grmob://lesson/<id>` for all 75 lessons (chapters 1–8:
5, 6, 5, 33, 8, 8, 5, 5), 2.5s wait, screenshot, the resumed activity
recorded, and `logcat *:E` alongside. Contact sheets were built with
`montage -label '%t' … -tile 6x1 -geometry 360x720+6+4`.
- **First run:** found §2 (4.13 onwards faded).
- **Re-run on 79a505b:** all 75 lessons stayed on `MainActivity` and drew
  normally on their first screen. The only app-process log lines were
  zygote's `Unknown bits set in runtime_flags` and Qualcomm's `Perf` noise.
- **5.4 sat ~15px high in one capture.** Two re-captures were aligned. The
  user had touched the screen to keep it awake.
- **4.14's three-column toolbar grid** matches the emulator. By design.
- **Item 58 seen again:** the RichTextEditor's "•a bullet".

**Screenshots are dimmed to 75%** (white reads 191) by
`MiuiContrastOverlay`, a system ColorLayer at the top of SurfaceFlinger's
stack in dark mode. It covers every app, so it is not GrMob's.

**TalkBack.** This phone has **Google** TalkBack 15.2 (plus the 8.1 system
copy), not Samsung's. It is the route item 62 wanted, but it's blocked on
the same MIUI lock.

## 5. State left on the phone

- `stay_on_while_plugged_in` = **7** (was 0), set via `svc power stayon
  true`. `usb` alone (2) did nothing because the phone charges as AC.
  Restore with `adb -s 420edad5 shell svc power stayon false`.
- Auto-rotate **off**, user rotation 0 (portrait). It was on.
- `debug.hwui.force_dark` is back to `true` (MIUI's own value) after each
  probe.
- Accessibility untouched (`enabled_accessibility_services` empty, as
  before).
- GrMob's debug build (79a505b) is installed. The emulator has it too.

## Pitfalls

- **`am start -W` on MIUI** can report `LaunchState: UNKNOWN (-1)` after a
  reinstall. The launch still happens.
- **`grep -vc` in a background task** exits 1 when the count is 0, so the
  task reads "failed" when every lesson passed.
- **A zsh `for`** over `--include=*.go` globs fails with "no matches found".
  Quote the globs.
- **The phone's owner was using it during runs:** Control Center, Settings,
  a rotation. Check `mResumedActivity` before judging a screenshot.
- **Pixel sampling scripts** at guessed coordinates cost more than a
  cropped montage viewed by eye.

## Files

- `android/app/src/main/java/com/grmob/app/MainActivity.kt`: the decor-view
  opt-out and its comment.
- `android/app/src/main/res/values/themes.xml`: new, `Theme.GrMob`.
- `android/app/src/main/AndroidManifest.xml`: the theme reference and its
  comment.
- `android/app/src/main/java/com/grmob/app/Haptics.kt`: `@RequiresApi`.

## Verification

- `android/verify/sources.sh` and `android/verify/run.sh` are OK after each
  change.
- Go is untouched, so `go test` was not re-run.
- `lintDebug`: Haptics clean; 4 older errors remain.
- **On the Mi Max 3 with force dark on:**
  - The home screen and 4.21-after-4.12 draw correctly.
  - The 75-lesson re-sweep is clean.

## Next

**Legend:** *age* is how many session docs ago the item was first raised,
counted from this doc (0 = raised here; `≥ k` = older than the 25-doc
window the 20:02/22:03 rebuild used). *value* is the payoff, not the
effort:
- **high:** worked around today, or a second consumer has arrived.
- **medium:** blocks one named thing, or is a visible defect.
- **low:** nobody has hit the gap yet.

Items are sorted by age, oldest first, then by value. `lapsed@<doc>` marks
an item that fell off the list without being done; the doc named is where
it went missing. Numbering carries on from the 22:03 doc.

**Closed this session:** none of 1–64. Force dark (6b8760d, 79a505b) and
the Haptics lint (6347a02) were new and are done. Progress on 33/61 (an
Android 10 first-screen sweep) and on 62 (a Google-TalkBack phone found) is
recorded in those items.

1. **(age ≥26 · value medium · lapsed@0917-1659) Lessons on hardware, and
   checks still open.**
   - Screen readers: radio and StepIndicator semantics, combobox active
     option, "pop-up" triggers, aria-current, Calendar's grid, CodeEditor
     toolbar role on Compose, SearchableSelect on the natives,
     AccessibilityHidden behind a Drawer.
   - iOS: sheet Dialog and ActionSheet filler, Drawer under Reduce Motion,
     RTL for Drawer and CodeEditor, the sideways editor, 4.6's list.
   - Pinning and Drawer: `Screen.Footer`, 100% layers in a pinned ZStack,
     `core.Focus` on a Button, a hardware keyboard reaching a shut panel.
   - Android: predictive back, MaxWidth where it binds on a tablet, RTL
     capped child.
   - Devices: the Fold6 (Android 16); the Mi Max 3 (Android 10, input
     locked without a SIM, so no RTL and no TalkBack from adb).
   - Lessons by feature: 4.9 Calendar, 4.13 CodeEditor, 4.16 radio and
     steps, 4.17 menus and SearchableSelect, 4.18 Drawer.
2. **(age ≥26 · value medium · lapsed@0917-1659) `.claude/settings.json`
   cannot be edited from a session** (`[Self-Modification]`). Worth an
   upstream report. Not re-checked since 2026-09-15.
3. **(age ≥26 · value low) F-keys through GameController have never reached
   the app from XCUITest.** `testFunctionKeyPressesTheButton` is a strict
   expected failure. Needs a real iPad keyboard.
4. **(age ≥26 · value low · lapsed@0917-1659) Compose's today
   `stateDescription` is not heard.**
   - Blocked on the Fold6 (Samsung TalkBack) and on the Mi Max 3 (MIUI's
     lock). See item 62.
   - Cheapest now is a person listening: TalkBack on, 4.9, touch day 11,
     listen for "today". The alternative is the emulator's method from
     2026-0918-1310.
5. **(age ≥26 · value low · lapsed@0917-1659) Merging a labelled node is
   unheard under TalkBack** (a labelled container holding two controls).
   Same blocker and same route as item 4; they land together.
6. **(age ≥26 · value low · lapsed@0917-1659) iOS chords.** Page-global
   chords were verified once, and the chord gate (behind a modal, inside a
   shut Drawer panel) is unheard. `primeKeyboard` makes simulator delivery
   dependable enough to retry.
7. **(age ≥26 · value low · lapsed@0917-1659) The cost of a paused
   TimelineView per node** in `GrMobMotion`. Not profiled.
8. **(age ≥26 · non-goal · lapsed@0917-1659)**
   - Rename `docs/components.md` to `comps.md`; rewrite `components` in
     the older plans.
   - Trim the Android shell's permissions; the iOS usage strings.
   - C4 `Carousel` until the host reports a scroll offset. Item 64 declines
     that offset, so this is effectively permanent.
   - Android `onBack` ranking (a late parent; an AppBar outside the
     Navigator).
   - Forward after browser back.
   - Sticky headers or `OnEndReached` in a List with no viewport on
     Compose.
   - MaxWidth with a growing sibling.
   - The typed-hash `history.length` fallback; a page's own `pushState`
     during a claim.
   - A Drawer's shut panel is composed on the natives.
   - A List with no Height is not lazy.
   - Unformatted commits already on a remote.
9. **(age 25 · value low · lapsed@0917-1659) Alarm sound and haptics are
   unheard,** and so is the Notify banner's default sound. The Fold6 or the
   Mi Max 3 can do it with a person holding it.
10. **(age 25 · value low · lapsed@0917-1659) Canvas still omits** text,
    clipping and per-shape hit-testing.
11. **(age 25 · non-goal · lapsed@0917-1659)** `DigitalClock` digits
    shifting by a pixel; `AnalogClock{Smooth}` spinning back when the
    midnight tick is skipped.
12. **(age 24 · value low · lapsed@0917-1659) A chart's hidden data table**
    was not built. It needs a screen-reader-only primitive: an API
    decision.
13. **(age 24 · value low · lapsed@0917-1659) Chart summaries are
    English.**
14. **(age 24 · non-goal · lapsed@0917-1659)** A 180° `Gauge` leaves its
    bottom half empty.
15. **(age 23 · value low · lapsed@0917-1659) A Notify alarm is a banner,
    not a ringing screen.** The route is AlarmKit or full-screen intents.
16. **(age 23 · value low · lapsed@0917-1659) `mobile.SetTimeZone` runs
    once at startup.** Fixing it needs core to hold the location: an API
    decision.
17. **(age 23 · value low · lapsed@0917-1659) The web's scheduled
    notification and its sweep are unseen in a real browser.**
18. **(age 22 · value low · user's decision · lapsed@0917-1659) Compose
    Rows don't shrink children in proportion.** Seen live on the Fold6's
    369dp cover screen: 4.19's "Allow exact alarms" row wraps its caption
    one or two words per line beside a full-width button.
19. **(age 21 · value low · lapsed@0917-1659)
    `testRelaunchSweepsWhatTheDeadProcessScheduled` needs notifications
    already granted.**
20. **(age 21 · value low · delete?) The double-post claim is
    unreproduced.** Carried ten times; proposed for deletion.
21. **(age 21 · non-goal · lapsed@0917-1659)** `MaxLines` on the natives
    applies to Text only; a stacked chart counts NaN as 0.
22. **(age 20 · value low · lapsed@0917-1659) `DefaultDarkChartColors` has
    no bundled consumer.** Related to item 67: a real dark theme would be
    its consumer.
23. **(age 20 · value low · delete?) `TutorialChartsUITests` failed once,
    reason not captured.** Never recurred; proposed for deletion.
24. **(age 20 · non-goal · lapsed@0917-1659)** `core.LinearGradient` is
    unused; kept per the no-removal rule.
25. **(age 19 · value low · lapsed@0917-1659) The alarm changes: what is
    left unseen.**
    - Still open: a re-run of `TutorialAlarmNotifyUITests` (iOS relaunch).
    - Still open: whether `OnRing` switches a one-off alarm off after a
      force-stop relaunch. The Fold6 run couldn't tell (the lesson state
      reset), and the Mi Max 3 can't tap.
26. **(age 19 · value medium · blocked · lapsed@0917-1659) The iOS Image
    floor runs high for a narrow image.** Unblocking it needs a px-width
    box that can shrink (`grMobDimension`).
27. **(age 19 · value low → non-goal? · lapsed@0917-1659) A zero basis is
    honoured on iOS only with a definite main extent.** Documented as
    deliberate; proposed as a non-goal.
28. **(age 19 · non-goal · lapsed@0917-1659)** Renaming a `NotifyGroup`
    strands its schedules.
29. **(age 18 · value medium · lapsed@0917-1659) An iOS Image with no
    background shows a black letterbox** on a "fit" image. Presumably open;
    unverified on 26.5.
30. **(age 18 · value low · lapsed@0917-1659) `barValueRoom` is still an
    estimate.**
31. **(age 18 · value low · lapsed@0917-1659) Square scatter dots are
    unseen in Chrome.** Compose was seen on the Fold6.
32. **(age 18 · value low · lapsed@0917-1659) `testStatTilesShareTheRow`
    asserts weakly.**
33. **(age 18 · value low · lapsed@0917-1659) Comps on Android in
    landscape are unswept.**
    - Chapter 4 was swept at 707dp portrait on the Fold6.
    - All 75 lessons were swept at 393dp on Android 10 (Mi Max 3), first
      screen only.
    - Landscape at 823dp was seen for 4.21 alone.
34. **(age 18 · value low → non-goal? · lapsed@0917-1659) Sparkline's
    `Area` is a flat tint.** Left alone on purpose; proposed as a non-goal.
35. **(age 17 · value medium · lapsed@0917-1659) Safe-area insets are not a
    record.** A TwoPane under the status bar can't find its `Origin.Y`.
    `core/` has no `SafeInsets`.
36. **(age 17 · value low · lapsed@0917-1659) Lesson 4.21's TwoPane sets no
    `Origin`.** On the Fold6 in book posture the split lands at 417dp,
    63dp right of the 354dp hinge. Harmless: the list is narrow.
37. **(age 17 · value low · lapsed@0917-1659) iOS `AppWindowReader` is
    type-checked only.** Not run in Split View or Stage Manager.
38. **(age 17 · value low · lapsed@0917-1659) The browser's
    segments/posture path is unseen in a real browser.**
39. **(age 17 · value low · lapsed@0917-1659) Folding shut onto the outer
    display.** At the Fold6's defaults the app goes to the background. The
    display switch needs Samsung's "Continue apps on cover screen", unseen.
40. **(age 17 · value low · lapsed@0917-1659) Housekeeping.**
    - The `GrMob_Foldable` AVD is still installed; less needed now.
    - The Mi Max 3's state from this session (§5): `stay_on_while_plugged_in`
      is 7 (was 0) and auto-rotate is off (was on).
41. **(age 17 · non-goal · lapsed@0917-1659)** More than one fold; a static
    HTML export of a TwoPane.
42. **(age 16 · value medium · lapsed@0917-2309) The round-two widgets have
    never been seen on iOS.** Screen.Floating and the FAB (4.22), the
    QRCode seams, and Countdown/Stopwatch (4.24). All verified on Android.
43. **(age 12 · value medium) Examples should adopt the shipped widgets.**
    - `examples/chat` hand-builds what `comps.MessageThread` does
      (`main.go:148`).
    - `examples/signup` never got `PINInput`.
    - Both change shotclaims, so they land together.
44. **(age 10 · non-goal)** A native time wheel; a sheet or Done on
    TimePicker.
45. **(age 9 · non-goal)** Heatmap as a continuous gradient; a Sequential
    ramp interpolated from `Primary`.
46. **(age 8 · value low · delete?) A sixth low-hanging-fruit round.** No
    candidates across six carries; proposed for deletion until one exists.
47. **(age 8 · non-goal)** A year-wide scrolling `CalendarHeatmap`; a
    caption-flip "Copied ✓" on CopyButton; a separate `Alert` widget (Banner
    is it).
48. **(age 7 · non-goal)** A two-pane layout on the natives; restructuring
    lesson bodies into guide and demo halves.
49. **(age 6 · non-goal)** Making iOS keep focus on every submit by
    default.
50. **(age 5 · value medium) The first hardware key after launch is lost on
    the iOS 26.5 simulator.** Every keyboard UI test works around it with
    `primeKeyboard`. Unchecked on a real iPad.
51. **(age 5 · non-goal)** A right-padding gutter or a toolbar header for
    code blocks.
52. **(age 4 · non-goal)** A smaller minimum for Buttons in general on
    Android. Only CopyButton opts out.
53. **(age 3 · value low) iOS UIKit field: a queued key during the caret
    correction.** Not observed in 6 correction writes. Candidate fix: drain
    through `input.inputDelegate` `selectionWillChange/DidChange` before
    `replace`, then `rebaseEdit(current, flushed, next)`.
54. **(age 2 · value low) `core.ScrollIntoView` inside an iOS
    `core.List`.** `GrMobList`'s `ScrollViewReader` does not inject
    `\.grMobScrollProxy`; rows are keyed by `rowKey` while
    `GrMobBringIntoView` scrolls by `viewID`. Measure before claiming it.
55. **(age 2 · value low) iOS Paragraph link colours on the iOS 17 floor.**
    Measured only on 26.5; the `.tint(firstLink)` fallback remains.
56. **(age 2 · value low · by design) The web's thread place-keeping
    applies only when the List is its own scroll box.**
57. **(age 1 · value medium) CodeEditor's `tabSize` is honoured on the web
    only.** A literal tab draws about one space wide on Compose (4.13's
    seed). Compose needs a display transformation with an offset mapping so
    caret and selection stay in buffer units. iOS is unmeasured.
58. **(age 1 · value low) The Android RichTextEditor draws list bullets
    without a gap** ("•a bullet"); the read-only RichTextView draws "• a
    bullet". Seen again this session on the emulator's 4.14.
59. **(age 1 · value low · contingent) The secure-field wrapper in
    `Renderer.kt` can go once the Compose BOM reaches foundation 1.10.**
    Contingent on a BOM bump that isn't planned.
60. **(age 1 · value low) PasswordField's width on iOS is unmeasured.** One
    look at 4.27 on the simulator settles it.
61. **(age 1 · value low) Other chapters at tablet width.** Only chapter 4
    was swept unfolded on the Fold6. Lands with item 33. The Mi Max 3 sweep
    was phone width, first screen only.
62. **(age 1 · value low) An Android TalkBack route that works on a real
    phone.**
    - Samsung (Fold6): "Display speech output" plus screenshots works, but
      TalkBack's first-run tutorial and phone-permission prompt take focus
      on every enable.
    - **Xiaomi (Mi Max 3): Google TalkBack 15.2 is installed**, but MIUI
      refuses `settings put secure` and injected input without "USB
      debugging (Security settings)", which needs a SIM.
    - A SIM plus a Mi account on the Mi Max 3 is the likely cheapest unlock.
      It would also unlock RTL (`debug.force_rtl`), taps and scrolling.
    - Unlocks items 4 and 5.
63. **(age 1 · non-goal)** Continuing apps onto the cover screen by
    default. That is a Samsung system setting, not the app's.
64. **(age 2 · non-goal)** A reported scroll offset from the hosts. The
    thread needed only the two List props (see `core/list_start.go`).
65. **(age 0 · value low) A below-the-fold sweep on Android 10.** The Mi
    Max 3 saw each lesson's first screen only. Demos further down (the 4.19
    permission rows on API 29, where notifications need no runtime grant and
    exact alarms no permission) are unseen. Needs item 62's unlock, or a
    person scrolling.
66. **(age 0 · value low) `lintDebug` still fails** on errors that
    predate this session:
    - 3 × `RestrictedApi` on `ComponentActivity.dispatchKeyEvent`
      (`MainActivity.kt:103–105`).
    - 1 × `PermissionImpliesUnsupportedChromeOsHardware`: CAMERA wants
      `<uses-feature android:name="android.hardware.camera"
      android:required="false"/>`.
    - Nothing runs lint today. Once it's clean it could join
      `android/verify`.
67. **(age 0 · value low · API decision) The Android app never follows the
    system's dark mode.** It is always light: force dark is now refused,
    and nothing reports night mode to Go. A real dark theme needs the host
    to send `uiMode` night as an event and core to pick a theme (item 22's
    `DefaultDarkChartColors` would be a consumer). iOS and the web are
    unchecked on the same question.
68. **(age 0 · value low) Why the decor-view force-dark flag stopped
    holding after an AndroidView attached is undiagnosed.** The theme
    switch makes it moot for this app. It would matter for any shell that
    keeps a platform theme. The redundant decor-view line in `MainActivity`
    could go (kept per the no-removal rule).
69. **(age 0 · non-goal)** MIUI's `MiuiContrastOverlay` dims every
    screenshot to 75% in dark mode. It's the system's, not the app's.
    Compare colours only relative to each other on this phone.

Read by value instead:
- **high:** none.
- **medium:** 1, 2, 26, 29, 35, 42, 43, 50, 57.
- **low:** 3–7, 9, 10, 12, 13, 15–20, 22, 23, 25, 27, 30–34, 36–40, 46,
  53–56, 58–62, 65–68.
- **non-goal:** 8, 11, 14, 21, 24, 28, 41, 44, 45, 47–49, 51, 52, 63, 64,
  69.
