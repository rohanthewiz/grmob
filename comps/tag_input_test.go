package comps

import (
	"slices"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// tagHarness renders a TagInput through the rowHarness, which keeps one
// Context across passes so the draft hook survives a re-render, and holds
// the tag set OnChange writes back.
type tagHarness struct {
	*rowHarness
	tags    []string
	changes int
	max     int
}

func newTagHarness(t *testing.T, tags []string, max int) *tagHarness {
	t.Helper()
	h := &tagHarness{tags: tags, max: max}
	h.rowHarness = newRowHarness(t, func() core.View {
		return TagInput{
			Tags:        h.tags,
			OnChange:    func(next []string) { h.changes++; h.tags = next },
			Label:       "Labels",
			Placeholder: "Add a label",
			Max:         h.max,
		}
	})
	return h
}

func (h *tagHarness) input() *core.Node {
	h.t.Helper()
	in := findFirst(h.node, func(n *core.Node) bool { return n.Type == "Input" })
	if in == nil {
		h.t.Fatal("no input drawn")
	}
	return in
}

// typeText reports s as the input's whole new text, then re-renders.
func (h *tagHarness) typeText(s string) {
	h.t.Helper()
	h.ctx.TriggerTextCallback(h.input().Props["onChange"].(string), s)
	h.render()
}

// submit presses return, then re-renders.
func (h *tagHarness) submit() {
	h.t.Helper()
	h.ctx.TriggerCallback(h.input().Props["onSubmit"].(string))
	h.render()
}

func (h *tagHarness) remove(name string) {
	h.t.Helper()
	b := findFirst(h.node, func(n *core.Node) bool {
		return n.Type == "Button" && n.Style.AccessibilityLabel == name
	})
	if b == nil {
		h.t.Fatalf("no button named %q", name)
	}
	h.ctx.TriggerCallback(b.Props["onClick"].(string))
	h.render()
}

func (h *tagHarness) wantTags(want ...string) {
	h.t.Helper()
	if !slices.Equal(h.tags, want) {
		h.t.Errorf("tags = %q, want %q", h.tags, want)
	}
}

func (h *tagHarness) wantDraft(want string) {
	h.t.Helper()
	if got := h.input().Props["value"]; got != want {
		h.t.Errorf("draft = %q, want %q", got, want)
	}
}

// The structure: a list of listitems, each the tag's text and a named ✕,
// above an input named by Label.
func TestTagInputStructureAndAccessibility(t *testing.T) {
	h := newTagHarness(t, []string{"design", "urgent"}, 0)

	list := findFirst(h.node, func(n *core.Node) bool { return n.Style.AccessibilityRole == core.RoleList })
	if list == nil || len(list.Children) != 2 {
		t.Fatal("want a list with one item per tag")
	}
	for i, tag := range []string{"design", "urgent"} {
		item := list.Children[i]
		if item.Style.AccessibilityRole != core.RoleListItem {
			t.Errorf("item %d role = %q", i, item.Style.AccessibilityRole)
		}
		if item.Style.AccessibilityLabel != "" {
			t.Errorf("item %d is named %q; a name would hide its ✕ on the natives", i, item.Style.AccessibilityLabel)
		}
		if findText(item, tag) == nil {
			t.Errorf("item %d does not draw %q", i, tag)
		}
		b := buttonsOf(item)
		if len(b) != 1 || b[0].Style.AccessibilityLabel != "Remove "+tag {
			t.Errorf("item %d: want one ✕ named %q", i, "Remove "+tag)
		}
	}
	if in := h.input(); in.Style.AccessibilityLabel != "Labels" || in.Props["placeholder"] != "Add a label" {
		t.Errorf("input name %q placeholder %v", in.Style.AccessibilityLabel, in.Props["placeholder"])
	}
}

// With no tags there is no list at all: the column is just the input.
func TestTagInputWithNoTagsDrawsOnlyTheInput(t *testing.T) {
	h := newTagHarness(t, nil, 0)
	if len(h.node.Children) != 1 || h.node.Children[0].Type != "Input" {
		t.Errorf("children = %d, want the input alone", len(h.node.Children))
	}
}

// Typing builds a draft; return commits it trimmed; a separator commits
// everything before it and keeps what follows.
func TestTagInputCommitsOnReturnAndOnSeparator(t *testing.T) {
	h := newTagHarness(t, nil, 0)

	h.typeText("desi")
	h.wantDraft("desi")
	h.wantTags()
	if h.changes != 0 {
		t.Error("a draft is not a change")
	}

	h.typeText("  design ")
	h.submit()
	h.wantTags("design")
	h.wantDraft("")

	h.typeText("urgent,")
	h.wantTags("design", "urgent")
	h.wantDraft("")

	// A paste: two complete, one still being typed.
	h.typeText("a, b, c")
	h.wantTags("design", "urgent", "a", "b")
	h.wantDraft(" c")
	h.submit()
	h.wantTags("design", "urgent", "a", "b", "c")
	if h.changes != 4 {
		t.Errorf("changes = %d, want one per commit that added something", h.changes)
	}
}

// Empty pieces and duplicates are dropped without a change; the draft still
// clears, because the tag the reader wanted is already there.
func TestTagInputDropsEmptiesAndDuplicates(t *testing.T) {
	h := newTagHarness(t, []string{"design"}, 0)
	h.typeText("design,")
	h.typeText(" , ,")
	h.submit()
	h.wantTags("design")
	h.wantDraft("")
	if h.changes != 0 {
		t.Errorf("changes = %d, want none", h.changes)
	}
}

// ✕ removes exactly its own tag.
func TestTagInputRemove(t *testing.T) {
	h := newTagHarness(t, []string{"a", "b", "c"}, 0)
	h.remove("Remove b")
	h.wantTags("a", "c")
	if h.changes != 1 {
		t.Errorf("changes = %d, want 1", h.changes)
	}
}

// Max stops a paste part-way and disables the input once reached; removing
// one enables it again.
func TestTagInputMax(t *testing.T) {
	h := newTagHarness(t, []string{"a"}, 3)
	h.typeText("b, c, d, e,")
	h.wantTags("a", "b", "c")
	if !h.input().Style.Disabled {
		t.Error("the input should be disabled at Max")
	}
	h.remove("Remove a")
	if h.input().Style.Disabled {
		t.Error("below Max the input is enabled again")
	}
}

// A custom separator set, including a multi-byte one.
func TestTagInputCustomSeparators(t *testing.T) {
	next, rest := splitDraft("one;two、three", ";、")
	if !slices.Equal(next, []string{"one", "two"}) || rest != "three" {
		t.Errorf("split = %q, rest %q", next, rest)
	}
}

// A TagInput with nowhere to report is reported.
func TestTagInputWithNoOnChangeReportsInert(t *testing.T) {
	newQuietRowHarness(t, func() core.View { return TagInput{Label: "Labels"} })
	if dump := core.DumpConcerns(); !strings.Contains(dump, ConcernTagInputInert) {
		t.Errorf("concerns = %q, want %s", dump, ConcernTagInputInert)
	}
}
