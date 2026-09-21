# Low-hanging fruit for `comps`, round four

**Status:** drafted 2026-09-21. **Phase 1 landed 2026-09-21** (J1–J3, lesson 4.34,
`examples/chat`). **Phase 2 landed 2026-09-21** (K1–K4, lesson 5.9), and it
broke this round's rule twice, on purpose: K4's spike found a caret bug in
the Android field and the look found a hole in the web runtime, and both
were fixed where they were (see K4). **Phase 3 landed 2026-09-21** (L1, L2,
lesson 4.35) with the rule kept: nothing under a renderer changed. Phases 4
to 6 are not started.

Rounds one to three (`comps-low-hanging-fruit.md`, `-2.md`, `-3.md`, Tiers
A–I) are complete. This round follows the same rule. Every item in Phases 1 to
5 is **pure composition over primitives that all four targets already
render.** No file under `htmlout/`, `wasm/`, `android/` or `ios/` is touched.
If an item turns out to need a renderer, it leaves its phase for Phase 6.

This round is organised as **phases**, where the earlier rounds had tiers. A
phase is a batch that lands together and shares one tutorial lesson. The items
keep the lettering (J, K, L, M), so a session doc can still say "K2".

| Phase | Theme | Items | Lesson |
|---|---|---|---|
| 1 | The chat family | J1 `TypingIndicator`, J2 `ReactionBar`, J3 `Poll` | one, chapter 4 |
| 2 | Inputs | K1 `NumberPad`, K2 `ColorSwatchPicker`, K3 `RangeSlider`, K4 `MaskedInput` | one, chapter 5 |
| 3 | Structure | L1 `TreeView`, L2 `Wizard` | one, chapter 4 |
| 4 | Charts on Canvas | M1 `CandlestickChart`, M2 `FunnelChart`, M3 `RadarChart`, M4 `Waveform` | one, chapter 6 |
| 5 | A spreadsheet-like grid | N1–N3 `EditableGrid` | one, chapter 4 |
| 6 | Renderer-gated | `SignaturePad`, `QRScanner`, `Confetti` | none; each gets a plan of its own |

The chapter for each lesson is a guess from where the round-three lessons
went (widgets in 4, inputs in 5). Check the chapter titles before appending.

## Where the candidates came from

Round three exhausted the Next list and the hand-built widgets in the
examples. This round's candidates came from a survey of `comps/` on
2026-09-21 (about 85 widgets), asking what an app built on GrMob would still
have to write by hand. Three families had visible gaps:

- **Chat** has `MessageBubble`, `MessageThread` and `InputRow`, and nothing
  for presence or reactions.
- **Inputs** has no on-screen keypad, no colour choice, no two-ended range
  and no formatted field.
- **Charts** has bar, line, area, scatter, donut, histogram, heatmap, gauge
  and sparkline. The scaffolding in `comps/chart.go` (`niceScale`,
  `cartesianFrame`, `legend`, `chartPalette`) makes another chart cheap.

Two premises were checked while drafting and turned out to be wrong. Both
changed the plan:

- **A chat `Composer` already exists.** It is `comps.InputRow`, and
  `examples/chat/main.go:189` uses it. See "Checked, and already there".
- **The digits-only keyboard is no longer blocked.** `core.Keyboard`
  (core/keyboard_kind.go) landed after round three's blocked list was
  written, and `PINInput` already uses `KeyboardDigits`. So `NumberPad` does
  not unblock anything. Its case is now narrower; see K1.

---

## Phase 1 — the chat family

Three small display widgets. Each is an hour or two, with one decision each.
`examples/chat` is the first consumer of J1 and J2.

### J1. `TypingIndicator`

Three dots that pulse one after another, with an optional "Alice is typing"
caption.

- Shape: `TypingIndicator{Who string, Label string, Caption bool, Style}`.
  - `Label` defaults to `Who + " is typing"`, or "Typing" when `Who` is
    empty. It is the spoken name either way.
  - `Caption` draws the label beside the dots. Off by default, as the dots
    usually sit where a bubble will appear.
- The dots are three round `Box`es in theirs-bubble colours (`Surface` with
  a `Border` hairline), so the indicator reads as a bubble about to arrive.
- The whole widget is one spoken stop with `RoleStatus`. The dots are hidden
  from accessibility.

**The decision: how the dots move.**

- (a) `hooks.UseIntervalWhile` steps a phase 0→1→2, and `core.Transition`
  eases each dot's opacity. This owns a hook, so the widget must render
  unconditionally. That is awkward, because a typing indicator is by nature
  conditional.
- (b) A `Visible bool` field. The widget always renders, the interval runs
  only while `Visible`, and hidden means a zero-height box. This keeps (a)'s
  animation and removes its hazard.
- (c) No animation: three static dots.

Leaning (b). `UseIntervalWhile` exists for exactly this, and `Countdown`'s
`ticking` is the precedent for an interval that runs only when needed.

