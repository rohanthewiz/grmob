# Session: a menu nobody could read, and a node nobody styled

Session: https://claude.ai/code/session_01YVcrPiEYG2vnpUePWiK5T5
Date: 2026-09-06 (follows "a-blue-a-corner-and-a-heading")

## Ask

"Take the next 2 items in the Next list."

The two oldest, both age 5 and both rated **low** — which is why they had sat
for five sessions:

1. A picker's open menu is invisible to Go.
2. A styleless node skips the border reset in the WASM runtime.

They looked unrelated and turned out to be the same shape: **a rule stated four
times, where one of the four statements could not be checked.** The picker's
run-splitting was written once per renderer and the two native copies were held
by grepping a 900-line file. The runtime's style pass was written once per node
type in `createElement` and once, properly, in `styleFromGrMob` — and the
`createElement` set was missing a member nobody had noticed.

## Phase 1 — the menu, and the part of it that is not a view

### Why "invisible" was the whole entry

A SwiftUI `Menu`'s content is a closure of views; a Compose `DropdownMenu`'s is
a composable lambda. Neither can be read back. So three facts about an open
picker — the runs, the headings, the refused rows — rested on
`mobile/verify` finding substrings in `Renderer.swift` and `Renderer.kt`.

The move is the one `GrMobFlex.swift` already made for the flex solver, and its
doc says why: *a SwiftUI `Layout` can only be exercised by mounting it in a view
hierarchy, while everything that is easy to get wrong is a function from numbers
to numbers.* The same sentence is true of a menu with "list of dictionaries to
list of structs" substituted, and nobody had written it.

### The authority came first, because four copies is the actual defect

`core.SelectMenuSections` (`core/select_menu.go`) turns the flattened
`[]map[string]string` into `[]SelectMenuSection` of `SelectMenuItem`. `htmlout`
calls it directly; the two natives carry transliterations.

The reason it is worth a type rather than a helper is the edge every copy had
to remember on its own:

> a run is closed by the **next** option naming a different heading, so the
> last run of a list has nothing following it and needs an explicit flush.

That is the exact line `htmlout` shipped wrong for a release, and the previous
session's break-test found it only because a break-test was aimed at it. A
shared decomposition removes the hole rather than documenting it.

Two design points recorded in the file:

- **`Index` is carried on the item.** Two options may share a label —
  `core.Select` says the Value is the identity and the Label is written to be
  read — so a `ForEach` needs something else to key on. A `[String: Any]` is
  not `Hashable` in Swift and a `map` is not comparable in Go, so the position
  is the only thing left.
- **An ungrouped run is a real section**, not an absence of one. Every renderer
  then draws sections in a single loop with the wrapper as its only branch,
  instead of a loop with a special case inside it.

### The Swift half is now run, not read

`ios/GrMob/Runtime/GrMobSelectMenu.swift` imports Foundation and nothing else.
`ios/verify/run.sh` compiles it into the harness, `gen.go` emits nine option
lists **with the answer computed by `core.SelectMenuSections`**, and
`selectmenu.swift` compares. The harness now says:

    OK: 9 picker menus match Go's decomposition

The expectations are generated rather than written out because a fixture
written twice by hand proves only that one person made the same mistake twice —
which is the same argument the transcript replay next door already rests on.

One JSON detail cost a debugging round: a Go nil slice marshals to `null`, and
Swift's `Decodable` refuses a `null` where a non-optional array is declared. The
empty-options case brought the whole harness down on a *usage* message rather
than reporting a menu of no sections. Both slices are allocated empty now, with
the reason written beside them.

### The Android half is honest rather than finished

`GrMobSelectMenu.kt` imports nothing at all — no Compose, no Android — so the
decomposition is plain Kotlin a JVM harness could run. There is no such harness:
the Android build's only check is `compileDebugKotlin`, and adding a unit-test
source set would mean resolving JUnit offline against a cache that may not have
it. So the shape that makes one possible is in place, and the item stays open
in Next with that named as the remaining work.

`TestNativeMenuDecompositionIsUIFree` is what keeps the shape: it fails on any
`import` line in either file except Swift's `Foundation`. Stated as an absence
of imports rather than of a particular symbol, because a new UI framework would
be spelled a way the test could not guess.

### What is left for a text check, and it is much less

`mobile/verify/select_test.go` now covers the one step no harness reaches on
either platform: the line handing a row's value to `textChanged` and its
disabled flag to the construct that refuses the tap. Plus two shape checks —
that each renderer *asks* the decomposition rather than splitting the list
again, and that the two transliterations agree.

