package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
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
//	the package's real name    taken from the import path by the conventions
//	                           the toolchain itself follows — the last
//	                           segment, with a `/v2` or a `.v2` understood as
//	                           a version rather than a name. Exact for every
//	                           path any census here uses, and still a
//	                           convention rather than a rule: a package may
//	                           declare any name, and settling it means loading
//	                           the package, which every walk here declines for
//	                           the same written reason. See packageBase, and
//	                           checkImportPathsAreImportable
//	                           for the half of that gap go.mod can close
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
//
// # And why these five functions are also in internal/themehistory
//
// They are a copy, for the reason the timings record is a copy: two separate
// `package main` programs, one under wasm/ and one under internal/, and a
// package existing so that five helpers could be five helpers is the more
// expensive of the two options.
//
// What is new is that the copy is HELD. The timings record could be counted
// because a record is a NAME — `…TimingsTakenOn`, readable off a parse — and
// the argument that a helper is only a shape was true of finding one and not
// of comparing two. copies_test.go does the comparison: it prints each
// declaration with its doc comment removed and holds the results to being
// identical, which is exact, and it holds the two packages to declaring the
// same SET of them, which is what a divergence would start as.
//
// The divergence it was written for had already happened: this copy resolved a
// package name with path.Base and the other one open-coded it, and this one
// reported a dot import through `t` while the other returned a bool its caller
// reported in its own words. Neither was wrong and nothing would ever have
// said they had drifted.
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
var dotImportsReported sync.Map

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

// The functions this repository keeps one copy of per package that resolves
// qualifiers, and which copies_test.go holds to being the same in each.
//
// # Why the whole set and not just the one that matters
//
// `importedAs` is the function with the reasoning behind it; the other four
// are what it is made of and what reports its one residue. A package that had
// `importedAs` and resolved the version suffix its own way, or that had it and
// reported dot imports in its own words, would be the divergence this is about
// — and both of those are exactly what the two copies had actually done before
// anything compared them.
//
// So the unit is the SET. A package declaring some of these and not the others
// is a copy being taken apart, which is a finding while the file is still in
// front of somebody rather than after the two have answered differently.
var importResolverShapes = []string{
	"packageBase",
	"isMajorVersion",
	"importedAs",
	"dotImportsIn",
	"qualifiersFor",
	"unquote",
}

// How many packages the copy argument covers.
//
// Two, and it is the same constant as timingsRecordCopies with the same
// reasoning behind it: two separate `package main` programs is a copy with a
// written reason, and a third is a shape being kept in step across three
// places by whoever remembers to.
//
// The difference from the timings record is what happens at two, and it is the
// point of this pair of numbers sitting beside each other. The record's copy
// was held only to being the same SHAPE — five fields and an arm — because the
// values are different numbers about different machines and were never meant
// to match. These are code, so the copies are held to being IDENTICAL, which
// is the stronger thing and the cheaper one: there is nothing to decide about
// whether a difference is meant.
const importResolverCopies = 2

