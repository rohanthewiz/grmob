# Session: the ARIA cluster, and a blocker that was its own absence

Session: https://claude.ai/code/session_01YVcrPiEYG2vnpUePWiK5T5
Date: 2026-09-06 (follows "the-four-oldest")

## Ask

"Work on all the aria related items in the Next list."

Ten of the twenty-three entries were about ARIA. Eight were closed:

| # | age·value | item |
|---|---|---|
| 2 | 5·medium | a hand-built strip's panel cannot say it is one |
| 3 | 5·medium | three widgets take `group` where ARIA has a better role |
| 4 | 5·medium | nothing catches a dangling `aria-controls` or a duplicate id |
| 5 | 5·low | an `AccessibilityID` is not validated |
| 6 | 4·medium | every ARIA claim in the docs is hand-checked prose |
| 9 | 4·low | an `ExpandedState` with no handler is inert on Android |
| 20 | 0·medium | neither web target writes `aria-orientation` |
| 21 | 0·low | a listbox has no typeahead |

Two were not, and the reasons are at the end: item 7 (`AccessibilityExpanded`
has no second consumer) and item 23 (`menu`, `tree`, `grid`).

What most of them shared, again, is the shape the last session named: **the
stated blocker was smaller than it looked, and in one case the blocker was the
item's own absence.**

## Phase 1 — the axis that behaved one way and announced another

`compositeIsVertical` read a container's resolved `flex-direction` to pick the
arrow pair, which is correct — an author who turns a tab strip on its side has
said which way its arrows go. Nothing wrote the answer down, and ARIA's default
for a `tablist` is *horizontal*, so a strip laid out as a `Column` took Up/Down
while telling a reader in browse mode the opposite. `listbox` has the same gap
in mirror, its default being vertical.

The fix is `aria-orientation` on both web targets, derived from the node's own
layout axis exactly as the CSS declaration is. What makes it more than a new
attribute is the second half: **the keyboard now reads it back.** The behaviour
and the announcement were two derivations of one fact, which is how the
divergence opened; they are now one string, and a drift between them is no
longer something that can be written — it would take deleting the attribute.

`COMPOSITE_DEFAULT_VERTICAL` went with it. The keyboard's per-role default and
the announcement's per-role default were always the same table.

`toolbar` is in the table and not in `COMPOSITE_MEMBERS`: it takes the
announcement and no keyboard, because ARIA does not say what a toolbar owns
(buttons, groups, separators, inputs) where the two composite pairs name their
members in the role itself.

## Phase 2 — three widgets, one new vocabulary

`ProgressBar`, `Skeleton` and `FormField`'s required marker were all named
containers, which both web targets rescue with `group` — a role that says these
things belong together and can carry no value at all.

Two of the three needed nothing new. `Skeleton` wants `status`: a skeleton is
not a group of things, it is one advisory that is *replaced*, which is ARIA's
own definition — and it is a live region, so the wait announces itself instead
of waiting to be walked onto. The required marker wants `img`: a glyph standing
in for a word, where `group` invites a reader to announce the label *and* the
asterisk it was replacing.

`ProgressBar` needed the value vocabulary the entry named. Four decisions:

**The numbers are strings.** A bar at the start of an upload is a stated `0`,
and `Style` merges on "non-zero wins" — a `float64` field could not tell it from
an unstated one. `SelectedState` solved the same problem the same way
(`SelectedOff` is the stated string `"false"`), and `core.ValueOf` keeps a caller
from having to think about it.

**They are one field, not three.** The two level ints merge independently *on
purpose*, because they answer disjoint questions. Now/Min/Max are one fact in
three parts — `45` is 45% out of `0..100` and step 45 out of `1..50` — so two
Styles each merging half a range would state a claim neither of them made. A
stated range replaces a stated range whole.

**An unstated range is not an omission.** ARIA spells an *indeterminate* bar by
leaving `aria-valuenow` off, so a bar running with no idea how far is exactly
this role and the zero value. Defaulting a 0 in would pin every one of them at
the start — which is the gap the break-tests found, below.

