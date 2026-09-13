# Tier B comps, radio roles, and a toolbar

**Session:** https://claude.ai/code/session_013PPKnYoYbsVv3vW7oAXA6p
**Date:** 2026-09-12 20:16
**Branch:** master

## 1. The asks

1. "Implement Tier B, one widget per commit" (`ai_docs/plans/comps-low-hanging-fruit.md`).
2. "Fix the scrollable ChipStrip" (a bug found while landing B4).
3. Mid-turn: "When complete next do the Radio roles in core and the RichTextEditor toolbar role".
4. `/sess-wrap`.

## 2. Commits

| Commit | What |
|---|---|
| b79a8c4 | B1 `comps.ActionSheet`, lesson 6.7 "Action sheets" (count 53 → 54) |
| f09ac54 | B3 `comps.Snackbar` + `hooks.UseTimeoutWhile`; 6.7 becomes "Action sheets & snackbars" |
| f3310c5 | B2 `comps.RadioGroup`, lesson 4.16 (count 54 → 55); composite guard rewritten |
| bab3c23 | B4 `comps.StepIndicator`; 4.16 becomes "Radio groups & step indicators" |
| f9b2154 | B5 `comps.Timeline`; 4.16 becomes "Radio groups, steps & timelines"; plan marks Tier B landed |
| 1397c8d | Horizontal Scrolls keep their height on the web host pages (ChipStrip fix); StepIndicator's wrapper Row removed |
| e7a508a | `core.RoleRadioGroup` / `core.RoleRadio`; RadioGroup moved onto them |
| b2f6bb1 | RichTextEditor's formatting strip is a named toolbar (`RichToolbar.Label`) |

All pushed. `go test ./...` green before every commit. Each widget was also
checked in Chrome through the shots harness (see §5).

## 3. Tier B: what landed and the decisions that differ from the plan

### B1 `ActionSheet` (`comps/action_sheet.go`)
- **Placement question answered, no ZStack.** Web: Modal chassis is a centred
  flex column (htmlout `modalChassis`, runtime `styleFromGrMob`). Compose:
  `Dialog > Column(fillMaxWidth) > ColumnChildren`, where FlexGrow becomes
  `weight`. SwiftUI: `.sheet` with medium/large detents, `PlainChildren`
  ignores grow.
- Modal holds two children: a growing, invisible filler Box
  (`FlexGrow(1)`, `AlignSelf(stretch)`, hidden, `OnClick(OnDismiss)`), then a
  `Card` (`Width 100%`, `Margin(0)`, label = Title). The card painting a
  background stops Compose's default white surface; the filler painting none
  keeps iOS taking the card's colour as the sheet surface.
- The filler reports taps above the panel: on the web the runtime's
  `attachModalDismiss` only fires for taps on the overlay element itself, and
  on Compose the filler is inside the dialog window.
- Actions are ghost full-width `Button`s, **not** listbox options (an option
  would read "Delete, not selected"). Picking an action runs `OnTap` then
  `OnDismiss`. Cancel is outlined, below a `Separator`.
- Chrome probe: filler 419–952, card 952–1195 = overlay bottom.

### B3 `Snackbar` (`comps/snackbar.go`) + `hooks.UseTimeoutWhile`
- `hooks.UseTimeout` arms once per slot for the life of the app, so a new hook
  `UseTimeoutWhile(ctx, active, fn, delay, deps...)` (`hooks/interval.go`):
  arms on the rising edge of `active`, cancels when it falls, re-arms when
  deps change (`reflect.DeepEqual`), fires once per activation. A `gen`
  counter drops a fire that raced a cancel; `OnClose` is registered outside the
  record's mutex. Tests run with `-race -count=3`.
