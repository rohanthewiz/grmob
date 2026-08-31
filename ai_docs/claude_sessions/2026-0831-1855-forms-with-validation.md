# Session: Forms with Validation

**Session ID:** session_01LZZTYuh1UXxGjKfkomQRM6
**Date:** 2026-08-31, ~18:55
**Branch:** master
**Follows:** `2026-0831-1830-renderer-gaps.md`

## Goal

The ROADMAP's In Progress list opened with "Forms with validation — `FormField`
renders an `Error`, but nothing validates". The widget has had an `Error` slot
since it was written and nothing has ever filled it.

Deciding *what* the feedback says — and *when* the user should see it — is not
a widget's job, so this is a new package rather than a field on `FormField`.

## Where it lives, and why not somewhere else

`forms/` — not `core`, because validation touches no node type and no
renderer; not `components`, because a struct widget cannot own state that
outlives one field. It imports `core` and nothing else in the repo, and
nothing in the repo imports it. `components.FormField` never learned about it:
the form produces the string, the widget frames it, and the coupling exists
only at the call site.

```go
components.FormField{
    Label: "Email",
    Hint:  "We never share it",
    Error: form.Error("email"),
    Input: form.Input("email", "you@example.com"),
}
```

## The decision everything else follows from: errors are derived, never stored

The record behind `UseForm` holds the values, which fields are touched,
whether a submit was attempted, the external (server) errors, and the
bookkeeping of which names have had their `Initial` applied. It does **not**
hold the errors the rules produce. Those are recomputed from `(values, spec)`
on every read.

A stored error map has to be invalidated on every write, every rule change and
every cross-field dependency; miss one and a field shows an error it has
already fixed. A derived map cannot be stale by construction. The cost is
O(fields × rules) per render, since `Error` is called once per field — string
checks on a handful of fields, so nanoseconds.

Three consequences, none of them the original task:

**The spec is re-read every pass.** Only the record survives between renders,
so a rule may close over live state (a bound that depends on another hook, a
list fetched at runtime) and take effect on the next pass with no
re-registration. Compare `hooks.UseMemo`, where the deps are the thing stored;
here nothing about the *checking* is stored at all.

**Rules run outside the lock.** `derived()` snapshots the values under the
record's mutex and releases it before evaluating anything. A `Rule` or a
`Validate` is caller code, and caller code running under a lock the caller can
reach — through the very `Form` the rule is attached to — is a deadlock
waiting for its first re-entrant read. There is a test that a rule reading its
own form does not hang.

**`problems()` is a union, not a precedence.** A mutation proved the overwrite
there was unobservable: its only callers (`Valid`, `Submit`) ask nothing of it
but `len() == 0`, and the key set is the same either way. Precedence now lives
only in `Errors`, where it is visible, and the comment no longer claims a
behavior no test can see.

## When errors appear is a policy, not a side effect

Validating as the user types is hostile — the second character of an address
is not yet a valid address. `Spec.Reveal`:

| | a field's error is visible when |
|---|---|
| `RevealOnSubmit` (zero value) | the first `Submit` has been attempted |
| `RevealOnTouch` | that field has been edited — **or** a submit was attempted |
| `RevealAlways` | always |

The policies are **cumulative, not exclusive**. A submit reveals everything
under all three, `RevealOnTouch` included: a policy where "on touch" kept
hiding an untouched field after a submit produces a form that refuses to
submit and shows no reason why.

The same reasoning produced a warning on `Valid()`: **do not disable the
submit button on `!form.Valid()`**. Under the default policy nothing explains
itself until a submit and no submit can happen while the button is disabled —
a form that refuses to work and refuses to say why. `core.Disabled` (shipped
last session) belongs on a submit that is *in flight*. The example says this
in a comment at the exact call site where the mistake would be made.

## Smaller decisions inside it

**Every rule but `Required` and `Accepted` is silent about an empty value.**
Emptiness is `Required`'s subject and nobody else's — otherwise an *optional*
field carrying `MinLen(8)` complains before the user types anything, and a
*required* one carrying both has two opinions about the same empty string
where `FormField` can show one. Implemented as an `optional()` lifter so the
rule reads as what it is.

**The first failing rule wins.** One line of feedback, so ordering the rules
is choosing which complaint is useful. `Required` first.

**`Spec.Validate` fills in only where a field rule was silent.** An empty
confirmation needs "Required", not "the two passwords differ" — true,
unhelpful, and what a last-writer-wins merge would show.

**Messages are arguments, with a default.** `forms.MinLen(8, "")` falls back to
the rule's own English. Copy and localization are the app's; a prototype
should not have to invent nine strings first.

**`Pattern` takes a compiled `*regexp.Regexp`.** A `Spec` is rebuilt every
render pass, so a rule compiling its own expression would run
`regexp.Compile` per pass per form — and `MustCompile` would turn a typo into
a panic on the render goroutine rather than at startup.

**`Values` is `map[string]string`.** Every native event is a string on the
wire (`core.NumericInput` already ships an int through the *text* channel and
parses on the way back), and keeping the raw text is what makes validation
possible: `"12x"` has to survive long enough for `Integer` to complain about
it. Hence also the note that a validated numeric field is `core.Input` + a
rule, because `NumericInput`'s callback **drops** unparseable text and the
rule could never fire.

**Server errors are a third kind.** `SetErrors` installs what a client cannot
compute. They ignore the reveal policy (a message that came back from a submit
is post-submit by definition), outrank a rule's message on the same field
(newer information), and each is dropped the moment that field changes — the
verdict was about the old text, so it disappears as the user starts fixing it
rather than after another round trip. Replaced wholesale, so `SetErrors(nil)`
is what a retry does first.

