// The tutorial's own building blocks: prose paragraphs, code blocks, the
// "try it" demo panel, key-point lists, and the little colored boxes the
// layout lessons push around. All of them are pure functions of their
// arguments (no hooks), so lessons may use them freely in any order.
//
// Each returns a core.ComponentFunc rather than a pre-built view because most
// of them read the theme — palette roles, typography, spacing — and a theme
// only exists once a Context arrives. This is the same deferral trick
// examples/social's TabButton uses.
package tutorial

import (
	"strings"

	"github.com/rohanthewiz/grmob/comps"
	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/highlight"
)

// prose is a body paragraph in the theme's Body typography.
func prose(text string) core.View {
	return core.ComponentFunc(func(ctx *core.Context) *core.Node {
		return core.Text(text, core.UseStyle(ctx.Theme().Typography.Body)).Render(ctx)
	})
}

// caption is de-emphasized small text — figure captions, hints under demos.
func caption(text string) core.View {
	return core.ComponentFunc(func(ctx *core.Context) *core.Node {
		t := ctx.Theme()
		return core.Text(text,
			core.UseStyle(t.Typography.Caption),
			core.TextColor(t.Colors.TextSecondary),
		).Render(ctx)
	})
}

// Code blocks keep their own fixed dark palette instead of theme roles: an
// editor-dark surface reads as "this is code" under any app theme, and the
// palette has no role for it.
//
// The surface, its default ink and the token colours are all one scheme now —
// highlight.Darcula, named on the widget below — where they used to be a pair
// of hex literals here plus a palette in highlight.go that happened to agree
// with them. Nothing hard-codes a colour in the tutorial chrome any more.

// codeBlock renders a Go snippet, syntax-highlighted.
//
// It is a read-only comps.CodeEditor, which is the widget's display half:
// the same monospace rows a core.TextGrid draws, plus a caret the reader can
// put in the code and a selection they can copy out of. A code block in a
// tutorial is exactly that — something to read and to take away — and the
// editor is the node built for it.
//
// It was a core.TextGrid until the editor existed, and the properties that made
// the grid right are the ones the editor inherits rather than replaces:
//
//   - Monospace. core.Style still has no font-family prop, and neither node
//     needs one: both are fixed-pitch on every target by construction.
//   - No wrapping, and sideways scrolling when a snippet is wider than the
//     phone. A wrapped code line restarts at column zero, which reads as a new
//     statement at the outermost indent — so wrapping destroys exactly the
//     structure indentation exists to show.
//   - Indentation, and blank lines that keep their height. A <span> in
//     ordinary flow collapses leading spaces; a grid row preserves them and an
//     empty row still takes a line.
//
// # What ReadOnly buys and what it costs
//
// It is not core.Disabled, and the difference is the whole point: a disabled
// control is inert and greyed and is skipped by assistive technology, while a
// read-only one is content the reader is meant to select. The cost on the web
// is one element — the runtime lays a transparent <textarea> over the rows so
// that the caret is real — and that element leaves the tab order when the
// editor is read-only, so a lesson with nine snippets in it still has no tab
// stops it did not have before.
//
// # No hooks, which is why this can be called anywhere
//
// comps.CodeEditor consumes no hook slot unless it is given a toolbar, and
// this one is not. That matters here more than anywhere: code blocks are built
// inside lesson bodies, inside conditionals, inside loops over a demo's state,
// and a widget with hook obligations could not be.
//
// The Trim is the tutorial's own: a snippet is written as a raw string literal
// that starts and ends with a newline, and those two newlines are formatting.
// See highlightGo, which says why the highlight package must not do it.
//
// # The copy button
//
// Every snippet carries a comps.CopyButton as a ZStack layer pinned to its
// top-end corner, so a reader can take the code into their own editor without
// selecting across a scrolling grid. CopyButton is stateless, which is what
// lets it live here: the rule above holds for the block as a whole. It copies
// the trimmed snippet — the exact string the editor draws — and is coloured
// from the scheme rather than the theme, because its backdrop is the code
// surface and not the page: the theme's on-light tones would be dark ink on
// Darcula's dark grey.
//
//	┌ ZStack ───────────────────────────────────────┐
//	│ ┌ CodeEditor (base layer) ──────────[ Copy ]┐ │  top-end, 6px in
//	│ │ func main() {                              │ │
//	│ │     …                                      │ │
//	│ └────────────────────────────────────────────┘ │
//	└────────────────────────────────────────────────┘
//
// The button carries ZIndex(1); see the comment at the prop for why the
// top layer has to say so on the web. The button can cover the end of a long
// first line. Code wider than the
// phone already scrolls sideways under it, and a separate header strip was
// rejected because it would add a row to every one of the tutorial's
// snippets to hold one small control.
func codeBlock(code string) core.View {
	snippet := strings.Trim(code, "\n")
	scheme := highlight.Darcula
	return core.ZStack(
		// Full width so the ZStack spans the lesson column the way the bare
		// editor did; a ZStack otherwise sizes to its largest layer, and the
		// editor's intrinsic width is its longest line.
		core.Width("100%"),
		comps.CodeEditor{
			Value:    snippet,
			Language: "go",
			Scheme:   scheme,
			ReadOnly: true,
			Style: []core.StyleProp{
				core.Width("100%"),
				core.Padding(14),
				core.BorderRadius(10),
				core.FontSize(13),
			},
		},
		comps.CopyButton{
			Text:               snippet,
			AccessibilityLabel: "Copy code",
			CopiedMessage:      "Code copied",
			Emphasis:           comps.EmphasisGhost,
			Style: []core.StyleProp{
				core.StackAlign(core.StackAlignTopEnd),
				// Above the editor explicitly. The web draws a CodeEditor as a
				// position:relative <pre> (the containing block for its caret
				// overlay), and a positioned element paints over a later
				// non-positioned sibling — so without this the button is laid
				// out in the corner and drawn underneath the code.
				core.ZIndex(1),
				core.Margin(6),
				core.PaddingVertical(2),
				core.PaddingHorizontal(8),
				core.FontSize(12),
				core.BorderRadius(6),
				core.TextColor(scheme.Ink),
				core.Background(scheme.Bg),
				core.BorderWidth(1),
				core.BorderColor(scheme.Comment),
			},
		},
	)
}

