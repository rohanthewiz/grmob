// Package shotclaims states what each screenshot in docs/images claims to
// show, so that a picture of this framework is held to the framework the
// way every other figure in this repository is.
//
// # The problem this exists for
//
// A screenshot is unpinned prose in image form. docs/images holds seven
// facts about seven screens, written down on the day they were true, with
// nothing that reads them — the same shape as the six stale lesson counts
// the session before this one fixed, and the repository's whole argument
// is that such a number is worth nothing. A widget restyle, a theme
// change or a renamed label silently turns docs/images into a picture of
// a version that no longer exists, and the README goes on describing it.
//
// Re-photographing on every change is not the answer: it is expensive, it
// needs a browser, and nothing about it FAILS — a stale picture and a
// fresh one look equally plausible to a reader, which is precisely why
// the drift is silent.
//
// # What is held instead
//
// The text. Every shot is a rendering of an app in a state, and the state
// is reachable in an ordinary Go test through render.Manager — the same
// entry point the native shells use and the same one the shots were
// driven through. So each claim below records the app, the state, and the
// strings that are legible in the image; the app's own package asserts
// them against a real rendered tree, and the check costs no browser and
// no pixels.
//
//	docs/images/todo.png          a picture, and nothing reads it
//	     │
//	     ├── Package "examples/todoapp"   ─┐
//	     ├── State   "four typed, two …"   ├─ examples/todoapp's own test
//	     └── Shows   "2 items left", …    ─┘  drives this and asserts these
//	                    │
//	                    └── Quoted ──── the README's alt text says so too
//
// What that cannot catch is a change with no text in it: a colour, an
// inset, a font. Those are the kinds of drift the reader loses least by,
// and they are already the subject of the browser pass's own palette and
// band checks. What it does catch is the kind that makes a caption a lie
// — a renamed button, a reworded empty state, a lesson count that grew.
//
// # Why the claims live in one package rather than beside each app
//
// Two of the three questions are about the SET. Every file in
// docs/images has to be claimed by something, and every claim has to name
// a file that is there; neither is answerable from inside one example
// package, which can see one app and one shot. The rendering assertions
// are the third question and they necessarily live with the app, because
// that is where the app is.
//
// So this package is the manifest and TestEveryScreenshotIsClaimedAndEvery
// ClaimIsShown holds it against the tree; the per-app tests it names are
// what turn each row into a reading of a rendered screen.
//
// # Why Test is a string
//
// Because a name in this repository's prose is held to resolving: the
// shared repository parse reads every string literal in every tracked Go
// file for Test-shaped names and fails when one names a test that does
// not exist (see wasm/verify/prosenames_test.go). Naming the driver here
// therefore costs nothing and buys the one link a manifest cannot check
// for itself — that the row is asserted by something rather than merely
// written down.
package shotclaims

import "strings"

// Claim is one image in docs/images and what it is held to.
type Claim struct {
	// File is the image's name within docs/images.
	File string

	// Package is the repository-relative directory of the app the shot is
	// of, empty for a composite. Written as a path rather than an import
	// path because it is printed in failures, where what a reader wants is
	// somewhere to go.
	Package string

	// State says how the app was driven to what the picture shows. Prose,
	// deliberately: it is the instruction for re-taking the shot, and the
	// test that asserts Shows is the executable form of it. A reader
	// comparing the two is the point — a State nobody could follow to the
	// asserted strings is a row that has drifted from its own test.
	State string

	// Shows is text legible in the image. Held to appearing in the app's
	// rendered tree in the state above.
	//
	// Substrings, not whole labels, so that a wording tweak in a sentence
	// the picture does not turn on is not a test edit. What goes in is
	// what a reader would name if asked what is on the screen.
	Shows []string

	// Quoted is the part of Shows the README's alt text also states, and
	// it is the join between the prose and the render. Every entry must be
	// in Shows and must appear verbatim in that image's alt text, so a
	// caption saying "Count: 3" over a screen that now counts differently
	// fails on both sides at once rather than on neither.
	//
	// A subset and not the whole of Shows: alt text describes a picture
	// and is not a transcript of it. Empty is legitimate for a shot whose
	// caption quotes nothing from the screen.
	Quoted []string

	// Test is the test that drives the app to State and asserts Shows.
	Test string

	// MadeOf names the claims a composite is assembled from, and is the
	// only field a composite has besides File. A composite is a picture of
	// pictures: it shows nothing the images under it do not, so holding it
	// to text of its own would be asserting the same strings twice with
	// one copy free to drift.
	MadeOf []string
}

