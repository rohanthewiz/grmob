# Session: Focus and Blur Events

**Session ID:** session_01LZZTYuh1UXxGjKfkomQRM6
**Date:** 2026-08-31, ~19:55
**Branch:** master
**Follows:** `2026-0831-1932-required-marker-and-keyboard-aware.md`

The backlog item that had been open longest: *"No focus or blur event anywhere
in the framework."* It named two things it was blocking. This session did the
first — `RevealOnTouch` keying on "has been edited" because "has been left" was
not observable — and left the second (dismissing the keyboard on a tap outside)
open, because it needs a different capability.

Two design questions were put to the user up front; both took the recommended
option: widen the leaf builders rather than add paired `WithX` constructors,
and let `forms` consume the new events rather than stopping at core.

---

# Part 1 — The events

## Two props, not one `func(bool)`

`core.OnFocus` and `core.OnBlur` are ordinary void `BehaviorProp`s built on the
existing `On(event, handler)`. Nothing new reaches the bridge: the edge itself
is the whole payload, and *which node* is already answered by which callback ID
the platform dispatches, so both ride the same void channel `onClick` and
`onSubmit` use.

A single bool-carrying handler was the alternative and it is worse. The two
edges are almost never handled together — a form reveals errors on blur and
does nothing on focus; a search box does the opposite — so one prop per edge
lets a node carry only the edge it wants instead of registering a callback that
ignores half its calls.

## The ordering caveat is real and is documented, not papered over

The framework guarantees the edges arrive in the order they happened. It does
**not** guarantee that the blur on the field being left precedes the focus on
the field being entered — that ordering belongs to the platform, and Android
and iOS disagree. So a handler must read the field its callback belongs to and
never ask "what is focused *now*". Both the doc comment and `docs/concepts/events.md`
say so.

# Part 2 — `leafNode`, and the widening that made the props reachable

`BehaviorProp` only ever reached nodes built by `containerNode`. The input
builders took `...StyleProp`, so there was literally no way to spell "this
field, on blur" — the props would have existed and been unattachable.

`core.Input` and the rest now take `...PropsAndChildren` through a new
`leafNode` helper: the container argument list minus the children. It is the
same widening `Scroll` got last session and source-compatible for the same
reason (a `StyleProp` is a `PropsAndChildren`), and it means every future event
prop lands on the inputs for free rather than spawning another `WithX` builder.

Three contracts, each mirroring `containerNode`'s:

- **Ordering.** The builder's own prop map (value, placeholder, its `onChange`
  ID) is populated *before* the items run, so a builder's intrinsic callbacks
  always hold the lower IDs of the pass and no argument a caller writes can
  move them. The items then register in argument order.
- **Nil.** A nil item is `MaybeProp`'s false path and leaves no trace.
- **A `View` is the one shape `containerNode` accepts and this cannot.** A leaf
  has nowhere to put a child, so it raises `ConcernUnknownItem` rather than
  vanishing.

## A first cut had a two-pass loop; it was deleted

The original `leafNode` applied style props in one pass and behavior props in a
second, with a comment claiming this kept callback IDs stable under
interleaving. Writing the mutation for it showed the claim was false: in a
single pass the behavior props still register in relative argument order and
the style props still apply in argument order, so the split bought nothing
observable. It was collapsed to one loop matching `containerNode`, and the
comment with it — the "dead precision" two sessions of notes have warned about.

## The one call shape the widening does break

Go will not spread a `[]StyleProp` into a `...PropsAndChildren`. Direct calls
with literal arguments are unaffected; a wrapper that *collected* style props
into a slice is not. `forms`' bound builders were the only such wrapper in the
tree and their own parameter widened to `[]core.PropsAndChildren`. This is
named in `leafNode`'s doc comment, in `forms/inputs.go`, and in
`docs/concepts/events.md`, because it is the kind of break that otherwise gets
rediscovered by whoever writes the next wrapper.

---

# Part 3 — `forms.RevealOnBlur`

## The policy

