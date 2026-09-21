package comps

import (
	"slices"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// The largest-remainder table. Every row with a vote in it must sum to 100,
// which the loop asserts as well as the exact shares.
func TestPollPercentsSumToAHundred(t *testing.T) {
	cases := []struct {
		name  string
		votes []int
		want  []int
	}{
		{"three thirds: the spare point goes to the earliest", []int{1, 1, 1}, []int{34, 33, 33}},
		{"six sixths: four spare points, in order", []int{1, 1, 1, 1, 1, 1}, []int{17, 17, 17, 17, 16, 16}},
		{"the largest remainder wins, not the largest share", []int{5, 3}, []int{63, 37}}, // 62.5 and 37.5 tie; earlier takes it
		{"a remainder beats an earlier, smaller one", []int{1, 2, 4}, []int{14, 29, 57}},  // .29 .57 .14 → the point goes to the second
		{"exact shares are untouched", []int{1, 3}, []int{25, 75}},
		{"one option takes it all", []int{0, 7, 0}, []int{0, 100, 0}},
		{"no votes: nothing to apportion", []int{0, 0}, []int{0, 0}},
		{"a negative tally reads as zero", []int{-4, 1}, []int{0, 100}},
		{"no options", nil, []int{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := pollPercents(c.votes)
			if !slices.Equal(got, c.want) {
				t.Errorf("pollPercents(%v) = %v, want %v", c.votes, got, c.want)
			}
			sum, any := 0, false
			for i, s := range got {
				sum += s
				any = any || c.votes[i] > 0
			}
			if any && sum != 100 {
				t.Errorf("shares %v sum to %d, want 100", got, sum)
			}
		})
	}
}

var askingOptions = []PollOption{{Label: "Tabs", Votes: 5}, {Label: "Spaces", Votes: 3}}

// Asking: a group named by the question, the question as a heading, and one
// outlined full-width button per option. No result is drawn before a vote.
func TestPollAskingIsAQuestionOverButtons(t *testing.T) {
	_, n := renderDebug(t, Poll{Question: "Tabs or spaces?", Options: askingOptions, OnVote: func(int) {}})

	if n.Style.AccessibilityRole != core.RoleGroup || n.Style.AccessibilityLabel != "Tabs or spaces?" {
		t.Errorf("role %q label %q", n.Style.AccessibilityRole, n.Style.AccessibilityLabel)
	}
	if q := findText(n, "Tabs or spaces?"); q == nil || q.Style.AccessibilityRole != core.RoleHeading {
		t.Error("the question should be drawn as a heading")
	}
	btns := buttonsOf(n)
	if len(btns) != 2 || btns[0].Props["label"] != "Tabs" || btns[1].Props["label"] != "Spaces" {
		t.Fatalf("buttons = %d, want Tabs and Spaces", len(btns))
	}
	if btns[0].Style.BorderWidth != 1 {
		t.Error("the options are outlined buttons: a ghost label does not read as a target")
	}
	if findText(n, "63%") != nil {
		t.Error("no share is drawn before the reader has voted")
	}
}

// A tap reports the option's index, once. The poll does not flip itself to
// results: the vote is the caller's to record.
func TestPollTapReportsTheIndex(t *testing.T) {
	var got []int
	ctx, n := renderDebug(t, Poll{Question: "Q", Options: askingOptions, OnVote: func(i int) { got = append(got, i) }})
	ctx.TriggerCallback(buttonsOf(n)[1].Props["onClick"].(string))
	if !slices.Equal(got, []int{1}) {
		t.Errorf("votes = %v, want exactly [1]", got)
	}
}

// Showing results: no buttons, one spoken stop per option with its parts
// hidden, the reader's choice checked and named, and the total underneath.
func TestPollResultsAreOneStopPerOption(t *testing.T) {
	_, n := renderDebug(t, Poll{Question: "Tabs or spaces?", Options: []PollOption{
		{Label: "Tabs", Votes: 5, Mine: true}, {Label: "Spaces", Votes: 3},
	}})
	th := core.DefaultTheme

	if len(buttonsOf(n)) != 0 {
		t.Error("a poll the reader has answered draws no buttons")
	}
	mine := findFirst(n, func(n *core.Node) bool {
		return n.Style.AccessibilityLabel == "Tabs, 63 percent, 5 votes, your choice"
	})
	other := findFirst(n, func(n *core.Node) bool {
		return n.Style.AccessibilityLabel == "Spaces, 37 percent, 3 votes"
	})
	if mine == nil || other == nil {
		t.Fatal("each result should be one stop named label, share, votes (and the reader's own, marked)")
	}
	label := findText(mine, "Tabs ✓")
	if label == nil || label.Style.TextColor != th.Colors.PrimaryOnLightColor() || !label.Style.AccessibilityHidden {
		t.Error("the reader's choice is checked, in the on-light Primary, and hidden under the stop's name")
	}
	if findText(other, "37%") == nil || findText(other, "Spaces") == nil {
		t.Error("the other option draws its plain label and its share")
	}
	// Both found by looking at the rendered lesson, not by a test: a result
	// keeps the theme's Column inset off, so it lines up with the question,
	// and the question is in the primary ink, not Subtitle's grey.
	if mine.Style.Padding != (core.EdgeInsets{}) {
		t.Errorf("result padding = %+v, want none: it must line up with the question", mine.Style.Padding)
	}
	if q := findText(n, "Tabs or spaces?"); q.Style.TextColor != th.Colors.TextPrimary {
		t.Errorf("question ink = %q, want TextPrimary", q.Style.TextColor)
	}
	if findText(n, "8 votes") == nil {
		t.Error("the total is drawn under the results")
	}
}

// ShowResults draws the bars with no vote from the reader, and a nil OnVote
// is then no concern. With no votes at all every share reads 0%.
func TestPollShowResultsWithNoVotes(t *testing.T) {
	_, n := renderDebug(t, Poll{Question: "Q", ShowResults: true,
		Options: []PollOption{{Label: "A"}, {Label: "B"}}})
	if len(buttonsOf(n)) != 0 {
		t.Error("ShowResults draws results, not buttons")
	}
	if findFirst(n, func(n *core.Node) bool { return n.Style.AccessibilityLabel == "A, 0 percent, 0 votes" }) == nil {
		t.Error("an unvoted poll's results read 0 percent")
	}
	if findText(n, "0 votes") == nil {
		t.Error("the total reads 0 votes")
	}
}

// Disabled keeps asking with inert buttons, and owes no OnVote.
func TestPollDisabledIsInertWithoutAConcern(t *testing.T) {
	_, n := renderDebug(t, Poll{Question: "Q", Options: askingOptions, Disabled: true})
	for _, b := range buttonsOf(n) {
		if !b.Style.Disabled {
			t.Errorf("button %q is not disabled", b.Props["label"])
		}
	}
}

// Asking with nowhere to report the answer is reported.
func TestPollAskingWithNoOnVoteReportsInert(t *testing.T) {
	newQuietRowHarness(t, func() core.View { return Poll{Question: "Q", Options: askingOptions} })
	if dump := core.DumpConcerns(); !strings.Contains(dump, ConcernPollInert) {
		t.Errorf("concerns = %q, want %s", dump, ConcernPollInert)
	}
}
