# Android device pass on a Galaxy Z Fold6: the secure field's lost weight

**Session:** afa475cf-0f0c-4311-a30b-522251dd256d
**Date:** 2026-09-18 22:03 (follows "next-list-thread-widget-caret-fixes-boot-frame-proof")
**Branch:** master (6a350c2 → this commit)

## The ask

1. `/next-list 25`: rebuild the Next list from the last 25 session docs, then
   write it into the 20:02 doc. Committed as 6a350c2.
2. "I have an Android phone connected (it's a foldable) … do the steps that
   need an Android device."
3. `/sw`.

**The device:**
- Samsung Galaxy Z Fold6 (SM-F956U1), Android 16 (API 36), serial
  `RFCX70FJXZN`.
- Cover display: 968×2376 px at 420 dpi, 369×905 dp.
- Inner display: 1856×2160 px, 707×823 dp. Its id for `screencap -d` is
  `4630946165277524611`.
- The emulator `emulator-5554` was also attached, so every command sets
  `ANDROID_SERIAL`.

## 1. The Next-list rebuild (6a350c2)

- **The finding:** a 62-item list was dropped whole at
  `2026-0917-1659-fab-screen-floating-and-round-two-plan`. The plan-driven
  sessions put only the plan's next step under Next, and the list restarted
  fresh at 0917-2309.
- **Nothing on the dropped list was worked since,** apart from the F-key and
  first-key threads. All of it was restored and marked `lapsed@<doc>`.
- **Also restored:** the round-two widgets' "Not verified" notes, and
  `examples/signup` never getting PINInput.

## 2. Fix: a password field would not grow in a Row on Android

**Symptom.** In lesson 4.27, `comps.PasswordField` inside a FormField sat at
its intrinsic width beside "Show".
- The uiautomator bounds were `[123..501]`, 378 px of a 1423 px slot.
- Tapping Show swaps in a plain `core.Input`, which filled the slot
  (`[123..1546]`).
- The fault was on phones too; the unfolded sweep is what made it
  conspicuous.

**Cause.** In foundation **1.7.6**, `BasicSecureTextField` composes its field
inside `DisableCutCopy`, which is a `Box(Modifier.onPreviewKeyEvent{…})`.
- Confirmed with `javap` on the 1.7.6 AAR: `DisableCutCopy$1` calls
  `BoxKt.maybeCachedBoxMeasurePolicy`.
- So the Row's direct child is that Box. The weight on our modifier (parent
  data) lands one layout too deep and is dropped, and the Box hands the field
  a zero minimum.
- Foundation 1.10's source has moved the filter into the modifier chain.

