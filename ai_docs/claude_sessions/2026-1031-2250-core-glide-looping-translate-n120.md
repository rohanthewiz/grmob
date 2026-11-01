# core.Glide, the looping translate, and a gliding StockTicker (N-120)

Session: `121c8fb9-9a2b-413f-8b54-c975c3b91696`
**Date:** 2026-10-31 22:50 · **Branch:** master (fa342fd, plus this commit)

## Goal

The user dropped N-120 from the next list: core had no looping translate, so
`comps.StockTicker` stepped instead of gliding. `core.Spin`'s doc had named
"a second consumer" as the trigger for a general loop. The ticker is that
consumer, and the result is a second narrow loop rather than a general one.
Afterwards the user asked for the skills to be updated accordingly.

## The design: `core.Glide(periodMs)`

- A node slides its own width toward the leading edge every period, then
  restarts from rest. A negative period plays the pass backwards (CSS's
  `reverse`: from one width away back to rest). Zero holds still.
- **Why the distance is the node's own width and not a parameter.** A loop's
  restart is invisible only when the end draws what the start did. For a
  translation that is true at exactly one distance, the node's own extent,
  and only with a copy of the node beside it: as the first copy leaves by its
  width the second arrives where it began. Any other distance is a visible
  jump every period. So the caller lays two identical copies side by side,
  each `FlexShrink(0)` (Compose measures a pinned child unbounded and reports
  its real width; SwiftUI floors it at its base size), inside an
  `Overflow("hidden")` box, with any gap carried as leading padding inside
  each copy so the seam is the same gap as between items.
- **It replaces `Translate` on its node**, on every target. CSS cannot add
  the two without a custom property per axis in the keyframes (an animation
  on `translate` overrides the inline declaration), so the rule is stated and
  the natives, which could add them, do the same.
- **Linear, always**, for Spin's reason.
- **Reduced motion: it stops**, the opposite of Spin: sliding content is what
  the setting exists to remove, and a tape's motion is a convenience rather
  than its message. Read live on every target.
- `Spin`'s "Why a spin and not a general looping transition" section now
  says Glide is the one other loop, narrow for the same reason, and that a
  pulse or shimmer (cycles that close only by reversing) would still need the
  general form.

### Per target

| target | glide | reduced motion |
|---|---|---|
| htmlout / WASM | `animation: grmob-glide <ms>ms linear infinite [reverse]` after the spin entry; `core.GlideKeyframes` on the individual `translate` property, x through `--grmob-inline`, so a glide also brings `core.TranslateDirectionCSS`; no inline `translate` written while gliding | the same constant declares `grmob-glide` a second time under `prefers-reduced-motion: reduce` with both ends at rest; of two `@keyframes` with one name the later wins while its media query matches |
| Compose | `GlideElement`/`GlideNode` in `GrMobStyle.kt`, a `LayoutModifierNode` in the translation's place (outside both rotations), `placeWithLayer { translationX = sign * fraction * width }` with the fraction read only in the layer block; the sign flips under RTL since a layer's translationX is physical; the fraction is the frame time modulo the period with **no per-node start**, so two copies attached a frame apart still agree | the loop reads `coroutineContext[MotionDurationScale]` (which the window recomposer installs from the animator duration scale) each frame and holds at rest while it is 0; absent, falls back to 1 |
| SwiftUI | `GrMobSpin` became `GrMobLoop(spinMs:glideMs:)`: one `TimelineView(.animation)` paused when `spinMs == 0 && !glides`, applying `rotationEffect` and then `GrMobTranslate(x: GrMobShift(points: 0, fraction: glideFraction(at:)))`; `GrMobMotion` gained `glideMs` and passes `.zero` for the fixed shift while gliding | `@Environment(\.accessibilityReduceMotion)` holds the fraction at 0 and pauses the timeline if nothing spins; the spin keeps turning |

`htmlout` got a `loopEntry(name, periodMs)` helper shared by Spin and Glide,
and the WASM runtime the same shape inline. `treeMotion` gained `glides`, and
`translatesX` is true for a glide too.

## The widget: `comps.StockTicker{Glide: ms}`

- `Glide` is milliseconds per pass of the whole tape; zero (the default) is
  the stepping `Scroll`. A pass takes the same time however many quotes, so
  the caller scales it with the count.
