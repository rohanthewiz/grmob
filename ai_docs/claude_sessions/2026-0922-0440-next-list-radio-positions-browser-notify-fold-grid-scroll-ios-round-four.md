# Radio positions, browser notifications and folds, a grid that scrolls, iOS round four

Session: `fe83f68b-f5c9-4da4-a6ec-5950284ed238`

## Ask

Do every Next-list item that needs neither hardware nor a user decision.
Emulator, simulator and headless Chrome runs count as "no hardware".

## N-077: TalkBack's wrong radio positions

### Diagnosis

Compose numbers a `selectableGroup`'s members itself
(`setCollectionItemInfo` in `CollectionInfo.android.kt`). An item's index is
the number of selectable siblings with a smaller `layoutNode.placeOrder`, and
placeOrder restarts at zero in every layout parent.

ColorSwatchPicker is a grid of Rows, so each Row was counted on its own. Every
heard number was one plus the count of lower places across both rows: orange
"3" because blue and purple are both at place 0, and so on.

Compose 1.10 has a second defect. `setCollectionItemInfo` sets a stated
`collectionItemInfo` and then falls through: when the parent is a
`selectableGroup`, it derives an index anyway and overwrites the stated one.
The first fix (stated info alone) was heard unchanged for that reason.
SelectableGroup has no other consumer in Compose.

### Fix

- **Renderer.kt:**
  - `LocalGrMobRadioPositions` and `RadioPositions`.
  - `radioPositions(group)` walks the Go tree in document order. It skips
    display none and AccessibilityHidden, and stops at a nested radiogroup.
  - `RenderNode` gives the group `collectionInfo(n, 1)` and each radio
    `collectionItemInfo(rowIndex = place)`. The walk sits behind
    `derivedStateOf`.
  - The provider count comment now says seven locals.
- **GrMobStyle.kt:** the `"radiogroup"` arm is `{}`, with the reason. The
  `selectableGroup` import was dropped.
- **Docs:** `core/role.go` (table and prose), `comps/radio_group.go`,
  `docs/components.md`. The API docs were regenerated.
- **Tests:** new `mobile/verify/radio_position_test.go`. It pins the walk,
  both semantics and the absence of `selectableGroup()` in `grMobRole`. The
  walk is read with `valuesIn`, because the nested `fun walk` cuts a
  declaration read.
- **Heard on the emulator:**
  - 5.9: orange 2 … red 8 of 8, and "Selected, blue, Radio button, 1 of 8.
    In list. 8 items".
  - 4.16 (reached by stepping Compose's focus with TalkBack off, then
    Shift+Tab under TalkBack): "Express … 2 of 3. In list. 3 items",
    "Standard … 1 of 3".
- The go-file tally sentences in `repowalks_test.go` and `timings_test.go`
  went 670 → 671.

## N-014: browser check 22, the scheduled notification

`Browser.grantPermissions` works from a page session.

The check posts through the real runtime on the plain page. The browser's
`Notification` is subclassed only to hear its "show" event, and a
`GrMobWASM.HostEvent` stand-in records `notification_swept`. The posts:

| Post | Prefix | Due |
|---|---|---|
| D | `grmob.alarm.` | immediate |
| A | `grmob.alarm.` | +1000ms |
| C | `other.` | +2500ms |
| B | `grmob.alarm.` | +2800ms |

A sweep of `grmob.alarm.` runs at ~1800ms, then a read at ~3800ms and a second
sweep. What must hold:

- D shows at once and A in [1000, 1800).
- B is never constructed.
- C shows.
- The sweeps answer `[{r1, [A]}, {r2, []}]`.

The first version had B at +60s. It passed with the sweep's `clearTimer`
removed, because B could never fire inside the check. B is now due between
the sweep and the read, and the mutant fails with both messages.

The header's tally grew a sixth kind. `tallyByKind` in
`checknumbering_test.go` takes six groups, and the number words go to
"twenty-three".

## N-029: browser check 23, the emulated fold, and a runtime gap

`Emulation.setDeviceMetricsOverride` with `displayFeature` splits
`window.viewport.segments`. `setDisplayFeaturesOverride` did not, in Chrome
152. `setDevicePostureOverride` fires `devicePosture` change.

**Found:** a segment change at an unchanged window size fires no `resize`. The
fold reached Go only on the next posture change. The runtime (`windowMetrics`)
now also reports on the change of `(horizontal|vertical)-viewport-segments: 2`.
Probed: those media queries flip on appear, swap and clear. A hinge moving at
the same count and size flips neither, and a physical hinge moves only with a
resize.

**Check 23** runs on the live build at 800 × 600, routed to 4.21. It steps
through five states:

| Step | Posture | Fold | TwoPane split |
|---|---|---|---|
| none | normal | none | no |
| vertical@390 mask 20 | normal | flat, vertical at x 390, separating | yes: 390px first pane, 20px seam |
| + folded | book | half_opened, vertical at x 390, separating | yes |
| horizontal@300 | tabletop | half_opened, horizontal at y 300, separating | no (4.21 sets IgnoreHorizontalFold) |
| gone | normal | none | no |

Mutation-tested by removing the listeners: the vertical and tabletop steps
fail. It is counted as a layout claim ("eight about layout"). The SKIP line
now names checks 15 to 20 and 23.

## N-072: the grid was not a scroll on either native

- `EditableGrid` put the grid in `core.Box(core.Horizontal())`. Only a browser
  reads `overflow: auto`, and neither native reads Overflow beyond "hidden".
- On the iOS simulator the 4.37 page grew to 490pt on a 402pt screen, and every
  paragraph was cut at both edges. On the emulator the grid was squeezed, with
  Amount clipped.
- Now `core.Scroll(core.Horizontal(), …)`, with a comment and the structure
  diagram updated.
