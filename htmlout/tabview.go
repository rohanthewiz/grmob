package htmlout

import (
	"strconv"
	"strings"

	"github.com/rohanthewiz/element"
	"github.com/rohanthewiz/grmob/core"
)

// The tab bar and page selection of a TabView, on the HTML side.
//
// # What this closes
//
// core.TabView's wire contract is a tabs prop (label/icon pairs), a controlled
// selectedIndex, an optional onTabChange callback ID, and one child per page.
// Both natives consume all four: Renderer.kt draws a Material TabRow above the
// selected page, Renderer.swift a hand-rolled bar above the same. Neither DOM
// target read any of them — a TabView exported as a bare box holding *every*
// page, with no bar and no way to switch — so an app whose whole navigation is
// a TabView had no navigation at all on the web, and its screens were stacked
// one under the other.
//
// The WASM runtime grew the same bar and the same hiding in the same pass;
// buildTabBar and syncTabView in grmob-runtime.js are its half. The two are
// authored twice rather than shared, exactly as the Modal chassis is: a
// declaration list is a string here and a property object there, and neither
// target can call into the other. TestRuntimeDrawsTheSameTabChrome and
// TestRuntimeWiresTabsToPanels pin the part that is a contract rather than a
// look — the roles, the ARIA state, the data attributes and the shape of the
// element ids — because those are what a reader (or a screen reader, or a
// script) reaching into either target's output will match on.
//
// # The chrome is not part of the node tree
//
// The bar is an element neither Go nor the reconciler knows about, so it
// carries no data-node-path, and it is marked with data-grmob-chrome so both
// the runtime's index arithmetic and wasm/verify's replay can tell it from a
// child. It always precedes the pages: the runtime resolves a patch's child
// slot by DOM position, and "the chrome comes first, the node children follow"
// is the rule that keeps that arithmetic a fixed offset rather than a search.
// The same order is what a screen reader and a no-CSS rendering want anyway.
//
// # The tabs and the pages point at each other
//
// A role="tablist" of role="tab"s is a well-formed strip on its own, but it
// says nothing about *which region of the screen* each tab governs. That
// relationship is aria-controls and aria-labelledby, both IDREFs, so it cannot
// be expressed without element ids — see tabScope for where they come from and
// why that answer is the node path, and tabPanelBoxes for the three cases in
// which a page is left unwired rather than described falsely.
//
// This is the one part of the chrome that lands on *node* elements rather than
// on chrome of the exporter's own making, which is what made it a separate
// decision from the bar: every attribute written onto a node element is one an
// author's Style could also be asked to set. Nothing collides today — no
// core.Style field maps onto id, role or aria-labelledby — and the two places
// where the author's intent could be overridden are both deferred to them: a
// page that names itself keeps its name, and a page they took out of the
// accessibility tree is not dragged back into it.
//
// # What is deliberately not drawn
//
// The icon half of a core.TabItem. Neither native renders it — Compose's Tab
// is built with `text = { Text(label) }` and the SwiftUI bar with a Text of
// the label — so drawing it here would be a new divergence rather than a
// closed one, this time with the web as the outlier.
//
// A tabindex on the panel, which the ARIA authoring practices suggest for a
// panel holding nothing focusable. tabindex changes the page's real tab order,
// which is a behavioral change to an app author's node rather than a statement
// about it; this wiring is deliberately semantic only.
const (
	// The bar: a row of equal-width tabs with a hairline under it. The
	// hairline is a neutral gray at low alpha rather than a theme color,
	// because it has to sit on whatever background the app painted and
	// nothing at this level knows what that is.
	tabBarChassis = "display:flex; flex-direction:row; align-items:stretch; " +
		"flex-shrink:0; border-bottom:1px solid rgba(128,128,128,0.35)"

	// One tab. `flex:1 1 0` is Android's equal-width TabRow and SwiftUI's
	// .frame(maxWidth: .infinity); the rest is a <button> talked out of
	// looking like one, since the natives' tabs are labels with an indicator
	// and not buttons with a chrome of their own.
	//
	// color:inherit and font:inherit are what make the bar theme-neutral: the
	// tabs take the app's own text color and face, so the bar reads correctly
	// on a dark surface without this file knowing there is one. currentColor
	// in the selected list below is the other half of that.
	tabChassis = "flex:1 1 0; padding:12px 8px 10px; border:none; " +
		"border-bottom:2px solid transparent; background:none; color:inherit; " +
		"font:inherit; text-align:center; cursor:pointer; opacity:0.6"

	// The selected tab, appended after the base list so its three
	// declarations win the browser's last-one-wins parse. The underline is
	// Android's TabRow indicator and the Rectangle in Renderer.swift's bar;
	// the weight and the full opacity are that bar's .semibold and its
	// accent-vs-secondary split, spelled without an accent color.
	tabSelectedChassis = "opacity:1; font-weight:600; border-bottom-color:currentColor"

	// display:none, appended to a page that is not the selected one.
	//
	// Hidden rather than dropped, which is where the two DOM targets have to
	// agree and where they both differ from the natives. The runtime cannot
	// drop a page: patch TargetIDs are positional ("root/1/0"), so its DOM
	// has to stay isomorphic to the node tree. This exporter could drop one —
	// it is a static snapshot with no patches to address — but an export that
	// silently lost four fifths of an app's screens would be a worse
	// document, and a divergence between the two web targets for no gain.
	tabHiddenPage = "display:none"
)

