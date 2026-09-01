# Interactive Tutorial — Phase 6: Chapter 6 (Navigation & Overlays)

Session: https://claude.ai/code/session_014sBCzrGB5gk88FhYPkSszw

## Goal

Phase 6 of the eight-phase interactive tutorial: add Chapter 6 "Navigation &
Overlays" to `examples/tutorial` (now 30 lessons). Committed as 7c907cf.

## The scope discovery

The phase plan predicted "framework change = one line" (append `chapter6()`),
but three gaps falsified that, and the tutorial's charter — every demo is
real — made closing them part of the phase:

- `core.Modal` had **no way to receive children**: `ModalNode.Content` had no
  setter among the ModalProps (Visible/OnDismiss/Backdrop were all of them).
- `core.SendSystemEvent` was an **empty stub**, so `ShowToast` reached nobody
  anywhere — browser, native, or test.
- The **wasm runtime rendered neither**: "Modal" fell through to a plain div
  (children always visible, `visible` prop ignored), and no toast layer
  existed.

## Framework changes (all additive)

- **core/modal.go** — `ModalContent(children ...View) ModalProp` (appends;
  `Content` name was taken by tabview). Doc comment carries the doctrine:
  content renders every pass regardless of Visible — a modal hides, it does
  not unmount — so toggling Visible is a prop patch and content state
  survives a close; reset it in OnDismiss when it shouldn't.
- **core/sys_events.go** — `SetSystemEventHandler(fn)` + RWMutex-guarded
  dispatch in `SendSystemEvent`. Package-level *by design*, unlike the
  registry/nav-stack migrations to the context tree: ShowToast takes no
  Context (fire-and-forget from any goroutine), and the resource reached is
  the process's one screen-overlay layer. Hosts register once at startup;
  nil detaches (correct for headless); tests register a recorder and restore
  nil in cleanup.
