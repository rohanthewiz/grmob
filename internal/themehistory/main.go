// Command themehistory reads how core.Theme's leaf population has moved,
// commit by commit, and prints the distribution of edit sizes.
//
// # Why this is a program and not a test
//
// wasm/verify/themenearmiss_test.go measures a band for each step in
// affordedBandSteps — every population one leaf from the record's, every
// population two from it, three — and stops at three because four costs
// 685ms. That bound used to be argued from a guess ("the fourth simultaneous
// field edit is a rarer thing than the third"), and the argument it has now is
// a distribution: how often core.Theme has actually moved one, two, three or
// more leaves in a single commit. Both halves of the guess turned out to be
// wrong.
//
// The reading was taken by a throwaway AST walker in a scratch directory, and
// what survived into the test was the numbers, the method in a paragraph, and
// the commit it was taken at. That is weaker than the convention the rest of
// that file has moved to: the record's census is re-walked on every run
// precisely because a number in a note is a number that has already moved. A
// test cannot shell out to git reliably — a shallow clone, a source tarball or
// a build container has no history to read — so the reading cannot become an
// arm. It can stop being a re-derivation:
//
//	go run ./internal/themehistory
//	go run ./internal/themehistory -names   (and the population at HEAD)
//
// # What it reads
//
// For every commit that touches core/, the package's sources at that revision
// are parsed and Theme is expanded to its distinct leaf NAMES — the same
// population wasm/verify's affordedLeafNames produces, by the same rule
// (recurse into struct-typed fields, take the last dotted segment, keep each
// name once). Each commit is then diffed against its predecessor.
//
// The expansion is syntactic and affordedLeafNames' is reflective, which is
// the one place the two could disagree: a field whose type this walker cannot
// resolve to a struct declared in core/ is treated as a leaf, where reflect
// would have recursed into it. That is not hypothetical — the first version
// of this walker stopped at Typography.Caption because it had already
// descended through Typography.Body and both are TextStyle, and it came back
// with ninety-one names rather than eighty. See internal/themeleaves, which is
// where the expansion now lives and which draws the shape of that bug.
//
// # And that agreement is an arm now, rather than a reading somebody took once
//
// It used to be checked by hand: run this with -names, diff the output against
// affordedLeafNames', believe the table. That is one reading of two things that
// both move, which is the shape this whole tool exists to get away from.
//
// The git half still cannot be a test — a shallow clone, a source tarball or a
// build container has no history to read. The EXPANSION half never needed git:
// themeleaves.InDir over a working tree is a pure parse of core/, and
// wasm/verify's TestTheHistoryWalkersExpansionIsTheOneThisFileMeasures holds it
// name-for-name against affordedLeafNames() on every run. What this command
// still owes a reader is the half that arm cannot see — that the reading it
// takes AT A REVISION is that same expansion — and the way it says so is to
// print its own HEAD, which is the one revision both mechanisms can read.
//
// The counts agreeing is the weak form of that check: two different sets of
// eighty print the same 80. -names prints the population itself.
//
// # And the filter above the expansion is an arm too, at the one revision that
// has a working tree
//
// leavesAt decides which of a revision's files reach themeleaves — a `.go`
// suffix test, a `_test.go` exclusion and, since `ls-tree -r` descends where
// themeleaves.InDir does not, a top-level test. The first two also exist
// inside themeleaves.Of, and the arm in wasm/verify is downstream of all
// three: it hands InDir a directory and never goes through leavesAt at all, so
// a filter that drifted here would move every row of the table and leave that
// test green.
//
// main_test.go compares the two file sets at HEAD, which is the one revision
// git and the working tree both describe. It skips only where there is no
// repository to read — and that is the one place in this repository a skipping
// git test is honest, because the git half IS the subject. A working tree that
// differs from HEAD under core/ is not a skip but a HOLE: `git status
// --porcelain` names the paths the two readings would take out of two
// different trees, those are held out by name, and what was left out is
// reported alongside what was compared.
package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rohanthewiz/grmob/internal/themeleaves"
)

// The directory whose commits are read and whose sources are parsed. The
// struct is core.Theme and nothing outside that package contributes a field to
// it, so a commit that does not touch core/ cannot move the population.
const themePkg = "core"

// The struct the population is an expansion of.
const themeType = "Theme"

func main() {
	flag.Parse()
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "themehistory:", err)
		os.Exit(1)
	}
}

// Whether to print the leaf names at HEAD. The counts below are only worth
// reading while this walker's population is the one wasm/verify measures, and
// the count agreeing is weaker evidence than the names agreeing: two
// different sets of eighty would print the same 80.
var showNames = flag.Bool("names", false,
	"also print the distinct leaf names at HEAD, one per line")

