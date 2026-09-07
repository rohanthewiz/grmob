# Session: a fixture that was wrong four times, a toolbar nobody could cross, and a check that could only pass

Session: https://claude.ai/code/session_017k47JkhG1WBZR5nP29c7Ge
Date: 2026-09-07 (follows "a-run-a-runner-and-a-browser")

## Ask

"Work on the next 5 items in the Next list", then mid-session: "do the next two
items then wrap the session."

| # | age·value | item |
|---|---|---|
| 1 | 3·low | `menu`, `tree` and `grid` are patterns `core.Role` has no vocabulary for |
| 2 | 2·medium | The ARIA fixture is transcribed, not generated |
| 3 | 2·low | The audit's debug-mode guard has no test that can fail |
| 4 | 2·low | `aria-orientation` is announced for `toolbar` and no target gives one a keyboard |
| 5 | 2·low | A typeahead match does not select, only focuses |
| 6 | 2·medium | The gobind type table cannot describe two of the shapes it refuses |
| 7 | 2·low | A `Header` override gets the row hiding and has to build its own control |

All seven closed. Three of them turned out to be one shape — items 1, 4 and 5
are all "the composite machinery has a rule it cannot express" — and two of them
turned out to be the same *mistake*: items 2 and 6 both had a fact this
repository could read, sitting in a file it already had, being deferred to a
document somebody would have to go and generate.

## Phase 1 — the ARIA fixture, generated (item 2)

`aria/verify/doc.go` had said this itself, two sessions ago, under a heading
called "What this is not":

> It is not generated. The ARIA specification publishes its role definitions in
> a machine-readable form, and transcribing them by hand is a weaker thing than
> reading that file at build time — a transcription can be wrong in exactly the
> way the prose it replaces could be wrong.

It was wrong. Generating the fixture for the first time changed four facts:

    list.requiredOwned held `group`      a listbox's allowance, not a list's,
                                         and the two had been read as one
    radiogroup.orientation held a value  ARIA 1.1 gave it one; 1.2 removed it.
                                         The role still takes the attribute
    treegrid.orientation held a value    which no version of ARIA states
    nameProhibited held `term`, `time`   both take an author name in 1.2

**Every one of the four was in a near-miss row, and nothing failed when they
were corrected.** That is the finding rather than a footnote: near misses exist
so a guard has something to argue with, which makes them the rows least likely
to be argued with — so the fixture's weak entries are exactly the ones no test
reaches, and checking them by hand was never going to be the fix.

### The parser is regexes, and that is the right size

The spec's role sections are ReSpec output, not editorial prose:

    <section class="role notoc" id="toolbar">
      <table class="role-features">
        <tr><th class="role-properties-head">…  <td class="role-properties">…
        <tr><th class="role-inherited-head">…   <td class="role-inherited">…
        <tr><th class="role-namefrom-head">…    <td class="role-namefrom">author

Every fact wanted is a cell, and every cell's content is a run of anchors. A
full HTML parser would buy nothing and would mean a dependency —
`golang.org/x/net` is not in this module. Four details are load-bearing and
three of them were found by a break-test:

    three attribute cells, not two   `role-required-properties` is where
                                     aria-selected on `option`, aria-level on
                                     `heading` and aria-valuenow on every range
                                     role live, and NOWHERE else. A parser
                                     reading only Supported and Inherited
                                     reports that ARIA forbids aria-selected on
                                     an option, and every guard held to that
                                     fixture then refuses the attribute the
                                     whole listbox pattern rests on
    section depth counting           several role descriptions hold a <section>
                                     of their own, and taking the first </section>
                                     truncates the feature table silently — the
                                     role parses as one that supports nothing
    the prohibited cell subtracted   `caption` inherits aria-label and then
                                     forbids it
    synonym resolution               `none` and `presentation` are mutual
                                     synonyms and only one carries a table

### The floors are the defence against a silent format change

Every failure this parser can have produces *empty cells* rather than an error:
a renamed class, a reworded sentence, a restructured table all yield roles that
support nothing and orient nothing — which reads downstream as ARIA having said
no. Only the document's own shape tells that apart from the truth, so `Parse`
refuses a document with under 80 roles or under 8 implicit-orientation
sentences.

