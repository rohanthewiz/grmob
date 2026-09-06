package components

import "github.com/rohanthewiz/grmob/core"

// The package's heading outline, and the one knob that lets a caller move a
// widget within it.
//
// core.AccessibilityHeadingLevel has run 1..6 since it existed, and for a while
// the framework only ever said 1 or 2: an AppBar's title is the screen's name
// and a GroupedList band is a section of that screen. Everything below the band
// — a card's title, an accordion's question — drew at heading weight and
// announced as prose, so a reader navigating by heading fell from the section
// straight to the end of the screen.
//
//	AppBar.Title      1   fixed by construction: an AppBar is the screen's own
//	                      bar, so there is nothing above it to be a section of
//	GroupHeader       2   a band titles a run of rows within a screen
//	Card.Title        2   a card is a section of a screen, same tier as a band
//	Accordion.Title   3   a disclosure sits inside one of those
//
// # Why three of those four are defaults and one is not
//
// A widget can state its own tier only where its position is fixed. An AppBar's
// is: it belongs to the screen it bars. The other three are *usually* where the
// table says and can legitimately be anywhere — a grouped list inside a card is
// a tier deeper than the card, a screen made entirely of accordions has them at
// the top, and a bandless feed on a screen with no bar starts at 2 with no 1
// above it, which GroupHeader used to document as the lesser of two wrongs and
// no longer has to.
//
// So those three take a HeadingLevel field, and this resolves it: zero — the
// value every caller written before the field has — means "the tier in the
// table", and anything else means the caller knows where the widget landed.
//
// # Asking for a heading with no tier at all
//
// Pass a negative level. core.Style.AccessibilityHeadingLevel drops rather than
// clamps an out-of-range value, so it survives to the exporters unchanged and
// is written by none of them, which is exactly "a heading, tier unstated" — the
// announcement every heading in the package made before any of this. It needs
// no case of its own here, and deliberately: a rule that special-cased it would
// have to pick a spelling, and the range rule already has one.
func headingLevel(asked, own int) int {
	if asked == 0 {
		return own
	}
	return asked
}

// The tiers the table above records, named so that "a card and a band are the
// same tier" is stated once and read at both sites rather than being two 2s
// that agree by coincidence. TestTheDefaultHeadingOutline spells the numbers
// out in literals instead, so a change here has to be a change there too.
const (
	headingLevelSection    = 2 // a card, a band: a section of a screen
	headingLevelSubsection = 3 // an accordion: a disclosure within a section
)

// headingProps is the pair every heading in this package sets together: the
// role that says a node *is* a heading and the tier that says where it sits.
//
// They travel as one because neither is much use alone. A level with no role is
// dropped by every target that reads it — ARIA scopes aria-level to the role,
// and so do both web exporters — and a role with no level announces a heading
// that a reader cannot place, which is the state this file exists to end.
func headingProps(asked, own int) []core.StyleProp {
	return []core.StyleProp{
		core.AccessibilityRole(core.RoleHeading),
		core.AccessibilityHeadingLevel(headingLevel(asked, own)),
	}
}
