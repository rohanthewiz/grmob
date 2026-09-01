# Pre-0.1.0 analysis and remediation plan

*Analysis date: 2026-09-01. Produced by a five-way review (core engine,
renderers and bridges, widgets/forms/examples, documentation audit, repo
hygiene) run against master at e236de2. Every finding below was verified by
reading the code, and the ones marked **repro** were confirmed with a
throwaway test outside the repo. Items marked *plausible* were not
reproduced.*

## Baseline at the time of analysis

All green: `gofmt -l .`, `go vet ./...`, `go test ./...`, `go test -race ./...`,
`GOOS=js GOARCH=wasm go build ./wasm`, `wasm/verify/run.sh`, `ios/verify/run.sh`.
No binary artifacts are tracked. No git tag exists. No CI workflow exists.

Coverage by package (statements):

| Package | Coverage | Package | Coverage |
|---|---|---|---|
| components | 98.7% | htmlout | 96.5% |
| core | 73.3% | mobile | 38.5% |
| forms | 97.0% | reconcile | 97.2% |
| hooks | 100% | render | 62.9% |

The three low ones (`render`, `core`, `mobile`) are exactly where the bugs in
Phase 1 live. Every Phase 1 fix should land with a test that would have
caught it, which also lifts these numbers.

Environment note: the local base Go is 1.25.4 with 1.26.1 auto-downloaded
via `GOTOOLCHAIN=auto`. `go test -cover` on a package with no test files
fails with "compile: version go1.26.1 does not match go tool version
go1.25.4". This is environmental, not a repo defect; installing 1.26.1 as
the base toolchain removes it.

---

## Phase 1 — Correctness bugs (fix before tagging v0.1.0)

Ordered by blast radius. Each item: what is wrong, where, how to fix, how to
pin it.

### 1.1 Hooks inside `core.WithTheme` / `WithConfig` alias the parent's slots — **repro**

- **Where:** `core/context.go:176-177` (`WithConfig`), `:198-199` (`WithTheme`).
  Both construct a new `Context` with `slots: ctx.slots, Cursor: ctx.Cursor`
  and a fresh `lock`. `core.WithTheme` (`core/theme.go:118`) calls
  `ctx.WithTheme(theme)` on every pass.
- **Effect:** a child's `NewState` on the copy reads/appends at the parent's
  *current* cursor and the parent's cursor never advances. Repro:
  `Column(WithTheme(Material, comp{NewState(c,0)}), comp{NewState(c,"s")})`
  panics on pass 2 with `interface conversion: interface {} is string, not
  int`; the inner `Set(42)` is discarded. Under `render.Manager` the panic is
  swallowed as `"[]"` plus a log line, so the screen silently stops updating.
  `examples/tutorial/chapter7.go:406-426` exercises this path.
- **Fix:** do not copy the slice header and cursor. Either (a) share the
  `*Context` and override only `theme`/`config` via a thin wrapper that
  delegates slot access to the parent, or (b) route themed children through
  `UseChildContext` so they get their own scope. Option (a) is the smaller
  change and keeps `ctx.Theme()` semantics.
- **Pin:** a test with stateful children on both sides of a `WithTheme`
  across three passes, asserting both values survive and the outer cursor
  advances correctly.

### 1.2 Pump delivers patches outside the render mutex — ordering race on native

- **Where:** `render/manager.go:143-150`. `out := m.RenderAgain()` acquires
  and releases `mu`; `l.ApplyPatches(out)` runs after release.
- **Effect:** between those two lines a `DispatchCallback` on the
  gomobile events thread can take `mu`, run pass N+1, return, and post its
  patches *before* the pump posts pass N. `GrMobRuntime.kt:70` and
  `GrMobRuntime.swift:70` rely on arrival order == emission order
  (documented in `mobile/bridge.go:18-23`) and TreeStore resolves positional
  paths, so N+1's `root/2` may land on the pre-N tree.
  `TestConcurrentDispatchAndTimerPushesDoNotRace` only checks final state.
- **Fix:** hold `mu` (or a dedicated delivery mutex acquired inside the render
  critical section) across the listener call, so emission and delivery are
  one atomic step. Alternative: sequence-number each payload and have the
  native TreeStores reorder; heavier and touches three runtimes.
- **Pin:** a test with a listener that records the order of received
  passes while dispatches and pushes interleave, asserting monotonic order.

### 1.3 Pump goroutine dies silently on a listener panic

- **Where:** `render/manager.go:147`. No `recover` around
  `l.ApplyPatches(out)`.
- **Effect:** one throw from `GrMobApplyPatches` (WASM) or a Java exception
  surfacing as a Go panic (gomobile) kills the pump forever. Every later
  `State.Set` fills the 1-slot buffer and is dropped; taps still work, so
  the failure looks like "async updates stopped".
- **Fix:** wrap the call in `core.Guard` and log via the same path as
  `logRenderPanic`.
