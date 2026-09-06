package htmlout

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// zstack builds an overlay node with the given children, styled or not.
func zstack(style *core.Style, children ...*core.Node) *core.Node {
	return &core.Node{Type: "ZStack", Style: style, Children: children}
}

// The overlay is a single-cell grid and every layer is placed in that cell.
//
// Both halves are asserted, because either alone is a working-looking box that
// lays its children out wrong: a grid whose children carry no grid-area runs
// them down the page in separate implicit rows, and a grid-area on children of
// a box that is not a grid does nothing at all.
func TestOverlayIsAGridWithEveryChildInOneCell(t *testing.T) {
	out := ExportHTML(zstack(&core.Style{Width: "160px"},
		textNode("under"),
		textNode("over"),
	))

	if !strings.Contains(out, "display:grid") {
		t.Errorf("the stack is not a grid:\n%s", out)
	}
	// Centre on both axes is core.ZStack's alignment contract, and the *items*
	// properties are the ones that place a child inside its cell.
	for _, decl := range []string{"align-items:center", "justify-items:center"} {
		if !strings.Contains(out, decl) {
			t.Errorf("the stack is missing %q:\n%s", decl, out)
		}
	}
	if n := strings.Count(out, "grid-area:1/1"); n != 2 {
		t.Errorf("%d of 2 layers were placed in the cell:\n%s", n, out)
	}
}

// A stack with no Style at all still overlays. core.ZStack carries no theme
// base, so this is the shape a stack written with children and nothing else
// arrives in, and a chassis that rode on the presence of a Style would leave
// exactly that stack laying its layers out down the page.
func TestOverlayNeedsNoStyleToBeAnOverlay(t *testing.T) {
	out := ExportHTML(zstack(nil, textNode("under"), textNode("over")))

	if !strings.Contains(out, "display:grid") {
		t.Errorf("an unstyled stack is not a grid:\n%s", out)
	}
	if n := strings.Count(out, "grid-area:1/1"); n != 2 {
		t.Errorf("%d of 2 layers were placed in the cell:\n%s", n, out)
	}
}

// The container props that promote any other box to a flex container must not
// promote this one. A ZStack that answered to Gap would stop overlaying
// entirely — a total loss of the node type's whole behavior, produced by a
// prop that is merely meaningless on an overlay.
func TestOverlayIsNotPromotedToAFlexContainer(t *testing.T) {
	for name, style := range map[string]*core.Style{
		"Gap":            {Gap: 8},
		"RowGap":         {RowGap: 8},
		"ColumnGap":      {ColumnGap: 8},
		"JustifyContent": {JustifyContent: core.JustifyBetween},
		"AlignItems":     {AlignItems: core.AlignItemsEnd},
		"FlexDirection":  {FlexDirection: core.FlexRow},
	} {
		out := ExportHTML(zstack(style, textNode("a")))
		if strings.Contains(out, "display:flex") {
			t.Errorf("%s turned the stack into a flex container:\n%s", name, out)
		}
		if !strings.Contains(out, "display:grid") {
			t.Errorf("%s cost the stack its grid:\n%s", name, out)
		}
	}
}

// Display is resolved against the grid exactly as it is against a flex
// container: "none" wins over the layout, "block" is already said by
// display:grid, and "inline" is folded in rather than emitted beside it.
func TestOverlayResolvesDisplayLikeAFlexContainer(t *testing.T) {
	none := ExportHTML(zstack(&core.Style{Display: core.DisplayNone}, textNode("a")))
	if !strings.Contains(none, "display:none") {
		t.Errorf("DisplayNone did not survive the grid:\n%s", none)
	}

	block := ExportHTML(zstack(&core.Style{Display: core.DisplayBlock}, textNode("a")))
	if strings.Contains(block, "display:block") {
		t.Errorf("display:block was written beside the grid, and wins the cascade:\n%s", block)
	}

	inline := ExportHTML(zstack(&core.Style{Display: core.DisplayInline}, textNode("a")))
	if !strings.Contains(inline, "display:inline-grid") {
		t.Errorf("an inline stack lost one of its two halves:\n%s", inline)
	}
}

// A Fragment layer forwards the cell to its own children rather than absorbing
// it. core.For wraps what it generates in a Fragment, so without this a
// generated set of layers would stack down the page inside one cell — and it
// would look right for a single-element For, which is the worst version of the
// bug.
func TestAFragmentLayerForwardsTheCellToItsChildren(t *testing.T) {
	out := ExportHTML(zstack(nil,
		textNode("under"),
		&core.Node{Type: "Fragment", Children: []*core.Node{textNode("a"), textNode("b")}},
	))

	if n := strings.Count(out, "grid-area:1/1"); n != 3 {
		t.Errorf("%d layers placed, want 3 (one child plus the Fragment's two):\n%s", n, out)
	}
}

// Only an overlay imposes the cell. Every other container has to keep emitting
// children that carry nothing of their parent's — the imposed channel is
// shared with the TabView page wiring, and a decl leaking onto ordinary
// children would show up as a grid-area on half the document.
func TestOrdinaryContainersImposeNothing(t *testing.T) {
	for _, nodeType := range []string{"Box", "Column", "Row", "Card", "Scroll", "List"} {
		out := ExportHTML(&core.Node{Type: nodeType, Children: []*core.Node{textNode("a")}})
		if strings.Contains(out, "grid-area") {
			t.Errorf("a %s placed its child in a grid cell:\n%s", nodeType, out)
		}
	}
}

// The overlay table and the tag table have to agree that a ZStack is a real
// node type. A row missing from `tags` would export the stack as the default
// div — the same element, so the document would look identical — while the
// census the tag table is meant to be quietly lost a type.
func TestOverlayTypesAreNodeTypes(t *testing.T) {
	tags := Tags()
	for _, nodeType := range OverlayTypes() {
		if _, ok := tags[nodeType]; !ok {
			t.Errorf("overlayTypes has %q, which is not a node type in Tags()", nodeType)
		}
		if StackAxisFor(nodeType) != "" {
			t.Errorf("%q is both an overlay and a flex stack; the two layouts are exclusive", nodeType)
		}
	}
}
