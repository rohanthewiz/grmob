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

// The SwiftUI Layout hands both of its questions to the solver, and does no
// converting of its own on the way.
//
// This is the piece of the overlay that no harness can run.
// GrMobStackSolver's decisions moved into GrMobStack.swift precisely so
// ios/verify could execute them against a recording fake; what stayed behind
// is subviews.map, one sizeThatFits and the place() call, which a simulator is
// still the only thing that exercises.
//
// So this is a source-text pin, which is the fallback this package uses
// wherever a link can break silently and nothing off-device can see it (see
// value_test.go's pair). What it catches is the shape that would put a rule
// back out of reach, and there are two of those:
//
//	the Layout computes a size or an origin itself   the decisions
//	the Layout converts a proposal itself            the vocabulary
//
// The second half used to be the honest limit here — "it cannot catch a
// conversion that is subtly wrong" — and it is not any more, because the
// conversion stopped being written here. GrMobStackBridge.swift holds both
// directions and ios/verify runs them against real ProposedViewSize values, so
// what this file needs to say is only that they are still being *called*: an
// inlined `ProposedViewSize(width:height:)` anywhere in the Layout or its
// adapter would type-check, would work, and would put the pairing of the two
// axes back where nothing executes it.
//
// What no source check can reach is whether a real LayoutSubview answers
// sizeThatFits the way the recording fake does. That is SwiftUI's behaviour
// rather than this framework's, and TestNativeZStackOverlaysItsChildren plus a
// simulator are what speak to it.
func TestTheSwiftStackLayoutDelegatesToTheSolver(t *testing.T) {
	// The whole struct, not declSource: both of its methods are declarations,
	// so declSource stops at the first one and would read half the subject.
	// A type's body ends at the first closing brace in column one, since
	// everything inside it is indented.
	body := swiftTypeBody(t, swiftRenderer, "private struct GrMobStackLayout: Layout {")

	for _, pin := range []struct{ expr, question string }{
		{"GrMobStackSolver.containerSize(",
			"which proposal each layer is measured with, and whether the result may " +
				"be clamped to it"},
		{"GrMobStackSolver.placements(",
			"what every layer is re-measured and placed with, which is bounds.size " +
				"rather than the incoming proposal"},
	} {
		if !strings.Contains(body, pin.expr) {
			t.Errorf("%s: GrMobStackLayout never calls %s — it is deciding %s for "+
				"itself, where ios/verify cannot reach it", swiftRenderer, pin.expr,
				pin.question)
		}
	}

	// And it does not do the arithmetic on the way past. Both of these are
	// how the decisions were spelled before they moved, so both are what a
	// reinlining would look like.
	for _, banned := range []struct{ expr, why string }{
		{".max()", "taking the largest child here is the container-sizing rule, and it " +
			"belongs where a test can run it"},
		{"bounds.minX", "computing an origin here is the placement rule, same"},
	} {
		if strings.Contains(body, banned.expr) {
			t.Errorf("%s: GrMobStackLayout contains %q — %s", swiftRenderer,
				banned.expr, banned.why)
		}
	}

	// Nor the converting. Both spellings below are how the adapter read before
	// it moved into GrMobStackBridge.swift, and each is one axis-swap away
	// from being wrong in a way only a simulator would show — which is the
	// whole reason it moved.
	//
	// GrMobStackSubview is checked alongside the Layout because it held the
	// third of the three expressions: the Layout converted the incoming
	// proposal and the outgoing placement, and the adapter converted back for
	// sizeThatFits.
	adapter := swiftTypeBody(t, swiftRenderer,
		"private struct GrMobStackSubview: GrMobStackLayer {")
	// The positive half, which is also what keeps the negative half from
	// passing over a cut that read nothing: each converted direction is named
	// where it is used, so a body that lost its call fails here rather than
	// silently satisfying the bans below.
	for _, pin := range []struct{ name, src, expr string }{
		{"GrMobStackSubview", adapter, "subview.sizeThatFits(proposal.proposedViewSize)"},
		{"GrMobStackLayout", body, "proposing: GrMobProposal(proposal)"},
		{"GrMobStackLayout", body, "proposal: placement.proposal.proposedViewSize"},
	} {
		if !strings.Contains(pin.src, pin.expr) {
			t.Errorf("%s: %s does not contain %q — the conversion ios/verify runs is "+
				"not the one the Layout uses", swiftRenderer, pin.name, pin.expr)
		}
	}
	for _, part := range []struct{ name, src string }{
		{"GrMobStackLayout", body},
		{"GrMobStackSubview", adapter},
	} {
		for _, banned := range []struct{ expr, why string }{
			{"ProposedViewSize(width:",
				"a proposal built here is a pairing of two optional axes that nothing " +
					"runs; GrMobProposal.proposedViewSize is the one ios/verify checks"},
			{"GrMobProposal(width:",
				"same in the other direction — GrMobProposal.init(_ ProposedViewSize) " +
					"is the converted-in half, and it is checked next to the other"},
		} {
			if strings.Contains(part.src, banned.expr) {
				t.Errorf("%s: %s contains %q — %s", swiftRenderer, part.name,
					banned.expr, banned.why)
			}
		}
	}
}

