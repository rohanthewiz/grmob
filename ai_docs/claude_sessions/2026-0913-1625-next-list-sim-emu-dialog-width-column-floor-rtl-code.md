# Next list on the simulator and emulator: dialog width, a Column floor, code in RTL

**Session:** https://claude.ai/code/session_013PPKnYoYbsVv3vW7oAXA6p
**Date:** 2026-09-13 16:25
**Branch:** master (after 38d593f)

## 1. The asks

1. `/sl` loaded "Native boxes, code blocks, and a sliding Drawer" (plus 38d593f,
   committed after that doc: Modal content taller than its space scrolls).
2. "Work the Next list, simulator/emulator only" — no real devices.
3. `/sw`: this doc, commit, push.

## 2. How the work was split

Two forks in git worktrees did the web-only items (no simulator/emulator); the
main session did everything native on the iOS 26.5 simulator (iPhone 17 Pro)
and the Android emulator (Medium Phone API 36.1).

| Stream | Items | Who | Commit |
|---|---|---|---|
| INK_FACE Latn, shot-claim clipping check, strict mkdocs | 18, 19 | fork | 83db117 (merge) |
| Calendar ARIA grid, today as aria-current=date | 3 | fork | e73c739 (merge) |
| Android dialog width for self-sized content | 27 | main | c95caa9 |
| iOS Column min-height floor | 25 | main | 6aac336 |
| CodeEditor left to right in RTL, all targets | new | main | 2be021e |
| iOS RichTextEditor height | new | main | 6bf5eda |
| Box-chain commit compared lesson by lesson | 26 | main | — (measurement) |
| Sideways nesting, List laziness on iOS | 15, 16 | main | — (measurement) |
| Emulator checks from item 2 | 2 | main | — |
| Android DataTable rows vanish | new | main | **not committed** (see 5) |

Both merges were clean; `go test ./...` passed after each.

## 3. Committed changes (main session)

### 27. A Modal child that sizes itself leaves Android's platform dialog width (c95caa9)
- Measured: every Compose Dialog window is 840px = 320dp wide on the phone, in
  portrait and landscape. DatePicker's card (MaxWidth 360) stopped at ~304dp;
  ActionSheet's card (Width 100%) was a floating 320dp card. iOS and the web
  draw 360 / edge to edge.
- `GrMobModal`: a direct child declaring Width or MaxWidth sets
  `DialogProperties(usePlatformDefaultWidth = false)`; MinWidth does not.
  `ColumnChildren(centreCapped = true)` centres a capped child
  (`fillMaxWidth().wrapContentWidth(CenterHorizontally)`), as the web overlay
  and the iOS sheet centre it.
- Emulator: 6.4 Modal and 6.6 Dialog keep 320dp; 6.7 sheet spans the screen and
  a tap above it still dismisses; 4.9 card 360dp centred in both orientations;
  4.17's Sort and ⋯ menus open full width and pick.
- ActionSheet's placement table documents the rule.

### 25. iOS Column children floor at their content height (6aac336)
- `GrMobMinContent.floorsHeightAtContent(node)` decides; FlexChildren sends
  `.infinity` (floor at base, clamped by minMains) or 0 as the vertical
  `GrMobFlexMin` value. The base height is what the layout already measures.
- Yes: Text, Button, Spacer, points Height, containers of only those. No: any
  Overflow but visible, scrolling types (Scroll, List, TextArea, CodeEditor,
  TextGrid, RichTextEditor), percentage Height, everything else. The scroll rule
  keeps nested scrolls absorbing the squeeze.
- Throwaway probe app, with and without the floor: a 220pt Column with a title
  over a FlexGrow Scroll — without it the title is crushed onto "Scrolled row 1",
  with it the title is whole. `ios/verify/mincontent.swift` checks every row of
  the verdict table.

### CodeEditor reads left to right in RTL (2be021e)
- Found under an Arabic app locale on the emulator (4.18): code blocks were
  right-aligned and bidi-reordered ("}comps.Drawer", ",()Open:").
- Compose: `LocalLayoutDirection provides Ltr` around the editor Row,
  `TextDirection.Ltr` on the text style. UIKit: views
  `.forceLeftToRight`, text view `.left`, a left-to-right paragraph style on
  every line. Web: `dir="ltr"` on the editor box in htmlout and the runtime.
- Emulator under locale ar: fixed. `go test ./htmlout`, `sh wasm/verify/run.sh`,
  `sh ios/verify/run.sh` pass. **iOS RTL not seen** (see 6).

### iOS RichTextEditor takes its document's height (6bf5eda)
- Found in the item-26 screenshots: 4.14's editor was a bar a few points tall
  in both builds. Scroll-enabled UITextView, no `sizeThatFits`.
- Now CodeEditor's contract: unbounded height → content height wrapped at the
  offered width; finite → default. Simulator: editor 157pt, whole sample doc.
- `RichTextEditor.MinHeight` doc says core.MinHeight is web-only (the platform
  table in styling-and-theming.md already did).

