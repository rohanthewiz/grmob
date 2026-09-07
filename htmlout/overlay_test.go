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

// A layer names its own corner; the rest keep the centre.
//
// The placement travels through the `imposed` channel rather than being
// written by the layer, so this is also the assertion that the channel now
// varies per child — every other declaration it has ever carried was one
// string for the whole sibling set.
func TestOverlayPlacesEachLayerWhereItAsked(t *testing.T) {
	out := ExportHTML(zstack(&core.Style{Width: "160px"},
		textNode("under"),
		&core.Node{Type: "Text", Props: map[string]any{"content": "N"},
			Style: &core.Style{StackAlign: core.StackAlignTop}},
		&core.Node{Type: "Text", Props: map[string]any{"content": "SE"},
			Style: &core.Style{StackAlign: core.StackAlignBottomEnd}},
	))

	for _, want := range []string{
		// The unplaced layer, stated rather than left to the chassis — see
		// stackPlacements on why the centre is in the table.
		"grid-area:1/1; justify-self:center; align-self:center",
		"grid-area:1/1; justify-self:center; align-self:start",
		"grid-area:1/1; justify-self:end; align-self:end",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
}

// Every declared placement must reach the markup, and no two may reach it as
// the same pair.
//
// A census rather than three spot checks, because the table is the kind of
// thing that is written once and copied downwards: two rows with the same
// value look right and put two different corners in one place. Driven off
// core.StackAlignments() so a ninth placement fails here until it has a row.
func TestEveryPlacementReachesTheMarkupAndIsDistinct(t *testing.T) {
	seen := map[string]core.StackAlignment{}
	for _, align := range core.StackAlignments() {
		out := ExportHTML(zstack(nil, &core.Node{Type: "Text",
			Props: map[string]any{"content": "x"},
			Style: &core.Style{StackAlign: align}}))

		decl := StackPlacementFor(align)
		if decl == StackPlacementFor(core.StackAlignCenter) {
			t.Errorf("%q resolves to the centre's declaration; a placement with no row of "+
				"its own renders as the default it was written to escape", align)
			continue
		}
		if !strings.Contains(out, decl) {
			t.Errorf("%q does not reach the markup as %q:\n%s", align, decl, out)
		}
		if prev, dup := seen[decl]; dup {
			t.Errorf("%q and %q both resolve to %q", align, prev, decl)
		}
		seen[decl] = align
	}
}

// The placement is the stack's to impose, so a StackAlign on a node whose
// parent is not an overlay must reach the markup as nothing at all.
//
// Not merely a nicety. `align-self` is a flexbox property that a *flex* item
// honours, so a layer prop written into the node's own declaration list would
// re-place a Row's children — on the two DOM targets only, since neither
// native reads it outside a stack. Routing it through the parent is what makes
// "inert outside a ZStack" true rather than approximately true.
func TestAPlacementOutsideAStackReachesNothing(t *testing.T) {
	out := ExportHTML(&core.Node{Type: "Row", Children: []*core.Node{
		{Type: "Text", Props: map[string]any{"content": "a"},
			Style: &core.Style{StackAlign: core.StackAlignTopEnd}},
	}})

	for _, unwanted := range []string{"justify-self", "align-self", "grid-area"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("a Row's child was given %q by a StackAlign:\n%s", unwanted, out)
		}
	}
}

// A layer's own Style.AlignSelf must not move it.
//
// This is the leak the centre row of stackPlacements closes. AlignSelf is
// flexbox's spelling and a grid item honours it too, so before the stack
// imposed a placement on *every* layer, one flex prop moved a layer on the two
// DOM targets and nowhere else — in flat contradiction of the alignment
// contract core.ZStack documents.
func TestALayersOwnAlignSelfDoesNotPlaceIt(t *testing.T) {
	out := ExportHTML(zstack(&core.Style{Width: "160px"},
		&core.Node{Type: "Text", Props: map[string]any{"content": "a"},
			Style: &core.Style{AlignSelf: core.AlignItemsEnd}},
	))

	// Both are present; the imposed one is written last and wins the
	// browser's last-one-wins parse, which is the whole reason `imposed.decl`
	// is appended rather than prepended.
	self := strings.Index(out, "align-self:flex-end")
	imposed := strings.Index(out, "align-self:center")
	if self < 0 || imposed < 0 {
		t.Fatalf("expected both the layer's own align-self and the imposed centre:\n%s", out)
	}
	if imposed < self {
		t.Errorf("the imposed centring is written before the layer's own align-self, so the "+
			"flex prop wins the cascade and places the layer:\n%s", out)
	}
}

// This package's two node-type sets and core's must be the same sets.
//
// overlayTypes and transparentTypes are this exporter's, and they carry the
// *rendering* consequence of each decision: a grid cell that keeps a layer in
// flow, a wrapper that would swallow the parent's gap and flex-direction. core
// now states the same two memberships for the placement audit
// (core.PlacingContainers and core.GroupingContainers), because a fact about
// core.ZStack belongs to the package that defines core.ZStack.
//
// Two lists is one too many unless something compares them, and the failure if
// nothing did is quiet in both directions. A type this exporter treats as an
// overlay and core does not is a stack whose every layer the audit reports; a
// type core places and this exporter does not is a placement written into a
// document that has no grid to honour it. Neither shows up as a test failure
// anywhere else, because each package's own tests are internally consistent.
//
// The comparison lives here rather than in core because only this direction
// compiles: htmlout imports core.
func TestHtmloutAgreesWithCoreOnWhoPlacesAndWhoGroups(t *testing.T) {
	for _, c := range []struct {
		what      string
		mine      []string
		theirs    []string
		mineName  string
		theirName string
	}{
		{"the containers that place their children",
			OverlayTypes(), core.PlacingContainers(),
			"htmlout.OverlayTypes", "core.PlacingContainers"},
		{"the containers with no box of their own",
			TransparentTypes(), core.GroupingContainers(),
			"htmlout.TransparentTypes", "core.GroupingContainers"},
	} {
		// Both are already sorted by their own accessors, which is what makes
		// a positional comparison honest here rather than lucky.
		if len(c.mine) != len(c.theirs) {
			t.Errorf("%s: %s() = %v, %s() = %v — the two lists must name the same node "+
				"types, or the exporter and the placement audit disagree about %s",
				c.what, c.mineName, c.mine, c.theirName, c.theirs, c.what)
			continue
		}
		for i := range c.mine {
			if c.mine[i] != c.theirs[i] {
				t.Errorf("%s: %s() = %v, %s() = %v", c.what,
					c.mineName, c.mine, c.theirName, c.theirs)
				break
			}
		}
	}
}
