package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The canary's two new readings, run.
//
// # A reading whose evidence is a machine's mood
//
// inkCanaryAgreement holds the probe's NAME against the advances' DISTANCE, and
// it declines on two page states: a probe that named more than one face for
// "Ag0", and a probe that came back with none. Both are counted now — see that
// function's `multi` and `unread` — because a comparison going quiet used to
// arrive in the tail as a smaller population with no reason beside it.
//
// The trouble with counting them is that neither happens here. Every probe on
// this machine's Chrome names exactly one face, so the arms that report the
// skips are dead on every green run and the only evidence they work at all
// would be a browser that behaves differently — which is evidence nobody can
// schedule.
//
// inkCanaryReqAxes has the same shape of problem from the other end: it says
// which axes of the request key the probes actually differ in, and the answer
// on this grid is "weight and size, at one style". A test that only ever sees
// that grid cannot tell a census that reports the axes from one that reports
// whatever it was handed.
//
// # So the functions are lifted and asked
//
// The real declarations, out of the real file, the way inkglyph_test.go lifts
// the fold: a regex finds each one, they are written to a module with fixtures
// underneath, and node runs it. What that buys over a rewrite in Go is the only
// thing worth buying — the thing under test is the code that ships, so a
// function that stops matching the shape lifted here is a failure rather than a
// test quietly describing a copy.
//
// # And the join the parse rests on
//
// inkCanaryReqAxes splits a request key on spaces and takes size and weight off
// the END, because fontStyle is the one of the three that can be two words. It
// is right about the key INK_CANARY_REQ_JS builds today and nothing held the
// two together — which is the same "a join on two spellings of a key" failure
// that expression was extracted to prevent, arriving in the reader instead of
// in the writer. The last case below builds a key with the real expression and
// asks the parse to recover the three fields out of it.
//
// # And the join between the readings and the clauses that print them
//
// Everything above asks whether the two readings are RIGHT, and the browser
// pass asks whether the page runs. Neither asks whether the number
// `asked.canvasGenericMulti` holds is the one `inkCanaryAgreement` put in
// `multi`. That is ten assignments at one site, three of them off a single
// returned object, and a transposed pair there costs nothing anybody can see:
// the clause reads "N of the probes naming more than one face" over the count
// of the probes that came back with none, and both this test and the browser
// pass stay green — one of them never looks at `asked` and the other never has
// a nonzero to put in it.
//
// So the site is lifted too, and run rather than re-described. The block comes
// out of browser.mjs by the same regex move the declarations do, is compiled
// with `new Function` so the names it calls resolve to parameters, and is
// handed stubs whose every field is a different number. What comes back is
// `asked` itself, and each field has to hold the sentinel belonging to the
// reading it is named after. A swapped pair is then two failures naming both
// halves.
//
// The guard is asked the same way. Five of the ten are inside `if
// (facesHeld)`, and a run that could not hold the faces has no comparison to
// report — so those five have to be absent rather than zero, which is the
// difference between a clause that stays silent and one that recites a
// measurement nobody took.
func TestTheCanaryReadingsAreAskedOfPopulationsThisMachineDoesNotProduce(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("no node on this machine, so browser.mjs's own canary readings " +
			"cannot be run. They are exercised end to end by the browser pass in " +
			"run.sh, on whatever population that browser happens to produce, which " +
			"is what this test exists to stop being the only reading.")
	}
	src, err := os.ReadFile("browser.mjs")
	if err != nil {
		t.Fatalf("browser.mjs cannot be read: %v", err)
	}
	// Each declaration by its own name, so a rename is a failure here rather
	// than a silently smaller lift.
	lifted := []string{}
	for _, want := range []struct{ what, pattern string }{
		{"LAYOUT_UNIT", `(?m)^const LAYOUT_UNIT = .*;$`},
		{"INK_CANARY_FALLBACK", `(?m)^const INK_CANARY_FALLBACK = .*;$`},
		{"INK_CANARY_REQ_JS", `(?s)\nconst INK_CANARY_REQ_JS = ` + "`" + `.*?` + "`" + `;\n`},
		{"inkFaceList", `(?s)\nfunction inkFaceList\(.*?\n\}\n`},
		{"inkCanvasGenericGap", `(?s)\nfunction inkCanvasGenericGap\(.*?\n\}\n`},
		{"inkCanaryFaceList", `(?s)\nfunction inkCanaryFaceList\(.*?\n\}\n`},
		{"inkCanaryFaceSpread", `(?s)\nfunction inkCanaryFaceSpread\(.*?\n\}\n`},
		{"inkCanaryReqAxes", `(?s)\nfunction inkCanaryReqAxes\(.*?\n\}\n`},
		{"inkCanaryAxisPhrase", `(?s)\nfunction inkCanaryAxisPhrase\(.*?\n\}\n`},
		{"inkCanarySpreadPhrase", `(?s)\nfunction inkCanarySpreadPhrase\(.*?\n\}\n`},
		{"inkCanaryPassedPhrase", `(?s)\nfunction inkCanaryPassedPhrase\(.*?\n\}\n`},
		{"inkCanaryAgreement", `(?s)\nfunction inkCanaryAgreement\(.*?\n\}\n`},
	} {
		found := regexp.MustCompile(want.pattern).Find(src)
		if found == nil {
			t.Fatalf("browser.mjs no longer declares %s in the shape this test lifts "+
				"it out in.\n\n"+
				"These readings are asked here of populations this machine's browser "+
				"does not produce — a probe naming two faces, a probe naming none, a "+
				"request grid varying an axis this theme does not. Find what does the "+
				"job there and lift that instead of deleting this: without it those "+
				"arms are only ever run on whatever the browser pass happened to see.",
				want.what)
		}
		lifted = append(lifted, string(found))
	}
	// And the join site itself, as source. Not one of the declarations above
	// because it is not a declaration: it is the ten statements that carry
	// each reading's answer into the field the tail prints, and the only way
	// to ask whether they carry it to the right one is to run them.
	//
	// Anchored on both ends rather than on a function header — the block lives
	// inside a scan and there is no smaller thing to name. A rewrite that moves
	// it is a failure here, which is the point: the alternative is a test that
	// quietly stops covering the site it is named for.
	wiring := regexp.MustCompile(`(?s)\n        asked\.canvasFallback = ` +
		`inkCanaryFaceList\(canaryFaces\);.*?` +
		`if \(agreement\.fault\) problems\.push\(agreement\.fault\);\n        \}\n`).
		Find(src)
	if wiring == nil {
		t.Fatalf("browser.mjs no longer carries the canary readings into `asked` in " +
			"the shape this test lifts out.\n\n" +
			"The block runs from the fallback list to the agreement's fault push, and " +
			"it is the only place the numbers these readings return become the " +
			"numbers the tail prints. Find where that happens now and lift that " +
			"instead of deleting this: without it a transposed pair of assignments " +
			"puts one reading's count in another's clause and every test here stays " +
			"green.")
	}
	// Carried into the module as a JSON string so the block's own quotes and
	// newlines survive, and compiled with `new Function` so the names it calls
	// resolve to parameters rather than to the real declarations lifted above
	// — which are the things this particular reading is not about.
	wiringJSON, err := json.Marshal(string(wiring))
	if err != nil {
		t.Fatalf("the lifted join site will not encode: %v", err)
	}

	// INK_CANARY_REQ_JS is source carried as a string, evaluated in the page.
	// Evaluated here too, for the same reason: the key the parse is asked
	// about has to be the key the mount and the measurement build.
	script := filepath.Join(t.TempDir(), "canary.mjs")
	if err := os.WriteFile(script, []byte(strings.Join(lifted, "\n")+`
const probe = (req, ...families) => ({
    req, faces: families.map((family) => ({ family, glyphs: 3 })),
});
const row = (req, family, named, base) => ({
    req, family, asked: true, named, base, text: "Ag0",
});
// Two names for one face and one for another, so a fixture can say which
// direction it is about.
const out = {};

// The axes, over grids this theme does not draw.
out.axes = {};
for (const [what, probes] of Object.entries({
    sizeOnly: ["12px", "19px", "24px"].map((s) => probe("normal 400 " + s, "Times")),
    twoAxes: ["400", "700"].flatMap((w) =>
        ["12px", "19px"].map((s) => probe("normal " + w + " " + s, "Times"))),
    oneRequest: [probe("normal 400 12px", "Times")],
    twoWordStyle: [probe("oblique 10deg 400 12px", "Times"),
        probe("normal 400 12px", "Times")],
    none: [],
})) {
    const axes = inkCanaryReqAxes(probes);
    out.axes[what] = { axes, phrase: inkCanaryAxisPhrase(axes) };
}

// What the per-request reads came to, over populations this browser does not
// give: a probe that answered nothing, every probe answering nothing, and a
// family list that answers per request.
//
// Taken beside inkCanaryFaceList over the SAME probes, because the two walk
// one population with the same filter written twice and the recital picks its
// clause from one of them and its noun from the other.
out.spread = {};
for (const [what, probes] of Object.entries({
    none: [],
    allRead: [probe("normal 400 12px", "Times"), probe("normal 400 19px", "Times")],
    oneUnread: [{ req: "normal 400 12px", faces: null },
        probe("normal 400 19px", "Times")],
    // An empty face array rather than null — a read that came back and
    // reported no face, which is a different failure from a read that did not
    // come back and is the same thing to every reader downstream.
    emptyFaces: [probe("normal 400 12px"), probe("normal 400 19px", "Times")],
    noneRead: [{ req: "normal 400 12px", faces: null }, probe("normal 400 19px")],
    opticalCut: [probe("normal 400 12px", "Times"),
        probe("normal 400 19px", "Georgia")],
    // The same family at two glyph counts. The answer is what inkFaceList
    // renders, counts and all, so this is two answers and not one.
    glyphCount: [probe("normal 400 12px", "Times"),
        { req: "normal 400 19px", faces: [{ family: "Times", glyphs: 2 }] }],
    // And the same two faces in the other order, which is also two answers:
    // the platform's list is ordered and the fallback that resolves first is
    // the one the sentence is about.
    ordering: [probe("normal 400 12px", "Times", "Menlo"),
        probe("normal 400 19px", "Menlo", "Times")],
})) {
    const s = inkCanaryFaceSpread(probes);
    out.spread[what] = {
        mounted: s.mounted, read: s.read, answers: s.answers,
        list: inkCanaryFaceList(probes),
        phrase: inkCanarySpreadPhrase({
            canvasGenericReqs: s.mounted, canvasGenericRead: s.read,
            canvasGenericAnswers: s.answers,
            canvasGenericAxes: inkCanaryReqAxes(probes),
        }),
    };
}

// The skips, over probe answers this browser does not give.
out.skips = {};
for (const [what, probes] of Object.entries({
    allSingle: [probe("normal 400 12px", "Times"), probe("normal 400 19px", "Times")],
    onePlural: [probe("normal 400 12px", "Times", "Menlo"),
        probe("normal 400 19px", "Times")],
    allPlural: [probe("normal 400 12px", "Times", "Menlo"),
        probe("normal 400 19px", "Times", "Menlo")],
    oneUnread: [{ req: "normal 400 12px", faces: null },
        probe("normal 400 19px", "Times")],
})) {
    const rows = [row("normal 400 12px", "Times", 30, 30),
        row("normal 400 19px", "Times", 40, 40)];
    const a = inkCanaryAgreement(probes, rows);
    out.skips[what] = {
        asked: a.asked, agreed: a.agreed, multi: a.multi, unread: a.unread,
        unjoined: a.unjoined, fault: a.fault === null,
        phrase: inkCanaryPassedPhrase({
            canvasGenericMulti: a.multi, canvasGenericUnread: a.unread,
            canvasGenericUnjoined: a.unjoined,
        }),
    };
}

// And the fault itself, which needs a name that agrees and advances that do
// not — the direction nothing else in this file reports.
{
    const probes = [probe("normal 400 12px", "Times")];
    const rows = [row("normal 400 12px", "Times", 30, 42)];
    const a = inkCanaryAgreement(probes, rows);
    out.disagreement = { asked: a.asked, agreed: a.agreed, fault: a.fault };
}

// The key the parse is asked about, built by the expression that builds it in
// the page.
out.keys = ["normal", "oblique 10deg", "italic"].map((fontStyle) => {
    // Evaluated rather than re-spelled, which is the whole point of that
    // constant being source: the page runs this expression and so does this.
    const req = eval(INK_CANARY_REQ_JS)(
        { fontStyle, fontWeight: "700", fontSize: "19px" });
    const axes = inkCanaryReqAxes([probe(req, "Times")]);
    return { fontStyle, req, fixed: axes.fixed };
});
// And the join site, run.
//
// Every stub answers with a number nobody else answers with, so the check on
// the Go side is not "is this field a number" but "is it THAT reading's
// number". The block is compiled rather than pasted in so its calls bind to
// these parameters instead of to the real declarations above: what is under
// test here is which field each answer lands in, and a real reading answering
// 0 for two of them would hide exactly the swap this is for.
const WIRE = `+string(wiringJSON)+`;
const wire = new Function(
    "asked", "canaryFaces", "canvasAsked", "facesHeld", "problems",
    "inkCanaryFaceList", "inkCanaryFaceSpread", "inkCanaryReqAxes",
    "inkCanvasFallbackFault", "inkCanaryAgreement", WIRE);
const spread = { mounted: 101, read: 102, answers: 103 };
const paired = {
    asked: 201, agreed: 202, multi: 203, unread: 204, unjoined: 205, fault: null,
};
out.wiring = {};
for (const held of [true, false]) {
    const asked = {}, problems = [];
    wire(asked, [{ req: "normal 400 12px" }], [{ req: "normal 400 12px" }],
        held, problems,
        () => "FACELIST", () => spread, () => "AXES", () => null, () => paired);
    out.wiring[held ? "held" : "unheld"] = asked;
}
console.log(JSON.stringify(out));
`), 0o644); err != nil {
		t.Fatalf("the lifted canary readings will not write: %v", err)
	}
	out, err := exec.Command(node, script).CombinedOutput()
	if err != nil {
		t.Fatalf("node will not run browser.mjs's own canary readings: %v\n%s",
			err, out)
	}
	var got struct {
		Axes map[string]struct {
			Axes *struct {
				Varied []struct {
					Name   string
					Values []string
				}
				Fixed []struct{ Name, Value string }
				Reads int
				Grid  int
			}
			Phrase string
		}
		Spread map[string]struct {
			Mounted, Read, Answers int
			List                   *string
			Phrase                 string
		}
		Skips map[string]struct {
			Asked, Agreed, Multi, Unread, Unjoined int
			Fault                                  bool
			Phrase                                 string
		}
		Disagreement struct {
			Asked, Agreed int
			Fault         string
		}
		Keys []struct {
			FontStyle, Req string
			Fixed          []struct{ Name, Value string }
		}
		// `asked` as the lifted join site left it, twice: once with the faces
		// held and once without. Raw, because the ten fields are of three
		// different types and what is being asked of them is which sentinel
		// they carry, not what they mean.
		Wiring map[string]map[string]json.RawMessage
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("the lifted readings returned something this test cannot read: "+
			"%v\n%s", err, out)
	}

	// What the per-request reads came to, and the sentence built on it.
	//
	// # The one reading nothing asked
	//
	// inkCanaryFaceSpread was the third of the canary's readings and the only
	// one no test put a question to: the wiring check below stubs it, which
	// asks where its three numbers land and not what they are. Its `mounted`,
	// `read` and `answers` are three of the ten fields the tail recites, and
	// the sentence they carry is the one a reader uses to decide whether six
	// probe reads were worth their round trips.
	//
	// The populations it separates are all populations this machine does not
	// produce. Every probe on this Chrome answers with exactly one face, so
	// `read` is always `mounted` and `answers` is always 1 — the live run
	// exercises one row of the table below and the arms that matter are the
	// other seven.
	//
	// # And the join between the count and the noun
	//
	// inkCanaryFaceList and inkCanaryFaceSpread walk the same probes with the
	// same filter written twice — `p.faces && p.faces.length > 0` in each —
	// and the recital takes its CLAUSE from the spread's `answers` and its
	// NOUN from the list. A drift between the two spellings is a sentence that
	// says one face at every request and then names three, and neither
	// function would be wrong on its own. So the two are asked together over
	// every population here, structurally rather than by spelling the strings
	// out: a list is null exactly when nothing was read, and it takes its
	// per-request form exactly when the answers differed.
	for _, c := range []struct {
		what                   string
		mounted, read, answers int
		// What the recital says about this population, and what it must not.
		// The third branch of inkCanarySpreadPhrase is the whole reason that
		// clause is a function: a run that read nothing used to take the
		// "different faces" branch and recite a family list answering per
		// request over probes that answered nothing at all.
		says, saysNot string
	}{
		// The live shape, as the control: every probe answered and they agreed.
		{what: "allRead", mounted: 2, read: 2, answers: 1,
			says:    "one face at every one of them",
			saysNot: "different faces across them"},
		// A probe that did not come back. `read` apart from `mounted` is what
		// stops the sentence being quietly about the rest — and the answers
		// that did come back still agree, so the clause is unchanged and the
		// two numbers beside it are not.
		{what: "oneUnread", mounted: 2, read: 1, answers: 1,
			says:    "one face at every one of them",
			saysNot: "different faces across them"},
		// A probe that came back reporting no face, which is a different
		// failure from one that did not come back and the same number here.
		{what: "emptyFaces", mounted: 2, read: 1, answers: 1,
			says:    "one face at every one of them",
			saysNot: "different faces across them"},
		// Nothing answered. Two probes mounted, no reading at all — and the
		// clause has to say so rather than reciting zero as a measurement.
		{what: "noneRead", mounted: 2, read: 0, answers: 0,
			says:    "no face at any of them",
			saysNot: "family list answering per request"},
		// No probes at all, which is the same sentence with nothing mounted.
		{what: "none", mounted: 0, read: 0, answers: 0,
			says:    "no face at any of them",
			saysNot: "family list answering per request"},
		// The case a single probe at the page default got wrong: a family list
		// answering per request, naming a face at one size that no canary at
		// the other is put to.
		{what: "opticalCut", mounted: 2, read: 2, answers: 2,
			says:    "2 different faces across them",
			saysNot: "no optical cut"},
		// The same family at two glyph counts. An answer is what inkFaceList
		// renders and that carries the counts, so this is two answers — which
		// is the honest reading: the same family covering different characters
		// at two sizes is the stack doing something a single probe would miss.
		{what: "glyphCount", mounted: 2, read: 2, answers: 2,
			says:    "2 different faces across them",
			saysNot: "no optical cut"},
		// And the same two faces in the other order, which is also two. The
		// platform's list is ordered and the face that resolves first is the
		// one the premise is about, so a reordering is a difference and not a
		// spelling.
		{what: "ordering", mounted: 2, read: 2, answers: 2,
			says:    "2 different faces across them",
			saysNot: "no optical cut"},
	} {
		row, ok := got.Spread[c.what]
		if !ok {
			t.Errorf("%s: the lifted spread returned nothing for this population.\n\n"+
				"Every row here is a probe answer this browser does not give, which is "+
				"the whole reason they are asked in node rather than left to the "+
				"browser pass.", c.what)
			continue
		}
		if row.Mounted != c.mounted || row.Read != c.read || row.Answers != c.answers {
			t.Errorf("%s: inkCanaryFaceSpread reports %d mounted, %d read, %d "+
				"answer(s), and the fixture is %d mounted, %d that answered, %d "+
				"distinct.\n\n"+
				"These three are what the tail recites in place of the assumption a "+
				"single probe at the page default was: how many requests the generic "+
				"was asked at, how many came back, and how many different faces those "+
				"were. `read` short of `mounted` is a request whose answer is missing "+
				"while the others are present, and `answers` above one is the family "+
				"list answering per request — the case one probe gets wrong.",
				c.what, row.Mounted, row.Read, row.Answers,
				c.mounted, c.read, c.answers)
			continue
		}
		// Rendered for the message rather than dereferenced in it: null is one
		// of the two answers this arm is about and a nil pointer through %q is
		// not a sentence.
		rendered := "no noun at all"
		if row.List != nil {
			rendered = fmt.Sprintf("%q", *row.List)
		}
		// The two spellings of one filter, held against each other.
		if (row.Read == 0) != (row.List == nil) {
			t.Errorf("%s: the spread read %d of %d probes and inkCanaryFaceList "+
				"answered %s.\n\n"+
				"Both walk this population with the same test for a probe that "+
				"answered, written out in each — so a list with a noun over a spread "+
				"that read nothing, or a null over a spread that read something, is "+
				"the two filters having come apart. The recital takes its clause from "+
				"one and its noun from the other, and neither is wrong on its own.",
				c.what, row.Read, row.Mounted, rendered)
		}
		perRequest := row.List != nil && strings.Contains(*row.List, " at ")
		if (row.Answers > 1) != perRequest {
			t.Errorf("%s: the spread reports %d distinct answers and "+
				"inkCanaryFaceList rendered %s.\n\n"+
				"The list collapses to one clause when every request came back with "+
				"the same answer and names them per request when they did not, which "+
				"is the same `answers` count decided a second time. A count above one "+
				"beside a collapsed noun is the recital saying the stack answered per "+
				"request and then naming one face for all of them.",
				c.what, row.Answers, rendered)
		}
		if !strings.Contains(row.Phrase, c.says) {
			t.Errorf("%s: the recital's clause over %d mounted, %d read and %d "+
				"answer(s) is %q, and it does not say %q.\n\n"+
				"inkCanarySpreadPhrase is the sentence and the reading in one "+
				"spelling. A population that reads nothing, one that agrees and one "+
				"that does not are three different findings, and the clause said two "+
				"of them until this test was written — a run where every probe came "+
				"back empty answered zero, took the branch written for a family list "+
				"answering per request, and recited it.",
				c.what, c.mounted, c.read, c.answers, row.Phrase, c.says)
		}
		if strings.Contains(row.Phrase, c.saysNot) {
			t.Errorf("%s: the recital's clause is %q, which says %q about a "+
				"population of %d mounted, %d read and %d answer(s).\n\n"+
				"The negative is half the reading: a three-way branch that reaches "+
				"the right sentence and also the wrong one is a two-way branch with "+
				"an extra string in it.",
				c.what, row.Phrase, c.saysNot, c.mounted, c.read, c.answers)
		}
	}

	// The axes, and the two halves of what they bound.
	for _, c := range []struct {
		what, varied, saysNot string
		grid, reads           int
	}{
		// One axis varied: the case the six reads were bought for, and the
		// case where the bound is one read per value of that axis.
		{what: "sizeOnly", varied: "size", grid: 3, reads: 3,
			saysNot: "nothing here is a reading about style or weight"},
		// Two: the grid is a product, and a read at one point of it says
		// nothing about the others.
		{what: "twoAxes", varied: "weight", grid: 4, reads: 4,
			saysNot: "nothing here is a reading about style"},
		// And none, which is the state a single probe was in before any of
		// this: the answers agree because there is only one of them.
		{what: "oneRequest", varied: "", grid: 1, reads: 1,
			saysNot: "says nothing beyond that one request"},
		// A style spelled with two words, which is the whole reason the parse
		// takes size and weight off the end.
		{what: "twoWordStyle", varied: "style", grid: 2, reads: 2,
			saysNot: "nothing here is a reading about weight or size"},
	} {
		row := got.Axes[c.what]
		if row.Axes == nil {
			t.Errorf("%s: inkCanaryReqAxes answered null over %d probes.\n\n"+
				"Null is reserved for a run that read nothing, where a sentence "+
				"should have no noun rather than a zero. A grid with probes in it "+
				"answering null leaves the tail saying \"no request this run could "+
				"read\" about requests it did read.", c.what, c.reads)
			continue
		}
		if row.Axes.Reads != c.reads || row.Axes.Grid != c.grid {
			t.Errorf("%s: inkCanaryReqAxes reports %d reads over a grid of %d, and "+
				"the fixture is %d probes over %d points.\n\n"+
				"The grid is the product of the varied axes' value counts and it is "+
				"the bound: it says how many reads could have exhibited a difference. "+
				"A grid that does not match the requests mounted is a bound about a "+
				"population nobody asked.",
				c.what, row.Axes.Reads, row.Axes.Grid, c.reads, c.grid)
		}
		if c.varied == "" {
			if len(row.Axes.Varied) != 0 {
				t.Errorf("%s: every probe carries one request and inkCanaryReqAxes "+
					"reports %d varied axes.\n\n"+
					"With one request there is nothing to have varied, and an axis "+
					"reported as varied over one value is the census claiming a "+
					"reading it does not have.", c.what, len(row.Axes.Varied))
			}
		} else if len(row.Axes.Varied) == 0 || row.Axes.Varied[0].Name != c.varied {
			t.Errorf("%s: inkCanaryReqAxes reports varied axes %v and the fixture "+
				"varies %s first.\n\n"+
				"The axes are reported in key order — style, weight, size — which is "+
				"the order the request is spelled in, so a reader can put the clause "+
				"beside the key. An axis reported as varied that the fixture holds "+
				"fixed is a reading about the parse rather than about the requests.",
				c.what, row.Axes.Varied, c.varied)
		}
		if !strings.Contains(row.Phrase, c.saysNot) {
			t.Errorf("%s: inkCanaryAxisPhrase does not say %q.\n\nIt said: %q\n\n"+
				"The half a tail without this gets wrong is the blind one: \"no "+
				"optical cut at these sizes\" is true and reads as \"one probe would "+
				"have done\", and neither is a statement about an axis every request "+
				"here shares. Naming the fixed axes is what stops the agreement being "+
				"read as wider than it is.", c.what, c.saysNot, row.Phrase)
		}
	}

	// And the skips: what the comparison declined to ask, and what it cost.
	for _, c := range []struct {
		what                                   string
		asked, agreed, multi, unread, unjoined int
		says                                   string
	}{
		// The state this machine is always in, where the clause has to be
		// silent: a clause reporting zeroes on every healthy run is a clause
		// nobody reads.
		{what: "allSingle", asked: 2, agreed: 2, says: ""},
		// One probe naming two faces. The run at that request is dropped, and
		// the drop is the number this whole reading was missing.
		{what: "onePlural", asked: 1, agreed: 1, multi: 1, unjoined: 1,
			says: "1 of the probes naming more than one face"},
		// Every probe plural, which is the comparison going entirely quiet —
		// the state that used to arrive as a smaller population with no reason
		// attached.
		{what: "allPlural", asked: 0, agreed: 0, multi: 2, unjoined: 2,
			says: "costing the join 2 runs"},
		// And a read that failed rather than answered plurally, which is a
		// different finding and is counted apart.
		{what: "oneUnread", asked: 1, agreed: 1, unread: 1,
			says: "1 of them coming back with no face at all"},
	} {
		row := got.Skips[c.what]
		if row.Asked != c.asked || row.Agreed != c.agreed || row.Multi != c.multi ||
			row.Unread != c.unread || row.Unjoined != c.unjoined {
			t.Errorf("%s: inkCanaryAgreement reports asked %d, agreed %d, multi %d, "+
				"unread %d, unjoined %d; the fixture is %d, %d, %d, %d, %d.\n\n"+
				"`asked` is the population the comparison had and the other three are "+
				"what it declined. Until they were counted a reading that had stopped "+
				"being made looked exactly like a grid with fewer runs in it, which is "+
				"the shape of thing this file reports everywhere else and did not "+
				"report about itself.",
				c.what, row.Asked, row.Agreed, row.Multi, row.Unread, row.Unjoined,
				c.asked, c.agreed, c.multi, c.unread, c.unjoined)
		}
		if !row.Fault {
			t.Errorf("%s: inkCanaryAgreement reports a fault over rows whose two "+
				"advances are equal.\n\n"+
				"This arm fires on one NAME over two advances. Equal advances are the "+
				"agreeing case, and a fault here would put a message about a "+
				"disagreement in front of a reader on a page where the two readings "+
				"agree.", c.what)
		}
		if c.says == "" {
			if row.Phrase != "" {
				t.Errorf("%s: inkCanaryPassedPhrase says %q with nothing skipped.\n\n"+
					"A clause that reports zeroes on every healthy run is a clause a "+
					"reader stops seeing — and the state it is for is the comparison "+
					"going quiet, which is exactly when the numbers stop being zero.",
					c.what, row.Phrase)
			}
			continue
		}
		if !strings.Contains(row.Phrase, c.says) {
			t.Errorf("%s: inkCanaryPassedPhrase does not say %q.\n\nIt said: %q\n\n"+
				"The runs this comparison passed over are the population where a "+
				"name-versus-metric disagreement is most likely — a family list "+
				"resolving past its head is the state both readings are trying to see "+
				"— and they were dropped without a count.", c.what, c.says, row.Phrase)
		}
	}

	// And the direction the fault is for, which no green browser run produces.
	if got.Disagreement.Asked != 1 || got.Disagreement.Agreed != 0 ||
		got.Disagreement.Fault == "" {
		t.Errorf("a probe naming the very family a run is drawn by, over a run whose "+
			"two advances are 12px apart, gives asked %d agreed %d and fault %q.\n\n"+
			"That is the arm inkCanaryAgreement exists for: same face, same request, "+
			"same canvas, same string is one advance, so one name over two advances "+
			"means one of the two readings is about something else. It has never "+
			"fired on this machine, and a reader's whole evidence that it can is "+
			"this.", got.Disagreement.Asked, got.Disagreement.Agreed,
			got.Disagreement.Fault)
	} else if !strings.Contains(got.Disagreement.Fault, "measures") ||
		!strings.Contains(got.Disagreement.Fault, "12.000000px") {
		t.Errorf("the disagreement message does not quote the distance: %q\n\n"+
			"The number is the half of that message a reader acts on — it is what "+
			"says the two requests reached faces of different widths while the probe "+
			"said they reached one.", got.Disagreement.Fault)
	}

	// And the parse against the expression that writes the key.
	//
	// Every field the shorthand carries has to come back out under its own
	// name. A style spelled with a space is the case that decides it: split on
	// the FIRST space and "oblique 10deg 400 19px" parses as style "oblique",
	// weight "10deg", size "400" — three fields, all wrong, and every axis
	// reading downstream is then about a key nobody wrote.
	for _, key := range got.Keys {
		fixed := map[string]string{}
		for _, a := range key.Fixed {
			fixed[a.Name] = a.Value
		}
		if fixed["style"] != key.FontStyle || fixed["weight"] != "700" ||
			fixed["size"] != "19px" {
			t.Errorf("INK_CANARY_REQ_JS builds %q from style %q, weight 700, size "+
				"19px, and inkCanaryReqAxes reads style %q, weight %q, size %q back "+
				"out of it.\n\n"+
				"That expression is carried as source and evaluated in the page "+
				"because the probe mount and the canvas measurement have to spell one "+
				"key — a join on two spellings matches nothing, silently. The parse "+
				"here is a THIRD reader of that key and it was held to it by nothing: "+
				"it takes size and weight off the end precisely because fontStyle can "+
				"be two words, and a shorthand that grows a field would leave every "+
				"axis clause naming the wrong one.",
				key.Req, key.FontStyle, fixed["style"], fixed["weight"], fixed["size"])
		}
	}
	// And the wiring: which reading's answer each field of `asked` is holding.
	//
	// The sentinels are all different, so this is a permutation check and not a
	// type check. Named one at a time rather than compared as a map, because
	// the failure that matters is a PAIR — the message has to be able to say
	// which reading's number turned up under which clause's name.
	held, wired := got.Wiring["held"], 0
	if held == nil {
		t.Errorf("the lifted join site produced no `asked` with the faces held.\n\n" +
			"That block is what carries each reading's answer into the field the " +
			"tail prints. With nothing back from it, nothing here is asking whether " +
			"the numbers reach the right clauses.")
	}
	wanted := []struct{ field, want, reading string }{
		{"canvasFallback", `"FACELIST"`, "inkCanaryFaceList's list of faces"},
		{"canvasGenericReqs", "101", "inkCanaryFaceSpread's `mounted`"},
		{"canvasGenericRead", "102", "inkCanaryFaceSpread's `read`"},
		{"canvasGenericAnswers", "103", "inkCanaryFaceSpread's `answers`"},
		{"canvasGenericAxes", `"AXES"`, "inkCanaryReqAxes's answer"},
		{"canvasGenericPaired", "201", "inkCanaryAgreement's `asked`"},
		{"canvasGenericAgreed", "202", "inkCanaryAgreement's `agreed`"},
		{"canvasGenericMulti", "203", "inkCanaryAgreement's `multi`"},
		{"canvasGenericUnread", "204", "inkCanaryAgreement's `unread`"},
		{"canvasGenericUnjoined", "205", "inkCanaryAgreement's `unjoined`"},
	}
	// And which reading each sentinel belongs to, so a swap can be named from
	// one message rather than inferred from two. A number that is nobody's is
	// reported as that, which is the field being assigned from something this
	// fixture does not stub.
	whose := map[string]string{}
	for _, c := range wanted {
		whose[c.want] = c.reading
	}
	for _, c := range wanted {
		if string(held[c.field]) == c.want {
			wired++
			continue
		}
		landed := "a value no reading in this fixture answered"
		if who, known := whose[string(held[c.field])]; known {
			landed = who
		}
		t.Errorf("asked.%s is the field the tail prints %s from, and it holds %s — "+
			"which is %s.\n\n"+
			"Every stub in this fixture answers with a different number, so a field "+
			"holding the wrong one is an assignment carrying the wrong reading, and "+
			"that is invisible everywhere else: the tail prints whatever is in the "+
			"field, the readings above are asked of the functions directly, and the "+
			"browser pass has never had a nonzero to put in these at all. A "+
			"transposed pair reads as \"N of the probes naming more than one face\" "+
			"over the count of the probes that named none. The whole of `asked` was "+
			"%v.",
			c.field, c.reading, string(held[c.field]), landed, held)
	}
	// And the guard, which is the other half of the same join.
	unheld := got.Wiring["unheld"]
	for _, field := range []string{"canvasGenericPaired", "canvasGenericAgreed",
		"canvasGenericMulti", "canvasGenericUnread", "canvasGenericUnjoined"} {
		if _, set := unheld[field]; set {
			t.Errorf("a run that could not hold the faces still set asked.%s, to "+
				"%s.\n\n"+
				"Those five come off inkCanaryAgreement and the comparison is only "+
				"made inside `if (facesHeld)` — with no faces held there is no join "+
				"and nothing to report. A field set here is a clause reciting a "+
				"measurement nobody took, which is the failure the counts themselves "+
				"were introduced for: a reading that has gone quiet arriving as a "+
				"number rather than as silence. The whole of `asked` was %v.",
				field, string(unheld[field]), unheld)
		}
	}
	// And the two that are NOT inside the guard, which have to survive it: the
	// spread and the axes are read off the probes alone and say what asking per
	// request bought whether or not a comparison could be made.
	for _, c := range []struct{ field, want string }{
		{"canvasGenericReqs", "101"},
		{"canvasGenericAxes", `"AXES"`},
	} {
		if got := string(unheld[c.field]); got != c.want {
			t.Errorf("a run that could not hold the faces left asked.%s holding %q "+
				"rather than %s.\n\n"+
				"The spread and the axes are read off the probes and not off the join, "+
				"so they are the reading that survives a run with no comparison in it "+
				"— and a guard that grew to cover them would take the tail's last "+
				"sentence about the canary away on exactly the runs that need one.",
				c.field, got, c.want)
		}
	}

	t.Logf("browser.mjs's own canary readings, lifted and run over %d request grids "+
		"and %d probe answers this machine's browser does not produce — a probe "+
		"naming two faces, a probe naming none, a style spelled with a space, and "+
		"a name agreeing over advances 12px apart — plus %d probe populations the "+
		"spread and the face list are asked TOGETHER over, which is the third "+
		"reading and the one nothing used to put a question to: a family list "+
		"answering per request, the same family at two glyph counts, two faces in "+
		"the other order, and a run where nothing answered at all, whose clause "+
		"used to recite a family list answering per request over probes that "+
		"answered nothing — plus %d keys built by INK_CANARY_REQ_JS itself and "+
		"parsed back into their three fields, and the join site itself lifted and "+
		"run over stubs answering a different number each: %d of `asked`'s fields "+
		"hold the reading they are named after, and the %d that are inside the "+
		"faces-held guard are absent on a run that could not hold them",
		len(got.Axes), len(got.Skips), len(got.Spread), len(got.Keys), wired, 5)
	_ = fmt.Sprint()
}
