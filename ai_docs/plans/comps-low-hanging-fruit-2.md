# Low-hanging fruit for `comps`, round two

**Status:** drafted 2026-09-17. D1 `FAB` + `Screen.Floating` started the same
day. Everything else is unstarted.

The first round (`comps-low-hanging-fruit.md`) landed entire on 2026-09-12:
Tiers A through C, fifteen widgets, one carousel left blocked on a scroll
offset. This file is the next crop, chosen the same way — **pure composition
over primitives every one of the four targets already renders, no file under
`htmlout/`, `wasm/`, `android/` or `ios/` touched.** Where one of these turns
out to need a renderer, it comes out of this plan and into one of its own.

What changed since the first round that makes some of these newly cheap:

- `core.ZStack` grew `core.StackAlign`, the per-layer nine-way placement,
  honoured on all four targets (`core/stack_align.go`). Anything that floats
  over content became a stack with an aligned layer.
- `core.Canvas` with `Rect`, `Circle`, `Sector`, paths and gradients is on
  every target, so anything that is "a picture computed in Go" is cheap.
- `hooks.UseNow`, `UseIntervalWhile` and `UseTimeoutWhile` exist, so time-driven
  widgets no longer re-render the app on a bare `UseInterval`.
- `core.FocusNext` / `FocusPrevious` and `FocusRef` exist, so a widget can move
  focus between its own inputs.
- `Calendar` carries `Min`, `Max`, `Marked` and `Today`, so a range is one more
  field rather than a new grid.

---

## Tier D — a few hours each

### D1. `FAB` and `Screen.Floating` — **in progress**

The floating action button: a circular, elevated, filled button pinned to the
bottom-end corner of a screen, over the content rather than below it. The
single most common Android affordance and one that `BottomBar` (below the
content) does not cover.

Two halves, because a floating thing needs a place to float:

- `comps.FAB{Icon, Label, OnTap, Variant, Size, Disabled, AccessibilityLabel,
  Style, FocusRef}`. A `comps.Button` in a circle: fixed width and height from
  the spacing scale, `BorderRadius` half of that, zero padding, `core.Shadow`.
  With `Label` set it is the *extended* form — a pill with the glyph and the
  word, the shape Material uses when there is room to name the action.
  Icon-only needs `AccessibilityLabel`, because a "+" is not a name.
- `Screen.Floating core.View`: a new slot. When set, the screen's content
  (the column, or the scroll around it) becomes the base layer of a
  `core.ZStack` that grows to fill the safe area, and `Floating` is the top
  layer placed `StackAlignBottomEnd` with a `Spacing.LG` margin. `Footer`
  stays below the stack, so a FAB floats above a `BottomBar` rather than on
  it. Nil keeps the tree byte-identical to today, which is the invariant
  `TestScreenZeroValueIsExactlySafeAreaColumn` pins.

**The decision this settles:** the slot lives on `Screen`, not the FAB. A
ZStack sizes to its largest layer and centres every layer that says nothing,
so a FAB that wrapped its own screen would have to know the screen's size; a
screen that hosts the layer already knows it fills the safe area. The base
layer states `Width("100%")` and `Height("100%")` — percentages resolve on all
four targets (`GrMobStyle.kt` maps them to `fillMaxWidth`/`fillMaxHeight`,
SwiftUI resolves them against the proposal, the DOM against the grid track).

**What is not verified:** neither native has been run on a device with the
stack in a screen. `ios/verify` measures a stack's size and its anchors, and
`mobile/verify` pins that the Compose arm overlays, so the two facts this
depends on are pinned separately; the composition of them on a full-height
screen is not. If a device run shows the base layer hugging its content, the
fix is in the base layer's props, not the renderers.

Content under a FAB: the last row of a scrolling list can end under the
button. The widget does not pad the content for it, because it cannot know
the list's own inset; the doc says to give the last item room.

### D2. `QRCode`

A `core.Canvas` of `Rect` modules. The roadmap wants a `cats://pair` deep link
handed over as a QR, and nothing in the module tree encodes one.

- `QRCode{Data string, Size float64, Level ECLevel, Quiet int, Label string,
  Style}`. The canvas is `CanvasFit` in a `Size × Size` box like `Gauge`.
- Dark modules in the theme's text colour on the surface colour, never
  inverted: readers assume dark-on-light.
- `RoleImage` with `Label` (default "QR code"), the data never spoken.
- **Decision to settle:** vendor an encoder (`github.com/skip2/go-qrcode`
  gives a bitmap and is small) or write one. Vendoring is a day cheaper and
  the licence is MIT; write one only if the dependency policy says so.

### D3. `Countdown` and `Stopwatch`

`hooks.UseNow` at one tick a second while running, formatted the way
`DigitalClock` formats, with `OnDone` for a countdown. Alarm rows and timers
both want it.

- `Countdown{Until time.Time, OnDone func(), Format, Label, Style}`;
  `Stopwatch{Since time.Time, Running bool, ...}`.
- The widget holds a hook, so it is *not* conditional-safe; a `Hidden` field
  as `Spinner` has, and the tick pauses while hidden or done.
- **Decision:** whether `OnDone` fires from the render pass (it must not — a
  render is not the place to run a handler) or from the hook's effect. The
  hook's effect, once, on the tick that crosses zero.

### D4. `SelectRow` and `SliderRow`

