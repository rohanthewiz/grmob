package tutorial

import (
	"fmt"
	"github.com/rohanthewiz/grmob/comps"
	"github.com/rohanthewiz/grmob/core"
)

// chapter1 — Views & Layout: the shape of a GrMob app before any state or
// events. Every demo here is still interactive (that is the tutorial's
// promise), so the demos quietly borrow core.NewState a chapter early; each
// lesson that does says so in a caption and defers the explanation to
// Chapter 2. The lesson *snippets* stay strictly on-topic.
//
// A note on the demos' hook discipline, since lessons are the one place in
// examples/ where many small stateful screens live side by side: each lesson
// Body runs inside its own Navigator frame (see lessonRoute), so its hooks
// occupy a private namespace, and every Body calls all of its hooks
// unconditionally at the top — the rules-of-hooks shape the tests verify
// under debug mode.
func chapter1() Chapter {
	return Chapter{
		Title:   "Views & Layout",
		Icon:    "🧱",
		Summary: "Views as plain Go values, and the flex layout system that arranges them.",
		Lessons: []Lesson{
			lessonHello(),
			lessonText(),
			lessonStacks(),
			lessonAlignment(),
			lessonSurfaces(),
		},
	}
}

// --- 1.1 -----------------------------------------------------------------

func lessonHello() Lesson {
	return Lesson{
		Title:   "Hello, GrMob",
		Summary: "Everything on screen is a View — a plain Go value you compose by calling functions.",
		Body: func(ctx *core.Context) core.View {
			// Composition switches for the live demo. State proper is
			// Chapter 2; here the toggles only exist to let the reader feel
			// "a view is a value you include or don't".
			showHeader := core.NewState(ctx, true)
			showStats := core.NewState(ctx, true)
			showBio := core.NewState(ctx, false)

			return core.Column(
				core.Gap(14),
				prose("A GrMob screen is not a template or a markup file — it is a Go value. "+
					"Anything implementing the one-method View interface can be put on screen, "+
					"and the usual way to make one is a plain function:"),
				codeBlock(`type View interface {
    Render(ctx *core.Context) *core.Node
}

func Profile(ctx *core.Context) core.View {
    return core.Column(
        Header(),          // views compose by
        Stats(),           // calling functions —
        core.Text("Bio"),  // no registration step
    )
}`),
				prose("Because views are values, composition is ordinary Go: call a function to "+
					"include its subtree, don't call it to leave it out. Toggle the pieces below and "+
					"watch the profile card recompose."),
				demoPanel("Choose which sub-views Profile() composes. (The checkboxes use state — Chapter 2's topic.)",
					// A Column, not the Row these three once shared: each is now a
					// full-width CheckboxRow, and three rows side by side do not fit a
					// phone. Gap 0 because a ListRow carries the theme's row padding.
					core.Column(
						core.Gap(0),
						checkRow("Header()", showHeader),
						checkRow("Stats()", showStats),
						checkRow("Bio text", showBio),
					),
					core.Card(
						core.Gap(8),
						core.If(showHeader.Get(), helloHeader()),
						core.If(showStats.Get(), helloStats()),
						core.If(showBio.Get(), prose("Go developer. Builds mobile apps without leaving Go.")),
					),
				),
				keyPoints(
					"Everything on screen implements core.View; core.ComponentFunc adapts a plain function.",
					"Composition is function calls — there is no registry, no template language.",
					"The tree you return is rendered natively on Android and iOS, and to HTML on the web.",
				),
			)
		},
	}
}

// checkRow is the demos' labelled checkbox: a comps.CheckboxRow bound to one
// bool state.
//
// It was a hand-rolled Row of a core.Checkbox and a caption, where only the
// box took a tap. CheckboxRow makes the whole row the target and names the
// box after its title, which is what a toggle row on a phone is expected to
// do, so every demo gets that from this one helper. The helper stays rather
// than the struct being written out at each of its ~30 call sites: a demo's
// point is the state it flips, and one line per toggle keeps that in view.
//
// OnToggle is s.Set, a setter, as CheckboxRow requires: on the web a tap on
// the box reaches Go twice (the row's click, then the input's change), and a
// setter turns the second report into a no-op where a flip would undo the
// first.
func checkRow(label string, s core.State[bool]) core.View {
	return comps.CheckboxRow{Title: label, Checked: s.Get(), OnToggle: s.Set}
}

