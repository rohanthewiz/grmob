# Lessons stop crashing on Android

**Session:** https://claude.ai/code/session_013PPKnYoYbsVv3vW7oAXA6p
**Date:** 2026-09-12 23:17
**Branch:** master

## 1. The asks

1. `/sl` loaded the "Browser back, predictive back, and a stale double back"
   session doc.
2. "Let's do item 30": every tutorial lesson crashed on Android with
   `IllegalStateException: Vertically scrollable component was measured with
   an infinity maximum height constraints`.
3. `/sw lessons-stop-crashing-on-android`: save this doc, commit all, push.

## 2. The cause: two scroll containers under a scrolled page

A lesson screen is `comps.Screen{Scroll: true}`, so everything in its column
is measured with an infinite maximum height. Compose's scroll containers check
for exactly that and throw. The DOM has no such rule (an `overflow: auto` box
of auto height is as tall as its content), so the Go tree was legal and drew
in the browser.

The first trace read, outermost first:

```
Root → InsetsPadding → BoxWithConstraints       the lesson Screen's Scroll
  → ScrollingLayoutNode (outer verticalScroll)
    → Column (fill, padding) → Column → …
      → ScrollingLayoutNode (inner)             ← throws
```

1. **Code blocks.** `codeBlock` (`examples/tutorial/widgets.go`) is a
   read-only `comps.CodeEditor` with no Height, and `GrMobCodeEditor` laid out
   as `Row(s.boxModifier(extra).verticalScroll(vertical))`. Nearly every lesson
   has one; the contents screen is a `core.List` with none, which is why it
   rendered. This predates the back-button work: code blocks became editors in
   3bbc378 ("Two editors").
