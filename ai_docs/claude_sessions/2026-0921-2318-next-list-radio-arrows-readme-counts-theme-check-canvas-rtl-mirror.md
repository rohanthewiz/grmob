# Next list: radio arrows, README counts, theme check, canvas RTL mirror

Session: `34b37205-c041-4cd4-8367-242527e91619`

## Ask

Work the Next list items that need no physical device, two at a time:

1. N-067 and N-063. Committed as `f5cb596`.
2. N-075 and N-071. N-071 was measured first and needed a decision. The user
   chose **flip the canvas under RTL**. Check 20 and the RTL findings were
   committed as `95a669b`; the mirror itself is in this doc's commit.

## N-067: radio groups answer both arrow pairs (web)

- `wasm/grmob-runtime.js`, `handleCompositeKey`: a `radiogroup` takes Down
  and Right as next, and Up and Left as previous, whatever its axis. This
  follows ARIA's radio group pattern.
  - Under RTL the horizontal pair swaps, in either orientation. The vertical
    pair never swaps.
  - One-axis composites (listbox, tablist, toolbar) still leave the cross
    pair to the page.
- Tests: three added to `wasm/verify/keynav_test.mjs`. They run against
  dom.mjs only; nothing was tried in a real Chrome on lesson 5.9.
- Docs: `docs/platforms/wasm.md` (a row in the key table) and the
  ColorSwatchPicker section of `docs/components.md`.

## N-063: two stale statements

- The README chapter table had three wrong counts: chapter 4 is 37 (was 14),
  chapter 5 is 9 (was 6), chapter 6 is 8 (was 5). The total of 80 was
  already right.
- New `examples/tutorial/readme_counts_test.go` holds the table rows and both
  "N lessons across M chapters" sentences (README and
  `docs/tutorial-interactive.md`) to `Chapters`. It skips the screenshot's alt
  text, which describes a picture taken on a given day.
- In `ai_docs/plans/comps-low-hanging-fruit-4.md`, "Per-corner radius" is
  struck through with a note.

## Pre-existing failure fixed on the way

The `wasm/verify` tracked-Go-file count was stale: the last session added
`theme.go` and `theme_test.go` without updating it. It went 666 → 669 in the
first commit and 669 → 670 for `comps/canvas_mirror_test.go`. Five sentences
quote it: `repowalks_test.go` lines 30, 532 and 730, and `timings_test.go`
lines 823 and 892.

## N-075: browser check 20, the theme switch

Check 20 in `wasm/verify/browser.mjs` boots the site page (`/site/#1.2`) at
1280px with Chrome's `prefers-color-scheme` emulated.

- **What it reads:** each pane's background, and a count of elements inside
  the pane in each theme's caption ink (TextSecondary, the witness
  theme_test.go uses).
- **The steps:**
  1. System on a dark OS.
  2. A click on Light.
  3. A reload with Light remembered. `THEME_BOOT_PROBE` reads `data-theme`
     and `--screen-bg` when `<body>` first exists, and which caption hex
     RenderInitial's first tree carries.
  4. A click on System.
  5. The OS turning light, with no reload.
- **Mutation-tested:** removing boot()'s `sendTheme()` fails step 1, and
  removing the head script's attribute fails the no-flash reading.
  `index.html` was restored both times.
- The header tallies moved to "four about paint" and "Twenty claims", and
  `run.sh`'s prose was updated.

## N-071: charts under RTL, measured, then fixed

### The measurement

- **Harness:** a scratch CDP script in the scratchpad (`rtlshots.mjs`)
  serving a copy of `wasm/`.
- **Gotcha:** `dir` set by addScriptToEvaluateOnNewDocument never held,
  because `<html>` doesn't exist yet at that point. Set it after load instead.
  The CSS applies live.
- **Before the fix:**
  - Line, area, bar, stacked bar, scatter and candlestick: labels mirrored,
    plots did not.
  - Horizontal bar: grew away from its own reversed axis.
  - Radar: labels swapped to the mirror spokes.
  - Donut, pie, gauge and funnel were fine.

### Decision (user)

Flip the canvas. It was built as **opt-in** (`core.CanvasMirrorsRTL`), not
default-on with an opt-out. That is Android's autoMirrored and SwiftUI's
flipsForRightToLeftLayoutDirection stance, and it keeps clocks and QR codes
fixed and the existing wire unchanged.

### What was built

- **core/canvas.go:**
  - `CanvasMirror` BehaviorProp and the `CanvasMirrorsRTL` const. It writes
    `mirror: true` only when set; `false` deletes it.
  - `MirrorCanvasMapping(sx, ox, boxW)` returns `(-sx, boxW-ox)`.
  - The doc covers why the canvas needs to be told, why opt-in, and what
    each target does.
