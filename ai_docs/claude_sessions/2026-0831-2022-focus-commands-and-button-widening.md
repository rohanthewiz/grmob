# Session: Focus Commands and Button's Widening

**Session ID:** session_01LZZTYuh1UXxGjKfkomQRM6
**Date:** 2026-08-31, ~20:22
**Branch:** master
**Follows:** `2026-0831-1955-focus-and-blur-events.md`

Last session made focus *observable* and left a note: "Focus is observable but
not settable. There is no `core.Focus(ref)` and no `DismissKeyboard()`, so the
second half of the old backlog item — nothing can dismiss the keyboard on a
tap outside — is still open." This session closed it, and took `core.Button`
— the last leaf still on `...StyleProp` — along with it.

Two design questions were put to the user up front; both took the recommended
option: ride the render tree with an epoch-stamped prop pair rather than
building a new bridge command channel, and acquire refs through a hook.

---

# Part 1 — The channel, which is the whole design

Go has exactly one way to reach a screen: the render tree and the patches that
update it. There is no imperative Go→native call. So a focus command is
expressed the only way anything reaches a screen here — as props:

```
Focus(ref) / DismissKeyboard(ctx)
        │  epoch++, target = ref (or nil)
        ▼
next render pass: every focusable leaf stamps
        "focusEpoch"  : N       ── the command generation
        "focusAction" : "focus" │ "blur" │ ""
        ▼
reconciler emits update-props for the leaves whose stamp changed
        ▼
renderer keys on focusEpoch *changing* and runs focusAction once
```

The alternative on the table was a real command channel — the Manager draining
a pending-command queue emitted alongside the patches. It has cleaner
semantics and would give future imperative APIs (scrollTo, haptics) a home,
but it changes the patch wire format, so all three renderers plus
`RenderInitial`'s tree-vs-array shape would have been reworked, and it widens
the gomobile bind surface. Rejected as too much for one feature; the note
stands if a second imperative API ever wants it.

## Three things that drove the encoding

**The epoch is a counter, not a flag.** Two identical prop maps produce no
patch and therefore no effect, so focusing the already-focused field — the
"try again after a failed submit" case — would silently do nothing. Only a
changed value can express a repeat.

**Three actions, not a bool.** The obvious encoding is `focused: true` on the
target and `false` on everyone else, and it is wrong on both platforms.
Focusing B while A holds focus would have A clearing and B requesting in an
order neither Compose (`LaunchedEffect`) nor SwiftUI (`onChange`) guarantees,
so B's request can land first and A's clear then throws it away. Saying what
each node should *do* removes the race: non-targets are told `""` and do
nothing, because requesting focus over there already takes it from here.

**Both keys always travel together.** An `update-props` patch carries the
*whole new props map* and the renderers iterate the keys it contains, so a key
that disappears between passes is invisible on the far side. A node that once
carried `focusAction` must keep carrying it — with the value `""` when it has
nothing to do — rather than dropping it. The test helper `stamp()` panics on
half a stamp for exactly this reason, because every individual assertion would
still pass without it.

## The cost, accepted deliberately

Every command re-stamps every focusable leaf, so a dismiss with six fields on
screen emits six `update-props` patches. The alternative — tracking which node
holds focus in Go so only one needs stamping — would require wiring `OnFocus`
on every field always, waking a render pass on every keyboard change. Six
cheap patches beat continuous traffic, and commands are rare and
user-initiated.

Until the first command, `stampFocus` writes **nothing at all**. An app that
never touches focus renders byte-identical trees to before, which is what
keeps the feature free for every existing screen and every existing test.

## The one caveat

A `core.Cached` subtree returns the identical `*Node` every pass and the
reconciler treats pointer equality as proof nothing changed, so a field inside
one never re-stamps and never hears a command. Documented in `core/focus.go`
and in `docs/concepts/events.md`: cache above the focus system or not at all.

---

# Part 2 — The surface

```go
email := core.UseFocusRef(ctx)

core.Input(v, "you@example.com", onChange,
    core.FocusTarget(email),
)
core.Button("Next", func() { core.Focus(email) })
core.Box(core.OnClick(func() { core.DismissKeyboard(ctx) }), form)
```

**`UseFocusRef` is a hook, not a constructor.** Identity is the entire content
of a `FocusRef` — the pointer *is* the name — so a ref built inline in a
render function is a new pointer every pass: `FocusTarget` would stamp one
identity and `Focus` would compare against another, and the field would
silently never focus. Going through `NewState` pins the pointer and reserves
the cursor slot properly. It lives in core rather than hooks because hooks
imports core and not the reverse.

**`DismissKeyboard` takes a `*Context` where `Focus` does not.** This is the
one deviation from the shape the design question previewed. A dismiss names no
node and therefore has no ref to carry the app instance, and reaching for a
package-level global would recreate exactly the bug `Context`'s shared-pointer
block documents as already fixed: two apps in one process (or two managers in
one test binary) sharing one keyboard. `focusState` is app-instance state on
Context, copied by pointer at all four construction sites, with tests on both
halves of that rule.