The orientation floor is the half a role count cannot cover: a document with
every role intact and the sentence reworded parses to 94 roles, none with a
default axis.

### The offline promise is intact, and the division is stated

    read from ARIA              which attribute each role supports, its implicit
                                orientation, its required owned elements, the
                                name prohibition
    chosen here                 spec.InScopeAttributes (nine), spec.NearMisses

Both remaining lists are *selections* rather than claims, so neither can be
wrong the way a transcribed fact can. `sh aria/fetch.sh` is the one thing in
this repository that touches the network and nothing on a verification path
depends on it: the fixture is committed, and `TestTheFixtureIsWhatTheSpecificationSays`
**skips** when no download is present — the stance `ios/verify` takes toward a
missing iPhoneOS SDK.

The cost is named rather than hidden: on a machine with no spec copy that is a
SKIP, so a hand-edited fixture could reach a commit. What stops it going further
is that the same edit fails the moment anyone runs the fetch, and that the file
now says GENERATED on line 2 with the command on line 3.

## Phase 2 — three guards, measured (item 3)

The three debug checks are each written the same way and each is
**unobservable**: `upsertConcern` carries its own `IsDebugMode` backstop, so
deleting any of them leaves the behaviour identical and every behavioural test
passing. `TestTheAuditIsSilentWithDebugModeOff` said so in as many words and
said that it was therefore asserting less than it looked like.

The item asked for a benchmark. A benchmark reports a number and passes whatever
the number is, which would have re-created the problem one level up.
`testing.AllocsPerRun` is the assertable form of the same measurement — it pins
GOMAXPROCS and warms up, so an exact count is a legitimate assertion rather than
a threshold somebody will keep raising.

    AuditTree            0 off, 289 on over a 40-row tree
    EndRenderPass        0 off, 3 on
    renderAll            the guard saves exactly checkDuplicateKeys' own 3

The third is a difference rather than an absolute, because the guard is at the
*call site* and `renderAll` allocates on every path — building the child slice
is its job. The difference is taken against the check's own cost measured in the
same test, so it is self-calibrating: a change to how the check sizes its map
moves both sides, and deleting the guard makes the left side zero.

Each pair asserts the **on** side too. An assertion that a guarded call
allocates nothing is vacuous if the thing behind the guard allocates nothing
either, and would go on passing after the work it guards was deleted.

## Phase 3 — a toolbar's members, which no role names (item 4)

`toolbar` had been in the orientation table and out of `COMPOSITE_MEMBERS` for
two releases: the axis announced, no tab stop moved. The reason was real — ARIA
calls a toolbar "a collection of commonly used function buttons or controls" and
defines no `toolbaritem`. The cost was equally real: `components.ChipStrip` is a
`Row` of `core.Button`s, so a twelve-chip filter bar was twelve stops in the
page's tab order where ARIA promises one.

**The rule the absence forces**: every focusable control not inside a nested
composite. Two ways to be one, and they are the two ways this framework builds a
control at all — a natively focusable tag, or a container carrying `RoleButton`
or `RoleLink` *with* an `OnTap`.

Deliberately not "anything carrying `tabindex`": this section writes `tabindex`
onto every member it finds, so a membership test that read the attribute would
answer differently on the second sync than on the first, and every member would
stay one forever.

### Two tables, not one with a sentinel

    COMPOSITE_MEMBERS      a fact about ARIA's vocabulary, pinned against
                           core.Role by wasm/verify/keynav_test.go
    COMPOSITE_FOCUSABLE    a decision about a walk

Collapsing them would make the pin read a sentinel as a role name. The pin also
asserts the two tables are disjoint — `compositeMembersOf` asks the role-named
one first, so a role in both would silently take the wrong walk.

### The two walks disagree about a nested composite, deliberately

`compositeMembers` descends *through* a composite of the other kind, because an
`option` below a `tablist` is still the listbox's option — the roles say whose
it is. Nothing says whose a `<button>` is. So the focusable walk stops.

The consequence is asserted rather than hidden: a tablist inside a toolbar keeps
its own roving tabindex, so that shape is **two** tab stops rather than one.
That is not what ARIA describes, and it is the honest outcome of a rule that
will not guess — every control stays reachable, which the alternatives lose.

