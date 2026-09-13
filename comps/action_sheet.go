package comps

import "github.com/rohanthewiz/grmob/core"

// ActionSheet is the "Share / Copy link / Delete" list that rises from the
// bottom edge: a short column of full-width actions and a separate Cancel,
// drawn over the screen in a core.Modal.
//
//	comps.ActionSheet{
//	    Visible: open.Get(),
//	    Title:   "Note",
//	    Actions: []comps.SheetAction{
//	        {Label: "Share", OnTap: share},
//	        {Label: "Delete", Variant: comps.VariantError, OnTap: del},
//	    },
//	    Cancel:    "Cancel",
//	    OnDismiss: func() { open.Set(false) },
//	}
//
// # The placement question, and what answered it
//
// core.Modal has no placement prop, and the four hosts do not agree on where
// a Modal's content goes:
//
//	target    Modal chassis                           content lands
//	───────   ─────────────────────────────────────   ─────────────────────
//	web ×2    fixed inset-0 flex column, align and    centred
//	          justify center (htmlout modalChassis,
//	          styleFromGrMob)
//	Compose   Dialog window > Column(fillMaxWidth)    centred window;
//	          > ColumnChildren (FlexGrow → weight)    children weighted
//	SwiftUI   .sheet, detents medium/large >          already the bottom
//	          VStack > PlainChildren (grow ignored)   sheet, top of it
//
// So the widget puts two children in the Modal instead of one:
//
//	┌ Modal ──────────────────────────────────────────┐
//	│ ┌ Box FlexGrow(1)  filler, tap = OnDismiss ───┐ │
//	│ │                                             │ │  ← pushes the card
//	│ └─────────────────────────────────────────────┘ │    down on web and
//	│ ┌ Card  Width 100%  AccessibilityLabel(Title) ┐ │    Compose; zero
//	│ │ Title                                       │ │    height on iOS
//	│ │ [ Share                                   ] │ │
//	│ │ [ Delete                                  ] │ │
//	│ │ ─────────────────────────────────────────── │ │
//	│ │ [ Cancel                                  ] │ │
//	│ └─────────────────────────────────────────────┘ │
//	└─────────────────────────────────────────────────┘
//
// Each host then does the right thing with no host change:
//
//   - Web: the filler grows along the overlay's column and the card sits on
//     the bottom edge. The overlay's align-items:center would shrink both
//     children to their content, so the filler takes AlignSelf(stretch) (a
//     web-only prop, which is exactly where it is needed) and the card takes
//     Width("100%").
//   - Compose: the filler is weighted, so the Column fills the dialog window's
//     height and the card is last in it. The card paints a background, which
//     is what tells GrMobModal not to add its own white surface around the
//     whole column.
//   - SwiftUI: the filler has no content and grow is not read inside a sheet,
//     so it is zero-height and the card is the sheet's content. The card's
//     background becomes the sheet's surface (GrMobModal reads the first child
//     that paints one; the filler paints none).
//
// The filler carries the scrim's job. On the web a backdrop tap is only
// reported when it lands on the overlay element itself, and on Compose the
// filler is inside the dialog window, so a tap above the card would do
// nothing unless the filler reports it. It is hidden from assistive
// technology: the Cancel action is the accessible way out, as the scrim is.
//
// Not device-verified: the Compose and SwiftUI rows above come from reading
// Renderer.kt and Renderer.swift, not from a run on hardware.
//
// # Actions are buttons, not a listbox
//
// The plan that proposed this widget suggested RoleListBox with RoleOption
// items. That pair is for choosing a value: an option states selected or not
// selected, and a reader would announce "Delete, not selected". An action
// sheet is a set of commands. ARIA's word for that is a menu, which core.Role
// does not have, so each action is a real comps.Button (a button on every
// target) inside the labelled card, and the Modal chassis supplies the dialog
// semantics as it does for Dialog.
//
// # Picking an action closes the sheet
//
// Unlike Dialog's Confirm, an action tap calls the action's OnTap and then
// OnDismiss. A sheet is a menu: UIKit's action sheet and Material's bottom
// sheet menu both close on selection, and a caller would otherwise write
// open.Set(false) into every handler. An action whose follow-up needs another
// question ("Delete?") opens a Dialog from its OnTap; both are Modals, and the
// same render pass closes one and opens the other.
//
// Cancel calls OnDismiss, like a scrim tap. There is no Cancel.OnTap to fall
// back from, because a cancel that does something other than dismiss is an
// action and belongs in Actions.
//
// # Theme roles read
//
//	Panel          Components.Card, through comps.Card
//	Title          Typography.Subtitle, bold (Card's title treatment)
//	Action ink     Variant.OnLight — Colors.PrimaryOnLight, or ErrorOnLight
//	               for a destructive action (ghost Button)
//	Cancel         Colors.PrimaryOnLight, outlined
//	Rule           Colors.Border, through Separator
type ActionSheet struct {
	// Visible is the caller's open/closed state.
	Visible bool

	// Title names the sheet. It is drawn as the card's heading and is the
	// sheet's accessible name. Empty draws no heading.
	Title string

	// Actions are drawn top to bottom, each a full-width ghost button.
	Actions []SheetAction

	// Cancel is the label of the separate dismiss button under the actions.
	// Empty omits it; the scrim and the platform gestures still dismiss.
	Cancel string

	// OnDismiss is called for a scrim tap, for Cancel, and after any action.
	// Nil makes the scrim inert and the sheet close only when the caller
	// flips Visible.
	OnDismiss func()

	// Backdrop overrides the scrim colour. Empty keeps core.Modal's default.
	Backdrop string

	// Style is applied to the card after the widget's own props, so it can
	// cap the width (core.MaxWidth) or replace the accessible name.
	Style []core.StyleProp
}

