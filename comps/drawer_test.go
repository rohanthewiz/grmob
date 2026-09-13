package comps

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

func sampleDrawer(open bool, onPick, onDismiss func()) Drawer {
	return Drawer{
		Open:      open,
		OnDismiss: onDismiss,
		Title:     "Notebook",
		Items: []DrawerItem{
			{Icon: "📥", Label: "Inbox", OnTap: onPick},
			{Icon: "⭐", Label: "Starred", Subtitle: "3 notes"},
		},
		Content: core.Text("Screen body"),
	}
}

// drawerParts returns the two layers and the panel layer's two children.
func drawerParts(t *testing.T, n *core.Node) (content, layer, panel, scrim *core.Node) {
	t.Helper()
	if n.Type != "ZStack" || len(n.Children) != 2 {
		t.Fatalf("root = %q with %d children, want ZStack(content, panel layer)", n.Type, len(n.Children))
	}
	content, layer = n.Children[0], n.Children[1]
	if content.Type != "Box" || layer.Type != "Row" || len(layer.Children) != 2 {
		t.Fatalf("layers = %q, %q with %d children; want Box, Row(panel, scrim)",
			content.Type, layer.Type, len(layer.Children))
	}
	return content, layer, layer.Children[0], layer.Children[1]
}

func TestDrawerShutHidesThePanelAndLeavesTheScreenReadable(t *testing.T) {
	_, n := renderDebug(t, sampleDrawer(false, nil, func() {}))
	content, layer, panel, _ := drawerParts(t, n)

	if layer.Style.Display != core.DisplayNone {
		t.Errorf("a shut drawer's panel layer must be Display none, got %q", layer.Style.Display)
	}
	if content.Style.AccessibilityHidden {
		t.Error("a shut drawer must leave the screen in the accessibility tree")
	}
	if findText(content, "Screen body") == nil {
		t.Error("Content must be inside the content layer")
	}
	if panel.Style.AccessibilityRole != core.RoleNavigation || panel.Style.AccessibilityLabel != "Notebook" {
		t.Errorf("panel a11y = role %q label %q, want a navigation landmark named by Title",
			panel.Style.AccessibilityRole, panel.Style.AccessibilityLabel)
	}
	if panel.Style.Width != "280px" {
		t.Errorf("default panel width = %q, want 280px", panel.Style.Width)
	}
}

func TestDrawerOpenShowsThePanelAndHidesTheScreen(t *testing.T) {
	_, n := renderDebug(t, sampleDrawer(true, nil, func() {}))
	content, layer, _, scrim := drawerParts(t, n)

	if layer.Style.Display == core.DisplayNone {
		t.Error("an open drawer must not hide its panel layer")
	}
	if !content.Style.AccessibilityHidden {
		t.Error("an open drawer must hide the screen behind it from assistive technology")
	}
	if !scrim.Style.AccessibilityHidden || scrim.Style.Background != drawerDefaultBackdrop {
		t.Errorf("scrim hidden=%v fill=%q, want hidden with Modal's default backdrop",
			scrim.Style.AccessibilityHidden, scrim.Style.Background)
	}
}

// Opening is a patch, not a replace: the tree has the same shape either way.
func TestDrawerOpenAndShutHaveTheSameShape(t *testing.T) {
	var shape func(*core.Node) string
	shape = func(n *core.Node) string {
		s := n.Type + "("
		for _, c := range n.Children {
			s += shape(c) + ","
		}
		return s + ")"
	}
	_, shut := renderDebug(t, sampleDrawer(false, nil, func() {}))
	_, open := renderDebug(t, sampleDrawer(true, nil, func() {}))
	if a, b := shape(shut), shape(open); a != b {
		t.Errorf("shapes differ:\nshut %s\nopen %s", a, b)
	}
}

func TestDrawerRowPicksThenDismisses(t *testing.T) {
	var order []string
	ctx, n := renderDebug(t, sampleDrawer(true,
		func() { order = append(order, "pick") },
		func() { order = append(order, "dismiss") }))

	row := findFirst(n, func(c *core.Node) bool {
		return c.Style != nil && c.Style.AccessibilityLabel == "Inbox, selected"
	})
	if row == nil {
		t.Fatal("the first item is Selected by default and must be named \"Inbox, selected\"")
	}
	ctx.TriggerCallback(row.Props["onClick"].(string))
	if len(order) != 2 || order[0] != "pick" || order[1] != "dismiss" {
		t.Errorf("calls = %v, want [pick dismiss]", order)
	}
}

func TestDrawerNegativeSelectedMarksNone(t *testing.T) {
	d := sampleDrawer(true, nil, func() {})
	d.Selected = -1
	_, n := renderDebug(t, d)
	if findFirst(n, func(c *core.Node) bool {
		return c.Style != nil && len(c.Style.AccessibilityLabel) > 10 &&
			c.Style.AccessibilityLabel[len(c.Style.AccessibilityLabel)-10:] == ", selected"
	}) != nil {
		t.Error("Selected -1 must mark no row as selected")
	}
}

