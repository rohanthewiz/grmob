package forms

import (
	"strconv"
	"sync"

	"github.com/rohanthewiz/grmob/core"
)

// Reveal decides when a field's error becomes visible. It is a display
// policy, not a validation policy: the rules run continuously either way, and
// Valid always answers for the form as it stands.
//
// The default exists because validating as the user types is hostile — the
// second character of an address is not yet a valid address, and saying so is
// scolding someone for not having finished. The rule of thumb is "reward
// early, punish late": say nothing until the user claims to be done, then
// stay live so every correction is confirmed the instant it lands.
type Reveal int

const (
	// RevealOnSubmit shows nothing until the first Submit, then shows every
	// field's error live. The zero value, and the right default for almost
	// every form.
	RevealOnSubmit Reveal = iota

	// RevealOnTouch shows a field's error once that field has been edited
	// (and every field's after a Submit). Suited to a field whose format is
	// unguessable and worth correcting mid-flight — a card number, a
	// one-time code — where waiting for submit wastes the user's typing.
	RevealOnTouch

	// RevealAlways shows every error from the first render, before the user
	// has touched anything. Mostly useful in tests and in a form that is
	// pre-populated from elsewhere and is being shown *because* it is wrong.
	RevealAlways
)

// Field declares one input's name, its starting text, and the rules its value
// must satisfy.
type Field struct {
	// Name is the field's key in Values and the handle every method on Form
	// takes. It is not shown to the user — the label lives on the
	// components.FormField that wraps the input.
	Name string

	// Initial seeds the value the first time this name is seen, and again
	// after Reset. It is not re-applied on later renders, so a field the user
	// has cleared stays cleared even though the spec still names a default.
	Initial string

	// Rules run in order and the first one to complain wins: a field shows a
	// single line of feedback (FormField has one slot for it), so ordering
	// them is choosing which complaint is the most useful one. Required
	// belongs first — it is the only rule that speaks about an empty value,
	// and every other rule stays silent about one.
	Rules []Rule
}

// Spec is the whole declaration of a form: its fields, an optional
// cross-field check, and when errors become visible.
type Spec struct {
	Fields []Field

	// Validate is the cross-field pass — the checks no single field's rules
	// can make because they need to see another field's value. It runs after
	// every field's rules, and its messages fill in only for fields that do
	// not already have one:
	//
	//	Validate: func(v forms.Values) map[string]string {
	//	    if v["confirm"] != v["password"] {
	//	        return map[string]string{"confirm": "The two passwords differ"}
	//	    }
	//	    return nil
	//	}
	//
	// Field rules win because they are the more specific complaint. If
	// "confirm" is empty, "Required" is what the user needs to read, not
	// "the two passwords differ" — which is true, unhelpful, and would be
	// what a last-writer-wins merge showed.
	//
	// Keys need not name a declared field. A key no input renders is a
	// form-level error, which a banner above the fields can read with
	// Error("form") or whatever name the app picks.
	//
	// Like a Rule, Validate must be pure and must not call back into the
	// Form. It is handed a private clone of the values and may do as it likes
	// with it.
	Validate func(Values) map[string]string

	// Reveal is the display policy; the zero value is RevealOnSubmit.
	Reveal Reveal
}

// formRecord is the part of a form that survives between render passes. It
// lives behind the hook slot by pointer, guarded by its own mutex, for the
// same reason hooks.UseReducer's record does: every mutation here is a
// read-modify-write of a map, and core.State offers only whole-value atomic
// Get and Set. Two keystrokes arriving from different goroutines would
// otherwise both read the old map and one would be lost.
//
// What is *not* here is the error map. Errors are recomputed from the values
// and the current Spec on every read (see Form.derived) — the package doc
// explains why that is the design and not an oversight.
type formRecord struct {
	mu sync.Mutex

	// values is every declared field's raw text, keyed by name.
	values Values

	// seeded records which names have had their Initial applied. Without it,
	// re-seeding on every render would resurrect a default the user had
	// deleted; with it, a field added to the spec later — a conditional
	// section, a repeated row — still gets its Initial on the pass it first
	// appears.
	seeded map[string]bool

	// touched marks the fields the user has changed. Read only by
	// RevealOnTouch.
	touched map[string]bool

	// external holds errors this form could not have computed — see
	// SetErrors. Always revealed, and cleared per field when that field
	// changes.
	//
	// Invariant: no entry is ever the empty string. SetErrors is the only
	// writer and it drops blanks there, so every reader below can treat a
	// present key as a real error rather than re-testing each message. A
	// blank stored here would otherwise be an invisible reason the form
	// refuses to submit.
	external map[string]string

	// submitted flips on the first Submit and stays on until Reset. It is
	// what RevealOnSubmit waits for.
	submitted bool
}

