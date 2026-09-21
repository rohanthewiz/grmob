package comps

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// checkout is the fixture: a plain step, an optional one, and the last.
func checkout(blockAddress, blockNote bool) []WizardStep {
	return []WizardStep{
		{Title: "Address", Body: core.Text("address fields"), Blocked: blockAddress},
		{Title: "Gift note", Body: core.Text("note field"), Optional: true, Blocked: blockNote},
		{Title: "Review", Body: core.Text("summary")},
	}
}

// wizardButton finds a footer button by its label.
func wizardButton(n *core.Node, label string) *core.Node {
	for _, b := range buttonsOf(n) {
		if b.Props["label"] == label || b.Props["content"] == label || b.Style.AccessibilityLabel == label {
			return b
		}
	}
	return nil
}

// isDisabled reads the platform-disabled flag comps.Button sets.
func isDisabled(b *core.Node) bool { return b.Style.Disabled }

// The parts, in order: a named group holding the indicator, the status line,
// the title as a heading, the current body alone, and the footer.
func TestWizardStructureAndAccessibility(t *testing.T) {
	_, n := renderDebug(t, Wizard{
		Label: "Checkout", Steps: checkout(false, false), Current: 1,
		OnChange: func(int) {}, OnFinish: func() {},
	})

	if n.Style.AccessibilityRole != core.RoleGroup || n.Style.AccessibilityLabel != "Checkout" {
		t.Errorf("role %q label %q", n.Style.AccessibilityRole, n.Style.AccessibilityLabel)
	}
	strip := findFirst(n, func(n *core.Node) bool { return n.Style.AccessibilityRole == core.RoleNavigation })
	if strip == nil || strip.Style.AccessibilityLabel != "Checkout, step 2 of 3: Gift note" {
		t.Errorf("the indicator should be a navigation named by position, got %+v", strip)
	}
	status := findText(n, "Step 2 of 3, optional")
	if status == nil || status.Style.AccessibilityRole != core.RoleStatus {
		t.Error("the position line should be a status reading \"Step 2 of 3, optional\"")
	}
	title := findFirst(n, func(n *core.Node) bool {
		return n.Type == "Text" && n.Props["content"] == "Gift note" && n.Style.AccessibilityRole == core.RoleHeading
	})
	if title == nil || title.Style.AccessibilityHeadingLevel != headingLevelSection {
		t.Error("the step's title should be a section heading")
	}

	if findText(n, "note field") == nil {
		t.Error("the current body is missing")
	}
	if findText(n, "address fields") != nil || findText(n, "summary") != nil {
		t.Error("only the current step's body is rendered")
	}
}

// The body is keyed by step, so moving on replaces it rather than morphing
// one form into the next.
func TestWizardBodyIsKeyedByStep(t *testing.T) {
	for cur, want := range []string{"wizard-step-0", "wizard-step-1", "wizard-step-2"} {
		_, n := renderDebug(t, Wizard{Steps: checkout(false, false), Current: cur, OnChange: func(int) {}, OnFinish: func() {}})
		if findFirst(n, func(n *core.Node) bool { return n.Key == want }) == nil {
			t.Errorf("step %d: no node keyed %q", cur, want)
		}
	}
}

// Which buttons each step offers, and what they say.
func TestWizardFooterPerStep(t *testing.T) {
	cases := []struct {
		name                   string
		current                int
		blockAddress, blockNot bool
		back                   bool
		forward                string
		forwardDisabled        bool
	}{
		{"first step: no Back at all", 0, false, false, false, "Next", false},
		{"a blocked step disables Next", 0, true, false, false, "Next", true},
		{"a middle step has both", 1, false, false, true, "Next", false},
		{"a blocked optional step offers Skip, enabled", 1, false, true, true, "Skip", false},
		{"the last step finishes", 2, false, false, true, "Finish", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, n := renderDebug(t, Wizard{
				Steps: checkout(c.blockAddress, c.blockNot), Current: c.current,
				OnChange: func(int) {}, OnFinish: func() {},
			})
			if got := wizardButton(n, "Back") != nil; got != c.back {
				t.Errorf("Back present = %v, want %v", got, c.back)
			}
			fwd := wizardButton(n, c.forward)
			if fwd == nil {
				t.Fatalf("no %q button", c.forward)
			}
			if isDisabled(fwd) != c.forwardDisabled {
				t.Errorf("%q disabled = %v, want %v", c.forward, isDisabled(fwd), c.forwardDisabled)
			}
		})
	}
}

