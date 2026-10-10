# Next list: the Validate sort, and N-089's macOS arm for `GrMobSurface.swift`

Session: `87a4b149-9ea8-4e83-ac76-96738a609050`
**Date:** 2026-10-10 17:49 · **Branch:** master (3ba2364 → one commit with this doc)

## Ask

1. `/next-list` with no arguments: living-list mode, the default `n` of 15
   and the default sort by age.
2. "fix N-089".
3. `/sw`: save this doc, commit and push.

## 1. The next-list pass

### Dating and lapses (L2, L3)

- 295 session docs, numbered oldest → newest. The newest,
  `2026-1010-1728-keystore-binding`, counts as age 0. Every `raised` stem
  exists.
- `git log -p --follow` over the file: every ID ever written (N-001…N-097)
  is still present, none twice. **Next ID** was N-098, above the highest.
  The working tree matched HEAD.
- Transition check: all 15 docs in the window use the `Closed: … Raised: …`
  summary, so nothing came in the old way. Cross-check: every Raised ID
  exists, every Closed ID is in Closed, and N-061 (Declined) is in Non-goals.
  N-086's closure names no doc stem. Commit `268681b` closed it, and no doc
  summary lists it as Closed. That is not a lapse.
- **No lapsed items.**
- No session since 2026-09-29 did device work. None of the last 12 docs
  mentions the Fold6, TalkBack, VoiceOver or a device pass. So no Validate
  check had been run since its last update.

### Premise checks (L4)

Two read-only Explore agents ran in parallel. One took the iOS items
(N-004, N-021, N-028, N-040, N-043, N-044, N-090). The other took the
Android and web items (N-013, N-015, N-024, N-045, N-047, N-051, N-092,
N-096) and details of N-062 and N-068. I re-ran `ios/verify/run.sh` and
traced N-002, N-003 and N-089 myself.

