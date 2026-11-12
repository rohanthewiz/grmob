package tutorial

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/grmobtest"
)

// Chapter 2 demo-liveness tests. Same discipline as the chapter-1 pair in
// app_test.go: drive the demo through dispatched callbacks exactly as a
// native shell would, re-reading the tree before every dispatch (callback IDs
// are per-pass), and let TestMain's debug mode audit every pass — the state
// lessons are the ones where a rules-of-hooks slip would actually bite.

// typeInto finds the first Input in tree order and dispatches its onChange
// with text, as the platform text watcher would. The chapter-2 screens hold
// at most one Input, so first-in-tree-order is unambiguous.
func typeInto(t *testing.T, app *grmobtest.Driver, text string) {
	t.Helper()
	n := grmobtest.Find(app.Tree(), func(n *node) bool { return n.Type == "Input" })
	if n == nil {
		t.Fatal("no Input in the current tree")
	}
	app.Manager().DispatchTextCallback(n.Props["onChange"].(string), text)
}

// --- 2.1 The counter ------------------------------------------------------

func TestCounterDemoCounts(t *testing.T) {
	app := newApp(t)
	openLesson(t, app, "State: the counter")

	if !grmobtest.HasText(app.Tree(), "0") {
		t.Fatal("counter should start at 0")
	}
	for range 3 {
		app.Tap("+1")
	}
	if !grmobtest.HasText(app.Tree(), "3") {
		t.Fatal("three +1 taps should show 3")
	}
	app.Tap("−1")
	if !grmobtest.HasText(app.Tree(), "2") {
		t.Fatal("−1 should step the count back to 2")
	}
	app.Tap("Reset")
	if !grmobtest.HasText(app.Tree(), "0") {
		t.Fatal("Reset should return the count to 0")
	}
	grmobtest.NoConcerns(t)
}

// --- 2.2 Events & handlers ------------------------------------------------

// The demo card carries both gestures on one node; this drives each through
// its own registered callback and watches the log respond, then Clear empties
// it. Long-press rides the same void-callback channel as click, so the
// ordinary DispatchCallback works for both.
func TestEventsDemoLogsBothGestures(t *testing.T) {
	app := newApp(t)
	openLesson(t, app, "Events & handlers")

	gestureCard := func() *node {
		n := grmobtest.Find(app.Tree(), func(n *node) bool {
			_, hasLong := n.Props["onLongPress"].(string)
			return hasLong && grmobtest.HasText(n, "Tap or long-press me")
		})
		if n == nil {
			t.Fatal("no gesture card with an onLongPress handler in the tree")
		}
		return n
	}

	if !hasTextContaining(app.Tree(), "No events yet") {
		t.Fatal("the log should start empty")
	}
	app.Manager().DispatchCallback(gestureCard().Props["onClick"].(string))
	if !hasTextContaining(app.Tree(), "1 · tap") {
		t.Fatal("a click should log '1 · tap'")
	}
	app.Manager().DispatchCallback(gestureCard().Props["onLongPress"].(string))
	if !hasTextContaining(app.Tree(), "2 · long press") {
		t.Fatal("a long press should log '2 · long press'")
	}

	app.Tap("Clear log")
	if !hasTextContaining(app.Tree(), "No events yet") {
		t.Fatal("Clear log should empty the log again")
	}
	grmobtest.NoConcerns(t)
}

// The keyboard demo declares two chords on one button, a modifier chord and a
// bare F-key, so both page-global forms are on screen for the natives' hardware
// checks. The press is a callback like any tap, so it is dispatched here.
func TestEventsDemoDeclaresScreenWideShortcuts(t *testing.T) {
	app := newApp(t)
	openLesson(t, app, "Events & handlers")

	button := grmobtest.Find(app.Tree(), func(n *node) bool {
		return n.Type == "Button" && n.Props["label"] == "Log from the keyboard"
	})
	if button == nil {
		t.Fatal(`no "Log from the keyboard" button in the tree`)
	}
	if button.Style == nil || button.Style.AccessibilityKeyShortcuts != "Control+Alt+K F6" {
		t.Fatalf(`the button should declare "Control+Alt+K F6"; its style is %+v`, button.Style)
	}
	app.Tap("Log from the keyboard")
	if !hasTextContaining(app.Tree(), "1 · button") {
		t.Fatal("pressing the keyboard button should log '1 · button'")
	}

	// The gesture card declares a chord too: a shortcut is not a Button's
	// alone, and iOS reaches a tappable box by its own route.
	card := grmobtest.Find(app.Tree(), func(n *node) bool {
		_, hasLong := n.Props["onLongPress"].(string)
		return hasLong && grmobtest.HasText(n, "Tap or long-press me")
	})
	if card == nil || card.Style == nil || card.Style.AccessibilityKeyShortcuts != "Control+Alt+J" {
		t.Fatalf(`the gesture card should declare "Control+Alt+J"; got %+v`, card)
	}
	grmobtest.NoConcerns(t)
}

