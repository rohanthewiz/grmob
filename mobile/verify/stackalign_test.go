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
		// name verbatim, since core.Style's json tags set no names (they are all ,omitzero).
		key string
	}{
		{file: swiftStyle, key: `str("StackAlign")`},
		{file: kotlinStyle, key: `optString("StackAlign")`},
	} {
		src := valuesIn(t, pin.file)
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
	src := codeIn(t, file)
	at := swiftDeclIndex(t, file, src, anchor)
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

// swiftDeclIndex finds where a declaration the anchor names actually begins.
//
// # Why finding it was the half still done by substring
//
// The *cut* is syntactic — matchingBrace counts braces while skipping comments
// and literals, so it ends a declaration where Swift ends it rather than where
// this repository happens to indent. Finding the declaration's start was still
// `strings.Index`, which is the same class of assumption one step earlier: an
// anchor is a plain substring, and a substring matches a mention of the type as
// readily as the type.
//
// That is not exotic in this codebase. Every declaration here carries a doc
// comment, several of those comments name neighbouring types, and a comment
// like
//
//	/// A sibling of `private struct GrMobSpacer`, which ...
//	/// ...
//	private struct GrMobSpacer: View {
//
// makes the cut start inside the comment. The first `{` after that point is
// whatever the comment's prose contains or, failing that, the real one — and
// the returned "header" then holds a paragraph of English. Every
// `strings.Contains` below would still run, against a body that is not the
// declaration's, which is the same silent-fragment hazard the scanner replaced
// and the reason it is worth closing at both ends.
//
// # The rule
//
// A declaration begins at the start of a line, in code. So the anchor has to
// match at a position whose line holds nothing but whitespace before it, and
// which is not inside a comment or a string literal. Both halves are needed:
// the line-start test alone still admits a `///` comment line that begins with
// the anchor text, and the code test alone still admits a match halfway along
// a line of code.
//
// # Why an ambiguous anchor is a failure
//
// Two declaration-position matches means the cut is choosing one silently. That
// cannot happen for a Swift type in one file (the language forbids the
// redeclaration) but it can for an anchor that is a prefix — "private struct
// GrMobColumn" matches both `GrMobColumn` and a `GrMobColumnHeader` — and the
// caller who wrote the shorter string is the one who would never find out.
func swiftDeclIndex(t *testing.T, file, src, anchor string) int {
	t.Helper()

	found := swiftDeclIndices(src, anchor)
	switch len(found) {
	case 0:
		t.Fatalf("%s: no declaration begins with %q — if it was renamed or "+
			"restructured, update this test rather than deleting it.\n\n"+
			"A mention of it in a doc comment does not count: this looks for the "+
			"anchor at the start of a line, in code, because that is where a Swift "+
			"declaration begins and a substring match on a comment would hand every "+
			"check below a paragraph of English to search.", file, anchor)
	case 1:
		return found[0]
	}
	t.Fatalf("%s: %d declarations begin with %q, and the cut would take the first "+
		"silently. Anchors here are prefixes, so this is what a name that is the "+
		"beginning of another name looks like — lengthen it until it names one.",
		file, len(found), anchor)
	return 0
}

// swiftDeclIndices is the decision, as a function of two strings: every offset
// in src where a declaration begins with the anchor.
//
// Split from the t.Fatalf above so the answers can be handed over directly.
// Both of the interesting ones — a mention that must not count, and two matches
// that must not be resolved silently — are otherwise reachable only by owning a
// source file with the fault in it, and the renderers this package reads are
// written the way the old substring assumed. That is the same argument
// startupVerdict and localCopyGate were extracted on.
func swiftDeclIndices(src, anchor string) []int {
	code := maskSwiftNonCode(src)
	var found []int
	for at := 0; ; {
		i := strings.Index(code[at:], anchor)
		if i < 0 {
			return found
		}
		i += at
		at = i + 1
		// In code: the mask blanks comment and literal characters, so an anchor
		// that survived it is code. (A blanked run cannot match a non-blank
		// anchor, and every anchor here has non-space characters.)
		line := strings.LastIndexByte(code[:i], '\n') + 1
		if strings.TrimSpace(code[line:i]) != "" {
			continue // something else on the line first: not a declaration
		}
		found = append(found, i)
	}
}

// maskSwiftNonCode returns src with every comment and string-literal character
// replaced by a space, and the same length; maskComments blanks the comments
// and leaves the literals where they are.
//
// Same length is the whole point: offsets in the mask are offsets in the
// source, so a match found here can be used there. It is also what lets
// matchingBrace count braces over the mask rather than carrying a scanner of
// its own — see maskNonCode, which is where the two used to be a copy apiece of
// one answer to "what in this file is code".
//
// Both spellings drop the scan's second return. It names an unterminated
// comment or literal, which is the counter's concern: these two are applied to
// files a compiler has already accepted.
//
// # Why there are two levels rather than one
//
// The checks in this package ask two different questions of a renderer, and
// they want opposite things from a string literal:
//
//	does it CALL this          `strings.Contains(body, "pinMainAxis(")`. A doc
//	                           comment saying the renderer calls it satisfies
//	                           that, which is the failure codeOf was written
//	                           for — and so would a string literal naming it.
//	                           Both go.
//
//	does it LIST this VALUE    a dispatch's arms are string literals ("center",
//	                           "flex-end"), and they are the subject. Blanking
//	                           them would delete the thing being read and leave
//	                           a parse that finds nothing, which is the one
//	                           result a check here must never produce quietly.
//
// Comments are noise to both, so that half is unconditional and every caller
// gets it. The literal half is the caller's choice, and each of the two
// spellings is named after what it keeps.
func maskSwiftNonCode(src string) string { masked, _, _ := maskNonCode(src, true); return masked }

// maskComments is the weaker mask: prose out, literals in.
//
// This is what declSource applies to every region it hands out, so that the
// doc comment its deliberately coarse cut carries along cannot answer a
// question about code. See declSource for why the cut is coarse and why the
// comment that rides on it was a real defect rather than an untidiness.
func maskComments(src string) string { masked, _, _ := maskNonCode(src, false); return masked }

// maskNonCode is the scanner every question about "what in this file is code"
// goes through.
//
// A literal is always *scanned* — its extent has to be known either way, or a
// `//` or a brace inside one would be read as code — and `literals` decides
// only whether it is also blanked. That asymmetry is the reason this is one
// function with a flag rather than two scanners: the two differ by a single
// conditional.
//
// # The third caller, and why it is not a third scanner
//
// matchingBrace used to be one. It counted braces while skipping comments and
// literals, which is this scan with the counting folded in — same four arms, in
// the same order, spelled the same way — and the two files each carried a
// comment asking the copies to agree. Nothing made them, and the failure of a
// drifted copy is the silent one: a brace-counter that mishandled `"""` would
// end a block early and hand every `strings.Contains` below it a fragment,
// still returning a string.
//
// So the counting is now done OVER the mask rather than beside it. Blanking
// preserves offsets and length, so an index into the mask is an index into the
// source, and a brace that survived the mask is a brace in code. That is the
// whole of the merge: matchingBrace holds the brace arms, this holds the
// not-code arms, and there is one answer to the question again.
//
// # unterminated
//
// The one thing the counter needs that a mask does not. A file that ends inside
// a block comment or a multi-line literal is a fault matchingBrace has always
// reported by name — the alternative is a mask that quietly blanks the rest of
// the file and a count that comes out wherever it comes out — so the scan names
// what was left open and the caller decides. The two mask spellings ignore it:
// their inputs are files a compiler has already accepted, and an unterminated
// construct there is a build failure long before it is a test failure.
//
// A `//` running to end of file is not one of these. It terminates at EOF by
// definition, and so does a double-quoted literal as far as this scan is
// concerned — matchingBrace never reported either. An unpaired APOSTROPHE is
// not one either, and for a stronger reason: that arm declines to read it as a
// literal at all, so nothing is blanked and nothing is left open.
//
// # comments
//
// The other thing a mask cannot say, and the reason proseSourceOf asked for it.
//
// Blanking preserves offsets and length, which is what makes an index into the
// mask an index into the source — and it throws away one distinction on the
// way: a line that is EMPTY inside a block comment is blank in the mask, and so
// is a line that is empty because it separates two paragraphs. Those are
// opposite answers to "does this declaration's note continue above here", and
// a walk that reads the mask alone gets the second one for both.
//
// So the scan says where its comments were. It is the same shape `unterminated`
// is — one more thing the scan already knows and the mask cannot carry — rather
// than a second pass over the file, which is the arrangement this whole
// function exists to be instead of.
//
// # The fourth language
//
// Three of the four this serves spell their literals identically — Swift,
// Kotlin, and Java by inheritance. Groovy does not: it has an apostrophe string
// and a tripled one alongside the double-quoted pair, and this scan did not
// know either.
//
// That was found by accident rather than by looking. mobile/verify reads
// android/app/build.gradle through the literals-KEPT mask, so the coordinates
// it looks for survived a scan that could not see them and the reading was
// right for a reason nobody had written down. Note which direction the gap ran:
// the risk was never a check that failed, it was a check whose subject is a
// string literal in a language the scan believed had none — a coordinate quoted
// in a comment satisfying a question asked under a reader named for code.
//
// Groovy has a THIRD form, and it is here for that reason rather than because
// anything writes one. `$/…/$` is the dollar-slashy string: multi-line,
// backslash-free, and the form somebody reaches for the day a coordinate or a
// path in build.gradle needs one. Nothing in this repository uses it — which is
// precisely the state the apostrophe was in, and the argument for adding the
// arm before rather than after a file starts depending on it.
func maskNonCode(src string, literals bool) (masked, unterminated string, comments []span) {
	out := []byte(src)
	comment := func(from, to int) { comments = append(comments, span{from, to}) }
	blank := func(from, to int) {
		for i := from; i < to && i < len(out); i++ {
			if out[i] != '\n' {
				out[i] = ' '
			}
		}
	}
	// The literal arms call this instead, so that skipping and blanking stay
	// one decision made in one place.
	blankLiteral := func(from, to int) {
		if literals {
			blank(from, to)
		}
	}
	for i := 0; i < len(src); i++ {
		switch {
		case strings.HasPrefix(src[i:], "//"):
			nl := strings.IndexByte(src[i:], '\n')
			if nl < 0 {
				blank(i, len(src))
				comment(i, len(src))
				return string(out), "", comments
			}
			blank(i, i+nl)
			comment(i, i+nl)
			i += nl
		case strings.HasPrefix(src[i:], "/*"):
			end := strings.Index(src[i+2:], "*/")
			if end < 0 {
				blank(i, len(src))
				comment(i, len(src))
				return string(out), "block comment", comments
			}
			blank(i, i+2+end+2)
			comment(i, i+2+end+2)
			i += 2 + end + 1
		case strings.HasPrefix(src[i:], `"""`):
			// Matched before the single-quote arm because that arm would read
			// `"""` as an empty string followed by an opening quote, and would
			// then take the first `"` of the *closing* delimiter as the end.
			// That happens to come out even for content with no quote in it,
			// and stops doing so for content with one.
			end := strings.Index(src[i+3:], `"""`)
			if end < 0 {
				blankLiteral(i, len(src))
				return string(out), "multi-line string", comments
			}
			blankLiteral(i, i+3+end+3)
			i += 3 + end + 2
		case src[i] == '"':
			// Escapes are honored so that a literal ending in \" does not read
			// as still open, which would swallow the rest of the file.
			j := i + 1
			for j < len(src) && src[j] != '"' {
				if src[j] == '\\' {
					j++
				}
				j++
			}
			blankLiteral(i, j+1)
			i = j
		case strings.HasPrefix(src[i:], "$/"):
			// Groovy's dollar-slashy string, which is the third literal form
			// and the one nothing here writes today. That is the state `'...'`
			// was in until somebody looked: the risk a mask runs is not a check
			// that fails, it is a check whose subject is a literal the scan
			// believes is code.
			//
			// Multi-line by construction — it exists so a regexp or a path can
			// carry backslashes and newlines unescaped — so it is bounded at
			// `/$` rather than at the newline, and an unterminated one is named
			// the way the other two multi-line forms are.
			//
			// The escapes are the dollar's: `$$` is a dollar and `$/` is a
			// slash, and both are honored for the reason the double-quote arm
			// honors a backslash. Without that, content holding an escaped
			// slash would end the literal early and leave the rest of it read
			// as code.
			j := i + 2
			closed := false
			for j < len(src) {
				if src[j] == '$' && j+1 < len(src) && (src[j+1] == '$' || src[j+1] == '/') {
					j += 2
					continue
				}
				if src[j] == '/' && j+1 < len(src) && src[j+1] == '$' {
					closed = true
					break
				}
				j++
			}
			if !closed {
				blankLiteral(i, len(src))
				return string(out), "dollar-slashy string", comments
			}
			blankLiteral(i, j+2)
			i = j + 1
		case strings.HasPrefix(src[i:], "'''"):
			// Groovy's multi-line literal, matched before the apostrophe arm
			// for the reason the triple-double-quote arm is matched before the
			// double-quote one: that arm would read the opening delimiter as an
			// empty string followed by a third quote.
			end := strings.Index(src[i+3:], "'''")
			if end < 0 {
				blankLiteral(i, len(src))
				return string(out), "multi-line string", comments
			}
			blankLiteral(i, i+3+end+3)
			i += 3 + end + 2
		case src[i] == '\'':
			// Groovy's ordinary string, and Kotlin's and Java's character
			// literal. See the paragraph above about the fourth language.
			//
			// Bounded at the newline, which is the one place this differs from
			// the double-quote arm. None of the four languages lets such a
			// literal span a line, and the bound is what makes the arm safe to
			// add: an apostrophe that is NOT a delimiter — which cannot happen
			// in code a compiler accepted, and can happen in a file this scan
			// is pointed at by mistake — blanks nothing rather than swallowing
			// the rest of the file. An unpaired one leaves its line as code,
			// which is the direction a mask has to fail in: a check can be
			// weakened by reading a literal as code, and can be silently
			// satisfied by a region blanked to nothing.
			j := i + 1
			for j < len(src) && src[j] != '\'' && src[j] != '\n' {
				if src[j] == '\\' && j+1 < len(src) && src[j+1] != '\n' {
					j++
				}
				j++
			}
			if j >= len(src) || src[j] != '\'' {
				break // an apostrophe, not a literal
			}
			blankLiteral(i, j+1)
			i = j
		}
	}
	return string(out), "", comments
}

// The mask and the brace counter are one scanner, and this is what says so.
//
// They were two. maskNonCode blanked comments and literals; matchingBrace
// skipped them while counting braces — four arms apiece, in the same order,
// with the same words — and each carried a comment asking the other to agree.
// A comment cannot make two copies agree, and the way they come apart is
// silent: the counter is the one whose mistakes still return a string.
//
// The counter now counts over the mask, so the agreement holds by construction.
// This is the statement of it, and it is what fails if somebody gives either
// side a scanner of its own again. Each case hides the same two things in the
// same construct — a `}` the counter must not count, and an anchor the mask
// must not find — so a construct one side handles and the other does not is a
// failure on that row rather than a divergence nobody looks for.
func TestTheMaskAndTheBraceCounterAgreeAboutWhatIsCode(t *testing.T) {
	// The anchor is hidden inside the construct and then declared for real
	// below it, so "found once, at the declaration" and "the block runs to the
	// end" are the same claim about the same characters.
	const anchor = "private struct GrMobHidden"

	for _, c := range []struct{ name, hidden string }{
		{
			name:   "a line comment",
			hidden: "    // private struct GrMobHidden closes like this: }\n",
		},
		{
			name:   "a block comment",
			hidden: "    /*\nprivate struct GrMobHidden }\n*/\n",
		},
		{
			// Verbatim content, so it may legally begin a line with either.
			//
			// The unpaired quote before the hidden text is what makes this row
			// discriminate, and it is the same trick
			// TestTheSwiftTypeCutIsSyntacticNotTypographic uses: a scanner that
			// knows only double-quoted strings pairs the three delimiter
			// characters off two at a time and comes out even whenever the
			// content holds an even number of quotes — so a fixture with none
			// has its content blanked by accident and passes either way. An odd
			// one leaves everything after it exposed.
			name:   "a multi-line literal",
			hidden: "    let s = \"\"\"\nan unpaired \" then private struct GrMobHidden }\n\"\"\"\n",
		},
		{
			name:   "a double-quoted literal",
			hidden: "    let s = \"private struct GrMobHidden }\"\n",
		},
		{
			// Groovy's ordinary string, in the delimiter the scan learned last.
			// Same hidden brace, same hidden anchor: the row that would have
			// passed on both counts before the apostrophe arm existed — the
			// mask leaving the anchor visible and the counter counting the
			// brace — is the row that says the arm is there.
			name:   "an apostrophe literal",
			hidden: "    def s = 'private struct GrMobHidden }'\n",
		},
		{
			// And its tripled form, carrying an unpaired apostrophe for the
			// reason the multi-line row above carries an unpaired quote: a
			// scanner pairing delimiters off two at a time comes out even
			// whenever the content holds an even number of them, so a fixture
			// with none would pass either way.
			name:   "a Groovy multi-line literal",
			hidden: "    def s = '''\nan unpaired ' then private struct GrMobHidden }\n'''\n",
		},
		{
			// Groovy's third form. The content carries an unpaired apostrophe
			// AND an unpaired quote, so a scan that fell through to either of
			// the delimiter arms would take one of them as an opening
			// delimiter and run somewhere else entirely — which is the failure
			// this row is meant to be able to see.
			name:   "a dollar-slashy literal",
			hidden: "    def s = $/\nan unpaired ' and \" then private struct GrMobHidden }\n/$\n",
		},
		{
			// The escapes, which are what keep the arm from ending the literal
			// early. Content holding the two characters `/$` is written `$/$$`
			// — the slash escaped as `$/`, the dollar escaped as `$$` — and the
			// middle two characters of that are a terminator to anything not
			// reading the dollars. A scan without them closes the literal four
			// bytes in and leaves the brace and the anchor standing as code,
			// which is the same failure the unpaired-quote rows above are
			// about, one escape convention over.
			name:   "a dollar-slashy literal with an escaped terminator",
			hidden: "    def s = $/a$/$$b private struct GrMobHidden }\n/$\n",
		},
	} {
		src := "{\n" + c.hidden + "    let tail = 1\n}\n" +
			anchor + ": View {\n    let real = 2\n}\n"

		// The counter: the hidden `}` is not the block's, so the block still
		// holds everything up to the real one.
		if body := matchingBrace(t, "synthetic.swift", c.name, src); !strings.Contains(body, "let tail = 1") {
			t.Errorf("%s: the brace counter ended the block at the hidden `}` — its "+
				"whole body is %q and what came back was:\n%s",
				c.name, "let tail = 1", body)
		}

		// The mask: the hidden anchor is not a declaration, so exactly the one
		// below it is found. Two would mean the mask read the construct as
		// code; zero would mean it blanked more than the construct.
		if found := swiftDeclIndices(src, anchor); len(found) != 1 {
			t.Errorf("%s: the mask found %d declarations of %q and there is one — the "+
				"other is inside the construct this row hides it in, which the brace "+
				"counter skipped and the mask did not",
				c.name, len(found), anchor)
		}
	}
}

// A file that ends inside a comment or a literal is a fault with a name.
//
// It is also the half of the scan that only the counter ever cared about:
// matchingBrace has always refused an unterminated `/*` or `"""` rather than
// counting whatever braces happened to follow, and folding the skipping into
// the mask would have thrown that away if the mask could not report it. So it
// reports it, and this is the arm — reachable as a value now, where before it
// was a t.Fatalf inside the scanner and could only be checked by owning a
// source file with the fault.
//
// The three that are NOT faults are here for the same reason the two that are:
// a `//` running to end of file terminates at EOF by definition, and so does a
// double-quoted literal as far as this scan is concerned. An unpaired
// APOSTROPHE is a third kind and a different one — the arm declines to read it
// as a literal at all, so there is nothing left open to name — and reporting
// any of the three would turn an ordinary last line into a failure.
func TestTheScanNamesWhatWasLeftOpen(t *testing.T) {
	for _, c := range []struct{ name, src, want string }{
		{name: "an unterminated block comment", src: "{\n/* on and on", want: "block comment"},
		{name: "an unterminated multi-line literal", src: "{\nlet s = \"\"\"\nand on", want: "multi-line string"},
		{name: "a line comment at end of file", src: "{\n// the last line", want: ""},
		{name: "a double-quoted literal at end of file", src: "{\nlet s = \"and on", want: ""},
		{name: "an unterminated Groovy multi-line literal", src: "{\ndef s = '''\nand on",
			want: "multi-line string"},
		// The third form is multi-line by construction, so an unclosed one is
		// the same kind of fault as an unclosed `'''` and gets a name of its
		// own rather than the shared one: the two are bounded by different
		// delimiters, and a reader told "multi-line string" would go looking
		// for the wrong character.
		{name: "an unterminated dollar-slashy literal", src: "{\ndef s = $/and on",
			want: "dollar-slashy string"},
		// And a dollar at end of file, which is not one. The arm needs two
		// characters to open at all, so a trailing `$` is code — the same
		// direction the apostrophe arm fails in, and for the same reason.
		{name: "a dollar at end of file", src: "{\ndef s = a$", want: ""},
		// The arm that reports nothing rather than reporting a fault. An
		// apostrophe with no partner on its line is not a literal, so the scan
		// leaves the line as code and has nothing to name — which is what keeps
		// the arm from turning a stray quote into a failure in a language where
		// it is prose.
		{name: "an apostrophe with no partner", src: "{\ndef s = it's fine\n}\n", want: ""},
		{name: "an ordinary block", src: "{\n    let a = 1\n}\n", want: ""},
	} {
		if _, got, _ := maskNonCode(c.src, true); got != c.want {
			t.Errorf("%s: the scan reported %q, want %q", c.name, got, c.want)
		}
	}
}

// A doc comment that names a type does not start it.
//
// The half of swiftTypeBody that was still typographic, and the one that
// survives a scanner: the cut is syntactic, and finding the declaration to cut
// was `strings.Index` — a plain substring, which matches a mention of the type
// as readily as the type. Every declaration in these renderers carries a doc
// comment and several of them name their neighbours, so this is not an exotic
// shape; it is the shape the file is already full of.
//
// The failure is the same silent-fragment one the scanner was written to end. A
// cut starting inside a comment still returns a string, the first brace after it
// is whatever the prose happens to contain, and every `strings.Contains` below
// goes on running against a paragraph of English.
//
// Synthetic sources, for the reason the table below gives: the renderers are
// written the way the old code assumed, which is exactly why the assumption
// survived.
func TestASwiftAnchorMustStartADeclarationAndNotMentionOne(t *testing.T) {
	const decl = "private struct GrMobSpacer"

	for _, c := range []struct{ name, src, want string }{
		{
			// The case that motivated it: a doc comment naming the type it
			// documents, above the type.
			name: "a doc comment naming the type above it",
			src: "/// A sibling of private struct GrMobSpacer, which does the\n" +
				"/// same job one axis over.\n" +
				"private struct GrMobSpacer: View {\n    let tail = 1\n}\n",
			want: "let tail = 1",
		},
		{
			// A block comment whose continuation line begins in column one,
			// which is the case the line-start test alone cannot tell from a
			// declaration — and the reason the mask exists rather than just a
			// "starts a line" rule.
			name: "a block comment whose line begins with the type",
			src: "/*\nprivate struct GrMobSpacer is documented here\n*/\n" +
				"private struct GrMobSpacer: View {\n    let tail = 2\n}\n",
			want: "let tail = 2",
		},
		{
			// The same shape in a multi-line literal, whose content is verbatim
			// and so may begin a line with anything at all. The other half of
			// what only the mask can reject.
			name: "a multi-line literal whose line begins with the type",
			src: "let note = \"\"\"\nprivate struct GrMobSpacer\n\"\"\"\n" +
				"private struct GrMobSpacer: View {\n    let tail = 3\n}\n",
			want: "let tail = 3",
		},
		{
			// And a mention mid-line in code, which is what only the line-start
			// test rejects: the mask leaves it alone, because it IS code.
			name: "a mention part-way along a line of code",
			src: "let kind = describing(private struct GrMobSpacer.self)\n" +
				"private struct GrMobSpacer: View {\n    let tail = 4\n}\n",
			want: "let tail = 4",
		},
	} {
		at := swiftDeclIndex(t, "synthetic.swift", c.src, decl)
		rest := c.src[at:]
		open := strings.IndexByte(rest, '{')
		if open < 0 {
			t.Errorf("%s: the anchor landed somewhere with no brace after it", c.name)
			continue
		}
		body := matchingBrace(t, "synthetic.swift", c.name, rest[open:])
		if !strings.Contains(body, c.want) {
			t.Errorf("%s: the cut did not reach the declaration — %q is its whole "+
				"body and what came back was:\n%s", c.name, c.want, body)
		}
	}
}

// An anchor that names two declarations is a failure, not a coin toss.
//
// Anchors here are prefixes, so "private struct GrMobColumn" would match a
// `GrMobColumnHeader` beside it. The cut taking the first silently is the shape
// where the person who wrote the short anchor never finds out, so swiftDeclIndex
// refuses it. The count is what this asserts: the refusal itself is a t.Fatalf,
// which is why the decision is a separate function taking two strings.
func TestAnAmbiguousSwiftAnchorIsRefused(t *testing.T) {
	src := "private struct GrMobColumn: View {\n    let a = 1\n}\n" +
		"private struct GrMobColumnHeader: View {\n    let b = 2\n}\n"

	if got := swiftDeclIndices(src, "private struct GrMobColumn"); len(got) != 2 {
		t.Errorf("the prefix anchor found %d declarations, want 2 — it names both "+
			"GrMobColumn and GrMobColumnHeader, and swiftDeclIndex refuses that "+
			"rather than taking whichever comes first in the file", len(got))
	}
	// And the longer anchor names one, which is what the failure tells the
	// caller to write.
	if got := swiftDeclIndices(src, "private struct GrMobColumnHeader"); len(got) != 1 {
		t.Errorf("the unambiguous anchor found %d declarations, want 1", len(got))
	}
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
			// scanner that knows only double-quoted strings pairs the three
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
		// Counted over the mask, which is the same scan matchingBrace made the
		// cut with — so this is the cut's own answer to "what is code" checked
		// against itself rather than a fourth one. It matters: the doc comments
		// above these declarations are full of prose braces, and the two
		// constructs an ad-hoc strip misses (a block comment, a `"""`) are
		// exactly the ones a hand-written count gets wrong.
		code := maskSwiftNonCode(body)
		if open, close := strings.Count(code, "{"), strings.Count(code, "}"); open != close {
			t.Errorf("%s: the cut of %q holds %d `{` and %d `}` — it ended somewhere "+
				"other than the declaration's closing brace, and every check "+
				"reading it is reading a fragment", swiftRenderer, anchor, open, close)
		}
	}
}

// span is a half-open byte range of the source, [from, to).
//
// One shape for the one thing maskNonCode reports positionally. Kept beside the
// scanner rather than beside its caller because it is the scanner's answer: the
// arms know where each construct began and ended, and everything downstream
// only ever asks whether an offset is inside one.
type span struct{ from, to int }
