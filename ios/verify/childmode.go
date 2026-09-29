package main

import (
	"encoding/json"
	"time"

	"github.com/rohanthewiz/grmob/comps"
	"github.com/rohanthewiz/grmob/core"
)

// childModeCase is one bundled widget, rendered by Go, with the VoiceOver
// shape each of its labelled containers should take on iOS (N-078): "ignore"
// (its children are all hidden: the label is the whole element), "contain"
// (its operable members stay elements under it) or "combine" (it is one
// element). The Swift harness computes the same thing with the renderer's own
// GrMobNode.containerStyle and grMobChildMode, and every labelled container
// in the tree has to be listed here, so a widget that changes shape shows up
// as a difference rather than a silence.
//
// The want column is a judgement per widget, written down where it can be
// argued with, not an output of the rule: "contain" where a reader has to be
// able to reach each member (two sliders, a grid of cells, a pad of keys),
// "combine" where the container is one thing to hear (a bubble, a row around
// one switch).
type childModeCase struct {
	Name string            `json:"name"`
	Tree json.RawMessage   `json:"tree"`
	Want map[string]string `json:"want"`
}

func childModeCases() []childModeCase {
	noop := func() {}
	cases := []struct {
		name string
		view core.View
		want map[string]string
	}{
		{"RangeSlider", comps.RangeSlider{Title: "Price", Max: 200, Low: 20, High: 80,
			OnChange: func(float64, float64) {}}, map[string]string{"Price": "contain"}},
		{"EditableGrid", comps.EditableGrid{Label: "Budget",
			Columns: []comps.GridColumn{{Title: "Item"}, {Title: "Amount", Kind: comps.GridNumber}},
			Rows:    [][]string{{"Rent", "1200"}, {"Bread", "4.5"}},
			Key:     func(i int) string { return []string{"a", "b"}[i] }, OnChange: func(int, int, string) {}},
			map[string]string{"Amount, row 1, 1200": "combine", "Amount, row 2, 4.5": "combine", "Budget": "contain", "Item, row 1, Rent": "combine", "Item, row 2, Bread": "combine", "Row": "combine"}},
		{"ColorSwatchPicker", comps.ColorSwatchPicker{Label: "Label colour", Value: "#2A78D6",
			Colors:   []comps.Swatch{{Hex: "#2A78D6", Name: "blue"}, {Hex: "#D65A2A", Name: "orange"}},
			OnChange: func(string) {}}, map[string]string{"Label colour": "contain", "blue": "ignore", "orange": "ignore"}},
		{"Stepper", comps.Stepper{Label: "Guests", Value: 2, Max: 9, OnChange: func(int) {}}, map[string]string{"Guests": "contain"}},
		{"NumberPad", comps.NumberPad{Label: "Number pad", OnKey: func(string) {}, OnBackspace: noop}, map[string]string{"Number pad": "contain"}},
		{"ReactionBar", comps.ReactionBar{Reactions: []comps.Reaction{{Emoji: "👍", Count: 3}, {Emoji: "🎉", Count: 1}},
			OnToggle: func(string) {}}, map[string]string{"Reactions": "contain"}},
		{"Poll", comps.Poll{Question: "Lunch?", Options: []comps.PollOption{{Label: "Soup"}, {Label: "Salad"}},
			OnVote: func(int) {}}, map[string]string{"Lunch?": "contain"}},
		{"Poll results", comps.Poll{Question: "Lunch?", ShowResults: true,
			Options: []comps.PollOption{{Label: "Soup", Votes: 3}, {Label: "Salad", Votes: 1}}}, map[string]string{"Lunch?": "contain", "Salad, 25 percent, 1 vote": "combine", "Soup, 75 percent, 3 votes": "combine"}},
		{"Wizard", comps.Wizard{Label: "Setup", Steps: []comps.WizardStep{
			{Title: "Name", Body: core.Text("Who are you?")}, {Title: "Done", Body: core.Text("All set")}},
			OnChange: func(int) {}, OnFinish: noop}, map[string]string{"Setup": "contain", "Step 1: Name": "ignore", "Step 2: Done": "ignore"}},
		{"TimePicker", comps.TimePicker{Label: "Alarm", Value: time.Date(2026, 9, 28, 7, 30, 0, 0, time.UTC),
			OnChange: func(time.Time) {}}, map[string]string{"Alarm": "contain"}},
		{"PINInput", comps.PINInput{Label: "Passcode", Length: 4, OnChange: func(string) {}}, map[string]string{"Passcode": "combine"}},
		{"Rating", comps.Rating{Label: "Rating", Value: 3, Max: 5, OnChange: func(int) {}}, map[string]string{"1 of 5": "combine", "2 of 5": "combine", "3 of 5": "combine", "4 of 5": "combine", "5 of 5": "combine", "Rating": "contain"}},
		{"Rating read-only", comps.Rating{Label: "Rating", Value: 3, Max: 5, ReadOnly: true}, map[string]string{"Rating": "ignore"}},
		{"TreeView", comps.TreeView{Label: "Files", Nodes: []comps.TreeNode{{ID: "a", Label: "a.go"}, {ID: "b", Label: "b.go"}},
			OnSelect: func(string) {}}, map[string]string{"Files": "contain", "a.go": "combine", "b.go": "combine"}},
		{"StepIndicator", comps.StepIndicator{Steps: []string{"Cart", "Pay", "Done"}, Current: 1, OnTap: func(int) {}},
			map[string]string{"Step 1: Cart, done": "ignore", "Step 2: Pay": "ignore", "Step 3: Done": "ignore"}},
		{"MessageBubble", comps.MessageBubble{Text: "Hi", Sender: "Ada", Time: "9:41"}, map[string]string{"Ada, Hi, 9:41": "ignore"}},
		{"Link", comps.Link{Text: "Docs", URL: "https://example.com"}, map[string]string{"Docs": "ignore"}},
		{"SwitchRow", comps.SwitchRow{Title: "Wi-Fi", On: true, OnToggle: func(bool) {}}, map[string]string{}},
		{"ListRow", comps.ListRow{Title: "Inbox", Subtitle: "3 new", OnTap: noop}, map[string]string{}},
		{"ListRow labelled", comps.ListRow{Title: "Ada Lovelace", Subtitle: "Hi there",
			AccessibilityLabel: "Ada Lovelace, Hi there, 2 unread", OnTap: noop,
			Leading: comps.Avatar{Name: "Ada Lovelace"}, Trailing: comps.Badge{Text: "2"}}, map[string]string{"Ada Lovelace": "combine", "Ada Lovelace, Hi there, 2 unread": "combine"}},
		{"Drawer", comps.Drawer{Open: true, Title: "Notebook", OnDismiss: noop,
			Items: []comps.DrawerItem{{Label: "Inbox"}, {Label: "Sent"}}, Content: core.Text("page")},
			map[string]string{"Inbox": "combine", "Notebook": "contain", "Sent": "combine"}},
	}
	out := make([]childModeCase, 0, len(cases))
	for _, c := range cases {
		ctx := core.NewContext()
		ctx.BeginRenderPass()
		tree, err := json.Marshal(c.view.Render(ctx))
		if err != nil {
			fatal("marshal %s: %v", c.name, err)
		}
		out = append(out, childModeCase{Name: c.name, Tree: tree, Want: c.want})
	}
	return out
}
