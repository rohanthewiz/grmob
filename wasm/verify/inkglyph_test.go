package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
}{
	build:   foldBuild{unicode: "16.0", icu: "76.1", node: "22.12.0"},
	changes: 15802,
	agreed:  748,
	gap:     15054,
	bearing: 299,

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
// 256. The census below finds them in three regions, and the numbers are not
// close: the widest gap INSIDE a region is 67 code points and the narrowest
// gap BETWEEN two of them is 1816. Anything from 68 to 1816 gives the same
// three, so the constant is not doing work — it is naming where a reader
// should look, and the log line prints both gaps so a build that moved them
// says so rather than quietly re-drawing the regions.
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
		where := make([]string, 0, len(astralClusters))
		for _, c := range astralClusters {
			where = append(where, fmt.Sprintf("U+%04X..U+%04X", c[0], c[1]))
		}
		astralSaid = fmt.Sprintf("and %d above the BMP, where it agrees on %d and "+
			"is narrower on %d, %d of which reach a seed's own letters%s — a "+
			"population that is not the BMP's made smaller: NFKD decomposes the "+
			"mathematical alphanumerics to bare letters, and nothing up there can be "+
			"inside 0x20..0x7e, so the arm holding all of them out is the same one "+
			"and it is holding for a different reason. Those %d sit in %d runs "+
			"making %d regions — %s, which is the outlined letters, the "+
			"mathematical alphanumerics and the enclosed ones — with the widest gap "+
			"inside a region %d code points and the narrowest between two %d, so "+
			"the %d this file separates them by is naming where to look rather than "+
			"choosing the answer",
			astral.changes, astral.agreed, astralGap, astral.bearing, example,
			astral.bearing, len(astral.runs), len(astralClusters),
			strings.Join(where, ", "), insideGap, betweenGap, foldBearingGap)
	}
	t.Logf("browser.mjs's fold changes %d of the BMP's code points; gen.go's "+
		"agrees on %d and is narrower on %d, %d of which NFKD turns into a letter "+
		"a seed is spelled with (e.g. U+%04X %q→%q, where gen.go says %q) — and %d of "+
		"it inside printable ASCII, which is the whole of what "+
		"inkGlyphPerCharacter's second arm refuses on and therefore the whole of "+
		"what is holding this gap shut. %s. The whole walk is %d code points "+
		"through node and cost less than a fifth of a second, which is what the "+
		"bound at U+FFFF was worth. Measured on Unicode %s (ICU %s, node %s), "+
		"which is %s the record was taken on%s",
		bmp.changes, agreed, gap, bearing, bearingExample, string(bearingExample),
		theirs[bearingExample], inkGlyphFold(string(bearingExample)), asciiCount,
		astralSaid, 0x110000,
		now.unicode, now.icu, now.node,
		map[bool]string{true: "the build", false: "NOT the build"}[now.unicode == foldMeasuredOn.build.unicode],
		foldBuildNote(now))
}
