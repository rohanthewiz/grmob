# Session: a slice nobody returns, a layout nobody mounted, and a heading nobody could write

Session: https://claude.ai/code/session_017k47JkhG1WBZR5nP29c7Ge
Date: 2026-09-07 (follows "a-strip-in-the-middle-of-a-band-and-a-palette-nobody-had-looked-at")

## Ask

"Do the oldest 5 items in the Next list", then mid-session: "once done, take the
next 3 items."

| # | age·value | item |
|---|---|---|
| 1 | 4·low | The wrapper test is a rule about a slice nobody returns |
| 2 | 3·medium | The overlay `Layout` is measured and has never been mounted |
| 3 | 3·low | `AmberTheme` is not used by any example app |
| 4 | 3·low | `wantWitnesses` is a hand-transcribed table a fourth theme widens |
| 5 | 2·medium | `grMobValue`'s three-way branch is checked by `strings.Contains` |
| 6 | 2·low | A `Spacer`'s size prop clobbers its own Style |
| 7 | 2·low | A heading with no options under it is inexpressible |
| 8 | 2·low | `browser.mjs` still asks one page four questions |

All eight closed. Three of them turned out to be the same shape — a rule that
exists only where nothing can run it (2, 5, 8) — and two more were the same
shape as each other from opposite ends: a type default outranking an author
(6) and an author's list outranking a derivation (3).

## Phase 1 — the slice a wrapper is actually handed (item 1)

`TestAppendRowsOutputIsOpaqueToItsCaller` was answering the admission rule
against `appendRows(ctx, nil, …)`. Both real callers pass a slice that already
holds the container's props, and `appendRows` appends into it and returns it —
so the thing a wrapper would receive is props first, children after.

The fixture now carries the prefix `GroupedList.Render` builds, and the first
leg is the one the old shape hid:

```
Padding(0)  Gap(0)  OnEndReached(fn)  BackgroundColor(#FFF)   ← styleFunc / behaviorFunc
Keyed(band) Keyed(row) Keyed(sep) …                            ← ComponentFunc
        every one of them reflect.Kind == Func
```

A core prop is a closure too — `styleFunc` is a `func(*Style)`, `behaviorFunc`
a `func(*Context, *Node)` — so "it is a func" identifies nothing and a wrapper
reaching for `reflect.Kind` first would take the container's padding for a row.
The only sound separator is the `core.View` assertion, and it is now asserted in
both directions before the ten children are examined.

## Phase 2 — the overlay layout's three decisions (item 2)

The item named them: measuring children with the *incoming* proposal, refusing
to clamp the container to it, re-proposing `bounds.size` at placement. All three
sat inside `GrMobStackLayout`'s `Layout` conformance with a type-check as their
only reader, and the reason was real — a `LayoutSubview` is an opaque proxy with
no public initializer, so nothing off-device can construct one.

But the layout only asks a subview two things. `GrMobStackLayer` is those two,
`GrMobProposal` is `ProposedViewSize` with the framework taken out, and the
decisions moved into `GrMobStack.swift` where `ios/verify` runs them against a
recording fake:

```
    sizing      containerSize(layers:proposing:)   every layer gets the incoming
                                                   offer; result is the content
    placement   placements(layers:in:)             -> [origin, proposal], the
                                                   offer being bounds.size
```

Restating `ProposedViewSize` is the cost. What it buys is that a greedy layer
can be *written* — one that returns whatever it is offered — so "a background
stating `maxWidth: .infinity` still makes the stack fill" is an assertion rather
than a sentence in a doc comment.

`Renderer.swift` keeps the adapter and the `place()` call, and that is the
honest remaining gap: see the break-tests.

## Phase 3 — a palette the tutorial had never heard of (item 3)

Both theme lessons wrote `[]*core.Theme{DefaultTheme, MaterialTheme}` by hand.
`AmberTheme` shipped two sessions ago and the chapter whose entire subject is
theming went on offering two.

`bundledThemes()` derives the list from `core.BundledThemes()` with one imposed
rule — the fallback leads, the rest alphabetically — because map iteration is
randomised and a picker that reorders itself between renders is not a picker.
Labels are derived too: a hand-written label list is a fourth palette named `""`
on the day somebody adds one.

