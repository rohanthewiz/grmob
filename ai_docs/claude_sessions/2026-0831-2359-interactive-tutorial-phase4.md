# Interactive Tutorial — Phase 4: Chapter 4 (The Widget Library)

Session: https://claude.ai/code/session_01Nm3azgkKS6SWsFoqJZEdUh

## Goal

Phase 4 of the eight-phase interactive tutorial: add Chapter 4 "The Widget
Library" to `examples/tutorial`. As in Phases 2–3, the framework needed
exactly one change — appending `chapter4()` to `Chapters` in lesson.go —
and IDs 4.x, home rows, progress, and the Next/Prev walk (now 20 lessons)
picked the chapter up automatically.

## What was built

- **chapter4.go** — five lessons, each with a live demo. Through-line: the
  components package's controlled-widget contract — named-field structs
  implementing `core.View`, looks from `ctx.Theme()` with Style overrides,
  state with the caller (Accordion the documented exception).
  - **4.1 Buttons: two axes** — Variant × Emphasis segmented pickers,
    Disabled/FullWidth checkboxes, a live preview button with a taps
    counter, and the demo's twist: a dynamic `codeBlock` that re-prints
    the exact `components.Button{...}` literal the knobs would build,
    omitting zero-value fields (`buttonSnippet`) — the omission is the
    "zero values contribute nothing" lesson. `variantNames`/`variantValues`
    and `emphasisNames`/`emphasisValues` are package vars: captions double
    as constant-name suffixes so captions and printed code can't disagree.
  - **4.2 Badges, chips & segments** — one pill family: Badge (states,
    never tapped) fed by a SegmentedControl through the shared index into
    `variantValues`; a multi-select chip group over one `map[string]bool`
    updated copy-flip-Set (`maps.Copy` — the mapsloop linter suggested it
    over a manual copy loop); a derived "picked: a · b" caption iterating
    `pillTopics` for stable order.
  - **4.3 ListRow & Avatar** — three-gopher roster (June Gopher / Rex
    Burrows / Sal Tunnels), Avatar leading slots (derived "JG" initials),
    badge-or-chevron trailing from one loop, toggling single selection
    keyed by member name. Teaches the FlexGrow-spine story (why not
    JustifyBetween) and that ListRow synthesizes no a11y label while
    Avatar does.
  - **4.4 Accordion: the stateful widget** — a FAQ *about Accordion
    itself* (three entries in `accordionFAQ`, first `InitiallyExpanded`),
    whose answers double as test sentinels visible only while expanded.
    Teaches: the widget's `NewState` claims a slot on the caller's
    context → render unconditionally in stable order; Content renders
    only while expanded → must stay hook-free.
  - **4.5 Tabs & the wire contract** — `components.Tabs` as a facade over
    `core.TabView` (the node-type wire contract lives in core, next to
    the renderers' registry). The live demo hand-composes the same
    controlled contract — SegmentedControl strip + `core.Match` pages —
    because the web host doesn't draw TabView (see below). A `notify`
    checkbox slot declared above the page switch demonstrates page state
    surviving switches.
- **chapter4_test.go** — five liveness tests plus one new primitive:
  `tapRow(t, mgr, sub)` clicks the outermost clickable non-Button node
  whose subtree contains a substring — ListRow and Accordion headers
  register OnClick on containers, out of reach of the label-based `tap`.
  Sentinel discipline: dynamic-code assertions use names absent from the
  static intro blocks (VariantWarning/EmphasisGhost, not Error/Outlined);
  badge text asserted with exact `hasText` so code-block lines can't
  false-match; FAQ answers carry unique phrases.
- **lesson.go** — `chapter4(),` appended to `Chapters`.

## Renderer facts learned/confirmed this phase

- **TabView is native-only.** Android's Renderer.kt draws a Material tab
  row (`GrMobTabView`); iOS's Renderer.swift hand-rolls a matching *top*
  bar because SwiftUI's own TabView is a bottom bar with locally-owned
  selection — the wrong shape for a controlled int prop from Go. Neither
  htmlout (falls to the default `renderContainer` div: all pages stacked,
  no strip — labels live in props, not children) nor wasm/grmob-runtime.js
  handles it. This is why examples/social hand-rolls TabButton, and why
  lesson 4.5's demo composes the contract instead of rendering a TabView
  that would look broken in the browser.
- `components.Chip` selection changes style only (Surface bg + Primary
  ink), no text change — tests must instrument selection through derived
  captions, not chip appearance.
- Dispatching a disabled `components.Button`'s callback is a registered
  no-op (never nil): testable as "taps stop landing" with no crash.

## Verification

- `go test ./...` fully green; gofmt and vet clean; TestMain debug mode
  audits every pass (all 20 lessons walked by the Next test).
- `go test -race -count=2 ./examples/tutorial/` clean.
- `GOOS=js GOARCH=wasm go build -o <scratch>/main.wasm ./wasm` compiles
  (bare `go build ./wasm` still fails on the wasm/ directory name).

## Next session: Phase 5

Per the phase list (Phase 1 doc, 2026-0831-2047): Ch.5 "Forms &
Validation" — rules, reveal policies, focus/blur, FormField. Source of
truth: docs/concepts/forms.md, the forms/ package, and the focus/blur
work from the two pre-tutorial sessions (2026-0831-1955 and -2022).
Framework change should again be exactly one line: `chapter5(),` in
lesson.go.
