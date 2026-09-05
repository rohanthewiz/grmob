package components

import "github.com/rohanthewiz/grmob/core"

// EmptyState is the centered placeholder that stands in for content a screen
// does not have: a list with nothing in it, a fetch still in flight, a fetch
// that failed.
//
//	components.EmptyState{Glyph: "🔍", Title: "No sermons match “grace”",
//	    Hint: "Try a different word, or clear the filters."}
//
// # One widget for empty, busy and failed
//
// Those three look like three states but they are one shape — a mark, a line
// saying what is going on, a quieter line saying what to do about it, and
// sometimes a way out — and screens that build them separately end up wording
// and spacing them differently:
//
//	empty   EmptyState{Glyph: "📭", Title: "No messages yet"}
//	busy    EmptyState{Title: "Loading sermons…"}
//	failed  EmptyState{Glyph: "☁", Title: err.Error(),
//	            ActionLabel: "Retry", OnAction: reload}
//
// The busy case is a line of text rather than a spinner on purpose: core has
// no indeterminate progress node, and ProgressBar is determinate (it takes a
// 0..1 value), so there is nothing here to animate a wait with. Naming what is
// loading is more useful than an animation in any case — "Loading sermons…"
// tells the user which of the screen's three sections is slow.
//
// # Width is load-bearing
//
// The column sets Width 100%, which looks redundant and is not. On both
// natives a column hugs its widest child (Compose wrap-content; grmob's
// SwiftUI layout does the same), so without it the whole block sits at the
// leading edge with its children centered inside a box only as wide as the
// longest line — centered text that is not centered on the screen. The two
// DOM targets fill the line already, as any block box does, so the bug is
// invisible on the target you are most likely to be looking at.
type EmptyState struct {
	// Glyph is the large mark above the text. Empty draws none, which is the
	// right call for the busy case where a mark would look like a state the
	// user is meant to read.
	//
	// It is decoration and is hidden from assistive technology: a reader
	// announcing "open mailbox with lowered flag" ahead of "No messages yet"
	// is noise. Title has to carry the meaning.
	Glyph string

	// Title is the primary line — what is going on. Hint is the quieter line
	// under it — what to do about it.
	Title string
	Hint  string

	// ActionLabel and OnAction render the way out: "Retry", "Clear filters",
	// "Invite someone". Action is the slot form and takes precedence, for a
	// pair of buttons or anything else.
	//
	// The built button is outlined, not filled. An empty state is a dead end,
	// not a call to action — a solid Primary button in the middle of an empty
	// screen is the loudest thing on it, and the screen has nothing to say.
	ActionLabel string
	OnAction    func()
	Action      core.View

	// Style is applied after the widget's own defaults, so the padding, the
	// centering and the full width are all overridable.
	Style []core.StyleProp
}

func (e EmptyState) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	items := make([]core.PropsAndChildren, 0, len(e.Style)+8)
	items = append(items,
		core.Width("100%"), // see the type comment — this is not redundant
		core.AlignItemsProp(core.AlignItemsCenter),
		core.Gap(float64(t.Spacing.SM)),
		core.Padding(t.Spacing.LG),
	)
	for _, sp := range e.Style {
		items = append(items, sp)
	}

	if e.Glyph != "" {
		items = append(items, core.Text(e.Glyph,
			// Roughly twice a title: big enough to read as an illustration
			// rather than as a character in the sentence below it.
			core.FontSize(t.Typography.Title.FontSize*1.2),
			core.TextColor(t.Colors.TextSecondary),
			core.AccessibilityHidden(),
		))
	}
	if e.Title != "" {
		items = append(items, core.Text(e.Title,
			core.UseStyle(t.Typography.Body),
			core.TextColor(t.Colors.TextPrimary),
			// Align, not AlignItems: this centers the *lines* of a title that
			// wraps, where the column's AlignItems only centers the box.
			core.Align(core.AlignCenter),
		))
	}
	if e.Hint != "" {
		items = append(items, core.Text(e.Hint,
			core.UseStyle(t.Typography.Caption),
			core.Align(core.AlignCenter),
		))
	}

	switch {
	case e.Action != nil:
		items = append(items, e.Action)
	case e.ActionLabel != "" && e.OnAction != nil:
		items = append(items, Button{
			Label:    e.ActionLabel,
			Emphasis: EmphasisOutlined,
			OnTap:    e.OnAction,
		})
	}

	// Column, not Box: the theme's Column base is the screen-level padding
	// this block wants to sit inside, and Padding above assigns over it
	// anyway. What matters is that a themed container carries the theme's
	// idea of a content block, which a placeholder standing in for content
	// should match.
	return core.Column(items...).Render(ctx)
}
