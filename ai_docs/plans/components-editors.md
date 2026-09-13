# Next components: CodeEditor (programmer's editor) and RichTextEditor

**Status:** Landed, both tiers. See the "What shipped, and where it differs"
section at the end of this file for the four places the implementation departed
from the plan below and why.
**Date:** 2026-09-11
**Driver:** Two editing surfaces the widget library cannot express today. A notes /
CMS-style screen wants formatted text (bold, headings, lists, links); the tutorial,
a snippet runner, and any config-editing screen want a code buffer with syntax colour,
line numbers and a monospace grid. `core.TextArea` is the ceiling right now: one plain
string, wrapped, proportional font, no decoration.

---

## What exists today (constraints the plan works within)

- `comps` is pure Go over the public core API. Anything that can be built there
  costs no renderer work and runs on all four targets. Neither editor can be: both need
  an editable surface that carries *styled runs*, and no primitive has one.
- `core.TextArea` is the only multiline input. Its three live hosts share one contract
  worth reusing verbatim: **controlled value + echo guard** (`pendingEchoes` in
  `Renderer.kt`/`Renderer.swift`, the `value` arm in `grmob-runtime.js`) — the buffer is
  the host's while focused, Go's otherwise, and Go's echo of its own onChange never
  moves the caret.
- `core.TextGrid` is the read-only half of a code editor already: a monospace container
  with one `GridRow` child per line, each row's `GridRun`s (`t`/`fg`/`bg`/`a`) as one
  prop, so a changed line is one patch. Every renderer draws it (Compose
  `AnnotatedString`, SwiftUI `AttributedString`, `<pre>` of `<div>`/`<span>` on the two
  DOM targets).
- `examples/tutorial/highlight.go` is a working Go highlighter: `go/scanner` tokens →
  a class byte per source byte → `GridRun`s in the Darcula palette. It is private to
  the tutorial.
- `core.Style` has **no font family**. A pure-Go overlay (transparent `TextArea` in a
  `ZStack` over a `TextGrid`) cannot line its glyphs up with the grid, and SwiftUI's
  `TextEditor` / Compose's `BasicTextField` cannot be pitch-matched to a separate text
  view from outside. That is the argument that closes the "do it in comps" door
  for both editors.
- Imperative commands ride the tree, not a bridge call: `core/focus.go` stamps
  `focusEpoch` + `focusAction` on leaves and each renderer fires the action once when
  the epoch *changes*. A toolbar's "make this bold" is the same shape.
- The bridge has four callback channels: void, bool, int, text. A selection (two
  ints) or a structured document (JSON) can ride the text channel and be parsed in
  core, exactly as `NumericInput` parses its string — no bridge change.
- A new **node type** costs four renderers plus the harness: `htmlout/export.go` +
  `tag.go`, `wasm/grmob-runtime.js` (+ a `wasm/verify/*_test.mjs` against the fake
  DOM), `android/.../runtime/Renderer.kt` (+ a textual contract test in
  `mobile/verify`), `ios/GrMob/Runtime/Renderer.swift` (+ `ios/verify` type-check).
  Known gap carried from the MapView plan, and **closed 2026-09-11**: Kotlin that
  imports Compose is now compiled by `android/verify/sources.sh`, against the
  classpath `:app:printVerifyClasspath` resolves and with the Compose compiler
  plugin. The textual contract tests stay — they check the rules, which a compile
  cannot.

The two editors share one design and are ordered so the smaller one proves it.

---

## Shared design: the "decorated buffer" node

Both editors are a controlled text surface whose *decoration* is decided in Go and
whose *buffer* is the host's while focused. Three rules, common to both, written once
in core and checked once per host:

1. **Echo guard, unchanged from TextArea.** `value` (or `doc`) from Go is applied only
   when it is not an echo of the host's own last onChange.
2. **Decoration is advisory and per line.** Go sends styled rows; a host applies a row's
   styling only if the row's concatenated text equals the host's current line. A line
   that disagrees (Go is a keystroke behind) is drawn in plain ink until the next patch.
   Never the other way round: decoration never rewrites the buffer.
