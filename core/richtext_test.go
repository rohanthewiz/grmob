package core

import (
	"testing"

	"github.com/rohanthewiz/grmob/richtext"
)

var testDoc = richtext.Doc{Blocks: []richtext.Block{
	{Kind: richtext.Heading1, Runs: []richtext.Run{{Text: "Title"}}},
	{Kind: richtext.Paragraph, Runs: []richtext.Run{{Text: "body", Bold: true}}},
}}

func TestRichTextEditorCarriesTheDocAsJSON(t *testing.T) {
	ctx := NewContext()
	n := RichTextEditor(testDoc, nil, Placeholder("Write something…")).Render(ctx)

	if n.Type != "RichTextEditor" {
		t.Fatalf("type = %q", n.Type)
	}
	if got := n.Props["doc"]; got != testDoc.JSON() {
		t.Errorf("doc = %#v, want the document's own JSON", got)
	}
	if n.Props["placeholder"] != "Write something…" {
		t.Errorf("placeholder = %#v", n.Props["placeholder"])
	}
	// Seeded so a host can hear the option turn off again; see the builder.
	if n.Props["readOnly"] != false {
		t.Errorf("readOnly = %#v, want a seeded false", n.Props["readOnly"])
	}
	// The doc is the styled buffer, so there is no separate decoration and no
	// children at all — the one way this node differs in shape from CodeEditor.
	if len(n.Children) != 0 {
		t.Errorf("the editor has %d children; the doc is the whole value", len(n.Children))
	}
}

func TestRichTextEditorParsesTheDocBeforeCallingBack(t *testing.T) {
	ctx := NewContext()
	var got richtext.Doc
	var calls int
	n := RichTextEditor(richtext.Doc{}, func(d richtext.Doc) { calls++; got = d }).Render(ctx)

	id, _ := n.Props["onChange"].(string)
	if id == "" {
		t.Fatal("onChange did not register a callback")
	}
	ctx.TriggerTextCallback(id, testDoc.JSON())
	if calls != 1 || len(got.Blocks) != 2 || got.Blocks[0].Kind != richtext.Heading1 {
		t.Fatalf("after a valid edit: calls %d, doc %#v", calls, got)
	}

	// A host sending something core cannot read is a bug in that host, and the
	// right response is to drop the edit rather than hand app code an empty
	// document — which would be echoed straight back and would delete the note.
	ctx.TriggerTextCallback(id, "not json")
	if calls != 1 {
		t.Errorf("a malformed payload reached the handler")
	}
}

func TestRichTextEditorWithNoHandlerIgnoresEdits(t *testing.T) {
	ctx := NewContext()
	n := RichTextEditor(testDoc, nil).Render(ctx)
	id, _ := n.Props["onChange"].(string)
	ctx.TriggerTextCallback(id, testDoc.JSON()) // must not panic
}

// The command vocabulary, including the two that carry an argument in the
// string because the command channel is one prop.
func TestRichTextCommandSpellings(t *testing.T) {
	if got := EditLink("https://example.com/a:b"); got != "link:https://example.com/a:b" {
		t.Errorf("EditLink = %q — everything after the first colon is the URL", got)
	}
	// An empty URL is Unlink's job, not a link to nowhere.
	if got := EditLink(""); got != EditUnlink {
		t.Errorf("EditLink(\"\") = %q, want %q", got, EditUnlink)
	}
	for kind, want := range map[richtext.BlockKind]string{
		richtext.Paragraph: "block:p",
		richtext.Heading2:  "block:h2",
		richtext.Bullet:    "block:bullet",
		richtext.BlockCode: "block:code",
	} {
		if got := EditBlock(kind); got != want {
			t.Errorf("EditBlock(%q) = %q, want %q", kind, got, want)
		}
	}
}

func TestOnRichSelectionChangeParsesThePayload(t *testing.T) {
	ctx := NewContext()
	var got RichSelection
	var calls int
	n := RichTextEditor(testDoc, nil,
		OnRichSelectionChange(func(sel RichSelection) { calls++; got = sel }),
	).Render(ctx)

	id, _ := n.Props["onSelectionChange"].(string)
	if id == "" {
		t.Fatal("onSelectionChange did not register a callback")
	}

	ctx.TriggerTextCallback(id,
		`{"s":12,"e":18,"marks":["bold","code"],"link":"https://x","block":"h2"}`)
	want := RichSelection{Start: 12, End: 18, Bold: true, Code: true,
		Link: "https://x", Block: richtext.Heading2}
	if calls != 1 || got != want {
		t.Errorf("calls %d, got %#v, want %#v", calls, got, want)
	}
	if !got.HasSelection() {
		t.Error("12–18 is a selection")
	}

	// A bare caret, and a mark a newer host reports that this Go does not know:
	// ignored rather than refused, because an older reader must not fail on a
	// newer writer.
	ctx.TriggerTextCallback(id, `{"s":3,"e":3,"marks":["highlight"],"block":"p"}`)
	if calls != 2 || got.HasSelection() || got.Bold {
		t.Errorf("after a bare caret: %#v", got)
	}

	// A drag upward is reported end-first by some hosts and is an ordinary
	// selection.
	ctx.TriggerTextCallback(id, `{"s":9,"e":4,"marks":[],"block":"p"}`)
	if got.Start != 4 || got.End != 9 {
		t.Errorf("a reversed range was not normalized: %#v", got)
	}

	// And anything unreadable is dropped rather than delivered as a zeroed
	// selection, which a toolbar would draw as "nothing is bold" — a lie that
	// looks exactly like the truth.
	before := calls
	for _, bad := range []string{"", "null-ish", "[1,2]", "{"} {
		ctx.TriggerTextCallback(id, bad)
	}
	if calls != before {
		t.Errorf("%d malformed payloads were delivered", calls-before)
	}
}

func TestRichTextEditorTakesEditorCommands(t *testing.T) {
	ctx := NewContext()
	var ref *EditorRef
	view := ComponentFunc(func(c *Context) *Node {
		ref = UseEditorRef(c)
		return RichTextEditor(testDoc, nil, EditorTarget(ref)).Render(c)
	})

	ctx.Reset()
	n := view.Render(ctx)
	if _, ok := n.Props["editorEpoch"]; ok {
		t.Error("an untouched editor carries a command stamp")
	}

	RunEditorCommand(ref, EditBlock(richtext.Heading2))
	ctx.Reset()
	n = view.Render(ctx)
	if n.Props["editorEpoch"] != 1 || n.Props["editorCommand"] != "block:h2" {
		t.Errorf("epoch %#v, command %#v", n.Props["editorEpoch"], n.Props["editorCommand"])
	}
}
