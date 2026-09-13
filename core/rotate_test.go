package core

import "testing"

func TestRotateSetsTheAngleIncludingZero(t *testing.T) {
	var s Style
	Rotate(45).Apply(&s)
	if s.Rotate != 45 {
		t.Fatalf("Rotate = %g, want 45", s.Rotate)
	}
	// The prop can clear what UseStyle cannot: the documented difference
	// between a setter and a merge.
	Rotate(0).Apply(&s)
	if s.Rotate != 0 {
		t.Errorf("Rotate(0) left %g", s.Rotate)
	}
}

func TestUseStyleMergesRotateButCannotClearIt(t *testing.T) {
	base := Style{Rotate: 90}
	UseStyle(Style{Rotate: 30}).Apply(&base)
	if base.Rotate != 30 {
		t.Fatalf("merge left %g, want 30", base.Rotate)
	}
	// A zero field is "unset" to a merge, so it layers nothing — the
	// UseStyle edge, restated here because a compass at north is exactly the
	// value that would silently fail to apply.
	UseStyle(Style{Rotate: 0}).Apply(&base)
	if base.Rotate != 30 {
		t.Errorf("a zero-valued merge cleared the angle to %g", base.Rotate)
	}
}

// Translate sets both axes, including back to "", which is how a drawer that
// opens clears the offset its shut style gave the panel.
func TestTranslateSetsBothAxesAndClears(t *testing.T) {
	var s Style
	Translate("-100%", "8px").Apply(&s)
	if s.TranslateX != "-100%" || s.TranslateY != "8px" {
		t.Fatalf("Translate = (%q, %q), want (-100%%, 8px)", s.TranslateX, s.TranslateY)
	}
	Translate("", "").Apply(&s)
	if s.TranslateX != "" || s.TranslateY != "" {
		t.Errorf("Translate(\"\", \"\") left (%q, %q)", s.TranslateX, s.TranslateY)
	}
}

// A merge layers each axis on its own, so a role style can shift one axis and
// leave the node's other one standing.
func TestUseStyleMergesTranslatePerAxis(t *testing.T) {
	base := Style{TranslateX: "12px", TranslateY: "4px"}
	UseStyle(Style{TranslateY: "-50%"}).Apply(&base)
	if base.TranslateX != "12px" || base.TranslateY != "-50%" {
		t.Errorf("merge = (%q, %q), want (12px, -50%%)", base.TranslateX, base.TranslateY)
	}
}

// The angle is deliberately not folded onto the circle; see Style.Rotate.
func TestRotateKeepsTheWindingItWasGiven(t *testing.T) {
	for _, deg := range []float64{-90, 370, 720} {
		var s Style
		Rotate(deg).Apply(&s)
		if s.Rotate != deg {
			t.Errorf("Rotate(%g) stored %g; the winding must survive", deg, s.Rotate)
		}
	}
}
