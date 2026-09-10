package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
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
// No arm depends on the hole being empty. The name-for-name comparison below
// is the one that nearly did — the same files with different text expand to
// different names, which is the uncommitted edit again — and rather than
// standing down it re-takes both readings over the kept paths alone, so the
// two expansions are over one file set by construction. The other two read
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
	tracked, err := treePaths("HEAD")
	if err != nil {
		t.Skipf("HEAD cannot be read, which is a repository with no commits in "+
			"it (or a detached state git will not resolve): %v", err)
	}
	if len(tracked) == 0 {
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
			"leavesAt drops test files before fetching them (a fetch per test file "+
			"per revision is text nobody parses, carried eighty-eight times) and "+
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
	//
	// # How to watch it fire
	//
	// The state it is about is a fact about git's tree, so no testdata
	// directory can carry it and no fixture can stand in for it: the arm reads
	// `ls-tree -r` at HEAD. What can be written down is the recipe, so that a
	// later reader has the arm, the message AND a way to see the two of them
	// meet. From the repository root, in a throwaway clone that is thrown away
	// afterwards:
	//
	//	git clone . /tmp/nested-core && cd /tmp/nested-core
	//	mkdir core/zsub
	//	printf 'package zsub\n\ntype Extra struct{ Pad int }\n' > core/zsub/extra.go
	//	git add core/zsub/extra.go && git commit -q -m 'a nested revision'
	//	go test ./internal/themehistory -run TestTheRevisions -v
	//
	// A subpackage declaring nothing core.Theme reaches is the LOGGING half —
	// a note to write, not a failure. For the failing half, give it a type the
	// top level already declares:
	//
	//	printf 'package zsub\n\ntype SpacingScale struct{ Ghost int }\n' \
	//	  > core/zsub/extra.go
	//
	// zsub and not sub, and this is the part worth having written down. Of
	// sorts its paths before building the flat map, and the LAST declaration
	// of a name wins: "core/sub/x.go" sorts before "core/theme.go" and loses
	// to it, so a colliding sub/ moves nothing and the run stays green.
	// "core/zsub/x.go" sorts after and takes the name. The hazard is real
	// either way — which of two packages answers for a field is decided by
	// sort order over file paths — and only one direction of it is visible.
	if len(atHead.nested) > 0 {
		reportNested(t, atHead)
	}

	// And that the same files gave the same answer. Redundant while the file
	// sets agree — Of is one function and the text is the same text — which is
	// the point: it costs nothing and it is the assertion that would survive
	// somebody giving leavesAt a second parse.
	//
	// # The hole narrows this arm too, rather than ending it
	//
	// It used to stand down completely on a dirty core/, on the ground that
	// Names is a reading of a WHOLE file set: a leaf is not attributable to
	// the file that declared it, so the two lists cannot be filtered down to
	// the paths the two trees agree about after the fact. Sound about
	// filtering — and it left this arm off in exactly the state the file-set
	// comparison above had just been taught to keep running through.
	//
	// A reading that cannot be filtered can be RE-TAKEN. themeleaves.Of is
	// exported and takes a sources map, so the kept paths are fetched out of
	// HEAD, the same paths are read off disk, and the two expansions are over
	// one file set by construction:
	//
	//	held out     nothing              something
	//	compared     Of(HEAD)      vs     Of(kept @ HEAD)  vs
	//	             Of(disk)             Of(kept on disk)
	//	population   core.Theme's         whatever the kept files declare
	//
	// The second reading is not core.Theme's population, and every message
	// under here says which of the two it is. A struct declared only in a
	// held-out file is missing from BOTH sides, so the lists can agree and be
	// short — fine for what this arm asks, which is whether two mechanisms
	// handed one file set agree, and not fine as a statement about the eighty
	// the table is about.
	headNames, treeNames := atHead.Names, inTree.Names
	// Two flags and not one. `subset` says which populations the names below
	// are, and it words every message under here. `compared` says whether
	// there are two comparable lists AT ALL — and with a hole in the tree the
	// whole expansions are not two of them: they differ by the uncommitted
	// edit, which is precisely the reading this arm used to stand down rather
	// than report as a drifted filter. So a re-reading that cannot be taken
	// leaves this arm silent, exactly as standing down did; what changed is
	// how often that happens.
	subset, compared := false, true
	if len(held) > 0 {
		headSub, treeSub, err := expansionsOver(headFiles)
		r := readSubset(headSub, treeSub, err, headFiles, held)
		switch {
		case r.failed:
			t.Errorf("%s", r.report)
		case r.report != "":
			t.Logf("%s", r.report)
		}
		if r.compared {
			headNames, treeNames = headSub.Names, treeSub.Names
		}
		subset, compared = r.compared, r.compared
	}
	if compared && !slices.Equal(headNames, treeNames) {
		// What the two lists are OVER, which is the difference between "the
		// expansion moved" and "the expansion of a subset of core/ moved".
		over := fmt.Sprintf("the same %d file(s)", len(headFiles))
		if subset {
			over = fmt.Sprintf("the %d file(s) the two trees agree about — the "+
				"%d that differ (%s) were held out of BOTH readings, so these "+
				"are not %s.%s's whole population on either side",
				len(headFiles), len(held), strings.Join(held, ", "), themePkg,
				themeType)
		}
		t.Errorf("%s.%s expands to %d leaf name(s) out of HEAD and %d out of "+
			"%s.\n\n"+
			"only HEAD's:        %s\nonly the tree's:    %s\n\n"+
			"Both readings are themeleaves.Of over the same names and the same "+
			"text, so this is not a filter: it is the two having been handed "+
			"different CONTENT for a file they both hold. `git cat-file` returning "+
			"something other than what is on disk for an unmodified path is a "+
			"checkout with filters or line-ending conversion configured, and every "+
			"revision in the table was parsed through the same conversion.",
			themePkg, themeType, len(headNames), len(treeNames), over,
			nameList(missing(headNames, treeNames)),
			nameList(missing(treeNames, headNames)))
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

	// And that nothing in core/ was declared twice. themeleaves.Of keys every
	// struct it parses by its bare NAME in one flat map, so a second
	// declaration overwrites the first and which one answers for a field is
	// decided by the order the paths sorted in — a resolution it makes and now
	// records (Expansion.Shadowed).
	//
	// At HEAD that state does not build, which is what makes this arm cheap
	// and worth having anyway: it is empty here for a reason outside this
	// package's control, so a non-empty one is either a `go vet`-clean tree
	// that would not compile or Of having started keying on something else.
	// The nested arm above is where a legitimate collision arrives — two
	// packages, one name, and both of them building.
	if len(atHead.Shadowed) > 0 {
		named := make([]string, 0, len(atHead.Shadowed))
		for _, sh := range atHead.Shadowed {
			named = append(named, fmt.Sprintf("%s (%s)", sh.Name,
				strings.Join(sh.Files, ", ")))
		}
		t.Errorf("%d struct name(s) are declared more than once in %s/ at "+
			"HEAD: %s.\n\n"+
			"themeleaves.Of resolved each of them to whichever path sorted "+
			"last, so this revision's expansion is over one declaration of each "+
			"and would look exactly as whole if it were over the other. One "+
			"package cannot declare a name twice and build, so this is a tree "+
			"that does not compile — or Of no longer resolving field types by "+
			"bare name, in which case every row of the edit-size table was "+
			"taken by a rule that has moved.",
			len(atHead.Shadowed), themePkg, strings.Join(named, ", "))
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
	switch {
	case subset:
		// The count is the subset's and not the struct's, so the sentence says
		// what it was taken over. A run reporting "the same 74 leaf name(s)"
		// with no such qualification would read as core.Theme having shrunk.
		expanded = fmt.Sprintf("which expand — over those %d alone, out of the "+
			"%d and %d each walk reads in full — to the same %d leaf name(s)",
			len(headFiles), len(atHead.Files), len(inTree.Files), len(headNames))
	case len(held) > 0:
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

// subsetReading is what a re-reading over the kept files came to: what to say
// about it, whether that is a failure, and whether it produced two lists worth
// comparing.
//
// report is empty when there is nothing to say, which is the ordinary case:
// the two readings agree about the struct and the caller goes on to compare
// their names.
type subsetReading struct {
	report   string
	failed   bool
	compared bool
}

// readSubset is the decision the four outcomes of expansionsOver make.
//
// # Why this is a function and not the switch it used to be
//
// It was four cases inline in the arm, and three of them are reachable only
// from states nobody can produce on demand: a fetch that fails between one
// `ls-tree` and the next read, a working tree whose theme.go is dirty, and a
// `git cat-file` that disagrees with the disk about a file `git status` calls
// clean. So the branch that ran was the fourth, on every green run, and the
// other three were read-only prose in a file whose whole subject is arms that
// do not run.
//
// Every input the decision uses is a value: two expansions, an error, and the
// two path lists. Lifted out, all four outcomes are a table — and what stays
// behind in the arm is the FETCH, which is the part that genuinely needs a
// repository and a dirty tree.
//
//	err != nil            a failure. The paths were all read moments ago
//	neither Found         a note. theme.go itself is held out, so both
//	                      re-readings are empty and equal for no reason
//	one Found             a failure, and a strange one: same bytes, two answers
//	both Found            the comparison the caller wanted, over a SUBSET
func readSubset(headSub, treeSub themeleaves.Expansion, err error,
	kept, held []string) subsetReading {
	switch {
	case err != nil:
		return subsetReading{failed: true, report: fmt.Sprintf(
			"the %d file(s) both walks reach and both trees agree about cannot "+
				"be re-read to compare the expansions over them: %v\n\nEvery "+
				"one of those paths was listed by `ls-tree` at HEAD and read off "+
				"disk by themeleaves.InDir in this same run, so a fetch that "+
				"fails here is the tree having moved underneath this test rather "+
				"than a finding about either walk.", len(kept), err)}
	case !headSub.Found && !treeSub.Found:
		// Two empty populations comparing equal is not evidence, and this is
		// the shape a dirty theme.go produces: the file the struct is declared
		// in is held out, so neither re-reading finds it. The remaining files
		// are still a comparison of the FILTERS, which ran above; they are not
		// a comparison of the expansion.
		return subsetReading{report: fmt.Sprintf(
			"the expansions were re-taken over the %d file(s) the two trees "+
				"agree about and neither reading declares %s.%s in them, so a "+
				"name-for-name comparison would be between two empty "+
				"populations. The %d held-out file(s) (%s) include the one that "+
				"declares it. The file FILTERS were still compared, over the "+
				"same %d. Commit or stash to get the whole reading back.",
			len(kept), themePkg, themeType, len(held),
			strings.Join(held, ", "), len(kept))}
	case headSub.Found != treeSub.Found:
		return subsetReading{failed: true, report: fmt.Sprintf(
			"over the same %d file(s) — the ones HEAD and this working tree "+
				"agree about — %s is declared at HEAD: %v, and in the working "+
				"tree: %v.\n\nThe text is the same text for every one of those "+
				"paths, so one reading finding the struct and the other not is "+
				"`git cat-file` and the disk disagreeing about a file neither "+
				"`git status` nor either walk calls different.",
			len(kept), themeType, headSub.Found, treeSub.Found)}
	default:
		return subsetReading{compared: true}
	}
}

// expansionsOver is the two readings re-taken over one named set of files: the
// same paths fetched out of HEAD and read off disk, each handed to
// themeleaves.Of.
//
// This is what the name-for-name arm runs on when something under core/ is
// dirty. Names is a reading of a WHOLE file set and cannot be filtered down
// after the fact — a leaf carries no record of the file that declared it — so
// the way to compare names over part of core/ is to take both readings over
// that part in the first place.
//
// The paths are the KEPT ones: every path is by construction one both trees
// agree about, so `cat-file` and the disk are being asked for the same bytes
// and a difference between the two expansions is a difference in the
// mechanism rather than in somebody's uncommitted edit. That is the whole
// premise, and it is why the caller passes the kept list rather than
// Expansion.Files.
//
// Cost is one request per kept file on a dirty tree, down the same
// `git cat-file --batch` the command itself reads through (see blob), so it is
// a write and a read on an open pipe rather than a process apiece. What it
// buys is an arm that runs while somebody is editing core/, which is when a
// filter moves.
func expansionsOver(rels []string) (atHead, inTree themeleaves.Expansion, err error) {
	headSrc := make(map[string]string, len(rels))
	treeSrc := make(map[string]string, len(rels))
	for _, rel := range rels {
		// Keyed by the repository-relative path on BOTH sides, so the two
		// readings sort their inputs identically. Of sorts the paths before
		// building its flat map of struct declarations precisely so that two
		// runs over one file set build it in one order; two runs keyed
		// differently would be two orders, and a package declaring one name
		// twice would answer differently on each side for no reason either
		// walk is about.
		key := path.Join(themePkg, rel)
		src, err := blob("HEAD", key)
		if err != nil {
			return atHead, inTree, fmt.Errorf("HEAD:%s: %w", key, err)
		}
		headSrc[key] = src
		// filepath for the disk read and `path` for the key: git speaks
		// slash-separated paths on every platform and this one has to open a
		// file on this machine.
		b, err := os.ReadFile(filepath.Join(themePkg, filepath.FromSlash(rel)))
		if err != nil {
			return atHead, inTree, fmt.Errorf("%s off disk: %w", key, err)
		}
		treeSrc[key] = string(b)
	}
	return themeleaves.Of(headSrc, themeType), themeleaves.Of(treeSrc, themeType), nil
}

// reportNested is the finding about non-test .go files git tracks under
// themePkg that are not directly in it — and the reading that turns the
// question into an answer.
//
// The arm used to be one Errorf that ASKED. It named the files and told the
// reader that if they declare nothing core.Theme reaches then the edit-size
// table is still exactly what it says it is, and that if they do then the
// table is short. That is a finding left undetermined — a red build with a
// paragraph asking somebody to go and think — and a legitimate core/internal/
// would have produced it on every run forever.
//
// It is a question this arm can answer itself. The files are in git: fetch
// them, parse them alongside the ones the reading already used, and compare
// the two populations.
//
//	Of(core/*.go)                  the reading every row of the table is
//	                               over — one package, one directory
//	Of(core/*.go + core/sub/*.go)  the same reading with the subdirectory's
//	                               declarations in the same flat map
//
// Same names: nothing under there reaches core.Theme, the table is intact, and
// what is left is a note to write — a Logf. Different names: the reading is
// short, or colliding, and that is the finding this arm exists for, with the
// leaves named.
//
// # Why the union is a probe and not a second definition
//
// themeleaves.Of resolves a field's type by BARE NAME against one flat map of
// every struct it parsed, which is exactly what InDir's one-package-one-
// directory rule exists to prevent: a TextStyle in core/sub and a TextStyle in
// core/ are one key in that map, and which of them answers is decided by sort
// order over paths. So a difference here has two possible causes — a leaf the
// subpackage really contributes, or a name it shadowed — and the union is a
// thing to look at rather than the population.
//
// # Which of the two it is, though, the probe can now say
//
// It used to print both causes and leave the reader to work out which. The
// choice is not a mystery to the code that made it: Of records every name it
// got more than one declaration of, in Expansion.Shadowed, with the paths that
// declared it and the winner last. So the message names the collisions if
// there are any and says there are none if there are not, and the reader is
// told which of the two paragraphs applies to them.
//
//	Shadowed empty      the subdirectory contributes or removes the leaves
//	                    listed — the reading is short by them
//	Shadowed non-empty   a bare name got two declarations, and at least part of
//	                    the difference is which of them answered
//
// # And WHERE the two declarations were, which the sentence used to assume
//
// It said "declared both in core/ and below it", which is the union probe's
// case and not the only one Shadowed can hold: Of records a repeat wherever it
// parsed one, so two subdirectories shadowing each other, one directory
// declaring a name twice, or one FILE declaring it twice would all have
// printed that sentence about something else. A reader with a moved population
// uses that sentence to decide where to look. It is shadowKind's answer now,
// per collision, and it is the same phrase the command prints on stderr —
// one rule with one copy, which is what the rest of this file's helpers are
// for.
func reportNested(t *testing.T, atHead revision) {
	t.Helper()
	union, err := unionWithNested(atHead)
	// Which of the two causes the union met, as a sentence both branches
	// below carry. Computed once and appended rather than written into each,
	// because the benign branch needs it too: two declarations of one name can
	// hold the same fields, in which case the populations agree AND a choice
	// was still made between them.
	shadow := "\n\nNo bare name is declared twice anywhere in this union, so " +
		"nothing here is a shadowed type: themeleaves.Of resolves field types " +
		"against one flat map of everything it parsed, and this union gave it " +
		"no name twice."
	if len(union.Shadowed) > 0 {
		names := make([]string, 0, len(union.Shadowed))
		for _, sh := range union.Shadowed {
			// shadowKind names the placement AND the paths with the winner
			// last, which is what Shadowed.Files is ordered for — a reader
			// with a moved population wants to know which of two declarations
			// the expansion above was actually taken over, and in which
			// directory to go and look.
			names = append(names, sh.Name+" is "+shadowKind(sh))
		}
		shadow = fmt.Sprintf("\n\n%d bare name(s) in this union got more than "+
			"one declaration:\n\n  %s\n\nthemeleaves.Of resolves field types "+
			"against one flat map of every struct it parsed, so two "+
			"declarations of one name are one key and the winner is whichever "+
			"path sorted LAST — %s/sub/x.go loses to %s/theme.go and "+
			"%s/zsub/x.go beats it. That is what the one-package-one-directory "+
			"rule exists to prevent, and it is why this union is a probe and "+
			"not a wider glob.",
			len(union.Shadowed), strings.Join(names, "\n  "),
			themePkg, themePkg, themePkg)
	}
	switch {
	case err != nil:
		t.Errorf("git tracks %d non-test .go file(s) under %s/ at HEAD that are "+
			"not directly in it (%s), and they cannot be fetched to find out "+
			"whether they move the population: %v\n\n"+
			"`ls-tree` named those paths at HEAD in this same run, so a "+
			"`cat-file` that fails on one of them is the repository having "+
			"changed underneath this test.",
			len(atHead.nested), themePkg, strings.Join(atHead.nested, ", "), err)
	case slices.Equal(union.Names, atHead.Names):
		// The benign half, and the reason this is a Logf and not a failure:
		// core/ has become more than one directory and the reading is still
		// the whole reading. Nothing here is wrong; something here is
		// undocumented.
		t.Logf("git tracks %d non-test .go file(s) under %s/ at HEAD that are "+
			"not directly in it: %s. Parsed alongside %s/'s own files they add "+
			"and remove no leaf — the union expands to the same %d name(s) — so "+
			"every row of the edit-size table under affordedBandSteps is still "+
			"over %s.%s's whole population.\n\n"+
			"Both walks decline them and neither is wrong to: the rule is one "+
			"package, one directory. What is left is a note — themePkg's doc "+
			"comment says core/ is where the struct and everything it holds are "+
			"declared, and that is now a statement about the top level of a "+
			"directory that has more in it.%s",
			len(atHead.nested), themePkg, strings.Join(atHead.nested, ", "),
			themePkg, len(atHead.Names), themePkg, themeType, shadow)
	default:
		t.Errorf("git tracks %d non-test .go file(s) under %s/ at HEAD that are "+
			"not directly in it — %s — and parsing them alongside %s/'s own "+
			"files MOVES the population: %d name(s) against the %d every row of "+
			"the edit-size table is over.\n\n"+
			"only the union's:      %s\nonly the top level's:  %s\n\n"+
			"Two things produce that, and the sentence below says which of them "+
			"this is. Either those files declare something %s.%s reaches, and "+
			"the table is short by it — in which case the reading needs the "+
			"subpackage, which is a different expansion and not a wider glob. Or "+
			"a bare name got two declarations somewhere in this union, and the "+
			"difference is which of them answered — in which case the sentence "+
			"names WHERE the two were, because that is where to go and look.%s\n\n"+
			"Both walks still decline these files identically, so the file-set "+
			"comparison stays green and this is the only arm that can say so.",
			len(atHead.nested), themePkg, strings.Join(atHead.nested, ", "),
			themePkg, len(union.Names), len(atHead.Names),
			nameList(missing(union.Names, atHead.Names)),
			nameList(missing(atHead.Names, union.Names)),
			themePkg, themeType, shadow)
	}
}

// unionWithNested is themePkg's expansion at HEAD re-taken with the files
// below it included — the reading leavesAt declined, taken once so the arm
// above can say whether declining it cost anything.
//
// Both halves are fetched rather than one being reused: leavesAt hands back
// the expansion and not the text it was over, and re-fetching what it already
// read is one batched request per file (see blob) on a run that is by
// definition rare — this repository has never had a revision with anything
// below core/. Paying it here keeps revision — and the command — free of a
// sources map that exists for a test.
func unionWithNested(rev revision) (themeleaves.Expansion, error) {
	sources := make(map[string]string, len(rev.Files)+len(rev.nested))
	for _, p := range append(append([]string{}, rev.Files...), rev.nested...) {
		src, err := blob("HEAD", p)
		if err != nil {
			return themeleaves.Expansion{}, fmt.Errorf("HEAD:%s: %w", p, err)
		}
		sources[p] = src
	}
	return themeleaves.Of(sources, themeType), nil
}

// underPkg is a list of paths as the two walks can be compared — each one
// relative to themePkg, sorted — split into the ones this run can read and the
// ones held out because HEAD and the working tree disagree about them.
//
// Relative rather than the base name: see the call, which is where the
// difference between the two matters. The keying itself is relToPkg, which
// dirtyPaths uses too — the two lists have to be keyed identically or the
// comparison is between two spellings, and one rule in two copies is the shape
// this whole test exists to catch one level up.
//
// Sorted here rather than relied on: themeleaves.Files comes out sorted by
// PATH, and two different prefixes can sort their shared suffixes into two
// different orders ("core/a.go" before "core/b.go" is also "a.go" before
// "b.go", but that is a property of these prefixes and not of any two).
func underPkg(paths []string, differ map[string]bool) (kept, held []string) {
	for _, p := range paths {
		rel := relToPkg(p)
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

// relToPkg is a repository-relative path as both sides of the comparison are
// keyed: relative to themePkg, slash-separated, and kept WHOLE if it cannot be
// made relative.
//
// # Why the two callers share this
//
// underPkg keys the walks' file lists and dirtyPaths keys the paths `git
// status` named, and the second is looked up in the first. Two spellings of
// one rule would not fail loudly — a dirty file keyed one way and looked up
// the other is simply never found, so it would be compared rather than held
// out, and the arm would report somebody's uncommitted edit as the two file
// filters having come apart. That is the same failure this test exists to
// distinguish from a real one, arriving through its own keying.
//
// Keeping an unresolvable path whole is the loud choice and it is pinned by
// TestAPathThatCannotBeMadeRelativeIsKeptWhole: it matches nothing, so the
// file-set arm names it exactly as it was handed over, where a fall back onto
// the base name would match a file it is not.
func relToPkg(p string) string {
	rel, err := filepath.Rel(themePkg, filepath.FromSlash(p))
	if err != nil {
		return p
	}
	return filepath.ToSlash(rel)
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
//	"?? core/zsub/\0"                 an untracked DIRECTORY, collapsed to one
//	                                  record with a trailing slash
//	"R· core/to.go\0core/from.go\0"   a rename, whose SOURCE is the record
//	                                  that follows it
//
// Both halves of a rename are held out. The source is gone from the working
// tree and present at HEAD and the destination is the other way round, so each
// of them is a path exactly one of the two walks reads.
//
// The collapsed directory is here because it is a shape this parse meets and
// NOT because it does anything. `git status` reports an untracked directory as
// ONE record rather than listing what is in it — that is what -uall is for —
// so a new core/zsub/ full of .go files arrives as the single record
// "?? core/zsub/", and the set gets a key for a directory.
//
// It comes out as "zsub" and not "zsub/": relToPkg goes through filepath.Rel,
// which cleans its result, and a trailing separator does not survive that.
// Either spelling is inert — this map is only ever consulted for paths one of
// the two walks RETURNED, both walks return .go files, and neither descends —
// so nothing turns on which one it is. It is in the table below so that
// "inert" is a thing this file has checked rather than a thing it assumes.
func dirtyPaths(z string) map[string]bool {
	out := map[string]bool{}
	records := strings.Split(z, "\x00")
	for i := 0; i < len(records); i++ {
		// "XY p" is the shortest a record can be; the trailing empty string
		// Split leaves after the final NUL is the usual reason to be here.
		if len(records[i]) < 4 {
			continue
		}
		status, p := records[i][:2], records[i][3:]
		out[relToPkg(p)] = true
		if strings.ContainsAny(status, "RC") {
			i++
			if i < len(records) && records[i] != "" {
				out[relToPkg(records[i])] = true
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

// The four outcomes of a re-reading, and which of them is a finding.
//
// # What this replaces
//
// The decision used to be a switch inline in the arm above, and only its last
// case ever ran: the other three need a fetch that fails mid-run, a working
// tree whose theme.go is dirty, or a `git cat-file` that disagrees with the
// disk about a file `git status` calls clean. Three branches, each of them the
// thing a reader meets in a failure message, none of them ever executed —
// which is the shape this whole file exists to get rid of.
//
// The inputs are all values, so the outcomes are a table. What is asserted is
// the DECISION — is this a failure, a note or nothing, and are there two lists
// worth comparing — plus the facts each message has to carry. Not the wording:
// a test that pinned the prose would fail on every rephrasing and would be
// asserting that somebody had not edited a paragraph.
func TestTheFourOutcomesOfARereadingAreToldApart(t *testing.T) {
	kept := []string{"colors.go", "sizes.go", "theme.go"}
	held := []string{"type.go"}
	found := themeleaves.Expansion{Found: true, Names: []string{"Background"}}
	empty := themeleaves.Expansion{}

	for _, c := range []struct {
		what       string
		head, tree themeleaves.Expansion
		err        error
		failed     bool
		compared   bool
		// Facts the message must carry, whatever words carry them. Each is a
		// substring the reader needs in order to act on the finding at all.
		carries []string
	}{
		{
			what: "a fetch that failed",
			head: empty, tree: empty,
			err:    fmt.Errorf("HEAD:core/theme.go: object missing"),
			failed: true,
			// The count of what could not be read, and git's own words —
			// without them the reader has a failure and no next step.
			carries: []string{"3 file(s)", "object missing"},
		},
		{
			what: "neither reading declaring the struct",
			head: empty, tree: empty,
			failed: false, compared: false,
			// Which files were held out, or the note is untraceable: the whole
			// finding is that one of THOSE declares Theme.
			carries: []string{"type.go", themeType, "Commit or stash"},
		},
		{
			what: "one reading declaring it and the other not",
			head: found, tree: empty,
			failed: true, compared: false,
			// Both answers, since which side found it is the finding.
			carries: []string{"HEAD: true", "tree: false"},
		},
		{
			what: "both readings declaring it",
			head: found, tree: found,
			failed: false, compared: true,
			// Nothing to say: the caller goes on to compare the names, and a
			// note here would be a sentence on every dirty-tree run.
			carries: nil,
		},
	} {
		r := readSubset(c.head, c.tree, c.err, kept, held)
		if r.failed != c.failed || r.compared != c.compared {
			t.Errorf("%s reads as failed=%v compared=%v, and failed=%v "+
				"compared=%v is the decision.\n\n"+
				"compared is what says the two name lists are worth comparing at "+
				"all — a false one leaves the name-for-name arm silent, exactly "+
				"as standing down on a dirty tree used to, and a wrongly true "+
				"one reports somebody's uncommitted field edit as the two file "+
				"filters having come apart.",
				c.what, r.failed, r.compared, c.failed, c.compared)
		}
		if (r.report == "") != (c.carries == nil) {
			t.Errorf("%s produces report %q, and %v is whether it should say "+
				"anything.\n\n"+
				"The silent outcome is the ordinary one: both readings found the "+
				"struct and the caller compares their names. Every other outcome "+
				"is the arm reporting on a reading it could not take.",
				c.what, r.report, c.carries != nil)
			continue
		}
		for _, want := range c.carries {
			if !strings.Contains(r.report, want) {
				t.Errorf("%s produces a message that does not carry %q:\n\n%s",
					c.what, want, r.report)
			}
		}
	}
}

// The batched fetch reads the same bytes the per-file one did.
//
// # Why this arm exists at all
//
// Every revision in the edit-size table is parsed out of text this reader
// produced. It replaced one `git cat-file -p` per file — eighty-eight commits
// times up to forty-nine files, around forty-four hundred processes, thirty-one
// seconds — with a single `git cat-file --batch` the whole run talks to, which
// is 1.8 seconds for the same sixteen rows.
//
// What that trades is a process boundary for a PARSE. `-p` hands back a
// process's entire stdout and cannot return the wrong thing; `--batch` hands
// back a stream of headers and counted bodies, and a reader that is one byte
// out anywhere is one byte out for every file after it. The failure would not
// look like a failure: go/parser would be handed something that begins in the
// middle of the previous file, and a revision whose expansion is short reads
// as a commit that removed leaves.
//
// So the two are held against each other over every file the walk actually
// reads, which is the population the table is over.
func TestTheBatchedFetchReadsWhatCatFileDoes(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git on this machine, so there is no fetch to compare — see " +
			"TestTheRevisionsFileSetIsTheOneTheWorkingTreeWalkReads, which is " +
			"the arm this repository's one honest git skip is stated for.")
	}
	root, err := git("rev-parse", "--show-toplevel")
	if err != nil {
		t.Skipf("this checkout is not a git repository: %v", err)
	}
	t.Chdir(strings.TrimSpace(root))

	listed, err := treePaths("HEAD")
	if err != nil {
		t.Skipf("HEAD cannot be read: %v", err)
	}
	var files []string
	for _, p := range listed {
		if strings.HasSuffix(p, ".go") && !strings.HasSuffix(p, "_test.go") &&
			path.Dir(p) == themePkg {
			files = append(files, p)
		}
	}
	if len(files) == 0 {
		t.Skipf("git tracks no non-test .go files directly in %s/ at HEAD, so "+
			"this is not the repository %s.%s is declared in.",
			themePkg, themePkg, themeType)
	}

	// Every file, and in the order the walk reads them, because the failure
	// this is about is positional: a reader that loses a byte on one file is
	// wrong from there on, so a spot check of one file would be the one check
	// that cannot see it.
	for _, p := range files {
		want, err := git("cat-file", "-p", "HEAD:"+p)
		if err != nil {
			t.Fatalf("`git cat-file -p HEAD:%s` failed, and `ls-tree` named that "+
				"path at HEAD in this same run: %v", p, err)
		}
		got, err := blob("HEAD", p)
		if err != nil {
			t.Fatalf("the batched reader could not fetch HEAD:%s, which "+
				"`git cat-file -p` just read: %v", p, err)
		}
		if got != want {
			t.Fatalf("HEAD:%s reads as %d byte(s) through `git cat-file --batch` "+
				"and %d through `git cat-file -p`.\n\n"+
				"first difference at byte %d.\n\n"+
				"The batched reader takes the body by COUNT out of a shared "+
				"stream — <oid> <type> <size>, then that many bytes, then a "+
				"terminating newline that is not in the size — so a length it "+
				"reads wrongly or a terminator it leaves unread puts every file "+
				"after this one at the wrong offset. Every row of the edit-size "+
				"table is go/parser over text this reader produced.",
				p, len(got), len(want), firstDiff(got, want))
		}
	}
	t.Logf("all %d file(s) the walk reads under %s/ at HEAD come back "+
		"byte-identical through `git cat-file --batch` and `git cat-file -p`. "+
		"The batch is why `go run ./internal/themehistory` is 1.8s rather than "+
		"31s: one git process for the run instead of one per file per revision.",
		len(files), themePkg)
}

// A missing object is an error that leaves the stream usable.
//
// This is the one error the batched reader can meet and carry on from, and it
// is worth pinning because carrying on WRONGLY is the failure that does not
// look like one. "<name> missing" is a complete response — one line and no
// body — so the stream is left at the start of the next header. A reader that
// treated it as a header with a body would go looking for bytes that are not
// there and hand the next caller a file read from the middle of another one,
// which go/parser would accept without complaint for as long as it happened to
// begin at a declaration.
func TestAMissingObjectDoesNotDesynchroniseTheBatch(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git on this machine, so there is no batch to desynchronise.")
	}
	root, err := git("rev-parse", "--show-toplevel")
	if err != nil {
		t.Skipf("this checkout is not a git repository: %v", err)
	}
	t.Chdir(strings.TrimSpace(root))

	real := path.Join(themePkg, "theme.go")
	want, err := git("cat-file", "-p", "HEAD:"+real)
	if err != nil {
		t.Skipf("HEAD:%s cannot be read, so this is not the repository %s.%s is "+
			"declared in: %v", real, themePkg, themeType, err)
	}

	// A path git will resolve the revision of and not find the object for,
	// which is what a fetch at the wrong revision looks like.
	absent := path.Join(themePkg, "no-such-file-a4232825.go")
	if _, err := blob("HEAD", absent); err == nil {
		t.Fatalf("the batched reader returned a blob for HEAD:%s, and nothing "+
			"is meant to be there. Either that file now exists — rename it in "+
			"this test — or a `missing` response is being read as an object.",
			absent)
	}

	// And the next request, which is the whole point: the stream has to be
	// sitting at the start of a header and not part-way through one.
	got, err := blob("HEAD", real)
	if err != nil {
		t.Fatalf("after a missing object, the batched reader could not fetch "+
			"HEAD:%s: %v\n\nA `missing` response is one line with no body, so "+
			"nothing should have been left unread behind it.", real, err)
	}
	if got != want {
		t.Errorf("after a missing object, HEAD:%s reads as %d byte(s) through "+
			"the batch and %d through `git cat-file -p`; first difference at "+
			"byte %d.\n\n"+
			"The stream is desynchronised: the reader consumed something for the "+
			"missing object that was not there, so this file was read starting "+
			"from the wrong offset. Every file after it in a run would be too.",
			real, len(got), len(want), firstDiff(got, want))
	}
}

// firstDiff is where two strings stop agreeing, for a message about a stream
// that is read by position. The length is the answer when one is a prefix of
// the other.
func firstDiff(a, b string) int {
	n := min(len(a), len(b))
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}

// A rename inside themePkg is two paths in one `git status` record, and both
// of them are a hole.
//
// # Why this one is a unit test and not another run of the arm above
//
// Every other shape dirtyPaths meets — modified, added, untracked, deleted —
// arrives on one record with one path, and any edit under core/ produces one.
// A rename does not: `git status --porcelain -z` writes the DESTINATION on the
// record carrying the status letters and the SOURCE on the record that follows
// it, unprefixed. That second record is a branch nothing had ever taken. The
// hole it fills is real — the source is at HEAD and gone from disk, the
// destination is the other way round, so each of them is a path exactly one of
// the two walks reads — and a run in which the source is not held out reports
// it as `ls-tree` having found a file InDir did not, which is the file filters
// having come apart. The failure would be loud, wrong, and about the wrong
// thing.
//
// Worse than losing it, in fact. The source record has no status prefix, so a
// loop that does not consume it reads its first two bytes AS one:
// "core/from.go" becomes the status "co" and the path "e/from.go", which
// relativises to "../e/from.go". The branch is not the difference between
// holding a path out and missing it — it is the difference between holding it
// out and inventing one.
//
// End to end that needs a real rename, which is a commit and a clone:
//
//	git clone . /tmp/renamed-core && cd /tmp/renamed-core
//	git mv core/alignment.go core/alignment2.go
//	go test ./internal/themehistory -run TestTheRevisions -v
//
// The arm is then over 48 of 49 files with both alignment.go and
// alignment2.go named as the hole. What that costs is a clone per run, for a
// branch whose whole content is "read the next record too" — so the recipe is
// written down and the BRANCH is taken here, over the format itself, where
// every shape one record can have is in one table and none of them needs git.
func TestARenameUnderThePackageHoldsOutBothOfItsPaths(t *testing.T) {
	// Written as records rather than as one string with \x00 in it, so the
	// terminator is applied uniformly and a missing one is not a typo that
	// silently merges two cases. git terminates every record, including the
	// last, which is why Split leaves a trailing empty string that dirtyPaths
	// has to survive.
	records := []string{
		" M core/theme.go",  // work tree modified
		"M  core/colors.go", // staged
		"?? core/new.go",    // untracked
		"D  core/gone.go",   // deleted
		" M core/a name.go", // the reason -z is asked for
		"?? core/zsub/",     // an untracked directory, collapsed by git
		"R  core/to.go",     // a rename: the destination
		"core/from.go",      // and its source, on its own record
		"C  core/copy.go",   // a copy, which has the same two-record
		"core/origin.go",    // shape and the same handling
		// Last, and after both two-record shapes, so a second record that
		// was consumed wrongly would show up here as a miss.
		"?? core/last.go",
	}
	z := strings.Join(records, "\x00") + "\x00"

	want := []string{"a name.go", "colors.go", "copy.go", "from.go", "gone.go",
		"last.go", "new.go", "origin.go", "theme.go", "to.go", "zsub"}
	got := dirtyPaths(z)
	names := make([]string, 0, len(got))
	for p := range got {
		names = append(names, p)
	}
	slices.Sort(names)
	if !slices.Equal(names, want) {
		t.Errorf("`git status --porcelain -z` over %d record(s) reads as %d "+
			"dirty path(s), and %d were expected.\n\n"+
			"only read:      %s\nonly expected:  %s\n\n"+
			"Both halves of a rename and of a copy are meant to be here: the "+
			"source is at HEAD and gone from the working tree and the "+
			"destination is the other way round, so each is a path exactly one "+
			"of the two walks reads. A source that goes missing from this set is "+
			"a file the revision walk finds and themeleaves.InDir does not, "+
			"which the arm above reports as the two file filters having come "+
			"apart.",
			len(records), len(names), len(want),
			nameList(missing(names, want)), nameList(missing(want, names)))
	}
}

// A path underPkg cannot make relative to themePkg is kept whole.
//
// # The branch, and why it is loud on purpose
//
// underPkg keys both walks on the path relative to core/, which is what makes
// them comparable at all: git's paths are repository-relative and InDir's are
// joined onto the directory it was handed. filepath.Rel can fail — an absolute
// path against a relative base is the reachable case — and the choice there is
// between two silences and one noise:
//
//	fall back to the base name   "/elsewhere/theme.go" becomes "theme.go" and
//	                             matches core/theme.go, a file it is not
//	drop it                      one walk's list is short and the other's
//	                             names a file, in a message about filters
//	keep it whole                it matches nothing, and the file-set arm
//	                             names it exactly as it was handed over
//
// The third is the one that survives being read: the arm prints the path, and
// a reader who sees an absolute path in a list of names relative to core/ has
// the whole finding in front of them. It is still loud inside a message about
// two file filters having come apart, which is NOT what an unresolvable path
// means — nothing in this repository has ever produced one — so what this test
// pins is the choice rather than the wording.
func TestAPathThatCannotBeMadeRelativeIsKeptWhole(t *testing.T) {
	// The premise first. An absolute path against the relative base themePkg
	// is what filepath.Rel refuses on every platform this builds for — it
	// cannot know how many levels up "core" sits from the root — and if some
	// future Rel started answering it, this test would be pinning a branch
	// that no longer exists rather than failing.
	unresolvable := filepath.Join(string(filepath.Separator), "elsewhere",
		"theme.go")
	if _, err := filepath.Rel(themePkg, unresolvable); err == nil {
		t.Fatalf("filepath.Rel(%q, %q) now resolves, so underPkg's error branch "+
			"is no longer reachable by this shape and the test below is over "+
			"nothing. Find the shape that still reaches it, or take the branch "+
			"out.", themePkg, unresolvable)
	}

	// Handed alongside the file it must not be confused with. The whole point
	// of keeping it whole is that it does not collide with core/theme.go, so
	// the collision is what the assertion is over.
	kept, held := underPkg([]string{
		filepath.ToSlash(unresolvable),
		themePkg + "/theme.go",
	}, map[string]bool{})

	want := []string{filepath.ToSlash(unresolvable), "theme.go"}
	slices.Sort(want)
	if !slices.Equal(kept, want) {
		t.Errorf("underPkg keyed a path it cannot make relative to %s/ as %s, "+
			"and it is meant to keep it whole (%s).\n\n"+
			"A fall back onto the base name would key %q as \"theme.go\", which "+
			"is the one outcome this branch exists to prevent: it would match "+
			"%s/theme.go, a different file, and the two walks would agree about "+
			"a pair that has nothing to do with each other.",
			themePkg, nameList(kept), nameList(want), unresolvable, themePkg)
	}
	if len(held) != 0 {
		t.Errorf("underPkg held out %s against an empty dirty set, and nothing "+
			"can be held out of a comparison when HEAD and the working tree "+
			"differ about no file at all.", nameList(held))
	}
}

// A path with a newline in it goes round the batch, and comes back whole.
//
// # Why this branch had never run, and why it now can
//
// blob sends a path holding a newline to a `cat-file -p` of its own, because
// the batch protocol is newline-terminated. That guard had never fired, and
// not because such a path is exotic — git tracks one happily — but because
// nothing was ever handing it one: `ls-tree --name-only` C-QUOTES a name it
// cannot write literally, so a file called `a<LF>b.go` arrived as the eleven
// characters `"a\nb.go"`, which holds a backslash and an `n`. The guard looked
// for a newline in a spelling that no longer had one.
//
// treePaths asks for -z now, so the path arrives as git holds it, and this is
// the arm over what happens next.
//
// # What the batch does with one, which is the reason for the branch
//
// Not a failed request — a desynchronised stream. git reads the newline as the
// end of one object name and the rest as the start of another, so one request
// draws TWO responses:
//
//	written:  HEAD:a<LF>b.go<LF>
//	read:     HEAD:a missing<LF>     ← this one is returned as the error
//	          b.go missing<LF>       ← nobody asked; nobody reads it
//
// The second line is then the answer to the NEXT request, and every response
// after that belongs to the request before it. That is the failure this file
// spends its longest comment on, arriving through a filename.
//
// So the assertion is in two halves: the bytes come back, and the reader that
// was NOT used is still usable — because a run where the fallback silently
// stopped firing would pass the first half on the very request that broke the
// second.
func TestAPathWithANewlineInItGoesRoundTheBatch(t *testing.T) {
	repo := scratchRepo(t, map[string]string{
		"core/theme.go": "package core\n\ntype Theme struct{ A string }\n",
		"core/a\nb.go":  "package core\n\ntype B struct{ C string }\n",
	})
	t.Chdir(repo)

	// The batch is used first and used after, so the file in the middle is the
	// only thing that could have moved it.
	before, err := blob("HEAD", "core/theme.go")
	if err != nil {
		t.Fatalf("HEAD:core/theme.go cannot be fetched out of a repository this "+
			"test just built: %v", err)
	}

	odd := "core/a\nb.go"
	got, err := blob("HEAD", odd)
	if err != nil {
		t.Fatalf("blob could not read %q, a path git tracks and `cat-file -p` "+
			"resolves: %v\n\n"+
			"This path holds a real newline, so it must not go down the batch: "+
			"git would read it as two object names and answer one request with "+
			"two `missing` lines.", odd, err)
	}
	if want := "package core\n\ntype B struct{ C string }\n"; got != want {
		t.Errorf("%q reads as %q and it was written as %q.", odd, got, want)
	}

	// And the half that says the fallback was actually taken. If the request
	// had gone down the batch, one unread `missing` line would be sitting in
	// the stream and this fetch would be answered by it.
	after, err := blob("HEAD", "core/theme.go")
	if err != nil {
		t.Fatalf("after fetching %q, HEAD:core/theme.go could not be read: %v\n\n"+
			"The stream is desynchronised, which is what happens when a path "+
			"holding a newline is written into a newline-terminated protocol: "+
			"the second of git's two answers is still in the pipe.", odd, err)
	}
	if after != before {
		t.Errorf("HEAD:core/theme.go reads as %q after fetching %q and %q "+
			"before it.\n\nOne request drew two responses and the second is "+
			"answering this one.", after, odd, before)
	}
}

// The batch is answered out of the repository the CALLER is in, not the one
// the process was born in.
//
// # The state this is about
//
// `git cat-file --batch` resolves `<rev>:<path>` from the top of the tree, so
// nothing about a fetch depends on the working directory — except which
// repository git DISCOVERS, which is all of it. The batch process is
// long-lived and its directory was fixed at the moment it started, and every
// test in this file happens to t.Chdir to this repository's root before
// fetching anything. t.Chdir puts the directory back afterwards; the git
// process keeps the one it was born in.
//
// So "benign" rested on the order two tests run in. A test that built a
// throwaway repository and fetched from it would be answered out of grmob —
// and the kindest way that ends is `missing`, because the paths would not
// resolve. The unkind way is a path that exists in both.
//
// blob compares the directory on every fetch and retires the reader when it
// has moved, which is what this arm is over: the same relative path, two
// repositories, two different files.
func TestTheBatchFollowsTheCallerIntoAnotherRepository(t *testing.T) {
	// A file at a path grmob also has, holding text grmob does not. Read out
	// of the wrong repository this comes back as core.Theme's real source.
	const decoy = "package core\n\ntype Theme struct{ Scratch string }\n"
	repo := scratchRepo(t, map[string]string{"core/theme.go": decoy})

	// The first fetch, taken from THIS repository, is what starts the reader
	// and fixes its directory. Without it the reader would be started inside
	// the scratch repo and the arm below would pass for the wrong reason.
	root, err := git("rev-parse", "--show-toplevel")
	if err != nil {
		t.Skipf("this checkout is not a git repository: %v", err)
	}
	func() {
		t.Chdir(strings.TrimSpace(root))
		if _, err := blob("HEAD", path.Join(themePkg, "theme.go")); err != nil {
			t.Skipf("HEAD:%s/theme.go cannot be read, so this is not the "+
				"repository %s.%s is declared in: %v", themePkg, themePkg,
				themeType, err)
		}
	}()

	t.Chdir(repo)
	got, err := blob("HEAD", "core/theme.go")
	if err != nil {
		t.Fatalf("after moving into another repository, HEAD:core/theme.go "+
			"could not be read: %v\n\n"+
			"The batch process still has the directory it was started in, so "+
			"it is answering out of a repository this path may not exist in.",
			err)
	}
	if got != decoy {
		t.Errorf("HEAD:core/theme.go read from a repository this test built "+
			"came back as %d byte(s) that are not the %d it was written with.\n\n"+
			"got:\n%s\nwant:\n%s\n"+
			"A `git cat-file --batch` resolves a revision from the top of the "+
			"tree it DISCOVERED, and it discovers one from the directory it was "+
			"started in. This answer came out of the repository the process was "+
			"born in rather than the one the caller is standing in.",
			len(got), len(decoy), got, decoy)
	}
}

// scratchRepo is a git repository with the given files committed, in a
// directory this test owns.
//
// # Why a real repository and not a fixture
//
// Both callers are about what GIT does with a path — a name it will not write
// literally, and a working directory it resolves a repository from. Neither
// question has an answer that can be written down: it is git's, it is the
// installed git's, and a fixture would be this file asserting its own belief
// about the tool it is checking.
//
// Committed with -c rather than by writing a config, so a machine whose global
// git has no identity (a container, a CI image) runs this the same way one
// with an identity does. `init.defaultBranch` likewise: the branch name is
// never used — everything is fetched at HEAD — and setting it silences a hint
// that would otherwise land on this process's stderr.
func scratchRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git on this machine, so there is no repository to build.")
	}
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("`git %s` failed while building a scratch repository, so "+
				"this machine's git cannot answer the question below: %v: %s",
				strings.Join(args, " "), err, strings.TrimSpace(string(out)))
		}
	}
	run("-c", "init.defaultBranch=main", "init", "-q", ".")
	for rel, src := range files {
		// filepath, not path: these are names on THIS machine, unlike
		// everything git hands back.
		full := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("cannot make the directory for %q: %v", rel, err)
		}
		if err := os.WriteFile(full, []byte(src), 0o644); err != nil {
			// A filesystem that will not hold the name is a skip and not a
			// failure: the newline case is legal on every platform this
			// package is built for and illegal on some it is not.
			t.Skipf("this filesystem will not hold a file called %q: %v", rel, err)
		}
	}
	run("add", "-A")
	run("-c", "user.email=themehistory@example.invalid", "-c",
		"user.name=themehistory", "commit", "-q", "-m", "scratch")
	return dir
}

// Every shape a batch response can arrive in is told apart, and the one that
// leaves the stream usable is told apart from the four that do not.
//
// # Why this is a table over strings and not five states of a git
//
// readResponse is a function of the BYTES. Nothing it decides depends on the
// process behind them, and four of the five shapes below cannot be produced by
// asking a working git for anything: a header that is not three fields, a size
// that will not parse, a body that stops early, a pipe that ends mid-response.
// Held as a table they are five rows; held as an integration test they are
// four gits nobody can make misbehave on purpose and one that can.
//
// What is asserted is the CLASSIFICATION and not the wording — errMissing for
// the complete one-line answer, errDesync for everything else — because that
// is what blob branches on. A message that got rephrased would fail a test
// pinned to prose and would be asserting that nobody had edited a paragraph;
// a `missing` reclassified as a desync would cost a process per absent object,
// and a desync reclassified as `missing` would put the whole rest of a run at
// an offset nobody knows.
func TestEveryBatchResponseShapeIsToldApart(t *testing.T) {
	// The body is the same six bytes wherever a row has one, so a row's
	// subject is the only thing that differs between it and the good case.
	const body = "package"
	good := fmt.Sprintf("%040x blob %d\n%s\n", 1, len(body), body)

	for _, c := range []struct {
		what   string
		stream string
		want   error // errMissing, errDesync, or nil for a whole object
		why    string
	}{
		{
			what:   "a whole object",
			stream: good,
			want:   nil,
			why: "the header, the counted body and the terminator that is not " +
				"counted in it",
		},
		{
			what:   "missing",
			stream: "HEAD:nope missing\n" + good,
			want:   errMissing,
			why: "git's complete one-line answer for an object it cannot " +
				"resolve. Nothing follows it, so the stream is left at the " +
				"start of the next header and the caller after this one is " +
				"answered correctly — which is what the second response in " +
				"this stream is here to make checkable.",
		},
		{
			what:   "a header that is not a header",
			stream: "fatal: not a git repository\n",
			want:   errDesync,
			why: "three fields by count and none of them a size. Anything " +
				"that is not <oid> <type> <size> and not a `missing` line " +
				"means the reader does not know what it is looking at.",
		},
		{
			what:   "a size that will not parse",
			stream: fmt.Sprintf("%040x blob seven\n%s\n", 1, body),
			want:   errDesync,
			why: "the body is read by COUNT, so a size that is not a number " +
				"is not a body that can be skipped",
		},
		{
			what:   "a body that stops early",
			stream: fmt.Sprintf("%040x blob 99\n%s\n", 1, body),
			want:   errDesync,
			why: "the header promised 99 bytes and the stream held seven. " +
				"Whatever is read next is not a header.",
		},
		{
			what:   "a stream that ends before the terminator",
			stream: fmt.Sprintf("%040x blob %d\n%s", 1, len(body), body),
			want:   errDesync,
			why: "the object is whole and the byte after it is not there. " +
				"This is the shape that would go unnoticed: the caller has " +
				"its file, and the reader is one byte into a header it will " +
				"never read correctly again.",
		},
	} {
		t.Run(c.what, func(t *testing.T) {
			r := bufio.NewReader(strings.NewReader(c.stream))
			got, err := readResponse(r, "HEAD:core/theme.go")
			switch {
			case c.want == nil:
				if err != nil {
					t.Fatalf("%s reads as an error: %v\n\n%s", c.what, err, c.why)
				}
				if got != body {
					t.Errorf("%s reads as %q and the body was %q.\n\n%s",
						c.what, got, body, c.why)
				}
			case !errors.Is(err, c.want):
				t.Fatalf("%s reads as %v, and it is %v.\n\n%s\n\n"+
					"blob branches on exactly this: errMissing costs the "+
					"caller its file and nothing else, and errDesync retires "+
					"the process. Confusing the two either pays a git per "+
					"absent object or reads the rest of the run out of a "+
					"stream at an unknown offset.",
					c.what, err, c.want, c.why)
			}
			if c.want != errMissing {
				return
			}
			// And the half a classification cannot state: after a `missing`
			// the stream really is where the next request expects it.
			next, err := readResponse(r, "HEAD:core/theme.go")
			if err != nil || next != body {
				t.Errorf("the response after a `missing` reads as (%q, %v), "+
					"and it is a whole object.\n\nA `missing` line is the "+
					"complete response. A reader that consumed anything "+
					"behind it would answer this request out of the middle "+
					"of something else.", next, err)
			}
		})
	}
}

// A batch whose process has gone is replaced, not read from.
//
// # The state, and why it is reachable
//
// Every error but `missing` leaves the stream at an offset nobody knows, and
// the reader was a package-level value that lived for the run: one such
// response and every later fetch in the process read from the wrong place,
// with nothing anywhere saying so. A walk carrying on would be handing
// go/parser text that begins in the middle of another file, and a short
// expansion reads as a commit that removed leaves.
//
// The process dying is the one shape of that a test can actually make — a
// crash, an OOM kill, a `git` that exits on its own — and it exercises the
// whole path rather than a fake: the read meets a closed pipe, the error
// carries errDesync, blob sees `dead` and retires, and a fresh git answers the
// next request. What it costs is one process, which is the trade this file
// takes deliberately: the batch was bought to save four thousand of them, and
// paying one back to leave a known-bad state is cheaper than any table
// assembled out of misaligned bytes.
func TestAKilledBatchIsReplacedRatherThanReadFrom(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git on this machine, so there is no batch to kill.")
	}
	root, err := git("rev-parse", "--show-toplevel")
	if err != nil {
		t.Skipf("this checkout is not a git repository: %v", err)
	}
	t.Chdir(strings.TrimSpace(root))

	real := path.Join(themePkg, "theme.go")
	want, err := blob("HEAD", real)
	if err != nil {
		t.Skipf("HEAD:%s cannot be read, so this is not the repository %s.%s "+
			"is declared in: %v", real, themePkg, themeType, err)
	}
	was := blobs.cmd.Process.Pid

	// A `missing` first, which is the error that must NOT cost a process.
	// Without this the arm below would pass for a reader that retired itself
	// on every error, and absent objects are the reachable ones.
	absent := path.Join(themePkg, "no-such-file-a4232825.go")
	if _, err := blob("HEAD", absent); !errors.Is(err, errMissing) {
		t.Fatalf("HEAD:%s reads as %v and it is meant to be errMissing. Either "+
			"that file now exists — rename it in this test — or a complete "+
			"one-line response is being read as something else.", absent, err)
	}
	if blobs.cmd.Process.Pid != was {
		t.Errorf("a missing object cost a git process: the batch was %d and is "+
			"now %d.\n\nA `missing` line is a COMPLETE response, so the stream "+
			"is still usable and there is nothing to recover from. Retiring "+
			"here would pay a process per absent object.",
			was, blobs.cmd.Process.Pid)
	}

	// And now the shape that is not recoverable.
	if err := blobs.cmd.Process.Kill(); err != nil {
		t.Skipf("this batch process cannot be killed, so the state below "+
			"cannot be reached: %v", err)
	}
	if _, err := blob("HEAD", real); !errors.Is(err, errDesync) {
		t.Fatalf("reading from a batch whose process has been killed reads as "+
			"%v, and it is meant to carry errDesync.\n\nThat is what marks the "+
			"reader dead: a pipe that ended mid-stream leaves the position "+
			"unknown, and every response after it would belong to some other "+
			"request.", err)
	}
	if blobs.dead == nil {
		t.Fatalf("the batch answered with a desynchronising error and was not " +
			"marked dead, so the next fetch would read one more byte out of a " +
			"stream whose position is a guess.")
	}

	// The recovery, which is the point: the caller after the failure gets its
	// own file out of a process that is not the broken one.
	got, err := blob("HEAD", real)
	if err != nil {
		t.Fatalf("after a killed batch, HEAD:%s could not be read: %v\n\nA dead "+
			"reader is meant to be retired and replaced, which costs one git "+
			"process and recovers the run.", real, err)
	}
	if got != want {
		t.Errorf("after a killed batch, HEAD:%s reads as %d byte(s) against the "+
			"%d it read before; first difference at byte %d.", real, len(got),
			len(want), firstDiff(got, want))
	}
	if blobs.cmd.Process.Pid == was {
		t.Errorf("the batch is still process %d, which was killed. Whatever "+
			"answered the fetch above did not come from it.", was)
	}
}

// Every place a collision can sit is told apart.
//
// # Why the placement needed a table
//
// themeleaves.Of records a bare name it parsed more than once WHEREVER the
// declarations were, and both readers of that record — the command's stderr
// line and reportNested's sentence — described every one of them as "declared
// both in core/ and below it". That is the union probe's case. It is the case
// this repository would actually meet, which is exactly why the other three
// went unnoticed: nothing here has ever produced one, so nothing has ever read
// the sentence and found it describing the wrong directory.
//
// A reader who meets this message is reading it because a population moved and
// they want to know which declaration answered. The sentence is how they
// decide where to look, and a sentence that names the wrong place is worse
// than one that names none.
//
// # What is asserted, and what is not
//
// Which SHAPE each row is recognised as — the distinguishing phrase — plus the
// two facts every row must carry: the paths, and the winner named last. Not
// the whole wording, which would fail on any rephrasing and would be asserting
// that nobody had edited a paragraph.
func TestEveryPlaceACollisionCanSitIsToldApart(t *testing.T) {
	for _, c := range []struct {
		what  string
		files []string
		// The phrase that distinguishes this shape from the other four.
		says string
		why  string
	}{
		{
			what:  "the top level and below it",
			files: []string{themePkg + "/theme.go", themePkg + "/zsub/extra.go"},
			says:  "both directly in " + themePkg + "/ and below it",
			why: "two packages, on purpose: this is the union probe, which " +
				"hands Of the top level AND a subdirectory to find out whether " +
				"the files leavesAt declines move the population",
		},
		{
			what:  "two directories below the top level",
			files: []string{themePkg + "/a/x.go", themePkg + "/b/x.go"},
			says:  "2 different directories below " + themePkg + "/",
			why: "two packages, neither of them " + themePkg + "/. Nothing " +
				"here reads both today, and the old sentence would have sent a " +
				"reader to the top level to look for a declaration that is not " +
				"there",
		},
		{
			what:  "twice in the top level",
			files: []string{themePkg + "/a.go", themePkg + "/b.go"},
			says:  "directly in " + themePkg + "/, which is one package",
			why: "ONE package declaring one name twice. It does not compile — " +
				"and this walk parses every commit that touched " + themePkg +
				"/, including ones caught mid-refactor, which is the whole " +
				"reason a revision that would not build is reported rather " +
				"than dropped",
		},
		{
			what:  "twice in one directory below the top level",
			files: []string{themePkg + "/sub/a.go", themePkg + "/sub/b.go"},
			says:  "in " + themePkg + "/sub, which is one package below",
			why: "the same state one directory down, and the reason the " +
				"top-level arm cannot simply be \"is it core/ or not\"",
		},
		{
			what:  "twice in one file",
			files: []string{themePkg + "/theme.go", themePkg + "/theme.go"},
			says:  "in ONE file",
			why: "go/parser reads a file that declares a name twice and the " +
				"compiler does not accept it. This is why Shadow.Files is " +
				"recorded as parsed rather than deduplicated: collapsing it " +
				"would make this row print as the one above",
		},
	} {
		t.Run(c.what, func(t *testing.T) {
			got := shadowKind(themeleaves.Shadow{Name: "SpacingScale",
				Files: c.files})
			if !strings.Contains(got, c.says) {
				t.Errorf("a name declared in %s reads as:\n\n  %s\n\nand it is "+
					"%s, which no phrase in that sentence says (%q).\n\n%s\n\n"+
					"A reader meets this message because a population moved "+
					"and they want the declaration the expansion was taken "+
					"over. The sentence is how they decide where to look.",
					nameList(c.files), got, c.what, c.says, c.why)
			}
			// And the two facts every shape carries, whatever it is called.
			// The winner last is what Shadow.Files is ordered for, and a
			// message that named the paths in any other order would be
			// pointing at the declaration that did NOT answer.
			for _, p := range c.files {
				if !strings.Contains(got, p) {
					t.Errorf("%q declared it and the sentence does not name "+
						"that file:\n\n  %s", p, got)
				}
			}
			last := c.files[len(c.files)-1]
			if i := strings.LastIndex(got, last); i < 0 ||
				strings.Contains(got[:i], "answered") {
				t.Errorf("the sentence for %s does not end its list with %q, "+
					"which is the declaration that answered:\n\n  %s\n\n"+
					"themeleaves.Of sorts its paths and a later declaration "+
					"overwrites an earlier one, so the LAST path is the one "+
					"the expansion was taken over. Any other order points at "+
					"the declaration that lost.", c.what, last, got)
			}
		})
	}
}
