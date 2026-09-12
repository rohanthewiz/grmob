package richtext

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// sample is a document using every block kind and every mark, which is what the
// round-trip tests below are run over. Written as Go values rather than parsed
// from Markdown, so that a bug in the parser cannot make a round-trip test pass
// by being wrong in both directions.
var sample = Doc{Blocks: []Block{
	{Kind: Heading1, Runs: []Run{{Text: "The title"}}},
	{Kind: Paragraph, Runs: []Run{
		{Text: "Plain, "},
		{Text: "bold", Bold: true},
		{Text: ", "},
		{Text: "italic", Italic: true},
		{Text: ", "},
		{Text: "under", Underline: true},
		{Text: ", "},
		{Text: "struck", Strike: true},
		{Text: ", "},
		{Text: "code()", Code: true},
		{Text: ", and a "},
		{Text: "link", Link: "https://example.com"},
		{Text: "."},
	}},
	{Kind: Heading2, Runs: []Run{{Text: "A section"}}},
	{Kind: Heading3, Runs: []Run{{Text: "A subsection"}}},
	{Kind: Bullet, Runs: []Run{{Text: "first"}}},
	{Kind: Bullet, Runs: []Run{{Text: "second", Bold: true}}},
	{Kind: Numbered, Runs: []Run{{Text: "one"}}},
	{Kind: Numbered, Runs: []Run{{Text: "two"}}},
	{Kind: Quote, Runs: []Run{{Text: "Someone said this."}}},
	{Kind: BlockCode, Runs: []Run{{Text: "func main() {\n\tprintln(1)\n}"}}},
	{Kind: Paragraph, Runs: []Run{{Text: "The end."}}},
}}

// --- JSON ------------------------------------------------------------------

func TestJSONRoundTrip(t *testing.T) {
	data, err := json.Marshal(sample)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back Doc
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(back, sample) {
		t.Errorf("round trip changed the document.\n got %#v\nwant %#v", back, sample)
	}
}

// The wire shape itself, pinned: the three hand-written host serializers read
// these keys, and a rename here is a silent break on three targets at once.
func TestJSONWireShape(t *testing.T) {
	doc := Doc{Blocks: []Block{
		{Kind: Heading2, Runs: []Run{{Text: "Hi", Bold: true, Link: "https://x"}}},
		{Kind: Paragraph, Runs: []Run{{Text: "plain"}}},
		{Kind: Paragraph},
	}}
	got, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"b":[{"k":"h2","r":[{"t":"Hi","b":1,"l":"https://x"}]},` +
		`{"r":[{"t":"plain"}]},{}]}`
	if string(got) != want {
		t.Errorf("wire shape:\n got %s\nwant %s", got, want)
	}
}

// A paragraph is the default, so it costs no key. That is the commonest block
// in any document by a wide margin, and this is the whole reason the kind is
// omitempty.
func TestParagraphKindIsNotWritten(t *testing.T) {
	got, _ := json.Marshal(Doc{Blocks: []Block{{Kind: Paragraph, Runs: []Run{{Text: "x"}}}}})
	if strings.Contains(string(got), `"k"`) {
		t.Errorf("a paragraph wrote its kind: %s", got)
	}
}

// The empty document, at both ends. `{}` is what an editor with nothing in it
// sends, and it has to decode to a document with no blocks rather than to a
// document with one empty one.
func TestEmptyDocument(t *testing.T) {
	got, _ := json.Marshal(Doc{})
	if string(got) != "{}" {
		t.Errorf("empty document marshalled as %s, want {}", got)
	}
	doc, err := ParseJSON("{}")
	if err != nil || len(doc.Blocks) != 0 {
		t.Errorf("ParseJSON({}) = %#v, %v", doc, err)
	}
	if !doc.IsEmpty() {
		t.Error("a document with no blocks is not empty")
	}
	// And the case IsEmpty exists for: an editor that has been focused and left
	// holds one empty paragraph on every host.
	if !(Doc{Blocks: []Block{{Kind: Paragraph}}}).IsEmpty() {
		t.Error("a document of one empty paragraph is not empty")
	}
}

// Everything arriving here came from outside Go, so nothing is trusted to be in
// the vocabulary.
func TestUnmarshalNormalizesWhatItIsGiven(t *testing.T) {
	doc, err := ParseJSON(`{"b":[
		{"k":"marquee","r":[{"t":"a"}]},
		{"r":[{"t":""},{"t":"b","b":true},{"t":"c","i":"1"}]},
		{"k":"h1"}
	]}`)
	if err != nil {
		t.Fatalf("a document with unknown values must still parse: %v", err)
	}
	want := Doc{Blocks: []Block{
		// An unknown kind degrades to a paragraph: the text is never at risk,
		// only its presentation.
		{Kind: Paragraph, Runs: []Run{{Text: "a"}}},
		// The empty run is dropped — invisible on every target, a span on all
		// four — and both a JSON bool and a JSON string read as truthy, because
		// two of the three hosts' JSON libraries write bools for bools.
		{Kind: Paragraph, Runs: []Run{{Text: "b", Bold: true}, {Text: "c", Italic: true}}},
		// An empty block is a blank line and is kept.
		{Kind: Heading1},
	}}
	if !reflect.DeepEqual(doc, want) {
		t.Errorf("got  %#v\nwant %#v", doc, want)
	}
}

func TestParseJSONRejectsGarbage(t *testing.T) {
	if _, err := ParseJSON("not json"); err == nil {
		t.Error("garbage parsed as a document")
	}
}

// --- Markdown --------------------------------------------------------------

// The round trip, which is what makes Markdown usable as an import/export door
// rather than as a one-way rendering.
func TestMarkdownRoundTripsTheSubset(t *testing.T) {
	md := sample.Markdown()
	back, err := FromMarkdown(md)
	if err != nil {
		t.Fatalf("FromMarkdown: %v", err)
	}
	if !reflect.DeepEqual(back, sample) {
		t.Errorf("round trip through Markdown changed the document.\n--- markdown ---\n%s\n\n"+
			"got  %#v\nwant %#v", md, back, sample)
	}
}

// The output itself, so a change to the emitter is a deliberate edit rather
// than a surprise. Read as a document — which is what the Markdown pair is for.
func TestMarkdownOutput(t *testing.T) {
	doc := Doc{Blocks: []Block{
		{Kind: Heading1, Runs: []Run{{Text: "Title"}}},
		{Kind: Paragraph, Runs: []Run{{Text: "Some "}, {Text: "bold", Bold: true}, {Text: "."}}},
		{Kind: Bullet, Runs: []Run{{Text: "one"}}},
		{Kind: Bullet, Runs: []Run{{Text: "two"}}},
		{Kind: Numbered, Runs: []Run{{Text: "first"}}},
		{Kind: Numbered, Runs: []Run{{Text: "second"}}},
		{Kind: Quote, Runs: []Run{{Text: "quoted"}}},
	}}
	const want = "# Title\n\n" +
		"Some **bold**.\n\n" +
		// A run of list items is one list, so no blank line inside it.
		"- one\n- two\n\n" +
		// And the ordinals are counted, because the Markdown is read by people.
		"1. first\n2. second\n\n" +
		"> quoted"
	if got := doc.Markdown(); got != want {
		t.Errorf("markdown:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

// The property the escaping exists for: a paragraph that happens to contain
// Markdown syntax has to come back as that paragraph, not as the formatting it
// looks like.
func TestMarkdownEscapesTextThatLooksLikeSyntax(t *testing.T) {
	for _, text := range []string{
		"a ** b", "use *stars*", "a [link] here", "back\\slash", "~~not struck~~",
		"<u>not underlined</u>", "# not a heading", "- not a bullet", "1. not a list",
		"2 * 3 = 6", "a_b_c", "trailing `tick",
	} {
		doc := Doc{Blocks: []Block{{Kind: Paragraph, Runs: []Run{{Text: text}}}}}
		back, err := FromMarkdown(doc.Markdown())
		if err != nil {
			t.Fatalf("%q: %v", text, err)
		}
		if got := back.PlainText(); got != text {
			t.Errorf("%q round-tripped as %q (markdown was %q)", text, got, doc.Markdown())
		}
	}
}

