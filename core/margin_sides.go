package core

// The per-side and per-axis margin props: one outer gap, named, without
// going through a whole EdgeInsets.
//
// # What was here before
//
// Margin had exactly one prop — Margin(all), which writes all four sides —
// where padding had six (Padding, PaddingHorizontal, PaddingVertical and the
// four sides). So every single-sided gap in the repository went through the
// struct:
//
//	core.UseStyle(core.Style{Margin: core.EdgeInsets{Bottom: 8}})
//
// That is worse for margin than the same shape was for padding, and for a
// reason specific to this field. UseStyle replaces Margin outright rather
// than merging edge by edge (see Style.applyTo), so the caller restates
// every side they did not want to change — but a margin's other sides are
// almost always *zero*, which means the restatement is invisible: the
// EdgeInsets above silently clears any top, left or right margin a theme or
// an earlier prop had supplied, and looks like it only set the bottom.
// Padding's version of that mistake shows up as a squashed row; margin's
// shows up as two elements touching, three screens away.
//
// Both live workarounds were of exactly that shape — components/separator.go
// asking for an inset with Margin: EdgeInsets{Horizontal: s.Inset}, and
// examples/chat asking for a gap between bubbles with
// Margin: EdgeInsets{Bottom: 8}. Both are one prop now.
//
// # It is the same machinery, not a parallel one
//
// EdgeInsets is one type and every renderer resolves a margin side exactly
// as it resolves a padding side — "the explicit field if non-zero, otherwise
// the shorthand for that axis" — because both go through one function on
// each target (htmlout.EdgeCSS, edgeToCSS in wasm/grmob-runtime.js,
// parseEdges in GrMobStyle.kt and GrMobStyle.swift, each called for Padding
// and Margin alike).
//
// So the side props here settle their axis with settleHorizontal and
// settleVertical — the same two functions the padding sides use, unchanged —
// and inherit their whole argument: without the settle, MarginLeft(0) over a
// Horizontal 16 would resolve back to 16 and quietly do nothing, which is
// the one thing that would make these props order differently from every
// other StyleProp. See core/padding_sides.go for the settle's derivation and
// for why it is resolution-preserving; nothing about it is padding-specific,
// which is why no renderer changes for any of this.
//
// # The axis props mirror PaddingHorizontal, deliberately
//
// MarginHorizontal and MarginVertical write the two explicit sides *and* the
// shorthand field, which is what PaddingHorizontal does and for the reason
// stated there: writing only the shorthand could never override a side that
// was already explicit. They need no settle of their own — assigning both
// sides of an axis leaves nothing on that axis for a stale shorthand to
// resolve through, and the shorthand they write is their own.

// MarginTop sets the top margin alone, leaving the other three as they were.
// A zero clears whatever a theme or an earlier prop supplied.
func MarginTop(px int) StyleProp {
	return styleFunc(func(s *Style) {
		settleVertical(&s.Margin)
		s.Margin.Top = px
	})
}

// MarginBottom sets the bottom margin alone. A zero clears.
//
// This is the stacking prop: the gap under one item in a run that a parent's
// Gap does not describe, which is what examples/chat's message bubble wanted.
//
//	core.Box(core.MarginBottom(8), bubble)
func MarginBottom(px int) StyleProp {
	return styleFunc(func(s *Style) {
		settleVertical(&s.Margin)
		s.Margin.Bottom = px
	})
}

// MarginLeft sets the left margin alone. A zero clears.
func MarginLeft(px int) StyleProp {
	return styleFunc(func(s *Style) {
		settleHorizontal(&s.Margin)
		s.Margin.Left = px
	})
}

// MarginRight sets the right margin alone. A zero clears.
func MarginRight(px int) StyleProp {
	return styleFunc(func(s *Style) {
		settleHorizontal(&s.Margin)
		s.Margin.Right = px
	})
}

// MarginHorizontal sets the left and right margins.
//
// This is the inset prop: a rule that stops short of the screen edge, which
// is what components.Separator's Inset wanted.
//
//	core.Box(core.MarginHorizontal(16), rule)
//
// It writes the explicit sides as well as the shorthand, for the reason
// given on PaddingHorizontal.
func MarginHorizontal(px int) StyleProp {
	return styleFunc(func(s *Style) {
		s.Margin.Horizontal = px
		s.Margin.Left = px
		s.Margin.Right = px
	})
}

// MarginVertical sets the top and bottom margins. Writes the explicit sides
// as well as the shorthand, for the reason given on PaddingHorizontal.
func MarginVertical(px int) StyleProp {
	return styleFunc(func(s *Style) {
		s.Margin.Vertical = px
		s.Margin.Top = px
		s.Margin.Bottom = px
	})
}
