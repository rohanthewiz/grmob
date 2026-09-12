# Session: two editors, a lexer, a document, and the hook that had to be the caller's

Session: https://claude.ai/code/session_018HHEppckHrYoZNwpi2jgha
Date: 2026-09-11 · Previous:
`2026-0911-1915-the-tags-stopped-a-level-too-high-and-a-noise-floor-worth-more-than-the-fix.md`

## Ask

> pls impl the plan in @ai_docs/plans/components-editors.md

Both tiers, in the order the plan's gates set. A1 → A2 (web + htmlout → iOS →
Android) → A3 → A4, then B1 → B2 (same order) → B3 → B4. Everything landed;
`go test ./...`, `wasm/verify/run.sh`, `ios/verify/run.sh` and
`android/verify/run.sh` are all green.

The four sentences worth keeping if the rest is lost:

1. **The toolbar had to name a ref rather than be a bool.** The plan's
   `Toolbar: true` would have made every `components.CodeEditor` a hook-slot
   consumer, and the plan's own closing line for A3 asks the tutorial's
   `codeBlock` to become one — a thing rendered inside conditionals and loops.
   Inverting it (the caller makes the ref, and naming it *is* asking for a
   toolbar) made both widgets hook-free in their display half.
2. **Every rich-text command is a pure transformation of the document.** The web
   runtime reads the selection as two character offsets, transforms the Doc,
   rebuilds and restores the caret. The Range is only ever read. That sidesteps
   the plan's own named risk (browsers splitting nodes under `contenteditable`)
   and makes the whole command vocabulary testable against a DOM with no
   Selection API.
3. **An editor adopts a standing command epoch without running it** — the one
   place an editor command deliberately differs from `core.Focus`, which
   re-fires on mount. Without it, returning to a screen re-indents its buffer.
4. **A `CodeEditor`'s filler rows are this runtime's first trailing chrome**, so
   `children.length - chromeOffset(el)` stopped being the node-child count.
   `nodeChildCount` counts children carrying a path instead.

---

## What landed

### A1 — `highlight/` (new package, module root)

`go/scanner` over Go, a deliberately tolerant hand lexer over JSON, `Plain()`,
and `ForLanguage(name)`. All four produce `[]core.GridRow` against a named
`Scheme` (`Darcula`, `Light`).

One hard contract, checked over every lexer against a corpus that includes
half-written input, CRLF, multibyte and mis-matched languages:

```
len(Rows(src, scheme)) == strings.Count(src, "\n") + 1
```

Both consumers address rows by line — `core.TextGrid` pairs them by index, and
the stale-line rule compares row *N* with line *N* — so line-for-line is the
property everything else rests on. Three more properties are pinned across every
lexer at once: the rows reassemble the source byte for byte, runs are maximal
(no two neighbours share a style), and no run is empty.

The JSON lexer is a hand lexer *because* it must not fail: a buffer mid-edit is
invalid JSON most of the time, and `encoding/json` answers "invalid" for all of
it — a document that loses every colour the moment somebody puts a cursor in it.

The tutorial's `highlight.go` is now a caller. Its own tests still pin that the
Darcula rows are byte-identical; the one test that reached into the private
scanner now asks the same question of the public API (does this snippet come
back identical to `highlight.Plain()`'s rows, which is exactly what the fallback
returns).

### A2 — `core.CodeEditor` + `core/editor.go`, four renderers

`core/editor.go` carries what both editors share: `EditorRef`, `UseEditorRef`,
`EditorTarget`, `RunEditorCommand`, `OnSelectionChange`, `ReadOnly`,
`Placeholder`, and the selection parse.

The one way the editor epoch differs from the focus epoch is worth restating:
focus state is per-app because a dismiss has to reach every field Go was never
told about, and an editor command *names* its editor — so the epoch lives on the
ref and exactly one node stamps anything.

Each host:

| Target | What it is |
|---|---|
| htmlout | The same `<pre>` a `TextGrid` is, plus a gutter as leading chrome. Rows stay direct children so both DOM targets are structurally identical. |
| WASM | A transparent `<textarea>` over a mirror of the grid's rows, inside one `<pre>`. Tab/Enter taken from the browser; `readOnly` leaves the tab order. |
| iOS | `UITextView`, rows applied as attributes over the existing characters — never `attributedText =`, which moves the caret. Gutter is a sibling `UILabel`. |
| Android | `BasicTextField` + a `VisualTransformation` with `OffsetMapping.Identity`; gutter is a `Column` sharing one vertical scroll. |

