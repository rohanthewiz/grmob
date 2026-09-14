# Next list: paging keys, min sizes on the natives, and a hook the classifier refused

**Session:** https://claude.ai/code/session_013PPKnYoYbsVv3vW7oAXA6p
**Date:** 2026-09-13 19:48
**Branch:** master (779a74f → HEAD)

## 1. The asks

1. `/sl`: load the last session doc (the pre-push gofmt guard).
2. "Do 2, and what we can of 19 - 30 in reasonable batches, committing between
   each batch, and when completely done do /sess-wrap".

## 2. Commits

| Commit | Batch | Next items |
|---|---|---|
| 95791d4 | A: hook and lexer | 28, 29 |
| f25f8ec | B: PopToRoot wording (and the state walk, no code) | 26, 23 |
| 8392b2d | C: keyboard and localisation | 2 |
| 589f310 | D: native min sizes, Compose collapses, shot claims | 19, 20, 21 |
| (this doc's commit) | E: iOS UI test, device and browser checks, this doc and the untracked 1911 doc | 24, 25 |

Not done: 27 (refused, see section 8). Non-goals 22 and 30 were left alone.

## 3. Batch A: the hook and the lexer

- **28.** `.githooks/pre-push` now runs `$(go env GOROOT)/bin/gofmt`, asked from
  the repository root. Under GOTOOLCHAIN=auto that is the toolchain go.mod
  selects, the same one CI installs. If the go command fails, it falls back to
  gofmt on PATH and prints a note. On this machine `go env GOROOT` names the
  module-cache toolchain go1.26.1. A stub `go` that exits 1 took the fallback.
- **29.** Three new lexer cases:
  - a brace group and a function body;
  - case arms, single-line and multi-line, with `b|c)` and `*)`;
  - command substitution: `$(…)`, quoted `"$(…)"`, and backticks in an
    assignment.

  All passed as written. Removing `{`, `}` and `)` from the separator set
  fails all three, plus the subshell case.

## 4. Batch B: PopToRoot, and the walk

- **26.** `core.PopToRoot`'s doc and lesson 6.3's prose no longer say the root
  keeps its scroll position. Hook state survives; the scroll offset lives in
  the host view. Since 7c40723 the root is keyed per entry, so it comes back
  at its top. The doc says how to restore an offset if a root needs one.
- **23.** A throwaway test (deleted) walked the wire tree of every lesson's
  initial state, plus one state per clickable or toggle tapped from it: 525
  states. It applied Compose's LocalGrMobUnboundedHeight rules, including
  visible Modals. The only affected node is still 4.6's `List` (from 4.6's
  states, and from 4.5 after "Next ›"). No open sheet or toggled demo adds
  one, so no kept test was written.

## 5. Batch C: item 2

- **PageUp/PageDown.**
  - New `core.Style.AccessibilityKeyShortcuts` (aria-keyshortcuts). Only the
    WASM runtime writes it. htmlout leaves it out, because a static page
    can't honour it. The natives don't read it.
  - `comps.Calendar`'s arrows declare `PageUp` and `PageDown`.
  - In a grid, the key presses the nearest control that declares it (searching
    outward from the grid). The runtime sets `pendingGridPage` before the
    invoke, because under WASM the render happens inside it.
  - `syncComposite` for that grid then lands focus on the enabled cell with
    the same leaf text, or on the last enabled cell.
  - The pending landing expires after 2s, and is dropped if focus left the
    grid.
  - A disabled arrow takes the key and does nothing. With Shift or Ctrl held,
    or with no declaring control, the key is left to the page.
- **RTL.** Horizontal composites and grids swap ArrowLeft and ArrowRight when
  `isRightToLeft` is true: the computed `direction`, or the nearest `dir`
  attribute in the shim. Home and End stay logical.
- **Native ", today".** The word now comes from the platform:
  - SwiftUI: Foundation `RelativeDateTimeFormatter(.named)`;
  - Compose: ICU `RelativeDateTimeFormatter.format(THIS, DAY)`, cached per
    locale.

  swiftc printed today / aujourd’hui / heute / 今日 / اليوم / hoy. On the
  emulator, 4.9's today cell reads "Wednesday, March 11, 2026, today".
- **", done".** Still has no platform state. `StepIndicator.StepLabel` and
  `PositionLabel` hand the names to the caller.
