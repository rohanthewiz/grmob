package verify

import (
	"os"
	"strings"
	"testing"
)

// core.FlexShrink(0) must reach a layout decision on both natives.
//
// # What this is the other half of
//
// wasm/verify's shrink_test.go pins the NUMBER: core.ShrinkNone is -1, and all
// four spellings outside Go — two JavaScript, one Swift, one Kotlin — must read
// that same number. A parser that reads the sentinel correctly and hands the
// answer to nothing passes every one of those checks, which is exactly the
// state both DOM targets were in before the sentinel existed: the prop
// compiled, applied, serialised, and did nothing.
//
// So this file pins the call site. It is the same division of labour the gap
// longhands have one file over (TestBothNativeParsersReadTheGapLonghands reads
// the parser, TestNativeContainersSpaceAlongTheirOwnAxis reads the container),
// and it is here rather than in wasm/verify because a call site is native
// source, which is this package's whole subject.
//
// # The two targets do different things with it, and both are the contract
//
//	SwiftUI   GrMobFlexSolver takes a per-item shrink factor and implements
//	          CSS's scaled-base rule over it, so every factor means what CSS
//	          says it means. The renderer's job is to hand the factor over.
//
//	Compose   a Row has no proportional shrink at all — it measures each
//	          unweighted child against the main-axis space the ones before it
//	          did not take — so a fractional factor has nothing to map onto.
//	          Zero is not a proportion but a refusal, and a refusal IS
//	          expressible: measure the child unbounded and report its own
//	          size. That is Modifier.pinMainAxis, and it is what makes
//	          core.FlexShrink(0) mean the same thing on the fourth target as
//	          on the other three.
//
// # And what the call-site pins are the other end of
//
// A call site is not an arithmetic. internal/pinfixture transcribes
// foundation-layout's zero-weight measure loop and the two lines below, runs
// one overflowing Row through it with the pin in each position, and
// ios/verify/pin.swift solves the same Row through GrMobFlexSolver — so the
// numbers core.FlexShrink(0) produces are now compared across the two targets
// rather than asserted about either.
//
// Which makes THESE checks the link that transcription hangs from. Nothing in
// this repository can run androidx's measure policy, so what stands between the
// fixture and the renderer is the pair of pins below: the renderer applies the
// modifier on the right axis from both loops, and the modifier measures
// unbounded and reports what it measured. Change either and the fixture goes on
// producing the same numbers about code that no longer exists.

// The SwiftUI renderer must hand the solver the reading, not the raw field.
//
// `shrinkFactor` and `flexShrink` are one character apart at a call site and
// mean opposite things for the two values that matter: an unset field is 0 and
// must become a factor of 1, and core.ShrinkNone is -1 and must become 0. A
// renderer that passed the field straight through would pin every node in
// every layout (0 read as a factor) and let the one node that asked to be
// pinned shrink harder than its siblings (-1 read as a factor), and both
// layouts would still be produced by code that compiles.
func TestTheSwiftUIRowHandsTheSolverTheShrinkReading(t *testing.T) {
	body := codeOf(t, swiftRenderer, "private struct FlexChildren: View {")
	if !strings.Contains(body, "GrMobFlexShrink.self") {
		t.Errorf("%s: FlexChildren no longer attaches a GrMobFlexShrink layout value. "+
			"GrMobFlexSolver's shrink arm takes a per-item factor and defaults it to 1; "+
			"with nothing attached, core.FlexShrink is inert on this target and the "+
			"only thing saying otherwise is core.ShrinkNone's doc comment.", swiftRenderer)
	}
	if !strings.Contains(body, "child.style?.shrinkFactor") {
		t.Errorf("%s: FlexChildren does not read child.style?.shrinkFactor. The raw "+
			"flexShrink field is the JSON as written — 0 for unset, -1 for "+
			"core.ShrinkNone — and passing it as a factor pins every node in every "+
			"layout while unpinning the one node that asked.", swiftRenderer)
	}
}

