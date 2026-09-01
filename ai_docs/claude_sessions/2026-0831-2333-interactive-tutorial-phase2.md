# Interactive Tutorial — Phase 2: Chapter 2 (State, Events & Lists)

Session: https://claude.ai/code/session_01Nm3azgkKS6SWsFoqJZEdUh

## Goal

Phase 2 of the eight-phase interactive tutorial: add Chapter 2
"State, Events & Lists" to `examples/tutorial`. The framework from Phase 1
needed exactly one change — appending `chapter2()` to `Chapters` in
lesson.go — everything else (IDs 2.x, home rows, progress, the Next/Prev
curriculum walk and its tests) picked the new chapter up automatically,
which was the design's whole bet.

## What was built

- **chapter2.go** — five lessons, each with a live demo. The chapter's
  through-line, stated in its header comment: state is the source of truth,
  the tree is a pure function of it, events are the only place state
  changes.
  - **2.1 State: the counter** — NewState slot mechanics (positional slots,
    call order, rules of hooks), Set → dirty → re-render. Demo: −1/+1/Reset
    counter with the count in Primary ink, captioned "a pure function of
    the slot".
  - **2.2 Events & handlers** — behavior props (`OnClick`/`OnLongPress`) on
    a plain Card; handlers as the only place state writes belong; immutable
    slice updates. Demo: gesture card feeding a capped (5), newest-first
    event log with a running total ("3 · tap" lines) and Clear. The
    `record` helper trims on write, showing the copy-don't-mutate idiom on
    both ends of the slice.
  - **2.3 Controlled inputs** — value in, intent out; the field has no
    private copy. Demo: Input echoing into a greeting + rune count
    (`utf8.RuneCountInString`), an UPPERCASE-on-the-way-in transform inside
    onChange, and a Clear button writing state from outside the field.
  - **2.4 Conditional rendering** — If/IfElse/Match, plus the rule:
    condition views, never hooks. Demo: Loading/Ready/Error
    SegmentedControl driving one `Match(status.Get(), Case(0,…), Case(1,…),
    Default[int](…))`, and a checkbox driving a separate `core.If` showing
    the raw status value. Error branch is the Default, styled with
    `Colors.Error` border/ink.
  - **2.5 Lists & keys** — For + Keyed with stable IDs from the data;
    per-row NewState in a changing list reads a neighbor's slot, so rows
    stay pure functions of their item (`taskRow(t, remove)`). Demo:
    seeded 3-task list, "＋ Add to top" (inserts at head — where keys
    matter), per-row ✕ remove; ids never renumber and the caption says the
    left number is the key, not the position. `demoTask{id, title}` +
    `nextID` state.
- **chapter2_test.go** — five liveness tests, one per demo:
  counter arithmetic (including −1, note U+2212 in the label); both
  gestures on one node — long-press dispatched via its own registered
  `onLongPress` callback ID (void-callback channel, same as click); typing
  via a new `typeInto` helper (first Input in tree order,
  `DispatchTextCallback`); Match branch switching by tapping segment chips;
  keyed add/remove asserting actual `Node.Key` values ("task-1"…"task-4")
  via a new `hasKey` helper.
- **app_test.go** — one line: the test `node` struct gained a `Key string`
  field so tests can see what the reconciler sees (`core.Node.Key`
  marshals as "Key").
- **lesson.go** — `chapter2(),` appended to `Chapters`.

## Verification

- `go test ./...` fully green; debug mode audits every pass (TestMain).
- Tutorial files `gofmt` clean; lint fix applied (`for range 3`).
- `GOOS=js GOARCH=wasm go build ./wasm` compiles (needs `-o` to a file —
  bare `go build ./wasm` fails with "output already exists and is a
  directory" because of the wasm/ dir name).
- Pre-existing, untouched: `gofmt -l` flags `examples/todoapp/store.go`
  (import ordering left over from the govinci→grmob rebrand commit
  8af25e8). Not part of this change; left alone.

## API facts learned/confirmed this phase

- `core.Match`/`Case`/`Default[T]` are generic over comparable; Default
  needs the explicit type param at the call site.
- Behavior props register as `Props["on"+event]` — the card carries both
  `onClick` and `onLongPress` IDs; both dispatch via `DispatchCallback`.
- `render.Manager` has typed dispatchers: DispatchTextCallback,
  DispatchBoolCallback, DispatchIntCallback.
- `core.Node.Key` is exported and JSON-visible, so tests can assert keying
  directly instead of inferring it.
- Segmented chips still render as Buttons (tap-by-label works for
  "Loading"/"Ready"/"Error" exactly as it did for "Column" in ch.1).

## Next session: Phase 3

Add `chapter3()` (Hooks & Effects) to `Chapters`: UseInterval clock,
UseTimeout, UseEffect (deps + once-on-mount), UseMemo (inline, for
expensive derivations), UseReducer (atomic dispatch; reducer must not
dispatch or mutate). Source of truth: `docs/concepts/state-and-hooks.md`
§"Side effects: the hooks package" and the `hooks` package itself. Tests
will need care around time-based hooks — prefer driving state and
asserting structure over sleeping; check how existing hooks tests fake or
wait on tickers before writing lesson demos.
