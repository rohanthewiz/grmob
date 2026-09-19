package comps

import "github.com/rohanthewiz/grmob/core"

// MessageBubble is one message in a conversation: a rounded box of text held
// to one side of the row — the trailing side for the reader's own, the
// leading side for everyone else's — with an optional sender line above the
// text and a time under it.
//
//	comps.MessageBubble{Text: "Já viste a nova versão?", Sender: "Ana", Time: "10:42"}
//	comps.MessageBubble{Text: "Ainda não", Mine: true, Time: "10:43"}
//
//	┌ Row  justify=start ───────────────────────────────────┐
//	│ ┌ Column  Surface + hairline ─────┐                   │  theirs
//	│ │ Ana                             │  bold caption     │
//	│ │ Já viste a nova versão?         │                   │
//	│ │                          10:42  │  caption, end     │
//	│ └─────────────────────────────────┘                   │
//	└───────────────────────────────────────────────────────┘
//	┌ Row  justify=end ─────────────────────────────────────┐
//	│                   ┌ Column  Primary ────────────────┐ │  mine
//	│                   │ Ainda não                 10:43 │ │
//	│                   └─────────────────────────────────┘ │
//	└───────────────────────────────────────────────────────┘
//
// # The two sides' colours
//
// Mine is the theme's Primary with the ink chosen by contrast against it
// (Variant.Ink), the fill every chat app gives the reader's own words. Theirs
// has no palette role to take: there is no muted container tone (Banner's doc
// has the long version of that gap), and inventing one is a theme decision a
// chat bubble should not force. So theirs is Surface with a Border hairline —
// Banner's answer to the same gap — which separates it from the page in every
// bundled theme without a colour anybody has to choose.
//
// # What it is not
//
//   - **Not a thread.** comps.MessageThread is: a List of bubbles that opens
//     at the newest and loads older ones as the reader scrolls up
//     (core.StartAtEnd, core.OnStartReached). A caller can still lay bubbles
//     out in its own Column, as examples/chat does, and the spacing between
//     them is then the caller's too.
//
// # The tail
//
// The bottom corner on the sender's side is nearly square (4 against 16):
// bottom-right on the reader's own messages, bottom-left on everyone else's.
// It is the shape every chat app uses to say whose a bubble is without a
// drawn point, and it costs nothing but a radius per corner
// (core.CornerRadii). Every bubble has it, not only the last of a run: a run
// is the caller's layout, and a bubble cannot see its neighbours. The
// corners are physical, so a right-to-left transcript that lines the
// reader's messages up on the left mirrors Mine itself.
//
// # Accessibility
//
// The bubble is one stop named "Ana, Já viste a nova versão?, 10:42" — who,
// what, when — with its parts hidden, so a reader moving through a transcript
// hears each message whole rather than as three fragments. The reader's own
// messages are named with MineLabel ("You") in the sender's place, since no
// sender is drawn on them. Put the bubbles under a core.RoleLog container
// (examples/chat does) so new ones are announced.
//
// # Theme roles read
//
//	Mine       Colors.Primary fill, ink by contrast (Variant.Ink)
//	Theirs     Colors.Surface fill, ColorPalette.BorderColor hairline, TextPrimary ink
//	Sender     Typography.Caption, bold
//	Text       Typography.Body
//	Time       Typography.Caption; TextSecondary on theirs, the ink on mine
type MessageBubble struct {
	// Text is the message.
	Text string

	// Sender is drawn above the text of someone else's message. Leave it
	// empty in a one-to-one chat, where it says nothing. It is never drawn
	// on Mine.
	Sender string

	// Continued marks the second and later of a run of messages from the same
	// sender: the sender line is not drawn, because the bubble above already
	// says who, and it is still spoken, because a screen reader moving through
	// the transcript meets each message on its own. Emptying Sender instead
	// hides the line and the name both. comps.MessageThread sets it.
	Continued bool

	// Mine puts the bubble on the trailing side in the Primary fill.
	Mine bool

	// Time is drawn small under the text, at the trailing edge ("10:42").
	// The widget does not format times; pass what the screen should show.
	Time string

	// MineLabel stands in for the sender in the spoken name of the reader's
	// own message; empty gives "You".
	MineLabel string

	// Style is applied to the outer row — the placement, where a margin
	// between messages belongs — after its defaults.
	Style []core.StyleProp
}

// Render builds Row(justify, Column(sender?, text, time?)). It takes no hook
// slot.
// bubbleRadius is a bubble's rounding and bubbleTail its one sharp corner.
const (
	bubbleRadius = 16
	bubbleTail   = 4
)

func (m MessageBubble) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	fill, ink, timeInk := t.Colors.Surface, t.Colors.TextPrimary, t.Colors.TextSecondary
	side := core.JustifyStart
	if m.Mine {
		fill = t.Colors.Primary
		ink = VariantDefault.Ink(t, fill)
		// The secondary ink is a grey meant for a light page; on the
		// Primary fill it would be illegible, so the time takes the text's
		// own ink there.
		timeInk = ink
		side = core.JustifyEnd
	}

	// Who, what, when — the order a reader wants a message in.
	who := m.Sender
	if m.Mine {
		who = orDefault(m.MineLabel, "You")
	}
	name := m.Text
	if who != "" {
		name = who + ", " + name
	}
	if m.Time != "" {
		name += ", " + m.Time
	}

	// The tail: the bottom corner on the sender's side, nearly square, so the
	// bubble points at whose it is (core.CornerRadii; see "The tail").
	tail := core.CornerRadii(bubbleRadius, bubbleRadius, bubbleRadius, bubbleTail)
	if m.Mine {
		tail = core.CornerRadii(bubbleRadius, bubbleRadius, bubbleTail, bubbleRadius)
	}
	bubble := []core.PropsAndChildren{
		core.Gap(2),
		core.PaddingVertical(8),
		core.PaddingHorizontal(12),
		core.BorderRadius(bubbleRadius),
		tail,
		core.BackgroundColor(fill),
		// A bubble that could span the row would read as a banner, and the
		// empty margin on the far side is what says whose message it is.
		// Percentages resolve on all four targets (GrMobMaxWidth on iOS,
		// widthModifier on Compose, CSS on the web).
		core.MaxWidth("80%"),
		core.AccessibilityRole(core.RoleGroup),
		core.AccessibilityLabel(name),
	}
	if !m.Mine {
		bubble = append(bubble, core.BorderWidth(1), core.BorderColor(t.Colors.BorderColor()))
	}
	if m.Sender != "" && !m.Mine && !m.Continued {
		bubble = append(bubble, core.Text(m.Sender,
			core.UseStyle(t.Typography.Caption),
			core.FontWeight(core.Bold),
			core.TextColor(ink),
			core.AccessibilityHidden()))
	}
	bubble = append(bubble, core.Text(m.Text,
		core.UseStyle(t.Typography.Body),
		core.TextColor(ink),
		core.AccessibilityHidden()))
	if m.Time != "" {
		bubble = append(bubble, core.Text(m.Time,
			core.UseStyle(t.Typography.Caption),
			core.TextColor(timeInk),
			core.Align(core.AlignEnd),
			core.AccessibilityHidden()))
	}

	row := make([]core.PropsAndChildren, 0, len(m.Style)+3)
	row = append(row,
		// The theme's Row inset is for list rows; a transcript's own
		// container pads the conversation once.
		core.Padding(0),
		core.Justify(side),
	)
	row = append(row, asProps(m.Style)...)
	row = append(row, core.Column(bubble...))
	return core.Row(row...).Render(ctx)
}