**A hand-written list in a tutorial is worse than one in a test.** The gap is
not a missed assertion; it is a palette the reader is never told exists. The new
test asks `core.BundledThemes()` and looks for an operable `Button` per palette
in *both* lessons, and the two existing lesson tests now drive Amber through as
well, because a picker that renders a button and wires it to nothing looks
identical in a tree dump to one that works.

## Phase 4 — the census, read down its columns (item 4)

Two findings, and the first was sitting in the file already.

**The prose was wrong and nothing could tell.** `wantWitnesses`' own doc said
two rules rested on the fixture alone; the table three lines below it listed
`AmberTheme` for both. True when written, wrong the moment a palette was added.
So a row now states what its witnesses *amount to* — `restsOnBundled`,
`restsOnFixture`, `restsOnNothing` — the test derives the same value from the
list and the bundled flag, and a row that does not rest on a shipped palette
owes a sentence. That is `internal/palette`'s backdrop-exclusion road, which the
item itself pointed at.

**The item's real subject was the other axis.** "Witnessed by nobody" and
"witnessed by everybody" are both legal and mean opposite things — and they are
facts about a *theme*, not a rule. A retint moves a row; a new palette adds a
column, and both of its extremes arrive looking like an ordinary row diff:

```
              declaration  pole flip  OnLight×4        a new palette in
                                                       no row      → adds no
                                                                     evidence
   Amber           ✓           ·      Primary Warning  every row   → every
   Default         ·           ·      Error Success                  fixture-only
   Material        ·           ·      Warning                        row just
   midTonePrimary  ✓           ✓      Primary Error                  became bundled
```

`TestEveryKnownThemeLandsSomewhereStated` reads the matrix that way and makes
each extreme cost a named entry with a reason. Both exception tables are empty
today, and they are held to the theme list in the other direction so an entry
naming a palette nobody measures is an argument for nothing.

The `knownTheme` struct is what makes any of it derivable: `bundled` comes from
`core.BundledThemes()`, so "is this the fixture" stopped being something a
reader had to know from the name.

## Phase 5 — a reading with no authority (item 5)

The item said the blocker first: the move is `GrMobSelectMenu.kt`'s — a pure
function in a file that imports nothing — and "what that needs first is a Go
authority to compare against, and there is none."

The authority was already there in prose. `core.ValueRange`'s own fields say
ARIA's implicit `0..100` and say that a missing position is an indeterminate
bar; the Kotlin was a transliteration of those sentences. `ValueRange.Progress`
makes them executable.

**It is four readings, not three.** The branch always had a fourth outcome and
it was spelled as an absence: `if (max > min)`, with the else doing nothing.

```
    Now unstated, a bound stated   indeterminate
    Now unstated, nothing else     unstated          }  both assign nothing,
    Now stated, max <= min         empty-range       }  for opposite reasons
    Now stated, max > min          determinate
```

`ProgressBarRangeInfo` throws on an empty range, so that case has to be told
apart from "nothing was claimed" *before* the property is assigned. Both silent
readings are named arms in the `when` now rather than an `else`, which is the
whole of what the fourth name buys.

The parse went into the pure file too (`grMobProgressNumber`), so the comparison
covers it: `"half"`, `""` and `"NaN"` are all checked to be no position rather
than assumed to be. Both parsers accept `NaN` — Go's `ParseFloat` and Kotlin's
`toFloatOrNull` — and no range property on any platform can hold one, so both
sides refuse the non-finite spellings and `internal/valuefixture` has the cases.

The web exporters do not call `Progress`, and saying why is the honest part:
they hand three attributes to a browser, which applies the same rules itself.
Its consumers are the transliteration and the harness between them.

## Phase 6 — the one place a type default beat an author (item 6)

The item said a `Spacer`'s size prop clobbers its own Style. It does, and the
export was worse: `htmlout` returned before the shared attribute assembly, so a
`Spacer`'s Style was dropped *whole*.

