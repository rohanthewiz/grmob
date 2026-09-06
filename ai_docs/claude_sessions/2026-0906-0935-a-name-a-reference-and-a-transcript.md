# Session: a role that makes a name audible, a reference that finds a panel, and a transcript nobody was testing

Session: https://claude.ai/code/session_01YVcrPiEYG2vnpUePWiK5T5
Date: 2026-09-06 (follows "a-stub-a-listbox-and-a-question")

## Ask

"Now do the oldest 3 items on the Next list."

The three oldest were items 1, 2 and 3: a hand-built tab strip unable to point
at its panel (age 3, low), `RoleLog` with one consumer and no test (age 3,
low), and an accessible name on an unroled container dropped on both web
targets (age 2, high).

Two of them turned out to be **the same fact about ARIA, met from opposite
ends**. A `<div>` is `role=generic`, and `generic` is a role that cannot be
named — which is why a labelled container announces nothing. It is also why the
tab item's own premise ("`Style` carries values, not references") had a hole in
it: the reason the vocabulary had no `aria-controls` was never that a reference
could not be a value, it was that every reference it had considered pointed at
*text*, which a value already carries.

Ordered smallest first: the log's missing test, the reference pair, the role.

## Phase 1 — `examples/chat` gets a test

### What the coverage checks could not see

`RoleLog` had four pins already and none of them touched a consumer:
`mobile/verify/role_test.go` holds both native dispatches against
`core.Roles()`, and `htmlout`'s `TestEveryRoleBecomesTheRoleAttribute` proves
every role becomes an attribute. Neither can see whether an app ever *asks* for
one.

That gap is the whole argument for the file. A role costs nothing to set and is
inert until something reads it, so "the one place it is used stopped using it"
compiles, renders, and shows up nowhere: the bubbles still draw, the send still
works, and a screen reader is simply no longer told that a message arrived.

### Three claims, and the one that would go silently

    the role is a log        not `status`. Same call on Android — both are
                             LiveRegionMode.Polite — so nothing on a phone
                             would report the swap. It is a web-only
                             distinction and therefore a web-only regression.
    it is on the Column      the element whose children change. The Scroll
                             around it is a viewport and gains nothing, so a
                             live region there names something that never
                             changes.
    it reaches the document  which is the target the role exists for.

The middle one is written as "the log is a Column with a Scroll above it"
rather than "the Scroll has no role", because the second passes for a tree with
no log role anywhere at all.

The file also picks up the example's own lesson — `send` as the single mutation
choke point — with the assertion that the message appears **once**, which is
what the copy-before-append prevents, plus the seeded thread surviving in
order (a log's contents are read back, so losing the history is a different
failure from failing to announce) and the composer clearing.

## Phase 2 — `AccessibilityID` and `AccessibilityControls`

### The line that keeps this from becoming a second ARIA

Half of ARIA is IDREF-shaped, and adding all of it would ask every app to mint
and track document-global ids for things there is a shorter way to say. The
rule written down instead:

> A reference prop earns its place only when what it points at cannot be said
> as a value.

`aria-labelledby` points at text and `AccessibilityLabel` carries text.
`aria-describedby` points at text and `AccessibilityHint` carries text — as
`aria-description`, which is that idea in value form and was already argued for
on exactly these grounds. Neither buys an author anything but a saved copy of a
string they are holding. `aria-controls` points at *another element*, and no
string stands in for one.

So the pair is two props and the vocabulary's only two references, with the
reason the other three are absent stated beside them rather than left as an
omission.

### Three collisions, resolved three different ways

The wiring `core.TabView` writes from the node type reaches for `id`, `role`
and `aria-labelledby`. Two of those now have a second writer.

**`id` — the author wins outright.** A page carrying an `AccessibilityID` is
left unwired on both web targets. It is the same theft rule an authored role
already follows, and worse in one way: something else on the page is pointing
at that string, so taking it would break a relationship rather than replace a
word. The runtime tells an author's id from one it wrote a sync ago by
comparing against `panelId(scope, i)`, which is the discrimination the role
does one line up.

**`role` — the wiring wins, and `htmlout` had to be told.** `renderNode` now
asks `imposesRole(from.attrs)` and skips its own. HTML gives an element one
value per attribute name and a browser keeps the *first*, so a second `role=`
would not be additive — it would decide the panel's role by document order.

**`aria-labelledby` — unchanged.** No `core.Style` field maps onto it, which is
the whole point of the rule above.

