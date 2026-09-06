package core

// The per-side padding props: one inset, named, without going through a whole
// EdgeInsets.
//
// # What was here before
//
// The set was Padding (all four), PaddingHorizontal, PaddingVertical and
// PaddingTop — three of the four sides had no prop at all. A screen that
// wanted only a left inset wrote the struct:
//
//	core.UseStyle(core.Style{Padding: core.EdgeInsets{Left: 16 * depth, Right: 16, Top: 4, Bottom: 4}})
//
// which is not merely longer. UseStyle replaces Padding outright rather than
// merging edge by edge (see Style.applyTo), so the caller has to restate the
// three sides they did not want to change, in the theme's numbers, and those
// numbers are now a copy that a theme edit will not reach. The tutorial's
// tree-indent helper says exactly that in its own comment, and it is the
// oldest live instance.
//
// # The shorthand has to be dissolved, not ignored
//
// EdgeInsets carries six fields: the four sides plus a Horizontal/Vertical
// pair. Every renderer resolves a side as "the explicit field if non-zero,
// otherwise the shorthand for that axis" (htmlout.EdgeCSS, edgeToCSS in
// wasm/grmob-runtime.js, parseEdges in GrMobStyle.kt and GrMobStyle.swift).
//
// A side prop that only assigned its own field would inherit that rule's one
// lossy edge: a zero side is indistinguishable from an unset one, so
// PaddingLeft(0) over a theme carrying Horizontal 16 would resolve back to 16
// and quietly do nothing. That is the same trap PaddingHorizontal was fixed
// for — a prop that cannot clear what came before it does not have
// last-one-wins ordering, and every other StyleProp does.
//
// So each side prop first *settles* its axis: it pushes the shorthand into
// whichever of the two sides is not already explicit, then clears it.
//
//	           before                    after settleHorizontal
//	{Horizontal: 16}              ->  {Left: 16, Right: 16}
//	{Horizontal: 16, Left: 24}    ->  {Left: 24, Right: 16}
//	{Left: 16, Right: 16}         ->  unchanged (no shorthand to dissolve)
//
// Settling is *resolution-preserving* by construction: every side resolves to
// the same number before and after, because the only writes it makes are into
// sides that were taking the shorthand anyway. Nothing in any renderer
// changes, and TestSettlingAnAxisPreservesEveryResolvedSide pins that from
// the outside by comparing htmlout.EdgeCSS across the transformation.
//
// With the axis settled, the prop's own assignment lands on a field that is
// now the only thing describing that side — so PaddingLeft(0) means zero, on
// all four targets, and the props order like every other StyleProp.
//
// # PaddingTop moved here and changed
//
// It was the one side prop that existed, and it wrote nothing but
// s.Padding.Top — so PaddingTop(0) after PaddingVertical(8) left the 8 in
// place. It settles its axis now, which makes the four consistent; the only
// behavior that differs is the case that used to silently fail.

// settleHorizontal dissolves the Horizontal shorthand into the Left/Right
// fields it was standing in for, then clears it. See the file comment for why
// this is resolution-preserving.
//
// Two details the break-tests separated. The zero check is a fast path and
// not a guard — with no shorthand to dissolve, the body writes zeros into
// sides that are already zero and clears a field already clear, so removing
// it changes nothing and no test fails. The "only if the side is unset"
// checks are the opposite: without them, settling promotes the shorthand over
// a side that had already superseded it, and the damage lands on the side the
// prop is *not* about to write, where the prop's own assignment cannot mask
// it. TestSettlingAnAxisPreservesEveryResolvedSide carries both orientations
// for exactly that reason.
func settleHorizontal(e *EdgeInsets) {
	if e.Horizontal == 0 {
		return
	}
	if e.Left == 0 {
		e.Left = e.Horizontal
	}
	if e.Right == 0 {
		e.Right = e.Horizontal
	}
	e.Horizontal = 0
}

// settleVertical is settleHorizontal for the Top/Bottom axis.
func settleVertical(e *EdgeInsets) {
	if e.Vertical == 0 {
		return
	}
	if e.Top == 0 {
		e.Top = e.Vertical
	}
	if e.Bottom == 0 {
		e.Bottom = e.Vertical
	}
	e.Vertical = 0
}

// PaddingTop sets the top inset alone, leaving the other three as they were.
// A zero clears whatever the theme or an earlier prop supplied.
func PaddingTop(px int) StyleProp {
	return styleFunc(func(s *Style) {
		settleVertical(&s.Padding)
		s.Padding.Top = px
	})
}

// PaddingBottom sets the bottom inset alone. A zero clears.
func PaddingBottom(px int) StyleProp {
	return styleFunc(func(s *Style) {
		settleVertical(&s.Padding)
		s.Padding.Bottom = px
	})
}

// PaddingLeft sets the left inset alone. A zero clears.
//
// This is the indent prop: a nested row states its own depth without having
// to restate the three sides its theme container already got right.
//
//	core.Row(core.PaddingLeft(16*depth), ...)
func PaddingLeft(px int) StyleProp {
	return styleFunc(func(s *Style) {
		settleHorizontal(&s.Padding)
		s.Padding.Left = px
	})
}

// PaddingRight sets the right inset alone. A zero clears.
func PaddingRight(px int) StyleProp {
	return styleFunc(func(s *Style) {
		settleHorizontal(&s.Padding)
		s.Padding.Right = px
	})
}
