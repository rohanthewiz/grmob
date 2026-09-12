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

	"github.com/rohanthewiz/grmob/components"
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
// It is a read-only components.CodeEditor, which is the widget's display half:
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
// components.CodeEditor consumes no hook slot unless it is given a toolbar, and
// this one is not. That matters here more than anywhere: code blocks are built
// inside lesson bodies, inside conditionals, inside loops over a demo's state,
// and a widget with hook obligations could not be.
//
// The Trim is the tutorial's own: a snippet is written as a raw string literal
// that starts and ends with a newline, and those two newlines are formatting.
// See highlightGo, which says why the highlight package must not do it.
func codeBlock(code string) core.View {
	return components.CodeEditor{
		Value:    strings.Trim(code, "\n"),
		Language: "go",
		Scheme:   highlight.Darcula,
		ReadOnly: true,
		Style: []core.StyleProp{
			core.Padding(14),
			core.BorderRadius(10),
			core.FontSize(13),
		},
	}
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
				components.Badge{Text: "TRY IT"},
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
// lines. Bullets are plain Rows — a list this short gains nothing from
// core.List's virtualization, and static children need no keys.
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
		for _, p := range points {
			items = append(items, core.Row(
				core.Gap(8),
				core.Text("•", core.TextColor(t.Colors.Primary), core.FontWeight(core.Bold)),
				core.Text(p, core.UseStyle(t.Typography.Body), core.FlexGrow(1)),
			))
		}
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
// It is set here per call site rather than inside components.SegmentedControl
// because that component is held to byte-for-byte parity with the hand-rolled
// bar it replaced (examples/todoapp's TestFilterBarMatchesLegacyMarkup), and
// a default flex-wrap would change the markup of every app using it to fix a
// problem only long captions have. Wrapping is inert while the segments fit.
var segWrap = []core.StyleProp{core.FlexWrap(true)}

// stepper is the −/+ control the demos use for numeric knobs (gap, font
// size, radius). Controlled like every GrMob input: it renders value and
// reports intent through onDelta; clamping is the caller's policy.
func stepper(label string, value string, onDelta func(delta int)) core.View {
	return core.ComponentFunc(func(ctx *core.Context) *core.Node {
		return core.Row(
			core.Gap(8),
			core.AlignItemsProp(core.AlignItemsCenter),
			caption(label),
			components.Button{Label: "−", OnTap: func() { onDelta(-1) }, Emphasis: components.EmphasisOutlined},
			core.Text(value, core.FontWeight(core.Bold)),
			components.Button{Label: "+", OnTap: func() { onDelta(+1) }, Emphasis: components.EmphasisOutlined},
		).Render(ctx)
	})
}
