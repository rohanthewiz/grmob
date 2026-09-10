package main

import (
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
		compared = false
		headSub, treeSub, err := expansionsOver(headFiles)
		switch {
		case err != nil:
			t.Errorf("the %d file(s) both walks reach and both trees agree "+
				"about cannot be re-read to compare the expansions over them: "+
				"%v\n\nEvery one of those paths was listed by `ls-tree` at HEAD "+
				"and read off disk by themeleaves.InDir in this same run, so a "+
				"fetch that fails here is the tree having moved underneath this "+
				"test rather than a finding about either walk.",
				len(headFiles), err)
		case !headSub.Found && !treeSub.Found:
			// Two empty populations comparing equal is not evidence, and this
			// is the shape a dirty theme.go produces: the file the struct is
			// declared in is held out, so neither re-reading finds it. The
			// remaining files are still a comparison of the FILTERS, which ran
			// above; they are not a comparison of the expansion.
			t.Logf("the expansions were re-taken over the %d file(s) the two "+
				"trees agree about and neither reading declares %s.%s in them, "+
				"so a name-for-name comparison would be between two empty "+
				"populations. The %d held-out file(s) (%s) include the one that "+
				"declares it. The file FILTERS were still compared, over the "+
				"same %d. Commit or stash to get the whole reading back.",
				len(headFiles), themePkg, themeType, len(held),
				strings.Join(held, ", "), len(headFiles))
		case headSub.Found != treeSub.Found:
			t.Errorf("over the same %d file(s) — the ones HEAD and this working "+
				"tree agree about — %s is declared at HEAD: %v, and in the "+
				"working tree: %v.\n\nThe text is the same text for every one of "+
				"those paths, so one reading finding the struct and the other "+
				"not is `git cat-file` and the disk disagreeing about a file "+
				"neither `git status` nor either walk calls different.",
				len(headFiles), themeType, headSub.Found, treeSub.Found)
		default:
			headNames, treeNames = headSub.Names, treeSub.Names
			subset, compared = true, true
		}
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
// Cost is one `git cat-file` per kept file on a dirty tree, which is the same
// fetch leavesAt already did for HEAD once; what it buys is an arm that runs
// while somebody is editing core/, which is when a filter moves.
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
		src, err := git("cat-file", "-p", "HEAD:"+key)
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
// subpackage really contributes, or a name it shadowed — and the union cannot
// separate them. It is reported as a thing to look at, with both directions
// printed, rather than as the population.
func reportNested(t *testing.T, atHead revision) {
	t.Helper()
	union, err := unionWithNested(atHead)
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
			"directory that has more in it.",
			len(atHead.nested), themePkg, strings.Join(atHead.nested, ", "),
			themePkg, len(atHead.Names), themePkg, themeType)
	default:
		t.Errorf("git tracks %d non-test .go file(s) under %s/ at HEAD that are "+
			"not directly in it — %s — and parsing them alongside %s/'s own "+
			"files MOVES the population: %d name(s) against the %d every row of "+
			"the edit-size table is over.\n\n"+
			"only the union's:      %s\nonly the top level's:  %s\n\n"+
			"Two things produce that and this arm cannot tell them apart. Either "+
			"those files declare something %s.%s reaches, and the table is short "+
			"by it — in which case the reading needs the subpackage, which is a "+
			"different expansion and not a wider glob. Or they declare a type "+
			"whose BARE NAME the top level already uses: themeleaves.Of resolves "+
			"field types by bare name against one flat map of everything it "+
			"parsed, so two packages in one map let a file below %s/ answer for "+
			"a field in %s/, or lose to it, on nothing better than the order "+
			"their paths sort in. That is what "+
			"the one-package-one-directory rule is for, and it is why the union "+
			"above is a probe rather than the new population.\n\n"+
			"Both walks still decline these files identically, so the file-set "+
			"comparison stays green and this is the only arm that can say so.",
			len(atHead.nested), themePkg, strings.Join(atHead.nested, ", "),
			themePkg, len(union.Names), len(atHead.Names),
			nameList(missing(union.Names, atHead.Names)),
			nameList(missing(atHead.Names, union.Names)),
			themePkg, themeType, themePkg, themePkg)
	}
}

// unionWithNested is themePkg's expansion at HEAD re-taken with the files
// below it included — the reading leavesAt declined, taken once so the arm
// above can say whether declining it cost anything.
//
// Both halves are fetched rather than one being reused: leavesAt hands back
// the expansion and not the text it was over, and re-fetching what it already
// read costs a `cat-file` per file on a run that is by definition rare (this
// repository has never had a revision with anything below core/). Paying it
// here keeps revision — and the command — free of a sources map that exists
// for a test.
func unionWithNested(rev revision) (themeleaves.Expansion, error) {
	sources := make(map[string]string, len(rev.Files)+len(rev.nested))
	for _, p := range append(append([]string{}, rev.Files...), rev.nested...) {
		src, err := git("cat-file", "-p", "HEAD:"+p)
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
		"last.go", "new.go", "origin.go", "theme.go", "to.go"}
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
