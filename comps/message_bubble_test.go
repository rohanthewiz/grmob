package comps

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

func bubbleOf(t *testing.T, row *core.Node) *core.Node {
	t.Helper()
	if row.Type != "Row" || len(row.Children) != 1 {
		t.Fatalf("want Row(bubble), got %q with %d children", row.Type, len(row.Children))
	}
	return row.Children[0]
}

// Theirs: leading side, Surface with a hairline, sender line drawn, one stop
// named who-what-when with the parts hidden.
func TestMessageBubbleTheirs(t *testing.T) {
	_, row := renderDebug(t, MessageBubble{Text: "oi", Sender: "Ana", Time: "10:42"})
	th := core.DefaultTheme

	if row.Style.JustifyContent != core.JustifyStart {
		t.Errorf("justify = %q, want start", row.Style.JustifyContent)
	}
	b := bubbleOf(t, row)
	if b.Style.Background != th.Colors.Surface || b.Style.BorderWidth != 1 {
		t.Errorf("theirs = fill %q border %v, want Surface with a hairline", b.Style.Background, b.Style.BorderWidth)
	}
	if b.Style.AccessibilityLabel != "Ana, oi, 10:42" {
		t.Errorf("name = %q", b.Style.AccessibilityLabel)
	}
	if b.Style.MaxWidth != "80%" {
		t.Errorf("max width = %q", b.Style.MaxWidth)
	}
	for _, s := range []string{"Ana", "oi", "10:42"} {
		n := findText(b, s)
		if n == nil || !n.Style.AccessibilityHidden {
			t.Errorf("%q should be drawn and hidden under the bubble's name", s)
		}
	}
	if findText(b, "10:42").Style.TextColor != th.Colors.TextSecondary {
		t.Error("their time is in the secondary ink")
	}
}

// Mine: trailing side, Primary, no sender drawn even if one is given, the
// time in the text's ink, and "You" in the name.
func TestMessageBubbleMine(t *testing.T) {
	_, row := renderDebug(t, MessageBubble{Text: "olá", Sender: "ignored", Mine: true, Time: "10:43"})
	th := core.DefaultTheme

	if row.Style.JustifyContent != core.JustifyEnd {
		t.Errorf("justify = %q, want end", row.Style.JustifyContent)
	}
	b := bubbleOf(t, row)
	if b.Style.Background != th.Colors.Primary || b.Style.BorderWidth != 0 {
		t.Errorf("mine = fill %q border %v, want Primary with no hairline", b.Style.Background, b.Style.BorderWidth)
	}
	if findText(b, "ignored") != nil {
		t.Error("no sender is drawn on the reader's own message")
	}
	if b.Style.AccessibilityLabel != "You, olá, 10:43" {
		t.Errorf("name = %q", b.Style.AccessibilityLabel)
	}
	text, tm := findText(b, "olá"), findText(b, "10:43")
	if text.Style.TextColor != VariantDefault.Ink(th, th.Colors.Primary) || tm.Style.TextColor != text.Style.TextColor {
		t.Errorf("ink = text %q time %q, want the contrast ink for both", text.Style.TextColor, tm.Style.TextColor)
	}
}

// No sender, no time: just the text, named by it alone; MineLabel localizes.
func TestMessageBubbleMinimalAndMineLabel(t *testing.T) {
	_, row := renderDebug(t, MessageBubble{Text: "oi"})
	if b := bubbleOf(t, row); len(b.Children) != 1 || b.Style.AccessibilityLabel != "oi" {
		t.Errorf("children %d name %q", len(b.Children), b.Style.AccessibilityLabel)
	}
	_, row = renderDebug(t, MessageBubble{Text: "oi", Mine: true, MineLabel: "Eu"})
	if got := bubbleOf(t, row).Style.AccessibilityLabel; got != "Eu, oi" {
		t.Errorf("name = %q", got)
	}
}

// Style lands on the row, where a gap between messages belongs.
func TestMessageBubbleStyleIsThePlacement(t *testing.T) {
	_, row := renderDebug(t, MessageBubble{Text: "oi", Style: []core.StyleProp{core.MarginBottom(8)}})
	if row.Style.Margin != (core.EdgeInsets{Bottom: 8}) {
		t.Errorf("row margin = %+v", row.Style.Margin)
	}
}

// Both sides pass the accessibility audit, which renderDebug does not run.
func TestMessageBubblePassesTheAudit(t *testing.T) {
	for _, m := range []MessageBubble{{Text: "oi", Sender: "Ana", Time: "10:42"}, {Text: "olá", Mine: true}} {
		_, row := renderDebug(t, m)
		core.ClearConcerns()
		core.AuditTree(row)
		if dump := core.DumpConcerns(); dump != "" {
			t.Errorf("%+v: audit raised:\n%s", m, dump)
		}
	}
}
