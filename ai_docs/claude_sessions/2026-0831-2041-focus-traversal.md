# Session: Focus Traversal

**Session ID:** session_01LZZTYuh1UXxGjKfkomQRM6
**Date:** 2026-08-31, ~20:41
**Branch:** master
**Follows:** `2026-0831-2022-focus-commands-and-button-widening.md`

Last session closed programmatic focus and left the note: "Focus traversal is
still missing. Focus can now be set and observed, but there is no
next/previous order and no IME 'next' action that walks one. That is what is
left of the Keyboard-navigation roadmap item." This session closed it, and
with it the last open Keyboard-navigation line on the roadmap.

Two design questions were put to the user up front; both took the recommended
option: declare the order explicitly as an ordered list of refs, and derive
the keyboard's "next" action from that order rather than asking each field to
spell it out.

---

# Part 1 — Why the order is declared and not inferred

The obvious design is the order the fields *render* in — what a browser's tab
order and Compose's `moveFocus(FocusDirection.Next)` both use, and it needs no
declaration at all. It was rejected for one concrete reason:

**A field cannot know, while it is being built, whether anything comes after
it.** Render order is only complete once the pass ends, and the props are
stamped during the pass. So the IME action would have had to be declared per
field ("this one shows Next") — trading one line per *form* for one line per
*field*, and a form that half-traverses when a field is missed.

A declared order also survives the two things render order does not: a
`core.Cached` subtree (which never re-renders, so it never re-registers) and a
layout whose reading order is not child order — a two-column `Row` of fields.

The cost, stated where it bites: **membership is read while each field's props
are stamped**, so the declaration has to run before the fields it names. Hooks
belong at the top of a render function anyway and the fields of a form are
rendered by the component that owns their refs, so the natural shape is
already the correct one — but a call moved into a child that renders *after*
the fields would stamp a form that never advances. Documented in three places.

---

# Part 2 — Where traversal is decided, and how it travels

In Go, not on the platform. Compose could walk its own focus graph and SwiftUI
could chain `@FocusState`, but the order would then be *layout's* idea of it —
derived differently by each platform, and invisible to the code that declared
it.

So the Next key is **not a new bridge call**. It is the ordinary `onSubmit`
dispatch, which every renderer already sends:

```
UseFocusOrder(ctx, a, b, c)
        │  each ref records its position
        ▼
FocusTarget(a) stamps   imeAction : "next"     ── what the key reads
                        onSubmit  : cb_N       ── wired to Focus(b)
        ▼
renderer labels its action key from imeAction
        ▼
user presses it  →  dispatch cb_N  →  Focus(b)  →  an ordinary focus command
```

Advertising a Next key therefore costs **one string prop and no new wire
surface at all**. It rides on top of last session's focus commands rather than
beside them: traversal issues a `core.Focus`, and everything downstream — the
epoch, the three actions, the four renderers' handling — is already built.

## Three rules the encoding needed

**An explicit `onSubmit` wins.** A field built with `InputWithSubmit` has been
told, in so many words, what its return key does; silently replacing that with
"advance" would break the one call shape whose entire purpose is to say
otherwise. It also must not advertise a Next it is not going to perform — a
keyboard that lies. The consequence is that a middle field cannot both submit
and advance, which is not a shape a keyboard can express either: it has one
action key.

**No wrap-around.** The last field of a form submits; it does not send the
user back to the top. Walking off either end also consumes **no epoch**, which
matters: a command with no target would still re-stamp every focusable field
on screen.

**The `ordered` flag is sticky for the life of the app.** It is the traversal
half's equivalent of the epoch-0 sentinel. Until some `UseFocusOrder` has run,
`FocusTarget` writes no `imeAction` key at all, so an app that never declares
an order renders byte-identical trees to before. Once one has run anywhere,
every named field carries the key forever after — which is what stops the key
from ever *vanishing* between passes. That matters for the same reason the
focus stamp's two keys always travel together: an `update-props` patch carries
the whole new props map and the renderers iterate the keys it contains, so a
key that disappears is invisible on the far side, and a field dropped from an
order would otherwise keep advertising a "next" that no longer exists.

---

# Part 3 — The surface

```go
email    := core.UseFocusRef(ctx)
password := core.UseFocusRef(ctx)
confirm  := core.UseFocusRef(ctx)

core.UseFocusOrder(ctx, email, password, confirm)

core.Input(v, "you@example.com", onChange, core.FocusTarget(email))
core.InputPassword(p, "", onChange, core.FocusTarget(password))
core.InputPassword(c, "", onChange, core.FocusTarget(confirm))
```

One line. The fields already carried the `FocusTarget` that `core.Focus`
needed anyway.

