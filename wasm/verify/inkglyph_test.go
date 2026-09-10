package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
	"unicode"
)

// The fold inkGlyphPerCharacter refuses through, and the one browser.mjs looks
// for the same pairs through.
//
// # Two tests about one property
//
// inkGlyphPerCharacter refuses a fixture string that holds a pair a face may
// draw as one glyph. browser.mjs's inkLigatureNote fires on a run where such a
// string reached the grid anyway, and says which pair. They are the same
// question asked at the two ends of one pipeline, and for a long time only the
// second folded: the refusal matched raw bytes and let the printable-ASCII arm
// stand in for everything a fold would have caught.
//
// That is now closed at both ends — see inkGlyphFold — and this is what holds
// it closed. Nothing else in either directory compares the two, and the failure
// they produce when they drift is a confident sentence about the wrong suspect
// rather than a crash.
func TestTheLigatureRefusalFoldsWhatTheNoteFolds(t *testing.T) {
	// The precomposed spelling of each seed that has one. Not every seed does
	// — "fb", "fh", "fj", "fk" and "ft" have no Presentation Form — and the
	// ones that do are exactly the characters inkLigatureForms carries.
	composed := map[string]string{
		"ff": "ﬀ",
		"fi": "ﬁ",
		"fl": "ﬂ",
		"st": "ﬅ",
	}
	for seed, one := range composed {
		if !strings.Contains(strings.Join(inkLigatureSeeds, " "), seed) {
			t.Fatalf("inkLigatureSeeds no longer names %q, so the row below is "+
				"about a pair nothing refuses.", seed)
		}
		// The word an author would actually write: the pair as one code point,
		// with ordinary letters around it.
		word := "a" + one + "e"
		why := inkGlyphPerCharacter(word)
		if why == "" {
			t.Errorf("inkGlyphPerCharacter accepts %q, which is %q written as one "+
				"code point.\n\n"+
				"The refusal exists so that a word a face is entitled to draw as one "+
				"glyph fails at the fixture rather than arriving in browser.mjs's "+
				"inkFaceFault as a font substitution. A precomposed ligature IS that "+
				"glyph, already joined, and the glyph count it produces is the one "+
				"the refusal is here to keep out.", word, seed)
			continue
		}
		if !strings.Contains(why, strconv.Quote(seed)) {
			t.Errorf("inkGlyphPerCharacter refuses %q with %q, which does not name "+
				"the pair %q.\n\n"+
				"Both arms are true of a precomposed ligature and only one of them "+
				"tells the author what they wrote. Sending them to look for an "+
				"invisible character when they have written a ligature pair is the "+
				"wrong-suspect failure inkLigatureNote's own fold was added to "+
				"prevent, arriving here from the other end.", word, why, seed)
		}
		if strings.Contains(why, "printable ASCII") {
			t.Errorf("inkGlyphPerCharacter refuses %q as non-ASCII (%q) rather than "+
				"as a pair. The two arms overlap on exactly this string and the pair "+
				"arm has to be asked first.", word, why)
		}
	}

	// A pair split by a character that carries neither an advance nor a glyph
	// is a pair the shaper still joins.
	for _, c := range []struct{ what, s string }{
		{"a zero-width space", "of​fice"},
		{"a soft hyphen", "of­fice"},
		{"a word joiner", "of⁠fice"},
	} {
		why := inkGlyphPerCharacter(c.s)
		if !strings.Contains(why, `"ff"`) {
			t.Errorf("inkGlyphPerCharacter says %q about %q, which is \"office\" with "+
				"%s inside its \"ff\".\n\n"+
				"The character between them has no advance and no glyph, so the face "+
				"joins the pair exactly as if it were not there — and a glyph count "+
				"for the result is the count this refusal exists to keep out of "+
				"inkFaceFault.", why, c.s, c.what)
		}
	}

	// And the other side of the fold, which is the half nothing would notice
	// breaking: the seeds are ASCII today, so folding the string alone finds
	// them. A seed added in a composed form would be looked for UNFOLDED inside
	// a folded string and could never be found — the refusal would go quiet
	// while the pair it names sat in the fixture.
	restore := inkLigatureSeeds
	defer func() { inkLigatureSeeds = restore }()
	inkLigatureSeeds = []string{"ﬁ"}
	if why := inkGlyphPerCharacter("file"); why == "" {
		t.Errorf("with the seed list spelling its pair as U+FB01, "+
			"inkGlyphPerCharacter accepts %q — a string holding that very pair as "+
			"two ordinary letters.\n\n"+
			"Folding one side is enough only while the other side is ASCII, which is "+
			"a property of the list and not of the fold. Both sides go through "+
			"inkGlyphFold so that the two are joined in the fold rather than in an "+
			"assumption about one of them.", "file")
	}
	inkLigatureSeeds = restore

	// An ordinary fixture string is still accepted, so none of the above is a
	// refusal that refuses everything.
	for _, ok := range []string{"January 2026", "illlim", "3", "Overview"} {
		if why := inkGlyphPerCharacter(ok); why != "" {
			t.Errorf("inkGlyphPerCharacter refuses %q: %s. This grid's own strings "+
				"have to pass it.", ok, why)
		}
	}
}

// The presentation forms the table carries, held to the block they come from.
//
// A table with a hole in it is the failure mode: U+FB03 missing leaves "ffi"
// arriving as one code point that folds to itself, contains no seed, and is
// refused — if at all — as non-ASCII. The block is seven characters long and
// every one of them is a pair or a triple this list is about.
func TestTheLigatureFormTableCoversItsBlock(t *testing.T) {
	for r := rune(0xFB00); r <= 0xFB06; r++ {
		to, ok := inkLigatureForms[r]
		if !ok {
			t.Errorf("inkLigatureForms has no expansion for %q (U+%04X), which is one "+
				"of the seven Alphabetic Presentation Forms this fold is about.\n\n"+
				"A character in this block that the table does not carry folds to "+
				"itself: it contains no seed, and a fixture holding it is refused for "+
				"being outside printable ASCII or not at all.", r, r)
			continue
		}
		// Every expansion has to be something the seed list could match, which
		// means ASCII letters and nothing else.
		if to != inkGlyphFold(to) {
			t.Errorf("inkLigatureForms expands %q to %q, which is not already folded. "+
				"The expansion is substituted into a string that is then matched "+
				"against folded seeds, so anything the fold would still change is a "+
				"pair the seeds cannot reach.", r, to)
		}
	}
}

// inkGlyphIgnorable and browser.mjs's INK_FOLD_IGNORABLE, as one set.
//
// Both name the characters that carry neither an advance nor a glyph, and they
// are written twice because one is a Go predicate and the other a JavaScript
// character class. Written twice is the state this whole item is about: a
// character dropped by one fold and kept by the other puts the refusal and the
// note on different strings, and the sentence that comes out names a pair the
// reader cannot find.
//
// Read out of browser.mjs rather than restated here, so this compares the two
// implementations and not two copies of one list.
func TestTheTwoFoldsDropTheSameCharacters(t *testing.T) {
	src, err := os.ReadFile("browser.mjs")
	if err != nil {
		t.Fatalf("browser.mjs cannot be read, and it is the other half of this "+
			"comparison: %v", err)
	}
	// The character class itself, between the brackets of the declaration.
	decl := regexp.MustCompile(`(?s)INK_FOLD_IGNORABLE\s*=\s*\n?\s*/\[([^\]]*)\]/`)
	m := decl.FindSubmatch(src)
	if m == nil {
		t.Fatalf("browser.mjs no longer declares INK_FOLD_IGNORABLE as a bracketed " +
			"character class, so this test cannot read the set it is comparing " +
			"against. Either the declaration moved or the fold stopped being a " +
			"regex — find what dropped those characters there and compare against " +
			"that instead of deleting this.")
	}
	item := regexp.MustCompile(`\\u([0-9A-Fa-f]{4})(?:-\\u([0-9A-Fa-f]{4}))?`)
	type span struct{ lo, hi rune }
	var spans []span
	for _, it := range item.FindAllSubmatch(m[1], -1) {
		lo, _ := strconv.ParseUint(string(it[1]), 16, 32)
		hi := lo
		if len(it[2]) > 0 {
			hi, _ = strconv.ParseUint(string(it[2]), 16, 32)
		}
		spans = append(spans, span{rune(lo), rune(hi)})
	}
	if len(spans) == 0 {
		t.Fatalf("INK_FOLD_IGNORABLE parsed to no characters at all, which is this " +
			"test reading the wrong thing rather than browser.mjs dropping nothing.")
	}
	// # And what the class cannot say, which is why this walk can go past it
	//
	// The items above are `\uXXXX`, which names a UTF-16 code unit — so this
	// class cannot express a code point above U+FFFF at all, and the parser
	// above is not narrower than the thing it reads. A `\u{...}` escape can,
	// and if browser.mjs grew one this parser would skip it silently and the
	// comparison below would report gen.go dropping a character browser.mjs
	// "keeps" while browser.mjs was dropping it too.
	//
	// So the spelling is asserted. It is what licenses `there = false` for
	// every code point above the BMP in the walk below — the half of this
	// comparison that used to stop at U+FFFF with nothing saying why.
	if bytes.Contains(m[1], []byte(`\u{`)) {
		t.Fatalf("INK_FOLD_IGNORABLE contains a \\u{...} escape, which names a code "+
			"point rather than a UTF-16 code unit, and the parser above reads only "+
			"\\uXXXX.\n\n"+
			"The class is %q. Every item this test can see is a BMP code unit, and "+
			"that is what lets the walk below treat every code point above U+FFFF as "+
			"absent from the set — a claim about the SPELLING, which is why it is "+
			"checked rather than assumed. Teach the parser the braced form before "+
			"the class uses it, or the comparison silently stops covering whatever "+
			"the braces hold.", m[1])
	}
	// Every code point, both ways, and not only the plane these characters live
	// in. The interesting failures are at the edges of a range, and enumerating
	// is cheaper than arguing about which edges — and the edge nothing walked
	// was U+FFFF itself: gen.go's predicate is a switch over rune ranges and
	// can name any code point, while the class above cannot name one past the
	// BMP, so an ignorable added up there is a disagreement neither side would
	// report. Today both sides drop nothing there, which is agreement and is
	// also the thing worth asserting: see foldOwnMeasuredOn's astral counts.
	for r := rune(0); r <= 0x10FFFF; r++ {
		there := false
		for _, s := range spans {
			if r >= s.lo && r <= s.hi {
				there = true
				break
			}
		}
		if here := inkGlyphIgnorable(r); here != there {
			above := ""
			if r > 0xFFFF {
				above = "\n\nAnd it is above the BMP, where the disagreement can only " +
					"go one way: INK_FOLD_IGNORABLE is a character class in \\uXXXX " +
					"escapes and cannot name a code point up here, so this is a range " +
					"gen.go's predicate grew and the other side has no spelling for. " +
					"Both files need it, and the class needs the braced escape form."
			}
			t.Fatalf("U+%04X is dropped by %s and kept by %s.%s\n\n"+
				"These two folds are asked the same question at the two ends of one "+
				"pipeline: gen.go refuses a fixture that holds a ligature pair, and "+
				"browser.mjs's inkLigatureNote says which pair a face joined when one "+
				"got through. A character one drops and the other keeps is the two of "+
				"them looking at different strings, and what comes out is a confident "+
				"sentence naming a pair the reader cannot see in the string in front "+
				"of them.", r,
				map[bool]string{true: "inkGlyphIgnorable", false: "INK_FOLD_IGNORABLE"}[here],
				map[bool]string{true: "inkGlyphIgnorable", false: "INK_FOLD_IGNORABLE"}[!here],
				above)
		}
	}
	t.Logf("inkGlyphIgnorable and INK_FOLD_IGNORABLE drop the same characters "+
		"over every one of Unicode's %d code points, not only the BMP: %d spans "+
		"parsed out of the class, and the class asserted to be spelled in \\uXXXX "+
		"escapes, which is what makes `absent above U+FFFF` a reading of it rather "+
		"than a gap in this parser",
		0x110000, len(spans))
}

// The two folds' answers, over the characters they both claim to fold.
//
// The test above compares the character sets the two DROP, which is the half
// that can be read out of a regex. This is the other half: what each of them
// turns a precomposed ligature INTO. browser.mjs gets that from NFKD, which is
// the platform's own table; gen.go gets it from inkLigatureForms, which is
// seven characters somebody wrote down. Nothing compared them, and a table
// written by hand against a normaliser is exactly the kind of claim that is
// true the day it is written.
//
// So browser.mjs's own fold is run — the real declaration and the real
// function, lifted out of the file rather than restated here — and asked the
// same strings.
//
// # Only where they claim to agree
//
// gen.go's fold is narrower everywhere else and says so: NFKD decomposes an
// accented letter and this does not, because a fixture holding one is refused
// by the printable-ASCII arm and never reaches a seed comparison. The inputs
// below are therefore built out of what both sides claim: ASCII, the seven
// presentation forms, the long s, and the format characters. An input outside
// that set would be measuring a divergence both files document.
func TestTheTwoFoldsAgreeOnTheFormsTheyBothCarry(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("no node on this machine, so browser.mjs's own fold cannot be run " +
			"and inkLigatureForms is held only by the block census above. Every " +
			"other check in this file still runs; this is the one that compares " +
			"the hand-written expansion table against a real NFKD.")
	}
	src, err := os.ReadFile("browser.mjs")
	if err != nil {
		t.Fatalf("browser.mjs cannot be read: %v", err)
	}
	decl := regexp.MustCompile(`(?s)const INK_FOLD_IGNORABLE =.*?;\n`).Find(src)
	fn := regexp.MustCompile(`(?s)\nfunction inkFold\(text\) \{.*?\n\}\n`).Find(src)
	if decl == nil || fn == nil {
		t.Fatalf("browser.mjs no longer declares INK_FOLD_IGNORABLE and inkFold in " +
			"the shape this test lifts them out in. Find what folds a string there " +
			"and run that instead of deleting this — the table it holds is seven " +
			"characters somebody typed against a normaliser they could not call.")
	}

	// Every character both folds carry, alone and inside a word, plus the
	// seeds themselves and this grid's own strings.
	inputs := []string{"file", "January 2026", "illlim", "3", "OFFICE"}
	for r := rune(0xFB00); r <= 0xFB06; r++ {
		inputs = append(inputs, string(r), "a"+string(r)+"e", "A"+string(r)+"E")
	}
	inputs = append(inputs, "\u017F", "\u017Ftate", "of\u200Bfice", "of\u00ADfice",
		"o\u2060f\u2060f\u2060i\u2060c\u2060e")
	inputs = append(inputs, inkLigatureSeeds...)

	in, err := json.Marshal(inputs)
	if err != nil {
		t.Fatalf("the inputs will not marshal: %v", err)
	}
	script := filepath.Join(t.TempDir(), "fold.mjs")
	if err := os.WriteFile(script, []byte(string(decl)+string(fn)+
		"console.log(JSON.stringify("+string(in)+".map(inkFold)));\n"), 0o644); err != nil {
		t.Fatalf("the lifted fold will not write: %v", err)
	}
	out, err := exec.Command(node, script).Output()
	if err != nil {
		t.Fatalf("node will not run browser.mjs's own fold: %v", err)
	}
	var theirs []string
	if err := json.Unmarshal(out, &theirs); err != nil {
		t.Fatalf("browser.mjs's fold returned something this test cannot read (%q): %v",
			out, err)
	}
	for i, input := range inputs {
		mine := inkGlyphFold(input)
		if mine == theirs[i] {
			continue
		}
		t.Errorf("the two folds disagree about %q: gen.go's inkGlyphFold says %q and "+
			"browser.mjs's inkFold says %q.\n\n"+
			"They are one question asked at the two ends of one pipeline. "+
			"inkGlyphPerCharacter refuses a fixture that holds a pair a face may draw "+
			"as one glyph; inkLigatureNote fires when such a string reached the grid "+
			"anyway and says which pair. A character the two expand differently is a "+
			"refusal and a note looking at different strings, and what comes out is a "+
			"confident sentence naming a pair the reader cannot find. This input is "+
			"one both files claim to carry — a presentation form, a long s, a format "+
			"character or plain ASCII — so the divergence is not the documented one.",
			input, mine, theirs[i])
	}
}

