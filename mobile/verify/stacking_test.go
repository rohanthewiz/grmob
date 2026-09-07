package verify

import (
	"regexp"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/htmlout"
)

// The start of a type-dispatch arm in each renderer, used to find where the
// arm *before* it ends. Written generically rather than anchored on whichever
// arm currently follows, so reordering the dispatch does not silently widen
// what a check reads.
//
//	Swift    case "TabView": …
//	Kotlin   "Column", "Card", "Box" -> …
var (
	swiftArmStart  = regexp.MustCompile(`(?m)^\s*case "`)
	kotlinArmStart = regexp.MustCompile(`(?m)^\s*"[A-Za-z]+"(?:, "[A-Za-z]+")* ->`)
)

// The start of the *next* top-level composite, for the checks below that read
// one of the composite views rather than a dispatch arm. A composite is not
// bounded by the arm regexps above — it holds no `case`/`->` labels of its own
// — and switchlabels_test.go's declStart stops only at a function, which walks
// straight through a Swift `struct`.
var (
	swiftCompositeStart  = regexp.MustCompile(`(?m)^private struct `)
	kotlinCompositeStart = regexp.MustCompile(`(?m)^private fun `)
)

// dispatchArm returns the *code* between a marker and the next match of next,
// with line comments removed. Its first use is one arm of a renderer's
// node-type dispatch — marker is the arm's label, next the start of any arm —
// and it bounds a whole composite declaration on the same terms, with next
// spelling the start of the following declaration instead.
//
// Stripping the comments is not tidiness. These arms are checked for the
// absence of a construct, and the arms now carry comments explaining which
// construct they stopped using — so the prose spells the very word the check
// forbids, and the first run of this test failed on its own explanation.
// declSource can afford to keep comments because the substrings held against
// it are expression fragments prose does not accidentally write; a bare type
// name is not one of those.
//
// `//` inside a string literal would be stripped too. Nothing in either
// dispatch contains one — the literals are node type and prop names — and the
// positive half of every check would fail loudly if a strip ever ate real code.
func dispatchArm(t *testing.T, file, marker string, next *regexp.Regexp) string {
	t.Helper()

	src := readNative(t, file)
	at := strings.Index(src, marker)
	if at < 0 {
		t.Fatalf("%s: no %s — if it was renamed or restructured, update this test", file, marker)
	}
	rest := src[at+len(marker):]
	if end := next.FindStringIndex(rest); end != nil {
		rest = rest[:end[0]]
	}
	return stripLineComments(marker + rest)
}

// stripLineComments removes `//`-to-end-of-line from each line, keeping the
// line breaks so the result still reads as source.
func stripLineComments(src string) string {
	lines := strings.Split(src, "\n")
	for i, line := range lines {
		if at := strings.Index(line, "//"); at >= 0 {
			lines[i] = line[:at]
		}
	}
	return strings.Join(lines, "\n")
}

