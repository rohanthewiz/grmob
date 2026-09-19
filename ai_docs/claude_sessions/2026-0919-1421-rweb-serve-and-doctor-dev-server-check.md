# The WASM server on RWeb, and doctor checks the dev server

**Session:** 88a40949-1b26-4326-b249-6fe5b0be4de5
**Date:** 2026-09-19 14:21
**Branch:** master (c6ec777 → this commit)

## The ask

1. "Serve the WASM server with RWeb instead of python."
2. "Do the recommended fixes and updates": the `grmob doctor` check offered at
   the end of step 1, plus the browser check of hot reload that step 1 skipped.
3. `/sw`.

## Finding: Python was already gone

`python3 -m http.server` survives only in old session docs. `serve/` was a Go
`net/http` server (`http.FileServer` plus a hand-rolled SSE stream in dev
mode). The job was to port `serve/` from `net/http` to RWeb. `go run ./serve`
and `go run ./serve -dev` keep their commands and flags.

## What was built

### `serve/main.go`: static files on RWeb

- RWeb v0.1.28 (`go get`; `go.mod` gains `require github.com/rohanthewiz/rweb`).
- **Routes.** `GET, HEAD` on `/`, and on `/*path` for everything else.
  - An explicit `/` route is required: RWeb's `/*path` wildcard needs at least
    one character after the slash. A probe confirmed `/` returns 404 on a
    wildcard-only server.
  - HEAD goes through `getAndHead`. RWeb drops the body of a HEAD response
    itself (Server.go, the `MethodHead` check in `writeResponse`).
    `http.FileServer` answered HEAD, and the first port returned 404 for it.
- **Why not `rweb.Server.StaticFiles`.** It refuses a prefix shorter than two
  characters (so not `/`), has no directory-index rule, and resolves
  `targetDir` as `"." + path` (an absolute `-dir` would break).
- **`staticFiles` type:**
  - Containment is done by `os.Root` (`OpenRoot`, `Root.Stat`,
    `Root.ReadFile`). It rejects `..`, absolute paths and symlinks that lead
    out of the root.
  - The wildcard parameter arrives raw, `..` included. It is
    `url.PathUnescape`d and `path.Clean`ed first, and `os.Root` is the backstop.
  - A directory is served by its `index.html`, one level only, with no
    listings.
  - `If-Modified-Since` returns 304 at one-second grain, the same as
    `http.FileServer`, and with the same blind spot for a rewrite in the same
    second. The comment first described that blind spot backwards; it was
    corrected.
  - Content types come from `rweb.FileWithModTime`, whose table maps `.wasm`
    to `application/wasm`.

### `serve/dev.go`: hot reload on RWeb

- `devServer` no longer implements `http.Handler`. `routes(s, files)`
  registers:
  - a `no-store` middleware;
  - `/__dev/events`;
  - `/__dev/client.js`;
  - `/` and `/index.html`, both with the client injected;
  - `/*path`, which goes to `staticFiles`.
- **SSE.** The hand-rolled subscriber map, flusher and ticker are replaced by
  `rweb.SSEHub`:
  - `ChannelSize` 8, `MaxDropped` 3, `HeartbeatInterval` 20 s.
  - Events go out through `BroadcastRaw` with `SSEvent{Type: name, Data:
    <json string>}`. RWeb prints `Data` with `%s`, so the wire format is
    unchanged. `Broadcast` would have wrapped every event as `message`, but the
    client listens by name.
- **Hello ordering.**
  - `subscribe()` makes the channel and queues the hello under `d.mu` before
    `hub.Register`.
  - `build()` now records its state and broadcasts under one hold of `d.mu`
    (`setStateAndBroadcast`).
  - Together these rule out a hello(old) arriving after a reload(new), which
    the client would read as a build to swap back to.
- **Known gap.** Because subscribe fills the channel itself, it cannot use
  `SSEHub.Handler`, which is the only thing that sets RWeb's private
  `sseCleanup`. A closed page's channel is freed only by eviction, a few
  minutes later. Raised as N-060.

### `grmob new` writes a tool line

- **The break.** The app never imports `serve`, so `go mod tidy` removed
  RWeb's go.sum lines and `./dev.sh` failed with "missing go.sum entry for
  module providing package github.com/rohanthewiz/rweb". This was reproduced
  in a scaffold.
