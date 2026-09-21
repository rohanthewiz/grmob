package main

import (
	"os"
	"regexp"
	"strconv"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// core.OpacityClear, held to the three places outside Go that spell it.
//
// # The second sentinel, and why it gets the first one's test
//
// Opacity is the second core.Style number whose CSS initial value is not zero
// (see core.OpacityClear), so an alpha of zero crosses the wire as -1 and each
// runtime needs one rule to read it:
//
//	wasm/grmob-runtime.js     styleFromGrMob, which writes opacity
//	android/.../GrMobStyle.kt alphaOf, resolved at parse
//	ios/.../GrMobStyle.swift  alpha, read by grMobBox
//
// shrink_test.go says why a drifting sentinel is the worst kind: everything
// keeps compiling and each copy starts meaning something different. Here the
// failure is quieter still. CSS clamps a negative opacity to 0, so the web
// would go on looking right with the wrong number while Compose (which throws
// on an alpha outside [0, 1]) and SwiftUI did not.
//
// Each pattern spans the unset arm as well as the sentinel arm, for the reason
// kotlinShrinkSentinel gives: an unset Opacity must read as 1. A runtime that
// defaulted it to the zero it is stored as would draw every node in every
// tree transparent, and no comparison of sentinels would notice.
//
// What mobile/verify holds is the other half of each native mapping: that the
// reading reaches a modifier, at the right layer.

// runtimeOpacitySentinel matches the comparison in the runtime's style mapping.
var runtimeOpacitySentinel = regexp.MustCompile(`style\.Opacity === (-?\d+)\s*\? "0"`)

// kotlinOpacitySentinel matches both arms of GrMobStyle.kt's alphaOf.
var kotlinOpacitySentinel = regexp.MustCompile(
	`(?s)when \(opacity\) \{\s*0f -> 1f\s*(-?\d+)f -> 0f`)

// swiftOpacitySentinel matches both arms of GrMobStyle.swift's alpha.
var swiftOpacitySentinel = regexp.MustCompile(
	`(?s)if opacity == 0 \{ return 1 \}\s*if opacity == (-?\d+) \{ return 0 \}`)

func TestTheOpacitySentinelIsTheSameNumberEverywhere(t *testing.T) {
	want := strconv.Itoa(core.OpacityClear)

	for _, c := range []struct {
		file string
		re   *regexp.Regexp
		what string
	}{
		{"../grmob-runtime.js", runtimeOpacitySentinel,
			"this is the only place the live runtime knows that an alpha of zero is " +
				"not a zero. Reading the wrong number writes the sentinel through as an " +
				"alpha, or reads a real fade-out as \"no declaration\" and draws it opaque"},
		{"../../android/app/src/main/java/com/grmob/runtime/GrMobStyle.kt",
			kotlinOpacitySentinel,
			"alphaOf is the whole of what the Compose runtime knows about the sentinel, " +
				"and Modifier.alpha throws outside [0, 1] where CSS would clamp, so a " +
				"sentinel read as a plain number is a crash and not a wrong pixel"},
		{"../../ios/GrMob/Runtime/GrMobStyle.swift", swiftOpacitySentinel,
			"alpha is what grMobBox hands .opacity, and its unset arm is what keeps " +
				"every node that never mentioned Opacity drawn at all"},
	} {
		raw, err := os.ReadFile(c.file)
		if err != nil {
			t.Fatalf("reading %s: %v", c.file, err)
		}
		m := c.re.FindStringSubmatch(string(raw))
		if m == nil {
			t.Errorf("%s no longer spells the opacity sentinel where this test looks "+
				"for it (%s).\n\n%s\n\nIf the mapping moved, re-point this; if it was "+
				"removed, core.Opacity(0) no longer fades anything out on this target.",
				c.file, c.re, c.what)
			continue
		}
		if m[1] != want {
			t.Errorf("%s spells the opacity sentinel %s and core.OpacityClear is %s.\n\n%s",
				c.file, m[1], want, c.what)
		}
	}
}
