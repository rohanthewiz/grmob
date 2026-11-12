package testreset_test

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	// Linked for their init functions, which are what this test audits.
	_ "github.com/rohanthewiz/grmob/core"
	_ "github.com/rohanthewiz/grmob/keystore"
	_ "github.com/rohanthewiz/grmob/permission"

	"github.com/rohanthewiz/grmob/internal/testreset"
)

// covered names the resets that are deliberately not registered, each with
// the registered reset that runs it. A reset that one registered reset
// already calls would otherwise run twice, and the comment on the caller is
// where the order between them is explained.
var covered = map[string]string{
	"permission.resetForegroundForTest": "permission.resetForTest",
}

// resetDecl matches a package-level reset seam: an unexported, argument-less
// function whose name starts with reset and ends with ForTest.
var resetDecl = regexp.MustCompile(`(?m)^func (reset[A-Za-z]*ForTest)\(\) \{`)

var packageClause = regexp.MustCompile(`(?m)^package ([a-z][a-z0-9_]*)`)

// Every reset…ForTest in the module's non-test source is registered (or run
// by one that is), and every registration names a function that exists.
//
// This is what keeps grmobtest.Reset whole. A package that grows a new
// process-wide record will grow a reset for its own tests, as eleven have;
// without this census the new one would reach that package's tests and never
// grmobtest's, and an app's tests would leak the record between them with
// nothing in this repository noticing.
//
// The source is scanned by pattern rather than parsed: the declaration shape
// is fixed by the pattern itself, and a scan of every Go file costs a few
// milliseconds where a parse costs a second.
func TestEveryResetSeamIsRegistered(t *testing.T) {
	root := moduleRoot(t)
	var found []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			// Hidden directories, testdata and node_modules hold no package of
			// this module; a directory with its own go.mod is another module.
			if path != root && (strings.HasPrefix(name, ".") || name == "testdata" || name == "node_modules") {
				return filepath.SkipDir
			}
			if path != root {
				if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		decls := resetDecl.FindAllSubmatch(src, -1)
		if len(decls) == 0 {
			return nil
		}
		pkg := packageClause.FindSubmatch(src)
		if pkg == nil {
			t.Errorf("%s declares a reset but has no package clause this test can read", path)
			return nil
		}
		for _, m := range decls {
			found = append(found, string(pkg[1])+"."+string(m[1]))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) == 0 {
		t.Fatal("found no reset…ForTest function in the module: the walk or the pattern is broken, and every check below would pass vacuously")
	}

	registered := testreset.Names()
	for _, name := range found {
		if by, ok := covered[name]; ok {
			if !slices.Contains(registered, by) {
				t.Errorf("%s is listed as run by %s, which is not registered", name, by)
			}
			continue
		}
		if !slices.Contains(registered, name) {
			t.Errorf("%s is not registered with internal/testreset, so grmobtest.Reset never runs it.\n"+
				"Add testreset.Register(%q, …) to its package's testreset.go, or, if another registered reset already calls it, list it in covered.",
				name, name)
		}
	}
	for _, name := range registered {
		// The order test below registers its own probes, and under -shuffle
		// it may run first; they are not seams in the source.
		if strings.HasPrefix(name, "testreset_test.") {
			continue
		}
		if !slices.Contains(found, name) {
			t.Errorf("%s is registered but no such reset…ForTest function is declared in the source", name)
		}
	}
}

// All runs what was registered, in order, and a name cannot be taken twice.
func TestAllRunsEveryRegistrationInOrderAndNamesAreUnique(t *testing.T) {
	var ran []string
	testreset.Register("testreset_test.first", func() { ran = append(ran, "first") })
	testreset.Register("testreset_test.second", func() { ran = append(ran, "second") })
	testreset.All()
	if !slices.Equal(ran, []string{"first", "second"}) {
		t.Fatalf("All ran %v, want [first second]", ran)
	}

	defer func() {
		if recover() == nil {
			t.Fatal("registering a name twice did not panic")
		}
	}()
	testreset.Register("testreset_test.first", func() {})
}

// moduleRoot walks up from the package directory to the go.mod that declares
// grmob, so the census reads the whole module whatever directory go test ran
// it from.
func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
		if err == nil && strings.Contains(string(data), "module github.com/rohanthewiz/grmob\n") {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("no grmob go.mod above the test's directory")
		}
		dir = parent
	}
}
