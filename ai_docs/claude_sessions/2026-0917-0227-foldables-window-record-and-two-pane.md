# Foldables: a window record, UseWindow, and a hinge-aware TwoPane

**Session:** cfb0150b-f554-41d5-8c5c-b31eddcb620a
**Date:** 2026-09-17 02:27 (follows "zero-basis-floor-area-fades-scatter-squares-android-look")
**Branch:** master (2f1d441 → this commit)

## 1. The ask

"support a foldable form-factor"

## 2. Design

Foldable support modelled on the existing host-event records (lifecycle,
heading): one host event, one process-wide record in core, one hook, and a
layout component on top.

```
Android  WindowManager (metrics + FoldingFeature) ─┐
iOS      GeometryReader size (no fold)            ├─"window" host event─▶ core.ReceiveWindow ─▶ OnWindow subs ─▶ hooks.UseWindow ─▶ RequestRender
Browser  innerWidth/Height + viewport.segments    ─┘                        (dedupe by ==)
         + navigator.devicePosture
```

Payload: `{width, height, fold?: {state, orientation, separating, occluding, x, y, width, height}}`,
lengths in layout units (dp / pt / CSS px), window coordinates (origin at the
window's top-left, under the system bars).

Decisions:

- **A record, not a layout node.** What a fold forces is usually a tree change
  (list beside detail, video above controls), which Go components already do.
  Cost: a pane not at the window origin must pass `Origin`.
- **`Window` stays comparable** (`HasFold bool` + `Fold` value rather than
  `*Fold`) so the record dedupes with `==`; a stale Fold behind `HasFold=false`
  is normalized away.
- **Validation split:** a NaN/negative/infinite size drops the whole report; an
  unknown fold state/orientation drops only the fold and keeps the size
  (forward compatibility, same stance as `ReceiveLifecycle`).
- **Size classes** are Material's (width 600/840, height 480/900). Unreported
  window reads compact.
- **Posture derived, not reported:** tabletop = half-opened + horizontal hinge,
  book = half-opened + vertical.
- **Browser reports a fold only with two viewport segments**; posture alone
  has no position, so none is invented.
- **`WindowRect`**, not `Rect` — `core.Rect` is the canvas shape constructor.

## 3. What was built

- `core/window.go` (+ tests): `Window`, `Fold`, `WindowRect`, `SizeClass`,
  `FoldState`, `FoldOrientation`, `Posture`, `CurrentWindow`, `OnWindow`,
  `ReceiveWindow`; consumed in `core/host_events.go`.
- `hooks/window.go` (+ test): `UseWindow`, same shape as `UseLifecycle`.
- `comps/two_pane.go` (+ tests): `TwoPane{First, Second, Ratio, Gap, SplitAt,
  Compact, Origin, IgnoreHorizontalFold, Style}`. Rules in order: separating
  vertical fold → Row pinned at the hinge; separating horizontal fold → Column;
  width ≥ SplitAt (medium) → ratio Row; else stack / first only / second only.
  An occluding hinge becomes an `AccessibilityHidden` spacer. Holds one hook
  (stable-position rule, like Snackbar).
- Android `app/AppWindow.kt` (collects `windowLayoutInfo` under
  `repeatOnLifecycle(STARTED)`, attached per Activity in `MainActivity`);
  `androidx.window:window:1.4.0` in `build.gradle`.
- iOS `App/AppWindow.swift` (`AppWindowReader` as GrMobRoot's background,
  safe area ignored); `GrMobApp.swift` frames the root to fill first.
- `wasm/grmob-runtime.js` `windowMetrics` module (`foldFrom`, `measure`,
  `report`), reported on resize, posture change and once in `waitForWasm`;
  `wasm/verify/window_test.mjs` (8 cases).
- `mobile/verify/window_test.go`: spellings agree across the three shells.
- Tutorial lesson **4.21 "Foldables and window size"**: live readout and a
  TwoPane list–detail demo (+ test). Lesson count 59 → 60 in README,
  `wasm/index.html`, `docs/tutorial-interactive.md`, `core/deeplink.go`,
  shotclaims; `tutorial-contents.png` re-shot.
- apidoc topics: `window.go` under core "Device services", `two_pane.go`
  under comps "Screens & structure"; docs/api regenerated.
- Tracked Go file count 547 → 554 in `wasm/verify` prose.
- ROADMAP entry.

## 4. Found on the emulator

- **A real bug the Go tests passed:** the first TwoPane used
  `core.Column(..., core.Horizontal())`. `Horizontal()` is the scroll strip's
  prop; the natives pick a stack's axis from the node **type**, so the panes
  stacked on Android. Fixed to `core.Row`; tests now assert `n.Type`.
- A 7.6" Fold-in AVD (`GrMob_Foldable`, API 36.1) on port 5556, headless.
  Reports reached Go: 674×841 medium, hinge x 336.76 dp, 0.38 dp wide,
  `type=HINGE` (so occluding and separating). TwoPane split at the hinge.
- The emulator's window extension reports hinge **state one posture change
  late** (half-opened device state → `FLAT`; `CLOSED` → `HALF_OPENED`), and
  fully open (180°) reports no feature. Seen in a temporary `Log.d` of the raw
  `WindowLayoutInfo` (removed). The shell passes WindowManager's word through.
- `adb emu fold` on the headless emulator set CLOSED but did not switch to the
  outer display.

## 5. Checks

`go test ./...`, `go vet`, `gofmt`, `wasm/verify/run.sh` (incl. Chrome),
`android/verify/run.sh` (sources compile + JVM harness), iOS app layer
`swiftc -typecheck` against the iPhoneOS SDK, `./gradlew assembleDebug` — all
pass.

## Next

**Legend:** *age* is how many session docs ago the item was first raised,
counted from this doc (0 = raised here). *value*: **high** = worked around
today or a second consumer has arrived; **medium** = blocks one named thing
or is a visible defect; **low** = nobody has hit the gap yet. Sorted by age,
oldest first; new items last.

**Closed this session:** none of the carried items. Item 53 was partly
touched (4.21 seen at 674 dp on the foldable emulator) but no sweep of the
comps widgets at tablet width was done.

1. **(age ≥25 · value high) Lessons on hardware, and checks still open.**
   - Real devices for everything (deferred by the user).
   - **Screen readers:**
     - radio and StepIndicator done-step semantics
     - combobox active option
     - "pop-up" on Menu/DatePicker triggers
     - aria-current/selected on current items
     - Calendar's grid on the web (needs a person with VoiceOver)
     - CodeEditor toolbar role on Compose
     - SearchableSelect natives reading field + list + status
     - AccessibilityHidden confining TalkBack/VoiceOver behind a Drawer
     - VoiceOver on the calendar cell now that it is one element
   - **iOS:**
     - sheet `Dialog` and ActionSheet filler
     - Drawer slide by eye and under Reduce Motion
     - RTL for Drawer and CodeEditor
     - editing in the sideways code editor
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
2. **(age ≥25 · non-goal) Rename `docs/components.md` to `comps.md`.**
3. **(age ≥25 · non-goal) Rewrite `components` in the older plans.**
4. **(age ≥25 · non-goal) Trim the copied Android shell's permissions.**
5. **(age ≥25 · non-goal) Replace the iOS usage strings further.**
6. **(age ≥25 · non-goal) C4 `Carousel`** until the host reports a scroll
   offset.
7. **(age 24 · non-goal) A parent that gains `onBack` after its descendants
   outranks them** on Android.
8. **(age 24 · non-goal) An AppBar outside the Navigator** with a custom
   `OnBack` is outranked by the Navigator's pop on Android.
9. **(age 23 · non-goal) Forward does not re-open a screen left by browser
   back.**
10. **(age 22 · non-goal) Sticky headers, placement animation and
    `OnEndReached` in a List with no viewport** on Compose.
11. **(age 20 · value low) The cost of a paused TimelineView per node** in
    SwiftUI's box chain. Not profiled.
12. **(age 20 · non-goal) MaxWidth with a growing sibling on the natives.**
13. **(age 20 · non-goal) The typed-hash fold's `history.length` fallback.**
14. **(age 20 · non-goal) A page's own `pushState` while a claim is on
    screen.**
15. **(age 20 · non-goal) A Drawer's shut panel is composed on the natives.**
16. **(age 18 · non-goal) A List with no Height in a scrolled page is not
    lazy** on iOS or Android.
17. **(age 12 · value low) Compose's today `stateDescription` not heard**
    under TalkBack on the emulator.
18. **(age 12 · value medium) F-keys on iOS unverified; key delivery to a
    simulator is unreliable.**
19. **(age 12 · value low) Page-global chords on iOS are verified once.**
20. **(age 11 · non-goal) GameController cannot take a key.**
21. **(age 11 · non-goal) Commits already on a remote carrying an unformatted
    file** are noted, not blocked, except at the tip.
22. **(age 9 · value low) The iOS chord gate is unheard.**
23. **(age 9 · value low) Merging a labelled node is unheard under
    TalkBack.**
24. **(age 9 · value medium) `.claude/settings.json` cannot be edited from a
    session.** Worth an upstream report.
25. **(age 8 · value low) Alarm sound and haptics are unheard.**
    - The lesson passes no `Sound`.
    - Ringing replaces core's single audio player and doesn't resume it
      (documented).
    - Haptic pulses were not felt on a device.
    - The Notify banner's default sound is unheard.
26. **(age 8 · non-goal) `DigitalClock` digits can shift width by a pixel.**
    There's no font family in `core.Style`.
27. **(age 8 · non-goal) `AnalogClock{Smooth}` spins back once if the tick at
    00:00:00 is skipped.**
28. **(age 8 · value low) Canvas still omits** text inside the drawing,
    clipping and per-shape hit-testing. Gradient fills, gradient strokes and
    the even-odd rule are in.
29. **(age 7 · value low) A chart's hidden data table** was not built.
    - The one-sentence summary is all a screen reader gets.
    - A BarChart with more than 8 categories only gives the low and high.
    - It needs a screen-reader-only primitive core doesn't have: an API
      decision.
30. **(age 7 · non-goal) A 180° `Gauge` leaves its bottom half empty.**
31. **(age 7 · value low) Chart summaries are English**, like other widgets'
    spoken strings. `AccessibilityLabel` overrides them.
32. **(age 6 · value low) A Notify alarm is a banner, not a ringing screen.**
    - There's no full-screen intent, no looping sound, and no Snooze action.
    - iOS AlarmKit and Android `AlarmClock` / full-screen intents are the
      route to a real system alarm. Large, and only a device will tell.
33. **(age 6 · value low) `mobile.SetTimeZone` runs once at startup.** A fix
    needs core to hold the current location: an API decision.
34. **(age 6 · value low) The web's scheduled notification and its sweep are
    unseen in a real browser.** Chrome's permission prompt needs a person to
    grant it once.
35. **(age 5 · value low · user's decision) Compose Rows don't shrink
    children in proportion.** Documented in `Renderer.kt` (pinMainAxis) and
    pinned by `internal/pinfixture`. A fix replaces Compose's Row measure
    policy for every Row on Android (a flex solver like iOS's); not attempted,
    put to the user.
36. **(age 4 · non-goal) `MaxLines` on the natives applies to Text only.**
    Documented on `core.MaxLines`. No comps widget sets it elsewhere, and
    SwiftUI's inherited `lineLimit` would cap each Text rather than clamp the
    container as CSS does.
37. **(age 4 · non-goal) A stacked chart counts NaN as 0 and stacks negatives
    through the band below.** Documented under "Stacked".
38. **(age 4 · value low) The double-post claim (old item 46) is
    unreproduced.**
39. **(age 4 · value low) `testRelaunchSweepsWhatTheDeadProcessScheduled`
    needs notifications already granted.**
40. **(age 3 · value low) `DefaultDarkChartColors` has no bundled consumer.**
    No dark theme ships. It was checked by the skill's published validation,
    not re-run here.
41. **(age 3 · value low) `TutorialChartsUITests` failed once, reason not
    captured.** It passed on all three runs this session.
42. **(age 3 · non-goal) `core.LinearGradient` (a CSS string helper) is
    unused and sits beside the canvas gradient names.** Kept per CLAUDE.md's
    no-removal rule.
43. **(age 2 · value medium) Last session's alarm changes are unseen on
    Android:** the kept banners after a force stop, and the exact-alarm
    re-check. (Stroke gradients and bar values were seen this session.)
44. **(age 2 · value low) A zero basis is honoured on iOS only when the main
    extent is definite.** An ideal-size query keeps content bases, and a
    Column child with the `.infinity` height verdict keeps its measured base.
    Both are deliberate (see `GrMobFlexZeroBasis`).
45. **(age 2 · value medium · blocked) The iOS Image floor runs high for an
    image narrower in proportion than its box.**
    - A natural-size store and a floor matching Chrome were built and
      reverted this session.
    - On the simulator a floor below the declared width made the row overlap
      the text under the image: a px-width box does not narrow when offered
      less.
    - Unblocking it needs a px-width box that can shrink on iOS (CSS lets an
      `<img>` flex item shrink to its content suggestion), which touches
      `grMobDimension` for every node.
46. **(age 2 · value low) Kept banners are unseen on a device.** The relaunch
    UI test (`TutorialAlarmNotifyUITests`) was not re-run. An Android alarm
    Doze delayed past its time is no longer cancelled on return, so its
    banner may post after `OnRing` already reported it (documented in the
    hook).
47. **(age 2 · non-goal) Renaming a `NotifyGroup` strands what the old name
    scheduled** until the OS fires it. Documented; groups are meant to be
    constants.
48. **(age 1 · value medium) An iOS Image with no background shows a black
    letterbox.** A "fit" image narrower or shorter in proportion than its box
    paints black in the rest. Seen with `AsyncImage` (committed) for PNG and
    JPEG; a `BackgroundColor` hides it. Not traced to a modifier.
49. **(age 1 · value low) `barValueRoom` is still an estimate.** It assumes a
    220 px plot and 0.6 em per rune, so a wide screen puts some labels inside
    that would fit outside, and a narrow plot can still cut one with an
    ellipsis.
50. **(age 1 · value low) Square scatter dots are unseen on any device.** No
    lesson has four series; they rest on a Go test and a SwiftUI render check
    of the 0.01-unit square cap. Compose and Chrome are unchecked.
51. **(age 1 · value low) `testStatTilesShareTheRow` asserts weakly.** It
    solves for a padding in 0–24 pt, which a content-biased split could also
    satisfy; the screenshot was the real check.
52. **(age 1 · value low) Sparkline's `Area` is still a flat tint**, unlike
    LineChart's new fade. Left alone: a sparkline's area runs to its bottom
    edge, not zero, and it has no axis to fade toward.
53. **(age 1 · value low) Comps on Android were seen at phone size only.**
    Tablet widths and landscape are unswept.
54. **(age 0 · value medium) Foldable behaviour unverified on real
    hardware.** Only the emulator was used, and its extension lags a posture
    behind; tabletop/book postures as a real device reports them are unseen.
55. **(age 0 · value medium) Safe-area insets are not a record.** A TwoPane
    under the status bar in tabletop posture cannot find its own `Origin.Y`;
    the caller must guess. Needs an API decision (a `core.SafeInsets` record
    over the same host event, likely).
56. **(age 0 · value low) Lesson 4.21's TwoPane demo sets no `Origin`,** so
    its split lands a panel-inset (~30 dp) right of the hinge. Harmless (the
    crease falls in the first pane's empty space) but not exact.
57. **(age 0 · value low) iOS `AppWindowReader` is type-checked only.** Not
    run on a simulator, iPad Split View or Stage Manager.
58. **(age 0 · value low) The browser's segments/posture path is unseen in a
    real browser.** Chrome DevTools' foldable emulation would show it; only the
    node harness covers `foldFrom`.
59. **(age 0 · value low) Folding shut onto the outer display is unseen.** The
    headless emulator did not switch displays; Activity recreation on a real
    display switch is assumed from the docs.
60. **(age 0 · non-goal) More than one fold.** Android reports the first
    FoldingFeature and core holds one; no shipping device has two.
61. **(age 0 · non-goal) Static HTML export of a TwoPane** always renders the
    compact arrangement (no window at export time).
62. **(age 0 · value low) Housekeeping:** the `GrMob_Foldable` AVD is left
    installed for re-testing, and `android/app/libs/grmob.aar` (gitignored) is
    currently built from `./examples/tutorial` rather than the default demo.
