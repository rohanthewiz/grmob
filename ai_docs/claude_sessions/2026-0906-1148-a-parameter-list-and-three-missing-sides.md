# Session: a parameter list that became a type, and the three sides nobody could name

Session: https://claude.ai/code/session_01YVcrPiEYG2vnpUePWiK5T5
Date: 2026-09-06 (follows "a-role-that-was-wrong-and-a-hex-that-was-two-jobs")

## Ask

"Take the two oldest items in the Next list."

Both were age 4 and both were rated low, which is why they had sat: item 1,
`appendRows` taking twelve positional arguments, and item 2, `core` having no
single-side padding props except `PaddingTop`.

Neither had a wrong premise this time. What both had instead was a **second,
larger failure hiding behind the small one the entry named** — and in the
padding case the fix turned out to remove a limitation that four separate doc
comments described as unavoidable.

## Phase 1 — `rowsSpec`, and the risk the entry named was not the one that bit

The entry said:

> The band knobs travel together through both collections; a struct would make
> the next one an addition rather than a transposition risk.

Transposition is real — `hideTrailingCount bool, sticky bool` sat adjacent and
a swap compiled — but it was already caught, by accident: each flag has a test
of its own in each widget, so crossing them fails four tests. Break-tested and
confirmed before writing anything.

**The uncovered failure was a knob wired at one call site and not the other.**
`DataTable.HeadingLevel` was never asserted anywhere. It had been forwarded
correctly since it was added, and nothing would have said so if it had not
been.

### What is positional and what is named

    appendRows(ctx, items, spec)

`ctx` and `items` stay positional because they are the call's *subject* and are
the same two arguments at both call sites. Everything that varies moved into
`rowsSpec[T]`, whose ten fields are spelled **exactly as the widgets' own
fields** — so `HideTrailingCount: g.HideTrailingCount` is checkable by eye where
the eighth positional argument was not. `Row` and `Rows` are the one divergence
(a table synthesizes its row from its resolved columns), and `Wrap` is the one
field with no widget field behind it: it is how the shared code lets DataTable
add the tap target and selection tint that GroupedList has no use for.

### The struct is not what removes the risk

Stated in the type's doc because it is the part that is easy to get wrong: a
struct does not stop `HideTrailingCount: d.StickyHeaders`. What the call site
*naming* each value removes is the eye-check problem, and what the struct
removes structurally is the next knob — a thirteenth positional argument
shifts every argument after its insertion point, a thirteenth field shifts
nothing.

### Two tests, doing different jobs

`TestRowsSpecCensus` is the reflective one, on `TestUseStyleMergesEveryField`'s
pattern: ten fields, named and typed, so a new knob fails here with a message
saying what else to go and do.

`TestBothWidgetsForwardEveryRowsSpecKnob` is the behavioural one — every knob
on at once, through both widgets, each assertion naming the field it stands
for. It is the first assertion of `DataTable.HeadingLevel` anywhere.

## Phase 2 — the three missing padding sides, and the limitation that was not one

### The entry undersold the problem

> An indent goes through a whole `EdgeInsets` in a `UseStyle`.

Longer, yes. But `UseStyle` merges `Padding` as **one comparable struct** — a
`Style` naming only `Left` replaces all four sides — so the workaround also
forced the caller to restate the three sides they did not care about, in
numbers copied out of the theme. Those copies are what a later theme edit
never reaches. The tutorial's `indentBy` said exactly that in its own comment
and then did it anyway, because there was no other option:

    Padding: core.EdgeInsets{Left: 16 * depth, Right: 16, Top: 4, Bottom: 4}

The theme's own ListRow is 8/16/8/16. The helper had been quietly overriding
the vertical padding of every outline row since it was written.

### The shorthand has to be dissolved, not ignored

`EdgeInsets` carries six fields, and every renderer resolves a side as
"explicit if non-zero, else the axis shorthand" (`htmlout.EdgeCSS`,
`edgeToCSS`, and `parseEdges` in both native files). A side prop that only
assigned its own field would inherit that rule's lossy edge: `PaddingLeft(0)`
over a theme carrying `Horizontal: 16` resolves back to 16 and quietly does
nothing — the same trap `PaddingHorizontal` was fixed for a while back.

