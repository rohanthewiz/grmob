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

// proseSourceOf cuts one declaration TOGETHER WITH the note above it, prose
// intact.
//
// # Why this is a different cut and not a flag on the one above
//
// declSourceOf blanks the comments and then cuts from the declaration line
// down, which is right for every question about code and wrong for every
// question about a note: the note a declaration carries is written ABOVE it,
// and that cut starts below. So a check whose subject is "grMobSelectedTrait
// explains that SwiftUI has no word for the off state" had one reader available
// — the whole file — and asked its question of a file rather than of a
// declaration. Two sites did exactly that, each with a comment saying why.
//
// # Where it starts and where it stops
//
// Both ends are the comment block, and they are the two halves of one rule: a
// declaration's note is the run of comment lines immediately above it, up to
// the first blank line or the first line with code on it.
//
//	the start   walk back from the anchor's line over that run, so the
//	            paragraph the check is about is inside the region
//	the end     the next declaration, minus ITS run — because the coarse cut's
//	            known defect is that it carries the next declaration's doc
//	            comment along, and for a reader that keeps comments that defect
//	            is the whole failure mode rather than a wrinkle
//
// Offsets come off the mask and the text comes off the source, which is what
// makes both walks cheap: maskComments blanks a comment without moving
// anything, so "this line is comment or blank" is "this line is blank in the
// mask", and the same index cuts the raw file.
//
// # The blank line that is not a separator
//
// "Up to the first blank line" is a rule about the SOURCE, and it was applied
// to a file whose comments the mask had already flattened. Those are two
// different questions wherever one comment construct spans a blank line, and
// exactly one construct here does: `/* … */`. A Kotlin KDoc or a Swift block
// comment with an empty line between its paragraphs is a single thing the
// scanner blanks whole, and the walk read that line as the end of the note —
// so the region began in the middle of a comment, with the top half of the
// paragraph outside it and nothing saying so.
//
// The direction is the silent one. The check that follows is a
// strings.Contains over the region, and a region missing its first paragraph
// still returns a string: a phrase that moved into the half that was cut off
// reads as a phrase that was deleted, and a note rewritten around a blank line
// fails a check about wording that did not change.
//
// So the walk asks the scanner where the comments were, rather than inferring
// it from blankness. A line inside a comment span continues the note whatever
// it looks like; a blank line outside one ends it, which is the rule the
// paragraph above states and now the rule this implements.
func proseSourceOf(src, anchor string) (string, bool) {
	// maskNonCode directly rather than maskComments, which is the same call with
	// the same argument: the two named spellings return the mask alone, and the
	// spans are the whole reason this walk is correct. Reaching past the name
	// is worth a line of explanation and not a third spelling — "prose out,
	// literals in" is still exactly what `false` asks for.
	code, _, comments := maskNonCode(src, false)

	at := strings.Index(code, anchor)
	if at < 0 {
		return "", false
	}

	// inComment reports whether an offset falls inside one of the scanner's
	// comments. Linear over a handful of spans per call and called once per
	// line walked, which is a few dozen times for the region this cuts — the
	// alternative is a sorted search over a slice that is usually shorter than
	// the search's own setup.
	inComment := func(i int) bool {
		for _, c := range comments {
			if i >= c.from && i < c.to {
				return true
			}
		}
		return false
	}

	// commentBlockStart walks back from the beginning of the line at `from`
	// over the contiguous run of comment-only lines. A blank line ends the run:
	// it is what separates one declaration's note from the paragraph above it
	// — unless the blank line is INSIDE a comment, in which case it separates
	// two paragraphs of one note and the run continues through it.
	commentBlockStart := func(from int) int {
		for from > 0 {
			prev := lineStart(src, from-1)
			line := src[prev : from-1]
			if strings.TrimSpace(line) == "" && !inComment(prev) {
				break // a blank line: the note starts below it
			}
			if strings.TrimSpace(code[prev:from-1]) != "" {
				break // code on the line: not part of the note
			}
			from = prev
		}
		return from
	}

	start := commentBlockStart(lineStart(src, at))

	end := len(src)
	if next := declStart.FindStringIndex(code[at+len(anchor):]); next != nil {
		end = commentBlockStart(at + len(anchor) + next[0])
	}
	return src[start:end], true
}

// lineStart is the index of the first character of the line containing i.
func lineStart(s string, i int) int { return strings.LastIndexByte(s[:i], '\n') + 1 }