func TestDrawerCloseAndScrimDismiss(t *testing.T) {
	dismissed := 0
	ctx, n := renderDebug(t, sampleDrawer(true, nil, func() { dismissed++ }))
	_, _, panel, scrim := drawerParts(t, n)

	buttons := buttonsOf(panel)
	if len(buttons) != 1 || buttons[0].Style.AccessibilityLabel != "Close Notebook" {
		t.Fatalf("want one ✕ named \"Close Notebook\", got %d buttons", len(buttons))
	}
	ctx.TriggerCallback(buttons[0].Props["onClick"].(string))
	ctx.TriggerCallback(scrim.Props["onClick"].(string))
	if dismissed != 2 {
		t.Errorf("OnDismiss ran %d times, want 2 (✕, scrim)", dismissed)
	}
}

func TestDrawerWithoutOnDismissHasNoCloseAndAnInertScrim(t *testing.T) {
	_, n := renderDebug(t, sampleDrawer(true, nil, nil))
	_, _, panel, scrim := drawerParts(t, n)
	if len(buttonsOf(panel)) != 0 {
		t.Error("no OnDismiss must draw no ✕")
	}
	if _, ok := scrim.Props["onClick"]; ok {
		t.Error("no OnDismiss must leave the scrim without a tap handler")
	}
}

func TestDrawerCloseLabelOverridesTheName(t *testing.T) {
	d := sampleDrawer(true, nil, func() {})
	d.Title = ""
	_, n := renderDebug(t, d)
	_, _, panel, _ := drawerParts(t, n)
	if got := buttonsOf(panel)[0].Style.AccessibilityLabel; got != "Close navigation" {
		t.Errorf("untitled ✕ name = %q, want \"Close navigation\"", got)
	}

	d.CloseLabel = "Hide menu"
	_, n = renderDebug(t, d)
	_, _, panel, _ = drawerParts(t, n)
	if got := buttonsOf(panel)[0].Style.AccessibilityLabel; got != "Hide menu" {
		t.Errorf("✕ name = %q, want CloseLabel", got)
	}
}

func TestDrawerBodyReplacesTheRows(t *testing.T) {
	d := sampleDrawer(true, nil, func() {})
	d.Body = core.Text("Custom body")
	_, n := renderDebug(t, d)
	_, _, panel, _ := drawerParts(t, n)
	if findText(panel, "Custom body") == nil || findText(panel, "Inbox") != nil {
		t.Error("Body must replace the Items rows")
	}
}

func TestDrawerStylesReachTheirNodes(t *testing.T) {
	d := sampleDrawer(false, nil, func() {})
	d.Style = []core.StyleProp{core.Height("360px")}
	d.PanelStyle = []core.StyleProp{core.BackgroundColor("#123456")}
	d.Width = "70%"
	d.Backdrop = "#00000044"
	_, n := renderDebug(t, d)
	_, _, panel, scrim := drawerParts(t, n)

	if n.Style.Height != "360px" || n.Style.Width != "100%" {
		t.Errorf("stack = %q × %q, want Style's height over the default width", n.Style.Width, n.Style.Height)
	}
	if panel.Style.Background != "#123456" || panel.Style.Width != "70%" {
		t.Errorf("panel fill %q width %q, want PanelStyle's fill and Width", panel.Style.Background, panel.Style.Width)
	}
	if scrim.Style.Background != "#00000044" {
		t.Errorf("scrim = %q, want Backdrop", scrim.Style.Background)
	}
}

// CloseRef reaches the ✕: a core.Focus on it stamps the button on the next
// pass.
func TestDrawerCloseRefTakesAFocusCommand(t *testing.T) {
	ctx := core.NewContext()
	var close *core.Node
	pass := func() {
		ctx.BeginRenderPass()
		// Reset rewinds the hook cursor, so UseFocusRef returns the same
		// ref on the second pass.
		ctx.Reset()
		ref := core.UseFocusRef(ctx)
		n := sampleDrawer(true, nil, func() {})
		n.CloseRef = ref
		_, _, panel, _ := drawerParts(t, n.Render(ctx))
		close = buttonsOf(panel)[0]
		ctx.EndRenderPass()
		if close.Props["focusAction"] == nil {
			core.Focus(ref)
		}
	}
	pass()
	pass()
	if close.Props["focusAction"] != "focus" {
		t.Errorf("✕ focusAction = %v, want \"focus\" after core.Focus(CloseRef)", close.Props["focusAction"])
	}
}

// Android's system back closes an open drawer. The handler rides the panel
// layer, so it is present exactly while the panel is, and absent when there is
// no OnDismiss to call.
func TestDrawerClaimsSystemBackOnlyWhileOpen(t *testing.T) {
	dismissed := 0
	ctx, n := renderDebug(t, sampleDrawer(true, nil, func() { dismissed++ }))
	_, layer, _, _ := drawerParts(t, n)
	id, ok := layer.Props["onBack"].(string)
	if !ok || id == "" {
		t.Fatalf("an open drawer's panel layer must claim back, props = %v", layer.Props)
	}
	ctx.TriggerCallback(id)
	if dismissed != 1 {
		t.Errorf("system back ran OnDismiss %d times, want 1", dismissed)
	}

	_, n = renderDebug(t, sampleDrawer(false, nil, func() {}))
	if _, layer, _, _ := drawerParts(t, n); layer.Props["onBack"] != nil {
		t.Error("a shut drawer must not claim back")
	}

	_, n = renderDebug(t, sampleDrawer(true, nil, nil))
	if _, layer, _, _ := drawerParts(t, n); layer.Props["onBack"] != nil {
		t.Error("with no OnDismiss there is nothing to call, so back must fall through")
	}
}