The rest of the settings-row family beside `SwitchRow` and `CheckboxRow`.

- `SelectRow`: a `ListRow` whose trailing text is the current choice; tapping
  opens an `ActionSheet` of `SheetAction`s with `Checked` on the current one.
  The row owns the one open/shut state, as `DatePicker` does.
- `SliderRow`: a `ListRow` over a `core.Slider`, reporting through
  `OnSliderChangeEnd` so a drag does not re-render the app per pixel.
- No decision; both are the shapes already in the package.

### D5. `PINInput`

N single-character `core.Input`s in a row with `FocusNext` on each change, for
one-time codes. The signup example wants it.

- `PINInput{Length int, Value string, OnChange func(string), OnComplete
  func(string), Secure bool, Label, Style}`.
- **Decisions:** a pasted full code lands in the first cell's `OnChange` as a
  multi-character string — spread it across the cells and jump to the end.
  Backspace on an empty cell cannot be seen (no key events), so a cleared cell
  stays focused; document it.

### D6. `DateRangePicker`

`Calendar` gains `RangeStart`, `RangeEnd time.Time` and draws the fill between
them; the picker is `DatePicker`'s sheet with the two-tap protocol.

- **Decision:** the third tap, with a range already set, starts a new range at
  that day rather than moving the nearer end. Simpler to state and to test.

### D7. `TimePicker`

Hour and minute `Stepper`s, or two `Select`s, in the sheet `DatePicker` uses.
Not a native wheel: that is a node type and stays out.

- `TimePicker{Value time.Time, OnChange, TwentyFourHour bool, MinuteStep int,
  Label, Style}`; `TwentyFourHour` matches `DigitalClock`.

## Tier E — an hour each, no decisions

- `KeyValueList{Rows []KeyValue}` for order details, profiles and about
  screens; a `Column` of `ListRow`s with the value trailing.
- `Breadcrumb{Items []string, OnTap func(i int)}` — ghost buttons with `›`
  between, `RoleNavigation`; the last item is current and not a button.
- `AvatarStack{Avatars []Avatar, Max int}` — a `ZStack`? No: a `Row` with a
  negative-looking overlap is not portable. A `Row` of `Avatar`s each in a
  `ZStack` layer placed `StackAlignStart` with an increasing left margin, and
  a `+N` disc last.
- `LabeledSeparator{Label string}` — the "or" line on signup screens.
- `PasswordField` — `FormField` whose input swaps between `core.Input` and
  `core.InputPassword` on a reveal toggle. The toggle is a ghost `Button`
  trailing the field, `aria-pressed` through `core.AccessibilitySelected`.
- `BarItem.Badge string` on `BottomBar` — a `Badge` placed `StackAlignTopEnd`
  over the icon, which is now a two-layer `ZStack`.
- `Lightbox{Src string, Open bool, OnDismiss}` — a `Modal` around an
  `ImageWithMode` fit, reusing `Dialog`'s scrim and dismiss plumbing.

## Tier F — worth it, with a catch to settle first

- `Heatmap` (calendar heatmap, matrix heatmap): a `Canvas` of `Rect`s, cheap;
  but it needs a *sequential* colour scale and the palette defines only the
  categorical `ChartColors`. Add `ColorPalette.SequentialColors()` first, or
  interpolate from `Primary` to the surface. That is a theme decision.
- `Histogram`: `BarChart` with the binning done in Go; the catch is only that
  the bar chart's categories are strings and a bin wants a numeric axis.
- `RichTextView`, a read-only renderer for a `richtext.Doc`: blocks map to
  `Text`, headings, lists and code; **inline marks do not**, because core has
  no inline span node — a bold word inside a paragraph would flatten on both
  natives. The web could use `Doc.HTML()`. Half a widget until spans exist.

## Still blocked on a renderer, so it is not re-derived

Carousel and pager dots (scroll offset), pull-to-refresh and swipe actions
(gestures), tooltip and anchored popover (layout measurement), native time
wheel (node type), a clipboard copy button (no clipboard bridge).

## Suggested order

| Order | Widget | Why |
|---|---|---|
| 1 | D1 `FAB` + `Screen.Floating` | StackAlign just made it possible; the most-asked-for Android shape |
| 2 | D2 `QRCode` | the roadmap's pairing flow has nothing to show |
| 3 | D4 `SelectRow`, `SliderRow` | completes a family, zero decisions |
| 4 | D3 `Countdown` | alarms exist and cannot show time left |
| 5 | D5 `PINInput`, D6 `DateRangePicker`, D7 `TimePicker` | in any order |
| 6 | Tier E as one bundle with one lesson | |
| 7 | Tier F once its theme or core prerequisite lands | |

## Definition of done, per widget

Unchanged from round one:

- `comps/<name>.go` with a doc comment that says what the widget settles and
  which theme roles it reads.
- `comps/<name>_test.go`: renders under `core.SetDebugMode(true)`, asserts an
  empty concern list, asserts the role and label, and for interactive widgets
  dispatches the callback and asserts exactly one state change.
- A section in `docs/components.md`; `docs/api/comps.md` regenerated with
  `go run ./internal/apidoc/gen`.
- A tutorial lesson, appended at the end of its chapter so deep-linked lesson
  numbers do not move.
- No file under `htmlout/`, `wasm/`, `android/` or `ios/` changed.