func run() error {
	// Oldest first, so a commit is diffed against the state before it. --follow
	// is deliberately not used: it is a per-file heuristic and this is a
	// directory.
	out, err := git("log", "--reverse", "--format=%H %ad", "--date=short",
		"--", themePkg)
	if err != nil {
		return err
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return fmt.Errorf("no commits touch %s/", themePkg)
	}

	// One commit that moved the population, with what it moved.
	type step struct {
		sha, date      string
		added, removed []string
		// The size AFTER this commit, so the listing reads as a running total
		// and the last row is HEAD.
		leaves int
	}
	var steps []step
	var prev map[string]bool
	for _, line := range lines {
		sha, date, _ := strings.Cut(line, " ")
		rev, err := leavesAt(sha)
		if err != nil {
			return fmt.Errorf("%s: %w", sha[:8], err)
		}
		// Two things themeleaves hands back that used to be invisible here.
		// A revision go/parser could not fully read is missing whatever those
		// files declared, and a revision with no Theme in it at all reads as a
		// population of nothing — which the diff below would report as every
		// leaf being REMOVED, in a table whose headline finding is that nothing
		// ever has been. Both are ordinary in a history and neither is a
		// failure; they are said out loud, on stderr, so the row they produce
		// is not read as a fact about the struct.
		if len(rev.Unparsed) > 0 {
			fmt.Fprintf(os.Stderr, "themehistory: %s: go/parser read nothing "+
				"from %s — any struct declared only there is missing from this "+
				"revision's expansion\n", sha[:8], strings.Join(rev.Unparsed, ", "))
		}
		// And the files this revision held under core/ that are not directly
		// in it. Same shape of finding as Unparsed and the same reason for
		// saying it here: the row below is over a population that is short by
		// whatever those files declare, and nothing downstream of this loop
		// could tell that from a revision where the struct was simply smaller.
		if len(rev.nested) > 0 {
			fmt.Fprintf(os.Stderr, "themehistory: %s: %d file(s) tracked under "+
				"%s/ are not directly in it and were not parsed: %s — one package "+
				"is one directory, which is the rule themeleaves.InDir reads the "+
				"working tree by, so any type declared only in there is missing "+
				"from this revision's expansion\n", sha[:8], len(rev.nested),
				themePkg, strings.Join(rev.nested, ", "))
		}
		// And a name this revision declared more than once, which themeleaves
		// resolved by sort order over the paths. Same shape of finding again:
		// the row below is over ONE of two declarations, chosen by a string
		// comparison, and every other reading of this revision would have been
		// over the same one and looked equally whole.
		//
		// Inside one package that state does not build — which is exactly why
		// it is worth a line here rather than nowhere. This walk parses every
		// commit that touched core/, including ones caught mid-refactor, and a
		// revision that parses and would not compile is the same kind of state
		// as a revision go/parser could not read: ordinary in a history, not a
		// failure, and not something to let through in silence.
		//
		// Empty at all 88 revisions of this repository, so this prints nothing
		// today and the table is unchanged by its existence.
		for _, sh := range rev.Shadowed {
			fmt.Fprintf(os.Stderr, "themehistory: %s: %s is %s and themeleaves "+
				"resolved it to the last of them by path order — any field of "+
				"that type in this revision's expansion is the reading of ONE "+
				"of the declarations\n", sha[:8], sh.Name, shadowKind(sh))
		}
		if !rev.Found {
			fmt.Fprintf(os.Stderr, "themehistory: %s: %s/ declares no %s at this "+
				"revision, so its population is empty and the next commit's row "+
				"reads as a creation\n", sha[:8], themePkg, themeType)
		}
		names := leafSet(rev.Expansion)
		// A commit that touched core/ without moving the population — a
		// comment, a method, a rename of something that is not a leaf — is not
		// a step. Only the commits that MOVED it are what the distribution is
		// over.
		var added, removed []string
		for name := range names {
			if !prev[name] {
				added = append(added, name)
			}
		}
		for name := range prev {
			if !names[name] {
				removed = append(removed, name)
			}
		}
		sort.Strings(added)
		sort.Strings(removed)
		if len(added) > 0 || len(removed) > 0 {
			steps = append(steps, step{sha: sha, date: date, added: added,
				removed: removed, leaves: len(names)})
		}
		prev = names
	}

	// The per-commit reading, which is what makes the histogram checkable.
	fmt.Printf("%d commit(s) touching %s/ moved %s.%s's leaf population:\n\n",
		len(steps), themePkg, themePkg, themeType)
	for i, s := range steps {
		what := fmt.Sprintf("+%d", len(s.added))
		if len(s.removed) > 0 {
			what += fmt.Sprintf(" -%d", len(s.removed))
		}
		moved := append(append([]string{}, s.added...), s.removed...)
		note := ""
		if i == 0 {
			note = "  (the commit that created it)"
		}
		fmt.Printf("  %s  %s  %-7s -> %3d leaves%s\n", s.sha[:8], s.date, what,
			s.leaves, note)
		fmt.Printf("            %s\n", wrap(moved, 66, "            "))
	}

	// And the distribution the band steps are argued from. The bootstrap
	// commit is counted like any other and marked, because "28 leaves at once"
	// is a real row and also obviously not a field edit.
	fmt.Printf("\nleaves moved   commits\n")
	by := map[int]int{}
	for _, s := range steps {
		by[len(s.added)+len(s.removed)]++
	}
	sizes := make([]int, 0, len(by))
	for n := range by {
		sizes = append(sizes, n)
	}
	sort.Ints(sizes)
	for _, n := range sizes {
		fmt.Printf("%12d   %7d\n", n, by[n])
	}

	// The direction, which is a finding of its own: affordedBand measures a
	// `losing` number — a population read against a LARGER one — and if
	// nothing has ever removed a leaf, that half of every band is over a
	// direction the struct has never gone. Counted as commits and not as
	// leaves, because one commit that removed thirty would still be one
	// occasion on which anybody did it.
	grew, shrank := 0, 0
	for _, s := range steps {
		if len(s.added) > 0 {
			grew++
		}
		if len(s.removed) > 0 {
			shrank++
		}
	}
	fmt.Printf("\n%d of the %d added leaves and %d removed any.\n",
		grew, len(steps), shrank)

	if *showNames {
		out := []string{}
		for n := range prev {
			out = append(out, n)
		}
		sort.Strings(out)
		fmt.Println(strings.Join(out, "\n"))
	}
	// And HEAD, which is what says the reading is measuring the population the
	// test measures. Compare against affordedMeasuredOn's `leaves`.
	if n := len(prev); n > 0 {
		fmt.Printf("HEAD expands to %d distinct leaf names — compare "+
			"affordedMeasuredOn.leaves in wasm/verify/themenearmiss_test.go.\n", n)
	}
	return nil
}