- Snackbar is controlled (`Visible`, `OnAction`, `OnTimeout`), keyed on
  `Message`, `Duration` 0 = `SnackbarDuration` (4s), negative = never. Inverse
  look (`TextPrimary` fill, `Background` ink); `Variant` fills and uses
  `Variant.Ink`. `RoleStatus`, `RoleAlert` for `VariantError`, no label.
  Hidden = `Display none`. Placement is the caller's: `Screen.Footer` or a
  bottom-aligned ZStack layer.

### B2 `RadioGroup` (`comps/radio_group.go`)
- Vertical only; rows are ListRows with a drawn ring (Column with border and
  dot, hidden), so the row is the only handler and no web double dispatch.
  `OnChange` only for a different, enabled option. No row tint.
- Shipped first on `RoleListBox` + `ListRow.Selectable`, then moved to the
  radio pair (§4).
- **Composite guard rewrite.** It tripped
  `TestNoWidgetDeclaresACompositeContainerRole`, whose premise was already false:
  `BottomBar` has declared `RoleToolbar` since Tier A via a variable the scan
  missed. Now `TestOnlyClosedWidgetsDeclareACompositeContainerRole`: scans every
  code reference to a container role constant; allowed only in files on
  `closedComposites`, which must declare no struct field of `core.View` /
  `[]core.View`; stale entries fail. Listed: `bottom_bar.go`, `radio_group.go`,
  `rich_text_editor.go`. `keynav_test.mjs` note and the RichTextEditor comment
  updated. A historical session doc that `sed` had rewritten was restored.

### B4 `StepIndicator` (`comps/step_indicator.go`)
- Horizontal `core.Scroll`, not a collapse (Go has no width). Done steps
  Success disc with ✓, current Primary disc, upcoming ring; rules Success after
  done steps. Only done steps tappable (`RoleButton`). Strip is
  `RoleNavigation` with `OnTap`, else `RoleGroup`; named "Step 2 of 4: X",
  `Label` prefix ("Checkout, step …"). `Current` clamped.

### B5 `Timeline` (`comps/timeline.go`)
- Not on `ListRow` (its row padding would break the line). Each event is a
  `Row(AlignItems stretch, Padding 0, RoleListItem)` with a rail column: fixed
  top segment (height centres the dot on the first text line, from
  `LineHeight` or 1.2 × `FontSize`), 12pt dot (`Variant.Color`), bottom
  segment `FlexGrow(1)`. Space below an event is `PaddingBottom(MD)` inside the
  body so the line runs through it; first top / last bottom keep size and paint
  nothing. List is `RoleList`, rail hidden.

## 4. After Tier B

### Horizontal Scroll collapse (1397c8d)
- Root cause: `wasm/index.html`, `wasm/shots/index.html` and
  `cmd/grmob/templates/wasm/index.html.tmpl` give every Scroll
  `flex: 1 1 0; min-height: 0`. In a column that is a zero-height basis: probe
  measured the lesson 4.8 ChipStrip at 0px with 22px chips, and StepIndicator's
  first version the same.
- Fix in all three pages, selected on the inline `flex-direction: row` the
  runtime writes (it writes `aria-orientation` only for composite roles, so no
  other hook exists):
  `[data-node-type="Scroll"][style*="flex-direction: row"] { flex: 0 0 auto }`
  and, inside a row parent, `flex: 1 1 0; min-width: 0`.
- Probe after: chip strip 62px (409px content in 296px), step strip 25px,
  screen Scroll still 776px. `docs/platforms/wasm.md` updated.

### Radio roles in core (e7a508a)
- `RoleRadioGroup`/`RoleRadio` in `core/role.go`, `Roles()` (27),
  `KeyboardComposites()` (4), `CompositeMemberRole`. Docs counts updated.
- State stays `Style.AccessibilitySelected`; both web exporters write
  `aria-checked` for a radio (runtime `ariaSelected` now returns a triple;
  htmlout `ariaSelected` gained the arm). `core/style.go` table updated.
