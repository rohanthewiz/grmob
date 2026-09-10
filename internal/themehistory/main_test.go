package main

import (
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/internal/themeleaves"
)

// The revision walk and the working-tree walk read the same files.
//
// # The half the other arm cannot see
//
// wasm/verify's TestTheHistoryWalkersExpansionIsTheOneThisFileMeasures holds
// themeleaves.InDir("../../core", "Theme") against affordedLeafNames() — the
// parsed expansion against the reflective one, at HEAD, on every run. That is
// the half of this command's reading that does not need git.
//
// The half it cannot see is the half above themeleaves: leavesAt fetches a
// revision's files with `ls-tree` and `cat-file` and hands them over as a map,
// and which files end up in that map is decided HERE — by a `.go` suffix test
// and a `_test.go` exclusion that now exist in this file as well as in
// themeleaves.Of. Two copies of one rule, and the way copies of a rule fail is
// quiet: a filter that drifted would give every revision a different
// population, the edit-size table would be a reading of that population, and
// the arm at HEAD would stay green because it never goes through leavesAt at
// all.
//
//	go run ./internal/themehistory
//	          │
//	          ├── leavesAt(sha) ── ls-tree ── cat-file ── filter  ← only here
//	          │                                             │
//	          └───────────────────────────────────── themeleaves.Of
//	                                                        │
//	   wasm/verify ── themeleaves.InDir ── ReadDir ── filter ┘  ← and here
//
// So the two filters are compared at the one revision both mechanisms can
// read: HEAD, against the working tree it was checked out into.
//
// # Why this one is allowed to skip
//
// The convention everywhere else in this repository is that a skipping test is
// a test that reports nothing, and the fix is to make it not need what it is
// missing. This is the exception the rule is stated against: what is being
// checked IS the git half, so a checkout with no history — a shallow clone, a
// source tarball, a build container — has nothing for it to be wrong about.
// The skip says which of those it is rather than reporting a pass.
//
// A working tree that differs from HEAD under core/ is the other skip, and it
// is not a shortcoming either: the two readings would then be over two
// different trees, and a difference between them would be the uncommitted edit
// rather than the filters. That is a fact about the checkout and the message
// says so.
func TestTheRevisionsFileSetIsTheOneTheWorkingTreeWalkReads(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git on this machine, so the revision half of this command " +
			"cannot be run and its file filter cannot be compared with " +
			"themeleaves.InDir's. The EXPANSION under both is held against " +
			"reflect on every run by wasm/verify's " +
			"TestTheHistoryWalkersExpansionIsTheOneThisFileMeasures.")
	}
	// The pathspecs below (`-- core`) are resolved against the process's
	// working directory, which under `go test` is this package's directory and
	// not the repository root. Everything this file does is about core/, so
	// the whole test runs from the root; t.Chdir puts it back afterwards.
	root, err := git("rev-parse", "--show-toplevel")
	if err != nil {
		t.Skipf("this checkout is not a git repository, so there is no revision "+
			"to read %s/ at: %v", themePkg, err)
	}
	t.Chdir(strings.TrimSpace(root))

	// Whether git has anything tracked under core/ at all. A vendored copy of
	// this module inside another repository would answer every command above
	// without holding this directory, and an empty answer there is "not this
	// repository" rather than "the filters disagree by 96 files".
	tracked, err := git("ls-tree", "-r", "--name-only", "HEAD", "--", themePkg)
	if err != nil {
		t.Skipf("HEAD cannot be read, which is a repository with no commits in "+
			"it (or a detached state git will not resolve): %v", err)
	}
	if strings.TrimSpace(tracked) == "" {
		t.Skipf("git tracks no files under %s/ at HEAD, so this is not the "+
			"repository %s.%s is declared in — a vendored copy, or a module "+
			"extracted into another tree. There is nothing here for the revision "+
			"walk to be wrong about.", themePkg, themePkg, themeType)
	}

	// And whether the two readings would be over the same tree. --porcelain
	// lists staged, unstaged and untracked paths alike, which is what is
	// wanted: an uncommitted new file in core/ is exactly the difference that
	// would otherwise read as a filter having drifted.
	dirty, err := git("status", "--porcelain", "--", themePkg)
	if err != nil {
		t.Skipf("git cannot report the state of %s/: %v", themePkg, err)
	}
	if strings.TrimSpace(dirty) != "" {
		t.Skipf("%s/ differs from HEAD in this working tree:\n%s\n"+
			"The revision walk reads HEAD and themeleaves.InDir reads the "+
			"directory, so the two file sets are expected to differ here and the "+
			"difference would be the uncommitted edit rather than the filters "+
			"this test is about. Commit or stash and run it again.",
			themePkg, strings.TrimSpace(dirty))
	}

	// The two readings. Both go through themeleaves.Of in the end, and what
	// differs is how the sources reached it.
	atHead, err := leavesAt("HEAD")
	if err != nil {
		t.Fatalf("HEAD's sources cannot be fetched out of git: %v\n\n"+
			"Every row of the edit-size table under affordedBandSteps is this "+
			"call at another revision, so a failure here is the whole reading "+
			"and not one commit of it.", err)
	}
	inTree, err := themeleaves.InDir(themePkg, themeType)
	if err != nil {
		t.Fatalf("%s/ cannot be read off disk: %v", themePkg, err)
	}

	// Compared by base name, because the two carry different prefixes for the
	// same file — git's paths are repository-relative ("core/theme.go") and
	// InDir's are joined onto the directory it was given. The file NAMES are
	// what both filters decide on.
	headFiles := baseNames(atHead.Files)
	treeFiles := baseNames(inTree.Files)
	if !slices.Equal(headFiles, treeFiles) {
		t.Errorf("the revision walk reads %d file(s) under %s/ at HEAD and "+
			"themeleaves.InDir reads %d in the working tree.\n\n"+
			"only the revision walk: %s\nonly the working tree:  %s\n\n"+
			"The trees are the same — this test skips when they are not — so this "+
			"is the two file filters having come apart. leavesAt drops test files "+
			"before fetching them (a `git cat-file` per test file per revision is "+
			"real time spent on text nobody parses) and themeleaves.Of drops them "+
			"again; InDir does its own directory read with the same rule. Whichever "+
			"of the three moved, every row of the edit-size table is now over a "+
			"different population than the arm at HEAD checks, and that arm cannot "+
			"see it: it never goes through leavesAt.\n\n"+
			"A name only the REVISION walk has is usually ls-tree's -r reaching "+
			"into a subdirectory of %s/, which InDir does not descend into — see "+
			"its note on why it is not recursive.",
			len(headFiles), themePkg, len(treeFiles),
			nameList(missing(headFiles, treeFiles)),
			nameList(missing(treeFiles, headFiles)), themePkg)
		// Everything below reads the expansions those files produced, and with
		// the file sets apart every one of them fails as a consequence of this:
		// a missing file is a missing struct is a missing leaf. The finding is
		// complete here and the rest would be its echo.
		return
	}

	// And that the same files gave the same answer. Redundant while the file
	// sets agree — Of is one function and the text is the same text — which is
	// the point: it costs nothing and it is the assertion that would survive
	// somebody giving leavesAt a second parse.
	if !slices.Equal(atHead.Names, inTree.Names) {
		t.Errorf("%s.%s expands to %d leaf name(s) out of HEAD and %d out of the "+
			"same %d file(s).\n\n"+
			"only HEAD's:        %s\nonly the tree's:    %s\n\n"+
			"Both readings are themeleaves.Of over the same names and the same "+
			"text, so this is not a filter: it is the two having been handed "+
			"different CONTENT for a file they both hold. `git cat-file` returning "+
			"something other than what is on disk for an unmodified path is a "+
			"checkout with filters or line-ending conversion configured, and every "+
			"revision in the table was parsed through the same conversion.",
			themePkg, themeType, len(atHead.Names), len(inTree.Names),
			len(headFiles),
			nameList(missing(atHead.Names, inTree.Names)),
			nameList(missing(inTree.Names, atHead.Names)))
	}
	if !atHead.Found {
		t.Errorf("no struct named %s is declared in %s/ at HEAD.\n\n"+
			"The working-tree walk %s find one, so this is not the type having "+
			"been renamed — it is the revision walk coming back empty, which is "+
			"the failure that looks like a quiet history rather than like an "+
			"error: every row of the table would read as a creation.",
			themeType, themePkg,
			map[bool]string{true: "does", false: "does not"}[inTree.Found])
	}
	if len(atHead.Unparsed) > 0 {
		t.Errorf("go/parser returned nothing for %s at HEAD.\n\n"+
			"In a HISTORY that is ordinary — a revision caught mid-refactor — and "+
			"the command says so on stderr and carries on. At HEAD it means this "+
			"commit does not parse, and the population every other revision is "+
			"diffed against is short by whatever those files declare.",
			strings.Join(atHead.Unparsed, ", "))
	}

	// Only on a run where every arm above held. A sentence reporting agreement
	// under a list of failures is a sentence that describes a state this run
	// was never in — the counts in it are the ones the failures are about.
	if t.Failed() {
		return
	}
	t.Logf("HEAD and the working tree hand themeleaves the same %d file(s) "+
		"under %s/, which expand to the same %d leaf name(s). That is the half "+
		"of this command's reading wasm/verify cannot check: it holds the "+
		"EXPANSION against reflect, and the expansion is downstream of the file "+
		"filter this test compares. The two filters are `ls-tree` plus leavesAt's "+
		"own suffix test on one side and themeleaves.InDir's directory read on "+
		"the other.", len(headFiles), themePkg, len(atHead.Names))
}

// baseNames is a list of paths as the two walks can be compared: the file name
// alone, sorted.
//
// Sorted here rather than relied on: themeleaves.Files comes out sorted by
// PATH, and two different prefixes can sort their shared base names into two
// different orders ("core/a.go" before "core/b.go" is also "a.go" before
// "b.go", but that is a property of these prefixes and not of any two).
func baseNames(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		out = append(out, filepath.Base(p))
	}
	slices.Sort(out)
	return out
}

// missing is the members of one sorted list the other does not hold.
//
// Both directions are wanted and each reads as a different finding — see the
// failures above, which say what each side means — so the difference is taken
// twice rather than returned as a pair. The same shape as wasm/verify's
// affordedNamesNotIn, which is that file's half of the same comparison.
func missing(these, those []string) []string {
	have := make(map[string]bool, len(those))
	for _, name := range those {
		have[name] = true
	}
	out := []string{}
	for _, name := range these {
		if !have[name] {
			out = append(out, name)
		}
	}
	return out
}

// nameList is a list of names for a failure message, with the empty list said
// as a word rather than as nothing.
func nameList(names []string) string {
	if len(names) == 0 {
		return "nothing"
	}
	return strings.Join(names, ", ")
}
