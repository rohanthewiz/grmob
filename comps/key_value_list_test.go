package comps

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

var orderFacts = []KeyValue{
	{Key: "Order", Value: "#40121"},
	{Key: "Placed", Value: "14 Mar 2026"},
	{Key: "Total", Value: "$42.10"},
}

// The list owns both halves of its structure, so it may claim it: a list of
// listitems, each named with its key and value together.
func TestKeyValueListIsAListOfNamedItems(t *testing.T) {
	_, n := renderDebug(t, KeyValueList{Rows: orderFacts, Label: "Order details"})

	if n.Style.AccessibilityRole != core.RoleList {
		t.Fatalf("role = %q, want list", n.Style.AccessibilityRole)
	}
	if n.Style.AccessibilityLabel != "Order details" {
		t.Errorf("label = %q", n.Style.AccessibilityLabel)
	}
	if len(n.Children) != len(orderFacts) {
		t.Fatalf("children = %d, want one row per fact and nothing else", len(n.Children))
	}
	for i, row := range n.Children {
		if row.Style.AccessibilityRole != core.RoleListItem {
			t.Errorf("row %d role = %q, want listitem", i, row.Style.AccessibilityRole)
		}
		want := orderFacts[i].Key + ", " + orderFacts[i].Value
		if got := row.Style.AccessibilityLabel; got != want {
			t.Errorf("row %d name = %q, want %q", i, got, want)
		}
	}

	// The key in the body ink, the value in the secondary one and at the
	// trailing edge.
	key, value := findText(n, "Total"), findText(n, "$42.10")
	if key == nil || value == nil {
		t.Fatal("the key and the value should both be drawn")
	}
	if value.Style.TextColor != core.DefaultTheme.Colors.TextSecondary {
		t.Errorf("value ink = %q, want TextSecondary", value.Style.TextColor)
	}
	if key.Style.TextColor == value.Style.TextColor {
		t.Error("the key and the value should not share an ink")
	}
	if value.Style.Align != core.AlignEnd {
		t.Errorf("value align = %q, want end", value.Style.Align)
	}
	// A long value wraps; the key keeps its line.
	if key.Style.FlexShrink != core.ShrinkNone {
		t.Error("the key should be pinned at its width so a long value cannot break it")
	}
}

// Dividers go between rows only, and are hidden, so the list's children as a
// reader counts them are still only its items.
func TestKeyValueListDividersGoBetweenRows(t *testing.T) {
	_, n := renderDebug(t, KeyValueList{Rows: orderFacts, Dividers: true})
	if len(n.Children) != 2*len(orderFacts)-1 {
		t.Fatalf("children = %d, want rows with a rule between each pair", len(n.Children))
	}
	for i, c := range n.Children {
		isRule := i%2 == 1
		if isRule != c.Style.AccessibilityHidden {
			t.Errorf("child %d hidden = %v, want %v", i, c.Style.AccessibilityHidden, isRule)
		}
	}
}

// A fact still loading is its key alone: no trailing Text, and a name with no
// dangling comma.
func TestKeyValueListEmptyValueDrawsTheKeyAlone(t *testing.T) {
	_, n := renderDebug(t, KeyValueList{Rows: []KeyValue{{Key: "Tracking"}}})
	row := n.Children[0]
	if got := row.Style.AccessibilityLabel; got != "Tracking" {
		t.Errorf("name = %q, want the key alone", got)
	}
	texts := 0
	findFirst(row, func(c *core.Node) bool {
		if c.Type == "Text" {
			texts++
		}
		return false
	})
	if texts != 1 {
		t.Errorf("texts = %d, want the key only", texts)
	}
}
