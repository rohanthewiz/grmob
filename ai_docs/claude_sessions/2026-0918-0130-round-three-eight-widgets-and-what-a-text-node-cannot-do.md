# Round three: eight widgets, and what a text node cannot do

**Session:** d7f2be25-c341-45bc-85a8-13cb036f9c0b
**Date:** 2026-09-18 01:30 (follows "tier-e-and-f-small-pieces-and-a-scale-for-quantities")
**Branch:** master (627cf5e → 8644931 → 77e6c15 → 045ebfb → fa5ec77 → 5ba6979 → this commit)

## 1. The ask

The session opened with the user asking to "plan a third low-hanging-fruit
round". Two follow-ups came mid-turn:

- "when ready save the plan to ai_docs/plans/ and start the first item"
- "keep working through the plan, committing and pushing between batches"

Then `/sw`.

## 2. The plan — `ai_docs/plans/comps-low-hanging-fruit-3.md`

An Explore agent searched session docs, plans, examples and docs for widgets
that were wanted but not built. Candidates came from three places:

- **The last Next list:** `CopyButton`, and a wide `CalendarHeatmap`.
- **Examples that build a widget by hand:**
  - chat's `MessageBubble`, with hex literals;
  - mobileapp's `audioTab` transport;
  - the tutorial's `keyPoints` bullets.
- **Promises in doc comments:** Rating's half-glyph, and variant.go's
  "future Alert".

The tiers continue the lettering from round two:

- **G** — a decision each: CopyButton, TagInput, AudioPlayer, MessageBubble.
- **H** — no decisions: Link, BulletList, the weeks helper.
- **I** — a catch to settle first: ExpandableText, Rating halves, a scrolling
  heatmap.

**Checked and already there:**

- Alert is `Banner`.
- A progress ring is `Gauge{Sweep: 360}`.
- Stacked and area charts exist.

**New entries on the blocked list:**

- MessageList / thread (needs a scroll offset)
- bubble tails (needs a per-corner radius)
- strikethrough and underline (core Style has no text decoration)
- a year-wide heatmap that opens on its newest week

## 3. What landed, per batch

| Commit | Widgets | Lesson | Build departures worth remembering |
|---|---|---|---|
| 8644931 | `CopyButton`, `Link`, `BulletList`, `CalendarHeatmap.WeeksFor` | 4.29 "Copy, link and list" | See below. |
| 77e6c15 | `TagInput` | 5.8 "Tags: a set typed one at a time" | See below. |
| 045ebfb | `AudioPlayer` | 4.30 "An audio player" | See below. |
| fa5ec77 | `MessageBubble` | 4.31 "Message bubbles" | See below. |
| 5ba6979 | `ExpandableText`, `Rating.Halves` | 4.32 "Read more, and half a star" | Covered in section 4. |

**Batch 1 (8644931)**

- **CopyButton**
  - It is stateless. The toast confirms the copy, so `codeBlock` can hold
    it; codeBlock's own doc forbids hooks.
  - An empty `Text` disables the button and never writes. It raises no
    concern.
  - Fields follow Button's; there is no Icon or Size.
- **Every tutorial code block** now has a Copy layer with `core.ZIndex(1)`.
- **`WeeksFor`** is a method. The grid always fits its width, so Weeks
  decides the *cell shape*: square cells, a 30px allowance for the labels,
  clamped to 1–53.
- **`Link`** is a RoleLink Box with `AlignSelf(start)` around a hidden Text.
  It is not underlined and cannot sit inline.
- **`BulletList`** sizes its ordered-marker column at 0.62em per character.
  `keyPoints` now uses it.

**Batch 2 (77e6c15) — TagInput**

- Tags sit above the input, not inline with it.
- The listitems are **unnamed**, because naming them could make the ✕
  unreachable on the natives.
- `splitDraft` is both the paste rule and the separator rule, and handles
  multi-byte separators. `RemoveLabel` was added.
- The widget holds the draft, so it holds a hook.

**Batch 3 (045ebfb) — AudioPlayer**

- It is a view of the singleton player, keyed by `Track.URL`.
- The widget holds the scrub reading, the reverse of SliderRow's choice: the
  drafted value is the host's position, not the app's.
- The seek bar states no AccessibilityValue until a duration is known.
- The title is Body in bold, because the bundled Subtitle is grey.
- `audioTab` now uses the widget. Its `clock` and `nextRate` moved into comps.

**Batch 4 (fa5ec77) — MessageBubble**

- Theirs is Surface with a hairline. Mine is Primary with ink chosen by
  contrast.
- Bubbles are capped at `MaxWidth("80%")`.
- `ShowSender` was dropped: an empty Sender hides the line.
- `MineLabel` ("You") and a who-what-when accessible name were added.
- Style goes on the outer row.
- `examples/chat` now uses the widget. Its package doc no longer claims to
  teach UseStyle.

Every batch also carried the standard fallout:

- the lesson count (69 → 74) in README.md, docs/tutorial-interactive.md,
  `screenshot_test.go` and `shotclaims.go`;