// checkImportResolverCopies holds every copy of the shapes above to being the
// same function, and every package that has one to having all of them.
//
// # What is compared, and why it is not the bytes
//
// The declaration is printed with go/printer and its doc comment removed. That
// makes the comparison exactly about the code: the two headers SHOULD differ —
// one says what the function is and the other says why it is a second copy —
// and interior comments and whitespace are normalised by the printer rather
// than argued about.
//
// What it cannot see is a difference in something the copies both call. That
// is the reason the set is the unit: a `packageBase` that had drifted is a
// `packageBase` this compares, and a helper outside the set would have to be
// added to it.
func checkImportResolverCopies(t *testing.T, decls []importResolverDecl) {
	t.Helper()

	byName := map[string][]importResolverDecl{}
	byDir := map[string]map[string]bool{}
	for _, d := range decls {
		byName[d.name] = append(byName[d.name], d)
		if byDir[d.dir] == nil {
			byDir[d.dir] = map[string]bool{}
		}
		byDir[d.dir][d.name] = true
	}

	// The walk reaching anything. Every arm here says this: a census over
	// nothing passes silently and reads as a clean result, and this one is
	// looking for NAMES that a rename would take away without touching a line
	// of what they do.
	var missing []string
	for _, name := range importResolverShapes {
		if len(byName[name]) == 0 {
			missing = append(missing, name)
		}
	}
	if len(missing) == len(importResolverShapes) {
		t.Errorf("none of the %d import-resolving shape(s) was found in the "+
			"repository: %s.\n\n"+
			"These are what every census that asks \"is this a call into "+
			"package P\" resolves the qualifier with, and this repository has "+
			"two copies of them — wasm/verify/importnames_test.go and "+
			"internal/themehistory/importnames_test.go. Either the walk is "+
			"not reaching them or they have been renamed, and in both cases "+
			"the two copies are back to being kept in step by whoever "+
			"remembers to.",
			len(importResolverShapes), strings.Join(importResolverShapes, ", "))
		return
	}

	// Every package that declares any of them declaring all of them. A partial
	// set is a copy coming apart, and it is the state that precedes two
	// packages answering the same question differently.
	dirs := make([]string, 0, len(byDir))
	for dir := range byDir {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)
	for _, dir := range dirs {
		var absent []string
		for _, name := range importResolverShapes {
			if !byDir[dir][name] {
				absent = append(absent, name)
			}
		}
		if len(absent) == 0 {
			continue
		}
		t.Errorf("%s declares %d of the %d import-resolving shapes and not "+
			"%s.\n\n"+
			"The set is the unit. `importedAs` is the function with the "+
			"reasoning behind it and the others are what it is made of and "+
			"what reports its one residue — a package that resolves a version "+
			"suffix its own way, or reports a dot import in its own words, has "+
			"a copy that will answer differently from the other one and "+
			"nothing that would ever say so.\n\n"+
			"Either take the whole set or none of it, and if this package "+
			"genuinely needs a different answer, that is a reason to write "+
			"down rather than a function to leave out.",
			dir, len(byDir[dir]), len(importResolverShapes),
			strings.Join(absent, ", "))
	}

	// And the copies themselves being the same function.
	for _, name := range importResolverShapes {
		found := byName[name]
		sort.Slice(found, func(i, j int) bool { return found[i].rel < found[j].rel })
		for i := 1; i < len(found); i++ {
			if found[i].text == found[0].text {
				continue
			}
			t.Errorf("%s is declared at %s:%d and at %s:%d and the two are "+
				"not the same function.\n\n"+
				"These are a copy on purpose — see the header of either file "+
				"— and a copy that has drifted is the worst of both: two "+
				"packages answering the same question differently, with the "+
				"reasoning written down once. The comparison is of the code "+
				"alone, with doc comments removed and the printer's "+
				"formatting, so the two headers are free to say different "+
				"things and this is a real difference in what the function "+
				"does.\n\n%s:%d has:\n\n%s\n\n%s:%d has:\n\n%s\n\n"+
				"Make one a copy of the other, or — if the two packages now "+
				"need different answers — say so where the copy argument is "+
				"and take this shape out of importResolverShapes.",
				name, found[0].rel, found[0].line, found[i].rel, found[i].line,
				found[0].rel, found[0].line, found[0].text,
				found[i].rel, found[i].line, found[i].text)
		}
	}

	// The trigger, which is the same one the timings record has and for the
	// same reason.
	whole := 0
	for _, dir := range dirs {
		if len(byDir[dir]) == len(importResolverShapes) {
			whole++
		}
	}
	if whole > importResolverCopies {
		t.Errorf("%d package(s) declare the import-resolving shapes and the "+
			"argument for copying them was written for %d: %s.\n\n"+
			"That argument is about two separate `package main` programs, one "+
			"under wasm/ and one under internal/, and the cost of a package "+
			"existing so that %d helpers could be %d helpers. At %d it is no "+
			"longer that trade: it is a shape kept in step across %d places by "+
			"a check, which works, and by whoever remembers to update the "+
			"third when the reasoning changes, which does not.\n\n"+
			"Extract them into a package all three can import, or raise "+
			"importResolverCopies with the reason beside the copy argument, so "+
			"the next person reads a decision rather than a number.",
			whole, importResolverCopies, strings.Join(dirs, ", "),
			len(importResolverShapes), len(importResolverShapes), whole, whole)
	}

	t.Logf("%d import-resolving shape(s), %d copy(ies) each, held identical "+
		"across %s.", len(importResolverShapes), whole,
		strings.Join(dirs, ", "))
}

