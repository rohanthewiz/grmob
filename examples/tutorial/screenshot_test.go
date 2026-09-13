package tutorial

import (
	"testing"

	"github.com/rohanthewiz/grmob/internal/shotclaims"
)

// showsText reports whether anything in the tree puts this text on the
// screen.
//
// hasTextContaining is not enough here and the difference is the point:
// it reads a Text node's content and one row of a code block, which is
// what the lesson assertions elsewhere in this package need. A screenshot
// also shows widget text that is a PROP — a chapter card's "5 lessons"
// count, a badge, a picker's options — so this widens the search to every
// prop value, nested ones included. See shotclaims.Shown.
//
// Both, and not one: hasTextContaining is what reassembles a highlighted
// code line out of its runs, and the lesson shot is mostly code.
func showsText(n *node, s string) bool {
	if hasTextContaining(n, s) {
		return true
	}
	return findNode(n, func(n *node) bool {
		return shotclaims.Shown(n.Props, s)
	}) != nil
}

// assertShows holds one claim's strings against a rendered tree.
//
// Shared by the two tests below because the tutorial is photographed
// twice — the contents screen and a lesson — and the only thing that
// differs between them is how the app is driven there.
func assertShows(t *testing.T, root *node, c shotclaims.Claim) {
	t.Helper()
	for _, want := range c.Shows {
		if !showsText(root, want) {
			t.Errorf("docs/images/%s shows %q and this app no longer does, "+
				"with %s.\n\n"+
				"The picture is now a statement about a version of this "+
				"screen that does not exist. Re-take it (see wasm/shots) and "+
				"move the claim and the README's caption with it.",
				c.File, want, c.State)
		}
	}
}

// claimFor is the one claim about this file, or a stop.
//
// By file rather than by package, because shotclaims.For("examples/
// tutorial") returns two and each test here drives the app somewhere
// different. A missing row is a t.Fatalf and not a skip: a test that
// silently asserts nothing is the failure mode this whole mechanism
// exists to end, and it would be reintroduced by the mechanism itself.
func claimFor(t *testing.T, file string) shotclaims.Claim {
	t.Helper()
	for _, c := range shotclaims.For("examples/tutorial") {
		if c.File == file {
			return c
		}
	}
	t.Fatalf("no claim names docs/images/%s as a shot of examples/tutorial, "+
		"and this test exists to hold that picture. Either the shot has gone "+
		"or the manifest has lost its row.", file)
	return shotclaims.Claim{}
}

// TestTheTutorialContentsScreenshotStillShowsWhatItClaims holds
// docs/images/tutorial-contents.png.
//
// The app is not driven at all: the picture is of the screen as it opens,
// with chapter 1 already expanded and nothing yet visited. That is also
// what makes its progress caption — "0 of 54 lessons opened" — the most
// fragile claim in the manifest, and the reason it is there. The count is
// computed from the curriculum on every render and transcribed into the
// README's alt text, the site page and this picture; a lesson added
// anywhere moves the computed one and none of the copies.
func TestTheTutorialContentsScreenshotStillShowsWhatItClaims(t *testing.T) {
	mgr := newApp(t)
	assertShows(t, tree(t, mgr), claimFor(t, "tutorial-contents.png"))
	assertNoConcerns(t)
}

// TestTheTutorialLessonScreenshotStillShowsWhatItClaims holds
// docs/images/tutorial-lesson.png.
//
// The picture is scrolled, so the lesson's own title bar — "1.1  Hello,
// GrMob" and its chapter tag — is above the top of the frame and is
// deliberately not claimed. What a scroll changes is which part of a tree
// a camera sees, not what the tree holds, so a claim naming text the
// reader cannot actually see would be held by this test and by nothing a
// person could check against the image. The strings here are the ones in
// the frame: the opening paragraph, the highlighted code, and the TRY IT
// panel under it.
func TestTheTutorialLessonScreenshotStillShowsWhatItClaims(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Hello, GrMob")
	assertShows(t, tree(t, mgr), claimFor(t, "tutorial-lesson.png"))
	assertNoConcerns(t)
}
