package verify

import (
	"os"
	"strings"
	"testing"
)

// readNative reads a whole native source file. declSource's narrower cut is
// the right tool for a check aimed at one declaration; the parser checks
// below are about the file as a unit — a parse function is one long literal
// and the keys could legitimately move within it.
func readNative(t *testing.T, file string) string {
	t.Helper()
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("reading %s: %v", file, err)
	}
	return string(raw)
}

// The two style files, which no other check in this package reads yet.
var (
	swiftStyle  = nativeFile("ios", "GrMob", "Runtime", "GrMobStyle.swift")
	kotlinStyle = nativeFile("android", "app", "src", "main", "java", "com", "grmob",
		"runtime", "GrMobStyle.kt")
)

// core.RowGap and core.ColumnGap must reach the native style parsers.
//
// Both fields have existed in core.Style and been honored by both web targets
// since the style-parity pass, and neither native parser mentioned them: the
// props parsed cleanly out of Go, crossed the bridge in the JSON, and were
// dropped on the floor. A prop that is declared, documented and silently
// inert on half the targets is the failure this whole package exists to
// notice, and no compiler can — an unread JSON key is not a type error in
// either language.
//
// Source text rather than behavior, for the reason switchlabels_test.go
// gives: neither native runs under `go test ./...`.
func TestBothNativeParsersReadTheGapLonghands(t *testing.T) {
	for _, pin := range []struct {
		file string
		// keys are the JSON lookups the parser must perform. They are the Go
		// field names verbatim, since core.Style carries no json tags.
		keys []string
	}{
		{file: swiftStyle, keys: []string{`num("RowGap")`, `num("ColumnGap")`}},
		{file: kotlinStyle, keys: []string{`optDouble("RowGap"`, `optDouble("ColumnGap"`}},
	} {
		src := readNative(t, pin.file)
		for _, key := range pin.keys {
			if !strings.Contains(src, key) {
				t.Errorf("%s: never parses %s — core.RowGap/core.ColumnGap render on the web "+
					"and do nothing on this platform", pin.file, key)
			}
		}
	}
}

// Every container that stacks its children must space them by the gap of the
// axis it stacks along, and must ask for that axis by name.
//
// The direction is the whole point of the pin. `row-gap` spaces items
// *vertically* — it is the gap between rows — so a vertical stack reaching
// for `rowGap` is correct and a horizontal one reaching for it is not, and
// the two are one character apart. Both renderers therefore expose the pair
// as verticalGap/horizontalGap, named for the axis rather than for the CSS
// property, and every container reads through them. This holds each
// container to the accessor for its own axis; a container that went back to
// the isotropic `gap` would still compile and would still look right in every
// app that sets Gap alone, which is why the check is on the source and not on
// a rendered tree.
func TestNativeContainersSpaceAlongTheirOwnAxis(t *testing.T) {
	for _, pin := range []struct {
		file, decl string
		// want is the accessor this container must read; reject is the one
		// that would be the wrong axis (empty when both axes are in play).
		want, reject string
	}{
		// iOS. The flex stack answers for Row and Column at once, so it picks
		// its accessor off its own axis and must mention both.
		{file: swiftRenderer, decl: "struct GrMobFlexStack", want: "verticalGap"},
		{file: swiftRenderer, decl: "struct GrMobFlexStack", want: "horizontalGap"},
		{file: swiftRenderer, decl: "private struct GrMobList", want: "verticalGap", reject: "horizontalGap"},
		{file: swiftRenderer, decl: "private struct GrMobScroll", want: "verticalGap", reject: "horizontalGap"},
		// The wrapping Row spaces both axes: chips along a line, lines apart.
		{file: swiftRenderer, decl: "private struct GrMobRow", want: "horizontalGap"},
		{file: swiftRenderer, decl: "private struct GrMobRow", want: "verticalGap"},

		// Android. The two packed* helpers are where Gap survives at all (see
		// the divergence note above horizontalArrangement), so they are the
		// containers' single point of contact with the spacing.
		{file: kotlinRenderer, decl: "private fun packedHorizontally(", want: "horizontalGap", reject: "verticalGap"},
		{file: kotlinRenderer, decl: "private fun packedVertically(", want: "verticalGap", reject: "horizontalGap"},
	} {
		src := declSource(t, pin.file, pin.decl)
		if !strings.Contains(src, pin.want) {
			t.Errorf("%s: %s does not read %s — it spaces its children along the wrong axis, "+
				"or ignores the gap longhands entirely", pin.file, pin.decl, pin.want)
		}
		if pin.reject != "" && strings.Contains(src, pin.reject) {
			t.Errorf("%s: %s reads %s, which is the other axis's spacing",
				pin.file, pin.decl, pin.reject)
		}
	}
}

// A Scroll is a flex column on both web targets — the WASM runtime lists it
// in STACK_CONTAINERS and htmlout's styleValue emits gap for it — so
// core.Gap on a Scroll spaced its children in the browser. Both natives
// built the scrolling stack with the spacing hard-coded to nothing, so the
// same declaration drew flush on a phone.
//
// The two spellings below are what "hard-coded to nothing" looks like in each
// language. Neither is a construct that would ever be written deliberately
// once the container reads a gap, so their absence is a usable pin even
// though it is a negative one; the positive half is the axis check above.
func TestNativeScrollDoesNotHardCodeZeroSpacing(t *testing.T) {
	for _, pin := range []struct{ file, decl, zero string }{
		{swiftRenderer, "private struct GrMobScroll", "spacing: 0"},
		// Compose spells "no arrangement" by omitting the argument, so the
		// pin is that the argument is present at all.
		{kotlinRenderer, "private fun GrMobScroll(", ""},
	} {
		src := declSource(t, pin.file, pin.decl)
		if pin.zero != "" && strings.Contains(src, pin.zero) {
			t.Errorf("%s: %s stacks with %q — core.Gap on a Scroll is dropped on this platform",
				pin.file, pin.decl, pin.zero)
		}
		if pin.zero == "" && !strings.Contains(src, "verticalArrangement") {
			t.Errorf("%s: %s passes no verticalArrangement — its Column falls back to "+
				"Arrangement.Top and core.Gap on a Scroll is dropped", pin.file, pin.decl)
		}
	}
}