- **Tests.**
  - Six new `keynav_test.mjs` cases. Removing the landing call fails three;
    forcing LTR fails the RTL case.
  - `TestCalendarMonthArrowsDeclareThePageKeys` and
    `TestStepIndicatorNamesComeFromTheCallerWhenGiven`.
  - The Swift and Kotlin pins in `selected_test.go` were updated.

## 6. Batch D: items 19, 20, 21

- **19, MinWidth/MinHeight.**
  - Compose: `widthModifier(width, maxWidth, minWidth)` folds the floor in
    after the cap, because CSS min-width wins over max-width. A new
    `heightModifier` is a layout lambda of the same shape. Both take px or %.
  - SwiftUI: `grMobMinimum`, a flexible frame's minimums, placed between the
    dimensions and the background. Points only.
  - The styling doc's table has a new row. `TestBothNativesApplyMinWidthAndMinHeight`
    pins parse, placement and floor order.
- **20, collapses.**
  - `LocalGrMobUnboundedWidth` is true inside a horizontal Scroll's content and
    cleared by a points Width. While it is set, RowChildren gives grow
    children no weight.
  - A List's lazy rows provide `LocalGrMobUnboundedHeight`.
  - RenderNode provides whichever locals changed in a single call (a list, not
    eight `when` arms).
  - Pinned in `unbounded_scroll_test.go`.
- **21, shot claims.** `inputTextRect` now:
  - measures with the field's `letter-spacing` (canvas `letterSpacing`);
  - places RTL values from the right edge, with CSSOM's negative scrollLeft.

  An RTL value that holds an LTR letter or a digit keeps the box rule. Four
  new Chrome cases pass. Measuring without the spacing fails one;
  `if (rtl) return null` fails two.
- **Checks.** `go test ./...`, `ios/verify/run.sh`, `android/verify/sources.sh`,
  `:app:compileDebugKotlin`, `wasm/verify/run.sh`, `wasm/shots/clipped_test.mjs`.

## 7. Batch E: devices and a browser

