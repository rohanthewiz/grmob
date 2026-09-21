package comps

import (
	"sort"
	"strconv"

	"github.com/rohanthewiz/grmob/core"
)

// ConcernPollInert is raised, in debug builds only, when a Poll is still
// asking — no option is Mine and ShowResults is off — and has no OnVote. It
// draws buttons that record nothing.
const ConcernPollInert = "poll-inert"

// PollOption is one answer and its tally.
type PollOption struct {
	// Label is the answer as drawn and spoken.
	Label string

	// Votes is how many people chose it, the reader included.
	Votes int

	// Mine marks the reader's own choice. Any option with Mine set turns the
	// poll from asking to showing results. See Poll, "Who holds the vote".
	Mine bool
}

// Poll is a question whose options turn into labelled result bars once the
// reader has voted: the poll in a chat, the quick survey at the end of an
// article.
//
//	comps.Poll{
//	    Question: "Tabs or spaces?",
//	    Options: []comps.PollOption{
//	        {Label: "Tabs", Votes: 5, Mine: true},
//	        {Label: "Spaces", Votes: 3},
//	    },
//	    OnVote: func(i int) { castVote(pollID, i) },
//	}
//
//	asking                               showing results
//	┌ Column  role=group  label=Q ───┐   ┌ Column  role=group  label=Q ───┐
//	│ Tabs or spaces?                │   │ Tabs or spaces?                │
//	│ ┌────────────────────────────┐ │   │ Tabs ✓                    63%  │
//	│ │           Tabs             │ │   │ ████████████░░░░░░░           │
//	│ └────────────────────────────┘ │   │ Spaces                    37%  │
//	│ ┌────────────────────────────┐ │   │ ███████░░░░░░░░░░░░           │
//	│ │          Spaces            │ │   │ 8 votes                        │
//	│ └────────────────────────────┘ │   └────────────────────────────────┘
//	└────────────────────────────────┘
//
// # Who holds the vote
//
// The caller does, for ReactionBar's reason: a poll's counts are server
// state that other people change, so the widget holds none of it. OnVote
// reports the tapped index, and the poll keeps asking until the caller's data
// comes back with an option marked Mine.
//
// The plan sketched the choice as `Voted int`, an index with -1 for "not
// yet". It is PollOption.Mine instead, because Go's zero value for that int
// is 0, and a Poll written without the field would have opened already voted
// for its first option, silently, with its buttons gone. A bool per option
// has the right zero, and it is the shape ReactionBar's Reaction.Mine already
// has, so the two chat widgets read the same way.
//
// # Percentages that sum to 100
//
// Three options with a vote each round to 33 + 33 + 33, and a poll whose
// results add up to 99% looks broken to exactly the reader who checks. The
// shares are apportioned by the largest-remainder method (pollPercents), so
// they always total 100 when there is at least one vote. With none, every
// bar is empty and every share reads 0%: there is nothing to apportion, and
// inventing an even split would report votes nobody cast.
//
// The bars are drawn from the true fractions, not from the rounded shares, so
// a bar's length never jumps by a rounding step.
//
// # Asking: outlined buttons, not ghost
//
// The plan sketched ghost buttons. A ghost button is a bare label, and a
// column of bare labels under a question reads as a list of text, not as
// things to tap — Stepper's doc records the same finding for its − and +. So
// the options are full-width EmphasisOutlined buttons.
//
// # ShowResults
//
// ShowResults draws the bars without a vote from the reader: a closed poll,
// or an author looking at their own. There is then nothing to tap, so a nil
// OnVote is not a concern in that state.
//
// # Accessibility
//
// The poll is a RoleGroup named by Question, and the question is drawn as a
// RoleHeading inside it. While asking, each option is a plain button. In the
// results each option is one stop named in full — "Tabs, 63 percent, 5
// votes, your choice" — with its text, its bar and its check hidden beneath
// it, so a reader hears each result whole, as MessageBubble does for a
// message. The bar is left unnamed for that reason: a named ProgressBar
// would announce the same number a second time.
//
// # Theme roles read
//
//	Question      Typography.Subtitle, in Colors.TextPrimary
//	Option text   Typography.Body; the reader's choice bold, in
//	              Colors.PrimaryOnLightColor()
//	Share, total  Typography.Caption, Colors.TextSecondary
//	Bars          comps.ProgressBar: Colors.Primary on Colors.Surface
//	Buttons       comps.Button outlined
type Poll struct {
	// Question is drawn above the options and names the group.
	Question string

	// Options are drawn in order. Order is the caller's and does not change
	// with the votes: results sorted by share would move the reader's own
	// choice away from where they tapped it.
	Options []PollOption

	// OnVote receives the index of the tapped option. Nil reports
	// ConcernPollInert while the poll is asking.
	OnVote func(index int)

	// ShowResults draws the results even though no option is Mine.
	ShowResults bool

	// Disabled keeps the poll asking and draws its buttons inert: a poll the
	// reader may see and not answer.
	Disabled bool

	// ChoiceLabel is appended to the spoken name of the reader's choice;
	// empty gives "your choice".
	ChoiceLabel string

	// Style is applied to the outer column after the widget's own props.
	Style []core.StyleProp
}

