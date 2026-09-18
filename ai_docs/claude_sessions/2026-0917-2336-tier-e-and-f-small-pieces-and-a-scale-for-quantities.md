# Tier E and Tier F: seven small pieces, and a scale for quantities

**Session:** 8a46a73e-20d4-4eb3-adf9-1063e1d518a5
**Date:** 2026-09-17 23:36 (follows "time-picker-and-the-sheet-it-did-not-need")
**Branch:** master (7952e5c → 610832b Tier E → 7a9da89 Tier F → this commit)

## 1. The ask

"start Tier E" (`ai_docs/plans/comps-low-hanging-fruit-2.md`, step 6: "Tier E as
one bundle with one lesson"). Mid-turn the user added: "when done, commit, and
followup with Tier F, then finish with /sess-wrap". So: Tier E → commit →
Tier F → commit → this doc → push.

## 2. Tier E — seven widgets, lesson 4.27 (610832b)

Everything lives in `comps/`. None of the renderer directories changed.

| Widget | File | Where the build departed from the sketch |
|---|---|---|
| `KeyValueList{Rows []KeyValue, Dividers, Label, Style}` | key_value_list.go | The key is ListRow's **Leading** with `FlexShrink(0)`, not its Title. As the Title, a long value ("12 Kingfisher Lane, Apartment 4, Portsmouth") squeezed "Ship to" onto two lines, which the first screenshot showed. The column is a `RoleList` and each row a listitem (`NestingLevel: 1`) named "Key, Value". |
| `Breadcrumb{Items, OnTap func(int), Label, Separator, Style}` | breadcrumb.go | Ancestors are ghost Buttons (horizontal padding cut to XS). The last item is Text with `CurrentPage`. With `OnTap` nil the whole trail is text, and that raises no concern. The row wraps. |
| `AvatarStack{Avatars, Max, Size, Overlap, RingWidth, RingColor, Label, Style}` | avatar_stack.go | A ZStack of layers placed `StackAlignStart` at growing `MarginLeft`, and the box is pinned. **The ring is its own Background disc layer, not a border.** The natives paint a border inside the box, while htmlout has no global `box-sizing` and paints it outside (content-box), so the web would be 4px wider. `Max` counts the +N disc. The stack is one `RoleImg` with a synthesized name ("A, B and 3 others"). Default overlap is **0.2**: 0.3 and 0.25 visibly cut the second initial. An empty stack is a hidden Box. |
| `LabeledSeparator{Label, Color, Style}` | labeled_separator.go | As sketched. The rules grow and are hidden; the label is read and has `FlexShrink(0)`. |
| `PasswordField{Value, OnChange, Placeholder, Label, ShowLabel, HideLabel, RevealLabel, Disabled, Style}` | password_field.go | **It is the input, not a FormField** (DatePicker's rule). It owns one hook, the reveal state, so it must render unconditionally. `Input` and `InputPassword` are swapped as different node types, which is a replace; harmless because focus is on the toggle. The toggle's caption flips Show/Hide, but its name stays "Show password" with `AccessibilitySelected` giving aria-pressed. `ConcernPasswordFieldInert`. |
| `BarItem.Badge`, `BarItem.BadgeLabel` | bottom_bar.go | The icon becomes a two-layer ZStack with the Badge at `StackAlignTopEnd` and `VariantError`. The glyph gets a symmetric `MarginHorizontal(XS+SM)` and **no top margin**, so badged icons stay level with their neighbours. The badge is compact (Caption−2, padding 1/5); at full size it covered the glyph. The count joins the cell's name ("Inbox, 3 unread") and the badge Text is hidden. With no icon, the label carries the badge. An unbadged item's tree is unchanged. |
| `Lightbox{Src, Alt, Caption, Open, OnDismiss, Height, CloseLabel, Style}` | lightbox.go | A Modal around a fitted image at `ImageWithMode(Fit)` and 400px. It is dark in every theme through fixed constants. The panel is filled because SwiftUI's Modal is a sheet on the system background. **The panel has no padding and the image is full bleed:** padding on a `Width(100%)` box overflows under content-box. The ✕ is a ghost Button named "Close". `ConcernLightboxInescapable`. |

Lesson 4.27 `lessonSmallPieces()` ("Seven small pieces") is an order-detail
screen that uses all seven. The Inbox badge clears when Inbox is tapped, which
shows the count is caller state. `orEmpty` helper in chapter4.go.
`TestSmallPiecesLessonDrivesEachPiece` covers it.

## 3. Tier F — two charts landed, one moved out (7a9da89)

- **Theme decision: `ColorPalette.Sequential []string`** + `SequentialColors()`,
  `DefaultSequentialColors()` and `DefaultDarkSequentialColors()` (the light
  list reversed, so "more" runs away from the page). It is set on all three
  bundled themes; `TestBundledThemesSetEveryColorRole` walks the struct
  reflectively and would otherwise have failed.
  - Rejected: a ramp from `Primary`. From white, AmberTheme's #FFA000 (OKLab L
    0.78) spans 0.22 of lightness and DefaultTheme's #0040DD (L 0.46) spans
    0.54.
  - Rejected: ColorBrewer Blues-5. Its light end bunches, ΔE 7.1 against 14.9.
  - Shipped: `#C4D9F6 #90B7EA #5E94DA #2B71C7 #0B519D`. These hold chart slot
    1's hue (OKLCh h −104.5°), with even L from 0.88 to 0.44 and adjacent ΔE
    11.0–11.6. The first step is ΔE 9.3 / 10.2 / 12.6 from the Default,
    Material and Amber Surface colours, which is the "no data" colour. The
    numbers were computed with a Python OKLab script during the session.
- **`Heatmap`** and **`CalendarHeatmap`** (heatmap.go) share a private
  `heatGrid`:
  - Colouring uses equal steps rather than a gradient. The top value falls in
    the top step, and a flat range is all top step.
  - NaN means no data and is painted Surface. `absent` cells are not drawn at
    all.
  - Drawing is one Canvas path per step plus one for no data. Nil paths keep
    their slots.
  - Labels: `spanLabels` lets an empty column label lend its slot to the label
    before it; a label spanning more than one slot aligns to the start.
  - The legend shows the low value, the swatches, then the high value.
  - `CalendarHeatmap`:
    - `Weeks` defaults to 17 and `WeekStart` to Sunday.
    - Entries are summed per date in `End`'s location.
    - Zero becomes NaN, and days after `End` are absent.
    - Rows are labelled Mon/Wed/Fri. A month is named where the week holds the
      1st; the first column is named only if the next label is at least 3
      columns away.
- **`Histogram`** (histogram.go) needed no new axis:
  - The n+1 edges use `pointLabels` (LineChart's edge-to-edge spacing), and
    bars have 0.94 fill.
  - Edges come from `niceScale(lo, hi, Bins+1, false)`, so `Bins` is a target
    and defaults to Sturges' rule.
  - Bins are half-open, with the maximum in the last bin.
  - The count axis step is forced to ≥ 1.
- **RichTextView moved out.** Core has no inline span node, and `TextGrid` is
  monospace and does not wrap. Adding one means renderer work, and the plan's
  opening rule sends that elsewhere. It is now on the plan's blocked list.
- **A stale plan entry was found.** The blocked list said a clipboard copy
  button was blocked on "no clipboard bridge", but `core.WriteClipboard` and
  `ReadClipboard` exist. It is now noted as unblocked; nothing was built.

Lesson 4.28 `lessonHeatAndSpread()` ("Heat and spread") has three demos:

- A calendar with a "Log a workout" button.
- A weekday × hour matrix with one NaN slot.
- A histogram of 160 deterministic latencies with an Auto/4/8/16 bin target.

`TestHeatAndSpreadLessonDrivesTheCharts` covers it.

## 4. Fallout, both commits

- `internal/apidoc/packages.go`: the new files are placed under structure,
  lists, inputs, overlays, display and charts, with blurbs extended.
  `docs/api/` is regenerated. apidoc tests fail if a file declaring constants
  is on no topic, which is how lightbox.go was caught.
- Lesson count went 67 → 68 → 69 in README.md, docs/tutorial-interactive.md,
  `examples/tutorial/screenshot_test.go` and `internal/shotclaims/shotclaims.go`.
  `docs/images/tutorial-contents.png` was re-taken twice with
  `wasm/shots/shoot.sh tutorial-contents` and read back as 69.
- The `wasm/verify` tracked-Go-file census went 572 → 584 → 588, across five
  sentences in repowalks_test.go and timings_test.go. Stage the new files with
  `git add -A` before reading the count, because it counts `git ls-files`.
- `docs/components.md` gained sections for KeyValueList, LabeledSeparator,
  AvatarStack, PasswordField, Breadcrumb, "### Badges" under BottomBar,
  Lightbox, Histogram, and "Heatmap & CalendarHeatmap". The charts had no
  components.md sections before this; these two are the first.
- `docs/concepts/styling-and-theming.md`: the colour-roles table gained `Chart`
  (it was missing) and `Sequential` rows, plus a "Chart is not Sequential"
  bullet.
- `core/theme_test.go`: `TestSequentialColorsResolve`.
- Tests added: 17 Tier E tests in comps plus 1 in the tutorial; 8 Tier F tests
  in comps plus 1 in core and 1 in the tutorial. The full suite, `go vet ./...`
  and `gofmt` are green at both commits.

## 5. Verified, and not

- **Seen in a browser:** `htmlout.ExportHTML` output was screenshotted with
  headless Chrome from a scratch module (with a `replace` to the repo) and
  showed every Tier E piece and all three Tier F charts. The screenshots
  caught four real problems: the key wrap, the initials overlap, the badge
  covering the glyph, and the Lightbox padding overflow.
- **Pitfall:** headless Chrome has a minimum window width of about 500px. A
  `--window-size=420` shot is a *crop* of a wider layout, which is why the
  Lightbox ✕ looked missing. Shoot at ≥ 600 wide, or pin the content width.
- **Not seen on a device:**
  - AvatarStack's margin-placed layers and ring discs.
  - The badged BottomBar ZStack.
  - Lightbox on an iOS sheet or Compose dialog.
  - Heatmap and Histogram canvases.
  - PasswordField's node-type swap on the natives (keyboard behaviour).
- htmlout's static export uses serif and content-box. The 48px two-initial
  avatars still lose a sliver of the second letter at 0.2 overlap in serif.
  The natives' sans should be narrower, but this is unverified.

## Next

- **A device pass**, carried from D6/D7 and now longer:
  - From D6: the range band's 8-digit hex fill and endpoint notch. If the
    notch reads badly, the fix is a per-corner radius in core, which is a
    renderer change outside this plan.
  - From D7: TimePicker's three menus in a row, and the 60-item minute list.
  - From Tier E: AvatarStack layers and rings; BarItem badge placement; the
    Lightbox on an iOS sheet and a Compose dialog; the PasswordField swap and
    keyboard.
  - From Tier F: Heatmap and Histogram canvases.
- **A third low-hanging-fruit round.** `comps-low-hanging-fruit-2.md` has
  nothing left unstarted. Candidates:
  - `CopyButton`: now pure composition over `core.WriteClipboard`, and the
    plan's blocked-list entry was stale.
  - Horizontal scroll for a wide `CalendarHeatmap`: 53 weeks does not fit a
    phone at 17's cell width.
- **An inline span node in core** (renderer work, a plan of its own). It
  unblocks `RichTextView`, the one Tier F item that left.
- **htmlout has no global `box-sizing: border-box`,** unlike the WASM host and
  shots pages. `Width(100%)` plus padding overflows only in static exports, and
  borders size differently from the natives. Lightbox and AvatarStack were
  written around it; fixing it is an htmlout change.
- **Breadcrumb's ghost buttons draw a faint frame in static exports.** This is
  the Button base's shadow, the same on every ghost Button (Dialog's Cancel
  too). Not addressed; it would be a Button-level decision.
- **CalendarHeatmap ties go to the earliest day.** "Most on" names the first
  of equal maxima. Undocumented beyond the code; revisit if someone wants the
  latest.
- *Non-goal, declined:* Heatmap as a continuous gradient. Discrete steps are
  what a legend can key (see heatmap.go, "Steps, not a gradient").
- *Non-goal, declined:* a ramp interpolated from `Primary` for the Sequential
  role (see core/theme.go, `Sequential`).
- *Non-goal, declined:* a native time wheel (carried from D7). It needs a node
  type, and the plan keeps it out.
- *Non-goal, declined:* a sheet or Done button on `TimePicker` (carried from
  D7). A time has no half-made value.
