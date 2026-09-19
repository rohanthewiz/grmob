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

- **iOS UIKit field: a queued key during the caret correction** (narrowed,
  carried). It was not observed in 6 correction writes, but the window
  exists after a change strictly after the caret. The candidate fix is to
  drain through `input.inputDelegate` (`_UIKeyboardStateManager`)
  `selectionWillChange/DidChange` before `replace`, merging any keys it
  delivers with `rebaseEdit(current, flushed, next)`.
- **iOS Paragraph link colours on the iOS 17 floor** (new). They were
  measured only on 26.5; the `.tint(firstLink)` fallback remains for older
  OSes.
- **`core.ScrollIntoView` inside an iOS `core.List`** (new, low-hanging).
  GrMobList now has a ScrollViewReader and could inject
  `\.grMobScrollProxy`, but row ids are now String (`rowKey`) while
  `GrMobBringIntoView` scrolls by `viewID`. Measure before claiming it.
- **`examples/chat` could use `comps.MessageThread`** (new). Its tests pin
  the hand-built Scroll + Column + RoleLog structure.
- **The web's thread place-keeping applies only when the List is its own
  scroll box** (new, by design). A List inside another scrolling ancestor
  keeps the browser's behaviour.
- **A sixth low-hanging round** (carried shape). No candidates beyond the
  above.
- **F-keys through GameController have never reached the app from
  XCUITest** (carried; needs hardware).
- **The first hardware key after launch is lost on the iOS 26.5 simulator**
  (carried; needs a real iPad).
- *Non-goal, declined:* a smaller minimum for Buttons in general on Android.
  Only CopyButton opts out (carried).
- *Non-goal, declined:* making iOS keep focus on every submit by default
  (carried).
- *Non-goal, declined:* a two-pane layout on the natives (carried).
- *Non-goal, declined:* restructuring lesson bodies into guide and demo
  halves (carried).
- *Non-goal, declined:* a year-wide scrolling `CalendarHeatmap` (carried).
- *Non-goal, declined:* a caption-flip "Copied ✓" on CopyButton (carried).
- *Non-goal, declined:* a separate `Alert` widget. Banner is it (carried).
- *Non-goal, declined:* Heatmap as a continuous gradient (carried).
- *Non-goal, declined:* a Sequential ramp interpolated from `Primary`
  (carried).
- *Non-goal, declined:* a native time wheel, and a sheet or Done on
  TimePicker (carried).
- *Non-goal, declined:* a right-padding gutter or a toolbar header for code
  blocks (carried).
- *Non-goal, declined:* a reported scroll offset from the hosts. The thread
  needed only the two List props (decided this session; see
  `core/list_start.go`).
