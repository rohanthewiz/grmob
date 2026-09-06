# Session: the four oldest, and a focus concept that was not needed

Session: https://claude.ai/code/session_01YVcrPiEYG2vnpUePWiK5T5
Date: 2026-09-06 (follows "a-menu-nobody-could-read-and-a-node-nobody-styled")

## Ask

"Take the oldest 4 items in the Next list."

All four were age 5 — the block that had sat longest:

1. **(high)** Neither a `listbox` nor a `tablist` has keyboard navigation on
   the web, and there are three consumers now.
2. **(medium)** Nothing re-checks a permission on foreground.
3. **(low)** The Android "asked before" flag does not survive a process
   restart.
4. **(low)** A browser `Request` opens the device to answer.

Three of them are one feature seen from three hosts, and the fourth is the
big one. What they turned out to share is the shape that had kept all four
open: **each entry named a blocker that was not there.** Item 1 said it needed
a focus concept `core` does not have; item 2 said it needed a rule for which
screen owns the re-check; item 3 said persisting a flag was not a framework
shell's business. In all three the stated obstacle dissolved on being looked
at, and what was left was small.

## Phase 1 — the keyboard, and the concept that already existed

### The claim three screens were making

`core.Role`'s own doc says the gap plainly, and has for a long time:

> A listbox is a real control in ARIA's model, and the pattern that goes with
> it is larger than two attributes: the container takes keyboard focus, the
> arrow keys move an active option, and the reader is told which option is
> active through a roving tabindex or aria-activedescendant.
>
> None of that is here.

So three shipped screens announced a pattern nothing implemented.
`examples/mobileapp`'s article list is a listbox of `<div>`s — not focusable at
all, so a keyboard user could not reach a row, never mind choose one.
`examples/social`'s bottom bar and tutorial 4.5 are strips of real `<button>`s,
which is worse in a quieter way: a reader announces "tab, 1 of 3" and then the
widget behaves like three unrelated buttons, three tab stops wide.

### Why no new vocabulary was needed

The entry had said this needs a focus concept `core` does not have, and pointed
at `core/focus.go` as the wrong one. That is true about `focus.go` — it is
about putting the cursor in a *named field*, from Go, as a command that rides
the render tree, and routing an arrow key through it would be a render pass per
keystroke. But the conclusion did not follow, because everything the pattern
needs was already crossing the wire:

    what is a member of what   role="listbox"/"option", role="tablist"/"tab" —
                               and the rule in core/role.go that a structural
                               role owns what is inside it
    which one is chosen        aria-selected, which a strip already sets on
                               every member and not only the live one
    which way the arrows go    the container's own resolved flex-direction,
                               which the runtime planted from stackAxisFor
    what activation means      the onClick the author already wired

So the work is a *target reading what was already being said*. No new prop, no
`core` type, and no line changed in any of the three consumers. `core.TabView`'s
own bar came along for free: it is chrome the runtime draws, it already writes
`role="tablist"` and `role="tab"` from the node type, and it goes through the
same table a hand-built strip does.

### The two rules the tab stop follows, and both were bugs first

The obvious implementation syncs the roving tabindex from `aria-selected` and
stops. That is wrong twice:

- A user who has arrowed to the third option **without choosing it** loses
  their place the moment any unrelated patch lands, because the sync puts the
  stop back on the second.
- A click that changed the selection while focus was elsewhere leaves the entry
  point on the old member, so Tab enters the widget at the wrong row.

`activeMemberIndex` answers both by asking in order: the member holding focus,
else the selected one, else whatever already holds the stop, else the first.
The first rule is what makes the widget stable while it is being used; the
second is what makes it correct on the way in.

### htmlout writes none of it, and that needed saying out loud

Everywhere else in this repository a difference between the two DOM targets is
a bug being closed — the Modal chassis, the tab bar, the stack axes are all
"the two web targets must agree" pins. This is the exception, and without a
stated reason the next reader closes it.

**A roving tabindex without the handler that moves it is strictly worse than no
pattern at all.** It takes every member but one out of the page's tab order and
supplies nothing that reaches the rest, so a static export would go from three
tab stops to one tab stop and two unreachable rows. `tabindex` is *behaviour*
here, not semantics, and `htmlout` is not a runtime — its own TabView bar is
already inert chrome. Every semantic half is still written by both targets,
which is exactly what lets the runtime supply the rest.

