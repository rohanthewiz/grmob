package comps

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

var sampleCountries = []core.SelectOption{
	{Value: "fr", Label: "France", Group: "Europe"},
	{Value: "de", Label: "Germany", Group: "Europe"},
	{Value: "jp", Label: "Japan", Group: "Asia"},
	{Value: "ke", Label: "Kenya", Group: "Africa"},
	{Value: "no", Label: "Norway", Group: "Europe", Disabled: true},
}

func sampleSelect(value, query string) SearchableSelect {
	return SearchableSelect{
		Label:         "Country",
		Options:       sampleCountries,
		Value:         value,
		OnChange:      func(string) {},
		Query:         query,
		OnQueryChange: func(string) {},
	}
}

func selectListbox(n *core.Node) *core.Node {
	return findFirst(n, func(n *core.Node) bool {
		return n.Style != nil && n.Style.AccessibilityRole == core.RoleListBox
	})
}

func selectStatus(t *testing.T, n *core.Node) *core.Node {
	t.Helper()
	s := findFirst(n, func(n *core.Node) bool {
		return n.Style != nil && n.Style.AccessibilityRole == core.RoleStatus
	})
	if s == nil {
		t.Fatal("the status line must always be in the tree")
	}
	return s
}

func selectRows(n *core.Node) []*core.Node {
	var out []*core.Node
	if lb := selectListbox(n); lb != nil {
		for _, c := range lb.Children {
			if c.Style != nil && c.Style.AccessibilityRole == core.RoleOption {
				out = append(out, c)
			}
		}
	}
	return out
}

func TestSearchableSelectIsShutWithNoQuery(t *testing.T) {
	_, n := renderDebug(t, sampleSelect("", ""))

	if n.Type != "Column" {
		t.Fatalf("root = %q, want Column", n.Type)
	}
	if findFirst(n, func(n *core.Node) bool {
		return n.Style != nil && n.Style.AccessibilityRole == core.RoleSearch
	}) == nil {
		t.Error("the field is a SearchField, the search region")
	}
	if selectListbox(n) != nil {
		t.Error("no listbox for an empty query")
	}
	if s := selectStatus(t, n); s.Style.Display != core.DisplayNone {
		t.Error("the status line is hidden while the list is shut")
	}
}

// Typing filters by label, case-insensitively, in Options order; the rows are
// options in a labelled listbox, with the group as subtitle.
func TestSearchableSelectFiltersIntoALabelledListbox(t *testing.T) {
	_, n := renderDebug(t, sampleSelect("", "AN"))

	lb := selectListbox(n)
	if lb == nil || lb.Style.AccessibilityLabel != "Country suggestions" {
		t.Fatal("matches are a listbox named after the field")
	}
	rows := selectRows(n)
	want := []string{"France", "Germany", "Japan"}
	if len(rows) != len(want) {
		t.Fatalf("got %d rows, want %v", len(rows), want)
	}
	for i, w := range want {
		if rows[i].Style.AccessibilityLabel != w {
			t.Errorf("row %d = %q, want %q", i, rows[i].Style.AccessibilityLabel, w)
		}
	}
	if findText(rows[0], "Europe") == nil {
		t.Error("the option's Group is its subtitle")
	}
	if s := selectStatus(t, n); s.Style.Display == core.DisplayNone || s.Props["content"] != "3 matches" {
		t.Errorf("status = %v (display %q), want a visible \"3 matches\"", s.Props["content"], s.Style.Display)
	}
}

func selectField(t *testing.T, n *core.Node) *core.Node {
	t.Helper()
	f := findFirst(n, func(n *core.Node) bool { return n.Type == "Input" })
	if f == nil {
		t.Fatal("no field in the select")
	}
	return f
}

