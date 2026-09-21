package comps

import (
	"fmt"
	"time"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/hooks"
)

// TypingIndicator is the three dots a chat draws while the other side is
// writing: a small theirs-coloured bubble on the leading side, one dot after
// another darkening in turn, with an optional "Ana is typing" caption.
//
//	comps.TypingIndicator{Visible: anaTyping.Get(), Who: "Ana"}
//
//	┌ Row  justify=start  role=status  label="Ana is typing" ─┐
//	│ ┌ Row  Surface + hairline ┐                             │
//	│ │   ●   ○   ○             │  Ana is typing   (Caption)  │
//	│ └─────────────────────────┘                             │
//	└─────────────────────────────────────────────────────────┘
//	      ▲ the dark dot moves one place every typingBeat
//
// # Visible, and why the widget is never left out
//
// A typing indicator is conditional by nature — it is there only while
// somebody types — and the animation needs two hook slots (the phase and its
// interval). Those two facts collide: a widget that owns hooks must render on
// every pass, in the same place, or every hook after it shifts onto a
// neighbour's slot. So the switch is a field, not a core.If around the
// widget:
//
//	comps.TypingIndicator{Visible: typing}        // right
//	core.If(typing, comps.TypingIndicator{...})   // wrong: a conditional hook
//
// Hidden is Display none — Spinner.Hidden's answer — so the tree keeps its
// shape both ways and showing the indicator is a style patch, not an
// inserted subtree. The interval is hooks.UseIntervalWhile keyed on Visible,
// so a hidden indicator costs one goroutine wake per beat and no render
// pass; comps.Countdown's ticking is the precedent.
//
// The zero value is hidden. That is the safe way round for a widget whose
// visible state costs render passes: forgetting the field draws nothing,
// rather than animating for the life of the screen.
//
// # How the dots move: opacity
//
// Every dot is the same fill, Colors.TextPrimary, and the phase picks which
// one is drawn at full strength; the other two rest at typingRestAlpha. Each
// carries a core.Transition, so a dot eases between the two alphas, and Go
// sends one patch per beat (two dots change); every frame between the beats
// is the platform's.
//
// The first build eased between two background colours, because core.Style
// had no opacity then. That needed a second tone that reads on the bubble in
// every theme, and finding one took three tries: TextSecondary already
// carries an alpha byte in DefaultTheme, so dimming it further did nothing;
// Border is 1.26:1 on the page and vanished; ControlBorderColor() worked.
// One colour at two alphas asks the theme for nothing but its text colour,
// which is legible on Surface by definition, so a custom theme cannot break
// the resting dots. The dark dot states Opacity(1) rather than leaving the
// field unset, so both states are written the same way and neither depends
// on what an absent key means to a renderer.
//
// This is not Spinner's old mistake restated. Spinner stepped twelve passes a
// second to fake a rotation a renderer could do itself, and core.Spin removed
// it. Here there is no native primitive to hand the loop to (a looping
// transition is what core.Spin's doc declines to generalise), the rate is
// two and a half passes a second, and it runs only while somebody is typing.
//
// # Reduce Motion
//
// Each host drops the Transition on its own side (core.Transition, "Reduced
// motion") and nothing tells Go, so under the setting the interval still
// steps the phase and the dots change alpha without the ease. The dots are
// six points across and the change is one grey to another, so the un-eased
// form is a quiet blink rather than a flash. It has not been looked at on a
// device with the setting on; that check is on the Next list.
//
// # Accessibility
//
// The whole widget is one RoleStatus stop named by Label — a polite live
// region, announced when it appears. The dots and the caption are hidden from
// assistive technology: the dots are decoration, and the caption repeats the
// name. The phase changes only the dots' opacity, so the name is stable and a
// beat re-announces nothing. Hidden is display:none and is not announced.
//
// # Theme roles read
//
//	Bubble        Colors.Surface fill, Colors.BorderColor() hairline — theirs,
//	              as MessageBubble draws it, so the dots read as a bubble
//	              about to arrive
//	Dots          Colors.TextPrimary, at typingRestAlpha except the dark one
//	Caption       Typography.Caption, Colors.TextSecondary
type TypingIndicator struct {
	// Visible shows the indicator and runs its animation. False (the zero
	// value) is Display none with the interval paused. See the type doc for
	// why this is a field and not a core.If.
	Visible bool

	// Who is the person typing ("Ana"). It feeds the default Label and
	// nothing else.
	Who string

	// Label is the spoken name, and the caption's text when Caption is set.
	// Empty gives Who + " is typing", or "Typing" when Who is empty too. Set
	// it to localise, or for a group chat ("Ana and Rui are typing").
	Label string

	// Caption draws Label beside the dots. Off by default, because the dots
	// usually sit where the next bubble will appear and say enough there.
	Caption bool

	// Style is applied to the outer row — the placement, as with
	// MessageBubble — after its defaults.
	Style []core.StyleProp
}