**Known limit: Reduce Motion never reaches Go.** Each host drops the
`Transition` on its own side (core/animation.go:40, `ReducedMotionCSS` and
SwiftUI's `accessibilityReduceMotion`). The interval would still step the
phase, so under Reduce Motion the dots would blink instead of easing. Keep
the opacity swing small (1.0 to 0.4), so that the un-eased version is a
gentle change and not a flash. Look at it with the setting on before the
lesson is written.

**Concern:** none. Every field has a default.

**What the build changed from the sketch.** Three things.

- **The dots fade by background colour, not opacity.** `core.Style` has no
  opacity, and adding one is renderer work on three targets. Background
  colour is what `core.Transition` animates everywhere, so the phase picks
  the dark dot (`TextPrimary`) and the others rest at `ControlBorderColor()`.
  The sketch's "1.0 to 0.4" swing has no equivalent; the Reduce Motion claim
  now rests on the dot being 6pt, and is N-062.
- **Decision (b) as leaned, with `Display none` for hidden**, not a
  zero-height box. That is `Spinner.Hidden`'s answer, and it keeps the
  status region unannounced. `Visible`'s zero value is hidden, on purpose:
  the visible state costs render passes, so forgetting the field should
  draw nothing.
- **The bubble carries `MessageBubble`'s tail and radius**, which landed
  after the sketch (per-corner radius is no longer blocked), so the
  indicator is replaced by a bubble without a jump.

### J2. `ReactionBar`

Emoji chips with counts, under a message. A tap toggles the reader's own
reaction.

- Shape: `ReactionBar{Reactions []Reaction, OnToggle func(emoji string),
  Disabled, Style}` with `Reaction{Emoji string, Count int, Mine bool,
  Label string}`.
- It is a `ChipStrip` of `Chip`s: `Label` is `"👍 3"`, `Selected` is `Mine`.
- Each chip's `AccessibilityLabel` is spelled out: "thumbs up, 3 reactions,
  including yours". `Reaction.Label` supplies the emoji's name, because no
  host names an emoji reliably and Go has no table of them.
- A `Count` of zero hides the chip, unless `Mine` (which would be a caller's
  bug, and is drawn rather than hidden).

**The decision: who holds the counts.** The caller does. A reaction is
server state, and an optimistic local count would be a second source of
truth. This is `AudioPlayer`'s reasoning from G3, pointing the other way.
The widget is stateless, like `Stepper`.

**Not included:** an "add reaction" picker. That is an emoji grid in a
popover, and an anchored popover is on the blocked list. A caller can put
its own `+` chip in `ChipStrip.Children` and open a `Dialog`.

**Concern:** `ConcernReactionBarInert` for reactions with no `OnToggle` and
not `Disabled`.

**What the build changed from the sketch.** Three things.

- **"including yours" is not in the spoken name.** `Chip` already sends
  `Mine` as `core.AccessibilitySelected`, and its doc argues at length that
  a state belongs in the state channel and not in a name. Saying both would
  announce it twice.
- **`Trailing core.View` was added** as the slot for the caller's "+" chip.
  The sketch pointed at `ChipStrip.Children`, which the bar does not expose.
- **`GroupLabel` was added** ("Reactions"), so the strip is a named
  `RoleGroup` and can be localized. An empty bar is `Display none`, and the
  chips are `Keyed` by emoji.

### J3. `Poll`

A question, and options that turn into labelled result bars after a vote.

- Shape: `Poll{Question string, Options []PollOption, Voted int, OnVote
  func(int), ShowResults bool, Style}` with `PollOption{Label string, Votes
  int}`.
- `Voted` is the index of the reader's choice, or -1.
- Before a vote: a `RadioGroup`-like list of ghost buttons.
- After a vote (or with `ShowResults`): each option is its label, a
  `ProgressBar` of its share, and a percentage. The reader's choice is marked
  with a check and `PrimaryOnLight`.

**The decision: percentages that sum to 100.** Independent rounding gives
33 + 33 + 33 = 99. Use the largest-remainder method, in one small pure
function with its own table test. With zero total votes, every bar is empty
and the percentage column reads "0%".

**Settled by precedent:** the caller holds `Voted` and the counts, for J2's
reason.

**Concern:** `ConcernPollInert` for `Voted < 0` with no `OnVote`.

**What the build changed from the sketch.** Four things.

- **`Voted int` became `PollOption.Mine bool`.** An `int` defaults to 0, so a
  `Poll` written without the field would have opened already voted for its
  first option. A bool per option has the right zero and is `Reaction.Mine`'s
  shape. `ConcernPollInert` is now "asking, no `OnVote`, and neither
  `ShowResults` nor `Disabled`".
- **Outlined buttons, not ghost.** A column of ghost labels under a question
  reads as text. `Stepper` had recorded the same finding.
- **`Disabled` and `ChoiceLabel` were added**, and the total ("8 votes") is
  drawn under the results.
- **The headless render found two defects.** The results sat indented from
  the question (the theme's Column inset), and the question was in
  Subtitle's grey and read as disabled. Both are fixed and pinned in
  `TestPollResultsAreOneStopPerOption`.

---

## Phase 2 — inputs

Four items. K1 to K3 are a few hours each. K4 has a catch to settle before
anything is built, in the way round three's Tier I did.

### K1. `NumberPad`

A 3×4 grid of digit buttons, with a backspace and one configurable corner
key.

- Shape: `NumberPad{OnKey func(string), OnBackspace func(), Extra string,
  ExtraLabel string, Disabled, Style}`.
  - `Extra` is the bottom-left key: `"."`, `"00"`, `"+"`, or empty for a
    blank cell.
- Every key is a `Button` at a 1:1 aspect, at least 48px. Backspace is named
  "Delete".
- The pad holds no value. It reports keys, and the caller builds the string.
  This keeps it useful for a PIN, an amount and a dialler alike.

**Why it is still worth building, now that `core.Keyboard` exists.** The
system number pad covers "type digits into a field". It does not cover:

- a lock screen or payment screen, where the pad *is* the screen and no
  field has focus;
- a kiosk or a calculator, where the system keyboard must never appear;
- htmlout and desktop web, where `inputmode` does nothing.

So the first consumer is an example, not `PINInput`. `PINInput` keeps the
system pad, which gives it SMS autofill.

**The decision: haptics.** A light haptic per key, as `CopyButton` does per
tap. A keypad without one feels dead on a phone. It stays off when
`Disabled`.

**Proof:** a lock-screen demo in the lesson, made of `PINInput`-style dots
(display only) over a `NumberPad`.

**Concern:** `ConcernNumberPadInert` for no `OnKey`.

**What the build changed from the sketch.** Four things.

- **Keys are not 1:1.** core has no aspect ratio. Keys are equal shares at
  least 56pt tall, which is also what the system pads draw.
- **The blank corner is a key that is not painted** (`Disabled`, hidden,
  `core.Opacity(0)`), not an empty `Box`. The headless render found it: on
  the web a flex share is the zero basis plus the cell's own padding and
  border, so an empty box was narrower than a key and the zero sat off-centre
  under the eight. It is `core.Opacity`'s second consumer, and the first one
  seen on a device (the Android emulator).
- **`OnBackspace` nil disables that key alone**, and raises nothing: an
  append-only pad is legitimate.
- **`BackspaceLabel` and `Label` were added**, and the handler guards
  `Disabled` itself, for the tap that races the disabling patch.

### K2. `ColorSwatchPicker`

A grid of tappable colour swatches.

- Shape: `ColorSwatchPicker{Colors []Swatch, Value string, OnChange
  func(string), Columns int, Label string, AllowCustom bool, Style}` with
  `Swatch{Hex string, Name string}`.
- It is a `RoleRadioGroup` of `RoleRadio` boxes. The selected one gets a
  ring in `PrimaryOnLight` and a check whose ink is chosen by the swatch's
  luminance.
- `Name` is the spoken label. With no `Name` the hex is spoken, which is
  poor, so debug builds raise a concern for it.
- `AllowCustom` adds a hex `Input` below the grid. It commits on a valid
  `#rgb` or `#rrggbb` and ignores anything else.

**The decision: the default palette.** `Colors` empty means the theme's
`chartPalette`, which is already tuned for both modes. The alternative, a
hard-coded Material row, would be a second palette to maintain.

**What it is not:** a hue/saturation square. That needs per-point
hit-testing on Canvas, which is N-007.

**Concern:** `ConcernColorSwatchUnnamed`, `ConcernColorSwatchInert`.

**What the build changed from the sketch.** Five things.

- **The default palette is `ChartColors()` and not `chartPalette`**, whose
  second half is 60% tints of the first. A tint of a colour already on offer
  is not a second choice.
- **Default swatches are named from their hues** (`colorName`: eight hue
  bands, plus light/dark, grey, black and white), since the unnamed concern
  would otherwise fire on the widget's own defaults. The eight default chart
  colours come out as eight different names, pinned by a test.
- **The ring is a border every swatch carries**, transparent when
  unselected, so a selection moves nothing. The grid's padding cells carry
  the same padding and border, for K1's reason.
- **The short hex form commits on return only.** Every six-digit colour
  passes through a valid three-digit one on its way (`#7B2` on the way to
  `#7B2FF0`), and committing it handed the caller a colour nobody chose.
- **It holds a hook** (the half-typed hex), taken whether or not
  `AllowCustom` is set. `ConcernColorSwatchBadHex`, `CustomLabel` and
  `Disabled` were added, a colour listed twice is offered once, and the file
  joined `closedComposites` (it declares a radiogroup).

Noticed, not touched: on the web the group's arrows are Up/Down only, because
the runtime gives a composite one axis. N-067.

### K3. `RangeSlider`

A minimum and a maximum that clamp each other.

- Shape: `RangeSlider{Title, Min, Max, Step float64, Low, High float64,
  OnChange func(low, high float64), Format func(float64) string, Labels
  [2]string, Disabled, Style}`.
- It draws two `SliderRow`s, labelled "Minimum" and "Maximum" by default,
  under one title line that shows the formatted range ("$20 – $80").
- Moving Low past High pushes High along with it, and the reverse. The
  alternative (stopping at the other thumb) makes a range impossible to move
  as a whole.

**The decision: be honest that this is two sliders.** A single track with
two thumbs is a node type, and it is on the blocked list below. Two labelled
sliders are also the accessible form of the control on every platform:
VoiceOver and TalkBack adjust one value per stop. So the composition is not
a stopgap for screen-reader users, only for the eye.

**Settled by precedent:** the caller holds both values and the drag draft,
as with `SliderRow` (D-tier). The values are the caller's.

**Concern:** `ConcernRangeSliderInert`. Debug builds also flag `Low > High`.

**What the build changed from the sketch.** Two things.

- **An inverted pair is drawn the right way round**, and
  `ConcernRangeSliderInverted` reports it, so the screen is right while the
  caller's state gets fixed.
- **The range in words is hidden from accessibility.** The two sliders state
  the same two numbers as their values, and a third telling is noise.

As sketched otherwise. There is no drag draft to hold: `SliderRow` commits on
the drag's end, so `OnChange` fires once per drag.

### K4. `MaskedInput` — a catch to settle first

A field that formats as the reader types: `(###) ###-####`, `#### ####
#### ####`, `##/##`.

- Shape: `MaskedInput{Mask string, Value string, OnChange func(raw,
  formatted string), Placeholder, Keyboard core.KeyboardKind, Label, Style}`.
  - `#` is a digit, `A` a letter, `*` either. Anything else is a literal.
- The formatting is a pure function, `applyMask(mask, raw)`, in the manner of
  `TagInput.splitDraft`. It gets a table test.

**The catch: every formatted keystroke is a rewrite.** Typing "5" into
"(55" makes Go answer "(555) ", which differs from what the host sent. The
text-edit ledger (core/text_edit.go) treats that as a rewrite and bumps the
epoch. Two things follow, and neither is known yet:

1. **Where the caret lands after a rewrite.** At the end is fine while
   typing forwards. It is wrong for an edit in the middle, where the caret
   would jump to the end on every key.
2. **Whether typing at speed survives.** The ledger was built for a rewrite
   per tag, not per keystroke. Keystrokes typed against a stale epoch are
   dropped or replayed, and at one rewrite per key that may drop real input.

**Settle it with a spike, not by reading:** a throwaway lesson demo with
`applyMask` in an `OnChange`, typed at machine speed on the Android emulator
(`adb shell input text`) and in headless Chrome. The outcomes:

- Both fine → build it as sketched.
- The caret jumps mid-edit, and speed is fine → ship it with the limit
  documented, as `Link` documents its missing underline.
- Input is dropped → it leaves this plan. The fix is a caret position in the
  ledger protocol, which is renderer work on all three live targets.

**A fallback that is always safe:** format on blur only. It needs no
rewrite while focused. It is less pleasant, and it is a real option if the
spike fails.

**What the spike found, and what the build changed.** The spike's two
questions had the wrong premise, in a useful way.

- **The ledger had moved on.** The hosts no longer replay only an insertion
  at either end; they do a three-way merge (`internal/rebasefixture`). Speed
  was never the problem: ten and sixteen digits by `adb shell input text`
  arrived whole, three runs of three.
- **The caret was the problem, and only on Android.** Keys 1 2 3 4 5 6 typed
  *a second apart* read back `(234) 651`. The Android field kept the caret's
  raw offset across a rewrite, so after `1` → `(1` the caret sat between the
  bracket and the digit. The web (`writeFieldValue`) and iOS (`write`)
  already carried the caret across Go's change. This was a host-parity bug
  that no earlier rewrite could show, because every earlier rewrite changed
  text at or after the caret or kept the length.
- **So this phase touched a renderer**, against the round's rule, and the
  reasoning is: the alternative was to drop K4 over a twenty-line bug whose
  fix is the rule two hosts already follow. `rebasefixture.Carry` states the
  rule in Go, `carryCaret` is the Kotlin copy, and `android/verify` runs it
  against `CarryCases`. Lesson 2.3's UPPERCASE field was re-run mid-text on
  the emulator afterwards (`HELLOABCWORLD`).
- **The web had a hole of its own**, found by typing through real `input`
  events in headless Chrome: a key the handler refuses changes no state, so
  no patch arrives and the key stayed drawn (`(555) 123-4567x`). The natives
  catch this through the ledger. `dispatchFromElement` in the runtime now
  gives a text field Go's text back after an `onChange` that left it
  different; `wasm/verify/fieldvalue_test.mjs` pins it. `NumericInput` is
  left out (a `type="number"` value reads empty mid-parse).
- **Outcome: the second of the three**, in a narrower form than feared. The
  caret is wrong only for a key typed mid-text *at the very end of a group*,
  where the reflow's differing span includes the caret. Documented on the
  widget.
- **Literals are written late** (`555` draws `(555`), which the sketch did
  not consider. Written eagerly, backspace on `(555) ` formats straight
  back to itself.
- **`unmask` walks the text against the mask** instead of stripping
  non-slot characters, so `+1 (###) …` does not read its own literal `1` as
  data. `Value` is the raw value. `OnComplete`, `Hint`, `Disabled` and a
  mask-shaped default `Placeholder` were added;
  `ConcernMaskedInputNoSlots` and `ConcernMaskedInputInert` are the concerns.
- The pure functions are in `comps/mask.go`.

---

## Phase 3 — structure

Two larger widgets, about a day each. Each has a decision that shapes its
API.

### L1. `TreeView`

An indented, expandable hierarchy: a file browser, an outline, a category
picker.

- Shape: `TreeView{Nodes []TreeNode, Expanded map[string]bool, OnToggle
  func(id string), OnSelect func(id string), Selected string, Indent
  float64, Label string, Style}` with `TreeNode{ID, Label string, Leading
  core.View, Children []TreeNode}`.
- A branch row is a chevron plus the label, with `core.ExpandedWhen`. A leaf
  row is the label alone.
- The caller owns `Expanded`. A tree's open state is navigation state that
  an app wants to restore, which is D6's test pointing at the caller.
- Collapsed children are not rendered, so a large tree costs only what is
  open.

**The decision: which roles.** `core/role.go:308` considered `tree` /
`treeitem` and left it out on purpose: "a tree is a third pattern with its
own expansion state and its own keyboard contract, and nothing in this
repository has one." This widget would be that something. The options:

- (a) **Nested `RoleList`s of `RoleListItem`s with a level**, and a button
  per branch that carries the expanded state. Every target renders all of
  this today. A screen reader hears "Documents, collapsed, button, level 2".
  There is no arrow-key tree navigation on the web; Tab walks the branches.
- (b) **Add `RoleTree` and `RoleTreeItem` to core.** This is the correct
  ARIA pattern, and it brings a roving tabindex and Left/Right to
  collapse/expand in `wasm/`. That is renderer work, so under this plan's
  rule it would be a plan of its own.

Leaning (a) now, written so that (b) can replace the roles later without an
API change. On the two phones there is no difference: VoiceOver and TalkBack
walk by swipe (the same note in role.go says so for listbox).

**Check first:** that `listitem` plus a level is announced on Compose and
SwiftUI, or is at least harmless there.

**Concern:** `ConcernTreeViewDuplicateID`, as a toggled ID must be unique.
`ConcernTreeViewInert` for branches with no `OnToggle`.

**What the build changed from the sketch.** Six things.

- **Decision (a), as leaned.** Nested `RoleList`s of `RoleListItem`s with
  `core.AccessibilityNestingLevel`, and a button *inside* each item rather
  than the item being one (a list's child must be a listitem, and a listitem
  is not a control; `ListRow.NestingLevel`'s doc states the same rule). The
  branch button is comps' shared `disclosure` with `Heading: false`. The
  roles are not in the API, so (b) can replace them without one.
- **The "check first" was answered from the source, not a device.**
  `Style.AccessibilityNestingLevel`'s doc already records it: the web writes
  `aria-level`, and neither native has a depth property, so the level is
  inert there and the role alone goes out. Harmless by construction; what is
  *heard* is N-068.
- **A branch toggles and a leaf selects.** The sketch did not say what a tap
  on a branch does when both callbacks are set. One row is one target: a
  selectable branch would be two buttons with one name. The data can say it
  instead (a first child that stands for the branch).
- **`TreeNode.Branch bool` was added,** for an empty folder or one whose
  children load on first open. Without it such a node is a leaf for ever.
- **`Indent` is an `int`,** as core's paddings are, and it is spent as a
  spacer leading a `Row` and not as `core.PaddingLeft`: a Row lays out from
  the reading direction's start, so RTL indents from the right for free. A
  leaf keeps an empty 20pt chevron column so labels line up.
- **The chosen leaf states `core.CurrentTrue`,** not the selected state:
  `aria-selected` is not allowed on a button. Stated only when `OnSelect`
  makes the row a button. `ConcernTreeViewDuplicateID` walks shut branches
  too.

### L2. `Wizard`

A `StepIndicator`, the current step's body, and a Back / Next footer.

- Shape: `Wizard{Steps []WizardStep, Current int, OnChange func(int),
  OnFinish func(), NextLabel, BackLabel, FinishLabel string, Style}` with
  `WizardStep{Title string, Body core.View, CanAdvance bool, Optional
  bool}`.
- Next is disabled while `!CanAdvance`. On the last step it reads
  `FinishLabel` and calls `OnFinish`.
- Back is hidden on the first step, not disabled: there is nothing it could
  ever do there.
- Completed steps in the indicator are tappable (`StepIndicator.OnTap`), and
  later ones are not.

**The decision: where the footer goes.** The natural home is
`Screen.Footer`, pinned above the keyboard. But the pinned footer still has
open device checks (N-002: "Pinning: `Screen.Footer`, 100% layers in a
pinned ZStack"). So the wizard draws its buttons inline at the end of its
own column, and the doc shows how a caller lifts them into `Screen.Footer`
with the exported `Wizard.Footer()` view. This keeps the widget independent
of the unverified path.

**Settled by precedent:**
- The caller holds `Current`, as with `Tabs` and `StepIndicator`.
- Only the current step's `Body` is rendered. A body that owns hooks is
  therefore rendered conditionally, which breaks the hook rule. The doc
  comment must say this loudly, and point at the fix: hold the state above
  the wizard.
- Moving to a step moves focus to its heading, which is the
  focus-after-navigation rule from c1829c4.

**Concern:** `ConcernWizardInert` (no `OnChange`), `ConcernWizardNoSteps`.

**What the build changed from the sketch.** Five things.

- **Focus does not move to the step's heading.** The sketch called it
  settled by c1829c4, and that commit gave `core.Focus` to a Compose
  *Button*. No target focuses a Text, so a heading cannot take it without
  renderer work. Under this round's rule that part leaves (N-069). What
  stands in: a visible `RoleStatus` line under the indicator, "Step 2 of 3,
  optional", which announces a step change where live regions are, and the
  title as a level-2 heading. VoiceOver has no live region, so iOS is silent
  on a step change.
- **`CanAdvance` became `Blocked`.** A `WizardStep{Title, Body}` with
  `CanAdvance`'s zero value would be a wall by default; the same argument
  that made `PollOption.Mine` a bool.
- **`Optional` has a meaning now:** an Optional step that is Blocked keeps
  Next enabled and reads `SkipLabel` ("Skip"). Never on the last step, where
  the button submits.
- **`DetachFooter bool` was added** beside the exported `Footer()`. Without
  it a caller lifting the footer into `Screen.Footer` would draw it twice.
  `Finish` with a nil `OnFinish` is drawn disabled.
- **The body is `Keyed` by step index,** so a step change replaces the body
  and does not morph one form into the next with a field's focus and text
  carried across. `PositionLabel` was added for the status line's words.
  The widget holds no hook, so it can itself sit in a `core.IfElse` (the
  lesson does this).

---

## Phase 4 — charts on Canvas

Four charts over `comps/chart.go`. They share a `Series` vocabulary, the
theme palette, the legend, and the rule every chart here follows: the Canvas
is `RoleImg` with a generated summary as its label, since a hidden data
table is still N-010.

Ordered by how much of the scaffolding each reuses.

### M1. `CandlestickChart`

- Shape: `CandlestickChart{Candles []Candle, Labels []string, Height
  float64, UpColor, DownColor string, Format func(float64) string, Style}`
  with `Candle{Open, High, Low, Close float64}`.
- It reuses `cartesianFrame`, `niceScale` (not zero-based) and `bandLabels`
  whole. A candle is a `Line` wick and a `Rect` body.
- **The decision: the colours.** Up is the theme's `Success` and down is
  its `Error` by default. That is the Western convention, and East Asian markets reverse
  it, so both are fields.
- A doji (open equals close) draws a 1px body so that it stays visible.

### M2. `FunnelChart`

- Shape: `FunnelChart{Stages []FunnelStage, Height float64, ShowRates bool,
  Style}` with `FunnelStage{Label string, Value float64, Color string}`.
- Each stage is a centred trapezoid whose top width is its value and whose
  bottom width is the next stage's. Labels go in a column beside the Canvas,
  because Canvas has no text (N-007).
- `ShowRates` adds the step conversion ("62%") between stages.
- **The decision:** a stage larger than the one before it is drawn as it is,
  and is not clamped. Real funnels have them (re-entry), and hiding it would
  misreport the data. Debug builds note it.

### M3. `RadarChart`

- Shape: `RadarChart{Axes []string, Series []ChartSeries, Max float64, Rings
  int, Size float64, Filled bool, Style}`.
- The grid is concentric polygons and spokes. Each series is a closed
  `Polyline` with a translucent fill (`withAlpha`).
- **The catch:** axis labels sit around a circle, and Canvas has no text.
  They have to be `Text` layers in a `ZStack` over the Canvas. `Compass`
  places its cardinal letters with `core.StackAlign`, which gives nine
  positions and so covers four axes, not five or seven. Labels at arbitrary
  angles need `core.Left` / `core.Translate` (core/style_props.go:265, 382)
  with offsets from the same trigonometry. **Check first** that a
  percentage `Left` and `Top` on a ZStack layer resolve the same way on
  Compose and SwiftUI. If they do not, M3 ships with the labels in the
  legend, numbered around the rim, and the rim labels join N-007.
- Fewer than three axes is not a radar. `ConcernRadarChartTooFewAxes`.

### M4. `Waveform`

- Shape: `Waveform{Peaks []float64, Progress float64, Height float64,
  BarWidth, Gap float64, PlayedColor, RestColor string, Style}`.
  - `Peaks` are 0 to 1. `Progress` is 0 to 1.
- One `Rect` per bar, mirrored about the centre line, rounded by the bar
  width. Bars before `Progress` take `PlayedColor` (`Primary`).
- If there are more peaks than bars fit the width, the widget downsamples by
  taking the maximum of each bucket, so a transient is never averaged away.
  The width comes from the caller (`hooks.UseWindow`), as with `WeeksFor`.
- **The peaks come from the caller.** No host decodes audio, so a server or
  a build step supplies them. Without peaks there is no waveform, and the
  doc says so in its first paragraph.
- **The decision: display only.** Seeking by tapping the waveform needs the
  tap's x position, which no event carries (N-007 again). So this does not
  replace `AudioPlayer`'s slider. It is offered as
  `AudioPlayer.Waveform []float64`, drawn above the seek bar.

---

## Phase 5 — `EditableGrid`, a spreadsheet-like table

One widget, and the largest in this round: several days, where everything
above is hours. It is a phase of its own for that reason, and it lands in
three steps (N1 to N3) so that each step is usable when it stops.

It is still composition. The pieces exist: `RoleGrid` / `RoleGridCell` with
the WASM runtime's two-axis arrow keys (docs/platforms/wasm.md, "A grid"),
`core.List` for a windowed body, `core.Horizontal()` scroll, `core.Input`
with `InputWithSubmit`, `Keyboard`, `OnBlur`, and `core.Focus`.

**What it is, against `DataTable`.** `DataTable[T]` is a read-only view of
typed rows: it sorts, groups, pages, and a row is the tap target. A grid's
unit is the *cell*, and its point is editing. Bolting cell editing onto
`DataTable` would give one widget two selection models (row and cell) and two
role sets (`table` and `grid`). So this is a new widget that borrows
`DataTable`'s column sizing (`Weight`, `Width`, `Align`) and its keyed,
windowed body.

### The shape

```go
EditableGrid{
    Columns  []GridColumn
    Rows     [][]string            // the caller's data, as text
    Key      func(row int) string  // row identity across insert and delete
    OnChange func(row, col int, value string)
    ReadOnly func(row, col int) bool
    RowHeaders bool                // 1, 2, 3 … down the side
    Label    string                // the grid's spoken name
    Compact  bool
    Style, HeaderStyle, CellStyle
}
GridColumn{
    Title    string
    Kind     GridCellKind          // Text, Number, Bool, Choice
    Options  []string              // for Choice
    Width, Weight float64
    Align    core.JustifyContent
    Format   func(string) string   // display only: "1234.5" → "$1,234.50"
    Validate func(string) string   // "" is valid; otherwise the message
    ReadOnly bool
}
```

**The decision: cells are strings, and the grid is not generic.**
`DataTable[T]` is generic because it reads typed rows through accessors. An
editable cell needs a setter per column as well, and a pair of closures per
column is a heavy API for what a text field produces anyway, which is a
string. So `Rows` is `[][]string`, `Kind` chooses the editor and the
keyboard, and the caller parses. A typed adapter (`GridOf[T]` with
getter/setter pairs) can be layered on later without changing this widget.

### The central decision: one editor at a time

- (a) **Every cell is an `Input`.** This is the simplest tree. But a 50×10
  sheet is 500 native text fields, each with its own text-edit ledger, and
  Tab walks all 500. The grid role's arrow keys would also fight the caret's
  arrow keys in every cell.
- (b) **Every cell is a `RoleGridCell` box showing text, and only the active
  cell becomes an `Input`.** This is how a spreadsheet works: navigate, then
  edit. It costs one native field for the whole grid. The arrow keys belong
  to the grid until an edit begins, and to the caret until it ends.

(b). The states:

```
            tap / Enter / Space (the cell's onClick)
  ┌──────────┐ ─────────────────────────────▶ ┌──────────┐
  │ NAVIGATE │                                │   EDIT   │
  │ cell has │ ◀───────────────────────────── │ cell is  │
  │ focus    │   return key  → commit, move ↓ │ an Input │
  └──────────┘   blur        → commit, stay   └──────────┘
                 ✕ button    → discard draft
```

- The grid owns two pieces of state: the active cell, and the draft text.
  This is D6's test again. No application wants a half-typed cell, so the
  draft is the widget's, and `OnChange` fires once per commit and not per
  keystroke. The widget owns hooks and must render unconditionally.
- **Commit** is the return key (`InputWithSubmit`), which then moves the
  active cell down a row as a spreadsheet does, or a blur (`OnBlur`), which
  does not move it.
- **Cancel has no key.** Key events are invisible to Go (the `PINInput`
  finding), so Escape cannot discard a draft. The editing cell shows a small
  ✕ at its end that does. This is a real limit, and the doc says so.
- **`Validate` failing keeps the cell in EDIT** with the message under the
  grid in a `RoleAlert` line, and tints the cell's border with `Error`. The
  caller never receives an invalid value.
- After a commit, focus returns to the cell's box with `core.Focus`, so the
  arrow keys work again at once. **Check first** that `core.Focus` on a
  non-input box holds on Compose and SwiftUI. It was fixed for a `Button`
  in c1829c4, and N-002 still lists it as open on iOS.

### The structure, and the role rule it must obey

"A grid owns rows, and rows own cells" (core/role.go, the grid pair). So:

```
Column  (the widget; carries Style)
├─ Box core.Horizontal()        only when the columns overflow
│  └─ Box RoleGrid, Label
│     ├─ Row RoleRow            header: RoleColumnHeader cells
│     └─ List                   windowed body, keyed by Key(row)
│        └─ Row RoleRow
│           ├─ Box RoleGridCell (row header "7", when RowHeaders)
│           └─ Box RoleGridCell × n   ← or the one Input, in EDIT
└─ Text RoleAlert               the validation message; outside the grid
```

- **Check first:** whether the WASM member walk finds `gridcell`s through
  the `List`'s own wrapper element, and whether a `columnheader` row inside
  a `grid` passes the a11y audit. `Calendar` is the only grid so far, and it
  has neither a `List` nor a header row inside it. If the audit refuses the
  header, it goes outside the grid container, hidden from accessibility, and
  each cell names its column in its own label, as `Calendar` does for its
  weekday captions.
- A cell's spoken label is "*column title*, row *n*, *value*", and
  "read only" is appended where it applies. The natives have no grid
  vocabulary and announce a gridcell as a button (role.go says so), so the
  label has to carry the position.
- `Bool` cells are a checkbox glyph that toggles on tap and never enter
  EDIT. `Choice` cells are a `core.Select` (core/input.go:280), which is the
  platform's own picker. Only `Text` and `Number`
  use the state machine above. `Number` asks for `KeyboardDecimal` and is
  right-aligned by default.

### The steps

- **N1. Navigate and edit text.** The structure, the state machine, `Text`
  and `Number` kinds, `ReadOnly`, the header row and `RowHeaders`. This is a
  usable widget on its own.
- **N2. Kinds and validation.** `Bool` and `Choice`, `Format`, `Validate`
  and its alert line.
- **N3. Rows.** `OnInsertRow func(after int)` and `OnDeleteRow func(row
  int)`, offered through a row-header `Menu` ("Insert below", "Delete"). The
  caller performs the change, since the rows are the caller's. This is where
  `Key` earns its place: without it a delete would re-pair every row below
  it by index, and an open editor would jump to the wrong row.

### What it is not, and why

- **No formulas.** A formula engine is a parser, a dependency graph and
  cycle detection. It is an application, not a widget. `Format` plus a
  caller that recomputes derived columns in its `OnChange` covers totals. A
  `Footer` row for sums is a fair N4 if someone asks.
- **No cell ranges, no fill handle, no column resize by drag.** All three
  need pointer-drag positions (Phase 6).
- **No frozen first column.** Sticky positioning on the horizontal axis is
  renderer work. `RowHeaders` scroll away with the rest. On a phone the
  honest advice is few columns, and the lesson says so.
- **No multi-cell paste.** `ReadClipboard` exists, but nothing tells Go that
  a paste happened, and a toolbar "Paste" button that overwrites a block of
  cells is too destructive to add without an undo. Leave it out.
- **No undo.** The caller holds the data and receives every commit, so an
  undo stack is one slice in the caller. The lesson shows it in ten lines.

### Cost, and the limit to state

Each visible cell is a node. `core.List` windows the rows, so the cost is
visible rows × columns, not the sheet. Columns are not windowed. State a
supported size in the doc (a first guess is 30 columns by any number of
rows) and measure a 30×1000 sheet on the Android emulator before writing the
number down. A commit re-renders the grid, and the reconciler should patch
one cell's text; assert that in the test with a patch count, as the
`TextGrid` tests do for a changed row.

**Concerns:** `ConcernEditableGridInert` (no `OnChange` and not wholly read
only), `ConcernEditableGridRagged` (a row whose length differs from
`Columns`), `ConcernEditableGridNoKey` (N3's callbacks set without `Key`),
`ConcernEditableGridChoiceNoOptions`.

**Proof:** a lesson with a small budget sheet: item, category (`Choice`),
amount (`Number`, formatted as currency, validated as not negative), paid
(`Bool`), and a total under it that the caller recomputes.

**Not verified until a device run:** the whole EDIT round trip under
TalkBack and VoiceOver, and the soft keyboard covering the active cell near
the bottom of the screen. Both go on the Next list when N1 lands.

---

## Phase 6 — renderer-gated, so each leaves for a plan of its own

These are recorded here so that the reasoning is not redone. Nothing in this
phase is built under this plan.

- **`SignaturePad`.** It needs a stream of pointer positions on a Canvas.
  `core.OnTouch` (core/behavioral_props.go:39) fires on pointer-down and
  carries no coordinates. The work is a drag event with x and y on all three
  live targets. That same event would unblock the hue square (K2), waveform
  seeking (M4), the grid's cell ranges and column resize (Phase 5), and
  swipe actions.
- **`QRScanner`.** `core.CameraView` captures a still to a file
  (`OnCapture(func(string))`) and decodes nothing. A scanner is a node type,
  or a prop on the camera, backed by MLKit or ZXing on Android, `AVCapture
  MetadataOutput` on iOS and `BarcodeDetector` on the web (which Safari
  lacks).
- **`Confetti` and Lottie-style effects.** Core animation is `Transition`
  and `Spin`. A particle burst needs either keyframes or a per-frame Canvas
  redraw, and a redraw per frame over the patch bridge is the wrong cost
  model. This is the least valuable of the three.

**If only one gets a plan, make it the pointer-drag event.** It is one piece
of renderer work with five consumers waiting.

## Checked, and already there

Written down so that they are not proposed again:

- **A chat `Composer`.** `comps.InputRow` is it: an input and a button wired
  to one submit, with the keyboard's return action. `examples/chat` uses it.
  Its doc already draws the line: "a composer that needs to restyle its
  input has outgrown this". A multi-line growing composer would be a new
  item, and nobody has asked for one.
- **`Countdown` and `Stopwatch`** (comps/timers.go).
- **A digits-only keyboard.** `core.Keyboard(core.KeyboardDigits)`. The
  entry in round three's "still blocked" list is stale.
- **A chat thread.** `MessageThread` landed after round three listed
  `MessageList` as blocked. `core.StartAtEnd` answered its scroll-to-end
  question. `StartAtEnd` is for a vertical `List`, so the year-wide
  `CalendarHeatmap` below stays blocked.

## Still blocked on a renderer, so it is not re-derived

Carried over from round three, less the two stale entries above:
- Carousel and pager dots (scroll offset)
- Pull-to-refresh and swipe actions (gestures)
- Tooltip and anchored popover (layout measurement)
- Native time wheel (node type)
- `RichTextView` with inline marks (inline span node)
- Per-corner radius, and so bubble tails
- Strikethrough and underline (text decoration)
- A year-wide `CalendarHeatmap` that opens on the newest week

New this round:
- **A two-thumb slider track** (node type; see K3).
- **A hue/saturation square, and tap-to-seek on a waveform** (pointer
  position; see Phase 6).
- **Escape to cancel a cell edit, cell ranges, a fill handle, column
  resize, a frozen column and multi-cell paste** (key events, pointer
  position, horizontal sticky, a paste event; see Phase 5).
- **`tree` / `treeitem` with arrow-key navigation** (role and keyboard
  contract; see L1).
- **Focus on a heading after an in-place navigation** (a focus command on
  a Text; see L2 and N-069).
- **An emoji picker popover** (anchored popover; see J2).

## Suggested order

| Order | Item | Why |
|---|---|---|
| 1 | ~~Phase 1 (J1–J3) + lesson~~ | smallest; `examples/chat` is waiting for J1 and J2. Landed as lesson 4.34; device checks are N-062 |
| 2 | ~~K4's spike~~ | an hour, and it decides whether K4 stays in Phase 2. It stayed, after an Android caret fix |
| 3 | ~~Phase 2 (K1–K3, and K4 if the spike allows) + lesson~~ | the form family's remaining gaps. Landed as lesson 5.9; device checks are N-065 and N-066 |
| 4 | ~~L1 `TreeView`~~ | the only hierarchical widget; its role decision is worth settling early. Settled as (a), nested lists |
| 5 | ~~L2 `Wizard` + the Phase 3 lesson~~ | builds on `StepIndicator`; lands last in its phase so N-002's footer checks have the most time. Landed as lesson 4.35; device checks are N-068, the heading focus is N-069 |
| 6 | Phase 4 (M1–M4) + lesson | independent of everything above; can be taken in any gap |
| 7 | Phase 5's two "check first" items | an hour; they decide the grid's structure before N1 is written |
| 8 | Phase 5 (N1, then N2, then N3) + lesson | the largest item; it goes last so the smaller phases are not held up behind it |
| 9 | Phase 6 | only a decision: which one, if any, gets a plan |

Phases 1, 2, 4 and 5 do not depend on each other. Only the lesson numbering
makes an order matter, as each lesson is appended to the end of its chapter.

## Definition of done, per widget

Unchanged from rounds one to three:

- `comps/<name>.go` with a doc comment that says what the widget settles and
  which theme roles it reads.
- `comps/<name>_test.go`. It renders under `core.SetDebugMode(true)`, asserts
  an empty concern list, and asserts the role and label. For interactive
  widgets it dispatches the callback and asserts exactly one state change.
  A widget with a value range also calls `core.AuditTree` directly (the G3
  lesson).
- A section in `docs/components.md`, and `docs/api/` regenerated with
  `go run ./internal/apidoc/gen`. A new file must be placed on a topic in
  `internal/apidoc/packages.go`.
- A tutorial lesson per phase, appended at the end of its chapter so
  deep-linked lesson numbers do not move. Update the lesson count in
  README.md, docs/tutorial-interactive.md,
  `examples/tutorial/screenshot_test.go` and
  `internal/shotclaims/shotclaims.go`, and re-take
  `docs/images/tutorial-contents.png`.
- A headless-Chrome render of the lesson, looked at. Rounds G1 and G3 each
  found a defect this way that no test had caught.
- Update the `wasm/verify` tracked-file census (`git add -A` first).
- No file under `htmlout/`, `wasm/`, `android/` or `ios/` changed.

Per phase:

- Strike the phase's rows in "Suggested order", and add a "What the build
  changed from the sketch" block to each item, as round three did.
- Anything that needs a device to verify goes on
  `ai_docs/todo/next-list.md` with a new ID. Round three's G1 left its toast
  and haptic check untracked; do not repeat that.
