package render_test

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/render"
	"github.com/rohanthewiz/grmob/richtext"
)

// The text-edit protocol (core/text_edit.go) through the Manager, the way a
// native shell drives it: TriggerTextEdit in, editSeq/editEpoch out.
//
// tagApp is the shape that exposed the race on the Android emulator: a draft
// field that commits a tag and clears itself when a comma is typed. Plain
// variables stand in for state, because every dispatch renders anyway.
func tagApp(draft *string, tags *[]string) func(*core.Context) core.View {
	return func(ctx *core.Context) core.View {
		return core.Column(
			core.Input(*draft, "", func(v string) {
				if strings.HasSuffix(v, ",") {
					*tags = append(*tags, strings.TrimSuffix(v, ","))
					*draft = ""
					return
				}
				*draft = v
			}),
		)
	}
}

// fieldStamps returns the edit stamps an update-props patch for the field
// carried, as the JSON numbers the host reads, with the value it carried.
func fieldStamps(t *testing.T, out string) (seq, epoch float64, value any, ok bool) {
	t.Helper()
	for _, p := range decode(t, out) {
		if p.Type == "update-props" && p.TargetID == "root/0" {
			s, _ := p.Changes["editSeq"].(float64)
			e, _ := p.Changes["editEpoch"].(float64)
			// A RichTextEditor keeps its value under "doc".
			v, ok := p.Changes["value"]
			if !ok {
				v = p.Changes["doc"]
			}
			return s, e, v, true
		}
	}
	return 0, 0, nil, false
}

func TestAKeystrokeTypedBeforeARewriteIsDroppedAndItsRebaseApplies(t *testing.T) {
	var draft string
	var tags []string
	mgr := render.New(core.NewContext(), tagApp(&draft, &tags))
	defer mgr.Close()

	initial := mgr.RenderInitial()
	if strings.Contains(initial, "editSeq") || strings.Contains(initial, "editEpoch") {
		t.Fatal("a field no edit has reached must carry no edit stamps")
	}
	id := "txt_cb_0"

	// Echoes: the value Go renders is the one the host sent, so the epoch
	// stays 0 and only the ack moves.
	mgr.DispatchTextEdit(id, "b", 1, 0)
	out := mgr.DispatchTextEdit(id, "be", 2, 0)
	if seq, epoch, _, ok := fieldStamps(t, out); !ok || seq != 2 || epoch != 0 {
		t.Fatalf("echo: stamps = (%v, %v), want (2, 0); got %s", seq, epoch, out)
	}

	// The comma commits and clears: Go's value is not the host's, so it is a
	// rewrite and the epoch moves to 1, acknowledged at seq 3.
	out = mgr.DispatchTextEdit(id, "be,", 3, 0)
	seq, epoch, value, ok := fieldStamps(t, out)
	if !ok || seq != 3 || epoch != 1 || value != "" {
		t.Fatalf("rewrite: stamps = (%v, %v, %q), want (3, 1, \"\"); got %s", seq, epoch, value, out)
	}

	// The keystroke typed before the host saw the rewrite. At machine speed
	// it was already on its way, carrying epoch 0 and the old text. It must
	// not reach the handler: run as new, "be,g" would commit "be" a second
	// time.
	out = mgr.DispatchTextEdit(id, "be,g", 4, 0)
	if len(tags) != 1 || draft != "" {
		t.Fatalf("stale edit reached the handler: tags %q, draft %q", tags, draft)
	}
	// Not acknowledged, and no second rewrite. editSeq names the base of the
	// rewrite, so it must stay on seq 3: a host that took "be,g" as the base
	// would replay nothing of the "g". Nothing about the field changed, so
	// the render carries no patch for it at all.
	if _, _, _, ok := fieldStamps(t, out); ok {
		t.Fatalf("dropped edit patched the field: %s", out)
	}

	// The host's rebase of "be,g" onto "": "g", at the new epoch.
	out = mgr.DispatchTextEdit(id, "g", 5, 1)
	if draft != "g" || len(tags) != 1 || tags[0] != "be" {
		t.Fatalf("rebased edit: tags %q, draft %q, want [be] and g", tags, draft)
	}
	if seq, epoch, _, ok := fieldStamps(t, out); !ok || seq != 5 || epoch != 1 {
		t.Fatalf("rebased edit: stamps = (%v, %v), want (5, 1); got %s", seq, epoch, out)
	}
}

