package comps

import (
	"fmt"
	"strings"

	"github.com/rohanthewiz/grmob/core"
)

// SearchableSelect is a choice from a list too long to scroll: a search field
// whose typing filters the options into a short list under it, where a tap
// picks one.
//
//	comps.SearchableSelect{
//	    Label:         "Country",
//	    Options:       countries,          // []core.SelectOption
//	    Value:         country.Get(),
//	    OnChange:      country.Set,
//	    Query:         query.Get(),
//	    OnQueryChange: query.Set,
//	}
//
//	┌ Column ────────────────────────────────────────────┐
//	│ ┌ SearchField (RoleSearch) ──────────────────────┐ │
//	│ │ 🔍  an                                     ✕   │ │
//	│ └────────────────────────────────────────────────┘ │
//	│ ┌ Column RoleListBox "Country suggestions" ──────┐ │  only while the
//	│ │ Argentina        South America     (option)    │ │  query is not the
//	│ │ Canada           North America     (option)    │ │  chosen label
//	│ └────────────────────────────────────────────────┘ │
//	│ Text RoleStatus  "2 of 6 matches"                  │  hidden while shut
//	└────────────────────────────────────────────────────┘
//
// # When the list shows
//
// While Query is not empty and is not exactly the chosen option's label.
// Picking an option sets the query to its label, so the list closes, and the
// field now reads as the choice. Editing that text opens the list again. The
// widget keeps no open flag: both halves of the condition are the caller's
// state already, which keeps the widget free of hooks and so safe to render
// conditionally, as SearchField is.
//
// Nothing shows for an empty query. Listing every option on focus would need
// a focus flag, held either in a hook or in more caller state, and a list
// short enough to show in full is one a Select or a RadioGroup already
// serves better.
//
// # Focus and the keyboard, which are the design
//
// The shape is a field and a list. What had to be decided is how the two
// share the keyboard. There are five parts:
//
//  1. The list never takes focus. It appears under a field the user is typing
//     in, and nothing issues a focus command, so typing carries on.
//  2. The return key belongs to the form, not to the list. The field has no
//     OnSubmit, so with FocusRef in a core.UseFocusOrder the keyboard shows
//     Next and moves on to the following field. The list is not in the
//     order: core.FocusNext walks declared refs only, and no option is one.
//     "Enter picks the top match" was the alternative. It was rejected
//     because an explicit submit suppresses the Next action (see
//     stampTraversal in core/focus_order.go), and a field in the middle of
//     a form cannot do both with one action key.
//  3. On the web, the list is a composite. Tab from the field lands on the
//     listbox as one stop, the arrows move among the options, and Enter or
//     Space picks. The WASM runtime supplies all of it from RoleListBox. The
//     highlight does not pick: selection does not follow focus, because an
//     arrow key that changed the value would close the list under the
//     user.
//  4. Picking dismisses the keyboard. On a phone the choice is made, so the
//     keyboard is in the way of the form. core.DismissKeyboard blurs only a
//     field that has focus, so a web user who picked with the arrow keys
//     keeps focus wherever it was.
//  5. On the web a keyboard pick removes the list, and the focused option
//     with it, so focus falls back to the page. Returning it to the field
//     would need core.Focus, which on a phone would raise the keyboard that
//     part 4 just put away. The widget cannot tell the two apart. A caller
//     that wants focus back can call core.Focus from OnChange.
//
// # It is not an ARIA combobox
//
// ARIA's pattern for this is role="combobox" on the field, with
// aria-expanded, aria-controls and aria-activedescendant naming the list and
// its active option. core.Role carries none of them, and adding the pattern
// means changes in both web exporters, the ARIA fixture and the runtime's
// keyboard. That is renderer work, which this plan excludes. What ships
// instead is a labelled search region, a labelled listbox, and a polite
// status line that says how many options match. The status is the part a
// screen reader user most needs, because the list appears silently under
// the field.
//
// The status line is hidden while the list is shut, and a live region that
// becomes visible is not announced reliably. Later keystrokes change its text
// while it is showing, and those are announced.
//
// # Options
//
// Options are core.SelectOption, the type core.Select takes, so a list can
// move from one to the other unchanged. Group becomes the row's subtitle
// rather than a heading: a listbox owns its options, and a heading among them
// would be a foreign child. Disabled and GroupDisabled rows are shown and
// announced, and a tap on one does nothing.
//
// # Theme roles read
//
//	Field       as SearchField (Surface, Components.Input.BorderRadius)
//	List frame  Colors.Background, Colors.Border, the input radius
//	Rows        as ListRow, with SelectedStyle for the current value
//	Status      Typography.Caption, Colors.TextSecondary
type SearchableSelect struct {
	// Options are the choices, in the order matches are listed.
	Options []core.SelectOption

	// Value is the chosen option's Value; empty means nothing is chosen.
	Value string

	// OnChange receives the Value of a picked option, and "" when the clear
	// button empties the field. It is not called when the user picks the
	// option already chosen.
	OnChange func(string)

	// Query is the field's text. It is the caller's, as SearchField.Value is.
	Query string

	// OnQueryChange receives every edit, the chosen label after a pick, and
	// "" on clear. Nil leaves a field that drops keystrokes.
	OnQueryChange func(string)

	// Label names the field and, with " suggestions", the list. Empty falls
	// back to the placeholder, as SearchField does.
	Label string

	// Placeholder is the empty field's prompt. Empty is "Search".
	Placeholder string

	// MaxResults caps the rows shown; the status line says how many more
	// matched. Zero is 6, which fits under a field above a phone keyboard.
	MaxResults int

	// Filter decides whether an option matches the query. The query is
	// trimmed first, and an option is only offered when Filter is true. Nil
	// matches a case-insensitive substring of Label.
	Filter func(opt core.SelectOption, query string) bool

	// Count writes the status line from the rows shown and the total that
	// matched. Nil writes "No matches", "1 match", "3 matches" or
	// "6 of 14 matches". Supply it to say it in another language.
	Count func(shown, total int) string

	// FocusRef names the field, so the field can join a core.UseFocusOrder
	// and be the target of core.Focus. See "Focus and the keyboard".
	FocusRef *core.FocusRef

	// Style is applied to the outer column after the widget's own props.
	Style []core.StyleProp
}

