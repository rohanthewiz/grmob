# Session: The WASM Checkbox and TextArea Gap

**Session ID:** session_01LZZTYuh1UXxGjKfkomQRM6
**Date:** 2026-08-31, ~23:47
**Branch:** master
**Follows:** `2026-0831-2333-wasm-js-test-harness.md`

The previous session built the JS test harness and, in the same breath,
recorded what it found but deliberately did not fix:

> **The WASM runtime does not render `Checkbox` state or `TextArea` rows.**
> `checked` and `rows` reach no DOM property, and a Checkbox has no
> `type="checkbox"` attribute, so it draws as a text box. This is the largest
> known correctness gap in the WASM renderer and now has a harness ready to
> test the fix.

This session closed it. The harness was ready, and the whole change took one
pass with no debugging — which is the return on the previous session.

---

# Part 1 — The gap was three things, not two

The backlog named two missing props. The `type` attribute is the third, and
it is the one that matters most, because it is not a wrong *state* — it is an
entirely wrong *control*.

`tagForType` collapses four Go node types onto `<input>`:

```
Input          ─┐
InputPassword  ─┤
NumericInput   ─┼──> <input>
Checkbox       ─┘
```

An `<input>` with no `type` attribute is a text box. So a Checkbox was drawn
as a text field, and `checked` had nowhere to appear even if the runtime had
tried to set it. Fixing `checked` without `type` would have been fixing the
state of a control that does not exist.

`inputTypeFor` is the table, kept in sync with `htmlout/export.go`'s. It is
set once in `createElement` and never on the update path, for a reason worth
recording: a node type cannot change under a patch. `reconcile/patch.go:68`
emits a `replace` when `old.Type != new.Type`, so the element carrying a given
type is always one `createElement` built. That was checked in the reconciler
rather than assumed.

## Why `checked` and `rows` are properties, not attributes

This is the substantive design call, and it is not stylistic.

A `checked` **attribute** is only the control's *default* state. The browser
stops consulting it the moment the user touches the box — after that, the
attribute and what the user sees are two different facts. The **property** is
the live state, and the live state is what Go is describing. Writing the
attribute would produce a checkbox that renders correctly until first use and
then silently stops tracking Go's state, which is a worse bug than the one
being fixed because it survives a casual look.

A mutation pins exactly this: *"checked read as an attribute, not the live
property"* — caught.

`rows` is a property for the ordinary reason `value` and `placeholder` are:
live control state the runtime keeps in step with Go, not markup written once.

## One helper, two call sites

`applyRows` exists rather than two inline guards because of the lesson the
previous session paid for: the payload-vocabulary bug was two statements of
one rule that drifted apart silently. `checked` is short enough to inline at
both sites; `rows` carries a guard, and a guard duplicated is a guard that
will eventually differ.

## The one deliberate divergence from htmlout

`rows` is "limited to only positive numbers" in the DOM — assigning 0 is an
error, not a request for a zero-line box. So `applyRows` applies only a
positive integer and otherwise leaves the browser's own default.

`htmlout` differs: it defaults an absent `rows` to 3, because it must emit
*some* attribute value where the runtime can simply say nothing. The
divergence is recorded in the comment and is unreachable through
`core.TextArea`, which always supplies a positive count — it covers only a
hand-built `core.Node`.

The echo guard was also deliberately *not* copied from `value`:

```js
} else if (k === "checked") {
    // No echo guard, unlike value above: assigning a boolean back onto a
    // checkbox costs nothing, where re-assigning a text field's value would
    // move the caret to the end mid-typing.
    el.checked = !!v;
}
```

The guard on `value` is not a general-purpose optimization; it exists for one
specific cost. Copying it without that cost would have implied a reason that
is not there.

---

# Part 2 — Turning a documented gap into coverage

The previous session left this comment in `replay_test.mjs`, and it was
written to be deleted:

> Deliberately absent: `checked` and `rows`. [...] It is left uncovered
> instead of asserted, because a test that pinned the current behavior would
> make the gap harder to close.

That judgment paid off exactly as intended — there was no wrong assertion to
unwind, only an absence to fill. The prop table gained `inputType`, `checked`
and `rows`, and `INPUT_TYPE` is restated from the contract rather than read
off the runtime, for the same reason the rest of the table is: a conformance
test that consulted `inputTypeFor` would only prove the runtime agrees with
itself.

`dom.mjs` gained `checked` and `rows`, both `undefined` by default rather than
at a browser default (`false`, `2`). That matches how `value` is already
treated and it is what makes the replay work: an element the runtime never
wrote to has to stay distinguishable from one it wrote a default into.

## The transcript could not tell the difference

The replay passed immediately — and that was worth being suspicious of. The
check: does the replay catch a runtime that renders a checkbox but hardcodes
`false`?

It did not, at first. Every checkbox tick in both scenarios was undone before
the final tree: `demo` returns to the Counter tab, so the Form tab's checkbox
is not in the final tree at all, and `signup`'s successful submit resets the
form. The transcript's only checkbox was an unticked one, and an unticked
checkbox cannot distinguish a renderer that reads the state from one that
hardcodes `false`.

Fixed by ending the signup scenario with the terms box ticked. Re-verified:

| mutation | replay alone |
|---|---|
| update path hardcodes unticked | **caught** |
| mount path hardcodes unticked | survived (no scenario mounts a ticked box) |
| control: unmutated | passes |

The surviving one is covered by the unit test *"a checkbox renders the state
Go gave it"*, and the split is recorded rather than papered over — the replay
covers the patch path, the unit test covers the mount path.

A false positive was caught on the way: the first run of that check reported
both mutants caught, because `GRMOB_TRANSCRIPT` pointed at `/private/tmp`
while `run.sh` writes under `$TMPDIR`. The tests were failing on a missing
file, not on the assertion. A control mutant (an unmutated runtime that must
*pass*) was added to the table so a bad harness path cannot read as a good
verdict again.

---

# Testing the tests

**10 mutations of the new runtime code, 10 caught**, each restored from a
scratchpad snapshot rather than `git checkout`:

no type attribute at all; Checkbox typed as a text field; NumericInput typed
as text; checked dropped at mount; checked dropped on the update path; checked
forced true on the update path; checked read as an attribute rather than the
live property; rows dropped at mount; rows dropped on the update path; rows
accepting a non-positive count.

Five new unit tests: the variant types (including that a `<textarea>` and a
`<span>` get *no* type attribute, matching htmlout); the mount state in both
directions; the patch state in both directions — the unticking direction
specifically, since Go owns the state and the tick the user keeps is the one
that came back through a patch; the rows patch; and the non-positive guard.

## Files touched

`wasm/grmob-runtime.js` (+68: `inputTypeFor`, `applyRows`, `checked`/`rows` on
both paths, the type attribute), `wasm/verify/dom.mjs` (+20: two properties
and the member census), `wasm/verify/replay_test.mjs` (+37: the prop table),
`wasm/verify/runtime_test.mjs` (+92: five tests),
`wasm/verify/gen.go` (+7: the ticked box), `docs/platforms/wasm.md` (+27: a
form-controls table and the property-vs-attribute reasoning).

Gate: `gofmt` clean, `go vet` clean, full Go suite, `GOOS=js GOARCH=wasm`
build, `ios/verify` (flex solver + 9 patch batches + Swift typecheck), and
`wasm/verify/run.sh` (34 tests, up from 29) — all green.
(`examples/todoapp/store.go` remains unformatted — pre-existing, untouched.)

## Not verified here

**Still no browser**, unchanged and unchangeable by this harness. Whether a
`type="checkbox"` input actually draws as a box, and whether `rows` produces
the height it asks for, are rendering questions. What is proven is that the
runtime writes what `htmlout` writes, and `htmlout`'s output is what a browser
has always rendered correctly.

**`TextArea` is in no example app**, so `rows` is covered by unit tests only —
the replay's prop table states the rule but no transcript exercises it. It
will be covered for free the moment any scenario grows a TextArea.

**Android is still unbuilt** and **iOS still type-checks without running**.

## Backlog

In Progress still holds only Packaging (`grmob build --target=…`).

Closed this session: the WASM renderer's Checkbox and TextArea gap — the
largest known correctness gap in that renderer — plus the `<input type>`
discriminator the backlog entry had not named.

Still open from earlier sessions:

- Theme contrast, and `Variant`'s third consumer.
- **Hooks have no unmount signal.**
- **`docs/concepts/architecture.md` does not mention frames.**
- **`core.Image` bases its style on `Theme.Components.Camera`.**
- The WASM runtime's style mapping is still much thinner than `htmlout`'s.
- **A bottom-docked bar has no way to ask for the keyboard on its own.**
- **`htmlout` renders `Scroll` as a plain div** with no `overflow`.
- **A second imperative API would justify the bridge command channel.**
- **`core.SendSystemEvent` is a dead stub** — `core/toast.go` is its only
  caller, so `ShowToast` currently reaches nothing.
- **A `Cached` subtree silently swallows focus commands** and order
  membership.
- **An app-drawn keyboard toolbar has no worked example.**
- **`imeAction` is a third prop that must not vanish**, guarded by a third
  sticky sentinel; a single helper could state the rule once.
- **`gen.go`'s transcripts still emit no `add`, `remove` or `add-child`.**
  `replace` is covered; the other three are unit-tested but not replayed. A
  growing list — `examples/todoapp` — would supply them.
- **The demo scenario's tab switches produce only `update-props`**, which is
  either a nice property of the reconciler or a sign the scenario is not
  switching what it thinks it is. Still not understood.
- **Nothing runs either verify harness automatically.** Both are shell scripts
  a human remembers to run; `go test ./...` does not reach them.

Noticed this session, not acted on:

- **No example app uses `core.TextArea`**, which is why the `rows` fix has no
  replay coverage and why the gap survived as long as it did. The same holds
  for `core.Image` and `CameraView` — a node type absent from every example is
  a node type no harness can reach.
- **`htmlout` and the WASM runtime now carry two copies of the node-type →
  input-type table** (`export.go` and `inputTypeFor`), plus a third
  restatement in the replay test. The third is deliberate — a conformance test
  must restate. The first two are a real duplication, and the same shape as
  the `objectFit` mapping already duplicated between them.
