package tutorial

import (
	"fmt"

	"github.com/rohanthewiz/grmob/components"
	"github.com/rohanthewiz/grmob/core"
)

// chapter7 — Theming & Styling: styling as plain data. A node's look is one
// Style struct assembled in a single pass — the theme's component base, then
// the caller's props in argument order — with no stylesheet, cascade, or
// specificity anywhere. The through-line is that every restyling gesture is
// therefore a data change the reconciler ships as ordinary update-style
// patches: append a prop (7.1), layer a named Style value (7.2), read a role
// off the theme (7.3), swap the whole theme under a subtree (7.4) — and
// Transition (7.5) is the one declaration that says how those patches should
// move instead of snap.
func chapter7() Chapter {
	return Chapter{
		Title:   "Theming & Styling",
		Icon:    "🎨",
		Summary: "Style props, merging, the theme's roles — and the live switch that re-skins a subtree, with transitions to make changes glide.",
		Lessons: []Lesson{
			lessonStylePipeline(),
			lessonStyleMerging(),
			lessonThemeAnatomy(),
			lessonThemeSwitch(),
			lessonTransitions(),
		},
	}
}

// --- 7.1 -----------------------------------------------------------------

func lessonStylePipeline() Lesson {
	return Lesson{
		Title:   "The style pipeline",
		Summary: "No stylesheets, no cascade — a theme base plus your props in argument order, and the last prop wins.",
		Body: func(ctx *core.Context) core.View {
			// The two competing coats. Both are BackgroundColor props; which
			// one shows is decided purely by where each sits in the argument
			// list, which is the entire conflict-resolution model.
			teal := core.NewState(ctx, false)
			plum := core.NewState(ctx, false)

			// Build the demo box's argument list exactly the way app code
			// does: start with the base look, append overrides in order. The
			// list IS the pipeline — there is nothing else to consult.
			props := []core.PropsAndChildren{
				core.BackgroundColor(boxBlue),
				core.BorderRadius(10),
				core.Padding(16),
			}
			if teal.Get() {
				props = append(props, core.BackgroundColor(boxTeal))
			}
			if plum.Get() {
				props = append(props, core.BackgroundColor(boxPlum))
			}
			props = append(props, core.Text("Painted by the last prop standing",
				core.TextColor("#FFFFFF"),
				core.FontWeight(core.Bold),
			))

			// Narrate the winner so the rule is stated by the running demo,
			// not just the prose. The order of these ifs mirrors the appends.
			winner := boxBlue + " — the base coat, nothing overrode it"
			if teal.Get() {
				winner = boxTeal + " — the teal coat ran after the base"
			}
			if plum.Get() {
				winner = boxPlum + " — appended last, so it wins"
			}

			return core.Column(
				core.Gap(14),
				prose("A widget's final look is one Style struct, assembled in one pass: the "+
					"theme's base for that widget kind (Components.Button, Components.Card, "+
					"Components.Input, …), then your props applied in argument order. Each prop "+
					"is a function that sets one field, later props overwrite the fields they "+
					"touch, and what the node renders with is exactly what the final struct "+
					"says. There is no stylesheet to consult, no cascade, no specificity — "+
					"reading the argument list top to bottom is reading the answer."),
				codeBlock(`// 1. the theme's component base   (Components.Button here)
// 2. your props, in argument order — later props win
core.Button("Save", onSave,
    core.BackgroundColor("#1F8A70"), // overrides the base's Primary fill
    core.Padding(14),                // overrides the base's padding
)

// Order is the whole conflict-resolution story:
core.Box(
    core.BackgroundColor(blue),
    core.BackgroundColor(plum), // wins — it ran last
)`),
				prose("One asymmetry to know: containers, buttons and inputs resolve a base "+
					"style from the theme's ComponentDefaults, but core.Text starts from a "+
					"bare Style — body text is too varied for one default to help. Typography "+
					"is opt-in instead: layer a role from the theme's type scale with "+
					"core.UseStyle(t.Typography.Body), which is the next lesson's subject."),
				demoPanel("Toggle the coats and watch the argument list resolve itself.",
					core.Column(props...),
					caption("Background now "+winner),
					checkRow("Add the teal coat", teal),
					checkRow("Add the plum coat — appended after teal", plum),
				),
				keyPoints(
					"A node's look is one Style struct: theme component base first, then props in argument order.",
					"Later props win — the argument list is the entire conflict-resolution model.",
					"Containers, buttons and inputs get a theme base from ComponentDefaults; core.Text starts bare and takes typography roles by choice.",
					"Style changes between renders ship as update-style patches — restyling is data, not a new subtree.",
				),
			)
		},
	}
}