// checkImportPathsAreImportable holds every import path a census names to
// being one this module could actually import.
//
// # What this closes
//
// importedAs resolves a qualifier by comparing it to the name the import path
// binds, and a path nothing could import binds nothing. So a typo in a census
// — `golang.org/x/mod/semvr`, `os/exex` — produces an empty name set, every
// file reads as not calling into that package, and the census passes over the
// whole repository saying nothing. That is the same silent direction the
// alias problem was, one level up: not the identifier a call is written with,
// but the PATH the census is asking about.
//
// # How a path is judged, with no package loaded
//
//	the standard library    a first element with no dot in it. That is the
//	                        module-path rule read backwards, and it is exact:
//	                        a module path must have a dot in its first element,
//	                        so a path without one is the standard library or
//	                        nothing
//	everything else         has to be this module or something go.mod
//	                        requires, by prefix. A path under a required
//	                        module might still be a directory that does not
//	                        exist, which only a build settles — but a path
//	                        under NO required module cannot be imported by
//	                        anything here, whatever is on disk
//
// The standard library side is the looser one: `os/exex` has no dot and passes.
// Closing that means a list of every stdlib path, which is a second copy of
// something that changes every release — so it is a written limit rather than
// a check, and the module side, which is where the version suffixes and the
// interesting names live, is exact.
//
// # And the argument that is not a literal
//
// A census whose path is computed cannot be checked here at all, and is
// reported for that reason alone. There is none today; the point is that the
// thing which would make this arm silently useless is itself a finding.
func checkImportPathsAreImportable(t *testing.T, root string, asks []importPathAsk) {
	t.Helper()
	module, requires, err := moduleRequirements(root)
	if err != nil {
		t.Errorf("reading go.mod to check the import paths these censuses "+
			"name: %v.\n\nWithout it, a path no module provides is a "+
			"qualifier that matches nothing and a census that passes over the "+
			"whole repository in silence.", err)
		return
	}
	if len(asks) == 0 {
		t.Errorf("no import path was found being asked about, and this " +
			"repository's censuses ask about at least six — os/exec, " +
			"go/parser, testing, strings, strconv and " +
			"golang.org/x/mod/semver.\n\nEither the scan is not reaching the " +
			"call sites or the helpers have been renamed, and either way the " +
			"paths those censuses resolve are now checked by nothing.")
		return
	}

	sort.Slice(asks, func(i, j int) bool {
		if asks[i].rel != asks[j].rel {
			return asks[i].rel < asks[j].rel
		}
		return asks[i].line < asks[j].line
	})
	seen := map[string]bool{}
	var paths []string
	for _, a := range asks {
		if a.path == "" {
			t.Errorf("%s:%d asks about an import path that is not a literal "+
				"(%s, in %s).\n\n"+
				"A census resolves a qualifier by the name its path binds, so "+
				"a path that is computed is one nothing here can hold to being "+
				"importable — and a path no module provides binds no name, "+
				"which means the census reads every file as clean and reports "+
				"nothing at all. Name the path where it is asked about, or "+
				"check it wherever it is built.", a.rel, a.line, a.how, a.fn)
			continue
		}
		if !seen[a.path] {
			seen[a.path] = true
			paths = append(paths, a.path)
		}
		first := a.path
		if i := strings.Index(first, "/"); i >= 0 {
			first = first[:i]
		}
		if !strings.Contains(first, ".") {
			// The standard library, by the module-path rule read backwards.
			continue
		}
		if a.path != module && !strings.HasPrefix(a.path, module+"/") {
			provided := false
			for _, req := range requires {
				if a.path == req || strings.HasPrefix(a.path, req+"/") {
					provided = true
					break
				}
			}
			if !provided {
				t.Errorf("%s:%d asks about %q, and go.mod requires no module "+
					"that provides it (module %s; %d requirement(s): %s).\n\n"+
					"A path nothing can import binds no identifier, so "+
					"importedAs comes back empty for every file, every call "+
					"into the package is invisible, and the census passes over "+
					"the whole repository having asked nothing. That is the "+
					"same silence an aliased import used to produce, one level "+
					"up — the path the question is about rather than the name "+
					"the answer is written with.\n\n"+
					"If this is a dependency being added, the require comes "+
					"first; if it is a typo, the census has been reporting "+
					"nothing since it was written.",
					a.rel, a.line, a.path, module, len(requires),
					strings.Join(requires, ", "))
				continue
			}
		}
		// And the name the path binds still being readable by convention. The
		// two version spellings are handled; a base that still carries a dot
		// is one packageBase has no rule for.
		if base := packageBase(a.path); strings.Contains(base, ".") ||
			isMajorVersion(base) {
			t.Errorf("%s:%d asks about %q, and the identifier an ordinary "+
				"import of it binds comes out as %q.\n\n"+
				"packageBase knows the two version conventions — a `/v2` "+
				"element and a `.v2` suffix — and this path is neither, so "+
				"the name is a guess that looks wrong. A qualifier resolved "+
				"to the wrong identifier matches nothing, which makes this "+
				"census silent rather than incorrect.\n\n"+
				"Either give packageBase the rule this path needs, or have "+
				"the census import the package under an explicit alias and "+
				"ask about that.", a.rel, a.line, a.path, base)
		}
	}
	sort.Strings(paths)
	t.Logf("%d import path(s) asked about by the censuses here, each one this "+
		"module could import: %s.", len(paths), strings.Join(paths, ", "))
}

