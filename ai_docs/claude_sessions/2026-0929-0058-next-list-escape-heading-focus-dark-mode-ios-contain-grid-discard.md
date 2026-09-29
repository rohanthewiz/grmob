# Next list: Escape, focus on a heading, dark mode, iOS `.contain`, the grid's ✕

Session: `c08874f6-f5f7-468f-9289-58366fb25806`

## Ask

Work every item in `ai_docs/todo/next-list.md` that needs no user attention,
answering API decisions with a best recommendation (new this time: the last
session left them alone). Skip only what needs the user. Commit after each
item, `sess-wrap` at the end.

## Commits

| Commit | Item |
| --- | --- |
| d840fc2 | N-084: the grid's ✕ keeps focus; the runtime drops events its own batch causes |
| 51a525b | N-034: signup's PINInput code step (chat half declined on recommendation) |
| 063e834 | N-078: iOS `.contain` for a labelled container with 2+ members |
| b958582 | N-050: the hosts report the colour scheme; container TextColor inherits on natives |
| 51cca4b | N-057: `core.OnEscape` |
| 4dcadf9 | N-069: `core.Focus` reaches a heading; `Wizard.TitleRef` |
| 86f0191 | Recommendations written into the decision items |

## N-084: EditableGrid's ✕ and a stale blur

- **Choice:** a press that keeps focus, over a blur payload naming where focus
  went. New `core.PressKeepsFocus()` (wire `pressKeepsFocus: true`). The web
  runtime turns it into a `mousedown` preventDefault (`applyPressKeepsFocus`,
  total on both paths). The natives need nothing.
  - `mousedown` rather than `pointerdown`: Chrome still focuses after a
    prevented pointerdown, and a touch tap arrives as a compat mousedown.
- **The ✕ as part of the field:** the ✕ also reports OnFocus/OnBlur as the
  field's, so Tab onto it is not a blur.
- **The second defect it exposed:**
  - The field was now focused when the discard removed it.
  - Chrome fires `blur` synchronously inside `removeChild`, while the element
    is still connected.
  - The batch is applied inside the click's `GoInvokeCallback`, so the blur
    re-entered Go with the removed field's `cb_4`.
  - IDs are positional, so `cb_4` now named another node's handler, and the
    next tap on the cell opened nothing.
- **Fix:** `mount`/`patch` run under `withTreeApplied` (`applyingTree`), and
  `dispatchFromElement` drops any element event fired while it is non-zero.
- **Measured** in headless Chrome with a scratch CDP probe (real mouse events)
  on 4.37:
  - 60, 120, 300 and 600ms holds all discard. Before, 300 and 600 committed.
  - Tab onto the ✕ outlasts the grace, and Enter there discards.
  - Return commits and lands on row 2.
  - A 300ms tap on another cell commits and opens it.
- **Tests:**
  - `TestEditableGridDiscardIsOneFocusUnitWithTheField`.
  - Three runtime tests: pressKeepsFocus twice, and the guard, which fakes
    Chrome's blur in `el.remove` and is mutation-tested.

## N-034: signup

- **The step:** a passing submit sets `pending`. A verification screen takes
  the code (246810, stated on screen: there is no mail server) in
  `comps.PINInput`, and it renders in `ctx.Scope("verify")` because
  PINInput holds hooks.
- **Wrong code:** cleared, with "That code is not the one we sent".
- **"Use a different address":** returns to the form with its values kept.
- **Not retaken:** `signup.png` shows the form's mismatch error, so it is
  unaffected, and its claims test passes.
- **Chat half declined on recommendation:** the example teaches `core.For` and
  `core.Keyed`.

## N-078: iOS `.contain`

- **The rule, by content:** a labelled container is `.ignore` if every child
  is hidden, `.contain` with 2+ *members* below it, and `.combine` otherwise
  (`GrMobNode.holdsControls`, `grMobChildMode`).
- **What counts as a member:**
  - a control type;
  - a pressable node with a label or a child that is not hidden;
  - a nested labelled container (Compose's merge boundaries).
- **Walk:** it stops at the second member and does not enter a member.
- **First cut was wrong:** operable-only members left Wizard (one Next)
  combining its step body, and Poll results as one stop. Nested named
  containers fixed both.
- **Census:** `ios/verify/childmode.go` + `childmode.swift` render 21 bundled
  widgets in Go and hold every labelled container to a shape table
  (mutation-tested).
- **Simulator:**
  - Grid cells are Buttons by name; "Price" holds labelled Minimum/Maximum
    sliders; "blue" is Selected, not the group.
  - The round-four tests were rewritten to reach members by name.
  - Range test: `adjust(toNormalizedSliderPosition:)` does nothing on a
    text-valued slider. It now drags from the thumb's centre, after making
    sure the slider is not under the status bar.
  - The Tutorial UI suite: 50/51, and `testAudioPlayer` (streams) passed alone.

## N-050: colour scheme

- **Core:** `core.Window.ColorScheme` + `Dark()`, sent as `scheme` on the
  existing "window" report.
- **Where each host reads it:**
  - Android: the configuration's night bit (a switch recreates the Activity).
  - iOS: the reader's `colorScheme` environment, watched with `onChange`.
  - Web: `prefers-color-scheme`, with a listener.
- **The tutorial:**
  - Follows the window's scheme until a page sends "theme" (`pageSaid`), so
    the web page's switch still wins.
  - On a native it paints the root in darkTheme's Background and TextPrimary
    (`paintPage`, a copied node, no wrapper).
- **Divergence found and fixed:** a container's TextColor inherits on the web
  (CSS `color`) but did not on the natives, so 1.1's plain Texts were black on
  the dark card.
  - Compose now provides `LocalContentColor`.
  - SwiftUI sets the container's foreground style (`grMobInk`, applied after
    grMobBox at seven container call sites).
  - Documented on `core.TextColor`.
