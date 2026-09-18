# Low-hanging fruit for `comps`, round two

**Status:** drafted 2026-09-17. D1 `FAB` + `Screen.Floating`, D2 `QRCode`,
D4 `SelectRow` + `SliderRow`, D3 `Countdown` + `Stopwatch`, D5 `PINInput` and
D6 `DateRangePicker` all landed the same day. Everything else is unstarted.

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

### D1. `FAB` and `Screen.Floating` — **landed 2026-09-17**

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

### D2. `QRCode` — **landed 2026-09-17**

A `core.Canvas` of `Rect` modules. The roadmap wants a `cats://pair` deep link
handed over as a QR, and nothing in the module tree encodes one.

- `QRCode{Data string, Size float64, Level ECLevel, Quiet int, Label string,
  Style}`. The canvas is `CanvasFit` in a `Size × Size` box like `Gauge`.
- Dark modules in the theme's text colour on the surface colour, never
  inverted: readers assume dark-on-light.
- `RoleImage` with `Label` (default "QR code"), the data never spoken.
- **Decision settled: write one**, in `internal/qr`. `go.mod` holds no runtime
  dependency outside this author's own packages and the gomobile toolchain, so
  `skip2/go-qrcode` — MIT and correct — would have been the first third-party
  package a widget here required, and it carries an `image/png` bitmap model a
  `Canvas` only has to undo. A QR encoder is a closed algorithm with published
  test vectors: byte mode, versions 1–40, four levels, eight masks with the
  standard penalty scoring, about 700 lines.

- **What shipped beyond the sketch.** Every dark module goes into *one* filled
  path with runs merged along each row, not a `Shape` per module: three
  thousand children is the reconciler's worst case, and separate shapes are
  antialiased against each other into hairlines a binarizer reads as light.
  Colour resolved as "the theme's ink and surface only when those are already
  dark-on-light with room to spare (0.6 luminance and 0.15), else black on
  white" — the sketch's "theme colours, never inverted" cannot hold in a dark
  theme, and an unscannable code fails silently. Data past a version-40 symbol
  keeps the box, draws nothing and reports `comps.ConcernQRDataTooLong` in
  debug builds.

- **How it is checked.** Against the standard's own printed format and version
  bit strings, its published byte capacities and alignment coordinates, the
  defining Reed-Solomon property (every codeword vanishes at the generator's
  roots), and a round trip back out of the finished grid for all forty
  versions. Then through Chrome's `BarcodeDetector` on the real rendered SVG,
  at all four levels, for a 50-byte and a 313-byte payload — the second past
  both the 16-bit character-count field and the version-information block.

### D3. `Countdown` and `Stopwatch` — **landed 2026-09-17**

One tick a second while running, formatted the way a phone timer formats, with
`OnDone` for a countdown. Alarm rows and timers both want it.

- `Countdown{Until time.Time, OnDone func(), Format, Size, Color, Hidden,
  AccessibilityLabel, Style}`; `Stopwatch{Since time.Time, Elapsed
  time.Duration, Running bool, ...}`.
- The widget holds a hook, so it is *not* conditional-safe; a `Hidden` field
  as `Spinner` has, and the tick pauses while hidden or done.
- **Decision settled: the hook's effect**, keyed on whether the deadline has
  passed, so `OnDone` fires once per *crossing* — which also makes moving
  `Until` forward the whole of a restart, and makes an already-past `Until`
  fire on the first pass.

**What the sketch left for the build.** Four things.

- **Not `hooks.UseNow`, and no `UseNowWhile` either.** `UseNow` aligns to the
  wall clock so a `DigitalClock` changes its digit when the status bar does; a
  countdown's own boundaries fall at `Until` minus a whole number of seconds,
  which nothing else on the screen is on. There being nothing to align to, the
  reason for the more expensive hook evaporates — and `UseNow` cannot be
  paused, which "pauses while hidden or done" needs. The tick is therefore
  `hooks.UseIntervalWhile` with an *empty* callback: both widgets read the
  clock in their own `Render`, so a tick's whole job is to bring the render
  back. Adding a pausable `UseNow` to `hooks` was considered and dropped for
  the same reason.
- **The clock is read before the hooks, not by them.** Whether to keep ticking
  and whether the deadline has passed are both answers about the same instant,
  and a hook cannot both report an instant and be told, in the same call, what
  it means.
- **"Pauses while hidden" needed one exception.** Hiding a countdown removes
  the reason to draw but not the reason to count: somebody is still waiting to
  be told it ran out. So a hidden countdown that owes an `OnDone` keeps
  ticking and one that owes nothing stops dead — a four-row table in the type
  doc, pinned by a table test over `ticking`. `Stopwatch` has no exception,
  because it owes nobody a callback.
- **`Stopwatch` needed a second field the sketch did not have.** `Since` alone
  cannot be paused — the instant the finger lifts is recorded nowhere — so the
  state is the pair every stopwatch keeps, `Elapsed` banked plus the current
  run, and the four moves are assignments the caller writes inline. Held in
  the widget it would be state the app cannot save or restore, which is
  `SliderRow`'s argument again.

