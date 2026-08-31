# Session: building components.ListRow and migrating the five call sites

**Session ID:** session_01Ej9aSqY1yefk9am5LZG367
**Date:** 2026-08-31, ~01:14
**Branch:** master

## Goal

Act on the top recommendation from the previous session
(`2026-0831-0058-component-gap-analysis.md`): build `ListRow`, the most-repeated
hand-rolled shape in the examples, then migrate all five instances onto it.

Both halves are done. The widget landed first, with tests; the migrations
followed as a separate explicit request.

## What was built

### `components/list_row.go`

A struct widget on `core.Row`, pure composition — **zero renderer work**, per
the constraint the gap analysis established.

```go
type ListRow struct {
    Leading  core.View   // checkbox, icon, avatar
    Title    string
    Subtitle string
    Content  core.View   // overrides Title/Subtitle — the Card.Header idiom
    Trailing core.View   // badge, value, delete button
    OnTap       func()
    OnLongPress func()
    Selected bool
    Style         []core.StyleProp
    SelectedStyle []core.StyleProp
    AccessibilityLabel, AccessibilityHint string
}
```

Rendered shape:

```
┌ Row ─────────────────────────────────────────────────────────┐
│ [Leading]  ┌ Column FlexGrow(1) ────────────┐    [Trailing]  │
│            │ Title                          │                │
│            │ Subtitle                       │                │
│            └────────────────────────────────┘                │
└──────────────────────────────────────────────────────────────┘
            └──────── takes all the slack ────┘
```

### The four design decisions, and why

**1. Pinning is `FlexGrow`, never `JustifyBetween`.** This is the disagreement
the gap analysis found split across the examples, and it is not a stylistic
tie. `JustifyBetween` distributes slack between *every* pair of children, so it
stops meaning "pin the trailing element" the moment a row has anything other
than exactly two slots — a row with no trailing slot pushes leading and title
apart. A growing middle column gives all the slack to one child, so leading and
trailing sit hard against the row's edges in every configuration. A test
asserts the row sets no `JustifyContent` at all, so the wrong mechanism cannot
creep back.

**2. The middle column renders even when empty** — deliberately unlike `Card`,
which omits empty regions. Here the middle is *structure, not content*: making
it conditional would make the pinning conditional too, which is precisely the
inconsistency the widget exists to remove.

**3. No accessibility label is synthesized from `Title`.** A row is a compound
control whose slots (a badge's amount, a trailing control's own name) carry
meaning the widget cannot see, and labelling the container overrides how those
children are announced. Naming the row stays the caller's job, exactly as it is
for `Chip`. What the widget *does* own is the `", selected"` suffix — the
convention that was copy-pasted by hand at two call sites.

**4. `OnLongPress` was added beyond the analysis's proposed struct.** Without
it the widget could not replace `mobileapp`'s feed row, which uses one. Both
handlers are guarded on non-nil, so a presentational row registers no callback
at all — an unconditional `OnClick` would put a live handler ID in the props of
every row in a 30-row list and hold it through the pass's callback sweep for
nothing.

### `components/list_row_test.go`

11 tests: slot order, the FlexGrow-not-Justify contract, the empty-middle rule,
`Content` precedence over `Title`/`Subtitle`, callback firing through
`ctx.TriggerCallback`, no-handler cleanliness, selection theme default,
`SelectedStyle` beating caller `Style`, caller `Style` beating widget defaults,
the a11y suffix, and the deliberate absence of a synthesized label.

Tests locate the middle column by predicate (`Type == "Column" && FlexGrow == 1`)
rather than child index, following the `helpers_test.go` discipline.

## The five migrations

| Site | Before | After |
|---|---|---|
| `fintechapp/main.go` `TransactionItem` | `Row(JustifyBetween, AlignItemsCenter, Text, Badge)` | `ListRow{Title, Trailing: Badge}` |
| `todoapp/app.go` `todoRow` | `Row` + manual `FlexGrow(1)` on the title | `ListRow{Leading: Checkbox, Content: Text, Trailing: Button}` |
| `todoapp/app.go` footer | `Row` + `FlexGrow(1)` + `core.If` around the clear button | `ListRow{Content, Trailing: clearButton}` |
| `mobileapp/app.go` feed row | `[]core.PropsAndChildren` append workaround + manual `", selected"` | `ListRow{Title, Selected, SelectedStyle, OnTap, OnLongPress}` |
| `mobileapp/app.go` subscribe toggle | bare `Row(Checkbox, Text)` | `ListRow{Leading, Title}` |

### Two of the three core gaps closed at these call sites

- **Gap 3 (`[]core.PropsAndChildren` append idiom).** `mobileapp`'s feed row used
  it because `core.If` emits a real child node and therefore cannot carry a
  conditional *style* prop. `SelectedStyle` is that conditional, declared. The
  workaround and its explanatory comment are gone.