// The container node types stack their children; they are not overlays.
//
// This is the divergence the gap sweep exposed rather than a gap bug of its
// own. Box and SafeArea were built from a Compose `Box` and a SwiftUI
// `ZStack` — both of which lay children on top of one another at the
// top-start corner — while both DOM targets stack them down the page (the
// WASM runtime lists Box and SafeArea in STACK_CONTAINERS). core.Box is
// documented as one of the flex-style containers, sharing Row/Column/Card/
// List's argument contract and differing from Column only in carrying no
// theme base, so the natives were the outlier and two children of one Box
// drew on top of each other on device.
//
// The single-child case diverged too, which is the half no overlap would have
// revealed: an overlay container lets its child size to its own content, so a
// screen's content column hugged its widest child rather than filling the
// width `align-items: stretch` gives it on the web. Routing through the
// Column implementation is what supplies the stretch, so "routes to
// GrMobColumn" is the claim worth pinning — not merely "is not a ZStack".
//
// The negative half is checked on the arm rather than the file, because both
// overlay constructs stay in use elsewhere in each renderer: CameraView is a
// genuine overlay on both natives, and Modal is one on Android. It is checked
// on the arm's code rather than its text, because the arms now carry comments
// naming what they stopped being — see dispatchArm.
func TestNativeContainersStackTheirChildrenAndDoNotOverlay(t *testing.T) {
	for _, pin := range []struct {
		file, marker string
		next         *regexp.Regexp
		// overlay is the construct this arm must no longer be built with.
		overlay string
	}{
		// Box shares Column's arm outright, which is the whole of its fix.
		{swiftRenderer, `case "Column", "Card", "Box":`, swiftArmStart, "ZStack"},
		{kotlinRenderer, `"Column", "Card", "Box" ->`, kotlinArmStart, "Box("},
		// SafeArea keeps its own arm — it carries chrome a Column has no
		// business knowing about (the window insets, the edge-to-edge
		// background, Android's system-bar icon appearance) — so what is
		// pinned is that the stacking underneath it is Column's.
		{swiftRenderer, `case "SafeArea":`, swiftArmStart, "ZStack"},
		{kotlinRenderer, `"SafeArea" ->`, kotlinArmStart, "Box("},
	} {
		arm := dispatchArm(t, pin.file, pin.marker, pin.next)
		if !strings.Contains(arm, "GrMobColumn(") {
			t.Errorf("%s: the %s arm does not route to GrMobColumn — its children are "+
				"either overlaid or stacked by a second implementation that no alignment "+
				"or gap check reaches", pin.file, pin.marker)
		}
		if strings.Contains(arm, pin.overlay) {
			t.Errorf("%s: the %s arm builds a %s — its children draw on top of each other, "+
				"and a lone child sizes to its content instead of filling the width, "+
				"while both DOM targets stack and stretch", pin.file, pin.marker, pin.overlay)
		}
	}
}

// The one container that *is* an overlay, and the counterpart to the check
// above: core.ZStack exists precisely because Box stopped being one.
//
// Positive and negative halves, as the container check has. The positive half
// is the construct — a SwiftUI ZStack and a Compose Box are what the node type
// was named for on each side. The negative half is the *centring*, which is
// the part a renderer diverges on silently: a Compose Box defaults to TopStart
// while a SwiftUI ZStack and a CSS grid cell both centre, so an arm that built
// the right construct and said nothing about alignment would draw the same
// tree three different ways on four targets. Both are therefore stated in each
// renderer, and both are read here.
//
// The alignment substrings are the renderers' own spellings rather than a
// shared token, because there is no shared token to have: SwiftUI names an
// argument and Compose names a parameter, and pretending otherwise would mean
// checking neither.
func TestNativeZStackOverlaysItsChildren(t *testing.T) {
	for _, pin := range []struct {
		file, marker string
		next         *regexp.Regexp
		// overlay is the construct the arm must be built from, and align the
		// centring it must state.
		overlay, align string
	}{
		// The Swift side is a custom Layout rather than a SwiftUI ZStack, and
		// the centre it must state is the *fallback anchor* rather than a
		// container alignment. Both changed together and for one reason: a
		// SwiftUI ZStack can only place a layer by wrapping it in a filling
		// frame, and a filling frame made an unsized stack greedy on this
		// target alone (see GrMobStack.swift). GrMobStackLayout is what
		// replaced it, so the construct to look for is the layout, and the
		// centring is the `?? .center` every unplaced layer takes.
		{swiftRenderer, "private struct GrMobZStack", swiftCompositeStart,
			"GrMobStackLayout {", "GrMobStackPlacement.self"},
		{kotlinRenderer, "private fun GrMobZStack", kotlinCompositeStart, "Box(", "contentAlignment = Alignment.Center"},
	} {
		body := dispatchArm(t, pin.file, pin.marker, pin.next)
		if !strings.Contains(body, pin.overlay) {
			t.Errorf("%s: %s builds no %s — a core.ZStack lays its layers out along an axis "+
				"here while both DOM targets draw them on top of each other",
				pin.file, pin.marker, pin.overlay)
		}
		if !strings.Contains(body, pin.align) {
			t.Errorf("%s: %s does not state %q — core.ZStack's alignment contract is a fixed "+
				"centre, and a renderer that leaves it to its own default is the one that drifts",
				pin.file, pin.marker, pin.align)
		}
	}

	// The Swift stack must not go back to a filling frame around a layer.
	//
	// That is the construct this target's one divergence was made of, and it
	// is the obvious thing to reach for the next time somebody needs a layer
	// placed — it is SwiftUI's own idiom, and it works, and it quietly makes
	// an unsized stack greedy on iOS alone. The Layout that replaced it has
	// no such failure available to it, so what needs guarding is the return.
	zstack := dispatchArm(t, swiftRenderer, "private struct GrMobZStack", swiftCompositeStart)
	if strings.Contains(zstack, "maxWidth: .infinity") {
		t.Error("Renderer.swift: GrMobZStack wraps a layer in a filling frame again — " +
			"that frame is greedy, so an unsized stack with an aligned layer grows to " +
			"its parent's proposal here and stays the size of its largest child on the " +
			"other three targets. Place by coordinate through GrMobStackSolver instead")
	}
}

