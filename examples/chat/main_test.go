package main

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/htmlout"
)

// The chat example, and in particular core.RoleLog — which this file is the
// only consumer of in the repository, and which until now had no test anywhere.
//
// That combination is the reason this file exists. A role costs nothing to set
// and is inert until something reads it, so "the one place it is used stopped
// using it" is a change that compiles, renders, and shows up nowhere: the
// bubbles still draw, the send still works, and a screen reader simply stops
// being told that new messages arrived. The coverage pins in mobile/verify hold
// each native's *dispatch* against core.Roles(), and htmlout proves every role
// becomes an attribute — neither can see whether any app ever asks for one.
//
// So what is asserted here is the consumer's half: that the transcript claims
// to be a log, that the claim is on the element it is true of, and that it
// survives the render pass into the document a browser would read.

// TestMain turns on debug mode for the package, so every pass driven below is
// audited for cursor drift and duplicate keys — the same discipline the other
// example tests follow. ChatApp allocates two hooks unconditionally at the top,
// which is exactly what that audit checks has not drifted.
func TestMain(m *testing.M) {
	core.SetDebugMode(true)
	m.Run()
}

// pass renders one hand-rolled pass against a fresh context and returns the
// tree, mirroring what main() does between its two prints.
func pass(t *testing.T, ctx *core.Context) *core.Node {
	t.Helper()
	n := renderPass(ctx)
	if n == nil {
		t.Fatal("render pass produced no tree")
	}
	return n
}

func find(n *core.Node, pred func(*core.Node) bool) *core.Node {
	if n == nil {
		return nil
	}
	if pred(n) {
		return n
	}
	for _, c := range n.Children {
		if got := find(c, pred); got != nil {
			return got
		}
	}
	return nil
}

func roled(role core.Role) func(*core.Node) bool {
	return func(n *core.Node) bool {
		return n.Style != nil && n.Style.AccessibilityRole == role
	}
}

// countText walks the tree and counts Text nodes whose content is s. Used to
// tell "the message is in the tree once" from "the message is in the tree
// twice", which is the failure an append-in-place bug produces.
func countText(n *core.Node, s string) int {
	if n == nil {
		return 0
	}
	total := 0
	if n.Type == "Text" && n.Props["content"] == s {
		total++
	}
	for _, c := range n.Children {
		total += countText(c, s)
	}
	return total
}

// The role is on the element whose children change, which is the whole of what
// a live region means: a reader watches one element and announces what appears
// inside it. MessageList's Scroll is a viewport onto that element and gains and
// loses nothing itself, so a log role there would name something that never
// changes.
//
// The check is written as "the log is a Column with a Scroll above it" rather
// than "the Scroll has no role", because the second passes for a tree that has
// no log role anywhere at all.
func TestTheTranscriptIsALogOnTheColumnThatGrows(t *testing.T) {
	ctx := core.NewContext().WithTheme(core.DefaultTheme)
	tree := pass(t, ctx)

	log := find(tree, roled(core.RoleLog))
	if log == nil {
		t.Fatal("no core.RoleLog in the tree: new messages are announced by nothing, and " +
			"this example is the role's only consumer")
	}
	if log.Type != "Column" {
		t.Errorf("the log role is on a %s; it belongs on the element whose children change, "+
			"not on the viewport onto it", log.Type)
	}
	scroll := find(tree, func(n *core.Node) bool { return n.Type == "Scroll" })
	if scroll == nil {
		t.Fatal("no Scroll around the transcript")
	}
	if find(scroll, roled(core.RoleLog)) != log {
		t.Error("the log is not inside the Scroll: the two have come apart")
	}
	if scroll.Style != nil && scroll.Style.AccessibilityRole != core.RoleNone {
		t.Errorf("the Scroll carries role %q; a viewport is not the live region",
			scroll.Style.AccessibilityRole)
	}
}

// `log` and not `status`, which is the distinction the role was added for and
// the one a future edit is most likely to erase — the two are the same call on
// Android and differ only on the web, so nothing on a phone would report it.
//
// status is one advisory that is *replaced*: a reader announces the region's
// new state and the old text is gone. log is a record that is *appended to* and
// whose order is meaningful, so what arrived is announced and everything before
// it is still there to be read back. A transcript marked status announces
// correctly and reads back as one region that just changed entirely.
func TestTheTranscriptIsALogAndNotAStatus(t *testing.T) {
	ctx := core.NewContext().WithTheme(core.DefaultTheme)
	tree := pass(t, ctx)

	if find(tree, roled(core.RoleStatus)) != nil {
		t.Error("the transcript is marked status: a conversation is appended to and read " +
			"back in order, which is what distinguishes the two roles on the web")
	}
	if find(tree, roled(core.RoleAlert)) != nil {
		t.Error("the transcript is marked alert: a message arriving would cut into whatever " +
			"the reader was saying")
	}
}

