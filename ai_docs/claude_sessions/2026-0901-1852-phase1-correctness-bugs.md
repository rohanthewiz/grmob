# Phase 1: the fourteen correctness bugs

Session: https://claude.ai/code/session_011WASqH3Z74UCj6VWGcQKAV

## Goal

Start on `ai_docs/plans/fable-pre-0.1.0-analysis.md`. The plan's own
suggested order puts Phase 1 first — items 1.1 through 1.14, "fix before
tagging v0.1.0" — each with a test that would have caught it. That is what
this session did, end to end. Phases 2 through 7 are untouched.

Baseline before starting (all green): `gofmt -l .`, `go vet ./...`,
`go test ./...`, `go test -race ./...`, `GOOS=js GOARCH=wasm go build ./wasm`,
`wasm/verify/run.sh`, `ios/verify/run.sh`. Same set green after.

## Method

Every fix was written, then the *pin* was written, then the source file (not
the test) was `git stash`ed and the pin re-run to prove it fails without the
fix. Fourteen for fourteen. That step caught two weak tests — the push-order
test originally failed with a misleading "expected at least two arrivals"
because it checked before the held-open delivery landed, and the Swift half
of the payload-guard test used a "guard textually precedes parse" rule that
Swift's `guard let x = (try? parse) as? [Any]` idiom cannot satisfy.

## Three commits

- `78f7aa0` Phase 1.1–1.6
- `526e378` Phase 1.7–1.9
- `c441809` Phase 1.10–1.14

## What changed, item by item

### 1.1 — `WithTheme`/`WithConfig` aliased the parent's hook slots

`core/context.go`. Both copies carried `slots []any` and `Cursor int` **by
value**, so a stateful child inside a theme and one outside it both landed on
slot 0; pass 2 read a string out of an int slot and panicked (swallowed by the
Manager as `"[]"` plus a log line, so the screen just stopped updating).

Fix: a `hooks *Context` field, set only on those copies, plus `hookOwner()`.
`NewState`, `UseChildContext` and `Reset` operate on the owner; the `State`
closures capture the owner so they outlive the transient copy. Two other
cursor readers had to follow — `cached.go`'s `debugRender` (whose
hook-consumption check would otherwise silently pass for every cached view
inside a theme) and `error_boundary.go`'s `renderRecovered`.

Pins: `core/context_scope_test.go`, three tests across three passes.

### 1.2 / 1.3 — the push pump

`render/manager.go`. The pump rendered under `mu`, released it, *then* called
the listener. A dispatch landing in that gap ran pass N+1 and enqueued it
first; both native runtimes funnel the two patch paths into one FIFO and
apply in arrival order, and paths are positional, so N+1 landed on the pre-N
tree. New `renderAndPush()` holds `mu` across both steps. The delivery call is
also wrapped in `core.Guard` — an unrecovered throw out of `GrMobApplyPatches`
killed the pump goroutine, and since it is the only consumer of
`renderRequests`, every later `State.Set` filled the 1-slot buffer and was
dropped while taps kept working.

Pins: `render/push_order_test.go`. The ordering test uses a listener that
holds delivery open for 150ms and signals on entry, so the dispatch is fired
at exactly the wrong moment; without the fix it records `[2 1]`.

### 1.4 — nil handlers in `Chip` / `InputRow`

`components/chip.go`, `components/input_row.go`. Both passed nil straight to
core, which registers it and calls it unguarded. Now the same no-op
substitution `components/button.go` and `segmented_control.go` already used.
This also makes `InputRow.OnChange`'s doc ("read-only in practice") true,
which closes doc item 6.1 #22.

### 1.5 — `MaxHeight()` wrote `Style.MaxWidth`

`Style` had no `MaxHeight` field at all. Added the field, the merge clause,
and `core/style_props_test.go` — a table over all six size constraints plus
the pair that actually collided. The reflective `TestUseStyleMergesEveryField`
picked up the new field for free.

### 1.6 — `UseInterval` / `UseTimeout` dead after `Close` + remount

