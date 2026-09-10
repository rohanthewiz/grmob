package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// Every git command in this repository that LISTS PATHS asks for them
// NUL-separated.
//
// # The rule, and the bug that produced it
//
// git writes a path back to you in one of two spellings, and which one you get
// is a property of the SUBCOMMAND rather than of the path:
//
//	on disk              core/a<LF>b.go     core/q"x.go      core/ä.go      core/a b.go
//	ls-tree --name-only  "core/a\nb.go"     "core/q\"x.go"   "core/\303\244.go"   core/a b.go
//	status --porcelain   "core/a\nb.go"     "core/q\"x.go"   "core/\303\244.go"   "core/a b.go"
//	either, with -z       core/a<LF>b.go     core/q"x.go      core/ä.go      core/a b.go
//
// C-quoting for a name git cannot write literally — and note the two commands
// disagree about a SPACE, so "the rule" is not one rule. -z is the only
// spelling both of them, and every other listing subcommand, agree on: NUL is
// the one byte a path cannot contain, so a record is a path and no unescaping
// is possible or needed.
//
// internal/themehistory had this exactly half right. Its `git status
// --porcelain -z` carried a paragraph arguing the case; its `ls-tree
// --name-only` next to it did not, and a quoted name ends `.go"` rather than
// `.go`, so it failed a suffix test and the file was DROPPED with nothing on
// stderr. With core.SpacingScale moved into a newline-named file the walker
// reported 76 leaves and a commit that removed four, in a table whose headline
// finding is that nothing ever has.
//
// # Why this is an arm and not the grep that found it
//
// The grep found no second gap: the only other listings in this repository are
// checknumbering_test.go's two `ls-files -z`, and they were already right. So
// this check passes today and passed before it was written, which is the point
// — it is not here to fix anything. A repository is one `git diff --name-only`
// away from the bug at any time, and the failure mode is a file silently
// missing from a reading rather than an error, which is the shape nothing
// notices. The grep was a fact about one afternoon; this is a fact about every
// run.
//
// # What counts as a listing
//
// Two triggers, either of which means paths are coming back on stdout:
//
//	the subcommand    ls-files, ls-tree and status write paths and nothing
//	                  else worth having
//	the flag          --name-only, --name-status or --porcelain on anything
//	                  (diff, log, show, status) turns it into a path listing
//
// Everything else is left alone deliberately. `rev-parse` answers with an oid,
// `cat-file` with an object, `check-ignore -q` with an exit status, and `log
// --format=%H` with whatever the format says — none of them is a path, and
// requiring -z of them would be a rule about the wrong thing.
func TestEveryGitListingAsksForNulSeparatedPaths(t *testing.T) {
	root := filepath.Join("..", "..")
	// `considered` and not the first return value, which is only the files
	// that carry a CITATION — the question here is about every Go file in the
	// repository, and a git call in a file with no "check N" in it is exactly
	// the one that would be missed. (It was: this check first reported two
	// invocations, both of them in the file that does the enumerating.)
	_, considered, from, err := citingFiles(root)
	if err != nil {
		t.Fatalf("enumerating the repository (%s): %v", from, err)
	}
	paths := make([]string, 0, len(considered))
	for p := range considered {
		paths = append(paths, p)
	}
	// Sorted because the map is not, and a check that reports its findings in
	// a different order on every run is a check whose output cannot be diffed.
	sort.Strings(paths)

	// Parsed rather than grepped, because the question is about one call's
	// argument list. A grep for "ls-tree" finds this file's own prose, the
	// comment three lines above the call, and the call — and cannot tell which
	// of them has a -z in it.
	fset := token.NewFileSet()
	// Two passes, and the first one exists because of what the second used to
	// assume. See gitArgs: a call to a bare identifier `git` was read as a git
	// invocation wherever it appeared, on the reasoning that a package running
	// more than two git commands writes such a helper and that nothing else
	// would be called that. The first half is true here; the second was a
	// guess, and a wrong guess would have this check saying something
	// confident about a call that has nothing to do with git.
	//
	// So the helpers are FOUND first, held to actually being git wrappers, and
	// a bare `git(…)` is read as an invocation only in a package that declares
	// one. What was a heuristic standing in for cross-package identifier
	// resolution is now a premise this file checks.
	type parsed struct {
		rel  string
		dir  string
		file *ast.File
	}
	var files []parsed
	// dir -> the position of the `func git` declared in it.
	helpers := map[string]token.Position{}
	for _, rel := range paths {
		if !strings.HasSuffix(rel, ".go") {
			continue
		}
		// This file's own tables name the subcommands and would read as
		// unflagged calls if anything here ever became one. It has no git call
		// in it and is skipped by name rather than by luck.
		if rel == "wasm/verify/gitquoting_test.go" {
			continue
		}
		src := filepath.Join(root, filepath.FromSlash(rel))
		file, err := parser.ParseFile(fset, src, nil, parser.SkipObjectResolution)
		if err != nil {
			// A file go/parser cannot read is not this check's business: the
			// build says so first, and every other reading in this repository
			// would be failing too.
			continue
		}
		dir := path.Dir(rel)
		files = append(files, parsed{rel: rel, dir: dir, file: file})
		for _, d := range file.Decls {
			fn, ok := d.(*ast.FuncDecl)
			// Recv nil because a METHOD called git is not what a bare `git(…)`
			// call resolves to — that would be `x.git(…)`, which reaches
			// gitArgs as a SelectorExpr and is rejected there.
			if !ok || fn.Recv != nil || fn.Name.Name != "git" {
				continue
			}
			at := fset.Position(fn.Name.Pos())
			if prev, dup := helpers[dir]; dup {
				// Two in one package do not compile, so this is a package
				// split across directories by the enumeration or a build tag —
				// either way the second one is what the calls below would
				// resolve to and it is worth naming.
				t.Errorf("%s declares a second `git` helper at line %d; the "+
					"first is %s:%d. This check reads a bare `git(…)` call as "+
					"a git invocation, and with two declarations in one "+
					"package it cannot say which one a call means.",
					rel, at.Line, prev.Filename, prev.Line)
			}
			helpers[dir] = at
			// The premise, checked rather than assumed: a function called
			// `git` in this repository runs git. A helper that did something
			// else would make every call to it a finding about the wrong
			// thing, stated with a line number and a subcommand.
			if !runsGit(fn) {
				t.Errorf("%s:%d declares `func git` and its body contains no "+
					"exec.Command(\"git\", …).\n\n"+
					"Calls to a bare `git(…)` in %s are read by this check as "+
					"git invocations and held to the -z rule. If this helper "+
					"is something else, the findings this check produces about "+
					"that package are about the wrong function — rename one of "+
					"them, or teach gitArgs which is which.", rel, at.Line, dir)
			}
		}
	}

	seen := 0
	for _, pf := range files {
		_, local := helpers[pf.dir]
		rel, file := pf.rel, pf.file
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			args, ok := gitArgs(call, local)
			if !ok {
				return true
			}
			seen++
			if !gitListsPaths(args) || hasArg(args, "-z") {
				return true
			}
			t.Errorf("%s:%d asks git for a listing of paths and does not ask "+
				"for -z:\n\n  git %s\n\n"+
				"git C-quotes any path it cannot write literally — a quote, a "+
				"backslash, a control byte, anything outside ASCII, and for "+
				"some subcommands a space — so `core/ä.go` arrives as "+
				"`\"core/\\303\\244.go\"`, quotes included. A reader that "+
				"tests the suffix drops it (the name now ends `.go\"`), and a "+
				"reader that passes it on hands git an object name it will not "+
				"resolve. Neither is an error anybody sees.\n\n"+
				"-z makes every record a NUL-terminated path with no escaping. "+
				"Split on \"\\x00\" and trim the trailing separator; do not "+
				"TrimSpace, which is a rule about a format this no longer is.",
				rel, fset.Position(call.Lparen).Line, strings.Join(args, " "))
			return true
		})
	}

	// The enumeration reaching the tree at all. Every other arm in this
	// repository that walks the repository says this, and for the same reason:
	// a walk that found nothing passes silently and reads as a clean result.
	if seen == 0 {
		t.Fatalf("no `git` invocation was found anywhere in %d Go file(s) "+
			"enumerated by %s, and this repository has several — "+
			"internal/themehistory's walk is built on them. The parse is not "+
			"reaching the tree, so this check is over nothing.", len(paths), from)
	}
	where := make([]string, 0, len(helpers))
	for dir := range helpers {
		where = append(where, dir)
	}
	sort.Strings(where)
	t.Logf("%d git invocation(s) across the repository's Go sources; every one "+
		"that lists paths asks for -z. %d package(s) declare a `git` helper, "+
		"held above to running one: %s. Enumerated by %s.",
		seen, len(helpers), strings.Join(where, ", "), from)
}

