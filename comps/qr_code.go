package comps

import (
	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/internal/qr"
)

// ECLevel is a QR Code's error-correction level: how much of the symbol is
// redundancy, and so how much of it may be covered, smudged or reflected off
// and still read.
//
// A string enum with an empty zero value, the package's idiom (see Variant),
// so that adding the field to an existing QRCode changes nothing. The values
// are the specification's own one-letter names, which is what every other QR
// tool a developer will compare against prints.
type ECLevel string

const (
	// ECDefault is the zero value and means ECMedium.
	ECDefault ECLevel = ""

	// ECLow recovers about 7% of the symbol, ECMedium about 15%, ECQuartile
	// about 25% and ECHigh about 30%.
	ECLow      ECLevel = "L"
	ECMedium   ECLevel = "M"
	ECQuartile ECLevel = "Q"
	ECHigh     ECLevel = "H"
)

// level resolves the enum to the encoder's, defaulting to Medium.
//
// Medium rather than High is the default because the failure this widget
// actually faces is not damage — a code on a screen is pristine — but *module
// size*: a phone camera has to resolve each module, and at a fixed drawn width
// a higher level means a larger symbol means smaller modules. Medium is the
// level a payment or pairing code is normally printed at for that reason.
func (l ECLevel) level() qr.Level {
	switch l {
	case ECLow:
		return qr.Low
	case ECQuartile:
		return qr.Quartile
	case ECHigh:
		return qr.High
	default:
		return qr.Medium
	}
}

// ConcernQRDataTooLong is raised, in debug builds only, when Data is longer
// than any QR Code can hold at the requested level. The widget then draws
// nothing: there is no half of a QR Code that is worth showing, and a symbol
// that encodes a truncated URL is worse than a blank space because it scans.
const ConcernQRDataTooLong = "qr-data-too-long"

// qrDefaultSize is the box's side in px when Size is not given: large enough
// that a version-4 symbol's modules land near 4px on a phone, which is about
// where camera decoding stops needing a steady hand.
const qrDefaultSize = 160

// qrDefaultQuiet is the light margin around the symbol, in modules. Four is
// the specification's minimum, and a reader that cannot find the margin does
// not find the symbol at all.
const qrDefaultQuiet = 4

// QRCode draws its Data as a QR Code: a core.Canvas of module rectangles,
// encoded in Go, with no image file, no network round trip and no dependency
// outside this module.
//
//	comps.QRCode{Data: "cats://pair?t=9f2c1a", Label: "Pairing code"}
//
// # What is drawn
//
// Two shapes and no more: one light rectangle covering the whole box, and one
// path holding every dark module. The modules go in a single path rather than
// a Shape each for two reasons. A version-10 symbol has some three thousand
// modules, and three thousand child nodes is a reconciler's worst case for a
// drawing that is either identical between passes or wholly different. And a
// single path is filled once, so adjacent modules have no seam between them —
// separate shapes would be antialiased against each other and leave hairlines
// that a decoder's binarizer can read as light.
//
// Within a row, consecutive dark modules are merged into one rectangle, which
// costs one comparison per module and typically halves the path.
//
//	██ ██████ ██      one row, four runs
//	└┘ └────┘ └┘      four rectangles, not eight
//
// # Size, and where the quiet zone comes from
//
// Size is the side of the whole box in px, the quiet zone included. The
// symbol itself is therefore Size × n/(n+2·Quiet) across, where n is the
// version's module count — a 160px box at the default quiet zone of 4 gives a
// 29-module version-3 symbol about 125px of picture and 17px of margin.
//
// Putting the margin inside the box rather than outside it is what makes the
// widget's footprint predictable: a caller lays out a 160px square and gets
// one, whatever the data does to the version.
//
// # Colour is not themed, and that is deliberate
//
// A QR Code is read by a camera, not by a person, and every decoder's
// binarizer assumes dark modules on a light field. Painting one in a dark
// theme's colours — light ink on a dark surface — produces a symbol that many
// readers simply will not see, and the ones that do invert are the exception.
//
// So the widget draws dark-on-light always. It uses the theme's own ink and
// surface when those *are* dark-on-light with room to spare, so that a light
// theme's code sits in the page rather than on a hard white patch; otherwise
// it falls back to black on white, which in a dark theme means a white square
// — the same thing every banking and payment app shows, for the same reason.
//
// There is no Foreground or Background field. Every colour a caller could pass
// is either the pair already chosen or a worse one, and an unscannable QR code
// fails silently: it looks exactly like a working one.
//
// # Cost
//
// Encoding runs on every render pass: choosing a version, laying out the
// blocks, and then scoring all eight data masks to pick one. For a link-sized
// payload that is around 0.2 ms on a current phone-class core, which is under
// a hundredth of a frame and not worth caching for a screen that shows a code
// and waits.
//
// A code that sits in a tree re-rendering every frame — an animation, or a
// list being scrolled — is a different matter, and the lever is core.Cached,
// which QRCode is a fit for: it holds no hooks and registers no callbacks.
//
//	core.Cached(comps.QRCode{Data: link, Label: "Pairing code"})
//
// # Accessibility
//
// One image element, named by Label ("QR code" when empty). The data is never
// spoken: a reader announcing a 300-character URL one character at a time
// helps nobody, and a person who needs the link needs it as a link. Give the
// code a Label that says what scanning it will do, and put the underlying
// action on screen as well where you can.
type QRCode struct {
	// Data is encoded in byte mode, so any string is legal — the only input
	// that cannot be drawn is one too long for a version-40 symbol (2953
	// bytes at ECLow, 1273 at ECHigh).
	Data string

	// Size is the box's side in px, quiet zone included; 0 means 160.
	Size float64

	// Level is the error-correction level; the zero value is ECMedium.
	Level ECLevel

	// Quiet is the light margin in modules; 0 means the specification's four.
	// A negative Quiet draws none, for a caller who is supplying the margin
	// from the surrounding layout instead.
	Quiet int

	// Label names the code to assistive tech; empty says "QR code".
	Label string

	// Style is applied last, to the canvas.
	Style []core.StyleProp
}

