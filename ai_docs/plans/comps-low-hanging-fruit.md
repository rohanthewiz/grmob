# Low-hanging fruit for `comps`

**Status:** Tier A landed 2026-09-12. A1 `Dialog` and A2
`SwitchRow`/`CheckboxRow` shipped with tutorial lesson 6.6 "Dialog and settings
rows". A3 `Stepper`, A4 `BottomBar` + `Screen.Footer`, A5 `Spinner` and A6
`Rating` shipped with lesson 4.15 "Small controls". **Tier B landed
2026-09-12**, one widget per commit. B1 `ActionSheet` and B3 `Snackbar` (with
`hooks.UseTimeoutWhile`) shipped with lesson 6.7 "Action sheets & snackbars".
B2 `RadioGroup`, B4 `StepIndicator` and B5 `Timeline` shipped with lesson 4.16
"Radio groups, steps & timelines". Tier C is open.

**Decisions that differ from the Tier A sketches below:**

- A3 `Stepper` buttons are outlined, not ghost: a ghost "−" has no visible
  edge. Bounds apply only when `Max > Min`, so the zero value is unbounded.
- A4 `BottomBar` cells each take `FlexGrow(1)` instead of `JustifyAround`, so
  the tap targets tile the bar. The current item is announced with ListRow's
  ", selected" name suffix. `Selected: -1` is the toolbar form.
- A5 `Spinner` uses a new `hooks.UseIntervalWhile` rather than
  `hooks.UseInterval`. `UseInterval` requests a render on every tick, so a
  spinner that had ever mounted would re-render the app ~12 times a second for
  the life of the process. `Hidden` pauses the ticks outright. `Rotate` is not
  interpolated on either native, so host-side transitions could not stand in
  for the steps. The ring carries an orbiting dot because a uniform ring looks
  static when rotated.
- A6 `Rating` fills with the warning on-light tone, not `Colors.Warning`,
  which is about 2:1 on a light surface.

**Decisions that differ from the Tier B sketches below:**

- B1 `ActionSheet`: the placement question is answered. Both web targets
  centre a Modal's content in a flex column, Compose weights a `FlexGrow`
  child inside its Dialog, and SwiftUI already presents a Modal as a bottom
  sheet and ignores grow. The widget therefore puts a growing, invisible
  filler above its Card, with no `ZStack`. The filler also reports taps above
  the panel as a dismiss, since those land inside the Modal's content on the
  web and Compose. The actions are `comps.Button`s, not
  `RoleListBox`/`RoleOption`: an option announces "not selected", and these
  are commands. Picking an action calls `OnTap` then `OnDismiss`.
- B3 `Snackbar` runs on a new `hooks.UseTimeoutWhile(ctx, active, fn, delay,
  deps...)`, not `hooks.UseTimeout`. `UseTimeout` arms once per slot for the
  life of the app, so a snackbar built on it would time out only the first
  time it was shown. The new hook arms on each rising edge of `active`,
  cancels when `active` falls, and restarts when `deps` change. The snackbar
  passes its `Message` as the dep. Visibility is `Visible` rather than
  conditional rendering, because the widget holds that hook. The widget does
  not place itself: `Screen.Footer` or a bottom-aligned `ZStack` layer is the
  caller's choice.
- B2 `RadioGroup` took option (b), `RoleListBox` + `ListRow.Selectable`, as
  planned. **Option (a) followed:** core now carries `RoleRadioGroup` and
  `RoleRadio` as a fourth keyboard composite. A radio's state is written as
  `aria-checked`, the runtime always follows focus inside a radio group, and
  Compose maps the pair to `selectableGroup()` and `Role.RadioButton`.
  `RadioGroup` uses them. The ring is a drawn Column rather than a platform control, so the
  row is the only handler and the web double dispatch `SwitchRow` guards
  against cannot occur. The selected row passes an empty `SelectedStyle`
  because the ring already carries the selection and a Surface tint on top
  reads as a second state. It is vertical only; `SegmentedControl` is the
  horizontal form.
