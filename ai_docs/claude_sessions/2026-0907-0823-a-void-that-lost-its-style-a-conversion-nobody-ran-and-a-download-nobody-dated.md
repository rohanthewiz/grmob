# Session: a void that lost its style, a conversion nobody ran, and a download nobody dated

Session: https://claude.ai/code/session_017k47JkhG1WBZR5nP29c7Ge
Date: 2026-09-07 (follows "a-slice-nobody-returns-a-layout-nobody-mounted-and-a-heading-nobody-could-write")

## Ask

"Do the oldest 5 Next list items", then mid-session: "once done, pick up the
next 3, then do session wrap."

| # | age·value | item |
|---|---|---|
| 1 | 3·low | A `Spacer`'s Style is honoured on one target of four |
| 2 | 3·medium | The `Layout`'s adapter is three lines nothing runs |
| 3 | 2·low | `aria/gen` is not on any verification path, and a stale download regenerates differently |
| 4 | 2·low | `CollapseBand` is not used by any example |
| 5 | 2·low | `data-grmob-selection-follows-focus` is written on any node that asks |
| 6 | 2·medium | A toolbar containing a composite is two tab stops, and only a comment says so |
| 7 | 2·medium | `CONTROL_ROLES` is two roles, hand-written, and core can grow a third |
| 8 | 2·low | The refusals table's `Shape` is prose again, with braces around it |

All eight closed. Four of them (5, 6, 7, 8) turned out to be the same move from
two directions: **a fact that lived only where nothing could ask it** — three of
them moved *into* Go so `core.AuditTree` and a Go census could read them, and
the fourth moved *out* of prose into a derivation from the ARIA fixture. Items
1 and 2 were the same shape one target over: a rule written where the compiler
is the only reader.

## Phase 1 — the one node type whose Style went missing (item 1)

A `Spacer`'s size arrives as a **prop**, not as a Style declaration, and that is
exactly what made it the node whose Style was dropped. Both native arms were one
expression built from the prop:

```swift
case "Spacer": Color.clear.frame(width: CGFloat(node.intProp("size")), …)
```
```kotlin
"Spacer" -> Spacer(Modifier.size(node.intProp("size").dp))
```

No `grMobBox`, no `boxModifier` — so a hand-assembled Spacer's `Background`,
`Margin`, `AccessibilityLabel` and `OnTap` were honoured in a browser and
invisible on a phone. `htmlout` had the same hole and closed it last session by
moving its branch into the shared assembly; the WASM runtime never had it.

Both arms now build the box, with the chassis **underneath** the author — and
the two languages get there by opposite mechanisms, which is the part worth
knowing before moving either line:

```
   SwiftUI   later in a chain is further OUT, and an outer frame wins
             -> chassis frame written BEFORE .grMobBox
             -> nil on an axis the Style claims, which keeps Color.clear
                flexible there so grMobBox's background fills the box

   Compose   constraints flow outside-in; an inner size() coerces itself
             into what it was handed
             -> boxModifier first, .size() after, and that is the whole of it
```

The per-axis `nil` is not tidiness. A fixed 10×10 clear inside a stated
200-wide frame paints a 10-point square in a 200-point hole; leaving the
claimed axis unconstrained is what makes the background fill.

Still diverging and unreachable from Go: a hand-assembled Spacer's **children**.
Both natives' spacers are leaves. Named rather than closed.

## Phase 2 — the adapter that was three honest lines (item 2)

Last session's own words: "Renderer.swift keeps the adapter and the `place()`
call, and that is the honest remaining gap." The gap was that a swapped axis
compiles, draws, and is invisible to
`TestTheSwiftStackLayoutDelegatesToTheSolver`, whose subject is a `Layout` that
computes a size *itself*.

The obstacle was never SwiftUI. It was `LayoutSubview` in particular — an opaque
proxy with no public initializer — and `ProposedViewSize` is an ordinary public
struct a check can construct. So both directions moved to
`GrMobStackBridge.swift`, which `ios/verify` compiles and runs:

```
   GrMobProposal(ProposedViewSize)     an offer, as the solver sees one
   .proposedViewSize                   the same offer, going back
```

A separate file rather than an extension in `GrMobStack.swift`, because that
file's claim about itself — CoreGraphics and nothing else — is what lets it be
linked into a plain command-line binary, and `import SwiftUI` would have made
its own doc comment false.

