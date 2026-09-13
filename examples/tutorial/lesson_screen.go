package tutorial

import (
	"fmt"

	"github.com/rohanthewiz/grmob/comps"
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
		return core.ComponentFunc(func(*core.Context) *core.Node {
			n := comps.Screen{
				Scroll: true,
				Gap:    16,
				Children: []core.View{
					t.lessonTopBar(ctx, e),
					lessonHeader(e),
					e.Body(ctx),
					comps.Separator{},
					t.lessonNav(ctx, e),
				},
			}.Render(ctx)
			// System back (Android) and browser back (web) take the same door
			// as ‹ Contents. Navigator's own pop would leave the stack right
			// but t.current naming the lesson and the address bar still on
			// its hash, so a link back to the same lesson would be ignored as
			// the one already on screen. A claim on the route's root node
			// replaces Navigator's (see core.withSystemBackPop), and Screen
			// builds a fresh node every pass, so writing into it is safe.
			core.OnBack(func() { t.toContents(ctx) }).Apply(ctx, n)
			return n
		})
	}
}

// lessonTopBar: a ghost back button on the left, the chapter tag on the
// right. Ghost, for the same reason as social's tab bar — a filled button
// here would out-shout the lesson content it sits above.
func (t *tutorial) lessonTopBar(ctx *core.Context, e lessonEntry) core.View {
	return core.Row(
		core.AlignItemsProp(core.AlignItemsCenter),
		comps.Button{
			Label:    "‹ Contents",
			OnTap:    func() { t.toContents(ctx) },
			Emphasis: comps.EmphasisGhost,
		},
		core.Box(core.FlexGrow(1)), // slack, so the tag pins right
		comps.Badge{Text: fmt.Sprintf("Chapter %d · %s", e.ChapterNum, e.ChapterTitle)},
	)
}

// lessonHeader is the lesson's own name and one-line summary.
//
// The title is the screen's heading at level 1. A lesson screen has no AppBar
// — the top bar here is a back button and a chapter badge, not a titled bar —
// so this is the top of the outline by construction, and it is the tier
// comps.AppBar would have claimed if there were one. Below it a lesson's
// "Key points" recap takes 2 and any Accordion in the body takes 3, which is
// the whole range the framework fills in without a call site.
func lessonHeader(e lessonEntry) core.View {
	return core.ComponentFunc(func(ctx *core.Context) *core.Node {
		th := ctx.Theme()
		return core.Column(
			core.Gap(4),
			core.Text(fmt.Sprintf("%s  %s", e.ID, e.Title),
				core.UseStyle(th.Typography.Title),
				core.AccessibilityRole(core.RoleHeading),
				core.AccessibilityHeadingLevel(1)),
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
		prev = comps.Button{
			Label:    "‹ Prev",
			Emphasis: comps.EmphasisOutlined,
			OnTap:    func() { t.open(ctx, target.Index, true) },
		}
	}

	var next core.View
	if e.Index < len(flatLessons)-1 {
		target := flatLessons[e.Index+1]
		next = comps.Button{
			Label: "Next ›",
			OnTap: func() { t.open(ctx, target.Index, true) },
		}
	} else {
		next = comps.Button{
			Label:   "Finish ✓",
			Variant: comps.VariantSuccess,
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
