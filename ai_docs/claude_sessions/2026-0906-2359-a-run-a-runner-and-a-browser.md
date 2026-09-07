# Session: a run nobody could disable, a file nothing ran, and three facts a shim could only restate

Session: https://claude.ai/code/session_017k47JkhG1WBZR5nP29c7Ge
Date: 2026-09-06 (follows "a-band-too-narrow-to-ship")

## Ask

"Work on the next 5 items in the Next list."

| # | age·value | item |
|---|---|---|
| 1 | 3·low | A `Select`'s group headings cannot be styled or ordered independently |
| 2 | 2·medium | The Kotlin decomposition has no runner |
| 3 | 2·low | The WASM runtime is the fourth copy and does not call the authority |
| 4 | 2·low | `styleFromGrMob` has one exemption and nothing states the rule |
| 5 | 2·low | The keyboard pattern is verified against a DOM that is not a browser |

All five closed. Items 1–3 turned out to be one item: the feature in 1 is what
removed 3's justification, and 2's harness is what makes 1 safe on the target
nobody could run.

## Phase 1 — the feature that ended an exemption

`SelectOption.GroupDisabled` marks a whole run unavailable. One design decision
carried the whole session, and it is the reading of *who states it*:

    the first option decides    a caller writing it on the second entry of a
                                run gets nothing, and has no way to find out —
                                a menu is drawn behind a tap, there is no error
                                channel, and the option looks exactly like one
                                that had been read

    any option decides          no silent failure; the cost is that a run's
                                state is not known until the run is CLOSED

"Any" is the reading with no silent failure in it, and it changed the shape of
the rule. Closing a run used to be an append. It is two steps now — the flush,
and a walk back over the items collected before the state was known — so
`closeRun` is a named step in all four renderers rather than an inline append.

### What the propagation is actually for

Every item of a disabled run is marked disabled on its way out, which reads like
a convenience and is not: it is the only mechanism two of the four targets have.
SwiftUI puts `.disabled` on the Button, Compose's dropdown has no section
construct at all, and on the web a run with no heading has no `<optgroup>` to
carry the attribute. So the *section's* flag exists to grey the **heading** —
`<optgroup disabled>`, a disabled SwiftUI `Section` — and the refusal always
rides on the items.

Renderer.swift gains the one modifier in that file that lands on a Section
rather than on its buttons. Its own doc used to say `.disabled` goes on the
Button "and never on the Section: disabling a Section would take its whole run
with it" — which is now the request rather than the accident, and the comment
says which declaration is which. Renderer.kt needed nothing: its heading item is
already `enabled = false`.

### The icon is refused, and that is the answer rather than a gap

An `<optgroup>`'s label is an attribute, so the web can hold text and nothing
else. A heading with an icon on two targets and without one on the other two is
the divergence this widget refuses everywhere else — the same argument that
keeps it off `.pickerStyle(.menu)` and `ExposedDropdownMenuBox`. Recorded as a
decision in `core.Select`'s doc and in `docs/concepts/forms.md`.

### And it ended the WASM runtime's exemption (item 3)

The runtime was the fourth *copy* rather than a transliteration, and the
argument was good: appending to a live DOM has no closing step, so a run ended
when the next option stopped being appended to the open `<optgroup>` — this
target alone had no flush to forget, which is exactly what htmlout, writing
markup, forgot for a release.

`GroupDisabled` reads forward. The run's state is stated by any of its options
and the `<optgroup>` has to be created when the run is *opened*, so no amount of
appending gets past it. `selectMenuSections` now exists in
`wasm/grmob-runtime.js`, `applySelectOptions` is a loop over sections, and all
four renderers are one rule.

Item 3 asked for a shared fixture. It got that too, and more of it than asked.

## Phase 2 — one fixture, three harnesses

`internal/menufixture` is the case table, and the expected answers are never
written: `Cases()` computes them with `core.SelectMenuSections`. It was one
table in `ios/verify/gen.go`; a second harness would have been a second copy of
it, which is the failure the whole mechanism exists to avoid one level down.

