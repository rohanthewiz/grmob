# Next list review: N-109 passes locally, the site is stale, N-110 re-rated

Session: `62b8dd94-da05-4ba9-83bb-95e2999dd799`
**Date:** 2026-10-11 23:05 · **Branch:** master (9857cb3, plus this doc)

## Goal

`/next-list` (living-list mode), straight after tagging v0.6.0 on red CI
(`2026-1011-2237-v0.6.0-release-on-red-ci`).

## L2/L3: dating and lapse checks, all clean

- 307 session docs, so N = 307. Every `raised` stem exists. The oldest
  unfinished item is N-002 (age 94); the newest are N-109 and N-110 (age 0).
- `git log -p --follow` on the file: 109 IDs have ever been items, all 109
  are present now, and there are no duplicates. **Next ID** N-111 is above
  the highest. The one gap, N-105, was never assigned, on purpose: the
  `2026-1010-2139` doc says so.
- The transition check found nothing: all of the last 15 docs' `## Next`
  sections use the `Closed: … Raised: …` summary form.
- The summary cross-check found nothing: every reported closure (N-021,
  N-058, N-089 to N-096, N-100 to N-108) is in Closed. Every
  `Moved → Validate` ID is in Validate, except N-090 and N-091, which later
  docs closed.

## L4: premises against the code

Every Open and Validate premise held:

- N-009: no AlarmKit or full-screen intent anywhere.
- N-013: `SetTimeZone` is still called once, from each `GomobileBridge`.
- N-015: Compose reads only `shrinkPinned`, with no proportional shrink.
- N-024: `barValueRoom` is still the 402pt guess.
- N-027: 4.21 still sets `IgnoreHorizontalFold` with no Origin.
- N-047: the BOM is still `2024.12.01`.
- N-098: `comps/disclosure.go` still draws "▸"/"▾" as Text.
- N-099: no Inert read in the Swift runtime.
- N-040: `primeKeyboard` is still in use.
- N-043: `input.replace` with no `inputDelegate`.
- N-044: only the iOS 26.5 runtime is installed.
- N-049: only an `android-36.1` system image is installed.
- N-004: `testEscapeClosesTheDrawer` exists.
- N-028: the rotation test exists.
- N-097: no keystore commits since 2026-10-10.

No item moved between Open and Validate.

### N-109: check 24 passes locally, and the site has not deployed

- `./wasm/verify/run.sh` ran locally on macOS with Chrome 152 and **passed**,
  browser check 24 included ("OK: roving tabindex, disabled focus, ArrowDown
  and the toolbar walk", exit 0). The overflow CI reports (tutorial lessons
  2.2, 4.29, 5.3, 6.5, 7.2 and 7.4 at 360/390px) therefore comes from CI's
  ubuntu-latest runner, its fonts or its Chrome version. To reproduce it,
  run the suite on Linux.
- `.github/workflows/site.yml`: `deploy` has `needs: verify`. In the latest
  failing run (38194498560), verify failed and deploy was skipped. **The
  published tutorial site has not been deployed since the last green run,
  2026-09-24.**
- Re-rated high from medium.

### N-110: re-rated high

v0.6.0 was tagged around it, and while `go test -race` fails, CI skips the
docs-server build, the WASM build and `wasm/verify`. A new failure in any of
those goes unseen. It still reproduces locally.

### Also

Removed a stray blank line left before N-109 when it was added.

## Waiting on the user

Five low-value, 46–69-doc-old items look like Non-goals: N-013, N-015 and
N-027 (each with a 2026-09-29 recommendation to decline, not acted on),
N-045 (by design) and N-051 (moot). None was moved, because declining is the
user's call. The two items keeping master red (N-109, N-110) are the
highest-value work; N-110 is a small test-only fix.

## Next

Closed: None. Declined: None. Raised: None. Deferred: None. Promoted: None.
Moved: None. Updated: N-109, N-110. Full list: `ai_docs/todo/next-list.md`.
