# N-145: precompressed wasm siblings for `serve` (plan P10d)

Session: `cb1ca8a5-fab1-4cb8-8699-9e66553a1f83`
**Date:** 2026-11-13 23:00 · **Branch:** master (7c2c560, plus this commit)

## Goal

The user pasted Next-list item N-145: nothing compressed the 13.1 MB
`main.wasm`, `serve` never set `Content-Encoding`, and nobody had checked
what GitHub Pages does for `.wasm`. The plan's fix was to precompress in
`build.sh` and have `serve` send the `.br` or `.gz` copy when the request
accepts it.

## What GitHub Pages does (checked first)

I fetched the live `https://rohanthewiz.github.io/grmob/main.wasm` with curl
under four `Accept-Encoding` values:

| Accept-Encoding | Content-Encoding | Bytes |
|---|---|---|
| `gzip, deflate, br` | gzip | 3,131,849 |
| `gzip` | gzip | 3,131,849 |
| `br` | (none) | 13,176,265 |
| `identity` | (none) | 13,176,265 |

Pages gzips `application/wasm` itself, sends `Vary: Accept-Encoding`, and
never offers brotli. It has no sibling negotiation, so a `.br` file would
only be reachable under its own URL. **The deployed site needed no change.**
`site.yml` now sets `GRMOB_PRECOMPRESS=0` on its build step, and its stage
step copies files by name, so siblings were never going to be published.

## Measurements that set the levels

On the 13.1 MB module (Apple gzip, Homebrew brotli):

| Command | Size | Time |
|---|---|---|
| gzip -6 | 3.05 MB | 0.29 s |
| gzip -9 | 3.00 MB | 1.03 s |
| brotli -q 5 | 2.48 MB | 0.20 s |
| brotli -q 9 | 2.38 MB | 0.85 s |
| brotli -q 11 | 2.09 MB | 19.5 s |

I chose brotli q9: q11 would turn a 2 s build into a 22 s one. Run side by
side, gzip -9 and brotli q9 take a cached `./build.sh` from 0.2 s to 1.3 s.

## The change

- **`build.sh`** deletes the old `.gz`, `.br` and `.tmp` siblings of
  `main.wasm`, `wasm_exec.js`, `grmob-runtime.js` and `index.html`. Unless
  `GRMOB_PRECOMPRESS=0` is set, it then writes `gzip -9 -n` (always) and
  `brotli -q 9` (only when the CLI is on PATH).
  - The two codecs run in parallel. Each file is written to a `.tmp` and
    renamed, so `serve` never reads half a file.
  - `wait PID` returns each job's status, and `set -e` turns a failure into
    a failed build.
  - `camera.js` (1.7 KB) is left out.
- **`serve/encoding.go`** (new):
  - `pickSibling` tries br first, then gzip. It skips a sibling that is not
    a regular file or is older than its original; `index.html` and
    `grmob-runtime.js` are edited by hand, so their copies can be stale. A
    missing `Accept-Encoding` header gets the plain file.
  - `acceptsEncoding` follows RFC 9110. A named `q=0` is a final refusal,
    even under `*`. `*` covers codings that are not named, `x-gzip` means
    gzip, and a malformed weight counts as not accepted.
- **`serve/main.go` `serveFile`**:
  - Sends `Vary: Accept-Encoding` on every file response, 304s included.
    Siblings come and go between builds, so the header can't depend on
    whether a sibling exists now.
  - Serves the chosen sibling through `rweb.FileWithModTime` under the
    original's name and time, so `Content-Type` stays `application/wasm` and
    `Last-Modified` matches the `If-Modified-Since` check. It also sets
    `Content-Encoding`.
  - Falls back to the original if reading the sibling fails.
- **`serve/dev.go`** runs `build.sh` with `GRMOB_PRECOMPRESS=0`. Dev serves
  localhost with `no-store`, so compression would only slow each rebuild.
  Because `build.sh` still deletes the old siblings, a plain build's copies
  can't outlive the module they were made from.
- **`.gitignore`**: `wasm/*.gz`, `wasm/*.br`, `wasm/*.tmp`.
- **Docs**: `docs/platforms/wasm.md` has a paragraph on the copies (what
  `serve` sends, what Pages does, dev skips them).
  `docs/tutorial-interactive.md` has its two command comments updated.
  Plan row P10d is marked landed.
- **`wasm/verify`**: the census of Go files (tracked plus untracked) went
  from 767 to 768 because of `serve/encoding.go`.
  `TestTheQuestionsOnTheSharedRepositoryParseAreTheOnesDecidedOn` asked for
  the five quoted figures in `repowalks_test.go` and `timings_test.go` to be
  updated, and they were.

## Verification

- **Tests:** two new tests in `serve/main_test.go`.
  - `TestStaticPrecompressedSiblings` covers 11 `Accept-Encoding` cases. In
    each it checks the body, `Content-Encoding`, `Content-Type`,
    `Last-Modified` and `Vary`, and it also checks a 304 with `Vary`.
  - `TestStaticStaleSiblingSkipped` covers a stale `.br` falling through to
    a fresh `.gz`, and a directory named like a sibling.
- **Mutation checks:** I removed each check in turn: the age check, the q=0
  refusal, `q > 0` changed to `q >= 0`, and the `Vary` header. Each removal
  failed a test. Removing the `IsRegular` check did not, because the
  read-failure fallback already covers a directory; it stays as the cheaper
  guard.
- **curl** against the real `serve`: br gave 2,385,160 bytes, gzip
  3,001,805, identity 13,176,272. `index.html` and `wasm_exec.js` were
  compressed too. Both encodings decoded to `main.wasm`'s exact bytes.
- **Headless Chrome 157 over CDP** (a scratch script, not committed) booted
  the tutorial from `serve`, and the page rendered its lessons list.
  - `localhost` and `127.0.0.1`: Chrome sent `gzip, deflate, br, zstd` and
    got br.
  - The LAN address over plain http: Chrome sent `gzip, deflate` and got
    gzip. A phone on the LAN gets gzip, which is why gzip is the copy
    `build.sh` always writes.
- **`serve -dev`**: the startup build left no siblings, and `main.wasm`
  went out plain with `Cache-Control: no-store`.
- **Packages:** `go vet ./serve` is clean, and `go test` passes for
  `./internal/... ./wasm/verify/... ./cmd/... ./serve/...`.

## Mistake worth remembering

The first curl check showed no compression. The loop passed the header as
`${enc:+-H "Accept-Encoding: $enc"}`. zsh doesn't split unquoted
expansions, so curl got one word, `-H Accept-Encoding: …`, and the header
never arrived. Explicit arguments in a function fixed it. The code was right
all along; the harness was wrong.

## Not done

- The `grmob new` scaffold's `build.sh` makes no siblings. That belongs
  with N-142's `grmob web -dist` (C7), which can use the same `name.br` and
  `name.gz` naming, since `serve` already understands it.

## Next

Closed: N-145. Declined: None. Raised: None. Deferred: None.
Promoted: None. Moved: None. Updated: None.
Full list: `ai_docs/todo/next-list.md`.
