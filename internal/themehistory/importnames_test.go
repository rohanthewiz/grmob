package main

import (
	"fmt"
	"go/ast"
	"sort"
	"strings"
	"sync"
	"testing"
)

// Which identifier in a file means which package, read off that file's import
// block.
//
// # Why this is here and also in wasm/verify
//
// It is the same set of functions — and the same registry they share — as
// wasm/verify/importnames_test.go, which is where the reasoning is written
// down: a census that resolves a package by its conventional NAME is blind to
// an alias, and blind in silence, which is the direction nobody argued for.
// Two of this package's four guarded names are selectors on a package —
// `os.Stdout` and the `fmt.Print` family — so concurrency_test.go asks
// exactly that question of every file here, and
// `import stdio "fmt"` with a `stdio.Printf` in a worker is the whole failure
// the os.Stdout row describes arriving under a name a conventional match would
// not look at.
//
// The copy is for the reason the timings record is a copy: these are two
// separate `package main` programs, one under wasm/ and one under internal/,
// and a package existing so that a handful of helpers could be a handful of
// helpers is the more expensive of the two options.
//
// # What holds the two together, which used to be nothing
//
// The copy's own header used to say there was no arm over it and that naming
// that was better than implying there was. It was right about the state and
// wrong about what could be done: a record can be counted because it is a
// NAME, and two declarations can be COMPARED whatever they are. wasm/verify/
// copies_test.go prints each of these with its doc comment removed and holds
// the results to being identical across every package that declares them, and
// holds those packages to declaring the same set — the functions and the
// package-level state, because the compiler's hold on `dotImportsReported` is
// its name and nothing else.
//
// So the doc comments differ — this one says why it is here, that one says
// what it is — and the code does not, by the same kind of check that holds the
// two timings records to the same five fields.
//
// # What it still does not resolve
//
// A package's real name is a convention rather than a rule; packageBase
// follows the two the toolchain itself follows and no more. A blank import
// binds nothing. A dot import binds the package's names into file scope, where
// no qualifier reaches them, and qualifiersFor reports that rather than
// reading the file as clean.
// Where a file's dot imports have already been reported, so that the same
// fact is a finding once.
//
// Keyed by file and import path. Every census that resolves qualifiers asks
// this question of every Go file it walks, and several of them walk the whole
// repository — so one dot import is one fact and as many findings as there are
// askers, which is the wall this repository writes one-per-row rules against
// everywhere else.
//
// A sync.Map rather than a plain one because the tests in a package may be
// given a t.Parallel() at any point and this is the only state these helpers
// keep. It lives for the test binary, which is the right lifetime: the fact it
// records is about a file on disk, and nothing in a run changes that.
//
// forgetDotImportsReported is the way back out, for the one reader that is not
// a run.
var dotImportsReported sync.Map

// forgetDotImportsReported empties the registry above and returns the
// `file\x00path` keys it held, sorted.
//
// # Why the registry needed a way out at all
//
// A binary-long lifetime is the right one for a run: the fact recorded is a
// fact about a file on disk, and re-reporting it would be the same sentence
// twice about one import block. It is the wrong one for the reader that is not
// a run — a break-test, which puts a deliberately broken input in front of a
// census and asserts the finding. Without this, qualifiersFor was the only
// census here that could not be broken and re-run in place: the second asking
// in a binary is silent by design, and nothing could ask what the first one
// had recorded.
//
// # Why one function and not two
//
// Reading and clearing are the same moment for the only caller there is. A
// break-test wants to know what its own asking recorded AND to leave the
// registry as it found it, and two calls would be two chances to do one of
// them and not the other — which would leave a key behind and make the NEXT
// census silent about a file nobody had reported.
//
// The keys come back sorted for the reason every finding here is sorted: a
// result that arrives in map order cannot be diffed against the last run.
//
// Nothing in an ordinary run calls this, and a census that did would be back
// to reporting one dot import once per asking, which is the wall this whole
// registry exists to stop.
func forgetDotImportsReported() []string {
	var keys []string
	dotImportsReported.Range(func(k, _ any) bool {
		if s, ok := k.(string); ok {
			keys = append(keys, s)
		}
		// Deleting during a Range is defined behaviour for sync.Map: the
		// iteration reflects at most one snapshot of the contents, and a key
		// removed while it runs is simply not visited again.
		dotImportsReported.Delete(k)
		return true
	})
	sort.Strings(keys)
	return keys
}

