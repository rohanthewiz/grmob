package comps

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

func sampleMenu(open bool, onOpen, onPick, onDismiss func()) Menu {
	return Menu{
		Trigger: Button{
			Label:              "⋯",
			AccessibilityLabel: "Note actions",
			Emphasis:           EmphasisGhost,
			// Set to prove the template's own handler never runs.
			OnTap: func() { panic("the trigger's template OnTap must be replaced by OnOpen") },
		},
		Open:      open,
		OnOpen:    onOpen,
		OnDismiss: onDismiss,
		Title:     "Groceries",
		Items: []SheetAction{
			{Label: "Rename", OnTap: onPick},
			{Label: "Delete", Variant: VariantError},
		},
		Cancel: "Cancel",
	}
}

// menuParts returns the trigger Button and the sheet's Modal: the wrapper's
// two children, in that order.
func menuParts(t *testing.T, n *core.Node) (trigger, modal *core.Node) {
	t.Helper()
	if n.Type != "Row" || len(n.Children) != 2 {
		t.Fatalf("root = %q with %d children, want Row(trigger, Modal)", n.Type, len(n.Children))
	}
	trigger, modal = n.Children[0], n.Children[1]
	if trigger.Type != "Button" || modal.Type != "Modal" {
		t.Fatalf("children = %q, %q; want Button, Modal", trigger.Type, modal.Type)
	}
	return trigger, modal
}

func TestMenuIsATriggerBesideAnActionSheet(t *testing.T) {
	_, n := renderDebug(t, sampleMenu(false, func() {}, func() {}, func() {}))
	trigger, modal := menuParts(t, n)

	if n.Style.Gap != 0 {
		t.Errorf("the wrapper must add no gap beside the Modal, got %v", n.Style.Gap)
	}
	if trigger.Props["label"] != "⋯" || trigger.Style.AccessibilityLabel != "Note actions" {
		t.Errorf("trigger = %v named %q, want the template's label and name",
			trigger.Props["label"], trigger.Style.AccessibilityLabel)
	}
	if trigger.Style.AccessibilityExpanded != core.ExpandedUnset {
		t.Error("a trigger that opens a dialog states no expanded state (core.Style's near miss)")
	}
	if modal.Props["visible"] != false {
		t.Error("Open false must render a closed sheet")
	}

	_, n = renderDebug(t, sampleMenu(true, func() {}, func() {}, func() {}))
	_, modal = menuParts(t, n)
	if modal.Props["visible"] != true {
		t.Error("Open true must render an open sheet")
	}
	if len(buttonsOf(modal)) != 3 {
		t.Errorf("want Rename, Delete, Cancel in the sheet, got %d buttons", len(buttonsOf(modal)))
	}
}

// The trigger calls OnOpen once; the template's own OnTap is never reached.
func TestMenuTriggerCallsOnOpen(t *testing.T) {
	opened := 0
	ctx, n := renderDebug(t, sampleMenu(false, func() { opened++ }, nil, nil))
	trigger, _ := menuParts(t, n)

	ctx.TriggerCallback(trigger.Props["onClick"].(string))
	if opened != 1 {
		t.Errorf("OnOpen ran %d times, want 1", opened)
	}
}

// A nil OnOpen still registers a handler, so a native tap finds one.
func TestMenuWithoutOnOpenHasAnInertTrigger(t *testing.T) {
	m := sampleMenu(false, nil, nil, nil)
	ctx, n := renderDebug(t, m)
	trigger, _ := menuParts(t, n)

	id, ok := trigger.Props["onClick"].(string)
	if !ok {
		t.Fatal("the trigger must still register a handler")
	}
	ctx.TriggerCallback(id)
}

// Picking an item runs it and closes the menu, as ActionSheet does.
func TestMenuItemRunsThenDismisses(t *testing.T) {
	var order []string
	ctx, n := renderDebug(t, sampleMenu(true,
		func() {},
		func() { order = append(order, "rename") },
		func() { order = append(order, "dismiss") },
	))
	_, modal := menuParts(t, n)

	ctx.TriggerCallback(buttonsOf(modal)[0].Props["onClick"].(string))
	if len(order) != 2 || order[0] != "rename" || order[1] != "dismiss" {
		t.Errorf("order = %v, want [rename dismiss]", order)
	}
}

// The trigger's Style survives the template copy, and the sheet's Style lands
// on the card rather than on the trigger.
func TestMenuStylesReachTheirOwnNodes(t *testing.T) {
	m := sampleMenu(true, func() {}, nil, nil)
	m.Trigger.Style = []core.StyleProp{core.MarginLeft(4)}
	m.Style = []core.StyleProp{core.MaxWidth("480px")}
	_, n := renderDebug(t, m)
	trigger, modal := menuParts(t, n)

	if trigger.Style.Margin.Left != 4 {
		t.Errorf("trigger margin = %v, want the template's Style", trigger.Style.Margin.Left)
	}
	card := findFirst(modal, func(n *core.Node) bool { return n.Type == "Card" })
	if card == nil || card.Style.MaxWidth != "480px" {
		t.Error("Menu.Style must land on the sheet's card")
	}
	if trigger.Style.MaxWidth != "" {
		t.Error("Menu.Style must not reach the trigger")
	}
}

// A checked item leads with ✓ and is named with a ", selected" suffix; an
// unchecked one keeps its plain label as its name.
func TestSheetActionCheckedMarksTheCurrentChoice(t *testing.T) {
	_, n := renderDebug(t, Menu{
		Trigger:   Button{Label: "Sort: Newest"},
		Open:      true,
		OnOpen:    func() {},
		OnDismiss: func() {},
		Title:     "Sort by",
		Items: []SheetAction{
			{Label: "Newest", Checked: true},
			{Label: "Oldest"},
		},
	})
	_, modal := menuParts(t, n)
	btns := buttonsOf(modal)
	if len(btns) != 2 {
		t.Fatalf("want two items, got %d", len(btns))
	}
	if btns[0].Props["label"] != "✓ Newest" || btns[0].Style.AccessibilityLabel != "Newest, selected" {
		t.Errorf("checked item = %v named %q", btns[0].Props["label"], btns[0].Style.AccessibilityLabel)
	}
	if btns[1].Props["label"] != "Oldest" || btns[1].Style.AccessibilityLabel != "" {
		t.Errorf("unchecked item = %v named %q", btns[1].Props["label"], btns[1].Style.AccessibilityLabel)
	}
	if btns[0].Style.AccessibilitySelected != core.SelectedUnset {
		t.Error("a checked action is a button: no selected state, which would read as pressed")
	}
}
