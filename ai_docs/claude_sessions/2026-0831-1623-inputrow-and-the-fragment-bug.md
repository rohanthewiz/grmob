# Session: `InputRow`, a Tier 2 re-survey that collapsed most of it, and the Fragment bug underneath `SegmentedControl`

**Session ID:** session_01Ej9aSqY1yefk9am5LZG367
**Date:** 2026-08-31, ~16:23
**Branch:** master
**Follows:** `2026-0831-1533-screen-scaffold.md`

## Goal

Tier 2 of the component gap analysis, taken in order: `InputRow`, then
`StatTile`. It ended up being `InputRow`, a re-survey that retired three of
Tier 2's five items, an htmlout renderer fix, and `SegmentedControl`.

## Part 1 — `components.InputRow`

The composer: a field that fills the row, and an optional trailing button that
commits it. Two sites, near-identical down to the `Gap(8)` and the
`FlexGrow(1)` — `chat`'s message composer and `todoapp`'s entry row.

```
Row (Gap)
  ├─ Input   ← FlexGrow(1), value / placeholder / onChange / onSubmit
  └─ Button  ← only when Button.Label is set; OnTap defaults to OnSubmit
```

The duplicated *layout* is the visible half. The half worth owning is the
**wiring**: both sites named the same commit helper twice, once as
`InputWithSubmit`'s `onSubmit` and once as the button's `OnTap`. The button
*is* the field's submit rendered as a tap target, so `OnSubmit` drives both and
they cannot drift. An explicit `Button.OnTap` still wins, but has to be said
out loud.

### `Gap` zero means the theme's step — the *opposite* of `Screen`'s

Worth stating plainly because two widgets now disagree on purpose:

- **`Screen.Gap`** — zero is *unset*. The spacing between a screen's sections
  is the app's decision, and a theme's `Column` base may already carry one that
  an unconditional `Gap(0)` would overwrite.
- **`InputRow.Gap`** — zero is *the theme's `SM` step*. This gap is the
  widget's **internal layout** (the field and the button must not touch), the
  way `FormField` owns the spacing between its label and its input.

Both bundled themes put `SM` at 8, which is exactly what both hand-written
sites had picked, so the migrations came out byte-identical. `Style:
[]core.StyleProp{core.Gap(0)}` is the escape hatch for a genuine zero.

### Two smaller calls

- **A zero `Button` renders no node at all** (keyed on `Label`), so a search
  field committing on the return key is the same widget with one less field
  set. Passing the whole `Button` through — rather than re-exporting
  `ButtonLabel`/`ButtonVariant`/`ButtonHint`/… — keeps `Variant`, `Emphasis`,
  `Disabled` and `Style` working for free.
- **A nil `OnSubmit` builds a plain `core.Input`**, not one wired to a no-op:
  the renderers read the prop to decide whether to advertise a submit
  affordance on the keyboard.

The input is *owned, not slotted* (unlike `FormField`) — so the field takes no
per-call styling. Documented as a limit rather than papered over with a
speculative `InputStyle`; a composer needing that has outgrown the widget.

### Verification

Eight mutations, each caught by exactly one test:

| mutation | caught by |
|---|---|
| `Gap` hardcoded to `8` | discriminating theme, `SM: 5` |
| `Style` applied before `Gap` | `Style Gap did not win: got 20, want 0` |
| `OnTap` not inherited | button tap never reaches `OnSubmit` |
| `OnTap` unconditionally overwritten | explicit-`OnTap` test |
| button always appended | zero `Button` renders a node |
| always `InputWithSubmit` | `onSubmit` prop present with nil handler |
| field doesn't grow | `FlexGrow = 0` |
| button rebuilt from `Label` | `Variant` / hint dropped |

The gap test needed a purpose-built theme for the third session running —
both bundled themes agree on 8, so a hardcoded literal passes under either.
**This is now the default move, not a special case.**

Handler wiring is proved by real dispatch (`ctx.TriggerCallback` /
`TriggerTextCallback`), not struct inspection — the only way to show the button
and the keyboard land on one func.

## Part 2 — the Tier 2 re-survey, and what it retired

