# Native boxes, code blocks, and a sliding Drawer

**Session:** https://claude.ai/code/session_013PPKnYoYbsVv3vW7oAXA6p
**Date:** 2026-09-13 11:25
**Branch:** master (18 commits after 987e755)

## 1. The asks

1. `/sl` loaded "Fifteen Next items: a combobox, inert, Spin, and a typed hash".
2. "Work the Next list but for now defer the testing of a device, just use the
   simulator / emulator."
3. `/sw`: save this doc, commit all, push.

## 2. How the work was split

Four forked agents in git worktrees (`.claude/worktrees/`), told not to touch the
simulator or emulator; merged into master with `--no-ff`. The main session did
the native items on the iOS 26.5 simulator (iPhone 17 Pro) and the Android
emulator.

| Stream | Items | Who | Merge |
|---|---|---|---|
| Docs and text | 26, 27, 28, 30, 31 | fork | b46152f |
| INK_FACE test | 29 | fork | 766c45b |
| DatePicker Tab, current-item | 21, 4 | fork | 97efe31 |
| Reduced motion, Drawer slide | 24, 25 | fork | 16ce79f |
| iOS Spinner, iOS code blocks, Android dot | 19, 20, 22 | main | e06a4a2, 8d5ef3e |
| Scaffold Row inset | 3 | main | 2f41cbe |
| Kept iOS nested-scroll test | 18 | main | 45d390d |
| Simulator/emulator checks | part of 2 | main | — |

Conflicts, as last session, only in generated or counted files: `docs/api/*`
(take a side, re-run `go run ./internal/apidoc/gen`) and the tracked-Go-file
count sentences in `wasm/verify/repowalks_test.go` / `timings_test.go`
(502 → 503 on one side, 504 on the other → 505, matching `git ls-files '*.go'`).

## 3. Main-session items

### 19. iOS Spinner collapsed (e06a4a2)
Found with a throwaway XCUITest probe (deep link, tap, `XCUIScreen` shots,
`debugDescription` dump; deleted before commit).
- **Cause 1:** `GrMobBoxModifier` applied `grMobDimension` (fixed Width/Height)
  *after* background, gestures, clip, border and shadow. GrMobFlexLayout hugs
  its children (a childless Box reports `.zero`), so a fixed-size box was
  painted around its content and the frame around it held nothing: a 24pt ring
  drawn as a hollow speck, the 6pt dot not at all.
- **Fix:** padding → dimension frames → background/gestures/clip/border/shadow →
  rotate → motion → margin. CSS border-box, and Compose's existing order.
- **Cause 2:** `frame(width:)` / `frame(height:)` / `containerRelativeFrame` were
  not passed `alignment`, so SwiftUI centred smaller content: the dot sat at the
  ring's centre and spinning moved nothing. Alignment now passed on all three.
- Simulator, lesson 4.15: ring and dot draw; the dot is at a different angle in
  two frames 250ms apart.

### 22. Android dot clipped (e06a4a2)
- Compose `Modifier.border` and SwiftUI's overlay reserve nothing, so content at
  the edge sat under the stroke (and a round clip cut the dot in half on
  Android). Both natives now inset content by a *drawn* border's width (same
  two-part guard as the border: width > 0 and a colour).
- Kotlin: `boxModifier` padding + `borderInset`. Swift:
  `GrMobStyle.contentInsets` / `borderInset`; `GrMobMinContent.capped` and
  `outerInsets` count it; `ios/verify/mincontent.swift` checks bordered and
  uncoloured cases.
- Emulator, lesson 4.15: dot whole, inside the rim, moving between frames.

### 20. iOS code blocks blank (8d5ef3e)
Compositor screenshot (`simctl io screenshot`) showed the bars were real, not an
XCUIScreen artefact. Four faults, each measured:
1. A scroll-enabled UITextView has no intrinsic height → a CodeEditor with no
   Height got its padding only. `GrMobCodeEditorRepresentable.sizeThatFits`
   now returns content height when the proposed height is nil/infinite, `nil`
   (fill) when finite — Compose's `verticalScrollWhenBounded` contract.
