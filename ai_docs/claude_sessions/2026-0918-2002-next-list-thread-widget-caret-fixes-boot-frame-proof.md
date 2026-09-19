# Next list: a thread widget, three caret fixes, and the boot frame proved

**Session:** 7cfb7682-eaa7-4e25-be9c-14e8e818b8b8
**Date:** 2026-09-18 20:02 (follows "next-list-paragraph-corners-scroll-to-one-field-pin")
**Branch:** master (2280991 → this commit)

## The ask

"Do all items in the Next list except hardware dep ones." Then `/sw`.

- **Skipped as hardware:** F-keys through GameController, and the first
  hardware key lost after launch.
- **Devices:** Android emulator `emulator-5554` (API 36), iOS simulator
  iPhone 17 Pro (iOS 26.5, `7572B953-…`), and headless Chrome (CDP).

## 1. Web: the `keyboard` prop is cleared by a patch that drops it

- **Fix:** `applyInputMode(el, props)` in `grmob-runtime.js`. It runs
  unconditionally before the per-key loop, as `applyEnterKeyHint` does,
  because the patch carries the whole new props map. It applies only to
  `<input>` and `<textarea>`, and the loop now skips `keyboard`.
- **Tests:** `paragraph_test.mjs`, "a patch without the keyboard prop clears
  inputmode" and "a props patch on a non-field leaves inputmode alone".

## 2. iOS caret under a rewrite that is not at the caret

- **New demo:** lesson 2.3 gained a "Capitalize each word" option
  (`capitalizeWords`, Go tests). UPPERCASE only changes the key just typed.
  Capitalizing changes text on both sides of the caret, which reaches the
  field's correction arm.
- **Real bug found (`GrMobTextInputCoordinator.write`):** a caret *inside* a
  length-kept span was sent to the span's end.
  - Typing "xyz" at offset 5 of "hello world" gave "Hellox Wyzorld".
  - Fix: `was <= start || delta == 0` keeps the caret, the rule
    `rebaseMapOffset` states.
- **The carried race did not reproduce:** a key arriving between `replace`
  and the caret correction was never seen.
  - The probe (file-based; `NSLog` never reached `log show`) saw 6 of 6
    corrections with no key landing.
  - The field's `inputDelegate` is `_UIKeyboardStateManager`, so a drain
    through `selectionWillChange/DidChange` is plausible. It was not added,
    because it can't be verified.
- **Test:** `testCapitalizedWordsKeepTheCaretWithTheTyping`, run from both
  seeds.
- **Android is fine:** it keeps the caret offset inside one atomic
  `field.edit`, so it has neither the span-end rule nor a window.

## 3. Web caret under a Go transform (found this session)

- **The bug:** `el.value = v` on a focused `<input>` sends the caret to the
  end. Measured over CDP, typing "abc" at offset 5 of "HELLO WORLD" under
  UPPERCASE gave "HELLOA WORLDBC".
- **Fix:** `writeFieldValue`, which maps the caret across old→new by the
  native hosts' rule (surrogate-safe).
  - No ledger is needed: the event and its patch are one synchronous call.
  - Tests: `fieldvalue_test.mjs`.
- **Browser check 18** types into the live build's 2.3. With the fix
  reverted it fails with exactly "HELLOA WORLDBC" at 14.

## 4. Stale iOS UI tests, and a Slider bug under one

- **GrMobUITests.testFeedListGesturesAndVirtualization:** the test was
  stale, not the app. The row now states its selection as a state
  (`AccessibilitySelected`), and the Go test pins that the ", selected"
  suffix is gone. The test now asserts `isSelected`.
