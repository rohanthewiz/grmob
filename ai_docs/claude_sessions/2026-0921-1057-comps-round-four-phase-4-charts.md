# Comps round four, phase 4: charts on Canvas

**Session:** 52c8a0d5-b2fb-4d58-bd23-a85fcd23f56b
**Date:** 2026-09-21 10:57
**Branch:** master (3a4694c → this commit)

## The ask

1. `/sl` (loaded `2026-0921-1035-comps-round-four-phase-3-structure` and the
   next list).
2. "Start phase 4 of the comps round four plan" —
   `ai_docs/plans/comps-low-hanging-fruit-4.md`, Phase 4: M1
   `CandlestickChart`, M2 `FunnelChart`, M3 `RadarChart`, M4 `Waveform`, and
   one lesson.
3. `/sw`.

## What landed

| Piece | Files |
|---|---|
| M1 `CandlestickChart` | `comps/candlestick_chart.go`, `_test.go` |
| M2 `FunnelChart` | `comps/funnel_chart.go`, `_test.go` |
| M3 `RadarChart` | `comps/radar_chart.go`, `_test.go` |
| M4 `Waveform`, `AudioPlayer.Waveform` | `comps/waveform.go`, `_test.go`, `comps/audio_player.go` |
| Lesson 4.36 "Four more charts" | `examples/tutorial/chapter4.go`, `chapter4_test.go` |
| Docs | `docs/components.md` (four sections before Compass, a bullet on AudioPlayer), `internal/apidoc/packages.go` (four files on "charts"), `docs/api/` regenerated |
| Lesson count 78 → 79 | README.md, docs/tutorial-interactive.md, `wasm/index.html`, `internal/shotclaims`, `pagecount_test.go`, `screenshot_test.go`, `docs/images/tutorial-contents.png` re-taken |
| Census 656 → 664 tracked Go files | `wasm/verify/repowalks_test.go`, `timings_test.go` |
| Bookkeeping | the plan (status, phase table, four "What the build changed" blocks, order row 6 struck, one new "still blocked" entry), `ai_docs/todo/next-list.md` |

`go test ./...` and `wasm/verify/run.sh` pass. The round's rule held: no file
under `htmlout/`, `android/` or `ios/` changed, and `wasm/` only for the
lesson count and the census.

## The lesson is 4.36, not in chapter 6

The plan guessed chapter 6 and said to check. Chapter 6 is "Navigation &
Overlays"; the chart lessons are 4.20 and 4.28. Appended to chapter 4.

## M1 `CandlestickChart`

- Reuses `cartesianFrame`, `niceScale(…, false)` and `bandLabels` whole.
- Four shapes carry every candle: rising wicks, falling wicks, rising
  bodies, falling bodies, each one path of subpaths. Wicks under bodies.
- The 1px doji is exact: the plot is `Height` px and `CanvasStretch` maps y
  linearly, so a px is `100/Height` units. A close equal to the open is up.
- The wick spans all four prices (`Candle.extent`), so a malformed row stays
  inside the axis. A candle with a NaN keeps its slot and draws nothing.
- Added beyond the sketch: `Subject`, `AccessibilityLabel`, `BodyFill`,
  `ShowLegend` with `UpLabel`/`DownLabel` (with the convention regional, a
  reader needs a key). The summary ends "2 up, 1 down", since colour is the
  only drawn difference.

## M2 `FunnelChart`

- Row of: names (`bandColumn`), the stretched Canvas, values, and rates.
  The rates column is the same px bands with weights ½, 1, …, 1, ½, which
  puts each rate level with the boundary it describes.
- A band is a trapezoid, own width on top, the next stage's at the bottom;
  the last is a rectangle. A 2px seam, converted through `100/h`. Widths are
  shares of the largest stage.
- **`AllowIncrease` was added** to silence `ConcernFunnelChartStageGrows`
  for data that is meant; "debug builds note it" alone would be permanent
  noise for real re-entry data.
- One hue fading (solid → 35% alpha), not the categorical palette.
  `funnelColumn` exists because `bandColumn`'s ink is fixed.

## M3 `RadarChart`

- **The plan's "check first" was not needed.** `core.Translate` takes px and
  `Size` is known, so each rim label is a layer the ZStack centres, moved by
  px from the drawing's own trigonometry. Anchored by side: start-aligned
  right of centre, end-aligned left, centred top and bottom (threshold 0.3
  on cos and sin). The stack is pinned to `Size` + 2·(`LabelWidth` + gap).
- The grid is polygons and spokes in one path; each series is a polygon and
  a markers path (two slots, always). After `Close()` a path is still
  "open", so each marker dot needs its own `MoveTo` before `Arc`.
- Ring values beside the upward spoke (`HideScale`). `radarStep` adds 2.5 to
  the nice ladder: with the ring count fixed, `niceNum` took a high of 9
  over four rings to a rim of 20.
- Under RTL the labels mirror and the Canvas does not. Read from source (no
  target has Canvas mirroring code); N-071.

## M4 `Waveform`

- **Bars are strokes.** A rounded `Rect` under `CanvasStretch` would get
  elliptical corners; a Canvas stroke is never scaled, so a bar is a
  vertical line `BarWidth` px thick with `CapRound`. ViewBox is n × Height,
  so x is a unit per bar and y is px; the caps' overshoot comes off the
  reach exactly. Two shapes: played and rest.
- `Bars` + `BarsFor(width)` instead of a width field (`WeeksFor`'s
  pattern). Downsampling by bucket maximum with integer bucket bounds.
  Unset `Bars` caps at 56. A silent bar is a half-px dot, not a zero-length
  subpath.
- `Decorative` hides it; `AudioPlayer.Waveform []float64` draws it above
  the seek bar with `Decorative` set, following the scrub reading.

## The look

Four throwaway shot scripts (`wasm/shots/scripts/zz-*.js`, deleted), with
`GRMOB_SHOTS_OUT=<scratch> ./shoot.sh zz-…`. `scrollTo` needs text that is
on the page: the funnel has no code block, so its anchor was the demo hint.
**Two defects found, both fixed:**

- **Radar ring values were struck through** by the profile's edge and
  marker. Each now sits on a chip of `Background` at 85% that hugs its text.
  Nowhere inside the rim is safe from every dataset, so legibility, not
  position.
- **The waveform was a solid band.** The lesson copied 4.29's
  `win.Width − 64`; the strip's real inset measured 118px, so 70 bars were
  asked of 296px. Now `min(win.Width, 430) − 120`: the cap is for the
  browser's two-pane layout, where the window is a desktop's and the demo a
  phone's. 4.29's calendar has the same exposure, masked by `WeeksFor`'s
  clamp at 53.

Candlestick and funnel were right first time.

## Two test notes

- `Style.Align` is the text alignment field (not `TextAlign`).
- The lesson test reads the candles' body fills in drawing order and asserts
  the swap exchanges the pair; a set comparison would pass with no swap.

## Not run

- `android/verify` and `ios/verify`: nothing under either changed.
- No device, emulator or screen reader. All of it is N-070.

## Next

Closed: None. Declined: None. Raised: N-070, N-071.
Deferred: None. Promoted: None.
Updated: None. Full list: `ai_docs/todo/next-list.md`.