**Fix (`Renderer.kt`, GrMobTextField's password branch).**
- `BoxWithConstraints(extra, propagateMinConstraints = true)` is now the
  direct child.
- The field takes `s.boxModifier().focusRequester(…)`, plus `fillMaxWidth()`
  when `constraints.hasFixedWidth`, which covers a weight's share or a
  stretch.
- The fill is appended after the box modifier, so a declared Width still
  wins.
- After the fix the hidden field measures `[123..1528]`.
- The comment says the wrapper can go with a BOM past foundation 1.10.

**A wrong first fix, reverted.**
- I first added `core.Width("100%")` to PasswordField's Row, with a comment
  saying Compose doesn't stretch a Column's child, plus a test pinning it.
- A build with the renderer fix and without the Go line proved the row
  already filled.
- The line, its comment and the test were all removed.

## 3. Fix: "Ran out 1 times" (lesson 4.24)

The caption now reads `Ran out %d time%s` through the tutorial's `plural`, the
same repair 0918-0713 made for "In-app link followed 1 times". It shows as
"Ran out 1 time" on the device.

## 4. Verified on the phone (cover screen unless noted)

- **4.22 FAB and Screen.Floating:**
  - The disc floats bottom-end over a full-height base.
  - A tap adds a note.
  - The inner `core.Scroll` scrolls under the fixed FAB, from Notes 1–5 to
    3–7.
- **4.23 QRCode, all four levels:**
  - A throwaway test in `internal/qr` sampled the screenshot's module grid
    against `qr.Encode` of the demo data. It was moved out afterwards.
  - Mismatches were 0 of 841, 1089, 1369 and 1681 (L, M, Q, H).
  - Module sizes ran from 10.7 to 14.2 px, and fractional.
  - The brightest pixel on any edge shared by two dark modules was **0**
    (black), so there are no hairline seams on Compose.
- **4.24 Countdown and Stopwatch:**
  - `OnDone` fired once per crossing (count 1, then 2).
  - The countdown rounds up and the stopwatch truncates.
  - Pause held at 0:16 for 3s; resume and reset work.
- **4.20 square scatter dots on Compose:**
  - I added four series to the demo temporarily, then reverted.
  - The 2nd and 4th series draw squares, with square legend swatches and dots
    the same size as the round ones (`scatter_chart.go:64`).
- **4.19 alarms:**
  - **Exact-alarm re-check:** "Allow exact alarms" opens Alarms & reminders.
    After granting and pressing Back, the button is gone.
  - **Kept banner:** I set "Ring at the next minute", pressed Home, then ran
    `am force-stop`. Nothing posted while stopped.
  - On relaunch after the minute, the `grmob.alarm.soon-1…` "Try it" banner
    posted late and was still shown 17s later, so the mount sweep left it.
  - Whether `OnRing` switched the one-off alarm off can't be read: the
    force-stop reset the lesson state.
- **4.21 on real fold hardware.** A watcher logged the readouts,
  `cmd device_state print-state` and the pid:

  | Position | device_state | Window dp | Classes | Posture | Fold |
  |---|---|---|---|---|---|
  | cover | CLOSED | 369×905 | compact/expanded | normal | none |
  | open flat | OPENED | 707×823 | medium/medium | normal | flat, vertical at x 354, continuous |
  | book | HALF_OPENED | 707×823 | medium/medium | book | half_opened, vertical at x 354, separating |
  | tabletop | HALF_OPENED | 823×707 | medium/medium | tabletop | half_opened, horizontal at y 354, separating |

  - The pid was unchanged throughout.
  - Unfolding recreates the Activity, since there are no `configChanges`, and
    the Go tree carried the lesson through.
  - **The TwoPane:**
    - Flat, it splits by Ratio 0.4 and ignores the continuous crease.
    - In tabletop it also uses the ratio split, by design (`IgnoreHorizontalFold`).
    - In book it splits at the hinge, but the Second pane starts at 417dp,
      **63dp right of the crease**. The TwoPane starts 63dp in from the window
      edge and passes no `Origin` (item 36).
  - **Folding shut** sent the app to the background and the launcher took the
    cover screen. That is Samsung's "Continue apps on cover screen" default.
- **Tablet width (707dp):** all of chapter 4 (4.1–4.33) was swept into
  per-lesson contact sheets. The only layout defect was the PasswordField in
  §2.

## 5. Found, not fixed

- **CodeEditor ignores `tabSize` on the natives.**
  - 4.13's seed indents `return` with a literal tab, and Compose draws it
    about one space wide.
  - Only `wasm/grmob-runtime.js:4546` reads `tabSize` (CSS `tab-size`); no
    `.kt` or `.swift` file does.
  - A fix needs a transformation with offset mapping on Compose (item 57).
- **Android RichTextEditor bullets** draw as "•a bullet" with no gap. The
  read-only RichTextView draws "• a bullet" (item 58).
- **4.19's permission rows on the 369dp cover screen:** "Without it Android may
  ring a minute late." wraps one or two words per line beside a full-width
  button. This is item 18's proportional-shrink gap, seen live.

## 6. What did not work: TalkBack on a Samsung phone

- **No utterance log.** The 13:10 method (disable TTS, read TalkBack's
  utterance log) doesn't carry over: Samsung's TalkBack (`com.samsung.android.
  accessibility.talkback`, 16.2) logs no utterance text.
  - With both engines disabled (`com.samsung.SMT`, `com.google.android.tts`),
    it spun in `FailoverTextToSpeech`, with about 38k `Create TextToSpeech`
    lines in seconds. The engines were re-enabled at once.
- **"Display speech output" works.** It's in TalkBack settings (reachable via
  `am start -n …/.TalkBackPreferencesActivity`) and shows each utterance at
  the bottom of the screen, where a screenshot can read it. Tab moves
  TalkBack's focus.
- **Why it still failed:**
  - Each TalkBack start opens its first-run tutorial.
  - It then asks for **phone permission** ("Allow phone access?", then the
    system prompt). Both grab keyboard focus.
  - Under TalkBack, injected taps don't activate, so the prompts could only be
    answered with TalkBack off. Force-stopping TalkBack closed the leftover
    screen.
