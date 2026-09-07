package verify

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Shared machinery for reading a native renderer's dispatch on a prop value —
// Swift's `switch mode { case "…": }` and Kotlin's `when (mode) { "…" -> }`.
//
// Both languages write the same construct: a list of string-literal arms
// followed by a catch-all. The two syntaxes differ only in punctuation, so the
// difference is a dispatchSyntax value and everything downstream — locating
// the function, refusing to read past the catch-all, validating the label
// lists, failing loudly on anything unexpected — is written once.
//
// These parse rather than compile, and that is the point rather than a
// compromise. A native compiler cannot answer the question being asked here:
// `default` and `else` make a string switch exhaustive by construction, so
// "you forgot a mode" is not a type error in either language and never will
// be. The only thing that can notice is something holding the arms up against
// Go's list, which means reading them out of the source. Doing it in Go keeps
// the check inside `go test ./...`, where it runs for anyone with a Go
// toolchain and no Xcode, no Android SDK and no memory of a run.sh.
//
// The cost is a constraint on how those functions may be written: one arm per
// line, the string literals first on the line, and the catch-all last. That
// constraint is stated in a comment beside each of them, and every violation
// of it below is a named fatal rather than a short result — a rewrite is meant
// to land as a failure that says what happened, not as an empty comparison
// that agrees with everything.
type dispatchSyntax struct {
	// file is the path to read, relative to this package.
	file string
	// fn is the text that anchors the search to the right function. Anchoring
	// on the function rather than on a bare `switch` is what stops an
	// unrelated dispatch elsewhere in a 600-line renderer from being read as
	// this one.
	fn string
	// open is the switch header, including the name of the value being
	// switched on. Requiring the name is what proves the arms found below are
	// keyed by the prop this test is about and not by something else.
	open string
	// arm matches one labeled arm and captures its label list. Both are
	// written to require the line to begin with a string literal, which is
	// what keeps a comment or a pattern-matching arm from being read as a
	// label.
	//
	// Neither requires the arm's body to be on the following line. Swift's
	// used to, and that turned out to be a constraint on the renderer rather
	// than on the parse: grMobScaled's arms are multi-line statements and read
	// well broken up, but grMobTextAlignment is a four-line expression switch
	// whose arms are a single value each (`case "center": .center`), and
	// forcing those onto two lines apiece to suit a regexp would be the test
	// dictating style. The capture is non-greedy up to the first colon, so a
	// body that itself contains one (`case "a": f(x: "y")`) still yields just
	// the labels — and an arm whose label list is not pure string literals is
	// still a fatal in parseLabelList, which is where that guarantee actually
	// lives.
	arm *regexp.Regexp
	// fallback matches the catch-all arm. Arms are only collected from *above*
	// it: anything below is unreachable, and a test that counted it would
	// report coverage the renderer does not have.
	fallback *regexp.Regexp
	// fallbackDesc names the catch-all in failure messages, in the language's
	// own spelling.
	fallbackDesc string
}

var (
	swiftSwitch = dispatchSyntax{
		arm:          regexp.MustCompile(`(?m)^[ \t]*case[ \t]+("[^\n]*?)[ \t]*:`),
		fallback:     regexp.MustCompile(`(?m)^[ \t]*default[ \t]*:`),
		fallbackDesc: "`default:`",
	}
	kotlinWhen = dispatchSyntax{
		arm:          regexp.MustCompile(`(?m)^[ \t]*("[^\n]*?)[ \t]*->`),
		fallback:     regexp.MustCompile(`(?m)^[ \t]*else[ \t]*->`),
		fallbackDesc: "`else ->`",
	}
)

// with fills in the per-check half of a syntax: which file, which function,
// which switch header. The punctuation halves above are language facts and are
// shared; these three are what a particular check is about.
func (d dispatchSyntax) with(file, fn, open string) dispatchSyntax {
	d.file, d.fn, d.open = file, fn, open
	return d
}

