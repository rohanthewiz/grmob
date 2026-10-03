# Canvas text, clipping and per-shape taps (N-007)

Session: `776697dc-8e01-4554-92a5-61a8da54d8a4`
**Date:** 2026-10-03 16:59 · **Branch:** master (fe8bf9e → this commit)

## Ask

N-007 from the cats-todo backlog: "Canvas still omits text, clipping and
per-shape hit-testing." All three were v1 non-goals in
`ai_docs/plans/clocks-canvas-charts.md`.

## Design

### API (`core/canvas.go`)

- **`Shape.Text *CanvasText`**: the shape becomes a single line of text. It is
  sent as a `CanvasText` node rather than a `CanvasShape`, so a slot that
  switches between the two is replaced, not patched.
  - `X, Y` is the anchor, in viewBox units.
  - `Size` is in layout units, like `StrokeWidth`; 0 means 12.
  - `Bold`.
  - `Align`: `CanvasAlignStart`, which is the zero value, then `Middle` and
    `End`.
  - `VAlign`: `CanvasVAlignMiddle`, which is the zero value, then `Top` and
    `Bottom`. The edges are the font's line box.
  - `Shape.Fill` colours the glyphs. No fill paints nothing.
  - In a `CanvasMirrorsRTL` canvas laid out right to left, the anchor mirrors
    but the glyphs do not, and start becomes the text's right-hand end.
- **`Shape.Clip *Path`**: confines that one shape's paint, using the nonzero
  rule. It is per shape so that axis strokes can run unclipped along a clipped
  plot area. An empty clip is sent as `[]`, which hides the shape.
- **`Shape.OnClick func()`**: per-shape taps. The canvas gets one text callback,
  `onShapeTap`, and only when some non-text shape has a handler and the canvas
  is not disabled. The wire is otherwise unchanged.

### Hit-testing in Go (`core/canvas_hit.go`)

The host reports `"x,y,w,h"`: the tap and the box size, in layout units. When a
mirrored drawing is mirrored, the host has already reflected the x back.

Go then works in the box:

- It maps each path into the box with `CanvasMapping`.
- It flattens cubics by uniform subdivision, choosing n from the bound
  ¾·d/n² ≤ 0.1, capped at 128.
- Fills are tested by winding number (nonzero) or parity (even-odd).
- Strokes are tested by segment distance against half the width, as if caps and
  joins were round and there were no dashes.
- The clip is tested as nonzero.

The rules:

- Shapes are tried topmost first, and only shapes with a handler take part.
- A miss runs the canvas's own `onClick` through `ctx.TriggerCallback`.
- Text is never hit.

The reason for doing it in Go is the same as for flattening arcs in Go: one
implementation, and every target picks the same shape.

### Web (`htmlout/canvas.go`, `wasm/grmob-runtime.js`)

- **Clips:** a `<clipPath clipPathUnits="userSpaceOnUse">` in the existing
  leading `<defs>`, after that shape's gradients. Its id is
  `CanvasClipID` = `grmob-<path>-clip-<i>`. A path refers to its clip right
  after `d`.
- **Text:** `<g clip-path><text …></g>`.
  - SVG scales `font-size` with the viewBox, so the `<text>` gets a CSS
    `scale: var(--grmob-canvas-ix,1) var(--grmob-canvas-iy,1)` about its anchor
    (`transform-box: view-box; transform-origin: Xpx Ypx`).
  - The runtime keeps `--grmob-canvas-ix/-iy`, which are 1/sx and 1/sy, on the
    `<svg>` from a ResizeObserver (`syncCanvasTextScale`). htmlout's static
    export leaves them at 1.
  - **Finding, checked headless:** a `clip-path` is resolved after the
    referencing element's own transform, so a clip on the scaled `<text>`
    landed in the wrong place. That is why the clip goes on a wrapper `<g>`.
  - Mirroring: a mirrored canvas multiplies the x scale by `--grmob-inline` and
    lets `direction` inherit, so `text-anchor: start` becomes the right-hand
    end under `dir="rtl"`. Other canvases pin `direction="ltr"`.
  - Vertical alignment uses `dominant-baseline`: `text-before-edge`, `central`
    or `text-after-edge`.
- **Taps:** `attachShapeTap`.
  - It handles pointer clicks only (`detail > 0`). `eventQualifies` keeps those
    clicks away from the canvas's `onClick`, while keyboard activation
    (`detail` 0) still reaches it.
  - It consumes the long-press flag itself.
  - `canvasTapPoint` computes the point from the bounding rect and
    `clientWidth`, and reflects x when `--grmob-inline` is -1 on a mirroring
    canvas.
- **Other runtime changes:**
  - `SVG_TAGS` gained `g` and `text`.
  - Canvas props are kept on the `<svg>` (`__grmobCanvasProps`).
  - `syncCanvasGradients` now also builds the clips and rewrites text
    attributes.
  - Style removal is guarded, because the DOM shim has no `removeProperty`.
- `htmlout/export.go` records `data-onshapetap`. `htmlout/tag.go` maps
  `CanvasText` to `g`.

### Compose (`GrMobCanvas.kt`)

- The draw loop now dispatches on the child type:
  - `drawCanvasText`: `TextMeasurer`, one line, dp converted to sp with the
    font scale divided out; the aligned share is reversed when reflected.
  - `drawCanvasShape`: the old body, factored out.
  - Each draw is wrapped in `clipPath(clip)` when the shape has a clip.
