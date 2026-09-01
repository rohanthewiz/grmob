package verify

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// Both TreeStores must refuse a payload that is not the shape they are about
// to parse, rather than throwing out of the delivery thread.
//
// The hazard is not hypothetical and does not come from the native side at
// all: render.renderJSON returns the string {"error":"failed to encode JSON"}
// whenever a payload will not marshal, and a single NaN or +Inf reaching a
// node's Props is enough to produce one (encoding/json rejects them). That
// string then travels the ordinary delivery path. Kotlin's JSONArray(json)
// threw on it, from inside the main-thread Handler, which crashes the app and
// buries the encode failure that caused it; Swift logged and returned.
//
// Read as source text for the same reason as everything else in this package
// — see doc.go. What is checked is narrow on purpose: that the parse is
// preceded by a shape test on the raw string. It cannot prove the guard is
// correct, only that one is there, which is exactly the thing that was
// missing on one side and present on the other.

var (
	kotlinTreeStore = nativeFile("android", "app", "src", "main", "java", "com", "grmob",
		"runtime", "TreeStore.kt")
	swiftTreeStore = nativeFile("ios", "GrMob", "Runtime", "TreeStore.swift")
)

// funcBody returns the source from anchor up to the first line that is
// indented no further than anchor's own line — i.e. the end of the member.
// Crude, and adequate: both files are conventionally indented, and a miss
// shows up as a fatal below rather than as a silent pass.
func funcBody(t *testing.T, file, anchor string) string {
	t.Helper()
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("reading %s: %v", file, err)
	}
	src := string(raw)
	at := strings.Index(src, anchor)
	if at < 0 {
		t.Fatalf("%s: no %q found — if it was renamed or restructured, update this test", file, anchor)
	}
	rest := src[at:]
	// The member ends at the first closing brace sitting at the anchor's own
	// indentation, which for a class member is four spaces.
	if end := regexp.MustCompile(`(?m)^ {4}\}`).FindStringIndex(rest); end != nil {
		rest = rest[:end[1]]
	}
	return rest
}

func TestPatchPayloadIsShapeCheckedBeforeParsing(t *testing.T) {
	cases := []struct {
		name   string
		file   string
		anchor string
		// guard matches the construct that must open *before* the parse, so
		// the parse's result is tested rather than trusted. The two languages
		// spell that differently: Kotlin tests the raw string and returns
		// early, Swift folds the parse into a `guard ... else` whose cast is
		// the test, so the cast itself is listed under `also` instead.
		guard *regexp.Regexp
		// also lists tokens that must appear somewhere in the body.
		also []*regexp.Regexp
		// parse matches the call that would throw on a bad payload.
		parse *regexp.Regexp
	}{
		{
			name:   "Kotlin applyPatches",
			file:   kotlinTreeStore,
			anchor: "fun applyPatches(",
			guard:  regexp.MustCompile(`startsWith\("\["\)`),
			also:   []*regexp.Regexp{regexp.MustCompile(`\breturn\b`)},
			parse:  regexp.MustCompile(`JSONArray\(json\)`),
		},
		{
			name:   "Swift applyPatches",
			file:   swiftTreeStore,
			anchor: "func applyPatches(",
			guard:  regexp.MustCompile(`\bguard\b`),
			also: []*regexp.Regexp{
				regexp.MustCompile(`as\?\s*\[Any\]`),
				regexp.MustCompile(`\breturn\b`),
			},
			parse: regexp.MustCompile(`JSONSerialization\.jsonObject`),
		},
		// The mount side of both, which has always been guarded — included so
		// the pair is checked as a pair and a future rewrite cannot quietly
		// drop the half that was already right.
		{
			name:   "Kotlin mount",
			file:   kotlinTreeStore,
			anchor: "fun mount(",
			guard:  regexp.MustCompile(`startsWith\("\{"\)`),
			also:   []*regexp.Regexp{regexp.MustCompile(`\breturn\b`)},
			parse:  regexp.MustCompile(`JSONObject\(json\)`),
		},
		{
			name:   "Swift mount",
			file:   swiftTreeStore,
			anchor: "func mount(",
			guard:  regexp.MustCompile(`hasPrefix\("\{"\)`),
			also:   []*regexp.Regexp{regexp.MustCompile(`\breturn\b`)},
			parse:  regexp.MustCompile(`JSONSerialization\.jsonObject`),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			body := funcBody(t, c.file, c.anchor)

			parseAt := c.parse.FindStringIndex(body)
			if parseAt == nil {
				t.Fatalf("%s: no %v found in %s — update this test if the parse moved",
					c.file, c.parse, c.anchor)
			}
			guardAt := c.guard.FindStringIndex(body)
			if guardAt == nil {
				t.Fatalf("%s: %s parses the payload with no shape check first.\n"+
					"A Go-side encode failure arrives here as the string "+
					`{"error":"failed to encode JSON"}, and an unguarded parse `+
					"throws out of the delivery thread instead of logging it.",
					c.file, c.anchor)
			}
			if guardAt[0] > parseAt[0] {
				t.Errorf("%s: the shape check in %s comes after the parse, so it never runs",
					c.file, c.anchor)
			}
			for _, re := range c.also {
				if !re.MatchString(body) {
					t.Errorf("%s: %s has no %v — the guard cannot bail out on a bad payload",
						c.file, c.anchor, re)
				}
			}
		})
	}
}
