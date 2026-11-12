package signup

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/grmobtest"
)

// TestMain turns on debug mode for the whole package: every render pass driven
// below is then audited for cursor drift, and the tests assert the audit came
// back empty. That matters more here than in most examples — App branches
// between the form and the confirmation screen, which is exactly the shape
// that shifts hook slots when the hooks are not hoisted above the branch.
func TestMain(m *testing.M) {
	core.SetDebugMode(true)
	m.Run()
}

// blur dispatches the field's onBlur, exactly as the platform does when focus
// leaves it: BasicTextField's interaction source on Android, @FocusState on
// iOS. Under RevealOnBlur the prop is there because the bound builder put it
// there, so a missing one is a real failure rather than a test that needs
// updating — hence the Fatalf rather than a skip.
func blur(t *testing.T, app *grmobtest.Driver, placeholder string) string {
	t.Helper()
	n := grmobtest.Find(app.Tree(), func(n *core.Node) bool {
		return n.Props["placeholder"] == placeholder
	})
	if n == nil {
		t.Fatalf("no field with placeholder %q in the current tree", placeholder)
	}
	id, ok := n.Props["onBlur"].(string)
	if !ok {
		t.Fatalf("field %q carries no onBlur under RevealOnBlur: %#v", placeholder, n.Props)
	}
	return app.Manager().DispatchCallback(id)
}

func tickTerms(t *testing.T, app *grmobtest.Driver, on bool) {
	t.Helper()
	n := grmobtest.Find(app.Tree(), func(n *core.Node) bool { return n.Type == "Checkbox" })
	if n == nil {
		t.Fatal("no Checkbox in the current tree")
	}
	app.Manager().DispatchBoolCallback(n.Props["onToggle"].(string), on)
}

// enterCode types a whole code into the verification step's PINInput, as a
// paste or an SMS autofill arrives: one change carrying every digit. The
// field is found by the name PINInput gives it, "<Label>, N of M entered".
func enterCode(t *testing.T, app *grmobtest.Driver, code string) string {
	t.Helper()
	n := grmobtest.Find(app.Tree(), func(n *core.Node) bool {
		return n.Type == "Input" && n.Props["keyboard"] == "digits"
	})
	if n == nil {
		t.Fatal("no code field in the current tree")
	}
	return app.Manager().DispatchTextCallback(n.Props["onChange"].(string), code)
}

// newApp starts the form under the default theme. grmobtest resets the
// process-wide state (the concern collector among it) on the way in and out,
// so no test inherits another's findings whatever order they run in.
func newApp(t *testing.T) *grmobtest.Driver {
	t.Helper()
	return grmobtest.NewWithContext(t, core.NewContext().WithTheme(core.DefaultTheme), App)
}

// fill types a complete, valid set of answers.
func fill(t *testing.T, app *grmobtest.Driver, email string) {
	t.Helper()
	app.Type("you@example.com", email)
	// Type matches on placeholder and both password fields share one, so it
	// always lands on the first. The confirmation is reached by position.
	app.Type("••••••••", "hunter2222")
	n := secondPasswordField(t, app)
	app.Manager().DispatchTextCallback(n.Props["onChange"].(string), "hunter2222")
	tickTerms(t, app, true)
}

