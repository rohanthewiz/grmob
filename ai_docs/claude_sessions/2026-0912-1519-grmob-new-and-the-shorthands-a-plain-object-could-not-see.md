# Session: `grmob new`, and the shorthands a plain object could not see

**Session ID:** session_01HbRxrY6havcS6djbXtapZN
**Date:** 2026-09-12, ~15:19
**Branch:** master

## Goal

Three asks, the second arriving after the first shipped:

1. **A bootstrap for apps in their own module.** On another machine, building
   the counter app took a helper session and ended with a hand-rolled
   `run.sh` (python3 http.server), a copied `grmob-runtime.js`, and a local
   patch to that copy for a `core.Gap` that rendered as no spacing. The ask: a
   script or app that starts a local app targeting WASM by default, and can
   later target the natives once their SDKs are installed.
2. **Fix the `overflowX` bug** that same helper session had found and left:
   `Overflow("hidden")` on a TextGrid measured `overflow-x: auto`.
3. **Make the test DOM expand CSS shorthands**, since without that the verify
   suite could not hold either fix in place. Then make the README say how to
   start a new app, and `/sess-wrap`.

## What the other machine had actually hit

Not a bug on master. The gap fix (`1c04156`, 2026-09-04) is in no tag:
`git tag --contains 1c04156` is empty, `@latest` resolves to **v0.2.4
(2026-09-02), 197 commits behind master**. So `go get` hands every new
machine the gap bug, no `webhost`, and none of this session's work. The helper
session's patch was a correct re-derivation of a fix that already existed
upstream.

The deeper cause was structural: nothing in this repository could build an app
that lives *outside* it. `./build.sh` builds `./wasm`, whose `main.go`
dot-imports an example; `android/build.sh` and `ios/build.sh` bind
`./examples/mobileapp` from grmob's root; `serve -dev` runs `build.sh` in the
module root and watches `./wasm`. An app in its own module could only copy
those files — pinned to whatever version it read.

## What was built

### `webhost` — the browser host as a library

`webhost.Run(ctx, app)` installs the whole `GrMobWASM` surface
(RenderInitial, RenderAgain, ReceiveEvent, IsDirty, HostEvent, Shutdown),
wires `GrMobApplyPatches` and `GrMobSystemEvent` when the page defines them,
and blocks until Shutdown so `go.run`'s promise settles for hot reload.

- `webhost/host.go` (`js && wasm`), `webhost/bindings.go` (untagged, so a
  host-GOOS test can read `Bindings`).
- `wasm/main.go` and `wasm/shots/host/main.go` were **not** migrated — each
  has documented reasons to differ, and several prose checks quote the former.
  `TestHandWrittenHostsInstallTheLibrarysBindings` parses both: the tutorial
  host must install exactly `Bindings`; the shots host (no hot reload, no
  Shutdown) a subset with no unknown name. This follows copies_test's "at three
  it becomes a shared package" rule without rewriting the two existing copies.
- Excluded from `internal/apidoc` (the generator reads the host GOOS and would
  render a page with one variable); documented in `docs/platforms/wasm.md`.

### `cmd/grmob` — `new`, `doctor`, `android`, `ios`

The rule every piece keeps: **anything grmob owns comes from the grmob module
in the app's go.mod, at build time.**

```
app module (yours)                     grmob module (go.mod pins it)
app/app.go         ─── imports ──────▶ core, mobile, ...
wasm/main.go       ─── webhost.Run ──▶ webhost
build.sh           ─── copies ◀──────── wasm/grmob-runtime.js, camera.js (if changed)
dev.sh             ─── go run ───────▶ serve -dev
android/, ios/     ◀── vendored once ── android/, ios/   (-refresh to re-copy)
```

- **`new <dir>`** renders embedded templates (`[[ ]]` delimiters, because
  build.sh speaks `{{.Dir}}`), writes `grmob.json` (name + application ID),
  runs `go mod init` / `go get` or `-replace` / `go mod tidy`, and does a first
  build. Version resolution: `-replace` > `-grmob` > the command's own module
  version (`go run …/cmd/grmob@vX`) > a dev build finds its checkout through
  `runtime.Caller` and uses a replace > `latest`. Go templates carry `.tmpl`
  so this module never compiles or counts them.
