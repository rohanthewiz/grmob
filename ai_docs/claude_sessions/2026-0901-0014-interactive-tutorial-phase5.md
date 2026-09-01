# Interactive Tutorial — Phase 5: Chapter 5 (Forms & Validation)

Session: https://claude.ai/code/session_01Nm3azgkKS6SWsFoqJZEdUh

## Goal

Phase 5 of the eight-phase interactive tutorial: add Chapter 5 "Forms &
Validation" to `examples/tutorial`. As in Phases 2–4, the framework needed
exactly one change — appending `chapter5()` to `Chapters` in lesson.go —
and IDs 5.x, home rows, progress, and the Next/Prev walk (now 25 lessons)
picked the chapter up automatically. Committed as eb0f528.

## What was built

- **chapter5.go** — five lessons, each with a live demo. Through-line: what
  the user sees is *derived* — errors recomputed from (values, this pass's
  spec) on every read, the required marker from the rules run against "",
  visibility from the reveal policy over the stored
  touched/blurred/submitted facts. Sources of truth: docs/concepts/forms.md,
  the forms/ package, examples/signup.
  - **5.1 A form in four calls** — Spec→UseForm, FormField frame, bound
    builder (name written once), OnSubmit. Demo: two-field RSVP under the
    default RevealOnSubmit — the failed submit turns explanations on,
    fixes confirm live, handler stores `v.Trimmed("name")` so padded input
    confirms clean ("RSVP received for June Gopher").
  - **5.2 Rules & the required marker** — RevealAlways playground (the
    caption justifies the policy: it's the tests-and-playgrounds one).
    Three checkboxes compose the rule list per pass — the
    spec-is-re-read-every-pass claim, demonstrated — plus an always-on
    closure rejecting reserved handles (root/admin). Caption prints
    `form.Required("handle") → true/false` live; unchecking Required takes
    FormField's asterisk with it. `lowercaseOnly` hoisted to a package var
    (the Pattern-takes-a-compiled-regexp lesson, in place).
  - **5.3 When errors appear** — the four Reveal policies on a segmented
    control re-policing one record; `revealNames`/`revealValues` parallel
    tables (captions double as constant suffixes, the ch4 trick). Flags
    caption instruments `touched · blurred · submitted`. "Check the form"
    submits with a nil handler (the attempt still records); "Start over"
    resets between experiments. Doctrine carried: policies are cumulative;
    never disable submit on !Valid; the blur binding exists only under
    RevealOnBlur (asserted: no onBlur prop under OnSubmit).
  - **5.4 Cross-field & server errors** — password/confirm via
    Spec.Validate (field rules win the merge: empty confirm says
    "Required", not the mismatch), simulated registry (`takenAddresses`)
    answering through SetErrors + core.Focus on a hoisted UseFocusRef —
    the signup pattern taught. External errors: reveal-blind, outrank
    rules, drop on the field's first edit.
  - **5.5 Values, initials & reset** — values are text: "12x" survives so
    Range can complain while a caption shows `Values().Int → (0, false)`;
    NumericInput-drops-the-event warning; gift checkbox `Initial: "true"`
    through v.Bool (labeled by a ListRow, so no FormField label/marker —
    signup's terms-row reasoning); clearing the field proves Initial is
    not re-applied; Reset re-seeds and clears submitted.
- **chapter5_test.go** — five liveness tests, new primitives:
  `fieldByPlaceholder`/`typeField`/`blurField` (chapter 2's first-Input
  helper can't address multi-field screens; blurField Fatals if onBlur is
  missing — under RevealOnBlur that's a real failure) and `countMarkers`
  (the asterisk is its own Text node). Sentinel discipline: live messages
  differ from the static code blocks' canonical ones ("These don't match
  the password above" live vs "The two passwords differ" in the snippet;
  "Someone got there first…" vs "That address is already registered").
  5.4 asserts the focus stamp: `focusAction: "focus"` on email, `""` on
  the password field (stamped, nothing to do — not "blur").
- **lesson.go** — `chapter5(),` appended to `Chapters`.

## Facts learned/confirmed this phase

- The wasm runtime wires both `onBlur` (→ blur event) and the focus
  commands (`focusEpoch`/`focusAction` via applyFocusCommand), so the
  RevealOnBlur and core.Focus demos work in the browser, not just under
  test. htmlout also exports onBlur (`data-onblur`).
- Switching Spec.Reveal from live state mid-session changes whether each
  field registers onBlur, shifting later pass-sequential callback IDs for
  one pass (blurProp's documented settle-next-pass shift) — harmless
  because tests re-read the tree per dispatch and the runtime re-registers
  per pass. Noted in a code comment on 5.3's Spec.
- `form.OnSubmit(nil)` is legal and useful: Submit records the attempt
  (the reveal trigger) and only guards the handler call.
- `Form.Blurred` stays false forever under policies that attach no blur
  binding — "no blur was observed" is the honest answer; 5.3 teaches it
  via the flags caption.
- forms.Range alone rejects unparseable text too, so Required + Range is
  a complete bounded-numeric spec (Integer would only change the message).

## Verification

- `go test ./...` fully green; gofmt and vet clean; TestMain debug mode
  audits every pass (all 25 lessons walked by the Next test).
- `go test -race -count=2 ./examples/tutorial/` clean.
- `GOOS=js GOARCH=wasm go build -o <scratch>/main.wasm ./wasm` compiles.

## Next session: Phase 6

Per the phase list (Phase 1 doc, 2026-0831-2047): Ch.6 "Navigation &
Overlays" — Push/Pop/Replace/Reset, Modal, Toast. Sources of truth:
docs/concepts/navigation.md and the core Navigator; the tutorial shell
itself is a worked Navigator example (lesson_screen.go's Replace walk,
home Push/Pop) worth teaching from. Framework change should again be
exactly one line: `chapter6(),` in lesson.go.