const (
	// typingBeat is how long each dot stays dark. 400ms puts a full cycle at
	// 1.2s, the pace the familiar chat apps settle near: slower reads as
	// stalled, faster reads as a loading spinner.
	typingBeat = 400 * time.Millisecond

	// typingRestAlpha is how strongly a resting dot is drawn. Low enough that
	// the dark dot is unmistakable beside it, high enough that three resting
	// dots still read as dots on Surface in a light theme and a dark one.
	// 0.4 is fitted to the tone the colour build was looked at with:
	// DefaultTheme's black at 0.4 over Surface (#F2F2F7) composites to about
	// #919194, against ControlBorderColor()'s #89898E.
	typingRestAlpha = 0.4

	// typingEaseMs is the opacity ease. Shorter than the beat, so each dot
	// reaches its dark tone and holds it briefly before the next takes over;
	// equal to the beat would leave every dot permanently mid-fade.
	typingEaseMs = 300

	// typingDots is the dot count, and typingDotSize their diameter in
	// points.
	typingDots    = 3
	typingDotSize = 6.0
)

// label resolves the spoken name: the caller's, else one built from Who.
func (ti TypingIndicator) label() string {
	switch {
	case ti.Label != "":
		return ti.Label
	case ti.Who != "":
		return ti.Who + " is typing"
	default:
		return "Typing"
	}
}

// Render takes two hook slots, unconditionally, and then draws
// Row(bubble(dot × 3), caption?).
func (ti TypingIndicator) Render(ctx *core.Context) *core.Node {
	// Both hooks come before any branch. phase is which dot is dark, 0 to 2.
	phase := core.NewState(ctx, 0)
	// The callback is re-bound every pass, so phase.Get() reads the current
	// value when the tick fires rather than the first pass's.
	hooks.UseIntervalWhile(ctx, ti.Visible, func() {
		phase.Set((phase.Get() + 1) % typingDots)
	}, typingBeat)

	t := ctx.Theme()
	label := ti.label()
	px := fmt.Sprintf("%gpx", typingDotSize)

	bubble := []core.PropsAndChildren{
		core.Gap(4),
		core.AlignItemsProp(core.AlignItemsCenter),
		// MessageBubble's padding and radius, so the indicator lines up with
		// the bubbles above it and is replaced by one without a jump.
		core.PaddingVertical(12),
		core.PaddingHorizontal(12),
		core.BorderRadius(bubbleRadius),
		core.CornerRadii(bubbleRadius, bubbleRadius, bubbleRadius, bubbleTail),
		core.BackgroundColor(t.Colors.Surface),
		core.BorderWidth(1),
		core.BorderColor(t.Colors.BorderColor()),
		core.AccessibilityHidden(),
	}
	for i := range typingDots {
		alpha := typingRestAlpha
		if i == phase.Get() {
			alpha = 1
		}
		bubble = append(bubble, core.Box(
			core.Width(px),
			core.Height(px),
			core.BorderRadius(typingDotSize/2),
			core.BackgroundColor(t.Colors.TextPrimary),
			core.Opacity(alpha),
			core.Transition(typingEaseMs, core.EaseInOut),
		))
	}

	row := make([]core.PropsAndChildren, 0, len(ti.Style)+9)
	row = append(row,
		// The theme's Row inset is for list rows; see MessageBubble.
		core.Padding(0),
		core.Gap(float64(t.Spacing.SM)),
		core.Justify(core.JustifyStart),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.AccessibilityRole(core.RoleStatus),
		core.AccessibilityLabel(label),
	)
	if !ti.Visible {
		row = append(row, core.Display(core.DisplayNone))
	}
	row = append(row, asProps(ti.Style)...)
	row = append(row, core.Row(bubble...))
	if ti.Caption {
		row = append(row, core.Text(label,
			core.UseStyle(t.Typography.Caption),
			core.TextColor(t.Colors.TextSecondary),
			core.AccessibilityHidden()))
	}
	return core.Row(row...).Render(ctx)
}