### The one role it is not theft to replace

`core.RoleGroup` is exempt from `tabPanelBox`'s authored-role rule, and it is
the only value that is. A group says these things belong together and this is
what they are called; a `tabpanel` says all of that *and* which tab shows it,
so writing one over the other adds a fact instead of destroying one — which is
exactly the test every other opt-out applies.

It also *has* to be exempt, and that is the load-bearing half: phase 3 supplies
a group to any named page whether the author asked or not, so treating it as an
authored role would silently stop every page carrying an `AccessibilityLabel`
from being wired at all. The runtime's `canBeTabPanel` accepts it for the same
reason, and its unwire path now restores `group` on a named page rather than
clearing the role — a page that stops being wired must not fall back into the
silence the group exists to close.

### What was deliberately not added

`RoleTabPanel`. The vocabulary's stated reason for its absence — a panel is one
end of a relationship whose other half is an IDREF a `Style` cannot carry —
*dissolves* with this change, and the constant is now defensible. What still
stands against it is narrower and mechanical: the WASM runtime tells its own
wiring apart from an author's role by the value `"tabpanel"` not being a
`core.Role`, with `TestNoRoleCollidesWithTheTabPanelWiring` holding the
vocabulary to it. Replacing that with a marker attribute is the right fix and
is a change to a load-bearing invariant rather than a constant, so it is on the
Next list with its reason rather than done in passing.

### Both natives, and the near miss that is a trap

Neither key is parsed, and the notes say why in the file where the next person
looks. There is no such relationship in either semantics vocabulary and neither
reader wants one: VoiceOver and TalkBack move through a screen by swiping to
the next element, not by following a reference.

The interesting part is what each note *turns down*. `accessibilityIdentifier`
(iOS) and `testTag` (Compose) look exactly like the mapping for
`AccessibilityID` and are neither accessibility properties nor exposed to
either reader — they are what XCUITest and Espresso select by. Mapping onto
them would quietly turn every hand-built tab into a test handle and still
announce nothing. `mobile/verify/idref_test.go` pins the notes, the absence of
a parse, *and* that each note still names the property it is refusing —
backticked, because a bare `testTag` also matches `testTagsAsResourceId` in the
sentence after it, which is a weak pin that passed a break-test it should have
failed.

### Adopted downstream twice

`examples/social`'s bottom bar was the app the gap was noticed in and had no
ARIA at all: three ghost buttons, a `core.Match`, and nothing saying any of
them belonged together. It now states the whole strip — `RoleTabList` on the
Row, `RoleTab` and `AccessibilitySelected` on every button (both values), a
name per glyph because an emoji otherwise announces as its dictionary entry,
and `AccessibilityControls` pointing at a `Box` wrapped around the `Match` that
carries the `AccessibilityID` and a name that follows the tab.

The tutorial's 4.5 lesson demos the same hand-assembled shape and now teaches
it. Its test asserts the half with no visible effect, which is the half a
future edit deletes without noticing.

One name for both ends, in both cases: a typo in an IDREF is not an error
anywhere, it is a tab announcing a region that does not exist.

## Phase 3 — `core.RoleGroup`

### The silence, and why it shipped

Every layout node exports as a `<div>` or a `<span>`. Both carry the implicit
ARIA role `generic`, ARIA lists `aria-label` and `aria-labelledby` among the
attributes prohibited on it, and browsers enforce that by pruning the name out
of the accessibility tree. So:

    core.Box(core.AccessibilityLabel("Unread messages"), …)

was read out perfectly by VoiceOver and TalkBack — both honour a label on any
node without asking what it is — and announced by nothing on either web target.
Two targets fine, two silent, and the two that worked are the two a developer
tests on.

It hit every `ListRow` with a label, every `Accordion` header, every named
`StatTile`, `Skeleton` and `ProgressBar`, and `FormField`'s required marker.
`RoleImg` had been closing it for exactly one widget since the compass.

### `group`, and the three candidates that lost

    region   also nameable, and a landmark. Naming six rows would put six
             entries in the screen's table of contents.
    button   claims a control, makes its children presentational (the heading
             inside an Accordion header would stop being a heading), and is a
             foreign child of any list around it — which is what ListRow turned
             it down for three sessions ago.
    img      claims the node is a picture whose parts should be hidden. True of
             Compass, false of a row that wants its contents read.
    group    nameable, not a landmark, no required children, no
             presentational-children rule. These things belong together and
             this is what they are called, and nothing else.

