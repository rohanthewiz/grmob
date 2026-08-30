# GrMob: Feasibility Analysis for Modern Android/iOS Apps

*Analysis date: 2026-08-29*

## Bottom line

The architecture is fundamentally sound — it is essentially React Native's original design
with Go in place of JavaScript — and Go is a legitimately good host language for it. What
stands between this codebase and "modern Android/iOS apps" is not the Go core (which is
close), but the native renderer surface and the bridge's one-way, synchronous shape. Both
are fixable with code, and one strategic choice (render into Compose/SwiftUI rather than
raw Views) cuts the remaining work by an order of magnitude.

## What the architecture actually is

- Go components build a `core.Node` tree (`core/node.go`).
- State lives in cursor-indexed slots on a `Context` (React-hooks style, `core/context.go`).
- The reconciler diffs old vs. new trees into JSON patches (`reconcile/patch.go`).
- A thin native shell applies patches — `PatchRenderer.kt` on Android, `grmob-runtime.js`
  for WASM. Events flow back as string callback IDs (`core/event.go`).

This is a proven pattern (React Native, NativeScript, Lynx). Go specifically is fine on
both platforms: AOT-compiled (no JIT, so no App Store problem), `gomobile bind` produces
an AAR/xcframework, GC pauses are sub-millisecond, and actual drawing is native. Cost is
~10–15MB of binary plus bridge ceremony.

## Strengths

- Clean DSL: `View`/`ComponentFunc`, functional style props, `If`/`Match`/`For`,
  thoughtful theming design (`docs/ui-architecture.md`).
- The `Node` tree is renderer-agnostic — three backends already exist (JSON, HTML, WASM).
- Small core (~1,900 lines) — every problem below is reachable.

## Gaps (concrete, in the code today)

1. **Reconciler undermines its own premise.** `styleChanged` compares style *pointers*
   (`a != b`), and styles are freshly allocated every render — so every styled node emits
   `update-style` on every render. Callback IDs come from a monotonically increasing
   global counter (`core/event.go`), so every button gets a new `onClick` ID each render,
   making `propsChanged` true for every interactive node every time. The "minimal diff"
   is currently a near-full tree broadcast. Fix: value-compare styles; derive callback
   IDs from the render path so they are stable across renders.

2. **Path-based node identity is fragile.** `TargetID` is a positional path
   (`root/0/2`). After one `add-child`/`remove-child`, sibling paths shift, but the
   Kotlin `viewMap` keeps stale paths — later patches can target the wrong views. Keyed
   reconciliation punts to `replace` on any key mismatch, so reordered lists destroy and
   rebuild views (losing scroll position, focus, input state). Needs stable per-node
   identity, then real move patches.

3. **No Go→native push channel.** The bridge is strictly request/response
   (`GrMobBridge.kt`). `UseInterval`, network responses, or any goroutine calling
   `state.Set()` cannot tell the native side "re-render now" (WASM polls `IsDirty`).
   Fix: `gomobile bind` lets native code register an interface implementation with Go;
   Go pushes patches to a `Renderer` callback on the UI thread. Small work, unblocks the
   whole async world.

4. **Global mutable state.** `navigatorStack` (`core/navigation.go`) and the callback
   registries are package-level globals; `Context` has its own `callbackMap` that is not
   the one actually used. Timers fire on goroutines while events arrive from the UI
   thread with no render lock. Fix: consolidate onto `Context`, one render mutex, marshal
   mutations onto one goroutine.

5. **Android renderer is a proof-of-concept.** Text/Button/Row/Column only, zero styles
   applied, lists are `LinearLayout` (no recycling). Text input/IME, scroll physics,
   images, gestures, animations, accessibility, keyboard avoidance, lifecycle — all
   absent. iOS has no runtime at all. This is where the honest 80% of remaining work
   lives.

6. **Zero tests.** The reconciler is pure-function code where table-driven Go tests are
   cheap and would have caught gaps 1 and 2.

## Strategic recommendation

Do not build the renderer against classic Android Views and UIKit. **Map the GrMob node
tree into Jetpack Compose and SwiftUI.** Both are declarative tree-diffing systems: hand
each new tree (or subtree) to Compose/SwiftUI and let their mature reconcilers handle view
identity, animation, recycling (`LazyColumn`/`List`), and accessibility. The Go reconciler
then only needs to answer "did anything change," not produce surgically precise patches.
This converts the hardest, longest-tail problem into a mostly-mechanical
`Node → @Composable` / `Node → some View` mapping.

## Feasibility verdict by app type

- **Business/CRUD/forms/dashboards**: very feasible. With the fixes above, ~2–4 months of
  focused work to a usable Android runtime; similar again for iOS.
- **Media-rich apps with standard interactions** (feeds, chat): feasible, dependent on the
  Compose/SwiftUI route and a list-virtualization protocol.
- **Gesture-driven, animation-heavy, 120Hz apps**: the serialize-JSON-over-bridge model is
  the ceiling (same reason React Native built JSI). Declare animations in Go but drive
  them natively, or accept the ceiling.

## Attack order

1. **Reconciler correctness fixes + tests** — value-based style comparison, panic-safe
   props comparison, nil guards, safe removal ordering, table-driven test suite.
2. **Stable callback/node identity** — path- or key-derived IDs so diffs stop churning;
   prerequisite for real move patches.
3. **Go→native push channel** — native registers a `Renderer` interface via gomobile; Go
   pushes patches for timer/async-driven renders.
4. **Compose-based Android renderer** replacing `PatchRenderer.kt`.
5. **iOS via SwiftUI** once the protocol is settled.
