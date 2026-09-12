package comps

import (
	"reflect"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// renderScreen renders s on a fresh context under the given theme and returns
// the SafeArea root. Screens carry no hooks, so a fresh context per call is
// the cheapest way to keep tests independent.
func renderScreen(t *testing.T, theme *core.Theme, s Screen) *core.Node {
	t.Helper()
	ctx := core.NewContext().WithTheme(theme)
	ctx.BeginRenderPass()
	return s.Render(ctx)
}

// column returns the screen's content column, located by type rather than by
// walking a fixed number of children — the Scroll wrapper changes the depth,
// and that is exactly what the Scroll tests below vary.
func column(t *testing.T, root *core.Node) *core.Node {
	t.Helper()
	c := findFirst(root, func(n *core.Node) bool { return n.Type == "Column" })
	if c == nil {
		t.Fatal("no Column in the rendered screen")
	}
	return c
}

// The load-bearing property, the same one Button's zero value has: with no
// fields set the widget contributes no style props at all, so its column is
// the theme's Column base byte for byte. Every migration in this session
// depends on it — a scaffold that quietly imposed a gap or a flex-grow would
// have shifted the layout of four apps that never asked for one.
func TestScreenZeroValueIsExactlySafeAreaColumn(t *testing.T) {
	for name, theme := range core.BundledThemes() {
		t.Run(name, func(t *testing.T) {
			body := core.Text("hello")

			ctx := core.NewContext().WithTheme(theme)
			ctx.BeginRenderPass()
			want := core.SafeArea(core.Column(body)).Render(ctx)

			got := renderScreen(t, theme, Screen{Children: []core.View{body}})

			// reflect.DeepEqual on the whole node: Style is not comparable
			// with == (it carries a PseudoStates map), and comparing the
			// trees rather than just the styles also pins the shape —
			// SafeArea > Column > Text, no extra wrapper.
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("zero-value Screen differs from SafeArea(Column(...)):\n got %s\nwant %s",
					describe(got), describe(want))
			}
		})
	}
}

// The previous test compares against a hand-built tree under the bundled
// themes, whose Column base happens to set no gap. That makes it blind to an
// implementation that always applies core.Gap(s.Gap): Gap(0) writes a zero
// over a zero, so the trees still match. This renders under a theme whose
// Column base carries a non-zero gap, where "apply nothing" and "apply zero"
// finally disagree.
func TestScreenZeroGapDoesNotOverwriteTheThemeColumnGap(t *testing.T) {
	theme := &core.Theme{
		Colors:     core.DefaultTheme.Colors,
		Typography: core.DefaultTheme.Typography,
		Spacing:    core.DefaultTheme.Spacing,
		Components: core.ComponentDefaults{
			Column: core.Style{Gap: 7},
		},
	}

	got := renderScreen(t, theme, Screen{Children: []core.View{core.Text("hi")}})
	if g := column(t, got).Style.Gap; g != 7 {
		t.Fatalf("zero Gap overwrote the theme's Column gap: got %v, want 7", g)
	}

	// And an explicit Gap still wins over the theme base, or the field would
	// be inert under any theme that sets one.
	got = renderScreen(t, theme, Screen{Gap: 12, Children: []core.View{core.Text("hi")}})
	if g := column(t, got).Style.Gap; g != 12 {
		t.Fatalf("explicit Gap = %v, want 12", g)
	}
}

func TestScreenFillSetsFlexGrowOnTheColumn(t *testing.T) {
	off := renderScreen(t, core.DefaultTheme, Screen{Children: []core.View{core.Text("hi")}})
	if g := column(t, off).Style.FlexGrow; g != 0 {
		t.Fatalf("Fill unset still set FlexGrow = %v, want 0", g)
	}

	on := renderScreen(t, core.DefaultTheme, Screen{Fill: true, Children: []core.View{core.Text("hi")}})
	if g := column(t, on).Style.FlexGrow; g != 1 {
		t.Fatalf("Fill = true gave FlexGrow = %v, want 1", g)
	}
}

