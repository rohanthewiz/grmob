# Low-hanging fruit for `comps`, round three

**Status:** drafted 2026-09-18. **G1 `CopyButton` and Tier H landed** the
same day as one batch, taught by lesson 4.29 "Copy, link and list".
**G2 `TagInput` landed** next, with lesson 5.8, then **G3 `AudioPlayer`**
with lesson 4.30 and **G4 `MessageBubble`** with lesson 4.31. Tier G is
complete. **Tier I** was settled: `ExpandableText` and `Rating.Halves` landed
with lesson 4.32, and the scrolling year-long `CalendarHeatmap` stays blocked.
Nothing in this file is unstarted.

Round one (`comps-low-hanging-fruit.md`, Tiers A–C) and round two
(`comps-low-hanging-fruit-2.md`, Tiers D–F) are both complete. This file is
the next crop and follows the same rule. Every item is **pure composition over
primitives that all four targets already render.** No file under `htmlout/`,
`wasm/`, `android/` or `ios/` is touched. If an item turns out to need a
renderer, it leaves this plan for one of its own.

The tiers continue the lettering: G, H, I.

## Where the candidates came from

Round two drew on a gap list. This round had none left to draw on: the
2026-08-31 gap analysis is fully built, and `docs/` lists no planned widgets.
The candidates came from three places instead:

- **The last Next list** named two: `CopyButton`, and a wide
  `CalendarHeatmap`.
- **Example code that builds a reusable widget by hand**:
  - `examples/chat/main.go:162` has a `MessageBubble` with literal hex
    colours.
  - `examples/mobileapp/app.go:268` has an `audioTab` with its own scrub
    slider, time row and transport buttons.
  - `examples/tutorial/widgets.go:141` has `keyPoints`, which is a bullet
    list made of plain Rows.
- **Promises in doc comments**:
  - `comps/rating.go:23` promises "a future half-glyph".
  - `comps/variant.go:12` and `docs/components.md:878` mention "a future
    Alert".

What is newly cheap, and why:

- **`core.WriteClipboard` and `ReadClipboard`** (core/clipboard.go). The
  round-two plan found its own blocked-list entry for these was stale.
- **`core.InputWithSubmit`** (core/input.go:39) reports the keyboard's
  return/done action. An input can therefore commit a value without a
  separate button.
- **`core.MaxLines`** (core/style_props.go:120) caps a Text and ellipsizes the
  last line on every target.
- **`hooks.UseAudio`** plus the `core.Audio*` commands form a whole transport.
  The only consumer is one example tab.
- **`core.CurrentWindow` and `hooks.UseWindow`** report the window width in
  layout units. A grid can be sized to a known width without measuring
  anything.

---

## Tier G — a few hours each, one decision each

### G1. `CopyButton` — **landed 2026-09-18**

**What the build changed from the sketch.**

