# Next list: a Row that hugs, chords on a box, and a strip that divides

**Session:** 54877e82-f53f-4a29-b22b-1f8275fc4355
**Date:** 2026-09-14 23:19
**Branch:** master (fb75a4b → HEAD)

## 1. The asks

1. `/sl`: load the last session doc (hook blame, cell elements, F-keys, a Row
   that fills).
2. "Do 20 - 27 in batches, committing between each, then /sess-wrap".

## 2. Commits

| Commit | Batch | Next items (previous numbering) |
|---|---|---|
| 97a58b5 | A: Compose Row hug | 27, 23, 25 |
| bdef4fc | B: iOS chords on a tappable box | 22, 24 |
| abccdfc | C: iOS strip with a grower | 26 |
| (this doc's commit) | the session doc | |

Items 20 and 21 were worked on and are not done (see 5 and 6).

## 3. Batch A: Compose (27, 23, 25)

- **The defect (27).** A Compose Row offers each unweighted child the space
  left as a maximum. A Column with no AlignItems stretches its Text
  (`fillMaxWidth`), so it fills that maximum and later children get 0.
- **Census.** A throwaway Go test walked every lesson's default tree for Row
  children the fix would change. Hits: 1.1, 1.3, 1.4, 4.15, 8.2. The 4.17 hits
  were Modal false positives.
- **Emulator, before:** 1.1 showed one stat of three, 1.3 and 1.4 box A alone,
  4.15 one star of five, 8.2 lost counter B. **After:** every child drawn.
- **Fix.** `RowChildren` gives an unweighted container child `hugRowOffer`:
  - measured within `min(offer, maxIntrinsicWidth)`;
  - a percentage MinWidth is resolved against the Row's offer, because
    `widthModifier` further in would see only the content width;
  - skipped for leaves, a Width, a percentage MaxWidth, and subtrees that
    cannot answer intrinsics (List, vertical Scroll).
  - `parseWidthCap`/`WidthCap` became internal.
- **23.** 1.4 with `MinWidth("40%")` on: A's box is 317px of the row's 792px.
  B starts one gap and padding later (529 = 461 + 21 + 26 + 21).
- **25.** A throwaway probe app (`examples/zzprobe`, deleted) had a strip of
  two growers with 63px and 361px of content. They measured 339px and 636px,
  a 297px difference against 298px of content, so content plus share holds.
- **What the probe also showed.** Lesson 1.1's stat Columns carry the theme's
  16pt side padding. The stats need 725px against 624px, so the third column
  breaks mid-word ("Fo llo wi ng"). CSS would overflow at the word
  (`min-width: auto`). 8.2's "Repair" Button is squeezed the same way, before
  and after the fix.
- **Checks.** `mobile/verify/row_hug_test.go` (two mutations each fail it),
  `android/verify/sources.sh` OK, `go test` for tutorial, mobile/verify,
  wasm/verify, pinfixture. `docs/platforms/native.md` has a new section.
  Tracked Go files 507 → 508.

## 4. Batches B and C: iOS (22, 24, 26)

- **22.** `GrMobGestures` hands a tappable box's chords to invisible Buttons,
  one per chord, each running the tap (`grMobBoxKeyShortcuts`, sharing
  `GrMobShortcutButtons` with a Button's further chords).
  - Lesson 2.2's gesture card declares "Control+Alt+J"; the tutorial test
    fails with it removed.
  - core.AccessibilityKeyShortcuts' target table was stale (it said SwiftUI
    takes the first chord only and F-keys are web and Compose); updated, and
    `docs/api` regenerated.
- **24.** `non_goals.md`: "An F-key is not a SwiftUI keyboard shortcut".
- **26.** On the simulator the footer count sat 16pt after "Start over".
  - A strip with a grower is now `GrMobFlexStack` inside
    `GrMobStripContentLayout`, proposed `max(ideal, viewport)`.
  - The viewport was first read with a GeometryReader preference. A
    marker-file probe (`tmp/grmob-strip.log` in the app container) showed it
    arriving as 0 every time, so the row was placed at 253–255pt.
  - `onGeometryChange` reads 308pt. `TutorialFooterStripUITests` **passed**
    twice: the count ends at 355pt (strip x 47 + 308) and is centred on the
    chip.
  - Pinned by `strip_test.go` (a mutation fails it). Tracked Go files 509.
- **Checks.** `ios/verify/run.sh` passed every step after each change.

## 5. Item 21 and the key tests, not finished

A new simulator, "GrMob Keys" (iPhone 17 Pro, iOS 26.5), was created. Runs:

| Run | Change | J on the card | K on the Button | footer |
|---|---|---|---|---|
| 1 | box route | **passed** | failed | failed |
| 2 | none (K alone) | | failed | |
| 3 | every chord on invisible Buttons, strip v1 | failed | failed | failed |
| 4 | revert of run 3, probe | failed | failed | failed |

- So key delivery to the simulator worked once and then stopped, as it did
  last session. It is not route-specific, and the run 3 change was reverted.
- F6 through GameController was never reached.

## 6. Item 20, TalkBack, not finished

- Every calendar cell is **two** accessibility nodes in `uiautomator dump`: a
  clickable, focusable, checkable View with no description, and inside it a
  non-clickable View carrying the label, offset about 3px. Seen on March 4,
  11 and 12.
- With TalkBack on, none of these moved the green focus frame (checked per
  screenshot with a `sips` → BMP pixel scan):
  - `input tap` on the cell;
  - `input swipe` right and left;
  - 30 × `input keycombination KEYCODE_ALT_LEFT KEYCODE_DPAD_RIGHT`.
- No speech overlay appeared. TalkBack was turned off and the emulator stopped.

## 7. What went wrong

- **A conclusion from two runs.** Run 2's K failure was read as "a Button's own
  keyboardShortcut does not fire", and every chord was moved to invisible
  Buttons. Run 3 failed J as well, so the change was reverted and the claim
  removed from comments.
- **Editing Swift during `ios/verify`** failed it ("modified during the build").
- **1.1's squeeze looked like a hug bug** until the probe showed the theme
  padding.
- **`cd` in a background command** moved the shell's working directory once.

## Next

**Legend:** *age* is how many session docs ago the item was first raised,
counted from this doc (0 = raised here; `≥ k` = older than the last rebuild's
window). *value*: **high** = worked around today or a second consumer has
arrived; **medium** = blocks one named thing or is a visible defect; **low** =
nobody has hit the gap yet. Sorted by age, oldest first; new items last.

Done this session (previous numbering): 22 (verified once), 23, 25, 26, 27;
24 moved to `non_goals.md`.

1. **(age ≥15 · value high) Lessons on hardware, and checks still open.**
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
     - VoiceOver on the calendar cell now that it is one element
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
2. **(age ≥15 · non-goal) Tracked-Go-file count sentences** in `wasm/verify` are
   hand-edited when Go files are added. Now 509.
3. **(age ≥15 · non-goal) Windowing.** Declined in `non_goals.md`.
4. **(age ≥15 · non-goal) Rename `docs/components.md` to `comps.md`.**
5. **(age ≥15 · non-goal) Rewrite `components` in the older plans.**
6. **(age ≥15 · non-goal) Trim the copied Android shell's permissions.**
7. **(age ≥15 · non-goal) Replace the iOS usage strings further.**
8. **(age ≥15 · non-goal) C4 `Carousel`** until the host reports a scroll
   offset.
9. **(age 14 · non-goal) A parent that gains `onBack` after its descendants
   outranks them** on Android; the web ranks by document order.
10. **(age 14 · non-goal) An AppBar outside the Navigator** with a custom
    `OnBack` is outranked by the Navigator's pop on Android.
11. **(age 13 · non-goal) Forward does not re-open a screen left by browser
    back.**
12. **(age 12 · non-goal) Sticky headers, placement animation and
    `OnEndReached` in a List with no viewport** on Compose. Give the List a
    Height.
13. **(age 10 · value low) The cost of a paused TimelineView per node** in
    SwiftUI's box chain (`GrMobMotion`). Not profiled.
14. **(age 10 · non-goal) MaxWidth with a growing sibling on the natives.**
15. **(age 10 · non-goal) The typed-hash fold's `history.length` fallback.**
16. **(age 10 · non-goal) A page's own `pushState` while a claim is on screen.**
17. **(age 10 · non-goal) A Drawer's shut panel is composed on the natives.**
18. **(age 8 · non-goal) A List with no Height in a scrolled page is not lazy**
    on iOS or Android. Give the List a Height to virtualise.
19. **(age 4 · value low) The pre-push hook is off in every other checkout.**
    `core.hooksPath` is per clone. The SessionStart hook needs a permission
    rule for `.claude/hooks/enable-githooks.sh` and `.claude/settings.json`,
    or the user making the edit; `hookconfig_test.go` should then hold it.
20. **(age 2 · value low) Compose's today `stateDescription` not heard.**
    - Injected tap, swipe and Alt+Right do not reach TalkBack on the emulator.
    - Next try: an instrumented test that performs ACTION_ACCESSIBILITY_FOCUS
      on the cell and reads its AccessibilityNodeInfo, or a person with the
      emulator's own mouse.
21. **(age 2 · value medium) F-keys on iOS unverified, and key delivery to a
    simulator is unreliable.** One pass in four runs on a new simulator.
    - Next: an iPad with a hardware keyboard, or find what stops XCUITest's
      `typeKey` after the first run (a Simulator app restart; the "Connect
      Hardware Keyboard" setting was not readable).
22. **(age 2 · value low) Page-global chords on iOS are verified once** on a
    tappable box (J) and three times last session on a Button (K). Neither
    passed on the new simulator after its first run.
23. **(age 1 · non-goal) GameController cannot take a key.** An F-key press
    also reaches UIKit's responders; it has no default action on iOS.
24. **(age 1 · non-goal) Commits already on a remote carrying an unformatted
    file** are noted, not blocked, except at the tip.
25. **(age 0 · value medium) Compose has no `min-width: auto`.** Lesson 1.1's
    third stat and 8.2's "Repair (set B = A)" button break mid-word where a
    browser overflows at the word. `minIntrinsicWidth` could floor a Row
    child the way `hugRowOffer` caps it.
26. **(age 0 · value low) Compose calendar cells are two accessibility
    nodes:** the label sits inside a separate clickable node. It may be why
    TalkBack's touch focused the whole grid last session.
27. **(age 0 · value low) `hugRowOffer` gaps:** a child with a percentage
    MaxWidth and a horizontal Scroll inside a Row keep filling the offer.
28. **(age 0 · value low) The Compose Row hug was swept on the tutorial only.**
    todoapp, mobileapp, fintechapp, social and chat were not looked at.
29. **(age 0 · value low) A SwiftUI chord behind an AccessibilityHidden
    subtree still fires** on iOS; the web, Compose and the iOS F-key walk skip
    hidden subtrees.
30. **(age 0 · value low) iOS strip longer than its viewport with a grower**
    is unseen; by construction it scrolls at its ideal width.