// moduleRequirements is this module's own path and the module paths it
// requires, read out of go.mod.
//
// # Why the file is lexed rather than loaded
//
// golang.org/x/mod/modfile would parse this properly and it is in the module
// graph, so the import is affordable. What it is not is FREE of the thing
// versionorder_test.go's whole census is about: x/mod is held here by the
// `tool` block as an indirect requirement, and importing a package from it
// makes it direct — which is a change to this file's own subject matter made
// in order to read this file.
//
// What is needed is much less than a parse. A require line is a module path
// and a version, in a block or on its own, and the paths are all that is read:
// versions, replace directives and the `tool` block are somebody else's
// question. A line this misreads produces a requirement that matches no ask,
// which reports a path as unprovided — the ASKING direction, and the safe one.
func moduleRequirements(root string) (module string, requires []string, err error) {
	raw, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", nil, err
	}
	inBlock := false
	for _, line := range strings.Split(string(raw), "\n") {
		// The comment first: `// indirect` is the common tail, and a commented
		// out require is not one.
		if i := strings.Index(line, "//"); i >= 0 {
			line = line[:i]
		}
		line = strings.TrimSpace(line)
		switch {
		case line == "":
			continue
		case inBlock:
			if line == ")" {
				inBlock = false
				continue
			}
		case strings.HasPrefix(line, "module "):
			module = strings.TrimSpace(strings.TrimPrefix(line, "module "))
			continue
		case line == "require (":
			inBlock = true
			continue
		case strings.HasPrefix(line, "require "):
			line = strings.TrimSpace(strings.TrimPrefix(line, "require "))
		default:
			continue
		}
		// What is left is `path version`, or a stray line in a block that is
		// not a requirement, which yields a path nothing matches.
		if i := strings.IndexAny(line, " \t"); i >= 0 {
			line = line[:i]
		}
		if line != "" && line != "(" && line != ")" {
			requires = append(requires, line)
		}
	}
	if module == "" {
		return "", nil, fmt.Errorf("no `module` line in %s",
			filepath.Join(root, "go.mod"))
	}
	sort.Strings(requires)
	return module, requires, nil
}

