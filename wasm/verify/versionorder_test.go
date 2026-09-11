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

// This repository has ONE comparator for dotted versions, and the reason for
// the way it is built was written for one.
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
// TestTheQuestionsOnTheSharedRepositoryParseAreTheOnesDecidedOn, which are
// the same shape of check for the same kind of reason.
//
// # What is being counted, which used to be the wrong noun
//
// This arm counted HAND-ROLLED PARSERS. The syntactic handle was the parser
// because an orderer cannot be recognised by shape and a parser can: ordering
// dotted versions means having the fields as numbers, which means splitting a
// string on "." and converting the pieces.
//
// That handle is right and the noun was wrong, and the two came apart at
// exactly the change this arm exists to prompt. An orderer built on
// golang.org/x/mod/semver never splits anything, so it was invisible here —
// written down as correct behaviour, on the reasoning that a semver-based
// comparator IS the change being asked for. It is, right up until somebody
// adds one and leaves versionFields where it is. The repository then holds two
// comparators that agree until the day they do not, and the count still reads
// one: the zero-found case covered both of them going, and nothing covered one
// of them ARRIVING.
//
// So the budget is about COMPARATORS and the kind is a column. They are the
// same number today, which is why the distinction could be left alone, and
// they will not be at the moment it matters.
//
//	what it finds        every function that orders dotted versions, by either
//	                     construction — a body that splits on "." and runs the
//	                     pieces through strconv (versionFields, which is what
//	                     compareVersions is built on), or one that calls
//	                     x/mod/semver's own orderer
//	what it will not     an orderer that neither splits nor calls semver —
//	                     one built on a third dependency, say. That is a
//	                     dependency arriving, which go.mod and the build both
//	                     report, and this arm does not
//	what it might        a function that splits some other dotted thing and
//	                     parses it. There is none today; one arriving is a row
//	                     for the table below and a decision either way, which
//	                     is the point of a failing arm rather than a silent one
//
// # What the two kinds mean when the count is right
//
//	1 hand-rolled, 0 semver   today. The twenty lines, and the reason above
//	1 semver, 0 hand-rolled   the change this arm asks for, having been made.
//	                          Passes, and is meant to
//	2 of anything             two comparators. The failure, whichever way they
//	                          are built, because that is the thing the reason
//	                          is about
//
// # And which `semver` is x/mod's, which is the same blind spot one along
//
// Adding the semver construction closed the gap at the level of the NOUN and
// left it open at the level of the identifier: a detector that matches the
// qualifier `semver` does not see `import sv "golang.org/x/mod/semver"`, which
// is a second comparator arriving in precisely the form this arm's message
// asks somebody to write. So the qualifiers here — semver's, and the `strings`
// and `strconv` the hand-rolled construction is made of — are read off each
// file's own import block. That is exact in both directions and costs nothing
// on a parse that has already happened; see importnames_test.go, which says
// what it still cannot resolve and reports rather than skips it.
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

	var found []versionComparator
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
		// Which identifiers THIS FILE binds to the three packages the two
		// constructions are made of. Read off the import block rather than
		// taken from the conventional spelling: `import sv
		// "golang.org/x/mod/semver"` is the way a second comparator arrives
		// and is invisible here, and it is invisible in SILENCE — see
		// importnames_test.go. Resolved once per file, because an import is a
		// fact about a file and not about a declaration.
		semverNames := qualifiersFor(t, rel, file, "golang.org/x/mod/semver")
		stringsNames := qualifiersFor(t, rel, file, "strings")
		strconvNames := qualifiersFor(t, rel, file, "strconv")
		for _, d := range file.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			// Both constructions, and a function that is somehow both is
			// reported once as hand-rolled: the split is what the reason in
			// compareVersions' header is about, and it is the half that would
			// need rewriting.
			kind := ""
			switch {
			case splitsOnADot(fn.Body, stringsNames) &&
				parsesANumber(fn.Body, strconvNames):
				kind = "hand-rolled"
			case ordersViaSemver(fn.Body, semverNames):
				kind = "x/mod/semver"
			default:
				continue
			}
			found = append(found, versionComparator{
				kind: kind,
				name: fn.Name.Name,
				at: fmt.Sprintf("%s:%d", rel,
					fset.Position(fn.Pos()).Line),
			})
		}
	}
	// Sorted so a run's findings can be diffed against the last one; the file
	// order above is already stable, and this makes the two kinds read
	// together.
	sort.Slice(found, func(i, j int) bool {
		if found[i].kind != found[j].kind {
			return found[i].kind < found[j].kind
		}
		return found[i].at < found[j].at
	})

	// The walk reaching anything. Every arm in this package that walks the
	// repository says this: a walk over nothing passes silently and reads as a
	// clean result, and this one is looking for a SHAPE that a rewrite could
	// take away without anybody meaning to.
	if len(found) == 0 {
		t.Fatalf("no function that orders dotted versions was found in %d Go "+
			"file(s) enumerated by %s, and this repository has one: "+
			"versionFields, in hookconfig_test.go, which compareVersions is "+
			"built on.\n\n"+
			"Both constructions are looked for — a split-and-parse, and a call "+
			"into x/mod/semver — so this is not the arm's own change having "+
			"been made. Either the walk is not reaching the file, or "+
			"hookconfig_test.go no longer orders versions at all, in which "+
			"case the question this arm exists for has gone with it and this "+
			"check should be retired in the same commit rather than left "+
			"passing over nothing.", len(paths), from)
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

	if len(found) > dottedVersionComparators {
		t.Errorf("this repository orders dotted versions in %d place(s) and "+
			"the argument for the way the one is built was written for %d: "+
			"%s.\n\n"+
			"That argument is in compareVersions' header and it is about ONE "+
			"caller: a wrapper around golang.org/x/mod/semver is needed either "+
			"way, because semver wants a leading `v` and the versions this "+
			"repository reads do not have one, so for a single caller the "+
			"wrapper and the parser cost about the same. At %d it is a "+
			"different trade — two comparators agree until the day they do "+
			"not, however each is built, and x/mod is already in this module's "+
			"graph as an indirect requirement, so the import moves a line in "+
			"go.mod rather than adding a dependency.\n\n"+
			"The kinds above are the shape of the decision, not the decision "+
			"itself. Two hand-rolled is the case the reason was written "+
			"against; one of each is worse, because the two disagree about "+
			"the cases semver has opinions on (build metadata, a bare `v`, a "+
			"pre-release tail) and each looks obviously correct beside its own "+
			"caller.\n\n"+
			"Either put the callers on one comparator — semver answers every "+
			"rule compareVersions has; the four of them are checked off in "+
			"this test's header — or raise dottedVersionComparators and write "+
			"the reason beside that header, so the next person reads a "+
			"decision rather than a number.",
			len(found), dottedVersionComparators, comparatorList(found),
			len(found))
	}

	t.Logf("%d dotted-version comparator(s) in %d Go file(s): %s. %d other "+
		"place(s) read a version without ordering one; see versionReaders. "+
		"Enumerated by %s.", len(found), len(paths), comparatorList(found),
		len(versionReaders), from)
}