// secondPasswordField returns the confirmation field. The two password inputs
// are distinguished only by position, which is why this walks rather than
// matching on a prop.
func secondPasswordField(t *testing.T, app *grmobtest.Driver) *core.Node {
	t.Helper()
	var found []*core.Node
	var walk func(*core.Node)
	walk = func(n *core.Node) {
		if n == nil {
			return
		}
		if n.Type == "InputPassword" {
			found = append(found, n)
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(app.Tree())
	if len(found) != 2 {
		t.Fatalf("expected 2 password fields, found %d", len(found))
	}
	return found[1]
}

// An empty form is invalid from the first render and says nothing about it:
// nothing has been left and nothing submitted, so RevealOnBlur has no reason
// to speak yet. The hints are what the user sees instead.
func TestNothingComplainsBeforeTheFirstSubmit(t *testing.T) {
	app := newApp(t)
	initial := app.Manager().RenderInitial()

	for _, msg := range []string{
		"We need an address to reach you",
		"Use at least 8 characters",
		"Please accept the terms to continue",
		"Required",
	} {
		if strings.Contains(initial, msg) {
			t.Errorf("untouched form is already complaining: %q", msg)
		}
	}
	if !strings.Contains(initial, "We never share it") {
		t.Error("the hint should be showing while there is no error")
	}
	app.NoConcerns()
}

// A failed submit is the event that turns the explanations on — and from then
// on each field clears the instant it is fixed, with no second submit.
func TestFailedSubmitRevealsEverythingThenClearsLive(t *testing.T) {
	app := newApp(t)

	after := app.Tap("Create account")
	for _, msg := range []string{
		"We need an address to reach you",
		"Please accept the terms to continue",
	} {
		if !strings.Contains(after, msg) {
			t.Errorf("submit did not reveal %q:\n%s", msg, after)
		}
	}
	// An error outranks the hint: FormField shows one line of feedback.
	if strings.Contains(after, "We never share it") {
		t.Error("the hint should have been replaced by the error")
	}

	// Fixing one field clears only that field's message.
	app.Type("you@example.com", "you@example.com")
	live := app.Manager().RenderInitial()
	if strings.Contains(live, "We need an address to reach you") {
		t.Error("the email error should clear as soon as it is fixed")
	}
	if !strings.Contains(live, "Please accept the terms to continue") {
		t.Error("the other fields' errors must survive an unrelated edit")
	}
	if !strings.Contains(live, "We never share it") {
		t.Error("the hint should return once the field is valid")
	}
	app.NoConcerns()
}

// Rule order in action: the first *failing* rule speaks, so a half-typed
// address reports its shape rather than its absence.
func TestTheFirstFailingRuleIsTheOneShown(t *testing.T) {
	app := newApp(t)
	app.Tap("Create account")

	app.Type("you@example.com", "you@example")
	after := app.Manager().RenderInitial()
	if strings.Contains(after, "We need an address to reach you") {
		t.Error("Required should be satisfied by any non-empty value")
	}
	if !strings.Contains(after, "Not a valid email address") {
		t.Errorf("expected the Email rule's default message:\n%s", after)
	}
}

// The cross-field check: no single field's rules can see both passwords.
func TestConfirmationMustMatch(t *testing.T) {
	app := newApp(t)
	app.Type("••••••••", "hunter2222")
	n := secondPasswordField(t, app)
	app.Manager().DispatchTextCallback(n.Props["onChange"].(string), "hunter1111")

	after := app.Tap("Create account")
	if !strings.Contains(after, "The two passwords differ") {
		t.Errorf("expected the cross-field message:\n%s", after)
	}

	n = secondPasswordField(t, app)
	app.Manager().DispatchTextCallback(n.Props["onChange"].(string), "hunter2222")
	if fixed := app.Manager().RenderInitial(); strings.Contains(fixed, "The two passwords differ") {
		t.Error("the cross-field error should clear once the values agree")
	}
	app.NoConcerns()
}

// A verdict the rules could not have reached, arriving after a valid submit.
func TestServerErrorShowsThenClearsWhenTheFieldChanges(t *testing.T) {
	app := newApp(t)
	fill(t, app, "taken@example.com")

	after := app.Tap("Create account")
	if !strings.Contains(after, "That address is already registered") {
		t.Fatalf("expected the external error:\n%s", after)
	}
	if strings.Contains(after, "Account created") {
		t.Fatal("the account should not have been created")
	}

	// It disappears as the user starts fixing it, not after another round
	// trip — the verdict was about the old text.
	app.Type("you@example.com", "fresh@example.com")
	if edited := app.Manager().RenderInitial(); strings.Contains(edited, "That address is already registered") {
		t.Error("the external error should be dropped when the field changes")
	}
	app.NoConcerns()
}

// The whole path, plus the reset: a second visit must open quiet rather than
// still showing the first one's complaints.
func TestSuccessfulSubmitAndReset(t *testing.T) {
	app := newApp(t)
	fill(t, app, "  New@Example.com  ")

	sent := app.Tap("Create account")
	if !strings.Contains(sent, "Check your inbox") || !strings.Contains(sent, "code to New@Example.com.") {
		t.Fatalf("expected the verification step, addressed to the trimmed address:\n%s", sent)
	}
	after := enterCode(t, app, demoCode)
	if !strings.Contains(after, "Account created") {
		t.Fatalf("expected the confirmation screen:\n%s", after)
	}
	// Trimmed, not raw: the value passed Required (which trims) still padded.
	if !strings.Contains(after, "on its way to New@Example.com.") {
		t.Errorf("the handler should have stored the trimmed address:\n%s", after)
	}

	back := app.Tap("Create another")
	if strings.Contains(back, "Please accept the terms to continue") {
		t.Errorf("Reset should have cleared the submitted flag:\n%s", back)
	}
	if !strings.Contains(back, "We never share it") {
		t.Errorf("the fresh form should be showing its hints:\n%s", back)
	}
	// And it is genuinely empty, not merely quiet.
	n := grmobtest.Find(app.Tree(), func(n *core.Node) bool { return n.Props["placeholder"] == "you@example.com" })
	if got := n.Props["value"]; got != "" {
		t.Errorf("email value after reset = %v, want empty", got)
	}
	app.NoConcerns()
}

// The marker is annotation, not validation — but it is fed from the form, so
// it appears exactly on the labelled fields whose rules reject an empty value
// and nowhere else. The terms field is required and carries no marker only
// because it has no label of its own to hang one on.
func TestRequiredMarkersFollowTheRules(t *testing.T) {
	app := newApp(t)

	markers := 0
	var walk func(*core.Node)
	walk = func(n *core.Node) {
		if n == nil {
			return
		}
		if n.Type == "Text" && n.Props["content"] == "*" {
			markers++
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(app.Tree())

	// Email, password, confirm. Not terms (no label) and not the title.
	if markers != 3 {
		t.Errorf("found %d required markers, want 3 (email, password, confirm)", markers)
	}
	app.NoConcerns()
}

// The reveal policy end to end: the field says nothing while it is being
// typed into, explains itself the moment focus leaves, and confirms the fix
// live — all through the same callback IDs a native shell would dispatch.
func TestLeavingAFieldRevealsOnlyThatFieldsError(t *testing.T) {
	app := newApp(t)

	// Two characters in, RevealOnTouch would already be calling this an
	// invalid address. RevealOnBlur says nothing: the user is not done.
	app.Type("you@example.com", "yo")
	mid := app.Manager().RenderInitial()
	if strings.Contains(mid, "Not a valid email address") {
		t.Errorf("the form complained mid-typing:\n%s", mid)
	}
	if !strings.Contains(mid, "We never share it") {
		t.Error("the hint should still be showing while the field is being edited")
	}

	// Leaving it is the user's claim to have finished.
	after := blur(t, app, "you@example.com")
	if !strings.Contains(after, "Not a valid email address") {
		t.Errorf("blur did not reveal the address error:\n%s", after)
	}
	// And only that field: the untouched password must stay quiet.
	if strings.Contains(after, "Use at least 8 characters") {
		t.Errorf("blur revealed a sibling field's error:\n%s", after)
	}

	// Live from then on, with no second blur.
	app.Type("you@example.com", "you@example.com")
	fixed := app.Manager().RenderInitial()
	if strings.Contains(fixed, "Not a valid email address") {
		t.Errorf("the error should clear the instant it is fixed:\n%s", fixed)
	}
	if !strings.Contains(fixed, "We never share it") {
		t.Error("the hint should return once the field is valid")
	}
	app.NoConcerns()
}

// Tabbing through a field without typing in it is still leaving it, so an
// empty required field reports itself. This is the case RevealOnTouch cannot
// see at all — nothing was edited.
func TestTabbingThroughAnEmptyFieldRevealsRequired(t *testing.T) {
	app := newApp(t)

	after := blur(t, app, "you@example.com")
	if !strings.Contains(after, "We need an address to reach you") {
		t.Errorf("an empty field left behind said nothing:\n%s", after)
	}
	app.NoConcerns()
}

// Reset clears the blur marks with everything else, so the second visit opens
// as quiet as the first.
func TestResetClearsTheBlurMarks(t *testing.T) {
	app := newApp(t)

	blur(t, app, "you@example.com")
	fill(t, app, "new@example.com")
	app.Tap("Create account")
	enterCode(t, app, demoCode)

	after := app.Tap("Create another")
	for _, msg := range []string{
		"We need an address to reach you",
		"Please accept the terms to continue",
	} {
		if strings.Contains(after, msg) {
			t.Errorf("the second visit inherited a complaint: %q\n%s", msg, after)
		}
	}
	app.NoConcerns()
}

// The server-error path's other half: an error message alone leaves the user
// to find the field again, since a submit has usually scrolled past it or put
// the keyboard away. The screen names the email field with a FocusTarget and
// core.Focus puts the cursor back in it.
func TestServerErrorPutsTheCursorBackInTheEmailField(t *testing.T) {
	app := newApp(t)
	fill(t, app, "taken@example.com")

	// Before the submit nothing has issued a command, so no field carries a
	// stamp at all — the property that keeps this feature free for screens
	// that never use it.
	email := func() *core.Node {
		return grmobtest.Find(app.Tree(), func(n *core.Node) bool {
			return n.Props["placeholder"] == "you@example.com"
		})
	}
	if _, ok := email().Props["focusEpoch"]; ok {
		t.Fatalf("the email field carries a stamp before any command: %#v", email().Props)
	}

	app.Tap("Create account")

	got := email().Props
	if got["focusAction"] != "focus" {
		t.Errorf("email focusAction = %v, want focus:\n%#v", got["focusAction"], got)
	}
	// The password field is stamped with the same command and nothing to do,
	// rather than being told to blur — a blur there would race the email
	// field's focus request on both native platforms.
	pw := grmobtest.Find(app.Tree(), func(n *core.Node) bool {
		return n.Props["placeholder"] == "••••••••"
	})
	if pw.Props["focusAction"] != "" {
		t.Errorf("password focusAction = %v, want empty:\n%#v", pw.Props["focusAction"], pw.Props)
	}
	app.NoConcerns()
}

// A submit that succeeds must not focus anything: the form is gone, and a
// stray command would land on whatever the confirmation screen shows.
func TestSuccessfulSubmitIssuesNoFocusCommand(t *testing.T) {
	app := newApp(t)
	fill(t, app, "fresh@example.com")

	after := app.Tap("Create account")
	if !strings.Contains(after, "Check your inbox") {
		t.Fatalf("expected the verification step:\n%s", after)
	}
	if strings.Contains(after, "focusEpoch") {
		t.Errorf("a successful submit issued a focus command:\n%s", after)
	}
	app.NoConcerns()
}

// The keyboard walks the form. core.UseFocusOrder names the three fields in
// one line at the top of App; what follows on screen is a return key that
// reads "Next" on every field but the last.
func TestTheFormAdvertisesTheNextAction(t *testing.T) {
	app := newApp(t)

	email := grmobtest.Find(app.Tree(), func(n *core.Node) bool {
		return n.Props["placeholder"] == "you@example.com"
	})
	password := grmobtest.Find(app.Tree(), func(n *core.Node) bool {
		return n.Type == "InputPassword"
	})
	confirm := secondPasswordField(t, app)

	for _, f := range []struct {
		name string
		n    *core.Node
		want string
	}{
		{"email", email, "next"},
		{"password", password, "next"},
		// Nothing follows the confirmation field, so its key stays the
		// platform's own. The terms checkbox is deliberately not in the
		// order: no platform here gives a checkbox keyboard focus, so a
		// fourth entry would advertise a Next that landed nowhere.
		{"confirm", confirm, ""},
	} {
		if got := f.n.Props["imeAction"]; got != f.want {
			t.Errorf("%s advertises imeAction %v, want %q", f.name, got, f.want)
		}
	}
	app.NoConcerns()
}

// Pressing that key is an ordinary submit dispatch — the same thing the
// renderers do for a Done key — and it lands the cursor one field down.
func TestNextOnTheEmailFieldFocusesThePassword(t *testing.T) {
	app := newApp(t)

	email := grmobtest.Find(app.Tree(), func(n *core.Node) bool {
		return n.Props["placeholder"] == "you@example.com"
	})
	id, ok := email.Props["onSubmit"].(string)
	if !ok {
		t.Fatalf("the email field advertises Next with no submit wired: %#v", email.Props)
	}
	app.Manager().DispatchCallback(id)

	password := grmobtest.Find(app.Tree(), func(n *core.Node) bool {
		return n.Type == "InputPassword"
	})
	if got := password.Props["focusAction"]; got != "focus" {
		t.Errorf("password focusAction = %v after Next on the email field, want focus:\n%#v",
			got, password.Props)
	}
	// And the field it came from is told nothing to do, rather than to blur —
	// requesting focus in the password field already takes it from here, and
	// having both act would race on both native platforms.
	email = grmobtest.Find(app.Tree(), func(n *core.Node) bool {
		return n.Props["placeholder"] == "you@example.com"
	})
	if got := email.Props["focusAction"]; got != "" {
		t.Errorf("email focusAction = %v after handing focus on, want empty", got)
	}
	app.NoConcerns()
}

// The end of the order is the end of the walk: the confirmation field has no
// submit to dispatch at all, so its return key falls through to the platform
// rather than quietly wrapping back to the top of the form.
func TestTheLastFieldHasNoNextToPress(t *testing.T) {
	app := newApp(t)
	if _, ok := secondPasswordField(t, app).Props["onSubmit"]; ok {
		t.Error("the confirmation field was wired a submit it has nothing to do with")
	}
	app.NoConcerns()
}

// The verification step: a wrong code is refused with a message and cleared,
// so the reader types again rather than deleting six digits; typing again
// clears the message; the right one creates the account. And "Use a different
// address" goes back to the form with its values kept.
func TestTheCodeStepRefusesAWrongCodeAndGoesBack(t *testing.T) {
	app := newApp(t)
	fill(t, app, "fresh@example.com")
	app.Tap("Create account")

	wrong := enterCode(t, app, "111111")
	if !strings.Contains(wrong, "That code is not the one we sent") {
		t.Fatalf("a wrong code said nothing:\n%s", wrong)
	}
	field := grmobtest.Find(app.Tree(), func(n *core.Node) bool { return n.Type == "Input" && n.Props["keyboard"] == "digits" })
	if field.Props["value"] != "" {
		t.Errorf("the wrong code was left in the field: %v", field.Props["value"])
	}
	if again := enterCode(t, app, "2"); strings.Contains(again, "That code is not the one we sent") {
		t.Error("typing again kept the message")
	}

	back := app.Tap("Use a different address")
	if !strings.Contains(back, "Create your account") {
		t.Fatalf("expected the form again:\n%s", back)
	}
	email := grmobtest.Find(app.Tree(), func(n *core.Node) bool { return n.Props["placeholder"] == "you@example.com" })
	if email.Props["value"] != "fresh@example.com" {
		t.Errorf("the form lost its address: %v", email.Props["value"])
	}
	// And a second submit starts the step clean.
	app.Tap("Create account")
	if strings.Contains(app.Manager().RenderInitial(), "That code is not the one we sent") {
		t.Error("the second visit to the step inherited the first one's message")
	}
	if got := enterCode(t, app, demoCode); !strings.Contains(got, "Account created") {
		t.Fatalf("the right code did not create the account:\n%s", got)
	}
	app.NoConcerns()
}
