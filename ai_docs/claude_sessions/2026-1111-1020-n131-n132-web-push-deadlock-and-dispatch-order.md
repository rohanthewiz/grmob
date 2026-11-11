# N-131 and N-132: the web's push deadlock, and a tap's patches arriving late

Session: `2ece2445-d8d5-41c3-8952-63656028d745`
**Date:** 2026-11-11 10:20 · **Branch:** master (076844b, then f1322ed and b6d84eb, plus this commit)

## Goal

`/work-next-list N-131 N-132 N-133`: the first three reliability items the
plan of 2026-11-11 raised (plan R1, R2, R3+R4). All three were doable without
the user. This doc covers the first two; N-133 follows in its own wrap.

## N-131: toolbar commands killed the Go runtime on the web

**Reproduced first.** A scratch CDP probe (a HEAD worktree built for js/wasm,
the live page's GoInvokeCallback, the tutorial routed to 4.13) pressed
Indent: the console printed `fatal error: all goroutines are asleep -
deadlock!` and `exit code: 2`, and every later call threw "Go program has
already exited". 4.14 was never reached on HEAD.

**The mechanism.** On js/wasm a call from JS into Go (`ReceiveEvent`) runs
every goroutine that becomes runnable before it returns, so the handler's
`State.Set` let the pump render and push inside that call. The pump holds
`Manager.mu` across `ApplyPatches`, the page applied the batch synchronously,
the batch's editor command rewrote the buffer and sent the edit back to Go
(`emitCodeChange` → `GoInvokeCallback`), and the nested `RenderAgain` locked
the same mutex on the same thread.

**Layer 1 (JS).** `dispatchToGo(id, payload)` in grmob-runtime.js; all 23
direct `GoInvokeCallback` sites go through it. It queues while
`applyingTree > 0` or `goPushDepth > 0` (the new `GrMob.applyPushed`, which
`GrMobApplyPatches` now calls, counts Go's push). `withTreeApplied`'s
outermost exit flushes the queue: at once when no push is on the stack (the
page path, where RenderAgain has already released the lock), in a microtask
(`Promise.resolve().then`, as the keystore does) when it is. The synchronous
flush keeps the dom.mjs suites' "patch() returned, so Go has heard it"
assumption true; with a microtask-only rule the existing selectAll test
failed. A flush is re-entrancy-guarded and one throwing call does not strand
the rest. `dispatchFromElement` keeps dropping surgery-caused events.

**Layer 2 (Go).** `webhost.PushGuard` (pushguard.go, no build tag): the
listener delivers under `Deliver`; `renderAgain` returns `[]` while
`Delivering()`; `receiveEvent` calls `Defer`, which logs loudly and runs the
event after the delivery on a fresh goroutine (inline would still be inside
the pump's lock). All three browser hosts use it; the hand-written two import
it rather than copy it. In the live build with the JS queue removed, the
presses held and the log named each deferred call.

**R9a.** `GrMob.watchExit(exited)` draws a fixed `role=alert` banner ("The app
stopped. Reload the page to start it again." + Reload, white on #8a1c1c,
9.3:1) when go.run's promise settles, unless the wrapped
`GrMobWASM.Shutdown` was called (the hot-reload client). Wired into
wasm/index.html and the `grmob new` template. dom.mjs refused the `font`
shorthand (cssstyle.mjs models CSS-wide keywords only), so the banner uses
longhands; it uses a flag, not getElementById, to stay inside the DOM members
dom.mjs models. Checked in a headless screenshot.

**Pins.** codeeditor_test.mjs: a pushed command reaches Go only after the
push; a page-path one at the batch's end with the whole batch applied; a
throwing call does not strand the queue; a census of direct
`window.GoInvokeCallback(` calls (exactly two). The first three fail with the
queue disabled. stopped_test.mjs (3 tests), pushguard_test.go (4),
`TestEveryBrowserHostDeliversUnderThePushGuard` (AST census of the three
hosts). Browser check 25: on the site page, Indent on 4.13 (witness: Go's
selection line "bytes 0–N selected … from line 1"), Bold over 4.14's first
heading (witness: Go's Markdown reads `## **A note**`, after one more event,
"Show the Markdown"), and no banner. The check tally regex in
checknumbering_test.go now takes any number of one-off kinds, and knows
"twenty-five".

## N-132: a tap's patches could arrive after a later push

`Dispatch*` returned its patches, and the lock was released before the host
queued them, so a pump pass in that gap could push the next pass first.
`render.Manager.deliverLocked`: with a listener attached, the pass goes to the
listener inside the lock (through `applyLocked`, the pump's guarded call, now
shared) and the dispatch returns `[]`. The listener is read once so a racing
`SetListener` cannot drop a pass. No host change; the native runtimes'
comments, mobile/bridge.go, docs/platforms/native.md and the regenerated
API pages say the listener is now also called inside a `Trigger*` call.

Pins in render/dispatch_order_test.go: a forced gap read `[2 1]` before and
`[1 2]` after; 2,000 taps whose handlers also write from a goroutine (writes
numbered under a mutex so the state itself only rises) failed before at
arrival 443 of 2,039. `TestConcurrentDispatchAndTimerPushesDoNotRace` hung
with the change: its 16-slot chan listener filled inside the lock while the
test read only after `wg.Wait`. It now drains while the dispatchers run.

## Verification

- `go vet ./...`, `gofmt -l .` clean; `go test ./...` green; `go test -race`
  over the whole module green.
- `GOOS=js GOARCH=wasm` build and vet of `./wasm`, the shots host, webhost.
- `wasm/verify/run.sh` green, including the browser pass (check 25 and the
  261-view overflow sweep).
- `ios/verify/run.sh` green after N-132 (9 patch batches, tree matches Go).
- The tracked-Go-file census moved twice (737 → 739 → 740) with the new
  files; the five sentences that quote it were updated each time.

## Lessons

- On js/wasm, a JS→Go call runs every runnable goroutine before it returns,
  so the pump's push happens inside `ReceiveEvent`, not after it. Any JS that
  calls back into Go from a batch is inside the render lock.
- The repository's file-count census counts untracked Go files too; park a
  next item's new test file outside the tree when verifying a commit that
  does not include it.

## Next

Closed: N-131, N-132. Declined: None. Raised: None.
Deferred: None. Promoted: None. Moved: None.
Updated: None. Full list: `ai_docs/todo/next-list.md`.
