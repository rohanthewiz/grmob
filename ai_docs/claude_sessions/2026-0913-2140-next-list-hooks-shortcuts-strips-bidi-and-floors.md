# Next list: hooks, page-global shortcuts, grow strips, bidi claims and native floors

**Session:** https://claude.ai/code/session_013PPKnYoYbsVv3vW7oAXA6p
**Date:** 2026-09-13 21:40
**Branch:** master (616edb8 → HEAD)

## 1. The asks

1. `/sl`: load the last session doc (paging keys and min sizes).
2. "Do 19 - 26 in batches, committing between each, then /sess-wrap".

## 2. Commits

| Commit | Batch | Next items (previous numbering) |
|---|---|---|
| 97b18f4 | A: pre-push every commit | 20 (19 refused, see 7) |
| ea8b2b5 | B: shortcuts and the live calendar | 22, 23 |
| 764337d | D: Compose grow strips, bidi claims | 24, 25 |
| c6a403e | C: native floors, today on Compose | 21, 26 |
| (this doc's commit) | the session doc | |

Batch D was committed before C because C's iOS work took longer.

## 3. Batch A: the pre-push hook

- **20.** `.githooks/pre-push` checks every commit of a push when a clone
  opts in: `git config grmob.prepush.everyCommit true`.
  - The range is `remote..local`. It falls back to `local --not --remotes`
    for a new branch, or when the remote's old tip isn't in this clone.
  - Commits are checked oldest first. Each commit's whole Go tree is
    exported and gofmt'd.
  - A ref with more than 100 new commits is checked at the tip only, with a
    note.
  - The default stays tip-only, for the reason the hook already gave.
  - The per-object check is now one function, `check_tree`.
- New `wasm/verify/prepush_test.go`: a temp repo with an unformatted commit
  under a clean tip.
  - Default passes, the setting on fails naming the commit, off passes again.
  - Global and system git config are shut out.
  - Forcing the default path to every-commit made the test fail.
- The tracked-Go-file count sentences went 505 → 506 → 507 (two new test
  files).

## 4. Batch B: shortcuts and check 15

- **23, page-global shortcuts.** A declared chord holding Control, Alt or
  Meta, or a bare F-key (web and Compose only), presses its control from
  anywhere. Bare keys stay widget-scoped. What each target does:

  | target | how |
  |---|---|
  | web | `handlePageShortcut`, one window keydown listener |
  | SwiftUI | `.keyboardShortcut` on GrMobButton, via `grMobKeyChord` |
  | Compose | `MainActivity.dispatchKeyEvent` → `GrMobRuntime.handleKeyEvent` |

  - **Web details.**
    - Modifiers must match exactly. Letters and digits also match by
      `e.code`, because macOS Option rewrites `e.key`.
    - Consumed, repeated and IME events are skipped.
    - Inert, aria-hidden and display:none subtrees are skipped.
    - A disabled match takes the key and does nothing.
  - **Compose details.**
    - The walk skips display none and AccessibilityHidden.
    - Pure chord logic lives in `GrMobKeyChord.kt`, which imports nothing.
    - Disabled state is carried down the walk. `isDisabled()` is
      @Composable, and calling it from the runtime failed to compile.
  - **Docs.** The `core.Style.AccessibilityKeyShortcuts` table, the style
    prop doc and `docs/platforms/wasm.md` were updated.
  - **Tests.**
    - Five keynav_test.mjs cases. Removing the listener fails four;
      removing the aria-hidden skip fails one.
    - `mobile/verify/keyshortcut_test.go` pins the native wiring.
- **22, check 15 in browser.mjs.**
  - It builds `./wasm` for js/wasm into a temp dir, with `wasm_exec.js` from
    GOROOT, and serves it under `/live/`.
  - It boots like index.html (no Leaflet) and sends `route` 4.9.
  - It passed:
    - PageUp from 11 March lands on February, focus 11;
    - PageDown returns to March, focus 11;
    - PageDown at Max stays on March, focus 11;
    - PageUp from the 31st lands on 28 February.
  - Tallies moved to five keyboard facts and Fifteen claims.
    `checknumbering_test.go` learned "fifteen". `run.sh`, the README and
    wasm.md were updated.
  - The OK line lost an apostrophe: `pinfixture`'s pinCodeOnly lexer read
    "4.9's" in a one-line template literal as an unterminated string.
- **Checks.**
  - `go test` passed for core, comps, mobile/verify, wasm/verify and
    internal.
  - `wasm/verify/run.sh` passed, including the browser pass.
  - `ios/verify/run.sh` and `android/verify/sources.sh` passed; the latter
    compiled the runtime and app packages.

## 5. Batch D: items 24 and 25

- **24, `GrMobGrowStrip`.** A horizontal Scroll with a FlexGrow child now
  uses a Layout of its own. How it measures:
  - A layout modifier outside the scroll records the viewport width
    (`StripViewport`) during the measure pass.
  - Non-growers are measured unbounded. Growers take their share of the free
    space as a minimum, so each is `max(content, share)`.
  - Every child is measured once, with no intrinsics.

  Rejected alternatives:
  - BoxWithConstraints: a SubcomposeLayout refuses intrinsics.
  - maxIntrinsicWidth over the content: a vertical Scroll inside would
    crash.

  Where it differs from CSS: with several growers of unequal content it
  takes the larger of content and share, not content plus share. This is the
  same as a weighted Row. It compiles and is pinned; see Next for what is
  unverified.
- **25, CLIPPED.**
  - Mixed-direction RTL field text is placed by a hidden laid-out copy of
    the value (`bidiRects`, one rect per bidi run), instead of the box rule.
  - Field text is allowed 1px (`FIELD_SLACK`), because a field's scroll
    offset is a whole pixel.
  - The first version reported "Omega" cut by 0.7px. A headless screenshot
    showed it whole.
  - The mixed-text test now expects the last word cut and the digits whole.
    A new case scrolls a Latin run into view.

## 6. Batch C: items 21 and 26

- **21, iOS percentage floors.**
  - `GrMobMinSize.floor` (in GrMobFlex.swift, checked by
    `checkMinSize` in ios/verify) and `GrMobMinimumLayout`, applied through a
    ViewModifier.
  - A percentage in either axis takes the layout; points keep
    `frame(minWidth:)`. The styling doc row now says `px, %` for iOS.
- **21, by eye.**
  - **Android emulator:**
    - the link prompt is about 304dp wide, so the 280 floor binds;
    - the DatePicker sheet is 360dp wide, so MaxWidth binds;
    - 7.3's role Text keeps its 110dp MinWidth: the hex starts exactly
      110dp plus the gap in;
    - 4.14's editor content is taller than its MinHeight.
  - **iOS simulator:** the link prompt is a sheet, so the floor holds but
    doesn't bind.
- **21, iOS UI tests.** New
  `ios/GrMobUITests/TutorialNativeFloorsUITests.swift`, both **passed**.
  - The link prompt test taps the editor first: Add link is disabled
    without a caret. The field is found as `otherElements["Link address"]`;
    the UITextField carries only the placeholder.
  - 4.9's today cell is named `"Wednesday, March 11, 2026, " + today`.
- **26, the today separator.**
  - Compose moved the word into `stateDescription`, so TalkBack joins it in
    the user's order. It falls back to `name + ", " + word` only when a
    stated AccessibilityValue holds that slot.
  - SwiftUI tried `accessibilityValue`, twice (outer, then beside the
    label). XCUITest read the value empty both times. The calendar cell
    stays an accessibility container, with children "11" and an empty
    Button, so the value doesn't reach it while the label does.
  - SwiftUI was therefore reverted to the label suffix, with the
    measurement recorded in `grMobCurrentLabel`'s doc and in
    `core.CurrentKind`'s table.

## 7. What went wrong

- **19 was refused again**, this time as "Self-Modification", on writing
  `.claude/settings.json` together with `.claude/hooks/enable-githooks.sh`.
  It was not worked around. The script was written (only in the refused
  command) to set `core.hooksPath` locally, and only when nothing sets it.
- **Parallel `cd` calls leaked the working directory twice.** One node call
  and one gradle call changed it, so later calls ran in the wrong place: a
  run.sh started from `wasm/verify`, and `go test` from `android/`. Use
  absolute paths and `go -C`.
- **The first iOS today assertion was a wrong design, not a flaky test.** It
  took three simulator runs to establish that the value channel doesn't
  reach the cell.
- **Gradle needs ANDROID_HOME in this shell** (`~/Library/Android/sdk`), or
  `:app:compileDebugKotlin` fails with "SDK location not found".
- **`uiautomator dump` shows no `state-description`**, so Compose's today
  state couldn't be read back on the emulator.

## Next

**Legend:** *age* is how many session docs ago the item was first raised,
counted from this doc (0 = raised here; `≥ k` = older than the last rebuild's
window). *value*: **high** = worked around today or a second consumer has
arrived; **medium** = blocks one named thing or is a visible defect; **low** =
nobody has hit the gap yet. Sorted by age, oldest first; new items last.

Done this session (previous numbering): 20, 21, 22, 23, 24, 25, 26
(26 on Compose; SwiftUI kept by measurement, see 6).

1. **(age ≥13 · value high) Lessons on hardware, and checks still open.**
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
2. **(age ≥13 · non-goal) Tracked-Go-file count sentences** in `wasm/verify` are
   hand-edited when Go files are added. Now 507.
3. **(age ≥13 · non-goal) Windowing.** Declined in `non_goals.md`.
4. **(age ≥13 · non-goal) Rename `docs/components.md` to `comps.md`.**
5. **(age ≥13 · non-goal) Rewrite `components` in the older plans.**
6. **(age ≥13 · non-goal) Trim the copied Android shell's permissions.**
7. **(age ≥13 · non-goal) Replace the iOS usage strings further.**
8. **(age ≥13 · non-goal) C4 `Carousel`** until the host reports a scroll
   offset.
9. **(age 12 · non-goal) A parent that gains `onBack` after its descendants
   outranks them** on Android; the web ranks by document order.
10. **(age 12 · non-goal) An AppBar outside the Navigator** with a custom
    `OnBack` is outranked by the Navigator's pop on Android.
11. **(age 11 · non-goal) Forward does not re-open a screen left by browser
    back.**
12. **(age 10 · non-goal) Sticky headers, placement animation and
    `OnEndReached` in a List with no viewport** on Compose. Give the List a
    Height.
13. **(age 8 · value low) The cost of a paused TimelineView per node** in
    SwiftUI's box chain (`GrMobMotion`). Not profiled.
14. **(age 8 · non-goal) MaxWidth with a growing sibling on the natives.**
15. **(age 8 · non-goal) The typed-hash fold's `history.length` fallback.**
16. **(age 8 · non-goal) A page's own `pushState` while a claim is on screen.**
17. **(age 8 · non-goal) A Drawer's shut panel is composed on the natives.**
18. **(age 6 · non-goal) A List with no Height in a scrolled page is not lazy**
    on iOS or Android. Give the List a Height to virtualise.
19. **(age 2 · value low) The pre-push hook is off in every other checkout.**
    `core.hooksPath` is per clone. The SessionStart hook
    (`.claude/hooks/enable-githooks.sh` plus a `SessionStart` entry in
    `.claude/settings.json`) was refused twice by the auto-mode classifier
    ("Unauthorized Persistence", then "Self-Modification"), even with the
    user asking for it. It needs a permission rule for those two paths, or
    the user making the edit. `hookconfig_test.go` should then hold its
    entry.
20. **(age 0 · value medium) The iOS calendar cell is an accessibility
    container.** XCUITest shows the labelled cell with children "11" and an
    empty Button, so `.combine` doesn't collapse it. VoiceOver may stop on
    the numeral instead of the named day, and an accessibilityValue never
    reaches the cell. Check with VoiceOver, and find what keeps the children
    separate (GrMobGestures' `.isButton` traits on the content are a
    suspect).
21. **(age 0 · value low) Compose's today `stateDescription` not heard.**
    `uiautomator dump` has no state description; TalkBack with speech output
    shown would settle the order ("today, Wednesday…").
22. **(age 0 · value low) `GrMobGrowStrip` has run nowhere.** It compiles and
    is pinned. No widget or lesson puts a FlexGrow child in a horizontal
    Scroll, so neither the fill nor the overflow case has been seen. A
    pinfixture-style case, or a tutorial strip, would.
23. **(age 0 · value low) Page-global shortcuts untried on hardware.** Web is
    tested against dom.mjs only (no browser.mjs fact). SwiftUI's
    `.keyboardShortcut` and Compose's `dispatchKeyEvent` have not had a
    keyboard pressed at them. No widget declares a modifier chord yet.
24. **(age 0 · value low) A percentage MinWidth/MinHeight on iOS is unseen.**
    `GrMobMinimumLayout` is type-checked and its arithmetic is checked. No
    widget declares a percentage floor, and alignment inside the floor
    (`grMobAlignmentFraction`) has no case.
25. **(age 0 · non-goal) Growers with unequal content in a strip** take
    `max(content, share)` rather than CSS's `content + share`; the same as a
    weighted Row on Compose.
26. **(age 0 · non-goal) F-keys as SwiftUI shortcuts.** KeyEquivalent has no
    function keys; a bare F-key chord is skipped on iOS.
27. **(age 0 · non-goal) Every-commit pre-push reports a commit for any
    unformatted file in its tree,** including one an earlier pushed commit
    introduced. It is one command with one meaning; diff attribution was
    declined.
28. **(age 0 · non-goal) Checking every commit of a push by default.** CI
    checks the tip; the mode stays opt-in.
