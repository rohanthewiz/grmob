// Command chat is the message-thread example: a scrolling conversation with
// sent/received bubbles and a composer row.
//
// What it exists to teach, and nothing else:
//
//   - core.For + core.Keyed — building a list from data, with the stable row
//     identity the reconciler needs to keep a row attached to its message when
//     the slice grows.
//   - core.UseStyle — one Style value carrying a whole visual role (the bubble),
//     rather than a scatter of individual style props.
//   - a single mutation choke point — every write to the thread goes through
//     `send`, so there is exactly one place where the message list changes.
//
// It is stateful, so like examples/runtime it drives two render passes by hand
// and prints the HTML before and after a simulated send. The feed/social UI
// this file used to hold now lives only in examples/social, which is the
// example that owns it.
package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/rohanthewiz/grmob/components"
	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/htmlout"
)

// Message is one line of the conversation. From is the sender's display name;
// the empty string means "us", which is what drives the sent-vs-received
// styling below.
type Message struct {
	ID   string
	From string
	Text string
}

func (m Message) Mine() bool { return m.From == "" }

func seedThread() []Message {
	return []Message{
		{ID: "1", From: "Ana", Text: "Já viste a nova versão do GrMob?"},
		{ID: "2", From: "", Text: "Ainda não — o que mudou?"},
		{ID: "3", From: "Ana", Text: "Componentes, cache e modo de depuração 🎉"},
	}
}

func ChatApp(ctx *core.Context) core.View {
	// Both hooks are allocated unconditionally, at the top, in a fixed order:
	// slots are positional, so a hook behind an `if` would shift every slot
	// after it the moment the condition flips.
	thread := core.NewState(ctx, seedThread())
	draft := core.NewState(ctx, "")

	// send is the only writer of the thread. Routing every mutation through one
	// helper is what keeps a growing app's state honest: the trimming rule, the
	// ID scheme, and the "clear the composer afterwards" step live here once,
	// instead of being re-derived at each call site (the submit key and the send
	// button are already two call sites).
	send := func() {
		text := strings.TrimSpace(draft.Get())
		if text == "" { // an empty composer is a no-op, not an empty bubble
			return
		}
		msgs := thread.Get()
		// Append onto a copy rather than in place: State.Set must be handed a
		// new value. Mutating the slice the state already holds would change
		// what the previous tree points at, and the reconciler diffs the two
		// trees against each other.
		next := make([]Message, len(msgs), len(msgs)+1)
		copy(next, msgs)
		next = append(next, Message{ID: strconv.Itoa(len(msgs) + 1), Text: text})
		thread.Set(next)
		draft.Set("")
	}

	// Screen.Scroll stays false: the scrolling region here is MessageList's
	// own Scroll, which is what lets the header and composer stay put while
	// only the thread moves. A scrolling screen would put a scroll view inside
	// a scroll view, and the two would fight over the same drag natively.
	return components.Screen{
		Children: []core.View{
			ThreadHeader("Ana"),
			MessageList(thread.Get()),
			Composer(draft, send),
		},
	}
}

func ThreadHeader(who string) core.View {
	return core.Row(
		core.BackgroundColor("#FFFFFF"),
		core.Padding(16),
		core.Text(who, core.FontSize(18), core.FontWeight(core.Bold)),
	)
}

// MessageList renders the thread. core.For turns the slice into views, and
// core.Keyed gives each row an identity derived from the message rather than
// from its position: when a message is appended (or one day inserted), keyed
// rows let the reconciler match old row to new row instead of rebuilding the
// slot — which on native means the row keeps its recycled view and any
// transient state attached to it.
//
// The row spacing rides on the bubble (a bottom margin), not on a Gap on this
// Column: core.For groups its output in a single Fragment child, so a Gap here
// would space the Fragment against its siblings, not the messages inside it.
func MessageList(msgs []Message) core.View {
	return core.Scroll(
		core.Column(
			core.Padding(12),
			core.For(msgs, func(m Message, _ int) core.View {
				return core.Keyed("msg-"+m.ID, MessageBubble(m))
			}),
		),
	)
}

