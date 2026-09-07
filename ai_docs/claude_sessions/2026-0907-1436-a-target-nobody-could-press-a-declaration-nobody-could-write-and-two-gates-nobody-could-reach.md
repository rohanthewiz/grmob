# Session: a target nobody could press, a declaration nobody could write, and two gates nobody could reach

Session: https://claude.ai/code/session_017k47JkhG1WBZR5nP29c7Ge
Date: 2026-09-07 (follows "a-half-nobody-could-reach-a-stance-nobody-stated-and-a-divergence-nobody-had-asked")

## Ask

"Commit the last set of changes then take on the oldest 6 items", then "take on
the next 6 items", then commit, push and wrap.

Twelve items closed in two commits. What ties them together is one distinction
used twelve times: **a claim and a measurement are not the same thing**, and
five of the twelve turned out to be claims that had never been either
measurable or expressible.

## Part one — the oldest six

| # | age·value | item |
|---|---|---|
| 1 | 2·medium | The disclosure band's control does not grow |
| 2 | 2·low | `core.ComponentDefaults.Text` is a field nothing reads |
| 3 | 2·low | The two-table coupling in `mobile/verify` is only as total as `bindableGoTypes` |
| 4 | 2·low | `internal/bandfixture`'s content sizes are made up |
| 5 | 2·low | The band check models one renderer |
| 6 | 1·medium | Whether four targets agree about overflow for a fixed-size box |

### Item 1 + 4 — one mount, two claims the arithmetic cannot reach

`internal/bandfixture` is the band as **numbers**, and that is the right shape
for a distribution and out of reach of two other things:

```
   the tap target   the disclosure branch puts the insets on a BUTTON one level
                    inside the Row's growing heading wrapper, so "does a press
                    land on the whole band" is a CROSS-axis question and
                    GrMobFlexSolver is a main-axis distributor. The picture in
                    components.bandInsets had been assuming it.

   the taller child every SameHeight case rests on the padded control being the
                    band's tallest child, and half the reason is "a bold caption
                    is no shorter than a plain one" — a measurement of glyphs.
```

`gen.go` now renders **real** `components.GroupHeader`s — three shapes through
three bundled themes — and emits the paths of the nodes to measure, found by
walking the rendered tree rather than spelled as indices. `browser.mjs` mounts
all nine at once.

**The button does fill the wrapper.** The tap target spans the band, and it does
so by a cross-axis default rather than by anything the widget declares.

**And measuring it found something the widget promises and does not quite do.**
`GroupHeader.ControlStyle` says a caller who adds `OnToggle` gets "a control and
not a relayout". Held to pixels that is exact for the chrome and off by a point
for the height, in every theme:

```
   the disclosure's button holds a chevron the plain band does not
   a control is as tall as its tallest child plus its own insets
   ▾'s line box is 16px where the caption's is 15px
```

