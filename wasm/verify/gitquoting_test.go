package main

import (
	"fmt"
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
			// `git` in this repository runs git WITH THE ARGUMENTS IT WAS
			// GIVEN. A helper that did something else would make every call to
			// it a finding about the wrong thing, stated with a line number
			// and a subcommand.
			if why := whyNotAGitWrapper(fn); why != "" {
				t.Errorf("%s:%d declares `func git` and it %s.\n\n"+
					"Calls to a bare `git(…)` in %s are read by this check as "+
					"git command lines — `git(\"ls-tree\", \"--name-only\")` is "+
					"reported as `git ls-tree --name-only` — and that reading "+
					"is only true when the helper's own arguments are what the "+
					"process is handed. If this helper is something else, the "+
					"findings this check produces about that package are about "+
					"the wrong function: rename one of them, or teach gitArgs "+
					"which is which.", rel, at.Line, why, dir)
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
		"held above to running git with the arguments they are handed: %s. "+
		"Enumerated by %s.",
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

// whyNotAGitWrapper is why a `func git` declaration is not the thing this
// check reads its callers as, or "" if it is one.
//
// # What this used to be, and the shape that satisfied it
//
// It was: `exec.Command("git", …)` somewhere in the body, anywhere. That is
// the right FIRST question and it is not the whole one, because it is
// satisfied by a helper that calls git once for a reason of its own and does
// something else entirely with the arguments it was handed:
//
//	func git(args ...string) (string, error) {
//	    root, _ := exec.Command("git", "rev-parse", "--show-toplevel").Output()
//	    return run(filepath.Join(string(root), args[0]), args[1:]...)
//	}
//
// That passes the old test and every finding this check produces about its
// package is then a git command line assembled out of arguments that never
// reached git. The reading is not merely loose there — it is about a different
// program.
//
// # What is checked instead
//
// That the helper's VARIADIC PARAMETER reaches the git call. That is the
// premise the caller-side reading rests on: `git("ls-tree", "--name-only")` is
// read as `git ls-tree --name-only`, which is true exactly when those strings
// are what the process is given.
//
// It is a dataflow question, and dataflow is what go/types answers and a
// parse does not. This package parses FILES rather than loading packages —
// deliberately, since it walks a repository including revisions and generated
// trees that need not build — so what is done here is the intraprocedural,
// syntactic approximation of it:
//
//	the variadic parameter's names start out tainted
//	an assignment whose right-hand side mentions a tainted name taints its
//	  left-hand names
//	the premise holds if a tainted name reaches exec.Command("git", …) after
//	  the program name, or is assigned into a field of the *exec.Cmd that call
//	  produced — `cmd.Args = append(cmd.Args, args...)` is an ordinary wrapper
//	  and reaches git just as surely
//
// # What it is loose about
//
// Statements are taken in source order and nothing here understands control
// flow, so a taint inside an `if` that never runs still counts, and a
// parameter shadowed by a `for args := range …` is still the parameter. Both
// are in the direction of ACCEPTING a helper, which is the safe direction:
// this check's purpose is to stop a wrong premise being stated confidently,
// not to audit wrappers. A helper that would fool this has to launder its
// arguments through something with no syntactic connection to them at all.
func whyNotAGitWrapper(fn *ast.FuncDecl) string {
	if fn.Body == nil {
		return "has no body, so nothing in it reaches git"
	}
	// The variadic parameter. Without one, `git("ls-tree", "--name-only")`
	// cannot be a command line: the arguments are going to named parameters
	// that mean whatever the helper says they mean.
	tainted := map[string]bool{}
	if fn.Type.Params != nil && len(fn.Type.Params.List) > 0 {
		last := fn.Type.Params.List[len(fn.Type.Params.List)-1]
		if _, variadic := last.Type.(*ast.Ellipsis); variadic {
			for _, n := range last.Names {
				if n.Name != "_" {
					tainted[n.Name] = true
				}
			}
		}
	}
	if len(tainted) == 0 {
		return "declares no variadic parameter, so a call's arguments are not " +
			"a command line"
	}

	// Idents holding the *exec.Cmd that `exec.Command("git", …)` returned, so
	// a later `cmd.Args = …` can be recognised as reaching the same process.
	cmds := map[string]bool{}
	sawGit := false
	reaches := false

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if reaches {
			return false
		}
		switch node := n.(type) {
		case *ast.AssignStmt:
			// `cmd.Args = append(cmd.Args, args...)` — the argument list of a
			// command already built. Checked before the taint rule below,
			// because the left-hand side here is a field and not a name.
			for _, lhs := range node.Lhs {
				sel, ok := lhs.(*ast.SelectorExpr)
				if !ok {
					continue
				}
				base, ok := sel.X.(*ast.Ident)
				if !ok || !cmds[base.Name] {
					continue
				}
				if mentionsAny(node.Rhs, tainted) {
					reaches = true
					return false
				}
			}
			if mentionsAny(node.Rhs, tainted) {
				for _, lhs := range node.Lhs {
					if id, ok := lhs.(*ast.Ident); ok && id.Name != "_" {
						tainted[id.Name] = true
					}
				}
			}
			// And which name holds the command, for the field rule above.
			if len(node.Lhs) > 0 && len(node.Rhs) == 1 {
				if isGitCommandCall(node.Rhs[0]) {
					if id, ok := node.Lhs[0].(*ast.Ident); ok {
						cmds[id.Name] = true
					}
				}
			}
		case *ast.CallExpr:
			if !isGitCommandCall(node) {
				return true
			}
			sawGit = true
			// args[0] is the program name and is not part of the command line.
			if mentionsAny(node.Args[1:], tainted) {
				reaches = true
				return false
			}
		}
		return true
	})

	switch {
	case reaches:
		return ""
	case !sawGit:
		return "contains no exec.Command(\"git\", …)"
	default:
		return "runs git, and no argument of its own reaches that call"
	}
}