// --- 7.2 -----------------------------------------------------------------

// calloutStyle is 7.2's named role — the docs' "define base styles as values"
// idiom. Package-level because a role is shared vocabulary, not per-screen
// state; DisplayBlock because the demo applies it to a bare Text node, and an
// inline span would not carry the padding and corners the lesson watches.
var calloutStyle = core.Style{
	Background:   boxPlum,
	TextColor:    "#FFFFFF",
	FontSize:     15,
	BorderRadius: 14,
	Padding:      core.EdgeInsets{Top: 16, Bottom: 16, Left: 16, Right: 16},
	Display:      core.DisplayBlock,
}

// bigType is the layering half of the demo: it names ONLY the type fields, so
// applying it proves the merge rule — fill, ink and corners pass through
// untouched because this value never set them.
var bigType = core.Style{
	FontSize:   24,
	FontWeight: core.Bold,
}

func lessonStyleMerging() Lesson {
	return Lesson{
		Title:   "UseStyle: layers that merge",
		Summary: "Name a Style value and layer it on: set fields win, zero fields pass through — and why zero can't clear.",
		Body: func(ctx *core.Context) core.View {
			layerBig := core.NewState(ctx, false)
			clearZero := core.NewState(ctx, false) // the documented no-op
			clearProp := core.NewState(ctx, false) // the working alternative

			// One node, one Style: every toggle below edits the same struct,
			// in the order the appends run, so the preview is a literal
			// visualization of the merge.
			props := []core.StyleProp{core.UseStyle(calloutStyle)}
			if layerBig.Get() {
				props = append(props, core.UseStyle(bigType))
			}
			if clearZero.Get() {
				// Applies cleanly and changes nothing: in a merge, a zero
				// field means "not set", so there is nothing to carry over.
				props = append(props, core.UseStyle(core.Style{BorderRadius: 0}))
			}
			if clearProp.Get() {
				// A direct prop assigns unconditionally — this is the tool
				// for forcing a value rather than layering one.
				props = append(props, core.BorderRadius(0))
			}

			return core.Column(
				core.Gap(14),
				prose("Once a look repeats, name it: define a Style value, and core.UseStyle "+
					"turns the whole value into a single prop. The merge rule is \"a set field "+
					"wins, an unset field is ignored\" — layering a role onto a theme base only "+
					"ever adds, never blanks out what the base supplied. Style.With composes "+
					"two values under the same rule, and the nested HoverStyle, FocusStyle and "+
					"PseudoStates merge recursively, so a value describing only \":hover\" "+
					"cannot delete a \":focus\" already present."),
				codeBlock(`var callout = core.Style{Background: plum, TextColor: "#FFFFFF", BorderRadius: 14}
var bigType = core.Style{FontSize: 24, FontWeight: core.Bold}

core.Text(msg, core.UseStyle(callout.With(bigType))) // one merged value
core.Text(msg, core.UseStyle(callout), core.UseStyle(bigType)) // same result

// The edge: zero means "not set", so UseStyle cannot CLEAR a field —
core.UseStyle(core.Style{BorderRadius: 0}) // a no-op by design
core.BorderRadius(0)                       // a direct prop assigns unconditionally`),
				prose("The rule has one unavoidable edge: a zero value is indistinguishable "+
					"from \"not set\", so UseStyle cannot clear a field the node already has — "+
					"Style{BorderRadius: 0} layers nothing, it just holds no opinion. When the "+
					"intent is to force a value, reach past the merge with the direct prop: "+
					"core.BorderRadius(0) runs in argument order like any other prop and "+
					"assigns whether or not zero is \"set\". The demo below walks the exact "+
					"trap and the exact fix."),
				demoPanel("Layer big type — then try both ways of clearing the corners.",
					core.Text("One node, one Style — watch the corners and the type.", props...),
					checkRow("Layer bigType via UseStyle — fill, ink and corners survive", layerBig),
					checkRow("Clear the radius with UseStyle(Style{BorderRadius: 0}) — nothing happens", clearZero),
					checkRow("Clear it with core.BorderRadius(0) — corners go square", clearProp),
					caption("The first clear merges a zero, and a zero field holds no opinion. "+
						"The second is an assignment, and it runs last."),
				),
				keyPoints(
					"UseStyle turns a named Style value into one prop: set fields win, zero fields pass through.",
					"Layering only ever adds — a role never blanks out what the theme base already supplied.",
					"Style.With composes two values; hover/focus/pseudo states merge recursively, key by key.",
					"Zero is indistinguishable from unset, so UseStyle cannot clear — use the direct prop to force a value.",
				),
			)
		},
	}
}

