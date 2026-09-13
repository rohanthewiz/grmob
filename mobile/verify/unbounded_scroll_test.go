package verify

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// A vertical scroll on Compose may be measured under an infinite height.
//
// Compose's ScrollingLayoutNode throws when its incoming maximum height is
// infinite ("Vertically scrollable component was measured with an infinity
// maximum height constraints"), and a vertical scroll inside another vertical
// scroll is exactly that. The DOM and SwiftUI lay the inner region out at its
// content height and carry on, so the Go tree is legal and renders on three
// hosts — and killed the Android app on the first layout of every tutorial
// lesson, because every code block is a Height-less comps.CodeEditor inside
// comps.Screen{Scroll: true}.
//
// Renderer.kt's verticalScrollWhenBounded is the one spelling that caps the
// viewport at the content when the height is unbounded. The check is a
// refusal: the runtime may call Compose's verticalScroll in exactly one place,
// the helper's own body, so the next scroll region added to the renderer
// cannot reintroduce the crash by reaching for the obvious modifier.
//
// Nothing here compiles Compose, so the helper's *behaviour* is held by a
// device pass (the tutorial's lessons open on an emulator); what is pinned is
// that the behaviour lives in one place and every caller goes through it.
func TestNoBareVerticalScrollOnCompose(t *testing.T) {
	const helper = "internal fun Modifier.verticalScrollWhenBounded("

	// A call, not a mention: the identifier followed by "(", and not the tail
	// of a longer name. codeIn blanks comments and literals, so a note that
	// explains why a bare call is refused does not count as one.
	bare := regexp.MustCompile(`(^|[^A-Za-z0-9_])verticalScroll\(`)

	dir := filepath.Dir(kotlinRenderer)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}

	total := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".kt") {
			continue
		}
		file := filepath.Join(dir, e.Name())
		n := len(bare.FindAllStringIndex(codeIn(t, file), -1))
		total += n
		if n > 0 && file != kotlinRenderer {
			t.Errorf("%s: %d bare verticalScroll call(s) — use verticalScrollWhenBounded, "+
				"or this region crashes the app whenever its parent scrolls too", file, n)
		}
	}

	// The helper's own call is the one allowed. Exactly one in the renderer
	// means nothing else there reached for it either.
	if total != 1 {
		t.Errorf("found %d bare verticalScroll calls in the Compose runtime, want 1 "+
			"(inside verticalScrollWhenBounded)", total)
	}

	body := codeOf(t, kotlinRenderer, helper)
	for _, pin := range []struct{ expr, why string }{
		{"constraints.hasBoundedHeight", "the branch: a bounded height is a real viewport"},
		{"maxIntrinsicHeight(", "the unbounded branch's content height, which intrinsics answer without the check"},
		{".verticalScroll(state)", "the bare call lives here and nowhere else"},
	} {
		if !strings.Contains(body, pin.expr) {
			t.Errorf("%s: verticalScrollWhenBounded has no %q — %s", kotlinRenderer, pin.expr, pin.why)
		}
	}

	// And the two callers that crashed, or would have: the editor every
	// lesson's code block is, and a Scroll nested in a Scroll.
	for _, c := range []struct{ file, decl string }{
		{kotlinCodeEditor, "internal fun GrMobCodeEditor("},
		{kotlinRenderer, "private fun GrMobScroll("},
	} {
		if !strings.Contains(codeOf(t, c.file, c.decl), "verticalScrollWhenBounded(") {
			t.Errorf("%s: %s does not scroll through verticalScrollWhenBounded", c.file, c.decl)
		}
	}
}

// The lazy half of the same crash. LazyColumn is a scroll container and throws
// under an infinite maximum height exactly as verticalScroll does, and a List
// with no Height inside a scrolled page is that — the tutorial's 4.3 outline
// and 4.6 GroupedLists crashed on it once every code block had stopped
// crashing first.
//
// No intrinsic trick is available for this one (a lazy list cannot answer
// intrinsics), so GrMobList asks BoxWithConstraints and takes a plain Column of
// every row when the height is unbounded — the picture the DOM draws for a List
// with no overflow of its own. Pinned: the branch is on the constraint, the
// lazy list is in the bounded arm, the other arm is not lazy, and no other lazy
// list exists in the runtime to have skipped the question.
func TestComposeListHasAnArmForAnUnboundedHeight(t *testing.T) {
	list := codeOf(t, kotlinRenderer, "private fun GrMobList(")
	for _, pin := range []struct{ expr, why string }{
		{"BoxWithConstraints(", "how composition learns the incoming height"},
		{"propagateMinConstraints = true", "so a bounded List measures exactly as the bare LazyColumn did"},
		{"constraints.hasBoundedHeight", "the branch"},
		{"GrMobListAtContentHeight(", "the unbounded arm"},
		{"LazyColumn(", "the bounded arm"},
	} {
		if !strings.Contains(list, pin.expr) {
			t.Errorf("%s: GrMobList has no %q — %s", kotlinRenderer, pin.expr, pin.why)
		}
	}
	if i, j := strings.Index(list, "hasBoundedHeight"), strings.Index(list, "LazyColumn("); i < 0 || j < i {
		t.Errorf("%s: GrMobList builds its LazyColumn before testing the height — the lazy list "+
			"has to be in the bounded arm, or a List in a scrolled page still throws", kotlinRenderer)
	}

	arm := codeOf(t, kotlinRenderer, "private fun GrMobListAtContentHeight(")
	if strings.Contains(arm, "Lazy") || !strings.Contains(arm, "Column(") {
		t.Errorf("%s: GrMobListAtContentHeight must lay its rows out in a plain Column; "+
			"a lazy container here is the crash it exists to avoid", kotlinRenderer)
	}

	// "(" or "{": Kotlin's trailing-lambda form calls LazyColumn { ... } with
	// no parentheses at all, and that spelling crashes the same way.
	lazy := regexp.MustCompile(`(^|[^A-Za-z0-9_])Lazy(Column|Row|VerticalGrid|HorizontalGrid)\s*[({]`)
	dir := filepath.Dir(kotlinRenderer)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	total := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".kt") {
			continue
		}
		total += len(lazy.FindAllStringIndex(codeIn(t, filepath.Join(dir, e.Name())), -1))
	}
	if total != 1 {
		t.Errorf("found %d lazy containers in the Compose runtime, want 1 (GrMobList's bounded "+
			"arm) — a new one needs the same unbounded-height arm", total)
	}
}
