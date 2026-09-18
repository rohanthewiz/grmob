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
//   - **Not a thread.** A conversation opens at its newest message, which is
//     a scroll offset, and no host reports or accepts one (Carousel's wall).
//     A caller lays bubbles out in its own Column or core.List, as
//     examples/chat does, and the spacing between them is the caller's too.
//   - **No tail.** The little point on a bubble's corner is one sharp corner
//     on a rounded box, and core has one radius, not four — DateRangePicker's
//     notch again.
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
	// empty to hide it — on the second of two consecutive messages from the
	// same person, or in a one-to-one chat. It is never drawn on Mine.
	Sender string

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

	bubble := []core.PropsAndChildren{
		core.Gap(2),
		core.PaddingVertical(8),
		core.PaddingHorizontal(12),
		core.BorderRadius(16),
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
	if m.Sender != "" && !m.Mine {
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