// packageBase is the identifier an import of this path binds, by convention.
//
// # Why this is not just the last segment
//
// It was, and the last segment is right for every path any census here names —
// `os/exec` is `exec`, `golang.org/x/mod/semver` is `semver`. It is wrong for
// two families that a module graph actually contains, and wrong in SILENCE:
// the qualifier resolves to a name nothing in the file binds, so a call into
// the package is invisible and the census reports a clean file.
//
//	github.com/x/y/v2    the major-version suffix is part of the module path
//	                     and not of the package name. `v2` binds nothing; `y`
//	                     does
//	gopkg.in/yaml.v2     the same idea spelled the older way, with the version
//	                     on the last element rather than after it
//
// Both are conventions rather than rules — a package may declare any name it
// likes, and only loading it would settle the question, which is the cost
// every walk in this repository declines. What these two rules buy is that the
// conventions the Go toolchain itself relies on are followed here too, so the
// remaining guess is the one nobody can close without a build.
//
// checkImportPathsAreImportable is the other half: it holds
// every path a census asks about to being one this module could actually
// import, so a path that is a typo or a module nothing requires is a finding
// rather than a qualifier that matches nothing.
func packageBase(importPath string) string {
	base := importPath
	if i := strings.LastIndex(base, "/"); i >= 0 {
		base = base[i+1:]
	}
	// `…/v2`: the name is the segment before it, and a bare `v2` as a whole
	// path is left alone because there is no segment before it to use.
	if isMajorVersion(base) {
		rest := strings.TrimSuffix(importPath, "/"+base)
		if rest != importPath {
			base = rest
			if i := strings.LastIndex(base, "/"); i >= 0 {
				base = base[i+1:]
			}
		}
	}
	// `yaml.v2`: the version is a suffix on the last element itself.
	if i := strings.LastIndex(base, "."); i >= 0 && isMajorVersion(base[i+1:]) {
		base = base[:i]
	}
	return base
}

// isMajorVersion is whether this path element is a `v` followed by digits.
//
// `v2`, not `v2beta` and not `version`: the major-version suffix a module path
// carries is exactly that shape, and anything looser would take the name off a
// package legitimately called `v1alpha`.
func isMajorVersion(s string) bool {
	if len(s) < 2 || s[0] != 'v' {
		return false
	}
	for _, c := range s[1:] {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// importedAs is the identifiers this file binds to the package at `importPath`,
// and whether it dot-imports it.
//
// A blank import binds nothing and contributes no name. A dot import binds the
// package's exported names into FILE SCOPE, where no qualifier can find them,
// which is why it comes back as its own answer rather than as an empty set —
// the caller reports it instead of reading the file as clean.
func importedAs(file *ast.File, importPath string) (names map[string]bool, dot bool) {
	names = map[string]bool{}
	for _, spec := range file.Imports {
		if unquote(spec.Path.Value) != importPath {
			continue
		}
		if spec.Name == nil {
			names[packageBase(importPath)] = true
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

// dotImportsIn is every path this file dot-imports, sorted.
//
// Read for the message rather than for the answer: a file that dot-imports two
// packages is one edit, and a finding that names one of them sends a reader
// back for the second on the next run.
func dotImportsIn(file *ast.File) []string {
	var out []string
	for _, spec := range file.Imports {
		if spec.Name != nil && spec.Name.Name == "." {
			out = append(out, unquote(spec.Path.Value))
		}
	}
	sort.Strings(out)
	return out
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
// # Why the finding is once per file and not once per asking
//
// Every census that resolves qualifiers asks this of every file it walks, and
// four of them walk the whole repository. One dot import is therefore one fact
// and as many findings as there are askers — the wall this repository writes
// one-per-row rules against everywhere else, arriving here as the same
// sentence printed five times about one line of somebody's import block.
//
// So the fact is recorded in dotImportsReported and reported by whichever
// census reaches it first. Which one that is depends on test order and does
// not matter: the finding names the FILE, the fix is in the file, and a reader
// running one census on its own still gets it, because the record is empty at
// the start of every binary.
//
// The one place that is visible is `-count=2`, where the second iteration of a
// census is silent about a file the first already reported. The run still
// fails — the first iteration's failure is the test's — and the fact being
// recorded is a fact about a file on disk, which does not change between two
// iterations of the same binary. A finding that did repeat there would be the
// same sentence twice about one import line, which is what this is for.
func qualifiersFor(t *testing.T, rel string, file *ast.File, importPath string) map[string]bool {
	t.Helper()
	names, dot := importedAs(file, importPath)
	if !dot {
		return names
	}
	if _, seen := dotImportsReported.LoadOrStore(rel+"\x00"+importPath, true); seen {
		return names
	}
	// Everything else the file dot-imports, so that a file with two of them is
	// one visit rather than two runs. The path asked about is the finding; the
	// rest are the same edit.
	also := ""
	if all := dotImportsIn(file); len(all) > 1 {
		also = fmt.Sprintf("\n\nThis file dot-imports %d packages — %s — and "+
			"the same is true of each of them, whichever census asks next.",
			len(all), strings.Join(all, ", "))
	}
	t.Errorf("%s dot-imports %s, and this check reads calls into that "+
		"package off the qualifier they are written with.\n\n"+
		"A dot import puts the package's exported names into this file's "+
		"own scope, so a call into it is spelled `Compare(a, b)` with "+
		"nothing to resolve — indistinguishable from a call to a function "+
		"of this file's package without loading the package, which is the "+
		"cost every repository walk here declines. This file is therefore "+
		"not read for that question, and the finding is here rather than "+
		"absent so that the gap is visible.\n\n"+
		"Import it the ordinary way — under its own name or an alias, "+
		"both of which resolve exactly — or teach this census the "+
		"unqualified spelling and say what it now costs.%s",
		rel, importPath, also)
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