Content, not chrome — so it is checked as the equation it is:

    disclosure band − plain band  =  max(0, chevron line box − label's)

which fails in both directions. More is chrome drifting between the branches;
less is a control no longer as tall as its tallest child.

**One bug of my own, caught by the pass.** The branch pairing keyed on
`collapsible` alone, so the count-hidden disclosure overwrote the badged one and
the comparison ran between two bands of different shapes. The failure said the
trailing edges differed by 38.5px, which is the badge — the message named the
thing that was wrong with the test.

### Item 2 — a field nothing reads

`core.Text` builds its Style from its own props and never touches the theme, so
two bundled themes stated a `Components.Text` — a size, an ink, a white
`Background`, twelve points of padding, a radius, copied from the field base
beside it — describing a widget that does not exist.

**Wiring it was the wrong fix and it is worth saying why.** An unconditional
theme base on every text node puts a white fill and twelve points of padding
behind every glyph in the framework, and stating an ink on every one overrides
the inheritance a label inside a filled control depends on. A run of words
already has an authority — `Typography`, which widgets spend explicitly — so
this was a second authority for something that had one, not a feature with a
missing implementation.

So the field is gone, and what replaces it is the general guard:
`TestEveryComponentDefaultReachesAWidget` renders a witness per field through a
theme whose base for that one field carries a marker no widget sets for itself
(`Animation`, which has no default, no resolver and no spender), and looks for
the marker on the node that comes back. Both directions break-tested: a widget
that stops merging its base fails, and so does a field added with nothing behind
it.

### Item 3 — enumerating what gobind carries

`bindableGoTypes` decides which functions the stub is *required* to declare, and
what was outside it was decided by hand with the wrong question — "does this
bridge use it". `error` was left out on exactly that reasoning, and gobind binds
it: such a function would have produced a symbol, gone undeclared in the stub,
and taken `GomobileBridge.swift`'s type-check with it.

The fix is to stop deciding. The carried set is parsed out of `bind/gen.go`'s
own `isSupported` in the pinned module cache, and every member must be
classified — spelled in `gobindSwiftTypes`, or refused with a reason in
`gobindCarriesUnused`.

**Refused rather than spelled, because the spellings are not readable.**

```
   uint8       objcType maps it to a bare `byte` nothing gobind emits declares
   []byte      NSData* is legible; the (value, error) split is not, and no
               golden exercises it
   int8..f64   the parameter spellings are in basictypes.objc.h.golden; the
               OUT-POINTER spelling for a (value, error) result is not, and
               gobindErrorOutPointer's two rows were read off a real bind —
               one of them is UnsafeMutablePointer<ObjCBool>, which no rule
               stated here would have predicted
```

A bridge function using a refused type now fails **by name** instead of dropping
out of the check, which is the hole closed rather than deferred.

**`isSupported` and not `objcType`, and they disagree.** The speller can spell
uint16/32/64; the gate does not admit them. Deriving from the speller would
demand classifications for three types `gomobile bind` will not bind.

### Item 5 + 6 — what Compose can and cannot be asked

Both items bottom out in the same limit, and establishing it precisely *is* the
deliverable.

**Item 6 was asked and the answer is per-axis**, which is not what the note that
raised it assumed:

```
                     main axis            cross axis
   WASM runtime      squeezed             spills          measured
   htmlout           squeezed             spills          inherited
   SwiftUI           squeezed             spills          derived
   Compose           squeezed             squeezed        derived
```

A browser does not simply let the child spill: it is a flex item whose shrink
factor defaults to 1 and whose automatic minimum, being empty, is 0. Two
containers are mounted, because with one, "the main axis is the one that
shrinks" cannot be told from "height is the one that shrinks".

The natives are derived from the platform call each renderer makes, pinned at
those call sites — and the two near-misses are what the pins refuse:
`Modifier.requiredSize` ignores incoming constraints and `sizeIn` sets a range,
either of which leaves the mapping compiling and the row silently wrong.

**Item 5's premise was wrong and that is the finding.** "android/verify is a JVM
pass and could" — it cannot. It runs Kotlin that *imports nothing*, and a
Compose measure policy measures `Measurable`s into `Placeable`s through
compose-ui. Nor can the policy be read and pinned the way the gobind mapping is:
this machine's cache holds `foundation-layout` **1.10.0** sources and the build
pins the BOM to `2024.06.00`, which resolves to **1.6.8** — whose sources are not
cached. A pin against the wrong version is the mistake `gobindVersion` exists to
prevent.

The derived answer is recorded (Compose agrees with the web, for a third reason:
no proportional shrink at all, so the whole deficit lands on the weighted control
and the badge keeps its width). What is **tested** is the premise —
`TestTheComposeRowDelegatesItsDistributionToCompose` — because a renderer that
stopped delegating would put the answer back in reach and make it ours.

## Part two — the next six

| # | age·value | item |
|---|---|---|
| 7 | 1·low | Descending nested-composite pairs are reachable and have no example |
| 8 | 1·low | `swiftTypeBody`'s anchor is still a substring |
| 9 | 1·low | `core.CompositeMemberRole` returns "" for two different reasons |
| 10 | 1·low | The two shell gates are inline conditions no test reaches |
| 11 | 0·high | `core.FlexShrink(0)` does nothing on any target |
| 12 | 0·medium | The sticky fixture's `FlexShrink: 0` is inert |

### Items 11 + 12 — a declaration nobody could write

Every optional number in a `core.Style` means "unset" by being zero, and that is
right for all of them but one:

```
   unset       the item shrinks under pressure, in proportion to its base
   zero        the item keeps its size and the container overflows
```

`Style.Merge`, `htmlout.Export` and the WASM runtime each guarded on
`FlexShrink != 0` — three independent guards, all correct for every other field
and all wrong for this one. The prop compiled, applied, serialised and did
nothing.

**`core.ShrinkNone = -1`**, because CSS forbids a negative shrink factor, so no
renderer can be handed one legitimately and `core.FlexShrink` is the only door
into the field. `Style.ShrinkFactor()` is the single reading, and the sentinel
crosses into three runtimes as a bare number because `core.Style` carries no
JSON field tags — which is what rules out the other answer (omit the key, let
absence mean unset).

Honoured on three targets and pinned across all four boundaries:

```
   htmlout        writes flex-shrink:0                    Go test
   WASM runtime   writes flexShrink "0"                   .mjs suite + regexp pin
   SwiftUI        GrMobFlexSolver takes per-item factors   ios/verify solver cases
   Compose        nothing — a Row has no proportional shrink to honour it with
```

**The Swift half had been waiting for this.** `GrMobFlex.swift`'s own comment
said `flex-shrink: 1` was "the only value the Go DSL's renderers honor". Its
shrink arm is now CSS's scaled-base rule; with every factor at 1 the factors
cancel, so no existing case moved.

**Item 12 turned into a measurement.** The sticky fixture's `FlexShrink: 0` was
the same inert zero. It works now — and the List is **400px in a 160px port
either way**, measured. A flex item's automatic minimum size is content-based
and those rows carry text, so the declaration never was the reason the fixture
overflows and could not be. The check now asserts the arrangement itself rather
than one of the mechanisms that could produce it.

### Item 7 — whether any real screen can produce a nested composite

**Nothing in `components` declares a composite *container* role.** Not one
widget. `RoleOption` is set by `ListRow`, `RoleTab` is documented for a
`SegmentedControl`'s segments — those are member roles, and being a member is a
fact about a node's parent. Every listbox, tablist and toolbar is a container
the caller built and roled themselves, with the recipe published in the widget's
doc comment.

So a nested composite needs an author to declare **both** container roles by
hand. Reachable, and not something composition falls into — which is a different
thing from contrived, and is why the five descending pairs are worth their
reasoning: the author who writes that tree wrote two deliberate declarations and
will read the finding that names them.

Parsed rather than grepped, because every one of those roles is named in a doc
comment in this package and a substring search would report each recipe as a
declaration.

### Item 8 — an anchor nobody bounded

The *cut* was syntactic; finding the declaration was `strings.Index`. Every
declaration in these renderers carries a doc comment and several name their
neighbours, so an anchor could match a mention and hand every
`strings.Contains` a paragraph of English.

The rule now: the anchor must match at the start of a line, **in code** — both
halves, because the line-start test alone still admits a block-comment
continuation line and the code test alone still admits a mid-line mention. Two
matches are refused rather than resolved silently, which is what a prefix anchor
(`GrMobColumn` vs `GrMobColumnHeader`) looks like.

**The first fixtures were catchable by one half alone**, so neither break-test
broke. Rewriting them — a block comment whose continuation line begins in column
one, a `"""` literal doing the same — made each half observable.

### Item 9 — two empty answers

`CompositeMemberRole` returned `""` for a toolbar (a keyboard, no member role
ARIA names) and for a `RoleHeading` (neither). The doc said callers separate them
by asking `KeyboardComposites` first, which a doc cannot enforce and which the
one caller inside core got right by never being handed a non-composite —
`CompositeWalkStopsAt(RoleHeading, RoleTabList)` returned a confident answer
about a walk that does not exist.

Comma-ok now, and the flag is held to `KeyboardComposites()` in both directions
over every declared role.

### Item 10 — two gates nobody could reach

Both were inline conditions whose arms needed a machine with the fault. Each is
now a shell function of values with a `gate_test.sh` beside it, run by `run.sh`
before the pass — the same move `startupVerdict` and `localCopyGate` made.

**Extracting the Android one found the order was wrong.** The script asked for a
`kotlinc` first and checked for a `java` afterwards. `kotlinc` is a JVM
application: on a machine with a compiler and no JDK the pass did not skip, it
ran `kotlinc`, which failed, and `set -e` turned an absent optional toolchain
into a red pass.

The iOS gate gained a distinction on the way past —
`[ -n "$sdk" ] && [ -d "$sdk" ]` is two machines with two remedies (no Xcode, or
an `xcrun` naming an SDK that is not there) and told them apart for nobody.

## The break-tests

**31 caught, 3 misses, 0 restore failures.**

    item 1     wrapper stops stretching, wrapper stops growing, control grows,
               row insets transposed, a leading inset left on the Row       5
    item 2     a widget stops merging its base, a field with no witness      2
    item 4     the band's vertical insets zeroed, chevron/label swapped      2
    item 6     the two axes transposed, a child that fits, a container that
               is not fixed-size, Compose moves to requiredSize, SwiftUI
               clips                                                        5
    item 5     Android grows its own distributor                            1
    item 3     a carried type in neither table, a bridge fn with an
               unspelled type, a []byte bridge fn, a kind this file cannot
               name, a type classified both ways                            5
    item 11    the runtime discards a zero factor (caught at three layers)   3
    item 7     a widget declares a container role, a comment must not count  2
    item 8     the mask dropped, the line-start test dropped                 2
    item 9     a non-composite claims a keyboard, the toolbar drops off
               KeyboardComposites                                           2
    item 10    the Android gate's order reverted, the iOS gate's two skips
               collapsed, the iOS gate runs with no SDK                     3

Three mutations are invisible and each is worth stating:

- **`CompositeWalkStopsAt`'s two arms return the same value.** A non-composite
  and a toolbar both stop; what differs is the reasoning, and the reasoning is
  what the next caller reads. Making it observable would need a third return,
  and the observable half of the fix is at `CompositeMemberRole`, which is where
  the item pointed.
- **The sticky fixture's shrink factor.** Removing it changes no pixel, and that
  is now a measured fact rather than a suspicion — see item 12.
- **Reverting the mask in `swiftDeclIndex` and the line-start test each fail via
  the ambiguity refusal** rather than via the cut: a mention becomes a second
  "declaration". Correct, and it means the two halves are not distinguishable
  from each other by their failure messages.

## Verification

Eight paths, green:

    gofmt / go vet / go build / go test ./... / go test -race ./...
    wasm/verify/run.sh          replay + 18 mjs suites + BROWSER PASS
                                (20 swatches, 6 widgets, a sticky band,
                                 22 value ranges, 3 bands x 2 arrangements x
                                 4 offers, 9 rendered bands, 4 fixed-size boxes)
    ios/verify/run.sh           the gate's own tests + flex + stack + 3 band
                                arrangements + 15 picker menus + replay + view
                                + app
    android/verify/run.sh       the gate's own tests + 15 picker menus and
                                22 value ranges, on a JVM
    android ./gradlew compileDebugKotlin --offline
    GOOS=js GOARCH=wasm go build ./...

New files:

    core/componentdefaults_test.go      the witness per ComponentDefaults field
    core/shrink_test.go                 the sentinel through prop and merge
    htmlout/fixedsize_test.go           the static exporter's row in the census
    components/nested_composite_test.go no widget declares a container role
    mobile/verify/fixedsize_test.go     the two native fixed-size call sites
    mobile/verify/banddistribution_test.go  why the band's census has three rows
    wasm/verify/shrink_test.go          the sentinel across three languages
    wasm/verify/shrink_test.mjs         the runtime's four answers
    ios/verify/gate.sh, gate_test.sh    the app-layer gate and every answer
    android/verify/gate.sh, gate_test.sh the JVM gate, including the order

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here.

1. **(age 2 · value low) The two shrink controls in the band check cannot guard
   their own deletion.** Dropping the control *and* `min-width: 0` together
   leaves both arrangements at their natural width, agreeing by never reaching
   the arithmetic. The same is true of the scroll control in check 3 and the
   scroller in check 6 — either an argument that this is what a control is, or
   an argument for one source-level pin over the three.
2. **(age 2 · value low) The band check mounts only definite offers and
   `max-content`.** Reading `bandfixture`'s negative offer as the CSS
   `max-content` is a judgement stated in a comment and held to nothing;
   `min-content` is the other candidate and a different question.
3. **(age 2 · value low) `browser.mjs`'s check numbering is not a sequence.**
   The header lists eight and the body has two sections numbered 6 — and now
   two more, 9 and 10, added to a broken sequence.
4. **(age 0 · value high) `core.FlexShrink(0)` is expressible and Compose
   cannot honour it.** Three targets shrink proportionally and take a factor;
   Compose's `Row` has no proportional shrink at all, so "do not shrink" is a
   declaration that means something on three targets and nothing on the fourth.
   That is a *stated* divergence now rather than a hole, and it is the biggest
   remaining gap in the flex vocabulary's portability.
5. **(age 0 · value medium) `CompositeWalkStopsAt`'s two stopping arms are
   indistinguishable from outside.** A non-composite outer and a toolbar both
   return true, so the separation is real in the code and invisible in the
   answer. A caller who wants to know which they have still cannot.
6. **(age 0 · value medium) The band's Compose row is derived and cannot be
   pinned.** `foundation-layout`'s sources are not in the cache at the version
   the BOM resolves, so the reading is of a version this build does not use. A
   `gradle -q dependencies`-derived version constant plus a cached sources jar
   would make it a pin; fetching one is a network call no harness here makes.
7. **(age 0 · value medium) Nothing measures the four targets' fixed-size
   answer on the two natives.** The DOM row is measured and the SwiftUI row is
   derived — but `GrMobFlexSolver` is ours and `ios/verify` executes it, so the
   SwiftUI main-axis column could be measured rather than reasoned about.
8. **(age 0 · value low) `gobindCarriesUnused` refuses six types whose
   parameter spellings are legible.** Only the `(value, error)` out-pointer
   spelling is unreadable, so a bridge function taking an `int8` and returning
   nothing is refused for a reason that does not apply to it. Splitting the
   refusal by position would admit those.
9. **(age 0 · value low) The rendered-band grid mounts nine trees and reads no
   pixels.** Every other browser check that mounts real widgets samples a
   colour; this one measures rects only, so a band that laid out correctly and
   painted nothing would pass.
10. **(age 0 · value low) `maskSwiftNonCode` and `matchingBrace` answer the same
    question twice.** They share "what in this file is code" and are two
    separate scanners with the arms written in the same order — which is a
    comment asking them to agree rather than a mechanism making them.
11. **(age 0 · value low) The two `gate_test.sh` scripts share a shape and no
    code.** Each has its own `expect` helper, and a third harness's gate would
    write a third. Small, and the kind of thing that is easier to unify at two
    than at four.
