# Session: deep links to tutorial lessons

Session: https://claude.ai/code/session_01Mv1iFwxr7pCAdxCFMKT8Pk
Date: 2026-09-03 (continues the earlier "tutorial-live-site" session doc)
Commit: e400757 "Tutorial: deep links to lessons (#2.3) and a copy-link button"
Live: https://rohanthewiz.github.io/grmob/#3.1

## Ask

Add deep links to lessons the way go-learn has them (go-learn 22adaa7:
`#<track>/<item>` parsed at boot, address bar synced via replaceState,
navigate on hashchange, a copy-link button).

## Design

The app must stay URL-agnostic (it also ships natively), so it speaks two
channels core already has and the host page translates:

```
page ──HostEvent("route", {lesson:"2.3"})──▶ app   boot + hashchange
page ◀──SendSystemEvent("route", {lesson})─── app   every navigation
```

- URL shape: `#<lessonID>`, e.g. `#2.3`. Regex on the page: `/^#(\d+\.\d+)$/`.
- No loops: the inbound hop is never echoed by Go, and the page rewrites
  the hash with `history.replaceState`, which fires no `hashchange`.
- replaceState, not pushState: the app owns its own back stack
  (‹ Contents); doubling it in browser history would cost two Backs per
  lesson.
- Natives: unknown system events fall through in `SystemEvents.swift` /
  `SystemEvents.kt`, and no native sends the host event — unaffected by
  construction.

## Go side — `examples/tutorial/deeplink.go`

- `tutorial.current core.State[string]` (new field, on the session scope
  above the Navigator): the lesson on screen or "" for the contents.
- `useDeepLinks(sctx)`: subscribes `core.OnHostEvent("route")` once per
  context tree, guarded by a hook-slot record exactly like
  `hooks.UseLifecycle`, released in `ctx.OnClose` — App runs every render
  pass, so the slot is the only per-tree memory.
- `goTo(ctx, id)` (inbound): same as current → no-op (keeps demo state);
  "" → `PopToRoot`; unknown ID → ignored; from contents → `Push`; from a
  lesson → `Replace`. Marks visited. Does NOT report back.
- `open(ctx, index, replace)` and `toContents(ctx)`: the single doors used
  by contents rows, Prev/Next, ‹ Contents, Finish. They mark visited, set
  `current`, mutate the stack, then `reportRoute(id)` →
  `core.SendSystemEvent("route", {"lesson": id})`.
- `home.go` / `lesson_screen.go` handlers now call those instead of
  `markVisited` + `Push/Replace/Pop` directly.

## Page side — `wasm/index.html`

- Wraps (does not edit) the runtime's `window.GrMobSystemEvent`: "route"
  → `showRoute(lesson)` (replaceState + copy button visibility); all else
  passes through.
- `sendRoute()` on boot (after `GrMob.mount`, only if a hash is present)
  and on `hashchange`; it also calls `showRoute` itself because the
  inbound hop is not echoed.
- Header gets `#copylink` (🔗 Copy link → ✓ Copied for 1.5 s via
  `navigator.clipboard`), hidden while on the contents.

## Tests — `examples/tutorial/deeplink_test.go`

Six tests: every door reports the expected route sequence; Finish reports
""; a route from the contents pushes (depth 2) and marks the row opened
without echoing; a route between lessons replaces (depth stays 2) and ""
lands home (depth 1); unknown IDs are ignored and re-routing to the current
lesson keeps its counter; a closed app stops listening while the next one
navigates. `newAppWithContext` helper keeps the root ctx because
`render.Manager` does not expose it and `core.StackDepth` needs one.

`chapter6_test.go`'s toast recorder now filters to `name == "toast"` —
opening the lesson also emits a route event.

## Verification

- `go test -race ./examples/tutorial`, `go vet` (incl. GOOS=js for
  ./wasm), gofmt clean, `wasm/verify/run.sh` OK.
- Locally in Chrome: boot on `#3.1`, Next → `#3.2`, `location.hash='#7.3'`
  navigates, `''` returns home, a row tap sets `#1.2`, ‹ Contents clears.
- Live after the site workflow went green: boot on `#4.2`, Next → `#4.3`,
  clear → contents with copy button hidden.

## Gotchas / notes

- The Chrome tool's ref-click on the Next button silently did nothing
  once; DOM `.click()` via javascript_tool was reliable. Also a navigate to
  an identical `#hash` URL may not reload — use a cache-busting `?v=N`.
- Known trade-off: a mistyped hash (`#9.9`) is ignored by Go but the page
  still shows the copy button, since the page trusts the hash it sends.
- Possible follow-ups: chapter-level links (`#3`), a "Copy link" affordance
  inside the app for natives via `core.OpenURL`/share sheet, and the
  mkdocs site alongside the tutorial.
