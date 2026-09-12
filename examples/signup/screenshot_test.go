package signup

import (
	"testing"

	"github.com/rohanthewiz/grmob/internal/shotclaims"
)

// showsText reports whether anything in the tree puts this text on the
// screen.
//
// Every prop rather than Text.content alone, because "on the screen" is
// not one node type: a FormField's label, a Button's label, an Input's
// placeholder and an error line are all words a reader sees, and this
// screen's whole subject is the error line. Style is not searched — it is
// a separate field on the wire, so a colour can never be read as a word.
func showsText(n *node, s string) bool {
	return findNode(n, func(n *node) bool {
		return shotclaims.Shown(n.Props, s)
	}) != nil
}

// TestTheSignupScreenshotStillShowsWhatItClaims drives this form to the
// state docs/images/signup.png was taken in and asserts the picture's
// strings are still on the screen.
//
// See internal/shotclaims for why a screenshot needs holding at all: it is
// a fact about a screen, written down on the day it was true, and the
// README quotes one of its sentences as prose. This is the half of that
// pair which renders the app.
//
// The state is reached by typing, not by posing. The mismatch in the
// picture is one transposed character — hunter2222 against hunter2221 —
// because that is what makes the cross-field message appear, and a value
// written straight into the form's state would be asserting that the
// message exists rather than that the form produces it.
func TestTheSignupScreenshotStillShowsWhatItClaims(t *testing.T) {
	claims := shotclaims.For("examples/signup")
	if len(claims) == 0 {
		t.Fatal("no claim names examples/signup, and docs/images/signup.png " +
			"is a picture of it. Either the shot has gone or the manifest has " +
			"lost its row; both leave this test asserting nothing.")
	}

	for _, c := range claims {
		mgr := newApp(t)

		typeInto(t, mgr, "you@example.com", "ada@lovelace.dev")
		typeInto(t, mgr, "••••••••", "hunter2222")
		confirm := secondPasswordField(t, mgr)
		mgr.DispatchTextCallback(confirm.Props["onChange"].(string), "hunter2221")
		tickTerms(t, mgr, true)
		tap(t, mgr, "Create account")

		root := tree(t, mgr)
		for _, want := range c.Shows {
			if !showsText(root, want) {
				t.Errorf("docs/images/%s shows %q and this app no longer "+
					"does, %s.\n\n"+
					"The picture is now a statement about a version of this "+
					"screen that does not exist. Re-take it (see wasm/shots) "+
					"and move the claim and the README's caption with it.",
					c.File, want, c.State)
			}
		}
	}
	assertNoConcerns(t)
}