3. **Commands are epoch-stamped props.** `editorEpoch` (int, increments) +
   `editorCommand` (string). A host runs the command once when the epoch changes, on the
   current selection. Same mechanism as `focusEpoch`, so every renderer already has the
   idiom.

Selection is reported through the text channel as `"start:end"` (byte offsets into the
UTF-8 value on the wire; each host converts from its own unit — UTF-16 on Android and
the DOM, `String.Index` on iOS). Core parses it before calling the Go func, so app code
sees `func(start, end int)`.

---

## Tier A — CodeEditor

Ordered first because its wire is small (a string plus rows `TextGrid` already
carries) and it exercises all three shared rules before RichText adds a structured
document on top.

### A1. `highlight` package (pure Go, no renderer work)

Lift `examples/tutorial/highlight.go` into a public package and give it a seam:

```go
package highlight

// Highlighter turns source into one styled row per line. Rows must be
// line-for-line with the input (the stale-line rule depends on it).
type Highlighter interface {
    Rows(src string, scheme Scheme) []core.GridRow
}

type Scheme struct{ Keyword, String, Number, Comment, DocComment, Func, Ink, Bg string }

var Darcula, Light Scheme        // Light is derived from the theme's Ink/Surface roles
func Go() Highlighter            // go/scanner, moved from the tutorial
func JSON() Highlighter          // small hand lexer: keys, strings, numbers, literals
func Plain() Highlighter         // one run per line, no colour
func ForLanguage(name string) Highlighter   // "go" | "json" | "" → Plain
```

The tutorial's `highlight.go` becomes a caller; its existing tests pin that the
Darcula rows are byte-identical to before. Markdown and shell lexers are deliberately
not in v1 — each is a real lexer and none has a driver yet.

**Tests:** run-boundary tests per lexer; a property test that `len(Rows(src)) ==
strings.Count(src, "\n")+1` for every lexer (the line-for-line contract).

### A2. `core.CodeEditor` node (four renderers)

```go
core.CodeEditor(value, onChange, rows,
    core.LineNumbers(),               // gutter on
    core.ReadOnly(),                  // caret and selection, no edits
    core.TabSize(4),
    core.OnSelectionChange(func(start, end int) {...}),
    core.Height("240px"),
)
```

Wire: `Type: "CodeEditor"`, props `value`, `onChange`, `onSelectionChange`,
`lineNumbers`, `readOnly`, `tabSize`, `editorEpoch`, `editorCommand`; children are
`GridRow` nodes exactly as `TextGrid` builds them (reuse `gridRowNode`). Commands in
v1: `indent`, `outdent`, `commentLine` (needs a `commentPrefix` prop), `selectAll`.

Behaviour every host must share (the contract `mobile/verify` and `wasm/verify` pin):
monospace, no wrap, horizontal scroll, Tab inserts a tab rather than moving focus,
Enter copies the previous line's leading whitespace, autocorrect / smart quotes off,
a `readOnly` buffer still selectable.

| Target | Construction |
|---|---|
| htmlout | `<div class=code>` with a gutter `<div>` of line numbers and the `TextGrid` `<pre>` emission, read-only. A snapshot has no caret; nothing else to draw. |
| WASM | A `<div>` holding a gutter, a `<pre>` mirror built with `applyGridRuns` (the TextGrid code path) and a transparent `<textarea>` overlaid on the mirror. The runtime owns both elements, so font and pitch match — the thing a Go-side overlay cannot get. `keydown` handles Tab/Enter; `input` fires onChange and `selectionchange` fires the selection. No CodeMirror: no dependency, and it is testable against `wasm/verify/dom.mjs`. |
| iOS | `UIViewRepresentable` over `UITextView` (monospaced system font, `autocorrectionType = .no`, `smartQuotesType = .no`). Rows are applied as attributes on `textStorage` over the existing text — never by replacing `attributedText`, which resets the caret. Gutter is a sibling `UIView` driven by the layout manager's line fragments and the text view's `contentOffset`. |
| Android | `BasicTextField` with `FontFamily.Monospace` and a `VisualTransformation` that builds an `AnnotatedString` from the rows with an identity `OffsetMapping` — the sanctioned Compose way to colour an editable buffer without touching its state. Gutter is a `Column` sharing the field's vertical scroll state. The IME's composing region must be left alone: rows-only patches touch the transformation, never `TextFieldValue`. |

