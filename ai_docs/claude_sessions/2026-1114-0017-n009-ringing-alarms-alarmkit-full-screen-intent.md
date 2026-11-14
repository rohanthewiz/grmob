# N-009: ringing alarms — AlarmKit and full-screen intents, opt-in per app

Session: `9aaae143-064c-4d12-8de4-3dba70af422b`

## 1. The item

N-009: "A Notify alarm is a banner, not a ringing screen. The route is AlarmKit
or full-screen intents." A 2026-10-10 session had parked it on a permission
and Play-policy decision.

The user decided both points at the start:

- **Scope:** both platforms. Android gets a full-screen intent that opens the
  app's own ringing screen over the lock screen. iOS 26+ gets AlarmKit, and
  iOS 17–25 keep the banner.
- **Permissions:** opt-in per app. `"alarms": true` in `grmob.json` makes
  `grmob android` / `grmob ios` keep the policy-gated declarations. An app
  without it carries nothing new, and its alarms stay banners.

## 2. Platform facts that shaped it

- **Android `USE_FULL_SCREEN_INTENT`:**
  - On 14+ (API 34) it is granted at install only to apps whose Play listing
    declares an alarm-clock or calling core function. Others get a Settings
    switch (`ACTION_MANAGE_APP_USE_FULL_SCREEN_INTENT`).
  - A sideloaded debug build had it granted on the API 36 emulators.
  - The shell targets SDK 34, so `canUseFullScreenIntent()` is available.
- **AlarmKit (Xcode 26.6, iOS 26.5 SDK):**
  - The API was read from the SDK's `.swiftinterface`.
  - `AlarmPresentation.Alert(title:secondaryButton:secondaryButtonBehavior:)`
    is **26.1+**. 26.0 needs the `stopButton:` initializer, which is
    deprecated in 26.1. `Alarms.alert` branches on this.
  - `schedule` is async and `cancel` is not.
  - Authorization needs `NSAlarmKitUsageDescription`.
- **Weak linking:** AlarmKit is weak-linked automatically
  (`LC_LOAD_WEAK_DYLIB` in `GrMobApp.debug.dylib`). The iOS 17 deployment
  target still launches, and `OTHER_LDFLAGS` needed no change.
- **AlarmKit snooze:** `.countdown` with a postAlert duration re-rings after
  the snooze without a widget extension (seen on the simulator). No
  countdown presentation (a Live Activity) is supplied.

## 3. Design

### Go

- **`core.LocalNotification.Alarm *NotificationAlarm{Snooze, RingFor}`:**
  - Wire: `"alarm": {snooze?, ring_for?}` in ms, rounded to the second.
    Absent means a banner.
  - Every host that can't ring an alarm (not opted in, not granted, iOS < 26,
    browser) posts the ordinary banner, so callers never ask what the host
    can do.
- **`core.OnRingingAlarmTap(fn)`:** the same `notification_tap` event, with
  `"ringing": true`. Android sets the flag when the tapped or full-screen
  alarm is still inside its `ring_for`. This keeps a tap on a stale
  missed-alarm banner from starting a ring.
- **`core.AlarmAnswered(id)`:** wire `{command: "answered", id}`. It tells the
  host a ring the app took over is done. On Android, the lock screen then
  covers the app again.
- **`permission.Alarms` (`"alarms"`):**
  - iOS 26+: AlarmManager authorization.
  - Android: the full-screen-intent permission.
  - Unavailable when not opted in, on iOS < 26, and in the browser.
  - The browser runtime short-circuits `request` to "unavailable"; it had no
    `prompt()` arm, so without that a request would never have been
    answered.
- **`hooks.UseAlarms`:**
  - Every Notify notification carries
    `Alarm{Snooze: opts.snooze(), RingFor: opts.ringFor()}`.
  - At mount it subscribes to `OnRingingAlarmTap`. `takeOver(id)` finds the
    alarm by id (enabled or not, since OnRing may have just switched off a
    once-alarm), rings it in-app, or queues it behind the one ringing, then
    `CancelNotification(id)`.
  - `takeovers` maps alarm ID to notification id. `silence()` calls
    `AlarmAnswered` for a taken-over ring. `answerTakeovers()` answers
    queued ones dropped by leaving the screen or unmounting.
  - OnRing is not called again; the return to the foreground or the sweep
    already reported the ring.

### Android (`Notifications.kt`, `MainActivity.kt`, `Permissions.kt`)

- **Opt-in check:** `alarmsDeclared` reads the package's requested
  permissions. The manifest line is the whole switch.
- **`postAlarm`:**
  - `grmob.alarms` channel: alarm stream, default alarm tone,
    `IMPORTANCE_HIGH`.
  - `CATEGORY_ALARM`, `FLAG_INSISTENT`, `setTimeoutAfter(ring_for)`,
    `setOngoing`, `setAutoCancel`.
  - Snooze and Stop actions go to the new `NotificationActionReceiver`
    (declared, not exported).
  - A full-screen intent is set when `canUseFullScreenIntent`.
- **Snooze** reschedules the **same id** at now + snooze. The store keeps the
  first ring as `"rang"`, so a sweep still reports the alarm fired. The
  store also keeps `"alarm"`, so a re-arm after reboot posts an alarm.
  `postScheduled` reads the alarm object from the claimed entry.
