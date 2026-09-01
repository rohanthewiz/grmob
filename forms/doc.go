// Package forms is grmob's validation layer: a vocabulary of rules, a hook
// that owns a form's values and decides when its errors become visible, and
// bound input builders that tie a field's value and its onChange to the same
// name in one call.
//
// components.FormField has always had an Error slot and nothing ever filled
// it — the widget renders feedback, but deciding *what* the feedback is, and
// *when* the user should see it, is not a widget's job. This package is that
// decision, kept out of core (validation touches no node type and no
// renderer) and out of components (a struct widget cannot own state that
// outlives one field).
//
//	form := forms.UseForm(ctx, forms.Spec{
//	    Fields: []forms.Field{
//	        {Name: "email", Rules: []forms.Rule{
//	            forms.Required("We need an address to reach you"),
//	            forms.Email(""),
//	        }},
//	        {Name: "password", Rules: []forms.Rule{
//	            forms.Required(""),
//	            forms.MinLen(8, "Use at least 8 characters"),
//	        }},
//	        {Name: "confirm"},
//	        {Name: "terms", Rules: []forms.Rule{forms.Accepted("Please accept the terms")}},
//	    },
//	    // Cross-field checks see the whole value set at once.
//	    Validate: func(v forms.Values) map[string]string {
//	        if v["confirm"] != v["password"] {
//	            return map[string]string{"confirm": "The two passwords differ"}
//	        }
//	        return nil
//	    },
//	})
//
//	components.Screen{Children: []core.View{
//	    components.FormField{
//	        Label: "Email",
//	        Hint:  "We never share it",
//	        Error: form.Error("email"),
//	        Input: form.Input("email", "you@example.com"),
//	    },
//	    components.FormField{
//	        Label: "Password",
//	        Error: form.Error("password"),
//	        Input: form.Password("password", "••••••••"),
//	    },
//	    components.Button{
//	        Label: "Create account",
//	        OnTap: form.OnSubmit(func(v forms.Values) { createAccount(v) }),
//	    },
//	}}
//
// # Errors are derived, never stored
//
// The record behind UseForm holds the values, which fields have been edited,
// whether a submit has been attempted, any errors handed in from outside (see
// SetErrors), and which names have had their Initial applied. It does *not*
// hold the errors the rules produce. Those are recomputed from (values, spec)
// every time they are read.
//
// That is the whole reason this package has no staleness bugs. A stored error
// map has to be invalidated on every write, on every rule change, and on
// every cross-field dependency — miss one and a field shows an error it has
// already fixed. A derived error map cannot be stale by construction. The
// cost is that a form with n fields recomputes its rules n times per render
// pass (Error is called once per field); rules are string checks and forms
// are a handful of fields, so this is nanoseconds, and it buys away an entire
// class of bug.
//
// # The spec is re-read every render
//
// Only the record survives between passes. The Spec — the fields, their
// rules, the cross-field Validate, the reveal policy — is whatever this
// render handed to UseForm, so a rule may close over live state (a currency
// list fetched at runtime, a maximum that depends on another hook) and take
// effect on the next pass with no re-registration. Compare hooks.UseMemo,
// where the deps are what get stored; here nothing about the *checking* is
// stored at all.
//
// # Whitespace
//
// Every rule in this package validates the *trimmed* value, and a value that
// is nothing but whitespace is empty. One policy, applied in one place (see
// optional in rules.go), so that two rules on the same field can never
// disagree about the same text — Required + MinLen(3) used to accept "ab ",
// and Pattern + Range used to split on " 12345".
//
// It also matches what apps actually persist: a submit handler reads
// Values.Trimmed, so a rule that measured untrimmed text was measuring
// something the app was never going to store.
//
// Values themselves are kept raw — the form holds exactly what the user
// typed, and Values.Trimmed is the accessor that applies the same policy on
// the way out. A check that genuinely cares about surrounding whitespace is
// written as an inline Rule, which receives the raw value.
//
// # Rules of hooks apply
//
// UseForm consumes exactly one slot on the context it is given (see
// core.NewState), so it must be called unconditionally, in a stable position,
// on every pass — the same discipline every other hook in grmob follows.
// Debug mode reports a violation as cursor drift.
package forms