That "nothing else" is the recommendation. The structural rule says a claim a
container cannot keep is worse than no role at all; `group` is the one value in
the vocabulary that promises nothing about what it holds, so it can be given to
a container nobody has looked inside.

### Supplied, not asked for

The exporters write it themselves: a node with a name, no role, a generic tag
and no self-roling node type gets `role="group"`. No call site changed and no
widget was touched — `ListRow`, `Accordion`, `StatTile`, `Skeleton`,
`ProgressBar` and `FormField` were all fixed by a rule in two files.

The argument for doing it silently is that it cannot make anything worse:
before, the name was invalid ARIA that was dropped; after, it is valid ARIA
that is announced. The one thing it could disturb is a structural container's
claim about its children — but a generic div inside a `role="list"` was never a
`listitem` either, so a group there is the same foreign child it already was,
one attribute louder.

An author's own role always wins; the fallback only fills an empty slot.

### The asymmetry with the selected state, stated on purpose

`aria-selected` on an unroled node is dropped for the same reason a name is,
and it gets no equivalent rescue. That is a decision rather than a
half-finished job: `group` fits any container, so supplying it invents nothing;
there is *no* role that carries a selection and fits any container — the four
that do are `option`, `tab`, `row` and `columnheader`, and choosing among them
would be the exporter deciding what the node is. A name is a fact the author
already stated. A role is not.

### The empty native arms that mean the opposite of the usual thing

Eleven roles are inert on SwiftUI and Compose because those platforms cannot
say the thing. `group` is inert because they do not *need* it: both already
announce a label on any node, so the role that unlocks the name on the web buys
them nothing. Both arms say so, which is the distinction the whole
spell-out-every-role convention exists to preserve.

## Verification

All six paths, green:

    gofmt / go build / go vet / go test ./... / go test -race ./...
    wasm/verify/run.sh          replay + 12 mjs suites
    ios/verify/run.sh           data + view + app layers
    android ./gradlew compileDebugKotlin --offline --rerun-tasks
    GOOS=js GOARCH=wasm go build ./...

New and changed tests:

    examples/chat/main_test.go          the log on the element that grows, log
                                        vs status, the role in the document,
                                        the send path, the whitespace no-op,
                                        and a clean debug audit
    htmlout/export_test.go              the supplied group on a Box and a Text,
                                        the three guards, hidden still winning,
                                        and the IDREF pair verbatim
    htmlout/tabview_test.go             the three shared slots on one document:
                                        a named page wired without a second
                                        role, an authored id left alone, an
                                        authored group replaced — plus a ZStack
                                        proving the suppression is scoped
    wasm/verify/a11y_test.mjs           the same five facts live, including the
                                        supplied role going away with the name
    wasm/verify/a11y_test.go            ariaRole restated, and the runtime's
                                        two shared slots
    mobile/verify/idref_test.go         both notes, the property each one turns
                                        down, and no parse of either key
    components/list_row_test.go         an unroled row's name announced, a
                                        roled row keeping its role
    components/accordion_test.go        the header named, and its heading still
                                        a heading
    examples/social/app_test.go         the strip's semantics and the selection
                                        following the tap
    examples/tutorial/chapter4_test.go  4.5's wiring, which draws identically
                                        whether or not it is there

Every new assertion was confirmed to bite by breaking the thing it guards: the
log role removed and swapped for `status`, the group fallback returning nothing
and losing its tag guard, `imposesRole` forced false (which writes two role
attributes), `tabPanelBox` losing the group exemption and the id opt-out, the
runtime's `ariaRole` neutered, `canBeTabPanel` losing `"group"`, the unwire path
clearing instead of restoring, the Swift group arm deleted, `AccessibilityID`
parsed on iOS, each native's near-miss property un-named, and the social and
tutorial strips losing their `aria-controls`, their tablist role, their
`SelectedWhen` and their panel id.

One mid-session mistake worth recording: a `git checkout -- .` inside a
break-test loop reverted every edit to a tracked file. The work was replayed
from the exact scripts in the transcript, and the break-tests were redone with
per-file `cp` snapshots instead. Never `git checkout` a dirty tree to undo a
deliberate one-line experiment.

## Docs