**`UseFocusOrder` reserves no hook slot**, despite the name. Everything it
records lives on the refs, which are slot-stable already. The `Use` prefix
says where it belongs — inside a render function, on every pass — which is the
part a caller has to get right; the consequence of not being a real hook is
only ever permissive (calling it conditionally is safe). It *is* rebuilt every
pass on purpose: that is what lets a conditionally rendered field join and
leave the order as it appears and disappears.

**`FocusNext(ref)` / `FocusPrevious(ref)` take the field to move *from***,
rather than reading "the focused field". Go does not reliably know which field
that is — the framework wires `OnFocus` only where an app asked for it, so the
field the user tapped into is one Go was never told about (last session's
"cost, stated plainly"). Naming the source is honest about that, and every
caller has it: the keyboard action is stamped on a known field, and an
app-drawn toolbar tracks the current field with `OnFocus` if it wants one.

**`FocusPrevious` has no keyboard action behind it on any platform here.**
Neither the Android IME nor the iOS keyboard offers a "previous" key, and
SwiftUI gives no input-accessory toolbar for free. It exists for the toolbar an
app draws itself above a `KeyboardAware` region, which is where a
back-and-forth pair of arrows actually belongs.

Nil tolerance throughout, matching `MaybeProp`: a nil ref in the list is
skipped rather than taking a position, a nil ref to `FocusNext` is a no-op, a
nil context to `UseFocusOrder` returns. A ref belonging to another app's
context is skipped for the same reason `focusState` is per-app. A repeated ref
keeps its last position — a typo is not worth a panic, and last-wins is the
only rule that leaves every other field's neighbours intact.

---

# Part 4 — Renderers

**Android.** `ImeAction.Next` when the prop says so, `ImeAction.Done` when
only an `onSubmit` is present, `Default` otherwise — Next tested first because
it is the more specific claim, and Go only stamps it on a field whose
`onSubmit` it wired itself, so the two can never disagree. `KeyboardActions`
gained an `onNext` arm dispatching the same ID as `onDone`: Compose routes by
the action the field advertised, and Go has already decided what that ID means.
Deliberately *not* `LocalFocusManager.moveFocus`.

**iOS.** One line: `.submitLabel(imeAction == "next" ? .next : (onSubmit.isEmpty
? .return : .done))`. The existing `.onSubmit` already dispatches.

**WASM had no submit path at all** — this session added one. `onSubmit` rides
a `keydown` listener (an `<input>` outside a `<form>` has no submit event of
its own), filtered to Enter without Shift by a new `eventQualifies` predicate
threaded through both listener sites, with `preventDefault` so nothing else
acts on the keypress. `extractEventPayload` now returns void for `keydown` for
exactly the reason it already did for focus/blur: a `{value}` envelope would
route a void callback into the *text* callback map, where its ID does not
exist, and the handler would silently never run.

`enterkeyhint` is applied by a helper called **after** the prop loop on create
and **before** it on update, because the hint is a function of two props
(`imeAction` and `onSubmit`) and `Object.entries` fixes no order between them.
It is removed rather than set to `""` when neither asks for one: an empty
`enterkeyhint` is not a valid value.

**htmlout.** `enterkeyhint="next"` / `"done"`, plus `data-onsubmit` added to
the callback attribute table. Unlike the focus command — whose epoch says
*when*, a question a snapshot cannot ask — the hint is a **standing property**
of the field, which is exactly the kind of thing a static document carries.
The *behavior* still travels as the callback ID, because a soft keyboard's
hint only relabels the key and a hardware keyboard ignores it entirely.

A `TextArea` in an order does not advance on any platform: a multiline field's
return key inserts a newline, which is the right call everywhere.

---

# Part 5 — The example

`examples/signup` gained two more refs and the one-line order. The terms
checkbox is deliberately **not** in it: no platform here gives a checkbox
keyboard focus, so a fourth entry would advertise a Next that landed nowhere.