// isGitCommandCall is whether an expression is `exec.Command("git", …)`.
func isGitCommandCall(e ast.Expr) bool {
	call, ok := e.(*ast.CallExpr)
	if !ok || len(call.Args) == 0 {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Command" {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "exec" && literal(call.Args[0]) == "git"
}

// mentionsAny is whether any of these expressions names a tainted identifier.
//
// A selector's field is deliberately not looked at — only its base — so a
// parameter called `Args` is not matched by every `cmd.Args` in the body. What
// is being asked is which VALUE this expression is built out of, and the field
// name is not one.
func mentionsAny(exprs []ast.Expr, tainted map[string]bool) bool {
	found := false
	for _, e := range exprs {
		ast.Inspect(e, func(n ast.Node) bool {
			if found {
				return false
			}
			switch node := n.(type) {
			case *ast.SelectorExpr:
				ast.Inspect(node.X, func(inner ast.Node) bool {
					if id, ok := inner.(*ast.Ident); ok && tainted[id.Name] {
						found = true
					}
					return !found
				})
				return false
			case *ast.Ident:
				if tainted[node.Name] {
					found = true
				}
			}
			return !found
		})
		if found {
			return true
		}
	}
	return false
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

// The wrapper premise answers for the shapes a wrapper can take, and for the
// one it was tightened because of.
//
// # Why these are written here rather than found in the repository
//
// This repository declares exactly ONE `func git`, in internal/themehistory,
// and it is the plain shape: a variadic parameter handed straight to
// exec.Command. So every rule in whyNotAGitWrapper except the first is
// exercised by nothing, and the rule that matters most — a helper that runs
// git and does something else with its arguments — cannot be exercised by a
// repository that does not contain one.
//
// That is the same position runsGit was in when it was only asking whether the
// word `git` appeared in an exec.Command: the check passed, and what it passed
// ON was one helper that happened to be right. A premise held up by there
// being one instance of the thing it is about is the shape this whole session
// is written against.
func TestWhatCountsAsAGitWrapper(t *testing.T) {
	cases := []struct {
		name string
		src  string
		// The substring the reason must contain, or "" for a helper this
		// check accepts.
		want string
	}{{
		name: "the shape this repository has",
		src: `func git(args ...string) (string, error) {
			cmd := exec.Command("git", args...)
			var out bytes.Buffer
			cmd.Stdout = &out
			return out.String(), cmd.Run()
		}`,
	}, {
		name: "arguments joined onto a fixed prefix",
		src: `func git(args ...string) (string, error) {
			all := append([]string{"-C", root}, args...)
			return exec.Command("git", all...).Output()
		}`,
	}, {
		// A wrapper that builds the command first and fills its argument list
		// after. The arguments reach git just as surely, and a check that
		// rejected this would be rejecting a correct helper.
		name: "arguments appended to cmd.Args",
		src: `func git(args ...string) (string, error) {
			cmd := exec.Command("git")
			cmd.Args = append(cmd.Args, args...)
			return "", cmd.Run()
		}`,
	}, {
		// The item this test is here for. It runs git, so the old check was
		// satisfied — and the arguments go somewhere else entirely, so every
		// finding about this package would have been a command line assembled
		// out of strings git never saw.
		name: "runs git, and the arguments go elsewhere",
		src: `func git(args ...string) (string, error) {
			root, _ := exec.Command("git", "rev-parse", "--show-toplevel").Output()
			return run(filepath.Join(string(root), args[0]), args[1:]...)
		}`,
		want: "no argument of its own reaches that call",
	}, {
		name: "does not run git at all",
		src: `func git(args ...string) (string, error) {
			return exec.Command("hg", args...).Output()
		}`,
		want: "contains no exec.Command",
	}, {
		// Without a variadic parameter a call's arguments are not a command
		// line at all — they are whatever this signature says they are, and
		// reading them in order as `git <a> <b>` is a guess about a helper
		// this file has never seen.
		name: "not variadic",
		src: `func git(sub string, paths []string) (string, error) {
			return exec.Command("git", append([]string{sub}, paths...)...).Output()
		}`,
		want: "declares no variadic parameter",
	}, {
		name: "declared, never defined",
		src:  `func git(args ...string) (string, error)`,
		want: "has no body",
	}}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fn := parseOneFunc(t, c.src)
			got := whyNotAGitWrapper(fn)
			switch {
			case c.want == "" && got != "":
				t.Errorf("this is a git wrapper and the check rejected it: %s\n\n"+
					"A rejection here is reported against the package that "+
					"declares the helper and stops nothing else — but it is a "+
					"finding about a correct file, which is the one kind of "+
					"noise a check nobody can weigh cannot afford.\n\n%s",
					got, c.src)
			case c.want != "" && !strings.Contains(got, c.want):
				t.Errorf("expected a reason containing %q and got %q.\n\n%s",
					c.want, got, c.src)
			}
		})
	}
}