// Each move reports exactly one index, to the right callback.
func TestWizardMovesReportOneChange(t *testing.T) {
	var moves []int
	finished := 0
	w := Wizard{
		Steps: checkout(false, false), Current: 1,
		OnChange: func(i int) { moves = append(moves, i) },
		OnFinish: func() { finished++ },
	}

	ctx, n := renderDebug(t, w)
	ctx.TriggerCallback(wizardButton(n, "Next").Props["onClick"].(string))
	ctx.TriggerCallback(wizardButton(n, "Back").Props["onClick"].(string))
	if len(moves) != 2 || moves[0] != 2 || moves[1] != 0 {
		t.Errorf("Next then Back from step 1 reported %v, want [2 0]", moves)
	}

	// A done step in the indicator goes through the same OnChange.
	done := findFirst(n, func(n *core.Node) bool { return n.Style.AccessibilityLabel == "Step 1: Address, done" })
	if done == nil {
		t.Fatal("no done step in the indicator")
	}
	ctx.TriggerCallback(done.Props["onClick"].(string))
	if len(moves) != 3 || moves[2] != 0 {
		t.Errorf("tapping done step 1 reported %v", moves)
	}

	w.Current = 2
	ctx, n = renderDebug(t, w)
	ctx.TriggerCallback(wizardButton(n, "Finish").Props["onClick"].(string))
	if finished != 1 || len(moves) != 3 {
		t.Errorf("Finish: finished %d, moves %v; it should call OnFinish alone", finished, moves)
	}
}

// Finish with no OnFinish is disabled, and "Skip" is never the last step's
// word even when that step is optional and blocked.
func TestWizardLastStepEdges(t *testing.T) {
	steps := []WizardStep{{Title: "One"}, {Title: "Two", Optional: true, Blocked: true}}

	_, n := renderDebug(t, Wizard{Steps: steps, Current: 1, OnChange: func(int) {}})
	if f := wizardButton(n, "Finish"); f == nil || !isDisabled(f) {
		t.Error("Finish with no OnFinish should be drawn disabled")
	}

	_, n = renderDebug(t, Wizard{Steps: steps, Current: 1, OnChange: func(int) {}, OnFinish: func() {}})
	if wizardButton(n, "Skip") != nil {
		t.Error("the last step should not offer Skip")
	}
	if f := wizardButton(n, "Finish"); f == nil || isDisabled(f) {
		t.Error("an optional last step should finish even while blocked")
	}
}

// Current is clamped, once, so the body and the buttons agree.
func TestWizardClampsCurrent(t *testing.T) {
	_, n := renderDebug(t, Wizard{Steps: checkout(false, false), Current: 9, OnChange: func(int) {}, OnFinish: func() {}})
	if findText(n, "summary") == nil || wizardButton(n, "Finish") == nil {
		t.Error("Current past the end should show the last step and Finish")
	}
	_, n = renderDebug(t, Wizard{Steps: checkout(false, false), Current: -3, OnChange: func(int) {}, OnFinish: func() {}})
	if findText(n, "address fields") == nil || wizardButton(n, "Back") != nil {
		t.Error("a negative Current should show the first step with no Back")
	}
}