2. Plain `UITextView()` is TextKit 2 and ignored the container → `UITextView(usingTextLayoutManager: false)`.
3. A file-logged probe showed the container narrowed to the view's 310pt after
   frame assignment even with `widthTracksTextView = false` →
   `GrMobUnwrappedTextView.holdContainerUnbounded()` before and after
   `super.layoutSubviews()` and after every frame set. Measurement goes through
   `layoutManager.usedRect`, not `sizeThatFits`.
4. With contentSize widened, a swipe revealed an empty band: UITextView draws
   text only inside its own width → the text view is now as wide as its longest
   line inside a horizontal `UIScrollView` (`sideways`); gutter outside it;
   `revealCaret()` scrolls sideways on selection change.
- Simulator, lesson 4.15: five whole lines, no wrap; a swipe shows the rest of
  each line.

### 3. Scaffold Row inset (2f41cbe)
`DefaultTheme.Components.Row` has 16pt left/right padding; inside the
template's `Padding(24)` Column the −/+ buttons were indented. Template Row gets
`core.Padding(0)` with a comment. `go test ./cmd/grmob` passes.

### 18. Kept iOS nested-scroll test (45d390d)
`ios/GrMobUITests/TutorialScrollUITests.swift` (needs the tutorial bound;
run alone with `-only-testing`). Opens `grmob://lesson/4.3`, scrolls until
"Romans" is hittable, asserts the six outline rows stack, the first is below the
demo caption, and the first key point starts below the last row.
Negative control: `.frame(height: 60)` on GrMobList fails it ("the outline's
last row never came on screen"); unmodified passes.

## 4. Fork items

- **26 (bbf6ff2):** checkbox leads when the row is the thing marked, trails when
  it is a named option. CheckboxRow doc example and table, docs/components.md,
  lesson 2.6 key point.
- **27 (3e3b693):** no retake can show the old claimed line whole (43 columns in
  a 38-column block). Claim is now `"func Profile(ctx *core.Context)"`;
  `shot.mjs` refuses a PNG where a claimed string is clipped by a scroller
  (`wasm/shots/claims/main.go`, `shoot.sh`). Form-field text checked by box.
- **28 (e76f3c0):** `grmob new`'s closing message names `grmob web`.
- **30 (e876d98):** comps → 134-line index + six topic pages; core-style →
  `core-style.md` (Style struct) + `core-style-props.md`. mkdocs nav nested.
- **31 (f046b9d):** Stepper doc wording.
- **29 (2485ea1):** Preferences write moved to `wasm/verify/inkface.mjs`
  (merges rather than replaces) with seven tests in `inkface_test.mjs`;
  `compositeMeasuredOnChrome = 152` beside hero's tolerance, logs on a different
  major. This Mac's Chrome is now 153, so it logs.
- **21 (7ea03d9):** the runtime gave a `role=button|link` Box/Row with onClick a
  tab stop only inside a toolbar. Now outside one: `tabindex=0`, Enter (and
  Space for buttons); disabled and aria-hidden get none; toolbar ones keep the
  roving order. Also fixes Disclosure headers, Rating stars, StepIndicator done
  steps, Calendar days, tappable StaticMaps. CDP: lesson 4.9 trigger reached on
  Tab 39, Enter/Space open it.
- **4 (0af96cb):** `core.CurrentKind` (`CurrentPage`, `CurrentStep`,
  `CurrentTrue`), `Style.AccessibilityCurrent`, `core.AccessibilityCurrent`.
  `aria-current` on both DOM targets; Compose `selected`, SwiftUI `.isSelected`
  (an explicit AccessibilitySelected wins). BottomBar, Drawer (via
  `ListRow.Current`), StepIndicator, ActionSheet dropped their ", selected" /
  ", current" suffixes; ", done" and Calendar's ", today" stay.
- **24 (9fa1c52):** reduced motion snaps Transition everywhere
  (`core.ReducedMotionCSS`; SwiftUI `accessibilityReduceMotion`; Compose already
  obeys "Remove animations"). Spin keeps turning (WCAG 2.3.3 essential; iOS's
  own indicator does). Also fixed htmlout writing style rules outside `<head>`.
- **25 (dd6378f):** `core.Translate(x, y)` (px, bare, % of own box; x from the
  leading edge, mirrored in RTL), applied outside Rotate/Spin; SwiftUI merges
  Spin and Translate into `GrMobMotion` (one more modifier layer crashed swiftc).
  `Overflow("hidden")` now clips on both natives. Drawer panel stays displayed,
  clipped, AccessibilityHidden + Inert while shut, moved its own width off the
  leading edge; opens 250ms ease-out, closes 200ms ease-in; shut scrim has no
  fill or tap.

## 5. Simulator / emulator checks (item 2, partial)

- **Android system back, 4.18:** drawer open → back closes it (lesson stays) →
  back again lands on contents. Screenshots confirm.
- **Android double back from 4.15:** first back to contents, second leaves to
  the launcher (back on contents); no lesson opens.
- **Android Drawer slide (16ce79f build):** shut panel passes the ☰ tap; a frame
  100ms in shows the panel partway; rows present; tapping Starred shows
  "Showing Starred"; reopens after close. "Remove animations": the 100ms frame
  is byte-identical to the 1.1s frame (snap), and the Spinner still turns.
- **iOS Drawer:** rows hittable, Starred tap works, reopens after close; the tree
  shows `'Inbox', Selected` (item 4). The slide itself could not be seen:
  XCUITest waits for idle before its screenshot, and a `simctl io screenshot`
  loop (~170ms/frame) caught no mid-slide frame.
- **MaxWidth in landscape, 4.9 DatePicker:** iOS card is 360pt (cap binds);
  Android card ~287dp (cap does not bind).
- **Found:** Modal content taller than its space. iOS rides a sheet with
  `.presentationDetents([.medium, .large])` and no scroll, so the Calendar is
  squeezed (vertical flex has no min-content floor) and the day-marker dots
  overlap their numbers, portrait and landscape. At 987e755 (baseline worktree
  build) the dots were not drawn at all — the same childless-box bug as 19. The
  inline Calendar draws them correctly. Android's Dialog in landscape clips the
  sheet after week 3 and a drag does not scroll.

## 6. Verification on the final tree (16ce79f)

- `go test ./...` — no failures.
- `sh wasm/verify/run.sh` — passes.
- `sh ios/verify/run.sh` — every stage OK (including Release optimisation and
  the iOS SDK type-check).
- Android: `android/build.sh ./examples/tutorial` + `./gradlew installDebug
  --offline` succeed (app and runtime compile); `android/verify/sources.sh`
  passed on the first merge.
- `TutorialScrollUITests` passes on the simulator.

Process notes:
- Probes: an untracked `ios/GrMobUITests/ZZProbeUITests.swift` driven by
  `TEST_RUNNER_PROBE_*` env vars (xcodebuild forwards `TEST_RUNNER_` prefixed
  vars), writing PNGs and `debugDescription` to the scratchpad. Run xcodegen after
  adding/removing it. Force `XCUIDevice.shared.orientation` every run — it
  persists between runs.
- The simulator's app can write host paths; a `FileHandle` append from the
  runtime was the reliable log (`simctl spawn booted log show` returned nothing).
- `sips` crops/resizes screenshots; ImageMagick is installed, ffmpeg is not.
- Android: uiautomator dump + `bounds` for taps; `settings put system
  user_rotation 1` for landscape; `settings put global animator_duration_scale 0`
  for Remove animations (restored to 1 and portrait afterwards).
- A baseline build: `git worktree add --detach <scratch> 987e755`, bind and
  xcodegen there, separate derived data. Removed afterwards.
- `go test ./... | grep -v ok | tail; echo $?` reports grep's status, not
  go's — read the output.
- Merge conflicts only in `docs/api/*` and the count sentences, again.

## Next

**Legend:** *age* is how many session docs ago the item was first raised,
counted from this doc (0 means this doc raised it; `≥ k` means older than the
last rebuild's window). *value* is the payoff of doing it, not the effort:
**high** means something is worked around today or a second consumer has
arrived; **medium** means it blocks one named thing or is a visible defect;
**low** means nobody has hit the gap yet. Sorted by age, oldest first; new items
last.

1. **(age ≥7 · value high) Tag a release.** v0.3.0 is the latest tag, now 73
   commits behind. This session adds native box sizing, the iOS code editor,
   reduced motion, Translate and a sliding Drawer, current-item state, and
   Tab-reachable controls. Pushing a tag needs the user's yes. Still open: run
   `go run github.com/rohanthewiz/grmob/cmd/grmob@latest new` on a clean machine
   without `-replace`.
2. **(age ≥7 · value high) Lessons on hardware, and the simulator/emulator
   checks still open.** Done this session on the emulator/simulator: Android
   system back on 4.18, double back, Android drawer slide and Remove animations,
   iOS drawer taps, 4.9 MaxWidth landscape. Still open:
   - Real devices for everything below (deferred by the user).
   - settings-row one-tap, `Screen.Footer` pinning, the iOS sheet `Dialog`
   - ActionSheet filler placement, Timeline row stretch, horizontal
     StepIndicator scroll, radio semantics
   - CodeEditor's toolbar role on Compose, a Modal inside a ListRow's trailing
     Row (Menu); editing in the new iOS sideways editor (caret reveal while
     typing past the edge, selection handles across the scroll view)
   - SearchableSelect: Next action and keyboard dismiss on pick; natives still
     read field + list + status
   - Drawer: 100% layers in a pinned-height ZStack, the panel's `Height 100%` in
     an HStack, AccessibilityHidden confining TalkBack/VoiceOver, `core.Focus`
     on a Button; the iOS slide by eye; iOS Reduce Motion snapping it; RTL on
     both natives (placeRelative, GeometryEffect hit-testing); a hardware
     keyboard reaching a shut panel's rows (documented cost)
   - Predictive back previewing home from contents only; 4.16 AppBar `OnBack`.
   - Unbounded arms by eye: 4.6 `grouped` and `banded` on Android; `banded`
     and a drag on a 4.6 list on iOS.
   - MaxWidth where it binds on Android (tablet), the chapter 4/6
     ActionSheet/Menu panels (520px), a Drawer with a percentage Width, a
     capped stretched child in RTL.
   - Screen readers: combobox active option (VoiceOver/NVDA), "pop-up" on Menu
     and DatePicker triggers, `aria-current` / selected on current items.
3. **(age ≥7 · value low) ", done" and ", today" remain English suffixes.** No
   platform has a completed or current-date state; Calendar days are ~30 tab
   stops on the web where ARIA's grid pattern would make one.
4. **(age ≥7 · non-goal) Tracked-Go-file count sentences** in `wasm/verify` are
   hand-edited when Go files are added. Now 505.
5. **(age ≥7 · non-goal) Windowing.** Declined in `non_goals.md`;
   `TestWhatWindowingWouldSave` is the profile.
6. **(age ≥7 · non-goal) Rename `docs/components.md` to `comps.md`.**
7. **(age ≥7 · non-goal) Rewrite `components` in the older plans.**
8. **(age ≥7 · non-goal) Trim the copied Android shell's permissions.**
9. **(age ≥7 · non-goal) Replace the iOS usage strings further.**
10. **(age ≥7 · non-goal) C4 `Carousel`** until the host reports a scroll offset.
11. **(age 5 · non-goal) A parent that gains `onBack` after its descendants
    outranks them** on Android; the web ranks by document order.
12. **(age 5 · non-goal) An AppBar outside the Navigator** with a custom
    `OnBack` is outranked by the Navigator's pop on Android.
13. **(age 4 · non-goal) Forward does not re-open a screen left by browser back.**
14. **(age 3 · non-goal) Sticky headers, placement animation and `OnEndReached`
    in a List with no viewport** on Compose. Give the List a Height.
15. **(age 2 · value low) Sideways nesting on iOS is unmeasured.** A
    `Horizontal()` Scroll/TextGrid/CodeEditor inside a `Horizontal()` Scroll.
    The iOS CodeEditor is now itself a horizontal UIScrollView, which makes this
    case more likely to matter.
16. **(age 2 · value low) Whether GrMobList's LazyVStack stays lazy in a
    scrolled page on iOS.** Needs a long List with no Height; the tutorial has
    none (home's List is the page).
17. **(age 1 · value low) The cost of a paused TimelineView per node** in
    SwiftUI's box chain, now inside `GrMobMotion`. Profile 4.3/4.6 with
    Instruments before adding more always-on modifiers.
18. **(age 1 · value low) `GRMOB_INK_FACE`'s `Zyyy` key** rests on Chrome's
    defaults; a `Latn` key was not tested. hero's composite images were taken on
    Chrome 152 and this Mac now runs 153 (the test logs it).
19. **(age 1 · value low) Shot-claim clipping check** has not been seen to fire
    for a string cut by the bottom of the frame, and checks form fields only by
    their outer box. The mkdocs build with the nested nav is unbuilt.
20. **(age 1 · non-goal) MaxWidth with a growing sibling on the natives.** A
    FlexGrow child in a Row whose cap binds keeps its share. Documented.
21. **(age 1 · non-goal) The typed-hash fold's `history.length` fallback**
    misreads a push at a browser's session-history cap.
22. **(age 1 · non-goal) A page's own `pushState` while a claim is on screen**
    makes the runtime push a second entry.
23. **(age 1 · non-goal) A Drawer's shut panel is composed on the natives**, so
    its rows cost layout while closed and a hardware keyboard can reach them.
    Documented on Drawer; `Disabled` would grey the ✕ during the slide-out.
24. **(age 0 · value med) Modal content taller than its space.** iOS sheet
    (`.presentationDetents([.medium, .large])`, no scroll) squeezes the
    DatePicker's Calendar so its day-marker dots overlap the numbers, portrait
    and landscape; Android's Dialog in landscape clips after week 3 with no
    scroll. A plain ScrollView wrapper is wrong — ActionSheet's content relies on
    a `FlexGrow(1)` filler inside the Modal. Scroll only when the content is
    taller (SwiftUI `ViewThatFits` or a measured `.height` detent; Compose
    `verticalScrollWhenBounded` with the filler's case checked).
25. **(age 0 · value low) Vertical flex has no min-content floor on iOS.**
    GrMobFlexMin floors Rows only, so a squeezed Column shrinks children below
    their content and they overlap (item 24's mechanism). CSS's
    `min-height: auto` would keep them whole and overflow instead.
26. **(age 0 · value low) The box-chain move changes every native box with a
    fixed size and smaller content.** Background, border, shadow and touch
    target now cover the declared frame, content sits top-start, and bordered
    boxes inset by 1–2pt. Checked on 4.15, 4.18, 4.9, 4.3; other lessons not
    compared against a baseline.
27. **(age 0 · value low) `DatePicker`'s card on Android is ~287dp in
    landscape**, below its 360 cap, where iOS reaches 360. Decide whether the
    Compose card should grow to the cap like the iOS sheet.
