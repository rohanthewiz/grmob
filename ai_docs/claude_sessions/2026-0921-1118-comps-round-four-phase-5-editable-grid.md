# Comps round four, phase 5: EditableGrid

**Session:** 05af2a7c-b98f-4cfc-b084-a455ec97159b
**Date:** 2026-09-21 11:18
**Branch:** master (b5e5dff → this commit)

## The ask

1. `/sl` (loaded `2026-0921-1057-comps-round-four-phase-4-charts` and the
   next list).
2. "Start phase 5 of the comps round four plan" —
   `ai_docs/plans/comps-low-hanging-fruit-4.md`, Phase 5: the two "check
   first" items, then `EditableGrid` N1 (navigate and edit), N2 (kinds and
   validation), N3 (rows), and one lesson.
3. `/sw`.

## What landed

| Piece | Files |
|---|---|
| `EditableGrid`, `GridColumn`, `GridCellKind`, four concerns | `comps/editable_grid.go`, `_test.go` (13 tests, one benchmark) |
| Closed-composite registration | `comps/nested_composite_test.go` |
| Lesson 4.37 "A grid you can type into" | `examples/tutorial/chapter4.go`, `chapter4_test.go` |
| Docs | `docs/components.md` (a section before AppBar), `internal/apidoc/packages.go` (on "lists"), `docs/api/` regenerated |
| Lesson count 79 → 80 | README.md, docs/tutorial-interactive.md, `wasm/index.html`, `internal/shotclaims`, `pagecount_test.go`, `screenshot_test.go`, `docs/images/tutorial-contents.png` re-taken |
| Census 664 → 666 tracked Go files | `wasm/verify/repowalks_test.go`, `timings_test.go` |
| Bookkeeping | the plan (status, phase table, a "What the build changed" block, order rows 7 and 8 struck, one new "still blocked" entry), `ai_docs/todo/next-list.md` |

`go test ./...` and `wasm/verify/run.sh` pass. The round's rule held: no file
under `htmlout/`, `android/` or `ios/` changed, and `wasm/` only for the
lesson count and the census. All three steps landed in one session.

## The two "check first" items, both from source

- **The member walk through a `List`.** `compositeMembers`
  (wasm/grmob-runtime.js) descends through any wrapper, so the cells are
  found. What the sketch missed is ownership: a grid owns rows, and an
  unroled `List` between them breaks that, as it did for `DataTable`. The
  `List` carries `RoleRowGroup` (ARIA allows a rowgroup under a grid; withheld
  from an empty body). core's audit checks no ownership, and a row may own
  columnheaders, so the header row stayed inside the grid.
- **`core.Focus` on a non-input box.** The web honours it (`el.focus()`; a
  gridcell has a tabindex from the composite sync). Compose reads the stamp
  on a Button only, SwiftUI on fields only. Harmless: the focus handed back
  after a commit exists to return the arrow keys, and no native has them.

## Decisions made in the build

- **Two cell roles.** The runtime gives every gridcell a keydown listener
  that owns the arrows, Enter and Space (`handleCompositeKey` is bound to the
  member and reads `e.currentTarget`), and a key typed in a field inside one
  bubbles to it. So a cell that *is* the control (text, number, bool, a row
  header with a menu) is `RoleGridCell`, and a cell that *holds* a native
  control or nothing to press (the editing cell, a choice, read only, a plain
  row number) is `RoleCell`. Price: the arrows step over a choice cell; its
  picker is a Tab stop. The editing cell is keyed `e<col>` against the box's
  `c<col>`, so the swap is a replace patch and the old element's listener and
  tab stop go with it.
