package forms_test

import (
	"fmt"
	"regexp"
	"sync"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/forms"
)

// render drives one render pass and returns the Form that pass produced.
// Every test goes through this rather than calling UseForm once and holding
// the handle, because a fresh Form per pass — carrying that pass's Spec over
// the same record — is the thing the design depends on.
func render(ctx *core.Context, spec forms.Spec) *forms.Form {
	ctx.Reset()
	ctx.BeginRenderPass()
	return forms.UseForm(ctx, spec)
}

func specOf(fields ...forms.Field) forms.Spec { return forms.Spec{Fields: fields} }

func TestInitialSeedsOnceAndDoesNotResurrect(t *testing.T) {
	ctx := core.NewContext()
	spec := specOf(forms.Field{Name: "plan", Initial: "monthly"})

	form := render(ctx, spec)
	if got := form.Value("plan"); got != "monthly" {
		t.Fatalf("mount value = %q, want the Initial", got)
	}

	// The user clears the field. The spec still names a default, so a hook
	// that re-seeded every pass would type it back in for them.
	form.SetValue("plan", "")
	form = render(ctx, spec)
	if got := form.Value("plan"); got != "" {
		t.Errorf("after clearing, value = %q; Initial must not be re-applied", got)
	}
}

func TestFieldAddedLaterGetsItsInitial(t *testing.T) {
	// A conditional section, or a row added to a repeating group: the name is
	// new on this pass, so it has never been seeded and its Initial applies.
	ctx := core.NewContext()
	render(ctx, specOf(forms.Field{Name: "a", Initial: "1"}))

	form := render(ctx, specOf(
		forms.Field{Name: "a", Initial: "1"},
		forms.Field{Name: "b", Initial: "2"},
	))
	if got := form.Value("b"); got != "2" {
		t.Errorf("late field value = %q, want %q", got, "2")
	}
}

func TestEveryDeclaredFieldIsPresentInValues(t *testing.T) {
	// Including the ones with no Initial: a submit handler ranging over the
	// map should see the form's shape, not only what the user filled in.
	ctx := core.NewContext()
	form := render(ctx, specOf(
		forms.Field{Name: "email"},
		forms.Field{Name: "phone", Initial: "+1"},
	))
	v := form.Values()
	if len(v) != 2 {
		t.Fatalf("Values() = %v, want both declared fields", v)
	}
	if _, ok := v["email"]; !ok {
		t.Error("a field with no Initial must still be present, as an empty string")
	}
}

func TestValuesIsACopy(t *testing.T) {
	ctx := core.NewContext()
	form := render(ctx, specOf(forms.Field{Name: "a", Initial: "1"}))

	snapshot := form.Values()
	snapshot["a"] = "tampered"
	if got := form.Value("a"); got != "1" {
		t.Errorf("mutating the returned map reached the live values: %q", got)
	}
}

func TestFirstFailingRuleWins(t *testing.T) {
	// A field shows one line of feedback, so rule order is the choice of
	// which complaint is the useful one. Required first means an empty field
	// says "Required" rather than "Must be at least 8 characters".
	ctx := core.NewContext()
	form := render(ctx, forms.Spec{
		Reveal: forms.RevealAlways,
		Fields: []forms.Field{{Name: "password", Rules: []forms.Rule{
			forms.Required("Required"),
			forms.MinLen(8, "Too short"),
		}}},
	})
	if got := form.Error("password"); got != "Required" {
		t.Errorf("empty field error = %q, want the first rule's message", got)
	}

	// Two rules that *both* fail on the same value — the case that actually
	// distinguishes "first wins" from "last wins". An empty field cannot: the
	// only rule with an opinion about "" is Required.
	form.SetValue("password", "abc")
	form = render(ctx, forms.Spec{
		Reveal: forms.RevealAlways,
		Fields: []forms.Field{{Name: "password", Rules: []forms.Rule{
			forms.Required("Required"),
			forms.MinLen(8, "Too short"),
			forms.Pattern(regexp.MustCompile(`^[0-9]+$`), "Digits only"),
		}}},
	})
	if got := form.Error("password"); got != "Too short" {
		t.Errorf("short field error = %q, want the first *failing* rule's message", got)
	}
}

// A nil entry in Rules is skipped rather than dereferenced — a slice built
// with an append that fell through a branch is a real shape, and panicking on
// the render goroutine is a poor way to report it.
func TestNilRuleIsSkipped(t *testing.T) {
	ctx := core.NewContext()
	form := render(ctx, forms.Spec{
		Reveal: forms.RevealAlways,
		Fields: []forms.Field{{Name: "a", Rules: []forms.Rule{nil, forms.Required("Required")}}},
	})
	if got := form.Error("a"); got != "Required" {
		t.Errorf("error = %q, want the non-nil rule to still run", got)
	}
}