Inserted between `RevealOnSubmit` and `RevealOnTouch`, which is the natural
"reveals latest → earliest" progression. Safe because the constants are never
serialized and `revealed` compares with `==` only.

It is the closest thing to the package's own "reward early, punish late" rule
that still says something before the submit. Leaving a field is the user's own
claim to have finished with it, so a complaint then is an answer rather than an
interruption — where `RevealOnTouch` fires on the *second keystroke* of an
address, and the default makes the user fill in four fields before hearing
about the first.

`revealed`'s switch deliberately has no default arm: `RevealOnSubmit` is the
zero value and its whole answer is the `submitted` check above it, so falling
through to false is exactly right, and a policy added later without a case here
fails closed rather than leaking every error.

## `blurred` is a separate map from `touched`

They answer different questions — "has been edited" versus "has been finished
with" — and a field can satisfy either without the other: tabbing straight
through blurs an untouched field, and a field still under the cursor is touched
but not blurred. Merging them would make each policy fire on the other's
occasions.

`MarkBlurred` therefore does **neither** of `SetValue`'s two side effects. It
does not mark touched (a tab-through has not edited anything) and does not drop
the external error (leaving a field changes no text, so a server's verdict on
that text still stands).

It requests a render only on a field's *first* blur. Every later one re-asserts
a flag that is already set and can change nothing on screen — which matters
because the platform dispatches blur on every focus change, including all the
ones a user makes moving back and forth through a form they have already been
round once.

## The binding is gated on the policy

`blurProp` returns the `core.OnBlur` binding under `RevealOnBlur` and `nil`
otherwise. Always attaching would not be free: a callback registered, a prop
the reconciler diffs, and — the part that actually matters — both native
renderers dispatching a Go event on every focus change, each waking a render
pass that under `RevealOnSubmit` can change nothing. The framework should not
manufacture traffic no policy reads.

The `nil` return is load-bearing rather than a sentinel: `leafNode` skips a nil
item, so an unwired field renders exactly the node it always did.

`Blurred(name)` reports what has been *observed*, not what happened. Under a
policy that wires no blur it stays false, which is the honest answer, and the
doc comment says so rather than leaving it to be read as a bug.

`Form.Checkbox` takes no binding: a tick is a commit, not a draft, so there is
no "still working on it" state for leaving it to end — and neither native
platform gives a checkbox keyboard focus. A required-unticked box is revealed
by the submit, like any field the user never visited.

---

# Part 4 — Renderers

**Android.** `GrMobTextField` already had `focused` from
`collectIsFocusedAsState`. The dispatch goes in a `LaunchedEffect(focused)`
rather than the composition body, because sending an event to Go wakes a render
pass and a composable's body may run any number of times for one logical
change. A `seenFocus` flag keeps mount quiet: `collectIsFocusedAsState` emits an
initial `false`, so without it every text field on screen would fire `onBlur`
the moment it appeared.

**iOS.** The existing `.onChange(of: focused)` — which already seeded the local
buffer — gained the two dispatches. The seeding stays first: a dispatch into Go
can land a render before the closure returns, and the buffer must already agree
with upstream when it does. No mount guard is needed, because `onChange(of:)`
fires on a *change* and not on the initial value. The asymmetry with Compose is
commented on both sides.

**WASM.** This one was a real bug, not just wiring. `extractEventPayload`
returned `{value: e.target.value}` for *any* event on an `<input>`. A focus
event would therefore have been sent as a string, and Go — seeing a string in
`ReceiveEventPayload` — would have dispatched it to the **text** callback map,
where a void ID does not exist. The handler would have silently never run. The
focus/blur guard is checked before the type test for exactly that reason.
`mapEventName` also lists both explicitly even though its fallback derives the
same names, because these are the two DOM events that do not bubble and the
listener has to sit on the element itself.

**htmlout.** Two more entries in the attribute mapping. Exported for every node
type that carries them, not just inputs: a browser gives focus to more than a
phone does (a link, anything with `tabindex`) and a static export has no
business narrowing that.

