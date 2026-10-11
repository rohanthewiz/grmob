# Next list, part 2: N-043 caret burst, N-091 natives look, N-100 AlignSelf on the natives

Session: `1e5d7494-420f-4e59-bc6e-2acf4044d880`
**Date:** 2026-10-10 19:55 · **Branch:** master (5bb11ec → 99dc23e, plus this doc)

## Ask

The same standing ask as part 1
(`2026-1010-1919-next-list-n096-export-viewport-n092-blb-proxy-n021-ios-image-floor`):
- do everything in the Next list that does not need the user;
- use judgement on minor decisions;
- commit and push per item;
- wrap every two or three items.

This is the second wrap.

## Evaluated and left for the user

- **N-099 (SwiftUI and Inert).** Core's own doc declined `.disabled(true)`
  on purpose, since it is a different claim. Under Full Keyboard Access, the
  Drawer's `AccessibilityHidden` already keeps the panel out. What is left is
  iPadOS's own Tab engine, which only a real keyboard can show (N-004's
  limit). Not touched.
- **N-098 (the RTL chevron).** No fix exists within the current API. The
  options are now written into the item: a bidi-mirrored "›", a mirroring
  Canvas, or the new Text prop. Each changes the look or adds API, so it is
  the user's call.
- **N-024, N-009, N-047, N-003:** contingent, or a product decision (a host
  measurement API, AlarmKit and full-screen intents, a BOM bump, the next
  hook change).

## 1. N-043: a 12-key burst through the caret correction

- **Where the window is.** 2.3's capitalize-words field reaches both
  correction arms of `GrMobTextInputCoordinator.write`: the span holding the
  caret, and the span after it. The first key is the one Go rewrites around,
  and every key typed after it waits in the keyboard's queue during the write
  and the correction.
- **The stress run.** A scratch XCUITest typed "xyzabcdefghi" at the caret
  inside "Hello". It did three launches per seed ("Hello world" and "hello
  world"). All six read `Helloxyzabcdefghi World`.
- **The committed test.**
  `TutorialDevicePassUITests.testCapitalizedWordsKeepTheCaretWithTheTyping`
  now types the 12-key burst instead of 3 keys, and passes.
- **Not applied:** the `inputDelegate` candidate fix. There is nothing to
  fix on the simulator.
- N-043 moves to Validate. What remains is a real device: a hardware keyboard
  typed fast, and a composing IME.
- Commit `e23a8b1`.

## 2. N-091: StripeCheckout, BibleVerse and Discussion on both natives

Built the tutorial for the API 36 emulator (`android/build.sh` and
`gradlew installDebug`). On iOS, a scratch XCUITest screenshotted lessons
4.38, 4.39 and 4.40 and wrote their trees.

| part | Android | iOS |
|---|---|---|
| StripeCheckout summary, Pay, lock glyph | right | right |
| BibleVerse loaded and failed, Retry hugs | right | right |
| Discussion header labels ("Ben, reply to Ana, 1h") | right (content-desc) | right (StaticText label) |
| Like chips | right (emoji ❤️ on the filled chip) | right (SF heart) |
| **2px thread line** | **missing** | **missing** |
| **Link "Read on Blue Letter Bible" tap target** | **708px for ~487px of text** | **260pt for ~191pt** |

Both defects have one cause: no native renderer read `core.Style.AlignSelf`
(the styling reference listed it as web-only). Raised as N-100 and fixed
below.

The headers' inner Texts ("Ben", "1h") still show up as nodes of their own in
both trees. Whether a reader stops on them is unheard: raised as N-101.

The emulator was asleep at first (black screencaps). It needed
`KEYCODE_WAKEUP` and `wm dismiss-keyguard`. The first deep link after an
install landed on Contents; a second cold start opened the lesson.

## 3. N-100: AlignSelf on Compose and SwiftUI

### The rule (both natives)

- **CSS's `align-self`.** The child's own value overrides the container's
  AlignItems for that child, and "" defers to the container.
- **Self-placed children are fit-content.** A child at start, center or end
  is sized to its ideal width, capped at the line, as CSS sizes an
  unstretched item.
  - Without this, the align alone did not help: Link's Box stayed full width
    because its own Text stretched (`fillMaxWidth`, a flexible frame) and
    filled the whole offer.
  - Applied only to a child with a stated AlignSelf, so no existing tree
    moves.

### SwiftUI

- **Style:** `GrMobStyle.alignSelf`, parsed from `AlignSelf`.
- **The rule, in `GrMobFlexSolver`:**
  - `selfAlign`;
  - `selfStretches` (a child's explicit value never reads as "unset", so a
    Column's stretch default stays the container's);
  - `isSelfPlaced`.
- **`FlexChildren`:**
  - the fill frame follows `selfStretches`;
  - it passes the raw value as the layout value `GrMobFlexAlignSelf`.
- **`GrMobFlexLayout.placeSubviews`:**
  - per-child stretch and offset;
  - a self-placed child is proposed `fitCross`.
  - `sizeThatFits` measures self-placed children the same way, so the
    container cannot report a width it does not draw.
- **`GrMobWrapLayout`:** places a child in its line by its own value. A
  stretch places at the top, as the container's stretch does there.

### Compose

- **Style:** `GrMobStyle.alignSelf`.
- **Placement:** `rowSelfAlignment` and `columnSelfAlignment`. Each `when`
  has one arm per `core.AlignItemsValues()` entry, with "stretch" mapping to
  null because it is a fill.
- **`RowChildren`:** a self-stretch fills only a definite height, either
  the one `rowPinsHeight` measured or the Row's own Height. A FlowRow child
  never self-stretches.
- **`rowPinsHeight`:** pins the Row with `IntrinsicSize.Max` when a child
  stretches itself, guarded by `answersIntrinsicWidth` so a List or a
  vertical Scroll sibling cannot throw.
- **`ColumnChildren`:**
  - the fill follows `selfStretches`;
  - placement uses `Modifier.align`;
  - a self-placed container child gets `fitContentWidth` (max-content
    capped at the offer), behind rowChildWidth's guards (`hugsRowOffer`,
    `sizedByRow`).