// demoPanel frames a live demo: a "TRY IT" badge and caption on top, then the
// demo itself. The border is the visual contract — everything inside the
// hairline is live, everything outside is exposition — so it uses the
// palette's Border role (via its resolver, per the theme docs) rather than a
// literal hex.
func demoPanel(hint string, children ...core.View) core.View {
	return core.ComponentFunc(func(ctx *core.Context) *core.Node {
		t := ctx.Theme()
		items := []core.PropsAndChildren{
			core.BorderColor(t.Colors.BorderColor()),
			core.BorderWidth(1),
			core.BorderRadius(12),
			core.Padding(14),
			core.Gap(12),
			core.Row(
				core.Gap(8),
				core.AlignItemsProp(core.AlignItemsCenter),
				// FlexShrink(0): a long hint is the one child here that should
				// give up width. Left shrinkable, the badge took its share of
				// the squeeze and "TRY IT" wrapped onto two lines under any
				// hint past about 25 characters.
				comps.Badge{Text: "TRY IT", Style: []core.StyleProp{core.FlexShrink(0)}},
				caption(hint),
			),
		}
		for _, c := range children {
			items = append(items, c)
		}
		return core.Column(items...).Render(ctx)
	})
}

// keyPoints is the recap list closing every lesson: a subtitle and bulleted
// lines. The bullets are a comps.BulletList — the widget these rows were the
// model for — so the points are also a list of listitems to a screen reader.
//
// "Key points" is a heading at level 2: a section of the lesson whose own name
// lessonHeader carries at level 1. It is the one heading on a lesson screen a
// reader is most likely to jump straight to, which is what the tier is for.
func keyPoints(points ...string) core.View {
	return core.ComponentFunc(func(ctx *core.Context) *core.Node {
		t := ctx.Theme()
		items := []core.PropsAndChildren{
			core.Gap(6),
			core.Text("Key points",
				core.UseStyle(t.Typography.Subtitle),
				core.AccessibilityRole(core.RoleHeading),
				core.AccessibilityHeadingLevel(2)),
		}
		items = append(items, comps.BulletList{Items: points})
		return core.Column(items...).Render(ctx)
	})
}

// demoBox is the labeled colored block the layout lessons arrange. extraPad
// varies box heights so alignment effects are visible; the label's ink is
// fixed white because every box color below is dark enough to carry it.
// extras lets a lesson attach one-off props (the FlexGrow demo) without the
// helper growing a field per experiment.
func demoBox(label, color string, extraPad int, extras ...core.PropsAndChildren) core.View {
	items := []core.PropsAndChildren{
		core.BackgroundColor(color),
		core.BorderRadius(8),
		core.Padding(10 + extraPad),
		core.Justify(core.JustifyCenter),
		core.Text(label,
			core.TextColor("#FFFFFF"),
			core.FontWeight(core.Bold),
			core.Align(core.AlignCenter),
		),
	}
	items = append(items, extras...)
	return core.Column(items...)
}

// The layout-demo palette: three distinguishable mid-dark hues that hold
// white text at AA contrast.
const (
	boxBlue = "#3B6FD4"
	boxTeal = "#1F8A70"
	boxPlum = "#8E4A9E"
)

// segWrap is the Style every SegmentedControl in the tutorial carries.
//
// Several of these controls spell out four longish captions — "Start /
// Center / Between / Evenly", "Default / Filled / Outlined / Ghost" — and a
// row of four does not fit a phone. Without wrapping the last segment sat
// past the right edge of the screen, reachable only by dragging the page
// sideways, on a control whose entire job is to show the available choices at
// a glance.
//
// It is set here per call site rather than inside comps.SegmentedControl
// because that component is held to byte-for-byte parity with the hand-rolled
// bar it replaced (examples/todoapp's TestFilterBarMatchesLegacyMarkup), and
// a default flex-wrap would change the markup of every app using it to fix a
// problem only long captions have. Wrapping is inert while the segments fit.
var segWrap = []core.StyleProp{core.FlexWrap(true)}

// stepper is the −/+ control the demos use for numeric knobs (gap, font
// size, radius): a caption naming the knob, then a comps.Stepper bound to one
// int state.
//
// It hand-rolled the two buttons and the number until comps.Stepper existed.
// The widget now owns what each call site used to pass as a closure: the step
// size and the clamp into [lo, hi]. It also disables the button that would
// leave the range, and names the buttons "Decrease"/"Increase" for screen
// readers, where a bare "−" is read as "minus" or not at all.
//
// The caption stays outside the widget because Stepper deliberately draws no
// label (its Label is only the group's accessible name), and a knob in a demo
// panel needs its name on screen. The same string is passed as that name, so
// what is seen and what is announced cannot drift apart.
func stepper(label string, s core.State[int], lo, hi, step int) core.View {
	return core.Row(
		core.Gap(8),
		core.AlignItemsProp(core.AlignItemsCenter),
		caption(label),
		comps.Stepper{Value: s.Get(), Min: lo, Max: hi, Step: step, OnChange: s.Set, Label: label},
	)
}