// pollPercents apportions 100 across votes by the largest-remainder method,
// so the returned shares sum to exactly 100 whenever any vote was cast.
//
//	votes      1, 1, 1
//	exact      33.33, 33.33, 33.33
//	floors     33, 33, 33          → 99, one point short
//	remainders .33, .33, .33       → the point goes to the largest remainder;
//	                                 a tie goes to the earlier option
//	shares     34, 33, 33
//
// The arithmetic is in integers throughout: floor is v*100/total and the
// remainder is v*100%total, so no share depends on how a float happened to
// round. Negative counts are read as zero, since a tally below nothing is a
// caller's bug and must not push another option's share past 100.
func pollPercents(votes []int) []int {
	shares := make([]int, len(votes))
	total := 0
	for _, v := range votes {
		total += max(v, 0)
	}
	if total == 0 {
		return shares
	}

	type rem struct{ index, remainder int }
	rems := make([]rem, len(votes))
	given := 0
	for i, v := range votes {
		v = max(v, 0)
		shares[i] = v * 100 / total
		rems[i] = rem{i, v * 100 % total}
		given += shares[i]
	}
	// Stable, so that equal remainders keep option order and the result is
	// the same on every pass: a percentage that flickered between two
	// options as the sort shuffled them would be worse than 99%.
	sort.SliceStable(rems, func(a, b int) bool { return rems[a].remainder > rems[b].remainder })
	// The floors fall short of 100 by fewer points than there are options,
	// so one pass over the sorted remainders hands out all of them.
	for k := 0; k < 100-given; k++ {
		shares[rems[k].index]++
	}
	return shares
}

// voted reports whether any option is the reader's.
func (p Poll) voted() bool {
	for _, o := range p.Options {
		if o.Mine {
			return true
		}
	}
	return false
}

