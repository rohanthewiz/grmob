package spec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A miniature specification in the shape ReSpec publishes, exercising every
// extraction this package does.
//
// Hand-written rather than an excerpt of the real document, and that is the
// point: these tests are on the `go test ./...` path, where the 1.4MB download
// is deliberately absent. A parser whose only test is "it agrees with the file
// it was written against" cannot distinguish "reads the markup correctly" from
// "reads this copy correctly", and the failure that would slip through is the
// one aria/verify's conformance test also cannot see — a parser and a fixture
// that are wrong together because one generated the other.
//
// Each role here is a case:
//
//	widget      the ordinary shape: supported + inherited + required, unioned
//	oriented    an implicit-orientation sentence, which is prose and not a cell
//	nested      a <section> inside the role description, which a naive scan for
//	            the next </section> would truncate at — taking the feature table
//	            with it and reporting a role that supports nothing
//	banned      an inherited attribute the role explicitly prohibits
//	owner       a Required Owned Elements cell holding a path (group → thing)
//	stub        a synonym with no feature table of its own
//
// The title heading is here for a different reason from the sections: Parse
// checks the document's edition before it believes anything else about it, so
// without one every test below that goes through Parse would be exercising the
// version guard rather than the thing it names. `specTitle` is the constant it
// has to satisfy, spelled from Version so a bump does not silently turn these
// into version-guard tests again.
const sample = specHeading + `
<section class="role notoc" id="widget">
  <div class="role-description"><p>A widget.</p></div>
  <table class="role-features"><tbody>
    <tr><th class="role-properties-head" scope="row">Supported States and Properties:</th>
        <td class="role-properties"><a href="#aria-expanded" class="state-reference"><code>aria-expanded</code></a></td></tr>
    <tr><th class="role-inherited-head" scope="row">Inherited States and Properties:</th>
        <td class="role-inherited"><ul>
        <li><a href="#aria-hidden" class="state-reference"><code>aria-hidden</code></a> (state)</li>
        </ul></td></tr>
    <tr><th class="role-required-properties-head" scope="row">Required States and Properties:</th>
        <td class="role-required-properties"><ul>
        <li><a href="#aria-selected" class="state-reference"><code>aria-selected</code></a></li>
        </ul></td></tr>
    <tr><th class="role-namefrom-head" scope="row">Name From:</th>
        <td class="role-namefrom">author</td></tr>
  </tbody></table>
</section>
<section class="role notoc" id="oriented">
  <div class="role-description">
    <p>Elements with the role <code>oriented</code> have an implicit
    <a href="#aria-orientation" class="property-reference"><code>aria-orientation</code></a>
    value of <code>vertical</code>.</p>
  </div>
  <table class="role-features"><tbody>
    <tr><th class="role-properties-head" scope="row">Supported States and Properties:</th>
        <td class="role-properties"><a href="#aria-orientation" class="property-reference"><code>aria-orientation</code></a></td></tr>
    <tr><th class="role-namefrom-head" scope="row">Name From:</th>
        <td class="role-namefrom">author</td></tr>
  </tbody></table>
</section>
<section class="role notoc" id="nested">
  <div class="role-description">
    <p>A role whose description holds a section of its own.</p>
    <section class="note"><p>Authors should note something.</p></section>
  </div>
  <table class="role-features"><tbody>
    <tr><th class="role-properties-head" scope="row">Supported States and Properties:</th>
        <td class="role-properties"><a href="#aria-level" class="property-reference"><code>aria-level</code></a></td></tr>
    <tr><th class="role-namefrom-head" scope="row">Name From:</th>
        <td class="role-namefrom">author</td></tr>
  </tbody></table>
</section>
<section class="role notoc" id="banned">
  <table class="role-features"><tbody>
    <tr><th class="role-inherited-head" scope="row">Inherited States and Properties:</th>
        <td class="role-inherited"><ul>
        <li><a href="#aria-level" class="property-reference"><code>aria-level</code></a></li>
        <li><a href="#aria-expanded" class="state-reference"><code>aria-expanded</code></a></li>
        </ul></td></tr>
    <tr><th class="role-disallowed-head" scope="row">Prohibited States and Properties:</th>
        <td class="role-disallowed"><a href="#aria-level" class="property-reference"><code>aria-level</code></a></td></tr>
    <tr><th class="role-namefrom-head" scope="row">Name From:</th>
        <td class="role-namefrom">prohibited</td></tr>
  </tbody></table>
</section>
<section class="role notoc" id="owner">
  <table class="role-features"><tbody>
    <tr><th class="role-mustcontain-head" scope="row">Required Owned Elements:</th>
        <td class="role-mustcontain"><ul>
        <li><a href="#group" class="role-reference"><code>group</code></a> <abbr title="containing" class="symbol">&rarr;</abbr> <a href="#widget" class="role-reference"><code>widget</code></a></li>
        <li><a href="#widget" class="role-reference"><code>widget</code></a></li>
        </ul></td></tr>
    <tr><th class="role-namefrom-head" scope="row">Name From:</th>
        <td class="role-namefrom">author</td></tr>
  </tbody></table>
</section>
<section class="role notoc" id="stub">
  <div class="role-description"><p>See synonym <a href="#banned" class="role-reference"><code>banned</code></a>.</p></div>
</section>
`

