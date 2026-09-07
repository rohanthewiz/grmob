package spec

import (
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
const sample = `
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
	if _, err := Parse("<html><body>not the specification</body></html>"); err == nil {
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
	if _, err := Parse(b.String()); err == nil {
		t.Error("Parse accepted 90 roles with no implicit orientation between " +
			"them — every oriented role's default would silently become \"\"")
	}
}
