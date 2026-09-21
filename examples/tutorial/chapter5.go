package tutorial

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/rohanthewiz/grmob/comps"
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
			lessonPicker(),
			lessonPINInput(),
			lessonTagInput(),
			lessonInputFamily(),
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
				prose("comps.FormField has always had an Error slot, and package forms is "+
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

comps.FormField{
    Label:    "Email",
    Required: form.Required("email"),  // derived from the rules, not declared
    Hint:     "We never share it",
    Error:    form.Error("email"),
    Input:    form.Input("email", "you@example.com"),
}

comps.Button{Label: "Sign up", OnTap: form.OnSubmit(create)}`),
				prose("UseForm is a hook — it consumes exactly one slot on this lesson's context, "+
					"so the rules of hooks apply: unconditional, stable position, every pass. What "+
					"the slot stores is only the values and a few facts (touched, blurred, "+
					"submitted). The errors are *derived* — recomputed from the values and this "+
					"pass's spec on every read. A stored error map has to be invalidated on every "+
					"write, every rule change, every cross-field dependency; a derived one cannot "+
					"be stale by construction. And because the spec is re-read each pass, a rule "+
					"may close over live state and take effect next pass with no re-registration."),
				demoPanel("Tap RSVP while everything is empty — the failed submit is what turns the explanations on.",
					comps.FormField{
						Label:    "Name",
						Required: form.Required("name"),
						Error:    form.Error("name"),
						Input:    form.Input("name", "June Gopher"),
					},
					comps.FormField{
						Label:    "Email",
						Required: form.Required("email"),
						Hint:     "Used once, for the invite",
						Error:    form.Error("email"),
						Input:    form.Input("email", "june@burrow.dev"),
					},
					comps.Button{
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
					comps.FormField{
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
					comps.SegmentedControl{
						Style:     segWrap,
						Labels:    revealNames,
						Selected:  policy.Get(),
						OnSelect:  func(i int) { policy.Set(i) },
						KeyPrefix: "reveal-",
					},
					comps.FormField{
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
						comps.Button{
							Label: "Check the form",
							// A nil handler still records the attempt — which
							// is the half of Submit this lesson is about.
							OnTap: form.OnSubmit(nil),
						},
						comps.Button{
							Label:    "Start over",
							Emphasis: comps.EmphasisOutlined,
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
					comps.FormField{
						Label:    "Email",
						Required: form.Required("email"),
						Error:    form.Error("email"),
						Input:    form.Input("email", "gopher@burrow.dev", core.FocusTarget(emailRef)),
					},
					comps.FormField{
						Label:    "Password",
						Required: form.Required("password"),
						Hint:     "At least 8 characters",
						Error:    form.Error("password"),
						Input:    form.Password("password", "choose a password"),
					},
					comps.FormField{
						Label:    "Confirm password",
						Required: form.Required("confirm"),
						Error:    form.Error("confirm"),
						Input:    form.Password("confirm", "type it again"),
					},
					comps.Button{
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
					comps.FormField{
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
					comps.ListRow{
						Leading: form.Checkbox("gift"),
						Title:   "Gift-wrap the shipment",
					},
					core.Row(
						core.Gap(8),
						comps.Button{
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
						comps.Button{
							Label:    "Start over",
							Emphasis: comps.EmphasisOutlined,
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

// --- 5.6 -----------------------------------------------------------------

func lessonPicker() Lesson {
	return Lesson{
		Title:   "Pickers: one value from a list",
		Summary: "core.Select stores the option's value, wears the theme's field style, and draws its own frame on every target.",
		Body: func(ctx *core.Context) core.View {
			// The confirmed booking, "" until a valid submit — lesson state,
			// as everywhere else in this chapter.
			booked := core.NewState(ctx, "")

			form := forms.UseForm(ctx, forms.Spec{
				Reveal: forms.RevealOnBlur,
				Fields: []forms.Field{
					// Two pickers, declared the two different ways, which is
					// the lesson's whole point. The class opens on a default
					// and needs no rule; the seat opens empty and is required,
					// which is what the leading blank option is for.
					{Name: "class", Initial: "economy"},
					{Name: "seat", Rules: []forms.Rule{
						forms.Required("Pick a seat before we can book it"),
					}},
				},
			})

			return core.Column(
				core.Gap(14),
				prose("core.Select is the picker: one value chosen from a fixed list. It stores the "+
					"option's Value, never its label or its index — the index is the one identity "+
					"that changes when a list is reordered, and a label is written to be read. So "+
					"every rule that reads a string reads it unchanged, and forms.Required rejects "+
					"an unchosen picker exactly as it rejects an empty field."),
				codeBlock(`form.Select("class", []core.SelectOption{
    {Value: "economy", Label: "Economy"},
    {Value: "business", Label: "Business"},
    {Value: "First"},                 // no label: the value is the label
})

{Name: "class", Initial: "economy"}   // opens on a default, no rule needed
{Name: "seat", Rules: []forms.Rule{forms.Required("")}}  // opens empty`),
				prose("A picker with a sensible default is declared with Field.Initial and no rule; "+
					"one without is declared with a leading empty option and a Required rule. Those "+
					"are two different forms and the widget takes no position on which is meant — "+
					"asking someone to choose a shipping speed before you will let them in is a "+
					"choice, and so is opening on the cheapest one."),
				demoPanel("Leave the seat picker without choosing: under RevealOnBlur a text field would complain, and this one does not.",
					comps.FormField{
						Label: "Cabin",
						Hint:  "Opens on its Initial",
						Error: form.Error("class"),
						Input: form.Select("class", []core.SelectOption{
							{Value: "economy", Label: "Economy"},
							{Value: "business", Label: "Business"},
							{Value: "First"},
						}),
					},
					comps.FormField{
						Label:    "Seat",
						Required: form.Required("seat"),
						Error:    form.Error("seat"),
						// The two optional fields, on the picker where they
						// read as the aircraft rather than as a feature demo:
						// Group files consecutive options under a heading, and
						// Disabled greys one out without taking it away.
						Input: form.Select("seat", []core.SelectOption{
							{Value: "", Label: "Choose a seat…"},
							{Value: "aisle", Label: "Aisle", Group: "Front cabin"},
							{Value: "window", Label: "Window", Group: "Front cabin"},
							{Value: "rear-aisle", Label: "Aisle", Group: "Rear cabin"},
							{Value: "exit", Label: "Exit row", Group: "Rear cabin", Disabled: true},
						}),
					},
					caption(fmt.Sprintf("class = %q   seat = %q",
						form.Values()["class"], form.Values()["seat"])),
					comps.Button{
						Label: "Book it",
						OnTap: form.OnSubmit(func(v forms.Values) {
							booked.Set(fmt.Sprintf("%s, %s seat", v["class"], v["seat"]))
						}),
					},
					core.IfElse(booked.Get() == "",
						caption("Nothing booked yet."),
						caption("✓ "+booked.Get()),
					),
				),
				prose("A picker is a field, so it reads the theme's Components.Input base and matches "+
					"the text inputs beside it — the frame, the radius, the fill and the padding all "+
					"arrive from there. That inheritance is load-bearing rather than cosmetic: on the "+
					"web the browser draws a <select> a frame of its own, and the framework writes "+
					"border:none over it precisely because the theme has one to put there. The same "+
					"reset a text field gets, for the same reason."),
				prose("Neither phone builds this from its platform picker. SwiftUI's .pickerStyle(.menu) "+
					"and Material's ExposedDropdownMenuBox each draw a frame and an indicator no Go "+
					"style can remove, which would leave one control in the vocabulary whose edge came "+
					"from the platform on two targets and from the theme on the other two. So each "+
					"native draws the style's own box and hangs a menu off it — a SwiftUI Menu, a "+
					"Compose DropdownMenu — and the only thing the platform supplies is the behaviour."),
				prose("Whether the list is showing is not something Go knows, and should not be: there "+
					"is no prop for it and no patch describes it. The selection stays controlled like "+
					"every other input's value; the open state belongs to whichever renderer is "+
					"drawing the menu. A picker that closed on every unrelated re-render would be "+
					"unusable, and that is exactly what putting the flag in the tree would cause."),
				prose("The seat picker above carries the option list's other two fields. Group files "+
					"an option under a heading — an <optgroup> on the web, a Section in the iOS menu, "+
					"an unclickable heading item in the Android dropdown — and Disabled greys one out "+
					"without taking it away. Note that the two aisle seats share a label and differ "+
					"in value, which is exactly the case the Value rule exists for: what the form "+
					"stores is \"rear-aisle\", not \"Aisle\" and not index 3."),
				codeBlock(`{Value: "aisle",      Label: "Aisle",    Group: "Front cabin"},
{Value: "window",     Label: "Window",   Group: "Front cabin"},
{Value: "rear-aisle", Label: "Aisle",    Group: "Rear cabin"},
{Value: "exit",       Label: "Exit row", Group: "Rear cabin", Disabled: true},`),
				prose("Grouping is by *runs*, not by gathering: consecutive options sharing a heading "+
					"become one section, in the order they were written. Put a Front cabin seat back "+
					"between the two Rear cabin ones and you get three sections, because the list's "+
					"order is yours — it is what a person sees and what the keyboard walks — and no "+
					"renderer is going to rearrange it to tidy up the headings. Sorting a list into "+
					"its sections is a line of Go at the call site; un-sorting one is not."),
				prose("A disabled option is drawn, announced, and unchoosable. That is the point of "+
					"disabling one rather than leaving it out: an option that vanishes takes its "+
					"explanation with it, and a list that changes length between renders is one a "+
					"person has to re-read. It does not stop *Go* from setting the value — a Select "+
					"shows whatever value it was passed, which is the same contract an out-of-list "+
					"value lands under."),
				keyPoints(
					"core.Select stores the option's Value — not its label, and never its index.",
					"A default is Field.Initial with no rule; no default is a leading empty option plus Required.",
					"The picker reads the theme's Input base, which is what makes it match the fields around it.",
					"The Go style owns the frame on all four targets, which is why the web's own <select> border is reset away.",
					"The value is controlled; whether the menu is open is the renderer's, and Go never hears about it.",
					"Group sections consecutive options; Disabled greys one out without removing it.",
				),
			)
		},
	}
}

