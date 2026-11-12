# Next list

The project's open follow-ups, kept in one place. Sessions edit this file in
place (see `/sess-save`) instead of copying a list forward from doc to doc, so
an item that leaves Open or Validate without a line in Closed or Non-goals
shows up as a deletion in git history. Seeded 2026-09-19 by
`/next-list seed` from the `## Next` of `2026-0919-1254-fold6-talkback-hid-harness-focus-after-navigation-inert-named-controls`,
with each item's `raised` traced back through all session docs.

## Conventions

- **IDs** (`N-001`…) are permanent and never reused. The old per-doc numbers
  are kept as `(was #n)` for the seed only.
- **`raised`** is the session-doc stem the item first appeared in, traced past
  any window.
- **Age** is computed (session docs since `raised`), never stored.
- **Value** is the payoff, not the effort:
  - `high`: worked around today, or a second consumer has arrived.
  - `medium`: blocks one named thing, or is a visible defect.
  - `low`: nobody has hit the gap yet, or contingent on something that
    does not exist.
  - A qualifier in parentheses (`API decision`, `user's decision`, `blocked`,
    `delete?`, `non-goal?`) says what the item waits on.
- **Open** is the next build work (code, a defect, a decision). **Validate**
  is the next testing: run, look, hear, measure, or write or repair a test,
  with no product change planned unless the check finds a defect. An Open
  item whose fix has landed but is unseen moves to Validate. An item that
  mixes a fix with a check stays in Open until the fix lands.
- **Nothing leaves Open or Validate** without a line in Closed (done or
  merged) or Non-goals (declined, with the reason). A move between them is
  not a lapse.
- **Open and Validate stay in ID order.** New items are appended to
  whichever fits with **Next ID**, which is then bumped.
- `lapsed@<doc>` marks an item that once fell off a hand-carried list without
  being done; kept as history.
- In the seed, a non-goal's `declined` stem is where the item was first
  raised; the decision itself may have come in a later doc.

**Next ID:** N-146

## Open

