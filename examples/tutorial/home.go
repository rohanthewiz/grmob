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

	// The page is a core.List, not a Column inside Screen.Scroll, and the
	// reason is a measurement on both hosts. SwiftUI and Compose each build a
	// view per node for a Scroll's whole content, so a contents screen of 49
	// two-line rows paid for all 49 before it could draw the four that fit;
	// a List materializes the rows on screen.
	//
	//	iOS       7.4s -> 1.7s     the floor examples/mobileapp sets
	//	Android   6.1s -> 5.0s     of which 2.5s is the tree, not the views
	//
	// The two hosts agree on the direction and disagree on how much of the
	// launch it was, which is worth knowing before reaching for this on a
	// third screen. On iOS the scrolled Column WAS the launch. On Android it
	// is 29% of the screen's cost: the same binary with these cards taken off
	// the screen launches in 2.5s, and the rest is what happens to 423KB of
	// JSON on its way across the bridge — paid for all 49 rows whether or not
	// Compose composes them. See android/device/launch.sh for the four arms
	// and TestHomeTreeSize for the bytes; LiveMapUITests carries the iOS
	// readings and what else was in that number.
	//
	// This is also what the tutorial teaches one lesson over ("Use Scroll for
	// short content and core.List for long data-driven collections"), applied
	// to itself: 49 rows is not short content.
	page := []core.PropsAndChildren{
		core.Gap(16), core.FlexGrow(1),
		core.Keyed("title", core.Column(
			core.Gap(4),
			titleText("GrMob Interactive Tutorial"),
			caption("Learn GrMob inside GrMob — every lesson is a live screen you can poke at."),
		)),
		core.Keyed("progress", progressCard(opened, total)),
	}
	page = append(page, asAny(t.chapterCardViews(ctx))...)

	// No Padding(0) here any more: the scaffold drops its own inset when its
	// whole content is a scrolling page, so the List insets this screen once.
	// See components.Screen, "A scrolling child is the page".
	return components.Screen{
		Fill:     true,
		Children: []core.View{core.List(page...)},
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

// chapterCardViews renders the curriculum, one Card per chapter. Chapter
// grouping is re-derived from each entry's ChapterNum while walking the flat
// index once — the flat index is the source of truth for order and IDs, and
// home just folds it back into sections.
//
// A slice rather than one Column, because the cards are children of Home's
// core.List and that is where the laziness lives: a Column of cards inside the
// List would be one child, and one child is either materialized whole or not
// at all. Keyed for the same reason every List child is — the lazy containers
// on both natives keep row state attached to the key across changes.
func (t *tutorial) chapterCardViews(ctx *core.Context) []core.View {
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

	for i, c := range cards {
		cards[i] = core.Keyed(fmt.Sprintf("chapter-%d", i), c)
	}
	return cards
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
				// Pinned for the reason the leading number is, and the
				// simulator showed the same symptom at the other end of the
				// row: a chevron compressed below one glyph is a chevron
				// sliced down the middle. One character is exactly the case
				// where "shrink by a proportion of your base" has nothing
				// left to give — and also the case the min-content floor now
				// covers, since a single glyph has no break opportunity in it.
				core.FlexShrink(0),
			).Render(ctx)
		})
	}

	return core.Keyed("lesson-"+entry.ID, components.ListRow{
		Leading: core.ComponentFunc(func(ctx *core.Context) *core.Node {
			return core.Text(entry.ID,
				core.TextColor(ctx.Theme().Colors.Primary),
				core.FontWeight(core.Bold),
				// Pinned, and the first simulator run of this app is why.
				//
				// A ListRow is a Row whose centre column claims the slack
				// (FlexGrow(1)), and these two-line titles overflow it on a
				// phone — at which point the flex deficit is shared out among
				// the children that can shrink. On the web and on Compose this
				// number survives anyway, because CSS gives every flex item
				// `min-width: auto` and will not compress one below its
				// min-content width. The iOS solver had no such floor, so
				// "4.12" was compressed to the width of one glyph and wrapped
				// down the side of the row as 4 / . / 1 / 2.
				//
				// That floor exists on iOS now (GrMobMinContent), and a run
				// with the two declarations below removed renders the numbers
				// whole — so this is no longer load-bearing for the symptom.
				// It stays because it says something stronger and true on
				// every host: a row number is four characters wide and that is
				// not negotiable, where the floor only promises "no narrower
				// than the content", which for wrappable text is less. The
				// declaration should always have been here.
				core.FlexShrink(0),
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