// --- 5.7 -----------------------------------------------------------------

func lessonPINInput() Lesson {
	return Lesson{
		Title:   "One-time codes: six boxes, one field",
		Summary: "comps.PINInput: a row of boxes drawn over one field that holds the whole code, and why it stopped being six fields.",
		Body: func(ctx *core.Context) core.View {
			// The code, and what the demo has been told about it. Both are
			// lesson state: the widget holds neither, exactly as every
			// controlled input in this chapter holds neither.
			code := core.NewState(ctx, "")
			submitted := core.NewState(ctx, "")
			attempts := core.NewState(ctx, 0)

			// "1 time" rather than "1 times": the caption is read after every
			// keystroke, so the one place the demo counts out loud should not
			// read as a placeholder someone forgot to finish.
			fired := "times"
			if attempts.Get() == 1 {
				fired = "time"
			}

			return core.Column(
				core.Gap(14),
				prose("A one-time code looks like six boxes, one character each. comps.PINInput "+
					"draws the boxes, and under them keeps one text field holding the whole "+
					"code. A tap anywhere on the row puts the caret in that field (core.Focus), "+
					"and every key, paste, backspace and SMS autofill is an ordinary edit of one "+
					"string; the boxes redraw from it."),
				codeBlock(`comps.PINInput{
    Length:     6,
    Value:      code.Get(),
    OnChange:   code.Set,
    OnComplete: func(c string) { verify(c) },
}`),
				demoPanel("Type a code. Paste a whole one. Backspace from anywhere. Watch the next box light while the field has focus.",
					comps.PINInput{
						Length:     6,
						Value:      code.Get(),
						Label:      "One-time code",
						OnChange:   code.Set,
						OnComplete: func(c string) { attempts.Set(attempts.Get() + 1) },
						Style:      []core.StyleProp{core.MaxWidth("320px")},
					},
					caption(fmt.Sprintf("Value = %q   ·   OnComplete fired %d %s",
						code.Get(), attempts.Get(), fired)),
					core.Row(
						core.Gap(8),
						comps.Button{
							Label:    "Submit",
							Disabled: len(code.Get()) < 6,
							OnTap:    func() { submitted.Set(code.Get()) },
						},
						comps.Button{
							Label:    "Clear",
							Emphasis: comps.EmphasisGhost,
							OnTap:    func() { code.Set(""); submitted.Set("") },
						},
					),
					core.IfElse(submitted.Get() == "",
						caption("Nothing submitted yet."),
						caption("✓ Submitted "+submitted.Get()),
					),
				),
				prose("It used to be six fields, one per box, with the widget moving the caret "+
					"from each to the next as a character landed. On the Android emulator, with "+
					"keys about 130 ms apart, that lost digits: a key typed while the caret was "+
					"on its way from one box to the next reached no field at all, and a box that "+
					"had already been left could not replay a key onto Go's rewrite of it. No "+
					"bookkeeping fixes a key that never arrived. One field has no caret moves "+
					"to lose keys in, and its code is one value under the same text-edit "+
					"protocol every search box uses."),
				prose("The rules fall out of the field. Value is its text, capped at Length, so "+
					"a longer paste keeps what fits. Backspace deletes the last character "+
					"wherever you tapped. OnComplete fires on every edit that leaves the code "+
					"full, a correction to a full code included, because OnComplete means "+
					"\"submit this\" and a corrected code that stays silent is a field that will "+
					"not submit. It fires from the change handler, so a screen restored with a "+
					"complete code does not resubmit itself on sight."),
				prose("The field is one point square with no frame, fill or ink, a ZStack layer "+
					"over the boxes: present and focusable, not seen. It is also what a screen "+
					"reader meets, named \"One-time code, 2 of 6 entered\"; the boxes are hidden "+
					"from it, being a picture of what the field holds. It asks for the number "+
					"pad with core.Keyboard(core.KeyboardDigits), which on iOS also marks it as "+
					"a one-time code field, so the code from a text message is offered above "+
					"the keyboard."),
				keyPoints(
					"PINInput is boxes drawn over one field that holds the whole code; a tap on the row focuses it.",
					"One field means no caret moves between boxes, which is where keys typed quickly were lost.",
					"Value is the field's text capped at Length; backspace, paste and autofill are the field's own.",
					"OnComplete fires whenever an edit leaves the code full, so a corrected code submits again.",
					"It holds hooks (the field's ref and its focus), so render it in a stable position every pass.",
				),
			)
		},
	}
}