`hooks/interval.go`. `close()` stopped the ticker but left
`started`/`scheduled` set, and the record lives in a hook slot on a context
that survives the Close. Both hosts re-mount exactly that way
(`wasm/main.go`'s `renderInitial`, `mobile.Register`), so after one re-mount
no timer ever fired again. `OnClose` now clears the flag under `rec.mu`
(which is why `started` needed the lock it previously did not) — matching the
drain-not-terminal semantics `cleanupRegistry` documents.

### 1.7 — stale DOM listeners reusing positional callback IDs

`wasm/grmob-runtime.js`. The update-props loop walked only the keys present in
`Changes`, so a node that dropped a handler kept the ID it last saw. IDs are
positional and re-derived per pass, so the next pass gave that ID to a
different node and clicking the stale element fired the other node's handler.
`update-props` carries the *whole* new props map, so `pruneStaleListeners`
deletes any `listener_*` dataset entry absent from it; the DOM listener stays
attached and goes inert (every listener re-reads its ID at dispatch time),
which also means a returning prop does not stack a second one.
`applyEnterKeyHint` is now unconditional on the update path, matching
`createElement` — the old `"imeAction" in Changes || "onSubmit" in Changes`
gate left a field that lost its submit action still advertising one.

### 1.8 — `Push`/`Replace` before the Navigator's first render

`core/navigation.go`. `takeTop` seeded `initial` only into an *empty* stack,
so a deep-link handler running before the first pass became the whole stack:
depth 1, `CanPop` false, Back exits the app. New `navigatorState.rooted`; the
seed is spliced *beneath* whatever was pushed. `Replace` and `Reset` are both
statements about the bottom of the stack, so they mark it rooted themselves —
`Replace` on an empty stack now installs the route instead of no-oping.

### 1.9 — `ReceiveEventPayload` could not dispatch numeric callbacks

`core/event.go` had no `float64` case, and JSON has one number type, so every
numeric event fell to the void branch and missed the int callback map
entirely. Added `float64` and `int` cases plus `float64` to the inner envelope
unwrap.

### 1.10 — `OnLongPress` promised, never wired; `OnTouch` dead

The largest item. Both renderers read the gesture off containers and leaves
through one shared path (`grMobBox`'s `onLongPress` argument on iOS,
`gestureModifier` on Android) — but a Button draws its own control and took
neither, so the gesture was unreachable on the one node type that exists to
be pressed.

- **Compose**: material3's `Button` has no long-click slot, and a
  `combinedClickable` on its modifier sits *outside* the Button's own
  clickable and never sees a pointer event. New `GrMobLongPressButton`
  (Surface + `combinedClickable` + a padded Box for the label), taken **only**
  when the prop is present — every existing button stays on the material3
  path untouched.
- **SwiftUI**: `.simultaneousGesture(LongPressGesture(minimumDuration: 0.5),
  including: onLongPress.isEmpty ? .subviews : .all)` plus a `longPressFired`
  `@State` flag, because a *simultaneous* gesture is by name allowed to also
  fire the Button's tap.
- **DOM**: there was no long-press at all — `mapEventName`'s fallback derived
  the nonexistent `"longpress"`. `attachLongPress` synthesizes it from
  `pointerdown` + a 500ms timer, disarms on
  `pointerup`/`pointercancel`/`pointerleave`, and suppresses the click that
  follows via a `longPressFired` dataset flag read in `eventQualifies`.
  `onTouch` had the same non-event problem and now maps to `pointerdown`.

500ms is `UILongPressGestureRecognizer`'s default, Android's
`ViewConfiguration`, and now `LONG_PRESS_MS`.

The wasm verify harness grew a **drainable timer queue**: `load.mjs`'s
`setTimeout` was a stub returning 0, so the gesture was untestable.
`drainTimers(minDelay)` fires what is queued at call time, filtered by delay
so a test can fire the 500ms gesture without also running `waitForWasm`'s
100ms self-requeueing poll.

Doc comments on `core.OnLongPress`, `core.OnTouch` and `core.Button` were
rewritten to say what is actually true (closing 6.1 #21).

### 1.11 — form rules trimmed inconsistently

`Required`, `Email`, `Integer`, `Range` trimmed; `MinLen`, `MaxLen`,
`Pattern`, `OneOf` did not. So `Required + MinLen(3)` accepted `"ab "` while
the submit handler stored `Values.Trimmed` and got two characters, and
`Pattern(^[0-9]{5}$)` rejected `" 12345"` that `Range(1,99999)` accepted on
the same text.

One policy, applied in one place — `optional()` trims, and a whitespace-only
value is empty. The four per-rule `TrimSpace` calls are gone (redundant now).
Values themselves stay raw; `Values.Trimmed` is the accessor. Written up
under a new "# Whitespace" heading in `forms/doc.go`.

### 1.12 — `RenderAndGetPatches` broken; dead subscribe API

It reset `r.context.Cursor = 0` instead of `Reset()`, so every child scope
resumed from the previous pass's cursor and appended a fresh slot per render;
it also never called `PurgeUnusedCallbacks` or `ClearDirty`. Kept (the
"mount-or-diff, don't tell me which" shape is useful) but now delegates:
`RenderInitial` was split into `renderInitialLocked`, and the function picks
that or `renderAgainLocked` on `currentTree == nil`. The unexported `render[T]`
helper — a byte-for-byte duplicate of `renderJSON` with one caller, the
deleted body — went with it.

`SubscribeRender` / `RegisterRender` deleted. Nothing ever triggered the
`"render_N"` ids they minted (`State.Set` has always notified the hardcoded
`"default"` key), while the map grew one entry per call. **This is an
exported-API removal** and belongs in the CHANGELOG.

### 1.13 — Android `applyPatches` threw on a non-array payload

`TreeStore.kt` called `JSONArray(json)` unguarded. `render.renderJSON` returns
`{"error":"failed to encode JSON"}` when a payload will not marshal (one NaN
in Props is enough), and that string travels the ordinary delivery path — so
it threw out of the main-thread Handler and took the app down, burying the
encode failure. Now mirrors the guard `mount` and Swift already had.

Pin: `mobile/verify/payloadguard_test.go`, covering all four
mount/applyPatches × Kotlin/Swift combinations so the pair stays a pair.

### 1.14 — iOS Column `Align(AlignStretch)` did not stretch

`FlexChildren` (which decides whether a child *accepts* the stretched size)
read `alignItems` alone, while `GrMobFlexStack` (which decides *placement*)
read the `Align` fallback via `crossAxisValue`. So a Column laid out stretched
and rendered unstretched. Compose's `isColumnStretch` had carried the fallback
since the List fix. Both are now in the existing
`TestListStretchFillReadsTheAlignFallback` table, which holds the two levels
(binding reads the helper; helper still reads `Align`).

## Coverage

The plan predicted the three low packages would rise. Two of them did; the
third had no Phase 1 items.

| Package | Before | After |
|---|---|---|
| core | 73.3% | 78.8% |
| render | 62.9% | 81.7% |
| mobile | 38.5% | 38.5% |
| components | 98.7% | 98.8% |
| forms | 97.0% | 97.0% |
| hooks | 100% | 100% |

## Caveats to carry forward

1. **The Kotlin change is not compiled.** No `kotlinc`, no `gradle` on this
   machine. `android/` is verified only by the source-reading tests in
   `mobile/verify`. `GrMobLongPressButton` needs a real Gradle build before
   the tag. (Related: plan item 7 wants the Gradle wrapper tracked, which
   would make this checkable.)
2. **`SubscribeRender`/`RegisterRender` removal** must land in the CHANGELOG
   before v0.1.0.
3. Neither native gesture path has been exercised on a device — Swift is
   type-checked by `ios/verify`, nothing more.
4. `wasm/main.go`'s `renderInitial` still re-mounts over the same global
   `ctx` (plan item 2.2). 1.6 fixed the hooks half of that; the stale-slot
   half is still open.

## Where the plan stands

- **Phase 1 — done.** All fourteen.
- Phase 2 (stability hardening), 3 (renderer parity), 4 (performance),
  5 (quick wins), 6 (documentation), 7 (repo hygiene) — untouched.

Plan's next step is 6.1 (22 wrong-doc entries) and the 6.3 CHANGELOG
additions, both of which the tag depends on, then tag `v0.1.0` and add CI.