- **N-003** · raised `2026-0913-2140-next-list-hooks-shortcuts-strips-bidi-and-floors` · value low
  **`.claude/settings.json` cannot be edited from a session**
  (`[Self-Modification]`). Worth an upstream report. Not re-checked since
  2026-09-15. (was #2; lapsed@0917-1659)
  - Re-rated 2026-10-10 from medium: nothing waits on a settings edit. The
    hook that hit it landed when the user pasted the entry (`bbf5fed`,
    2026-09-15). Claude Code now ships an `update-config` skill as its route
    for settings.json edits; whether the classifier still refuses through it
    is untried. Try that route the next time a hook changes, and file the
    report only if it is still refused.
- **N-009** · raised `2026-0916-1032-clocks-canvas-alarm` · value low
  **A Notify alarm is a banner, not a ringing screen.** The route is AlarmKit
  or full-screen intents. (was #15; lapsed@0917-1659)
- **N-015** · raised `2026-0916-1229-next-list-exact-alarms-boot-rearm-and-sweeps` · value low (user's decision)
  **Compose Rows don't shrink children in proportion.** Seen on the Fold6's
  cover screen. (was #18; lapsed@0917-1659)
  - Recommendation (2026-09-29, not acted on): leave Compose as it is.
    Proportional shrink needs a custom Row measure policy (the flex solver
    iOS already has), a large change for one cover-screen look; comps that
    must fit already pin with `FlexShrink(0)` or wrap.
- **N-024** · raised `2026-0916-1557-small-fixes-stroke-gradients-bar-values-alarm-groups` · value low
  **`barValueRoom` is still an estimate.** Exact needs host measurement of the
  plot. (was #30)
- **N-045** · raised `2026-0918-2002-next-list-thread-widget-caret-fixes-boot-frame-proof` · value low (by design)
  The web's thread place-keeping applies only when the List is its own scroll
  box. (was #56)
- **N-047** · raised `2026-0918-2203-android-device-pass-fold6-secure-field-weight` · value low (contingent)
  **The secure-field wrapper in `Renderer.kt` can go once the Compose BOM
  reaches foundation 1.10.** Check then:
  - whether Compose still reports read-only fields as editable;
  - whether it still clears the View's focus when the focused node leaves
    (2026-0919-1254 §2);
  - whether it still splits a merged node's label into a fake child
    (2026-0919-1254 §4). (was #59)
  - whether `setCollectionItemInfo` still overwrites a stated
    `collectionItemInfo` under a `selectableGroup` (N-077's second defect).
  - Re-checked 2026-09-25: BOM `2024.12.01` resolves foundation and ui to
    **1.7.6** (`gradlew :app:dependencies`, debugRuntimeClasspath), so the
    premise holds. N-077's closure and the comment at `Renderer.kt:349`
    call the overwrite a "Compose 1.10" defect, but it was heard on the app's
    1.7.6. The 1.10.0 jars in the Gradle cache are not on the classpath.
- **N-051** · raised `2026-0918-2310-mi-max-3-android-10-force-dark-theme-sweep` · value low
  **Why the decor-view force-dark flag stopped holding after an AndroidView
  attached is undiagnosed.** Moot for this app. (was #68)
- **N-098** · raised `2026-0922-0204-n076-logical-insets-hidden-fill-ios-carry-talkback-sweep` · value medium (API decision)
  **The disclosure chevron does not mirror under RTL.** A collapsed "▸"
  points right, against the reading direction, on all three hosts: seen on
  TreeView in headless Chrome and on the Compose emulator (2026-09-21), and
  on the iOS simulator (2026-09-22, `testTreeViewUnderArabic`). The glyph
  is a plain `core.Text` made by the shared `disclosure` helper
  (`comps/disclosure.go`), so Accordion, Discussion and the grouping
  widgets draw it too. A Text cannot mirror. The API decision is a
  mirror-under-RTL prop for Text, like `CanvasMirrorsRTL`. Split out of
  N-068 on 2026-10-10, where it sat inside a list of device checks.
  - Looked for a fix within the existing API (2026-10-10), so the decision
    has its options in front of it. None is free:
    - A `core.Text` cannot mirror, and "▸" (U+25B8) is not Bidi_Mirrored, so
      no host's text engine flips it.
    - A bidi-mirrored glyph does flip by itself under RTL on every host:
      "›" (U+203A) renders as "‹" in a right-to-left paragraph. But it is an
      angle quote, not a filled triangle, so every disclosure widget changes
      its look.
    - The chevron could be a small Canvas with `CanvasMirrorsRTL`. It needs
      ChevronStyle's font size turned into a box size, and it changes the
      node type that four renderers draw for every disclosure.
    - The prop the item names (mirror-under-RTL for Text) is the general
      answer, and new API.
- **N-099** · raised `2026-0919-1254-fold6-talkback-hid-harness-focus-after-navigation-inert-named-controls` · value low
  **SwiftUI does not read `Inert`.** Compose reads its keyboard half
  (2026-0919-1254 §5). On iOS a hardware keyboard can still Tab into a
  Drawer's shut panel. Documented on `core.Style.Inert` and in
  `comps/drawer.go`. Nobody has hit it on a device. Split out of N-002 on
  2026-10-10, where it sat as a check whose answer was already known. N-001
  declines composing the shut panel differently. This item is only the
  focus gate.
- **N-113** · raised `2026-1017-2211-gallery-widget` · value low
  **core's paint transforms stop at Rotate and Translate.** A 3-D card flip
  needs rotation about the vertical axis with perspective, and a real zoom
  needs a scale transform (plus pinch). `comps.Gallery` uses a crossfade or a
  rise for the flip, and a taller stage or a Lightbox for the zoom. Nobody
  has asked for more than that yet.
- **N-129** · raised `2026-1031-2336-n121-stock-ticker-natives` · value low
  **The reconciler has no move patch, so a keyed reorder is a replace of
  every slot whose key changed.** `reconcile/doc.go` says so ("What it does
  not do yet"); identity-based node IDs are named as the prerequisite.
  Found by N-121's "a step arrives as moves" check: on the web, after a
  Step of `comps.StockTicker` with the interval paused, none of the five
  quote elements was reused (a scratch CDP probe compared element identity
  before and after). The widget's doc, lesson 4.42 and `docs/components.md`
  claimed a step reorders the nodes on screen; all three now say a step
  never patches one quote's figures into another's slot and name the
  replace. Visually identical on every target. The cost is transient native
  state: on the web, focus on a quote would go with its element at each
  step; Compose's `key(child.key)` in RowChildren keeps the keyed
  composable, so TalkBack re-read a focused quote's new price at a tick
  without losing it (2026-10-31, emulator). Low until a consumer needs a
  reorder to keep focus or a scroll offset.
- **N-130** · raised `2026-1031-2336-n121-stock-ticker-natives` · value low
  **Compose centres a lone pinned child that overflows a bounded Row of
  mixed children.** Compose coerces a Placeable's width into the constraints
  the child was measured with and applies the surplus as a centring offset
  (ui 1.7.6 `Placeable.kt`, `recalculateWidthAndHeight` /
  `apparentToRealOffset`), so `pinMainAxis`'s unclamped report never reaches
  the Row: a pinned child wider than its offer is placed as the offer's width
  with its content shifted left by half the overflow, and the next sibling
  follows the offer, not the real width. Seen with `comps.StockTicker`'s two
  gliding copies on the API 36 emulator (each 2835px in an 834px clip, drawn
  over each other 417px apart, starting 1000px left of the clip). Fixed for a
  Row whose every child is pinned and none weighted
  (`overflowsAtContentWidth` in Renderer.kt measures such a Row unbounded
  and places the overflow from the leading edge, `placeRelative`), which is
  the glide's shape; pinned by `TestTheComposePinnedStripIsMeasuredUnboundedAndPlacedFromTheStart`.
  Left: a pinned child among unpinned siblings in a bounded Row (CSS would
  start-align it and push the siblings out; Compose centres it and lays the
  siblings over it). No bundled screen hits it: every pinned child in comps
  fits its Row. `internal/pinfixture`'s transcription of the Row loop sums
  real widths and is right only under an unbounded maximum; its header and
  `docs/platforms/native.md`'s census prose should say so when this is
  taken up (native.md has a one-paragraph correction already).
- **N-134** · raised `2026-1111-0120-reliability-performance-convenience-plan` · value medium
  **Android builds a new runtime, store and bridge in every
  `MainActivity.onCreate` (plan R5, R6, P5, R9m).** Rotation, a dark-mode
  switch and a fold all recreate. Pushes in the remount window go to the dead
  store, `lastEditSeq` restarts under Go's ledger, `setTimeZone` is rewritten
  (N-013 assumes once), and a deep link is replayed
  (`MainActivity.kt:90` is unguarded). Fix: a process-scoped owner; the
  Activity only calls `setContent`. Same wave: decode payloads off the main
  thread with `JsonReader` (P5), and split the patch applier from the
  Compose state holders so `android/verify` can replay the Go transcript
  (R9m).
  - When R9m lands, its replay should also cover `TreeStore.kt`'s drift
    handling from N-133 (`onDrift`, `resyncPending`, the root replace that
    clears it), which compiles and lints but no test executes: ios/verify's
    `checkResync` is the model (a faithful replay never asks; a node-sized
    hole asks once and the answer matches Go's final tree).
- **N-138** · raised `2026-1111-0120-reliability-performance-convenience-plan` · value medium
  **The engine's per-pass cost (plan P0-P3).** Every event renders the whole
  app twice (20 taps ran the root 40 times), and the diff is 10x slower than
  a generated `Style.Equal` with lazy child paths (975 µs against 97 µs at
  1,001 nodes). First commit the benchmarks and debug-mode pass statistics
  (P0; ROADMAP's "Benchmark diff/patch engine"). Then a generation counter
  so the pump skips a pass the dispatch already rendered, the web host
  dropping its redundant `RenderAgain` and `IsDirty` poll (P1), the
  generated equality (P2), and the registry purging in place (P3).
  - P0 landed 2026-11-11 (`2026-1111-1851-n140-n144-n138-grmobtest-useasync-shuffle-fake-clocks-p0`).
    The benchmarks: `BenchmarkDiffUnchanged` and `BenchmarkDiffOneChange`
    in reconcile (1,001 and 10,001 nodes; 1.06 ms and 1,150 allocs at
    1,001 unchanged on an M3), `BenchmarkManagerNoopPass` and
    `BenchmarkManagerTap` in render (1.40 ms and 1.42 ms at 250 rows, debug
    mode off). P1's pin is `TestATapRendersTheRootTwiceToday` (render): 20
    taps with a listener render the root 40 times; P1 changes its `perTap`
    to 1. Debug mode records a `core.PassStat` per pass (handler, build,
    diff, encode, patches, bytes) in `core.PassStats` (last 512), with
    `core.SetPassLogging` for a line per pass. P1, P2 and P3 remain.
- **N-139** · raised `2026-1111-0120-reliability-performance-convenience-plan` · value low
  **Host and library hot spots (plan P6-P9).** The one felt daily is P7:
  the dev watcher runs `go list -deps` on every 250 ms poll, about half a
  core idle and 0.6 s on every save. The rest: `applyStyle` writing all 48
  properties on create, six post-batch ancestor walks, `applySwipe` on every
  node (P6); `CodeEditor` re-tokenising, `DataTable` re-sorting, O(fields²)
  form error reads (P8); iOS min-content re-measurement and uncached,
  full-resolution images (P9).
- **N-141** · raised `2026-1111-0120-reliability-performance-convenience-plan` · value medium
  **CLI: `go tool grmob`, a one-ABI bind, and shell upgrades that keep the
  app's edits (plan C3-C5).** Register `cmd/grmob` as a `go.mod` tool in the
  scaffold; add `version`, `dev` and the missing doctor checks. A `-fast`
  flag that binds arm64 only and skips an unchanged bind. Sync the
  framework-owned runtime sources on every build, copy app-owned files once,
  keep a pristine copy for `grmob upgrade --diff`, stamp a content hash, and
  delete files the new shell dropped.
- **N-142** · raised `2026-1111-0120-reliability-performance-convenience-plan` · value low (user's decision)
  **Release builds, a web `dist`, dev conveniences and the larger bets (plan
  C6-C10, P4).** Signed AAB and iOS archive, with `targetSdk` raised from 34;
  `grmob web -dist` with hashed, precompressed assets; LAN URL and QR, panic
  overlay, state across a hot swap; concerns and a patch log visible on the
  device. Bets needing a design note first: a `RemoteBridge` for live reload
  on a real device, a hidden-scope signal so tickers in undrawn scopes
  pause, pull-to-refresh, `UsePersisted`, and the `core.Style` size
  experiment (1,024 bytes a node; keep only for a 30% gain). Which of these
  are wanted is the user's call.
- **N-143** · raised `2026-1111-0120-reliability-performance-convenience-plan` · value low
  **Housekeeping (plan Part D).** ~~`chat` (a 3.9 MB Mach-O at the repo root,
  21% of every module download) and `cmd/docs/docs` (9.6 MB) are tracked:
  untrack, ignore, and add a census check.~~ Publish tag notes as GitHub
  Releases and run apidiff in CI. Dead fields in `core/context.go`. Docs and
  ROADMAP drift (README's 84 lessons, `hooks/doc.go:22`, deep links and
  Packaging listed as planned). Whether `ai_docs/` (19% of the zip) stays in
  the module is the user's decision.
  - D1 landed 2026-11-11 (`2026-1111-1142-n137-d-e-lazy-keys-and-web-teardown`).
    Both binaries are untracked with `git rm --cached` (still on disk) and
    listed in `.gitignore`. Nothing consumed them: the docs image builds its
    server from source. The new `internal/trackedfiles` test fails on any
    tracked compiled program, by content (ELF, Mach-O in both byte orders,
    PE; not the fat header, which a Java class file shares). It named
    exactly these two before the untrack. History still carries their bytes
    unless rewritten, which is not proposed.
  - One drift line fixed 2026-11-11 (`2026-1111-1851-n140-n144-n138-grmobtest-useasync-shuffle-fake-clocks-p0`):
    `hooks/doc.go`'s dependency table said a dep-less `UseEffect` runs every
    pass (it runs once), and `ai_docs/SKILL.md`'s hooks list said the same;
    both corrected with N-140's UseAsync docs. The rest of D6 (README's 84
    lessons, `docs/index.md`'s five hooks, `build.sh`'s 40%,
    `wasm/index.html`'s 51) and D3, D5, D7 remain.
- **N-145** · raised `2026-1111-1851-n140-n144-n138-grmobtest-useasync-shuffle-fake-clocks-p0` · value low
  **Nothing compresses the 13.1 MB wasm module (plan P10d).** gzip gives
  2.98 MB and brotli 2.07 MB, and `serve` never sets `Content-Encoding`.
  The fix the plan names: precompress in `build.sh` and serve the `.br` or
  `.gz` when the request accepts it; what GitHub Pages does for `.wasm` was
  not checked. Split out of N-144 when its P10 landed, because N-144 is
  repairs to tests and CI only and this one changes what `serve` sends.
  Related: N-142's `grmob web -dist` with precompressed assets (C7).

## Validate

Items whose remaining work is purely testing: run, look, hear, measure, or
write or repair a test. No product change is planned unless a check finds a
defect, which then becomes its own Open item.

- **N-002** · raised `2026-0912-1821-tier-a-comps-landed` · value medium
  **Lessons on hardware, and checks still open.**
  - Done on Compose (2026-0919-1254): radio, StepIndicator and aria-current
    heard; AccessibilityHidden behind a Drawer heard; `core.Focus` on a Button
    and a keyboard reaching a shut panel fixed.
  - Screen readers: combobox active option, "pop-up" triggers, CodeEditor
    toolbar role on Compose, SearchableSelect on the natives.
  - Fixed but unheard (2026-0919-2146): the Stepper group's and the Drawer
    panel's utterances on Compose. The duplicate is gone from the
    accessibility tree, measured on the emulator; a group node is not
    reachable by Tab and TalkBack's reading-order keys ignore `adb input`, so
    hearing the line needs the Fold6's HID keyboard (Meta+Right).
  - iOS: sheet Dialog and ActionSheet filler, Drawer under Reduce Motion, RTL
    for Drawer and CodeEditor, the sideways editor, 4.6's list; `core.Focus`
    on a Button. Inert is not a check there: SwiftUI does not read it
    (`core.Style.Inert`'s doc), so an iPad keyboard reaching a shut panel is
    a known gap, split out as N-099 (2026-10-10).
  - iOS looks done 2026-10-10 on the 26.5 simulator, with a scratch XCUITest
    in LTR and in forced RTL.
    - The sheet Dialog (6.6) and the ActionSheet (6.7) draw as GrMobModal
      documents: a medium-detent sheet with the card at its top, and the
      filler zero-height.
    - RTL: the Drawer's panel comes from the right, with its title and rows
      mirrored. CodeEditor keeps its code left to right with the gutter on
      the left, and only the toolbar mirrors.
    - The sideways editor pans its long lines under a fixed gutter.
    - 4.6's banded list draws its month band and rows.
    - Drawer under Reduce Motion snaps (a simctl screen recording, one jump).
      But it also snapped without Reduce Motion, on both natives: N-106,
      fixed. It now slides with motion on and still snaps under the setting.
    `core.Focus` on a Button is still open. Tried 2026-10-10: XCUITest
    reports `hasFocus` false for every element on the iPhone simulator,
    before and after 4.18's drawer hands focus to its ✕ and back, so the
    simulator cannot answer it. It needs a real iPad keyboard (N-004's
    route).
  - Pinning: `Screen.Footer`, 100% layers in a pinned ZStack.
  - Android: predictive back, MaxWidth where it binds on a tablet, RTL capped
    child. All three done 2026-10-10 on the API 36 emulator.
    - Predictive back: a slow edge swipe from Contents shrinks the window to
      preview what is behind it. From a lesson, and with 4.18's drawer open,
      it shows only the back arrow and goes back inside the app.
    - MaxWidth where it binds, in landscape (914dp) rather than on a tablet:
      the 520px cap on 6.7's ActionSheet and 4.17's Menu held, but the panels
      sat at the window's start. That is N-107, now fixed: both are centred.
      4.9's DatePicker card (360) is centred and scrolls.
    - RTL capped child: 5.7's PIN row (320px cap) sits at the start of its
      line, left in LTR and right in Arabic.
  - Devices: the Mi Max 3 (Android 10, input locked without a SIM). (was #1;
    lapsed@0917-1659)
- **N-004** · raised `2026-0913-2250-next-list-hook-blame-cell-elements-f-keys-and-a-row-that-fills` · value low
  **F-keys through GameController have never reached the app from XCUITest.**
  Needs a real iPad keyboard. (was #3)
  - 2026-09-29: bare Escape joins it (N-057): XCUITest's Escape reaches no
    SwiftUI keyboardShortcut on the simulator, so core.OnEscape on iOS, and
    whether a Modal closes on it there, wait on the same keyboard.
  - 2026-10-10: `testEscapeClosesTheDrawerAndTheDialog` checks the drawer
    only and never opens 6.6's dialog, despite its name. An iOS Modal still
    has no Escape claim of its own (`core.OnEscape`'s doc), so for a Modal
    the check is whether SwiftUI's own dismissal answers a real Escape.
  - Renamed 2026-10-10 to `testEscapeClosesTheDrawer`, with a doc that says
    why the Dialog is this item's check and not the test's (`421ff1b`). It
    still passes with its expected failure.
- **N-005** · raised `2026-0914-2319-next-list-row-hug-box-chords-and-a-strip-that-divides` · value low
  **iOS chords.** Page-global chords were verified once, and the chord gate
  (behind a modal, inside a shut Drawer panel) is unheard. (was #6;
  lapsed@0917-1659)
- **N-006** · raised `2026-0916-1032-clocks-canvas-alarm` · value low
  **Alarm sound and haptics are unheard,** and so is the Notify banner's
  default sound. Needs a person holding a phone. (was #9; lapsed@0917-1659)
- **N-028** · raised `2026-0917-0227-foldables-window-record-and-two-pane` · value low
  **iOS `AppWindowReader` is unrun in Split View or Stage Manager.** (was
  #37; lapsed@0917-1659)
  - Corrected 2026-10-10: it was "type-checked only", which is stale. It is
    mounted on every launch (`GrMobApp.swift`'s `.background`), the
    tutorial's theme reads its scheme, and that report was seen live on the
    26.5 simulator (N-050, N-085). Still unseen: the size and insets it
    reports changing under Split View, Stage Manager or rotation. No UI test
    asserts on them.
  - Rotation done 2026-10-10. Lesson 4.21 now prints the insets beside the
    size, and `TutorialWindowUITests.testTheWindowRecordFollowsARotation`
    rotates the iOS 26.5 simulator. The size follows (402 × 874 to 874 ×
    402 and back). The insets did not: every iOS report was zero, which is
    N-103, now fixed. They now read top 62, bottom 34 in portrait, and top 0,
    bottom 20, left 62, right 62 in landscape. On the API 36 emulator the
    same readouts go 411 × 914 to 914 × 411, with top 24 and bottom 24 both
    ways (no cutout). Still unseen: Split View and Stage Manager, which need
    an iPad window a script cannot resize.
- **N-030** · raised `2026-0917-0227-foldables-window-record-and-two-pane` · value low
  **Folding shut onto the outer display** needs Samsung's "Continue apps on
  cover screen". Unseen. (was #39; lapsed@0917-1659)
- **N-040** · raised `2026-0918-0910-next-list-copy-strip-edit-epochs-accent-and-the-lost-first-key` · value medium
  **The first hardware key after launch is lost on the iOS 26.5 simulator.**
  Worked around by `primeKeyboard`. Unchecked on a real iPad. (was #50)
- **N-043** · raised `2026-0918-1310-next-list-paragraph-corners-scroll-to-one-field-pin` · value low
  **iOS UIKit field: a queued key during the caret correction.** Not observed.
  Candidate fix: drain through `input.inputDelegate` before `replace`. (was
  #53)
  - Re-checked 2026-10-10: `write()` in GrMobTextInput.swift still calls
    `input.replace(range, withText:)` and then `setCaret`, and nothing under
    ios/ uses `inputDelegate`. Since `carryPlan` (N-065) the common case
    (caret at or after the change) needs no correction, so the window is
    narrower than when this was raised.
  - Stress-tested 2026-10-10 on the iOS 26.5 simulator, and moved to
    Validate: nothing to fix there. 2.3's capitalize-words field reaches both
    correction arms (the span holding the caret, and the span after it). The
    first key typed is the one Go rewrites around, and every key after it
    waits in the keyboard's queue during the write and the correction. Six
    launches, three per seed, typed a 12-key burst ("xyzabcdefghi"), leaving
    11 keys queued behind the rewrite. All six read "Helloxyzabcdefghi World",
    so no key was lost. `testCapitalizedWordsKeepTheCaretWithTheTyping` now
    types that burst instead of three keys (`e23a8b1`). The `inputDelegate`
    candidate fix was not applied. What remains is a real device: a hardware
    keyboard typed fast, and an IME that composes.
- **N-044** · raised `2026-0918-2002-next-list-thread-widget-caret-fixes-boot-frame-proof` · value low
  **iOS Paragraph link colours on the iOS 17 floor.** No iOS 17 runtime is
  installed. (was #55)
- **N-049** · raised `2026-0918-2310-mi-max-3-android-10-force-dark-theme-sweep` · value low
  **A below-the-fold sweep on Android 10.** Needs the Mi Max 3 unlocked, or a
  person scrolling (or an API 29 emulator image). The HID keyboard
  (2026-0919-1254 §1) may drive it. (was #65)
- **N-062** · raised `2026-0921-0912-comps-round-four-phase-1-chat-family` · value medium
  **Phase 1's chat widgets, unrun on a device.** Lesson 4.34 and
  `examples/chat` were looked at in headless Chrome only.
  - `TypingIndicator` under Reduce Motion: the hosts drop the `Transition`
    and Go still steps the phase. Seen on Compose (2026-09-23, the emulator
    with all three animation scales at 0): 30 raw screencaps of the three
    dots' centres read exactly two values, one dark dot at a time, so the
    quiet blink the doc claims is what draws. The web, seen 2026-09-28 in
    headless Chrome on 4.34 (a scratch CDP probe, per-frame computed opacity
    for 3s): each dot passed through about 20 values with a 0.3s transition,
    and exactly two, 0.4 and 1.0, with the transition at 0s under an emulated
    `prefers-reduced-motion: reduce`. SwiftUI, seen 2026-09-28 on the iPhone
    17 Pro simulator (a scratch XCUITest taking about 75 screenshots in 6s,
    the dots' centre pixels read offline): 17–18 luminance values between
    rest (145) and full (0) normally, and exactly two, 145 and 0, with
    `com.apple.Accessibility ReduceMotionEnabled` set in the simulator
    (turned back off after). All three hosts now draw the quiet blink.
  - That a `core.Opacity` `Transition` on a 6pt `Box` eases: seen on
    Compose (2026-09-23): at normal scale the dots' centre pixels passed
    through 34, 47, 59, 73, 87, 111… between rest (145) and full (4–9).
    SwiftUI seen too (2026-09-28): the 17–18 values in the bullet above.
    Lesson 4.34's prose still said the fade was the
    background colour because "core.Style has no opacity"; it now names
    `core.Opacity` and the reduced-motion blink.
  - `TypingIndicator`'s `RoleStatus` appearing from `Display none`: heard on
    TalkBack? VoiceOver is expected to say nothing (the known live-region
    gap, core/role.go).
    Heard on the emulator's TalkBack (2026-10-10). A temporary probe flipped
    4.34's indicator every 4s, then alternated the name (reverted). With the
    harness listening, TalkBack said "Ben is typing" and "Ana is typing" as
    the indicator appeared. It did not repeat a name it had just said: with
    one fixed name it spoke once in a minute of flips. That is TalkBack
    suppressing a repeated announcement, not the node's fault. A version of
    the widget whose region stayed in the tree while hidden gave the same
    pattern, and was dropped.
  - `ReactionBar` chips: emoji glyphs drawn in a `Button` label on both
    natives. The selected state is done: heard on the emulator's TalkBack
    (2026-09-22) as "Selected, thumbs up, 3 reactions, Button" and, after a
    tap, "Selected, party popper, 2 reactions, Button"; on the iOS simulator
    the tapped chip reads "party popper, 2 reactions" and isSelected
    (`TutorialRoundFourUITests.testReactionChipReportsItsSelection`).
  - `Poll` results: one stop per option, with the hidden `ProgressBar` and
    texts not reachable by swipe. Not reached by the emulator's Tab sweep of
    4.34 (N-058's gap), so still unheard.
    With N-058 fixed (2026-10-11), Tab speaks each option once on the
    emulator: "Tabs, Button", "Spaces, Button", "Whatever gofmt says,
    Button". Whether a swipe skips the hidden ProgressBar and texts is still
    unheard (Tab visits controls only).
- **N-064** · raised `2026-0921-0935-core-opacity-and-typing-indicator-fade` · value medium
  **`core.Opacity` is unrun on a device.** It
  compiles on all four targets and its call sites, layer order and sentinel
  are pinned from source, but nobody has seen it fade.
  - Compose: that `Modifier.alpha` outside `Modifier.shadow` keeps the whole
    shadow mid-fade. An alpha below 1 composites the layer offscreen, and an
    offscreen layer may clip what is drawn outside the node's bounds.
  - Compose: `animatedStyle` easing the alpha, and snapping under "Remove
    animations": both seen on the emulator (2026-09-23) through
    TypingIndicator's dots; see N-062. The offscreen-layer shadow question
    above is still open.
  - SwiftUI: the fade under the node's one `.animation` is seen
    (2026-09-28): TypingIndicator's dots pass through 17–18 values on the
    simulator, and snap under Reduce Motion (see N-062). Whether a view at
    exactly 0 still takes taps and VoiceOver focus is still unmeasured (the
    doc says taps stop; that is from SwiftUI's known behaviour).
  - Measured 2026-10-10 with a temporary demo (reverted). An Opacity(0) Box
    with an OnClick was tapped where it sits.
    - The iOS 26.5 simulator: the tap did not reach it (0 taps), as core's
      doc says, and it is still in XCUITest's accessibility tree. Whether
      VoiceOver itself stops there is unheard.
    - The emulator: the same box is in the tree, clickable, and took the tap
      (1 tap), as core documents for Compose.
  - The shadow question above is answered: an alpha below 1 cut the shadow
    off entirely on Compose (a 0.5 Card cast none). Fixed as N-104, and the
    0.5 Card's shadow now darkens the panel by half of what the opaque one's
    does. iOS's 0.5 Card kept its shadow, with an exact interior.
  - Fixed (2026-09-21): Compose's `DisplayHidden` alpha used to sit at the
    foot of `boxModifier`, inside the background and border, so a hidden node
    with a fill still drew the fill. It now shares the Opacity layer
    (`layerAlpha`), as iOS's `.opacity` already did; pinned from source by
    `TestComposeHidesTheWholePaintedBox`. Compiled, not watched.
    Watched 2026-10-10 on the emulator, with a temporary probe (reverted). A
    DisplayHidden Box with a fill, a 3px border and a shadow, between two
    visible ones, drew nothing and kept its space.
  - Seen once on Compose (2026-09-21, the emulator): `NumberPad`'s unpainted
    corner key is `Opacity(0)` and draws nothing, so the sentinel reaches
    `Modifier.alpha` as 0. A static zero only; no fade has been watched.
  - Its first consumer is `TypingIndicator`'s dots (looked at in headless
    Chrome only), so N-062's device checks are this field's too. No lesson
    teaches the prop itself.
- **N-065** · raised `2026-0921-1019-comps-round-four-phase-2-inputs` · value medium
  **The caret carry and the mask, unrun on iOS and on hardware.**
  - Android: `carryCaret` replaced "keep the raw offset" in `GrMobTextField`.
    Run on the emulator only (the mask at 1s and machine speed, backspace,
    mid-text, and lesson 2.3's UPPERCASE mid-text). Not on the Fold6, and not
    with an IME that composes (Gboard's suggestions, Samsung's keyboard):
    `adb shell input text` commits whole keys.
  - Composing IME done on the emulator (2026-10-10). Gboard's on-screen keys
    were tapped by coordinate into 2.3's field with UPPERCASE on, and
    `show_ime_with_hard_keyboard` was restored to 0 after.
    - "hello world" read "HELLO WORLD", with Gboard composing (its strip
      offering "world" and "would").
    - "abc" typed with the caret at the start of "WORLD" read "HELLO
      ABCWORLD", caret after the C.
    - " wor" then a tap on the "world" suggestion committed "WORLD " (with
      Gboard's space), and a following "x" read "X".
    Still not on the Fold6, nor with Samsung's keyboard.
  - iOS: done on the simulator (2026-09-21). `write`'s arithmetic is
    `carryPlan` in GrMobTextEdits.swift, run by `ios/verify` against
    `CarryCases` (the caret, and that the span ends at the caret when the
    caret follows the change; both mutation-tested). It gained Carry's
    surrogate guard. `testMaskedInputFormatsAsItIsTyped` types 5.9's phone
    mask in one burst ("(555) 123-4567"), refuses a letter, backspaces
    through the group break, and types the card mask a key a second. Not on
    a real iPhone.
  - The known limit (a key typed mid-text at the end of a group leaves the
    caret after the reflow) is the same on all three hosts by construction;
    seen on Android only.
- **N-066** · raised `2026-0921-1019-comps-round-four-phase-2-inputs` · value medium
  **Phase 2's inputs, unrun on a device beyond a look.** Lesson 5.9 was
  looked at in headless Chrome and on the Android emulator (pad and
  swatches drawn right; two pad keys tapped).
  - `NumberPad`: the haptic per key felt on a phone; the unpainted corner
    (`Opacity(0)`, `Disabled`, hidden) skipped by VoiceOver. XCUITest's tree on
    the iOS 26.5 simulator (2026-09-29) lists that corner as a disabled
    Button labelled " " despite Go sending `AccessibilityHidden` (and the
    Button's box keeps the flag through `marginAndSizeOnly`); XCUITest's tree
    is not VoiceOver's order (it also lists the lock screen's hidden dots),
    so whether VoiceOver stops there is still Accessibility Inspector's
    question. On Compose it
    is not a Tab stop (2026-09-21) and has no node at all in the
    accessibility tree (2026-09-23, uiautomator: the bottom row holds 0 and
    Delete only), so TalkBack cannot land on it.
  - `ColorSwatchPicker`: the ring and check on SwiftUI, seen 2026-09-28 on
    the simulator (the selected blue swatch ringed with a gap and a white
    check, the other seven plain, the custom well and hex field below);
    VoiceOver still unheard. Done (2026-09-22): TalkBack
    says "Selected, blue, Radio button, 1 of 8" and every position right
    since N-077; the hex field's return commits the short form on both
    natives ("#2a7" leaves Value at #2A78D6 while typed and commits #22AA77
    on return: the emulator, and `testSwatchesHexShortFormAndTheRangeThatCannotCross`
    on the simulator).
  - `RangeSlider`: done on both simulators' input paths (2026-09-22): an
    injected drag of Minimum from $20 to past $80 on the emulator left both
    thumbs and readouts at $115; XCUITest's adjust on the simulator left the
    two sliders' values equal. A real finger is untried. Its values are now
    the readouts (N-079, closed). Its group is a container since N-078
    (closed 2026-09-29): "Price" holds Slider "Minimum" $20 and "Maximum"
    $80 on the simulator.
  - The lock-screen dots' `RoleStatus` line: heard on the emulator's TalkBack
    per key (2026-09-22), "Passcode, 1 of 4 entered", then 2 and 3.
    VoiceOver is expected to say nothing (the known live-region gap,
    core/role.go).
  - Heard on the emulator's TalkBack (2026-09-21, Tab sweep): pad keys as
    "2, Button" … "Delete, Button"; swatches as "Not selected, orange, Radio
    button" (the selected one was not reached). Compose's Tab order goes
    9 → 0 → Delete, so the unpainted corner is not a Tab stop. The swatches'
    positions are wrong: see N-077.
- **N-068** · raised `2026-0921-1035-comps-round-four-phase-3-structure` · value medium
  **Phase 3's structure widgets, unrun on a device.** Lesson 4.35 was looked
  at in headless Chrome only (tree alignment, indents, the wizard's footer).
  - `TreeView`: a branch heard as "docs, collapsed, button" and the chosen
    leaf as selected on TalkBack and VoiceOver; that a `listitem` Box holding
    a button and a nested list is walked in order by swipe; that the level,
    inert on both natives by design, does no harm there. On the web, done
    as far as the browser (2026-09-23): headless Chrome's accessibility tree
    on 4.35 gives guide.md and api `listitem` level 2 inside docs' level 1
    (scratch CDP probe, not a browser check). A screen reader's speech is
    unheard.
  - `TreeView` under RTL: seen mirrored in headless Chrome and on the Compose
    emulator (per-app Arabic locale), 2026-09-21, and on the iOS simulator
    (2026-09-22, `testTreeViewUnderArabic`; `-AppleLanguages (ar)` alone did
    not flip the app, the two forced writing-direction defaults did). On all
    three the collapsed chevron "▸" still points right, against the
    reading direction: split out as N-098 (2026-10-10), since the fix is an
    API decision and it is not TreeView's alone.
  - Heard on the emulator's TalkBack: "expanded. docs. Expands or collapses
    the branch, Button", "Selected, guide.md, Button", "collapsed. api. …".
    TalkBack skipped src, assets and README.md on Tab although Compose's own
    focus visits all six rows (N-058's gap).
    With N-058 fixed (2026-10-11): "collapsed. src. …", "collapsed. assets.
    …" and "README.md, Button" are heard too, then "Collapse all" and
    "Reveal core.md".
  - `Wizard`: the `RoleStatus` line heard on a step change on TalkBack (tried
    on the emulator 2026-09-23 and blocked by N-058; the Fold6's HID harness
    is the route) and in a browser's screen reader; and where TalkBack's
    focus lands when Next replaces the body (the keyed body is a
    replacement, and Compose clears View focus when the focused node leaves:
    2026-0919-1254 §2). On the web and iOS focus now goes to the title
    (N-069); Compose cannot move it.
    The TalkBack half was done 2026-10-10 with a temporary probe (reverted)
    advancing 4.35's wizard every 5s under the emulator's harness. At first
    the line was never spoken, which is N-108: a live-region Text's text
    change went unannounced on Compose. Fixed: "Step 3 of 3", "Step 1 of
    3" and "Step 2 of 3, optional" are now heard at each step. Still open:
    a browser's screen reader, and where TalkBack's focus lands after Next
    when a person presses it.
    From the keyboard (2026-10-11, after N-058): with a name typed, Tab spoke
    "Next, Button", and Enter pressed it. TalkBack said "Step 2 of 3,
    optional", then read "Order, Step 2 of 3, optional, Gift note, Note,
    Optional: …" in one utterance, which looks like a group rather than the
    new title. Tab then went "Step 1: Your name, done" → the note field →
    Back → Skip, and Enter on Skip gave "Step 3 of 3".
  - `Wizard.Footer()` in `Screen.Footer` above the keyboard is N-002's
    pinning check with a consumer now; no bundled screen does it yet.
- **N-070** · raised `2026-0921-1057-comps-round-four-phase-4-charts` · value medium
  **Phase 4's charts, unrun on a device.** Lesson 4.36 was looked at in
  headless Chrome only (all four demos; the look found two defects, fixed).
  - Seen on the emulator and the iOS simulator (2026-09-22,
    `testRoundFourChartsDraw`): `Waveform`'s bars are round-capped and even
    on both (the stroke is unscaled under `CanvasStretch`); the doji draws;
    the funnel draws. The half-px silent bar as a dot was not looked for.
    Looked for 2026-10-10, on both natives: a run of ten zero peaks draws as
    a row of dots (four after bucketing) on the emulator and the iOS 26.5
    simulator.
  - `RadarChart` on SwiftUI was drawn 62pt left of centre. Fixed
    (2026-09-23): the iOS ZStack layout reported its largest layer (the
    180pt canvas) and the `.frame(width: 304)` round it placed that at its
    leading edge, 62pt = (304 − 180) / 2 short. `GrMobStackSolver.containerSize`
    now takes `fills:` and reports the offer on an axis the box is sized on
    (a stated Width/Height or a fill), as Compose and CSS size the box
    first; checked by ios/verify's stack section and pinned by
    `TestNativeZStackOverlaysItsChildren`. On the simulator the chart's
    image is centred on the panel (x 82.8 + 236.7/2 = 201) and "Vision"
    reads whole (`testRoundFourChartsDraw`'s shot). All 48 Tutorial UI
    tests pass with it.
  - `AudioPlayer.Waveform` with a real stream: the strip filling as the
    status ticks, and under the finger while scrubbing.
    Done 2026-10-10 on the emulator and the iOS 26.5 simulator, with a
    temporary `Waveform` on 4.30's player (reverted) and the SoundHelix
    stream.
    - The strip's played part tracks playback (the first bars blue at
      0:09–0:13 of 6:13).
    - Mid-drag it follows the thumb: 2:31 on Android (an adb swipe over 5s),
      3:30 on iOS (`press(forDuration:thenDragTo:…thenHoldForDuration:)`,
      captured from outside with `simctl io screenshot`).
    - After release it resumes from the drop: 3:34 and 3:51.
  - Every chart's one spoken sentence on TalkBack and VoiceOver.
  - 2026-10-03: iOS charts now carry an `AXChartDescriptor` (N-010), which
    type-checks and survives the Release build but has not been opened in
    VoiceOver's chart details or played as an Audio Graph.
- **N-072** · raised `2026-0921-1118-comps-round-four-phase-5-editable-grid` · value medium
  **`EditableGrid`, unrun on a device.** Lesson 4.37 was looked at and driven
  in headless Chrome only (the look found two defects, fixed; the keyboard
  round trip was probed: arrows, Enter into a focused field, Space reaching
  the text, return landing on the cell below with the arrows live).
  - Done on the emulator and the simulator (2026-09-22): a tap opens the
    field with the keyboard up, return commits (Undo (1)), and the ✕ throws
    a typed draft away without the blur committing it first (Undo stays
    (1)). After return the keyboard drops on Compose and stays up on iOS.
    `testGridEditRoundTripAndDiscard` holds the simulator half. The 150ms
    grace, seen in headless Chrome with CDP's real mouse events (2026-09-28):
    a 60ms and a 120ms press on the ✕ discard the draft; a 300ms and a 600ms
    press commit it. That is N-084.
  - The horizontal box was a `core.Box(core.Horizontal())`, which neither
    native scrolls (only a browser reads `overflow: auto`): on the simulator
    the lesson page grew to 490pt on a 402pt screen with every paragraph cut
    at both edges, and on the emulator the grid was squeezed with Amount
    clipped. Now a `core.Scroll`, and Compose's strip gives a grower with a
    points MinWidth a definite width (`GrMobGrowStrip`, pinned by
    `TestComposeStripGrowerWithAFloorIsMeasuredAtAWidth`), so the rows keep
    their weights: seen aligned and scrolling sideways on the emulator, the
    simulator and in headless Chrome.
  - The frameless `core.Select` in a cell keeps the row at the other cells'
    height on both natives. The iOS row in EDIT that drew its Category text
    about 3pt left is fixed (2026-09-23): a zero-basis child's base was its
    padding alone (`zeroBasisPadding`), and the editing cell trades 2pt of
    padding a side for its 2pt ring, so it started 4pt short and the weights
    moved the rest of the row. It now reads `contentInsets` (padding plus a
    drawn border, as CSS counts a basis); pinned in
    `TestIOSFlexHonoursAZeroBasis`, and seen aligned on the simulator.
  - The soft keyboard covering the active cell near the bottom of the grid:
    judged on Compose (2026-09-28, the emulator with
    `show_ime_with_hard_keyboard` 1, restored to 0 after): a tap on row 4's
    Item cell, 1870–1967px on a 2400px screen, brought the editor up to sit
    just above Gboard's suggestion strip, text and ✕ whole. On the iOS
    simulator row 2's editor sits clear of the keyboard. Row 4 (the last) was
    tried 2026-09-28 and opens its editor, but that simulator had a hardware
    keyboard connected, so the soft keyboard was off screen (its frame at y
    952 of 874) and covered nothing: still unjudged on iOS.
  - Judged on iOS 2026-10-10, with the simulator's hardware keyboard turned
    off (and restored after). It was covered: row 4's editor opened at y 756
    under a keyboard whose top was at 583, not hittable. Raised and fixed as
    N-102. Row 4's editor now sits at y 502 above the keyboard, and
    `testTheLastGridRowsEditorIsAboveTheKeyboard` holds it. This bullet is
    done on both natives.
  - On iOS the grid was one static text "Budget" with its text cells out of
    the accessibility tree. N-078 (closed 2026-09-29) made it a container
    whose every cell is a Button ("Amount, row 1, $1200.00"), seen in
    XCUITest's tree; VoiceOver itself is unheard.
  - TalkBack and VoiceOver: a cell heard as "Amount, row 2, $310.50, button",
    the editor's name, the `RoleAlert` message heard on a refused commit
    (TalkBack; VoiceOver is expected to say nothing), and the row menu.
    Heard on the emulator (2026-09-21): "Amount, row 2, $310.50. Edits the
    cell, Button" and "Row 1. Opens the row's menu, Button". A Category cell
    (the frameless `core.Select`) says "Category, row 1, Home" with no role.
    By Tab after N-058's fix (2026-10-11), in Compose's order: "Row 1. Opens
    the row's menu", "Item, row 1, Rent. Edits the cell", "Category, row 1,
    Home", "Amount, row 1, $1200.00…", "Paid, row 1, checked, Button", then
    row 2. Moving to row 2 scrolls the grid back sideways, and TalkBack first
    says "Amount, row 2" before the correction to "Row 2. Opens the row's
    menu". The editor, the RoleAlert and the row menu are still unheard.
  - The cost on a phone: the doc's "about 5,000 cells" is 4µs a cell measured
    on an M3, times a guess. Type into a 10 × 500 sheet on the Fold6.
- **N-097** · raised `2026-1010-1728-keystore-binding` · value low
  **The keystore is unrun on physical hardware.** Checked with a throwaway
  probe on the API 36 emulator and the iOS 26.5 simulator. Covered: save,
  read, overwrite, a non-ASCII value, an empty value, a double delete, a
  20-call burst, persistence across relaunch, a tampered entry, a simulated
  lost key (a renamed alias), and iOS items outliving uninstall until the
  fresh install's wipe. Unchecked:
  - A hardware-backed key: the Fold6's TEE/StrongBox, and the Mi Max 3 on
    Android 10, the oldest Keystore GCM-with-AAD in reach.
  - A real Auto Backup restore (`bmgr`) instead of the renamed alias. Done
    2026-10-10 on the API 36 emulator, with a throwaway probe app that was
    deleted after.
    - The probe saved "probe". `bmgr backupnow` ran on the local transport,
      then uninstall (which drops the AndroidKeyStore key) and `adb
      install`. The install restored `grmob_keystore.xml` by itself, with
      one entry sealed under the old key.
    - On launch Keystore.kt logged "no key for 1 sealed entries (restored
      from a backup?); discarding them". Get answered found=false with no
      error, and a fresh Save then read back.
    - Afterwards: the local backup wiped, Backup Manager disabled again on
      the GMS transport, the tutorial reinstalled, and POST_NOTIFICATIONS
      and SCHEDULE_EXACT_ALARM granted again.
  - An iOS background launch before first unlock, which should answer
    errSecInteractionNotAllowed (-25308) rather than hang.

  church_mobile's N-004 swap is the natural first real consumer.

- **N-101** · raised `2026-1010-1955-next-list-n043-caret-burst-n091-natives-look-n100-align-self` · value low
  **Discussion's header rows: does a screen reader read the name and time
  again after the row's label?** Each header row carries the label the
  natives need ("Ben, reply to Ana, 1h", seen in both trees on 2026-10-10).
  Its inner Texts ("Ben", "1h") still appear as separate nodes in the
  emulator's uiautomator dump and in XCUITest's tree. Neither tree is what
  TalkBack or VoiceOver walks, so whether a swipe stops on them is unheard.
  The emulator's Tab-driven harness cannot settle it (N-058); the Fold6's HID
  harness can. N-058's fix (2026-10-11) does not change that: Tab visits
  controls, and the question is about the texts a swipe stops on.

- **N-111** · raised `2026-1017-2211-gallery-widget` · value medium
  **`comps.Gallery` has been seen on the web and partly on Compose** (headless
  Chrome shots of lesson 4.41: theme, cinema with deck and thumbnails,
  polaroid flipped, minimal zoomed in place, fullscreen Lightbox). Compose,
  2026-10-19, in the catadopt app on a Galaxy Z Fold6's cover screen
  (`2026-1019-2323-android-image-agent-and-scroll-strip-floor`): theme look
  under AmberTheme, slide transition, the thumbnail strip (it overflowed its
  arrows until N-117) and the fade flip to a Details back were seen, and
  photos loaded once N-116 gave Coil a User-Agent. SwiftUI, 2026-10-19, in
  the same app on the iOS 26.5 simulator (iPhone 17 Pro;
  `2026-1019-2343-ios-filled-image-frame-and-screen-fill-scroll`): the front
  face, the caption and the thumbnail strip render at the screen's width
  once N-118 stopped a filled picture widening its box. Before that the whole
  screen was about 693pt wide. Nothing was tapped there (simctl cannot tap).
  Still to check on Compose unless noted:
  - the card layers' Translate, Opacity and Rotate transitions;
  - deck paint order: the natives use child order, while the web needed
    `ZIndex`;
  - the in-place zoom's height, padding and radius transitions;
  - the rise flip clipped by the frame's Overflow;
  - the back face's Scroll at a pinned height;
  - the hidden tap-zone Row taking taps, and long press zooming. On the
    web the zones never took a tap until N-115 (the 4.41 shots never
    tapped the photo). On Compose a tap in the middle zone flipped the card
    (Fold6, 2026-10-19); the edge zones, long press and SwiftUI are
    untapped;
  - the `core.WithTheme` wrapper around a card;
  - TalkBack and VoiceOver reaching only the current card.
- **N-114** · raised `2026-1017-2211-gallery-widget` · value medium
  **`core.Share`, the system share sheet, seen on iOS only.** Added
  2026-10-21 (`2026-1021-1132-work-next-list-n110-n109`): `core.Share(text,
  url, done)` sends a "share" system event with an id; each host answers
  `{id, ok}`, where ok means a sheet was shown (a dismissal still counts).
  The web uses `navigator.share` (ok:false without it, when canShare says
  no, or on any rejection except AbortError). Android sends ACTION_SEND
  through `Intent.createChooser` (`Share.kt`). iOS presents a
  UIActivityViewController over the top presented controller, anchored
  centre on iPad (`Share.swift`). `comps.Gallery`'s default Share now calls
  it, and falls back to the clipboard and a toast on ok:false. Checked so
  far: core and Gallery Go tests; share_test.mjs; mobile/verify's spelling
  and dispatch tests; a Kotlin compile (`:app:compileDebugKotlin`) and an
  iOS simulator build. On the iOS 26.5 simulator,
  `TutorialShareSheetUITests` taps 4.41's Share and the sheet comes up,
  with the picsum.photos link preview, Copy and Add to Reading List. Still
  to check:
  - Android: the chooser opening from 4.41 on the emulator or the Fold6,
    and the text and link arriving joined in a messenger;
  - the web in a phone browser (Chrome on Android, Safari on iOS), where
    the tap's user activation must reach `navigator.share` through the
    wasm callback;
  - a desktop browser without the Web Share API (Firefox) falling back to
    the clipboard toast;
  - the iPad popover anchor.
- **N-122** · raised `2026-1020-1933-core-swipe-gesture` · value medium
  **`core.OnSwipe` / `OnVerticalSwipe` on the natives and under real touch.**
  Checked so far: the Go tests and the DOM shim's tests. In headless Chrome
  with CDP mouse input, a drag across Gallery's middle zone stepped both ways
  without flipping the card, and a 5px press still flipped it. The Kotlin
  (`GrMobSwipe.kt`) passed kotlinc's type check against the cached Compose
  1.7.6 jars, but not a gradle build: there was no NDK for the gomobile AAR.
  The Swift passed `ios/verify` (typecheck plus whole-module optimisation),
  with no simulator run. Still to check:
  - Compose: a swipe that starts on a clickable tap zone steps and does not
    also tap; a vertical drag over a swipeable card still scrolls the page; a
    long press in the middle zone still zooms;
  - SwiftUI: the same three, plus whether a tap child really fails once the
    finger travels (the doc says "expected"), and a child Button's own
    tracking;
  - RTL on both natives: leading is a finger moving right. iOS reads the
    translation in the global space on the assumption that it is physical;
  - the web under real touch: `touch-action: pan-y` hands vertical pans to
    the page, and a horizontal swipe on a phone browser steps the gallery.
- **N-123** · raised `2026-1021-1113-shopping-cart-library` · value medium
  **The cart widgets on the natives and with a screen reader.** Checked so
  far: Go tests (cart, cartdb with a relaunch, UseCart, every widget, lesson
  4.43) and screenshots of lesson 4.43 in the browser shots host. Still to
  check:
  - Android and iOS: CartLine's filled 64-pt thumbnail, the Stepper and
    Remove row, CartButton (a Row acting as a button) inside an AppBar's
    Actions, and PromoCodeField's input beside its Apply button;
  - TalkBack and VoiceOver: each line's text as one phrase, the Stepper
    named "Quantity, <item>", "Remove <item>", "Cart, 3 items", and the
    refusal alert;
  - SwipeToRemove under real touch, with N-122's checks;
  - `cartdb` on a device: a cart under `mobile.DataDir()` survives killing
    the app.

- **N-126** · raised `2026-1023-1428-bible-study-widget` · value medium
  **BibleStudy on the natives and with a screen reader.** Checked so far:
  Go tests (comps, blb, lesson 4.44), browser shots of lesson 4.44 with the
  rows and the open sheet, and the live BLB check that
  `StudyURL(17, ToolCrossRefs)` opens John 3:17's page. Still to check:
  - Android and iOS: verse rows as tap targets inside the card, the hanging
    number on a wrapped verse, the ActionSheet sibling opening over the
    card, and `core.OpenURL` opening the BLB study tab in the browser;
  - TalkBack and VoiceOver: each row read as "Verse 2, He maketh me…" with
    the hint, the prompt caption skipped, and the sheet named
    "Psalms 23:2 (KJV)";
  - BLB's study page on a phone browser landing on the named tab
    (Cross-Refs, not the default Interlinear).
- **N-144** · raised `2026-1111-0120-reliability-performance-convenience-plan` · value medium
  **Test order, a CI flake, and CI time (plan R9n, R9o, P10).** Repairs to
  tests and CI scripts only. `go test -shuffle` fails on seeds 2 and 3
  (`hooks/location_test.go:48`, `examples/todoapp/app_test.go:128`) through
  process-wide singletons; fix both and add `-shuffle=on` to CI (the reset
  helper comes with N-140). The Android remedy drill failed run 40020908087
  in its cleanup trap after printing its own `OK:`
  (`android/verify/remedy.sh:67`; `ios/verify/remedy.sh:42` has the same
  shape). CI runs the race detector over source-census packages
  (`wasm/verify` 5.7 s plain, 54.9 s under `-race`); timer tests sleep real
  time where `testing/synctest` ran 10 fake seconds in 0.57 ms; the site
  workflow repeats CI's verify.
  - R9o landed 2026-11-11 (`2026-1111-1142-n137-d-e-lazy-keys-and-web-teardown`).
    Under `set -e`, a command that fails in an EXIT trap becomes the
    script's status (shown in sh, bash and dash), and the Android drill's
    `rm -rf` raced the gradle daemon that `sources.sh` leaves writing into
    the drill's `GRADLE_USER_HOME`. Both drills' cleanup now cannot fail:
    Android stops that daemon first (only if one ran) and then removes
    `"${cold:?}"`, with `|| true` on each step. INT and TERM `exit`, so the
    cleanup runs once from EXIT instead of deleting the directory and
    carrying on. `ios/verify/remedy.sh` passed end to end. The Android
    drill was not run here (it fills a cold gradle cache from the network),
    so whether CI stays green is this item's check. R9n and P10 remain.
  - R9n and P10 a–c landed 2026-11-11 (`2026-1111-1851-n140-n144-n138-grmobtest-useasync-shuffle-fake-clocks-p0`),
    and the item moved here from Open: what is left is watching CI.
    - R9n: hooks tests that feed a device or permission record start with
      `freshProcessState` (`grmobtest.Reset` in and out); todoapp's two
      journeys start with `freshApp`. CI runs `-shuffle=on`. The full suite
      passed under eight seeds and once with `-race -shuffle=on`.
    - P10a: the race step covers product packages only; the source
      censuses (`CENSUS_PKGS` in ci.yml) run in their own step without
      `-race` (about 32 s against 127 s over that set, locally).
    - P10b: timer tests run on `grmobtest.WithFakeClock` (synctest plus a
      30 s watchdog): hooks 5.3 s to 0.3 s, the tutorial 9.4 s to 1.6 s.
    - P10c: site.yml deploys on a green CI `workflow_run` of a push to
      master, checking out `head_sha`; a manual dispatch keeps the old gate.
    - P10d (serving a compressed module) is a product change, split out as
      N-145.
    - The R9o check so far: run 43070809214 (2026-11-11, `15f2b4c`) ran
      `android/verify/remedy.sh` and its Android job passed. The flake was
      intermittent, so one green run is not yet the answer.
  Checks left: the Android remedy drill stays green across CI runs; the
  split test steps and `-shuffle=on` run green in CI; the site workflow's
  first `workflow_run` deploy succeeds after CI and publishes the commit CI
  verified.

## Non-goals

- **N-001** · declined `2026-0912-1744-the-widget-library-answers-to-comps` —
  a bundle of small declines, each recorded where it was raised: (was #8;
  lapsed@0917-1659)
  - Rename `docs/components.md` to `comps.md`.
  - Trim the Android shell's permissions; the iOS usage strings.
  - C4 `Carousel` (N-046 declines the scroll offset it needs).
  - Android `onBack` ranking.
  - Forward after browser back.
  - Sticky headers or `OnEndReached` in a List with no viewport on Compose.
  - MaxWidth with a growing sibling.
  - The typed-hash `history.length` fallback; a page's own `pushState` during
    a claim.
  - A Drawer's shut panel is composed on the natives (unfocusable on Compose
    since 2026-0919-1254).
  - A List with no Height is not lazy.
  - Unformatted commits already on a remote.
- **N-008** · declined `2026-0916-1032-clocks-canvas-alarm` — `DigitalClock`
  digits shifting by a pixel; `AnalogClock{Smooth}` spinning back when the
  midnight tick is skipped. (was #11; lapsed@0917-1659)
- **N-011** · declined `2026-0916-1129-charts-on-canvas` — Chart summaries
  are English. Every chart's `AccessibilityLabel` already replaces the
  summary, so an app that needs another language states its own. Declined by
  the user 2026-09-25. (was #13)
- **N-012** · declined `2026-0916-1129-charts-on-canvas` — A 180° `Gauge`
  leaves its bottom half empty. (was #14; lapsed@0917-1659)
- **N-013** · raised `2026-0916-1157-next-list-charts-on-devices-tier-e-alarms-timezone`
  · declined 2026-10-21, `2026-1021-1132-work-next-list-n110-n109` — `mobile.SetTimeZone` runs once at
  startup, and that stays. The item's recommendation (2026-09-29) named this
  default. Holding the zone in core (`core.Local()`, set from a "timezone"
  host event) would leave app code calling `time.Now()` on the launch zone,
  so there would be two clocks that can disagree. A zone change mid-run is
  rare, and a relaunch picks it up. (was #16; lapsed@0917-1659)
  Premise corrected 2026-11-11: on Android it runs on every Activity
  recreation, not once (`GomobileBridge.kt:28-35`), which
  `mobile/timezone.go` calls a data race. N-134 makes it once again; the
  decision stands.
- **N-017** · declined
  `2026-0916-1331-next-list-sweep-maxlines-chart-types-evenodd` — `MaxLines`
  on the natives applies to Text only; a stacked chart counts NaN as 0. (was
  #21; lapsed@0917-1659)
- **N-020** · declined `2026-0916-1410-chart-palette-and-canvas-gradients` —
  `core.LinearGradient` is unused; kept per the no-removal rule. (was #24;
  lapsed@0917-1659)
- **N-023** · declined
  `2026-0916-1557-small-fixes-stroke-gradients-bar-values-alarm-groups` —
  Renaming a `NotifyGroup` strands its schedules. (was #28; lapsed@0917-1659)
- **N-027** · raised `2026-0917-0227-foldables-window-record-and-two-pane`
  · declined 2026-10-21, `2026-1021-1132-work-next-list-n110-n109` — lesson 4.21's TwoPane sets no `Origin`.
  The item's recommendation (2026-09-29) named this default. The missing
  term is `Origin.X`, the demo panel's own 55px left inset, which is a
  layout constant no host reports. Hard-coding it would teach a guess, and
  the lesson's prose already names the missing term. (was #36;
  lapsed@0917-1659)
- **N-031** · declined 2026-10-04 · raised `2026-0917-0227-foldables-window-record-and-two-pane`
  — **Housekeeping.**
  - The `GrMob_Foldable` AVD is still installed. Delete it? (User's call.)
  - The Mi Max 3 still has `stay_on_while_plugged_in` 7 (was 0) and
    auto-rotate off (was on).
  - The emulator's GrMob app has POST_NOTIFICATIONS and SCHEDULE_EXACT_ALARM
    granted.
  - New: the Fold6's Samsung TalkBack now has READ_PHONE_STATE granted (was
    denied), kept for the harness; "Display speech output" is still on. (was
    #40; lapsed@0917-1659)
- **N-032** · declined `2026-0917-0227-foldables-window-record-and-two-pane` —
  More than one fold; a static HTML export of a TwoPane. (was #41;
  lapsed@0917-1659)
- **N-033** · declined `2026-0917-1659-fab-screen-floating-and-round-two-plan`
  — A native time wheel; a sheet or Done on TimePicker. (was #44)
- **N-035** · declined
  `2026-0917-2336-tier-e-and-f-small-pieces-and-a-scale-for-quantities` —
  Heatmap as a continuous gradient; a Sequential ramp interpolated from
  `Primary`. (was #45)
- **N-037** · declined
  `2026-0918-0130-round-three-eight-widgets-and-what-a-text-node-cannot-do` —
  A year-wide scrolling `CalendarHeatmap`; a caption-flip "Copied ✓"; a
  separate `Alert` widget. (was #47)
- **N-038** · declined `2026-0918-0150-two-pane-tutorial` — A two-pane layout
  on the natives; restructuring lesson bodies into guide and demo halves. (was
  #48)
- **N-039** · declined
  `2026-0918-0713-device-pass-nine-bugs-the-simulators-found` — Making iOS
  keep focus on every submit by default. (was #49)
- **N-041** · declined
  `2026-0918-0910-next-list-copy-strip-edit-epochs-accent-and-the-lost-first-key`
  — A right-padding gutter or a toolbar header for code blocks. (was #51)
- **N-042** · declined
  `2026-0918-1046-next-list-textfieldstate-editor-stamps-button-floor-compact-copy`
  — A smaller minimum for Buttons in general on Android. (was #52)
- **N-046** · declined
  `2026-0918-2002-next-list-thread-widget-caret-fixes-boot-frame-proof` — A
  reported scroll offset from the hosts. (was #64)
- **N-048** · declined
  `2026-0918-2203-android-device-pass-fold6-secure-field-weight` — Continuing
  apps onto the cover screen by default. (was #63)
- **N-052** · declined
  `2026-0918-2310-mi-max-3-android-10-force-dark-theme-sweep` — MIUI's
  `MiuiContrastOverlay` dims screenshots to 75% in dark mode. (was #69)
- **N-053** · declined
  `2026-0919-0443-next-list-no-device-image-fill-fab-fill-tab-stops` — Lint's
  23 warnings. `lint.sh` gates on errors only, on purpose. (was #74)
- **N-054** · declined
  `2026-0919-1118-next-list-readonly-code-tab-stop-button-min-fill-onring-relaunch`
  — 4.19 can't show OnRing after a force-stop. Its alarms live in memory. The
  path works. (was #76)
- **N-061** · raised `2026-0919-2303-safe-insets-record-inert-codeeditor-sse-cleanup`
  · declined 2026-10-03, `2026-1003-1559-n061-n088-safe-area-insets-on-the-web` — the browser reports no safe-area insets, by design.
  Zero is the true answer for a page in a browser window; the one case it is
  wrong (a page drawn behind a notch with `viewport-fit=cover`, typically an
  installed PWA) would cost a probe element and a `getComputedStyle` per
  report, and that page owns its HTML: one CSS padding of
  `env(safe-area-inset-*)` on the element hosting the app keeps the content
  clear with no number reaching Go. Recorded in `core.SafeInsets`' doc
  comment (and docs/api/core-device.md) and in the runtime's windowMetrics
  comment. The tutorial page that does set cover is N-088.

## Closed

- **N-140** · raised `2026-1111-0120-reliability-performance-convenience-plan` · value high
  · closed 2026-11-11, `2026-1111-1851-n140-n144-n138-grmobtest-useasync-shuffle-fake-clocks-p0` — both halves landed.
  - C1 (`5adce9a`): `grmobtest.New(t, App)` returns a Driver over
    render.Manager with `Tree` (a `*core.Node` decoded from the wire),
    `Find`/`FindAll`, `Tap`, `TapWhere`, `Type`, `Toggle`, `HasText` (exact
    Text), `Shows` (any word on screen, never a callback ID), `AwaitText`
    and `NoConcerns` (which also fails with debug mode off). `Reset` runs
    the eleven unexported resets through `internal/testreset` (core,
    keystore and permission register from init; a census test holds every
    `reset…ForTest` to a registration), plus the concern and pass records.
    New resets when it opens the only live Driver and the last close
    resets again, so a host's boot report is sent after New. Migrated: the
    scaffold, counter, signup, social, mobileapp, todoapp and the tutorial
    (its `node` is now `core.Node`). Later (`34cdb34`) `WithFakeClock`:
    synctest plus a real-time watchdog.
  - C2 (`4addc41`): `hooks.UseAsync[T](ctx, fn, deps...)` returns
    `Async{Data, Err, Loading, Reload}`. fn's context is cancelled on a
    deps change, on Reload and on close; a generation check drops any
    superseded answer; a deps change clears Data and Err, and Reload keeps
    Data and clears Err at once; a panic becomes `ErrAsyncPanicked`.
    Mutation-checked on four guarantees. Documented in hooks' package doc,
    state-and-hooks.md, SKILL.md, BibleVerse's doc and the Effects lesson.
- **N-137** · raised `2026-1111-0120-reliability-performance-convenience-plan` · value low
  · closed 2026-11-11, `2026-1111-1142-n137-d-e-lazy-keys-and-web-teardown` — the plan's eleven small
  reliability items (R9 b–l), each landed and pinned on its own; one line
  each below. Sessions: `2026-1111-1116-n137-reliability-letters-b-c-f-through-l`
  (b, c, f–l) and this one (d, e).
  - f landed 2026-11-11: the fire clears `d.timer` when it is still the
    current timer, so `Pending` is false while the call runs and after. A
    fire that lost its race with a newer `Call` leaves that one pending.
    Pinned by two tests in `hooks/debounce_test.go`. The stale-fire test
    holds the debouncer's lock to force the race, and fails on an
    unconditional clear.
  - b landed 2026-11-11: every `mobile` entry point (and `ReportHostEvent`)
    goes through `manager()`. Before `Register`, `RenderInitial` returns a
    one-Text tree saying `mobile.Register was never called`, every other call
    returns `[]`, `SetListener` is a no-op, and the log says it once. The
    plan's `{"error": …}` was not used: hosts refuse that payload (N-133's
    finding), and a tree draws with no host change. Pinned by
    `mobile/unregistered_test.go`, which panics with one entry point
    unguarded. native.md's bridge table says it.
  - c landed 2026-11-11: `renderAll` (every container, Fragment, For,
    Modal, TabView, camera overlays) and `WithTheme`'s loop drop a nil
    child instead of keeping a slot, so Go diffs by the indices every host
    holds. Both natives skipped JSON null while parsing, and the web's
    `renderNode` would have thrown on it. `core.Keyed` over a nil returns nil
    instead of panicking on `n.Key`. The natives' null-skip stays as a
    defence. Pinned by `core/nil_child_test.go`; the old code fails all three
    tests. `go test ./...`, wasm/verify and ios/verify are green.
  - i and j landed 2026-11-11 (`serve/dev.go`). The module root comes from
    `go env GOMOD` (`locateModule`), which names the module containing the
    working directory even under a go.work. `go list -m` printed every
    workspace module, and their joined lines failed the build.sh check.
    `goSources` now skips only `GOMODCACHE` (`watchesPackageDir`, with a
    `filepath.Rel` containment test), so a `replace => ../grmob` checkout is
    watched. Pinned in `serve/dev_test.go` by a real scratch workspace, a
    directory outside any module, and the watch rule's cases. Not run live
    with a browser attached.
  - g, k and l landed 2026-11-11 (`cmd/grmob`).
    - g: `installShell` copies, patches and records in a
      `.<dir>-partial-*` staging directory beside the shell, then renames
      it into place. A refresh merges the staged, already-patched tree over
      the old one (keeping the app's own files) and writes the record last.
      A killed run's staging is cleared by the next one.
    - k: the scaffold's steps moved into `scaffoldApp`. On any failure, or
      a Ctrl-C (caught for the scaffold's length), `removeScaffold` removes
      what the run wrote: the whole directory if the run created it, else
      everything but `.git`. Chosen over the plan's `-force` alternative,
      which would have to decide what to overwrite. A failed first build is
      kept, since the app is complete by then.
    - l: the templates escape the name for each place it lands:
      `[[html .Name]]` in `<title>`, `[[js .Name]]` in the boot error's JS
      string, `[[printf "%q" .Name]]` for app.go's two Go strings. An
      ordinary name renders byte-identically, so `grmob web`'s "differs"
      check stays quiet on existing pages. `validateName` refuses control
      characters (a line break would split app.go's comment and the
      README's heading).
    Pinned by `cmd/grmob/shell_install_test.go` (4) and two tests in
    `new_test.go`. The k and l tests fail with their fix removed. g's tests
    encode the failure the plan names; there was no `installShell` before
    to run them against.
  - h landed 2026-11-11: `GrMob.instantiateWasm(response, importObject)` in
    the runtime, called by all three host pages (the site, the `grmob new`
    template, the shots page). It streams only on exactly `application/wasm`
    and buffers anything else, with no catch-and-retry, so a CompileError or
    LinkError arrives as itself. The test found that engines differ past the
    exact value: the spec ignores case, Node is case-sensitive, and both refuse
    parameters. Pinned by `wasm/verify/instantiate_test.mjs` (real modules
    and Responses), which also shows the old shape reporting the body
    instead of the error. `wasm/verify/run.sh`'s browser pass boots the site
    page through it. wasm.md's host contract says it.
  - d landed 2026-11-11. The premise is confirmed from source: `GrMobList`
    flattens every For group's Fragment into one row list, and Go's
    duplicate-key check reads one sibling list at a time. So two groups
    keyed by the same IDs gave `LazyColumn` a repeated key, and Compose
    throws on that. `grMobLazyKeys` (new, import-free `GrMobLazyKeys.kt`)
    keeps a first occurrence's key as it was, keys a repeat as
    `Pair(key, n)` (Go's `idScope.seen` rule, unable to meet a real key),
    and leaves unkeyed rows positional. Run by android/verify's JVM harness
    (hand-written cases, since Go has no counterpart); returning the raw
    keys fails it. `run.sh` compiles with the Compose plugin and lint is
    clean. The crash itself was not reproduced on a device. SwiftUI's
    list identifies rows by `viewID`, not the Go key, so it has no such
    throw.
  - e landed 2026-11-11: the web runtime keeps a teardown registry
    (`onDetach(el, name, release)`, `disposeDetached` at the outermost exit
    of every mount and batch, releasing each registered element that is no
    longer connected). A map stops its location watch and is removed from
    Leaflet. A list's end/start observer disconnects. A canvas leaves the
    page-long `canvasTextObserver`, which had held every text-drawing canvas
    for good. A map element removed before its creation frame no longer
    retries every frame forever (one frame, then it gives up). `dom.mjs` now
    models `isConnected`, `load.mjs` has an opt-in recording ResizeObserver,
    and the fake Leaflet counts `remove()`. Pinned by
    `wasm/verify/teardown_test.mjs` (7). Six fail with the prune removed;
    with the old retry, the seventh never returns. `wasm/verify/run.sh`
    (browser pass) is green. wasm.md says it.
- **N-136** · raised `2026-1111-0120-reliability-performance-convenience-plan` · value low
  · closed 2026-11-11, `2026-1111-1057-n135-n136-hook-goroutine-panics-and-typed-json` — typed text that is
  valid JSON reaches its handler as typed on the web (plan R8).
  `core.Context.ReceiveEventPayload` now picks the handler by the callback
  ID's kind prefix (`txt_cb_`, `bool_cb_`, `int_cb_`, else void; named once
  as constants that registration also uses) and checks the value against
  it. A text value is never parsed. A value of the wrong type is logged
  ("dropping event for … want a string, got float64") and dropped; before, it
  missed its map silently. A void ID ignores a stray value. No host sent a
  nested envelope, so nothing depended on the unwrap: the web runtime's
  senders (`extractEventPayload`, the code and rich editors, tabs, maps,
  swipes, shape taps) send one plain `{value}` each, and their comments now
  describe routing by the ID.
  - **Pins** (`core/event_test.go`): seven typed values (`{"value":"x"}`,
    `{"value":true}`, `{"value":7}`, `{"value":null}`, a rich document,
    `"quoted"`, `42`) each arrive verbatim. The old `event.go` delivered
    `x` and lost two of the seven. Also: a keyed ID routes by its prefix, a
    void or back ID ignores its value, and mismatches are logged.
    `go test ./...` and `wasm/verify/run.sh` (browser pass) are green. Not
    typed into a live page.
- **N-135** · raised `2026-1111-0120-reliability-performance-convenience-plan` · value medium
  · closed 2026-11-11, `2026-1111-1057-n135-n136-hook-goroutine-panics-and-typed-json` — a panic in a callback a
  hook runs on its own goroutine no longer ends the process (plan R7).
  `core.Safely(where, fn)` (`core/safely.go`, on `core.Guard`) recovers, logs
  as `logRenderPanic` does ("grmob: recovered panic during UseInterval tick:
  …" plus the stack) and, under debug mode, reports `handler-panic` with the
  place in the detail. It wraps `go effect()`, the interval ticker's call,
  the `UseTimeout` and `UseTimeoutWhile` fires, the `Debouncer`'s
  `AfterFunc`, and `UseAlarms`' `OnRing` from its ticker (not in the plan;
  the same shape). `now.go`, which the item named, runs no app code on its
  goroutine, so it has nothing to guard. Each hook still requests the render
  it would have after a normal return. Exported, so app goroutines can use
  it; documented in error-boundaries.md, debug-mode.md's table and
  state-and-hooks.md.
  - **Pins:** `hooks/panic_test.go` (a panicking interval's next tick fires
    and its render is still requested; a panicking effect, timeout,
    timeout-while and debounced call are each recovered and the hook works
    after); `core/safely_test.go`. Mutation-checked: the bare `f()` back in
    the ticker crashes the test binary with "panic: first tick fails".
    Full `go test ./...`, `-race` on core, hooks and render, and the wasm
    build are green.
- **N-133** · raised `2026-1111-0120-reliability-performance-convenience-plan` · value medium
  · closed 2026-11-11, `2026-1111-1035-n133-unencodable-pass-and-resync` — an unencodable pass is
  dropped in Go instead of sent as `{"error":...}`, and every host asks Go
  for its whole tree when its own has drifted (plan R3, R4).
  - **R3** (`render/manager.go`): a pass is encoded before it is committed.
    On failure it is abandoned like a panicking pass (currentTree kept, purge
    skipped, dirty cleared, `[]` returned) and logged with the first
    non-finite number's place (`render/nonfinite.go`: "root/1 (Slider)
    Props.value = NaN"); under debug mode also the new
    `core.ConcernUnencodablePass` (`unencodable-pass`, in
    docs/concepts/debug-mode.md). An unencodable mount stands in the panic
    placeholder, which the first encodable pass replaces. Pinned by
    `render/encode_test.go` (a NaN in a style float and in a prop beside an
    unrelated text change, which arrives with the next good pass; the mount;
    the concern; the push path).
  - **R4**: `render.Manager.Resync` sends currentTree as one replace of
    `root`, delivered like a dispatch (through the listener inside the lock,
    else returned), re-rendering nothing (`render/resync_test.go`). Bridged as
    `mobile.Resync` (stub, both bridges, native.md) and `GrMobWASM.Resync` on
    all three browser hosts (deferred under PushGuard; `webhost.Bindings`).
    Hosts ask once per drift (a path that resolves to nothing, a payload
    that is not a list, a patch that threw) behind a "resync pending" flag a
    root replace clears. The web runtime queues the request through
    dispatchToGo's queue (now of calls), wraps the batch so post-passes run,
    and can answer a root replace with no root on the page. Swift and Kotlin
    TreeStores stopped missing updates silently (`?.props =` was a quiet
    no-op on a miss).
  - **Checked:** resync_test.mjs (each symptom, once per drift, after the
    push, a polling page, a Go without the binding, and a replay that lost a
    batch ends on Go's final tree); the replay suite asserts a faithful
    replay reports no drift; ios/verify's `checkResync` (no false alarm, and a
    node-sized hole asks once and matches Go after the answer; a lost batch or
    patch left nothing dangling on that transcript); a live probe of all 87
    lessons with their buttons pressed (no drift, no Resync); and a live
    induced drift on 2.2 (one Resync, the node back with the new event).
    Android compiles against a rebuilt AAR and `android/verify/run.sh` is
    green, but its drift handling is executed by no test until R9m (noted on
    N-134). Full Go, race, wasm/verify (browser pass) and ios/verify green.
- **N-132** · raised `2026-1111-0120-reliability-performance-convenience-plan` · value medium
  · closed 2026-11-11, `2026-1111-1020-n131-n132-web-push-deadlock-and-dispatch-order` — a
  tap's patches can no longer reach the host after a later pushed pass (plan
  R2). With a listener attached, every `Dispatch*` (and `Dispatch`, the
  host-event path) hands its pass to the listener inside the lock hold
  (`render.Manager.deliverLocked`, sharing the pump's guarded
  `applyLocked`) and returns `[]`; without one it returns the patches as
  before. No host change: both natives apply `[]` as a no-op and their
  listeners already hop to the main thread; their comments,
  `mobile/bridge.go`, `docs/platforms/native.md` and the regenerated API
  pages now say the listener is also called inside a `Trigger*` call.
  - **Pins** (`render/dispatch_order_test.go`): a forced gap (the "host"
    holds the dispatch's return until a goroutine write has been pushed)
    read `[2 1]` before the fix and `[1 2]` after; and 2,000 taps whose
    handlers also write from a goroutine never deliver a lower counter after
    a higher one (it failed before, at arrival 443 of 2,039).
  - `TestConcurrentDispatchAndTimerPushesDoNotRace` now drains its
    16-slot listener while the dispatchers run: a full buffer blocked inside
    the lock, which the listener contract forbids a real host to do.
  - `ios/verify/run.sh` green (9 patch batches, tree matches Go's final
    render); Android's transcript attaches no listener, so it is unchanged.
- **N-131** · raised `2026-1111-0120-reliability-performance-convenience-plan` · value medium
  · closed 2026-11-11, `2026-1111-1020-n131-n132-web-push-deadlock-and-dispatch-order` — editor
  toolbar commands no longer deadlock the Go runtime on the web (plan R1), and
  a program that dies shows a banner (R9a). Reproduced first on HEAD with a
  scratch CDP probe of the live tutorial: Indent on 4.13 printed `fatal
  error: all goroutines are asleep - deadlock!` and every later call threw
  "Go program has already exited".
  - **Layer 1, JS:** every runtime call into Go goes through one
    `dispatchToGo` (the 23 direct `GoInvokeCallback` sites rerouted). A call
    made while a batch is applying, or while Go's push is on the stack
    (`GrMobApplyPatches` now calls `GrMob.applyPushed`, which counts it), is
    queued and sent at the outermost batch's end: at once when no Go frame is
    below, in a microtask when the batch was pushed (the lock is held until
    the push returns). `dispatchFromElement` still drops surgery-caused
    events.
  - **Layer 2, Go:** `webhost.PushGuard` (no build tag, unit-tested): the
    listener delivers under it; `renderAgain` returns `[]` and `receiveEvent`
    defers and logs while it is up. All three browser hosts use it (webhost,
    `wasm/main.go`, the shots host); `TestEveryBrowserHostDeliversUnderThePushGuard`
    holds them to it. Checked alone in the live build (JS queue removed):
    presses held, deferral logged.
  - **R9a:** `GrMob.watchExit(exited)` draws a fixed `role=alert` "The app
    stopped… Reload" banner when go.run's promise settles, unless
    `GrMobWASM.Shutdown` (wrapped) was asked for. Wired in `wasm/index.html`
    and the `grmob new` template. Seen in a headless screenshot.
  - **Pins:** codeeditor_test.mjs (a pushed command reaches Go only after
    the push, a page-path one at the batch's end, a throw does not strand
    the queue, a census of direct calls; each fails with the queue
    disabled), stopped_test.mjs, pushguard_test.go, and browser check 25
    (4.13 Indent and 4.14 Bold on the site page, Go's selection line and
    Markdown as witnesses, no banner). Full `wasm/verify/run.sh` green,
    including the 261-view overflow sweep. The check-tally regex in
    checknumbering_test.go now takes any number of one-off kinds.
- **N-121** · raised `2026-1020-1656-stock-ticker-widget` · value medium
  · closed 2026-10-31, `2026-1031-2336-n121-stock-ticker-natives` — `comps.StockTicker` and lesson 4.42 seen on all four
  targets, stepping and gliding, with three defects found and fixed on the
  way.
  - **Web** (headless Chrome, a scratch CDP probe on the shots host): the
    stepping Scroll is `role=group` with five `button` quotes whose labels are
    the spoken sentences; gliding, two copies with `15s linear infinite
    grmob-glide`, the second `aria-hidden`, both at 73 px/s (1101px / 15s),
    seam 0px, gap 24px within a copy and 24px across the seam; a real mouse
    click on the hidden copy's AAPL read "OnTap got Apple (AAPL)"; under
    `dir="rtl"` `--grmob-inline` is −1 and the tape moves right at 72 px/s;
    `prefers-reduced-motion: reduce` holds both copies at rest and they move
    again when it lifts; Pause gives back the Scroll (overflow-x auto, 1081
    of 296px).
  - **Compose** (API 36 emulator): the stepping tape keeps each quote's width
    (TSLA 499px, AMZN cut at the clip) and pans under a swipe when paused.
    Gliding, the two copies were drawn over each other (N-130's cause); after
    the Row fix the band is one layer, gap 63px (24dp) within and across the
    seam, both copies at 189 px/s (2835px / 15s), a tap past the first copy's
    end read "OnTap got Apple (AAPL)", a zero animator scale holds the tape
    at rest with AAPL one gap inside the clip and the same spot every time
    (so `MotionDurationScale` does reach the node's coroutine context), and
    it moves again at scale 1; under a per-app Arabic locale it glides right
    with the quotes mirrored. TalkBack (the logcat harness): every quote
    heard as "Nvidia, NVDA, 120.43, up 2.86, 2.43 percent, Button" and the
    tape as "Stock prices"; a Tab walk of the stepping tape visited two of
    five quotes before leaving the tape, and a Shift+Tab walk heard all five,
    not diagnosed.
  - **SwiftUI** (iOS 26.5 simulator, new `TutorialStockTickerUITests`, 4
    tests passing): the stepping tape steps within 4s and pans on a swipe
    when paused; gliding, the labelled copy moves left at one quotes-span per
    15s, a tap 119pt past AMZN's end reads "OnTap got Apple (AAPL)" (the
    hidden copy), Pause leaves it still; under forced RTL it moves right at
    the same speed; with the simulator's Reduce Motion on it holds still
    with AAPL 24pt inside the clip. The stepping tape was one combined
    Button named "Stock prices" with no quote inside it: a labelled Scroll
    was not a labelled container type, so it always took `.combine`. Fixed:
    `Scroll` joined `GrMobNode.labelledContainerTypes` and GrMobScroll reads
    `containerStyle`; the tape is now a ScrollView holding one Button per
    quote, and the ios/verify census gained StockTicker (both forms,
    contain) plus Wizard's body Scroll and StepIndicator's strip, which were
    merged elements before and now contain their members. VoiceOver itself
    is still unheard (no VoiceOver on the simulator).
  - **Keys**: a step is not a move anywhere; the reconciler replaces every
    slot whose key changed (N-129, docs corrected).
  - Not done: a real finger on either native; VoiceOver's speech.

- **N-120** · raised `2026-1020-1656-stock-ticker-widget` · value low
  · closed 2026-10-31, `2026-1031-2250-core-glide-looping-translate-n120` — `core.Glide(periodMs)` is the looping translate: a
  node slides its own width toward the leading edge every period and
  restarts from rest, so two copies side by side read as one endless tape
  (the one distance at which a translation's loop closes, which is why it is
  not a parameter). Mapped on all four targets like Spin: an `animation`
  entry plus `core.GlideKeyframes` on the web (both targets, pinned to each
  other by wasm/verify), a `GlideNode` graphics layer on Compose, and a
  `GrMobLoop` TimelineView on SwiftUI that now carries Spin too. It replaces
  Translate on its node, mirrors under RTL, and — the opposite of Spin —
  stops under reduce-motion on every target (a second `@keyframes` under the
  media query; `MotionDurationScale`; `accessibilityReduceMotion`).
  `comps.StockTicker{Glide: ms}` is the consumer: a clipping box with two
  gliding copies, the second hidden from readers but tappable; Glide zero
  is the stepping tape, so a paused glide can be panned. Lesson 4.42 has a
  "Glide the tape" switch. Pinned by core, htmlout, wasm/verify (.mjs and
  the constant), mobile/verify and comps tests; Kotlin compiles
  (`compileDebugKotlin`) and the Swift runtime typechecks. Not yet seen
  moving on a device or in a browser (that check joins N-121).
  **core has no looping translate, so a ticker cannot glide.**
  `comps.StockTicker` steps instead: the caller advances `Start` on a
  `hooks.UseIntervalWhile` counter, and keyed quotes are reordered.
  `core.Spin` is the only looping animation, and its doc names a second
  consumer as the trigger for a general loop. A ticker is a candidate, but
  nobody has asked for a gliding one. `core.ScrollIntoView` is not a
  substitute, because it is one app-wide command and would pull the page
  back to the tape. Related to N-113 (paint transforms).

- **N-128** · raised `pbstudy_mobile 2026-1023-1530-ipad-first-writing-desk` (its N-015) · value high
  · closed 2026-10-23 — the web RichTextEditor now sends its document in Go's
  exact spelling (`richWireJSON` in wasm/grmob-runtime.js), and decides
  "already showing it" over both sides respelled. Pinned from both ends:
  three tests in wasm/verify/richtext_test.mjs and
  `TestJSONMatchesTheWebEditorsSpelling` in richtext. `wasm/verify/run.sh` is
  green. Re-checked in pbstudy_mobile's preview: typing after a template's
  headings, and between them, stays in order.
  **In the browser, typing in a rich document with a heading, a list, a quote
  or a blank line reset the caret on every keystroke.** The echo guard
  compares strings, but the DOM serializer built blocks as `{r, k}` and blank
  lines as `{r: []}`, while richtext marshals `{"k":…,"r":…}` and `{}`, and
  escapes `<`, `>` and `&` (encoding/json's HTML-safe form). So Go's echo never
  matched what was queued, every echo landed as a rewrite, and text typed after
  a heading came out reversed at the top of the document. The natives compare
  parsed documents and were never affected. A plain-paragraph document spells
  the same either way, which is why the existing echo test passed.
- **N-127** · raised `2026-1023-1428-bible-study-widget` · value low
  · closed 2026-10-23, `2026-1023-1438-n127-bible-study-gutter` — every verse row
  now leads with a number gutter sized for the passage's widest number
  (0.62 em a digit at the body size, BulletList's estimate), with the number
  right-aligned; an unnumbered line keeps an empty gutter. Seen in the 4.44
  browser shot.
  **BibleStudy's verse text does not line up across one- and two-digit
  numbers.** Each verse row puts its number and then the text, which starts
  wherever the number ends. So verse 1's text sits a few pixels to the left
  of verse 2's, and in a passage that runs from 9 to 10, verse 10's text
  starts further right again. A printed Bible hangs its numbers in a fixed
  gutter. A fixed width sized to the passage's widest number, or a
  core-level min-width on the number Text, would fix it. Seen in the 4.44
  browser shot; nobody has complained.
- **N-109** · raised `2026-1011-2237-v0.6.0-release-on-red-ci` · value high
  · closed 2026-10-21, `2026-1021-1132-work-next-list-n110-n109` — check 24 failed only on CI's wider Linux
  fonts. Reproduced on macOS by swapping the site page's face for Verdana
  (the same eight findings, plus one). Two causes:
  - A growing Text or Column's flex floor was its longest word
    ("core.AccessibilityKeyShortcuts", "UseStyle(Style{BorderRadius:"), so
    BulletList items and 7.2's CheckboxRow column spilled their Row. Now
    any FlexGrow > 0 with no stated MinWidth gets `min-width: 0` (the
    runtime's `styleFromGrMob` and htmlout's `ZeroesFieldFloor`), which
    matches a Compose weight. The word breaks under `overflow-wrap:
    break-word`.
  - 7.4's title row could not fit "McGrMob" at Title size beside the PRO
    badge. That row now wraps.

  Checked with `wasm/verify/run.sh` passing under Verdana and under the
  system face (258 lesson views). New tests in htmlout
  (`TestAGrowerGivesUpItsLongestWord`) and styleless_test.mjs.
- **N-110** · raised `2026-1011-2237-v0.6.0-release-on-red-ci` · value high
  · closed 2026-10-21, `2026-1021-1132-work-next-list-n110-n109` — `go test -race ./comps` failed on the
  EditableGrid harness, not the grid: the blur grace's commit calls
  `OnChange` from `UseTimeoutWhile`'s timer goroutine, and the test read
  `rows`/`changes` after a `time.Sleep`. `gridHarness` now holds a mutex:
  OnChange writes under it, each pass snapshots `rows` under it, and
  assertions read `reported()`, a locked copy. Checked with `go test -race
  -count=5 -run TestEditableGrid ./comps/` and a clean `go test -race ./...`,
  plus the steps CI was skipping (docs server build, WASM build).
- **N-125** · raised `2026-1021-1113-shopping-cart-library` · value low
  · closed 2026-10-21, `2026-1021-1122-n124-n125-promo-code-async-and-text` — the refusal was drawn as a raw,
  lowercase Go error string. `comps.PromoCodeField` now sentence-cases a
  refusal (first letter only) and takes `ErrorText func(error) string` to
  replace that; `cart.ErrUnknownCode` keeps its lowercase text so it still
  composes and compares with `errors.Is`. Decided: the cart's other English
  defaults stay per-field overrides, the way every comps widget localises,
  with no cart-wide string bundle. Tested in `comps` and lesson 4.43.
- **N-124** · raised `2026-1021-1113-shopping-cart-library` · value low
  · closed 2026-10-21, `2026-1021-1122-n124-n125-promo-code-async-and-text` — a promo code could only be checked
  synchronously. `comps.PromoCodeField.OnApplyAsync(code, done)` returns at
  once; until `done`, the input and Apply are disabled and Apply reads
  `PendingLabel` ("Checking…"). `done` may come from any goroutine and only
  its first call counts. The documented server recipe dispatches
  `cart.ApplyCode` once the server accepts. Race-tested in `comps`.
- **N-112** · raised `2026-1017-2211-gallery-widget`
  · closed 2026-10-20, `2026-1020-1933-core-swipe-gesture` — core had no swipe gesture. Added in 5e93a45:
  `core.OnSwipe` and `core.OnVerticalSwipe`, a text callback reporting
  `SwipeLeading`/`SwipeTrailing` (resolved for RTL on the host) or
  `SwipeUp`/`SwipeDown` once at release, with no stream of deltas. Every
  host applies one rule: at least `core.SwipeMinDistance` (40) along the
  axis and strictly further along it than across. Each prop claims only its
  axis. Wired on the web (pointer capture, touch-action, the trailing click
  swallowed), Compose (a slop-gated drag detector at RenderNode) and
  SwiftUI (a simultaneous DragGesture at RenderNode). `comps.Gallery`'s
  stage steps on it. Pinned by `core/swipe_test.go`,
  `wasm/verify/swipe_test.mjs` and `TestGallerySwipeSteps`. Seen working in
  headless Chrome with real mouse input. The natives are N-122.
- **N-119** · raised `2026-1019-2343-ios-filled-image-frame-and-screen-fill-scroll`
  · closed 2026-10-19, `2026-1019-2343-ios-filled-image-frame-and-screen-fill-scroll` — `Screen{Scroll, Fill}` left a white band
  below a short page on SwiftUI. Fill grew the column inside the Scroll but
  not the Scroll. SwiftUI measures a Scroll in a column at its content
  height and reads its viewport off its own size, so the minimum was the
  content again. The browser (host CSS) and Compose made the Scroll full
  height anyway. Fixed in e5b0e80: with Fill the Scroll gets FlexGrow(1),
  as a Footer already gave it. Trees without Fill are unchanged.
  `TestScreenFillWithScrollGrowsTheScrollRegion` pins it. On the simulator
  the background reached the bottom; unchanged on an Android emulator.
- **N-118** · raised `2026-1019-2343-ios-filled-image-frame-and-screen-fill-scroll`
  · closed 2026-10-19, `2026-1019-2343-ios-filled-image-frame-and-screen-fill-scroll` — A ContentModeFill Image widened its box on
  SwiftUI. GrMobImage's `frame(maxWidth: .infinity)` had no minimum, so it
  reported max(proposal, child), and `scaledToFill` reports the filled size.
  Gallery keeps every card in the tree, so one wide photo (500×306, filled
  to a 424pt stage, about 693pt) laid the whole screen out twice the
  iPhone's width. Fixed in 379ddca: a zero minimum on stated axes, with the
  clip moved onto the frame. On the simulator the screen fit. No contract
  test pins it; it was seen on the device.

- **N-117** · raised `2026-1019-2323-android-image-agent-and-scroll-strip-floor`
  · closed 2026-10-19, `2026-1019-2323-android-image-agent-and-scroll-strip-floor` — On Compose a horizontal Scroll in a Row
  overflowed instead of scrolling. `rowChildWidth` floors an unweighted Row
  child at min-content (CSS `min-width: auto`), but CSS makes that 0 for a
  scroll container, and `horizontalScroll` forwards intrinsics to its
  content, so the floor was the whole strip. `comps.Gallery`'s thumbnails
  were drawn over both arrows. Fixed in 6e820c8: RowChildren passes
  `scrollContainer = child.type == "Scroll"`, which drops the floor.
  `TestComposeDoesNotFloorAScrollContainerInARow` pins it. On the Fold6 the
  strip sat between the arrows and swiped.
- **N-116** · raised `2026-1019-2323-android-image-agent-and-scroll-strip-floor`
  · closed 2026-10-19, `2026-1019-2323-android-image-agent-and-scroll-strip-floor` — Android images 403'd at Wikimedia. Coil
  fetched through OkHttp with its default `okhttp/4.x` User-Agent, which
  Wikimedia refuses ("Please set a user-agent and respect our robot
  policy"), so Commons photos were blank on Android only. Fixed in c2e98ad:
  a new `GrMobApplication` in the shell manifest is Coil's
  ImageLoaderFactory and sets `<label>/<version> (<appId>; Android <rel>)
  grmob` on requests that carry no agent. `mobile/verify/image_agent_test.go`
  pins it. On the Fold6 the photos loaded.

- **N-115** · raised `2026-1019-2246-gallery-tapzone-zindex`
  · closed 2026-10-19, `2026-1019-2246-gallery-tapzone-zindex` — `comps.Gallery`'s tap zones never took
  a tap on the web. Every card layer carries `core.ZIndex` (`zOrder`, up to
  n+2), but the tap-zone Row had none, so the cards painted over it and the
  picture took every click: tapping the photo didn't flip it and tapping
  the edges didn't step. Found building a cat-adoption app on Gallery.
  Fixed in dcf9274: `tapZOrder()` gives the Row n+3, and
  `TestGalleryTapZonesPaintAboveTheCards` covers all four transitions from
  both ends of the deck. Clicking the photo flipped the card in Chrome.

- **N-058** · raised `2026-0919-1254-fold6-talkback-hid-harness-focus-after-navigation-inert-named-controls`
  · closed 2026-10-11, `2026-1011-2105-next-list-n058-talkback-follows-keyboard-focus` — TalkBack did not follow Tab on Compose. Compose's
  focus was right; TalkBack was not told. Compose 1.7.6 sends
  TYPE_VIEW_FOCUSED (the event TalkBack follows) only for a node it already
  had in its previous 100ms batch, and its batches hold only on-screen
  nodes. So a Tab onto a control below the fold was never announced: on the
  emulator, 4.18's ☰ and ‹ Prev, and most Tabs on 4.34 and 4.16. Two more
  ways it was lost:
  - TalkBack ignored a TYPE_VIEW_FOCUSED that came mid-scroll (4.34's poll
    options).
  - TalkBack moved its own focus to visible prose as its node scrolled
    away, which cut Compose's bring-into-view short. On 4.9, ‹ Prev was
    left 745px off screen.
  That was the "Enter pressed a different node from the one just spoken".
  The fix is the new `GrMobFocusAnnouncer.kt`, run for GrMobRoot and for
  every Modal. It acts only with touch exploration on and out of touch
  mode. It holds the newly focused node until TalkBack's own focus is on it
  (asked of Compose's provider). It scrolls an off-screen target on through
  its scroller's ScrollBy action, as ACTION_SHOW_ON_SCREEN would. Then it
  sends the TYPE_VIEW_FOCUSED Compose would have sent, with one retry 1.2s
  later. After (emulator, TalkBack 16):
  - 4.18: Copy → Open navigation → ‹ Prev → Next › → ‹ Contents.
  - 4.34: every Tab spoke its control: the poll's Tabs, Spaces and
    "Whatever gofmt says", then ‹ Prev.
  - 4.9: Copy, ‹ Prev and Next › after the date field.
  - 4.35: TreeView's src, assets and README.md. Enter on the Wizard's Next
    and Skip pressed the control just spoken.
  - The DatePicker's Modal, opened from the keyboard, speaks "Service date"
    then "Clear, Button", where Compose put focus.
  - With TalkBack off the Tab order is unchanged.
  Still heard: when TalkBack's node scrolls away, TalkBack may first read
  what it moved to ("Amount, row 2" on 4.37) before the correction. A long
  scroll's correction can take over 2s. Once in four 4.34 walks, ‹ Prev went
  unspoken. android lint passes; mobile/verify pins it
  (focus_announce_test.go). Not tried: Samsung TalkBack on the Fold6.
  `394bf92`.

- **N-108** · raised `2026-1010-2207-next-list-n068-wizard-status-n108-compose-live-text`
  · closed 2026-10-10, `2026-1010-2207-next-list-n068-wizard-status-n108-compose-live-text` — found by N-068's Wizard check: on
  Compose a live-region Text (RoleStatus, RoleAlert, RoleLog) was silent
  when its text changed. A probe Text counting every 5s went unannounced
  for 32s; the same Text with its words as its label was announced every
  time. TalkBack hears a live region's description change, not a Text's
  text change. GrMobText now gives a live-region Text with no label of its
  own its content as its content description. After: the plain probe was
  heard at every change, and so was the Wizard's step line. android/verify
  passes, and mobile/verify pins it (liveregion_test.go). `f8a4a5f`.

- **N-107** · raised `2026-1010-2154-next-list-n002-android-checks-n107-modal-capped-centre`
  · closed 2026-10-10, `2026-1010-2154-next-list-n002-android-checks-n107-modal-capped-centre` — found by N-002's "MaxWidth where it
  binds" check. On Compose a capped child of a Modal that states its own
  Width was left at the start of a wide window: 6.7's ActionSheet card and
  4.17's Menu panel (Width 100% under a 520px cap) sat at the bottom-left in
  landscape, where the web centres them. `centreCapped`'s `wrapContentWidth`
  centres only inside a minimum forced by the column's `fillMaxWidth`, which
  `hugsContent` withholds from a child with a Width. The column now also
  aligns such a child to the centre, unless it states its own AlignSelf.
  Both panels now span 517–1882 of 2400px. android/verify passes, and
  mobile/verify pins the alignment. `96f6154`.

- **N-106** · raised `2026-1010-2139-next-list-n002-ios-looks-n106-native-view-identity`
  · closed 2026-10-10, `2026-1010-2139-next-list-n002-ios-looks-n106-native-view-identity` — found by N-002's Reduce Motion check:
  comps.Drawer's documented slide never ran on either native. The iOS
  recording went from shut to open in one 60fps frame; the emulator at 10×
  animator scale was fully open in the first frame. Each renderer expressed
  a flipping flag as a branch around the content, so the subtree was built
  again instead of updated. That restarted its animation at the target, and
  on every open and close it also lost any state in the screen behind.
  Three fixes:
  - iOS `grMobAccessibility`: AccessibilityHidden is now an argument, not an
    if/else arm.
  - iOS `GrMobEscapeClaim`: the condition is inside the background.
  - Android RenderNode: one CompositionLocalProvider call site, with an
    empty provider when nothing changes.
  Results:
  - iOS needed both of its fixes. With both, the recording shows the slide
    (seven intermediate positions over about 300ms), and it still snaps
    under Reduce Motion.
  - Android's 10× frames show the panel and scrim moving step by step.
  - All 13 Tutorial UI test classes pass (55 tests), and ios/verify (with
    the WMO guard), android/verify and mobile/verify pass, with new pins in
    viewidentity_test.go.
  `3f9e9c1`.

- **N-104** · raised `2026-1010-2039-next-list-n004-rename-n065-gboard-n064-n104-compose-shadow-alpha`
  · closed 2026-10-10, `2026-1010-2039-next-list-n004-rename-n065-gboard-n064-n104-compose-shadow-alpha` — found by N-064's shadow check: on
  Compose an Opacity below 1 cut a box's shadow off entirely. The alpha's
  offscreen buffer is the node's size, and the shadow was drawn inside it.
  A box with both now takes the alpha inside the shadow.
  - API 28+: the shadow on its own layer, tinted
    `DefaultShadowColor.copy(alpha = layerAlpha)`, with `.alpha` inside it
    (fill, border and content one faded group). At 0.5 the shadow darkens
    the panel by 11 against the opaque card's 23, which is linear, as CSS
    is.
  - Below 28 (no shadow colours): the alpha shares the shadow's layer,
    alpha squared, so a quarter shadow. That still beats none.
  - Left as the platform's: an elevation shadow shows faintly through a
    translucent box (228 inside a 0.5 card where CSS gives 238).
  mobile/verify's opacity pins are updated plus a new one, and android/verify
  passes. `c864d21`.

- **N-103** · raised `2026-1010-2027-next-list-n097-backup-restore-n028-rotation-n103-ios-insets`
  · closed 2026-10-10, `2026-1010-2027-next-list-n097-backup-restore-n028-rotation-n103-ios-insets` — found by N-028's rotation check:
  `core.Window.Insets` was always zero on iOS. Lesson 4.21's new Insets
  line read "top 0, bottom 0, left 0, right 0" in both orientations on an
  iPhone 17 Pro. AppWindowReader read them off the size reader's
  `geo.safeAreaInsets`, but that reader ignores the safe area (to measure the
  whole window), which consumes the insets its proxy reports. They now come
  from UIKit: `GrMobWindowInsetsProbe`, a clear full-window UIView, reads
  `window.safeAreaInsets` on moving to a window, a safe-area change and
  layout. A change of size or insets sends the latest of both. After: top 62
  and bottom 34 in portrait; top 0, bottom 20, left 62 and right 62 in
  landscape (Safari on the same simulator reported 0/62/20/62, N-088).
  `TutorialWindowUITests` passes twice; mobile/verify pins the UIKit read and
  refuses `geo.safeAreaInsets`; core.SafeInsets' doc names the new source.
  `7145cec`.

- **N-102** · raised `2026-1010-2011-next-list-n090-mirrored-canvas-text-n102-ios-keyboard-reveal`
  · closed 2026-10-10, `2026-1010-2011-next-list-n090-mirrored-canvas-text-n102-ios-keyboard-reveal` — found by N-072's iOS keyboard check: a
  GrMob field low on a scrolled page opened under the soft keyboard on iOS
  (4.37's row 4: editor at y 756, keyboard top at 583, not hittable). The
  field is UIKit, and SwiftUI's ScrollView scrolls only its own TextField
  into view, which core.KeyboardAware's doc had counted on.
  `GrMobKeyboardReveal` (GrMobTextInput.swift) remembers the focused field
  (the coordinator's began and ended) and the keyboard's frame. On the
  keyboard's did-show or did-change-frame, or on focus with the keyboard
  already up, it scrolls each ancestor UIScrollView that has height to give.
  Each scroll is the least that puts the field, plus 16pt, above the keyboard
  and inside the insets; a horizontal strip is skipped. After: the editor at
  y 502, whole. `testTheLastGridRowsEditorIsAboveTheKeyboard` checks it with
  the soft keyboard on and skips with the reason under a hardware keyboard
  (the simulator's default). TutorialRoundFourUITests and the 2.3 caret test
  pass both ways. mobile/verify pins the wiring, and KeyboardAware's doc
  names the iOS reveal. GrMobCodeEditor and GrMobRichTextEditor (UITextViews
  with no begin-editing hook) are not wired. `e945bc9`.

- **N-090** · raised `2026-1003-1659-n007-canvas-text-clip-shape-taps`
  · closed 2026-10-10, `2026-1010-2011-next-list-n090-mirrored-canvas-text-n102-ios-keyboard-reveal` — seen on both natives, with a temporary
  canvas at the top of lesson 1.4 (reverted, never committed). It was a
  200×60 `CanvasMirrorsRTL` canvas: ticks at x 20, 100 and 180, a start-corner
  bar, and "start", "mid" and "end" texts aligned start, middle and end on
  them. It was drawn LTR and RTL:
  - on the API 36 emulator (per-app `ar` locale, reset after);
  - on the iOS 26.5 simulator (the forced right-to-left launch arguments);
  - in headless Chrome on its htmlout export (`dir="rtl"`).
  All three agree. Under RTL the bar moves to the top-right, "start" ends at
  its tick (the text's right-hand end, as core's doc says), "mid" stays
  centred, "end" begins at its tick, and the glyphs read normally on every
  target.

- **N-100** · raised `2026-1010-1955-next-list-n043-caret-burst-n091-natives-look-n100-align-self`
  · closed 2026-10-10, `2026-1010-1955-next-list-n043-caret-burst-n091-natives-look-n100-align-self` — found by N-091's look: Compose and
  SwiftUI ignored `core.Style.AlignSelf`. Discussion's thread line (an empty
  2px Box with `AlignSelf(stretch)`) was not drawn on either phone, and
  Link's `AlignSelf(start)` was dropped, so "Read on Blue Letter Bible" took
  taps across the whole card. Both natives now apply CSS's rule (the child's
  value overrides AlignItems, "" defers):
  - SwiftUI uses `GrMobFlexAlignSelf` with `GrMobFlexSolver.selfStretches`
    and `selfAlign`, for both the fill frame and the layout.
  - Compose uses `Modifier.align` or a fill, and pins a Row's height when a
    child stretches itself. That pin is skipped when a child cannot answer
    intrinsics (List, vertical Scroll).
  - A self-placed child (start, center or end) is fit-content across the
    line, as CSS sizes an unstretched item; on Compose that is
    `fitContentWidth`, on SwiftUI `fitCross`. Children with no AlignSelf are
    sized as before.
  After: the thread lines draw on both phones, and the link is 487px on
  Android (708 before) and 191pt on iOS (260 before). 6.7's ActionSheet and
  4.29's links are unchanged on Android. android/verify passes; ios/verify
  has the rule's cases; mobile/verify pins the parse, the arms and the
  bindings; 17 tutorial UI tests pass. The styling reference moves AlignSelf
  to the all-targets row. `99dc23e`.

- **N-091** · raised `2026-1003-1852-stripe-checkout-bible-verse-discussion-widgets`
  · closed 2026-10-10, `2026-1010-1955-next-list-n043-caret-burst-n091-natives-look-n100-align-self` — looked at on the API 36 emulator and the
  iOS 26.5 simulator (lessons 4.38, 4.39 and 4.40, screenshots and both
  accessibility trees).
  - StripeCheckout draws right on both: the summary, the Pay button, and the
    lock glyph sized to and aligned with the first line of its note.
  - BibleVerse draws right on both, loaded and failed; Retry hugs its label.
  - Discussion's header rows carry their labels on both ("Ben, reply to Ana,
    1h", "Chen, reply to Ben, 45m"). The Like chips size alike (a filled
    chip with the red emoji heart on Android, the white SF heart on iOS).
  - The 2px thread line was missing on both: N-100, fixed and seen.
  - Link's tap target spanning the card was the same cause.
  - Whether a reader repeats a header's inner name and time is unheard: N-101.

- **N-096** · raised `2026-1007-0210-n094-headless-window-minimum-not-overflow`
  · closed 2026-10-10, `2026-1010-1919-next-list-n096-export-viewport-n092-blb-proxy-n021-ios-image-floor` — every export now has a `<head>`
  (`documentHead`): `<meta charset="utf-8">` first, then a viewport of
  `width=device-width, initial-scale=1`, then motionStylesheet's conditional
  `<style>`s. The scaffold's `viewport-fit=cover` is left out on purpose: an
  export pads nothing by `env(safe-area-inset-*)`, so cover would run it under
  the notch in landscape. Measured with a scratch CDP probe on
  `examples/layout`'s export, 420px with mobile emulation: innerWidth 980 at
  scale 0.43 before, 420 at scale 1 after. The three "no head" tests now
  assert "no `<style>`", their real point; new tests pin the metas' order and
  keep viewport-fit out. `2c50935`.

- **N-092** · raised `2026-1003-1852-stripe-checkout-bible-verse-discussion-widgets`
  · closed 2026-10-10, `2026-1010-1919-next-list-n096-export-viewport-n092-blb-proxy-n021-ios-image-floor` — `blb.Proxy` (an `http.Handler`) and
  `Proxy.Forward` (the same without net/http) forward `blb.EndpointPath` to
  Blue Letter Bible. It is narrow on purpose, since it fronts a third party:
  fixed upstream and path, GET and HEAD only, only `id`, `style` and `target`
  (each at most 256 bytes), none of the caller's headers, and a reply over
  1 MiB refused rather than cut. `serve` mounts it in both modes, so a
  `./dev.sh` app gets it with no setup; the app sets `Client.BaseURL` to the
  page's origin. rweb's `Server.Proxy` was not used because it forwards
  everything. Checked live: curl through `serve` got John 3:16 (KJV), and a
  scratch wasm build in headless Chrome fetched "Psalms 23:1-2" (2 verses and
  the link). Lesson 4.39 stays canned (GitHub Pages is static); its prose and
  `docs/components.md` name the proxy. `2be6c39`.

- **N-021** · raised `2026-0916-1557-small-fixes-stroke-gradients-bar-values-alarm-groups`
  · closed 2026-10-10, `2026-1010-1919-next-list-n096-export-viewport-n092-blb-proxy-n021-ios-image-floor` — GrMobImage is no longer an AsyncImage. It
  loads with URLSession, decodes a UIImage, and records the natural size in
  `GrMobImageSizes` (@Observable, by src). `GrMobMinContent.imageFloor` is
  CSS's min(declared width, natural width carried through a points Height),
  falling back to the declared width while loading or on failure. A Row marks
  an image child squeezable (`GrMobGrow.squeezesWidth`), so its points Width
  is the flexible frame `relativeCap` uses, clamped by a points MaxWidth.
  Checked on the iOS 26.5 simulator with a temporary demo, not committed:
  three 200px padded Rows, each with an Image (110×40 box, 50×50 PNG) beside
  a pinned box of 180, 130 or 60. iOS drew 40, 40 and 108pt; Chrome's export
  of the same tree gave 40, 40 and 108px; the old runtime kept 110 in all
  three. TutorialNativeFloorsUITests 6/6, and 4.27's receipt loads through
  the new loader. ios/verify has six floor cases (mutation-tested), and
  mobile/verify pins the wiring. Not on a real iPhone. `dfd2ee6`.

- **N-089** · raised `2026-1003-1659-n007-canvas-text-clip-shape-taps`
  · closed 2026-10-10, `2026-1010-1749-next-list-validate-sort-n089-ios-verify-macos-arm` — `GrMobSurface.swift` now takes
  the shape the other UIKit runtime files do. `import UIKit` sits behind
  `canImport(UIKit)`, with AppKit as the macOS import. `GrMobSystemScheme`
  and `GrMobWindowStyle` are UIKit-only, and the `#else` arm is a
  `GrMobWindowStyle` stub that draws nothing (GrMobRoot names the type on
  every platform). `GrMobSystemScheme` has no macOS arm, since only
  App/AppWindow.swift names it and the App layer is checked against the
  iOS SDK alone. `grMobIsDark`'s luminance rule stays shared; only the read
  of the colour's components is per platform (UIColor `getRed` as before,
  NSColor in extended sRGB on macOS), so both passes compile the rule that
  must match Android. The iOS behaviour is unchanged. `ios/verify/run.sh`
  exits 0 with all 16 passes OK, including the view-layer typecheck, the
  `-O -wmo` Release guard and the iOS-SDK typecheck of Runtime and App,
  where it had stopped at `no such module 'UIKit'`. A throwaway macOS probe
  ran `grMobIsDark` on seven colours: #FFFFFF, #F2F2F7 and #BCBCBC (just
  over 0.5) read light; #1C1C1E, #2C2C2E, black and #2A78D6 read dark.
  Not run in the simulator or on a device. The file's header gained a
  "The macOS build" section.

- **N-095** · raised `2026-1003-2008-n093-gallery-lessons-checkout-verse-discussion`
  · closed 2026-10-07, `2026-1007-0216-n095-progress-track-border-role` — confirmed and fixed in `comps.ProgressBar`, not
  the tutorial. The default track was `Colors.Surface`, and DarkTheme's
  Card fill *is* Surface (#2C2C2E), so a stock bar in a stock dark Card had
  a 1:1 track: absent, not faint. The default is now
  `Colors.BorderColor()`, which `comps.Gauge`'s track already used. Measured
  against every `palette.Backdrops` fill of every bundled theme, the old
  default was 1.00:1 on each theme's Surface, on DarkTheme's Card and on
  Amber's Input and TextArea. Border's lowest is 1.13:1 (DefaultTheme on
  Surface). `TestProgressBarDefaultTrackShowsOnEveryBackdrop` holds the
  default to a 1.1:1 floor on every pair; it fails 7 pairs with the old
  default. Seen before and after in headless Chrome at a pinned 420px: the
  tutorial's card shape under DarkTheme, runtime-mounted. At 0% the old
  default drew the caption over nothing, and the new one draws a full-width
  #38383A groove. Light themes' tracks get darker too (DefaultTheme #F2F2F7
  → #E5E5EA), so `docs/images/tutorial-contents.png` is now a shade off; it
  was not retaken. `wasm/verify/run.sh` and `go test ./...` pass.

- **N-094** · raised `2026-1003-1852-stripe-checkout-bible-verse-discussion-widgets`
  · closed 2026-10-07, `2026-1007-0210-n094-headless-window-minimum-not-overflow` — not a bug: the premise was a headless Chrome
  artifact. `--headless=new --window-size=420,900` lays the page out at
  Chrome's minimum window width, 500px (`innerWidth` 500), while
  `--screenshot` saves a 420px-wide PNG of it. That crop cuts off the right
  80px, which is what looked like overflow. Re-run on 2026-10-07 with one
  `comps.Screen{Scroll: true}` holding a stock `comps.Card` and
  `comps.InputRow`. It was rendered both as `htmlout.ExportHTML` and as the
  runtime's JSON mounted by `GrMob.mount` in the `grmob new` host page, then
  measured over CDP with `Emulation.setDeviceMetricsOverride` pinning a true
  420px viewport. Both reported `scrollWidth` 420 and no element past the
  right edge, and the screenshot shows both widgets inside the frame. Real
  differences seen, neither an overflow: the export keeps `<body>`'s
  default 8px margin, and it has no viewport `<meta>` (raised as N-096). A
  check at a phone width should pin the viewport over CDP, as
  `wasm/shots/shot.mjs` does with its clip, not rely on `--window-size`.

- **N-087** · raised `2026-1003-0746-component-defined-readme-docs-skill`
  · closed 2026-10-07, `2026-1007-0204-n087-docsnippets-compile-component-docs` — `internal/docsnippets`. Its test takes the ```go
  fences under a fixed table of headings in the README ("Your own
  components"), `docs/concepts/components.md` (leaf widget, owned state,
  accessibility, testing) and `ai_docs/SKILL-component.md` (§2, §6, §8). It
  writes one package per doc into a scratch module that `replace`s grmob with
  this checkout, then runs `go vet` and `go test -v` there. A fence with no
  package clause gets one plus imports for the qualifiers it uses. The concern
  snippet is pasted into a method body on Tally. The concept page's
  `findFirst`/`findText`, which its prose describes but does not show, are
  supplied as `given_test.go`. `//line` directives make errors name the
  markdown line. The table's fence counts fail a renamed heading, and every
  `func Test…` a fence declares must PASS once per doc. Mutation-checked:
  a renamed symbol failed at `components.md:310`, an inverted reveal at
  `SKILL-component.md:436`, and a renamed README heading failed the count. It
  takes about a second. Not behind `-short`. Both docs now say the check
  exists. Also bumped wasm/verify's tracked-Go-file figure from 688 to 691;
  the N-083 commit's `hbox_test.go` had already left it stale.

- **N-083** · raised `2026-0924-1211-tutorial-overflow-sweep-web-floors-nested-scroll`
  · closed 2026-10-07, `2026-1007-0157-n083-hbox-paddingless-row` — added `core.HBox`, a Row with no theme base,
  as the horizontal twin of `core.Box`. The 2026-09-29 recommendation was to
  decline, on the grounds that a paddingless constructor would double the
  stack API. But `Box` has always been the paddingless Column, so only the Row
  half was missing. `HBox` emits an ordinary `Row` node: the theme base is
  applied in Go by the constructor and no renderer looks one up by type, so no
  target changed. The theme default stays as it is, so no app's layout moves.
  The existing `Padding(0)` sites were not migrated. An AST count found 66
  `core.Row`/`core.Column` calls with a literal `Padding(0)` argument (22 in
  examples/tutorial, 42 in comps) out of about 435, plus slice-built ones it
  does not see. The grep counts are larger (42 lines in examples/tutorial,
  256 in non-test Go) because they include other nodes. Pinned by
  `TestHBoxIsARowNode` and `TestHBoxCarriesNoThemeBase`. Row and Column
  gained doc comments pointing at HBox and Box. `docs/concepts/views.md`,
  `docs/concepts/components.md` and both `ai_docs/SKILL*.md` mention it, and
  `docs/api` is regenerated. Not seen on a device, since no renderer changed.

- **N-022** · raised `2026-0916-1557-small-fixes-stroke-gradients-bar-values-alarm-groups`
  · closed 2026-10-04, `2026-1004-2022-n022-zero-basis-two-pass-measure-ios` — the two-pass measure, in `GrMobFlexLayout`
  (Renderer.swift). The real gap at placement was a zero-basis Column child
  whose `min-height: auto` is the `.infinity` verdict: it kept its measured
  content height as its base, so a weighted Column was content-biased.
  `minMains` now measures that child's content and passes it as the minimum,
  so `baseMains` can start it at its padding. Under an ideal-size query,
  `sizeThatFits` still sizes the container from content (pass 1), then lays
  the children out inside that length from zero bases (pass 2, `zeroBased`),
  so the cross size is measured at the mains placement draws. Seen on the
  iPhone 17 Pro simulator with a throwaway app (deleted): a 300pt Column
  with two weight-1 zero-basis cells, 1 and 4 lines tall, went from 247/384px
  to 315/316px. Pass 2 made no visible difference there, because the outer
  Row measures the inner one again at a definite width; no screen was found
  that shows it. `TestIOSFlexHonoursAZeroBasis` pins both halves.
  `TutorialZeroBasisAndGradientsUITests` (calendar, StatTiles, gradients)
  and `TutorialChartsUITests` pass. Not seen on Android or the web, which
  were not affected. (was #27; lapsed@0917-1659)

- **N-093** · raised `2026-1003-1852-stripe-checkout-bible-verse-discussion-widgets`
  · closed 2026-10-03, `2026-1003-2008-n093-gallery-lessons-checkout-verse-discussion` — lessons 4.38 "Checkout with Stripe", 4.39 "Quoting
  a Bible passage" and 4.40 "Threaded comments", each with a test driving it
  through the app. 4.38 plays Stripe's page and the server's answer with two
  buttons (paid / declined) and shows the same minor units in USD, JPY and
  KWD. 4.39 uses three canned KJV passages in `blb.Passage`'s shape, with
  segments playing the fetch (loaded / loading / failed); the tutorial does
  not import `blb`, which would add net/http to the wasm build. 4.40 is a
  working thread with MaxDepth 2. Counts moved to 83 lessons and 40 in
  chapter 4 (README, `docs/tutorial-interactive.md`, `wasm/index.html`,
  `internal/shotclaims`), and `tutorial-contents.png` was retaken. Two
  fixes the work turned up: `comps.Discussion`'s indented replies column
  now has `MinWidth("0")` (a Composer one level down pushed the whole level
  past a 414px phone's edge on the web), and `wasm/shots/shot.mjs` emulates
  `prefers-color-scheme: light`, since the tutorial follows the system scheme
  and a dark-mode Mac took a dark shot. Seen in headless Chrome at 414px.
  Not seen on Android or iOS.

- **N-007** · raised `2026-0916-1032-clocks-canvas-alarm`
  · closed 2026-10-03, `2026-1003-1659-n007-canvas-text-clip-shape-taps` — all three added to `core.Canvas`:
  `Shape.Text` (`core.CanvasText`: anchor in viewBox units, size in layout
  units, start/middle/end × top/middle/bottom, start following a mirrored
  drawing's reading direction), `Shape.Clip` (a path, nonzero) and
  `Shape.OnClick`. Taps are hit-tested in Go (`core/canvas_hit.go`): hosts
  report "x,y,w,h" in the box to one `onShapeTap` text callback, shapes are
  tried topmost first and only those with a handler take part, a miss runs
  the canvas's own OnClick. Web: `<clipPath>` in the leading `<defs>`, text
  as `<g clip-path><text>` counter-scaled by `--grmob-canvas-ix/-iy` from a
  ResizeObserver (htmlout's static export leaves them at 1). Seen in lesson
  4.19's new week chart on Chrome (headless CDP), the iPhone 17 Pro simulator
  (new `TutorialCanvasTapUITests`) and the Android emulator: the same four
  taps picked Tue, missed on the clipped-off pill end, picked Thu, cleared.
  Not seen on a device: a mirrored canvas's text under RTL on the natives
  (checked on the web only). Text is not hit-testable.

- **N-088** · raised `2026-1003-1559-n061-n088-safe-area-insets-on-the-web`
  · closed 2026-10-03, `2026-1003-1559-n061-n088-safe-area-insets-on-the-web` — kept `viewport-fit=cover` and padded `body` with the
  four `env(safe-area-inset-*)` values (`wasm/index.html`). Measured on the
  iPhone 17 Pro simulator (iOS 26.5, Safari): portrait reports 0/0/0/0, so
  the bottom-edge worry was unfounded there and the rule is a no-op;
  landscape reports 0/62/20/62 (t/r/b/l), and before the fix the Dynamic
  Island hid the header's "Docs" link. After it the header clears the island
  and the page clears the home indicator; portrait screenshots unchanged.
  Dropping cover was rejected: Safari reports the 20px bottom inset without
  cover too, so the padding was needed either way. The cost is that the
  header's panel colour no longer reaches the side edges in landscape (the
  strips are the page background, as Safari's own letterboxing would be).
  Rotation was driven by a throwaway XCUITest setting `XCUIDevice`
  orientation; osascript has no assistive access here. `wasm/verify/run.sh`
  passes.
- **N-086** · raised `2026-1003-0453-n085-shell-surface-follows-go-tree`
  · closed 2026-10-03 — the candidate: a dark MaterialTheme scheme below
  GrMobRoot when the bars' colour is dark. `ShellMaterialTheme`
  (`GrMobSurface.kt`) wraps the tree in `MaterialTheme(darkColorScheme())`
  or `lightColorScheme()` and provides LocalContentColor white or black,
  from the same answer as the bar icons (`barsColor`, now shared with
  ShellSurface); a derivedStateOf, so only crossing the threshold
  recomposes. The light row is exactly the old defaults. MaterialTheme also
  provides a ripple, selection colours and bodyLarge as LocalTextStyle
  (read from the material3 1.3.1 bytecode), which would have changed every
  light app's press feedback and text metrics, so their outer values are
  provided again inside it. A Go accent's ink (Checkbox tick, Switch
  thumb) and a Button label on a Go Background with no TextColor are pinned
  to `OnGoColor` (the light scheme's onPrimary, white): the dark scheme's
  onPrimary drew a deep purple thumb on the demo's blue track
  (`TestAndroidInkOnTheAccentIsNotTheSchemes`). Seen on the emulator, demo
  with Screen painted #1C1C1E on a light system: dark tab row, white header
  and "Hello, stranger." ink, white tick and thumb. The unpainted demo by
  day is pixel-identical to before apart from the status-bar icons; the
  tutorial at night is unchanged except that an unchecked checkbox's
  outline is the dark scheme's (light grey, was near-invisible dark grey).
  `core.ColorScheme`'s doc names the chrome. Not covered: a dark SafeArea
  over an unpainted root composes light for one frame (claims join after
  the first composition), and the window theme (Theme.Material.Light)
  still styles the classic views (the rich-text EditText, Toasts).

- **N-085** · raised `2026-0929-0058-next-list-escape-heading-focus-dark-mode-ios-contain-grid-discard`
  · closed 2026-10-03, `2026-1003-0453-n085-shell-surface-follows-go-tree` — the first
  candidate: the shells take their surface and bar style from the Go tree,
  not the system. One rule on both (`GrMobSurface.kt`, `GrMobSurface.swift`):
  the bars' colour is the innermost painted SafeArea's Background, else the
  root's, else the shell page (#FFFFFF, DefaultTheme's Background); the
  surface is the root's Background or that page; dark bars' colour (WCAG
  luminance ≤ 0.5) means light icons. Android writes the window background
  and both bars' icon appearance from one `ShellSurface` composable composed
  after the tree; a painted SafeArea registers a claim instead of setting the
  icons itself (`SystemBarIcons`, which a root-level writer would have raced
  on a recomposition that skipped the SafeArea). The claim is read inside
  the SideEffect, which runs after the claims' DisposableEffects in the same
  apply, so the first frame is right. iOS paints the surface under the bars
  behind the root, carries the SafeArea's colour up as a preference
  (`transformPreference`, so the innermost wins) and sets the window's
  `overrideUserInterfaceStyle` from it — status bar and native chrome
  together, since an App cannot subclass SwiftUI's hosting controller for
  `preferredStatusBarStyle`. The override hides the system scheme from the
  colorScheme environment, so `AppWindowReader` now reads it from the window
  scene's traits (`GrMobSystemScheme`, watched with
  `registerForTraitChanges`), which overrides do not reach. Seen: the demo
  app (follows no scheme) on a dark iPhone 17 Pro simulator (iOS 26.5) and a
  night-mode emulator draws a white page with dark status-bar text / icons;
  the tutorial on iOS is dark with light status text on a dark system and
  switches live to light and back (so the scene read reaches Go through the
  override); on Android it is dark with light icons at night and light with
  dark icons by day; the demo with its Screen painted #1C1C1E on a light
  system gets light icons on both. `core.ColorScheme`'s doc says the scheme
  is the system's, not the shell's. The Material chrome left light on
  Android is N-086.
  - Note: the iOS half of the Renderer.swift wiring landed inside
    `c43e3bc` (another session's commit swept the working tree); the rest
    lands with this item's commit.

- **N-073** · raised `2026-0921-1118-comps-round-four-phase-5-editable-grid`
  · closed 2026-10-03, `2026-1003-0208-keyed-callback-ids-dark-theme-chart-data` — identity-keyed IDs. `core.Keyed`
  and each Navigator frame open an ID scope while their subtree renders
  (`Context.keyScope`), so its callbacks are numbered under the key path
  (`cb_r2/e0/1`) and the enclosing counters do not move; root IDs keep `cb_N`.
  Keys are escaped (`/`, `~`, `%`) and a key repeated in one scope is told
  apart by occurrence (`0~1`). ErrorBoundary's rollback now walks a trail of
  registrations and key counts. EditableGrid's EDIT transitions re-bind no
  other cell (`TestEditableGridEditMovesNoOtherCellsHandler`, which shows
  `cb_1`→`cb_6` without the scope); a stale ID from a popped frame reaches
  nothing; Navigator's own Pop stays outside the frame so two quick backs pop
  twice. No host parses IDs. Mutation-tested (`core/keyed_ids_test.go`).
- **N-074** · raised `2026-0921-1419-tutorial-light-dark-theme`
  · closed 2026-10-03, `2026-1003-0208-keyed-callback-ids-dark-theme-chart-data` — `core.DarkTheme` (a recoloured copy of
  DefaultTheme, `newDarkTheme`) is in `BundledThemes` and every census passes
  over it; palette.mjs gained its rows and the browser paints them. The four
  `*OnLight` fields are now `PrimaryInk`/`SuccessInk`/`WarningInk`/`ErrorInk`
  with no field alias (a deprecated duplicate would make "clear the tone when
  re-branding" a silent no-op); the resolvers and `OnLight` keep deprecated
  wrappers, and the reverse lookup is `AsInk` (`Variant.Ink` was taken). The
  tutorial's `darkTheme` is `core.DarkTheme`. Census figures replaced the
  hand-computed ones (Border 1.45:1, not 1.5).
- **N-018** · raised `2026-0916-1410-chart-palette-and-canvas-gradients`
  · closed 2026-10-03, `2026-1003-0208-keyed-callback-ids-dark-theme-chart-data` — `core.DarkTheme` spends
  `DefaultDarkChartColors` and `DefaultDarkSequentialColors`.
- **N-010** · raised `2026-0916-1129-charts-on-canvas`
  · closed 2026-10-03, `2026-1003-0208-keyed-callback-ids-dark-theme-chart-data` — per host, as recommended: no
  screen-reader-only primitive. `core.AccessibilityChart(core.ChartData)`
  puts the numbers on the chart's labelled RoleImg node (series of points,
  categorical or numeric x; NaN dropped). Web runtime and htmlout write a
  visually hidden `<table>` as leading chrome and the role as `figure` (an
  img's children are presentational); iOS builds an `AXChartDescriptor`
  (`GrMobChartAccessibility` on Row and Column); Android keeps the sentence.
  Every summarising chart carries it except Sparkline and Gauge. Held across
  languages by gen.go `chartCases` + `chart_test.mjs`; Chrome's own AX tree
  (CDP `getFullAXTree`) shows each figure's table with row headers and cells
  on lessons 4.20 and 4.36. The descriptor is unheard (N-070).
- **N-069** · raised `2026-0921-1035-comps-round-four-phase-3-structure`
  · closed 2026-09-29, `2026-0929-0058-next-list-escape-heading-focus-dark-mode-ios-contain-grid-discard` — `core.Focus` now reaches a node that
  takes no input focus, per host, with no new core API: the web makes it a
  programmatic target (`tabindex="-1"`, out of the Tab order) and focuses it;
  iOS moves VoiceOver's focus to a stamped Text (`AccessibilityFocusState`,
  once per epoch, attached only to Texts that carry a stamp); Compose has no
  way for a node to take TalkBack's focus, so it reads nothing. A Text takes
  style props only, so it is named by applying `FocusTarget` to the node it
  renders. `comps.Wizard` gained `TitleRef` (the caller's ref, as Drawer's
  CloseRef is) and focuses the title after every step change; lesson 4.35
  passes one. Seen in headless Chrome on 4.35: after Next the focused
  element is the "Gift note" heading and after Back "Your name"; without the
  tabindex it is the page body. iOS builds and is unheard (no VoiceOver on
  the simulator). Tests: `TestWizardTitleRefTakesFocusOnEveryChange`, a
  runtime test for the programmatic target.

- **N-057** · raised `2026-0919-1254-fold6-talkback-hid-harness-focus-after-navigation-inert-named-controls`
  · closed 2026-09-29, `2026-0929-0058-next-list-escape-heading-focus-dark-mode-ios-contain-grid-discard` — the layer-only shape: `core.OnEscape`, a
  claim of its own beside OnBack (Escape closes what is open over a screen
  and does not navigate, so a Navigator route and an AppBar's back arrow
  claim back and not Escape), with an open Modal's OnDismiss counting as a
  claim. `comps.Drawer`'s open panel carries `OnEscape(OnDismiss)`, so every
  Modal-based widget and the Drawer close on Escape.
  - Web: one window keydown listener (shared with the page shortcuts, which
    keynav_test pins to one) runs the last claimant in document order; a
    handler that consumed the key first (a combobox clearing its active
    option) wins. Seen in headless Chrome with CDP keys: 4.18's drawer and
    6.6's dialog close; both stay open with the listener removed. Four
    tests in browserback_test.mjs.
  - Android: the Activity walks the tree for the last claimant
    (`GrMobRuntime.lastEscapeClaim`). A Modal's Dialog is its own window and
    did *not* close on Escape (seen on the emulator: the platform maps
    Escape to back only with predictive back off, and this shell opts in),
    so the Dialog's content wraps its window's callback (`DialogEscape`).
    Seen: `input keyevent KEYCODE_ESCAPE` closes 4.18's drawer and 6.6's
    dialog, and back still closes the dialog.
  - iOS: unverified. The claim is an invisible Button with Escape as its
    keyboardShortcut (`GrMobEscapeClaim`). XCUITest's Escape reached neither
    it nor `.cancelAction`, while the same claim bound to Control+Option+E
    closed the drawer; the Mac's own Escape could not be sent (osascript has
    no keystroke permission here). `testEscapeClosesTheDrawerAndTheDialog` is
    a strict XCTExpectFailure, like F6's. A Modal has no Escape claim on iOS
    yet. Carried by N-004's real-iPad-keyboard check.

- **N-050** · raised `2026-0918-2310-mi-max-3-android-10-force-dark-theme-sweep`
  · closed 2026-09-29, `2026-0929-0058-next-list-escape-heading-focus-dark-mode-ios-contain-grid-discard` — the hosts report the system's scheme and
  the app picks. `core.Window` gained `ColorScheme` ("light", "dark", or
  empty when no host said) and `Dark()`, carried as `scheme` on the existing
  "window" report and deduped with it: Android reads the configuration's
  night bit (a switch recreates the Activity, which reports again), iOS the
  reader's `colorScheme` environment (watched, so a switch re-reports), the
  web `prefers-color-scheme` (with a change listener). Core still bundles no
  dark theme (N-074). The tutorial follows the window's scheme until a page
  sends its own "theme" choice, so the web's Light/Dark/System switch still
  wins there; on a native it paints the root in darkTheme's Background and
  ink, the colours the web page's CSS supplies. That exposed a real
  divergence: a container's TextColor was inherited on the web (CSS `color`)
  and not on the natives, so lesson 1.1's plain `core.Text` name and numbers
  drew Compose's black on the dark card. Now Compose provides
  `LocalContentColor` below a container with TextColor and SwiftUI sets the
  container's foreground style (`grMobInk`), documented on `core.TextColor`.
  Seen: the emulator (`cmd uimode night yes`) draws 1.1, 4.37 and 5.9
  readable dark and returns to light on `no`; the iOS simulator
  (`simctl ui appearance dark`) draws 1.1 dark and comes back live to light.
  Tests: `TestWindowHostEventDecodesTheColorScheme`,
  `TestTheSystemSchemeRulesUntilAPageSays` (mutation-tested), two
  window_test.mjs cases, and the four-shell spelling census gained the key.
  The surfaces behind apps that do not follow the scheme are N-085.

- **N-078** · raised `2026-0922-0440-next-list-radio-positions-browser-notify-fold-grid-scroll-ios-round-four`
  · closed 2026-09-29, `2026-0929-0058-next-list-escape-heading-focus-dark-mode-ios-contain-grid-discard` — decided by content, not role. A labelled
  container is `.ignore` when every child is hidden (as before), `.contain`
  when two or more *members* sit below it, and `.combine` otherwise. A member
  is a control type, a pressable node with something to announce (a label or
  a child not hidden), or a nested labelled container, which is Compose's
  merge boundary per clickable and per labelled node; the walk does not
  enter a member and stops at the second (`GrMobNode.holdsControls`,
  `grMobChildMode`). Nested named containers count so that Poll's results
  (named rows, no controls) are one stop per option and Wizard (one Next,
  a step's body) does not merge its step. PINInput's tap-catching row
  (unnamed, all children hidden) is not a member, so it still combines.
  `ios/verify`'s new census renders 21 bundled widgets in Go and holds every
  labelled container's shape to a table (`childmode.go` /
  `childmode.swift`; mutation-tested by removing the contain arm).
  On the simulator (iOS 26.5): "Budget" is a container whose every cell is
  a Button named "Amount, row 1, $1200.00" etc.; "Price" holds Slider
  "Minimum" $20 and "Maximum" $80; "Label colour" is not itself Selected,
  "blue" is. The round-four tests now reach cells, sliders and the ✕ by
  name; the range test drags the thumb, because XCUITest's
  `adjust(toNormalizedSliderPosition:)` reads the value as a percentage and
  did nothing on "$20". All 51 Tutorial UI tests pass (`testAudioPlayer`,
  which streams, failed once in the full run and passed alone). VoiceOver
  itself is still unheard: Accessibility Inspector or a device. Open
  questions left with it: Stepper's value is now read as its visible number
  between the buttons, the web's reading; a SliderRow's title Text and its
  slider both say "Minimum".

- **N-034** · raised `2026-0917-1935-pin-input-and-a-code-with-no-gaps`
  · closed 2026-09-29, `2026-0929-0058-next-list-escape-heading-focus-dark-mode-ios-contain-grid-discard` — half done, half declined on the session's
  recommendation (the user's decision; reopen to overrule).
  - `examples/signup` → `PINInput`: done. A passing submit now sends a code
    (the example states it, 246810, having no mail server) and a verification
    step takes six digits in `comps.PINInput` before "Account created". A
    wrong code is cleared with "That code is not the one we sent"; "Use a
    different address" returns to the form with its values kept. The step
    renders in `ctx.Scope("verify")` because PINInput holds hooks.
    `TestTheCodeStepRefusesAWrongCodeAndGoesBack`, and three tests now pass
    through the step. `docs/images/signup.png` is the form's mismatch error,
    which the step does not touch, so it was not retaken (its claims test
    passes).
  - `examples/chat` → `comps.MessageThread`: declined. The example exists to
    teach `core.For` and `core.Keyed`, which MessageThread hides, and the
    widget is taught by lessons 4.33 and 4.34 already; it also cannot grow to
    fill yet.

- **N-084** · raised `2026-0928-1917-next-list-overflow-check-ios-width-cap-sparkline-fade-reduce-motion`
  · closed 2026-09-28, `2026-0929-0058-next-list-escape-heading-focus-dark-mode-ios-contain-grid-discard` — the ✕ carries a new
  `core.PressKeepsFocus()`, a no-arg prop (`pressKeepsFocus: true` on the
  wire) that the web runtime turns into a `mousedown` preventDefault, so a
  press never blurs the field; the natives need nothing. Picked over a blur
  payload naming where focus went, which would put a node path in every blur
  for one widget. The ✕ also reports its own focus and blur as the field's, so
  Tab onto it is not a blur and Enter there discards. The fix exposed a second
  defect: the field was now focused when the discard removed it, and Chrome
  fires `blur` synchronously inside the removal, which re-entered Go with the
  removed field's positional ID (N-073's hazard, now firing): the next tap on
  the cell opened nothing. The runtime now drops any element event fired while
  it is mounting or applying a batch (`applyingTree`). Measured in headless
  Chrome with CDP's real mouse path on 4.37: 60, 120, 300 and 600ms holds all
  discard (before: 300 and 600 committed); Tab onto the ✕ outlasts the grace
  and Enter discards; return commits and lands on row 2; a 300ms tap on
  another cell commits and opens it. Pinned by
  `TestEditableGridDiscardIsOneFocusUnitWithTheField` and three runtime tests
  (the guard's mutation-tested).

- **N-081** · raised `2026-0924-1211-tutorial-overflow-sweep-web-floors-nested-scroll`
  · closed 2026-09-28, `2026-0928-1917-next-list-overflow-check-ios-width-cap-sparkline-fade-reduce-motion` — browser check 24. `wasm/verify/overflow.mjs` holds the
  sweep's reading (in the page) and its judgement (a pure function), and
  `overflow_test.mjs` reaches every rule with hand-built boxes. The check opens
  every lesson gen.go numbers from `tutorial.Chapters` (a new `lessons` field in
  the transcript) through the real site page's hash deep link, in the split at
  1280px and in the phone layout at 390 and 360, and fails on a box that
  escapes a parent that shows overflow, is cut off by one that hides it, runs
  past the screen's side, holds a word wider than itself, or is drawn smaller
  than its stated px size. Skipped: code editors, rotated layers, SVG
  internals, and inert subtrees (4.18's shut Drawer panel was the only finding
  on today's tree). 240 views in about a minute. Mutation-tested by reverting
  each fix of `2026-0924-1211` in turn: 1.1's stats row (escape), Avatar's
  FlexShrink(0) (squeezed 34 of 36 at 360), the field floors (Send and Show
  escape), the mount's overflow-wrap (6.8's and 4.11's text) and the nested
  Scroll rule (1.5 squeezed to 2px at all three views). Still first state only.
- **N-082** · raised `2026-0924-1211-tutorial-overflow-sweep-web-floors-nested-scroll`
  · closed 2026-09-28, `2026-0928-1917-next-list-overflow-check-ios-width-cap-sparkline-fade-reduce-motion` — `docs/images/tutorial-lesson.png` retaken with
  `wasm/shots/shoot.sh tutorial-lesson`; the card shows "Following" whole.
- **N-080** · raised `2026-0924-1211-tutorial-overflow-sweep-web-floors-nested-scroll`
  · closed 2026-09-28, `2026-0928-1917-next-list-overflow-check-ios-width-cap-sparkline-fade-reduce-motion` — seen on both natives, and SwiftUI had a defect.
  - Compose (the emulator at 411dp and, via `wm density 480`, at 360dp, reset
    after): 4.11's StaticMap frame takes a 318dp and a 266dp column and keeps
    its 180dp height; 1.1's avatar is round and the three stats sit inside
    the card; 1.5's short Scroll is 474px (158dp) inside its 1dp border; 4.3's
    avatars are round beside rows whose subtitle wraps.
  - SwiftUI (iPhone 17 Pro simulator, 402pt): 1.5's viewport is 158pt, 1.1's
    avatar 40×40, 4.3's 36×36. 4.11's map was **320pt wide at x 47, 13pt past
    its 306pt column** — `grMobDimension` drew a points Width as a rigid
    `frame(width:)`, which `GrMobMaxWidthLayout`'s narrowed proposal cannot
    shrink, and a percentage cap has no length to fold in. A points Width
    under a percentage cap is now a flexible frame (`minWidth: 0,
    idealWidth: w, maxWidth: w`, the `relativeCap` argument): 306pt, even
    margins. StaticMap is the only bundled node with that pair. Pinned in
    `TestSwiftCapsOutsideTheGrowFrame`; three new tests in
    `TutorialNativeFloorsUITests` (the map, the nested Scroll, the avatars),
    the map one seen failing on the old Swift at x 367 against 354.
  - The even crop under ContentModeFill is still unseen with a real picture:
    4.11's provider host is gone, which is the lesson's own subject.
- **N-025** · raised `2026-0916-1643-zero-basis-floor-area-fades-scatter-squares-android-look`
  · closed 2026-09-28, `2026-0928-1917-next-list-overflow-check-ios-width-cap-sparkline-fade-reduce-motion` — Sparkline's `Area` fades as AreaChart's does, 4D under
  the line's highest point to 0A at the bottom edge its area closes on
  (`areaFadeTo`, which `areaShape` now calls with the scale's zero), and keeps
  the flat 33 tint for a non-hex colour or no finite value.
  `TestSparklineAreaFadesTowardItsBottomEdge`; seen in headless Chrome on
  4.20. The natives already draw `LinearGradientFill` for AreaChart. (was #34)

- **N-016** · raised `2026-0916-1331-next-list-sweep-maxlines-chart-types-evenodd`
  · closed 2026-09-26 by the user, not done — the double-post claim was never
  reproduced in forty sessions of carrying it. A recurrence is a new item.
  (was #20)
- **N-036** · raised `2026-0918-0130-round-three-eight-widgets-and-what-a-text-node-cannot-do`
  · closed 2026-09-26 by the user, not done — a sixth low-hanging-fruit round
  with no candidates behind it. (was #46)

- **N-019** · raised `2026-0916-1410-chart-palette-and-canvas-gradients`
  · closed 2026-09-25, `/next-list` re-check. The one failure never came
  back: `2026-0923-1149` ran all Tutorial* classes, 48 tests with 0 failures,
  and that includes `TutorialChartsUITests.testChartsAreOneElementEachAndRedrawOnUpdate`
  (48 is the count of `func test` in ios/GrMobUITests/Tutorial*.swift). The
  original failure's reason was never captured, so a recurrence is a new item.

- **N-079** · raised `2026-0922-0440-next-list-radio-positions-browser-notify-fold-grid-scroll-ios-round-four`
  · closed 2026-09-23, `2026-0923-1149-next-list-slider-values-radar-centre-zero-basis-border-opacity-seen` — `comps.SliderRow` now states its readout as the
  slider's `core.AccessibilityValue` text (Format, or the default precision;
  nothing when Format returns ""). Both natives already honoured the text on
  any node (Compose's stateDescription, SwiftUI's accessibilityValue); the
  two web exporters' `ariaValue` gained a Slider arm that writes
  `aria-valuetext` alone, the input keeping its own value/min/max. Heard on
  the emulator's TalkBack: "$20, Minimum, Slider" and "$80, Maximum,
  Slider". On the simulator the combined "Price" element reads "$20, $80"
  and "$130, $130" after the push (`testSwatchesHexShortFormAndTheRangeThatCannotCross`
  asserts both); the inner sliders XCUITest synthesizes under the combine
  still say 10%/40%, which is N-078. Documented on `core.Slider`,
  `core.Style.AccessibilityValue` and docs/components.md.

- **N-077** · raised `2026-0922-0204-n076-logical-insets-hidden-fill-ios-carry-talkback-sweep`
  · closed 2026-09-22, `2026-0922-0440-next-list-radio-positions-browser-notify-fold-grid-scroll-ios-round-four` — Compose numbers a
  `selectableGroup`'s members by `layoutNode.placeOrder`, which restarts in
  every layout parent, so a grid of Rows was counted per Row (every heard
  number was one plus the count of lower places across both rows). Compose
  1.10's `setCollectionItemInfo` also overwrites a stated item info whenever
  the parent is a `selectableGroup`, so the radiogroup role no longer writes
  one: `RenderNode` states the group's `collectionInfo` and each radio's
  `collectionItemInfo`, counted in document order from the Go tree
  (`LocalGrMobRadioPositions`). Heard on the emulator: 5.9's swatches 1…8 of
  8 in order, and 4.16's RadioGroup "2 of 3". Pinned by
  `TestComposeStatesEachRadiosPlaceInItsGroup`.
- **N-029** · raised `2026-0917-0227-foldables-window-record-and-two-pane`
  · closed 2026-09-22, `2026-0922-0440-next-list-radio-positions-browser-notify-fold-grid-scroll-ios-round-four` — no person needed: CDP's
  `setDeviceMetricsOverride` takes a `displayFeature` and
  `setDevicePostureOverride` a posture, and browser check 23 drives lesson
  4.21 through a vertical hinge, the book posture, a tabletop hinge and none.
  It found that a segment change at an unchanged window size fires no
  `resize`, so the fold reached Go only on the next posture change; the
  runtime now also reports on the `(horizontal|vertical)-viewport-segments: 2`
  media queries. Mutation-tested by removing those listeners.
- **N-014** · raised `2026-0916-1157-next-list-charts-on-devices-tier-e-alarms-timezone`
  · closed 2026-09-22, `2026-0922-0440-next-list-radio-positions-browser-notify-fold-grid-scroll-ios-round-four` — the permission is granted over
  CDP (`Browser.grantPermissions`), so no person is needed. Browser check 22
  posts four notifications through the real runtime in headless Chrome and
  hears each real `Notification`'s "show": an immediate one at once, a
  scheduled one at its time, a sweep answering with that id alone, a
  swept timer never firing, another prefix's firing, and a second sweep
  reporting nothing. Mutation-tested by removing the sweep's `clearTimer` (the
  first version scheduled the swept post at +60s, which could never fire
  inside the check, and passed the mutant; it is now due between the sweep
  and the read).

- **N-076** · raised `2026-0921-2318-next-list-radio-arrows-readme-counts-theme-check-canvas-rtl-mirror`
  · closed 2026-09-22, `2026-0922-0204-n076-logical-insets-hidden-fill-ios-carry-talkback-sweep` — Left
  means leading, everywhere: the web now writes `padding-block`/`padding-inline`
  and `margin-block`/`margin-inline` (the runtime's `edgeLogicalCSS`,
  `htmlout.EdgeLogicalCSS`), matching Compose's `start` and SwiftUI's
  `leading`, and never a physical side for an inset; the code editor's gutter
  inset is `padding-inline-start` (the editor is `dir="ltr"`). The field names
  stay Left/Right, documented on `core.EdgeInsets` and the four side props.
  cssstyle.mjs models the logical pairs and refuses an element given both a
  physical and a logical name for one box; its CSSOM table gained six rows,
  replayed against Chrome by check 14. Browser check 21 now also reads a Left
  inset's computed side under each direction (mutation-tested by restoring
  the physical shorthand). BarChart's spacer Row is kept, its comment updated.

- **N-071** · raised `2026-0921-1057-comps-round-four-phase-4-charts`
  · closed 2026-09-21, `2026-0921-2318-next-list-radio-arrows-readme-counts-theme-check-canvas-rtl-mirror` — `core.CanvasMirrorsRTL`, an opt-in Canvas prop
  (`mirror: true` on the wire, only when set), reflects the drawing about its
  box's centre under RTL. Web and htmlout: `scale: var(--grmob-inline, 1) 1`
  on the `<svg>`, with `core.TranslateDirectionCSS` added on first use.
  Compose and SwiftUI: the viewport mirrored (`core.MirrorCanvasMapping`,
  restated as `CanvasViewport.mirrored` / `GrMobCanvasViewport.mirrored`)
  when the layout direction is RTL, held to `internal/canvasfixture`'s three
  mirrored cases by both native harnesses. Adopted by Line/Area, Bar
  (both orientations), Histogram, Scatter, Candlestick, Radar, Heatmap,
  Sparkline, Waveform and Rating's half star; Donut, Pie, Gauge, Funnel and
  QRCode stay fixed (`comps/canvas_mirror_test.go` is the census). Seen
  right under RTL in headless Chrome (4.20, 4.36) and on the Compose emulator
  with a per-app Arabic locale (4.20, 4.36's radar); browser check 21 pins
  the reflection, mutation-tested. SwiftUI type-checks and its geometry is
  verified, but it is unrun.
- **N-067** · raised `2026-0921-1019-comps-round-four-phase-2-inputs`
  · closed 2026-09-21, `2026-0921-2318-next-list-radio-arrows-readme-counts-theme-check-canvas-rtl-mirror` — a `radiogroup` now answers both arrow pairs on the
  web (Down/Right forward, Up/Left back, the horizontal pair mirrored under
  RTL, whatever the group's axis), in `handleCompositeKey`. One-axis
  composites still leave the cross pair to the page. Pinned in
  `wasm/verify/keynav_test.mjs` (dom.mjs only; not tried in a real Chrome on
  lesson 5.9).
- **N-063** · raised `2026-0921-0912-comps-round-four-phase-1-chat-family`
  · closed 2026-09-21, `2026-0921-2318-next-list-radio-arrows-readme-counts-theme-check-canvas-rtl-mirror` — README's chapter table restated (chapters 4, 5, 6
  were 14/6/5, are 37/9/8), and `examples/tutorial/readme_counts_test.go`
  now holds the table and both "N lessons across M chapters" sentences to
  `Chapters`. The plan doc's per-corner-radius entry is struck through.
- **N-075** · raised `2026-0921-1419-tutorial-light-dark-theme`
  · closed 2026-09-21, `2026-0921-2318-next-list-radio-arrows-readme-counts-theme-check-canvas-rtl-mirror` — browser check 20 (`wasm/verify/browser.mjs`)
  boots the site page at 1280px on lesson 1.2 and holds each pane's
  background and caption ink to the scheme under System on a dark OS, a
  click on Light, a reload with Light remembered (the head script's
  attribute before `<body>`, and RenderInitial's first tree), System again,
  and an OS change with no reload. Mutation-tested: removing boot()'s
  sendTheme and removing the head script's attribute each fail it.

- **N-055** · raised
  `2026-0919-1254-fold6-talkback-hid-harness-focus-after-navigation-inert-named-controls`
  · closed `2026-0919-2146-labelled-group-echo-on-compose` — the Stepper said its value
  twice on Compose ("2, Guests, 2"). (was #77)
- **N-056** · raised
  `2026-0919-1254-fold6-talkback-hid-harness-focus-after-navigation-inert-named-controls`
  · closed `2026-0919-2146-labelled-group-echo-on-compose` — a Drawer panel read
  "Notebook, Notebook". (was #78)

  Both were one bug: a labelled node merges its descendants, and Compose emits
  the label as a fake child beside the Texts rather than in place of them, so
  any child Text repeating the label or the stated value was spoken twice.
  `LocalGrMobGroupSaid` in `android/…/Renderer.kt` carries the nearest labelled
  ancestor's label and value text, and `GrMobText` clears the semantics of a
  Text that folds to either. Measured on the emulator's accessibility tree:
  the Stepper group's `'2'` TextView child and the panel's `'Notebook'`
  TextView child are both gone. Not the same fix as `LocalGrMobNamedControl`,
  which silences a *control's* whole content; a group's content stays readable
  apart from the echo.

- **N-026** · raised `2026-0917-0227-foldables-window-record-and-two-pane`
  · closed `2026-0919-2303-safe-insets-record-inert-codeeditor-sse-cleanup` — safe-area insets are a record. `core.SafeInsets`
  (`Top`/`Bottom`/`Left`/`Right`) is a field on `core.Window` rather than a
  record of its own: `Window` is `==`-comparable by design and dedupes on
  that, so insets ride the same `"window"` host event, the same dedupe and
  the existing `hooks.UseWindow` with no second pub/sub. Invalid insets are
  dropped on their own and the size kept, the same stance an unknown fold
  gets. Android reports `systemBars | displayCutout` — `safeDrawing` minus
  the IME, so the numbers match what its own `SafeArea` node applies — on an
  additive `OnGlobalLayoutListener` rather than
  `setOnApplyWindowInsetsListener`, which would displace the inset chain
  edge-to-edge and Compose depend on. iOS reads the root `GeometryReader`'s
  `safeAreaInsets`, which are the window's because the reader already
  ignores the safe area, and watches them as well as the size. The browser
  sends nothing (see N-061). Unrun on a device. (was #35)
- **N-059** · raised
  `2026-0919-1254-fold6-talkback-hid-harness-focus-after-navigation-inert-named-controls`
  · closed `2026-0919-2303-safe-insets-record-inert-codeeditor-sse-cleanup` — the premise was inverted, the leak was real. On this
  Compose version a nearer `focusProperties` does *not* beat an outer one:
  `fetchFocusProperties` walks up and the outermost wins, so RenderNode's
  head-of-chain placement was already right. What it cannot do is *reach*
  the CodeEditor's field — the walk takes `untilType = Nodes.FocusTarget`
  and stops at the first one it meets, and the field sits behind two scroll
  boxes that each delegate a focus target. An *editable* CodeEditor in a
  shut Drawer panel therefore stayed a hardware-keyboard Tab stop (a
  read-only one was saved by its own gate answering no). Fixed by reading
  `LocalGrMobInert.current` in the editor and conjoining it into the gate:
  `canFocus = !inert && (!readOnly || gate.open)`. Pinned by
  `TestComposeCodeEditorHonoursAnInertAncestor`. Still no bundled Inert
  subtree holds a CodeEditor, so it is unrun on a device. (was #81)
- **N-060** · raised `2026-0919-1421-rweb-serve-and-doctor-dev-server-check`
  · closed `2026-0919-2303-safe-insets-record-inert-codeeditor-sse-cleanup` — no upstream hook was needed after all. The constraint
  was self-imposed: `subscribe` queued the "hello" *into* the channel before
  registering it, which is what ruled out `SSEHub.Handler` and with it
  RWeb's on-close cleanup. Registering first and *broadcasting* the hello
  second keeps the same ordering guarantee — `mu` is held across both steps
  and across every broadcast, so no build can land in between — and lets the
  channel come from `Handler`, which unregisters it the instant the stream
  ends. The lingering channel and the "SSE Channel closed and drained"
  stdout line both go with it (the line came from the eviction closing a
  channel `sendSSE` was still reading; now the close happens after it has
  returned). The price is that hello reaches every open page, so
  `devclient.js` ignores a hello that tells it nothing.

Closures before the seed are written up in the session docs; the
most recent are in `2026-0919-1254-…` (#5, #62, #71, #75) and
`2026-0919-1118-…` (#7, #25, #70, #72, #73).