- **The fix.** `cmd/grmob/new.go` runs
  `go mod edit -tool=github.com/rohanthewiz/grmob/serve` before tidy. A fresh
  scaffold's `dev.sh` builds and serves `application/wasm`, and `dev.sh`
  itself is unchanged.

### `grmob doctor`: dev server row

- `devServerCheck(root)` in `cmd/grmob/doctor.go` is appended to the Browser
  table only when `findAppRoot` succeeds.
- **The probe.** It runs `go list -deps github.com/rohanthewiz/grmob/serve` in
  the app.
  - Plain `go list` of the package passes with the sums missing, which was
    verified.
  - Checking the tool line alone would miss a tool line that was never
    followed by a tidy.
- **On failure.**
  - The detail is go's first stderr line, with the `file:line:col:` prefix and
    the `; to add:` tail stripped.
  - Go's own remedy is dropped because, under a replace directive, it names a
    pseudo-version (`v0.0.0-00010101…`) that does not exist.
  - The fix printed is
    `go mod edit -tool=github.com/rohanthewiz/grmob/serve && go mod tidy`.
- **Optional (`-`).** `./build.sh` and `grmob web` do not compile `serve`;
  only `dev.sh` does.

### Docs

`docs/platforms/wasm.md` names RWeb for `go run ./serve`, and the hot-reload
table row now says "polling stats (no fsnotify) … fanned out through an
`rweb.SSEHub`". The package comment in `serve/main.go` lists the routes.

## Verification

- **Unit tests.** `go test ./serve ./cmd/grmob` and `go vet` pass.
  - `serve/main_test.go` (new) covers:
    - content types (`/main.wasm` is `application/wasm`);
    - `/`, `/shots/` and `/shots` return the index page, and `/empty/`
      returns 404;
    - escapes (`/../`, `%2e%2e`, `..%2f`, a symlink out) never serve the
      secret;
    - 304 for an unchanged file and 200 for a stale copy.
  - `serve/dev_test.go` uses RWeb's synthetic `Server.Request`:
    - the client tag is injected on both `/` and `/index.html`, with
      `no-store`;
    - files and client.js are served;
    - hello comes first on the channel, before a broadcast;
    - after `setStateAndBroadcast`, a later hello carries the new state.
  - `TestNewScaffoldsAnAppThatBuildsAndPasses` checks `devServerCheck`: it
    passes on a fresh scaffold, and after `-droptool` and tidy it reports
    "missing go.sum entry".
- **Live runs (curl).**
  - Plain server: correct types, `/../go.mod` returns 404, 304 on
    revalidation, HEAD returns 200.
  - Dev server: `hello`, then `building` and `reload` after `touch
    wasm/main.go`.
  - A fresh `grmob new -replace` app: `dev.sh` built in 63 ms and served
    `application/wasm`.
  - `grmob doctor` in a broken app and in a fixed one showed both rows as
    expected.
- **Chrome.** The tutorial was served from `serve -dev` on :18083:
  - it booted (`#app` mounted, `GrMobWASM.Shutdown` defined);
  - the pill said "hot reload on";
  - after `touch wasm/main.go` the pill said "reloaded", and a
    `window.__swapMarker` set before the rebuild survived, so there was no page
    reload;
  - there were no console errors.
- Every server was stopped by the PID listening on its port, and the tab was
  closed.

## Files

- `serve/main.go`: RWeb server, `staticFiles`, `getAndHead`.
- `serve/dev.go`: `routes`, `newDevHub`, `subscribe`, `broadcast`, `rawEvent`,
  `setStateAndBroadcast`.
- `serve/dev_test.go` (rewritten) and `serve/main_test.go` (new).
- `cmd/grmob/new.go`: the tool directive.
- `cmd/grmob/doctor.go`: `devServerCheck`, `firstLine`, `goPosPrefix`.
- `cmd/grmob/new_test.go`: both states of the check.
- `docs/platforms/wasm.md`, `go.mod`, `go.sum`.
- `ai_docs/todo/next-list.md`: N-060.

## Notes for existing apps

An app made before this change fails `./dev.sh` with a missing go.sum entry
once it upgrades to this grmob. `grmob doctor`, run inside the app, now says
so and prints the fix:
`go mod edit -tool=github.com/rohanthewiz/grmob/serve && go mod tidy`.

## Next

Closed: None. Declined: None. Raised: N-060. Updated: None.
Full list: `ai_docs/todo/next-list.md`.