// DetachFooter leaves the buttons out of the column, and Footer() builds the
// same buttons for the caller to place.
func TestWizardDetachedFooter(t *testing.T) {
	var moves []int
	w := Wizard{
		Steps: checkout(false, false), Current: 1, DetachFooter: true,
		OnChange: func(i int) { moves = append(moves, i) }, OnFinish: func() {},
	}
	_, n := renderDebug(t, w)
	if len(buttonsOf(n)) != 0 {
		t.Errorf("a detached wizard drew %d buttons", len(buttonsOf(n)))
	}

	ctx, f := renderDebug(t, w.Footer())
	if wizardButton(f, "Back") == nil || wizardButton(f, "Next") == nil {
		t.Fatal("Footer() should hold Back and Next")
	}
	ctx.TriggerCallback(wizardButton(f, "Next").Props["onClick"].(string))
	if len(moves) != 1 || moves[0] != 2 {
		t.Errorf("the detached Next reported %v", moves)
	}
}

func TestWizardLabelsAndPositionAreTheCallers(t *testing.T) {
	_, n := renderDebug(t, Wizard{
		Steps: checkout(false, true), Current: 1,
		OnChange: func(int) {}, OnFinish: func() {},
		BackLabel: "Zurück", SkipLabel: "Überspringen",
		PositionLabel: func(cur, total int, optional bool) string {
			if !optional || cur != 1 || total != 3 {
				t.Errorf("PositionLabel(%d, %d, %v)", cur, total, optional)
			}
			return "Schritt 2 von 3"
		},
	})
	if wizardButton(n, "Zurück") == nil || wizardButton(n, "Überspringen") == nil {
		t.Error("the caller's button words were not used")
	}
	if findText(n, "Schritt 2 von 3") == nil {
		t.Error("the caller's position line was not used")
	}
}

func TestWizardConcerns(t *testing.T) {
	cases := []struct {
		name string
		w    Wizard
		want string
	}{
		{"no steps", Wizard{OnChange: func(int) {}}, ConcernWizardNoSteps},
		{"steps and no OnChange", Wizard{Steps: checkout(false, false), OnFinish: func() {}}, ConcernWizardInert},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			core.SetDebugMode(true)
			core.ClearConcerns()
			t.Cleanup(func() { core.SetDebugMode(false); core.ClearConcerns() })
			ctx := core.NewContext()
			ctx.BeginRenderPass()
			c.w.Render(ctx)
			ctx.EndRenderPass()
			if dump := core.DumpConcerns(); !strings.Contains(dump, c.want) {
				t.Errorf("want %s, got:\n%s", c.want, dump)
			}
		})
	}

	// One step needs no OnChange: there is nowhere to move.
	renderDebug(t, Wizard{Steps: []WizardStep{{Title: "Only"}}, OnFinish: func() {}})
}

// Next tapped with no OnChange is a reported bug and not a panic.
func TestWizardNilOnChangeDoesNotPanic(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := Wizard{Steps: checkout(false, false)}.Render(ctx)
	ctx.EndRenderPass()
	ctx.TriggerCallback(wizardButton(n, "Next").Props["onClick"].(string))
}

// Moving between steps whose bodies are plain views shifts no hook slot: the
// wizard holds none, so a state declared after it keeps its value.
func TestWizardStepChangeMovesNoHookSlot(t *testing.T) {
	core.SetDebugMode(true)
	core.ClearConcerns()
	t.Cleanup(func() { core.SetDebugMode(false); core.ClearConcerns() })

	ctx := core.NewContext()
	pass := func(cur int, initial string) string {
		ctx.BeginRenderPass()
		ctx.Reset()
		defer ctx.EndRenderPass()
		Wizard{Steps: checkout(false, false), Current: cur, OnChange: func(int) {}, OnFinish: func() {}}.Render(ctx)
		after := core.NewState(ctx, initial)
		return after.Get()
	}
	pass(0, "first")
	for cur := 1; cur <= 2; cur++ {
		if got := pass(cur, "later"); got != "first" {
			t.Errorf("step %d: the slot after the wizard moved: %q", cur, got)
		}
	}
	if dump := core.DumpConcerns(); dump != "" {
		t.Errorf("concerns raised:\n%s", dump)
	}
}
