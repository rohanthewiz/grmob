# N-150: a missed-alarm banner, and where this run stopped

Session: `94ba3d17-9a13-41bc-96af-d1aadbbafc95`

The last batch of an unattended `/work-next-list` run. The earlier docs are
`2026-1114-1650-…`, `2026-1114-1702-…` and `2026-1114-1736-…`. The third of
these was first committed as `…-1745-…`; it was renamed to the time of its
commit so the docs sort in order, and the list's references were updated.

## 1. N-150: a ringing alarm that times out leaves a record (`e25c7cb`, `6064326`)

- **The change, in `android/…/app/Notifications.kt`:** `postAlarm` gives the
  ringing notification a delete intent (`ACTION_RANG_OUT`, keyed by action
  in the data like Snooze and Stop). It carries the id, title, body, rang-at
  and ring_for.
- **When the system sends it.** The timeout from `setTimeoutAfter` sends it.
  The app's own cancels do not: Snooze, Stop and Go's takeover. Nor does a
  tap, which `autoCancel` handles.
- **`onRangOut`:**
  - It posts a silent banner (`setSilent`, on the plain `grmob` channel)
    under the same id: the title, and "Missed alarm · <body>".
  - It skips the banner if the delete came before ring_for had passed. From
    Android 14 a user can swipe an ongoing notification, and someone who
    swiped a ringing alarm did not miss it.
  - It also calls `leaveLockScreen`.
- **What a tap on the banner does.** The tap carries no rang-at extra, so it
  reports `ringing` false, and `core.OnRingingAlarmTap` ignores it.
- **Seen** on the API 36 foldable emulator, using a temporary
  `RingFor: 25 * time.Second` on lesson 4.19 (reverted, then the tutorial
  reinstalled from HEAD):
  - "Ring at the next minute", then Home.
  - At 17:43:00 the shade held "Try it" on `grmob.alarms`, with flags
    ONGOING|INSISTENT.
  - At 17:43:25 the same tag held "Try it / Missed alarm · 5:43 PM" on
    `grmob`, with flags AUTO_CANCEL|SILENT.
- `android/verify/lint.sh` is clean. core's alarm table in `notifications.go`
  mentions the banner.
- **Slip:** the first commit edited a doc comment in core without
  regenerating `docs/api`, so `internal/apidoc`'s staleness test was red
  for one commit. `6064326` regenerated the docs, and the full suite was
  green after it.
- The emulator was shut down afterwards (`adb emu kill`; it was not running
  when the run began).

## 2. Run summary

- **Closed:**
  - N-151: group-echo pins.
  - N-138: P1 one pass per event, P2 generated `Style.Equal` with lazy
    paths, P3 registry purged in place.
  - N-144: CI watched.
  - N-146: a harness artifact.
  - N-150: the missed-alarm banner.
- **Progressed, still Open:**
  - N-139: P7 watcher, P8a/d highlight cache, P8c form memo. P6 and P9
    remain, and P8b is a decision.
  - N-141: C3 `go tool grmob` and `grmob version`.
  - N-143: D3, D5, D6 and D7. Only the user's `ai_docs/` decision is left.
- **Validate, progressed by an emulator TalkBack pass:** N-062, N-070,
  N-072, N-114, N-123, N-126.
- **Raised:** N-152 (Validate) and N-153 (Open).
- **Doable but not reached:**
  - N-149: snoozes across a relaunch. Go, Kotlin and Swift, plus device
    runs.
  - N-134: the Android process-scoped runtime, a large refactor.
  - N-147: the XCUITest tree for Discussion headers.
  - N-153.
  - The rest of N-139 and N-141.

## Next

Closed: N-150. Declined: None. Raised: None. Deferred: None. Promoted: None.
Moved: None. Updated: None. Full list: `ai_docs/todo/next-list.md`.
