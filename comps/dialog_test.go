package comps

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// renderDialog renders under debug mode so every test also proves the widget
// raises no framework concern (hook drift, unknown items, a11y audit).
func renderDialog(t *testing.T, d Dialog) (*core.Context, *core.Node) {
	t.Helper()
	core.SetDebugMode(true)
	core.ClearConcerns()
	t.Cleanup(func() { core.SetDebugMode(false); core.ClearConcerns() })

	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := d.Render(ctx)
	if dump := core.DumpConcerns(); dump != "" {
		t.Errorf("Dialog raised concerns:\n%s", dump)
	}
	return ctx, n
}

// buttonsOf returns the footer's Button nodes in render (= visual) order.
func buttonsOf(n *core.Node) []*core.Node {
	var out []*core.Node
	var walk func(*core.Node)
	walk = func(n *core.Node) {
		if n.Type == "Button" {
			out = append(out, n)
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(n)
	return out
}

func TestDialogIsAModalAroundALabelledCard(t *testing.T) {
	_, n := renderDialog(t, Dialog{
		Visible: true,
		Title:   "Delete note?",
		Message: "This cannot be undone.",
		Confirm: DialogAction{Label: "Delete", Variant: VariantError, OnTap: func() {}},
		Cancel:  DialogAction{Label: "Keep", OnTap: func() {}},
	})

	if n.Type != "Modal" {
		t.Fatalf("root = %q, want Modal: the chassis owns the scrim and the dialog role", n.Type)
	}
	if n.Props["visible"] != true {
		t.Errorf("visible = %v, want true", n.Props["visible"])
	}
	if len(n.Children) != 1 || n.Children[0].Type != "Card" {
		t.Fatalf("want exactly one Card inside the Modal, got %+v", n.Children)
	}
	card := n.Children[0]
	if got := card.Style.AccessibilityLabel; got != "Delete note?" {
		t.Errorf("card label = %q, want the Title: it is the dialog's accessible name", got)
	}

	title := findText(card, "Delete note?")
	if title == nil {
		t.Fatal("title text missing")
	}
	if title.Style.AccessibilityRole != core.RoleHeading {
		t.Errorf("title role = %q, want heading", title.Style.AccessibilityRole)
	}
	if findText(card, "This cannot be undone.") == nil {
		t.Error("message text missing")
	}
}

// Cancel leading, Confirm trailing, pushed to the trailing edge — the one
// convention the widget exists to fix.
func TestDialogButtonOrderAndEmphasis(t *testing.T) {
	_, n := renderDialog(t, Dialog{
		Visible: true,
		Title:   "Delete note?",
		Confirm: DialogAction{Label: "Delete", Variant: VariantError, OnTap: func() {}},
		Cancel:  DialogAction{Label: "Keep", OnTap: func() {}},
	})

	btns := buttonsOf(n)
	if len(btns) != 2 {
		t.Fatalf("want 2 buttons, got %d", len(btns))
	}
	if btns[0].Props["label"] != "Keep" || btns[1].Props["label"] != "Delete" {
		t.Errorf("order = %v, %v; want Keep then Delete", btns[0].Props["label"], btns[1].Props["label"])
	}
	if btns[0].Style.Background != ColorTransparent {
		t.Errorf("cancel background = %q, want transparent (ghost)", btns[0].Style.Background)
	}
	if btns[1].Style.Background != core.DefaultTheme.Colors.Error {
		t.Errorf("confirm background = %q, want the theme Error fill %q",
			btns[1].Style.Background, core.DefaultTheme.Colors.Error)
	}

	card := n.Children[0]
	footer := card.Children[len(card.Children)-1]
	if footer.Type != "Row" || footer.Style.JustifyContent != core.JustifyEnd {
		t.Errorf("footer must be a Row packed to the trailing edge, got %q justify=%q",
			footer.Type, footer.Style.JustifyContent)
	}
}

func TestDialogCancelFallsBackToOnDismiss(t *testing.T) {
	dismissed := 0
	ctx, n := renderDialog(t, Dialog{
		Visible:   true,
		Title:     "Discard draft?",
		Confirm:   DialogAction{Label: "Discard", OnTap: func() {}},
		Cancel:    DialogAction{Label: "Keep"},
		OnDismiss: func() { dismissed++ },
	})

	cancelID, ok := buttonsOf(n)[0].Props["onClick"].(string)
	if !ok {
		t.Fatal("cancel has no onClick")
	}
	ctx.TriggerCallback(cancelID)
	if dismissed != 1 {
		t.Errorf("cancel tap called OnDismiss %d times, want 1", dismissed)
	}

	dismissID, ok := n.Props["onDismiss"].(string)
	if !ok {
		t.Fatal("OnDismiss set, so the Modal must carry onDismiss")
	}
	ctx.TriggerCallback(dismissID)
	if dismissed != 2 {
		t.Errorf("backdrop dismiss count = %d, want 2", dismissed)
	}
}

func TestDialogConfirmFiresOnlyItsOwnHandler(t *testing.T) {
	var confirmed, dismissed int
	ctx, n := renderDialog(t, Dialog{
		Visible:   true,
		Title:     "Delete?",
		Confirm:   DialogAction{Label: "Delete", OnTap: func() { confirmed++ }},
		Cancel:    DialogAction{Label: "Keep"},
		OnDismiss: func() { dismissed++ },
	})

	ctx.TriggerCallback(buttonsOf(n)[1].Props["onClick"].(string))
	if confirmed != 1 || dismissed != 0 {
		t.Errorf("confirm tap: confirmed=%d dismissed=%d, want 1 and 0 — confirming does not close",
			confirmed, dismissed)
	}
}

// With no OnDismiss the Modal carries no onDismiss prop, which is how a host
// knows the scrim is inert.
func TestDialogWithoutOnDismissCannotBeDismissedByScrim(t *testing.T) {
	_, n := renderDialog(t, Dialog{
		Visible: true,
		Title:   "Update required",
		Confirm: DialogAction{Label: "Update", OnTap: func() {}},
	})
	if _, ok := n.Props["onDismiss"]; ok {
		t.Error("no OnDismiss should mean no onDismiss prop on the Modal")
	}
}

// The three shapes are one struct with fields left zero.
func TestDialogShapesFollowTheActions(t *testing.T) {
	cases := []struct {
		name    string
		d       Dialog
		buttons int
	}{
		{"confirm", Dialog{Title: "x", Confirm: DialogAction{Label: "OK"}, Cancel: DialogAction{Label: "No"}}, 2},
		{"alert", Dialog{Title: "x", Confirm: DialogAction{Label: "OK"}}, 1},
		{"sheet", Dialog{Title: "x", Body: core.Text("custom")}, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, n := renderDialog(t, c.d)
			if got := len(buttonsOf(n)); got != c.buttons {
				t.Errorf("buttons = %d, want %d", got, c.buttons)
			}
			if c.buttons == 0 {
				for _, ch := range n.Children[0].Children {
					if ch.Type == "Row" {
						t.Error("a sheet must omit the footer row, not render an empty one")
					}
				}
			}
		})
	}
}

func TestDialogBodyWinsOverMessage(t *testing.T) {
	_, n := renderDialog(t, Dialog{
		Title:   "Rename",
		Message: "ignored",
		Body:    core.Text("field goes here"),
	})
	if findText(n, "ignored") != nil {
		t.Error("Message must be suppressed when Body is set")
	}
	if findText(n, "field goes here") == nil {
		t.Error("Body slot should render")
	}
}

func TestDialogStyleAndBackdropOverrides(t *testing.T) {
	_, n := renderDialog(t, Dialog{
		Title:    "Delete?",
		Backdrop: "#000000CC",
		Style:    []core.StyleProp{core.AccessibilityLabel("Confirm deletion"), core.MaxWidth("320px")},
	})
	if n.Props["backdrop"] != "#000000CC" {
		t.Errorf("backdrop = %v", n.Props["backdrop"])
	}
	card := n.Children[0]
	if card.Style.AccessibilityLabel != "Confirm deletion" {
		t.Errorf("caller Style must beat the Title-derived label, got %q", card.Style.AccessibilityLabel)
	}
	if card.Style.MaxWidth != "320px" {
		t.Errorf("caller MaxWidth not applied, got %q", card.Style.MaxWidth)
	}
}

func TestDialogDisabledConfirmDropsTheTap(t *testing.T) {
	fired := false
	ctx, n := renderDialog(t, Dialog{
		Title:   "Delete?",
		Confirm: DialogAction{Label: "Delete", Disabled: true, OnTap: func() { fired = true }},
	})
	b := buttonsOf(n)[0]
	if !b.Style.Disabled {
		t.Error("a disabled action must carry the platform disabled flag")
	}
	ctx.TriggerCallback(b.Props["onClick"].(string))
	if fired {
		t.Error("a disabled Confirm must not reach its handler")
	}
}
