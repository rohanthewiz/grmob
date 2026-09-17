# D2: comps.QRCode, and a QR encoder of our own

**Session:** 11453ed0-46ac-41cb-b3fc-9b7ee88acb83
**Date:** 2026-09-17 17:26 (follows "fab-screen-floating-and-round-two-plan")
**Branch:** master (af6e9f4 → this commit)

## 1. The ask

"start D2, the QRCode comp" — the next item in
`ai_docs/plans/comps-low-hanging-fruit-2.md`, which the previous session wrote.

## 2. The decision the plan left open: vendor or write

The plan named `github.com/skip2/go-qrcode` (MIT, small, gives a bitmap) and
said to write one "only if the dependency policy says so". There is no written
policy, so the repository was asked instead: `go.mod`'s only runtime
requirements are `rohanthewiz/bytdb` and `rohanthewiz/element` — this author's
own packages — plus the gomobile toolchain held by the `tool` block. Vendoring
would have made an encoder the **first third-party package any widget in this
framework required**, and it carries an `image/png` bitmap model a `Canvas`
only has to undo.

A QR encoder is also a closed, fully specified algorithm with published test
vectors, which is the kind of thing cheaper to own than to track. So:
`internal/qr`, about 700 lines.

## 3. `internal/qr`

Byte mode only, one segment, versions 1–40, levels L/M/Q/H, the eight data
masks with the standard penalty scoring.

- **Byte mode only** is not a simplification that costs anything here: the
  caller encodes URLs and deep links, and `cats://pair?...` is lowercase, so
  alphanumeric mode (uppercase, digits and nine punctuation marks) could not
  take it anyway. Byte mode makes the API total — no input is rejected for its
  characters, only for length.
- **The capacity table is derived, not tabulated.** The commonly printed
  160-row table is replaced by two block tables plus `rawDataModules(v)`, which
  computes the leftover area from the geometry. That is what turned up the one
  real bug in this session (§6).
- **Structured append and ECI are out.** Both change the API from "a string"
  to "a message", and neither's readers are targeted.
- `gfMul` is Russian-peasant multiplication in GF(2⁸) mod 0x11D rather than
  log/antilog tables: a symbol's worth of multiplications is in the tens of
  thousands, far below where a 512-byte table earns its keep, and there is no
  zero special case to get wrong.

## 4. `comps.QRCode`

```go
comps.QRCode{Data: "cats://pair?t=9f2c1a", Label: "Scan to pair this device"}
```