`wasm/verify/keynav_test.go` holds the line in both directions: the runtime's
role table against `core.Role`'s spellings, and htmlout's output against any
`tabindex` at all.

### Delegation was rejected, and the shim is why

The natural implementation is one delegated `keydown` on the root. The harness
DOM says no, in a comment that was written before it mattered:

> No bubbling and no capture: the runtime attaches every listener to the
> element that owns the prop... **A future move to delegated listeners would
> break here, loudly, which is the right outcome.**

So the listener goes on each member, stamped with `data-grmob-key-nav` so a
member reached by four patches is wired once and a member built fresh by a
`replace` is wired again. That is the same `has_listener_*` discipline
`createElement` already uses, and it is cheap: the sync pass is total over the
member list on every call, exactly as `applyStyle` and `applyAccessibility` are
and for the same reason.

### The patch pass needs both directions

`syncTouchedTabViews` and `syncTouchedOverlays` walk *up* from the touched
elements. This one has to do both:

    up      a member changing its aria-selected moves the widget's tab stop,
            and a member cannot see what contains it — a row inside the wrapper
            core.For produces is two levels below its listbox
    down    an added or replaced subtree may carry a whole composite, and
            nothing above it is one

A `seen` set bounds the down-walks so a batch touching a container and four of
its children does not walk that subtree five times.

It runs **after** `syncTouchedTabViews`, and that ordering is load-bearing:
`syncTabView` is what writes `aria-selected` onto the bar, and the stop follows
the selection.

## Phase 2 — three hosts, one permission

### The re-check, and the objection that assumed too much

`hooks.UsePermission` deliberately does not re-check on foreground, and its doc
gave the reason: a hook cannot see whether its screen is still the one on top,
and five screens in a stack would each fire a check per resume. That is a real
objection to putting the subscription *in the hook*, and for five sessions it
read as an objection to the feature.

It is not, because it assumes a **screen** has to own the re-check. Nothing
about the question is per-screen: there is one device with one camera, and "is
the camera permission still what it was" has one answer however many screens
are asking. So the owner is the permission, and the shape is the one
`core.StartHeading` already found:

    five screens watching Camera         one "permission" event per resume
    one screen watching Camera+Location  two, one per kind
    no screens watching anything         no lifecycle subscription at all

The package comment had said this file was "core.Heading's shape, **minus the
reference counting**". It has the reference counting now, and the sentence
about what the counting is for — two screens watching, one closing, and a plain
on/off flag turning the re-check off under the other with nothing in any log —
transfers unchanged.

The last line is what makes it cheap. And a resume that changed nothing reaches
the record and stops, because `set` notifies only on a change: no subscriber
runs, no render is requested. The cost of watching is a round trip per resume
per kind, and the benefit is that the one resume that *did* change something is
seen.

`hooks.UsePermissionLive` is that plus the mount check. Its own slot is a
second record rather than a field on `permissionRecord`, because the two hooks
do not share a cursor position — folding the flag in would make them read each
other's memory.

### The Android flag, and the failure persisting it introduced

The old comment argued that persisting the asked-before flag "is the kind of
thing an app does with its own preferences, not something a framework shell
should be writing to disk unasked." What tipped it: **the fact being remembered
is not the app's, it is the platform's.** "Has this install ever asked for X"
is bookkeeping Android keeps and simply will not answer, and a quirk of one
platform belongs in the one file that knows about it rather than in every app
built on top. The cost of not having it was paid on every cold start — one dead
button press, forever.

Persisting it introduced a failure the in-memory version could not have.
Android 11+ **auto-reset** revokes an unused app's permissions and clears
don't-ask-again with them; the flag on disk still says "asked", so `status`
answers "denied" while the truth is that a request would show the dialog again.
The old request path short-circuited on anything that was not "prompt", which
would have locked the user out of a permission the system had just handed back.

So `request` short-circuits on `granted` and `unavailable` only. That is safe
because `registerForActivityResult` *always* delivers a result — a permanent
refusal comes straight back as DENIED with nothing on screen, one no-op round
trip for the same answer as before. The old comment's stated reason for the
wider guard ("a screen waiting on a dialog would wait forever") was not true.

Note which way this cuts against the browser's short-circuit, which is the
opposite: there, `denied` *does* stop the request, because it is the browser's
own live answer rather than bookkeeping the host keeps itself.