`core/role.go`: the twenty-three-value table, the `RoleGroup` block with the
full candidate argument, and the `RoleImg` paragraph rewritten now that it is
one of two doors rather than the only one.
`core/style.go`: the IDREF pair's field doc (the reference rule, what asked for
it, the reserved `grmob-` prefix, the two near-miss native properties), and the
`AccessibilitySelected` paragraph rewritten around the asymmetry.
`htmlout/export.go`: `ariaRole` with the silence and its three guards, the
`roleImposed` channel, and `ariaSelected`'s near-miss note.
`htmlout/tabview.go` and `wasm/grmob-runtime.js`: the sixth opt-out, the group
exemption, and the restore-on-unwire.

`components/list_row.go` and `components/accordion.go`: both widgets documented
the silence and both now say which half closed and which did not.
`components/compass.go`: why `RoleImg` is still the right value there.

`docs/concepts/styling-and-theming.md`: `RoleGroup` with its own subsection, a
new `AccessibilityID`/`AccessibilityControls` section carrying the reference
rule, the role table at twenty-three, and the selected-state asymmetry.
`docs/components.md`: `SegmentedControl`'s "the panel cannot be pointed at from
here" replaced with the three-prop wiring, and `ListRow`'s suffix note.
`docs/platforms/exporters.md` and `docs/platforms/wasm.md`: the supplied role,
the IDREF pair, and the panel opt-out table grown to six rows with the group
exemption argued.
`docs/platforms/native.md`: `group` told apart from the eleven other empty
arms, and an `AccessibilityID`/`AccessibilityControls` section naming the test
selectors it refuses.

`ROADMAP.md`: two new entries and the roles entry at twenty-three.

Tutorial: **4.5 grew an accessibility half** — the relationship a hand-built
strip has to state for itself, taught on the SegmentedControl-plus-Match demo
that lesson already had.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here. Value is the payoff of doing it, not the effort: `high` means
something is being worked around today or a second consumer has arrived,
`medium` means it blocks one named thing, `low` means it is a gap nobody has
bumped into yet.

1. **(age 3 · value medium) `core` has no `aria-expanded`.** `Accordion` is the
   package's one stateful widget and its header cannot say whether it is open.
   The natural shape is `core.AccessibilityExpanded` with `SelectedState`'s
   three values, scoped by role on the web and mapped to Compose's
   `expand`/`collapse` actions. The header now has a legal role to carry it
   (`group` supports `aria-expanded`), which removes the objection that stood
   in front of it.
2. **(age 3 · value medium) `Colors.Border` is still spent as a control
   boundary by `Chip`.** A `ProminenceQuiet` chip is a Surface fill on the
   page's Background (1.09:1) inside a `Border` hairline (1.26:1), so the
   control that filters the screen is close to invisible as a control.
3. **(age 3 · value low) `appendRows` takes twelve positional arguments.** The
   band knobs travel together through both collections; a struct would make the
   next one an addition rather than a transposition risk.
4. **(age 3 · value low) `core` has no single-side padding props except
   `PaddingTop`.** An indent goes through a whole `EdgeInsets` in a `UseStyle`.
5. **(age 2 · value medium) `DefaultTheme` paints 4.02:1 on its own primary.**
   `inkOn` returns the theme's declared white on `#007AFF`, below AA for body
   text, and every filled `Button` plus the calendar's selected day spends it.
   The fix is the theme's — most likely Apple's accessible `#0040DD`, which the
   palette already carries as `PrimaryOnLight` — and it would move every filled
   button in every app, which is why it is a decision rather than a patch.
6. **(age 2 · value medium) `ZStack` has no per-child alignment.** Every layer
   centres, and a layer that wants a corner wraps itself in a box sized to the
   stack. An `AlignSelf`-shaped prop is the obvious shape and all three
   constructs have a spelling for it, but one consumer is not a vocabulary.
7. **(age 2 · value low) A `Select` cannot be grouped or disabled per option.**
   `<optgroup>` and a disabled `<option>` have no place in `SelectOption`, and
   neither native menu would read one without work.
8. **(age 2 · value low) A picker's open menu is invisible to Go.**
   Deliberate, but it means no test can drive a native picker's list, and the
   two `mobile/verify` pins read source text rather than behaviour.
9. **(age 2 · value low) A styleless node skips the border reset in the WASM
   runtime.** `applyStyle` only runs when `node.Style` is non-nil. No node
   `core` builds is ever in that state, so this is a hand-assembled-tree gap.
