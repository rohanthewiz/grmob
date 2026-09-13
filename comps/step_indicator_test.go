package comps

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

var checkoutSteps = []string{"Account", "Address", "Payment", "Review"}

// stripOf asserts the widget renders its Scroll strip directly, with no
// wrapper, and returns it.
func stripOf(t *testing.T, root *core.Node) *core.Node {
	t.Helper()
	if root.Type != "Scroll" {
		t.Fatalf("root = %q, want the Scroll strip itself: the host pages no longer need a wrapper", root.Type)
	}
	return root
}

// stepCells returns the named step cells in order, skipping the rules.
func stepCells(n *core.Node) []*core.Node {
	var out []*core.Node
	for _, c := range n.Children {
		if c.Type == "Row" {
			out = append(out, c)
		}
	}
	return out
}

func TestStepIndicatorIsAHorizontalScrollNamedByPosition(t *testing.T) {
	_, root := renderDebug(t, StepIndicator{Steps: checkoutSteps, Current: 1})
	n := stripOf(t, root)

	if n.Type != "Scroll" || n.Style.FlexDirection != core.FlexRow {
		t.Fatalf("root = %q direction %q, want a horizontal Scroll: long flows scroll", n.Type, n.Style.FlexDirection)
	}
	if n.Style.AccessibilityRole != core.RoleGroup {
		t.Errorf("no OnTap: role = %q, want group — a picture of progress is not navigation", n.Style.AccessibilityRole)
	}
	if n.Style.AccessibilityLabel != "Step 2 of 4: Address" {
		t.Errorf("strip name = %q", n.Style.AccessibilityLabel)
	}

	cells := stepCells(n)
	if len(cells) != 4 || len(n.Children) != 7 {
		t.Fatalf("want 4 steps joined by 3 rules, got %d cells in %d children", len(cells), len(n.Children))
	}
	for i, want := range []string{"Step 1: Account, done", "Step 2: Address, current", "Step 3: Payment", "Step 4: Review"} {
		if cells[i].Style.AccessibilityLabel != want {
			t.Errorf("cell %d name = %q, want %q", i, cells[i].Style.AccessibilityLabel, want)
		}
	}
}

func TestStepIndicatorDrawsDoneCurrentAndUpcoming(t *testing.T) {
	_, root := renderDebug(t, StepIndicator{Steps: checkoutSteps, Current: 1})
	n := stripOf(t, root)
	th := core.DefaultTheme
	cells := stepCells(n)

	done, current, upcoming := cells[0].Children[0], cells[1].Children[0], cells[2].Children[0]
	if done.Style.Background != th.Colors.SuccessColor() || findText(done, "✓") == nil {
		t.Error("a done step is a ticked Success disc")
	}
	if current.Style.Background != th.Colors.Primary || findText(current, "2") == nil {
		t.Error("the current step is a numbered Primary disc")
	}
	if upcoming.Style.Background != "" || upcoming.Style.BorderWidth != 2 || findText(upcoming, "3") == nil {
		t.Error("an upcoming step is a numbered ring with no fill")
	}
	for _, c := range []*core.Node{done, current, upcoming} {
		if !c.Style.AccessibilityHidden {
			t.Error("circles are drawn; the cell's name is what is read")
		}
	}

	if r := n.Children[1]; r.Style.Background != th.Colors.SuccessColor() {
		t.Errorf("the rule after a done step = %q, want Success", r.Style.Background)
	}
	if r := n.Children[3]; r.Style.Background != th.Colors.BorderColor() {
		t.Errorf("the rule after the current step = %q, want Border", r.Style.Background)
	}
	if label := findText(cells[1], "Address"); label == nil || label.Style.FontWeight != core.Bold {
		t.Error("the current step's label is bold")
	}
}

func TestStepIndicatorOnlyDoneStepsAreTappable(t *testing.T) {
	var got []int
	ctx, root := renderDebug(t, StepIndicator{Steps: checkoutSteps, Current: 2, Label: "Checkout",
		OnTap: func(i int) { got = append(got, i) }})
	n := stripOf(t, root)

	if n.Style.AccessibilityRole != core.RoleNavigation {
		t.Errorf("with OnTap, role = %q, want navigation", n.Style.AccessibilityRole)
	}
	if n.Style.AccessibilityLabel != "Checkout, step 3 of 4: Payment" {
		t.Errorf("Label prefix: name = %q", n.Style.AccessibilityLabel)
	}

	cells := stepCells(n)
	for i, c := range cells {
		_, tappable := c.Props["onClick"]
		isButton := c.Style.AccessibilityRole == core.RoleButton
		if want := i < 2; tappable != want || isButton != want {
			t.Errorf("step %d: tappable=%v button=%v, want %v", i, tappable, isButton, want)
		}
	}
	ctx.TriggerCallback(cells[0].Props["onClick"].(string))
	if len(got) != 1 || got[0] != 0 {
		t.Errorf("taps = %v, want [0]", got)
	}
}

func TestStepIndicatorClampsCurrent(t *testing.T) {
	_, root := renderDebug(t, StepIndicator{Steps: checkoutSteps, Current: 9})
	n := stripOf(t, root)
	if n.Style.AccessibilityLabel != "Step 4 of 4: Review" {
		t.Errorf("an index past the end clamps to the last step, got %q", n.Style.AccessibilityLabel)
	}
	_, root = renderDebug(t, StepIndicator{})
	n = stripOf(t, root)
	if len(n.Children) != 0 || n.Style.AccessibilityLabel != "" {
		t.Error("no steps: an empty strip with no name")
	}
}