// How much narrower gen.go's fold is than the one it is held against, and what
// is holding the difference.
//
// # A documented gap is not a measured one
//
// The test above asks only about inputs both sides claim — ASCII, the seven
// presentation forms, the long s, the format characters — and says so: NFKD
// decomposes an accented letter and inkGlyphFold does not, because a fixture
// holding one is refused by inkGlyphPerCharacter's printable-ASCII arm and
// never reaches a seed comparison. That is the same lean the pair arm was
// taken off, still holding up everything else, and nothing said how wide it
// was or where its edge is.
//
// So the whole plane is walked. browser.mjs's own fold is run over every BMP
// code point — the real declaration and the real function, lifted out of the
// file — and each answer compared with gen.go's.
//
// \tagreed        what both folds do the same thing to
// \tgap           what NFKD reaches and inkGlyphFold does not
// \tseed-bearing  the part of the gap that MATTERS: a character NFKD turns
// \t              into a letter one of the seeds is spelled with, so that it
// \t              completes a pair for inkLigatureNote and not for the refusal
//
// # The two edges
//
// The gap is allowed to be wide. What it may not do is reach printable ASCII,
// because that is the whole of what the second arm refuses on: a character both
// inside 0x20..0x7e and folded by NFKD alone would pass the pair arm (the
// narrow fold finds nothing), pass the ASCII arm (it is ASCII), and arrive in
// browser.mjs as the glyph-count mismatch this refusal exists to stop.
//
// And gen.go may not be WIDER anywhere: a string it folds into a seed that
// NFKD does not is a fixture refused for a pair inkLigatureNote could never
// report, which is the same two ends looking at different strings from the
// other direction.
//
// Both are asserted. The census is logged, because a bound with no reading of
// the population under it is the state the item before this one was about.
// The build a census of this gap is a fact about.
//
// # A number with nothing behind it
//
// The census below is NFKD as the ICU compiled into one node implements it,
// held against a table somebody wrote down. Both halves move, and they move for
// different reasons: the table moves when somebody edits gen.go, and the fold
// moves when the machine's node does — a newer ICU decomposes characters an
// older one left alone, and every number in the log line moves with it.
//
// Only one of those is a finding. Until this record existed the line printed
// whichever numbers it got and asserted the two edges, so a run whose gap had
// halved because the table lost half its rows printed the new number and said
// nothing — the same shape INK_OWN_MEASURED_ON is for, one directory over, and
// the one this file's own affordedMeasuredOn-shaped neighbours already record.
//
// # And the record makes the difference decidable
//
// With the build in hand the two are separable rather than hedged between:
//
// \tthe build moved      this run's ICU is not the record's. Every count is
// \t                     expected to differ, the two edges still hold, and what
// \t                     is wanted is a re-take rather than an investigation.
// \tthe table moved      same ICU, same Unicode, different numbers. NFKD cannot
// \t                     have changed, so the difference is gen.go's fold — and
// \t                     a fold that quietly reaches further or less far is the
// \t                     thing this whole file is about.
//
// Unicode rather than the ICU or node version alone, because that is the
// version of the DATA: two node builds carrying one ICU answer identically, and
// pinning the node version would report a re-take as needed every time somebody
// upgrades a patch release. The other two are recorded beside it so a reader
// who has to re-take the reading knows what they are re-taking it on.
type foldBuild struct {
	unicode string
	icu     string
	node    string
}

// What the two folds came to, and on what.
//
// The counts are this census's own measurement and the only one anybody has
// taken. Written down here rather than left in the log line for the reason
// affordedMeasuredOn is: a number in a passing run's output is a number nobody
// is holding, and the whole of this test's argument is that an unheld number
// slides.
var foldMeasuredOn = struct {
	build foldBuild
	// Every code point of the BMP browser.mjs's fold changes, how many of them
	// gen.go's fold agrees with, how many it is narrower on, and how many of
	// THAT reaches a letter a seed is spelled with. The fourth is the one that
	// matters — it is the population the printable-ASCII arm is holding shut —
	// and it is the one that would go quiet without a word.
	changes, agreed, gap, bearing int
	// And the shape of that seed-bearing population rather than its size, the
	// same pair the astral half records below.
	//
	// This is the census that most deserved a shape and was the last to get
	// one. The astral 257 were named first because they were new — the walk
	// had just stopped at U+FFFF — but the BMP's 299 are the population the
	// printable-ASCII arm is holding shut on the plane every fixture in this
	// repository lives on, and 299 could as easily have been three blocks a
	// reader can look up or 299 singletons scattered across the plane. The
	// runs were computed by the walk and thrown away.
	//
	// They are eight regions, and they are not a smaller version of the astral
	// three: Latin letters and modifier letters low down, numerals and
	// symbols in the enclosed and superscript ranges, fullwidth letters at the
	// top — NFKD reaches a seed's own letters from every direction the BMP
	// has, where above it the whole population is three blocks of decorated
	// alphabets in one script. Which regions those are is derived and recorded
	// rather than written out here; see bearingRegions below, and
	// foldRegionsNamed for why a block name is not what came back.
	bearingRuns, bearingClusters int
	// And each of those regions as this run names it: the range, and what
	// Go's Unicode tables say is in it. See foldRegionsNamed.
	//
	// # Why the ranges are recorded here and not in a sentence
	//
	// The eight were named in prose beside the log line — "the accented Latin
	// letters, the phonetic modifiers, the superscripts and Roman numerals" —
	// and that sentence was the half of this census a reader actually used
	// and the only half nothing held. The counts above are held: 96 runs, 8
	// regions. The RANGES were not, and a count cannot see them move — a
	// build whose NFKD re-drew every region while leaving 96 runs in 8
	// groups would print eight different spans under the same two numbers and
	// the same sentence.
	//
	// So the naming is derived on every run and recorded as a whole. It is
	// asserted only when BOTH Unicode versions match their records, because
	// it reads two of them: the population is node's NFKD (foldMeasuredOn's
	// build) and the scripts and categories are Go's (foldOwnMeasuredOn's
	// goUnicode).
	//
	// Each entry is the sentence, the kind words that went into it, and the
	// categories those words are the collapse of. The columns are not
	// decoration: foldReached is a claim about which entries of
	// foldRegionKinds these populations have ever printed, and it used to
	// recover the words by splitting the sentence back up — a parser of this
	// file's own formatter, held to it by nothing.
	//
	// The third column is what makes that claim a whole one. Seventeen
	// categories print six words, so "letters has been printed" is a fact
	// about the union of Lu, Ll, Lt and Lo and about none of them; five
	// entries of the mapping sit inside a reached word having never been
	// reached. See foldRegionName and foldReached.
	//
	// All three are held: against this build by the arm in
	// TestHowWideTheNarrowerFoldIsAndWhatHoldsTheGap, which needs node and
	// both Unicode versions, and against each other on every machine by
	// TestTheRecordedRegionNamesHoldTheirOwnColumns, which needs neither.
	bearingRegions []foldRegionName
	// And the same four over the sixteen planes above it, which this census
	// used to stop at without saying so.
	//
	// # What it cost to stop stopping there
	//
	// The gen.go side of the fold had its astral bound asserted and this side
	// did not: the walk ran `for cp = 0; cp <= 0xFFFF` in node and nothing
	// said what NFKD does above U+FFFF. The reason given was that nobody had
	// priced a walk of a million code points through a child process.
	//
	// It is 128ms and 30KB of JSON — the whole plane set, folded and returned
	// as the 2307 rows that change — against the BMP walk's own cost, which is
	// the same shape and a sixteenth of the size. So the bound was not a cost;
	// it was a walk nobody had run.
	//
	// # And what is up there, which is not nothing
	//
	// The astral gap is not the BMP's gap made smaller. NFKD decomposes the
	// mathematical alphanumerics — U+1D41F MATHEMATICAL BOLD SMALL F folds to
	// "f" — so the seed-bearing population, the one that completes a pair for
	// inkLigatureNote and completes none for the refusal, has hundreds of
	// members above the BMP where the whole of what holds it shut is again the
	// printable-ASCII arm. The witness below is built and asked on both sides
	// of U+FFFF for exactly that reason.
	astralChanges, astralAgreed, astralGap, astralBearing int
	// And the shape of that seed-bearing population rather than its size: how
	// many runs of consecutive code points it falls in, and how many regions
	// those runs make when the ones within foldBearingGap of each other are
	// taken together. Three regions is two blocks and a supplement, which is
	// something a reader can go and look at; the count alone said nothing
	// about whether it was that or 257 characters scattered over sixteen
	// planes.
	astralBearingRuns, astralBearingClusters int
	// And the same for those three. The first of them is the reading's own
	// evidence that it is derived: U+1CCD7..U+1CCE9 is a block Unicode 16.0
	// added, node's NFKD decomposes it, and Go's 15.0.0 tables have no script
	// and no category for a single code point in it. The prose called it "the
	// outlined letters", which was a claim about a block this toolchain does
	// not know exists.
	// Three columns, like the BMP's. This one has the empty pair: Go's tables
	// have no category for a single code point of U+1CCD7..U+1CCE9, so it has
	// neither kind words nor categories, which is the same fact its sentence
	// states — and which the record's own arm holds it to, since an empty
	// column is only allowed under a sentence that says why.
	astralBearingRegions []foldRegionName
}{
	build:   foldBuild{unicode: "16.0", icu: "76.1", node: "22.12.0"},
	changes: 15802,
	agreed:  748,
	gap:     15054,
	bearing: 299,

	bearingRuns:     96,
	bearingClusters: 8,
	// The sentence, and the kind words under it. Both columns are derived and
	// both are asserted; the second is the one foldReachedWords reads.
	bearingRegions: []foldRegionName{
		{text: "U+00CC..U+02E2 Latin letters/modifier letters",
			kinds: []string{"letters", "modifier letters"},
			cats:  []string{"Ll", "Lm", "Lt", "Lu"}},
		{text: "U+1D2E..U+1ECB Latin letters/modifier letters",
			kinds: []string{"letters", "modifier letters"},
			cats:  []string{"Ll", "Lm", "Lu"}},
		{text: "U+2071..U+217C Common/Latin letters/modifier letters/numerals/symbols",
			kinds: []string{"letters", "modifier letters", "numerals", "symbols"},
			cats:  []string{"Ll", "Lm", "Lu", "Nl", "Sc", "So"}},
		{text: "U+249D..U+24E3 Common symbols",
			kinds: []string{"symbols"}, cats: []string{"So"}},
		{text: "U+2C7C Latin modifier letters",
			kinds: []string{"modifier letters"}, cats: []string{"Lm"}},
		{text: "U+3250..U+33FF Common symbols",
			kinds: []string{"symbols"}, cats: []string{"So"}},
		{text: "U+A7F3 Latin modifier letters",
			kinds: []string{"modifier letters"}, cats: []string{"Lm"}},
		{text: "U+FF22..U+FF54 Latin letters",
			kinds: []string{"letters"}, cats: []string{"Ll", "Lu"}},
	},

	astralChanges: 2307,
	// 260, which is every code point above the BMP that gen.go's fold touches
	// at all — see foldOwnMeasuredOn.astralLowered, the same number reached
	// from the other side. So the two folds agree about every astral character
	// either of them changes, and the 2047 below is entirely NFKD reaching
	// where the narrow fold does not.
	astralAgreed:  260,
	astralGap:     2047,
	astralBearing: 257,

	astralBearingRuns:     118,
	astralBearingClusters: 3,
	astralBearingRegions: []foldRegionName{
		// No kinds at all: Go's 15.0.0 has no category for a single code point
		// in this range, which is the reading's own evidence that it is derived.
		{text: "U+1CCD7..U+1CCE9 assigned in neither script nor category by Go's " +
			"Unicode 15.0.0, so newer than it"},
		{text: "U+1D401..U+1D69D Common letters",
			kinds: []string{"letters"}, cats: []string{"Ll", "Lu"}},
		{text: "U+1F111..U+1F190 Common symbols",
			kinds: []string{"symbols"}, cats: []string{"So"}},
	},
}

// foldBuildNote is the sentence a census adds when this run is not the run
// those numbers were taken on.
//
// inkOwnBuildNote's move, and affordedMeasuredNote's: the first thing a reader
// needs is whether the recorded measurement still describes the thing measured.
// Empty when the Unicode versions agree, because then it does.
func foldBuildNote(now foldBuild) string {
	if now.unicode == foldMeasuredOn.build.unicode {
		return ""
	}
	return fmt.Sprintf("\n\nfoldMeasuredOn was taken on Unicode %s (ICU %s, node %s) "+
		"and this run is on Unicode %s (ICU %s, node %s). NFKD is that data, so every "+
		"count above is expected to differ and the difference is not a finding about "+
		"either fold — re-take the record against this build, from the census in the "+
		"log line, and leave the two edges asserting what they assert.\n\n"+
		"What is NOT off on this build is gen.go's own half: "+
		"TestTheNarrowFoldsOwnWidthIsAFactAboutGenGoAlone counts what "+
		"inkGlyphIgnorable drops and what inkLigatureForms rewrites, neither of "+
		"which goes near an ICU, and holds both on any machine that can build this "+
		"package. If what is being looked for here is whether the narrow fold "+
		"changed width, that is the test with the answer.",
		foldMeasuredOn.build.unicode, foldMeasuredOn.build.icu,
		foldMeasuredOn.build.node, now.unicode, now.icu, now.node)
}

