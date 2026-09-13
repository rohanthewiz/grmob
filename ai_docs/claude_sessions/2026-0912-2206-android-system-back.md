# Android system back

**Session:** https://claude.ai/code/session_013PPKnYoYbsVv3vW7oAXA6p
**Date:** 2026-09-12 22:06
**Branch:** master

## 1. The asks

1. `/sl` loaded the Next-list rebuild session doc.
2. "Let's do item 23, Android system back."
3. `/sw android-system-back`: save this doc, commit all, and push.

## 2. What was wrong

The Android shell never handled the back button. No `.kt` file under
`android/` had a `BackHandler`, `OnBackPressedCallback` or `onKeyDown`, and core
had no back hook. Back left the app from every screen: from a pushed Navigator
frame, and from an open Drawer. Only a Modal closed on back, and only because
Compose's `Dialog` reports back through `onDismissRequest` itself.

## 3. Design: a node prop, not a host event

Lifecycle and deep links arrive as host events. Back can't. Android's
`OnBackPressedDispatcher` checks its callbacks' enabled flags on the UI thread
before the press is delivered, and finishes the Activity if none is enabled. A
host event reaches Go after that decision. A shell built on one would need a
second, Go→host "enabled" signal kept in step with app state.

A prop is that signal already. Presence in the tree is the enabled flag, and
the tree diff keeps the shell's copy current:

- `core.OnBack(fn)` is `On("Back", fn)`, which is the `onBack` callback prop on
  the wire.
- Compose's `RenderNode` (the funnel every node passes through) calls
  `BackHandler { runtime.click(onBack) }` when the prop is present. It is not
  `BackHandler(enabled = …)` on every node, which would put the whole tree in
  the dispatcher.
- **Innermost wins.** The dispatcher runs the most recently registered
  callback. Handlers composed in one pass register parent before child, and one
  composed later (a drawer that just opened) outranks those already there.

```
Navigator route root  onBack = Pop            outermost, runs last
  AppBar row          onBack = AppBar's back action
  Drawer panel layer  onBack = OnDismiss      while Open; runs first
```

Nothing handles it at the root frame, so back falls through and leaves the app.

## 4. What changed

| File | Change |
|---|---|
| `core/behavioral_props.go` | `OnBack`, with a doc covering the design, the ordering and each host |
| `core/navigation.go` | `takeTop` also returns `canPop`, read under the same lock as the top frame. `Navigator` calls `withSystemBackPop` when poppable |
| `comps/drawer.go` | The panel layer carries `core.OnBack(OnDismiss)` while `Open && OnDismiss != nil`. The type doc and the `OnDismiss` field doc were updated |
| `comps/app_bar.go` | New `backAction(ctx)`, resolved once and handed to both the arrow and `core.OnBack` on the bar row. `leading` now takes the func |
| `android/.../Renderer.kt` | `import androidx.activity.compose.BackHandler`, and the `BackHandler` in `RenderNode` |
| `wasm/grmob-runtime.js` | An `onBack` no-op branch before the generic `on*` branch, on both the create and update paths |
| `mobile/verify/back_test.go` | New source pin: core's `On("Back"`, RenderNode's read, `BackHandler` and import, and the two JS branches |
| Tests | `core/navigation_test.go` (3), `comps/drawer_test.go` (1), `comps/app_bar_test.go` (1) |
| Docs | `docs/concepts/navigation.md` (new "Android's system back" section and table), `events.md` ("System back"), the `components.md` Drawer table, regenerated `docs/api/*` |
| Tutorial | The key points in chapter 4.18 (Drawer) and chapter 6 (Navigator) no longer say back is unwired |
| `wasm/verify/{repowalks,timings}_test.go` | Tracked-Go-file count 497 → 498 (the non-goal arm, working as designed) |

### Decisions worth remembering

- **Navigator copies the route's node** instead of writing into it: Type, Key,
  Style and Children are shared, and only the Props map is fresh. A
  `core.Cached` route returns the same node every pass, so a write would pin the
  callback ID from whichever pass wrote it. Pinned by
  `TestNavigatorDoesNotWriteBackIntoACachedNode`.
- **A route's own `onBack` on its root node is kept.** Navigator doesn't add
  its pop, so a form can confirm before leaving.
- **The handler registers on the host context**, not the frame. The frame's
  scope is exactly what the pop discards.