- **State left on the phone:**
  - TalkBack off, TTS engines enabled, a11y settings back to `null`/`0`.
  - "Display speech output" still on.
  - TalkBack's READ_PHONE_STATE denied (USER_SET).
  - GrMob granted notifications and exact alarms.
  - GrMob's debug build installed.

## Pitfalls

- **Two devices attached:** `adb` without `-s` fails. Export
  `ANDROID_SERIAL`.
- **The Fold's displays:**
  - `screencap` without `-d` warns and captures the first display.
  - Unfolded, use `-d 4630946165277524611`.
  - uiautomator dumps sometimes came back empty on the inner display, so
    screenshots were the reliable reading there.
- **`ui.sh tap`** matches by substring. "Allow" hit the dialog title, and a
  single letter ("L") hit other text. Tap exact bounds from the dump instead.
- **zsh:**
  - A function named `d` collides with an alias.
  - `set -- $w` doesn't word-split (use `${=w}`), which broke a capture
    script's portrait test.
  - `echo ===` fails.
- **Foreground `sleep`** longer than a moment is blocked by the harness.
  Waits went into `run_in_background` until-loops.
- **A capture keyed on `device_state == HALF_OPENED`** can fire before the app
  has re-laid out. The first "book" frame showed the flat layout. Put the
  Fold readout on screen with the thing being judged.
- **Contact sheets:** `montage p*.png -tile x1 -geometry 460x+6+0` makes
  one image per lesson. A slow `input swipe … 900` pages without a fling.

## Files

- `android/app/src/main/java/com/grmob/runtime/Renderer.kt`: the password
  branch's wrapper.
- `examples/tutorial/chapter4.go`: 4.24's plural.

## Verification

- `go vet ./...` and `go test ./...` are clean.
- `android/verify/run.sh` is OK. `android/verify/sources.sh` is OK: the
  runtime compiles against Gradle's classpath with the Compose plugin.
- **On the device:**
  - 4.27's hidden field fills (`[123..1528]`).
  - 4.24 reads "Ran out 1 time".
  - The scatter and QR checks passed as above.

## Next

**Legend:** *age* is how many session docs ago the item was first raised,
counted from this doc (0 = raised here; `≥ k` = older than the 25-doc window
the last rebuild used). *value* is the payoff, not the effort:
- **high:** worked around today, or a second consumer has arrived.
- **medium:** blocks one named thing, or is a visible defect.
- **low:** nobody has hit the gap yet.

Items are sorted by age, oldest first, then by value. `lapsed@<doc>` marks an
item that fell off the list without being done; the doc named is where it
went missing.

**Closed this session** (numbering from the 20:02 doc):
- **35:** foldable behaviour on real hardware.
- **Part of 25:** the Android exact-alarm re-check and the kept banner after a
  force stop.
- **Part of 31:** Compose's square dots.
- **Part of 33:** chapter 4 at tablet width.
- **Part of 43:** the round-two widgets on Android.

The remainders are carried below.

1. **(age ≥25 · value medium · lapsed@0917-1659) Lessons on hardware, and
   checks still open.**
   - Screen readers: radio and StepIndicator semantics, combobox active
     option, "pop-up" triggers, aria-current, Calendar's grid, CodeEditor
     toolbar role on Compose, SearchableSelect on the natives,
     AccessibilityHidden behind a Drawer.
   - iOS: sheet Dialog and ActionSheet filler, Drawer under Reduce Motion,
     RTL for Drawer and CodeEditor, the sideways editor, 4.6's list.
   - Pinning and Drawer: `Screen.Footer`, 100% layers in a pinned ZStack,
     `core.Focus` on a Button, a hardware keyboard reaching a shut panel.
   - Android: predictive back, MaxWidth where it binds on a tablet, RTL capped
     child.
   - A real Android device is now available (Fold6).
2. **(age ≥25 · value medium · lapsed@0917-1659) `.claude/settings.json`
   cannot be edited from a session** (`[Self-Modification]`). Worth an
   upstream report. Not re-checked since 2026-09-15.
3. **(age ≥25 · value low) F-keys through GameController have never reached
   the app from XCUITest.** `testFunctionKeyPressesTheButton` is a strict
   expected failure. Needs a real iPad keyboard.
