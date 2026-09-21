# Comps round four, phase 3: structure (TreeView, Wizard)

**Session:** 33148f68-892a-4edd-aafa-9ef338fe0b6b
**Date:** 2026-09-21 10:35
**Branch:** master (414c696 → this commit)

## The ask

1. `/sl` (loaded `2026-0921-1019-comps-round-four-phase-2-inputs` and the
   next list).
2. "Start phase 3 of the comps round four plan" —
   `ai_docs/plans/comps-low-hanging-fruit-4.md`, Phase 3: L1 `TreeView`, L2
   `Wizard`, and one chapter 4 lesson.
3. `/sw`.

## What landed

| Piece | Files |
|---|---|
| L1 `TreeView` | `comps/tree_view.go`, `_test.go` |
| L2 `Wizard` | `comps/wizard.go`, `_test.go` |
| Lesson 4.35 "Trees and wizards" | `examples/tutorial/chapter4.go`, `chapter4_test.go` |
| Docs | `docs/components.md` (Wizard and TreeView, after StepIndicator), `internal/apidoc/packages.go` (both files on "structure"), `docs/api/` regenerated |
| Lesson count 77 → 78 | README.md, docs/tutorial-interactive.md, `wasm/index.html`, `internal/shotclaims`, `pagecount_test.go`, `screenshot_test.go`, `docs/images/tutorial-contents.png` re-taken |
| Census 652 → 656 tracked Go files | `wasm/verify/repowalks_test.go`, `timings_test.go` |
| Bookkeeping | the plan (status, two "What the build changed" blocks, order rows 4 and 5 struck, one new "still blocked" entry), `ai_docs/todo/next-list.md` |

`go test ./...` and `wasm/verify/run.sh` pass. The round's rule held this
time: no file under `htmlout/`, `android/` or `ios/` changed, and `wasm/`
only for the lesson count and the census.

## L1 `TreeView`: decision (a), nested lists

```
Column role=list "Project files"
  Box role=listitem level=1            keyed by TreeNode.ID
    Row role=button expanded "docs"    comps' shared disclosure, Heading: false
    Row                                 (open branches only)
      Box  Indent wide, hidden          the indent, as a leading spacer
      Column role=list "docs", grows
        Box role=listitem level=2
          Row role=button current "guide.md"
```

- **The button is inside the item, not the item.** A list's child must be a
  listitem and a listitem is not a control (`ListRow.NestingLevel`'s doc
  states the same rule). No audit checks the list pair, so
  `TestTreeViewNestsListsAndCountsLevels` does.
- **The plan's "check first" was answered from source.**
  `Style.AccessibilityNestingLevel`'s doc already says it: the web writes
  `aria-level`, neither native has a depth property. Harmless by
  construction; what is heard is N-068.
- **A branch toggles, a leaf selects.** One row, one target. A selectable
  branch would be two buttons with one name; the data can say it instead (a
  first child standing for the branch).
- **`TreeNode.Branch bool`** was added: an empty folder, or one that loads
  on first open.
- **`Indent` is an `int`** (core's paddings are), and it is a spacer leading
  a Row and not `core.PaddingLeft`, which is a physical side: a Row lays out
  from the reading direction's start, so RTL indents from the right. A leaf
  keeps an empty 20pt chevron column so labels line up.
- **The chosen leaf states `core.CurrentTrue`**, only when `OnSelect` makes
  the row a button. `aria-selected` is not allowed on a button.
- `ConcernTreeViewDuplicateID` walks shut branches too;
  `ConcernTreeViewInert` for a branch with no `OnToggle`, whose handler is
  guarded so release builds do not panic.
- Holds no hook.

## L2 `Wizard`

- **Focus does not move to the step's heading.** The plan called it
  "settled by precedent" (c1829c4), and that commit gave `core.Focus` to a
  Compose *Button*. `focusAction` is read by fields, the editors and that
  Button; no target focuses a Text. Renderer work, so it left the phase as
  **N-069**. What stands in: a visible `RoleStatus` line under the
  indicator ("Step 2 of 3, optional") and the title as a level-2 heading.
  VoiceOver speaks no live region, so iOS is silent on a step change.
- **`CanAdvance` became `Blocked`**, so the zero value advances (the
  argument that made `PollOption.Mine` a bool).
- **`Optional` means something:** Blocked and Optional keeps Next enabled
  and reads `SkipLabel`. Never on the last step, where the button submits.
- **`DetachFooter`** beside the exported `Footer()`, so lifting the footer
  into `Screen.Footer` does not draw it twice. Inline is the default, as the
  plan decided (N-002's pinning checks are still open). `Finish` with a nil
  `OnFinish` is drawn disabled.
- **The body is `Keyed` by step index**, so a step change replaces it and
  does not morph one form into the next.
- `PositionLabel` localizes the status line. `StepIndicator.OnTap` is passed
  `OnChange` as it is: done steps only was already its rule.
- Holds no hook, so it can sit in a `core.IfElse`; lesson 4.35 does, and
  its test ends on an empty concern list with bodies coming and going. The
  doc comment says the body rule in capitals, as the plan asked.

## Two test-harness notes

- A multi-pass test over a bare context must call `ctx.Reset()` after
  `BeginRenderPass` (see `renderPass` in `accordion_test.go`). Without it the
  cursor never rewinds and debug mode reports drift that is the test's own.
  My first hook-slot test failed this way, not the widget.
- `comps.Button`'s disabled state is `Style.Disabled` on the node, not a
  prop.

## The look

Two throwaway shot scripts (`wasm/shots/scripts/zz-*.js`, deleted), taken
with `GRMOB_SHOTS_OUT=<scratch> ./shoot.sh zz-tree zz-wizard`. A lesson in a
shut chapter needs `await tap("Chapter 4 — The Widget Library")` before the
lesson's title. **Nothing was found this time**, the first round of four:
labels align across branch and leaf rows, indents accumulate, the chosen
leaf is tinted; the wizard's status line, heading and Back / Skip sit right.
The step strip clips its third step at 414pt and scrolls, by design.

## Not run

- `android/verify` and `ios/verify`: nothing under either changed.
- No device, emulator or screen reader. The DOM was not read for
  `aria-level`. All of it is N-068.

## Next

Closed: None. Declined: None. Raised: N-068, N-069.
Deferred: None. Promoted: None.
Updated: None. Full list: `ai_docs/todo/next-list.md`.