`compositeOf` had to grow the mirror image, and getting it wrong first was
instructive: a single walk-out that returned `null` at the first composite broke
the existing "the walk out and the walk in agree about a crossed pair" test,
which is exactly the invariant its own comment had been claiming for a release.

## Phase 4 — selection follows focus (item 5)

The standing argument had two halves and the *premise* was the false one:

> This moves focus and leaves selection to the author's `onClick`, which is the
> safe default and also not a choice the framework can make, since
> `aria-selected` is written from Go state a keystroke cannot reach without a
> render pass.

`activateCompositeMember` has called `GoInvokeCallback` since the day the
pattern was written. `Enter` on a member has always reached Go and always
produced exactly that render pass. What was missing was not a mechanism, it was
a *decision about when* — and that is not one the framework can make, because
ARIA recommends it for tabs over cheap panels and warns about it for a listbox
whose selection is expensive. So it is a prop.

`core.AccessibilitySelectionFollowsFocus()` goes on the **container**. Three
things are load-bearing:

    moveCompositeFocus is the hook      the single funnel for the arrows, Home,
                                        End and a typeahead match — so a typed
                                        jump selects, which is the case the item
                                        was named after
    unreachable from syncComposite      a selection fired from the patch pass
                                        calls into Go, produces a patch, and
                                        fires again
    no NATIVELY_ACTIVATED guard         activation returns early for a <button>
                                        because Enter already makes the browser
                                        fire a click. An arrow fires nothing, so
                                        the guard would be wrong in the case
                                        that matters most — a tab strip is
                                        buttons all the way down

There is no ARIA attribute for it, because ARIA says what a widget *is* and this
says what its keyboard does. So it rides as `data-grmob-selection-follows-focus`,
written totally like everything beside it. `htmlout` writes nothing (no key
handler, same argument as the roving tabindex) and neither native reads it
(no arrow keys — both readers cross a collection by swipe).

## Phase 5 — the refusal that could not go stale (item 1)

Three sessions of "core has no vocabulary for `menu`, `tree` and `grid`", in
four places, agreeing because somebody read them all. The honest closure was not
to add the roles — a role with no arm falls into a catch-all and is silently
inert, which `core/role.go`'s own header warns about — but to make the refusal a
thing that can fail.

`aria/verify/refusals_test.go` is six rows, each naming the container role, the
member roles its arrows move between, which half is still the first blocker, and
what shape the walk would need. The mechanism is that **`Blocked` is computable**
from `core.Roles()`, so writing it down turns "somebody will notice" into a
failing test that names the row and quotes what is left of the argument.

Two findings came out of building it:

- **`treegrid`'s vocabulary is already complete.** Its members are `row`s, core
  carries `RoleRow`, and a row takes aria-level and aria-expanded — which core
  spells as `AccessibilityNestingLevel` and `AccessibilityExpanded`, both
  already scoped to that role. Nothing is missing but the walk, and the walk is
  the hardest one on the list. The first draft of the table said otherwise and
  the test said so.
- **`radiogroup` is the one whose walk this machinery could already do** — a
  flat run of members, one selected, arrows between them, which is a listbox
  with a different word. It is absent because nothing builds a radio group.

The worked example is `toolbar`, which is no longer in the table. Both halves of
its refusal turned out false in an interesting way: the missing member role was
not missing, it was *never going to exist*. Once that was the reading, the walk
was twenty lines.

## Phase 6 — the gobind table reads gobind (item 6)

Two refusals, both saying in effect "read it off `Headers/Mobile.objc.h` and add
the row" — which made the next bridge function of either shape blocked on
somebody having run a `gomobile bind`, on a Mac with Xcode, for a fact **sitting
in the module cache the whole time**. `golang.org/x/mobile` is in `go.mod`, held
there by the `tool` block that pins the toolchain the stub is written against.

`bind/genobjc.go` settles both:

    objcParamType     special-cases exactly one Go type, String, and falls
                      through to objcType for everything else
    objcType          an implementable bound interface is `id<Prefix Name>
                      _Nullable`, in every position
    funcSummary       three result arms, not two

