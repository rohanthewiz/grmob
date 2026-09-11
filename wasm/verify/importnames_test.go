package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
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
// # And why these functions are also in internal/themehistory
//
// They are a copy, for the reason the timings record is a copy: two separate
// `package main` programs, one under wasm/ and one under internal/, and a
// package existing so that a handful of helpers could be a handful of helpers
// is the more expensive of the two options.
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

// The registry above being readable and clearable, and being what makes the
// second asking silent.
//
// # Which half of the rule this can hold, and which half is a break-test
//
// The rule has two directions and only one of them can be asserted from
// inside the binary it is about:
//
//	the first asking     reports, and records the fact. Asserting that means
//	                     catching a t.Errorf, and a *testing.T cannot be made
//	                     to fail quietly — a subtest that fails fails its
//	                     parent, which is the whole point of it. This half is
//	                     a break-test: put a dot import in a file, run any
//	                     census, read the finding
//	the second asking    is silent, because the fact is already recorded. That
//	                     is a `qualifiersFor` that must NOT report, which is
//	                     exactly the shape a test can assert
//
// So the fact is recorded first — by hand, which is what the registry's own
// key format is for — and the asking below is therefore the second one. What
// it proves is the thing the previous arrangement could not: that the
// suppression is real, that what was recorded can be read back, and that it
// can be taken out again.
//
// # Why the registry is put back
//
// It lives for the test binary and every census in this package reads it. A
// test that emptied it would make the NEXT census report a dot import that
// something had already reported — the same sentence twice about one import
// block, which is the wall the registry exists to be. Nothing here dot-imports
// anything today, so the set being restored is empty; the restore is for the
// day it is not.
func TestTheDotImportRegistryCanBeReadAndCleared(t *testing.T) {
	// A path no file has, so that seeding it cannot collide with a real
	// finding some census in this binary is about to make.
	const rel = "wasm/verify/testdata/no-such-file.go"
	// The import path is spelled out at each use rather than held in a
	// constant, and that is a fact about the arm two functions down:
	// checkImportPathsAreImportable reads the last argument of every
	// qualifiersFor call and holds it to being a path this module could
	// import, and what it reads is a LITERAL. A constant here would be
	// reported as a path nothing can check — correctly, and about this test
	// rather than about any census.
	key := rel + "\x00" + "go/parser"

	held := forgetDotImportsReported()
	defer func() {
		for _, k := range held {
			dotImportsReported.Store(k, true)
		}
	}()

	src := "package p\n\nimport . \"go/parser\"\n\nvar _ = Mode(0)\n"
	file, err := parser.ParseFile(token.NewFileSet(), rel, src,
		parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parsing the synthetic dot-importing file this test asks "+
			"about: %v.\n\nWithout it there is nothing to ask, and the "+
			"registry's suppression is unchecked.", err)
	}

	// The fact, recorded as the first asking would have recorded it.
	dotImportsReported.Store(key, true)
	if names := qualifiersFor(t, rel, file, "go/parser"); len(names) != 0 {
		// A dot import binds nothing a qualifier can reach, which is the
		// reason the finding exists at all.
		t.Errorf("qualifiersFor returned %d name(s) for a file that only "+
			"dot-imports go/parser, and a dot import binds the package's "+
			"names into file scope where no qualifier reaches them.",
			len(names))
	}
	// If the suppression had not worked, the call above would have failed this
	// test with the dot-import finding, which is the assertion.

	after := forgetDotImportsReported()
	if len(after) != 1 || after[0] != key {
		t.Errorf("the registry held %d key(s) after one asking about a "+
			"dot-imported path, and the one recorded was %q: %q.\n\n"+
			"forgetDotImportsReported is what makes this census breakable "+
			"and re-runnable in place — see its header. A registry that "+
			"cannot be read back is one whose suppression nothing can tell "+
			"from a census that never asked.", len(after), key, after)
	}
	if left := forgetDotImportsReported(); len(left) != 0 {
		t.Errorf("the registry still held %d key(s) after being cleared: "+
			"%q.\n\nClearing and reading are one call because a break-test "+
			"has to leave the registry as it found it; one that only read "+
			"would leave the key behind and make the next census silent "+
			"about a file nobody had reported.", len(left), left)
	}
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

// The functions this repository keeps one copy of per package that resolves
// qualifiers, and which copies_test.go holds to being the same in each.
//
// # Why the whole set and not just the one that matters
//
// `importedAs` is the function with the reasoning behind it; the others are
// what it is made of, what reports its one residue, and what lets that residue
// be unrecorded again. A package that had `importedAs` and resolved the
// version suffix its own way, or that had it and reported dot imports in its
// own words, would be the divergence this is about — and both of those are
// exactly what the two copies had actually done before anything compared them.
//
// So the unit is the SET. A package declaring some of these and not the others
// is a copy being taken apart, which is a finding while the file is still in
// front of somebody rather than after the two have answered differently.
var twoCopyFunctionShapes = []string{
	"packageBase",
	"isMajorVersion",
	"importedAs",
	"dotImportsIn",
	"qualifiersFor",
	"unquote",
	"forgetDotImportsReported",
	// Not an import-resolving helper, and the reason this set stopped being
	// named for them. recordedBand reads a timings record's band back out of
	// the prose the field opens with; both packages compare a reading
	// against a band and neither can import the other's tests, so it is a
	// second copy with the same standing as the seven above. See
	// internal/themehistory/band_test.go and
	// wasm/verify/wholefileband_test.go.
	"recordedBand",
	// The sentence recordedBand's two callers print about where a reading
	// fell in the band. Here for a reason the others are not: the two
	// verdicts are DIFFERENT functions — one is a t.Logf fragment and the
	// other a TestMain line to stderr — and the one thing they must not
	// diverge on is what a placement means. A reading "at the floor" in one
	// package and "3% up" in the other, from the same arithmetic spelled
	// twice, is two records that cannot be read against each other, which is
	// the whole purpose of their being a pair.
	"bandPlacement",
	// The sentence built out of a placement, a band and a machine comparison:
	// UNDER by this much, OVER by that much, or in the band and where. Both
	// packages build it and the two must not drift, for the reason
	// bandPlacement is here — a reader holding one record's verdict against
	// the other's needs the same words to mean the same thing.
	"againstBandGiven",
	// The lever's reader, registered beside the two constants it reads rather
	// than on its own account: the three together are what make
	// `GRMOB_BAND_VERDICT=required` mean the same thing in both packages, and
	// a reader that compared the value differently — a prefix, say, or a
	// case-insensitive match — would diverge as quietly as a misspelled name.
	// It was registrable all along; nothing had looked.
	"bandVerdictWanted",
}