// What gen.go's own fold changes, split by which of its three parts changed
// it, and on what.
//
// # The census that stops at the edge of this machine
//
// foldMeasuredOn is four numbers about the two folds together, and its bracket
// runs only where this run's Unicode version matches the record's. That is the
// right gate for what it holds — NFKD is the ICU's data and every one of those
// four moves when the data does — but it leaves a developer on a different
// node, or on no node at all, with the two edges and no census reading
// whatsoever, which is the state the record was written to replace. The gate
// closes exactly where somebody is most likely to be reading it.
//
// The guess was that a bracket might survive a build change as a FRACTION: the
// gap over what the note's fold changes moves less than either count does.
// Nobody could check it — the stability of that ratio is a claim about two ICU
// versions and there is one on this machine — and it turns out not to be
// needed, because the census's gen.go half can be taken with no ICU in it at
// all.
//
// inkGlyphFold is three things in a row and every one of them is local:
//
//	ignorable   inkGlyphIgnorable drops the character. gen.go's own predicate.
//	table       inkLigatureForms rewrites it. gen.go's own map.
//	lowered     strings.ToLower changes it, and nothing else did. Go's Unicode
//	            tables, which move with the toolchain and not with node.
//
// Each code point that the fold changes is changed by exactly one of them —
// the three are tried in that order and the third is the whole of what is left
// — so the counts partition the width of the fold, and the partition is
// asserted rather than assumed. Two of the three are facts about gen.go and
// are held on every machine that can build this package; the third is gated on
// Go's Unicode version the way the other census is gated on node's, which is
// the same reading about a different vendor's copy of the same data.
//
// This is the number the whole file is actually about. Both edges pass over a
// fold that quietly narrowed — narrower is the direction they allow — and
// until now the only thing that could see it was a census that needs node and
// the right ICU.
// # And the plane it was taken over, which is now stated
//
// Every walk in this file ran `for cp := rune(0); cp <= 0xFFFF` and none of
// them said so as a bound. That is a real bound and not a formality: the fold
// is a function of a rune and the sixteen planes above the BMP hold most of
// Unicode's case pairs and every one of its tag characters. Nothing anywhere
// said what the fold does to them, and the arm that compares this census's
// `table` against `len(inkLigatureForms)` had to name "a key outside the BMP"
// as one of two possible causes because nothing could tell it from the other.
//
// So the walk is both, counted apart. Apart rather than summed, because they
// are two different facts: the BMP counts are what every other reading in this
// file is about — the fixtures are printable ASCII and the ligature forms are
// at U+FB00 — and the astral counts are the bound, which is the interesting
// number precisely when it stops being what it is today.
//
//	astralIgnorable   0. inkGlyphIgnorable's ranges all end below U+FFFF, and
//	                  browser.mjs's INK_FOLD_IGNORABLE is a character class
//	                  spelled in \uXXXX escapes, which cannot name a code
//	                  point above the BMP at all. So the two folds agree there
//	                  by construction — and both are silent about U+E0000's tag
//	                  characters and U+1D173's musical format controls, which
//	                  are default-ignorable and which neither drops.
//	astralTable       0. Every row of inkLigatureForms is a presentation form
//	                  in U+FB00's block. This is the count that makes the
//	                  row-census arm below able to say which of its two causes
//	                  it found.
//	astralLowered     260, and Go's tables, so gated the way the BMP's 1173 is.
//	                  Deseret at U+10400 first.
var foldOwnMeasuredOn = struct {
	// Go's Unicode version, from the unicode package this binary was compiled
	// against. The `lowered` counts are that data and nothing else; the others
	// are gen.go's and are asserted whatever this says.
	goUnicode                 string
	ignorable, table, lowered int
	// And the same three over the sixteen planes above the BMP, which every
	// walk in this file used to stop at without saying so.
	astralIgnorable, astralTable, astralLowered int
}{
	goUnicode: "15.0.0",
	ignorable: 24,
	table:     8,
	lowered:   1173,

	astralIgnorable: 0,
	astralTable:     0,
	astralLowered:   260,
}

// foldSpread is how wide inkGlyphFold is over one range of code points, split
// by which of its three steps changed each one.
//
// A struct and a function because the walk is now done twice — the BMP, which
// is what the rest of this file is about, and the planes above it, which is the
// bound nothing stated. Two copies of the switch would be two classifications
// that can drift apart while both look right, which is the complaint the
// pinfixture census one directory over is built around.
type foldSpread struct {
	ignorable, table, lowered, unexplained         int
	ignorableAt, tableAt, loweredAt, unexplainedAt rune
}

// changed is the width of the fold over that range, which the three named
// causes partition.
func (f foldSpread) changed() int {
	return f.ignorable + f.table + f.lowered + f.unexplained
}

// foldSpreadOver walks a range and counts what the fold did to it.
//
// Surrogates are skipped: Go turns a lone one into U+FFFD, so a fold asked
// about it is answering about a different character. They only fall inside the
// BMP, and the test is written so the skip is a property of the walk rather
// than of the caller.
func foldSpreadOver(lo, hi rune) foldSpread {
	f := foldSpread{ignorableAt: -1, tableAt: -1, loweredAt: -1, unexplainedAt: -1}
	for cp := lo; cp <= hi; cp++ {
		if cp >= 0xD800 && cp <= 0xDFFF {
			continue
		}
		s := string(cp)
		if inkGlyphFold(s) == s {
			continue
		}
		// In the order the fold applies them, because that is what makes these
		// four counts a partition: a character that is both ignorable and in
		// the table is dropped and never rewritten, and counting it twice
		// would make the total a number that is not a population.
		switch {
		case inkGlyphIgnorable(cp):
			f.ignorable++
			if f.ignorableAt < 0 {
				f.ignorableAt = cp
			}
		case inkLigatureForms[cp] != "":
			f.table++
			if f.tableAt < 0 {
				f.tableAt = cp
			}
		case strings.ToLower(s) != s:
			f.lowered++
			if f.loweredAt < 0 {
				f.loweredAt = cp
			}
		default:
			f.unexplained++
			if f.unexplainedAt < 0 {
				f.unexplainedAt = cp
			}
		}
	}
	return f
}

func TestTheNarrowFoldsOwnWidthIsAFactAboutGenGoAlone(t *testing.T) {
	// The plane the rest of this file is about, and the sixteen above it —
	// which every walk here used to stop at without saying so. See
	// foldOwnMeasuredOn: counted apart because they are two different facts.
	bmp := foldSpreadOver(0, 0xFFFF)
	astral := foldSpreadOver(0x10000, 0x10FFFF)
	ignorable, table, lowered, unexplained :=
		bmp.ignorable, bmp.table, bmp.lowered, bmp.unexplained
	ignorableAt, tableAt, loweredAt, unexplainedAt :=
		bmp.ignorableAt, bmp.tableAt, bmp.loweredAt, bmp.unexplainedAt
	changed := bmp.changed()

	// The partition, which every count below is a share of.
	//
	// Unreachable while the fold is those three steps: a character the first
	// two leave alone is written out unchanged and then lowered, so the fold
	// changing it and ToLower not changing it cannot both be true. Spelled out
	// rather than left as a `default` that quietly adds to a total, because a
	// fourth cause is a fold that has grown a step nothing here knows about —
	// and the three counts would go on looking like a census of it.
	if unexplained > 0 {
		t.Errorf("inkGlyphFold changes %d of this plane's code points for a reason "+
			"none of its three steps explains, the first being U+%04X (%q→%q).\n\n"+
			"The fold drops ignorables, rewrites what inkLigatureForms names, and "+
			"lowercases what is left — so a character none of those three touches is "+
			"a character the fold returns unchanged. A fourth cause is a step "+
			"somebody added, and the counts below are then shares of a population "+
			"that is not the fold's width.",
			unexplained, unexplainedAt, string(unexplainedAt),
			inkGlyphFold(string(unexplainedAt)))
	}

	// The two halves that are gen.go's own, held on every machine.
	//
	// Neither goes near node or the ICU, so neither is gated on anything: this
	// is the reading foldMeasuredOn's census cannot make off its own build, and
	// it is the half that a change to gen.go actually moves.
	if ignorable != foldOwnMeasuredOn.ignorable {
		t.Errorf("inkGlyphIgnorable drops %d of this plane's code points and "+
			"foldOwnMeasuredOn records %d (the first this run being U+%04X).\n\n"+
			"That predicate is gen.go's own and reads no table anybody else ships, "+
			"so this number moved because somebody edited it. Both of this file's "+
			"edges pass over a fold that dropped one character fewer — a fixture "+
			"holding it is then refused for a pair inkLigatureNote would not report, "+
			"or accepted with a pair nothing here can see, depending which way the "+
			"predicate moved.",
			ignorable, foldOwnMeasuredOn.ignorable, ignorableAt)
	}
	if table != foldOwnMeasuredOn.table {
		t.Errorf("inkLigatureForms rewrites %d of this plane's code points and "+
			"foldOwnMeasuredOn records %d (the first this run being U+%04X).\n\n"+
			"That map is the seven-character table the whole narrow-fold argument "+
			"rests on. A row added or removed moves this number and moves nothing "+
			"else in this file: the pair arm still asks its question in this fold, "+
			"and a form that is no longer rewritten is a spelling of a seed that "+
			"stops being recognised.",
			table, foldOwnMeasuredOn.table, tableAt)
	}
	// And the same count read off the map itself, which is the reading that
	// says every row in it does something.
	//
	// The walk above counts code points this plane holds and the map is what it
	// looks them up in, so the two are the same number unless a row's key is
	// outside the BMP — where no walk here would ever reach it — or a row maps
	// a character to itself, which is a row that is in the table and is not in
	// the fold.
	//
	// It used to name two causes and be unable to tell them apart, because the
	// walk stopped at U+FFFF and nothing counted what was above it. Both are
	// counted now: the astral half of the walk says how many rows the fold
	// rewrites up there, and the map itself says how many rows are keyed there
	// and how many map a character to itself. A message that names the cause it
	// found is the difference between a reader checking one thing and checking
	// the table.
	astralKeys, selfMapping := 0, 0
	var astralKeyAt, selfMappingAt rune = -1, -1
	for key, to := range inkLigatureForms {
		if key > 0xFFFF {
			astralKeys++
			if astralKeyAt < 0 || key < astralKeyAt {
				astralKeyAt = key
			}
		}
		if to == string(key) {
			selfMapping++
			if selfMappingAt < 0 || key < selfMappingAt {
				selfMappingAt = key
			}
		}
	}
	if table+astral.table != len(inkLigatureForms) {
		// Said in the direction the table is actually in, so a reader is sent
		// to the row rather than to the two possibilities.
		cause := "and neither a key above the BMP nor a row mapping its character " +
			"to itself accounts for the difference, which leaves a row this walk " +
			"reaches and does not count — read the switch above"
		switch {
		case astralKeys > 0 && selfMapping > 0:
			cause = fmt.Sprintf("of which %d are keyed above the BMP (first "+
				"U+%04X) and %d map their character to themselves (first U+%04X)",
				astralKeys, astralKeyAt, selfMapping, selfMappingAt)
		case astralKeys > 0:
			cause = fmt.Sprintf("of which %d are keyed above the BMP, the first "+
				"being U+%04X", astralKeys, astralKeyAt)
		case selfMapping > 0:
			cause = fmt.Sprintf("of which %d map their character to themselves, "+
				"the first being U+%04X", selfMapping, selfMappingAt)
		}
		t.Errorf("inkLigatureForms has %d rows and %d of them change a code point "+
			"— %d on the BMP and %d above it — %s.\n\n"+
			"A row that changes nothing is a row present in the table and absent "+
			"from the fold: a spelling of a seed that stops being recognised, with "+
			"the table still looking as though it covers it. A row keyed above the "+
			"BMP is a different thing again — the fold reaches it, and nothing else "+
			"in this file does, because every fixture here is printable ASCII and "+
			"every form the note names is in U+FB00's block.",
			len(inkLigatureForms), table+astral.table, table, astral.table, cause)
	}

	// And the third, which is Go's copy of the same data node has its own copy
	// of.
	//
	// Gated the way foldMeasuredOn's census is gated, and for the same reason:
	// ToLower is the Unicode tables the toolchain was built with, so a Go
	// upgrade moves this number and the move is not a finding about gen.go.
	// What makes it worth recording anyway is that it is the OTHER thing that
	// can change the fold's width, and without it a table edit and a toolchain
	// upgrade would arrive at the total as the same number.
	if unicode.Version == foldOwnMeasuredOn.goUnicode &&
		lowered != foldOwnMeasuredOn.lowered {
		t.Errorf("strings.ToLower changes %d of this plane's code points that "+
			"inkGlyphFold does not otherwise touch, and foldOwnMeasuredOn records "+
			"%d, over the same Go Unicode %s (the first this run being U+%04X).\n\n"+
			"That data has not moved, so what moved is which characters reach the "+
			"lowercasing at all — the two steps before it. This is the same finding "+
			"as the two above arriving through the arm that is supposed to be about "+
			"the toolchain.",
			lowered, foldOwnMeasuredOn.lowered, unicode.Version, loweredAt)
	}
	// # And the same three above the BMP, which is the bound nothing stated
	//
	// The two gen.go counts are held here on every machine, exactly as they are
	// on the BMP — and what they say today is that the fold does NOTHING up
	// there but lowercase. That is worth an assertion rather than a sentence:
	// inkGlyphIgnorable's ranges all end below U+FFFF and browser.mjs's
	// INK_FOLD_IGNORABLE is spelled in \uXXXX escapes, so a range added above
	// the BMP on this side is a range the other side's character class cannot
	// express — the two folds coming apart in the one region no test walked.
	if astral.unexplained > 0 {
		t.Errorf("inkGlyphFold changes %d code points above the BMP for a reason "+
			"none of its three steps explains, the first being U+%04X (%q→%q).\n\n"+
			"The same partition the BMP arm asserts, over the planes nothing here "+
			"used to walk. A fourth cause is a step somebody added to the fold, and "+
			"the counts beside it are then shares of a population that is not the "+
			"fold's width.",
			astral.unexplained, astral.unexplainedAt, string(astral.unexplainedAt),
			inkGlyphFold(string(astral.unexplainedAt)))
	}
	if astral.ignorable != foldOwnMeasuredOn.astralIgnorable {
		t.Errorf("inkGlyphIgnorable drops %d code points above the BMP and "+
			"foldOwnMeasuredOn records %d (the first this run being U+%04X).\n\n"+
			"That predicate is gen.go's own, so this number moved because somebody "+
			"edited it — and the edit is in the region the other fold cannot follow "+
			"it into. browser.mjs's INK_FOLD_IGNORABLE is a character class written "+
			"in \\uXXXX escapes, which name BMP code points and nothing else, so a "+
			"range added here is a character gen.go drops and browser.mjs keeps with "+
			"TestTheTwoFoldsDropTheSameCharacters unable to say so unless it walks "+
			"this far. If tag characters (U+E0000) or the musical format controls "+
			"(U+1D173) are what belong in the fold, both sides need them and the "+
			"class needs a spelling that can reach them.",
			astral.ignorable, foldOwnMeasuredOn.astralIgnorable, astral.ignorableAt)
	}
	if astral.table != foldOwnMeasuredOn.astralTable {
		t.Errorf("inkLigatureForms rewrites %d code points above the BMP and "+
			"foldOwnMeasuredOn records %d (the first this run being U+%04X).\n\n"+
			"Every row of that table is a presentation form in U+FB00's block, which "+
			"is where the ligatures a face joins are spelled. A row keyed above the "+
			"BMP is reached by the fold and by nothing else in this file — the "+
			"fixtures are printable ASCII and the block census walks U+FB00's block "+
			"— so it is a row whose effect no other reading here can see.",
			astral.table, foldOwnMeasuredOn.astralTable, astral.tableAt)
	}
	// And the third, gated on Go's Unicode version for the reason the BMP's is:
	// most of Unicode's newer case pairs live up here, so this is the count a
	// toolchain upgrade moves the most.
	if unicode.Version == foldOwnMeasuredOn.goUnicode &&
		astral.lowered != foldOwnMeasuredOn.astralLowered {
		t.Errorf("strings.ToLower changes %d code points above the BMP that "+
			"inkGlyphFold does not otherwise touch, and foldOwnMeasuredOn records "+
			"%d, over the same Go Unicode %s (the first this run being U+%04X).\n\n"+
			"That data has not moved, so what moved is which characters reach the "+
			"lowercasing at all — the two steps before it, both of which record 0 up "+
			"here. This is the same finding as the two above arriving through the "+
			"arm that is supposed to be about the toolchain.",
			astral.lowered, foldOwnMeasuredOn.astralLowered, unicode.Version,
			astral.loweredAt)
	}

	note := ""
	if unicode.Version != foldOwnMeasuredOn.goUnicode {
		note = fmt.Sprintf(" — foldOwnMeasuredOn's lowercase count was taken on Go "+
			"Unicode %s and this toolchain carries %s, so that one number is expected "+
			"to differ and is not asserted here; the two gen.go counts above are, and "+
			"they are the ones a change to the fold moves",
			foldOwnMeasuredOn.goUnicode, unicode.Version)
	}
	t.Logf("inkGlyphFold changes %d of the BMP's code points: %d dropped as "+
		"ignorable (e.g. U+%04X) and %d rewritten by inkLigatureForms (e.g. U+%04X), "+
		"both of them gen.go's own and both held on any machine that can build this "+
		"package, and %d lowercased by Go's Unicode %s (e.g. U+%04X). The first two "+
		"are what foldMeasuredOn's census cannot read off a build that is not the "+
		"one it was taken on. Above the BMP — the bound every walk in this file used "+
		"to stop at without saying so — it changes %d: %d ignorable and %d rewritten "+
		"by the table, both asserted at %d and %d, so the fold does nothing up there "+
		"but lowercase, which it does to %d (e.g. U+%04X). That the two gen.go "+
		"counts are zero is what keeps the two folds agreeing there: "+
		"INK_FOLD_IGNORABLE is a character class in \\uXXXX escapes and cannot name "+
		"a code point above U+FFFF at all%s",
		changed, ignorable, ignorableAt, table, tableAt, lowered, unicode.Version,
		loweredAt, astral.changed(), astral.ignorable, astral.table,
		foldOwnMeasuredOn.astralIgnorable, foldOwnMeasuredOn.astralTable,
		astral.lowered, astral.loweredAt, note)
}

