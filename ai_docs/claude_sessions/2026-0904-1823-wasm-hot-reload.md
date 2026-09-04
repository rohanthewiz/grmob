# Session: hot reload for the WASM target

Session: https://claude.ai/code/session_01J7qf4UetyH8cQ3D7Egn9dC
Date: 2026-09-04 (follows "tabpanel-aria-wiring")

## Ask

"Explore hot-reload mechanism even if it's only to WASM."

ROADMAP had "Hot module replacement" unchecked under DevTools. The exploration
became an implementation for the one target where it is possible, plus a
written account of why it is only that target and what a fuller version
would need.

---

## What existed before

- `./build.sh` writes `wasm/main.wasm` (gitignored) with the site's recipe;
  `go run ./serve` is a plain `http.FileServer` over `wasm/`.
- `wasm/main.go`'s `renderInitial` already closed a previous manager on
  re-entry, and `render.Manager.Close`'s doc named "hot-reload hosts" as a
  reason it exists. The Go side was half-ready.
- `wasm/index.html` booted once, inline: `new Go()`, fetch, instantiate,
  `RenderInitial`, `GrMob.mount`, then the deep-link route replay.
- An incremental js/wasm build of `./wasm` takes ~0.3–0.7 s; a no-op ~0.13 s.
  Cheap enough that a polling watcher plus a full rebuild is the whole
  mechanism — no incremental linking, no module splitting.

## The design

```
editor saves a .go file
   │
   ▼  (poll, 250 ms; file set = `go list -deps ./wasm`, re-read after each build)
serve -dev ──▶ sh build.sh ──▶ wasm/main.wasm
   │                 │
   │                 └─ compile error ──▶ SSE "buildfail" ──▶ overlay on the page
   ▼
SSE "reload" ──▶ page: GrMobWASM.Shutdown()   stop the old module, let it exit
                       GrMobHost.boot()       fetch + instantiate the new one
                       route + scroll replay  same lesson, same place
```

Four pieces:

| Piece | File | Role |
|---|---|---|
| `GrMobWASM.Shutdown` | `wasm/main.go` | closes the manager (context tree, every ticker on it), then closes the package-level `done` channel so `main` returns → `wasmExit` → the old instance is collectable |
| `GrMobHost.boot` | `wasm/index.html` | the page's boot as a re-callable function; resolves to `{go, exited}` (`exited` is `go.run`'s promise, which settles on exit) |
| watcher + build + SSE | `serve/dev.go` | zero deps: stat polling, `sh build.sh`, `text/event-stream`; injects the client at `</body>` with the current build id on the tag |
| the client | `serve/devclient.js` | `//go:embed`ed; serialized swaps on a promise chain; status pill; error overlay; scroll save/restore by `data-node-path` |

### Decisions and why

- **Run `build.sh`, not a private `go build`.** One recipe; what hot reload
  shows is byte-for-byte what deploys. `-s -w` does not lose Go tracebacks
  (pclntab survives), so the dev build needs nothing different.
- **Poll, don't fsnotify.** Keeps the module's non-Go dependency count at
  zero. Stat of a few hundred files every 250 ms is not measurable.
- **Watch the build graph, not the repo.** `go list -e -deps` for js/wasm,
  filtered to in-module dirs, non-test `.go` files, plus `go.mod`/`go.sum`/
  `build.sh`. `-e` keeps the set intact while a file is mid-edit and does not
  parse. Re-stamped after each build so a new import is watched from then on.
- **Two file classes, two responses.** Go files → rebuild + swap. Host files
  (`index.html`, `*.js` in `wasm/`) → plain `location.reload()`, since the
  runtime cannot be swapped under a mounted tree. `wasm_exec.js` and
  `main.wasm` are excluded from the host set because `build.sh` rewrites
  `wasm_exec.js` on every build — without the exclusion every build would
  page-reload the page it had just hot-swapped.
- **Baseline stamps are taken before the build starts.** An edit that lands
  while the compiler runs shows as a difference on the next tick and gets its
  own build, instead of being folded silently into one that never saw it.
- **SSE, not WebSocket.** One-way traffic, `EventSource` reconnects on its
  own, no library either side. The opening `hello` carries the current build
  id and any standing compile error so a late or reconnecting page catches
  up; the client compares it to the id stamped on its script tag.