// labels returns the string labels of the described dispatch, in source order,
// having first proved it found the right one.
//
// Failure at every step rather than an empty slice, for the reason the replay
// harness grew a control mutant: a check that reads nothing must not be able to
// read as a pass. Renaming the function, switching on a different value, or
// deleting the catch-all all land here as fatals that name what changed.
func (d dispatchSyntax) labels(t *testing.T) []string {
	t.Helper()

	raw, err := os.ReadFile(d.file)
	if err != nil {
		t.Fatalf("reading %s: %v", d.file, err)
	}
	// Comments out, literals in — the arms ARE string literals, so the
	// stronger mask would delete the subject. The offsets are unchanged (see
	// maskNonCode), so every index below still points where it would have
	// pointed; what changes is that a function named in a doc comment can no
	// longer be the anchor, a commented-out `case "x":` can no longer be
	// counted as an arm, and prose mentioning `default:` can no longer stand in
	// for a catch-all that was deleted.
	src := maskComments(string(raw))

	fn := strings.Index(src, d.fn)
	if fn < 0 {
		t.Fatalf("%s: no %s found — if it was renamed or restructured, update this test", d.file, d.fn)
	}
	body := src[fn:]

	open := strings.Index(body, d.open)
	if open < 0 {
		t.Fatalf("%s: %s does not dispatch on %q — this test cannot tell which arms are the ones it "+
			"is about", d.file, d.fn, d.open)
	}
	// The header search runs forward from the anchor, so on its own it will
	// happily leave the anchored function and find a lookalike further down.
	// Both files are full of lookalikes by construction: GrMobFlex.swift has
	// two `switch justify {` and Renderer.kt has two identical
	// `when (s?.alignItems?.ifEmpty { s.align }) {`, and the anchor is the only
	// thing telling each pair apart. Changing `leading` to switch on something
	// other than `justify` therefore used to pass — the test silently read
	// `gap`'s arms instead, which cover the same six values, and reported
	// coverage `leading` no longer had.
	//
	// So an intervening declaration is a fatal. This is the header-search
	// counterpart of the matchingBrace bound below, which fixed the same defect
	// one level down (arms collected out of the *next* switch); the two
	// together are what make "this dispatch" mean the anchored one.
	if at := declStart.FindStringIndex(body[len(d.fn):open]); at != nil {
		t.Fatalf("%s: the first %q after %s lies past the start of a later declaration (%q), so it "+
			"is not %s's — either it was changed to dispatch on something else, or the anchor no "+
			"longer names the function that holds it",
			d.file, d.open, d.fn, strings.TrimSpace(body[len(d.fn)+at[0]:len(d.fn)+at[1]]), d.fn)
	}
	// Bounded to the dispatch's own braces before anything is read out of it.
	// Both renderers are ~600 lines with several string switches in them — one
	// on text alignment, one on justify-content, both with a "center" arm — so
	// a scan that simply ran forward from here would happily collect arms out
	// of the next switch down, and would find *its* catch-all when this one had
	// been deleted. That is not a hypothetical: it is what the first version of
	// this test did, and the mutation that deleted grMobScaled's `default` arm
	// went uncaught because of it.
	body = matchingBrace(t, d.file, d.fn+"'s dispatch block", body[open+len(d.open)-1:])

	// Cutting at the catch-all before collecting arms does two jobs: it proves
	// the catch-all still exists (the contract that an absent or unrecognized
	// value has a defined rendering), and it keeps unreachable arms below it
	// from counting as coverage.
	stop := d.fallback.FindStringIndex(body)
	if stop == nil {
		t.Fatalf("%s: %s has no %s arm — every native renderer is required to define what an absent "+
			"or unrecognized value does", d.file, d.fn, d.fallbackDesc)
	}

	var labels []string
	for _, m := range d.arm.FindAllStringSubmatch(body[:stop[0]], -1) {
		labels = append(labels, d.parseLabelList(t, m[1])...)
	}
	if len(labels) == 0 {
		t.Fatalf("%s: %s's arms parsed as empty, which is not a pass — the dispatch was found but no "+
			"labels were read out of it", d.file, d.fn)
	}
	return labels
}