// The field is the combobox: expanded and pointing at the list while rows show,
// collapsed and pointing nowhere while they do not. Each row carries the slot id
// the runtime's aria-activedescendant names.
func TestSearchableSelectFieldIsAComboboxControllingTheList(t *testing.T) {
	_, n := renderDebug(t, sampleSelect("", "an"))
	field := selectField(t, n)
	if field.Style.AccessibilityRole != core.RoleComboBox {
		t.Fatalf("field role = %q, want combobox on the input itself", field.Style.AccessibilityRole)
	}
	if field.Style.AccessibilityExpanded != core.ExpandedOpen {
		t.Error("the field says expanded while the list shows")
	}
	lb := selectListbox(n)
	if lb.Style.AccessibilityID != "searchable-select-country" ||
		field.Style.AccessibilityControls != lb.Style.AccessibilityID {
		t.Errorf("list id %q, field controls %q; want both searchable-select-country",
			lb.Style.AccessibilityID, field.Style.AccessibilityControls)
	}
	for i, want := range []string{
		"searchable-select-country-option-0",
		"searchable-select-country-option-1",
		"searchable-select-country-option-2",
	} {
		if got := selectRows(n)[i].Style.AccessibilityID; got != want {
			t.Errorf("row %d id = %q, want %q", i, got, want)
		}
	}
	if search := findFirst(n, func(n *core.Node) bool {
		return n.Style != nil && n.Style.AccessibilityRole == core.RoleSearch
	}); search == nil || search.Type == "Input" {
		t.Error("the search landmark stays on the row around the field")
	}

	// Shut and no-match alike: collapsed, and no aria-controls to dangle.
	// renderDebug audits the tree, so a dangling reference would fail there too.
	for _, query := range []string{"", "zz", "Japan"} {
		value := ""
		if query == "Japan" {
			value = "jp"
		}
		_, n := renderDebug(t, sampleSelect(value, query))
		field := selectField(t, n)
		if field.Style.AccessibilityExpanded != core.ExpandedClosed || field.Style.AccessibilityControls != "" {
			t.Errorf("query %q: expanded %q controls %q, want false and none",
				query, field.Style.AccessibilityExpanded, field.Style.AccessibilityControls)
		}
	}
}

// ID replaces the derived list id; the derivation keeps only letters and digits.
func TestSearchableSelectIDNamesTheList(t *testing.T) {
	s := sampleSelect("", "an")
	s.ID = "ship-country"
	_, n := renderDebug(t, s)
	if got := selectListbox(n).Style.AccessibilityID; got != "ship-country" {
		t.Errorf("list id = %q, want the caller's ID", got)
	}
	if got := selectRows(n)[0].Style.AccessibilityID; got != "ship-country-option-0" {
		t.Errorf("row id = %q, want it prefixed by ID", got)
	}

	s.ID = ""
	s.Label = "  Ship to: country!"
	_, n = renderDebug(t, s)
	if got := selectListbox(n).Style.AccessibilityID; got != "searchable-select-ship-to-country" {
		t.Errorf("derived list id = %q", got)
	}
}

func TestSearchableSelectCapsRowsAndSaysSo(t *testing.T) {
	s := sampleSelect("", "an")
	s.MaxResults = 2
	_, n := renderDebug(t, s)

	if rows := selectRows(n); len(rows) != 2 {
		t.Errorf("MaxResults 2 shows %d rows", len(rows))
	}
	if got := selectStatus(t, n).Props["content"]; got != "2 of 3 matches" {
		t.Errorf("status = %v, want \"2 of 3 matches\"", got)
	}
}

// No matches: no listbox (it would be an empty tab stop), and the status says so.
func TestSearchableSelectWithNoMatchesSaysSo(t *testing.T) {
	_, n := renderDebug(t, sampleSelect("", "zz"))

	if selectListbox(n) != nil {
		t.Error("an empty listbox must not be rendered")
	}
	if got := selectStatus(t, n).Props["content"]; got != "No matches" {
		t.Errorf("status = %v, want \"No matches\"", got)
	}
}

// Once the query is the chosen option's label the list is shut; editing it
// opens the list again with the chosen row marked.
func TestSearchableSelectShutsOnTheChosenLabel(t *testing.T) {
	_, n := renderDebug(t, sampleSelect("jp", "Japan"))
	if selectListbox(n) != nil {
		t.Error("a query equal to the chosen label shows no list")
	}

	_, n = renderDebug(t, sampleSelect("jp", "Jap"))
	rows := selectRows(n)
	if len(rows) != 1 || rows[0].Style.AccessibilitySelected != core.SelectedOn {
		t.Error("editing the chosen label reopens the list with the chosen row selected")
	}
}