// importResolverDecl is one declaration of a shared shape that the walk found.
type importResolverDecl struct {
	name string
	// The directory, which is the unit a copy is counted in: these are
	// package-level functions and a package is a directory here.
	dir  string
	rel  string
	line int
	// The declaration printed with its doc comment removed — see
	// declarationText.
	text string
}

// importPathAsk is one place a census names an import path.
type importPathAsk struct {
	rel  string
	line int
	// The path, or "" when the argument is not a string literal.
	path string
	// How it is written, for the message when it is not a literal.
	how string
	// The declaration the call is in.
	fn string
}

// declarationText is a function declaration as code, with its doc comment
// removed.
//
// The doc comment is the part two copies are SUPPOSED to differ in — one
// header says what the function is, the other says why there is a second one
// — and the printer normalises everything else, so what comes back is exactly
// the thing a copy has to keep in step.
func declarationText(fset *token.FileSet, fn *ast.FuncDecl) string {
	stripped := *fn
	stripped.Doc = nil
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, &stripped); err != nil {
		// A declaration the printer cannot render is not a comparison this
		// can make; the error travels as the text so that two of them compare
		// unequal and the finding names the file.
		return "unprintable: " + err.Error()
	}
	return buf.String()
}

// importResolverDeclarationsIn collects the shared shapes this file declares,
// and the import paths its censuses ask about.
//
// Both readings are of the same parse and are kept together because they are
// the same walk over the same declarations: one looks at what a file DECLARES
// and the other at what it CALLS.
//
// A call inside one of the shapes is not an ask. `qualifiersFor` calls
// `importedAs` with the path it was handed, which is the implementation rather
// than a question put to the repository, and reporting it would mean every
// copy of the helper reporting itself as an unliteral path.
func importResolverDeclarationsIn(fset *token.FileSet, rel string,
	file *ast.File) ([]importResolverDecl, []importPathAsk) {

	shape := map[string]bool{}
	for _, name := range importResolverShapes {
		shape[name] = true
	}
	var decls []importResolverDecl
	var asks []importPathAsk
	for _, d := range file.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		if fn.Recv == nil && shape[fn.Name.Name] {
			decls = append(decls, importResolverDecl{
				name: fn.Name.Name,
				dir:  path.Dir(rel),
				rel:  rel,
				line: fset.Position(fn.Pos()).Line,
				text: declarationText(fset, fn),
			})
			// The shapes calling each other is what they are made of.
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			id, ok := call.Fun.(*ast.Ident)
			if !ok || len(call.Args) == 0 {
				return true
			}
			// The two entry points, whose last argument is the path. Anything
			// else that happens to be called with a path is somebody else's
			// question.
			if id.Name != "importedAs" && id.Name != "qualifiersFor" {
				return true
			}
			arg := call.Args[len(call.Args)-1]
			ask := importPathAsk{
				rel:  rel,
				line: fset.Position(call.Pos()).Line,
				fn:   fn.Name.Name,
			}
			if lit, ok := arg.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				ask.path = unquote(lit.Value)
			} else {
				ask.how = fmt.Sprintf("a %T", arg)
			}
			asks = append(asks, ask)
			return true
		})
	}
	return decls, asks
}