// --- 2.3 Controlled inputs ------------------------------------------------

func TestInputDemoEchoesTransformsAndClears(t *testing.T) {
	app := newApp(t)
	openLesson(t, app, "Controlled inputs")

	if !hasTextContaining(app.Tree(), "Nothing typed yet") {
		t.Fatal("the echo should start on its empty-state caption")
	}

	typeInto(t, app, "gopher")
	cur := app.Tree()
	if !grmobtest.HasText(cur, "Hello, gopher!") {
		t.Fatal("typing should echo through state into the greeting")
	}
	if !hasTextContaining(cur, "6 characters") {
		t.Fatal("the character count should read the same state")
	}

	// The transform runs on the way in: with UPPERCASE on, the next change
	// callback stores — and therefore displays — the uppercased value.
	toggleCheckbox(t, app, 0, true)
	typeInto(t, app, "gopher")
	if !grmobtest.HasText(app.Tree(), "Hello, GOPHER!") {
		t.Fatal("with the transform on, the stored value should be uppercased")
	}

	// Clear writes state from outside the field; the controlled field follows.
	app.Tap("Clear")
	if !hasTextContaining(app.Tree(), "Nothing typed yet") {
		t.Fatal("Clear should return the demo to its empty state")
	}
	grmobtest.NoConcerns(t)
}

// The TextArea demo: newlines are characters of the one string, the caption
// reads the same state, and Tidy lines rewrites the middle of the value from
// outside the field.
func TestTextAreaDemoCountsLinesAndTidiesThem(t *testing.T) {
	app := newApp(t)
	openLesson(t, app, "Controlled inputs")

	area := grmobtest.Find(app.Tree(), func(n *node) bool { return n.Type == "TextArea" })
	if area == nil {
		t.Fatal("lesson 2.3 should hold a TextArea")
	}
	if got := area.Props["value"]; got != "Milk\nEggs  " {
		t.Fatalf("the TextArea should start on its seeded two lines, got %q", got)
	}
	if !hasTextContaining(app.Tree(), "2 lines, 11 characters") {
		t.Fatal("the caption should count the seeded value's lines and characters")
	}

	app.Manager().DispatchTextCallback(area.Props["onChange"].(string), "Milk\n\nEggs  \nBread ")
	if !hasTextContaining(app.Tree(), "4 lines") {
		t.Fatal("typed newlines should count as lines")
	}

	app.Tap("Tidy lines")
	area = grmobtest.Find(app.Tree(), func(n *node) bool { return n.Type == "TextArea" })
	if got := area.Props["value"]; got != "Milk\nEggs\nBread" {
		t.Fatalf("Tidy lines should trim each line and drop the blank one, got %q", got)
	}
	grmobtest.NoConcerns(t)
}

