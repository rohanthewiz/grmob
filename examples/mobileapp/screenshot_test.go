package mobileapp

import (
	"strconv"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/grmobtest"
	"github.com/rohanthewiz/grmob/internal/shotclaims"
	"github.com/rohanthewiz/grmob/mobile"
)

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
	defer grmobtest.NoConcerns(t)

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

	root := grmobtest.Parse(t, mobile.RenderInitial())
	for _, c := range claims {
		for _, want := range c.Shows {
			if !grmobtest.Shows(root, want) {
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
	strip := grmobtest.Find(grmobtest.Parse(t, mobile.RenderInitial()), func(n *core.Node) bool { return n.Type == "TabView" })
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
	row := innermostTappableShowing(grmobtest.Parse(t, mobile.RenderInitial()), title)
	if row == nil {
		t.Fatalf("no tappable row showing %q", title)
	}
	mobile.TriggerCallback(row.Props["onClick"].(string))
}

func innermostTappableShowing(n *core.Node, title string) *core.Node {
	if n == nil {
		return nil
	}
	for _, c := range n.Children {
		if found := innermostTappableShowing(c, title); found != nil {
			return found
		}
	}
	if _, clickable := n.Props["onClick"].(string); clickable && grmobtest.Shows(n, title) {
		return n
	}
	return nil
}
