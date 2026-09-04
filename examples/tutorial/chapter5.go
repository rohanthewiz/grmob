package tutorial

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/rohanthewiz/grmob/components"
	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/forms"
)

// chapter5 — Forms & Validation: package forms and the FormField frame. The
// through-line is that everything the user is shown is *derived*: errors are
// recomputed from (values, spec) on every read, the required marker from the
// rules run against "", and visibility from the reveal policy applied to the
// stored touched/blurred/submitted facts. Only the values and those facts are
// stored, which is why nothing in this chapter can ever show a stale
// complaint — there is no error map to forget to invalidate.
func chapter5() Chapter {
	return Chapter{
		Title:   "Forms & Validation",
		Icon:    "📝",
		Summary: "Rules, reveal policies, cross-field checks, server errors — package forms and the FormField frame.",
		Lessons: []Lesson{
			lessonFirstForm(),
			lessonRules(),
			lessonReveal(),
			lessonCrossField(),
			lessonValuesReset(),
		},
	}
}

// --- 5.1 -----------------------------------------------------------------

func lessonFirstForm() Lesson {
	return Lesson{
		Title:   "A form in four calls",
		Summary: "UseForm owns the values, FormField frames the feedback, a bound builder ties them together.",
		Body: func(ctx *core.Context) core.View {
			// The submitted guest's name, "" until a valid submit lands. Lesson
			// state, not form state: the form only knows about its declared
			// fields.
			rsvp := core.NewState(ctx, "")

			form := forms.UseForm(ctx, forms.Spec{
				Fields: []forms.Field{
					{Name: "name", Rules: []forms.Rule{
						forms.Required("Tell us who's coming"),
					}},
					{Name: "email", Rules: []forms.Rule{
						forms.Required("An address for the invite"),
						forms.Email(""), // "" takes the rule's own default message
					}},
				},
			})

			return core.Column(
				core.Gap(14),
				prose("components.FormField has always had an Error slot, and package forms is "+
					"what fills it. A whole form is four calls: declare the fields and their rules "+
					"in a Spec handed to UseForm; frame each input in a FormField; bind the input "+
					"with a bound builder, which writes the field name once instead of three times "+
					"in three roles; and commit through form.OnSubmit, which checks everything and "+
					"calls your handler only when the form is clean."),
				codeBlock(`form := forms.UseForm(ctx, forms.Spec{
    Fields: []forms.Field{
        {Name: "email", Rules: []forms.Rule{
            forms.Required("We need an address to reach you"),
            forms.Email(""),  // "" falls back to the rule's default message
        }},
    },
})

components.FormField{
    Label:    "Email",
    Required: form.Required("email"),  // derived from the rules, not declared
    Hint:     "We never share it",
    Error:    form.Error("email"),
    Input:    form.Input("email", "you@example.com"),
}

components.Button{Label: "Sign up", OnTap: form.OnSubmit(create)}`),
				prose("UseForm is a hook — it consumes exactly one slot on this lesson's context, "+
					"so the rules of hooks apply: unconditional, stable position, every pass. What "+
					"the slot stores is only the values and a few facts (touched, blurred, "+
					"submitted). The errors are *derived* — recomputed from the values and this "+
					"pass's spec on every read. A stored error map has to be invalidated on every "+
					"write, every rule change, every cross-field dependency; a derived one cannot "+
					"be stale by construction. And because the spec is re-read each pass, a rule "+
					"may close over live state and take effect next pass with no re-registration."),
				demoPanel("Tap RSVP while everything is empty — the failed submit is what turns the explanations on.",
					components.FormField{
						Label:    "Name",
						Required: form.Required("name"),
						Error:    form.Error("name"),
						Input:    form.Input("name", "June Gopher"),
					},
					components.FormField{
						Label:    "Email",
						Required: form.Required("email"),
						Hint:     "Used once, for the invite",
						Error:    form.Error("email"),
						Input:    form.Input("email", "june@burrow.dev"),
					},
					components.Button{
						Label: "RSVP",
						// Trimmed, not raw: Required trims before deciding a
						// field is empty, so a value that passed validation may
						// still be padded.
						OnTap: form.OnSubmit(func(v forms.Values) {
							rsvp.Set(v.Trimmed("name"))
						}),
					},
					core.IfElse(rsvp.Get() == "",
						caption("Under the default policy nothing complains until the first submit "+
							"— then every correction is confirmed the instant it lands."),
						caption("✓ RSVP received for "+rsvp.Get()),
					),
				),
				keyPoints(
					"UseForm is a hook: one slot, so call it unconditionally, in a stable position, above any branch that swaps screens.",
					"Errors are derived from (values, this pass's spec) on every read — there is no stored error map to go stale.",
					"The bound builders (form.Input and friends) write the field name once; the unbound spelling names it three times and nothing checks they agree.",
					"OnSubmit records the attempt either way and calls the handler with a private copy of the values only when the form is clean.",
				),
			)
		},
	}
}

