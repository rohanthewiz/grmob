package verify

import (
	"regexp"
	"strings"
	"testing"
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

// dispatchArm returns the *code* of one arm of a renderer's node-type
// dispatch: from its marker up to the next arm, with line comments removed.
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
		t.Fatalf("%s: no %s arm — if the dispatch was restructured, update this test", file, marker)
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
