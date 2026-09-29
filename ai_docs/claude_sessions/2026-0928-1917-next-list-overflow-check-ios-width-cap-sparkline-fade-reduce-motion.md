# Next list: the overflow check, an iOS width cap, Sparkline's fade, Reduce Motion seen

Session: `9690921f-40f9-4978-a4a5-fc4b38cb1dce`

## Ask

Go through `ai_docs/todo/next-list.md` and work every item that needs neither
the user's input nor a hardware device. Emulators and simulators count as
fair game; the Fold6, the Mi Max 3 and a real iPad do not.

## Triage

Workable: N-081 (sweep as a check), N-082 (screenshot), N-080 (natives
unseen), N-025 (Sparkline fade, kept open by the user), and the simulator- or
browser-reachable halves of N-062, N-064, N-066 and N-072.

Left alone, and why:

- API or user decisions: N-010, N-013, N-015, N-034, N-050, N-057, N-069,
  N-073, N-074, N-078, N-083, and N-022 (both limits are recorded as
  deliberate in `GrMobFlexZeroBasis`; changing them is a design call).
- Hardware or a person: N-002, N-004 to N-006, N-009, N-028, N-030, N-040,
  N-044, N-049, N-058.
- Blocked or contingent: N-018, N-021, N-024, N-047, N-051.
- Other: N-003 (needs a settings self-edit), N-027 (lesson content choice),
  N-031 (housekeeping for the user), N-043 (unobserved; no repro), N-045 (by
  design), N-061 (undone on purpose).

## N-082: the lesson screenshot

`wasm/shots/shoot.sh tutorial-lesson` retook `docs/images/tutorial-lesson.png`.
Lesson 1.1's card now shows "Following" whole.

## N-081: browser check 24, the overflow sweep

### Shape

