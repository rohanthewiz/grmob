package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// This repository hand-rolls ONE parser for dotted versions, and the reason
// for hand-rolling it was written for one.
//
// # What this is about
//
// hookconfig_test.go needs to know whether the `claude` on this machine is
// ahead of or behind the build its key tables were read from, because those
// are opposite findings — a key a later release added, against a key an older
// install has. That turns on ORDER, so compareVersions orders two dotted
// versions: numeric field by field, a missing field is 0, a pre-release tail
// is behind its release, and anything it cannot read comes back unordered
// rather than as a side.
//
// Twenty lines and no dependency was the right call for one caller. It is the
// SECOND caller that changes the trade, and the change is invisible at the
// moment it happens: a second orderer arrives as an obviously-correct twenty
// lines somewhere else, and what this repository then has is two hand-rolled
// comparators that agree until the day they do not.
//
// # What the alternative actually is, since the comment used to get it wrong
//
// compareVersions' header said "no dependency, against a repository whose
// go.mod has one line in it". Both halves were wrong. go.mod has several
// requirements, and one of them is `golang.org/x/mod` — held indirect by the
// `tool` block that pins gomobile — so importing golang.org/x/mod/semver moves
// a line in go.mod rather than adding a download.
//
// And semver answers every rule compareVersions has, which was checked rather
// than assumed:
//
//	semver.Compare("v2.1.267", "v2.1.30")       +1   numeric, not lexical
//	semver.Compare("v2.1", "v2.1.0")             0   a missing field is 0
//	semver.Compare("v2.2.0-nightly", "v2.2.0")  -1   a pre-release is behind
//	semver.IsValid on anything else           false   the unordered case
//
// So the residue of the argument is small and it is not nothing: semver wants
// a leading `v` and `claude --version` prints `2.1.267`, so a caller wraps it
// either way. One caller with a wrapper is a wash; two is not, and at two the
// wrapper is the thing that should be shared rather than the parser.
//
// # Why the count is taken out of the source
//
// A census somebody has to remember to update is what this repository writes
// arms instead of — see TestTheGitWrapperTaintWalkIsTheSizeItsReasonCovers and
// TestEveryTimingsRecordIsTheSameShape, which are the same shape of check for
// the same kind of reason.
//
// The syntactic handle is the PARSER rather than the comparator, because an
// orderer cannot be recognised by shape and a parser can: ordering dotted
// versions means having the fields as numbers, which means splitting a string
// on "." and converting the pieces. A function that does both is a dotted
// version being taken apart by hand, whatever the function is called.
//
//	what it finds        one function body that splits on "." and runs the
//	                     pieces through strconv — versionFields, which is what
//	                     compareVersions is built on
//	what it will not     an orderer that never splits, because it was written
//	                     on top of x/mod/semver. That is the change this arm
//	                     is asking for, so not seeing it is correct
//	what it might        a function that splits some other dotted thing and
//	                     parses it. There is none today; one arriving is a row
//	                     for the table below and a decision either way, which
//	                     is the point of a failing arm rather than a silent one
func TestTheDottedVersionParsersAreTheOnesTheReasonCovers(t *testing.T) {
	root := filepath.Join("..", "..")
	_, considered, from, err := citingFiles(root)
	if err != nil {
		t.Fatalf("enumerating the repository (%s): %v", from, err)
	}
	paths := make([]string, 0, len(considered))
	for p := range considered {
		if strings.HasSuffix(p, ".go") {
			paths = append(paths, p)
		}
	}
	// Sorted for the reason every walk in this package sorts: findings that
	// arrive in map order cannot be diffed against the last run.
	sort.Strings(paths)

	var found []string
	fset := token.NewFileSet()
	for _, rel := range paths {
		src := filepath.Join(root, filepath.FromSlash(rel))
		file, parseErr := parser.ParseFile(fset, src, nil, parser.SkipObjectResolution)
		if parseErr != nil {
			// A file go/parser cannot read is not this check's business — the
			// build says so first, and every other reading here would be
			// failing too.
			continue
		}
		for _, d := range file.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			if !splitsOnADot(fn.Body) || !parsesANumber(fn.Body) {
				continue
			}
			found = append(found, fmt.Sprintf("%s (%s:%d)", fn.Name.Name, rel,
				fset.Position(fn.Pos()).Line))
		}
	}

	// The walk reaching anything. Every arm in this package that walks the
	// repository says this: a walk over nothing passes silently and reads as a
	// clean result, and this one is looking for a SHAPE that a rewrite could
	// take away without anybody meaning to.
	if len(found) == 0 {
		t.Fatalf("no function that splits a string on \".\" and parses the "+
			"pieces was found in %d Go file(s) enumerated by %s, and this "+
			"repository has one: versionFields, in hookconfig_test.go.\n\n"+
			"Either the walk is not reaching it, or compareVersions has been "+
			"rewritten on top of something that does the parsing — which is "+
			"the change this check exists to prompt, and which should retire "+
			"this arm in the same commit rather than leave it passing over "+
			"nothing.", len(paths), from)
	}

	// Every row of the table naming a file that is still there, because a row
	// pointing at a renamed file is a reader nothing describes and a census
	// that reads as though something does.
	for _, r := range versionReaders {
		if _, ok := considered[r.file]; !ok {
			t.Errorf("versionReaders describes %s in %s, and %s enumerated no "+
				"such file.\n\n"+
				"These rows are what a reader of the count below finds out "+
				"what the one comparator sits NEXT TO — the places that read a "+
				"version and decide something without ordering it. A row for a "+
				"file that has moved describes nothing.", r.what, r.file, from)
		}
	}

	if len(found) > dottedVersionParsers {
		t.Errorf("this repository takes a dotted version apart by hand in %d "+
			"place(s) and the argument for doing it by hand was written for "+
			"%d: %s.\n\n"+
			"That argument is in compareVersions' header and it is about ONE "+
			"caller: a wrapper around golang.org/x/mod/semver is needed either "+
			"way, because semver wants a leading `v` and the versions this "+
			"repository reads do not have one, so for a single caller the "+
			"wrapper and the parser cost about the same. At %d it is a "+
			"different trade — two hand-rolled comparators agree until the "+
			"day they do not, and x/mod is already in this module's graph as "+
			"an indirect requirement, so the import moves a line in go.mod "+
			"rather than adding a dependency.\n\n"+
			"Either put the callers on one comparator — semver answers every "+
			"rule compareVersions has; the four of them are checked off in "+
			"this test's header — or raise dottedVersionParsers and write the "+
			"reason beside that header, so the next person reads a decision "+
			"rather than a number.",
			len(found), dottedVersionParsers, strings.Join(found, ", "),
			len(found))
	}

	t.Logf("%d dotted-version parser(s) in %d Go file(s): %s. %d other place(s) "+
		"read a version without ordering one; see versionReaders. Enumerated "+
		"by %s.", len(found), len(paths), strings.Join(found, ", "),
		len(versionReaders), from)
}

