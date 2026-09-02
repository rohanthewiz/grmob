package core

import "testing"

// The renderers resolve each inset side as "explicit if non-zero, else the
// axis shorthand". A prop that wrote only the shorthand therefore could not
// override a side a theme had already set — PaddingHorizontal(0) after the
// theme's Column style (Left/Right 16) left the 16 in place, which on a phone
// cost every nested Column 32dp and wrapped an 11pt tab label mid-word. These
// pin that the shorthand props write the explicit sides too, so ordinary
// last-prop-wins ordering applies to them.
func TestPaddingShorthandOverridesExplicitSides(t *testing.T) {
	theme := Style{Padding: EdgeInsets{Top: 12, Bottom: 12, Left: 16, Right: 16}}

	// Zero must clear: every side resolves to 0 in every renderer.
	s := theme
	PaddingHorizontal(0).Apply(&s)
	PaddingVertical(0).Apply(&s)
	if s.Padding != (EdgeInsets{}) {
		t.Fatalf("PaddingHorizontal(0)+PaddingVertical(0) over a theme padding = %+v, want all zero", s.Padding)
	}

	// Non-zero must win over the theme's explicit sides, not lose to them.
	s = theme
	PaddingHorizontal(24).Apply(&s)
	if s.Padding.Left != 24 || s.Padding.Right != 24 {
		t.Fatalf("PaddingHorizontal(24) over Left/Right 16 = %+v, want 24 on both sides", s.Padding)
	}
	if s.Padding.Top != 12 || s.Padding.Bottom != 12 {
		t.Fatalf("PaddingHorizontal touched the vertical sides: %+v", s.Padding)
	}

	s = theme
	PaddingVertical(4).Apply(&s)
	if s.Padding.Top != 4 || s.Padding.Bottom != 4 || s.Padding.Left != 16 || s.Padding.Right != 16 {
		t.Fatalf("PaddingVertical(4) over a theme padding = %+v, want Top/Bottom 4, sides untouched", s.Padding)
	}
}
