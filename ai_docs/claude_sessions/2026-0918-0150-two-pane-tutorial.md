# Two-pane tutorial: the guide beside the phone

**Session:** e9c744c7-8931-438a-8dfc-f30c73566b0a
**Date:** 2026-09-18 01:50 (follows "round-three-eight-widgets-and-what-a-text-node-cannot-do")
**Branch:** master (f37325e → this commit)

## The ask

The web tutorial (`wasm/index.html`) ran the whole app inside a phone bezel
about 430px wide. It looked good, but the lessons were hard to read and follow
on a wide screen. The requested change: keep the phone layout as an option,
and make the default a two-pane layout, with the guide (instructions) on the
left and the phone on the right.

## The problem

Each lesson `Body` is a single view tree that mixes prose, code, `demoPanel`s
and key points. Hooks are positional, so the obvious ways to split it would
have broken things:

- **Rendering the body twice** (once per pane) doubles every ticker, location
  watch and permission check. It also leaves refs pointing at nodes that do
  not exist in one of the copies.
- **Rendering the pieces in a different order per mode** shifts hook slots
  when the reader flips the toggle mid-lesson.
- **Restructuring 60 lessons** into separate guide and demo parts was not
  worth the churn.
- **A CSS-only split** (grid with `display: contents`) cannot move demos into
  a separate phone.

## The design: split the rendered tree after render

The body renders exactly as before, in its own frame. `withLayout` then works
on the rendered `*core.Node` tree (`examples/tutorial/split.go`):

- `demoPanel` keys its root `"tryit/" + hint`. Every hint is a literal, so the
  key is stable and the reconciler diffs the panel in place, as it did before.
- `lessonRoute` keys its root `tutorial-lesson`. Navigator keeps that key as a
  suffix of `nav:frame:N`.
- `liftDemos` copies the tree on write. It moves each keyed panel to the phone
  and leaves a `demoPointer` ("TRY IT · hint · on the phone →") in its place.
  It also moves any stray `Modal` onto the phone. Input nodes are never
  written, which follows the frozen-Node contract.
- Hook slots were all claimed during render, and callback IDs travel with
  their nodes. Switching modes therefore only rearranges the tree, and demo
  state survives the switch (tested).

What the split shows depends on the top of the stack:

| top of stack | guide | phone |
|---|---|---|
| contents (depth 1) | contents | splash |
| lesson (key suffix) | lesson minus demos, with pointers | LIVE header + Scroll of demos + lifted modals |
| a demo's pushed screen (ch. 6) | note naming the lesson | the pushed screen, whole |

The mode arrives as a host event, `"layout"` with payload `{mode: "split"|"phone"}`.
It is held in session state `t.split`, which defaults to false. The natives,
the tests and the shots host never send the event, so they keep the phone
layout by construction.

The panes carry `core.AccessibilityID`s: `tutorial-split`, `tutorial-guide`,
`tutorial-phone`, `tutorial-phone-screen` and `tutorial-phone-content`. On the
web these become `id` attributes, and the natives never read them. All the
visual styling lives in the page's stylesheet, using the page's palette
tokens.

## Page (`wasm/index.html`)

- **Header switch:** "Guide + phone" / "Phone only", using `aria-pressed`.
  The choice is stored in localStorage (key `grmob-tutorial-layout`), with
  try/catch around every access.
- **Breakpoint:** `(min-width: 900px)`. Below it the page uses the phone
  layout whatever the choice, and the switch is hidden.
- **When the mode is sent:** after mount (the subscription is taken on first
  render), on each toggle click, and on media-query changes.
- **The chrome follows the tree:** `main:has(#tutorial-split)` switches off
  the outer bezel, so there is never a frame where the tree and the chrome
  disagree.
- **Guide pane:** fills whatever the phone leaves. Its text column
  (`Scroll`/`List` children) is capped at 820px and centred.
- **Phone:** 400px wide, up to 860px tall. I first used
  `margin-block: auto` + `stretch`, which collapsed the phone to its header
  height. It is now `height: 100%` + `align-self: center`.
- **Modals:** `#tutorial-phone-screen` has `transform: translateZ(0)`, so a
  Modal is contained in the phone.
- **Toasts:** the runtime hangs the toast layer on `.screen`. In the split it
  is pinned over the phone with `!important`, because its placement is inline
  style.

## Verification

- `split_test.go` covers:
  - the default is phone;
  - every lesson splits: no panel or Modal left in the guide, pointers equal
    demos, and Next still works;
  - contents show the splash;
  - a demo control on the phone still drives lesson state;
  - switching layouts keeps demo state;
  - a pushed screen from 6.1 runs on the phone;
  - `liftDemos` does not write to its input.
