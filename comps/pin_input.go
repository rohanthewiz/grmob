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
// into: N single-character inputs in a row, with the cursor moving itself.
//
//	comps.PINInput{
//	    Length:     6,
//	    Value:      code.Get(),
//	    OnChange:   code.Set,
//	    OnComplete: func(c string) { verify(c) },
//	}
//
//	┌───┐ ┌───┐ ┌───┐ ┌───┐ ┌───┐ ┌───┐
//	│ 4 │ │ 1 │ │ 7 │ │ 2 │ │   │ │   │
//	└───┘ └───┘ └───┘ └───┘ └───┘ └───┘
//	                          ▲ the cursor, put there by the cell before it
//
// It is the first widget in the package to drive core's focus system, and
// that is the whole of what it adds over a Row of fields: a character typed
// into a cell moves the cursor to the next one, so a six-digit code is six
// keystrokes rather than six keystrokes and six taps.
//
// # The value is one string, and therefore a prefix
//
// Value is the whole code, not a cell array: cell i draws the i-th character
// and empty cells are the ones past the end. A plain string cannot hold a
// gap, so the cells fill strictly left to right and the two edits follow from
// that with no cases left over:
//
//	typing    Value = code[:i] + typed + whatever was past the typed run
//	clearing  Value = code[:i]        — everything from cell i on is dropped
//
// Clearing is the asymmetric one and it is worth being plain about. A cleared
// middle cell has to either shift the tail left — so cells the finger never
// touched change under it — or drop the tail. Dropping is the one a person
// can predict, because it is what "start again from here" means, and it is
// what backspacing through an OTP field amounts to on every platform that has
// one.
//
// The same invariant answers a question the cells can otherwise ask: a
// character typed into a cell past the end of the code (the web lets a click
// land anywhere) lands at the end instead, because there is no position for
// it to occupy.
//
// # A paste and a second character are the same event
//
// A cell whose OnChange arrives with more than one character is a paste — the
// whole code dropped into the first box — and it is also what typing into an
// already-full cell looks like, since the field is controlled and reports its
// entire contents. Both are handled as one rule: **the incoming string is
// written from this cell forward, and the cursor lands after the last cell it
// filled.** A six-character paste into cell 0 fills the field; a "2" typed
// into a cell already holding "1" arrives as "12", rewrites cell 0 with the
// character that was already there and puts the new one in cell 1. Characters
// past the last cell are dropped.
//
// The one case it reads wrongly is a character inserted *before* an existing
// one (the caret parked at the left edge of a full cell), which arrives as
// "21" and is written in that order. Nothing in the event says where the caret
// was, so no widget here can tell the two apart.
//
// # Backspace on an empty cell does nothing, and cannot
//
// There are no key events in this framework — a field reports its text, not
// the keys that produced it — so a backspace in an *empty* cell changes
// nothing and is therefore never reported. The cursor stays where it is, and
// clearing a run of cells means one backspace per cell with a tap in between,
// or one backspace in the leftmost filled cell, which drops everything after
// it by the rule above. Document it to callers rather than working around it:
// the workaround is a key channel, and that is a renderer change.
//
// # OnComplete fires on every change that leaves the code full
//
// Not once per crossing, which is what Countdown.OnDone does and is
// deliberately not what this does. A caller's OnComplete is "submit the code",
// and a person who mistypes one digit of a full code, corrects it, and gets
// silence has a field that will not submit. So a complete code re-reports
// whenever it changes.
//
// It fires from the change handler rather than from an effect, so it never
// fires for a Value that merely arrived complete — a screen restored with a
// code already in it does not resubmit itself on mount.
//
// A change that produces the value already held is treated as an echo: no
// OnChange, no cursor move, no OnComplete. Both natives can report their own
// text back after a Go-side update, and none of the three is worth doing
// twice.
//
// # It holds hooks, so it is not conditional-safe
//
// One FocusRef per cell, and refs must be stable across passes or a focus
// command aims at last pass's identity. So this is a hook caller with
// Accordion's rule: render it in a stable position every pass rather than
// inside a core.If.
//
// The hook count does not follow Length. It follows the largest Length this
// widget has ever been rendered with, held in one slot of its own, because a
// Length that shrank between passes would otherwise retire hook slots from
// the middle of the sequence and drift every cursor after them. Growing is
// safe — new slots are appended past the ones already bound — and never
// shrinking is what makes it so. The cost is a handful of FocusRefs that
// nothing points at, which cost a slice entry each and are never stamped onto
// a node.
//
// # What the cells are, and what they are not
//
// Each cell is an ordinary core.Input (core.InputPassword when Secure), so it
// wears the theme's field frame and matches the text inputs above it in a
// form. They divide the row equally — core.FlexGrow with a zero core.FlexBasis,
// the pair Calendar's day cells use, which is what makes the four targets
// agree on "equal shares" rather than "equal shares of the leftovers". The row
// therefore fills the width it is given; cap it with Style.
//
// They take the platform's text keyboard, not its number pad. The keyboard
// type is chosen by node type on both natives — "NumericInput" is the numeric
// one — and that node carries an int value, which cannot express an empty
// cell: clearing one would report nothing at all, so backspace would stop
// working entirely. A digits-only keyboard needs a keyboard-type prop on
// core.Input, which is a renderer change and not this widget's to make.
//
// # Accessibility
//
// The row is a core.RoleGroup named by Label, and each cell is named
// "<Label>, N of M" so a reader moving between them says which box it is in.
// Label is the accessible name only — there is no visible caption, as with
// InputRow; wrap this in a FormField when one is wanted.
//
// # Theme roles read
//
//	Cells   Components.Input — the same frame every other field in the form has
//	Gap     Spacing.SM between cells
type PINInput struct {
	// Length is the number of cells. Zero means six, the one-time code length.
	Length int

	// Value is the code so far, in full. The field is controlled: it draws
	// exactly this, one character per cell from the left, and OnChange is the
	// only way it changes.
	Value string

	// OnChange receives the whole code after every edit, never a single cell.
	// Without it the field is read-only and reports ConcernPINInputInert.
	OnChange func(string)

	// OnComplete receives the code on every edit that leaves it as long as
	// the field — including an edit to a code that was already complete. Nil
	// is a field the caller reads from Value instead.
	OnComplete func(string)

	// Secure masks the characters, as a device PIN rather than an emailed
	// code. The cells become core.InputPassword.
	Secure bool

	// Label is the accessible name of the group and the stem of each cell's
	// name. Empty means "Code". It draws nothing.
	Label string

	// Style is applied to the row, after the gap and the accessibility pair,
	// so a caller can override any of them — or cap the width, which is the
	// common one: MaxWidth stops four cells from spreading across a tablet.
	Style []core.StyleProp
}

