# Safe-area insets as a record, Inert reaches the CodeEditor, and the SSE channel frees itself

**Session:** b9bbe0d0-d4bf-4f17-8f18-198b1b883201
**Date:** 2026-09-19 23:03
**Branch:** master (9e36de4 → this commit)

## The ask

1. "Work what pure code tasks we can from the Next list
   `ai_docs/todo/next-list.md`" — i.e. the items that need no device, no
   person holding a phone, and no decision that is not the code's to make.
2. `/sw`.

## Method

Three items were scoped in parallel by read-only agents before a line was
written (N-057 Escape, N-059 Inert vs `canFocus`, N-026/N-027 safe-area
insets), and N-060 was read directly out of RWeb's own sources. That ordering
mattered: two of the four items turned out to have **wrong premises**, and
both would have been "fixed" incorrectly by starting from the item text.

Two decisions were the user's and were asked before implementing: the shape of
the insets API, and how far to take Escape. Both answers are recorded below.

---

## N-060 — the dev server's SSE channel frees itself (closed)

**The item said** an upstream RWeb hook was needed: `serve/dev.go`'s
`subscribe` queues the "hello" before registering, so it cannot use
`SSEHub.Handler` (the only thing that sets `sseCleanup`), and a closed page's
channel lingers for minutes.

**The finding: the constraint was self-imposed.** The requirement is only that
no build can land *between* a page registering and that page being greeted —
otherwise a client can see hello(old) after reload(new) and swap *back* to the
older build. Queueing the hello into the channel first is one way to get that.
Holding `mu` across both steps is another, and `mu` is already held across
every broadcast.

