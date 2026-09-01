package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/htmlout"
)

// The node type -> <input> type table exists twice by necessity: once in Go
// (htmlout.InputTypeFor, the authority) and once in JavaScript, because the
// WASM runtime sets the attribute in the browser and cannot call into Go to
// ask. Two copies of one rule drift silently — that is exactly how the
// Checkbox gap survived — so this test reads the runtime's copy out of its
// source and compares it against the Go one.
//
// It parses rather than executes: a Go test that needed Node to run would
// stop running for anyone who has only Go, and this table is a flat object
// literal with no computation in it, so text is enough. The whole point is
// that `go test ./...` reaches this check, unlike run.sh, which a human has
// to remember.

// The runtime's lookup, which the parse has to find intact:
//
//	function inputTypeFor(type) {
//	    return {
//	        Input: "text",
//	        ...
//	    }[type] || "";
//	}
var (
	jsFuncStart = regexp.MustCompile(`function\s+inputTypeFor\s*\([^)]*\)\s*\{`)
	jsPair      = regexp.MustCompile(`(\w+)\s*:\s*"([^"]*)"`)
)

func TestRuntimeInputTypesMatchGo(t *testing.T) {
	table := parseRuntimeInputTypes(t, runtimeSource(t))

	want := htmlout.InputTypes()
	for nodeType, jsType := range table {
		if goType := want[nodeType]; goType != jsType {
			t.Errorf("%s: runtime says %q, htmlout says %q", nodeType, jsType, goType)
		}
		delete(want, nodeType)
	}
	// Anything left is a type Go maps onto an <input> and the runtime does
	// not, which is the direction that silently draws the wrong control.
	for nodeType, goType := range want {
		t.Errorf("%s: htmlout says %q, runtime has no entry", nodeType, goType)
	}
}

// The unknown-type answer is half the contract and lives in the fallback
// rather than the table, so it is checked separately: htmlout's lookup yields
// "" for a node with a tag of its own (a <textarea>, a <span>), and the
// runtime must agree, since createElement only sets the attribute when the
// lookup is truthy.
func TestRuntimeInputTypeFallbackIsEmpty(t *testing.T) {
	if got := htmlout.InputTypeFor("TextArea"); got != "" {
		t.Fatalf("htmlout.InputTypeFor(TextArea) = %q, want the empty default", got)
	}
	if !strings.Contains(runtimeSource(t), `}[type] || "";`) {
		t.Error(`grmob-runtime.js: inputTypeFor no longer falls back to "" for an unlisted node type`)
	}
}

// runtimeSource reads the real runtime the browser loads, from the path
// load.mjs uses. A missing or unreadable file fails loudly rather than
// yielding an empty string that every assertion below would then "pass".
func runtimeSource(t *testing.T) string {
	t.Helper()
	path := filepath.Join("..", "grmob-runtime.js")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(src)
}

// parseRuntimeInputTypes lifts the object literal out of inputTypeFor.
//
// Every step fails the test rather than returning a short map, for the reason
// the replay harness grew a control mutant: a check that reads nothing must
// not be able to read as a pass. A rename or a rewrite of that function is
// meant to land here as a failure, not as an empty comparison.
func parseRuntimeInputTypes(t *testing.T, src string) map[string]string {
	t.Helper()

	loc := jsFuncStart.FindStringIndex(src)
	if loc == nil {
		t.Fatal("grmob-runtime.js: no inputTypeFor function found — if it was renamed, update this test")
	}
	body := src[loc[1]:]

	open := strings.Index(body, "{")
	if open < 0 {
		t.Fatal("grmob-runtime.js: inputTypeFor has no object literal")
	}
	// The literal is flat (no nested braces), so the first closing brace ends
	// it. The [type] that must follow is what proves this brace pair is the
	// lookup table and not some later construct.
	end := strings.Index(body[open:], "}")
	if end < 0 {
		t.Fatal("grmob-runtime.js: inputTypeFor's object literal is unterminated")
	}
	end += open
	if !strings.HasPrefix(strings.TrimSpace(body[end+1:]), "[type]") {
		t.Fatal("grmob-runtime.js: the braces after inputTypeFor are not the [type] lookup table")
	}

	table := map[string]string{}
	for _, m := range jsPair.FindAllStringSubmatch(body[open:end], -1) {
		table[m[1]] = m[2]
	}
	if len(table) == 0 {
		t.Fatal("grmob-runtime.js: inputTypeFor's table parsed as empty")
	}
	return table
}