- **Seen:**
  - Emulator: `cmd uimode night yes/no` on 1.1, 4.37 and 5.9.
  - Simulator: `simctl ui appearance dark/light` on 1.1, switching live.
- **Raised N-085:** with the old framework, the simulator in dark drew 1.1's
  light theme on SwiftUI's black surface (title invisible). Android's
  status-bar icons go white over its always-light window.

## N-057: `core.OnEscape`

- **Shape:** a claim of its own beside OnBack. Escape closes layers; it must
  not pop a route or fire an AppBar back arrow. An open Modal's dismiss counts
  as a claim, and Drawer's open panel carries it.
- **Web:**
  - One page-wide keydown listener, now shared with the shortcuts, because
    keynav_test pins exactly one; the last claimant in document order wins.
  - Headless Chrome: 4.18's drawer and 6.6's dialog close, and stay open with
    the listener removed.
- **Android:**
  - The Activity walks the tree (`lastEscapeClaim`).
  - A Dialog did **not** close by itself: the platform maps Escape to back only
    with predictive back off, and the manifest opts in. So `DialogEscape`
    wraps the dialog window's `Window.Callback`.
  - Emulator: both close, and back still closes the dialog.
- **iOS, unverified:**
  - The claim is an invisible Button with an Escape keyboardShortcut.
  - XCUITest's Escape reached neither it nor `.cancelAction`, while the same
    claim bound to Control+Option+E closed the drawer.
  - osascript is not allowed to send keystrokes on this machine.
  - The test is a strict `XCTExpectFailure`, like F6's.

## N-069: focus on a heading

- **Web:** a focus command on a node the browser does not focus sets
  `tabindex="-1"` first.
- **iOS:** a stamped Text gets `AccessibilityFocusState` (once per epoch),
  through `GrMobFocusIfStamped`, so only stamped Texts hold the state.
- **Compose:** nothing, since there is no API for TalkBack focus.
- **Wizard:** `Wizard.TitleRef` (the caller's ref). `change()` focuses the
  title after OnChange. The title is named by applying `FocusTarget` to the
  node `core.Text` renders, because Text takes style props only.
- **Seen in Chrome on 4.35:** after Next, focus is the "Gift note" heading.
  Without the tabindex it is the page body.

## Recommendations recorded, not acted on

| Item | Recommendation |
| --- | --- |
| N-010 | per-host (web sr-only table, iOS AXChartDescriptor), no primitive |
| N-013 | leave it (non-goal) |
| N-015 | leave Compose |
| N-022 | keep, low |
| N-027 | decline |
| N-061 | non-goal |
| N-073 | do it as its own session |
| N-074 | promote with `*Ink` renames |
| N-083 | decline |

## Verification

- `go test ./...` passes; the API docs were regenerated.
- `wasm/verify/run.sh` passes, browser checks included.
- `ios/verify/run.sh` passes, with the new census.
- The tracked-Go-file count moved to 672. `sharedparse_test` asked for five
  sentences in repowalks/timings to be updated, and they were.

## Gotchas

- **Kotlin:** `KeyEvent.hasModifiers(int)` needs a mask; use
  `hasNoModifiers()`.
- **uiautomator:** lesson code samples contain the demo's words ("Notebook",
  "Delete note?"). Witness with a control only the open layer has ("Close
  Notebook", "Keep").
- **iOS framework:** the installed app carries the last
  `ios/build.sh ./examples/tutorial`. The first dark screenshot was of the old
  framework.
- **Committing interleaved changes:** a scratch `stage_hunks.py`
  (`git apply --cached` of selected hunks) split them per item.
  `git stash --keep-index` let the API docs be generated for exactly one
  commit.

## Left on the devices

- The emulator's night mode is no and the simulator's appearance is light,
  both as found.
- The emulator and simulator have today's tutorial builds installed.
- `/tmp` probe files were removed.

## Next

Closed: N-034, N-050, N-057, N-069, N-078, N-084. Declined: None (N-034's chat
half, on recommendation). Raised: N-085. Deferred: None. Promoted: None.
Updated: N-004, N-010, N-013, N-015, N-022, N-027, N-061, N-066, N-068,
N-073, N-074, N-083. Full list: `ai_docs/todo/next-list.md`.
