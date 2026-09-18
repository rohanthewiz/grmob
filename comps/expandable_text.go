package comps

import (
	"unicode/utf8"

	"github.com/rohanthewiz/grmob/core"
)

// ExpandableText is body text capped at a few lines, with a "Read more" that
// opens it in place and a "Read less" that closes it again: a product
// description, a review, the summary of an episode.
//
//	comps.ExpandableText{Text: episode.Summary, Lines: 3}
//
//	┌ Column ────────────────────────────────────────────┐
//	│ The third episode follows the team to the coast,   │  core.MaxLines(3)
//	│ where the survey that was meant to take a week     │  while collapsed
//	│ turns into a month of weather, tides and a …       │
//	│ [ Read more ]                                      │  ghost, aria-expanded
//	└────────────────────────────────────────────────────┘
//
// # When the toggle shows: a threshold, because nothing measures
//
// The honest rule is "show Read more when the cap actually cut something",
// and no host can say whether it did: that is a rendered height, the layout
// measurement Tooltip is blocked on. A toggle that is always there shows
// "Read more" under a two-line paragraph, which opens onto nothing.
//
// So the rule is a length the caller can tune. The toggle shows when the
// text has more than ToggleAfter characters (runes) — by default Lines × 40,
// about what fits a phone's line of body text. A caller who knows better
// states it: a negative ToggleAfter always shows the toggle, a very large one
// never does. Where the estimate is wrong the failure is mild in both
// directions: a toggle that reveals a few more words, or a paragraph a line
// longer than the cap that is simply shown in full — collapsed with no
// toggle means no cap, since a cap with no way to lift it would hide the end
// of the text for good.
//
// # The open state is the widget's
//
// Whether the text is open is held here, in a hook — no application wants it,
// PasswordField's test — so render an ExpandableText unconditionally, in a
// stable position.
//
// # Accessibility
//
// The text node carries the whole string on every target; the cap is visual
// only (a screen reader reads a clamped Text in full on the web, and VoiceOver
// and TalkBack read a lineLimit / maxLines Text's full content). The toggle
// is a button whose name stays "Read more" and whose expanded state is
// stated with core.AccessibilityExpanded — aria-expanded — so the name does
// not flip between two different-sounding buttons. See PasswordField's
// "Accessibility" for the same rule.
//
// # Theme roles read
//
//	Text     Typography.Body
//	Toggle   a ghost Button (Primary's on-light tone)
//	Gap      Spacing.XS
type ExpandableText struct {
	// Text is the full text.
	Text string

	// Lines is the collapsed cap; 0 means 3.
	Lines int

	// ToggleAfter is the length in runes past which the text is capped and
	// the toggle shown; 0 means Lines × 40, negative always shows it. See
	// "When the toggle shows".
	ToggleAfter int

	// MoreLabel and LessLabel caption the toggle; empty gives "Read more"
	// and "Read less". MoreLabel is also its accessible name in both states.
	MoreLabel, LessLabel string

	// Style is applied to the text after its defaults.
	Style []core.StyleProp
}

// Render draws the text and, when it is long enough, the toggle. It takes one
// hook, the open state.
func (e ExpandableText) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	// Before any branch: the hook rule PasswordField's doc spells out.
	open := core.NewState(ctx, false)

	lines := e.Lines
	if lines <= 0 {
		lines = 3
	}
	after := e.ToggleAfter
	if after == 0 {
		after = lines * 40
	}
	long := after < 0 || utf8.RuneCountInString(e.Text) > after

	text := make([]core.StyleProp, 0, len(e.Style)+2)
	text = append(text, core.UseStyle(t.Typography.Body))
	if long && !open.Get() {
		text = append(text, core.MaxLines(lines))
	}
	text = append(text, e.Style...)

	items := []core.PropsAndChildren{
		core.Padding(0),
		core.Gap(float64(t.Spacing.XS)),
		core.Text(e.Text, text...),
	}
	if long {
		more := orDefault(e.MoreLabel, "Read more")
		caption := more
		if open.Get() {
			caption = orDefault(e.LessLabel, "Read less")
		}
		items = append(items, Button{
			Label:              caption,
			OnTap:              func() { open.Set(!open.Get()) },
			Emphasis:           EmphasisGhost,
			AccessibilityLabel: more,
			Style: []core.StyleProp{
				core.AccessibilityExpanded(core.ExpandedWhen(open.Get())),
				// Under the text's first letter, not centred under the
				// paragraph: the toggle reads as the paragraph's last word.
				core.AlignSelf(core.AlignItemsStart),
				// A ghost button's own horizontal padding would indent the
				// caption past the text's edge.
				core.PaddingHorizontal(0),
			},
		})
	}
	return core.Column(items...).Render(ctx)
}