### The browser, reading before and after

Two costs the old module stated rather than fixed, and both were fixable:

**A granted request opened the camera.** The recording indicator lighting up to
answer a question the browser had already written down. Every request now
queries first — `granted` and `denied` are reported from the record, `prompt`
reaches for the device (opening it *is* the prompt), and a browser with no
descriptor for the kind is asked by asking, which is what this always did.

**A refusal could not say which refusal.** `NotAllowedError` is the same for a
Block and a dismissed prompt, and the two want different words: one can be
asked again, the other wants the user sent to the site settings. The Permissions
API knows — a Block is recorded `denied`, a dismissal leaves `prompt` — so the
refusal path reads it back. Only `prompt` upgrades the answer: a query claiming
`granted` straight after a rejected request is a browser contradicting itself,
and reporting the grant would hand the app a camera that had just refused it.

`NotFoundError` and its neighbours are deliberately outside that read-back —
there is no permission state that describes a camera the machine does not have.

## The break-tests

**Forty-five run, forty-one caught first time. Four gaps, all closed.**

**Gap 1 — an up-walk nothing needed.** Deleting `syncTouchedComposites`'s
up-walk broke nothing, because `touched` carries each element *and its parent*,
so the down-walk from the parent finds the container whenever the member is a
direct child. Every test had direct children. The up-walk earns its place only
for a member inside a wrapper — which is what `core.For` and `core.Keyed`
actually produce — and there is a test for that now.

**Gap 2 — a guard that could not fire.** The member walk skips
`aria-hidden="true"`, and the test for it passed with the guard deleted:
`applyAccessibility` already drops a hidden node's *role*, so a hidden option
is not an option before this walk ever sees it. The guard is live for a hidden
**wrapper**, whose children keep their own roles because that function answers
one node at a time and cannot see an ancestor. The test was repointed at the
case the code is actually for, and the comment now says which case that is.

**Gap 3 — a map with two keys.** `TestEachWatchedKindIsCheckedOnce` asserted
declaration order over two permissions, and Go's randomized map iteration lands
in the right order often enough that the map-order mutant walked through. Split
into two tests: one for the count, and one that asserts the order over six
resumes and three kinds, where "it happened to come out right" is no longer
available.

**Gap 4 — a pin satisfied by half its subject.** The Android persistence pin
matched `getSharedPreferences(...)`, and the mutation that opened the store and
never read it left that string standing. This is the same shape as the last two
sessions' flush-pin misses: *a source-text pin matches the setup, not the use.*
The pin now requires the `getStringSet` as well.

One mutation of mine was simply bad — it *added* a composite pass before the
tab pass while leaving the real one after it, so the second call fixed what the
first got wrong. Rewritten as a move rather than an insertion.

The full list, by phase:

**Composite keyboard (22, three gaps).** Every member a tab stop; the stop
ignoring focus; the stop ignoring the selection; movement clamping instead of
wrapping; every container horizontal; the movement key left to the page; Enter
synthesized for a `<button>` too; activation dropped; Home and End dropped; the
listener stamp dropped; a nested composite's members pooled; **a hidden
wrapper's children counted — the gap**; a disabled control left in the rotation;
the down-walk dropped; **the up-walk dropped — the gap**; the initial render
unsynced; **the pass moved ahead of the tab pass — a bad mutation, rewritten**;
the role table drifting from `core`; the orientation table losing a row; the
container matched loosely; htmlout starting to write a tab stop.

**The foreground re-check (11, one gap).** A resume checking nothing; every
transition re-checking; a resume requesting instead of checking; the count
reduced to a flag; a repeated cancel decrementing again; the subscription never
released; an unspelled permission watched; **the resume order taken from the
map — the gap**; the hook taking no watch; the hook never releasing it; the
live hook no longer reporting a status.

**Android (4, one gap).** **The flag opened and never read — the gap**; the
flag never written; a denied request short-circuiting again; a check writing to
disk from inside a render pass.

**The browser request (8, all caught).** The query skipped entirely; `granted`
no longer short-circuiting; a descriptor-less browser never asked; a refusal
denied without asking which; a refusal reporting whatever the query said; a
missing device sent through the read-back; a refused location denied outright;
storage falling into a query it has no descriptor for.

### A note on the driver