// The Compose renderer must apply pinMainAxis, on the right axis, from both
// children loops.
//
// The axis argument is the half worth pinning by value. A Row's main axis is
// its width and a Column's is its height; transposing the two leaves a pinned
// child unbounded across the axis nothing was measuring it on, so it is
// squeezed exactly as before and the declaration goes back to doing nothing —
// with the modifier still applied, still named, and still compiling.
func TestTheComposeChildrenLoopsPinOnTheirOwnAxis(t *testing.T) {
	for _, loop := range []struct {
		anchor, want, axis string
	}{
		{"private fun RowScope.RowChildren(", "pinMainAxis(horizontal = true)",
			"a Row stacks along its width"},
		{"private fun ColumnScope.ColumnChildren(", "pinMainAxis(horizontal = false)",
			"a Column stacks along its height"},
	} {
		body := codeOf(t, kotlinRenderer, loop.anchor)
		if !strings.Contains(body, "shrinkPinned") {
			t.Errorf("%s: %s never reads shrinkPinned — core.FlexShrink(0) parses on "+
				"this target and reaches no layout decision, which is the state the "+
				"other three targets were in before core.ShrinkNone existed",
				kotlinRenderer, loop.anchor)
		}
		if !strings.Contains(body, loop.want) {
			t.Errorf("%s: %s does not apply %s (%s). A pin on the cross axis is not a "+
				"weaker version of a pin on the main one — it is no pin at all, since "+
				"nothing was constraining that axis, and the child is squeezed exactly "+
				"as it was.", kotlinRenderer, loop.anchor, loop.want, loop.axis)
		}
	}
}

// And pinMainAxis must do the two things that make it a pin.
//
// Both halves are load-bearing and each fails silently on its own:
//
//	measure unbounded   without it the child is measured against the space that
//	                    is left, which is the squeeze the declaration exists to
//	                    refuse
//
//	report the measured a `layout(constraints.constrain(...))` compiles, looks
//	size                more correct — a well-behaved layout respects its
//	                    constraints — and clamps the size the parent is told
//	                    about, so the Row's running total never exceeds its own
//	                    maximum, nothing overflows, and the child is drawn
//	                    clipped to a box the parent still believes fits
func TestTheComposePinMeasuresUnboundedAndReportsWhatItMeasured(t *testing.T) {
	body := codeOf(t, kotlinRenderer, "private fun Modifier.pinMainAxis(")
	for _, want := range []string{
		"maxWidth = Constraints.Infinity",
		"maxHeight = Constraints.Infinity",
		"layout(placeable.width, placeable.height)",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("%s: pinMainAxis does not contain %q. Without it the modifier is "+
				"applied, named after what it no longer does, and core.FlexShrink(0) is "+
				"inert on this target again.", kotlinRenderer, want)
		}
	}
	if strings.Contains(body, "constraints.constrain(") {
		t.Errorf("%s: pinMainAxis constrains the size it reports. The parent then "+
			"learns a size that fits, so its running total never overflows and the "+
			"pinned child is drawn spilling out of a box the Row believes it fits in "+
			"— which is neither the squeeze nor the overflow, and matches no target.",
			kotlinRenderer)
	}
}