**`ValueText` is off by default.** ARIA and Compose both announce it *instead
of* the number, and both localize the number themselves; SwiftUI has no numeric
value slot at all. So the default serves three targets and the field exists for
the one it does not.

The natives split over *which half* they can say, which is the tab pair's shape
one layer down: Compose takes the numbers (`progressBarRangeInfo`, which
TalkBack localizes), SwiftUI takes only the words. And `ValueRange.Text` turns
out to be the value channel `GrMobStyle.swift`'s `AccessibilityExpanded` note
says the framework has no room for — the difference being whose words they are.

## Phase 3 — the constant that was blocked by its own absence

`core.Role` had no `RoleTabPanel`, and the entry said why: the WASM runtime told
a panel it had wired from one an author roled **by the value**. `"tabpanel"` was
a string no `core.Role` spelled, so an element carrying it could only have come
from `wireTabPanel`. That worked, and it made *the absence of the constant
load-bearing* — a hand-built strip could mint ids and wire `aria-controls` and
still never name the region it pointed at.

`data-grmob-panel` says the same thing about the *element* rather than about the
vocabulary, in the channel `data-grmob-chrome` already uses. It is a better
answer even setting the constant aside: the old test asked "is this value one no
author could have written", which is a fact about the vocabulary standing in for
a fact about the element.

One case the marker alone cannot decide: an author writing `RoleTabPanel`
themselves puts the wiring's own value in the slot. `applyAccessibility` clears
the marker whenever the Style states a role, which is the only place the runtime
has the Style in hand — and is the same decision `tabPanelBox` makes by reading
the Style directly, which is what keeps the two web targets agreeing about which
pages are panels. `RoleGroup` needs no exemption: the claim is dropped there and
re-earned a moment later, because `canBeTabPanel` accepts `group` on its own
terms.

### A bug the old shape was hiding

Rewriting the unwire path surfaced one. `wireTabPanel` cleared the panel id
**unconditionally**:

```js
setOrRemove(page, "id", wired ? panelId(scope, i) : "");
```

A page carrying its own `core.Style.AccessibilityID` is exactly the page the
wiring stands down for — and it was being stood down for by having that id
deleted. Something else on the page was pointing at that string. `htmlout` never
had it, because a static exporter decides once and writes nothing it did not
decide; the marker is what lets the live target do the same.

The id comes off by **prefix**, not by equality, because a page can move: a tab
reorder leaves page *i* holding the id minted for slot *j*, which is still the
wiring's to clear. The reserved `grmob-` prefix is what makes that safe, and is
now enforced rather than merely documented — see the next phase.

## Phase 4 — the failures nothing could see

Items 4, 5 and 9 are one debug-mode walk of the finished tree, beside the cursor
audit and the duplicate-key check.

A walk rather than a guard in the exporters, because three of the four findings
are facts about *relationships between elements* and no renderer can see one.
Both exporters say so in their own comments — "a dangling IDREF is inert" — and
both are right that they cannot do better from where they sit. A finished tree
can be walked, which is what `core.SetDebugMode` already does twice.

    duplicate-accessibility-id   two elements, one id: invalid HTML, and every
                                 aria-controls resolves to whichever the browser
                                 saw first
    dangling-aria-reference      a control announcing that it governs nothing,
                                 which sounds exactly like a control
    invalid-accessibility-id     whitespace (an id is a single token, so the
                                 element sits there looking right and nothing
                                 can find it), and the reserved grmob- prefix
                                 (which breaks a TabView wiring the app never
                                 wrote and never mentions)
    inert-disclosure             an ExpandedState with no OnClick or
                                 OnLongPress: announced on both web targets and
                                 silently nothing on Android

The line for what belongs here: **would a reader be told something false, with
nothing anywhere saying so.** A state on a role that cannot carry it is already
dropped by both exporters *by design* and documented at each guard, and a
structural role over foreign children is a judgement about content that no walk
can make.