The docs' `LoginScreen` example lost its `InputWithSubmit(..., func() {
core.Focus(password) })` — the manual spelling of what `UseFocusOrder` now
does — and became a plain `core.Focus` demonstration instead.

---

# Testing

New: `core/focus_order_test.go` (16 tests), `render/focus_order_test.go` (4,
driving real Next presses through `render.Manager` and decoding the patch JSON
the way a native shell sees it), 3 in `htmlout/export_test.go`, 3 in
`examples/signup/app_test.go`.

**Mutation-checked: 33 breaks, 33 caught, no survivors on the first pass.**

The tests did prove one branch redundant. `focusNeighbor` guarded its target
against nil before calling `Focus`, but `Focus` documents nil as a no-op — so
the guard was provably dead code saying the same thing twice, and removing it
also removes the "walking off the end consumes no epoch" claim's second
implementation. Deleted, with the reasoning recorded where the guard was.

Caught: every arm of the state machine; the sticky `ordered` flag both ways;
indices assigned wrong, reversed, or not at all; `neighbor` walking backwards,
wrapping, ignoring membership, or collapsing to the first field; the last
field dropping its key or advertising Next; an explicit submit being
overwritten; a Next advertised without its submit and a submit wired without
its label; the Next callback refocusing its own field; `UseFocusOrder` keeping
nils, admitting foreign refs, retaining the caller's slice, or ignoring a nil
context; `FocusNext`/`FocusPrevious` swapped; `FocusTarget` not stamping
traversal at all; htmlout over- and under-exporting both hints and dropping
`data-onsubmit`; and the example declaring no order, ordering backwards, or
leaving a field unnamed.

The harness snapshots the four target files to the scratchpad and restores
from there, with a `changed()` guard reporting `NOT-APPLIED` rather than a
false verdict. **`git checkout` was not used to undo a mutation** — see two
sessions back on why.

## Files touched

`core/focus_order.go` (**new**), `core/focus_order_test.go` (**new**),
`core/focus.go` (the `ordered` flag, `FocusRef`'s membership fields, and
`FocusTarget`'s second stamp), `render/focus_order_test.go` (**new**),
`htmlout/export.go` + test, `examples/signup/app.go` + test,
`android/.../Renderer.kt`, `ios/GrMob/Runtime/Renderer.swift`,
`wasm/grmob-runtime.js`, `docs/concepts/events.md`, `ROADMAP.md`.

Gate: `gofmt` clean, `go vet` clean, full suite, `go test -race` on
core/components/render/forms/examples-signup, `GOOS=js GOARCH=wasm` build,
`node --check` on the runtime, and `ios/verify` (data-layer replay + Swift
typecheck) — all green. (`examples/todoapp/store.go` remains unformatted —
pre-existing, untouched.)

## Not verified here

**The Android side is unbuilt.** Still no Kotlin toolchain on this machine, so
`ImeAction.Next` and the `onNext` arm are checked by reading only. Whether
Compose routes a Next press to `onNext` rather than `onDone` in every field
configuration is something only an emulator will show.

**iOS type-checks but has not run.** Now owed from five sessions ago.
`.submitLabel(.next)`'s actual effect on the keyboard, and whether `.onSubmit`
fires identically for `.next` and `.done`, are runtime questions.

**The WASM Enter path is newly written and untested by anything but reading.**
There is still no JS test harness in the tree; `node --check` is syntax only.
This is the largest new gap in the session: the `keydown` listener, the
`eventQualifies` filter, the void payload and the `enterkeyhint` helper are all
new code that a browser would exercise in seconds.

## Backlog

In Progress still holds only Packaging (`grmob build --target=…`).

Closed this session: focus traversal, and with it the whole Keyboard
navigation roadmap item — the last open line under Extensions besides
router-style web navigation.

Still open from earlier sessions:

- Theme contrast, and `Variant`'s third consumer.
- **Hooks have no unmount signal.** A navigation frame is the first lifetime
  the framework can dispose of; component-level unmount remains unsolved.
- **`docs/concepts/architecture.md` does not mention frames.**
- **`core.Image` bases its style on `Theme.Components.Camera`.**
- The WASM runtime's style mapping is still much thinner than `htmlout`'s.
- **A bottom-docked bar has no way to ask for the keyboard on its own.**
- **`htmlout` renders `Scroll` as a plain div** with no `overflow`.
- **A second imperative API would justify the bridge command channel.**
  Traversal did *not* need one — it rode `onSubmit`, which already existed —
  so the arithmetic is unchanged; `scrollTo` or haptics would change it.
- **`core.SendSystemEvent` is a dead stub** — `core/toast.go` is its only
  caller, so `ShowToast` currently reaches nothing.
- **A `Cached` subtree silently swallows focus commands**, and now silently
  swallows order membership too. Documented, but the framework could plausibly
  detect it in debug mode.

Noticed this session, not acted on:

- **There is still no JS test harness.** Three sessions have now added
  renderer logic that only a browser can check, and this one added a whole
  event path. A jsdom or Playwright harness over `wasm/grmob-runtime.js` would
  pay for itself immediately.
- **An app-drawn keyboard toolbar has no worked example.** `FocusPrevious`
  exists for exactly that shape and nothing in the tree demonstrates it,
  which also means the `OnFocus`-tracks-the-current-field pattern is described
  in prose only.
- **`imeAction` is a third prop that must not vanish**, guarded by a third
  sticky sentinel. The rule is now stated in three files; a single helper that
  owns "props that must always travel once they start" would state it once.