// renderTabView writes the bar and then the pages, the selected one visible
// and the rest hidden.
//
// attrs is the node's own attribute list, already assembled by renderNode —
// the TabView box keeps its style, its callbacks and its accessibility
// attributes like any other container; all this adds is what goes inside.
func renderTabView(b *element.Builder, node *core.Node, attrs []string, path string) {
	scope := tabScope(path)
	// Which pages can carry the panel half of the wiring, decided once and
	// read by both loops. The bar has to know before it is written — a tab
	// whose page was skipped must not carry an aria-controls pointing at an
	// id nothing answers to — and the bar is written first.
	panels := tabPanelBoxes(node)

	e := b.Ele(TagFor(node.Type), attrs...)
	renderTabBar(b, node, scope, panels)
	selected := tabSelectedIndex(node.Props)
	for i, child := range node.Children {
		from := imposed{}
		if i != selected {
			from.decl = tabHiddenPage
		}
		if box := panels[i]; box != nil {
			from.attrs = tabPanelAttrs(box, scope, i)
		}
		renderNode(b, child, from, childPath(path, i))
	}
	e.R()
}

// tabScope is the id prefix every element id inside one TabView is built from.
//
// # Why the node path
//
// aria-controls and aria-labelledby are IDREFs, so the wiring cannot be
// expressed without ids, and ids are document-global: two TabViews in one
// document must not both call their first tab "tab-0". The node path is the
// identity that is already unique per element on both web targets, and using
// it means the ids are not merely well-formed but *identical* across the two —
// the runtime derives its own from the data-node-path it wrote (tabScope in
// grmob-runtime.js), and this exporter from the path renderNode walked. So the
// contract the pair share is the literal string, not just its shape.
//
// The uniqueness is exactly the uniqueness the addressing scheme already rests
// on: if two live elements could share a data-node-path, every patch aimed at
// either of them is already going to the wrong one, and duplicate ids would be
// the least of it.
//
// The slashes become dashes because an id is also a CSS selector fragment in
// every tool that reads a document, and "#grmob-root/1-tab-0" needs escaping
// to be one. Nothing is lost: a path is "root" followed by digits, so no
// dash can appear in it and no two paths can collapse onto one scope.
func tabScope(path string) string {
	return "grmob-" + strings.ReplaceAll(path, "/", "-")
}

// tabID and panelID are the two ids one tab/page pair uses to point at each
// other. Restated in grmob-runtime.js and pinned by
// TestRuntimeWiresTabsToPanels; see tabScope.
func tabID(scope string, i int) string   { return scope + "-tab-" + strconv.Itoa(i) }
func panelID(scope string, i int) string { return scope + "-panel-" + strconv.Itoa(i) }

