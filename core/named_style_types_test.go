package core

import "testing"

// The named flex-container types apply themselves, so the type-conversion
// spelling is a real prop and not a value the container dispatch drops.
func TestNamedFlexTypesAreStyleProps(t *testing.T) {
	ctx := NewContext()
	n := Render(ctx, Column(
		AlignItems(AlignItemsCenter),
		JustifyContent(JustifyBetween),
		FlexDirection(FlexRow),
	))
	if n.Style.AlignItems != AlignItemsCenter {
		t.Errorf("AlignItems = %q, want %q", n.Style.AlignItems, AlignItemsCenter)
	}
	if n.Style.JustifyContent != JustifyBetween {
		t.Errorf("JustifyContent = %q, want %q", n.Style.JustifyContent, JustifyBetween)
	}
	if n.Style.FlexDirection != FlexRow {
		t.Errorf("FlexDirection = %q, want %q", n.Style.FlexDirection, FlexRow)
	}
}