Two roundings, both conservative and opposite: the countdown rounds *up* so it
never says you have less time than you do, the stopwatch truncates so it never
claims more elapsed time than has passed. Neither shows hundredths, for the
render-per-tick reason. A zero `Until`, and a `Running` stopwatch with no
`Since`, report `ConcernCountdownUntilUnset` and `ConcernStopwatchSinceUnset`
in debug builds: both are permanently-wrong states that look on screen exactly
like ordinary ones.

### D4. `SelectRow` and `SliderRow` — **landed 2026-09-17**

The rest of the settings-row family beside `SwitchRow` and `CheckboxRow`.

- `SelectRow`: a `ListRow` whose trailing text is the current choice; tapping
  opens an `ActionSheet` of `SheetAction`s with `Checked` on the current one.
  The row owns the one open/shut state, as `DatePicker` does.
- `SliderRow`: a `ListRow` over a `core.Slider`, reporting through
  `OnSliderChangeEnd` so a drag does not re-render the app per pixel.
- No decision; both are the shapes already in the package.

**What the sketch left for the build.** Three things, and the first two are
the same question asked of each widget: *who holds the state?*

- **`SelectRow` holds one piece and `SliderRow` holds none**, and the pair is
  now the clearest statement of the trade in the package. The sketch's "the
  row owns the open/shut state" was right and carries `DatePicker`'s hook
  obligation with it. The slider's tempting mirror — hold the in-flight drag
  value so the reading follows the finger — was rejected: `State.Set` requests
  a render of the *whole tree*, so a widget holding the draft would charge
  every `SliderRow` the render-per-tick this item exists to avoid, to make one
  of them look livelier. `OnDrag` hands that choice to the caller, who holds
  the draft. Lesson 6.8's demo does exactly that, and its driving test asserts
  the two numbers disagreeing mid-drag.
- **The row is the control, or it isn't.** `SelectRow` takes `RoleButton` and
  `PopupDialog` and is the whole tap target, because there is nothing else in
  the row to be the control — the sketch's "not `core.Select` in the trailing
  slot" needed the reason, which is that on the web a click on a control
  bubbles to the row, and two *openings* have nothing to converge on the way
  `SwitchRow`'s two setters do. `SliderRow` is the opposite: no `OnTap`, no
  role, because a tap on the row has no value it could honestly mean.
- **Where the track goes.** Title and reading on one line, track under them,
  and both inside the row's *growing middle column* rather than beside it —
  that is what aligns the track with the title whatever the leading icon's
  width, with no arithmetic against the theme's row padding. It costs one
  `Box`, since `ListRow.Content` is a single view and there are two things to
  put in it.

`core.SelectOption.Group` is the one field that does not survive the trip: a
sheet action is a button with one line and no section construct. Documented as
a limit pointing at `core.Select` and `SearchableSelect`, which have each
already answered it. A `Value` no option carries reports
`comps.ConcernSelectRowValueNotAnOption` in debug builds, because on screen
that mistake is indistinguishable from a row nobody has set.

### D5. `PINInput` — **landed 2026-09-17**

N single-character `core.Input`s in a row with `FocusNext` on each change, for
one-time codes. The signup example wants it.

- `PINInput{Length int, Value string, OnChange func(string), OnComplete
  func(string), Secure bool, Label, Style}` — shipped exactly as sketched, no
  field added or dropped.
- **Decisions:** a pasted full code lands in the first cell's `OnChange` as a
  multi-character string — spread it across the cells and jump to the end.
  Backspace on an empty cell cannot be seen (no key events), so a cleared cell
  stays focused; document it.

**What the sketch left for the build.** Four things.

- **`Value` is one string, so the cells are a prefix.** Cell *i* draws the
  *i*-th character, and a plain string cannot hold a gap — which makes typing
  and clearing the only two edits there are, and forces the answer to the case
  the sketch did not ask about: a *cleared* cell must either shift the tail
  left, changing cells the finger never touched, or drop it. Dropping is the
  predictable one, because it is what "start again from here" means. The same
  invariant answers a character typed into a cell past the end of the code —
  it lands at the end, there being no position for it to occupy.
- **The paste rule is also the overflow rule**, which is what made it cheap. A
  cell reporting more than one character is either a pasted code or a keystroke
  in a box that already held something: the field is controlled, so it reports
  its whole contents either way, and one rule covers both — write the incoming
  string from that cell forward, cursor after the last cell filled. The one
  case it reads wrongly is a character inserted *before* an existing one,
  which arrives as `"21"`; nothing in the event says where the caret was.
- **`OnComplete` fires on every edit that leaves the code full**, and this is
  the decision that is deliberately *not* `Countdown.OnDone`'s. Once per
  crossing would leave a corrected digit unsubmitted, and `OnComplete` means
  "submit this". It comes from the change handler rather than from an effect,
  so a screen restored with a complete code does not resubmit itself on mount;
  an edit producing the value already held is treated as an echo and does
  nothing at all.
