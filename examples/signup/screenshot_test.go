package signup

import (
	"testing"

	"github.com/rohanthewiz/grmob/grmobtest"
	"github.com/rohanthewiz/grmob/internal/shotclaims"
)

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
		app := newApp(t)

		app.Type("you@example.com", "ada@lovelace.dev")
		app.Type("••••••••", "hunter2222")
		confirm := secondPasswordField(t, app)
		app.Manager().DispatchTextCallback(confirm.Props["onChange"].(string), "hunter2221")
		tickTerms(t, app, true)
		app.Tap("Create account")

		root := app.Tree()
		for _, want := range c.Shows {
			if !grmobtest.Shows(root, want) {
				t.Errorf("docs/images/%s shows %q and this app no longer "+
					"does, %s.\n\n"+
					"The picture is now a statement about a version of this "+
					"screen that does not exist. Re-take it (see wasm/shots) "+
					"and move the claim and the README's caption with it.",
					c.File, want, c.State)
			}
		}
		app.NoConcerns()
	}
}
