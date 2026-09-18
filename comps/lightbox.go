package comps

import (
	"fmt"

	"github.com/rohanthewiz/grmob/core"
)

// ConcernLightboxInescapable is raised, in debug builds only, when a Lightbox
// has no OnDismiss. The close button then closes nothing and the scrim is
// inert, so on the web the image covers the screen until the page is
// reloaded. Dialog allows a nil OnDismiss because a dialog can be one that
// must be answered; a lightbox asks nothing, so the only way out is the one
// this callback provides.
const ConcernLightboxInescapable = "lightbox-inescapable"

// Lightbox shows one image large, over everything, on a dark ground — the
// photo tapped in a feed, the receipt tapped in an expense.
//
//	comps.Lightbox{
//	    Src:       photo.URL,
//	    Alt:       photo.Description,
//	    Open:      viewing.Get(),
//	    OnDismiss: func() { viewing.Set(false) },
//	}
//
//	┌ Modal (near-black scrim) ─────────────────────┐
//	│ ┌ Column (black panel, full width) ─────────┐ │
//	│ │                                       [✕] │ │
//	│ ├ ImageWithMode(Fit) Width 100% ────────────┤ │
//	│ │                                           │ │
//	│ │             the whole image,              │ │
//	│ │       letterboxed, never cropped          │ │
//	│ │                                           │ │
//	│ ├───────────────────────────────────────────┤ │
//	│ │ Caption                                   │ │
//	│ └───────────────────────────────────────────┘ │
//	└───────────────────────────────────────────────┘
//
// # Dialog's plumbing, not Dialog
//
// It is a core.Modal driven exactly as Dialog drives one — controlled by
// Open, every way out reported through OnDismiss, nothing closed by the
// widget itself — with an image where the card would be. Dialog's Card would
// be the wrong frame: a white card with a heading around a photo is a
// document, and a lightbox is a viewer.
//
// # Fit, because a lightbox is for seeing all of it
//
// The image is core.ContentModeFit, the one mode that neither crops nor
// distorts. A thumbnail in the feed was Fill, cropped to its tile; the reason
// to open it is to see what the crop cut off.
//
// # Dark on every target, in every theme
//
// The scrim and the panel are near-black and black, and the close glyph and
// the caption are white, whatever the theme — photo viewers are dark in a
// light theme too, because a white ground around an image changes how its
// colours read. The panel is filled, not left transparent over the scrim,
// because SwiftUI presents a Modal as a sheet on the system's own background
// rather than over the scrim, and white ink on that sheet would vanish. Filled,
// the panel is the same black on all four targets.
//
// # Accessibility
//
// The image is named by Alt. With no Alt it is hidden, and the Caption, when
// there is one, is what is read — Avatar's rule that an unnamed image is
// better silent than announced as "image" or as its URL. The close button is
// named "Close", since "✕" is not a word. The chassis supplies the dialog
// role and modality on the web and the platform's own presentation on the
// natives, as it does for Dialog.
//
// # Theme roles read
//
//	None for colour — see "Dark on every target". Spacing.SM for the gaps
//	and for the close row's and the caption's inset (the image is full
//	bleed), Typography.Body for the caption.
type Lightbox struct {
	// Src is the image URL.
	Src string

	// Alt describes the image for assistive technology. Empty hides the
	// image from it; see "Accessibility".
	Alt string

	// Caption is drawn under the image, in white.
	Caption string

	// Open is the caller's open/closed state.
	Open bool

	// OnDismiss is called by the close button, a scrim tap and the
	// platform's dismiss gesture. Nil reports ConcernLightboxInescapable.
	OnDismiss func()

	// Height is the image's height in px; 0 means 400. The width is the
	// panel's, and Fit letterboxes within the two.
	Height float64

	// CloseLabel names the close button; empty gives "Close".
	CloseLabel string

	// Style is applied to the panel after its defaults.
	Style []core.StyleProp
}

// Lightbox's fixed colours. They are not theme roles because a lightbox is
// dark in every theme; see "Dark on every target, in every theme".
const (
	lightboxScrim = "#000000E6"
	lightboxPanel = "#000000"
	lightboxInk   = "#FFFFFF"
)

// Render builds Modal > Column(close, image, caption).
func (l Lightbox) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	if core.IsDebugMode() && l.OnDismiss == nil {
		core.ReportConcern(ConcernLightboxInescapable,
			"Lightbox has no OnDismiss, so neither its close button nor the scrim can close it")
	}

	height := l.Height
	if height == 0 {
		height = 400
	}

	img := []core.StyleProp{
		core.Width("100%"),
		core.Height(fmt.Sprintf("%gpx", height)),
	}
	if l.Alt != "" {
		img = append(img, core.AccessibilityLabel(l.Alt))
	} else {
		img = append(img, core.AccessibilityHidden())
	}

	onClose := l.OnDismiss
	if onClose == nil {
		onClose = func() {}
	}

	// The panel is full width and carries no padding of its own; the close
	// row and the caption take their inset and the image runs edge to edge.
	// Padding on a Width("100%") box is added *outside* the width under the
	// static export's content-box sizing, which pushed the panel 16px past
	// the window and the ✕ with it — and a full-bleed image is what a viewer
	// wants anyway.
	panel := make([]core.PropsAndChildren, 0, len(l.Style)+8)
	panel = append(panel,
		core.Width("100%"),
		core.Padding(0),
		core.Gap(float64(t.Spacing.SM)),
		core.BackgroundColor(lightboxPanel),
	)
	panel = append(panel, asProps(l.Style)...)
	panel = append(panel,
		// The close button in a row of its own, packed to the trailing edge,
		// so it is never drawn over the image it would hide a corner of.
		core.Row(
			core.Padding(t.Spacing.SM),
			core.Justify(core.JustifyEnd),
			Button{
				Label:              "✕",
				OnTap:              onClose,
				Emphasis:           EmphasisGhost,
				AccessibilityLabel: orDefault(l.CloseLabel, "Close"),
				Style:              []core.StyleProp{core.TextColor(lightboxInk)},
			},
		),
		core.ImageWithMode(l.Src, core.ContentModeFit, img...),
	)
	if l.Caption != "" {
		panel = append(panel, core.Text(l.Caption,
			core.UseStyle(t.Typography.Body),
			core.TextColor(lightboxInk),
			core.PaddingHorizontal(t.Spacing.SM),
			core.PaddingBottom(t.Spacing.SM),
		))
	}

	props := []core.ModalProp{
		core.Visible(l.Open),
		core.Backdrop(lightboxScrim),
		core.ModalContent(core.Column(panel...)),
	}
	// Registered only when set, as Dialog does; the nil case has been
	// reported above.
	if l.OnDismiss != nil {
		props = append(props, core.OnDismiss(l.OnDismiss))
	}
	return core.Modal(props...).Render(ctx)
}
