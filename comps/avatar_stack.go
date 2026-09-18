package comps

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/rohanthewiz/grmob/core"
)

// AvatarStack is the overlapping row of faces that says who is in a thread,
// who is going, who has seen it — with a "+N" disc when there are more than
// fit.
//
//	comps.AvatarStack{Avatars: attendees, Max: 4}
//
//	  ╭──╮╭──╮╭──╮╭──╮
//	 │AL││GH││KJ││+3│      Max 4 of 6: three faces and the surplus
//	  ╰──╯╰──╯╰──╯╰──╯
//
// # A ZStack, because a Row cannot overlap
//
// Overlap in a Row needs a negative margin or a negative gap, and neither is
// portable: CSS honours both, Compose's spacedBy and padding reject a negative
// value, and SwiftUI's stack spacing takes one but lays the row out wider
// than it draws. So the stack is a core.ZStack whose every layer is placed
// core.StackAlignStart — the leading edge, centred vertically — and pushed
// right by a MarginLeft that grows by one step per face. A margin on a stack
// layer is the same machinery Screen.Floating spends to hold a FAB off the
// corner, so nothing here is new to any renderer.
//
//	stack  Width  = ring + step·(n-1),  Height = ring
//	layer  2i     the ring disc,  MarginLeft(step·i)
//	layer  2i+1   the avatar,     MarginLeft(step·i + RingWidth)
//
// The stack's box is pinned, as core.ZStack asks: "start" of a box with no
// size is wherever the largest layer happens to end.
//
// # The ring is a layer, not a border
//
// Each face sits on a disc of the theme's Background a little larger than
// itself, which is what cuts the face under it and keeps two photos from
// blurring into one. It is drawn as its own layer rather than as a border on
// the Avatar because a border is sized differently across targets: the
// natives paint it inside the box, and the static export's content-box sizing
// paints it outside, so a bordered avatar is 4px wider on the web than on a
// phone and the overlap arithmetic would be off by the difference. A plain
// Box with a width, a height and a fill is the same size everywhere.
//
// # Later faces sit on top, and the surplus last of all
//
// Render order is paint order, so each face overlaps the one before it and
// the "+N" disc, drawn last, is never covered — it carries text, and a count
// half under a photo is a count nobody can read.
//
// # Max counts the surplus disc
//
// Max is the most discs drawn, the "+N" among them, so a Max of 4 is a stack
// four discs wide whatever the list's length — the width a layout reserves.
// Six avatars under Max 4 draw three faces and "+3". Max ≤ 0 draws every
// face.
//
// # Accessibility
//
// The stack is RoleImg with one name: "Ada Lovelace, Grace Hopper and 3
// others". A facepile is a picture of who is here, and the reader wants the
// sentence once rather than six images announced in turn — the case RoleImg
// exists for (see core/role.go), and every face inside is hidden behind it.
// The name counts the faces that were not drawn as well as the unnamed ones,
// because both are people the picture stands for. Label replaces it, for a
// caller that wants "6 attendees" or a language other than English.
//
// # Theme roles read
//
//	Faces       Avatar's own roles (Primary disc, Background initials)
//	Ring        Colors.Background
//	Surplus     Colors.Surface disc, TextSecondary count
type AvatarStack struct {
	// Avatars are the faces, drawn leading to trailing. Each one's Size is
	// replaced by the stack's, so a stack is one size throughout.
	Avatars []Avatar

	// Max is the most discs drawn, counting the "+N" disc. Zero or less draws
	// every face.
	Max int

	// Size is each face's diameter in px; 0 means 32, a row's worth.
	Size float64

	// Overlap is how much of each face the next one covers, as a fraction of
	// Size; 0 means 0.2. Kept below 0.5 by the reader's eye rather than by
	// the widget — past half, a face is more hidden than shown. The default is
	// set by initials rather than photos: a photo survives losing a third of
	// itself, but at 0.3 and at 0.25 the next disc and its ring cut into the
	// second letter of a two-letter pair (seen in a static export's serif,
	// the widest face any target draws). 0.2 is also what the web's common
	// avatar groups ship.
	Overlap float64

	// RingWidth is the gap drawn around each face in px; 0 means 2. Negative
	// draws no ring.
	RingWidth float64

	// RingColor is the ring's fill; empty takes the theme's Background, which
	// is right when the stack sits on the screen's own background. On a Card
	// or a Surface panel, pass that panel's fill.
	RingColor string

	// Label replaces the synthesized accessible name. See "Accessibility".
	Label string

	// Style is applied to the stack after its defaults.
	Style []core.StyleProp
}

