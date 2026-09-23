# Slider values, a centred radar, a zero basis that counts its border, Opacity seen

Session: `6f1f8c3e-a548-4afa-a0e4-d7afa0dfa322`

## Ask

Work the Next-list items that need no hardware device. Emulator, simulator and
headless Chrome count as "no hardware".

## N-079: a slider's spoken value was a percentage (closed)

### Diagnosis

`core.ValueRange.Text` was already honoured on any node by both natives:

- Compose: `grMobValue` sets `stateDescription`.
- SwiftUI: `grMobValueText` sets `accessibilityValue`. The Slider arm keeps
  the value on the Slider itself.

`comps.SliderRow` simply never stated one. The web exporters' `ariaValue`
guard also admitted `progressbar` only, so a Slider could not carry
`aria-valuetext`.

### Fix

- **`comps/slider_row.go`:** `track` adds
  `core.AccessibilityValue(core.ValueRange{Text: r.format(value)})` when the
  readout is non-empty. It carries Text only, never the numbers.
- **`htmlout/export.go`:** `ariaValue(s, nodeType)` gained a Slider arm ahead
  of the role switch that writes `aria-valuetext` alone. The section is "A
  Slider takes the words and nothing else".
- **`wasm/grmob-runtime.js`:** `ariaValue(style, nodeType)` has the same arm.
  `TestRuntimeGuardsTheValueTheSameWay` pins it.
- **Docs:**
  - `core.Slider` has a new section, "What a reader says for the value".
  - `core.Style.AccessibilityValue` notes the Text exception.
  - The RangeSlider doc now says "Minimum, slider, $20".
  - `docs/components.md` SliderRow has a new bullet.
  - The API docs were regenerated.
- **Tests:**
  - `TestSliderRowSaysItsReadoutAsTheSlidersValue`
  - `TestASliderTakesTheWordsAlone` (htmlout)
  - "a Slider takes the words alone" (`a11y_test.mjs`)
- **Heard on the emulator (TalkBack):** "$20, Minimum, Slider" and "$80,
  Maximum, Slider".
- **Simulator:** the combined "Price" element's value is "$20, $80", and
  "$130, $130" after the push. `testSwatchesHexShortFormAndTheRangeThatCannotCross`
  asserts on the group. The inner sliders XCUITest synthesizes under the
  combine still read 10%/40%; that is N-078.

## N-070: RadarChart 62pt left of centre on SwiftUI (fixed)

The `Player` image frame was x 20.8, w 236.7, so its centre was at 139 against
the panel's 201.

`GrMobStackSolver.containerSize` reported the largest layer (180pt, the
canvas). grMobBox's `.frame(width: 304)` around the layout then placed it at
`grMobFrameAlignment`, which is leading for a ZStack. The shift is
(304 − 180) / 2 = 62.

### Fix

- `containerSize(layers:proposing:fills:)`: on an axis the box is sized on,
  it reports `max(content, offer)`. An unspecified offer still gives the
  content.
- `GrMobZStack` passes
  `fills: (grow.fillWidth || grMobStatesExtent(width), …)`.
  `grMobStatesExtent` treats anything but "" and "auto" as sized.
- ios/verify's stack section has four new checks (fill, never below the
  content, unspecified, centred placement).
- The pin in `stacking_test.go` now reads `GrMobStackLayout(fills:`.
- **Seen:** the image is at x 82.8, centre 201. "Vision" reads whole.
- `testRoundFourChartsDraw` now scrolls to "Player:", skips `lift` for the
  radar, and dumps each chart.

## N-072: the iOS EDIT row's Category text sat about 3pt left (fixed)

`zeroBasisPadding` (`Renderer.swift`) started a zero-basis child from its
padding alone. EditableGrid's editing cell trades 2pt of padding a side for a
2pt ring, so it was 4pt short, and the weights spread the difference across
the row.

- It now uses `style.contentInsets`, which is padding plus a drawn border, as
  CSS counts a basis.
- Pinned in `TestIOSFlexHonoursAZeroBasis`.
- Seen aligned in `r4-4.37-draft.png`. Row 2's editor sits clear of the iOS
  keyboard.

## Device-free observations

- **N-062 / N-064, Compose (emulator, 4.34):**
  - Raw `adb exec-out screencap` samples at the three dots' centres
    (165/192/219, 1026).
  - At normal animation scale, the red channel passes through 34, 47, 59, 73,
    87, 111… between rest 145 and full 4–9, so the Opacity ease runs.
  - With all three animation scales at 0 it reads exactly two values, one
    dark dot at a time: the quiet blink.
  - Lesson 4.34's prose ("the background colour: core.Style has no opacity")
    was corrected.
- **N-066:** NumberPad's unpainted corner has no node in Compose's
  accessibility tree (uiautomator: the bottom row is 0 and Delete).
- **N-068, web:** headless Chrome's AX tree on 4.35 gives guide.md and api
  `listitem` level 2 inside docs at level 1. This came from a scratch CDP
  probe (`tree.mjs` / `treeax.mjs` in the scratchpad), not a browser check.
- **N-068, Wizard on TalkBack:** blocked by N-058. With Compose's focus on
  Skip and TalkBack then turned on, Enter pressed "‹ Contents". Injected taps
  don't reach the emulator's touch explorer.
  - 4.35's name field was empty ("Ada Lovelace" is the placeholder), so Next
    was disabled and rightly not a Tab stop.

## Verification

- `go test ./...`: pass. The API docs were regenerated.
- `wasm/verify/run.sh` (browser checks included): pass.
- `android/verify/run.sh`: pass.
- `ios/verify/run.sh`: pass.
- iOS UI tests:
  - All Tutorial* classes, 48 tests: 0 failures, run after both Swift
    changes.
  - A full 55-test run: 6 failures, all inside the non-tutorial tests
    (Audio, GrMobUITests, TodoApp). They need their own example app's
    framework, not the tutorial's.
- The tutorial AAR and APK were rebuilt and installed on the emulator. The
  iOS framework was rebuilt from `./examples/tutorial`, and `wasm/main.wasm`
  was rebuilt with `./build.sh`.
- **Emulator restored:** accessibility services null, `accessibility_enabled`
  0, Google TTS enabled, animation scales 1.

## Gotchas

- In zsh, `xcodebuild … $ARGS` with a space-joined string passes one
  argument ("Unknown build action"). Use an array: `ARGS+=(…)`, then
  `"${ARGS[@]}"`.
- There is no ffmpeg on this machine. To watch an animation on the emulator,
  sample `adb exec-out screencap` (raw RGBA; the header is
  `len − w·h·4` bytes) at a fixed pixel.
- The emulator's screen is 1080 × 2400. A `sips -Z 800` shot scales by 3.
- In XCUITest, the inner elements of a combined container report UIKit's own
  values, not SwiftUI's `accessibilityValue`. Assert on the combined element.
- The Monitor tool must be loaded through ToolSearch before use. A
  backgrounded `until` loop works instead.

## Next

Closed: N-079. Declined: None. Raised: None.
Deferred: None. Promoted: None.
Updated: N-058, N-062, N-064, N-066, N-068, N-070, N-072. Full list:
`ai_docs/todo/next-list.md`.