// --- 5.2 -----------------------------------------------------------------

// lowercaseOnly is compiled once, at package level, which is the spelling
// forms.Pattern is designed to force: a Spec is rebuilt on every render pass,
// so a rule that compiled its own expression would run regexp.Compile per
// pass, per form — and a typo would panic on a render goroutine instead of at
// startup.
var lowercaseOnly = regexp.MustCompile(`^[a-z]+$`)

// reservedHandles feeds 5.2's custom closure rule — the check no built-in
// covers, written as a plain function because a Rule is one.
var reservedHandles = map[string]bool{"root": true, "admin": true}

func lessonRules() Lesson {
	return Lesson{
		Title:   "Rules & the required marker",
		Summary: "The first failing rule speaks, empties are Required's subject alone, and the asterisk is derived.",
		Body: func(ctx *core.Context) core.View {
			// Which rules this pass's spec carries. Three hook slots, claimed
			// before UseForm's — all unconditional, so the cursor never drifts.
			reqOn := core.NewState(ctx, true)
			minOn := core.NewState(ctx, true)
			patOn := core.NewState(ctx, false)

			// The rule list is assembled fresh every pass from the checkboxes —
			// legal because only the form's record survives between passes; the
			// spec is whatever this render hands in. Toggling a rule changes the
			// error AND the required marker on the very next pass.
			var rules []forms.Rule
			if reqOn.Get() {
				rules = append(rules, forms.Required("Every gopher needs a handle"))
			}
			if minOn.Get() {
				rules = append(rules, forms.MinLen(5, "Give it at least 5 characters"))
			}
			if patOn.Get() {
				rules = append(rules, forms.Pattern(lowercaseOnly, "Lowercase letters only"))
			}
			rules = append(rules, func(v string) string {
				if reservedHandles[strings.ToLower(strings.TrimSpace(v))] {
					return "That handle is reserved for the system"
				}
				return ""
			})

			form := forms.UseForm(ctx, forms.Spec{
				// RevealAlways is mostly for tests — and for exactly this: a
				// playground whose point is watching the rules react per
				// keystroke. Real forms want 5.3's kinder policies.
				Reveal: forms.RevealAlways,
				Fields: []forms.Field{{Name: "handle", Rules: rules}},
			})

			return core.Column(
				core.Gap(14),
				prose("A Rule is func(value string) string — the message, or \"\" when nothing "+
					"is wrong. Two behaviors carry the whole design. The first failing rule wins: "+
					"a field shows one line of feedback, so ordering the rules is choosing which "+
					"complaint is the most useful one, and Required belongs first. And every rule "+
					"except Required and Accepted is silent about an empty value — emptiness is "+
					"Required's subject and nobody else's, so an optional field carrying MinLen "+
					"says nothing until there is something to measure."),
				codeBlock(`{Name: "handle", Rules: []forms.Rule{
    forms.Required(""),            // the only rule that minds emptiness
    forms.MinLen(5, ""),           // silent about "" — not its subject
    forms.Pattern(lowercase, ""),  // *regexp.Regexp: compiled once, hoisted
    func(v string) string {        // an app's own check is a plain func
        if reserved[v] {
            return "That one's taken"
        }
        return ""
    },
}}`),
				prose("The asterisk FormField draws beside a required label is fed from the form, "+
					"and the form *derives* it: form.Required runs the field's rules against \"\" "+
					"and answers whether any of them complains. There is deliberately no "+
					"Field.Required flag — a flag would be a second claim about the same field, "+
					"correct only while someone keeps it in step with the rules, and the failures "+
					"it allows (a starred field that submits empty, an unstarred one that won't) "+
					"are exactly what the marker exists to prevent. Watch it below: the Required "+
					"checkbox takes the asterisk with it."),
				demoPanel("Compose the rule list live — the spec is re-read every pass, so a toggled rule applies instantly.",
					checkRow("Required", reqOn),
					checkRow("MinLen(5)", minOn),
					checkRow("Pattern: lowercase only", patOn),
					components.FormField{
						Label:    "Handle",
						Required: form.Required("handle"),
						Error:    form.Error("handle"),
						Input:    form.Input("handle", "gopherella"),
					},
					caption(fmt.Sprintf("form.Required(%q) → %v — FormField's asterisk is fed this bool",
						"handle", form.Required("handle"))),
					caption("The closure rule is always in the list — try \"root\" or \"admin\"."),
				),
				keyPoints(
					"The first failing rule wins — order the rules by usefulness, Required first.",
					"Every rule but Required and Accepted ignores an empty value; without that, an optional MinLen field would scold an untouched form.",
					"A custom rule is a plain closure: pure, synchronous, no I/O — it runs on every read of the errors.",
					"form.Required is derived by running the rules against \"\" — the marker is exactly as live as the rules, with no flag to drift.",
					"Pattern takes a compiled *regexp.Regexp: the spec is rebuilt per pass, so hoist the compile to a package var.",
				),
			)
		},
	}
}

