package comps

import (
	"strings"

	"github.com/rohanthewiz/grmob/core"
)

// ConcernMaskedInputInert is raised, in debug builds only, when a MaskedInput
// has no OnChange and is not Disabled. Every keystroke is then unmasked,
// discarded, and painted over by the next pass: PINInput's inert case.
const ConcernMaskedInputInert = "masked-input-inert"

// ConcernMaskedInputNoSlots is raised, in debug builds only, when Mask holds
// no slot character (#, A or *). Such a field can hold nothing: every key is
// refused, and it looks exactly like a field nobody has typed into yet. The
// usual cause is a mask written in another library's alphabet ("999-999",
// "000 000").
const ConcernMaskedInputNoSlots = "masked-input-no-slots"

// MaskedInput is a text field that formats as the reader types: a phone
// number that gains its brackets, a card number that groups in fours, an
// expiry date that gets its slash.
//
//	comps.MaskedInput{
//	    Mask:     "(###) ###-####",
//	    Value:    phone.Get(),                          // "5551234567"
//	    OnChange: func(raw, _ string) { phone.Set(raw) },
//	    Keyboard: core.KeyboardDigits,
//	    Label:    "Phone",
//	}
//
//	the reader types      the field shows       Value (raw)
//	5                     (5                    5
//	555                   (555                  555
//	5556                  (555) 6               5556
//	5551234567            (555) 123-4567        5551234567
//
// # The mask
//
//	#   a digit          A   a letter          *   either
//
// Every other character is a literal, drawn as written. A literal is written
// only once a character follows it, so the text never ends in one and a
// backspace always removes something the reader typed (comps/mask.go says what
// the eager form costs). A key the next slot refuses is dropped, and so is
// anything past the last slot.
//
// # Value is the raw value
//
// The caller holds the characters the reader meant, "5551234567", and not the
// drawn text. That is the form an application stores, validates and sends, so
// holding the other one would mean unmasking it at every use. OnChange hands
// over both, for the caller who does want the drawn text (to show it again
// elsewhere, or to submit it as typed).
//
// One consequence, stated rather than hidden: a mask with a literal that one
// of its own slots could hold, as the "1" in "+1 (###) ###-####", reads a
// typed character equal to that literal as the literal. A raw value under
// that mask cannot begin with 1. For a North American number that is the
// right answer (no area code begins with 1, and a reader who types the 1
// means the country code), and it is the price of reading the text back
// without mistaking the literal for data on every keystroke.
//
// # Every formatted keystroke is a rewrite
//
// The reader types "5" after "(555" and the host shows "(5555"; Go answers
// "(555) 5", which is not what the host sent. Under the text-edit protocol
// (core/text_edit.go) that is a rewrite, not an echo, and this widget makes
// one for most keys, where TagInput makes one per tag. Two things were
// measured before it was built, on the Android emulator with `adb shell input
// text` and keys a second apart:
//
//   - Typing at machine speed loses nothing. The hosts replay in-flight keys
//     onto each rewrite (internal/rebasefixture), and ten digits and sixteen
//     digits arrived whole, in order, on every run.
//   - The caret has to travel with its text. A mask inserts literals before
//     the caret, and a host that kept the caret's raw offset put every later
//     key in front of the first: 1 2 3 4 5 6 read back "(234) 651". All three
//     live hosts now carry the caret across Go's change by one rule
//     (rebasefixture.Carry); the Android field was the one that did not.
//
// The limit that remains: a key typed mid-text at the end of a group, as a
// digit after "(555" in "(555) 123", reflows everything after it, and the one
// differing span the hosts can see then includes the caret. It lands after
// the reflowed text instead of after the key. Nothing is lost and the next
// key still goes in; the caret is in the wrong place for it. Typing anywhere
// else mid-text keeps its place. Fixing it needs a caret position in the
// protocol, which no host sends.
//
// # It takes no hooks
//
// The draft is the value, and the value is the caller's. So a MaskedInput may
// be rendered conditionally, unlike TagInput and PINInput.
//
// # Accessibility
//
// A plain text field, named by Label. The mask is not announced: a screen
// reader reads the field's text, literals included, which is the formatted
// value a sighted reader sees. Hint carries anything more ("Ten digits").
//
// # Theme roles read
//
// Those of core.Input (Components.Input), and nothing else.
type MaskedInput struct {
	// Mask is the format: # a digit, A a letter, * either, anything else a
	// literal. A mask with no slot reports ConcernMaskedInputNoSlots.
	Mask string

	// Value is the raw value: slot characters only, with no literals. See
	// "Value is the raw value". Characters the mask refuses, or past its last
	// slot, are not drawn.
	Value string

	// OnChange receives the raw value and the text as drawn, on every edit
	// that changes the raw value. A refused key changes nothing and reports
	// nothing. Nil reports ConcernMaskedInputInert unless Disabled.
	OnChange func(raw, formatted string)

	// OnComplete receives the raw value on every edit that fills the mask's
	// last slot, as PINInput's does: including a correction to a value that
	// was already complete, and never for a Value that merely arrived full.
	OnComplete func(raw string)

	// Placeholder is drawn in the empty field. Empty means the mask with each
	// slot drawn as "_", "(___) ___-____", which shows the reader the shape
	// of what is wanted without putting example data in front of them.
	Placeholder string

	// Keyboard asks for a soft keyboard. The zero value is the default
	// keyboard; a mask of digits wants core.KeyboardDigits.
	Keyboard core.KeyboardKind

	// Label is the field's accessible name. It draws nothing: wrap the widget
	// in a FormField for a visible caption.
	Label string

	// Hint is the field's accessibility hint.
	Hint string

	// Disabled greys the field and drops its reports.
	Disabled bool

	// Style is applied to the input after its defaults.
	Style []core.StyleProp
}

