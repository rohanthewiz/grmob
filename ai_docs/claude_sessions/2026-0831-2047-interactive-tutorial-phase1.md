# Interactive Tutorial — Phase 1: Shell + Chapter 1 (Views & Layout)

Session: https://claude.ai/code/session_01Nm3azgkKS6SWsFoqJZEdUh

## Goal

Build an extensive interactive tutorial for GrMob, phased. This session
delivered Phase 1 of eight: the tutorial's framework plus Chapter 1.

## The design decision

The tutorial is itself a GrMob app — `examples/tutorial` — rather than more
prose in `docs/`. Every lesson is a live screen: explanation, the code under
discussion, and a bordered "TRY IT" panel wired to real state and callbacks.
Being an ordinary example package means the one artifact:

- runs in the browser via the wasm host,
- ships natively through the mobile bridge (`init` + `AppName`, the
  todoapp contract),
- and is driven headless by its tests through `render.Manager`, so a lesson
  whose demo breaks fails CI.

## The phase plan (this session started Phase 1)

1. **Shell + Ch.1 "Views & Layout"** ← done here
2. Ch.2 "State, Events & Lists" — counter, controlled inputs, conditionals, keyed lists
3. Ch.3 "Hooks & Effects" — UseInterval clock, UseTimeout, UseEffect, UseMemo, UseReducer
4. Ch.4 "The Widget Library" — Button variants×emphasis, Badge/Chip/SegmentedControl, ListRow, Accordion, Tabs
5. Ch.5 "Forms & Validation" — rules, reveal policies, focus/blur, FormField
6. Ch.6 "Navigation & Overlays" — Push/Pop/Replace/Reset, Modal, Toast
7. Ch.7 "Theming & Styling" — live Default↔Material switcher, style merging, transitions
8. Ch.8 "Robustness" + polish — error boundaries, debug mode, caching; docs page, mkdocs nav, README

## What was built (all new files in `examples/tutorial/`)

- **lesson.go** — `Lesson`/`Chapter` model; `Chapters` curriculum var
  (chapters append per phase); `flatLessons`, the flattened ordered index.
  Lesson IDs ("1.2") are derived from position, never stored, so they can't
  drift. Prev/next is flat-index arithmetic; chapter grouping is re-derived
  by the contents screen.
- **app.go** — `App`: Navigator whose initial route is Home. Progress
  (`visited` map, lesson ID → opened) lives in `ctx.Scope("tutorial-session")`
  *above* the Navigator — session state, not frame state, so a future Reset
  can't orphan it. `markVisited` copies the map (immutability rule);
  only ever called from handlers, never during render.
- **home.go** — table of contents: title, progress card ("n of m lessons
  opened" + ProgressBar), one Card per chapter, `ListRow` per lesson (keyed
  by ID) with a green "opened" Badge once visited. Row tap = markVisited +
  Push.
- **lesson_screen.go** — the scaffold every lesson renders in: ghost
  "‹ Contents" back button + chapter Badge, header, `e.Body(ctx)`,
  Separator, then Prev (outlined) / Next (filled) / Finish (success) footer.
  Prev/Next use `core.Replace` so the stack stays [contents, lesson] and
  each lesson's demo state resets with its frame. The absent Prev on lesson
  one is a plain Go nil (containers skip nil), NOT `core.If` (empty Fragment
  child) and NOT `MaybeProp` (eager eval would index flatLessons[-1]).
- **widgets.go** — tutorial building blocks, all hook-free:
  `prose`/`caption` (theme typography via ComponentFunc deferral),
  `codeBlock` (line-per-Text with leading spaces swapped to NBSP — HTML
  collapses newlines and indent runs; fixed dark palette #22272E/#ADBAC7
  since there's no font-family prop yet), `demoPanel` (Border-role hairline
  + "TRY IT" badge), `keyPoints`, `demoBox(label, color, extraPad, extras...)`,
  `stepper` (−/+ control, controlled: caller owns clamping).
- **chapter1.go** — five lessons, each with an interactive demo:
  1.1 Hello, GrMob (checkboxes toggle which sub-views compose a profile card);
  1.2 Text & typography (size stepper, bold toggle, palette-role ink
  SegmentedControl); 1.3 Rows/Columns/spacing (axis switch + gap stepper);
  1.4 Alignment & flex (Justify/AlignItems segmented controls over
  mixed-height boxes, FlexGrow toggle); 1.5 Surfaces (Box vs Card side by
  side, radius/shadow knobs). Demos borrow NewState a chapter early — noted
  in captions; snippets stay on-topic.
- **app_test.go** — 6 tests, signup-example discipline: debug mode on in
  TestMain, every pass audited (`assertNoConcerns`), callback IDs always
  read from a freshly rendered tree. Covers: home lists everything; open →
  progress + badge + back; Next walks all lessons and Finish pops home with
  full progress; Prev steps back; two demo-liveness tests (1.1 recompose,
  1.3 axis switch).
- **wasm/main.go** — dot-import switched `examples/social` →
  `examples/tutorial` (the documented one-line app switch; swap back to
  restore the social demo).

## Verification

- `go test ./...` fully green; `gofmt` clean.
- `GOOS=js GOARCH=wasm go build ./wasm` compiles.
- Browser check attempted (served on localhost:8477 from scratchpad) but
  Chrome's extension showed an error page for localhost while curl got 200 —
  likely missing localhost site permission in the claude-in-chrome
  extension. Needs a human eyeball in the browser:
  `GOOS=js GOARCH=wasm go build -o wasm/main.wasm ./wasm && (cd wasm && python3 -m http.server 8080)`.

## API facts worth re-knowing next phase

- `components.Chip` renders as a `core.Button` (label prop) — the test
  `tap` helper reaches segments directly.
- `Text` content lives in `Props["content"]`; node JSON is
  Type/Props/Children (Style is separate; tests assert content/type, not
  style).
- Container builders take `...PropsAndChildren`; can't mix a fixed arg with
  a spread — build the slice (`asAny` helper in home.go widens []View).
- `ListRow` registers `onClick` only when OnTap set; find rows by
  clickable-with-descendant-text.
- Theme roles: `Colors.BorderColor()/SuccessColor()/WarningColor()` are
  resolver methods (fallbacks for pre-role themes); Typography has
  Title/Subtitle/Body/Caption.

## Next session: Phase 2

Add `chapter2()` (State, Events & Lists) to `Chapters` in lesson.go:
counter + NewState mechanics, controlled Input echo, Checkbox/conditionals
(If/IfElse/Match), For + Keyed list with add/remove showing why keys matter.
Framework needs no changes — append the chapter file, lessons pick up IDs
2.x automatically, tests extend the same helpers.
