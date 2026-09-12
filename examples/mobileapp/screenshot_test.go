package mobileapp

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/internal/shotclaims"
	"github.com/rohanthewiz/grmob/mobile"
)

// findNode is the general walk this file needs and app_test.go's findProp
// is not: the question here is "does any node say this", which no search
// keyed on a node type and a prop name can ask.
func findNode(n *node, pred func(*node) bool) *node {
	if n == nil {
		return nil
	}
	if pred(n) {
		return n
	}
	for _, c := range n.Children {
		if found := findNode(c, pred); found != nil {
			return found
		}
	}
	return nil
}

// showsText reports whether anything in the tree puts this text on the
// screen. See shotclaims.Shown for why the search reaches into nested prop
// values — this screen's tab strip is exactly that shape, a list of tab
// objects under one prop of the TabView.
func showsText(n *node, s string) bool {
	return findNode(n, func(n *node) bool {
		return shotclaims.Shown(n.Props, s)
	}) != nil
}

func currentTree(t *testing.T) *node {
	t.Helper()
	var root node
	if err := json.Unmarshal([]byte(mobile.RenderInitial()), &root); err != nil {
		t.Fatalf("tree is not valid JSON: %v", err)
	}
	return &root
}

// TestTheTabsScreenshotStillShowsWhatItClaims drives this demo to the state
// docs/images/tabs-list.png was taken in and asserts the picture's strings
// are still on the screen.
//
// See internal/shotclaims for why a screenshot needs holding at all. What
// this arm adds to the manifest's own checks is the only thing a manifest
// cannot do for itself: it renders the app.
//
// The two dispatches are the two gestures in the picture — the Feed tab,
// then the third row — and both go through the bridge the native shells
// call, which is what makes "the third article row tapped" a description
// of something that happened rather than of state that was written in.
func TestTheTabsScreenshotStillShowsWhatItClaims(t *testing.T) {
	core.ClearConcerns()
	defer assertNoConcerns(t)

	claims := shotclaims.For("examples/mobileapp")
	if len(claims) == 0 {
		t.Fatal("no claim names examples/mobileapp, and " +
			"docs/images/tabs-list.png is a picture of it. Either the shot " +
			"has gone or the manifest has lost its row; both leave this test " +
			"asserting nothing.")
	}

	// The Feed tab is the third, and it is named rather than indexed so
	// that a tab inserted before it moves this with it instead of
	// silently photographing a different screen.
	mobile.TriggerIntCallback(mustFindT(t, "TabView", "onTabChange"), feedTabIndex(t))
	tapArticle(t, 3)

	root := currentTree(t)
	for _, c := range claims {
		for _, want := range c.Shows {
			if !showsText(root, want) {
				t.Errorf("docs/images/%s shows %q and this app no longer "+
					"does, with %s.\n\n"+
					"The picture is now a statement about a version of this "+
					"screen that does not exist. Re-take it (see wasm/shots) "+
					"and move the claim and the README's caption with it.",
					c.File, want, c.State)
			}
		}
	}
}

// feedTabIndex is the position of the tab labelled "Feed" in the strip.
func feedTabIndex(t *testing.T) int {
	t.Helper()
	strip := findNode(currentTree(t), func(n *node) bool { return n.Type == "TabView" })
	if strip == nil {
		t.Fatal("no TabView in the current tree")
	}
	tabs, ok := strip.Props["tabs"].([]any)
	if !ok {
		t.Fatalf("the TabView carries no tab list: %#v", strip.Props)
	}
	for i, raw := range tabs {
		if shotclaims.Shown(raw, "Feed") {
			return i
		}
	}
	t.Fatalf("no tab labelled Feed among %d: %#v", len(tabs), tabs)
	return 0
}

// tapArticle taps the row for "Article n".
//
// Innermost, for the reason examples/todoapp's row search is: every
// ancestor of a row also shows the row's title and also carries a click,
// so the outermost match is the list and tapping it would select whatever
// its own handler selects. Children are asked before their parent, so the
// first node to answer is the deepest one that can.
func tapArticle(t *testing.T, n int) {
	t.Helper()
	title := "Article " + strconv.Itoa(n)
	row := innermostTappableShowing(currentTree(t), title)
	if row == nil {
		t.Fatalf("no tappable row showing %q", title)
	}
	mobile.TriggerCallback(row.Props["onClick"].(string))
}

func innermostTappableShowing(n *node, title string) *node {
	if n == nil {
		return nil
	}
	for _, c := range n.Children {
		if found := innermostTappableShowing(c, title); found != nil {
			return found
		}
	}
	if _, clickable := n.Props["onClick"].(string); clickable && showsText(n, title) {
		return n
	}
	return nil
}
