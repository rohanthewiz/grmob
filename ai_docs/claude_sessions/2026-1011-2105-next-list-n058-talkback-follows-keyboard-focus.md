# Next list: N-058, TalkBack follows keyboard focus onto off-screen controls

Session: `295761ac-3863-4037-a805-b9b22bbeca3b`
**Date:** 2026-10-11 21:05 · **Branch:** master (3016412 → 394bf92, plus this doc)

## Ask

1. Work N-058 from `ai_docs/todo/next-list.md`, pasted from the user's
   backlog: "TalkBack does not follow Tab onto 4.18's ☰." It blocked the
   emulator half of N-062's Poll, N-068's TreeView and Wizard checks, and
   N-072's TalkBack pass.
2. `/sess-wrap`.

**Device:** the API 36 emulator (`emulator-5554`), Google TalkBack
16.0.0.777931756. The Fold6 was not attached. No grmob peer sessions were
running.

## 1. Reproducing it

- The harness as in memory `emulator-talkback-harness`: TTS disabled so each
  utterance logs, TalkBack enabled by settings, keys from `input keyevent`.
- A temporary diagnostic in `MainActivity` (reverted) logged each key, the
  focused View, and every accessibility event Compose sent. An
  AccessibilityDelegate on `android.R.id.content` sees each one in
  `onRequestSendAccessibilityEvent`.
- **Compose's own order** (TalkBack off, `uiautomator dump`, the node with
  `focused="true"`) on 4.18: ‹ Contents → Copy → ☰ (Open navigation) →
  ‹ Prev → Next ›. That is correct.
