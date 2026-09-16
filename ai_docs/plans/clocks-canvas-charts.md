# Clocks, an in-app alarm, core.Canvas, and charts

**Date:** 2026-09-16
**Status:** Tiers A, B and C landed 2026-09-16, with tutorial lesson 4.19
("Clocks, drawing and alarms") exercising all three. Tier D (charts) landed
the same day with lesson 4.20 ("Charts"). Tier E (scheduled/system alarms) is
not started.

What landed, and where it differs from the sketches below:

- **A** — `hooks/now.go` (`UseNow`), `comps/clock.go` (`DigitalClock`,
  `AnalogClock`). The DigitalClock width floor was dropped: `MinWidth` on a
  Text is not honoured the same way on every target, and centring hides the
  pixel of jitter. Checked by eye in Chrome via htmlout.
- **B** — `core/canvas.go` gained `CanvasMapping`, the Go statement of the
  viewBox mapping, because the natives needed an authority to be held to.
  `Sector` (pie/donut wedge) and `Polyline` were added for Tier D.
  Renderers: `htmlout/canvas.go`; the canvas section of
  `wasm/grmob-runtime.js`; `GrMobCanvas.kt` + `GrMobCanvasGeometry.kt`;
  `GrMobCanvas` in `Renderer.swift` + `GrMobCanvasGeometry.swift`. The
  geometry files import nothing SwiftUI/Compose so the harnesses run them.
  Verification: `wasm/verify/canvas_test.mjs` (runtime attrs == htmlout's, via
  gen.go `canvasCases`), `internal/canvasfixture` run by `android/verify`
  (JVM) and `ios/verify` (macOS), `nodetypes_test.go` for the dispatch arms,
  and a live Chrome check of both DOM targets. Not yet seen on a device or
  simulator.
- **C** — `alarm/` (pure time arithmetic, DST-aware: `time.Date` resolves a
  spring-forward gap to the earlier offset, so `Alarm.on` recomputes),
  `hooks/alarms.go` (`UseAlarms`, `AlarmRinger` with Snooze/Dismiss, a queue
  for alarms due while one rings, RingFor timeout), `comps/alarm.go`
  (`AlarmRow`, `AlarmRinging`). Rang, snoozed and self-disabled in the live
  WASM tutorial.
- **D** — `comps/chart.go` (shared: `ChartSeries`, Heckbert nice ticks in
  `niceScale`, k/M tick units chosen per axis, the theme-role palette with
  60%-alpha tints past the roles, y labels placed by an exact Gap and a
  half-line spacer, x labels by FlexGrow weights + zero FlexBasis),
  `sparkline.go`, `line_chart.go` (`LineChart`, `AreaChart`), `bar_chart.go`,
  `donut_chart.go` (`DonutChart`, `PieChart`, largest-remainder percentages),
  `gauge.go`. No renderer changes. Differences from the sketch: the spoken
  lead-in is a `Subject` field (not drawn); dots and points are round-capped
  zero-length strokes, because a Circle under CanvasStretch is an ellipse; the
  donut's track is drawn only when there is no data (it showed through the
  slice gaps); the gauge box is square so its text centres on the arc. No
  hidden data table yet. Checked in Chrome through htmlout and live in the
  WASM tutorial (Shift and "Use 12%" re-render and re-summarise). Not seen on
  a simulator or emulator.
- Adding lesson 4.19 moved the lesson count to 58 in every copy (README,
  site page, docs, shotclaims) and re-took `docs/images/tutorial-contents.png`.

Order of work: Tier A (clocks), Tier B (core.Canvas), Tier C (in-app alarm),
Tier D (charts). Scheduled/system alarms are Tier E.

## What exists today (constraints the plan works within)

- `core.ZStack` + `core.StackAlign` draw layers over each other on all four
  targets; `comps.Compass` is the precedent for a dial built from them.
- `core.Rotate(deg)` is a paint-time turn about the node's **own centre** (no
  transform-origin, by design — see `core/style.go`). `core.Transition`
  animates it natively. `core.AngleDelta` exists for unwrapped angles.
- `hooks.UseInterval` / `UseIntervalWhile` tick a render; ticks start at mount,
  not on a wall-clock boundary.
