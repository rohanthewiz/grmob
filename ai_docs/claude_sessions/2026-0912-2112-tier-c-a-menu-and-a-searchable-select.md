# Tier C: a Menu and a SearchableSelect

**Session:** https://claude.ai/code/session_013PPKnYoYbsVv3vW7oAXA6p
**Date:** 2026-09-12 21:12
**Branch:** master

## 1. The asks

1. `/sl` — load the Next-list sweep session doc.
2. "Let's implement Tier C" (`ai_docs/plans/comps-low-hanging-fruit.md`).
3. Mid-turn: "please commit C1 also" (C3 was then committed the same way).
4. `/sw`.

## 2. Commits

| Commit | What |
|---|---|
| 217de2d | C1 `comps.Menu` + `SheetAction.Checked`; lesson 4.17 "Menus" (count 55 → 56) |
| ca817a4 | C3 `comps.SearchableSelect` + `SearchField.FocusRef`; 4.17 becomes "Menus & searchable selects" |

Before each commit: `go test ./...`, `wasm/verify/run.sh` and
`node --test wasm/verify/keynav_test.mjs` (72 tests) green, plus a Chrome probe
through the shots harness.

## 3. Scope decided

Tier C had four items. C2 `Drawer` is a recorded non-goal, and C4 `Carousel`
needs a host scroll-offset signal (renderer work, excluded by the plan). Only
C1 and C3 were built.

## 4. C1 `Menu` (`comps/menu.go`)

- `Row(Padding 0, Gap 0, AlignItems center) > [trigger Button, ActionSheet]`.
  The Row exists only because Render returns one node.
- **Trigger is a `comps.Button` template**, not a `core.View` slot: a widget
  cannot attach a tap to a View it did not build. The template's `OnTap` is
  overwritten with `OnOpen`; Button already swaps nil for a no-op.
- **`Open` is controlled** (`Open`, `OnOpen`, `OnDismiss`), unlike DatePicker's
  hook-held flag. The common case is a "⋯" per row, and positional hook slots
  in a loop drift when the row count changes. One caller state holding the
  open row's ID serves every menu. It also keeps Menu conditional-safe.
- **No expanded state on the trigger**: `core/style.go` names "a control that
  opens a dialog" as the aria-expanded near miss (aria-haspopup is not in
  core). Checked before writing.
- Items are `[]SheetAction`. **`SheetAction.Checked`** (additive to B1): label
  `"✓ " + Label`, `AccessibilityLabel` `Label + ", selected"`. Not
  `AccessibilitySelected` (on a button that is aria-pressed, a toggle), not the
  radio pair (the runtime checks a radio on arrow, which would run the action
  and close the sheet).
- `Style` goes to the sheet's card; trigger styling is `Trigger.Style`.
- Tests: `comps/menu_test.go` (shape, no expanded state, OnOpen once, nil
  OnOpen still registers, item runs then dismisses, styles reach their nodes,
  Checked label and name).
- Lesson 4.17 (`lessonMenus`, `chapter4.go`): three notes each with a ⋯ Menu
  (Pin to top / Unpin, Delete) sharing `open` state, a "Sort: Newest ▾" picker
  Menu with Checked items (Newest / Oldest / Title), Restore button.
  Tests: `openSheets`, `tapInSheet` (labels repeat across closed menus, so taps
  are scoped to the one visible Modal), `menuNoteTitles` (skips Modal
  subtrees, whose card titles repeat the note titles).
- Chrome: rows Trip ideas / Groceries / Books to read; one visible Modal after
  ⋯; panel at the overlay's bottom edge with Pin to top, Delete, Cancel;
  Delete removes the row and closes; sort sheet names "Newest, selected";
  picking Title reorders and relabels the trigger "Sort: Title".

## 5. C3 `SearchableSelect` (`comps/searchable_select.go`)

- `Column(Padding 0, Gap XS) > [SearchField, listbox?, status Text]`.
- Controlled on both halves: `Value`/`OnChange`, `Query`/`OnQueryChange`. No
  hook. Options are `core.SelectOption`; `Group` is the row subtitle (a heading
  inside a listbox would be a foreign child); `Disabled`/`GroupDisabled` rows
  carry `core.Disabled(true)` and a guarded handler.
- **List shows** while `Query != ""` and `Query` is not the chosen option's
  label. Nothing for an empty query (show-all-on-focus would need a focus
  flag).