// declSource returns the source of the declaration that begins at the first
// occurrence of anchor, cut at the next function declaration (or the end of
// the file), with the comments blanked out.
//
// labels() bounds a dispatch by its braces; this coarser cut exists for the
// checks that read something other than a switch. One of the declarations
// they read is expression-bodied (Renderer.kt's isColumnStretch), which has
// no block to bound — its body is the rest of its own line — so the next
// declaration is the boundary that works for every shape.
//
// # Why the prose is taken out, for every caller
//
// The cost of the coarseness is that the NEXT declaration's doc comment rides
// along with the region, and the comment above this one used to argue that the
// cost was affordable: the substrings held against these regions are
// expression fragments (`ifEmpty { s.align }`), which prose does not
// accidentally spell.
//
// That argument was wrong, and a break-test found it. Renderer.kt's
// Modifier.pinMainAxis sits below RowChildren and its doc comment explains
// that the renderer reads `shrinkPinned`; deleting the call and keeping the
// comment passed a `strings.Contains(body, "shrinkPinned")` that existed to
// find the call. The fragments are not all expressions — several are plain
// identifiers — and a doc comment's whole job is to name the thing below it,
// so the cases where prose spells the substring are exactly the cases where
// somebody wrote a careful comment.
//
// So the mask is applied here rather than at three call sites that remembered
// to ask for it. Offsets are preserved (see maskNonCode), the literals are
// left alone so a check reading a string literal still finds it, and the
// masking happens BEFORE the anchor search: an anchor that survives only in a
// comment now fails to be found, which is the swiftDeclIndices rule reaching
// the coarse cut too.
func declSource(t *testing.T, file, anchor string) string {
	t.Helper()

	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("reading %s: %v", file, err)
	}
	src, ok := declSourceOf(string(raw), anchor)
	if !ok {
		t.Fatalf("%s: no %s found in code — if it was renamed or restructured, "+
			"update this test.\n\n"+
			"A mention of it in a comment does not count: the comments are blanked "+
			"before this looks, because the cut runs to the next declaration and "+
			"would otherwise hand every check below a paragraph of English to "+
			"search.", file, anchor)
	}
	return src
}

// declSourceOf is the decision, as a function of two strings.
//
// Split out for the reason swiftDeclIndices was: the interesting inputs — an
// anchor that appears only in prose, a doc comment below the declaration that
// spells what the code is supposed to do — are otherwise reachable only by
// owning a renderer with the fault in it, and the renderers are written the
// way the old substring assumed.
func declSourceOf(src, anchor string) (string, bool) {
	code := maskComments(src)

	at := strings.Index(code, anchor)
	if at < 0 {
		return "", false
	}
	rest := code[at+len(anchor):]
	if next := declStart.FindStringIndex(rest); next != nil {
		rest = rest[:next[0]]
	}
	return anchor + rest, true
}

// The prose a coarse cut carries must not answer a question about code.
//
// This is the defect that put the mask in declSource, written out as the
// synthetic sources that reach it. The renderers are of course written the way
// the old substring assumed — that is exactly why the assumption survived — so
// each case here is a shape one of them already has, with the call removed.
//
//	the carried doc comment   declSource runs to the NEXT declaration, so the
//	                          comment introducing that declaration is inside
//	                          the region. Renderer.kt's pinMainAxis documents
//	                          the `shrinkPinned` read that RowChildren makes,
//	                          and deleting the read passed.
//
//	the anchor in prose       the other end of the same cut: an anchor found
//	                          in a comment starts the region in the middle of
//	                          a paragraph, and everything below runs against
//	                          English. swiftDeclIndices refuses this for Swift
//	                          types; this is the coarse cut's version.
//
// And one case in the other direction, because a mask that took too much would
// fail just as quietly: the string literals must survive, since several checks
// in this package read a value out of one.
func TestDeclSourceDoesNotHandBackTheProseAroundIt(t *testing.T) {
	const anchor = "fun RowChildren("

	for _, c := range []struct {
		name, src string
		// want is a substring the region must contain, unwanted one it must
		// not; each case states the one it is about.
		want, unwanted string
		// missing says the anchor must not be found at all.
		missing bool
	}{
		{
			name: "the next declaration's doc comment rides along",
			src: "fun RowChildren() {\n    render(child)\n}\n\n" +
				"/** Applied by RowChildren when shrinkPinned is set. */\n" +
				"fun pinMainAxis() {}\n",
			unwanted: "shrinkPinned",
		},
		{
			name: "a line comment inside the declaration",
			src: "fun RowChildren() {\n" +
				"    // shrinkPinned used to be read here\n" +
				"    render(child)\n}\n",
			unwanted: "shrinkPinned",
		},
		{
			name: "a KDoc above the anchor mentioning it",
			src: "/**\n * The loop below; see fun RowChildren( for the pin.\n */\n" +
				"fun RowChildren() {\n    render(shrinkPinned)\n}\n",
			want: "shrinkPinned",
		},
		{
			// Nothing but a mention: there is no declaration to cut, and the
			// answer has to be "not found" rather than a region starting in
			// the middle of a sentence.
			name:    "only a mention, and no declaration",
			src:     "// fun RowChildren( is gone; see the changelog.\nfun Other() {}\n",
			missing: true,
		},
		{
			// The mask must stop here. Renderer.kt's dispatches are read for
			// their arms, which are string literals, and a mask that blanked
			// them would turn every one of those checks into a parse that
			// finds nothing.
			name: "a string literal is code and stays",
			src:  "fun RowChildren() {\n    when (s.align) { \"center\" -> pad() }\n}\n",
			want: `"center"`,
		},
	} {
		got, ok := declSourceOf(c.src, anchor)
		if c.missing {
			if ok {
				t.Errorf("%s: declSourceOf found the anchor and returned %q. A cut that "+
					"starts inside a comment hands every check below it a paragraph of "+
					"English, which reads as a pass for anything the paragraph happens "+
					"to say.", c.name, got)
			}
			continue
		}
		if !ok {
			t.Errorf("%s: declSourceOf did not find %q at all", c.name, anchor)
			continue
		}
		if c.unwanted != "" && strings.Contains(got, c.unwanted) {
			t.Errorf("%s: the region still contains %q, which is only in the prose. "+
				"A strings.Contains meant to find the call is satisfied by the comment "+
				"explaining it, so deleting the call passes.\n%s", c.name, c.unwanted, got)
		}
		if c.want != "" && !strings.Contains(got, c.want) {
			t.Errorf("%s: the region no longer contains %q, which is code. The mask has "+
				"taken more than the prose, and every check reading a value out of a "+
				"literal now finds nothing.\n%s", c.name, c.want, got)
		}
	}
}

