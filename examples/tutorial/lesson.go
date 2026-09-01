package tutorial

import (
	"fmt"

	"github.com/rohanthewiz/grmob/core"
)

// Lesson is one teachable unit: a titled screen of prose, code, and a live
// demo. Body is a plain view function — the same shape as a Navigator route —
// so a lesson may use hooks freely: every lesson renders inside its own stack
// frame (see lessonRoute), which gives it a private hook namespace, and
// moving between lessons swaps frames, so demo state resets per lesson by
// construction rather than by cleanup code.
type Lesson struct {
	Title   string
	Summary string // one line under the title, and the row subtitle on the contents screen
	Body    func(ctx *core.Context) core.View
}

// Chapter groups lessons under a theme. Chapters are numbered by position in
// Chapters, lessons by position within their chapter — IDs like "1.2" are
// derived, not stored, so they cannot drift from the actual order.
type Chapter struct {
	Title   string
	Icon    string // a glyph for the contents screen — decoration only
	Summary string
	Lessons []Lesson
}

// Chapters is the whole curriculum in reading order. Each phase of the
// tutorial's construction appends one entry; content lives in chapterN.go
// files so a chapter is reviewable as a unit.
var Chapters []Chapter

// lessonEntry is one row of the flattened, ordered lesson index — the
// structure navigation actually walks. Prev/next is a flat walk (the last
// lesson of chapter 1 precedes the first of chapter 2), so the index is
// flat; chapter grouping is a presentation concern the contents screen
// re-derives.
type lessonEntry struct {
	Lesson
	ID           string // "2.3": chapter and lesson ordinals, 1-based
	ChapterTitle string
	ChapterNum   int
	Index        int // position in flatLessons; what prev/next arithmetic uses
}

// flatLessons is built once at package init. Package-level rather than
// per-App because the curriculum is immutable data, like todoapp's
// filterLabels.
var flatLessons []lessonEntry

// The curriculum is assembled in init rather than in the two variables'
// initializer expressions because lesson 8.5 reports the curriculum's own
// size: its Body reads Chapters and flatLessons, and a var initializer that
// transitively references itself is an initialization cycle to the compiler
// even when the reference sits inside a closure that only runs at render
// time. init functions run after all package variables exist, so building
// here breaks the cycle while keeping both variables immutable-after-init.
func init() {
	Chapters = []Chapter{
		chapter1(),
		chapter2(),
		chapter3(),
		chapter4(),
		chapter5(),
		chapter6(),
		chapter7(),
		chapter8(),
	}
	for ci, ch := range Chapters {
		for li, l := range ch.Lessons {
			flatLessons = append(flatLessons, lessonEntry{
				Lesson:       l,
				ID:           fmt.Sprintf("%d.%d", ci+1, li+1),
				ChapterTitle: ch.Title,
				ChapterNum:   ci + 1,
				Index:        len(flatLessons),
			})
		}
	}
}