// How far apart two runs of seed-bearing code points have to be before they
// are counted as different regions.
//
// 256, and it is asked of both planes' populations, which have different
// margins around it:
//
//	above the BMP   three regions. The widest gap INSIDE one is 67 code points
//	                and the narrowest BETWEEN two is 1816, so anything from 68
//	                to 1816 gives the same three.
//	the BMP         eight regions, and the margin is narrower but real: 162
//	                inside and 422 between, so anything from 163 to 421 gives
//	                the same eight.
//
// So the constant is not doing work on either side — it is naming where a
// reader should look. The log line prints both gaps for both planes, because a
// build that moved them is a build that re-drew the regions, and the region
// COUNTS are recorded and asserted, so a constant that started choosing the
// answer would show up as a count that moved rather than as nothing at all.
const foldBearingGap = 0x100

// foldClustersOf merges runs of code points into regions, so a census of a
// population says what SHAPE it is and not only how large.
//
// 257 characters in three regions is two blocks a reader can look up; 257
// scattered singletons would be a different fact about NFKD, and the count on
// its own cannot tell them apart. Returns the regions and the two gaps the
// constant above sits between — the widest inside a region and the narrowest
// between two — because those are what say whether the constant chose the
// answer.
//
// Asked of both planes. The astral population was shaped first because it was
// new; the BMP's 299 are the ones the printable-ASCII arm holds shut on the
// plane every fixture lives on, and they were a count with nothing behind it
// for exactly as long.
func foldClustersOf(runs [][2]rune) (clusters [][2]rune, inside, between rune) {
	between = -1
	for _, run := range runs {
		if n := len(clusters); n > 0 && run[0]-clusters[n-1][1] <= foldBearingGap {
			if gap := run[0] - clusters[n-1][1]; gap > inside {
				inside = gap
			}
			clusters[n-1][1] = run[1]
			continue
		}
		if n := len(clusters); n > 0 {
			if gap := run[0] - clusters[n-1][1]; between < 0 || gap < between {
				between = gap
			}
		}
		clusters = append(clusters, run)
	}
	return clusters, inside, between
}

// foldRegionKinds is the general categories Unicode gives a set of code
// points, as the words a sentence about them would use.
//
// One word per category rather than the category's own two letters, because
// what a reader wants out of a region is what KIND of thing is in it — the
// difference between "the circled letters" and "the accented ones" is Lu/Ll
// against So, and "So" says nothing to somebody reading a log line. The
// mapping is a translation of a derived fact and not a judgement about which
// region is which: the categories come from Go's tables, and an unmapped one
// falls through as its own two letters rather than as silence.
//
// # Seventeen of the thirty-one, and the other fourteen are written down too
//
// This list was assembled by looking at what these two populations happen to
// contain. The fall-through is honest — an unmapped category arrives as its
// own two letters, which a reader can look up — but it made two very
// different things look identical: a category deliberately left as its code,
// and a category nobody had thought about. A build whose NFKD reached into Nd
// would print "numerals" and one that reached Cf would print "Cf", and
// nothing said which of those was a decision.
//
// So the complement is enumerated as well, in foldUnwordedCategories, and
// TestTheRegionKindsAccountForEveryCategoryGoHas asserts that the two lists
// partition unicode.Categories' two-letter entries exactly. Every category
// Go has is then either a word or a written-down decision not to give it one,
// and a Unicode version that adds a category to Go's tables is a failure
// naming it rather than a two-letter code in a log line.
var foldRegionKinds = []struct{ category, word string }{
	{"Lu", "letters"},
	{"Ll", "letters"},
	{"Lt", "letters"},
	{"Lo", "letters"},
	{"Lm", "modifier letters"},
	{"Nd", "numerals"},
	{"Nl", "numerals"},
	{"No", "numerals"},
	{"So", "symbols"},
	{"Sk", "symbols"},
	{"Sm", "symbols"},
	{"Sc", "symbols"},
	{"Mn", "marks"},
	{"Mc", "marks"},
	{"Me", "marks"},
	{"Pd", "punctuation"},
	{"Po", "punctuation"},
}

// foldUnwordedCategories is every two-letter general category foldRegionKinds
// deliberately gives no word to, and why.
//
// Not a gap: the complement of the list above, written down so the two
// together are a partition of what Go's tables carry. A category here prints
// as its own two letters (see foldKindOf), which is the same output an
// unmapped one would produce — and the whole point of the list is that the
// output being the same does not make the two cases the same. One is a
// decision and the other is an oversight, and only one of them should survive
// a Unicode version bump.
//
// The reasons fall into three groups.
//
//	the punctuation shapes   Pc Ps Pe Pi Pf. "punctuation" is the word Pd and
//	                         Po get, and flattening the paired and quoting
//	                         forms into it would lose the only thing that
//	                         makes them worth naming: a region of Ps/Pe is
//	                         brackets, which is a different sentence from a
//	                         region of dashes. There is no one-word English
//	                         for "initial quotation mark", so the code stands.
//	the separators           Zs Zl Zp. Not a kind of glyph at all — a region
//	                         of these is whitespace, and nothing NFKD produces
//	                         from a seed's letters lands there. If one ever
//	                         does, the two letters are the right amount of
//	                         surprise.
//	the non-graphic          Cc Cf Cs Co, and LC. The C-classes are controls,
//	                         formatting, surrogates and private use: they have
//	                         no appearance to describe, and "Cf" in a log line
//	                         is a more honest thing to print than a word
//	                         implying there is something to look at. LC is not
//	                         a category at all but Go's union of Lu, Ll and
//	                         Lt, all three of which are worded above — a code
//	                         point can only reach it by way of one of those,
//	                         so it is unreachable rather than undescribed.
//
// Cn is absent from both lists on purpose and the arm knows it: unassigned is
// not a kind of thing, it is the absence of one, and foldRegionsNamed reports
// it as "newer than Go's tables" a level up rather than as a category.
//
// # The three groups are in the data now, and each of them is checkable
//
// The groups above were a paragraph and the entries were a flat map from a
// category to a sentence, and the arm only ever read the KEY. So the reasons
// were prose nothing held: "no appearance to describe" beside a category that
// acquired one, or a shape filed under the wrong heading, would sit here
// indefinitely — in a list whose entire purpose is to be the written-down half
// of a decision.
//
// Each entry carries its group, and the group is not decoration either. It is
// re-derived from the category's own name (P, Z and C are the three, and LC is
// the one entry that is not a category), and each group makes a claim Go's
// tables can answer:
//
//	punctuation shapes   the reason is "the word would be punctuation, which Pd
//	                     and Po already have" — so foldRegionKinds must still
//	                     give Pd and Po one word between them. Reword either and
//	                     these five reasons are describing a collision that has
//	                     stopped happening.
//	separators           "a region of these is whitespace, not a kind of glyph"
//	                     — so every code point in them is unicode.IsSpace.
//	non-graphic          "no appearance to describe" — so not one code point in
//	                     them is unicode.IsGraphic.
//	not a category       LC is Go's union of Lu, Ll and Lt, and the claim that
//	                     nothing can reach it undescribed is the claim that all
//	                     three of those are worded above.
//
// See TestTheRegionKindsAccountForEveryCategoryGoHas, which holds all four.
type foldUnwordedGroup string

const (
	foldPunctuationShapes foldUnwordedGroup = "punctuation shapes"
	foldSeparators        foldUnwordedGroup = "separators"
	foldNonGraphic        foldUnwordedGroup = "non-graphic"
	foldNotACategory      foldUnwordedGroup = "not a category"
)

var foldUnwordedCategories = map[string]struct {
	group foldUnwordedGroup
	why   string
}{
	"Pc": {foldPunctuationShapes, "connector punctuation: the word would be " +
		"\"punctuation\", which Pd and Po already have and which loses what " +
		"makes this one worth naming"},
	"Ps": {foldPunctuationShapes,
		"open punctuation — a region of these is brackets, not \"punctuation\""},
	"Pe": {foldPunctuationShapes, "close punctuation, for the same reason as Ps"},
	"Pi": {foldPunctuationShapes,
		"initial quotation mark, which has no one-word English"},
	"Pf": {foldPunctuationShapes, "final quotation mark, for the same reason as Pi"},
	"Zs": {foldSeparators, "a space separator is not a kind of glyph"},
	"Zl": {foldSeparators, "a line separator is not a kind of glyph"},
	"Zp": {foldSeparators, "a paragraph separator is not a kind of glyph"},
	"Cc": {foldNonGraphic, "a control character has no appearance to describe"},
	"Cf": {foldNonGraphic, "a formatting character has no appearance to describe"},
	"Cs": {foldNonGraphic,
		"a surrogate is an encoding artefact and never a character"},
	"Co": {foldNonGraphic,
		"private use: by definition nobody outside its user knows what it is"},
	"LC": {foldNotACategory, "not a category but Go's union of Lu, Ll and Lt, " +
		"all three of which are worded above, so nothing can reach this that " +
		"has not already been named"},
}

// foldUnwordedGroupOf is the group a category's own name puts it in.
//
// The point of deriving it is that the group in the data is then a claim
// rather than a label: an entry filed under the wrong heading is a failure,
// where a `group` field nothing re-derived would be one more piece of prose in
// a list of prose.
//
// LC is the exception and is the reason the second return exists: it is not a
// general category at all but Go's union of the three cased-letter ones, and
// its first letter says "letter", which is exactly what it is NOT one of.
func foldUnwordedGroupOf(category string) (foldUnwordedGroup, bool) {
	if category == "LC" {
		return foldNotACategory, true
	}
	switch {
	case strings.HasPrefix(category, "P"):
		return foldPunctuationShapes, true
	case strings.HasPrefix(category, "Z"):
		return foldSeparators, true
	case strings.HasPrefix(category, "C"):
		return foldNonGraphic, true
	}
	return "", false
}

// foldCodePointFailing is the first code point of a category that does not
// satisfy a predicate, walking Go's own range table for it.
//
// Every code point rather than a sample: the claim being checked is about all
// of them ("no appearance to describe", "a region of these is whitespace"),
// and a sampled claim is the shape of thing this file keeps replacing. The
// largest of these categories is Co at 137,468 code points and the walk is a
// few milliseconds; the alternative — trusting three endpoints per range —
// would miss exactly the one code point somebody carved out.
func foldCodePointFailing(table *unicode.RangeTable, holds func(rune) bool) (rune, bool) {
	for _, r := range table.R16 {
		for cp := rune(r.Lo); cp <= rune(r.Hi); cp += rune(r.Stride) {
			if !holds(cp) {
				return cp, true
			}
		}
	}
	for _, r := range table.R32 {
		for cp := rune(r.Lo); cp <= rune(r.Hi); cp += rune(r.Stride) {
			if !holds(cp) {
				return cp, true
			}
		}
	}
	return 0, false
}