// revision is one revision's reading of core.Theme: the expansion, plus what
// the fetch above it declined before themeleaves ever saw the text.
//
// Expansion is embedded rather than held in a field because everything in it
// is still exactly what a caller wants — Names, Files, Unparsed, Found — and
// `nested` is one more thing THIS half of the reading knows and the
// working-tree half structurally cannot.
type revision struct {
	themeleaves.Expansion
	// The tracked, non-test .go files under themePkg at this revision that
	// are not directly in it: core/sub/theme.go and anything deeper.
	//
	// The two halves of this reading used to disagree about these, silently
	// and in the direction nothing could report. `ls-tree -r` descends and
	// themeleaves.InDir does not, so a revision that split core/ into
	// subdirectories was parsed WITH those files — producing a row in the
	// edit-size table — while every arm that checks this expansion runs
	// against InDir at HEAD and had nothing to notice.
	//
	//	revision:      ls-tree -r ──> core/theme.go, core/sub/theme.go
	//	working tree:  ReadDir    ──> core/theme.go
	//	                              └─ one package, one directory
	//
	// So the rule here is InDir's rule, and the files it drops are NAMED
	// rather than dropped quietly. Empty at every revision this repository
	// has — which is why adopting the rule leaves the table byte-identical —
	// and a revision that ever put one there now says so on stderr instead of
	// contributing a population nothing else could reproduce.
	nested []string
}

// leavesAt is core.Theme's leaf population at one revision.
//
// git for the sources, themeleaves for the expansion — which is the split that
// lets the expansion be a test. Everything below this line is the half that
// needs a repository; everything themeleaves does is the half that needs only
// text, and wasm/verify holds THAT half against reflect on every run.
//
// Test files are filtered before the fetch rather than left to themeleaves
// (which filters them too): a fetch per test file in core/ at every revision
// is text nobody will parse, carried down the pipe eighty-eight times. That
// cost is small now that the fetch is batched (see blob) and it was the
// difference between this walk and a noticeably slower one before, which is
// why the filter is on this side and not left to the parse.
//
// Which makes that filter a second copy of themeleaves.Of's rule, and a copy
// of a rule is a thing that can move on its own. Expansion.Files is what each
// reading says it read, and main_test.go holds this one's against InDir's at
// HEAD — see TestTheRevisionsFileSetIsTheOneTheWorkingTreeWalkReads.
//
// The filter itself is themeSourcesAt, immediately below, and it is a function
// rather than four lines in the loop here because what it decides is also the
// walk's unit of work — see its header.
//
// # Why -r is still asked for, when the descent it does is undone here
//
// A listing without it reports a subdirectory of core/ as one tree object and
// never mentions the .go files inside it, so the paths this walk declines
// would be paths it could not name. -r is how they are SEEN; revision.nested
// is where they go.
func leavesAt(sha string) (revision, error) {
	direct, nested, err := themeSourcesAt(sha)
	if err != nil {
		return revision{}, err
	}
	rev := revision{nested: nested}
	sources := map[string]string{}
	for _, p := range direct {
		src, err := blob(sha, p)
		if err != nil {
			return revision{}, err
		}
		sources[p] = src
	}
	rev.Expansion = themeleaves.Of(sources, themeType)
	return rev, nil
}

// themeSourcesAt is which of a revision's tracked files this walk will FETCH,
// and which ones it declines for being under themePkg rather than in it.
//
// # Why the filter is a function of its own
//
// It was four lines inside the loop above, and what it decides is the walk's
// unit of work: one object per path it returns, per revision. That makes it
// the answer to "how many objects does a whole run fetch", which used to be a
// number in a comment (see blob) and is now derived —
// TestTheWholeWalkGoesRoundOneBatchProcess sums this over the history and
// holds the batch reader's own fetch count to it.
//
// Extracted rather than re-implemented in that test on purpose. A test that
// spelled the suffix rule and the directory rule again would be asserting its
// own copy of them against the walk's, which is two copies of a filter and a
// check that passes while both are wrong together. Asking the walk what it
// intends to fetch and then counting what it fetched is a question about the
// FETCH, which is what the count is a claim about.
//
// The rules themselves are unchanged and both are explained above: `.go` and
// not `_test.go` because nobody will parse the rest, and `path.Dir(p) ==
// themePkg` because one package is one directory — which is InDir's rule, and
// the declined paths are named on stderr rather than dropped in silence.
//
// # Who asks, and why a green run asks twice
//
// A previous session removed a walk this program was taking twice in one run.
// This is one arriving by a different door, and it is worth naming rather than
// leaving for somebody to rediscover as the same finding:
//
//	leavesAt              once per revision, in the program. The fetch itself
//	themeSourcesAcross    once per revision, in the whole-walk arm, to have an
//	                      expectation that is not a constant
//	themeSourcesAcross    once per revision again, in the per-object arm —
//	                      which is off unless GRMOB_PER_OBJECT_FETCH is set,
//	                      so a green run pays two of these three
//
// The duplication is the point rather than an oversight: an expectation
// derived from the walk's own fetch would be the fetch checking itself, and
// the arm's whole claim is that the two agree. What it costs is one
// `git ls-tree` per commit, twice — and the test side of that is pooled, so
// the second pass is about a quarter of the first (see enumWorkers). If this
// ever reaches a third caller on a green run, the question is not how to share
// the walk but why two arms want the same expectation.
func themeSourcesAt(sha string) (direct, nested []string, err error) {
	files, err := treePaths(sha)
	if err != nil {
		return nil, nil, err
	}
	// git speaks slash-separated, repository-relative paths on every platform,
	// so `path` and not `path/filepath`: these are not paths on this machine.
	for _, p := range files {
		if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			continue
		}
		// Before the fetch, so a subdirectory of core/ costs no `cat-file`
		// either. Test files are dropped above this rather than below it: a
		// nested _test.go contributes nothing to any expansion by either rule,
		// and listing it here would be noise in a report about declarations
		// that went missing.
		if path.Dir(p) != themePkg {
			nested = append(nested, p)
			continue
		}
		direct = append(direct, p)
	}
	return direct, nested, nil
}

