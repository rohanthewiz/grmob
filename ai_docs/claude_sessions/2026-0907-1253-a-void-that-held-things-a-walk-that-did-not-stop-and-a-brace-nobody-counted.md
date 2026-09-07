# Session: a void that held things, a walk that did not stop, and a brace nobody counted

Session: https://claude.ai/code/session_017k47JkhG1WBZR5nP29c7Ge
Date: 2026-09-07 (follows "a-fill-nobody-draws-on-a-bind-nobody-ran-and-a-band-nobody-measured")

## Ask

"Commit the previous session's work and take on the oldest 5 items."

| # | age·value | item |
|---|---|---|
| 1 | 2·medium | A hand-assembled `Spacer`'s children are dropped on both natives |
| 2 | 2·medium | The nested-composite finding is reported in Go and asserted in JavaScript, and the two are not held together |
| 3 | 2·low | `swiftTypeBody`'s cut is "the first closing brace in column one" |
| 4 | 2·low | `notTappable`'s reasons are prose by construction |
| 5 | 2·low | The stale-download skip is a guard nothing exercises |

All five closed, in two commits: `e7ec50e` (the previous session's tree, verified
green first) and `7e7c7f2`.

Three of the five share a shape the last two sessions' did not: **one fact stated
in one place, and a second reader that had its own answer.** A Spacer's arity, a
walk's stopping rule, and a Swift type's end were each written down once and
consulted by something that had quietly decided otherwise.

**One of the five was a live wrong answer rather than an unheld claim.** Item 2's
report has been telling authors the opposite of what the runtime does for five of
nine pairs.

## Phase 1 — a void that held things (item 1)

The four properties a Spacer's arm used to drop — its `Style`, accessibility
props, callback IDs and margin — were fixed a session ago. The children were
named as still open, on the grounds that the divergence was **unreachable from
Go**.

That is true of `core.Spacer(n)` and false of `*core.Node`. `htmlout` exports
any node a caller hands it and the WASM runtime renders any tree the wire
carries, so the observable behaviour was a subtree that appeared in a browser
and vanished on a phone.

**The DOM side is also the side that cannot move**, which is what settled the
direction:

```
   the runtime addresses patches by data-node-path attributes written while
   walking node.Children, so an element it declines to emit takes every patch
   beneath it with it
```

That is the same rule that keeps `Fragment` and `Theme` boxed there. So "make
all four agree by dropping" was never available, and the natives emit.

**The axis had to be stated, not chosen.** A Compose `Box` and a SwiftUI
`ZStack` overlay; the DOM lays a non-stack type's children out in block flow. A
native stack beside DOM block flow is exactly what `stackAxes` exists to
prevent — it is the argument `Box` and `SafeArea` were moved off `ZStack` by. So
`Spacer` went into the table on `column`, and both renderers read it from there:
**a Spacer with children is a `Box` with a fixed size.** The row costs a
childless one nothing — a childless flex box of fixed size lays out as a
childless block box of the same size.

Each native took the route its existing code already had:

```
   Compose   GrMobColumn(node, extra, outer = Modifier.size(...))
             which produces boxModifier(extra, gesture).then(size) — the
             identical chain the arm used to spell out

   SwiftUI   Color.clear stays as the thing that is SIZED, with the flex stack
             laid over it
```

The SwiftUI half is where the reading mattered. `grMobBox` paints its background
**inside** its own dimension frame, so on an axis the `Style` claims, the fill
covers only what the content asks for. `Color.clear` is flexible and asks for
all of it; an empty flex stack is zero-sized and would ask for none — which
would have repainted the documented `Width(200)` case as a fill of nothing.
Overlaying leaves that whole argument untouched.

**Item 1 moved item 3's pins.** `spacer_test.go` held `boxModifier` and `.size()`
in order *in the arm*; the ordering claim now lives one call down in
`GrMobColumn`, so both ends are checked — an arm that handed the size to a
composite which applied it first would otherwise satisfy the pin.

## Phase 2 — a walk that did not stop (item 2)

The Go audit said, for every ordered pair of composites:

> the pair is two tab stops and **the outer widget's arrows step over the inner
> one whole**

The runtime's own comment says the opposite for half of them:

> `compositeMembers` descends *through* a composite of the other kind, because
> an option below a tablist is still the listbox's option — the roles say whose
> it is

**Nine ordered pairs, four stop and five descend.** A `listbox` around a
`tablist` pools any option buried in the strip into the outer widget's rotation:
its arrows land *inside* a widget they are not steering, which is the opposite
of stepping over it and a different bug to go looking for.

