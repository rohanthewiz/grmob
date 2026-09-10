package main

import (
	"fmt"
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
// # And a dirty core/ is a hole in the reading rather than the end of it
//
// The other skip used to be a working tree that differs from HEAD under core/,
// on the sound ground that the two readings would then be over two different
// trees and a difference between them would be the uncommitted edit rather
// than the filters. Sound, and all-or-nothing: core/ is dirty for as long as
// anybody is editing it, so the arm ran on a clean checkout and in CI and
// never for the person who might actually move a filter.
//
// It does not have to be. `git status --porcelain` already NAMES the paths
// that differ, so the comparison is taken over the files that do not, and what
// was held out is said out loud — in the failure message and in the log line
// alike. That is a reading with a stated hole in place of a skip that covered
// everything.
//
//	core/  theme.go   colors.go   type.go   sizes.go
//	                   modified               new, untracked
//	       └─ compared ─┘        └─ compared ─┘
//	                   └──── held out, and named ────┘
//
// Two arms depend on the hole being empty rather than on it being small. The
// name-for-name comparison below is one — the same files with different text
// expand to different names, which is the uncommitted edit again — and it says
// so rather than running over a tree it cannot describe. The other two read
// HEAD alone and are unaffected by anything on disk.
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

	// And WHICH paths would be read out of two different trees. --porcelain
	// lists staged, unstaged and untracked paths alike, which is what is
	// wanted: an uncommitted new file in core/ is exactly the difference that
	// would otherwise read as a filter having drifted.
	//
	// -z rather than the default output, because the default QUOTES any path
	// holding a space, a quote or a non-ASCII byte — `"core/a b.go"`, with the
	// quotes as part of the line — and a path read that way matches nothing
	// coming out of ls-tree or off disk. A file whose name needed quoting
	// would then be counted as clean, which is the one direction this reading
	// must not fail in: it would put a file the two walks disagree about back
	// INTO the comparison and report the disagreement as a drifted filter.
	dirty, err := git("status", "--porcelain", "-z", "--", themePkg)
	if err != nil {
		t.Skipf("git cannot report the state of %s/, so there is no way to tell "+
			"which of its files the two walks would read out of two different "+
			"trees: %v", themePkg, err)
	}
	differ := dirtyPaths(dirty)

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

	// Compared as paths RELATIVE to core/, which is a key both sides can
	// produce: git's are repository-relative ("core/theme.go") and InDir's are
	// joined onto the directory it was handed, which is core/ here.
	//
	// Relative rather than the base name alone, though the two are the same
	// string for every file either walk currently reaches. leavesAt declines a
	// nested path outright now and names it (revision.nested), so the arm
	// below is where a subdirectory of core/ arrives; keying on the relative
	// path is what stops THIS comparison from depending on that filter still
	// being there.
	//
	// The difference is not academic, and it is worse than a wrong match. Run
	// against a revision holding core/sub/theme.go with the descent put back,
	// the two keys report:
	//
	//	relative   50 against 49 — only the revision walk: sub/theme.go
	//	base name  50 against 49 — only the revision walk: nothing
	//	                           only the working tree:  nothing
	//
	// Because "theme.go" is then in the revision's list TWICE, and `missing`
	// is a set difference: every name one side has, the other has too, and the
	// lists are still different lengths. The failure is real, it is loud, and
	// it names no file at all.
	headFiles, headHeld := underPkg(atHead.Files, differ)
	treeFiles, treeHeld := underPkg(inTree.Files, differ)
	held := heldOut(headHeld, treeHeld)

	// The hole, as a sentence every reading below can carry. Held out are the
	// paths one walk or the other reaches AND that differ between HEAD and
	// disk; a modified file that neither walk reads — core/README.md, a
	// _test.go — is not a hole in anything and is not mentioned.
	hole := ""
	if len(held) > 0 {
		hole = fmt.Sprintf("\n\n%d file(s) the walks reach under %s/ differ "+
			"between HEAD and this working tree and were held out of the "+
			"comparison above: %s. For those paths the readings are over two "+
			"different trees, so a difference in them would be the uncommitted "+
			"edit rather than the filters. Everything else in %s/ was compared.",
			len(held), themePkg, strings.Join(held, ", "), themePkg)
	}

	switch {
	case len(headFiles) == 0 && len(treeFiles) == 0:
		// Every file both walks reach is uncommitted, so the comparison is
		// over nothing and slices.Equal would report that as agreement. Said
		// rather than passed: the arms below this one read HEAD alone and did
		// run, and a log line claiming the filters were compared would be
		// describing a comparison that had no members.
		t.Logf("every file the two walks reach under %s/ differs from HEAD in "+
			"this working tree, so the file-set comparison had nothing left to "+
			"run over.%s", themePkg, hole)
	case !slices.Equal(headFiles, treeFiles):
		t.Errorf("the revision walk reads %d file(s) under %s/ at HEAD and "+
			"themeleaves.InDir reads %d in the working tree, out of the files "+
			"the two trees agree about.\n\n"+
			"only the revision walk: %s\nonly the working tree:  %s\n\n"+
			"Those paths are the same in both trees — the ones that are not were "+
			"held out — so this is the two file filters having come apart. "+
			"leavesAt drops test files before fetching them (a `git cat-file` per "+
			"test file per revision is real time spent on text nobody parses) and "+
			"themeleaves.Of drops them again; InDir does its own directory read "+
			"with the same rule. Whichever of the three moved, every row of the "+
			"edit-size table is now over a different population than the arm at "+
			"HEAD checks, and that arm cannot see it: it never goes through "+
			"leavesAt.\n\n"+
			"A path with a directory in it on the REVISION side is leavesAt "+
			"having stopped declining what `ls-tree -r` descends into — see "+
			"revision.nested, and the arm below, which is where that arrives "+
			"when the two are otherwise agreeing.%s",
			len(headFiles), themePkg, len(treeFiles),
			nameList(missing(headFiles, treeFiles)),
			nameList(missing(treeFiles, headFiles)), hole)
		// Everything below reads the expansions those files produced, and with
		// the file sets apart every one of them fails as a consequence of this:
		// a missing file is a missing struct is a missing leaf. The finding is
		// complete here and the rest would be its echo.
		return
	}

	// And what `ls-tree -r` found below core/ that no reading of this package
	// is over. HEAD only, and true whatever is on disk: it is a fact about
	// what git tracks.
	//
	// Not a divergence between the two walks — leavesAt declines these exactly
	// as InDir does, so the arm above stays green — which is what makes it its
	// own arm. Both readings would be short by the same files, agreeing about
	// a population neither of them is over.
	if len(atHead.nested) > 0 {
		t.Errorf("git tracks %d non-test .go file(s) under %s/ at HEAD that are "+
			"not directly in it: %s.\n\n"+
			"Both walks decline them and so they do not show up above: leavesAt "+
			"names them (revision.nested) and themeleaves.InDir never descends. "+
			"The rule is one package, one directory, and it is the right rule — "+
			"themeleaves.Of resolves a field's type by BARE NAME against one flat "+
			"map of every struct it parsed, so two packages' declarations in that "+
			"map would let a type from %s/sub answer for a field in %s/, or lose "+
			"to it, on nothing better than sort order.\n\n"+
			"What it means is that %s/ has become more than one package's worth "+
			"of directory, and every row of the edit-size table under "+
			"affordedBandSteps is over the top level of it alone. If those files "+
			"declare nothing %s.%s reaches, the table is still exactly what it "+
			"says it is and this arm wants the new directory written into "+
			"themePkg's note. If they do, the table is short and the reading "+
			"needs the subpackage — which is a different expansion, not a wider "+
			"glob.",
			len(atHead.nested), themePkg, strings.Join(atHead.nested, ", "),
			themePkg, themePkg, themePkg, themePkg, themeType)
	}

	// And that the same files gave the same answer. Redundant while the file
	// sets agree — Of is one function and the text is the same text — which is
	// the point: it costs nothing and it is the assertion that would survive
	// somebody giving leavesAt a second parse.
	//
	// This is the one arm the hole is fatal to rather than merely narrowing.
	// The expansions are over WHOLE readings — Names is not per file and
	// cannot be filtered down to the paths the two trees agree about — so with
	// anything held out the two lists differ by the uncommitted edit and the
	// failure would be a sentence about a file somebody is in the middle of
	// writing.
	switch {
	case len(held) > 0:
		t.Logf("the two expansions were not compared name-for-name: %d of the "+
			"file(s) the walks reach differ between HEAD and this working tree "+
			"(%s), and Names is a reading of the whole file set rather than "+
			"something that can be taken over part of it. The file FILTERS were "+
			"still compared, over the rest. Commit or stash to get this arm back.",
			len(held), strings.Join(held, ", "))
	case !slices.Equal(atHead.Names, inTree.Names):
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
	// What was compared, and what was not. The second half is the point of
	// saying it at all: this arm no longer reports one of two states, and a
	// line that said "the same N files" without saying which N would be the
	// same sentence on a clean checkout and on a tree with half of core/
	// rewritten.
	expanded := fmt.Sprintf("which expand to the same %d leaf name(s)",
		len(atHead.Names))
	if len(held) > 0 {
		expanded = fmt.Sprintf("out of the %d and %d each walk reads in full",
			len(atHead.Files), len(inTree.Files))
	}
	t.Logf("HEAD and the working tree hand themeleaves the same %d file(s) "+
		"under %s/, %s. That is the half of this command's reading wasm/verify "+
		"cannot check: it holds the EXPANSION against reflect, and the expansion "+
		"is downstream of the file filter this test compares. The two filters are "+
		"`ls-tree` plus leavesAt's own suffix and top-level tests on one side and "+
		"themeleaves.InDir's directory read on the other.%s",
		len(headFiles), themePkg, expanded, hole)
}

// underPkg is a list of paths as the two walks can be compared — each one
// relative to themePkg, sorted — split into the ones this run can read and the
// ones held out because HEAD and the working tree disagree about them.
//
// Relative rather than the base name: see the call, which is where the
// difference between the two matters. A path this function cannot make
// relative is kept whole, which is the loud version of the failure — it will
// not match its opposite number and the file-set arm will name it — rather
// than a silent fall back onto the base name, which would match one it is not.
//
// Sorted here rather than relied on: themeleaves.Files comes out sorted by
// PATH, and two different prefixes can sort their shared suffixes into two
// different orders ("core/a.go" before "core/b.go" is also "a.go" before
// "b.go", but that is a property of these prefixes and not of any two).
func underPkg(paths []string, differ map[string]bool) (kept, held []string) {
	for _, p := range paths {
		rel, err := filepath.Rel(themePkg, filepath.FromSlash(p))
		if err != nil {
			rel = p
		} else {
			rel = filepath.ToSlash(rel)
		}
		if differ[rel] {
			held = append(held, rel)
			continue
		}
		kept = append(kept, rel)
	}
	slices.Sort(kept)
	slices.Sort(held)
	return kept, held
}

// heldOut is the two walks' held-out paths as one list, each named once.
//
// A modified file is normally in both — it is the same path in both readings
// — and an added or deleted one is in exactly one. The union is what the
// sentence about the hole is over, and it is deliberately NOT every dirty path
// under themePkg: a modified README or _test.go is not a hole in a comparison
// neither walk was going to read it for.
func heldOut(these, those []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, list := range [][]string{these, those} {
		for _, p := range list {
			if !seen[p] {
				seen[p] = true
				out = append(out, p)
			}
		}
	}
	slices.Sort(out)
	return out
}

// dirtyPaths is `git status --porcelain -z` as the set of paths, relative to
// themePkg, that HEAD and the working tree disagree about.
//
// # The format, which is why -z is worth the parse
//
// Records are NUL-terminated rather than newline-terminated, and a path is
// never quoted or escaped — which is the whole reason for asking (see the
// call). Each record is two status letters — index, then work tree — a space,
// and the path. Written with · for a space, since one of the two letters
// routinely is one:
//
//	"·M core/theme.go\0"              index clean, work tree modified
//	"?? core/new.go\0"                untracked
//	"R· core/to.go\0core/from.go\0"   a rename, whose SOURCE is the record
//	                                  that follows it
//
// Both halves of a rename are held out. The source is gone from the working
// tree and present at HEAD and the destination is the other way round, so each
// of them is a path exactly one of the two walks reads.
func dirtyPaths(z string) map[string]bool {
	rel := func(p string) string {
		r, err := filepath.Rel(themePkg, filepath.FromSlash(p))
		if err != nil {
			return p
		}
		return filepath.ToSlash(r)
	}
	out := map[string]bool{}
	records := strings.Split(z, "\x00")
	for i := 0; i < len(records); i++ {
		// "XY p" is the shortest a record can be; the trailing empty string
		// Split leaves after the final NUL is the usual reason to be here.
		if len(records[i]) < 4 {
			continue
		}
		status, p := records[i][:2], records[i][3:]
		out[rel(p)] = true
		if strings.ContainsAny(status, "RC") {
			i++
			if i < len(records) && records[i] != "" {
				out[rel(records[i])] = true
			}
		}
	}
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
