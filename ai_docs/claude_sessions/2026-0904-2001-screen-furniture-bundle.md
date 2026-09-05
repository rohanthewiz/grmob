# Session: A2, the lifted widget bundle

Session: https://claude.ai/code/session_01J7qf4UetyH8cQ3D7Egn9dC
Date: 2026-09-04 (follows "collection-widget-followups")

## Ask

"Let's do A2: the lifted widget bundle" — item 1 of the previous session's
Next list, and the A2 row of
`ai_docs/plans/components-datatable-compass-map.md`: seven widgets each
hand-rolled in church_mobile or in the examples (AppBar, Banner, EmptyState,
SearchField, ChipStrip, Skeleton, StatTile).

All seven landed, plus one hook the plan turned out to need. Tutorial lesson
4.7 demonstrates them as one screen. Nothing in church_mobile was changed —
adopting them downstream is the natural follow-up, as it was for A1.

## The scope call: `hooks.UseDebounce`

The plan said SearchField would debounce "via `UseTimeout`". It cannot.
`UseTimeout` arms once per mount and *stays fired* — deliberately, because the
store-keyed version it replaced used to re-arm on whichever unrelated render
came next. A debounce is defined by re-arming, so the two differ in exactly
the thing the type is about.

So `hooks/debounce.go` is new: a `*Debouncer` in a hook slot with
`Call`/`Cancel`/`Pending`. It is the first addition to `hooks` from a
components-driven need, and it was flagged as a scope stretch before writing.

Design decisions worth keeping:

- **A separate hook, not a flag on UseTimeout.** See above.
- **The delay is re-read every render**, where `UseInterval`'s period is fixed
  by the first. It lives on the record rather than inside a running timer, so
  a duration driven by state (a "search as I type" preference) takes effect on
  the next `Call`. Pinned by a test that mutation-checks the alternative.
- **A zero delay runs `fn` inline**, and requests no render — the call is then
  indistinguishable from the caller having invoked `fn` itself, inside event
  handling that will render anyway. The delayed path does request one, for the
  same reason `UseTimeout` does.
- **A nil `fn` is inert and does not cancel.** Reading "schedule nothing" as
  "unschedule everything" would make `d.Call(handlerFor(mode))` silently
  destructive when the handler is nil.
- **`Cancel` promises no *further* call, not an undo** — `Stop` returns false
  once the timer has fired. Documented rather than papered over.

## The seven widgets

Each is stateless and controlled; none calls a hook, so `components/doc.go`'s
"only Accordion" line still holds. That mattered most for SearchField: a
search box is a thing screens render conditionally (a header that appears when
"Search" is tapped), which is precisely what a hook-slot consumer must not be.

- **AppBar** — the back control is drawn exactly when `core.CanPop` is true,
  so a tab root gets none and a pushed screen gets one, both without wiring.
  `OnBack` *replaces* `core.Pop` rather than running before it, which is what
  makes confirm-before-leaving possible (mutation-checked). Title takes the
  theme's **Subtitle** size with primary ink and Bold: `Typography.Title` is
  28pt under DefaultTheme and does not fit in a bar. `HideSeparator` is a
  negative bool because the zero value has to work on the zero-value screen —
  an unstyled bar shares the content's Background and needs the rule.
- **Banner** — the variant is spent on the **border and glyph, never the
  fill**. A saturated Error red across a screen reads as a failure of the app,
  and the palette has no muted container tone (the same gap Button's doc names
  as wanting a second "on-light" tone per role). The side effect is that a
  banner's contrast is identical whatever it is saying. The action button is a
  *default* ghost, not variant-tinted: three tellings would put the least
  legible combination in the package on the one control meant to be hit.
- **EmptyState** — absorbed `emptyNote`, `busyNote` *and* `errorRetry`: three
  states, one shape, so their wording and spacing cannot drift. `Width: 100%`
  is load-bearing and looks redundant — a column hugs its widest child on both
  natives, and the DOM targets hide the bug. Its action is outlined, because
  an empty state is a dead end.
- **SearchField** — controlled, hook-free, debounce delegated (above). The row
  is the frame and the input inside it is flattened (transparent, no radius,
  `Padding(0)`), or the theme's Input base draws a second box inside the
  first. `AccessibilityLabel` falls back to the resolved placeholder, because
  a placeholder is not a label on any platform.
- **ChipStrip** — takes `[]Chip`, not a parallel labels/selection vocabulary:
  a chip in a strip is then configured exactly like a chip anywhere. Assigns
  `Padding(0)` over the theme's Row inset. No `Scrollable` field waiting on
  B1 — a prop that does nothing is worse than one that does not exist.