`QRCode{Data, Size, Level, Quiet, Label, Style}` over `ECLevel`, a string enum
with an empty zero value (the package's idiom, per `Variant`) whose values are
the specification's own `L`/`M`/`Q`/`H`.

**One path, not a rectangle per module.** A version-10 symbol has some three
thousand modules. Three thousand child nodes is the reconciler's worst case for
a drawing that is either identical between passes or wholly different — and
separate shapes are antialiased *against each other*, leaving hairlines between
adjacent modules that a decoder's binarizer can read as light. A single filled
path has no interior seams. Within a row, consecutive dark modules merge into
one rectangle: one comparison per module, and it typically halves the path.

**The quiet zone is inside the box.** `Size` is the whole square, so the symbol
is `Size × n/(n+2·Quiet)` across. Outside it, the widget's footprint would
depend on how long the data turned out to be.

**Colour, where the plan had to be overruled.** The plan said "the theme's text
colour on the surface colour, never inverted" — the two halves contradict each
other in a dark theme. Settled as: the theme's own pair **only when it is
already dark-on-light with room to spare** (surface luminance > 0.6, ink <
0.15), otherwise black on white. In a dark theme that is a white square, which
is what every banking and payment app shows, for this reason. No `Foreground`
or `Background` field: every colour a caller could pass is either the pair
already chosen or a worse one, and an unscannable code fails *silently* — it
looks exactly like a working one.

**Data too long** keeps the box (so the screen does not reflow when it is
fixed), draws nothing, and reports `comps.ConcernQRDataTooLong` through
`core.ReportConcern` in debug builds. There is no half of a QR code worth
showing: a truncated one still scans, just to the wrong thing.

**`ECMedium` is the zero value, not `ECHigh`.** The failure a code on a screen
faces is not damage — the glass is pristine — but module size: more redundancy
means a bigger symbol at a fixed drawn width, means smaller modules, which is
what a camera struggles with.

## 5. How it is checked

Three layers, because a QR encoder that is subtly wrong still draws something
that looks exactly like a QR code.

1. **Against published values.** The standard's own printed format strings
   (`L`/mask 0 = `111011111000100`) and version strings (v7, v40), its byte
   capacities, its alignment coordinates. Plus the two BCH codes' minimum
   distances — 7 over the 32 format strings, 8 over the 34 version strings —
   which a wrong generator or XOR mask would collapse.
2. **Against the definition.** A Reed-Solomon codeword *is* a polynomial
   divisible by the generator, so every codeword must vanish at α⁰…α^(n−1).
   That check uses only multiplication, so it is independent of the division
   code it tests.
3. **Round trip.** A decoder in the test file reads the format information back
   out of the finished grid, undoes the mask, walks the zigzag, de-interleaves
   the blocks and unpacks the segment — for all forty versions and every level.
   It is the only way to cover the placement chain, which has no published
   vector to check piecemeal.

Then, on a real render: a scratch shot script (not committed) drove the
tutorial to the new lesson and handed the rendered SVG to **Chrome's own
`BarcodeDetector`**. All four levels decoded exactly, for a 50-byte payload and
for a 313-byte one — the second past both the 16-bit character-count field and
the version-information block, which are the two things only the larger
versions exercise.

## 6. What the cross-check caught

`numBlocks[High][8]` was 5 where the standard says 6. Nothing self-consistent
would have found it: the symbol still encoded, still round-tripped, still
scored masks — it was simply a *different* version-8-H symbol than any other
implementation builds, and no reader would have decoded it. The published byte
capacity for v8-H (84) is what disagreed; the derived one said 110.

Two of the test's own expectations were wrong rather than the code: v9's
figures (I had written the data-codeword counts, 182/132/100, where the byte
capacities are 180/130/98) and v31's alignment coordinates (I had copied v33's
row, which does not fit in a 141-module symbol).

## 7. A render that was 0.9 ms

The first working version took 915 µs per render pass — a twentieth of a frame,
for a widget a pairing screen with a ticking countdown would redraw every
second. The cost was the mask search's rule-3 scan, which rebuilt an
eleven-module window at every position through a direction-branching closure:
some 1M closure calls per encode.

Rewritten to slide an eleven-bit register one module at a time over a line
copied into a scratch buffer — each module read once, the direction branch
hoisted out of the loop. **915 µs → 190 µs.** That is under a hundredth of a
frame and the widget's doc now says so, and points at `core.Cached` for a code
in a tree that really does redraw every frame (`QRCode` fits: no hooks, no
callbacks).

## 8. Docs, lesson, fallout

- `docs/components.md`: a `## QRCode` section after Compass. `docs/api/`
  regenerated; `qr_code.go` registered under "Data display & maps" in
  `internal/apidoc/packages.go` (the generator refuses a file no topic lists).
- Tutorial lesson **4.23 "Codes a camera can read"**, appended for the reason
  4.15 and 4.22 were. The demo is the error-correction level, because that is
  the one field whose effect is visible rather than argued. Driving test
  `TestQRCodeLessonRedrawsTheSymbolPerLevel` watches the Canvas's viewBox —
  the symbol's module count is a prop, not anything in the tree's text.
- "0 of 61 lessons opened" → 62 in README.md, docs/tutorial-interactive.md,
  `examples/tutorial/screenshot_test.go`, `internal/shotclaims/*`;
  `docs/images/tutorial-contents.png` re-taken.
- The file census in `wasm/verify` went 556 → 560 across five sentences.

## 9. Not verified

The symbol on the two natives. The four renderers share the Canvas path
contract and `comps` pins the shapes it emits, but "adjacent subpaths of one
fill leave no seam" has only been seen on the DOM. If Compose or SwiftUI
hairline between module runs, the fix is in `qrModulePath` — merge vertically
too, or overlap the rectangles by a hair — not in the renderers.

## 10. Next

D4 per the plan's order: `SelectRow` and `SliderRow`, the rest of the
settings-row family beside `SwitchRow` and `CheckboxRow`. No decisions to
settle; both are shapes already in the package.
