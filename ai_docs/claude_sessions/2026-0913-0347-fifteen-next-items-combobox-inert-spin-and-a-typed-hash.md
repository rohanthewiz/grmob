# Fifteen Next items: a combobox, inert, Spin, and a typed hash

**Session:** https://claude.ai/code/session_013PPKnYoYbsVv3vW7oAXA6p
**Date:** 2026-09-13 03:47
**Branch:** master

## 1. The asks

1. `/sl` loaded "Nested scrolls on iOS, and sideways scrolls on Android".
2. "What is in the Next list" — listed by value.
3. "Let's do 4, 5, 20, 23, 6, 7, 9, 10, 11, 12, 21, 22, 24, 27, 28".
4. `/sw`: save this doc, commit all, push.

## 2. How the work was split

Six forked agents in git worktrees (`.claude/worktrees/`), each committing on its
own branch, merged into master with `--no-ff`. The main session did 5, 23, 27
and 28 directly, then merged, resolved conflicts and re-verified.

| Stream | Items | Who |
|---|---|---|
| API reference split | 4 | fork |
| MaxWidth on natives | 24 | fork |
| Looping rotation | 7 | fork |
| Tutorial hand-rolls | 6 | fork |
| Tooling | 9, 10, 11, 12 | fork |
| Combobox + haspopup | 20, 21, 22 | fork |
| `ios -run`, Inert, null Changes, typed hash | 5, 23, 27, 28 | main |

Forks were told not to touch the emulator or simulator; native runs were done by
the main session after merging.

Merge conflicts were only ever in generated or counted files:
- `docs/api/*`: resolved by taking one side and re-running `go run ./internal/apidoc/gen`.
- The tracked-Go-file count sentences in `wasm/verify/repowalks_test.go` (lines
  30, 532, 730) and `timings_test.go` (823, 892): 499 → 500 → 501 → 502.