// Render allocates the refs, declares their order and draws the cells.
func (p PINInput) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()
	n := p.length()

	// The high-water mark, in a slot of its own ahead of the refs so the
	// sequence below it never moves. A pointer rather than the slot's value
	// because it is bumped during a render pass: State.Set would request a
	// render of the whole tree for a number no tree reads. See "It holds
	// hooks" for why the count may only grow.
	//
	// The two-step through a variable is UseFocusRef's: State's accessors
	// have pointer receivers and NewState's return value is not addressable.
	slot := core.NewState(ctx, new(int))
	high := slot.Get()
	if n > *high {
		*high = n
	}
	refs := make([]*core.FocusRef, *high)
	for i := range refs {
		refs[i] = core.UseFocusRef(ctx)
	}
	// Only the live cells are in the order, so the keyboard's Next key walks
	// the field and stops at its end rather than at the high-water mark.
	core.UseFocusOrder(ctx, refs[:n]...)

	code := []rune(p.Value)
	if core.IsDebugMode() {
		if p.OnChange == nil {
			core.ReportConcern(ConcernPINInputInert,
				"PINInput has no OnChange, so every keystroke is discarded and the field can never be filled")
		}
		if len(code) > n {
			core.ReportConcern(ConcernPINValueTooLong, fmt.Sprintf(
				"PINInput has %d cells and a Value of %d characters: the last %d are never drawn and cannot be edited",
				n, len(code), len(code)-n))
		}
	}
	if len(code) > n {
		// Drawn as the field can hold it. The concern above is the report;
		// truncating here is what keeps the cells and the cursor arithmetic
		// working off one length.
		code = code[:n]
	}

	label := p.Label
	if label == "" {
		label = "Code"
	}

	items := make([]core.PropsAndChildren, 0, len(p.Style)+n+4)
	items = append(items,
		// Row's theme padding is screen-level and would inset the cells away
		// from whatever is above them; the gap is the widget's own layout, as
		// InputRow's is.
		core.Padding(0),
		core.Gap(float64(t.Spacing.SM)),
		core.AccessibilityRole(core.RoleGroup),
		core.AccessibilityLabel(label),
	)
	items = append(items, asProps(p.Style)...)

	for i := 0; i < n; i++ {
		ch := ""
		if i < len(code) {
			ch = string(code[i])
		}
		cell := []core.PropsAndChildren{
			// Equal shares on all four targets: the natives divide the axis
			// by weight and ignore the basis, CSS divides only the leftover
			// and needs the zero to start from. Without it a filled cell
			// would be a hair wider than an empty one and the boxes would
			// shuffle as the code is typed.
			core.FlexGrow(1),
			core.FlexBasis("0"),
			core.Align(core.AlignCenter),
			core.FocusTarget(refs[i]),
			core.AccessibilityLabel(fmt.Sprintf("%s, %d of %d", label, i+1, n)),
		}
		onChange := p.cellChanged(code, refs, i)
		if p.Secure {
			items = append(items, core.InputPassword(ch, "", onChange, cell...))
		} else {
			items = append(items, core.Input(ch, "", onChange, cell...))
		}
	}

	return core.Row(items...).Render(ctx)
}

