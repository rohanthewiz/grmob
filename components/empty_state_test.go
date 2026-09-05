package components

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// The one assertion in this file that is about a real bug rather than a
// preference: without a full width the whole block sits at the leading edge on
// both natives, with its children centered inside a box only as wide as the
// longest line. The two DOM targets fill the line anyway, so this is invisible
// on the target you are most likely to be looking at.
func TestEmptyStateFillsItsWidth(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := EmptyState{Title: "No sermons yet"}.Render(ctx)

	if n.Style.Width != "100%" {
		t.Errorf("width = %q, want 100%% — a column hugs its widest child on both natives", n.Style.Width)
	}
	if n.Style.AlignItems != core.AlignItemsCenter {
		t.Errorf("alignItems = %q, want center", n.Style.AlignItems)
	}
}

func TestEmptyStateRoles(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	theme := core.DefaultTheme

	n := EmptyState{Glyph: "📭", Title: "No messages yet", Hint: "Start a conversation."}.Render(ctx)

	glyph := findText(n, "📭")
	if glyph == nil {
		t.Fatal("no glyph")
	}
	// Decoration: "open mailbox with lowered flag" ahead of "No messages yet"
	// is noise, so Title has to carry the meaning.
	if !glyph.Style.AccessibilityHidden {
		t.Error("the glyph should be hidden from assistive tech")
	}

	title := findText(n, "No messages yet")
	if title == nil {
		t.Fatal("no title")
	}
	if title.Style.TextColor != theme.Colors.TextPrimary {
		t.Errorf("title ink = %q, want TextPrimary", title.Style.TextColor)
	}
	// Align, not AlignItems: this is what centers the *lines* of a title that
	// wraps, where the column's alignment only centers the box.
	if title.Style.Align != core.AlignCenter {
		t.Errorf("title align = %q, want center", title.Style.Align)
	}

	hint := findText(n, "Start a conversation.")
	if hint == nil {
		t.Fatal("no hint")
	}
	if hint.Style.FontSize != theme.Typography.Caption.FontSize {
		t.Errorf("hint size = %v, want the Caption role %v", hint.Style.FontSize, theme.Typography.Caption.FontSize)
	}
}

// The busy case: a line of text and no mark. A glyph there would look like a
// state the user is meant to read.
func TestEmptyStateGlyphIsOptional(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := EmptyState{Title: "Loading sermons…"}.Render(ctx)

	if len(n.Children) != 1 {
		t.Errorf("a titled state with no glyph should render one child, got %d", len(n.Children))
	}
}

func TestEmptyStateActionIsOutlined(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	retried := false
	n := EmptyState{
		Glyph: "☁", Title: "Could not reach the server.",
		ActionLabel: "Retry", OnAction: func() { retried = true },
	}.Render(ctx)

	retry := findButton(n, "Retry")
	if retry == nil {
		t.Fatal("no action button")
	}
	// Outlined, not filled: an empty state is a dead end, and a solid Primary
	// button in the middle of an empty screen is the loudest thing on it.
	if retry.Style.Background != ColorTransparent {
		t.Errorf("action background = %q, want transparent (outlined)", retry.Style.Background)
	}
	if retry.Style.BorderWidth != 1 {
		t.Errorf("action border width = %v, want 1 (outlined)", retry.Style.BorderWidth)
	}
	ctx.TriggerCallback(retry.Props["onClick"].(string))
	if !retried {
		t.Error("tapping the action should invoke OnAction")
	}
}

func TestEmptyStateActionSlotWinsOverTheBuiltButton(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := EmptyState{
		Title:       "No filters match",
		ActionLabel: "Clear", OnAction: func() {},
		Action: core.Row(Button{Label: "Clear filters", OnTap: func() {}},
			Button{Label: "Browse all", OnTap: func() {}}),
	}.Render(ctx)

	if findButton(n, "Clear filters") == nil || findButton(n, "Browse all") == nil {
		t.Error("Action should be rendered")
	}
	if findButton(n, "Clear") != nil {
		t.Error("Action should replace the built button")
	}
}

func TestEmptyStateHalfConfiguredActionDrawsNothing(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	if n := (EmptyState{Title: "x", ActionLabel: "Retry"}).Render(ctx); findButton(n, "Retry") != nil {
		t.Error("an ActionLabel with no OnAction should draw no button")
	}
	if n := (EmptyState{Title: "x", OnAction: func() {}}).Render(ctx); len(n.Children) != 1 {
		t.Error("an OnAction with no ActionLabel should draw no button")
	}
}

func TestEmptyStateStyleOverridesTheDefaults(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := EmptyState{Title: "x", Style: []core.StyleProp{core.Width("50%")}}.Render(ctx)

	if n.Style.Width != "50%" {
		t.Errorf("width = %q, want the caller's override", n.Style.Width)
	}
}
