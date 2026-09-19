package comps

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

func threadMessages() []ThreadMessage {
	return []ThreadMessage{
		{Key: "1", Sender: "Ana", Text: "Já viste a nova versão?", Time: "10:42"},
		{Key: "2", Sender: "Ana", Text: "Saiu ontem", Time: "10:42"},
		{Key: "3", Mine: true, Text: "Ainda não", Time: "10:43"},
		{Key: "4", Sender: "Ana", Text: "Vê!", Time: "10:44"},
	}
}

// threadList is the thread's List: the Column's second child, under the
// caption line.
func threadList(t *testing.T, n *core.Node) *core.Node {
	t.Helper()
	if n.Type != "Column" || len(n.Children) != 2 || n.Children[1].Type != "List" {
		t.Fatalf("want Column(caption, List), got %q with %d children", n.Type, len(n.Children))
	}
	return n.Children[1]
}

// The List: bounded, opening at the end, an unnamed log, the top edge
// wired to OnLoadOlder, and nothing in it but the messages, each keyed.
func TestMessageThreadIsAKeyedLogThatOpensAtTheEnd(t *testing.T) {
	_, root := renderDebug(t, MessageThread{Messages: threadMessages(), OnLoadOlder: func() {}})
	n := threadList(t, root)
	if n.Style.Height != "360px" {
		t.Errorf("height = %q, want the 360px default", n.Style.Height)
	}
	if n.Props["startAtEnd"] != true {
		t.Error("the thread does not open at the end")
	}
	if id, _ := n.Props["onStartReached"].(string); id == "" {
		t.Error("OnLoadOlder is not wired to the top edge")
	}
	// A log, and unnamed: a name on a container folds its children into one
	// stop on iOS, and a transcript is read message by message.
	if n.Style.AccessibilityRole != core.RoleLog || n.Style.AccessibilityLabel != "" {
		t.Errorf("role %q label %q, want an unnamed log", n.Style.AccessibilityRole, n.Style.AccessibilityLabel)
	}
	want := []string{"msg:1", "msg:2", "msg:3", "msg:4"}
	if len(n.Children) != len(want) {
		t.Fatalf("%d rows, want %d", len(n.Children), len(want))
	}
	for i, k := range want {
		if n.Children[i].Key != k {
			t.Errorf("row %d key %q, want %q", i, n.Children[i].Key, k)
		}
	}
}

// A run from one sender draws the name once and speaks it on every bubble.
// Mine breaks a run, and a sender after it is drawn again.
func TestMessageThreadDrawsARunsSenderOnce(t *testing.T) {
	_, root := renderDebug(t, MessageThread{Messages: threadMessages()})
	n := threadList(t, root)
	drawn := func(i int) bool { return findText(bubbleOf(t, n.Children[i]), "Ana") != nil }
	if !drawn(0) {
		t.Error("the first of Ana's run should draw her name")
	}
	if drawn(1) {
		t.Error("the second of Ana's run should not draw her name again")
	}
	if got := bubbleOf(t, n.Children[1]).Style.AccessibilityLabel; got != "Ana, Saiu ontem, 10:42" {
		t.Errorf("the continued bubble is named %q; the sender must still be spoken", got)
	}
	if !drawn(3) {
		t.Error("Ana after the reader's own message starts a new run and draws her name")
	}
}

// The List holds only messages, loading or not, so the guard cannot reopen
// mid-fetch and the first row, which every host keeps the place by, is always
// a message. The captions are the line above, which is there even blank.
func TestMessageThreadCaptionsAreALineAboveTheList(t *testing.T) {
	msgs := threadMessages()
	render := func(th MessageThread) (*core.Node, *core.Node) {
		_, root := renderDebug(t, th)
		return root.Children[0], threadList(t, root)
	}
	idleLine, idle := render(MessageThread{Messages: msgs, OnLoadOlder: func() {}})
	loadingLine, loading := render(MessageThread{Messages: msgs, OnLoadOlder: func() {}, Loading: true})
	startLine, start := render(MessageThread{Messages: msgs})
	if len(idle.Children) != len(msgs) || len(loading.Children) != len(msgs) || len(start.Children) != len(msgs) {
		t.Fatalf("rows idle %d, loading %d, at the start %d: want %d each, the messages alone",
			len(idle.Children), len(loading.Children), len(start.Children), len(msgs))
	}
	if idleLine.Props["content"] != "\u00a0" || !idleLine.Style.AccessibilityHidden {
		t.Errorf("the idle line should be a hidden no-break space, got %q", idleLine.Props["content"])
	}
	if loadingLine.Props["content"] != "Loading older messages…" || loadingLine.Style.AccessibilityHidden {
		t.Errorf("loading line %q, want the spoken loading caption", loadingLine.Props["content"])
	}
	if startLine.Props["content"] != "This is the start of the conversation." {
		t.Errorf("start line %q", startLine.Props["content"])
	}
	if _, ok := start.Props["onStartReached"]; ok {
		t.Error("with nothing older the top edge is still wired")
	}
}

// Loading older runs the handler once per row count, through the real
// callback, and a prepended page reopens it.
func TestMessageThreadLoadsOlderOncePerPage(t *testing.T) {
	ctx := core.NewContext()
	loads := 0
	render := func(msgs []ThreadMessage) string {
		ctx.BeginRenderPass()
		root := MessageThread{Messages: msgs, OnLoadOlder: func() { loads++ }}.Render(ctx)
		ctx.PurgeUnusedCallbacks()
		return threadList(t, root).Props["onStartReached"].(string)
	}
	msgs := threadMessages()
	id := render(msgs)
	ctx.TriggerCallback(id)
	ctx.TriggerCallback(id)
	if loads != 1 {
		t.Fatalf("two reports of the top loaded %d pages, want 1", loads)
	}
	older := append([]ThreadMessage{{Key: "0", Sender: "Ana", Text: "Olá"}}, msgs...)
	ctx.TriggerCallback(render(older))
	if loads != 2 {
		t.Fatalf("after an older page landed, %d loads in total, want 2", loads)
	}
}
