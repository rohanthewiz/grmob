package gateharness

import (
	"os/exec"
	"strings"
	"testing"
)

// harness_test.sh, run where anyone with a Go toolchain runs it.
//
// # Why this exists when both run.sh already run it
//
// harness.sh is the shared shape of two gate tests, and harness_test.sh reaches
// its arms — the counting, the prefix match, gate_distinct, the footer — which
// only ever execute when a gate is WRONG. Both ios/verify/run.sh and
// android/verify/run.sh run it before their own gate, which is the right place
// for it: the file is checked wherever it is used.
//
// It is also the only place, and that is the gap. Those two scripts need a Mac
// with the Command Line Tools or a machine with a JDK and a gradle cache; a
// contributor with neither runs `go test ./...`, sees no failure, and has no
// way to learn that the helper both passes count with had stopped counting.
// wasm/verify/checknumbering_test.go is the precedent for the other answer — a
// check about a file no Go code imports, kept inside `go test` so that it runs
// for anyone with a Go toolchain and none of the machinery.
//
// # What this does not become
//
// A second copy of the assertions. The subject is the shell script and the
// script is the assertions; this runs it and reports what it printed. A Go
// transcription of the same arms would be the exact duplication harness.sh was
// written to end, one level further out.
func TestTheGateHarnessPassesItsOwnTests(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		// A machine with no Bourne shell cannot run either verify pass either,
		// so this is the same skip those take rather than a gap in this check.
		t.Skipf("no sh on PATH to run harness_test.sh with: %v", err)
	}

	// Run from this directory, which is what the script's own `cd
	// "$(dirname "$0")"` assumes and what `go test` gives it anyway.
	out, err := exec.Command(sh, "harness_test.sh").CombinedOutput()
	if err != nil {
		t.Fatalf("internal/gateharness/harness_test.sh failed (%v).\n\n"+
			"harness.sh is what both ios/verify and android/verify count their gate "+
			"cases with, so an arm of it that has stopped working takes BOTH gate "+
			"tests green with it — which is strictly worse than the two untested "+
			"copies it replaced. Its output:\n\n%s", err, out)
	}
	// A script that exits 0 having printed nothing would be one whose cases had
	// all been deleted, which is the same silent pass one layer down.
	if !strings.Contains(string(out), "OK") {
		t.Errorf("internal/gateharness/harness_test.sh exited 0 without printing an OK "+
			"footer, so nothing says its cases ran:\n\n%s", out)
	}
}