- **`build.sh`** copies the runtime JS with `cmp -s || cp`. A plain `cp` gives
  the file a new mtime every build, `serve -dev` reads a changed `.js` in the
  served dir as a page edit, and every hot swap would be followed by a page
  reload that loses the app's state. The copies are gitignored: never edited,
  never stale.
- **`doctor`** — one check table per target, the same tables `android` and
  `ios` stop on, so the advice printed is exactly the condition enforced.
  Android: SDK, NDK, JDK 17+ (by version — macOS's `/usr/bin/java` stub exists
  and fails; Android Studio's JBR is accepted), adb (optional). iOS: macOS,
  full Xcode, simulator SDK, xcodegen.
- **`android` / `ios`**: prerequisites → vendor the shell once (skip lists for
  build state, `verify/`, the shells' own `build.sh`; `gradlew` made
  executable because the module cache drops modes) with anchored identity
  patches → `go get` + `go build` gomobile/gobind from the app's graph into
  `.grmob/bin` (the version grmob's `tool` block pins; no global install) →
  `gomobile bind grmob/mobile ./app` → Gradle `assembleDebug` / `xcodegen` +
  simulator `xcodebuild`. A `.grmob-shell` file records the version vendored;
  a build against a different go.mod warns.
- Identity patches: Android `applicationId` and `android:label` (the Gradle
  namespace stays `com.grmob.app` — it names the shell's classes; activity
  component is `<id>/com.grmob.app.MainActivity`); iOS
  `PRODUCT_BUNDLE_IDENTIFIER` and a `CFBundleDisplayName` inserted at
  `UILaunchScreen`'s indentation. `TestShellPatchesStillApply` runs them against
  this repository's own shells, so moving an anchor fails here, not on a user's
  first native build.

### Verified end to end on a scaffolded app (scratchpad `hello`, replace → checkout)

| Target | Result |
|---|---|
| Browser | Column `row-gap` 12px, Row `column-gap` 8px; two taps → Count: 2; no console errors; editing the heading hot-swapped with a `window` marker surviving (no page load) |
| Android | `BUILD SUCCESSFUL in 17s`; `aapt dump badging`: package `com.example.hello`, label "Hello GrMob" |
| iOS | xcframework (device + simulator), xcodegen, simulator `xcodebuild`; built `Info.plist`: `com.example.hello`, "Hello GrMob". Built, **not launched** |

`TestNewScaffoldsAnAppThatBuildsAndPasses` holds the browser path: scaffold,
the app's own build.sh, `go test`/`go vet` in it, a byte-identical runtime
copy, and a second build that must not rewrite the runtime's mtime.

### Guards this repository raised, and the answers

- **Tracked Go file count** 461 → 469 in five sentences
  (repowalks_test.go ×3, timings_test.go ×2).
- **A second `-short` lever**: removed rather than budgeted — the scaffold test
  is ~1s on a warm cache.
- **A second hand-rolled version comparator** (`versionLess`, to pick the
  newest NDK): deleted. gomobile already picks the newest *compatible*
  side-by-side NDK by reading each one's `meta/platforms.json` and
  `meta/abis.json`, which is a better answer than a name sort; doctor now only
  asks whether one exists, and `ANDROID_NDK_HOME` is passed only when the user
  set it.
- **apidoc**: `webhost` added to `excluded()` with the reason.

## The shorthand bug class

### What was wrong with the harness

`dom.mjs` gave every element `style = {}`. For independent properties that is
faithful; for a shorthand and its longhands it is blind by construction.
`styleFromGrMob` is *total* — it restates every property and writes `""` for
the unset ones — so it is precisely the code that assigns a shorthand and then
clears a longhand of it in the same object. It did it twice: `gap` then
rowGap/columnGap (fixed 2026-09-04), and the TextGrid chassis's
`out.overflowX = out.overflow ? "" : "auto"`, where the `""` did not mean
"leave it" but removed the x half of the author's `overflow`.

### `wasm/verify/cssstyle.mjs`

A Proxy-backed inline style:

- Longhands are the only stored truth; shorthands expand on write (nested:
  `border` → `borderTop` → `borderTopColor`) and compose on read.
- Modeled: gap, overflow, padding, margin, inset, border and its sides,
  borderWidth/Style/Color, borderRadius (no slash form), flex, gridArea, font
  (CSS-wide keywords only).
- Reads: all longhands never written → `undefined`; any missing or `""` → `""`;
  otherwise the author's text if no longhand changed since, else a
  serialization.
- **Deliberate differences, each because a suite depends on it:** never-written
  reads `undefined` (browsers say `""`); unchanged shorthands read back as the
  author's text ("0px 16px 0px 16px") where Chrome returns its shortest form;
  no validation or colour canonicalization.
- **Refuses to guess:** an unmodeled value form throws, and so does an element
  given both an UNMODELED shorthand (background, outline, textDecoration,
  transition, animation, flexFlow, place-*, gridRow/Column, logical box
  properties, …) and one of its longhands.
- Enumeration stays "what was assigned", so totality_test's `Object.keys`
  sweep is unchanged in meaning.
- `erasedShorthands(style)`: shorthands last assigned a real value with a
  longhand since set to `""` — the bug class as a query. An override (`border`
  then `borderBottom`, the tab strip's own sequence) is not an erasure.

### Grounded in a real browser, not memory

Nineteen assignment sequences were run against
`document.createElement("div").style` in Chrome; `shorthand_test.mjs`'s CSSOM
table holds the shim to Chrome's reads. Two outcomes worth remembering: Chrome
serializes `overflow: hidden` + `overflowX: auto` as `"auto hidden"`, and
`border: none` resets a side to `medium none currentcolor`. All nineteen
matched the model as first written.

Considered and not done: a claim in `browser.mjs` that re-checks the table in
headless Chrome on every run (9.5k lines with counted claims in its header).
The table records where its values came from and how to re-check a row.

### The fix, proven by failing first

- `totality_test.mjs`: `FULL_STYLE` gained `Overflow` (its absence is how the
  bug hid under a green sweep), and a new test sweeps every node type for
  erased shorthands on creation, after a restyling patch, and after a patch
  that styles a bare node. **Run against the unfixed runtime it failed with
  `TextGrid on creation: [ 'overflow' ]`.**
- `grmob-runtime.js`: `out.overflowX = out.overflow || "auto"` — never `""`,
  the author's value wins on both axes, matching htmlout (whose
  `textGridChassis` `overflow-x:auto` precedes the author's `overflow`).
- `shorthand_test.mjs` pins the TextGrid contract by name on mount, when
  Overflow is dropped (x back to auto, y cleared), and on a new value.
- `runtime_test.mjs`'s gap test previously asserted `style.gap === undefined`,
  which a CSSOM model makes false. Restated as its actual decision: the runtime
  never *assigns* `gap` (not in `Object.keys`), and `gap` reads "8px" composed
  from the longhands.

Against the new shim before any fix: 433/434 Node tests passed, the one
failure being that gap assertion; no unmodeled-mix guard fired anywhere.

## Docs

- **README "Run it"** now opens with *Start your own app* (`grmob new`,
  `dev.sh`, `go test ./app`, then doctor/android/ios, and the "everything
  grmob owns comes from go.mod" rule), keeps `go get` for existing modules,
  and moves the checkout's tutorial commands under *Run this repository's
  apps*. "One app, four targets" points own-module apps at `grmob android|ios`.
- `docs/getting-started.md`: *Start your own app*.
- `docs/platforms/wasm.md`: webhost for apps in their own module, and why
  build.sh copies only on change.
- `docs/platforms/native.md`: what `grmob android|ios` do.
- `serve/main.go`: the log said "GrMob tutorial" for every app; now "GrMob app".

## Verification at the end

- `go test ./...`: all ok.
- `wasm/verify/run.sh`: `EXIT=0` — Node suites (now including the CSSOM table,
  the erasure sweep and the TextGrid test) and the headless-Chrome pass.

## Files

New: `webhost/{host.go,bindings.go,webhost_test.go}`,
`cmd/grmob/{main.go,new.go,doctor.go,native.go,new_test.go}`,
`cmd/grmob/templates/{app/app.go.tmpl,app/app_test.go.tmpl,wasm/main.go.tmpl,wasm/index.html.tmpl,build.sh.tmpl,dev.sh.tmpl,dot-gitignore,README.md.tmpl}`,
`wasm/verify/{cssstyle.mjs,shorthand_test.mjs}`.

Changed: `wasm/grmob-runtime.js`, `wasm/verify/{dom.mjs,totality_test.mjs,runtime_test.mjs,repowalks_test.go,timings_test.go}`,
`internal/apidoc/apidoc_test.go`, `serve/main.go`, `README.md`,
`docs/{getting-started.md,platforms/wasm.md,platforms/native.md}`.

## Next

1. **(new · value high) Tag a release.** `@latest` is v0.2.4, which has neither
   `cmd/grmob` nor the gap fix, so the README's first command fails on any
   other machine until a tag (e.g. v0.2.5) is pushed. Once it is, run
   `go run github.com/rohanthewiz/grmob/cmd/grmob@latest new` on a clean
   machine without `-replace` — every end-to-end run this session used a
   replace to the checkout, so the module-cache path (read-only files, no
   execute bits, a real `go get`) is exercised only by reasoning.
2. **(new · value med) Launch the scaffolded app on a simulator and a device.**
   iOS was built for the simulator but never launched; Android assembled but
   `-install` never ran (no device connected). Both prove linking, not that
   the shell renders the app.
3. **(new · value med) `grmob ios -run`.** The command prints a
   `simctl install/launch` line instead of doing it. Booting a simulator and
   picking a device is the missing step to a one-command iOS loop.
4. **(new · value med) The CodeEditor gutter clears an author's left
   padding.** `syncCodeGutter` writes `el.style.paddingLeft = ""` when line
   numbers are off, which removes a `Padding.Left` the Style set — the same
   write-`""`-to-a-longhand shape as the overflowX bug, on a path the erasure
   sweep does not visit (it happens after the style pass, and CodeEditor is not
   in totality_test's NODE_TYPES). Unconfirmed as a user-visible bug; worth a
   test first.
5. **(new · value low) `FULL_STYLE` is kept by hand.** Nothing checks it against
   the Style fields `styleFromGrMob` reads, which is how `Overflow` was missing.
   A source scan (every `style.X` read appears as a key) would close it.
6. **(new · value low) Re-check the CSSOM table in a browser automatically.**
   `shorthand_test.mjs`'s expected values were taken from Chrome once, by hand.
   A `browser.mjs` claim could replay the table; declined this session for its
   cost against a table that is not expected to move.
7. **(new · value low) Scaffold hygiene.** A native build leaves
   `golang.org/x/mobile` in the app's go.mod as indirect, and a later
   `go mod tidy` drops it again (each native build re-adds it — harmless
   churn). The copied iOS `project.yml` keeps the demo's permission usage
   strings and the `grmob://` URL scheme, and `GrMobUITests` rides along.
8. **(new · value low) The shots host still has no `Shutdown`.** Allowed as a
   subset by `TestHandWrittenHostsInstallTheLibrarysBindings`; if it ever
   runs under `serve -dev` it will page-reload instead of hot-swap.
9. **(carried · value med) `docs/api/core.md` is one 7,700-line page.**
   Splitting would need the flat-sibling link scheme reconsidered.
10. **(carried · value med) Nothing checks links from `docs/api/` out into the
    narrative pages.** `TestEveryGeneratedLinkResolves` skips `../`
    destinations; the overview's `../concepts/architecture.md` is unverified.
11. **(carried · value low) `aria/spec` in a user-facing reference.** One line
    in `Packages` either way.
12. **(carried · value low) The skill's pointer URL is dead until merge.**
    Glance after the first push to master that the raw URL serves.
13. **(carried · value low) The Android remedy drill has never run.** Needs an
    SDK, a network and a cold gradle cache; the first CI run touching it is its
    first real test.
14. **(carried · value low) `hero.png` is a picture of pictures and its parts
    are held twice.** Stops being right the moment the hero is cropped.
15. **(carried · value low) A third face is still a skip.** The "WHICH DOES NOT
    SEPARATE" arm is unreachable by any face this project has met.
16. **(carried · value low) Windowing as a proposal.** Declined on the payload
    in `non_goals.md`; `TestWhatWindowingWouldSave` is the profile to re-take.
17. **(moved) The four hardware items.** `ai_docs/plans/need_hardware.md`.
18. **(declined, non-goal)** Twenty-seven entries, unchanged. See
    `ai_docs/plans/non_goals.md`.