// nativeFile locates a renderer source relative to this package. The natives
// live outside the Go module tree, so every check here reaches up two levels
// and back down; spelling that once keeps the paths from drifting apart.
func nativeFile(parts ...string) string {
	return filepath.Join(append([]string{"..", ".."}, parts...)...)
}

// The files every check in this package reads.
var (
	swiftRenderer = nativeFile("ios", "GrMob", "Runtime", "Renderer.swift")
	swiftFlex     = nativeFile("ios", "GrMob", "Runtime", "GrMobFlex.swift")
	// The overlay's arithmetic and its placement vocabulary. A sibling of
	// GrMobFlex.swift in every sense: pure, CoreGraphics-only, split out of a
	// SwiftUI Layout so ios/verify can measure it rather than type-check it.
	swiftStack     = nativeFile("ios", "GrMob", "Runtime", "GrMobStack.swift")
	kotlinRenderer = nativeFile("android", "app", "src", "main", "java", "com", "grmob",
		"runtime", "Renderer.kt")
)

// stringArray reads the string literals out of a single-line array literal
// that follows an anchor, for the one copy of a core list in either renderer
// that is not a switch: GrMobFlexSolver.justifyClaimsFreeSpace, which asks
// whether a justify-content value spends the container's leftover space and
// spells the answer as membership in a five-element array.
//
// It is deliberately narrow — one line, one bracket pair, string literals and
// commas and nothing else — because a general Swift expression parser is not
// what this needs and would be far more code than the thing it checks. The
// narrowness is stated beside the array itself, and every way of violating it
// is a named fatal here rather than a short list, for the reason the rest of
// this file fails loudly: a check that reads nothing must not read as a pass.
func stringArray(t *testing.T, file, anchor string) []string {
	t.Helper()

	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("reading %s: %v", file, err)
	}
	src := string(raw)

	at := strings.Index(src, anchor)
	if at < 0 {
		t.Fatalf("%s: no %s found — if it was renamed or restructured, update this test", file, anchor)
	}
	open := strings.Index(src[at:], "[")
	if open < 0 {
		t.Fatalf("%s: no array literal after %s", file, anchor)
	}
	open += at
	close := strings.Index(src[open:], "]")
	if close < 0 {
		t.Fatalf("%s: the array literal after %s is unterminated", file, anchor)
	}
	close += open

	body := src[open+1 : close]
	if strings.Contains(body, "\n") {
		t.Fatalf("%s: the array literal after %s spans more than one line; this parse reads one",
			file, anchor)
	}

	var out []string
	rest := stringLiteral.ReplaceAllStringFunc(body, func(lit string) string {
		out = append(out, strings.Trim(lit, `"`))
		return ""
	})
	if leftover := strings.Trim(rest, " \t,"); leftover != "" {
		t.Fatalf("%s: the array after %s holds something this test cannot read: %q (unhandled: %q) — "+
			"it is required to be string literals so its coverage can be checked", file, anchor, body, leftover)
	}
	if len(out) == 0 {
		t.Fatalf("%s: the array after %s parsed as empty, which is not a pass", file, anchor)
	}
	return out
}

