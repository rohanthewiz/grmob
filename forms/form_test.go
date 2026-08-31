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