The two stakes pull opposite ways and both are asserted: an unspecified offer
arriving as a number sizes every layer against a box nobody proposed; a stated
one arriving as `nil` makes every greedy background report its intrinsic size
and stop covering the stack. Zero is checked too — a real offer, not an absence.

`mobile/verify`'s delegation pin gained the other half: `ProposedViewSize(width:`
and `GrMobProposal(width:` are now banned inside both the `Layout` and its
adapter, with the three call sites asserted positively so a cut that read
nothing cannot satisfy the bans.

## Phase 3 — a download nobody had dated (item 3)

The failure is specific and nasty. W3C keeps every ARIA revision at its own URL
forever, the download is not committed, and 1.1 and 1.2 are the same ReSpec
output with different cells. So a stale copy:

- parses cleanly to ~94 roles
- passes every floor (`role count`, the orientation sentence, the markup)
- regenerates a fixture differing on **exactly the four facts 1.2 changed**

and the conformance test then prints "aria.json is not what aria/gen would
write", naming radiogroup's orientation as the first difference — a true
statement about 1.1, offered as a transcription error in a file nobody touched.

Every other signal is identical between the two revisions, because the *format*
did not change. Only the content did. The one thing that differs is the
document's own title heading, and nothing read it.

`spec.Version` is the constant now, `SpecVersion(html)` reads the heading, and
`Parse` refuses any other edition **before** believing anything else about the
document. Three consequences:

- `LocalPath` derives its filename from `Version`, so a bump moves the
  download and the guard together.
- `aria/fetch.sh`'s URL is the third statement and cannot read a Go constant,
  so a test holds the script to it.
- `aria/verify`'s conformance test **skips** with a sentence about the fetch
  rather than failing with a diff — the same stance it already takes toward a
  missing copy, for the same reason: a checkout is not broken because the copy
  beside it is old.

The trap the derivation had to avoid: a current 1.2 document links to
`wai-aria-1.1` fifteen times in its own change log, so anything scanning the
body reads 1.1 out of a perfectly good file. The heading is the only statement
of the edition, and that is asserted.

## Phase 4 — a widget with no example (item 4)

`CollapseBand`'s only readers were its own tests, and its second field
(`ControlStyle`) exists to answer a question that only arises when somebody
assembles a *real* custom band — so the absence was a slightly bigger hole than
an unused widget usually is.

Lesson 4.6 assembles one, and every band starts **shut**, which is the half a
`Header` override does not own:

```
   Collapse reaches PAST the override  ->  the run is withheld by the widget
   Collapse stops AT the override      ->  the control is the caller's
                                           (a band the widget also built
                                            would be a second control)
```

The demo derives the shut set from the fixture rather than writing four month
keys, so a month added to the archive is shut too rather than being the one run
that opens for no stated reason.

The badge sits outside the button, which is the other half: a reader does not
descend into a control, so a count put inside would stop being announced and
would join the button's name.

## Phase 5 — a keyboard contract on a widget with no keyboard (item 5)

The runtime writes `data-grmob-selection-follows-focus` for any node that asks,
and that is deliberate: consulting the composite tables where attributes are
written would put those tables in two places. The cost is a claim in the DOM
that nothing reads, on a `list` or on a `Box` whose role was never set.

The audit could not ask the runtime — the runtime is JavaScript — so the list
moved to Go as `core.KeyboardComposites()`, and `wasm/verify` holds it to the
**union** of `COMPOSITE_MEMBERS`' keys and `COMPOSITE_FOCUSABLE`. Not either
half: which of the two tables a role lands in is the runtime's business (does
ARIA name its members) and core has no opinion, so imposing one would be
inventing a fact to check.

`ConcernInertFollowsFocus` is the finding. The empty role is the case worth
reporting most — the strip looks right, announces as a plain box, and the flag
is the only evidence anyone meant a composite.

A toolbar counts as a reader and is **not** reported, which is not an
endorsement: ARIA recommends selection-follows-focus for tabs and says nothing
about toolbars, but this check's subject is only whether anything reads the
flag. Reporting a role that does read it would be a second, weaker claim
wearing this one's name.

## Phase 6 — two tab stops, told to the person who built them (item 6)

The divergence was already asserted (`keynav_test.mjs`) and already documented
(`docs/platforms/wasm.md`). What it was not was reachable from Go, where the
tree is written.

Both member walks stop at a nested composite, for different reasons that come
to the same outcome — a listbox's stops at a nested listbox because the two
would pool their options; a toolbar's stops at any composite because nothing
names its members. Either way the inner container keeps its own roving
tabindex:

```
   Tab  ->  [ All ] ( Sermons | Articles ) [ More ]     the toolbar's stop
   Tab  ->          (   ^ the strip's own stop   )      the tablist's
```

ARIA describes one, by making the inner widget's *current* member the outer
widget's member — which means two widgets writing tabindex onto one element and
needs a rule for when they disagree. There is none, and inventing one silently
is worse than the divergence.

So `ConcernNestedComposite` reports it, in Go, in debug mode. The rule is
uniform over every ordered pair rather than special-cased to the toolbar,
because in every nesting the inner composite keeps its own stop — which also
means it needs no second Go table and does not contradict what phase 5 just
wrote about core having no opinion on the membership split.

## Phase 7 — a list that a new role would have walked past (item 7)

`CONTROL_ROLES` was three copies of one fact: two bare strings in the runtime, a
pair of constants in `wasm/verify`, and a table in the docs. The pin held all
three together and none of them was the fact.

What none of them could catch is a role that should have been added. A future
`RoleCheckbox` is a tappable container by exactly the argument `core.Role` makes
for `RoleButton` — a Box the renderers draw as scenery, which a reader announces
as text until the role says otherwise — and it would compile, ship, and leave a
toolbar of checkbox rows with one tab stop and nothing to arrow to.

`core.TappableContainerRoles()` is the fact now, and `role_control_test.go` is
what makes it a *property*: every role in `Roles()` is either in the list or has
a stated reason for not being. Writing the twenty-three reasons produced three
distinctions nobody had had to state before:

```
   structure     arrangement, not operation
   containers    they own a keyboard, they are not a control
   members       their tab stop belongs to their container   <- the sharp one
   read-only     ARIA's own progressbar/slider division
```

The members line is the one that matters: `option` and `tab` are interactive and
deliberately excluded, because taking one as a toolbar's control hands the same
element two owners writing tabindex — the same collision `ConcernNestedComposite`
reports one level up.

## Phase 8 — half a sentence with an authority (item 8)

`Blocked` was derived from `core.Roles()` and checked. `Shape` sat in the same
struct looking the same and was a sentence nothing could contradict — a reader
had no way to tell which of the two fields was load-bearing.

There is no authority for most of `Shape` and there is one for a piece of it.
ARIA's Required Owned Elements row says, mechanically, whether a pattern's
members are the container's own or somebody else's:

```
   listbox  owns option                        a descent for one role
   tablist  owns tab                           the same descent
   grid     owns row, and a row owns gridcell  two axes  <- what is missing
```

That is exactly the difference between the walk this runtime does and the one it
does not have, so "two dimensions" stopped being prose. `Nesting` is declared
and derived, the same mechanism `Blocked` uses; the reachability half is checked
too, one hop deep on purpose (two hops would let any role reach any other
through `row` and `group` and answer yes for everything).

What is left in `Shape` — a submenu with its own Escape, a tree's expand-in-place
arrows — is about ARIA's *authoring practices* rather than its role definitions,
which this repository does not fetch. Saying which field is checked and which is
not is the deliverable; pretending otherwise was the finding.

## The break-tests

**36 caught, 1 miss by design, 0 restore failures** (after the harness was
fixed — see below).

    item 1   the box, the ordering, the per-axis nil     5
    item 2   both conversion directions, three call sites 7
    item 3   the guard, the constant, the regex, the script 5
    item 4   the Collapse wiring, the toggle, the seed    4
    item 5   the core list both ways, the walk, the arm   4
    item 6   the call, the report, the ancestor thread    3
    item 7   the list both ways, the runtime, a NEW ROLE  5
    item 8   Nesting flipped on three rows                3

Three things are worth recording.

- **The harness's restore was itself a bug.** Restoring by reversing the edit
  lands on the *wrong occurrence* when the replacement text also appears
  elsewhere in the file, which silently corrupted `Renderer.swift` — a
  `spacerExtent` frame grafted onto `GrMobRow`'s wrap branch. Caught by the
  hash check, repaired by hand, and the harness now snapshots the whole file
  and restores from the copy. A reverse-edit restore is not a restore.
- **One assertion could not fail, and a mutation said so.** The banded demo's
  test checked `Padding.Left == 16` on the control — and a disclosure's control
  is a `core.Row`, which arrives carrying the theme's own 16. Deleting the
  demo's `PaddingLeft(16)` passed. The check moved to the trailing inset and the
  vertical one, which are the demo's own numbers and are not the theme's.