- **Pin:** listener that panics once; assert the next push still arrives.

### 1.4 `Chip` and `InputRow` panic on nil handlers — **repro**

- **Where:** `components/chip.go:60` passes `c.OnTap` straight to
  `core.Button`; `components/input_row.go:127-131` passes `r.OnChange` to
  `core.Input`/`InputWithSubmit`. `core.Button` registers whatever it is
  handed (`core/button.go:38-42`) and `TriggerCallback` calls it unguarded
  (`core/event.go:270-274`).
- **Effect:** `Chip{Label:"x"}` tap → nil-func panic; `InputRow{}` keystroke
  → nil-func panic. Via the Manager it becomes a `[handler-panic]` concern
  and a dead control; via bare `ctx.TriggerCallback` (`examples/chat/main.go:238`)
  it escapes. The InputRow doc at `:78-80` claims "read-only in practice".
- **Fix:** copy the guard `components/button.go:186-189` and
  `segmented_control.go:133-137` already use: substitute `func(){}` /
  `func(string){}` when nil.
- **Pin:** nil-handler cases in `chip_test.go` and `input_row_test.go`.

### 1.5 `MaxHeight` writes `MaxWidth`

- **Where:** `core/style_props.go:154-156`. `Style` has no `MaxHeight` field
  (`grep MaxHeight core/style.go` is empty).
- **Fix:** add `MaxHeight string` to `Style`, set it in the prop, and thread
  it through the enum/field pin tests. Decide whether any renderer should
  honor it (none read `MaxWidth` today either — see Phase 3).
- **Pin:** style-prop test asserting the right field.

### 1.6 `UseInterval` / `UseTimeout` dead after `Close` + remount — **repro**

- **Where:** `hooks/interval.go:58,118`. `close()` stops the ticker but
  `rec.started` / `rec.scheduled` stay true.
- **Effect:** `wasm/main.go:31-44` and `mobile/bridge.go:69-71` both call
  `Manager.Close()` then remount over the same context. After remount no
  interval ever fires again. Repro: 10 ticks before close, still 10 after
  remount + 30 ms. `hooks/cleanup.go` documents drain semantics that these
  hooks do not honor.
- **Fix:** the `OnClose` function clears `rec.started`/`rec.scheduled` under
  `rec.mu`.
- **Pin:** close, remount, assert ticks resume.

### 1.7 Stale DOM listeners reuse positional callback IDs

- **Where:** `wasm/grmob-runtime.js:710-789`. The update-props loop iterates
  only `Object.entries(p.Changes)`; `el.dataset.listener_onClick` etc.
  survive when the key disappears. `applyEnterKeyHint` re-runs only when
  `"onSubmit" in p.Changes`.
- **Effect:** IDs are positional per pass (`core/event.go:50-70`). `cb_3`
  dropped from node A is reassigned to node B next pass; clicking A fires
  B's handler. A field that loses `onSubmit` keeps its keydown listener and
  `enterkeyhint`. `runtime_test.mjs:129` tests value change, not key removal.
- **Fix:** after the loop, walk `el.dataset` for `listener_*` keys absent
  from `Changes`, remove the listener and the dataset entry; clear
  `enterkeyhint` when `onSubmit` is absent.
- **Pin:** node test: prop removed → listener gone → next-pass ID reuse does
  not misfire.

### 1.8 `Push` / `Replace` before the Navigator's first render replaces the initial route — **repro**

- **Where:** `core/navigation.go:103`. `takeTop` seeds `initial` only when
  `len(stack)==0`, but `Push` appends unconditionally.
- **Effect:** `Push(ctx, pushed)` then `Navigator(initial)` → depth 1,
  `CanPop=false`, "pushed" shown, initial screen never exists, Back exits the
  app. Deep-link handlers that run before `RenderInitial` hit this.
- **Fix:** seed the initial route when the stack is empty on `Push`/`Replace`
  too, or record "initial pending" and splice it beneath on first render.
- **Pin:** push-before-render test asserting depth 2 and `CanPop`.

### 1.9 `ReceiveEventPayload` cannot dispatch numeric callbacks — **repro**

- **Where:** `core/event.go:326-331`. No `float64` case; `case nil:` and
  `default:` are identical.
- **Effect:** `{"callback":"int_cb_0","value":1.0}` falls to
  `TriggerCallback(id)` → `voidCBs` miss → silent no-op. Latent because the
  DOM runtime never emits `onTabChange`; the moment it does, tabs are dead.
- **Fix:** add `case float64: TriggerIntCallback(id, int(v))` (and a bool
  case if not already routed).
- **Pin:** payload test with a numeric value.

### 1.10 `Button.OnLongPress` promised, never wired; `OnTouch` dead everywhere

- **Where:** `core/button.go:39-49` says both natives wire it; `GrMobButton`
  reads only `onClick` (`Renderer.kt:383`, `Renderer.swift:619`).
  `core.OnTouch` (`core/behavioral_props.go:29`) is read by nothing; the DOM
  maps it to a nonexistent `"touch"` event.
