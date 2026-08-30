# Compose-Based Android Renderer (Attack-Order Step 4)

*Design date: 2026-08-29. Companion to `grmob-mobile-feasibility-analysis.md`.*

## The one strategic decision

Patches are applied to a **Kotlin data tree, never to views**. The old
`PatchRenderer.kt` kept a `path → View` map and mutated Android Views by hand —
which meant re-implementing view identity, recycling, animation, and
accessibility from scratch (the "honest 80%" of the feasibility analysis).
Instead, the Compose runtime holds a `GrMobNode` mirror of Go's tree whose
mutable aspects are Compose **snapshot state**; composables read that state and
Compose's own reconciler does everything downstream.

```
Go side                     bridge                Kotlin side
───────                     ──────                ───────────
core.Node tree ─ diff ─▶ patch JSON ─▶ TreeStore.applyPatches
                                          │ mutates snapshot state
                                          ▼
                                    GrMobNode tree ─ read ─▶ @Composables
                                                                 │ (Compose reconciler:
                                                                 ▼  identity, recycling, a11y)
                                                              Android UI
```

## How patches short-circuit into recomposition

Each `GrMobNode` splits its mutable surface into independently observable
state so a patch invalidates the narrowest possible composable scope:

| patch                    | mutates                              | recomposes                       |
|--------------------------|--------------------------------------|----------------------------------|
| `update-props`           | `node.props` (`mutableStateOf`)      | only readers of this node's props |
| `update-style`           | `node.style` (`mutableStateOf`)      | only this node's box/text styling |
| `add`/`add-child`/`remove`/`remove-child`/`replace` | `parent.children` (`SnapshotStateList`) | only the parent's children loop |

Children are emitted under `key(child.key or index)`, so structural changes
preserve sibling composition state (focus, scroll, text selection) — the thing
the positional-path View renderer could never do.

Paths are resolved against the current tree **at apply time** (`TreeStore`
walks `root/0/2` through live children), so there is no stale path cache. The
Go-side ordering guarantees (patches in emitted order, sibling removals
highest-index-first) are exactly what make the walk safe within a batch.

## Threading model

One rule: **all patch application happens on the main thread, in arrival
order.** Both delivery paths funnel into a single `Handler.post`:

- **Event path** — composable handlers call `GrMobRuntime.click/textChanged/
  toggled/intChanged`, which run the bridge `Trigger*` call on a dedicated
  single-thread executor (a bridge call spans a full Go render pass; the single
  thread also keeps events ordered), then post the returned patches to main.
- **Push path** — Go's `PatchListener.ApplyPatches` arrives on a Go goroutine
  and is posted straight to main.

The Go side completes the story (fixed in this step): `State.Set` slot access
is now mutex-guarded in `core.Context`, so app goroutines can write state while
the pump renders; render passes were already serialized behind
`render.Manager.mu`. `go test -race` covers the storm case
(`TestConcurrentStateWritesDoNotRace`).

## Node → Composable mapping

Material3 is used where interaction/a11y semantics are expensive to hand-roll
(Button ripple, Checkbox, TabRow); GrMob styles flow into their color/shape/
padding slots. Everything else is foundation primitives.

| Go node | Composable | notes |
|---|---|---|
| Text | `Text` | style → `TextStyle` (color, sp, weight = CSS numeric scale, align) |
| Button | m3 `Button` | bg/radius/padding via slots, not modifiers, to keep ripple correct |
| Input / InputPassword / NumericInput / TextArea | `BasicTextField` | controlled-input compromise below |
| Checkbox | m3 `Checkbox` | bool callback |
| Row / Column / Card | `Row` / `Column` | JustifyContent → Arrangement, Gap → spacedBy, FlexGrow → scope `weight` |
| Box | `Box` | |
| Spacer | `Spacer(size.dp)` | |
| Scroll | `Column + verticalScroll` | LazyColumn is future work (needs list-virtualization protocol) |
| SafeArea | `Box + safeDrawingPadding` | |
| Fragment / Theme | children inline | grouping nodes emit no box |
| TabView | m3 `TabRow` + selected child | int callback; only selected child composed |
| Modal | `Dialog` | visible-gated; onDismiss → void callback |
| Image | Coil `AsyncImage` | INTERNET permission in manifest |
| CameraView | styled `Box` stub | real CameraX integration is its own pass |
| unknown | `Column` of children | forward compatibility |

