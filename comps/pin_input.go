package comps

import (
	"fmt"

	"github.com/rohanthewiz/grmob/core"
)

// ConcernPINInputInert is raised, in debug builds only, when a PINInput has no
// OnChange. The field is then read-only in practice — every keystroke reaches
// the handler, is discarded, and the next pass paints Value back over it — and
// on screen an inert PINInput is indistinguishable from one nobody has typed
// into yet. Disclosure's inert case is reported for the same reason: a widget
// that cannot do the one thing it exists for should say so somewhere other
// than in a bug report.
const ConcernPINInputInert = "pin-input-inert"

// ConcernPINValueTooLong is raised, in debug builds only, when Value holds
// more characters than there are cells. The extra ones are not drawn and can
// never be typed away, so a field that looks full is carrying a value its
// caller cannot see — and OnComplete's "the code is as long as the field"
// test would be met by characters nobody entered.
const ConcernPINValueTooLong = "pin-value-too-long"

// defaultPINLength is what Length means when it is left unset. Six is the
// one-time code every SMS and authenticator app sends; four is a device PIN
// and is one field away.
const defaultPINLength = 6

// PINInput is the boxed one-character-per-cell field a one-time code is typed
// into: a row of boxes showing the code, over one field that holds it.
//
//	comps.PINInput{
//	    Length:     6,
//	    Value:      code.Get(),
//	    OnChange:   code.Set,
//	    OnComplete: func(c string) { verify(c) },
//	}
//
//	┌───┐ ┌───┐ ┌───┐ ┌───┐ ┏━━━┓ ┌───┐
//	│ 4 │ │ 1 │ │ 7 │ │ 2 │ ┃   ┃ │   │
//	└───┘ └───┘ └───┘ └───┘ ┗━━━┛ └───┘
//	                          ▲ the next box to fill, marked while focused
//
// # One field owns the code
//
// The boxes are drawing, and the typing goes into a single text field under
// them that holds the whole code. Tapping anywhere on the row puts the caret
// in that field (core.Focus); every key, paste, backspace and autofill is an
// ordinary edit of one string, and the boxes redraw from it.
//
// It used to be six fields, one per box, with the widget moving the caret
// from each to the next as a character landed. That lost digits, and not in
// a way any bookkeeping could fix. On the Android emulator, keys typed about
// 130ms apart:
//
//	"3" in box 0, "1" in box 0 before the caret moved     → spread: "31"
//	the "4" typed while focus was on its way to box 2     → never reached a field
//	box 0 blurred before Go's rewrite of it arrived       → nothing left to replay "314"
//
// Keys typed during a focus move have nowhere to go, and a per-box rewrite
// can only be replayed by the box that was typed into, which has already been
// left. One field has no focus moves and no per-box rewrites: the code is one
// value under the text-edit protocol (core/text_edit.go) like any other
// field's, so a burst of keys is replayed onto Go's text as it is in a search
// box. It is also what the platforms' own OTP fields are: one hidden input,
// which is what SMS autofill and a password manager fill.
//
// # The rules that fall out
//
//   - Value is the field's text, capped at Length: a paste of a longer string
//     keeps its first Length characters.
//   - Backspace deletes the last character, wherever the reader tapped, and an
//     empty field's backspace does nothing: the field's own behaviour.
//   - OnComplete fires on every change that leaves the code full, including a
//     correction to a code that was already full, and never for a Value that
//     merely arrived complete (a restored screen does not resubmit itself). A
//     change that produces the value already held is an echo: no OnChange, no
//     OnComplete.
//
// # The field
//
// A core.Input (core.InputPassword when Secure), one point square, with no
// frame, fill or ink, in a ZStack layer under the boxes: present, focusable,
// and filled by the keyboard, but nothing a reader sees or taps directly. It
// asks for the number pad with core.Keyboard(core.KeyboardDigits), which on
// iOS also marks it as a one-time code field, so the system offers a code from
// a text message above the keyboard. The pad is a hint: a hardware keyboard or
// a paste can still put letters in, and they are drawn as typed, because a
// code is not always digits.
//
// # It holds hooks
//
// A FocusRef for the field and whether the field has focus (so the next box
// can be marked). So it has Accordion's rule: render it in a stable position
// every pass rather than inside a core.If.
//
// # Accessibility
//
// The field is the control, named "<Label>, N of M entered", so a reader
// moving onto it hears how far the code has got; the boxes are hidden, being
// a picture of what the field holds. The row is a core.RoleGroup named by
// Label. Label is the accessible name only; wrap this in a FormField when a
// visible caption is wanted.
//
// # Theme roles read
//
//	Boxes     Components.Input: the frame every other field in the form has
//	Marker    Colors.Primary: the next box's border while the field has focus
//	Gap       Spacing.SM between boxes
type PINInput struct {
	// Length is the number of boxes. Zero means six, the one-time code length.
	Length int

	// Value is the code so far, in full. The field is controlled: the boxes
	// draw exactly this, one character per box from the left, and OnChange is
	// the only way it changes.
	Value string

	// OnChange receives the whole code after every edit.
	// Without it the field is read-only and reports ConcernPINInputInert.
	OnChange func(string)

	// OnComplete receives the code on every edit that leaves it as long as
	// the field, including an edit to a code that was already complete. Nil
	// is a field the caller reads from Value instead.
	OnComplete func(string)

	// Secure masks the characters, as a device PIN rather than an emailed
	// code: the boxes draw a dot, and the field is core.InputPassword.
	Secure bool

	// Label is the accessible name of the group and the stem of the field's
	// name. Empty means "Code". It draws nothing.
	Label string

	// Style is applied to the row, after the gap and the accessibility pair,
	// so a caller can override any of them, or cap the width, which is the
	// common one: MaxWidth stops four boxes from spreading across a tablet.
	Style []core.StyleProp
}