- B2 tripped `comps/nested_composite_test.go`, whose premise was "no widget
  declares a keyboard container role". That premise was already false:
  `BottomBar` has declared `RoleToolbar` since A4, through a variable the
  scan could not see. The guard now scans every code reference to a
  container role and allows it only in *closed* widgets, meaning no declared
  struct holds a `core.View`. `BottomBar` and `RadioGroup` are the two
  listed. Two closed widgets cannot nest by composition, so a nested
  composite still needs at least one container the caller roled by hand.
- B4 `StepIndicator` scrolls rather than collapsing, as planned. The strip is
  `RoleNavigation` only when `OnTap` is set, because a picture of progress is
  not navigation, and `RoleGroup` otherwise. The plan named `RoleNavigation`
  unconditionally. Done steps are tappable and upcoming ones never are.
  Its first version wrapped the Scroll in a zero-padding Row. The web host
  pages (`wasm/index.html`, `wasm/shots/index.html`, and the `grmob new`
  template) gave every Scroll `flex: 1 1 0; min-height: 0`, so a horizontal
  strip whose parent is a column measured 0px tall in Chrome. The same bug
  collapsed `ChipStrip{Scrollable: true}` in lesson 4.8, and it predated
  Tier B. It was then fixed in the pages rather than per widget: a horizontal
  Scroll keeps its content height, and inside a row it takes the row's width.
  With that fix the wrapper was removed.
- B5 `Timeline` is not built on `ListRow`. `ListRow`'s theme row padding sits
  outside its slots and would break the line between rows. Each event is a
  `Row` with `AlignItems` stretch and `Padding(0)`. Its rail has a fixed top
  segment that centres the dot on the first line of text, the dot, and a
  bottom segment that grows. The space below an event is padding inside the
  body, so the line runs through it. The first top and last bottom segments
  keep their size and paint nothing.

**Correction found while landing A2:** rendering the row's control
`core.Disabled(true)` (the approach sketched under A2) was rejected. Disabled
is the platform disabled state, so every target draws the control greyed and
screen readers call it dimmed. The shipped row keeps the control live and
routes both the row's tap and the control's toggle through one guarded
setter; see the `SwitchRow` doc comment for the per-target dispatch table.
**Date:** 2026-09-12
**Driver:** A survey of `comps/` against the widget set a typical mobile app
reaches for first, filtered by one question: *can it be built entirely on
primitives every renderer already draws?* Anything that needs a new node type
costs four renderers (`htmlout/export.go`, `wasm/grmob-runtime.js`,
`android/.../Renderer.kt`, `ios/GrMob/Runtime/Renderer.swift`) and is out of
scope for this plan by definition.

## What exists today (the constraints the plan works within)

`comps` has 38 widgets: `Screen` `AppBar` `Card` `Button` `InputRow`
`SearchField` `SegmentedControl` `ListRow` `GroupedList[T]` `DataTable[T]`
`Separator` `Avatar` `ProgressBar` `Chip` `ChipStrip` `Badge` `Banner`
`FormField` `Accordion` `Collapse` `CollapseBand` `Tabs` `Calendar`
`DatePicker` `EmptyState` `Skeleton` `Pagination` `LoadMore` `StatTile`
`MapPanel` `StaticMap` `CodeEditor` `RichTextEditor` `RichToolbar` `Compass`.

`core` already owns the primitives the missing widgets need, so none of the
tiers below touch a renderer:

| Primitive | Where | What it unlocks |
|---|---|---|
| `core.Modal` (`Visible`, `OnDismiss`, `Backdrop`, `ModalContent`) | `core/modal.go` | dialogs, sheets, pickers |
| `core.Switch`, `core.Checkbox`, `core.Slider`, `core.NumericInput`, `core.TextArea`, `core.Select` | `core/switch.go`, `core/input.go`, `core/slider.go` | every settings-row and form-row shape |
| `core.ZStack` + `core.StackAlign` | `core/layout.go` | badges on avatars, FAB placement, overlays |
| `core.Box` + `core.OnClick` + `core.AccessibilityRole(RoleButton)` | `core/behavioral_props.go`, `core/role.go` | any tappable surface |
| `core.StickyHeader`, `core.OnEndReached` | `core/layout.go`, `core/list.go` | already used by `GroupedList`; reusable |
| `core.Rotate`, `core.Transition` | `core/animation.go` | chevrons, spinners |
| `RoleDialog`, `RoleCheckbox`, `RoleAlert`, `RoleStatus`, `RoleNavigation`, `RoleToolbar`, `RoleListBox`/`RoleOption` | `core/role.go` | every widget below has a role it can wear honestly |
| `comps.Variant`, `comps.Emphasis`, `comps.Prominence`, `disclosure`, `headingProps` | `comps/variant.go`, `button.go`, `chip.go`, `disclosure.go`, `heading.go` | shared vocabulary; new widgets reuse rather than invent |