So each side prop **settles** its axis first: push the shorthand into whichever
side is not already explicit, then clear it.

               before                    after settleHorizontal
    {Horizontal: 16}              ->  {Left: 16, Right: 16}
    {Horizontal: 16, Left: 24}    ->  {Left: 24, Right: 16}
    {Left: 16, Right: 16}         ->  unchanged

Settling is **resolution-preserving by construction** — the only sides it
writes are ones that were taking the shorthand anyway — which is the property
that made this a transformation on the `Style` value and **not a fifth copy of
the resolution rule**. No renderer changed. Not one line.

### The limitation four doc comments called unavoidable

`htmlout/edges.go`, `wasm/grmob-runtime.js`, `docs/concepts/styling-and-theming.md`
and the tutorial all named the same example as the rule's lossy edge:

> `PaddingHorizontal(16)` plus `PaddingLeft(0)` cannot ask for a zero left
> inset.

`PaddingLeft` **did not exist when that was written** — it was describing a
hypothetical. It exists now and it works: what reaches every renderer is
`{Left: 0, Right: 16}`. All four texts were rewritten to say the rule is lossy
for a *hand-built* `EdgeInsets` and that the DSL is not subject to it.

### `PaddingTop` moved and changed

It was the one side prop that existed and it wrote nothing but
`s.Padding.Top`, so `PaddingTop(0)` after `PaddingVertical(8)` left the 8. It
settles now, which makes the four consistent; the only behaviour that differs
is the case that used to silently fail. The function moved out of
`core/style.go` into `core/padding_sides.go` so the set is one thing.

`Margin` deliberately untouched — see Next.

### The two consumers converted, one on purpose and one not

`indentBy` is now `core.PaddingLeft(16 * depth)`, a `StyleProp` rather than a
`Style`, so it composes: the theme Row's vertical padding stays, and a caller
can add `PaddingRight` after it without either undoing the other. The outline
rows gained back their 8/8 vertical padding, which is a visual improvement
nobody asked for and the old comment had been apologizing for.

## The break-test that got through, and what it changed

Sixteen break-tests. **One passed that should not have**, and it is the useful
one to record.

Dropping settle's "only fill a side that is unset" guard — so it promotes the
shorthand over a side that had already superseded it — passed everything.

`TestSettlingAnAxisPreservesEveryResolvedSide` had the case
`{Horizontal: 8, Left: 20}` with `PaddingLeft(20)`. With the guard gone,
settle sets `Left: 8` — and then the prop's own write puts 20 straight back.
**The identity write masked the damage on the side under test.** The damage
always lands on the *opposite* side, where nothing writes over it.

Both orientations are in the test now, in `htmlout` and in `core`, and the
guard's doc says why. A second run caught it.

The mirror finding: settle's `if e.Horizontal == 0 { return }` is a **fast path
and not a guard** — removing it fails nothing, because writing zeros into
already-zero sides and clearing an already-clear field is a no-op. Stated in
the comment so nobody later mistakes it for load-bearing.

A third: pinning the tutorial indent needed a *measured* baseline. A first
draft asserted "the other three sides are non-zero and equal across rows",
which the old whole-`EdgeInsets` version passes — it sets 4/16/4 on every row.
The test now reads its baseline off the same lesson's listbox rows: the same
widget with no inset prop on it. Comparing against real rows and not against
literals is the point, since a test spelling the theme's numbers would be a
third copy of the very thing the conversion removed.

## Verification

All six paths, green:

    gofmt / go build / go vet / go test ./... / go test -race ./...
    wasm/verify/run.sh          replay + 12 mjs suites
    ios/verify/run.sh           data + view + app layers
    android ./gradlew compileDebugKotlin --offline --rerun-tasks
    GOOS=js GOARCH=wasm go build ./...

New and changed tests:

    components/rows_spec_test.go        the ten-field census; every knob at
                                        once through both widgets, including
                                        DataTable.HeadingLevel's first
                                        assertion anywhere; and the two band
                                        flags held apart
    core/padding_sides_test.go          each side alone; the zero clearing over
                                        explicit sides, over a raw shorthand
                                        and over PaddingHorizontal; the axis
                                        left stated per-side; the explicit
                                        opposite side surviving the settle;
                                        both orderings; and the four composing
    htmlout/edges_settle_test.go        settling preserving every resolved side
                                        in both orientations, and the zero that
                                        now really is a zero
    examples/tutorial/chapter7_test.go  7.2's box-model half: the whole-struct
                                        layer zeroing three sides, the side
                                        prop leaving them, and no shorthand
                                        left behind
    examples/tutorial/chapter4_test.go  the outline indent measured against an
                                        un-indented ListRow on the same screen
    examples/tutorial/app_test.go       nodeStyle decodes Padding, six fields
    components/data_table_test.go       the trailing-count comment repointed at
                                        the new all-knobs test

