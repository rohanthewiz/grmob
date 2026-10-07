# N-095: ProgressBar's default track is the Border role

Session: `ca6ca295-dc54-4e57-95ba-f0c109e7604b`
**Date:** 2026-10-07 02:16 · **Branch:** master (0c1e0eb → one commit with this doc)

Earlier in this session: N-083 (`2026-1007-0157-…`), N-087
(`2026-1007-0204-…`) and N-094 (`2026-1007-0210-…`), each in its own doc
and commit.

## Ask

N-095 from the next-list, pasted in from the cats-todo backlog (value low,
unverified): the tutorial's contents card drew "0 of 83 lessons opened" over
no visible progress track in dark mode, where the light shot shows a pale
one. It was seen only once, at 0%, in headless Chrome, so it might have been
a track at too low a contrast.

## Cause

- `examples/tutorial/home.go` `progressCard` is a `core.Card` holding a
  caption and a default `comps.ProgressBar`.
- `ProgressBar`'s default track was `t.Colors.Surface`
  (`comps/progress_bar.go`).
- `newDarkTheme` sets `th.Components.Card.Background = surface`, the same
  #2C2C2E as `Colors.Surface`. The track was therefore 1:1 against its
  Card: not low contrast but absent. This holds for every app, not just the
  tutorial.

Measured with `internal/palette` against every fill a control can be drawn
on (`palette.Backdrops`), in every bundled theme:

| theme | old Surface track, worst pairs | Border track, lowest |
|---|---|---|
| DefaultTheme | 1.00 on Surface | 1.13 on Surface |
| MaterialTheme | 1.00 on Surface, 1.04 on Input/TextArea | 1.21 on Surface |
| AmberTheme | 1.00 on Surface, Input, TextArea | 1.24 on Surface/Input |
| DarkTheme | 1.00 on Surface and **Card** | 1.19 on Surface/Card |

`comps.Gauge` already defaulted its track to `Colors.BorderColor()`.

## What landed

- `comps/progress_bar.go`: the default `track` is now
  `t.Colors.BorderColor()`. The comment above it argues the choice:
  - the bar cannot see its parent, so the default must clear every backdrop;
  - Surface is one of the backdrops;
  - a groove is divider-weight, and the value is carried by the Primary
    fill, so the role with no 3:1 floor is the right one.

  It also has a before/after sketch. The `TrackColor` field comment says
  Border and points there.
- `comps/progress_bar_test.go`:
  - `TestProgressBarStructureAndDefaults` now expects
    `theme.Colors.BorderColor()`.
  - **New:** `TestProgressBarDefaultTrackShowsOnEveryBackdrop` renders a
    default bar in each `core.BundledThemes()` and requires at least 1.1:1
    against every `palette.Backdrops` fill. The floor is visibility, not WCAG:
    it sits just under the old light track on a white Card (1.12:1). With the
    old default restored it fails 7 pairs (each theme's Surface, Material's
    and Amber's Card, DarkTheme's Card).
- `docs/api/comps-overlays.md` regenerated for the field comment.

## Checks

- **Before/after in a browser.** The N-094 scratch probe (outside the repo)
  rendered the progress-card shape under `core.DarkTheme`, at 0% and at
  30/83, through `render.New(...).RenderInitial()`. It was mounted with
  `GrMob.mount` in the `grmob new` host page (frame painted #1C1C1E) and
  screenshotted with `Emulation.setDeviceMetricsOverride` at 420px.
  - Before: the 0% card has the caption over nothing, and the 30% bar's
    groove stops at the fill.
  - After: a full-width #38383A groove in both.
  - The `Surface` line was swapped in for the "before" build and restored
    with an exact sed; `git diff` shows only the intended change.
- `go vet ./...`, `go test ./...` and `wasm/verify/run.sh` (runtime
  conformance plus the headless Chrome facts, about 90s) all passed. The
  tree was clean afterwards apart from this change.
- `wasm/verify/browser.mjs` builds its progressbar nodes by hand rather than
  from `comps.ProgressBar`, so nothing there pinned the old track.

## Loose ends

- Light themes' grooves are darker too (DefaultTheme #F2F2F7 → #E5E5EA), so
  `docs/images/tutorial-contents.png` is now a shade off. It was not retaken,
  which needs the wasm build and `wasm/shots/shoot.sh`. The user was told.
- The other session's uncommitted N-031 move in `next-list.md` is again left
  out of this commit.

## Next

Closed: N-095. Declined: None. Raised: None.
Deferred: None. Promoted: None.
Updated: None. Full list: `ai_docs/todo/next-list.md`.
