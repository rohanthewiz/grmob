package comps

import (
	"testing"
	"time"

	"github.com/rohanthewiz/grmob/core"
)

// typingDotsOf returns the three dot boxes: the children of the inner bubble
// row, which is the outer row's first child.
func typingDotsOf(t *testing.T, n *core.Node) []*core.Node {
	t.Helper()
	if n.Type != "Row" || len(n.Children) == 0 {
		t.Fatalf("want Row(bubble, caption?), got %q with %d children", n.Type, len(n.Children))
	}
	dots := n.Children[0].Children
	if len(dots) != typingDots {
		t.Fatalf("bubble holds %d dots, want %d", len(dots), typingDots)
	}
	return dots
}

// darkDot is the index of the one dot drawn at full strength, or -1. Every
// dot is the same fill; the dark one is the one whose opacity reads as 1, and
// the others must rest at typingRestAlpha, declared (an undeclared opacity is
// also 1, which would make a resting dot dark).
func darkDot(t *testing.T, n *core.Node) int {
	t.Helper()
	dark := -1
	for i, d := range typingDotsOf(t, n) {
		if d.Style.Background != core.DefaultTheme.Colors.TextPrimary {
			t.Fatalf("dot %d is filled %q; every dot is TextPrimary and only the alpha differs",
				i, d.Style.Background)
		}
		alpha, declared := d.Style.OpacityFactor()
		if !declared || (alpha != 1 && alpha != typingRestAlpha) {
			t.Fatalf("dot %d: opacity (%g, declared %v), want a declared 1 or %g",
				i, alpha, declared, typingRestAlpha)
		}
		if alpha == 1 {
			if dark >= 0 {
				t.Fatalf("dots %d and %d are both dark; exactly one should be", dark, i)
			}
			dark = i
		}
	}
	return dark
}

// Visible: one status stop named for who is typing, a theirs-coloured bubble
// hidden from assistive technology, three dots of which the first is dark,
// each with the transition that lets the platform draw the fade.
func TestTypingIndicatorIsOneStatusStopOverThreeDots(t *testing.T) {
	h := newRowHarness(t, func() core.View { return TypingIndicator{Visible: true, Who: "Ana"} })
	defer h.ctx.Close()
	n, th := h.node, core.DefaultTheme

	if n.Style.AccessibilityRole != core.RoleStatus || n.Style.AccessibilityLabel != "Ana is typing" {
		t.Errorf("role %q label %q, want status / Ana is typing", n.Style.AccessibilityRole, n.Style.AccessibilityLabel)
	}
	if n.Style.Display == core.DisplayNone {
		t.Error("a visible indicator must not be display none")
	}
	if len(n.Children) != 1 {
		t.Errorf("children = %d, want the bubble alone: Caption is off by default", len(n.Children))
	}
	bubble := n.Children[0]
	if bubble.Style.Background != th.Colors.Surface || bubble.Style.BorderWidth != 1 || !bubble.Style.AccessibilityHidden {
		t.Errorf("bubble = fill %q border %v hidden %v, want theirs colours, hidden",
			bubble.Style.Background, bubble.Style.BorderWidth, bubble.Style.AccessibilityHidden)
	}
	if got := darkDot(t, n); got != 0 {
		t.Errorf("dark dot = %d, want 0 on the first pass", got)
	}
	for i, d := range typingDotsOf(t, n) {
		if d.Style.Transition == "" {
			t.Errorf("dot %d has no Transition, so its opacity would snap on every target", i)
		}
	}
}

// The label's three sources, in order of precedence.
func TestTypingIndicatorLabelDefaults(t *testing.T) {
	cases := []struct {
		ti   TypingIndicator
		want string
	}{
		{TypingIndicator{}, "Typing"},
		{TypingIndicator{Who: "Ana"}, "Ana is typing"},
		{TypingIndicator{Who: "Ana", Label: "Ana está a escrever"}, "Ana está a escrever"},
	}
	for _, c := range cases {
		if got := c.ti.label(); got != c.want {
			t.Errorf("label(%+v) = %q, want %q", c.ti, got, c.want)
		}
	}
}

// Caption draws the label beside the dots, hidden: the row already speaks it.
func TestTypingIndicatorCaptionIsDrawnAndNotSpokenTwice(t *testing.T) {
	h := newRowHarness(t, func() core.View { return TypingIndicator{Visible: true, Who: "Ana", Caption: true} })
	defer h.ctx.Close()
	c := findText(h.node, "Ana is typing")
	if c == nil || !c.Style.AccessibilityHidden {
		t.Error("the caption should be drawn, and hidden under the row's name")
	}
}

// The zero value is hidden, and hidden is a style on the same tree: the dots
// are still there, so showing the indicator is a patch and not an insert.
func TestTypingIndicatorHiddenKeepsItsShape(t *testing.T) {
	h := newRowHarness(t, func() core.View { return TypingIndicator{} })
	defer h.ctx.Close()
	if h.node.Style.Display != core.DisplayNone {
		t.Errorf("display = %q, want none for the zero value", h.node.Style.Display)
	}
	typingDotsOf(t, h.node)
}

// The interval moves the dark dot while Visible, and only then. Polled rather
// than slept for: the ticker's first beat is typingBeat after the first pass,
// and the claim is that it arrives, not exactly when.
func TestTypingIndicatorStepsOnlyWhileVisible(t *testing.T) {
	visible := true
	h := newRowHarness(t, func() core.View { return TypingIndicator{Visible: visible} })
	defer h.ctx.Close()

	moved := false
	for deadline := time.Now().Add(5 * typingBeat); time.Now().Before(deadline); {
		time.Sleep(typingBeat / 8)
		h.render()
		if darkDot(t, h.node) != 0 {
			moved = true
			break
		}
	}
	if !moved {
		t.Fatal("the dark dot never left the first place while Visible")
	}

	// Hidden: the pause is read from the most recent pass, so render once
	// with the new value, note where the dot is, and expect it to stay.
	visible = false
	h.render()
	at := darkDot(t, h.node)
	time.Sleep(3 * typingBeat)
	h.render()
	if got := darkDot(t, h.node); got != at {
		t.Errorf("dark dot moved %d → %d while hidden; the interval should be paused", at, got)
	}
}

// Style lands on the outer row, the placement, as with MessageBubble.
func TestTypingIndicatorStyleIsThePlacement(t *testing.T) {
	h := newRowHarness(t, func() core.View {
		return TypingIndicator{Visible: true, Style: []core.StyleProp{core.MarginBottom(8)}}
	})
	defer h.ctx.Close()
	if h.node.Style.Margin != (core.EdgeInsets{Bottom: 8}) {
		t.Errorf("row margin = %+v", h.node.Style.Margin)
	}
}
