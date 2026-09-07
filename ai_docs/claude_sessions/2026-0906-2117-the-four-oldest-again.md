# Session: the four oldest, and an argument that became a fixture

Session: https://claude.ai/code/session_01YVcrPiEYG2vnpUePWiK5T5
Date: 2026-09-06 (follows "the-aria-cluster")

## Ask

"Work on the four oldest items in the Next list."

| # | age·value | item |
|---|---|---|
| 1 | 7·low | the gomobile stub's types are unchecked |
| 2 | 5·medium | `Accordion` is the only disclosure, and `AccessibilityExpanded` has no second consumer |
| 3 | 4·low | `Colors.ControlBorder` is 2.92:1 against `Colors.Surface` under `DefaultTheme` |
| 4 | 3·medium | `Margin` has no per-side props at all |

All four closed. What three of them share is a shape the last two sessions
did not have: **the item's own stated reason for not doing it was an argument
that had stopped being true**, and in two cases the argument had been right
when it was written.

## Phase 1 — three rows are not a reimplementation

Item 1. `mobile/verify/gomobilestub_test.go` held the stub's *names* — every
bindable symbol declared, nothing extra — and said in its own header why it
stopped there: copying gobind's type mapping into a test would be
reimplementing gobind to check a file that exists to avoid running gobind, and
a wrong signature fails the Swift type-check in `ios/verify` anyway.

Both halves needed re-examining and both gave way.

The first was right about gobind's *general* mapping and wrong about this
one. Package `mobile` is written to a narrow surface on purpose, and
`bindableSignature` already refuses anything outside `string`, `bool`, `int`
and a local interface — so the whole mapping the stub can possibly need is
three rows and a protocol rule. Stating three rows is not reimplementing
anything; it is what `aria/verify/testdata/aria.json` did for role semantics
last session.

The second quietly assumed every declaration has a call site. Three do not —
`MobileDataDir`, `MobileRenderAgain` and `MobileReportHostEvent` are all
reachable from a shell that never touches them — and for those the type-check
proved nothing at all. That is the gap, and it is exactly the one the Next
entry named: *what is genuinely unguarded is a stub declaration the shell
never touches.*

The check builds the expected Swift declaration from the Go AST and compares
it **whole**, character for character, rather than type by type. One string
comparison catches a reordered parameter pair, a dropped argument label and a
missing `?` with the same assertion.

Two details are transcriptions rather than derivations, and both are facts
about the generated header:

    string -> String? / String     gobind annotates every NSString* parameter
                                   _Nullable and every return _Nonnull, so the
                                   mapping is split by position. A stub taking
                                   String everywhere would accept shell code
                                   the real framework rejects, which is the
                                   drift the file exists to catch.

    labels                         A bound package function becomes a C
                                   function, which Swift imports with no
                                   argument labels; a bound interface method
                                   becomes an ObjC selector, whose pieces Swift
                                   turns into labels — first folded into the
                                   name, the rest the Go parameter's own name.
                                   Both shapes are in the stub and they differ.

Two things are **refused** rather than mapped: a multi-result signature (gobind
makes it throwing, and none has ever crossed) and a returned interface (the
nullability of a returned protocol is a header fact nobody has read). The
error message in each case says to go and read the header, which is the same
stance `bindableGoTypes` already took on parameter types.

## Phase 2 — a second disclosure, and what it was for

Item 2. The Next entry was explicit that this is the one ARIA item whose
answer is a *widget*, and that building a combobox is a feature rather than
plumbing. The cheaper shape it named — a `GroupedList` band that collapses —
is what landed.

**The state is the caller's.** `GroupedList` calls no hook, which is what lets
it be rendered conditionally without disturbing the caller's hook cursor, and
owning collapse state would end that. It is also the wrong owner on the
merits: which months are shut is screen state, it usually wants to survive a
pager reload, and a screen that wants "collapse all" cannot reach inside a
widget's `NewState`. `Accordion` stays the right answer for a single section,
where owning the state is a convenience rather than a cage.