The rule is two lines of JavaScript and each is one case:

```
   compositeMembers   stops where the nested container's member role is its own
   focusableMembers   stops at any composite — a toolbar names no member role,
                      so nothing says whose a plain button is
```

`core.CompositeMemberRole` and `core.CompositeWalkStopsAt` state that once, for
the same reason `KeyboardComposites` states its own list in core: the audit is
the pass whose job is telling a Go author what their tree amounts to, and it
cannot ask a runtime written in JavaScript. The audit now picks between two
sentences, and both directions are asserted — a message carrying the wrong
phrase, or both, fails.

Three checks hold it together, and each covers what the others cannot:

```
   keynav_test.go    the two runtime lines are the two lines core describes
   keynav_test.go    both outcomes are reachable — a rule that answered the
                     same way for all nine pairs would pass the source pins
   keynav_test.mjs   a listbox > tablist > option tree in a real DOM
```

The mjs case is contrived on purpose. It is the smallest tree that tells the two
rules apart, and without it a runtime that stopped at every composite would pass
every other check in that file.

## Phase 3 — a brace nobody counted (item 3)

`swiftTypeBody` ended a declaration at the first `\n}\n`. True of every Swift
type in `Renderer.swift`; not true of Swift, where indentation carries no
meaning. Three legal shapes break it — a `#if` block, a multi-line string
literal's verbatim content, a nested type formatted flat — and **the failure
mode is the bad one**: a short cut still returns a string, so every
`strings.Contains` below goes on running against a body that lost its second
half and passes by reading nothing.

The fix was already in the package. `matchingBrace` counts braces while skipping
comments and string literals, written for the other language's dispatch blocks;
it took a `dispatchSyntax` only for its error message, so it now takes the two
strings it actually needed.

It grew a `"""` arm, which is the one case no reformatting can fix. Writing the
break-test for that arm is what proved it was needed and nearly proved it was
not:

```
   content with no quote      the single-quote scanner pairs the three
   content with "a word"      delimiter characters off two at a time and comes
                              out even — the fixture passes either way

   content with one quote     the closing delimiter is half-consumed and the
                              brace is exposed
```

The first fixture written was a MISS. An odd quote is what makes the case
discriminate.

## Phase 4 — four kinds with an authority (item 4)

`notTappable` censuses every `core.Role` against "should this make a Box one of a
toolbar's controls", and carried a comment arguing that a reason string cannot
be checked and that what a census buys is the moment of writing one.

Half right. "This role is one of ARIA's structural containers" is not an
opinion — it is the Required Owned Elements row, which `aria/verify` already has
in machine form and already uses to keep `refusal.Nesting` honest.

**Four of the seven kinds derive; three do not**, and the type says which:

```
   structure   ARIA requires the role to own children, or a role that is not a
               composite requires to own it
   composite   core.KeyboardComposites()
   member      core.CompositeMemberRole() over those containers
   valued      the fixture's attribute list carries aria-valuenow

   landmark    ARIA's landmark and live-region groupings are prose in the
   live        specification's text rather than rows in a role definition, and
               aria/gen reads role definitions
   content     the fallback — a derivation for it would be the negation of the
               other six and would agree with anything
```

That is the `Nesting`/`Shape` split one file over: declare, derive, and let the
two disagree in a test rather than in somebody's memory.

The table moved to `aria/verify`, because the fixture cannot be imported
backwards into `core`. What stays in `core` is the half with no ARIA in it — its
two exported role lists agreeing with each other.

**Two clauses in the structure derivation are load-bearing and both were found by
the check failing:**

```
   exclude composites and       listbox owns option and tablist owns tab, so
   their members                without it every composite and every member
                                derives as structure and three kinds collapse
                                into one

   restrict to core.Roles()     menu, menubar and tree own `group` — three
                                refused patterns — and would have filed the
                                content fallback as structure
```

`group` is the case that shows the rule is doing work rather than agreeing by
construction: it is in `listbox`'s owned list, ARIA uses it as a pure wrapper,
the table files it as content, and the derivation agrees for the stated reason.

## Phase 5 — a branch a test can reach (item 5)

`aria/verify` turns a 1.1 copy of the specification into a SKIP rather than a
fixture diff. That guard had never executed on any machine.

Its *subject* was covered — `aria/spec` proves `Parse` refuses a 1.1 document
and names the fetch in the refusal. But `Parse` **fails**; this guard exists so
the same document produces a skip instead, because a checkout is not broken
because the copy beside it is old. "Turn that failure into a skip" is a
different claim, and nothing had made it.