So the asymmetry the table's two columns exist for is `string` **and nothing
else**, and a returned protocol is `GrMobXProtocol?` exactly as a parameter is.
That refusal is gone.

The multi-result refusal stays and now names which arm applies. The third arm is
the one worth having written down: **three or more results gobind refuses
outright** (`g.errorf("too many result values")`), so that is not a gap in this
table — `gomobile bind` will not build the function — and reporting it as a
missing mapping would send the next person to read a header for a declaration
that was never generated.

`gobindVersion` pins the version the readings were made at, checked against
`go.mod`. It is not a claim that a newer gobind is wrong; it is the only moment
at which anyone would look, because both shapes this file describes are refusals
and no bound function has either.

## Phase 7 — `components.CollapseBand` (item 7)

`Collapse` reaches past a `Header` override for the row emission and stops at it
for the control, and both halves of that are right: an override's run still
hides, and a band the widget also built would be a second control for the same
run.

What it left the override author holding was three things at once — a button, an
`aria-expanded` stated on every pass, and a heading wrapper whose nesting order
is four paragraphs in an unexported type. The shape most likely to come out of
that is one of the two `Accordion` tried and discarded, **both of which look
correct in an export**.

`CollapseBand` is the disclosure alone: no Surface, no padding, no count badge.
It takes the caller's own `Collapse`, which is what keeps the control and the
row hiding answering to one state. An inactive `Collapse` builds a plain heading
rather than a control with nothing behind it — a stated expansion with no
handler is precisely what `core.AuditTree` reports as `ConcernInertDisclosure`,
so building one here would be building the thing the audit exists to find.

The load-bearing test compares it against `GroupHeader` rather than pinning it,
on the argument `TestBothDisclosuresBuildTheSameShape` already makes: a third
construction of this arrangement checked only against its own expectations would
drift the first time either of the other two was touched.

## The check that could only pass

The best finding of the session came out of a break-test, and it is about the
harness rather than the code.

`browser.mjs`'s new toolbar check **hung for its whole timeout** the first time
its subject was broken, having passed every time it worked. The cause:

    await evaluate(`new Promise((d) => setTimeout(
        () => requestAnimationFrame(() => requestAnimationFrame(() => d(true))), 50))`)

A key that changes *nothing* — no focus move, no scroll, no DOM edit — gives the
browser no reason to paint, so `requestAnimationFrame` is never called back.
Which is precisely the case every check in that file is trying to detect. **A
check that waits for a frame waits for its own subject to work**, and the three
existing checks had never revealed it because every key they send does something
when the runtime is correct.

Two fixes, both improvements regardless of the bug:

- the wait races a wall clock beside the frames (resolving twice is harmless)
- every DevTools round trip is bounded at 15s, so an unanswered call is a
  failure rather than a hang

A hang is strictly worse than a failure: it is what a CI run does for its whole
timeout instead of reporting in a minute. The break-test harness grew a timeout
of its own for the same reason, and a `HUNG` verdict distinct from `MISSED`.

## The break-tests

**Thirty-five run, thirty-five caught, zero MISSED, zero restore failures — and
one HUNG before the harness was fixed.**

Same discipline as the last seven sessions — snapshot by content hash, one
exact-string mutation, run the suite, restore, assert the restore by hashing,
and an anchor that does not appear exactly once is a `SKIP` — now with a hard
per-mutation timeout, since this was the session that found out why one is
needed.

    item 2   the ARIA parser              6
    item 3   the debug guards             3
    item 1   the refusals table           3
    item 4   the toolbar walk             6
    item 5   selection follows focus      8
    item 6   the gobind table             4
    item 7   CollapseBand                 5

Worth recording what caught what.

- **Dropping `role-required-properties` was caught by both layers** — the
  offline parser test (`option` loses aria-selected) and the conformance test
  against the real spec. That is the shape both were written for: the unit tests
  hold the parser without the download, the conformance test holds the fixture
  with it.
- **Two mutations were caught only by the conformance test**: the truncating
  section scan and the reworded orientation sentence both produce a *plausible*
  fixture. A machine with no spec copy would SKIP them, which is the cost named
  in `generated_test.go`'s own doc.