// A marker with no closer is literal text, which is the forgiving reading and
// the only one that keeps a document of prose from sprouting formatting nobody
// asked for.
func TestFromMarkdownLeavesUnclosedMarkersAlone(t *testing.T) {
	doc, _ := FromMarkdown("an *unclosed emphasis and a `tick")
	if got := doc.PlainText(); got != "an *unclosed emphasis and a `tick" {
		t.Errorf("got %q", got)
	}
	for _, run := range doc.Blocks[0].Runs {
		if run.Italic || run.Code {
			t.Errorf("an unclosed marker produced formatting: %#v", run)
		}
	}
}

// Nesting, which the recursive parser handles for free and which a hand-written
// document is full of.
func TestFromMarkdownNesting(t *testing.T) {
	doc, _ := FromMarkdown("**bold and *both* bold** and [a *link*](https://x)")
	want := []Run{
		{Text: "bold and ", Bold: true},
		{Text: "both", Bold: true, Italic: true},
		{Text: " bold", Bold: true},
		{Text: " and "},
		{Text: "a ", Link: "https://x"},
		{Text: "link", Italic: true, Link: "https://x"},
	}
	if !reflect.DeepEqual(doc.Blocks[0].Runs, want) {
		t.Errorf("got  %#v\nwant %#v", doc.Blocks[0].Runs, want)
	}
}

// A code block is taken whole and never inline-parsed: its content is the
// content, asterisks and all.
func TestFromMarkdownCodeFenceIsVerbatim(t *testing.T) {
	doc, _ := FromMarkdown("```\nx := *p\n// **not bold**\n```")
	if len(doc.Blocks) != 1 || doc.Blocks[0].Kind != BlockCode {
		t.Fatalf("got %#v", doc.Blocks)
	}
	if got := doc.Blocks[0].Text(); got != "x := *p\n// **not bold**" {
		t.Errorf("got %q", got)
	}
}

