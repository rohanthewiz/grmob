# N-076 logical insets, the hidden fill, iOS caret carry, a TalkBack sweep

Session: `ee96aa1f-0b41-45cc-ab5c-eeda6a41a765`

## Ask

Do N-076, then every other Next-list item that needs neither hardware nor a
user decision. Emulator and simulator runs counted as "no hardware".

## N-076: Left means leading on the web too

### The decision taken

The item offered two fixes: the web writes logical padding, or Left means
left and the natives change. The web was changed to match the natives:

- two of the three live targets already drew Left as leading (Compose
  `padding(start = left)`, SwiftUI `EdgeInsets(leading: left)`);
- text-align had gone the same way earlier (`docs/platforms/wasm.md`).

Corners stay physical on every target (`core/corners.go` already says so).
Position's Left/Right were not touched; the natives do not read them.

### What changed

- **Runtime** (`styleFromGrMob`):
  - `padding` and `margin` became `paddingBlock`/`paddingInline` and
    `marginBlock`/`marginInline`, from `edgeLogicalCSS(prop, edge)`. It
    replaced `edgeToCSS`, which had no other caller.
  - Block is "top bottom" and inline is "left right".
  - TextGrid's and CodeEditor's `margin || "0"` became
    `zeroMarginUnlessSet(out)`.
  - The code editor gutter writes `paddingInlineStart` in both places, and
    `basePaddingLeft` reads it. The editor is `dir="ltr"`, so it is the same
    edge.
- **htmlout:**
  - `EdgeLogicalCSS(e) (block, inline string)` is new in `edges.go`.
    `EdgeCSS` is kept, because bandfixture and the native harnesses parse its
    four numbers.
  - `export.go` writes `padding-block:…; padding-inline:…` and the margin
    pair.
  - `codeEditorPadding` writes `padding-inline-start:`.
- **core docs:** a "Left and Right are leading and trailing" section on
  `EdgeInsets`, and a line on each of the four side props. The API docs were
  regenerated.
- **cssstyle.mjs:**
  - The four logical pairs are modelled as `pair` shorthands over logical
    longhands. They were removed from UNMODELED.
  - New `DIRECTIONAL` table plus `refuseDirectionalMix`: an element given
    both physical and logical names for one box throws, because Chrome
    resolves such a mix by declaration order.
  - Six CSSOM_READS rows were added, including the gutter sequence in logical
    terms. Check 14 replays them in Chrome, and they pass.
- **Tests:**
  - `runtime_test.mjs`: the shorthand test now reads the logical pairs.
  - New runtime test: "an inset is written without a physical side…".
  - `styleless_test.mjs`: the TextGrid margin.
  - `codeeditor_test.mjs`: `paddingInlineStart` and `paddingBlockStart`.
  - `htmlout/export_test.go`: the new strings, plus
    `TestInsetsAreWrittenLeadingAndTrailing`.
  - `htmlout/codeeditor_test.go`.
- **Browser check 21** also mounts a Box with Padding Left 30 and Margin
  Left 7. Under rtl the computed right sides must be 30px/7px, and under ltr
  the left sides.
  - Mutation-tested: putting the physical shorthand back failed with
    `{"pl":"30px","pr":"0px",…}`.
  - The header comment and the OK sentence were updated. The check count is
    unchanged, so `checknumbering_test.go` was not touched.
- **Docs:**
  - `docs/platforms/wasm.md`: a paragraph after the text-align one.
  - `docs/components.md`: RadarChart's stale "the drawing does not mirror"
    now says it mirrors (a leftover from N-071).
- **BarChart** keeps its spacer-Row gap. Its comment now says padding would
  also work and the Row stays on purpose.

## N-064: the hidden fill on Compose

`display == "hidden"` was an `alpha(0f)` at the foot of `boxModifier`,
inside the shadow, background and border, so it faded only the content.

- It now shares the Opacity layer:
  `val layerAlpha = if (display == "hidden") 0f else opacity`. iOS's
  `.opacity(s?.display == "hidden" ? 0 : …)` already did this.
- `mobile/verify/opacity_test.go` now looks for the `layerAlpha` line.
- New `TestComposeHidesTheWholePaintedBox`. It uses `valuesIn`, because
  `codeIn` blanks string literals.
- Compiled (android/verify). Not watched on screen.

## N-065: the iOS caret carry, extracted and typed

