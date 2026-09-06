# Session: a module that does not exist, a role that had to, and a package that was pretending

Session: https://claude.ai/code/session_01YVcrPiEYG2vnpUePWiK5T5
Date: 2026-09-06 (follows "two-containers-and-an-ink")

## Ask

"Now do the oldest 3 items on the Next list."

The three oldest were items 1, 2 and 3: the dead `permission` package (age 3,
medium), three Swift app-layer files checked by nothing (age 3, low), and
`ListRow` still unable to announce its selection (age 2, high).

Two of them turned out to be **the same shape from opposite ends**: a thing
that exists and does nothing. A Swift file the compiler never sees, and a Go
package with a vocabulary and no functions. The third is a widget that has been
saying a true thing in the wrong place for three sessions because the
vocabulary had no word for the right one.

Ordered by size, smallest first: the stub, the listbox pair, the permission
package.

## Phase 1 — a module that does not exist, compiled

### Only one of the three files was the problem

The item said all three import the generated framework. They do not.
`AppLifecycle.swift` imports SwiftUI and touches `GrMobRuntime`; `GrMobApp.swift`
imports SwiftUI and *constructs* `GomobileBridge`; only `GomobileBridge.swift`
says `import GrMob`. One import held all three out, including the `@main` entry
point.

That is worth stating because it changes what the fix is. Not "find a way to
run gomobile in the harness" — which needs full Xcode *and* a gomobile
toolchain, and would cost the harness most of the people who need it — but
"make one import resolve".

### A stand-in module, not a stub file

