package tutorial

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// The colour scheme (theme.go). Delivered the way wasm/main.go delivers every
// host event, as the split tests do, so these stand where the page stands.

// setScheme sends what the page sends at boot, on its header switch, and when
// the OS flips while the reader is on System.
func setScheme(scheme string) {
	core.ReceiveHostEvent(themeEvent, map[string]any{"scheme": scheme})
}

// The captions' ink is the cheapest witness of which palette a tree was drawn
// in: every screen has captions, and the two themes' TextSecondary differ.
var (
	lightCaption = core.DefaultTheme.Colors.TextSecondary
	darkCaption  = darkTheme.Colors.TextSecondary
)

// Without a theme event nothing changes: DefaultTheme on every host, and not
// one darkTheme colour anywhere in the tree.
func TestLightSchemeIsTheDefault(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Hello, GrMob")
	wire := mgr.RenderInitial()
	if !strings.Contains(wire, lightCaption) {
		t.Fatalf("expected DefaultTheme's caption ink %s in the light tree", lightCaption)
	}
	if strings.Contains(wire, darkCaption) {
		t.Fatalf("darkTheme's caption ink %s leaked into the light tree", darkCaption)
	}
	assertNoConcerns(t)
}

// "dark" repaints the whole tree, and "light" puts it back byte for byte: the
// scheme is a context swap, not a wrapper node, so the light tree is exactly
// the tree the app drew before theme.go existed.
func TestDarkSchemeRepaintsAndLightRestoresTheTree(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Hello, GrMob")
	before := mgr.RenderInitial()

	setScheme("dark")
	dark := mgr.RenderInitial()
	if !strings.Contains(dark, darkCaption) {
		t.Fatalf("expected darkTheme's caption ink %s after \"dark\"", darkCaption)
	}
	if strings.Contains(dark, lightCaption) {
		t.Fatalf("DefaultTheme's caption ink %s survived \"dark\"", lightCaption)
	}

	setScheme("light")
	if after := mgr.RenderInitial(); after != before {
		t.Fatal("dark then light should restore the light tree exactly")
	}
	assertNoConcerns(t)
}

// Both panes of the split are drawn in the scheme: withScheme sits above the
// layout, so the guide and the demos lifted onto the phone share one palette.
func TestDarkSchemeReachesBothPanes(t *testing.T) {
	mgr := newSplitApp(t)
	openLesson(t, mgr, "Hello, GrMob")
	setScheme("dark")
	guide, phone := panes(t, tree(t, mgr))
	for name, pane := range map[string]*node{"guide": guide, "phone": phone} {
		dim := findNode(pane, func(n *node) bool {
			return n.Style != nil && n.Style.TextColor == darkCaption
		})
		if dim == nil {
			t.Errorf("the %s pane has no text in darkTheme's caption ink", name)
		}
		stale := findNode(pane, func(n *node) bool {
			return n.Style != nil && n.Style.TextColor == lightCaption
		})
		if stale != nil {
			t.Errorf("the %s pane still has text in DefaultTheme's caption ink", name)
		}
	}
	assertNoConcerns(t)
}

// A demo's state survives the switch: the scheme changes colours, not hook
// slots (theme.go, "How the palette is swapped").
func TestSchemeSwitchKeepsDemoState(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Hello, GrMob")

	toggleCheckbox(t, mgr, 2, true) // Bio on, in the light scheme
	setScheme("dark")
	if !hasTextContaining(tree(t, mgr), "without leaving Go") {
		t.Fatal("the bio turned on before the switch should still be on after it")
	}
	toggleCheckbox(t, mgr, 2, false) // and off, in the dark scheme
	setScheme("light")
	if hasTextContaining(tree(t, mgr), "without leaving Go") {
		t.Fatal("the bio turned off in the dark scheme should still be off in the light one")
	}
	assertNoConcerns(t)
}

// The page sends the scheme before RenderInitial, so a dark boot's first frame
// is dark; and, as with the layout, a scheme is the boot value only of the app
// it was sent to.
func TestASchemeSentBeforeTheFirstRenderIsTheFirstFrame(t *testing.T) {
	setScheme("dark")
	mgr := newApp(t)
	if !strings.Contains(mgr.RenderInitial(), darkCaption) {
		t.Fatal("a scheme sent before the first render should be the first frame's")
	}
	mgr.Close()

	next := newApp(t)
	if strings.Contains(next.RenderInitial(), darkCaption) {
		t.Fatal("the closed app's scheme leaked into the next app's boot")
	}
	setScheme("dark")
	if !strings.Contains(next.RenderInitial(), darkCaption) {
		t.Fatal("a scheme sent to a live tree should switch it")
	}
	if bootDark() {
		t.Fatal("a scheme sent while a tree listens must not become a boot value")
	}
}

// darkTheme is built from a copy of DefaultTheme; building it must not have
// written through to the original.
func TestDarkThemeLeavesDefaultThemeAlone(t *testing.T) {
	d := core.DefaultTheme
	if d.Colors.Background != "#FFFFFF" || d.Colors.TextPrimary != "#000000" ||
		d.Components.Button.TextColor != "#FFFFFF" || d.Components.Card.Background != "#FFFFFF" ||
		d.Typography.Body.TextColor != "#000000" {
		t.Fatal("newDarkTheme wrote through to core.DefaultTheme")
	}
	if darkTheme.Colors.Background == d.Colors.Background {
		t.Fatal("darkTheme should not share DefaultTheme's page colour")
	}
}