// --- 5.8 -----------------------------------------------------------------

// 5.8 — G2 of the third low-hanging-fruit round. The lesson is organised
// around who holds what: the set is the form's, the half-typed tag is the
// widget's, and the reason is the same test DateRangePicker's pending start
// passed.
func lessonTagInput() Lesson {
	return Lesson{
		Title:   "Tags: a set typed one at a time",
		Summary: "comps.TagInput: return or a comma commits, each tag has its own ✕, and the half-typed tag is the widget's.",
		Body: func(ctx *core.Context) core.View {
			// The set is lesson state, as every value in this chapter is. The
			// draft is not here: the widget holds it.
			tags := core.NewState(ctx, []string{"design", "urgent"})

			return core.Column(
				core.Gap(14),
				prose("A tag field collects a set of short strings — recipients, labels, "+
					"interests — one at a time. comps.TagInput draws the set as a wrapping list "+
					"of pills above an input, and commits whatever is typed when the reader "+
					"presses return or types a comma."),
				codeBlock(`comps.FormField{
    Label: "Labels",
    Input: comps.TagInput{
        Tags:        tags.Get(),
        OnChange:    tags.Set,
        Label:       "Labels",
        Placeholder: "Add a label",
        Max:         6,
    },
}`),
				demoPanel("Type a label and press return. Paste \"a, b, c\". Remove one with its ✕.",
					comps.FormField{
						Label: "Labels",
						Hint:  "Up to six. Return or a comma adds one.",
						Input: comps.TagInput{
							Tags:        tags.Get(),
							OnChange:    tags.Set,
							Label:       "Labels",
							Placeholder: "Add a label",
							Max:         6,
						},
					},
					caption(fmt.Sprintf("Tags = %q", tags.Get())),
				),
				prose("The set is yours and the half-typed tag is the widget's. A form submits the "+
					"set, and \"desi\" is not a member of it, so no application wants the draft — "+
					"the same test DateRangePicker's pending start passed. Holding it makes "+
					"TagInput hook-owning, so render it unconditionally, like PasswordField."),
				prose("Each tag's ✕ is its own button, and the tag's text is not one. A ✕ inside a "+
					"Chip would be a button inside a button, which no accessibility tree can "+
					"represent; a chip that removed itself when tapped would bind its largest "+
					"surface to a destructive action. A reader hears the tag, then \"Remove "+
					"design, button\"."),
				prose("A paste is the same rule as a comma: everything before the last separator "+
					"is committed, and what follows it stays in the input. Pieces are trimmed, and "+
					"empties and duplicates are dropped. There is no \"backspace in an empty "+
					"field removes the last tag\" — the field reports its text, not its keys, "+
					"which is the wall 5.7 met."),
				keyPoints(
					"TagInput: tags as a wrapping list of pills above an input; return or a separator commits.",
					"The set is the caller's; the draft is the widget's, so it holds a hook.",
					"Each tag is inert text plus a ✕ named \"Remove …\" — never a button inside a button.",
					"Pastes split on the separators; trimmed, no empties, no duplicates, no more than Max.",
				),
			)
		},
	}
}

