# Comps round four, phase 2: inputs (NumberPad, ColorSwatchPicker, RangeSlider, MaskedInput)

**Session:** b21e42e4-e4a7-4763-ae4e-4660640df6b8
**Date:** 2026-09-21 10:19
**Branch:** master (5322aa6 → this commit)

## The ask

1. `/sl` (loaded `2026-0921-0935-core-opacity-and-typing-indicator-fade` and
   the next list).
2. "Start phase 2 of the comps round four plan" —
   `ai_docs/plans/comps-low-hanging-fruit-4.md`, Phase 2: K4's spike first,
   then K1 `NumberPad`, K2 `ColorSwatchPicker`, K3 `RangeSlider`, K4
   `MaskedInput`, and one chapter 5 lesson.
3. `/sw`.

## What landed

| Piece | Files |
|---|---|
| K1 `NumberPad` | `comps/number_pad.go`, `_test.go` |
| K2 `ColorSwatchPicker` | `comps/color_swatch_picker.go`, `_test.go`, `comps/nested_composite_test.go` (on `closedComposites`) |
| K3 `RangeSlider` | `comps/range_slider.go`, `_test.go` |
| K4 `MaskedInput` | `comps/mask.go` (`applyMask`, `unmask`, `maskCapacity`), `comps/masked_input.go`, both `_test.go` |
| The caret rule, stated in Go | `internal/rebasefixture/carry.go`, `carry_test.go` (`Carry`, `CarryCases`) |
| Android caret fix | `GrMobTextEdits.kt` (`carryCaret`), `Renderer.kt` (`GrMobTextField`), `android/verify/gen.go`, `Harness.kt` |
| Web refused-key fix | `wasm/grmob-runtime.js` (`dispatchFromElement`, `noteGoValue`, `CONTROLLED_TEXT_TYPES`), `wasm/verify/fieldvalue_test.mjs` |
| Lesson 5.9 "Keypads, swatches, ranges and masks" | `examples/tutorial/chapter5.go`, `chapter5_test.go` |
| Docs | `docs/components.md` (four sections), `internal/apidoc/packages.go` (the five files on "inputs"), `docs/api/` regenerated |
| Lesson count 76 → 77 | README.md, docs/tutorial-interactive.md, `wasm/index.html`, `internal/shotclaims`, two tutorial test comments, `docs/images/tutorial-contents.png` re-taken |
| Census 640 → 652 tracked Go files | `wasm/verify/repowalks_test.go`, `timings_test.go` |
| Bookkeeping | the plan (status, four "What the build changed" blocks, order rows 2 and 3 struck), `ai_docs/todo/next-list.md` |

`go test ./...`, `wasm/verify/run.sh` and `android/verify/run.sh` pass.

## The round's rule was broken twice, on purpose

The plan says a phase touches no file under `htmlout/`, `wasm/`, `android/` or
`ios/`, and that an item needing a renderer leaves for Phase 6. Phase 2 touched
two. Both are host bugs the widget exposed and not features it needed, and
each fix is the rule the other hosts already followed.

### K4's spike: the premise was wrong, usefully

The plan feared typing at speed would lose keys, because "the ledger was built
for a rewrite per tag". The ledger had moved on since that was written: the
hosts do a three-way merge (`internal/rebasefixture`), not an ends-only replay.
A throwaway app (`examples/zzmaskspike`, deleted) with `applyMask` in an
`OnChange`, on the Android emulator:

| Test | Before the fix | After |
|---|---|---|
| `5551234567` by `adb shell input text` | `5514567325` | `5551234567` |
| `1 2 3 4 5 6`, a second apart | `(234) 651` | `(123) 456` |
| 16-digit card, machine speed, 3 runs | — | whole, 3/3 |
| 4 backspaces from a full phone number | — | `(555) 123` |

Scrambled even at one key a second, so not a race: **the Android field kept
the caret's raw offset across a rewrite.** After `1` → `(1` the caret sat
between the bracket and the digit. The web (`writeFieldValue`) and iOS
(`write`) already carry the caret across Go's change. No earlier rewrite could
show it, because every earlier one changed text at or after the caret
(TagInput) or kept the length (UPPERCASE).

`rebaseCaret` was not the answer for a plain field: with nothing in flight
(local equals basis, the ordinary case) it sends the caret to the end, which
breaks UPPERCASE mid-text. So the rule is from **two** texts, not three:
`rebasefixture.Carry(before, after, caret)` reads before→after as one span and
maps the caret with the existing `mapOffset`; inside a span that changed
length it goes to the end of the new span. `carryCaret` is the Kotlin copy,
run against 11 `CarryCases` by `android/verify`. Lesson 2.3's UPPERCASE field
was re-run mid-text on the emulator afterwards: `HELLOABCWORLD`.