**The two functions are one type.** `Collapse{IsCollapsed, OnToggle}` rather
than two fields, for the reason `core.ValueRange` gives for its three numbers:
they are halves of one fact and are useless apart. A predicate with no handler
would hide rows behind a band nobody can open — a feed that silently loses
items — so `Collapse.hides` requires both, in one place, and the band's own
state is derived from the same pair. A run that is hidden always has a control
announcing it as collapsed.

**A shut run emits no rows at all**, not hidden ones: nothing for the
reconciler to walk, nothing in the document. The band keeps its key across the
toggle, so re-opening patches the rows back rather than remounting the band
and taking its sticky pin and any focus inside it along.

### What the second consumer was actually for

The payoff is not the feature. `Accordion`'s ARIA arrangement took three
attempts, and its own doc records that **both rejected versions looked
correct** — a named `generic` (dropped by every browser) and a `group` with
the heading on the words (which cannot carry `aria-expanded` at all). A second
widget copying that by hand would have been checked only against its own
expectations.

So the shape is one value now — `components.disclosure` — and
`TestBothDisclosuresBuildTheSameShape` renders the two side by side and
compares the tier, both names, the state and the chevron, in both directions.
`Accordion`'s four-paragraph argument moved onto the type and the widget kept
its own two decisions.

One place the two genuinely differ, and it is a decision rather than an
accident: **the band's count badge stays outside the button.** A button's
children are presentational, so a count inside the control stops being
announced — and a count is content, not chrome. Folding it into the button's
accessible name was the alternative, and that is assembling an English phrase
in a renderer, which is the move `Chip`'s `", selected"` suffix was deleted
for. The cost is that the badge and the band's insets are not tappable.

`DataTable` does not get this. Its own comment already says why a grouped
table's bands are the wrong shape — they sit inside the body's rowgroup, where
ARIA has no reading for them even as plain headings — and making one a button
would be a second claim on a structure that is already wrong. The fix is
per-band rowgroups, which is a change to the shared row emission.