The precedent was three files over. `modalChassis` states a node type's fixed
look **ahead of** the author's declarations, and the runtime says the same thing
with `out.x = out.x || …` inside `styleFromGrMob`. `core.ModalNode` carries no
Style either; the `||` is there for the hand-assembled node.

The Spacer could not join Modal's chassis for one reason — `styleFromGrMob`
never sees a prop, and the size is one — so `applyStyle` runs the chassis
immediately after the style assignment instead.

**The first fix was wrong and the resize test caught it.** Reading the live
property back ("still empty" = "the author said nothing") works exactly once,
right after the style pass. A size change arrives as an update-props patch with
no Style in it, and the read then finds the chassis's own last write and defers
to it — so a resized Spacer kept its first gap. Which of the three the author
claimed is recorded at style time instead.

The same move closed a latent half nobody had reported: `styleFromGrMob` is
total, so an update-style patch on a Spacer used to clear the gap with nothing
to put it back. That is the failure Modal's chassis comment already warns about.

Moving `htmlout`'s branch into the assembly gave a Spacer back four things the
other DOM renderer had been giving the same node all along — its accessibility
attributes, its callback IDs, its children, and its own style. On every tree
`core` builds the output is byte-for-byte what the early return emitted.

## Phase 7 — a heading with nothing under it (item 7)

Not implemented, and that is the answer rather than a deferral.

`Group` is a field on an *option*, so a section with no options has nothing to
declare it. Declaring one independently means a second list and a matching
problem — which headings are in use, what an unmatched heading does, what an
option naming a missing heading does — and the run-based reading was chosen
precisely to have none.

What nobody had asked for was an empty section. What they had asked for was
"this category is empty, say so", and that is expressible today:

```go
{Group: "Archive", Label: "Nothing archived yet", Disabled: true}
```

It is *better* than an empty section on all four targets, not merely available:
an `<optgroup>` with no `<option>`, a SwiftUI `Section` with no `Button` and a
Compose heading with no rows are each a label a reader announces and a pointer
cannot reach, and none of them says why.

So the limit is stated where a caller reads it (`SelectOption.Group`,
`docs/concepts/forms.md`), pinned as a property rather than a fact about a
fixture — `TestAnEmptySectionIsUnreachable` sweeps the shapes somebody would
reach for, including the sentinel `{"group": "Archive"}` with nothing else on it
— and the recommended shape is in `internal/menufixture`, so all four picker
menus are held to it.

## Phase 8 — the first layout question (item 8)

Five checks in, everything `browser.mjs` asked could have been asked of a page
with no geometry: four about the keyboard, one about a colour.

`core.StickyHeader()` writes three declarations and claims a `List` child stays
put while the rows scroll under it. `dom.mjs` has no layout at all, so a suite
there can say the three landed on the element and stop — and "the property is
written" is exactly what stays true when the box around it defeats the pin: an
ancestor with `overflow` other than `visible`, a flex item shrunk to its
container, a containing block that is not the scroller.

The check mounts a scroller, scrolls it 120px, and asks twice:

```
   before          after
   ┌──────────┐    ┌──────────┐   band.y - port.y == 0     (the rect)
   │ ▓▓ band  │    │ ▓▓ band  │   row moved by 120
   │ ░░ row 0 │    │ ░░ row 2 │   pixel at the band's middle == #123456
   │ ░░ row 1 │    │ ░░ row 3 │                              (the paint)
```

The pixel is not a duplicate of the rect. A rect is what the browser laid out;
the sample is what it put on the screen there, and the two differ whenever
something is drawn over the band. A control step comes first, as the `ArrowDown`
check's does: a scroller with nothing to scroll passes every assertion by never
moving anything, and the `FlexShrink: 0` on the fixture's list is there because
without it the flex column compresses its child instead of overflowing.

## The break-tests

**40 caught, 1 genuine miss, 0 restore failures.** Same discipline as the last
nine sessions: snapshot by hash, one exact-string mutation, run the suite,
restore, assert the restore by hashing, and an anchor that does not appear
exactly once is a `SKIP` rather than a silent no-op.

    item 1   the returned slice                     2
    item 2   the layer seam and the delegation pin   6
    item 3   the derived theme list                  3
    item 4   the rests column and the theme axis     6
    item 5   Progress, GrMobProgress.kt, the fixture 12
    item 6   the two chassis                         5
    item 7   the empty-section property              2
    item 8   the pin and the paint                   3