// How many hand-rolled dotted-version parsers the written reason covers.
//
// One, and the second is a decision rather than a number to raise — see the
// arm above, and gitWrapperAcceptRules and timingsRecordCopies, which are the
// same shape of constant for the same kind of reason.
//
// # The direction this fails in
//
// A second one is somebody writing twenty correct lines, which this arm fails.
// That is the same deliberate direction timingsRecordCopies documents: the
// failure IS the prompt, the change being asked for is in THIS repository, and
// the message names what to do rather than reporting a mistake.
const dottedVersionParsers = 1

// The places that read a version and do NOT order one.
//
// Listed because a count of comparators does not say what the comparator sits
// beside, and these are what a second orderer would most likely grow out of:
// each is a version being read out of text for a decision, and three of the
// four decisions happen to be equality. The fourth was, until it needed a
// direction — which is how the one comparator got here.
var versionReaders = []struct {
	file string
	what string
}{{
	file: "wasm/verify/hookconfig_test.go",
	what: "compareVersions, the only orderer: `claude --version` against the " +
		"build the hook key tables were read from, where ahead and behind are " +
		"opposite findings",
}, {
	file: "wasm/verify/timings_test.go",
	what: "runtime.Version() against a recorded toolchain by PREFIX — a patch " +
		"release is a different toolchain and worth naming, so what is wanted " +
		"is `is this the same build`, not an order",
}, {
	file: "internal/themehistory/timings_test.go",
	what: "the same prefix comparison, in the other timings record. Held to " +
		"being the same shape by TestEveryTimingsRecordIsTheSameShape",
}, {
	file: "mobile/verify/composelayout_test.go",
	what: "a Compose release DERIVED out of the BOM's own pom and used to " +
		"find a jar — read the way gradle reads it, and compared to nothing",
}}

// splitsOnADot is whether this body splits a string on the literal ".".
//
// The spellings a version parser realistically uses, and no more: a wider net
// here — every strings call with a "." in it — would match path handling all
// over the repository and turn the count above into a list somebody has to
// argue with.
func splitsOnADot(body *ast.BlockStmt) bool {
	return callsStrings(body, map[string]bool{
		"Split": true, "SplitN": true, "SplitSeq": true, "Cut": true,
	}, ".")
}

// parsesANumber is whether this body converts a string to an integer.
func parsesANumber(body *ast.BlockStmt) bool {
	return callsStrconv(body, map[string]bool{
		"Atoi": true, "ParseInt": true, "ParseUint": true,
	})
}

// callsStrings is whether the body calls one of these strings functions with
// `lit` as its second argument.
func callsStrings(body *ast.BlockStmt, names map[string]bool, lit string) bool {
	hit := false
	ast.Inspect(body, func(n ast.Node) bool {
		if hit {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) < 2 {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || !names[sel.Sel.Name] {
			return true
		}
		pkg, ok := sel.X.(*ast.Ident)
		if !ok || pkg.Name != "strings" {
			return true
		}
		if literal(call.Args[1]) == lit {
			hit = true
			return false
		}
		return true
	})
	return hit
}

// callsStrconv is whether the body calls one of these strconv functions.
func callsStrconv(body *ast.BlockStmt, names map[string]bool) bool {
	hit := false
	ast.Inspect(body, func(n ast.Node) bool {
		if hit {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || !names[sel.Sel.Name] {
			return true
		}
		if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "strconv" {
			hit = true
			return false
		}
		return true
	})
	return hit
}