**Risks named up front.** (1) Newline insertion shifts every following row by index, so
one keystroke can patch N rows; measure on the tutorial's largest snippet before
deciding whether rows need keys. (2) The Compose half is not compiled here; the
textual contract test is the only check until `android/verify` grows a Compose
classpath, which this plan does not take on.

> **Risk (2) came true, and was closed afterwards.** `android/verify` grew that
> classpath on 2026-09-11 (`sources.sh`), and the first run of it found that
> `GrMobCodeEditor.kt` called `GrMobNode.isDisabled()` — which `Renderer.kt`
> declares `private`, and a `private` top-level declaration in Kotlin is
> file-private. The Android app had not compiled since this tier landed, and the
> textual contract test could not have said so: it checks that the right calls
> are made, not that they resolve.

### A3. `comps.CodeEditor` widget (pure Go)

```go
comps.CodeEditor{
    Value:       state.Src,
    OnChange:    func(s string) { state.Src = s },
    Language:    "go",                // or Highlighter: a highlight.Highlighter
    Scheme:      highlight.Darcula,   // zero value → derived from the theme
    LineNumbers: true,
    ReadOnly:    false,
    Height:      "240px",
    Toolbar:     true,                // indent / outdent / comment, as Buttons
}
```

Runs the highlighter each render under `UseMemo` keyed on `Value`; `go/scanner` on a
thousand lines is well under a millisecond, so no debounce. The toolbar is
`comps.Button`s driving A2's commands. A `ReadOnly` editor with `Toolbar: false`
is the tutorial's code block, which then drops its private `TextGrid` path.

### A4. Docs and lesson

Lesson 4.13 (edit a Go snippet live, watch the colour follow), a `## CodeEditor`
section in `docs/components.md`, roadmap lines.

---

## Tier B — RichTextEditor

### B1. `richtext` document model (pure Go, no renderer work)

The editor's value is a document, not a string. One model, owned by Go, so every host
edits the same thing and storage never sees platform markup:

```go
package richtext

type Doc struct{ Blocks []Block }
type Block struct {
    Kind BlockKind        // Paragraph | Heading1..3 | Bullet | Numbered | Quote | Code
    Runs []Run
}
type Run struct {
    Text                                   string
    Bold, Italic, Underline, Strike, Code  bool
    Link                                   string   // "" = not a link
}

func (d Doc) MarshalJSON / UnmarshalJSON   // short keys, GridRun style: {"b":[{"k":"p","r":[{"t":"..","B":1}]}]}
func (d Doc) Markdown() string             // CommonMark subset: the seven kinds and six marks above
func FromMarkdown(s string) (Doc, error)
func (d Doc) HTML() string                 // <h1>/<p>/<ul>/<ol>/<blockquote>/<pre>, <strong>/<em>/<u>/<s>/<code>/<a>
func (d Doc) PlainText() string
```

Persisting is the app's business (bytdb takes the JSON). The Markdown pair is for
import/export and for tests that read well; it is *not* the wire.

**Tests:** JSON round trip; Markdown round trip on the subset; HTML output pinned.

### B2. `core.RichTextEditor` node (four renderers)

```go
core.RichTextEditor(doc, onChange,
    core.Placeholder("Write something…"),
    core.ReadOnly(),
    core.OnSelectionChange(func(sel core.RichSelection) {...}),
)
```

Wire: `Type: "RichTextEditor"`, props `doc` (the JSON string), `onChange` (text channel
carrying the JSON; core unmarshals before calling `func(richtext.Doc)`),
`onSelectionChange` (text channel carrying
`{"s":12,"e":18,"marks":["bold"],"block":"h2"}` so a toolbar can show active state),
`placeholder`, `readOnly`, `editorEpoch`, `editorCommand`.