// Scroll changes the tree's shape, not just a style prop, so both directions
// are pinned: the wrapper exists and sits between SafeArea and the Column when
// set, and there is no Scroll node anywhere when it is not.
func TestScreenScrollWrapsTheColumn(t *testing.T) {
	off := renderScreen(t, core.DefaultTheme, Screen{Children: []core.View{core.Text("hi")}})
	if n := findFirst(off, func(n *core.Node) bool { return n.Type == "Scroll" }); n != nil {
		t.Fatal("Scroll unset still produced a Scroll node")
	}
	if off.Type != "SafeArea" || len(off.Children) != 1 || off.Children[0].Type != "Column" {
		t.Fatalf("want SafeArea > Column, got %s", describe(off))
	}

	on := renderScreen(t, core.DefaultTheme, Screen{Scroll: true, Children: []core.View{core.Text("hi")}})
	if on.Type != "SafeArea" || len(on.Children) != 1 || on.Children[0].Type != "Scroll" {
		t.Fatalf("want SafeArea > Scroll, got %s", describe(on))
	}
	scroll := on.Children[0]
	if len(scroll.Children) != 1 || scroll.Children[0].Type != "Column" {
		t.Fatalf("want Scroll > Column, got %s", describe(on))
	}
}

// Children flows into core.Column's argument list, which skips nil items.
// This is the contract behind the conditional-region idiom in the doc comment,
// and the one todoapp's footer already relies on for a slot. What it pins is
// that Screen passes children through untouched: wrapping each one — in a
// Fragment, a Box, anything — would turn the absent child back into a real
// node and reopen the stray-slot bug MaybeProp exists to close. (Filtering the
// nils out first would also pass, and that is fine: it is the same tree. The
// failure this guards against is a node appearing, not how it fails to.)
func TestScreenSkipsNilChildren(t *testing.T) {
	var absent core.View

	root := renderScreen(t, core.DefaultTheme, Screen{
		Children: []core.View{core.Text("first"), absent, core.Text("second")},
	})

	col := column(t, root)
	if len(col.Children) != 2 {
		t.Fatalf("nil child left a node: %d children, want 2 — %s", len(col.Children), describe(root))
	}
	// Order must survive the skip, not just the count.
	if col.Children[0].Props["content"] != "first" || col.Children[1].Props["content"] != "second" {
		t.Fatalf("children out of order after the skip: %s", describe(root))
	}
}

// Style is appended after Gap and Fill so a caller can override either. If the
// order were reversed the widget's own props would silently win and the escape
// hatch would not be one.
func TestScreenStyleOverridesTheWidgetsOwnProps(t *testing.T) {
	root := renderScreen(t, core.DefaultTheme, Screen{
		Gap:      12,
		Fill:     true,
		Style:    []core.StyleProp{core.Gap(3), core.FlexGrow(2), core.Padding(16)},
		Children: []core.View{core.Text("hi")},
	})

	st := column(t, root).Style
	if st.Gap != 3 {
		t.Errorf("Style Gap did not win: got %v, want 3", st.Gap)
	}
	if st.FlexGrow != 2 {
		t.Errorf("Style FlexGrow did not win: got %v, want 2", st.FlexGrow)
	}
	if st.Padding.Top != 16 {
		t.Errorf("Style Padding not applied: got %+v", st.Padding)
	}
}

// A screen with no children is still a screen: it must render the scaffold
// rather than panic on the empty slice, since a data-driven screen legitimately
// has nothing to show on its first pass.
func TestScreenWithNoChildrenRendersTheScaffold(t *testing.T) {
	root := renderScreen(t, core.DefaultTheme, Screen{Gap: 8})
	col := column(t, root)
	if len(col.Children) != 0 {
		t.Fatalf("empty Screen produced %d children", len(col.Children))
	}
	if col.Style.Gap != 8 {
		t.Fatalf("props still apply with no children: Gap = %v, want 8", col.Style.Gap)
	}
}