// A value Go refuses to change is a rewrite too. The old value queue never
// saw it, because the value on the wire did not change and so no patch came;
// the ack does change, and the epoch says the host's text is not Go's.
func TestARefusedEditIsARewrite(t *testing.T) {
	value := "abc"
	app := func(ctx *core.Context) core.View {
		return core.Column(core.Input(value, "", func(v string) {
			if len(v) <= 3 {
				value = v
			}
		}))
	}
	mgr := render.New(core.NewContext(), app)
	defer mgr.Close()
	mgr.RenderInitial()

	out := mgr.DispatchTextEdit("txt_cb_0", "abcd", 1, 0)
	seq, epoch, _, ok := fieldStamps(t, out)
	if !ok || seq != 1 || epoch != 1 {
		t.Fatalf("stamps = (%v, %v), want (1, 1); got %s", seq, epoch, out)
	}
}

// The web host dispatches through DispatchTextCallback, which has no seq to
// carry. It must never grow the stamps, or every web input would start
// receiving patches it has no use for.
func TestTheUnsequencedPathStampsNothing(t *testing.T) {
	var draft string
	var tags []string
	mgr := render.New(core.NewContext(), tagApp(&draft, &tags))
	defer mgr.Close()
	mgr.RenderInitial()

	for _, v := range []string{"a", "ab", "ab,"} {
		if out := mgr.DispatchTextCallback("txt_cb_0", v); strings.Contains(out, "edit") {
			t.Fatalf("DispatchTextCallback(%q) stamped the field: %s", v, out)
		}
	}
}

// A CodeEditor takes the same stamps as a field. Its value is plain text, so
// the rule is the field's unchanged: an echo moves only the ack, and a render
// that differs from what the host sent is a rewrite.
func TestACodeEditorCarriesTheEditStamps(t *testing.T) {
	src := "x := 1"
	app := func(ctx *core.Context) core.View {
		return core.Column(core.CodeEditor(src, func(v string) {
			// A formatter of the smallest kind: tabs become four spaces.
			src = strings.ReplaceAll(v, "\t", "    ")
		}, nil))
	}
	mgr := render.New(core.NewContext(), app)
	defer mgr.Close()
	if initial := mgr.RenderInitial(); strings.Contains(initial, "editSeq") {
		t.Fatalf("an editor no edit has reached must carry no stamps: %s", initial)
	}

	out := mgr.DispatchTextEdit("txt_cb_0", "x := 12", 1, 0)
	if seq, epoch, _, ok := fieldStamps(t, out); !ok || seq != 1 || epoch != 0 {
		t.Fatalf("echo: stamps = (%v, %v), want (1, 0); got %s", seq, epoch, out)
	}
	out = mgr.DispatchTextEdit("txt_cb_0", "x := 12\n\t", 2, 0)
	seq, epoch, value, ok := fieldStamps(t, out)
	if !ok || seq != 2 || epoch != 1 || value != "x := 12\n    " {
		t.Fatalf("rewrite: stamps = (%v, %v, %q), want (2, 1, the spaces); got %s", seq, epoch, value, out)
	}
}

// A RichTextEditor's host sends its own JSON of the document, which is not
// Go's byte for byte: org.json escapes the slash, and neither host is bound
// to encoding/json's key order or its omission of empty fields. The same
// document in another spelling is an echo, not a rewrite. Were it a rewrite,
// the host would rebuild its attributed text on every keystroke.
func TestARichTextEchoInTheHostsSpellingIsNotARewrite(t *testing.T) {
	var doc richtext.Doc
	app := func(ctx *core.Context) core.View {
		return core.Column(core.RichTextEditor(doc, func(d richtext.Doc) { doc = d }))
	}
	mgr := render.New(core.NewContext(), app)
	defer mgr.Close()
	mgr.RenderInitial()

	hostSpelling := `{"b":[{"r":[{"b":0,"t":"a\/b"}],"k":"p"}]}`
	out := mgr.DispatchTextEdit("txt_cb_0", hostSpelling, 1, 0)
	seq, epoch, value, ok := fieldStamps(t, out)
	if !ok || seq != 1 || epoch != 0 {
		t.Fatalf("stamps = (%v, %v), want (1, 0); got %s", seq, epoch, out)
	}
	if value != `{"b":[{"r":[{"t":"a/b"}]}]}` {
		t.Fatalf("doc = %v, want Go's spelling of the same document", value)
	}

	// The next keystroke is compared with Go's bytes, which the ledger kept.
	out = mgr.DispatchTextEdit("txt_cb_0", `{"b":[{"r":[{"t":"a\/bc"}]}]}`, 2, 0)
	if seq, epoch, _, ok := fieldStamps(t, out); !ok || seq != 2 || epoch != 0 {
		t.Fatalf("second echo: stamps = (%v, %v), want (2, 0); got %s", seq, epoch, out)
	}

	// A different document from Go is still a rewrite.
	doc = richtext.Doc{Blocks: []richtext.Block{{Kind: richtext.Paragraph, Runs: []richtext.Run{{Text: "reset"}}}}}
	out = mgr.DispatchTextEdit("txt_cb_1", "", 3, 0) // any dispatch renders; the ID is unknown
	if _, epoch, _, ok := fieldStamps(t, out); !ok || epoch != 1 {
		t.Fatalf("rewrite: epoch = %v, want 1; got %s", epoch, out)
	}
}

