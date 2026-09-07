package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// browser.mjs's sticky fixture, held to core.StickyHeader().
//
// # What was wrong with writing them out
//
// browser.mjs is the only pass in this repository that can say whether a
// pinned band actually pins: dom.mjs has no layout, so every other suite can
// assert that three declarations landed on an element and nothing more. The
// fixture it mounts is a JSON tree, which means it cannot call
// core.StickyHeader() — it has to state the declarations itself, the same
// position palette.mjs is in with the bundled hexes.
//
// palette.mjs is pinned (palette_test.go rebuilds it from core.BundledThemes()
// and internal/palette); this fixture was not. What that costs is specific:
// core.StickyHeader() supplies three declarations, and if it grew a fourth —
// or if one of the three changed spelling — the browser pass would go on
// pinning a band with the old set and reporting a pass. The check would still
// be *a* true statement about position:sticky in a box; it would have stopped
// being a statement about the framework's prop.
//
// # Why the whole set and not the three values
//
// The comparison is between the Style fields core.StickyHeader() *touches* and
// the keys of STICKY_DECLARATIONS, not between three names somebody typed on
// each side. A value check alone passes a prop that has learned to write a
// fourth declaration, which is exactly the change that would matter: the
// fixture would be mounting a band the framework no longer builds, and the
// browser would be answering a question about last year's pin.
//
// It is derived by applying the prop to an empty Style and diffing the JSON
// against a zero one. core.Style marshals every field, including the zero
// ones, so the diff is total — a field set to its own zero value would be
// invisible, and there is no such declaration here (a Top of "" never sticks
// and a ZIndex of 0 is painted over, which is why StickyHeader supplies both).

// jsPair is one `Key: value` of a runtime object literal, where the value is a
// quoted string or a bare number. Shared with nothing: jstable_test.go's
// version reads string values only, and ZIndex is a number.
var stickyPair = regexp.MustCompile(`(\w+)\s*:\s*(?:"([^"]*)"|(-?\d+(?:\.\d+)?))`)

// stickyDeclarations is what core.StickyHeader() writes, as field name → value
// in the spelling a mounted JSON tree uses.
//
// Read through encoding/json rather than by reflecting over the struct,
// because the JSON field names are what browser.mjs's tree actually carries:
// the runtime is handed this object and reads `Position`, not `position`. A
// json tag added to core.Style would move both sides together.
func stickyDeclarations(t *testing.T) map[string]string {
	t.Helper()

	var got core.Style
	core.StickyHeader().Apply(&got)

	asMap := func(s core.Style) map[string]any {
		raw, err := json.Marshal(s)
		if err != nil {
			t.Fatalf("marshalling a core.Style: %v", err)
		}
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Fatalf("unmarshalling a core.Style: %v", err)
		}
		return m
	}

	zero := asMap(core.Style{})
	out := map[string]string{}
	for name, v := range asMap(got) {
		if fmt.Sprint(v) != fmt.Sprint(zero[name]) {
			out[name] = jsonScalar(v)
		}
	}
	if len(out) == 0 {
		t.Fatal("core.StickyHeader() applied to an empty Style changed nothing — " +
			"either the prop has been emptied or Apply is not what sets these fields, " +
			"and a comparison against nothing would agree with any fixture")
	}
	return out
}

// jsonScalar renders a decoded JSON value the way the .mjs literal spells it,
// so the two sides compare as text: a number that came back as float64(1) is
// "1" rather than "1e+00", and a string is its own contents.
func jsonScalar(v any) string {
	if f, ok := v.(float64); ok {
		return strings.TrimSuffix(fmt.Sprintf("%g", f), ".0")
	}
	return fmt.Sprint(v)
}

// browserFixtureObject lifts a named `const NAME = { … };` object literal out
// of browser.mjs.
//
// Fails rather than returning an empty map for the reason parseRuntimeTable
// does: a rename would otherwise turn this pin into a comparison between two
// empty sets, which passes.
func browserFixtureObject(t *testing.T, name string) map[string]string {
	t.Helper()

	src := browserSource(t)
	re := regexp.MustCompile(`const ` + name + `\s*=\s*\{([^}]*)\}`)
	m := re.FindStringSubmatch(src)
	if m == nil {
		t.Fatalf("no `const %s = { … }` in browser.mjs — the fixture this test pins "+
			"has been renamed or rewritten, and a check that found nothing must not "+
			"read as a pass", name)
	}
	out := map[string]string{}
	for _, pair := range stickyPair.FindAllStringSubmatch(m[1], -1) {
		value := pair[2]
		if pair[3] != "" {
			value = pair[3]
		}
		out[pair[1]] = value
	}
	if len(out) == 0 {
		t.Fatalf("`const %s` in browser.mjs holds no `Key: value` pairs this test can "+
			"read", name)
	}
	return out
}

// browserSource reads the browser pass, from the path run.sh runs. A missing
// or unreadable file is a failure rather than an empty string, for the reason
// runtimeSource gives: every assertion below would then "pass" against nothing.
func browserSource(t *testing.T) string {
	t.Helper()
	src, err := os.ReadFile(filepath.Join(".", "browser.mjs"))
	if err != nil {
		t.Fatalf("reading browser.mjs: %v", err)
	}
	return string(src)
}

// The fixture declares exactly what the prop writes.
func TestTheBrowserStickyFixtureIsCoreStickyHeader(t *testing.T) {
	want := stickyDeclarations(t)
	got := browserFixtureObject(t, "STICKY_DECLARATIONS")

	if !sameDeclarations(got, want) {
		t.Errorf("browser.mjs mounts %s and core.StickyHeader() writes %s.\n"+
			"The browser pass is the only thing here that can tell a pinned band from "+
			"a band with the properties written on it, and it is pinning a set the "+
			"framework no longer produces — update STICKY_DECLARATIONS, and check the "+
			"prose in core/list.go and docs/platforms/wasm.md that names these three",
			declString(got), declString(want))
	}

	// And the fixture spreads that one statement rather than restating it. A
	// second copy of `Position: "sticky"` inside the band's own Style would
	// satisfy the comparison above while the pin the browser actually mounts
	// went unchecked.
	src := browserSource(t)
	if !strings.Contains(src, "...STICKY_DECLARATIONS") {
		t.Error("browser.mjs no longer spreads STICKY_DECLARATIONS into the band's " +
			"Style — the constant this test pins is not the one being mounted")
	}
}

// sameDeclarations compares two field sets by name and by value.
func sameDeclarations(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// declString renders a field set in a stable order, so a failure prints the
// same two lines every run.
func declString(m map[string]string) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = fmt.Sprintf("%s: %q", k, m[k])
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}
