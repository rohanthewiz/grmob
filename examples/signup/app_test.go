package signup

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/render"
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

type node struct {
	Type     string
	Props    map[string]any
	Children []*node
}

func findNode(n *node, pred func(*node) bool) *node {
	if n == nil {
		return nil
	}
	if pred(n) {
		return n
	}
	for _, c := range n.Children {
		if found := findNode(c, pred); found != nil {
			return found
		}
	}
	return nil
}

// tree re-renders and parses the current tree. Callback IDs are per-pass
// sequence numbers, so every dispatch reads its ID from a freshly rendered
// tree rather than reusing one — the discipline the native shells follow.
func tree(t *testing.T, mgr *render.Manager) *node {
	t.Helper()
	var root node
	if err := json.Unmarshal([]byte(mgr.RenderInitial()), &root); err != nil {
		t.Fatalf("tree is not valid JSON: %v", err)
	}
	return &root
}

// typeInto finds the field showing the given placeholder and dispatches its
// change callback, exactly as a keystroke arriving from a native control does.
func typeInto(t *testing.T, mgr *render.Manager, placeholder, value string) {
	t.Helper()
	n := findNode(tree(t, mgr), func(n *node) bool {
		return n.Props["placeholder"] == placeholder
	})
	if n == nil {
		t.Fatalf("no field with placeholder %q in the current tree", placeholder)
	}
	mgr.DispatchTextCallback(n.Props["onChange"].(string), value)
}

// blur dispatches the field's onBlur, exactly as the platform does when focus
// leaves it: BasicTextField's interaction source on Android, @FocusState on
// iOS. Under RevealOnBlur the prop is there because the bound builder put it
// there, so a missing one is a real failure rather than a test that needs
// updating — hence the Fatalf rather than a skip.
func blur(t *testing.T, mgr *render.Manager, placeholder string) string {
	t.Helper()
	n := findNode(tree(t, mgr), func(n *node) bool {
		return n.Props["placeholder"] == placeholder
	})
	if n == nil {
		t.Fatalf("no field with placeholder %q in the current tree", placeholder)
	}
	id, ok := n.Props["onBlur"].(string)
	if !ok {
		t.Fatalf("field %q carries no onBlur under RevealOnBlur: %#v", placeholder, n.Props)
	}
	return mgr.DispatchCallback(id)
}

func tickTerms(t *testing.T, mgr *render.Manager, on bool) {
	t.Helper()
	n := findNode(tree(t, mgr), func(n *node) bool { return n.Type == "Checkbox" })
	if n == nil {
		t.Fatal("no Checkbox in the current tree")
	}
	mgr.DispatchBoolCallback(n.Props["onToggle"].(string), on)
}

func tap(t *testing.T, mgr *render.Manager, label string) string {
	t.Helper()
	n := findNode(tree(t, mgr), func(n *node) bool {
		return n.Type == "Button" && n.Props["label"] == label
	})
	if n == nil {
		t.Fatalf("no Button labeled %q in the current tree", label)
	}
	return mgr.DispatchCallback(n.Props["onClick"].(string))
}

func assertNoConcerns(t *testing.T) {
	t.Helper()
	if cs := core.Concerns(); len(cs) != 0 {
		t.Fatalf("debug concerns raised:\n%s", core.DumpConcerns())
	}
}

func newApp(t *testing.T) *render.Manager {
	t.Helper()
	core.ClearConcerns() // the collector is process-wide; do not inherit
	mgr := render.New(core.NewContext().WithTheme(core.DefaultTheme), App)
	t.Cleanup(mgr.Close)
	return mgr
}

// fill types a complete, valid set of answers.
func fill(t *testing.T, mgr *render.Manager, email string) {
	t.Helper()
	typeInto(t, mgr, "you@example.com", email)
	// typeInto matches on placeholder and both password fields share one, so
	// it always lands on the first. The confirmation is reached by position.
	typeInto(t, mgr, "••••••••", "hunter2222")
	n := secondPasswordField(t, mgr)
	mgr.DispatchTextCallback(n.Props["onChange"].(string), "hunter2222")
	tickTerms(t, mgr, true)
}