- **AudioUITests:** also stale.
  - Changed queries: "State: playing", "Forward 15 seconds",
    "Speed 1×"/"1.25×". The network was never the problem.
  - **The real bug:** `adjust(toNormalizedSliderPosition:)` failed ("Unable
    to get expected attributes for slider", scrubber at 0,0).
    - Cause: grMobBox names a labelled node with
      `.accessibilityElement(children: .combine)`, which wraps a native
      control in a synthesized element.
    - Fix: `GrMobSlider` names the Slider itself (`grMobControlName`) and
      strips the label from its box (`unnamed`, local to Renderer.swift
      because ios/verify compiles it without GrMobTextInput.swift).
  - The whole transport passes: 0:04 → seek 5:41 → paused → ended →
    stopped.

## 5. iOS Paragraph links: no one-colour limit

- **Measured:** 4.29 gained a sentence with two links (Link.Span in Primary,
  "report a problem" in Error). On iOS 26.5 each link draws in its own
  `foregroundColor`.
- **Kept:** the `.tint(firstLink)` fallback stays, with the comment
  corrected. The iOS 17 floor is unmeasured (only 26.5 is installed).
- **Test:** `testParagraphLinksKeepTheirOwnColours`, where tapping the
  second link runs its own callback.
- **Also:** 4.29's prose no longer claims core has no inline span.

## 6. The web boot frame, proved

- **Browser check 17** loads the real `wasm/index.html` at 1280px. Its
  unpkg lines are removed, because the pass promises no network.
  - A trap on the `GrMobWASM` global records the tree `RenderInitial`
    returns.
  - A MutationObserver records the DOM at the first microtask checkpoint
    after the mount.
- **The first probe was fooled** by `#app`'s own "Loading the tutorial…"
  spinner. It now waits for `#app [data-node-path]`.
- **Finding:** with the old order restored, the first reading fails (phone
  tree) and the second holds.
  - The layout patch is pushed from inside the `HostEvent` call, in the same
    task as the mount, so **no phone frame was ever painted**.
  - The old order's cost was a whole phone tree built, mounted and patched
    away.
  - The comments in split.go and index.html are corrected.

## 7. The thread widget

**Core (`core/list_start.go`):**
- `OnStartReached(fn)` is `OnEndReached`'s mirror, sharing its ledger, which
  is keyed by callback ID.
- `StartAtEnd()` opens a List at its end and keeps it there on appends while
  the reader is at the end.
- The doc has a per-host table, and every row of it was measured.

**Widget (`comps.MessageThread`):**
- Fields: `Messages []ThreadMessage` (oldest first, stable `Key`),
  `OnLoadOlder` (nil at the beginning), `Loading`, `Height`, `LoadingText`,
  `StartText`, `MineLabel`, `Style`.
- Each bubble is `Keyed("msg:"+Key)`.
- A run from one sender uses `MessageBubble.Continued` (new field). The line
  is not drawn; the name is still spoken.
- **The caption is a line *above* the List, never a row in it.** As a
  permanent top row it was the first visible row, which every host anchors
  on. It stayed first after a prepend, so each arrival at the top would
  cascade to the beginning.
- **The List is an unnamed RoleLog.** On iOS a name on a container folds
  every bubble into one stop.

**Tutorial:**
- Lesson 4.33 "Message threads": a pretend 48-message server, 12 per page,
  and `hooks.UseTimeoutWhile` 700ms as the fetch. It has a Go test.
- 4.31 now uses `Continued`.

**Web:**
- `data-key` on keyed nodes.
- A start `IntersectionObserver` on the first child.
- A thread's place: around every patch batch, `captureListAnchors` records
  the first visible row's key and offset, and `restoreListAnchors` puts it
  back. This runs synchronously, before the re-pointed observer's first
  report.
- A list at its end stays there; `openListsAtEnd` runs after a mount or
  batch.
- Tests: `startreached_test.mjs`.

**Browser check 19:**
- It opens at the end, and one arrival at the top loads exactly one page.
- The row in view is back at 12px (scrollTop 0 → 831).
- A send at the end is followed; a send while scrolled back moves nothing.
- The negative control (restore disabled) fails with "loaded a second older
  page".
- It needed `Page.bringToFront`: by then the page drew no frames, so rAF and
  IntersectionObserver never ran.

**Android:**
- Initial index at the last row.
- `StartReachedReporter`: `firstVisibleItemIndex < 2`.
- `StickToEnd`: at-end is remembered from `!canScrollForward` and applied
  with `scrollToItem(last)` on a row-count change.
- The place is kept by LazyColumn's keyed anchoring, needing nothing extra.
- Emulator: Message 37 stayed at y=1230 as the page landed; a second arrival
  loaded "36 of 48"; "hello" was followed; "again" while scrolled back moved
  nothing, and was delivered.

**iOS, which took three tries:**
- `defaultScrollAnchor(.bottom)` alone opened at the end but kept no place,
  and all 3 pages loaded.
- The first row's `.onAppear` is too eager: the lazy stack materializes rows
  ahead of the viewport.
- **The working version:**
  - `scrollPosition(id: $topRow, anchor: .top)` with `.scrollTargetLayout()`.
  - The top edge is read from that binding (`topEdgeReached`: the top row is
    one of the first two).
  - **The binding must be typed `String?`** (`GrMobNode.rowKey`, plus
    `.id(child.rowKey)` on rows). Typed `AnyHashable?`, it reported
    SwiftUI's internal `UniqueID`s.
  - `stickToEnd` scrolls to a new last row through a `ScrollViewReader`
    while `nearEnd`. `nearEnd` is set by the last row's appear and disappear
    events.
- Test: `testMessageThreadOpensAtTheEndAndKeepsThePlace`.

**htmlout:** `data-onstartreached`, with a test.

**Docs:** `docs/components.md` (MessageThread, `Continued`), a ROADMAP entry,
and `docs/api/*` regenerated (`list_start.go`, `message_thread.go` added to
apidoc topics).

## Pitfalls

- **The lesson count is in six places.** Adding lesson 4.33 meant 74 → 75 in
  index.html, README (twice), docs/tutorial-interactive.md, shotclaims, and
  two test comments. It also meant retaking `tutorial-contents.png`
  (`wasm/shots/shoot.sh tutorial-contents`).
- **The census of tracked Go files** went 621 → 625 (the four new files) in
  `wasm/verify/{repowalks,timings}_test.go`.
- **`internal/pinfixture`'s lexer trips on an apostrophe** inside a template
  literal nested in `${}` in browser.mjs. Keep apostrophes out of nested
  templates.
- **`mjsnames_test`** rejects constant names that live elsewhere (SPLIT_MIN
  is index.html's).
- **The browser.mjs numbering** is checked by `checknumbering_test.go`, and
  the tallies are number words. The map gained "seventeen" through
  "twenty".
- **CDP evaluates time out at 15s**, so polls inside one must stay under it.
- **zsh:** `echo ===` fails (`=cmd` expansion), and `$args` is not
  word-split. Use arrays.
- **iOS UI tests:**
  - `lift()` can push a box off-screen while its elements still report
    relative frames.
  - `app.scrollViews.containing(label CONTAINS 'Message 4')` stops matching
    once those rows leave the lazy stack.
- **Test-runner logs:** app `NSLog` did not show in `simctl log show`. Probe
  to a file in the app container's Documents instead.
- **Chrome extension typing** was lost to the occluded tab again. A scratch
  CDP script (headless) was reliable.

## Files

- **Go core:** `list_start.go` (new, + test).
- **comps:** `message_thread.go` (new, + test), `message_bubble.go`.
- **htmlout:** `export.go`, `collection_props_test.go`.
- **internal:** `apidoc/packages.go`, `shotclaims/shotclaims.go`.
- **Tutorial:** `chapter2.go`, `chapter4.go` (+ tests), `split.go`,
  `pagecount_test.go`, `screenshot_test.go`.
- **Web:**
  - `grmob-runtime.js`, `index.html`;
  - `wasm/verify/{browser.mjs,run.sh,checknumbering_test.go,paragraph_test.mjs,repowalks_test.go,timings_test.go}`;
  - new `fieldvalue_test.mjs` and `startreached_test.mjs`.
- **Android:** `Renderer.kt`.
- **iOS:** `Renderer.swift`, `GrMobTextInput.swift`, and UI tests
  (`TutorialDevicePassUITests`, `GrMobUITests`, `AudioUITests`).
- **Docs:** `README.md`, `ROADMAP.md`, `docs/components.md`,
  `docs/tutorial-interactive.md`, `docs/api/*`,
  `docs/images/tutorial-contents.png`.

## Verification

- `go test ./...` and `go vet ./...` are clean, and gofmt is clean.
- wasm/verify (all node suites, plus browser checks 1–19), android/verify,
  ios/verify and mobile/verify pass.
- **iOS UI tests:**
  - All 39 tutorial tests pass.
  - GrMobUITests (feed) and AudioUITests pass on the mobileapp build.
  - TodoAppUITests passes on the todoapp build.
- **Android:** the emulator walk of lesson 4.33 described in section 7.
- **Negative controls:** check 17 fails with the old boot order, check 18
  without `writeFieldValue`, and check 19 without `restoreListAnchors`.

## Next

**Legend:** *age* is how many session docs ago the item was first raised,
counted from this doc (0 = raised here; `≥ 24` = older than this 25-doc
window, 2026-0915-1926 → this doc). *value* is the payoff, not the effort:
**high** = worked around today or a second consumer has arrived; **medium** =
blocks one named thing or is a visible defect; **low** = nobody has hit the
gap yet. Sorted by age, oldest first, then by value. `lapsed@<doc>` = the item
fell off the list without being done; the doc named is where it went missing.

**Rebuilt from history.** A whole list of 62 items was dropped at
`2026-0917-1659-fab-screen-floating-and-round-two-plan`: from there the Next
section held only the plan's next step, and the list restarted fresh at
2026-0917-2309. None of the dropped items were worked in the 15 docs since,
except the F-key and first-key threads. They are restored below.

1. **(age ≥24 · value medium · lapsed@0917-1659) Lessons on hardware, and
   checks still open.**
   - Screen readers: radio and StepIndicator semantics, combobox active
     option, "pop-up" triggers, aria-current, Calendar's grid, CodeEditor
     toolbar role on Compose, SearchableSelect on the natives,
     AccessibilityHidden behind a Drawer.
   - iOS: sheet Dialog and ActionSheet filler, Drawer under Reduce Motion,
     RTL for Drawer and CodeEditor, the sideways editor, 4.6's list.
   - Pinning and Drawer: `Screen.Footer`, 100% layers in a pinned ZStack,
     `core.Focus` on a Button, a hardware keyboard reaching a shut panel.
   - Android: predictive back, MaxWidth on a tablet, RTL capped child.
   - *Partly stale premise:* simulator device passes and a TalkBack utterance
     log now exist. Re-sort into what a simulator can check and what truly
     needs hardware.
2. **(age ≥24 · value medium · lapsed@0917-1659) `.claude/settings.json`
   cannot be edited from a session** (`[Self-Modification]`). Worth an
   upstream report. Not re-checked since 2026-09-15.
3. **(age ≥24 · value low) F-keys through GameController have never reached
   the app from XCUITest.** `testFunctionKeyPressesTheButton` is a strict
   expected failure; needs a real iPad keyboard. Merges the older "F-keys on
   iOS unverified" and "GameController cannot take a key".
4. **(age ≥24 · value low · lapsed@0917-1659) Compose's today
   `stateDescription` is not heard.** Now cheap: TalkBack logs its utterances
   when it has no TTS engine (2026-0918-1310, section 2), the method this item
   said was missing.
5. **(age ≥24 · value low · lapsed@0917-1659) Merging a labelled node is
   unheard under TalkBack.** The shape is a labelled container holding two
   controls; 2026-0918-1310 covered only a labelled Button. Same method as
   item 4; they land together.
6. **(age ≥24 · value low · lapsed@0917-1659) iOS chords.** Page-global chords
   were verified once, and the chord gate (behind a modal, inside a shut
   Drawer panel) is unheard. `primeKeyboard` now makes simulator delivery
   dependable enough to retry.
7. **(age ≥24 · value low · lapsed@0917-1659) The cost of a paused
   TimelineView per node** in `GrMobMotion`. Not profiled.
8. **(age ≥24 · non-goal · lapsed@0917-1659)**
   - Rename `docs/components.md` to `comps.md`; rewrite `components` in the
     older plans.
   - Trim the copied Android shell's permissions; replace the iOS usage
     strings further.
   - C4 `Carousel` until the host reports a scroll offset. Item 58 now
     declines that offset, so this is effectively permanent.
   - Android `onBack` ranking: a parent gaining `onBack` late, and an AppBar
     outside the Navigator.
   - Forward does not re-open a screen left by browser back.
   - Sticky headers, placement animation and `OnEndReached` in a List with
     no viewport on Compose.
   - MaxWidth with a growing sibling on the natives.
   - The typed-hash fold's `history.length` fallback; a page's own
     `pushState` while a claim is on screen.
   - A Drawer's shut panel is composed on the natives.
   - A List with no Height in a scrolled page is not lazy.
   - Commits already on a remote carrying an unformatted file.
9. **(age 23 · value low · lapsed@0917-1659) Alarm sound and haptics are
   unheard,** and so is the Notify banner's default sound.
10. **(age 23 · value low · lapsed@0917-1659) Canvas still omits** text,
    clipping and per-shape hit-testing.
11. **(age 23 · non-goal · lapsed@0917-1659)** `DigitalClock` digits shifting
    by a pixel; `AnalogClock{Smooth}` spinning back when the 00:00:00 tick is
    skipped.
12. **(age 22 · value low · lapsed@0917-1659) A chart's hidden data table** was
    not built. It needs a screen-reader-only primitive: an API decision.
13. **(age 22 · value low · lapsed@0917-1659) Chart summaries are English.**
14. **(age 22 · non-goal · lapsed@0917-1659)** A 180° `Gauge` leaves its bottom
    half empty.
15. **(age 21 · value low · lapsed@0917-1659) A Notify alarm is a banner, not a
    ringing screen.** The route is AlarmKit or full-screen intents.
16. **(age 21 · value low · lapsed@0917-1659) `mobile.SetTimeZone` runs once at
    startup.** A fix needs core to hold the location: an API decision.
17. **(age 21 · value low · lapsed@0917-1659) The web's scheduled notification
    and its sweep are unseen in a real browser.**
18. **(age 20 · value low · user's decision · lapsed@0917-1659) Compose Rows
    don't shrink children in proportion.**
19. **(age 19 · value low · lapsed@0917-1659)
    `testRelaunchSweepsWhatTheDeadProcessScheduled` needs notifications
    already granted.**
20. **(age 19 · value low · delete?) The double-post claim is unreproduced.**
    Carried nine times without a reproduction. Proposed for deletion.
21. **(age 19 · non-goal · lapsed@0917-1659)** `MaxLines` on the natives
    applies to Text only; a stacked chart counts NaN as 0.
22. **(age 18 · value low · lapsed@0917-1659) `DefaultDarkChartColors` has no
    bundled consumer.**
23. **(age 18 · value low · delete?) `TutorialChartsUITests` failed once,
    reason not captured.** It never recurred. Proposed for deletion.
24. **(age 18 · non-goal · lapsed@0917-1659)** `core.LinearGradient` is unused;
    kept per the no-removal rule.
25. **(age 17 · value medium · lapsed@0917-1659) The alarm changes are unseen
    on devices.** Kept banners after a force stop, the Android exact-alarm
    re-check, and a re-run of `TutorialAlarmNotifyUITests`. Merges old items 43
    and 46, which land together.
26. **(age 17 · value medium · blocked · lapsed@0917-1659) The iOS Image floor
    runs high for an image narrower in proportion than its box.** Unblocking
    it needs a px-width box that can shrink (`grMobDimension`).
27. **(age 17 · value low → non-goal? · lapsed@0917-1659) A zero basis is
    honoured on iOS only with a definite main extent.** Documented as
    deliberate (`GrMobFlexZeroBasis`). Proposed as a non-goal.
28. **(age 17 · non-goal · lapsed@0917-1659)** Renaming a `NotifyGroup`
    strands what the old name scheduled.
29. **(age 16 · value medium · lapsed@0917-1659) An iOS Image with no
    background shows a black letterbox** on a "fit" image. No letterbox
    handling exists in `ios/`, so it is presumably open; unverified on 26.5.
30. **(age 16 · value low · lapsed@0917-1659) `barValueRoom` is still an
    estimate** (a 220px plot, 0.6em per rune).
31. **(age 16 · value low · lapsed@0917-1659) Square scatter dots are unseen**
    on Compose and Chrome.
32. **(age 16 · value low · lapsed@0917-1659) `testStatTilesShareTheRow`
    asserts weakly.**
33. **(age 16 · value low · lapsed@0917-1659) Comps on Android were seen at
    phone size only.** Tablet widths and landscape are unswept.
34. **(age 16 · value low → non-goal? · lapsed@0917-1659) Sparkline's `Area` is
    a flat tint.** It was left alone on purpose. Proposed as a non-goal.
35. **(age 15 · value medium · lapsed@0917-1659) Foldable behaviour is
    unverified on real hardware** (tabletop and book postures).
36. **(age 15 · value medium · lapsed@0917-1659) Safe-area insets are not a
    record.** A TwoPane under the status bar can't find its `Origin.Y`.
    Verified: `core/` has no `SafeInsets`.
37. **(age 15 · value low · lapsed@0917-1659) Lesson 4.21's TwoPane sets no
    `Origin`,** so the split lands ~30dp right of the hinge.
38. **(age 15 · value low · lapsed@0917-1659) iOS `AppWindowReader` is
    type-checked only.** Not run in Split View or Stage Manager.
39. **(age 15 · value low · lapsed@0917-1659) The browser's segments/posture
    path is unseen in a real browser.**
40. **(age 15 · value low · lapsed@0917-1659) Folding shut onto the outer
    display is unseen.**
41. **(age 15 · value low · lapsed@0917-1659) Housekeeping:** the
    `GrMob_Foldable` AVD is still installed (verified in `~/.android/avd`).
42. **(age 15 · non-goal · lapsed@0917-1659)** More than one fold; a static
    HTML export of a TwoPane.
43. **(age 14 · value medium · lapsed@0917-2309) The round-two widgets have
    never been seen on a device:** Screen.Floating and the FAB (4.22), the
    QRCode module seams (a hairline between runs could stop a scan), and
    Countdown/Stopwatch (4.24). They sat in "Not verified" sections; the device
    pass (2026-0918-0713) covered 4.25–4.32 and 5.8. SliderRow was seen in
    2026-0918-0910's accent check, and PINInput was redone and seen since.
44. **(age 10 · value medium) Examples should adopt the shipped widgets.**
    `examples/chat` hand-builds what `comps.MessageThread` does (Scroll +
    Column + RoleLog, `main.go:148`), and `examples/signup` never got
    `PINInput`. Both change shotclaims, so they land together. The signup half
    was first written in 2026-0917-1935 and dropped; this doc raised the chat
    half.
45. **(age 8 · non-goal)** A native time wheel; a sheet or Done on TimePicker.
46. **(age 7 · non-goal)** Heatmap as a continuous gradient; a Sequential ramp
    interpolated from `Primary`.
47. **(age 6 · value low · delete?) A sixth low-hanging-fruit round.** No
    candidates across four carries. Proposed for deletion until one exists.
48. **(age 6 · non-goal)** A year-wide scrolling `CalendarHeatmap`; a
    caption-flip "Copied ✓" on CopyButton; a separate `Alert` widget (Banner
    is it).
49. **(age 5 · non-goal)** A two-pane layout on the natives; restructuring
    lesson bodies into guide and demo halves.
50. **(age 4 · non-goal)** Making iOS keep focus on every submit by default.
51. **(age 3 · value medium) The first hardware key after launch is lost on
    the iOS 26.5 simulator.** Every keyboard UI test works around it with
    `primeKeyboard`. Unchecked on a real iPad.
52. **(age 3 · non-goal)** A right-padding gutter or a toolbar header for code
    blocks.
53. **(age 2 · non-goal)** A smaller minimum for Buttons in general on
    Android. Only CopyButton opts out.
54. **(age 1 · value low) iOS UIKit field: a queued key during the caret
    correction.** Not observed in 6 correction writes, but the window exists
    after a change strictly after the caret. The candidate fix is to drain
    through `input.inputDelegate` (`_UIKeyboardStateManager`)
    `selectionWillChange/DidChange` before `replace`, merging any keys it
    delivers with `rebaseEdit(current, flushed, next)`.
55. **(age 0 · value low) `core.ScrollIntoView` inside an iOS `core.List`.**
    Verified: `GrMobList`'s `ScrollViewReader` (Renderer.swift:1454) does not
    inject `\.grMobScrollProxy`; only `GrMobScroll` does (1316, 1355). Rows
    are keyed by `rowKey` (String) while `GrMobBringIntoView` scrolls by
    `viewID`. Measure before claiming it.
56. **(age 0 · value low) iOS Paragraph link colours on the iOS 17 floor.**
    Measured only on 26.5; the `.tint(firstLink)` fallback remains.
57. **(age 0 · value low · by design) The web's thread place-keeping applies
    only when the List is its own scroll box.** A List inside another
    scrolling ancestor keeps the browser's behaviour.
58. **(age 0 · non-goal)** A reported scroll offset from the hosts. The thread
    needed only the two List props (see `core/list_start.go`).

Read by value instead: **high** none · **medium** 1, 2, 25, 26, 29, 35, 36,
43, 44, 51 · **low** 3–7, 9, 10, 12, 13, 15–20, 22, 23, 27, 30–34, 37–41, 47,
54–57 · **non-goal** 8, 11, 14, 21, 24, 28, 42, 45, 46, 48–50, 52, 53, 58.