// Every category Go's tables have is either a word or a written-down decision
// not to give it one.
//
// The two lists above are a partition of unicode.Categories' two-letter
// entries, with Cn the one deliberate hole — and this is the arm that says so.
// Without it foldRegionKinds is a list assembled from what two populations
// happen to contain, and the difference between "left as its code on purpose"
// and "never considered" is invisible in the output, because both print the
// same two letters.
//
// Held against Go's own tables rather than against a count, so a toolchain
// whose Unicode data grew a category is a failure that NAMES it. A count would
// only say the total had moved.
func TestTheRegionKindsAccountForEveryCategoryGoHas(t *testing.T) {
	worded := map[string]string{}
	for _, k := range foldRegionKinds {
		if had, seen := worded[k.category]; seen {
			t.Errorf("foldRegionKinds gives %s two words, %q and %q.\n\n"+
				"foldKindOf returns the first match, so the second is dead and a "+
				"reader editing it would change nothing. One entry per category.",
				k.category, had, k.word)
		}
		worded[k.category] = k.word
	}

	// Go's own list, which is the thing this is a claim about. Two-letter
	// entries only: unicode.Categories also carries the one-letter unions (L,
	// N, P, S, Z, C, M), which are not categories a code point is IN so much
	// as sets of the ones it can be.
	goHas := map[string]bool{}
	for name := range unicode.Categories {
		if len(name) == 2 {
			goHas[name] = true
		}
	}

	for name := range goHas {
		_, isWord := worded[name]
		_, isUnworded := foldUnwordedCategories[name]
		switch {
		case name == "Cn":
			// The deliberate hole. See foldUnwordedCategories.
			if isWord || isUnworded {
				t.Errorf("Cn is listed in one of the two category lists.\n\n" +
					"Unassigned is not a kind of thing, it is the absence of one, " +
					"and foldRegionsNamed reports a region of it as \"newer than " +
					"Go's tables\" — a fact about two Unicode versions " +
					"disagreeing rather than a description of what is there. A " +
					"word or a code for it here would turn that finding into a " +
					"line of ordinary output.")
			}
		case isWord && isUnworded:
			t.Errorf("%s is in both lists: foldRegionKinds words it %q and "+
				"foldUnwordedCategories says %q.\n\n"+
				"The two are supposed to partition Go's categories, so a name in "+
				"both makes the reason beside it dead text — foldKindOf finds the "+
				"word first and the sentence explaining why there is no word is "+
				"describing something that does not happen.",
				name, worded[name], foldUnwordedCategories[name].why)
		case !isWord && !isUnworded:
			t.Errorf("Go's Unicode %s carries the general category %s and neither "+
				"list here mentions it.\n\n"+
				"foldKindOf would print it as its own two letters, which is what a "+
				"category deliberately left uncoded prints as too — so a reader "+
				"seeing %q in a region's description cannot tell whether somebody "+
				"decided that or nobody looked. Give it a word in foldRegionKinds "+
				"if a sentence about a region of them would want one, or a reason "+
				"in foldUnwordedCategories if it would not.",
				unicode.Version, name, name)
		}
	}

	// And the other direction: a name in either list that Go does not have.
	// A category removed from the tables leaves a word nothing can reach,
	// which is the same dead text the both-lists case is.
	unwordedNames := map[string]bool{}
	for name := range foldUnwordedCategories {
		unwordedNames[name] = true
	}
	wordedNames := map[string]bool{}
	for name := range worded {
		wordedNames[name] = true
	}
	for _, l := range []struct {
		what string
		of   map[string]bool
	}{{"foldRegionKinds", wordedNames},
		{"foldUnwordedCategories", unwordedNames}} {
		for name := range l.of {
			if !goHas[name] {
				t.Errorf("%s names the category %s and Go's Unicode %s has no "+
					"two-letter category by that name.\n\n"+
					"Nothing can reach that entry, so the word or the reason beside "+
					"it is describing a classification this toolchain does not make. "+
					"Either it is a typo or the tables have dropped a category.",
					l.what, name, unicode.Version)
			}
		}
	}

	// # And the reasons, which were prose nothing read
	//
	// Every entry's group is re-derived from its own name, and every group
	// makes a claim Go's tables can answer. See foldUnwordedCategories: the
	// three headings used to be a paragraph in a doc comment, so a category
	// filed under the wrong one — or a reason that had quietly stopped being
	// true — was invisible to everything here.
	byGroup := map[foldUnwordedGroup][]string{}
	for name, entry := range foldUnwordedCategories {
		byGroup[entry.group] = append(byGroup[entry.group], name)
		want, derivable := foldUnwordedGroupOf(name)
		switch {
		case !derivable:
			t.Errorf("foldUnwordedCategories files %s under %q and its name says "+
				"nothing about which group it belongs in.\n\n"+
				"The three groups are the three general-category classes that get "+
				"no word — P, Z and C — plus LC, which is not a category at all. A "+
				"name outside those four is either a category this list has never "+
				"had (a letter, a mark, a number, a symbol), in which case the "+
				"question is why it has no word, or Go's tables have grown a class "+
				"nobody here has considered.", name, entry.group)
		case entry.group != want:
			t.Errorf("foldUnwordedCategories files %s under %q and its name puts it "+
				"under %q.\n\n"+
				"The group is not a label, it is the claim the reason beside it "+
				"rests on: the punctuation shapes are excused because Pd and Po "+
				"already own the word, the separators because a region of them is "+
				"whitespace, the C-classes because there is nothing to look at. A "+
				"category under the wrong heading is being excused by an argument "+
				"that was made about something else.\n\nIts reason: %s",
				name, entry.group, want, entry.why)
		}
	}

	// The punctuation shapes' reason names two entries of the OTHER list, and
	// that is the whole of it: "the word would be punctuation, which Pd and Po
	// already have". Reword either of those and these five reasons are
	// describing a collision that no longer happens.
	if len(byGroup[foldPunctuationShapes]) > 0 {
		pd, hasPd := worded["Pd"]
		po, hasPo := worded["Po"]
		if !hasPd || !hasPo || pd != po {
			t.Errorf("%d categories are left uncoded because \"punctuation\" is the "+
				"word Pd and Po already have, and foldRegionKinds now words Pd %q "+
				"and Po %q.\n\n"+
				"That is the reason those five entries give — flattening the "+
				"paired, quoting and connector forms into one word would lose the "+
				"only thing that makes them worth naming — and it is an argument "+
				"about a word two other categories share. If they no longer share "+
				"one, the argument has to be re-made rather than left standing: "+
				"%s.",
				len(byGroup[foldPunctuationShapes]),
				map[bool]string{true: pd, false: "nothing"}[hasPd],
				map[bool]string{true: po, false: "nothing"}[hasPo],
				strings.Join(byGroup[foldPunctuationShapes], ", "))
		}
	}

	// And the two groups whose reason is a fact about the code points
	// themselves. Both are stated as absolutes — "a region of these is
	// whitespace", "no appearance to describe" — so both are checked over
	// every code point Go puts in the category.
	for _, c := range []struct {
		group foldUnwordedGroup
		// What the reason claims, as a predicate, and the sentence a
		// counterexample is reported with.
		holds func(rune) bool
		said  string
	}{
		{foldSeparators, unicode.IsSpace,
			"a region of these is whitespace and not a kind of glyph, which is " +
				"why no word for one would be describing anything a reader could " +
				"look at"},
		{foldNonGraphic, func(cp rune) bool { return !unicode.IsGraphic(cp) },
			"these have no appearance to describe, which is why their own two " +
				"letters are a more honest thing to print than a word implying " +
				"there is something to see"},
	} {
		for _, name := range byGroup[c.group] {
			table, ok := unicode.Categories[name]
			if !ok {
				// Already reported by the arm above, which names it as a
				// category Go does not have.
				continue
			}
			cp, failed := foldCodePointFailing(table, c.holds)
			if !failed {
				continue
			}
			t.Errorf("foldUnwordedCategories excuses %s as %q, and U+%04X %q is in "+
				"%s and is not.\n\n"+
				"The group's reason is that %s. Go's own tables now disagree with "+
				"it for at least this code point, so either the category has grown "+
				"members the reason was not written about or the reason was never "+
				"right — and the output is the same two letters either way, which "+
				"is exactly the state this list exists to make impossible.\n\n"+
				"Its reason: %s",
				name, c.group, cp, string(cp), name, c.said,
				foldUnwordedCategories[name].why)
		}
	}

	// And LC, whose reason is a claim about three entries of the other list.
	for _, name := range byGroup[foldNotACategory] {
		if name != "LC" {
			t.Errorf("foldUnwordedCategories files %s under %q, and LC is the only "+
				"name that group is about.\n\n"+
				"It is there because it is Go's union of Lu, Ll and Lt rather than "+
				"a category a code point is in. Another name under the same heading "+
				"is a second such union, which foldKindOf's fall-through would "+
				"print as two letters without anybody having decided that.",
				name, foldNotACategory)
			continue
		}
		for _, member := range []string{"Lu", "Ll", "Lt"} {
			if _, isWord := worded[member]; isWord {
				continue
			}
			t.Errorf("LC is excused as Go's union of Lu, Ll and Lt, \"all three of "+
				"which are worded above\" — and %s has no word in "+
				"foldRegionKinds.\n\n"+
				"That sentence is the whole reason LC needs no word of its own: a "+
				"code point can only reach it by way of one of the three, so it is "+
				"unreachable rather than undescribed. With %s unworded, a region of "+
				"%s prints as its own two letters and LC's entry is describing a "+
				"guarantee this file no longer makes.", member, member, member)
		}
	}

	// The groups as a sentence, in the order the doc comment argues them
	// rather than in map order. Said out loud because the counts are the thing
	// a mis-filed entry moves: five, three, four and one is the partition, and
	// "four punctuation shapes and four non-graphic" is a heading somebody
	// changed.
	groupSaid := []string{}
	for _, g := range []foldUnwordedGroup{foldPunctuationShapes, foldSeparators,
		foldNonGraphic, foldNotACategory} {
		names := append([]string{}, byGroup[g]...)
		sort.Strings(names)
		groupSaid = append(groupSaid, fmt.Sprintf("%d %s (%s)", len(names), g,
			strings.Join(names, " ")))
	}

	reach := foldReached()
	t.Logf("Go's Unicode %s carries %d two-letter general categories: %d have a "+
		"word in foldRegionKinds, %d are written down in foldUnwordedCategories "+
		"as deliberately printing their own code — %s, each group held to the "+
		"claim its reason makes rather than to the sentence alone — and Cn is "+
		"neither because "+
		"unassigned is reported as two Unicode versions disagreeing rather than "+
		"as a kind of character. Of the %d worded entries, the recorded regions "+
		"have printed %d (%s) and have never printed %d (%s) — those last are the "+
		"entries the mapping was written for and these populations have not "+
		"reached, which is a guess about what a region of them would be called "+
		"rather than a translation of one that turned up. At WORD granularity "+
		"that reads as %s printed and %s never, which is the weaker half of the "+
		"same reading: an unprinted word is a sound claim about every category "+
		"under it, and a printed one covers for %d that have never turned up at "+
		"all (%s). See foldReached.",
		unicode.Version, len(goHas), len(worded), len(foldUnwordedCategories),
		strings.Join(groupSaid, ", "),
		len(worded), len(reach.cats), foldCategorySaid(reach.cats),
		len(reach.unusedCats), foldCategorySaid(reach.unusedCats),
		strings.Join(reach.words, ", "), strings.Join(reach.unusedWords, ", "),
		len(reach.hidden), foldCategorySaid(reach.hidden))
}

// foldReached is which entries of foldRegionKinds the recorded regions have
// actually exercised, and which of them nothing has ever printed.
//
// The mapping above was assembled by looking at what these two populations
// contain, and the partition arm fixes only one half of that: it says every
// category Go has is accounted for. The other half is that some of the entries
// were written for categories these populations have never reached, and an
// entry nothing prints is a guess about what a region of them would want to be
// called. Saying so in the log line does not make it wrong — the day NFKD
// reaches a mark, "marks" is the right word and it is already there — it makes
// it visible, which is the difference between a mapping and a list of what
// happened to turn up.
//
// Read off foldMeasuredOn's own recorded regions rather than off this run,
// because those are the populations the mapping was written from and they are
// the ones the claim is about. This runs on any machine — the record is a
// literal — where the regions themselves need node and a matching Unicode
// version to re-derive.
//
// # It used to read the words back out of the sentence
//
// A region's description is "scripts words" with each half "/"-joined, so
// splitting on "/" and matching the longest trailing word recovers the kinds:
// "Latin modifier letters" is Lm and not Lm and Lu at once. It worked, and it
// made this function a parser of foldRegionsNamed's output — two formatters
// with an undeclared contract between them, one of which nobody would think to
// update when the other's separator or wording changed. The words are recorded
// beside the sentence now (see foldRegionName), and the same arm that holds
// the sentence holds them.
//
// # And it answered in WORDS, which is half a claim
//
// Seventeen categories map onto six words, so the answer "letters has been
// printed" is a fact about the union of Lu, Ll, Lt and Lo and about none of
// them. That direction is where the guesses live. An unreached word is sound —
// "marks" unprinted means neither Mn nor Mc nor Me has turned up, because
// nothing else could have printed it — but a reached one covers for however
// many of its categories were never there:
//
//	letters   Lu Ll Lt Lo   printed, and Lo never has been
//	numerals  Nd Nl No      printed by Nl alone
//	symbols   So Sk Sm Sc   printed by So and Sc
//
// Five entries of the mapping sit inside a reached word without ever having
// been reached, and at word granularity they were indistinguishable from the
// four that had. So the regions record their CATEGORIES and this reads those;
// the words are still reported, because they are what the sentence prints, and
// they are derived from the categories rather than counted alongside them.
type foldReach struct {
	// Categories with a word in foldRegionKinds that these populations have
	// printed, and those they never have. In the mapping's own order, which
	// groups them by word — Lu Ll Lt Lo, then Lm, then the numerals — so the
	// two lists read against the table rather than alphabetically.
	cats, unusedCats []string
	// The same at word granularity: a word is printed if any of its categories
	// is, and unprinted only if none of them is.
	words, unusedWords []string
	// The categories nothing has reached whose WORD something else made
	// reached. These are the entries a word-level reading could not see, and
	// they are the ones the mapping is most plainly a guess about: nobody has
	// ever seen a region of them, and the word that would name one is already
	// spoken for.
	hidden []string
}

func foldReached() foldReach {
	seen := map[string]bool{}
	regions := append(append([]foldRegionName{},
		foldMeasuredOn.bearingRegions...),
		foldMeasuredOn.astralBearingRegions...)
	for _, region := range regions {
		for _, cat := range region.cats {
			seen[cat] = true
		}
	}
	// Which words got printed at all, taken before the per-word verdicts
	// below: a word is reached by ANY of its categories, so the whole table
	// has to be read before any entry can be called unreached.
	wordSeen := map[string]bool{}
	for _, k := range foldRegionKinds {
		if seen[k.category] {
			wordSeen[k.word] = true
		}
	}

	var out foldReach
	toldWord := map[string]bool{}
	for _, k := range foldRegionKinds {
		if seen[k.category] {
			out.cats = append(out.cats, k.category)
		} else {
			out.unusedCats = append(out.unusedCats, k.category)
			if wordSeen[k.word] {
				out.hidden = append(out.hidden, k.category)
			}
		}
		if toldWord[k.word] {
			continue
		}
		toldWord[k.word] = true
		if wordSeen[k.word] {
			out.words = append(out.words, k.word)
		} else {
			out.unusedWords = append(out.unusedWords, k.word)
		}
	}
	sort.Strings(out.words)
	sort.Strings(out.unusedWords)
	return out
}

// foldCategorySaid is a category list as the log line prints it, with the
// empty list said as a word rather than as nothing.
func foldCategorySaid(cats []string) string {
	if len(cats) == 0 {
		return "none"
	}
	return strings.Join(cats, " ")
}