// gitArgs is a call's git arguments as string literals, and whether it is a
// git call at all.
//
// # The two shapes, and what `local` settles
//
// A git call in this repository is either `exec.Command("git", …)` or a call
// to a package-local helper spelled `git(…)` — internal/themehistory has one,
// and a helper is the obvious thing to write once a package runs more than two
// commands.
//
// The identifier form was once matched wherever it appeared, on the argument
// that the loose direction was the safe one: the worst case is a message
// naming a call somebody can look at. That is true and it is not the whole
// cost. A check that says something confident and wrong about a call is a
// check people stop reading, and this one's whole value is that it is believed
// when it fires — the thing it reports (a file silently missing from a
// listing) is invisible by construction, so a reader has nothing else to weigh
// its findings against.
//
// `local` is the caller's answer to "does this file's package declare a `git`
// helper", found by the pass above and held there to actually running git. A
// bare identifier in a package that declares no such function is some other
// `git` — a variable, a dot-imported name, a helper in a package this walk
// enumerated under a different directory — and is not read as an invocation.
// The heuristic is now a premise with a check under it.
//
// Only literal arguments are collected. A subcommand assembled at runtime is
// beyond what a parse can say, and there are none: every git call here names
// its subcommand as a constant, which is the property that makes this check
// possible rather than an assumption it makes.
func gitArgs(call *ast.CallExpr, local bool) ([]string, bool) {
	args := call.Args
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		if fn.Name != "git" || !local {
			return nil, false
		}
	case *ast.SelectorExpr:
		// exec.Command("git", …) — and the first argument is the program, so
		// it is consumed here rather than counted as an argument.
		pkg, ok := fn.X.(*ast.Ident)
		if !ok || pkg.Name != "exec" || fn.Sel.Name != "Command" {
			return nil, false
		}
		if len(args) == 0 || literal(args[0]) != "git" {
			return nil, false
		}
		args = args[1:]
	default:
		return nil, false
	}
	out := make([]string, 0, len(args))
	for _, a := range args {
		if s := literal(a); s != "" {
			out = append(out, s)
		}
	}
	return out, true
}

