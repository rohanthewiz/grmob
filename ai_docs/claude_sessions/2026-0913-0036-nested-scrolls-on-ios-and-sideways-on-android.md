# Nested scrolls on iOS, and sideways scrolls on Android

**Session:** https://claude.ai/code/session_013PPKnYoYbsVv3vW7oAXA6p
**Date:** 2026-09-13 00:36
**Branch:** master

## 1. The asks

1. `/sl` loaded "Lessons stop crashing on Android".
2. "Let's do item 30 and 31":
   - **30:** the same lessons on iOS. Does SwiftUI's `ScrollView { LazyVStack }`
     (GrMobList) collapse or mis-size inside a scrolled lesson (4.3, 4.6)?
   - **31:** horizontal scrolls under an unbounded width on Compose.
3. `/sw nested-scrolls-on-ios-and-sideways-on-android`: save this doc, commit all, push.

## 2. Item 30: iOS needs no unbounded arm

### How it was read

- `ios/build.sh ./examples/tutorial`, `xcodegen generate`, then
  `xcodebuild test … -only-testing:GrMobUITests/ZZNestedScrollProbe` on the
  iPhone 17 Pro simulator (iOS 26.5), with `-derivedDataPath` in the scratchpad.
- The probe was a throwaway XCUITest (deleted). `app.open(URL("grmob://lesson/4.3"))`
  reached a lesson **without** the "Open in GrMobApp?" prompt that
  `simctl openurl` puts up. That's a better route than scrolling the contents list
  (compare `LiveMapUITests.openLesson`).
- It dumped every `app.scrollViews` frame plus the frames of anchor texts, wrote
  screenshots to the scratchpad, and dragged from inside a list's frame.
- Code blocks (UITextView) don't show up in `app.scrollViews`, so the
  scroll views besides the page are the Lists.

### What it showed

| Lesson | Inner scroll view | Content inside it |
|---|---|---|
| 4.3 outline `core.List` | 310×228pt | "Matthew" at +85, "Romans" ends at +220: all six rows fit |
| 4.6 `grouped` GroupedList | 310×248pt | March band at +0, Load more button ends at +230 |
| 4.6 `banded` GroupedList | 310×143pt | Shut month bands |

- The page's ScrollView proposes no height. The inner ScrollView answers with its
  content's height, which is what the DOM draws.
- The 4.6 screenshot shows the band, three rows and Load more drawn whole.
- **Drag:** a 300pt drag starting inside 4.3's list moved the list's frame from
  y=340 to y=−119, so the page scrolled. The inner ScrollView, with nothing to
  scroll, did not keep the gesture.
- The 4.6 drag probe did not run. Its anchor "Load more" matched the demo panel's
  caption first ("Load more reveals three rows…"), which sits outside any scroll
  view.

### What changed

- `ios/GrMob/Runtime/Renderer.swift`: `GrMobList` gets a comment with these
  numbers, saying why it has no unbounded arm. Laziness under that proposal is
  marked unmeasured.
- `core/layout.go`, `core.Scroll` doc: "SwiftUI's sizing of a nested ScrollView
  has not been measured" is replaced with the measurement.
- `docs/components.md`: one sentence on SwiftUI needing no guard.

## 3. Item 31: `horizontalScrollWhenBounded`

### The helper (Renderer.kt)

It is the sideways twin of `verticalScrollWhenBounded`:

```kotlin
internal fun Modifier.horizontalScrollWhenBounded(state: ScrollState): Modifier =
    layout { measurable, constraints ->
        val bounded = if (constraints.hasBoundedWidth) constraints
        else constraints.copy(maxWidth = maxOf(
            measurable.maxIntrinsicWidth(constraints.maxHeight), constraints.minWidth))
        …measure(bounded), place at 0,0
    }.horizontalScroll(state)
```

It has the same doc shape as the vertical helper: the crash, the shapes that produce it
(a strip in a strip, a TextGrid in a strip, a CodeEditor in a strip), and the
constraint table. It uses a layout modifier rather than BoxWithConstraints, so
intrinsic measurement still passes through.

All three sideways regions now go through it:

1. `GrMobTextGrid`
2. `GrMobScroll`'s `flexDirection == "row"` branch
3. `GrMobCodeEditor`'s field `Box`. Its `horizontalScroll` import is dropped.

### Guards

- `mobile/verify/unbounded_scroll_test.go`, new `TestNoBareHorizontalScrollOnCompose`:
  - exactly one bare `horizontalScroll(` call in the Compose runtime;
  - the helper branches on `hasBoundedWidth` and uses `maxIntrinsicWidth(`;
  - the three callers each call the helper.
  - Mutation-checked: reverting TextGrid's call produced both "found 2" and
    "GrMobTextGrid does not scroll sideways through…".
- `collections_test.go` `TestNativeScrollHonoursTheHorizontalAxis` and
  `codeeditor_test.go` `TestNeitherNativeCodeEditorWraps`: their Kotlin pins
  now name `horizontalScrollWhenBounded(`.

### Emulator reproduction

- A throwaway `examples/zzhscroll` (deleted) put a `Horizontal()` Scroll holding
  another `Horizontal()` Scroll, a `TextGrid` and a `core.CodeEditor` in a Column.
- `android/build.sh ./examples/zzhscroll`, then `./gradlew installDebug --offline`
  twice:
  - **HEAD's Kotlin** (restored with `git show HEAD:…`, new files copied back
    right after): `FATAL EXCEPTION … Horizontally scrollable component was
    measured with an infinity maximum width constraints`.
  - **New Kotlin:** 0 FATAL. The inner strip draws at content width (pink band),
    the grid row follows, and two `adb shell input swipe`s pan the outer strip to
    the editor and "· outer end".
- The first crash to fire was the one reported, so the old-runtime run proves the
  shape crashes, not that each of the three sites did. Each site is held by the
  guard test.

## 4. What changed

| File | Change |
|---|---|
| `android/.../runtime/Renderer.kt` | `horizontalScrollWhenBounded` + doc; TextGrid and GrMobScroll's row branch use it |
| `android/.../runtime/GrMobCodeEditor.kt` | Field Box scrolls through the helper; comment; unused import dropped |
| `ios/GrMob/Runtime/Renderer.swift` | GrMobList comment: measured nested sizing, no unbounded arm |
| `core/layout.go` | `core.Scroll` doc: SwiftUI measured; the sideways shape on Compose |
| `docs/components.md` | SwiftUI sentence |
| `docs/api/core.md` | Regenerated (`go run ./internal/apidoc/gen`) |
| `mobile/verify/unbounded_scroll_test.go` | New horizontal test |
| `mobile/verify/collections_test.go`, `codeeditor_test.go` | Pins renamed to the helper |

No Go files were added, so the tracked-Go-file count sentences didn't change.

## 5. Verification

- `go test ./...` passes after regenerating the API pages. The first run failed
  on `TestGeneratedPagesAreUpToDate` because of the `core.Scroll` doc edit.
  `go vet ./core ./mobile/verify` is clean.
- iOS simulator and Android emulator, as above.
- **Not verified:**
  - SwiftUI's sizing of a horizontal ScrollView inside a horizontal ScrollView.
    Both docs now say so, rather than claiming it matches.
  - Whether GrMobList's LazyVStack stays lazy in a scrolled page on iOS.
  - The drag-on-list check on 4.6, and the `banded` GroupedList by eye on iOS.
  - 4.6 on the Android emulator by eye; Android verification is still only
    "doesn't crash".

Process notes:
- The emulator and `android/app/libs` still hold the `zzhscroll` build. Run
  `android/build.sh ./examples/tutorial` before the next Android pass.
- `ios/Frameworks` holds the tutorial build, which is what the iOS lessons need.
- A Bash call that ran `cd android` in parallel with another moved the
  session's working directory, and the next relative-path script failed. Use
  absolute paths when parallel calls `cd`.
- zsh expands a bare `=====` in `echo`; quote separators.

## Next