- **`delete the guard in renderAll` was caught by an allocation difference**,
  which is the only form that claim can take: the absolute is not zero, because
  rendering is what the function does.
- **`if (MEMBER_ROLES.has(role))` → `if (false)` failed 32 of 67 mjs tests**,
  which is the signal that the walk-out rule is load-bearing for the two
  existing composites and not only for the new one.
- **One mutation compiled to nothing and was re-run**: an early attempt at
  "htmlout starts writing the flag" used helper names that do not exist, and a
  build failure is not a catch. Redone against the real `attrs` slice.

## Verification

Eight paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    wasm/verify/run.sh          replay + 17 mjs suites + BROWSER PASS (4 checks)
    ios/verify/run.sh           flex + stack solvers + 14 picker menus
                                + replay + view + app
    android/verify/run.sh       14 picker menus, on a JVM
    android ./gradlew compileDebugKotlin --offline --rerun-tasks
    GOOS=js GOARCH=wasm go build ./...

New files:

    aria/spec/spec.go                  the specification parser
    aria/spec/fixture.go               the scope, and the fixture renderer
    aria/spec/spec_test.go             the parser, on a miniature spec
    aria/gen/main.go                   the generator command
    aria/fetch.sh                      the one thing here that needs the network
    aria/verify/generated_test.go      the fixture, held to ARIA
    aria/verify/refusals_test.go       the six patterns core refuses, checkable
    core/debug_cost_test.go            what the three guards buy
    components/collapse_band_test.go   CollapseBand, against the default band
    mobile/verify/followsfocus_test.go the three targets that stay silent

Changed:

    aria/verify/doc.go                 "What this is not" -> "Where it comes from"
    aria/verify/testdata/aria.json     GENERATED; four facts corrected
    core/style.go                      AccessibilitySelectionFollowsFocus + merge
    core/style_props.go                the prop
    core/a11y_audit_test.go            points at the cost test next door
    components/grouping.go             CollapseBand
    components/grouped_list.go         two cross-references to it
    htmlout/orientation.go             the toolbar row has a keyboard now
    wasm/grmob-runtime.js              COMPOSITE_FOCUSABLE, focusableMembers,
                                       isComposite, compositeMembersOf,
                                       MEMBER_ROLES, compositeOf's two rules,
                                       selectCompositeMember, the data attribute
    wasm/verify/keynav_test.mjs        14 toolbar tests, 8 follows-focus tests
    wasm/verify/keynav_test.go         the two new tables, pinned to core
    wasm/verify/a11y_test.mjs          the toolbar test that asserted the opposite
    wasm/verify/browser.mjs            a fourth check; the rAF fix; a CDP timeout
    mobile/verify/gomobilestub_test.go the readings, the version pin, both
                                       refusals, and a test for the shapes
    ios/.../GrMobStyle.swift           the follows-focus note
    android/.../GrMobStyle.kt          the follows-focus note
    .gitignore                         the spec download

## Docs

`docs/platforms/wasm.md`: the toolbar member rule and why a nested composite is
two tab stops, selection-follows-focus and the premise that was wrong, the
fourth browser check, and the rAF finding. `docs/platforms/native.md`: where the
gobind facts come from, and both resolved refusals. `docs/components.md`:
`CollapseBand`, beside `GroupHeader`, with the division between them.
`docs/concepts/debug-mode.md`: what "zero-cost when off" actually costs, as a
table. `README.md`: the two ARIA commands and that nothing on a verification
path needs them. `ROADMAP.md`: six new entries.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here. Value is the payoff of doing it, not the effort: `high` means
something is being worked around today or a second consumer has arrived,
`medium` means it blocks one named thing, `low` means it is a gap nobody has
bumped into yet.

1. **(age 3 · value low) A collapsible band's tap target excludes its own
   padding and its badge.** The button fills the space between the band's
   insets and stops where the count begins, which is a consequence of keeping
   the count announceable. The fix is not obvious: moving the chrome onto the
   button would leave the badge without its trailing inset, and the band Row
   is the node `StickyHeader` has to sit on. `CollapseBand` does not change
   this — it is the control alone, so a caller placing it in their own row
   inherits the same question one level out.