- **GrMobTextEdits.swift:**
  - `carryPlan(a, b, caret:) -> CarryPlan{start, end, caret}` holds `write`'s
    arithmetic.
  - The span is the differing one, extended to the caret when the caret
    follows the change. The caret is `Carry`'s.
  - It gained Carry's surrogate/over-length guard.
  - `carryCaret(before:after:caret:)` wraps it.
- **GrMobTextInput.swift:** `write` calls `carryPlan` and keeps the UIKit
  half and its history comment.
- **ios/verify:** gen.go ships `CarryCases`, main.swift decodes them, and
  rebase.swift's `checkCarry` checks three things:
  - the caret;
  - that the splice reproduces `after`;
  - that the replacement ends at the caret whenever the caret sat at or
    after the differing span. The span is found independently in the
    harness.
- Two mutations were each caught: dropping the extension, and sending the
  inside-changed-length arm to the text's end.
- **XCUITest** `testMaskedInputFormatsAsItIsTyped` (in
  TutorialDevicePassUITests, lesson 5.9) passes on the iPhone 17 Pro
  simulator:
  - burst "5551234567" reads "(555) 123-4567";
  - "x" is refused;
  - 6 deletes give "(555) 1", because the "-" goes with the 4, and 1 more
    gives "(555";
  - the card typed a key a second reads "4242 4242".
  - The first run failed because the field sat at the screen's bottom edge;
    `lift(app)` fixed it. My first delete arithmetic was wrong, not the
    field.

## Device-free checks on the emulator and in headless Chrome

### TreeView under RTL (4.35)

- **Headless Chrome:** the scratch `rtlshots.mjs` from the last session was
  run over a fresh `wasm/main.wasm` build. The indent is mirrored.
- **Compose emulator:** per-app `ar` locale, cold deep link. The indent is
  mirrored.
- On both, the collapsed "▸" chevron still points right. This was recorded
  in N-068 as an API decision.

### TalkBack Tab sweep

Emulator recipe from memory. The utterance parser joins every `{text:`
fragment on the line; the earlier sed kept only the last one ("Button").

- **4.35:**
  - "expanded. docs. Expands or collapses the branch, Button"
  - "Selected, guide.md, Button"
  - "collapsed. api…"
  - src, assets and README.md were never spoken. With TalkBack off, a
    focused-node trace shows Compose's Tab visits all six rows, so this is
    N-058, not TreeView.
- **5.9:**
  - Pad keys: "2, Button" … "Delete, Button".
  - Swatches: "Not selected, orange, Radio button. 3 of 8. In list. 8 items".
  - The positions are wrong. The grid (from a uiautomator dump) is blue,
    orange, teal, yellow, pink, green / purple, red, but TalkBack said orange
    3, teal 5, yellow 6, pink 7, green 8, purple 1 and red 3. Raised as
    N-077.
  - With TalkBack off, Tab goes 9 → 0 → Delete: the corner is not a stop.
- **4.37:**
  - "Amount, row 2, $310.50. Edits the cell, Button"
  - "Row 1. Opens the row's menu, Button"
  - Category cells say "Category, row 1, Home" with no role.
- **4.34:** "Not selected, party popper, 1 reaction, Button".
- **Emulator restored:** accessibility services null, `accessibility_enabled`
  0, Google TTS re-enabled, app locales `[]`.

## Verification

- `go test ./...`: all pass. The API docs were regenerated with
  `go run ./internal/apidoc/gen`.
- `wasm/verify/run.sh`: all pass, including checks 14 and 21 in Chrome 152.
- `ios/verify/run.sh`: pass, with "11 writes into a focused field carry the
  caret as rebasefixture.Carry does".
- `android/verify/run.sh`: pass.
- `xcodebuild test -only-testing:…/testMaskedInputFormatsAsItIsTyped`: TEST
  SUCCEEDED.
- The tutorial AAR and APK were rebuilt and installed on the emulator. The
  iOS framework was rebuilt with `ios/build.sh ./examples/tutorial`, which
  must be run from the repo root.
- `wasm/main.wasm` was rebuilt (untracked).

## Gotchas

- The shell's `grep` is a function wrapper that hid matches in
  cssstyle.mjs. Use `/usr/bin/grep` when a result looks impossible.
- `node --test *.mjs` picks up `browser.mjs`. Use `*_test.mjs`, as run.sh
  does, or run browser.mjs with `GRMOB_TRANSCRIPT` set to the file run.sh
  writes.

## Next

Closed: N-076. Declined: None. Raised: N-077. Deferred: None. Promoted: None.
Updated: N-062, N-064, N-065, N-066, N-068, N-072. Full list:
`ai_docs/todo/next-list.md`.
