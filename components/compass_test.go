package components

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// renderCompass renders c against a fresh context and returns the root node.
func renderCompass(t *testing.T, c Compass) *core.Node {
	t.Helper()
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	return c.Render(ctx)
}

// rose finds the rotating rose: the one node carrying a rotation.
func rose(t *testing.T, n *core.Node) *core.Node {
	t.Helper()
	found := findFirst(n, func(n *core.Node) bool {
		return n.Style != nil && n.Style.BorderRadius > 0 && n.Style.Width != ""
	})
	if found == nil {
		t.Fatal("no rose found in the rendered compass")
	}
	return found
}

// The rose turns the opposite way to the device — the whole premise of a
// navigation compass. A sign error here points the widget exactly backwards
// and looks plausible until you turn the phone.
func TestRoseTurnsAgainstTheHeading(t *testing.T) {
	for _, deg := range []float64{0, 90, 217.5, 359} {
		got := rose(t, renderCompass(t, Compass{Heading: deg})).Style.Rotate
		if got != -deg {
			t.Errorf("Heading %g rotated the rose %g, want %g", deg, got, -deg)
		}
	}
}

// The bearing handed to the rotation keeps its winding, while the bearing
// shown to a person is folded onto the circle. Both halves matter and they
// disagree on purpose.
func TestDisplayIsNormalisedWhileRotationIsNot(t *testing.T) {
	n := renderCompass(t, Compass{Heading: -45, ShowDegrees: true})
	if got := rose(t, n).Style.Rotate; got != 45 {
		t.Errorf("rotation = %g, want 45 (the negation, unfolded)", got)
	}
	if findText(n, "315° NW") == nil {
		t.Errorf("readout missing; want the normalised 315° NW in:\n%s", dumpText(n))
	}
}

func TestRoseCarriesTheFourCardinalLetters(t *testing.T) {
	n := renderCompass(t, Compass{})
	for _, letter := range []string{"N", "E", "S", "W"} {
		if findText(n, letter) == nil {
			t.Errorf("cardinal %q missing from:\n%s", letter, dumpText(n))
		}
	}
}

func TestDegreesReadoutIsOffByDefault(t *testing.T) {
	n := renderCompass(t, Compass{Heading: 90})
	if strings.Contains(dumpText(n), "°") {
		t.Errorf("a degrees readout appeared without ShowDegrees:\n%s", dumpText(n))
	}
	if findText(renderCompass(t, Compass{Heading: 90, ShowDegrees: true}), "90° E") == nil {
		t.Error("ShowDegrees did not produce the readout")
	}
}

// A rotating rose read in tree order is "N W E S" whatever the bearing, so the
// widget must speak for itself and silence its parts. This is the test that
// fails if someone later "helpfully" labels the letters.
func TestTheWidgetSpeaksOnceAndHidesItsParts(t *testing.T) {
	n := renderCompass(t, Compass{Heading: 312, ShowDegrees: true})

	want := "Heading 312 degrees, northwest"
	if n.Style == nil || n.Style.AccessibilityLabel != want {
		t.Fatalf("container label = %q, want %q", labelOf(n), want)
	}
	// Without the role the label is dropped on both web targets: ARIA forbids
	// an accessible name on a generic element, and a core container exports as
	// exactly that. The natives honour a bare label, so this is the half that
	// fails only where nobody is looking.
	if n.Style.AccessibilityRole != core.RoleImg {
		t.Errorf("container role = %q, want %q — a label alone does not survive "+
			"on a generic element", n.Style.AccessibilityRole, core.RoleImg)
	}
	// Every descendant that carries a style must be hidden or inherit from a
	// hidden ancestor; nothing inside may announce itself separately.
	if !hiddenOrInsideHidden(n, false) {
		t.Error("some part of the compass is still visible to assistive tech")
	}
}

func TestAccessibilityLabelOverridesTheSpokenBearing(t *testing.T) {
	n := renderCompass(t, Compass{Heading: 90, AccessibilityLabel: "Direction to Mecca"})
	if labelOf(n) != "Direction to Mecca" {
		t.Errorf("label = %q, want the override", labelOf(n))
	}
}

// Size is the only geometry knob: the circle, the padding and the lettering
// all derive from it, so a caller cannot end up with an ellipse or with
// letters that outgrow the rose.
func TestSizeDrivesTheCircleAndTheLettering(t *testing.T) {
	small := rose(t, renderCompass(t, Compass{Size: 96}))
	if small.Style.Width != "96px" || small.Style.Height != "96px" {
		t.Errorf("rose = %s x %s, want 96px square", small.Style.Width, small.Style.Height)
	}
	if small.Style.BorderRadius != 48 {
		t.Errorf("radius = %g, want half the diameter", small.Style.BorderRadius)
	}

	// The letter floor: at 60px an eighth of the diameter is 7.5px, which is
	// not readable on a phone, so it clamps.
	tiny := findText(renderCompass(t, Compass{Size: 60}), "N")
	if tiny == nil || tiny.Style == nil || tiny.Style.FontSize != 10 {
		t.Errorf("small compass letter size = %v, want the 10px floor", fontOf(tiny))
	}
	big := findText(renderCompass(t, Compass{Size: 240}), "N")
	if big == nil || big.Style == nil || big.Style.FontSize != 30 {
		t.Errorf("large compass letter size = %v, want 240/8", fontOf(big))
	}
}

func TestDefaultSizeAppliesWhenUnsetOrNegative(t *testing.T) {
	for _, size := range []float64{0, -20} {
		if got := rose(t, renderCompass(t, Compass{Size: size})).Style.Width; got != "160px" {
			t.Errorf("Size %g gave width %q, want the 160px default", size, got)
		}
	}
}

// --- helpers ----------------------------------------------------------------

func labelOf(n *core.Node) string {
	if n == nil || n.Style == nil {
		return ""
	}
	return n.Style.AccessibilityLabel
}

func fontOf(n *core.Node) any {
	if n == nil || n.Style == nil {
		return nil
	}
	return n.Style.FontSize
}

// hiddenOrInsideHidden reports whether every styled descendant of n is hidden
// from assistive tech, either directly or by sitting under something that is.
// n itself is the labelled container and is exempt.
func hiddenOrInsideHidden(n *core.Node, underHidden bool) bool {
	for _, c := range n.Children {
		hidden := underHidden || (c.Style != nil && c.Style.AccessibilityHidden)
		if c.Style != nil && !hidden {
			return false
		}
		if !hiddenOrInsideHidden(c, hidden) {
			return false
		}
	}
	return true
}

// dumpText collects every Text node's content, for readable failures.
func dumpText(n *core.Node) string {
	var b strings.Builder
	var walk func(*core.Node)
	walk = func(n *core.Node) {
		if n == nil {
			return
		}
		if n.Type == "Text" {
			if s, ok := n.Props["content"].(string); ok {
				b.WriteString(s)
				b.WriteString("\n")
			}
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}
