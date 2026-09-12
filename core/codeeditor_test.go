package core

import "testing"

// core.CodeEditor's wire shape: the node type, the seeded defaults, the rows
// as GridRow children, and the options each landing on their own key.

func TestCodeEditorRendersRowsAsGridRowChildren(t *testing.T) {
	ctx := NewContext()
	rows := []GridRow{
		{{Text: "func", Fg: "#CC7832"}, {Text: " main() {"}},
		nil,
		{{Text: "}"}},
	}
	n := CodeEditor("func main() {\n\n}", func(string) {}, rows, FontSize(13)).Render(ctx)

	if n.Type != "CodeEditor" {
		t.Fatalf("type = %q", n.Type)
	}
	if n.Style == nil || n.Style.FontSize != 13 {
		t.Errorf("style props did not reach the editor: %+v", n.Style)
	}
	if len(n.Children) != 3 {
		t.Fatalf("children = %d, want one per row", len(n.Children))
	}
	for i, c := range n.Children {
		if c.Type != "GridRow" {
			t.Errorf("child %d type = %q, want GridRow — the rows must be the same node "+
				"TextGrid builds, or the renderers need a second row reader", i, c.Type)
		}
	}
	// A nil row normalizes to an empty GridRow for the reason gridRowNode
	// gives: a row that goes from "no runs" to "no runs" must not patch.
	if r, ok := n.Children[1].Props["runs"].(GridRow); !ok || r == nil || len(r) != 0 {
		t.Errorf("the nil row carried as %#v, want an empty GridRow", n.Children[1].Props["runs"])
	}
	if got, ok := n.Children[0].Props["runs"].(GridRow); !ok || len(got) != 2 || got[0].Fg != "#CC7832" {
		t.Errorf("row 0's runs did not survive: %#v", n.Children[0].Props["runs"])
	}
}

// Every option is written on every pass, at its default when nothing asked for
// it. The absence of a key is invisible to a host across an update-props patch
// (the patch carries the whole new map and the hosts iterate what is in it),
// so "off" has to be a value.
func TestCodeEditorSeedsEveryOptionSoNoneCanVanish(t *testing.T) {
	ctx := NewContext()
	plain := CodeEditor("x", func(string) {}, nil).Render(ctx)

	for key, want := range map[string]any{
		"lineNumbers":   false,
		"readOnly":      false,
		"tabSize":       4,
		"commentPrefix": "//",
	} {
		got, present := plain.Props[key]
		if !present {
			t.Errorf("%q is absent from a plain editor; a host would never hear it turn off", key)
			continue
		}
		if got != want {
			t.Errorf("%q = %#v, want the default %#v", key, got, want)
		}
	}
	if plain.Props["value"] != "x" {
		t.Errorf("value = %#v", plain.Props["value"])
	}
	if id, _ := plain.Props["onChange"].(string); id == "" {
		t.Error("onChange did not register a callback")
	}
}

func TestCodeEditorOptionsOverwriteTheDefaults(t *testing.T) {
	ctx := NewContext()
	n := CodeEditor("x", func(string) {}, nil,
		LineNumbers(),
		ReadOnly(),
		TabSize(2),
		CommentPrefix("#"),
		Placeholder("type here"),
	).Render(ctx)

	for key, want := range map[string]any{
		"lineNumbers":   true,
		"readOnly":      true,
		"tabSize":       2,
		"commentPrefix": "#",
		"placeholder":   "type here",
	} {
		if got := n.Props[key]; got != want {
			t.Errorf("%q = %#v, want %#v", key, got, want)
		}
	}
}

// A negative tab size is clamped rather than refused: a render pass is not a
// place to panic over an argument, and "no spaces" is the honest reading.
func TestTabSizeClampsNegativeToZero(t *testing.T) {
	ctx := NewContext()
	n := CodeEditor("x", func(string) {}, nil, TabSize(-2)).Render(ctx)
	if got := n.Props["tabSize"]; got != 0 {
		t.Errorf("tabSize = %#v, want 0", got)
	}
}

// An editor with no toolbar stamps no command props at all, so an app that
// never issues one renders byte-identical trees to before the mechanism
// existed. This is the epoch-0 sentinel, and it is the same claim focus.go
// makes about focusEpoch.
func TestCodeEditorStampsNoCommandUntilOneIsIssued(t *testing.T) {
	ctx := NewContext()
	var ref *EditorRef
	view := ComponentFunc(func(c *Context) *Node {
		ref = UseEditorRef(c)
		return CodeEditor("x", func(string) {}, nil, EditorTarget(ref)).Render(c)
	})

	ctx.Reset()
	n := view.Render(ctx)
	if _, ok := n.Props["editorEpoch"]; ok {
		t.Errorf("an editor that has never been commanded carries %#v", n.Props["editorEpoch"])
	}
	if _, ok := n.Props["editorCommand"]; ok {
		t.Error("editorCommand is stamped with no command issued")
	}

	RunEditorCommand(ref, EditIndent)
	ctx.Reset()
	n = view.Render(ctx)
	if n.Props["editorEpoch"] != 1 || n.Props["editorCommand"] != EditIndent {
		t.Fatalf("after one command: epoch %#v, command %#v", n.Props["editorEpoch"], n.Props["editorCommand"])
	}

	// The repeat case, which is the whole reason the epoch is a counter: the
	// same command twice has to reach the host twice, and two identical prop
	// maps produce no patch.
	RunEditorCommand(ref, EditIndent)
	ctx.Reset()
	n = view.Render(ctx)
	if n.Props["editorEpoch"] != 2 {
		t.Errorf("a repeated command left the epoch at %#v; the host would never see it", n.Props["editorEpoch"])
	}
	if n.Props["editorCommand"] != EditIndent {
		t.Errorf("command = %#v", n.Props["editorCommand"])
	}
}