**Legend:** *age* is how many session docs ago the item was first raised,
counted from this doc (0 means this doc raised it; `≥ k` means it's older than
the last rebuild's window). *value* is the payoff of doing it, not the effort:
**high** means something is worked around today or a second consumer has
arrived; **medium** means it blocks one named thing or is a visible defect;
**low** means nobody has hit the gap yet. The list is sorted by age, oldest
first, and new items come last.

1. **(age ≥5 · value high) Tag a release.** v0.3.0 is the latest tag, with 30
   commits since once this session's two land. Tier A, Tier B, radio roles,
   the sweep, Tier C (Menu, SearchableSelect, Drawer), system back, browser
   back, lessons that open on Android, and both scroll axes guarded on Compose
   make a natural v0.4.0. Still open: run
   `go run github.com/rohanthewiz/grmob/cmd/grmob@latest new` on a clean
   machine without `-replace`.
2. **(age ≥5 · value high) Run lessons 6.6, 6.7, 4.15, 4.16, 4.17 and 4.18 on a
   simulator and a device.** Every lesson opens on the Android emulator, and on
   iOS `app.open(URL)` in an XCUITest reaches any lesson without a prompt.
   Unchecked on real hardware:
   - settings-row one-tap, `Screen.Footer` pinning, the iOS sheet `Dialog`,
     the stepped spinner
   - ActionSheet filler placement on Compose and SwiftUI, Timeline row
     stretch, horizontal StepIndicator scroll, radio semantics
   - CodeEditor's toolbar role on Compose, a Modal inside a ListRow's
     trailing Row (Menu)
   - SearchableSelect's Next action and keyboard dismiss on pick
   - Drawer: 100% layers filling a pinned-height ZStack on Compose and
     SwiftUI, the panel's `Height 100%` in an HStack, `AccessibilityHidden`
     confining TalkBack/VoiceOver, `core.Focus` on a Button on both natives
   - **System back:** with 4.18's drawer open, back closes it and a second
     back returns to contents; 4.16's AppBar `OnBack` shows its note. For the
     drawer, find the ☰ by its node bounds, not by text: the summary paragraph
     contains "☰".
   - **Predictive back:** the swipe previews home from contents and does not
     preview-exit from a lesson or an open drawer.
   - **Double back:** two quick `KEYCODE_BACK` presses from a lesson land on
     contents and open no lesson (the `back_cb_` sequence).
   - **Unbounded arms, by eye:** 4.6's `grouped` and `banded` GroupedLists on
     the Android emulator (`GrMobListAtContentHeight`). On iOS `grouped` is
     seen; `banded` and a drag starting on a 4.6 list are not. Anchor on a
     row title, not "Load more", which also matches the panel caption.
3. **(age ≥5 · value med) Launch the scaffolded app on a simulator and a
   device.**
4. **(age ≥5 · value med) `docs/api/core.md` is one page of ~7,800 lines.**
5. **(age ≥5 · value med) Try `grmob ios -run`** once with a simulator booted
   and once without, to read the hint (`cmd/grmob/native.go:352`).
6. **(age ≥5 · value low) Migrate hand-rolls to the new comps:** the
   tutorial's `stepper` (`examples/tutorial/widgets.go:206`), `checkRow`
   (`chapter1.go:96`, also chapters 4 and 6.4), confirm flows in
   `todoapp`/`mobileapp`, the signup step header, and hand-rolled "⋯" overflow
   buttons, except `chapter6.go:734`. Retake screenshots afterwards.
7. **(age ≥5 · value low) A looping `Transition`** so `Spinner` can drop its
   stepping and Drawer could slide. `core/animation.go` has no repeat.
8. **(age ≥5 · value low) Current-item semantics.** BottomBar's ", selected",
   StepIndicator's ", done" and ", current", `SheetAction.Checked` and Drawer
   are English fallbacks. The proper shape is an `aria-current`-style state in
   core.
9. **(age ≥5 · value low) The Android remedy drill has never run.** Candidate
   for deletion.
10. **(age ≥5 · value low) `hero.png` is a picture of pictures, stored twice.**
    Candidate for deletion.
11. **(age ≥5 · value low) A third face is still a skip.** Candidate for
    deletion.
12. **(age ≥5 · value low) No `grmob` command refreshes a scaffolded app's
    `wasm/index.html`.** `-refresh` only re-copies `android/` and `ios/`.
13. **(age ≥5 · non-goal) Tracked-Go-file count sentences** in `wasm/verify`
    are hand-edited when Go files are added. None were added this session.
14. **(age ≥5 · non-goal) Windowing.** Declined in `non_goals.md`;
    `TestWhatWindowingWouldSave` (`examples/tutorial/app_test.go:863`) is the
    profile.
15. **(age ≥5 · non-goal) Rename `docs/components.md` to `comps.md`.** The URL
    stays stable.
16. **(age ≥5 · non-goal) Rewrite `components` in the older plans.** Kept as
    history.
17. **(age ≥5 · non-goal) Trim the copied Android shell's permission
    declarations.** An app's build-time choice; see `androidPatches`.
18. **(age ≥5 · non-goal) Replace the iOS usage strings further.** They
    already name the app.
19. **(age ≥5 · non-goal) C4 `Carousel`** stays blocked until the host can
    report a scroll offset.
20. **(age 5 · value med) ARIA combobox for SearchableSelect.** A
    `RoleComboBox` with aria-expanded/aria-controls in core, plus both web
    exporters, the ARIA fixture and the runtime keyboard.
21. **(age 5 · value low) On the web, focus falls to the page after a keyboard
    pick in SearchableSelect.**
22. **(age 5 · value low) `aria-haspopup` for the Menu and DatePicker
    triggers.**
23. **(age 4 · value med) Web focus containment for Drawer.** Needs an
    `inert`-style prop in core and the web renderers.
24. **(age 4 · value low) `MaxWidth` on the natives.** Nothing in Kotlin or
    Swift reads `core.Style.MaxWidth`.
25. **(age 3 · non-goal) A parent that gains `onBack` after its descendants
    outranks them** on Android. The web ranks by document order and gets this
    case right, so the two hosts can disagree here. The built-ins nest
    correctly.
26. **(age 3 · non-goal) An AppBar placed outside the Navigator** with a
    custom `OnBack` is outranked by the Navigator's pop on Android. On the web
    it wins if it follows the Navigator in document order. Put the AppBar
    inside the route.
27. **(age 2 · value low) The reconciler sends `update-props` with
    `Changes: null`** when a node loses every prop (`reconcile/patch.go:82`).
    All three hosts now tolerate it, but a Go-side `map[string]any{}` would
    remove the trap for a hand-rolled host. Check the transcripts and goldens
    before changing it.
28. **(age 2 · value low) A typed or pasted hash while a claim is on screen
    adds a page entry above the runtime's.** The runtime then pushes a second
    entry, so back takes one extra press to unwind. Rare; revisit if a
    hashchange-routed app reports it.
29. **(age 2 · non-goal) Forward does not re-open a screen left by browser
    back.** The single entry is deliberate: the popped frame's state is gone,
    so there is nothing faithful to replay.
30. **(age 1 · non-goal) Sticky headers, placement animation and
    `OnEndReached` in a List with no viewport** on Compose.
    `GrMobListAtContentHeight` has nothing for them to act on; the page
    scrolls, not the list. Give the List a Height when they matter.
31. **(age 0 · value low) Sideways nesting on iOS is unmeasured.** A
    `Horizontal()` Scroll, TextGrid or CodeEditor inside a `Horizontal()`
    Scroll: SwiftUI should answer with the content width, as it does
    vertically, but nothing has read it. The `zzhscroll` shape from this
    session (§3) plus a frame dump is the probe.
32. **(age 0 · value low) Whether GrMobList's LazyVStack stays lazy in a
    scrolled page on iOS.** Under the page's nil height proposal it may
    materialize every row. It doesn't matter for the tutorial's six-row lists;
    it would for a long feed with no Height, which on Compose already takes the
    non-lazy arm.
33. **(age 0 · value low) Nothing holds item 30's iOS measurement.** It lives in
    a comment. `ios/verify` can't measure layout; a kept XCUITest (deep link
    via `app.open`, compare a List's frame with its last row's) could. The probe
    was deleted as throwaway.

Read by value instead: **high** 1, 2 · **medium** 3, 4, 5, 20, 23 ·
**low** 6–12, 21, 22, 24, 27, 28, 31, 32, 33 · **non-goal** 13–19, 25, 26, 29, 30.