// A payload the document parser cannot read never reaches the app, so Go's
// document stands, and the render says so with a new epoch.
func TestAnUnreadableRichTextEditIsARewrite(t *testing.T) {
	doc := richtext.Doc{Blocks: []richtext.Block{{Runs: []richtext.Run{{Text: "keep"}}}}}
	app := func(ctx *core.Context) core.View {
		return core.Column(core.RichTextEditor(doc, func(d richtext.Doc) { doc = d }))
	}
	mgr := render.New(core.NewContext(), app)
	defer mgr.Close()
	mgr.RenderInitial()

	out := mgr.DispatchTextEdit("txt_cb_0", `{"b":[`, 1, 0)
	if seq, epoch, _, ok := fieldStamps(t, out); !ok || seq != 1 || epoch != 1 {
		t.Fatalf("stamps = (%v, %v), want (1, 1); got %s", seq, epoch, out)
	}
}

// changesFor returns the update-props changes a render carried for one node.
func changesFor(t *testing.T, out, target string) map[string]any {
	t.Helper()
	for _, p := range decode(t, out) {
		if p.Type == "update-props" && p.TargetID == target {
			return p.Changes
		}
	}
	return nil
}

// comps.PINInput, when it was one field per box, typed at about 130ms a key on
// the Android emulator lost digits: 314159 arrived as 3 1 4 5 9. Two fields
// are involved, and the one that loses the key has never been edited, which is
// why a ledger made by a field's first edit could not see it.
//
// PINInput is one field now (comps/pin_input.go), but the rule this pins is
// core's and any screen can hit it: a field Go rewrites before its first edit.
// So a three-box field is built here, with the mechanics the old widget had:
// a second character typed into box 0 spills into box 1. The sequence is the
// emulator's, one dispatch per host edit, and what the host's rebase sends.
func TestAFieldGoRewroteBeforeItsFirstEditDropsTheStaleKey(t *testing.T) {
	var code string
	at := func(i int) string {
		r := []rune(code)
		if i < len(r) {
			return string(r[i])
		}
		return ""
	}
	// Box i writes what it reports from position i on, capped at three.
	write := func(i int) func(string) {
		return func(v string) {
			r := []rune(code)
			if i > len(r) {
				i = len(r)
			}
			next := string(r[:i]) + v
			if len([]rune(next)) > 3 {
				next = string([]rune(next)[:3])
			}
			code = next
		}
	}
	app := func(ctx *core.Context) core.View {
		return core.Column(
			core.Input(at(0), "", write(0)),
			core.Input(at(1), "", write(1)),
			core.Input(at(2), "", write(2)),
		)
	}
	mgr := render.New(core.NewContext(), app)
	defer mgr.Close()
	if initial := mgr.RenderInitial(); strings.Contains(initial, "editSeq") {
		t.Fatal("no field is stamped before the host has sent an edit")
	}
	box0, box1 := "txt_cb_0", "txt_cb_1"

	// "3" into box 0. From here the host is sequenced, and every field is
	// stamped from the render on.
	out := mgr.DispatchTextEdit(box0, "3", 1, 0)
	if c := changesFor(t, out, "root/1"); c == nil || c["editEpoch"] != float64(0) {
		t.Fatalf("box 1 should be stamped at epoch 0 once the host is sequenced: %v in %s", c, out)
	}

	// "1" lands in box 0 too, because focus has not moved yet: box 0 reports
	// "31". Go writes the "1" into box 1, which is a rewrite of a field the
	// host shows as "".
	out = mgr.DispatchTextEdit(box0, "31", 2, 0)
	if code != "31" {
		t.Fatalf("code = %q, want 31", code)
	}
	if c := changesFor(t, out, "root/1"); c == nil || c["value"] != "1" || c["editEpoch"] != float64(1) {
		t.Fatalf("box 1 = %v, want value 1 at epoch 1; got %s", c, out)
	}

	// "4", typed into box 1 before that render reached the host. On the old
	// text, so it must not reach the handler: applied, it would overwrite the 1.
	mgr.DispatchTextEdit(box1, "4", 3, 0)
	if code != "31" {
		t.Fatalf("stale key applied: code = %q, want 31", code)
	}

	// The host's rebase of "4" onto box 1's "1" (internal/rebasefixture's "a
	// PIN cell Go filled under a typed key"): box 1 reports "14", which the
	// field writes from box 1 on.
	mgr.DispatchTextEdit(box1, "14", 4, 1)
	if code != "314" {
		t.Fatalf("rebased key: code = %q, want 314", code)
	}
}