// The taint walk is the size the reason for having it covers, and every rule
// in it is exercised by a case.
//
// # What this is about, which is an approximation that will be asked to grow
//
// whyNotAGitWrapper answers a DATAFLOW question — do this helper's own
// arguments reach the process — with a syntactic walk, because this package
// parses FILES rather than loading packages. That reason is good and it is
// written down once, in that function's header: the walk covers revisions and
// generated trees that need not build, and go/types needs a package that does.
//
// That reason is now carrying two approximations in this package rather than
// one. The other is themenearmiss's float census, which declines the type
// checker in the same words and for the same cost — "a second toolchain inside
// a test suite that imports half the repository" — and which has grown a
// name-keyed table per question it could not answer, four of them, plus a
// count of the collisions it knows it cannot resolve. That is what an
// approximation looks like after several sessions of obviously-correct
// additions, and it is the shape this one is on the first rung of.
//
// So the moment to notice is the THIRD rule, not the sixth.
//
//	rules the reason covers   two, both listed below, each with a case
//	the third                 this arm fails and says what the decision is
//
// # Why the rules are counted out of the source rather than listed
//
// A census somebody has to remember to update is the thing this repository
// writes arms instead of. So the count comes from the function itself: an
// acceptance in whyNotAGitWrapper is a `reaches = true`, and the walk below
// counts them. A rule added without a row here fails this check, in the
// commit that adds it, which is the only moment the trade is being made.
//
// The one loophole in counting that way is an acceptance spelled some other
// way — an early `return ""` in the middle of the walk — so the arm also holds
// the function to having exactly ONE `return ""`, the accepting arm of the
// switch at its end. With that, `reaches` is the only way out.
//
// # The two ways round that, which are closed here rather than named
//
// Counting `reaches = true` is a syntactic count of a syntactic thing, and it
// trusted the shape of the function twice over. Both gaps are the same move —
// an acceptance that happens somewhere this census does not read — and both
// are now checks rather than assumptions:
//
//	reaches = reachesVia(…)   an assignment to `reaches` whose right-hand side
//	                          is not the literal `true`. The count would not
//	                          move and the rule would be live, with its content
//	                          in a function nothing here counts. Every
//	                          assignment to `reaches` is held to `= true`
//	if acceptsVia(…) { … }    a helper called from the body, deciding the rule
//	                          and leaving `reaches = true` as its punctuation.
//	                          The callees are held to gitWrapperTaintHelpers,
//	                          so a new one is a row and a decision in the
//	                          commit that adds it
//
// What remains beyond that is one level further out and is not closable by a
// parse of this function: the helpers themselves could grow. They are three
// lines each and listed by name, which is the point of listing them.
func TestTheGitWrapperTaintWalkIsTheSizeItsReasonCovers(t *testing.T) {
	const self = "gitquoting_test.go"
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, self, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parsing %s, which is this file: %v", self, err)
	}
	var fn *ast.FuncDecl
	for _, d := range file.Decls {
		if f, ok := d.(*ast.FuncDecl); ok && f.Recv == nil &&
			f.Name.Name == "whyNotAGitWrapper" {
			fn = f
		}
	}
	if fn == nil || fn.Body == nil {
		t.Fatalf("%s declares no `func whyNotAGitWrapper` with a body. It is "+
			"in this file and this check counts the rules inside it; a rename "+
			"leaves this arm passing over nothing.", self)
	}

	accepts := 0
	emptyReturns := 0
	// Assignments to `reaches` whose right-hand side is not the literal
	// `true`, and unqualified calls the body makes. Both are ways an
	// acceptance rule can live somewhere the count above does not read; see
	// the header.
	var indirect []string
	callees := map[string]bool{}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			// Unqualified only: a `pkg.Fn(…)` is the standard library, which
			// cannot assign to a local. A call to a function in this package
			// is spelled as a bare name and is the shape being watched for.
			if id, ok := node.Fun.(*ast.Ident); ok {
				callees[id.Name] = true
			}
			return true
		case *ast.AssignStmt:
			// `reaches = true` — one acceptance rule, wherever it sits.
			if len(node.Lhs) != 1 || len(node.Rhs) != 1 {
				return true
			}
			id, ok := node.Lhs[0].(*ast.Ident)
			if !ok || id.Name != "reaches" {
				return true
			}
			lit, isIdent := node.Rhs[0].(*ast.Ident)
			switch {
			case node.Tok == token.ASSIGN && isIdent && lit.Name == "true":
				accepts++
			case node.Tok == token.DEFINE && isIdent && lit.Name == "false":
				// `reaches := false`, the declaration the rules write into.
				// Not a rule, and the only `:=` this name takes.
			default:
				// An acceptance whose CONTENT is elsewhere. Recorded with its
				// line so the message can point at it rather than describe the
				// shape.
				indirect = append(indirect, fmt.Sprintf("line %d",
					fset.Position(node.Pos()).Line))
			}
		case *ast.ReturnStmt:
			// An acceptance spelled as a return rather than through
			// `reaches`. One of these is the switch's own accepting arm.
			if len(node.Results) != 1 {
				return true
			}
			if literal(node.Results[0]) == "" &&
				isEmptyStringLit(node.Results[0]) {
				emptyReturns++
			}
		}
		return true
	})

	if len(indirect) > 0 {
		t.Errorf("whyNotAGitWrapper assigns to `reaches` from something other "+
			"than the literal `true` at %s, and this check counts its rules by "+
			"counting `reaches = true`.\n\n"+
			"An assignment like `reaches = reachesVia(...)` is an acceptance "+
			"rule whose content is in another function: the count here does "+
			"not move, the budget below is not the number of rules there are, "+
			"and the looseness of the new rule is written down nowhere. Spell "+
			"it as a condition around `reaches = true` — which is what every "+
			"rule in this function already is — or teach this check the other "+
			"spelling.", strings.Join(indirect, ", "))
	}

	// The helpers this function is allowed to call. A rule decided inside one
	// of them is a rule this census cannot see, so a new callee is a row in
	// the table and a decision about whether the approximation has stopped
	// being one function.
	for name := range callees {
		if gitWrapperTaintHelpers[name] != "" {
			continue
		}
		var known []string
		for h := range gitWrapperTaintHelpers {
			known = append(known, h)
		}
		sort.Strings(known)
		t.Errorf("whyNotAGitWrapper calls %s(…), which is not one of the "+
			"helpers this census knows about: %s.\n\n"+
			"The rule count above reads THIS function's body and nothing else, "+
			"so a helper that decides an acceptance — `if acceptsVia(x) { "+
			"reaches = true }` — is a rule whose content the census never "+
			"hears about, and whose looseness gitWrapperTaintLimits does not "+
			"describe. Add a row to gitWrapperTaintHelpers saying what the new "+
			"one answers, and while writing it, the question the row is "+
			"really asking: whether the approximation is still one function a "+
			"reader can hold in their head, which is what its budget of %d is "+
			"about.", name, strings.Join(known, ", "), gitWrapperAcceptRules)
	}

	if emptyReturns != 1 {
		t.Errorf("whyNotAGitWrapper has %d `return \"\"` and this check counts "+
			"its rules by counting `reaches = true`.\n\n"+
			"One is the accepting arm of the switch at the end of the "+
			"function. A second is an acceptance rule that does not go through "+
			"`reaches`, which means the count below is short and the budget "+
			"this arm is holding is not the number of rules there are. Route "+
			"the new rule through `reaches` — the switch is what turns it into "+
			"a return — or teach this check the other spelling.", emptyReturns)
	}
	if accepts != len(gitWrapperTaintRules) {
		t.Errorf("whyNotAGitWrapper accepts a helper in %d place(s) and "+
			"gitWrapperTaintRules lists %d.\n\n"+
			"The census is where a reader finds out what the approximation "+
			"actually covers and which case exercises each part of it — the "+
			"question nobody could answer about `runsGit`, which passed on "+
			"one helper that happened to be right. A rule with no row here is "+
			"a rule whose looseness nobody has written down.",
			accepts, len(gitWrapperTaintRules))
	}

	// Every row naming a case that is really in the table, because a row
	// pointing at a case that has been renamed away is a rule nothing
	// exercises and a census that reads as though something does.
	cases := taintCaseNames(t, file)
	for _, r := range gitWrapperTaintRules {
		if !cases[r.exercisedBy] {
			t.Errorf("gitWrapperTaintRules says %q is exercised by the case "+
				"%q, and %s declares no such case in %s.\n\n"+
				"Cases in this file are the only thing that exercises these "+
				"rules: this repository declares exactly ONE `func git` and it "+
				"is the plainest shape there is, so every rule but the first "+
				"is covered by the table or by nothing.",
				r.what, r.exercisedBy, self, gitWrapperCaseTable)
		}
	}

	// The trigger. Two is the number the reason in whyNotAGitWrapper's header
	// was written about; the third is where the trade changes.
	if accepts > gitWrapperAcceptRules {
		t.Errorf("whyNotAGitWrapper now accepts a helper in %d place(s) and "+
			"the argument for approximating this syntactically was written "+
			"for %d.\n\n"+
			"That argument is about a parse being able to read revisions and "+
			"generated trees that need not build, against the cost of one "+
			"loose rule. At %d rules it is a different trade: the function is "+
			"a small dataflow engine with its own looseness table, answering "+
			"badly a question go/types answers exactly, and the next case "+
			"will look just as obviously correct as this one did.\n\n"+
			"Either load the package and ask go/types — and say what that "+
			"costs the walk over revisions, which is the reason it was not "+
			"done — or raise gitWrapperAcceptRules and write the reason "+
			"beside the header's argument, so the next person reads a "+
			"decision rather than a number.",
			accepts, gitWrapperAcceptRules, accepts)
	}

	t.Logf("whyNotAGitWrapper accepts a helper in %d place(s), each spelled "+
		"`reaches = true`, each with a row in gitWrapperTaintRules and a case "+
		"in %s; %d looseness(es) written down beside them, and %d helper(s) "+
		"called out of a body with one way out. Counted out of %s rather than "+
		"listed, so a rule added without a row fails this arm in the commit "+
		"that adds it.",
		accepts, gitWrapperCaseTable, len(gitWrapperTaintLimits),
		len(callees), self)
}

