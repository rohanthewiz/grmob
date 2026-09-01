package core

import "testing"

// TestSizeConstraintPropsWriteTheirOwnField pins each of the four size
// constraints to the Style field its name promises.
//
// MaxHeight did not: it wrote Style.MaxWidth (Style had no MaxHeight field at
// all), so `MaxHeight("200px")` silently constrained the wrong axis — and any
// style that also set MaxWidth had one of the two clobbered depending on
// argument order. A table rather than four asserts, so the next size prop is
// one line.
func TestSizeConstraintPropsWriteTheirOwnField(t *testing.T) {
	cases := []struct {
		name  string
		prop  StyleProp
		field func(*Style) string
	}{
		{"Width", Width("1px"), func(s *Style) string { return s.Width }},
		{"Height", Height("2px"), func(s *Style) string { return s.Height }},
		{"MinWidth", MinWidth("3px"), func(s *Style) string { return s.MinWidth }},
		{"MinHeight", MinHeight("4px"), func(s *Style) string { return s.MinHeight }},
		{"MaxWidth", MaxWidth("5px"), func(s *Style) string { return s.MaxWidth }},
		{"MaxHeight", MaxHeight("6px"), func(s *Style) string { return s.MaxHeight }},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var s Style
			c.prop.Apply(&s)
			if got := c.field(&s); got == "" {
				t.Fatalf("%s() left Style.%s empty — the prop writes some other field", c.name, c.name)
			}
		})
	}

	// The pair that actually collided: both must survive together, in either
	// order.
	var s Style
	MaxWidth("100px").Apply(&s)
	MaxHeight("200px").Apply(&s)
	if s.MaxWidth != "100px" || s.MaxHeight != "200px" {
		t.Errorf("MaxWidth/MaxHeight collided: MaxWidth=%q MaxHeight=%q", s.MaxWidth, s.MaxHeight)
	}
}