- **The hook count follows the high-water `Length`, not the current one.** One
  `FocusRef` per cell means one hook slot per cell, and a `Length` that shrank
  between passes would retire slots from the middle of the sequence and drift
  every cursor after it. The count therefore only ever grows, held in one slot
  of its own ahead of the refs, at the cost of a few refs nothing points at.
  `TestPINInputKeepsItsHookSlotsWhenLengthShrinks` pins it with a state
  allocated *after* the widget, which is where the drift would land.

The cells divide the row with `FlexGrow` and a zero `FlexBasis` — `Calendar`'s
day-cell pair, the one that makes the natives' "equal shares" and CSS's "equal
shares of the leftovers" agree. They take the platform's **text** keyboard and
not its number pad: the numeric keyboard is chosen by node type, and that node
carries an `int`, which has no way to say "this cell is empty" — so backspace
would stop working. A digits-only keyboard needs a keyboard-type prop on
`core.Input`, which is a renderer change and stays out of this plan.

Two concerns, both the "permanently wrong and looks ordinary" bar:
`ConcernPINInputInert` for a field with no `OnChange`, and
`ConcernPINValueTooLong` for characters past the last cell, which are never
drawn and can never be typed away.

### D6. `DateRangePicker` — **landed 2026-09-17**

`Calendar` gains `RangeStart`, `RangeEnd time.Time` and draws the fill between
them; the picker is `DatePicker`'s sheet with the two-tap protocol.

- **Decision:** the third tap, with a range already set, starts a new range at
  that day rather than moving the nearer end. Simpler to state and to test.
- `DateRangePicker{Start, End, OnChange func(start, end time.Time), OnClear,
  Placeholder, Format, Separator, Calendar, Title, ClearLabel, CloseLabel,
  Disabled, AccessibilityLabel, AccessibilityHint, Style}`.

**What the sketch left for the build.** Four things.

- **The decision turned out to be a *consequence*, not a rule.** The widget
  holds one piece of state — a pending start — and the whole protocol is one
  line over it: no pending start and this tap becomes it, a pending start and
  this tap is the other end. The third tap starts a new range because
  completing one clears the pending start; a day tapped twice is a one-day
  range because the second tap is the other end wherever it falls. Neither
  needed a case of its own, and there is no "nearer end" arithmetic anywhere.
- **The second tap may be the earlier one, and the pair is ordered rather than
  restarted.** "The other end" is not a claim about which end, and a restart
  would throw away a tap the reader made on purpose — the third tap is already
  the way to start over. `OnChange` therefore promises start ≤ end.
- **The pending start is the widget's, which is the opposite of `SliderRow`'s
  call and the same test.** Hold state in the widget only when no application
  wants it: a form's field is a span or it is nothing, and "from the 14th, no
  end yet" is a value it would have to invent a way to hold and to draw. The
  decisive half is that a caller holding it would make the backdrop, the ✕ and
  the back gesture *destructive* — the first tap would already have overwritten
  the range the reader opened the sheet to check. So this is the package's
  third hook-owning widget, with three states to `DatePicker`'s two, and every
  way out of the sheet discards the pending one.
- **The band is a third fill, and the corner radius is what makes it a band.**
  Endpoints wear the selected day's fill — one "this day is chosen" look in the
  grid, so `Selected` and an endpoint are deliberately indistinguishable — and
  the interior wears `Primary` at 20% with `BorderRadius(0)`, which is the
  whole of what stops a run of cells reading as a row of pills. The rounded
  endpoint meets the square band with a notch, because `core` has one radius
  and not four and the alternative was a second `Box` in all 42 cells;
  Material draws the same notch. Today's ring came out of the fill switch so it
  survives inside the band, and the band runs through the adjacent days rather
  than stopping at the 1st.

Two more things the build settled. The range joins `Calendar`'s **anchor
chain** ahead of `Today`, which is what lets the picker open on the span while
clearing `Selected` and holding `Month` at zero. And the ends of a range are
the one thing this calendar announces as a **name suffix** rather than as a
state: `AccessibilitySelected` goes on every day of the band (ARIA's own
date-range grid), and no target has a property that could tell the two ends
from the days between, so the alternative is fourteen identically named
selected days with no findable edge.

Two concerns: `ConcernDateRangePickerInert` for a picker with no `OnChange`,
and `ConcernCalendarRangeReversed` on `Calendar` — a `RangeEnd` before its
`RangeStart` has nothing between them, so the grid draws two lone endpoints,
which looks exactly like two days picked out. `rangeBand` also refuses to thin
a `Primary` it cannot parse and leaves the span unfilled rather than painting
an opaque brand colour over the day numbers, with a test that the three bundled
themes are not in that case.

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
| 4 | D3 `Countdown`, `Stopwatch` | alarms exist and cannot show time left |
| 5 | ~~D5 `PINInput`~~, ~~D6 `DateRangePicker`~~, D7 `TimePicker` | in any order |
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
