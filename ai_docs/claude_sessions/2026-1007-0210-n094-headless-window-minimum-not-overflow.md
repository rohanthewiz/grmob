# N-094: the "overflow" was headless Chrome's minimum window width

Session: `ca6ca295-dc54-4e57-95ba-f0c109e7604b`
**Date:** 2026-10-07 02:10 · **Branch:** master (002f7b2 → one commit with this doc)

This session also did N-083 (`2026-1007-0157-n083-hbox-paddingless-row`) and
N-087 (`2026-1007-0204-n087-docsnippets-compile-component-docs`), each in
its own doc and commit.

## Ask

N-094 from the next-list, pasted in from the cats-todo backlog (value low,
unverified): "Cards overflow a 420px viewport in a bare `htmlout` export." A
stock `comps.Card` and `comps.InputRow` inside `comps.Screen{Scroll: true}`
ran past the right edge in headless Chrome (`--window-size=420,…`). The
suspicion was that the export lacks the web runtime's CSS. It had never been
compared with the runtime.

The item was raised in `2026-1003-1852-…`. That session wrote
`htmlout.ExportHTML` from a throwaway test and screenshotted it with
headless Chrome at 420px.

## What was done

All in the scratchpad (nothing in the repo):

- **A scratch Go module** (`replace` grmob with this checkout) rendered one
  `comps.Screen{Scroll: true}` holding a stock `Card` (title + body text) and
  `InputRow` (placeholder + Send button) two ways:
  - `htmlout.ExportHTML` of the rendered node;
  - `render.New(...).RenderInitial()` JSON, mounted with
    `GrMob.mount(json, "app")` in `cmd/grmob/templates/wasm/index.html.tmpl`.
    The wasm boot script was swapped out and `wasm/grmob-runtime.js` loaded
    directly. The runtime's `mount` works without a wasm build, which makes a
    like-for-like comparison of the two web targets cheap.
- **A Node CDP probe** (Node 22's built-in `WebSocket`, Chrome launched
  `--headless=new --window-size=420,900`). It reports `innerWidth`,
  `clientWidth`, `scrollWidth`, `<body>`'s margin and every element whose
  right edge passes the viewport. The modes: the window alone;
  `Emulation.setDeviceMetricsOverride` at 420 with `mobile: false`; and the
  same with `mobile: true`.

## Findings

| page | `--window-size=420` only | override 420, desktop | override 420, mobile |
|---|---|---|---|
| htmlout export | innerWidth **500**, nothing over | 420, nothing over | **980**, nothing over |
| runtime in scaffold page | innerWidth **500**, nothing over | 420, nothing over | 420, nothing over |

- **The overflow was a crop.** `--headless=new` will not make a window
  narrower than 500px, so the page lays out at 500. Chrome's own
  `--screenshot` then writes a **420×900** PNG of that layout, cutting off the
  right 80px. Reproduced exactly: the Card and the Send button are cut at the
  edge in the CLI shot, and both sit well inside the frame in the 420px
  override shot.
- **No overflow on either target** at a true 420px: `scrollWidth` equals the
  viewport and no element passes it. So the item is closed as a false
  premise, not declined.
- **Real differences between the export and the runtime page**, neither an
  overflow:
  - `<body>` keeps the browser's default 8px margin in the export (0 in the
    scaffold page).
  - **The export has no viewport `<meta>`.** Under mobile emulation it lays
    out at the 980px desktop fallback. Raised as N-096.
  - The export also writes no `<head>` at all for this tree:
    `motionStylesheet` writes one only for spin, transition, x-translate or a
    sized-and-padded node (`needsBorderBox`), and neither widget has one.
    That is by design (`borderBoxCSS`'s comment), and it did not matter
    here.
- **`--screenshot` under `--headless=new` did not exit** after writing its
  PNG. It ran as a background task and was stopped by task ID. Only that
  task was stopped, and no stray Chrome process was left.

## Repo changes

- `ai_docs/todo/next-list.md`:
  - N-094 moved to Closed with the measurements.
  - N-096 appended to Open.
  - **Next ID** is now N-097.

Memory: `cdp-browser-emulation.md` now records the 500px window floor, the
`--screenshot` crop, and pinning phone widths with
`Emulation.setDeviceMetricsOverride`.

## Next

Closed: N-094. Declined: None. Raised: N-096.
Deferred: None. Promoted: None.
Updated: None. Full list: `ai_docs/todo/next-list.md`.