2. **(age 3 · value low) A collapsed run is invisible to `OnEndReached`.**
   An infinite feed whose last group is shut has no rows near the bottom, so
   the edge sensor sits on the band and the next page never loads. Neither
   widget knows the two features are in tension. The honest fix is probably a
   footer that stays reachable, which `LoadMore` already is, plus a note; the
   interesting one is whether a shut trailing group should suppress the
   auto-load rather than starve it.
3. **(age 3 · value low) The gomobile stub's *doc comment* is still prose.**
   The declarations are pinned character-for-character and the header comment
   above them is a hand-written description of rules that live in
   `gobindSwiftTypes` and the two signature builders. It agrees today because
   it was written from them. This session made it *more* worth doing rather
   than less: the comment now also describes gobind's three result arms, which
   is a reading of a file that can move under it, where `gobindVersion` is the
   only thing that would say so.
4. **(age 3 · value medium) The `ControlBorder` retint is not visible on any
   screenshot.** Now two retints deep: `AmberTheme` adds a whole palette that
   no verification path has ever *looked* at. Four renderers emit its hexes and
   all four passes are arithmetic, type-checks or DOM shims. `browser.mjs` runs
   a real Chrome, so a screenshot is a `Page.captureScreenshot` away — what is
   still missing is what to compare it against, which is the whole question a
   palette poses.
5. **(age 3 · value low) `boundaryBackdrops` is a list of fills somebody
   remembered.** The census crosses the tone with `Background`, `Surface`,
   `Card`, `Input` and `TextArea`, named by hand, with `Camera` excluded by
   name. A new `ComponentDefaults` field carrying a `Background` — a `Sheet`, a
   `Popover` — is a backdrop a control can sit on and the census would not
   know. Reflecting over `ComponentDefaults` would close it, at the cost of
   needing the `Camera` exclusion to survive as data rather than as a line in a
   slice literal.
6. **(age 3 · value low) The wrapper test is a rule about a slice nobody
   returns.** `appendRows` appends into the caller's `[]core.PropsAndChildren`
   and returns it, so "can this knob be done to the result" is answered against
   a slice that also holds the container's own props. The opacity assertion
   measures the children it emitted, which is the right subject, but a real
   wrapper would have to find them among the props first.
7. **(age 2 · value medium) The overlay `Layout` is measured and has never been
   mounted.** `GrMobStackSolver` is checked hard, and `GrMobStackLayout` — the
   part that proposes sizes to subviews and reads the `LayoutValueKey` — is
   only type-checked. Three things it does are judgement calls a simulator
   would settle in a minute and no test can: measuring children with the
   incoming proposal rather than an unspecified one, *not* clamping the
   container to the proposal, and re-proposing `bounds.size` at placement.
8. **(age 2 · value low) `AmberTheme` is not used by any example app.** Three
   palettes ship and the tutorial's theme chapter still lists two by hand. The
   new theme's only readers are tests, so the one question a bundled theme
   exists to answer — does a real screen look right in it — has not been asked.
9. **(age 2 · value low) `wantWitnesses` is a hand-transcribed table that a
   fourth theme silently widens.** The census derives its theme list from
   `core.BundledThemes()`, so a new palette is asked every rule — and its
   answers land as a diff against a map written by hand. What is not stated is
   what a new row should look like when somebody is adding a theme rather than
   debugging one: "witnessed by nobody" and "witnessed by everybody" are both
   legal and mean opposite things.
10. **(age 1 · value medium) `grMobValue`'s three-way branch is still checked
    by `strings.Contains`.** The runner exists and the branch is out of its
    reach, because `grMobValue` is a `SemanticsPropertyReceiver` extension and
    the harness has no Compose. The move is the one `GrMobSelectMenu.kt` made:
    split the branch into a pure function in a file that imports nothing. What
    that needs first is a Go authority to compare against, and there is none —
    htmlout passes the three ARIA attributes straight through, which is a
    different mapping from Compose's three-way one.
