package components

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/htmlout"
)

// pushedContext returns a context whose navigation stack has depth 2, which
// is the state core.CanPop reports true for — a pushed screen, the only place
// the automatic back control should appear. The initial route has to be
// rendered before the push: the Navigator installs its root lazily on the
// first pass, so pushing onto an unrendered stack leaves depth at 1.
func pushedContext(t *testing.T) *core.Context {
	t.Helper()
	ctx := core.NewContext()
	app := core.Navigator(func(*core.Context) core.View { return core.Text("home") })
	core.Render(ctx, app)
	core.Push(ctx, func(*core.Context) core.View { return core.Text("detail") })
	if !core.CanPop(ctx) {
		t.Fatal("test setup: CanPop should be true after a push")
	}
	ctx.BeginRenderPass()
	return ctx
}

// findButton finds the first Button node whose label is s.
func findButton(n *core.Node, s string) *core.Node {
	return findFirst(n, func(n *core.Node) bool {
		return n.Type == "Button" && n.Props["label"] == s
	})
}

func TestAppBarZeroValueIsATitleAndARule(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := AppBar{Title: "Sermons"}.Render(ctx)

	if findText(n, "Sermons") == nil {
		t.Error("the title should be rendered")
	}
	// No navigation stack, so nothing to go back to and no arrow drawn.
	if findButton(n, "‹") != nil {
		t.Error("a bar on a screen that cannot pop must not draw a back control")
	}
	// The rule is the bar's second child; it is what separates an unstyled bar
	// from the content under it.
	if len(n.Children) != 2 {
		t.Fatalf("bar should be a row plus its separator, got %d children", len(n.Children))
	}
	rule := n.Children[1]
	if rule.Style.Background != core.DefaultTheme.Colors.BorderColor() {
		t.Errorf("separator background = %q, want the theme Border role", rule.Style.Background)
	}
}

func TestAppBarBackAppearsOnlyWhenThereIsSomewhereToGo(t *testing.T) {
	ctx := pushedContext(t)
	n := AppBar{Title: "Sermon"}.Render(ctx)

	back := findButton(n, "‹")
	if back == nil {
		t.Fatal("a pushed screen should get an automatic back control")
	}
	// A screen reader announcing "‹" announces nothing.
	if back.Style.AccessibilityLabel != "Back" {
		t.Errorf("back control label = %q, want %q", back.Style.AccessibilityLabel, "Back")
	}

	id, ok := back.Props["onClick"].(string)
	if !ok {
		t.Fatal("the back control should register a handler")
	}
	ctx.TriggerCallback(id)
	if core.CanPop(ctx) {
		t.Error("tapping back should have popped the stack")
	}
}

func TestAppBarHideBackAndOnBack(t *testing.T) {
	ctx := pushedContext(t)
	if n := (AppBar{Title: "Step 2", HideBack: true}).Render(ctx); findButton(n, "‹") != nil {
		t.Error("HideBack should suppress the automatic control even when CanPop is true")
	}

	asked := false
	n := AppBar{Title: "Step 2", OnBack: func() { asked = true }}.Render(ctx)
	back := findButton(n, "‹")
	if back == nil {
		t.Fatal("OnBack should not remove the control, only change what it does")
	}
	ctx.TriggerCallback(back.Props["onClick"].(string))
	if !asked {
		t.Error("OnBack should replace core.Pop")
	}
	// And it must replace it, not run alongside it: a confirm-before-leaving
	// handler that popped anyway would be useless.
	if !core.CanPop(ctx) {
		t.Error("OnBack ran the default Pop as well")
	}
}

func TestAppBarLeadingSlotWinsOverTheBackControl(t *testing.T) {
	ctx := pushedContext(t)
	n := AppBar{Title: "Compose", Leading: Button{Label: "✕", OnTap: func() {}}}.Render(ctx)

	if findButton(n, "✕") == nil {
		t.Error("Leading should be rendered")
	}
	if findButton(n, "‹") != nil {
		t.Error("Leading should replace the automatic back control, not sit beside it")
	}
}

func TestAppBarBackGlyphIsOverridable(t *testing.T) {
	ctx := pushedContext(t)
	n := AppBar{Title: "Sermon", BackGlyph: "←"}.Render(ctx)

	if findButton(n, "←") == nil {
		t.Error("BackGlyph should replace the default chevron")
	}
	if findButton(n, "‹") != nil {
		t.Error("the default chevron should be gone")
	}
}

func TestAppBarActionsArePinnedTrailing(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := AppBar{
		Title: "Sermons",
		// A nil entry stands for a conditional action; it must cost no node
		// rather than force the caller to filter the slice.
		Actions: []core.View{nil, Button{Label: "↻", OnTap: func() {}}},
	}.Render(ctx)

	row := n.Children[0]
	last := row.Children[len(row.Children)-1]
	if last.Props["label"] != "↻" {
		t.Errorf("the action should be the row's last child, got %q", last.Type)
	}
	// The pinning is done by the growing middle, not by a justify rule; that
	// is what keeps leading and trailing hard against the edges in every
	// configuration (see ListRow for the same choice).
	middle := row.Children[len(row.Children)-2]
	if middle.Style.FlexGrow != 1 {
		t.Errorf("the middle should take the slack, FlexGrow = %v", middle.Style.FlexGrow)
	}
}