Three harnesses read it now:

    ios/verify       runs the Swift function directly
    android/verify   runs the Kotlin one on a JVM            (new, item 2)
    wasm/verify      mounts a picker and rebuilds the sections out of the DOM

The WASM comparison is deliberately *not* against the runtime's internal
function — the runtime exposes `mount` and `patch` and nothing else, and the
interesting subject is the DOM the function produced anyway. `sectionsFromDOM`
walks the `<select>`: each `<optgroup>` is a section, each maximal run of
top-level `<option>` is the headingless section between two of them. That
reconstruction is unambiguous because two ungrouped runs can never be adjacent.

### The table has an admission test

The break-test that would have been a MISS was "a case is dropped". Each
harness carries a count guard, which catches an empty table and nothing else —
a case disappearing while the count stays plausible weakens three harnesses at
once and none of them is in a position to notice, because none of them knows
what the table is *for*.

So `TestTheFixtureCoversEveryPropertyItExistsFor` names eight properties with
the predicate that finds a case covering each. A case may be rewritten or
replaced freely; what it may not do is take the last witness of a property with
it. Two of the eight are the ones a transliteration gets wrong — a run disabled
by an option that is not its first, and a disabled run followed by a live one.

## Phase 3 — `android/verify`: Kotlin without gradle

`GrMobSelectMenu.kt` imports nothing precisely so a plain JVM could run it, and
for two sessions nothing did. The Android build's only check was
`compileDebugKotlin`, which proves the file parses.

The Next item named the blocker as "a `src/test` source set means resolving
JUnit against a `--offline` gradle cache that may not carry it". The cache does
carry JUnit here — and the source set is still the wrong shape, for a reason
that survives that: a check which only runs on a machine that has already
downloaded the right test dependencies is a check nobody runs, and it would drag
the whole AGP pipeline in to execute a function touching no Android API.

So the script compiles two files and runs them. `kotlinc` if one is on PATH;
otherwise the compiler jars the gradle cache already holds. **Six** jars, which
is worth writing down because the first five are not enough:

    kotlin-compiler-embeddable   deliberately not a fat jar
    kotlin-stdlib                also on the compiler's OWN classpath
    kotlin-reflect               ArgumentUtilsKt reads annotations reflectively
    kotlin-daemon-embeddable
    kotlinx-coroutines-core-jvm
    org.jetbrains:annotations    the code generator stamps @NotNull onto every
                                 non-nullable parameter — so a hello-world
                                 compiles without it and anything with a
                                 function signature does not

No jars, no java, no kotlinc → **SKIP**, the stance `ios/verify` takes toward a
missing iPhoneOS SDK.

### The table crosses as Kotlin source, not JSON

The one place this harness differs from `ios/verify`, and the reason is the
standard library. Swift decodes a JSON transcript with `Decodable`; Kotlin's
standard library has no JSON parser and neither does the JDK. A JSON fixture
here would mean a dependency — the thing the harness exists to avoid — or a
hand-written parser standing between the fixture and the code under test. So
`gen.go` emits Kotlin literals and the only code in the harness is the
comparison. The wire keys are written in a fixed order, because Go's map
iteration is randomised and a fixture that reordered itself every run would make
any diff of the generated source unreadable.

`mobile/verify/select_test.go` shrank accordingly — but not to nothing. Its
pins are what runs under a bare `go test ./...`, where neither harness does.

## Phase 4 — the rule for a second exemption (item 4)

`styleFromGrMob` deletes a `Modal`'s `display` rather than assigning it, because
that property IS the dialog's open/closed state and the `visible` prop owns it.
The line was pinned. What was never written down is the test a *second*
exemption has to pass — and "abstain by deleting the key" is now a technique
available to any property.

Three conditions, and the second is the one that is easy to miss:

1. Some **prop** owns the property. Totality is not being dropped, it is being
   handed over, and it has to be a prop — a Style field would have been assigned.
2. That owner writes the property in **every** state. `visible ? "flex" : "none"`
   qualifies; a prop that assigns only when truthy does not, because the value
   it wrote last would then stand forever. That is the stale-declaration bug
   totality exists to prevent, moved one channel over rather than fixed.
