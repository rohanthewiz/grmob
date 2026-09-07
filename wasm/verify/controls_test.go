package main

import (
	"regexp"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/internal/bandfixture"
)

// browser.mjs's controls, pinned from outside the file they live in.
//
// # What a control is, and the hole in it
//
// Three checks in the browser pass open by proving they have a subject. The
// ArrowDown check scrolls the page with nothing focused, because a document
// that cannot scroll would pass whether or not the key was consumed. The sticky
// check measures the scroller's overflow, because a band with nothing to stay
// put against stays put trivially. The band check asks whether the label ended
// up with less room than it wanted, because two arrangements that never reached
// the arithmetic agree by not having done any.
//
// Each of those is a runtime assertion about a fixture in the same file. So the
// pair is deletable: drop the control AND the declaration that gives it its
// subject, and the pass goes on printing OK about a check that has stopped
// asking anything. That is not a hypothetical — the band check's subject is one
// `MinWidth: "0"` in bandTree, and removing it makes both arrangements sit at
// their natural width and agree.
//
// No control can close that, because a control is code in the file whose
// completeness is in question. The only thing that can is something outside it,
// which is this.
//
// # What is derived and what is merely declared
//
// Two of the three subjects are numbers the file states, so they are ARITHMETIC
// here rather than substrings: the page's filler against the window Chrome is
// launched with, and the sticky fixture's content against its port. Those pins
// cannot be satisfied by prose and they fail on a fixture edited to fit rather
// than only on one deleted.
//
// The rest is substrings, and the reason is worth naming rather than hiding: a
// cleared flex minimum and a control's own failure text are not numbers this
// package can recompute, so what is held is that they are still there. That is
// the weaker half, and it is exactly the half a Go test can offer for
// JavaScript it cannot run.
//
// # The limit
//
// This table is written out, not derived. A fourth control added to browser.mjs
// is not pinned until somebody adds a row here, and nothing detects that —
// "which assertions in this file are controls" is a question about intent, and
// the alternative (a naming convention the checks must obey) would be this test
// dictating how the pass is written to buy a totality it still could not prove.

// The numbers browser.mjs states about the two fixtures whose subject is a
// size. Each is a single-line literal in a file this package does not otherwise
// parse, which is what makes a regexp the right tool — the same reading
// fixedsize_test.go takes of the same file.
var (
	windowSize   = regexp.MustCompile(`--window-size=(\d+),(\d+)`)
	pageFiller   = regexp.MustCompile(`<div style="height: (\d+)px">`)
	stickyPort   = regexp.MustCompile(`Width: "300px", Height: "(\d+)px", Overflow: "auto"`)
	stickyRows   = regexp.MustCompile(`const STICKY_ROWS = (\d+);`)
	stickyBandH  = regexp.MustCompile(`Height: "(\d+)px", Background: BAND_FILL`)
	stickyRowH   = regexp.MustCompile(`Height: "(\d+)px", Background: ROW_FILL`)
	clearedMinRe = regexp.MustCompile(`MinWidth: "0"`)
)

// The ArrowDown control's subject: a document taller than the window.
//
// The filler and the window size are both declared in browser.mjs, so this is
// the whole of the claim rather than a proxy for it. A filler shortened to fit
// the viewport would leave check 3 measuring a scroll that could not happen —
// and with the control gone too, measuring it against a scrollY that is 0 for
// the wrong reason.
func TestTheScrollControlStillHasAPageToScroll(t *testing.T) {
	src := joinedSource(t)

	filler := controlNumber(t, src, pageFiller, "the page's tall filler")
	win := windowSize.FindStringSubmatch(src)
	if win == nil {
		t.Fatalf("%s no longer launches Chrome with an explicit --window-size (%s). The "+
			"scroll control's subject is the filler against the viewport, and a headless "+
			"default is not something this repository can hold anything to.",
			browserChecks, windowSize)
	}
	height := atoi(t, win[2], "the window height")
	if filler <= height {
		t.Errorf("%s: the page's filler is %dpx in a %dpx window, so the document does "+
			"not scroll. Check 3 asks whether a listbox's ArrowDown handler prevented a "+
			"scroll; with nothing to scroll it passes on a scrollY of 0 that means "+
			"nothing, and the control that would have said so is in the same file as "+
			"the filler.", browserChecks, filler, height)
	}
	mustContain(t, src, "did not scroll the page",
		"the ArrowDown control's own failure text. It is what says the browser scrolls "+
			"at all before check 3 asks whether it was stopped")
}

// The sticky control's subject: content taller than its port.
//
// Both numbers and the row count are in the fixture, so the overflow the check
// needs is arithmetic rather than an assertion. Two controls in the pass rest on
// it — one that there is overflow, one that the overflow is the List standing at
// its own height rather than its rows spilling out of a compressed List — and
// both are named here, because the second is the one that distinguishes the
// arrangement the fixture describes from a different arrangement that is also
// green.
func TestTheStickyControlStillHasSomethingToScroll(t *testing.T) {
	src := joinedSource(t)

	port := controlNumber(t, src, stickyPort, "the sticky Scroll's port height")
	rows := controlNumber(t, src, stickyRows, "the sticky fixture's row count")
	band := controlNumber(t, src, stickyBandH, "the sticky band's height")
	row := controlNumber(t, src, stickyRowH, "the sticky fixture's row height")

	content := band + rows*row
	if content <= port {
		t.Errorf("%s: the sticky fixture holds a %dpx band and %d %s of %dpx — %dpx — "+
			"in a %dpx port, so there is nothing for the band to stay put against. "+
			"Check 6 would then pass on a band that never had to be pinned, and the two "+
			"controls that say so are in the same file as the numbers.",
			browserChecks, band, rows, plural(rows, "row"), row, content, port)
	}
	for _, want := range []struct{ text, why string }{
		{"there is nothing for the band to stay put against",
			"the control that says the scroller has overflow"},
		{"so it was compressed to fit",
			"the control that says the overflow is the one the fixture describes — a " +
				"List at its own height inside a shorter Scroll, rather than a compressed " +
				"List with its rows spilling out of it, which is also green and is a " +
				"different arrangement"},
	} {
		mustContain(t, src, want.text, want.why)
	}
}

