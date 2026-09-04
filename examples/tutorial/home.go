package tutorial

import (
	"fmt"

	"github.com/rohanthewiz/grmob/components"
	"github.com/rohanthewiz/grmob/core"
)

// Home is the table of contents: title, overall progress, then one Card per
// chapter listing its lessons as tappable rows. It is the Navigator's initial
// route; lessons are pushed on top, so backing out of a lesson always lands
// here with scroll and progress intact (the root frame never leaves the
// stack).
func (t *tutorial) Home(ctx *core.Context) core.View {
	opened, total := t.progress()

	return components.Screen{
		// The whole page scrolls and nothing inside it does, which is
		// exactly the case Screen.Scroll exists for.
		Scroll: true,
		Gap:    16,
		Children: []core.View{
			core.Column(
				core.Gap(4),
				titleText("GrMob Interactive Tutorial"),
				caption("Learn GrMob inside GrMob — every lesson is a live screen you can poke at."),
			),
			progressCard(opened, total),
			t.chapterCards(ctx),
		},
	}
}

// titleText is the theme's Title typography; local to home because lesson
// screens draw their titles through the scaffold in lesson_screen.go.
func titleText(s string) core.View {
	return core.ComponentFunc(func(ctx *core.Context) *core.Node {
		return core.Text(s, core.UseStyle(ctx.Theme().Typography.Title)).Render(ctx)
	})
}

// progressCard shows how far the reader has gotten. The label carries the
// numbers rather than the bar alone: ProgressBar announces its percentage to
// assistive tech, but sighted readers want the count, and "3 of 5" is the
// count the visited map actually measures.
func progressCard(opened, total int) core.View {
	return core.Card(
		core.Gap(8),
		caption(fmt.Sprintf("%d of %d lessons opened", opened, total)),
		components.ProgressBar{
			Value:              float64(opened) / float64(total),
			AccessibilityLabel: "Tutorial progress",
		},
	)
}

// chapterCards renders the curriculum. Chapter grouping is re-derived from
// each entry's ChapterNum while walking the flat index once — the flat index
// is the source of truth for order and IDs, and home just folds it back into
// sections.
func (t *tutorial) chapterCards(ctx *core.Context) core.View {
	var cards []core.View
	var rows []core.PropsAndChildren
	flush := func(ci int) {
		if len(rows) == 0 {
			return
		}
		ch := Chapters[ci]
		cards = append(cards, core.Card(
			core.Gap(2),
			core.Text(fmt.Sprintf("%s  Chapter %d — %s", ch.Icon, ci+1, ch.Title),
				core.FontWeight(core.Bold)),
			caption(ch.Summary),
			core.Column(rows...),
		))
		rows = nil
	}

	current := 0
	for _, e := range flatLessons {
		if e.ChapterNum-1 != current {
			flush(current)
			current = e.ChapterNum - 1
		}
		rows = append(rows, t.lessonRow(ctx, e))
	}
	flush(current)

	items := append([]core.PropsAndChildren{core.Gap(16)}, asAny(cards)...)
	return core.Column(items...)
}

// lessonRow is one tappable line of the contents. The row is keyed by lesson
// ID: rows never reorder today, but the "opened" badge appears per-row as
// state changes, and keyed rows keep those patches addressed to the right
// occupant if a later phase inserts lessons.
func (t *tutorial) lessonRow(ctx *core.Context, e lessonEntry) core.View {
	entry := e // capture a copy: the range variable's fields feed closures below
	var trailing core.View
	if t.visited.Get()[entry.ID] {
		trailing = components.Badge{Text: "opened", Variant: components.VariantSuccess}
	} else {
		trailing = core.ComponentFunc(func(ctx *core.Context) *core.Node {
			return core.Text("›",
				core.FontSize(20),
				core.TextColor(ctx.Theme().Colors.TextSecondary),
			).Render(ctx)
		})
	}

	return core.Keyed("lesson-"+entry.ID, components.ListRow{
		Leading: core.ComponentFunc(func(ctx *core.Context) *core.Node {
			return core.Text(entry.ID,
				core.TextColor(ctx.Theme().Colors.Primary),
				core.FontWeight(core.Bold),
			).Render(ctx)
		}),
		Title:    entry.Title,
		Subtitle: entry.Summary,
		Trailing: trailing,
		// Opening is what "visited" means, so the tap both records progress
		// and navigates (open does both, and reports the route to the host
		// for the address bar). All of it happens in the handler — state
		// writes never belong in render.
		OnTap:              func() { t.open(ctx, entry.Index, false) },
		AccessibilityLabel: fmt.Sprintf("Lesson %s, %s", entry.ID, entry.Title),
		AccessibilityHint:  "Opens the lesson",
	})
}

// asAny widens []core.View to the []core.PropsAndChildren the container
// builders take. A Go slice does not convert element-wise on its own, and the
// loop is clearer at one call site than a generics dance.
func asAny(views []core.View) []core.PropsAndChildren {
	out := make([]core.PropsAndChildren, len(views))
	for i, v := range views {
		out[i] = v
	}
	return out
}