Same Python harness as last session — snapshot by content, one exact-string
mutation, run, restore, assert the restore by hashing, and an anchor that does
not appear exactly once is a `SKIP` rather than a silent no-op. Zero skips and
zero restore failures across both batches.

## Verification

All six paths, green:

    gofmt / go build / go vet / go test ./... / go test -race ./...
    wasm/verify/run.sh          replay + 15 mjs suites
    ios/verify/run.sh           flex solver + 9 picker menus + replay + view + app
    android ./gradlew compileDebugKotlin --offline --rerun-tasks
    GOOS=js GOARCH=wasm go build ./...

New files:

    permission/foreground.go        the reference-counted watch, and the
                                    argument for who owns a re-check
    permission/foreground_test.go   twelve properties, including the concurrent
                                    settle under -race
    wasm/verify/keynav_test.mjs     35 cases: the stop, the movement, the
                                    activation, staying right across patches,
                                    and what is not a member
    wasm/verify/keynav_test.go      the role table against core, and the one
                                    deliberate difference between the two DOM
                                    targets, held from Go

Changed:

    wasm/grmob-runtime.js           the composite keyboard section; the
                                    query-first request and the refusal
                                    read-back
    hooks/permission.go             UsePermissionLive and its own slot record
    permission/permission.go        Check and Denied point at the new half;
                                    resetForTest drops the watches first
    android/.../Permissions.kt      the flag on disk; the narrowed
                                    short-circuit and why auto-reset needs it
    mobile/verify/permission_test.go   the persistence pinned in three parts
    wasm/verify/permission_test.go     two pins repointed, one test added
    wasm/verify/permission_test.mjs    11 cases for the read before and after
    hooks/permission_test.go        six cases for the live hook

## Docs

`core/role.go`: both structural pairs rewritten — the listbox section's "None
of that is here" became a table of what the vocabulary already says and who
supplies the rest; the tab pair gained the same.
`components/list_row.go`: `Selectable`'s "What it does not buy" became "What it
buys on the keyboard, and what the caller has to do for it" — the keyboard
arrives with the *container's* role, not with the flag.
`examples/social/tab.go`: the missing-keyboard paragraph became the argument
for roles as a vocabulary — the file gained the behaviour without gaining a
line.
`permission/permission.go`: the package comment's two-things list grew a third,
"the return"; `Check` points at `WatchForeground`.

`docs/platforms/wasm.md`: a new "Composite widgets are operable" section (what
already said it, what the keys do, the two tab-stop rules, the two exclusions,
and why htmlout writes none of it); the permissions section grew the query-first
table and the Block/dismissal read-back.
`docs/platforms/native.md`: the Android flag on disk, the auto-reset rule, and
the foreground re-check.
`docs/platforms/exporters.md`: "No `tabindex`, on any node, ever."
`docs/concepts/styling-and-theming.md`: "The two pairs that come with a
keyboard".
`docs/concepts/state-and-hooks.md`: `UsePermissionLive`, and the objection that
used to stand in its way.
`docs/components.md`: `ListRow.Selectable` and `SegmentedControl`-as-tab-strip
both gained their keyboard paragraph.
`examples/tutorial/chapter4.go`: two prose blocks and a key point.
`ROADMAP.md`: four new entries, and the roles entry grew the keyboard half.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here. Value is the payoff of doing it, not the effort: `high` means
something is being worked around today or a second consumer has arrived,
`medium` means it blocks one named thing, `low` means it is a gap nobody has
bumped into yet.

1. **(age 6 · value low) The gomobile stub's types are unchecked.** The pin
   holds the *names* — every bindable symbol declared, nothing extra —
   because copying gobind's type mapping into a test would be reimplementing
   gobind. A wrong signature fails the Swift typecheck the moment the shell
   calls it, so what is genuinely unguarded is a stub declaration the shell
   never touches.
2. **(age 5 · value medium) A hand-built tab strip's panel still cannot say it
   is one.** `AccessibilityControls` closes the pointing and `RoleTabPanel` is
   now defensible. What stands in the way is mechanical: the WASM runtime
   tells its own wiring apart from an author's role by `"tabpanel"` not being
   a `core.Role`, held there by `TestNoRoleCollidesWithTheTabPanelWiring`.
   Replacing that discriminator with a `data-grmob-panel` marker (which both
   web targets would write, as they already do `data-grmob-chrome`) is the
   right fix and unblocks the constant. Two native arms and a large doc block
   go with it. **Cheaper than it was:** the runtime now reads roles for
   behaviour as well as for attributes, so a marker it writes and reads back
   is an established shape rather than a new one.