3. The exemption is keyed on the node type.

### Why the source scan is the load-bearing part

The first thing written was a behavioural sweep: mount every node type with a
full Style, patch an empty one, and require the element to end up identical to a
freshly built styleless one. It passes, it catches a *guarded write* (the other
way to break totality), and it is **blind to an exemption** — because a deleted
key never reaches the element on either mount, so an abstention is
indistinguishable from a property nobody asked for.

A break-test found that, not a reading of the code. The mutation
`delete out.borderRadius` was caught by exactly one of five tests.

So `EXEMPTIONS` is a table, and the source is scanned for `delete out.X` and
held to it: a new deletion fails until a row explains it, and a row whose
deletion is gone fails too. The behavioural tests then drive the owning prop —
including driving every state on **one** element in sequence, which is what a
live dialog does and what a fresh mount per state would not catch.

## Phase 5 — three facts a shim could only restate (item 5)

`dom.mjs` is a faithful model of the runtime's bookkeeping and a poor model of a
browser. Three of the keyboard pattern's claims are about the browser:

- `tabindex="-1"` really takes a `<button>` out of the tab order — the whole
  reason a tab strip is one stop rather than five
- a disabled control refuses focus
- `preventDefault` on `ArrowDown` really stops the page scrolling

The Next item's own note said the honest answer is a browser-driven pass rather
than a wider shim, and the reason is sharper than "the shim is incomplete":
there, `tabindex` is a string nobody reads, `focus()` is an assignment and
`defaultPrevented` is a flag the shim set itself. Widening it could only ever
restate the claims.

`wasm/verify/browser.mjs` drives a headless Chrome over the DevTools protocol
with Node's built-in `WebSocket`. No npm, no lockfile, no `node_modules`, no
network — `run.sh`'s promise intact. Keys go through `Input.dispatchKeyEvent`,
so the tab order is walked by the browser's own focus algorithm and a scroll is
a real scroll. It skips when there is no Chrome, or on a Node before v21.

Three details are load-bearing and each was found by the harness failing:

    sentinel buttons either side    "the strip is one stop" is a statement
    of the mount point              about where focus goes NEXT, so there has
                                    to be a next

    a 4000px filler                 a document that cannot scroll passes the
                                    scroll check whether or not the key was
                                    consumed — and the control half (the same
                                    key with nothing composite focused MUST
                                    scroll) is what keeps it from being vacuous

    --disable-smooth-scrolling,     the first run read scrollY one frame after
    and a beat after the key        the key and got 3px: a smooth scroll is an
                                    animation, so "did not scroll" and "has not
                                    finished scrolling" were one measurement

## The break-tests

**Twenty-four run, twenty-four caught, zero MISSED, zero skips. Zero restore
failures.**

Same harness as the last six sessions — snapshot by content, one exact-string
mutation, run the suite, restore, assert the restore by hashing, and an anchor
that does not appear exactly once is a `SKIP` — with mutations routed to five
different suites (`go test`, `ios/verify`, `android/verify`, `node --test`,
`browser.mjs`).

Worth recording what caught what.

- **The three Kotlin mutations were caught by `android/verify`.** Before this
  session the spelling mutation would have been caught by `mobile/verify`'s
  `strings.Contains` and the other two by nothing at all.
- **`delete out.borderRadius` was caught by the source scan alone**, which is
  the finding that rewrote phase 4's own claims about its sweep.
- **The three browser mutations were caught by `browser.mjs` alone.** Removing
  `e.preventDefault()` before `moveCompositeFocus` passes every existing suite:
  `keynav_test.mjs` asserts `defaultPrevented`, which the shim sets itself.
- **One mutation was MISSED on the first run** — dropping a case from the shared
  fixture — and that is what `TestTheFixtureCoversEveryPropertyItExistsFor` was
  written for. A second attempt at the same mutation, aimed at a case that was
  the *only* witness of two properties, is caught.

## Verification

