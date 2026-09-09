package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
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
	// Every code point of the plane these characters live in, both ways. The
	// interesting failures are at the edges of a range, and enumerating is
	// cheaper than arguing about which edges.
	for r := rune(0); r <= 0xFFFF; r++ {
		there := false
		for _, s := range spans {
			if r >= s.lo && r <= s.hi {
				there = true
				break
			}
		}
		if here := inkGlyphIgnorable(r); here != there {
			t.Fatalf("U+%04X is dropped by %s and kept by %s.\n\n"+
				"These two folds are asked the same question at the two ends of one "+
				"pipeline: gen.go refuses a fixture that holds a ligature pair, and "+
				"browser.mjs's inkLigatureNote says which pair a face joined when one "+
				"got through. A character one drops and the other keeps is the two of "+
				"them looking at different strings, and what comes out is a confident "+
				"sentence naming a pair the reader cannot see in the string in front "+
				"of them.", r,
				map[bool]string{true: "inkGlyphIgnorable", false: "INK_FOLD_IGNORABLE"}[here],
				map[bool]string{true: "inkGlyphIgnorable", false: "INK_FOLD_IGNORABLE"}[!here])
		}
	}
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
