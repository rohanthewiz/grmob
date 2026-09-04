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
	"github.com/rohanthewiz/grmob/components"
	"github.com/rohanthewiz/grmob/core"
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
// palette has no role for it. These two are the surface and its default ink;
// the syntax colours layered on top are in highlight.go. Together they are
// the only hard-coded colors in the tutorial chrome.
//
// The pair is Darcula's own editor background and default foreground, so the
// surface and the token colours come from one scheme rather than from two
// that happen to both be dark.
const (
	codeBg  = "#2B2B2B" // Darcula editor background
	codeInk = "#A9B7C6" // Darcula default foreground
)

// codeBlock renders a Go snippet, syntax-highlighted.
//
// It is a core.TextGrid — the node type built for "rows of styled runs" —
// because that is the shape highlighting needs: a code line is several
// colours, so the unit carrying a colour has to be smaller than a line. The
// grid's Style is the surface and the default ink; every run that is not
// default ink comes from highlightGo, which lexes the snippet with
// go/scanner.
//
// Three things this used to do by hand come free with the grid, and are worth
// naming because their absence here is not an oversight:
//
//   - Monospace. Style still has no font-family prop, and a TextGrid does not
//     need one: it is fixed-pitch on every target by construction. The old
//     block's own comment recorded this as a limitation it was living with.
//   - No wrapping, and sideways scrolling when a snippet is wider than the
//     phone. A wrapped code line restarts at column zero, which reads as a
//     new statement at the outermost indent — so wrapping destroys exactly
//     the structure indentation exists to show. The grid's rows are
//     single-line and its chassis scrolls horizontally, which is what the old
//     block spelled out per-line with WhiteSpace("nowrap") plus an Overflow.
//   - Indentation, and blank lines that keep their height. A <span> in
//     ordinary flow collapses leading spaces, which is why the old block
//     substituted non-breaking spaces and padded empty lines with one of
//     them; a grid row preserves its spaces and an empty row still takes a
//     line.
func codeBlock(code string) core.View {
	return core.TextGrid(highlightGo(code),
		core.BackgroundColor(codeBg),
		core.Padding(14),
		core.BorderRadius(10),
		core.FontSize(13),
		core.TextColor(codeInk),
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
func keyPoints(points ...string) core.View {
	return core.ComponentFunc(func(ctx *core.Context) *core.Node {
		t := ctx.Theme()
		items := []core.PropsAndChildren{
			core.Gap(6),
			core.Text("Key points", core.UseStyle(t.Typography.Subtitle)),
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