3. **(age 5 · value medium) Three widgets take `group` where ARIA has a better
   role.** The supplied fallback made their names audible and stopped there.
   `ProgressBar` wants `progressbar` with `aria-valuenow`/`valuemin`/`valuemax`
   — it is already computing the percentage into its name, which is the
   workaround. `Skeleton` wants `status`, since "Loading" is an advisory that
   is replaced. `FormField`'s required marker is a glyph standing in for a
   word, which is `img`'s shape. Each is a widget change plus, for
   `ProgressBar`, a value vocabulary `core` does not have.
4. **(age 5 · value medium) Nothing catches a dangling `aria-controls` or a
   duplicate `AccessibilityID`.** Both are written verbatim and neither
   exporter can see the whole document at the moment it writes one — but a
   finished tree *can* be walked, which is exactly what `core.SetDebugMode`
   already does for cursor drift and duplicate keys. A debug-mode pass
   reporting an id claimed twice and a reference resolving to nothing would
   catch the one failure mode the pair has, and it is a failure that is
   invisible on every target rather than merely quiet on two.
5. **(age 5 · value low) An `AccessibilityID` is not validated.** An id
   containing a space is invalid HTML; an empty one is written as `id=""`; one
   starting with `grmob-` collides with `core.TabView`'s minted ids and breaks
   a wiring the author never wrote. The prefix is documented as reserved and
   nothing enforces it. A drop would be silent too, so the honest fix is
   probably the debug-mode pass in item 4 rather than a guard in the
   exporters.
6. **(age 4 · value medium) Every ARIA claim in the docs is hand-checked
   prose.** A Next-list item once asserted "`group` supports `aria-expanded`",
   which survived three re-sorts and was false. There are dozens of such
   claims now — three role lists, two name prohibitions, four `aria-level`
   scopes, six `aria-selected` roles, and as of this session a keyboard
   contract per composite role. A generated table (from the ARIA spec's own
   machine-readable role definitions, checked in as a fixture) would make them
   testable instead of reviewable. Build-time rather than runtime, which is
   why it is medium.
7. **(age 4 · value medium) `Accordion` is the only disclosure, and
   `AccessibilityExpanded` has no second consumer.** One consumer is not a
   vocabulary, and the field's role list carries five arms nothing reaches —
   `link`, `listbox`, `row`, `columnheader` and `tab`. The shapes that would
   reach them are real and absent: a `GroupedList` band that collapses (row),
   a combobox built out of `SearchField` plus a listbox (link/listbox), a
   `DataTable` with collapsible column groups (columnheader). **Three sessions
   running have now unblocked an item stuck on exactly this argument** —
   `StackAlign` by a fact about the three platforms, the picker menu by a fact
   about what a view closure can be asked, and the listbox keyboard by a fact
   about what was already on the wire. Ask the same question here before
   waiting for a widget: a combobox is `SearchField` plus a listbox that now
   has arrow keys, which is less missing than it was a session ago.
8. **(age 4 · value low) `Colors.ControlBorder` is 2.92:1 against
   `Colors.Surface` under `DefaultTheme`.** The outer edge clears 3:1 and is
   what identifies a quiet chip, which is argued in two places and asserted in
   one — but any future widget that draws a boundary *on* a Surface fill
   inherits the shortfall without inheriting the argument. The honest fixes
   are both theme decisions: darken past Apple's systemGray, or give the quiet
   chip a fill that is not Surface.
9. **(age 4 · value low) An `ExpandedState` on a node with no handler is
   silently inert on Android.** `grMobDisclosure` is in `gestureModifier`,
   which returns early when a node carries neither `onClick` nor `onLongPress`,
   so the state reaches the renderer and buys nothing. That is deliberate — an
   action nothing can perform is worse than none — but it is a rule stated
   only in a comment, where the equivalent web rule (a state on an unroled
   node is dropped) has a test on both targets. `core.SetDebugMode` reporting
   a disclosure with no way to open it would put it where the other two are.