// tabPanelBoxes decides, for each of a TabView's children, which node (if any)
// will carry the panel half of the ARIA wiring. Index i holds the node whose
// element stands in for page i, or nil when the page is not wired at all.
//
// The three ways a page opts out, each one a case where wiring it would say
// something false:
//
//	no tab at index i     a tabpanel with no tab is not part of a tab set, and
//	                      the aria-controls would have nothing to sit on. A
//	                      TabView with no tabs prop draws no bar (see
//	                      renderTabBar) and so wires nothing at all.
//	a roled element       role="tabpanel" replaces the role the browser gave
//	                      the element rather than adding to it, so a Button,
//	                      an Image or an input page is left as it is — see
//	                      genericTags in tag.go.
//	an authored role      the page already says what it is
//	                      (core.AccessibilityRole), and replacing that is the
//	                      same theft as replacing the browser's — the author's
//	                      word beats the wiring's, which is the call
//	                      aria-labelledby already makes against
//	                      AccessibilityLabel below.
//	a self-roling type    a Modal is a dialog by virtue of being a Modal, with
//	                      no Style to say so; the same theft, one layer down.
//	AccessibilityHidden   the author took the page out of the accessibility
//	                      tree on purpose; naming it as a panel would assert a
//	                      relationship they severed.
//
// Transparency is resolved rather than refused. A page that is a core.WithTheme
// (a Theme node) or a single-child Fragment has no box of its own here, but
// exactly one element still stands in for it, and that element is the panel.
// More than one child, or none, and there is no single element to name — the
// id would have to be written twice, which is invalid, or nowhere.
//
// The runtime needs none of this resolution: it boxes Fragment and Theme (see
// transparentTypes), so page i is always exactly the element at child slot i.
// The rule it does share is the other two, and it applies them to the element
// rather than to the node.
func tabPanelBoxes(node *core.Node) []*core.Node {
	boxes := make([]*core.Node, len(node.Children))
	tabs := len(tabLabels(node.Props))
	for i, child := range node.Children {
		if i >= tabs {
			continue
		}
		boxes[i] = tabPanelBox(child)
	}
	return boxes
}

// tabPanelBox returns the node whose element will stand in for this page, or
// nil when there is no single element of the right kind to name. See
// tabPanelBoxes for what each answer means.
func tabPanelBox(page *core.Node) *core.Node {
	switch {
	case page == nil:
		return nil
	case IsTransparent(page.Type):
		if len(page.Children) != 1 {
			return nil
		}
		return tabPanelBox(page.Children[0])
	case page.Style != nil && page.Style.AccessibilityHidden:
		return nil
	case page.Style != nil && page.Style.AccessibilityRole != core.RoleNone:
		return nil
	// A node type that states its own role, the way a Modal is a dialog (see
	// modalSemantics). The tag is generic and the Style is nil, so neither
	// guard above catches it, and wiring it would write a second role
	// attribute onto the same element. The runtime's canBeTabPanel gets this
	// for free: it reads the role back off the live element, where the
	// chassis already put it.
	case CarriesOwnRole(page.Type):
		return nil
	case !IsGenericTag(TagFor(page.Type)):
		return nil
	}
	return page
}

// tabPanelAttrs is the panel half of the wiring, as attributes the TabView
// imposes on the page's element.
//
// aria-labelledby is dropped when the page names itself. A tab panel is
// normally named by its tab — that is the whole point of the reference — but
// core.Style.AccessibilityLabel is an explicit act by the app author, and
// aria-labelledby would win over the aria-label it produced (the reference
// takes precedence in the accessible-name calculation), silently discarding
// the name they chose. The tab still points *at* the panel either way; only
// the name is left to the author.
func tabPanelAttrs(box *core.Node, scope string, i int) []string {
	attrs := []string{
		"id", panelID(scope, i),
		"role", "tabpanel",
	}
	if box.Style == nil || box.Style.AccessibilityLabel == "" {
		attrs = append(attrs, "aria-labelledby", tabID(scope, i))
	}
	return attrs
}