- There is **no vector primitive**. Nothing can draw a line between two points,
  an arc, or a filled polygon on all four targets. Charts beyond bars need one.
- A new node type costs four renderers — `htmlout` (`tag.go`, `export.go`),
  `wasm/grmob-runtime.js`, `android/.../Renderer.kt`,
  `ios/GrMob/Runtime/Renderer.swift` — and is held to them by
  `mobile/verify/nodetypes_test.go` (native dispatch arms against htmlout's tag
  census) and `wasm/verify` (runtime tag table against htmlout's).
- `core.LocalNotification` posts immediately; there is no scheduled trigger,
  and iOS suspends a backgrounded app within seconds. So an alarm that rings
  with the app closed is out of scope until Tier E.

## Tier A — clocks (no renderer changes)

### A1 `hooks.UseNow(ctx, every time.Duration) time.Time`

The clock widgets take a `time.Time` field and never tick themselves — the same
split as `comps.Compass{Heading}` + `hooks.UseHeading`. That keeps the widgets
pure (testable with a fixed time, usable for "time in Tokyo", a stopwatch, a
replay), and puts the one ticker where the app decides how often it renders.

Unlike `UseInterval`, the ticker is **aligned to the wall-clock boundary**
(`every` = 1s ticks at :00.000 of each second, 1m at the top of each minute).
An unaligned 1 s ticker shows a second that is up to 999 ms stale, which is
visible next to the phone's status-bar clock.

### A2 `comps.DigitalClock`

`Time`, `Hour24`, `ShowSeconds`, `ShowDate`, `Size`, `Color`, `Style`,
`AccessibilityLabel`. AM/PM drawn smaller beside the digits. Spoken label is a
sentence ("10:42 PM"), and the widget is one accessibility element.
Known limit: `core.Style` has no font family, so digits are proportional and
the width can shift by a pixel as they change (left as is; see Status).

### A3 `comps.AnalogClock`

A `ZStack` of dial-sized layers:

```
  layer 0  face        circle Box (radius = size/2), border
  layer 1..12 ticks    dial-sized transparent Column, Rotate(30·i),
                       a short stick at its top
  layer    numerals    same, with the Text counter-rotated (-30·i) to stay upright
  layer    hour hand   dial-sized Column, Rotate(angle):
  layer    minute hand     ┌──────────┐
  layer    second hand     │  spacer  │  height = size/2 − length
  layer    hub             │    ┃     │  the stick: its bottom sits on the centre
                           │    ┃     │
                           │    · ────┼── centre = the layer's rotation origin
                           │          │
                           └──────────┘
```

A hand pivots at the dial's centre because the *layer* is what turns, and the
layer's centre is the dial's centre — the workaround `Style.Rotate`'s doc
prescribes for "no transform-origin".

Fields: `Time`, `Size`, `ShowSeconds`, `Numerals`, `MinuteTicks`, `Smooth`,
colours (`Face`, `Ink`, `SecondColor` default theme Primary), `Style`,
`AccessibilityLabel`.

**Winding.** With `Smooth` a `Transition` animates each step, so angles must not
wrap at 60 s / 60 m / 12 h or the hand unwinds backwards. Angles are unwrapped
across the local day (second hand ≤ 518 400°, which Compose's Float holds to
~0.03°). At midnight the day rolls over; the pass whose time is in the first
second of the day omits the Transition so the hands jump instead of spinning
back 1 440 turns. Without `Smooth` (the default: a quartz tick) no Transition is
emitted and none of this arises.

## Tier B — `core.Canvas` (four renderers)

### Shape of the API

```go
p := core.NewPath().MoveTo(0, 80).LineTo(40, 20).CubicTo(...).Arc(cx, cy, r, 0, 90).Close()

core.Canvas(100, 100, []core.Shape{
    {Path: core.Circle(50, 50, 48), Fill: "#fff", Stroke: "#333", StrokeWidth: 2},
    {Path: p, Stroke: t.Colors.Primary, StrokeWidth: 2, Cap: core.CapRound},
}, core.Width("100%"))
```

- `Canvas(w, h, shapes, props...)` — `w × h` is the drawing's coordinate space
  (a viewBox), not its size on screen. The box is sized by Style like any node.
- `CanvasScale` prop: `CanvasFit` (default; uniform, centred, like an SVG's
  `xMidYMid meet`) or `CanvasStretch` (each axis independently — a chart filling
  a wide box).
- **Stroke widths are in layout units (px/dp/pt), never scaled.** A 2 px line
  stays 2 px on a stretched chart; SVG gets `vector-effect: non-scaling-stroke`,
  the natives stroke a path they transformed themselves.
- Height left unset takes the viewBox's aspect ratio (CSS `aspect-ratio`,
  Compose `Modifier.aspectRatio`, SwiftUI `.aspectRatio`).

