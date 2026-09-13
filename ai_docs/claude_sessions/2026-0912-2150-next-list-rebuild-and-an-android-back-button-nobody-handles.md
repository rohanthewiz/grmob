# Next-list rebuild, and an Android back button nobody handles

**Session:** https://claude.ai/code/session_013PPKnYoYbsVv3vW7oAXA6p
**Date:** 2026-09-12 21:50
**Branch:** master

## 1. The asks

1. `/sl` loaded the C2 Drawer session doc.
2. `/next-list 3` rebuilt the Next list from the last three session docs
   instead of copying it forward.
3. "Create a new sess doc with the updated Next list items."

No code changed. This doc's only job is the rebuilt `## Next` below.

## 2. Window

`n = 3`, all dated 2026-09-12, oldest → newest:

| Pos | Doc |
|---|---|
| 1 | `2026-0912-2044-next-list-sweep-disabled-radios-a-code-toolbar-and-a-reference-without-aria-spec.md` |
| 2 | `2026-0912-2112-tier-c-a-menu-and-a-searchable-select.md` |
| 3 | `2026-0912-2137-c2-a-drawer.md` |

Age = 3 − first position. Items carried into doc 1 have ages capped by the
window and are written `≥2`. Items doc 1 called "new" have an exact age of 2.

## 3. What the rebuild found

### Lapsed items
No item dropped out entirely. Some detail was lost from the wording along the
way:
- The `TestWhatWindowingWouldSave` pointer dropped out of the windowing
  non-goal in doc 2. Restored (the test is still at
  `examples/tutorial/app_test.go:863`).
- "The next host CSS change will need another note" dropped out of the
  `wasm/index.html` item in doc 2. Restored.
- "And count untracked files" dropped out of the count-sentence non-goal in
  doc 3. Left out, since the counted sentences say "tracked".

### Premises checked against the code
- **Android system back (widened).** Doc 3 raised it as a Drawer-only gap
  ("a Modal gets this from `Dialog`"). The whole Android shell ignores system
  back:
  - `android/` has no `BackHandler`, `OnBackPressedCallback`, `onKeyDown` or
    `finish()` in any `.kt` file, `MainActivity.kt` included.
  - core, mobile and webhost have no host "back" event.
  - The tutorial's `OnBack` is the in-app AppBar back button, not the system
    one.

  So back closes neither a Drawer nor core's navigation stack. That makes two
  consumers for one back-press hook, so the value goes from medium to high.
- **Hand-rolls (corrected).** "Chapter 6.4's and chapter 1's `checkRow`" is
  one helper (`chapter1.go:96`) called from chapters 4 and 6. The "⋯" at
  `chapter6.go:734` is the ActionSheet lesson opening its own sheet, so
  migrating it to `Menu` would hide what that lesson teaches. The one at
  `chapter4.go:3193` is already a `Menu`.
- **Still true:**
  - No tag after v0.3.0 (22 commits since).
  - `docs/api/core.md` is 7,769 lines.
  - `grmob ios -run` exists (`native.go:352`).
  - `-refresh` re-copies only `android/` and `ios/`.
  - No `inert` in core or `wasm/grmob-runtime.js`.
  - `core.Style.MaxWidth` has no Kotlin or Swift reader.
  - No combobox role, `aria-haspopup` state or `aria-current`-style state in
    core. The only haspopup mentions are comments in `core/style.go:428` and
    `comps/menu.go:85`.
  - No `OnScroll` or scroll offset in core.
  - No repeat in `core/animation.go`, and `Spinner` is still stepped from Go.

### Deletion candidates
Items 9–11 (remedy drill, `hero.png`, third face) are low value and go back
at least to the `0440` and `1627` docs. They should be deleted rather than
carried again. They are still listed below so they aren't dropped silently.

## Next