// Claims is every image in docs/images.
//
// Ordered as the README presents them rather than alphabetically, because
// that is the order a person re-taking them works in, and the harness in
// wasm/shots is driven one at a time from this list.
var Claims = []Claim{{
	File:   "hero.png",
	MadeOf: []string{"todo.png", "signup.png", "tabs-list.png"},
}, {
	File:    "counter.png",
	Package: "examples/counter",
	State:   "as it opens, then three taps of the + button",
	Shows:   []string{"Counter", "Count: 3", "−", "+"},
	Quoted:  []string{"Count: 3"},
	Test:    "TestTheCounterScreenshotStillShowsWhatItClaims",
}, {
	File:    "todo.png",
	Package: "examples/todoapp",
	State: "an empty list, then four titles typed into the input and " +
		"added — Buy oat milk, Read the GrMob docs, Ship the beta, " +
		"Call Ada — with the second and the fourth ticked done",
	Shows: []string{
		"Todos", "What needs doing?", "Add", "All", "Active", "Done",
		"Buy oat milk", "Read the GrMob docs", "Ship the beta", "Call Ada",
		"2 items left", "Clear completed",
	},
	Quoted: []string{"Clear completed"},
	Test:   "TestTheTodoScreenshotStillShowsWhatItClaims",
}, {
	File:    "signup.png",
	Package: "examples/signup",
	State: "ada@lovelace.dev and a password typed, the confirmation " +
		"mistyped by one character, the terms ticked, and Create account " +
		"tapped",
	Shows: []string{
		"Create your account",
		"Email", "ada@lovelace.dev", "We never share it",
		"Password", "At least 8 characters",
		"Confirm password", "The two passwords differ",
		"Plan", "Free", "You can change this later",
		"I accept the terms of service", "Create account",
	},
	Quoted: []string{"The two passwords differ"},
	Test:   "TestTheSignupScreenshotStillShowsWhatItClaims",
}, {
	File:    "tabs-list.png",
	Package: "examples/mobileapp",
	State:   "the Feed tab selected, then the third article row tapped",
	Shows: []string{
		"GrMob Demo", "Counter", "Form", "Feed", "Audio",
		"Selected: Article 3", "Article 1", "Article 3", "Article 8",
	},
	Quoted: []string{"Feed"},
	Test:   "TestTheTabsScreenshotStillShowsWhatItClaims",
}, {
	File:    "tutorial-contents.png",
	Package: "examples/tutorial",
	State:   "as it opens: chapter 1 expanded, nothing yet visited",
	Shows: []string{
		"GrMob Interactive Tutorial",
		"Learn GrMob inside GrMob",
		"0 of 55 lessons opened",
		"Chapter 1 — Views & Layout", "5 lessons",
		"Hello, GrMob", "Text & typography", "Rows, Columns & spacing",
		"Alignment & flex",
	},
	Quoted: []string{"0 of 55 lessons opened"},
	Test:   "TestTheTutorialContentsScreenshotStillShowsWhatItClaims",
}, {
	File:    "tutorial-lesson.png",
	Package: "examples/tutorial",
	State: "lesson 1.1 opened from the contents screen and scrolled to " +
		"the TRY IT panel, so the lesson's own title bar is above the " +
		"top of the frame",
	Shows: []string{
		"A GrMob screen is not a template or a markup file",
		"type View interface {",
		"func Profile(ctx *core.Context) core.View {",
		"Because views are values, composition is ordinary Go",
		"TRY IT",
		"Header()", "Stats()", "Bio text",
		"Gopher McGrMob",
	},
	Quoted: []string{"TRY IT"},
	Test:   "TestTheTutorialLessonScreenshotStillShowsWhatItClaims",
}}

// For is every claim about the app in this repository-relative directory.
//
// Plural because one app can be photographed more than once — the tutorial
// is, on its contents screen and on a lesson — and a caller that assumed
// one would have to change the day a second arrived.
func For(pkg string) []Claim {
	var out []Claim
	for _, c := range Claims {
		if c.Package == pkg {
			out = append(out, c)
		}
	}
	return out
}

// Composite reports whether this claim is a picture of other pictures.
func (c Claim) Composite() bool { return len(c.MadeOf) > 0 }

// Shown reports whether one decoded node prop puts this text on the
// screen, nested values included.
//
// # Why it recurses
//
// "On the screen" is not one node type and not one prop shape. A Button's
// label is a string; a Select's options are a list of objects with a label
// in each; a syntax-highlighted code block is a row of runs, each an
// object with the glyphs under "t". A check that only read top-level
// strings would silently stop finding the picker's "Free" and every line
// of every code block — which is a check that passes by looking away,
// the exact failure mode a screenshot already has.
//
// Style is deliberately not reachable from here: it crosses the wire as a
// field of its own beside Props, so a colour or a font name can never be
// read as a word on the screen. The callers walk nodes; this walks one
// node's props.
func Shown(prop any, s string) bool {
	switch v := prop.(type) {
	case string:
		return strings.Contains(v, s)
	case []any:
		for _, e := range v {
			if Shown(e, s) {
				return true
			}
		}
	case map[string]any:
		for _, e := range v {
			if Shown(e, s) {
				return true
			}
		}
	}
	return false
}