`ios/verify/gomobile_stub.swift` is compiled *as* `GrMob`:

    swiftc -emit-module -module-name GrMob \
      -emit-module-path "$out/GrMob.swiftmodule" ... gomobile_stub.swift
    swiftc -typecheck -I "$out" ../GrMob/Runtime/*.swift ../GrMob/App/*.swift

The `.swiftmodule` goes to the scratch directory rather than beside the
sources, so a stale one can never shadow the real framework in an Xcode build.
Nothing links; nothing ships.

It copies gobind's two-step renaming rather than approximating it — the
package name capitalized onto every symbol (`mobile.RenderInitial` →
`MobileRenderInitial`, which is why a framework named `GrMob` is full of
`Mobile*`), and a Go interface becoming both an ObjC protocol and a class of
that name, which Swift disambiguates by suffixing the protocol. It also copies
the *nullability*: gobind annotates every `NSString*` parameter `_Nullable` and
every return `_Nonnull`, so a Go `string` argument arrives as `String?` and a
`string` result as `String`. A stub taking `String` everywhere would accept
shell code the real framework rejects.

### The new way to be wrong, and the pin that closes it

A hand-written stand-in for generated code is a copy, and the drift is
unusually bad: the shell keeps type-checking green against a bridge Go no
longer has, which is the exact opposite of what the pass was added for.

`mobile/verify/gomobilestub_test.go` reads package `mobile`'s syntax tree and
requires a declaration for every bindable exported function and every exported
interface — and requires that nothing in the stub names a symbol Go does not
have. Both directions, for the reason `requireRoleCoverage` checks both: a
missing declaration silently drops the app layer out of the check, a surplus
one reads as support for a bridge function that will not exist after a bind.

`Register` is the case the bindable filter exists for. It takes a
`*core.Context` and a func, so gobind emits `// skipped function Register with
unsupported parameter or return types` where the declaration would be, and a
stub that declared it would offer the shell a call that cannot exist.

Deliberately **not** checked: the parameter and result types. Copying gobind's
full type mapping into a test would be reimplementing gobind to check a file
that exists to avoid running gobind, and a wrong signature fails the Swift
typecheck the moment the shell calls it — which is the same pass, three lines
down.

The syntax tree rather than text, unlike everything else in `mobile/verify`,
because here the subject is Go's own source and the question ("which functions
have a signature gobind can carry?") is one no substring match can answer.

## Phase 2 — `option` inside `listbox`

### The item was right that both doors were shut

`ListRow` has appended `", selected"` to its own accessible name since it was
written, and its doc has explained why for three sessions: `RoleButton` would
make the row a foreign child of any `role="list"` around it, costing the whole
list its shape for one row's announcement, and `RoleListItem` — the honest
description of a row — is not a role ARIA scopes `aria-selected` to.

    aria-selected is defined for: gridcell, option, row, tab, columnheader,
                                  rowheader
    and pointedly not for:        listitem

So the fix was never a widget change. It was two constants.

### `RoleListBox` / `RoleOption`, the fourth structural block

Both natives get an **empty arm**, and the empty arm is the interesting part
rather than a gap. Neither SwiftUI nor Compose has a listbox in its semantics
vocabulary at all — both spell a chosen item as a *state* (`.isSelected`,
`selected`) and honour it on any node without being told what contains it. So
the pair costs them nothing to leave out, and the row still reads as chosen on
device. What is missing there is only the container's word for what the choice
is among, which neither phone's reader navigates by.

That makes this a web-only fix in effect and a vocabulary fix in fact, which is
why it belongs in `core.Role` rather than in `htmlout`.

### What is not bought, said out loud

A listbox in ARIA's full pattern takes keyboard focus, moves an active option
with the arrow keys, and reports which one through a roving tabindex. None of
that is here: `core.Role` is a vocabulary, and nothing in `core` stamps a
`tabindex` or reads an arrow key (`core/focus.go` is about putting the cursor
in a named field, a different question). So the semantics are stated and the
behaviour is the author's — the same shape the structural rule already takes,
where a `list` role's promise about its children is the caller's to keep. On
both phones the gap costs nothing.

### `ListRow.Selectable`, and the depth that loses

    Selectable: true   ->  role=option + aria-selected, on both values
                           of Selected, and no ", selected" suffix
    Selectable: false  ->  exactly the row that shipped yesterday

Both values, not just the chosen one. A listbox where only the selection
answers announces the rest as plain rows — `core.SelectedOff` doing the job it
was added for, one widget over from the tab strip whose argument it was.

`Selectable` and `NestingLevel` are exclusive because a node has one role and
the two roles carry opposite attributes: `option` takes `aria-selected` and no
`aria-level`, `listitem` the reverse. A row asking for both describes a
container that is a list and a listbox at once. `Selectable` wins and the depth
is not merely ignored downstream — it is never set, so no exporter has to know
about a combination `core` does not produce.

ARIA's role for an item that is genuinely both is `treeitem` inside a `tree`,
and it is deliberately absent: a tree is a third pattern with its own expansion
state and keyboard contract, and nothing here has one.

The suffix stays for a row that is not `Selectable`, because that row still has
no role that could carry the state, and saying the true thing weakly beats not
saying it.

### Adopted downstream twice

`examples/mobileapp`'s article list is a `core.List` holding nothing but rows
with a single selection — a listbox by construction. It now says so, which puts
`role="option"` and a live `aria-selected` through the bridge transcript that
both `ios/verify` and `wasm/verify` replay. Its own test asserts the state
crosses in the tap's patch *and* that the label stopped growing a suffix.

The tutorial's `ListRow` lesson (4.3) teaches the pair on the roster demo it
already had, and gained the sentence the vocabulary needed: a row is a choice
or a depth, never both.

## Phase 3 — the `permission` package, made real

### Deciding between deleting it and finishing it

The package was four `Permission` constants, three `PermissionStatus`
constants, and two commented-out functions in Portuguese calling a
`core.InvokeNative` that is not in the codebase. Nothing imported it.

The argument for deleting it is strong and is already written down: three
sessions ago the compass work concluded that a permission moment belongs to the
capability that needs it — `core.StartHeading` makes the browser's motion
prompt itself, on the explicit reasoning that a separate API is one more thing
to forget.

What settled it the other way is one comment, in
`ios/GrMob/App/HeadingSensor.swift`:

    An app that wants true north asks for location authorization by its own
    route; nothing here prompts, because a permission dialog the user did not
    expect is a worse failure than a bearing that is a few degrees off a map.

That route did not exist. `Heading.HasTrue` is false on iOS until location is
authorized, and the framework had no way to ask. A named, in-repo consumer, one
file over from the feature that declined to solve it.

So: two things a capability's own start cannot serve, and the package exists
for exactly those —

    the rationale    "we will need your location, here is why", *before* the
                     OS dialog. Asking mid-render is how an app earns a
                     permanent refusal.
    the read-back    a settings row saying "Location: denied". Check never
                     prompts, so it is safe on mount, and it is the only way
                     to draw that row at all.

### Shaped like the sensor, minus the refcounting

There is no request/reply primitive on this bridge, and inventing one for this
would be a second way to do what the sensor already does. So:

    Request/Check ──"permission" system event──▶ host authorization API
    Current ◀── status record ◀──"permission" host event── host

The consequence is the point: a screen reads state and re-renders on change,
so there is no callback to leak, no request id to correlate, and a prompt the
user leaves standing for a minute strands nothing.

### Four statuses, and the two that are usually one

Spelled the W3C Permissions API's way, for the reason `core.Role`'s values are
ARIA's: the browser then needs no mapping table, and the two natives map their
richer enums onto it, which is the same trade in the same place.

    Unknown       the zero value. Nobody has asked. Not a platform state — no
                  host may report it — and it exists for the reason
                  Heading.Received does: a spinner is right for "not yet" and
                  wrong for "no".
    Granted       usable now
    Prompt        undecided. Request will show a dialog. The one status with a
                  button on it.
    Denied        refused. Asking again shows nothing on most platforms; the
                  fix is the system settings.
    Unavailable   this device or platform cannot grant it at all.

`Denied` and `Unavailable` are the pair that usually collapse, and separating
them is not tidiness: a denial is fixed in Settings and this is not, so a
screen offering "Open Settings" for both sends someone to a page with no switch
on it.

`Pending` did not survive. It named "not yet asked", `Prompt` is the published
vocabulary's word for that, and the old constant was in a package where nothing
could produce or consume it. `PermissionStatus` did survive, as a deprecated
alias for `Status` — one line, where a rename would break an import that may
exist outside this repository.

### The headless short-circuit

`core.SendSystemEvent` drops events when no host has registered a handler,
which is right for a toast and wrong here: a screen would sit on `Unknown`
drawing the spinner that state is for, forever, in every Go test and every
static export. `core.HasSystemEventHandler()` is new (six lines) and `send`
answers `Unavailable` immediately when nothing is listening — because "there is
no platform to ask" is precisely what `Unavailable` means.

A shell that *has* a handler but no arm for this event name is a case Go cannot
see, and stays on `Unknown`. That is what the host coverage checks are for.

### A malformed answer changes nothing

Every alternative is a lie with a cost — `Granted` opens a device the OS did
not authorise, `Denied` hides a feature that works, `Unavailable` sends the
user to a settings page with nothing on it. So a payload missing either key,
naming an unknown permission, or carrying an unspelled status is dropped and
the record stays where it was, which leaves the screen showing what it was
showing. `Unknown` is refused as an *incoming* status for the same reason it is
not in `Statuses()`: a host reporting it would be answering the question with
the absence of an answer.

### Three hosts, three genuinely different problems

**Browser.** The host where the two commands are different operations and one
of them mostly cannot be honoured. `check` is a real `navigator.permissions.query`
answering in exactly Go's words; `request` has *no API* — a page obtains a
capability by calling the feature, and the prompt is a side effect. So a
request is a `getUserMedia` whose tracks are stopped the instant it resolves
(the prompt is the point, the stream is not, and a live track is a recording
indicator nobody asked for), or a single `getCurrentPosition`. Two seams
matter: `query()` *throws* for a descriptor it does not know rather than
resolving — Firefox has no `camera` at all, so the catch is a common path — and
a rejected `getUserMedia` says both "refused" and "there is no camera" through
one channel, which `NotFoundError`/`OverconstrainedError`/`NotReadableError`
split off into `Unavailable`.

**iOS.** Four capabilities, four frameworks, four enums — the mapping table
Go's spelling avoids on the web is unavoidable here, and is written once. Three
of the four are static calls; `CLLocationManager` is a stateful object that
reports through its delegate and reports *nothing* if released while its prompt
is up, so `Permissions` is a singleton holding one. Deliberately not
`HeadingSensor`'s manager: that one asks for nothing on purpose, and sharing
the object would make one of those decisions the other's. Photos' `.limited`
maps to `Granted` — the app really can read the photos the user picked.

**Android.** Two states where Go wants three. `checkSelfPermission` answers
GRANTED or DENIED, and `shouldShowRequestPermissionRationale` is false both for
"never asked" and for "don't ask again". The shell tracks whether *this
process* has asked and breaks the tie towards `Denied`, because reporting
`Prompt` for a permanent refusal leaves a screen offering a button that does
nothing. The flag is in memory; persisting it is something an app does with its
own preferences, not something a framework shell writes to disk unasked.

### Two build-time halves no Go code can supply

An iOS prompt whose `NS*UsageDescription` key is missing **terminates the app**
at the moment it would appear — the failure lands right after everything looked
fine. An Android permission missing from the manifest is auto-denied with
nothing on screen.

Both are pinned. The iOS check reads `ios/project.yml` and not the
`Info.plist`: the plist is generated by xcodegen and gitignored, so a check
reading it fails on a fresh clone for the wrong reason — and the spec is the
file a fix has to touch anyway, since editing the generated plist alone is
undone by the next `xcodegen generate`. Both checks match the *declaration*
(a key with a non-empty value, an `android:name="…"` attribute) rather than a
bare substring, so a permission named in a neighbouring comment cannot satisfy
them.

### The govinci twin, found late

`grep` for `grmob/permission` found nothing outside the package. It also missed
a second copy of the same dead feature: `wasm/main.go`'s exported
`RequestPermission`, its own third `Permission` vocabulary, and a
`window.GrMobRequestPermission` in the runtime — camera-only, Portuguese
comments, unreachable from `package main`. All three are gone, along with the
`docs/platforms/wasm.md` section that documented the page global as the way
permissions work.

## Verification

All six paths, green:

    go build / go vet / gofmt / go test ./... / go test -race ./...
    wasm/verify/run.sh          replay + 12 mjs suites
    ios/verify/run.sh           data + view + app layers (the app layer for
                                the first time including all six files)
    android ./gradlew compileDebugKotlin --offline --rerun-tasks
    GOOS=js GOARCH=wasm go build ./...

New and changed tests:

    mobile/verify/gomobilestub_test.go  the stub against mobile's syntax tree,
                                        both directions, with Register's
                                        unbindable signature filtered out
    mobile/verify/permission_test.go    both shells' dispatch and attach, both
                                        commands, every Permission covered,
                                        only declared statuses reported, and
                                        the two manifests
    permission/permission_test.go       the wire, the record's isolation per
                                        permission, seven malformed payloads,
                                        Unknown refused as an answer, change-
                                        only notification, a subscriber
                                        re-entering, the headless answer, and
                                        both enums pinned to their const blocks
    hooks/permission_test.go            check-once-never-request, the filter
                                        that keeps one permission's answer out
                                        of another's hook, re-subscription
                                        after Close, and the headless resolve
    wasm/verify/permission_test.go      the descriptor table, the wire, the
                                        statuses the host writes itself, and
                                        that a request calls the feature
    wasm/verify/permission_test.mjs     13 live cases: the forwarded word, the
                                        descriptor rename, a throwing query,
                                        the stopped tracks, the denied/absent
                                        split, geolocation's three codes
    components/list_row_test.go         both sides of the choice, the suffix
                                        dropped, Selectable taking the role
                                        from the depth, an unroled row claiming
                                        nothing, and the look unchanged
    examples/tutorial/chapter4_test.go  the roster as a listbox with every row
                                        answering, and the compass lesson's
                                        permission resolving to Unavailable
                                        rather than spinning
    examples/mobileapp/app_test.go      the option state in the tap's patch,
                                        and the name that stopped moving

Every new assertion was confirmed to bite by breaking the thing it guards: the
stub gaining a symbol Go lacks and losing one it has, a Swift typo'd status, a
dropped Kotlin permission arm, `SelectedWhen` narrowed to `SelectedOn`, the
suffix guard removed, the role precedence flipped, `HasSystemEventHandler`
deleted, a usage-description key misspelled, and a manifest permission removed.

## Docs

`core/role.go`: the twenty-two-value table, the fourth structural block with
the listbox pair's full argument (why not a state on `listitem`, what a listbox
promises that this does not supply, and the depth question answered the other
way). `htmlout/export.go` and `wasm/grmob-runtime.js`: `option` in the
`aria-selected` arm, and the near-miss paragraph rewritten now that `listitem`
is a live distinction rather than a missing constant.

`components/list_row.go`: "How the state is announced, and why it took a role
to do it" replaces "Why this row still spells its state into the name", plus
the `Selectable` field's five sections.

`docs/concepts/styling-and-theming.md`: the listbox pair in the role table and
a new paragraph before the structural rule; `option` in the `aria-selected`
scoping table. `docs/components.md`: `Selectable` with its example and the
precedence note.

`docs/concepts/state-and-hooks.md`: "Permissions: `UsePermission`" beside the
sensor hook — check vs request, four statuses, the foreground gap, and why this
is not a second way to do what `StartHeading` does.
`docs/platforms/native.md`: a "Permissions" section with the per-host status
table, Android's three-out-of-two reconstruction, the two build-time halves,
and why iOS needs an object where Android needs the Activity.
`docs/platforms/wasm.md`: the stale `GrMobRequestPermission` section replaced
with the two command tables and the four seams.

`ROADMAP.md`: the permission entry, the roles entry at twenty-two, and the
selected-state entry naming `ListRow` as its third adopter. `README.md`: the
`permission/` package and `UsePermission`.

Tutorial: 4.3 teaches the listbox pair on the roster it already had; **4.10
grew a permission half** — the live location status, an ask button that appears
only for `Prompt`, and the four-status argument, which is the lesson where the
compass's own limitation is already on screen.

## Next

Sorted by **age**, oldest first — the default. To read the same list by payoff,
re-sort by **value** (high → medium → low, age breaking ties). Age is how many
saved sessions ago the item was first raised, counted in
`ai_docs/claude_sessions/` and measured from this doc, so `age 0` means it was
raised here. Value is the payoff of doing it, not the effort: `high` means
something is being worked around today or a second consumer has arrived,
`medium` means it blocks one named thing, `low` means it is a gap nobody has
bumped into yet.

1. **(age 3 · value low) A hand-built tab strip cannot point at its panel.**
   `aria-controls` and `aria-labelledby` are IDREFs and `Style` carries values.
   `core.TabView` covers the wired case, so this is only a gap for a strip
   built out of chips — no consumer yet.
2. **(age 3 · value low) `RoleLog` has one consumer and no test of its own.**
   `examples/chat` has no test file at all.
3. **(age 2 · value high) An accessible name on an unroled container is
   dropped on both web targets.** `<div aria-label="…">` with no role is
   `role=generic`, where ARIA prohibits the attribute. It hits every `ListRow`
   with an `AccessibilityLabel` and every `Accordion` header. Both natives
   honour the name, so this is a two-target silence, and the fix is a design
   question: which role does a named, tappable, non-button row get? Note that
   `RoleOption` now answers it for one shape — a selectable row has a role and
   therefore a legal name — which narrows the question rather than closing it.
4. **(age 2 · value medium) `core` has no `aria-expanded`.** `Accordion` is the
   package's one stateful widget and its header cannot say whether it is open.
   The natural shape is `core.AccessibilityExpanded` with `SelectedState`'s
   three values, scoped by role on the web and mapped to Compose's
   `expand`/`collapse` actions.
5. **(age 2 · value medium) `Colors.Border` is still spent as a control
   boundary by `Chip`.** A `ProminenceQuiet` chip is a Surface fill on the
   page's Background (1.09:1) inside a `Border` hairline (1.26:1), so the
   control that filters the screen is close to invisible as a control.
6. **(age 2 · value low) `appendRows` takes twelve positional arguments.** The
   band knobs travel together through both collections; a struct would make the
   next one an addition rather than a transposition risk.
7. **(age 2 · value low) `core` has no single-side padding props except
   `PaddingTop`.** An indent goes through a whole `EdgeInsets` in a `UseStyle`.
8. **(age 1 · value medium) `DefaultTheme` paints 4.02:1 on its own primary.**
   `inkOn` returns the theme's declared white on `#007AFF`, below AA for body
   text, and every filled `Button` plus the calendar's selected day spends it.
   The fix is the theme's — most likely Apple's accessible `#0040DD`, which the
   palette already carries as `PrimaryOnLight` — and it would move every filled
   button in every app, which is why it is a decision rather than a patch.
9. **(age 1 · value medium) `ZStack` has no per-child alignment.** Every layer
   centres, and a layer that wants a corner wraps itself in a box sized to the
   stack. An `AlignSelf`-shaped prop is the obvious shape and all three
   constructs have a spelling for it, but one consumer is not a vocabulary.
10. **(age 1 · value low) A `Select` cannot be grouped or disabled per
    option.** `<optgroup>` and a disabled `<option>` have no place in
    `SelectOption`, and neither native menu would read one without work.
11. **(age 1 · value low) A picker's open menu is invisible to Go.**
    Deliberate, but it means no test can drive a native picker's list, and the
    two `mobile/verify` pins read source text rather than behaviour.
12. **(age 1 · value low) A styleless node skips the border reset in the WASM
    runtime.** `applyStyle` only runs when `node.Style` is non-nil. No node
    `core` builds is ever in that state, so this is a hand-assembled-tree gap.
13. **(age 0 · value medium) A `listbox` has no keyboard navigation on the
    web.** `RoleListBox`/`RoleOption` state the semantics and supply none of
    the behaviour ARIA's pattern promises: the container should take focus, the
    arrow keys should move an active option, and a roving tabindex or
    `aria-activedescendant` should say which. Both phones navigate by swipe and
    lose nothing; a keyboard user on the web gets a role that claims more than
    the widget does. Needs a focus concept `core` does not have — `core/focus.go`
    is about putting the cursor in a named field.
14. **(age 0 · value medium) Nothing re-checks a permission on foreground.**
    A user can grant one in Settings and come back, and no platform says so.
    `hooks.UsePermission` deliberately does not (it cannot see whether its
    screen is still on top, and five screens would each fire a check per
    resume), so every consumer writes the same `UseLifecycle` pairing by hand.
    A `hooks.UsePermissionLive` — or a `RecheckOnForeground` option — is the
    obvious shape; what is missing is a rule for which screen owns it.
15. **(age 0 · value low) The Android "asked before" flag does not survive a
    process restart.** It is what separates a permanent refusal from "never
    asked", so after a restart a permanently-denied permission reads as
    `Prompt` until the next request proves otherwise — one dead button press.
    Persisting it means a framework shell writing to disk unasked, which is why
    it was not done; an app that cares can track it in its own preferences.
16. **(age 0 · value low) A browser `Request` opens the device to answer.**
    There is no request API, so `getUserMedia` is the only thing that prompts,
    and a granted camera check has genuinely opened the camera for a moment. A
    refused one also reads as `denied` whether the user pressed Block or
    dismissed the prompt, because `NotAllowedError` does not say which.
17. **(age 0 · value low) The gomobile stub's types are unchecked.** The pin
    holds the *names* — every bindable symbol declared, nothing extra — and
    stops there, because copying gobind's type mapping into a test would be
    reimplementing gobind. A wrong signature fails the Swift typecheck the
    moment the shell calls it, so what is genuinely unguarded is a stub
    declaration the shell never touches.

Read by value instead: **high** 3 · **medium** 4, 5, 8, 9, 13, 14 · **low**
1, 2, 6, 7, 10, 11, 12, 15, 16, 17.
