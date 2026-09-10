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

// Every wall-clock record in this repository has the same five machine fields
// and the same reporting arm — and there are two of them.
//
// # What this is about, which is a duplication that is currently correct
//
// wasm/verify/timings_test.go and internal/themehistory/timings_test.go hold
// the same struct twice: `machine` for a reader, `goos`/`goarch`/`goVersion`/
// `cores` for an arm, and then whatever timings that package has. Each carries
// the same reporting test, which says whether this run is standing on the
// machine the numbers came from.
//
// The second one says out loud why it is a copy: the two are separate
// `package main` programs, one under wasm/ and one under internal/, and an
// `internal/timingrecord` imported by both would be a package existing so that
// two structs could be one. That is a good reason. It is a good reason for
// TWO — at three it stops being an argument about a shared package's cost and
// becomes an argument for keeping a shape in step by remembering to, which is
// exactly the kind of thing this repository writes arms instead of.
//
// So the trigger is written down rather than left as a judgement somebody has
// to make again from scratch when a fourth package grows a wall clock in a
// comment. Two: the copy is the cheaper option, and this check holds the two
// to being the same shape. Three: the check fails, and says what the decision
// is.
//
// # Why the shape and not the values
//
// The values are DIFFERENT NUMBERS about different machines-at-different-
// moments and were never meant to match; each record's own arm holds it to
// being filled in. What has to stay in step is the five fields and the arm
// over them, because that is what makes a number in this repository
// attributable at all — a record with no `cores` in it is a record a reader on
// another machine cannot use.
func TestEveryTimingsRecordIsTheSameShape(t *testing.T) {
	root := filepath.Join("..", "..")
	_, considered, from, err := citingFiles(root)
	if err != nil {
		t.Fatalf("enumerating the repository (%s): %v", from, err)
	}
	paths := make([]string, 0, len(considered))
	for p := range considered {
		paths = append(paths, p)
	}
	// Sorted for the reason every other walk in this package sorts: findings
	// that arrive in map order cannot be diffed against the last run.
	sort.Strings(paths)

	// dir -> the records declared in it, and whether the reporting arm is
	// there. Keyed by directory rather than by file because the arm and the
	// record need not share one, and the thing that has to hold is a property
	// of the package.
	type found struct {
		name string
		rel  string
		line int
	}
	var records []found
	arms := map[string]bool{}

	fset := token.NewFileSet()
	for _, rel := range paths {
		if !strings.HasSuffix(rel, ".go") {
			continue
		}
		// This file names the fields and the arm in its own prose and declares
		// neither. Skipped by name for the reason gitquoting_test.go skips
		// itself: a check that reads its own explanation as a finding is a
		// check nobody can leave a comment in.
		if rel == "wasm/verify/timingsrecords_test.go" {
			continue
		}
		src := filepath.Join(root, filepath.FromSlash(rel))
		file, parseErr := parser.ParseFile(fset, src, nil, parser.SkipObjectResolution)
		if parseErr != nil {
			// A file go/parser cannot read is not this check's business — the
			// build says so first, and every other reading here would be
			// failing too.
			continue
		}
		dir := path.Dir(rel)
		for _, d := range file.Decls {
			switch decl := d.(type) {
			case *ast.FuncDecl:
				if decl.Recv == nil && decl.Name.Name == timingsArm {
					arms[dir] = true
				}
			case *ast.GenDecl:
				if decl.Tok != token.VAR {
					continue
				}
				for _, sp := range decl.Specs {
					vs, ok := sp.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for i, n := range vs.Names {
						if !strings.HasSuffix(n.Name, "TimingsTakenOn") {
							continue
						}
						at := fset.Position(n.Pos())
						records = append(records, found{name: n.Name, rel: rel,
							line: at.Line})
						if i < len(vs.Values) {
							checkRecordShape(t, rel, at.Line, n.Name, vs.Values[i])
						}
					}
				}
			}
		}
	}

	// The walk reaching anything. Every arm in this package that walks the
	// repository says this, and for the same reason: a walk over nothing
	// passes silently and reads as a clean result — and this one is looking
	// for a NAME, which a rename would take away without touching a line of
	// the thing it names.
	if len(records) == 0 {
		t.Fatalf("no `…TimingsTakenOn` record was found in %d file(s) "+
			"enumerated by %s, and this repository has two: "+
			"wasm/verify/timings_test.go and "+
			"internal/themehistory/timings_test.go. Either the walk is not "+
			"reaching them or they have been renamed, and in both cases this "+
			"check is over nothing.", len(paths), from)
	}

	// The arm beside each record. A record with no reporting test is a struct
	// nothing reads: the numbers are attributed in the source and nothing puts
	// the attribution in front of the person holding a different reading.
	seen := map[string]bool{}
	for _, r := range records {
		dir := path.Dir(r.rel)
		seen[dir] = true
		if !arms[dir] {
			t.Errorf("%s:%d declares %s and %s declares no `func %s`.\n\n"+
				"A timings record with no reporting arm beside it is a struct "+
				"nothing reads. The record is how a number in a comment is "+
				"attributed; the arm is what puts that attribution in front of "+
				"the person holding a reading that disagrees with it, which is "+
				"the only moment either one is worth anything.",
				r.rel, r.line, r.name, dir, timingsArm)
		}
	}

	names := make([]string, 0, len(records))
	for _, r := range records {
		names = append(names, fmt.Sprintf("%s (%s:%d)", r.name, r.rel, r.line))
	}
	sort.Strings(names)

	// The trigger. See the header: two is a copy with a written reason, three
	// is a shape being kept in step by memory.
	if len(records) > timingsRecordCopies {
		t.Errorf("this repository has %d timings record(s) and the argument "+
			"for copying the struct was written for %d: %s.\n\n"+
			"internal/themehistory/timings_test.go says why it is a copy — two "+
			"separate `package main` programs, and an `internal/timingrecord` "+
			"imported by both would be a package existing so that two structs "+
			"could be one. That reasoning is about the cost of a package "+
			"against the cost of one duplicate. At %d it is no longer that "+
			"trade: the five machine fields and the reporting arm are being "+
			"kept in step across %d files by whoever remembers to, and this "+
			"check is the memory.\n\n"+
			"Extract the record and the arm into a package both can import, or "+
			"— if there is a reason the third is different — raise "+
			"timingsRecordCopies and write that reason down where the copy "+
			"argument is, so the next person reads a decision rather than a "+
			"number.",
			len(records), timingsRecordCopies, strings.Join(names, ", "),
			len(records), len(records))
	}

	t.Logf("%d timings record(s), each with the %d machine field(s) and a "+
		"`%s` in its package: %s. Enumerated by %s.",
		len(records), len(timingsMachineFields), timingsArm,
		strings.Join(names, ", "), from)
}