10. **(age 3 · value medium) `Margin` has no per-side props at all.** Padding
    got its four; margin still has only `Margin(all)`, so every single-side
    margin goes through a whole `EdgeInsets` in a `UseStyle` — and that is
    worse for margin than it was for padding, since there is no
    `MarginHorizontal` or `MarginTop` either. Two live workarounds today:
    `components/separator.go` writes `Margin: EdgeInsets{Horizontal: s.Inset}`
    for its inset rule, and `examples/chat`'s bubble writes
    `Margin: EdgeInsets{Bottom: 8}` for the gap between messages. The work is
    mechanical — the same `settleHorizontal`/`settleVertical` helpers apply
    unchanged, and no renderer would move — which is why this is medium and not
    high: it is scope, not difficulty.
11. **(age 3 · value low) A side prop cannot express "clear this side" on a
    node whose axis a later prop will set.** `PaddingLeft(0)` then
    `PaddingHorizontal(16)` gives 16 on both sides, which is correct
    last-one-wins and is also the only way to write the pair. Recorded because
    the settle makes every *other* zero work, and the one remaining hole should
    be written down rather than rediscovered.
12. **(age 3 · value low) `rowsSpec` is shared by exactly two widgets and one
    of its ten fields by exactly one.** `Wrap` exists so DataTable can add a
    tap target and a selection tint that GroupedList has no use for, which
    makes it a widget-specific knob living in a shared type. That is the right
    trade at two callers; at three it would be worth asking whether the
    decoration belongs in the spec or in a wrapper around `appendRows`' output.
13. **(age 2 · value medium) `declaredInk` is now unobservable under both
    bundled themes.** Each pairs white with a fill dark enough that measurement
    picks white too, so an implementation that deleted the first step and only
    measured would paint identical pixels under DefaultTheme *and*
    MaterialTheme. The rule is right and still applies to any theme whose house
    button is a mid-tone — but its whole evidence is now one test fixture
    (`midTonePrimaryTheme`), which is a thinner thread than a bundled theme.
    The honest options are to leave it and accept the fixture, or to give one
    bundled theme a mid-tone button base on purpose, which is a look decision.
    Same shape one property over: `PrimaryOnLight` is an identity on both
    bundled themes now, so the *reverse lookup* has no live consumer either.