- **The fields are Button's.** `Button` has no `Icon` or `Size`, so the
  sketch's fields went. The shipped fields are `CopyButton{Text, Label,
  CopiedMessage, Variant, Emphasis, Disabled, AccessibilityLabel,
  AccessibilityHint, Style, FocusRef}`.
- **An empty `Text` disables the button, and there is no concern.** A link
  still loading is a legitimate empty state, so a concern would be noise. The
  tap handler also refuses an empty write, which covers the race between a
  press and the patch that disables the button.
- **A layer over a `CodeEditor` needs `core.ZIndex(1)`.** On the web the
  editor is a `position:relative` `<pre>`, and a positioned element paints
  over a later sibling that is not positioned. So `codeBlock`'s button was
  laid out in the corner and drawn under the code. This was caught in a
  headless-Chrome render of the static export.
- **The tutorial's FAB test had to change.** It looked for the first
  two-layer ZStack on the screen, and every code block is now one. It now
  finds the stack by the FAB it holds.

The sketch, as drafted:

A button that puts a fixed string on the clipboard and says so. Typical uses
are code snippets, invite links, a QR code's payload, and an order number.

- Shape: `CopyButton{Text string, Label, CopiedMessage, Icon, Variant, Size,
  Disabled, AccessibilityLabel, Style}`.
  - `Label` defaults to "Copy". `CopiedMessage` defaults to "Copied".
  - `AccessibilityLabel` defaults to "Copy", or to `"Copy " + Label` when the
    visible label is something else.
- On tap: `core.WriteClipboard(Text)`, `core.Haptic(core.HapticLight)`,
  `core.ShowToast(CopiedMessage)`.

**The decision: confirm with the toast, not by swapping the button's
caption.** The in-button "Copied ✓" flash is the more familiar web idiom. It
needs a hook to time it back, and a hook makes the widget unsafe to render
conditionally. The first consumer settles it:

- The tutorial's `codeBlock` (widgets.go:88) is built "inside conditionals,
  inside loops over a demo's state, and a widget with hook obligations could
  not be" (its own comment).
- `Banner`'s doc already assigns the job: "Use the toast for 'Copied'".

So the widget stays stateless, like `Stepper`.

**First consumer:**
- `codeBlock` gains a `CopyButton` as a `StackAlignTopEnd` layer over the
  editor.
- The copied text is the trimmed snippet, which is the same string the editor
  shows.

**Concern:** `ConcernCopyButtonEmpty` for an empty `Text` in debug builds.
`WriteClipboard("")` clears the clipboard. That is legitimate when called
directly, but never what a button labelled "Copy" means.

**Not verified until a device run:**
- The toast and the haptic on both natives.
- With no host (a headless test, htmlout), `WriteClipboard` is silent, so the
  test asserts the event was sent, not what landed.

### G2. `TagInput` — **landed 2026-09-18**

**What the build changed from the sketch.** Four things.

- **The list and its items are unnamed.** The sketch had the list follow
  `KeyValueList`, which names every row. But a tag's row holds an
  interactive ✕, and a named container stands in for its children on the
  targets that merge a labelled container. The ✕ would stop being
  reachable. So the tag's text is read as is, followed by "Remove *tag*".
- **Tags go above the input, not beside it.** A wrapping Row whose last item
  is a growing input depends on how each native wraps a growing item. Tags on
  their own row, with the input full width below, is layout every target
  already does.
- **`RemoveLabel`** was added so the ✕ names can be localized.
- **The paste rule and the separator rule are one function** (`splitDraft`).
  Everything before the last separator is committed, and the rest is the
  draft. Separators may be multi-byte.

Shipped: `TagInput{Tags, OnChange, Placeholder, Label, Max, Separators,
RemoveLabel, Disabled, Style}`.

The sketch, as drafted:

A wrapping strip of removable tags followed by a text input. Typical uses are
email recipients, labels and interests.

- Shape: `TagInput{Tags []string, OnChange func([]string), Placeholder,
  Label, Max int, Separators string, Disabled, Style}`.
- A tag is committed by the return key (`InputWithSubmit`) or by typing a
  character in `Separators` (default ","). A pasted "a, b, c" commits three.
- Tags are trimmed. Empty strings are dropped, and so are exact duplicates.

**The decision: the tag's ✕ is its own button, and the tag's text is not
one.** `Chip` has exactly one tap target. Putting a ✕ inside it would nest a
button inside a button, which no accessibility tree can represent. Making the
chip itself remove on tap would bind the tag's most prominent surface to a
destructive action. So each tag is a pill `Row`:

- the label as inert `Text`;
- a ghost ✕ `Button` named "Remove *tag*".

The list is a `RoleList`, like `KeyValueList`.

**Settled by precedent:**
- **The draft text belongs to the widget.** No application wants a half-typed
  tag; this is D6's test. That makes `TagInput` hook-owning, like
  `PasswordField`, so it must render unconditionally.
- **No "backspace on empty removes the last tag".** `PINInput` found that key
  events are invisible, so an empty input's backspace cannot be seen.
- **`Max` reached disables the input rather than hiding it.** The label and
  placeholder still say what the field is for.

**Concern:** `ConcernTagInputInert`.

### G3. `AudioPlayer` — **landed 2026-09-18**

**What the build changed from the sketch.** Four things.

- **The time row shows the total, not the time remaining.** The example
  showed the total, and the value spoken by the seek bar ("12:04 of 41:30")
  reads naturally against it.
- **The seek bar states no value until a duration is known.** A 0-to-0
  `ValueRange` is one Compose cannot express, and the a11y audit reports it.
  The widget's own harness did not catch this, because the audit only runs
  inside a full render manager. `examples/mobileapp`'s app test did. There is
  now a comps test that calls `core.AuditTree` directly.
- **The title is `Typography.Body` in bold.** The bundled `Subtitle` is a
  grey secondary heading, which a headless-Chrome render showed.
- **The second line is the Artist**, except for this track's "Loading…" and
  "Couldn't play: …". `audioTab` keeps its raw state line as demo
  diagnostics. Its `clock` and `nextRate` helpers moved into the widget.

Shipped: `AudioPlayer{Track, SkipSeconds, Rates, ShowStop, Style}`, plus
`ConcernAudioPlayerNoTrack`.

The sketch, as drafted:

The transport that `examples/mobileapp`'s `audioTab` builds by hand, as a
widget: title line, scrub slider, elapsed / remaining row, −15 / play-pause /
+15, and an optional rate button.

- Shape: `AudioPlayer{Track core.AudioTrack, SkipSeconds float64, Rates
  []float64, ShowStop bool, Style}`.
- It reads `hooks.UseAudio`, so it owns hooks.

**The decision: the scrub draft belongs to the widget.** This is the opposite
of `SliderRow`, and the reasoning is the same. `SliderRow` gave the draft to
the caller because the value being drafted was the caller's. Here it is the
host's playback position:

- No application holds that position.
- A caller-held draft would be a second source of truth racing
  `OnAudioStatus`.

The widget holds "dragging to *t*" and shows *t* until `OnSliderChangeEnd`
seeks.

**Settled by the example:**
- The player is one global transport. A widget whose `Track` is not the loaded
  one shows its track idle, and Play loads it. `audioTab`'s `mine` test is
  this rule.
- Times are formatted `m:ss`, or `h:mm:ss` past an hour. The example's
  `clock()` formatter moves in.
- Controls disabled while not `mine` stay visible.

**Proof:** `audioTab` is rewritten over the widget. What remains of it is its
track and its status line.

### G4. `MessageBubble` — **landed 2026-09-18**

**What the build changed from the sketch.**

- **`ShowSender` was dropped.** An empty `Sender` hides the line, which is
  what "the speaker has not changed" already means to a caller building the
  list. The sender is never drawn on `Mine`.
- **`MineLabel` was added** ("You" by default). The bubble is one spoken
  stop named who-what-when, and the reader's own messages need a word where
  the sender goes.
- **`Style` goes on the outer row**, which is where `examples/chat` puts
  its between-message margin. Its pinned test,
  `TestTheGapBetweenMessagesIsOnTheBottomOfTheBubble`, passes unchanged
  through the widget.
- **Bubbles are capped at `MaxWidth("80%")`.** Percentages resolve on both
  natives (`GrMobMaxWidth`, `widthModifier`).
- The chat example's package doc used to say it taught `core.UseStyle`
  through the bubble. It now says the bubble is a widget and points to where
  `UseStyle` is taught.

Shipped: `MessageBubble{Text, Sender, Mine, Time, MineLabel, Style}`.

The sketch, as drafted:

The chat example's bubble, themed: a sender line (theirs only), the text, and
an optional time. It is aligned to the end for `Mine` and to the start for
theirs.

- Shape: `MessageBubble{Text, Sender, Mine bool, Time string, ShowSender bool,
  Style}`.

**The decision: which roles colour the two sides.** Mine is `Primary` with
white ink, as the example has it. The palette has no container tone for
theirs, and a new role is a theme decision this item should not force. So
theirs is `Surface` with a `Border` hairline, which is `Banner`'s answer to
the same gap.

**What it is not, and why:**
- **Not a thread.** A chat list opens at its newest message, which is a scroll
  offset. That is the same wall `Carousel` hit, so `MessageList` is on the
  blocked list.
- **No tail.** A tail is one sharp corner, and core has one radius, not four.
  This is the `DateRangePicker` notch again.

**Proof:** `examples/chat` switches to it and drops its hex literals.

---

## Tier H — an hour each, no decisions — **landed 2026-09-18**

All three landed together with G1, taught by lesson 4.29.

**What the build changed from the sketch.**

- **The weeks helper is a method, `CalendarHeatmap.WeeksFor(width)`.** As a
  method it reads the heatmap's own `CellHeight`. "Fit" also turned out to be
  the wrong word: the grid's columns stretch, so it always fits. What `Weeks`
  decides is the cells' shape. `WeeksFor` returns the count at which the
  cells come out square, clamped to between 1 and 53. It sets aside a 30px
  estimate for the weekday labels.
- **`Link` is a `Box` with `RoleLink` and `AlignSelf(start)`, around a hidden
  `Text`.** This is `StaticMap`'s tappable pattern. The `AlignSelf` keeps
  the empty width beside the link from being a tap target.
- **`BulletList` sizes ordered markers from their character count**
  (0.62 em each at the body size), because no host reports a width. The
  tutorial's `keyPoints` now builds its rows with it.

The sketch, as drafted:

- **`Link{Text, URL, OnTap, Style}`**: a `Text` with `RoleLink` in
  `PrimaryOnLight`. `OnTap` wins if set; otherwise the link calls
  `core.OpenURL(URL)`. Core Style has no text decoration, so the link is **not
  underlined**. That limit is documented; the colour and the role carry the
  link. `ConcernLinkInert` for neither a URL nor an OnTap.
- **`BulletList{Items []string, Ordered bool, Marker string, Style}`**: a
  `RoleList` of listitems.
  - The marker is hidden from accessibility and uses `FlexShrink(0)`, so a
    long item wraps under itself and not under the bullet.
  - `Ordered` numbers the items "1." and so on, right-aligned in a column
    wide enough for the last number.
  - Proof: the tutorial's `keyPoints` builds its rows with it.
- **`CalendarHeatmapWeeksFor(width float64) int`**: the number of week
  columns that fit a width, at the widget's default cell size and gap. It
  needs no field and no hook. The caller already knows its content width
  (from `hooks.UseWindow` less its own padding) and passes the result as
  `Weeks`. This is the cheap half of "53 weeks does not fit a phone". The
  other half is in Tier I.

## Tier I — worth it, with a catch to settle first — **settled 2026-09-18**

How each catch was settled:

- **`ExpandableText`: (b), with (a) folded into it.** `ToggleAfter` is a
  rune threshold, `Lines` × 40 by default. A negative value always shows
  the toggle, and a very large one never does, so a caller who knows can
  say so without a second field. One rule came out of the build: **a cap
  only comes with a toggle.** Short text is drawn uncapped, because a cap
  with no way to lift it would hide the end of the text for good. That
  makes a wrong estimate mild either way. The toggle's name stays "Read
  more", with `aria-expanded`.
- **`Rating` half-glyphs: a Canvas star, not a clip.** Reading the SwiftUI
  mapping answered the plan's question (`Overflow("hidden")` does clip
  there: `RoundedCornerShapeIfAny(clips:)`), but it raised a worse one.
  SwiftUI truncates a `Text` proposed less than its width to "…" instead
  of letting it overflow to be cut. So the clip would cut an ellipsis, not
  half a star. `Rating.Halves` (opt-in) therefore draws every position as a
  22px Canvas star, so full, half and empty are one drawing. The half is
  the star's exact left-half polygon, because a regular star is symmetric
  about the axis through its top point and bottom inner vertex. `Glyph` and
  `EmptyGlyph` do not apply with `Halves`. Without it the tree is
  unchanged. A headless-Chrome render put the Canvas stars beside "★" text
  glyphs, and they are indistinguishable at that size.
- **A scrolling `CalendarHeatmap`: not built.** This is the leaning below,
  unchanged. `WeeksFor` (Tier H) covers the phone case, and a year that
  opens on its oldest week is worse than 21 weeks that open on today.

The sketch, as drafted:

- **`ExpandableText`**: body text capped with `MaxLines`, plus a "Read more /
  Read less" toggle.
  - **The catch:** no host reports whether a capped Text actually truncated,
    because that is layout measurement, the wall `Tooltip` is behind. A short
    paragraph would then show a "Read more" that reveals nothing.
  - Options:
    - (a) The caller says `Expandable bool`.
    - (b) A rune-count threshold the caller can tune.
    - (c) Always show the toggle.
  - Leaning (a) with (b) as its default. Settle before building.
- **`Rating` half-glyphs**, promised at rating.go:23.
  - **The catch:** glyphs are text ("★"), so half a star is a clip. The clip
    would be a 50%-wide `Box` with `Overflow("hidden")` over the empty glyph.
  - Compose reads only `"hidden"` as a clip (GrMobStyle.kt:675). SwiftUI's
    `clipped()` exists in Renderer.swift, but nobody has checked that it is
    what `Overflow` maps to.
  - The alternative is a Canvas star, which gives up font glyphs and
    `Glyph`/`EmptyGlyph` customization.
  - Read the SwiftUI mapping first. If it clips, this drops to Tier H.
- **A scrolling `CalendarHeatmap`** for a full year.
  - `core.Horizontal()` scroll exists and a 53-week grid would pan. **The
    catch:** a scroll region opens at its start, the oldest week, and nothing
    can move it to the newest. That is the scroll-offset wall again.
  - Reversing the axis (newest on the left) is the only composition-only fix,
    and it breaks the convention every contribution graph follows.
  - Leaning: ship the Tier H helper and leave this blocked unless someone asks
    for a year on a phone.

## Checked, and already there

Written down so that they are not proposed again:

- **`Alert` / inline status callout.** `Banner` is it: an inline strip with a
  `Variant` tint, a leading glyph and optional actions. The "future Alert"
  lines in `comps/variant.go:12` and `docs/components.md:878` predate
  `Banner`. They get a one-line fix when G1 lands.
- **Progress ring.** `Gauge` with `Sweep: 360` is a determinate ring
  (gauge.go: "60 to 360").
- **A stacked or area chart.** `BarChart.Stacked` and `LineChart.Area` /
  `AreaChart` exist.

## Still blocked on a renderer, so it is not re-derived

Carried over:
- Carousel and pager dots (scroll offset)
- Pull-to-refresh and swipe actions (gestures)
- Tooltip and anchored popover (layout measurement)
- Native time wheel (node type)
- `RichTextView` with inline marks (inline span node)
- Per-corner radius (the `DateRangePicker` notch)
- A digits-only keyboard on `core.Input` (keyboard-type prop, from D5)

New this round:
- **`MessageList` / chat thread** (scroll offset: it must open at the end).
- **Bubble tails** (per-corner radius).
- **Strikethrough and underline.** Core Style has no text decoration.
  `examples/todoapp/app.go:334` already says so, and `Link` is not underlined
  for the same reason.
- **A year-wide `CalendarHeatmap` that opens on the newest week** (scroll
  offset; see Tier I).

## Suggested order

| Order | Item | Why |
|---|---|---|
| 1 | ~~G1 `CopyButton`~~ | named twice in Next lists; the tutorial's code blocks are the first consumer |
| 2 | ~~Tier H bundle + lesson~~ | three small pieces, each with a consumer waiting |
| 3 | ~~G2 `TagInput`~~ | the one new *input*; completes the form family |
| 4 | ~~G3 `AudioPlayer`~~ | an extraction; the example proves it |
| 5 | ~~G4 `MessageBubble`~~ | an extraction; the example proves it |
| 6 | ~~Tier I~~ | each after its catch is settled |

## Definition of done, per widget

Unchanged from rounds one and two:

- `comps/<name>.go` with a doc comment that says what the widget settles and
  which theme roles it reads.
- `comps/<name>_test.go`. It renders under `core.SetDebugMode(true)`, asserts
  an empty concern list, and asserts the role and label. For interactive
  widgets it dispatches the callback and asserts exactly one state change.
- A section in `docs/components.md`, and `docs/api/` regenerated with
  `go run ./internal/apidoc/gen`. A new file must be placed on a topic in
  `internal/apidoc/packages.go`.
- A tutorial lesson, appended at the end of its chapter so deep-linked lesson
  numbers do not move. Update the lesson count in README.md,
  docs/tutorial-interactive.md, `examples/tutorial/screenshot_test.go` and
  `internal/shotclaims/shotclaims.go`, and re-take
  `docs/images/tutorial-contents.png`.
- Update the `wasm/verify` tracked-file census (`git add -A` first).
- No file under `htmlout/`, `wasm/`, `android/` or `ios/` changed.
