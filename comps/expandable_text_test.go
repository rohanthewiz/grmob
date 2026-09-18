package comps

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

var longBlurb = strings.Repeat("The survey that was meant to take a week became a month. ", 4)

func (h *rowHarness) expandText() *core.Node {
	h.t.Helper()
	n := findFirst(h.node, func(n *core.Node) bool { return n.Type == "Text" })
	if n == nil {
		h.t.Fatal("no text")
	}
	return n
}

// Long text: capped at Lines with a "Read more" that opens and closes it,
// its name stable and its expanded state stated both ways.
func TestExpandableTextOpensAndCloses(t *testing.T) {
	h := newRowHarness(t, func() core.View { return ExpandableText{Text: longBlurb, Lines: 2} })

	for pass, want := range []struct {
		lines   int
		caption string
		state   core.ExpandedState
	}{
		{2, "Read more", core.ExpandedClosed},
		{0, "Read less", core.ExpandedOpen},
		{2, "Read more", core.ExpandedClosed},
	} {
		if got := h.expandText().Style.MaxLines; got != want.lines {
			t.Errorf("pass %d: MaxLines = %d, want %d", pass, got, want.lines)
		}
		if got := h.expandText().Props["content"]; got != longBlurb {
			t.Errorf("pass %d: the text node must always carry the whole string", pass)
		}
		b := buttonsOf(h.node)
		if len(b) != 1 {
			t.Fatalf("pass %d: buttons = %d, want the toggle", pass, len(b))
		}
		if b[0].Props["label"] != want.caption || b[0].Style.AccessibilityLabel != "Read more" {
			t.Errorf("pass %d: caption %v name %q", pass, b[0].Props["label"], b[0].Style.AccessibilityLabel)
		}
		if b[0].Style.AccessibilityExpanded != want.state {
			t.Errorf("pass %d: expanded = %q, want %q", pass, b[0].Style.AccessibilityExpanded, want.state)
		}
		h.ctx.TriggerCallback(b[0].Props["onClick"].(string))
		h.render()
	}
}

// Short text: no cap and no toggle — a cap with no way to lift it would hide
// the end of the text for good.
func TestExpandableTextShortHasNoToggleAndNoCap(t *testing.T) {
	_, n := renderDebug(t, ExpandableText{Text: "Two short lines."})
	if len(buttonsOf(n)) != 0 {
		t.Error("short text should draw no toggle")
	}
	if m := findFirst(n, func(n *core.Node) bool { return n.Type == "Text" }).Style.MaxLines; m != 0 {
		t.Errorf("MaxLines = %d, want uncapped", m)
	}
}

// ToggleAfter: the default is Lines × 40 runes (runes, not bytes), negative
// always toggles, and a caller's own threshold wins.
func TestExpandableTextThreshold(t *testing.T) {
	for _, tc := range []struct {
		name   string
		e      ExpandableText
		toggle bool
	}{
		{"120 runes at the default 3×40", ExpandableText{Text: strings.Repeat("a", 120)}, false},
		{"121 runes", ExpandableText{Text: strings.Repeat("a", 121)}, true},
		{"120 two-byte runes", ExpandableText{Text: strings.Repeat("é", 120)}, false},
		{"negative always", ExpandableText{Text: "short", ToggleAfter: -1}, true},
		{"caller's threshold", ExpandableText{Text: strings.Repeat("a", 50), ToggleAfter: 20}, true},
	} {
		_, n := renderDebug(t, tc.e)
		if got := len(buttonsOf(n)) == 1; got != tc.toggle {
			t.Errorf("%s: toggle = %v, want %v", tc.name, got, tc.toggle)
		}
	}
}