// The dispatch has to reach the composite above. A composite nothing routes to
// is dead code that every check in this file would still pass.
func TestNativeZStackIsDispatched(t *testing.T) {
	for _, pin := range []struct {
		file, marker string
		next         *regexp.Regexp
	}{
		{swiftRenderer, `case "ZStack":`, swiftArmStart},
		{kotlinRenderer, `"ZStack" ->`, kotlinArmStart},
	} {
		arm := dispatchArm(t, pin.file, pin.marker, pin.next)
		if !strings.Contains(arm, "GrMobZStack(") {
			t.Errorf("%s: the %s arm does not route to GrMobZStack — a ZStack falls through to "+
				"the renderer's default arm, which is a vertical stack", pin.file, pin.marker)
		}
	}
}

// TabView is a column stack on both natives, which is the claim htmlout's
// stackAxes row for it rests on — and the last of the four targets' stacking
// disagreements: TabView stacked here and ran in block flow on the two DOM
// targets, where a <div>'s inline children share a line. The row was held out
// of that table's first cut because the two web targets at least agreed with
// each other; this pin is what lets it go in on evidence rather than on the
// prose in stack.go.
//
// Positional rather than merely present, because both composites hold a
// horizontal stack — the tab bar — inside the vertical one. "Builds a VStack"
// would pass just as well on a renderer that had the two the wrong way round,
// and that renderer would be laying the bar out beside the page instead of
// above it. Comparing the offsets is what says which one is the outer box,
// and that box's axis is what the table row states.
//
// The tab bar is located by its own construct rather than by a second VStack
// or Column, so that a missing bar fails as a missing bar instead of quietly
// leaving the ordering check with nothing to compare against.
func TestNativeTabViewIsAColumnStack(t *testing.T) {
	for _, pin := range []struct {
		file, marker string
		next         *regexp.Regexp
		// outer is the vertical stack the composite must be built from;
		// bar is the horizontal one it must contain, not be contained by.
		outer, bar string
	}{
		{swiftRenderer, "private struct GrMobTabView", swiftCompositeStart, "VStack(", "HStack("},
		{kotlinRenderer, "private fun GrMobTabView", kotlinCompositeStart, "Column(", "TabRow("},
	} {
		body := dispatchArm(t, pin.file, pin.marker, pin.next)
		outer := strings.Index(body, pin.outer)
		bar := strings.Index(body, pin.bar)
		switch {
		case outer < 0:
			t.Errorf("%s: %s builds no %s — htmlout/stack.go stacks TabView along the column "+
				"axis on the strength of this renderer doing so", pin.file, pin.marker, pin.outer)
		case bar < 0:
			t.Errorf("%s: %s builds no %s — the tab bar this check locates is gone, so the "+
				"ordering below would compare against nothing", pin.file, pin.marker, pin.bar)
		case outer > bar:
			t.Errorf("%s: %s opens with %s, not %s — the tab bar is the outer box, so the "+
				"composite stacks along the row axis and the DOM targets' column row is wrong",
				pin.file, pin.marker, pin.bar, pin.outer)
		}
	}
}