`StatTile` was next. Before building, the three claimed sites were checked
against the current tree (the gap analysis's line numbers still point at live
code, so this was not staleness — the sites were simply never the same shape):

| item | claimed | actual |
|---|---|---|
| `InputRow` | 2 | **2** ✓ |
| `StatTile{Caption,Value,Delta}` | 3 | **1** — and **0** sites with a `Delta` |
| `EmptyState{Icon,Title,Message,Action}` | 2 | **1** — a single `Text` |
| `ToggleRow{Label,Checked,OnChange}` | 2 | **0** — both already on `ListRow` |
| `SegmentedControl` | 3 | **2** |

- **StatTile** — only `fintechapp`'s `BalanceCard` is caption-above-figure.
  `mobileapp`'s `counterTab` bakes the caption into the string (`"Count: 5"`)
  and has a Button mid-block; `todoapp`'s footer is one dim line in a
  `ListRow`. Every other large-text site in the tree is a screen *title*.
- **ToggleRow** was silently completed by the `ListRow` session — both sites
  are already `ListRow{Leading: Checkbox, Title}`. It would now be a facade
  over `ListRow` saving one field.

The gap analysis's own Tier 3 rule ("hold until an example needs one") applies
to all three. **Tier 2 is effectively closed.**

Recorded here so the next session does not re-derive it.

## Part 3 — the bug under `SegmentedControl`

`core.For` wraps its generated children in a `Fragment`. Checking how that
renders before building on it turned up a three-target divergence:

| target | grouping node (`Fragment` / `Theme`) |
|---|---|
| iOS | `Group { PlainChildren }` — **transparent** |
| Android | `RenderChildren` into the parent's scope — **transparent** |
| htmlout | `<div>` via `tagForType` — **opaque** |

Neither node type ever carries a `Style` (all three construction sites build
them with `Children` and nothing else), so the `div` was not merely redundant:
**inside a flex container it becomes the single flex item**, and the parent's
`gap`, `flex-direction` and alignment stop at the wrapper instead of reaching
the children.

Visible in `todoapp`: the filter bar's `Row` carried `gap:8px` and exactly one
child, so the three chips sat flush in the browser while keeping their spacing
on device.

```html
<!-- before -->
<div style="...flex-direction:row; gap:8px">
  <div>                          <!-- Fragment: no flex, no style -->
    <button>All</button>         <!-- gap:8px never reaches these -->
    <button>Active</button>
    <button>Done</button>
  </div>
</div>
```

### The fix, and how it was verified

`renderNode` now emits grouping-node children directly, before the shared
attribute block. `tagForType`'s `Fragment`/`Theme` entry is unreachable and
says so.

Verified across **all six** examples by extracting every attributed tag, every
style, every callback ID and every text run in order:

```
chat.txt               content+styles identical: True   bare <div>: 6 -> 4
runtime.txt            content+styles identical: True   bare <div>: 0 -> 0
materialwallet.html    content+styles identical: True   bare <div>: 6 -> 6
layout.html            content+styles identical: True   bare <div>: 2 -> 1
todoapp.html           content+styles identical: True   bare <div>: 2 -> 1
mobileapp.html         content+styles identical: True   bare <div>: 3 -> 2
```

Nothing changed but the disappearance of four styleless wrappers. Four new
htmlout tests, all confirmed to fail with the fix reverted — including one
pinned on *child count* rather than on the absence of a div, since it is the
number of things the flex container can see that was wrong.

### Two knock-on comment corrections

The fix made existing comments false, which is the same defect it was fixing:

1. **`chat/main.go`'s `MessageList`** documented its bubble-margin choice as a
   framework constraint — "a `Gap` here would space the Fragment against its
   siblings". That was an htmlout artifact and **never applied to the natives
   at all**. Corrected; the margin stays because it is also the more honest
   expression (spacing belongs to a bubble wherever placed), and switching to
   `Gap` is flagged as cleanup rather than a fix.

