# Package forms

```go
import "github.com/rohanthewiz/grmob/forms"
```

Package forms is grmob's validation layer: a vocabulary of rules, a hook that owns a form's values and decides when its errors become visible, and bound input builders that tie a field's value and its onChange to the same name in one call.

components.FormField has always had an Error slot and nothing ever filled it — the widget renders feedback, but deciding \*what\* the feedback is, and \*when\* the user should see it, is not a widget's job. This package is that decision, kept out of core (validation touches no node type and no renderer) and out of components (a struct widget cannot own state that outlives one field).

	form := forms.UseForm(ctx, forms.Spec{
	    Fields: []forms.Field{
	        {Name: "email", Rules: []forms.Rule{
	            forms.Required("We need an address to reach you"),
	            forms.Email(""),
	        }},
	        {Name: "password", Rules: []forms.Rule{
	            forms.Required(""),
	            forms.MinLen(8, "Use at least 8 characters"),
	        }},
	        {Name: "confirm"},
	        {Name: "terms", Rules: []forms.Rule{forms.Accepted("Please accept the terms")}},
	    },
	    // Cross-field checks see the whole value set at once.
	    Validate: func(v forms.Values) map[string]string {
	        if v["confirm"] != v["password"] {
	            return map[string]string{"confirm": "The two passwords differ"}
	        }
	        return nil
	    },
	})

	components.Screen{Children: []core.View{
	    components.FormField{
	        Label: "Email",
	        Hint:  "We never share it",
	        Error: form.Error("email"),
	        Input: form.Input("email", "you@example.com"),
	    },
	    components.FormField{
	        Label: "Password",
	        Error: form.Error("password"),
	        Input: form.Password("password", "••••••••"),
	    },
	    components.Button{
	        Label: "Create account",
	        OnTap: form.OnSubmit(func(v forms.Values) { createAccount(v) }),
	    },
	}}

## Errors are derived, never stored

The record behind UseForm holds the values, which fields have been edited, whether a submit has been attempted, any errors handed in from outside (see SetErrors), and which names have had their Initial applied. It does \*not\* hold the errors the rules produce. Those are recomputed from (values, spec) every time they are read.

That is the whole reason this package has no staleness bugs. A stored error map has to be invalidated on every write, on every rule change, and on every cross-field dependency — miss one and a field shows an error it has already fixed. A derived error map cannot be stale by construction. The cost is that a form with n fields recomputes its rules n times per render pass (Error is called once per field); rules are string checks and forms are a handful of fields, so this is nanoseconds, and it buys away an entire class of bug.

## The spec is re-read every render

Only the record survives between passes. The Spec — the fields, their rules, the cross-field Validate, the reveal policy — is whatever this render handed to UseForm, so a rule may close over live state (a currency list fetched at runtime, a maximum that depends on another hook) and take effect on the next pass with no re-registration. Compare hooks.UseMemo, where the deps are what get stored; here nothing about the \*checking\* is stored at all.

## Whitespace

Every rule in this package validates the \*trimmed\* value, and a value that is nothing but whitespace is empty. One policy, applied in one place (see optional in rules.go), so that two rules on the same field can never disagree about the same text — Required + MinLen(3) used to accept "ab ", and Pattern + Range used to split on " 12345".

It also matches what apps actually persist: a submit handler reads Values.Trimmed, so a rule that measured untrimmed text was measuring something the app was never going to store.

Values themselves are kept raw — the form holds exactly what the user typed, and Values.Trimmed is the accessor that applies the same policy on the way out. A check that genuinely cares about surrounding whitespace is written as an inline Rule, which receives the raw value.

## Rules of hooks apply

UseForm consumes exactly one slot on the context it is given (see core.NewState), so it must be called unconditionally, in a stable position, on every pass — the same discipline every other hook in grmob follows. Debug mode reports a violation as cursor drift.

## Index

