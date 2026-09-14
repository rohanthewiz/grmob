# Next list: hook blame, calendar cells as one element, F-keys, and a Row that fills

**Session:** https://claude.ai/code/session_013PPKnYoYbsVv3vW7oAXA6p
**Date:** 2026-09-13 22:50
**Branch:** master (73a8f70 → HEAD)

## 1. The asks

1. `/sl`: load the last session doc (hooks, shortcuts, strips, bidi, floors).
2. "Do 20 - 28 in batches, committing between each, then /sess-wrap".

## 2. Commits

| Commit | Batch | Next items (previous numbering) |
|---|---|---|
| f8ab906 | A: pre-push blame and default | 27, 28 |
| e8ab4ae | B: iOS cells, floors, chords | 20, 24, 26, and 23 on iOS |
| 91563ff | D: browser check 16 | 23 on the web |
| f1cab8a | C: Compose strips, lesson 4.8 | 22, 25 |
| (this doc's commit) | the session doc | |

Item 21 was worked on and is not done (see 6). Item 23's Android half was
measured on the emulator and needed no code.

## 3. Batch A: the pre-push hook (27, 28)

- **28.** Every commit of a push is checked by default.
  `git config grmob.prepush.everyCommit false` checks the tip only.
- **27.** Each unformatted file is sorted by
  `git diff -z --name-only <first parent> <commit>`:

  | the file in a commit | report | blocks |
  |---|---|---|
  | changed by the commit (or a root commit) | listed under it | yes |
  | unchanged since a parent of this push | "+ N unchanged since X, listed above" | yes |
  | unchanged since before the push | listed once as "from before this push", then "noted above" | only at the ref's tip |

  - Origins carry forward in a status file per commit in the temp dir.
  - The tip still blocks on anything, because CI's gofmt runs on it.
  - `-z` was required by `gitscript_test.go`. The records go through a file,
    so a failed diff is seen.
- **Tests.** `wasm/verify/prepush_test.go` has three commits and five pushes:
  - default and `true` both fail, naming the first commit, and the second
    commit says "listed above";
  - `false` passes;
  - with commit 1 already on the remote, pushing 2..3 passes with a note,
    and pushing 2 alone fails at the tip.
- **Mutations.** Counting every file as the commit's own failed the test 6
  times. A default of `false` failed it 5 times.

## 4. Batch B: iOS (20, 24, 26; 23 on iOS)

- **20, the calendar cell.**
  - A probe XCUITest dumped each day cell as a labelled Button holding a
    Button "11" and an empty 0×0 Button.
  - Cause: GrMobGestures put `.isButton` and `.accessibilityAction` on the
    box's content, inside the label's `.combine`. They spread to each child,
    and `.combine` keeps interactive children as separate elements.
  - Fix: `GrMobGestureAccessibility`, applied after the combine. Its children
    now read StaticText "11" and Other.
  - A separate step in GrMobBoxModifier's chain crashed SILGen
    (substOpaqueTypes). So `grMobAccessibility` and the new modifier are one
    `GrMobAccessibilityModifier`.
  - UI test: the today cell has no button descendants. Passed.
- **24, percentage floors.**
  - Lesson 1.4 gained `MinWidth("40%")` on box A as a checkbox.
  - On the simulator the floor did not bind: a Row proposes a child nothing,
    then its own slot, so 40% of that never binds.
  - `GrMobFlexSolver.percentFloors` now resolves the fraction against the
    container's offer. It raises base and min, and FlexChildren gives the
    child a main-axis fill.
  - The arithmetic is checked in ios/verify's `checkMinSize`, including
    lesson 1.4's numbers and a squeeze.
  - UI test `testAPercentageMinWidthFloorsTheBox` **passed**. A demoBox label
    reports its whole box's frame to XCUITest, so the test reads the boxes
    directly. Centring was checked on the screenshot.
- **Every chord a shortcut.** `grMobKeyChords` returns all page-global
  chords. The first goes on the Button; each further one is an invisible,
  accessibility-hidden Button behind it.
- **26, F-keys.** Two routes failed, measured:
  - a KeyEquivalent from AppKit's function-key character (U+F709 for F6)
    never fired, as the first chord or the second;
  - `NSPrincipalClass` names a UIApplication subclass, but a SwiftUI App
    ignores it: the running class was `SwiftUIApplication`.

  F-keys now go through `GrMobFunctionKeys`
  (`GCKeyboard.keyChangedHandler`) to `GrMobRuntime.pressFunctionKey`. That
  is a tree walk like Compose's: first onClick node in tree order, hidden
  subtrees skipped, a disabled match takes the key.
- **Lesson 2.2.** It declares "Control+Alt+K F6" on "Log from the keyboard".
  - `TutorialKeyShortcutsUITests`: Control+Option+K passed on three runs
    before the simulator rebooted.
  - F6 through GameController is **unverified** (see 7).
- **Checks.** `ios/verify/run.sh` passed every step, WMO and the app layer
  against the iOS SDK included. mobile/verify pins the chord list, the
  F-key skip, the listener and the walk.

## 5. Batches C and D (22, 25; 23 on the web and Android)

- **23, web.** Browser check 16 runs in check 15's live build:
  - opens lesson 2.2 with nothing focused;
  - Control+Alt+K and F6 each log a press;
  - a bare K logs nothing.

  **Passed** in headless Chrome. Tallies read "six about the keyboard" and
  "Sixteen claims". `checknumbering_test.go` learned "sixteen"; run.sh and
  wasm.md were updated.
- **23, Android.** After `installDebug`, on lesson 2.2:
  - `adb shell input keycombination KEYCODE_CTRL_LEFT KEYCODE_ALT_LEFT
    KEYCODE_K` logged "1 · button";
  - `KEYCODE_F6` logged "2 · button";
  - a bare `KEYCODE_K` logged nothing.
- **25, content plus share.** GrMobGrowStrip gives each grower its
  `maxIntrinsicWidth` plus its share.
  - `answersIntrinsicWidth` refuses a subtree with a List or a vertical
    Scroll (SubcomposeLayouts); those keep `max(content, share)`.
  - Pinned with literals intact through `valuesOf`, since `codeOf` blanks
    strings.
- **22, a strip that runs.** Lesson 4.8's footer is a scrollable ChipStrip:
  Start over, a FlexGrow Box, and the count.
  - Children replace Chips in a ChipStrip, so the chip went into Children.
  - On the emulator the count sits at the strip's far edge, while the chip
    strip above still overflows.
  - `chapter4_test.go` pins the footer's one grower and its caption.
- **Checks.** `android/verify/sources.sh` OK; `wasm/verify/run.sh` OK;
  `go test` passed for examples/tutorial, mobile/verify and wasm/verify.

## 6. Item 21, TalkBack, not finished

- `uiautomator dump` shows only the content description.
- TalkBack 16 logs no utterances at logcat's default level.
- TalkBack developer setting "Display speech output" was turned on in the
  emulator's TalkBack settings, and is still on. The service name that
  starts it is `com.google.android.marvin.talkback/.TalkBackService`.
- Tapping the 11th with TalkBack on put focus on the **whole calendar grid**
  (a green frame around it), not on the cell.
- Right swipes then read "Thursday, April 2, 2026", "Friday, April 3" and so
  on: the trailing adjacent-month cells. The 11th was not reached within the
  capped number of swipes, so its words are still unheard.
- A first attempt was blocked by TalkBack's notification-permission dialog.
  `pm grant … POST_NOTIFICATIONS` avoids it.

## 7. What went wrong

- **XCUITest key presses stopped reaching the app after the simulator
  rebooted.**
  - Control+Option+K passed on three runs before the reboot.
  - Afterwards it failed on six of seven runs: with the Simulator window
    open, brought to front, and with the GameController listener removed.
  - So F6 through GameController was never observed, and the marker file
    shows no keys at all.
  - `osascript` has no assistive access, so "Connect Hardware Keyboard"
    could not be read.
- **A probe crashed the app.** Reading `UIApplication.shared` in `App.init`
  is a nil dereference (EXC_BAD_ACCESS in `GrMobApp.init`). NSLog probes
  never showed in `log stream`; a marker file in the app's tmp dir did.
- **`cd` in parallel calls leaked the working directory three more times.**
  One `rm` and heredoc did not run because of it. Use absolute paths.
- **A Python edit script aborted partway.** GrMobApplication.swift was
  deleted and GrMobApp.swift edited before a later anchor failed. Check every
  anchor first, or edit one file per script.
- **A ChipStrip's Children replace its Chips.** The first footer edit lost
  the Start over chip, and no test noticed until one was added.
- **Memory pressure.** The simulator shut itself down once, and the emulator
  exited once, with both running beside xcodebuild and Gradle.
- **Lesson 1.4's row fills on Android whatever the checkbox says.** A, a
  stretching Column, takes the whole Row, and B and C are gone. The same
  screenshot with the floor off shows it (see Next).

## Next

**Legend:** *age* is how many session docs ago the item was first raised,
counted from this doc (0 = raised here; `≥ k` = older than the last rebuild's
window). *value*: **high** = worked around today or a second consumer has
arrived; **medium** = blocks one named thing or is a visible defect; **low** =
nobody has hit the gap yet. Sorted by age, oldest first; new items last.

Done this session (previous numbering): 20, 22 (Compose), 23 (web and
Android), 24 (iOS), 25 (code), 26 (code), 27, 28.

1. **(age ≥14 · value high) Lessons on hardware, and checks still open.**
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
2. **(age ≥14 · non-goal) Tracked-Go-file count sentences** in `wasm/verify` are
   hand-edited when Go files are added. Still 507.
3. **(age ≥14 · non-goal) Windowing.** Declined in `non_goals.md`.
4. **(age ≥14 · non-goal) Rename `docs/components.md` to `comps.md`.**
5. **(age ≥14 · non-goal) Rewrite `components` in the older plans.**
6. **(age ≥14 · non-goal) Trim the copied Android shell's permissions.**
7. **(age ≥14 · non-goal) Replace the iOS usage strings further.**
8. **(age ≥14 · non-goal) C4 `Carousel`** until the host reports a scroll
   offset.
9. **(age 13 · non-goal) A parent that gains `onBack` after its descendants
   outranks them** on Android; the web ranks by document order.
10. **(age 13 · non-goal) An AppBar outside the Navigator** with a custom
    `OnBack` is outranked by the Navigator's pop on Android.
11. **(age 12 · non-goal) Forward does not re-open a screen left by browser
    back.**
12. **(age 11 · non-goal) Sticky headers, placement animation and
    `OnEndReached` in a List with no viewport** on Compose. Give the List a
    Height.
13. **(age 9 · value low) The cost of a paused TimelineView per node** in
    SwiftUI's box chain (`GrMobMotion`). Not profiled.
14. **(age 9 · non-goal) MaxWidth with a growing sibling on the natives.**
15. **(age 9 · non-goal) The typed-hash fold's `history.length` fallback.**
16. **(age 9 · non-goal) A page's own `pushState` while a claim is on screen.**
17. **(age 9 · non-goal) A Drawer's shut panel is composed on the natives.**
18. **(age 7 · non-goal) A List with no Height in a scrolled page is not lazy**
    on iOS or Android. Give the List a Height to virtualise.
19. **(age 3 · value low) The pre-push hook is off in every other checkout.**
    `core.hooksPath` is per clone. The SessionStart hook
    (`.claude/hooks/enable-githooks.sh` plus a `SessionStart` entry in
    `.claude/settings.json`) was refused twice by the auto-mode classifier.
    It needs a permission rule for those two paths, or the user making the
    edit; `hookconfig_test.go` should then hold its entry. Every-commit is
    now the default, so a clone that enables it gets the stricter check.
20. **(age 1 · value low) Compose's today `stateDescription` not heard.**
    - TalkBack explore-by-touch on 4.9 focused the whole grid, not a cell;
      that may be a defect of its own (grid semantics on Compose).
    - Swipes read the trailing April cells, and the 11th was not reached.
    - Next try: focus the 11th another way (a keyboard's arrow keys under
      TalkBack, or starting from the month header) and read the "Display
      speech output" overlay, which is now on in the emulator.
21. **(age 1 · value medium) F-keys on iOS unverified.**
    - `GrMobFunctionKeys` compiles, and GCKeyboard connects; the handler is
      installed.
    - No key of any kind reached the app from XCUITest after the simulator
      rebooted, so neither F6 nor the pre-reboot-verified Control+Option+K
      could be re-checked.
    - Next: a fresh simulator (erase or a new device), confirm
      Control+Option+K, then F6. If GameController does not see XCUITest's
      keys, try a hardware keyboard on an iPad.
22. **(age 1 · value low) Page-global shortcuts: iOS modifier chords reach
    Buttons only** (they ride SwiftUI's keyboardShortcut), where the web,
    Compose and iOS F-keys press any node with an onClick.
23. **(age 1 · value low) A percentage MinWidth on Android is unseen**
    (lesson 1.4 cannot show it; see 27). Alignment inside the floor
    (`grMobAlignmentFraction`) now has a screenshot on iOS only.
24. **(age 1 · non-goal) F-keys as SwiftUI shortcuts.** Measured not to fire;
    GameController carries them instead.
25. **(age 0 · value low) Unequal growers in a strip are unseen on a device.**
    Content plus share is compiled and pinned. Lesson 4.8's footer has one
    grower with no content, where both rules agree.
26. **(age 0 · value low) Lesson 4.8's footer strip on iOS is unseen.** It is
    not known whether SwiftUI's horizontal ScrollView gives the FlexGrow
    spacer the free space; the count may sit right after the chip.
27. **(age 0 · value medium) Lesson 1.4's demo row on Android: box A fills
    the Row, and B and C disappear.**
    - Pre-existing: the emulator shot with the MinWidth checkbox off shows
      it.
    - Likely cause: A is a demoBox Column that stretches its Text
      (fillMaxWidth). A Compose Row offers an unweighted child the remaining
      width as its maximum, so the stretched child fills it. CSS sizes that
      item by max-content.
    - A fix is measuring unweighted Row children against
      `min(offer, maxIntrinsicWidth)` where `answersIntrinsicWidth` allows.
      That touches every Row on Android and needs a sweep of the lessons.
28. **(age 0 · non-goal) GameController cannot take a key.** An F-key press
    also reaches UIKit's responders; it has no default action on iOS.
29. **(age 0 · non-goal) Commits already on a remote carrying an unformatted
    file** are noted, not blocked, except at the tip.