// --- 5.3 -----------------------------------------------------------------

// revealNames double as the 5.3 segment captions and the constant-name
// suffixes ("OnBlur" → forms.RevealOnBlur) — the chapter-4 trick that keeps
// captions and code from disagreeing. revealValues is the parallel value
// table, ordered as the constants are declared, so index 0 is the zero value.
var (
	revealNames  = []string{"OnSubmit", "OnBlur", "OnTouch", "Always"}
	revealValues = []forms.Reveal{
		forms.RevealOnSubmit,
		forms.RevealOnBlur,
		forms.RevealOnTouch,
		forms.RevealAlways,
	}
)

func lessonReveal() Lesson {
	return Lesson{
		Title:   "When errors appear",
		Summary: "Reward early, punish late: the four reveal policies, and why the submit button stays enabled.",
		Body: func(ctx *core.Context) core.View {
			policy := core.NewState(ctx, 0)

			form := forms.UseForm(ctx, forms.Spec{
				// Reveal is read from live state, so switching the segment
				// re-polices the same record — touched, blurred and submitted
				// persist until Reset, which is what the Start-over button is
				// for. (A switched policy also changes whether the field
				// registers onBlur, which shifts later callback IDs for one
				// pass — the same settle-next-pass shift any conditional
				// subtree causes.)
				Reveal: revealValues[policy.Get()],
				Fields: []forms.Field{
					{Name: "email", Rules: []forms.Rule{
						forms.Required("An address is needed"),
						forms.Email(""),
					}},
				},
			})

			return core.Column(
				core.Gap(14),
				prose("Validating as the user types is hostile: the second character of an "+
					"address is not yet a valid address, and saying so is scolding someone for "+
					"not having finished. The rule of thumb is reward early, punish late — say "+
					"nothing until the user claims to be done, then stay live so every correction "+
					"is confirmed the instant it lands. Spec.Reveal picks the moment: the first "+
					"submit (the default), leaving the field, the first edit, or always. The "+
					"policies are cumulative, not exclusive — a submit reveals everything under "+
					"all four, so a form can never refuse to submit while showing no reason."),
				codeBlock(`forms.Spec{
    Reveal: forms.RevealOnBlur,  // speaks when the user leaves the field
    Fields: ...,
}

// The bound builders attach the blur listener themselves — and only under
// RevealOnBlur, so no other form pays for an event nothing reads. A control
// built by hand out of Value and OnChange must report the edge itself:
core.Input(form.Value("email"), "you@example.com",
    form.OnChange("email"),
    core.OnBlur(form.OnBlur("email")))`),
				prose("RevealOnBlur is the closest thing to the rule of thumb that still speaks "+
					"before the submit: leaving a field is the user's own claim to have finished "+
					"it, so a complaint then is an answer, not an interruption. It is the right "+
					"default for multi-field forms — examples/signup uses it. One trap to refuse: "+
					"do not disable the submit button on !form.Valid(). Under the default policy "+
					"that is a dead end — nothing explains itself until a submit, and no submit "+
					"can happen while the button is disabled. Let the submit run and fail; "+
					"failing is the event that turns the explanations on."),
				demoPanel("Pick a policy, type an unfinished address, leave the field, submit — watch when it speaks.",
					components.SegmentedControl{
						Style:     segWrap,
						Labels:    revealNames,
						Selected:  policy.Get(),
						OnSelect:  func(i int) { policy.Set(i) },
						KeyPrefix: "reveal-",
					},
					components.FormField{
						Label:    "Email",
						Required: form.Required("email"),
						Hint:     "Errors replace this hint when revealed",
						Error:    form.Error("email"),
						Input:    form.Input("email", "you@burrow.dev"),
					},
					// The reveal inputs, instrumented. Blurred reports what has
					// been *observed*: under any policy but OnBlur no listener
					// is attached, so it stays false — the honest answer, not a
					// bug.
					caption(fmt.Sprintf("touched: %v · blurred: %v · submitted: %v",
						form.Touched("email"), form.Blurred("email"), form.Submitted())),
					core.If(form.Submitted() && form.Valid(),
						caption("✓ that submit would have gone through"),
					),
					core.Row(
						core.Gap(8),
						components.Button{
							Label: "Check the form",
							// A nil handler still records the attempt — which
							// is the half of Submit this lesson is about.
							OnTap: form.OnSubmit(nil),
						},
						components.Button{
							Label:    "Start over",
							Emphasis: components.EmphasisOutlined,
							OnTap:    func() { form.Reset() },
						},
					),
					caption("Start over between experiments: touched, blurred and submitted "+
						"persist until Reset, and a submit reveals under every policy."),
				),
				keyPoints(
					"The policies are cumulative: a submit reveals everything under all four — a hidden reason to refuse a submit would be a dead end.",
					"RevealOnBlur is 'reward early, punish late' for multi-field forms; OnTouch fires on the second keystroke, so save it for unguessable formats.",
					"The blur listener is attached only under RevealOnBlur — a hand-built control must attach core.OnBlur(form.OnBlur(name)) itself.",
					"Never disable the submit on !form.Valid(): the failed submit is what turns the explanations on. Disabled is for a submit in flight.",
				),
			)
		},
	}
}