Seven paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    wasm/verify/run.sh          replay + 16 mjs suites + BROWSER PASS
    ios/verify/run.sh           flex + stack solvers + 14 picker menus
                                + replay + view + app
    android/verify/run.sh       14 picker menus, on a JVM          (new)
    android ./gradlew compileDebugKotlin --offline --rerun-tasks
    GOOS=js GOARCH=wasm go build ./...

New files:

    internal/menufixture/menufixture.go       the shared case table
    internal/menufixture/menufixture_test.go  its admission test
    android/verify/gen.go                     the table, as Kotlin source
    android/verify/Harness.kt                 the comparison, and main()
    android/verify/run.sh                     kotlinc, or the gradle cache
    wasm/verify/totality_test.mjs             the exemption rule, as a table
    wasm/verify/browser.mjs                   the three facts, in a Chrome

Changed:

    core/input.go                  GroupDisabled; the wire key; the icon
                                   decision
    core/select_menu.go            Section.Disabled; closeRun; the header's
                                   account of four transliterations
    core/select_menu_test.go       five cases for the run-level flag
    core/select_test.go            the flattening seam writes only what was said
    htmlout/export.go              <optgroup disabled>
    htmlout/select_test.go         both halves, including the headingless run
    ios/GrMob/Runtime/GrMobSelectMenu.swift   closeRun and the propagation
    ios/GrMob/Runtime/Renderer.swift          .disabled on the Section
    ios/verify/{gen.go,selectmenu.swift}      onto the shared fixture
    android/.../GrMobSelectMenu.kt            closeRun and the propagation
    wasm/grmob-runtime.js          selectMenuSections; the optgroup attribute;
                                   the exemption rule stated at its site
    wasm/verify/gen.go             the transcript is an object now
    wasm/verify/load.mjs           loadTranscript
    wasm/verify/{replay,select}_test.mjs      the new shape, and four new tests
    wasm/verify/{dom,keynav_test}.mjs         what needs a browser, and why
    wasm/verify/run.sh             the browser pass, wired
    mobile/verify/select_test.go   re-pinned, plus the propagation

## Docs

`docs/platforms/native.md`: `GroupDisabled` across the two natives, both
harnesses named, and a section on why `android/verify` is not a gradle test
source set. `docs/platforms/wasm.md`: the three keyboard facts and the browser
pass; the three conditions a second totality exemption must satisfy.
`docs/platforms/exporters.md`: `<optgroup disabled>` and why the per-option
attribute is still written beside it. `docs/concepts/forms.md`: `GroupDisabled`,
the "any option" reading, and the icon decision. `README.md`: the four verify
commands, and that none of them needs a device or the network. `ROADMAP.md`:
five new entries.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here. Value is the payoff of doing it, not the effort: `high` means
something is being worked around today or a second consumer has arrived,
`medium` means it blocks one named thing, `low` means it is a gap nobody has
bumped into yet.

1. **(age 3 · value low) `menu`, `tree` and `grid` are patterns `core.Role`
   has no vocabulary for.** Each needs more than arrows: a menu has submenus
   and Escape, a tree has expansion state per node, a grid is two-dimensional.
   `components.disclosure` is the heading-around-button shape a tree's twisty
   needs and `Collapse` is the caller-owned expansion state a tree node would
   want; what is still absent is the recursion and the `treeitem` role, and
   the reason remains that no widget in this repository is one.
2. **(age 2 · value medium) The ARIA fixture is transcribed, not generated.**
   `aria/verify/testdata/aria.json` makes every guard checkable against one
   statement, but the statement is hand-written from the spec and can be wrong
   the way the prose it replaced could be wrong. The W3C publishes the role
   definitions machine-readably. The same shape now exists five times —
   `gobindSwiftTypes`, `wantWitnesses`, `wantRowsSpecFields`, this session's
   `EXEMPTIONS` and the ARIA fixture are all hand-written tables other tests
   are held to — the difference being that the first four are one to eleven
   rows, where the ARIA fixture is a spec.