## Phase 2 — the node with no Style, and the chassis that had to move

### The entry, and the half of it the entry did not know

> `applyStyle` only runs when `node.Style` is non-nil. No node `core` builds is
> ever in that state, so this is a hand-assembled-tree gap.

True, and the interesting part is *why* the gap existed at all.
`createElement` made up for the conditional call with three branches of its
own — a `TextGrid` chassis, a stack default, an overlay default — which between
them covered every type-keyed answer `styleFromGrMob` gives **except** the
border. Three of four, and the fourth was the one nobody restated.

The asymmetry has a second half worth recording: the *patch* path has always
called `applyStyle` unconditionally, because reconcile emits the whole new Style
and there is no "no style" case to guard. So a styleless `<button>` kept the
browser's 2px outset rule until something gave it a Style — and that Style then
took the border away. One node drawn two ways depending on whether anything had
touched it since it was built.

The fix is `applyStyle(el, node.Style || {}, node.Type)` and the deletion of all
three branches.

### The Modal is where this got interesting

`Modal` was the one node type whose chassis lived in `createElement` as a raw
`Object.assign`. Making the style pass total clobbers it — and the first attempt
(assign the chassis *after* the pass, so it wins) passed every test and was
**wrong**, because `htmlout` writes `modalChassis` *ahead* of the author's
declarations so the cascade gives the author the last word. Chassis-wins on one
web target and author-wins on the other is a divergence, and it was one I had
just introduced.

What the two targets had actually been doing before:

    htmlout    chassis first, author's Style wins per-property
    runtime    chassis first, then a total pass wiped the whole chassis

So they had already disagreed — differently — and nothing said so.

The answer is the one the grid chassis had already found: **a set of node-type
defaults has to live in `styleFromGrMob`**, because every property there is
reassigned on every update-style patch and a chassis set only at creation is
wiped by the first one. Nine lines of `out.x = out.x || …`, which is how a total
function says what `htmlout` gets from the cascade.

### `display` is the exemption, and it is deliberate

A Modal's `display` **is** its open/closed state, written from the `visible`
prop — and `styleFromGrMob` never sees a prop. Assigning anything would close an
open dialog on the next restyle; assigning `""` would open a closed one.

    delete out.display;

`Object.assign` leaves an absent property alone, so deleting the key is how a
total pass abstains. That is the one hole in the totality rule in this file, and
it is now the only one.

The two targets get `display` by different routes for a reason that is symmetric
rather than arbitrary: `htmlout` writes its whole declaration list at once from
props it can see, so its chassis carries the display; the runtime's style pass
cannot, so its chassis does not.

### It is pinned, including the `||`

`htmlout.ModalChassis()` exports the nine declarations as a table (the string
literal became one), and `TestRuntimeModalChassisMatchesGo` parses the runtime's
block and compares. It checks the **pattern**, not just the values: a plain
`out.position = "fixed"` would pass a value comparison while making the runtime
the one target where a hand-built Modal's own Style loses.

The property names are captured twice and compared in Go rather than written as
a backreference — RE2 has none, and a pattern that quietly matched
`out.top = out.left || "0"` would be worse than one that needs a line of Go.

## The break-tests

**Thirty-three run, thirty-one caught first time. Two gaps, both closed.**

**Gap 1 — a flush pin that matched the wrong line.**
`TestNativeMenuDecompositionsAgree` checked for `if (building)` as the
end-of-loop flush. That same guard opens the loop's *own* append, so deleting
the flush outright left the substring standing and the test passed.

This is the same shape as the trailing-`<optgroup>` miss the previous session
recorded, and it is not a coincidence: **an end-of-loop flush is hard to check
precisely because it looks like the thing inside the loop.** The pin now carries
the `return` with it, so it can only match the statement after the loop.

**Gap 2 — a Modal with no `visible` prop at all.**
Every fixture carried one, so nothing covered the state `createElement` has to
plant for itself. With the style pass abstaining from `display`, a hand-built
Modal naming no `visible` would have had its dialog body laid out inline in the
middle of the page — the exact bug `htmlout`'s `modalChassis` was written for.
Caught by a break-test deleting `el.style.display = "none"`, which nothing then
noticed.

The full list, by phase:

**Runtime totality (14, one gap).** The style pass conditional again; the pass
ignoring the node's own Style; the Modal chassis before the pass; the chassis
deleted; `baseDisplay` no longer recorded; the border reset dropping `<select>`;
the chassis losing its author-wins fallback; drifting from htmlout's z-index;
losing a declaration; gaining one htmlout does not have; a fallback reading a
different property than it writes; the pass no longer abstaining from `display`;
htmlout's chassis dropping its centring; **a Modal no longer starting closed —
the gap**.

**The Go authority and htmlout (7, all caught).** The trailing run never
flushed; a gather instead of runs; `disabled` reading any non-empty string; the
label fallback removed; the index made section-relative; the `<optgroup>`
wrapper skipped; a disabled option written as choosable.

**Swift (9, all caught).** The trailing run never flushed; a returning heading
joining the earlier run; the row dispatching its label; the index restarting per
run; the disabled read inverted; the label fallback removed; the file reaching
for SwiftUI; `Renderer.swift` dropping `.disabled`; `Renderer.swift` dispatching
the label.

**Kotlin (5, one gap then caught).** The raw option list drawn instead of the
sections; a disabled option made choosable; **the trailing run never flushed —
the gap, then caught after the pin was tightened**; the run never closing on a
heading change; the index dropped; the file reaching for Compose.

### A process note, and this time the harness held

Two sessions running, the break-test harness itself was the thing that broke (an
unquoted zsh scalar that did not word-split; a `cd` that leaked into the next
batch). This session's driver is Python: it snapshots by content, applies one
exact-string mutation, runs, restores, and then **asserts the restore by
hashing**. An anchor that does not appear exactly once is a `SKIP`, not a silent
no-op. Nothing leaked.

## Two tripwires fired, and both said something true

`go test ./...` after the runtime edit:

    TestRuntimeGivesAStylelessModalItsSemantics   pinned the Modal branch's
                                                  own applyAccessibility call
    TestRuntimeAppliesTheStackDefault             pinned createElement reading
                                                  stackAxisFor

Both were written by earlier sessions to hold a line that had just stopped
existing, and in both cases the *fact* still needed guarding — so both were
repointed at the line that now makes it true rather than deleted. They pin the
same line for different consequences, and each says so.

## Verification

All six paths, green:

    gofmt / go build / go vet / go test ./... / go test -race ./...
    wasm/verify/run.sh          replay + 13 mjs suites
    ios/verify/run.sh           flex solver + 9 picker menus + replay + view + app
    android ./gradlew compileDebugKotlin --offline --rerun-tasks
    GOOS=js GOARCH=wasm go build ./...

New files:

    core/select_menu.go             the authority: sections, items, the split
    core/select_menu_test.go        seven properties, and the specification the
                                    three transliterations are held to
    ios/GrMob/Runtime/GrMobSelectMenu.swift   Foundation only, so it can be run
    ios/verify/selectmenu.swift     the comparison, reported per difference
    android/.../GrMobSelectMenu.kt  no imports at all
    wasm/verify/styleless_test.mjs  the four type-keyed answers a styleless node
                                    gets, and the Modal's three
    wasm/verify/modal_test.go       the chassis compared to htmlout's, pattern
                                    and all

Changed:

    htmlout/export.go               renderSelect calls the authority;
                                    renderSelectOptions split out; modalChassis
                                    became a table with ModalChassis() beside it
    wasm/grmob-runtime.js           the total style pass; the Modal chassis in
                                    styleFromGrMob; the display exemption
    ios/GrMob/Runtime/Renderer.swift  Menu is a loop over sections; the rows
                                    moved into grMobMenuItems taking items
    android/.../Renderer.kt         the same, over the same shape
    ios/verify/{gen.go,main.swift,run.sh}  the menu cases and the new pass
    mobile/verify/select_test.go    repointed at what a text check is still for
    wasm/verify/{a11y,stack}_test.go  the two tripwires, repointed

## Docs

`core/select_menu.go` carries the argument for the type existing at all —
four copies, and the flush each had to remember.
`core/input.go`: `SelectOption.Group` and `Select` both point at the authority.
`htmlout/export.go`: `renderSelect`'s "Groups are runs" section now says the
decomposition is core's and what is left here is the markup; `ModalChassis`
explains why `display` and `background` are not in it.
`wasm/grmob-runtime.js`: `createElement`'s style pass, the Modal chassis, the
`delete out.display` exemption, `stackAxisFor`'s "one place now", and
`applySelectOptions`' note that a DOM append has no flush to forget.

