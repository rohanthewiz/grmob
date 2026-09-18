package comps

import "github.com/rohanthewiz/grmob/core"

// CopyButton is a button that puts a fixed string on the system clipboard
// and confirms it: a code snippet, an invite link, a QR code's payload, an
// order number.
//
//	comps.CopyButton{Text: inviteURL, Label: "Copy link"}
//
//	tap ──► core.WriteClipboard(Text)
//	    ──► core.Haptic(HapticLight)
//	    ──► core.ShowToast(CopiedMessage)          "Copied"
//
// # The toast confirms, not the caption
//
// The familiar web idiom flips the button's own caption to "Copied ✓" for a
// second or two. That needs a timer to flip it back, and a timer here is a
// hook (hooks.UseTimeoutWhile), which would make CopyButton unsafe to render
// inside a conditional or a loop. Its first consumer is the tutorial's
// codeBlock, which is built in exactly those places and says in its own doc
// that a widget with hook obligations could not be. Banner's doc had already
// assigned the job: "Use the toast for 'Copied'". So the platform's transient
// overlay confirms, and the widget stays stateless, like Stepper and
// TimePicker.
//
// The haptic is the light tick, the one a successful small action gets; a
// device without a motor, and every web target, ignores it.
//
// # Nothing to copy means inert
//
// An empty Text disables the button rather than reporting a concern. It is a
// legitimate state — an invite link still being fetched, an order number not
// yet assigned — and a disabled Copy says truthfully that there is nothing to
// copy yet. The one thing an enabled Copy must not do with an empty string is
// write it: core.WriteClipboard("") clears the clipboard, which is a real
// request when made on purpose and never what a button labelled "Copy"
// means.
//
// # Accessibility
//
// Several copy buttons on one screen — one per code block — would all be
// announced "Copy", so AccessibilityLabel is where a caller says what is
// copied ("Copy code", "Copy invite link"). The copied text itself is not
// read out: it is usually on screen beside the button, and a URL or a
// snippet spoken in full is noise. The toast is announced by each platform's
// own toast machinery.
//
// # Theme roles read
//
// None of its own: the button is a comps.Button, and reads what Button reads
// for the Variant and Emphasis given.
type CopyButton struct {
	// Text is what lands on the clipboard. Empty disables the button; see
	// "Nothing to copy means inert".
	Text string

	// Label is the visible caption; empty gives "Copy".
	Label string

	// CopiedMessage is the toast shown after a copy; empty gives "Copied".
	CopiedMessage string

	// Variant and Emphasis are passed to the Button unchanged. The zero value
	// is a filled Primary button; a copy action sitting beside the thing it
	// copies usually wants EmphasisOutlined or EmphasisGhost.
	Variant  Variant
	Emphasis Emphasis

	// Disabled makes the button inert even with Text set.
	Disabled bool

	// AccessibilityLabel names the button for screen readers; empty uses the
	// visible Label. AccessibilityHint describes the effect; empty gives
	// "Copies to the clipboard".
	AccessibilityLabel string
	AccessibilityHint  string

	// Style is applied to the Button after its variant treatment.
	Style []core.StyleProp

	// FocusRef names the button for core.Focus.
	FocusRef *core.FocusRef
}

// Render draws the button. It takes no hook slot, so it may be rendered
// conditionally.
func (c CopyButton) Render(ctx *core.Context) *core.Node {
	label := orDefault(c.Label, "Copy")
	message := orDefault(c.CopiedMessage, "Copied")

	// Captured by value: the closure must copy the Text this pass drew, not
	// whatever a later pass's CopyButton value holds.
	text := c.Text

	return Button{
		Label: label,
		OnTap: func() {
			// Guarded as well as disabled: a tap can arrive in the window
			// between the press and the patch that disables the button (the
			// race Button's own Disabled doc describes), and an empty write
			// would clear the clipboard.
			if text == "" {
				return
			}
			core.WriteClipboard(text)
			core.Haptic(core.HapticLight)
			core.ShowToast(message)
		},
		Variant:            c.Variant,
		Emphasis:           c.Emphasis,
		Disabled:           c.Disabled || text == "",
		Style:              c.Style,
		AccessibilityLabel: orDefault(c.AccessibilityLabel, label),
		AccessibilityHint:  orDefault(c.AccessibilityHint, "Copies to the clipboard"),
		FocusRef:           c.FocusRef,
	}.Render(ctx)
}