// The package-level VALUE declarations the two packages share, held to being
// the same declaration in every copy for the same reason the functions are.
//
// # The name, which was `twoCopyValueShapes` and had stopped being true
//
// It was named for its first and only entry: `dotImportsReported` is state,
// and the paragraphs below are about what the compiler holds about state and
// what it does not. Then the band reader's pattern arrived, which is a `var`
// and not state, and then three `const` levers, which are neither. A list
// named for what its first member happened to be is the defect this
// repository renamed a whole file over — see sharedparse_test.go's header.
//
// What actually unifies them is how the walk FINDS them: a ValueSpec inside a
// GenDecl, which is one branch of the parse whether the keyword is `var` or
// `const`. That is a fact about the shape of the source, it is what the
// paragraph below already argues the list is for, and it will stay true of
// every future entry — which is what a name should be built on.
//
// # What the compiler was already holding, and what it was not
//
// `dotImportsReported` has to exist in both packages or neither builds:
// qualifiersFor reads it, and a copy without it is a compile error rather than
// a divergence. That is real and it is only the NAME. The compiler has no
// opinion about what kind of thing it is, so a plain `map[string]bool` in one
// package and a `sync.Map` in the other compiles in both places, reads
// identically, and differs the moment either package's tests are given a
// t.Parallel() — which is exactly the state the two `importedAs` bodies were
// in before anything compared them: two packages answering the same question
// differently, with the reasoning written down once.
//
// So the state is held to the same rule the code is. The walk finds it the
// same way — a package-level declaration with this name — and the comparison
// is of the declaration as the printer renders it, which for a `var` is the
// name and the type and whatever initialiser there is.
//
// # Why this is a separate list and not seven entries in the one above
//
// The two are found differently. A function is an *ast.FuncDecl and this is a
// ValueSpec inside a GenDecl, and the walk has to know which it is looking for
// before it can look. Keeping them apart makes that a fact about the list
// rather than a guess about the name — and a `func dotImportsReported` added
// by mistake is then a shape that is MISSING from its package rather than one
// that quietly matched.
var twoCopyValueShapes = []string{
	"dotImportsReported",
	// recordedBand's pattern. State for the same reason dotImportsReported
	// is: the compiler insists both packages HAVE it, because the function
	// beside it reads it, and has no opinion about whether the two spell the
	// same range. A band reader that accepted a hyphen in one package and an
	// en dash in the other would compile, pass, and silently stop comparing
	// half the fields in one record.
	"recordedBandForm",
	// The three levers, which are `const` and are why this list is no longer
	// named for state.
	//
	// # Why a lever is worth holding, which is sharper than it looks
	//
	// Both records' doc comments print the same recipe —
	// `GRMOB_BAND_VERDICT=required go test …` — and each package reads it
	// through a constant of its own. Nothing compared the two strings. A
	// `GRMOB_BANDVERDICT` in one of them would compile, pass every test, and
	// leave a reader following the documented command with a verdict from one
	// package and SILENCE from the other.
	//
	// Silence is the part that matters: it is indistinguishable from a pass.
	// The verdict line simply does not print, and the records these levers
	// belong to are the ones that have now had four floors found stale
	// between them. Same for the figure the placement arm is handed — a lever
	// nobody's shell spells the way the code reads it is an arm nobody ever
	// runs.
	//
	// Measured when they were registered: 3 consts, 0 divergent, and the scan
	// that found them found 21 shared names across the two packages of which
	// 16 are identical and 5 differ for reasons each one states.
	"bandVerdictEnv",
	"bandVerdictAsked",
	"packageReadingEnv",
}

// allTwoCopyShapes is both lists, in the order a reader would read
// them: the functions, then the state they keep.
//
// Built rather than written out a third time, because a shape named in two
// places and not the third is precisely the drift these lists exist to catch,
// arriving in the check itself.
func allTwoCopyShapes() []string {
	all := make([]string, 0, len(twoCopyFunctionShapes)+len(twoCopyValueShapes))
	all = append(all, twoCopyFunctionShapes...)
	all = append(all, twoCopyValueShapes...)
	return all
}

// How many packages the copy argument covers.
//
// Defined AS timingsRecordCopies rather than as 2, and the reason is that this
// comment used to say "it is the same constant as timingsRecordCopies" while
// being a literal. A sentence claiming two numbers are one number, beside a
// second number, is the copy this repository spends most of its censuses on,
// arriving in the constant that exists to bound a duplication.
//
// coresNoteScanDirs got this right two files over and says why: "a bare 2 here
// would be a second number to keep in step by hand", and what the separate
// NAME buys is a place for the second answer when somebody decides the two
// should part. That argument applies here unchanged.
//
// The reasoning itself is the record's: two separate `package main` programs is
// a copy with a written reason, and a third is a shape being kept in step
// across three places by whoever remembers to.
//
// The difference from the timings record is what happens at two, and it is the
// point of this pair of numbers sitting beside each other. The record's copy
// was held only to being the same SHAPE — five fields and an arm — because the
// values are different numbers about different machines and were never meant
// to match. These are code, so the copies are held to being IDENTICAL, which
// is the stronger thing and the cheaper one: there is nothing to decide about
// whether a difference is meant.
const twoCopyPackages = timingsRecordCopies

