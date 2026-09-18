package comps

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

var photoTrail = []string{"Files", "Photos", "2026"}

func TestBreadcrumbIsANavigationWithTheCurrentPageLast(t *testing.T) {
	var taps []int
	ctx, n := renderDebug(t, Breadcrumb{Items: photoTrail, OnTap: func(i int) { taps = append(taps, i) }})

	if n.Style.AccessibilityRole != core.RoleNavigation || n.Style.AccessibilityLabel != "Breadcrumb" {
		t.Fatalf("root role/label = %q/%q, want a navigation named Breadcrumb",
			n.Style.AccessibilityRole, n.Style.AccessibilityLabel)
	}
	if n.Style.FlexWrap != "wrap" {
		t.Error("a long trail should wrap, not run off the edge")
	}

	// Every ancestor is a button; the current page is not.
	btns := buttonsOf(n)
	if len(btns) != len(photoTrail)-1 {
		t.Fatalf("buttons = %d, want one per ancestor", len(btns))
	}
	for i, b := range btns {
		if b.Props["label"] != photoTrail[i] {
			t.Errorf("button %d = %v, want %q", i, b.Props["label"], photoTrail[i])
		}
	}
	cur := findText(n, "2026")
	if cur == nil || cur.Style.AccessibilityCurrent != core.CurrentPage {
		t.Fatal("the last item should be text stating the current page")
	}

	// The chevrons are drawn between items and hidden.
	chevrons := 0
	findFirst(n, func(c *core.Node) bool {
		if c.Type == "Text" && c.Props["content"] == "›" {
			chevrons++
			if !c.Style.AccessibilityHidden {
				t.Error("a chevron should be hidden from assistive technology")
			}
		}
		return false
	})
	if chevrons != len(photoTrail)-1 {
		t.Errorf("chevrons = %d, want one between each pair", chevrons)
	}

	ctx.TriggerCallback(btns[1].Props["onClick"].(string))
	if len(taps) != 1 || taps[0] != 1 {
		t.Errorf("taps = %v, want exactly [1]", taps)
	}
}

// With no OnTap the trail is a location label: all text, no dead controls.
func TestBreadcrumbWithoutOnTapIsReadOnly(t *testing.T) {
	_, n := renderDebug(t, Breadcrumb{Items: photoTrail})
	if btns := buttonsOf(n); len(btns) != 0 {
		t.Errorf("buttons = %d, want none on a read-only trail", len(btns))
	}
	for _, s := range photoTrail {
		if findText(n, s) == nil {
			t.Errorf("%q should be drawn as text", s)
		}
	}
	if findText(n, "2026").Style.AccessibilityCurrent != core.CurrentPage {
		t.Error("the current page is still stated on a read-only trail")
	}
}