- **A missed mutation exposed a weak test rather than a weak rule.** Making
  `checkNestedComposite` forget its ancestor at every non-composite passed,
  because the fixture's intermediate containers had no `Style` and the walk only
  reaches the check for nodes that do. The fixture now uses styled
  intermediates, which is the ordinary case — nearly every real container has a
  Style — and the mutation is caught.

**The one miss is a mutation of a guard, not of its subject.** Disabling the
version check in `aria/verify`'s conformance test cannot be caught, because
there is no stale download on disk to trigger it. The load-bearing half —
`spec.Parse` refusing a 1.1 document — is exercised directly and is caught.

## Verification

Eight paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    wasm/verify/run.sh          replay + 17 mjs suites + BROWSER PASS
    ios/verify/run.sh           flex + stack solver, its three Layout
                                decisions AND its proposal conversion
                                + 15 picker menus + replay + view + app
    android/verify/run.sh       15 picker menus and 22 value ranges, on a JVM
    android ./gradlew compileDebugKotlin --offline
    GOOS=js GOARCH=wasm go build ./...

New files:

    ios/GrMob/Runtime/GrMobStackBridge.swift  the two-way size conversion
    mobile/verify/spacer_test.go              the box, and the chassis order
    core/role_control_test.go                 the tappable-container census

Changed:

    ios/.../Renderer.swift             GrMobSpacer, spacerExtent, the adapter
    ios/.../GrMobStack.swift           where the line is now
    ios/verify/{run.sh,main.swift,stack.swift}  the bridge and its checks
    android/.../Renderer.kt            the Spacer arm, through boxModifier
    mobile/verify/stackalign_test.go   the conversion bans, swiftTypeBody
    aria/spec/{spec.go,fixture.go}     Version, SpecVersion, the refusal
    aria/spec/spec_test.go             the stale copy, the heading, fetch.sh
    aria/verify/generated_test.go      skip on a stale edition
    aria/verify/refusals_test.go       Nesting, declared and derived
    aria/{fetch.sh,gen/main.go}        the pin, and the third way to be wrong
    core/role.go                       KeyboardComposites, TappableContainerRoles
    core/a11y_audit.go                 two findings and the ancestor thread
    core/style.go                      what makes a follows-focus flag inert
    examples/tutorial/chapter4.go      the banded demo and its Header override
    examples/tutorial/chapter4_test.go the assembly, and findBand
    wasm/grmob-runtime.js              two cross-references to core
    wasm/verify/keynav_test.go         the composites pin, controlRoles()

## Docs

`docs/concepts/debug-mode.md`: two rows in the concerns table.
`docs/platforms/wasm.md`: why the flag is written totally and where it is
reported, the nested-composite divergence and its owner-rule cost, and
`CONTROL_ROLES` coming from core. `docs/platforms/native.md`: a new section on a
Spacer's own Style with the two constraint models side by side, and where the
`Layout`'s vocabulary went. `docs/components.md`: the tutorial builds a
`CollapseBand` now. `docs/tutorial-interactive.md`: chapter 4's row.
`ROADMAP.md`: eight new entries.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here. Value is the payoff of doing it, not the effort: `high` means
something is being worked around today or a second consumer has arrived,
`medium` means it blocks one named thing, `low` means it is a gap nobody has
bumped into yet.

1. **(age 3 · value low) `aria/gen` still runs on no verification path.** The
   version guard closed the half that produced a misleading failure; the command
   itself — `repoRoot`, the error it prints when the download is absent, the
   write — is exercised by nothing. `go test ./...` reaches `spec.Parse` and
   `Fixture.Render` through the conformance test and never reaches `main.go`.
   A test that runs `run()` against a temp root would cost little; the argument
   against is that the only thing left in it is filesystem plumbing.
2. **(age 3 · value low) The two exception tables for themes are empty.**
   `quietThemes` and `universalThemes` cost a sentence each and neither has an
   entry, so the arms that read them have never run against a real case. The
   backdrop exclusions had two entries on their first day.
3. **(age 3 · value low) The sticky check writes `core.StickyHeader()`'s three
   declarations by hand.** `browser.mjs` mounts JSON trees, so the fixture
   states `Position`, `Top` and `ZIndex` itself rather than asking Go. A change
   to the Go prop that dropped one would leave the browser check green.
   `wasm/verify` already generates tables from Go for the tag, stack-axis and
   palette pins; this one could join them.