`docs/platforms/wasm.md`: a new "Every node is styled" section; the stack-table
paragraph rewritten around one read; the Modal chassis and the display
exemption.
`docs/platforms/native.md`: "The menu is invisible, so the decision was moved
out of it", including what Android still lacks.
`docs/platforms/exporters.md`, `docs/concepts/forms.md`: the shared
decomposition.
`docs/concepts/styling-and-theming.md`: the Modal's visual chassis stated as
author-wins on both web targets.
`ROADMAP.md`: `core.SelectMenuSections` as its own entry; the border-reset entry
grew the runtime's totality.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here. Value is the payoff of doing it, not the effort: `high` means
something is being worked around today or a second consumer has arrived,
`medium` means it blocks one named thing, `low` means it is a gap nobody has
bumped into yet.

1. **(age 5 · value high) Neither a `listbox` nor a `tablist` has keyboard
   navigation on the web, and there are three consumers now.** The roles state
   semantics ARIA's patterns pair with behaviour: the container takes focus,
   the arrow keys move the active item, and a roving tabindex or
   `aria-activedescendant` says which. `examples/mobileapp`'s article list is
   the listbox; `examples/social`'s bottom bar and tutorial 4.5 are the
   tablists, and all of them hand a keyboard user a role that claims more than
   the widget does. Needs a focus concept `core` does not have —
   `core/focus.go` is about putting the cursor in a named field. Both phones
   navigate by swipe and lose nothing. An accordion header is a third consumer
   in a different form: a real `button` on the web and therefore already
   keyboard-operable, which *narrows* the item rather than widening it.
2. **(age 5 · value medium) Nothing re-checks a permission on foreground.**
   A user can grant one in Settings and come back, and no platform says so.
   `hooks.UsePermission` deliberately does not (it cannot see whether its
   screen is still on top, and five screens would each fire a check per
   resume), so every consumer writes the same `UseLifecycle` pairing by hand.
   A `hooks.UsePermissionLive` — or a `RecheckOnForeground` option — is the
   obvious shape; what is missing is a rule for which screen owns it.
3. **(age 5 · value low) The Android "asked before" flag does not survive a
   process restart.** It is what separates a permanent refusal from "never
   asked", so after a restart a permanently-denied permission reads as
   `Prompt` until the next request proves otherwise — one dead button press.
4. **(age 5 · value low) A browser `Request` opens the device to answer.**
   There is no request API, so `getUserMedia` is the only thing that prompts,
   and a granted camera check has genuinely opened the camera for a moment. A
   refused one also reads as `denied` whether the user pressed Block or
   dismissed the prompt, because `NotAllowedError` does not say which.
5. **(age 5 · value low) The gomobile stub's types are unchecked.** The pin
   holds the *names* — every bindable symbol declared, nothing extra —
   because copying gobind's type mapping into a test would be reimplementing
   gobind. A wrong signature fails the Swift typecheck the moment the shell
   calls it, so what is genuinely unguarded is a stub declaration the shell
   never touches.
6. **(age 4 · value medium) A hand-built tab strip's panel still cannot say it
   is one.** `AccessibilityControls` closes the pointing and `RoleTabPanel` is
   now defensible. What stands in the way is mechanical: the WASM runtime
   tells its own wiring apart from an author's role by `"tabpanel"` not being
   a `core.Role`, held there by `TestNoRoleCollidesWithTheTabPanelWiring`.
   Replacing that discriminator with a `data-grmob-panel` marker (which both
   web targets would write, as they already do `data-grmob-chrome`) is the
   right fix and unblocks the constant. Two native arms and a large doc block
   go with it.
7. **(age 4 · value medium) Three widgets take `group` where ARIA has a better
   role.** The supplied fallback made their names audible and stopped there.
   `ProgressBar` wants `progressbar` with `aria-valuenow`/`valuemin`/`valuemax`
   — it is already computing the percentage into its name, which is the
   workaround. `Skeleton` wants `status`, since "Loading" is an advisory that
   is replaced. `FormField`'s required marker is a glyph standing in for a
   word, which is `img`'s shape. Each is a widget change plus, for
   `ProgressBar`, a value vocabulary `core` does not have.
8. **(age 4 · value medium) Nothing catches a dangling `aria-controls` or a
   duplicate `AccessibilityID`.** Both are written verbatim and neither
   exporter can see the whole document at the moment it writes one — but a
   finished tree *can* be walked, which is exactly what `core.SetDebugMode`
   already does for cursor drift and duplicate keys. A debug-mode pass
   reporting an id claimed twice and a reference resolving to nothing would
   catch the one failure mode the pair has, and it is a failure that is
   invisible on every target rather than merely quiet on two.
