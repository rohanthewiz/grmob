# Session: the bundle, seen from its first consumer

Session: https://claude.ai/code/session_01J7qf4UetyH8cQ3D7Egn9dC
Date: 2026-09-04 (follows "screen-furniture-bundle")

## Ask

"Adopt the bundle in church_mobile" — item 1 of the previous session's Next
list.

**No grmob code changed.** The whole diff is in `../church/church_mobile`
(branch `roh/use-grmob`), whose own doc
`ai_docs/claude_sessions/2026-0904-2019-adopt-widget-bundle.md` carries the
detail. What is worth recording *here* is what the widgets looked like from
the other side of the API, since that is the only thing this session can teach
the library.

## The scorecard

Four of the seven found homes, and the app's hand-rolled helpers kept their
names while losing their bodies:

    screenHeader          -> AppBar
    errorRetry            -> EmptyState (glyph + title + Retry)
    emptyNote             -> EmptyState{Title:}
    busyNote              -> EmptyState{Hint:}
    noticeStrip           -> Banner, flattened
    chipRow               -> ChipStrip{Children:}
    pagerFooter's error   -> EmptyState, glyphless, footer padding
    homeHeader's logo bar -> the same AppBar, logo in Leading

Three found none, and each for a reason that is not a gap:

- **SearchField** — the app has no search UI anywhere. It was lifted from the
  examples, not from here.
- **Skeleton** — the app's loading placeholders are text by an argued choice,
  not for want of a skeleton.
- **StatTile** — nothing in the app is stat-shaped. Also from the examples.

## What the API got right, confirmed under load

- **`AppBar`'s automatic back control.** The app's hand-rolled bar had the
  same `core.CanPop` test, the same "‹", the same 26pt, the same tightened
  padding and the same "Back" label — arrived at independently. The widget
  deleted all of it. That is the clearest evidence the lift picked the right
  boundary.
- **`AppBar.Leading` doubling as a logo slot.** The app had *two* headers, one
  with a logo and one without, because the hand-rolled bar had nowhere to put
  an image. `Leading` is otherwise the back control's, and a tab root never
  has a back control, so the slot was free and the two headers collapsed into
  one. Nothing was designed for this; the slot just fit.
- **`Banner`'s doc naming the flush case.** The app's strip is pinned edge to
  edge above the content it describes, so it wanted no frame. The doc already
  said which two props do that, and the adoption was a copy of the doc line.
- **`ChipStrip.Children` over `.Chips`.** Every call site in the app already
  wraps its chips in `core.Keyed`, so they arrive as views, not `Chip` values.
  Without the escape hatch the adoption would have had to either drop the keys
  (breaking list identity) or skip the widget.
- **`EmptyState`'s three roles.** `errorRetry`, `emptyNote` and `busyNote`
  were three separately-worded, separately-spaced blocks in the app. They are
  one widget now, which is what the "one widget for empty, busy and failed"
  argument predicted.

## The one documented departure

`busyNote` maps to **`Hint`**, not `Title` as `EmptyState`'s own busy example
shows.

The app's rule ends up being *errors and empties speak in the primary line,
waiting speaks quietly*, and the forcing case is a pager footer: a wait also
renders **mid-list**, under the last row of a paged screen, where a 17pt
primary-ink "Loading…" reads as another row rather than as the list still
arriving.

Worth deciding whether that generalizes. The widget's example is right for a
busy state that owns the whole screen and wrong for one at a list's tail, and
the widget cannot tell which it is in. Options, none taken yet:

1. Leave it — `Hint` already expresses "quiet", and the caller knows.
2. Add the mid-list case to `EmptyState`'s doc, next to the three-state
   example, so the next adopter does not have to re-derive it.

(2) is nearly free and is the likely follow-up.

## Friction found, not fixed

- **`components.Chip`'s selected look** (Surface background, Primary ink)
  reads as the *less* prominent state, so the app's year filter looks
  inverted. Latent since the A1 session; adopting `ChipStrip` did not cause it
  but does put a row of them on screen. Still the top cosmetic item.
- **No `Role`/`AccessibilityRole` prop**, so nothing in the bundle can
  announce itself as a banner, a heading or a search landmark. The app's
  adoption inherits the gap exactly as the previous session predicted. One
  core follow-up would serve this and `DataTable`'s `<table>` semantics.
- **Adoption leaves dead `*core.Context` parameters behind.** Ten of them in
  the app: the helpers took a context to read the theme, and the widgets read
  it themselves at render time now. Go does not warn, so they sat there
  describing a dependency that no longer existed until a scan found them. Not
  a library problem, but it is a predictable consequence of *any* adoption and
  worth a line in the migration notes if those are ever written.

## Downstream verification

`go test ./... -race`, `go vet`, `gofmt`, WASM build — all clean in
church_mobile. All eleven of its goldens moved in exactly one way: the
header's padding 10/12 → 8/16 (the theme's Row base, where the hand-rolled bar
used literals) and its title 20 → 22pt (`Typography.Subtitle`). That is the
entire visual delta of taking the widget, which is about as small as an
adoption gets.

Two new goldens there cover the placeholder states, which had no shape
coverage at all before.

## Next

Unchanged from the previous session, minus item 1:

1. B1–B3 in one renderer pass (horizontal scroll, `OnEndReached`, sticky
   headers). `ChipStrip` is a third waiting consumer for B1.
2. Plan A3: Calendar / DatePicker.
3. `components.Chip`'s selected look — see above; now demonstrated downstream.
4. A `Role`/`AccessibilityRole` prop in core, serving the bundle and
   `DataTable` together.
5. Small: add the mid-list busy case to `EmptyState`'s doc.