## 4. Measurements (no code)

- **26, box-chain commit (e06a4a2) vs its parent (b46152f):** XCUITest probe
  shot every lesson at the top and after three held drags, both builds (same Go
  bind, scratch worktree). First frames: the only visible change is 1.4's
  Justify container, which now spans the width (the fix — before, it hugged
  A/B/C). Others (4.11, 2.3, 4.14, …) are 1–2pt shifts from border insets.
  Scrolled frames all differ because offsets accumulate; not aligned.
- **15:** a horizontal Scroll inside a horizontal Scroll on iOS — a drag inside
  the inner moves only the inner (dx −110); a drag on the outer text moves the
  outer. Works.
- **16:** a List with no Height in a scrolled page on iOS materialises every
  row: 400 of 400 in the accessibility tree at launch, "Row 399" exists. Not
  lazy; Android's unbounded arm documents the same.
- **Emulator (item 2):** Drawer under locale ar (☰ and panel at the right,
  75% = 625px of 834, ✕ mirrored, Starred picks); 6.6 one tap on a row's text
  toggles switch and checkbox; 4.17 SearchableSelect — keyboard on focus, "Ca"
  lists 2, pick fills and hides keyboard and list, Enter moves to City with the
  keyboard up; 4.16 radios — tap moves the check, disabled option refuses, rows
  are `selectableGroup`/`Role.RadioButton` in source (uiautomator shows View);
  StepIndicator scrolls sideways to step 4, a done step taps back, a future one
  does not; Timeline rails meet between rows; 4.6 GroupedList draws.
- **4.7 AppBar OnBack:** the arrow runs OnBack. System back also runs it (as
  AppBar documents) and never pops — the lesson cannot be left by system back.
- **iOS Drawer (LTR):** opens, Starred picks, "Showing Starred".

## 5. In flight: Android DataTable rows (not committed)

- 4.6's DataTable on the emulator: header then pager, no rows. DataTable's rows
  List has FlexGrow(1); Compose `weight` in a Column with an infinite maximum
  and no minimum measures the child at 0, and GrMobList then takes its bounded
  arm with no height. Rebinding the AAR at HEAD did not change it.
- Fix written in `android/.../Renderer.kt`: `LocalGrMobUnboundedHeight`. True
  inside GrMobScroll's content; cleared by a points Height (RenderNode), a grow
  child given `heightIn(min = viewport)`, a stretched Row's children
  (IntrinsicSize.Max), and by GrMobModal when a growing child has the window as
  its minimum. `ColumnChildren` skips `weight` when true. A local, not
  BoxWithConstraints, because a stretched Row asks children for intrinsics.
  `android/verify/sources.sh` passes.
- Verification running at save time: a sweep of all 57 lessons × 6 frames with
  HEAD's runtime, then with the change, then an ImageMagick diff. While it runs
  the working file is HEAD's; the change is in the session scratchpad as
  `Renderer.kt.new` and the sweep script copies it back before the second pass.
  **If the sweep did not finish, restore that file before anything else.**
- The first two sweep attempts were lost: zsh does not word-split `$L` (one
  bogus "lesson"), then the OS killed both device jobs for memory.

## 6. Process notes

- Memory: simulator + emulator + `xcodebuild test` + a sweep + Gradle daemons at
  once got both background jobs killed (24 GB, ~10 GB compressed). Run one
  device job at a time; `./gradlew --stop` after installs.
- zsh: `for id in $L` does not split; write the list out or use `${=L}`.
  `$(seq …)` does split.
- Android: `wm user-rotation lock 1|0` rotates (the `settings put system
  user_rotation` form did nothing this time). `debug.force_rtl` did not flip the
  app; `cmd locale set-app-locales com.grmob.app --locales ar` did (reset with
  `--locales ""`). Demos are below the fold, so a uiautomator dump sees only
  what is on screen — scroll to the element before concluding.
- iOS: `-AppleLanguages (ar)` does not flip the app — it ships no ar
  localization — so iOS RTL is still unmeasured.
- XCUITest probes: `TEST_RUNNER_PROBE_*` env vars; add the file, `xcodegen
  generate`, run `-only-testing`, delete the file, regenerate. Use
  `.matching(identifier:).firstMatch` — two nodes share "Open navigation".
- Compare script: crop status/home bars, `magick compare -metric AE -fuzz 3%`.
  Only unscrolled frames compare meaningfully.
- Old agent worktrees remain in `.claude/worktrees/` (six, all merged).

## Next

**Legend:** *age* is how many session docs ago the item was first raised,
counted from this doc (0 = raised here; `≥ k` = older than the last rebuild's
window). *value*: **high** = worked around today or a second consumer has
arrived; **medium** = blocks one named thing or is a visible defect; **low** =
nobody has hit the gap yet. Sorted by age, oldest first; new items last.

