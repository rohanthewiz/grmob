package comps

import (
	"time"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/hooks"
)

// Snackbar is the "Note deleted · Undo" strip: one line of text and at most
// one action, shown for a few seconds and then gone.
//
//	comps.Snackbar{
//	    Visible:   undo.Get() != nil,
//	    Message:   "Note deleted",
//	    Action:    "Undo",
//	    OnAction:  restore,
//	    OnTimeout: func() { undo.Set(nil) },
//	}
//
// # Why a widget and not core.ShowToast
//
// core.ShowToast is fire-and-forget: Go hands the host a string and the host
// draws and removes it. What it cannot carry is a button, and an Undo is the
// whole reason a snackbar exists. Giving the toast an action callback would be
// a change on all four renderers. A widget the caller renders needs none, so
// this is the widget and the toast extension stays the eventual shape.
//
// # Controlled, and it holds one hook
//
// The caller owns visibility, as with Dialog. Snackbar never hides itself; it
// reports two things and leaves the decision with the caller:
//
//   - OnTimeout, Duration after Visible turns true. The timer is
//     hooks.UseTimeoutWhile keyed on Message, so a replaced message gets its
//     full Duration, and hiding the snackbar cancels a pending timeout.
//   - OnAction, when the action is tapped. The caller usually undoes the work
//     and hides the snackbar in the same handler.
//
// Because of the hook, the rules Spinner documents apply: render a Snackbar
// in a stable position on every pass and drive Visible, rather than leaving
// it out of the tree. A hidden snackbar is Display none and its timer is
// cancelled, so it costs nothing.
//
// # Where it goes
//
// It is a strip, not an overlay, so the caller places it. Two placements
// work on every target:
//
//	Screen.Footer   core.Column(snackbar, bottomBar): pinned above the bar,
//	                and a hidden snackbar takes no space
//	core.ZStack     a layer with core.StackAlign(core.StackAlignBottom) over
//	                content that fills the stack
//
// The Footer form never covers content: the scroll region shrinks while the
// strip is up. That is usually what a list with an Undo wants, since the row
// that was just removed is the one a user looks for.
//
// # Accessibility
//
// The strip is RoleStatus, a polite live region, so the message is read at
// the next pause and does not interrupt. VariantError makes it RoleAlert,
// which does interrupt; a failure the user must hear before continuing is
// what that role is for. No accessible name is set, because a live region
// announces its content and a label would replace the message.
//
// # Theme roles read
//
//	Strip       Colors.TextPrimary as the fill, Colors.Background as the ink:
//	            the inverse of the page, which is what makes it read as a
//	            transient layer rather than as content
//	Variant     Variant.Color as the fill and Variant.Ink for contrast, when
//	            a Variant is set
//	Message     Typography.Body
//	Spacing     Spacing.SM gap and vertical padding, Spacing.MD horizontal
type Snackbar struct {
	// Visible is the caller's shown/hidden state. Rising to true starts the
	// timeout; falling to false cancels it.
	Visible bool

	// Message is the one line of text. A new Message while visible restarts
	// the timeout.
	Message string

	// Action is the button label. Empty, or a nil OnAction, draws no button.
	Action string

	// OnAction is called when the action is tapped. It does not hide the
	// snackbar; set Visible false in it.
	OnAction func()

	// OnTimeout is called once, Duration after Visible turns true. Nil, like
	// a negative Duration, means the snackbar stays until the caller hides it.
	OnTimeout func()

	// Duration is how long the snackbar stays up. Zero uses
	// SnackbarDuration; a negative value never times out.
	Duration time.Duration

	// Variant tints the strip. VariantError also makes it an alert.
	Variant Variant

	// Style is applied to the strip after the widget's own props.
	Style []core.StyleProp
}

// SnackbarDuration is the default time a Snackbar stays up: long enough to
// read a short sentence and reach an Undo, and within Material's four-to-ten
// second range for a snackbar with an action.
const SnackbarDuration = 4 * time.Second

// Render builds Row(message, action) and holds the timeout hook.
func (s Snackbar) Render(ctx *core.Context) *core.Node {
	d := s.Duration
	if d == 0 {
		d = SnackbarDuration
	}
	// Unconditional, before anything reads Visible: the slot must exist on
	// every pass. The closure is refreshed each render, so the fire calls the
	// OnTimeout of the latest render.
	hooks.UseTimeoutWhile(ctx, s.Visible && d > 0 && s.OnTimeout != nil, func() {
		if s.OnTimeout != nil {
			s.OnTimeout()
		}
	}, d, s.Message)

	t := ctx.Theme()
	fill, ink := t.Colors.TextPrimary, t.Colors.Background
	if s.Variant != VariantDefault {
		fill = s.Variant.Color(t)
		ink = s.Variant.Ink(t, fill)
	}
	role := core.RoleStatus
	if s.Variant == VariantError {
		role = core.RoleAlert
	}

	items := make([]core.PropsAndChildren, 0, len(s.Style)+10)
	items = append(items,
		core.Gap(float64(t.Spacing.SM)),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.PaddingVertical(t.Spacing.SM),
		core.PaddingHorizontal(t.Spacing.MD),
		core.BorderRadius(8),
		core.BackgroundColor(fill),
		core.AccessibilityRole(role),
	)
	if !s.Visible {
		items = append(items, core.Display(core.DisplayNone))
	}
	for _, sp := range s.Style {
		items = append(items, sp)
	}

	// FlexGrow on the message, not JustifyBetween on the row: the ListRow
	// argument, so the action stays on the trailing edge with or without a
	// long message.
	items = append(items, core.Text(s.Message,
		core.UseStyle(t.Typography.Body),
		core.TextColor(ink),
		core.FlexGrow(1),
	))
	if s.Action != "" && s.OnAction != nil {
		items = append(items, Button{
			Label:    s.Action,
			OnTap:    s.OnAction,
			Emphasis: EmphasisGhost,
			// A ghost button spends the on-light ink, which disappears on the
			// inverse fill; the strip's own ink replaces it.
			Style: []core.StyleProp{core.TextColor(ink), core.FontWeight(core.Bold)},
		})
	}
	return core.Row(items...).Render(ctx)
}
