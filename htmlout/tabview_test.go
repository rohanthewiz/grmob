package htmlout

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// A TabView with two tabs and one page each, the shape core.TabView builds.
func tabViewNode(selected int, onChange string) *core.Node {
	props := map[string]any{
		"selectedIndex": selected,
		"tabs": []map[string]string{
			{"label": "Home", "icon": "🏠"},
			{"label": "Search", "icon": "🔍"},
		},
	}
	if onChange != "" {
		props["onTabChange"] = onChange
	}
	return &core.Node{
		Type:  "TabView",
		Props: props,
		Children: []*core.Node{
			{Type: "Column", Children: []*core.Node{{Type: "Text", Props: map[string]any{"content": "home page"}}}},
			{Type: "Column", Children: []*core.Node{{Type: "Text", Props: map[string]any{"content": "search page"}}}},
		},
	}
}

// The gap this closed: none of core.TabView's four wire props were read here,
// so a TabView exported as a bare box holding every page, with no bar — while
// both natives drew a bar above the selected page alone.
func TestTabViewExportsABar(t *testing.T) {
	out := ExportHTML(tabViewNode(0, "cb_9"))
	for _, want := range []string{
		`data-grmob-chrome="tabbar"`,
		`role="tablist"`,
		`data-ontabchange="cb_9"`,
		`role="tab"`,
		`data-tab-index="0"`,
		`data-tab-index="1"`,
		">Home<",
		">Search<",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
}

// The bar comes first. It is the order the runtime's chrome offset assumes,
// and the order a screen reader and a style-less rendering want anyway.
func TestTabBarPrecedesThePages(t *testing.T) {
	out := ExportHTML(tabViewNode(0, "cb_9"))
	bar := strings.Index(out, `data-grmob-chrome="tabbar"`)
	page := strings.Index(out, "home page")
	if bar < 0 || page < 0 {
		t.Fatalf("want both a bar and a page:\n%s", out)
	}
	if bar > page {
		t.Errorf("the bar was written after the first page:\n%s", out)
	}
}

// The icon is drawn by no target: Compose's Tab is built with
// `text = { Text(label) }` and the SwiftUI bar with a Text of the label, so
// drawing it here would make the web the outlier rather than close a gap.
func TestTabBarDrawsLabelsAndNotIcons(t *testing.T) {
	out := ExportHTML(tabViewNode(0, "cb_9"))
	if strings.Contains(out, "🏠") || strings.Contains(out, "🔍") {
		t.Errorf("an icon reached the export:\n%s", out)
	}
}

func TestSelectedTabIsMarkedAndStyled(t *testing.T) {
	out := ExportHTML(tabViewNode(1, "cb_9"))
	tabs := tabTags(t, out)
	if len(tabs) != 2 {
		t.Fatalf("want two tabs, got %d:\n%s", len(tabs), out)
	}
	if !strings.Contains(tabs[0], `aria-selected="false"`) {
		t.Errorf("tab 0 is not marked unselected:\n%s", out)
	}
	if strings.Contains(tabs[0], tabSelectedChassis) {
		t.Errorf("tab 0 carries the selected styling:\n%s", out)
	}
	if !strings.Contains(tabs[1], `aria-selected="true"`) {
		t.Errorf("tab 1 is not marked selected:\n%s", out)
	}
	if !strings.Contains(tabs[1], tabSelectedChassis) {
		t.Errorf("the selected tab carries no selected styling:\n%s", out)
	}
}

// tabTags returns each tab button's opening tag, in document order. Sliced
// rather than matched with a regexp because everything asserted about a tab
// lives in one tag and the boundary is unambiguous: element writes the whole
// tag on one line, and the label follows the ">".
func tabTags(t *testing.T, out string) []string {
	t.Helper()
	var tags []string
	rest := out
	for {
		at := strings.Index(rest, `<button type="button" role="tab"`)
		if at < 0 {
			return tags
		}
		rest = rest[at:]
		end := strings.Index(rest, ">")
		if end < 0 {
			t.Fatalf("unterminated tab tag:\n%s", out)
		}
		tags = append(tags, rest[:end])
		rest = rest[end:]
	}
}

// The pages are hidden, not dropped. The runtime cannot drop one — its patch
// TargetIDs are positional, so its DOM has to stay isomorphic to the node tree
// — and an export that silently lost every screen but one would be a worse
// document for no gain.
func TestOnlyTheSelectedPageIsVisible(t *testing.T) {
	out := ExportHTML(tabViewNode(1, ""))
	if !strings.Contains(out, "home page") || !strings.Contains(out, "search page") {
		t.Fatalf("a page was dropped rather than hidden:\n%s", out)
	}
	home := strings.Index(out, "home page")
	search := strings.Index(out, "search page")
	// The hidden page's own box carries the declaration; find the style
	// attribute immediately preceding its text.
	if !strings.Contains(out[:home], tabHiddenPage) {
		t.Errorf("the unselected page was not hidden:\n%s", out)
	}
	if strings.Contains(out[home:search], tabHiddenPage) {
		t.Errorf("the selected page was hidden:\n%s", out)
	}
}

// display:none has to outrank the display:flex a stack container is given
// unconditionally (see stackAxes), which is what the last-one-wins ordering of
// the declaration list buys.
func TestHiddenPageKeepsItsOwnStyleAndLoses(t *testing.T) {
	out := ExportHTML(tabViewNode(1, ""))
	home := strings.Index(out, "home page")
	decls := out[:home]
	flex := strings.LastIndex(decls, "display:flex")
	none := strings.LastIndex(decls, "display:none")
	if flex < 0 || none < 0 {
		t.Fatalf("want both declarations on the hidden page:\n%s", out)
	}
	if none < flex {
		t.Errorf("display:none precedes display:flex, so the page is not hidden:\n%s", out)
	}
}

// Both natives guard the page rather than clamping the index
// (`children.indices.contains` in Swift, `getOrNull` in Kotlin), and the Swift
// bar compares `i == selected` with no clamp either.
func TestOutOfRangeSelectionShowsNoPage(t *testing.T) {
	out := ExportHTML(tabViewNode(7, ""))
	if strings.Count(out, tabHiddenPage) != 2 {
		t.Errorf("want both pages hidden:\n%s", out)
	}
	if strings.Contains(out, `aria-selected="true"`) {
		t.Errorf("a tab was marked selected for an out-of-range index:\n%s", out)
	}
}

// A bar with no tabs to name would be a hairline across the top of the node,
// which is a visible artifact of an absent prop.
func TestNoTabsMeansNoBar(t *testing.T) {
	n := &core.Node{Type: "TabView", Children: []*core.Node{{Type: "Text", Props: map[string]any{"content": "only"}}}}
	out := ExportHTML(n)
	if strings.Contains(out, "tabbar") {
		t.Errorf("a bar was drawn for a TabView with no tabs:\n%s", out)
	}
	// And the one page is page 0, so it stays visible.
	if strings.Contains(out, tabHiddenPage) {
		t.Errorf("the only page was hidden:\n%s", out)
	}
}

// A TabView with no onTabChange — core.TabView omits the prop for a nil
// handler, keeping a static strip diff-stable — records no callback.
func TestNoHandlerMeansNoCallbackAttribute(t *testing.T) {
	if out := ExportHTML(tabViewNode(0, "")); strings.Contains(out, "data-ontabchange") {
		t.Errorf("a callback attribute was written for a handler-less TabView:\n%s", out)
	}
}

// A tab page with no box of its own still has to be hidden: renderNode forwards
// the parent's declaration through the transparent branch to the children
// standing in for it.
func TestATransparentPageIsStillHidden(t *testing.T) {
	n := tabViewNode(0, "")
	n.Children[1] = &core.Node{Type: "Fragment", Children: []*core.Node{
		{Type: "Text", Props: map[string]any{"content": "fragment page"}},
	}}
	out := ExportHTML(n)
	at := strings.Index(out, "fragment page")
	if at < 0 {
		t.Fatalf("the fragment page was dropped:\n%s", out)
	}
	// The <span> holding it is what carries the hiding, since the Fragment
	// itself has no element.
	span := strings.LastIndex(out[:at], "<span")
	if span < 0 || !strings.Contains(out[span:at], tabHiddenPage) {
		t.Errorf("a transparent page's children were not hidden:\n%s", out)
	}
}

// The tabs prop after a JSON round trip, which is what a tree handed to this
// exporter from outside Go looks like.
func TestTabsSurviveAJSONShapedPropMap(t *testing.T) {
	n := &core.Node{
		Type: "TabView",
		Props: map[string]any{
			"selectedIndex": float64(1),
			"tabs": []any{
				map[string]any{"label": "One"},
				map[string]any{"label": "Two"},
			},
		},
		Children: []*core.Node{
			{Type: "Text", Props: map[string]any{"content": "first"}},
			{Type: "Text", Props: map[string]any{"content": "second"}},
		},
	}
	out := ExportHTML(n)
	for _, want := range []string{">One<", ">Two<"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
	first := strings.Index(out, "first")
	if !strings.Contains(out[:first], tabHiddenPage) {
		t.Errorf("the float64 selectedIndex was not read:\n%s", out)
	}
}

// The TabView box itself keeps everything any other container gets: its stack
// default, its own Style, and its accessibility attributes. The bar is added to
// that, not substituted for it.
func TestTabViewKeepsItsOwnBox(t *testing.T) {
	n := tabViewNode(0, "")
	n.Style = &core.Style{Background: "#fff"}
	out := ExportHTML(n)
	for _, want := range []string{"background:#fff", "display:flex", "flex-direction:column"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q on the TabView box:\n%s", want, out)
		}
	}
}

// Spacer takes an early return, before the shared attribute assembly, so it is
// the one node type that could silently ignore what its parent imposed on it.
// A Spacer page is an odd thing to build, but "every page but one is hidden"
// has to be true of every page.
func TestASpacerPageIsStillHidden(t *testing.T) {
	n := tabViewNode(0, "")
	n.Children[1] = &core.Node{Type: "Spacer", Props: map[string]any{"size": 40}}
	out := ExportHTML(n)
	at := strings.Index(out, "height:40px")
	if at < 0 {
		t.Fatalf("the Spacer page was dropped:\n%s", out)
	}
	// The whole of the tag the size landed in, so the check reads that one
	// element's declaration list and not a neighbour's.
	open := strings.LastIndex(out[:at], "<div")
	close := strings.Index(out[at:], ">")
	if open < 0 || close < 0 {
		t.Fatalf("unterminated Spacer tag:\n%s", out)
	}
	if !strings.Contains(out[open:at+close], tabHiddenPage) {
		t.Errorf("the Spacer page was not hidden:\n%s", out)
	}
}