// searchableSelectMaxResults is MaxResults' default: enough rows to read as a
// list, few enough to stay above a phone keyboard under a field placed
// halfway down a form.
const searchableSelectMaxResults = 6

// Render builds Column(SearchField, listbox?, status) as drawn in the type
// doc.
func (s SearchableSelect) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	label := orDefault(s.Label, orDefault(s.Placeholder, "Search"))
	open := s.open()
	var shown []core.SelectOption
	total := 0
	if open {
		shown, total = s.matches()
	}

	items := make([]core.PropsAndChildren, 0, len(s.Style)+6)
	// No inset: the field and the list sit where the caller put the widget.
	// XS keeps the list visibly attached to the field it belongs to.
	items = append(items, core.Padding(0), core.Gap(float64(t.Spacing.XS)))
	for _, sp := range s.Style {
		items = append(items, sp)
	}

	items = append(items, SearchField{
		Value:              s.Query,
		Placeholder:        s.Placeholder,
		OnChange:           s.OnQueryChange,
		OnClear:            s.clear,
		AccessibilityLabel: label,
		FocusRef:           s.FocusRef,
	})
	// The listbox only while it has rows. An empty listbox would still be a
	// tab stop on the web with nothing to arrow to, and "No matches" is the
	// status line's to say.
	if len(shown) > 0 {
		items = append(items, s.list(ctx, t, label, shown))
	}
	items = append(items, s.status(t, open, len(shown), total))

	return core.Column(items...).Render(ctx)
}

// open reports whether the list shows: a query that is not already the chosen
// option's label. See "When the list shows".
func (s SearchableSelect) open() bool {
	if s.Query == "" {
		return false
	}
	for _, o := range s.Options {
		if o.Value == s.Value && s.Value != "" {
			return s.Query != o.Label
		}
	}
	return true
}