// shadowKind is where one collision's declarations sit, as a phrase that
// completes "<Name> is …".
//
// # Why the placement is part of the finding
//
// themeleaves.Of records any bare name it parsed more than once, wherever the
// declarations were, and the two readers of that record used to describe it
// with one sentence: "declared both in core/ and below it". That is the union
// probe's case — main_test.go hands Of the top level of core/ AND a
// subdirectory on purpose — and it is only one of the shapes the record can
// hold. Two subdirectories declaring one name, or two files in one directory,
// would have printed the same sentence about the wrong thing, and a reader
// chasing a moved population would have gone looking in the wrong place.
//
//	core/theme.go + core/zsub/x.go   two packages. The union probe's case, and
//	                                 the one the rule exists to prevent
//	core/a/x.go   + core/b/x.go      two packages, neither of them core/.
//	                                 Nothing in this repository reads both
//	core/x.go     + core/y.go        ONE package declaring a name twice. Does
//	                                 not compile; reachable at a revision
//	                                 caught mid-refactor, which this walk parses
//	core/x.go     + core/x.go        one FILE declaring it twice. go/parser
//	                                 reads it; the compiler does not accept it
//
// The last two are why Shadow.Files is recorded as parsed rather than
// deduplicated: a path listed twice is a fact about the file, and collapsing
// it would make the two bottom rows print identically.
//
// Every phrase ends by naming the paths with the winner last, because a reader
// with a moved population wants the declaration the expansion was taken over
// and Files is ordered for exactly that.
func shadowKind(sh themeleaves.Shadow) string {
	where := fmt.Sprintf(" (declared in %s; the last of those is the one that "+
		"answered)", strings.Join(sh.Files, ", "))

	// One file listed more than once is the narrowest case and has to be
	// tested before any question about directories: every path is the same, so
	// every directory is too, and the directory-based arms below would call it
	// "twice in one directory" and lose the sharper fact.
	same := true
	for _, p := range sh.Files {
		if p != sh.Files[0] {
			same = false
			break
		}
	}
	if same {
		return fmt.Sprintf("declared %d time(s) in ONE file%s", len(sh.Files),
			where)
	}

	// Otherwise the shape is a question about the directories, and the two
	// that matter are "is the top level one of them" and "how many are there".
	dirs := map[string]bool{}
	atTop := false
	for _, p := range sh.Files {
		d := path.Dir(p)
		dirs[d] = true
		if d == themePkg {
			atTop = true
		}
	}
	switch {
	case atTop && len(dirs) > 1:
		return fmt.Sprintf("declared both directly in %s/ and below it%s",
			themePkg, where)
	case atTop:
		return fmt.Sprintf("declared %d time(s) directly in %s/, which is one "+
			"package declaring one name twice%s", len(sh.Files), themePkg, where)
	case len(dirs) == 1:
		var only string
		for d := range dirs {
			only = d
		}
		return fmt.Sprintf("declared %d time(s) in %s, which is one package "+
			"below %s/ declaring one name twice%s", len(sh.Files), only,
			themePkg, where)
	default:
		return fmt.Sprintf("declared in %d different directories below %s/, "+
			"none of them %s/ itself%s", len(dirs), themePkg, themePkg, where)
	}
}

// treePaths is every path git tracks under themePkg at one revision, as git
// actually spells it on disk.
//
// # Why -z, which is the whole reason this is a function
//
// `ls-tree --name-only` C-QUOTES a path it cannot write literally: a name
// holding a quote, a backslash, a control byte or any byte outside ASCII comes
// back wrapped in double quotes with the offending bytes escaped, and the
// quotes are part of the line rather than around it.
//
//	on disk           core/a<LF>b.go    core/q"x.go      core/ä.go
//	--name-only       "core/a\nb.go"    "core/q\"x.go"   "core/\303\244.go"
//	--name-only -z    core/a<LF>b.go    core/q"x.go      core/ä.go
//
// Each of those quoted spellings then travels as if it were a path, and both
// ways it can go are wrong:
//
//	the suffix test   the name now ends `.go"` and not `.go`, so the file is
//	                  DROPPED and the revision's population is silently short
//	                  by whatever it declared
//	the fetch         a name that survives the suffix test is handed to
//	                  `cat-file` as `<rev>:"core/…"`, which git does not
//	                  resolve — the walk stops with an error naming a path
//	                  nothing on disk is called
//
// The newline case is the one that shows how far the quoting reaches. blob
// sends a path holding a real newline to a `cat-file -p` of its own, because
// the batch protocol is newline-terminated — and `"core/a\nb.go"` holds a
// backslash and an `n` rather than a newline, so that guard never sees it and
// the request goes down the batch as an object name git cannot resolve. The
// guard is not wrong; it was being handed a spelling with nothing left in it to
// guard against. With -z the path arrives as git holds it and the two halves
// agree about what a path is.
//
// -z is git's answer: NUL-terminated records and no quoting at all, on every
// version of git this repository has ever been read with. main_test.go already
// argues exactly this for `git status --porcelain -z` — and that argument
// stopped at the one command, while the other two readings of a path in this
// package kept the default. The quoting RULES are not even the same: `status`
// quotes a path holding a space (the space is its field separator) and
// `ls-tree --name-only` does not, so a file called `core/a b.go` comes back
// quoted from one and bare from the other, and a comparison between them is
// between two spellings of one path.
//
// A NUL is the one byte a path cannot contain, so a record is a path and the
// split cannot be wrong. The trailing NUL after the last record is trimmed
// rather than yielding an empty final element, and an empty listing — a
// revision with nothing under themePkg — yields no paths rather than one empty
// one.
func treePaths(sha string) ([]string, error) {
	out, err := git("ls-tree", "-r", "-z", "--name-only", sha, "--", themePkg)
	if err != nil {
		return nil, err
	}
	out = strings.TrimSuffix(out, "\x00")
	if out == "" {
		return nil, nil
	}
	return strings.Split(out, "\x00"), nil
}