4. **(age ≥25 · value low · lapsed@0917-1659) Compose's today
   `stateDescription` is not heard.**
   - Tried this session on the Fold6 and blocked; see §6.
   - The emulator's Google TalkBack logs utterances; Samsung's does not.
   - Samsung TalkBack opens a tutorial and a phone-permission prompt on every
     start, and both steal focus.
   - Cheapest now is a person listening: TalkBack on, 4.9, touch day 11,
     listen for "today".
   - The alternative is the emulator's method from 2026-0918-1310.
5. **(age ≥25 · value low · lapsed@0917-1659) Merging a labelled node is
   unheard under TalkBack** (a labelled container holding two controls).
   Same blocker and same route as item 4; they land together.
6. **(age ≥25 · value low · lapsed@0917-1659) iOS chords.** Page-global
   chords were verified once, and the chord gate (behind a modal, inside a
   shut Drawer panel) is unheard. `primeKeyboard` makes simulator delivery
   dependable enough to retry.
7. **(age ≥25 · value low · lapsed@0917-1659) The cost of a paused
   TimelineView per node** in `GrMobMotion`. Not profiled.
8. **(age ≥25 · non-goal · lapsed@0917-1659)**
   - Rename `docs/components.md` to `comps.md`; rewrite `components` in the
     older plans.
   - Trim the Android shell's permissions; the iOS usage strings.
   - C4 `Carousel` until the host reports a scroll offset. Item 64 declines
     that offset, so this is effectively permanent.
   - Android `onBack` ranking (a late parent; an AppBar outside the
     Navigator).
   - Forward after browser back.
   - Sticky headers or `OnEndReached` in a List with no viewport on Compose.
   - MaxWidth with a growing sibling.
   - The typed-hash `history.length` fallback; a page's own `pushState`
     during a claim.
   - A Drawer's shut panel is composed on the natives.
   - A List with no Height is not lazy.
   - Unformatted commits already on a remote.
9. **(age 24 · value low · lapsed@0917-1659) Alarm sound and haptics are
   unheard,** and so is the Notify banner's default sound. The Fold6 can do
   it with a person holding it.
10. **(age 24 · value low · lapsed@0917-1659) Canvas still omits** text,
    clipping and per-shape hit-testing.
11. **(age 24 · non-goal · lapsed@0917-1659)** `DigitalClock` digits shifting
    by a pixel; `AnalogClock{Smooth}` spinning back when the midnight tick is
    skipped.
12. **(age 23 · value low · lapsed@0917-1659) A chart's hidden data table**
    was not built. It needs a screen-reader-only primitive: an API decision.
13. **(age 23 · value low · lapsed@0917-1659) Chart summaries are English.**
14. **(age 23 · non-goal · lapsed@0917-1659)** A 180° `Gauge` leaves its bottom
    half empty.
15. **(age 22 · value low · lapsed@0917-1659) A Notify alarm is a banner, not a
    ringing screen.** The route is AlarmKit or full-screen intents.
16. **(age 22 · value low · lapsed@0917-1659) `mobile.SetTimeZone` runs once at
    startup.** Fixing it needs core to hold the location: an API decision.
17. **(age 22 · value low · lapsed@0917-1659) The web's scheduled notification
    and its sweep are unseen in a real browser.**