- **Fix:** Kotlin: `Modifier.combinedClickable` on the button content or a
  custom button; Swift: `.simultaneousGesture(LongPressGesture())`; DOM:
  see 3.1. For `OnTouch`, either delete it or map to `pointerdown` /
  `Modifier.pointerInput` / `DragGesture(minimumDistance:0)`. Update the
  button doc comment either way.
- **Pin:** `mobile/verify` source-reading tests for both renderers.

### 1.11 Form rules trim inconsistently

- **Where:** `forms/rules.go`. `Required` (l.71), `Email` (l.117), `Integer`
  (l.152), `Range` (l.165) trim; `MinLen`/`MaxLen`/`Pattern`/`OneOf`
  (l.58-65, 81-103, 135-142, 180-189) test raw.
- **Effect:** `Required + MinLen(3)` accepts `"ab "`; `signup/app.go:202`
  stores `v.Trimmed("email")` → 2 chars. `Pattern(^[0-9]{5}$)` rejects
  `" 12345"` that `Range(1,99999)` accepts on the same text.
- **Fix:** pick one policy (trim everywhere except where the doc at l.55-57
  justifies raw for the emptiness check) and document it in `forms/doc.go`.
- **Pin:** trailing-whitespace cases for `MinLen`, `Pattern`, `OneOf`.

### 1.12 `RenderAndGetPatches` is broken and unused — **repro**

- **Where:** `render/manager.go:354`. Resets `r.context.Cursor = 0` instead of
  `Reset()`, never calls `PurgeUnusedCallbacks` or `ClearDirty`.
- **Effect:** child-scope state lost every pass; slots grow by one per pass.
  Zero callers in the repo.
- **Fix:** delete it, or make it delegate to `renderAgainLocked`. Also delete
  the dead `SubscribeRender` / `RegisterRender` surface in
  `core/render_manager.go:20,69` (map grows unboundedly if called per render;
  no callers) unless a documented use exists.

### 1.13 Android `applyPatches` throws on a non-array payload

- **Where:** `android/.../TreeStore.kt:48` — `JSONArray(json)` unguarded.
  `render.renderJSON` can return `{"error":"failed to encode JSON"}` (e.g. a
  NaN in Props). `mount` guards it; `applyPatches` does not. Swift logs and
  returns (`TreeStore.swift:41`).
- **Fix:** mirror the Swift guard.

### 1.14 iOS Column `Align(AlignStretch)` without `AlignItems` does not stretch

- **Where:** `ios/.../Renderer.swift:237`. `FlexChildren.stretch =
  alignItems == "stretch"` ignores the `Align` fallback that
  `GrMobFlexStack` (l.303) honors. Kotlin's `ColumnChildren` is correct
  (`Renderer.kt:744`). The existing pin
  (`mobile/verify/alignment_test.go:327`) covers `GrMobList` only.
- **Fix:** read `crossAxisValue` in `FlexChildren` too.
- **Pin:** extend the alignment source test to `FlexChildren`.

---

## Phase 2 — Stability hardening

### 2.1 Native parsers drop nil children; the reconciler keeps them positional — *plausible*

`android/.../GrMobNode.kt:57` and `ios/.../GrMobNode.swift:51` skip nil slots
at parse ("Diff treats nil slots as absent too" is wrong: `reconcile.Diff`
emits `add` at `path/i` when old is nil). Any user `View.Render` returning nil
desyncs `root/2` → `root/1`. The DOM runtime is correct. Latent today because
core builders never emit nil. Fix: keep a placeholder node, or have the
reconciler compact nils before emitting. Also make `core.Keyed`
(`core/view.go:16`) nil-tolerant — it is the one place that dereferences a
child node unconditionally; `Diff`/`renderAll`/`checkDuplicateKeys` all
tolerate nil.

### 2.2 WASM host bypasses the Manager mutex and remounts over a closed context — *plausible*