- **wasm/grmob-runtime.js** — Modal: fixed-overlay chassis assigned at
  createElement (position fixed, flex column centered, display none, z 1000);
  `visible` → display flex/none, `backdrop` → background, `onDismiss` →
  `attachModalDismiss` (click listener with `e.target !== el` guard so
  content clicks never dismiss; latest-callback-ID dataset discipline; void
  payload `{}` like focus/blur). All three handled in BOTH createElement and
  the update-props patch path, and *before* the generic `on*` branch — the
  generic branch would attach a never-firing "dismiss" DOM listener AND mark
  the listener slot taken. Toast: lazy `toastLayer` (fixed bottom-center,
  pointer-events none, z 2000 — above Modal so a toast confirming a modal's
  action isn't drawn behind it); per-toast element, default dark look,
  `payload.style` mapped through the existing `styleFromGrMob`; double-rAF
  fade-in; removed after duration + fade. Page-level `GrMobSystemEvent(name,
  payloadJSON)` dispatches "toast" → `GrMob.showToast`; unknown names drop.
- **wasm/main.go** — `registerSystemEvents()`: installs the core handler only
  when the page defines `GrMobSystemEvent` (same feature-check pattern as
  GrMobApplyPatches); payload crosses as JSON text because js.ValueOf can't
  marshal nested maps or *core.Style.

## What was built (tutorial)

- **chapter6.go** — five lessons. Through-line: ownership of state — a frame
  owns its hooks and takes them to the grave; a Modal owns nothing and merely
  hides; a Toast isn't in the tree at all. The nav demos drive the app's
  REAL Navigator: pushed demo screens are genuine frames (scaffolded by
  `navDemoScreen` — NAV DEMO badge + live `stackCaption` reading
  StackDepth/CanPop per pass; every pushed screen must offer its own Pop,
  since ‹ Contents lives on the frame underneath).
  - **6.1 Screens are a stack** — Push/Pop; the tutorial itself as worked
    example (contents = root, opening the lesson was a Push). Demo: lesson
    counter + pushed `detailScreen` with its own slot-0 counter — bump, push,
    pop: lesson count intact; re-push: detail count fresh.
  - **6.2 Replace: steps with no way back** — checkout step 1 (a one-field
    UseForm, chosen so a whole form record visibly dies with the frame) →
    "Place the order" Replaces to confirmation at the SAME depth; Pop lands
    on the lesson, never step 1; re-enter → empty note. Prev/Next's Replace
    quoted as the worked example.
  - **6.3 Unwinding: PopToRoot vs Reset** — `drillRoute(level)`: parameters
    are closure captures, a screen pushes deeper copies of itself. "Done —
    back to contents" is a live PopToRoot past the lesson to the root frame,
    progress intact (it lives above the Navigator — app.go's session scope
    quoted in the code block). Reset taught in prose: NO live demo, and not
    as a compromise — the docs' own line is that the two are identical on
    screen; only the root frame's state could tell them apart. (Also: a live
    Reset can't be built from a lesson — routes get frame ctx, scopes are
    per-context maps, `Context.parent` is unexported, so `t.Home` is
    unreachable from chapter code. Deliberated and dropped.)
  - **6.4 Modal: the overlay that hides** — controlled like an Input:
    Visible renders state, OnDismiss reports intent; stack untouched (depth
    caption still 2 while open). Confirm-order dialog: gift-receipt checkRow
    whose state survives close/reopen — the stated opposite of 6.1's popped
    frame. Cancel/Confirm/backdrop all close via app state.
  - **6.5 Toast: fire and forget** — three buttons (default 2000ms, 
    Duration(5000), UseToastStyle green); lesson counts sends so the tree has
    something to see. Doctrine: handlers/effects only (render runs any
    number of times), confirmation-only (needs a button → it's a modal),
    never gate on the user having seen it.
- **chapter6_test.go** — five tests + `modalNode` helper. 6.4 exercises the
  chapter's unique dispatch path: `onDismiss` void callback on a non-Button
  node via `mgr.DispatchCallback`; asserts content in tree while
  `visible == false`, checkbox `checked` persists across close/reopen. 6.5
  registers a recorder handler (cleanup restores nil — it's process-wide),
  asserts zero events from rendering alone, then message/duration/style per
  tap. Depth assertions ride the `StackDepth N` captions.
- **lesson.go** — `chapter6(),` appended to `Chapters`.

## Facts learned/confirmed this phase

- `navigatorState` is per context tree and shared by ALL derived contexts —
  a demo can't embed a second independent Navigator; nesting one would
  recurse on the same stack. Demos must (and profitably do) drive the app's
  real stack.
- Scopes (`ctx.Scope`) are per-context maps, NOT tree-wide: a frame calling
  `ctx.Scope("tutorial-session")` gets a fresh scope, not App's. Session
  state above the Navigator is reachable only by closure capture.
- The reconciler delivers Modal open/close as an update-props patch (content
  stays mounted); the JS patch path previously dropped unknown prop keys
  silently, which is why visible/backdrop/onDismiss needed explicit cases.
- `components.Chip`-style tap() reaches every demo control; a pushed screen
  and the lesson can reuse a label ("+1") since only the top frame renders.
- Toast payload reaches the test recorder as native Go values (`duration`
  int, `style` *core.Style) — no JSON round trip in-process.

## Verification

- `go test ./...` fully green (first run); gofmt and vet clean (todoapp/
  store.go was already unformatted pre-session — untouched).
- `go test -race -count=2 ./examples/tutorial/` clean.
- `GOOS=js GOARCH=wasm go build ./wasm` compiles; `node --check` passes on
  grmob-runtime.js.
- Browser check attempted (served scratchpad copy on 127.0.0.1:8478; curl
  200) but the claude-in-chrome extension still shows an error page for
  localhost — the same site-permission blocker as Phase 1. The new Modal/
  toast JS therefore still needs a human eyeball:
  `GOOS=js GOARCH=wasm go build -o wasm/main.wasm ./wasm && (cd wasm && python3 -m http.server 8080)`.

## Next session: Phase 7

Per the phase list (Phase 1 doc, 2026-0831-2047): Ch.7 "Theming & Styling" —
live Default↔Material switcher, style merging, transitions. Sources of
truth: docs/concepts/styling-and-theming.md, core/theme.go, core/style.go
(style_merge_test.go). Watch for: theme is carried on the Context
(`WithTheme` copies share scopes), so a live switcher likely needs the theme
as state feeding a re-wrapped context — verify how examples handle it before
assuming the framework change is zero.
