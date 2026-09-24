# Tutorial overflow sweep: web floors, a nested Scroll, the theme's stack padding

Session: `9c8e5f40-0554-4476-abbd-83e58eba2c76`

## Ask

Lesson 1.1 of the interactive tutorial overflowed in the browser's split view
(a screenshot showed "Following" cut off past the profile card, and a
squeezed "GM" avatar). Fix it, commit, then check and fix the rest of the
tutorial for overflow.

## Lesson 1.1 and the split panes (commit `52ef3ac`)

- **Cause:** `core.Row` and `core.Column` start from the theme's
  `Components.Row` (8/16) and `Components.Column` (12/16) padding. Nested in
  the demo's Card, the stats row spent 128px on padding against a card about
  230px wide.
- `helloHeader`/`helloStats` state `Padding(0)`, and the stats row wraps.
- **The split layout had the same leak.** The pane containers carried the
  theme padding as inline style, which beats `wasm/index.html`'s stylesheet:
  - the bezel was 16px, not the page's 12px;
  - the glass was 368px, not the 376px the toast rule assumes;
  - demos had 336px of width;
  - the phone's LIVE header floated off the glass's edges.
- `split.go`'s `paneStack` (`core.Padding(0)`) writes no declaration, since an
  all-zero EdgeInsets is omitted, so the page's CSS sizes the panes.

## The sweep

A scratch CDP probe opened every lesson through its hash deep link (all 80)
and checked for two things:

- **Escape:** a box outside its non-clipping parent, off the screen, or text
  wider than its box.
- **Squeeze:** a box with an inline px Width or Height drawn smaller than
  that.

**Where it ran:**

- split view (phone and guide panes);
- phone layout at 390 and at 360, via `Emulation.setDeviceMetricsOverride`.
  Headless Chrome will not size a window below 500.

**Skipped:** code editors (they scroll sideways by design) and rotated layers
(the compass rose and the clock's hands; their bounding boxes grow).

The harness was a copy of `wasm/shots/shot.mjs` with a longer CDP timeout,
serving a copy of `wasm/index.html` that sets `__grmobReady`. It is not in the
repo: see N-081.

## What it found, and the fixes

### Web-only floors (runtime and htmlout, kept in step)

- **Long words.** `comps.ConcernSelectRowValueNotAnOption` in 6.8's prose and
  4.11's requested URL spilled 18–20px past their column. Both natives break
  a word that cannot fit a line.
  - The runtime's `mount` sets `overflow-wrap: break-word` on the mount point.
  - htmlout puts `bodyStyle` on `<body>`.
  - `break-word` rather than `anywhere`, so min-content sizing is unchanged.
- **Text fields.** An `<input>`'s ~20-character intrinsic width is a flex
  floor. InputRow's Send (4.31, 4.33) and PasswordField's Show (4.27) ran
  12–25px past a phone-width row.
  - `fieldFloorTypes` (htmlout/tag.go) and the runtime's `FIELD_FLOOR_TYPES`
    cover Input, InputPassword, NumericInput and TextArea.
  - They write `min-width: 0` unless a MinWidth is stated.
  - `TestRuntimeFieldFloorTypesMatchGo` and `TestFieldFloorTypesAreTextFields`
    are in `wasm/verify/border_test.go`.
  - An InputRow-only `MinWidth("0")` was tried first and reverted in favour of
    this.
- **Slider margin.** Chrome's 2px margin on a range input made
  SliderRow's `Width("100%")` track 4px too wide (4.30, 5.9, 6.8).
  - The runtime's Slider chassis calls `zeroMarginUnlessSet`.
  - htmlout has `sliderChassis`.

### Host pages

- **A nested vertical Scroll collapsed.** `#app [data-node-type="Scroll"]`'s
  `flex: 1 1 0; min-height: 0` replaced a nested Scroll's stated Height with
  a zero basis. Lesson 1.5's 160px "short Scroll of twelve rows" drew as its
  2px border.
- A new rule gives a vertical Scroll inside a Scroll, whose parent is not a
  row, `flex: 0 0 auto`. This is core.Scroll's documented nested behaviour
  and the same exception the List rule already had.
- It is in `wasm/index.html`, `wasm/shots/index.html` and
  `cmd/grmob/templates/wasm/index.html.tmpl`. Nothing tests that the three
  agree.

### Widgets

- **`comps.Avatar` pins `FlexShrink(0)`.** A stated Width still shrinks in a
  CSS flex row, and 4.3's 36px disc drew 34 wide. 1.1's call-site pin was
  dropped for this.
  - The ListRow field doc's claim that "an icon with a Width is unaffected"
    was corrected.
  - `docs/components.md` gained a bullet.
- **`comps.StaticMap`** puts `MaxWidth("100%")` on its frame and its image.
  - Its default 320px spilled a 296px column.
  - In a narrower column it now crops evenly under ContentModeFill, and the
    request is unchanged.
  - New section in the type doc: "Narrower than the request". Also noted in
    `docs/components.md`.

### Lessons

- **4.12's button row** and **4.19's clock row:** `Padding(0)` +
  `FlexWrap(true)`.
- **4.36's RadarChart:** `Size: 150, LabelWidth: 50`, which is 262pt wide. The
  widget is exact px, the drawing plus a label box each side, so it cannot
  shrink. All five labels are whole.
- **7.3's swatch rows:** the row and its column state `Padding(0)`, and the
  chip is `FlexShrink(0)`. TextSecondary's chip had drawn 2px wide.

## Verification

- After the fixes, all three sweeps (split, 390, 360) are empty, and every
  lesson was visited (80). The 1.5, 4.36 and 7.3 fixes were checked by
  screenshot.
- `go test ./...` passes, and the API docs were regenerated.
- `wasm/verify/run.sh` passes, browser checks included. The first run timed
  out on a CDP evaluate; a rerun passed.
- `./build.sh` rebuilt `wasm/main.wasm`.
- htmlout tests that scanned the whole document tripped on `<body style>`:
  `TestDocumentWrapper`, the one-style-attribute count, and fixedsize's
  "overflow" substring. They now look at the element under test, or at the
  clipping properties by name.
- Not run: the Android emulator and the iOS simulator (N-080).

## Gotchas

- Adding a Go file trips `TestTheQuestionsOnTheSharedRepositoryParseAreTheOnesDecidedOn`,
  which quotes the tracked-Go-file count (671) in five sentences. New tests
  went into an existing file instead.
- Headless Chrome's minimum window width is 500. Use CDP device-metrics
  emulation for phone widths.
- `Page.captureScreenshot` clips need the page's own `__grmobReady`. The real
  `wasm/index.html` does not set it; the shots host does.
- `core.Padding(0)` and `core.Margin(0)` write nothing on the web, so they
  hand the element back to the user agent's stylesheet. That is why the
  slider's margin and the fields' floor are chassis rules and not props.

## Next

Closed: None. Declined: None. Raised: N-080, N-081, N-082, N-083.
Deferred: None. Promoted: None.
Updated: None. Full list: `ai_docs/todo/next-list.md`.