Style mapping lives in `GrMobStyle.boxModifier()`, applied in CSS box-model
order (margin → size → shadow → clip → background → border → padding — the
order is load-bearing). JSON keys are Go's exported field names verbatim
(no json tags on `core.Style`). `Display: "none"` skips composition;
`"hidden"` keeps space via `alpha(0)`. Web-oriented fields (Position, ZIndex,
Transition, pseudo-states) are deliberately unmapped.

**Controlled inputs**: Go owns the value, but the IME needs instant echo and
the Go round trip is async. Resolution: the field is locally owned *while
focused* (every edit sent upstream; late echoes never snap the cursor), and
Go-owned when unfocused (async upstream changes land as soon as the user isn't
typing).

## File map

- `android/app/src/main/java/com/grmob/runtime/` — app-agnostic runtime:
  `GrMobNode.kt`, `GrMobStyle.kt`, `TreeStore.kt`, `GrMobRuntime.kt`
  (+ `GrMobBridge` interface), `Renderer.kt`.
- `android/app/src/main/java/com/grmob/app/` — shell: `MainActivity.kt`,
  `GomobileBridge.kt` (impl over gomobile-generated `mobile.Mobile`).
- `examples/mobileapp/` — demo Go app bound into the AAR; its `init()` calls
  `mobile.Register`. Exercises all four event kinds + the push channel, with a
  Go-side smoke test (`app_test.go`) that replays the exact Kotlin call
  sequence.
- `android/build.sh` — `gomobile bind ./mobile ./examples/mobileapp` →
  `app/libs/grmob.aar` (replaces the broken bind-the-repo-root + raw-JNI
  setup).

## Fixed along the way (Go side)

- Int callbacks (TabView) folded into the render-pass ID system — their
  counter never reset, so `onTabChange` churned IDs every render and entries
  leaked; `mobile.TriggerIntCallback` was missing entirely.
- `hooks.UseInterval`/`UseTimeout` reserved a cursor position without
  appending a slot, silently cross-wiring every `NewState` slot that followed
  in the same render (bool landing in a string slot ⇒ type-assertion panic).
  They now reserve through `NewState`.
- `core.Context` slot access mutex-guarded (the step-3 caveat).
- `htmlout` `%d`-on-float vet errors (pre-existing, broke `go test ./...`).
- **gobind linkage trap** (cost a debugging session on-device): a bound package
  with zero *bindable* exported symbols gets a generated Java class but is
  never imported by the gobind glue — so its `init()` (the app registration!)
  never runs, and the bridge crashes on a nil manager. An app package must
  export at least one bindable symbol; `mobileapp.AppName()` exists solely for
  this, with the comment explaining why.

## Verified on-device (emulator, API 36)

The demo APK was built and exercised end-to-end: tab switching (int events),
Increment (void events, `Count: 0 → 1`), text input with IME (`Hello, Ada!`),
checkbox toggle (bool events), and the interval clock advancing with no native
event in flight — the Go→native push channel driving recomposition.

## Known limits / next

- Lists are non-virtualized (`Scroll` = eager Column). A `LazyColumn` node
  type + windowing protocol is the next renderer-side milestone.
- Keyed reorders still arrive as `replace` (Go-side positional identity);
  Compose `key()` is already in place to benefit when real move patches land.
- `UseEffect` in `hooks` still uses a process-global index that never resets
  per pass — same bug class as the interval one; untouched this session.
- Rotation remounts via `RenderInitial` (Go is the source of truth); fine for
  now, `rememberSaveable` is irrelevant since state lives in Go.
- Step 5 (SwiftUI) can reuse this design one-to-one: `TreeStore` ↔ an
  `ObservableObject` tree, `boxModifier` ↔ ViewModifiers, same bridge surface.