// The shape a recorded region's sentence and its two columns are in, held
// against each other with no population at all.
//
// # The half a reader cannot eyeball, and the half no machine was checking
//
// foldMeasuredOn's regions carry three columns: the sentence a log line
// prints, the kind words that went into it, and the categories those words are
// the collapse of. All three are derived on a run that has node and a matching
// Unicode version at both ends, and the arm in
// TestHowWideTheNarrowerFoldIsAndWhatHoldsTheGap compares all three against
// what this build produces.
//
// That arm does not run here. It is gated on node's Unicode being 16.0 AND
// Go's being 15.0.0, which is right — the spans come from one and the naming
// from the other, so on a build where either moved the naming is expected to
// move with it — and it leaves the record unchecked by anything on a machine
// with no node, or a newer one. The sentence is fine there: a reader looks at
// "U+FF22..U+FF54 Latin letters" and can tell it from a wrong one. The columns
// are not. A kinds cell reading `marks` under a sentence that says "letters"
// looks like nothing at all, and it is what foldReached reads.
//
// So the record is held to its own internal shape, which needs no Unicode data
// and no child process:
//
//	U+2071..U+217C Common/Latin letters/modifier letters/numerals/symbols
//	└─ range ────┘ └ scripts ┘ └────────── kinds, "/"-joined ───────────┘
//	   cats: Ll Lm Lu Nl Sc So  ── foldWordsFor ──> letters, modifier letters,
//	                                                numerals, symbols
//
// The sentence's tail IS the kinds column joined, and the kinds column IS the
// categories mapped through foldRegionKinds. Those are the two contracts
// foldRegionsNamed writes the record under, and until now the only thing
// keeping a hand-edited cell honest was a build with node on it.
//
// This is the parse the previous session deleted, back as an ARM rather than
// as the way the words are obtained. The difference is the whole point: read
// forward, a parser of this file's own formatter is a second formatter nobody
// updates; read backward against a column that was written independently, it
// is a check.
func TestTheRecordedRegionNamesHoldTheirOwnColumns(t *testing.T) {
	// U+ and four to six hex digits, singly or as a span. Anchored at the
	// start because everything after it is the naming.
	where := regexp.MustCompile(`^U\+([0-9A-F]{4,6})(?:\.\.U\+([0-9A-F]{4,6}))? `)
	// The clause foldRegionsNamed adds for a region only PART of which Go's
	// tables know. It sits after the kinds, so it comes off before the tail is
	// compared with them.
	tail := regexp.MustCompile(` \(and \d+ of the \d+ unassigned in Go's Unicode [^)]+\)$`)
	// The two sentences that carry no kinds at all, as suffixes. Both are
	// whole-region findings rather than namings — see foldRegionsNamed — and
	// an empty kinds column is only allowed under one of them.
	noKinds := []string{
		"so newer than it",
		"holds none of the runs it was built from",
	}

	// Every category the mapping accounts for, either with a word or with a
	// written-down decision not to give it one. A cats cell holding anything
	// else is a two-letter code that came from neither list, which the
	// partition arm above would never have let through on a derived run.
	known := map[string]bool{}
	for _, k := range foldRegionKinds {
		known[k.category] = true
	}
	for name := range foldUnwordedCategories {
		known[name] = true
	}

	for _, c := range []struct {
		what    string
		regions []foldRegionName
	}{
		{"foldMeasuredOn.bearingRegions", foldMeasuredOn.bearingRegions},
		{"foldMeasuredOn.astralBearingRegions",
			foldMeasuredOn.astralBearingRegions},
	} {
		if len(c.regions) == 0 {
			t.Errorf("%s is empty.\n\n"+
				"The counts beside it — how many runs, how many regions — say "+
				"there are some, and this list is the only record of WHERE they "+
				"are. An empty one is a record that was never re-taken rather "+
				"than a population that vanished.", c.what)
			continue
		}
		for i, r := range c.regions {
			at := fmt.Sprintf("%s[%d]", c.what, i)
			m := where.FindStringSubmatch(r.text)
			if m == nil {
				t.Errorf("%s reads %q, which does not begin with a range.\n\n"+
					"Every sentence foldRegionsNamed produces starts with "+
					"U+XXXX or U+XXXX..U+YYYY and a space — it is the only part "+
					"of the naming that is about the code points rather than "+
					"about what Go's tables say of them, and it is what a reader "+
					"goes and looks up.", at, r.text)
				continue
			}
			lo, _ := strconv.ParseInt(m[1], 16, 32)
			hi := lo
			if m[2] != "" {
				hi, _ = strconv.ParseInt(m[2], 16, 32)
			}
			if hi < lo {
				t.Errorf("%s reads %q, whose range runs backwards.", at, r.text)
			}

			// A region with no kinds is one of two whole-region findings, and
			// the two columns have to be empty together: a categories cell
			// under a sentence that says Go's tables know nothing about this
			// range is a cell contradicting its own sentence.
			if len(r.kinds) == 0 {
				said := false
				for _, suffix := range noKinds {
					if strings.HasSuffix(r.text, suffix) {
						said = true
					}
				}
				if !said {
					t.Errorf("%s has no kind words and its sentence is %q.\n\n"+
						"foldRegionsNamed leaves the kinds empty in exactly two "+
						"cases and both of them SAY so: a region Go's tables "+
						"assign in neither script nor category (%q), and a "+
						"region holding none of the runs it was built from "+
						"(%q). A sentence that names scripts and categories "+
						"with an empty column beside it is the column having "+
						"been dropped in a re-take.",
						at, r.text, noKinds[0], noKinds[1])
				}
				if len(r.cats) > 0 {
					t.Errorf("%s has no kind words and the categories %s.\n\n"+
						"The words are what the categories print, so a region "+
						"with categories and no words is the two columns saying "+
						"opposite things about the same code points.",
						at, foldCategorySaid(r.cats))
				}
				continue
			}

			// The forward contract: the categories are what was seen, and the
			// words are those mapped through foldRegionKinds.
			if len(r.cats) == 0 {
				t.Errorf("%s prints the kind words %s and records no "+
					"categories.\n\n"+
					"Every word came from a category — that is the only way "+
					"foldRegionsNamed can produce one — so an empty second "+
					"column under a non-empty first is a record taken before "+
					"the categories were recorded, or one hand-edited since. "+
					"foldReached reads that column and would count these "+
					"entries as never reached.", at, strings.Join(r.kinds, ", "))
				continue
			}
			for _, cat := range r.cats {
				if !known[cat] {
					t.Errorf("%s records the category %q, which is in neither "+
						"foldRegionKinds nor foldUnwordedCategories.\n\n"+
						"Those two partition every two-letter category Go's "+
						"tables carry (see the partition arm above), so a third "+
						"answer is not a category at all — a typo in the "+
						"record, or a word that got into the wrong column.",
						at, cat)
				}
			}
			if sorted := slices.Clone(r.cats); !slices.IsSorted(sorted) ||
				len(slices.Compact(sorted)) != len(r.cats) {
				t.Errorf("%s records the categories %s, which are not sorted "+
					"and distinct.\n\n"+
					"foldWordList produces them that way so two runs over one "+
					"population cannot differ by map iteration. A record in "+
					"another order would fail the derived arm on a machine with "+
					"node for a reason that is not about the fold.",
					at, foldCategorySaid(r.cats))
			}
			if want := foldWordsFor(r.cats); !slices.Equal(r.kinds, want) {
				t.Errorf("%s records the kind words %s and its categories %s "+
					"print %s.\n\n"+
					"The words ARE the categories mapped through "+
					"foldRegionKinds — one traversal produces both in "+
					"foldRegionsNamed — so the two columns disagreeing is one "+
					"of them having been edited without the other. Which "+
					"matters in opposite directions: the words are what the "+
					"sentence shows a reader, and the categories are what "+
					"foldReached counts.",
					at, strings.Join(r.kinds, ", "),
					foldCategorySaid(r.cats), strings.Join(want, ", "))
			}

			// And the backward one: the sentence's tail is those same words.
			// This is where a kinds cell that disagrees with the text it sits
			// under is caught, which is the failure a reader cannot see.
			naming := tail.ReplaceAllString(r.text[len(m[0]):], "")
			joined := strings.Join(r.kinds, "/")
			if !strings.HasSuffix(naming, " "+joined) {
				t.Errorf("%s reads %q and its kind words are %s.\n\n"+
					"The sentence is the scripts \"/\"-joined, a space, and the "+
					"kinds \"/\"-joined — so it has to END with %q, and this one "+
					"does not. A cell that disagrees with its own sentence looks "+
					"like nothing to a reader, and on a machine with no node "+
					"(or a newer Unicode) the derived arm that would catch it "+
					"never runs.", at, r.text, strings.Join(r.kinds, ", "),
					joined)
				continue
			}
			if scripts := strings.TrimSuffix(naming, " "+joined); scripts == "" {
				t.Errorf("%s reads %q, which names no script before its kind "+
					"words.\n\n"+
					"A region with members Go's tables have a category for but "+
					"no script at all is not a case foldRegionsNamed produces: "+
					"the two are read together, and a code point in neither is "+
					"counted as unassigned instead.", at, r.text)
			}
		}
	}

	reach := foldReached()
	t.Logf("the %d recorded regions hold their own three columns with no "+
		"Unicode data and no child process: every sentence begins with a range, "+
		"ends with its own kind words \"/\"-joined, and those words are the "+
		"recorded categories mapped through foldRegionKinds. That is the half "+
		"the derived arm in TestHowWideTheNarrowerFoldIsAndWhatHoldsTheGap "+
		"cannot check on a machine with no node or a Unicode newer than %s — "+
		"and it is the half a reader cannot eyeball, because a kinds cell "+
		"contradicting its own sentence looks like nothing. Between them the "+
		"columns say the recorded populations have printed %d of "+
		"foldRegionKinds' %d entries; see foldReached.",
		len(foldMeasuredOn.bearingRegions)+
			len(foldMeasuredOn.astralBearingRegions),
		foldMeasuredOn.build.unicode, len(reach.cats),
		len(reach.cats)+len(reach.unusedCats))
}

// foldRegionsNamed is each region as a range and what Unicode says is in it.
//
// # The sentence this replaced, and why a block name is not what came back
//
// The eight BMP regions were named in prose — "the accented Latin letters,
// the phonetic modifiers, the superscripts and Roman numerals" — typed beside
// a list of eight ranges whose ORDER had to match and which nobody
// re-derived. That is the half of the census a reader actually uses, and it
// was the only half held by nothing: the ranges move with the Unicode version
// and the sentence does not, so the first NFKD change that re-draws a region
// leaves a log line naming the wrong thing, in a file whose whole subject is
// two implementations disagreeing about a fold.
//
// The obvious repair is the block each region falls in, and Go's standard
// library does not carry one. `unicode` has Categories, Scripts and
// Properties and no Blocks table at all, and adding golang.org/x/text to a
// repository with three dependencies for a log line's prose is not a trade
// worth making. So a region is named by what the stdlib does carry: the
// scripts its code points are in and the categories they belong to. That is
// less evocative than "the enclosed CJK squares" and it is derived, which the
// sentence was not.
//
// # And it reads a different Unicode version than the population
//
// The regions come from node's NFKD — Unicode 16.0 on the build this was
// taken on — and the scripts and categories come from Go's tables, which are
// 15.0.0 (see foldOwnMeasuredOn.goUnicode). A code point new in 16.0 has no
// script Go knows, and it comes back as "unassigned in Go's tables" rather
// than being dropped: the two versions disagreeing is a fact about the build
// and this is where it would show.
// foldRegionName is one region: what a log line prints for it, and the kind
// words that went into that sentence.
//
// # Why the words come back beside the string and are not read out of it
//
// foldReachedWords wants to know which of foldRegionKinds' words the recorded
// regions actually use, and it used to get them by SPLITTING the sentence: cut
// on "/", match the longest trailing word. That works because of how the two
// halves are joined below — scripts "/"-joined, a space, kinds "/"-joined — so
// the piece at the seam reads "Latin letters" and gives up "letters" to a
// suffix match. Two formatters, one of them a parser of the other's output,
// held together by nothing but both being in this file. A separator that
// became ", ", a script whose name ended in a kind word, a kind word that
// became two — each of them silently changes what the parser recovers, and
// what it recovers is a claim about which words this mapping has ever printed.
//
// So the words ride along. The record carries both columns and the arm holds
// both, which means the kinds beside a recorded region are as derived as the
// range is: a build where the naming changed fails on the column that changed.
type foldRegionName struct {
	// Exactly what the log line prints.
	text string
	// The words foldKindOf returned for this region's members, sorted and
	// deduplicated — the same set that went into text, in the same order.
	// Empty for the two regions that have no kinds to name: one whose code
	// points Go's tables do not know, and one that holds none of the runs it
	// was built from.
	kinds []string
	// And the categories those words are the collapse of, sorted and
	// deduplicated.
	//
	// # Why both columns, when one is a function of the other
	//
	// kinds IS foldWordsFor(cats) and the arm below asserts exactly that, so
	// the second column carries no information the first does not — in that
	// direction. The other direction is where the reading lives: seventeen
	// categories map onto six words, and a region that printed "letters" has
	// reached at least one of Lu, Ll, Lt and Lo without saying which.
	//
	// foldReached is a claim about which entries of the mapping these
	// populations have ever exercised, and at word granularity it can only
	// make half of it. "marks" unreached is sound — it means neither Mn nor Mc
	// nor Me has turned up. "letters" reached is not a claim about Lu, Ll, Lt
	// and Lo; it is a claim about their union, and three of the four can be
	// guesses hiding inside a word something else made true.
	//
	// So the words stay, because they are what the sentence prints and what a
	// reader compares against it, and the categories come with them, because
	// they are what the mapping is written at.
	cats []string
}

// foldRegionTexts is the printable half of a list of regions.
func foldRegionTexts(named []foldRegionName) []string {
	out := make([]string, 0, len(named))
	for _, n := range named {
		out = append(out, n.text)
	}
	return out
}

// foldRegionLines is a list of regions as a failure prints them: the sentence,
// and the words under it that a reader cannot see in the sentence.
func foldRegionLines(named []foldRegionName) []string {
	out := make([]string, 0, len(named))
	for _, n := range named {
		said := "(no kind words)"
		if len(n.kinds) > 0 {
			said = strings.Join(n.kinds, ", ")
		}
		cats := "(no categories)"
		if len(n.cats) > 0 {
			cats = strings.Join(n.cats, " ")
		}
		out = append(out, n.text+"   [kinds: "+said+"; categories: "+cats+"]")
	}
	return out
}

// foldRegionSame is two readings of one region agreeing in all three columns.
//
// The third is not redundant with the second here, even though kinds is a
// function of cats: a build where Lo stopped turning up and Lu started would
// print the same word, hold the same sentence, and be a different population.
// That is the finding the category column was added to be able to have.
func foldRegionSame(a, b foldRegionName) bool {
	return a.text == b.text && slices.Equal(a.kinds, b.kinds) &&
		slices.Equal(a.cats, b.cats)
}

