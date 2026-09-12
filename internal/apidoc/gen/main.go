// Command gen writes docs/api/ — the generated API reference — from the doc
// comments of grmob's public packages.
//
//	go run ./internal/apidoc/gen
//
// It is a developer step, not a verification step: the pages it writes are
// committed, and reading the docs (or serving them with `./docs.sh`) needs
// neither this command nor a toolchain. What keeps the committed copy honest is
// internal/apidoc's own test, which regenerates in memory and fails when the
// two disagree — so a declaration that changes without a regeneration breaks
// `go test ./...`, in the same way aria/gen's fixture is held to the spec.
//
// Pages that no longer correspond to a documented package are deleted, so
// removing an entry from apidoc.Packages (or a whole package from the tree)
// cannot leave an orphan page behind, reachable by URL and updated by nothing.
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/rohanthewiz/grmob/internal/apidoc"
)

func main() {
	if err := run(os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "apidoc/gen:", err)
		os.Exit(1)
	}
}

// run is the whole command with its output stream passed in, so that the
// summary line — the only report the command makes about what it did — is
// something a test can read. main supplies the real stream.
func run(out io.Writer) error {
	root, err := apidoc.RepoRoot()
	if err != nil {
		return err
	}

	pages, err := apidoc.Generate(root)
	if err != nil {
		return err
	}

	docsDir := filepath.Join(root, "docs")
	apiDir := filepath.Join(docsDir, apidoc.DocsSubdir)
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		return err
	}

	// Written before the prune so that a failure part way through leaves the
	// tree with too many pages rather than too few.
	names := make([]string, 0, len(pages))
	for name := range pages {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		dest := filepath.Join(docsDir, filepath.FromSlash(name))
		if err := os.WriteFile(dest, pages[name], 0o644); err != nil {
			return err
		}
	}

	pruned, err := prune(apiDir, pages)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "OK: %d pages -> docs/%s/", len(pages), apidoc.DocsSubdir)
	if pruned > 0 {
		fmt.Fprintf(out, " (%d stale removed)", pruned)
	}
	fmt.Fprintln(out)
	return nil
}

// prune removes markdown files under apiDir that this run did not write.
//
// Only .md files are considered. The directory is part of the served docs tree,
// and gkdocs serves whatever else lives there as a static file, so a reader who
// puts an image next to a page should not have it deleted by the next
// regeneration.
func prune(apiDir string, pages map[string][]byte) (int, error) {
	entries, err := os.ReadDir(apiDir)
	if err != nil {
		return 0, err
	}

	var n int
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".md" {
			continue
		}
		if _, written := pages[apidoc.DocsSubdir+"/"+e.Name()]; written {
			continue
		}
		if err := os.Remove(filepath.Join(apiDir, e.Name())); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}
