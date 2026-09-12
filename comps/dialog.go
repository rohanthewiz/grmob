package comps

import "github.com/rohanthewiz/grmob/core"

// Dialog is the "Delete this note?" moment: a title, a sentence, and a row of
// at most two buttons, drawn over the screen in a core.Modal.
//
//	comps.Dialog{
//	    Visible:   confirming.Get(),
//	    Title:     "Delete note?",
//	    Message:   "This cannot be undone.",
//	    Confirm:   comps.DialogAction{Label: "Delete", Variant: comps.VariantError, OnTap: del},
//	    Cancel:    comps.DialogAction{Label: "Keep"},
//	    OnDismiss: func() { confirming.Set(false) },
//	}
//
// # Why the widget exists
//
// core.Modal draws the scrim and the sheet and nothing inside them, so every
// caller hand-rolled the card, the title and the button row — and the
// hand-rolls disagreed on the two things a dialog must be consistent about:
// which side the cancel button sits on, and which button looks dangerous.
// Dialog settles both and adds nothing else.
//
// # One struct, three shapes
//
// The shape follows from which actions carry a Label, not from a mode field:
//
//	Confirm  Cancel   shape
//	───────  ──────   ─────────────────────────────────────────────
//	  set     set     confirm — the two-button "are you sure?"
//	  set      —      alert   — one acknowledgement button
//	   —       —      sheet   — title and body only; the Body slot
//	                            carries its own controls, and a backdrop
//	                            tap (OnDismiss) is the way out
//
// Three widgets for these would share every line but the footer, and a caller
// moving from an alert to a confirm would have to change type rather than add
// a field. A Cancel with no Confirm is drawn as a lone button too; it is an
// alert whose one button happens to be the dismissive one.
//
// # Button order is fixed: cancel leading, confirm trailing
//
// Material 3 and Apple's HIG both put the dismissive action on the leading
// side and the affirmative one on the trailing side of a horizontal pair, and
// the web has no convention strong enough to argue with them. There is no
// order knob: an order field is exactly the per-call-site disagreement this
// widget exists to remove. The pair is packed against the trailing edge
// (JustifyEnd), which is where both platforms put it and where a thumb is.
//
// Cancel is always drawn EmphasisGhost so it reads as the way out. Confirm is
// filled and takes its own Variant, so a destructive action says VariantError
// and gets the theme's Error fill with an ink chosen for contrast by Button.
//
// # Controlled, like core.Modal
//
// Dialog holds no state and never closes itself. Visible renders the caller's
// state; every way out reports intent and leaves the decision with the caller:
//
//   - A backdrop tap calls OnDismiss, and so do the platform gestures the
//     hosts route through the same prop: Compose's Dialog reports the back
//     gesture and an outside tap through onDismissRequest, and SwiftUI (which
//     presents a Modal as a sheet, not a centred card) reports a swipe-down.
//     With OnDismiss nil the web scrim is inert, which is how a dialog that
//     must be answered is written; the natives may still hide the sheet, but
//     Visible stays true in Go and the next render shows it again.
//   - Cancel.OnTap, when nil, falls back to OnDismiss. "Keep" and a tap on the
//     scrim almost always mean the same thing, and writing the same closure
//     twice is how the two drift apart.
//   - Confirm.OnTap does not close the dialog. Confirming usually starts work
//     whose outcome decides what shows next (close, or show an error), so the
//     caller's handler sets Visible false when it is ready to.
//
// Because core.Modal hides rather than unmounts, a Body slot's hook state
// survives a close; reset it in OnDismiss if the dialog should forget.
//
// # Accessibility
//
// Both web targets write role="dialog" and aria-modal on the Modal chassis and
// both natives present a platform dialog, so the widget does not add a role
// (core.Role deliberately has no RoleDialog; see core/role.go). What the
// chassis cannot know is the dialog's name, so the card carries
// AccessibilityLabel(Title). The title text is also a section heading, via
// Card, so a reader can jump to it.
//
// # Theme roles read
//
//	Card base       Components.Card, through core.Card
//	Title           Typography.Subtitle, bold (Card's title treatment)
//	Message         Typography.Body
//	Confirm fill    Variant.Color — Colors.Primary, or Error/Success/Warning
//	Cancel ink      Colors.Primary's on-light tone (ghost Button)
//	Button gap      Spacing.SM
type Dialog struct {
	// Visible is the caller's open/closed state. The Modal renders its content
	// on every pass regardless and the host maps this to visibility.
	Visible bool

	// Title names the dialog. It is drawn as the card's heading and doubles as
	// the dialog's accessible name. Leave it empty only when Body names itself.
	Title string

	// Message is the one or two sentences under the title. Ignored when Body
	// is set.
	Message string

	// Body replaces Message with arbitrary content: a checkbox, a text field,
	// a list. It is the escape hatch in the same simple-path-plus-slot idiom
	// as ListRow.Content and Card.Header.
	Body core.View

	// Confirm is the affirmative action, drawn filled on the trailing side. A
	// zero Label omits it.
	Confirm DialogAction

	// Cancel is the dismissive action, drawn ghost on the leading side. A zero
	// Label omits it. A nil OnTap falls back to OnDismiss.
	Cancel DialogAction

	// OnDismiss is called for a backdrop tap and as Cancel's fallback. Nil
	// makes the scrim inert.
	OnDismiss func()

	// Backdrop overrides the scrim colour. Empty keeps core.Modal's default.
	Backdrop string

	// Style is applied to the card, after the widget's own props, so it can
	// override padding, width or background.
	Style []core.StyleProp
}