### The wire: four opcodes only

`Path` records `MoveTo`, `LineTo`, `QuadTo`, `CubicTo`, `Arc`, `Close`, but
**Go flattens everything to M / L / C / Z** before it reaches a renderer:
quadratics are raised to cubics, and arcs become ≤ 90° cubic segments
(k = 4/3·tan(θ/4)). Each renderer therefore implements four drawing calls that
all three platforms have natively, and arc conventions (SVG's endpoint form vs
Compose's rect+angles vs SwiftUI's centre+radians and flipped clockwise flag)
never have to be reconciled in three languages.

A path travels as one flat `[]float64`: `0 x y` move, `1 x y` line,
`2 x1 y1 x2 y2 x y` cubic, `3` close. Angles in the Go API are degrees,
clockwise from 3 o'clock — the same "positive = clockwise" as `Rotate`.

### Node shape

`Canvas` is a container node with one `CanvasShape` child per shape, the
`TextGrid`/`GridRow` pattern: the reconciler pairs children by index, so a clock
whose second hand moved sends one prop patch, not the whole drawing.

| Target | Canvas | CanvasShape |
|---|---|---|
| htmlout / WASM | `<svg viewBox preserveAspectRatio>` (`createElementNS`) | `<path d fill stroke ...>` |
| Compose | `Canvas(Modifier)` drawing each child's `Path` with a scale matrix | read as data |
| SwiftUI | `Canvas { ctx, size in }` with `Path.applying(transform)` | read as data |

### Accessibility

A bare Canvas is decorative (hidden) unless it carries an
`AccessibilityLabel`, in which case it is one `img` element. Chart widgets
always label it and may add a hidden data table.

### Non-goals for v1

Text inside a canvas (labels and legends are ordinary `Text` around it —
platform text rendering in a canvas is where the three targets disagree most),
gradients, clipping, hit-testing individual shapes, even-odd fill rule.

### Verification

- `core`: path flattening tests (arc endpoints, sweep splitting, quad→cubic).
- `htmlout`: golden SVG output; tag census gains `Canvas`/`CanvasShape`.
- `wasm/verify`: runtime creates SVG-namespace elements and patches `d`.
- `mobile/verify/nodetypes_test.go` forces the two native arms.
- `ios/verify` type-checks the Swift; `android/verify/sources.sh` compiles the
  Kotlin when a compiler is available.

## Tier C — in-app alarm

- `alarm` package (pure Go): `Alarm{ID, Hour, Minute, Label, Days, Enabled}`,
  `Next(after)`, `Due(a, from, to)` — "did this alarm's ring time fall in
  (from, to]", which is robust to a skipped or late tick, unlike `now == time`.
- `hooks.UseAlarms(ctx, alarms, onRing)`: a 1 s wall-aligned ticker that keeps
  the last checked instant and calls `onRing` for each alarm due since then.
- `comps.AlarmRow` (time, label, days, `core.Switch`) and `comps.AlarmRinging`
  (label, Snooze, Dismiss). Ringing plays a caller-supplied track through
  `core.AudioLoad`/`AudioPlay` and pulses `core.Haptic`.
- Limit, stated on the type: rings only while the app is running and on screen.

## Tier D — charts on core.Canvas

`Sparkline`, `LineChart`/`AreaChart`, `BarChart`, `DonutChart`/`PieChart`,
`Gauge`. Pure Go in `comps`: scales and "nice" axis ticks computed in Go,
axes/legends as `Text` around a `Canvas`, series colours from theme roles, one
spoken summary per chart.

## Tier E — later

Scheduled local notification (`LocalNotification.At`) for alarms that ring with
the app closed; iOS AlarmKit / Android `AlarmClock` for true system alarms.