// leafSet is an expansion as the diff below reads it.
func leafSet(exp themeleaves.Expansion) map[string]bool {
	set := make(map[string]bool, len(exp.Names))
	for _, name := range exp.Names {
		set[name] = true
	}
	return set
}

// wrap is a list of names as indented lines, so a 28-name commit does not
// print as one unreadable row.
func wrap(names []string, width int, indent string) string {
	var b strings.Builder
	line := 0
	for _, name := range names {
		if line > 0 && line+len(name)+2 > width {
			b.WriteString("\n" + indent)
			line = 0
		}
		if line > 0 {
			b.WriteString(", ")
			line += 2
		}
		b.WriteString(name)
		line += len(name)
	}
	return b.String()
}

// blob is one file's text at one revision.
//
// # Why this is not `git cat-file -p` per file
//
// It was, and each revision is read with one `ls-tree` plus one `cat-file` per
// non-test .go file in it — a git process per object, for a table of sixteen
// rows. Almost none of that time is git doing anything: it is fork, exec, the
// repository being opened, and the process being torn down, once per file.
//
// # The two numbers this paragraph used to assert, and what they are now
//
// "Around forty-four hundred" stood here for several sessions and it was a
// BOUND arithmetic away from the source — eighty-eight commits times up to
// forty-nine files — written in the position a count goes. It is now the
// repository's own answer on every run: themeSourcesAt names the objects a
// revision contributes and TestTheWholeWalkGoesRoundOneBatchProcess sums it
// across the history, holds the batch reader's fetch count to that sum, and
// reports it. Getting on for three thousand over this repository's history,
// and a number
// that moves with the checkout rather than a constant tuned against one run.
//
// "Thirty-one seconds" was worse: taken once, in the session that deleted the
// code which cost it. It is now a RATIO of two readings taken in the same run
// over the same objects —
// TestOneProcessPerObjectIsSlowerThanOneProcessForAllOfThem re-creates the
// per-object shape rather than describing it, and is off unless
// GRMOB_PER_OBJECT_FETCH=required because it is thirty seconds and three
// thousand processes:
//
//	one `cat-file -p` each   one process per object, thirty seconds
//	one `cat-file --batch`   one process, about four hundred milliseconds
//	                         ────────────────────────────────────────────
//	                         seventy-odd × on the fetches alone
//
// The figures are deliberately orders of magnitude here and not readings: the
// readings live in themehistoryTimingsTakenOn.perObjectRun, which is where
// they are re-taken, and a copy of them in this comment would be a second
// place to keep in step. They were exactly that until tonight — this table
// said 30.14–30.42s against 399–401ms while the record said something else,
// both attributed to the same afternoon.
//
// The batched figure there is the fetches on their own, which is why it is
// well under wholeRun — that one also pays an `ls-tree` per commit and every
// revision's parse. The thirty-one seconds carried in this comment for
// several sessions turns out to have been right, and it is no longer what the
// argument rests on: what rests here now is a comparison anybody can re-take
// with one environment variable, on their own machine, in either direction.
//
// The arm asserts the DIRECTION and not the multiple, for the reason every
// number in themehistoryTimingsTakenOn is reported rather than asserted: 75×
// is a reading of this machine's fork cost against its pipe cost.
//
//	before   ls-tree ── cat-file ── cat-file ── cat-file ── ...   per revision
//	after    ls-tree ── ┐
//	                    └── one `cat-file --batch`, for the whole run
//
// `--batch` is git's answer to exactly this: one process reads object names on
// its stdin and writes each object to its stdout, for as long as the pipe is
// open. The whole run then costs 90 processes — one `log`, one `ls-tree` per
// revision, one `cat-file --batch` — rather than one per object, and the
// reading is byte-for-byte the one it was — `--batch` and `-p` both write a blob's
// contents raw, with no filters and no line-ending conversion, which is what
// makes this a change in how the text is fetched and not in what it says.
//
// # The protocol, since a misread here is a silently wrong table
//
// One request per line, "<rev>:<path>". One response, in three parts:
//
//	<oid> SP <type> SP <size> LF   the header
//	<size> bytes                   the object, raw
//	LF                             a terminator that is NOT in the size
//
// So the body is read by COUNT and not by any delimiter, and the trailing LF
// is consumed separately. A response that is short by that byte leaves the
// next read starting one byte into the following header, and every file after
// it in the run would come back wrong.
//
// # And one shape that goes back to a process of its own
//
// The request is newline-terminated, so a path containing a newline cannot be
// asked for this way — and asking anyway is not a failed request but a
// DESYNCHRONISED stream. git reads the newline as the end of one name and the
// rest as the start of another, so one request gets two responses:
//
//	written:  HEAD:core/a<LF>b.go<LF>
//	read:     HEAD:core/a missing<LF>
//	          b.go missing<LF>       ← nobody asked, nobody reads it
//
// The second line then answers the NEXT request, and every response after it
// belongs to the request before it. So such a path gets a `cat-file -p` of its
// own, which takes its object name from argv where a newline is an ordinary
// byte. Nothing in this repository has one; git will track one; and since
// treePaths asks for -z, a path that has one now arrives here intact instead of
// as `"core/a\nb.go"`, which is what this guard was previously being handed and
// could not see. See TestAPathWithANewlineInItGoesRoundTheBatch.
//
// # And why the reader can be replaced, when nothing has ever replaced it
//
// Two things make the one long-lived process a state that can go bad and stay
// bad, and both of them are cheap to fix once fixing them is possible at all:
//
//	a desynchronised   every error but `missing` leaves the stream at an offset
//	stream             nobody knows. Carrying on means reading a body as a
//	                   header and a header as a body, for the rest of the run.
//	a moved cwd        the process resolves `<rev>:<path>` from the top of the
//	                   repository, so only repository DISCOVERY depends on its
//	                   working directory — but discovery is what decides WHICH
//	                   repository, and this process's directory was fixed at the
//	                   moment it started. A test that t.Chdir's into a throwaway
//	                   clone and fetches would be answered out of the one it was
//	                   born in, and `missing` is the kindest way that ends.
//
// Neither is recovered from by cleverness: the reader is RETIRED and the next
// call starts a fresh one. A process is what the batch was bought to save
// thousands of, and paying one back to leave a known-bad state is the trade
// this whole file already makes for correctness over count.
func blob(rev, path string) (string, error) {
	if strings.ContainsAny(path, "\n") {
		return git("cat-file", "-p", rev+":"+path)
	}
	// Read before the lock rather than inside newBatchReader, so the check
	// below and the process's own Dir are the same string. One getcwd per
	// fetch is a syscall against a pipe round trip; at the few thousand fetches a
	// whole run over this repository takes — counted by batchReader.reads and
	// reported by TestTheWholeWalkGoesRoundOneBatchProcess — it is not
	// measurable.
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("git cat-file --batch: the working directory "+
			"cannot be read, so there is no way to tell which repository a "+
			"running batch would answer out of: %w", err)
	}
	blobsMu.Lock()
	defer blobsMu.Unlock()
	if blobs != nil && (blobs.dead != nil || blobs.dir != wd) {
		blobs.retire()
		blobs = nil
	}
	if blobs == nil {
		b, err := newBatchReader(wd)
		if err != nil {
			return "", err
		}
		blobs = b
	}
	return blobs.read(rev, path)
}