Two shapes are **not** low-hanging and are listed at the end so nobody
re-derives that: anything needing a new node type (a native date/time wheel,
a bottom-sheet with a drag handle that tracks the finger, a real popover
anchored to a trigger) and anything needing a scroll offset in Go.

## How the tiers are ordered

By *hand-roll count* first (how many times the shape is already written in
`examples/`, `../church/church_mobile`, or the tutorial), then by how much of
the widget is pure composition of what is in the table above. Tier A widgets
are each an afternoon; Tier B a day; Tier C is worth doing but has a design
question to settle first.

Every widget follows the discipline in `comps/doc.go`: a struct implementing
`core.View`, colours from `ctx.Theme().Colors` by role, sizes from
`Spacing`/`Typography`, a `Style []core.StyleProp` override field, and no
hooks unless the state is purely presentational (`Accordion` and `DatePicker`
are the two precedents; a widget that takes a hook is documented as one).

Each widget ships with: `comps/<name>.go`, `comps/<name>_test.go` asserting
`core.DumpConcerns() == ""` and the expected node shape, a section in
`docs/components.md`, and a tutorial lesson in `examples/tutorial` if it
introduces a new interaction (the dialog family does; a `SwitchRow` does not).

---

## Tier A — pure composition, a few hours each

### A1. `Dialog` (alert / confirm)

Every app has a "Delete this?" moment, and `core.Modal` gives only the
backdrop and the sheet. Callers hand-roll the title, message and button row
every time, and they disagree on button order and which one is destructive.

```go
comps.Dialog{
    Visible:   confirming.Get(),
    Title:     "Delete note?",
    Message:   "This cannot be undone.",
    Confirm:   comps.DialogAction{Label: "Delete", Variant: comps.VariantError, OnTap: del},
    Cancel:    comps.DialogAction{Label: "Keep", OnTap: dismiss},
    OnDismiss: dismiss,          // backdrop tap and back gesture
    Body:      nil,              // slot; wins over Message when set
}
```

- Built on `core.Modal` + `ModalContent(Column(...))` + two `comps.Button`s.
- Cancel is drawn `EmphasisGhost`, Confirm takes the caller's `Variant`.
  `VariantError` reads `Colors.Danger`, which `DangerColor` already resolves.
- Button order is a platform convention (iOS: cancel left, Android: cancel
  left too since Material 3, web: varies). Settle on **cancel left, confirm
  right** and document it; do not expose an order knob.
- `core.Modal` already writes `role="dialog"`; the widget adds
  `AccessibilityLabel(Title)` on the content column and nothing else.
- **Design question, settle up front:** a `Dialog` with no `Cancel` is an
  alert; a `Dialog` with neither is a plain sheet. Both are the same struct
  with fields left zero. Do not make three widgets.

### A2. `SwitchRow` and `CheckboxRow`

The settings-screen row: a label, optional subtitle, control on the trailing
edge, and the **whole row** tappable, not just the control. `ListRow` already
has the leading / spine / trailing geometry, so this is `ListRow` with the
trailing slot fixed and `OnTap` wired to the toggle.

```go
comps.SwitchRow{Title: "Notifications", Subtitle: "Push and email",
    On: notify.Get(), OnToggle: notify.Set}
comps.CheckboxRow{Title: "I agree to the terms", Checked: ok.Get(), OnToggle: ok.Set}
```