func TestCrossFieldValidateFillsOnlyUnclaimedFields(t *testing.T) {
	// Field rules are the more specific complaint. An empty confirmation
	// needs "Required", not "The two passwords differ" — which is true,
	// unhelpful, and what a last-writer-wins merge would show.
	spec := func() forms.Spec {
		return forms.Spec{
			Reveal: forms.RevealAlways,
			Fields: []forms.Field{
				{Name: "password", Initial: "hunter22"},
				{Name: "confirm", Rules: []forms.Rule{forms.Required("Required")}},
			},
			Validate: func(v forms.Values) map[string]string {
				if v["confirm"] != v["password"] {
					return map[string]string{"confirm": "The two passwords differ"}
				}
				return nil
			},
		}
	}

	ctx := core.NewContext()
	form := render(ctx, spec())
	if got := form.Error("confirm"); got != "Required" {
		t.Errorf("empty confirm = %q, want the field rule to win", got)
	}

	form.SetValue("confirm", "hunter11")
	form = render(ctx, spec())
	if got := form.Error("confirm"); got != "The two passwords differ" {
		t.Errorf("mismatched confirm = %q, want the cross-field message", got)
	}

	form.SetValue("confirm", "hunter22")
	form = render(ctx, spec())
	if got := form.Error("confirm"); got != "" {
		t.Errorf("matching confirm = %q, want no error", got)
	}
}

func TestValidateMayNameAFieldNoInputRenders(t *testing.T) {
	// A form-level error: a key nothing is bound to, read by a banner.
	ctx := core.NewContext()
	form := render(ctx, forms.Spec{
		Reveal:   forms.RevealAlways,
		Fields:   []forms.Field{{Name: "amount", Initial: "500"}},
		Validate: func(forms.Values) map[string]string { return map[string]string{"form": "Not enough balance"} },
	})
	if got := form.Error("form"); got != "Not enough balance" {
		t.Errorf("form-level error = %q", got)
	}
	if form.Valid() {
		t.Error("a form-level error must make the form invalid")
	}
}

// Empty messages from Validate are dropped rather than stored as blank
// errors, so a Validate that builds its map unconditionally cannot make a
// clean form invalid.
func TestValidateEmptyMessagesAreIgnored(t *testing.T) {
	ctx := core.NewContext()
	form := render(ctx, forms.Spec{
		Reveal:   forms.RevealAlways,
		Fields:   []forms.Field{{Name: "a", Initial: "x"}},
		Validate: func(forms.Values) map[string]string { return map[string]string{"a": ""} },
	})
	if !form.Valid() {
		t.Error("an empty message is not an error")
	}
	if got := form.Error("a"); got != "" {
		t.Errorf("Error = %q, want none", got)
	}
}

func TestRevealOnSubmitHidesUntilTheFirstSubmit(t *testing.T) {
	spec := func() forms.Spec {
		return forms.Spec{Fields: []forms.Field{
			{Name: "email", Rules: []forms.Rule{forms.Required("Required")}},
		}}
	}
	ctx := core.NewContext()
	form := render(ctx, spec())

	// Invalid from the first render — but silent, because the user has not
	// claimed to be done. This is the whole point of the default policy.
	if form.Valid() {
		t.Fatal("an empty required field is not valid")
	}
	if got := form.Error("email"); got != "" {
		t.Errorf("pre-submit error = %q, want silence", got)
	}

	// Typing does not reveal it either, under this policy.
	form.SetValue("email", "x")
	form = render(ctx, spec())
	if got := form.Error("email"); got != "" {
		t.Errorf("post-edit error = %q, want silence until submit", got)
	}
	form.SetValue("email", "")

	form = render(ctx, spec())
	if form.Submit(nil) {
		t.Fatal("Submit reported success on an invalid form")
	}
	if got := form.Error("email"); got != "Required" {
		t.Errorf("post-submit error = %q, want it revealed", got)
	}

	// And it stays live: the fix is confirmed the instant it lands, without
	// another submit.
	form.SetValue("email", "you@example.com")
	form = render(ctx, spec())
	if got := form.Error("email"); got != "" {
		t.Errorf("error after the fix = %q, want it cleared", got)
	}
}

func TestRevealOnTouchShowsPerFieldThenEverythingOnSubmit(t *testing.T) {
	spec := func() forms.Spec {
		return forms.Spec{
			Reveal: forms.RevealOnTouch,
			Fields: []forms.Field{
				{Name: "card", Rules: []forms.Rule{forms.Required("Required")}},
				{Name: "cvv", Rules: []forms.Rule{forms.Required("Required")}},
			},
		}
	}
	ctx := core.NewContext()
	form := render(ctx, spec())
	if form.Error("card") != "" || form.Error("cvv") != "" {
		t.Fatal("untouched fields must be silent")
	}

	// Touching a field and leaving it empty reveals only that field.
	form.SetValue("card", "4")
	form.SetValue("card", "")
	form = render(ctx, spec())
	if got := form.Error("card"); got != "Required" {
		t.Errorf("touched field error = %q, want it revealed", got)
	}
	if got := form.Error("cvv"); got != "" {
		t.Errorf("untouched sibling error = %q, want silence", got)
	}

	// A submit reveals the rest. The alternative — "on touch" continuing to
	// hide an untouched field — is a form that refuses to submit and shows
	// no reason why.
	form.Submit(nil)
	form = render(ctx, spec())
	if got := form.Error("cvv"); got != "Required" {
		t.Errorf("untouched field after submit = %q, want it revealed", got)
	}
}

