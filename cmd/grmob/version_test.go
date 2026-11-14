package main

import (
	"strings"
	"testing"
)

// `grmob version` always names the command and a Go version, and never
// prints the empty string where a version belongs.
func TestVersionStringNamesTheBuild(t *testing.T) {
	v := versionString()
	if !strings.HasPrefix(v, "grmob ") || !strings.Contains(v, "go1.") {
		t.Fatalf("versionString() = %q, want \"grmob <version> go1.x\"", v)
	}
	if strings.Contains(v, "grmob  ") {
		t.Fatalf("versionString() = %q has an empty version", v)
	}
}