- Both are thin: build a `ListRow` with `Trailing: core.Switch(...)` and
  `OnTap: func(){ OnToggle(!On) }`.
- The row's tap and the control's own toggle must not double-fire. `ListRow`
  makes the whole Box tappable; the `Switch` inside also fires. Either the row
  suppresses its own `OnTap` when the tap lands on the control (not knowable
  in Go) or the control is rendered **non-interactive** (`core.Disabled(true)`
  with its label announced by the row) and the row is the only handler. The
  second is the honest one and is what `ListRow`'s role discussion already
  argues for. Test that one tap flips the state exactly once.
- `CheckboxRow` is also the form-shape `forms.Form.Checkbox` wants; add a
  `FormField`-compatible example.

### A3. `Stepper`

A number with `−` / `+` and optional bounds. `core.NumericInput` is a text
field; this is the tappable pair the counter example hand-rolls and the
tutorial repeats.

```go
comps.Stepper{Value: qty.Get(), Min: 1, Max: 20, Step: 1, OnChange: qty.Set,
    Label: "Quantity"}
```

- `Row(Button("−"), Text(value), Button("+"))` with the buttons disabled at
  the bounds. Reuse `comps.Button` with `EmphasisGhost` so the pair is quiet.
- Accessibility: the group gets `AccessibilityLabel(Label)` and
  `AccessibilityValue(fmt.Sprint(Value))`. There is no `RoleSpinbutton` in
  `core/role.go`; do **not** add one for this, `RoleGroup` plus the value is
  what both natives announce anyway.

### A4. `Toolbar` / `BottomBar`

The fixed action strip at the bottom of a screen: two to five icon-or-label
buttons, evenly spread, one optionally selected. `SegmentedControl` is close
but is a *choice*, not a set of *actions*, and its selected styling is wrong
for a bar.

```go
comps.BottomBar{
    Items: []comps.BarItem{{Label: "Home", OnTap: home}, {Label: "Search", OnTap: find},
                           {Label: "Me", OnTap: profile}},
    Selected: 0,
}
```

- `Row` with `Justify(JustifyAround)`, each item a `comps.Button` ghost
  emphasis, the selected one in `Colors.Primary`.
- `AccessibilityRole(RoleNavigation)` on the bar when `Selected` is set (it is
  a tab-like navigator), `RoleToolbar` when it is not (it is an action strip).
  Both roles exist. The role choice is the one design decision; make it
  follow whether `Selected >= 0`.
- Pairs with `Screen`: add `Screen.Footer core.View` as a pinned slot outside
  the scrolling region, since a bar inside `Scroll` scrolls away. This is the
  only change to an existing widget in Tier A and it is additive.

### A5. `Spinner` / `ActivityIndicator`

`Skeleton` covers the "content is coming" case for known layouts;
`ProgressBar` covers determinate progress. The third case, "something is
happening, shape unknown", is a spinner and every app draws one.

- `core.Rotate` exists but `core.Transition` is a one-shot duration and easing
  (`core/animation.go`), not a loop, so there is no declarative spin today. The
  v1 is a `Box` border ring plus `hooks.UseInterval` stepping `Rotate` by 30°
  every 80 ms, which makes this the third hook-holding widget and must be
  documented as one. A looping transition prop would be a four-renderer change
  and is noted as a follow-up, not a prerequisite.
- `AccessibilityRole(RoleStatus)` + `AccessibilityLabel("Loading")` so it is
  announced once and not re-announced every frame.
- `Size` field with `Small`/`Medium`/`Large` reading `Spacing`.

### A6. `Rating`

Five tappable stars (or any glyph, any count), read-only or interactive.
Trivial `Row` of `Text("★")`/`Text("☆")` in `Colors.Warning` (or `Primary`)
with `OnClick` per glyph.

- `Value float64` so half-stars can render later without a signature change;
  v1 rounds.
- `ReadOnly bool` drops the handlers and the `RoleButton`s.
- `AccessibilityValue("3 of 5")` on the group.

---

## Tier B — composition plus one decision, a day each

### B1. `ActionSheet` (bottom choice list)

