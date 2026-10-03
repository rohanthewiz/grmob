# N-085: the shells' surface and system bars follow the Go tree

Session: `13840e3c-29a7-495d-8adb-c95d6854d306`

## Ask

Do N-085, pasted from the cats-todo backlog: a shell's surface and system
bars followed the system's dark mode, and a light Go theme did not. On iOS a
light app on a dark iPhone drew its dark ink on SwiftUI's black
systemBackground. On Android the window stayed light, but
`enableEdgeToEdge`'s automatic bar style turned the icons white in night
mode. Then `/sw`.

## Decision

The item's first candidate: the shells paint the Go theme's page and pick the
bar style from the Go tree, not the system. One rule on both shells:

```
bars' colour = innermost painted SafeArea's Background
               ?: root node's Background
               ?: shell page (#FFFFFF, core.DefaultTheme's Background)
surface      = root's Background ?: shell page
icons        = light over a dark bars' colour (WCAG luminance ≤ 0.5), dark otherwise
```

An app that follows `Window.ColorScheme` already paints its root (the
tutorial's `paintPage`) or its `comps.Screen`, whose background is forwarded
to the SafeArea. An app that does not follow it states no root colour and
gets the light page and dark icons on any system.

Not chosen: pinning iOS light, the item's second candidate. The tutorial
paints itself dark on a dark system, so a pinned light window would put dark
status text on its dark page.

## Android: `runtime/GrMobSurface.kt` (new)

- `ShellSurface(root, claims)` is composed in `GrMobRoot` after the tree.
  - Its SideEffect sets the window background drawable (the root's colour or
    the shell page; it was Theme.Material.Light's #FAFAFA).
  - It also sets `isAppearanceLightStatusBars` and
    `isAppearanceLightNavigationBars`.
- A painted SafeArea now calls `ClaimSystemBars(bg)` rather than
  `SystemBarIcons`, which was moved out of Renderer.kt with its rationale
  kept.
  - The claim joins a snapshot list (`LocalSystemBarClaims`) in a
    DisposableEffect and leaves it on dispose.
  - Why a claim: with the root writing the icons as well, a recomposition
    that skipped an unchanged SafeArea would have let the root overwrite its
    colour (last SideEffect wins).
- The bars' colour is read twice.
  - Once in composition, which subscribes `ShellSurface` to the claims.
  - Again inside the SideEffect, which is the value written. An apply
    dispatches every DisposableEffect before any SideEffect, so the claims
    have already joined by then, and the first frame is right.
- `MainActivity`'s `enableEdgeToEdge()` is unchanged. Its automatic style is
  overwritten before the first draw.

## iOS: `Runtime/GrMobSurface.swift` (new)

- `GrMobRoot` paints `(root bg ?? white).ignoresSafeArea()` behind the tree,
  so the window no longer shows systemBackground.
- The SafeArea arm reports its background with `grMobClaimsBars`.
  - This uses `transformPreference` on `GrMobBarSurfaceKey`, so the innermost
    SafeArea wins. A plain `.preference` lets the outer view replace its
    subtree's value.
  - `GrMobRoot` stores the result in `@State barSurface`.
- `GrMobWindowStyle` is a zero-size UIViewRepresentable probe. It sets
  `window.overrideUserInterfaceStyle` from the bars' colour.
  - One style covers both the status bar and native chrome. Under the SwiftUI
    App lifecycle the root hosting controller can't be subclassed, so there
    is no separate `preferredStatusBarStyle`.
  - Android sets only the icons. That divergence is N-086.
- The system scheme is now read from the scene.
  - The window override would hide it from `@Environment(\.colorScheme)`,
    which is what `AppWindowReader` used to report.
  - `GrMobSystemScheme` (an @Observable singleton) reads
    `windowScene.traitCollection.userInterfaceStyle`. Trait overrides flow
    down, not up, so the scene still carries the system's style.
  - It watches the style with `registerForTraitChanges`, which compiles for
    UIWindowScene on the iOS 17 target.
  - The reader falls back to the environment until the probe has run. Before
    that, nothing has been overridden.

## Docs

- `core.ColorScheme`'s doc (`core/window.go`) now says it is the system's
  scheme, not what the shell draws, and gives the surface rule.
- Two phrases there went stale with `core.DarkTheme` ("core bundles no dark
  theme", "every palette core bundles assumes" light) and were fixed.
- `docs/api` was regenerated (`go run ./internal/apidoc/gen`).

## Verification

| Case | iOS sim (iPhone 17 Pro, 26.5) | Android emulator |
| --- | --- | --- |
| Demo app (follows no scheme), system dark | white page, dark status text | white page, dark icons and handle |
| Tutorial, system dark | dark page, light status text | dark page, light icons |
| Tutorial, switched to light while running | light page, dark text | light page, dark icons (`cmd uimode night no`) |
| Tutorial, switched back to dark while running | dark again | not run |
| Demo with Screen painted #1C1C1E, system light | light status text | light icons |

- The tutorial switching live on iOS shows the scene-read scheme reaches Go
  through the window override.
- The painted demo was a temporary edit to `examples/mobileapp/app.go`, since
  reverted.
- `go test ./core ./internal/apidoc` passes. Both shells build: gradle
  `compileDebugKotlin`/`installDebug`, and `xcodebuild` after
  `xcodegen generate` for the new Swift file.
- Both shells were left installed with the tutorial, the simulator in light
  appearance, and the emulator with night mode off.

## Gotchas

- **A commit swept part of this work.** Another session's commit `c43e3bc`
  ("Keyed callback IDs, a bundled DarkTheme, chart data per host", already
  pushed) took this session's `ios/GrMob/Runtime/Renderer.swift` edits from
  the shared working tree, but not the new `GrMobSurface.swift`. iOS
  therefore doesn't build at `c43e3bc` alone; this session's commit completes
  it. This session commits only its own paths for the same reason: another
  session had uncommitted work in the tree at `/sw` time (README,
  ai_docs/SKILL*, comps/doc.go, core/view.go, docs pages).
- **Emulator asleep.** A screenshot of an asleep emulator is a black frame.
  Run `adb shell input keyevent KEYCODE_WAKEUP` and
  `adb shell wm dismiss-keyguard` first.
- **Framework contents.** The iOS and Android frameworks hold whatever app
  the last `build.sh` bound. Rebuild with `./examples/tutorial` after
  testing another app.

## Next

Closed: N-085. Declined: None. Raised: N-086.
Deferred: None. Promoted: None.
Updated: None. Full list: `ai_docs/todo/next-list.md`.