- **Gap 3 again, other shape.** `todoapp`'s footer wrapped its bulk-clear button
  in `core.If`, which emitted an empty child for the reconciler to diff on every
  pass. It is now a `var clearButton core.View` assigned only inside the branch,
  passed as `Trailing` — a nil slot emits no node at all. (Interface nil, not a
  typed nil: the variable is only ever assigned inside the `if`.)

Gap 1 (`UseStyle` silently dropping most of `Style`) was **not** hit, because
`ListRow` applies `core.FlexGrow(1)` as a direct `StyleProp` rather than through
`UseStyle`. It remains open and still gates `Avatar`/`ProgressBar`/`Separator`.

### Judgment calls in the migrations

- **`Content` rather than `Title` in two places.** `todoapp`'s task title must
  dim when done, and its footer count is 13px secondary ink. `Title` takes the
  theme's Body role verbatim with no per-use hook, so those two use the escape
  hatch. The other three use `Title` and pick up themed typography they did not
  have before — a small intentional visual change.
- **`Selected` is deliberately not used for todoapp's done state.** The widget's
  flag appends `", selected"`; that app announces `", completed"` /
  `", not completed"` via `rowAccessibilityLabel`. Done is not selected. A code
  comment records this so nobody "fixes" it later.
- **All migrated rows now centre their slots vertically.** The bare `Row`s in
  `mobileapp` did not, leaving a checkbox on its top edge against the text.
  That is the fix, not a regression.
- **No new field was added for a styled title.** `Content` covers it; a
  `TitleStyle` field would have been feature creep discovered mid-migration.

## Verification

`go build ./...`, `go vet ./...`, and the full `go test ./...` are green
(components, core, examples/{mobileapp,social,todoapp}, hooks, htmlout, mobile,
reconcile, render).

The existing example tests were the real check and all still pass unchanged:

- `mobileapp/app_test.go` `TestFeedTabListGestures` locates the first `Row` with
  `onClick` in the live tree and dispatches tap and long-press — `ListRow` still
  renders a `Row` carrying both, so this passed without edits.
- `todoapp/app_test.go` drives the whole lifecycle by finding `Button "✕"`,
  `Button "Clear completed"`, and `Checkbox` — all preserved as `ListRow` slots.
- Both packages run with `core.SetDebugMode(true)` and assert an empty concern
  dump; `ListRow` calls no hooks, so no cursor drift, and its children carry no
  keys, so no duplicate-key findings.

Beyond the suite, each migrated tree was rendered and its node structure
inspected directly (a throwaway module in the scratchpad with a `replace`
directive, driving `mobile.RenderInitial` / `TriggerCallback`):

```
--- todo row (done) ---
Row  [Gap=10, AlignItems=center, Pad=8/8, AccessibilityLabel=Buy milk, completed, Transition=200ms ease-in-out]
  Checkbox
  Column  [Gap=4, FlexGrow=1]
    Text {"content": "Buy milk"}  [TextColor=#3C3C4399, FontSize=16]
  Button {"label": "✕"}  [Background=#B3261E, AccessibilityLabel=Delete Buy milk]

--- mobileapp feed row (selected) ---
Row {"onClick","onLongPress"}  [Gap=8, AlignItems=center, Pad=12/12,
     Background=#E8F0FE, AccessibilityLabel=Article 1, selected, ...]
  Column  [Gap=4, FlexGrow=1]
    Text {"content": "Article 1"}
```

The `", selected"` suffix above is now widget-generated; the neighbouring
unselected row carries a plain label and no background. The fintech HTML export
confirms the badge pinned by `flex-grow:1` on the middle column.

## Files touched

**New**
- `components/list_row.go`
- `components/list_row_test.go`

**Modified**
- `examples/fintechapp/main.go` — `TransactionItem`
- `examples/todoapp/app.go` — `todoRow`, footer row, new `clearButton` var
- `examples/mobileapp/app.go` — feed row, subscribe toggle, `components` import
- `docs/components.md` — `## ListRow` section (diagram, pinning rationale,
  migration notes)
- `docs/index.md` — widget list

Pre-existing formatting noise left alone deliberately: `components/badge.go` and
`examples/todoapp/store.go` are both unformatted in the tree already and are
unrelated to this change.

## Backlog after this session

From the gap analysis, still unbuilt and still free:

- **Tier 1:** `Button` variants (5 instances), `Screen` scaffold (5),
  `Separator` (2). `ListRow` is done.
- **Tier 2:** `InputRow`/composer, `StatTile`, `EmptyState`, `ToggleRow`
  (partly absorbed — `mobileapp`'s toggle is now a `ListRow`),
  `SegmentedControl`.
- **Core gaps:** gap 1 (`UseStyle` drops Width/Height/flex/accessibility
  fields — the highest-value fix, gates `Avatar`/`ProgressBar`/`Separator`) and
  gap 2 (palette has no `Border`/`Success`/`Warning`, which is why the hairline
  color is hardcoded in two packages) are both still open.
- **Incidental, still true:** `ROADMAP.md` lists `UseMemo` and `UseReducer` as
  done; neither identifier exists in the tree.

Also untouched: `examples/{chat,social,layout}` have row-ish shapes that were
not among the five and were left alone.
