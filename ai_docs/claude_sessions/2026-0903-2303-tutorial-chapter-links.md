# Session: chapter-level deep links

Session: https://claude.ai/code/session_01Mv1iFwxr7pCAdxCFMKT8Pk
Date: 2026-09-03 (follows "tutorial-deep-links" in this directory)
Commit: b909809 "Tutorial: chapter-level deep links (#3 opens where chapter 3 starts)"
Live: https://rohanthewiz.github.io/grmob/#3

## Ask

Add chapter-level links like `#3` to the lesson deep links added earlier.

## Decision

`#3` opens chapter 3's first lesson. The tutorial has no chapter screen and
the framework exposes no scroll-to control, so "a link to a chapter is a
link to where reading it starts" is the only landing that exists. The
address bar keeps `#3` as typed (the inbound hop is not echoed); the app
reports only lesson IDs, so the first Next rewrites it to `#3.2`.

## Changes

- `examples/tutorial/deeplink.go`
  - `lessonByID` → `resolveRoute(id)`: exact lesson ID first, else a bare
    chapter number → the first `flatLessons` entry with that `ChapterNum`
    (the flat index is in reading order, so that is N.1).
  - `chapterNumber` is hand-rolled: `strconv.Atoi` accepts `+3`, ` 3`,
    `03`, none of which the app ever reports, and a link should round-trip
    exactly. Leading zero / empty / non-digit → 0 → no match.
  - `goTo` compares against `current` *after* resolving, so `#3` while 3.1
    is on screen is a no-op and the demo state survives. The `""` branch
    became explicit (`PopToRoot` only when a lesson is showing).
- `wasm/index.html`: hash pattern `^#(\d+(?:\.\d+)?)$`.
- `docs/tutorial-interactive.md`: one sentence on chapter links.
- `examples/tutorial/deeplink_test.go`: `TestChapterRouteOpensFirstLesson`
  — `route("3")` shows 3.1 at depth 2; toggling lesson 3.1's "Pause the
  count" checkbox then `route("3")` keeps the "Paused —" caption; `"9"`,
  `"0"`, `"03"`, `"+3"`, `" 3"`, `"3."` are all ignored.

## Gotcha

The first version of the test tapped "Pause the count" as a Button; it is
a checkbox (`checkRow`), so use the package's `toggleCheckbox(t, mgr, 0,
true)` helper and assert on the caption text, not a "Resume" label.

## Verification

- `go test ./examples/tutorial`, `go vet`, gofmt clean.
- Local (`?v=5` cache-bust): boot `#6` → 6.1; `#8` → 8.1; `#9` ignored
  (stays on 8.1); Next → `#8.2`.
- Live after the site workflow went green: `#3` → "3.1 The clock:
  UseInterval", copy-link button visible.