3. **(age 2 · value low) The audit's debug-mode guard has no test that can
   fail.** `AuditTree` returns early when debug mode is off, and deleting that
   line changes nothing observable, because `upsertConcern` carries its own
   guard. What the guard buys is cost, and nothing measures it. A benchmark
   asserting "off is free" would cover all three debug checks.
4. **(age 2 · value low) `aria-orientation` is announced for `toolbar` and no
   target gives one a keyboard.** ARIA's toolbar pattern has arrow-key
   navigation, and `components.ChipStrip` is a toolbar of real controls a
   keyboard crosses one tab stop at a time. What stops a third row in
   `COMPOSITE_MEMBERS` is that ARIA does not name a toolbar's members, so the
   member walk needs a second rule — "every focusable descendant not inside a
   nested composite" is probably it. `browser.mjs` is now the place a new
   composite's tab-order claim can actually be checked.
5. **(age 2 · value low) A typeahead match does not select, only focuses.**
   ARIA's listbox pattern lets a single-select listbox move the selection with
   the focus. This moves focus and leaves selection to the author's `onClick`,
   which is the safe default and also not a choice the framework can make,
   since `aria-selected` is written from Go state a keystroke cannot reach
   without a render pass.
6. **(age 2 · value medium) The gobind type table cannot describe two of the
   shapes it refuses.** `swiftType` rejects a returned bound interface and
   `swiftResult` rejects a multi-result signature, in both cases with a
   message saying to read `Headers/Mobile.objc.h` and add the row. That is the
   right stance for a table that may only hold facts read off a real header —
   and it means the *next* bridge function of either shape is blocked on
   someone having run a `gomobile bind` at least once.
7. **(age 2 · value low) A `Header` override gets the row hiding and has to
   build its own control.** `Collapse` deliberately reaches past an override
   for the row emission, so an override author writes their own button, their
   own `aria-expanded` and their own heading wrapper, with
   `components.disclosure` unexported beside them. Exporting it is one option;
   a `Collapse.Band` helper that returns the default control alone is another.
8. **(age 2 · value low) A collapsible band's tap target excludes its own
   padding and its badge.** The button fills the space between the band's
   insets and stops where the count begins, which is a consequence of keeping
   the count announceable. The fix is not obvious: moving the chrome onto the
   button would leave the badge without its trailing inset, and the band Row
   is the node `StickyHeader` has to sit on.
9. **(age 2 · value low) A collapsed run is invisible to `OnEndReached`.**
   An infinite feed whose last group is shut has no rows near the bottom, so
   the edge sensor sits on the band and the next page never loads. Neither
   widget knows the two features are in tension. The honest fix is probably a
   footer that stays reachable, which `LoadMore` already is, plus a note; the
   interesting one is whether a shut trailing group should suppress the
   auto-load rather than starve it.
10. **(age 2 · value low) The gomobile stub's *doc comment* is still prose.**
    The declarations are pinned character-for-character now, and the header
    comment above them is a hand-written description of rules that live in
    `gobindSwiftTypes` and the two signature builders. It agrees today because
    it was written from them.
11. **(age 2 · value medium) The `ControlBorder` retint is not visible on any
    screenshot.** Now two retints deep: `AmberTheme` adds a whole palette that
    no verification path has ever *looked* at. Four renderers emit its hexes
    and all four passes are arithmetic, type-checks or DOM shims. **This
    session moved the goalposts:** `browser.mjs` runs a real Chrome over the
    real runtime, so a screenshot of a themed screen is now a
    `Page.captureScreenshot` away — what is still missing is what to compare it
    against, which is the whole question a palette poses.
12. **(age 2 · value low) `boundaryBackdrops` is a list of fills somebody
    remembered.** The census crosses the tone with `Background`, `Surface`,
    `Card`, `Input` and `TextArea`, named by hand, with `Camera` excluded by
    name. A new `ComponentDefaults` field carrying a `Background` — a `Sheet`,
    a `Popover` — is a backdrop a control can sit on and the census would not
    know. Reflecting over `ComponentDefaults` for non-empty `Background`s would
    close it, at the cost of needing the `Camera` exclusion to survive as data
    rather than as a line in a slice literal.
