// Package signup is the worked example for package forms: a sign-up screen
// that exercises every part of the validation story in one screen.
//
//	rules            Required / Email / MinLen / Accepted, one message each
//	cross-field      the confirmation must match the password
//	reveal policy    nothing complains until the first submit, then live
//	server errors    a uniqueness check only the back end can make
//	the widget       components.FormField, whose Error slot has been waiting
//	                 for something to fill it since it was written
//
// Every field is declared once, in the Spec, and rendered through a bound
// builder — so the name that reads the value is the same name that writes it,
// by construction rather than by care.
package signup

import (
	"strings"

	"github.com/rohanthewiz/grmob/components"
	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/forms"
)

// registered stands in for the only check a client cannot make: whether this
// address already has an account. A real app asks a server, which is why the
// answer arrives through Form.SetErrors rather than through a Rule — a rule is
// pure and synchronous, and this is neither.
var registered = map[string]bool{
	"taken@example.com": true,
}

// App is the screen. Both hooks run before anything branches on their values,
// which is the rule of hooks doing its usual job: UseForm consumes a slot on
// every pass or every later hook reads its neighbour's.
func App(ctx *core.Context) core.View {
	// The address of the account just created, or "" while the form is up.
	created := core.NewState(ctx, "")

	form := forms.UseForm(ctx, forms.Spec{
		// Reveal is left at its zero value, RevealOnSubmit: the form says
		// nothing until the user claims to be done, then explains itself live
		// as each field is fixed. Validating as they type would mean telling
		// someone their address is invalid two characters in.
		Fields: []forms.Field{
			{Name: "email", Rules: []forms.Rule{
				// Required first, always: it is the only rule with an opinion
				// about an empty value, and every rule after it stays silent
				// about one. Reversed, an empty field would be told it is not
				// a valid address, which is true and useless.
				forms.Required("We need an address to reach you"),
				forms.Email(""), // "" takes the rule's own default message
			}},
			{Name: "password", Rules: []forms.Rule{
				forms.Required(""),
				forms.MinLen(8, "Use at least 8 characters"),
			}},
			{Name: "confirm", Rules: []forms.Rule{forms.Required("")}},
			{Name: "terms", Rules: []forms.Rule{
				forms.Accepted("Please accept the terms to continue"),
			}},
		},
		// The one check no single field can make, because it needs to see
		// another field's value. Its message fills in only where the field's
		// own rules had nothing to say — an empty confirmation needs
		// "Required", not "the two passwords differ".
		Validate: func(v forms.Values) map[string]string {
			if v["confirm"] != v["password"] {
				return map[string]string{"confirm": "The two passwords differ"}
			}
			return nil
		},
	})

	if addr := created.Get(); addr != "" {
		return confirmation(addr, func() {
			// Reset puts the form back to its declaration — values, touched,
			// submitted, external errors — so the second visit opens quiet
			// rather than still showing the first one's complaints.
			form.Reset()
			created.Set("")
		})
	}

	return components.Screen{
		Scroll: true,
		Gap:    16,
		Children: []core.View{
			core.Text("Create your account", core.UseStyle(ctx.Theme().Typography.Title)),

			components.FormField{
				Label: "Email",
				Hint:  "We never share it",
				Error: form.Error("email"),
				Input: form.Input("email", "you@example.com"),
			},
			components.FormField{
				Label: "Password",
				Hint:  "At least 8 characters",
				Error: form.Error("password"),
				Input: form.Password("password", "••••••••"),
			},
			components.FormField{
				Label: "Confirm password",
				Error: form.Error("confirm"),
				Input: form.Password("confirm", "••••••••"),
			},

			// A checkbox has no error line of its own — but FormField's Input
			// slot takes any view, so wrapping the row is all it takes to give
			// one to a control that was never designed for it. No Label here:
			// the ListRow's title is the label.
			components.FormField{
				Error: form.Error("terms"),
				Input: components.ListRow{
					Leading: form.Checkbox("terms"),
					Title:   "I accept the terms of service",
				},
			},

			components.Button{
				Label:     "Create account",
				FullWidth: true,
				// Deliberately *not* core.Disabled(!form.Valid()). Under the
				// default reveal policy that is a dead end: nothing explains
				// itself until a submit, and no submit can happen while the
				// button is disabled, so the user gets a form that refuses to
				// work and refuses to say why. Let the submit run and fail —
				// failing is what turns the explanations on.
				OnTap: form.OnSubmit(func(v forms.Values) { submit(form, created, v) }),
			},
		},
	}
}

// submit is the handler Form.Submit calls once every rule has passed.
//
// It runs on the calling goroutine — the native event thread, for a tap — so
// a real back end goes through a goroutine and comes back through SetErrors,
// which is safe to call from anywhere:
//
//	go func() {
//	    if errs := api.CreateAccount(v); errs != nil {
//	        form.SetErrors(errs) // requests a render of its own
//	        return
//	    }
//	    created.Set(v.Trimmed("email"))
//	}()
//
// The values handed in are already a private copy, so the goroutine may
// outlive the render pass that started it.
func submit(form *forms.Form, created core.State[string], v forms.Values) {
	// Trimmed, not raw: Required trims before deciding a field is empty, so a
	// value that passed validation may still be padded.
	email := v.Trimmed("email")

	if registered[strings.ToLower(email)] {
		// An error the rules could not have produced. It shows immediately
		// whatever the reveal policy says, outranks anything the rules have
		// to say about the same field, and disappears the moment the user
		// edits the address — which is the whole point: the verdict was about
		// the old text.
		form.SetErrors(map[string]string{
			"email": "That address is already registered",
		})
		return
	}

	created.Set(email)
}

// confirmation is the post-submit screen. Hook-free by construction: it is
// built inside a branch, and anything that allocated a slot in here would
// shift every slot on the passes where the branch is not taken.
func confirmation(addr string, again func()) core.View {
	return components.Screen{
		Gap: 16,
		Children: []core.View{
			components.Card{
				Title: "Account created",
				Body:  core.Text("A confirmation is on its way to " + addr + "."),
			},
			components.Button{
				Label:    "Create another",
				Emphasis: components.EmphasisOutlined,
				OnTap:    again,
			},
		},
	}
}
