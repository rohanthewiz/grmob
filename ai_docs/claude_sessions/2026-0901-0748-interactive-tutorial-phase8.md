# Interactive Tutorial — Phase 8 (finale): Chapter 8 (Robustness) + docs polish

Session: https://claude.ai/code/session_01RqyVsuqXEL719QsPhLf8ht

## Goal

Phase 8 of eight — the finale. Chapter 8 "Robustness" added to
`examples/tutorial` (curriculum now 40 lessons across 8 chapters), plus the
polish items from the Phase 1 plan: a docs page for the tutorial, an mkdocs
nav entry, and the README mention. The eight-phase plan is now complete.

## The caveat, resolved

Phase 7's doc warned: the tutorial's TestMain runs `SetDebugMode(true)` and
every test ends with `assertNoConcerns` — demos that deliberately provoke
panics/concerns would trip every test that renders them (the curriculum-walk
test renders ALL lessons). Resolution: **every provoker defaults to off**, so
walking the curriculum records nothing; the chapter's own tests arm them,
assert the expected concern WAS filed (`assertOnlyConcernKind` — at least one
of the kind, none of any other), then disarm, clear the collector, drive one
benign pass, and assert it stayed empty. No sandboxed manager needed.

## What was built

### chapter8.go — five lessons (no framework changes)

Through-line: the framework assumes app code will fail; one failure costs a
panel or a frame, never the process.

- **8.1 Error boundaries** — inbox panel with the planted `messages[0]` read;
  checkbox OUTSIDE the boundary empties the slice (stays tappable while the
  panel is down — and that's the honest scoping story). Custom fallback shows
  the real `err.Error()`. Unticking shows the non-latching heal. Child kept
  hook-free (boundary gives it a private child context/hook namespace —
  state reaches it by closure, same shape as 7.4's themed subtree).
- **8.2 When a handler panics** — one handler advances counters A and B with
  `panic(...)` armed between the writes; driver's `guardHandler` survives it,
  the visible skew IS the half-applied-state lesson; "Repair (set B = A)"
  teaches repair-is-app-policy.
- **8.3 Debug mode** — live inspector over `core.Concerns()` (kind ×count +
  detail), duplicate-key provoker (`core.If` + two `Keyed("dup", …)` rows),
  "Clear concerns" button, and the REAL process-wide `SetDebugMode` checkbox.
  Key mechanic: the switch and collector are globals, not state — writing
  them marks nothing dirty, so each control also bumps a throwaway `poke`
  state ("state is the only render trigger", taught as a key point).
  Inspector renders BELOW the provoker in the tree, so a concern filed this
  pass is already visible to it (renderAll renders children in order).
- **8.4 Cached: freeze the static** — package-level
  `var cachedStamp = core.Cached(stampCard(...))` beside a per-pass live
  twin; both stamp `time.Now()` at render, so staleness IS the display. The
  probe is otherwise contract-clean (no hooks/callbacks/theme reads) so the
  debug bypass files no cached-* concern — itself asserted. Mode caption
  reads `core.IsDebugMode()` live: tests see "Cached is bypassed", the
  browser (debug off) sees the actual freeze. Cross-references 8.3's switch.
- **8.5 The whole model** — finale: the model in one sentence, Chapter 1's
  counter annotated with chapter refs, live stats from
  `len(flatLessons)/len(Chapters)`, where-next pointers, completion toast
  ("Take a bow 🎉" → `ShowToast`).

### The initialization-cycle fix (lesson.go)

8.5's Body closure references `flatLessons`/`Chapters`; Go's init-cycle
analysis traces references inside closures transitively, so
`var Chapters = []Chapter{ … chapter8() … }` became a compile error
(InvalidInitCycle) even though the reads only run at render time. Fix: both
vars are now bare declarations populated in a single `func init()` in
lesson.go (init runs after all package vars exist, dissolving the cycle);
rationale documented at the site.

### chapter8_test.go — six tests, all green on first run

- Boundary: fallback substitution with the real panic message, failed
  panel's content absent, `render-panic` concern positively asserted, heal.
- Handler: lockstep → armed panic → `A: 2 / B: 1` + "Skewed by 1" +
  `handler-panic` concern → repair. (The `panic(...)` line in verbose test
  output is the driver's expected stderr log of the RECOVERED panic.)
- Debug ×2: record → persists after disarm → demo's Clear empties it; and
  the zero-cost test — debug off via the demo's own switch, bad list renders,
  collector stays empty (with `t.Cleanup` restoring debug on).
- Cached: both stamps render, bypass caption asserted, pass counter live,
  `assertNoConcerns` (meaningful: bypass measured the probe and found it
  clean).
- Finale: stats string computed from the live curriculum, toast recorded via
  `SetSystemEventHandler` (ch6's recorder pattern, restored on cleanup).

Checkbox indices per lesson (toggleCheckbox is positional): 8.1 idx0 explode;
8.2 idx0 explode; 8.3 idx0 = debug switch, idx1 = provoker.

### Docs polish

- **docs/tutorial-interactive.md** (new) — what it is, two-command browser
  run (`GOOS=js GOARCH=wasm go build -o wasm/main.wasm ./wasm` + serve
  wasm/), native-bridge note, chapter table, "how it stays honest" (headless
  tests under debug mode).
- **mkdocs.yml** — nav: "Tutorial — Interactive" above the Todo tutorial.
- **README.md** — "📖 Tutorial" → "📖 Tutorials", leading with the
  interactive tutorial + run commands.
- **docs/index.md** — "Learn by tapping" row in the where-to-go-next table.

## Facts learned/confirmed this phase

- Go init-cycle detection sees through function literals: a package var
  initializer calling a function whose closures read that var (even
  render-time-only) is `InvalidInitCycle`. Populate in `init()` instead.
- `SetDebugMode`/`ClearConcerns` don't mark any tree dirty — demos driving
  them must pair each write with a state bump or the effect renders late.
- `guardHandler` (render/manager.go:331) files `ConcernHandlerPanic` with
  the callback ID in the detail — assert concerns by Kind, not detail
  (per-pass IDs vary, defeating dedup across passes anyway).
- ErrorBoundary's concern detail (`"ErrorBoundary caught a panic from %T"`)
  is stable across passes → dedups to one entry with rising count.
- Reading `core.Concerns()` during render is safe (snapshot); sibling order
  makes same-pass findings visible to a later-rendered inspector.
- The wasm host does NOT set debug mode — browser runs show production
  behavior (real Cached freeze; provoked bad lists record nothing until the
  demo's own switch is flipped).

## Verification

- `go vet ./...` clean; full `go test ./...` green; gofmt clean;
  `go test -race -count=2 ./examples/tutorial/` green;
  `GOOS=js GOARCH=wasm go build -o <scratch>/main.wasm ./wasm` compiles.
- Browser eyeball still pending (same claude-in-chrome localhost blocker as
  Phases 1/6/7): build wasm/main.wasm and `python3 -m http.server 8080` from
  wasm/.

## State

The eight-phase interactive-tutorial plan is COMPLETE: 40 lessons, 8
chapters, tests, wasm host pointed at it, docs page, nav, README. Possible
follow-ups (not planned): human browser pass over all 40 lessons; merging
this worktree branch (`worktree/grmob-interact-tutorial`) to master; a
completion-quiz phase (app.go notes "opened, not completed" is deliberate).
