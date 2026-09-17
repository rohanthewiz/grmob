package comps

import (
	"fmt"

	"github.com/rohanthewiz/grmob/core"
)

// ConcernSelectRowValueNotAnOption is raised, in debug builds only, when Value
// is set to something no option in Options carries. The row has nothing to put
// in its trailing slot then and shows Placeholder, which looks exactly like an
// unset field — so the mismatch would otherwise be invisible until somebody
// noticed a setting that never displays its own value.
const ConcernSelectRowValueNotAnOption = "select-row-value-not-an-option"

// SelectRow is the settings-screen row for a value chosen from a short list:
// the title on the leading edge, the current choice on the trailing edge, and
// a sheet of the alternatives behind a tap on the row.
//
//	┌──────────────────────────────────┐        ┌─────────────────────┐
//	│ Theme                    Dark ›  │  tap → │ Theme               │
//	└──────────────────────────────────┘        │   System            │
//	                                            │   Light             │
//	                                            │ ✓ Dark              │
//	                                            │   Cancel            │
//	                                            └─────────────────────┘
//
//	comps.SelectRow{
//	    Title: "Theme",
//	    Options: []core.SelectOption{
//	        core.Option("system", "System"),
//	        core.Option("light", "Light"),
//	        core.Option("dark", "Dark"),
//	    },
//	    Value:    theme.Get(),
//	    OnChange: theme.Set,
//	}
//
// It completes the settings-row family: SwitchRow for a boolean that acts on
// the tap, CheckboxRow for one a form collects, SliderRow for a number, and
// this for one of a few named values.
//
// # Why a sheet and not core.Select in the trailing slot
//
// core.Select is the platform's own picker and is the right control inside a
// form, where it sits in a FormField beside text inputs. It is the wrong one
// *in a row*, for two reasons that are both about the row rather than about
// the picker:
//
//   - The row is the target. A settings list is scanned and tapped anywhere
//     along its width; a picker in the trailing slot is a control the width of
//     its longest label, and the rest of the row does nothing.
//   - A row that both opened a sheet and held a native picker would open two
//     things on the web, where a click on the control bubbles to the row — the
//     same double dispatch SwitchRow's doc works through. There the two
//     handlers converge on one value and the guard drops the second; two
//     *openings* have nothing to converge on.
//
// So the row owns the whole gesture and the choices are drawn as an
// ActionSheet, which is the shape both phones use for this (iOS's action sheet,
// Material's list dialog) and the one comps already has.
//
// # The row owns one piece of state
//
// Whether the sheet is open, and nothing else — the same single state
// DatePicker owns, for the same reason: no application wants to hold it, and
// every one of them would hold it identically. The consequence is the hook
// rule, in full: render a SelectRow unconditionally, in a stable position,
// every pass. A list of settings rows built by a loop is fine; a row that
// appears only when some other switch is on is not, and wants core.When around
// a whole screen section rather than around this widget.
//
// # Options are core.SelectOption, with one of its fields undrawn
//
// The list is the type core.Select and SearchableSelect take, so it moves
// between the three unchanged. Label defaults to Value at the same seam
// core.Select defaults it, and Disabled or GroupDisabled greys an action and
// drops its taps — read per option, as SearchableSelect reads them, rather
// than propagated along a run.
//
// Group is *not* drawn. A sheet action is a button with a label and has no
// second line to put a heading on and no section construct to open, and
// inventing one here would be a third answer to a question core.Select and
// SearchableSelect have each already answered (an <optgroup>, and the row's
// subtitle). A grouped list is a list long enough to want one of those two.
//
// # Accessibility
//
// The row takes core.RoleButton and core.PopupDialog, as DatePicker's trigger
// does: a Row is scenery on every target until a role says otherwise, and the
// popup declaration needs a role ARIA defines it on. Its name comes from its
// own text — "Theme, Dark" — which is why no label is synthesized here, the
// rule ListRow states for every row.
//
// The chosen action states core.CurrentTrue through SheetAction.Checked rather
// than a selected state; see that field for why a run of buttons cannot be a
// radio group.
//
// # Theme roles read
//
// Everything ListRow reads, plus Colors.TextSecondary for the trailing summary
// and everything ActionSheet reads for the sheet.
type SelectRow struct {
	// Title is the setting's name, drawn as the row's primary line.
	Title string

	// Subtitle is the quieter second line under the title.
	Subtitle string

	// Leading is an optional icon or avatar before the text, as in ListRow.
	Leading core.View

	// Options are the choices, in the order the sheet lists them.
	Options []core.SelectOption

	// Value is the chosen option's Value. A value no option carries shows
	// Placeholder and reports ConcernSelectRowValueNotAnOption in debug
	// builds; empty shows Placeholder quietly, since "nothing chosen yet" is
	// a state a settings row legitimately starts in.
	Value string

	// OnChange receives the picked option's Value and the sheet closes. It is
	// not called for a tap on the option already chosen — a sheet that is
	// dismissed by choosing what was already there has changed nothing, and a
	// setter that writes the value it was given is the one shape that survives
	// both that and a caller who inverts state. Nil leaves a row that opens a
	// sheet nothing can be picked from; use Disabled for one that should not
	// open at all.
	OnChange func(string)

	// Placeholder is the trailing text when Value matches no option. Empty
	// leaves the slot blank but still tappable.
	Placeholder string

	// SheetTitle names the sheet and is its accessible name. Empty uses Title,
	// which is right whenever the row's name is already the question ("Theme")
	// and wrong when it is only half of one ("Sort" → "Sort by").
	SheetTitle string

	// CancelLabel captions the sheet's dismiss button; empty gives "Cancel".
	// There is always one: the scrim dismisses too, but a way out that is not
	// a target is not a way out for everybody.
	CancelLabel string

	// Disabled marks the row inert: it neither opens nor announces itself as
	// actionable, and the sheet's own actions are disabled with it so a tap
	// racing the patch cannot land in an open sheet.
	Disabled bool

	// Style is passed to the underlying ListRow and so beats its defaults.
	Style []core.StyleProp
}

