package main

import (
	"os"
	"regexp"
	"strconv"
	"testing"
)

// The fixed-size census's two harnesses must mount the same box.
//
// # What the census is
//
// docs/platforms/native.md records what each of the four targets does with a
// child bigger than a fixed-size container, and the answer is per-axis: every
// target squeezes along the container's main axis, three of them let the child
// spill across the cross axis, and Compose squeezes both. Two harnesses
// establish the halves of that:
//
//	browser.mjs check 10    mounts a fixed-size core.Box and core.Row in a real
//	                        browser, each holding a child too big for it on both
//	                        axes, and reads the rects
//	ios/verify/flex.swift   runs GrMobFlexSolver — this repository's own CSS flex
//	                        arithmetic, which is where SwiftUI's main-axis
//	                        squeeze actually comes from — over the same box
//
// # Why the numbers are pinned rather than each harness choosing its own
//
// The census's claim is that the targets AGREE about the main axis. Two
// harnesses that agree while measuring different boxes are a weaker statement
// than two that agree on one box, and the difference is invisible in either
// pass: each would keep printing OK while the sentence they jointly support
// quietly stopped being about a single fixture.
//
// This is the same arrangement shrink_test.go makes for core.ShrinkNone and
// palette_test.go for the bundled hexes — one fact, several languages, pinned
// where the fact lives rather than transcribed into each of them. It lives in
// this package because browser.mjs does.
//
// The pin is deliberately not "the Swift file contains 120": the numbers are
// read out of both files by name and compared, so a fixture edited on either
// side names the pair that no longer matches.
func TestTheFixedSizeCensusUsesOneSetOfNumbers(t *testing.T) {
	// Both spellings are single-line constant declarations, which is what
	// makes a regexp the right tool rather than a parser: each is one number
	// on one line in a file whose language this test does not otherwise read.
	mjs := readFixtureFile(t, "browser.mjs")
	swift := readFixtureFile(t, "../../ios/verify/flex.swift")

	// Keyed by what the dimension is, because that is what a failure has to
	// say: "the container's width disagrees" is actionable and "VOID_W
	// disagrees" sends the reader to grep for a name that exists in one of the
	// two files.
	got := map[string][2]int{}
	for _, pair := range []struct {
		what string
		// The two spellings of one dimension. jsRe captures it out of
		// browser.mjs's paired `const A = 1, B = 2;` line, swiftRe out of
		// flex.swift's one-per-line `private let` block.
		jsRe, swiftRe *regexp.Regexp
	}{
		{"the container's width",
			regexp.MustCompile(`const VOID_W = (\d+)`),
			regexp.MustCompile(`private let voidW: CGFloat = (\d+)`)},
		{"the container's height",
			regexp.MustCompile(`VOID_H = (\d+)`),
			regexp.MustCompile(`private let voidH: CGFloat = (\d+)`)},
		{"the child's width",
			regexp.MustCompile(`const OVERSIZE_W = (\d+)`),
			regexp.MustCompile(`private let oversizeW: CGFloat = (\d+)`)},
		{"the child's height",
			regexp.MustCompile(`OVERSIZE_H = (\d+)`),
			regexp.MustCompile(`private let oversizeH: CGFloat = (\d+)`)},
	} {
		js, jsOK := fixtureNumber(t, "browser.mjs", mjs, pair.jsRe, pair.what)
		sw, swOK := fixtureNumber(t, "ios/verify/flex.swift", swift, pair.swiftRe, pair.what)
		if !jsOK || !swOK {
			continue
		}
		got[pair.what] = [2]int{js, sw}
		if js != sw {
			t.Errorf("%s is %d in browser.mjs and %d in ios/verify/flex.swift.\n\n"+
				"The two harnesses establish the DOM half and the SwiftUI half of one "+
				"census row, and the row says the two targets agree. Measuring "+
				"different boxes does not support that sentence, and neither pass "+
				"would notice: both would keep printing OK.", pair.what, js, sw)
		}
	}

	// And the child really is bigger than the container on both axes, or every
	// case in both harnesses is an ordinary layout: nothing overflows, nothing
	// is squeezed, and a pinned child is indistinguishable from an unpinned
	// one. The pair above can agree perfectly on numbers that measure nothing,
	// which is why this is asked separately.
	for _, axis := range []struct{ container, child string }{
		{"the container's width", "the child's width"},
		{"the container's height", "the child's height"},
	} {
		box, boxOK := got[axis.container]
		kid, kidOK := got[axis.child]
		if !boxOK || !kidOK {
			continue
		}
		if kid[0] <= box[0] {
			t.Errorf("%s (%d) is not greater than %s (%d), so the fixed-size fixtures "+
				"no longer overflow on this axis and every check over them passes by "+
				"never reaching the squeeze", axis.child, kid[0], axis.container, box[0])
		}
	}
}

// readFixtureFile is this file's own reader: the checks here span two
// harnesses' fixture files rather than the runtime, so a failure to read one
// is a fatal — a pin that silently checked nothing would be worse than no pin.
func readFixtureFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(raw)
}

// fixtureNumber pulls one captured number out of a file, reporting rather than
// returning on a miss so that a moved constant names itself instead of failing
// as a mismatch against a zero.
func fixtureNumber(t *testing.T, file, src string, re *regexp.Regexp, what string) (int, bool) {
	t.Helper()
	m := re.FindStringSubmatch(src)
	if m == nil {
		t.Errorf("%s no longer spells %s where this test looks for it (%s).\n\n"+
			"If the fixture moved, re-point this; if it was removed, one half of the "+
			"fixed-size census is no longer measured by anything.", file, what, re)
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		t.Errorf("%s spells %s as %q, which is not a number", file, what, m[1])
		return 0, false
	}
	return n, true
}