Two details that are decisions rather than mechanics. References resolve *after*
the walk, because the ordinary shape points both ways — a bottom tab bar sits
under the region it switches. And the paths are sorted before the detail string
is built, because the collector deduplicates on that string: an unsorted list
from a map iteration would make one duplicate id read as a fresh finding every
few passes, and the count meant to say "wrong for 200 frames" would sit at 1.

## Phase 5 — the typeahead, and the only state the section owns

Everything in the composite keyboard is derived from the DOM on demand, which is
what makes it survive every patch for nothing. A typed string cannot be derived
from anything, so item 21 is the first thing there that is *held* — and the
holding is kept as small as it can be:

    one buffer, not one per widget   only one thing has focus at a time, so a
                                     second widget's buffer could never be the
                                     live one. The container is recorded beside
                                     the text, so a keystroke elsewhere starts
                                     over rather than continuing someone else's
                                     search.
    a timestamp, not a timer         setTimeout would need cancelling on unmount,
                                     and a widget removed by a patch has no
                                     unmount hook to cancel it from. Expiry is
                                     checked on the next keystroke, which is the
                                     only moment it can matter.

ARIA's two search modes are one line: the query is the first character when
every character is the same (`"sss"` is the third press of s, and cycles) and
the whole buffer otherwise (`"seq"` refines and stays put). The start of the
search follows from it. A `tablist` has no typeahead, which is ARIA's division
rather than a shortcut — a strip's members are all on screen.

The name walk descends to the **leaves** rather than reading `textContent` at
each level, and the harness DOM cannot tell the difference: `dom.mjs` stores
textContent on leaves only, so a container answers `""` and a per-level read adds
nothing. In a browser it concatenates every descendant, and the same code would
count a member's words once per ancestor. That is item 22's shape exactly, and
the check had to be a Go pin rather than a behaviour.

## Phase 6 — one place where an ARIA fact is stated

Item 6. Every role list in this repository was hand-checked prose — three in
`core/style.go`, two more in each web exporter, four `aria-level` scopes, six
`aria-selected` roles, a naming prohibition argued in three places. They agreed
because somebody had read them all.

The failure that proved it was small: a Next-list entry once asserted that
`group` supports `aria-expanded`. It survived three re-sorts, was false, and
nothing in several thousand assertions could have said so.

`aria/verify/testdata/aria.json` states each role's attributes, its ARIA default
orientation and its required children. The tests then ask the *export* rather
than the source: every core.Role is a real role; each of the four state guards
accepts exactly the roles the fixture says support that attribute and refuses
the rest; the orientation table holds ARIA's own defaults for exactly the roles
ARIA scopes it to; `group` is nameable and `generic` is not, which is the whole
premise of the fallback both exporters supply; and the runtime's composite
members are the children ARIA says those containers own.

It is **transcribed, not generated**, and the doc says so — the spec publishes
its role definitions machine-readably, and reading that at build time is
strictly better. What it buys anyway is that the fact is stated *once*: a wrong
entry here is one wrong entry that fails a test the moment a guard disagrees,
where a wrong sentence in a doc comment was one of thirty restatements and
disagreed with nothing. Confirmed by re-introducing the historical false claim:
the fixture fails on it.

It is deliberately partial — only the attributes this framework writes — because
an unchecked entry is prose again with braces around it. Roles core does not
carry are present only where they are the near misses the guards argue with
(`gridcell`, `meter`, `slider`, `treeitem`), which turns "cell is not gridcell"
from a remark into an assertion.

## The break-tests

**Seventy-two run, sixty-six caught first time.** Six results to resolve: three
real gaps, one guard no behavioural test can reach, and two bad mutations of my
own.

**Gap 1 — an indeterminate bar with a range.** Defaulting `aria-valuenow` to
`"0"` when unstated passed everything. The test for the indeterminate case used a
range with *nothing* stated, which returns before the per-member writes are ever
reached; the shape that matters in the wild is a bar that knows its bounds and
not its position. Added.