// describe renders a node tree as a compact indented type list for failure
// messages. The scaffold's bugs are shape bugs (a missing wrapper, an extra
// child), and a %+v of a *Node is unreadable at that depth.
func describe(n *core.Node) string {
	var walk func(*core.Node, string) string
	walk = func(n *core.Node, indent string) string {
		if n == nil {
			return indent + "<nil>\n"
		}
		s := indent + n.Type
		if c, ok := n.Props["content"]; ok {
			s += " " + toString(c)
		}
		s += "\n"
		for _, c := range n.Children {
			s += walk(c, indent+"  ")
		}
		return s
	}
	return "\n" + walk(n, "  ")
}

func toString(v any) string {
	if s, ok := v.(string); ok {
		return `"` + s + `"`
	}
	return ""
}

// KeyboardAware rides the Scroll wrapper when there is one.
func TestScreenKeyboardAwareMarksTheScrollRegion(t *testing.T) {
	on := renderScreen(t, core.DefaultTheme, Screen{
		Scroll:        true,
		KeyboardAware: true,
		Children:      []core.View{core.Text("hi")},
	})
	scroll := findFirst(on, func(n *core.Node) bool { return n.Type == "Scroll" })
	if scroll == nil {
		t.Fatal("no Scroll node to carry the prop")
	}
	if scroll.Props["keyboardAware"] != true {
		t.Errorf("Scroll props = %#v, want keyboardAware:true", scroll.Props)
	}

	// Unset, the wrapper is byte-identical to what it was before the field
	// existed — the MaybeProp false path leaves no trace in the tree.
	plain := renderScreen(t, core.DefaultTheme, Screen{Scroll: true, Children: []core.View{core.Text("hi")}})
	ps := findFirst(plain, func(n *core.Node) bool { return n.Type == "Scroll" })
	if _, present := ps.Props["keyboardAware"]; present {
		t.Errorf("a plain scrolling screen should carry no prop, got %#v", ps.Props)
	}
}

// With no Scroll to shorten, the column is what lifts — the docked-composer
// case. What must not happen is the prop landing on the SafeArea: that would
// pull the whole screen up by the keyboard's height, header included, and on
// Android it would consume the inset before any inner region could ask for it.
func TestScreenKeyboardAwareWithoutScrollMarksTheColumn(t *testing.T) {
	root := renderScreen(t, core.DefaultTheme, Screen{
		KeyboardAware: true,
		Children:      []core.View{core.Text("hi")},
	})

	if root.Type != "SafeArea" {
		t.Fatalf("want a SafeArea root, got %s", describe(root))
	}
	if _, present := root.Props["keyboardAware"]; present {
		t.Error("the safe area must not carry the prop")
	}
	col := column(t, root)
	if col.Props["keyboardAware"] != true {
		t.Errorf("content column props = %#v, want keyboardAware:true", col.Props)
	}
}