Three mutations in the first round were malformed — they mutated an assertion
rather than its subject, or were no-ops — which is its own finding: a mutation
that edits the check is guaranteed to "miss" and says nothing. Re-anchored on
the subject, all three were caught.

**The one real miss is a boundary, and it is now written down.** Changing the
proposal `GrMobStackLayout` passes *into* the solver — `GrMobProposal(width:
nil, height: nil)` instead of the incoming one — is invisible to every harness:
`Renderer.swift` is type-checked and never run. That produced
`TestTheSwiftStackLayoutDelegatesToTheSolver`, a source-text pin in
`mobile/verify` that catches the shape that would put the decisions back out of
reach (a Layout computing a size or an origin itself), and it catches both
reinlining mutations. It does not catch a conversion that is subtly wrong, which
is the honest limit and a Next item.

Worth recording what caught what.

- **The Spacer resize test caught the first draft of its own fix.** Reading the
  live property back is correct on the style path and wrong on the prop path,
  and the existing patch-path test — written two sessions ago for a different
  bug — is what said so.
- **The `rests` column caught the file's own prose**, which had been wrong for a
  release. Nothing else could have: the table and the paragraph above it were
  two independent statements and only one of them was executable.
- **`TestEveryKnownThemeLandsSomewhereStated` needed a mutation of the rules,
  not of the guard.** Disabling the guard is not a break-test; making
  `onLightRule` unwitnessable is, and it drops `DefaultTheme` and
  `MaterialTheme` to witnessing nothing.
- **The palette-swatch lesson repeated itself.** A mutation inside
  `GrMobProgress.kt` that changes both the reading and nothing else is caught
  because Go holds the expectation; a mutation to the *fixture* would move both
  sides and pass, which is why the fixture has its own coverage tests
  (every reading reached, the clamp and the defaulting each exercised by
  something).

## Verification

Eight paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    wasm/verify/run.sh          replay + 17 mjs suites + BROWSER PASS
                                (4 keyboard checks + 22 palette swatches
                                 + a sticky band)
    ios/verify/run.sh           flex + stack solver and its three Layout
                                decisions + 15 picker menus + replay
                                + view + app
    android/verify/run.sh       15 picker menus and 22 value ranges, on a JVM
    android ./gradlew compileDebugKotlin --offline
    GOOS=js GOARCH=wasm go build ./...

New files:

    internal/valuefixture/               the shared table of wire ranges
    android/.../runtime/GrMobProgress.kt the numeric reading, importing nothing

Changed:

    components/rows_spec_test.go       the opacity test, on the real slice
    components/palette_witness_test.go rests, the theme axis, knownTheme
    core/value.go                      ValueRange.Progress and its four readings
    core/value_test.go                 the authority, pinned by hand
    core/input.go                      why an empty section is unwritable
    core/select_menu_test.go           that property, and the placeholder shape
    internal/menufixture               the placeholder case
    examples/tutorial/chapter7.go      bundledThemes(), themeLabels()
    examples/tutorial/chapter7_test.go every palette offered, and Amber driven
    htmlout/export.go                  spacerChassis, out of the early return
    htmlout/export_test.go             the author wins; the four things restored
    ios/.../GrMobStack.swift           GrMobProposal, GrMobStackLayer, the two
                                       layer-level entry points
    ios/.../Renderer.swift             the adapter, and two two-line methods
    ios/verify/stack.swift             the recording fake and the three rules
    android/.../GrMobStyle.kt          grMobValue as a when over four readings
    android/verify/{gen.go,Harness.kt,run.sh}  the second table and its runner
    mobile/verify/{value,stackalign,select}_test.go  the new pins
    wasm/grmob-runtime.js              applySpacerChassis
    wasm/verify/{runtime,totality}_test.mjs  the chassis, and Spacer in the sweep
    wasm/verify/browser.mjs            check 6

