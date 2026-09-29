// Package signup is the worked example for package forms: a sign-up screen
// that exercises every part of the validation story in one screen.
//
//	rules            Required / Email / MinLen / Accepted, one message each
//	cross-field      the confirmation must match the password
//	reveal policy    each field complains when the user leaves it, then live
//	server errors    a uniqueness check only the back end can make
//	the widget       comps.FormField, whose Error slot has been waiting
//	                 for something to fill it since it was written
//	verification     a six-digit code in comps.PINInput, the step a real
//	                 sign-up takes between "submitted" and "created"
//
// Every field is declared once, in the Spec, and rendered through a bound
// builder — so the name that reads the value is the same name that writes it,
// by construction rather than by care.
package signup

import (
	"strings"

	"github.com/rohanthewiz/grmob/comps"
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
	// The address a code was sent to, while the verification step is up; ""
	// otherwise. The code typed so far and the message a wrong one earned
	// live beside it, up here with the other hooks.
	pending := core.NewState(ctx, "")
	code := core.NewState(ctx, "")
	codeErr := core.NewState(ctx, "")

	// Names the email field so the server-error path below can put the cursor
	// back in it. A hook, and therefore unconditional and up here with the
	// others: a ref built inline would be a new pointer every pass, so
	// FocusTarget would stamp one identity and Focus would compare against
	// another and the field would never focus.
	emailField := core.UseFocusRef(ctx)
	passwordField := core.UseFocusRef(ctx)
	confirmField := core.UseFocusRef(ctx)

	// The order the return key walks. One line, declared above the fields it
	// names — membership is read while each field's props are stamped, so a
	// field rendered before this call would see the previous pass's order.
	//
	// It gives the first two fields a keyboard whose action key reads "Next"
	// and moves the cursor down the form; the confirm field has no successor,
	// so its key stays the platform's default. That is the shape a signup
	// form wants: three fields filled without the user reaching back to the
	// screen between them.
	//
	// The terms checkbox is deliberately not in the order. No platform here
	// gives a checkbox keyboard focus, so a fourth entry would advertise a
	// Next that lands nowhere.
	core.UseFocusOrder(ctx, emailField, passwordField, confirmField)

	form := forms.UseForm(ctx, forms.Spec{
		// RevealOnBlur: each field explains itself the moment the user is
		// done with it, and every field's error appears once a submit is
		// attempted. Leaving a field is the user's own claim to have
		// finished it, so a complaint then is an answer rather than an
		// interruption — where RevealOnTouch would say "not a valid address"
		// two characters in, and the default RevealOnSubmit would make the
		// user fill in four fields before hearing about the first.
		//
		// It costs nothing at the call sites: the bound builders below attach
		// core.OnBlur themselves under this policy.
		Reveal: forms.RevealOnBlur,
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
			// The picker, declared with an Initial rather than a Required
			// rule. Both shapes are legitimate and they are different forms:
			// a picker with no sensible default opens empty and complains
			// when the user submits without choosing, while this one opens on
			// the answer most people want and the user changes it or does
			// not. A signup that made someone pick a plan before it would let
			// them in is the worse of the two.
			{Name: "plan", Initial: "free"},
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

	if addr := pending.Get(); addr != "" {
		// A scope of its own: PINInput holds hooks (its focus state), and a
		// hook inside a branch would shift every slot after it on the passes
		// where the branch is not taken. The scope owns its slots, so the
		// step can come and go.
		return core.ComponentFunc(func(c *core.Context) *core.Node {
			return verification(addr, code, codeErr, func() {
				// The right code: the account exists now.
				pending.Set("")
				code.Set("")
				created.Set(addr)
			}, func() {
				// A different address: back to the form, which kept its
				// values, with nothing left of this attempt.
				pending.Set("")
				code.Set("")
				codeErr.Set("")
			}).Render(c.Scope("verify"))
		})
	}

	if addr := created.Get(); addr != "" {
		return confirmation(addr, func() {
			// Reset puts the form back to its declaration — values, touched,
			// blurred, submitted, external errors — so the second visit opens
			// quiet rather than still showing the first one's complaints.
			form.Reset()
			created.Set("")
		})
	}

	return comps.Screen{
		Scroll: true,
		// The form is taller than a phone with the keyboard up, and the
		// terms row and the button are the parts that go under it. Set here,
		// the scroll region ends where the keyboard begins, which is what
		// gives the platform's scroll-the-focused-field-into-view somewhere
		// visible to put the field. See core.KeyboardAware.
		KeyboardAware: true,
		Gap:           16,
		Children: []core.View{
			core.Text("Create your account", core.UseStyle(ctx.Theme().Typography.Title)),

			// Required is asked of the form rather than written as true: the
			// marker is then the same fact as the rule, and dropping
			// forms.Required from the spec above takes the asterisk with it
			// instead of leaving a screen that stars a field it will happily
			// accept empty.
			comps.FormField{
				Label:    "Email",
				Required: form.Required("email"),
				Hint:     "We never share it",
				Error:    form.Error("email"),
				Input:    form.Input("email", "you@example.com", core.FocusTarget(emailField)),
			},
			comps.FormField{
				Label:    "Password",
				Required: form.Required("password"),
				Hint:     "At least 8 characters",
				Error:    form.Error("password"),
				Input:    form.Password("password", "••••••••", core.FocusTarget(passwordField)),
			},
			comps.FormField{
				Label:    "Confirm password",
				Required: form.Required("confirm"),
				Error:    form.Error("confirm"),
				Input:    form.Password("confirm", "••••••••", core.FocusTarget(confirmField)),
			},

			// The picker. Not in the focus order above and deliberately: the
			// order is the return key's path through the *text* fields, and a
			// picker takes no keyboard focus on either phone — a fourth entry
			// would advertise a Next that lands nowhere, which is the same
			// reason the terms checkbox is left out.
			comps.FormField{
				Label: "Plan",
				Hint:  "You can change this later",
				Error: form.Error("plan"),
				Input: form.Select("plan", []core.SelectOption{
					{Value: "free", Label: "Free"},
					{Value: "pro", Label: "Pro — $9/month"},
					// No label: the value is what a reader would want to see
					// anyway, and core.Select fills it in.
					{Value: "Team"},
				}),
			},

			// A checkbox has no error line of its own — but FormField's Input
			// slot takes any view, so wrapping the row is all it takes to give
			// one to a control that was never designed for it. No Label here:
			// the ListRow's title is the label — which is also why Required is
			// left off, since the marker has nothing to sit beside. (The field
			// *is* required: Accepted rejects an unticked box, and
			// form.Required("terms") would say so.)
			comps.FormField{
				Error: form.Error("terms"),
				Input: comps.ListRow{
					Leading: form.Checkbox("terms"),
					Title:   "I accept the terms of service",
				},
			},

			comps.Button{
				Label:     "Create account",
				FullWidth: true,
				// Deliberately *not* core.Disabled(!form.Valid()). Under the
				// default reveal policy that is a dead end: nothing explains
				// itself until a submit, and no submit can happen while the
				// button is disabled, so the user gets a form that refuses to
				// work and refuses to say why. Let the submit run and fail —
				// failing is what turns the explanations on.
				OnTap: form.OnSubmit(func(v forms.Values) { submit(form, pending, emailField, v) }),
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
//
// Success does not create the account yet: it sends a code (see
// verification), so what it sets is the pending address.
func submit(form *forms.Form, pending core.State[string], emailField *core.FocusRef, v forms.Values) {
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
		// And put the cursor where the problem is. The submit has almost
		// certainly scrolled past the email field or closed the keyboard on
		// it, so an error message alone leaves the user to find the field
		// again — core.Focus both reopens the keyboard and brings the field
		// into view, because the platform scrolls to whatever it focuses.
		//
		// It is safe to issue from here even though this runs on the native
		// event thread mid-dispatch: the command only sets state and requests
		// a render, and the pass the dispatch already schedules carries the
		// stamp out.
		core.Focus(emailField)
		return
	}

	pending.Set(email)
}

// demoCode is the code this example "sends". There is no mail server, so the
// step says what it is; a real app compares against what its back end
// issued, and does it on the server.
const demoCode = "246810"

// verification is the step between a submit that passed and an account that
// exists: the code sent to the address, typed into comps.PINInput.
//
// OnComplete checks the code the moment the sixth digit lands, so there is no
// Verify button to reach for. A wrong code is cleared with a message rather
// than left for the reader to delete six digits of: the field is controlled,
// so emptying code empties the boxes.
//
// Hook-free itself; the PINInput inside it holds hooks, which is why App
// renders this in a scope of its own.
func verification(addr string, code, codeErr core.State[string], verified, restart func()) core.View {
	return comps.Screen{
		Gap: 16,
		Children: []core.View{
			comps.Card{
				Title: "Check your inbox",
				Body: core.Text("We sent a six-digit code to " + addr + ". " +
					"This example has no mail server, so here it is: " + demoCode + "."),
			},
			comps.FormField{
				Label: "Verification code",
				Error: codeErr.Get(),
				Input: comps.PINInput{
					Length: 6,
					Label:  "Verification code",
					Value:  code.Get(),
					OnChange: func(v string) {
						code.Set(v)
						// The reader is answering the message.
						if v != "" {
							codeErr.Set("")
						}
					},
					OnComplete: func(v string) {
						if v == demoCode {
							codeErr.Set("")
							verified()
							return
						}
						code.Set("")
						codeErr.Set("That code is not the one we sent")
					},
				},
			},
			comps.Button{
				Label:    "Use a different address",
				Emphasis: comps.EmphasisOutlined,
				OnTap:    restart,
			},
		},
	}
}

// confirmation is the post-submit screen. Hook-free by construction: it is
// built inside a branch, and anything that allocated a slot in here would
// shift every slot on the passes where the branch is not taken.
func confirmation(addr string, again func()) core.View {
	return comps.Screen{
		Gap: 16,
		Children: []core.View{
			comps.Card{
				Title: "Account created",
				Body:  core.Text("A confirmation is on its way to " + addr + "."),
			},
			comps.Button{
				Label:    "Create another",
				Emphasis: comps.EmphasisOutlined,
				OnTap:    again,
			},
		},
	}
}