// --- 5.9 -----------------------------------------------------------------

// 5.9 — Phase 2 of the fourth low-hanging-fruit round (K1 to K4). Four inputs
// in one lesson, held together by the question every widget in this chapter
// answers: who holds the value. Three of the four hold nothing; the one that
// does (the swatch picker's half-typed hex) holds it for TagInput's reason.
func lessonInputFamily() Lesson {
	return Lesson{
		Title:   "Keypads, swatches, ranges and masks",
		Summary: "comps.NumberPad, ColorSwatchPicker, RangeSlider and MaskedInput: four inputs the system keyboard and a plain field do not cover.",
		Body: func(ctx *core.Context) core.View {
			t := ctx.Theme()

			// Every value below is lesson state. NumberPad, RangeSlider and
			// MaskedInput take no hooks at all; ColorSwatchPicker takes one,
			// for the hex it has not finished reading.
			pin := core.NewState(ctx, "")
			colour := core.NewState(ctx, "#2A78D6")
			low := core.NewState(ctx, 20.0)
			high := core.NewState(ctx, 80.0)
			phone := core.NewState(ctx, "")
			card := core.NewState(ctx, "")

			const pinLength = 4
			unlocked := len(pin.Get()) == pinLength

			// The lock screen's dots: display only. They are PINInput's boxes
			// without the field, because here no field exists; the pad is the
			// whole input.
			dots := []core.PropsAndChildren{
				core.Padding(0), core.Gap(12),
				core.Justify(core.JustifyCenter),
				core.AccessibilityRole(core.RoleStatus),
				core.AccessibilityLabel(fmt.Sprintf("Passcode, %d of %d entered", len(pin.Get()), pinLength)),
			}
			for i := 0; i < pinLength; i++ {
				fill := comps.ColorTransparent
				if i < len(pin.Get()) {
					fill = t.Colors.TextPrimary
				}
				dots = append(dots, core.Box(
					core.Width("14px"), core.Height("14px"), core.Padding(0),
					core.BorderRadius(7), core.BorderWidth(2),
					core.BorderColor(t.Colors.TextPrimary),
					core.BackgroundColor(fill),
					core.AccessibilityHidden(),
				))
			}

			return core.Column(
				core.Gap(14),
				prose("Four inputs a form reaches for after the text field, the picker and the "+
					"switch. They share this chapter's rule: the value is yours. Three of them "+
					"hold nothing at all, so they take no hooks and may be rendered inside a "+
					"core.If; the fourth holds only what no application wants."),

				prose("comps.NumberPad is twelve keys and no value. It reports a key, and what a "+
					"key does to the value is a line of your Go: append and cap at four for a "+
					"passcode, allow one \".\" for an amount. A field that wants digits should ask "+
					"the platform for its own pad with core.Keyboard(core.KeyboardDigits), as "+
					"5.7 does. This is for where that pad cannot go: a lock screen with no field "+
					"to focus, a kiosk where the system keyboard must never rise, a static "+
					"export where an inputmode hint does nothing."),
				codeBlock(`comps.NumberPad{
    OnKey: func(k string) {
        if len(pin.Get()) < 4 {
            pin.Set(pin.Get() + k)
        }
    },
    OnBackspace: func() { pin.Set(dropLast(pin.Get())) },
}`),
				demoPanel("A lock screen: dots over a pad, and no text field anywhere. Each key gives a light haptic on a phone.",
					core.Column(
						core.Padding(0), core.Gap(16),
						core.AlignItemsProp(core.AlignItemsCenter),
						core.Row(dots...),
						comps.NumberPad{
							Disabled: unlocked,
							OnKey: func(k string) {
								if len(pin.Get()) < pinLength {
									pin.Set(pin.Get() + k)
								}
							},
							OnBackspace: func() {
								if r := []rune(pin.Get()); len(r) > 0 {
									pin.Set(string(r[:len(r)-1]))
								}
							},
							Style: []core.StyleProp{core.MaxWidth("300px")},
						},
					),
					core.IfElse(unlocked,
						caption("✓ Four digits entered; the pad is Disabled."),
						caption(fmt.Sprintf("Passcode = %q", pin.Get())),
					),
					comps.Button{
						Label:    "Clear",
						Emphasis: comps.EmphasisGhost,
						OnTap:    func() { pin.Set("") },
					},
				),

				prose("comps.ColorSwatchPicker is a radiogroup of colours. With no Colors it "+
					"offers the theme's chart colours, which are already chosen to be told "+
					"apart. The choice is a ring and a check, never the colour alone, and the "+
					"check's ink is black or white by contrast with its swatch. Give your own "+
					"swatches a Name: it is what a screen reader says, and \"Brand blue\" means "+
					"more than a hue guessed from the hex."),
				codeBlock(`comps.ColorSwatchPicker{
    Label:       "Label colour",
    Value:       colour.Get(),
    OnChange:    colour.Set,
    AllowCustom: true,
}`),
				demoPanel("Pick a swatch. Then type a hex such as #7B2FF0 into the field: it commits when it is a whole colour.",
					comps.ColorSwatchPicker{
						Label:       "Label colour",
						Value:       colour.Get(),
						OnChange:    colour.Set,
						AllowCustom: true,
					},
					caption("Value = "+colour.Get()),
				),

				prose("comps.RangeSlider is a minimum and a maximum that cannot cross. It is two "+
					"sliders and says so: one track with two thumbs is a control no target here "+
					"has, and two labelled sliders are what VoiceOver and TalkBack are given for "+
					"a range in any case. Drag the minimum past the maximum and it carries the "+
					"maximum along, so a range can be moved as a whole from either end."),
				codeBlock(`comps.RangeSlider{
    Title: "Price",
    Min:   0, Max: 200, Step: 5,
    Low:   low.Get(),
    High:  high.Get(),
    OnChange: func(l, h float64) {
        low.Set(l)
        high.Set(h)
    },
    Format: dollars,
}`),
				demoPanel("Drag Minimum past Maximum and let go: both report, and the pair stays ordered.",
					comps.RangeSlider{
						Title: "Price", Min: 0, Max: 200, Step: 5,
						Low: low.Get(), High: high.Get(),
						OnChange: func(l, h float64) { low.Set(l); high.Set(h) },
						Format:   func(v float64) string { return fmt.Sprintf("$%.0f", v) },
					},
					caption(fmt.Sprintf("Low = %.0f   ·   High = %.0f", low.Get(), high.Get())),
				),

				prose("comps.MaskedInput formats as you type. The mask is written in three slot "+
					"characters (# a digit, A a letter, * either) and everything else is a "+
					"literal. Value is the raw value, 5551234567, which is the form you store "+
					"and send; OnChange hands over the drawn text as well. A literal is written "+
					"only once a character follows it, so the text never ends in one and "+
					"backspace always removes something you typed."),
				codeBlock(`comps.MaskedInput{
    Mask:     "(###) ###-####",
    Value:    phone.Get(),
    OnChange: func(raw, _ string) {
        phone.Set(raw)
    },
    Keyboard: core.KeyboardDigits,
    Label:    "Phone",
}`),
				demoPanel("Type ten digits, fast. Backspace through the brackets. Try a letter: the mask refuses it.",
					comps.FormField{
						Label: "Phone",
						Input: comps.MaskedInput{
							Mask:     "(###) ###-####",
							Value:    phone.Get(),
							OnChange: func(raw, _ string) { phone.Set(raw) },
							Keyboard: core.KeyboardDigits,
							Label:    "Phone",
						},
					},
					comps.FormField{
						Label: "Card number",
						Input: comps.MaskedInput{
							Mask:     "#### #### #### ####",
							Value:    card.Get(),
							OnChange: func(raw, _ string) { card.Set(raw) },
							Keyboard: core.KeyboardDigits,
							Label:    "Card number",
						},
					},
					caption(fmt.Sprintf("phone = %q   ·   card = %q", phone.Get(), card.Get())),
				),
				prose("Every formatted keystroke is Go answering with text the host did not send, "+
					"which the text-edit protocol calls a rewrite; TagInput makes one per tag "+
					"and this makes one per key. Typed at machine speed on the Android emulator "+
					"it loses nothing, because each host replays the keys in flight onto the "+
					"rewrite. What it needed was the caret to travel with its text: a mask puts "+
					"a bracket before the caret, and a host that kept the caret's raw offset "+
					"typed 1 2 3 4 5 6 as (234) 651. One limit remains. A key typed mid-text at "+
					"the very end of a group lands correctly and leaves the caret after the "+
					"reflowed digits, since no host tells Go where its caret is."),
				keyPoints(
					"NumberPad reports keys and holds no value: the rule for a key is a line of Go in OnKey. Use the system pad (core.Keyboard) when there is a field.",
					"ColorSwatchPicker is a radiogroup; selected is a ring and a check, and a swatch's Name is what is spoken.",
					"RangeSlider is two labelled sliders whose thumbs push each other, so OnChange always reports an ordered pair.",
					"MaskedInput's Value is the raw value; # A * are slots, anything else is a literal written only once a character follows it.",
					"Only ColorSwatchPicker holds a hook (the half-typed hex), so it alone must render unconditionally.",
				),
			)
		},
	}
}
