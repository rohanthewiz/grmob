package comps

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

func sampleSheet(onShare, onDelete, onDismiss func()) ActionSheet {
	return ActionSheet{
		Visible: true,
		Title:   "Note",
		Actions: []SheetAction{
			{Label: "Share", OnTap: onShare},
			{Label: "Delete", Variant: VariantError, OnTap: onDelete},
		},
		Cancel:    "Cancel",
		OnDismiss: onDismiss,
	}
}

// The two-child shape is the whole placement answer: a growing filler first,
// the labelled card last. See the type doc's per-host table.
func TestActionSheetIsAFillerAboveALabelledCard(t *testing.T) {
	_, n := renderDebug(t, sampleSheet(func() {}, func() {}, func() {}))

	if n.Type != "Modal" || n.Props["visible"] != true {
		t.Fatalf("root = %q visible=%v, want an open Modal", n.Type, n.Props["visible"])
	}
	if len(n.Children) != 2 {
		t.Fatalf("want filler + card, got %d children", len(n.Children))
	}
	filler, card := n.Children[0], n.Children[1]
	if filler.Type != "Box" || filler.Style.FlexGrow != 1 || filler.Style.AlignSelf != core.AlignItemsStretch {
		t.Errorf("filler = %q grow=%v alignSelf=%q, want a growing stretched Box",
			filler.Type, filler.Style.FlexGrow, filler.Style.AlignSelf)
	}
	if !filler.Style.AccessibilityHidden {
		t.Error("the filler is scrim, not content: it must be hidden from assistive technology")
	}
	if filler.Style.Background != "" {
		t.Error("the filler must paint nothing: iOS takes the first painted child as the sheet surface")
	}
	if card.Type != "Card" || card.Style.Width != "100%" || card.Style.AccessibilityLabel != "Note" {
		t.Errorf("card = %q width=%q label=%q", card.Type, card.Style.Width, card.Style.AccessibilityLabel)
	}
	if card.Style.Background == "" {
		t.Error("the card must paint a background: Compose adds its own surface otherwise")
	}
}

func TestActionSheetActionsAreFullWidthGhostButtonsThenCancel(t *testing.T) {
	_, n := renderDebug(t, sampleSheet(func() {}, func() {}, func() {}))

	btns := buttonsOf(n)
	if len(btns) != 3 {
		t.Fatalf("want Share, Delete, Cancel; got %d buttons", len(btns))
	}
	for i, want := range []string{"Share", "Delete", "Cancel"} {
		if btns[i].Props["label"] != want {
			t.Errorf("button %d = %v, want %s", i, btns[i].Props["label"], want)
		}
		if btns[i].Style.Width != "100%" {
			t.Errorf("%s width = %q, want full width", want, btns[i].Style.Width)
		}
	}
	if btns[0].Style.Background != ColorTransparent {
		t.Error("actions are ghost buttons")
	}
	if btns[1].Style.TextColor != core.DefaultTheme.Colors.ErrorOnLightColor() {
		t.Errorf("a destructive action takes the error ink, got %q", btns[1].Style.TextColor)
	}
	if btns[2].Style.BorderWidth != 1 {
		t.Error("Cancel is outlined so it reads as leaving the list")
	}
	if findFirst(n, func(n *core.Node) bool {
		return n.Style != nil && n.Style.AccessibilityRole == core.RoleOption
	}) != nil {
		t.Error("actions are commands, not options: no listbox roles")
	}
}

// One tap on an action runs the action and then closes the sheet, once each.
func TestActionSheetActionRunsThenDismisses(t *testing.T) {
	var order []string
	ctx, n := renderDebug(t, sampleSheet(
		func() { order = append(order, "share") },
		func() { order = append(order, "delete") },
		func() { order = append(order, "dismiss") },
	))

	ctx.TriggerCallback(buttonsOf(n)[1].Props["onClick"].(string))
	if len(order) != 2 || order[0] != "delete" || order[1] != "dismiss" {
		t.Errorf("order = %v, want [delete dismiss]", order)
	}
}