**Legend:** *age* is how many session docs ago the item was first raised,
counted back from the newest doc in a 3-doc window (0 means that doc raised
it; `≥ k` means it's older than the window). *value* is the payoff of doing it,
not the effort: **high** means something is worked around today or a second
consumer has arrived; **medium** means it blocks one named thing or is a
visible defect; **low** means nobody has hit the gap yet.
The list is sorted by age, oldest first.

1. **(age ≥2 · value high) Tag a release.** v0.3.0 is the latest tag and there
   are 22 commits since. Tier A, Tier B, the radio roles, the sweep and Tier C
   (Menu, SearchableSelect, Drawer) would make a natural v0.4.0. Still open:
   run `go run github.com/rohanthewiz/grmob/cmd/grmob@latest new` on a clean
   machine without `-replace`.
2. **(age ≥2 · value high) Run lessons 6.6, 6.7, 4.15, 4.16, 4.17 and 4.18 on a
   simulator and a device.** None of this has been checked on real hardware:
   - settings-row one-tap, `Screen.Footer` pinning, the iOS sheet `Dialog`,
     the stepped spinner
   - ActionSheet filler placement on Compose and SwiftUI, Timeline row
     stretch, horizontal StepIndicator scroll, radio semantics
   - CodeEditor's toolbar role on Compose, and a Modal inside a ListRow's
     trailing Row (Menu)
   - SearchableSelect's Next action and keyboard dismiss on pick
   - Drawer: 100% layers filling a pinned-height ZStack on Compose and
     SwiftUI, the panel's `Height 100%` in an HStack, `AccessibilityHidden`
     confining TalkBack/VoiceOver, and `core.Focus` on a Button on both natives
3. **(age ≥2 · value med) Launch the scaffolded app on a simulator and a
   device.**
4. **(age ≥2 · value med) `docs/api/core.md` is one page of 7,769 lines.**
5. **(age 2 · value med) Try `grmob ios -run`** once with a simulator booted
   and once with none booted, to read the hint. The flag exists
   (`cmd/grmob/native.go:352`) but has never been run.
6. **(age ≥2 · value low) Migrate hand-rolls to the new comps:**
   - the tutorial's `stepper` helper (`examples/tutorial/widgets.go:206`)
   - `checkRow` (`chapter1.go:96`, also used by chapters 4 and 6.4)
   - confirm flows in `todoapp`/`mobileapp`
   - the signup flow's step header
   - hand-rolled "⋯" overflow buttons

   Screenshots need retaking afterwards. Leave the "⋯" at `chapter6.go:734`
   alone: that is the ActionSheet lesson opening its own sheet.
7. **(age ≥2 · value low) A looping `Transition`** so `Spinner` can drop its
   stepping. It would also allow a sliding Drawer. `core/animation.go` has no
   repeat today.
8. **(age ≥2 · value low) Current-item semantics.** These are all English
   fallbacks: BottomBar's ", selected", StepIndicator's ", done" and
   ", current", and the ", selected" in `SheetAction.Checked` and Drawer. The
   proper shape is an `aria-current`-style state in core.
9. **(age ≥2 · value low) The Android remedy drill has never run.** Candidate
   for deletion.
10. **(age ≥2 · value low) `hero.png` is a picture of pictures, and its parts
    are stored twice.** Candidate for deletion.
11. **(age ≥2 · value low) A third face is still a skip.** Candidate for
    deletion.
12. **(age 2 · value low) No `grmob` command refreshes a scaffolded app's
    `wasm/index.html`.** `-refresh` only re-copies `android/` and `ios/`. The
    next change to the host page's CSS will need another README note unless
    such a command exists.
13. **(age ≥2 · non-goal) Tracked-Go-file count sentences** in `wasm/verify`
    are hand-edited on every commit that adds Go files. This is working as
    designed.
14. **(age ≥2 · non-goal) Windowing.** Declined in `non_goals.md`.
    `TestWhatWindowingWouldSave` (`examples/tutorial/app_test.go:863`) is the
    profile to re-run.
15. **(age 2 · non-goal) Rename `docs/components.md` to `comps.md`.** The page
    keeps its name so its URL stays stable.
16. **(age 2 · non-goal) Rewrite `components` in the older plans.** Only the
    two most recent plans were updated; the rest stay as history.
17. **(age 2 · non-goal) Trim the copied Android shell's permission
    declarations.** Removing one is a build-time choice the app should make;
    see `androidPatches`.
18. **(age 2 · non-goal) Replace the iOS usage strings further.** They already
    name the app.
19. **(age 2 · non-goal) C4 `Carousel`** stays blocked until the host can
    report a scroll offset (`OnScroll` or a paged-scroll node). Core has
    neither.
20. **(age 1 · value med) ARIA combobox for SearchableSelect.** Needs:
    - a `RoleComboBox` with aria-expanded and aria-controls (optionally
      aria-activedescendant) in core
    - support in both web exporters, the ARIA fixture and the runtime keyboard

    This is renderer work.
21. **(age 1 · value low) On the web, focus falls to the page after a keyboard
    pick in SearchableSelect.** Returning focus on the web only would need a
    host distinction that core doesn't expose.
22. **(age 1 · value low) `aria-haspopup` for the Menu and DatePicker
    triggers.** Core has no way to say a button opens a dialog or menu.
23. **(age 0 · value high) Android system back.** The Android shell never
    handles the back button: there is no `BackHandler`,
    `OnBackPressedCallback` or `onKeyDown` anywhere under `android/`, and core
    has no host "back" event. So back closes neither a Drawer nor core's
    navigation stack. Only a Modal closes on it, because `Dialog` does that
    itself. One back-press hook in core, wired on Compose, would serve both.
    *(Widened this session: doc 3 raised it as a Drawer-only gap.)*
24. **(age 0 · value med) Web focus containment for Drawer.** Tabbing past the
    panel reaches the screen behind it, which is aria-hidden. Needs an
    `inert`-style prop in core and the web renderers. Neither the runtime nor
    core has one today.
25. **(age 0 · value low) `MaxWidth` on the natives.** `core.Style.MaxWidth`
    exists, but nothing in Kotlin or Swift reads it, so Drawer can't cap a
    percentage width on a tablet.

Read by value instead: **high** 1, 2, 23 · **medium** 3, 4, 5, 20, 24 ·
**low** 6–12, 21, 22, 25 · **non-goal** 13–19.
