# Session: the border a Button never drew, and aria-level's other two roles

Session: https://claude.ai/code/session_01AM7rDTyaXrFQ4X6rPDdUWp
Date: 2026-09-05 (item 1 from "heading-level-and-modal-dialog", plus the
`aria-level` on `listitem`/`row` item from that doc's unranked list)

## Ask

"Let's do the button border divergence and the aria item in the next list."
Two items in that list start with ARIA, so the second was pinned by asking:
`aria-level` on `listitem`/`row`, not ARIA's `log`. The shape was also asked
— the doc had left "widen the field or add a second one" open — and the answer
was **add a second field**.

## Part 1 — the Button border, which was wrong in both directions

### What was recorded, and what was actually there

The carried item said: a `<button>` keeps the user agent's default border on
both DOM renderers, so `EmphasisGhost` — "outlined without the rule" — draws a
rule on the web and none on the natives. True, and half the story.

Reading `GrMobButton` on both natives turned up the mirror image. A Button is
one of the few node types that draws its own container, so each renderer hands
it a style stripped of the box-drawing fields — Compose's `marginAndSize`,
SwiftUI's `marginAndSizeOnly`, both of which clear background, border, radius,
shadow and padding — and feeds them back through the platform control's own
slots. Background, radius and padding were fed back. **The border was stripped
and never fed back**, so `core.BorderColor`/`BorderWidth` were silently dropped
on Buttons alone and `EmphasisOutlined`'s documented 1px rule did not exist on
either phone.

So `EmphasisOutlined` and `EmphasisGhost` — two looks whose entire difference
is the rule — were rendering identically on device *and* identically on the
web, in opposite ways. Both halves are the same disagreement: whether "a width
**and** a color" decides the border on four targets or on two.

### The web half: a third value for `border`

`border` is now the one property with three values rather than two: the styled
declaration, `""`, and `"none"`. The reset is in the **else arm of the same
expression**, not a guard of its own, because `styleFromGrMob` and `styleValue`
are total — an update-style patch carries the whole new Style, so a guarded
write would leave the old border standing on a button that stopped having one,
and an unguarded `""` hands the element back to the user-agent stylesheet,
which is the bug rather than the fix.

Where the knowledge lives: `borderResetTags` in `htmlout/tag.go`, restated as
`BORDER_RESET_TAGS` in the runtime and pinned by
`TestRuntimeBorderResetTagsMatchGo` — the treatment `tags`, `inputTypes` and
`genericTags` already get. A set rather than `tag == "button"`, because the
question is per-tag and the membership will change if the theme question below
is ever settled.

**`<input>` and `<textarea>` are deliberately excluded**, and this is the part
worth remembering. They have a UA border too and the same argument would remove
it — but neither bundled theme gives `Components.Input` a `BorderColor`, so the
reset would leave every text field on the web as an unmarked rectangle. The
natives already have that problem (Compose uses a bare `BasicTextField`,
SwiftUI a `.plain` textFieldStyle, both drawing only what the style asks for),
so the honest fix is a border in the themes' Input style — a palette decision,
not a renderer one. Until then the web is the target that happens to be right,
and taking its border away would be levelling down.

### The native half

Compose: a `BorderStroke` into material3's `border` slot — *and* into the
`Surface` that `GrMobLongPressButton` rebuilds the button out of, because an
outlined button must not lose its rule the moment it grows an `OnLongPress`.
The test counts two occurrences rather than finding one, since one match is
exactly the divergence that would be hardest to notice.

SwiftUI: `GrMobButtonStyle` gained the pair and applies `.grMobBorder` on the
same `RoundedRectangle` the fill is clipped to. That required dropping
`fileprivate` from `grMobBorder` in `GrMobStyle.swift` — Swift's access control
is per file, so the alternative was a second copy of the `strokeBorder` logic,
which is a second border rule to keep in step with Compose's. The test pins
against the file-private spelling coming back, because *that* would break the
build loudly while a copy would not.

`strokeBorder` on both: it insets the stroke inside the shape rather than
straddling the edge, which is `Modifier.border`'s placement and what `grMobBox`
already does everywhere else.

## Part 2 — `AccessibilityNestingLevel`

### The decision: a second int, not a widened first one

Both fields become `aria-level`, so one `AccessibilityLevel` reading all three
of ARIA's roles was the obvious alternative. What it cannot carry is that the
two are validated differently, and not by accident:

    heading   1-6        HTML has h1-h6, SwiftUI has .h1-.h6; a 7 has no
                         spelling on any target that can express a tier
    nesting   1 and up   ARIA asks only for "an integer greater than or equal
                         to 1"; depth 7 is not malformed

One field needs one rule and either rule is wrong for the other half: capping a
depth at 6 flattens a legitimate tree, lifting the heading cap exports a tier
nothing can honor. The names are the other half — a caller reaching for "the
level" on a list item should not have to read a doc to learn the field is
spelled for headings.

### One attribute, two writers, and why that is not a precedence rule

The two could have been two functions each guarding on its own roles, which
would leave one attribute slot with two writers and a rule about who wins.
Instead `headingLevel` became **`ariaLevel`, a switch on the role**, in both
DOM renderers. A node has one role, the arms are disjoint, and no arrangement
of the two fields can produce two values for one attribute. Setting both is not
an error and needs no rule: whichever the role does not name is not read.

The sharpest case is only visible live, and `a11y_test.mjs` covers it — an item
that keeps both fields and changes only its role has to *swap* which one is
written, not keep the value the previous role selected.

### Both natives are inert, and this time that is the whole story

The heading tier splits the platforms (SwiftUI carries it, Compose cannot), so
half of it is provable by finding a mapping. This field maps to nothing on
either: SwiftUI has no nesting-depth property at all, and Compose's nearest one
— `collectionItemInfo` — states an item's index and span *within one*
collection rather than its depth *within nested ones*, which is a different
claim; filling it from this field would tell TalkBack something the app never
said.

So there is nothing to pin but the two things that keep silence honest: the
note saying the gap is deliberate, and the absence of a parse that would
contradict it. Both notes live where a reader would look for the mapping —
`GrMobStyle.kt` beside the role dispatch, `GrMobStyle.swift` beside
`grMobHeadingLevel`, which is where someone who has just seen the heading third
mapped will ask about the other two.

### No consumer, and that is stated

Nothing in the framework sets one. `RoleList`/`RoleListItem` have no in-repo
user at all and `DataTable`'s `RoleRow` rows are flat, so unlike the heading
pair — which `AppBar` and `GroupedList` supply for every app with no call site
— this is a prop an application reaches for when it builds the nesting itself.
Written into the prop's doc so the next reader is not left looking for the
widget that uses it.

## Verification

All five paths, green:

    go build / go vet / gofmt / go test ./... / go test -race ./...
    wasm/verify/run.sh          replay + 6 mjs suites
    ios/verify/run.sh           data layer + view-layer typecheck
    android ./gradlew compileDebugKotlin --offline
    GOOS=js GOARCH=wasm go build ./...

The Kotlin compile is new here — previous sessions pinned Kotlin by source text
only. It caught nothing, but it is the check that would have caught the smart
cast `borderStroke` was first written with (`s?.borderColor ?: return null`
does not smart-cast `s`), which was rewritten to an explicit null check before
the compile ran.

New tests:

    core/nesting_level_test.go        merge is independent of the role AND of
                                      the other level, in both merge orders;
                                      the prop invents no role and does not
                                      write the heading tier
    htmlout/export_test.go            four border tests (ghost resets, a real
                                      border is not swallowed, half a border
                                      resets, and a walk of the whole tag table
                                      against ResetsUABorder); four level tests
                                      (depths with no ceiling on both roles,
                                      six drop cases, the role deciding which
                                      field is read with exactly one attribute
                                      written, hidden beating a depth)
    wasm/verify/border_test.go        the set conformance + a source pin that
                                      the reset is in the false arm
    wasm/verify/border_test.mjs       five behavioural, including the one a
                                      static export cannot have: a button that
                                      loses its border does not get the
                                      browser's back
    wasm/verify/a11y_test.mjs         seven more, including the role-swap case
    mobile/verify/button_border_test  both natives consult the border; the
                                      Kotlin guard; grMobBorder is not
                                      fileprivate again
    mobile/verify/nesting_level_test  both notes present, neither parse present

`TestRuntimeGuardsTheHeadingLevelTheSameWay` became
`TestRuntimeGuardsTheLevelsTheSameWay` and now pins the dispatch, all four
arms and both ranges.

No goldens moved — this repo has none. Downstream, every button on both web
targets loses 4px of chrome and outlined buttons gain their rule on device.

## Docs

`docs/components.md` states the closed divergence under Button's emphasis
table. `docs/concepts/styling-and-theming.md` gained the border rule beside the
box-model paragraph and a full `AccessibilityNestingLevel` section.
`docs/platforms/native.md` gained "A `core.Button`'s border" and
"`AccessibilityNestingLevel`". `docs/platforms/exporters.md` gained three
bullets. `docs/platforms/wasm.md` gained "The user-agent border, and the third
value totality needs" and "One attribute, two level fields". ROADMAP has two
new checked lines.

## Next

Carried, minus the two done here:

1. `Calendar.Deselectable` and a counted `Marked`.
2. Small: "emit `OnEndReached` before the children" in its doc.
3. Small: the mid-list busy case in `EmptyState`'s doc.
4. Then Tier C: heading plumbing + `Rotate` + Compass.
5. An "on-light" tone per palette role, which both Button's outlined treatment
   and `Chip.ProminenceLoud` are working around.
6. **No `tab` / `tablist` role.** Needs a *state* (`aria-selected`), which has
   no home on `Style` and which both natives spell differently.
7. **ARIA's `log`** for a chat transcript. `RoleStatus` is the nearest thing
   and is not the same promise.
8. **Heading level 6 is reachable and nothing in the framework goes past 2.**

Newly raised here:

- **The themes give `Components.Input` no border.** This is the reason
  `<input>` and `<textarea>` are excluded from the UA border reset, and it is a
  real gap in its own right: a text field on both natives is drawn only by its
  background, so on a white surface it is invisible on device today. The fix is
  a border in both bundled themes' Input style, which would then let the reset
  set widen to every tag the browser draws on and make the rule uniform. It is
  a palette decision, so it is recorded rather than taken.
- **`AccessibilityNestingLevel` has no consumer.** It ships ahead of one,
  deliberately. Worth watching whether the first nested list downstream reaches
  for it or invents something else — if it invents something, the field's name
  or its scoping is wrong.
- **A `<select>` will need the same treatment** if `core.Switch` or a picker
  node type ever lands: it is a form control with a UA border, and it would
  join `borderResetTags` or be excluded for the Input reason, not by default.