// The role survives the export, which is the target it exists for. Both natives
// collapse log onto Compose's polite live region or drop it (SwiftUI announces
// through an imperative call, not a view property), so the web is where the
// distinction is actually spent — see core/role.go.
func TestTheLogRoleReachesTheDocument(t *testing.T) {
	ctx := core.NewContext().WithTheme(core.DefaultTheme)
	html := htmlout.ExportHTML(pass(t, ctx))

	if !strings.Contains(html, `role="log"`) {
		t.Fatalf("no role=\"log\" in the exported document:\n%s", html)
	}
	if strings.Count(html, `role="log"`) != 1 {
		t.Errorf("expected exactly one live region:\n%s", html)
	}
}

// The example's other lesson, and the reason the log has anything to announce:
// every write to the thread goes through `send`, which trims, appends onto a
// *copy*, and clears the composer.
//
// Driven the way main() drives it — a text callback then a tap — because the
// callback IDs are a per-pass contract and re-deriving them from a fresh tree
// is what the native shells do.
func TestSendAppendsOneMessageAndClearsTheComposer(t *testing.T) {
	ctx := core.NewContext().WithTheme(core.DefaultTheme)
	before := pass(t, ctx)

	if n := countText(before, "Vou experimentar hoje!"); n != 0 {
		t.Fatalf("the message is in the tree before it was sent (%d times)", n)
	}

	ctx.ReceiveEventPayload(map[string]any{"callback": "txt_cb_0", "value": "Vou experimentar hoje!"})
	ctx.ReceiveEventPayload(map[string]any{"callback": "cb_1"})
	after := pass(t, ctx)

	// Once, not twice. Appending in place onto the slice the state already
	// holds would leave the previous tree pointing at the new backing array,
	// which is the bug the copy in `send` exists to prevent.
	if n := countText(after, "Vou experimentar hoje!"); n != 1 {
		t.Errorf("the sent message appears %d times in the tree, want 1", n)
	}
	// The seed thread is still there and still in order: a log's contents are
	// read back, so losing the history is a different failure from failing to
	// announce.
	for _, seeded := range []string{
		"Já viste a nova versão do GrMob?",
		"Ainda não — o que mudou?",
		"Componentes, cache e modo de depuração 🎉",
	} {
		if countText(after, seeded) != 1 {
			t.Errorf("the transcript lost %q", seeded)
		}
	}

	// The composer is empty again — the third thing `send` owns, and the one a
	// caller re-deriving the send path at a second call site forgets.
	input := find(after, func(n *core.Node) bool { return n.Type == "Input" })
	if input == nil {
		t.Fatal("no Input in the composer")
	}
	if v, _ := input.Props["value"].(string); v != "" {
		t.Errorf("composer still holds %q after sending", v)
	}
}

// An empty composer is a no-op, not an empty bubble. The guard is one line in
// `send` and it is the only thing between a tapped Enviar and a blank message
// in the transcript.
func TestSendingNothingChangesNothing(t *testing.T) {
	ctx := core.NewContext().WithTheme(core.DefaultTheme)
	pass(t, ctx)

	// Whitespace, not the empty string: the guard trims first, which is what
	// makes a composer holding a space behave like one holding nothing.
	ctx.ReceiveEventPayload(map[string]any{"callback": "txt_cb_0", "value": "   "})
	ctx.ReceiveEventPayload(map[string]any{"callback": "cb_1"})
	after := pass(t, ctx)

	log := find(after, roled(core.RoleLog))
	if log == nil {
		t.Fatal("no log region after the no-op send")
	}
	// Three seeded messages, and core.For groups its output in a Fragment, so
	// the count is taken over the bubbles rather than over the log's children.
	if got := len(find(log, func(n *core.Node) bool { return n.Type == "Fragment" }).Children); got != 3 {
		t.Errorf("thread holds %d messages after sending whitespace, want the 3 it started "+
			"with", got)
	}
}

// Debug mode is on for the package, so the audit ran on every pass above. This
// asserts it found nothing — hooks allocated unconditionally and in a fixed
// order, and no two rows claiming the same key.
//
// The keys are core.Keyed("msg-"+ID), which is what lets the reconciler match
// an old row to its message when the thread grows rather than rebuilding the
// slot. A duplicate would be reported here rather than seen.
func TestNoDebugConcernsAcrossAConversation(t *testing.T) {
	core.ClearConcerns()

	ctx := core.NewContext().WithTheme(core.DefaultTheme)
	pass(t, ctx)
	ctx.ReceiveEventPayload(map[string]any{"callback": "txt_cb_0", "value": "olá"})
	ctx.ReceiveEventPayload(map[string]any{"callback": "cb_1"})
	pass(t, ctx)

	if cs := core.Concerns(); len(cs) != 0 {
		t.Fatalf("debug concerns raised:\n%s", core.DumpConcerns())
	}
}