// The middle is structure, not content: it has to be there even when there is
// nothing in it, or the pinning above becomes conditional.
func TestAppBarMiddleIsRenderedEvenWhenEmpty(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := AppBar{Actions: []core.View{Button{Label: "↻", OnTap: func() {}}}}.Render(ctx)

	row := n.Children[0]
	if len(row.Children) != 2 {
		t.Fatalf("row should hold the empty middle and the action, got %d children", len(row.Children))
	}
	if row.Children[0].Style.FlexGrow != 1 {
		t.Error("the growing middle should be rendered even with no title")
	}
}

func TestAppBarContentSlotReplacesTheTitle(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := AppBar{Title: "Sermons", Content: SearchField{Value: "grace"}}.Render(ctx)

	if findText(n, "Sermons") != nil {
		t.Error("Content should replace the Title stack, not sit above it")
	}
	if findFirst(n, func(n *core.Node) bool { return n.Type == "Input" }) == nil {
		t.Error("Content should be rendered")
	}
}

func TestAppBarSubtitleAndTitleRoles(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := AppBar{Title: "Sermons", Subtitle: "42 recordings"}.Render(ctx)

	theme := core.DefaultTheme
	title := findText(n, "Sermons")
	if title == nil {
		t.Fatal("no title")
	}
	// Subtitle's size — Typography.Title is the screen's large heading and
	// does not fit in a bar — but the primary ink and a bold weight, because a
	// bar title is the screen's name rather than a supporting line.
	if title.Style.FontSize != theme.Typography.Subtitle.FontSize {
		t.Errorf("title size = %v, want the theme's Subtitle size %v", title.Style.FontSize, theme.Typography.Subtitle.FontSize)
	}
	if title.Style.TextColor != theme.Colors.TextPrimary {
		t.Errorf("title ink = %q, want TextPrimary %q", title.Style.TextColor, theme.Colors.TextPrimary)
	}
	if title.Style.FontWeight != core.Bold {
		t.Errorf("title weight = %v, want Bold", title.Style.FontWeight)
	}

	sub := findText(n, "42 recordings")
	if sub == nil {
		t.Fatal("no subtitle")
	}
	if sub.Style.FontSize != theme.Typography.Caption.FontSize {
		t.Errorf("subtitle size = %v, want the Caption role", sub.Style.FontSize)
	}
}

func TestAppBarHideSeparator(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := AppBar{Title: "Sermons", HideSeparator: true}.Render(ctx)

	// With no rule there is no wrapper: the bar *is* the row, so a caller who
	// styles it is styling the node they see.
	if n.Type != "Row" {
		t.Errorf("a bar with no separator should be the row itself, got %q", n.Type)
	}
}

func TestAppBarStyleOverridesTheDefaults(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := AppBar{Title: "Sermons", Style: []core.StyleProp{
		core.BackgroundColor("#101010"),
	}}.Render(ctx)

	if got := n.Children[0].Style.Background; got != "#101010" {
		t.Errorf("bar background = %q, want the caller's override", got)
	}
}

func TestAppBarExportsHTML(t *testing.T) {
	ctx := pushedContext(t)
	n := AppBar{Title: "Sermon", Subtitle: "Mar 22", Actions: []core.View{
		Button{Label: "Share", OnTap: func() {}},
	}}.Render(ctx)

	html := htmlout.ExportHTML(n)
	for _, want := range []string{"Sermon", "Mar 22", "Share", "Back"} {
		if !strings.Contains(html, want) {
			t.Errorf("exported HTML should contain %q", want)
		}
	}
}

// The two roles the bar can honestly claim: it is the screen's banner, and
// its Title is the screen's heading. The heading is the one that means
// something on all four targets — SwiftUI and Compose both have a header
// notion and neither has landmarks — which is also why only the Title takes
// it, and not the Subtitle, which is a supporting line.
func TestAppBarNamesItselfAndItsTitle(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := AppBar{Title: "Sermons", Subtitle: "42 recordings"}.Render(ctx)

	bar := findFirst(n, func(n *core.Node) bool { return n.Type == "Row" })
	if bar == nil || bar.Style.AccessibilityRole != core.RoleBanner {
		t.Fatalf("the bar row should carry the banner landmark, got %+v", bar)
	}
	if title := findText(n, "Sermons"); title == nil || title.Style.AccessibilityRole != core.RoleHeading {
		t.Errorf("the title should be a heading, got %+v", title)
	}
	if sub := findText(n, "42 recordings"); sub == nil || sub.Style.AccessibilityRole != core.RoleNone {
		t.Errorf("the subtitle is a supporting line, not a second heading: %+v", sub)
	}
}

// The banner lands on the same element in both shapes. With the rule hidden
// the row is the outermost node; with it, a Box wraps the row and the rule —
// and the role stays on the row, because that is the element Style reaches and
// a role that moved with a cosmetic flag could not be overridden in one place.
func TestAppBarBannerSitsOnTheRowInBothShapes(t *testing.T) {
	for _, hide := range []bool{false, true} {
		ctx := core.NewContext()
		ctx.BeginRenderPass()
		n := AppBar{Title: "Sermons", HideSeparator: hide}.Render(ctx)

		row := findFirst(n, func(n *core.Node) bool { return n.Type == "Row" })
		if row == nil || row.Style.AccessibilityRole != core.RoleBanner {
			t.Errorf("HideSeparator=%v: the row lost the banner role", hide)
		}
		if !hide && n.Style.AccessibilityRole != core.RoleNone {
			t.Errorf("the separator Box should not be a second banner, got %q", n.Style.AccessibilityRole)
		}
	}
}