- [`type Field`](#type-field)
- [`type Form`](#type-form)
    - [`func UseForm`](#func-useform)
    - [`func (*Form) Blurred`](#func-form-blurred)
    - [`func (*Form) Checkbox`](#func-form-checkbox)
    - [`func (*Form) Checked`](#func-form-checked)
    - [`func (*Form) Error`](#func-form-error)
    - [`func (*Form) Errors`](#func-form-errors)
    - [`func (*Form) Input`](#func-form-input)
    - [`func (*Form) InputWithSubmit`](#func-form-inputwithsubmit)
    - [`func (*Form) MarkBlurred`](#func-form-markblurred)
    - [`func (*Form) OnBlur`](#func-form-onblur)
    - [`func (*Form) OnChange`](#func-form-onchange)
    - [`func (*Form) OnSubmit`](#func-form-onsubmit)
    - [`func (*Form) OnToggle`](#func-form-ontoggle)
    - [`func (*Form) Password`](#func-form-password)
    - [`func (*Form) Required`](#func-form-required)
    - [`func (*Form) Reset`](#func-form-reset)
    - [`func (*Form) Select`](#func-form-select)
    - [`func (*Form) SetErrors`](#func-form-seterrors)
    - [`func (*Form) SetValue`](#func-form-setvalue)
    - [`func (*Form) Submit`](#func-form-submit)
    - [`func (*Form) Submitted`](#func-form-submitted)
    - [`func (*Form) TextArea`](#func-form-textarea)
    - [`func (*Form) Touched`](#func-form-touched)
    - [`func (*Form) Valid`](#func-form-valid)
    - [`func (*Form) Value`](#func-form-value)
    - [`func (*Form) Values`](#func-form-values)
- [`type Reveal`](#type-reveal)
- [`type Rule`](#type-rule)
    - [`func Accepted`](#func-accepted)
    - [`func Email`](#func-email)
    - [`func Integer`](#func-integer)
    - [`func MaxLen`](#func-maxlen)
    - [`func MinLen`](#func-minlen)
    - [`func OneOf`](#func-oneof)
    - [`func Pattern`](#func-pattern)
    - [`func Range`](#func-range)
    - [`func Required`](#func-required)
- [`type Spec`](#type-spec)
- [`type Values`](#type-values)
    - [`func (Values) Bool`](#func-values-bool)
    - [`func (Values) Clone`](#func-values-clone)
    - [`func (Values) Float`](#func-values-float)
    - [`func (Values) Int`](#func-values-int)
    - [`func (Values) Trimmed`](#func-values-trimmed)

## Types

### type Field

```go
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
```

Field declares one input's name, its starting text, and the rules its value must satisfy.

<small>[forms/form.go:59](https://github.com/rohanthewiz/grmob/blob/master/forms/form.go#L59)</small>

### type Form

```go
type Form struct {
	// contains filtered or unexported fields
}
```

Form is the handle returned by UseForm: the live record plus this render pass's Spec.

A fresh Form is built on every pass and is cheap (two pointers and the spec's header). The record is the shared part; the spec is not, which is what lets a rule close over state that changes between passes.

<small>[forms/form.go:173](https://github.com/rohanthewiz/grmob/blob/master/forms/form.go#L173)</small>

#### func UseForm

```go
func UseForm(ctx *core.Context, spec Spec) *Form
```

UseForm allocates (or re-binds) a form on the context's hook slot at the current cursor and returns a handle to it.

It consumes exactly one slot, so the rules of hooks apply: call it unconditionally, in a stable position, on every pass.

<small>[forms/form.go:184](https://github.com/rohanthewiz/grmob/blob/master/forms/form.go#L184)</small>

#### func (*Form) Blurred

```go
func (f *Form) Blurred(name string) bool
```

Blurred reports whether focus has entered and left the field since mount or Reset.

It reports what has been \*observed\*, which is not the same as what has happened: nothing polls the platform, so a field whose control never attaches Form.OnBlur reads as unblurred forever. Under any policy but RevealOnBlur the bound builders do not attach it, so this stays false throughout — that is the honest answer ("no blur was reported"), not a bug.

<small>[forms/form.go:290](https://github.com/rohanthewiz/grmob/blob/master/forms/form.go#L290)</small>

#### func (*Form) Checkbox

```go
func (f *Form) Checkbox(name string, props ...core.PropsAndChildren) core.View
```

Checkbox is a boolean control bound to name, storing "true"/"false" as the field's text (see Values.Bool). A box that starts ticked is declared with Field{Name: "terms", Initial: "true"}.

A checkbox has no label of its own; the usual pairing is a ListRow, which centers the box against its title:

	components.ListRow{Leading: form.Checkbox("terms"), Title: "I accept the terms"}

No blur binding, unlike the text builders: a tick is a commit, not a draft, so there is no moment where the user is "still working on" a checkbox and nothing for leaving it to signal. Neither native platform gives a checkbox keyboard focus anyway. Under RevealOnBlur a required-but-unticked box therefore says nothing until the submit reveals it, which is the same treatment a field the user never visited gets.

<small>[forms/inputs.go:121](https://github.com/rohanthewiz/grmob/blob/master/forms/inputs.go#L121)</small>

#### func (*Form) Checked

```go
func (f *Form) Checked(name string) bool
```

Checked reads a field as a checkbox. See Values.Bool for what counts.

<small>[forms/form.go:229](https://github.com/rohanthewiz/grmob/blob/master/forms/form.go#L229)</small>

#### func (*Form) Error

```go
func (f *Form) Error(name string) string
```

Error is the message to show for one field, or "" when there is nothing to show — which is exactly what components.FormField.Error wants:

	components.FormField{
	    Label: "Email",
	    Hint:  "We never share it",
	    Error: form.Error("email"),
	    Input: form.Input("email", "you@example.com"),
	}

An empty result can mean either "valid" or "not revealed yet"; ask Valid or Submitted if the difference matters.

<small>[forms/form.go:533](https://github.com/rohanthewiz/grmob/blob/master/forms/form.go#L533)</small>

#### func (*Form) Errors

```go
func (f *Form) Errors() map[string]string
```

Errors is what the user should currently see: the derived errors for the fields the reveal policy has unlocked, plus every external error.

External errors ignore the policy on purpose. A message that came back from a submit is by definition post-submit, and one installed before any submit is an app deliberately putting a field in an error state — in both cases hiding it would be hiding the only thing the form knows.

They are also laid over the derived ones rather than filling in around them, because they are the newer information: a server verdict comes from a check the rules cannot make, and it is dropped the moment the field changes (see SetValue), so the two can only meet on a value the server has already seen and rejected.

<small>[forms/form.go:505](https://github.com/rohanthewiz/grmob/blob/master/forms/form.go#L505)</small>

#### func (*Form) Input

```go
func (f *Form) Input(name, placeholder string, props ...core.PropsAndChildren) core.View
```

Input is a single-line text field bound to name.

<small>[forms/inputs.go:80](https://github.com/rohanthewiz/grmob/blob/master/forms/inputs.go#L80)</small>

#### func (*Form) InputWithSubmit

```go
func (f *Form) InputWithSubmit(name, placeholder string, handler func(Values), props ...core.PropsAndChildren) core.View
```

InputWithSubmit is Input plus the keyboard's return/done key wired to a submit of the whole form — the one-field form (a search box, a promo code) where the return key is the only commit affordance there is.

<small>[forms/inputs.go:87](https://github.com/rohanthewiz/grmob/blob/master/forms/inputs.go#L87)</small>

#### func (*Form) MarkBlurred

```go
func (f *Form) MarkBlurred(name string)
```

MarkBlurred records that focus has left the field.

Unlike SetValue this deliberately does \*not\* mark the field touched or drop its external error: leaving a field changes nothing about its value, so a server's verdict on that value still stands, and a field the user tabbed through without typing in has not been edited. Conflating the two would let a tab-through silently satisfy RevealOnTouch.

Safe to call from any goroutine.

<small>[forms/form.go:305](https://github.com/rohanthewiz/grmob/blob/master/forms/form.go#L305)</small>

#### func (*Form) OnBlur

```go
func (f *Form) OnBlur(name string) func()
```

OnBlur returns the field's blur handler, ready to hand to core.OnBlur:

	core.Input(form.Value("email"), "you@example.com", form.OnChange("email"),
	    core.OnBlur(form.OnBlur("email")))

The bound builders attach this themselves under RevealOnBlur; this is for a control they do not cover.

<small>[forms/form.go:332](https://github.com/rohanthewiz/grmob/blob/master/forms/form.go#L332)</small>

#### func (*Form) OnChange

```go
func (f *Form) OnChange(name string) func(string)
```

OnChange returns the field's change handler, ready to hand to any builder that takes one.

	core.Input(form.Value("email"), "you@example.com", form.OnChange("email"))

The bound builders (Form.Input and friends) are this pairing pre-made; use this one for a control they do not cover.

<small>[forms/form.go:265](https://github.com/rohanthewiz/grmob/blob/master/forms/form.go#L265)</small>

#### func (*Form) OnSubmit

```go
func (f *Form) OnSubmit(handler func(Values)) func()
```

OnSubmit adapts Submit to the void-callback shape every commit affordance takes — a Button's OnTap, an InputRow's OnSubmit, the keyboard's return key:

	components.Button{Label: "Create account", OnTap: form.OnSubmit(createAccount)}

<small>[forms/form.go:599](https://github.com/rohanthewiz/grmob/blob/master/forms/form.go#L599)</small>

#### func (*Form) OnToggle

```go
func (f *Form) OnToggle(name string) func(bool)
```

OnToggle returns a checkbox's handler, storing the bool as text through the same spelling Values.Bool reads.

<small>[forms/form.go:271](https://github.com/rohanthewiz/grmob/blob/master/forms/form.go#L271)</small>

#### func (*Form) Password

```go
func (f *Form) Password(name, placeholder string, props ...core.PropsAndChildren) core.View
```

Password is a masked single-line field bound to name.

<small>[forms/inputs.go:94](https://github.com/rohanthewiz/grmob/blob/master/forms/inputs.go#L94)</small>

#### func (*Form) Required

```go
func (f *Form) Required(name string) bool
```

Required reports whether the field rejects an empty value — which is what components.FormField.Required wants, so the marker beside a label and the rule that justifies it cannot disagree:

	components.FormField{
	    Label:    "Email",
	    Required: form.Required("email"),
	    Error:    form.Error("email"),
	    Input:    form.Input("email", "you@example.com"),
	}

It is \*derived\*, not declared: the field's rules are run against "" and the answer is whether any of them complains. There is deliberately no Field.Required flag to read instead — a flag would be a second claim about the same field, true only while someone keeps it in step with the rules, and the failure it permits (a starred field that submits empty, an unmarked one that will not) is exactly the disagreement the marker exists to avoid. It is the same reasoning that keeps the error map derived; see the package doc.

Three consequences worth knowing:

  - Any rule that speaks about an empty value counts, not just Required. Accepted does — an unticked box is "false", never "" — so a terms-of-service checkbox reads as required, which is what it is. So does an app's own closure that rejects "".
  - A rule closing over live state makes the answer live too: a field whose Required is added only in some app state loses its marker on the pass the rule goes away, with no bookkeeping.
  - Spec.Validate is not consulted. A cross-field requirement ("confirm is needed once password is set") is not a property of the field, and the probe has no other field's value to give it. Mark those by hand.

Unknown names are not required, which is also the answer for a field with no rules at all.

<small>[forms/form.go:371](https://github.com/rohanthewiz/grmob/blob/master/forms/form.go#L371)</small>

#### func (*Form) Reset

```go
func (f *Form) Reset()
```

Reset returns the form to its starting state: every declared field back to its Initial, nothing touched, nothing submitted, no external errors.

Initials are re-read from \*this pass's\* Spec, not from the ones the form mounted with, which is also how a form is populated from data that arrives late: render the loaded values as Initial and call Reset once they land.

Fields not named by the current spec are dropped entirely rather than left behind, so a reset form's Values is exactly its declaration.

<small>[forms/form.go:647](https://github.com/rohanthewiz/grmob/blob/master/forms/form.go#L647)</small>

#### func (*Form) Select

```go
func (f *Form) Select(name string, options []core.SelectOption, props ...core.PropsAndChildren) core.View
```

Select is a picker bound to name, storing the chosen option's Value as the field's text — so every rule that reads a string reads it unchanged, and forms.Required rejects an unchosen picker exactly as it rejects an empty field.

A picker with no natural default is declared with a leading empty option and a Required rule; one that has a default is declared with Field{Name: "plan", Initial: "free"}. The two are genuinely different forms and the widget takes no position on which is meant.

No blur binding, for the reason Checkbox has none: a choice is a commit rather than a draft, so there is no moment where the user is "still working on" a picker and nothing for leaving it to signal. Under RevealOnBlur an unchosen picker therefore says nothing until the submit reveals it, which is the same treatment an untouched field gets.

<small>[forms/inputs.go:159](https://github.com/rohanthewiz/grmob/blob/master/forms/inputs.go#L159)</small>

#### func (*Form) SetErrors

```go
func (f *Form) SetErrors(errs map[string]string)
```

SetErrors installs the errors this form could not have computed itself — the ones that come back from a server, where uniqueness, authorization and business rules live.

	form.SetErrors(map[string]string{"email": "That address is already registered"})

They are always shown regardless of the reveal policy, they outrank a derived error on the same field, and each one is dropped as soon as that field changes (see SetValue).

The set is replaced wholesale rather than merged, so a nil or empty map clears them — which is what a retry should do before it starts.

Blank messages are dropped here and nowhere else: this is the only writer of the external map, so filtering at the boundary is what lets every reader treat a present key as a real error. Storing one would make the form refuse to submit for a reason it then declines to display.

Safe to call from any goroutine; this is the one form method whose usual caller is a network response.

<small>[forms/form.go:623](https://github.com/rohanthewiz/grmob/blob/master/forms/form.go#L623)</small>

#### func (*Form) SetValue

```go
func (f *Form) SetValue(name, value string)
```

SetValue writes a field, marks it touched, and drops any external error standing against it.

All three follow from the one fact that the value changed, whoever changed it: the field is no longer in its initial state (touched), and a server's verdict on the old text ("that address is already registered") is no longer about the text on screen. Clearing on change rather than on the next submit is what makes the error feel answered — the message disappears as the user starts fixing it, not after another round trip.

Safe to call from any goroutine.

<small>[forms/form.go:244](https://github.com/rohanthewiz/grmob/blob/master/forms/form.go#L244)</small>

#### func (*Form) Submit

```go
func (f *Form) Submit(handler func(Values)) bool
```

Submit checks the form and, if it is clean, calls handler with a private copy of the values. It reports whether the form was valid.

The attempt is recorded either way, which is what makes RevealOnSubmit work: a failed submit is the moment the form starts explaining itself.

	if !form.Submit(save) {
	    // errors are now visible; nothing else to do
	}

handler runs on the calling goroutine — the native event thread, for a button tap — so a submit that talks to a network should hand off:

	form.Submit(func(v forms.Values) {
	    go func() {
	        if errs := api.CreateAccount(v); errs != nil {
	            form.SetErrors(errs)
	        }
	    }()
	})

The values are already a copy, so the goroutine is free to outlive the pass.

<small>[forms/form.go:573](https://github.com/rohanthewiz/grmob/blob/master/forms/form.go#L573)</small>

#### func (*Form) Submitted

```go
func (f *Form) Submitted() bool
```

Submitted reports whether Submit has been attempted since mount or Reset — successfully or not. It is what RevealOnSubmit keys on, and it is also the honest test for "has the user asked for this form to be checked yet".

<small>[forms/form.go:395](https://github.com/rohanthewiz/grmob/blob/master/forms/form.go#L395)</small>

#### func (*Form) TextArea

```go
func (f *Form) TextArea(name string, rows int, props ...core.PropsAndChildren) core.View
```

TextArea is a multi-line field bound to name.

core.TextArea takes no placeholder, so a text area's guidance goes in the wrapping FormField's Label or Hint rather than inside the box.

<small>[forms/inputs.go:102](https://github.com/rohanthewiz/grmob/blob/master/forms/inputs.go#L102)</small>

#### func (*Form) Touched

```go
func (f *Form) Touched(name string) bool
```

Touched reports whether the field has been changed since mount or Reset.

<small>[forms/form.go:276](https://github.com/rohanthewiz/grmob/blob/master/forms/form.go#L276)</small>

#### func (*Form) Valid

```go
func (f *Form) Valid() bool
```

Valid reports whether the form would submit as it stands, ignoring the reveal policy entirely.

Note what \*not\* to do with it. Under the default RevealOnSubmit, disabling the submit button on !Valid() produces a dead end: nothing is revealed until a submit, and no submit can happen while the button is disabled, so the user is left with a form that refuses to work and says nothing about why. Let the submit run and fail — that is the event that turns the explanations on. core.Disabled belongs on a submit that is \*in flight\*, not on one that is invalid.

<small>[forms/form.go:547](https://github.com/rohanthewiz/grmob/blob/master/forms/form.go#L547)</small>

#### func (*Form) Value

```go
func (f *Form) Value(name string) string
```

Value reads a field's current text.

<small>[forms/form.go:215](https://github.com/rohanthewiz/grmob/blob/master/forms/form.go#L215)</small>

#### func (*Form) Values

```go
func (f *Form) Values() Values
```

Values returns an independent copy of every field's text.

<small>[forms/form.go:222](https://github.com/rohanthewiz/grmob/blob/master/forms/form.go#L222)</small>

### type Reveal

```go
type Reveal int
```

Reveal decides when a field's error becomes visible. It is a display policy, not a validation policy: the rules run continuously either way, and Valid always answers for the form as it stands.

The default exists because validating as the user types is hostile — the second character of an address is not yet a valid address, and saying so is scolding someone for not having finished. The rule of thumb is "reward early, punish late": say nothing until the user claims to be done, then stay live so every correction is confirmed the instant it lands.

<small>[forms/form.go:19](https://github.com/rohanthewiz/grmob/blob/master/forms/form.go#L19)</small>

```go
const (
	// RevealOnSubmit shows nothing until the first Submit, then shows every
	// field's error live. The zero value, and the right default for almost
	// every form.
	RevealOnSubmit Reveal = iota

	// RevealOnBlur shows a field's error once focus has left that field (and
	// every field's after a Submit). It is the closest thing to the "reward
	// early, punish late" rule of thumb that still says something before the
	// submit: leaving a field is the user's own claim to have finished with
	// it, so a complaint then is an answer rather than an interruption, and
	// the correction is still confirmed live because the rules never stop
	// running.
	//
	// It needs the input to report the edge. The bound builders (Form.Input
	// and friends) attach core.OnBlur themselves under this policy; a control
	// built by hand out of Value and OnChange must attach Form.OnBlur(name)
	// or it will reveal nothing until the submit.
	RevealOnBlur

	// RevealOnTouch shows a field's error once that field has been edited
	// (and every field's after a Submit). Suited to a field whose format is
	// unguessable and worth correcting mid-flight — a card number, a
	// one-time code — where waiting for submit wastes the user's typing.
	//
	// Note this fires on the *second keystroke* of an address, not when the
	// user is done with it; RevealOnBlur is usually the kinder reading of
	// "do not wait for the submit".
	RevealOnTouch

	// RevealAlways shows every error from the first render, before the user
	// has touched anything. Mostly useful in tests and in a form that is
	// pre-populated from elsewhere and is being shown *because* it is wrong.
	RevealAlways
)
```

### type Rule

```go
type Rule func(value string) string
```

Rule reports what is wrong with a field's value, or "" when nothing is.

A rule is a plain function, not an interface: every rule below is a closure over its own parameters, and an app's own checks ("this username is reserved", "this date is in the past") are written inline with no type to implement.

	{Name: "handle", Rules: []forms.Rule{
	    forms.Required(""),
	    func(v string) string {
	        if reserved[strings.ToLower(v)] {
	            return "That handle is taken by the system"
	        }
	        return ""
	    },
	}}

A rule must be pure. It is evaluated on every read of the form's errors — several times per render pass — so it must not have side effects, must not mutate what it is given, and must not perform I/O. (It runs outside the form's lock, so it \*may\* safely read the form it belongs to, but a rule that needs another field's value belongs in Spec.Validate instead.)

<small>[forms/rules.go:33](https://github.com/rohanthewiz/grmob/blob/master/forms/rules.go#L33)</small>

#### func Accepted

```go
func Accepted(msg string) Rule
```

Accepted requires a checkbox to be ticked — the terms-of-service field.

Not wrapped in optional: an unticked box is "false", not "", so there is no empty case to skip, and treating it as one would make the rule silent exactly when it matters.

<small>[forms/rules.go:212](https://github.com/rohanthewiz/grmob/blob/master/forms/rules.go#L212)</small>

#### func Email

```go
func Email(msg string) Rule
```

Email checks that a value has the shape of an address. See emailShape for what this rule does and does not claim.

<small>[forms/rules.go:129](https://github.com/rohanthewiz/grmob/blob/master/forms/rules.go#L129)</small>

#### func Integer

```go
func Integer(msg string) Rule
```

Integer requires a whole number.

Pair it with core.Input, not core.NumericInput: NumericInput's change callback parses with strconv.Atoi and \*drops the event\* when the text does not parse, so the unparseable value never reaches the form and this rule can never fire. A validated numeric field is a text field with a rule.

<small>[forms/rules.go:166](https://github.com/rohanthewiz/grmob/blob/master/forms/rules.go#L166)</small>

#### func MaxLen

```go
func MaxLen(n int, msg string) Rule
```

MaxLen allows at most n characters (runes; see MinLen).

A max is a display-time complaint, not a keystroke filter: the field still accepts the extra characters and the user sees why they are too many. A hard cap belongs on the native control, which grmob does not expose — and silently dropping keystrokes is the worse behavior anyway.

<small>[forms/rules.go:110](https://github.com/rohanthewiz/grmob/blob/master/forms/rules.go#L110)</small>

#### func MinLen

```go
func MinLen(n int, msg string) Rule
```

MinLen requires at least n characters — runes, not bytes, so "héllo" is five and a name in a non-Latin script is not silently held to a longer standard than an ASCII one.

<small>[forms/rules.go:95](https://github.com/rohanthewiz/grmob/blob/master/forms/rules.go#L95)</small>

#### func OneOf

```go
func OneOf(msg string, allowed ...string) Rule
```

OneOf restricts a value to a fixed set — a picker or segmented control whose selection is carried as text.

The message comes first here, against the convention of every other rule in this file, because Go requires the variadic parameter to be last. The alternative was a \[]string parameter, which would read forms.OneOf(\[]string{"card", "bank"}, "") at every call site.

<small>[forms/rules.go:196](https://github.com/rohanthewiz/grmob/blob/master/forms/rules.go#L196)</small>

#### func Pattern

```go
func Pattern(re *regexp.Regexp, msg string) Rule
```

Pattern requires the value to match re.

It takes a compiled \*regexp.Regexp rather than an expression string on purpose. A Spec is rebuilt on every render pass, so a rule that compiled its own pattern would run regexp.Compile on every pass of every form — and, with MustCompile, would turn a typo in a pattern into a panic on a render goroutine rather than at startup. A package-level var compiles once:

	var postcode = regexp.MustCompile(`^[0-9]{5}$`)
	...
	forms.Pattern(postcode, "Five digits")

<small>[forms/rules.go:151](https://github.com/rohanthewiz/grmob/blob/master/forms/rules.go#L151)</small>

#### func Range

```go
func Range(lo, hi int, msg string) Rule
```

Range requires a whole number between lo and hi inclusive. A value that is not a number at all fails this rule too, so Range alone is enough for a bounded numeric field; adding Integer before it only changes which of the two messages an unparseable value gets.

<small>[forms/rules.go:179](https://github.com/rohanthewiz/grmob/blob/master/forms/rules.go#L179)</small>

#### func Required

```go
func Required(msg string) Rule
```

Required rejects a value that is empty or nothing but whitespace. It is the only rule that speaks about emptiness; see optional.

<small>[forms/rules.go:83](https://github.com/rohanthewiz/grmob/blob/master/forms/rules.go#L83)</small>

### type Spec

```go
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
```

Spec is the whole declaration of a form: its fields, an optional cross-field check, and when errors become visible.

<small>[forms/form.go:80](https://github.com/rohanthewiz/grmob/blob/master/forms/form.go#L80)</small>

### type Values

```go
type Values map[string]string
```

Values is a form's field set: name to raw text.

Strings all the way down, deliberately. Every event the framework carries from a native control is a string on the wire — core.NumericInput already stores an int and ships it through registerTextCallback, parsing on the way back in — so a form that kept typed values would be converting twice and would need a heterogeneous map to hold them. Keeping the raw text is also what makes validation possible at all: "12x" has to survive long enough for Integer to complain about it, and a map\[string]int has nowhere to put it.

Values is a plain map type, so ordinary indexing is the way to read a field: v\["email"]. The methods below are for the values that are not text.

<small>[forms/values.go:20](https://github.com/rohanthewiz/grmob/blob/master/forms/values.go#L20)</small>

#### func (Values) Bool

```go
func (v Values) Bool(name string) bool
```

Bool reads a checkbox field. Anything strconv.ParseBool does not accept as true — including an absent field and any free text — reads as false.

No ok result, unlike Int: a checkbox has two states and "not checked" is a complete answer for every value that is not "true". The form writes these values itself (OnToggle formats with strconv.FormatBool), so the lossy case cannot arise from user input.

<small>[forms/values.go:37](https://github.com/rohanthewiz/grmob/blob/master/forms/values.go#L37)</small>

#### func (Values) Clone

```go
func (v Values) Clone() Values
```

Clone returns an independent copy. The form hands a clone to every rule evaluation and to every submit handler, so neither can reach back into the live map — a handler that spawns a goroutine (the usual shape for a network submit) would otherwise be reading a map the user is still typing into.

<small>[forms/values.go:69](https://github.com/rohanthewiz/grmob/blob/master/forms/values.go#L69)</small>

#### func (Values) Float

```go
func (v Values) Float(name string) (float64, bool)
```

Float reads a decimal field, reporting whether it parsed. See Int.

<small>[forms/values.go:60](https://github.com/rohanthewiz/grmob/blob/master/forms/values.go#L60)</small>

#### func (Values) Int

```go
func (v Values) Int(name string) (int, bool)
```

Int reads a numeric field, reporting whether it parsed. The ok result is real here — the field is free text a user typed, and "" and "twelve" are both reachable — so a caller either pairs the field with an Integer rule and can ignore ok inside a submit handler, or checks it.

<small>[forms/values.go:54](https://github.com/rohanthewiz/grmob/blob/master/forms/values.go#L54)</small>

#### func (Values) Trimmed

```go
func (v Values) Trimmed(name string) string
```

Trimmed reads a field with leading and trailing space removed. Required trims before deciding a field is empty, so a value that passed validation may still be padded; a submit handler that stores what it was given generally wants this rather than the raw text.

<small>[forms/values.go:26](https://github.com/rohanthewiz/grmob/blob/master/forms/values.go#L26)</small>

