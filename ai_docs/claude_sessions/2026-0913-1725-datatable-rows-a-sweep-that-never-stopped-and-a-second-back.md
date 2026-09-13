# DataTable rows, a sweep that never stopped, and a second back

**Session:** https://claude.ai/code/session_013PPKnYoYbsVv3vW7oAXA6p
**Date:** 2026-09-13 17:25
**Branch:** master (5ede455 → d4bb256)

## 1. The asks

1. `/sl` loaded "Next list on the simulator and emulator: dialog width, a Column
   floor, code in RTL".
2. "Finish the Android DataTable fix. Then see what other low-hanging fruit we
   can finish in the Next list."
3. `/sw`: this doc, commit, push.

## 2. Commits

| Commit | What | Next item |
|---|---|---|
| 9d2a351 | CI `docs` job: pinned mkdocs + `mkdocs build --strict` | 24 (half) |
| b2b5309 | Android: a FlexGrow child in a scrolled Column keeps its content height | 20 |
| d4bb256 | Lesson 4.7: a second back leaves the lesson | 21 |
| — | Removed the six merged, clean agent worktrees and their branches | 26 |

## 3. The DataTable fix (b2b5309)

- The change is last session's `Renderer.kt.new`, unchanged:
  `LocalGrMobUnboundedHeight`. True inside GrMobScroll's content. Cleared by a
  points Height, a grow child that gets `heightIn(min = viewport)`, a stretched
  Row's children, and a Modal whose growing child has the window as its minimum.
  While it is true, `ColumnChildren` skips `weight`. Its KDoc has the table of
  who sets and clears it.
- `mobile/verify/banddistribution_test.go` matched the exact signature
  `private fun RowScope.RowChildren(node: GrMobNode)`. The fix added an
  `intrinsicHeight` parameter, so the anchor is now `RowScope.RowChildren(`, the
  prefix `shrink_test.go` already uses.
- **Verification.**
  - Tree walk: a throwaway Go test (deleted) walked every lesson's initial wire
    tree with the same rules. **The only affected node is 4.6's
    `/SafeArea/Scroll/Column/Column/Column/Column > List grow=1`.** It covers
    initial state only; demo toggles and open overlays are not walked.
  - Emulator: 4.6 draws four rows between the header and "Page 1 of 3". 6.4's
    modal stays centred. 6.7's sheet sits on the bottom edge (filler intact).
  - Checks: `go test ./...`, `android/verify/sources.sh` and
    `sh wasm/verify/run.sh` pass.
- No pixel sweep. See section 5 for why.

## 4. The other items

- **21, 4.7 back trap (d4bb256).** The demo AppBar's OnBack shows the note on
  the first press and calls `core.Pop(ctx)` on the second. The caption and demo
  hint say so. `TestScreenFurnitureSecondBackLeavesTheLesson` presses it twice.
  On the emulator, two `KEYCODE_BACK`s end on the contents screen ("1 of 57
  lessons opened").
- **24, mkdocs in CI (9d2a351).** A fourth independent job in `ci.yml`: Python
  3.12, then `mkdocs==1.6.1 mkdocs-material==9.7.7 pymdown-extensions==11.0.2`,
  then `mkdocs build --strict`. The header table and "three runners" sentences
  now say four. The strict build passes locally (three INFO lines about
  `../examples/...` links, none of them warnings). The shot-claim RTL half of
  item 24 is still open.
- **26, worktrees.** `git worktree remove` + `git branch -d` on each (both
  refuse dirty or unmerged). `git worktree list` now shows only master.
- **22, MinWidth/MinHeight on the natives: skipped.** Not low-hanging. On
  Android it has to fit into `widthModifier`'s layout lambda (percentages, a
  stretched parent forcing min = max, the MaxWidth cap). iOS needs the same, and
  its column floor (6aac336) already reads minimum heights.

## 5. What went wrong: two sweeps on one emulator