**Nil is a no-op, not a panic**, for `Focus`, `DismissKeyboard` and
`FocusTarget` alike — matching `MaybeProp`'s contract, since `FocusTarget`
returning a nil `BehaviorProp` is skipped by `leafNode`.

`focusableLeafTypes` is the four text inputs. `Checkbox` and `Button` go
through `leafNode` but are deliberately absent: neither platform gives them
keyboard focus, so a stamp would be a patch per command that nothing reads.
`FocusTarget` still stamps them if an app insists, which is why it writes the
complete pair itself rather than relying on `leafNode`.

---

# Part 3 — `core.Button`'s widening

`Button` was the last leaf on `...StyleProp`, which made the one node type
that exists to be pressed the one that could not carry a gesture prop —
`OnLongPress` was unreachable on it. It now goes through `leafNode` like the
inputs, and so does `ButtonWithEvent`.

Source-compatible for the same reason the inputs' widening was: a `StyleProp`
is a `PropsAndChildren`. The shape it breaks is forwarding, and there were
exactly three sites, all of which *collected* into a `[]core.StyleProp`:
`components.Button`, `components.Chip`, and a legacy-markup comparison test in
`examples/todoapp`.

**The widgets' `Style []core.StyleProp` fields stayed narrow on purpose.** A
`Button{}` places its styles at a documented point in a treatment order it
controls, and it has nowhere sensible to put a behavior prop — accepting one
would mean silently deciding whether it lands before or after the variant
colors. So the two widgets convert at the call site through a small `asProps`
helper that states that reasoning once, and callers who need a behavior prop
reach for `core.Button` directly.

`Button` was also added to `core/focus_test.go`'s `leafBuilders` table, which
now runs the whole leaf argument contract over all seven builders.

---

# Part 4 — Renderers

**Android.** A `FocusRequester` per field, appended after the box modifier so
it sits on the innermost element rather than the padding, driven by
`LaunchedEffect(focusEpoch)`. Keyed on the epoch alone, never the action — the
action is *what*, the epoch is *when*.

Unlike the edge-dispatch effect above it, this deliberately has **no mount
guard**. A field that mounts while it is already the target takes focus, which
is what makes "push a screen and put the cursor in its search box" work: the
command is issued in the handler that navigates, one pass before the field it
names exists. The cost — returning to that screen re-applies the command — is
documented rather than papered over.

`requestFocus()` is wrapped in `runCatching`: it throws if the requester is
not attached to a placed node, which `LaunchedEffect` covers for the ordinary
case but not for a field composed inside a lazy list that has not laid the row
out. "The command missed" is the honest outcome there; a crashed screen is not.

**iOS.** Two modifiers where Compose needs one. `onChange(of:)` does not fire
for the initial value — the same asymmetry that lets this renderer skip
Compose's `seenFocus` flag — so a field mounting *as* the target would never
hear its command without an `onAppear`. Both call one `applyFocusCommand`. The
asymmetry is commented on both sides, as the edge dispatches already were.

**WASM.** `focus()` / `blur()` deferred one frame, because on the initial
render `createElement` builds the tree detached (mount appends it only once
assembled) and `focus()` on a node outside the document is a silent no-op. The
update path stores the epoch on the element's dataset and compares: an
`update-props` patch carries the entire new map, so a field re-rendered for
its *value* would otherwise re-run whatever command was last issued. A bare
`focusAction` key is explicitly skipped — on its own it says *what* and not
*when*.

**htmlout.** A standing `"focus"` exports as `autofocus="autofocus"`, long
form to match `disabled` and `checked`. Neither raw prop reaches the document:
the epoch asks *when*, which a snapshot cannot ask. `"blur"` and `""` export
as nothing, because a freshly loaded page has no focus to release.

---

# Part 5 — The example

`examples/signup` gained the natural consumer. Its server-error path already
showed "That address is already registered" — but a submit has usually
scrolled past the email field or closed the keyboard on it, so the message
alone left the user to find the field again. `core.Focus(emailField)` both
reopens the keyboard and brings the field into view, because the platform
scrolls to whatever it focuses.

Two tests on that path, including one asserting a *successful* submit issues
no command at all — the form is gone by then and a stray command would land on
the confirmation screen.

---

# Testing

New: `core/focus_command_test.go` (20 tests), `render/focus_command_test.go`
(3, driving real commands through `render.Manager` and decoding the patch JSON
the way a native shell sees it), 2 in `htmlout/export_test.go`, 2 in
`examples/signup/app_test.go`, plus `Button` added to the `leafBuilders` table.

**Mutation-checked: 28 breaks, 28 caught** — after a first pass with 2
survivors, both worth recording:

**A real gap.** "DismissKeyboard does not clear a stale target" survived
because *every* dismiss test dismissed without a prior focus, so the target
was already nil. That bug would have had a dismiss **re-focus** the field the
app last focused — the exact opposite of what was asked for. Fixed by
`TestDismissAfterFocusReleasesTheNamedField`.

**An equivalent mutant.** `epoch == 0` meant both "no command has ever been
issued" and the wire sentinel, which made the guard in `command()` provably
redundant with the one in `stampFocus`. Rather than leave it, `command()` now
returns a named `issued bool`: zero stays the sentinel on the wire (each
renderer checks it) while the Go side asks the question it actually means.
That killed the mutant and reads better.

Caught: every arm of the state machine; the epoch not advancing on either
command; `stampFocus` writing one key without the other; both commands
skipping `RequestRender`; `UseFocusRef` allocating per pass; `FocusTarget`
accepting nil or not stamping; `Checkbox` gaining and `TextArea` losing
focusability; `leafNode` never stamping, stamping every leaf, or stamping
after the items run; `Button` dropping its onClick or discarding its props;
`ButtonWithEvent` wiring a click; child contexts getting their own focus
state; htmlout over- and under-exporting autofocus; `components.Button`
dropping caller styles; `asProps` truncating.

The harness snapshots the seven target files to the scratchpad and restores
from there, with a `changed()` guard reporting `NOT-APPLIED` rather than a
false verdict. **`git checkout` was not used to undo a mutation** — see last
session's note on why.

---

## Files touched

`core/focus.go` (**new**), `core/focus_command_test.go` (**new**),
`core/context.go` (the `focus *focusState` pointer at all four construction
sites), `core/layout.go` (leafNode's focus contract), `core/button.go`,
`core/keyboard.go` (a claim this change made false), `core/focus_test.go`,
`components/button.go` (+ `asProps`), `components/chip.go`,
`htmlout/export.go` + test, `render/focus_command_test.go` (**new**),
`examples/signup/app.go` + test, `examples/todoapp/chip_migration_test.go`,
`android/.../Renderer.kt`, `ios/GrMob/Runtime/Renderer.swift`,
`wasm/grmob-runtime.js`, `docs/concepts/events.md`, `docs/concepts/views.md`,
`ROADMAP.md`.

Gate: `gofmt` clean, `go vet` clean, full suite, `go test -race` on
core/components/render/forms/examples-signup, `GOOS=js GOARCH=wasm` build,
`node --check` on the runtime, and `ios/verify` (data-layer replay + Swift
typecheck) — all green. (`examples/todoapp/store.go` remains unformatted —
pre-existing, untouched.)

## Not verified here

**The Android side is unbuilt.** Still no Kotlin toolchain on this machine, so
the `FocusRequester`, the `LaunchedEffect(focusEpoch)` and the `runCatching`
are checked by reading only. The mount-focus behavior and the lazy-list
`requestFocus` throw are both things only an emulator will show.

**iOS type-checks but has not run.** Now owed from four sessions ago. The
`onAppear` + `onChange` pairing in particular is the kind of thing SwiftUI
settles at runtime.

**The WASM focus path is untested by anything but reading.** There is still no
JS test harness in the tree; `node --check` is syntax only. The
`requestAnimationFrame` deferral and the dataset epoch guard would both show
themselves in a browser in seconds.

## Backlog

In Progress still holds only Packaging (`grmob build --target=…`).

Closed this session: programmatic focus, the tap-outside-dismisses-keyboard
half of the old event gap, and `core.Button`'s `...StyleProp` inconsistency.

Still open from earlier sessions:

- Theme contrast, and `Variant`'s third consumer.
- **Hooks have no unmount signal.** A navigation frame is the first lifetime
  the framework can dispose of; component-level unmount remains unsolved.
- **`docs/concepts/architecture.md` does not mention frames.**
- **`core.Image` bases its style on `Theme.Components.Camera`.**
- The WASM runtime's style mapping is still much thinner than `htmlout`'s.
- **A bottom-docked bar has no way to ask for the keyboard on its own.**
- **`htmlout` renders `Scroll` as a plain div** with no `overflow`.

Noticed this session, not acted on:

- **Focus traversal is still missing.** Focus can now be set and observed, but
  there is no next/previous order and no IME "next" action that walks one.
  That is what is left of the Keyboard-navigation roadmap item.
- **A second imperative API would justify the bridge command channel.** This
  one rode the tree because a wire-format change was too much for one feature;
  `scrollTo` or haptics arriving would change that arithmetic, and the queue
  design is written up in this session's question.
- **`core.SendSystemEvent` is a dead stub** — `core/toast.go` is its only
  caller, so `ShowToast` currently reaches nothing. Found while looking for an
  existing imperative channel.
- **A `Cached` subtree silently swallows focus commands.** Documented, but the
  framework could plausibly detect it in debug mode: a cached node whose
  subtree contains a focusable leaf is nearly always a mistake once focus
  commands are in play.
