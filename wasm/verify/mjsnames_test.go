package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// A constant named in this directory's prose is a constant this directory has.
//
// # What this is about
//
// The ink scan's skip told a reader who could not run it what to do instead:
// re-measure INK_ROWS, INK_EDGE_CLEARANCE and INK_ROW_GLYPH_FLOOR against the
// face this machine resolved. Two of those three exist. The third never has —
// not renamed, not moved, never declared — and it was printed on every Linux
// run for a session, in the one message whose whole job is to tell somebody
// what to go and do. A reader who followed it had nothing to grep for.
//
// That is the same defect prosenames_test.go was written for — its rule reaches
// this package as the "tests named in prose" arm of
// TestTheQuestionsOnTheSharedRepositoryParseAreTheOnesDecidedOn — one namespace
// over, and it arrived the same way: the sentence was true when it was written
// about a plan and was never true about the code. That rule caught the first
// draft of THIS comment, which cited its own name wrongly — a guess at what
// prosenames_test.go's rule is called, written without looking. The name is not
// repeated here, because the rule reads text and cannot tell a reference from a
// counter-example: a comment that quoted the wrong name in order to say it was
// wrong would fail it forever.
//
// # Why a SCREAMING_SNAKE name is the recognisable case
//
// The general rule — an identifier in prose resolves to something declared —
// cannot be written for JavaScript prose any more than it could for Go's:
// English is full of identifier-shaped words, and this file's comments are full
// of CSS properties, DevTools methods and platform types. What can be
// recognised without resolving anything is the shape this directory uses for
// its own module constants and uses for nothing else: capitals with at least
// one underscore. `OK`, `SKIP`, `PNG`, `IDAT` and every shouted word in a
// comment — ONE, NOT, THESE — have no underscore and are not in the population.
//
// # What was measured before this was written
//
// Over wasm/verify's twenty .mjs files and the runtime they exercise:
//
//	108 declarations       of that shape
//	16 unresolved          against browser.mjs's own declarations alone
//	7 unresolved           once the sibling .mjs files and grmob-runtime.js are
//	                       in the namespace, which is where six of those nine
//	                       were declared
//	2 unresolved           once `process.env.X` counts as a declaration of X
//	                       (GRMOB_TRANSCRIPT and GRMOB_CHROME are the harness's
//	                       two environment variables, named in prose about the
//	                       harness)
//	2 remaining            PERMISSION_DENIED and POSITION_UNAVAILABLE, which are
//	                       GeolocationPositionError's, named in a comment about
//	                       what the platform's numbers mean
//
// So the rule fires on two classes it should not, both of them "a real name,
// belonging to something other than this code", and both are the same platform
// object. They are exempted by name below rather than by a pattern, because a
// pattern for "a web API constant" is a pattern for most of this population.
//
// And it fires on two it should. INK_ROW_GLYPH_FLOOR is the one this was
// written for; INK_CALIBRATIONS was introduced by the commit that wrote the
// instrument and caught on the first run of this test — a constant named three
// times in the prose describing where a calibration gets recorded, where the
// thing that records it is called INK_CALIBRATED_ON.
func TestTheConstantsNamedInProseResolve(t *testing.T) {
	// The runtime is in the namespace because these tests are ABOUT it: a
	// comment in keynav_test.mjs naming FOCUSABLE_TAGS is naming the runtime's
	// set, and a rule that could not see it would report six mentions that take
	// a reader exactly where they are going.
	paths, err := filepath.Glob("*.mjs")
	if err != nil {
		t.Fatalf("globbing this directory's modules: %v", err)
	}
	sort.Strings(paths)
	paths = append(paths, filepath.Join("..", "grmob-runtime.js"))

	// Capitals with at least one underscore. The underscore is the whole of
	// what separates this population from shouted English and from three-letter
	// acronyms — see the note above.
	name := regexp.MustCompile(`\b[A-Z][A-Z0-9]*(?:_[A-Z0-9]+)+\b`)
	// A declaration is any declarator of a const/let/var statement, so
	// `const VOID_W = 120, VOID_H = 40;` declares both — the first parser here
	// saw only the first of that pair and reported VOID_H eleven times.
	declLine := regexp.MustCompile(`^\s*(?:export\s+)?(?:const|let|var)\s`)
	declarator := regexp.MustCompile(`\b([A-Z][A-Z0-9]*(?:_[A-Z0-9]+)+)\s*=[^=]`)
	fnDecl := regexp.MustCompile(`function\s+([A-Z][A-Z0-9]*(?:_[A-Z0-9]+)+)`)
	// An environment variable is declared by being read: process.env.X is the
	// only place a harness variable can be said to exist, and the prose that
	// names one is prose about the harness.
	envRead := regexp.MustCompile(`process\.env\.([A-Z][A-Z0-9]*(?:_[A-Z0-9]+)+)`)

	type mention struct{ where, name string }
	declared := map[string]bool{}
	var sources []struct {
		path  string
		lines []string
	}
	for _, p := range paths {
		src, readErr := os.ReadFile(p)
		if readErr != nil {
			t.Fatalf("%s is part of the namespace this rule resolves against "+
				"and cannot be read: %v", p, readErr)
		}
		lines := strings.Split(string(src), "\n")
		sources = append(sources, struct {
			path  string
			lines []string
		}{p, lines})
		for _, line := range lines {
			if declLine.MatchString(line) {
				for _, m := range declarator.FindAllStringSubmatch(line+" ", -1) {
					declared[m[1]] = true
				}
			}
			for _, m := range fnDecl.FindAllStringSubmatch(line, -1) {
				declared[m[1]] = true
			}
			for _, m := range envRead.FindAllStringSubmatch(line, -1) {
				declared[m[1]] = true
			}
		}
	}
	if len(declared) < 50 {
		t.Fatalf("this rule found %d constants of its own shape in %d files, and "+
			"browser.mjs alone declares more than fifty. The parser is reading "+
			"something other than this directory's declarations, and a namespace "+
			"that small would report most of the prose in it.", len(declared), len(paths))
	}

	// The platform's own names, which are real and are not this code's. Listed
	// one by one with the object they belong to: a pattern that admitted "web
	// API constant" would admit most of this population, and the point of the
	// rule is that a name in prose is checkable.
	platform := map[string]string{
		"PERMISSION_DENIED":    "GeolocationPositionError.PERMISSION_DENIED",
		"POSITION_UNAVAILABLE": "GeolocationPositionError.POSITION_UNAVAILABLE",
		"TIMEOUT":              "GeolocationPositionError.TIMEOUT",
	}

	var unresolved []mention
	for _, s := range sources {
		for i, line := range s.lines {
			for _, n := range name.FindAllString(line, -1) {
				if declared[n] || platform[n] != "" {
					continue
				}
				unresolved = append(unresolved, mention{
					where: fmt.Sprintf("%s:%d", s.path, i+1), name: n,
				})
			}
		}
	}
	if len(unresolved) == 0 {
		return
	}
	var said []string
	for _, m := range unresolved {
		said = append(said, fmt.Sprintf("%s names %s", m.where, m.name))
	}
	t.Errorf("%d mention(s) of a constant this directory does not declare:\n  %s\n\n"+
		"A name in a comment is text, and the only things in this repository that "+
		"read comments as text are rules like this one. The mention that prompted "+
		"it was INK_ROW_GLYPH_FLOOR, printed in the ink scan's SKIP — the one "+
		"message whose entire job is to tell a reader on a machine that cannot run "+
		"the check what to go and do — naming a constant that has never existed "+
		"under any name.\n\n"+
		"Three ways a mention gets here, and they want different things: the "+
		"constant was renamed (fix the sentence), it was never there (fix the "+
		"sentence, and ask what the sentence was supposed to be about), or it "+
		"belongs to the platform rather than to this code (add it to `platform` "+
		"above, with the object it is a member of).",
		len(unresolved), strings.Join(said, "\n  "))
}