// --- 5.4 -----------------------------------------------------------------

// takenAddresses stands in for the one check a client cannot make: whether an
// address already has an account. A real app asks a server — which is why the
// answer arrives through Form.SetErrors rather than a Rule, since a rule must
// be pure and synchronous and a network call is neither.
var takenAddresses = map[string]bool{"taken@example.com": true}

func lessonCrossField() Lesson {
	return Lesson{
		Title:   "Cross-field & server errors",
		Summary: "Validate sees every value at once; SetErrors installs the verdicts only a server can reach.",
		Body: func(ctx *core.Context) core.View {
			// The address just claimed, "" while the form is still up.
			claimed := core.NewState(ctx, "")

			// Names the email field so the server-error path can put the cursor
			// back in it. A hook — hoisted up here with the others, because a
			// ref built inline is a new pointer every pass: FocusTarget would
			// stamp one identity and Focus would compare against another.
			emailRef := core.UseFocusRef(ctx)

			form := forms.UseForm(ctx, forms.Spec{
				Fields: []forms.Field{
					{Name: "email", Rules: []forms.Rule{
						forms.Required(""),
						forms.Email(""),
					}},
					{Name: "password", Rules: []forms.Rule{
						forms.Required(""),
						forms.MinLen(8, "Use at least 8 characters"),
					}},
					{Name: "confirm", Rules: []forms.Rule{forms.Required("")}},
				},
				// The pass that sees every value at once. Its message fills in
				// only where the field's own rules said nothing: an empty
				// confirmation needs "Required", not a mismatch complaint that
				// is true, unhelpful, and what last-writer-wins would show.
				Validate: func(v forms.Values) map[string]string {
					if v["confirm"] != v["password"] {
						return map[string]string{"confirm": "These don't match the password above"}
					}
					return nil
				},
			})

			return core.Column(
				core.Gap(14),
				prose("A field's rules see one string. Two checks need more: the comparison no "+
					"single field can make, and the verdict no client can reach. Spec.Validate is "+
					"the first — it runs after every field's rules with all the values, and its "+
					"messages fill in only for fields that don't already have one, because the "+
					"field's own rule is the more specific complaint. A key nothing renders is a "+
					"form-level error a banner can read with form.Error(\"form\")."),
				codeBlock(`Validate: func(v forms.Values) map[string]string {
    if v["confirm"] != v["password"] {
        return map[string]string{"confirm": "The two passwords differ"}
    }
    return nil
},

// And after the server answers what no rule could know:
form.SetErrors(map[string]string{
    "email": "That address is already registered",
})`),
				prose("SetErrors installs the second kind — uniqueness, authorization, business "+
					"rules. Three behaviors follow from where such an error came from: it ignores "+
					"the reveal policy (a message that came back from a submit is by definition "+
					"post-submit), it outranks a rule's message on the same field (it is the newer "+
					"information), and it is dropped the moment that field changes (the verdict "+
					"was about the old text, so it disappears as the user starts fixing it — not "+
					"after another round trip). The demo pairs it with core.Focus: the submit has "+
					"closed the keyboard on the field, so the message alone would leave the user "+
					"to find it again."),
				demoPanel("taken@example.com already has an account — try claiming it.",
					components.FormField{
						Label:    "Email",
						Required: form.Required("email"),
						Error:    form.Error("email"),
						Input:    form.Input("email", "gopher@burrow.dev", core.FocusTarget(emailRef)),
					},
					components.FormField{
						Label:    "Password",
						Required: form.Required("password"),
						Hint:     "At least 8 characters",
						Error:    form.Error("password"),
						Input:    form.Password("password", "choose a password"),
					},
					components.FormField{
						Label:    "Confirm password",
						Required: form.Required("confirm"),
						Error:    form.Error("confirm"),
						Input:    form.Password("confirm", "type it again"),
					},
					components.Button{
						Label: "Claim address",
						OnTap: form.OnSubmit(func(v forms.Values) {
							addr := v.Trimmed("email")
							if takenAddresses[strings.ToLower(addr)] {
								form.SetErrors(map[string]string{
									"email": "Someone got there first — that address is registered",
								})
								// Reopen the keyboard on the problem field; the
								// platform scrolls to whatever it focuses.
								core.Focus(emailRef)
								return
							}
							claimed.Set(addr)
						}),
					},
					core.IfElse(claimed.Get() == "",
						caption("No address claimed yet."),
						caption("✓ claimed "+claimed.Get()),
					),
				),
				keyPoints(
					"Spec.Validate is the only pass that sees every value — it runs after the field rules, and only fills gaps they left.",
					"Field rules outrank Validate's message on the same field: \"Required\" beats a mismatch complaint that is true but unhelpful.",
					"SetErrors is reveal-blind, outranks the rules, and each entry drops on that field's first edit — all three follow from it being a server's verdict on old text.",
					"Pair a server error with core.Focus on a UseFocusRef target: put the cursor (and keyboard, and scroll) where the problem is.",
				),
			)
		},
	}
}