// Two editors on one screen are two refs and no cross-talk. This is the one
// way the editor epoch differs from the focus epoch, which is per-app because
// a dismiss has to reach every field.
func TestEditorCommandsReachOnlyTheirOwnEditor(t *testing.T) {
	ctx := NewContext()
	var left, right *EditorRef
	view := ComponentFunc(func(c *Context) *Node {
		left, right = UseEditorRef(c), UseEditorRef(c)
		return containerNode(c, "Column", Style{}, []PropsAndChildren{
			CodeEditor("a", func(string) {}, nil, EditorTarget(left)),
			CodeEditor("b", func(string) {}, nil, EditorTarget(right)),
		})
	})

	ctx.Reset()
	view.Render(ctx)
	RunEditorCommand(left, EditSelectAll)
	ctx.Reset()
	n := view.Render(ctx)

	if n.Children[0].Props["editorCommand"] != EditSelectAll {
		t.Errorf("the commanded editor carries %#v", n.Children[0].Props["editorCommand"])
	}
	if _, ok := n.Children[1].Props["editorEpoch"]; ok {
		t.Errorf("the other editor was stamped too: %#v", n.Children[1].Props)
	}
}

// A nil ref degrades rather than panicking, at both ends.
func TestNilEditorRefIsInert(t *testing.T) {
	ctx := NewContext()
	n := CodeEditor("x", func(string) {}, nil, EditorTarget(nil)).Render(ctx)
	if _, ok := n.Props["editorEpoch"]; ok {
		t.Error("a nil EditorTarget stamped something")
	}
	RunEditorCommand(nil, EditIndent) // must not panic
}

// The selection's wire format, parsed in core so app code sees two ints.
func TestOnSelectionChangeParsesThePayload(t *testing.T) {
	ctx := NewContext()
	var got [2]int
	var calls int
	n := CodeEditor("x", func(string) {}, nil,
		OnSelectionChange(func(start, end int) { calls++; got = [2]int{start, end} }),
	).Render(ctx)

	id, _ := n.Props["onSelectionChange"].(string)
	if id == "" {
		t.Fatal("onSelectionChange did not register a callback")
	}

	ctx.TriggerTextCallback(id, "3:9")
	if calls != 1 || got != [2]int{3, 9} {
		t.Errorf("after 3:9 — calls %d, got %v", calls, got)
	}

	// A drag upward is reported end-first by some hosts and is an ordinary
	// selection, so it is normalized rather than refused.
	ctx.TriggerTextCallback(id, "9:3")
	if calls != 2 || got != [2]int{3, 9} {
		t.Errorf("after 9:3 — calls %d, got %v", calls, got)
	}

	// Anything that is not two non-negative integers is dropped, so a
	// malformed report never reaches app code looking like a deselection.
	for _, bad := range []string{"", "3", "3:", ":9", "a:b", "-1:2", "3:4:5", "3 9"} {
		before := calls
		ctx.TriggerTextCallback(id, bad)
		if calls != before {
			t.Errorf("%q was delivered as a selection", bad)
		}
	}
}

// A nil handler produces no prop at all rather than a registered callback that
// would panic on the first selection change.
func TestOnSelectionChangeIgnoresANilHandler(t *testing.T) {
	ctx := NewContext()
	n := CodeEditor("x", func(string) {}, nil, OnSelectionChange(nil)).Render(ctx)
	if _, ok := n.Props["onSelectionChange"]; ok {
		t.Error("a nil handler still registered a callback")
	}
}

func TestParseSelectionNormalizesAndRejects(t *testing.T) {
	for payload, want := range map[string][3]int{
		"0:0":     {0, 0, 1},
		"12:18":   {12, 18, 1},
		"18:12":   {12, 18, 1},
		"007:009": {7, 9, 1},
	} {
		start, end, ok := parseSelection(payload)
		if !ok || start != want[0] || end != want[1] {
			t.Errorf("parseSelection(%q) = %d,%d,%v", payload, start, end, ok)
		}
	}
	for _, bad := range []string{"", "5", "a:1", "1:b", "-1:-1", "1:2:3"} {
		if _, _, ok := parseSelection(bad); ok {
			t.Errorf("parseSelection(%q) accepted it", bad)
		}
	}
}