18. **(age 21 · value low · user's decision · lapsed@0917-1659) Compose Rows
    don't shrink children in proportion.**
    - Seen live this session on the 369dp cover screen: 4.19's "Allow exact
      alarms" row wraps its caption one or two words per line beside a
      full-width button.
    - A visible instance rather than a theoretical one.
19. **(age 20 · value low · lapsed@0917-1659)
    `testRelaunchSweepsWhatTheDeadProcessScheduled` needs notifications
    already granted.**
20. **(age 20 · value low · delete?) The double-post claim is unreproduced.**
    Carried nine times; proposed for deletion.
21. **(age 20 · non-goal · lapsed@0917-1659)** `MaxLines` on the natives
    applies to Text only; a stacked chart counts NaN as 0.
22. **(age 19 · value low · lapsed@0917-1659) `DefaultDarkChartColors` has no
    bundled consumer.**
23. **(age 19 · value low · delete?) `TutorialChartsUITests` failed once,
    reason not captured.** Never recurred; proposed for deletion.
24. **(age 19 · non-goal · lapsed@0917-1659)** `core.LinearGradient` is unused;
    kept per the no-removal rule.
25. **(age 18 · value low · lapsed@0917-1659) The alarm changes: what is left
    unseen.**
    - Android was seen this session: the exact-alarm re-check, and the kept
      banner after a force stop.
    - Still open: a re-run of `TutorialAlarmNotifyUITests` (iOS relaunch).
    - Still open: whether `OnRing` switches a one-off alarm off after a
      force-stop relaunch. This run couldn't tell, because the lesson state
      reset with the process.
    - Value lowered from medium: the risky half is now seen.
26. **(age 18 · value medium · blocked · lapsed@0917-1659) The iOS Image floor
    runs high for a narrow image.** Unblocking it needs a px-width box that
    can shrink (`grMobDimension`).
27. **(age 18 · value low → non-goal? · lapsed@0917-1659) A zero basis is
    honoured on iOS only with a definite main extent.** Documented as
    deliberate; proposed as a non-goal.
28. **(age 18 · non-goal · lapsed@0917-1659)** Renaming a `NotifyGroup`
    strands its schedules.
29. **(age 17 · value medium · lapsed@0917-1659) An iOS Image with no
    background shows a black letterbox** on a "fit" image. Presumably open;
    unverified on 26.5.
30. **(age 17 · value low · lapsed@0917-1659) `barValueRoom` is still an
    estimate.**
31. **(age 17 · value low · lapsed@0917-1659) Square scatter dots are unseen
    in Chrome.** Compose was seen this session.
32. **(age 17 · value low · lapsed@0917-1659) `testStatTilesShareTheRow`
    asserts weakly.**
33. **(age 17 · value low · lapsed@0917-1659) Comps on Android in landscape
    are unswept.**
    - All of chapter 4 was swept at 707dp portrait this session, and found
      only the PasswordField (fixed).
    - Landscape at 823dp was seen for 4.21 alone.
    - The other chapters were not swept at width.
34. **(age 17 · value low → non-goal? · lapsed@0917-1659) Sparkline's `Area`
    is a flat tint.** Left alone on purpose; proposed as a non-goal.
35. **(age 16 · value medium · lapsed@0917-1659) Safe-area insets are not a
    record.** A TwoPane under the status bar can't find its `Origin.Y`.
    `core/` has no `SafeInsets`.
36. **(age 16 · value low · lapsed@0917-1659) Lesson 4.21's TwoPane sets no
    `Origin`.**
    - Measured on the Fold6 in book posture: the split lands at 417dp,
      **63dp** right of the 354dp hinge (the old note guessed ~30).
    - It is harmless: the list is narrow, so nothing sits on the crease.
    - The inset is the page margin plus the panel padding, both known to the
      tutorial.
37. **(age 16 · value low · lapsed@0917-1659) iOS `AppWindowReader` is
    type-checked only.** Not run in Split View or Stage Manager.
38. **(age 16 · value low · lapsed@0917-1659) The browser's segments/posture
    path is unseen in a real browser.**
39. **(age 16 · value low · lapsed@0917-1659) Folding shut onto the outer
    display.**
    - On the Fold6 at defaults, the app goes to the background.
    - The display switch happens only with Samsung's "Continue apps on cover
      screen" setting on, and that is unseen.
    - Seen: unfolding recreates the Activity with no `configChanges`, and the
      process-held Go tree keeps the lesson and its state.
40. **(age 16 · value low · lapsed@0917-1659) Housekeeping:** the
    `GrMob_Foldable` AVD is still installed. It is less needed now that a real
    Fold6 is available.
41. **(age 16 · non-goal · lapsed@0917-1659)** More than one fold; a static
    HTML export of a TwoPane.
42. **(age 15 · value medium · lapsed@0917-2309) The round-two widgets have
    never been seen on iOS.**
    - Screen.Floating and the FAB (4.22), the QRCode seams, and
      Countdown/Stopwatch (4.24).
    - All three were verified on Android this session; QR was pixel-exact
      with no seams.
43. **(age 11 · value medium) Examples should adopt the shipped widgets.**
    - `examples/chat` hand-builds what `comps.MessageThread` does
      (`main.go:148`).
    - `examples/signup` never got `PINInput`.
    - Both change shotclaims, so they land together.
44. **(age 9 · non-goal)** A native time wheel; a sheet or Done on TimePicker.
45. **(age 8 · non-goal)** Heatmap as a continuous gradient; a Sequential ramp
    interpolated from `Primary`.
46. **(age 7 · value low · delete?) A sixth low-hanging-fruit round.** No
    candidates across five carries; proposed for deletion until one exists.
47. **(age 7 · non-goal)** A year-wide scrolling `CalendarHeatmap`; a
    caption-flip "Copied ✓" on CopyButton; a separate `Alert` widget (Banner
    is it).
48. **(age 6 · non-goal)** A two-pane layout on the natives; restructuring
    lesson bodies into guide and demo halves.
49. **(age 5 · non-goal)** Making iOS keep focus on every submit by default.
50. **(age 4 · value medium) The first hardware key after launch is lost on
    the iOS 26.5 simulator.** Every keyboard UI test works around it with
    `primeKeyboard`. Unchecked on a real iPad.
51. **(age 4 · non-goal)** A right-padding gutter or a toolbar header for code
    blocks.
52. **(age 3 · non-goal)** A smaller minimum for Buttons in general on
    Android. Only CopyButton opts out.
53. **(age 2 · value low) iOS UIKit field: a queued key during the caret
    correction.**
    - Not observed in 6 correction writes.
    - The candidate fix: drain through `input.inputDelegate`
      `selectionWillChange/DidChange` before `replace`, then
      `rebaseEdit(current, flushed, next)`.
54. **(age 1 · value low) `core.ScrollIntoView` inside an iOS `core.List`.**
    - `GrMobList`'s `ScrollViewReader` does not inject `\.grMobScrollProxy`.
    - Rows are keyed by `rowKey`, while `GrMobBringIntoView` scrolls by
      `viewID`.
    - Measure before claiming it.
55. **(age 1 · value low) iOS Paragraph link colours on the iOS 17 floor.**
    Measured only on 26.5; the `.tint(firstLink)` fallback remains.
56. **(age 1 · value low · by design) The web's thread place-keeping applies
    only when the List is its own scroll box.**
57. **(age 0 · value medium) CodeEditor's `tabSize` is honoured on the web
    only.**
    - A literal tab draws about one space wide on Compose. Seen in 4.13's
      seed, whose `return` is tab-indented.
    - Only `wasm/grmob-runtime.js` reads the prop; no Kotlin or Swift does.
    - Compose needs a display transformation with an offset mapping, so the
      caret and selection stay in buffer units.
    - iOS is unmeasured.
58. **(age 0 · value low) The Android RichTextEditor draws list bullets
    without a gap** ("•a bullet"). The read-only RichTextView draws
    "• a bullet". Cosmetic.
59. **(age 0 · value low · contingent) The secure-field wrapper in
    `Renderer.kt` can go once the Compose BOM reaches foundation 1.10,** which
    moved `DisableCutCopy`'s key filter into the modifier chain. Contingent on
    a BOM bump that isn't planned.
60. **(age 0 · value low) PasswordField's width on iOS is unmeasured.** The
    Android defect was a foundation wrapper, and SwiftUI's SecureField is a
    different shape. One look at 4.27 on the simulator settles it.
61. **(age 0 · value low) Other chapters at tablet width.** Only chapter 4 was
    swept on the unfolded Fold6. `sweep.sh` and the montage recipe in
    Pitfalls make the rest cheap. Lands with item 33.
62. **(age 0 · value low) An Android TalkBack route that works on Samsung.**
    - "Display speech output" plus screenshots works.
    - It is blocked only by TalkBack's first-run tutorial and phone-permission
      prompt on every enable.
    - Pre-granting or permanently denying READ_PHONE_STATE, and finishing
      the tutorial once by hand, may clear it for good.
    - Unlocks items 4 and 5.
63. **(age 0 · non-goal)** Continuing apps onto the cover screen by default.
    That is a Samsung system setting, not the app's.
64. **(age 1 · non-goal)** A reported scroll offset from the hosts. The thread
    needed only the two List props (see `core/list_start.go`).

Read by value instead:
- **high:** none.
- **medium:** 1, 2, 26, 29, 35, 42, 43, 50, 57.
- **low:** 3–7, 9, 10, 12, 13, 15–20, 22, 23, 25, 27, 30–34, 36–40, 46,
  53–56, 58–62.
- **non-goal:** 8, 11, 14, 21, 24, 28, 41, 44, 45, 47–49, 51, 52, 63, 64.