- Shape: a clipping `HBox` (`Overflow("hidden")`, the caller's `Style` lands
  here) holding two `HBox` copies keyed `tape` and `tape:copy`, each
  `Glide`, `FlexShrink(0)`, `Gap(LG)`, `PaddingLeft(LG)` (leading on every
  target, so the seam between copies is one `LG`). The first copy carries
  the label and the list/group role; the second is `AccessibilityHidden`,
  its quotes have no role or spoken label (a listitem inside a hidden node
  is a contradiction the audit would name) but keep `OnClick`, since the
  reader sees that copy half of the time.
- `Render` was split: `tapeRole()`, `keyedQuotes(t, decorative)` and
  `quote(t, q, decorative)` serve both forms; the stepping `Scroll` is
  unchanged in output.
- The paused glide is the stepping tape: Glide zero is the Scroll at Start,
  which can be panned. That fell out of the lesson test (a paused glide
  rendered the Scroll) and was kept as the better behaviour.

## Lesson 4.42

A third checkbox, "Glide the tape". Gliding, `Start` is held at 0 (a step
would reorder the quotes under the glide) while the tick still wobbles the
prices; the period is `3000 * len(quotes)` = 15 s; Pause sets Glide to 0.
Summary, two prose paragraphs and a key point updated. The test toggles
checkbox 2, expects the swipeable Scroll while paused, then a `Row` copy with
`Glide == 15000` once unpaused. `nodeStyle` in `app_test.go` mirrors `Glide`.

## Registration and docs

- `docs/concepts/styling-and-theming.md`: `Glide` in the everywhere row of
  the support table, and a `## Glide` section after Spin with both mapping
  tables.
- `docs/components.md`: StockTicker's "Controlled, no hooks" paragraph no
  longer says core has no looping translate; an "Or it glides" paragraph.
- `docs/api/` regenerated (`go run ./internal/apidoc/gen`).
- `ai_docs/SKILL.md`: a "Motion" subsection under Styling and theming
  (Transition, the paint transforms, Spin, Glide and its two-copies rule,
  reduced motion, the pause obligation). `ai_docs/SKILL-component.md` §7: the
  "a `core.Style` field" row is now the full checklist of sites, with Glide
  as the template and the compile-only native commands. The installed copies
  under `~/.claude/skills/` were rebuilt from the sources, keeping their
  11-line pointer headers.

## Tests added

- `core/animation_test.go`: Glide written as given, merges like Spin,
  serializes under its key only when set, and the keyframes' shape (two
  declarations of the name, the reduced-motion twin).
- `htmlout/export_test.go`: the animation list, keyframes and direction rule;
  reverse and the suppressed inline translate (matched on translateDecl's
  spelling, since the keyframes also contain `translate:calc(`); a spin alone
  carries neither glide rule.
- `wasm/verify/rotate_test.mjs`: the same on the live DOM, three sheets once
  each, and a stopped glide letting a fixed translate through again.
  `totality_test.mjs`'s `FULL_STYLE` carries `Glide`. `jstable_test.go` pins
  `GLIDE_KEYFRAMES` to `core.GlideKeyframes` (the runtime literal is on one
  line for that).
- `mobile/verify/rotate_test.go`: `TestBothNativesParseAndApplyGlide` pins
  parse, placement before the rotation, the layer read, the shared clock, the
  RTL sign, the `MotionDurationScale` read, the Swift `GeometryEffect`, the
  zeroed fixed shift and the reduce-motion guards. The Spin and Translate
  pins were updated for `GrMobLoop`.
- `comps/stock_ticker_test.go`: the glide shape, taps on both copies, Start
  and Style under glide, and Glide zero being the Scroll.

## Verification

- `gofmt -l .` empty; `go vet` clean on every touched package;
  `go test ./...` green; the whole `wasm/verify` Node suite green.
- Kotlin compiles: `cd android && ANDROID_HOME=$HOME/Library/Android/sdk
  ./gradlew compileDebugKotlin --offline -q` (exit 0, no output). Swift
  typechecks: `cd ios/verify && swiftc -typecheck -target
  arm64-apple-macos14.0 ../GrMob/Runtime/*.swift` (exit 0, no warnings).
  Lesson: in zsh a piped command's own exit is `${pipestatus[1]}`; the first
  run masked both exits behind `tail`'s.
- Not seen moving anywhere yet: no device, simulator or browser run. Whether
  the window recomposer installs `MotionDurationScale` in a modifier node's
  coroutine context is reasoned from Compose's source. Both joined N-121.

## Next

Closed: N-120. Declined: None. Raised: None.
Deferred: None. Promoted: None. Moved: None.
Updated: N-121 (the glide checks appended). Full list: `ai_docs/todo/next-list.md`.