// SheetAction is one row of an ActionSheet.
type SheetAction struct {
	// Label is the button text.
	Label string

	// OnTap is called before the sheet's OnDismiss.
	OnTap func()

	// Variant tints the label. VariantError marks a destructive action.
	Variant Variant

	// Disabled greys the action and drops its taps. A disabled action does
	// not dismiss the sheet either, because it does not dispatch at all.
	Disabled bool
}

// Render builds Modal > (filler, Card) as drawn in the type doc.
func (s ActionSheet) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	props := []core.ModalProp{core.Visible(s.Visible)}
	if s.OnDismiss != nil {
		props = append(props, core.OnDismiss(s.OnDismiss))
	}
	if s.Backdrop != "" {
		props = append(props, core.Backdrop(s.Backdrop))
	}

	style := make([]core.StyleProp, 0, len(s.Style)+3)
	// Width first so a caller's MaxWidth can narrow it on a wide browser
	// window. Margin(0) because the theme's Card base carries an 8px margin
	// on every side, and 100% plus a horizontal margin overflows the overlay
	// on the web; a sheet meets the screen edges anyway. The label goes
	// before Style so a caller can replace it, the same order Dialog uses.
	style = append(style, core.Width("100%"), core.Margin(0))
	if s.Title != "" {
		style = append(style, core.AccessibilityLabel(s.Title))
	}
	style = append(style, s.Style...)

	card := Card{
		Title:  s.Title,
		Body:   s.actions(t),
		Footer: s.cancel(t),
		Style:  style,
	}
	props = append(props, core.ModalContent(s.filler(), card))
	return core.Modal(props...).Render(ctx)
}

// filler is the growing, invisible first child that pushes the card to the
// bottom edge on the web and Compose, and reports a tap on it as a dismiss.
// See "The placement question" in the type doc.
func (s ActionSheet) filler() core.View {
	items := []core.PropsAndChildren{
		core.FlexGrow(1),
		core.AlignSelf(core.AlignItemsStretch),
		core.AccessibilityHidden(),
	}
	// Guarded like the Modal's own prop: no OnDismiss means an inert scrim,
	// and a registered no-op would make the filler look tappable to a host.
	if s.OnDismiss != nil {
		items = append(items, core.OnClick(s.OnDismiss))
	}
	return core.Box(items...)
}

// actions builds the column of full-width ghost buttons, or nil when there
// are none so Card omits the region.
func (s ActionSheet) actions(t *core.Theme) core.View {
	if len(s.Actions) == 0 {
		return nil
	}
	items := make([]core.PropsAndChildren, 0, len(s.Actions)+2)
	// Padding(0) drops the theme Column's screen inset, which would double
	// the card's own; XS keeps adjacent targets apart without reading as
	// separate groups.
	items = append(items, core.Padding(0), core.Gap(float64(t.Spacing.XS)))
	for _, a := range s.Actions {
		items = append(items, Button{
			Label:     a.Label,
			OnTap:     s.pick(a),
			Variant:   a.Variant,
			Emphasis:  EmphasisGhost,
			FullWidth: true,
			Disabled:  a.Disabled,
		})
	}
	return core.Column(items...)
}

// pick returns the handler for one action: its own OnTap, then the sheet's
// OnDismiss. Taken per action so each closure holds its own SheetAction.
func (s ActionSheet) pick(a SheetAction) func() {
	return func() {
		if a.OnTap != nil {
			a.OnTap()
		}
		if s.OnDismiss != nil {
			s.OnDismiss()
		}
	}
}

// cancel builds the rule and the outlined Cancel button, or nil when Cancel
// is empty. The rule is what makes Cancel read as leaving the list rather
// than as its last entry.
func (s ActionSheet) cancel(t *core.Theme) core.View {
	if s.Cancel == "" {
		return nil
	}
	onTap := s.OnDismiss
	if onTap == nil {
		// Button registers its handler unconditionally; a nil func in the
		// registry panics when a native tap arrives for it.
		onTap = func() {}
	}
	return core.Column(
		core.Padding(0),
		core.Gap(float64(t.Spacing.SM)),
		Separator{},
		Button{
			Label:     s.Cancel,
			OnTap:     onTap,
			Emphasis:  EmphasisOutlined,
			FullWidth: true,
		},
	)
}