func foldRegionsNamed(clusters, runs [][2]rune) []foldRegionName {
	out := make([]foldRegionName, 0, len(clusters))
	for _, c := range clusters {
		where := fmt.Sprintf("U+%04X..U+%04X", c[0], c[1])
		if c[0] == c[1] {
			where = fmt.Sprintf("U+%04X", c[0])
		}
		// The members, which are the runs inside this region and not the
		// whole span: a region is runs merged across gaps of up to
		// foldBearingGap, and the code points in those gaps are not in the
		// population being described.
		// Categories rather than words, with the words derived from them
		// below. A region that reaches Lu and one that reaches Lo both print
		// "letters", and which of them it was is the reading foldReached
		// could not take while this collected the word. See foldRegionName.
		scripts, cats := map[string]bool{}, map[string]bool{}
		unknown := 0
		members := 0
		for _, run := range runs {
			if run[0] < c[0] || run[1] > c[1] {
				continue
			}
			for cp := run[0]; cp <= run[1]; cp++ {
				members++
				script, cat := foldScriptOf(cp), foldCategoryOf(cp)
				if script == "" && cat == "" {
					unknown++
					continue
				}
				if script != "" {
					scripts[script] = true
				}
				if cat != "" {
					cats[cat] = true
				}
			}
		}
		// A region Go's tables have nothing to say about at all. That is not
		// a gap in this reading, it is the reading working: the population
		// comes from node's NFKD and the naming from Go's stdlib, and a block
		// that exists in one Unicode version and not the other lands here.
		// A region with no members at all is not a fact about Unicode, it is
		// the runs and the clusters having come apart — every cluster is
		// built out of runs, so one holding none of them is a caller passing
		// the wrong plane's population. Said as itself rather than falling
		// into the version arm below, where it would read as a whole region
		// of characters newer than Go's tables.
		if members == 0 {
			out = append(out, foldRegionName{
				text: where + " holds none of the runs it was built from"})
			continue
		}
		if unknown == members {
			out = append(out, foldRegionName{text: fmt.Sprintf("%s assigned in "+
				"neither script nor category by Go's Unicode %s, so newer than it",
				where, unicode.Version)})
			continue
		}
		// One traversal for both columns and the sentence: catList is what the
		// region records, kinds is that mapped through foldRegionKinds, and
		// the sentence is those same words joined. Three readings of one map
		// would be three chances for them to say different things.
		catList := foldWordList(cats)
		kinds := foldWordsFor(catList)
		said := foldSortedWords(scripts) + " " + strings.Join(kinds, "/")
		if unknown > 0 {
			said += fmt.Sprintf(" (and %d of the %d unassigned in Go's Unicode %s)",
				unknown, members, unicode.Version)
		}
		// The same set the sentence was built from, kept rather than left to be
		// recovered from it. See foldRegionName.
		out = append(out, foldRegionName{
			text:  where + " " + said,
			kinds: kinds,
			cats:  catList,
		})
	}
	return out
}

// foldScriptOf is the Unicode script Go's tables put a code point in, or empty
// where they have none for it.
func foldScriptOf(cp rune) string {
	for name, table := range unicode.Scripts {
		if unicode.Is(table, cp) {
			return name
		}
	}
	return ""
}

// foldCategoryOf is the two-letter general category Go's tables put a code
// point in, or empty for one they have not assigned.
//
// # Why this is what a region records, and the word is derived from it
//
// A region used to collect WORDS. Seventeen categories map onto six words, so
// a region that printed "letters" had reached at least one of Lu, Ll, Lt and
// Lo and the reading could not say which — and the whole point of
// foldReached's unreached list is to name the entries the mapping is a guess
// about. "marks" being unreached is a sound claim about Mn, Mc and Me
// together; "letters" being reached is a claim about none of its four. The
// half that could not be said was the half that mattered.
//
// So the category is what comes back and the word is a function of it. The
// sentence a region prints is unchanged — foldWordsFor maps the categories
// through foldRegionKinds — and the column beside it is now at the granularity
// the mapping is written at.
//
// # The order of the two loops is load-bearing
//
// unicode.Categories holds "LC" — Go's union of Lu, Ll and Lt — and it is two
// letters long, so the fall-through would return it for any cased letter that
// reached it. Nothing does: every category in foldRegionKinds is tried first,
// and Lu, Ll and Lt are all in there. That is the same fact foldUnwordedCategories
// gives as LC's whole reason for needing no word, asserted from the other end
// by TestTheRegionKindsAccountForEveryCategoryGoHas.
func foldCategoryOf(cp rune) string {
	for _, k := range foldRegionKinds {
		if table, ok := unicode.Categories[k.category]; ok &&
			unicode.Is(table, cp) {
			return k.category
		}
	}
	// A category nobody gave a word to arrives as its own two letters, which
	// a reader can look up. Cn — unassigned — is the one that is not a
	// category so much as an absence, and it is reported as one above.
	for name, table := range unicode.Categories {
		if len(name) == 2 && name != "Cn" && unicode.Is(table, cp) {
			return name
		}
	}
	return ""
}

// foldWordFor is the word a category prints, which for one nobody gave a word
// to is its own two letters. See foldRegionKinds and foldUnwordedCategories,
// which partition Go's categories between those two answers.
func foldWordFor(category string) string {
	for _, k := range foldRegionKinds {
		if k.category == category {
			return k.word
		}
	}
	return category
}

// foldWordsFor is the words a set of categories prints, deduplicated and in
// the fixed order a sentence lists them.
//
// This is the whole of the collapse the second column exists to undo: Lu, Ll,
// Lt and Lo arrive here and "letters" leaves, and nothing downstream of it can
// tell which of the four was there.
func foldWordsFor(cats []string) []string {
	set := make(map[string]bool, len(cats))
	for _, c := range cats {
		set[foldWordFor(c)] = true
	}
	return foldWordList(set)
}

// foldKindOf is the word for a code point's general category, or empty for a
// code point Go's tables have not assigned one. See foldRegionKinds.
func foldKindOf(cp rune) string {
	if cat := foldCategoryOf(cp); cat != "" {
		return foldWordFor(cat)
	}
	return ""
}

// foldSortedWords is a set of words as a sentence lists them, in a fixed
// order so a log line does not depend on Go's map iteration.
func foldSortedWords(set map[string]bool) string {
	return strings.Join(foldWordList(set), "/")
}

// foldWordList is that same set as a list, which is what a region carries
// beside its sentence.
//
// The two are one function because the sentence and the column have to be the
// same words in the same order — a region whose text says one thing and whose
// kinds say another is the drift foldRegionName exists to make impossible, and
// building them from two different traversals of the map is how it would get
// back in.
func foldWordList(set map[string]bool) []string {
	words := make([]string, 0, len(set))
	for w := range set {
		words = append(words, w)
	}
	sort.Strings(words)
	return words
}