func parseSample(t *testing.T) Doc {
	t.Helper()
	doc, _ := parseDoc(sample)
	if len(doc) != 6 {
		t.Fatalf("parsed %d sections, want 6: %v", len(doc), keys(doc))
	}
	return doc
}

func keys(d Doc) []string {
	out := make([]string, 0, len(d))
	for k := range d {
		out = append(out, k)
	}
	sortStrings(out)
	return out
}

// The three attribute cells are one answer.
//
// Required is the one that is easy to leave out and the one that matters most:
// aria-selected on `option`, aria-level on `heading` and aria-valuenow on every
// range role are *required* properties and appear in no other cell. A parser
// reading only Supported and Inherited would report that ARIA does not allow
// aria-selected on an option — and every guard held to that fixture would then
// refuse the attribute the whole listbox pattern is built on.
func TestTheThreeAttributeCellsAreUnioned(t *testing.T) {
	got := parseSample(t)["widget"].Attributes
	want := []string{"aria-expanded", "aria-hidden", "aria-selected"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("widget.Attributes = %v, want %v", got, want)
	}
}

// A role may inherit an attribute and then forbid it. `caption` is the real
// case: it takes aria-label from its superclass and prohibits it outright, so
// the union alone reports an attribute the specification explicitly refuses.
func TestAProhibitedAttributeIsRemovedFromTheUnion(t *testing.T) {
	got := parseSample(t)["banned"].Attributes
	want := []string{"aria-expanded"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("banned.Attributes = %v, want %v — the prohibited cell was not "+
			"subtracted", got, want)
	}
}

// The implicit orientation is the one fact that is not a table cell. Nine roles
// state it in prose and all nine word it identically.
func TestTheImplicitOrientationIsReadFromTheSentence(t *testing.T) {
	doc := parseSample(t)
	if got := doc["oriented"].Orientation; got != "vertical" {
		t.Errorf("oriented.Orientation = %q, want %q", got, "vertical")
	}
	// Supporting the attribute and having a default are different facts, and
	// conflating them is one of the four errors that were in the transcribed
	// fixture: ARIA 1.1 gave radiogroup an implicit orientation, 1.2 removed
	// it, and the role still takes aria-orientation.
	if got := doc["widget"].Orientation; got != "" {
		t.Errorf("widget.Orientation = %q, want \"\" — a role with no implicit "+
			"value must not acquire one", got)
	}
}

// A role description holding a <section> of its own — a note, an example —
// truncates every naive scan at the first </section>, and the damage is silent:
// the feature table falls outside the body and the role parses as one that
// supports nothing at all.
func TestANestedSectionDoesNotTruncateTheRole(t *testing.T) {
	got := parseSample(t)["nested"].Attributes
	if strings.Join(got, ",") != "aria-level" {
		t.Errorf("nested.Attributes = %v, want [aria-level] — the section body was "+
			"cut at the first </section> and took the feature table with it", got)
	}
}

