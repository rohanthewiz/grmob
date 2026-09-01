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
// palette has no role for it. These are the only hard-coded colors in the
// tutorial chrome.
const (
	codeBg  = "#22272E" // GitHub dark-dimmed editor background
	codeInk = "#ADBAC7"
)

// codeBlock renders a Go snippet. Two renderer realities shape it:
//
//   - There is no font-family style prop yet, so it cannot be truly
//     monospaced; the dark surface and tighter size carry the "code" reading
//     instead.
//   - It renders one Text per line inside a Column rather than one Text with
//     embedded newlines, because HTML collapses both newlines and leading
//     runs of spaces — the indentation that makes Go readable would vanish in
//     the browser. Leading spaces are swapped for non-breaking spaces
//     (U+00A0), which every target honors, and a blank line becomes a single
//     NBSP so the line still occupies height.
func codeBlock(code string) core.View {
	lines := strings.Split(strings.Trim(code, "\n"), "\n")
	items := []core.PropsAndChildren{
		core.BackgroundColor(codeBg),
		core.Padding(14),
		core.BorderRadius(10),
		core.Gap(2),
	}
	for _, line := range lines {
		display := line
		if trimmed := strings.TrimLeft(line, " "); len(trimmed) < len(line) {
			display = strings.Repeat(" ", len(line)-len(trimmed)) + trimmed
		}
		if display == "" {
			display = " "
		}
		items = append(items, core.Text(display,
			core.FontSize(13),
			core.TextColor(codeInk),
		))
	}
	return core.Column(items...)
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