// How many acceptance rules the written reason covers. Two, and the third one
// is a decision — see the arm above, and timingsRecordCopies, which is the
// same shape of constant for the same kind of reason.
const gitWrapperAcceptRules = 2

// The test whose table is the only thing exercising those rules.
const gitWrapperCaseTable = "TestWhatCountsAsAGitWrapper"

// What the approximation accepts a helper ON, one row per `reaches = true`.
var gitWrapperTaintRules = []struct {
	// The rule, as the premise it establishes.
	what string
	// The case in TestWhatCountsAsAGitWrapper that would fail if it went.
	exercisedBy string
}{{
	what: "a tainted name reaches exec.Command(\"git\", …) past the program " +
		"name, which is the wrapper this repository actually has",
	exercisedBy: "the shape this repository has",
}, {
	what: "a tainted name is assigned into a field of the *exec.Cmd that call " +
		"produced — `cmd.Args = append(cmd.Args, args...)` reaches git just " +
		"as surely",
	exercisedBy: "arguments appended to cmd.Args",
}}

// The functions whyNotAGitWrapper is allowed to call, and what each answers.
//
// Not a style rule. The rule count above is a walk over one function's body,
// so a helper is where an acceptance can be decided without the census
// noticing — `if acceptsVia(x) { reaches = true }` reads as one rule and is
// two. Each row says what the helper answers, and what makes it not a rule is
// in the answer: none of these three has an opinion about whether the premise
// holds, they report a syntactic fact the walk then decides on.
//
// `len` is here because it is spelled the same way a local helper is. The
// standard library is not, and does not need to be: a `pkg.Fn(…)` cannot
// assign to a local in this function, so the walk only collects bare names.
var gitWrapperTaintHelpers = map[string]string{
	"len":              "a builtin, and unqualified like everything else here",
	"mentionsAny":      "whether an expression names a tainted identifier",
	"isGitCommandCall": "whether an expression is exec.Command(\"git\", …)",
}