10. **(age 1 · value high) Neither a `listbox` nor a `tablist` has keyboard
    navigation on the web, and there are two consumers now.** The roles state
    semantics ARIA's patterns pair with behaviour: the container takes focus,
    the arrow keys move the active item, and a roving tabindex or
    `aria-activedescendant` says which. `examples/mobileapp`'s article list is
    the listbox; `examples/social`'s bottom bar and tutorial 4.5 are the
    tablists, both new this session, and all of them hand a keyboard user a
    role that claims more than the widget does. Needs a focus concept `core`
    does not have — `core/focus.go` is about putting the cursor in a named
    field. Both phones navigate by swipe and lose nothing.
11. **(age 1 · value medium) Nothing re-checks a permission on foreground.**
    A user can grant one in Settings and come back, and no platform says so.
    `hooks.UsePermission` deliberately does not (it cannot see whether its
    screen is still on top, and five screens would each fire a check per
    resume), so every consumer writes the same `UseLifecycle` pairing by hand.
    A `hooks.UsePermissionLive` — or a `RecheckOnForeground` option — is the
    obvious shape; what is missing is a rule for which screen owns it.
12. **(age 1 · value low) The Android "asked before" flag does not survive a
    process restart.** It is what separates a permanent refusal from "never
    asked", so after a restart a permanently-denied permission reads as
    `Prompt` until the next request proves otherwise — one dead button press.
13. **(age 1 · value low) A browser `Request` opens the device to answer.**
    There is no request API, so `getUserMedia` is the only thing that prompts,
    and a granted camera check has genuinely opened the camera for a moment. A
    refused one also reads as `denied` whether the user pressed Block or
    dismissed the prompt, because `NotAllowedError` does not say which.
14. **(age 1 · value low) The gomobile stub's types are unchecked.** The pin
    holds the *names* — every bindable symbol declared, nothing extra — because
    copying gobind's type mapping into a test would be reimplementing gobind. A
    wrong signature fails the Swift typecheck the moment the shell calls it, so
    what is genuinely unguarded is a stub declaration the shell never touches.
15. **(age 0 · value medium) A hand-built tab strip's panel still cannot say it
    is one.** `AccessibilityControls` closes the pointing and `RoleTabPanel` is
    now defensible — the vocabulary's stated reason for its absence was that a
    panel is one end of a relationship an author could not write, which is no
    longer true. What stands in the way is mechanical: the WASM runtime tells
    its own wiring apart from an author's role by `"tabpanel"` not being a
    `core.Role`, held there by `TestNoRoleCollidesWithTheTabPanelWiring`.
    Replacing that discriminator with a `data-grmob-panel` marker (which both
    web targets would write, as they already do `data-grmob-chrome`) is the
    right fix and unblocks the constant. Two native arms and a large doc block
    go with it.
16. **(age 0 · value medium) Three widgets now take `group` where ARIA has a
    better role.** The fallback made their names audible and stopped there.
    `ProgressBar` wants `progressbar` with `aria-valuenow`/`valuemin`/`valuemax`
    — it is already computing the percentage into its name, which is the
    workaround. `Skeleton` wants `status`, since "Loading" is an advisory that
    is replaced. `FormField`'s required marker is a glyph standing in for a
    word, which is `img`'s shape. Each is a widget change plus, for
    `ProgressBar`, a value vocabulary `core` does not have.
17. **(age 0 · value medium) Nothing catches a dangling `aria-controls` or a
    duplicate `AccessibilityID`.** Both are written verbatim and neither
    exporter can see the whole document at the moment it writes one — but a
    finished tree *can* be walked, which is exactly what `core.SetDebugMode`
    already does for cursor drift and duplicate keys. A debug-mode pass
    reporting an id claimed twice and a reference resolving to nothing would
    catch the one failure mode the pair has, and it is a failure that is
    invisible on every target rather than merely quiet on two. Both consumers
    written this session route their two ends through a shared constant
    precisely because nothing else would notice.
18. **(age 0 · value low) An `AccessibilityID` is not validated.** An id
    containing a space is invalid HTML; an empty one is written as `id=""`; one
    starting with `grmob-` collides with `core.TabView`'s minted ids and breaks
    a wiring the author never wrote. The prefix is documented as reserved and
    nothing enforces it. A drop would be silent too, so the honest fix is
    probably the debug-mode pass in item 17 rather than a guard in the
    exporters.

Read by value instead: **high** 10 · **medium** 1, 2, 5, 6, 11, 15, 16, 17 ·
**low** 3, 4, 7, 8, 9, 12, 13, 14, 18.