// checkTwoCopyDecls holds every copy of the shapes above to being the
// same declaration, and every package that has one to having all of them.
//
// # What is compared, and why it is not the bytes
//
// The declaration is printed with go/printer and its doc comment removed. That
// makes the comparison exactly about the code: the two headers SHOULD differ —
// one says what the function is and the other says why it is a second copy —
// and interior comments and whitespace are normalised by the printer rather
// than argued about.
//
// The set is both lists — the functions and the package-level state they keep
// — because the compiler's hold on the state is only its NAME, and a copy that
// declares that name as a different kind of thing compiles in both places and
// differs under t.Parallel(). See twoCopyValueShapes.
//
// What it cannot see is a difference in something the copies both call. That
// is the reason the set is the unit: a `packageBase` that had drifted is a
// `packageBase` this compares, and a helper outside the set would have to be
// added to it.
//
// # A comparison that could not be made is not a difference
//
// A declaration go/printer rejects used to come back as the string
// `unprintable: …`, which compares unequal to the other copy and produced a
// drift finding naming a file — the right file, for the wrong reason, with a
// message sending a reader to look for a difference that is not there. It is
// reported as itself instead, and the comparison for that shape is skipped
// rather than made against nothing.
func checkTwoCopyDecls(t *testing.T, decls []twoCopyDecl) {
	t.Helper()

	byName := map[string][]twoCopyDecl{}
	byDir := map[string]map[string]bool{}
	for _, d := range decls {
		// The declarations recorded as differing on purpose ride the same
		// walk and are not shapes — see twoCopyDecl.excused. They are read by
		// checkSharedNamesAreAccountedFor, which holds them to still
		// differing; counting them here would make every `package main` in
		// the repository a package that declares some of the two-copy shapes.
		if d.excused {
			continue
		}
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
	shapes := allTwoCopyShapes()
	var missing []string
	for _, name := range shapes {
		if len(byName[name]) == 0 {
			missing = append(missing, name)
		}
	}
	if len(missing) == len(shapes) {
		t.Errorf("none of the %d two-copy shape(s) was found in the "+
			"repository: %s.\n\n"+
			"These are what every census that asks \"is this a call into "+
			"package P\" resolves the qualifier with, and this repository has "+
			"two copies of them — wasm/verify/importnames_test.go and "+
			"internal/themehistory/importnames_test.go. Either the walk is "+
			"not reaching them or they have been renamed, and in both cases "+
			"the two copies are back to being kept in step by whoever "+
			"remembers to.",
			len(shapes), strings.Join(shapes, ", "))
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
		for _, name := range shapes {
			if !byDir[dir][name] {
				absent = append(absent, name)
			}
		}
		if len(absent) == 0 {
			continue
		}
		// Counted as the shapes this directory has, not as everything the
		// walk collected for it: the walk also collects the names recorded as
		// differing on purpose, and len(byDir[dir]) would count those too —
		// a numerator about a different set from its denominator.
		hasShapes := 0
		for _, name := range shapes {
			if byDir[dir][name] {
				hasShapes++
			}
		}
		t.Errorf("%s declares %d of the %d two-copy shapes and not "+
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
			dir, hasShapes, len(shapes),
			strings.Join(absent, ", "))
	}

	// And the copies themselves being the same declaration.
	for _, name := range shapes {
		found := byName[name]
		sort.Slice(found, func(i, j int) bool { return found[i].rel < found[j].rel })
		// A declaration the printer could not render is its own finding, and
		// it is reported before the comparison rather than inside it: the
		// text for such a declaration is "", which compares unequal to every
		// real one and would report a drift that is not there.
		unprintable := false
		for _, d := range found {
			if d.printErr == nil {
				continue
			}
			unprintable = true
			t.Errorf("%s at %s:%d could not be rendered by go/printer: %v.\n\n"+
				"This is not a difference between the copies — it is the "+
				"comparison not being MADE. Every other copy of %s is "+
				"therefore unchecked on this run, which is the state all of "+
				"them were in before anything compared them.\n\n"+
				"The printer failing on a declaration go/parser accepted is "+
				"either a syntax tree something has edited in place or a "+
				"toolchain fault; in both cases the finding is about this "+
				"check and not about the copy.",
				d.name, d.rel, d.line, d.printErr, d.name)
		}
		if unprintable {
			continue
		}
		for i := 1; i < len(found); i++ {
			if found[i].text == found[0].text {
				continue
			}
			t.Errorf("%s is declared at %s:%d and at %s:%d and the two are "+
				"not the same declaration.\n\n"+
				"These are a copy on purpose — see the header of either file "+
				"— and a copy that has drifted is the worst of both: two "+
				"packages answering the same question differently, with the "+
				"reasoning written down once. The comparison is of the code "+
				"alone, with doc comments removed and the printer's "+
				"formatting, so the two headers are free to say different "+
				"things and this is a real difference in what the "+
				"declaration is.\n\n%s:%d has:\n\n%s\n\n%s:%d has:\n\n%s\n\n"+
				"Make one a copy of the other, or — if the two packages now "+
				"need different answers — say so where the copy argument is "+
				"and take this shape out of twoCopyFunctionShapes or "+
				"twoCopyValueShapes.",
				name, found[0].rel, found[0].line, found[i].rel, found[i].line,
				found[0].rel, found[0].line, found[0].text,
				found[i].rel, found[i].line, found[i].text)
		}
	}

	// The trigger, which is the same one the timings record has and for the
	// same reason.
	whole := 0
	for _, dir := range dirs {
		if len(byDir[dir]) == len(shapes) {
			whole++
		}
	}
	// Both directions, because the count can be wrong in two ways and only one
	// of them was being asked about.
	//
	// Upwards is the trade changing: a third package means a shape kept in step
	// across three places. Downwards is the copies quietly becoming ONE — and
	// that is the direction nothing could see. A package that drops the whole
	// set leaves the other one complete, the per-directory arm above with
	// nothing to report, and this trigger under its limit; the comparison then
	// runs over a single declaration and passes, which is exactly what a census
	// over nothing looks like from the outside.
	//
	// It is reachable without a compile error: these helpers are used by the
	// censuses in the file that declares them, so deleting the file and its
	// callers together builds fine and halves the copies. The arm for "none of
	// them anywhere" is above; this is the arm for "all of them, in one place".
	if whole < twoCopyPackages {
		t.Errorf("%d package(s) declare the whole set of %d two-copy shapes "+
			"and the copy argument is written for %d: %s.\n\n"+
			"Fewer copies than the argument covers is not a saving, it is the "+
			"check going quiet: every shape is still found, every comparison "+
			"still runs, and a comparison of one declaration against nothing "+
			"passes. The two copies exist because two `package main` programs "+
			"cannot import each other's tests — if that has stopped being "+
			"true, the shapes belong in a package both import and these lists "+
			"belong deleted, which is a better outcome than this check and is "+
			"the one thing it cannot tell you has happened.\n\n"+
			"Either restore the copy, or lower twoCopyPackages with the reason "+
			"beside the copy argument.",
			whole, len(shapes), twoCopyPackages, strings.Join(dirs, ", "))
	}
	if whole > twoCopyPackages {
		t.Errorf("%d package(s) declare the two-copy shapes and the "+
			"argument for copying them was written for %d: %s.\n\n"+
			"That argument is about two separate `package main` programs, one "+
			"under wasm/ and one under internal/, and the cost of a package "+
			"existing so that %d helpers could be %d helpers. At %d it is no "+
			"longer that trade: it is a shape kept in step across %d places by "+
			"a check, which works, and by whoever remembers to update the "+
			"third when the reasoning changes, which does not.\n\n"+
			"Extract them into a package all three can import, or raise "+
			"twoCopyPackages with the reason beside the copy argument, so "+
			"the next person reads a decision rather than a number.",
			whole, twoCopyPackages, strings.Join(dirs, ", "),
			len(shapes), len(shapes), whole, whole)
	}

	// "%d package(s) declare the whole set" and not "%d copies each": on a
	// run where one package is short a shape, the count is 1 and the old
	// wording read "1 copy each, held identical across" both of them — a
	// summary contradicting the finding printed above it.
	t.Logf("%d shape(s) kept in two copies — %d function(s) and %d value "+
		"declaration(s) — %d package(s) declaring the whole set, held "+
		"identical, across %s.",
		len(shapes), len(twoCopyFunctionShapes), len(twoCopyValueShapes),
		whole, strings.Join(dirs, ", "))
}

// namesDeclaredBy is what one top-level declaration declares, as names.
//
// Not `packageLevelNames`, which this package already has: that one scans this
// directory for redeclarations of predeclared identifiers, and it is a
// different question with a name that would fit either. This is about one
// declaration; that is about one package.
//
// Methods are left out: a method is a name on a type rather than a name in the
// package, and two packages declaring `func (x foo) String()` on their own
// `foo` types are not keeping a copy of anything. Imports are left out for the
// same reason one level along — an import spec names a package, not a
// declaration of this one.
//
// Types and funcs give one name each, a value spec gives all of its, and that
// is the whole of what the inversion needs: it asks which names exist in both
// packages, and leaves what they ARE to the comparison that already runs.
func namesDeclaredBy(d ast.Decl) []string {
	switch decl := d.(type) {
	case *ast.FuncDecl:
		if decl.Recv != nil {
			return nil
		}
		return []string{decl.Name.Name}
	case *ast.GenDecl:
		var out []string
		for _, sp := range decl.Specs {
			switch spec := sp.(type) {
			case *ast.ValueSpec:
				for _, n := range spec.Names {
					out = append(out, n.Name)
				}
			case *ast.TypeSpec:
				out = append(out, spec.Name.Name)
			}
		}
		return out
	}
	return nil
}

// The names both record packages declare and which are NOT copies of each
// other, with the reason for each.
//
// # Why this list exists, which is the measurement that prompted it
//
// The two-copy lists above are an INCLUSION list: a shape is held identical
// because somebody registered it. That arrangement can only fail one way, and
// it is the way nothing can see — a declaration that arrives in both packages
// and is registered in neither is simply unheld, and reads exactly like a
// declaration nobody has drifted yet.
//
// Measured: of **21 names declared in both packages, 16 were identical and 4
// of those 16 were unregistered**, three of them for several sessions with the
// argument for registering them already written down twice. The only thing
// that found them was a session happening to scan for them.
//
// So the question is asked the other way round as well:
// checkSharedNamesAreAccountedFor holds every shared name to being in one of
// the two-copy lists or in this one. A name in neither is a finding that names
// both declarations and asks for a decision — register it, or say here why the
// two differ.
//
// # Why the reasons are values and not a comment
//
// Each entry is read back in the finding that reports a stale exemption, so a
// reason written here is a reason a reader is handed rather than one they have
// to come and find. And the reasons are all the same SHAPE, which is worth
// seeing in one place: every one of these differs because it reads something
// that belongs to its own package — its own record, its own note, its own
// program.
var twoCopyNamesThatDiffer = map[string]string{
	"main": "two different programs. wasm/verify generates a transcript for " +
		"the browser pass; internal/themehistory walks this repository's " +
		"theme history. They share a name because Go requires it of a " +
		"`package main`, and nothing else",
	timingsArm: "each reports on its own record, which is an anonymous " +
		"struct literal in its own package — see timingsRecordCopies for why " +
		"the records are two copies rather than one type",
	timingsCoresNote: "each says what a core count is worth in ITS package, " +
		"and the two answers are measured and different: a git process per " +
		"commit improving to eight workers in one, one program's own " +
		"goroutines flat from two cores up in the other. Two notes saying " +
		"the same thing would mean one of them was not measured",
	"recordMachineDiffers": "each compares this machine against its own " +
		"record, for the same reason the arms do. A shared function would " +
		"need a shared type and there is none — which is the copy argument " +
		"itself, one level down",
	"TestTheFigureGoTestPrintedForThisPackageIsPlacedInItsBand": "each " +
		"places its own record's headline field — wholeFile against " +
		"wholePackage — and names its own package in the recipe it prints " +
		"when the lever is unset. The sentence they both build out of that " +
		"reading IS held identical: see againstBandGiven",
}

// checkSharedNamesAreAccountedFor holds every name both record packages
// declare to being either a shape held identical or a difference with a reason.
//
// # The three directions
//
// A shared name in neither list is the gap this closes: unheld, and
// indistinguishable from held until the day the two answers diverge.
//
// An exemption that names a shape which is NOT shared is an exemption for
// something that has been renamed or moved, and it silences nothing any more —
// the same rot the cores note's second direction exists for.
//
// An exemption whose two declarations turn out to be IDENTICAL is the
// interesting one: the reason written here has stopped being true, and the
// shape is a copy nobody is holding. That is read off the comparison the
// census above already makes, which is why the differing names are collected
// by the same walk — there is one reading of "are these the same declaration"
// in this repository and this is not a second one.
func checkSharedNamesAreAccountedFor(t *testing.T, recordDirs []string,
	declaredIn map[string]map[string]bool, decls []twoCopyDecl) {

	t.Helper()
	// The subject is the pair of packages carrying a timings record. With
	// fewer than two there is nothing to share, and the records question's own
	// Fatalf is where that is reported — saying it again here would be one
	// fact twice.
	if len(recordDirs) < 2 {
		return
	}
	held := map[string]bool{}
	for _, name := range allTwoCopyShapes() {
		held[name] = true
	}
	// The text of the EXCUSED declarations only. The held ones are compared by
	// checkTwoCopyDecls and nothing here needs to repeat that; what this needs
	// is the text of the exemptions, to say whether an exemption still exempts
	// anything.
	text := map[string]map[string]string{}
	for _, d := range decls {
		if !d.excused {
			continue
		}
		if text[d.name] == nil {
			text[d.name] = map[string]string{}
		}
		text[d.name][d.dir] = d.text
	}

	// A name in a shape list AND in the exemption list, which is a
	// contradiction rather than a drift: one says hold these identical and the
	// other says they differ on purpose. Reported before anything else,
	// because the walk resolves it in favour of HELD and that choice is this
	// check's to explain rather than the walk's to make quietly.
	for _, name := range keysOf(twoCopyNamesThatDiffer) {
		if !held[name] {
			continue
		}
		t.Errorf("%s is in a two-copy shape list and in "+
			"twoCopyNamesThatDiffer.\n\n"+
			"The first says the two packages must declare it identically; the "+
			"second says they differ on purpose, for this reason: %q.\n\n"+
			"Both cannot be true. The walk treats it as held, so the "+
			"comparison is being made and the exemption is doing nothing — "+
			"but which of the two somebody meant is not readable from here. "+
			"Take it out of one of them.",
			name, twoCopyNamesThatDiffer[name])
	}

	var shared, unaccounted []string
	for name, dirs := range declaredIn {
		in := 0
		for _, dir := range recordDirs {
			if dirs[dir] {
				in++
			}
		}
		if in < 2 {
			continue
		}
		shared = append(shared, name)
		if held[name] {
			continue
		}
		if _, excused := twoCopyNamesThatDiffer[name]; excused {
			continue
		}
		unaccounted = append(unaccounted, name)
	}
	sort.Strings(shared)
	sort.Strings(unaccounted)

	// The walk reaching anything. These two packages share a couple of dozen
	// names and always have; zero is this reading being broken rather than a
	// repository that has stopped duplicating.
	if len(shared) == 0 {
		t.Errorf("no name is declared in both %s, and they share the "+
			"import-resolving helpers, the band reader and the verdict "+
			"levers.\n\n"+
			"Either the walk is not collecting package-level names or the two "+
			"packages no longer exist under these paths, and in both cases "+
			"this check says nothing while passing.",
			strings.Join(recordDirs, " and "))
		return
	}

	for _, name := range unaccounted {
		t.Errorf("%s is declared in both %s and is in neither two-copy "+
			"list.\n\n"+
			"A name in both packages is either a copy kept in step on purpose "+
			"or two declarations that differ on purpose, and which one it is "+
			"cannot be read off the source. Unregistered, it is simply "+
			"unheld: the two can drift and every test in this repository "+
			"passes.\n\n"+
			"That is not hypothetical. Four names were in exactly this state "+
			"when this check was written — the three `GRMOB_` levers and the "+
			"function that reads them — and three of them had been for "+
			"several sessions, with the argument for holding them already "+
			"written twice.\n\n"+
			"Add it to twoCopyFunctionShapes or twoCopyValueShapes if the two "+
			"are meant to be the same declaration, or to "+
			"twoCopyNamesThatDiffer with the reason if they are not.",
			name, strings.Join(recordDirs, " and "))
	}

	for _, name := range keysOf(twoCopyNamesThatDiffer) {
		// A name the lists contradict each other about has been reported
		// above, and the walk resolved it as held — so it has no excused text
		// and the arms below would report that as a collection failure. One
		// fact, one finding.
		if held[name] {
			continue
		}
		dirs := declaredIn[name]
		in := 0
		for _, dir := range recordDirs {
			if dirs[dir] {
				in++
			}
		}
		if in < 2 {
			t.Errorf("twoCopyNamesThatDiffer has an entry for %s and it is "+
				"not declared in both %s.\n\n"+
				"The reason it carries is %q, which is about a pair that no "+
				"longer exists: one of the two has been renamed, moved or "+
				"deleted. An exemption for a shape that is not shared "+
				"silences nothing and reads as a decision somebody made "+
				"about the code that is there.",
				name, strings.Join(recordDirs, " and "),
				twoCopyNamesThatDiffer[name])
			continue
		}
		if len(text[name]) < 2 {
			// Not collected for comparison, which means the name is in this
			// list and not in the walk's set. Reported rather than skipped:
			// the direction below is the one that finds a stale reason, and
			// it cannot run on a shape nothing rendered.
			t.Errorf("twoCopyNamesThatDiffer has an entry for %s and the walk "+
				"collected %d of its declarations rather than 2.\n\n"+
				"The names in this list are collected the same way the held "+
				"shapes are, so that an exemption whose two declarations have "+
				"become identical can be reported. One that is not collected "+
				"is an exemption nothing checks — see "+
				"twoCopyDeclarationsIn, which reads both lists.",
				name, len(text[name]))
			continue
		}
		same := ""
		for dir, body := range text[name] {
			if same == "" {
				same = body
				continue
			}
			if body != same {
				same = ""
				break
			}
			_ = dir
		}
		if same == "" {
			continue
		}
		t.Errorf("twoCopyNamesThatDiffer says %s differs between the two "+
			"packages and the two declarations are identical.\n\n"+
			"The reason recorded is %q.\n\n"+
			"So either that reason has stopped being true — in which case "+
			"this is a copy nothing is holding, and it belongs in "+
			"twoCopyFunctionShapes or twoCopyValueShapes — or the two have "+
			"been made the same by accident, which is the same finding from "+
			"the other side. An exemption that exempts nothing is worse than "+
			"no exemption: it reads as a decision and it holds a shape out of "+
			"the one check that would notice it drifting.",
			name, twoCopyNamesThatDiffer[name])
	}

	t.Logf("%d name(s) declared in both %s: %d held identical, %d recorded as "+
		"differing with a reason.", len(shared),
		strings.Join(recordDirs, " and "),
		len(shared)-len(twoCopyNamesThatDiffer), len(twoCopyNamesThatDiffer))
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
//	                        nothing. WHICH of the two is then settled against
//	                        the toolchain's own sources — see stdlibSource and
//	                        holdsAPackage
//	everything else         has to be this module or something go.mod
//	                        requires, by prefix. A path under a required
//	                        module might still be a directory that does not
//	                        exist, which only a build settles — but a path
//	                        under NO required module cannot be imported by
//	                        anything here, whatever is on disk
//
// # The standard-library half, which used to be a written limit
//
// It was this: no dot in the first element, therefore stdlib, therefore fine.
// `os/exex` passed, and the census asking about it reported nothing forever —
// which matters more here than anywhere, because five of the eight paths asked
// about today are stdlib, so the half that was exact covered three of them.
//
// What was written down as the only two ways to close it were `go list std` —
// a second copy of something that changes every release — and a build, which
// is the cost every walk here declines. There is a third, and it is cheaper
// than both: a standard library package is a DIRECTORY under the toolchain's
// own `$GOROOT/src`, and asking whether that directory holds a Go file is one
// read against the very toolchain this test is running on. Nothing is copied,
// nothing is loaded, and the answer moves with the release because it IS the
// release.
//
// It is a read and not a stat, and that is the difference between "the
// directory is there" and "there is a package in it": a path removed from the
// standard library can leave its directory behind, and a check that only
// stat'd would pass it. See holdsAPackage.
//
// The one thing it is still not is a guarantee that THIS build would accept
// those files — a directory whose Go files are all excluded by build tags
// holds a `.go` file and no package. That is the same residue the module half
// has and for the same reason, and it is a much smaller one than "no dot,
// therefore fine".
//
// When GOROOT is not on disk — a stripped container, a toolchain shipped
// without its sources — there is nothing to stat and the half goes back to
// being the rule it was. That is said out loud in the log rather than passed
// over, because a check that quietly stops asking is the shape this whole file
// is about.
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
	// Where this toolchain keeps the standard library's sources, or "" when
	// they are not on disk. Read once: it is a stat of one directory, and the
	// answer is the same for every path below.
	stdlib := stdlibSource()
	// Whether a directory under it holds a package, remembered. The loop
	// below is over ASKINGS and not over paths — six censuses asking about
	// `os/exec` is six times round it — and the finding stays per asking,
	// because each call site is a place somebody would fix. The READ does
	// not: what is in a directory does not change between two questions about
	// it in one run.
	holds := map[string]bool{}
	holdsAPackageAt := func(dir string) bool {
		if answer, done := holds[dir]; done {
			return answer
		}
		answer := holdsAPackage(dir)
		holds[dir] = answer
		return answer
	}
	seen := map[string]bool{}
	var paths []string
	// Counted in distinct PATHS and not in askings, because that is what the
	// line below lists: six censuses asking about `os/exec` is one path this
	// half either judged or did not.
	stdlibPaths := map[string]bool{}
	// And the ones this arm judged and refused, so that the line at the bottom
	// says what the run found rather than repeating a claim the run has just
	// contradicted. See the Logf.
	unimportable := map[string]bool{}
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
			// The standard library, by the module-path rule read backwards —
			// or a typo, which is what the stat settles.
			stdlibPaths[a.path] = true
			if stdlib == "" {
				continue
			}
			dir := filepath.Join(stdlib, filepath.FromSlash(a.path))
			if holdsAPackageAt(dir) {
				continue
			}
			unimportable[a.path] = true
			t.Errorf("%s:%d asks about %q, and this toolchain's standard "+
				"library has no such package: %s holds no Go file.\n\n"+
				"A path with no dot in its first element cannot be a module "+
				"path, so it is the standard library or it is nothing, and "+
				"this is which. A path that is nothing binds no identifier, "+
				"so importedAs comes back empty for every file, every call "+
				"into the package is invisible, and the census passes over "+
				"the whole repository having asked nothing.\n\n"+
				"That is the same silence an aliased import used to produce, "+
				"one level up — the path the question is about rather than "+
				"the name the answer is written with — and it is worth more "+
				"here than on the module half, because most of what these "+
				"censuses ask about is stdlib.\n\n"+
				"If the package has MOVED between releases, this is the "+
				"census being told so by the toolchain it is running on; if "+
				"it is a typo, the census has been reporting nothing since it "+
				"was written.", a.rel, a.line, a.path, dir)
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
				unimportable[a.path] = true
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
	// What the standard-library half was actually worth on this run, said out
	// loud. A check that has stopped asking reads exactly like one that asked
	// and found nothing, which is the whole subject of this file.
	// Counted as the ones that ANSWERED, not the ones that were asked: a path
	// this half judged and refused is reported above, and including it here
	// under "each a package" would be the line contradicting the finding two
	// lines up.
	stdOK := 0
	for path := range stdlibPaths {
		if !unimportable[path] {
			stdOK++
		}
	}
	std := fmt.Sprintf("%d of them in the standard library, each a package "+
		"under %s", stdOK, stdlib)
	if stdlib == "" {
		std = fmt.Sprintf("%d of them in the standard library, which this "+
			"run could not check: go/build reports GOROOT as %q and there is "+
			"no `src` directory there, so those paths are held only to the "+
			"rule that a module path has a dot in its first element",
			len(stdlibPaths), build.Default.GOROOT)
	}
	// The claim, made about the paths that actually earned it. This line used
	// to say "each one this module could import" on every run, including the
	// run that had just reported one it could not — the one sentence on it
	// that the findings above had contradicted. A count is a reading; "each
	// one" was an assertion, and the arm above is the thing entitled to make
	// it.
	judged := fmt.Sprintf("all %d of which this module could import", len(paths))
	if n := len(unimportable); n > 0 {
		refused := make([]string, 0, n)
		for path := range unimportable {
			refused = append(refused, path)
		}
		sort.Strings(refused)
		judged = fmt.Sprintf("%d of which this module could import and %d of "+
			"which it could not (%s — see the finding(s) above)",
			len(paths)-n, n, strings.Join(refused, ", "))
	}
	t.Logf("%d import path(s) asked about by the censuses here, %s; %s: %s.",
		len(paths), judged, std, strings.Join(paths, ", "))
}

// holdsAPackage is whether this directory is one the Go build could compile a
// package out of: it exists, and it has a `.go` file in it.
//
// # Why the stat was not enough
//
// The standard-library half of the check above turns a path into a directory
// under `$GOROOT/src` and asked whether that directory was there. That is
// exact for a typo, which is what it was written for, and it is loose about
// the case it did not consider: a directory with no Go file in it. A package
// removed from the standard library can leave its directory behind — a
// testdata tree, a README, an empty shell — and `os.Stat` says yes to all of
// them, which is the arm passing a path nothing can import.
//
// One `os.ReadDir` closes it and the cost is the same order as the stat it
// replaces: seven directories on a green run, read once each, stopping at the
// first `.go` file.
//
// # What it still does not settle
//
// Whether THIS build would accept those files. A directory whose Go files are
// all excluded by build tags — `//go:build ignore`, or a GOOS this run is not
// — holds a `.go` file and no package, and telling the two apart is
// go/build.Import, which loads. That is the cost every walk in this repository
// declines, and the residue left is much smaller than the directory it
// replaced: a path in the standard library whose only Go files are excluded
// everywhere is not a path a census here is going to be asking about.
func holdsAPackage(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") {
			return true
		}
	}
	return false
}

