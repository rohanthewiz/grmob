package components

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

func TestChipRendersAsButton(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	tapped := false
	n := Chip{Label: "Active", OnTap: func() { tapped = true }}.Render(ctx)

	if n.Type != "Button" || n.Props["label"] != "Active" {
		t.Fatalf("chip should be a themed Button, got %q %v", n.Type, n.Props)
	}
	id, ok := n.Props["onClick"].(string)
	if !ok {
		t.Fatal("chip should register an onClick callback")
	}
	ctx.TriggerCallback(id)
	if !tapped {
		t.Error("tapping the chip should invoke OnTap")
	}
	// Unselected: nothing but the theme Button base.
	theme := core.DefaultTheme
	if n.Style.Background != theme.Components.Button.Background {
		t.Errorf("unselected chip background = %q, want the Button base %q", n.Style.Background, theme.Components.Button.Background)
	}
}

func TestChipSelectedThemeDefault(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := Chip{Label: "Done", Selected: true, OnTap: func() {}}.Render(ctx)

	theme := core.DefaultTheme
	if n.Style.Background != theme.Colors.Surface {
		t.Errorf("selected default background = %q, want theme Surface %q", n.Style.Background, theme.Colors.Surface)
	}
	if n.Style.TextColor != theme.Colors.Primary {
		t.Errorf("selected default ink = %q, want theme Primary %q", n.Style.TextColor, theme.Colors.Primary)
	}
}

func TestChipSelectedStyleOverride(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := Chip{
		Label:    "Done",
		Selected: true,
		OnTap:    func() {},
		SelectedStyle: []core.StyleProp{
			core.BackgroundColor("#E8F0FE"),
			core.TextColor("#0B57D0"),
		},
	}.Render(ctx)

	if n.Style.Background != "#E8F0FE" || n.Style.TextColor != "#0B57D0" {
		t.Errorf("SelectedStyle should replace the theme default: %+v", n.Style)
	}
}

func TestChipAccessibilityAnnouncesSelection(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	base := Chip{Label: "All", OnTap: func() {}, AccessibilityLabel: "Show all tasks", AccessibilityHint: "Filters the task list"}
	unselected := base.Render(ctx)
	base.Selected = true
	selected := base.Render(ctx)

	if got := unselected.Style.AccessibilityLabel; got != "Show all tasks" {
		t.Errorf("unselected label = %q", got)
	}
	if got := selected.Style.AccessibilityLabel; got != "Show all tasks, selected" {
		t.Errorf("selected label = %q, want the state appended", got)
	}
	if got := selected.Style.AccessibilityHint; got != "Filters the task list" {
		t.Errorf("hint = %q", got)
	}
}

// TestChipNilOnTapDoesNotPanic pins the nil-handler guard. A decorative chip
// (a tag, a status pill) is a reasonable thing to write, and Chip handed a nil
// OnTap straight to core.Button, which registers whatever it is given; the
// registry then invoked it unguarded on the first tap.
func TestChipNilOnTapDoesNotPanic(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	n := Chip{Label: "Tag"}.Render(ctx)
	id, ok := n.Props["onClick"].(string)
	if !ok {
		t.Fatal("chip should still register an onClick callback with no OnTap")
	}
	if err := core.Guard(func() { ctx.TriggerCallback(id) }); err != nil {
		t.Fatalf("tapping a chip with no OnTap panicked: %v", err.Value)
	}
}