- Tests must render once before sending the event, because the subscription
  is taken on first render. The page does the same (it sends after mount).
- `go test ./...` passes.
- `wasm/verify`'s tracked-file-count check fired: 602 became 604 with the two
  new files. Five sentences in `repowalks_test.go` and `timings_test.go` were
  updated, as the test instructs.
- **Browser check** (`./build.sh`, `go run ./serve -addr :8093`):
  - The Chrome viewport was stuck at 852 CSS px because the page is zoomed,
    so the 900px breakpoint could not be crossed. I sent the event from the
    console instead.
  - Checked in split mode: the split renders; the checkbox on the phone
    recomposes the demo; the 6.4 modal opens inside the phone rect with none
    in the guide; the contents show the splash; switching back to phone
    restores the bezel.
  - Checked the toggle by unhiding it: it looks right and saves the choice.
    I cleared the saved choice afterwards.
- **Mishap:** while freeing port 8093 I killed a PID found by `lsof` without
  checking it first. It was Chrome's network-service helper, not the server,
  which had already exited. Chrome restarts that helper. Lesson: look at the
  PID's command before any kill.

## Files

- `examples/tutorial/split.go` (new): layout event, `withLayout`, `liftDemos`,
  panes, pointer, splash, pushed-screen note, `nodeView`.
- `examples/tutorial/split_test.go` (new).
- `examples/tutorial/app.go`: `split` state, `useLayoutMode`, Navigator
  wrapped in `withLayout`.
- `examples/tutorial/lesson_screen.go`: root key `lessonRootKey`.
- `examples/tutorial/widgets.go`: `demoPanel` keyed.
- `wasm/index.html`: switch, split CSS, layout JS.
- `wasm/verify/repowalks_test.go`, `wasm/verify/timings_test.go`: file count
  602 → 604.

## Next

- **See the split at a real ≥900px viewport.** This session only forced it
  from the console, because the zoomed tab never crossed the breakpoint.
  Check:
  - the switch appearing and disappearing on resize;
  - the 820px text cap;
  - toast placement over the phone (the CSS offset is computed and was never
    seen);
  - the light and dark page palettes.
- **The guide goes blank-ish while a chapter 6 pushed screen is up.** It
  shows a note, not the lesson, because Navigator renders only the top frame.
  Caching the last lesson guide was considered and skipped: its callback IDs
  would be stale.
- **Pointer to demo navigation.** Tapping a guide pointer could scroll the
  phone to its panel. That needs a scroll-to host capability.
- **`hooks.UseWindow` in split mode** reports the browser window, not the
  400px phone. The phone layout already had this issue; the foldables lesson
  (4.x) is the one to check.
- **A brief phone-layout frame at boot** before the layout patch arrives.
  Cosmetic.
- **A device pass.** Carried:
  - D6: the range band's hex fill and endpoint notch.
  - D7: TimePicker's menus.
  - Tier E: AvatarStack, BarItem badge, Lightbox, PasswordField swap.
  - Tier F: Heatmap and Histogram canvases.
  - Round three: all of section 5's "Not seen on a device".
- **Most comps widget tests do not run `core.AuditTree`** (carried). Add the
  audit to `renderDebug` or `rowHarness` and see what surfaces.
- **An inline span node in core** (carried; renderer work, a plan of its
  own). It unblocks `RichTextView`, an inline `Link`, and text decoration.
- **A per-corner radius in core** (carried; renderer). It unblocks the
  DateRangePicker notch fix and bubble tails.
- **htmlout has no global `box-sizing: border-box`** (carried).
  `Width(100%)` plus padding overflows in static exports only.
- **Breadcrumb's ghost buttons draw a faint frame in static exports**
  (carried). This is a Button-level decision.
- **CalendarHeatmap ties go to the earliest day** (carried). Undocumented
  beyond the code.
- **A fourth low-hanging-fruit round** (carried). No candidates gathered.
- *Non-goal, declined:* a two-pane layout on the natives. The split is a web
  host choice by design.
- *Non-goal, declined:* restructuring lesson bodies into separate guide and
  demo halves. The tree surgery makes that unnecessary.
- *Non-goal, declined:* a year-wide scrolling `CalendarHeatmap` (carried).
- *Non-goal, declined:* a `MessageList` / thread widget until a host can
  report or accept a scroll offset (carried).
- *Non-goal, declined:* a caption-flip "Copied ✓" on CopyButton (carried).
- *Non-goal, declined:* a separate `Alert` widget. Banner is it (carried).
- *Non-goal, declined:* Heatmap as a continuous gradient (carried).
- *Non-goal, declined:* a Sequential ramp interpolated from `Primary`
  (carried).
- *Non-goal, declined:* a native time wheel, and a sheet or Done on
  TimePicker (carried).