// --- 7.3 -----------------------------------------------------------------

// swatchRow is one line of 7.3's palette inspector: a color chip, the role
// name, the literal hex. The chip carries a hairline in the APP theme's
// Border role — without it the inspected Surface swatch (near-white on a
// white page) would be invisible, which is itself the Border-vs-Surface
// lesson from the docs, drawn.
func swatchRow(role, hex string) core.View {
	return core.ComponentFunc(func(ctx *core.Context) *core.Node {
		return core.Row(
			core.Gap(10),
			core.AlignItemsProp(core.AlignItemsCenter),
			core.Box(
				core.Width("22px"),
				core.Height("22px"),
				core.BackgroundColor(hex),
				core.BorderRadius(6),
				core.BorderColor(ctx.Theme().Colors.BorderColor()),
				core.BorderWidth(1),
			),
			core.Text(role, core.FontSize(14), core.FontWeight(core.Bold), core.MinWidth("110px")),
			caption(hex),
		).Render(ctx)
	})
}

func lessonThemeAnatomy() Lesson {
	return Lesson{
		Title:   "Inside a Theme",
		Summary: "One value holds the design system: color roles, a type scale, spacing, and per-widget bases.",
		Body: func(ctx *core.Context) core.View {
			// Which bundled theme the inspector reads. This lesson only READS
			// the two package vars as plain data — nothing is installed on any
			// context, which is exactly the boundary between this lesson and
			// the next.
			inspect := core.NewState(ctx, 0)
			bundled := []*core.Theme{core.DefaultTheme, core.MaterialTheme}
			th := bundled[inspect.Get()]

			// Roles in the docs' order: brand pair, grounds, inks, status
			// triad, stroke. The three late roles go through their resolvers —
			// redundant for a bundled theme, but the habit is the lesson.
			roles := []struct{ name, hex string }{
				{"Primary", th.Colors.Primary},
				{"Secondary", th.Colors.Secondary},
				{"Background", th.Colors.Background},
				{"Surface", th.Colors.Surface},
				{"TextPrimary", th.Colors.TextPrimary},
				{"TextSecondary", th.Colors.TextSecondary},
				{"Error", th.Colors.Error},
				{"Success", th.Colors.SuccessColor()},
				{"Warning", th.Colors.WarningColor()},
				{"Border", th.Colors.BorderColor()},
			}
			swatches := []core.PropsAndChildren{core.Gap(6)}
			for _, r := range roles {
				swatches = append(swatches, swatchRow(r.name, r.hex))
			}

			return core.Column(
				core.Gap(14),
				prose("A Theme is the design system as one Go value, in four sections: Colors "+
					"(named roles, never literals), Typography (a type scale of four Styles), "+
					"Spacing (five steps), and Components (the per-widget base styles 7.1's "+
					"pipeline starts from). Widgets read it from the context — ctx.Theme(), "+
					"which falls back to DefaultTheme when none is installed — so a screen "+
					"written in roles restyles itself the moment the value changes."),
				codeBlock(`type Theme struct {
    Colors     ColorPalette      // Primary, Surface, TextPrimary, Border, …
    Typography Typography        // Title, Subtitle, Body, Caption — each a Style
    Spacing    SpacingScale      // XS SM MD LG XL
    Components ComponentDefaults // base Style per widget: Button, Card, Input, …
}

t := ctx.Theme() // DefaultTheme when none is installed
core.Text("Hello", core.UseStyle(t.Typography.Title))
core.Box(core.BackgroundColor(t.Colors.Surface))
core.BorderColor(t.Colors.BorderColor()) // late roles resolve through methods`),
				prose("The role names carry two distinctions worth pinning. Border is a stroke "+
					"and Surface is a fill — near neighbors on a light theme, which is why the "+
					"inspector's swatches wear a hairline. And Success is not Secondary even "+
					"when a theme tints both the same green, as Default does: Secondary is a "+
					"brand slot a theme may make teal (Material does — see for yourself below), "+
					"while Success carries meaning. Border, Success and Warning arrived after "+
					"the original seven roles, so read them through their resolver methods — "+
					"BorderColor(), SuccessColor(), WarningColor() — and a theme written before "+
					"they existed degrades to the documented fallback instead of to no color at "+
					"all. When you write a theme of your own, fill in Components.Button at "+
					"minimum: ComponentDefaults has no resolvers, and a missing base is "+
					"genuinely no styling."),
				demoPanel("Pick a bundled theme and read its data — nothing is installed here; installing is the next lesson.",
					components.SegmentedControl{
						Style:     segWrap,
						Labels:    []string{"Default", "Material"},
						Selected:  inspect.Get(),
						OnSelect:  func(i int) { inspect.Set(i) },
						KeyPrefix: "inspect-",
					},
					core.Column(swatches...),
					caption(fmt.Sprintf("Type scale — Title %.0f · Subtitle %.0f · Body %.0f · Caption %.0f",
						th.Typography.Title.FontSize, th.Typography.Subtitle.FontSize,
						th.Typography.Body.FontSize, th.Typography.Caption.FontSize)),
					caption(fmt.Sprintf("Button base — fill %s · radius %.0f",
						th.Components.Button.Background, th.Components.Button.BorderRadius)),
				),
				keyPoints(
					"A Theme is one value: color roles, a type scale, spacing steps, and per-widget base styles.",
					"Read it with ctx.Theme() — DefaultTheme is the fallback when nothing is installed.",
					"Name the role, never the literal: Border is a stroke not a fill, Success carries meaning while Secondary is brand.",
					"Border, Success and Warning postdate the original roles — read them through their resolvers and old themes degrade gracefully.",
					"ComponentDefaults has no resolvers: a theme of your own should set Components.Button at minimum.",
				),
			)
		},
	}
}

