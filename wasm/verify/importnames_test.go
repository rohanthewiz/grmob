package main

import (
	"go/ast"
	"path"
	"strings"
	"testing"
)

// Which identifier in a file means which package, read off that file's import
// block.
//
// # What this is about
//
// Every census in this repository that asks "is this call into package P"
// answers it by comparing the qualifier to P's conventional NAME:
//
//	pkg.Name == "semver"    versionorder_test.go, the comparator detector
//	pkg.Name == "parser"    repowalks_test.go, the walk-depth reader
//	pkg.Name == "testing"   shortlever_test.go, the lever walk
//	pkg.Name == "exec"      gitquoting_test.go, the git-wrapper taint
//	pkg.Name == "strings"   versionorder_test.go, the split-and-parse half
//	pkg.Name == "os"/"fmt"  internal/themehistory/concurrency_test.go
//
// Six files and one assumption, and each of them wrote down the same defence:
// matching a bare name is loose in the direction of REPORTING, which is the
// safe direction for a census whose failure is a question. That is true and it
// is only half of the question, because the assumption fails in BOTH
// directions and only one of them was argued:
//
//	the false positive    a local variable called `semver`, or a package of
//	                      this repository's own aliased to it, that exports a
//	                      Compare. Reported as a comparator; a reader looks and
//	                      dismisses it. This is the half every header described
//	the false negative    `import sv "golang.org/x/mod/semver"`, then
//	                      `sv.Compare(a, b)`. Invisible. Nothing is reported,
//	                      the count still reads one, and the census that exists
//	                      to notice a second comparator arriving does not
//
// The second half is the one that matters, because it is silent: a census that
// cannot see the change it asks for stops being true at the moment somebody
// makes it. And it is the change versionorder_test.go's own message asks for
// by name — "semver answers every rule compareVersions has" — so the one
// spelling most likely to arrive is the one an alias hides.
//
// # Why this is settled rather than argued about
//
// Every one of those walks has already parsed the file. `file.Imports` is
// right there, it is exact for the question being asked, and reading it costs
// nothing measurable: an import block is a handful of specs, and the walks
// that use this are already handing every Go file in the repository to
// go/parser.
//
// So the qualifier is no longer a convention. It is what THIS FILE binds, and
// a file that binds nothing to the path binds nothing: `semver.Compare` in a
// file that does not import x/mod/semver is some other `semver`, which closes
// the false positive in the same reading.
//
// # What it does not resolve, and why each is where it is
//
//	the package's real name    taken as the last segment of the import path.
//	                           Exact for every path any census here names —
//	                           `os/exec` is `exec`, `golang.org/x/mod/semver`
//	                           is `semver` — and wrong in general, for
//	                           `gopkg.in/yaml.v2` and the handful like it.
//	                           Getting that right means loading the package,
//	                           which is the cost every one of these walks
//	                           declines for the same written reason: they read
//	                           revisions and generated trees that need not
//	                           build
//	a blank import             `_ "…"` binds no identifier, so no call can
//	                           qualify through it and it contributes no name
//	a dot import               `. "…"` puts the package's exported names into
//	                           FILE SCOPE, so a call into it has no qualifier
//	                           at all and no amount of reading the import
//	                           block finds it. That one is reported rather
//	                           than approximated — see qualifiersFor
//
// The last is the only residue, and it is a finding rather than a silence,
// which is the whole difference this file is about.
func importedAs(file *ast.File, importPath string) (names map[string]bool, dot bool) {
	names = map[string]bool{}
	for _, spec := range file.Imports {
		if unquote(spec.Path.Value) != importPath {
			continue
		}
		// No name on the spec is the ordinary case: the identifier is the
		// package's own name, which for every path named by a census here is
		// the last segment of the path.
		if spec.Name == nil {
			names[path.Base(importPath)] = true
			continue
		}
		switch spec.Name.Name {
		case "_":
			// Imported for its initialisation. Nothing can be called through
			// it, so it is not a name.
		case ".":
			dot = true
		default:
			names[spec.Name.Name] = true
		}
	}
	return names, dot
}

// qualifiersFor is importedAs with the dot-import case turned into a finding.
//
// # Why the dot import is an error and not a shrug
//
// A census that resolves qualifiers can say something exact about every file
// that imports a package the ordinary way, and nothing at all about one that
// dot-imports it: the calls it is looking for are spelled `Compare(a, b)`,
// which is indistinguishable from a call to a function of the file's own
// package without the type information these walks deliberately do not have.
//
// That is a hole of exactly one shape, and the two things that could be done
// with it are to widen every census to treat bare calls as possible package
// calls — which would report most of this repository — or to say so when it
// happens. Nothing here dot-imports anything today, so the second costs
// nothing and the first would cost the census its usefulness.
//
// The message is the same wherever it fires, which is why it is here rather
// than repeated at five call sites: a reader who hits it needs to know that
// the file was not read, not that one particular arm gave up on it.
func qualifiersFor(t *testing.T, rel string, file *ast.File, importPath string) map[string]bool {
	t.Helper()
	names, dot := importedAs(file, importPath)
	if dot {
		t.Errorf("%s dot-imports %s, and this check reads calls into that "+
			"package off the qualifier they are written with.\n\n"+
			"A dot import puts the package's exported names into this file's "+
			"own scope, so a call into it is spelled `Compare(a, b)` with "+
			"nothing to resolve — indistinguishable from a call to a function "+
			"of this file's package without loading the package, which is the "+
			"cost every repository walk in wasm/verify declines. This file is "+
			"therefore not read for that question, and the finding is here "+
			"rather than absent so that the gap is visible.\n\n"+
			"Import it the ordinary way — under its own name or an alias, "+
			"both of which resolve exactly — or teach this census the "+
			"unqualified spelling and say what it now costs.", rel, importPath)
	}
	return names
}

// unquote is an import path's value without its quotes.
//
// Import paths are ordinary interpreted string literals with nothing in them
// worth decoding — no Go path contains an escape — so this is a trim rather
// than a strconv.Unquote. What matters is that a malformed spec comes back as
// something that matches no path rather than as a bare prefix of one.
func unquote(lit string) string {
	return strings.Trim(lit, "`\"")
}