// The six ways this package reads a native file, named for the question each
// one answers.
//
// # The decision that nobody used to make
//
// There is one scanner now (maskNonCode) and it has two levels — comments out,
// or comments and string literals out. Which level a check got was decided by
// which HELPER it happened to call, and the helpers were named for what they
// read rather than for what they were being asked:
//
//	readNative   the whole file, raw. 53 call sites, and most of them asked
//	             "does the renderer call this" — a question a doc comment
//	             mentioning the call satisfies, which is the exact defect that
//	             put the mask into declSource in the first place.
//	declSource   one declaration, comments blanked. Better, and still open on
//	             the other half: a string literal naming a call satisfies it.
//	codeOf       one declaration, comments and literals blanked. Used by
//	             shrink_test.go and nowhere else.
//
// So the strongest reader existed, was correct for a good deal of the package,
// and was used by one file. That is not a fault in any particular check; it is
// a decision made by accident 74 times.
//
// It was also not a decision anybody could have made correctly by inspection.
// Converting every call site to the code-level reader and running the suite
// moved 43 checks — more than half — and what moved them was the compiler and
// the tests rather than a reading: a check whose subject is a dispatch arm
// fails loudly when the arm is blanked, and a check whose subject is a note
// fails loudly when the note is. The distribution that came out is
//
//	codeIn / codeOf       44   does the renderer DO this
//	valuesIn / valuesOf   33   does it LIST this value
//	proseIn               11   does the file SAY this
//
// which is not a shape anybody would have guessed, and is the reason the answer
// had to be per question rather than per file.
//
// # What replaces it
//
// The question is the name. A call site says which of three things it is asking
// and gets the mask that suits it, and the two primitives are no longer called
// anywhere else — TestEveryNativeReadNamesItsQuestion holds that.
//
//	codeIn / codeOf       does the renderer DO this: attach a modifier, call a
//	                      function, read a field. Comments and literals both
//	                      blanked, because either can spell the thing being
//	                      looked for without doing it.
//	valuesIn / valuesOf   does the renderer LIST this VALUE: a dispatch's arms
//	                      are string literals ("center", "flex-end") and they
//	                      ARE the subject. Comments blanked, literals kept.
//	proseIn / proseOf     does the file SAY this: a refusal's own wording, a
//	                      paragraph a reader is sent to. The subject is the
//	                      prose, so nothing is blanked — and naming it is what
//	                      keeps "I want the comments" from being the accidental
//	                      default it used to be.
//
// The `In` suffix takes a whole file and `Of` cuts one declaration; that half of
// the choice was already deliberate. Prose had no `Of` at first, and that was a
// gap in the cut rather than in the naming: declSource blanks the comments
// before it looks and then runs from the declaration line DOWN, and a
// declaration's note is written above it. proseOf is the cut that goes the
// other way — see proseSourceOf — so a question about one function's note is
// asked of that function rather than of a thousand-line renderer.
//
// maskSwiftNonCode is named for Swift and is not specific to it. Kotlin and
// Java spell line comments, block comments and both kinds of double-quoted
// literal identically, which is the same argument matchingBrace makes for
// serving two languages with one scanner; Groovy adds an apostrophe string and
// a tripled one, which the scan knows too — that was the one place a reader
// here was stronger than the scanner behind it.
func codeOf(t *testing.T, file, anchor string) string {
	t.Helper()
	return maskSwiftNonCode(declSource(t, file, anchor))
}

// codeIn is codeOf over a whole file, for the checks whose subject is not one
// declaration — a call that must appear twice, a spelling that must appear
// nowhere.
func codeIn(t *testing.T, file string) string {
	t.Helper()
	return maskSwiftNonCode(readNative(t, file))
}

// valuesOf is one declaration with its literals intact: a dispatch's arms are
// the subject, and blanking them would delete the thing being read and leave a
// parse that finds nothing.
func valuesOf(t *testing.T, file, anchor string) string {
	t.Helper()
	return declSource(t, file, anchor)
}

// valuesIn is the same over a whole file.
func valuesIn(t *testing.T, file string) string {
	t.Helper()
	return maskComments(readNative(t, file))
}

// proseIn is the file as written, for the checks whose subject is a comment: a
// refusal's own wording, a paragraph a reader is sent to.
//
// It is the weakest reader and the only one a comment can satisfy, which is why
// it has a name of its own rather than being what a caller gets for not
// choosing.
func proseIn(t *testing.T, file string) string {
	t.Helper()
	return readNative(t, file)
}