// Render draws the question over either the buttons or the results. It takes
// no hook slot.
func (p Poll) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()
	results := p.ShowResults || p.voted()

	if core.IsDebugMode() && !results && p.OnVote == nil && !p.Disabled {
		core.ReportConcern(ConcernPollInert,
			"Poll is asking and has no OnVote, so its buttons record nothing; set OnVote, ShowResults or Disabled")
	}

	items := make([]core.PropsAndChildren, 0, len(p.Options)+len(p.Style)+6)
	items = append(items,
		core.Gap(float64(t.Spacing.SM)),
		core.AccessibilityRole(core.RoleGroup),
		core.AccessibilityLabel(p.Question),
	)
	items = append(items, asProps(p.Style)...)
	items = append(items, core.Text(p.Question,
		core.UseStyle(t.Typography.Subtitle),
		// Subtitle carries the secondary ink, which is right under a title
		// and wrong here: the question is the poll's title, and in grey over
		// a column of buttons it read as disabled (the headless render of
		// lesson 4.34).
		core.TextColor(t.Colors.TextPrimary),
		core.AccessibilityRole(core.RoleHeading)))

	if !results {
		for i, o := range p.Options {
			// Keyed by index with a state prefix: the asking and the showing
			// child at one position are different node types, and distinct
			// keys make the switch a clean replace rather than a re-prop of
			// a Button into a Column.
			items = append(items, core.Keyed("ask-"+strconv.Itoa(i), Button{
				Label:     o.Label,
				Emphasis:  EmphasisOutlined,
				FullWidth: true,
				Disabled:  p.Disabled,
				OnTap:     p.voteFor(i),
			}))
		}
		return core.Column(items...).Render(ctx)
	}

	votes := make([]int, len(p.Options))
	total := 0
	for i, o := range p.Options {
		votes[i] = max(o.Votes, 0)
		total += votes[i]
	}
	shares := pollPercents(votes)
	for i, o := range p.Options {
		fraction := 0.0
		if total > 0 {
			fraction = float64(votes[i]) / float64(total)
		}
		items = append(items, core.Keyed("result-"+strconv.Itoa(i), p.result(t, o, shares[i], fraction)))
	}
	items = append(items, core.Text(countNoun(total, "vote"),
		core.UseStyle(t.Typography.Caption),
		core.TextColor(t.Colors.TextSecondary)))
	return core.Column(items...).Render(ctx)
}

// voteFor is option i's tap handler: nil when there is no OnVote, which
// comps.Button turns into a no-op.
func (p Poll) voteFor(i int) func() {
	if p.OnVote == nil {
		return nil
	}
	onVote := p.OnVote
	return func() { onVote(i) }
}

// result is one option after the vote: a label line over its bar, the whole
// thing a single spoken stop.
func (p Poll) result(t *core.Theme, o PollOption, share int, fraction float64) core.View {
	label, ink, weight := o.Label, t.Colors.TextPrimary, core.Normal
	spoken := o.Label + ", " + strconv.Itoa(share) + " percent, " + countNoun(max(o.Votes, 0), "vote")
	if o.Mine {
		// A check as well as the colour: colour alone is not a signal
		// (WCAG 1.4.1), and the on-light tone is the ink-weight Primary, the
		// one measured against a light page.
		label += " ✓"
		ink = t.Colors.PrimaryOnLightColor()
		weight = core.Bold
		spoken += ", " + orDefault(p.ChoiceLabel, "your choice")
	}

	return core.Column(
		core.Gap(float64(t.Spacing.XS)),
		// Assigned over the theme's Column inset. Left on, every result sat
		// indented from the question above it and the total below it, which
		// the same render showed and no tree assertion would.
		core.Padding(0),
		core.AccessibilityRole(core.RoleGroup),
		core.AccessibilityLabel(spoken),
		core.Row(
			// The theme's Row inset is for list rows; this line sits inside a
			// column the caller has already placed.
			core.Padding(0),
			core.Justify(core.JustifyBetween),
			core.AlignItemsProp(core.AlignItemsCenter),
			core.Text(label,
				core.UseStyle(t.Typography.Body),
				core.TextColor(ink),
				core.FontWeight(weight),
				core.AccessibilityHidden()),
			core.Text(strconv.Itoa(share)+"%",
				core.UseStyle(t.Typography.Caption),
				core.TextColor(t.Colors.TextSecondary),
				core.AccessibilityHidden()),
		),
		// Unnamed, so it is hidden from assistive technology: the stop's own
		// name already carries the number.
		ProgressBar{Value: fraction},
	)
}

// countNoun is "1 vote" / "3 votes": a count with its noun in the right
// number.
func countNoun(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return strconv.Itoa(n) + " " + noun + "s"
}