// The second transform: the first letter of each word, and nothing else.
func TestCapitalizeWords(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"", ""},
		{"hello world", "Hello World"},
		{"hellox world", "Hellox World"},
		{"hello mcGrMob", "Hello McGrMob"},
		{"  two  spaces ", "  Two  Spaces "},
		{"été 1st", "Été 1st"},
	} {
		if got := capitalizeWords(c.in); got != c.want {
			t.Errorf("capitalizeWords(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// With "Capitalize each word" on, the stored value is capitalized.
func TestInputDemoCapitalizesWords(t *testing.T) {
	app := newApp(t)
	openLesson(t, app, "Controlled inputs")
	toggleCheckbox(t, app, 1, true)
	typeInto(t, app, "ada lovelace")
	if !grmobtest.HasText(app.Tree(), "Hello, Ada Lovelace!") {
		t.Fatal("with the word transform on, each word should be capitalized")
	}
	grmobtest.NoConcerns(t)
}

func TestLineSummaryAndTidyLines(t *testing.T) {
	for _, c := range []struct{ in, summary, tidy string }{
		{"", "Empty: zero lines.", ""},
		{"one", "1 line, 3 characters", "one"},
		{"a \n\n\tb\t", "3 lines", "a\n\tb"},
	} {
		if got := lineSummary(c.in); !strings.Contains(got, c.summary) {
			t.Errorf("lineSummary(%q) = %q, want it to contain %q", c.in, got, c.summary)
		}
		if got := tidyLines(c.in); got != c.tidy {
			t.Errorf("tidyLines(%q) = %q, want %q", c.in, got, c.tidy)
		}
	}
}

// --- 2.4 Conditional rendering --------------------------------------------

func TestConditionalDemoMatchesStatus(t *testing.T) {
	app := newApp(t)
	openLesson(t, app, "Conditional rendering")

	if !hasTextContaining(app.Tree(), "Fetching gophers") {
		t.Fatal("status should start on the Loading branch")
	}

	// The segments are chips, which render as Buttons — the tap helper
	// reaches them by label, as in the chapter-1 axis test.
	app.Tap("Error")
	cur := app.Tree()
	if !grmobtest.HasText(cur, "Something went wrong") {
		t.Fatal("selecting Error should render the Default branch")
	}
	if hasTextContaining(cur, "Fetching gophers") {
		t.Fatal("Match renders one branch — Loading should be gone")
	}

	app.Tap("Ready")
	if !grmobtest.HasText(app.Tree(), "All systems go") {
		t.Fatal("selecting Ready should render the ready card")
	}

	// The raw-status line is a separate core.If, reading the same state.
	toggleCheckbox(t, app, 0, true)
	if !hasTextContaining(app.Tree(), "status = 1 (Ready)") {
		t.Fatal("the If branch should reveal the raw status value")
	}
	grmobtest.NoConcerns(t)
}

// --- 2.5 Lists & keys -----------------------------------------------------

// hasKey reports whether any node under n carries exactly this key — how the
// test sees what the reconciler sees, since Keyed stamps Node.Key.
func hasKey(n *node, key string) bool {
	return grmobtest.Find(n, func(n *node) bool { return n.Key == key }) != nil
}

func TestListDemoAddsRemovesAndKeysRows(t *testing.T) {
	app := newApp(t)
	openLesson(t, app, "Lists & keys")

	cur := app.Tree()
	for _, title := range []string{"Feed the gopher", "Write some Go", "Ship the app"} {
		if !grmobtest.HasText(cur, title) {
			t.Fatalf("seed task %q missing from the list", title)
		}
	}
	for _, key := range []string{"task-1", "task-2", "task-3"} {
		if !hasKey(cur, key) {
			t.Fatalf("row key %q missing — For rows must be Keyed", key)
		}
	}

	app.Tap("＋ Add to top")
	if !hasKey(app.Tree(), "task-4") {
		t.Fatal("adding should insert a row keyed task-4")
	}

	// The first ✕ in tree order belongs to the top row — the one just added.
	app.Tap("✕")
	cur = app.Tree()
	if grmobtest.HasText(cur, "Task 4") {
		t.Fatal("removing the top row should drop Task 4")
	}
	if !grmobtest.HasText(cur, "Feed the gopher") || !hasKey(cur, "task-1") {
		t.Fatal("removal must not disturb the remaining keyed rows")
	}
	grmobtest.NoConcerns(t)
}

// --- 2.6 Two booleans -----------------------------------------------------

// The lesson's claim is about *when* a change takes effect, so the test drives
// each control and watches for the thing that distinguishes them: the switch's
// copy moves on the toggle alone, and the checkbox's does not move until Save.
//
// Both controls are in one tree, which is why this reaches for the node type
// rather than "the first bool control" — on the wire they are the same prop map
// (core.Switch), and position would be a silent way to drive the wrong one.
func TestBooleanDemoSeparatesTheInstantFromTheDeferred(t *testing.T) {
	app := newApp(t)
	openLesson(t, app, "Two booleans: checkbox and switch")

	// The switch starts on, and its line says so.
	if !hasTextContaining(app.Tree(), "Sending you a push") {
		t.Fatal("the switch should start on, with its subtitle reading as on")
	}

	// One toggle, no confirmation, and the setting has already changed.
	toggleBool(t, app, "Switch", 0, false)
	cur := app.Tree()
	if !hasTextContaining(cur, "Silent") {
		t.Fatal("the switch's own toggle should change the setting with nothing else involved")
	}
	if !hasTextContaining(cur, "off the moment it was tapped") {
		t.Fatal("the caption should follow the switch immediately")
	}

	// The checkbox, which changes nothing but itself.
	if !hasTextContaining(app.Tree(), "Not ticked") {
		t.Fatal("the checkbox should start unticked")
	}
	toggleCheckbox(t, app, 0, true)
	if !hasTextContaining(app.Tree(), "Ticked, not saved") {
		t.Fatal("a tick must not commit anything — that is the whole distinction")
	}

	// And the button, which is what commits it.
	app.Tap("Save")
	if !hasTextContaining(app.Tree(), "Saved: accepted") {
		t.Fatal("Save should commit the ticked value")
	}

	// Re-ticking revokes the save, so the two facts stay visibly separate.
	toggleCheckbox(t, app, 0, false)
	if !hasTextContaining(app.Tree(), "Not ticked") {
		t.Fatal("unticking after a save should drop back to the untouched reading")
	}
	grmobtest.NoConcerns(t)
}