1. **(age ≥8 · value high) Tag a release.** v0.3.0 is 86 commits behind with
   this doc. Pushing a tag needs the user's yes. Still open: `go run
   github.com/rohanthewiz/grmob/cmd/grmob@latest new` on a clean machine without
   `-replace`.
2. **(age ≥8 · value high) Lessons on hardware, and checks still open.** Done
   this session on the emulator/simulator: see section 4. Still open:
   - Real devices for everything (deferred by the user).
   - Screen readers: radio and StepIndicator done-step semantics (uiautomator
     reports done steps clickable=false though taps work), combobox active
     option, "pop-up" on Menu/DatePicker triggers, aria-current/selected on
     current items, Calendar's grid on the web, CodeEditor toolbar role on
     Compose, SearchableSelect natives reading field + list + status,
     AccessibilityHidden confining TalkBack/VoiceOver behind a Drawer.
   - iOS: sheet `Dialog`, ActionSheet filler, Drawer slide by eye and under
     Reduce Motion, RTL for Drawer and CodeEditor (needs a way to force layout
     direction), editing in the sideways code editor (caret reveal, selection
     handles), `banded` and a drag on 4.6's list.
   - `Screen.Footer` pinning; Drawer: 100% layers in a pinned-height ZStack, the
     panel's `Height 100%` in an HStack, `core.Focus` on a Button, a hardware
     keyboard reaching a shut panel; predictive back from contents; MaxWidth
     where it binds on Android (tablet) and a capped stretched child in RTL.
3. **(age ≥8 · value low) ", done" and native ", today".** The web now says
   today with aria-current="date" and Calendar is one grid tab stop (item 3,
   e73c739). Natives still append English ", today"; StepIndicator keeps ",
   done" (documented: no ARIA state fits). PageUp/PageDown do not change month;
   composite arrow keys are not flipped for RTL.
4. **(age ≥8 · non-goal) Tracked-Go-file count sentences** in `wasm/verify` are
   hand-edited when Go files are added. Still 505.
5. **(age ≥8 · non-goal) Windowing.** Declined in `non_goals.md`.
6. **(age ≥8 · non-goal) Rename `docs/components.md` to `comps.md`.**
7. **(age ≥8 · non-goal) Rewrite `components` in the older plans.**
8. **(age ≥8 · non-goal) Trim the copied Android shell's permissions.**
9. **(age ≥8 · non-goal) Replace the iOS usage strings further.**
10. **(age ≥8 · non-goal) C4 `Carousel`** until the host reports a scroll offset.
11. **(age 6 · non-goal) A parent that gains `onBack` after its descendants
    outranks them** on Android; the web ranks by document order.
12. **(age 6 · non-goal) An AppBar outside the Navigator** with a custom
    `OnBack` is outranked by the Navigator's pop on Android.
13. **(age 5 · non-goal) Forward does not re-open a screen left by browser back.**
14. **(age 4 · non-goal) Sticky headers, placement animation and `OnEndReached`
    in a List with no viewport** on Compose. Give the List a Height.
15. **(age 2 · value low) The cost of a paused TimelineView per node** in
    SwiftUI's box chain (`GrMobMotion`). Not profiled.
16. **(age 2 · non-goal) MaxWidth with a growing sibling on the natives.**
17. **(age 2 · non-goal) The typed-hash fold's `history.length` fallback.**
18. **(age 2 · non-goal) A page's own `pushState` while a claim is on screen.**
19. **(age 2 · non-goal) A Drawer's shut panel is composed on the natives.**
20. **(age 0 · value high) Finish the Android DataTable fix** (section 5): read
    the sweep diff (`asweep/diff/report.txt`), confirm 4.6 draws rows and no
    other lesson regressed, check 6.7's filler and a Screen{Fill, Scroll} form,
    commit. If the sweep did not finish, restore `Renderer.kt.new` first.
21. **(age 0 · value medium) 4.7 traps Android system back.** The demo AppBar's
    OnBack never pops and system back runs it, so nothing but "‹ Contents"
    leaves the lesson. Options: pop on a second back while the note shows, or
    demo OnBack in a pushed screen.
22. **(age 0 · value medium) MinWidth/MinHeight on the natives.** Documented
    web-only, but DatePicker (MinWidth 300), the rich-text link prompt (280) and
    RichTextEditor (MinHeight) rely on them.
23. **(age 0 · value low) A Row in a horizontal scroll with FlexGrow children**
    has the same zero-weight collapse on Compose; the DataTable fix covers
    Columns only. Rows inside a bounded LazyColumn with growing children too.
24. **(age 0 · value low) Shot-claim field-text check** makes no RTL or
    letter-spacing adjustment (RTL fields use the box check), and nothing runs
    `mkdocs build --strict` in CI.
25. **(age 0 · non-goal) A List with no Height in a scrolled page is not lazy**
    on iOS or Android (item 16's answer). Give the List a Height to virtualise.
26. **(age 0 · value low) Remove the merged agent worktrees** in
    `.claude/worktrees/` (six).