- **A blur commits after `gridBlurGrace` (150ms), not at once.** In a browser
  a press on the ✕ blurs the field before the click arrives; an immediate
  commit would remove the ✕ from under the pointer and the discard would have
  committed. The blur marks the editor and `hooks.UseTimeoutWhile` (deps: the
  editor's row key and column) commits unless the ✕, another cell, return or
  a refocus settles it first. That commit reaches `OnChange` from a timer
  goroutine. Reasoned, not seen: headless Chrome dispatches callbacks, not
  pointer events.
- **Tapping another cell commits the open one and edits the new one,** in one
  dispatch; a refused commit holds the reader there. Bool toggle, choice pick
  and the row menu commit first too. Every transition reads state with
  `State.Get` and not a render-time capture, because two run in one dispatch
  and one runs from a timer.
- **Two FocusRefs.** One on the editor; one "landing" ref carried by whichever
  cell `landState` names (the cell below after return, the same cell after
  ✕). No per-cell hooks.
- **Rows by key.** The editor and the open menu hold their row as key plus
  index hint (`rowIndex` tries the hint first). A row deleted under an open
  editor resolves to -1 and the editor is not drawn.
- **The row menu is one `ActionSheet` beside the grid,** not a `Menu` per
  header (`Menu` renders its own trigger). The row callbacks force
  `RowHeaders` on.
- **`MinWidth`** replaced "only when the columns overflow": Go cannot know.
- **Added beyond the sketch:** `GridColumn.Keyboard` (iOS's decimal pad has no
  minus), a built-in "not a number" check ahead of `Validate`, a stand-in
  option for a choice value outside `Options`, default `Weight` 1 for an
  unsized column (header and body are separate rows), `Compact`.
- **Lines are the rows'.** No target has per-side borders, so a row is filled
  with the border colour and shows 1px between and under its cells (`ruled`).

## The size, measured

`BenchmarkEditableGrid30x1000`: 119ms a pass on an M3, about 4µs a cell.
`List` windows the natives, but Go builds and diffs every cell each pass and
a keystroke in the editor is a pass. The doc states about 5,000 cells
(10 × 500), not the plan's "30 columns by any number of rows". Measured in Go
rather than on the emulator because Go is the bound.

The patch-count test holds for a changed value: at most two patches, one
cell. Entering or leaving EDIT also re-binds every later cell's `onClick`,
because callback IDs are issued in render order (core/event.go `beginPass`)
and the editor registers three void callbacks to the box's one. N-073.

## The look

Throwaway shot scripts (`wasm/shots/scripts/zz-grid*.js`, deleted), with
`GRMOB_SHOTS_OUT=<scratch> ./shoot.sh zz-…`. A lesson in a collapsed chapter
needs `tap("The Widget Library")` before `tap(<lesson title>)`. **Two defects
found, both fixed:**

- **The picker drew its own frame and padding** inside the cell: rows half
  again as tall, one letter of the choice ("H", "F"). The cell is the frame
  now (`Padding(0)`, `BorderWidth(0)`, transparent), as for the editor.
- **A grey band under the last row.** The first build filled the grid with the
  line colour and gapped the rows 1px; a body taller than its rows showed the
  leftover. Hence `ruled`.

The lesson's four columns plus row numbers also read "$12…" in a phone's demo
panel, so it sets `MinWidth: 460` and scrolls sideways.

**A `--probe` script drove the keyboard in headless Chrome** (synthetic
`keydown`s): grid children are `row`, `rowgroup`; 16 gridcells, one tab stop;
ArrowDown and ArrowRight move (Right steps over the choice cell); Enter opens
an `INPUT` that is `document.activeElement`, in a `role="cell"`; a Space
keydown in it is not `defaultPrevented`; after a submit the active element is
"Item, row 3, Bus pass" and ArrowUp works from there.

## Test notes

- `comps` tests may import `reconcile` (no cycle).
- A widget that declares a composite container role must be listed in
  `nested_composite_test.go`'s `closedComposites`, with its reason.
- `core.Text` takes `...StyleProp`, not `PropsAndChildren`; paddings take
  `int`; "display none" is `core.Display(core.DisplayNone)`.

## Not run

- `android/verify` and `ios/verify`: nothing under either changed.
- No device, emulator or screen reader. All of it is N-072.

## Next

Closed: None. Declined: None. Raised: N-072, N-073.
Deferred: None. Promoted: None.
Updated: None. Full list: `ai_docs/todo/next-list.md`.