- **Skeleton** — bars take the **Border** role, not Surface, for the reason
  Separator already documents (a Surface bar inside a card disappears). Last
  bar short only when `Lines > 1`. No shimmer, and that is not a shortcut: a
  moving highlight is a repeating keyframe animation and `core.Transition`
  animates between two declared values; looping it from Go would push a render
  pass and a bridge patch per frame of a decoration.
- **StatTile** — no frame. "Tile" names the content; the card is the caller's,
  which is what lets three tiles share one card or take one each. Its delta's
  zero variant is **neutral, not Primary** — the one place in the package
  where `VariantDefault` departs — because whether a number going up is good
  is the caller's domain (attendance up is success, spend up is not, latency
  up is an incident). `Fill` sets `FlexGrow` *and* `FlexBasis("0")`: Compose
  and SwiftUI divide the whole axis by weight, CSS divides only the leftover,
  and the prop that is inert on two targets is exactly the one that converges
  the other two.

## Method note

Every behavioral claim was mutation-checked, and two of the checks paid:

1. **Three debounce tests were passing for the wrong reason.** They used
   `ctx.BeginRenderPass()` for a "second render", which only cycles the
   *callback* registry — the hook cursor is rewound by `ctx.Reset()`. So the
   second `UseDebounce` was allocating a fresh slot and the tests never
   exercised re-binding at all. Caught because removing `bound = false` from
   the `OnClose` did not fail the remount test. Fixed to `Reset`, plus an
   identity assertion (`d != first` → fatal) so the same mistake cannot recur
   silently.
2. Ten mutations across the widgets — back ignoring `CanPop`, `OnBack` also
   popping, `EmptyState`'s width dropped, the input's flattening removed, the
   clear button moved off the row's tail, `FlexBasis` dropped, a single
   Skeleton bar shortened, Banner filling with its variant, ChipStrip keeping
   the Row inset, Banner's action taking the variant — each makes exactly the
   test that should fail, fail.

Two accidental collisions with the tutorial's `hasTextContaining`, which also
reads code-block grid rows: the sample Banner text in the lesson's snippet
matched the live banner's assertion. Changed the snippet's wording rather than
weakening the assertion.

## Tutorial 4.7

`Screen furniture: bars, banners & placeholders` — the seven assembled as one
miniature archive screen over chapter 4.6's existing fixture, with a real
`hooks.UseDebounce` behind the search. The demo's AppBar draws its *real*
automatic back arrow, because a lesson is a pushed screen; it is harmless only
because the demo sets `OnBack`, and the lesson says so. Two tests: the
search/filter/empty-state path and the banner/skeleton path.

The debounce assertions deliberately go through the race-free paths —
`OnSubmit` (Cancel then act) and `awaitText` — because "the list has not moved
yet" is a race by construction.

## Also touched

- `docs/components.md` gained the seven sections **and** a Collections section
  for GroupedList/DataTable, which the previous session never added: a widget
  doc omitting two shipped widgets while gaining seven more is worse than one
  that lags.
- `docs/concepts/state-and-hooks.md` gained `UseDebounce`, next to
  UseInterval/UseTimeout, with the "why not a flag on UseTimeout" line.
- ROADMAP, the A2 plan section (departures from the sketch recorded inline),
  and 41 → 42 lessons in README and `docs/tutorial-interactive.md`.

## Verification

`go test ./... -race` green; `go vet ./...` clean; `gofmt` clean; the WASM
target builds. church_mobile (which `replace`s to this working tree) builds
and its app tests pass unchanged — everything here is additive.

## Next

1. Adopt the bundle in church_mobile: `screenHeader`/`headerAction` → AppBar,
   `noticeStrip` → Banner, `emptyNote`/`busyNote`/`errorRetry` → EmptyState,
   `chipRow` → ChipStrip. Its goldens are the useful confirmation, exactly as
   they were for `HideTrailingCount`.
2. B1–B3 in one renderer pass (horizontal scroll, `OnEndReached`, sticky
   headers). ChipStrip is now a third waiting consumer for B1, alongside the
   sermon month bands and the events tab for B3.
3. Plan A3: Calendar / DatePicker.
4. Still latent from last session: `components.Chip`'s selected look (Surface
   bg, Primary ink) reads as the *less* prominent state, so a filter row looks
   inverted. Now more visible, since ChipStrip puts a row of them on screen in
   both the tutorial and any adopting app.
5. Not done, noticed here: there is no `Role`/`AccessibilityRole` prop, so
   nothing in this bundle can announce itself as a banner, a heading or a
   search landmark. Skeleton's labelled-container trick is the workaround, and
   its doc names the limit. The A1 plan already flagged the same gap for
   DataTable's `<table>` semantics — one core follow-up would serve both.