- **Pick**: `OnChange(value)` only if different, then `OnQueryChange(label)`
  (closes the list), then `core.DismissKeyboard(ctx)`.
- **Clear**: `OnQueryChange("")`, and `OnChange("")` when a value was set.
- **Focus decisions** (documented as five numbered parts in the type doc):
  1. The list never takes focus.
  2. The field has no submit, so with `FocusRef` in `core.UseFocusOrder` the
     IME action is Next and skips the list (FocusNext walks refs only). "Enter
     picks top match" rejected: explicit onSubmit suppresses Next
     (`stampTraversal`).
  3. Web: the rows are `ListRow{Selectable}` in a `RoleListBox` column, so the
     runtime gives one tab stop, arrows, Enter/Space; no selection-follows-focus.
  4. A pick dismisses the keyboard; the runtime's blur only acts on a focused
     field.
  5. Web keyboard pick removes the focused option, so focus falls to body.
     Documented, not fixed: `core.Focus` would re-raise a phone keyboard.
- Not an ARIA combobox (no combobox role, aria-expanded/controls/
  activedescendant in core; renderer work). A `RoleStatus` caption says
  "No matches" / "1 match" / "3 matches" / "5 of 6 matches", hidden with
  `Display none` while shut. `Count func(shown, total int) string` overrides
  the English. `MaxResults` default 6; `Filter` overrides case-insensitive
  label contains on the trimmed query.
- An empty listbox is never rendered (it would be an empty tab stop).
- `SearchField.FocusRef` (additive): `core.FocusTarget(s.FocusRef)` on the
  flattened input; nil-safe.
- `searchable_select.go` added to `closedComposites` in
  `comps/nested_composite_test.go`.
- Tests: `comps/searchable_select_test.go` (shut with no query, filter into a
  labelled listbox with group subtitle, cap + status, no matches, shut on
  chosen label and reopen with selected row, pick order value→label and
  re-pick only restores label, disabled refused, clear empties both, FocusRef
  in an order gives `imeAction: next`).
- Lesson 4.17 extended: `searchCountries` (15 entries incl. Ghana and a
  disabled Antarctica), Country SearchableSelect (`MaxResults: 5`) + City
  `core.Input`, `UseFocusOrder(countryRef, cityRef)`, "Ships to:" caption.
  Tests: filters/picks/disabled/no matches/clear; return key Next focuses City.
- **Gotcha:** "Argentina" does not contain "an". The first test expectation
  was wrong; Ghana was added so "an" still matches six (Canada, France,
  Germany, Ghana, Japan, Antarctica).
- Chrome: `enterkeyhint="next"`; typing "an" keeps focus in the field; options
  Canada (tabindex 0) … Japan (-1), aria-selected false; status "5 of 6
  matches"; ArrowDown moves focus and stop to France without picking; Enter
  picks France, list closes, field reads France, activeElement is body.
  Screenshot viewed: framed list under the field, status line, then City.

## 6. Docs, counters

- `docs/components.md`: `## Menu` (after ActionSheet), ActionSheet `Checked`
  bullet, SearchField `FocusRef` paragraph, `## SearchableSelect` with a focus
  question/answer table.
- `docs/api/*` regenerated both commits (`go run ./internal/apidoc/gen`).
- Lesson count 55 → 56: README (alt text + prose), `wasm/index.html`,
  `docs/tutorial-interactive.md`, `internal/shotclaims` (+ test comment),
  `screenshot_test.go`, `core/deeplink.go`, `app_test.go`.
  `docs/images/tutorial-contents.png` retaken with
  `wasm/shots/shoot.sh tutorial-contents` and viewed ("0 of 56 lessons").
- Tracked Go files 491 → 493 → 495 in `wasm/verify/repowalks_test.go` and
  `timings_test.go`. Two of the five sentences wrap the number to a line end
  ("— 491"), so a `N tracked Go files` sed misses them; fix by line.
- Plan: status says Tier C landed as far as it can without a renderer;
  "Decisions that differ from the Tier C sketches" covers C1 and C3.

## 7. Verification harness notes

- Probe scripts live in the scratchpad; header
  `// grmob-shot: {"app": "tutorial", "w": 414, "h": 800}`, open a lesson with
  `GrMobWASM.HostEvent("route", JSON.stringify({ lesson: "4.17" }))`.