// And with a Scroll, the column must *not* also carry it: two nested nodes
// both insetting would take the keyboard's height out of the layout twice on
// any renderer that does not consume insets as Compose does.
func TestScreenKeyboardAwareLandsOnExactlyOneNode(t *testing.T) {
	root := renderScreen(t, core.DefaultTheme, Screen{
		Scroll:        true,
		KeyboardAware: true,
		Children:      []core.View{core.Text("hi")},
	})

	marked := 0
	var walk func(*core.Node)
	walk = func(n *core.Node) {
		if n == nil {
			return
		}
		if n.Props["keyboardAware"] == true {
			marked++
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)
	if marked != 1 {
		t.Errorf("%d nodes carry keyboardAware, want exactly 1", marked)
	}
}

// The background travels to the safe area and nothing else does. A screen
// that paints its column dark but leaves the inset box alone shows a light
// strip under the status bar on both natives; the scaffold closes that by
// painting the same colour behind the bars. Padding must stay on the column
// only, or the screen would be inset twice.
func TestScreenPaintsItsBackgroundOnTheSafeArea(t *testing.T) {
	root := renderScreen(t, core.DefaultTheme, Screen{
		Style:    []core.StyleProp{core.Padding(20), core.BackgroundColor("#101010")},
		Children: []core.View{core.Text("x")},
	})
	if root.Type != "SafeArea" || root.Style == nil {
		t.Fatalf("want a styled SafeArea root, got %s", describe(root))
	}
	if root.Style.Background != "#101010" {
		t.Errorf("SafeArea background = %q, want the screen's", root.Style.Background)
	}
	if root.Style.Padding != (core.EdgeInsets{}) {
		t.Errorf("SafeArea took the column's padding: %+v", root.Style.Padding)
	}
	if col := column(t, root); col.Style.Background != "#101010" || col.Style.Padding.Top != 20 {
		t.Errorf("column lost its own style: %+v", col.Style)
	}

	// With no background set, the safe area carries no style at all — the
	// zero-value equivalence TestScreenZeroValueIsExactlySafeAreaColumn pins
	// must survive a screen that sets only padding.
	plain := renderScreen(t, core.DefaultTheme, Screen{
		Style:    []core.StyleProp{core.Padding(20)},
		Children: []core.View{core.Text("x")},
	})
	if plain.Style != nil && !reflect.DeepEqual(*plain.Style, core.Style{}) {
		t.Errorf("a screen without a background styled its SafeArea: %+v", plain.Style)
	}
}

// --- the page-child rule ---------------------------------------------------
//
// core.List and core.Column are both built on the theme's Components.Column,
// so a screen whose whole content is a List used to be inset twice — 16 points
// of horizontal padding from the scaffold and 16 more from the list, with
// neither written anywhere a reader could see it. The tests below pin the rule
// that unstacks them and, just as importantly, the four cases it must not fire
// on.

// themeInset is the horizontal inset the bundled themes put on a Column, read
// from the theme rather than written as 16 so these tests keep testing the
// rule and not the palette.
func themeInset(theme *core.Theme) core.EdgeInsets { return theme.Components.Column.Padding }

func TestScreenDropsItsInsetWhenTheOnlyChildIsAList(t *testing.T) {
	for name, theme := range core.BundledThemes() {
		t.Run(name, func(t *testing.T) {
			root := renderScreen(t, theme, Screen{
				Children: []core.View{core.List(core.Text("row"))},
			})

			col := column(t, root)
			if col.Style.Padding != (core.EdgeInsets{}) {
				t.Errorf("the scaffold's column kept its inset around a List: %+v", col.Style.Padding)
			}
			// The list keeps its own, or the page would now be inset zero
			// times instead of twice — the opposite error and just as wrong.
			list := findFirst(root, func(n *core.Node) bool { return n.Type == "List" })
			if list == nil {
				t.Fatalf("no List in the rendered screen: %s", describe(root))
			}
			if list.Style.Padding != themeInset(theme) {
				t.Errorf("List inset = %+v, want the theme's %+v", list.Style.Padding, themeInset(theme))
			}
		})
	}
}

// The count is taken after nil entries are skipped, which is what keeps the
// conditional-slot idiom compatible with the rule: a screen holding an absent
// banner and a list is a single-child screen.
func TestScreenPageChildRuleCountsAfterNilChildren(t *testing.T) {
	var absent core.View

	root := renderScreen(t, core.DefaultTheme, Screen{
		Children: []core.View{absent, core.List(core.Text("row")), absent},
	})

	col := column(t, root)
	if len(col.Children) != 1 {
		t.Fatalf("nil children left nodes behind: %s", describe(root))
	}
	if col.Style.Padding != (core.EdgeInsets{}) {
		t.Errorf("a nil sibling defeated the rule: column inset = %+v", col.Style.Padding)
	}
}

// The rule is decided on the rendered child, which is why a widget in this
// package that *becomes* a List is treated as one. Nothing spells core.List at
// this call site, and a rule written against the Go value would have missed it
// — GroupedList is the widget most screens reach for when the whole screen is
// the list, so missing it would have missed the common case.
func TestScreenPageChildRuleSeesThroughAWidget(t *testing.T) {
	root := renderScreen(t, core.DefaultTheme, Screen{
		Children: []core.View{GroupedList[string]{
			Items: []string{"a", "b"},
			Key:   func(s string) string { return s },
			Row:   func(s string) core.View { return core.Text(s) },
		}},
	})

	col := column(t, root)
	if col.Style.Padding != (core.EdgeInsets{}) {
		t.Errorf("a GroupedList page was inset twice: column inset = %+v", col.Style.Padding)
	}
	if len(col.Children) != 1 || col.Children[0].Type != "List" {
		t.Fatalf("GroupedList no longer renders as a List: %s", describe(root))
	}
}

// A List that is one of several children is a block within the page, not the
// page: the column's inset is what puts the other children where they belong,
// and dropping it would slide a header and a footer to the screen edge to fix
// a doubling that only the list has.
func TestScreenKeepsItsInsetWhenTheListHasSiblings(t *testing.T) {
	root := renderScreen(t, core.DefaultTheme, Screen{
		Children: []core.View{core.Text("header"), core.List(core.Text("row"))},
	})

	if p := column(t, root).Style.Padding; p != themeInset(core.DefaultTheme) {
		t.Errorf("column inset = %+v, want the theme's %+v — the rule fired on a multi-child screen",
			p, themeInset(core.DefaultTheme))
	}
}

// Everything that is not a scrolling, self-insetting page leaves the scaffold
// alone. core.Scroll is the case worth naming: it scrolls but carries no theme
// base, so its content is inset once — by this column — and stripping that
// would move the page rather than unstack it.
func TestScreenKeepsItsInsetForNonPageChildren(t *testing.T) {
	cases := map[string]core.View{
		"a text":     core.Text("hi"),
		"a column":   core.Column(core.Text("hi")),
		"a scroll":   core.Scroll(core.Text("hi")),
		"a card":     core.Card(core.Text("hi")),
		"a box":      core.Box(core.Text("hi")),
		"a list two": core.Column(core.List(core.Text("row"))), // a List, but not the child
	}
	for name, child := range cases {
		t.Run(name, func(t *testing.T) {
			root := renderScreen(t, core.DefaultTheme, Screen{Children: []core.View{child}})
			if p := column(t, root).Style.Padding; p != themeInset(core.DefaultTheme) {
				t.Errorf("column inset = %+v, want the theme's %+v", p, themeInset(core.DefaultTheme))
			}
		})
	}
}

// The cleared padding is applied ahead of the caller's Style, so the escape
// hatch still opens in both directions: a screen can ask for its own inset
// around a list, and a screen that wants the old doubled behavior can spell it.
func TestScreenStyleStillWinsOverThePageChildRule(t *testing.T) {
	root := renderScreen(t, core.DefaultTheme, Screen{
		Style:    []core.StyleProp{core.Padding(24)},
		Children: []core.View{core.List(core.Text("row"))},
	})

	if p := column(t, root).Style.Padding; p.Left != 24 || p.Top != 24 {
		t.Errorf("column inset = %+v, want the caller's 24 on every side", p)
	}
}

// Nothing but the inset is dropped. The rule is about one duplicated property,
// and a screen that also sets a gap, a fill or a background must keep all
// three — the column is still the flex parent the List grows inside.
func TestScreenPageChildRuleDropsOnlyTheInset(t *testing.T) {
	root := renderScreen(t, core.DefaultTheme, Screen{
		Fill:     true,
		Gap:      9,
		Style:    []core.StyleProp{core.BackgroundColor("#202020")},
		Children: []core.View{core.List(core.Text("row"))},
	})

	st := column(t, root).Style
	if st.Padding != (core.EdgeInsets{}) {
		t.Errorf("inset survived: %+v", st.Padding)
	}
	if st.FlexGrow != 1 || st.Gap != 9 || st.Background != "#202020" {
		t.Errorf("the rule took more than the inset: %+v", st)
	}
}

// The pre-render above is only safe because Screen's own props register no
// callbacks: every one of them is a style prop or KeyboardAware, which writes a
// flag and asks nothing of the context. containerNode's ordering contract says
// a container's callback IDs precede its children's, and a Screen that started
// registering one would break that contract silently — the IDs would still be
// assigned, in the other order, and the only symptom would be a handler wired
// to the wrong node after a diff.
//
// So the assumption is checked rather than commented: the sole child's handler
// takes the first ID of the pass. A prop added to Screen that registers
// anything moves it, and this fails pointing here.
func TestScreenRegistersNoCallbacksOfItsOwn(t *testing.T) {
	ctx := core.NewContext().WithTheme(core.DefaultTheme)
	ctx.BeginRenderPass()

	screen := Screen{
		Fill:          true,
		Gap:           8,
		KeyboardAware: true,
		Style:         []core.StyleProp{core.BackgroundColor("#fff")},
		Children:      []core.View{core.Box(core.OnClick(func() {}), core.Text("tap"))},
	}
	root := screen.Render(ctx)

	box := findFirst(root, func(n *core.Node) bool { return n.Type == "Box" })
	if box == nil {
		t.Fatalf("no Box in the rendered screen: %s", describe(root))
	}
	// cb_0 is the first void callback a fresh registry hands out.
	if got := box.Props["onClick"]; got != "cb_0" {
		t.Errorf("the sole child's handler is %v, want cb_0 — something in Screen "+
			"registered a callback before its child rendered, which the "+
			"page-child pre-render in Screen.Render assumes nothing does", got)
	}
}

// Screen renders its sole child itself to classify it, so the one thing that
// could go wrong is rendering it twice — which would register every callback
// in the page a second time and leave the first set pointing at a tree the
// reconciler has already thrown away. A child that counts its own renders is
// the only witness, since the duplicate node would be discarded and the tree
// would look correct.
func TestScreenRendersItsSoleChildExactlyOnce(t *testing.T) {
	for name, child := range map[string]func(int) core.View{
		"a list": func(n int) core.View { return core.List(core.Text("row")) },
		"a text": func(n int) core.View { return core.Text("hi") },
	} {
		t.Run(name, func(t *testing.T) {
			renders := 0
			counted := core.ComponentFunc(func(ctx *core.Context) *core.Node {
				renders++
				return child(renders).Render(ctx)
			})

			renderScreen(t, core.DefaultTheme, Screen{Children: []core.View{counted}})
			if renders != 1 {
				t.Errorf("the sole child rendered %d times, want 1", renders)
			}
		})
	}
}

// --- Footer ------------------------------------------------------------------

func TestScreenFooterIsPinnedBelowAGrowingColumn(t *testing.T) {
	_, n := renderDebug(t, Screen{
		Children: []core.View{core.Text("body")},
		Footer:   core.Text("footer"),
	})
	if n.Type != "SafeArea" || len(n.Children) != 2 {
		t.Fatalf("want SafeArea(content, footer), got %q with %d children", n.Type, len(n.Children))
	}
	content, footer := n.Children[0], n.Children[1]
	if content.Type != "Column" || content.Style.FlexGrow != 1 {
		t.Errorf("content = %q grow=%v; a footer needs the column to take the leftover height",
			content.Type, content.Style.FlexGrow)
	}
	if footer.Props["content"] != "footer" {
		t.Errorf("footer must render last, got %v", footer.Props)
	}
}

func TestScreenFooterWithScrollGrowsTheViewportNotTheColumn(t *testing.T) {
	_, n := renderDebug(t, Screen{
		Scroll:   true,
		Children: []core.View{core.Text("body")},
		Footer:   core.Text("footer"),
	})
	scroll := n.Children[0]
	if scroll.Type != "Scroll" || scroll.Style == nil || scroll.Style.FlexGrow != 1 {
		t.Fatalf("the Scroll must grow so the footer stays outside it, got %q %+v", scroll.Type, scroll.Style)
	}
	if col := scroll.Children[0]; col.Style != nil && col.Style.FlexGrow != 0 {
		t.Errorf("the scrolled column must not be forced to grow, got %v", col.Style.FlexGrow)
	}
	if findText(scroll, "footer") != nil {
		t.Error("the footer must not be inside the scroll region")
	}
}

func TestScreenWithoutFooterKeepsItsSingleChild(t *testing.T) {
	_, n := renderDebug(t, Screen{Scroll: true, Children: []core.View{core.Text("body")}})
	if len(n.Children) != 1 {
		t.Fatalf("children = %d, want 1", len(n.Children))
	}
	if s := n.Children[0].Style; s != nil && s.FlexGrow != 0 {
		t.Errorf("no footer means no FlexGrow on the Scroll, got %v", s.FlexGrow)
	}
}