- **Web runtime** (`applyCanvasProps`): `el.style.scale = "var(--grmob-inline,
  1) 1"` plus `ensureRule(TRANSLATE_DIRECTION_CSS)`, cleared when the prop
  goes. It uses the individual `scale` property because `transform` is
  Rotate's and `translate` is Translate's.
- **htmlout:**
  - `CanvasMirrorScale(props)` is exported. `canvasChassis` appends it.
  - `treeMotion` counts a mirrored canvas as needing the direction rule.
- **Compose** (`GrMobCanvas.kt`): `vp = base.mirrored(width)` when
  `mirror && layoutDirection == Rtl`, read inside the DrawScope.
  `CanvasViewport.mirrored` is in `GrMobCanvasGeometry.kt`.
- **SwiftUI:**
  - `@Environment(\.layoutDirection)` on GrMobCanvas.
  - `GrMobCanvasViewport.mirrored(width:)`, with a private memberwise init.
  - The mirror is applied over the unpadded box, so the outset translation is
    unaffected.
  - Gradients follow on both natives, since they're built from the viewport.
- **internal/canvasfixture:**
  - A new `Mirror` field; the old positional literals gained `false`.
  - Three mirrored cases: a stretch, a fit with slack, and a circle.
  - Threaded through `android/verify` (gen.go and Harness.kt) and `ios/verify`
    (gen.go and canvas.swift).
- **Adopted by:** Line/Area, Bar, Histogram, Scatter, Candlestick, Radar,
  Heatmap, Sparkline, Waveform (with a comment about the seek slider), and
  Rating (the half star).
- **Fixed on purpose:** Donut, Pie, Gauge, Funnel and QRCode. AnalogClock is
  not a Canvas at all (boxes turned by Rotate).
- **BarChart horizontal value labels:** the gap moved from
  `PaddingLeft`/`PaddingRight` to a fixed-width Box in a Row (`gapLead` or
  `gapTrail`). The web's padding is physical and the natives' is leading, so
  only a Row mirrors the same everywhere. This raised N-076.
  `TestHorizontalValuesFollowOrEnterTheBar` was updated to find the Text
  through `findFirst` and to assert the gap side.

### Tests

- `core/canvas_test.go`:
  - `TestCanvasMirrorIsOptInAndCanvasOnly`.
  - `TestMirrorCanvasMappingReflectsTheBox`.
- `comps/canvas_mirror_test.go`: the census.
- `wasm/verify`:
  - A gen.go case, "a mirrored stretched line with a gradient", with
    `Scale` from htmlout.
  - `canvas_test.mjs` compares `style.scale` and checks that update-props
    turns it on and off, with the rule added once.
- Browser check 21: two 200px canvases with a mark at x 0..10.
  - Under rtl, the mirrored mark must span 0.9..1 and the plain one 0..0.1.
  - Under ltr, both span 0..0.1.
  - Mutation-tested by blanking the runtime's scale.
  - `checknumbering_test.go` learned "twenty-one" (the regex now takes `[\w-]`).
- API docs regenerated (`go run ./internal/apidoc/gen`).

### Seen

- Headless Chrome under RTL, lessons 4.20 and 4.36: every chart matches its
  labels.
- Android emulator with a per-app Arabic locale
  (`adb shell cmd locale set-app-locales com.grmob.app --locales ar`, later
  restored to empty): 4.20's charts and 4.36's radar mirrored correctly.
  - `debug.force_rtl 1` did not take effect without a system restart, and was
    reset to 0.
  - Cold start with `am start -W -a VIEW -d grmob://lesson/X com.grmob.app`
    after a force-stop. A warm deep link to the running instance did not
    navigate.
- SwiftUI type-checks and its geometry is verified in `ios/verify`, but it is
  unrun.

## Verification

- `go test ./...`: all pass.
- `wasm/verify/run.sh`: all pass, including browser checks 20 and 21 in
  Chrome 152.
- `ios/verify/run.sh` and `android/verify/run.sh`: pass. "10 canvas drawings
  map and decode as Go's do", including the three mirrored ones.
- The view layer type-checks, and Compose compiles with the Compose plugin.
- `wasm/main.wasm` rebuilt (untracked).

## Next

Closed: N-063, N-067, N-075, N-071. Declined: None. Raised: N-076.
Deferred: None. Promoted: None.
Updated: None (N-071 was rewritten with its measurements before being
closed). Full list: `ai_docs/todo/next-list.md`.