// parseLabelList reads the string literals out of one arm's label list,
// covering the multi-label forms both languages allow (`case "a", "b":`,
// `"a", "b" ->`).
//
// The list is validated rather than merely scraped: after the literals and
// their separators are removed, anything left means the arm says something
// this parse does not understand, and the honest response is to stop. Scraping
// alone would silently truncate such an arm to whichever labels it happened to
// recognize, and a truncated arm reads as missing coverage — a confusing
// failure — or, worse, as coverage that is not there.
func (d dispatchSyntax) parseLabelList(t *testing.T, list string) []string {
	t.Helper()

	var out []string
	rest := stringLiteral.ReplaceAllStringFunc(list, func(lit string) string {
		out = append(out, strings.Trim(lit, `"`))
		return ""
	})
	if leftover := strings.Trim(rest, " \t,"); leftover != "" {
		t.Fatalf("%s: %s has an arm this test cannot read: %q (unhandled: %q) — arms are required to "+
			"be string literals so their coverage can be checked", d.file, d.fn, list, leftover)
	}
	return out
}

var stringLiteral = regexp.MustCompile(`"[^"]*"`)

// declStart matches the beginning of a function declaration in either
// language: any run of modifier words (`private`, `static`, `@Composable`)
// followed by `func` or `fun` and a name, at the start of a line.
//
// Deliberately only functions. `val` and `var` lines are everywhere inside
// these bodies — GrMobRow opens with `val s = animatedStyle(node.style)` — and
// treating them as boundaries would reject every dispatch that is not the
// first statement of its function. A function declaration is the boundary that
// matters here, because it is what the header search can wrongly run past.
var declStart = regexp.MustCompile(`(?m)^[ \t]*(?:[\w@]+[ \t]+)*(?:func|fun)[ \t]+\w`)

// matchingBrace returns the contents of the block that src opens with, from
// just after the opening brace to just before the brace that closes it.
//
// Brace counting has to skip comments and string literals or it counts the
// wrong braces — a `{` inside an arm's explanatory comment would end the block
// early, and both renderers do contain string literals with braces in them
// elsewhere. Swift and Kotlin spell all three constructs the same way, so one
// scanner serves both. Neither language's extras matter here: Swift's `\(…)`
// interpolation nests parentheses rather than braces, and Kotlin's `${…}` is
// balanced, so it counts a `{` and its `}` and comes out even.
//
// # Why the skipping is not here
//
// It used to be: four arms, in the same order and with the same words as
// maskNonCode's, one directory over — because "skip comments and literals" and
// "blank comments and literals" are one question asked twice. Each file carried
// a comment asking the two to agree, and a comment is not a mechanism. The
// failure of a drifted copy is the quiet one: a counter that mishandled `"""`
// ends the block early, still returns a string, and every `strings.Contains`
// below it goes on running against half a declaration.
//
// So the count runs over the MASK. maskNonCode blanks every comment and literal
// character to a space and preserves length, so offsets in the mask are offsets
// in the source, and a brace that survives it is a brace in code. What is left
// here is the counting, which is the part that was never shared, and the result
// is sliced out of the original source — the callers below read literals, and a
// body handed back blanked would answer "does it LIST this value" with nothing.
//
// file and what name the source and the construct, and are used only to say
// where an unterminated block was found. They were a dispatchSyntax when the
// dispatch parser was the only caller; swiftTypeBody is the second, and a type
// declaration is not a dispatch, so the two strings it actually needed are
// what it takes now.
func matchingBrace(t *testing.T, file, what, src string) string {
	t.Helper()

	// literals blanked, because a brace inside one is not a brace.
	code, unterminated := maskNonCode(src, true)
	if unterminated != "" {
		t.Fatalf("%s: unterminated %s inside %s", file, unterminated, what)
	}

	depth := 0
	for i := 0; i < len(code); i++ {
		switch code[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return src[1:i]
			}
		}
	}
	t.Fatalf("%s: %s is unterminated — its opening brace has no match", file, what)
	return ""
}
