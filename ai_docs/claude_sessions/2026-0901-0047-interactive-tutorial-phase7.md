# Interactive Tutorial — Phase 7: Chapter 7 (Theming & Styling)

Session: https://claude.ai/code/session_014sBCzrGB5gk88FhYPkSszw

## Goal

Phase 7 of the eight-phase interactive tutorial: add Chapter 7 "Theming &
Styling" to `examples/tutorial` (now 35 lessons). Committed as 63d5baf.

## The caveat, resolved

Phase 6's doc flagged: "theme is carried on the Context (`WithTheme` copies
share scopes), so a live switcher likely needs the theme as state feeding a
re-wrapped context — verify before assuming the framework change is zero."
Verified: **the framework change IS zero**. `core.WithTheme(theme,
children...)` is already a View wrapper (core/theme.go) whose ComponentFunc
derives a themed context per pass and renders children with it — so a live
switcher is nothing but `core.NewState(ctx, 0)` indexing
`[]*core.Theme{DefaultTheme, MaterialTheme}` and feeding the wrapper.

The one REAL constraint found while verifying: `ctx.WithTheme()` copies the
Context by struct literal — it **shares the slots slice but forks the Cursor
int** (and is a fresh ephemeral struct each pass, outside the children/audit
tree). A hook claimed inside a WithTheme subtree would take a slot index that
collides with whatever the lesson claims after the wrapper. Rule: keep the
themed subtree hook-free; own its state in the frame and pass it down by
closure. Documented on `themePreview` and taught as a 7.4 key point.

## What was built (all in examples/tutorial/, no framework changes)

- **chapter7.go** — five lessons. Through-line: styling is plain data — one
  Style struct per node, assembled theme-base-then-props-in-order; every
  restyling gesture (append a prop, layer a role, swap the theme) is a data
  change shipped as update-style patches, and Transition declares how those
  patches move.
  - **7.1 The style pipeline** — later-props-win. Demo: props list built by
    appends; two toggleable BackgroundColor "coats" (teal, then plum) over a
    blue base; a caption narrates which coat won and why. Also teaches the
    asymmetry: containers/buttons/inputs resolve a `ComponentDefaults` base,
    `core.Text` starts bare (verified in core/text.go — Text does NOT read
    Components.Text; only leafNode/containerNode callers get bases).
  - **7.2 UseStyle: layers that merge** — package vars `calloutStyle`
    (Display: DisplayBlock so a bare Text carries padding/corners in HTML)
    and `bigType` (type fields only). Demo walks the docs' exact edge:
    layering bigType leaves fill/ink/corners alone (merge rule), then
    `UseStyle(Style{BorderRadius: 0})` visibly does nothing vs
    `core.BorderRadius(0)` which squares the corners.
  - **7.3 Inside a Theme** — live inspector reading `core.DefaultTheme` /
    `core.MaterialTheme` as plain data (nothing installed — that boundary is
    the segue to 7.4). `swatchRow` helper: color chip (hairline in the APP
    theme's Border role — otherwise the near-white Surface swatch vanishes,
    which is itself the Border-vs-Surface lesson), role name, hex caption.
    Plus type-scale and Button-base caption lines. Late roles read through
    resolvers even though bundled themes set them — the habit is the lesson.
  - **7.4 Two themes, one tree** — THE switcher. SegmentedControl feeds
    `core.WithTheme(installed[pick.Get()], themePreview(...))`. Preview: a
    profile Card written 100% in roles (Typography, palette, widget bases);
    hook-free; Follow button flips lesson-owned state through the themed
    boundary; Message fires a 6.5 toast. Caption points out the chips/lesson
    outside the wrapper stay DefaultTheme — scoping is the point.
  - **7.5 Transitions: declare the motion** — Snap(0)/250ms/800ms pace picker
    on a box flipping color+padding+radius (padding so the SIZE glides, not
    just a fade). Teaches declare-in-Go/drive-natively (platform animates,
    never patches over the bridge), `Transition(0)` clears, serialized
    "\<ms\>ms \<easing\>", no completion callback by design.
- **chapter7_test.go** — five tests. This chapter's claims land in
  `Node.Style`, not text/structure, so tests read decoded styles: helpers
  `styledText` (Text by exact content, must carry style) and `demoBoxNode`
  (innermost Column with a background + marker text; ancestors have no bg).
  7.4 asserts the scoped swap precisely: preview Follow button turns
  `#6200EE`/r4 under Material while the lesson's own `Next ›` (outside the
  wrapper) stays `#007AFF`; state tap doesn't disturb the theme choice.
  7.5 asserts `"250ms ease-in-out"`, clearing at Snap, re-declare at 800ms.
- **app_test.go** — wire `node` struct gained `Style *nodeStyle`
  (Background/TextColor/FontSize/FontWeight/BorderRadius/Transition).
  Additive: the wire tree is a straight `json.Marshal(core.Node)`, so keys
  are Go field names; existing tests untouched.
- **lesson.go** — `chapter7(),` appended to `Chapters`.

## Facts learned/confirmed this phase

- `RenderInitial` output is a plain `json.Marshal` of `core.Node` — `Style`
  serializes under Go field names, so tests can decode any style field.
- `core.Text` takes `...StyleProp` (NOT `...PropsAndChildren` like Button/
  containers) — a conditionally-built prop slice for Text must be
  `[]core.StyleProp`.
- `MaterialTheme.Components` sets no `Text` entry and DefaultTheme's is
  irrelevant to core.Text anyway (Text reads no component base at all).
- `core.Transition(0, …)` sets `Style.Transition = ""` — clearing, not
  "0ms"; canonical form is `"<ms>ms <easing>"`; wasm runtime maps it to
  `transition: all <that>` (grmob-runtime.js line ~241).
- Chip segments render as core.Buttons labeled by caption, so the plain
  `tap` helper drives SegmentedControls ("Material", "Snap", …).
- Two SegmentedControls in different lessons may reuse caption sets;
  KeyPrefix ("inspect-", "installed-", "pace-") is per-control hygiene only.

## Verification

- `go test ./...` fully green (chapter tests green on first run); gofmt and
  vet clean; `go test -race -count=2 ./examples/tutorial/` clean.
- `GOOS=js GOARCH=wasm go build ./wasm` compiles (note: needs `-o` somewhere
  since bare build tries to write a binary named `wasm` over the directory).
- Browser eyeball still pending (same claude-in-chrome localhost blocker as
  Phases 1/6). 7.5's glide especially is only asserted as a declaration:
  `GOOS=js GOARCH=wasm go build -o wasm/main.wasm ./wasm && (cd wasm && python3 -m http.server 8080)`.

## Next session: Phase 8 (finale)

Per the phase list (Phase 1 doc, 2026-0831-2047): Ch.8 "Robustness" +
polish — error boundaries, debug mode, caching (core/error_boundary.go,
core/debug.go, core/cached.go; sessions 2026-0831-1733-error-boundaries,
2026-0830-1919-cached-diff-fast-path, 2026-0830-1937-debug-mode). Plus the
docs page for the tutorial, mkdocs nav entry, and README mention. Watch for:
the tutorial's own TestMain already runs SetDebugMode(true) — a debug-mode
lesson demo that deliberately provokes a concern would trip every test's
assertNoConcerns; the demo likely needs to read/clear concerns carefully or
demonstrate on a sandboxed manager, not the app's own tree.