**Sixteen break-tests, fifteen caught, one caught after the test was fixed.**
DataTable dropping HeadingLevel and dropping Wrap; GroupedList dropping
Dividers and crossing its two bools; a thirteenth field added to rowsSpec;
PaddingLeft not settling; settle leaving the shorthand set; settle overwriting
an explicit side *(the one that got through)*; settleVertical's guard dropped;
PaddingLeft settling the vertical axis too; PaddingBottom writing Top;
PaddingTop reverted to its old bare assignment; settle's early return removed
*(correctly no failure — it is a fast path)*; the tutorial demo using UseStyle
for both toggles; indentBy back to the whole EdgeInsets *(passed until the
baseline was made a measured one)*; and indentBy using PaddingRight.

Break-tests were done with per-file `cp` snapshots. No `git checkout` was run.

One process note: a `cd android` for the gradle run left the shell there, and
the next break-test batch ran against a non-existent path — it failed loudly
and modified nothing, but the batch scripts now `cd` to the repo root first.

## Docs

`components/grouped_list.go`: `rowsSpec` — why a struct, what the struct does
*not* fix, why the field names are the widgets' field names, and per-field docs
pointing at the widget field each comes from. `appendRows` restated: what stays
positional and why.
`components/data_table.go`: the literal's one comment, on the field neither
widget spells the same way.

`core/padding_sides.go`: new file — what was here before and why a whole
`EdgeInsets` is not merely longer, the settle drawn as a table,
resolution-preservation named as the property that kept the renderers out of
it, and `PaddingTop`'s move and change. `settleHorizontal` carries the two
break-test findings: the early return is a fast path, the unset-side checks are
not.
`core/style.go`: `applyTo`'s box-model comment now names the whole per-side set.
`htmlout/edges.go` and `wasm/grmob-runtime.js`: the lossy edge restated as a
hand-built-`EdgeInsets` limitation, with the DSL exempted and the reason it
needed no change on either side of the wire.

`docs/concepts/styling-and-theming.md`: a new "One side at a time" subsection —
the granularity argument, the settle, the ordering, and `Margin` named as still
missing.
`ROADMAP.md`: the per-side props under the shorthand entry, with "no renderer
changed" and the margin gap stated.

Tutorial: **7.2 grew a box-model half** — the same "a layer cannot clear"
lesson one granularity up, with a two-toggle demo showing the whole-struct
layer collapsing three sides and the side prop leaving them, plus a fifth key
point. **4.2's `indentBy`** is one prop and says what it used to be.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here. Value is the payoff of doing it, not the effort: `high` means
something is being worked around today or a second consumer has arrived,
`medium` means it blocks one named thing, `low` means it is a gap nobody has
bumped into yet.

1. **(age 4 · value medium) `DefaultTheme` paints 4.02:1 on its own primary.**
   `inkOn` returns the theme's declared white on `#007AFF`, below AA for body
   text, and every filled `Button` plus the calendar's selected day spends it.
   The fix is the theme's — most likely Apple's accessible `#0040DD`, which the
   palette already carries as `PrimaryOnLight` — and it would move every filled
   button in every app, which is why it is a decision rather than a patch.
2. **(age 4 · value medium) `ZStack` has no per-child alignment.** Every layer
   centres, and a layer that wants a corner wraps itself in a box sized to the
   stack. An `AlignSelf`-shaped prop is the obvious shape and all three
   constructs have a spelling for it, but one consumer is not a vocabulary.
3. **(age 4 · value low) A `Select` cannot be grouped or disabled per option.**
   `<optgroup>` and a disabled `<option>` have no place in `SelectOption`, and
   neither native menu would read one without work.
4. **(age 4 · value low) A picker's open menu is invisible to Go.**
   Deliberate, but it means no test can drive a native picker's list, and the
   two `mobile/verify` pins read source text rather than behaviour.
