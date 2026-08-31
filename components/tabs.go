package components

import "github.com/rohanthewiz/grmob/core"

// Tabs is the named-field facade over core.TabView.
//
// Wrap, not supersede (the open question in the element-lessons plan):
// core.TabView defines a wire contract — the "TabView" node type with its
// tabs/selectedIndex/onTabChange props that the native renderers consume —
// and node-type contracts belong in core, next to the registry of types the
// renderers know. What TabView lacks is only ergonomics: four positional
// option props with no record of which is which at a call site. This struct
// supplies the field names and delegates everything else, so there is exactly
// one tab implementation to keep in sync with the renderers.
type Tabs struct {
	Items []core.TabItem // build with core.Tab(label, icon)
	// Selected is the controlled selection index; pair it with OnChange
	// writing to the state it is read from.
	Selected int
	OnChange func(int)
	// Content holds the tab pages. By the core.TabView contract all pages
	// are children of the node; the native side shows the selected one.
	Content []core.View
}

func (t Tabs) Render(ctx *core.Context) *core.Node {
	props := []core.TabViewProp{
		core.Tabs(t.Items...),
		core.SelectedIndex(t.Selected),
		core.Content(t.Content...),
	}
	// Only register a callback when there is a handler: TabView skips the
	// onTabChange prop for a nil handler, keeping the node diff-stable for
	// static tab strips.
	if t.OnChange != nil {
		props = append(props, core.OnTabChange(t.OnChange))
	}
	return core.TabView(props...).Render(ctx)
}
