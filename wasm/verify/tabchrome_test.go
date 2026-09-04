package main

import (
	"strings"
	"testing"
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
