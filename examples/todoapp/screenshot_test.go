package todoapp

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/internal/shotclaims"
	"github.com/rohanthewiz/grmob/mobile"
)

// showsText reports whether anything in the current tree puts this text on
// the screen. See shotclaims.Shown for why the search reaches into nested
// prop values rather than stopping at top-level strings.
func showsText(n *node, s string) bool {
	return findNode(n, func(n *node) bool {
		return shotclaims.Shown(n.Props, s)
	}) != nil
}

// TestTheTodoScreenshotStillShowsWhatItClaims drives this app to the state
// docs/images/todo.png was taken in and asserts the picture's strings are
// still on the screen.
//
// See internal/shotclaims for why a screenshot needs holding at all. What
// this arm adds to the manifest's own checks is the only thing a manifest
// cannot do for itself: it renders the app.
//
// # Why it empties the list first
//
// This package's tests share one store. The data directory is created in
// TestMain and the bytdb file under it outlives every test in the package,
// which is the whole point of TestTodoPersistence — so by the time this
// runs there are rows in it from the tests above, and "2 items left" would
// be a reading of their leftovers rather than of the four titles below.
//
// Emptied through the app's own delete button rather than through the
// store, for the reason the shot itself was taken that way: the picture is
// of a screen a person arrived at, and a state written in behind the app's
// back is a state the app was never asked to produce.
func TestTheTodoScreenshotStillShowsWhatItClaims(t *testing.T) {
	core.ClearConcerns()
	claims := shotclaims.For("examples/todoapp")
	if len(claims) == 0 {
		t.Fatal("no claim names examples/todoapp, and docs/images/todo.png " +
			"is a picture of it. Either the shot has gone or the manifest has " +
			"lost its row; both leave this test asserting nothing.")
	}

	emptyTheList(t)

	// The four titles in the picture, in the order they were typed. The
	// second and the fourth are the ticked ones, which is what makes the
	// footer say two and the bulk-clear button exist at all.
	titles := []string{
		"Buy oat milk", "Read the GrMob docs", "Ship the beta", "Call Ada",
	}
	for _, title := range titles {
		addTodo(t, title)
	}
	for _, title := range []string{titles[1], titles[3]} {
		tickRow(t, title)
	}

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

	// Left as it was found, so that a test added after this one does not
	// inherit four todos it never created.
	emptyTheList(t)
	assertNoConcerns(t)
}

// emptyTheList taps the first row's delete button until no row has one.
//
// Bounded rather than looping until the predicate says stop: a delete that
// stopped deleting would otherwise hang the package's whole test run
// instead of failing it, and a hang is strictly worse than a failure —
// it is what a CI job does for its entire timeout. The bound is generous
// against a list these tests build a handful of rows in.
func emptyTheList(t *testing.T) {
	t.Helper()
	const bound = 100
	for i := 0; ; i++ {
		row := findNode(currentTree(t), buttonLabeled("✕"))
		if row == nil {
			return
		}
		if i == bound {
			t.Fatalf("still deleting after %d rows: the delete button is "+
				"there and the row it belongs to is not going away.", bound)
		}
		mobile.TriggerCallback(row.Props["onClick"].(string))
	}
}

// tickRow ticks the checkbox belonging to the row with this title.
//
// The row is found as the innermost node that both shows the title and
// holds a checkbox, which is what a row IS on this screen: the title and
// the tick are siblings under one container. Matching on the accessibility
// label would have been more direct and is not available here — that is a
// Style field rather than a prop, and the node shape these tests decode
// carries props only.
//
// Innermost matters. Every ancestor of a row also shows the title and also
// holds a checkbox, so the outermost match is the whole screen and ticking
// the checkbox found under it would tick the first row every time. The
// search is therefore post-order: children are asked before their parent,
// and the first node to answer is the deepest one that can.
func tickRow(t *testing.T, title string) {
	t.Helper()
	row := innermostRowShowing(currentTree(t), title)
	if row == nil {
		t.Fatalf("no row showing %q with a checkbox in it", title)
	}
	box := findNode(row, byType("Checkbox"))
	mobile.TriggerBoolCallback(box.Props["onToggle"].(string), true)
}

func innermostRowShowing(n *node, title string) *node {
	if n == nil {
		return nil
	}
	for _, c := range n.Children {
		if found := innermostRowShowing(c, title); found != nil {
			return found
		}
	}
	if showsText(n, title) && findNode(n, byType("Checkbox")) != nil {
		return n
	}
	return nil
}