// renderTabBar writes the tab strip, or nothing at all when the node carries
// no tabs.
//
// Nothing at all, rather than an empty bar: the chassis draws a hairline, and
// a rule across the top of a TabView that has no tabs to name is a visible
// artifact of an absent prop. It also keeps the chrome offset honest on the
// runtime side, where "how many leading chrome children are there" is counted
// rather than assumed.
//
// data-ontabchange carries the callback ID, in the same spirit as the
// data-onclick family: the export cannot dispatch, so it records which handler
// the edge belongs to and leaves the wiring to whatever loads the document.
// The ID sits on the bar and the index on each tab, because one callback
// serves every tab and the argument is what distinguishes them — which is
// exactly the shape core.OnTabChange has.
func renderTabBar(b *element.Builder, node *core.Node, scope string, panels []*core.Node) {
	labels := tabLabels(node.Props)
	if len(labels) == 0 {
		return
	}
	attrs := []string{
		"data-grmob-chrome", "tabbar",
		"role", "tablist",
		"style", tabBarChassis,
	}
	if id := getStr(node.Props["onTabChange"]); id != "" {
		attrs = append(attrs, "data-ontabchange", id)
	}
	bar := b.Div(attrs...)
	selected := tabSelectedIndex(node.Props)
	for i, label := range labels {
		style, ariaSelected := tabChassis, "false"
		if i == selected {
			style, ariaSelected = addDecl(tabChassis, tabSelectedChassis), "true"
		}
		// type="button" so a bar inside a <form> cannot submit it. TE
		// escapes the label, which is app data like any Text content.
		//
		// The id is written whether or not this tab controls a panel: it costs
		// nothing, an id names nothing on its own, and it means the panel half
		// can point back here without the two halves having to agree twice.
		tabAttrs := []string{
			"type", "button",
			"id", tabID(scope, i),
			"role", "tab",
			"aria-selected", ariaSelected,
			"data-tab-index", strconv.Itoa(i),
		}
		// aria-controls only where there is a panel to control. A dangling
		// IDREF is worse than an absent one — a screen reader announcing "tab,
		// controls" and finding nothing is a lie about the document, whereas a
		// tab with no aria-controls is merely a tab that has not said which
		// region it governs. See tabPanelBoxes for the three ways a page opts
		// out of being one.
		if i < len(panels) && panels[i] != nil {
			tabAttrs = append(tabAttrs, "aria-controls", panelID(scope, i))
		}
		tabAttrs = append(tabAttrs, "style", style)
		b.Button(tabAttrs...).TE(label)
	}
	bar.R()
}

// tabSelectedIndex reads the controlled selection out of the props.
//
// Deliberately unclamped, and the out-of-range case is not an error here: an
// index past the end selects no tab and shows no page, which is what both
// natives do (Renderer.swift's `children.indices.contains(selected)` and
// Renderer.kt's `getOrNull(selected)` for the page, and the Swift bar's plain
// `i == selected` for the indicator). Clamping would light up a tab the app
// did not ask for.
//
// An absent prop reads as 0, matching core.TabViewNode's zero value: a TabView
// built with no SelectedIndex shows its first page on every target.
func tabSelectedIndex(props map[string]any) int {
	switch v := props["selectedIndex"].(type) {
	case int:
		return v
	case float64:
		// A tree that has been through JSON — the props map is `any`-valued,
		// and JSON has one number type. Not the path core.TabView takes, but
		// this exporter is handed trees from elsewhere too.
		return int(v)
	}
	return 0
}

// tabLabels reads the tab strip out of the tabs prop.
//
// core.TabView builds []map[string]string, which is the shape that matters;
// []any of maps is the same list after a JSON round trip, and is accepted for
// the reason tabSelectedIndex accepts a float64. Anything else yields no
// labels, and therefore no bar, rather than a partially drawn one.
//
// The icon is read past on purpose — see the package comment above: neither
// native draws it.
func tabLabels(props map[string]any) []string {
	switch tabs := props["tabs"].(type) {
	case []map[string]string:
		out := make([]string, 0, len(tabs))
		for _, t := range tabs {
			out = append(out, t["label"])
		}
		return out
	case []any:
		out := make([]string, 0, len(tabs))
		for _, t := range tabs {
			switch item := t.(type) {
			case map[string]string:
				out = append(out, item["label"])
			case map[string]any:
				out = append(out, getStr(item["label"]))
			default:
				return nil
			}
		}
		return out
	}
	return nil
}
