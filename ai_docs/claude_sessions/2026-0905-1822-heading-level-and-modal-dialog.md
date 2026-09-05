# Session: the heading level, and the role a Modal already was

Session: https://claude.ai/code/session_01AM7rDTyaXrFQ4X6rPDdUWp
Date: 2026-09-05 (items 2 and 3 from "chip-prominence"; item 1 had landed in
`cb87366` before this session opened)

## Ask

"Let's do the heading-level question and Remaining: the heading-level and
core.Modal carrying dialog." Both were friction the first downstream consumer
reported when it adopted roles, and both were deliberately deferred then —
the note in `roles-seen-from-the-consumer` was "not worth a field until
something has three, but the shape of the fix is worth deciding before a
consumer invents one."

## Part 1 — `AccessibilityHeadingLevel`

### The problem, restated

A screen with an `AppBar` title over a run of `GroupedList` month bands has two
tiers and announces a flat list of peers. A reader navigating by heading cannot
tell the screen's name from a band inside it. The vocabulary had `heading` and
nothing to say where a heading sits.

### The decision: an int beside the role, not `RoleHeading2`

The alternative was six more `Role` constants. Three arguments against, and the
first is decisive:

**`core.Role`'s values are ARIA's own spellings, which is the entire reason the
two DOM renderers need no mapping table.** There is no `role="heading2"`. The
moment the enum carries one, both web targets need exactly the table the
vocabulary was chosen to avoid — and that choice is load-bearing enough to be
the first section of `role.go`.

**"Every renderer names every role" would cost twelve new arms** — six per
native — every one of which maps to the heading primitive the plain `heading`
arm already maps to. The coverage checks would be pinning six spellings of one
fact.

**Level and role are independent questions.** A reader asks "what is this" once
and "where does it sit" separately, and every caller that existed before this
field should stay correct by leaving a field alone rather than by picking from
a lettered set.

### The two rules, both decisions rather than defensive coding

**Read only alongside `RoleHeading`.** That is ARIA's own scoping: `aria-level`
is defined for `heading`, `listitem` and `row`, and notably *not* for
`columnheader` — which is why `DataTable`'s column headers take the role and no
tier, and why the guard is spelled identically in all three renderers that can
express a level.

**Out of range is dropped, not clamped.** `0` is the zero value and means "a
heading, tier unstated", which is what every heading in every tree was before
the field existed. A `7` has no spelling on any target, and rewriting it to a
`6` would export a structure the caller never described. Pinned in Swift by
requiring the `default: return .unspecified` arm, not just the mapping.

### What each target does with it

    HTML / WASM   aria-level
    iOS           .accessibilityHeading(.h1 … .h6)
    Android       nothing — Compose's heading() takes no argument and the
                  semantics package has no level property at all

The Android gap is the interesting one, because "nothing to become" and "nobody
read the key" render identically on device. It is written down in a
`GrMobStyle.kt` doc section — *"AccessibilityHeadingLevel is not read here, and
cannot be"* — and pinned two ways by `mobile/verify/heading_level_test.go`: the
paragraph must be present, *and* the file must not parse the JSON key, so the
note and the behaviour cannot drift apart. Same rule the eleven unmapped roles
already live under.

### The SwiftUI detail worth recording

`grMobRole` carries the level rather than a step of its own, and applies
`.accessibilityHeading(...)` unconditionally with `.unspecified` as the identity
case. A `@ViewBuilder` branch would add another `_ConditionalContent` layer to
`grMobBox`'s opaque-type tower — the thing `grMobTransition`'s comment says
crashes the Swift compiler ("non-terminating conformance substitution"). Both
arms of the existing `guard` take the same two modifiers so the return types
match. `ios/verify/run.sh` type-checks the view layer, which is what proved it.

### The two-tier case is fixed with no call site

`AppBar`'s title takes level 1 — an AppBar is the screen's own bar, so there is
nothing above it for it to be a section of. `GroupedList`'s band labels take 2.
Both widgets already asserted `RoleHeading` without asking, so the tier is the
same kind of decision one layer on, and the exact pair the consumer reported is
fixed for every app without anyone adopting anything.

A banded feed with no bar starts at 2 with no 1 above it — a soft lint on the
web, nothing at all to either native — which is the lesser of the two wrongs
against announcing a screen's name and its March band as peers. Written into
`grouping.go` so the next person to notice finds the argument.

## Part 2 — `core.Modal` announces as a dialog

### What was wrong

`confirmDialog` downstream announced as a container with a heading inside it.
Both natives were already fine: iOS presents a Modal as a sheet, Android as a
Compose `Dialog`, and both announce themselves and confine exploration. The two
DOM renderers drew a plain `div`. The overlay was the one target where a dialog
was not a dialog.

### The decision: no `RoleDialog`

The role is written by the **node type**, as part of the Modal chassis, beside
the fixed-overlay rules — `role="dialog"` plus `aria-modal="true"`. This is
Button's arrangement one layer down: a `core.Button` is a `<button>` and nobody
says `button`, and `RoleButton` exists for the *other* case, a tappable Box.

Against a vocabulary entry:

- It would hand the author work three of the four targets already do unasked.
- It would cost two native arms that could only be empty — and in this
  vocabulary an empty arm means *"this platform cannot say it"*, which is the
  opposite of the truth here. Nine roles are honestly inert; this one would be
  a lie about the natives.
- `aria-modal` is not expressible through `Role` at all. It is a second
  attribute, and a Modal is the only node in the framework that knows the rest
  of the screen is inert behind it.