// The Required Owned cell can hold a path — "group → option" means an option
// inside a group counts — and every consumer here asks the flat question.
func TestARequiredOwnedPathIsFlattened(t *testing.T) {
	got := parseSample(t)["owner"].RequiredOwned
	want := []string{"group", "widget"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("owner.RequiredOwned = %v, want %v", got, want)
	}
}

// `none` and `presentation` are mutual synonyms and only one of the pair carries
// a feature table. Without resolution the other parses as a role that allows no
// attributes and permits a name, which is the opposite of true.
func TestASynonymTakesItsTargetsFacts(t *testing.T) {
	doc := parseSample(t)
	if !doc["stub"].NameProhibited {
		t.Error("stub.NameProhibited = false — the synonym did not inherit, so a " +
			"role ARIA forbids a name on reads as one that allows it")
	}
	if strings.Join(doc["stub"].Attributes, ",") != strings.Join(doc["banned"].Attributes, ",") {
		t.Errorf("stub.Attributes = %v, want its synonym's %v",
			doc["stub"].Attributes, doc["banned"].Attributes)
	}
}

// Name From is a word, not a list of links, so it is the one cell read as text.
func TestNameProhibitionIsReadOffTheNameFromCell(t *testing.T) {
	doc := parseSample(t)
	if doc["widget"].NameProhibited {
		t.Error("widget.NameProhibited = true, and its Name From cell says author")
	}
	if !doc["banned"].NameProhibited {
		t.Error("banned.NameProhibited = false, and its Name From cell says prohibited")
	}
}

// The floors are the whole defence against a silent format change.
//
// Every failure this parser can have produces empty cells rather than an error:
// a renamed class, a reworded sentence, a restructured table all yield roles
// that support nothing and orient nothing, which reads downstream as ARIA having
// said no. Only the document's own shape can tell that apart from the truth.
func TestAParseThatFoundAlmostNothingIsAnError(t *testing.T) {
	if _, err := Parse(sample); err == nil {
		t.Error("Parse accepted a six-role document — the role floor is not " +
			"reached, so a markup change that dropped nine roles in ten would pass")
	}
	if _, err := Parse(specHeading + "<html><body>not the specification</body></html>"); err == nil {
		t.Error("Parse accepted a document with no role sections at all")
	}

	// The orientation floor is the half that a role-count check cannot cover:
	// a document with every role intact and the orientation sentence reworded
	// parses to ~94 roles, none of which has a default axis.
	var b strings.Builder
	for i := 0; i < 90; i++ {
		b.WriteString(`<section class="role notoc" id="r`)
		b.WriteString(string(rune('a'+i/26)) + string(rune('a'+i%26)))
		b.WriteString(`"><td class="role-namefrom">author</td></section>`)
	}
	if _, err := Parse(specHeading + b.String()); err == nil {
		t.Error("Parse accepted 90 roles with no implicit orientation between " +
			"them — every oriented role's default would silently become \"\"")
	}
}

// The ReSpec title heading, at the edition this package reads.
//
// Built from Version rather than written out, so the constant is what these
// tests agree with — a hand-typed "1.2" here would go on satisfying the guard
// after a bump and would leave every test below exercising a version the
// package no longer claims.
const specHeading = `<h1 id="title" class="title">Accessible Rich Internet ` +
	`Applications (WAI-ARIA) ` + Version + `</h1>`