That leaves `rowsSpec` with **two** single-widget knobs, `Wrap` (DataTable's)
and `Collapse` (GroupedList's), which is the count Next item 6 was watching.
The census now records an owner per field and fails at three, so the question
that item asks gets asked by a test rather than by somebody noticing.

## Phase 3 — the argument that became a fixture

Item 3. `Colors.ControlBorder` clears WCAG 1.4.11's 3:1 floor against a page
and falls 0.08 short against `DefaultTheme`'s `Surface`, which is the quiet
chip's own fill. The entry said the honest fixes are both theme decisions —
darken past Apple's systemGray, or give the quiet chip a fill that is not
Surface — and neither is mine to make.

Neither is what the item is really about. Its second sentence is: *any future
widget that draws a boundary on a Surface fill inherits the shortfall without
inheriting the argument.* The shortfall is defensible; being invisible is not.

So the fix is a census. Every bundled boundary tone crossed with every fill a
control can sit on — the page, the `Surface` panel, a `Card`, a field's own
fill — measured, with each pair required to clear or to be recorded:

    DefaultTheme    page 3.26   Surface 2.92   Card 3.26   field 3.26
    MaterialTheme   page 4.61   Surface 4.23   Card 4.61   field 4.41

Exactly one entry in `knownBoundaryShortfalls`, carrying the pair, the number
and a précis of the argument. Three properties are worth stating:

- **The recorded ratio is compared, not just the key.** A retint that left the
  pair failing but changed how badly would slip past a bare exemption, and the
  argument is about a specific distance — "the last 0.08" stops being true at
  2.4:1.
- **A sibling test deletes the exemption** if a retint ever closes the gap.
  Same shape as `TestTheDividerRoleIsWhyTheFieldFrameIsSeparate` one section
  up, and for the same reason: an exemption nobody revisits is how the next
  real shortfall gets waved through by something that looks like precedent.
- **`Components.Camera` is excluded by name**, not by a lightness test. A rule
  that skipped dark fills would also skip a dark theme's page.

The two prose sites — `ColorPalette.ControlBorder` and `chipRing` — now point
at the table instead of restating it.

## Phase 4 — one identifier changed, six times

Item 4, and the mechanical one. `MarginTop`/`Bottom`/`Left`/`Right`/
`Horizontal`/`Vertical`, using `settleHorizontal` and `settleVertical`
unchanged: `EdgeInsets` is one type, and every renderer resolves a margin side
exactly as it resolves a padding side, through one function per target called
for both. No renderer moved.

What is worth recording is that this field is a **worse** trap than padding
was, for a reason specific to it. `UseStyle` replaces `Margin` outright, and a
margin's other three sides are usually zero — so the struct spelling looks
like it set one gap while silently clearing the rest. Padding's version of
that mistake shows up as a squashed row; margin's shows up as two elements
touching, three screens away.

Both live workarounds are one prop each now, and the separator's case turned
out to be a live instance of exactly that: `Margin: EdgeInsets{Horizontal:
s.Inset}` cleared any vertical margin the caller's own `Style` asked for, and
there is a test for that now.

The test nobody would think to write is
`TestTheMarginAndPaddingFamiliesDoNotReachIntoEachOther`. Every prop in the
new file is a copy of its padding twin with one identifier changed, and the
identifier is the argument to the settle helper — `settleHorizontal(&s.Padding)`
inside `MarginLeft` compiles, type-checks, and produces a prop that clears a
container's left padding while setting its left margin. Both directions, since
the same slip is available in either file.

## The break-tests

**Fifty-six run, fifty-three caught first time.** Three real gaps, all closed
and re-run.

**Gap 1 — an example's spacing was unpinned.** `examples/chat` has a test file
and nothing in it measured the bubble's margin. That was survivable while the
margin was a whole `EdgeInsets`, which is hard to get *partly* wrong; it is
one prop now, and `MarginTop` is a plausible typo that moves every gap to the
wrong side of every bubble and changes nothing else. Pinned, all four sides.

**Gap 2 — the band's announcement was not wired to anything.** Inverting
`gh.Expanded = !collapsed(group)` passed the whole suite: every test in the
file asked whether the *rows* were emitted, and the announcement is derived
from the same predicate on a separate line. That is the worst of the three —
correct rows and every reader told the opposite of what the screen shows.
`TestTheBandAnnouncesWhetherItsOwnRunIsShowing` checks both runs in one pass,
so a band hard-coding either state fails.

**Gap 3 — a heading inside the button went unseen.** `readDisclosure` found
the first heading with `findFirst`, which is the wrapper, so adding heading
props back onto the label *inside* the control changed nothing. That is the
precise failure the code comment warns about: written into the document,
pruned out of the accessibility tree by every browser, and still correct in an
export. The reader is now an assertion in the shared helper, so both widgets
carry it.

**A property worth noting: the census tests hold each other.** Mutating the
census's own fixture — dropping the `Surface` backdrop, moving the exemption
to the wrong theme, drifting the recorded ratio — was caught every time, which
is not the usual "tests are not covered by tests" result. Dropping the
backdrop makes the *sibling* test fail, because the exemption then names
something the census does not measure.

Same Python harness as the last three sessions: snapshot by content, one
exact-string mutation, run, restore, assert the restore by hashing, and an
anchor that does not appear exactly once is a `SKIP`. Zero skips and zero
restore failures. One mutation of mine was bad rather than missed (deleting
the chevron append leaves an unused variable, so it was caught by the
compiler); it was rewritten as a real escape — the chevron emitted outside the
button — and caught properly.

## Verification

All six paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    wasm/verify/run.sh          replay + 15 mjs suites
    ios/verify/run.sh           flex solver + 9 picker menus + replay + view + app
    android ./gradlew compileDebugKotlin --offline --rerun-tasks
    GOOS=js GOARCH=wasm go build ./...

New files:

    core/margin_sides.go            the six props, and why this field is the
                                    worse trap
    core/margin_sides_test.go       the settle on a second field, and the
                                    cross-family check
    components/disclosure.go        ARIA's disclosure shape, stated once
    components/disclosure_test.go   the two widgets compared against each other
    components/collapse_test.go     the band, the runs, and the export

Changed:

    mobile/verify/gomobilestub_test.go  the signature check and its three-row
                                        type table; the header comment that
                                        used to argue against it
    ios/verify/gomobile_stub.swift      the nullability note is load-bearing now
    components/variant_test.go          the boundary census and its exemption
    core/theme.go                       ControlBorder points at the census
    components/chip.go                  chipRing points at the census
    components/accordion.go             onto the shared shape; the doc condensed
    components/grouping.go              Collapse, and the band as a disclosure
    components/grouped_list.go          the field, the wiring, the row skip
    components/rows_spec_test.go        an owner per field, and the count of
                                        single-widget knobs
    components/separator.go             MarginHorizontal, and the third spelling
                                        of Inset
    components/separator_test.go        the vertical margin the old shape ate
    examples/chat/main.go               MarginBottom
    examples/chat/main_test.go          the gap, pinned
    htmlout/edges_settle_test.go        the settle and the resolution, on margin

## Docs

`docs/concepts/styling-and-theming.md`: the six margin props with the
`UseStyle` trap spelled out; the boundary census as a 2x4 table with the one
shortfall and where its argument now lives.
`docs/components.md`: the disclosure as a shared shape and why; a
"Collapsible bands" section — caller-owned state, one type, no rows at all,
the badge outside the control, and why `DataTable` is out.
`docs/platforms/native.md`: a "The bridge stand-in" section — the two
directions, the two levels, the three-row table, and the assumption the old
argument made.
`ROADMAP.md`: four new entries.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here. Value is the payoff of doing it, not the effort: `high` means
something is being worked around today or a second consumer has arrived,
`medium` means it blocks one named thing, `low` means it is a gap nobody has
bumped into yet.

1. **(age 5 · value low) `Colors.ControlBorder` is still 2.92:1 against
   `Colors.Surface` under `DefaultTheme`.** The *invisibility* is closed — the
   census measures every pair and the exemption carries the argument — and the
   number is unchanged, because both honest fixes are look decisions nobody has
   made: darken past Apple's systemGray, or give the quiet chip a fill that is
   not Surface. Cheaper to decide now than it was: the consequences of either
   are one test run rather than an audit, and the census says immediately
   whether a candidate hex clears every pair or only some.
2. **(age 4 · value medium) `Margin` and `Padding` have no way to say "clear
   this side" ahead of an axis prop.** `PaddingLeft(0)` then
   `PaddingHorizontal(16)` gives 16 on both sides, which is correct
   last-one-wins and is also the only way to write the pair. Now on both
   families rather than one, which does not change the shape of the hole but
   doubles the surface it is on.
3. **(age 4 · value low) `rowsSpec` carries two single-widget knobs and a test
   that fails at three.** `Wrap` is DataTable's and `Collapse` is
   GroupedList's, and the census records an owner per field so the question
   gets asked rather than noticed. What the test cannot decide is the answer —
   whether the third one means moving the decoration into a wrapper around
   `appendRows`' output, or splitting the spec, or accepting that a shared
   parameter list for two similar widgets legitimately has per-widget halves.
4. **(age 3 · value medium) `declaredInk` is unobservable under both bundled
   themes.** Each pairs white with a fill dark enough that measurement picks
   white too, so an implementation that deleted the first step and only
   measured would paint identical pixels under both. The rule is right and
   still applies to any theme whose house button is a mid-tone — but its whole
   evidence is one test fixture (`midTonePrimaryTheme`), which is a thinner
   thread than a bundled theme. Same shape one property over: `PrimaryOnLight`
   is an identity on both bundled themes, so the reverse lookup has no live
   consumer either.
5. **(age 3 · value medium) An unsized `ZStack` with a placed layer diverges on
   iOS.** SwiftUI has no per-child stack alignment, so a placed layer is
   wrapped in a filling frame, and a filling frame grows an otherwise unsized
   stack to its parent's proposal — where a Compose `Box` and a CSS grid track
   stay the size of their largest child. Documented in four places and
   avoidable by pinning the stack's box, which `ZStack` already asks for. What
   would close it is a SwiftUI layout that places without filling (a custom
   `Layout`, or an `alignmentGuide` scheme that can see the container's size),
   and neither is small. Nothing pins the divergence either — `ios/verify`
   type-checks and replays, it does not measure.
6. **(age 3 · value low) `StackAlign` is inert outside a `ZStack` and nothing
   says so at the call site.** Inert deliberately and by construction on the
   web, and by omission on the natives. A `core.StackAlign` on a `Column`'s
   child compiles, merges, crosses the wire and does nothing on all four
   targets with no diagnostic. **Still the cheapest unclaimed item in the
   list:** `core.AuditTree` is a live tree walk with four findings in it, and
   "a placement on a node whose parent is not an overlay" is a fifth arm rather
   than a new mechanism.
7. **(age 3 · value low) A `Select`'s group headings cannot be styled or
   ordered independently.** A heading is a string on an option, so there is no
   way to give one an icon, mark a whole run disabled, or state a heading with
   no options under it. `<optgroup disabled>` exists in HTML and both natives
   could express a disabled section. `core.SelectMenuSections` is the one place
   a section is described, so this would be a change to `SelectMenuSection` and
   its three transliterations rather than to four renderers.
8. **(age 2 · value medium) The Kotlin decomposition has no runner.**
   `GrMobSelectMenu.kt` imports nothing precisely so a JVM harness could
   execute it, and `ios/verify` proves what such a harness buys. Android has
   only `compileDebugKotlin`, and adding a `src/test` source set means
   resolving JUnit against a `--offline` gradle cache that may not carry it.
   **The pile resting on source-text pins alone did not grow this session** —
   nothing new was added to `GrMobStyle.kt` — but `grMobValue`'s three-way
   branch from last session is still checked by `strings.Contains`, and the
   prefix-matching failure that shape produces has now appeared in three
   consecutive sessions' break-tests.
9. **(age 2 · value low) The WASM runtime is the fourth copy of the
   decomposition and does not call the authority.** `applySelectOptions` walks
   the flat list itself. Defensible — a DOM append has no closing step — but a
   change to `SelectMenuSection`'s shape has three consumers to update and one
   to remember. What is missing is the shared fixture: `wasm/verify/gen.go`
   emitting the same `menuCases` table.
10. **(age 2 · value low) `styleFromGrMob` has one exemption from its own
    totality rule and nothing states the rule for adding a second.**
    `delete out.display` for a `Modal` is right, and "abstain by deleting the
    key" is now a technique available to any property. The test pins the line;
    what is unwritten is the *test a candidate has to pass*.
11. **(age 2 · value low) The keyboard pattern is verified against a DOM that
    is not a browser.** `wasm/verify/dom.mjs` has no bubbling, no layout, and
    its `focus()` is an assignment. Still unverified: that `tabindex="-1"`
    really removes a `<button>` from the tab order, that a disabled control
    refuses focus, that `preventDefault` on `ArrowDown` stops the scroll. The
    honest answer is a browser-driven pass rather than a wider shim.
12. **(age 2 · value low) `menu`, `tree` and `grid` are patterns `core.Role`
    has no vocabulary for.** Each needs more than arrows: a menu has submenus
    and Escape, a tree has expansion state per node, a grid is two-dimensional.
    **A tree got closer this session:** `components.disclosure` is the
    heading-around-button shape a tree's twisty needs, and `Collapse` is the
    caller-owned expansion state a tree node would want — what is still absent
    is the recursion and the `treeitem` role, and the reason remains that no
    widget in this repository is one.
13. **(age 1 · value medium) The ARIA fixture is transcribed, not generated.**
    `aria/verify/testdata/aria.json` makes every guard checkable against one
    statement, but the statement is hand-written from the spec and can be wrong
    the way the prose it replaced could be wrong. The W3C publishes the role
    definitions machine-readably. **The same shape now exists twice:**
    `knownBoundaryShortfalls` and `gobindSwiftTypes` are both hand-transcribed
    tables that other tests are held to — the difference being that those two
    are three rows and one row, where the ARIA fixture is a spec.
14. **(age 1 · value low) The audit's debug-mode guard has no test that can
    fail.** `AuditTree` returns early when debug mode is off, and deleting that
    line changes nothing observable, because `upsertConcern` carries its own
    guard. What the guard buys is cost, and nothing measures it. A benchmark
    asserting "off is free" would cover all three debug checks.
15. **(age 1 · value low) `aria-orientation` is announced for `toolbar` and no
    target gives one a keyboard.** ARIA's toolbar pattern has arrow-key
    navigation, and `components.ChipStrip` is a toolbar of real controls a
    keyboard crosses one tab stop at a time. What stops a third row in
    `COMPOSITE_MEMBERS` is that ARIA does not name a toolbar's members, so the
    member walk needs a second rule — "every focusable descendant not inside a
    nested composite" is probably it.
16. **(age 1 · value low) A typeahead match does not select, only focuses.**
    ARIA's listbox pattern lets a single-select listbox move the selection with
    the focus. This moves focus and leaves selection to the author's `onClick`,
    which is the safe default and also not a choice the framework can make,
    since `aria-selected` is written from Go state a keystroke cannot reach
    without a render pass.
17. **(age 0 · value medium) The gobind type table cannot describe two of the
    shapes it refuses.** `swiftType` rejects a returned bound interface and
    `swiftResult` rejects a multi-result signature, in both cases with a
    message saying to read `Headers/Mobile.objc.h` and add the row. That is the
    right stance for a table that may only hold facts read off a real header —
    and it means the *next* bridge function of either shape is blocked on
    someone having run a `gomobile bind` at least once. Recording the numbers
    would take one bind and a paste; nobody has needed to yet.
18. **(age 0 · value low) A `Header` override gets the row hiding and has to
    build its own control.** `Collapse` deliberately reaches past an override
    for the row emission, because which rows exist is not inside a view the
    caller built — so an override author writes their own button, their own
    `aria-expanded` and their own heading wrapper, with `components.disclosure`
    unexported beside them. Exporting it is one option; a `Collapse.Band` helper
    that returns the default control alone is another. Low because the shape a
    caller wants an override *for* is usually one they want to draw entirely.
19. **(age 0 · value low) A collapsible band's tap target excludes its own
    padding and its badge.** The button fills the space between the band's
    insets and stops where the count begins, which is the ordinary shape of a
    header row with a trailing badge and is a consequence of keeping the count
    announceable. Recorded because the fix is not obvious: moving the chrome
    onto the button would leave the badge without its trailing inset, and the
    band Row is the node `StickyHeader` has to sit on.
20. **(age 0 · value low) A collapsed run is invisible to `OnEndReached`.**
    An infinite feed whose last group is shut has no rows near the bottom, so
    the edge sensor sits on the band and the next page never loads — the reader
    sees a short list of headings and no way to know more exists. Neither
    widget knows the two features are in tension. The honest fix is probably a
    footer that stays reachable, which `LoadMore` already is, plus a note; the
    interesting one is whether a shut trailing group should suppress the
    auto-load rather than starve it.
21. **(age 0 · value low) The gomobile stub's *doc comment* is still prose.**
    The declarations are pinned character-for-character now, and the header
    comment above them — the two-step ObjC/Swift renaming table, the
    nullability note — is a hand-written description of rules that live in
    `gobindSwiftTypes` and the two signature builders. It agrees today because
    it was written from them. That is the same shape item 13 describes one
    level up, at a much smaller scale.
