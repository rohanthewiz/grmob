// Command gen regenerates aria/verify/testdata/aria.json from the W3C ARIA
// specification.
//
//	sh aria/fetch.sh      # once, and whenever the spec is revised
//	go run ./aria/gen
//
// It is a developer step, not a verification step: the fixture it writes is
// committed, and everything that reads the fixture — `go test ./...`,
// wasm/verify, the two web exporters' guards — works from the committed copy
// with no spec, no network and no run of this command. What holds the committed
// copy honest is aria/verify's own conformance test, which regenerates in memory
// and compares whenever a spec download happens to be present.
//
// So there are two ways this fixture can be wrong and both are covered: it can
// disagree with the spec, which the conformance test catches, and it can
// disagree with the code, which every other test in aria/verify catches.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rohanthewiz/grmob/aria/spec"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "aria/gen:", err)
		os.Exit(1)
	}
}

func run() error {
	// Resolved from this file's own package rather than from the working
	// directory, so `go run ./aria/gen` works from anywhere in the tree the
	// way every other generator here does.
	root, err := repoRoot()
	if err != nil {
		return err
	}

	specPath := filepath.Join(root, spec.LocalPath)
	raw, err := os.ReadFile(specPath)
	if err != nil {
		return fmt.Errorf("%w\n\nRun `sh aria/fetch.sh` first — the specification "+
			"is deliberately not committed", err)
	}

	doc, err := spec.Parse(string(raw))
	if err != nil {
		return err
	}
	fx, err := spec.Scope(doc)
	if err != nil {
		return err
	}

	out := filepath.Join(root, spec.FixturePath)
	if err := os.WriteFile(out, fx.Render(), 0o644); err != nil {
		return err
	}
	fmt.Printf("OK: %d roles, %d name-prohibited -> %s\n",
		len(fx.Order), len(fx.NameProhibited), spec.FixturePath)
	return nil
}

// repoRoot walks up from the working directory to the module root, which is the
// directory holding go.mod. `go run ./aria/gen` leaves the working directory
// wherever the caller was, and both paths this command touches are stated
// relative to the module.
func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no go.mod above %s — run this from inside the repository", dir)
		}
		dir = parent
	}
}