5. **(age 4 · value low) A styleless node skips the border reset in the WASM
   runtime.** `applyStyle` only runs when `node.Style` is non-nil. No node
   `core` builds is ever in that state, so this is a hand-assembled-tree gap.
6. **(age 3 · value high) Neither a `listbox` nor a `tablist` has keyboard
   navigation on the web, and there are three consumers now.** The roles state
   semantics ARIA's patterns pair with behaviour: the container takes focus,
   the arrow keys move the active item, and a roving tabindex or
   `aria-activedescendant` says which. `examples/mobileapp`'s article list is
   the listbox; `examples/social`'s bottom bar and tutorial 4.5 are the
   tablists, and all of them hand a keyboard user a role that claims more than
   the widget does. Needs a focus concept `core` does not have —
   `core/focus.go` is about putting the cursor in a named field. Both phones
   navigate by swipe and lose nothing. An accordion header is a third consumer
   in a different form: a real `button` on the web and therefore already
   keyboard-operable, which *narrows* the item rather than widening it.
7. **(age 3 · value medium) Nothing re-checks a permission on foreground.**
   A user can grant one in Settings and come back, and no platform says so.
   `hooks.UsePermission` deliberately does not (it cannot see whether its
   screen is still on top, and five screens would each fire a check per
   resume), so every consumer writes the same `UseLifecycle` pairing by hand.
   A `hooks.UsePermissionLive` — or a `RecheckOnForeground` option — is the
   obvious shape; what is missing is a rule for which screen owns it.
8. **(age 3 · value low) The Android "asked before" flag does not survive a
   process restart.** It is what separates a permanent refusal from "never
   asked", so after a restart a permanently-denied permission reads as
   `Prompt` until the next request proves otherwise — one dead button press.
9. **(age 3 · value low) A browser `Request` opens the device to answer.**
   There is no request API, so `getUserMedia` is the only thing that prompts,
   and a granted camera check has genuinely opened the camera for a moment. A
   refused one also reads as `denied` whether the user pressed Block or
   dismissed the prompt, because `NotAllowedError` does not say which.
10. **(age 3 · value low) The gomobile stub's types are unchecked.** The pin
    holds the *names* — every bindable symbol declared, nothing extra —
    because copying gobind's type mapping into a test would be reimplementing
    gobind. A wrong signature fails the Swift typecheck the moment the shell
    calls it, so what is genuinely unguarded is a stub declaration the shell
    never touches.
11. **(age 2 · value medium) A hand-built tab strip's panel still cannot say it
    is one.** `AccessibilityControls` closes the pointing and `RoleTabPanel` is
    now defensible. What stands in the way is mechanical: the WASM runtime
    tells its own wiring apart from an author's role by `"tabpanel"` not being
    a `core.Role`, held there by `TestNoRoleCollidesWithTheTabPanelWiring`.
    Replacing that discriminator with a `data-grmob-panel` marker (which both
    web targets would write, as they already do `data-grmob-chrome`) is the
    right fix and unblocks the constant. Two native arms and a large doc block
    go with it.
12. **(age 2 · value medium) Three widgets take `group` where ARIA has a better
    role.** The supplied fallback made their names audible and stopped there.
    `ProgressBar` wants `progressbar` with `aria-valuenow`/`valuemin`/`valuemax`
    — it is already computing the percentage into its name, which is the
    workaround. `Skeleton` wants `status`, since "Loading" is an advisory that
    is replaced. `FormField`'s required marker is a glyph standing in for a
    word, which is `img`'s shape. Each is a widget change plus, for
    `ProgressBar`, a value vocabulary `core` does not have.
13. **(age 2 · value medium) Nothing catches a dangling `aria-controls` or a
    duplicate `AccessibilityID`.** Both are written verbatim and neither
    exporter can see the whole document at the moment it writes one — but a
    finished tree *can* be walked, which is exactly what `core.SetDebugMode`
    already does for cursor drift and duplicate keys. A debug-mode pass
    reporting an id claimed twice and a reference resolving to nothing would
    catch the one failure mode the pair has, and it is a failure that is
    invisible on every target rather than merely quiet on two.
