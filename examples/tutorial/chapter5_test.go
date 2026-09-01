package tutorial

import (
	"testing"

	"github.com/rohanthewiz/grmob/render"
)

// Chapter 5 demo-liveness tests. The form lessons hold several text fields per
// screen, so the chapter-2 first-Input helper is not enough: fields here are
// addressed by placeholder, the way the signup example's tests do it. Every
// dispatch still re-reads the tree first (callback IDs are per-pass), and
// TestMain's debug mode audits every pass — UseForm and UseFocusRef are hooks,
// so the audit is what catches a lesson that mis-hoists one.

// fieldByPlaceholder returns the current node of the input showing this
// placeholder — for prop assertions (value, onBlur, focusAction) as well as
// for dispatching.
func fieldByPlaceholder(t *testing.T, mgr *render.Manager, placeholder string) *node {
	t.Helper()
	n := findNode(tree(t, mgr), func(n *node) bool {
		return n.Props["placeholder"] == placeholder
	})
	if n == nil {
		t.Fatalf("no field with placeholder %q in the current tree", placeholder)
	}
	return n
}

// typeField dispatches the field's change callback, as a platform text
// watcher would.
func typeField(t *testing.T, mgr *render.Manager, placeholder, value string) {
	t.Helper()
	n := fieldByPlaceholder(t, mgr, placeholder)
	mgr.DispatchTextCallback(n.Props["onChange"].(string), value)
}

// blurField dispatches the field's onBlur, as the platform does when focus
// leaves it. The prop exists only when the bound builder attached it — that
// is, under RevealOnBlur — so a missing one here is a real failure, not a
// test to soften.
func blurField(t *testing.T, mgr *render.Manager, placeholder string) {
	t.Helper()
	n := fieldByPlaceholder(t, mgr, placeholder)
	id, ok := n.Props["onBlur"].(string)
	if !ok {
		t.Fatalf("field %q carries no onBlur — expected one under RevealOnBlur: %#v",
			placeholder, n.Props)
	}
	mgr.DispatchCallback(id)
}