- Runtime: `COMPOSITE_MEMBERS` gains `radiogroup: "radio"`; `activeMemberIndex`
  reads `aria-checked` too; `moveCompositeFocus` always selects inside a
  radiogroup; `ARIA_ORIENTATIONS` gains `radiogroup: ""` and `ariaOrientation`
  tests membership instead of truthiness (htmlout map got the same row).
- Natives: Compose `"radiogroup" -> selectableGroup()`,
  `"radio" -> role = Role.RadioButton` (import added); SwiftUI empty arm with
  its reason.
- ARIA fixture: `aria/fetch.sh` re-downloaded the spec (gitignored),
  `aria-checked` added to `InScopeAttributes`, `radiogroup`/`radio` removed from
  `NearMisses`, regenerated. ARIA 1.2 also allows `aria-checked` on `option`, so
  `TestTheStateGuardsMatchARIAsScoping` checks the selection spellings as a
  family: exactly one supported spelling written, never an unsupported one.
- Radiogroup refusal row deleted; tappable census, composite member pairs,
  static-export test and runtime pins updated. Three new `keynav_test.mjs`
  tests (aria-checked only, stop on checked radio, arrow checks with no flag).
- RadioGroup: container `RoleRadioGroup`, rows pass `RoleRadio` +
  `AccessibilitySelected(SelectedWhen)` through `ListRow.Style` (not
  `Selectable`). Chrome on 4.16: rows `aria-checked`, group
  `aria-orientation="vertical"`, ArrowDown moves stop and check to Express and
  the Go caption follows.

### RichTextEditor toolbar (b2f6bb1)
- Strip carries `RoleToolbar` and `AccessibilityLabel(orDefault(bar.Label,
  "Formatting"))`; new `RichToolbar.Label`. Doc and `docs/components.md` no
  longer tell callers to wrap the editor in a toolbar. Read-only editor: all
  buttons disabled, so no members and no stop.
- Chrome on 4.14: one toolbar, 12 buttons, one tab stop, ArrowRight moves
  focus and stop from Bold to Italic.

## 5. Tutorial, docs, counters

- Lesson 6.7 "Action sheets & snackbars" (`chapter6.go`): a note with Share,
  Duplicate, Delete; Delete raises a Snackbar with Undo. Tests cover action,
  Cancel, filler and scrim dismiss, and Undo.
- Lesson 4.16 "Radio groups, steps & timelines" (`chapter4.go`): a four-step
  checkout whose Shipping step is the RadioGroup, then a growing order
  Timeline. Tests cover rows, disabled option, done-step navigation and
  timeline growth.
- Lesson count 53 → 55 in README, `wasm/index.html`,
  `docs/tutorial-interactive.md`, `internal/shotclaims` (+ test),
  `screenshot_test.go`, `core/deeplink.go`, `app_test.go`;
  `docs/images/tutorial-contents.png` retaken twice and viewed.
- `wasm/verify` tracked-Go-file sentences 481 → 491 across the commits. The
  check counts untracked files, so unrelated work-in-progress files have to be
  moved aside when committing one widget (done once for B4 with the Timeline
  files; the API docs were regenerated without them).
- `docs/components.md`: ActionSheet, Snackbar, RadioGroup, StepIndicator,
  Timeline sections; RichTextEditor toolbar paragraph. `docs/api/*`
  regenerated each commit.
- Plan doc: Tier B landed, per-widget decisions, the Scroll fix and radio roles.
- Verification harness used: build `wasm/shots/host` into the scratchpad,
  copy `index.html`, `grmob-runtime.js`, `wasm_exec.js`, then
  `node wasm/shots/shot.mjs --dir <www> --out - <probe.js>` for DOM probes or
  `--out file.png` for shots.

## Next

1. **(carried, partly done · value high) Tag a release.** v0.3.0 is tagged.
   Tier A + Tier B + radio roles are a natural v0.4.0. Still open: run
   `go run github.com/rohanthewiz/grmob/cmd/grmob@latest new` on a clean
   machine without `-replace`.