Extracted as `localCopyGate(raw, readErr)`, whose inputs are bytes and an error
— so all its answers can be handed over directly rather than arranged on
somebody's filesystem.

**Exercising it turned up a fourth answer.** A truncated download, an error page
or a proxy interstitial has no title heading at all, and was folded in with the
edition mismatch:

```
   the local copy of the specification is WAI-ARIA "" and the fixture is
   generated from 1.2
```

That is not a sentence to hand somebody. It is its own branch now.

## The break-tests

**28 caught, 0 missed, 0 restore failures.**

    item 1   the kotlin arm, FlexChildren, Color.clear, the axis, the row    5
    item 2   both directions of the rule, the toolbar's member role, the
             audit's split, both runtime walks, the DOM behaviour            7
    item 3   the column-one cut, the comment skip, the multi-line arm        3
    item 4   four misfiled roles, a dropped row, an empty argument, the
             ownership derivation, a tappable container that owns children   8
    item 5   a stale edition let through, a skip on a current copy, the
             headingless branch, the fetch command, SpecVersion's heading    5

Two of them only became catchable after the first attempt missed:

- **The `"""` arm was invisible to its own fixture.** Content with an even
  number of quotes comes out even under the single-quote scanner, so the case
  had to carry an unpaired one.
- **`stackAxes` losing the Spacer row** had to be checked against
  `TestNativeSpacer...` as well as the runtime conformance test, because the
  runtime restates the table and would have agreed with its absence.

## Verification

Eight paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    wasm/verify/run.sh          replay + 17 mjs suites + BROWSER PASS
                                (68 keynav subtests, 20 swatches, 6 real
                                 widgets on both edges, a sticky band,
                                 22 value ranges)
    ios/verify/run.sh           flex + stack + 3 band arrangements
                                + 15 picker menus + replay + view + app
    android/verify/run.sh       15 picker menus and 22 value ranges, on a JVM
    android ./gradlew compileDebugKotlin --offline
    GOOS=js GOARCH=wasm go build ./...

New files:

    aria/verify/tappable_test.go       the census, with four kinds derived

Changed:

    android/.../Renderer.kt            the Spacer arm, and one dropped import
    ios/GrMob/Runtime/Renderer.swift   GrMobSpacer's overlay and its argument
    htmlout/stack.go                   the Spacer row and its reason
    htmlout/stack_test.go              Spacer out of the block-flow census
    wasm/grmob-runtime.js              the Spacer row in stackAxisFor
    core/role.go                       CompositeMemberRole, CompositeWalkStopsAt
    core/a11y_audit.go                 the two reaches, and which is reported
    core/a11y_audit_test.go            the reach asserted per pair
    core/role_control_test.go          reduced to the half with no ARIA in it
    aria/verify/generated_test.go      localCopyGate and its four answers
    aria/verify/doc.go                 the new check in the charter
    mobile/verify/stackalign_test.go   swiftTypeBody's real cut, and its tests
    mobile/verify/switchlabels_test.go matchingBrace's signature and `"""`
    mobile/verify/spacer_test.go       both ends of the Compose chain
    mobile/verify/stacking_test.go     the Spacer children pin
    wasm/verify/keynav_test.go         the walks held to core's rule
    wasm/verify/keynav_test.mjs        a descending pair in a real DOM

## Docs

`docs/platforms/native.md`: how each native emits the children and why the
axis is not theirs to choose. `docs/platforms/wasm.md`: `Spacer` in the stack
table with the reason it moved, and the two reaches of a nested composite.
`docs/concepts/debug-mode.md`: a table of which pairs step over and which land
inside. `aria/verify/doc.go`: the derived census. `ROADMAP.md`: five entries.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here. Value is the payoff of doing it, not the effort: `high` means
something is being worked around today or a second consumer has arrived,
`medium` means it blocks one named thing, `low` means it is a gap nobody has
bumped into yet.

1. **(age 2 · value low) The browser's value-range divergence half cannot fail
   on this machine.** A row with an unparseable number must *not* agree with
   `core.Progress`, and it does not — so the assertion has never fired and a
   mutation of it is invisible. This is the same shape item 5 just closed, one
   target over: the subject is covered and the branch is not.
