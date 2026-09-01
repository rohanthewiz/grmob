package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Shared machinery for the tests that pin the WASM runtime's lookup tables to
// their Go authorities (inputtype_test.go, tagtype_test.go).
//
// Two tables live twice by necessity — once in Go, once in JavaScript, because
// the runtime builds elements in the browser and cannot call into Go to ask.
// Two copies of one rule drift silently, which is exactly how the Checkbox gap
// survived, so these tests read the runtime's copies out of its source and
// compare them with Go's.
//
// They parse rather than execute. A Go test that needed Node to run would stop
// running for anyone who has only Go, and both tables are flat object literals
// with no computation in them, so text is enough. That is the whole point:
// `go test ./...` reaches these checks, unlike run.sh, which a human has to
// remember.

// The shape all three runtime lookups are written in, and which the parse
// below requires:
//
//	function <name>(<param>) {
//	    return {
//	        Key: "value",
//	        ...
//	    }[<param>] || "<fallback>";
//	}
//
// The subscript is checked against the function's own parameter name, read off
// the signature, rather than against a fixed "type" — objectFitFor calls its
// argument "mode", and forcing all three to agree on one name would be making
// the runtime read worse to suit its test. Checking it at all is what proves
// the braces the parse found are the lookup table and not some later object
// literal in the same function.
//
// This is a constraint on how those functions may be rewritten, and it is
// written down in grmob-runtime.js beside each of them. Every violation of it
// is a named fatal below rather than a short map, so a rewrite lands as a
// failure that says what happened.
var jsPair = regexp.MustCompile(`(\w+)\s*:\s*"([^"]*)"`)

// runtimeSource reads the real runtime the browser loads, from the path
// load.mjs uses. A missing or unreadable file fails loudly rather than
// yielding an empty string that every assertion would then "pass" against.
func runtimeSource(t *testing.T) string {
	t.Helper()
	path := filepath.Join("..", "grmob-runtime.js")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(src)
}

// parseRuntimeTable lifts the object literal out of the named runtime lookup
// and returns it as a map, having first proved that it found the right braces:
// the literal must be followed by the `[type] || "<fallback>"` that makes it a
// lookup, with the fallback the caller expects.
//
// Every step fails the test rather than returning a short map, for the reason
// the replay harness grew a control mutant: a check that reads nothing must not
// be able to read as a pass. A rename or a rewrite of one of these functions is
// meant to land here as a failure that names it, not as an empty comparison
// that agrees with everything.
//
// Folding the fallback check in here rather than leaving it to a separate test
// is what makes the parse total: the fallback carries the half of each contract
// that lives outside the table — "" is what leaves a <textarea> with no type
// attribute at all, "div" is what an unknown node type becomes — and a table
// comparison alone would pass a runtime whose default had drifted.
func parseRuntimeTable(t *testing.T, src, funcName, fallback string) map[string]string {
	t.Helper()

	start := regexp.MustCompile(`function\s+` + regexp.QuoteMeta(funcName) + `\s*\(\s*(\w+)\s*\)\s*\{`)
	loc := start.FindStringSubmatchIndex(src)
	if loc == nil {
		t.Fatalf("grmob-runtime.js: no %s function taking a single named argument found — "+
			"if it was renamed or its signature changed, update this test", funcName)
	}
	param := src[loc[2]:loc[3]] // the argument the lookup must be subscripted by
	body := src[loc[1]:]

	open := strings.Index(body, "{")
	if open < 0 {
		t.Fatalf("grmob-runtime.js: %s has no object literal", funcName)
	}
	// The literals are flat (no nested braces), so the first closing brace ends
	// the table. The lookup that must follow is what proves this brace pair is
	// the table and not some later construct.
	end := strings.Index(body[open:], "}")
	if end < 0 {
		t.Fatalf("grmob-runtime.js: %s's object literal is unterminated", funcName)
	}
	end += open

	wantTail := "[" + param + `] || "` + fallback + `";`
	if got := strings.TrimSpace(body[end+1:]); !strings.HasPrefix(got, wantTail) {
		t.Fatalf("grmob-runtime.js: %s's table is not followed by %s\n  found: %.40s...",
			funcName, wantTail, got)
	}

	table := map[string]string{}
	for _, m := range jsPair.FindAllStringSubmatch(body[open:end], -1) {
		table[m[1]] = m[2]
	}
	if len(table) == 0 {
		t.Fatalf("grmob-runtime.js: %s's table parsed as empty", funcName)
	}
	return table
}