// stdlibSource is where this toolchain keeps the standard library's sources,
// or "" when they are not on disk.
//
// # Why go/build and not `go env GOROOT`
//
// go/build.Default is the same answer the toolchain gives itself — the GOROOT
// environment variable if it is set, and otherwise the path this binary's
// toolchain was built at — and reading it is a field access rather than a
// process. `go env` would be a subprocess in a package whose own timings
// record counts git processes as its largest term, to learn something already
// in memory.
//
// runtime.GOROOT() is the same value and is deprecated as of Go 1.24, which is
// the other reason this is spelled the way it is.
//
// # Why the stat, and what "" means to the caller
//
// GOROOT can name a directory that is not there: a toolchain installed without
// its sources, a container with `src` stripped, an environment variable
// pointing somewhere stale. A join against a path like that produces a stat
// that fails for every package in the standard library, which would report
// every stdlib path a census names as a typo — a census failing loudly about
// the machine it is on rather than about the repository.
//
// So the directory is checked once, and "" is the caller's signal to fall back
// to the rule this replaced rather than to report. The fallback is logged, not
// silent.
func stdlibSource() string {
	root := build.Default.GOROOT
	if root == "" {
		return ""
	}
	src := filepath.Join(root, "src")
	if info, err := os.Stat(src); err != nil || !info.IsDir() {
		return ""
	}
	return src
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

// twoCopyDecl is one declaration of a shared shape that the walk found.
type twoCopyDecl struct {
	name string
	// The directory, which is the unit a copy is counted in: these are
	// package-level functions and a package is a directory here.
	dir  string
	rel  string
	line int
	// The declaration printed with its doc comment removed — see
	// declarationText. "" when the printer could not render it, in which case
	// `printErr` says so and the comparison is reported as one that could not
	// be made rather than as a difference.
	text     string
	printErr error
	// Whether this declaration was collected because it is recorded as
	// DIFFERING between the two packages rather than because it is a shape
	// held identical — see twoCopyNamesThatDiffer.
	//
	// One walk collects both, so there is one reading of what a declaration
	// is, and the flag is what keeps the two questions apart: an excused
	// declaration is not a two-copy shape and must not be counted as one.
	// Leaving it out cost a run: `main` is an excused name, every `package
	// main` in the repository declares it, and the arm that holds a package
	// declaring SOME of the shapes to declaring all of them reported every
	// one of them.
	excused bool
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

// declarationText is a declaration as code, with its doc comment removed.
//
// The doc comment is the part two copies are SUPPOSED to differ in — one
// header says what the function is, the other says why there is a second one
// — and the printer normalises everything else, so what comes back is exactly
// the thing a copy has to keep in step.
//
// # The two shapes, and why a var is printed from its spec
//
// A function is an *ast.FuncDecl and prints whole. The state is a ValueSpec
// inside a GenDecl, and printing the GenDecl would compare a `var (…)` block
// against a single `var` line — a difference in how the file is arranged
// rather than in what is declared. So the SPEC is printed and `var ` is put in
// front of it, which renders `dotImportsReported sync.Map` the same way
// wherever it sits.
//
// # Why the error comes back rather than travelling as the text
//
// It used to be returned as `unprintable: …`, which compares unequal to the
// other copy and produces a drift finding naming a file — right by accident
// and wrong in what it says. The real finding is that the comparison could not
// be MADE, and a reader told that two functions differ will go looking for a
// difference that is not there.
func declarationText(fset *token.FileSet, node ast.Node,
	keyword string) (string, error) {

	var buf bytes.Buffer
	switch decl := node.(type) {
	case *ast.FuncDecl:
		stripped := *decl
		stripped.Doc = nil
		if err := printer.Fprint(&buf, fset, &stripped); err != nil {
			return "", err
		}
		return buf.String(), nil
	case *ast.ValueSpec:
		stripped := *decl
		stripped.Doc = nil
		// The trailing `// …` on the same line goes too: it is a comment, and
		// the two copies are free to differ in those.
		stripped.Comment = nil
		if err := printer.Fprint(&buf, fset, &stripped); err != nil {
			return "", err
		}
		// The keyword comes from the GenDecl and not from here, because a
		// ValueSpec does not carry one: the printer renders `x = "y"` for both
		// a var and a const. It is prefixed at all so that the text a reader
		// is shown in a finding is a declaration they can search for, and it
		// is the REAL keyword so that a `const` in one package and a `var` of
		// the same name and value in the other is a difference this reports
		// rather than one it renders away.
		return keyword + " " + buf.String(), nil
	}
	return "", fmt.Errorf("a %T is not a declaration this compares", node)
}

// twoCopyDeclarationsIn collects the shared shapes this file declares,
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
func twoCopyDeclarationsIn(fset *token.FileSet, rel string,
	file *ast.File) ([]twoCopyDecl, []importPathAsk) {

	// Two memberships, and they are not the same question. `held` is what the
	// lists say must be IDENTICAL; `shape` and `values` are what this walk
	// collects, which is that plus the names recorded as differing on purpose.
	// Conflating them made every excused name read as a held one.
	held := map[string]bool{}
	shape := map[string]bool{}
	for _, name := range twoCopyFunctionShapes {
		shape[name] = true
		held[name] = true
	}
	values := map[string]bool{}
	for _, name := range twoCopyValueShapes {
		values[name] = true
		held[name] = true
	}
	// And the names recorded as DIFFERING on purpose, collected the same way.
	//
	// They are not held to being identical — that is what the exemption means
	// — but their text is needed for the direction that reports an exemption
	// whose two declarations have BECOME identical. Collected here rather than
	// by a second walk so that there is one reading of what a declaration is
	// in this repository, and the check that holds shapes together and the
	// check that holds exemptions apart are reading the same text.
	//
	// A differing name can be a func or a value, so both sets get it: these
	// are membership tests, and a name in the wrong one simply never matches.
	for name := range twoCopyNamesThatDiffer {
		shape[name] = true
		values[name] = true
	}
	dir := path.Dir(rel)
	// One declaration of a shared shape, however it is spelled. The two arms
	// below differ only in what they hand the printer.
	record := func(name string, at token.Pos, node ast.Node,
		keyword string) twoCopyDecl {

		text, err := declarationText(fset, node, keyword)
		_, excused := twoCopyNamesThatDiffer[name]
		return twoCopyDecl{
			name:     name,
			dir:      dir,
			rel:      rel,
			line:     fset.Position(at).Line,
			text:     text,
			printErr: err,
			// A name in a shape list is HELD, whatever else it is in. The
			// contradiction — a name in both lists — is reported by
			// checkSharedNamesAreAccountedFor rather than resolved silently
			// here, because which list the author meant is not something
			// this can know.
			excused: excused && !held[name],
		}
	}
	var decls []twoCopyDecl
	var asks []importPathAsk
	for _, d := range file.Decls {
		// The package-level state, which is a ValueSpec inside a GenDecl
		// rather than a declaration of its own — see twoCopyValueShapes
		// for why it is held to the same rule as the code that reads it.
		if gen, ok := d.(*ast.GenDecl); ok {
			// `const` as well as `var`, which is the change that let the
			// levers be held. The walk read VAR only, so registering a
			// constant reported it MISSING from both packages — a list entry
			// that fails rather than holds, which is why three levers sat
			// unregistered while the argument for registering them had
			// already been written twice.
			if gen.Tok != token.VAR && gen.Tok != token.CONST {
				continue
			}
			for _, sp := range gen.Specs {
				vs, ok := sp.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, n := range vs.Names {
					if !values[n.Name] {
						continue
					}
					decls = append(decls,
						record(n.Name, n.Pos(), vs, gen.Tok.String()))
				}
			}
			continue
		}
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		if fn.Recv == nil && shape[fn.Name.Name] {
			decls = append(decls,
				record(fn.Name.Name, fn.Pos(), fn, "func"))
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