// countMarkers counts FormField's required asterisks. The marker is its own
// Text node (it carries the error ink and an accessibility label), so it is
// countable without string surgery on the label.
func countMarkers(n *node) int {
	count := 0
	var walk func(*node)
	walk = func(n *node) {
		if n == nil {
			return
		}
		if n.Type == "Text" && n.Props["content"] == "*" {
			count++
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(n)
	return count
}

// --- 5.1 A form in four calls ----------------------------------------------

func TestFirstFormRevealsOnSubmitThenConfirmsLive(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "A form in four calls")

	// Untouched, the form shows hints and nothing else: under the default
	// policy nothing has earned a complaint yet.
	cur := tree(t, mgr)
	if !hasTextContaining(cur, "Used once, for the invite") {
		t.Fatal("the email hint should show while there is no error")
	}
	if hasTextContaining(cur, "Tell us who's coming") ||
		hasTextContaining(cur, "An address for the invite") {
		t.Fatal("an untouched form should not be complaining")
	}

	// The failed submit is the event that turns the explanations on — and an
	// error outranks the hint in FormField's single feedback line.
	tap(t, mgr, "RSVP")
	cur = tree(t, mgr)
	if !hasTextContaining(cur, "Tell us who's coming") ||
		!hasTextContaining(cur, "An address for the invite") {
		t.Fatal("a failed submit should reveal every field's error")
	}
	if hasTextContaining(cur, "Used once, for the invite") {
		t.Fatal("the error should have replaced the hint")
	}

	// From then on corrections confirm live, field by field.
	typeField(t, mgr, "June Gopher", "  June Gopher  ")
	cur = tree(t, mgr)
	if hasTextContaining(cur, "Tell us who's coming") {
		t.Fatal("the name error should clear the instant it is fixed")
	}
	if !hasTextContaining(cur, "An address for the invite") {
		t.Fatal("the email error must survive an unrelated edit")
	}

	// First-failing-rule ordering: a half-typed address reports its shape,
	// not its absence.
	typeField(t, mgr, "june@burrow.dev", "june@burrow")
	cur = tree(t, mgr)
	if hasTextContaining(cur, "An address for the invite") {
		t.Fatal("Required should be satisfied by any non-empty value")
	}
	if !hasTextContaining(cur, "Not a valid email address") {
		t.Fatal("expected the Email rule's default message")
	}

	// A clean submit hands the handler a copy — and the demo stores the
	// Trimmed name, so the padding typed above must be gone.
	typeField(t, mgr, "june@burrow.dev", "june@burrow.dev")
	tap(t, mgr, "RSVP")
	if !hasTextContaining(tree(t, mgr), "RSVP received for June Gopher") {
		t.Fatal("a valid submit should confirm with the trimmed name")
	}
	assertNoConcerns(t)
}

// --- 5.2 Rules & the required marker ----------------------------------------

func TestRulesPlaygroundOrderingAndDerivedMarker(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Rules & the required marker")

	// RevealAlways: the empty field complains from the first render, and the
	// derived marker reflects the Required rule being in the list.
	cur := tree(t, mgr)
	if !hasTextContaining(cur, "Every gopher needs a handle") {
		t.Fatal("RevealAlways should show the Required message immediately")
	}
	if got := countMarkers(cur); got != 1 {
		t.Fatalf("found %d required markers, want 1", got)
	}
	if !hasTextContaining(cur, `form.Required("handle") → true`) {
		t.Fatal("the caption should report the derived Required as true")
	}

	// First failing rule wins: a non-empty value moves the complaint to
	// MinLen, the next rule in the list.
	typeField(t, mgr, "gopherella", "ab")
	cur = tree(t, mgr)
	if hasTextContaining(cur, "Every gopher needs a handle") {
		t.Fatal("Required should be satisfied by any non-empty value")
	}
	if !hasTextContaining(cur, "Give it at least 5 characters") {
		t.Fatal("MinLen should be the first failing rule for a 2-rune value")
	}

	// The spec is re-read every pass: unchecking MinLen removes the rule, and
	// the same value is suddenly clean.
	toggleCheckbox(t, mgr, 1, false) // 0: Required, 1: MinLen, 2: Pattern
	if hasTextContaining(tree(t, mgr), "Give it at least 5 characters") {
		t.Fatal("a rule removed from the spec should stop firing on the next pass")
	}

	toggleCheckbox(t, mgr, 2, true)
	typeField(t, mgr, "gopherella", "Ab1")
	if !hasTextContaining(tree(t, mgr), "Lowercase letters only") {
		t.Fatal("the Pattern rule should fire once toggled in")
	}

	// The custom closure is always in the list — a plain func is a Rule.
	typeField(t, mgr, "gopherella", "root")
	if !hasTextContaining(tree(t, mgr), "That handle is reserved for the system") {
		t.Fatal("the closure rule should reject a reserved handle")
	}

	// Dropping Required takes the marker with it (it is derived, not
	// declared), and an empty value then draws no complaint at all: every
	// remaining rule is silent about "".
	toggleCheckbox(t, mgr, 0, false)
	typeField(t, mgr, "gopherella", "")
	cur = tree(t, mgr)
	if got := countMarkers(cur); got != 0 {
		t.Fatalf("found %d required markers after dropping Required, want 0", got)
	}
	if !hasTextContaining(cur, `form.Required("handle") → false`) {
		t.Fatal("the caption should report the derived Required as false")
	}
	if hasTextContaining(cur, "Lowercase letters only") ||
		hasTextContaining(cur, "reserved for the system") {
		t.Fatal("every rule but Required is silent about an empty value")
	}
	assertNoConcerns(t)
}

// --- 5.3 When errors appear --------------------------------------------------

func TestRevealPoliciesGateTheSameError(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "When errors appear")

	// Under the default OnSubmit policy the field carries no onBlur at all —
	// the binding is attached only under the policy that reads it.
	cur := tree(t, mgr)
	if _, ok := fieldByPlaceholder(t, mgr, "you@burrow.dev").Props["onBlur"]; ok {
		t.Fatal("no blur binding should be registered under RevealOnSubmit")
	}
	if !hasTextContaining(cur, "touched: false · blurred: false · submitted: false") {
		t.Fatal("the reveal inputs should open unset")
	}

	// OnSubmit: typing changes nothing on screen; the failed submit reveals.
	typeField(t, mgr, "you@burrow.dev", "yo")
	cur = tree(t, mgr)
	if hasTextContaining(cur, "Not a valid email address") {
		t.Fatal("RevealOnSubmit should say nothing while the user types")
	}
	if !hasTextContaining(cur, "touched: true · blurred: false · submitted: false") {
		t.Fatal("typing should mark the field touched")
	}
	tap(t, mgr, "Check the form")
	cur = tree(t, mgr)
	if !hasTextContaining(cur, "Not a valid email address") {
		t.Fatal("the failed submit should reveal the error")
	}
	if hasTextContaining(cur, "would have gone through") {
		t.Fatal("an invalid form must not claim its submit would pass")
	}

	// Live after the reveal — and a now-clean form says so.
	typeField(t, mgr, "you@burrow.dev", "you@burrow.dev")
	cur = tree(t, mgr)
	if hasTextContaining(cur, "Not a valid email address") {
		t.Fatal("the error should clear the instant it is fixed")
	}
	if !hasTextContaining(cur, "would have gone through") {
		t.Fatal("a submitted, now-valid form should say the submit would pass")
	}

	// Reset clears values and all three reveal inputs.
	tap(t, mgr, "Start over")
	cur = tree(t, mgr)
	if !hasTextContaining(cur, "touched: false · blurred: false · submitted: false") {
		t.Fatal("Reset should clear touched, blurred and submitted")
	}
	if got := fieldByPlaceholder(t, mgr, "you@burrow.dev").Props["value"]; got != "" {
		t.Fatalf("email value after reset = %v, want empty", got)
	}

	// OnBlur: silent while typing, speaks when focus leaves the field.
	tap(t, mgr, "OnBlur")
	typeField(t, mgr, "you@burrow.dev", "yo")
	if hasTextContaining(tree(t, mgr), "Not a valid email address") {
		t.Fatal("RevealOnBlur should say nothing while the field is being edited")
	}
	blurField(t, mgr, "you@burrow.dev")
	cur = tree(t, mgr)
	if !hasTextContaining(cur, "Not a valid email address") {
		t.Fatal("leaving the field should reveal its error under RevealOnBlur")
	}
	if !hasTextContaining(cur, "touched: true · blurred: true · submitted: false") {
		t.Fatal("the blur should be recorded as observed")
	}

	// OnTouch: the edit itself reveals — on the second keystroke, not when
	// the user is done.
	tap(t, mgr, "Start over")
	tap(t, mgr, "OnTouch")
	typeField(t, mgr, "you@burrow.dev", "yo")
	if !hasTextContaining(tree(t, mgr), "Not a valid email address") {
		t.Fatal("RevealOnTouch should reveal as soon as the field is edited")
	}

	// Always: the empty form complains with no interaction at all.
	tap(t, mgr, "Start over")
	tap(t, mgr, "Always")
	if !hasTextContaining(tree(t, mgr), "An address is needed") {
		t.Fatal("RevealAlways should show the error from the first render")
	}
	assertNoConcerns(t)
}