func TestActionSheetCancelAndFillerDismiss(t *testing.T) {
	dismissed := 0
	ctx, n := renderDebug(t, sampleSheet(nil, nil, func() { dismissed++ }))

	ctx.TriggerCallback(buttonsOf(n)[2].Props["onClick"].(string))
	ctx.TriggerCallback(n.Children[0].Props["onClick"].(string))
	ctx.TriggerCallback(n.Props["onDismiss"].(string))
	if dismissed != 3 {
		t.Errorf("Cancel, filler and scrim dismissed %d times, want 3", dismissed)
	}

	// A nil action OnTap still closes the sheet.
	ctx.TriggerCallback(buttonsOf(n)[0].Props["onClick"].(string))
	if dismissed != 4 {
		t.Errorf("an action with no OnTap should still dismiss, count = %d", dismissed)
	}
}

// With no OnDismiss neither the Modal nor the filler registers a handler, so
// the scrim is inert on every host; Cancel still renders and is harmless.
func TestActionSheetWithoutOnDismissHasAnInertScrim(t *testing.T) {
	ctx, n := renderDebug(t, ActionSheet{Title: "Note", Cancel: "Close",
		Actions: []SheetAction{{Label: "Share"}}})

	if _, ok := n.Props["onDismiss"]; ok {
		t.Error("no OnDismiss: the Modal must carry no onDismiss prop")
	}
	if _, ok := n.Children[0].Props["onClick"]; ok {
		t.Error("no OnDismiss: the filler must not be tappable")
	}
	for _, b := range buttonsOf(n) {
		ctx.TriggerCallback(b.Props["onClick"].(string)) // must not panic
	}
}

func TestActionSheetOmitsEmptyRegions(t *testing.T) {
	_, n := renderDebug(t, ActionSheet{Actions: []SheetAction{{Label: "Share"}}})
	card := n.Children[1]
	if findFirst(card, func(n *core.Node) bool {
		return n.Style != nil && n.Style.AccessibilityRole == core.RoleHeading
	}) != nil {
		t.Error("no Title: no heading")
	}
	if card.Style.AccessibilityLabel != "" {
		t.Error("no Title: no label")
	}
	if len(buttonsOf(n)) != 1 {
		t.Error("no Cancel: only the action button")
	}
}

func TestActionSheetDisabledActionDoesNotDispatchOrDismiss(t *testing.T) {
	var tapped, dismissed bool
	ctx, n := renderDebug(t, ActionSheet{
		Visible:   true,
		Actions:   []SheetAction{{Label: "Delete", Disabled: true, OnTap: func() { tapped = true }}},
		OnDismiss: func() { dismissed = true },
	})
	b := buttonsOf(n)[0]
	if !b.Style.Disabled {
		t.Error("a disabled action carries the platform disabled flag")
	}
	ctx.TriggerCallback(b.Props["onClick"].(string))
	if tapped || dismissed {
		t.Errorf("disabled action: tapped=%v dismissed=%v, want neither", tapped, dismissed)
	}
}

func TestActionSheetStyleAndBackdropOverrides(t *testing.T) {
	_, n := renderDebug(t, ActionSheet{
		Title:    "Note",
		Backdrop: "#000000CC",
		Style:    []core.StyleProp{core.MaxWidth("480px"), core.AccessibilityLabel("Note actions")},
	})
	if n.Props["backdrop"] != "#000000CC" {
		t.Errorf("backdrop = %v", n.Props["backdrop"])
	}
	card := n.Children[1]
	if card.Style.MaxWidth != "480px" || card.Style.AccessibilityLabel != "Note actions" {
		t.Errorf("caller Style must land on the card last: maxWidth=%q label=%q",
			card.Style.MaxWidth, card.Style.AccessibilityLabel)
	}
}
