package main

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// core.ShrinkNone, held to the four places outside Go that spell it.
//
// # Why a number crosses a language boundary at all
//
// Every optional number in a core.Style means "unset" by being zero, and
// flex-shrink is the one property whose CSS initial value is not zero — so a
// factor of zero has to be spelled as something else, and core.Style carries no
// JSON field tags, which rules out the other obvious answer (omit the key and
// let its absence mean unset). The value travels as written into four places
// outside Go, each of which needs one rule to read it:
//
//	wasm/grmob-runtime.js     styleFromGrMob, which writes flexShrink
//	wasm/verify/browser.mjs   SHRINK_NONE, for the sticky fixture's List
//	ios/.../GrMobStyle.swift  shrinkFactor, read by the flex solver
//	android/.../GrMobStyle.kt shrinkFactor, read by the two children loops
//
// A sentinel that drifts is the worst kind: every one of those keeps compiling
// and each starts meaning something different, and the failure is a layout
// nobody looks at. So the number is pinned here rather than transcribed, which
// is the same arrangement sticky_test.go makes for core.StickyHeader()'s three
// declarations and palette_test.go for the bundled hexes.
//
// The two JavaScript ones are the files this package already owns; the two
// native ones are read here anyway rather than from mobile/verify, because the
// number itself lives in this file and splitting four copies of one fact
// across two packages is how a sentinel drifts. What mobile/verify holds is
// the other half of each native mapping — that the renderer still *reads*
// the factor — which is a claim about a call site rather than about a number.

// shrinkNoneConst matches the constant in browser.mjs.
var shrinkNoneConst = regexp.MustCompile(`(?m)^const SHRINK_NONE = (-?\d+);`)

// runtimeShrinkSentinel matches the comparison in the runtime's style mapping.
var runtimeShrinkSentinel = regexp.MustCompile(`style\.FlexShrink === (-?\d+)`)

// kotlinShrinkSentinel matches the sentinel arm of GrMobStyle.kt's shrinkFactor.
//
// Anchored on the whole `when`, first arm included, rather than on the arm
// alone: `-1f -> 0f` is two floats and an arrow, which is a shape a Kotlin file
// could grow elsewhere, and matching the unset arm beside it also pins the one
// thing that would otherwise be checked by nothing — that an unset factor
// still reads as 1 rather than as the zero it is stored as.
var kotlinShrinkSentinel = regexp.MustCompile(
	`(?s)when \(flexShrink\) \{\s*0f -> 1f\s*(-?\d+)f -> 0f`)

func TestTheShrinkSentinelIsTheSameNumberEverywhere(t *testing.T) {
	want := strconv.Itoa(core.ShrinkNone)

	for _, c := range []struct {
		file string
		re   *regexp.Regexp
		what string
	}{
		{"browser.mjs", shrinkNoneConst,
			"the sticky fixture's List is pinned with this number, and a fixture " +
				"spelling a factor the framework no longer recognises is a mounted " +
				"tree whose declaration is silently discarded"},
		{"../grmob-runtime.js", runtimeShrinkSentinel,
			"this is the only place the live runtime knows that a factor of zero is " +
				"not a zero, and reading the wrong number turns \"do not shrink\" into " +
				"no declaration at all — the opposite instruction"},
		{"../../android/app/src/main/java/com/grmob/runtime/GrMobStyle.kt",
			kotlinShrinkSentinel,
			"this is the whole of what the Compose runtime knows about the sentinel. " +
				"Compose has no proportional shrink to honour a fractional factor with, " +
				"so zero is the one shrink declaration it can act on at all — and it " +
				"acts on it through shrinkPinned, which is this reading",
		},
	} {
		raw, err := os.ReadFile(c.file)
		if err != nil {
			t.Fatalf("reading %s: %v", c.file, err)
		}
		m := c.re.FindStringSubmatch(string(raw))
		if m == nil {
			t.Errorf("%s no longer spells the shrink sentinel where this test looks "+
				"for it (%s).\n\n%s\n\nIf the mapping moved, re-point this; if it was "+
				"removed, core.FlexShrink(0) has gone back to doing nothing on this "+
				"target.", c.file, c.re, c.what)
			continue
		}
		if m[1] != want {
			t.Errorf("%s spells the shrink sentinel %s and core.ShrinkNone is %s.\n\n%s",
				c.file, m[1], want, c.what)
		}
	}
}

// And the sentinel is a number no author can produce by accident.
//
// The whole warrant for a magic value here is that CSS forbids a negative
// flex-shrink — the grammar is <number [0,∞]> — so a renderer can never be
// handed one legitimately and core.FlexShrink is the only door into the field.
// A sentinel that moved into the valid range would be indistinguishable from a
// factor somebody meant.
func TestTheShrinkSentinelIsOutsideTheValidRange(t *testing.T) {
	if core.ShrinkNone >= 0 {
		t.Errorf("core.ShrinkNone is %v, which is a flex-shrink factor an author "+
			"could legitimately ask for. The sentinel only works because CSS forbids "+
			"a negative factor, so a value in [0,∞) makes \"do not shrink\" and \"shrink "+
			"by this much\" the same number.", core.ShrinkNone)
	}
	// And the prop maps zero to it, which is what makes the range claim matter.
	var s core.Style
	core.FlexShrink(0).Apply(&s)
	if s.FlexShrink != core.ShrinkNone {
		t.Errorf("core.FlexShrink(0) stored %v, not core.ShrinkNone — the sentinel is "+
			"pinned above and nothing puts it in the field", s.FlexShrink)
	}
}

// The Swift runtime's reading, which is checked twice over.
//
// The sweep above already holds every spelling of the number to core.ShrinkNone,
// this one included. What it cannot see is the OTHER arm: an unset factor has
// to read as 1, because flex-shrink is the one flex property whose initial
// value is not zero, and a runtime that defaults it to 0 pins every item in
// every layout — a whole-app regression that no comparison of sentinels would
// notice. The Kotlin mapping gets the same guarantee from
// kotlinShrinkSentinel's anchor, which spans both arms; Swift's two arms are
// two statements, so they are asked for separately here.
func TestTheSwiftRuntimeReadsTheSameShrinkSentinel(t *testing.T) {
	raw, err := os.ReadFile("../../ios/GrMob/Runtime/GrMobStyle.swift")
	if err != nil {
		t.Fatalf("reading GrMobStyle.swift: %v", err)
	}
	src := string(raw)
	want := "if flexShrink == " + strconv.Itoa(core.ShrinkNone) + " { return 0 }"
	if !strings.Contains(src, want) {
		t.Errorf("GrMobStyle.swift's shrinkFactor does not contain %q.\n\n"+
			"That line is the whole of what the SwiftUI runtime knows about "+
			"core.ShrinkNone; without it a pinned flex item shrinks like every other "+
			"one, and GrMobFlexSolver's shrink arm is back to the single value the Go "+
			"side used to be able to express.", want)
	}
	// The default is the other half and it is not zero: an absent declaration
	// has to mean the CSS initial value, or every node in the tree stops
	// shrinking.
	if !strings.Contains(src, "if flexShrink == 0 { return 1 }") {
		t.Errorf("GrMobStyle.swift's shrinkFactor does not map an unset FlexShrink to " +
			"1. flex-shrink is the one flex property whose initial value is not zero, " +
			"so a runtime that defaults it to 0 pins every item in every layout.")
	}
}