## Docs

`docs/platforms/wasm.md`: the Spacer chassis beside Modal's, and a new section
for the sticky check. `docs/platforms/native.md`: where the line between
`GrMobStack.swift` and the `Layout` is now, and the value reading's authority.
`docs/concepts/forms.md`: the empty heading, and what to write instead.
`docs/concepts/styling-and-theming.md`: the `rests` column and the census's
second axis. `docs/tutorial-interactive.md`: the switcher is over every bundled
palette. `ROADMAP.md`: eight new entries.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here. Value is the payoff of doing it, not the effort: `high` means
something is being worked around today or a second consumer has arrived,
`medium` means it blocks one named thing, `low` means it is a gap nobody has
bumped into yet.

1. **(age 3 · value low) A `Spacer`'s Style is honoured on one target of four.**
   Fixing the ordering made the two DOM renderers agree with each other and with
   `modalChassis`, and it widened the gap with the natives: `Spacer(Modifier
   .size(n.dp))` and `Color.clear.frame(width:height:)` both ignore the node's
   Style completely, so a hand-assembled Spacer carrying a `Background` is
   coloured in a browser and invisible on a phone. Closing it is one modifier on
   each renderer; the argument against is that `core.Spacer(n)` cannot produce
   such a node at all, which is also the argument `Modal` sits under.
2. **(age 3 · value medium) The `Layout`'s adapter is three lines nothing
   runs.** `GrMobStackSubview` converts between `ProposedViewSize` and
   `GrMobProposal` in both directions, and a mutation inside it — passing an
   unspecified proposal to `containerSize` — is caught by nothing.
   `TestTheSwiftStackLayoutDelegatesToTheSolver` catches a reinlining, which is
   the shape that matters and not this one. A simulator settles it; so would
   moving the conversion itself behind the seam, which is a smaller change than
   it sounds.
3. **(age 2 · value low) `aria/gen` is not on any verification path, and a
   stale download regenerates differently.** The conformance test compares the
   fixture against whatever HTML is on disk, so a checkout holding an ARIA 1.1
   copy would fail with a diff that looks like a fixture error rather than a
   stale-download one. The spec's own version string is in the document and
   nothing reads it.
4. **(age 2 · value low) `CollapseBand` is not used by any example.** Its only
   readers are its tests. It has a second field (`ControlStyle`) whose whole
   justification is how a real custom band is assembled, which makes the absence
   of one a slightly bigger hole than it was.
5. **(age 2 · value low) The `data-grmob-selection-follows-focus` attribute is
   written on any node that asks.** `applyAccessibility` does not consult the
   composite tables, deliberately — knowing them there would put the tables in
   two places — so the flag on a `list` or a plain `Box` is an attribute nobody
   reads, and a claim in the DOM that is not true of the element. The audit is
   the natural place to report it and has no rule for it.
6. **(age 2 · value medium) A toolbar containing a composite is two tab stops,
   and only a comment says so.** `focusableMembers` stops at a nested composite
   and does not take it as a member, so a `tablist` inside a `toolbar` keeps its
   own roving tabindex. Every control stays reachable, which is why this shape
   was chosen, and it is not what ARIA describes — the pattern makes the nested
   widget's *current* member the outer widget's member. Doing that means two
   composites writing tabindex on one element, which needs an owner rule the
   section does not have.
7. **(age 2 · value medium) `CONTROL_ROLES` is two roles, hand-written, and
   core can grow a third.** The runtime treats a `Box` as one of a toolbar's
   controls when it carries `role="button"` or `role="link"` *and* an OnTap.
   Those are exactly the two `core.Role` documents as "a tappable container",
   and `wasm/verify/keynav_test.go` pins the pair — but the pin is against a
   hand-written list, not against the property. A future `RoleCheckbox` would be
   a control on the same argument and would silently not be a toolbar member.
8. **(age 2 · value low) The refusals table's `Shape` is prose again, with
   braces around it.** `Blocked` is derived from `core.Roles()` and checked;
   `Shape` — "two dimensions", "a submenu with its own Escape" — is a sentence
   nothing can contradict. There is no obvious authority for it, which is the
   honest reason it is not checked, and saying so is not the same as checking it.