func TestRevealAlwaysShowsFromTheFirstRender(t *testing.T) {
	ctx := core.NewContext()
	form := render(ctx, forms.Spec{
		Reveal: forms.RevealAlways,
		Fields: []forms.Field{{Name: "a", Rules: []forms.Rule{forms.Required("Required")}}},
	})
	if got := form.Error("a"); got != "Required" {
		t.Errorf("error = %q, want it visible with nothing touched or submitted", got)
	}
	if form.Touched("a") || form.Submitted() {
		t.Error("RevealAlways must not fake touched/submitted to get its visibility")
	}
}

func TestSubmitPassesACopyOfTheValues(t *testing.T) {
	ctx := core.NewContext()
	spec := specOf(forms.Field{Name: "email", Initial: "you@example.com"})
	form := render(ctx, spec)

	var got forms.Values
	if !form.Submit(func(v forms.Values) { got = v }) {
		t.Fatal("a form with no rules must submit")
	}
	if got["email"] != "you@example.com" {
		t.Fatalf("handler saw %v", got)
	}

	// The handler is free to keep the map — the usual shape is a goroutine
	// that outlives the pass — so later typing must not reach it.
	form.SetValue("email", "someone@else.com")
	if got["email"] != "you@example.com" {
		t.Error("the handler's map aliased the live values")
	}
}

func TestSubmitOnAnInvalidFormSkipsTheHandler(t *testing.T) {
	ctx := core.NewContext()
	form := render(ctx, specOf(forms.Field{Name: "a", Rules: []forms.Rule{forms.Required("Required")}}))

	called := false
	if form.Submit(func(forms.Values) { called = true }) {
		t.Error("Submit reported success on an invalid form")
	}
	if called {
		t.Error("the handler ran on an invalid form")
	}
	if !form.Submitted() {
		t.Error("a failed submit still counts as an attempt; it is what turns the errors on")
	}
}

// Submitting an invalid form changes what every field displays without moving
// a single value, and nothing on that path writes a hook slot — so Submit has
// to ask for the render itself or the newly revealed errors never paint.
func TestSubmitRequestsARenderEvenWhenNoValueChanges(t *testing.T) {
	ctx := core.NewContext()
	form := render(ctx, specOf(forms.Field{Name: "a", Rules: []forms.Rule{forms.Required("Required")}}))

	ctx.ClearDirty()
	form.Submit(nil)
	if !ctx.IsDirty() {
		t.Error("a failed submit left the tree clean; the revealed errors would never paint")
	}
}

func TestSetValueRequestsARender(t *testing.T) {
	ctx := core.NewContext()
	form := render(ctx, specOf(forms.Field{Name: "a"}))

	ctx.ClearDirty()
	form.SetValue("a", "x")
	if !ctx.IsDirty() {
		t.Error("the field would not repaint as the user types")
	}
}

func TestOnSubmitAndOnChangeAdaptToTheCallbackShapes(t *testing.T) {
	ctx := core.NewContext()
	form := render(ctx, specOf(forms.Field{Name: "a"}))

	// OnChange is what a text callback hands to; OnToggle is the bool one.
	form.OnChange("a")("typed")
	if got := form.Value("a"); got != "typed" {
		t.Errorf("OnChange wrote %q", got)
	}
	form.OnToggle("a")(true)
	if !form.Checked("a") {
		t.Errorf("OnToggle wrote %q, which does not read back as checked", form.Value("a"))
	}
	form.OnToggle("a")(false)
	if form.Checked("a") {
		t.Error("unticking must read back as unchecked")
	}

	// OnSubmit is the void shape a Button.OnTap takes.
	ran := false
	form.OnSubmit(func(forms.Values) { ran = true })()
	if !ran {
		t.Error("OnSubmit did not run the handler")
	}
}

