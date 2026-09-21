package comps

import (
	"strconv"

	"github.com/rohanthewiz/grmob/core"
)

// ConcernNumberPadInert is raised, in debug builds only, when a NumberPad has
// no OnKey and is not Disabled. Every digit then does nothing, on a widget
// that is twelve buttons and nothing else: there is no value on screen to
// show that the taps are going nowhere, so the report is the only sign.
const ConcernNumberPadInert = "number-pad-inert"

// numberPadKeyHeight is the least height of a key, in points. 56 is above the
// 48 every platform's guidance asks of a touch target, and close to the
// height of the system number pad's own keys, so a lock screen built on this
// does not feel smaller than the keyboard it stands in for.
const numberPadKeyHeight = 56

// NumberPad is an on-screen keypad: the ten digits, a backspace, and one
// corner key the caller chooses.
//
//	comps.NumberPad{
//	    OnKey:       func(k string) { pin.Set(pin.Get() + k) },
//	    OnBackspace: func() { pin.Set(dropLast(pin.Get())) },
//	}
//
//	┌───────┬───────┬───────┐
//	│   1   │   2   │   3   │
//	├───────┼───────┼───────┤
//	│   4   │   5   │   6   │
//	├───────┼───────┼───────┤
//	│   7   │   8   │   9   │
//	├───────┼───────┼───────┤
//	│ Extra │   0   │   ⌫   │   Extra: ".", "00", "+", or a blank cell
//	└───────┴───────┴───────┘
//
// # When to use it, and when not to
//
// core.Keyboard(core.KeyboardDigits) already asks the platform for its own
// number pad, and a field that wants digits should use that: it is the
// keyboard the reader knows, and on iOS it is what SMS autofill fills
// (PINInput uses it for that reason). This widget is for the three places the
// system pad does not reach:
//
//   - a lock or payment screen, where the pad is the screen and no field has
//     focus;
//   - a kiosk or a calculator, where the system keyboard must never appear;
//   - a static HTML export and the desktop web, where an inputmode hint does
//     nothing.
//
// # It holds no value
//
// The pad reports keys and the caller builds the string. That is what lets
// one widget serve a PIN (append, cap at four), an amount (one "." at most,
// two decimals) and a dialler ("+" only first) without a mode for each: the
// rule for what a key does to the value is the application's, and it is a
// line or two of Go in OnKey. It also means the pad takes no hooks, so it may
// be rendered conditionally.
//
// # Haptics
//
// Every key that reports gives the light tick CopyButton gives (core.Haptic
// with HapticLight). An on-screen key has no travel, and on a phone the tick
// is what says the tap landed; a keypad without it reads as unresponsive. A
// Disabled pad, and a key with no handler, stay silent: a tick would claim
// something happened.
//
// # Layout
//
// Four Rows of three equal shares (FlexGrow 1 over a zero basis, PINInput's
// rule for boxes that must not shuffle), each key at least 56 points tall.
// The pad takes the width it is given, so on a tablet a caller caps it with
// core.MaxWidth in Style. Keys are not forced square: core has no aspect
// ratio, and a key wider than it is tall is what the system pads draw.
//
// # Accessibility
//
// The pad is a core.RoleGroup named by Label ("Number pad"). Each key is a
// real Button whose label is its digit. Backspace is drawn as "⌫" and named
// BackspaceLabel ("Delete"), and Extra is named ExtraLabel when its glyph
// does not read well aloud ("." as "Decimal point"). The blank corner cell is
// disabled and hidden from accessibility, being there only to keep the zero
// in the middle.
//
// # Theme roles read
//
//	Keys   outlined Buttons: Components.Button's shape, PrimaryOnLight ink
//	Gap    Spacing.SM between keys and between rows
type NumberPad struct {
	// OnKey receives the key's text: "0" to "9", or Extra. Nil reports
	// ConcernNumberPadInert unless the pad is Disabled.
	OnKey func(string)

	// OnBackspace is called for the ⌫ key. Nil disables that key alone: a pad
	// that only ever appends (a dialler that clears with its own button) is
	// legitimate, and a dead key drawn as live is not.
	OnBackspace func()

	// Extra is the bottom-left key's text, reported through OnKey like a
	// digit: ".", "00", "+". Empty leaves the cell blank.
	Extra string

	// ExtraLabel is Extra's accessible name, for a glyph that does not read
	// well aloud. Empty means Extra itself.
	ExtraLabel string

	// BackspaceLabel is the ⌫ key's accessible name. Empty means "Delete".
	BackspaceLabel string

	// Label is the group's accessible name. Empty means "Number pad".
	Label string

	// Disabled greys every key and drops every report.
	Disabled bool

	// Style is applied to the outer column after its defaults.
	Style []core.StyleProp
}

