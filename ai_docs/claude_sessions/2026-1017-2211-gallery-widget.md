# comps.Gallery: an immersive picture viewer

Session: `4127d87c-e7ac-49fb-a272-f9ab45c1854d`
**Date:** 2026-10-17 22:11 · **Branch:** master (`9dffd88`, plus this doc)

## Goal

This item came from the user's backlog, pasted in: "Add a fully configurable
immersive gallery component", with these requirements:

- picture titles, subtitles, descriptions and flip-over details;
- configurable animations;
- mouse and touch support;
- sane defaults throughout;
- a few themes;
- zoom of the current card;
- a "learn more" action button;
- sharing of the current card.

It was built with the grmob-component skill, as a widget contributed to
`comps`.

## What landed (`9dffd88`)

### `comps/gallery.go`

The widget, `comps.Gallery`, with these types:

- `GalleryItem`: the content fields (`Key`, `Src`, `Alt`, `Title`,
  `Subtitle`, `Description`); `Details []KeyValue` or a `Back` view; and
  `ActionLabel`/`ActionURL` and `ShareURL`/`ShareText`.
- `GalleryState`: `Index`, `Flipped`, `Zoomed` and `Paused`. It is
  comparable.
- `GalleryLook`: theme (the zero value), minimal, cinema or polaroid.
- `GalleryMotion`:
  - `Transition`: slide (zero), fade, deck or none;
  - `Flip`: fade (zero), rise or none;
  - `DurationMs`, `FlipDurationMs` and `ZoomDurationMs`: 0 means the
    default, a negative value snaps;
  - `Easing`.
- `GalleryIndicator`: auto (zero) picks dots for 10 or fewer cards and a
  counter beyond that. Dots, counter, thumbnails or none can be chosen
  directly.
- `GalleryZoom`: expand in place (zero), fullscreen through `Lightbox`, or
  off.
- `GalleryLabels`: every word in the widget, plus a `Position` formatter.

These fields are on `Gallery` itself:

- `Wrap`, `Autoplay`, `Height` and `ZoomedHeight`;
- `DescriptionLines`, `HeadingLevel` and `HideControls`;
- `OnAction` and `OnShare`;
- `Style`, `StageStyle`, `CardStyle` and `CardTheme`.

The concern `ConcernGalleryFrozen` is raised when the gallery is Controlled
without an `OnChange`.

The tree is a root Column (a labelled group). Inside it:

- the stage, a clipped ZStack holding:
  - a Keyed card layer per item, emitted last item first;
  - a hidden tap-zone Row on top;
- a navigation row: ‹, the indicator, ›;
- a tool row: Details, Zoom, Share, Autoplay, then Learn more;
- a `Lightbox`, only when zoom is fullscreen.

### Design decisions

**Every card stays in the tree.** Each card is posed with Translate, Opacity
and Rotate under a Transition, so a change of card is only a style patch and
each host animates it natively. If cards entered and left the tree, a card
entering would have no "before" style to animate from.

**Paint order.** Cards are emitted last item first, so earlier cards paint
over later ones. The deck needs this, and child order is what the natives
paint by. On the web, `ZIndex` restates that order. A transformed element is
its own stacking context there and paints above an untransformed sibling, so
the deck's fanned cards were covering the current card. Shots found this.

**State.** The gallery owns its state by default, seeded from `State`.
`Controlled` makes State the caller's. `OnChange` is called in both modes.
One hook slot is taken either way, so switching modes never shifts the
caller's hooks. Every change goes through a single `update` path that:

- clamps the index;
- shows the front of a newly current card;
- drops no-ops;
- reads the latest state at the moment of the tap or tick, not the rendered
  copy.

**Autoplay** is `hooks.UseTimeoutWhile`, keyed on the card index and the
interval. `UseIntervalWhile` was the first try and was replaced:

- It fixes its duration on the first pass, so autoplay switched on later
  ticked hourly.
- A duration of 0 panics in `time.NewTicker`.
- The timeout restarts the delay after a manual step.

Autoplay pauses while a card is flipped or zoomed, and always has a pause
toggle (WCAG 2.2.2).