func TestExternalErrorsAreShownRegardlessOfPolicyAndClearedOnChange(t *testing.T) {
	// The default policy, so anything visible here is visible *because* it is
	// external, not because a submit revealed it.
	spec := func() forms.Spec {
		return specOf(forms.Field{Name: "email", Initial: "you@example.com"})
	}
	ctx := core.NewContext()
	form := render(ctx, spec())

	form.SetErrors(map[string]string{"email": "That address is already registered"})
	form = render(ctx, spec())
	if got := form.Error("email"); got != "That address is already registered" {
		t.Errorf("external error = %q, want it shown with no submit", got)
	}
	if form.Valid() {
		t.Error("an external error must make the form invalid")
	}

	// It disappears as the user starts fixing it, not after another round
	// trip: the server's verdict was about the old text.
	form.SetValue("email", "you2@example.com")
	form = render(ctx, spec())
	if got := form.Error("email"); got != "" {
		t.Errorf("error after editing = %q, want it dropped", got)
	}
	if !form.Valid() {
		t.Error("form should be valid again once the rejected value is gone")
	}
}

func TestExternalErrorOutranksADerivedOne(t *testing.T) {
	ctx := core.NewContext()
	spec := forms.Spec{
		Reveal: forms.RevealAlways,
		Fields: []forms.Field{{Name: "email", Rules: []forms.Rule{forms.Email("Not a valid email address")}}},
	}
	form := render(ctx, spec)
	form.SetValue("email", "nope")
	form.SetErrors(map[string]string{"email": "Rejected upstream"})

	form = render(ctx, spec)
	if got := form.Error("email"); got != "Rejected upstream" {
		t.Errorf("error = %q, want the newer external verdict", got)
	}
}

func TestSetErrorsReplacesWholesale(t *testing.T) {
	ctx := core.NewContext()
	spec := specOf(forms.Field{Name: "a"}, forms.Field{Name: "b"})
	form := render(ctx, spec)

	form.SetErrors(map[string]string{"a": "one", "b": "two"})
	form.SetErrors(map[string]string{"a": "three"})
	form = render(ctx, spec)
	if got := form.Error("a"); got != "three" {
		t.Errorf("a = %q, want the new message", got)
	}
	if got := form.Error("b"); got != "" {
		t.Errorf("b = %q, want the previous set to have been replaced, not merged", got)
	}

	// nil is how a retry clears them before it starts.
	form.SetErrors(nil)
	if got := form.Error("a"); got != "" {
		t.Errorf("a = %q after SetErrors(nil), want cleared", got)
	}
	// An empty message is dropped rather than stored as a blank error, which
	// would otherwise be an invisible reason the form will not submit.
	form.SetErrors(map[string]string{"a": ""})
	if !form.Valid() {
		t.Error("a blank message must not count as an error")
	}
}

func TestErrorsReportsEveryVisibleMessage(t *testing.T) {
	ctx := core.NewContext()
	spec := forms.Spec{Fields: []forms.Field{
		{Name: "a", Rules: []forms.Rule{forms.Required("Required")}},
		{Name: "b", Rules: []forms.Rule{forms.Required("Required")}},
		{Name: "c"},
	}}
	form := render(ctx, spec)

	if len(form.Errors()) != 0 {
		t.Fatalf("pre-submit Errors() = %v, want empty under the default policy", form.Errors())
	}
	form.Submit(nil)
	form.SetErrors(map[string]string{"c": "from the server"})

	form = render(ctx, spec)
	got := form.Errors()
	if len(got) != 3 || got["a"] != "Required" || got["b"] != "Required" || got["c"] != "from the server" {
		t.Errorf("Errors() = %v, want both revealed rules and the external one", got)
	}
}

func TestResetReturnsToTheDeclaredStartingState(t *testing.T) {
	ctx := core.NewContext()
	spec := forms.Spec{Fields: []forms.Field{
		{Name: "note", Initial: "draft"},
		{Name: "email", Rules: []forms.Rule{forms.Required("Required")}},
	}}
	form := render(ctx, spec)

	form.SetValue("note", "edited")
	form.SetErrors(map[string]string{"email": "upstream"})
	form.Submit(nil)

	form = render(ctx, spec)
	form.Reset()
	form = render(ctx, spec)

	if got := form.Value("note"); got != "draft" {
		t.Errorf("value = %q, want the Initial back", got)
	}
	if form.Touched("note") {
		t.Error("Reset must clear touched")
	}
	if form.Submitted() {
		t.Error("Reset must clear submitted, or the fresh form opens shouting")
	}
	if got := form.Error("email"); got != "" {
		t.Errorf("error after reset = %q, want silence", got)
	}
}

func TestResetReReadsInitialsFromTheCurrentSpec(t *testing.T) {
	// This is how a form is populated from data that arrives late: render the
	// loaded values as Initial and reset once they land.
	ctx := core.NewContext()
	render(ctx, specOf(forms.Field{Name: "email", Initial: ""}))

	loaded := specOf(forms.Field{Name: "email", Initial: "loaded@example.com"})
	form := render(ctx, loaded)
	// Seeding alone will not do it — the name has been seen before.
	if got := form.Value("email"); got != "" {
		t.Fatalf("value = %q; a seeded field must not be re-seeded", got)
	}
	form.Reset()
	if got := form.Value("email"); got != "loaded@example.com" {
		t.Errorf("value after reset = %q, want the current spec's Initial", got)
	}
}