// The band control's subject: a cleared flex minimum, and an offer that
// overflows.
//
// The offer half is derived, and from the fixture rather than from the browser
// file: internal/bandfixture states the offers, and if none of them is narrower
// than the band's natural width then no case reaches the shrink arithmetic at
// all. The cleared minimum is the half that cannot be derived — CSS gives a
// flex item a content-based minimum, the fixture's label is a box with a
// declared width, and without `MinWidth: "0"` on both items nothing shrinks and
// the two arrangements agree by never having divided anything.
func TestTheBandControlStillHasADeficitToDivide(t *testing.T) {
	src := joinedSource(t)

	// Both flex items: the growing control and the badge beside it. One of the
	// two would leave the other pinned at its own content width, which is the
	// same vacuous agreement one level down.
	if got := len(clearedMinRe.FindAllString(src, -1)); got != 2 {
		t.Errorf("%s: bandTree clears the automatic flex minimum on %d children, want 2 "+
			"(the growing control and the badge). CSS will not shrink a flex item below "+
			"its own min-content width, and the fixture's label is a box with a declared "+
			"width — so an item that kept its minimum sits at its natural size in both "+
			"arrangements, and check 9's two answers agree without either having reached "+
			"the arithmetic.", browserChecks, got)
	}
	mustContain(t, src, "never reaching the arithmetic this check is about",
		"the band control's own failure text. It is what says the label really was "+
			"squeezed before the two arrangements are compared")

	// And the fixture still offers something narrower than the band.
	overflowing := 0
	for _, c := range bandfixture.Cases() {
		natural := c.Now.Row.Left + c.Now.Row.Right + c.Now.Control.Left +
			c.Now.Control.Right + c.Label.W
		if c.Badge.W > 0 {
			natural += c.Now.Gap + c.Badge.W
		}
		for _, offer := range c.Offers {
			if offer >= 0 && offer < natural {
				overflowing++
			}
		}
	}
	if overflowing == 0 {
		t.Errorf("internal/bandfixture offers no band less than its natural width, so " +
			"check 9's overflow arm is never entered and the divergence it records is " +
			"unmeasured. The control in browser.mjs guards the arithmetic once it is " +
			"reached; this is whether it is reached at all.")
	}
}

// jsConcat is a string literal continued on the next line: `…text ` + `more…`,
// which is how every message in browser.mjs longer than a line is written.
var jsConcat = regexp.MustCompile("[\"`]\\s*\\+\\s*\n\\s*[\"`]")

// joinedSource is browser.mjs with those joins closed up, so a sentence this
// file pins is the sentence a reader of the pass sees rather than whichever
// fragment happens to fit on one line.
//
// Without it the pins would be short fragments chosen to avoid the wrap points,
// which move whenever a message is reworded — so the pin would break on edits
// that changed nothing and hold nothing against the edit that mattered.
func joinedSource(t *testing.T) string {
	t.Helper()
	return jsConcat.ReplaceAllString(readFixtureFile(t, browserChecks), "")
}

// plural is for the one count in this file that can legitimately be 1: a
// fixture edited down to a single row is exactly the failure below, and
// "1 rows" is the sentence a reader would have to look past to see it.
func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}

// controlNumber pulls one number out of browser.mjs, failing rather than
// returning a zero: a pin that silently found nothing would compare two zeroes
// and agree.
func controlNumber(t *testing.T, src string, re *regexp.Regexp, what string) int {
	t.Helper()
	m := re.FindStringSubmatch(src)
	if m == nil {
		t.Fatalf("%s no longer spells %s where this looks for it (%s). If it moved, "+
			"re-point this; if it was removed, a control in that file has lost the "+
			"subject it exists to prove.", browserChecks, what, re)
	}
	return atoi(t, m[1], what)
}

func atoi(t *testing.T, s, what string) int {
	t.Helper()
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			t.Fatalf("%s is %q, which is not a number", what, s)
		}
		n = n*10 + int(c-'0')
	}
	return n
}

// mustContain holds one of the controls' own sentences to the file.
//
// The failure text rather than the assertion around it: an `if` can be rewritten
// a dozen ways and the sentence it prints is the thing a reader of the pass
// actually sees, so it is the half worth naming.
func mustContain(t *testing.T, src, want, why string) {
	t.Helper()
	if !strings.Contains(src, want) {
		t.Errorf("%s no longer contains %q — %s.\n\nA control and the fixture it is "+
			"about live in the same file, so removing both is invisible to the pass "+
			"itself. This is the only thing that can say so.", browserChecks, want, why)
	}
}