---

# Testing

New: `core/focus_test.go` (11 tests, incl. a table over every `leafNode`
builder), 13 in `forms/form_test.go`, 3 in `examples/signup/app_test.go`
driving real `onBlur` IDs through `render.Manager`, 2 in
`htmlout/export_test.go`.

**Mutation-checked: 18 breaks, 17 caught first pass.** The miss was `blurProp`
binding a hard-coded `"email"` — the test that should have caught it happened
to render that same field. Fixed by running it over every field name and
asserting the siblings stay unmarked.

Caught: `revealed` keying on touched; `MarkBlurred` marking touched; it
clearing the external error; `Reset` leaving the marks; never repainting;
repainting on every blur; `blurProp` ungated; `blurProp` never attaching;
`Form.OnBlur` a no-op; `leafNode` ignoring behavior props, losing its nil arm,
discarding the builder's props, swallowing a `View`; `OnFocus` writing the
`onBlur` prop; `On` replacing the props map; htmlout dropping the attributes
and emitting a wrong name.

## The mutation harness destroyed the working tree once

The first harness reverted each mutation with `git checkout -- .`, which
reverted **every uncommitted change** — the whole session's work on tracked
files. Only the untracked `core/focus_test.go` survived. Everything was rewritten
from context, and the pass re-run with a harness that snapshots the six target
files to the scratchpad and restores from there, plus a `changed()` guard so a
mutation that failed to apply reports `NOT-APPLIED` instead of a false verdict.

Results 2–8 of the first run were invalid (they ran against restored code with
a stale test file, so every one "failed" for a compile error) and were
discarded. **Never use `git checkout` to undo a mutation in a dirty tree.**

---

## Files touched

`core/behavioral_props.go`, `core/layout.go` (leafNode), `core/input.go`,
`core/focus_test.go` (**new**), `forms/form.go`, `forms/inputs.go`,
`forms/form_test.go`, `htmlout/export.go` + test, `examples/signup/app.go` +
test, `android/.../Renderer.kt`, `ios/GrMob/Runtime/Renderer.swift`,
`wasm/grmob-runtime.js`, `docs/concepts/events.md`, `docs/concepts/views.md`,
`docs/concepts/forms.md`, `ROADMAP.md`.

Gate: `gofmt` clean, `go vet` clean, full suite, `go test -race` on
core/components/render/forms, `GOOS=js GOARCH=wasm` build, `node --check` on
the runtime, and `ios/verify` (data-layer replay + Swift typecheck) — all
green. (`examples/todoapp/store.go` remains unformatted — pre-existing,
untouched.)

## Not verified here

**The Android side is unbuilt.** No Kotlin toolchain on this machine, so the
`LaunchedEffect` and its `seenFocus` guard are checked by reading only. This
one wants an emulator pass more than most: the mount-quiet behavior is the
whole reason the flag exists.

**iOS type-checks but has not run.** Still owed from three sessions ago.

**The WASM focus path is untested by anything but reading.** There is no JS
test harness in the tree; `node --check` is syntax only. The payload bug it
fixes is exactly the kind a browser would have shown in a second.

## Backlog

In Progress still holds only Packaging (`grmob build --target=…`).

Closed this session: the focus/blur half of the old event gap, and
`RevealOnTouch`'s "has been left is not observable" limitation.

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

- **Focus is observable but not settable.** There is no `core.Focus(ref)` and
  no `DismissKeyboard()`, so the second half of the old backlog item — nothing
  can dismiss the keyboard on a tap outside — is still open. It needs
  node identity or a focus ref, which the framework does not have; the same
  missing piece the identity-keyed callback IDs want.
- **`core.Button` still takes `...StyleProp`.** It is now the one remaining
  leaf that cannot carry a behavior prop. Left alone as out of scope; a phone
  does not focus a button, but the inconsistency is real.
- **Neither native renderer wires focus on `Checkbox`.** Correct today (no
  platform focus there) but the prop is accepted, so a caller can attach a
  handler that never fires.
