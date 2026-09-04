package htmlout

import (
	"strconv"
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
//
// Every <button> is cut and then filtered on role="tab", rather than matched
// on a fixed opening prefix: a tab's attribute list grew an id and an
// aria-controls between the tag name and the role, and a helper that pins the
// order of a tab's attributes fails for the wrong reason every time one is
// added.
func tabTags(t *testing.T, out string) []string {
	t.Helper()
	var tags []string
	rest := out
	for {
		at := strings.Index(rest, "<button")
		if at < 0 {
			return tags
		}
		rest = rest[at:]
		end := strings.Index(rest, ">")
		if end < 0 {
			t.Fatalf("unterminated button tag:\n%s", out)
		}
		if tag := rest[:end]; strings.Contains(tag, `role="tab"`) {
			tags = append(tags, tag)
		}
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

// --------------------------------------------------------------------------
// The tab/panel wiring
// --------------------------------------------------------------------------

// panelTag returns the opening tag of the element carrying the given id, so an
// assertion reads that one element's attribute list and not a neighbour's.
// element writes a whole opening tag on one line, which is what makes the
// slice unambiguous.
func panelTag(t *testing.T, out, id string) string {
	t.Helper()
	at := strings.Index(out, `id="`+id+`"`)
	if at < 0 {
		return ""
	}
	open := strings.LastIndex(out[:at], "<")
	end := strings.Index(out[at:], ">")
	if open < 0 || end < 0 {
		t.Fatalf("unterminated tag around %s:\n%s", id, out)
	}
	return out[open : at+end]
}

// The relationship a bare tablist cannot express: which region of the screen
// each tab governs, and which tab names each region.
//
// The literal ids are asserted rather than merely their agreement, because
// they are shared with the WASM runtime — "a tab and its page point at each
// other" in wasm/verify/tabview_test.mjs asserts the same strings for the same
// tree. Both derive them from the node path (tabScope), so a TabView at "root"
// is "grmob-root-…" on both web targets; ids that merely happened to be
// internally consistent on each side would let the two drift apart while every
// test still passed.
func TestTabsAndPanelsPointAtEachOther(t *testing.T) {
	out := ExportHTML(tabViewNode(0, "cb_9"))

	tabs := tabTags(t, out)
	if len(tabs) != 2 {
		t.Fatalf("want two tabs, got %d:\n%s", len(tabs), out)
	}
	for i, tab := range tabs {
		id := "grmob-root-tab-" + strconv.Itoa(i)
		panel := "grmob-root-panel-" + strconv.Itoa(i)
		if !strings.Contains(tab, `id="`+id+`"`) {
			t.Errorf("tab %d does not carry id %q:\n%s", i, id, tab)
		}
		if !strings.Contains(tab, `aria-controls="`+panel+`"`) {
			t.Errorf("tab %d does not control %q:\n%s", i, panel, tab)
		}

		page := panelTag(t, out, panel)
		if page == "" {
			t.Fatalf("nothing answers to %q, so the tab's aria-controls dangles:\n%s", panel, out)
		}
		if !strings.Contains(page, `role="tabpanel"`) {
			t.Errorf("page %d is referenced as a panel but is not one:\n%s", i, page)
		}
		if !strings.Contains(page, `aria-labelledby="`+id+`"`) {
			t.Errorf("page %d is not named by its tab:\n%s", i, page)
		}
	}
}

// The hidden pages are wired too. display:none takes a page out of the
// accessibility tree, so nothing is announced from it either way — but the
// wiring is a property of the document's structure, not of the current
// selection, and a version of it that had to be built and torn down on every
// switch would be one more thing for syncTabView to get wrong.
func TestEveryPageIsWiredNotOnlyTheVisibleOne(t *testing.T) {
	out := ExportHTML(tabViewNode(0, ""))
	if strings.Count(out, `role="tabpanel"`) != 2 {
		t.Errorf("want both pages wired as panels:\n%s", out)
	}
}

// Ids are document-global, so two TabViews in one document must not both call
// their first tab "tab-0". The scope comes from the node path, which is
// already unique per element on both web targets.
func TestPanelIDsAreScopedPerTabView(t *testing.T) {
	n := &core.Node{Type: "Column", Children: []*core.Node{
		tabViewNode(0, ""),
		tabViewNode(0, ""),
	}}
	out := ExportHTML(n)

	seen := map[string]int{}
	rest := out
	for {
		at := strings.Index(rest, `id="`)
		if at < 0 {
			break
		}
		rest = rest[at+len(`id="`):]
		end := strings.Index(rest, `"`)
		if end < 0 {
			t.Fatalf("unterminated id:\n%s", out)
		}
		seen[rest[:end]]++
	}
	if len(seen) == 0 {
		t.Fatalf("no ids at all, so this test proves nothing:\n%s", out)
	}
	for id, n := range seen {
		if n > 1 {
			t.Errorf("id %q appears %d times; the two TabViews share a scope:\n%s", id, n, out)
		}
	}
	// And the scopes are the two node paths, not a counter that happens to be
	// unique — which is what keeps them equal to the runtime's.
	for _, want := range []string{"grmob-root-0-tab-0", "grmob-root-1-tab-0"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q — the scope is not the node path:\n%s", want, out)
		}
	}
}

// role="tabpanel" replaces the role a browser already gave an element rather
// than adding to it, so a page that is a Button, an Image or an input is left
// alone — and its tab must not point at it either, since an aria-controls
// naming an element that is not a panel is a lie about the document.
func TestARoledPageIsNotMadeAPanel(t *testing.T) {
	for _, page := range []*core.Node{
		{Type: "Button", Props: map[string]any{"label": "press"}},
		{Type: "Image", Props: map[string]any{"src": "a.png"}},
		{Type: "Input", Props: map[string]any{"value": "v"}},
	} {
		n := tabViewNode(0, "")
		n.Children[1] = page
		out := ExportHTML(n)

		if strings.Count(out, `role="tabpanel"`) != 1 {
			t.Errorf("a %s page was given the panel role:\n%s", page.Type, out)
		}
		tabs := tabTags(t, out)
		if len(tabs) != 2 {
			t.Fatalf("want two tabs, got %d:\n%s", len(tabs), out)
		}
		if strings.Contains(tabs[1], "aria-controls") {
			t.Errorf("the tab of a %s page still claims to control a panel:\n%s", page.Type, tabs[1])
		}
		// The eligible page beside it is unaffected: opting one page out is
		// not opting the tab set out.
		if !strings.Contains(tabs[0], `aria-controls="grmob-root-panel-0"`) {
			t.Errorf("the sibling tab lost its wiring:\n%s", tabs[0])
		}
	}
}

// The author took the page out of the accessibility tree on purpose; naming it
// as a panel would assert a relationship they severed.
func TestAnAccessibilityHiddenPageIsNotMadeAPanel(t *testing.T) {
	n := tabViewNode(0, "")
	n.Children[1].Style = &core.Style{AccessibilityHidden: true}
	out := ExportHTML(n)

	if strings.Count(out, `role="tabpanel"`) != 1 {
		t.Errorf("an aria-hidden page was given the panel role:\n%s", out)
	}
	tabs := tabTags(t, out)
	if strings.Contains(tabs[1], "aria-controls") {
		t.Errorf("the tab of an aria-hidden page still claims to control it:\n%s", tabs[1])
	}
}

// aria-labelledby wins over aria-label in the accessible-name calculation, so
// writing it unconditionally would discard a name the app author chose. The
// tab still points at the panel; only the naming is left alone.
func TestAPageThatNamesItselfKeepsItsName(t *testing.T) {
	n := tabViewNode(0, "")
	n.Children[0].Style = &core.Style{AccessibilityLabel: "Your home feed"}
	out := ExportHTML(n)

	page := panelTag(t, out, "grmob-root-panel-0")
	if page == "" {
		t.Fatalf("the page was not wired at all:\n%s", out)
	}
	if !strings.Contains(page, `aria-label="Your home feed"`) {
		t.Errorf("the author's label was dropped:\n%s", page)
	}
	if strings.Contains(page, "aria-labelledby") {
		t.Errorf("the tab's name was allowed to override the author's:\n%s", page)
	}
	if !strings.Contains(page, `role="tabpanel"`) {
		t.Errorf("the page stopped being a panel because it had a name:\n%s", page)
	}
	if tabs := tabTags(t, out); !strings.Contains(tabs[0], `aria-controls="grmob-root-panel-0"`) {
		t.Errorf("the tab stopped pointing at a page that named itself:\n%s", tabs[0])
	}
}

// A Theme (core.WithTheme) or a single-child Fragment has no box of its own
// here, but exactly one element still stands in for the page, and that element
// is the panel. This is where the two web targets differ in *which* element
// carries the wiring — the runtime boxes both node types — while agreeing on
// the ids and on the fact that tab 1 controls page 1.
func TestATransparentPageIsWiredThroughToItsElement(t *testing.T) {
	n := tabViewNode(0, "")
	n.Children[1] = &core.Node{Type: "Theme", Children: []*core.Node{
		{Type: "Column", Children: []*core.Node{{Type: "Text", Props: map[string]any{"content": "themed"}}}},
	}}
	out := ExportHTML(n)

	page := panelTag(t, out, "grmob-root-panel-1")
	if page == "" {
		t.Fatalf("the themed page was not wired:\n%s", out)
	}
	if !strings.Contains(page, `role="tabpanel"`) {
		t.Errorf("the element standing in for the page is not a panel:\n%s", page)
	}
	if tabs := tabTags(t, out); !strings.Contains(tabs[1], `aria-controls="grmob-root-panel-1"`) {
		t.Errorf("the tab does not point at the transparent page:\n%s", tabs[1])
	}
}

// More than one element stands in for the page, so there is no single one to
// name: the id would have to be written twice, which is invalid HTML, or
// nowhere. Nowhere, and the tab says nothing rather than pointing at one of
// them arbitrarily.
func TestAMultiElementTransparentPageIsNotWired(t *testing.T) {
	n := tabViewNode(0, "")
	n.Children[1] = &core.Node{Type: "Fragment", Children: []*core.Node{
		{Type: "Text", Props: map[string]any{"content": "one"}},
		{Type: "Text", Props: map[string]any{"content": "two"}},
	}}
	out := ExportHTML(n)

	if strings.Count(out, `role="tabpanel"`) != 1 {
		t.Errorf("a page with no single element was wired anyway:\n%s", out)
	}
	if tabs := tabTags(t, out); strings.Contains(tabs[1], "aria-controls") {
		t.Errorf("the tab points at one of several elements:\n%s", tabs[1])
	}
	// The hiding still reaches every one of them; only the naming is skipped.
	if strings.Count(out, tabHiddenPage) != 2 {
		t.Errorf("the fragment's children were not both hidden:\n%s", out)
	}
}

// A tabpanel with no tab is not part of a tab set, and there would be nothing
// for its aria-controls to sit on.
func TestAPageWithNoTabIsNotAPanel(t *testing.T) {
	n := tabViewNode(0, "")
	n.Children = append(n.Children, &core.Node{Type: "Column"})
	out := ExportHTML(n)

	if strings.Count(out, `role="tabpanel"`) != 2 {
		t.Errorf("the third page, which no tab names, was wired:\n%s", out)
	}
	if strings.Contains(out, "grmob-root-panel-2") {
		t.Errorf("an id was written for a page with no tab:\n%s", out)
	}
}

// No bar, no tab set, nothing to wire.
func TestNoTabsMeansNoPanels(t *testing.T) {
	n := &core.Node{Type: "TabView", Children: []*core.Node{
		{Type: "Column", Children: []*core.Node{{Type: "Text", Props: map[string]any{"content": "only"}}}},
	}}
	out := ExportHTML(n)
	if strings.Contains(out, "tabpanel") || strings.Contains(out, "grmob-root") {
		t.Errorf("a TabView with no tabs wired a panel:\n%s", out)
	}
}

// Spacer takes an early return, before the shared attribute assembly, so it is
// the one node type whose element could silently ignore what its parent put on
// it — the hiding *and* the wiring travel through the same channel.
func TestASpacerPageIsStillWired(t *testing.T) {
	n := tabViewNode(0, "")
	n.Children[1] = &core.Node{Type: "Spacer", Props: map[string]any{"size": 40}}
	out := ExportHTML(n)

	page := panelTag(t, out, "grmob-root-panel-1")
	if page == "" {
		t.Fatalf("the Spacer page was not wired:\n%s", out)
	}
	if !strings.Contains(page, `role="tabpanel"`) || !strings.Contains(page, "height:40px") {
		t.Errorf("the Spacer page lost either its size or its role:\n%s", page)
	}
}
