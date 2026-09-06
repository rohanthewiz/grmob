package social

import (
	"github.com/rohanthewiz/grmob/components"
	"github.com/rohanthewiz/grmob/core"
)

// TabPanelID is the id of the region the bottom tab bar switches, and the
// string every tab's core.AccessibilityControls points at.
//
// A constant rather than three copies of a literal, because the whole value of
// an IDREF is that both ends agree: a typo in one of them is not an error
// anywhere, it is a tab that announces itself as controlling a region that
// does not exist. One name, referenced twice, is what makes that unwritable.
//
// The "grmob-" prefix is reserved for core.TabView's own minted ids (see
// core.Style.AccessibilityID), which is why this one does not use it.
const TabPanelID = "social-tab-panel"

// TabButton is one glyph in the bottom tab bar.
//
// It is a ghost button, and that fixes a real bug rather than restyling
// anything. This used to be a bare core.Button with a TextColor override and
// no Background override — which meant it inherited the theme's Button base,
// a solid Primary pill (#007AFF at the time; the role has since been darkened
// to Apple's accessible blue). The active tab therefore rendered the accent on
// itself, an invisible glyph at 1:1 contrast, and the inactive one #555 at
// about 2.6:1. Setting only half of a color pair is exactly the mistake the
// widget's emphasis axis exists to prevent: EmphasisGhost punches the fill out
// instead of leaving it to whatever the theme put there.
//
// Wrapped in a ComponentFunc so the inactive tint can come from the palette.
// The function returns a View before any render happens, so there is no theme
// to read until a Context arrives.
//
// # Why the strip is marked up by hand
//
// This bar is the shape core.TabView exists for and deliberately does not use
// it: the example switches pages with core.Match on a piece of state, which is
// how an app with a custom-looking strip has to do it. TabView owns its own
// chrome, so anything that wants a different bar gives up the whole wiring
// TabView writes from the node type — the tablist and tab roles, the selected
// state, and the aria-controls that says which region each tab shows.
//
// All four are stated here instead, which is what the pair of IDREF props was
// added for. What a reader gets out of it:
//
//	role="tablist"    on the Row, so the three buttons are announced as a set
//	                  of tabs rather than three buttons that happen to be
//	                  adjacent. See core/role.go on what a structural role
//	                  claims — a strip holding a "+" button would not qualify.
//	role="tab"        on each button, replacing the role its <button> tag
//	                  implies, which is the honest upgrade: a tab *is* a
//	                  button plus the fact that it belongs to a set.
//	aria-selected     on every tab, not only the live one. A strip in which
//	                  nothing says it is selected announces all three as
//	                  unselected — core.SelectedOff is a value with a job.
//	id/aria-controls  the relationship. Without it a reader announces "tab 1
//	                  of 3" and has no way to reach what the tab governs.
//
// The keyboard half of ARIA's tablist pattern — one tab stop for the strip,
// arrow keys within it — is not written here either, and does not need to be.
// The WASM runtime builds it from exactly the four things above: role="tablist"
// says what the strip is, role="tab" says what its members are, aria-selected
// says where a keyboard should enter, and the Row's own axis says which arrows
// move. This file gained the behaviour without gaining a line, which is the
// argument for the roles being a vocabulary rather than a widget.
//
// Both phones navigate a strip by swipe and never had the gap; a static
// htmlout export still does, deliberately — see core.RoleListBox.
//
// name is what the glyph is read as: an emoji alone announces as whatever the
// reader's emoji dictionary calls it ("house building"), which is not the name
// of a destination.
func TabButton(icon, tab, name string, selected core.State[string]) core.View {
	return core.ComponentFunc(func(ctx *core.Context) *core.Node {
		on := selected.Get() == tab
		b := components.Button{
			Label:    icon,
			OnTap:    func() { selected.Set(tab) },
			Emphasis: components.EmphasisGhost,
			// AccessibilityLabel rather than a hidden glyph plus text: the
			// Button widget names the control and the label replaces what the
			// reader would otherwise make of the emoji.
			AccessibilityLabel: name,
			Style: []core.StyleProp{
				core.FontSize(20),
				core.Padding(12),
				core.Align(core.AlignCenter),
				core.AccessibilityRole(core.RoleTab),
				// Both values, which is the whole reason SelectedState has
				// three of them — see the type comment above.
				core.AccessibilitySelected(core.SelectedWhen(on)),
				// The tab's own id is not pointed at by anything here; it is
				// written because an element that can be the target of a
				// relationship should be nameable, and because a panel that
				// one day wants aria-labelledby back has somewhere to point.
				core.AccessibilityID("social-tab-" + tab),
				core.AccessibilityControls(TabPanelID),
			},
		}
		// Ghost already inks the label with the variant's color — the theme's
		// Primary — which is the selected look. Only the unselected tab needs
		// to say anything, and it dims to the palette's secondary ink rather
		// than to the literal grey this carried before.
		if !on {
			b.Style = append(b.Style, core.TextColor(ctx.Theme().Colors.TextSecondary))
		}
		return b.Render(ctx)
	})
}