- **Exit the old module, don't just overwrite the global.** Overwriting
  `GrMobWASM` would leave the old Go runtime parked in memory forever, ~20 MB
  per reload. Returning from `main` lets `wasm_exec.js` drop the instance.
- **The stale scheduler timeout.** `wasm_exec.js` leaves the runtime's
  pending wake-up armed after exit; when it fires, `_resume` throws "Go
  program has already exited" into the console. The client clears
  `go._scheduledTimeouts` after awaiting `exited`. Private field, guarded — a
  rename costs one console line per reload, nothing else.
- **`Shutdown` is synchronous in practice but not by contract.** On js/wasm
  goroutines are scheduled cooperatively inside the `resume()` export, so
  `main` has returned before the call comes back; the client still awaits
  the promise (bounded at 2 s, fallback = page reload).

### What survives a swap

Decided by where the state lives. Go-side state — `NewState` slots, the
navigation stack, a half-typed input — is heap memory of the discarded
instance and cannot cross to a new one. What survives has a representation
*outside* the module:

- the lesson: the tutorial already reports it as a `"route"` system event and
  accepts it back as a host event (`examples/tutorial/deeplink.go`); `boot()`
  replays the hash;
- scroll offsets: read off `Scroll` nodes by path before the swap, written
  back two frames after the mount.

An app wanting more has exactly that tool: report it, accept it.

### Why hook slots are not replayed

The obvious next step — snapshot every context's slots to JSON, re-seed
positionally — is what React Fast Refresh and Flutter rest on, and is
deliberately not done. Slots are positional in call order; the edit that
prompted the reload is precisely what reorders/adds/retypes them, and a
stale value in the wrong slot is the failure class debug mode's cursor-drift
check exists to catch (an `interface conversion` panic, or a plausible wrong
value). Flutter keeps the heap and patches code; React re-runs hooks against
a preserved fiber and resets on signature change. Neither holds for a fresh
WASM instance. A safe version needs: a serializable form per hook kind, a
typed guard on restore, a reset on any shape mismatch. That is a design of
its own; noted in `docs/platforms/wasm.md`.

### Why only WASM

The natives are a `gomobile bind` product linked into a host app. Swapping
Go code in a running process means a second Go runtime (c-shared cannot be
unloaded; two runtimes in one process is unsupported), and iOS forbids
loading code outside the simulator. The native loop is rebuild-and-relaunch;
the framework's "see it now" is this target, on the same `render.Manager`.

## Verified in Chrome

1. Edit lesson 2.3's summary → page stayed on `#2.3`, `scrollTop` 400
   restored, new text on screen. Console: `GrMob WASM stopped.` then
   `GrMob WASM ready.`, no errors.
2. Introduce `undefinedThing` → overlay with the compiler output over the
   still-running app; `git checkout` the file → overlay cleared, swap done.
3. Lesson 3.1 (UseInterval clock): after a swap, `GrMobApplyPatches` was
   called exactly 3 times in 3 s — one ticker, the old one dead.
4. `touch wasm/camera.js` → log `changed: camera.js (page reload)`.
5. `go test ./...` and `sh wasm/verify/run.sh` pass; three new tests in
   `serve/dev_test.go` (injection before `</body>` + `no-store`, the `hello`
   payload, `diffStamps`).

## Files

- `wasm/main.go` — `Shutdown`, package-level `done`/`doneOnce`, `main` parks
  on `<-done`.
- `wasm/index.html` — `boot()` + `window.GrMobHost = { boot, running }`;
  boot element renamed `bootEl` to free the name.
- `serve/main.go` — `-dev` flag; `serve/dev.go`, `serve/devclient.js`,
  `serve/dev_test.go` new.
- `docs/platforms/wasm.md` — `Shutdown` row in the contract table; new
  "Hot reload" section. `docs/tutorial-interactive.md` — the `-dev` command.
  `ROADMAP.md` — HMR checked, WASM-only.

## Still outstanding / ideas

- The SSE stream is a natural transport for the other DevTools items
  (patch logging, a state inspector overlay) — same injected client, more
  event names.
- The mermaid diagram in `wasm.md`'s contract section still lists four
  functions in the `GrMobWASM` box; `Shutdown` is in the table only.
- Positional slot replay, if ever wanted, per the sketch above.