// How many copies of the record the written argument covers.
//
// Not a magic number and not a limit for its own sake: it is the number the
// reasoning in internal/themehistory/timings_test.go was written about. Moving
// it is a decision, which is why the failure above asks for the reason to be
// written beside that reasoning rather than just for the constant to go up.
//
// # The direction this fails in, which is chosen rather than overlooked
//
// A third timings record is a package that has grown a wall clock and written
// down which machine it came from — the right thing to do — and this check
// fails it. That is a CORRECT addition failing an arm, which is the shape
// hookSchemaReadFrom exists to apologise for: a key a later Claude Code adds
// is a key the tables have never heard of, so a correct settings.json fails a
// check meant to catch an incorrect one.
//
// The two are the same direction and they are not the same decision, because
// what a reader can do about them is not the same:
//
//	the hook tables    the failure is a MISTAKE about another program's
//	                   schema. Nothing here can tell it from a release that
//	                   moved, and softening the check would let `if` sit in
//	                   settings.json again — so the finding stays and arrives
//	                   with the two versions, which is all this side can
//	                   honestly offer
//	this constant      the failure IS the prompt. The third record is the
//	                   moment the trade changes, the change wanted is in THIS
//	                   repository, and the message asks for an extraction or a
//	                   written reason
//
// So the failing green run is the mechanism here and the residue there. The
// cost is the same in both: somebody reads a message and edits a file. The
// difference is that this one is asking for a decision it can name, and that
// is worth stating rather than leaving to be rediscovered as an inconsistency
// between two arms in the same repository.
const timingsRecordCopies = 2

