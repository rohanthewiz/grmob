package comps

import (
	"strconv"

	"github.com/rohanthewiz/grmob/core"
)

// ConcernReactionBarInert is raised, in debug builds only, when a ReactionBar
// that is not Disabled has no OnToggle. Its chips look tappable and do
// nothing, which is the one silent way to get this widget wrong.
const ConcernReactionBarInert = "reaction-bar-inert"

// Reaction is one emoji's tally under a message.
type Reaction struct {
	// Emoji is drawn on the chip and handed back to OnToggle ("👍").
	Emoji string

	// Count is how many people reacted with it, the reader included.
	Count int

	// Mine reports that the reader is one of them. It draws the chip
	// selected.
	Mine bool

	// Label is the emoji's spoken name ("thumbs up"). No host names an emoji
	// reliably — the same glyph is "thumbs up sign", "like" or silence
	// depending on the platform and its voice — and Go carries no table of
	// them, so the caller, who chose the emoji, says what it is called. Empty
	// falls back to the emoji itself, which is whatever the platform makes
	// of it.
	Label string
}

// ReactionBar is the row of emoji chips under a chat message: each chip an
// emoji and its count, the reader's own drawn selected, a tap toggling the
// reader's reaction.
//
//	comps.ReactionBar{
//	    Reactions: []comps.Reaction{
//	        {Emoji: "👍", Count: 3, Mine: true, Label: "thumbs up"},
//	        {Emoji: "🎉", Count: 1, Label: "party popper"},
//	    },
//	    OnToggle: func(emoji string) { toggleReaction(msgID, emoji) },
//	}
//
//	┌ ChipStrip  role=group  label="Reactions" ─┐
//	│  (👍 3)   ( 🎉 1 )                         │
//	│   ▲ Mine: the selected chip               │
//	└───────────────────────────────────────────┘
//
// # The caller holds the counts
//
// The widget is stateless, like Stepper: it draws Reactions and reports a tap
// through OnToggle, and the caller decides what the tap did. A reaction is
// server state — other people change the same count — so a count bumped
// locally on tap would be a second source of truth that disagrees with the
// first the moment the server answers. A caller who wants the optimistic
// bump does it in its own state, where it can also undo it. This is
// AudioPlayer's reasoning pointing the other way: that widget holds its
// position because nobody else can move it, and this one holds nothing
// because everybody else can.
//
// # It is a ChipStrip, and adds only the vocabulary
//
// Each reaction becomes a comps.Chip (Label "👍 3", Selected from Mine) in a
// wrapping comps.ChipStrip, so the look, the 3:1 control ring and the
// selected state's announcement are Chip's and stay in step with every other
// chip in an app. What this type adds is the spoken name, the zero-count
// rule and OnToggle's single callback in place of a closure per chip.
//
// # A count of zero
//
// A reaction nobody holds is not drawn: a caller can pass its whole emoji
// table and see only the ones in use. The exception is Count 0 with Mine
// set. That is a caller's bug (the reader is one of nobody), and it is drawn
// rather than hidden, because a chip reading "👍 0" is a bug somebody will
// see and fix, and a missing chip is one nobody will.
//
// A bar with nothing to draw renders as Display none rather than as an empty
// row, so it takes no gap in the column under its message.
//
// # What it is not
//
//   - **Not an emoji picker.** Adding a reaction that is not yet on the bar
//     needs a grid of emoji in an anchored popover, and the popover is
//     blocked on a renderer (layout measurement). Trailing is the slot for a
//     caller's own "+" chip, which can open a comps.Dialog.
//
// # Accessibility
//
// The strip is a RoleGroup named GroupLabel ("Reactions"). Each chip is a
// button named in full — "thumbs up, 3 reactions" — with the reader's own
// stated as the chip's selected state (core.AccessibilitySelected, from
// Chip), not as words in the name: a name is meant to be stable, and a
// toggle should be announced as a state change. See Chip's
// AccessibilityLabel for the long version.
//
// # Theme roles read
//
// None of its own; see comps.Chip and comps.ChipStrip.
type ReactionBar struct {
	// Reactions are drawn in order. Order is the caller's: most apps keep
	// first-used first, so a chip does not move when its count changes.
	Reactions []Reaction

	// OnToggle receives the tapped chip's Emoji. Whether that adds or removes
	// the reader's reaction is the caller's to decide from its own state.
	// Nil reports ConcernReactionBarInert unless Disabled.
	OnToggle func(emoji string)

	// Trailing is drawn after the last chip: the slot for an "add reaction"
	// chip of the caller's own. It is drawn even when no reaction is, since
	// an empty bar with a "+" is how a first reaction gets added.
	Trailing core.View

	// GroupLabel is the strip's spoken name; empty gives "Reactions".
	GroupLabel string

	// Disabled draws the chips inert: a locked thread, a reader without
	// permission to react.
	Disabled bool

	// Style is applied to the strip after the widget's own props.
	Style []core.StyleProp
}

// spoken is a reaction's accessible name: "thumbs up, 3 reactions". countNoun
// spells the singular, because "1 reactions" is the kind of slip a screen
// reader user hears on every message.
func (r Reaction) spoken() string {
	return orDefault(r.Label, r.Emoji) + ", " + countNoun(r.Count, "reaction")
}

// shown applies the zero-count rule from the type doc.
func (r Reaction) shown() bool { return r.Count > 0 || r.Mine }

// Render builds the ChipStrip. It takes no hook slot, so a bar may be
// rendered conditionally.
func (rb ReactionBar) Render(ctx *core.Context) *core.Node {
	if core.IsDebugMode() && rb.OnToggle == nil && !rb.Disabled {
		core.ReportConcern(ConcernReactionBarInert,
			"ReactionBar has no OnToggle and is not Disabled, so its chips look tappable and do nothing")
	}

	children := make([]core.View, 0, len(rb.Reactions)+1)
	for _, r := range rb.Reactions {
		if !r.shown() {
			continue
		}
		chip := Chip{
			Label:              r.Emoji + " " + strconv.Itoa(r.Count),
			Selected:           r.Mine,
			AccessibilityLabel: r.spoken(),
		}
		if rb.Disabled {
			// core.Disabled is what makes each platform refuse the tap and
			// announce the control as unavailable; OnTap stays nil, which
			// Chip turns into a no-op.
			chip.Style = []core.StyleProp{core.Disabled(true)}
		} else if rb.OnToggle != nil {
			// Captured per iteration: the handler must name this chip's
			// emoji, and the callback itself must be this pass's.
			emoji, toggle := r.Emoji, rb.OnToggle
			chip.OnTap = func() { toggle(emoji) }
		}
		// Keyed by emoji, so a reaction appearing in the middle of the bar
		// inserts one chip instead of re-propping every chip after it.
		children = append(children, core.Keyed("reaction-"+r.Emoji, chip))
	}
	if rb.Trailing != nil {
		children = append(children, rb.Trailing)
	}

	style := make([]core.StyleProp, 0, len(rb.Style)+3)
	style = append(style,
		core.AccessibilityRole(core.RoleGroup),
		core.AccessibilityLabel(orDefault(rb.GroupLabel, "Reactions")),
	)
	if len(children) == 0 {
		// Nothing to draw: no row, and no empty group for a screen reader to
		// stop on.
		style = append(style, core.Display(core.DisplayNone))
	}
	style = append(style, rb.Style...)

	// Children, not Chips: the entries are Keyed wrappers and the caller's
	// Trailing view, neither of which is a Chip value. The slice is non-nil
	// even when empty, which is what makes ChipStrip take this branch.
	return ChipStrip{Children: children, Style: style}.Render(ctx)
}