### Checks

- `ios/verify/flex.swift`: cases for `selfAlign`, `selfStretches` (with the
  reasons named: Link, the thread line), `isSelfPlaced`, and an offset.
- `mobile/verify/alignself_test.go`:
  - both parsers read `AlignSelf`;
  - the Kotlin arms are held to `core.AlignItemsValues()`;
  - every binding is pinned (FlexChildren, the layout's proposal and
    offset, the wrap layout, RowChildren, ColumnChildren, `rowPinsHeight`
    with its guard, `fitContentWidth`, `fitCross`);
  - ios/verify calls the rule.
- `android/verify/run.sh` passes (it compiles Kotlin, runs lint and the JVM
  harness), and so do `ios/verify/run.sh` and `go test ./...`.
- **Seen after the fix:**
  - Thread lines on both levels of replies: Android, and the iOS simulator.
  - The link: Android [186,673] (487px, was 708); iOS 191pt (was 260).
  - 4.29's two links hug their text. 4.32's "Read more" hugs. 6.7's
    ActionSheet still sits at the bottom with a full-width card (Android).
- **iOS UI tests:** 17 pass. That is `TutorialNativeFloorsUITests` (6),
  `TutorialRoundFourUITests` (5), `TutorialZeroBasisAndGradientsUITests` (3),
  `TutorialFooterStripUITests` (2) and `TutorialScrollUITests` (1).

### Docs

- `core.Style.AlignSelf` and `core.AlignSelf` gained doc comments.
- StackAlign's note no longer calls the flex fields DOM-only.
- `docs/concepts/styling-and-theming.md` moves AlignSelf into the
  all-targets row, with a paragraph on what the natives do.
- `docs/concepts/views.md` updated, and the API pages regenerated.

### wasm/verify's file count

`go test ./...` failed on
`TestTheQuestionsOnTheSharedRepositoryParseAreTheOnesDecidedOn`, which holds
five sentences to the tracked Go file count. They said 694; the tree has 698
counting this commit's new test. It had gone stale in part 1's N-092 commit
(`blb/proxy*.go`), because only package tests were run there. The figures now
say 698.

Commit `99dc23e`.

## What went wrong

- **Part 1 left `go test ./...` red.** N-092 and N-021 were committed after
  running their packages' tests only. wasm/verify's file-count prose then
  failed until this part's commit.
- **The first AlignSelf build did not change the link on Android.** The
  align worked, but the Box's own stretched Text kept it full width. That is
  what led to the fit-content rule.
- **A tap by guessed coordinates missed** ("Failed" on 4.39). Reading the
  bounds out of a uiautomator dump fixed it.
- **The first N-043 scratch test broke in its own reset step** (select all
  and delete), not in the app. It was rewritten to relaunch per round.

## Files touched

- `ios/GrMobUITests/TutorialDevicePassUITests.swift` (N-043)
- `android/app/src/main/java/com/grmob/runtime/{GrMobStyle,Renderer}.kt`
- `ios/GrMob/Runtime/{GrMobFlex,GrMobStyle,Renderer}.swift`,
  `ios/verify/flex.swift`
- `mobile/verify/alignself_test.go`
- `core/style.go`, `core/style_props.go`, `docs/concepts/*.md`, `docs/api/*`
- `wasm/verify/{repowalks,timings}_test.go` (the count)
- `ai_docs/todo/next-list.md`, this doc

## Next

Closed: N-091, N-100. Declined: None. Raised: N-100, N-101. Deferred: None.
Promoted: None. Moved: N-043 → Validate. Updated: N-043, N-098. Full list:
`ai_docs/todo/next-list.md`.