// The one `git cat-file --batch` this process talks to, started on first use.
//
// Lazy because a program that never reads a blob — `themehistory` cannot be
// one, but a test binary that only runs the unit tests can be — should not
// start a git process to find that out. Retired and replaced rather than
// closed at the end: nothing here knows when the last fetch was, so what ends
// the surviving process is this one exiting and the pipe closing with it,
// which is the same lifetime the per-file processes had in aggregate.
//
// The mutex is not for this program, which is single-threaded. It is for the
// tests, which share this package and could be given a t.Parallel() at any
// point; a batch reader driven from two goroutines would interleave two
// requests and hand each caller the other's file.
var (
	blobs   *batchReader
	blobsMu sync.Mutex
)

// errMissing is git's `<name> missing` — a complete one-line response, and the
// one error that leaves the stream where the next request can use it.
//
// A sentinel rather than a string, because the difference between this and
// every other error is not cosmetic: this one costs a caller its file and the
// next caller nothing, and every other one costs the reader.
var errMissing = errors.New("no such object at that revision")

// errDesync is any response the reader could not consume WHOLE.
//
// A header that is not three fields, a size that will not parse, a body that
// ends early, a pipe that closed, a request that could not be written — after
// any of them the stream is at an offset nobody knows, and the next response
// read out of it would be some part of this one. Every error carrying this is
// a reason to retire the process rather than a reason to stop the run: see
// blob, which does exactly that.
var errDesync = errors.New("the batch stream is at an unknown offset")

// batchReader is a running `git cat-file --batch` and the two ends of its
// pipes.
type batchReader struct {
	cmd *exec.Cmd
	in  io.WriteCloser
	// The raw stdout, kept alongside the buffered one so retire can drain it:
	// Wait may not be called while a read is outstanding, and a git holding
	// bytes nobody has taken would block writing them.
	pipe io.ReadCloser
	// Buffered because the body is read by count immediately after a header
	// read that stops at a newline: an unbuffered read would need the header
	// consumed one byte at a time to avoid swallowing the object behind it.
	out *bufio.Reader
	// The working directory this process was started in, which is the
	// directory git discovered its repository from. Compared on every fetch —
	// see blob — because a caller that has moved is asking a different
	// question and this process cannot hear it.
	dir string
	// Why this reader is finished, or nil. Set by read on any error but
	// errMissing; once set, nothing is ever read out of this reader again.
	dead error
	// How many requests have gone round this process.
	//
	// # Why a counter is in a struct that is otherwise all mechanism
	//
	// The whole argument for `--batch` is a count: every object a revision
	// contributes costs a process, and the point of the batch is that all of
	// them cost ONE. Every part of that sentence was a number in a comment.
	//
	// This is the part a program can answer. With it,
	// TestTheWholeWalkGoesRoundOneBatchProcess says that a real run fetched
	// exactly the objects themeSourcesAt names across the history and started
	// one process to do it — the claim, stated about the run that just
	// happened rather than about a run somebody remembers. The count is held
	// to an EQUALITY rather than a floor, so it fails both ways: a revision
	// the walk skipped, and an object fetched twice.
	//
	// Not guarded: this program is single-threaded and every read goes
	// through blobsMu; see the note on that mutex.
	reads int
}