- Last session's sweep, a zsh that waited for an iOS UI test and then swept
  Android, **was still running**. It started a new session's worth of work at
  16:19 and was on its "after" pass at 16:46. It had already copied the fix into
  `Renderer.kt` and installed that build.
- My sweep ran on the same emulator at the same time. Both did force-stops,
  1500ms and 900ms drags, and `animator_duration_scale 0`. The result: splash
  frames, a lesson showing another lesson's content, and first frames that
  started mid-page. I misread this twice, first as deep-link Replace keeping a
  scroll offset, then as cold starts restoring one. I found it only when
  `git status` showed `Renderer.kt` modified while my own chain had been killed
  before its copy step.
- Stopped that one job (`pkill -P 88533; kill 88533`) and set
  `animator_duration_scale` back to 1. Its screenshots and mine were both
  contaminated, so the verification became the tree walk plus the targeted
  checks in section 3.
- My chain's first run also failed on `SDK location not found`: `./gradlew`
  needs `ANDROID_HOME` exported, and only `android/build.sh` defaults it.

## 6. Found: Next keeps the previous lesson's scroll offset on Android

- With nothing else touching the emulator: cold-start 1.3 (it opens at its
  top), drag to the bottom, tap "Next ›". 1.4 opens mid-page, with its title
  off screen and the demo in view.
- Likely cause: `core.Navigator` emits no wrapper node, and the route's root
  node carries no key. `RenderChildren` keys by `child.key.ifEmpty { i }`, so the
  replaced lesson lands at the same composition slot, and GrMobScroll's
  `rememberScrollState()` survives. A fix would key the frame by route id
  (`routeScopeKey`), but that changes reconciliation on every target (a Replace
  becomes a subtree swap), so it needs its own pass. iOS and the web are
  unchecked.

## 7. Process notes

- **Before any device job, `pgrep -fl` for sweeps, probes and Gradle left by
  earlier sessions.** A background zsh outlives the session that started it.
- A deterministic regression surface beats a pixel sweep here. When a change's
  trigger can be stated as a tree predicate, walk the wire trees in a Go test
  first and look only where it matches.
- `go test` in package-list mode hides `fmt.Print` output from passing tests.
  Use `-v` and `t.Log`.
- `uiautomator dump` sees only what is on screen. The `tapfind` loop (dump, look
  for text or content-desc within 150..2250 px, else drag 800px at 900ms) found
  "Open the modal" and "Open note actions" (the latter a content-desc).
- Emulator memory held with the iOS simulator shut down (`xcrun simctl
  shutdown`) and `./gradlew --stop` after each install.

## Next

**Legend:** *age* is how many session docs ago the item was first raised,
counted from this doc (0 = raised here; `≥ k` = older than the last rebuild's
window). *value*: **high** = worked around today or a second consumer has
arrived; **medium** = blocks one named thing or is a visible defect; **low** =
nobody has hit the gap yet. Sorted by age, oldest first; new items last.

