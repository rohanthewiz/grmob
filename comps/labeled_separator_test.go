package comps

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// Two hidden rules growing equally, and a label between them that is read.
func TestLabeledSeparatorCentresAReadableLabel(t *testing.T) {
	_, n := renderDebug(t, LabeledSeparator{Label: "or"})
	if n.Type != "Row" || len(n.Children) != 3 {
		t.Fatalf("root = %q with %d children, want rule, label, rule", n.Type, len(n.Children))
	}
	left, label, right := n.Children[0], n.Children[1], n.Children[2]
	for _, r := range []*core.Node{left, right} {
		if r.Style.FlexGrow != 1 || !r.Style.AccessibilityHidden {
			t.Errorf("rule grow/hidden = %v/%v, want equal growth and hidden", r.Style.FlexGrow, r.Style.AccessibilityHidden)
		}
		if r.Style.Background != core.DefaultTheme.Colors.BorderColor() {
			t.Errorf("rule tint = %q, want the theme's Border", r.Style.Background)
		}
	}
	if label.Props["content"] != "or" || label.Style.AccessibilityHidden {
		t.Error("the label should be drawn and read")
	}
	if label.Style.FlexShrink != core.ShrinkNone {
		t.Error("the label should never be squeezed; the rules give way")
	}
}

func TestLabeledSeparatorWithoutLabelIsOneRule(t *testing.T) {
	_, n := renderDebug(t, LabeledSeparator{})
	if len(n.Children) != 1 {
		t.Fatalf("children = %d, want one unbroken rule", len(n.Children))
	}
}
