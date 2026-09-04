# Session: htmlout's block flow, and Darcula in the tour

Session: https://claude.ai/code/session_01Mv1iFwxr7pCAdxCFMKT8Pk
Date: 2026-09-04 (follows "safearea-stacking" in this directory. That doc, and
the two before it, left the same item outstanding; this is it done — plus a
second, unrelated ask)

## Ask

"Now fix the htmlout block flow issue and In the interactive tour add syntax
highlighting to the code snippets - use a Darcula theme or similar."

---

# Part 1 — htmlout renders every stack container as block flow

## What it was

Carried across three session docs verbatim. The WASM runtime plants
`display:flex` on every `STACK_CONTAINERS` type whether or not the Style asks;
htmlout had no such default. A `<div>` is block flow, which runs inline
children together on one line (Text exports as a `<span>`) and ignores `gap`,
`justify-content` and `align-items` outright.

Both natives have no such mode — a Compose Row/Column and a SwiftUI
HStack/VStack are stacks by construction, and Box, Card, Scroll, SafeArea and
List all route through one of them. So htmlout was the outlier, one target of
four.

The named case, `examples/layout`'s `BodySection`, before and after:

```diff
-    <div>                                          ← the SafeArea root
+    <div style="display:flex; flex-direction:column">
-        <div style="background:#6200EE; padding:16px…">        ← Header's Row
+        <div style="background:#6200EE; display:flex; flex-direction:row; padding:16px…">
-        <div style="padding:8px 16px 8px 16px">                ← BodySection's Row
+        <div style="display:flex; flex-direction:row; padding:8px 16px 8px 16px">
-        <div style="background:#EEEEEE; text-align:center; padding:12px…">   ← Footer's Row
+        <div style="background:#EEEEEE; text-align:center; display:flex; flex-direction:row; padding:12px…">
```

Three Rows that laid out down the page in the exported HTML and across it on
every other target. Nothing errored, because block flow is a perfectly good
layout — just not the one the node asked for.

## The table

`htmlout/stack.go` is new, the sixth in the family. `stackAxisFor` answers
both halves of the question in one lookup — *whether* a type stacks and *along
which axis* — because membership and direction are one fact: a Row is a stack
*because* it stacks horizontally.

```go
"Row": "row",  "Column": "column", "Card": "column", "Box": "column",
"Scroll": "column", "SafeArea": "column", "List": "column",
```

That also removed a third copy of `nodeType == "Row" ? "row" : "column"`,
which htmlout and the runtime each had written out by hand.

The runtime's copy had been a bare `new Set([...])` beside that ternary. It is
now a `stackAxisFor` function in the same flat-literal shape the other five
tables use, which means **the existing `parseRuntimeTable` machinery pins it
for free** — no new parse, and the same failure shape as the tag and
cross-axis tables.

**Deliberately absent**, each with its reason in the table's doc:

- `Modal` — its fixed-overlay chassis sets display itself and toggles it
  through the `visible` prop, which a default here would fight.
- `Spacer` — a sized void with no children.
- `TabView` — a stack on the natives, but neither DOM target has ever
  defaulted it to flex. Left alone rather than changed as a rider: the two web
  targets agree about it today, which is the property this file protects.
- `Fragment`/`Theme` — the known divergence. The runtime boxes both in real
  divs to keep positional patch addressing valid; htmlout emits no element at
  all. The conformance test narrows the exemption to the exact axis rather
  than skipping the comparison, exactly as `TestRuntimeTagsMatchGo` does for
  the tag.

## Two reads, not one

The non-obvious half. `createElement` / `renderNode` plants the default on the
element as it is built — and that is not enough. `styleFromGrMob` and
`styleValue` are *total*: an update-style patch carries the whole new Style, so
a `display` they do not write is a `display` they erase. A runtime that read
the table only when building would stack a container until its first style
patch and then quietly drop it into block flow.

`TestRuntimeAppliesTheStackDefault` pins both reads with expression
substrings, so a comment naming `stackAxisFor` cannot satisfy it.

## The nil-Style path

`styleValue` returned `""` for a nil Style. A stack container has a
declaration list even with no Style at all, so nil is now normalized to an
empty Style and every other branch reads the zero value and emits nothing — a
non-container with no Style still returns `""`.

## Tests that had to move

Three existing tests used a container type to exercise a "not a flex node"
case, which is no longer a thing a container can be:

- `TestDisplayStaysVerbatimOnNonFlexNodes` (was a Column) → a Text.
- `TestFlexWrapAloneDoesNotCreateAFlexContainer` (was a Box) → a Text, which
  is what the runtime's twin of that test already used.
- `TestAlignFallbackDeclines`'s "Column with justify" case asserted *both* "no
  align-items" and "no flex container". The second half is dead for a Column
  now; the first was always the point (AlignJustify names no cross-axis
  placement), so the forbidden string narrowed to `align-items`.