- In-page helpers from `shot.mjs` PRELUDE: `tap`, `typeInto(placeholder, v)`,
  `invoke(el, attr, payload)`, `scrollTo`, `settle`, `byAttr`.
- Keyboard can be driven with synthetic
  `new KeyboardEvent("keydown", {key, bubbles: true})` on the focused option;
  the runtime's composite handlers run for it.
- Visible Modals: `[data-node-type="Modal"]` filtered on computed display.

## Next

1. **(carried, partly done · value high) Tag a release.** v0.3.0 is tagged.
   Tier A + B + radio roles + the sweep + Tier C (Menu, SearchableSelect) are a
   natural v0.4.0. Still open: run
   `go run github.com/rohanthewiz/grmob/cmd/grmob@latest new` on a clean
   machine without `-replace`.
2. **(carried, widened · value high) Run lessons 6.6, 6.7, 4.15, 4.16 and 4.17
   on a simulator and a device.** Unverified on hardware: settings-row one-tap,
   `Screen.Footer` pinning, iOS sheet `Dialog`, stepped spinner, ActionSheet
   filler placement on Compose and SwiftUI, Timeline row stretch, horizontal
   StepIndicator scroll, radio semantics, CodeEditor's toolbar role on Compose,
   a Modal inside a ListRow's trailing Row (Menu) on both natives, and
   SearchableSelect's Next action and keyboard dismiss on pick.
3. **(new · value med) ARIA combobox for SearchableSelect.** A `RoleComboBox`
   with aria-expanded/aria-controls (and optionally aria-activedescendant) in
   core, both web exporters, the ARIA fixture and the runtime keyboard, so the
   field announces its list and ArrowDown from the field enters it. Renderer
   work, so outside the comps plan.
4. **(new · value low) Web focus after a keyboard pick in SearchableSelect**
   falls to the page. A platform-aware "return focus on the web only" would
   need a host distinction core does not expose.
5. **(new · value low) `aria-haspopup` for Menu and DatePicker triggers.**
   core has no way to say a button opens a dialog/menu; both triggers state
   nothing.
6. **(new · value med) Exercise `grmob ios -run`** on a machine with a booted
   simulator, and once with none booted to read the hint. (carried)
7. **(carried · value med) Launch the scaffolded app on a simulator and a
   device.**
8. **(carried · value med) `docs/api/core.md` is one 7,700-line page.**
9. **(carried · value low) Migrate hand-rolls to the new comps:** the
   tutorial's `stepper` helper, chapter 6.4's and chapter 1's `checkRow`,
   confirm flows in `todoapp`/`mobileapp`, the signup flow's step header, and
   any overflow "⋯" hand-rolls (now `Menu`). Retakes screenshots.
10. **(carried · value low) A looping `Transition`** so `Spinner` can drop its
    stepping.
11. **(carried, widened · value low) Current-item semantics.** BottomBar's
    ", selected", StepIndicator's ", done"/", current", and now
    `SheetAction.Checked`'s ", selected" are English fallbacks; an
    `aria-current`-style state in core would be the proper shape.
12. **(carried · value low) No `grmob` command refreshes a scaffolded app's
    `wasm/index.html`.**
13. **(carried · value low) The Android remedy drill has never run.**
14. **(carried · value low) `hero.png` is a picture of pictures and its parts
    are held twice.**
15. **(carried · value low) A third face is still a skip.**
16. **(carried · non-goal) Tracked-Go-file count sentences** in `wasm/verify`
    are hand-edited on every commit that adds Go files, and count untracked
    files. Working as designed.
17. **(non-goal) C4 `Carousel`** stays blocked on a host scroll-offset signal
    (`OnScroll` or a paged-scroll node); not low-hanging by the plan's
    definition.
18. **(non-goal) Drawer (plan C2)** stays skipped until asked for; `BottomBar`
    covers the same navigation need with no overlay.
19. **(non-goal) Rename `docs/components.md` to `comps.md`.**
20. **(non-goal) Rewrite `components` in the older plans.**
21. **(non-goal) Trim the copied Android shell's permission declarations.**
22. **(non-goal) Replace the iOS usage strings further.**
23. **(non-goal) Windowing.** Declined in `non_goals.md`.

Closed since the previous doc: "Tier C of the plan" (C1 `Menu` and C3
`SearchableSelect` shipped; C2 and C4 recorded as non-goals above).