// Form is the handle returned by UseForm: the live record plus this render
// pass's Spec.
//
// A fresh Form is built on every pass and is cheap (two pointers and the
// spec's header). The record is the shared part; the spec is not, which is
// what lets a rule close over state that changes between passes.
type Form struct {
	ctx  *core.Context
	rec  *formRecord
	spec Spec
}

// UseForm allocates (or re-binds) a form on the context's hook slot at the
// current cursor and returns a handle to it.
//
// It consumes exactly one slot, so the rules of hooks apply: call it
// unconditionally, in a stable position, on every pass.
func UseForm(ctx *core.Context, spec Spec) *Form {
	// Through core.NewState rather than a bare slot write, for the reasons
	// hooks.UseMemo spells out: NewState is what keeps the slot array and the
	// cursor in step, and slot position is what makes the hook per-call-site.
	// The record allocated here is kept only on the first pass.
	slot := core.NewState(ctx, &formRecord{
		values:   Values{},
		seeded:   map[string]bool{},
		touched:  map[string]bool{},
		external: map[string]string{},
	})
	rec := slot.Get()

	rec.mu.Lock()
	for _, fld := range spec.Fields {
		if !rec.seeded[fld.Name] {
			rec.seeded[fld.Name] = true
			// Assigned even when Initial is "", so every declared field is
			// present in Values. A submit handler ranging over the map then
			// sees the form's full shape rather than only the fields the user
			// happened to fill in.
			rec.values[fld.Name] = fld.Initial
		}
	}
	rec.mu.Unlock()

	return &Form{ctx: ctx, rec: rec, spec: spec}
}

// Value reads a field's current text.
func (f *Form) Value(name string) string {
	f.rec.mu.Lock()
	defer f.rec.mu.Unlock()
	return f.rec.values[name]
}

// Values returns an independent copy of every field's text.
func (f *Form) Values() Values {
	f.rec.mu.Lock()
	defer f.rec.mu.Unlock()
	return f.rec.values.Clone()
}

// Checked reads a field as a checkbox. See Values.Bool for what counts.
func (f *Form) Checked(name string) bool {
	return isTrue(f.Value(name))
}

// SetValue writes a field, marks it touched, and drops any external error
// standing against it.
//
// All three follow from the one fact that the value changed, whoever changed
// it: the field is no longer in its initial state (touched), and a server's
// verdict on the old text ("that address is already registered") is no longer
// about the text on screen. Clearing on change rather than on the next submit
// is what makes the error feel answered — the message disappears as the user
// starts fixing it, not after another round trip.
//
// Safe to call from any goroutine.
func (f *Form) SetValue(name, value string) {
	f.rec.mu.Lock()
	f.rec.values[name] = value
	f.rec.touched[name] = true
	delete(f.rec.external, name)
	f.rec.mu.Unlock()

	// Explicitly, for the reason hooks.UseReducer's dispatch does it: the
	// state hangs off a pointer, so the hook slot's *value* never changes and
	// core.State.Set — the usual carrier of the render request — is never
	// called. Without this the field would not repaint as the user types.
	f.ctx.RequestRender()
}

// OnChange returns the field's change handler, ready to hand to any builder
// that takes one.
//
//	core.Input(form.Value("email"), "you@example.com", form.OnChange("email"))
//
// The bound builders (Form.Input and friends) are this pairing pre-made; use
// this one for a control they do not cover.
func (f *Form) OnChange(name string) func(string) {
	return func(v string) { f.SetValue(name, v) }
}

// OnToggle returns a checkbox's handler, storing the bool as text through the
// same spelling Values.Bool reads.
func (f *Form) OnToggle(name string) func(bool) {
	return func(b bool) { f.SetValue(name, strconv.FormatBool(b)) }
}