// proseOf is one declaration and the note above it, prose intact.
//
// # The reader that was missing
//
// Every other question here had both an `In` and an `Of`, and prose had only
// the file. That was not a gap in the naming, it was a gap in the CUT:
// declSource blanks the comments before it looks for the anchor and then runs
// from the declaration line DOWN, and a declaration's note is written above it.
// So a check about "grMobSelectedTrait explains that SwiftUI has no word for
// the off state" had to be a check about GrMobStyle.swift, and two sites were —
// each with a comment saying it would rather not be.
//
// The difference that makes is the same one every `Of` makes, and it is larger
// here than anywhere else: proseIn is satisfied by the phrase appearing
// ANYWHERE in a 1,000-line renderer, so a note moved to another declaration, or
// left behind after the declaration it explained was deleted, goes on passing.
// A note is exactly the kind of thing that gets left behind.
//
// See proseSourceOf for where the region starts and stops. Both ends are the
// comment block, which is what makes this a cut about a note rather than a cut
// that happens to keep them.
func proseOf(t *testing.T, file, anchor string) string {
	t.Helper()
	src, ok := proseSourceOf(readNative(t, file), anchor)
	if !ok {
		t.Fatalf("%s: no %s found in code — if it was renamed or restructured, "+
			"update this test.\n\n"+
			"A mention of it in a comment does not count: the anchor is looked for in "+
			"the masked source even though what is returned is the raw one, because "+
			"otherwise a note ABOUT this declaration could locate the region the note "+
			"is then read out of.", file, anchor)
	}
	return src
}

// Every read of a native file goes through one of the five, and this is what
// says so.
//
// # Why a scan and not a convention
//
// The whole of item 2 was that "which mask does this check get" was answered by
// which helper somebody reached for, and the helpers were named after what they
// read. Renaming them fixes the 74 call sites that exist; it does nothing about
// the 75th. A new check written next week reaches for the nearest thing that
// returns a string, and the nearest thing is whatever the file above it used.
//
// So the two primitives are closed. readNative and declSource are the raw read
// and the declaration cut; they are called by the five readers and nowhere
// else, which means a new call site has to name its question to get a string at
// all. That is the difference between a rule and a convention.
//
// # The recursion, which is deliberate
//
// This scans the package's own source through maskNonCode — the same scanner
// the readers are about. It has to: this file's own comment names both
// primitives repeatedly, and so does the paragraph above. A check that read its
// own explanation as a call site would fail on the sentence describing what it
// enforces, which is the failure wasm/verify's citation walk exempts itself
// from for the same reason.
//
// Counting rather than locating, because the count is the claim: each primitive
// is called exactly as many times as there are readers that need it, and a
// sixth call is a call site that has not chosen.
func TestEveryNativeReadNamesItsQuestion(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("reading this package: %v", err)
	}

	for _, c := range []struct {
		primitive string
		// want is how many call sites there should be, and by is which readers
		// they are — named so a failure says what the budget is spent on.
		want int
		by   string
		why  string
	}{
		{
			primitive: "readNative(t,", want: 4,
			by: "codeIn, valuesIn, proseIn and proseOf",
			why: "the raw read. A check calling it directly is one whose subject can " +
				"be satisfied by a comment, and that is the defect the mask was put " +
				"into declSource for in the first place",
		},
		{
			primitive: "declSource(t,", want: 2,
			by:  "codeOf and valuesOf",
			why: "the declaration cut, which blanks comments and leaves literals",
		},
	} {
		found := 0
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
				continue
			}
			raw, err := os.ReadFile(e.Name())
			if err != nil {
				t.Fatalf("reading %s: %v", e.Name(), err)
			}
			// Through the scanner this whole exercise is about: the prose
			// above names both primitives, and so does every doc comment that
			// explains which reader to use.
			code, _, _ := maskNonCode(string(raw), true)
			found += strings.Count(code, c.primitive)
		}
		if found != c.want {
			t.Errorf("%s is called %d times in this package and should be called %d, by "+
				"%s.\n\n%s\n\nA check reads a native file by naming the question it is "+
				"asking — does the renderer DO this (codeIn/codeOf), does it LIST this "+
				"VALUE (valuesIn/valuesOf), or does the file SAY this (proseIn/proseOf) — and "+
				"gets the mask that suits it. A call to the primitive is a call site "+
				"that has not chosen, which is how the strongest reader in this package "+
				"came to be used by one file out of twenty-five.",
				c.primitive, found, c.want, c.by, c.why)
		}
	}
}