2. **(carried, widened · value high) Run lessons 6.6, 6.7, 4.15 and 4.16 on a
   simulator and a device.** Unverified on hardware: settings-row one-tap,
   `Screen.Footer` pinning, iOS sheet `Dialog`, stepped spinner, ActionSheet
   filler placement on Compose and SwiftUI, Timeline row stretch, horizontal
   StepIndicator scroll, and the new radio semantics (Compose
   `selectableGroup`/`RadioButton`).
3. **(new · value med) Apps scaffolded before 1397c8d have the old host CSS.**
   A horizontal Scroll collapses in them; an Upgrading note (or a
   `grmob` command that refreshes `index.html`) would say to copy the two rules.
4. **(new · value med) Tier C of the plan:** C1 `Menu` (ActionSheet with a
   trigger slot), C3 `SearchableSelect` (focus behaviour first), C4 `Carousel`
   stays blocked on a scroll-offset signal.
5. **(new · value low) Arrowing onto a disabled radio.** The runtime focuses a
   disabled radio and calls its OnTap (Go ignores it), so focus sits on a radio
   that cannot be checked. ARIA's pattern skips disabled radios.
6. **(new · value low) `docs/components.md` ChipStrip paragraph is stale:** it
   says a toolbar role gets no roving focus, which the runtime now supplies.
   Also `comps/list_row.go` still says "nine of core's twenty roles".
7. **(new · value low) CodeEditor's toolbar** has the same shape as the rich
   text strip and could take the toolbar role under the closed rule.
8. **(carried · value low) Migrate hand-rolls to the new comps:** the
   tutorial's `stepper` helper, chapter 6.4's and chapter 1's `checkRow`,
   confirm flows in `todoapp`/`mobileapp`, and the signup flow's step header.
9. **(carried · value low) A looping `Transition`** so `Spinner` can drop its
   stepping.
10. **(carried · value low) `BottomBar` current-item semantics.** ", selected"
    (also StepIndicator's ", done"/", current") is an English fallback; an
    `aria-current`-style state in core would be the proper shape.
11. **(carried · value low) Tracked-Go-file count sentences** in `wasm/verify`
    are hand-edited on every commit that adds Go files, and count untracked
    files. Working as designed.
12. **(carried · value low) `docs/components.md` still carries the old name.**
13. **(carried · value low) `ai_docs/plans/*.md` still say `components`** in
    older plans.
14. **(carried · value med) Launch the scaffolded app on a simulator and a
    device.**
15. **(carried · value med) `grmob ios -run`** still prints a `simctl` line
    instead of doing it.
16. **(carried · value low) The copied shells' comments still cite `grmob://`.**
17. **(carried · value low) The copied Android shell declares every demo
    permission.**
18. **(carried · value low) `insertLineBefore` duplicates `uniqueLine`'s search.**
19. **(carried · value low) The iOS usage strings are placeholders.**
20. **(carried · value med) `docs/api/core.md` is one 7,700-line page.**
21. **(carried · value med) Nothing checks links from `docs/api/` out into the
    narrative pages.**
22. **(carried · value low) `aria/spec` in a user-facing reference.**
23. **(carried · value low) The skill's pointer URL is dead until merge.**
24. **(carried · value low) The Android remedy drill has never run.**
25. **(carried · value low) `hero.png` is a picture of pictures and its parts
    are held twice.**
26. **(carried · value low) A third face is still a skip.**
27. **(carried · value low) Windowing as a proposal.**
28. **(non-goal) Drawer (plan C2)** stays skipped until asked for; `BottomBar`
    covers the same navigation need with no overlay.

Closed since the previous doc: "Answer the ActionSheet placement question",
"Tier B of the plan" (all five widgets), and "`RoleRadio`/`RoleRadioGroup` in
`core/role.go`".
