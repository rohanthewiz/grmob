package core

import "testing"

// core.FlexShrink(0) has to survive every guard between the prop and a renderer.
//
// # The mutation that could not break
//
// This field's whole history is a break-test that changed nothing. Flipping a
// fixture's FlexShrink from 1 to 0 moved no pixel on any target, which is how a
// declaration nobody can write announces itself: Style.Merge, htmlout.Export and
// the WASM runtime each guarded on `FlexShrink != 0` — correct for every other
// number in a Style and wrong for the one whose CSS initial value is not zero.
//
// So the cases below are the three places the zero used to be swallowed, and
// each is written as the assertion the old code would have failed.
func TestAZeroShrinkFactorSurvivesTheProp(t *testing.T) {
	var s Style
	FlexShrink(0).Apply(&s)

	if s.FlexShrink == 0 {
		t.Fatalf("core.FlexShrink(0) left the field at zero, which every guard in " +
			"this framework reads as \"nothing was set\". The prop is the only door " +
			"into the field and mapping the zero here is the whole mechanism.")
	}
	if s.FlexShrink != ShrinkNone {
		t.Errorf("core.FlexShrink(0) stored %v, want core.ShrinkNone (%v). The value "+
			"crosses into three other runtimes as a bare number in JSON, and each of "+
			"them matches on this one.", s.FlexShrink, ShrinkNone)
	}

	factor, declared := s.ShrinkFactor()
	if !declared || factor != 0 {
		t.Errorf("ShrinkFactor() on a FlexShrink(0) style is (%v, %v), want (0, true)",
			factor, declared)
	}
}

// The other two answers, and the distinction the two returns exist for.
func TestShrinkFactorSeparatesUnsetFromZero(t *testing.T) {
	if factor, declared := (Style{}).ShrinkFactor(); declared || factor != 0 {
		t.Errorf("an untouched Style reports (%v, %v), want (0, false) — a renderer "+
			"writing flex-shrink:0 for every node that never mentioned one would pin "+
			"the whole tree", factor, declared)
	}

	var s Style
	FlexShrink(2).Apply(&s)
	if factor, declared := s.ShrinkFactor(); !declared || factor != 2 {
		t.Errorf("FlexShrink(2) reports (%v, %v), want (2, true)", factor, declared)
	}
}

// A shrink factor of zero overrides a base, which is the second guard.
//
// This is the one that makes the prop usable at all from a widget: a component
// default or a theme base carrying a shrink factor would otherwise win over a
// caller's explicit "do not shrink", because Style.Merge copies only non-zero
// fields and the caller's value was a zero.
func TestAZeroShrinkFactorOverridesABase(t *testing.T) {
	base := Style{FlexShrink: 1}
	var caller Style
	FlexShrink(0).Apply(&caller)

	target := base
	// applyTo is what a widget's base and a caller's props go through: the
	// theme's Style first, then each StyleProp over it.
	caller.applyTo(&target)

	factor, declared := target.ShrinkFactor()
	if !declared || factor != 0 {
		t.Errorf("merging FlexShrink(0) over a base with FlexShrink 1 gives (%v, %v), "+
			"want (0, true) — the caller said do not shrink and the base won",
			factor, declared)
	}
}
