// Command gen writes core/style_equal_gen.go, core.Style's generated Equal
// method; see internal/styleeq for why it is generated.
//
//	go run ./internal/styleeq/gen
//
// Run it after adding, removing or retyping a field of core.Style.
// internal/styleeq's test fails until it is run.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rohanthewiz/grmob/internal/apidoc"
	"github.com/rohanthewiz/grmob/internal/styleeq"
)

func main() {
	root, err := apidoc.RepoRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, "styleeq/gen:", err)
		os.Exit(1)
	}
	dir := filepath.Join(root, "core")
	src, err := styleeq.Generate(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "styleeq/gen:", err)
		os.Exit(1)
	}
	out := filepath.Join(dir, styleeq.GeneratedFile)
	if err := os.WriteFile(out, src, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "styleeq/gen:", err)
		os.Exit(1)
	}
	fmt.Println("OK:", out)
}