- **Android emulator** (tutorial build):
  - 4.6 still draws its rows ("The Narrow Gate" between the header and "Page 1
    of 3");
  - 4.9's DatePicker sheet lays out normally. MaxWidth 360 binds there, so the
    300 floor is not what sizes it;
  - today's name carries ICU's word.

  The RichTextEditor's MinHeight was not eyeballed.
- **24, iOS.** New `ios/GrMobUITests/TutorialNextUITests.swift`. The Xcode
  project is generated and untracked; `xcodegen generate` picks up the file.
  - `testNextFromTheBottomOfALessonOpensTheNextAtItsTop` **passed** on the
    iPhone 17 Pro simulator: `.id(root.viewID)` works.
  - `testATapInsideALessonKeepsItsScroll` failed its first run on the query,
    not the behaviour: the CheckboxRow is not a staticText. Now it matches
    any element. On the rerun both tests **passed** (11.4s and 11.5s): a
    CheckboxRow tap in 1.4 left the row within 2pt of where it was.
- **25, web.** A throwaway CDP script against `go run ./serve -dev -addr :8765`
  in headless Chrome:
  - from the bottom of 1.3 (scrollTop 665), "Next ›" opened 1.4 at scrollTop 0
    with its title at y=215;
  - scrolled 1.4 to 700 and touched `examples/tutorial/app.go`. The server
    rebuilt (309ms), the page swapped main.wasm without a load (a window
    marker survived), and it stayed on #1.4 at scrollTop 700.

## 8. What went wrong

- **27 was refused.** The auto-mode classifier denied writing
  `.claude/hooks/enable-githooks.sh`, a SessionStart hook that would set
  `core.hooksPath`, as "Unauthorized Persistence". It was not worked around.
  The last doc had already called it "a write to git config nobody asked for".
  It needs the user's explicit go-ahead, or a permission rule.
- **A template-literal regex lost its backslashes.** `CLIPPED` is source text
  evaluated in the page, so `\p{…}` needs `\\p{…}` in the .mjs. The first
  parse check caught it.
- **Swift pins read with `codeIn`** strip string literals, so anchors holding
  `""` never matched. Use `valuesIn`.
- **`RenderNode` is not private.** The anchor `private fun RenderNode(` missed.
- **Commands with `cd` in parallel calls.** One `go run ./internal/apidoc/gen`
  ran in another call's directory. Use absolute paths, as the gradle call
  finally did (`./gradlew` "no such file").
- **The previous session's doc (1911) was never committed.** It sat untracked
  and goes in with this one.

## Next

**Legend:** *age* is how many session docs ago the item was first raised,
counted from this doc (0 = raised here; `≥ k` = older than the last rebuild's
window). *value*: **high** = worked around today or a second consumer has
arrived; **medium** = blocks one named thing or is a visible defect; **low** =
nobody has hit the gap yet. Sorted by age, oldest first; new items last.

Done this session (previous numbering): 2, 19, 20, 21, 23, 24 (see 7),
25, 26, 28, 29.

1. **(age ≥12 · value high) Lessons on hardware, and checks still open.**
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
2. **(age ≥12 · non-goal) Tracked-Go-file count sentences** in `wasm/verify` are
   hand-edited when Go files are added. Still 505.
3. **(age ≥12 · non-goal) Windowing.** Declined in `non_goals.md`.
4. **(age ≥12 · non-goal) Rename `docs/components.md` to `comps.md`.**
5. **(age ≥12 · non-goal) Rewrite `components` in the older plans.**
6. **(age ≥12 · non-goal) Trim the copied Android shell's permissions.**
7. **(age ≥12 · non-goal) Replace the iOS usage strings further.**
8. **(age ≥12 · non-goal) C4 `Carousel`** until the host reports a scroll
   offset.
9. **(age 11 · non-goal) A parent that gains `onBack` after its descendants
   outranks them** on Android; the web ranks by document order.
10. **(age 11 · non-goal) An AppBar outside the Navigator** with a custom
    `OnBack` is outranked by the Navigator's pop on Android.
11. **(age 10 · non-goal) Forward does not re-open a screen left by browser
    back.**
12. **(age 9 · non-goal) Sticky headers, placement animation and
    `OnEndReached` in a List with no viewport** on Compose. Give the List a
    Height.
13. **(age 7 · value low) The cost of a paused TimelineView per node** in
    SwiftUI's box chain (`GrMobMotion`). Not profiled.
14. **(age 7 · non-goal) MaxWidth with a growing sibling on the natives.**
15. **(age 7 · non-goal) The typed-hash fold's `history.length` fallback.**
16. **(age 7 · non-goal) A page's own `pushState` while a claim is on screen.**
17. **(age 7 · non-goal) A Drawer's shut panel is composed on the natives.**
18. **(age 5 · non-goal) A List with no Height in a scrolled page is not lazy**
    on iOS or Android. Give the List a Height to virtualise.
19. **(age 1 · value low) The pre-push hook is off in every other checkout.**
    `core.hooksPath` is per clone. The SessionStart hook that would set it was
    written this session and refused by the auto-mode classifier (see 8).
    Needs the user's explicit go-ahead, or a permission rule, before
    `.claude/hooks/enable-githooks.sh` can be added, and `hookconfig_test.go`
    should then hold its settings entry.
20. **(age 1 · non-goal) Checking every commit in a push, not just the tip.**
21. **(age 0 · value low) Min sizes by eye on the natives.** Not checked on a
    screen: RichTextEditor's MinHeight on either native, the link prompt's
    280 floor, and any floor that actually binds. The DatePicker sheet was
    sized by its MaxWidth. A percentage MinWidth/MinHeight is ignored on iOS
    (`grMobFloorPoints`).
22. **(age 0 · value low) Paging and RTL arrows in a real Chrome with a live
    Go render.** Both are tested against dom.mjs with hand-written patches. A
    `browser.mjs` fact could run 4.9's calendar: PageDown, then focus on the
    same day in the next month.
23. **(age 0 · value low) AccessibilityKeyShortcuts beyond page keys.** Only a
    grid's PageUp/PageDown is honoured. Any other declared key is written and
    does nothing (the field's doc says to add the behaviour first). An iPad
    hardware keyboard could map it to `.keyboardShortcut`.
24. **(age 0 · non-goal) A horizontal strip shorter than its viewport** does
    not give leftover width to its growers on Compose (CSS does). They keep
    their content width; see `LocalGrMobUnboundedWidth`.
25. **(age 0 · non-goal) Mixed-direction RTL field text in shot claims** keeps
    the box rule. Bidi reordering is not arithmetic.
26. **(age 0 · value low) The ", " before the native today word is fixed.**
    Fine for TTS pauses, but not a localised list separator.