**Blank messages are filtered at one boundary.** `SetErrors` is the only
writer of the external map, so it drops empties and every reader trusts the
invariant. A stored blank would be an invisible reason the form will not
submit.

**`Initial` seeds once, keyed by a `seeded` set.** Re-seeding every pass would
type a deleted default back in; not seeding at all would leave a field added
on a later pass (a conditional section, a repeated row) without its default.
`Reset` re-reads initials from *this pass's* spec, which is also how a form is
populated from data that arrives late.

**Bound builders exist to make one bug unwritable.** The unbound spelling
names a field three times in three roles and nothing checks they agree; copy a
field, change two of three, and you get a text box that will not accept typing
with no error anywhere. `form.Input` / `InputWithSubmit` / `Password` /
`TextArea` / `Checkbox` write the name once and forward style props unchanged.
`Value` and `OnChange` stay exported for the controls with no binding.

## The example

`examples/signup` — a sign-up screen exercising rules, the cross-field
confirmation, a checkbox, a server rejection and the reset. Two things it
demonstrates that are not obvious from the API:

- Wrapping a `ListRow` in a `FormField` gives a **checkbox an error line**,
  which it has no slot for. The `Input` slot taking any view is what makes
  that free.
- Both hooks are hoisted above the branch between the form and the
  confirmation screen. Moving `UseForm` below it — verified — raises two
  cursor-drift concerns and fails the test, so the audit assertion has teeth.

## Testing

**Go — new:** `forms/rules_test.go` (every rule accepted/rejected, the
empty-skip, the message defaults and their quoted bounds),
`forms/form_test.go` (seeding, reveal policies, rule order, cross-field
precedence, external errors, submit, reset, spec freshness, a rule reading its
own form, an 8-field concurrent hammer under `-race`, slot isolation across
two forms and two apps), `forms/inputs_test.go` (each bound builder driven
through a real callback dispatch with a decoy field present, plus the
end-to-end `FormField` integration), and `examples/signup/app_test.go` (the
whole flow through `render.Manager`, debug mode on).

### Mutation-checked

21 deliberate breaks. Three survived the first pass and each one taught
something:

| survivor | what it exposed |
|---|---|
| last-failing-rule wins | **a weak test** — an empty field cannot distinguish first from last, since only `Required` speaks about `""`. Case became two rules that both fail on `"abc"`. |
| external does not outrank derived (in `problems`) | **dead precision** — `problems`'s callers only ask whether it is empty. Precedence moved to `Errors`; comments corrected. |
| `SetErrors` stores blank messages | **a redundant guard** — three places filtered blanks. Filtered once at the writer, invariant stated on the field, readers simplified. |

All three then caught. Everything else was caught first time, including: the
`optional` skip, `Validate` overriding field rules, `RevealOnTouch` ignoring
submit, `Errors` ignoring the policy, `Submit`/`SetValue` skipping
`RequestRender`, `Submit` handing over the live map, `Reset` leaving
`submitted` on, `Initial` re-seeding, `derived` holding the lock across rules
(deadlock → timeout), `Accepted` wrapped in `optional`, and a bound builder
crossing its wires. Backups diffed clean.

## Files touched

- `forms/doc.go`, `values.go`, `rules.go`, `form.go`, `inputs.go` — **new**
- `forms/rules_test.go`, `form_test.go`, `inputs_test.go` — **new**
- `examples/signup/app.go`, `app_test.go` — **new**
- `components/form_field.go` — doc comment only: where `Error` comes from
- `docs/concepts/forms.md` — **new**; `mkdocs.yml` nav
- `docs/components.md` (FormField), `docs/concepts/state-and-hooks.md`
  (`UseForm` in the hook catalogue), `docs/index.md`, `README.md`
- `ROADMAP.md` — moved to Done under Extensions

Gate: `gofmt` clean, `go vet` clean, full suite passes, `go test -race` on
core/render/hooks/htmlout/components/forms/examples clean, `GOOS=js
GOARCH=wasm build` clean. (`examples/todoapp/store.go` remains unformatted —
pre-existing, untouched.)

## Not verified here

**No native run.** The package produces ordinary nodes through the existing
`core` builders, so there is nothing new on the wire for the Android or iOS
renderers to learn — but `examples/signup` has only ever been rendered to
JSON, never to pixels. The two natives from last session are still owed a
simulator/emulator pass of their own.

## Backlog

In Progress now holds:

- Keyboard-aware scroll area for mobile.
- Packaging (`grmob build --target=…`).

Still open from earlier sessions:

- Theme contrast, and `Variant`'s third consumer.
- **Hooks have no unmount signal.** A navigation frame is the first lifetime
  the framework can dispose of; component-level unmount remains unsolved.
- **`docs/concepts/architecture.md` does not mention frames.**
- **`core.Image` bases its style on `Theme.Components.Camera`** — every Image
  inherits a black background and a block display. Wants a decision of its own.
- The WASM runtime's style mapping is still much thinner than `htmlout`'s.

Noticed this session, not acted on:

- **No focus or blur event anywhere in the framework.** `RevealOnTouch` is
  keyed on "has been edited" because "has been left" is not observable. Real
  blur-time validation — the conventional web behavior — needs a fifth event
  channel on the bridge.
- **`FormField` has no required marker.** Purely cosmetic (an asterisk beside
  the label), but it is the one piece of form furniture the widget still
  lacks, and now that `Required` exists as a rule the two would want to agree.