// And the prose cut, at both of its ends.
//
// proseOf is the one reader that KEEPS the comments, so where its region starts
// and stops is the whole of what it means. The two ends are one rule read in
// two directions — a declaration's note is the run of comment lines immediately
// above it — and each end fails in its own way:
//
//	the start   a cut beginning at the declaration line misses the note
//	            entirely, which is the state the two converted sites were in:
//	            they read the whole file because the only cut available started
//	            below their subject.
//	the end     a cut running to the next declaration carries THAT
//	            declaration's note, which is declSource's known coarseness. For
//	            a reader that blanks comments it is a wrinkle; for this one it
//	            is the failure mode, because the thing being searched for is
//	            prose and the carried paragraph is prose about something else.
//
// Synthetic sources for the same reason the table above uses them: the
// renderers are written the way a correct cut expects, so the interesting
// inputs are otherwise reachable only by breaking one.
func TestProseOfCutsTheNoteThatBelongsToTheDeclaration(t *testing.T) {
	const anchor = "func grMobSelectedTrait("

	for _, c := range []struct {
		name, src string
		// want is a phrase the region must contain, unwanted one it must not.
		want, unwanted string
		missing        bool
	}{
		{
			// The reason the reader exists. The note is above the declaration
			// and declSource would start below it.
			name: "the declaration's own note is inside the region",
			src: "/// SwiftUI has no word for the off state.\n" +
				"private func grMobSelectedTrait(_ s: String) -> Traits {\n    []\n}\n",
			want: "no word for the off state",
		},
		{
			// The other end, and the one that makes this more than a
			// convenience: a note about the NEXT declaration answering a
			// question about this one is exactly the failure a whole-file read
			// already had, moved five lines.
			//
			// The phrase is the neighbour's OWN wording rather than a
			// restatement of this declaration's. It was the latter, and that
			// made the row unfalsifiable: the fixture read "Nothing here has a
			// word for the off state either" and the row refused "no word for
			// the off state", which is not a substring of it — so the
			// assertion held whatever the cut did, including when the cut
			// carried the whole neighbour along.
			name: "the next declaration's note stays out",
			src: "private func grMobSelectedTrait(_ s: String) -> Traits {\n    []\n}\n\n" +
				"/// The neighbour's own note, about the off state.\n" +
				"private func grMobOther() {}\n",
			unwanted: "neighbour's own note",
		},
		{
			// A blank line ends the note. Without that rule the walk back would
			// run to the top of the file and swallow whatever precedes it,
			// which for a renderer is the previous declaration's paragraph.
			name: "a paragraph separated by a blank line is not this note",
			src: "/// Some other function's note about the off state.\n" +
				"\n" +
				"/// Go's core.SelectedState as a trait.\n" +
				"private func grMobSelectedTrait(_ s: String) -> Traits {\n    []\n}\n",
			unwanted: "other function's note",
		},
		{
			// And a line with code on it ends it too, so the declaration above
			// does not come along with its trailing comment.
			name: "the previous declaration does not come along",
			src: "private func grMobRole() { off() }\n" +
				"/// Go's core.SelectedState as a trait.\n" +
				"private func grMobSelectedTrait(_ s: String) -> Traits {\n    []\n}\n",
			unwanted: "grMobRole",
		},
		{
			// One comment, with a blank line in the middle of it. This is the
			// case the old rule got backwards: the walk stopped at the blank
			// line, so the region started four characters into a construct the
			// scanner treats as one — the top paragraph outside it, and a
			// strings.Contains below still returning a string.
			name: "a doc comment with a blank line inside it",
			src: "/**\n * Go's core.SelectedState as a trait.\n\n" +
				" * SwiftUI has no word for the off state.\n */\n" +
				"private func grMobSelectedTrait(_ s: String) -> Traits {\n    []\n}\n",
			want: "core.SelectedState as a trait",
		},
		{
			// And the same construct at the OTHER end. The end walk is the same
			// function, so a next declaration whose note holds a blank line
			// would have had its bottom half carried into this region — the
			// coarse-cut failure the end walk exists to prevent, arriving
			// through the half of the note the walk could not see.
			name: "the next declaration's note stays out, blank line and all",
			src: "private func grMobSelectedTrait(_ s: String) -> Traits {\n    []\n}\n\n" +
				"/**\n * The neighbour's own note, about the off state.\n\n" +
				" * A second paragraph, so the blank line is inside the note.\n */\n" +
				"private func grMobOther() {}\n",
			unwanted: "neighbour's own note",
		},
		{
			// The rule the one above must not have swallowed. A blank line
			// between two SEPARATE comments is outside both of them, so it goes
			// on ending the note — which is what keeps "this declaration's
			// note" from meaning "every comment above it".
			name: "a blank line between two block comments still ends the note",
			src: "/* Some other function's note about the off state. */\n" +
				"\n" +
				"/* Go's core.SelectedState as a trait. */\n" +
				"private func grMobSelectedTrait(_ s: String) -> Traits {\n    []\n}\n",
			unwanted: "other function's note",
		},
		{
			// The anchor is looked for in the MASK even though the raw source
			// is what comes back. Otherwise a note naming the declaration would
			// locate the region the note is then read out of, which is a check
			// that finds its own subject.
			name:    "an anchor that appears only in prose",
			src:     "/// See func grMobSelectedTrait( in the other file.\nfunc other() {}\n",
			missing: true,
		},
	} {
		got, ok := proseSourceOf(c.src, anchor)
		if c.missing {
			if ok {
				t.Errorf("%s: proseSourceOf found the anchor and returned %q. The anchor "+
					"is searched for in the masked source precisely so a mention of it "+
					"in a comment cannot start a region.", c.name, got)
			}
			continue
		}
		if !ok {
			t.Errorf("%s: proseSourceOf did not find %q at all", c.name, anchor)
			continue
		}
		if c.want != "" && !strings.Contains(got, c.want) {
			t.Errorf("%s: the region does not contain %q, so a check about this "+
				"declaration's note cannot find it and has to read the whole file "+
				"again.\n%s", c.name, c.want, got)
		}
		if c.unwanted != "" && strings.Contains(got, c.unwanted) {
			t.Errorf("%s: the region contains %q, which belongs to another "+
				"declaration. A note answering a question about its neighbour is the "+
				"failure a whole-file read already had.\n%s", c.name, c.unwanted, got)
		}
	}
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
	code, unterminated, _ := maskNonCode(src, true)
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