// A pick reports a changed value once, then writes the label into the field.
func TestSearchableSelectPickReportsValueThenLabel(t *testing.T) {
	var calls []string
	s := sampleSelect("", "an")
	s.OnChange = func(v string) { calls = append(calls, "value:"+v) }
	s.OnQueryChange = func(q string) { calls = append(calls, "query:"+q) }
	ctx, n := renderDebug(t, s)

	ctx.TriggerCallback(selectRows(n)[2].Props["onClick"].(string))
	if len(calls) != 2 || calls[0] != "value:jp" || calls[1] != "query:Japan" {
		t.Errorf("calls = %v, want [value:jp query:Japan]", calls)
	}

	// Picking the option already chosen only restores the label.
	calls = nil
	s.Value = "jp"
	s.Query = "ja"
	ctx, n = renderDebug(t, s)
	ctx.TriggerCallback(selectRows(n)[0].Props["onClick"].(string))
	if len(calls) != 1 || calls[0] != "query:Japan" {
		t.Errorf("re-picking the chosen option: calls = %v, want [query:Japan]", calls)
	}
}

func TestSearchableSelectDisabledOptionIsShownButNotPicked(t *testing.T) {
	picked := 0
	s := sampleSelect("", "nor")
	s.OnChange = func(string) { picked++ }
	s.OnQueryChange = func(string) { picked++ }
	ctx, n := renderDebug(t, s)

	rows := selectRows(n)
	if len(rows) != 1 || !rows[0].Style.Disabled {
		t.Fatal("Norway is listed and marked disabled")
	}
	ctx.TriggerCallback(rows[0].Props["onClick"].(string))
	if picked != 0 {
		t.Errorf("a disabled option reported %d changes", picked)
	}
}

// Clearing empties the field and the choice, so the form does not hold a
// value it no longer shows.
func TestSearchableSelectClearEmptiesQueryAndValue(t *testing.T) {
	var calls []string
	s := sampleSelect("jp", "Japan")
	s.OnChange = func(v string) { calls = append(calls, "value:"+v) }
	s.OnQueryChange = func(q string) { calls = append(calls, "query:"+q) }
	ctx, n := renderDebug(t, s)

	clear := findFirst(n, func(n *core.Node) bool {
		return n.Type == "Button" && n.Style != nil && n.Style.AccessibilityLabel == "Clear Country"
	})
	if clear == nil {
		t.Fatal("a field with text has a clear button named after it")
	}
	ctx.TriggerCallback(clear.Props["onClick"].(string))
	if len(calls) != 2 || calls[0] != "query:" || calls[1] != "value:" {
		t.Errorf("calls = %v, want [query: value:]", calls)
	}
}

// The return key is the form's: no submit of the widget's own, so a FocusRef in
// an order makes the field advertise Next. The list holds no field, so the
// order has nothing of it to walk.
func TestSearchableSelectLeavesTheReturnKeyToTheFocusOrder(t *testing.T) {
	core.SetDebugMode(true)
	core.ClearConcerns()
	t.Cleanup(func() { core.SetDebugMode(false); core.ClearConcerns() })

	ctx := core.NewContext()
	ctx.BeginRenderPass()
	country := core.UseFocusRef(ctx)
	city := core.UseFocusRef(ctx)
	core.UseFocusOrder(ctx, country, city)
	s := sampleSelect("", "an")
	s.FocusRef = country
	n := s.Render(ctx)
	ctx.EndRenderPass()
	core.AuditTree(n)
	if dump := core.DumpConcerns(); dump != "" {
		t.Errorf("concerns raised:\n%s", dump)
	}

	input := findFirst(n, func(n *core.Node) bool { return n.Type == "Input" })
	if input == nil || input.Props["imeAction"] != "next" {
		t.Fatalf("the field in an order advertises Next, got %v", input.Props["imeAction"])
	}
	if findFirst(selectListbox(n), func(n *core.Node) bool { return n.Type == "Input" }) != nil {
		t.Error("the list must hold no field for the order to walk")
	}
}
