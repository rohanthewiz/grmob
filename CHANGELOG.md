# Changelog

All notable changes to GrMob are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project
uses [Semantic Versioning](https://semver.org/) once tagged.

## [Unreleased]

### Build
- Android toolchain moved to Gradle 9.7.1, AGP 9.4.0 and Kotlin 2.4.10.
  AGP 8.x cannot run on Gradle 9.6+ (it uses `InternalProblems`, which Gradle
  removed there), so the AGP major bump was the only way off the 8 line.
  Kotlin 2.x also replaces `composeOptions`/`kotlinCompilerExtensionVersion`
  with the `org.jetbrains.kotlin.plugin.compose` plugin, and AGP 9 supplies
  Kotlin itself, so the `kotlin-android` plugin is gone. The build is now free
  of Gradle deprecation warnings.
- Added a committed Gradle wrapper (`android/gradlew`, pinned to 9.7.1 with a
  distribution checksum). Building no longer requires a system Gradle install;
  `cd android && ./gradlew assembleDebug` works from a clean clone.
- Debug APK shrank from 19.3 MB to 16.1 MB: AGP 9 strips `libgojni.so`, which
  AGP 8.1 reported it could not do.

## [0.1.0] — 2026-09-01

First tagged release. Everything below was in place at the cut.

### Core engine
- Declarative views as Go functions (`View`, `ComponentFunc`), with `If`,
  `Match` and `For` for conditional and iterative rendering.
- `NewState` local state, plus hooks: `UseInterval`, `UseTimeout`,
  `UseEffect`, `UseMemo`, `UseReducer`, `UseChildContext`.
- Reconciler that diffs two trees into a minimal patch list, addressed by
  path-based render IDs, with a cached-subtree fast path.
- Render manager with dirty tracking, callback dispatch under one render
  mutex, and a push channel for renderers that can accept patches.
- Error boundaries and driver-level panic guards: a panicking component
  costs its subtree, not the process.
- Debug mode that reports cursor drift, duplicate keys, unknown items and
  panics, and is zero-cost when off.

### Layout, styling and theming
- `Row`, `Column`, `Box`, `Card`, `Scroll`, `SafeArea`, keyed `List`,
  `Spacer`, `Image` with `ContentMode`.
- Flex weights, gap, justify and align on every target; positioning,
  z-index, shadows, borders, radius and transitions.
- `Theme` with palette, typography, spacing and component defaults; two
  bundled themes (`DefaultTheme`, `MaterialTheme`); scoped `WithTheme`.
- A native disabled state that propagates through the subtree.

### Widgets, forms and navigation
- `components`: `Screen`, `Button`, `InputRow`, `SegmentedControl`, `Card`,
  `ListRow`, `Badge`, `Chip`, `Separator`, `Avatar`, `ProgressBar`,
  `FormField`, `Accordion`, `Tabs`, with `Variant` × `Emphasis` color axes.
- `forms`: a rule vocabulary, cross-field checks, four reveal policies and
  server-side errors.
- Focus and blur events, programmatic focus, focus traversal order, and
  keyboard-aware regions.
- `Navigator` with `Push`, `Pop`, `Replace`, `PopToRoot`, `Reset` and
  per-frame state; `Modal` and toasts.

### Targets
- Android shell and renderer in Jetpack Compose; iOS shell and renderer in
  SwiftUI; both driven through the gomobile bridge in `mobile`.
- Browser target: WebAssembly host plus `wasm/grmob-runtime.js`, applying
  patches to the DOM.
- `htmlout` and `jsonout` exporters for previews, tooling and byte-for-byte
  snapshot tests.
- Verify harnesses for every renderer that replay the Go engine's own patch
  transcripts.

### Documentation and examples
- MkDocs site: getting started, concepts, widget library, platforms.
- An interactive tutorial (40 lessons, 8 chapters) that is itself a GrMob
  app, and a start-to-finish todo-app tutorial with `bytdb` persistence.

[Unreleased]: https://github.com/rohanthewiz/grmob/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/rohanthewiz/grmob/releases/tag/v0.1.0