// literal is a string literal's value, or "" for anything else.
//
// Unquoted with strconv.Unquote in spirit but by hand: every git argument in
// this repository is an ordinary interpreted string with no escapes worth
// decoding, and the values compared against are flags and subcommands. What
// matters is that a non-literal returns "" rather than something that might
// match a flag.
func literal(e ast.Expr) string {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return ""
	}
	return strings.Trim(lit.Value, "`\"")
}

// runsGit is whether a `func git` declaration actually runs git.
//
// The test is `exec.Command("git", …)` somewhere in the body, which is what
// every wrapper of this shape is built on and what the one in
// internal/themehistory is. A helper that reached git some other way — a
// go-git binding, a shelled-out `sh -c` — would fail this and would be right
// to: this check reads such a call's LITERAL ARGUMENTS as a git command line,
// and that reading is only correct for a wrapper that passes them to git.
func runsGit(fn *ast.FuncDecl) bool {
	if fn.Body == nil {
		return false
	}
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) == 0 {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Command" {
			return true
		}
		pkg, ok := sel.X.(*ast.Ident)
		if !ok || pkg.Name != "exec" || literal(call.Args[0]) != "git" {
			return true
		}
		found = true
		return false
	})
	return found
}

// The subcommands whose whole output is paths.
//
// `status` is here rather than under the flag rule below because its default
// output is paths too — the flag only changes how much else is on the line.
var gitPathSubcommands = map[string]bool{
	"ls-files": true,
	"ls-tree":  true,
	"status":   true,
}

// The flags that turn any command into a path listing.
//
// --name-only and --name-status are diff's and log's and show's; --porcelain
// is status's and is listed for the case where a future caller reaches it on
// something else. A flag written as `--name-only=…` is not a form any of these
// take, so an exact match is the whole test.
var gitPathFlags = map[string]bool{
	"--name-only":   true,
	"--name-status": true,
	"--porcelain":   true,
}

// gitListsPaths is whether this argument list makes git write paths to stdout.
func gitListsPaths(args []string) bool {
	for _, a := range args {
		if gitPathSubcommands[a] || gitPathFlags[a] {
			return true
		}
	}
	return false
}

func hasArg(args []string, want string) bool {
	for _, a := range args {
		if a == want {
			return true
		}
	}
	return false
}