4. **(age 3 · value low) `ValueRange.Progress` has no Go consumer.**
   `components.ProgressBar` always states all three numbers, so it is
   determinate by construction and never asks. Its only readers are a test and a
   transliteration.
5. **(age 3 · value medium) `internal/valuefixture` is compared on one
   platform.** `internal/menufixture` reaches three harnesses; this one reaches
   the JVM alone. The web half is not as settled as that sentence: a browser
   applying ARIA's `0..100` default is a claim nothing here has watched, and
   `browser.mjs` can now read a rendered node's computed `aria-valuenow`.
6. **(age 2 · value low) A `Header` override still cannot be told the edge was
   withheld.** `GroupedList` withholds `OnEndReached` when the trailing run is
   shut, and the reader's only cue is that scrolling stops loading. A feed whose
   `Footer` is a bare `LoadMore` works; one that hides the footer when `HasMore`
   is false looks finished.
7. **(age 2 · value low) The palette check paints swatches, not widgets.** The
   pairs measured are `(ControlBorder, fill)` drawn by `browser.mjs` itself.
   What a real `Input` or a quiet `Chip` puts on screen goes through
   `components`, which the browser pass does not run.
8. **(age 2 · value low) `internal/palette` reflects and `core` cannot tell it
   not to.** A `ComponentDefaults` field whose `Background` is a fill no control
   is drawn on has to be excluded by name in a package the theme author does not
   edit. A struct tag would put the fact next to the field.
9. **(age 2 · value low) The two-result gobind arm is still transcribed
   nowhere.** `swiftResult` names which of the three arms a refusal is about and
   the stub's comment is checked against that naming, so the *report* is right.
   What is missing is a bridge function of that shape to make it worth writing.
10. **(age 1 · value medium) The band's insets are on the control and no
    renderer but the web has been asked about it.** Padding moved from a parent
    to a stretched child, which is exactly identical in CSS flex and is an
    *assumption* everywhere else. `ios/verify`'s stack solver could now settle
    the SwiftUI half — it has a proposal vocabulary, a recording fake, and as of
    this session a real `ProposedViewSize` conversion it can construct offers
    with — and nothing asks it.
11. **(age 0 · value medium) A hand-assembled `Spacer`'s children are dropped on
    both natives.** Closing item 1 made three targets agree about a Spacer's
    Style and left one divergence standing: both DOM renderers emit a Spacer's
    children like any other element's, and a Compose `Spacer` and a
    `Color.clear` are leaves. `core.Spacer(n)` builds none, which is the
    argument `core.Modal` sits under — and it is the same argument that was made
    about the Style right up until it was closed.
12. **(age 0 · value medium) The nested-composite finding is reported in Go and
    asserted in JavaScript, and the two are not held together.**
    `core.AuditTree` says a composite inside a composite is two tab stops;
    `keynav_test.mjs` demonstrates it. Nothing checks that the Go rule describes
    the JavaScript one — a runtime that started pooling members would leave the
    audit reporting a divergence that had been fixed. `core.KeyboardComposites()`
    is pinned to the runtime's tables; the *walk's stopping rule* is not.
13. **(age 0 · value low) `swiftTypeBody`'s cut is "the first closing brace in
    column one".** True of every Swift type in `Renderer.swift` and not true of
    Swift — a type declared inside a `#if` block, or one whose body holds a
    string literal containing `\n}\n`, would cut short. `mobile/verify` already
    fails loudly on an anchor it cannot find; this failure mode is a cut that
    reads *less* than it should and still finds what it was looking for.
14. **(age 0 · value low) `notTappable`'s reasons are prose by construction.**
    `role_control_test.go` checks that an answer exists, not that it is right,
    and says so. The four kinds it sorts roles into (structure, container,
    member, read-only) are almost a derivable property — `KeyboardComposites()`
    already states one of them, and the ARIA fixture's `requiredOwned` states
    another — so the census could grow a `kind` column derived the way the
    refusals table's `Nesting` now is.
15. **(age 0 · value low) The stale-download skip is a guard nothing exercises.**
    `aria/verify`'s conformance test refuses to run against a wrong-edition copy,
    and the only way to reach that branch is to have one on disk. Its subject —
    `spec.Parse` refusing 1.1 — is covered; the skip itself is the arm that has
    never run, which is the same shape as the two empty theme tables above.
