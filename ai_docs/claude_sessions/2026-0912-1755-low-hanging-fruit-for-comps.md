# Low-hanging fruit for comps

**Session:** https://claude.ai/code/session_013PPKnYoYbsVv3vW7oAXA6p
**Date:** 2026-09-12 17:55
**Branch:** master

## 1. The ask

"Are there any low-hanging fruit in terms of typical components that can be
added to grmob? Create a plan doc for this in `ai_docs/plans/`."

## 2. What was surveyed

- `comps/` as it stands after the rename: 38 widgets (`Screen` through
  `Compass`), read from the type list and `comps/doc.go`.
- `core`'s primitives that a composition-only widget can lean on: `Modal`
  (`Visible`/`OnDismiss`/`Backdrop`/`ModalContent`), `Switch`, `Checkbox`,
  `Slider`, `NumericInput`, `TextArea`, `Select`, `ZStack`+`StackAlign`,
  `Rotate`, `Transition`, `StickyHeader`, `OnEndReached`, and the role set in
  `core/role.go` (`RoleDialog`, `RoleCheckbox`, `RoleStatus`, `RoleAlert`,
  `RoleNavigation`, `RoleToolbar`, `RoleListBox`/`RoleOption`; no radio or
  spinbutton role).
- The prior plan's constraint statement (`components-datatable-compass-map.md`):
  a new node type costs four renderers; a pure `comps` widget costs none.
- Hand-roll evidence in `examples/` (modal, switch, numeric input, slider and
  select call sites across `mobileapp`, `todoapp` and tutorial chapters 1–8).

Two names in the first draft were wrong and were corrected against the source:
the danger variant is `comps.VariantError` (not `Danger`), and
`core.Transition` is a one-shot duration+easing, not a loop, so a `Spinner`
needs `hooks.UseInterval` and is a hook-holding widget.

## 3. What was written

`ai_docs/plans/comps-low-hanging-fruit.md`, structured like the earlier
component plans: constraints, tier ordering, per-widget sketch with the API
shape, the accessibility role it wears, and the one decision each needs.

| Tier | Widgets | Cost |
|---|---|---|
| A | `Dialog`, `SwitchRow`/`CheckboxRow`, `Stepper`, `BottomBar` (+ additive `Screen.Footer`), `Spinner`, `Rating` | hours each, pure composition |
| B | `ActionSheet`, `RadioGroup`, `Snackbar`, `StepIndicator`, `Timeline` | a day each, one decision first |
| C | `Menu`, `Drawer`, `Autocomplete`, `Carousel` | shape to settle; carousel blocked on scroll offset |

Explicitly not low-hanging (needs a renderer): native pickers, gestures
(drag-to-dismiss, swipe-to-delete, pull-to-refresh), anchored popovers, a
host-drawn toast with an action, anything needing scroll offset.

Suggested order: `Dialog` → settings rows → `BottomBar`+`Screen.Footer` →
`Spinner` → `Stepper`/`Rating` → `ActionSheet` → the rest of B.

## 4. Decisions recorded in the plan

- `Dialog` with no `Cancel` is an alert; with neither is a sheet. One struct.
- Settings rows render the control non-interactive and make the row the only
  handler, so a tap flips state exactly once.
- `BottomBar` wears `RoleNavigation` when `Selected >= 0`, else `RoleToolbar`.
- `RadioGroup` ships on `RoleListBox`/`RoleOption` + `AccessibilitySelected`;
  adding `RoleRadio` to `core/role.go` is a follow-up, not a blocker.
- `ActionSheet`'s one unknown: whether `core.Modal` can pin content to the
  bottom edge on all four renderers. Read `ModalNode` and each modal branch
  before starting.

## Next

1. **(carried, partly done · value high) Tag a release.** v0.3.0 is tagged.
   Still open: run `go run github.com/rohanthewiz/grmob/cmd/grmob@latest new`
   on a clean machine without `-replace`.
2. **(new · value high) Implement Tier A of `comps-low-hanging-fruit.md`.**
   Start with `Dialog` and the settings rows; one tutorial lesson for the
   bundle.
3. **(new · value med) Answer the `ActionSheet` placement question** (can
   `core.Modal` content sit at the bottom edge?) before any Tier B work.
4. **(new · value low) `RoleRadio`/`RoleRadioGroup` in `core/role.go`** once
   `RadioGroup` exists; three files, not four renderers.
5. **(new · value low) A looping `Transition`** so `Spinner` can drop its
   interval hook. Four-renderer change; deferred.
6. **(carried · value low) `docs/components.md` still carries the old name.**
7. **(carried · value low) `ai_docs/plans/*.md` still say `components`** in
   older plans; the new plan uses `comps` throughout.
8. **(carried · value med) Launch the scaffolded app on a simulator and a
   device.**
9. **(carried · value med) `grmob ios -run`** still prints a `simctl` line
   instead of doing it.
10. **(carried · value low) The copied shells' comments still cite `grmob://`.**
11. **(carried · value low) The copied Android shell declares every demo
    permission.**
12. **(carried · value low) `insertLineBefore` duplicates `uniqueLine`'s search.**
13. **(carried · value low) The iOS usage strings are placeholders.**
14. **(carried · value med) `docs/api/core.md` is one 7,700-line page.**
15. **(carried · value med) Nothing checks links from `docs/api/` out into the
    narrative pages.**
16. **(carried · value low) `aria/spec` in a user-facing reference.**
17. **(carried · value low) The skill's pointer URL is dead until merge.**
18. **(carried · value low) The Android remedy drill has never run.**
19. **(carried · value low) `hero.png` is a picture of pictures and its parts
    are held twice.**
20. **(carried · value low) A third face is still a skip.**
21. **(carried · value low) Windowing as a proposal.**

Closed since the previous doc: "Let upgraders know about the rename" landed
as commit 46d410a (an Upgrading section).
