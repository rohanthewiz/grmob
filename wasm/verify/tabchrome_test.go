package main

import (
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/htmlout"
)

// The TabView chrome is authored twice — htmlout/tabview.go writes a static
// document, grmob-runtime.js builds live elements — because a CSS declaration
// list is a string on one side and a property object on the other, and neither
// target can call into the other. That is the same arrangement the Modal
// chassis has lived under since it existed.
//
// What can be pinned, and is worth pinning, is the half that is a *contract*
// rather than a look: the roles, the ARIA state and the data attributes. Those
// are what a reader, a screen reader or a script reaching into either target's
// output matches on, and a drift in one of them is silent — the bar still draws,
// still switches, and simply stops being the same thing on the two web targets.
//
// A substring check rather than a table parse, for the reason
// TestRuntimeAppliesTheStackDefault is one: what is being pinned here is that
// particular expressions appear in the source, not the contents of a lookup.
func TestRuntimeDrawsTheSameTabChrome(t *testing.T) {
	src := runtimeSource(t)
	for _, want := range []struct{ expr, why string }{
		{`bar.dataset.grmobChrome = "tabbar"`,
			"the marker that tells a bar from a page — chromeOffset and wasm/verify's replay both read it"},
		{`bar.setAttribute("role", "tablist")`, "the bar's role"},
		{`button.setAttribute("role", "tab")`, "each tab's role"},
		{`button.dataset.tabIndex = i`, "the index a tab carries, which is the argument onTabChange takes"},
		{`"aria-selected", on ? "true" : "false"`, "the selection state, restated on every sync"},
		{`bar.setAttribute("data-ontabchange", cbId)`, "the callback ID mirrored onto the bar"},
	} {
		if !strings.Contains(src, want.expr) {
			t.Errorf("grmob-runtime.js: %q not found — %s. htmlout/tabview.go writes it, so the "+
				"two web targets no longer draw the same tab bar", want.expr, want.why)
		}
	}
}

// The other half of the contract: the bar is chrome, so nothing in the runtime
// may give it a data-node-path, and both places that turn a node child index
// into a DOM child index have to shift past it.
//
// The two index sites are the failure this pass could most easily have
// introduced and the one no visual check would catch: a page added to a TabView
// landing in front of the bar, and every path after it one slot out of step
// with the element that answers to it.
func TestRuntimeShiftsChildIndicesPastTheChrome(t *testing.T) {
	src := runtimeSource(t)
	for _, want := range []string{
		// "add": the slot a path names, shifted past the chrome.
		`parent.children[index + chromeOffset(parent)]`,
		// "add-child": the new child's node index, which is the DOM count
		// minus the chrome ahead of it.
		`el.children.length - chromeOffset(el)`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("grmob-runtime.js: %q not found — a TabView's bar occupies a DOM child slot, "+
				"so a patch addressing a child by node index lands one element late", want)
		}
	}
}

// The tab/panel wiring: the half of the ARIA contract that is a *relationship*
// rather than a state.
//
// A tab bar that is a well-formed tablist still says nothing about which region
// of the screen each tab governs, and that relationship cannot be expressed
// without ids — aria-controls and aria-labelledby are IDREFs. The ids are
// therefore derived rather than invented, and derived the same way on both web
// targets: from the node path, so a TabView at "root/1" calls its first tab
// "grmob-root-1-tab-0" in the exported document and in the live DOM alike.
//
// Pinned as expressions rather than as behavior because that is all a Go test
// can reach — the derivation is three lines of JavaScript. Each target's own
// suite then asserts the concrete strings for a TabView at "root"
// (TestTabsAndPanelsPointAtEachOther here in Go, "a tab and its page point at
// each other" in tabview_test.mjs), so the two agree on the format here and on
// the literal there.
func TestRuntimeWiresTabsToPanels(t *testing.T) {
	src := runtimeSource(t)
	for _, want := range []struct{ expr, why string }{
		{`"grmob-" + (el.getAttribute("data-node-path") || "").replace(/\//g, "-")`,
			"the id scope, derived from the node path so the ids match htmlout's character for character"},
		{"`${scope}-tab-${i}`", "a tab's id, which its panel's aria-labelledby names"},
		{"`${scope}-panel-${i}`", "a page's id, which its tab's aria-controls names"},
		{`page.setAttribute("role", "tabpanel")`,
			"the panel role, written on every sync because eligibility can change under a patch"},
		{`page.getAttribute("role") === "tabpanel"`,
			"and removed on every sync — but only when the standing value is the wiring's own, " +
				"since core.AccessibilityRole writes the same attribute and its value is the " +
				"author's (see canBeTabPanel)"},
		{`setOrRemove(tab, "aria-controls",`,
			"the tab -> panel reference, omitted rather than left dangling when the page is not a panel"},
		{`setOrRemove(page, "aria-labelledby", wired && !named ? tabId(scope, i) : "")`,
			"the panel -> tab reference, dropped when the page carries an accessibility label of its own"},
	} {
		if !strings.Contains(src, want.expr) {
			t.Errorf("grmob-runtime.js: %q not found — %s. htmlout/tabview.go writes it, so the "+
				"two web targets no longer describe the same tab set", want.expr, want.why)
		}
	}
}

// The set of tags a role= attribute may be written onto, which is what decides
// whether a page can be a panel at all.
//
// Go is the authority (genericTags in htmlout/tag.go) and the runtime restates
// it, so the two are compared here rather than remembered — the same treatment
// the tag and input-type tables get, and for the same reason: a page whose tag
// is eligible on one target and not the other is a tab set that means one thing
// in the exported document and another in the live one.
//
// Parsed with a regexp rather than through parseRuntimeTable, which reads
// key/value object literals; this one is a flat array of strings.
func TestRuntimeGenericTagsMatchGo(t *testing.T) {
	src := runtimeSource(t)

	m := regexp.MustCompile(`const GENERIC_TAGS = new Set\(\[([^\]]*)\]\);`).FindStringSubmatch(src)
	if m == nil {
		t.Fatalf("grmob-runtime.js: no `const GENERIC_TAGS = new Set([...])` found — if it was " +
			"renamed or spread over several lines, update this test rather than deleting it")
	}
	var got []string
	for _, q := range regexp.MustCompile(`"([^"]*)"`).FindAllStringSubmatch(m[1], -1) {
		got = append(got, q[1])
	}
	sort.Strings(got)

	want := htmlout.GenericTags()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GENERIC_TAGS is %v; htmlout.GenericTags() is %v — a tab page eligible for "+
			"role=\"tabpanel\" on one web target and not the other", got, want)
	}
}