// The reporting arm's name, which is the same in both packages and is what
// makes the two a pair rather than two structs that happen to look alike.
const timingsArm = "TestTheTimingsInThisPackageSayWhichMachineTheyCameFrom"

// The fields every record has, and the type each one is.
//
// The four machine fields are read by the arm — runtime answers all of them —
// and `machine` is the one for a person, which nothing checks and which is the
// only one that says what the computer actually was. The per-package timings
// are deliberately NOT listed: they are different numbers about different
// things, and a list of them here would be this check having an opinion about
// what a package is allowed to measure.
var timingsMachineFields = map[string]string{
	"machine":   "string",
	"goos":      "string",
	"goarch":    "string",
	"goVersion": "string",
	"cores":     "int",
}

// checkRecordShape holds one record's declaration to carrying the five machine
// fields, at the right types.
//
// The value is an anonymous struct literal in both records, which is what
// makes the shape readable from a parse: the fields are right there in the
// composite literal's type. A record that became a NAMED type elsewhere would
// fail this, and the message says so rather than reporting a missing field —
// the two are different findings and only one of them is a mistake.
func checkRecordShape(t *testing.T, rel string, line int, name string, val ast.Expr) {
	t.Helper()
	lit, ok := val.(*ast.CompositeLit)
	if !ok {
		t.Errorf("%s:%d declares %s and its value is %T, not a struct "+
			"literal. This check reads the five machine fields out of the "+
			"literal's own type; a record built some other way needs this "+
			"check taught how to read it.", rel, line, name, val)
		return
	}
	st, ok := lit.Type.(*ast.StructType)
	if !ok {
		t.Errorf("%s:%d declares %s as a %T rather than an anonymous struct.\n\n"+
			"If the record has been given a named type — which is what "+
			"sharing one between packages looks like — this check has been "+
			"overtaken by the change it exists to prompt, and should be "+
			"rewritten against that type instead of deleted.",
			rel, line, name, lit.Type)
		return
	}

	got := map[string]string{}
	for _, f := range st.Fields.List {
		typ := types(f.Type)
		for _, n := range f.Names {
			got[n.Name] = typ
		}
	}
	for _, field := range keysOf(timingsMachineFields) {
		want := timingsMachineFields[field]
		have, present := got[field]
		if !present {
			t.Errorf("%s:%d declares %s and it has no `%s %s` field.\n\n"+
				"The five machine fields are what makes a number in this "+
				"package's prose attributable: a reading a few percent off one "+
				"of them is a difference between two computers before it is "+
				"anything else, and a record short a field cannot say which "+
				"difference. The other record is the reference — see %s.",
				rel, line, name, field, want, timingsArm)
			continue
		}
		if have != want {
			t.Errorf("%s:%d declares %s with `%s %s` and the other record has "+
				"`%s %s`. The arm compares this against what runtime answers, "+
				"so the type is part of the comparison rather than a detail of "+
				"the declaration.", rel, line, name, field, have, field, want)
		}
	}
}

// types is an expression's type as it is written, for the field comparison
// above.
//
// Only the shapes a record's fields actually take are spelled out, which is
// identifiers and nothing else. Anything else comes back as a description
// rather than as a type name, so the message says what it found instead of
// comparing "" against "string" and reporting a field that is not there.
func types(e ast.Expr) string {
	if id, ok := e.(*ast.Ident); ok {
		return id.Name
	}
	return fmt.Sprintf("%T", e)
}
