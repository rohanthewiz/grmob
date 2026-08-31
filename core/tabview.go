package core

type TabViewNode struct {
	SelectedIndex int
	OnTabChange   func(int)
	Tabs          []TabItem
	Content       []View
}

type TabItem struct {
	Label string
	Icon  string
}

type TabViewProp interface {
	Apply(*TabViewNode)
}

func TabView(props ...TabViewProp) View {
	return ComponentFunc(func(ctx *Context) *Node {
		node := &TabViewNode{}

		for _, p := range props {
			p.Apply(node)
		}

		// serialização dos tabs
		tabs := []map[string]string{}
		for _, t := range node.Tabs {
			tabs = append(tabs, map[string]string{
				"label": t.Label,
				"icon":  t.Icon,
			})
		}

		propMap := map[string]any{
			"selectedIndex": node.SelectedIndex,
			"tabs":          tabs,
		}
		if node.OnTabChange != nil {
			propMap["onTabChange"] = ctx.registerIntCallback(node.OnTabChange)
		}

		return &Node{
			Type:     "TabView",
			Props:    propMap,
			Children: renderAll(ctx, "TabView", node.Content),
		}
	})
}

type tabViewFunc func(*TabViewNode)

func (f tabViewFunc) Apply(t *TabViewNode) { f(t) }

func SelectedIndex(i int) TabViewProp {
	return tabViewFunc(func(t *TabViewNode) {
		t.SelectedIndex = i
	})
}

func OnTabChange(fn func(int)) TabViewProp {
	return tabViewFunc(func(t *TabViewNode) {
		t.OnTabChange = fn
	})
}

func Tabs(tabs ...TabItem) TabViewProp {
	return tabViewFunc(func(t *TabViewNode) {
		t.Tabs = tabs
	})
}

func Content(views ...View) TabViewProp {
	return tabViewFunc(func(t *TabViewNode) {
		t.Content = views
	})
}

func Tab(label string, icon string) TabItem {
	return TabItem{Label: label, Icon: icon}
}