// How many dotted-version comparators the written reason covers.
//
// One, and the second is a decision rather than a number to raise — see the
// arm above, and gitWrapperAcceptRules and timingsRecordCopies, which are the
// same shape of constant for the same kind of reason.
//
// # Comparators and not hand-rolled parsers, which is the whole point
//
// This used to be `dottedVersionParsers`, counting only the split-and-parse
// construction. That made the arm blind in exactly one direction — a second
// comparator built on x/mod/semver — and that direction is the one the arm's
// own message asks somebody to walk in. A check that cannot see the change it
// requests is a check that stops being true at the moment it matters.
//
// # The direction this fails in
//
// A second one is somebody writing twenty correct lines, or twenty correct
// lines' worth of semver call, which this arm fails either way. That is the
// same deliberate direction timingsRecordCopies documents: the failure IS the
// prompt, the change being asked for is in THIS repository, and the message
// names what to do rather than reporting a mistake.
const dottedVersionComparators = 1

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
		"being the same shape by " +
		"TestTheQuestionsOnTheSharedRepositoryParseAreTheOnesDecidedOn",
}, {
	file: "mobile/verify/composelayout_test.go",
	what: "a Compose release DERIVED out of the BOM's own pom and used to " +
		"find a jar — read the way gradle reads it, and compared to nothing",
}}