The "Share / Copy link / Delete" sheet that slides from the bottom.
`core.Modal` gives the backdrop; the widget is a `Column` of full-width
`comps.Button` ghosts pinned to the bottom with `StackAlign`.

```go
comps.ActionSheet{
    Visible: open.Get(), Title: "Note",
    Actions: []comps.SheetAction{{Label: "Share", OnTap: share},
                                 {Label: "Delete", Variant: comps.VariantError, OnTap: del}},
    Cancel:  "Cancel", OnDismiss: close,
}
```

- **Decision:** does `core.Modal` let content be placed at the bottom edge,
  or does it centre? Read `ModalNode` in `core/modal.go` and the four
  renderers' modal branch. If it centres unconditionally, the sheet needs a
  `core.ZStack` inside the modal content sized to the screen, which is fine
  on web and needs checking on SwiftUI (a `ZStack` inside a sheet may not fill).
  This is the single unknown and is why the widget is Tier B.
- No drag-to-dismiss; that needs a gesture node and is out of scope.
- Roles: the sheet is `RoleDialog` (Modal supplies it); the action list is
  `RoleListBox` with `RoleOption` items, which `core/role.go` already
  documents as the pair for "a list of choices".

### B2. `RadioGroup`

A vertical or horizontal set of mutually exclusive options. `Select` is the
compact form; a radio group is the form for two to five options a user should
see at once. `SegmentedControl` is the horizontal case for short labels; this
is the vertical case with subtitles.

```go
comps.RadioGroup{Options: []comps.RadioOption{{Value: "std", Label: "Standard", Subtitle: "3–5 days"},
                                              {Value: "exp", Label: "Express"}},
    Value: ship.Get(), OnChange: ship.Set}
```

- Each option is a `ListRow` with a leading ring glyph (a `Box` with
  `BorderRadius` half its width, filled when selected) and `OnTap`.
- **Decision:** there is no `RoleRadio`/`RoleRadioGroup` in `core/role.go`.
  Two honest choices: (a) add the pair to `core/role.go` (an additive change
  to `core`, plus the two web exporters' role table; the natives map roles to
  traits and would need the trait added too, which is three files, not four
  renderers, since `htmlout` and the runtime share `Roles()`), or (b) use
  `RoleListBox`/`RoleOption` with `AccessibilitySelected`, which announces
  correctly today. Ship (b), note (a) as a follow-up. Do not block on it.

### B3. `Snackbar` (toast with an action)

`core.ShowToast` is fire-and-forget from Go and is drawn by the host. What it
cannot do is carry an **Undo** button. A `Snackbar` is a widget the caller
renders at the bottom of a `ZStack` for a few seconds.

```go
comps.Snackbar{Visible: undo.Get() != nil, Message: "Note deleted",
    Action: "Undo", OnAction: restore, OnTimeout: clear, Duration: 4*time.Second}
```

- Holds a `hooks.UseTimeout` keyed on `Visible`, so it is a hook-holding
  widget and is documented as one. The timeout calls `OnTimeout`; the caller
  owns visibility, exactly as with `Dialog`.
- `RoleStatus` (polite), not `RoleAlert`, unless `Variant` is `VariantError`.
- **Decision:** whether to keep this or extend `core.ShowToast` with an
  action callback. The latter is a host change on four renderers, so the
  widget wins for this plan; note the core extension as the eventual right
  shape.

### B4. `Breadcrumb` / `StepIndicator`

The "Step 2 of 4" header for a multi-screen flow (signup example has one
hand-rolled). A `Row` of numbered circles joined by `Separator`s, the current
one filled with `Primary`, the done ones with `Success`.

- `Steps []string`, `Current int`, optional `OnTap(i)` for done steps only.
- `RoleNavigation` with `AccessibilityLabel("Step 2 of 4: Address")`.
- Trivial; Tier B only because the horizontal overflow past ~5 steps needs a
  choice (`core.Horizontal` scroll, or collapse to "2 / 4"). Ship scroll.

### B5. `Timeline`

