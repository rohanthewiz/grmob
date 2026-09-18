package core

// KeyboardKind is which software keyboard a text field asks for.
//
// # Why a prop, and not a node type
//
// The keyboard used to follow the node type alone: NumericInput took the
// number pad and every other field the text keyboard. NumericInput carries an
// int, though, and a one-time code, a phone number or a card number is text
// that happens to be typed on a number pad: "0123" is not 123, and an empty
// field is not 0. comps.PINInput documented the gap in so many words ("a
// digits-only keyboard needs a keyboard-type prop on core.Input"). This is
// that prop.
//
// It is a hint, as every platform treats it: a hardware keyboard and a paste
// still put any text in the field, so a caller that needs digits only checks
// the value in its OnChange, as it would anyway.
type KeyboardKind string

const (
	// KeyboardText is the default, stated.
	KeyboardText KeyboardKind = ""
	// KeyboardDigits is the number pad: 0-9 and nothing else where the
	// platform has such a pad (iOS numberPad, Android TYPE_CLASS_NUMBER, web
	// inputmode="numeric").
	KeyboardDigits KeyboardKind = "digits"
	// KeyboardDecimal adds the decimal separator.
	KeyboardDecimal KeyboardKind = "decimal"
	// KeyboardPhone is the telephone pad.
	KeyboardPhone KeyboardKind = "phone"
	// KeyboardEmail puts @ and . on the first layer.
	KeyboardEmail KeyboardKind = "email"
	// KeyboardURL puts / and . on the first layer.
	KeyboardURL KeyboardKind = "url"
)

// Keyboard asks a text field (Input, InputPassword, TextArea) for a keyboard.
// It travels as the "keyboard" prop; NumericInput ignores it, having its own.
//
//	core.Input(code, "", setCode, core.Keyboard(core.KeyboardDigits))
//
// An empty kind writes nothing, so a tree that never asks is unchanged.
func Keyboard(kind KeyboardKind) BehaviorProp {
	return behaviorFunc(func(_ *Context, n *Node) {
		if kind == KeyboardText {
			return
		}
		if n.Props == nil {
			n.Props = map[string]any{}
		}
		n.Props["keyboard"] = string(kind)
	})
}
