package core

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
)

// The prop's whole job is the mapping at the edges: clamp to CSS's range, and
// spell zero as the sentinel so it survives the three "non-zero" guards
// between here and a screen (omitzero, applyTo, each renderer).
func TestOpacityClampsAndSpellsZeroAsTheSentinel(t *testing.T) {
	for _, c := range []struct {
		in, stored, alpha float64
		declared          bool
	}{
		{0.4, 0.4, 0.4, true},
		{1, 1, 1, true},
		{0, OpacityClear, 0, true},
		{-3, OpacityClear, 0, true}, // below the range is the bottom of it
		{7, 1, 1, true},             // and above it the top
		{math.NaN(), 0, 1, false},   // not a number: unset, which is opaque
	} {
		var s Style
		Opacity(c.in).Apply(&s)
		if s.Opacity != c.stored {
			t.Errorf("Opacity(%g) stored %g, want %g", c.in, s.Opacity, c.stored)
		}
		alpha, declared := s.OpacityFactor()
		if alpha != c.alpha || declared != c.declared {
			t.Errorf("Opacity(%g).OpacityFactor() = (%g, %v), want (%g, %v)",
				c.in, alpha, declared, c.alpha, c.declared)
		}
	}
}

// An untouched Style is opaque and says nothing, which is what keeps every
// existing tree and export byte-identical.
func TestAnUnsetOpacityIsOpaqueAndUndeclared(t *testing.T) {
	if alpha, declared := (Style{}).OpacityFactor(); alpha != 1 || declared {
		t.Errorf("zero Style: OpacityFactor() = (%g, %v), want (1, false)", alpha, declared)
	}
}

// A hand-built Style can hold what the prop never stores. The reading clamps
// it, so a renderer is never handed an alpha it has to range-check.
func TestOpacityFactorClampsAHandBuiltStyle(t *testing.T) {
	for _, c := range []struct{ field, want float64 }{
		{2.5, 1}, {-0.5, 0}, {-7, 0},
	} {
		if alpha, declared := (Style{Opacity: c.field}).OpacityFactor(); alpha != c.want || !declared {
			t.Errorf("Style{Opacity: %g}: (%g, %v), want (%g, true)", c.field, alpha, declared, c.want)
		}
	}
}

// Both ends of the range are stored non-zero, so a merge can take a node to
// transparent AND back to opaque: the clear that Rotate's merge cannot do.
func TestUseStyleMergesOpacityToBothEnds(t *testing.T) {
	base := Style{Opacity: 0.5}

	var clear Style
	Opacity(0).Apply(&clear)
	UseStyle(clear).Apply(&base)
	if alpha, _ := base.OpacityFactor(); alpha != 0 {
		t.Fatalf("merging Opacity(0) left an alpha of %g", alpha)
	}

	var opaque Style
	Opacity(1).Apply(&opaque)
	UseStyle(opaque).Apply(&base)
	if alpha, _ := base.OpacityFactor(); alpha != 1 {
		t.Errorf("merging Opacity(1) left an alpha of %g", alpha)
	}

	// And a style that says nothing layers nothing.
	base = Style{Opacity: 0.5}
	UseStyle(Style{FontSize: 12}).Apply(&base)
	if base.Opacity != 0.5 {
		t.Errorf("an unrelated merge changed Opacity to %g", base.Opacity)
	}
}

// The wire is where a plain zero would have been lost first: omitzero drops
// it. The sentinel crosses, and an unset field still costs no bytes.
func TestOpacityZeroCrossesTheWireAndUnsetDoesNot(t *testing.T) {
	var s Style
	Opacity(0).Apply(&s)
	raw, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(raw); got != `{"Opacity":-1}` {
		t.Errorf("Opacity(0) serialised as %s, want {\"Opacity\":-1}", got)
	}

	raw, err = json.Marshal(Style{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "Opacity") {
		t.Errorf("an unset Opacity is on the wire: %s", raw)
	}
}

// The sentinel only works while no author can mean it. CSS clamps opacity to
// [0, 1], so a negative is out of range, and it is ShrinkNone's number on
// purpose: one rule in each of the four runtimes.
func TestTheOpacitySentinelIsOutsideTheValidRange(t *testing.T) {
	if OpacityClear >= 0 {
		t.Errorf("OpacityClear is %v, an alpha an author could ask for", OpacityClear)
	}
	if OpacityClear != ShrinkNone {
		t.Errorf("OpacityClear (%v) and ShrinkNone (%v) have drifted apart; every "+
			"runtime reads -1 as \"this property's zero\"", OpacityClear, ShrinkNone)
	}
}
