# Session: The Required Marker, and Keyboard-Aware Regions

**Session ID:** session_01LZZTYuh1UXxGjKfkomQRM6
**Date:** 2026-08-31, ~19:32
**Branch:** master
**Follows:** `2026-0831-1855-forms-with-validation.md`

Two items, both from the previous session's backlog: the required marker
`FormField` still lacked, and the ROADMAP's last open UI-DSL entry,
"keyboard-aware scroll area for mobile".

---

# Part 1 — The required marker

Last session ended with: *"`FormField` has no required marker. Purely
cosmetic, but now that `Required` exists as a rule the two would want to
agree."* The whole design question is that last clause.

## The marker is a node, not a suffix

`FormField.Required bool` draws `*` after the label. It is a sibling `Text`
rather than `" *"` glued onto the label string, because being its own element
is what lets it do the two things that matter: carry `Colors.Error` while the
label keeps primary ink, and carry `AccessibilityLabel("required")` so a
screen reader says "required" instead of spelling out a star.

The pairing `Row` is built **only** on that branch, so an optional field
renders the exact tree it did before — one `Text`, no wrapper for a native
renderer to lay out. The Row zeroes the theme Row's screen-level padding
(8/16) for the same reason the field's Column does.

Ignored when `Label` is empty; there is nothing to mark.

## The agreement problem: requiredness is derived, not declared

`forms.Form.Required(name) bool` runs the field's rules against `""` and
reports whether any of them complains.

There is deliberately **no `Field.Required` flag**. A flag is a second claim
about the same field, true only while someone keeps it in step with the
rules, and the drift it permits is exactly what the marker exists to prevent:
a starred field that submits empty, or an unstarred one the user cannot get
past. It is the same reasoning that keeps the error map derived — see last
session's package doc.

Four consequences, three of them free:

- **Any rule that speaks about `""` counts**, not only `Required`. `Accepted`
  does — an unticked box is `"false"`, never `""` — so a terms checkbox reads
  as required, which it is. An app's own closure rejecting `""` counts too:
  nothing here recognises rule identities, only behavior.
- **Rule order does not matter.** Required-first is advice; the probe tries
  every rule.
- **The answer is as live as the rules.** A rule that only applies in some app
  state takes its marker with it, no bookkeeping.
- **`Spec.Validate` is not consulted.** A cross-field requirement is not a
  property of the field, and the probe has no other field's value to hand it.
  Mark those by hand.

No lock is taken: it reads this pass's spec and calls rules, which for the
reasons in `derived` must never run under the record's mutex.

## Call site

`examples/signup` asks the form rather than writing `true`:

```go
components.FormField{
    Label:    "Email",
    Required: form.Required("email"),
    Error:    form.Error("email"),
    Input:    form.Input("email", "you@example.com"),
}
```

The terms checkbox stays unmarked — its label belongs to the `ListRow` — with
a comment noting the field *is* required and `form.Required("terms")` would
say so.

## Testing

New: 4 tests in `forms/form_test.go`, 3 in `components/form_field_test.go`,
1 end-to-end in `examples/signup/app_test.go` (exactly 3 markers in the tree).

**Mutation-checked: 8 breaks, all caught first pass** — probe reading the live
value instead of `""`, scan stopping at the first *passing* rule, unknown name
defaulting to true, marker drawn for optional fields, marker in label ink, row
keeping theme padding, missing accessibility label, marker rendered with no
label. Two test strengthenings came out of writing them: a field ordering
`Email` before `Required` (so the scan cannot stop early unnoticed), and an
assertion that a field whose *current value* is merely invalid is not
required.

---

# Part 2 — Keyboard-aware regions

## `core.Scroll` became a real container first

`Scroll` took a bare `...View` while both native renderers had always applied
a Scroll node's style (`boxModifier`, `grMobBox`) — Go simply had no way to
set one. It now goes through `containerNode` like Row/Column/Box.

Source-compatible: a `View` is a `PropsAndChildren`, so both existing call
sites compile untouched. No theme base, like `Box` — a themed base would put
the Column's screen padding on every scrolling region, insetting content the
screen has already inset. It joined the container-behavior-props contract
test, which is the proof the widening is real.

## What the prop means, in two shapes

`core.KeyboardAware()` writes one prop. What it does depends on what it is on:

| on a… | effect | the case it serves |
|---|---|---|
| scrolling node (`Scroll`, `List`) | the **viewport** shortens | a form: the platform's scroll-the-focused-field-into-view now has somewhere visible to land it |
| anything else | the subtree **lifts whole** | a docked composer or checkout bar — outside any scroll region by construction, and the one thing the keyboard covers |

The first is the roadmap item as written. The second is what made the design
work at all, and it arrived by way of a near-regression (below).

`components.Screen.KeyboardAware` puts it on the `Scroll` when there is one
and on the content column otherwise — **never on the `SafeArea`**, which would
pull the header off the top and, on Android, consume the inset before any
inner region could ask for it.

## The design turned on a question about SafeArea

The first cut restricted the prop to scrolling nodes and raised a debug
concern (`ConcernInertProp`) anywhere else. That fell apart on `examples/chat`.

Android needs `enableEdgeToEdge()` for `WindowInsets.ime` to report anything
at all — while the decor view still fits system windows, the IME inset arrives
already consumed and `imePadding()` is a no-op. But going edge-to-edge makes
`safeDrawingPadding()` on the `SafeArea` node start doing its job, and
`WindowInsets.safeDrawing` **bundles the IME in with the system bars**. So:

- Leave `SafeArea` on `safeDrawing` → every screen resizes whole, *and* the
  inset is consumed there, so a Scroll asking for it receives nothing (nested
  window-inset modifiers consume what they apply). The flag becomes a no-op
  on Android in every case that goes through `components.Screen`.
- Subtract the IME → the flag works, but chat's composer sits behind the
  keyboard with nothing able to say otherwise.

The resolution is the second shape of the prop. `SafeArea` is the system bars
and the cutout, not the keyboard — the same split SwiftUI makes, where the
keyboard is its own safe-area region and `.ignoresSafeArea(.keyboard)` exists
because of it. The keyboard then belongs to whichever node asked for it, and
chat asks for it on its column.

That also deleted `ConcernInertProp` and its tests: with the Android inset
injected at the render funnel, no node type is inert, and an unused concern
kind is the "dead precision" the last session's mutation pass warned about.

## Renderers

**Android.** `Modifier.imePadding()` injected once in `RenderNode`, the single
funnel every node passes through, rather than at each container that might
want it. Position matters: it lands between the parent-scope modifiers and the
node's own (`boxModifier` starts from `extra`), so for a Scroll it precedes
`verticalScroll`. Constraints flow left to right, so an inset written *before*
the scroll shrinks the viewport; written after, it would pad the scrolled
content and leave the viewport still claiming the rows the keyboard covers.

Two window-level changes were required for any of it to be observable:

```
MainActivity          enableEdgeToEdge()
AndroidManifest.xml   android:windowSoftInputMode="adjustResize"
```

and `SafeArea` moved from `safeDrawingPadding()` to
`windowInsetsPadding(WindowInsets.safeDrawing.exclude(WindowInsets.ime))`.

**iOS.** SwiftUI already insets a `ScrollView` for the keyboard by itself, so
the shrink is the platform default with or without the flag. What the flag
adds is `.scrollDismissesKeyboard(.interactively)` on the two scrolling node
types — `.interactively` rather than `.immediately` so a scroll that only
meant to peek at the field above does not cost the user their keyboard.

**HTML / WASM.** Nothing. A browser has no software keyboard to inset for.

That asymmetry is the reason this is a flag and not something `Scroll` always
does: free on iOS, a window-level opt-in plus a per-region decision on
Android, and a Go app should name the region without knowing either.

## Testing

New: `core/keyboard_test.go` (every container carries the prop; a prop written
into an existing props map does not clobber a handler registered before it;
the default carries no prop; Scroll's style is the zero Style), 3 in
`components/screen_test.go` (scroll case, column case, and *exactly one* node
marked — two nested insets would take the keyboard's height out twice on any
renderer that does not consume insets as Compose does).

**Mutation-checked: 6 breaks, all caught** — prop overwriting the props map,
Scroll gaining the theme Column base, Screen marking the column as well when
scrolling, Screen never marking the column, Screen never marking the scroll
region, and the flag leaking onto screens that did not ask (MaybeProp
dropped).

---

## Files touched

**Part 1:** `components/form_field.go` + test, `forms/form.go` + test,
`examples/signup/app.go` + test, `docs/components.md`,
`docs/concepts/forms.md` (new "The required marker" section).

**Part 2:** `core/keyboard.go` + `core/keyboard_test.go` (**new**),
`core/layout.go` (Scroll), `core/container_behavior_test.go`,
`components/screen.go` + test, `examples/signup/app.go`,
`examples/chat/main.go`, `android/.../Renderer.kt`,
`android/.../MainActivity.kt`, `android/app/src/main/AndroidManifest.xml`,
`ios/GrMob/Runtime/Renderer.swift`, `docs/concepts/views.md`,
`docs/components.md`, `ROADMAP.md`.

Gate: `gofmt` clean, `go vet` clean, full suite, `go test -race` on
core/components/render/forms, `GOOS=js GOARCH=wasm` build, and `ios/verify`
(data-layer replay + Swift typecheck of the view layer) — all green.
(`examples/todoapp/store.go` remains unformatted — pre-existing, untouched.)

## Not verified here

**The Android side is unbuilt.** No Kotlin toolchain on this machine, so
`Renderer.kt`, `MainActivity.kt` and the manifest are unchecked by anything
but reading. And the edge-to-edge switch changes inset handling for *every*
Android screen, not only keyboard-aware ones — it wants an emulator pass
before it is trusted.

**iOS type-checks but has not run.** Still owed from two sessions ago.

## Backlog

In Progress now holds only Packaging (`grmob build --target=…`).

Still open from earlier sessions:

- Theme contrast, and `Variant`'s third consumer.
- **Hooks have no unmount signal.** A navigation frame is the first lifetime
  the framework can dispose of; component-level unmount remains unsolved.
- **`docs/concepts/architecture.md` does not mention frames.**
- **`core.Image` bases its style on `Theme.Components.Camera`.**
- The WASM runtime's style mapping is still much thinner than `htmlout`'s.
- **No focus or blur event anywhere in the framework.** It now blocks two
  things: `RevealOnTouch` keys on "has been edited" because "has been left" is
  not observable, and nothing can dismiss the keyboard on a tap outside.

Noticed this session, not acted on:

- **A bottom-docked bar has no way to ask for the keyboard on its own.** It
  rides its screen's column, which is fine when the screen has one — but a
  bar inside a scrolling screen has no answer.
- **`htmlout` renders `Scroll` as a plain div** with no `overflow` — now more
  visible than it was, since Scroll can finally carry style.