- `pointerInput(detectTapGestures)` sits inside the node's clickable, so it
  consumes the press before the clickable sees it. It sends the point in dp and
  forwards a long press to `onLongPress`.
- `Renderer.kt` gained a `"CanvasText" -> Unit` arm.

### SwiftUI (`Renderer.swift`)

- Each shape draws on a copy of the context, clipped when it has a clip.
- `grMobDrawCanvasText` uses `ctx.draw(Text, at:, anchor: UnitPoint)` with a
  fixed `.system(size:)`.
- `grMobCanvasPath` was factored out.
- The outset estimate covers text: `min(160, size·(0.6·chars + 1))`.
- The tap layer is a clear `GeometryReader` overlay with
  `onTapGesture(coordinateSpace: .local)`. A child gesture beats grMobBox's own
  tap. The layer also forwards a long press.
- A `case "CanvasText"` arm was added.

### Tutorial 4.19

- A new prose paragraph, a code block and a "week chart" panel.
- Five pills are clipped flat at the baseline. Each has a value label above it
  and a day label below it, drawn as canvas text. A tap picks a bar; a tap
  anywhere else clears it through the canvas's own `OnClick`.
- The caption under the chart reports the pick.
- The panel title is fixed. `demoPanel` keys by its title, so a title that
  changed with the pick would rebuild the canvas on every tap.

## Seen

Taps were aimed at the same viewBox points on all three targets.

| Tap | Expected | Chrome (headless CDP) | iPhone 17 Pro sim | Android emulator |
|---|---|---|---|---|
| Tue bar (39, 30) | "Tue: 42" | ✔ | ✔ | ✔ |
| Under Tue's baseline (39, 52–54): the clipped-off pill end | miss, then clear | ✔ | ✔ | ✔ |
| Thu bar (85, 40) | "Thu: 36" | ✔ | ✔ | ✔ |
| Empty corner | clear | ✔ | ✔ | ✔ |

- In the screenshots, all three agree: labels at the layout size, the picked
  value in bold, pills round on top and flat at the baseline.
- On the web, `--grmob-canvas-ix` was 0.4, which is 1/2.5 for a 160px-tall
  120×64 drawing.
- **Mirrored text under RTL:** checked on Chrome only, with a static page
  exported by htmlout and `dir="rtl"`. The mirrored canvas's anchor mirrors,
  the glyphs read normally, and start grows leftward. The unmirrored canvas
  keeps its labels where they were drawn. **Not seen on the natives** (N-090).

## Tests

- **Core** (`core/canvas_hit_test.go`):
  - topmost-with-handler wins, a gridline is transparent, a stroke counts only
    within half its width, and the clip;
  - fit letterbox and stretch;
  - fill rule;
  - `onShapeTap` present only when it can fire;
  - `parseCanvasTap`;
  - the `CanvasText` wire;
  - the `clip` wire, including an empty clip.
- **htmlout:** `TestCanvasClipAndTextExport`, which includes the XML case of
  `clipPath`, `g` and `text`, and `TestCanvasTextExportInAMirroredCanvas`.
- **Web runtime:**
  - `gen.go` has two new cases (clips and text, fit; text in a mirrored
    stretched canvas);
  - `canvas_test.mjs` compares text groups and clip servers with htmlout's;
  - a text update-props patch;
  - a tap payload, including mirrored, a keyboard click, and losing
    `onShapeTap`.
- **iOS:** new `ios/GrMobUITests/TutorialCanvasTapUITests.swift`, run after
  `xcodegen generate`. It passed (24.9 s).

## Verification

- `gofmt -l` is clean, `go vet ./...` is clean, and `go test ./...` passes.
- `wasm/verify/run.sh` exits 0, including the browser.mjs checks.
- `android/verify/run.sh` passes, including "10 canvas drawings map and
  decode". `android/verify/sources.sh` passes.
- **`ios/verify/run.sh` fails** with `no such module 'UIKit'` in
  `GrMobSurface.swift`. It fails identically on a clean worktree of HEAD
  (fe8bf9e), so it predates this session (N-089).
- Knock-on changes:
  - `docs/api/*` regenerated.
  - The tracked-Go-file count went from 678 to 680 in `repowalks_test.go` and
    `timings_test.go`, for the two new `core` files.
  - The plan doc's non-goals now record that these three features landed.

## Harness notes

- **The claude-in-chrome tab froze.** `requestAnimationFrame` and
  ResizeObserver never fired in a background tab, and a `Runtime.evaluate` that
  awaited a frame timed out. Headless Chrome driven over CDP from a scratch Node
  script (`--remote-debugging-port`, `Input.dispatchMouseEvent`,
  `Page.captureScreenshot` with a clip) was reliable.
- **The emulator was asleep, not locked.** The uiautomator dump showed a
  passcode keypad, but `isKeyguardShowing=false`, and `input keyevent
  KEYCODE_WAKEUP` brought the app back.
- **The emulator canvas box can be read from the dump.** uiautomator gives the
  canvas's px bounds through its `content-desc`, so a viewBox point maps to a
  screen point with `CanvasFit`'s rule, and taps go through `adb shell input
  tap`.

## Next

Closed: N-007. Declined: None. Raised: N-089, N-090.
Deferred: None. Promoted: None.
Updated: None. Full list: `ai_docs/todo/next-list.md`.