// The profile card's pieces sit inside a Card, whose own 16px padding is the
// card's one inset. So every stack here states Padding(0): the theme's Row
// base (8/16) and Column base (12/16) apply to *every* plain stack, and
// inside the card they nest.
//
// Before the zeros, the stats row carried 16+16 of its own inline padding and
// each stat another 16+16. That was 128px of padding and 36px of gap before
// any text, against a card about 200px wide on the split layout's 376px glass.
// "Following" ran off the card's right edge. The header lost 32px the same
// way, which squeezed the avatar into an oval and wrapped the name.
func helloHeader() core.View {
	return core.Row(
		core.Padding(0),
		core.Gap(10),
		core.AlignItemsProp(core.AlignItemsCenter),
		// The avatar pins its own size (comps.Avatar states FlexShrink(0)),
		// so the text column is what gives up width, by wrapping, when the
		// card is narrow.
		comps.Avatar{Name: "Gopher McGrMob"},
		core.Column(
			core.Padding(0),
			core.Text("Gopher McGrMob", core.FontWeight(core.Bold)),
			caption("Wrote this profile in pure Go"),
		),
	)
}

func helloStats() core.View {
	stat := func(value, label string) core.View {
		return core.Column(
			core.Padding(0),
			core.Text(value, core.FontWeight(core.Bold)),
			caption(label),
		)
	}
	return core.Row(
		core.Padding(0),
		core.Gap(18),
		// Wrap rather than overflow. With the zeros above the three stats need
		// about 186px, which fits the narrowest card the tutorial draws with
		// little to spare. A larger text size, or a longer label, should move
		// the last stat onto a second line inside the card, not past its edge.
		core.FlexWrap(true),
		stat("128", "Posts"),
		stat("1.2k", "Followers"),
		stat("180", "Following"),
	)
}

// --- 1.2 -----------------------------------------------------------------

// inkRoles are the palette roles the color picker below cycles through.
// Roles, not hex strings: naming the role is the habit the whole framework
// is built around — swap the theme and every role re-resolves.
var inkRoles = []string{"Ink", "Primary", "Success", "Error"}

func lessonText() Lesson {
	return Lesson{
		Title:   "Text & typography",
		Summary: "Style props are functions; the theme supplies a type scale so screens agree on sizes.",
		Body: func(ctx *core.Context) core.View {
			size := core.NewState(ctx, 20)
			bold := core.NewState(ctx, true)
			colorIdx := core.NewState(ctx, 0)

			th := ctx.Theme()
			inkFor := []string{
				th.Colors.TextPrimary,
				th.Colors.Primary,
				th.Colors.SuccessColor(),
				th.Colors.Error,
			}

			weight := core.Normal
			if bold.Get() {
				weight = core.Bold
			}

			return core.Column(
				core.Gap(14),
				prose("core.Text takes the string and any number of style props — small functions "+
					"like FontSize and TextColor that each set one field of the node's Style:"),
				codeBlock(`core.Text("Read me",
    core.FontSize(20),
    core.FontWeight(core.Bold),
    core.TextColor(ctx.Theme().Colors.Primary),
)

// Or take a whole role from the theme's type scale:
core.Text("Read me", core.UseStyle(ctx.Theme().Typography.Title))`),
				prose("Prefer the theme's roles over literals: Typography.Title/Subtitle/Body/Caption "+
					"for sizes, and the color palette's named roles for ink. A screen written in roles "+
					"restyles itself when the theme changes."),
				demoPanel("Drive the preview's style props.",
					core.Text("The quick brown gopher jumps over the lazy bridge.",
						core.FontSize(float64(size.Get())),
						core.FontWeight(weight),
						core.TextColor(inkFor[colorIdx.Get()]),
					),
					stepper("FontSize", size, 12, 40, 2),
					checkRow("Bold", bold),
					comps.SegmentedControl{
						Style:     segWrap,
						Labels:    inkRoles,
						Selected:  colorIdx.Get(),
						OnSelect:  func(i int) { colorIdx.Set(i) },
						KeyPrefix: "ink-",
					},
				),
				keyPoints(
					"Style props are composable functions; later props win over earlier ones.",
					"core.UseStyle layers a whole Style value — set fields win, zero fields pass through.",
					"Reach for Typography and palette roles first; literal hex is the exception.",
				),
			)
		},
	}
}

// --- 1.3 -----------------------------------------------------------------

