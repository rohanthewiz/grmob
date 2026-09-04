package components

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/rohanthewiz/grmob/core"
)

// Avatar is the circular portrait that fronts a person in a list row, a
// header, or a comment: a remote image when there is one, and initials on a
// colored disc when there is not.
//
//	components.Avatar{Src: user.PhotoURL, Name: user.Name}   // image, labelled
//	components.Avatar{Name: "Ada Lovelace"}                  // "AL" on a disc
//
// # The circle
//
// Both branches are the same square with BorderRadius = Size/2. An oversized
// radius would be simpler (Badge uses 999 to get a stadium at any height), but
// a circle needs the radius to track the diameter exactly: a fixed 999 on a
// square still yields a circle, yet any caller Style that changes Size would
// silently keep the old geometry. Deriving it means Size stays the single
// knob.
//
// # A note on non-square images
//
// The iOS renderer scales images with .scaledToFit, so a portrait that is not
// square letterboxes inside the circle rather than filling it (Compose's
// AsyncImage defaults to Fit as well). That is a renderer-level choice shared
// with every other Image in the framework, not something Avatar can set from
// Go today — the fix is a ContentMode prop on Image, which would want its own
// pass across both renderers.
//
// # Accessibility
//
// Unlike ListRow, Avatar *does* synthesize a label, because it can: an avatar
// has exactly one meaning, the person it depicts, and Name is that meaning.
// The rule is:
//
//	AccessibilityLabel set  -> used verbatim
//	Name set                -> used as the label
//	neither                 -> the node is hidden from assistive tech
//
// The last case is the important one. An avatar with no name is decoration
// sitting next to text that already names the person, and an unlabeled image
// in that position is announced as an unhelpful "image" — or, worse, as its
// URL. Hiding it is the correct default rather than a fallback.
type Avatar struct {
	// Src is the image URL. Empty falls back to the initials disc.
	Src string

	// Name is the person's full name: the source of both the derived initials
	// and the accessibility label.
	Name string

	// Initials overrides what the fallback disc shows. Set it when the derived
	// pair is wrong — a mononym, a handle, a name whose ordering the
	// first-word/last-word rule gets backwards.
	Initials string

	// Size is the diameter in px; 0 means 40.
	Size float64

	// Background is the fallback disc's fill; empty uses the theme's Primary.
	// TextColor is the initials' ink; empty uses the theme's Background, which
	// is the palette's designated on-Primary color.
	Background string
	TextColor  string

	// Style is applied last, over both branches, so it can restyle the image
	// and the disc identically (a ring, a shadow, a square crop).
	Style []core.StyleProp

	// AccessibilityLabel names the avatar, overriding Name. See the type
	// comment for what happens when both are empty.
	AccessibilityLabel string
}

func (a Avatar) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	size := a.Size
	if size == 0 {
		size = 40
	}
	dim := fmt.Sprintf("%gpx", size)

	// Shared by both branches so an image avatar and an initials avatar are
	// interchangeable in a layout — same box, same corner, same label.
	shared := make([]core.StyleProp, 0, len(a.Style)+5)
	shared = append(shared,
		core.Width(dim),
		core.Height(dim),
		core.BorderRadius(size/2),
	)
	if label := a.label(); label != "" {
		shared = append(shared, core.AccessibilityLabel(label))
	} else {
		shared = append(shared, core.AccessibilityHidden())
	}
	shared = append(shared, a.Style...)

	if a.Src != "" {
		// core.Image bases every image on the theme's Camera style, whose
		// background is solid black — correct behind a viewfinder, wrong
		// behind a portrait that has not downloaded yet, where it shows as a
		// black disc until the bytes arrive. Surface is the neutral the rest
		// of the palette uses for a resting shape, so an image avatar loads in
		// as a grey circle and fills.
		bg := a.Background
		if bg == "" {
			bg = t.Colors.Surface
		}
		props := make([]core.StyleProp, 0, len(shared)+1)
		props = append(props, core.BackgroundColor(bg))
		props = append(props, shared...)
		return core.Image(a.Src, props...).Render(ctx)
	}

	// The disc's own default is Primary, not Surface: it is a filled badge
	// carrying legible initials, not a placeholder waiting on a network.
	bg := a.Background
	if bg == "" {
		bg = t.Colors.Primary
	}
	ink := a.TextColor
	if ink == "" {
		ink = t.Colors.Background
	}

	// Row rather than Box for the disc. Originally forced: Box drew as a
	// SwiftUI ZStack pinned to .topLeading and a Compose Box pinned to
	// TopStart, neither of which centers a child. Box is a vertical stack on
	// every target now, so it could centre one too — but a Row centres on the
	// same two props with no theme padding to think about, and the markup is
	// pinned by TestAvatarFallbackDisc, so the choice stands on its own merits
	// rather than being rewritten for a constraint that no longer exists.
	// Row honors JustifyContent on the main axis and AlignItems on the cross
	// axis on both platforms, which is what puts the initials in the middle.
	// Padding(0) undoes the theme Row's screen-level padding, which would
	// otherwise inflate the disc past Size.
	items := make([]core.PropsAndChildren, 0, len(shared)+5)
	items = append(items,
		core.Padding(0),
		core.BackgroundColor(bg),
		core.Justify(core.JustifyCenter),
		core.AlignItemsProp(core.AlignItemsCenter),
	)
	for _, sp := range shared {
		items = append(items, sp)
	}

	if initials := a.initials(); initials != "" {
		// Sized as a fraction of the diameter so the disc scales as one piece
		// — a fixed font size would overflow a 24px avatar and rattle around
		// in a 96px one. 0.4 keeps a two-letter pair comfortably inside the
		// circle's inscribed square (2/5 of the diameter per glyph row).
		items = append(items, core.Text(initials,
			core.FontSize(size*0.4),
			core.FontWeight(core.Bold),
			core.TextColor(ink),
		))
	}

	return core.Row(items...).Render(ctx)
}

// label resolves the accessibility name: explicit first, then Name, then none.
func (a Avatar) label() string {
	if a.AccessibilityLabel != "" {
		return a.AccessibilityLabel
	}
	return a.Name
}

// initials returns what the fallback disc shows: the explicit Initials when
// set, otherwise the first letter of the first and last words of Name.
//
// First-and-last rather than first-two, because "Ada King Lovelace" reads as
// AL, not AK. A single word yields one letter. The result is uppercased and
// rune-based throughout, so a name in a non-Latin script keeps whole
// characters instead of a mangled leading byte.
func (a Avatar) initials() string {
	if a.Initials != "" {
		return a.Initials
	}

	words := strings.Fields(a.Name)
	if len(words) == 0 {
		return ""
	}

	first := []rune(words[0])[0]
	out := []rune{unicode.ToUpper(first)}
	if len(words) > 1 {
		last := []rune(words[len(words)-1])[0]
		out = append(out, unicode.ToUpper(last))
	}
	return string(out)
}