- **Compose follow-on:** in the strip, the grid Column (a grower) was measured
  with a minimum over an infinite maximum, and its rows under
  `LocalGrMobUnboundedWidth` lost their weights. No two rows lined up.
- `GrMobGrowStrip` now measures a grower that states a points MinWidth at one
  width, `max(base + share, MinWidth)`, and provides
  `LocalGrMobUnboundedWidth false` beneath it. `StripGrow` gained `floorDp`,
  and there is a new `pointsMinWidth`. The doc section is "A grower with a
  floor".
- New pin `TestComposeStripGrowerWithAFloorIsMeasuredAtAWidth`. The old pin in
  `TestComposeGrowChildrenKeepContentSizeWhereNothingIsBounded` was
  respelled to `val want = …` and `minWidth = want`.
- **Seen:**
  - Emulator: aligned, scrolls to Amount and Paid, and "-5" refused with a red
    border and the message.
  - Simulator: the page is back to screen width, and the grid scrolls.
  - Headless Chrome at 400px: a 290px scroller holds the 462px grid.

## Device-free passes on the emulator

- **5.9:**
  - "#2a7" leaves Value at #2A78D6 until return, then commits #22AA77.
  - An injected drag of Minimum past Maximum leaves both at $115.
  - TalkBack says "Passcode, 1 of 4 entered", then 2 and 3, per Enter on a
    pad key.
- **4.34:** after a tap with TalkBack off, TalkBack says "Selected, thumbs up,
  3 reactions, Button" and "Selected, party popper, 2 reactions, Button".
  - Under TalkBack, Enter pressed a different node from the one spoken (N-058).
  - The Poll was never reached.
- **4.37 (before the Scroll change):**
  - A tap opens the field with the keyboard shown, and return commits (Undo
    (1)) and drops the keyboard.
  - The ✕ discards "GroceriesXYZZ", leaving GroceriesXY and Undo (1).
  - Covering by the keyboard was unjudged: the emulator shows only its
    floating stylus toolbar.
- **4.36:** the Waveform's round caps are even, the radar labels and chips are
  placed, and the doji draws.
- **Emulator restored:** accessibility services null, `accessibility_enabled`
  0, Google TTS enabled, app locales `[]`.

## The iOS simulator: TutorialRoundFourUITests (new)

`ios/GrMobUITests/TutorialRoundFourUITests.swift` is new and needs
`xcodegen generate`. All five tests pass:

- **`testSwatchesHexShortFormAndTheRangeThatCannotCross`:**
  - The short form commits on return only.
  - The sliders are reached by position inside the "Price" slider (N-078), and
    after `adjust(0.6)` both values are equal.
- **`testGridEditRoundTripAndDiscard`:**
  - Cells are reached by coordinate off the "Budget" element's frame, because
    they are not in the tree (N-078).
  - The keyboard shows, return gives Undo (1), and the ✕ leaves Undo (1).
  - After return the keyboard stays up on iOS.
  - The row in EDIT draws its Category text about 3pt left.
- **`testReactionChipReportsItsSelection`:** "party popper, 2 reactions",
  isSelected.
- **`testRoundFourChartsDraw`** (screenshots):
  - The Waveform, doji and funnel are fine.
  - **The radar is drawn 62pt left of centre, and "Vision" is cut to "sion".**
  - Undiagnosed; recorded in N-070.
- **`testTreeViewUnderArabic`:** it needs `-AppleTextDirection YES
  -NSForceRightToLeftWritingDirection YES`; `-AppleLanguages (ar)` alone
  stayed LTR. The tree is mirrored, and the chevron still points right
  (N-068).

The iOS framework was rebuilt with `ios/build.sh ./examples/tutorial`, from
the repo root.

## Found, not fixed

- **N-078:** SwiftUI's `.combine` over every labelled container swallows its
  members.
  - RangeSlider is one Slider "Price" with the value "10%, 40%".
  - The grid is a static text "Budget", with its text cells absent.
  - The swatch radiogroup is a Button "Label colour", Selected.
  - Stepper, Drawer and MessageBubble rely on the combine, so this is an API
    decision.
- **N-079:** slider values are spoken as percentages on both natives, against
  RangeSlider's doc.

## Verification

- `go test ./...`: pass. The API docs were regenerated twice.
- `wasm/verify/run.sh`: pass, including checks 22 and 23 in Chrome, both
  mutation-tested.
- `android/verify/run.sh`: pass, lint 0 errors.
- `ios/verify/run.sh`: pass.
- `xcodebuild test -only-testing:GrMobUITests/TutorialRoundFourUITests`: TEST
  SUCCEEDED.
- The tutorial AAR and APK were rebuilt and installed on the emulator.
  `wasm/main.wasm` was rebuilt (untracked).

## Gotchas

- `adb shell input text '#2a7'`: the remote shell eats `#` as a comment. Quote
  the whole command: `adb shell "input text '#2a7'"`.
- A declaration read (`codeOf` / `valuesOf`) stops at a nested `fun`. Use
  `valuesIn` for a function with a local fun.
- In Compose, uiautomator's `selected=` stays false for the `selected`
  semantics. TalkBack still says "Selected".
- Scratch probes (`cdp.mjs`, `notify.mjs`, `fold*.mjs`, `grid.mjs`) are in the
  session scratchpad, not the repo. The memory notes `cdp-browser-emulation`
  and `ios-simulator-rtl-and-xcuitest` hold the recipes.

## Next

Closed: N-077, N-029, N-014. Declined: None. Raised: N-078, N-079.
Deferred: None. Promoted: None.
Updated: N-027, N-058, N-062, N-066, N-068, N-070, N-072. Full list:
`ai_docs/todo/next-list.md`.