func lessonStacks() Lesson {
	return Lesson{
		Title:   "Rows, Columns & spacing",
		Summary: "The two flex stacks, Gap for rhythm, Spacer for one-off gaps.",
		Body: func(ctx *core.Context) core.View {
			axis := core.NewState(ctx, 0) // 0 = Row, 1 = Column
			gap := core.NewState(ctx, 8)

			boxes := []core.PropsAndChildren{
				core.Gap(float64(gap.Get())),
				demoBox("A", boxBlue, 0),
				demoBox("B", boxTeal, 0),
				demoBox("C", boxPlum, 0),
			}

			return core.Column(
				core.Gap(14),
				prose("Row and Column are the layout workhorses: flex stacks taking a mixed "+
					"argument list of style props and children, in any order."),
				codeBlock(`core.Row(
    core.Gap(8),          // uniform spacing between children
    core.Text("left"),
    core.Text("right"),
)

core.Column(
    header,
    core.Spacer(16),      // a one-off fixed gap
    body,
)`),
				prose("Gap spaces every pair of children uniformly — layout rhythm belongs to the "+
					"container, not to margins sprinkled on children. Spacer is the one-off version: "+
					"a fixed empty block between two specific siblings."),
				demoPanel("Flip the axis and stretch the gap.",
					comps.SegmentedControl{
						Style:     segWrap,
						Labels:    []string{"Row", "Column"},
						Selected:  axis.Get(),
						OnSelect:  func(i int) { axis.Set(i) },
						KeyPrefix: "axis-",
					},
					stepper("Gap", gap, 0, 32, 4),
					core.IfElse(axis.Get() == 0,
						core.Row(boxes...),
						core.Column(boxes...),
					),
				),
				keyPoints(
					"Row lays children on the main horizontal axis, Column on the vertical.",
					"Gap is the container's spacing; prefer it over per-child margins.",
					"Containers accept props and children interleaved — argument order among children is layout order.",
				),
			)
		},
	}
}

// --- 1.4 -----------------------------------------------------------------

// The alignment demos map segment indices straight onto the enum values —
// the same index-is-the-value contract SegmentedControl documents.
var (
	justifyLabels = []string{"Start", "Center", "Between", "Evenly"}
	justifyValues = []core.JustifyContent{core.JustifyStart, core.JustifyCenter, core.JustifyBetween, core.JustifyEvenly}
	alignLabels   = []string{"Start", "Center", "End", "Stretch"}
	alignValues   = []core.AlignItems{core.AlignItemsStart, core.AlignItemsCenter, core.AlignItemsEnd, core.AlignItemsStretch}
)

func lessonAlignment() Lesson {
	return Lesson{
		Title:   "Alignment & flex",
		Summary: "Justify places children along the axis, AlignItems across it, FlexGrow claims slack.",
		Body: func(ctx *core.Context) core.View {
			justifyIdx := core.NewState(ctx, 0)
			alignIdx := core.NewState(ctx, 0)
			growB := core.NewState(ctx, false)
			floorA := core.NewState(ctx, false)

			return core.Column(
				core.Gap(14),
				prose("Three props do most layout work. Justify distributes children along the "+
					"container's main axis; AlignItems aligns them across it; FlexGrow lets one "+
					"child absorb whatever space is left."),
				codeBlock(`core.Row(
    core.Justify(core.JustifyBetween),
    core.AlignItemsProp(core.AlignItemsCenter),
    avatar,
    core.Text(title, core.FlexGrow(1)),  // takes the slack
    chevron,
)`),
				demoPanel("The boxes have different heights, so cross-axis alignment is visible.",
					core.ComponentFunc(func(ctx *core.Context) *core.Node {
						return core.Row(
							core.BackgroundColor(ctx.Theme().Colors.Surface),
							core.BorderRadius(8),
							core.Padding(8),
							core.Width("100%"),
							core.Gap(8),
							core.Justify(justifyValues[justifyIdx.Get()]),
							core.AlignItemsProp(alignValues[alignIdx.Get()]),
							// A percentage floor: at least 40% of the row's
							// width, whatever its label needs. It resolves
							// against the row on every target (on iOS through
							// GrMobMinimumLayout, per proposal), and A's label
							// stays centred inside the wider box.
							demoBox("A", boxBlue, 0,
								core.MaybeProp(floorA.Get(), core.MinWidth("40%"))),
							demoBox("B", boxTeal, 8,
								core.MaybeProp(growB.Get(), core.FlexGrow(1))),
							demoBox("C", boxPlum, 16),
						).Render(ctx)
					}),
					core.Column(
						core.Gap(6),
						caption("Justify — main axis"),
						comps.SegmentedControl{
							Style:     segWrap,
							Labels:    justifyLabels,
							Selected:  justifyIdx.Get(),
							OnSelect:  func(i int) { justifyIdx.Set(i) },
							KeyPrefix: "justify-",
						},
					),
					core.Column(
						core.Gap(6),
						caption("AlignItems — cross axis"),
						comps.SegmentedControl{
							Style:     segWrap,
							Labels:    alignLabels,
							Selected:  alignIdx.Get(),
							OnSelect:  func(i int) { alignIdx.Set(i) },
							KeyPrefix: "align-",
						},
					),
					checkRow("FlexGrow(1) on B — B soaks up the slack, and Justify has none left to distribute", growB),
					checkRow("MinWidth(\"40%\") on A — a floor measured against the row, with A's label centred inside it", floorA),
				),
				keyPoints(
					"Justify spends the container's spare main-axis space; once a child grows, there is none.",
					"AlignItems works on the cross axis; Stretch equalizes the children's cross size.",
					"FlexGrow is proportional on every target — weights, not fixed sizes.",
					"MinWidth takes a percentage of the parent as well as points: a floor that scales with the container, which the box's content aligns inside.",
				),
			)
		},
	}
}