Each edit carries the reason inline, so the next reader sees a decision rather
than a weakened assertion.

New in `htmlout/stack_test.go`: the default with a nil Style and with a
non-layout Style (the shape the bug was actually found in — a Row carrying
only padding), the axis per type, non-containers staying in block flow,
FlexDirection still overriding, DisplayNone still winning, the map copy, and
that every stack type is a known non-transparent tag.

---

# Part 2 — syntax highlighting in the interactive tour

## go/scanner, not regexps

`examples/tutorial/highlight.go`. The design decision is the lexer, and it
buys three things separately:

- The snippets are real Go, so the only tokenizer guaranteed to agree with the
  reader's own editor is the compiler's.
- String literals, rune literals and block comments stop being special cases.
  A `//` inside a string is a string, and the scanner knows that without being
  told. There is a test for exactly that case, because it is the one a
  regexp-based highlighter gets wrong.
- The token set cannot drift as the language grows.

A snippet is a *fragment* — several are a bare expression with no package
clause — which matters to a parser and not at all to a scanner. That is
precisely why the lexical layer is the right one to build on.

### Per source byte, not a list of spans

The output is rows, and a row is a line, so the classification has to be
sliced at every newline — and a block comment or a raw string crosses those
boundaries. One class byte per source byte makes that split a slice expression
with no case analysis: line 3's portion of a multi-line comment is just
`classes[start:end]`. Slicing at a newline can never split a rune, since a
newline is one ASCII byte and every byte of a multi-byte rune is inside one
token.

### The identity check

Converting scanner positions back to byte offsets is the one place this could
silently go wrong, so every token is checked against the source it claims to
cover (`src[off:off+len(text)] == text`) before it is trusted. One mismatch
abandons the whole snippet.

Auto-inserted semicolons are skipped: the scanner reports them with `"\n"` as
the literal because there is no semicolon in the source to point at, and
recording one would paint a class onto the newline or past the end of the
line.

### What is deliberately not coloured

A lexer knows what a token *is* and nothing about what it *refers to*, and
`classify`'s doc says so rather than guessing:

- Package qualifiers stay default ink. In `core.Text(...)` only `Text` is
  coloured — `core` could be a variable in the next snippet.
- Struct field names stay default ink, though Darcula purples them.
  `Scroll: true`, a `case x:` label and a map key are the same three tokens to
  a scanner.
- User-defined type names stay default ink. The predeclared ones do not,
  because the universe block is known without any scope analysis.

The one lookahead rule is "an identifier followed by `(`", which stands in for
"function name" and is right for both halves of what these snippets are made
of — `func Profile(` and `core.Text(` have the same shape at this level.
Predeclared identifiers are checked first, since `len`, `make` and `string`
are followed by a paren as often as not.

### The fallback, and the test that makes it acceptable

A snippet that does not lex cleanly is returned unhighlighted rather than
half-coloured: a run of code tinted as a string is actively misleading in a
document whose job is to teach the syntax. But a silent fallback is a perfect
hiding place, so `TestEveryTutorialSnippetHighlights` parses the package's own
sources with `go/parser`, pulls the argument out of every ``codeBlock(`...`)``
call, and holds all of them to the clean path. Reading the sources rather than
the rendered lessons is what makes it total — a rendered TextGrid has already
been through the highlighter, so a test driven off the tree would be checking
the output against itself.

`TestHighlightPreservesEverySourceByte` runs over the same corpus: highlighting
is a colouring, and the glyphs that come out must be the glyphs that went in.

## The code block is a TextGrid now

A code line is several colours, so the unit carrying a colour has to be
smaller than a line — which is exactly `core.TextGrid`'s shape. Three things
the old block did by hand came free:

- **Monospace.** Style still has no font-family prop; a TextGrid does not need
  one. The old block's own comment recorded this as a limitation it was living
  with.
- **No wrapping, and sideways scrolling.** A wrapped code line restarts at
  column zero, which reads as a new statement at the outermost indent — so
  wrapping destroys exactly the structure indentation exists to show. The old
  block spelled this out per-line with `WhiteSpace("nowrap")` plus an Overflow.
- **Indentation and blank lines.** The old block substituted non-breaking
  spaces and padded empty lines with one; a grid row preserves its spaces and
  an empty row still takes a line.

`codeBg`/`codeInk` moved from GitHub dark-dimmed to Darcula's own background
and foreground, so the surface and the token colours come from one scheme.

## A test helper that would have gone vacuous