9. **(age 2 · value low) Selection-follows-focus has no browser check.** It is
   covered thoroughly in `keynav_test.mjs`, where `focus()` is an assignment and
   a dispatched callback is an array push. The two facts a browser would add are
   that the *real* focus move fires it exactly once and that a `<button>`
   receiving both an arrow and its own click behaviour does not double-fire.
   `browser.mjs` records dispatches in `window.__dispatched` and asserts nothing
   about them.
10. **(age 1 · value medium) The band's insets are on the control and no
    renderer but the web has been asked about it.** Padding moved from a parent
    to a stretched child, which is exactly identical in CSS flex and is an
    *assumption* everywhere else: SwiftUI's `padding` on a view inside an
    `HStack` and Compose's `Modifier.padding` both behave the same way, and
    neither was measured. `ios/verify`'s stack solver could now settle the
    SwiftUI half — it has a proposal vocabulary and a recording fake — and
    nothing asks it.
11. **(age 1 · value low) A `Header` override still cannot be told the edge was
    withheld.** `GroupedList` withholds `OnEndReached` when the trailing run is
    shut, and the reader's only cue is that scrolling stops loading. A feed
    whose `Footer` is a bare `LoadMore` shows "Load more" and works; one that
    hides the footer when `HasMore` is false shows nothing and looks finished.
12. **(age 1 · value low) The palette check paints swatches, not widgets.** The
    pairs measured are `(ControlBorder, fill)` from the theme, drawn by
    `browser.mjs` itself. What a real `Input` or a quiet `Chip` puts on screen
    goes through `components`, which the browser pass does not run — so a widget
    that stopped reading `Colors.ControlBorder` would keep passing while drawing
    an edge nobody measured. Check 6 is the first that mounts a *layout* rather
    than a swatch, which makes the widget version of this one step closer.
13. **(age 1 · value low) `internal/palette` reflects and `core` cannot tell it
    not to.** A `ComponentDefaults` field whose `Background` is a fill no control
    is ever drawn on has to be excluded by name in a package the theme author
    does not edit. A struct tag would put the fact next to the field, at the cost
    of a tag `core` reads for a test's benefit.
14. **(age 1 · value low) The two-result gobind arm is still transcribed
    nowhere.** `swiftResult` names which of the three arms a refusal is about and
    the stub's comment is checked against that naming, so the *report* is right —
    and a `(T, error)` bridge function would still be refused rather than
    declared. What is missing is a bridge function of that shape to make it worth
    writing.
15. **(age 0 · value medium) `internal/valuefixture` is compared on one
    platform.** `internal/menufixture` reaches three harnesses; this one reaches
    the JVM alone, because SwiftUI has no numeric accessibility value and the web
    hands the strings to a browser. The web half is not as settled as that
    sentence: a browser applying ARIA's `0..100` default is a claim nothing in
    this repository has watched, and `browser.mjs` can now read a rendered
    `<progress>`-roled node's computed `aria-valuenow`.
16. **(age 0 · value low) `ValueRange.Progress` has no Go consumer.**
    `components.ProgressBar` always states all three numbers, so it is
    determinate by construction and never asks. The function is core's own rule
    made executable and its only readers are a test and a transliteration —
    defensible, and one honest step from a rule that lives in a test.
17. **(age 0 · value low) The sticky check writes `core.StickyHeader()`'s three
    declarations by hand.** `browser.mjs` mounts JSON trees, so the fixture
    states `Position`, `Top` and `ZIndex` itself rather than asking Go. A change
    to the Go prop that dropped one would leave the browser check green and
    passing about a tree nobody builds. `wasm/verify` already generates tables
    from Go for the tag, stack-axis and palette pins; this one could join them.
18. **(age 0 · value low) The two exception tables for themes are empty.**
    `quietThemes` and `universalThemes` cost a sentence each and neither has an
    entry, so the arms that read them have never run against a real case. That is
    the honest starting state and it is also a rule nobody has exercised — the
    backdrop exclusions had two entries on their first day.