// Touched reports whether the field has been changed since mount or Reset.
func (f *Form) Touched(name string) bool {
	f.rec.mu.Lock()
	defer f.rec.mu.Unlock()
	return f.rec.touched[name]
}

// Submitted reports whether Submit has been attempted since mount or Reset —
// successfully or not. It is what RevealOnSubmit keys on, and it is also the
// honest test for "has the user asked for this form to be checked yet".
func (f *Form) Submitted() bool {
	f.rec.mu.Lock()
	defer f.rec.mu.Unlock()
	return f.rec.submitted
}

// revealed applies the Spec's policy to one field name.
//
// The policies are cumulative rather than exclusive: a submit reveals
// everything under all three, and RevealOnTouch also honours it, so a field
// the user skipped entirely still reports its error once they try to submit.
// A policy where "on touch" hid an untouched field's error after a submit
// would let a form refuse to submit while showing no reason.
func (f *Form) revealed(name string) bool {
	if f.spec.Reveal == RevealAlways {
		return true
	}
	f.rec.mu.Lock()
	defer f.rec.mu.Unlock()
	if f.rec.submitted {
		return true
	}
	return f.spec.Reveal == RevealOnTouch && f.rec.touched[name]
}

// derived computes the rule and cross-field errors from the current values
// and the current Spec. Reveal-blind: it answers what is wrong, not what the
// user should be shown.
//
// The values are snapshotted under the lock and every rule then runs with the
// lock released. That is deliberate — a Rule or a Validate is caller code, and
// caller code running under a lock the caller can reach (through the very
// Form the rule is attached to) is a deadlock waiting for its first re-entrant
// read. It also means a slow rule does not block a keystroke arriving on
// another goroutine.
func (f *Form) derived() map[string]string {
	vals := f.Values()

	out := make(map[string]string)
	for _, fld := range f.spec.Fields {
		for _, r := range fld.Rules {
			if r == nil {
				continue
			}
			if msg := r(vals[fld.Name]); msg != "" {
				// First complaint wins; see Field.Rules.
				out[fld.Name] = msg
				break
			}
		}
	}

	if f.spec.Validate != nil {
		// vals is this call's private clone and is not read again, so it goes
		// straight in rather than being cloned a second time.
		for name, msg := range f.spec.Validate(vals) {
			if msg == "" {
				continue
			}
			if _, taken := out[name]; !taken {
				out[name] = msg
			}
		}
	}
	return out
}

// problems is every reason this form is not submittable: the derived errors
// unioned with the external ones. Reveal-blind, because a hidden error is
// still a reason — Valid and Submit are its only callers and both ask nothing
// of it but whether it is empty.
//
// Which message a collision keeps is therefore not decided here; it is
// decided in Errors, which is what anyone reads.
func (f *Form) problems() map[string]string {
	out := f.derived()

	f.rec.mu.Lock()
	defer f.rec.mu.Unlock()
	for name, msg := range f.rec.external {
		out[name] = msg
	}
	return out
}

// Errors is what the user should currently see: the derived errors for the
// fields the reveal policy has unlocked, plus every external error.
//
// External errors ignore the policy on purpose. A message that came back from
// a submit is by definition post-submit, and one installed before any submit
// is an app deliberately putting a field in an error state — in both cases
// hiding it would be hiding the only thing the form knows.
//
// They are also laid over the derived ones rather than filling in around
// them, because they are the newer information: a server verdict comes from a
// check the rules cannot make, and it is dropped the moment the field changes
// (see SetValue), so the two can only meet on a value the server has already
// seen and rejected.
func (f *Form) Errors() map[string]string {
	out := make(map[string]string)
	for name, msg := range f.derived() {
		if f.revealed(name) {
			out[name] = msg
		}
	}

	f.rec.mu.Lock()
	defer f.rec.mu.Unlock()
	for name, msg := range f.rec.external {
		out[name] = msg
	}
	return out
}

// Error is the message to show for one field, or "" when there is nothing to
// show — which is exactly what components.FormField.Error wants:
//
//	components.FormField{
//	    Label: "Email",
//	    Hint:  "We never share it",
//	    Error: form.Error("email"),
//	    Input: form.Input("email", "you@example.com"),
//	}
//
// An empty result can mean either "valid" or "not revealed yet"; ask Valid or
// Submitted if the difference matters.
func (f *Form) Error(name string) string {
	return f.Errors()[name]
}

