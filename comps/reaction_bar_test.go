package comps

import (
	"slices"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

var testReactions = []Reaction{
	{Emoji: "👍", Count: 3, Mine: true, Label: "thumbs up"},
	{Emoji: "🎉", Count: 1, Label: "party popper"},
	{Emoji: "😢", Count: 0, Label: "crying face"},
}

// A labelled group of chips: one per reaction in use, the reader's own
// selected, each named in words with its count in the right number.
func TestReactionBarIsAGroupOfNamedChips(t *testing.T) {
	_, n := renderDebug(t, ReactionBar{Reactions: testReactions, OnToggle: func(string) {}})

	if n.Style.AccessibilityRole != core.RoleGroup || n.Style.AccessibilityLabel != "Reactions" {
		t.Errorf("role %q label %q, want group / Reactions", n.Style.AccessibilityRole, n.Style.AccessibilityLabel)
	}
	chips := buttonsOf(n)
	if len(chips) != 2 {
		t.Fatalf("chips = %d, want 2: a count of zero is not drawn", len(chips))
	}
	if chips[0].Props["label"] != "👍 3" || chips[0].Style.AccessibilityLabel != "thumbs up, 3 reactions" {
		t.Errorf("first chip = %q / %q", chips[0].Props["label"], chips[0].Style.AccessibilityLabel)
	}
	if chips[1].Style.AccessibilityLabel != "party popper, 1 reaction" {
		t.Errorf("second chip's name = %q, want the singular", chips[1].Style.AccessibilityLabel)
	}
	if chips[0].Style.AccessibilitySelected != core.SelectedOn || chips[1].Style.AccessibilitySelected != core.SelectedOff {
		t.Errorf("selected = %q, %q; want Mine on and the other stated off",
			chips[0].Style.AccessibilitySelected, chips[1].Style.AccessibilitySelected)
	}
}

// A tap reports the chip's emoji, once, and changes nothing by itself: the
// counts are the caller's.
func TestReactionBarTapReportsTheEmoji(t *testing.T) {
	var got []string
	ctx, n := renderDebug(t, ReactionBar{Reactions: testReactions, OnToggle: func(e string) { got = append(got, e) }})
	ctx.TriggerCallback(buttonsOf(n)[1].Props["onClick"].(string))
	if !slices.Equal(got, []string{"🎉"}) {
		t.Errorf("toggles = %v, want exactly [🎉]", got)
	}
}

// Count 0 with Mine is a caller's bug, and is drawn so that it gets seen.
func TestReactionBarDrawsAZeroCountThatIsMine(t *testing.T) {
	_, n := renderDebug(t, ReactionBar{
		Reactions: []Reaction{{Emoji: "👍", Mine: true, Label: "thumbs up"}},
		OnToggle:  func(string) {},
	})
	if chips := buttonsOf(n); len(chips) != 1 || chips[0].Props["label"] != "👍 0" {
		t.Errorf("want the one chip reading 👍 0, got %d chips", len(chips))
	}
}

// Nothing to draw is display none, not an empty labelled group; a Trailing
// view is something to draw.
func TestReactionBarEmptyIsHiddenUnlessItHasATrailingView(t *testing.T) {
	_, empty := renderDebug(t, ReactionBar{OnToggle: func(string) {}})
	if empty.Style.Display != core.DisplayNone {
		t.Errorf("display = %q, want none for an empty bar", empty.Style.Display)
	}
	_, add := renderDebug(t, ReactionBar{
		OnToggle: func(string) {},
		Trailing: Chip{Label: "+", AccessibilityLabel: "Add reaction", OnTap: func() {}},
	})
	if add.Style.Display == core.DisplayNone || len(buttonsOf(add)) != 1 {
		t.Error("a bar with only a Trailing view should draw it")
	}
}

// Disabled: every chip refuses the tap, and no concern is raised for the
// missing OnToggle.
func TestReactionBarDisabledIsInertWithoutAConcern(t *testing.T) {
	fired := false
	ctx, n := renderDebug(t, ReactionBar{Reactions: testReactions, Disabled: true, OnToggle: func(string) { fired = true }})
	for _, c := range buttonsOf(n) {
		if !c.Style.Disabled {
			t.Errorf("chip %q is not disabled", c.Props["label"])
		}
		ctx.TriggerCallback(c.Props["onClick"].(string))
	}
	if fired {
		t.Error("a disabled bar must not report a toggle")
	}
	renderDebug(t, ReactionBar{Reactions: testReactions, Disabled: true})
}

// Tappable-looking chips with nowhere to report are reported.
func TestReactionBarWithNoOnToggleReportsInert(t *testing.T) {
	newQuietRowHarness(t, func() core.View { return ReactionBar{Reactions: testReactions} })
	if dump := core.DumpConcerns(); !strings.Contains(dump, ConcernReactionBarInert) {
		t.Errorf("concerns = %q, want %s", dump, ConcernReactionBarInert)
	}
}

// The caller's Style and GroupLabel land on the strip.
func TestReactionBarStyleAndGroupLabel(t *testing.T) {
	_, n := renderDebug(t, ReactionBar{
		Reactions: testReactions, OnToggle: func(string) {},
		GroupLabel: "Reações", Style: []core.StyleProp{core.MarginTop(4)},
	})
	if n.Style.AccessibilityLabel != "Reações" || n.Style.Margin != (core.EdgeInsets{Top: 4}) {
		t.Errorf("label %q margin %+v", n.Style.AccessibilityLabel, n.Style.Margin)
	}
}
