package comps

import (
	"strconv"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// A list of listitems named by their text, with the marker hidden and pinned
// and the text growing beside it.
func TestBulletListIsAListOfNamedItems(t *testing.T) {
	items := []string{"Free delivery", "Cancel any time", "Priority support"}
	_, n := renderDebug(t, BulletList{Items: items, Label: "What's included"})

	if n.Style.AccessibilityRole != core.RoleList || n.Style.AccessibilityLabel != "What's included" {
		t.Fatalf("list a11y = role %q name %q", n.Style.AccessibilityRole, n.Style.AccessibilityLabel)
	}
	if len(n.Children) != len(items) {
		t.Fatalf("children = %d, want one row per item", len(n.Children))
	}
	for i, row := range n.Children {
		if row.Style.AccessibilityRole != core.RoleListItem || row.Style.AccessibilityLabel != items[i] {
			t.Errorf("row %d = role %q name %q", i, row.Style.AccessibilityRole, row.Style.AccessibilityLabel)
		}
		if len(row.Children) != 2 {
			t.Fatalf("row %d children = %d, want marker and text", i, len(row.Children))
		}
		mark, text := row.Children[0], row.Children[1]
		if mark.Props["content"] != "•" {
			t.Errorf("row %d marker = %v, want •", i, mark.Props["content"])
		}
		if mark.Style.FlexShrink != core.ShrinkNone || !mark.Style.AccessibilityHidden {
			t.Errorf("row %d marker should be pinned and hidden", i)
		}
		if mark.Style.Width != "" {
			t.Errorf("row %d: an unordered marker needs no column width, got %q", i, mark.Style.Width)
		}
		if text.Style.FlexGrow != 1 {
			t.Errorf("row %d text should grow so a long item wraps under itself", i)
		}
	}
}

// Ordered markers count from Start and share one right-aligned column, so
// "9." and "10." end at the same x.
func TestBulletListOrderedMarkersShareAColumn(t *testing.T) {
	items := make([]string, 3)
	for i := range items {
		items[i] = "step"
	}
	_, n := renderDebug(t, BulletList{Items: items, Ordered: true, Start: 9})

	var widths []string
	for i, want := range []string{"9.", "10.", "11."} {
		mark := n.Children[i].Children[0]
		if mark.Props["content"] != want {
			t.Errorf("marker %d = %v, want %s", i, mark.Props["content"], want)
		}
		if mark.Style.Align != core.AlignEnd {
			t.Errorf("marker %d align = %q, want end", i, mark.Style.Align)
		}
		widths = append(widths, mark.Style.Width)
	}
	if widths[0] == "" || widths[0] != widths[1] || widths[1] != widths[2] {
		t.Errorf("marker widths = %v, want one shared non-empty width", widths)
	}

	// Fewer digits, narrower column.
	_, short := renderDebug(t, BulletList{Items: items[:2], Ordered: true})
	if one, two := pxOf(t, short.Children[0].Children[0].Style.Width), pxOf(t, widths[0]); one >= two {
		t.Errorf("a one-digit column (%vpx) should be narrower than a two-digit one (%vpx)", one, two)
	}
}

func pxOf(t *testing.T, w string) float64 {
	t.Helper()
	v, err := strconv.ParseFloat(strings.TrimSuffix(w, "px"), 64)
	if err != nil {
		t.Fatalf("width %q is not a px length", w)
	}
	return v
}

// A custom marker, and an empty list that is still a (childless) list.
func TestBulletListCustomMarkerAndEmpty(t *testing.T) {
	_, n := renderDebug(t, BulletList{Items: []string{"a"}, Marker: "–"})
	if got := n.Children[0].Children[0].Props["content"]; got != "–" {
		t.Errorf("marker = %v, want –", got)
	}
	_, empty := renderDebug(t, BulletList{})
	if len(empty.Children) != 0 {
		t.Errorf("empty list children = %d", len(empty.Children))
	}
}