2. **(age 2 · value low) `TestTheWidgetSwatchRingIsThePaletteRole` cannot see
   the swap it is named for**, in both directions: the field swatch can claim
   `Colors.ControlBorder` as its authority and pass, because the role and
   `Components.Input.BorderColor` hold the same hex in all three bundled
   themes. A fourth bundled theme that split them would sharpen the whole
   family — and would be a palette added for a test.
3. **(age 2 · value low) `browser.mjs` requires the transcript and skips on no
   Chrome.** Two preconditions with two stances, both right and neither obvious
   from outside.
4. **(age 2 · value low) The widget swatches mount one tree per case and
   screenshot each.** Six Chrome round trips now rather than three, because the
   Input swatch doubled them — each page paints its own background, so a single
   page carrying all of them would halve the browser pass's slowest section at
   the cost of the per-theme fill being a sibling rather than the document.
5. **(age 1 · value medium) Nothing has asked a browser whether the band's
   insets survive overflow.** `ios/verify` says the two arrangements divide an
   overflow deficit differently, because that solver shrinks in proportion to a
   base that includes the child's own padding. CSS distributes shrink over the
   *inner* flex base size, so a browser is likely to agree where SwiftUI does
   not — which would make it a genuine cross-target divergence rather than an
   artefact. `wasm/verify` already mounts trees and measures rects.
6. **(age 1 · value medium) The disclosure band's control does not grow.** The
   collapsible band puts `FlexGrow(1)` on the heading wrapper and the insets on
   the *button* inside it, so whether the tap target spans the band depends on
   whether a non-growing child fills a growing parent — a cross-axis question
   `GrMobFlexSolver` does not answer and no target has been asked. The picture
   in `bandInsets`' doc comment assumes the answer.
7. **(age 1 · value low) `core.ComponentDefaults.Text` is a field nothing
   reads.** `core.Text` builds its Style from its own props and never touches
   the theme, so the two bundled themes that state a `Components.Text` are
   describing a widget that does not exist. It costs the census nothing; what
   it still costs is a theme author who fills it in and sees nothing happen.
8. **(age 1 · value low) The two-table coupling in `mobile/verify` is only as
   total as `bindableGoTypes`.** `error` is in it because a bind was run; the
   next type gobind can carry and this bridge does not use is in exactly the
   position `error` was. Nothing enumerates what gobind binds.
9. **(age 1 · value low) `internal/bandfixture`'s content sizes are made up.**
   That is right for the arithmetic and it means the height agreement is
   checked against *insets* rather than against text — the other half, that a
   bold caption is no shorter than a plain one, is stated and unmeasured.
10. **(age 1 · value low) The band check models one renderer.** Compose applies
    padding through a `Modifier` and distributes weight through its own `Row`,
    so the same question has a second answer nobody has asked; `android/verify`
    is a JVM pass and could.
11. **(age 0 · value medium) A Spacer's children overflow on three targets and
    are measured into the size on Compose.** `Modifier.size` sets min and max,
    so a child taller than the void is squeezed where CSS and SwiftUI let it
    spill. It is not a Spacer property — it is what every fixed-size container
    does on that target — which makes it the more interesting question: nothing
    anywhere has asked whether the four targets agree about overflow for *any*
    fixed-size box.
12. **(age 0 · value low) The descending nested-composite pairs are reachable
    and have no example.** `keynav_test.mjs` builds a `listbox > tablist >
    option` tree because it is the smallest thing that tells the two rules
    apart, and the doc comment calls it contrived. Whether any real screen can
    produce one is unasked — if none can, the five descending pairs are a
    finding nobody will ever see, and that is worth knowing before the next
    reader sharpens the message further.
13. **(age 0 · value low) `swiftTypeBody`'s anchor is still a substring.** The
    cut is syntactic now; finding the declaration is `strings.Index`, so an
    anchor matching a mention of the type in a doc comment above it would cut
    from there. Every current anchor starts with `private struct` or `struct`,
    which is what keeps it working.
14. **(age 0 · value low) `core.CompositeMemberRole` returns "" for two
    different reasons.** A toolbar has a keyboard and no member role; a
    `RoleHeading` has neither. The doc says callers separate them by asking
    `KeyboardComposites` first, which is a contract nothing enforces — a
    caller that forgot would treat every non-composite role as a toolbar.
15. **(age 0 · value low) `localCopyGate` is the only gate of its kind that is
    tested.** `ios/verify` skips on a missing iPhoneOS SDK and
    `android/verify` on a missing Kotlin compiler, both by the same argument
    and both as inline shell conditions no test reaches. The Go one was
    extractable because its inputs are bytes; those two are `command -v`.