2. **Height-less Lists.** With the editor fixed, a deep-link walk of all 57
   lessons still crashed on **4.3** (`outlineDemo`'s `core.List`) and **4.6**
   (the `grouped` and `banded` GroupedLists). The trace there was
   `LazyListKt$rememberLazyListMeasurePolicy` → `checkScrollableContainerConstraints`:
   LazyColumn is a scroll container with the same check.

## 3. The fix

### `verticalScrollWhenBounded` (Renderer.kt)

```kotlin
internal fun Modifier.verticalScrollWhenBounded(state: ScrollState): Modifier =
    layout { measurable, constraints ->
        val bounded = if (constraints.hasBoundedHeight) constraints
        else constraints.copy(maxHeight = maxOf(
            measurable.maxIntrinsicHeight(constraints.maxWidth), constraints.minHeight))
        …measure(bounded), place at 0,0
    }.verticalScroll(state)
```

- Bounded: identical to `verticalScroll`.
- Unbounded: the viewport is capped at the content's intrinsic height, so
  there is nothing to scroll and nothing to throw. The scroll node answers
  intrinsics by asking its content, and intrinsics skip the constraint check.
- The incoming minHeight is kept as a floor, so ColumnChildren's
  `heightIn(min = viewport)` for a FlexGrow child still holds.
- A layout modifier, not BoxWithConstraints: an editor may be measured
  intrinsically (a stretched Row asks for `IntrinsicSize.Max`), and a
  SubcomposeLayout throws on that.
- Used by `GrMobCodeEditor` and by `GrMobScroll`. The latter's own doc claimed a
  Scroll nested in a Scroll "keeps wrapping"; it would have thrown the same way.

### `GrMobList` gets an unbounded arm

Laziness needs a viewport, and LazyColumn cannot answer intrinsics, so the
intrinsic trick is not available. `GrMobList` now asks:

```
BoxWithConstraints(node modifiers, propagateMinConstraints = true)
  bounded    → LazyColumn, as before
  unbounded  → GrMobListAtContentHeight: every row, keyed, in a plain Column
```

- `propagateMinConstraints = true` hands the Box's min/max to the LazyColumn,
  so a bounded List measures exactly as the bare LazyColumn did: a fixed
  Height is still both bounds, and a wrapping List still wraps.
- The alignment `when` was hoisted into `val rowAlignment`, shared by both
  arms. Its header text is unchanged, so `TestKotlinListAlignmentCoversEveryAlignItems`
  still reads it inside `fun GrMobList(`. The sticky and stretch pins also
  still sit in that declaration.
- `listState` and `EndReachedReporter` are composed before the branch, so a
  flip between arms keeps the scroll position and the composition shape.
- In the unbounded arm, sticky headers don't pin, placement doesn't animate,
  and `OnEndReached` doesn't fire. The page is what scrolls; an endless feed
  needs a Height (4.8 has one). Written up on the helper.

### Guards

`mobile/verify/unbounded_scroll_test.go`, both mutation-checked:

- `TestNoBareVerticalScrollOnCompose`: exactly one `verticalScroll(` call in
  the Compose runtime, inside the helper. The helper branches on
  `hasBoundedHeight` and uses `maxIntrinsicHeight`, and both the editor and
  `GrMobScroll` call it.
- `TestComposeListHasAnArmForAnUnboundedHeight`: GrMobList has
  BoxWithConstraints, `propagateMinConstraints = true`, the branch before the
  LazyColumn, and a non-lazy `GrMobListAtContentHeight`. There is exactly one
  lazy container in the runtime, matched as `Lazy…\s*[({]` so the
  trailing-lambda `LazyColumn { }` counts. That form was missed by the first
  regex and caught by a mutant.

`codeeditor_test.go`'s gutter pin now names `verticalScrollWhenBounded(vertical)`.

## 4. What changed

| File | Change |
|---|---|
| `android/.../runtime/Renderer.kt` | `verticalScrollWhenBounded` (doc with the constraint table), `GrMobScroll` uses it, `GrMobList` split into bounded/unbounded arms, `GrMobListAtContentHeight`, `ScrollState` import |
| `android/.../runtime/GrMobCodeEditor.kt` | Editor Row scrolls through the helper; comment on the crash; unused `verticalScroll` import dropped |
| `mobile/verify/unbounded_scroll_test.go` | New: the two tests above |
| `mobile/verify/codeeditor_test.go` | Pin renamed to the helper |
| `core/layout.go` | `core.Scroll` doc: "Inside another vertical scroll" |
| `docs/components.md` | `Screen.Scroll`: a content-sized region isn't the nested-scroll case; the Compose guard |
| `docs/api/core.md` | Regenerated |
| `wasm/verify/repowalks_test.go`, `timings_test.go` | Tracked-Go-file count 498 → 499 (the new test file) |

## 5. Verification

- `go test ./...` passes; `go vet` is clean on `core` and `mobile/verify`.
- **Emulator** (Medium_Phone_API_36.1): `android/build.sh ./examples/tutorial`,
  then `ANDROID_HOME=~/Library/Android/sdk ./gradlew installDebug --offline`.
  Gradle fails with "SDK location not found" without `ANDROID_HOME`.
  - Walk 1, editor fix only: 57 deep links (`grmob://lesson/<id>`), crashes
    on 4.3 and 4.6 only.
  - Walk 2, both fixes: **57 of 57 open, 0 FATAL**.
  - Screens seen: 2.3's code block draws and pans sideways; 4.3's indented
    outline List (Matthew, Mark, Letters, Romans) draws at content height.
  - System back: back on 4.18 lands on contents, a second back leaves the
    app, no FATAL. Checked twice.
- **Not verified:**
  - 4.6's GroupedLists on screen. The 1400px swipes overshot them to the key
    points; the walk only proves they don't crash.
  - Back closing an open drawer. The ☰ matcher also matched the lesson
    summary's text ("opened by a ☰"), so the tap hit the paragraph and the
    drawer never opened.
  - iOS: nothing was run there this session.

Process notes:
- The lesson IDs came from a throwaway `zz_ids_scratch_test.go`, deleted in
  the same command. `go test .` hides a passing test's stdout; it needs `-v`.
- A walk's final `grep CRASH` exits 1 when there are no crashes, so the task
  reported "failed" on a clean walk.
- One python re-indent inserted the unindented block it had just computed, and
  one regex replace missed on over-escaped backslashes. Both were caught by
  reading the output.

## Next

**Legend:** *age* is how many session docs ago the item was first raised,
counted from this doc (0 means this doc raised it; `≥ k` means it's older than
the last rebuild's window). *value* is the payoff of doing it, not the effort:
**high** means something is worked around today or a second consumer has
arrived; **medium** means it blocks one named thing or is a visible defect;
**low** means nobody has hit the gap yet. The list is sorted by age, oldest
first, and new items come last.

1. **(age ≥5 · value high) Tag a release.** v0.3.0 is the latest tag, with 27
   commits since once this session's two land. Tier A, Tier B, radio roles,
   the sweep, Tier C (Menu, SearchableSelect, Drawer), system back, browser
   back, and lessons that open on Android make a natural v0.4.0. Still open:
   run `go run github.com/rohanthewiz/grmob/cmd/grmob@latest new` on a clean
   machine without `-replace`.
2. **(age ≥5 · value high) Run lessons 6.6, 6.7, 4.15, 4.16, 4.17 and 4.18 on a
   simulator and a device.** Unblocked on Android: every lesson opens on the
   emulator now. Unchecked on real hardware:
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
     back returns to contents; 4.16's AppBar `OnBack` shows its note. Back on
     a lesson → contents and the root frame leaving the app are checked on the
     emulator. For the drawer, find the ☰ by its node bounds, not by text: the
     summary paragraph contains "☰".
   - **Predictive back:** the swipe previews home from contents and does not
     preview-exit from a lesson or an open drawer.
   - **Double back:** two quick `KEYCODE_BACK` presses from a lesson land on
     contents and open no lesson (the `back_cb_` sequence).
   - **Unbounded arms, by eye:** 4.6's `grouped` and `banded` GroupedLists
     (Collapse bands, LoadMore footer) drawn by `GrMobListAtContentHeight`.
3. **(age ≥5 · value med) Launch the scaffolded app on a simulator and a
   device.**
4. **(age ≥5 · value med) `docs/api/core.md` is one page of ~7,800 lines.**
5. **(age 5 · value med) Try `grmob ios -run`** once with a simulator booted
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
12. **(age 5 · value low) No `grmob` command refreshes a scaffolded app's
    `wasm/index.html`.** `-refresh` only re-copies `android/` and `ios/`.
    The Compose runtime changes here do reach scaffolded apps through
    `-refresh`.
13. **(age ≥5 · non-goal) Tracked-Go-file count sentences** in `wasm/verify`
    are hand-edited when Go files are added. This session added one (498 →
    499, five sentences).
14. **(age ≥5 · non-goal) Windowing.** Declined in `non_goals.md`;
    `TestWhatWindowingWouldSave` (`examples/tutorial/app_test.go:863`) is the
    profile.
15. **(age 5 · non-goal) Rename `docs/components.md` to `comps.md`.** The URL
    stays stable.
16. **(age 5 · non-goal) Rewrite `components` in the older plans.** Kept as
    history.
17. **(age 5 · non-goal) Trim the copied Android shell's permission
    declarations.** An app's build-time choice; see `androidPatches`.
18. **(age 5 · non-goal) Replace the iOS usage strings further.** They
    already name the app.
19. **(age 5 · non-goal) C4 `Carousel`** stays blocked until the host can
    report a scroll offset.
20. **(age 4 · value med) ARIA combobox for SearchableSelect.** A
    `RoleComboBox` with aria-expanded/aria-controls in core, plus both web
    exporters, the ARIA fixture and the runtime keyboard.
21. **(age 4 · value low) On the web, focus falls to the page after a keyboard
    pick in SearchableSelect.**
22. **(age 4 · value low) `aria-haspopup` for the Menu and DatePicker
    triggers.**
23. **(age 3 · value med) Web focus containment for Drawer.** Needs an
    `inert`-style prop in core and the web renderers.
24. **(age 3 · value low) `MaxWidth` on the natives.** Nothing in Kotlin or
    Swift reads `core.Style.MaxWidth`.
25. **(age 2 · non-goal) A parent that gains `onBack` after its descendants
    outranks them** on Android. The web ranks by document order and gets this
    case right, so the two hosts can disagree here. The built-ins nest
    correctly.
26. **(age 2 · non-goal) An AppBar placed outside the Navigator** with a
    custom `OnBack` is outranked by the Navigator's pop on Android. On the web
    it wins if it follows the Navigator in document order. Put the AppBar
    inside the route.
27. **(age 1 · value low) The reconciler sends `update-props` with
    `Changes: null`** when a node loses every prop (`reconcile/patch.go:82`).
    All three hosts now tolerate it, but a Go-side `map[string]any{}` would
    remove the trap for a hand-rolled host. Check the transcripts and goldens
    before changing it.
28. **(age 1 · value low) A typed or pasted hash while a claim is on screen
    adds a page entry above the runtime's.** The runtime then pushes a second
    entry, so back takes one extra press to unwind. Rare; revisit if a
    hashchange-routed app reports it.
29. **(age 1 · non-goal) Forward does not re-open a screen left by browser
    back.** The single entry is deliberate: the popped frame's state is gone,
    so there is nothing faithful to replay.
30. **(age 0 · value med) The same lessons on iOS.** Nothing was run on a
    simulator this session. SwiftUI's sizing of a `ScrollView` inside a
    `ScrollView` is unmeasured: `GrMobList` is a `ScrollView { LazyVStack }`,
    and 4.3 and 4.6 put one in a scrolled lesson. If it collapses or
    mis-sizes there, iOS needs the analogue of the unbounded arm. The
    `core.Scroll` doc deliberately makes no SwiftUI claim until this is read.
31. **(age 0 · value low) Horizontal scrolls under an unbounded width.**
    `horizontalScroll` has the same check on its axis. Today's three uses
    (`Renderer.kt:450`, `GrMobScroll`'s row branch, the editor's field `Box`)
    all sit under a bounded width, but a horizontal Scroll inside a horizontal
    Scroll, or an editor inside a sideways strip, would throw. A
    `horizontalScrollWhenBounded` plus a refusal arm would close it.
32. **(age 0 · non-goal) Sticky headers, placement animation and
    `OnEndReached` in a List with no viewport.** `GrMobListAtContentHeight`
    has nothing for them to act on; the page scrolls, not the list. Give the
    List a Height when they matter.

Read by value instead: **high** 1, 2 · **medium** 3, 4, 5, 20, 23, 30 ·
**low** 6–12, 21, 22, 24, 27, 28, 31 · **non-goal** 13–19, 25, 26, 29, 32.
