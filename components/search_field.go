package components

import "github.com/rohanthewiz/grmob/core"

// SearchField is a text field dressed as a search box: a leading magnifier, a
// flexible input, and a clear button that appears once there is something to
// clear.
//
//	components.SearchField{
//	    Value:    query.Get(),
//	    OnChange: query.Set,
//	    OnSubmit: run,
//	}
//
//	┌ Row (Surface, rounded) ──────────────────────────────────────┐
//	│ 🔍   ┌ Input FlexGrow(1) ─────────────┐   [✕]                │
//	│      └────────────────────────────────┘                      │
//	└──────────────────────────────────────────────────────────────┘
//
// # It holds no state and calls no hook
//
// Value is the caller's, exactly as with Chip's Selected and DataTable's
// Sort, and every keystroke arrives through OnChange for the caller to store.
// That is worth stating because the obvious alternative — a field that owns
// its own text — would make this the second widget in the package with hook
// obligations, and a search box is a thing screens render conditionally (in a
// header that appears when a "Search" action is tapped), which is precisely
// what a hook-slot consumer must not be.
//
// # Debouncing is deliberately not in here
//
// A controlled field cannot debounce its own OnChange: the value has to reach
// state on the keystroke or the characters do not appear. What wants delaying
// is the *reaction* — the query, the filter, the fetch — and that lives in the
// caller. hooks.UseDebounce is the piece for it:
//
//	d := hooks.UseDebounce(ctx, 250*time.Millisecond)
//	components.SearchField{
//	    Value: query.Get(),
//	    OnChange: func(s string) {
//	        query.Set(s)                       // now, so typing looks like typing
//	        d.Call(func() { search(s) })       // in 250ms, if the typing stopped
//	    },
//	    OnSubmit: func() { d.Cancel(); search(query.Get()) },
//	}
//
// Splitting it that way is also what makes the two paths honest: Enter should
// search immediately, and Cancel is how the pending call gets out of the way.
//
// # The frame
//
// The row paints the theme's Surface at the theme's own field radius and the
// input inside it is flattened — transparent, no radius, no padding of its
// own. Without that the theme's Input base (its own background and corners)
// would draw a second box inside the first, which is what a hand-rolled
// search row looks like before someone notices.
type SearchField struct {
	// Value is the current text. The field is controlled: it renders what it
	// is given and reports edits through OnChange.
	Value string

	// Placeholder is the empty-field prompt. Empty is "Search".
	Placeholder string

	// OnChange receives every edit, including the clear. A nil OnChange makes
	// the field read-only in practice — it will render Value and drop
	// keystrokes — so it is worth setting even on a field you expect not to
	// change.
	OnChange func(string)

	// OnSubmit fires on the keyboard's return / IME done action. When nil the
	// field carries no submit callback at all, which keeps a
	// search-as-you-type box off the InputWithSubmit path entirely.
	OnSubmit func()

	// OnClear replaces what the clear button does. The default is
	// OnChange(""), which is what a clear means for a controlled field; set
	// this when clearing also has to dismiss results, cancel a pending
	// request, or restore a previous view.
	OnClear func()

	// Glyph is the leading mark. Empty is 🔍; NoGlyph drops it, for a field
	// whose surroundings already say what it searches.
	Glyph   string
	NoGlyph bool

	// AccessibilityLabel names the field for screen readers. Empty falls back
	// to the placeholder, which is the resolved one — so the default field
	// announces as "Search" rather than as nothing.
	//
	// The fallback exists because a placeholder is not a label on any
	// platform: it is a value hint that vanishes on the first keystroke, so a
	// field relying on it alone is unnamed for exactly the users who most
	// need the name.
	AccessibilityLabel string

	// Style is applied to the row after the widget's own frame, so the fill,
	// the radius and the padding are all overridable.
	Style []core.StyleProp
}

func (s SearchField) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	placeholder := s.Placeholder
	if placeholder == "" {
		placeholder = "Search"
	}
	label := s.AccessibilityLabel
	if label == "" {
		label = placeholder
	}

	items := make([]core.PropsAndChildren, 0, len(s.Style)+6)
	items = append(items,
		core.BackgroundColor(t.Colors.Surface),
		// The theme's own field radius, so the search box matches every other
		// input on the screen rather than agreeing with them by coincidence.
		core.BorderRadius(t.Components.Input.BorderRadius),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.Gap(float64(t.Spacing.SM)),
		core.PaddingVertical(t.Spacing.XS),
	)
	for _, sp := range s.Style {
		items = append(items, sp)
	}

	if glyph := s.glyph(); glyph != "" {
		items = append(items, core.Text(glyph,
			core.FontSize(t.Typography.Body.FontSize),
			core.TextColor(t.Colors.TextSecondary),
			// The field next to it is already named; a reader announcing the
			// magnifier as well would name the same control twice.
			core.AccessibilityHidden(),
		))
	}
	items = append(items, s.input(placeholder, label))

	// Only when there is something to clear, and last in the row so its
	// arrival never shifts the input's position among its siblings (row
	// children are unkeyed, so identity there is positional).
	if s.Value != "" {
		items = append(items, Button{
			Label:              "✕",
			Emphasis:           EmphasisGhost,
			OnTap:              s.clear,
			AccessibilityLabel: "Clear " + label,
			Style: []core.StyleProp{
				core.TextColor(t.Colors.TextSecondary),
				core.PaddingHorizontal(6),
				core.PaddingVertical(0),
			},
		})
	}

	return core.Row(items...).Render(ctx)
}

// input builds the flattened field: the theme's Input base with its frame
// removed, since the row around it is already the frame.
func (s SearchField) input(placeholder, label string) core.View {
	onChange := s.OnChange
	if onChange == nil {
		// core.Input registers whatever it is given and the dispatcher
		// invokes it unguarded, so a nil here panics on the first keystroke
		// rather than ignoring it. Same guard Chip and Button apply to OnTap.
		onChange = func(string) {}
	}

	flatten := []core.PropsAndChildren{
		core.FlexGrow(1),
		core.BackgroundColor(ColorTransparent),
		core.BorderRadius(0),
		// Assigns rather than merges, which is the point: the theme's Input
		// padding would inset the text from a frame that is no longer there.
		core.Padding(0),
		core.AccessibilityLabel(label),
	}

	if s.OnSubmit == nil {
		return core.Input(s.Value, placeholder, onChange, flatten...)
	}
	return core.InputWithSubmit(s.Value, placeholder, onChange, s.OnSubmit, flatten...)
}

// clear runs the caller's OnClear, or empties the value through OnChange.
func (s SearchField) clear() {
	if s.OnClear != nil {
		s.OnClear()
		return
	}
	if s.OnChange != nil {
		s.OnChange("")
	}
}

func (s SearchField) glyph() string {
	if s.NoGlyph {
		return ""
	}
	if s.Glyph != "" {
		return s.Glyph
	}
	return "🔍"
}