- **Under TalkBack:** Compose sent TYPE_VIEW_FOCUSED for Contents, Copy and
  Next, and nothing for ☰ and ‹ Prev. Those two were the ones Tab had to
  scroll up from below the fold. TalkBack then read prose ("12 notes in this
  section") that no Tab visits.

## 2. Cause, read in Compose 1.7.6's bytecode

- `ui-android-1.7.6`'s `AndroidComposeViewAccessibilityDelegateCompat`
  (javap on the Gradle cache's AAR):
  - `sendSemanticsPropertyChangeEvents` sends TYPE_VIEW_FOCUSED (8) when
    `SemanticsProperties.Focused` turns true, comparing each node with its
    copy from the previous 100ms batch.
  - A node with no previous copy is skipped. The maps hold only nodes in the
    visible region. So an off-screen control that a Tab scrolls in is never
    announced.
- Two more ways TalkBack lost the keyboard, found later in the session:
  - **Ignored mid-scroll.** 4.34's poll options sat just past the edge.
    Compose did send the event, but TalkBack did nothing with it.
  - **Scroll cut short.** On 4.9 a Tab from the date field to ‹ Prev
    stopped scrolling at 7879 of 8624. TalkBack moved its own focus to
    visible prose as its node left the screen, which interrupted Compose's
    bring-into-view and left ‹ Prev off screen. Its semantics bounds read
    zero because `boundsInRoot` is clipped to the viewport.

## 3. Fix: `GrMobFocusAnnouncer.kt` (`394bf92`)

- `AnnounceFocusMovesToScreenReader()` runs in `GrMobRoot` and inside
  `GrMobModal`'s Dialog (its own window and Compose view).
- It sets an AccessibilityDelegate on the Compose view's parent, so it sees
  every batch. Any delegate already there is kept (API 29+).
- **Gate:** touch exploration on, and the view out of touch mode. That means
  a screen reader driven from a keyboard.
- After each batch it finds the node with Focused (via public
  `RootForTest.semanticsOwner`) and holds it as the target until TalkBack's
  focus is on it. "TalkBack's focus" is asked of Compose's provider:
  `findFocus(FOCUS_ACCESSIBILITY) == createAccessibilityNodeInfo(id)`.
  - **Off screen once scrolling settles:** scroll it on through the nearest
    scroller's `SemanticsActions.ScrollBy`. This is Compose's own
    ACTION_SHOW_ON_SCREEN math (its `scrollDelta`, with reverseScrolling and
    RTL flips), which can't be invoked for a node outside Compose's map.
  - **Settled (300ms quiet), TalkBack elsewhere:** send the
    TYPE_VIEW_FOCUSED that Compose's `createEvent` would build, and retry
    once 1.2s later.
  - At most two scrolls and two events per target; a target lapses after 3s
    without progress.
- **Cost:** the semantics walk measured 60µs (median) on 4.34. That page
  animates and sends a batch about every 100ms.
- **Retry timing:** the first design retried at 600ms. A Modal opened from
  the keyboard ignored both events while TalkBack settled the new window.
  At 1.2s, the retry lands.
- **Pinned:** `mobile/verify/focus_announce_test.go`. Lint caught a
  `getAccessibilityDelegate` (API 29) call in `detach`; it is now gated.

## 4. Heard after the fix (emulator, TalkBack 16)

- **4.18:** Copy → Open navigation → ‹ Prev → Next › → ‹ Contents. Enter on
  ☰ gives "Notebook", "Close Notebook"; Escape returns to "Open navigation".
- **4.34:** every Tab speaks its control, including the poll's "Tabs",
  "Spaces" and "Whatever gofmt says". Three of four full walks spoke ‹ Prev;
  one did not.
- **4.16:** Copy ×3, Next, "Advance the order", ‹ Prev, Next ›; Shift+Tab
  back too.
- **4.9:** Copy, ‹ Prev and Next › after the date field. Each was first
  scrolled on by the announcer ("dy=429", "dy=682"), then spoken.
- **4.35:**
  - TreeView's src, assets and README.md are now spoken.
  - With a name typed, Tab spoke "Next, Button" and Enter pressed it.
    TalkBack then said "Step 2 of 3, optional".
  - On step 2: Tab → "Step 1: Your name, done" → note field → Back → Skip.
    Enter on Skip gave "Step 3 of 3".
  - The disabled Next is skipped by Tab with TalkBack off as well. That is
    correct.
- **4.9's DatePicker Modal**, opened by Enter: "Service date", then "Clear,
  Button", where Compose put focus. Tab speaks Close; Escape returns to
  "Choose a date".
- **4.37's grid by Tab:** "Row 1. Opens the row's menu", "Item, row 1, Rent",
  "Category, row 1, Home", "Amount, row 1, $1200.00", "Paid, row 1,
  checked", then row 2. Moving to row 2 scrolls the grid sideways, and
  TalkBack first says "Amount, row 2", then "Row 2…".
- **TalkBack off:** the Tab order on 4.18 is unchanged.
- **Still heard:** when TalkBack's node scrolls away, TalkBack may read what
  it moved to before the correction. After a long scroll the correction can
  take over 2s.

## Pitfalls

- **The emulator was asleep:** a black screencap, and "App is in stopped
  state" in logcat. `KEYCODE_WAKEUP` plus `wm dismiss-keyguard` fixed it.
  `svc power stayon` was set for the session and set back to false.
- **INSTALL_FAILED_INSUFFICIENT_STORAGE:** /data was at 6% free, under
  Android's min(10%, 500MB) floor. The space is other projects' apps, which
  were left alone. `sys_storage_threshold_percentage 3` and
  `sys_storage_threshold_max_bytes 150000000` let the install through. Both
  were unset before and were deleted after.
- **Utterances log as several `text:` fragments.** The first parser kept
  only the first ("Selected" for "Selected, thumbs up, 3 reactions,
  Button").
- **Counted Tabs drift between launches.** TalkBack sometimes puts its focus
  (and Compose's) on ‹ Contents at launch, so the first Tab is silent and
  correct. Navigate by what is spoken, not by a count.
- **Adding a Go file moves wasm/verify's file-count prose** (701 → 702),
  which the test asks to be edited.
- Memory `emulator-talkback-harness` was updated with the fragments, the
  ground-truth method, and the storage and sleep fixes.

## Next

Closed: N-058. Declined: None. Raised: None.
Deferred: None. Promoted: None. Moved: None.
Updated: N-062, N-068, N-072, N-101. Full list: `ai_docs/todo/next-list.md`.