1. **(age ≥9 · value high) Tag a release.** v0.3.0 is 90 commits behind with
   this doc. Pushing a tag needs the user's yes. Also still open: `go run
   github.com/rohanthewiz/grmob/cmd/grmob@latest new` on a clean machine without
   `-replace`.
2. **(age ≥9 · value high) Lessons on hardware, and checks still open.**
   - Real devices for everything (deferred by the user).
   - **Screen readers:**
     - radio and StepIndicator done-step semantics
     - combobox active option
     - "pop-up" on Menu/DatePicker triggers
     - aria-current/selected on current items
     - Calendar's grid on the web
     - CodeEditor toolbar role on Compose
     - SearchableSelect natives reading field + list + status
     - AccessibilityHidden confining TalkBack/VoiceOver behind a Drawer
   - **iOS:**
     - sheet `Dialog` and ActionSheet filler
     - Drawer slide by eye and under Reduce Motion
     - RTL for Drawer and CodeEditor (needs a way to force layout direction)
     - editing in the sideways code editor (caret reveal, selection handles)
     - `banded` and a drag on 4.6's list
   - **Pinning and Drawer:**
     - `Screen.Footer` pinning
     - 100% layers in a pinned-height ZStack
     - the panel's `Height 100%` in an HStack
     - `core.Focus` on a Button
     - a hardware keyboard reaching a shut panel
   - **Android:**
     - predictive back from contents
     - MaxWidth where it binds (tablet)
     - a capped stretched child in RTL
3. **(age ≥9 · value low) ", done" and native ", today".** Natives still
   append English ", today". StepIndicator keeps ", done" (documented: no ARIA
   state fits). PageUp/PageDown do not change month, and composite arrow keys
   are not flipped for RTL.
4. **(age ≥9 · non-goal) Tracked-Go-file count sentences** in `wasm/verify` are
   hand-edited when Go files are added. Still 505.
5. **(age ≥9 · non-goal) Windowing.** Declined in `non_goals.md`.
6. **(age ≥9 · non-goal) Rename `docs/components.md` to `comps.md`.**
7. **(age ≥9 · non-goal) Rewrite `components` in the older plans.**
8. **(age ≥9 · non-goal) Trim the copied Android shell's permissions.**
9. **(age ≥9 · non-goal) Replace the iOS usage strings further.**
10. **(age ≥9 · non-goal) C4 `Carousel`** until the host reports a scroll
    offset.
11. **(age 7 · non-goal) A parent that gains `onBack` after its descendants
    outranks them** on Android; the web ranks by document order.
12. **(age 7 · non-goal) An AppBar outside the Navigator** with a custom
    `OnBack` is outranked by the Navigator's pop on Android.
13. **(age 6 · non-goal) Forward does not re-open a screen left by browser
    back.**
14. **(age 5 · non-goal) Sticky headers, placement animation and
    `OnEndReached` in a List with no viewport** on Compose. Give the List a
    Height.
15. **(age 3 · value low) The cost of a paused TimelineView per node** in
    SwiftUI's box chain (`GrMobMotion`). Not profiled.
16. **(age 3 · non-goal) MaxWidth with a growing sibling on the natives.**
17. **(age 3 · non-goal) The typed-hash fold's `history.length` fallback.**
18. **(age 3 · non-goal) A page's own `pushState` while a claim is on screen.**
19. **(age 3 · non-goal) A Drawer's shut panel is composed on the natives.**
20. **(age 1 · value medium) MinWidth/MinHeight on the natives.** Documented as
    web-only, but DatePicker (MinWidth 300), the rich-text link prompt (280)
    and RichTextEditor (MinHeight) rely on them. Section 4 explains why this is
    not a quick one.
21. **(age 1 · value low) A Row in a horizontal scroll with FlexGrow children**
    has the same zero-weight collapse on Compose. `LocalGrMobUnboundedHeight`
    covers Columns only. Rows inside a bounded LazyColumn with growing children
    too.
22. **(age 1 · value low) Shot-claim field-text check** makes no RTL or
    letter-spacing adjustment (RTL fields use the box check). The mkdocs half
    of this item is done (9d2a351).
23. **(age 1 · non-goal) A List with no Height in a scrolled page is not lazy**
    on iOS or Android. Give the List a Height to virtualise.
24. **(age 0 · value medium) Next keeps the previous lesson's scroll offset on
    Android** (section 6). Likely fix: key the Navigator frame by route id.
    Check iOS and the web first, and the patch cost of a Replace becoming a
    subtree swap.
25. **(age 0 · value low) The DataTable tree walk covered initial state only.**
    Open overlays and toggled demos (4.6 Compact, 6.x sheets with growing
    content) were not walked. If a layout bug turns up there, turn the walk
    into a kept test over a few scripted states.