**Looks.** Cinema and polaroid are fixed looks. They render the card under
`core.DarkTheme` and `core.DefaultTheme` through `core.WithTheme`, so a
Details list or a caller's Back view stays legible. Cinema reuses Lightbox's
black. The only new colour is the cinema caption band `#000000B3`. White ink
on it is about 8:1 even over a white picture.

**Caption.** The caption strip overlays the bottom of the picture in every
look, so the picture can take a known pixel height. Flex image sizing
differs between hosts. Only the current card's caption is opaque, because
fanned deck cards peeked stray caption text below the current card.

**Tap zones.** The leading quarter steps back, the middle flips (and a long
press zooms), and the trailing quarter steps forward. The zones are hidden
from assistive technology because the named buttons do the same things. A
zone with nothing to do registers no handler. The zone layer is not
displayed while the card is flipped, so a Back view takes the taps.

**Accessibility:**

- Only the current card is reachable; the others are AccessibilityHidden and
  Inert.
- Each card is a group named "3 of 12, Title". The title is a heading at
  level 2 unless `HeadingLevel` says otherwise.
- Details states Expanded. The in-place Zoom and Autoplay state Selected.
  Fullscreen Zoom states HasPopup dialog.
- Dots and thumbnails are named buttons, and the current one is marked
  `CurrentTrue`.
- The counter is a status region only when autoplay is off.
- Escape and Android back undo a flip, then an in-place zoom. They are
  claimed only while there is something to undo.

**Share and Learn more.** Share calls `OnShare`, or falls back to
CopyButton's behaviour: clipboard, haptic and toast. Learn more calls
`OnAction`, or opens `ActionURL` with `core.OpenURL`.

**Thumbnails** scroll horizontally. The indicator box got `MinWidth("0")`,
and the thumbnails start at the leading edge. Without that, the strip widened
the row and pushed › off screen. Shots found this too.

### Tests: `comps/gallery_test.go`, 20 tests

All pass under `-race`. They cover:

- the zero value;
- reachability of the current card only;
- Next, wrap and no-op dedupe;
- controlled mode, and the frozen concern;
- flip, with Escape claimed and released;
- the Back slot;
- default and custom share and action;
- buttons omitted when they have nothing to do;
- the three zoom modes;
- the tap zones;
- each transition's pose;
- every look × every bundled theme;
- the auto indicator;
- thumbnails;
- the autoplay toggle, advancing, and holding;
- the caller's Style;
- labels.

### Registration

- `internal/apidoc/packages.go`: `display` topic, file and blurb. `docs/api/`
  regenerated.
- `docs/components.md`: a `## Gallery` section before ActionSheet.
- `ai_docs/SKILL.md` and the installed grmob-native-mobile-go skill: added
  to the widget table.
- `wasm/verify`: tracked Go file count 706 → 708.
- Tutorial lesson **4.41 "An immersive picture gallery"**
  (`examples/tutorial/chapter4.go`):
  - controls for look, transition, flip and indicator, fullscreen zoom and
    autoplay;
  - an `OnChange` readout;
  - picsum.photos images by id.
- Lesson counts 83 → 84 in `wasm/index.html`, `README.md` (and the chapter
  4 row, 40 → 41), `docs/tutorial-interactive.md` and
  `internal/shotclaims`.
- `docs/images/tutorial-contents.png` re-taken.

### Verification

- `gofmt -l .` printed nothing.
- `go vet` on `comps` and `examples/tutorial` passed.
- `go test ./...` passed.
- Visual check in headless Chrome, through throwaway
  `wasm/shots/scripts/zz-gal-*.js` scripts (deleted afterwards). Seen:
  - theme look with dots;
  - cinema + deck + thumbnails, after the ZIndex and MinWidth fixes;
  - polaroid flipped;
  - minimal zoomed in place (540px);
  - fullscreen Lightbox.

## Gaps in core that the gallery works around

These are raised as N-112–N-114. Each would touch every host.

- **No swipe or drag gesture.** Tap zones are used instead.
- **No 3-D or scale transform, and no pinch.** The flip is a crossfade or
  rise, and zoom is a taller stage or a Lightbox.
- **No system share sheet.** The fallback is the clipboard, or `OnShare`.

## Next

Closed: None. Declined: None. Raised: N-111, N-112, N-113, N-114.
Deferred: None. Promoted: None. Moved: None. Updated: None.
Full list: `ai_docs/todo/next-list.md`.