- `cmd/grmob/main.go` usage comment (`-run` and `web` lines, both kept).
- `GrMobStyle.kt` imports (both sets kept).
- `docs/concepts/styling-and-theming.md` platform table (Spin row + MaxWidth's own row + `Inert` web-only).

## 3. Per item

### 4. `docs/api/core.md` split (7d92334)
- `apidoc.Pkg.Topics []Topic{Slug, Title, Blurb, Files}`; only core uses it.
- `core.md` is now a 409-line overview (package doc, topics table, index of
  declarations linking to topic pages) plus ten sibling pages
  `core-{views,layout,style,theme,controls,editors,events,navigation,accessibility,device}.md`.
- Placement is by declaring file; constructors by their own file, not go/doc's
  grouping under the returned type.
- The symbol index carries the page, so `[core.Node]` from hooks resolves to
  `core-views.md#type-node`.
- Guards: an unassigned declaring file, a missing file, a file in two topics,
  and a page-name collision all fail generation. `TestTopicsPartitionTheirPackage`.
- `mkdocs.yml` nests the core section.
- The guard fired on the combobox merge: `core/popup.go` had no topic. Fixed in
  4f91a5a by adding it to the accessibility topic. The merge commit a0ef492
  itself carries stale pages (a `;`-chained command committed after the
  generator failed); 4f91a5a regenerates them.

### 5. `grmob ios -run` (7037c60)
- Scaffolded `grmob new … -replace <repo> -no-build` in the scratchpad.
- **Booted simulator:** bind, xcodegen, simulator build, `simctl install booted`,
  launch — app shows "Iosrun / Count: 0 / − +".
- **None booted:** the full bind and build ran, then `simctl install` printed
  "No devices are booted" and grmob added its hint. Correct, but minutes late.
- Fix: `bootedSimulators()` reads `xcrun simctl list devices booted -j`;
  `countBooted` parses the JSON and counts `state == "Booted"`. With `-run` and
  zero booted, the command stops before the bind. A listing that fails to read
  does not block. New `cmd/grmob/native_test.go`.
- Re-run with none booted: fails in 0.9s with the hint. Simulator booted again afterwards.

### 27. `Changes: null` (f97625e)
- `reconcile/patch.go`: `wireProps` (nil → `map[string]any{}`) and `wireStyle`
  (nil → `&core.Style{}`, which marshals `{}` since every field is omitzero).
  Only the wire value changes; the node tree keeps its nil.
- Checking the transcripts found this was live, not hypothetical: the runtime's
  `update-style` case called `applyStyle(el, null)` and threw
  `Cannot read properties of null (reading 'FontSize')`, dropping the batch.
  Now `p.Changes || {}`, matching createElement's `node.Style || {}`.
- Tests: `TestDiffLosingEverythingSendsEmptyNotNull` (asserts the marshalled
  `"Changes":{}`), and two runtime tests in `styleless_test.mjs` (null and {}
  both clear the style and the next patch in the batch still applies).
- The test run for this commit was piped through `tail`, which hid a failing
  `wasm/verify` (the count sentences); 99f2076 fixed it. Use `set -o pipefail`.

### 23. `core.Inert` (7373318)
- `Style.Inert bool` + `core.Inert(bool)`; merge sets it when true.
- Why not reuse: AccessibilityHidden also marks nodes that must stay tappable
  (Drawer's scrim); Disabled is announced as disabled. The doc has the
  tree / Tab / pointer / announced table.
- htmlout writes `inert=""` outside `accessibilityAttrs` (which returns early
  for hidden nodes). The runtime sets/removes the attribute on every style
  pass (not `setOrRemove`, which treats "" as remove).
- Natives do not read it; a hardware keyboard there can still reach the screen.
  Documented on the field.
- Drawer's content layer: `AccessibilityHidden(), Inert(true)` while open.
  Tutorial 4.18 key point updated.
- **Chrome check (CDP `Input.dispatchKeyEvent`, lesson 4.18, drawer open):**
  Shift-Tab from ✕ goes to the code listing above the demo, never the ☰. Control
  with `inert` removed by hand: Shift-Tab lands on "Open navigation".
- The Claude-in-Chrome extension's `key Tab` did not move focus at all (no
  focusin fired); use CDP for keyboard facts.

### 28. Typed hash while a claim is on screen (cb6f968, 1c7fe91)
- **Chrome, HEAD's runtime:** deep-link `#2.3`, then `Page.navigate` to `#4.18`
  (a browser-initiated fragment push). The popstate was read as back: the
  lesson's handler ran, the tab showed contents at `/`, 4.18 never opened, and
  the next back re-opened 2.3. Worse than the Next item described.
- Fix in the runtime:
  - `backEntryMark` records `{navigation.currentEntry.index, history.length}`
    whenever the runtime's entry is current (after push and on every sync);
    cleared on unwind.
  - `landedAboveRuntimeEntry()`: `FOLDED_STATE` on the entry, else Navigation
    API index greater than the mark, else `history.length` grew.
  - On such a popstate: mark the entry `grmobFolded`, set `backUnwinding` with
    `backUnwindHref = location.href`, `history.back()`. The existing unwind
    carries the typed URL onto the runtime's entry. No handler runs.
- **Chrome with the fold:** typed `#4.18` → lesson 4.18, runtime entry current at
  `#4.18`; back 1 → contents; back 2 leaves the site.
- Tests in `browserback_test.mjs` (model gains `length`, optional `navigation`,
  `typeURL`): fold with and without the Navigation API, Forward onto a folded
  entry, and an unwound entry leaves no stale mark.
- `docs/platforms/wasm.md` Browser back gains the bullet.
- Known limit: the `history.length` fallback misreads a push at a browser's
  session-history cap (Chrome 50).

### 6. Tutorial hand-rolls (3cf0d92)
- `stepper` wraps `comps.Stepper`; `checkRow` wraps `comps.CheckboxRow` (~30
  demo toggles across chapters 1–8, including 6.4 and 4.x); chapter 8's debug
  checkbox → CheckboxRow; chapter 4's Marker/GPS rows → `comps.SwitchRow`.
  1.1's three toggles stack in a Column.
- Left: 6.4's modal card (teaches core.Modal), chapter 2's checkbox lesson and
  mobileapp's rows (leading checkbox is the lesson; mobileapp feeds the replay),
  chapter6.go:734 (excluded). No confirm flows in todoapp/mobileapp, no step
  header in signup, no other "⋯".
- `docs/images/tutorial-lesson.png` retaken, scrolled to the code block; claim
  and script updated.

### 7. `core.Spin` (5ca48f3)
- `core.Spin(periodMs)`: one revolution per period, negative anticlockwise, 0
  still, added to Rotate. `Style.Spin int`. Deliberately rotation only.
- Web: `animation: grmob-spin <ms>ms linear infinite [reverse]`, keyframes on the
  `rotate` property (`core.SpinKeyframes`), injected once by the runtime; htmlout
  writes a `<head>` only when something spins.
- Compose: `SpinElement`/`SpinNode` after `Modifier.rotate`, only when spin != 0.
- SwiftUI: `GrMobSpin`, a `TimelineView(.animation(paused:))` always in the box
  chain, so identity is stable when Spin flips.
- Spinner: `Spin(1000)` on the ring; no state slot, no interval, conditional-safe.
- Drawer documents why it does not slide (no offset style; Display none panels
  have nothing to animate from).
- **Chrome (lesson 4.15):** nothing animating before the tap; after,
  `grmob-spin` running, 1000ms, infinite, rotate 252°.
- **Android emulator:** two frames 250ms apart show the dot moved about 90°.
- **iOS simulator:** see §4 — the ring is collapsed on iOS before and after.

### 9. Android remedy drill (f5d504d)
- Already passed in CI (run 34740773216). Passed locally with
  `ANDROID_HOME=~/Library/Android/sdk` and a cold gradle home; iOS drill too.
  The second remedy's gradle output is now discarded like the first.

### 10. `hero.png` (5603db9)
- Kept. A subtest reads the layout from `hero.html` and checks: height a whole
  multiple of the page height (crops), width from the parts' aspect ratios
  within 1 CSS px, and each 12px square's mean colour within 13/255 of the part
  (stale or swapped part). Worst on the committed images 9.3/255; one glyph
  painted out reads 17.2 and 24.2.

### 11. Third face (94583a3)
- `GRMOB_INK_FACE` writes a face into Chrome's scratch profile. Eight macOS
  serifs (Times New Roman, Charter, Didot, Bodoni 72, Baskerville, Georgia, Big
  Caslon, Hoefler Text) all hit "WHICH DOES NOT SEPARATE" for
  `INK_ASCENDER_SEPARATION`; Georgia, Big Caslon, Hoefler Text also for
  `INK_ROW_ROUNDING`. Control: Times with itself removed from the calibrated
  list brackets both numbers. Skipping uncalibrated faces is right. Table beside
  `INK_CALIBRATED_ON`.

### 12. `grmob web` (92fa242)
- `grmob web [-refresh] [-no-build]` renders the host page from go.mod's grmob,
  byte-compares: missing → write; same → silent; different → warn; `-refresh` →
  overwrite; then `build.sh`. `renderTemplate` shared with `new`.
  `TestWebHostPageSync`.
- Run on the scaffolded app: unchanged silent; an appended comment warns;
  `-refresh` removes it.

### 24. MaxWidth on natives (696b608)
- px / bare number / %, `""`/`none`/`auto` uncapped; caps the padded box,
  margin outside; never grows a box; wider Width loses; stretched Column child
  fills to the cap at the start edge (mirrored RTL).
- Compose: one `widthModifier` handles Width and MaxWidth inside the margin.
- SwiftUI: `GrMobMaxWidthLayout` (wrapped in `GrMobMaxWidthModifier` after a
  swiftc signal-6 crash) outside the grow frame; points cap folded into fixed
  frames and the min-content estimate; arithmetic in `GrMobFlex.swift`
  (`GrMobMaxWidth`) for ios/verify.
- `mobile/verify/maxwidth_test.go` (the 501st Go file).
- Known documented gap: a FlexGrow child in a Row whose cap binds keeps its share.
- iOS DatePicker sheet (lesson 4.9) is pixel-identical to the pre-merge build:
  the card is 340pt on a 402pt window, so the 360 cap never binds there.

### 20, 21, 22. Combobox, focus after pick, aria-haspopup (dc2e657)
- `core.RoleComboBox` on the input (`SearchField.InputStyle`); `aria-expanded`
  only while rows show; `aria-controls` to the listbox id (`ID` field or from
  Label); rows `<ID>-option-N`.
- Runtime keyboard: focus stays in the field; arrows move an active option
  (`aria-activedescendant`, 2px outline in text colour); Enter picks; Escape,
  Home, End, left/right, typing clear it; listbox holds no tab stop. Enter on a
  reached option does not also run Next.
- 21: a keyboard pick keeps focus; the runtime ignores the one blur that
  DismissKeyboard causes within 500ms. A tap still dismisses.
- 22: `core.PopupKind` with `PopupDialog`, `Style.AccessibilityHasPopup`;
  written for button, link, tab, columnheader, combobox and a plain Button.
  Menu's trigger carries it; DatePicker's trigger does and is now RoleButton.
- ARIA fixture parser now drops attributes the spec marks "deprecated on this
  role"; `combobox` left the near misses. Natives map neither, noted beside each.
- Fork's headless Chrome run on lesson 4.17: "an" → five options, none tab
  stops; two arrows → France active, focus in Country; Enter → value France,
  focus still in Country; Tab from an open list → clear button; tapping Germany
  picks and moves focus to the page.

## 4. Native pass after merging

- `ios/build.sh ./examples/tutorial`, xcodegen, and a throwaway XCUITest
  (`ZZProbeUITests`, deleted) using `app.open(URL("grmob://lesson/…"))`: 4.15
  spinner, 4.9 DatePicker sheet, 4.6 swipes. Then the same probe against a
  detached worktree at e51604e (before the Spin merge) for a baseline.
- `android/build.sh ./examples/tutorial` + `./gradlew installDebug --offline`
  (needs `ANDROID_HOME` exported); deep link via `am start -d grmob://lesson/4.15`,
  `uiautomator dump` to find "Simulate loading", `input tap`, two `screencap`s.
  Baseline the same way at e51604e, then reinstalled HEAD.
- A small Go crop/diff/4x-pair tool in the scratchpad (no PIL here).

Findings, each confirmed pre-existing against the baseline:
- **iOS Spinner ring is collapsed** to about 6pt with no dot, and does not move
  between frames. So Spin cannot be seen on iOS until the ring renders.
- **iOS code blocks draw as blank dark bars** in XCUITest screenshots.
- **Android Spinner dot is clipped in half** at the ring's rim (the web draws a
  whole dot inset from the border). Compose chain is rotate → spin → clip →
  border; the stepped Rotate version clips identically.