// versionComparator is one function that orders dotted versions: which
// construction it is, what it is called, and where.
type versionComparator struct {
	kind, name, at string
}

// comparatorList is what the walk found, for a message.
func comparatorList(found []versionComparator) string {
	out := make([]string, 0, len(found))
	for _, c := range found {
		out = append(out, fmt.Sprintf("%s, %s (%s)", c.name, c.kind, c.at))
	}
	return strings.Join(out, "; ")
}

// ordersViaSemver is whether this body calls golang.org/x/mod/semver's own
// orderer.
//
// # Which calls count, and why not every semver call does
//
// semver.IsValid, semver.Canonical and semver.MajorMinor answer questions
// about ONE version and order nothing — a caller of those is a version reader,
// which is what versionReaders is a list of, and counting them here would put
// four of this repository's existing lines into a budget of one.
//
// Compare is the order. Sort and Max are it applied to a collection, and both
// are worth catching for the same reason: a function reaching for either is
// deciding which of several versions is ahead, which is precisely the thing
// two comparators can disagree about.
//
// # Which qualifier is semver's, which used to be whichever one said `semver`
//
// The name was matched and the import path was not, on the reasoning that a
// local package aliased to `semver` exporting a `Compare` would be a second
// comparator reported as one — the safe direction. That argument covers the
// false POSITIVE and says nothing about the other end, which is the one that
// matters here: `import sv "golang.org/x/mod/semver"` and a `sv.Compare` is a
// second comparator this arm was written to prompt for, arriving in the
// spelling the arm's own message asks for, and the name test cannot see it.
//
// So `semver` is whatever the calling file binds to golang.org/x/mod/semver,
// which is exact in both directions — a file that does not import it has no
// call into it, whatever its identifiers are called. See importnames_test.go.
func ordersViaSemver(body *ast.BlockStmt, semver map[string]bool) bool {
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
		if !ok {
			return true
		}
		switch sel.Sel.Name {
		case "Compare", "Sort", "Max":
		default:
			return true
		}
		if pkg, ok := sel.X.(*ast.Ident); ok && semver[pkg.Name] {
			hit = true
			return false
		}
		return true
	})
	return hit
}

// splitsOnADot is whether this body splits a string on the literal ".".
//
// The spellings a version parser realistically uses, and no more: a wider net
// here — every strings call with a "." in it — would match path handling all
// over the repository and turn the count above into a list somebody has to
// argue with.
func splitsOnADot(body *ast.BlockStmt, strs map[string]bool) bool {
	return callsStrings(body, strs, map[string]bool{
		"Split": true, "SplitN": true, "SplitSeq": true, "Cut": true,
	}, ".")
}

// parsesANumber is whether this body converts a string to an integer.
func parsesANumber(body *ast.BlockStmt, conv map[string]bool) bool {
	return callsStrconv(body, conv, map[string]bool{
		"Atoi": true, "ParseInt": true, "ParseUint": true,
	})
}

// callsStrings is whether the body calls one of these strings functions with
// `lit` as its second argument.
//
// `strs` is what the calling file binds to the `strings` import, for the
// reason ordersViaSemver takes its own: a qualifier is a fact about one file's
// import block, and reading it there is both cheaper and exact.
func callsStrings(body *ast.BlockStmt, strs, names map[string]bool, lit string) bool {
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
		if !ok || !strs[pkg.Name] {
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
//
// `conv` is the calling file's own binding for the strconv import; see
// callsStrings.
func callsStrconv(body *ast.BlockStmt, conv, names map[string]bool) bool {
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
		if pkg, ok := sel.X.(*ast.Ident); ok && conv[pkg.Name] {
			hit = true
			return false
		}
		return true
	})
	return hit
}