- `docs/images/tutorial-contents.png` re-taken (and `tutorial-lesson.png`
  once);
- a `docs/components.md` section per widget;
- `internal/apidoc/packages.go` topics, with `docs/api/` regenerated;
- the `wasm/verify` census 588 → 602, bumped with `perl -pi -e 's/\bN\b/M/g'`.

**Pitfall:** BSD sed has no `\b`. The first census bump silently did nothing.

## 4. The three things only a render or an audit showed

1. **A positioned `<pre>` paints over a later non-positioned sibling.**
   `CodeEditor` is `position:relative` on the web, so a ZStack top layer
   over it was laid out correctly and drawn underneath. Fix: `core.ZIndex(1)`
   on the layer. Grid items take z-index without needing position.
2. **An empty ValueRange (0-to-0) fails the a11y audit.** comps' `rowHarness`
   and `renderDebug` do not run `core.AuditTree`; only a full
   render.Manager does. The mobileapp test caught it. Widget tests now call
   `core.AuditTree(node)` directly (AudioPlayer, MessageBubble). Other
   widgets' tests still do not.
3. **SwiftUI truncates a squeezed Text to "…" rather than letting it
   overflow.** So a half-star cannot be a text glyph clipped by
   `Overflow("hidden")`, even though all three targets do clip. The fix
   draws stars on a Canvas; the half is the exact left-half polygon
   (vertices 0, 9, 8, 7, 6, 5). The Canvas stars are indistinguishable from
   "★" at 22px in headless Chrome.

The Tier I settlements:

- **ExpandableText** uses a rune threshold (`ToggleAfter`, `Lines` × 40;
  negative always shows the toggle), and a cap only comes with a toggle.
- The **scrolling heatmap** is not built.

Tutorial test support: the tutorial `nodeStyle` mirror gained
`AccessibilityValue struct{ Text string }`. The FAB lesson test now finds
its ZStack by the FAB it holds, because code blocks are two-layer ZStacks
too.

## 5. Verified, and not

- **Seen in headless Chrome** (htmlout export from a scratch module with a
  `replace` to the repo): every new widget.
  - The screenshots caught the Copy-button z-order and the grey player
    title.
  - `Button` in a Column stretches to full width, which shows on a disabled
    `CopyButton`. This is Button's behaviour, not new.
- **Not seen on a device:**
  - CopyButton's toast and haptic, and a clipboard write on the natives.
  - TagInput's `InputWithSubmit` on native keyboards, and the wrapping pill
    row.
  - AudioPlayer against a real player: scrub, the lock screen, rate.
  - MessageBubble's 80% max width on the natives.
  - Canvas stars on the natives.
  - Link's RoleLink Box and ExpandableText's MaxLines toggle on the natives.
  - ZIndex on the codeBlock copy layer on the natives. ZStack order should
    already suffice there.

## Next

- **A device pass.** Carried and now longer:
  - D6: the range band's hex fill and endpoint notch.
  - D7: TimePicker's menus.
  - Tier E: AvatarStack, BarItem badge, Lightbox, PasswordField swap.
  - Tier F: Heatmap and Histogram canvases.
  - Round three: all of section 5's "Not seen on a device".
- **Most comps widget tests do not run `core.AuditTree`.** The harnesses skip
  the audit that render.Manager runs, which is how the AudioPlayer range
  concern got past them. Worth adding the audit to `renderDebug` or
  `rowHarness`, and seeing what surfaces.
- **An inline span node in core** (renderer work, a plan of its own). It
  unblocks `RichTextView`, an inline `Link`, and text decoration
  (underline, strikethrough; see `examples/todoapp/app.go:334`).
- **A per-corner radius in core** (renderer). It unblocks the
  DateRangePicker notch fix and bubble tails.
- **htmlout has no global `box-sizing: border-box`.** `Width(100%)` plus
  padding overflows in static exports only: Lightbox, AvatarStack, and now
  codeBlock's editor. Fixing it is an htmlout change.
- **Breadcrumb's ghost buttons draw a faint frame in static exports.** This
  is a Button-level decision. Not addressed.
- **CalendarHeatmap ties go to the earliest day.** Undocumented beyond the
  code.
- **A fourth low-hanging-fruit round.** The round-three plan has nothing
  unstarted. No candidates were gathered beyond the blocked list.
- *Non-goal, declined:* a year-wide scrolling `CalendarHeatmap`. It would
  open on the oldest week (no scroll offset); `WeeksFor` covers phones.
- *Non-goal, declined:* a `MessageList` / thread widget, until a host can
  report or accept a scroll offset.
- *Non-goal, declined:* a caption-flip "Copied ✓" on CopyButton. The timer
  would be a hook, which codeBlock cannot hold.
- *Non-goal, declined:* a separate `Alert` widget. Banner is it.
- *Non-goal, declined:* Heatmap as a continuous gradient (carried).
- *Non-goal, declined:* a Sequential ramp interpolated from `Primary`
  (carried).
- *Non-goal, declined:* a native time wheel, and a sheet or Done on
  TimePicker (carried from D7).
