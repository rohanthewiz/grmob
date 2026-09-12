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
	// was 29% of the screen's cost: the same binary with these cards taken off
	// the screen launches in 2.5s, and the rest was what happened to 423KB of
	// JSON on its way across the bridge — paid for all 49 rows whether or not
	// Compose composes them.
	//
	// That 423KB is 51KB now, and the Android launch 3.5s: 92% of it was
	// core.Style's zero-valued fields, written out for every node, and the
	// fields are `,omitzero` (the last 2KB of it came off later, when the same
	// tags reached core.EdgeInsets). So the second number above is the reading
	// that prompted the fix rather than the reading today. See
	// android/device/launch.sh for the four arms and the attribution,
	// TestHomeTreeSize for the bytes, and the note above core.Style for the
	// tags; LiveMapUITests carries the iOS readings and what else was in that
	// number.
	//
	// This is also what the tutorial teaches one lesson over ("Use Scroll for
	// short content and core.List for long data-driven collections"), applied
	// to itself: 49 rows is not short content.
	//
	// # And then the rows stopped being on the screen at all
	//
	// The chapter cards collapse now — one open, the rest shut — so the List
	// is materializing eight cards of which seven are a band and a summary.
	// That took the screen from 53,156 bytes to 17,366, which is more than
	// windowing the List over the bridge was sized at, with no protocol change
	// and no placeholder children.
	//
	// The List stays, and not out of caution: the two answers are to different
	// questions. Collapsing decides what is *on the screen*, and a reader who
	// opens a forty-lesson chapter is back to a screen of forty rows —
	// windowing is what keeps that from being paid for all at once. What has
	// changed is that the tutorial is no longer the screen arguing for it.
	// TestHomeTreeSize carries the three measurements.
	//
	// Windowing was then re-profiled against this screen rather than the one it
	// was sized on, and declined: 42-50% of 17,366 bytes is ~7-9KB, about 55ms
	// on the one emulator anybody has measured, against a protocol change and
	// placeholder children in four renderers. See ai_docs/plans/non_goals.md
	// and TestWhatWindowingWouldSave, which asserts its own share now — the
	// old table was quoted at 66-76% for a session after this collapse made it
	// false, because nothing checked it.
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
	seen := t.visited.Get()
	open := t.expanded.Get()

	var cards []core.View
	var rows []core.PropsAndChildren
	var opened int
	flush := func(ci int) {
		if len(rows) == 0 {
			return
		}
		ch := Chapters[ci]
		chapter := ci // captured per card: the toggle below outlives this loop

		// The accessible name of both the heading and the button. The icon is
		// not in it: a CollapseBand names its control from Group.Label and
		// treats Content as presentational, so the glyph is drawn and never
		// announced — which is what "decoration only" on Chapter.Icon has
		// always claimed and nothing enforced until the title became a button.
		label := fmt.Sprintf("Chapter %d — %s", ci+1, ch.Title)

		// "6 lessons" until the reader has opened one, then "2 of 6". The
		// count is outside the button deliberately: a CollapseBand's Content
		// is inside the control, and a button's children are presentational,
		// so a progress count put there would stop being announced at exactly
		// the moment it started being worth announcing.
		count := fmt.Sprintf("%d lessons", len(rows))
		if opened > 0 {
			count = fmt.Sprintf("%d of %d", opened, len(rows))
		}

		card := []core.PropsAndChildren{
			core.Gap(2),
			core.Row(
				core.AlignItemsProp(core.AlignItemsCenter),
				core.Gap(8),
				components.CollapseBand{
					Collapse: components.Collapse{
						IsCollapsed: func(g components.Group) bool { return !open[chapter] },
						OnToggle:    func(g components.Group) { t.setExpanded(chapter, !open[chapter]) },
					},
					Group: components.Group{Key: fmt.Sprintf("chapter-%d", ci), Label: label},
					// The tutorial's own title typing, so collapsing a card did
					// not also restyle it: a CollapseBand with no Content draws
					// the label in Caption weight and secondary ink, which is
					// right for a list band and wrong for a card title.
					Content: []core.View{core.Text(
						chapterBandText(ci),
						core.FontWeight(core.Bold),
					)},
					Style: []core.StyleProp{core.FlexGrow(1)},
				},
				caption(count),
			),
			// The summary stays visible while the card is shut. It is one line
			// per chapter and it is the whole of what a reader has to choose
			// from when the rows are away — collapsing the cards to save the
			// rows and then hiding the description of what is behind them
			// would be a contents screen that contains nothing.
			caption(ch.Summary),
		}
		// The rows, only while open. This is where the payload goes: the
		// lesson rows are 97.4% of this screen's JSON, and a shut chapter puts
		// none of them on the wire. See TestHomeTreeSize.
		if open[chapter] {
			card = append(card, core.Column(rows...))
		}
		cards = append(cards, core.Card(card...))
		rows = nil
		opened = 0
	}

	current := 0
	for _, e := range flatLessons {
		if e.ChapterNum-1 != current {
			flush(current)
			current = e.ChapterNum - 1
		}
		if seen[e.ID] {
			opened++
		}
		// Built even for a shut chapter, and dropped by flush if it stays
		// shut. Rendering a row is building a Go value; what costs anything is
		// the node reaching the wire, and keeping the walk uniform is what
		// keeps the per-chapter count above honest whether or not the card
		// happens to be open.
		rows = append(rows, t.lessonRow(ctx, e))
	}
	flush(current)

	for i, c := range cards {
		cards[i] = core.Keyed(fmt.Sprintf("chapter-%d", i), c)
	}
	return cards
}

// chapterBandText is the words drawn inside a chapter card's disclosure
// button: the icon, then the same "Chapter N — Title" that names the control.
//
// A function rather than a Sprintf at the one call site because the tests need
// the same string to press the band with, and a contents screen whose chapters
// could not be opened from a test would be a screen no lesson test could reach
// past. See openLesson in app_test.go.
func chapterBandText(ci int) string {
	return fmt.Sprintf("%s  Chapter %d — %s", Chapters[ci].Icon, ci+1, Chapters[ci].Title)
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