// Valid reports whether the form would submit as it stands, ignoring the
// reveal policy entirely.
//
// Note what *not* to do with it. Under the default RevealOnSubmit, disabling
// the submit button on !Valid() produces a dead end: nothing is revealed
// until a submit, and no submit can happen while the button is disabled, so
// the user is left with a form that refuses to work and says nothing about
// why. Let the submit run and fail — that is the event that turns the
// explanations on. core.Disabled belongs on a submit that is *in flight*, not
// on one that is invalid.
func (f *Form) Valid() bool {
	return len(f.problems()) == 0
}

// Submit checks the form and, if it is clean, calls handler with a private
// copy of the values. It reports whether the form was valid.
//
// The attempt is recorded either way, which is what makes RevealOnSubmit
// work: a failed submit is the moment the form starts explaining itself.
//
//	if !form.Submit(save) {
//	    // errors are now visible; nothing else to do
//	}
//
// handler runs on the calling goroutine — the native event thread, for a
// button tap — so a submit that talks to a network should hand off:
//
//	form.Submit(func(v forms.Values) {
//	    go func() {
//	        if errs := api.CreateAccount(v); errs != nil {
//	            form.SetErrors(errs)
//	        }
//	    }()
//	})
//
// The values are already a copy, so the goroutine is free to outlive the pass.
func (f *Form) Submit(handler func(Values)) bool {
	f.rec.mu.Lock()
	f.rec.submitted = true
	f.rec.mu.Unlock()

	probs := f.problems()

	// Before the handler runs, and unconditionally: flipping submitted
	// changes what every field displays even when no value moved, and nothing
	// else on this path writes a slot, so this is the only thing that gets
	// the newly revealed errors onto the screen.
	f.ctx.RequestRender()

	if len(probs) > 0 {
		return false
	}
	if handler != nil {
		handler(f.Values())
	}
	return true
}

// OnSubmit adapts Submit to the void-callback shape every commit affordance
// takes — a Button's OnTap, an InputRow's OnSubmit, the keyboard's return key:
//
//	components.Button{Label: "Create account", OnTap: form.OnSubmit(createAccount)}
func (f *Form) OnSubmit(handler func(Values)) func() {
	return func() { f.Submit(handler) }
}

// SetErrors installs the errors this form could not have computed itself —
// the ones that come back from a server, where uniqueness, authorization and
// business rules live.
//
//	form.SetErrors(map[string]string{"email": "That address is already registered"})
//
// They are always shown regardless of the reveal policy, they outrank a
// derived error on the same field, and each one is dropped as soon as that
// field changes (see SetValue).
//
// The set is replaced wholesale rather than merged, so a nil or empty map
// clears them — which is what a retry should do before it starts.
//
// Blank messages are dropped here and nowhere else: this is the only writer
// of the external map, so filtering at the boundary is what lets every reader
// treat a present key as a real error. Storing one would make the form refuse
// to submit for a reason it then declines to display.
//
// Safe to call from any goroutine; this is the one form method whose usual
// caller is a network response.
func (f *Form) SetErrors(errs map[string]string) {
	next := make(map[string]string, len(errs))
	for name, msg := range errs {
		if msg != "" {
			next[name] = msg
		}
	}

	f.rec.mu.Lock()
	f.rec.external = next
	f.rec.mu.Unlock()

	f.ctx.RequestRender()
}

// Reset returns the form to its starting state: every declared field back to
// its Initial, nothing touched, nothing submitted, no external errors.
//
// Initials are re-read from *this pass's* Spec, not from the ones the form
// mounted with, which is also how a form is populated from data that arrives
// late: render the loaded values as Initial and call Reset once they land.
//
// Fields not named by the current spec are dropped entirely rather than left
// behind, so a reset form's Values is exactly its declaration.
func (f *Form) Reset() {
	f.rec.mu.Lock()
	f.rec.values = make(Values, len(f.spec.Fields))
	f.rec.seeded = make(map[string]bool, len(f.spec.Fields))
	f.rec.touched = map[string]bool{}
	f.rec.external = map[string]string{}
	f.rec.submitted = false
	for _, fld := range f.spec.Fields {
		f.rec.values[fld.Name] = fld.Initial
		f.rec.seeded[fld.Name] = true
	}
	f.rec.mu.Unlock()

	f.ctx.RequestRender()
}
