# comps.StockTicker and tutorial lesson 4.42

Session: `3eefa321-022c-475e-8581-177e04e8185a`
**Date:** 2026-10-20 16:56 · **Branch:** master (729eee7, plus this commit)

## Goal

The user asked for "a stock ticker component". The work followed the
component guide in `ai_docs/SKILL-component.md`: a hook-free widget in
`comps`, registered in every census and doc page. The user then asked for
a tutorial lesson.

## The widget: `comps/stock_ticker.go`

```go
comps.StockTicker{Quotes: []comps.StockQuote{
    {Symbol: "AAPL", Name: "Apple", Price: 189.12, Change: 1.23},
    {Symbol: "MSFT", Name: "Microsoft", Price: 411.65, Change: -2.10},
}, OnTap: func(q comps.StockQuote) { open(q.Symbol) }}
```

- **Shape:** a `core.Scroll` with `core.Horizontal()`, following
  `ChipStrip`'s `Scrollable` branch. Each quote is an `HBox` (symbol in bold,
  price, and the movement in Caption), with `FlexShrink(0)` so it keeps its
  natural width. Quotes sit `Spacing.LG` apart.
- **Fields:**
  - `Quotes []StockQuote` (`Symbol`, `Name` spoken only, `Price`, `Change`)
  - `Start` (the quote drawn first; wraps, negative values included)
  - `Format` (receives the magnitude; nil gives two decimal places)
  - `HidePercent`, `RedRises`, `OnTap func(StockQuote)`, `Label`
  - `UpLabel` / `DownLabel` / `UnchangedLabel` / `PercentLabel` (spoken words)
  - `Style`, applied to the Scroll
- **Direction:** ▲/▼ glyphs plus a spoken word, so colour is never the only
  signal. Colours are the Success/Error ink tones (`Variant.AsInk`, as in
  StatTile), and TextSecondary when unchanged. Unlike StatTile, the widget
  colours by sign by default, because on a price tape the convention is the
  meaning. `RedRises` swaps the pair (CN/TW/JP/KR).
- **Percent** is derived from `Change / (Price − Change)`, so it cannot
  disagree with the drawn change. It is omitted when the previous close is
  zero or negative, or when a value is not finite.
- **Accessibility:** each quote is one stop for a screen reader, read as a
  sentence ("Apple, AAPL, 189.12, up 1.23, 0.65 percent"), with its drawn
  Text hidden. The tape is a `RoleList` of listitems (nesting level 1, as in
  BulletList) with the default label "Stock prices". With `OnTap`, each quote
  is a `RoleButton` and the tape a `RoleGroup`, because a list of
  non-listitems is announced as an empty list.
- **Keys:** `"quote:"+Symbol`, with a `#n` occurrence suffix for duplicate
  symbols. Occurrences are counted in the caller's order, so a quote keeps
  its key however far the tape is stepped.

### Why it steps instead of gliding

- Core has no looping translate. `core.Spin`'s doc explains why only
  rotation loops.
- A glide driven from Go would be a patch per frame over the bridge.
- `core.ScrollIntoView` was considered and rejected. It is one app-wide
  command, so a ticker using it would take the command away from the app's
  own links, and would pull the page back to the tape when the reader had
  scrolled away.
- So the widget has no hooks, and the caller steps `Start` with
  `hooks.UseIntervalWhile`. The caller also owns the pause switch that
  WCAG 2.2.2 requires.
- Raised as N-120.

## Registration

- Tests: `comps/stock_ticker_test.go` covers:
  - structure and spoken sentences, and the hidden glyph Text;
  - the direction inks across every bundled theme, with and without
    `RedRises`;
  - `Start` wrapping (0, 1, 4, −1);
  - duplicate symbols getting distinct callback IDs, under the AuditTree;
  - `OnTap` and the group role;
  - `Format`, `HidePercent` and the labels;
  - `percentChange`'s base guard;
  - the zero value, and the caller's Style winning.
- `internal/apidoc/packages.go`: added to the `display` topic (file and
  blurb), and `docs/api/` regenerated.
- `docs/components.md`: a `## StockTicker` section after StatTile.
- `ai_docs/SKILL.md`: added to the Data display & maps row.
- `wasm/verify/repowalks_test.go` and `timings_test.go`: Go file count
  709 → 711.
- No canvas, chart, composite role or own padding, so no other census
  applied.

## Lesson 4.42, "A stock ticker"

- `lessonStockTicker()` is appended at the end of chapter 4 in
  `examples/tutorial/chapter4.go`.
- One `tick` counter, driven by `hooks.UseIntervalWhile(ctx,
  !paused.Get(), step, 3*time.Second)`, feeds both `Start` and the prices.
- `tutorialQuotes(tick)` wobbles each price on a sine of the tick, rounded to
  the cent, with the tick-0 wobble subtracted. Tick 0 is therefore exactly
  the listed figures, which keeps tests and shots deterministic, and TSLA
  opens unchanged.
- Controls: Pause the tape, Red rises, a Step button, and a caption that
  shows what OnTap received.
- `TestStockTickerLesson`:
  - pauses first, so the interval cannot step the tape mid-test;
  - checks the opening sentence and that TSLA opens unchanged;
  - checks that Step puts Microsoft first;
  - checks the tap readout;
  - checks that Red rises turns a rise's ink to the Error role's.
- Lesson-count updates:
  - 84 → 85 lessons in `README.md` (twice), `docs/tutorial-interactive.md`,
    `wasm/index.html` and `internal/shotclaims/shotclaims.go`;
  - the README's chapter-4 row, 41 → 42;
  - `docs/images/tutorial-contents.png` re-taken with
    `wasm/shots/shoot.sh tutorial-contents`.

## Verification

- `gofmt -l .` prints nothing, and `go vet` is clean on `comps` and
  `examples/tutorial`.
- `go test ./...` passes.
- A throwaway `wasm/shots/scripts/zz-ticker.js` shot of the lesson showed
  the tape rendering on the web: bold symbol, price, and a green ▲ movement,
  with the second quote clipped at the panel's edge (it pans). The script was
  deleted afterwards.
- Not run on Android or iOS (N-121).

## Next

Closed: None. Declined: None. Raised: N-120, N-121.
Deferred: None. Promoted: None. Moved: None. Updated: None.
Full list: `ai_docs/todo/next-list.md`.
