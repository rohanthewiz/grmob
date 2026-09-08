package verify

import (
	"strings"
	"testing"
)

// Why the band's cross-target census has three rows and not four.
//
// # What the census is
//
// components.GroupHeader moved its padding from the band Row onto the growing
// control inside it, so that a press lands on the whole band. The warrant is
// that the move is free — padding on a stretched child fills exactly the space
// the same padding on its parent held — and internal/bandfixture carries the
// geometry so a renderer can be asked whether that is true there:
//
//	the web              measured. wasm/verify/browser.mjs lays both
//	                     arrangements out in a real Chrome at every offer.
//	                     They agree, overflow included.
//	the SwiftUI solver   measured. ios/verify/band.swift runs GrMobFlexSolver,
//	                     which is THIS REPOSITORY'S CSS flex arithmetic, split
//	                     out of the SwiftUI Layout precisely so it can be
//	                     executed without a simulator. Under overflow it
//	                     disagrees: it shrinks each child in proportion to a
//	                     base that includes the child's own padding, where CSS
//	                     uses the inner flex base size.
//	Compose              not measured, and this test is why.
//
// # Why Compose is not the fourth row
//
// The other two are executable because the arithmetic is ours. GrMobFlexSolver
// is a file in this repository; the browser is a browser. Compose's Row is
// neither: a FlexGrow child is handed Modifier.weight and androidx's own
// RowColumnMeasurePolicy does the distributing, so the question "do the two
// arrangements agree on Compose" is a question about androidx's code.
//
// android/verify runs Kotlin on a plain JVM, which is what made it look like the
// place to ask — and the thing it can run is Kotlin that imports nothing
// (GrMobSelectMenu.kt and GrMobProgress.kt were written that way for exactly
// this reason). Compose's measure policy is not that: it measures Measurables
// into Placeables through compose-ui, which needs the Android runtime. Nor can
// the policy be read and pinned the way the gobind mapping is: this machine's
// gradle cache holds sources for foundation-layout 1.10.0, and
// android/app/build.gradle pins the Compose BOM at 2024.06.00, which resolves
// foundation-layout to 1.6.8 — whose sources are not cached, only its .aar. A
// pin against the wrong version is the mistake gobindVersion exists to prevent.
//
// So the honest state is: three targets asked, one derived. What the derivation
// says — and it is a derivation, not a measurement, which is why it lives in a
// comment rather than in an assertion — is that Compose agrees with the web and
// for a third reason again. Its Row has no proportional shrink at all: an
// unweighted child is measured with what is left, a weighted one gets
// (available - fixed) / total weight, and the band's badge is unweighted. So the
// entire deficit lands on the growing control in both arrangements, the badge
// keeps its width where both other renderers shrink it, and the two
// arrangements come out equal because the control's padding is inside its
// weighted extent either way.
//
// # What this test holds
//
// The premise. The census stops at three rows because Android delegates its
// distribution, and a renderer that stopped delegating — that grew arithmetic of
// its own, the way the SwiftUI side has — would put the answer back within
// reach AND make it something this repository is responsible for. Either way
// somebody has to look, and this is where they are told to.
func TestTheComposeRowDelegatesItsDistributionToCompose(t *testing.T) {
	body := codeOf(t, kotlinRenderer, "private fun RowScope.RowChildren(node: GrMobNode)")

	if !strings.Contains(body, "Modifier.weight(grow)") {
		t.Errorf("%s: RowChildren no longer hands a FlexGrow child Modifier.weight.\n\n"+
			"The band's cross-target census (see this file's header, and "+
			"internal/bandfixture) has three rows rather than four precisely because "+
			"this target's main-axis distribution is androidx's code and not this "+
			"repository's. If that has changed, the fourth answer is now reachable — "+
			"and it is now one of ours to be wrong about.", kotlinRenderer)
	}

	// The other half of "delegates": the child's own padding must not be lifted
	// out of the weighted extent. RenderNode applies the child's style through
	// boxModifier on top of the `extra` the parent hands down, which is what
	// puts the control's insets INSIDE its weighted share — the arrangement the
	// band's whole claim is about. A parent that padded the child itself would
	// be doing the distribution after all.
	if !strings.Contains(body, "RenderNode(child, m)") {
		t.Errorf("%s: RowChildren no longer passes the weight down as the child's "+
			"`extra` modifier for RenderNode to build the child's own box on top of. "+
			"The band's two inset arrangements are only the same question on this "+
			"target while the control's padding sits inside its weighted share.",
			kotlinRenderer)
	}
}