## 5. Verification on the final tree (4f91a5a)

- `go test ./...` — passes.
- `sh wasm/verify/run.sh` — Node suite and the headless Chrome checks pass.
- `sh ios/verify/run.sh` — every stage OK, including Release optimisation and
  the iOS SDK type-check.
- `android/verify/sources.sh` with `ANDROID_HOME` — runtime and app compile.
- **Not verified:**
  - A screen reader announcing the combobox's active option, or "pop-up" on
    Menu/DatePicker triggers.
  - MaxWidth where the cap binds (tablet, landscape, RTL) on either native.
  - Spin on iOS by eye (ring collapsed), and "Remove animations" on Android.
  - The cost of a paused TimelineView per node in a long SwiftUI list.
  - `GRMOB_INK_FACE`'s Preferences write has no test; hero's 13/255 tolerance
    was measured on Chrome 152.

Process notes:
- `android/local.properties` does not exist; export
  `ANDROID_HOME=~/Library/Android/sdk` for gradle and `sources.sh`.
- The emulator holds HEAD's tutorial build; `ios/Frameworks` holds the tutorial.
- A parallel Bash call that `cd`s moves the session's cwd; use absolute paths.
- Pipe test output through `tail` only with `set -o pipefail`, or the commit
  after `&&` runs on a failure.