| Item | Verdict | What the code says |
| --- | --- | --- |
| N-089 | **partly wrong** | The data-layer checks build and pass. Only the view-layer `swiftc -typecheck ../GrMob/Runtime/*.swift` fails, which leaves it and the `-O -wmo` Release guard unrun. Five other UIKit runtime files sit behind `#if canImport(UIKit)` with a macOS stub. `GrMobSurface.swift` was the only one without. |
| N-021 | **partly stale** | No longer blocked. `grMobDimension`'s `relativeCap` arm (`0f23f7f`, N-080) is the shrinkable px box, but it is gated on a percentage MaxWidth. The floor is unchanged (`GrMobMinContent.width`, pinned at 110). `GrMobImage` is a plain `AsyncImage` with no natural size. |
| N-028 | **partly stale** | "Type-checked only" was stale. `AppWindowReader` is mounted on every launch, and its scheme report was seen live (N-050, N-085). Split View, Stage Manager and rotation are still unseen. |
| N-062 | stale line | "SwiftUI unseen" for the Opacity ease. The same item's first bullet records 17–18 values on the simulator on 2026-09-28. |
| N-066, N-072 | stale refs | Both pointed at N-078 as pending, but it closed on 2026-09-29. The grid is now a container of Buttons, and "Price" holds two named sliders. |
| N-004 | finding | `testEscapeClosesTheDrawerAndTheDialog` checks only the drawer, despite its name. An iOS Modal has no Escape claim of its own (`core.OnEscape`'s doc). |
| N-002 | finding | "Inert (an iPad keyboard still reaches a shut panel)" is not an open question. SwiftUI does not read Inert, as `core.Style.Inert` documents. |
| N-068 | finding | The "▸" comes from the shared `disclosure` helper (`comps/disclosure.go`), so Accordion, Discussion and the grouping widgets share TreeView's RTL chevron. No mirror prop for Text exists. |
| N-003 | re-rated | Nothing waits on a settings edit. The hook that hit it landed by the user pasting it (`bbf5fed`). Claude Code now has an `update-config` skill as the route for settings edits, untried. |
| N-043 | holds, narrower | `write()` still does `replace` then `setCaret`, and nothing uses `inputDelegate`. Since `carryPlan`, the common case needs no correction. |
| N-013, N-015, N-024, N-040, N-044, N-045, N-047, N-051, N-090, N-092, N-096, N-097 | hold | BOM still `2024.12.01`. Force dark is opted out twice (`themes.xml`, `MainActivity`). No iOS 17 runtime is installed. `primeKeyboard` is still used. No proxy for `blb`. No viewport meta in `htmlout`. |

### Edits to `ai_docs/todo/next-list.md`

- **Conventions:** a bullet defining Open (build) and Validate (testing).
  "Nothing leaves" and "ID order" now cover both sections. Validate got a
  three-line intro.
- **Moved Open → Validate (18):** N-002, N-004, N-005, N-006, N-028, N-030,
  N-040, N-044, N-049, N-062, N-064, N-065, N-066, N-068, N-070, N-072,
  N-090, N-091. The remaining work of each is only a check. The Validate
  section arrived with N-097 in the previous session, and nothing had been
  sorted into it yet.
- **Kept in Open:**
  - N-058: a blocker to diagnose.
  - N-089: the fix is in a product file.
  - N-043: a guessed defect with a candidate fix, not a check.
- **Split out:**
  - **N-098** (medium, API decision): the disclosure chevron under RTL, from
    N-068. Its `raised` is `2026-0922-0204`, where the chevron was first
    written down.
  - **N-099** (low): SwiftUI does not read Inert, from N-002. Its `raised` is
    `2026-0919-1254`.
  - Both keep their first appearance as `raised`, per the file's
    Conventions, not this doc.
- **Re-rated:** N-003 from medium to low. N-021 drops "(blocked)" and stays
  medium.
- **Corrected in place:** N-002, N-004, N-021, N-028, N-043, N-062, N-066,
  N-068 (also a garbled Wizard sentence), N-072, N-089, N-090.
- The moves were done by a scratch script that cuts items at `- **N-###**`
  boundaries. A diff of the sorted non-blank lines before and after showed
  nothing lost or altered.

### Left for the user

- **Candidates for Non-goals:**
  - N-013, N-015 and N-027 each have a standing "leave it" recommendation.
  - N-045 is by design.
  - N-051 is moot.
- **Candidates for Roadmap** (the file has no Roadmap section yet):
  - N-009 waits on AlarmKit or full-screen intents.
  - N-047 waits on a Compose BOM past foundation 1.10.
  - N-092 waits on a browser app wanting verses.

## 2. N-089: `ios/verify/run.sh` compiles again

### Cause

`GrMobSurface.swift` (N-085, `6fda8ec`) opened with a bare `import UIKit`.
The run script typechecks `Runtime/*.swift` for `arm64-apple-macos14.0` with
the Command Line Tools' `swiftc`, where UIKit doesn't exist, so the
view-layer typecheck stopped there. The Release-build guard after it never
ran.

### Fix

`GrMobSurface.swift` now follows the `canImport(UIKit)` pattern the other
UIKit runtime files use:

- `import UIKit` under `#if canImport(UIKit)`, `import AppKit` under
  `#elseif canImport(AppKit)`.
- The pure SwiftUI parts compile on both platforms untouched:
  `grMobShellPage`, `GrMobBarSurfaceKey` and `grMobClaimsBars`.
- `grMobIsDark`'s luminance rule stays shared, and only the component read
  is per platform:
  - iOS: `grMobSRGBComponents` via `UIColor.getRed`, exactly the old
    behaviour (nil reads as light, as before).
  - macOS: `NSColor(color).usingColorSpace(.extendedSRGB)`. Extended sRGB
    because that is the space UIColor's `getRed` reports in.
  - So the rule that must match Android's `luminance() <= 0.5` is compiled
    by both passes, not stubbed out on one.
- `GrMobSystemScheme` and `GrMobWindowStyle` (the `UIViewRepresentable`
  probe) are UIKit-only.
  - The `#else` arm defines `GrMobWindowStyle` as a `View` whose body is
    `EmptyView()`, because `GrMobRoot` in Renderer.swift names it on every
    platform.
  - `GrMobSystemScheme` has no macOS arm. Only `App/AppWindow.swift` uses it,
    and the App layer is typechecked against the iOS SDK alone.
- The file header gained a "The macOS build" section.

### Verified

- `sh ios/verify/run.sh` exits 0, with all 16 passes OK. These include "view
  layer type-checks", "view layer survives whole-module optimisation (the
  Release build)" and "app layer type-checks against the iOS SDK". The last
  covers the UIKit arm.
- A throwaway macOS probe (scratchpad, not in the repo) compiled
  `GrMobSurface.swift` with a `main.swift` and ran `grMobIsDark` through the
  NSColor arm:
  - Light: #FFFFFF, #F2F2F7, and #BCBCBC (luminance about 0.503, just over
    the threshold).
  - Dark: #1C1C1E, #2C2C2E, black and #2A78D6.
  - All as expected.
- **Not run:** the app in the simulator or on a device. The iOS code path is
  the same `UIColor.getRed` call moved into a helper.

## Files touched

- `ios/GrMob/Runtime/GrMobSurface.swift`: guards, the shared component read,
  the macOS stub, and the header section.
- `ai_docs/todo/next-list.md`: the pass above, plus N-089 moved to Closed.
- This doc.

## What went wrong

- **zsh globbing in greps.** `grep -rn … --include=*.swift` unquoted failed
  with "no matches found", and so did `echo ======` (`=word` expansion).
  Quote `--include='*.swift'`, and avoid bare `=` runs in separators.
- **The N-057 closure overstated a test.** It says
  `testEscapeClosesTheDrawerAndTheDialog` covers 6.6's dialog, but the test
  never opens it. Recorded under N-004, not fixed.
- **One path mistake in a prompt.** The Android agent noted that
  `Renderer.kt` lives in `android/app/src/main/java/com/grmob/runtime/`.
  It also noted that the comment near `Renderer.kt:347` concerns the
  radiogroup `collectionItemInfo`, not the secure field. N-047's text
  already said so.

## Next

Closed: N-089. Declined: None. Raised: N-098, N-099 (split out of N-068 and
N-002; `raised` is where each first appeared). Deferred: None. Promoted: None.
Moved: N-002, N-004, N-005, N-006, N-028, N-030, N-040, N-044, N-049, N-062,
N-064, N-065, N-066, N-068, N-070, N-072, N-090, N-091 → Validate.
Updated: N-002, N-003, N-004, N-021, N-028, N-043, N-062, N-066, N-068,
N-072, N-090. Full list: `ai_docs/todo/next-list.md`.