14. **(age 2 · value medium) An unsized `ZStack` with a placed layer diverges
    on iOS.** SwiftUI has no per-child stack alignment, so a placed layer is
    wrapped in a filling frame, and a filling frame grows an otherwise unsized
    stack to its parent's proposal — where a Compose `Box` and a CSS grid track
    stay the size of their largest child. Documented in four places and
    avoidable by pinning the stack's box, which `ZStack` already asks for. What
    would actually close it is a SwiftUI layout that places without filling
    (a custom `Layout`, or an `alignmentGuide` scheme that can see the
    container's size), and neither is a small piece of work. Nothing pins the
    divergence today either — `ios/verify` type-checks and replays, it does not
    measure.
15. **(age 2 · value low) `StackAlign` is inert outside a `ZStack` and nothing
    says so at the call site.** It is inert deliberately and by construction on
    the web (the stack imposes it, so it never reaches a flex child), and by
    omission on the natives (only the stack renderers read the field). But a
    `core.StackAlign` on a `Column`'s child compiles, merges, crosses the wire
    and does nothing on all four targets with no diagnostic. `core.SetDebugMode`
    already walks a finished tree for cursor drift and duplicate keys; a
    placement on a node whose parent is not an overlay is the same kind of
    finding, and it would join items 4 and 9 in the same pass.
16. **(age 2 · value low) A `Select`'s group headings cannot be styled or
    ordered independently.** A heading is a string on an option, so there is no
    way to give one an icon, mark a whole run disabled, or state a heading that
    has no options under it. `<optgroup disabled>` exists in HTML and both
    natives could express a disabled section; nothing in `SelectOption` can ask
    for it. `core.SelectMenuSections` is now the one place a section is
    described, so a heading with fields of its own would be a change to
    `SelectMenuSection` and its three transliterations rather than to four
    independent renderers.
17. **(age 1 · value medium) The Kotlin decomposition has no runner.**
    `GrMobSelectMenu.kt` imports nothing precisely so a JVM harness could
    execute it, and `ios/verify` proves what such a harness buys — nine cases
    generated from `core.SelectMenuSections`, compared against the real Swift
    function. Android has no equivalent: `compileDebugKotlin` is the only check
    the build runs, and adding a `src/test` source set means resolving JUnit
    against a `--offline` gradle cache that may not carry it. **A second thing
    now rests on source-text pins alone:** this session's asked-before
    persistence, which is three `strings.Contains` in `mobile/verify` and a
    regexp — and this session's own gap 4 was exactly one of those pins passing
    on half its subject. The persistence itself needs an Android context and
    would not run in a plain JVM harness, so this does not widen the item; it
    sharpens why the weaker kind of pin is worth replacing where it can be.
18. **(age 1 · value low) The WASM runtime is the fourth copy of the
    decomposition and does not call the authority.** `applySelectOptions` walks
    the flat list itself, appending into a live `<optgroup>`. That is defensible
    — a DOM append has no closing step, so the flush that the other three have
    to remember does not exist here — but it means a change to
    `SelectMenuSection`'s shape has three consumers to update and one to
    remember. The mjs suite covers the behaviour live; what is missing is the
    *shared fixture*, i.e. `wasm/verify/gen.go` emitting the same `menuCases`
    table so the runtime is compared against Go's answer rather than against a
    hand-written expectation.
19. **(age 1 · value low) `styleFromGrMob` has exactly one exemption from its
    own totality rule, and nothing states the rule for adding a second.**
    `delete out.display` for a `Modal` is right: the display is the open/closed
    state, it arrives through the prop channel, and a total pass cannot see a
    prop. But "abstain by deleting the key" is now a technique available to any
    property, and the next one to reach for it will not have this argument
    attached. The test pins the line; what is unwritten is the *test a candidate
    has to pass* — that the property is owned by a channel this function cannot
    read, and that some other path is total for it instead.
20. **(age 0 · value medium) Neither web target writes `aria-orientation`, and
    a vertical strip now behaves in a way it does not announce.** The runtime
    reads the container's resolved flex-direction to pick the arrow pair, so a
    `tablist` laid out as a `Column` takes Up/Down — correctly. ARIA's default
    for `tablist` is horizontal, so a reader in browse mode is told the
    opposite of what the widget does. This is pure semantics and belongs on
    **both** web targets, derived from the same axis both already compute
    (`StackAxisFor` here, `stackAxisFor` there) — which makes it a shared-table
    change of the kind `objectfit.go` and `stack.go` already are, and the
    reason it was left out of this session rather than done half. `listbox`'s
    ARIA default is vertical, so the horizontal case has the same gap in
    mirror.
21. **(age 0 · value low) A listbox has no typeahead.** ARIA's pattern includes
    type-to-jump — pressing "m" moves to the next option starting with "m" —
    and it is the thing that makes a long listbox usable at all with a keyboard;
    `examples/mobileapp`'s article list is exactly the shape that wants it. It
    needs a per-widget key buffer with a timeout, which is the first piece of
    *state* this section would own (everything it holds today is derived from
    the DOM on demand, which is what makes it survive every patch for free).
    That is the whole argument for and against.
22. **(age 0 · value low) The keyboard pattern is verified against a DOM that
    is not a browser.** `wasm/verify/dom.mjs` models attribute storage, parent
    links, listener dispatch and which element holds focus — enough for
    everything the roving tabindex does — but it has no bubbling, no layout,
    and its `focus()` is an assignment. So what is genuinely unverified is the
    browser's half of the contract: that `tabindex="-1"` really does remove a
    `<button>` from the tab order, that a disabled control really does refuse
    focus, that `preventDefault` on `ArrowDown` really does stop the scroll.
    All three are well-specified and none is in doubt; the point is that
    nothing here would notice if one stopped being true. The same sentence is
    true of `enterkeyhint` and the software keyboard, and the honest answer for
    both is a browser-driven pass rather than a wider shim.
23. **(age 0 · value low) `menu`, `tree` and `grid` are patterns `core.Role`
    has no vocabulary for, and the machinery for them now exists.** The
    composite section is driven by one two-row table and would take a third row
    without changing shape — but each of those patterns needs more than arrows:
    a menu has submenus and Escape, a tree has expansion state per node (which
    is `AccessibilityExpanded` looking for a consumer, item 7), a grid is
    two-dimensional. Recorded because the *reason* they are absent has changed:
    it used to be "there is no behaviour to hang on them", and now it is "no
    widget in this repository is one".