// --- 5.5 -----------------------------------------------------------------

func lessonValuesReset() Lesson {
	return Lesson{
		Title:   "Values, initials & reset",
		Summary: "Values are text all the way down — typed reads are methods — and Reset returns to the declaration.",
		Body: func(ctx *core.Context) core.View {
			// The confirmed order line, "" until a valid submit. Lesson state:
			// Reset only owns the form's record, so Start over clears both.
			placed := core.NewState(ctx, "")

			form := forms.UseForm(ctx, forms.Spec{
				Fields: []forms.Field{
					{Name: "quantity", Initial: "2", Rules: []forms.Rule{
						forms.Required(""),
						forms.Range(1, 12, "Order between 1 and 12 gophers"),
					}},
					// No rules: the box is optional, so form.Required reports
					// false and FormField would draw no marker.
					{Name: "gift", Initial: "true"},
				},
			})

			// Derived live, like everything else on display: the raw text and
			// what Int makes of it, re-read every pass.
			qty, qtyOK := form.Values().Int("quantity")

			return core.Column(
				core.Gap(14),
				prose("forms.Values is map[string]string, and strings all the way down is "+
					"deliberate: every event a native control sends is a string on the wire, and "+
					"keeping the raw text is what makes validation possible at all — \"12x\" has "+
					"to survive long enough for Range to complain about it, and a map of ints has "+
					"nowhere to put it. That is also why a validated number uses core.Input, not "+
					"core.NumericInput: NumericInput parses in its change callback and drops the "+
					"event when the text does not parse, so an unparseable value never reaches "+
					"the form and the rule can never fire."),
				codeBlock(`v.Trimmed("email")  // Required trims, so a valid value may still be padded
v.Bool("gift")      // checkbox: anything not "true" is false
v.Int("quantity")   // (int, bool) — the field is free text, the ok is real

{Name: "quantity", Initial: "2",
    Rules: []forms.Rule{forms.Required(""), forms.Range(1, 12, "")}},

form.Reset()  // back to the declaration: values, touched, submitted, errors`),
				prose("Field.Initial seeds a value the first time the name is seen, and again "+
					"after Reset — never in between, so a field the user has cleared stays "+
					"cleared even though the spec still names a default. Reset returns the whole "+
					"form to its declaration: every field to its Initial, nothing touched, "+
					"nothing submitted, no external errors. Initials are re-read from this "+
					"pass's spec, which is also the prefill trick for data that arrives late: "+
					"render the loaded values as Initial and Reset once they land."),
				demoPanel("The quantity field is free text — feed it \"12x\" and watch Range complain while Int reports (0, false).",
					components.FormField{
						Label:    "Quantity",
						Required: form.Required("quantity"),
						Hint:     "1–12 per order",
						Error:    form.Error("quantity"),
						Input:    form.Input("quantity", "how many?"),
					},
					caption(fmt.Sprintf("Values().Int(%q) → (%d, %v)", "quantity", qty, qtyOK)),
					// The checkbox's label belongs to the ListRow, which is why
					// this field carries no FormField label (and no marker) —
					// the signup example's terms row, same reasoning.
					components.ListRow{
						Leading: form.Checkbox("gift"),
						Title:   "Gift-wrap the shipment",
					},
					core.Row(
						core.Gap(8),
						components.Button{
							Label: "Place order",
							OnTap: form.OnSubmit(func(v forms.Values) {
								n, _ := v.Int("quantity")
								order := fmt.Sprintf("order placed: quantity %d", n)
								if v.Bool("gift") {
									order += ", gift-wrapped"
								}
								placed.Set(order)
							}),
						},
						components.Button{
							Label:    "Start over",
							Emphasis: components.EmphasisOutlined,
							OnTap: func() {
								form.Reset()
								placed.Set("")
							},
						},
					),
					core.IfElse(placed.Get() == "",
						caption("No order yet — quantity opens at its Initial of 2, gift-wrap ticked."),
						caption("✓ "+placed.Get()),
					),
				),
				keyPoints(
					"Values is map[string]string: the wire carries text, and validation needs the raw text to survive.",
					"Typed reads are methods — Trimmed, Bool, Int, Float — and the (value, ok) pair is honest because the field is free text.",
					"A validated number is core.Input plus a rule; NumericInput drops unparseable events before the form ever sees them.",
					"Initial seeds once per name (and again after Reset); a cleared field stays cleared while the spec still names its default.",
					"Reset returns the form to this pass's declaration — which is also how late-arriving data prefills a form.",
				),
			)
		},
	}
}
