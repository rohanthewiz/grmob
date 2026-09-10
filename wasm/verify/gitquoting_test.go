package main

import (
	"go/ast"
	"go/parser"
	"go/token"
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
	seen := 0
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
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			args, ok := gitArgs(call)
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
	t.Logf("%d git invocation(s) across the repository's Go sources; every one "+
		"that lists paths asks for -z. Enumerated by %s.", seen, from)
}

// gitArgs is a call's git arguments as string literals, and whether it is a
// git call at all.
//
// # The two shapes, and why a heuristic is honest here
//
// A git call in this repository is either `exec.Command("git", …)` or a call
// to a package-local helper spelled `git(…)` — internal/themehistory has one,
// and a helper is the obvious thing to write once a package runs more than two
// commands. Matching the bare identifier is loose: a function called `git`
// that took something else would be read as a git invocation. That direction
// is the safe one — it can only produce a message naming a call somebody can
// look at — and the alternative is resolving identifiers across packages to
// check a flag.
//
// Only literal arguments are collected. A subcommand assembled at runtime is
// beyond what a parse can say, and there are none: every git call here names
// its subcommand as a constant, which is the property that makes this check
// possible rather than an assumption it makes.
func gitArgs(call *ast.CallExpr) ([]string, bool) {
	args := call.Args
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		if fn.Name != "git" {
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