9. **(age 4 · value low) An `AccessibilityID` is not validated.** An id
   containing a space is invalid HTML; an empty one is written as `id=""`; one
   starting with `grmob-` collides with `core.TabView`'s minted ids and breaks
   a wiring the author never wrote. The prefix is documented as reserved and
   nothing enforces it. A drop would be silent too, so the honest fix is
   probably the debug-mode pass in item 8 rather than a guard in the
   exporters.
10. **(age 3 · value medium) Every ARIA claim in the docs is hand-checked
    prose.** A Next-list item once asserted "`group` supports `aria-expanded`",
    which survived three re-sorts and was false. There are dozens of such
    claims now — three role lists, two name prohibitions, four `aria-level`
    scopes, six `aria-selected` roles. A generated table (from the ARIA spec's
    own machine-readable role definitions, checked in as a fixture) would make
    them testable instead of reviewable. Build-time rather than runtime, which
    is why it is medium.
11. **(age 3 · value medium) `Accordion` is the only disclosure, and
    `AccessibilityExpanded` has no second consumer.** One consumer is not a
    vocabulary, and the field's role list carries five arms nothing reaches —
    `link`, `listbox`, `row`, `columnheader` and `tab`. The shapes that would
    reach them are real and absent: a `GroupedList` band that collapses (row),
    a combobox built out of `SearchField` plus a listbox (link/listbox), a
    `DataTable` with collapsible column groups (columnheader). Two sessions
    running have now unblocked an item stuck on exactly this argument —
    `StackAlign` by a fact about the three platforms, the picker menu by a
    fact about what a view closure can be asked. Worth asking the same
    question here before waiting for a widget.
12. **(age 3 · value low) `Colors.ControlBorder` is 2.92:1 against
    `Colors.Surface` under `DefaultTheme`.** The outer edge clears 3:1 and is
    what identifies a quiet chip, which is argued in two places and asserted in
    one — but any future widget that draws a boundary *on* a Surface fill
    inherits the shortfall without inheriting the argument. The honest fixes
    are both theme decisions: darken past Apple's systemGray, or give the quiet
    chip a fill that is not Surface. Cheaper than it was: a role's spelling is
    movable when the palette is the authority, and one has been moved.
13. **(age 3 · value low) An `ExpandedState` on a node with no handler is
    silently inert on Android.** `grMobDisclosure` is in `gestureModifier`,
    which returns early when a node carries neither `onClick` nor `onLongPress`,
    so the state reaches the renderer and buys nothing. That is deliberate — an
    action nothing can perform is worse than none — but it is a rule stated
    only in a comment, where the equivalent web rule (a state on an unroled
    node is dropped) has a test on both targets. `core.SetDebugMode` reporting
    a disclosure with no way to open it would put it where the other two are.
14. **(age 2 · value medium) `Margin` has no per-side props at all.** Padding
    got its four; margin still has only `Margin(all)`, so every single-side
    margin goes through a whole `EdgeInsets` in a `UseStyle` — and that is
    worse for margin than it was for padding, since there is no
    `MarginHorizontal` or `MarginTop` either. Two live workarounds today:
    `components/separator.go` writes `Margin: EdgeInsets{Horizontal: s.Inset}`
    for its inset rule, and `examples/chat`'s bubble writes
    `Margin: EdgeInsets{Bottom: 8}` for the gap between messages. The work is
    mechanical — the same `settleHorizontal`/`settleVertical` helpers apply
    unchanged, and no renderer would move — which is why this is medium and not
    high: it is scope, not difficulty.
15. **(age 2 · value low) A side prop cannot express "clear this side" on a
    node whose axis a later prop will set.** `PaddingLeft(0)` then
    `PaddingHorizontal(16)` gives 16 on both sides, which is correct
    last-one-wins and is also the only way to write the pair. Recorded because
    the settle makes every *other* zero work, and the one remaining hole should
    be written down rather than rediscovered.
16. **(age 2 · value low) `rowsSpec` is shared by exactly two widgets and one
    of its ten fields by exactly one.** `Wrap` exists so DataTable can add a
    tap target and a selection tint that GroupedList has no use for, which
    makes it a widget-specific knob living in a shared type. That is the right
    trade at two callers; at three it would be worth asking whether the
    decoration belongs in the spec or in a wrapper around `appendRows`' output.
