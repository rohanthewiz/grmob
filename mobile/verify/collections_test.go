package verify

import (
	"strings"
	"testing"
)

// Roadmap tier B — horizontal scroll (B1), end-reached (B2) and sticky
// headers (B3) — held against both native renderers.
//
// These three are the package's typical subject: a Go prop crosses the bridge
// as a style field or a props entry, both web targets already honour it (two
// of the three needed no web code at all), and the only thing standing
// between "implemented" and "silently inert on device" is whether each native
// renderer bothers to read the key. Nothing compiles differently when it does
// not — an unread JSON key is not a type error in Kotlin or in Swift — and
// neither shell runs under `go test ./...`, so the check is on the source
// text, as everywhere else here.
//
// Each pin names the platform primitive as well as the read. Reading the key
// and doing nothing with it is the same outcome as not reading it, and the
// primitives are what a reviewer would look for anyway: horizontalScroll and
// ScrollView(.horizontal), stickyHeader and pinnedViews.

// The style fields the three props travel in. Two of them are ordinary CSS
// properties that the natives had no reason to parse until now, which is
// exactly the shape of the gap TestBothNativeParsersReadTheGapLonghands
// caught for RowGap/ColumnGap.
func TestBothNativeParsersReadTheCollectionStyleFields(t *testing.T) {
	for _, pin := range []struct {
		file string
		// keys are the JSON lookups the parser must perform. They are the Go
		// field names verbatim, since core.Style carries no json tags.
		keys []string
	}{
		{file: swiftStyle, keys: []string{`str("FlexDirection")`, `str("Position")`}},
		{file: kotlinStyle, keys: []string{`optString("FlexDirection")`, `optString("Position")`}},
	} {
		src := readNative(t, pin.file)
		for _, key := range pin.keys {
			if !strings.Contains(src, key) {
				t.Errorf("%s: never parses %s — core.Horizontal and core.StickyHeader render "+
					"on both web targets and do nothing on this platform", pin.file, key)
			}
		}
	}
}

// B1. A Scroll carrying core.Horizontal arrives as Style.FlexDirection ==
// "row", and each native has exactly one way to spell a sideways scrolling
// region. The pin is the pair: the test that picks the branch, and the
// primitive the branch is picked for.
func TestNativeScrollHonoursTheHorizontalAxis(t *testing.T) {
	for _, pin := range []struct {
		file, decl string
		reads      string // the branch test
		primitive  string // the platform's sideways scroller
	}{
		{swiftRenderer, "private struct GrMobScroll",
			`flexDirection == "row"`, "ScrollView(.horizontal"},
		{kotlinRenderer, "private fun GrMobScroll(",
			`flexDirection == "row"`, "horizontalScroll("},
	} {
		src := declSource(t, pin.file, pin.decl)
		if !strings.Contains(src, pin.reads) {
			t.Errorf("%s: %s never tests %s — core.Horizontal() pans a strip in the browser "+
				"and stacks it down the screen here", pin.file, pin.decl, pin.reads)
		}
		if !strings.Contains(src, pin.primitive) {
			t.Errorf("%s: %s reads the axis but never builds a %s — the branch exists and "+
				"scrolls the wrong way", pin.file, pin.decl, pin.primitive)
		}
	}
}

// B3. A List child carrying core.StickyHeader arrives as Style.Position ==
// "sticky", and each native pins it with its lazy container's own mechanism:
// Compose has a stickyHeader item, SwiftUI has pinned section headers. The
// two are structurally different — one is a per-item call, the other a
// property of the stack plus a regrouping of its children — so the pin names
// each platform's piece rather than looking for one shared word.
func TestNativeListPinsStickyHeaders(t *testing.T) {
	for _, pin := range []struct {
		file, decl string
		// wants are read together: the marker test, then the primitive.
		wants []string
	}{
		{swiftRenderer, "private struct GrMobList",
			[]string{"pinnedViews:", ".sectionHeaders", "Section {"}},
		{swiftRenderer, "func listSections(",
			[]string{`position == "sticky"`}},
		{kotlinRenderer, "private fun GrMobList(",
			[]string{"isStickyHeader(row)", "stickyHeader("}},
		{kotlinRenderer, "fun isStickyHeader(",
			[]string{`position == "sticky"`}},
	} {
		src := declSource(t, pin.file, pin.decl)
		for _, want := range pin.wants {
			if !strings.Contains(src, want) {
				t.Errorf("%s: %s does not contain %q — core.StickyHeader pins a group band in "+
					"the browser and scrolls away with its rows here",
					pin.file, pin.decl, want)
			}
		}
	}
}

// B2. core.OnEndReached crosses as a props entry, not a style field, so each
// native reads it with stringProp and dispatches the ID it finds.
//
// How the edge is detected is deliberately not pinned: Compose watches the
// lazy list's last visible index, SwiftUI rides the last row's .onAppear, and
// the two are as different as their platforms. What must hold is that the
// prop is read at all and that firing it reaches Go.
func TestNativeListReportsTheEndReachedEdge(t *testing.T) {
	for _, pin := range []struct {
		file, decl string
		dispatch   string
	}{
		// Anchored on the row builder rather than on the struct: declSource
		// cuts a declaration at the next function, and GrMobList's own
		// methods are functions, so the struct's range ends before rowView.
		{swiftRenderer, "func rowView(", "dispatch?(endReached)"},
		{kotlinRenderer, "fun EndReachedReporter(", "runtime.click(callbackId)"},
	} {
		src := declSource(t, pin.file, pin.decl)
		if !strings.Contains(src, `stringProp("onEndReached")`) {
			t.Errorf("%s: %s never reads onEndReached — an infinite feed loads its first page "+
				"and then stops on this platform", pin.file, pin.decl)
			continue
		}
		if !strings.Contains(src, pin.dispatch) {
			t.Errorf("%s: %s reads onEndReached but never dispatches it (%s) — the prop is "+
				"parsed and dropped", pin.file, pin.decl, pin.dispatch)
		}
	}
}