// Render draws the field.
func (in MaskedInput) Render(ctx *core.Context) *core.Node {
	capacity := maskCapacity(in.Mask)
	if core.IsDebugMode() {
		if capacity == 0 {
			core.ReportConcern(ConcernMaskedInputNoSlots,
				"MaskedInput's Mask \""+in.Mask+"\" has no slot (#, A or *), so the field can hold nothing")
		}
		if in.OnChange == nil && !in.Disabled {
			core.ReportConcern(ConcernMaskedInputInert,
				"MaskedInput has no OnChange and is not Disabled, so every keystroke is discarded")
		}
	}

	// Drawn from the mask and not from Value as given, and then read back:
	// held is the raw value the field can actually show, so a Value with a
	// stray character in it does not make every later edit look like a change.
	shown := applyMask(in.Mask, in.Value)
	held := unmask(in.Mask, shown)

	props := make([]core.PropsAndChildren, 0, len(in.Style)+5)
	props = append(props, core.Width("100%"))
	if in.Keyboard != "" {
		props = append(props, core.Keyboard(in.Keyboard))
	}
	props = append(props, controlProps(in.Label, in.Hint, false)...)
	props = append(props, asProps(in.Style)...)
	// After the caller's styles, as Button orders it: being inert is not a
	// look, so a Style override must not re-enable the field.
	if in.Disabled {
		props = append(props, core.Disabled(true))
	}

	placeholder := in.Placeholder
	if placeholder == "" {
		placeholder = maskPlaceholder(in.Mask)
	}
	return core.Input(shown, placeholder, in.changed(held, capacity), props...).Render(ctx)
}

// changed builds the field's handler. held is the raw value this pass drew,
// captured and not re-read, as PINInput.changed explains: a handler runs
// against the tree that registered it.
func (in MaskedInput) changed(held string, capacity int) func(string) {
	return func(typed string) {
		if in.Disabled {
			return
		}
		raw := unmask(in.Mask, typed)
		if raw == held {
			// An echo of Go's own text, or a key the mask refused. Nothing to
			// report. The pass that follows every dispatch renders the same
			// value again, and a host showing the refused key is sent Go's
			// text back as a rewrite.
			return
		}
		if in.OnChange != nil {
			in.OnChange(raw, applyMask(in.Mask, raw))
		}
		if in.OnComplete != nil && capacity > 0 && len([]rune(raw)) == capacity {
			in.OnComplete(raw)
		}
	}
}

// maskPlaceholder is the mask with every slot drawn as an underscore.
func maskPlaceholder(mask string) string {
	var b strings.Builder
	for _, m := range mask {
		if _, slot := maskSlot(m); slot {
			b.WriteRune('_')
		} else {
			b.WriteRune(m)
		}
	}
	return b.String()
}