// --- 5.4 Cross-field & server errors -----------------------------------------

func TestCrossFieldAndServerErrors(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Cross-field & server errors")

	// With the confirmation empty, its own Required wins over Validate's
	// mismatch — the field rule is the more specific complaint.
	typeField(t, mgr, "gopher@burrow.dev", "fresh@burrow.dev")
	typeField(t, mgr, "choose a password", "hunter2222")
	tap(t, mgr, "Claim address")
	cur := tree(t, mgr)
	if !hasText(cur, "Required") {
		t.Fatal("the empty confirmation should show its own Required message")
	}
	if hasTextContaining(cur, "don't match the password") {
		t.Fatal("Validate must not outrank the field's own rule")
	}

	// A filled-but-different confirmation is the gap only Validate can see.
	typeField(t, mgr, "type it again", "hunter1111")
	if !hasTextContaining(tree(t, mgr), "These don't match the password above") {
		t.Fatal("the cross-field mismatch should show once Required is satisfied")
	}
	typeField(t, mgr, "type it again", "hunter2222")
	if hasTextContaining(tree(t, mgr), "These don't match the password above") {
		t.Fatal("the cross-field error should clear once the values agree")
	}

	// A verdict only the registry can reach, arriving through SetErrors — and
	// paired with a focus command that puts the cursor back on the problem.
	typeField(t, mgr, "gopher@burrow.dev", "taken@example.com")
	tap(t, mgr, "Claim address")
	cur = tree(t, mgr)
	if !hasTextContaining(cur, "Someone got there first") {
		t.Fatal("the taken address should come back with the server's error")
	}
	if hasTextContaining(cur, "✓ claimed") {
		t.Fatal("the taken address must not be claimed")
	}
	email := fieldByPlaceholder(t, mgr, "gopher@burrow.dev")
	if email.Props["focusAction"] != "focus" {
		t.Fatalf("email focusAction = %v, want focus", email.Props["focusAction"])
	}
	// The other fields are stamped with the same command and nothing to do —
	// not told to blur, which would race the focus on both native platforms.
	pw := fieldByPlaceholder(t, mgr, "choose a password")
	if pw.Props["focusAction"] != "" {
		t.Fatalf("password focusAction = %v, want empty", pw.Props["focusAction"])
	}

	// The external error drops on the field's first edit — the verdict was
	// about the old text, so no second round trip is needed to clear it.
	typeField(t, mgr, "gopher@burrow.dev", "fresh@burrow.dev")
	if hasTextContaining(tree(t, mgr), "Someone got there first") {
		t.Fatal("editing the field should drop the server's error")
	}
	tap(t, mgr, "Claim address")
	if !hasTextContaining(tree(t, mgr), "✓ claimed fresh@burrow.dev") {
		t.Fatal("a fresh address should claim successfully")
	}
	assertNoConcerns(t)
}