- Chain `go run ./internal/apidoc/gen` with `&&` before `git commit`.
- Claude-in-Chrome key presses don't trigger default Tab navigation; CDP does.
- Forks in worktrees merge cleanly except generated docs and the count sentences;
  regenerate and re-count after each merge.

## Next

**Legend:** *age* is how many session docs ago the item was first raised,
counted from this doc (0 means this doc raised it; `≥ k` means older than the
last rebuild's window). *value* is the payoff of doing it, not the effort:
**high** means something is worked around today or a second consumer has
arrived; **medium** means it blocks one named thing or is a visible defect;
**low** means nobody has hit the gap yet. Sorted by age, oldest first; new items
last.

1. **(age ≥6 · value high) Tag a release.** v0.3.0 is the latest tag, with 53
   commits since. Tier A/B/C comps, radio roles, system and browser back,
   lessons open on Android, both scroll axes guarded, MaxWidth on natives,
   Spin, Inert, the combobox, `grmob web` make a natural v0.4.0. Still open:
   run `go run github.com/rohanthewiz/grmob/cmd/grmob@latest new` on a clean
   machine without `-replace`.
2. **(age ≥6 · value high) Run lessons 6.6, 6.7, 4.9, 4.15–4.18 on a simulator
   and a device.** Every lesson opens on the Android emulator; on iOS
   `app.open(URL)` in an XCUITest reaches any lesson without a prompt.
   Unchecked on real hardware:
   - settings-row one-tap, `Screen.Footer` pinning, the iOS sheet `Dialog`
   - ActionSheet filler placement, Timeline row stretch, horizontal
     StepIndicator scroll, radio semantics
   - CodeEditor's toolbar role on Compose, a Modal inside a ListRow's trailing
     Row (Menu)
   - SearchableSelect: Next action and keyboard dismiss on pick; the combobox
     role is web-only, so check the natives still read field + list + status
   - Drawer: 100% layers in a pinned-height ZStack, the panel's `Height 100%` in
     an HStack, `AccessibilityHidden` confining TalkBack/VoiceOver, `core.Focus`
     on a Button
   - **System back:** 4.18 drawer open → back closes it, second back to
     contents; 4.16 AppBar `OnBack` note. Find ☰ by node bounds, not text.
   - **Predictive back** previews home from contents only.
   - **Double back** from a lesson lands on contents and opens nothing.
   - **Unbounded arms by eye:** 4.6 `grouped` and `banded` on Android; `banded`
     and a drag on a 4.6 list on iOS (anchor on a row title, not "Load more").
   - **MaxWidth where it binds:** DatePicker card (360px) and the chapter 4/6
     ActionSheet/Menu panels (520px) on a tablet or landscape; a Drawer with a
     percentage Width and PanelStyle MaxWidth; a capped stretched child in RTL.
   - **Spin:** smooth ring on Android with "Remove animations" on and off;
     iOS once the ring renders (item 19).
   - **Screen readers:** the combobox's active option on the web
     (VoiceOver/NVDA), "pop-up" on Menu and DatePicker triggers.
3. **(age ≥6 · value med) Launch the scaffolded app on a device.** The iOS
   simulator run is done (§3, item 5). Note the scaffold's `−`/`+` Row sits
   indented from the text above it (a theme Row's screen inset); decide whether
   the template wants `Padding(0)`.
4. **(age ≥6 · value low) Current-item semantics.** BottomBar's ", selected",
   StepIndicator's ", done"/", current", `SheetAction.Checked` and Drawer are
   English fallbacks. The shape is an `aria-current`-style state in core.
5. **(age ≥6 · non-goal) Tracked-Go-file count sentences** in `wasm/verify` are
   hand-edited when Go files are added. Now 502.
6. **(age ≥6 · non-goal) Windowing.** Declined in `non_goals.md`;
   `TestWhatWindowingWouldSave` is the profile.
7. **(age ≥6 · non-goal) Rename `docs/components.md` to `comps.md`.**
8. **(age ≥6 · non-goal) Rewrite `components` in the older plans.**
9. **(age ≥6 · non-goal) Trim the copied Android shell's permissions.**
10. **(age ≥6 · non-goal) Replace the iOS usage strings further.**
11. **(age ≥6 · non-goal) C4 `Carousel`** until the host reports a scroll offset.
12. **(age 4 · non-goal) A parent that gains `onBack` after its descendants
    outranks them** on Android; the web ranks by document order.
13. **(age 4 · non-goal) An AppBar outside the Navigator** with a custom `OnBack`
    is outranked by the Navigator's pop on Android. Put it inside the route.
14. **(age 3 · non-goal) Forward does not re-open a screen left by browser back.**
15. **(age 2 · non-goal) Sticky headers, placement animation and `OnEndReached`
    in a List with no viewport** on Compose. Give the List a Height.
16. **(age 1 · value low) Sideways nesting on iOS is unmeasured.** A
    `Horizontal()` Scroll/TextGrid/CodeEditor inside a `Horizontal()` Scroll; the
    `zzhscroll` shape plus a frame dump is the probe.
17. **(age 1 · value low) Whether GrMobList's LazyVStack stays lazy in a
    scrolled page on iOS.**
18. **(age 1 · value low) Nothing holds the iOS nested-scroll measurement.** A
    kept XCUITest (deep link, compare a List's frame with its last row's) could.
19. **(age 0 · value med) iOS Spinner ring is collapsed.** About 6pt, no dot,
    before and after Spin (baseline e51604e). The ring is a Column with Width/
    Height px, BorderRadius, BorderWidth 2 and a Box child; find which modifier
    loses the fixed frame. Blocks seeing Spin on iOS.
20. **(age 0 · value med) iOS code blocks are blank dark bars** in XCUITest
    screenshots of lessons 4.15 (before and after this session). Check by eye on
    the simulator whether the UITextView text is really missing or only absent
    from `XCUIScreen` captures.
21. **(age 0 · value med) DatePicker's trigger is not reachable by Tab on the
    web.** Pre-existing; its new RoleButton does not change that.
22. **(age 0 · value low) Android Spinner dot is clipped in half** at the ring's
    rim; the web insets it inside the border. Compose's `clip(shape)` plus
    `border` do not inset content the way CSS border-box does.
23. **(age 0 · value low) The cost of a paused TimelineView per node** in
    SwiftUI's box chain (`GrMobSpin`). Profile a long list (4.3/4.6) against
    e51604e with Instruments before adding more always-on modifiers.
24. **(age 0 · value low) Reduced motion.** No platform reads the reduce-motion
    setting for Spin or Transition.
25. **(age 0 · value low) Drawer slide.** Needs an offset/translate style and a
    panel that is not created from Display none on open.
26. **(age 0 · value low) Tutorial disagrees with itself on checkbox placement.**
    Chapter 2 says a checkbox leads its row; CheckboxRow (now used by the demo
    toggles) puts it at the end.
27. **(age 0 · value low) `tutorial-lesson.png` clips a line its claim lists**
    ("func Profile(ctx *core.Context) core.View {" at "core.Vie"). The claim test
    reads the DOM, so it passes.
28. **(age 0 · value low) `grmob new`'s closing message does not mention
    `grmob web`.**
29. **(age 0 · value low) `GRMOB_INK_FACE`'s Preferences write is untested**, and
    hero's 13/255 tolerance was measured on this Mac's Chrome 152.
30. **(age 0 · value low) `core-style.md` is still 1,762 lines and `comps.md`
    5,932.** `Pkg.Topics` can split both.
31. **(age 0 · value low) `comps.Stepper`'s doc says the examples hand-roll it
    three times in chapter 1.** True as history, false now.
32. **(age 0 · non-goal) MaxWidth with a growing sibling on the natives.** A
    FlexGrow child in a Row whose cap binds keeps its share; CSS redistributes.
    Documented.
33. **(age 0 · non-goal) The typed-hash fold's `history.length` fallback**
    misreads a push at a browser's session-history cap; the Navigation API path
    has no such limit.
34. **(age 0 · non-goal) A page's own `pushState` while a claim is on screen**
    fires no popstate, so the runtime pushes a second entry. Pages should use
    `replaceState` or set `GrMobBrowserBack = false`.

Read by value instead: **high** 1, 2 · **medium** 3, 19, 20, 21 · **low** 4,
16–18, 22–31 · **non-goal** 5–15, 32–34.