2. **`MaybeProp`'s rationale** — "an empty `Fragment` takes a slot in the flex
   layout and opens a stray Gap" — was the *same artifact*, restated in five
   live places. Probed it directly:

   ```
   node children: 3      [0] Text  [1] Fragment children=0  [2] Text
   <div style="...gap:8px"><span>a</span><span>b</span></div>
   ```

   An empty Fragment now renders nothing on any target. **`MaybeProp`'s
   justification survives** on the grounds that actually hold: the node still
   exists for the reconciler to walk and diff every pass and still occupies a
   child index, and it is still the only expression form for an optional
   *style prop* or *handler* (`If` is `View → View`). Corrected in
   `core/conditionals.go`, its test, `components/screen.go`,
   `examples/chat/main.go` and `docs/concepts/views.md` — the claim that
   changed is the *rendered layout* one, not the widget's reason to exist.

   Session notes were left alone: they are a historical record.

`docs/concepts/views.md` gained a **"Fragments are transparent"** section
documenting the three-target behavior and the fixed divergence.

## Part 4 — `components.SegmentedControl`

A controlled single-select rendered as a row of chips. The extraction of
`todoapp`'s filter bar — which is the loop `Chip` itself came out of. `Chip`
solved one segment; what stayed hand-written was the row, the gap, the keying,
the index comparison and the per-segment accessibility label — the parts with
the quiet failure modes.

### `Segment` is a template, not pass-through fields

```go
components.SegmentedControl{
    Labels:    filterLabels,
    Selected:  active,
    OnSelect:  onSelect,
    KeyPrefix: "filter-",
    Segment: components.Chip{
        Style:             []core.StyleProp{core.FontSize(13)},
        SelectedStyle:     []core.StyleProp{core.BackgroundColor(colorAccent)},
        AccessibilityHint: "Filters the task list",
    },
    SegmentLabel: func(label string, _ int) string {
        return "Show " + strings.ToLower(label) + " tasks"
    },
}
```

`Label`, `Selected` and `OnTap` on the template are overwritten — those three
are exactly what the control computes. The alternative was re-exporting Chip's
surface as `SegmentStyle`/`SelectedSegmentStyle`/`SegmentHint`, which grows a
field every time `Chip` does. **Same move `InputRow` makes with `Button`** —
this is now the established pattern for a widget that renders many of another.

### Three smaller calls

- **`SegmentLabel` is a function** because the accessibility name is the one
  thing that varies per segment and is not derivable from the caption
  ("Active" → "Show active tasks"). A parallel `[]string` would have to be kept
  in step with `Labels` by hand. `Chip` still appends `", selected"`.
- **An out-of-range `Selected` selects nothing** — a legal "no scope chosen"
  state, not a defensive clamp, and not a reason to grow a fourth segment.
- **Segments go straight into the row, not through `core.For`.** Now that the
  grouping node is inlined everywhere the HTML is the same either way, but the
  `Fragment` is still a node the reconciler walks for no gain — the control
  already has the slice and is already building the row's argument list.

### Verification

`todoapp/filterBar` migrated **byte-identical** to the post-fix baseline. The
existing `examples/todoapp/chip_migration_test.go` now **transitively pins the
new widget against the original hand-rolled markup** across all three selected
states — an acceptance check that has survived two extractions.

Eleven mutations, each caught:

| mutation | caught by |
|---|---|
| `Gap` hardcoded to `8` | discriminating theme (`SM: 5`) |
| `Style` before `Gap` | `StyleOverridesTheWidgetsOwnGap` |
| segments wrapped in a Fragment | `SegmentsAreDirectChildrenOfTheRow` |
| no keys / `KeyPrefix` ignored | `KeysEachSegment` |
| closure captures the last index | `ReportsTheTappedIndex` |
| `Selected` off by one | five tests |
| template not copied | `AppliesTheSegmentTemplate` |
| template `OnTap` wins | `TemplateCannotOverrideComputedFields` |
| nil `OnSelect` unguarded | `WithoutOnSelectIsInert` |
| `SegmentLabel` always set | `NilSegmentLabelFallsBackToChip` |

Keys are pinned in the widget's own tests rather than left to a markup diff:
they never appear in exported HTML but drive reconciler matching and native
view recycling.

The duplicate-caption claim in the doc comment was **verified rather than
asserted** — debug mode reports it and names the container:

```
concern: {Kind:duplicate-key Detail:Row has multiple children with key
"filter-All": sibling keys must be unique for keyed reconciliation to track
identity Count:1}
```

## Method notes worth keeping

- **Check the gap analysis's evidence before building to its spec.** Three of
  five Tier 2 entries did not survive contact with the current tree. The line
  numbers were fine; the *shapes* were never what the table said.
- **Check how a primitive actually renders before building on it.** The
  Fragment bug was found by asking what `core.For` emits, not by hunting bugs.
- **When a fix falsifies a comment, the comment is part of the fix.** Five
  places asserted a symptom that no longer exists. Leaving them would have been
  the same defect the fix addressed.
- **A discriminating theme is the default for any "reads the theme" test.**
  Third session running.

## Files touched

**New**
- `components/input_row.go`, `components/input_row_test.go` (10 tests)
- `components/segmented_control.go`, `components/segmented_control_test.go` (13 tests)

**Modified**
- `htmlout/export.go` — grouping nodes emit children directly
- `htmlout/export_test.go` — 4 tests
- `core/conditionals.go`, `core/conditionals_test.go` — corrected `MaybeProp` rationale
- `components/screen.go` — same correction
- `examples/chat/main.go` — `Composer` → `InputRow`; two comment corrections
- `examples/todoapp/app.go` — entry row → `InputRow`; `filterBar` → `SegmentedControl`
- `docs/components.md` — `## InputRow`, `## SegmentedControl`
- `docs/concepts/views.md` — "Fragments are transparent"; `MaybeProp` correction
- `docs/index.md` — widget list

`go build ./...`, `go vet ./...`, full `go test ./...` green. All six example
outputs re-verified unchanged at the end. `examples/todoapp/store.go` remains
the only `gofmt` offender. Throwaway baseline-dump tests were added under an
env var and **deleted after**.

## Backlog after this session

- **Tier 2 is closed.** `StatTile`, `EmptyState` and `ToggleRow` lack the
  evidence to justify building — see the table above. Revisit only when an
  example actually needs one.
- **`fintechapp`'s `BalanceCard`** still carries a comment saying
  `components.Card` renders its caption in the *Subtitle* role and is "visibly
  wrong". That gap is real and unclosed; a 1-site `StatTile{Caption, Value}`
  would close it if a second site ever appears.
- **`social`'s tab bar was deliberately not migrated** to `SegmentedControl`.
  It is structurally identical (a row of controls, one selected) but
  semantically navigation, not selection — iOS ships `UITabBar` and
  `UISegmentedControl` as different controls — and its ghost-glyph treatment is
  a documented contrast fix. Converting it would be a redesign.
- **`chat`'s `MessageList` could now use `Gap`** instead of a per-bubble
  margin; the constraint that blocked it is gone. Cleanup, not a fix.
- **`chat`'s scaffold sets no `Fill`** even though its middle region scrolls —
  unchanged from last session, worth a look when the keyboard-aware scroll
  area lands.
- **Still open against the theme, unchanged:** `DefaultTheme.Components.Button`
  is white on `#007AFF` at **4.02:1**, below WCAG AA for 17px text; and a
  second palette value per status role (an "on-light" tone) would make Button's
  Outlined/Ghost legible for Success and Warning.
- **`Variant` has two consumers** (Badge, Button). `Alert`/banner is the
  obvious third; `Chip{Variant:}` the fourth — Chip still hand-rolls a selected
  treatment that `Emphasis` could express.
- **A `Neutral`/muted variant** mapping to Surface is still what a status set
  usually wants next; `Button.Disabled` already hand-rolls exactly that pair.
- **Renderer work, none of it blocking, unchanged:**
  1. Proportional flex weights on iOS (custom SwiftUI `Layout`) — lets
     `ProgressBar` move off percentage widths.
  2. `AlignItems: "stretch"` on both renderers — unblocks `Separator.Vertical`.
  3. A `ContentMode` prop on `Image` — unblocks avatar images that fill rather
     than letterbox.
  4. No renderer carries a disabled state.
- **Still true from seven sessions ago:** `ROADMAP.md` lists `UseMemo` and
  `UseReducer` as done; neither identifier exists in the tree.