// How many `git cat-file --batch` processes this program has started.
//
// The other half of the count above, and it has to live outside the reader
// because what it is counting is readers: a batch that desynchronises is
// retired and REPLACED (see blob), so a per-reader field would be reset by the
// very event worth noticing. One for a whole run is the claim; two is a run
// that hit a desync and carried on, which is correct behaviour and a different
// performance story.
//
// Atomic rather than plain because this one is written by newBatchReader,
// which the tests call directly and outside blobsMu.
//
// # Nothing resets it, and that is the point
//
// It counts for the lifetime of the process, so a reader of it takes a DELTA
// around whatever it is asking about — batchesStartedSince in
// internal/themehistory/timings_test.go is that discipline written down once.
// A reset would be the wrong tool twice over: it would lose the retire-and-
// replace this counter exists to notice, and it would silently subtract from
// any delta another test had open, in a package where nothing is parallel
// today and the mutex above exists because that can change.
var batchesStarted atomic.Int64

func newBatchReader(dir string) (*batchReader, error) {
	cmd := exec.Command("git", "cat-file", "--batch")
	// Stated rather than inherited. exec would use this process's working
	// directory anyway, and that is the point: the directory this process
	// resolves a repository from is now a field somebody can compare, rather
	// than whatever the caller's cwd happened to be at the moment of the first
	// fetch in the run.
	cmd.Dir = dir
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("git cat-file --batch: stdin: %w", err)
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("git cat-file --batch: stdout: %w", err)
	}
	// stderr goes to this process's, so a git that has something to say says
	// it where the rest of this command's diagnostics go rather than into a
	// buffer nobody reads.
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("git cat-file --batch: %w", err)
	}
	// Counted after Start, so what this counts is processes that exist rather
	// than attempts to make one.
	batchesStarted.Add(1)
	return &batchReader{cmd: cmd, in: in, pipe: out, out: bufio.NewReader(out),
		dir: dir}, nil
}

// How long retire waits for a git to leave on its own before killing it.
//
// # Why this is not a timeout in the usual sense
//
// A timeout tuned to a workload is a number somebody has to re-tune. This is
// not one: it is a bound on "a healthy git has already gone", and the healthy
// case is bounded by a pipe drain from a local process plus a process exit —
// microseconds to low milliseconds. No run that is working can reach it, which
// is the property that matters, and the cost of being generous is that a
// wedged git is waited on for five seconds once rather than forever.
//
// # And what that argument was worth once it was measured
//
// The paragraph above was reasoning, and it stood on its own for a session:
// "four orders of magnitude under this" was a claim about a pipe drain that
// nobody had timed. It is now taken on every run.
// TestRetiringAHealthyGitLeavesBeforeTheDeadline retires seven healthy batch
// readers and records what they cost — themehistoryTimingsTakenOn.batchRetire
// is the reading, and the slowest of them is about 1/17000 of the grace,
// and the a priori figure was right to the order.
//
// That is the smaller half of what the measurement buys. The larger half is
// that the test asserts the OUTCOME rather than the clock: a healthy git exits
// 0 because it noticed the EOF, and a git the deadline killed does not exit 0
// at all. So the sentence "no run that is working can reach it" now fails a
// test on the day it stops being true, and it cannot fail because a machine
// was busy — a slow retire is still a 0 exit right up to the deadline.
//
// A variable rather than a const so the arm can lower it: a test for the
// deadline that had to wait out the real one would be a five-second test, and
// what it is checking is that the deadline EXISTS.
var batchRetireGrace = 5 * time.Second

// retire ends this reader's process and waits for it.
//
// `git cat-file --batch` reads requests until its stdin reaches EOF and then
// exits, so closing the write end is the whole shutdown. What has to happen
// before Wait is the DRAIN: this is called on a reader whose stream may hold
// bytes nobody took — the second half of a desynchronising response, or a whole
// response for a request whose caller gave up — and a git blocked writing into
// a full pipe would never see the EOF.
//
// # Why there is a deadline behind it
//
// Both of those steps terminate for a `cat-file --batch` and neither is
// guaranteed to. The drain ends when the pipe reaches EOF, which is when the
// process exits; the process exits when it notices the EOF on its stdin. A git
// wedged for any other reason — a filesystem that will not answer, a pack being
// rewritten underneath it — holds the drain open, and the drain holds blobsMu,
// and that is the whole run. So the shutdown gets a deadline and a Kill behind
// it, and the Kill is what makes the wait terminate: it closes the process's
// end of the pipe, the drain reaches EOF, and Wait returns.
//
// Errors are dropped on purpose. Every caller is already on its way to
// starting a fresh reader, and there is no answer this could give that would
// change that.
func (b *batchReader) retire() {
	b.in.Close()
	done := make(chan struct{})
	go func() {
		defer close(done)
		io.Copy(io.Discard, b.pipe)
		b.cmd.Wait()
	}()
	select {
	case <-done:
	case <-time.After(batchRetireGrace):
		b.cmd.Process.Kill()
		// Waited for even after the Kill, so the goroutine and the process are
		// both finished when this returns. Killing a process does not reap it,
		// and a retire that left one behind on every desynchronising response
		// would trade a wedged run for a leak.
		<-done
	}
}

