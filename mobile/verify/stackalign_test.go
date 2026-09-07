package verify

import (
	"regexp"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// core.StackAlign on the two natives.
//
// The property is the per-layer opt-out from core.ZStack's centre-on-both-axes
// contract, and it is the one placement prop in the framework that all four
// renderers honour — the flexbox AlignSelf beside it is read by the two DOM
// targets alone. That is the whole reason it exists as a type of its own, so a
// native arm quietly missing is the failure that would undo it.
//
// The failure is silent in the way this package's checks always are: a value
// with no arm falls to `default`/`else`, which returns nil, which means "leave
// this layer where the stack put it". So a layer asking for the top-left
// corner sits in the middle on a phone and in the corner in a browser, and
// nothing errors anywhere.
//
// See switchlabels_test.go for the parse and why it is one.
func TestNativeStackAlignmentsCoverEveryPlacement(t *testing.T) {
	required := asStrings(core.StackAlignments())

	for _, c := range []struct {
		syntax dispatchSyntax
		file   string
		fn     string
	}{
		{
			swiftSwitch.with(swiftStack, "public func grMobStackAnchor(", "switch align {"),
			"GrMobStack.swift", "grMobStackAnchor",
		},
		{
			kotlinWhen.with(kotlinRenderer, "private fun grMobStackAlignment(", "when (align) {"),
			"Renderer.kt", "grMobStackAlignment",
		},
	} {
		coverage{
			file:     c.file,
			fn:       c.fn,
			required: required,
			consequence: "returns the centre, so the layer stays where the stack's own " +
				"alignment put it and the placement is honoured on the web alone",
		}.check(t, c.syntax.labels(t))
	}
}

// core.Style.StackAlign must reach both native style parsers.
//
// The mappings below can be complete and the stack renderers can consult them
// and the property still never arrives: an unread JSON key is not a type error
// in either language, so a parser that never looks for "StackAlign" hands
// every layer the empty string and every mapping answers "centre". The result
// is a prop that is declared, documented, honoured on both web targets and
// silently inert on both phones — the exact failure this package exists to
// notice, and the one the coverage checks below cannot see.
func TestBothNativeParsersReadTheStackAlignment(t *testing.T) {
	for _, pin := range []struct {
		file string
		// The JSON lookup the parser must perform. The key is the Go field
		// name verbatim, since core.Style carries no json tags.
		key string
	}{
		{file: swiftStyle, key: `str("StackAlign")`},
		{file: kotlinStyle, key: `optString("StackAlign")`},
	} {
		src := readNative(t, pin.file)
		if !strings.Contains(src, pin.key) {
			t.Errorf("%s: never parses %s — core.StackAlign places a layer on the web and "+
				"does nothing on this platform", pin.file, pin.key)
		}
	}
}

// The centre must NOT have an arm, on either native.
//
// It is core.StackAlignment's zero value, so every node in every tree carries
// it, and both mappings answer for it by returning nil — the caller substitutes
// the centre. An arm returning it directly would look equivalent and is not: it
// makes "said nothing" and "asked for the centre" two states where the rest of
// the framework has one, and on Android it would replace the Modifier an
// unplaced layer has always been given. On iOS it used to be worse still, since
// nil was what kept a filling frame off an unplaced layer; the Layout that
// replaced that frame has no such cost, and the rule stands for the reason
// core.StackAlignments() excludes the centre in the first place.
//
// The coverage check above cannot say this — an unlisted arm is a failure
// there, but the empty string is not a value it would think to look for.
func TestNativeStackAlignmentsLeaveTheCentreToTheCatchAll(t *testing.T) {
	for _, c := range []struct {
		syntax dispatchSyntax
		file   string
	}{
		{swiftSwitch.with(swiftStack, "public func grMobStackAnchor(", "switch align {"), "GrMobStack.swift"},
		{kotlinWhen.with(kotlinRenderer, "private fun grMobStackAlignment(", "when (align) {"), "Renderer.kt"},
	} {
		for _, label := range c.syntax.labels(t) {
			if label == string(core.StackAlignCenter) {
				t.Errorf("%s: the placement mapping has an arm for the empty placement; the "+
					"centre is core.StackAlignCenter, which every node in every tree "+
					"carries, and an arm for it would be an arm for \"the default\"", c.file)
			}
		}
	}
}

// The two stack renderers must actually consult the mapping, and must apply it
// per layer rather than to the stack.
//
// The mapping being complete is worth nothing if nobody calls it, and neither
// native compiler would say so: an unused private function is a warning at
// most. What each renderer has to show is the read of the *child's* style —
// the placement is a layer's property and the stack's decision to honour it,
// which is the same split htmlout makes through its `imposed` channel.
func TestNativeZStacksPlaceEachLayer(t *testing.T) {
	for _, c := range []struct {
		file   string
		anchor string
		next   *regexp.Regexp
		want   []string
	}{
		{
			swiftRenderer, "private struct GrMobZStack", swiftCompositeStart, []string{
				// Read per child, off the child's own style.
				`grMobStackAnchor(child.style?.stackAlign ?? "")`,
				// SwiftUI has no per-child ZStack alignment, so the layer's
				// anchor rides to the Layout on a LayoutValueKey — a Layout
				// sees opaque subview proxies and cannot get back to the node.
				"GrMobStackPlacement.self",
			},
		},
		{
			kotlinRenderer, "private fun GrMobZStack", kotlinCompositeStart, []string{
				`grMobStackAlignment(child.style?.stackAlign ?: "")`,
				// BoxScope's own placement modifier: it places without
				// resizing, and a placed child still contributes its size to
				// the Box.
				"Modifier.align(placed)",
			},
		},
	} {
		// Bounded at the next composite, and comment-stripped, for the reason
		// dispatchArm gives: the prose beside these renderers explains what
		// they do, so an unbounded substring search would find the
		// explanation instead of the code.
		body := dispatchArm(t, c.file, c.anchor, c.next)
		for _, want := range c.want {
			if !strings.Contains(body, want) {
				t.Errorf("%s: %s does not contain %q — a core.StackAlign is honoured on the "+
					"web and dropped here", c.file, c.anchor, want)
			}
		}
	}
}