So the order is inverted: register first (through `SSEHub.Handler`, which is
what installs RWeb's on-close cleanup), then broadcast the hello.

```
before:  ch := make(chan any, 8)
         mu.Lock(); ch <- hello; hub.Register(ch); mu.Unlock()   ← no cleanup possible
         SetupSSE(ctx, ch)

after:   mu.Lock()
           hub.Handler(server)(ctx)    ← makes, registers, sets ctx.sseCleanup
           hub.BroadcastRaw(hello)     ← the new channel is registered, so it gets it
         mu.Unlock()
```

`subscribe` keeps its name and its doc, and takes a `join func()` so the
ordering property is still testable without a `rweb.Context`:

```go
func (d *devServer) subscribe(join func()) {
    d.mu.Lock()
    defer d.mu.Unlock()
    join()
    d.hub.BroadcastRaw(rawEvent(sseEvent{"hello", …}))
}
```

**Why a blind alley was avoided.** The obvious `defer d.hub.Unregister(ch)` in
`serveEvents` does not work: `SetupSSE` → `ctx.SetSSE` only *records* the
channel and returns; `sendSSE` runs later, after the handler has returned
(Server.go:1202). The defer would fire immediately. This was checked in the
module cache rather than assumed.

**Two costs, one of them gone.**

- The stdout line `"SSE Channel closed and drained"` on every eviction also
  disappears. It came from the hub's eviction closing a channel `sendSSE` was
  still selecting on; now the close happens in `sseCleanup`, *after* the loop
  has returned, so that branch is never taken.
- The hello is a broadcast, so every already-open page is greeted again
  whenever a new one connects. The hub addresses channels, not pages, and this
  one is not ours to name. Filtered on the client rather than the server:
  `devclient.js` keeps a `greeted` flag and only shows the pill for the first
  hello of a connection; a hello carrying a *different* build is still acted on
  however often it arrives, and `onerror` clears the flag so a reconnect
  re-greets.

`newDevHub`'s doc comment was rewritten: `MaxDropped` is now the backstop it
was meant to be (a live-but-not-draining socket), not the thing that
eventually reaps dead pages.

**Tests.** The two existing tests adapt to the `join` seam; one new test,
`TestSubscribeGreetsTheNewPageAndTheOpenOnes`, pins that the joining page
cannot *lose* its greeting — the tolerable half of the broadcast is that others
hear it, not that the newcomer might not.

---

## N-059 — Inert vs a control's own `canFocus` (closed; premise inverted)

**The item said** "a control that sets its own `canFocus` later in its chain
may override Inert. The read-only CodeEditor's focus gate does."

**Half of that is backwards and the other half understates it.**

Compose resolves a focus target's properties in `FocusTargetNode`
`fetchFocusProperties`, which calls `visitSelfAndAncestors(Nodes.FocusProperties,
untilType = Nodes.FocusTarget)`. Two consequences, read out of the cached
`ui-android` sources:

1. **The outermost `focusProperties` wins,** not the nearest: each visit
   overwrites `canFocus`, and the walk goes nearest-first, so the last write is
   the outermost. `RenderNode`'s head-of-chain placement was therefore already
   correct, and a control's nearer `canFocus = true` would have been
   overwritten. The stated mechanism does not exist on this version.

2. **The walk stops at the first intervening `FocusTarget`** — and that is the
   real hole. Compose's scroll container delegates a focus target of its own
   (`Scrollable.kt`: `FocusTargetModifierNode(focusability = Focusability.Never)`),
   and the CodeEditor's field sits behind two of them.

```
GrMobCodeEditor
  Row(extra: canFocus = false)        ← where RenderNode puts Inert
   └ verticalScrollWhenBounded   ── FocusTarget
      └ Box + horizontalScroll    ── FocusTarget   ← the walk from the field ends here
         └ BasicTextField          ── FocusTarget  ← never sees canFocus = false
```

**So Inert never reached the editor's field at all.** It went unnoticed because
it only bites an *editable* editor: a read-only one answers `false` to a Tab
search through its own gate whatever the ancestor says. An editable CodeEditor
inside a shut Drawer panel stayed a hardware-keyboard Tab stop.

**Fix** (`android/…/runtime/GrMobCodeEditor.kt`): the editor reads the local
itself, in its own composition — which is in scope for both cases, since
`CompositionLocalProvider` wraps `RenderNodeContent` for the inert node too,
not only for its children.

```kotlin
val inert = LocalGrMobInert.current
…
.focusProperties { canFocus = !inert && (!readOnly || gate.open) }
```

The refusal is the conjunction of the two gates: whichever says no, wins.

**Tests.** `mobile/verify/codeeditor_test.go`'s existing pin updated to the new
expression; a new `TestComposeCodeEditorHonoursAnInertAncestor` in
`inert_focus_test.go` pins both lines with the diagram and the reason, so a
future edit that drops either fails with an explanation rather than a diff.

**Still unrun on a device**, and there is still no bundled Inert subtree that
holds a CodeEditor, so nothing in the tutorial exercises it.

---

## N-026 — safe-area insets are a record (closed)

**User's decision, asked before implementing:** a field on `core.Window`, not a
separate record with its own pub/sub and `hooks.UseSafeInsets`.

The argument that carried it: `Window` is `==`-comparable *by design* — the
existing doc says so, and the record dedupes on exactly that — so a plain
four-float struct rides the existing dedupe, the existing `"window"` host
event and the existing `hooks.UseWindow` with no new machinery at all. The
alternative duplicates ~120 lines of `core/window.go` and adds a second host
event for numbers that arrive from the same measurement.

### `core/window.go`

```go
type SafeInsets struct {
    Top, Bottom, Left, Right float64
}
```

On `Window`, beside `HasFold`/`Fold`, with the comparability constraint
restated where someone would be tempted to make it a pointer.

- Physical edges, not leading/trailing — because the fold bounds beside them
  are physical, and a cutout is where it is whatever the writing direction is.
- Wire: an optional `insets: {top,bottom,left,right}` object on the same
  payload. Absent decodes to zero, so an older shell still reports a window and
  a newer core reads it correctly. The payload doc comment now says that
  explicitly, since it is the second time the property has paid off.
- `validInsets` rejects negative, NaN and infinite edges, and an invalid set is
  dropped **on its own** with the size kept — the same forward-compatibility
  stance an unknown fold gets. A negative inset is the signature of a
  conversion that divided by a zero density, which is exactly the case where
  the size is still right.

### The hosts

| host | source | trigger |
|---|---|---|
| Android | `systemBars \| displayCutout` ÷ density | fold emission **and** a layout listener |
| iOS | root `GeometryReader`'s `safeAreaInsets` | size change **and** inset change |
| Browser | — | sends no `insets` key |

**Android** (`app/AppWindow.kt`). Two things worth keeping:

- **Which insets.** `systemBars | displayCutout` is `WindowInsets.safeDrawing`
  *minus the IME*, deliberately the same set `Renderer.kt`'s `SafeArea` node
  applies, so the numbers Go reads describe the edge its own `SafeArea` keeps
  content off. The keyboard is a transient overlay with its own story
  (`core/keyboard.go`), not an edge of the window.
- **Why not `setOnApplyWindowInsetsListener`.** It *replaces* a view's inset
  handling rather than observing it, and on the decor view that is the chain
  `enableEdgeToEdge` and Compose's own `WindowInsets` depend on. An additive
  `OnGlobalLayoutListener` consumes nothing and fires on the layout pass an
  inset change causes anyway; the last reported set is remembered (a `data
  class`, so the test is a value comparison) and the frequent calls collapse to
  that comparison. The bars move without the window resizing — a rotation that
  carries the cutout to the other edge, a gesture-nav bar that changes height —
  so the fold emission alone was not enough.
- Both triggers' state (`latest`, `reported`) lives in `attach`'s closure
  rather than on the `object`, so it dies with the Activity instead of leaking
  across a recreation.

**iOS** (`App/AppWindow.swift`). The cheapest of the three: the reader already
`.ignoresSafeArea()`, so its proxy's `safeAreaInsets` *are* the window's. A
third `.onChange(of: geo.safeAreaInsets)` was added — `onChange(of: size)`
alone misses a rotation that moves the housing without resizing. The file's
existing "serialized rather than interpolated … so the shape survives the fold
key being added someday" note was updated to say what that foresight actually
bought.

**Browser.** Nothing to send, and the runtime now says so in prose rather than
being silently short: a page in a browser window has the whole viewport, and an
installed PWA's values exist only as CSS `env(safe-area-inset-*)`, with no JS
reading. See N-061.

### Tests and docs

- `core/window_test.go`: decoding, the absent key, **insets alone are a
  change** (the bars come and go without a resize), and invalid insets keeping
  the size.
- `mobile/verify/window_test.go`: the cross-host spelling contract gains an
  `insets` word list, required of the two shells with system bars and
  explicitly *not* of the browser.
- `comps/two_pane_test.go`:
  `TestTwoPaneTakesItsOriginFromTheWindowInsets` — the record's purpose made
  executable, including the off-by-a-status-bar it exists to prevent.
- `comps/two_pane.go`'s `Origin` doc and `ROADMAP.md`'s foldables entry now
  point at it; `docs/api/` regenerated.

---

## N-057 — Escape (answered, not built)

**User's decision:** record the finding, leave the feature open.

Worth recording because the item's framing was wrong. It read as an
Android-only gap ("Escape does not close an open Drawer on Android … Check
what the web and iPad do"). **Nothing closes on Escape on any host** — not
Drawer, and not Dialog, Menu, ActionSheet or Lightbox either.

- **Web.** `core.Modal` renders as a plain `div`, not `<dialog>`, so there is
  no free browser Escape. The only bare-key listener is the combobox's, which
  clears its active option and explicitly leaves Escape to the page.
- **Android.** `dispatchKeyEvent` does name the key `"Escape"`, but
  `pageGlobal` requires a modifier or an F-key, and `findKeyShortcut` only
  matches nodes carrying an `onClick` — a Drawer panel carries `onBack`. Hence
  "reaches the app unhandled".
- **iPad.** The chord gate drops every bare key too. The one asymmetry: a
  sheet-backed Modal may close for free through SwiftUI's `isPresented`
  binding, which already calls `onDismiss`. A Drawer is a ZStack layer (on
  purpose — neither a Compose Dialog window nor a SwiftUI sheet has a leading
  edge) and never can.

Both implementation shapes are written onto the item: route Escape into the
existing back claim (`onBackPressedDispatcher`, the web's
`innermostBackClaim()`) — no Go API, but it would also pop a Navigator route
and fire an AppBar back arrow — or a layer-only `core.OnEscape` beside
`OnBack`.

---

## Two scouting claims that were wrong, and were checked

Recorded because both would have produced worse code if taken on trust.

1. "No test exercises `TwoPane.Origin` with a non-zero value." False —
   `two_pane_test.go` already has *origin shifts the lead* and *hinge behind
   the origin*. The test added is a different one: `Origin` sourced from
   `Insets`.
2. "N-027 is blocked on N-026." No longer true, and arguably never was for
   4.21 specifically. The lesson sets `IgnoreHorizontalFold`, so its live axis
   is the **vertical** hinge and the missing term is `Origin.X` — the demo
   panel's own left padding, a layout constant no host reports. A number there
   would be a hard-coded guess. The item's text was corrected rather than
   closed.

## Verification

`go build ./...` and `go test ./...` green (including the regenerated
`docs/api/` staleness check and the `wasm/verify` node suite);
`./gradlew :app:compileDebugKotlin` exit 0; `gofmt` and `go vet` clean.
Nothing was run on a device or in a browser this session — every claim above
is from source, tests, or the compilers.

## Next

Closed: N-026, N-059, N-060. Declined: None. Raised: N-061.
Updated: N-027 (no longer blocked by N-026; the missing term is `Origin.X`),
N-057 (the finding, on all three hosts). Full list:
`ai_docs/todo/next-list.md`.