// read is one request and its response.
//
// # Which errors leave the stream usable, and which do not
//
// "<name> missing" is a COMPLETE response — one line, no body — so a request
// for an object git cannot resolve costs an error and nothing else: the stream
// is still sitting at the start of the next header and the caller after this
// one gets its own file. That is the case worth being exact about, because it
// is the reachable one (a path fetched at the wrong revision) and because a
// reader that resynchronised wrongly would hand somebody else's bytes to a
// walk that would parse them without complaint.
//
// Every other error here — a header that is not three fields, a body that ends
// early, a pipe that closed — leaves the stream at an unknown offset, and the
// error carries errDesync to say so. This reader is finished at that point:
// `dead` is set, and blob retires the process and starts another rather than
// reading one more byte out of a stream whose position is a guess.
//
// Recovering costs a process, which is the thing the batch exists to save
// thousands of — and it is still the right trade, because the alternative is
// not a slower run but a wrong one. Nothing has ever produced such a response;
// the point is that if one arrives, what follows it is a fresh git rather than
// a table assembled out of misaligned bytes.
func (b *batchReader) read(rev, path string) (string, error) {
	name := rev + ":" + path
	// Counted before the write, so a request that failed on the way out is
	// still a request this process was asked for — the count is of what the
	// run demanded, which is what the batch's cost argument is about.
	b.reads++
	if _, err := io.WriteString(b.in, name+"\n"); err != nil {
		// The request may have been written in part. Whether git saw a whole
		// name, half of one, or nothing at all is not knowable from here, so
		// what the stream holds next is not knowable either.
		b.dead = fmt.Errorf("git cat-file --batch: asking for %s: %w: %w",
			name, err, errDesync)
		return "", b.dead
	}
	src, err := readResponse(b.out, name)
	if err != nil && !errors.Is(err, errMissing) {
		b.dead = err
	}
	return src, err
}

// readResponse is one `git cat-file --batch` response, taken off a stream.
//
// A function of a reader rather than a method, because everything that can go
// wrong here is a property of the BYTES and not of the process behind them:
// held this way the four failure shapes are a table over hand-built streams
// (see TestEveryBatchResponseShapeIsToldApart) rather than four states of a
// git nobody can make misbehave on purpose.
//
// name is carried only for the messages; the protocol does not echo it back
// except in the `missing` line, and a reader that matched on that echo would
// be trusting the stream to tell it where it is.
func readResponse(r *bufio.Reader, name string) (string, error) {
	header, err := r.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("git cat-file --batch: reading the header for "+
			"%s: %w: %w", name, err, errDesync)
	}
	line := strings.TrimSuffix(header, "\n")
	// "<oid> <type> <size>" — or "<name> missing", which is git's answer for
	// an object it cannot resolve and is not an error on the pipe. It is an
	// error HERE: every path this is asked for was named by `ls-tree` at the
	// same revision in this same run.
	fields := strings.Fields(line)
	if len(fields) == 2 && fields[1] == "missing" {
		return "", fmt.Errorf("git cat-file --batch: %s: %w", name, errMissing)
	}
	// Anything else that is not a header, and `ambiguous` is the one worth
	// naming: git answers `<name> SP ambiguous LF` for an object name that
	// matches more than one object, and it is a COMPLETE one-line response
	// exactly as `missing` is. Recognising it would be correct and would cost
	// nothing; it is deliberately not recognised, and the reason is what the
	// two mistakes cost.
	//
	//	read as desynchronising   one git process, once, and a run that carries
	//	                          on with correct bytes
	//	read as complete when it   a body that was never there is skipped or a
	//	is not                    header is read as one, and every response
	//	                          after it belongs to the request before it
	//
	// So the set of lines this reader will trust is kept to the one it has
	// actually seen git send. `ambiguous` needs a short or otherwise ambiguous
	// object name and every name asked for here is `<full sha>:<path>` built
	// from `ls-tree`'s own output, so nothing in this walk can produce one —
	// which is why the conservative reading is free rather than merely cheap.
	// See TestEveryBatchResponseShapeIsToldApart, where the choice is a row
	// rather than a consequence of `len(fields) != 3`.
	if len(fields) != 3 {
		return "", fmt.Errorf("git cat-file --batch: %s: %q: %w", name, line,
			errDesync)
	}
	size, err := strconv.Atoi(fields[2])
	if err != nil {
		return "", fmt.Errorf("git cat-file --batch: %s: unreadable size in "+
			"%q: %w: %w", name, line, err, errDesync)
	}
	body := make([]byte, size)
	if _, err := io.ReadFull(r, body); err != nil {
		return "", fmt.Errorf("git cat-file --batch: %s: reading %d byte(s): "+
			"%w: %w", name, size, err, errDesync)
	}
	// The terminator, which is not counted in size. Left unread it would be
	// the first byte of the next response's header.
	if _, err := r.ReadByte(); err != nil {
		return "", fmt.Errorf("git cat-file --batch: %s: reading the byte after "+
			"the object: %w: %w", name, err, errDesync)
	}
	return string(body), nil
}

func git(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err,
			strings.TrimSpace(errb.String()))
	}
	return out.String(), nil
}