`wasm/main.go:24-27` calls `ctx.ReceiveEventPayload` directly instead of
`manager.Dispatch*`; benign while js/wasm is single-threaded and handlers have
no yield points, wrong the moment a handler blocks. `renderInitial`
(`:44-49`) calls `manager.Close()` then `render.New(ctx, App)` on the same
global ctx, so prior hook slots survive. Fix: dispatch through the Manager;
`core.NewContext()` per mount (also resolves 1.6's remount half).

### 2.3 Text events are JSON-sniffed

`core/event.go:307-324`: every keystroke is `json.Unmarshal`ed and, if it
parses to an object with `value`, unwrapped. A user typing `{"value":"x"}`
into an Input delivers `x`. The DOM runtime now always sends the envelope at
the top level (`extractEventPayload`), so the inner sniff can go. Envelope
shape should be decided by the host, not by user content.

### 2.4 `UseEffect` has no cleanup return

`hooks/effect.go:53`. An effect that subscribes leaks; effects run as
`go effect()` with no ordering. Quick-win: accept `func() func()` and run the
cleanup via `ctx.OnClose` and on deps change. (API addition; consider for
0.2.)

### 2.5 `TriggerRender` spawns a goroutine per notification

`core/render_manager.go:33`. `requestRender` is a non-blocking select, so the
goroutine is pure overhead; a loop populating 10k rows spawns 10k goroutines.
Call inline.

### 2.6 Dead fields and dead code

- `core/context.go:35-36` `children`/`childrenCursor` are never appended to.
- `wasm/main.go:57-63,157-223` `RequestPermission`, `Permission` consts,
  `TabsComponent`, `HomeScreen`, `DetailsScreen`, `ifThen` are unreferenced;
  `:33` comment says "examples/social" while the import is `examples/tutorial`.
- `permission/camera.go` is entirely commented-out API plus unused types;
  the package is imported by nothing. Delete or implement.
- `examples/components.go` is an orphan `package main` byte-identical to
  `examples/runtime/main.go:10-20`.
- `wasm/grmob-runtime.js:5` `DEBUG` unused. Both natives read an `"alt"`
  prop no builder emits.
- `render/manager.go:70` `if ctx.Theme() == nil` is dead (`Theme()` always
  falls back to `DefaultTheme`).
- `core/style.go:401` `DisplayFlex = "flex"` is an untyped const in the
  wrong block, invisible to the enum-pin tests; type it as `DisplayMode`.

### 2.7 Example-level nits

- `examples/social/pages.go:62` — `Input("", ..., func(string){})` can never
  show typed text; the tutorial (`chapter2.go:182-184`) explains why this is
  the anti-pattern. Fix the example.
- `examples/todoapp/store.go:84-94` — after a failed `bytdb.Open` the store
  re-attempts `Open` (with a log line) on every render pass. Cache the
  failure. *Plausible*: if `SetDataDir` were called after first render,
  `nextID` restarts at 1 and INSERTs collide; shells call it first today.
- `examples/todoapp/app.go:59-60,345-347` — literal `#000000` / dim hex
  instead of `TextPrimary`/`TextSecondary`, against the package's own rule.
- `examples/chat/main.go:73` — message ID is `len(msgs)+1`; collides on first
  removal. *Plausible*, appends only today.
- `examples/tutorial/chapter3.go:209-213` — effect goroutine does a
  read-modify-write `runs.Set(runs.Get()+1)` off the render goroutine.
- `examples/tutorial/chapter3.go:106` — `timeoutRevealDelay = 2500ms` is
  shared with the test, so the tutorial suite sleeps 2.5 s (6.5 of the 12 s
  total). Make it injectable.

---

## Phase 3 — Renderer parity (web pair catches up to the natives)

The Go side emits more than the DOM/htmlout pair honors. Consolidated table
(Compose / SwiftUI / DOM / htmlout); every "missing" is a small,
table-shaped addition.

| Item | Compose | SwiftUI | DOM (wasm) | htmlout | Action |
|---|---|---|---|---|---|
| Padding/Margin `Horizontal`/`Vertical` shorthand | yes | yes | missing (`edgeToCSS` reads four sides only, js:462) | missing | fill unset sides from shorthand in both; drop the hand workaround in `components/separator.go:48-50` |
| `Shadow` | yes | yes | missing | missing | `box-shadow` |
| `Display none` | yes | yes | missing (deliberately dropped, js:412-422) | yes | emit `display:none` after the flex block |
| `Display hidden` | alpha 0 | opacity 0 | missing | **broken**: emits `display:hidden` (invalid CSS); same for `visible` | `visibility:hidden` in both |
| `LineHeight` | yes | approx | missing | missing | `line-height` |
| `AccessibilityLabel/Hint/Hidden` | yes | yes | missing | missing | `aria-label`, `aria-description`, `aria-hidden` |
| `Spacer` axis | size×size | size×size | height only (js:12-14) | height only | set width and height, `flex-shrink:0` |
| `TabView` | yes | yes | bare div, all children shown | missing | tab bar + selected child; emit `onTabChange` (unblocks 1.9) |
| `Modal visible` | yes | yes | yes | missing (always rendered) | honor `visible` |
| `Modal backdrop` | ignored | ignored | yes | ignored | natives: scrim toggle |
| `onLongPress` on containers/Text/Image | yes | yes | missing (`mapEventName` → nonexistent `longpress`) | n/a | `pointerdown` + 500 ms timer, cancel on `pointerup`/`pointerleave` |
| `Spacer size` in update-props | yes | yes | missing | n/a | add branch |
| `FlexDirection` | missing (Row/Column fixed-axis) | missing | yes | yes | low priority; document |
| `Gap` with non-start `JustifyContent` | **dropped** (Renderer.kt:913-922) | yes | yes | yes | `Arrangement.spacedBy(gap, alignment)` |
| `MinWidth/MinHeight/MaxWidth/Overflow/WhiteSpace/Position/Top/Left/Right/Bottom/ZIndex/FlexWrap/AlignSelf/FlexBasis/FlexShrink/RowGap/ColumnGap/Animation/HoverStyle/FocusStyle/PseudoStates` | missing | missing | missing | missing | these exist on `Style` but no target reads them; either implement on DOM/htmlout (cheap: direct CSS) and document the native gap, or prune the ROADMAP claim (`PositionSticky, Absolute, Relative` is listed as Done) |
| `CameraView` | placeholder | placeholder | plain div (`camera.js` never instantiated) | `[Camera View]` | wire `camera.js` or mark experimental in docs |

---

## Phase 4 — Performance

None of these are urgent at current app sizes; they scale linearly with tree
size on every pass, including no-op passes.

1. **Reflection in the diff.** `reconcile/patch.go:156,172` uses
   `reflect.DeepEqual` per node for Props and the ~45-field `Style`.
   Benchmark: an unchanged 1000-node tree costs ~740 µs and ~1,900 allocs on
   top of building it (927 µs vs 185 µs build-only). `Style` is
   `==`-comparable except `HoverStyle`, `FocusStyle`, `PseudoStates`; a
   hand-written `Equal` comparing the comparable prefix with `==` and
   recursing on the three ref fields removes reflection. Props: fast-path
   `string`/`bool`/`int`/`float64` before falling back.
2. **Eager child paths.** `reconcile/patch.go:96` concatenates `childPath`
   for every child every pass (most of the 1,900 allocs). Build lazily, only
   when a patch is emitted or the child differs.
3. **Registry purge rebuilds four maps per pass.** `core/event.go:192-196`.
   With stable positional IDs survivors are ~100%; delete the few unmarked
   entries in place and `clear(r.used)`.
4. **Forms error reads are O(n²).** `forms/form.go:505-519,533-535,371-390`:
   `Error(name)` → `Errors()` → `derived()` runs every field's rule chain,
   `Validate`, and a `Values()` clone, per field, per pass; `FormField` also
   calls `form.Required(name)` which re-runs rules against `""`. Signup's 4
   fields = 16 rule-chain runs + 4 clones + ~20 lock acquisitions per
   keystroke. The `Form` is rebuilt per pass (`:167-177`), so a `sync.Once`
   memo on it is safe. Also `UseForm` (`:189-195`) allocates a five-map
   `formRecord` every pass that `NewState` discards after the first.
5. **WASM patch lookup.** `wasm/grmob-runtime.js:697` does a
   `document.querySelector('[data-node-path=...]')` per patch (O(DOM)); a
   focus dismiss with N fields costs N full scans. Keep a `Map<path, el>`
   or walk `children[i]` from the root.
6. **WASM polls `IsDirty` every frame** (`js:921-928`) even though
   `GrMobApplyPatches` is always defined by the runtime itself. Skip
   `checkLoop` when the push path is installed.
7. **`update-style` ships the whole Style** (`reconcile/patch.go:79-91`) and
   all three decoders re-parse it; `Padding`/`Margin` always serialize as
   full objects and `edgeToCSS` writes `0px 0px 0px 0px` on every styled
   node. Acceptable now; note for a future wire-format pass.
8. **Nits:** `fmt.Sprintf("%d")` in `core/input.go:78` and
   `core/layout.go:207` per render → `strconv.Itoa`. Tutorial `codeBlock`
   (`examples/tutorial/widgets.go:57-79`) re-splits and rebuilds a static
   10–40-line block per pass; it is theme-free and a textbook `core.Cached`
   candidate.

---

## Phase 5 — Quick-win features

### API
- **`Text` takes `...StyleProp`** while every other leaf takes
  `...PropsAndChildren` (`core/text.go:3`), so a tappable label needs a
  wrapping `Box`. Widening is source-compatible.
- **Missing style-prop constructors** for fields that exist on `Style`:
  `Top()`, `Position()`, `LineHeight()`, `WhiteSpace()`, `Animation()`,
  `HoverStyle()`/`FocusStyle()`, `PaddingBottom/Left/Right`,
  `MarginTop/Bottom/Left/Right/Horizontal/Vertical`.
- **`Range(lo > hi)`** is silently unsatisfiable; panic or swap at
  construction.

### Widgets (each hand-rolled ≥2 times in `examples/`)
- Labeled checkbox row (`signup/app.go:162-165`, `mobileapp/app.go:211-214`,
  `tutorial/chapter1.go:96-103`, `chapter5.go:1083-1086`)
- Stepper (`tutorial/widgets.go:162`)
- TopBar / AppBar (`tutorial/lesson_screen.go:39-50`, `chat/main.go:98-104`)
- EmptyState (`todoapp/app.go:180-186,261-265`, tutorial 2.5)
- Alert / Banner (promised by `components/variant.go:13-14`)
- Indeterminate ProgressBar / Spinner
- Select / Picker (`forms/inputs.go:21-23` says "built by hand")
- Modal facade over `core.Modal` (Tabs got one over `core.TabView`)

### Widget consistency
- `Accordion` is the only selection widget that is uncontrolled (no
  `Expanded`/`OnToggle`, `accordion.go:17-31`); it also never announces
  expanded/collapsed state and its chevron `Text` (l.55) is not
  `AccessibilityHidden`. Tests assert only the glyph.
- `Badge`, `Card`, `Tabs` lack `AccessibilityLabel`/`AccessibilityHidden`;
  `Tabs` is the only widget without a `Style` field.
- `Tabs` with `len(Content) != len(Items)` passes silently (`tabs.go:26-38`).
- `forms`: a `Validate`-only key ("form-level error") is revealed only by
  submit under `RevealOnBlur`/`RevealOnTouch` because `blurred["form"]` /
  `touched["form"]` are never set (`form.go:99-102,414-430`);
  `docs/concepts/forms.md:222-224` advertises the banner idiom without the
  caveat.

### Missing rules
`Float`/`Decimal` (`Values.Float` exists with no guard), `Min`/`Max` for
decimals, `ExactLen`, `MatchesField`/`Equals` (signup and tutorial hand-write
the same `Validate`), `URL`, `Digits`/`Phone`, `NotIn`, `When(cond, rule)`.

### Test gaps to close alongside
`chip_test.go` nil `OnTap`; `input_row_test.go` nil `OnChange`;
`rules_test.go` `Range(lo>hi)`, `MinLen` trailing whitespace, leading-`+`
integer, non-ASCII email; `form_test.go` `Validate`-only key under a per-field
reveal policy; `accordion_test.go` accessibility; `tabs_test.go` length
mismatch; plus the Phase 1 pins. Snapshot brittleness is not a concern: the
one byte-for-byte comparison (`todoapp/chip_migration_test.go:52-65`) renders
a legacy builder through the same exporter rather than pinning a literal.

---

## Phase 6 — Documentation

### 6.1 Wrong (contradicts the code)

| # | Location | Problem |
|---|---|---|
| 1 | `README.md:164-168` | `core.Default(core.Text(...))` does not compile (cannot infer T); needs `core.Default[string](...)` as `views.md:136` has it |
| 2 | `docs/tutorial-todo.md:273-289` | `core.Button(label, fn, styles...)` with `[]core.StyleProp` does not compile; Button takes `...PropsAndChildren` |
| 3 | `docs/tutorial-todo.md:5-7` and section 4 | Claims excerpts are "verbatim" from `examples/todoapp`, but shows the pre-`components` version: root is `components.Screen` not `SafeArea(Column(...))`; entry row is `components.InputRow`; `filterBar` is `SegmentedControl`; `todoRow` is `ListRow` + `Button{Variant: VariantError}`; `colorDanger` no longer exists; hairline is `components.Separator{}`; `addTodo` omits `created := …` |
| 4 | `docs/tutorial-todo.md:372-376` | "Leaf widgets take only style props … put `OnClick` on a container" — false; all inputs and Button take behavior props (`events.md:71-99` says so) |
| 5 | `docs/tutorial-todo.md:213-214` | "`Box` is the only container with no theme base" — `Scroll` and `List` have none either |
| 6 | `docs/tutorial-todo.md:598-600` | Links `docs/reconciliation.md` and `docs/ui-architecture.md`; neither exists |
| 7 | `docs/concepts/events.md:174` | Links `hooks.md`; should be `state-and-hooks.md#the-rules-of-hooks` |
| 8 | `docs/concepts/events.md:96` | `theme.Danger` — `ColorPalette` has no `Danger` (it is `Error`) |
| 9 | `docs/concepts/styling-and-theming.md:79` | `form.Submitting` — `*forms.Form` has `Submitted()` |
| 10 | `docs/platforms/native.md:52` | `SetDataDir` "Documents on iOS" — shell passes `applicationSupportDirectory` (`GomobileBridge.swift:34`) |
| 11 | `docs/components.md:577-580` | "neither renderer maps `AlignItems: stretch`" — both do (ROADMAP Done) |
| 12 | `docs/components.md:624-626` | "Filling needs a `ContentMode` prop … a two-renderer pass of its own" — `ImageWithMode` exists |
| 13 | `docs/components.md:647-661` and `components/progress_bar.go:17-28` | ProgressBar "FlexGrow silently wrong on iOS" — `GrMobFlexStack` fixed it |
| 14 | `docs/concepts/architecture.md:43`, `docs/index.md:56` | Hook list omits `UseMemo`, `UseReducer` |
| 15 | `docs/platforms/wasm.md:32-38` | `GrMobApplyPatches` "optional … pages without it keep polling" — the shipped runtime always defines it (js:917) |
| 16 | `README.md:278`, `ROADMAP.md:38` | "Patch logging and inspection on every render" — no such logging exists |
| 17 | `CHANGELOG.md:9-11,62-63` | "First tagged release", links `v0.1.0` — no tag exists yet |
| 18 | `docs/tutorial-todo.md:526` | "the build in section 6" — it is section 7 |
| 19 | `docs/components.md:92-94` | "Of the five app roots, exactly one (`fintechapp`) scrolls as a whole" — nine roots; `signup`, tutorial `home`, `lesson_screen` all `Scroll: true` |
| 20 | `wasm/main.go:33` | Comment "examples/social's root view"; import is `examples/tutorial` |
| 21 | `core/button.go:39-49` | `OnLongPress` "both native renderers already wire" — neither does (Phase 1.10) |
| 22 | `components/input_row.go:78-80` | nil `OnChange` "read-only in practice" — it panics (Phase 1.4) |

### 6.2 Missing

- **Package doc comments absent:** `core`, `reconcile`, `render`, `hooks`,
  `jsonout`, `permission`, `wasm`. Present: `components`, `forms`, `htmlout`,
  `mobile`.
- **Undocumented exports:**

  | Package | Exported | Undocumented |
  |---|---|---|
  | core | 279 | 182 |
  | components | 40 | 17 (14 `Render` methods, `VariantSuccess/Warning/Error`) |
  | render | 12 | 4 (`Manager`, `New`, `RenderInitial`, `RenderAndGetPatches`) |
  | permission | 9 | 9 |
  | wasm | 8 | 8 |
  | jsonout | 1 | 1 (`Export`) |
  | reconcile, hooks, forms, htmlout, mobile | — | 0 |

  Worst core offenders: the entire style-prop set (~60), all containers,
  `Text`, the input family, all conditionals, `Context`/`State`/`Scope`,
  `View`/`ComponentFunc`/`Keyed`, `Theme` and friends, Modal/TabView/Toast/
  Camera families, `Style`, `StyleProp`. `RenderAgain`'s comment is garbled
  (`render/manager.go:210`). The newer files (focus, navigation, cleanup,
  error_boundary) are the opposite extreme; aim for a consistent middle.
- **Never named anywhere in docs:** toasts (`ShowToast`/`Duration`/
  `UseToastStyle`, though claimed in README/CHANGELOG/ROADMAP),
  `Responsive`/`ResponsiveStyle` (ROADMAP Done), `SendSystemEvent`/
  `SetSystemEventHandler`, `ModalContent`, TabView props, camera props,
  `Display*`, `Position*`, `Overflow`, `MaxHeight`/`MinHeight`,
  `RowGap`/`ColumnGap`, `AlignSelf`, `PaddingTop/Horizontal/Vertical`,
  all easings but `EaseInOut`, the seven `Concern*` constants, `FocusRef`,
  `Guard`, `ReportConcern`, `IsDebugMode`, `WithConfig`/`Config()` ("App
  Config Injection" in README:17), `MarkDirty`/`ClearDirty`,
  `render.DispatchBoolCallback`/`DispatchIntCallback`, all 14 `htmlout`
  table helpers, `forms.Values.Clone`, `components.ColorTransparent`, the
  whole `permission` package.
- **ROADMAP Done with no doc section:** Camera (one table row),
  "Responsive layouts via style merging", hover styles (one parenthetical),
  `PositionSticky/Absolute/Relative` (no `Position()`/`Top()` prop exists —
  see Phase 5), patch logging (nothing exists — see 6.1 #16), `Modal`/
  `TabView` (one row each), App Config Injection.

### 6.3 Stale

- `README.md:260`, `docs/platforms/native.md:62`: `go install
  …gomobile@latest`, while `go.mod` pins gomobile via a `tool` block. Both
  build scripts look for `gomobile` on PATH and never run `go tool gomobile`,
  so the pin is inert for every documented workflow. Either switch the
  scripts to `go tool gomobile` or document why not.
- `CHANGELOG.md` 0.1.0 omits vs `git log`: accessibility props (1c0d1fd),
  `core.MaybeProp` (d4f90dc), `CameraView`/`TabView` (b09590d, fdbf82a),
  `Responsive`, `ImageWithMode` by name, htmlout rebuilt on `element` with
  the HTML-escaping fix (6b04a80), the `UseStyle` fix (a1acc1c), the
  `mobile/verify` source-reading tests. `[Unreleased]` compare link presumes
  the missing tag.
- `docs/concepts/events.md:57-59`, `architecture.md:131-133`,
  `reconciliation.md:69-74`: identity-based node IDs / move patches called
  "planned" but absent from ROADMAP Planned. Add them or drop the claim.
- `docs/concepts/styling-and-theming.md:136-140,188`: dated "Before
  2026-08-31" / "added on 2026-08-31" changelog prose in a reference page;
  `docs/concepts/forms.md:3` "until now nothing ever filled it". Move to
  CHANGELOG.
- `ROADMAP.md:143-146` "Join the discussion or check the GitHub repo" has no
  link.

### 6.4 Onboarding dead ends (a `go get` user cannot reach a device or a page)

1. **Native build from your own module.** `getting-started.md:101-104` says
   `android/build.sh ./counter`; both scripts `cd "$(dirname "$0")/.."` into
   the grmob checkout and `gomobile bind ./mobile "$APP_PKG"`, so the app
   package must live inside the repo. Nothing says so, and `go get` yields no
   shells or scripts. Decide the supported story (clone-and-add-your-package,
   or a template/`grmob init`) and document it.
2. **WASM for your own app** has the same problem: `wasm.md:10-12` "edit the
   import" means editing `wasm/main.go` inside the checkout.
3. **No path to a running page from `wasm.md` or `getting-started.md`.** Both
   build `-o main.wasm` into cwd; neither mentions the shipped
   `wasm/index.html`, `grmob-runtime.js`, `camera.js`, or that `index.html`
   fetches from `wasm/`. Only `README.md:210-213` and
   `tutorial-interactive.md:23-26` give the working recipe.
4. **iOS: the Xcode project is untracked** (`ios/project.yml:1-4` says so;
   needs `brew install xcodegen && cd ios && xcodegen generate`).
   `native.md:88-89` and `README.md:267` say "open the Xcode project under
   `ios/`"; `tutorial-todo.md:562` runs xcodegen without saying to install it.
5. **Android: no Gradle wrapper** (`gradlew`, `gradle/wrapper/` untracked),
   so `gradle assembleDebug` depends on a system Gradle; AGP 8.1.0 needs
   Gradle 8.0+. JDK 17 and `ANDROID_HOME` appear only in
   `tutorial-todo.md:569`, not `native.md:59-76`. Track the wrapper.
6. `android/build.sh:31` / `ios/build.sh:34` default `APP_PKG` to
   `./examples/mobileapp` while the header comments (l.6 in both) say
   `./examples/todoapp`.
7. `getting-started.md:84`: "`examples/todoapp/app_test.go` shows the
   `render.Manager` pattern including `htmlout`" — that file uses
   `mobile.RenderInitial`; the htmlout use is in `chip_migration_test.go`.
8. `wasm/wasm_exec.js` carries no note of which Go toolchain it came from;
   add a header comment with the version (must match the `main.wasm` builder).

---

## Phase 7 — Repo hygiene

- **Tag `v0.1.0`** once Phase 1 lands (`git tag -a v0.1.0 && git push --tags`);
  the CHANGELOG already links it.
- **Add CI** (`.github/workflows/ci.yml`): gofmt check, `go vet`,
  `go test -race ./...`, `GOOS=js GOARCH=wasm go build ./wasm`,
  `wasm/verify/run.sh` (needs node), `ios/verify/run.sh` on a macOS runner.
- **Root module dependency footprint:** `go.mod` requires `bytdb` (+
  `btypedb`, `serr`, `btype`) though only `examples/todoapp` imports it, so
  every consumer inherits it. Options: move examples to a nested module, or
  accept and document.
- **Delete dead code** listed in 2.6 (`examples/components.go`,
  `permission/`, `wasm/main.go` demo views, dead render APIs).
- CONTRIBUTING / SECURITY absent (low priority); CODE_OF_CONDUCT was dropped
  in 1966395.

---

## Suggested order of work

1. Phase 1 items 1.1–1.9 plus their pins (one PR each or grouped by
   package), then 1.10–1.14. Re-run the full baseline after each.
2. Phase 6.1 (wrong docs) and 6.3 CHANGELOG additions — these are cheap and
   the tag depends on the CHANGELOG being honest.
3. Tag `v0.1.0`. Add CI.
4. Phase 3 renderer parity, DOM/htmlout first (pure CSS additions), then the
   two native gaps (`Gap`+`Justify` on Compose, `Modal backdrop`).
5. Phase 2 hardening and Phase 4 items 1–4.
6. Phase 5 quick wins and Phase 6.2/6.4 (package docs, onboarding story) —
   candidates for 0.2.0.

## Verification checklist per phase

```
gofmt -l .                       # empty
go vet ./...
go test -race ./...
GOOS=js GOARCH=wasm go build -o wasm/main.wasm ./wasm
wasm/verify/run.sh
ios/verify/run.sh
go test -cover ./core/... ./render/... ./mobile/...   # expect all three to rise
```

Docs: re-extract every Go fence under `README.md` and `docs/**/*.md` into a
scratch module with `replace github.com/rohanthewiz/grmob => .` and `go vet`
it; the audit found 114 of 118 compiling, and the fix for 6.1 should take
that to 118 of 118. Consider making that extraction a test so it stays there.