// A whole Swift type declaration, from its opening line to the brace that
// closes it.
//
// declSource is the wrong cut for a type: its boundary is the next `func`, and
// a type's own methods are funcs — so it would return the header and the
// stored properties and stop before the bodies, which on these types is
// everything worth reading.
//
// # The cut this used to make, and what was wrong with it
//
// It looked for the first "\n}\n" — a closing brace in column one — on the
// grounds that every line inside a type body is indented. That is true of
// every Swift type in Renderer.swift and it is not true of Swift: indentation
// carries no meaning in the language, so the cut was a claim about this file's
// formatting standing in for a claim about its syntax. Three things would have
// broken it, none of them exotic:
//
//	a body line starting in column one    a `#if os(iOS)` block, or a wrapped
//	                                      expression an author did not indent
//	a multi-line string literal           `"""` content is verbatim, so a line
//	                                      of it may begin with `}`
//	a nested type formatted flat          legal, and gofmt has no Swift twin
//	                                      to prevent it
//
// The failure mode is the bad one: a short cut still returns a string, so
// every `strings.Contains` below would go on running against a body that had
// silently lost its second half, and the checks would pass by reading nothing.
// That is the same hazard parseRuntimeTable's proof-of-braces guards against,
// and the fix is the scanner this package already had for the other language's
// dispatch blocks.
//
// matchingBrace counts braces while skipping comments and string literals, so
// it answers the syntactic question directly and fails loudly when the block
// is unterminated. What is left here is finding the declaration's opening
// brace — the first one at or after the anchor, which for a type declaration
// is the one that opens its body. A generic parameter list or a conformance
// clause can sit in between (`struct GrMobFlexStack<Content: View>: View {`)
// and neither contains a brace.
func swiftTypeBody(t *testing.T, file, anchor string) string {
	t.Helper()
	src := readNative(t, file)
	at := strings.Index(src, anchor)
	if at < 0 {
		t.Fatalf("%s: no %s — if it was renamed or restructured, update this test "+
			"rather than deleting it", file, anchor)
	}
	rest := src[at:]
	open := strings.IndexByte(rest, '{')
	if open < 0 {
		t.Fatalf("%s: %s has no opening brace — the anchor is matching something "+
			"that is not a declaration", file, anchor)
	}
	// The header is kept in the result, as it was: the checks below read the
	// declaration line as well as the body.
	return rest[:open] + matchingBrace(t, file, anchor, rest[open:])
}

// The three shapes the old column-one cut would have got wrong.
//
// swiftTypeBody used to end a declaration at the first line consisting of a
// single `}`, which is a claim about how this repository indents rather than
// about Swift. The scanner it uses now answers the syntactic question, and
// these are the cases that tell the two apart — each one is a legal Swift type
// whose body contains a brace in column one, and on each the old cut returned
// a body missing everything after it while still returning a string, so every
// `strings.Contains` run against it would have passed by reading nothing.
//
// Synthetic sources rather than the renderer, deliberately. The renderer is
// formatted the way the old cut assumed, so it cannot exercise this — which is
// precisely why the assumption survived. matchingBrace takes its source as an
// argument, so the cases can be written out.
func TestTheSwiftTypeCutIsSyntacticNotTypographic(t *testing.T) {
	for _, c := range []struct{ name, src, want string }{
		{
			// A conditional-compilation directive, which Swift convention puts
			// in column one, followed by an ordinary nested block.
			name: "a directive in column one",
			src: "{\n#if os(iOS)\n    var body: some View {\n        Text(\"a\")\n" +
				"}\n#endif\n    let tail = 1\n}\n",
			want: "let tail = 1",
		},
		{
			// A multi-line string literal. Its content is verbatim, so a line
			// of it may begin with a brace, and no amount of reformatting the
			// file can move it.
			name: "a brace inside a multi-line literal",
			// The content carries an unpaired quote as well as the brace,
			// which is what makes the case need a `"""` arm of its own. A
			// scanner that knows only single-quoted strings pairs the three
			// delimiter characters off two at a time and comes out even
			// whenever the content holds an even number of quotes — so a
			// fixture with none, or with a quoted word in it, passes either
			// way and proves nothing. An odd one leaves the closing delimiter
			// half-consumed and the brace exposed.
			src:  "{\n    let s = \"\"\"\nan unpaired \" and then\n}\n\"\"\"\n    let tail = 2\n}\n",
			want: "let tail = 2",
		},
		{
			// A brace in a comment, which the scanner has always skipped and
			// which is worth keeping in this table: it is the case that made
			// the scanner exist for the other language's dispatch, and the
			// same source now serves both cuts.
			name: "a brace in a comment",
			src:  "{\n    // one closes like this: }\n    let tail = 3\n}\n",
			want: "let tail = 3",
		},
	} {
		got := matchingBrace(t, "synthetic.swift", c.name, c.src)
		if !strings.Contains(got, c.want) {
			t.Errorf("%s: the cut stopped early — %q is inside the type and the "+
				"body came back without it:\n%s", c.name, c.want, got)
		}
	}
}

// And the real declarations this package cuts come back balanced.
//
// The check above proves the scanner handles shapes the renderer does not
// contain; this one proves the renderer's own declarations are cut where a
// balanced count says they end, which is the property every `strings.Contains`
// in this package silently depends on. A cut that stopped early would leave
// more `{` than `}` in what it returned.
func TestEverySwiftTypeCutComesBackBalanced(t *testing.T) {
	for _, anchor := range []string{
		"private struct GrMobStackLayout: Layout {",
		"private struct GrMobSpacer",
		"private struct GrMobColumn",
		"struct GrMobFlexStack<Content: View>: View {",
	} {
		body := swiftTypeBody(t, swiftRenderer, anchor)
		// Comments and literals are stripped the same way the scanner skips
		// them, so the count is over code alone — the doc comments above these
		// declarations are full of prose braces.
		code := stringLiteral.ReplaceAllString(stripLineComments(body), `""`)
		if open, close := strings.Count(code, "{"), strings.Count(code, "}"); open != close {
			t.Errorf("%s: the cut of %q holds %d `{` and %d `}` — it ended somewhere "+
				"other than the declaration's closing brace, and every check "+
				"reading it is reading a fragment", swiftRenderer, anchor, open, close)
		}
	}
}