// Render builds ZStack(ring, face, ring, face, …, ring, +N).
func (s AvatarStack) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	size := s.Size
	if size == 0 {
		size = 32
	}
	overlap := s.Overlap
	if overlap == 0 {
		overlap = 0.2
	}
	ring := s.RingWidth
	switch {
	case ring == 0:
		ring = 2
	case ring < 0:
		ring = 0
	}
	ringColor := orDefault(s.RingColor, t.Colors.Background)

	shown, surplus := s.split()
	discs := len(shown)
	if surplus > 0 {
		discs++
	}
	if discs == 0 {
		// Nobody to show. An empty picture named "0 people" would be read
		// out as a thing on the screen, so the stack is a hidden, sizeless
		// box: nothing to see and nothing to hear.
		return core.Box(core.AccessibilityHidden()).Render(ctx)
	}

	// Margins are whole pixels on every target, so the step is rounded once
	// here and every layer's offset is an exact multiple of it — rounding each
	// offset separately would let the gaps drift by a pixel along the row.
	step := int(size*(1-overlap) + 0.5)
	outer := size + 2*ring
	width := outer
	if discs > 1 {
		width += float64(step * (discs - 1))
	}

	items := make([]core.PropsAndChildren, 0, len(s.Style)+2*discs+4)
	items = append(items,
		core.Width(fmt.Sprintf("%gpx", width)),
		core.Height(fmt.Sprintf("%gpx", outer)),
		core.AccessibilityRole(core.RoleImg),
		core.AccessibilityLabel(orDefault(s.Label, s.spoken(shown, surplus))),
	)
	items = append(items, asProps(s.Style)...)

	// The ring is only worth a layer when it is drawn; a zero ring leaves the
	// face alone at its offset.
	inset := int(ring + 0.5)
	layer := func(i int, face Avatar) {
		left := step * i
		if ring > 0 {
			items = append(items, core.Box(
				core.Width(fmt.Sprintf("%gpx", outer)),
				core.Height(fmt.Sprintf("%gpx", outer)),
				core.BorderRadius(outer/2),
				core.BackgroundColor(ringColor),
				core.StackAlign(core.StackAlignStart),
				core.MarginLeft(left),
				core.AccessibilityHidden(),
			))
		}
		face.Size = size
		// After the caller's own Style on the face, so a face that named
		// itself is still hidden behind the stack's one name.
		face.Style = append(append([]core.StyleProp{}, face.Style...),
			core.StackAlign(core.StackAlignStart),
			core.MarginLeft(left+inset),
			core.AccessibilityHidden(),
		)
		items = append(items, face)
	}

	for i, face := range shown {
		layer(i, face)
	}
	if surplus > 0 {
		layer(len(shown), Avatar{
			Initials:   "+" + strconv.Itoa(surplus),
			Background: t.Colors.Surface,
			TextColor:  t.Colors.TextSecondary,
		})
	}

	return core.ZStack(items...).Render(ctx)
}

// split returns the faces to draw and how many are left over for the "+N"
// disc. The disc takes one of Max's places, so an overflowing list shows
// Max-1 faces; a list that fits exactly shows every face and no disc.
func (s AvatarStack) split() ([]Avatar, int) {
	if s.Max <= 0 || len(s.Avatars) <= s.Max {
		return s.Avatars, 0
	}
	keep := s.Max - 1
	return s.Avatars[:keep], len(s.Avatars) - keep
}

// spoken is the stack's synthesized name: the drawn faces that have a name,
// and a count of everyone else — the surplus, and any drawn face with no name.
//
//	"Ada Lovelace"
//	"Ada Lovelace and Grace Hopper"
//	"Ada Lovelace, Grace Hopper and 3 others"
//	"4 people"                       no face named at all
func (s AvatarStack) spoken(shown []Avatar, surplus int) string {
	names := make([]string, 0, len(shown))
	for _, a := range shown {
		if n := a.label(); n != "" {
			names = append(names, n)
		}
	}
	others := surplus + len(shown) - len(names)

	if len(names) == 0 {
		if others == 1 {
			return "1 person"
		}
		return strconv.Itoa(others) + " people"
	}
	if others > 0 {
		tail := "1 other"
		if others > 1 {
			tail = strconv.Itoa(others) + " others"
		}
		names = append(names, tail)
	}
	if len(names) == 1 {
		return names[0]
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}
