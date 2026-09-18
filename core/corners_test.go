package core

import (
	"encoding/json"
	"strings"
	"testing"
)

func styleOf(props ...StyleProp) Style {
	var s Style
	for _, p := range props {
		p.Apply(&s)
	}
	return s
}

// Radii states the precedence every renderer transliterates: named corners
// replace the one radius, and without them the one radius is all four.
func TestRadiiPrecedence(t *testing.T) {
	if got := (&Style{BorderRadius: 8}).Radii(); got != (Corners{8, 8, 8, 8}) {
		t.Errorf("one radius: %+v", got)
	}
	s := styleOf(BorderRadius(8), CornerRadii(12, 0, 0, 12))
	if got := s.Radii(); got != (Corners{12, 0, 0, 12}) {
		t.Errorf("named corners should replace the radius, square ones included: %+v", got)
	}
	var nilStyle *Style
	if nilStyle.Radii() != (Corners{}) {
		t.Error("a nil style has no corners")
	}
}

// The last of the two props in an argument list is the shape.
func TestBorderRadiusAfterCornerRadiiWins(t *testing.T) {
	s := styleOf(CornerRadii(12, 0, 0, 12), BorderRadius(6))
	if s.Corners.Set() || s.Radii() != (Corners{6, 6, 6, 6}) {
		t.Errorf("a later BorderRadius should replace the corners: %+v", s.Radii())
	}
}

// A base's corners survive a style that names none, and a style that names
// any replaces all four: four radii are one shape.
func TestCornersMergeWhole(t *testing.T) {
	target := Style{Corners: Corners{1, 2, 3, 4}}
	Style{FontSize: 12}.applyTo(&target)
	if target.Corners != (Corners{1, 2, 3, 4}) {
		t.Errorf("a style naming no corners erased the base's: %+v", target.Corners)
	}
	Style{Corners: Corners{TopLeft: 9}}.applyTo(&target)
	if target.Corners != (Corners{TopLeft: 9}) {
		t.Errorf("named corners should replace all four: %+v", target.Corners)
	}
}

// Zero corners are not on the wire, so every existing tree is unchanged.
func TestZeroCornersAreNotOnTheWire(t *testing.T) {
	b, _ := json.Marshal(Style{BorderRadius: 8})
	if strings.Contains(string(b), "Corners") {
		t.Errorf("a style with no corners marshalled them: %s", b)
	}
	b, _ = json.Marshal(Style{Corners: Corners{TopLeft: 3}})
	if !strings.Contains(string(b), `"Corners":{"TopLeft":3}`) {
		t.Errorf("one corner should cross alone: %s", b)
	}
}