// numberPadRows is the telephone layout (1 at the top), which is what every
// phone's own pad and lock screen use. A calculator's layout (7 at the top)
// is the other convention and is not offered: a field in a struct would be
// the only way to ask, and nobody has.
var numberPadRows = [3][3]string{
	{"1", "2", "3"},
	{"4", "5", "6"},
	{"7", "8", "9"},
}

// Render draws the four rows.
func (p NumberPad) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	if core.IsDebugMode() && p.OnKey == nil && !p.Disabled {
		core.ReportConcern(ConcernNumberPadInert,
			"NumberPad has no OnKey and is not Disabled, so every digit does nothing")
	}

	gap := float64(t.Spacing.SM)
	items := make([]core.PropsAndChildren, 0, len(p.Style)+8)
	items = append(items,
		core.Padding(0),
		core.Gap(gap),
		core.Width("100%"),
		core.AccessibilityRole(core.RoleGroup),
		core.AccessibilityLabel(orDefault(p.Label, "Number pad")),
	)
	items = append(items, asProps(p.Style)...)

	for _, row := range numberPadRows {
		items = append(items, p.row(gap,
			p.key(row[0], ""), p.key(row[1], ""), p.key(row[2], "")))
	}

	corner := p.blank()
	if p.Extra != "" {
		corner = p.key(p.Extra, p.ExtraLabel)
	}
	items = append(items, p.row(gap, corner, p.key("0", ""), p.backspace()))

	return core.Column(items...).Render(ctx)
}

// row lays three cells out in equal shares.
func (p NumberPad) row(gap float64, cells ...core.View) core.View {
	items := make([]core.PropsAndChildren, 0, len(cells)+3)
	items = append(items, core.Padding(0), core.Gap(gap), core.Width("100%"))
	for _, c := range cells {
		items = append(items, c)
	}
	return core.Row(items...)
}

// cellStyle is what makes a cell one third of its row on all four targets:
// the natives divide the axis by weight, and CSS divides only the leftover
// and needs the zero basis to start from (the note on PINInput's boxes).
func cellStyle() []core.StyleProp {
	return []core.StyleProp{
		core.FlexGrow(1),
		core.FlexBasis("0"),
		core.MinHeight(strconv.Itoa(numberPadKeyHeight) + "px"),
	}
}

// key is one reporting key. name is its accessible name when the text itself
// will not do.
func (p NumberPad) key(text, name string) core.View {
	var onTap func()
	if p.OnKey != nil {
		onTap = p.tap(func() { p.OnKey(text) })
	}
	return Button{
		Label:    text,
		OnTap:    onTap,
		Emphasis: EmphasisOutlined,
		// A key with no handler is drawn disabled, so the pad never shows a
		// live key that does nothing (the inert concern is the report).
		Disabled:           p.Disabled || onTap == nil,
		AccessibilityLabel: name,
		Style:              cellStyle(),
	}
}

// backspace is the ⌫ key. It is a key like any other, but reports through its
// own callback: "delete" is not text, and a caller switching on OnKey's
// argument should never have to compare it against a glyph.
func (p NumberPad) backspace() core.View {
	var onTap func()
	if p.OnBackspace != nil {
		onTap = p.tap(p.OnBackspace)
	}
	return Button{
		Label:              "⌫",
		OnTap:              onTap,
		Emphasis:           EmphasisOutlined,
		Disabled:           p.Disabled || onTap == nil,
		AccessibilityLabel: orDefault(p.BackspaceLabel, "Delete"),
		Style:              cellStyle(),
	}
}

// blank fills the corner when there is no Extra, so the zero stays under the
// eight. It is scenery: disabled, hidden from accessibility, and drawn at
// core.Opacity(0).
//
// It is a Button and not an empty Box because it has to be exactly as wide as
// a key. On the web a cell's share is its zero basis plus its own padding and
// border, so an empty Box came out narrower than the keys beside it and the
// zero sat off-centre under the eight (seen in headless Chrome). A key that
// is not painted has a key's box by construction, on every target, with no
// copy of the theme's button padding kept here to drift.
func (p NumberPad) blank() core.View {
	return Button{
		Label:    " ",
		Emphasis: EmphasisOutlined,
		Disabled: true,
		Style:    append(cellStyle(), core.Opacity(0), core.AccessibilityHidden()),
	}
}

// tap wraps a key's report with the guard and the haptic.
//
// The guard repeats Disabled on purpose. A native tap can arrive in the
// window between the press and the patch that disables the button (the race
// Button's Disabled doc describes), and on a payment screen that is a digit
// entered after the caller said stop.
func (p NumberPad) tap(report func()) func() {
	return func() {
		if p.Disabled {
			return
		}
		core.Haptic(core.HapticLight)
		report()
	}
}
