package main

import (
	"regexp"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/htmlout"
)

// The Modal overlay chassis lives twice — htmlout writes a CSS declaration
// list, the WASM runtime assigns CSSOM properties — and the comment on each
// side claims they are the same nine declarations. This is what makes the
// claim checkable.
//
// The cost of drift is a dialog that is a centred fixed overlay on one web
// target and an ordinary block in the middle of the page on the other, with
// nothing failing anywhere: core.ModalNode carries no Style, so there is no
// author declaration for either target to disagree with and no test outside
// this one that would notice.
//
// Parsed rather than executed, for the reason jstable_test.go gives: a Go test
// that needed Node would stop running for anyone who has only Go, and this
// block is nine assignments with no computation in them.

// The shape the runtime's chassis has to keep, and which the parse requires:
//
//	if (nodeType === "Modal") {
//	    out.<property> = out.<property> || "<value>";
//	    ...
//	}
//
// The `||` is not decoration. htmlout writes its chassis *ahead* of the
// author's declarations, so the cascade gives the author the last word; the
// runtime's style pass is total and assigns the author's value first, so the
// fallback is how it says the same thing. A plain assignment here would make
// the runtime the one target where a hand-built Modal's own Style loses — and
// would still pass a test that only compared values, which is why the pattern
// itself is what is matched.
//
// The two property names are captured separately and compared in the loop
// rather than written as a backreference: Go's regexp is RE2 and has none, and
// a pattern that quietly matched `out.top = out.left || "0"` would be worse
// than one that needs a line of Go to finish the job.
var jsChassisDecl = regexp.MustCompile(`out\.(\w+) = out\.(\w+) \|\| "([^"]*)";`)

// cssName turns a CSSOM property name into its CSS spelling, which is the
// mapping the DOM itself makes: alignItems is align-items. Only the four
// hyphenated properties in the chassis exercise it, but it is written as the
// general rule rather than a four-row table, since the next chassis property
// would otherwise need a row nobody would remember to add.
func cssName(prop string) string {
	var b strings.Builder
	for _, r := range prop {
		if r >= 'A' && r <= 'Z' {
			b.WriteByte('-')
			b.WriteRune(r + ('a' - 'A'))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func TestRuntimeModalChassisMatchesGo(t *testing.T) {
	src := runtimeSource(t)

	const marker = `if (nodeType === "Modal") {`
	at := strings.Index(src, marker)
	if at < 0 {
		t.Fatalf("grmob-runtime.js: no %q — styleFromGrMob no longer states the Modal "+
			"chassis, so a modal keeps it only until the first update-style patch", marker)
	}
	block := src[at:]
	if end := strings.Index(block, "\n        }"); end > 0 {
		block = block[:end]
	}

	got := map[string]string{}
	for _, m := range jsChassisDecl.FindAllStringSubmatch(block, -1) {
		if m[1] != m[2] {
			t.Errorf("grmob-runtime.js: `out.%s = out.%s || ...` — the fallback reads a "+
				"different property than it writes", m[1], m[2])
			continue
		}
		got[cssName(m[1])] = m[3]
	}

	for _, decl := range htmlout.ModalChassis() {
		prop, want := decl[0], decl[1]
		switch have, ok := got[prop]; {
		case !ok:
			t.Errorf("%s: htmlout writes %s:%s and the runtime states nothing — "+
				"a modal is a fixed overlay on one web target and a block on the other",
				prop, prop, want)
		case have != want:
			t.Errorf("%s: runtime says %q, htmlout says %q", prop, have, want)
		}
		delete(got, prop)
	}
	for prop, have := range got {
		t.Errorf("%s: the runtime states %q and htmlout states nothing", prop, have)
	}

	// display is the exemption, and it is an exemption in both directions:
	// htmlout writes it from the visible prop (so it is not in ModalChassis),
	// and the runtime deletes the key so its total pass leaves the property to
	// the prop channel. A style pass that assigned one would close an open
	// dialog on the next restyle, which styleless_test.mjs holds behaviorally;
	// this is the line that makes it true.
	if !strings.Contains(block, "delete out.display;") {
		t.Error("grmob-runtime.js: styleFromGrMob no longer abstains from a Modal's display — " +
			"the next update-style patch will decide whether the dialog is open")
	}
}