// Markdown's first documented loss: a blank line is the block separator and
// nothing else, so an empty paragraph has no representation. Dropped on export,
// never invented on import. The JSON keeps them, and the JSON is the wire.
func TestMarkdownHasNoEmptyParagraph(t *testing.T) {
	doc := Doc{Blocks: []Block{
		{Runs: []Run{{Text: "a"}}},
		{Kind: Paragraph},
		{Kind: Paragraph},
		{Runs: []Run{{Text: "b"}}},
	}}
	if got := doc.Markdown(); got != "a\n\nb" {
		t.Errorf("markdown = %q, want the two blocks and one separator", got)
	}
	back, _ := FromMarkdown("a\n\n\n\n\nb")
	texts := make([]string, len(back.Blocks))
	for i, block := range back.Blocks {
		texts[i] = block.Text()
	}
	if strings.Join(texts, "|") != "a|b" {
		t.Errorf("blocks = %q, want a and b — a run of blank lines is one separator", texts)
	}
	// An empty code block is the exception and survives: an empty fence is a
	// fence, and reads back as one.
	fenced := Doc{Blocks: []Block{{Kind: BlockCode}}}
	if got := fenced.Markdown(); got != "```\n\n```" {
		t.Errorf("empty fence = %q", got)
	}
}

// Markdown's one documented loss, stated by a test so it cannot become a
// surprise: a code span carries its text literally, so the other marks on it
// are not expressible and are dropped on export. The JSON keeps all six.
func TestMarkdownDropsMarksOnACodeSpan(t *testing.T) {
	doc := Doc{Blocks: []Block{{Runs: []Run{{Text: "x", Code: true, Bold: true}}}}}
	if got := doc.Markdown(); got != "`x`" {
		t.Errorf("got %q", got)
	}
	back, _ := FromMarkdown(doc.Markdown())
	if run := back.Blocks[0].Runs[0]; !run.Code || run.Bold {
		t.Errorf("got %#v, want code and not bold", run)
	}
}

// --- HTML ------------------------------------------------------------------

func TestHTMLOutput(t *testing.T) {
	const want = "<h1>The title</h1>" +
		"<p>Plain, <strong>bold</strong>, <em>italic</em>, <u>under</u>, <s>struck</s>, " +
		`<code>code()</code>, and a <a href="https://example.com">link</a>.</p>` +
		"<h2>A section</h2><h3>A subsection</h3>" +
		// Consecutive list items are gathered into one list, which is the one
		// thing this output has to do that the model does not express.
		"<ul><li>first</li><li><strong>second</strong></li></ul>" +
		"<ol><li>one</li><li>two</li></ol>" +
		"<blockquote>Someone said this.</blockquote>" +
		"<pre><code>func main() {\n\tprintln(1)\n}</code></pre>" +
		"<p>The end.</p>"
	if got := sample.HTML(); got != want {
		t.Errorf("html:\n got %s\nwant %s", got, want)
	}
}

// A document is a value, not markup. Every string that reaches the output is
// escaped, a link's href included — the one way a text editor's content could
// re-enter the page as a tag.
func TestHTMLEscapesEverything(t *testing.T) {
	doc := Doc{Blocks: []Block{
		{Kind: Paragraph, Runs: []Run{{Text: `<script>alert("x")</script>`, Link: `" onmouseover="x`}}},
		{Kind: BlockCode, Runs: []Run{{Text: "<b>not bold</b>"}}},
	}}
	got := doc.HTML()
	if strings.Contains(got, "<script>") || strings.Contains(got, "<b>") {
		t.Errorf("markup survived escaping: %s", got)
	}
	if strings.Contains(got, `" onmouseover="`) {
		t.Errorf("an href escaped its attribute: %s", got)
	}
}

// --- The small shared pieces ----------------------------------------------

func TestPlainText(t *testing.T) {
	if got := sample.PlainText(); !strings.HasPrefix(got, "The title\nPlain, bold") {
		t.Errorf("got %q", got)
	}
	// Blocks join with a single newline, because a Block is a line: an empty
	// block is already how a writer says "blank line here".
	doc := Doc{Blocks: []Block{{Runs: []Run{{Text: "a"}}}, {}, {Runs: []Run{{Text: "b"}}}}}
	if got := doc.PlainText(); got != "a\n\nb" {
		t.Errorf("got %q", got)
	}
}

// The census, which four consumers read: the widget's toolbar and the `block:`
// command in each of the three live hosts.
func TestBlockKindsIsTheCensus(t *testing.T) {
	kinds := BlockKinds()
	if len(kinds) != 8 {
		t.Errorf("BlockKinds has %d entries: %v", len(kinds), kinds)
	}
	seen := map[BlockKind]bool{}
	for _, kind := range kinds {
		if seen[kind] {
			t.Errorf("%q appears twice", kind)
		}
		seen[kind] = true
		if normalizeKind(kind) != kind {
			t.Errorf("%q is in the census and does not normalize to itself", kind)
		}
	}
	// And it is a copy, so a caller cannot rewrite the vocabulary.
	kinds[0] = "nonsense"
	if BlockKinds()[0] != Paragraph {
		t.Error("BlockKinds handed out its own slice")
	}
}
