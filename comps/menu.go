package comps

import "github.com/rohanthewiz/grmob/core"

// Menu is the overflow "⋯" and the "Sort by" picker: a button that opens a
// short list, where the tap that picks is the tap that closes it.
//
//	comps.Menu{
//	    Trigger:   comps.Button{Label: "⋯", AccessibilityLabel: "Note actions",
//	                            Emphasis: comps.EmphasisGhost},
//	    Open:      open.Get(),
//	    OnOpen:    func() { open.Set(true) },
//	    OnDismiss: func() { open.Set(false) },
//	    Title:     "Groceries",
//	    Items: []comps.SheetAction{
//	        {Label: "Rename", OnTap: rename},
//	        {Label: "Delete", Variant: comps.VariantError, OnTap: del},
//	    },
//	    Cancel: "Cancel",
//	}
//
//	┌ Row (no padding, no gap) ───────────────────────────┐
//	│ [ ⋯ ]  Trigger, OnTap = OnOpen                      │
//	│ ActionSheet{Visible: Open, Actions: Items, …}       │  ← a Modal: drawn
//	└─────────────────────────────────────────────────────┘    over the screen,
//	                                                          no room taken here
//
// # It is an ActionSheet with a trigger, and nothing more
//
// A dropdown anchored under its button needs a popover positioned against the
// trigger's frame, and no host sends a frame to Go; that is a node type, and
// the plan that proposed this widget listed it as out of reach. Every target
// can already present a Modal, and ActionSheet has settled where a Modal's
// list goes on each of them, so the menu is that sheet. On a phone that is the
// platform's own shape for a menu of actions; in a browser window it is the
// same panel on the bottom edge, which is the one place it reads as a
// compromise.
//
// So the items are SheetActions, picking one runs its OnTap and then
// OnDismiss, and Cancel, the scrim and a tap above the panel all dismiss. The
// ActionSheet type doc is where each of those is argued.
//
// # The trigger is a Button template, not a View slot
//
// A core.View handed in by the caller cannot be given a tap handler: a widget
// cannot add props to a View it did not build. So the trigger is a
// comps.Button whose Label, Variant, Emphasis, Disabled, Style and names all
// apply and whose OnTap is overwritten with OnOpen, the same template move
// DatePicker makes with its Calendar. A trigger that must be something other
// than a button (a whole row) is an ActionSheet next to that row.
//
// # Open is the caller's, which is what makes a menu per row cost one state
//
// DatePicker owns its open flag in a hook, and a Menu could too. It does not,
// because the commonest menu is the "⋯" on every row of a list, and a
// hook-holding widget in a loop takes a positional slot per row: the slots
// drift as soon as a filter or a delete changes the row count. Controlled,
// one state holds which row's menu is open, and every row compares against
// it:
//
//	open := core.NewState(ctx, "")
//	for _, n := range notes {
//	    comps.Menu{
//	        Open:      open.Get() == n.ID,
//	        OnOpen:    func() { open.Set(n.ID) },
//	        OnDismiss: func() { open.Set("") },
//	        …
//	    }
//	}
//
// It also leaves Menu conditional-safe, like every widget that takes no hook.
//
// # A picker is a menu whose items are checked
//
// "Sort by: Newest" is the same widget: each item sets one value, and the
// current one is SheetAction.Checked. There is no Value/OnChange pair, since
// the items already carry their handlers and a second path to the same
// setter is a second thing to keep in step. RadioGroup is the form for a
// choice that should be compared on screen rather than behind a tap.
//
// # What the trigger announces
//
// Its own label and hint, and no expanded state. core.Style's disclosure
// table names this exact near miss: a control that opens a dialog is not
// expanded, ARIA spells the relationship aria-haspopup, and core does not
// carry it. Once the sheet is open the Modal's dialog role is what a reader
// is inside. An icon trigger such as "⋯" needs Trigger.AccessibilityLabel,
// and a picker's trigger reads best with the current value in its label.
//
// # Theme roles read
//
//	Trigger    as comps.Button, from the template's Variant and Emphasis
//	Sheet      as ActionSheet
type Menu struct {
	// Trigger is the button that opens the menu. Its OnTap is ignored and
	// replaced by OnOpen; see the type doc.
	Trigger Button

	// Open is the caller's open/closed state.
	Open bool

	// OnOpen is called when the trigger is tapped. Nil leaves a trigger that
	// does nothing, which is Trigger.Disabled's job done badly; set that
	// instead.
	OnOpen func()

	// OnDismiss is called for a scrim tap, for Cancel, and after any item,
	// exactly as ActionSheet.OnDismiss.
	OnDismiss func()

	// Title names the sheet: its heading and its accessible name. Empty
	// draws no heading, which leaves the sheet unnamed; set it.
	Title string

	// Items are the menu's entries, top to bottom.
	Items []SheetAction

	// Cancel labels the dismiss button under the items. Empty omits it.
	Cancel string

	// Backdrop overrides the scrim colour, as ActionSheet.Backdrop.
	Backdrop string

	// Style is applied to the sheet's card, as ActionSheet.Style. The
	// trigger is styled through Trigger.Style.
	Style []core.StyleProp
}

// Render builds Row(trigger, ActionSheet) as drawn in the type doc.
func (m Menu) Render(ctx *core.Context) *core.Node {
	trigger := m.Trigger
	// Button already swaps a nil OnTap for a no-op before registering it, so
	// a nil OnOpen needs no guard here.
	trigger.OnTap = m.OnOpen

	return core.Row(
		// The wrapper exists only because Render returns one node. No padding
		// and no gap, so the Row measures exactly the trigger: a theme Row's
		// screen inset would pad an AppBar's trailing slot, and a gap would
		// open beside the Modal child, which draws nothing in this row on
		// any target.
		core.Padding(0),
		core.Gap(0),
		core.AlignItemsProp(core.AlignItemsCenter),
		trigger,
		ActionSheet{
			Visible:   m.Open,
			Title:     m.Title,
			Actions:   m.Items,
			Cancel:    m.Cancel,
			OnDismiss: m.OnDismiss,
			Backdrop:  m.Backdrop,
			Style:     m.Style,
		},
	).Render(ctx)
}