**Gap 2 — a pin satisfied by a prefix.** `toFloatOrNull() ?: 0f` — the fix
somebody reaches for to make the Kotlin type non-null, and which reintroduces
the exact bug the nullable field exists to prevent — left the pinned substring
standing. This is the third session running with this shape: *a source-text pin
matching a prefix of the mutation.* The pin now takes the whole line, terminator
included.

**Gap 3 — a check nothing drove.** Nothing asserted that `render.Manager`
actually calls `core.AuditTree`. The checks could have been perfectly correct and
never invoked, with every test of them still passing. Held now on both passes,
the way `EndRenderPass` is.

**Gap 4 — the leaf-only text walk**, described above: the shim cannot tell, so
it is a Go pin with the reason attached.

**Not a gap: a guard with no observable effect.** Deleting `AuditTree`'s
`IsDebugMode()` check changes nothing, because `upsertConcern` carries its own as
a backstop. What the guard buys is *cost* — a per-frame tree walk in every
production app — and no behavioural test can distinguish the two. The test was
repointed at what it actually proves and says so, rather than claiming something
it had not measured.

**Not gaps: two mutations of a test file.** Breaking assertions inside
`aria/verify/aria_test.go` and finding no other test complains is expected —
tests are not covered by tests. The right check was mutating the *subject*, which
batch 1 did (`the tablist default flips` → caught).

Same Python harness as the last two sessions: snapshot by content, one
exact-string mutation, run, restore, assert the restore by hashing, and an anchor
that does not appear exactly once is a `SKIP` rather than a silent no-op. Zero
skips and zero restore failures across all three batches.

## Verification

All six paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    wasm/verify/run.sh          replay + 15 mjs suites
    ios/verify/run.sh           flex solver + 9 picker menus + replay + view + app
    android ./gradlew compileDebugKotlin --offline --rerun-tasks
    GOOS=js GOARCH=wasm go build ./...

New files:

    core/value.go                   ValueRange, and why the numbers are strings
    core/value_test.go              the merge-as-a-unit rule and the stated zero
    core/a11y_audit.go              the debug walk, and why it is not in an exporter
    core/a11y_audit_test.go         four findings, both directions each
    htmlout/orientation.go          the role -> default table both web targets read
    wasm/verify/orientation_test.go the two tables held together, and the pin that
                                    keeps the keyboard reading the announcement
    mobile/verify/value_test.go     Compose as a mapping, SwiftUI in both directions
    aria/verify/                    the fixture, its doc, and the conformance tests

Changed:

    core/role.go                    RoleProgressBar, RoleTabPanel, and the rewrite
                                    of "there is deliberately no RoleTabPanel"
    core/style.go                   AccessibilityValue and its guard table
    core/style_props.go             the AccessibilityValue prop
    htmlout/export.go               ariaValue, and aria-orientation in the attrs
    htmlout/tabview.go              the data-grmob-panel marker
    wasm/grmob-runtime.js           orientation; the value family; the marker and
                                    the id bug it uncovered; the typeahead
    components/progress_bar.go      the role, the range, and ValueText
    components/skeleton.go          RoleStatus
    components/form_field.go        RoleImg on the required marker
    render/manager.go               AuditTree on both passes
    android/.../GrMobStyle.kt       ValueRange, grMobValue, two new role arms
    ios/.../GrMobStyle.swift        ValueRange, grMobValueText, two new role arms

## Docs

`core/role.go`: the target table gained a `progressbar` row whose dashes mean
less than the others'; the "no RoleTabPanel" section became the argument for why
there is one now and what replaced the block; every count re-derived (25 roles,
14 inert on both natives).
`core/style.go`, `core/value.go`: the fourth state field, its guard beside the
other three, and the string question.
`docs/platforms/wasm.md`: two new sections — "Which way a composite runs" and
"The value of a valued control" — plus the typeahead and the orientation read-back
in the composite section.
`docs/platforms/exporters.md`: the marker, the two new attribute families, and
the debug pass now reporting what the exporters could only tolerate.
`docs/platforms/native.md`: `AccessibilityValue`, and `RoleTabPanel` joining the
IDREF pair for the same reason.
`docs/concepts/styling-and-theming.md`: `RoleTabPanel` with the strip it
completes, `AccessibilityValue`, the orientation, and the typeahead.
`docs/concepts/debug-mode.md`: four new concern kinds and the audit's own section.
`docs/components.md`: ProgressBar's value channel, Skeleton's `status`,
FormField's `img`.
`ROADMAP.md`: six new entries.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here. Value is the payoff of doing it, not the effort: `high` means
something is being worked around today or a second consumer has arrived,
`medium` means it blocks one named thing, `low` means it is a gap nobody has
bumped into yet.