Commands: `bold` `italic` `underline` `strike` `code` `link:<url>` `unlink`
`block:p|h1|h2|h3|bullet|numbered|quote|code` `undo` `redo`. Each toggles on the
selection, or sets typing attributes when the selection is empty.

Echo guard compares the doc JSON string, same as `value`. The stale-line rule is not
needed: there is no separate decoration, the doc *is* the styled buffer.

| Target | Construction |
|---|---|
| htmlout | `richtext.HTML()` inside the node's box, read-only. |
| WASM | A `contenteditable` `<div>`. The runtime owns Doc→DOM and DOM→Doc serializers over the seven block kinds and six marks; `beforeinput` for typing, `Range`/`Selection` for commands (**not** `execCommand`, which is deprecated and differs per browser). Paste is serialized through DOM→Doc, so foreign markup is dropped at the edge. |
| iOS | `UIViewRepresentable` over `UITextView`. Doc↔`NSAttributedString` mapping: marks as font traits / underline / strikethrough / link attributes, block kinds as paragraph styles with list prefixes drawn as text. Commands edit `textStorage` over `selectedRange` or set `typingAttributes`. |
| Android | `EditText` with `Spannable` via `AndroidView` (the way osmdroid is hosted), not Compose `BasicTextField`. Spannable has had every mark and paragraph span this needs for a decade; building block structure into one `AnnotatedString` field is the riskiest line item in the plan and the classic-view route removes it. Same textual-contract caveat as A2. |

**Risks named up front.** DOM→Doc on the web is the piece most likely to grow cases
(browsers split and merge nodes freely under `contenteditable`); keep the serializer
normalizing rather than trusting structure, and pin it with `wasm/verify` fixtures.
Undo on the web is the runtime's own stack of Docs (native undo managers do not
survive programmatic attribute edits reliably); the natives use `UndoManager` /
`EditText`'s.

### B3. `comps.RichTextEditor` widget (pure Go)

```go
comps.RichTextEditor{
    Doc:         state.Doc,
    OnChange:    func(d richtext.Doc) { state.Doc = d },
    Placeholder: "Write something…",
    Toolbar:     comps.RichToolbarDefault,   // or a custom []RichToolItem
    MinHeight:   "160px",
    ReadOnly:    false,
}
```

The toolbar is a `comps.ChipStrip`-style row of toggle buttons whose selected
state comes from the last `OnSelectionChange` (widget-private state, the DatePicker
bar: presentation only). "Link" opens a `core` modal with an `Input` and dispatches
`link:<url>`. A `ReadOnly` editor with no toolbar is the display half: a comment, a
note, a description — the renderer's own text engine, no second "RichText view" node.

### B4. Docs and lesson

Lesson 4.14 (a note with a toolbar, a "Markdown" tab showing `Doc.Markdown()` live),
a `## RichTextEditor` section in `docs/components.md`, roadmap lines.

---

## Order and gates

1. **A1** — a day, no renderer, the tutorial adopts it immediately.
2. **A2 web + htmlout** — proves the overlay, the stale-line rule and the command epoch
   against the fake DOM before any native code exists.
3. **A2 natives**, iOS first (`ios/verify` type-checks it), Android second.
4. **A3, A4.** CodeEditor ships.
5. **B1** — pure Go, can start in parallel with step 3.
6. **B2 web + htmlout → iOS → Android**, same reasoning.
7. **B3, B4.** RichTextEditor ships.

Each renderer step lands with its contract test (`mobile/verify/codeeditor_test.go`,
`richtext_test.go`; `wasm/verify/codeeditor_test.mjs`, `richtext_test.mjs`) and a
manual device pass for the things no harness reaches: the IME composing region on
Android, hardware-keyboard Tab on iPad, paste from another app.

## Decisions to confirm before A2

