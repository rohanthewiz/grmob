package comps

import "github.com/rohanthewiz/grmob/core"

// ThreadMessage is one message in a MessageThread.
type ThreadMessage struct {
	// Key identifies the message for as long as it exists: an ID from the
	// server, not an index. It is the row's identity, and the only thing
	// that tells a host which row the reader was looking at when an older
	// page lands above it.
	Key string

	Text   string
	Sender string
	Mine   bool
	// Time is drawn as given; see MessageBubble.Time.
	Time string
}

// MessageThread is a conversation: MessageBubbles in a List that opens on the
// newest message, loads older ones as the reader scrolls back to the top, and
// keeps the reader's place while they land.
//
//	comps.MessageThread{
//	    Messages:    thread.Get(),           // oldest first
//	    OnLoadOlder: loadOlder,              // nil once there is nothing older
//	    Loading:     fetching.Get(),
//	}
//
//	  "Loading older messages…" | the start caption | (blank)   ← one line,
//	┌ List  Height, StartAtEnd, OnStartReached, RoleLog ─────┐     always there
//	│  Ana     Já viste a nova versão?                        │
//	│          Ainda não                            (mine)    │
//	│  Ana     Saiu ontem                                     │
//	│          (Continued: the sender line not drawn)         │
//	└─────────────────────────────────────────────────────────┘
//	                                              opens here ▲
//
// # What the hosts do, and what this does
//
// Opening at the end, reporting the top, and keeping the reader's place on a
// prepend are the host's (core.StartAtEnd and core.OnStartReached, whose doc
// has the per-host table). What is here is the transcript:
//
//   - Each bubble is Keyed by its message's Key, which is what the hosts'
//     place-keeping reads. Without keys every prepend would look, to every
//     host, like new text in every row.
//   - A run of messages from one sender draws the sender once
//     (MessageBubble.Continued), and still speaks it on each.
//   - The loading and start captions are a line above the List, not a row
//     in it. As a row it broke both halves of the top edge. The guard counts
//     rows, and a loading row that came and went reopened it mid-fetch. And
//     every host keeps the reader's place by the first visible row, which at
//     the top was the caption row, still first after the prepend: the reader
//     stayed at the top, looking at the older page, and the edge fired for
//     the page before that, and so on to the beginning. Outside the List,
//     the first row is always a message. The line is there when it is blank
//     too, so the box does not move when the caption appears.
//   - The list is a core.RoleLog, the role for a transcript: what arrives is
//     announced, and what came before stays in order (see examples/chat).
//
// # Loading
//
// OnLoadOlder runs when the reader reaches the top, once per row count. The
// caller fetches, sets Loading while it does, and prepends what came back.
// A fetch that returns nothing leaves the count unchanged and the guard shut,
// so a thread at its true beginning is not asked again; set OnLoadOlder to nil
// then, and the caption line says the conversation starts here.
type MessageThread struct {
	Messages []ThreadMessage

	// OnLoadOlder asks for the page before the first message. Nil means
	// there is none, and the caption line says so.
	OnLoadOlder func()

	// Loading shows "Loading older messages…" in the caption line.
	Loading bool

	// Height is the thread's viewport; a List needs a bounded height to
	// scroll. Empty gives "360px".
	Height string

	// LoadingText and StartText replace the caption line's two captions.
	LoadingText string
	StartText   string

	// MineLabel is passed to each bubble; see MessageBubble.MineLabel.
	MineLabel string

	// Style is applied to the List (the scrolling box, not the caption
	// line above it) after its defaults.
	Style []core.StyleProp
}

func (m MessageThread) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	items := make([]core.PropsAndChildren, 0, len(m.Messages)+len(m.Style)+10)
	items = append(items,
		core.Height(orDefault(m.Height, "360px")),
		// The web's List is a plain column; the natives' List scrolls by
		// itself. Overflow is what makes the web one scroll inside its box.
		core.Overflow("auto"),
		core.Padding(12),
		core.Gap(8),
		// The role and no name. A name on a container is how a row becomes
		// one stop (SwiftUI's accessibilityElement(children: .combine), and
		// grMobBox applies it to any labelled node), and a transcript read as
		// one stop is every message at once. examples/chat's log carries no
		// name for the same reason.
		core.AccessibilityRole(core.RoleLog),
		core.StartAtEnd(),
	)
	if m.OnLoadOlder != nil {
		items = append(items, core.OnStartReached(m.OnLoadOlder))
	}
	items = append(items, asProps(m.Style)...)

	for i, msg := range m.Messages {
		// A run: the same sender as the message above, neither of them the
		// reader's own (Mine draws no sender line to repeat).
		continued := i > 0 && !msg.Mine && !m.Messages[i-1].Mine &&
			msg.Sender != "" && msg.Sender == m.Messages[i-1].Sender
		items = append(items, core.Keyed("msg:"+msg.Key, MessageBubble{
			Text:      msg.Text,
			Sender:    msg.Sender,
			Mine:      msg.Mine,
			Time:      msg.Time,
			MineLabel: m.MineLabel,
			Continued: continued,
		}))
	}
	return core.Column(
		core.Padding(0),
		core.Gap(4),
		m.captionLine(t),
		core.List(items...),
	).Render(ctx)
}

// captionLine is the line above the List: the loading caption, the
// start-of-conversation caption, or, while there is more to load and nothing
// in flight, a no-break space in the same style. The blank line keeps its
// height, so the thread's box stays put when the caption appears, and it is
// hidden from screen readers because it says nothing.
func (m MessageThread) captionLine(t *core.Theme) core.View {
	text, hidden := "\u00a0", true
	switch {
	case m.Loading:
		text, hidden = orDefault(m.LoadingText, "Loading older messages…"), false
	case m.OnLoadOlder == nil:
		text, hidden = orDefault(m.StartText, "This is the start of the conversation."), false
	}
	props := []core.StyleProp{
		core.UseStyle(t.Typography.Caption),
		core.TextColor(t.Colors.TextSecondary),
		core.Align(core.AlignCenter),
		core.Width("100%"),
	}
	if hidden {
		props = append(props, core.AccessibilityHidden())
	}
	return core.Text(text, props...)
}
