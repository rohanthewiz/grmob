# Session: a refusal Compose could express, two arms nobody could tell apart, and a version nobody had read

Session: https://claude.ai/code/session_017k47JkhG1WBZR5nP29c7Ge
Date: 2026-09-07 (follows "a-target-nobody-could-press-a-declaration-nobody-could-write-and-two-gates-nobody-could-reach")

## Ask

"Show me what's in the Next list", then "take all the high and medium items".

Four items, one commit. Each was a claim resting on something nobody had
looked at: a target's capability, a return value, a solver we already own, and
somebody else's source at somebody else's version.

| # | age·value | item |
|---|---|---|
| 4 | 0·high | `core.FlexShrink(0)` is expressible and Compose cannot honour it |
| 5 | 0·medium | `CompositeWalkStopsAt`'s two stopping arms are indistinguishable |
| 6 | 0·medium | The band's Compose row is derived and cannot be pinned |
| 7 | 0·medium | Nothing measures the fixed-size answer on the two natives |

## Item 4 — a refusal is not a proportion

The note that raised this said Compose "has no proportional shrink at all, so
'do not shrink' is a declaration that means something on three targets and
nothing on the fourth". The first half is exactly right and the second does not
follow from it.

`foundation-layout`'s measure policy, read from the version this build
resolves:

```kotlin
   // First measure children with zero weight.
   mainAxisMax = (mainAxisMax - fixedSpace).coerceAtLeast(0)
```

Each unweighted child is offered the main-axis space its predecessors did not
take. **There is no factor anywhere in that arithmetic**, which is why a
*fractional* flex-shrink has nothing to scale — and zero is not a fraction. It
is a refusal, and a refusal is expressible:

```kotlin
private fun Modifier.pinMainAxis(horizontal: Boolean) = layout { measurable, constraints ->
    val unbounded = if (horizontal) constraints.copy(minWidth = 0, maxWidth = Constraints.Infinity)
                    else            constraints.copy(minHeight = 0, maxHeight = Constraints.Infinity)
    val placeable = measurable.measure(unbounded)
    layout(placeable.width, placeable.height) { placeable.place(0, 0) }
}
```

**The second line is the one that fails silently.** Reporting
`constraints.constrain(...)` instead compiles, looks like the better-behaved of
the two — a well-mannered layout respects its constraints — and produces a
third thing that matches no target: the row's running total never overflows, so
the child is drawn spilling out of a box its parent still believes it fits
inside. `mobile/verify` refuses that spelling by name.

```
   Row(maxWidth = 120)   [  pinned child, 200 wide  ][ next child ]
                         └──── reports 200 ────┘      └ offered 0 ┘
```

What still diverges is the **siblings**: CSS shares the deficit among the items
that can shrink, in proportion to their bases, and a Compose Row gave the
earlier children what they asked for and offers the later ones what is left.
That is the no-proportional-shrink divergence, unchanged. The pinned child's own
size — which is what the declaration is *about* — now agrees on all four
targets, and order does not matter to it: `remaining` is ignored whether the pin
is first or last.

A weighted child is not given the modifier, and that is stated rather than
omitted: `Modifier.weight` already fixes that child's main axis as both minimum
and maximum, and CSS never applies grow and shrink at once either.

`GrMobStyle.kt` joins the sentinel census as its fourth spelling, anchored on
the whole `when` so that the *unset* arm is pinned too — a runtime that
defaulted the factor to 0 would pin every item in every layout, which no
comparison of sentinels would notice.

## Item 5 — two arms, one answer

`CompositeWalkStopsAt` returned `true` for a container with no keyboard and for
a toolbar whose walk genuinely stops. The distinction was written into the code
with a paragraph on each arm, and thrown away in the return.

```
   CompositeWalkNotApplicable   no keyboard, so no walk for this to be about
   CompositeWalkStops           the arrows step over the inner widget whole
   CompositeWalkDescends        the arrows can land inside a widget they are
                                not steering
```

The bool is kept and is now a *reading* of the enum rather than a second
implementation — `!= CompositeWalkDescends`, held to it over every ordered pair
of declared roles, because the pairs where the two could disagree are exactly
the ones a sweep over `KeyboardComposites()` would never mount.