- **AppBar's handler goes on the row**, because `comps.Button` takes style props
  only. The row exists in both the separator and `HideSeparator` shapes. With a
  `Leading` slot or `HideBack` there is no arrow and no handler, so back falls
  to Navigator's pop.
- **Drawer's handler is only on while open**, even though a `Display none`
  panel isn't composed on Android. Otherwise every pass would register a
  callback for an invisible drawer.
- **iOS** has no system back, and the SwiftUI renderer uses no
  `UINavigationController`. **The web**: the browser's back button moves the
  page's history, which the page owns. **htmlout** exports callbacks from an
  explicit table, so it already ignored `onBack`.

## 5. Verification

- `go test ./...` passes, after the 497 → 498 count bump.
- `ANDROID_HOME=~/Library/Android/sdk android/gradlew compileDebugKotlin
  --offline`: BUILD SUCCESSFUL. The first attempt without `ANDROID_HOME` failed
  with "SDK location not found" and compiled nothing.
- It has not been run on an emulator or device (see Next 2).

A process slip: the scratchpad replace helper treated a probe's
`@@NEW_PLACEHOLDER` marker as replacement text and wrote a stray line into
`core/navigation.go`. It was removed in the next edit, and a grep for `@@`
confirmed it was gone.

## Next

**Legend:** *age* is how many session docs ago the item was first raised,
counted from this doc (0 means this doc raised it; `≥ k` means it's older than
the last rebuild's window). *value* is the payoff of doing it, not the effort:
**high** means something is worked around today or a second consumer has
arrived; **medium** means it blocks one named thing or is a visible defect;
**low** means nobody has hit the gap yet. The list is sorted by age, oldest
first.

1. **(age ≥3 · value high) Tag a release.** v0.3.0 is the latest tag and there
   are 23 commits since, 24 with this one. Tier A, Tier B, the radio roles, the
   sweep, Tier C (Menu, SearchableSelect, Drawer) and system back would make a
   natural v0.4.0. Still open: run
   `go run github.com/rohanthewiz/grmob/cmd/grmob@latest new` on a clean
   machine without `-replace`.
2. **(age ≥3 · value high) Run lessons 6.6, 6.7, 4.15, 4.16, 4.17 and 4.18 on a
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
   - **System back (new):** back on a lesson returns to contents; back at
     contents leaves the app; with 4.18's drawer open, back closes the drawer
     and a second back returns to contents; 4.16's AppBar demo with `OnBack`
     set shows its note instead of popping. Check both the 3-button nav bar and
     the gesture nav.
3. **(age ≥3 · value med) Launch the scaffolded app on a simulator and a
   device.**
4. **(age ≥3 · value med) `docs/api/core.md` is one page of ~7,800 lines.**
5. **(age 3 · value med) Try `grmob ios -run`** once with a simulator booted
   and once with none booted, to read the hint. The flag exists
   (`cmd/grmob/native.go:352`) but has never been run.
6. **(age ≥3 · value low) Migrate hand-rolls to the new comps:**
   - the tutorial's `stepper` helper (`examples/tutorial/widgets.go:206`)
   - `checkRow` (`chapter1.go:96`, also used by chapters 4 and 6.4)
   - confirm flows in `todoapp`/`mobileapp`
   - the signup flow's step header
   - hand-rolled "⋯" overflow buttons

   Screenshots need retaking afterwards. Leave the "⋯" at `chapter6.go:734`
   alone: that is the ActionSheet lesson opening its own sheet.
7. **(age ≥3 · value low) A looping `Transition`** so `Spinner` can drop its
   stepping. It would also allow a sliding Drawer. `core/animation.go` has no
   repeat today.
8. **(age ≥3 · value low) Current-item semantics.** These are all English
   fallbacks: BottomBar's ", selected", StepIndicator's ", done" and
   ", current", and the ", selected" in `SheetAction.Checked` and Drawer. The
   proper shape is an `aria-current`-style state in core.
9. **(age ≥3 · value low) The Android remedy drill has never run.** Candidate
   for deletion.
10. **(age ≥3 · value low) `hero.png` is a picture of pictures, and its parts
    are stored twice.** Candidate for deletion.
11. **(age ≥3 · value low) A third face is still a skip.** Candidate for
    deletion.
12. **(age 3 · value low) No `grmob` command refreshes a scaffolded app's
    `wasm/index.html`.** `-refresh` only re-copies `android/` and `ios/`. The
    next change to the host page's CSS will need another README note unless
    such a command exists. (This session's `Renderer.kt` change does reach
    scaffolded apps through `-refresh`.)
13. **(age ≥3 · non-goal) Tracked-Go-file count sentences** in `wasm/verify`
    are hand-edited on every commit that adds Go files. This is working as
    designed; this session bumped them to 498.
14. **(age ≥3 · non-goal) Windowing.** Declined in `non_goals.md`.
    `TestWhatWindowingWouldSave` (`examples/tutorial/app_test.go:863`) is the
    profile to re-run.
15. **(age 3 · non-goal) Rename `docs/components.md` to `comps.md`.** The page
    keeps its name so its URL stays stable.
16. **(age 3 · non-goal) Rewrite `components` in the older plans.** Only the
    two most recent plans were updated; the rest stay as history.
17. **(age 3 · non-goal) Trim the copied Android shell's permission
    declarations.** Removing one is a build-time choice the app should make;
    see `androidPatches`.
18. **(age 3 · non-goal) Replace the iOS usage strings further.** They already
    name the app.
19. **(age 3 · non-goal) C4 `Carousel`** stays blocked until the host can
    report a scroll offset (`OnScroll` or a paged-scroll node). Core has
    neither.
20. **(age 2 · value med) ARIA combobox for SearchableSelect.** Needs:
    - a `RoleComboBox` with aria-expanded and aria-controls (optionally
      aria-activedescendant) in core
    - support in both web exporters, the ARIA fixture and the runtime keyboard

    This is renderer work.
21. **(age 2 · value low) On the web, focus falls to the page after a keyboard
    pick in SearchableSelect.** Returning focus on the web only would need a
    host distinction that core doesn't expose.
22. **(age 2 · value low) `aria-haspopup` for the Menu and DatePicker
    triggers.** Core has no way to say a button opens a dialog or menu.
23. **(age 1 · value med) Web focus containment for Drawer.** Tabbing past the
    panel reaches the screen behind it, which is aria-hidden. Needs an
    `inert`-style prop in core and the web renderers. Neither the runtime nor
    core has one today.
24. **(age 1 · value low) `MaxWidth` on the natives.** `core.Style.MaxWidth`
    exists, but nothing in Kotlin or Swift reads it, so Drawer can't cap a
    percentage width on a tablet.
25. **(age 0 · value med) Browser back → Navigator.** On the web, `onBack` is
    deliberately skipped, and the browser's back button moves history, which
    the page owns. The tutorial bridges this itself with its "route" host
    event (`examples/tutorial/deeplink.go`), but a plain app's Navigator
    doesn't pop on browser back. A generic shape would push a history entry
    per Push and pop on `popstate`. Consumers: every web-previewed app with a
    Navigator.
26. **(age 0 · value low) Predictive back.** `android:enableOnBackInvokedCallback`
    is not set, so Android 14+ shows no back-preview animation. `BackHandler`
    (activity-compose 1.9.0) already works with it. Flip the manifest flag and
    check that a pushed frame, an open drawer and the root all behave on a
    device.
27. **(age 0 · value low) Stale callback ID on a double back.** Two back
    presses dispatched before the first pass's patches land can run a re-used
    positional ID. This is the framework-wide caveat in
    `docs/concepts/events.md` ("identity-keyed IDs are the planned fix"), but
    back is where a quick repeat is most likely. Watch for it in the device run
    (Next 2).
28. **(age 0 · non-goal) A parent that gains `onBack` after its descendants
    outranks them.** This is Compose's dispatcher order, and the ordering rule
    is written into `core.OnBack`'s doc instead of being fixed. The built-in
    users (Navigator, AppBar, Drawer) all nest correctly.
29. **(age 0 · non-goal) An AppBar placed outside the Navigator** (as chrome
    above it) with a custom `OnBack` is outranked by the Navigator's plain pop
    on system back. Put the AppBar inside the route, which is where
    `comps.Screen` layouts already put it.

Read by value instead: **high** 1, 2 · **medium** 3, 4, 5, 20, 23, 25 ·
**low** 6–12, 21, 22, 24, 26, 27 · **non-goal** 13–19, 28, 29.