// MessageBubble is the sent/received styling, expressed as two Style values
// chosen by one predicate.
//
// core.UseStyle takes a whole Style at once, which is the right shape when a
// group of properties travels together as a visual role — here "our bubble" vs
// "their bubble". Individual style props (core.Padding, core.FontSize, ...)
// remain the right shape for one-off adjustments; both compose, and later props
// win over earlier ones.
func MessageBubble(m Message) core.View {
	bubble := core.Style{
		Background:   "#E9E9EB",
		TextColor:    "#000000",
		BorderRadius: 16,
		Padding:      core.EdgeInsets{Top: 8, Bottom: 8, Left: 12, Right: 12},
		Gap:          2,
	}
	// Which end of the row the bubble sits at. JustifyContent (not Align) is
	// the main-axis property: on a Row it is what Compose maps to
	// Arrangement.End and CSS to justify-content, whereas Align is the cross
	// axis.
	side := core.JustifyStart
	if m.Mine() {
		bubble.Background = core.PrimaryColor()
		bubble.TextColor = "#FFFFFF"
		side = core.JustifyEnd
	}

	return core.Row(
		core.Justify(side),
		// The gap between consecutive messages; see MessageList for why it
		// lives here rather than on the list container.
		core.UseStyle(core.Style{Margin: core.EdgeInsets{Bottom: 8}}),
		core.Column(
			core.UseStyle(bubble),
			// Only their messages are labelled: our own name on our own
			// bubbles is noise. core.MaybeProp and not core.If, because a
			// false core.If still returns a Fragment, and an empty Fragment
			// is a real child — it would take a slot in the bubble's own flex
			// layout and open a stray gap where the label is absent.
			// MaybeProp's false path is an untyped nil, which the container
			// skips outright, so the tree is identical to one written without
			// the label at all.
			core.MaybeProp(!m.Mine(),
				core.Text(m.From, core.FontSize(12), core.FontWeight(core.Bold))),
			core.Text(m.Text, core.FontSize(15)),
		),
	)
}

// Composer is the input row. InputWithSubmit registers both an onChange (every
// keystroke, keeping the field controlled by Go state) and an onSubmit (the
// keyboard's send key) — the button is the third path to the same helper.
func Composer(draft core.State[string], send func()) core.View {
	return core.Row(
		core.BackgroundColor("#FFFFFF"),
		core.Padding(12),
		core.Gap(8),
		core.InputWithSubmit(draft.Get(), "Mensagem…",
			func(val string) { draft.Set(val) },
			send,
			core.FlexGrow(1),
		),
		// The theme's Button base already is a filled Primary with padding
		// and a radius, so this re-spelled it by hand in four props. The
		// widget's zero value applies nothing at all, which is how the same
		// look survives a theme swap instead of staying pinned to these.
		components.Button{Label: "Enviar", OnTap: send},
	)
}

// renderPass drives one hand-rolled render pass, the same boundary
// examples/runtime documents in full: Begin (callback IDs restart), Render
// (which Resets the hook cursor), End (debug audit), Purge (drop handlers not
// re-registered this pass).
func renderPass(ctx *core.Context) *core.Node {
	ctx.BeginRenderPass()
	node := core.Render(ctx, core.ComponentFunc(func(ctx *core.Context) *core.Node {
		return ChatApp(ctx).Render(ctx)
	}))
	ctx.EndRenderPass()
	ctx.PurgeUnusedCallbacks()
	return node
}

func main() {
	ctx := core.NewContext().WithTheme(core.DefaultTheme)

	tree := renderPass(ctx)
	fmt.Println("Conversa inicial:")
	fmt.Println(htmlout.ExportHTML(tree))

	// Simulate the native side sending two events for one message: the text
	// change from the field, then the tap on "Enviar". The callback IDs are the
	// ones the pass above registered — "txt_cb_0" is the composer's onChange
	// (the only text callback in the tree) and "cb_1" is the send button
	// (onSubmit takes cb_0, registered first).
	ctx.ReceiveEventPayload(map[string]any{"callback": "txt_cb_0", "value": "Vou experimentar hoje!"})
	ctx.ReceiveEventPayload(map[string]any{"callback": "cb_1"})

	tree = renderPass(ctx)
	fmt.Println("\nApós enviar:")
	fmt.Println(htmlout.ExportHTML(tree))
}