`checkNestedComposite` switches on all three, and the third arm returns without
a finding: the previous shape would have written "the outer widget's arrows step
over the inner one whole" for a pair with no rotation at all.

**This is the mutation the last session recorded as invisible.** Swapping the
two arms' bodies now fails.

## Item 6 — reading someone else's source at the right version

The census's Compose row was a reading of `foundation-layout` **1.10.0**, which
is what a gradle cache happened to hold; the BOM resolves **1.6.8**. Two things
were missing, and neither costs a network call at test time.

**The version is derived.** `compose-bom-2024.06.00.pom` is in the cache on any
machine that has ever built the app, its `dependencyManagement` names
`foundation-layout`, and that listing is what the gradle file's number is held
to. A BOM bump now fails by name — verified against the other BOM this machine
happens to hold cached.

**The sources are fetchable once.** A `composeLayoutSources` configuration
(`transitive = false`, artifact-only; without it the resolution drags in the
desktop artifacts and skiko's two runtime variants cannot be chosen between) and
a `fetchComposeLayoutSources` task. Run it, and the claims are read out of the
jar:

```
   Size.kt                          SizeElement(minWidth = width,
                                                maxWidth = width,
                                                enforceIncoming = true)
   RowColumnMeasurementHelper.kt    mainAxisMax - fixedSpace
```

**Both claims hold at 1.6.8.** The census was right; what it lacked was any way
to have been wrong. Machines that have never fetched skip that half with the
command in the message, which is why the version half is a separate test — that
one runs everywhere.

## Item 7 — measuring what we already run

The census's SwiftUI row was derived, and half of it did not have to be. The
cross axis is a SwiftUI fact (`.frame` proposes and does not enforce), but the
main-axis squeeze is `GrMobFlexSolver`'s — ours, and `ios/verify` executes it.

`checkFixedSizeContainer` runs the census's own box: the container takes its
declared extent, the child is squeezed to it, and a pinned child keeps its size
and overflows — on both containers, because "the main axis is the one that
squeezes" cannot be told from "width is the one that squeezes" with one.

It uses `browser.mjs`'s numbers, and `wasm/verify` holds the two harnesses to
one fixture. **Two passes agreeing about different boxes is a weaker statement
than the census makes, and neither pass could tell** — both would keep printing
OK. The same test also asks whether the child is still bigger than the
container, because the four numbers can agree perfectly while measuring nothing.

## The break-tests

**18 run, 17 caught, 1 miss — fixed rather than recorded.**

    item 5     the two stopping arms swap returns, the bool becomes its own
               implementation, two values print the same                     3
    item 7     a harness edits its own fixture, the child stops being
               bigger, the squeeze expectation inverted                      3
    item 4     the Row stops pinning, a doc comment stands in for the call,
               the axes transposed, the pin clamps what it reports, the pin
               measures bounded, the Kotlin sentinel drifts, an unset factor
               stops meaning 1, the Swift row passes the raw field           8
    item 6     the sources version stops matching the BOM, the BOM is bumped
               and the reading is not, a claim stops matching the source,
               both skip paths under an empty GRADLE_USER_HOME               4

**The miss, and why it is worth naming.** "The Row stops pinning" passed:
`RowChildren`'s check for `shrinkPinned` was satisfied by the doc comment of the
function *below* it, which `declSource`'s deliberately coarse cut carries along.
That is the anchor-matches-a-mention failure `swiftDeclIndices` was written for
one file over, and it has the same answer — mask the comments and literals, then
ask the question of what is left. `maskSwiftNonCode` is named for Swift and is
not specific to it: Kotlin spells all four constructs identically.

One mutation remains invisible and is stated rather than hidden: the audit's
`CompositeWalkNotApplicable` arm cannot be reached from `checkNestedComposite`,
which tests `hasKeyboard` on both ends before it asks. The arm is written out
anyway, and the enum is what makes it possible to write.

## Verification

Eight paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    wasm/verify/run.sh          replay + 19 mjs suites + BROWSER PASS
    ios/verify/run.sh           gate + flex (now incl. the fixed-size census)
                                + stack + 3 bands + 15 menus + replay + view
                                + app
    android/verify/run.sh       gate + 15 picker menus + 22 value ranges
    android ./gradlew compileDebugKotlin --offline
    GOOS=js GOARCH=wasm go build ./...

New files:

    mobile/verify/shrink_test.go        both natives' shrink call sites
    mobile/verify/composelayout_test.go the derived version and the source it reads
    wasm/verify/fixedsize_test.go       one fixture across two harnesses

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

1. **(age 3 · value low) The two shrink controls in the band check cannot guard
   their own deletion.** Dropping the control *and* `min-width: 0` together
   leaves both arrangements at their natural width, agreeing by never reaching
   the arithmetic. The same is true of the scroll control in check 3 and the
   scroller in check 6 — either an argument that this is what a control is, or
   an argument for one source-level pin over the three.
2. **(age 3 · value low) The band check mounts only definite offers and
   `max-content`.** Reading `bandfixture`'s negative offer as the CSS
   `max-content` is a judgement stated in a comment and held to nothing;
   `min-content` is the other candidate and a different question.
3. **(age 3 · value low) `browser.mjs`'s check numbering is not a sequence.**
   The header lists eight and the body has two sections numbered 6, plus 9 and
   10 added to a broken sequence.
4. **(age 1 · value low) `gobindCarriesUnused` refuses six types whose
   parameter spellings are legible.** Only the `(value, error)` out-pointer
   spelling is unreadable, so a bridge function taking an `int8` and returning
   nothing is refused for a reason that does not apply to it. Splitting the
   refusal by position would admit those.
5. **(age 1 · value low) The rendered-band grid mounts nine trees and reads no
   pixels.** Every other browser check that mounts real widgets samples a
   colour; this one measures rects only, so a band that laid out correctly and
   painted nothing would pass.
6. **(age 1 · value low) `maskSwiftNonCode` and `matchingBrace` answer the same
   question twice.** They share "what in this file is code" and are two separate
   scanners with the arms written in the same order — a comment asking them to
   agree rather than a mechanism making them. Now with a third caller (`codeOf`),
   which raises the cost of them disagreeing.
7. **(age 1 · value low) The two `gate_test.sh` scripts share a shape and no
   code.** Each has its own `expect` helper, and a third harness's gate would
   write a third.
8. **(age 0 · value high) Nothing executes `pinMainAxis`.** Every other claim in
   this session's work is measured or read from source; the one piece of new
   *runtime behaviour* is held by a source-level pin and a compile. The
   arithmetic it produces (the pinned child keeps its base, the ones after it
   are offered nothing) is a statement about androidx's measure policy that
   could be checked the way `GrMobFlexSolver` is — if a Compose measure could be
   run anywhere in this repository, which today it cannot.
9. **(age 0 · value medium) The Compose sibling distribution after a pin is
   documented and unmeasured.** CSS shares the deficit proportionally; Compose
   offers the later children what is left, which after an overflow is nothing.
   That divergence is now written down in three places and rests on the same
   reading of the measure policy — one more claim about someone else's
   arithmetic with no fixture behind it.
10. **(age 0 · value medium) The source-level half of the Compose census skips
    on every machine that has not fetched.** That is the honest state, but it
    means the check most likely to catch an androidx change is the one least
    likely to run — a CI machine skips it silently. A `run.sh` that names the
    skip, or a fetch wired into `android/verify`, would make the gap visible.
11. **(age 0 · value medium) Only three checks in `mobile/verify` mask their
    comments.** `codeOf` exists because a doc comment satisfied a
    `strings.Contains` that was meant to find a call. Every other check in that
    package still reads `declSource`'s output raw, and the coarse cut carries the
    next declaration's doc comment into every one of them. The hole was found in
    one file by a break-test; nothing has looked for it in the others.
12. **(age 0 · value low) `ext.composeLayoutVersion` is a second spelling of a
    derived fact.** It exists because the resolution that would derive it cannot
    be asked for a sources jar without dragging in the whole Compose graph. A
    test holds it to the BOM, which is the right shape — but a reader of
    `build.gradle` alone still sees two versions and no mechanism.