// --- 7.4 -----------------------------------------------------------------

// themePreview is the subtree 7.4 re-skins: a profile card written entirely
// in roles — Typography for the sizes, palette roles for every color, widget
// bases for the button and badge — so the theme swap accounts for every pixel
// in it. Written against whatever theme its context carries and nothing else.
//
// Deliberately hook-free: core.WithTheme hands its children a derived context
// that shares the frame's slot table but forks the hook cursor, so a hook
// claimed in here could collide with one the lesson claims after the wrapper.
// The preview's state therefore lives in the lesson's frame and arrives by
// closure — the same ownership shape every controlled widget follows.
func themePreview(following bool, onFollow func()) core.View {
	return core.ComponentFunc(func(ctx *core.Context) *core.Node {
		t := ctx.Theme()

		followLabel := "Follow"
		if following {
			followLabel = "Following ✓"
		}

		return core.Card(
			core.Gap(10),
			core.Row(
				core.Gap(8),
				core.AlignItemsProp(core.AlignItemsCenter),
				core.Text("Gopher McGrMob", core.UseStyle(t.Typography.Title)),
				core.Box(core.FlexGrow(1)), // slack, so the badge pins right
				components.Badge{Text: "PRO"},
			),
			core.Text("Every color, size and corner on this card resolves from the theme "+
				"this subtree was handed — the card base, the title's scale, the button "+
				"fills, the badge tint.", core.UseStyle(t.Typography.Body)),
			core.Text("Written in roles, so the swap is total.",
				core.UseStyle(t.Typography.Caption),
				core.TextColor(t.Colors.TextSecondary),
			),
			core.Row(
				core.Gap(8),
				components.Button{Label: followLabel, OnTap: onFollow},
				components.Button{
					Label:    "Message",
					Emphasis: components.EmphasisOutlined,
					// A toast, as 6.5 taught: fire-and-forget confirmation,
					// drawn by the host above everything — themes included.
					OnTap: func() { core.ShowToast("Message sent to the gopher") },
				},
			),
		).Render(ctx)
	})
}

