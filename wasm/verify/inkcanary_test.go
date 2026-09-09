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
		{"inkCanaryReqAxes", `(?s)\nfunction inkCanaryReqAxes\(.*?\n\}\n`},
		{"inkCanaryAxisPhrase", `(?s)\nfunction inkCanaryAxisPhrase\(.*?\n\}\n`},
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
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("the lifted readings returned something this test cannot read: "+
			"%v\n%s", err, out)
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
	t.Logf("browser.mjs's own canary readings, lifted and run over %d request grids "+
		"and %d probe answers this machine's browser does not produce — a probe "+
		"naming two faces, a probe naming none, a style spelled with a space, and "+
		"a name agreeing over advances 12px apart — plus %d keys built by "+
		"INK_CANARY_REQ_JS itself and parsed back into their three fields",
		len(got.Axes), len(got.Skips), len(got.Keys))
	_ = fmt.Sprint()
}