`hasTextContaining` matched a `Text` node's content. Code blocks are GridRows
now, so several lessons that print a live literal into their block and assert
on it (chapter 4's button demo names the variant it just built) would have
stopped finding their substring — **a passing negative check and a failing
positive one**, which is how the failure actually surfaced. The helper reads
both node types now, reassembling a row from its runs, since where the run
boundaries fall depends on how Go tokenizes the line.

---

# Part 3 — the defect Part 2 uncovered

`element`'s pretty-printer **discards any text node that is entirely white
space**, and inserts its own newlines and indentation around non-inline
elements. Inside a `white-space: pre` element that indentation is *content*.

So htmlout's TextGrid export was already broken before this session, in two
ways nobody had looked at: every row gained a trailing line break, the grid
gained a blank line between each pair of rows, and any run made only of spaces
exported as an empty `<span>`. The existing test never exercised a
whitespace-only run, so it went unnoticed.

Highlighted code is *nothing but* whitespace-only runs — every indent, every
gap between two coloured tokens — so this had to be fixed rather than noted.

## Three levels, each saying something different

```
grid  white-space:normal   overrides the <pre> default, so the newlines and
                           indentation *between* rows are formatting
row   white-space:nowrap   a code line or a terminal row is one line and must
                           not break between two runs; nowrap still collapses,
                           so the break before each </div> disappears
run   white-space:pre      a run's own spaces are the only white space in a
                           grid that means anything
```

Pushing the significance down to the run makes a grid indifferent to how the
markup around it is laid out — worth having whatever the formatter does.

The WASM runtime states the same three rules. It has no formatter that could
disturb a grid; it carries them so that it does not *differ* from the exporter
that does.

## The character references

The row/run split does not save a run that is *only* spaces — the formatter
still deletes its text node. Those are written as `&#32;` instead: not white
space to the formatter, exactly a space to the browser.

Written unescaped, and only there. The branch is entered only when every rune
is white space, and `spaceRefs` emits nothing but digits inside `&#...;`, so
no character that could open a tag or an entity can reach the output through
it. Every run with a glyph still goes through `TE`, and
`TestGridRunWithGlyphsIsStillEscaped` pins that the escaping guarantee has no
hole in it.

## Verified in a browser

Not just in tests: the exported block was served over localhost and opened in
Chrome. Indentation preserved and aligned on both indented lines, the blank
line one line tall, no double-height rows, comments italic, tokens in their
Darcula colours.

---

## Changes

**Part 1**
- `htmlout/stack.go` (new) — `stackAxes`, `StackAxisFor`, `StackAxes`,
  `StackTypes`.
- `htmlout/export.go` — `styleValue` reads the table for the axis and the
  promotion; nil Style normalized to empty.
- `wasm/grmob-runtime.js` — `STACK_CONTAINERS` → `stackAxisFor` in the pinned
  flat-literal shape; `createElement` and `styleFromGrMob` read it.
- `wasm/verify/stack_test.go` (new) — table conformance plus the two-read pin.
- `htmlout/stack_test.go` (new), and three existing tests moved to a
  non-container type with the reason inline.
- `docs/platforms/wasm.md` (a "Stack containers" section; the cross-axis
  section's stale "the three containers" corrected to five),
  `docs/platforms/exporters.md`.

**Part 2**
- `examples/tutorial/highlight.go` (new) — the lexer, the Darcula palette, the
  universe-block table.
- `examples/tutorial/highlight_test.go` (new) — the corpus sweep, byte
  preservation, per-class colours, the `//`-in-a-string case, multi-line block
  comments, the total fallback, maximal runs.
- `examples/tutorial/widgets.go` — `codeBlock` is a TextGrid; the palette is
  Darcula's.
- `examples/tutorial/app_test.go` — `hasTextContaining` reads GridRows;
  `gridRowText`.
- `examples/tutorial/app.go` — structure comment names highlight.go.
- `docs/tutorial-interactive.md` — a "Code blocks" section.

**Part 3**
- `htmlout/export.go` — the three-level chassis, `spaceRefs`, `gridRunStyle`
  always carrying `white-space:pre`.
- `wasm/grmob-runtime.js` — the same three rules.
- `htmlout/textgrid_test.go`, `wasm/verify/textgrid_test.mjs` — expectations
  updated, plus new tests for whitespace survival and for the escaping
  guarantee.
- `docs/platforms/wasm.md` — the three-level table under "Text grids".

## Still outstanding

- **`TabView` is a stack on the natives and block flow on both DOM targets.**
  Newly named rather than newly true: it was outside the runtime's set before
  this and is outside the shared table now, so the two web targets still agree
  with each other and still disagree with the natives. Fixing it means
  deciding what a TabView's tab bar and its one visible child should do under
  flex, which is its own pass.
- **`element.PrettyHTML` mangles whitespace-significant content.** Part 3 works
  around it in htmlout rather than fixing it upstream — the dependency is
  pinned at v0.7.0 and changing that is not this session's call. The workaround
  is self-contained and documented at both ends.
- **CameraView stays an overlay** on both natives, which is what it is for.
  Modal is one on Android for the same reason.

## Verification

`go test ./...`, `gofmt`, `go vet`, `sh wasm/verify/run.sh`,
`sh ios/verify/run.sh`, and `:app:compileDebugKotlin` (`--rerun-tasks`) — all
clean. Plus the exported code block rendered and inspected in Chrome.