func (q QRCode) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	size := q.Size
	if size <= 0 {
		size = qrDefaultSize
	}
	quiet := q.Quiet
	if quiet == 0 {
		quiet = qrDefaultQuiet
	} else if quiet < 0 {
		quiet = 0
	}
	dim := px(size)

	code, err := qr.Encode(q.Data, q.Level.level())
	if err != nil {
		if core.IsDebugMode() {
			core.ReportConcern(ConcernQRDataTooLong,
				"QRCode.Data is "+itoa(len(q.Data))+" bytes, too long for a QR Code at level "+q.Level.level().String())
		}
		// The box is still reserved, so a screen laid out around the code does
		// not reflow when the data is fixed. Nothing is drawn in it and nothing
		// is announced: an empty square is honest about there being no code,
		// where a partial one would not be.
		return core.Box(core.Width(dim), core.Height(dim)).Render(ctx)
	}

	fg, bg := qrColors(t)
	view := float64(code.Size + 2*quiet)

	items := make([]core.PropsAndChildren, 0, 3+len(q.Style))
	items = append(items, core.Width(dim), core.Height(dim))
	for _, sp := range q.Style {
		items = append(items, sp)
	}

	return core.Canvas(view, view, []core.Shape{
		{Path: core.Rect(0, 0, view, view), Fill: bg},
		{Path: qrModulePath(code, quiet), Fill: fg},
	}, append(items, core.AccessibilityLabel(q.label()))...).Render(ctx)
}

// qrModulePath walks the symbol row by row and emits one rectangle per run of
// consecutive dark modules, offset by the quiet zone. One unit of the viewBox
// is one module, so the rectangles are on integer coordinates and every
// renderer's rasteriser lands them on the same pixels.
func qrModulePath(code *qr.Code, quiet int) *core.Path {
	p := core.NewPath()
	for y := 0; y < code.Size; y++ {
		for x := 0; x < code.Size; x++ {
			if !code.Dark(x, y) {
				continue
			}
			// Take the whole run in one rectangle.
			run := 1
			for x+run < code.Size && code.Dark(x+run, y) {
				run++
			}
			fx, fy, fw := float64(x+quiet), float64(y+quiet), float64(run)
			p.MoveTo(fx, fy).LineTo(fx+fw, fy).LineTo(fx+fw, fy+1).LineTo(fx, fy+1).Close()
			x += run - 1
		}
	}
	return p
}

// qrColors picks the dark and light colours; see "Colour is not themed" above.
//
// The thresholds are generous on purpose. A surface has to be clearly light
// and an ink clearly dark before the theme's pair is used at all, because the
// cost of being wrong is asymmetric: a slightly whiter-than-necessary patch is
// an aesthetic complaint, and a low-contrast symbol is a code that does not
// work and gives no sign of why.
func qrColors(t *core.Theme) (dark, light string) {
	surface, okSurface := relativeLuminance(t.Colors.Surface)
	ink, okInk := relativeLuminance(t.Colors.TextPrimary)
	if okSurface && okInk && surface > 0.6 && ink < 0.15 {
		return t.Colors.TextPrimary, t.Colors.Surface
	}
	return "#000000", "#FFFFFF"
}

func (q QRCode) label() string {
	if q.Label != "" {
		return q.Label
	}
	return "QR code"
}