**The limit that remains** (documented on the widget): a key typed mid-text at
the very end of a group (`(555|) 123`) reflows everything after it, the one
differing span then includes the caret, and the caret lands after the reflow.
Nothing is lost. Typing anywhere else mid-text keeps its place
(`(5|55) 123` + `9` → `(59|5) 512-3`, measured on both hosts). A real fix
needs a caret position in the protocol.

### The web: a refused key left no trace

Found by typing through real `input` events in headless Chrome (`shoot.sh
--probe`; `typeInto` calls Go directly and would not have shown it). A key the
handler refuses changes no state, the next render equals the last, no patch
arrives, and the key stayed drawn: `(555) 123-4567x`. The natives catch this
through the ledger (the host's text differs from the render, so it is a
rewrite) — now pinned by `TestMaskedInputRefusedKeyIsRewrittenOnANativeHost`
through `render.Manager.DispatchTextEdit`.

`dispatchFromElement` records the text Go last rendered (`noteGoValue`, on
create and on every value patch, echoes included) and, after an `onChange`
dispatch returns, gives a text field Go's text back if it differs. Safe
because the call is synchronous. `NumericInput` is excluded: a
`type="number"` input's `.value` reads `""` mid-parse. Four cases in
`fieldvalue_test.mjs`. This also makes a PINInput-with-no-OnChange actually
"paint Value back", which its doc already claimed.

## Where the widgets left the sketch

All written into the plan, under each item. The ones worth carrying:

- **K1.** Keys are equal shares ≥56pt, not 1:1 (core has no aspect ratio). The
  blank corner is **a key that is not painted** (`Disabled`, hidden,
  `core.Opacity(0)`), see the look below. `OnBackspace` nil disables that key
  alone. The handler guards `Disabled` itself for the tap that races the patch.
- **K2.** Default palette is `ChartColors()`, not `chartPalette` (whose second
  half is tints). Defaults are named from their hue by `colorName` (eight
  bands + light/dark/grey/black/white; the eight default chart colours come
  out as eight names, pinned). The ring is a border every swatch carries,
  transparent when unselected. **The short hex form commits on return only**:
  `#7B2` is a valid colour on the way to `#7B2FF0`, and the first test caught
  it being reported. Holds one hook, taken whether or not `AllowCustom`.
- **K3.** As sketched. An inverted pair is drawn the right way round and
  reported (`ConcernRangeSliderInverted`). The range in words is hidden from
  accessibility: the sliders state the same numbers as values.
- **K4.** **Literals are written late** (`555` draws `(555`): written eagerly,
  backspace on `(555) ` formats straight back to itself.
  `TestMaskBackspaceAlwaysBites` holds it. **`unmask` walks the text against
  the mask** rather than stripping non-slot characters, so `+1 (###) …` does
  not read its own literal `1` as data and grow by a `1` per keystroke; the
  price (a raw value cannot begin with such a literal) is in the doc. `Value`
  is the raw value. Takes no hooks.

## What the look found

Fourth round running (G1, G3, J3, now K1/K2) that looking found what asserting
did not.

1. **The pad's zero sat off-centre under the eight, and the swatch grid's
   last row was wider than the rest.** On the web a flex share is the zero
   basis *plus the cell's own padding and border*, so an empty `Box` blank was
   narrower than a `Button`. The pad's blank is now an unpainted key, which
   has a key's box by construction; the swatch blanks carry a swatch's padding
   and ring border.
2. The `RangeSlider` code block ran off the frame. Reformatted.

On the Android emulator (tutorial rebuilt, `grmob://lesson/5.9`): pad and
swatches drawn right, two keys tapped, the status line read
"Passcode, 2 of 4 entered" in the accessibility tree. The pad's corner is
**`core.Opacity`'s first sighting on a device**: a static 0, no fade watched
(noted on N-064).

How the probe was taken, for next time: a script under `wasm/shots/scripts/`
that ends `return out;`, run with `./shoot.sh --probe <name>`. A real
keystroke is `el.value = …; el.setSelectionRange(…); el.dispatchEvent(new
Event("input", {bubbles: true}))`. Synthetic `KeyboardEvent`s reach the
composite key handler.

## Noticed, not touched

- **A radiogroup's arrows follow one axis on the web** (N-067). The picker is
  a column of rows, so Up/Down walk the swatches and Right does nothing.
- N-063's two stale statements are still there.

## Not run

- iOS: nothing. No Swift changed, so `ios/verify` was not run; the iOS field's
  `write` is *read* to follow `Carry`'s rule and is not held to `CarryCases`.
- No real phone; no composing IME (`adb shell input text` commits whole keys).
- The emulator now has the tutorial app installed (it replaced the spike app).

## Next

Closed: None. Declined: None. Raised: N-065, N-066, N-067.
Deferred: None. Promoted: None.
Updated: N-064. Full list: `ai_docs/todo/next-list.md`.
