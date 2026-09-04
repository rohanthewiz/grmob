# Session: the interactive tutorial as a live site

Session: https://claude.ai/code/session_01Mv1iFwxr7pCAdxCFMKT8Pk
Date: 2026-09-03
Commit: a7b1e07 "Site: publish the interactive tutorial to GitHub Pages"
Live: https://rohanthewiz.github.io/grmob/

## Ask

Make the GrMob interactive tutorial (`examples/tutorial`, the app the wasm
host mounts) into a live site experience the way `~/projs/go/go-learn` is:
a GitHub Pages deploy from a workflow, a build script, a local server, and
a host page that looks like a product rather than a bare `<div id="app">`.

## What go-learn does (the pattern copied)

- `build.sh` builds the wasm with `-trimpath -ldflags='-s -w'` and copies
  `wasm_exec.js` from GOROOT (`lib/wasm` on Go >= 1.24, `misc/wasm` before).
- `serve/main.go` is a plain `http.FileServer`.
- `.github/workflows/site.yml`: a verify job, then a deploy job that stages
  `_site/` and uses `upload-pages-artifact@v3` + `deploy-pages@v4`.
  Pages must be enabled with source = GitHub Actions (`build_type=workflow`).
- One dark/light palette on `:root` tokens; a header with title + links.

## What was added to grmob

| File | Role |
|---|---|
| `build.sh` | `wasm/main.wasm` (5.3 MB stripped) + refreshed `wasm/wasm_exec.js` |
| `serve/main.go` | `go run ./serve` hosts `wasm/` on :8080 (`-addr`, `-dir` flags) |
| `.github/workflows/site.yml` | verify (`go test ./examples/tutorial/...`, `wasm/verify/run.sh`) → build → deploy the five site files: `index.html grmob-runtime.js camera.js wasm_exec.js main.wasm` |
| `wasm/index.html` | Rewritten host page: go-learn header/palette, app in a 430×900 phone bezel on wide screens, full-bleed under 520px, loading spinner, error panel, `instantiateStreaming` with buffered fallback |
| `README.md` | Live link at the top |
| `docs/tutorial-interactive.md` | "Running it" now leads with the live URL and `./build.sh && go run ./serve` |
| `docs/platforms/wasm.md` | Documents the build route and the Scroll rule below |

GitHub Pages was enabled through the API rather than the settings UI:
`gh api -X POST repos/rohanthewiz/grmob/pages -f build_type=workflow`.
The full release gate stays in `ci.yml`; `site.yml` deliberately runs only
the two checks that can break the site.

## The one real finding: Scroll needs overflow inside a fixed box

`grmob-runtime.js` gives a `Scroll` node its flex chassis but no
`overflow`. On the old bare page the *document* scrolled, so nobody
noticed. Inside a fixed-height `.screen` the contents list (5824px tall)
was simply clipped. Fix lives in the host page, not the runtime (the
runtime is pinned by the transcript replay and the natives own scrolling
themselves):

```css
#app > * { flex: 1; min-height: 0; }
#app [data-node-type="Scroll"] { flex: 1 1 0; min-height: 0; overflow-y: auto; overscroll-behavior: contain; }
```

Any hand-rolled host that constrains the app's height needs the same rule;
it is written up in `docs/platforms/wasm.md`.

## Verification

- Local: `./build.sh`, `go run ./serve -addr :8091`, opened in Chrome.
  Contents scrolls inside the bezel; lesson 2.2's tap card logged both taps.
- Live (after the workflow's verify + deploy both went green): page mounts
  with no console errors; lesson 2.1's `+1` ×3 shows 3 (event round trip);
  lesson 3.1's `UseInterval` clock went 0 s → 3 s unattended (push channel).
- `go test ./examples/tutorial/` and `wasm/verify/run.sh` pass locally.

## Notes for next time

- Screenshots from the Chrome tool are scaled/cropped oddly; trust
  `getBoundingClientRect` / `scrollWidth` over the image when judging
  overflow. There was none.
- `wasm/main.wasm` stays gitignored; `wasm/wasm_exec.js` is tracked and was
  already identical to the toolchain's copy.
- Possible follow-ups: deploy the mkdocs site alongside (needs pip in the
  workflow, as go-learn does not do), deep links to a lesson via URL hash,
  and a visible version/commit stamp in the header.