// Render builds the row and its sheet.
func (r SelectRow) Render(ctx *core.Context) *core.Node {
	// One hook, every pass, in a fixed position. See the type comment.
	open := core.NewState(ctx, false)
	closeSheet := func() { open.Set(false) }

	// Box, not Column, and the sheet a sibling of the row rather than a child
	// of it: both for DatePicker's reasons — a Column arrives with the theme's
	// screen inset, and an overlay nested inside a row inherits that row's
	// clip on any target honouring overflow.
	return core.Box(
		r.row(ctx, open),
		r.sheet(ctx, open, closeSheet),
	).Render(ctx)
}

// row is the closed state: the ListRow the family shares, with the current
// choice pinned to the trailing edge.
func (r SelectRow) row(ctx *core.Context, open core.State[bool]) core.View {
	style := make([]core.StyleProp, 0, len(r.Style)+3)
	style = append(style, r.Style...)
	// After the caller's styles, as DatePicker orders them: what the control
	// *is* and whether it is inert are not looks, so a Style override must not
	// quietly remove either.
	style = append(style,
		core.AccessibilityRole(core.RoleButton),
		core.AccessibilityHasPopup(core.PopupDialog),
	)
	if r.Disabled {
		style = append(style, core.Disabled(true))
	}

	return ListRow{
		Leading:  r.Leading,
		Title:    r.Title,
		Subtitle: r.Subtitle,
		Trailing: r.summary(ctx),
		// Registered even when disabled, and a no-op then: core.Disabled is
		// what makes every renderer refuse to dispatch, and a handler still in
		// the registry is what a tap already in flight lands on.
		OnTap: func() {
			if r.Disabled {
				return
			}
			open.Set(true)
		},
		Style: style,
	}
}

// summary is the trailing slot: the current choice and a chevron.
func (r SelectRow) summary(ctx *core.Context) core.View {
	t := ctx.Theme()

	text := r.selectedLabel()
	if text == "" {
		text = r.Placeholder
	}

	// Padding(0): the theme's Row base insets a row by 16pt on each side, and
	// this row is *inside* one — the inset would push the chevron off the
	// parent's own padding and make the summary's height disagree with the
	// title's.
	return core.Row(
		core.Padding(0),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.Gap(float64(t.Spacing.XS)),
		core.Text(text,
			core.UseStyle(t.Typography.Body),
			core.TextColor(t.Colors.TextSecondary),
		),
		// Decoration: the row already announces itself as a button that opens
		// a dialog, and "greater-than sign" read after the value is noise.
		core.Text("›", core.AccessibilityHidden()),
	)
}

// selectedLabel is the label of the option Value names, or "" when no option
// carries it. The debug concern is raised here rather than in Render because
// this is the one place that has both the lookup and its result.
func (r SelectRow) selectedLabel() string {
	for _, o := range r.Options {
		if o.Value != r.Value {
			continue
		}
		// The same default core.Select applies at its own seam: an empty
		// Label means the value is readable enough.
		if o.Label != "" {
			return o.Label
		}
		return o.Value
	}
	if r.Value != "" {
		core.ReportConcern(ConcernSelectRowValueNotAnOption, fmt.Sprintf(
			"SelectRow %q: Value %q is not among the %d options, so the row shows its placeholder",
			r.Title, r.Value, len(r.Options)))
	}
	return ""
}

// sheet is the ActionSheet the row opens: one action per option, a check on
// the current one, and a way out.
func (r SelectRow) sheet(ctx *core.Context, open core.State[bool], closeSheet func()) core.View {
	title := r.SheetTitle
	if title == "" {
		title = r.Title
	}
	cancel := r.CancelLabel
	if cancel == "" {
		cancel = "Cancel"
	}

	actions := make([]SheetAction, 0, len(r.Options))
	for _, o := range r.Options {
		label := o.Label
		if label == "" {
			label = o.Value
		}
		actions = append(actions, SheetAction{
			Label: label,
			// Read per option rather than propagated along a run, which is
			// what SearchableSelect does with the same pair: a sheet has no
			// section for the run-level flag to grey.
			Disabled: o.Disabled || o.GroupDisabled || r.Disabled,
			Checked:  o.Value == r.Value,
			OnTap:    r.pick(o.Value),
		})
	}

	return ActionSheet{
		Visible:   open.Get(),
		Title:     title,
		Actions:   actions,
		Cancel:    cancel,
		OnDismiss: closeSheet,
	}
}

// pick returns one action's handler. ActionSheet closes itself after any
// action (it calls OnDismiss), so this only has to report — and only when the
// value actually moved, which is the guard SwitchRow's setter applies one
// widget over.
func (r SelectRow) pick(value string) func() {
	return func() {
		if r.OnChange == nil || r.Disabled || value == r.Value {
			return
		}
		r.OnChange(value)
	}
}
