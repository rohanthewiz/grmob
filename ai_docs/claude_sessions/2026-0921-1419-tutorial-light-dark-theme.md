# Tutorial light / dark theme switching

Session: `24d82d8c-d4d7-4340-aae9-a6e45430fe0a`

## Ask

Light/dark theme switching for the web-based dual-pane interactive tutorial.

## Decisions (user)

- **Scope: full app dark theme**, not page chrome only. The guide pane and the
  phone repaint, not just the header and bezel.
- **Palette lives in the tutorial** (`examples/tutorial/theme.go`), not as a
  bundled `core.DarkTheme`. Core's palette censuses assume a light page, so
  a bundled theme would be a framework decision of its own (N-074).

## What was there

- `wasm/index.html` already switched its chrome tokens on
  `prefers-color-scheme`. The guide pane (`#tutorial-guide`), the phone glass
  (`#tutorial-phone-screen`), the phone-layout `.screen` and the `.boot` state
  were hard-coded `#ffffff` / `#000000`.
- Core had no dark theme. It did ship `DefaultDarkChartColors` and
  `DefaultDarkSequentialColors`, which had no consumer (N-018).
- Tutorial content and comps are nearly all theme-driven. Code blocks are
  Darcula in both schemes. `cachedStamp` reads no theme.

## What was built

### Go: `examples/tutorial/theme.go` (new)

- **`theme` host event**, payload `{scheme: "light"|"dark"}`. It mirrors
  split.go's `layout` event:
  - `bootTheme`: a package-level subscriber holds a scheme sent before the
    first render, with the same live-tree counting, so it never leaks into the
    next app in the process.
  - `useColorScheme(sctx)`: a once-per-tree subscription on the session scope.
    It calls Set only on a change.
- **`withScheme`** renders the app through `ctx.WithTheme(darkTheme)`, a
  context swap rather than `core.WithTheme`, because `core.WithTheme` adds a
  `"Theme"` wrapper node. Switching would then move the whole tree down a
  level and rebuild it. Why the swap is safe:
  - The themed copy delegates its hook slots to `hookOwner`.
  - `Scope` and `disposableScope` re-inherit the theme on every pass, so
    frames that were already pushed repaint too.
  - Light is byte-identical to the old tree.
- **`darkTheme`**: a copy of `DefaultTheme`, with colours restated from
  Apple's dark system palette.
  - The page (Background) is `#1C1C1E` and Surface/Card is `#2C2C2E`.
  - Primary is `#409CFF`. The Button declares black ink over it (7.4:1),
    because white measures only 2.8:1. That is AmberTheme's pattern.
  - The `*OnLight` tones are set to light inks: in this theme they mean "ink
    on the page".
  - Chart and Sequential use the Dark lists.
  - The contrast table in the doc comment was computed (a WCAG script), not
    guessed. The first draft's numbers were off and were corrected.
- `app.go` adds a `dark core.State[bool]` seeded from `bootDark()`, calls
  `t.useColorScheme(sctx)`, and returns
  `t.withScheme(t.withLayout(core.Navigator(t.Home)))`.

### Page: `wasm/index.html`

- **Tokens.** `:root` is dark by default. `@media (prefers-color-scheme:
  light)` is guarded by `:root:not([data-theme="dark"])`, and
  `:root[data-theme="light"]` restates the light block. The two light blocks
  must stay identical.
  - New tokens: `--screen-bg`/`--screen-fg`, equal to the app theme's
    Background/TextPrimary, and `--boot-*` for the loading state.
  - `color-scheme` is set, so the browser's scrollbars and form controls match.
- **Early head script.** It sets `data-theme` from localStorage before first
  paint, so there is no flash for an explicit pick. Under System the attribute
  is absent and the media query decides.
- **Header switch.** System / Light / Dark, saved under
  `grmob-tutorial-theme`.
  - The `.layout-toggle` CSS was generalized to `.pill-toggle`, which both
    switches use.
  - `sendTheme()` resolves System against `matchMedia`, updates
    `aria-pressed`, and sends the host event. It runs at page load, on a
    click, on an OS change, and in `boot()` before `RenderInitial`.

### Tests: `examples/tutorial/theme_test.go` (new)

The tests use the two themes' TextSecondary (the caption ink) as the witness
for which palette a tree was drawn in. They check:

- light is the default;
- dark repaints, and dark→light restores the exact wire JSON;
- both split panes get the dark palette;
- a demo keeps its state across a switch;
- a scheme sent before the first render is the first frame's and doesn't leak
  into the next app;
- `newDarkTheme` doesn't write through to `DefaultTheme`.

## Verification

- `go test ./examples/tutorial/ ./core/ ./comps/` all pass.
- `./build.sh` rebuilt `wasm/main.wasm`.
- One manual Chrome pass on `localhost:8093/#1.2` in the split layout:
  - Dark: the guide background computes to `rgb(28,28,30)`, and the guide and
    phone demos are both dark.
  - Light: the guide background is white, and the choice persisted to
    localStorage.
- Reset to System afterwards.

## Behaviour change

A dark-OS reader used to get dark chrome around white panes. They now get the
whole tutorial dark unless they pick Light.

## Next

Closed: None. Declined: None. Raised: N-074, N-075.
Deferred: None. Promoted: None.
Updated: N-018, N-050. Full list: `ai_docs/todo/next-list.md`.
