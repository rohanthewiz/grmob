package core

// Corners is a radius per corner, in points, in CSS's order: top-left,
// top-right, bottom-right, bottom-left.
//
// # Why per corner, and why physical
//
// Style had one BorderRadius, and two widgets documented what that cost:
//
//	DateRangePicker   a range endpoint is rounded on the outside and square
//	                  where it meets the band, so a rounded endpoint beside a
//	                  square band left a notch at every join
//	MessageBubble     a tail is one corner sharper than the other three,
//	                  pointing at the sender
//
// Both are shapes, not decorations: one box with four radii, which every
// target draws natively (CSS border-radius with four values, Compose's
// AbsoluteRoundedCornerShape, SwiftUI's UnevenRoundedRectangle).
//
// The corners are physical, as CSS's border-radius is, and not start/end.
// Both callers mean a side of the screen: a band runs left to right through
// the week, and a bubble's tail points at where its sender's bubbles line up.
// A caller that wants a logical corner mirrors the value itself for a
// right-to-left layout. SwiftUI names its corners by leading and trailing, so
// that renderer swaps them back under a right-to-left layout direction.
type Corners struct {
	TopLeft     float64 `json:",omitzero"`
	TopRight    float64 `json:",omitzero"`
	BottomRight float64 `json:",omitzero"`
	BottomLeft  float64 `json:",omitzero"`
}

// Set reports whether any corner is non-zero. A zero Corners is "not stated"
// and leaves BorderRadius in charge; four square corners are BorderRadius(0),
// which already says so.
func (c Corners) Set() bool {
	return c.TopLeft != 0 || c.TopRight != 0 || c.BottomRight != 0 || c.BottomLeft != 0
}

// Radii returns the four radii a renderer draws, with BorderRadius standing
// in for all four when no corner is stated. It is the one statement of the
// precedence, for the Go-side renderer (htmlout) and as the reference the
// others transliterate.
func (s *Style) Radii() Corners {
	if s == nil {
		return Corners{}
	}
	if s.Corners.Set() {
		return s.Corners
	}
	r := s.BorderRadius
	return Corners{TopLeft: r, TopRight: r, BottomRight: r, BottomLeft: r}
}

// CornerRadii gives each corner its own radius, in CSS's order:
//
//	core.CornerRadii(18, 18, 4, 18)   // a bubble whose tail is bottom-right
//	core.CornerRadii(12, 0, 0, 12)    // a range's start, square toward the band
//
// A zero is a square corner. It replaces BorderRadius while any corner is
// non-zero, and a later BorderRadius replaces it in turn, so the last of the
// two in an argument list is the shape drawn: the rule every other style prop
// follows.
func CornerRadii(topLeft, topRight, bottomRight, bottomLeft float64) StyleProp {
	return styleFunc(func(s *Style) {
		s.Corners = Corners{TopLeft: topLeft, TopRight: topRight, BottomRight: bottomRight, BottomLeft: bottomLeft}
	})
}