// A stale download is named as one, rather than reported as a fixture that
// disagrees with the specification.
//
// This is the whole of the finding. The 1.1 and 1.2 documents are the same
// ReSpec output with different content in the cells, so every other signal
// this package has — the role count, the class names, the orientation sentence
// — is identical between them, and a 1.1 copy parses cleanly to ~94 roles and
// regenerates a fixture differing on exactly the four facts the revision
// changed. What that produces downstream is aria/verify printing "aria.json is
// not what aria/gen would write" and naming radiogroup's orientation as the
// first difference: a true statement about 1.1 offered as a transcription
// error in a file nobody touched.
func TestAStaleDownloadIsRefusedBeforeItIsParsed(t *testing.T) {
	stale := strings.Replace(sample,
		"(WAI-ARIA) "+Version, "(WAI-ARIA) 1.1", 1)
	if stale == sample {
		t.Fatal("the sample's title heading no longer holds the version; update " +
			"specHeading rather than deleting this test")
	}
	_, err := Parse(stale)
	if err == nil {
		t.Fatal("Parse accepted a WAI-ARIA 1.1 document: the fixture it would " +
			"generate disagrees with the committed one on four facts, and the " +
			"failure reads as a broken fixture rather than a stale download")
	}
	// The message has to name the download, because the reader arrives at it
	// holding a fixture diff and a working checkout.
	for _, want := range []string{"1.1", Version, "aria/fetch.sh"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not mention %q: %v", want, err)
		}
	}

	// A document with no heading at all is refused too. It is the shape an
	// error page or a partial download takes, and "no version" must not read
	// as "the right version".
	if _, err := Parse(strings.Replace(sample, specHeading, "", 1)); err == nil {
		t.Error("Parse accepted a document that does not say which edition it is")
	}
}

// The three places the edition is written must agree.
//
// Version is the Go constant, LocalPath derives its filename from it, and
// aria/fetch.sh's URL is the third — a shell script, which cannot read a Go
// constant and so is the one that drifts. The failure it produces is the
// nastiest of the family: the script downloads 1.2 into a file whose name says
// 1.3, or 1.3 into a file the guard then refuses, and either way the person
// following the three commands in fetch.sh's own header gets an error about a
// document they just fetched correctly.
//
// A source check because there is nothing else available: making the script
// ask Go for the number would put a `go run` in front of a curl, and this is
// the one thing in the repository that is allowed to need the network and
// nothing else.
func TestTheFetchScriptAgreesWithTheVersionConstant(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "fetch.sh"))
	if err != nil {
		t.Fatalf("reading aria/fetch.sh: %v", err)
	}
	src := string(raw)

	wantURL := "https://www.w3.org/TR/wai-aria-" + Version + "/"
	if !strings.Contains(src, wantURL) {
		t.Errorf("aria/fetch.sh does not fetch %s — the script and spec.Version "+
			"name different editions of ARIA, and whichever one the download lands "+
			"as, one of them is about to refuse it", wantURL)
	}

	// And into the path this package reads it back from. LocalPath is stated
	// from the module root; the script cd's into aria/ first, so what it must
	// hold is the tail.
	wantOut := strings.TrimPrefix(LocalPath, "aria/")
	if !strings.Contains(src, wantOut) {
		t.Errorf("aria/fetch.sh does not write %s — the fetch would succeed and "+
			"every reader would go on skipping for want of a local copy", wantOut)
	}
}

// SpecVersion reads the edition and nothing near it.
//
// Exported separately from Parse because the two callers want different
// things: Parse folds it into a refusal, and aria/verify's conformance test
// wants to skip with a sentence about the download rather than fail with a
// diff. So it answers "" for a document that has no heading instead of
// guessing, and it is not fooled by the fifteen other places the string
// "wai-aria-1.1" appears in the published 1.2 document (its own change log
// links to the previous revision throughout).
func TestSpecVersionReadsTheTitleHeadingAlone(t *testing.T) {
	if got := SpecVersion(sample); got != Version {
		t.Errorf("SpecVersion(sample) = %q, want %q", got, Version)
	}
	if got := SpecVersion("<html><body>nothing</body></html>"); got != "" {
		t.Errorf("SpecVersion of a document with no heading = %q, want \"\"", got)
	}
	// The trap the real document sets: a 1.2 copy links to
	// https://www.w3.org/TR/wai-aria-1.1/ fifteen times, so anything scanning
	// the body for a version string reads 1.1 out of a perfectly current file
	// and refuses it.
	linky := specHeading + `<p>See <a href="https://www.w3.org/TR/wai-aria-1.1/">` +
		`WAI-ARIA 1.1</a> and <a href="https://www.w3.org/TR/wai-aria-1.0/">1.0</a>.</p>`
	if got := SpecVersion(linky); got != Version {
		t.Errorf("SpecVersion read %q out of a %s document that links to its own "+
			"predecessors — the heading is the only statement of the edition", got, Version)
	}
}