11. **(age 1 · value low) A `Spacer`'s size prop clobbers its own Style.**
    `applySpacerSize` writes width, height and flex-shrink, and `renderNode`
    calls it *after* `createElement` — so a Spacer carrying a `Width` in its
    Style loses it to a prop that is not exempt from anything. Nothing in
    `core` builds such a node; a hand-assembled one reaches it.
12. **(age 1 · value low) A heading with no options under it is still
    inexpressible.** A `Group` is a string on an option, so a section that
    exists to say "nothing here yet" has no way to be written. The blocker is
    authoring rather than rendering: it needs a heading declared independently
    of an option, which means a second list and a matching problem the run-based
    reading was chosen to avoid.
13. **(age 1 · value low) `browser.mjs` still asks one page four questions.**
    Advanced rather than closed: it grew a fourth check and, more usefully, a
    fixed wait and a bounded round trip. What is still absent is a check for
    anything that is not the keyboard — whether a focus ring is visible on a
    themed control, whether an `aria-live` region announces, whether a
    `<select>`'s menu opens where the theme's frame says it should. Adding one
    is a function call.
14. **(age 0 · value medium) A toolbar containing a composite is two tab stops,
    and only a comment says so.** `focusableMembers` stops at a nested
    composite and does not take it as a member, so a `tablist` inside a
    `toolbar` keeps its own roving tabindex. Every control stays reachable,
    which is why this shape was chosen, and it is not what ARIA describes — the
    pattern makes the nested widget's *current* member the outer widget's
    member. Doing that means two composites writing tabindex on one element,
    which needs an owner rule the section does not have. Nothing here builds
    one; `keynav_test.mjs` asserts the current answer so the next reader finds
    the decision rather than the accident.
15. **(age 0 · value medium) `CONTROL_ROLES` is two roles, hand-written, and
    core can grow a third.** The runtime treats a `Box` as one of a toolbar's
    controls when it carries `role="button"` or `role="link"` *and* an OnTap.
    Those are exactly the two `core.Role` documents as "a tappable container",
    and `wasm/verify/keynav_test.go` pins the pair — but the pin is against a
    hand-written list, not against the property. A future `RoleCheckbox` or
    `RoleSwitch` would be a control on the same argument and would silently not
    be a toolbar member. What is missing is a name in `core` for "roles that
    make a container a control", which the vocabulary does not currently have.
16. **(age 0 · value low) The refusals table's `Shape` is prose again, with
    braces around it.** `Blocked` is derived from `core.Roles()` and checked,
    which is what makes the table a mechanism; `Shape` — "two dimensions", "a
    submenu with its own Escape" — is a sentence nothing can contradict. It is
    the field a reader acts on when a refusal's vocabulary half ends, and it is
    the one that could be wrong by then. There is no obvious authority for it,
    which is the honest reason it is not checked, and saying so is not the same
    as checking it.
17. **(age 0 · value low) Selection-follows-focus has no browser check.** It is
    covered thoroughly in `keynav_test.mjs`, where `focus()` is an assignment
    and a dispatched callback is an array push. The two facts a browser would
    add are that the *real* focus move fires it exactly once and that a
    `<button>` receiving both an arrow and its own click behaviour does not
    double-fire. `browser.mjs` already records dispatches in
    `window.__dispatched` and asserts nothing about them.
18. **(age 0 · value low) `aria/gen` is not on any verification path, and a
    stale download regenerates differently.** The conformance test compares the
    fixture against whatever HTML is on disk, so a checkout holding an ARIA 1.1
    copy would fail with a diff that looks like a fixture error rather than a
    stale-download one. The spec's own version string is in the document and
    nothing reads it.
19. **(age 0 · value low) `CollapseBand` is not used by any example.** Its only
    readers are its tests, which compare it against `GroupHeader` — a good test
    and not the question a helper for override authors exists to answer, which
    is whether a real custom band reads well built out of it.
20. **(age 0 · value low) The `data-grmob-selection-follows-focus` attribute is
    written on any node that asks.** `applyAccessibility` does not consult the
    composite tables, deliberately — knowing them there would put the tables in
    two places — so the flag on a `list` or a plain `Box` is an attribute nobody
    reads. That is the same nothing that happened before the field existed, and
    it is also a claim in the DOM that is not true of the element. The audit is
    the natural place to report it and has no rule for it.