func TestResetDropsFieldsTheCurrentSpecNoLongerDeclares(t *testing.T) {
	ctx := core.NewContext()
	render(ctx, specOf(forms.Field{Name: "a"}, forms.Field{Name: "gone", Initial: "x"}))

	form := render(ctx, specOf(forms.Field{Name: "a"}))
	form.Reset()
	if v := form.Values(); len(v) != 1 {
		t.Errorf("Values() = %v, want exactly the current declaration", v)
	}
}

// The rules are re-read from the Spec every pass, so a rule may close over
// state that changes between renders and take effect with no re-registration.
// Nothing about the checking is stored.
func TestRulesComeFromTheCurrentPassNotTheMountingOne(t *testing.T) {
	ctx := core.NewContext()
	limit := 3
	spec := func() forms.Spec {
		return forms.Spec{
			Reveal: forms.RevealAlways,
			Fields: []forms.Field{{Name: "code", Initial: "abcd", Rules: []forms.Rule{
				forms.MaxLen(limit, fmt.Sprintf("At most %d", limit)),
			}}},
		}
	}

	form := render(ctx, spec())
	if got := form.Error("code"); got != "At most 3" {
		t.Fatalf("error = %q", got)
	}

	limit = 8
	form = render(ctx, spec())
	if got := form.Error("code"); got != "" {
		t.Errorf("error = %q, want the relaxed rule from this pass to apply", got)
	}
}

// A rule may read the form it belongs to. Rules run with the record's lock
// released precisely so this cannot deadlock — a form whose validation hangs
// the render goroutine is worse than one that validates loosely.
func TestARuleMayReadTheFormWithoutDeadlocking(t *testing.T) {
	ctx := core.NewContext()
	var form *forms.Form
	spec := forms.Spec{
		Reveal: forms.RevealAlways,
		Fields: []forms.Field{
			{Name: "a", Initial: "x"},
			{Name: "b", Initial: "y", Rules: []forms.Rule{func(v string) string {
				if form != nil && v == form.Value("a") {
					return "must differ from a"
				}
				return ""
			}}},
		},
	}
	form = render(ctx, spec)

	if got := form.Error("b"); got != "" {
		t.Fatalf("error = %q, want none while the values differ", got)
	}
	form.SetValue("b", "x")
	form = render(ctx, spec)
	if got := form.Error("b"); got != "must differ from a" {
		t.Errorf("error = %q", got)
	}
}

// Sequencing under concurrency is why the record carries a mutex rather than
// living directly in the hook slot: every write here is a read-modify-write of
// a map, which core.State's whole-value Get/Set cannot make atomic. Run with
// -race.
func TestConcurrentWritesAndReadsAreSafe(t *testing.T) {
	ctx := core.NewContext()
	fields := make([]forms.Field, 8)
	for i := range fields {
		fields[i] = forms.Field{Name: fmt.Sprintf("f%d", i), Rules: []forms.Rule{forms.Required("Required")}}
	}
	form := render(ctx, forms.Spec{Fields: fields, Reveal: forms.RevealAlways})

	var wg sync.WaitGroup
	for i := range fields {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				form.SetValue(fmt.Sprintf("f%d", i), fmt.Sprintf("v%d", j))
			}
		}(i)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				form.Error(fmt.Sprintf("f%d", i))
				form.Valid()
				form.Values()
			}
		}(i)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < 50; j++ {
			form.SetErrors(map[string]string{"f0": "upstream"})
			form.Submit(nil)
		}
	}()
	wg.Wait()

	// Every write landed somewhere; the point is that none of them was lost
	// mid-map or read half-applied.
	for i := range fields {
		if got := form.Value(fmt.Sprintf("f%d", i)); got == "" {
			t.Errorf("f%d was left empty after 50 writes", i)
		}
	}
}

// Two forms in one component, and the same shape in a second app: four
// distinct slots. A form keyed on anything but slot position would let these
// overwrite each other's values.
func TestFormsOnDistinctSlotsAreIndependent(t *testing.T) {
	ctx := core.NewContext()
	pass := func() (*forms.Form, *forms.Form) {
		ctx.Reset()
		ctx.BeginRenderPass()
		return forms.UseForm(ctx, specOf(forms.Field{Name: "x", Initial: "a"})),
			forms.UseForm(ctx, specOf(forms.Field{Name: "x", Initial: "b"}))
	}

	first, second := pass()
	first.SetValue("x", "changed")
	first, second = pass()

	if got := first.Value("x"); got != "changed" {
		t.Errorf("first form value = %q", got)
	}
	if got := second.Value("x"); got != "b" {
		t.Errorf("second form value = %q, want its own Initial", got)
	}

	other := core.NewContext()
	otherForm := render(other, specOf(forms.Field{Name: "x", Initial: "c"}))
	if got := otherForm.Value("x"); got != "c" {
		t.Errorf("second app's form value = %q; contexts must be isolated", got)
	}
}