// --- 1.5 -----------------------------------------------------------------

func lessonSurfaces() Lesson {
	return Lesson{
		Title:   "Surfaces",
		Summary: "Box is unopinionated, Card is themed, Scroll adds a viewport — and Screen scaffolds it all.",
		Body: func(ctx *core.Context) core.View {
			radius := core.NewState(ctx, 12)
			shadow := core.NewState(ctx, true)

			shadowElev := 0.0
			if shadow.Get() {
				shadowElev = 6
			}

			surfaceContent := func(name, verdict string) []core.PropsAndChildren {
				return []core.PropsAndChildren{
					core.Gap(4),
					core.FlexGrow(1),
					core.Text(name, core.FontWeight(core.Bold)),
					caption(verdict),
				}
			}

			return core.Column(
				core.Gap(14),
				prose("Box and Card hold children like the stacks do, but carry different "+
					"opinions. Box has no theme base at all — a neutral div. Card takes the theme's "+
					"surface treatment: background, padding, radius, shadow."),
				codeBlock(`core.Box(child)                    // nothing but a container
core.Card(core.Gap(8), title, body) // themed surface

// A whole screen, scaffolded: safe area + scroll + column.
comps.Screen{
    Scroll:   true,
    Gap:      16,
    Children: []core.View{hero, section, footer},
}`),
				prose("Style props override the theme base per use — the demo below drives Card's "+
					"radius and shadow directly. For a screen's outer frame, reach for "+
					"comps.Screen instead of hand-stacking SafeArea, Scroll, and Column; this "+
					"very lesson renders inside one."),
				demoPanel("Same content, two surfaces. The steppers restyle only the Card.",
					core.Row(
						core.Gap(12),
						core.Box(surfaceContent("Box", "No background, no padding — bring your own opinion.")...),
						core.Card(append(surfaceContent("Card", "Theme surface, restyled per use below."),
							core.BorderRadius(float64(radius.Get())),
							core.Shadow(shadowElev),
						)...),
					),
					stepper("BorderRadius", radius, 0, 28, 4),
					checkRow("Shadow", shadow),
				),
				prose("A Scroll's position belongs to the platform, and Go never reads it. To take "+
					"the reader somewhere, name the node with core.ScrollTarget and issue "+
					"core.ScrollIntoView from a handler: the host scrolls the least that shows it, "+
					"the way each platform's own bring-into-view does."),
				codeBlock(`core.Row(core.ScrollTarget("row-10"), core.Text("Row 10"))
core.Button("Jump to row 10", func() { core.ScrollIntoView(ctx, "row-10") })`),
				demoPanel("A short Scroll of twelve rows; the buttons move it, not your finger.",
					scrollRows(),
					core.Row(
						core.Gap(8),
						core.FlexWrap(true),
						comps.Button{Label: "Jump to row 10", Emphasis: comps.EmphasisOutlined,
							OnTap: func() { core.ScrollIntoView(ctx, "surfaces-row-10") }},
						comps.Button{Label: "Back to row 1", Emphasis: comps.EmphasisGhost,
							OnTap: func() { core.ScrollIntoView(ctx, "surfaces-row-1") }},
					),
				),
				keyPoints(
					"Box is the escape hatch with no theme base; Card is the themed surface.",
					"Per-use style props layer over the theme's component base and win.",
					"comps.Screen is the root scaffold: safe area, optional scroll region, content column.",
					"Use Scroll for short content and core.List for long data-driven collections — never nest them.",
					"core.ScrollTarget names a node; core.ScrollIntoView brings it into view, once per call.",
				),
			)
		},
	}
}

// scrollRows is the scroll-to demo's box: a Scroll with a points height, so
// it is a viewport of its own inside the lesson's, holding twelve named rows.
// The rows are the Scroll's direct children, which is what SwiftUI's
// scrollTo needs to find them (Renderer.swift, GrMobBringIntoView).
func scrollRows() core.View {
	return core.ComponentFunc(func(ctx *core.Context) *core.Node {
		th := ctx.Theme()
		rows := []core.PropsAndChildren{
			core.Height("160px"),
			core.BorderColor(th.Colors.BorderColor()),
			core.BorderWidth(1),
			core.BorderRadius(8),
		}
		for i := 1; i <= 12; i++ {
			rows = append(rows, core.Row(
				core.ScrollTarget(fmt.Sprintf("surfaces-row-%d", i)),
				core.PaddingHorizontal(12),
				core.PaddingVertical(10),
				core.Text(fmt.Sprintf("Row %d", i)),
			))
		}
		return core.Scroll(rows...).Render(ctx)
	})
}