1. **(age 7 · value low) The gomobile stub's types are unchecked.** The pin
   holds the *names* — every bindable symbol declared, nothing extra — because
   copying gobind's type mapping into a test would be reimplementing gobind. A
   wrong signature fails the Swift typecheck the moment the shell calls it, so
   what is genuinely unguarded is a stub declaration the shell never touches.
2. **(age 5 · value medium) `Accordion` is the only disclosure, and
   `AccessibilityExpanded` has no second consumer.** Deliberately not taken this
   session, because it is the one ARIA entry whose answer is a *widget* rather
   than a semantics change — the field's role list carries five arms nothing
   reaches (`link`, `listbox`, `row`, `columnheader`, `tab`), and building a
   combobox to reach two of them is a feature, not plumbing. **It got cheaper
   twice over here:** a listbox now has arrow keys *and* typeahead, which is most
   of a combobox's popup, and `core.SetDebugMode` now reports a disclosure with
   no handler, which is the failure a new consumer would most likely ship. The
   shapes that reach the other arms are still real and still absent: a
   `GroupedList` band that collapses (row), a `DataTable` with collapsible column
   groups (columnheader).
3. **(age 4 · value low) `Colors.ControlBorder` is 2.92:1 against
   `Colors.Surface` under `DefaultTheme`.** The outer edge clears 3:1 and is
   what identifies a quiet chip, which is argued in two places and asserted in
   one — but any future widget that draws a boundary *on* a Surface fill
   inherits the shortfall without inheriting the argument. The honest fixes are
   both theme decisions: darken past Apple's systemGray, or give the quiet chip
   a fill that is not Surface.
4. **(age 3 · value medium) `Margin` has no per-side props at all.** Padding got
   its four; margin still has only `Margin(all)`, so every single-side margin
   goes through a whole `EdgeInsets` in a `UseStyle` — and that is worse for
   margin than it was for padding, since there is no `MarginHorizontal` or
   `MarginTop` either. Two live workarounds today: `components/separator.go`
   writes `Margin: EdgeInsets{Horizontal: s.Inset}` for its inset rule, and
   `examples/chat`'s bubble writes `Margin: EdgeInsets{Bottom: 8}` for the gap
   between messages. The work is mechanical — the same
   `settleHorizontal`/`settleVertical` helpers apply unchanged, and no renderer
   would move — which is why this is medium and not high: it is scope, not
   difficulty.
5. **(age 3 · value low) A side prop cannot express "clear this side" on a node
   whose axis a later prop will set.** `PaddingLeft(0)` then
   `PaddingHorizontal(16)` gives 16 on both sides, which is correct
   last-one-wins and is also the only way to write the pair. Recorded because
   the settle makes every *other* zero work, and the one remaining hole should
   be written down rather than rediscovered.
6. **(age 3 · value low) `rowsSpec` is shared by exactly two widgets and one of
   its ten fields by exactly one.** `Wrap` exists so DataTable can add a tap
   target and a selection tint that GroupedList has no use for, which makes it a
   widget-specific knob living in a shared type. That is the right trade at two
   callers; at three it would be worth asking whether the decoration belongs in
   the spec or in a wrapper around `appendRows`' output.