// --- 5.5 Values, initials & reset --------------------------------------------

func TestValuesInitialsAndReset(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Values, initials & reset")

	// Initial seeds the declared defaults: the quantity text and the ticked
	// gift box, with the live caption showing a clean parse.
	cur := tree(t, mgr)
	if got := fieldByPlaceholder(t, mgr, "how many?").Props["value"]; got != "2" {
		t.Fatalf("quantity should open at its Initial of 2, got %v", got)
	}
	if !hasTextContaining(cur, "→ (2, true)") {
		t.Fatal("the parse caption should show the seeded quantity")
	}
	gift := findNode(cur, func(n *node) bool { return n.Type == "Checkbox" })
	if gift == nil || gift.Props["checked"] != true {
		t.Fatal("the gift checkbox should open ticked from its Initial")
	}

	// Values are text: the unparseable value survives to the form, the ok
	// comes back false, and the Range rule gets its chance to complain.
	typeField(t, mgr, "how many?", "12x")
	if !hasTextContaining(tree(t, mgr), "→ (0, false)") {
		t.Fatal("an unparseable quantity should read as (0, false)")
	}
	tap(t, mgr, "Place order")
	cur = tree(t, mgr)
	if !hasTextContaining(cur, "Order between 1 and 12 gophers") {
		t.Fatal("the failed submit should reveal the Range error")
	}
	if hasTextContaining(cur, "order placed") {
		t.Fatal("an invalid form must not place an order")
	}

	// The typed getters feed the handler: Int for the quantity, Bool for the
	// checkbox seeded true.
	typeField(t, mgr, "how many?", "3")
	if !hasTextContaining(tree(t, mgr), "→ (3, true)") {
		t.Fatal("the parse caption should confirm the fix live")
	}
	tap(t, mgr, "Place order")
	if !hasTextContaining(tree(t, mgr), "order placed: quantity 3, gift-wrapped") {
		t.Fatal("the order should carry the parsed quantity and the ticked gift box")
	}

	// Unticking the box flows through Values.Bool on the next submit.
	toggleCheckbox(t, mgr, 0, false)
	tap(t, mgr, "Place order")
	cur = tree(t, mgr)
	if !hasTextContaining(cur, "order placed: quantity 3") {
		t.Fatal("the re-placed order should keep the quantity")
	}
	if hasTextContaining(cur, "gift-wrapped") {
		t.Fatal("the unticked box should read false")
	}

	// Initial is not re-applied between renders: a cleared field stays
	// cleared even though the spec still names its default.
	typeField(t, mgr, "how many?", "")
	if got := fieldByPlaceholder(t, mgr, "how many?").Props["value"]; got != "" {
		t.Fatalf("a cleared field should stay cleared, got %v", got)
	}

	// Reset is what re-reads the Initials — and clears the submitted flag, so
	// the fresh form opens quiet.
	tap(t, mgr, "Start over")
	cur = tree(t, mgr)
	if got := fieldByPlaceholder(t, mgr, "how many?").Props["value"]; got != "2" {
		t.Fatalf("Reset should re-seed the Initial, got %v", got)
	}
	if hasTextContaining(cur, "order placed") {
		t.Fatal("Start over should clear the confirmation caption")
	}
	if hasTextContaining(cur, "Order between 1 and 12 gophers") {
		t.Fatal("Reset should clear the submitted flag, hiding the errors again")
	}
	gift = findNode(cur, func(n *node) bool { return n.Type == "Checkbox" })
	if gift == nil || gift.Props["checked"] != true {
		t.Fatal("Reset should re-tick the gift box from its Initial")
	}
	assertNoConcerns(t)
}