// And what it is loose about, which is the half a rule count does not say.
//
// Each of these is in the direction of ACCEPTING a helper, which is the safe
// direction for this check: its purpose is to stop a wrong premise being
// stated confidently about a package's git calls, not to audit wrappers. They
// are listed because a third rule arriving would be arriving on top of these,
// and the question at that point is whether the pile is still cheaper than
// loading a package.
var gitWrapperTaintLimits = []string{
	"statements are taken in source order and nothing here understands " +
		"control flow, so a taint inside an `if` that never runs counts",
	"nothing tracks scope, so a parameter shadowed by a `for args := range …` " +
		"is still the parameter",
	"only this function's body is read, so a helper that hands its arguments " +
		"to another function in the package is not followed",
}

// isEmptyStringLit is whether an expression is the literal "".
//
// literal() returns "" both for an empty string literal and for anything that
// is not a literal at all, and the count above needs those told apart: a
// `return err.Error()` is not an acceptance.
func isEmptyStringLit(e ast.Expr) bool {
	lit, ok := e.(*ast.BasicLit)
	return ok && lit.Kind == token.STRING && (lit.Value == `""` || lit.Value == "``")
}

// taintCaseNames is the `name:` of every case in the wrapper table.
//
// Read out of the test's own source rather than by running it, because what
// the census claims is that a rule is exercised BY A CASE — a fact about the
// table, which a parse can settle without the cases having to pass.
func taintCaseNames(t *testing.T, file *ast.File) map[string]bool {
	t.Helper()
	names := map[string]bool{}
	for _, d := range file.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Name.Name != gitWrapperCaseTable {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			kv, ok := n.(*ast.KeyValueExpr)
			if !ok {
				return true
			}
			key, ok := kv.Key.(*ast.Ident)
			if !ok || key.Name != "name" {
				return true
			}
			if lit, ok := kv.Value.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				names[strings.Trim(lit.Value, "`\"")] = true
			}
			return true
		})
	}
	if len(names) == 0 {
		t.Fatalf("no case name was read out of %s. The census rows below "+
			"claim each rule is exercised by one of them, and with none found "+
			"that claim is checked against nothing.", gitWrapperCaseTable)
	}
	return names
}

// parseOneFunc is the declaration in a fragment of Go source.
//
// Wrapped in a package clause and the imports the fragments use, because
// go/parser wants a file and the fragments are written as bodies. Parsed with
// SkipObjectResolution for the reason the walk above uses it: nothing here
// resolves an identifier to a declaration, and the resolution pass is the
// expensive half.
func parseOneFunc(t *testing.T, src string) *ast.FuncDecl {
	t.Helper()
	const preamble = "package p\n\nimport (\n\t\"bytes\"\n\t\"os/exec\"\n\t" +
		"\"path/filepath\"\n)\n\nvar root string\n\n" +
		"func run(string, ...string) (string, error) { return \"\", nil }\n\n"
	file, err := parser.ParseFile(token.NewFileSet(), "fragment.go",
		preamble+src+"\n", parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("the fragment does not parse: %v\n\n%s", err, src)
	}
	for _, d := range file.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok && fn.Name.Name == "git" {
			return fn
		}
	}
	t.Fatalf("the fragment declares no `func git`:\n%s", src)
	return nil
}