17. **(age 1 · value medium) `declaredInk` is now unobservable under both
    bundled themes.** Each pairs white with a fill dark enough that measurement
    picks white too, so an implementation that deleted the first step and only
    measured would paint identical pixels under DefaultTheme *and*
    MaterialTheme. The rule is right and still applies to any theme whose house
    button is a mid-tone — but its whole evidence is now one test fixture
    (`midTonePrimaryTheme`), which is a thinner thread than a bundled theme.
    The honest options are to leave it and accept the fixture, or to give one
    bundled theme a mid-tone button base on purpose, which is a look decision.
    Same shape one property over: `PrimaryOnLight` is an identity on both
    bundled themes now, so the *reverse lookup* has no live consumer either.
18. **(age 1 · value medium) An unsized `ZStack` with a placed layer diverges
    on iOS.** SwiftUI has no per-child stack alignment, so a placed layer is
    wrapped in a filling frame, and a filling frame grows an otherwise unsized
    stack to its parent's proposal — where a Compose `Box` and a CSS grid track
    stay the size of their largest child. Documented in four places and
    avoidable by pinning the stack's box, which `ZStack` already asks for. What
    would actually close it is a SwiftUI layout that places without filling
    (a custom `Layout`, or an `alignmentGuide` scheme that can see the
    container's size), and neither is a small piece of work. Nothing pins the
    divergence today either — `ios/verify` type-checks and replays, it does not
    measure.
19. **(age 1 · value low) `StackAlign` is inert outside a `ZStack` and nothing
    says so at the call site.** It is inert deliberately and by construction on
    the web (the stack imposes it, so it never reaches a flex child), and by
    omission on the natives (only the stack renderers read the field). But a
    `core.StackAlign` on a `Column`'s child compiles, merges, crosses the wire
    and does nothing on all four targets with no diagnostic. `core.SetDebugMode`
    already walks a finished tree for cursor drift and duplicate keys; a
    placement on a node whose parent is not an overlay is the same kind of
    finding, and it would join items 8 and 13 in the same pass.
20. **(age 1 · value low) A `Select`'s group headings cannot be styled or
    ordered independently.** A heading is a string on an option, so there is no
    way to give one an icon, mark a whole run disabled, or state a heading that
    has no options under it. `<optgroup disabled>` exists in HTML and both
    natives could express a disabled section; nothing in `SelectOption` can ask
    for it. **Cheaper than it was:** `core.SelectMenuSections` is now the one
    place a section is described, so a heading with fields of its own would be
    a change to `SelectMenuSection` and its three transliterations rather than
    to four independent renderers. The flat string map on the wire is still
    what keeps them reading one shape.
21. **(age 0 · value medium) The Kotlin decomposition has no runner.**
    `GrMobSelectMenu.kt` imports nothing precisely so a JVM harness could
    execute it, and `ios/verify` proves what such a harness buys — nine cases
    generated from `core.SelectMenuSections`, compared against the real Swift
    function. Android has no equivalent: `compileDebugKotlin` is the only check
    the build runs, and adding a `src/test` source set means resolving JUnit
    against a `--offline` gradle cache that may not carry it. What is unguarded
    today is a Kotlin transliteration that drifts from the Go authority in a
    way the source-text pins do not name — and those pins are the weaker kind,
    as this session's flush gap showed. A harness here would also be the place
    to run any *future* UI-free Kotlin, which is currently a category of one.
22. **(age 0 · value low) The WASM runtime is the fourth copy of the
    decomposition and does not call the authority.** `applySelectOptions` walks
    the flat list itself, appending into a live `<optgroup>`. That is defensible
    — a DOM append has no closing step, so the flush that the other three have
    to remember does not exist here, which is why this target never had the bug
    — but it means a change to `SelectMenuSection`'s shape has three consumers
    to update and one to remember. The mjs suite covers the behaviour live; what
    is missing is the *shared fixture*, i.e. `wasm/verify/gen.go` emitting the
    same `menuCases` table so the runtime is compared against Go's answer
    rather than against a hand-written expectation.
23. **(age 0 · value low) `styleFromGrMob` has exactly one exemption from its
    own totality rule, and nothing states the rule for adding a second.**
    `delete out.display` for a `Modal` is right: the display is the open/closed
    state, it arrives through the prop channel, and a total pass cannot see a
    prop. But "abstain by deleting the key" is now a technique available to any
    property, and the next one to reach for it will not have this argument
    attached. The test pins the line; what is unwritten is the *test a candidate
    has to pass* — that the property is owned by a channel this function cannot
    read, and that some other path is total for it instead.