// DialogAction is one button in a Dialog's footer.
type DialogAction struct {
	// Label is the button text. Empty means the action is absent.
	Label string

	// OnTap is called when the button is pressed.
	OnTap func()

	// Variant colours a Confirm button's fill (VariantError for a destructive
	// action). A Cancel button is always ghost and reads only the variant's
	// on-light ink, so Cancel's Variant is rarely worth setting.
	Variant Variant

	// Disabled greys the button and drops its taps, for a Confirm that waits
	// on a Body field ("type DELETE to confirm") or on work in flight.
	Disabled bool
}

// Render builds Modal > Card(title, body, footer).
//
//	┌ Modal (scrim; role=dialog on the web) ────────────────┐
//	│  ┌ Card  AccessibilityLabel(Title) ────────────────┐  │
//	│  │ Title                                (heading)  │  │
//	│  │ Message  — or —  Body                           │  │
//	│  │                        ┌ Row JustifyEnd ──────┐ │  │
//	│  │                        │ [Cancel]  [Confirm]  │ │  │
//	│  │                        └──────────────────────┘ │  │
//	│  └─────────────────────────────────────────────────┘  │
//	└───────────────────────────────────────────────────────┘
func (d Dialog) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	props := []core.ModalProp{core.Visible(d.Visible)}
	// Registered only when set: a Modal with no onDismiss prop is one the host
	// will not close on a scrim tap, which is the "must answer" dialog.
	if d.OnDismiss != nil {
		props = append(props, core.OnDismiss(d.OnDismiss))
	}
	if d.Backdrop != "" {
		props = append(props, core.Backdrop(d.Backdrop))
	}

	// The label goes first so a caller's Style can still replace it — the
	// same "widget default, then caller" order every comps widget follows.
	style := make([]core.StyleProp, 0, len(d.Style)+1)
	if d.Title != "" {
		style = append(style, core.AccessibilityLabel(d.Title))
	}
	style = append(style, d.Style...)

	card := Card{
		Title:  d.Title,
		Body:   d.body(t),
		Footer: d.footer(t),
		Style:  style,
	}
	props = append(props, core.ModalContent(card))
	return core.Modal(props...).Render(ctx)
}

// body returns the Body slot, else the Message as body text, else nil so Card
// omits the region rather than drawing an empty Text.
func (d Dialog) body(t *core.Theme) core.View {
	if d.Body != nil {
		return d.Body
	}
	if d.Message == "" {
		return nil
	}
	return core.Text(d.Message, core.UseStyle(t.Typography.Body))
}

// footer builds the button row, or returns nil when neither action has a
// label — the sheet shape, where Card then omits the footer region entirely.
func (d Dialog) footer(t *core.Theme) core.View {
	hasCancel, hasConfirm := d.Cancel.Label != "", d.Confirm.Label != ""
	if !hasCancel && !hasConfirm {
		return nil
	}

	items := []core.PropsAndChildren{
		core.Gap(float64(t.Spacing.SM)),
		core.Justify(core.JustifyEnd),
		core.AlignItemsProp(core.AlignItemsCenter),
		// The theme's Row base carries padding for free-standing rows; inside
		// the card it would double the card's own inset around the buttons.
		core.Padding(0),
	}

	// Cancel first: leading position is the platform convention for the
	// dismissive action (see the type doc), and render order is visual order.
	if hasCancel {
		onTap := d.Cancel.OnTap
		if onTap == nil {
			onTap = d.OnDismiss
		}
		items = append(items, Button{
			Label:    d.Cancel.Label,
			OnTap:    onTap,
			Variant:  d.Cancel.Variant,
			Emphasis: EmphasisGhost,
			Disabled: d.Cancel.Disabled,
		})
	}
	if hasConfirm {
		items = append(items, Button{
			Label:    d.Confirm.Label,
			OnTap:    d.Confirm.OnTap,
			Variant:  d.Confirm.Variant,
			Disabled: d.Confirm.Disabled,
		})
	}
	return core.Row(items...)
}