func lessonThemeSwitch() Lesson {
	return Lesson{
		Title:   "Two themes, one tree",
		Summary: "The theme rides the context, so a subtree can run Material while the app stays Default — live.",
		Body: func(ctx *core.Context) core.View {
			// The switcher's entire mechanism: which theme to hand the
			// wrapper, held as ordinary lesson state. Both hooks live in the
			// lesson's frame — see themePreview for why none live inside the
			// re-themed subtree.
			pick := core.NewState(ctx, 0)
			following := core.NewState(ctx, false)
			installed := []*core.Theme{core.DefaultTheme, core.MaterialTheme}

			return core.Column(
				core.Gap(14),
				prose("The theme travels on the Context: installing one derives a context that "+
					"carries the new value, and every widget under it re-resolves its roles "+
					"from there, each pass. Install at the root for the whole app — this "+
					"tutorial runs that way — or wrap any subtree in core.WithTheme to scope "+
					"one: a Material settings panel inside a Default app is one wrapper, and "+
					"everything outside it never notices."),
				codeBlock(`// Install at the root — the whole app renders under it:
ctx := core.NewContext().WithTheme(core.DefaultTheme)

// Or scope a subtree — children resolve every role from the new theme:
core.WithTheme(core.MaterialTheme, settingsPanel)

// A LIVE switcher is nothing more than the theme held as state:
installed := []*core.Theme{core.DefaultTheme, core.MaterialTheme}
pick := core.NewState(ctx, 0)
core.WithTheme(installed[pick.Get()], preview)`),
				prose("Because which theme a node sees is decided at render time, a live "+
					"switcher needs no framework machinery at all: hold the choice as state, "+
					"hand the chosen theme to the wrapper, and the flip is an ordinary render "+
					"whose changes ship as update-style patches. The swap only reaches what "+
					"was written in roles — 7.3's habit pays out here. The tutorial's code "+
					"blocks keep their fixed editor palette under any theme for exactly that "+
					"reason: a literal is a promise the theme can't touch."),
				demoPanel("Flip the card between themes — then notice the chips you're tapping don't change.",
					components.SegmentedControl{
						Style:     segWrap,
						Labels:    []string{"Default", "Material"},
						Selected:  pick.Get(),
						OnSelect:  func(i int) { pick.Set(i) },
						KeyPrefix: "installed-",
					},
					core.WithTheme(installed[pick.Get()],
						themePreview(following.Get(), func() { following.Set(!following.Get()) }),
					),
					caption("Only the card sits inside the wrapper. These captions, the chips "+
						"above, and the lesson around you are outside it — still on the app's "+
						"DefaultTheme. Scoping is the point."),
				),
				keyPoints(
					"The theme rides the Context: WithTheme derives a themed copy, and widgets re-resolve their roles from it every pass.",
					"Install at the root with NewContext().WithTheme, or scope any subtree with core.WithTheme(theme, children...).",
					"A live switcher is the theme as state feeding the wrapper — an ordinary render, shipped as update-style patches.",
					"The swap reaches exactly what was written in roles; literals are promises the theme can't touch.",
					"Keep hooks out of a WithTheme subtree: the themed context forks the hook cursor — own the state in the frame and pass it down.",
				),
			)
		},
	}
}

