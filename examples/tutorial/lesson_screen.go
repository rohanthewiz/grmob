package tutorial

import (
	"fmt"

	"github.com/rohanthewiz/grmob/components"
	"github.com/rohanthewiz/grmob/core"
)

// lessonRoute builds the Navigator route for one lesson: the scaffold (top
// bar, title, prev/next) around the lesson's own Body.
//
// Prev/next open with replace=true (core.Replace rather than Push): the reader walking a chapter
// should not build a back-stack ten lessons deep — the stack stays
// [contents, current lesson], and the system back gesture always means
// "back to contents". Replace also swaps the stack frame, and since a frame
// owns its hook namespace, each lesson's demo state starts fresh and is
// discarded on leaving — no cross-lesson bleed, with no cleanup code.
func (t *tutorial) lessonRoute(index int) func(*core.Context) core.View {
	e := flatLessons[index]
	return func(ctx *core.Context) core.View {
		return components.Screen{
			Scroll: true,
			Gap:    16,
			Children: []core.View{
				t.lessonTopBar(ctx, e),
				lessonHeader(e),
				e.Body(ctx),
				components.Separator{},
				t.lessonNav(ctx, e),
			},
		}
	}
}

// lessonTopBar: a ghost back button on the left, the chapter tag on the
// right. Ghost, for the same reason as social's tab bar — a filled button
// here would out-shout the lesson content it sits above.
func (t *tutorial) lessonTopBar(ctx *core.Context, e lessonEntry) core.View {
	return core.Row(
		core.AlignItemsProp(core.AlignItemsCenter),
		components.Button{
			Label:    "‹ Contents",
			OnTap:    func() { t.toContents(ctx) },
			Emphasis: components.EmphasisGhost,
		},
		core.Box(core.FlexGrow(1)), // slack, so the tag pins right
		components.Badge{Text: fmt.Sprintf("Chapter %d · %s", e.ChapterNum, e.ChapterTitle)},
	)
}

func lessonHeader(e lessonEntry) core.View {
	return core.ComponentFunc(func(ctx *core.Context) *core.Node {
		th := ctx.Theme()
		return core.Column(
			core.Gap(4),
			core.Text(fmt.Sprintf("%s  %s", e.ID, e.Title), core.UseStyle(th.Typography.Title)),
			core.Text(e.Summary,
				core.UseStyle(th.Typography.Body),
				core.TextColor(th.Colors.TextSecondary),
			),
		).Render(ctx)
	})
}

// lessonNav is the footer: Prev when there is one, then Next — or, on the
// last lesson, Finish, which pops back to the contents. Next goes through
// the same open as a contents-row tap, so progress and the reported route
// are identical whichever door a lesson is entered through.
//
// The absent Prev on the first lesson is a plain Go nil, not core.If: a
// false If still leaves an empty Fragment child for the row to space
// against, while the container builders skip a nil outright (the MaybeProp
// contract) — and MaybeProp itself can't be used here because its argument
// is evaluated eagerly, which would index flatLessons[-1].
func (t *tutorial) lessonNav(ctx *core.Context, e lessonEntry) core.View {
	var prev core.View
	if e.Index > 0 {
		target := flatLessons[e.Index-1]
		prev = components.Button{
			Label:    "‹ Prev",
			Emphasis: components.EmphasisOutlined,
			OnTap:    func() { t.open(ctx, target.Index, true) },
		}
	}

	var next core.View
	if e.Index < len(flatLessons)-1 {
		target := flatLessons[e.Index+1]
		next = components.Button{
			Label: "Next ›",
			OnTap: func() { t.open(ctx, target.Index, true) },
		}
	} else {
		next = components.Button{
			Label:   "Finish ✓",
			Variant: components.VariantSuccess,
			OnTap:   func() { t.toContents(ctx) },
		}
	}

	return core.Row(
		core.Gap(8),
		prev,
		core.Box(core.FlexGrow(1)),
		next,
	)
}