7. **(age 2 · value medium) `declaredInk` is now unobservable under both bundled
   themes.** Each pairs white with a fill dark enough that measurement picks
   white too, so an implementation that deleted the first step and only measured
   would paint identical pixels under DefaultTheme *and* MaterialTheme. The rule
   is right and still applies to any theme whose house button is a mid-tone —
   but its whole evidence is now one test fixture (`midTonePrimaryTheme`), which
   is a thinner thread than a bundled theme. The honest options are to leave it
   and accept the fixture, or to give one bundled theme a mid-tone button base
   on purpose, which is a look decision. Same shape one property over:
   `PrimaryOnLight` is an identity on both bundled themes now, so the *reverse
   lookup* has no live consumer either.
8. **(age 2 · value medium) An unsized `ZStack` with a placed layer diverges on
   iOS.** SwiftUI has no per-child stack alignment, so a placed layer is wrapped
   in a filling frame, and a filling frame grows an otherwise unsized stack to
   its parent's proposal — where a Compose `Box` and a CSS grid track stay the
   size of their largest child. Documented in four places and avoidable by
   pinning the stack's box, which `ZStack` already asks for. What would actually
   close it is a SwiftUI layout that places without filling (a custom `Layout`,
   or an `alignmentGuide` scheme that can see the container's size), and neither
   is a small piece of work. Nothing pins the divergence today either —
   `ios/verify` type-checks and replays, it does not measure.
9. **(age 2 · value low) `StackAlign` is inert outside a `ZStack` and nothing
   says so at the call site.** It is inert deliberately and by construction on
   the web (the stack imposes it, so it never reaches a flex child), and by
   omission on the natives (only the stack renderers read the field). But a
   `core.StackAlign` on a `Column`'s child compiles, merges, crosses the wire
   and does nothing on all four targets with no diagnostic. **Cheaper than it
   was:** `core.AuditTree` is now a live tree walk with four findings in it, and
   "a placement on a node whose parent is not an overlay" is the same kind of
   finding in the same pass — a fifth arm rather than a new mechanism.
10. **(age 2 · value low) A `Select`'s group headings cannot be styled or
    ordered independently.** A heading is a string on an option, so there is no
    way to give one an icon, mark a whole run disabled, or state a heading that
    has no options under it. `<optgroup disabled>` exists in HTML and both
    natives could express a disabled section; nothing in `SelectOption` can ask
    for it. `core.SelectMenuSections` is now the one place a section is
    described, so a heading with fields of its own would be a change to
    `SelectMenuSection` and its three transliterations rather than to four
    independent renderers.
11. **(age 1 · value medium) The Kotlin decomposition has no runner.**
    `GrMobSelectMenu.kt` imports nothing precisely so a JVM harness could
    execute it, and `ios/verify` proves what such a harness buys — nine cases
    generated from `core.SelectMenuSections`, compared against the real Swift
    function. Android has no equivalent: `compileDebugKotlin` is the only check
    the build runs, and adding a `src/test` source set means resolving JUnit
    against a `--offline` gradle cache that may not carry it. **A third thing
    now rests on source-text pins alone:** this session's `grMobValue`, whose
    three-way branch on an indeterminate range is checked by
    `strings.Contains` — and this session's own gap 2 was exactly one of those
    pins passing on a prefix of the mutation, the third session running with
    that shape.
12. **(age 1 · value low) The WASM runtime is the fourth copy of the
    decomposition and does not call the authority.** `applySelectOptions` walks
    the flat list itself, appending into a live `<optgroup>`. That is defensible
    — a DOM append has no closing step, so the flush that the other three have
    to remember does not exist here — but it means a change to
    `SelectMenuSection`'s shape has three consumers to update and one to
    remember. The mjs suite covers the behaviour live; what is missing is the
    *shared fixture*, i.e. `wasm/verify/gen.go` emitting the same `menuCases`
    table so the runtime is compared against Go's answer rather than against a
    hand-written expectation.
13. **(age 1 · value low) `styleFromGrMob` has exactly one exemption from its
    own totality rule, and nothing states the rule for adding a second.**
    `delete out.display` for a `Modal` is right: the display is the open/closed
    state, it arrives through the prop channel, and a total pass cannot see a
    prop. But "abstain by deleting the key" is now a technique available to any
    property, and the next one to reach for it will not have this argument
    attached. The test pins the line; what is unwritten is the *test a candidate
    has to pass* — that the property is owned by a channel this function cannot
    read, and that some other path is total for it instead.
14. **(age 1 · value low) The keyboard pattern is verified against a DOM that is
    not a browser.** `wasm/verify/dom.mjs` models attribute storage, parent
    links, listener dispatch and which element holds focus — enough for
    everything the roving tabindex does — but it has no bubbling, no layout, and
    its `focus()` is an assignment. **This session found the first place it
    actually bit:** the typeahead's name walk descends to the leaves rather than
    reading `textContent` at each level, and the shim cannot tell the two apart
    because it stores textContent on leaves only — so the check had to be a Go
    source pin rather than a behaviour. The rest is still true and still
    unverified: that `tabindex="-1"` really removes a `<button>` from the tab
    order, that a disabled control refuses focus, that `preventDefault` on
    `ArrowDown` stops the scroll. The honest answer for all of it is a
    browser-driven pass rather than a wider shim.
15. **(age 1 · value low) `menu`, `tree` and `grid` are patterns `core.Role` has
    no vocabulary for.** Re-examined this session and left alone deliberately.
    The composite section is driven by one two-row table and would take a third
    row without changing shape — but each pattern needs more than arrows: a menu
    has submenus and Escape, a tree has expansion state per node (which is item
    2 looking for a consumer), a grid is two-dimensional. The reason they are
    absent is still "no widget in this repository is one", and adding roles a
    widget could not keep is the move `core.Role`'s own structural rule forbids.
16. **(age 0 · value medium) The ARIA fixture is transcribed, not generated.**
    `aria/verify/testdata/aria.json` makes every guard checkable against one
    statement, which is the whole win — but the statement itself is hand-written
    from the spec, so it can be wrong in exactly the way the prose it replaced
    could be wrong. The W3C publishes the role definitions machine-readably;
    reading that file (checked in, regenerated by a script) would close the last
    hop. Medium rather than low because the fixture is now load-bearing for six
    tests: a wrong entry no longer sits in a doc comment, it *changes what the
    exporters are allowed to do*.
17. **(age 0 · value low) The audit's debug-mode guard has no test that can
    fail.** `AuditTree` returns early when debug mode is off, and deleting that
    line changes nothing observable, because `upsertConcern` carries its own
    guard as a backstop. What the guard buys is cost — a full tree walk per
    frame in every production app — and nothing measures it. The same is true of
    the other two debug checks and always has been; recorded here because this
    is the first one whose cost is a *tree traversal* rather than an atomic load,
    and because a benchmark asserting "off is free" is a small piece of work that
    would cover all three.
18. **(age 0 · value low) `aria-orientation` is announced for `toolbar` and no
    target gives one a keyboard.** ARIA's toolbar pattern has arrow-key
    navigation like the two composites, and `components.ChipStrip` is a toolbar
    of real controls that a keyboard crosses one tab stop at a time. What stops
    it being a third row in `COMPOSITE_MEMBERS` is that ARIA does not name a
    toolbar's members in its role — a toolbar may hold buttons, groups,
    separators and inputs — so the member walk would need a second rule about
    what counts. "Every focusable descendant that is not inside a nested
    composite" is probably that rule, and it is a different shape from the two
    the table encodes today.
19. **(age 0 · value low) A typeahead match does not select, only focuses.**
    ARIA's listbox pattern lets a single-select listbox move the *selection* with
    the focus, which is what makes arrowing through a `<select>` feel right; this
    moves focus and leaves the selection to the author's own `onClick`. That is
    the safe default — a selection that follows focus fires the app's handler on
    every keystroke — and it is also not a choice the framework can make for a
    widget, since `aria-selected` is written from Go state that a keystroke here
    cannot reach without a render pass. Recorded because the asymmetry is real:
    a mouse user selects by clicking one row, and a keyboard user has to arrow
    and then press Enter.