There is also nowhere for a role to ride: `core.ModalNode` has no `Style` field,
so `node.Style` is nil for every dialog `core.Modal` builds. A prop-carried role
would have required a new field on the node type to hold something the type
already knows.

### The three precedence rules, each borrowed from something already settled

**Author's role wins.** A hand-built Modal node that states its own role means
it. Same precedent `modalChassis` sets for style — its declarations go first so
the node's own style outranks them. `aria-modal` survives whatever role they
chose, because it was never theirs to replace.

**Hidden wins over both.** An overlay pruned from the accessibility tree has no
element for `role="dialog"` to describe, and `aria-modal` on it would claim the
document behind it is inert for something no reader can reach. Same exclusive
choice `aria-hidden` already makes against a name and a role.

**A closed modal needs no special case.** `modalChassis` gives it
`display:none`, which takes the element and its subtree out of the accessibility
tree, so the claim never reaches a reader it could mislead.

### The knock-on nobody would have looked for

A Modal is a generic `<div>` with a nil `Style`, so it slipped past both of
`tabPanelBox`'s existing guards — the tag check and the authored-role check —
and a Modal used as a TabView page would have been given `role="tabpanel"` on
top of the role its type had just written. One attribute, two values, invalid
markup.

Closed by `htmlout.CarriesOwnRole`, and pinned **by construction** rather than
by naming "Modal" twice: for every node type the tag table knows, a bare node of
that type either exports a role attribute or does not, and `CarriesOwnRole` has
to agree. A future self-roling type gets the guard for free or fails the test.

The WASM runtime got this for nothing — `canBeTabPanel` reads the role back off
the live element, and `dialog` is neither `null` nor `"tabpanel"`.

### The WASM totality trap

`applyAccessibility` is total on purpose: an update-style patch carries the
whole new Style, so a field at its zero value means "unset now", and every
attribute is set-or-removed on every call. Writing the dialog role only in
`createElement` would have had the next style patch strip it.

So both routes end in the same function — `applyAccessibility(el, style,
nodeType)` — with `createElement`'s Modal branch calling it with
`node.Style || {}` because the `applyStyle` path never runs for a Style-less
node. They cannot drift.

## Verification

Upstream: `go build`, `go vet`, `gofmt`, `go test ./...`, `-race`,
`wasm/verify/run.sh`, `ios/verify/run.sh`, `GOOS=js` build — all clean.

New tests:

    core/heading_level_test.go        merge is independent of the role (a theme
                                      can set the tier a widget names the role
                                      for); zero is unset, not level zero; the
                                      prop invents no role
    htmlout/export_test.go            all six levels; five drop cases including
                                      columnheader; hidden beats level; Modal is
                                      a dialog; authored role wins; hidden beats
                                      the modal pair; CarriesOwnRole agrees with
                                      the export for every node type
    wasm/verify/a11y_test.mjs         ten behavioural tests against the minimal
                                      DOM, including the two that matter most:
                                      a level that goes away takes its attribute
                                      with it, and a Modal's semantics survive
                                      an update-style patch
    wasm/verify/a11y_test.go          the same contract as source pins, because
                                      run.sh needs Node and `go test ./...` does
                                      not
    mobile/verify/heading_level_test  Swift parses, maps, and applies; the role
                                      guard and the .unspecified catch-all;
                                      Kotlin documents the gap and does not
                                      parse the key

Existing role tests in `app_bar_test.go` and `grouped_list_test.go` gained the
level assertions rather than getting new tests, since they are the same claim.

No goldens moved — this repo has none containing `role="heading"`; the
downstream app's fifteen will gain `aria-level` on their bar titles and band
labels, and `role="dialog"`/`aria-modal` on every Modal.

## Docs

`docs/concepts/styling-and-theming.md` gained two sections —
`AccessibilityHeadingLevel` (with the target table, the two rules, and the
argument against lettered constants) and "Roles a node type carries for itself".
`docs/components.md` states the 1/2 pair on both widgets.
`docs/platforms/native.md` states the iOS/Android split and why the natives need
no dialog role. `docs/platforms/exporters.md` and `wasm.md` carry the exporter
and tab-wiring halves; wasm's opt-out table went from four rows to five.
ROADMAP has two new checked lines.

## Next

Carried from the last doc, minus items 2 and 3:

1. The `<button>` user-agent border divergence on the two DOM renderers
   (`EmphasisGhost` draws a rule on the web and not on the natives).
2. `Calendar.Deselectable` and a counted `Marked` (carried over).
3. Small: "emit `OnEndReached` before the children" in its doc (carried over).
4. Small: the mid-list busy case in `EmptyState`'s doc (carried over).
5. Then Tier C: heading plumbing + `Rotate` + Compass.
6. An "on-light" tone per palette role, which is what both Button's outlined
   treatment and `Chip.ProminenceLoud` are working around (carried over).

Newly raised here, unranked:

- **No `tab` / `tablist` role**, still. Untouched by this pass; it needs a
  *state* (`aria-selected`), which has no home on `Style` and which both natives
  spell differently again.
- **ARIA's `log`** for a chat transcript. `RoleStatus` is the nearest thing and
  is not the same promise.
- **Heading level 6 is reachable and nothing in the framework goes past 2.**
  Worth watching whether a third tier ever appears, or whether the field's range
  is wider than the framework will ever need.
- **`aria-level` on `listitem` and `row`.** ARIA defines it for both, and the
  field is deliberately named for the one role it serves. If a nested list ever
  wants depth, the question is whether to widen this field or add a second one —
  the current name makes the second the honest option.