// A Spacer's children reach both natives, and reach them as a column.
//
// # What this closes
//
// core.Spacer(n) builds a node with a size prop and nothing else, so a Spacer
// with children only ever arrives by hand-assembly — the same position
// core.ModalNode is in, and the same position a Spacer's Style, accessibility
// props, callback IDs and margin were all in before earlier passes moved them.
// Those four were fixed and the children were named as still-open, on the
// grounds that the divergence was unreachable from Go: a Compose `Spacer` and
// a SwiftUI `Color.clear` are leaves, while both DOM renderers emit a Spacer's
// children like any other element's.
//
// Unreachable from core is not unreachable. htmlout exports any *core.Node a
// caller hands it, and the WASM runtime renders any tree the wire carries, so
// the observable behaviour was a subtree that appeared in a browser and
// vanished on a phone. Dropping content is the wrong side of a divergence to
// be on, and the DOM side additionally cannot move: the runtime's patches are
// addressed by data-node-path attributes written while walking node.Children,
// so an element it declines to emit takes every patch beneath it with it (the
// reason Fragment and Theme are boxed there — see tagForType).
//
// # Why the axis is checked and not just the presence
//
// A renderer that emits the children with a construct of its own choosing
// re-opens the divergence one level down: a Compose Box and a SwiftUI ZStack
// both draw two children on top of one another, where the DOM lays them out
// one after another. That is the exact failure TestNativeContainersStack...
// above records for Box and SafeArea. So the axis comes from one place —
// htmlout's stackAxes, restated in the WASM runtime and pinned to it by
// TestRuntimeStackAxesMatchGo — and each native is held to building the stack
// that table names.
//
// The two natives are read at different depths because they are written at
// different depths: Kotlin dispatches the whole node into GrMobColumn from the
// arm, while Swift's arm names a composite (GrMobSpacer) that has geometry of
// its own to keep — Color.clear is what makes a claimed axis fill, so the
// stack is laid over it rather than replacing it. Reading GrMobSpacer's body
// is what makes the Swift half a claim about the code that runs.
func TestNativeSpacerChildrenAreStackedNotDropped(t *testing.T) {
	if got := htmlout.StackAxisFor("Spacer"); got != "column" {
		t.Fatalf("htmlout stacks a Spacer along %q, so the column stacks pinned below "+
			"are answering to a table that no longer says column", got)
	}

	// Kotlin: the arm itself is the whole of it.
	arm := dispatchArm(t, kotlinRenderer, `"Spacer" ->`, kotlinArmStart)
	if !strings.Contains(arm, "GrMobColumn(") {
		t.Errorf("%s: the Spacer arm does not route to GrMobColumn — its children are "+
			"either dropped (a bare Compose Spacer is a leaf) or stacked by a second "+
			"implementation no alignment or gap check reaches", kotlinRenderer)
	}
	if strings.Contains(arm, "Box(") {
		t.Errorf("%s: the Spacer arm builds a Compose Box — its children would draw on "+
			"top of one another where both DOM targets stack them", kotlinRenderer)
	}

	// Swift: the composite the arm names.
	body := swiftTypeBody(t, swiftRenderer, "private struct GrMobSpacer")
	for _, want := range []struct{ expr, why string }{
		{"FlexChildren(node: node",
			"the node's children are never rendered, so a hand-assembled Spacer's " +
				"subtree is still dropped on this target alone"},
		{"GrMobFlexStack(axis: .vertical",
			"the children are laid out by something other than the column stack " +
				"htmlout's stackAxes names for this type"},
		{"Color.clear",
			"the sized base is gone — grMobBox paints its background inside its own " +
				"dimension frame, so an axis the Style claims is filled by the content " +
				"asking for it, and an empty flex stack asks for nothing"},
	} {
		if !strings.Contains(body, want.expr) {
			t.Errorf("%s: GrMobSpacer no longer contains %q — %s",
				swiftRenderer, want.expr, want.why)
		}
	}
	if strings.Contains(body, "ZStack(") {
		t.Errorf("%s: GrMobSpacer builds a ZStack — its children would draw on top of "+
			"one another where both DOM targets stack them", swiftRenderer)
	}
}