- **Rich text wire = JSON `Doc`, with Markdown as import/export only.** Alternative: a
  Markdown string as the value. Declined here because Markdown cannot represent a
  selection-preserving edit and every host would need its own Markdown parser.
- **Web CodeEditor = own textarea-over-pre overlay, not CodeMirror.** Alternative:
  CodeMirror 6 loaded by the host page, Leaflet-style. Declined for v1: a large optional
  dependency for colour the Go side already computes, and untestable in `wasm/verify`.
  Reconsider if autocomplete or folding become drivers.
- **Android RichText = `EditText`/`Spannable` via `AndroidView`, not Compose.** See B2.
- **Package names:** `highlight` and `richtext` at the module root, beside `hooks` and
  `permission`, rather than under `comps`, because both are models with no view.

## Non-goals (v1)

Collaborative editing, autocomplete / LSP, code folding, find-and-replace, images or
tables in rich text, full CommonMark, a host-side highlighter grammar, per-run font
family. Each is a driver away, and none changes the shared design above.

---

## What shipped, and where it differs

Both tiers landed in one pass, in the order the gates above set. The plan held
almost everywhere; four decisions came out differently, each for a reason worth
recording.

**1. The toolbar names a ref instead of being a bool** (A3, B3). The plan wrote
`Toolbar: true` and had the widget make its own `core.EditorRef`. A ref must be
stable across passes, so making one is a hook — and a widget that calls a hook
must be rendered unconditionally on every pass. That is a fine obligation for an
editor with a toolbar and a wrong one for a *code block*, which is exactly the
thing rendered inside an `if`, inside a loop, inside a lesson body: the plan's
own closing line for A3 asks the tutorial's `codeBlock` to become one.

So the ref is the caller's and it *is* the Toolbar field:
`Toolbar: ref` for `CodeEditor`, `Toolbar: comps.UseRichToolbar(ctx)` for
`RichTextEditor`. An editor with no toolbar touches no hook. Same reasoning
retired A3's `UseMemo`: the highlighter runs every pass, as the tutorial's has
since it had snippets, and a buffer big enough to change that arithmetic wants a
caller-supplied `Highlighter` that caches rather than a widget that starts
consuming slots.

**2. Rich-text commands are pure document transformations on the web** (B2). The
plan said `Range`/`Selection` for commands, not `execCommand`. What shipped
reads the selection as two character offsets, transforms the *document*, rebuilds
and restores the caret — so the Range is only ever read, never operated on. That
is what makes the whole command vocabulary testable against `wasm/verify`'s DOM,
which has no Selection API at all, and it sidesteps the case the plan named as
the riskiest (browsers splitting and merging nodes under `contenteditable`)
rather than handling it.

The same pass added one thing the plan did not anticipate: the editor *remembers*
its selection, because clicking a toolbar button can take focus out of a
`contenteditable` and collapse the live one before the handler runs.

**3. An editor adopts a standing command epoch without running it** (A2, B2).
`core.Focus` deliberately re-fires on a field that mounts while it is the target.
An editor command names a moment and an edit, so an editor that was not on screen
missed it — otherwise returning to a screen re-indents its buffer. All four hosts
agree, and `mobile/verify` pins it on both natives.

**4. The add-child index is counted, not derived** (A2). A `CodeEditor`'s filler
rows — the stand-ins for buffer lines Go has not sent a row for yet — are the
runtime's first *trailing* chrome, so `children.length - chromeOffset(el)`
stopped being the node-child count. `nodeChildCount` counts the children carrying
a path instead, which answers the same question for a TabView and keeps
answering it for an editor.

Smaller notes. `htmlout` emits a `CodeEditor` as the `<pre>` a `TextGrid` is
rather than wrapping one, so both DOM targets stay structurally identical — the
property `wasm/verify`'s replay exists to protect. Markdown loses two things the
plan did not name (an empty paragraph, and the other marks on a code span), both
pinned by tests. `core.Focus`/`DismissKeyboard` reach neither editor; that is
stated on both node types as a known gap rather than left silent.