// secondPasswordField returns the confirmation field. The two password inputs
// are distinguished only by position, which is why this walks rather than
// matching on a prop.
func secondPasswordField(t *testing.T, mgr *render.Manager) *node {
	t.Helper()
	var found []*node
	var walk func(*node)
	walk = func(n *node) {
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
	walk(tree(t, mgr))
	if len(found) != 2 {
		t.Fatalf("expected 2 password fields, found %d", len(found))
	}
	return found[1]
}

// An empty form is invalid from the first render and says nothing about it:
// nothing has been left and nothing submitted, so RevealOnBlur has no reason
// to speak yet. The hints are what the user sees instead.
func TestNothingComplainsBeforeTheFirstSubmit(t *testing.T) {
	mgr := newApp(t)
	initial := mgr.RenderInitial()

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
	assertNoConcerns(t)
}

// A failed submit is the event that turns the explanations on — and from then
// on each field clears the instant it is fixed, with no second submit.
func TestFailedSubmitRevealsEverythingThenClearsLive(t *testing.T) {
	mgr := newApp(t)

	after := tap(t, mgr, "Create account")
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
	typeInto(t, mgr, "you@example.com", "you@example.com")
	live := mgr.RenderInitial()
	if strings.Contains(live, "We need an address to reach you") {
		t.Error("the email error should clear as soon as it is fixed")
	}
	if !strings.Contains(live, "Please accept the terms to continue") {
		t.Error("the other fields' errors must survive an unrelated edit")
	}
	if !strings.Contains(live, "We never share it") {
		t.Error("the hint should return once the field is valid")
	}
	assertNoConcerns(t)
}

// Rule order in action: the first *failing* rule speaks, so a half-typed
// address reports its shape rather than its absence.
func TestTheFirstFailingRuleIsTheOneShown(t *testing.T) {
	mgr := newApp(t)
	tap(t, mgr, "Create account")

	typeInto(t, mgr, "you@example.com", "you@example")
	after := mgr.RenderInitial()
	if strings.Contains(after, "We need an address to reach you") {
		t.Error("Required should be satisfied by any non-empty value")
	}
	if !strings.Contains(after, "Not a valid email address") {
		t.Errorf("expected the Email rule's default message:\n%s", after)
	}
}

// The cross-field check: no single field's rules can see both passwords.
func TestConfirmationMustMatch(t *testing.T) {
	mgr := newApp(t)
	typeInto(t, mgr, "••••••••", "hunter2222")
	n := secondPasswordField(t, mgr)
	mgr.DispatchTextCallback(n.Props["onChange"].(string), "hunter1111")

	after := tap(t, mgr, "Create account")
	if !strings.Contains(after, "The two passwords differ") {
		t.Errorf("expected the cross-field message:\n%s", after)
	}

	n = secondPasswordField(t, mgr)
	mgr.DispatchTextCallback(n.Props["onChange"].(string), "hunter2222")
	if fixed := mgr.RenderInitial(); strings.Contains(fixed, "The two passwords differ") {
		t.Error("the cross-field error should clear once the values agree")
	}
	assertNoConcerns(t)
}

// A verdict the rules could not have reached, arriving after a valid submit.
func TestServerErrorShowsThenClearsWhenTheFieldChanges(t *testing.T) {
	mgr := newApp(t)
	fill(t, mgr, "taken@example.com")

	after := tap(t, mgr, "Create account")
	if !strings.Contains(after, "That address is already registered") {
		t.Fatalf("expected the external error:\n%s", after)
	}
	if strings.Contains(after, "Account created") {
		t.Fatal("the account should not have been created")
	}

	// It disappears as the user starts fixing it, not after another round
	// trip — the verdict was about the old text.
	typeInto(t, mgr, "you@example.com", "fresh@example.com")
	if edited := mgr.RenderInitial(); strings.Contains(edited, "That address is already registered") {
		t.Error("the external error should be dropped when the field changes")
	}
	assertNoConcerns(t)
}

// The whole path, plus the reset: a second visit must open quiet rather than
// still showing the first one's complaints.
func TestSuccessfulSubmitAndReset(t *testing.T) {
	mgr := newApp(t)
	fill(t, mgr, "  New@Example.com  ")

	after := tap(t, mgr, "Create account")
	if !strings.Contains(after, "Account created") {
		t.Fatalf("expected the confirmation screen:\n%s", after)
	}
	// Trimmed, not raw: the value passed Required (which trims) still padded.
	if !strings.Contains(after, "on its way to New@Example.com.") {
		t.Errorf("the handler should have stored the trimmed address:\n%s", after)
	}

	back := tap(t, mgr, "Create another")
	if strings.Contains(back, "Please accept the terms to continue") {
		t.Errorf("Reset should have cleared the submitted flag:\n%s", back)
	}
	if !strings.Contains(back, "We never share it") {
		t.Errorf("the fresh form should be showing its hints:\n%s", back)
	}
	// And it is genuinely empty, not merely quiet.
	n := findNode(tree(t, mgr), func(n *node) bool { return n.Props["placeholder"] == "you@example.com" })
	if got := n.Props["value"]; got != "" {
		t.Errorf("email value after reset = %v, want empty", got)
	}
	assertNoConcerns(t)
}

// The marker is annotation, not validation — but it is fed from the form, so
// it appears exactly on the labelled fields whose rules reject an empty value
// and nowhere else. The terms field is required and carries no marker only
// because it has no label of its own to hang one on.
func TestRequiredMarkersFollowTheRules(t *testing.T) {
	mgr := newApp(t)

	markers := 0
	var walk func(*node)
	walk = func(n *node) {
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
	walk(tree(t, mgr))

	// Email, password, confirm. Not terms (no label) and not the title.
	if markers != 3 {
		t.Errorf("found %d required markers, want 3 (email, password, confirm)", markers)
	}
	assertNoConcerns(t)
}

// The reveal policy end to end: the field says nothing while it is being
// typed into, explains itself the moment focus leaves, and confirms the fix
// live — all through the same callback IDs a native shell would dispatch.
func TestLeavingAFieldRevealsOnlyThatFieldsError(t *testing.T) {
	mgr := newApp(t)

	// Two characters in, RevealOnTouch would already be calling this an
	// invalid address. RevealOnBlur says nothing: the user is not done.
	typeInto(t, mgr, "you@example.com", "yo")
	mid := mgr.RenderInitial()
	if strings.Contains(mid, "Not a valid email address") {
		t.Errorf("the form complained mid-typing:\n%s", mid)
	}
	if !strings.Contains(mid, "We never share it") {
		t.Error("the hint should still be showing while the field is being edited")
	}

	// Leaving it is the user's claim to have finished.
	after := blur(t, mgr, "you@example.com")
	if !strings.Contains(after, "Not a valid email address") {
		t.Errorf("blur did not reveal the address error:\n%s", after)
	}
	// And only that field: the untouched password must stay quiet.
	if strings.Contains(after, "Use at least 8 characters") {
		t.Errorf("blur revealed a sibling field's error:\n%s", after)
	}

	// Live from then on, with no second blur.
	typeInto(t, mgr, "you@example.com", "you@example.com")
	fixed := mgr.RenderInitial()
	if strings.Contains(fixed, "Not a valid email address") {
		t.Errorf("the error should clear the instant it is fixed:\n%s", fixed)
	}
	if !strings.Contains(fixed, "We never share it") {
		t.Error("the hint should return once the field is valid")
	}
	assertNoConcerns(t)
}

// Tabbing through a field without typing in it is still leaving it, so an
// empty required field reports itself. This is the case RevealOnTouch cannot
// see at all — nothing was edited.
func TestTabbingThroughAnEmptyFieldRevealsRequired(t *testing.T) {
	mgr := newApp(t)

	after := blur(t, mgr, "you@example.com")
	if !strings.Contains(after, "We need an address to reach you") {
		t.Errorf("an empty field left behind said nothing:\n%s", after)
	}
	assertNoConcerns(t)
}

// Reset clears the blur marks with everything else, so the second visit opens
// as quiet as the first.
func TestResetClearsTheBlurMarks(t *testing.T) {
	mgr := newApp(t)

	blur(t, mgr, "you@example.com")
	fill(t, mgr, "new@example.com")
	tap(t, mgr, "Create account")

	after := tap(t, mgr, "Create another")
	for _, msg := range []string{
		"We need an address to reach you",
		"Please accept the terms to continue",
	} {
		if strings.Contains(after, msg) {
			t.Errorf("the second visit inherited a complaint: %q\n%s", msg, after)
		}
	}
	assertNoConcerns(t)
}