13. **(age 2 · value low) The wrapper test is a rule about a slice nobody
    returns.** `appendRows` appends into the caller's `[]core.PropsAndChildren`
    and returns it, so "can this knob be done to the result" is answered
    against a slice that also holds the container's own props. The opacity
    assertion measures the children it emitted, which is the right subject, but
    a real wrapper would have to find them among the props first.
14. **(age 1 · value medium) The overlay `Layout` is measured and has never
    been mounted.** `GrMobStackSolver` is checked hard, and `GrMobStackLayout`
    — the part that proposes sizes to subviews and reads the `LayoutValueKey`
    — is only type-checked. Three things it does are judgement calls a
    simulator would settle in a minute and no test can: measuring children with
    the incoming proposal rather than an unspecified one, *not* clamping the
    container to the proposal, and re-proposing `bounds.size` at placement.
    Each is argued in the file; none is observed.
15. **(age 1 · value low) `AmberTheme` is not used by any example app.**
    Three palettes ship and the tutorial's theme chapter still lists two by
    hand (`examples/tutorial/chapter7.go` builds its own slice). Nothing is
    wrong; it is that the new theme's only readers are tests, so the one
    question a bundled theme exists to answer — does a real screen look right
    in it — has not been asked.
16. **(age 1 · value low) `wantWitnesses` is a hand-transcribed table that a
    fourth theme silently widens.** The census derives its *theme list* from
    `core.BundledThemes()`, so a new palette is asked every rule — and its
    answers land as a diff against a map written by hand. What is not stated
    anywhere is what a new row should look like when somebody is adding a theme
    rather than debugging one: "witnessed by nobody" and "witnessed by
    everybody" are both legal and mean opposite things.
17. **(age 0 · value medium) `grMobValue`'s three-way branch is still checked
    by `strings.Contains`.** The runner it was waiting for exists now, and the
    branch is still out of its reach, because `grMobValue` is a
    `SemanticsPropertyReceiver` extension and the harness has no Compose. The
    *decision* is import-free — given a text, a now, a min and a max, which of
    "state a range", "state indeterminate" and "say nothing" — so the move is
    the one `GrMobSelectMenu.kt` already made: split the branch into a pure
    function in a file that imports nothing, leave `grMobValue` as the two
    lines that apply it, and `android/verify` picks it up. What that needs
    first is a Go authority to compare against, and there is none — htmlout
    passes the three ARIA attributes straight through, which is a different
    mapping from Compose's three-way one.
18. **(age 0 · value low) A `Spacer`'s size prop clobbers its own Style.**
    `applySpacerSize` writes width, height and flex-shrink, and `renderNode`
    calls it *after* `createElement` — so a Spacer carrying a `Width` in its
    Style loses it to a prop that is not exempt from anything. This is a write
    ordering rather than an abstention (`styleFromGrMob` still manages all
    three), which is why `totality_test.mjs` leaves Spacer out of its sweep and
    says so. Nothing in `core` builds such a node; a hand-assembled one reaches
    it.
19. **(age 0 · value low) A heading with no options under it is still
    inexpressible.** A `Group` is a string on an option, so a section that
    exists to say "nothing here yet" has no way to be written. `<optgroup>`
    renders empty in every browser and both natives could draw a bare header,
    so the blocker is authoring rather than rendering: it needs a heading to be
    a thing declared independently of an option, which means a second list and
    a matching problem the run-based reading was chosen to avoid. The nearest
    thing today is a disabled run holding one explanatory option.
20. **(age 0 · value low) `browser.mjs` is one page and three facts.** It
    proves the shape works and stops there. `enterkeyhint` relabelling a soft
    keyboard and `focus()` opening one still need a device, but several things
    that need only a browser do not have a check: whether the focus ring is
    visible on a themed control, whether an `aria-live` region actually
    announces, whether a `<select>`'s menu opens where the theme's frame says
    it should. Adding one is now a function call rather than a project.