- **`wasm/verify/overflow.mjs`** splits the sweep in two:
  - `OVERFLOW_READ_JS(root)` runs in the page and returns one flat record per
    element that draws a box (rect, offsetWidth/Height, overflow per axis,
    position, inline px width/height, inline max caps, own text, scroll and
    client widths, a skip reason, and the parent's index).
  - `overflowFindings` is a pure function over those records;
    `overflowLine` formats one finding.
- **Rules:** escape (an in-flow box outside a parent that shows overflow),
  clipped (outside the nearest ancestor that hides rather than scrolls),
  offscreen (sideways only, with no clipping ancestor), text (a word wider
  than a box that shows overflow), squeezed (drawn under a stated px size,
  unless a max cap is stated). A spill is reported once, at its outermost box.
- **Skips, with subtrees:** CodeEditor, rotated transforms, SVG internals, and
  inert subtrees. The last one was added for 4.18's shut Drawer panel, the
  only finding on today's tree, at all three widths.
- **`overflow_test.mjs`**: 14 tests with hand-built boxes, one firing case and
  one nearest-pass case per rule.
- **`gen.go`**: the transcript gained `lessons`, derived from
  `tutorial.Chapters` (`lessonIDs`), so a new lesson is swept with no edit.
- **`browser.mjs`**: check 24 loads the site page (the real
  `wasm/index.html`) at 1280 (split), 390 and 360, and routes lessons by hash.
  It reads each lesson once its tree has been quiet for 200ms, capped at 3s,
  and shows 12 findings per view.

### Bookkeeping the meta-tests asked for

- The header's tallies: "nine about layout", "Twenty-four claims".
- `checknumbering_test.go`'s number words gained "twenty-four".
- `run.sh`'s description, the SKIP line, and the OK tail.

### Mutation test

Each of `2026-0924-1211`'s fixes was reverted in turn with a scratch driver
(the same probe, 60s for all 240 views):

| Reverted | Found |
| --- | --- |
| 1.1's stats row (padding and wrap) | escape at all three views |
| Avatar's FlexShrink(0) | squeezed 34/36 at 360 on 4.3 |
| `FIELD_FLOOR_TYPES` | Send (4.31, 4.33) and Show (4.27) escape |
| the mount's overflow-wrap | text on 6.8 and 4.11 |
| the nested Scroll rule | 1.5 squeezed to 2px at all three views |

Reverting only 1.1's padding found nothing, because the row now wraps; the
whole fix had to go before the check fired.

## N-080: the natives, and an iOS defect

### Compose (the emulator)

Seen at 411dp and, via `wm density 480`, at 360dp (reset after):

- 4.11's StaticMap frame takes a 318dp and a 266dp column and keeps 180dp;
- 1.1's avatar is round, and the stats sit inside the card;
- 1.5's short Scroll is 474px (158dp) inside its 1dp border;
- 4.3's avatars are round beside wrapping subtitles.

### SwiftUI (iPhone 17 Pro simulator, 402pt)

- 1.5's viewport is 158pt, 1.1's avatar 40×40, 4.3's 36×36.
- **4.11's map was 320pt wide at x 47, ending at 367 against a column ending
  at 354.**
  - **Cause:** `grMobDimension` drew a points Width as a rigid
    `frame(width:)`. `GrMobMaxWidthLayout` narrowed the proposal and reported
    306, but a rigid frame ignores its proposal. A percentage cap has no
    length to fold in the way a points cap is (`fixedLimit`).
  - **Fix:** a new `relativeCap` argument. A points Width under a percentage
    MaxWidth is `frame(minWidth: 0, idealWidth: w, maxWidth: w)`, so it
    reports w with no proposal and min(proposal, w) otherwise.
    `GrMobMinContent` reads the declared Width off the tree, so min-content
    floors do not move. StaticMap is the only bundled node with that pair
    (MessageBubble's 80% has no points Width).
  - **Pins:** `TestSwiftCapsOutsideTheGrowFrame` gained two entries.
    `codeIn` masks string literals, so the pin stops before `""`.
  - **Tests:** three new ones in `TutorialNativeFloorsUITests`:
    `testAWidthUnderAPercentageCapTakesTheColumn` (seen failing on the old
    Swift: "ends at x 367.0, past the column's 354.0"),
    `testANestedScrollKeepsItsStatedHeight` and
    `testAnAvatarBesideWrappingTextKeepsItsSize`.
  - **Docs:** StaticMap's "Narrower than the request" doc now names the iOS
    mechanism.

The even crop under ContentModeFill is still unseen with a real picture,
because 4.11's provider host is gone.

## N-025: Sparkline's Area fades

`areaShape` now calls a new `areaFadeTo(area, color, values, scale, base)`.
Sparkline passes `chartView`, the bottom edge its area closes on, so it fades
4D under the highest point to 0A at the bottom, with the same flat-33
fallbacks. `TestSparklineAreaFadesTowardItsBottomEdge` covers it, and it was
seen in headless Chrome on 4.20.

## Measurements on open items

- **N-062 / N-064, TypingIndicator under Reduce Motion:**
  - **Web:** headless Chrome, per-frame computed opacity. About 20 values
    with a 0.3s transition normally; exactly 0.4 and 1.0 with a 0s transition
    under an emulated `prefers-reduced-motion: reduce`.
  - **SwiftUI:** a scratch XCUITest took about 75 screenshots in 6s, and the
    dots' centre pixels were read offline (`sips` to BMP, plain Python).
    17–18 luminance values normally; exactly 145 and 0 with
    `defaults write com.apple.Accessibility ReduceMotionEnabled -bool true`
    in the simulator (turned back off after). All three hosts now blink.
- **N-066:** ColorSwatchPicker's ring and check look right on SwiftUI.
- **N-072, keyboard over the active cell:**
  - Android: with `show_ime_with_hard_keyboard` 1 (restored to 0), tapping
    row 4's Item cell scrolled the page so the editor sits just above
    Gboard's suggestion strip.
  - iOS: row 4's editor opens, but the soft keyboard was off screen (the
    simulator has a hardware keyboard connected), so that half is still
    unjudged.
- **N-072's 150ms grace → N-084.** CDP's real mouse path on 4.37 typed
  "GroceriesZ", then pressed the ✕:

  | Press on the ✕ | Result |
  | --- | --- |
  | 60ms, 120ms | discarded, Undo (0) |
  | 300ms, 600ms | committed, Undo (1) |

  Pointer-down blurs the field, so the grace timer wins any press longer
  than 150ms. The fixes (a press that keeps focus, or a blur payload naming
  where focus went) are API changes; recorded, not made.

## Verification

- `go test ./...` passes. The API docs were regenerated
  (`go run ./internal/apidoc/gen`); `TestGeneratedPagesAreUpToDate` caught the
  StaticMap doc edit.
- `wasm/verify/run.sh` passes with check 24: 240 lesson views spill, clip and
  squeeze nothing.
- `mobile/verify` passes. `ios/verify/run.sh` passes.
- All 11 Tutorial* UI classes (51 tests) with the fix: 49 passed in the full
  run. `testAudioPlayer` ("Play did not become Pause") and
  `testMaskedInputFormatsAsItIsTyped` (six deletes left "(555) 12") both
  passed when rerun alone.
- The scratch probes were removed from `TutorialNativeFloorsUITests.swift`
  before commit.

## Gotchas

- **CDP:** `Page.navigate` to the current URL with only a new hash is a
  same-document navigation and fires no `loadEventFired`. The scratch driver
  hung on it; check 24 reloads for every view after the first.
- **Emulator:** it had gone away mid-session. `adb devices` was empty and
  `installDebug` failed with "No connected devices!" rather than a build
  error.
- **Emulator screen:** a black screencap meant the screen was off
  (`KEYCODE_WAKEUP`, `wm dismiss-keyguard`). `uiautomator dump` prints "null
  root node" and leaves the previous `/sdcard/ui.xml`, which reads as a stale
  screen.
- **Simulator:** it was shut down after an xcodebuild run. `simctl spawn
  booted` needs `simctl boot` first; pass `id=<udid>` to xcodebuild.
- **XCUITest:** `XCUIScreen` has no `scale` member.

## Left on the devices

- The emulator's `screen_off_timeout` is 1800000; its prior value was not
  read.
- The emulator's density, IME setting and the simulator's Reduce Motion are
  back as they were.

## Next

Closed: N-025, N-080, N-081, N-082. Declined: None. Raised: N-084.
Deferred: None. Promoted: None.
Updated: N-062, N-064, N-066, N-072. Full list: `ai_docs/todo/next-list.md`.
