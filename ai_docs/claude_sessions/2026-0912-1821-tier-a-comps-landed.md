# Tier A comps landed

**Session:** https://claude.ai/code/session_013PPKnYoYbsVv3vW7oAXA6p
**Date:** 2026-09-12 18:21
**Branch:** master

## 1. The ask

"Implement Tier A: A1 and A2" from `ai_docs/plans/comps-low-hanging-fruit.md`.
Mid-turn the user added: "Once completely done, commit and push then continue
with the remainder of Tier A then do /sess-wrap."

## 2. Commits

| Commit | What |
|---|---|
| 3055259 | A1 `comps.Dialog`, A2 `comps.SwitchRow` / `comps.CheckboxRow`, tutorial lesson 6.6 |
| 129b4e6 | A3 `Stepper`, A4 `BottomBar` + `Screen.Footer`, A5 `Spinner` (+ `hooks.UseIntervalWhile`), A6 `Rating`, tutorial lesson 4.15 |

Both pushed. `go test ./...` green at each commit.

## 3. What landed, and the decisions that differ from the plan

### A1 `Dialog` (`comps/dialog.go`)
- Modal > `comps.Card` (title as heading, `AccessibilityLabel(Title)` on the
  card; the Modal chassis already writes `role="dialog"`) > footer Row.
- Shape from which actions have a `Label`: confirm (both), alert (Confirm
  only), sheet (neither, `Body` slot).
- Cancel leading + ghost, Confirm trailing + filled with its `Variant`,
  `JustifyEnd`. No order knob.
- `Cancel.OnTap == nil` falls back to `OnDismiss`. Confirm never closes the
  dialog. `OnDismiss == nil` omits the Modal's `onDismiss` prop (inert scrim).
- Verified in the renderers: Compose `Dialog.onDismissRequest` reports back
  gesture and outside tap; SwiftUI presents a Modal as a **sheet** (swipe-down
  reports dismiss), not a centred card. Comment and docs say so.

### A2 `SwitchRow` / `CheckboxRow` (`comps/settings_row.go`)
- **Plan correction:** the plan's `core.Disabled(true)` on the control was
  rejected. Disabled is the platform disabled state: greyed control on every
  target, "dimmed" to screen readers.
- Actual design: control stays live; row `OnTap` and control `onToggle` both go
  through `toggleSetter` — `if v != current { OnToggle(v) }`; row sets `!On`,
  control sets the platform's value.
- Why it works: Compose's Switch and SwiftUI's Toggle consume their own press
  (control only fires). On the web the click bubbles to the row div first,
  Go re-renders, and callback IDs are positional per pass so the following
  `change` event hits the fresh closure whose `current` already equals `v`.
  Same-pass double dispatch is idempotent anyway. `OnToggle` is documented as
  a setter.
- Control carries `AccessibilityLabel(Title)` + `AccessibilityHint(Subtitle)`;
  row takes no role. `Disabled` disables both, handler stays registered.
- **Not device-verified:** the Compose/SwiftUI "control consumes the press"
  claims come from reading the renderers, not from a device run.

### A3 `Stepper` (`comps/stepper.go`)
- Outlined buttons, not ghost (a ghost "−" reads as text).
- Bounds only when `Max > Min`; zero value unbounded. Clamp in widget,
  `OnChange` only on real change, button at a bound `Disabled`.
- `RoleGroup` + `Label` + `AccessibilityValue` (numbers only when bounded, text
  always). Buttons named "Decrease"/"Increase", overridable; `Format` hook.

### A4 `BottomBar` + `Screen.Footer` (`comps/bottom_bar.go`, `comps/screen.go`)
- `Selected >= 0` → `RoleNavigation`, negative → `RoleToolbar` (zero value
  selects the first item, like SegmentedControl/Tabs).
- Cells are `Column` + `FlexGrow(1)` + `RoleButton` (not `JustifyAround`), so
  targets tile the bar. Current item: bold, primary on-light tone, name gains
  ", selected" (ListRow precedent; no aria-current in core). Icon hidden.
- `Screen.Footer`: SafeArea gets a second child; with `Scroll` the Scroll
  takes `FlexGrow(1)`, without it the column grows. Nil Footer leaves the tree
  byte-identical. Verified SafeArea is a flex column honouring grow on
  Compose (`GrMobColumn`), SwiftUI (flex stack) and web.

### A5 `Spinner` (`comps/spinner.go`) + `hooks.UseIntervalWhile`
- Findings that shaped it: `core.Transition` is one-shot; `Style.Rotate` is
  applied un-interpolated on both natives (`Modifier.rotate`,
  `rotationEffect`); `Style.Animation` is web-only. So the spin is stepped
  from Go: 30° every 80 ms.
- `hooks.UseInterval` calls `ctx.RequestRender()` after every tick for the
  life of the process, so a spinner on it would re-render the app ~12×/s
  forever. Added `hooks.UseIntervalWhile(ctx, active, fn, interval)`
  (`hooks/interval.go`, shared `useInterval` body, `paused` on the record):
  a paused tick skips both `fn` and the render. `UseInterval` behaviour is
  unchanged.
- Third hook-holding widget (after Accordion, DatePicker): render
  unconditionally, toggle `Hidden` (Display none + paused ticks).
- Ring + orbiting Primary dot (uniform ring looks static rotated; core has no
  per-side border colour). `RoleStatus` + `Label` (default "Loading"), ring
  `AccessibilityHidden`. Sizes: Spacing MD/LG/XL.