// matches returns the first MaxResults matching options in Options order, and
// how many matched in all.
func (s SearchableSelect) matches() ([]core.SelectOption, int) {
	limit := s.MaxResults
	if limit <= 0 {
		limit = searchableSelectMaxResults
	}
	q := strings.TrimSpace(s.Query)
	match := s.Filter
	if match == nil {
		lower := strings.ToLower(q)
		match = func(o core.SelectOption, _ string) bool {
			return strings.Contains(strings.ToLower(o.Label), lower)
		}
	}

	var shown []core.SelectOption
	total := 0
	for _, o := range s.Options {
		if !match(o, q) {
			continue
		}
		total++
		if len(shown) < limit {
			shown = append(shown, o)
		}
	}
	return shown, total
}

// list builds the framed listbox of matching rows.
func (s SearchableSelect) list(ctx *core.Context, t *core.Theme, label string, shown []core.SelectOption) core.View {
	items := make([]core.PropsAndChildren, 0, len(shown)+8)
	items = append(items,
		// Padding(0) and no gap, as RadioGroup: each row carries the theme's
		// row padding, and a gap would leave dead strips between targets.
		core.Padding(0),
		core.Gap(0),
		core.BackgroundColor(t.Colors.Background),
		core.BorderWidth(1),
		core.BorderColor(t.Colors.BorderColor()),
		core.BorderRadius(t.Components.Input.BorderRadius),
		core.AccessibilityRole(core.RoleListBox),
		core.AccessibilityLabel(label+" suggestions"),
	)
	for _, o := range shown {
		items = append(items, s.row(ctx, o))
	}
	return core.Column(items...)
}

// row builds one option. A disabled row keeps its handler and carries
// core.Disabled, the same contract RadioGroup's rows follow: a native tap
// racing the disabling patch must still find a handler, and the guard in the
// handler is what refuses it.
func (s SearchableSelect) row(ctx *core.Context, o core.SelectOption) ListRow {
	disabled := o.Disabled || o.GroupDisabled
	var style []core.StyleProp
	if disabled {
		style = append(style, core.Disabled(true))
	}
	return ListRow{
		Title:              o.Label,
		Subtitle:           o.Group,
		Selectable:         true,
		Selected:           o.Value == s.Value,
		OnTap:              s.pick(ctx, o, disabled),
		Style:              style,
		AccessibilityLabel: o.Label,
		AccessibilityHint:  o.Group,
	}
}

// pick returns one row's handler: report a changed value, write the label
// into the field so the list closes, and put the keyboard away (part 4 of
// "Focus and the keyboard").
func (s SearchableSelect) pick(ctx *core.Context, o core.SelectOption, disabled bool) func() {
	return func() {
		if disabled {
			return
		}
		if o.Value != s.Value && s.OnChange != nil {
			s.OnChange(o.Value)
		}
		if s.OnQueryChange != nil {
			s.OnQueryChange(o.Label)
		}
		core.DismissKeyboard(ctx)
	}
}

// clear empties the field and, when something was chosen, the choice: a field
// showing nothing over a value still set would be a form that submits what it
// does not show.
func (s SearchableSelect) clear() {
	if s.OnQueryChange != nil {
		s.OnQueryChange("")
	}
	if s.Value != "" && s.OnChange != nil {
		s.OnChange("")
	}
}

// status builds the polite count line. It is always in the tree, hidden while
// the list is shut, so it keeps its position among the column's unkeyed
// children and is not added and removed on every open and close.
func (s SearchableSelect) status(t *core.Theme, open bool, shown, total int) core.View {
	text := ""
	if open {
		count := s.Count
		if count == nil {
			count = matchCount
		}
		text = count(shown, total)
	}
	props := []core.StyleProp{
		core.FontSize(t.Typography.Caption.FontSize),
		core.TextColor(t.Colors.TextSecondary),
		core.AccessibilityRole(core.RoleStatus),
	}
	if !open {
		props = append(props, core.Display(core.DisplayNone))
	}
	return core.Text(text, props...)
}

// matchCount is Count's English default.
func matchCount(shown, total int) string {
	switch {
	case total == 0:
		return "No matches"
	case shown < total:
		return fmt.Sprintf("%d of %d matches", shown, total)
	case total == 1:
		return "1 match"
	}
	return fmt.Sprintf("%d matches", total)
}