// Render draws the boxes and the field under them.
func (p PINInput) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()
	n := p.length()

	field := core.UseFocusRef(ctx)
	focusedState := core.NewState(ctx, false)
	focused := focusedState.Get()

	code := []rune(p.Value)
	if core.IsDebugMode() {
		if p.OnChange == nil {
			core.ReportConcern(ConcernPINInputInert,
				"PINInput has no OnChange, so every keystroke is discarded and the field can never be filled")
		}
		if len(code) > n {
			core.ReportConcern(ConcernPINValueTooLong, fmt.Sprintf(
				"PINInput has %d boxes and a Value of %d characters: the last %d are never drawn and cannot be edited",
				n, len(code), len(code)-n))
		}
	}
	if len(code) > n {
		// Drawn as the field can hold it. The concern above is the report.
		code = code[:n]
	}

	label := p.Label
	if label == "" {
		label = "Code"
	}

	// The next box to fill: the one after the code, or the last when full.
	next := len(code)
	if next >= n {
		next = n - 1
	}

	base := t.Components.Input
	boxes := make([]core.PropsAndChildren, 0, n+4)
	boxes = append(boxes,
		core.Padding(0),
		core.Gap(float64(t.Spacing.SM)),
		// The ZStack's width, stated: the natives stretch a layer to its
		// stack and the web's grid places it at its content width, which drew
		// six narrow boxes in the browser (the outer ZStack says the same).
		core.Width("100%"),
		// A tap anywhere on the row is a tap on the field.
		core.OnClick(func() { core.Focus(field) }),
	)
	for i := 0; i < n; i++ {
		ch := ""
		if i < len(code) {
			ch = string(code[i])
			if p.Secure {
				ch = "•"
			}
		}
		box := []core.PropsAndChildren{
			core.UseStyle(base),
			// Equal shares on all four targets: the natives divide the axis
			// by weight and ignore the basis, CSS divides only the leftover
			// and needs the zero to start from. Without it a filled box
			// would be a hair wider than an empty one and the boxes would
			// shuffle as the code is typed.
			core.FlexGrow(1),
			core.FlexBasis("0"),
			core.AlignItemsProp(core.AlignItemsCenter),
			core.AccessibilityHidden(),
		}
		if focused && i == next {
			box = append(box, core.BorderColor(t.Colors.Primary), core.BorderWidth(2))
		}
		// A space rather than "" in an empty box, so every box has the one
		// line of text height a filled one has on every host.
		shown := ch
		if shown == "" {
			shown = " "
		}
		box = append(box, core.Text(shown,
			core.FontSize(base.FontSize),
			core.TextColor(base.TextColor),
			core.Align(core.AlignCenter)))
		boxes = append(boxes, core.Box(box...))
	}

	input := []core.PropsAndChildren{
		core.FocusTarget(field),
		// The number pad (core.Keyboard); on iOS also the SMS code offered
		// above it.
		core.Keyboard(core.KeyboardDigits),
		core.OnFocus(func() { focusedState.Set(true) }),
		core.OnBlur(func() { focusedState.Set(false) }),
		// One point, no frame, no fill, no ink: in the tree and focusable,
		// not seen.
		core.Width("1px"),
		core.Height("1px"),
		// Bottom-start, not top-start. A platform scrolls a focused field
		// into view above its keyboard, and it scrolls the least that shows
		// the *field*: at the top corner that was one point of the row, and
		// on the Android emulator the boxes sat behind the keyboard. At the
		// bottom corner, the row's whole height comes up with it.
		//
		// Inset into the first box, and under the row (it is the ZStack's
		// first layer): the box's own fill covers it, so the focus ring a
		// browser draws round a focused input, and the caret a native draws
		// in it, are behind the box rather than a dot beside it (seen in the
		// browser before this). The row, on top, takes every tap and focuses
		// the field itself.
		core.MarginLeft(8),
		core.MarginBottom(8),
		core.Padding(0),
		core.BorderWidth(0),
		core.BackgroundColor(ColorTransparent),
		core.TextColor(ColorTransparent),
		core.StackAlign(core.StackAlignBottomStart),
		core.AccessibilityLabel(fmt.Sprintf("%s, %d of %d entered", label, len(code), n)),
	}
	onChange := p.changed(string(code))
	var control core.View
	if p.Secure {
		control = core.InputPassword(string(code), "", onChange, input...)
	} else {
		control = core.Input(string(code), "", onChange, input...)
	}

	outer := make([]core.PropsAndChildren, 0, len(p.Style)+5)
	outer = append(outer,
		core.Padding(0),
		// The width it is given, on every target (a caller's MaxWidth in
		// Style still caps it): the natives stretch a ZStack in a column, and
		// the web's grid is as wide as its content unless told.
		core.Width("100%"),
		core.AccessibilityRole(core.RoleGroup),
		core.AccessibilityLabel(label),
	)
	outer = append(outer, asProps(p.Style)...)
	outer = append(outer, control, core.Row(boxes...))
	return core.ZStack(outer...).Render(ctx)
}

// length is Length with its default applied.
func (p PINInput) length() int {
	if p.Length <= 0 {
		return defaultPINLength
	}
	return p.Length
}

// changed builds the field's handler. held is the pass's drawn code, captured
// rather than re-read: a handler dispatched from the registry runs against the
// tree that registered it, which is the state the reader was looking at.
func (p PINInput) changed(held string) func(string) {
	n := p.length()
	return func(typed string) {
		next := []rune(typed)
		if len(next) > n {
			// A paste longer than the field keeps what fits.
			next = next[:n]
		}
		if string(next) == held {
			// An echo, or a paste that fits to what was already there.
			return
		}
		if p.OnChange != nil {
			p.OnChange(string(next))
		}
		if p.OnComplete != nil && len(next) == n {
			p.OnComplete(string(next))
		}
	}
}
