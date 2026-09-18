package render_test

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/render"
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
// carried, as the JSON numbers the host reads.
func fieldStamps(t *testing.T, out string) (seq, epoch float64, value any, ok bool) {
	t.Helper()
	for _, p := range decode(t, out) {
		if p.Type == "update-props" && p.TargetID == "root/0" {
			s, _ := p.Changes["editSeq"].(float64)
			e, _ := p.Changes["editEpoch"].(float64)
			return s, e, p.Changes["value"], true
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
