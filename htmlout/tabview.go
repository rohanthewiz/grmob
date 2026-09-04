package htmlout

import (
	"strconv"

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
// target can call into the other. TestRuntimeDrawsTheSameTabChrome pins the
// part that is a contract rather than a look — the roles, the ARIA state and
// the data attributes — because those are what a reader (or a script) reaching
// into either target's output will match on.
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
// # What is deliberately not drawn
//
// The icon half of a core.TabItem. Neither native renders it — Compose's Tab
// is built with `text = { Text(label) }` and the SwiftUI bar with a Text of
// the label — so drawing it here would be a new divergence rather than a
// closed one, this time with the web as the outlier.
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
func renderTabView(b *element.Builder, node *core.Node, attrs []string) {
	e := b.Ele(TagFor(node.Type), attrs...)
	renderTabBar(b, node)
	selected := tabSelectedIndex(node.Props)
	for i, child := range node.Children {
		hidden := tabHiddenPage
		if i == selected {
			hidden = ""
		}
		renderNode(b, child, hidden)
	}
	e.R()
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
func renderTabBar(b *element.Builder, node *core.Node) {
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
		b.Button("type", "button", "role", "tab",
			"aria-selected", ariaSelected,
			"data-tab-index", strconv.Itoa(i),
			"style", style).TE(label)
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