Rule 2 (decoration is advisory *per line*) is the part that took the most care
on the web, because it has to be **reversible**: a line that disagreed and then
agrees again must get its colours back with no patch from Go. The runs are
therefore stored on the row element and the decision is re-made on every sync,
rather than the row being repainted when a patch happens to arrive.

Filler rows are the other half: press Enter and the buffer has a line Go has no
row for yet. They are trailing chrome, which is what forced `nodeChildCount`.

### A3/A4 — `components.CodeEditor`, lesson 4.13, docs

Runs the highlighter, picks `Darcula` or `Light` from the theme's own
`Background`, builds the toolbar. **No hooks.** The tutorial's `codeBlock` is now
a read-only one and dropped its private `TextGrid` path; `codeBg`/`codeInk` went
with it, so nothing in the tutorial chrome hard-codes a colour any more.

### B1 — `richtext/` (new package, module root)

Seven block kinds, six marks, a flat sequence of blocks (a list is a *run* of
blocks, not a container — which is what keeps the model mappable onto three
native text engines that have flat paragraphs with paragraph styles).

JSON is the wire *and* the storage. `markFlag` writes `1` and reads anything
truthy, which is the asymmetry that lets a host whose JSON library writes bools
for bools be understood rather than silently unformatted.

Markdown loses exactly two things and both are pinned by tests: an **empty
paragraph** (a blank line is Markdown's block separator and nothing else) and
the **other marks on a code span** (a code span's content is literal by
definition). Underline has no CommonMark spelling and is written as `<u>`.

### B2 — `core.RichTextEditor`, four renderers

Shares the echo guard (compared on the doc's JSON, since Go marshals with a
fixed key order) and the command epoch. The stale-line rule is absent and does
not need to be: the doc **is** the styled buffer.

| Target | What it is |
|---|---|
| htmlout | `richtext.Doc.HTML()` in the node's box, read-only, plus the placeholder. |
| WASM | `contenteditable` div; every command a pure Doc transformation; paste intercepted and re-done as text; undo is a stack of Docs. |
| iOS | `NSAttributedString`, with two custom attribute keys so the block kind and the drawn list prefix are *read* rather than inferred. Marks edit `textStorage` in place; block kinds take the long way round. |
| Android | `EditText` + `Spannable` through `AndroidView` — the one deliberate reach past Compose. An empty selection is `SPAN_INCLUSIVE_INCLUSIVE`, which is Android's own spelling of typing attributes. |

### B3/B4 — `components.RichTextEditor`, lesson 4.14, docs

A wrapping toolbar whose buttons draw from the host's own selection report,
because Go owns the document and the host owns the caret. `UseRichToolbar` is
the hook and it is the caller's, for A3's reason.

---

## The four places the plan changed, and why

Written up at the end of `ai_docs/plans/components-editors.md` as well, so the
plan and the code do not disagree about what happened.

1. **`Toolbar` is a ref, not a bool** — see the summary above. Same move retired
   A3's `UseMemo`: the highlighter runs every pass, as the tutorial's has since
   it had snippets.
2. **Rich-text commands are pure Doc transformations** rather than Range
   surgery.
3. **Adopt-on-first-sight for the command epoch.**
4. **`nodeChildCount` replaced `children.length - chromeOffset(el)`.**

Smaller: `htmlout` emits a `CodeEditor` as a `<pre>` rather than wrapping a
`TextGrid`'s `<pre>`, so both DOM targets stay isomorphic — the property
`wasm/verify`'s replay exists to protect. `core.Focus`/`DismissKeyboard` reach
neither editor; stated on both node types as a known gap rather than left
silent.

---

## Two things the repo's own guards caught

Worth recording because both were the guard working exactly as written, on a
change it had never seen.

**`core.RoleToolbar` on the rich-text strip.**
`TestNoWidgetDeclaresACompositeContainerRole` refused it: a widget declaring a
keyboard composite's container role makes a *nested* composite reachable by
ordinary composition, which is a finding `core.AuditTree` exists to report and
no screen should produce by accident. The role came off, with the argument and
the caller's escape hatch written into the doc comment.

**`readNative` instead of `valuesIn`.**
`TestEveryNativeReadNamesItsQuestion` refused the new contract test's raw file
read: a check whose subject can be satisfied by a *comment* is the defect the
masking exists for. Switched to `valuesIn` (comments blanked, literals kept),
which is the right strength — most of what is pinned is a call with a prop name
in it.

Both were fixed by doing what the guard said rather than by widening the guard.

---

## What is checked, and what is not

- `go test ./...` — green, including `mobile/verify`'s node-type census, which
  is what forced both native arms to exist.
- `wasm/verify/run.sh` — 25 new `richtext_test.mjs` cases and 25 new
  `codeeditor_test.mjs` cases, plus the existing replay and the browser pass.
- `ios/verify/run.sh` — the macOS typecheck compiles the `#else` fallbacks; the
  **app-layer pass compiles the real UIKit halves against the iPhoneOS SDK**,
  which is stronger than the plan assumed was available.
- `android/verify/run.sh` — green, and does not compile Kotlin. Both Kotlin
  editors are held to `mobile/verify/{codeeditor,richtext}_test.go` textually.
  That gap is the plan's, carried forward and stated in both files.
- **Not checked anywhere:** anything needing a real caret. The web runtime's two
  selection functions return null where there is no Selection API rather than
  guessing, which is why the whole command vocabulary was made pure — a shim
  cannot answer a question about a caret, and a number invented there would make
  every command test pass for the wrong reason.

---

## Next

1. **(new · value medium) A device pass on all four editors.** The three things
   no harness in this repository reaches, named by the plan and still true: the
   IME composing region on Android (both editors guard on it and nothing has
   exercised the guard), hardware-keyboard Tab on iPad, and paste from another
   app into the rich-text editor. Everything else has a test; these have an
   argument.
2. **(new · value medium) Kotlin that imports Compose or Android is compiled by
   nothing here.** `GrMobCodeEditor.kt` and `GrMobRichText.kt` are the third and
   fourth files in this gap (`GrMobMapView.kt` was the first). The textual
   contract tests are real checks of the *rules* and no check at all of whether
   the file compiles. `android/verify` growing a classpath is the fix; this
   session did not take it on, as the plan said it would not.
3. **(new · value low) `core.Focus` and `core.DismissKeyboard` reach neither
   editor.** Both node types are deliberately absent from `focusableLeafTypes`,
   so a background tap will not put an editor's keyboard away. Four renderer
   changes and no core change. Documented on both node types rather than left
   silent; nothing drives it yet.
4. **(new · value low) A mark command with an empty selection is a full rebuild
   on the web.** The pending-mark path re-renders the document once per pending
   mark and restores the caret, which is invisible but is the one place the web
   editor does more work than the natives, where `typingAttributes` and a
   zero-length `INCLUSIVE_INCLUSIVE` span are free. Worth revisiting only if a
   profile ever says so.
5. **(carried · value medium) `launch.sh`'s numbers are one emulator's, and two
   things now rest on them.** Unchanged. One device, five cold launches,
   `--stages`. The emulator↔simulator gap is 200x on the same bytes, so this is
   not a formality.
6. **(carried · value low) `HeadingSensor.retryIfArmed` is unmeasured.**
   Unchanged. `headingAvailable()` is false on the simulator; needs a physical
   iPhone, one lesson, several taps in Settings.
7. **(carried · value medium) Windowing `core.List` over the bridge.** Unchanged
   and still gated on item 5. Worth 66–76% of the payload, needs no new bridge
   surface, costs a bootstrap guess plus placeholder children.
8. **(carried · value low) The contents screen's eight chapter cards are all
   expanded, and nothing asked for that.** Unchanged. 97.4% of the payload is
   inside them; collapsing by default would take more off the wire than
   windowing at no protocol cost. A UX decision about the tutorial, which is why
   it is written here rather than done.
9. **(closed) A short wire vocabulary.** Unchanged: sized at 17–28% and declined
   with the numbers in hand, dominated by item 7.
10. **(declined, non-goal)** Twenty-five entries. See
    `ai_docs/plans/non_goals.md`. This session adds nothing to it: the plan's own
    v1 non-goals (collaborative editing, autocomplete/LSP, code folding,
    find-and-replace, images or tables in rich text, full CommonMark, a
    host-side highlighter grammar, per-run font family) are recorded in
    `ai_docs/plans/components-editors.md` where they were decided, and none
    changes the shared design.