A vertical list of events with a line down the leading edge and a dot per
row. Chat, order-tracking and activity feeds all draw it. `GroupedList` gives
the band structure; the widget is `ListRow` with a leading `Column(dot, line)`
whose line is `FlexGrow(1)`.

- The line between rows is the hard part: each row draws its own segment
  above and below the dot, first and last rows omit one. A `ListRow` slot for
  leading content that stretches the row's full height may need
  `AlignSelf(Stretch)` on the leading column; check `ListRow` does not pin
  `AlignItems` to centre. If it does, that is the one `ListRow` change.

---

## Tier C — worth it, but settle the shape first

### C1. `Menu` / `Dropdown` (tap a trigger, pick from a list)

Overflow menus and "sort by" pickers. There is no anchored popover primitive,
so on every target this is a `Modal` with a list, which is a **bottom sheet**
on mobile and a **centred dialog** on web. Ship it as `ActionSheet` with a
trigger slot and call it done; a real anchored popover is a node type.

### C2. `Drawer` (side navigation)

A `ZStack` with the backdrop and a left-pinned `Column`. No slide gesture; a
hamburger button toggles it. Feasible now; the question is whether it earns
its place before `BottomBar` (A4), which covers the same navigation need with
no overlay. Recommendation: **skip until asked for**, and record here why.

### C3. `SearchableSelect` / `Autocomplete`

`SearchField` + a filtered `List` under it. `hooks.UseDebounce` exists. The
shape is easy; the focus and keyboard behaviour (does the list steal focus,
does `FocusNext` skip it) is the work, and `core/focus_order.go` is the
place to read before starting.

### C4. `Carousel` / horizontal pager

`core.Horizontal` scroll exists, but there is no scroll offset in Go, so
"which page is showing" cannot be known and the dots cannot follow. Blocked
on a host signal (`OnScroll` or a paged-scroll node). Not low-hanging.

---

## Not low-hanging (needs a renderer), listed so it is not re-derived

- Native time picker / date wheel: node type.
- Drag-to-dismiss sheet, swipe-to-delete row, pull-to-refresh: gestures.
- Anchored popover / tooltip: needs layout measurement the hosts do not send.
- Toast with action drawn by the host: extends `core.ShowToast` on four renderers.
- Any widget that needs the scroll offset (carousel dots, parallax header).

## Suggested order

| Order | Widget | Why first |
|---|---|---|
| 1 | A1 `Dialog` | highest hand-roll count; unlocks confirm flows in every example |
| 2 | A2 `SwitchRow` / `CheckboxRow` | settings screens; trivial on `ListRow` |
| 3 | A4 `BottomBar` + `Screen.Footer` | the one structural gap in `Screen` |
| 4 | A5 `Spinner` | completes the loading trio with `Skeleton` and `ProgressBar` |
| 5 | A3 `Stepper`, A6 `Rating` | small, self-contained |
| 6 | B1 `ActionSheet` | reuses A1's modal plumbing once the placement question is answered |
| 7 | B2 `RadioGroup`, B3 `Snackbar`, B4 `StepIndicator`, B5 `Timeline` | in any order |

Land A1–A6 as one PR with one tutorial lesson ("Dialogs and settings rows"),
or as six small commits in the style of the A2 bundle in
`components-datatable-compass-map.md`. Tier B is one widget per commit, each
with the decision it settled recorded in the widget's doc comment.

## Definition of done, per widget

- `comps/<name>.go` with a doc comment that says what the widget settles (the
  `ListRow` comment is the model) and which theme roles it reads.
- `comps/<name>_test.go`: renders under `core.SetDebugMode(true)`, asserts an
  empty concern list, asserts the accessibility role and label, and for
  interactive widgets dispatches the callback and asserts exactly one state
  change.
- A `## <Name>` section in `docs/components.md` and the generated
  `docs/api/comps.md` regenerated with `go run ./internal/apidoc/gen`.
- The widget appears in a tutorial lesson so it is exercised on all four
  targets. (`examples/components.go` is a small input demo, not a gallery.)
- No file under `htmlout/`, `wasm/`, `android/` or `ios/` changed. If one had
  to be, the widget was not low-hanging and belongs in a different plan.