func TestRequiredIsDerivedFromTheRules(t *testing.T) {
	// The probe is "does any rule complain about an empty value", which is
	// what makes the marker on FormField and the rule that justifies it the
	// same fact rather than two claims that can drift.
	ctx := core.NewContext()
	form := render(ctx, specOf(
		forms.Field{Name: "email", Rules: []forms.Rule{forms.Required(""), forms.Email("")}},
		// Every rule but Required and Accepted is silent about "", so a field
		// carrying only those is optional however many it has.
		forms.Field{Name: "nickname", Rules: []forms.Rule{forms.MinLen(3, ""), forms.MaxLen(20, "")}},
		forms.Field{Name: "bio"},
		// An unticked checkbox is "false", not "", so Accepted speaks about
		// the empty value too — and a must-tick box is required.
		forms.Field{Name: "terms", Rules: []forms.Rule{forms.Accepted("")}},
		// An app's own closure counts for exactly the same reason: nothing
		// here recognises rule identities, only behaviour.
		forms.Field{Name: "code", Rules: []forms.Rule{func(v string) string {
			if v == "" {
				return "Enter the code we sent you"
			}
			return ""
		}}},
		// A nil rule is skipped, as everywhere else rules are run.
		forms.Field{Name: "spare", Rules: []forms.Rule{nil}},
		// Required-first is the advice, not a constraint: every rule is
		// probed, so a spec that orders them badly still gets its marker.
		forms.Field{Name: "billing", Rules: []forms.Rule{forms.Email(""), forms.Required("")}},
	))

	for name, want := range map[string]bool{
		"email":    true,
		"nickname": false,
		"bio":      false,
		"terms":    true,
		"code":     true,
		"spare":    false,
		"billing":  true,
		"unknown":  false,
	} {
		if got := form.Required(name); got != want {
			t.Errorf("Required(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestRequiredIgnoresValidate(t *testing.T) {
	// A cross-field check is not a property of the field — and the probe has
	// no other field's value to hand it — so Validate is not consulted, even
	// when it would reject the empty form.
	ctx := core.NewContext()
	form := render(ctx, forms.Spec{
		Fields: []forms.Field{{Name: "confirm"}},
		Validate: func(v forms.Values) map[string]string {
			return map[string]string{"confirm": "Always wrong"}
		},
	})
	if form.Required("confirm") {
		t.Error("Required must answer from the field's own rules only")
	}
}

func TestRequiredFollowsTheLiveSpec(t *testing.T) {
	// The spec is re-read every pass, so a rule that appears or disappears
	// with app state moves the marker with it and needs no bookkeeping.
	ctx := core.NewContext()
	optional := specOf(forms.Field{Name: "vat"})
	mandatory := specOf(forms.Field{Name: "vat", Rules: []forms.Rule{forms.Required("")}})

	if render(ctx, optional).Required("vat") {
		t.Fatal("field with no rules must not be required")
	}
	if !render(ctx, mandatory).Required("vat") {
		t.Fatal("a rule added on a later pass must be seen")
	}
	if render(ctx, optional).Required("vat") {
		t.Error("a rule removed on a later pass must stop being seen")
	}
}

func TestRequiredDoesNotDisturbTheForm(t *testing.T) {
	// The probe runs the rules against "" rather than against the value, so
	// it must not touch what the user typed, nor mark the field touched —
	// which under RevealOnTouch would reveal an error nobody asked for.
	ctx := core.NewContext()
	form := render(ctx, forms.Spec{
		Reveal: forms.RevealOnTouch,
		Fields: []forms.Field{{Name: "email", Initial: "bad", Rules: []forms.Rule{forms.Email("")}}},
	})

	form.Required("email")

	if got := form.Value("email"); got != "bad" {
		t.Errorf("value after the probe = %q, want it untouched", got)
	}
	if form.Touched("email") {
		t.Error("probing must not mark the field touched")
	}
	if got := form.Error("email"); got != "" {
		t.Errorf("probing must not reveal anything, got %q", got)
	}

	// And the answer is about emptiness, not about the value on screen: this
	// field's current text fails its rule, but nothing here demands content.
	if form.Required("email") {
		t.Error("a field whose current value is merely invalid is not required")
	}
}

// ---- RevealOnBlur ----

func blurSpec() forms.Spec {
	return forms.Spec{
		Reveal: forms.RevealOnBlur,
		Fields: []forms.Field{
			{Name: "email", Rules: []forms.Rule{forms.Required("Required")}},
			{Name: "name", Rules: []forms.Rule{forms.Required("Required")}},
		},
	}
}

func TestRevealOnBlurStaysSilentWhileTyping(t *testing.T) {
	// The whole point of the policy: an address is not yet an address at its
	// second character, and RevealOnTouch would already be complaining.
	ctx := core.NewContext()
	form := render(ctx, blurSpec())

	form.SetValue("email", "y")
	form = render(ctx, blurSpec())
	if got := form.Error("email"); got != "" {
		t.Fatalf("error while typing = %q, want silence until the field is left", got)
	}
	if !form.Touched("email") {
		t.Error("the field was edited; Touched must still say so")
	}
}

func TestRevealOnBlurShowsPerFieldThenEverythingOnSubmit(t *testing.T) {
	ctx := core.NewContext()
	form := render(ctx, blurSpec())
	if form.Error("email") != "" || form.Error("name") != "" {
		t.Fatal("nothing blurred yet; both fields must be silent")
	}

	form.MarkBlurred("email")
	form = render(ctx, blurSpec())
	if got := form.Error("email"); got != "Required" {
		t.Errorf("blurred field error = %q, want it revealed", got)
	}
	if got := form.Error("name"); got != "" {
		t.Errorf("un-blurred sibling error = %q, want silence", got)
	}

	// Cumulative with submit, for the same reason RevealOnTouch is: a form
	// that refuses to submit while showing no reason is the bug.
	form.Submit(nil)
	form = render(ctx, blurSpec())
	if got := form.Error("name"); got != "Required" {
		t.Errorf("never-blurred field after submit = %q, want it revealed", got)
	}
}

func TestRevealOnBlurStaysLiveAfterTheFix(t *testing.T) {
	// Reward early, punish late — but once punished, confirm the correction
	// the instant it lands rather than waiting for another blur.
	ctx := core.NewContext()
	form := render(ctx, blurSpec())
	form.MarkBlurred("email")
	form = render(ctx, blurSpec())
	if form.Error("email") == "" {
		t.Fatal("setup: the error should be revealed")
	}

	form.SetValue("email", "you@example.com")
	form = render(ctx, blurSpec())
	if got := form.Error("email"); got != "" {
		t.Errorf("error after the fix = %q, want it cleared without a second blur", got)
	}
}

func TestBlurDoesNotMarkTouchedAndTouchDoesNotMarkBlurred(t *testing.T) {
	// The two flags answer different questions, and folding either into the
	// other would make each policy fire on the other's occasions: a
	// tab-through would satisfy RevealOnTouch, and a keystroke would satisfy
	// RevealOnBlur.
	ctx := core.NewContext()
	form := render(ctx, blurSpec())

	form.MarkBlurred("email")
	if form.Touched("email") {
		t.Error("leaving a field must not count as editing it")
	}

	form.SetValue("name", "x")
	if form.Blurred("name") {
		t.Error("editing a field must not count as leaving it")
	}
}

func TestBlurDoesNotClearAnExternalError(t *testing.T) {
	// SetValue drops the server's verdict because the text it judged is gone.
	// A blur changes no text, so the verdict still stands.
	ctx := core.NewContext()
	form := render(ctx, blurSpec())
	form.SetErrors(map[string]string{"email": "Already registered"})

	form.MarkBlurred("email")
	form = render(ctx, blurSpec())
	if got := form.Error("email"); got != "Already registered" {
		t.Errorf("error after blur = %q, want the external one intact", got)
	}
}

func TestResetClearsBlurred(t *testing.T) {
	ctx := core.NewContext()
	form := render(ctx, blurSpec())
	form.MarkBlurred("email")

	form = render(ctx, blurSpec())
	form.Reset()
	form = render(ctx, blurSpec())

	if form.Blurred("email") {
		t.Error("Reset left the field marked blurred")
	}
	if got := form.Error("email"); got != "" {
		t.Errorf("error after Reset = %q, want a form back at its starting state", got)
	}
}

func TestBoundBuildersAttachBlurOnlyUnderRevealOnBlur(t *testing.T) {
	// The gate. Under any other policy nothing reads the edge, so the field
	// must not make both native renderers dispatch an event per focus change
	// — and, more visibly here, must render exactly the node it always did.
	cases := []struct {
		name   string
		reveal forms.Reveal
		want   bool
	}{
		{"OnSubmit", forms.RevealOnSubmit, false},
		{"OnTouch", forms.RevealOnTouch, false},
		{"Always", forms.RevealAlways, false},
		{"OnBlur", forms.RevealOnBlur, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := core.NewContext()
			form := render(ctx, forms.Spec{
				Reveal: tc.reveal,
				Fields: []forms.Field{{Name: "email"}},
			})
			n := form.Input("email", "you@example.com").Render(ctx)
			_, got := n.Props["onBlur"]
			if got != tc.want {
				t.Fatalf("onBlur present = %v, want %v: %#v", got, tc.want, n.Props)
			}
		})
	}
}

func TestBoundBuildersBlurHandlerMarksTheRightField(t *testing.T) {
	// The bound builders exist because the unbound spelling names the field
	// three times; the blur binding is a fourth, and it is the one nobody
	// writes by hand, so it had better be the same name.
	//
	// Every field is exercised, not just the first: a binding that closed
	// over a constant name rather than the argument would satisfy a test that
	// only ever blurred the field it happened to hard-code.
	for _, name := range []string{"email", "name"} {
		t.Run(name, func(t *testing.T) {
			ctx := core.NewContext()
			form := render(ctx, blurSpec())

			n := form.Input(name, "ph").Render(ctx)
			ctx.TriggerCallback(n.Props["onBlur"].(string))

			if !form.Blurred(name) {
				t.Errorf("dispatching %s's onBlur did not mark it blurred", name)
			}
			for _, other := range []string{"email", "name"} {
				if other != name && form.Blurred(other) {
					t.Errorf("blurring %s also marked %s", name, other)
				}
			}
		})
	}
}

func TestEveryTextBuilderCarriesTheBlurBinding(t *testing.T) {
	// A text builder that forgets it is a field whose error never appears
	// under RevealOnBlur — silent, and invisible in a diff.
	ctx := core.NewContext()
	form := render(ctx, forms.Spec{
		Reveal: forms.RevealOnBlur,
		Fields: []forms.Field{{Name: "f"}},
	})

	builders := map[string]core.View{
		"Input":           form.Input("f", "ph"),
		"InputWithSubmit": form.InputWithSubmit("f", "ph", nil),
		"Password":        form.Password("f", "ph"),
		"TextArea":        form.TextArea("f", 3),
	}
	for name, v := range builders {
		if _, ok := v.Render(ctx).Props["onBlur"]; !ok {
			t.Errorf("%s carries no blur binding under RevealOnBlur", name)
		}
	}

	// The checkbox deliberately does not: a tick is a commit, not a draft.
	if _, ok := form.Checkbox("f").Render(ctx).Props["onBlur"]; ok {
		t.Error("Checkbox took a blur binding; a checkbox has no draft state to leave")
	}
}

func TestBoundBuilderKeepsCallerPropsAndStyle(t *testing.T) {
	// withBlur appends, so the caller's props keep the order they were
	// written in — and it must not have replaced them.
	ctx := core.NewContext()
	form := render(ctx, blurSpec())

	n := form.Input("email", "ph", core.Padding(8), core.OnFocus(func() {})).Render(ctx)
	if n.Style.Padding.Top != 8 {
		t.Errorf("caller style prop lost: Padding.Top = %v", n.Style.Padding.Top)
	}
	if _, ok := n.Props["onFocus"]; !ok {
		t.Errorf("caller behavior prop lost: %#v", n.Props)
	}
	if _, ok := n.Props["onBlur"]; !ok {
		t.Errorf("blur binding lost: %#v", n.Props)
	}
}

func TestFirstBlurRequestsARenderAndRepeatsDoNot(t *testing.T) {
	// The record hangs off a pointer, so core.State.Set never fires and
	// nothing else would ask for the repaint — without the explicit request
	// the revealed error would sit in the record until some unrelated event
	// happened to trigger a pass.
	//
	// The second half is the reason MarkBlurred bothers to check: the
	// platform dispatches blur on every focus change, including all the ones
	// a user makes moving back through a form they have already been round,
	// and re-asserting a flag that is already set can change nothing on
	// screen.
	ctx := core.NewContext()
	form := render(ctx, blurSpec())

	ctx.ClearDirty()
	form.MarkBlurred("email")
	if !ctx.IsDirty() {
		t.Error("the first blur did not request a render")
	}

	ctx.ClearDirty()
	form.MarkBlurred("email")
	if ctx.IsDirty() {
		t.Error("a repeat blur requested a render that can change nothing")
	}

	// A different field is a different flag, so it must still ask.
	ctx.ClearDirty()
	form.MarkBlurred("name")
	if !ctx.IsDirty() {
		t.Error("the first blur of a second field did not request a render")
	}
}

func TestBlurredIsFalseWhenNothingReportsIt(t *testing.T) {
	// Blurred reports what was observed, not what happened. Under a policy
	// that wires no blur, that is honestly nothing.
	ctx := core.NewContext()
	form := render(ctx, forms.Spec{
		Reveal: forms.RevealOnTouch,
		Fields: []forms.Field{{Name: "email"}},
	})
	form.Input("email", "ph").Render(ctx)
	if form.Blurred("email") {
		t.Error("Blurred true with no blur ever dispatched")
	}
}

func TestMarkBlurredIsSafeConcurrently(t *testing.T) {
	// Blur arrives from the platform, off the render goroutine, and both
	// edges of a focus move can land at once.
	ctx := core.NewContext()
	form := render(ctx, blurSpec())

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				form.MarkBlurred("email")
			} else {
				form.MarkBlurred("name")
			}
		}(i)
	}
	wg.Wait()

	if !form.Blurred("email") || !form.Blurred("name") {
		t.Error("a concurrent blur was lost")
	}
}
