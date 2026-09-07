package verify

import (
	"strings"
	"testing"
)

// What each target does with a child bigger than a fixed-size container, and
// the two call sites the census's native half rests on.
//
// # The question
//
// core.Spacer became "a Box with a fixed size" so that all four targets would
// lay a Spacer's children out the same way, and the note closing that work
// recorded a difference nobody had measured: Compose constrains a child to the
// declared size where the DOM was believed to let it spill. It is not a Spacer
// property — it is what every fixed-size container on that target does — which
// makes it the more interesting question, and nothing anywhere had asked whether
// the four targets agree about overflow for ANY fixed-size box.
//
// # The four answers
//
//	                     main axis            cross axis
//	WASM runtime         squeezed             spills          measured
//	htmlout              squeezed             spills          inherited
//	SwiftUI              squeezed             spills          measured / derived
//	Compose              squeezed             squeezed        derived
//
// The DOM row was MEASURED, and it is not what the note assumed: a browser does
// not simply let the child spill. wasm/verify/browser.mjs check 10 mounts a
// fixed-size core.Box and a fixed-size core.Row, each holding a child too big
// for it on both axes, and reads the rects. The child is squeezed along the
// container's main axis — it is a flex item, its shrink factor defaults to 1,
// and an empty box's automatic minimum is 0 — and it keeps its own size across
// the cross axis, because nothing shrinks a flex item across the line. htmlout
// emits the same three declarations for the same tree
// (TestAFixedSizeBoxExportsTheDeclarationsTheBrowserMeasured), so it inherits
// that answer rather than being measured again.
//
// The SwiftUI row is half measured and half derived, and the split is not
// arbitrary: the cross axis is SwiftUI's own behaviour, while the main-axis
// squeeze is GrMobFlexSolver's — this repository's arithmetic, which ios/verify
// executes. checkFixedSizeContainer in ios/verify/flex.swift runs the census's
// box through it, on browser.mjs's numbers, and gets the browser's answer;
// wasm/verify's TestTheFixedSizeCensusUsesOneSetOfNumbers holds the two
// harnesses to one fixture.
//
// What is left is DERIVED, from the platform call each renderer makes, and this
// file is what holds the renderers to those calls. That is the whole of what
// this package can do — its subject is "a rule the native shells must obey that
// no native toolchain can see" — and it is worth being exact about which half is
// which:
//
//	SwiftUI   .frame(width:) / .frame(height:) PROPOSES a size to its content
//	          and reports the fixed size to its parent. Content that insists on
//	          being larger keeps its size and is drawn overflowing, and nothing
//	          clips it: Renderer.swift spends .clipped() on exactly two image
//	          content modes and nowhere else. The main-axis squeeze is not the
//	          frame's doing at all — it is GrMobFlexSolver, this repository's own
//	          CSS flex arithmetic, which is why that column matches the DOM's.
//
//	Compose   Modifier.width(n) / Modifier.height(n) set the child's MINIMUM and
//	          MAXIMUM alike (foundation-layout's Size.kt: SizeElement(minWidth =
//	          width, maxWidth = width, enforceIncoming = true), whose node
//	          measures its child with constraints.constrain(targetConstraints)).
//	          A maximum is what the other three targets do not impose, and it is
//	          the whole of the divergence: the child is squeezed on both axes
//	          rather than on one.
//
// # Why the call site is pinned even though the source is now readable
//
// The Compose reading used to be of whichever foundation-layout a gradle cache
// happened to hold, which was not the version this module builds against.
// composelayout_test.go closes that: the version is derived from the BOM's own
// pom and the claims are read out of the sources jar itself, so the paragraph
// above is now held to androidx's code rather than to somebody's memory of it.
//
// This file is still the other half, and the halves answer different questions.
// composelayout_test.go asks "is Modifier.width still what the census says it
// is"; this asks "is Modifier.width still what the renderer calls". A renderer
// that moved off it — to requiredSize, which ignores incoming constraints, or
// to sizeIn, which sets a maximum and no minimum — would change this target's
// answer with a perfectly accurate reading of androidx sitting beside it.
const fixedSizeWhy = "the four-target answer for a child bigger than a " +
	"fixed-size container is derived from this call (see the header), and a " +
	"renderer that no longer makes it has an answer nothing here describes"

// Compose's fixed dimensions must be Modifier.width/height.
//
// Named rather than any-sizing-modifier, because the alternatives are the point:
// Modifier.requiredSize ignores the incoming constraints entirely (its own doc
// contrasts the two), and Modifier.sizeIn sets a range. Either would leave the
// mapping compiling and the census wrong.
func TestTheComposeFixedDimensionSetsAMaximum(t *testing.T) {
	body := declSource(t, kotlinStyle,
		"private fun dimensionModifier(value: String, horizontal: Boolean): Modifier")
	for _, want := range []string{"Modifier.width(", "Modifier.height("} {
		if !strings.Contains(body, want) {
			t.Errorf("%s: dimensionModifier does not call %s — %s",
				kotlinStyle, want, fixedSizeWhy)
		}
	}
	// The two that would silently change the answer.
	for _, unwanted := range []string{"requiredSize", "requiredWidth", "requiredHeight",
		"widthIn", "heightIn", "sizeIn"} {
		if strings.Contains(body, unwanted) {
			t.Errorf("%s: dimensionModifier calls %s. That is a different constraint "+
				"contract from Modifier.width/height — required* ignores the incoming "+
				"constraints and *In sets a range — so this target's row in the "+
				"overflow census no longer follows from the code.", kotlinStyle, unwanted)
		}
	}
}

// SwiftUI's fixed dimensions must be .frame(width:) / .frame(height:), and
// nothing on that path may clip.
//
// The clip half is the one a "calls frame" check cannot see. A .clipped() added
// to the dimension path would compile, would look tidier, and would make this
// target hide an overflow the DOM shows — which is a divergence in the opposite
// direction from Compose's and one no test would have reported.
func TestTheSwiftUIFixedDimensionProposesAndDoesNotClip(t *testing.T) {
	body := declSource(t, swiftStyle,
		"@ViewBuilder fileprivate func grMobDimension(")
	for _, want := range []string{"frame(width:", "frame(height:"} {
		if !strings.Contains(body, want) {
			t.Errorf("%s: grMobDimension does not call .%s) — %s",
				swiftStyle, want, fixedSizeWhy)
		}
	}
	if strings.Contains(body, "clipped()") {
		t.Errorf("%s: grMobDimension clips. A SwiftUI frame proposes a size and does "+
			"not enforce one, so the overflow census's SwiftUI row says a child bigger "+
			"than the box is drawn spilling out of it; clipping hides that, on one "+
			"target only, with nothing else in the repository saying so.", swiftStyle)
	}
}