### A6 `Rating` (`comps/rating.go`)
- `Value float64`, rounded in v1. Interactive glyphs are `RoleButton` Boxes
  named "3 of 5"; tapping the current value is a no-op. Read-only / nil
  `OnChange` registers nothing and hides glyphs; group states "4 of 5".
- Filled colour `WarningOnLightColor()` (raw Warning is ~2:1 on light).

## 4. Tutorial, docs and the counters that moved

- Lesson 6.6 "Dialog and settings rows" (`examples/tutorial/chapter6.go`):
  note list, confirm-first switch, attachments checkbox, Dialog. Tests drive
  row taps and the web double dispatch through the real Manager.
- Lesson 4.15 "Small controls: Stepper, Rating, Spinner & BottomBar"
  (`chapter4.go`), appended at the chapter end so `grmob://lesson/4.12` does
  not move. Test file's `nodeStyle` gained `Display`.
- `docs/components.md`: sections for Dialog, SwitchRow & CheckboxRow, Stepper,
  Rating, BottomBar, Spinner; Screen diagram/table/paragraph for Footer.
  `docs/api/{comps,hooks,index}.md` regenerated (`go run ./internal/apidoc/gen`).
- Lesson count 51 → 53 in README (alt text + prose), `wasm/index.html`,
  `docs/tutorial-interactive.md`, `internal/shotclaims` (+ test comment),
  `screenshot_test.go`, `core/deeplink.go`, `app_test.go` comments.
  `docs/images/tutorial-contents.png` re-taken twice with
  `wasm/shots/shoot.sh tutorial-contents` (Chrome + Node 22 local) and viewed.
- `wasm/verify` tracked-Go-file count sentences 469 → 473 → 481 (five lines in
  `repowalks_test.go` / `timings_test.go`). The check counts untracked files
  too, so it fires before a commit that adds Go files.
- Plan doc: status now "Tier A landed", with the deviations listed; its
  "appears in `examples/components.go`" DoD line corrected (that file is a
  small Portuguese input demo, not a gallery).

## Next

1. **(carried, partly done · value high) Tag a release.** v0.3.0 is tagged.
   Still open: run `go run github.com/rohanthewiz/grmob/cmd/grmob@latest new`
   on a clean machine without `-replace`. Tier A is a natural v0.4.0.
2. **(new · value high) Run lessons 6.6 and 4.15 on a simulator and a device.**
   The settings-row one-tap claim for Compose and SwiftUI, `Screen.Footer`
   pinning, the iOS sheet presentation of `Dialog`, and the stepped spinner
   were verified by reading renderers and by Go tests, not on hardware.
3. **(carried · value med) Answer the `ActionSheet` placement question** (can
   `core.Modal` content sit at the bottom edge?) before any Tier B work. Note
   iOS already presents every Modal as a sheet with medium/large detents.
4. **(new · value med) Tier B of the plan:** `ActionSheet`, `RadioGroup`,
   `Snackbar` (can use `hooks.UseTimeout`), `StepIndicator`, `Timeline`, one
   widget per commit.
5. **(new · value low) Migrate hand-rolls to the new comps:** the tutorial's
   `stepper` helper (`widgets.go`, three chapter-1 call sites), chapter 6.4's
   and chapter 1's `checkRow`, and any confirm flows in `todoapp`/`mobileapp`.
6. **(carried · value low) A looping `Transition`** so `Spinner` can drop its
   stepping. Four-renderer change; less urgent now that `UseIntervalWhile`
   makes a hidden spinner free.
7. **(carried · value low) `RoleRadio`/`RoleRadioGroup` in `core/role.go`**
   once `RadioGroup` exists.
8. **(new · value low) `BottomBar` current-item semantics.** ", selected" in
   the name is an English fallback; an `aria-current`-style state in core
   would be the proper shape.
9. **(new · value low) The tracked-Go-file count sentences** in `wasm/verify`
   must be hand-edited on every commit that adds Go files. Working as designed
   per its header; noted so it is not mistaken for a regression.
10. **(carried · value low) `docs/components.md` still carries the old name.**
11. **(carried · value low) `ai_docs/plans/*.md` still say `components`** in
    older plans.
12. **(carried · value med) Launch the scaffolded app on a simulator and a
    device.**
13. **(carried · value med) `grmob ios -run`** still prints a `simctl` line
    instead of doing it.
14. **(carried · value low) The copied shells' comments still cite `grmob://`.**
15. **(carried · value low) The copied Android shell declares every demo
    permission.**
16. **(carried · value low) `insertLineBefore` duplicates `uniqueLine`'s search.**
17. **(carried · value low) The iOS usage strings are placeholders.**
18. **(carried · value med) `docs/api/core.md` is one 7,700-line page.**
19. **(carried · value med) Nothing checks links from `docs/api/` out into the
    narrative pages.**
20. **(carried · value low) `aria/spec` in a user-facing reference.**
21. **(carried · value low) The skill's pointer URL is dead until merge.**
22. **(carried · value low) The Android remedy drill has never run.**
23. **(carried · value low) `hero.png` is a picture of pictures and its parts
    are held twice.**
24. **(carried · value low) A third face is still a skip.**
25. **(carried · value low) Windowing as a proposal.**
26. **(non-goal) Drawer (plan C2)** stays skipped until asked for; `BottomBar`
    covers the same navigation need with no overlay.

Closed since the previous doc: "Implement Tier A" (all six widgets, commits
3055259 and 129b4e6).
