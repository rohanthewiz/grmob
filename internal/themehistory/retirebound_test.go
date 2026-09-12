package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// The argv retire's bound is stated for.
//
// Written here rather than read out of main.go, because a test that took the
// command from the code it is checking would agree with any command: the whole
// content of this rule is that somebody decided, once, which process may sit on
// the other end of those pipes, and that decision is this slice.
var batchArgv = []string{"git", "cat-file", "--batch"}

// The process behind a retired pipe is the one whose non-forking is what makes
// retire terminate.
//
// # The gap this closes
//
// retire kills a git that will not leave, and the Kill is what makes the wait
// after it finish: it closes the process's end of the pipe, the drain reaches
// EOF, and Wait returns. That chain has a precondition — Kill signals ONE
// process, so it reaches the pipe only if that process is the last thing
// holding the write end. True of a child that did not fork, and `git cat-file
// --batch` does not: it answers out of its own object store, spawns nothing,
// and is never passed the `--filters` that would have it run a clean or smudge
// filter over the bytes.
//
// The comment above retire says all of that, and says what to do if it stops
// being true — Setpgid at Start and a negative Kill, a process group per reader
// — and then says the cost is not worth paying for a case this command cannot
// produce. That argument is sound and it rests entirely on WHICH COMMAND, and
// nothing checked which command. A later edit that put a shell, a `git log`
// with a pager, or any porcelain that shells out behind those pipes would leave
// every sentence in that paragraph in place and make all of them false, and the
// symptom would be the one TestRetiringAWedgedProcessDoesNotWaitForever spent
// its first day on: a retire that never returns, under blobsMu, taking the
// whole run with it.
//
// # Why this is a source rule and not an assertion in newBatchReader
//
// Because the thing being held is a DECISION, not a value. newBatchReader
// builds the command itself, so a check inside it would be comparing the
// command with itself; what has to be pinned is that the package has exactly
// one place where a process is attached to a batchReader, and that the process
// there is still the one somebody proved terminates. Both halves are facts
// about the source, and the second is the one a runtime check cannot have an
// opinion about.
//
// # Why the test files are out of it
//
// main_test.go constructs a batchReader around `sleep`, deliberately: retire's
// deadline is what that test is about, and a cooperative git would never reach
// it. `sleep` is one process by construction — the test's own comment carries
// the shell-that-forked story that made it so — so it is not a counterexample
// to this rule, it is the rule's other arm, and a census that reported it would
// have to be silenced on the very file that exercises the bound.
func TestTheProcessBehindARetiredPipeDoesNotFork(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("reading this package's directory: %v", err)
	}
	var names []string
	for _, e := range entries {
		n := e.Name()
		if !e.IsDir() && strings.HasSuffix(n, ".go") && !strings.HasSuffix(n, "_test.go") {
			names = append(names, n)
		}
	}
	// Sorted for the reason every census in this package sorts: findings that
	// arrive in directory order cannot be diffed against the last run.
	sort.Strings(names)

	fset := token.NewFileSet()
	// Which functions attach a process to a batchReader, and what each one
	// starts. Keyed by function so a finding can name the place rather than the
	// file: "some file builds one" is not a sentence anybody can act on.
	builders := map[string]bool{}
	commands := map[string][][]string{}

	for _, name := range names {
		file, parseErr := parser.ParseFile(fset, name, nil, parser.SkipObjectResolution)
		if parseErr != nil {
			// A file go/parser cannot read is not this check's business — the
			// build says so first, and louder.
			continue
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				switch node := n.(type) {
				case *ast.CompositeLit:
					// `batchReader{…}` and `&batchReader{…}` are the same node
					// here: the & is a UnaryExpr around this one.
					if id, ok := node.Type.(*ast.Ident); ok && id.Name == "batchReader" {
						builders[fn.Name.Name] = true
					}
				case *ast.CallExpr:
					if argv, ok := execArgv(node); ok {
						commands[fn.Name.Name] = append(commands[fn.Name.Name], argv)
					}
				}
				return true
			})
		}
	}

	// First half: one place, and the one everybody's comments name.
	var built []string
	for fn := range builders {
		built = append(built, fn)
	}
	sort.Strings(built)
	if len(built) != 1 || built[0] != "newBatchReader" {
		t.Fatalf("this package attaches a process to a batchReader in %v, and the "+
			"bound retire is stated for is about one of them.\n\n"+
			"retire's deadline terminates because the Kill behind it reaches "+
			"whatever holds the pipe's write end, which is true for a child that "+
			"does not fork and false for one that does. `git cat-file --batch` is "+
			"the process that argument was made about, in newBatchReader, and this "+
			"rule exists so that a SECOND place — or a renamed first one — is a "+
			"failing test rather than a paragraph that has quietly stopped "+
			"describing the code. If the new site is deliberate, the change "+
			"retire's own comment names is Setpgid at Start and a negative Kill.",
			built)
	}

	// Second half: the process there is still the one that was proved to
	// terminate. Every exec in the builder, not just the first, because a
	// second one would be a second process nobody had reasoned about.
	got := commands["newBatchReader"]
	if len(got) != 1 || !sameArgv(got[0], batchArgv) {
		t.Fatalf("newBatchReader starts %v, and retire's bound is stated for "+
			"exactly %v.\n\n"+
			"The bound is: Kill reaches the one process it signals, so it closes "+
			"the pipe only while that process is the last thing holding the write "+
			"end. `git cat-file --batch` answers out of its own object store and "+
			"spawns nothing — not even a filter, since `--filters` is the opt-in "+
			"this never passes — so the drain after the Kill reaches EOF and Wait "+
			"returns. Another command makes that a guess: a shell, a porcelain "+
			"that pages, or this one with `--filters` all put a grandchild on the "+
			"write end, and a retire whose Kill does not reach it never returns — "+
			"under blobsMu, which is the whole run.\n\n"+
			"Two ways forward, and the choice is not this test's: prove the new "+
			"command does not fork and update batchArgv with the argument, or "+
			"make the bound unconditional the way retire's comment describes — "+
			"Setpgid at Start, a negative Kill, a process group per reader.",
			got, batchArgv)
	}
}

// The argv of an os/exec call, when every word of it is a literal.
//
// Both spellings, because CommandContext is the ordinary way this would be
// rewritten the day somebody wants a timeout on the start itself, and a rule
// that only knew the plain form would go quiet on exactly that edit.
//
// A call with a computed argument returns what it can with a marker in place of
// the word, which fails the comparison above rather than passing it: a command
// this test cannot read is a command nobody has proved anything about, and the
// safe direction for a rule whose failure is a question is to ask it.
func execArgv(call *ast.CallExpr) ([]string, bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil, false
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok || pkg.Name != "exec" {
		return nil, false
	}
	args := call.Args
	switch sel.Sel.Name {
	case "Command":
	case "CommandContext":
		// The context is not part of the argv.
		if len(args) > 0 {
			args = args[1:]
		}
	default:
		return nil, false
	}
	argv := make([]string, 0, len(args))
	for _, a := range args {
		lit, ok := a.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			argv = append(argv, "<not a literal>")
			continue
		}
		s, err := strconv.Unquote(lit.Value)
		if err != nil {
			argv = append(argv, "<unreadable literal>")
			continue
		}
		argv = append(argv, s)
	}
	return argv, true
}

func sameArgv(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