func TestHowWideTheNarrowerFoldIsAndWhatHoldsTheGap(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("no node on this machine, so browser.mjs's own fold cannot be run " +
			"and the width of the gap between the two cannot be measured. The two " +
			"folds' character sets are still compared above, out of the regex.")
	}
	src, err := os.ReadFile("browser.mjs")
	if err != nil {
		t.Fatalf("browser.mjs cannot be read: %v", err)
	}
	decl := regexp.MustCompile(`(?s)const INK_FOLD_IGNORABLE =.*?;\n`).Find(src)
	fn := regexp.MustCompile(`(?s)\nfunction inkFold\(text\) \{.*?\n\}\n`).Find(src)
	if decl == nil || fn == nil {
		t.Fatalf("browser.mjs no longer declares INK_FOLD_IGNORABLE and inkFold in " +
			"the shape this test lifts them out in. Find what folds a string there " +
			"and measure against that instead of deleting this — the gap it reports " +
			"is what the printable-ASCII arm is holding shut.")
	}
	// Only the code points the note's fold actually changes come back: the
	// answer for the rest is the character itself, gen.go's fold is asked of
	// them below anyway, and a plane of unchanged strings is half a megabyte of
	// JSON saying nothing. Surrogates are skipped on both sides — Go turns a
	// lone one into U+FFFD and JavaScript does not, so the pair would be
	// comparing two different characters.
	//
	// Every plane, not only the first. See foldMeasuredOn's astral counts for
	// what that cost when it was finally measured — 128ms and 30KB — and for
	// what is up there that the BMP does not have.
	script := filepath.Join(t.TempDir(), "gap.mjs")
	// The build comes back with the rows, out of the same process that did the
	// folding. See foldMeasuredOn: every number below is NFKD as the ICU
	// compiled into THIS node implements it, and a census with no build beside
	// it is a measurement of an unnamed thing. Read from process.versions
	// rather than asked of the shell, so the two cannot be about different
	// binaries.
	if err := os.WriteFile(script, []byte(string(decl)+string(fn)+`
const out = [];
for (let cp = 0; cp <= 0x10FFFF; cp++) {
    if (cp >= 0xD800 && cp <= 0xDFFF) continue;
    const ch = String.fromCodePoint(cp);
    const folded = inkFold(ch);
    if (folded !== ch) out.push([cp, folded]);
}
console.log(JSON.stringify({
    unicode: process.versions.unicode || "", icu: process.versions.icu || "",
    node: process.versions.node, rows: out,
}));
`), 0o644); err != nil {
		t.Fatalf("the lifted fold will not write: %v", err)
	}
	out, err := exec.Command(node, script).Output()
	if err != nil {
		t.Fatalf("node will not run browser.mjs's own fold: %v", err)
	}
	var answer struct {
		Unicode, ICU, Node string
		Rows               [][]json.RawMessage
	}
	if err := json.Unmarshal(out, &answer); err != nil {
		t.Fatalf("browser.mjs's fold returned something this test cannot read: %v", err)
	}
	rows := answer.Rows
	// What this run is, in the shape the record is written in.
	now := foldBuild{unicode: answer.Unicode, icu: answer.ICU, node: answer.Node}
	if now.unicode == "" {
		// A node with no ICU folds nothing the table does not, which would make
		// the whole census a reading about a build that cannot answer the
		// question. Named rather than compared against a record it cannot be
		// compared with.
		t.Skipf("this node reports no Unicode version (process.versions.unicode is "+
			"empty), which is a build without full ICU — String.prototype.normalize "+
			"there is not the NFKD the census below is a measurement of, so the "+
			"numbers would be about a fold nobody ships. node %s, icu %q.",
			now.node, now.icu)
	}
	if len(rows) == 0 {
		t.Fatalf("browser.mjs's fold changed nothing on any of Unicode's 1114112 " +
			"code points, which is this test having lifted something that is not a " +
			"fold rather than a fold that does nothing.")
	}

	// The letters the seeds are spelled with, after their own fold — the
	// alphabet a gap character has to reach to be able to complete a pair.
	seedLetters := map[rune]bool{}
	for _, seed := range inkLigatureSeeds {
		for _, r := range inkGlyphFold(seed) {
			seedLetters[r] = true
		}
	}
	theirs := map[rune]string{}
	// The census, kept per plane-set rather than summed. The BMP's four
	// numbers are what every other reading in this file is about — the
	// fixtures are printable ASCII and the presentation forms are at U+FB00 —
	// and the astral four are the bound that used to be a place the walk
	// stopped. Two different facts, counted apart, the way foldOwnMeasuredOn
	// counts gen.go's own width.
	type foldSide struct {
		changes, agreed, bearing int
		bearingAt                rune
		// The seed-bearing code points as contiguous runs, which is the shape
		// of that population rather than a count of it. See the arm that reads
		// them: 257 characters in a handful of runs is two or three blocks
		// somebody can look up, and 257 scattered singletons would be a
		// different fact about NFKD entirely.
		runs   [][2]rune
		lastAt rune
	}
	bmp := foldSide{bearingAt: -1, lastAt: -2}
	astral := foldSide{bearingAt: -1, lastAt: -2}
	// A character both inside printable ASCII and in the gap, which is the one
	// thing the second arm cannot catch. First one found; there is nothing to
	// be gained from a list of a fault that should have no members. Only the
	// BMP can hold one, which is why it is not a per-side count: 0x20..0x7e is
	// the whole of the arm's range.
	ascii, asciiCount := rune(-1), 0
	for _, row := range rows {
		var cp rune
		var their string
		if err := json.Unmarshal(row[0], &cp); err != nil {
			t.Fatalf("a code point came back unreadable: %v", err)
		}
		if err := json.Unmarshal(row[1], &their); err != nil {
			t.Fatalf("a folded answer came back unreadable: %v", err)
		}
		theirs[cp] = their
		side := &bmp
		if cp > 0xFFFF {
			side = &astral
		}
		side.changes++
		mine := inkGlyphFold(string(cp))
		if mine == their {
			side.agreed++
			continue
		}
		if cp >= 0x20 && cp <= 0x7e {
			asciiCount++
			if ascii < 0 {
				ascii = cp
			}
		}
		for _, r := range their {
			if seedLetters[r] && !strings.ContainsRune(mine, r) {
				side.bearing++
				if side.bearingAt < 0 {
					side.bearingAt = cp
				}
				// The rows come back in ascending order, so a run extends
				// while the code points are consecutive and starts again when
				// they are not.
				if cp == side.lastAt+1 && len(side.runs) > 0 {
					side.runs[len(side.runs)-1][1] = cp
				} else {
					side.runs = append(side.runs, [2]rune{cp, cp})
				}
				side.lastAt = cp
				break
			}
		}
	}
	agreed, bearing := bmp.agreed, bmp.bearing
	bearingExample := bmp.bearingAt
	gap := bmp.changes - bmp.agreed
	astralGap := astral.changes - astral.agreed
	// And where the seed-bearing characters above the BMP actually are. See
	// foldClustersOf: the shape of that population is what a count of it
	// cannot say, and it is read here so both the arms and the log line have
	// it.
	astralClusters, insideGap, betweenGap := foldClustersOf(astral.runs)
	// And the same for the BMP's, which is the population the printable-ASCII
	// arm is actually holding shut on the plane every fixture in this
	// repository lives on. Its runs were being computed by the loop above —
	// the walk is written once and both sides go through it — and thrown away.
	bmpClusters, bmpInsideGap, bmpBetweenGap := foldClustersOf(bmp.runs)

	// The edge the printable-ASCII arm is the whole of.
	if ascii >= 0 {
		t.Errorf("U+%04X (%q) is printable ASCII, and the two folds answer "+
			"differently about it: browser.mjs's says %q and gen.go's says %q.\n\n"+
			"inkGlyphPerCharacter has two arms and this character is outside both. "+
			"The pair arm asks its question in gen.go's fold, which finds nothing "+
			"here; the second arm refuses everything outside 0x20..0x7e, and this is "+
			"inside it. A fixture holding this character is accepted, and "+
			"inkLigatureNote — which folds with NFKD — would find a pair in the very "+
			"string that was let through. That is the gap between the two folds "+
			"reaching the one population nothing is holding.",
			ascii, string(ascii), theirs[ascii], inkGlyphFold(string(ascii)))
	}

	// And the other direction, which would be the refusal wider than the note.
	//
	// Over every plane, now that every plane came back. This is the edge the
	// astral half of the census buys that the gen.go-only walk could not:
	// foldOwnMeasuredOn asserts that inkGlyphFold does nothing above the BMP
	// but lowercase, and that is a fact about one side. Whether the OTHER side
	// lowercases the same characters is a fact about node's Unicode, and until
	// this walk went past U+FFFF nothing compared them. A case pair Go has and
	// node does not is gen.go reaching further than NFKD up here, which is the
	// direction this file does not allow.
	wider := []rune{}
	for cp := rune(0); cp <= 0x10FFFF && len(wider) < 4; cp++ {
		if cp >= 0xD800 && cp <= 0xDFFF {
			continue
		}
		if _, changed := theirs[cp]; changed {
			continue
		}
		if inkGlyphFold(string(cp)) != string(cp) {
			wider = append(wider, cp)
		}
	}
	if len(wider) > 0 {
		names := make([]string, 0, len(wider))
		for _, cp := range wider {
			names = append(names, fmt.Sprintf("U+%04X %q→%q", cp, string(cp),
				inkGlyphFold(string(cp))))
		}
		t.Errorf("gen.go's fold changes %s and browser.mjs's leaves them alone.\n\n"+
			"The narrow fold is allowed to reach less than NFKD and says so — that "+
			"is the whole argument for a seven-character table. It is not allowed to "+
			"reach MORE: a string refused here for holding a pair is a string "+
			"inkLigatureNote would never report a pair in, so the fixture author is "+
			"sent to find a ligature that the check at the other end of the pipeline "+
			"does not believe is there.", strings.Join(names, ", "))
	}

	// And the lean itself, on one string, end to end.
	//
	// A gap character that completes a seed is what the pair arm cannot see:
	// the note's fold turns it into the seed's own letter and gen.go's leaves
	// it. The refusal still holds — and it holds by the OTHER arm, which is
	// what "the printable-ASCII arm is what makes that safe" means when it is
	// asked of a string instead of asserted in a comment.
	//
	// Asked once per plane-set, because the two are not the same question. The
	// BMP's witness is built out of an accented letter and the astral one out
	// of a decomposing letter FORM — U+1CCD7 folds to "b" and U+1D41F
	// MATHEMATICAL BOLD SMALL F to "f" — and the second population did not
	// exist as far as this file was concerned until the walk went past U+FFFF. What holds them shut is
	// the same arm in both cases and it is holding for two different reasons:
	// on the BMP because the character is outside 0x20..0x7e, and above it
	// because nothing up there can be inside a range of two ASCII bytes at all.
	witnessFor := func(what string, bearingAt rune) {
		if bearingAt < 0 {
			return
		}
		witness := ""
		for _, seed := range inkLigatureSeeds {
			folded := inkGlyphFold(seed)
			if len(folded) < 2 {
				continue
			}
			head := folded[:len(folded)-1]
			if strings.Contains(head+theirs[bearingAt], folded) {
				witness = head + string(bearingAt)
				break
			}
		}
		if witness == "" {
			return
		}
		held := ""
		for _, seed := range inkLigatureSeeds {
			if strings.Contains(inkGlyphFold(witness), inkGlyphFold(seed)) {
				held = seed
			}
		}
		if held != "" {
			t.Errorf("gen.go's fold finds %q in %q after all, so this string is "+
				"not the witness this arm is about (%s).", held, witness, what)
		}
		why := inkGlyphPerCharacter(witness)
		switch {
		case why == "":
			t.Errorf("inkGlyphPerCharacter accepts %q, built from %s.\n\n"+
				"browser.mjs's fold turns U+%04X into %q, so that string holds a "+
				"pair for inkLigatureNote and holds none for the refusal — the "+
				"gap between the two folds, on one string. What kept it out was "+
				"the printable-ASCII arm, and it has stopped: a fixture can now "+
				"carry a pair past this refusal and arrive as the font "+
				"substitution it exists to prevent.",
				witness, what, bearingAt, theirs[bearingAt])
		case !strings.Contains(why, "printable ASCII"):
			t.Errorf("inkGlyphPerCharacter refuses %q (built from %s) with %q.\n\n"+
				"That string holds a pair only under NFKD, and gen.go's fold is "+
				"narrower there by design — so the arm that has to catch it is "+
				"the printable-ASCII one. A different sentence means the pair arm "+
				"has widened, which is a better refusal and makes this reading "+
				"stale: re-measure the gap and say what is holding it now.",
				witness, what, why)
		}
	}
	witnessFor("the BMP's gap", bmp.bearingAt)
	witnessFor("the gap above the BMP", astral.bearingAt)

	// And the population the arm above is asked over, which had no floor at all.
	//
	// The whole of the previous paragraph runs inside `if bearingExample >= 0`.
	// A census that found no seed-bearing character in the gap would skip it in
	// silence — no witness built, no refusal asked, and a log line printing
	// U+FFFFFFFF for a rune nobody found. That is the one arm here that reads a
	// string end to end rather than a code point, and it going quiet is exactly
	// what "the printable-ASCII arm is what makes the rest safe" would stop
	// being evidence for.
	// And the same for the plane-set above, where the population is larger and
	// newer: the mathematical alphanumerics decompose to bare letters, so a
	// gap character up there completes a seed as readily as one on the BMP and
	// the arm that asks whether the refusal still holds has to be asked of it
	// too. Zero here would be NFKD having stopped decomposing them, which is a
	// build change rather than a fold change — and the note says so.
	if astral.bearing == 0 {
		t.Errorf("not one of the %d code points in the gap ABOVE the BMP is turned "+
			"by NFKD into a letter a seed is spelled with.\n\n"+
			"U+1D41F MATHEMATICAL BOLD SMALL F decomposes to \"f\", and the block it "+
			"is in is the reason this half of the census is worth taking: the "+
			"seed-bearing population up there is hundreds of characters, none of "+
			"which gen.go's fold touches, and what holds every one of them out of a "+
			"fixture is the printable-ASCII arm. With none of them the astral witness "+
			"is never built and that reading passes by being skipped. The seeds fold "+
			"to %v.%s",
			astralGap, seedLetters, foldBuildNote(now))
	}

	if bearing == 0 {
		t.Errorf("not one of the %d code points in the gap is turned by NFKD into a "+
			"letter a seed is spelled with.\n\n"+
			"That population is the whole reason this gap is interesting: a character "+
			"in it completes a pair for inkLigatureNote and completes none for the "+
			"refusal, which is the two ends of the pipeline looking at one string and "+
			"disagreeing about whether there is a ligature in it. With none of them "+
			"the witness below is never built and the arm that asks whether the "+
			"printable-ASCII refusal is still holding never runs — it passes by being "+
			"skipped. The seeds fold to %v.%s",
			gap, seedLetters, foldBuildNote(now))
	}

	// And the census itself, against the one recorded for this build.
	//
	// See foldMeasuredOn. NFKD is the ICU's data, so on a matching Unicode
	// version these four numbers are a function of gen.go's table alone: a
	// difference is that table having changed, which is a fold reaching further
	// or less far than the one this file's edges were reasoned about. On a
	// different Unicode version the numbers are expected to move and nothing is
	// asserted about them — the two edges above hold on every build, and they
	// are what the refusal actually rests on.
	//
	// And the gap that leaves is closed elsewhere rather than hedged about
	// here. See foldOwnMeasuredOn: the width of gen.go's fold splits into what
	// the ignorable predicate drops, what the table rewrites and what ToLower
	// changes, the first two have no ICU anywhere in them, and both are held on
	// every machine — including one with no node on it at all, where this whole
	// test skips.
	if now.unicode == foldMeasuredOn.build.unicode {
		for _, c := range []struct {
			what      string
			got, want int
		}{
			{"the code points browser.mjs's fold changes", bmp.changes,
				foldMeasuredOn.changes},
			{"the ones gen.go's fold agrees with", agreed, foldMeasuredOn.agreed},
			{"the gap between them", gap, foldMeasuredOn.gap},
			{"the part of the gap that reaches a seed's own letters", bearing,
				foldMeasuredOn.bearing},
			{"the runs of consecutive code points THAT falls in",
				len(bmp.runs), foldMeasuredOn.bearingRuns},
			{"the regions those runs make", len(bmpClusters),
				foldMeasuredOn.bearingClusters},
			{"the code points browser.mjs's fold changes above the BMP",
				astral.changes, foldMeasuredOn.astralChanges},
			{"the ones gen.go's fold agrees with above the BMP", astral.agreed,
				foldMeasuredOn.astralAgreed},
			{"the gap between them above the BMP", astralGap,
				foldMeasuredOn.astralGap},
			{"the part of THAT gap that reaches a seed's own letters",
				astral.bearing, foldMeasuredOn.astralBearing},
			{"the runs of consecutive code points it falls in",
				len(astral.runs), foldMeasuredOn.astralBearingRuns},
			{"the regions those runs make", len(astralClusters),
				foldMeasuredOn.astralBearingClusters},
		} {
			if c.got == c.want {
				continue
			}
			t.Errorf("%s comes to %d on this run and foldMeasuredOn records %d, over "+
				"the same Unicode %s.\n\n"+
				"NFKD is that version's data and it has not moved, so this is gen.go's "+
				"own table having changed width — inkGlyphFold now reaches somewhere "+
				"it did not, or has stopped reaching somewhere it did. That is the "+
				"thing this whole file is about and it is invisible from either edge: "+
				"both of them still pass over a fold that quietly narrowed, because "+
				"narrower is the direction they allow. Re-read what changed in the "+
				"table, and if the new width is intended, re-take foldMeasuredOn from "+
				"the census in the log line. This run: %d changed, %d agreed, %d gap, "+
				"%d seed-bearing, %d inside printable ASCII.",
				c.what, c.got, c.want, now.unicode,
				len(rows), agreed, gap, bearing, asciiCount)
		}
	}

	// # And where those regions are, which the counts above cannot see
	//
	// Two Unicode versions meet here, so the gate is both of them: the spans
	// come from node's NFKD and the words come from Go's script and category
	// tables. On a build where either has moved the naming is expected to move
	// with it and nothing is asserted — the two counts above are what hold on
	// every build, and this is what holds the ranges those counts are silent
	// about. See foldMeasuredOn.bearingRegions.
	if now.unicode == foldMeasuredOn.build.unicode &&
		unicode.Version == foldOwnMeasuredOn.goUnicode {
		for _, c := range []struct {
			what       string
			got, want  []foldRegionName
			plane      string
			runs, regs int
		}{
			{"the BMP", foldRegionsNamed(bmpClusters, bmp.runs),
				foldMeasuredOn.bearingRegions, "the BMP",
				len(bmp.runs), len(bmpClusters)},
			{"above the BMP", foldRegionsNamed(astralClusters, astral.runs),
				foldMeasuredOn.astralBearingRegions, "above the BMP",
				len(astral.runs), len(astralClusters)},
		} {
			// Both columns, because both are derived and each fails for its own
			// reason: the sentence moves when the characters move, and the kind
			// words move when Go's categories do. A run where only the second
			// changed is a naming that has started saying something else about
			// the same range — see foldRegionName.
			if slices.EqualFunc(c.got, c.want, foldRegionSame) {
				continue
			}
			t.Errorf("the seed-bearing regions %s come to\n\t%s\nand "+
				"foldMeasuredOn records\n\t%s\n\n"+
				"Both Unicode versions match their records — node's %s for the "+
				"population and Go's %s for the naming — so neither table has "+
				"moved and this is where those characters ARE having changed. The "+
				"run and region counts (%d and %d) are asserted above and they "+
				"cannot see this: the same number of runs falling into the same "+
				"number of groups says nothing about which code points they hold, "+
				"and the sentence that used to name them was typed rather than "+
				"derived, so it could not either.\n\n"+
				"If gen.go's table changed width, that is the finding and the arms "+
				"above will have said so too. If it did not, the fold is reaching a "+
				"different population for the same count — re-read what moved "+
				"before re-taking the record.",
				c.what, strings.Join(foldRegionLines(c.got), "\n\t"),
				strings.Join(foldRegionLines(c.want), "\n\t"), now.unicode,
				unicode.Version, c.runs, c.regs)
		}
	}

	// The astral half of the sentence, which is a different fact and is said
	// as one. Its seed-bearing example is the interesting number: gen.go's
	// fold does nothing above the BMP but lowercase (foldOwnMeasuredOn asserts
	// it), so every one of these is a character that completes a pair for
	// inkLigatureNote and completes none for the refusal.
	astralSaid := "and no code point above the BMP at all, which is this walk " +
		"having stopped at U+FFFF again"
	if astral.changes > 0 {
		example := ""
		if astral.bearingAt >= 0 {
			example = fmt.Sprintf(" (e.g. U+%04X %q→%q, where gen.go says %q)",
				astral.bearingAt, string(astral.bearingAt), theirs[astral.bearingAt],
				inkGlyphFold(string(astral.bearingAt)))
		}
		// Where those seed-bearing characters actually are, which is the half
		// a count cannot say. See foldClustersOf.
		where := foldRegionTexts(foldRegionsNamed(astralClusters, astral.runs))
		astralSaid = fmt.Sprintf("and %d above the BMP, where it agrees on %d and "+
			"is narrower on %d, %d of which reach a seed's own letters%s — a "+
			"population that is not the BMP's made smaller: NFKD decomposes the "+
			"mathematical alphanumerics to bare letters, and nothing up there can be "+
			"inside 0x20..0x7e, so the arm holding all of them out is the same one "+
			"and it is holding for a different reason. Those %d sit in %d runs "+
			"making %d regions — %s — with the widest gap "+
			"inside a region %d code points and the narrowest between two %d, so "+
			"the %d this file separates them by is naming where to look rather than "+
			"choosing the answer",
			astral.changes, astral.agreed, astralGap, astral.bearing, example,
			astral.bearing, len(astral.runs), len(astralClusters),
			strings.Join(where, ", "), insideGap, betweenGap, foldBearingGap)
	}
	// And where the BMP's seed-bearing characters are, which is the half a
	// count cannot say. Same reading as the astral half's, over the population
	// that actually matters here.
	bmpWhere := foldRegionTexts(foldRegionsNamed(bmpClusters, bmp.runs))
	t.Logf("browser.mjs's fold changes %d of the BMP's code points; gen.go's "+
		"agrees on %d and is narrower on %d, %d of which NFKD turns into a letter "+
		"a seed is spelled with (e.g. U+%04X %q→%q, where gen.go says %q) — and %d of "+
		"it inside printable ASCII, which is the whole of what "+
		"inkGlyphPerCharacter's second arm refuses on and therefore the whole of "+
		"what is holding this gap shut. Those %d sit in %d runs making %d regions, "+
		"each named by what Go's Unicode %s says is in it rather than by a "+
		"sentence nobody re-derives — %s. "+
		"The widest gap inside a region is %d code points and the narrowest "+
		"between two is %d, so the %d this file separates them by is naming where "+
		"to look rather than choosing the answer — and this population is not the "+
		"astral one made larger: up there it is three blocks of decorated "+
		"alphabets and here NFKD reaches a seed's letters from every direction the "+
		"plane has. %s. The whole walk is %d code points "+
		"through node and cost less than a fifth of a second, which is what the "+
		"bound at U+FFFF was worth. Measured on Unicode %s (ICU %s, node %s), "+
		"which is %s the record was taken on%s",
		bmp.changes, agreed, gap, bearing, bearingExample, string(bearingExample),
		theirs[bearingExample], inkGlyphFold(string(bearingExample)), asciiCount,
		bearing, len(bmp.runs), len(bmpClusters), unicode.Version,
		strings.Join(bmpWhere, ", "),
		bmpInsideGap, bmpBetweenGap, foldBearingGap,
		astralSaid, 0x110000,
		now.unicode, now.icu, now.node,
		map[bool]string{true: "the build", false: "NOT the build"}[now.unicode == foldMeasuredOn.build.unicode],
		foldBuildNote(now))
}