// length is Length with its default applied.
func (p PINInput) length() int {
	if p.Length <= 0 {
		return defaultPINLength
	}
	return p.Length
}

// cellChanged builds cell i's handler. code is the pass's drawn code and refs
// its cells, both captured rather than re-read: a handler dispatched from the
// registry runs against the tree that registered it, which is exactly the
// state the user was looking at when they typed.
func (p PINInput) cellChanged(code []rune, refs []*core.FocusRef, i int) func(string) {
	return func(typed string) {
		next, last := p.write(code, i, typed)
		if next == string(code) {
			// An echo, or a retyped character: nothing changed, so nothing
			// happens — including the cursor, which must not jump on a
			// renderer's own report of the text Go just gave it.
			return
		}
		if p.OnChange != nil {
			p.OnChange(next)
		}
		if last >= 0 {
			// Off the end of the order this is a no-op, which is what keeps
			// the cursor in the last cell once the code is full.
			core.FocusNext(refs[last])
		}
		if p.OnComplete != nil && len([]rune(next)) == p.length() {
			p.OnComplete(next)
		}
	}
}

// write applies typed at cell i and returns the new code, together with the
// index of the last cell it filled — or -1 when it filled none, which is the
// cleared case and the one where the cursor stays put.
//
// code must already be no longer than the field; Render truncates it.
func (p PINInput) write(code []rune, i int, typed string) (string, int) {
	n := p.length()
	// No holes: a cell past the end of the code has no position of its own,
	// so an edit aimed at one is an edit at the end. Clamping here rather
	// than at each use keeps the two branches below reading as the rule.
	if i > len(code) {
		i = len(code)
	}

	in := []rune(typed)
	if len(in) == 0 {
		// Cleared: the tail goes with it. See "The value is one string".
		return string(code[:i]), -1
	}

	// The run this write covers, clipped to the field. A paste longer than
	// the cells left loses its overflow rather than wrapping or growing the
	// value past what the widget can show.
	end := i + len(in)
	if end > n {
		end = n
		in = in[:end-i]
	}

	out := make([]rune, 0, n)
	out = append(out, code[:i]...)
	out = append(out, in...)
	if end < len(code) {
		// Whatever the run did not cover survives: typing over one cell of a
		// full code replaces that cell and nothing else.
		out = append(out, code[end:]...)
	}
	return string(out), end - 1
}