// --- 7.5 -----------------------------------------------------------------

func lessonTransitions() Lesson {
	return Lesson{
		Title:   "Transitions: declare the motion",
		Summary: "Transition(ms, easing) makes patched style changes glide — Go declares, the platform animates.",
		Body: func(ctx *core.Context) core.View {
			alert := core.NewState(ctx, false)
			// Start on 250 ms so the very first flip glides — the demo leads
			// with the feature, and Snap is one tap away for the contrast.
			pace := core.NewState(ctx, 1)
			durations := []int{0, 250, 800}

			// The two looks the box moves between. Padding is in the diff on
			// purpose: it animates the box's SIZE, so the glide is visibly
			// more than a color fade.
			bg, pad, radius := boxBlue, 14, 10.0
			if alert.Get() {
				bg, pad, radius = boxPlum, 26, 24.0
			}

			return core.Column(
				core.Gap(14),
				prose("Interactions arrive as update-style patches — 7.4's theme swap, a "+
					"selection change, this box flipping its look. core.Transition(durationMs, "+
					"easing) declares that changes to the node's animatable properties should "+
					"glide over that duration instead of snapping. This is the declare-in-Go, "+
					"drive-natively model: Go ships the declaration once, in the style, and "+
					"every animated frame is produced by the platform's own animation system — "+
					"CSS transitions in the browser, Compose and SwiftUI on the phones — never "+
					"by patches over the bridge."),
				codeBlock(`core.Column(
    core.Transition(250, core.EaseInOut), // serialized as "250ms ease-in-out"
    core.BackgroundColor(bg),             // animates when bg changes between renders
    core.Padding(pad),                    // size changes glide too
)

// Easings: core.EaseLinear, Ease, EaseIn, EaseOut, EaseInOut.
// Transition(0, …) clears the declaration — changes snap again.`),
				prose("Put the declaration on the node whose style responds to state, and "+
					"declare it every pass — it is just a Style field, present or absent like "+
					"any other. Two boundaries keep it honest: a transition animates the "+
					"change between two rendered styles (it is not a keyframe animation), and "+
					"it is presentation only — there is no completion callback, deliberately, "+
					"so nothing in app logic can wait on a glide the platform may shorten, "+
					"skip, or never start."),
				demoPanel("Flip the look at each pace — the state change and the patch are identical every time.",
					core.Column(
						core.Transition(durations[pace.Get()], core.EaseInOut),
						core.BackgroundColor(bg),
						core.Padding(pad),
						core.BorderRadius(radius),
						core.Text("Same patch, different journey",
							core.TextColor("#FFFFFF"),
							core.FontWeight(core.Bold),
						),
					),
					core.Row(
						core.Gap(8),
						core.AlignItemsProp(core.AlignItemsCenter),
						components.Button{
							Label: "Flip the look",
							OnTap: func() { alert.Set(!alert.Get()) },
						},
						components.SegmentedControl{
							Style:     segWrap,
							Labels:    []string{"Snap", "250 ms", "800 ms"},
							Selected:  pace.Get(),
							OnSelect:  func(i int) { pace.Set(i) },
							KeyPrefix: "pace-",
						},
					),
					caption("Snap is Transition(0): the declaration clears and the change lands "+
						"in one frame. The other two ship the exact same patch with a different "+
						"declared journey."),
				),
				keyPoints(
					"Transition(ms, easing) declares motion in the style; the platform's animation system produces every frame.",
					"It animates the change between rendered styles — colors, size, padding, placement — not keyframes.",
					"Declare it on the node whose style responds to state; Transition(0) clears it and changes snap.",
					"The serialized form is \"<ms>ms <easing>\", one string every renderer parses.",
					"Presentation only: no completion callback exists, so logic can never depend on a glide finishing.",
				),
			)
		},
	}
}
