# Next list, part 4: N-097 backup restore, N-028 rotation, N-103 iOS insets

Session: `1e5d7494-420f-4e59-bc6e-2acf4044d880`
**Date:** 2026-10-10 20:27 · **Branch:** master (fc46eb6 → 7145cec, plus this doc)

## Ask

The standing ask from part 1: do everything in the Next list that does not
need the user, commit and push per item, and wrap every two or three items.
This is the fourth wrap.

## 1. N-097: a real Auto Backup restore on the emulator

- **The question.** The keystore session simulated a lost key by renaming the
  alias. N-097 asked for a real `bmgr` restore instead. The app's manifest has
  `allowBackup="true"` and no rules, so Auto Backup does carry
  `grmob_keystore.xml`, and the AndroidKeyStore key never travels with it.
- **The probe.** `examples/keystoreprobe`, deleted after and never
  committed, ran on mount: Get "probe", Save a timestamped value, Get again,
  logging each result to logcat.
- **The sequence:**
  1. Record the app's grants: POST_NOTIFICATIONS granted, SCHEDULE_EXACT_ALARM
     allow, Backup Manager disabled on the GMS transport.
  2. Install the probe and launch it: found=false, the save took about 2.5s
     (key generation), then found=true.
  3. `bmgr enable true`, `bmgr transport …LocalTransport`, and `bmgr
     backupnow com.grmob.app`: Success.
  4. `adb uninstall`, then `adb install` of the same APK. The install
     restored `shared_prefs/grmob_keystore.xml` by itself, holding one
     sealed `probe` entry.
  5. Launch. `GrMobKeystore` logged "no key for 1 sealed entries (restored
     from a backup?); discarding them". Get answered found=false with no
     error. Save and the reread succeeded under a fresh key.
- **Restored afterwards:** `bmgr wipe` of the local transport,
  `bmgr transport` back to GMS, `bmgr enable false`, the tutorial reinstalled
  (`android/build.sh ./examples/tutorial` and `installDebug`), the two
  permissions granted again, and the probe directory removed.
- N-097 stays in Validate for the hardware-backed key (Fold6, Mi Max 3) and
  the iOS launch before first unlock.

## 2. N-028: the iOS window record under rotation, which found N-103

- **Lesson 4.21** prints an Insets line under Window ("top T, bottom B, left
  L, right R", physical edges). Its demo already invites the reader to
  "rotate the device and watch every line change".
- **`TutorialWindowUITests.testTheWindowRecordFollowsARotation`** is new. It
  rotates to landscape-left and back, waiting for the record to change each
  time, and puts the device back upright in setUp and tearDown, because
  rotation persists on the simulator.
- **The first run:**
  - The size followed (402 × 874 to 874 × 402).
  - The insets read **0, 0, 0, 0** in both orientations. After a 3s wait,
    still zero.

### N-103: the cause and the fix

- **Cause.** AppWindowReader read the insets from the size reader's
  `geo.safeAreaInsets`. That reader ignores the safe area so it can measure
  the whole window, and `.ignoresSafeArea()` consumes the very insets the
  proxy would report. The file's doc said the opposite. A reader that
  respects the safe area cannot measure the window, so one SwiftUI reader
  cannot give both numbers.
- **Fix.** `GrMobWindowInsetsProbe` is a clear, untouchable, full-window
  UIView in the reader's background.
  - It reads `window.safeAreaInsets` in `didMoveToWindow`,
    `safeAreaInsetsDidChange` and `layoutSubviews`, which a rotation always
    brings since the view spans the window.
  - It dedupes, and posts to SwiftUI state on the next turn of the main
    loop.
  - The reader reports when either the size or the insets change. UIKit's
    window figure leaves the keyboard out, which matches Android's
    `safeDrawing` minus the IME.
- **After:**
  - portrait: top 62, bottom 34, left 0, right 0;
  - landscape: top 0, bottom 20, left 62, right 62;
  - back to portrait: the portrait record.
  - Safari's insets on the same simulator were 0/62/20/62 (N-088).
- **Robustness.** At launch the size and the insets reach Go in separate
  reports, so the test waits for a non-zero top before taking the portrait
  record. It passed twice, alongside TutorialNativeFloorsUITests and
  TutorialFooterStripUITests.
- **Pins and docs.** `mobile/verify/window_test.go` pins
  `let bars = window.safeAreaInsets` and refuses `geo.safeAreaInsets` in
  AppWindow.swift. `core.SafeInsets`' doc names the new iOS source.
- **Android data point.** The emulator's 4.21 readouts go 411 × 914 to
  914 × 411, with top 24 and bottom 24 both ways. The image has no cutout.
  Rotation was set with `settings put system user_rotation` and then
  restored to 0. `accelerometer_rotation` was set to 0 without reading it
  first; 0 is the image's default.

Commit `7145cec`.

## What went wrong

- **The emulator kept going to sleep.** Black screencaps and dumps of an old
  screen. It needed `KEYCODE_WAKEUP` and `wm dismiss-keyguard` before each
  look.
- **The first deep link after `installDebug` does not navigate.** It
  happened twice. A second cold start works, as
  [[android-tutorial-install]] says.
- **`accelerometer_rotation` was overwritten before being read.** It is
  likely harmless (0 is the default), but the order should have been read,
  then write.

## Files touched

- `ios/GrMob/App/AppWindow.swift`
- `ios/GrMobUITests/TutorialWindowUITests.swift` (new)
- `examples/tutorial/chapter4.go` (4.21's Insets line)
- `mobile/verify/window_test.go`
- `core/window.go`, `docs/api/core-device.md`
- `ai_docs/todo/next-list.md`, this doc

## Next

Closed: N-103. Declined: None. Raised: N-103. Deferred: None. Promoted: None.
Moved: None. Updated: N-028, N-097. Full list: `ai_docs/todo/next-list.md`.