- **MainActivity:**
  - Puts `ringing` on the tap payload (`tapRinging`: rangAt + ringFor still
    in the future).
  - Sets `setShowWhenLocked` and `setTurnScreenOn` (window flags below 27)
    only for that alarm.
  - Registers a release closure with `Notifications.showingOverLockScreen`.
    "answered" or Stop releases it.
  - A configuration change keeps the claim (`activityGone(changing)`,
    re-applied in `onCreate`).
- One log line each for going over the lock screen and coming back under it.

### iOS (`Alarms.swift` new, `Notifications.swift`, `Permissions.swift`)

- **`Alarms.usable`:** iOS 26+, the usage key, and `.authorized`. It never
  prompts at post time.
- **Ids:** the Go id is mapped to a UUID by SHA-256 (version-8 stamp), so
  cancel and re-post need no table.
- **Ordering:** every AlarmKit call goes through one serial `Task` chain, so
  a cancel cannot lose to an earlier in-flight schedule.
- **Commands:**
  - post: removes any banner under the id, records it for the sweep, then
    `Alarms.schedule` (`.fixed(at)`; Snooze `.countdown` when snooze > 0).
  - cancel: also calls `Alarms.cancel`.
  - sweep: also calls `Alarms.sweep`, which leaves an `.alerting` alarm
    alone, as a shown banner is left.
- **`ring_for`** is unused; AlarmKit has no such setting. Core's doc says so.

### grmob CLI

- `appConfig.Alarms` (`json:"alarms,omitempty"`).
- **Android:** `androidPatches` removes the `<!-- permission.Alarms` comment
  and the `USE_FULL_SCREEN_INTENT` line unless opted in.
- **iOS:** `iosPatches` either rewrites `NSAlarmKitUsageDescription` to name
  the app ("X rings the alarms you set, even when your phone is locked or
  silenced."), or removes it with its comment.
- Tested by `TestShellPatchesFollowTheAlarmsOptIn`.

### Tutorial 4.19

- A live `permission.Alarms` row: "Ring on the lock screen", shown on Prompt
  or Denied.
- A key point added.

## 4. Verified

- **iOS 26.5 simulator (iPhone 17 Pro):** `TutorialAlarmKitUITests` passed
  in 164 s.
  - AlarmKit dialog ("Allow GrMobApp to schedule alarms and timers?", with
    the usage string).
  - The alarm "Try it" with zzz and ✕ appeared with the app terminated.
  - Snooze silenced it, and it re-rang a minute later; Stop ended it.
- **Android API 36 `GrMob_Foldable` emulator, locked:**
  - Needed `locksettings set-disabled false`: both AVDs ship with the lock
    screen disabled, which made the first run on Medium_Phone prove
    nothing.
  - At the minute the device woke and MainActivity showed over the keyguard
    (`isKeyguardShowing=true`, occluded). UseAlarms took over: the app's
    AlarmRinging panel was showing, and the notification was cancelled.
  - Dismiss logged "answered; back under the lock screen", and
    `mKeyguardOccluded=false`.
- **Same emulator, unlocked:**
  - A heads-up "Try it" with Snooze and Stop, on channel `grmob.alarms`,
    flags `ONGOING_EVENT|INSISTENT|AUTO_CANCEL`, with a fullscreenIntent.
  - Snooze took it down and set an exact RTC_WAKEUP alarm +1 min; it re-rang
    under the same tag. Stop, from the expanded shade, cleared it.
- **Tests:**
  - New: `TestAlarmNotificationWireShape`,
    `TestOnRingingAlarmTapHearsOnlyRingingTaps`,
    `TestAlarmsNotifyPostsAlarms`, `TestAlarmsTakeOverARingingTap`,
    `TestUseAlarmsHearsRingingTaps`, `TestAlarmsAnswerTakeovers`,
    `TestShellPatchesFollowTheAlarmsOptIn`,
    `TestAlarmNotificationSpellingsAgree`,
    `TestTheAndroidAlarmIsWiredThroughTheShell`.
  - The iOS usage-key test now includes AlarmKit.
  - `go test ./...` is green except the item below; `-race` on hooks and
    core is green.
- **Failing before this session:**
  `TestComposeDoesNotRepeatWhatALabelledGroupAlreadySaid` fails at HEAD
  762584b, confirmed with the changes stashed. N-101 changed `Renderer.kt`
  without the pinned strings. Raised as N-151.

## 5. Environment notes

- **Medium_Phone_API_36.1:** /data is 98% full. "Requested internal only, but
  not enough space" even after uninstalling `com.grmob.app`, which is
  therefore **no longer installed there**. The other apps' data was left
  alone. Use `GrMob_Foldable` (710 MB free).
- **Lock screens:** both AVDs' `locksettings` were put back to disabled, and
  both emulators and the simulator were shut down afterwards.
- **Tapping by text:** match the exact text. A "contains" match on "Dismiss"
  hit the code sample (`ringer.Dismiss`).

## Next

Closed: N-009. Declined: None. Raised: N-148, N-149, N-150, N-151.
Deferred: None. Promoted: None. Moved: None.
Updated: None. Full list: `ai_docs/todo/next-list.md`.