14. **(age 2 · value low) An `AccessibilityID` is not validated.** An id
    containing a space is invalid HTML; an empty one is written as `id=""`; one
    starting with `grmob-` collides with `core.TabView`'s minted ids and breaks
    a wiring the author never wrote. The prefix is documented as reserved and
    nothing enforces it. A drop would be silent too, so the honest fix is
    probably the debug-mode pass in item 13 rather than a guard in the
    exporters.
15. **(age 1 · value medium) A Next-list item asserted a spec fact that was
    wrong, and nothing could have caught it.** "`group` supports
    `aria-expanded`" was written into the list, survived three re-sorts, and
    was false. Every ARIA claim in this framework's docs is hand-checked prose,
    and there are now dozens — three role lists, two name prohibitions, four
    `aria-level` scopes, six `aria-selected` roles. A generated table (from the
    ARIA spec's own machine-readable role definitions, checked in as a fixture)
    would make the claims testable instead of reviewable. It is a build-time
    concern rather than a runtime one, which is why it is medium rather than
    high.
16. **(age 1 · value medium) `Accordion` is the only disclosure, and
    `AccessibilityExpanded` has no second consumer.** One consumer is not a
    vocabulary, and the field's role list carries five arms nothing reaches —
    `link`, `listbox`, `row`, `columnheader` and `tab`. The shapes that would
    reach them are real and absent: a `GroupedList` band that collapses (row),
    a combobox built out of `SearchField` plus a listbox (link/listbox), a
    `DataTable` with collapsible column groups (columnheader). Each is a widget
    rather than a prop, so this is a note about where the next one should look
    rather than a task.
17. **(age 1 · value low) `Colors.ControlBorder` is 2.92:1 against
    `Colors.Surface` under `DefaultTheme`.** The outer edge clears 3:1 and is
    what identifies a quiet chip, which is argued in two places and asserted in
    one — but any future widget that draws a boundary *on* a Surface fill
    inherits the shortfall without inheriting the argument. The honest fixes
    are both theme decisions: darken past Apple's systemGray, or give the quiet
    chip a fill that is not Surface.
18. **(age 1 · value low) An `ExpandedState` on a node with no handler is
    silently inert on Android.** `grMobDisclosure` is in `gestureModifier`,
    which returns early when a node carries neither `onClick` nor `onLongPress`,
    so the state reaches the renderer and buys nothing. That is deliberate — an
    action nothing can perform is worse than none — but it is a rule stated
    only in a comment, where the equivalent web rule (a state on an unroled
    node is dropped) has a test on both targets. `core.SetDebugMode` reporting
    a disclosure with no way to open it would put it where the other two are.
19. **(age 0 · value medium) `Margin` has no per-side props at all.** Padding
    got its four this session; margin still has only `Margin(all)`, so every
    single-side margin goes through a whole `EdgeInsets` in a `UseStyle` — and
    that is worse for margin than it was for padding, since there is no
    `MarginHorizontal` or `MarginTop` either. Two live workarounds today:
    `components/separator.go` writes `Margin: EdgeInsets{Horizontal: s.Inset}`
    for its inset rule, and `examples/chat`'s bubble writes
    `Margin: EdgeInsets{Bottom: 8}` for the gap between messages. The work is
    mechanical — the same `settleHorizontal`/`settleVertical` helpers apply
    unchanged, and no renderer would move — which is why this is medium and not
    high: it is scope, not difficulty. Held out of this session because the
    Next entry was about padding and margin is a second decision.
20. **(age 0 · value low) A side prop cannot express "clear this side" on a
    node whose axis a later prop will set.** `PaddingLeft(0)` then
    `PaddingHorizontal(16)` gives 16 on both sides, which is correct
    last-one-wins and is also the only way to write the pair — there is no
    "left stays zero" once the axis prop runs. It has never come up; it is
    recorded because the settle makes every *other* zero work, and the one
    remaining hole should be written down rather than rediscovered.
21. **(age 0 · value low) `rowsSpec` is shared by exactly two widgets and one
    of its ten fields by exactly one.** `Wrap` exists so DataTable can add a
    tap target and a selection tint that GroupedList has no use for, which
    makes it a widget-specific knob living in a shared type. That is the right
    trade at two callers; at three it would be worth asking whether the
    decoration belongs in the spec or in a wrapper around `appendRows`'
    output. Noted because the census test is where a third caller will land.
