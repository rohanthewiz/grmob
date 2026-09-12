// Package tutorial is GrMob's interactive tutorial — a GrMob app that teaches
// GrMob. Each lesson is a live screen: an explanation, the code under
// discussion, and a "try it" panel wired to real state and real callbacks, so
// the reader learns every concept by tapping it rather than by reading about
// it.
//
// Being an ordinary app in examples/ is the whole design. The same package:
//
//   - runs in a browser via the wasm host (point wasm/main.go's example
//     import at this package),
//   - ships natively through the mobile bridge (the init below registers it,
//     exactly as examples/todoapp does),
//   - and is driven headless by app_test.go through render.Manager, which is
//     what keeps every demo honest — a lesson whose demo breaks fails CI.
//
// Structure: lesson.go defines the Lesson/Chapter model and the flattened
// lesson index; home.go is the table of contents; lesson_screen.go is the
// scaffold every lesson renders inside; widgets.go holds the tutorial's own
// building blocks (prose, code blocks, demo panels) and highlight.go the
// go/scanner-based syntax highlighting the code blocks use; chapterN.go files
// hold the content. Later chapters append to Chapters in lesson.go.
package tutorial

import (
	"maps"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/mobile"
)

func init() {
	mobile.Register(core.NewContext(), App)
}

// AppName exists to be bindable. gobind only links a bound package when it
// references at least one bindable exported symbol; App is not bindable
// (function-typed parameters are unsupported), and without this the package —
// including the init that registers the app — would be dropped from the
// native library, leaving the bridge with a nil manager. Same contract as
// examples/todoapp.
func AppName() string { return "GrMob Tutorial" }

// tutorial bundles the state that must outlive any single screen — today just
// the reader's progress — plus the navigation helpers every screen shares.
// Screens are methods on it, so route closures capture one receiver instead
// of a growing argument list.
type tutorial struct {
	// visited maps lesson ID ("1.2") → opened. It drives the progress bar and
	// the per-row "opened" badge on the contents screen. "Opened" rather than
	// "completed" is deliberate for now: there is nothing to grade, and a
	// completion quiz is a later phase's concern.
	visited core.State[map[string]bool]
	// current is the lesson ID on screen, or "" on the contents. It exists
	// for deep links (deeplink.go): the inbound route handler needs to know
	// whether to Push or Replace, and whether the link names the lesson
	// already showing. Written only by the navigation doors — open,
	// toContents, goTo — so it is always the stack's top frame by another
	// name.
	current core.State[string]
	// expanded maps a chapter's zero-based index → whether its card is open
	// on the contents screen. A chapter that is not in the map is shut.
	//
	// # Why the contents screen does not own this
	//
	// components.Accordion owns its own expansion and is the right answer for
	// a single section; components.Collapse's doc says where that stops, and
	// this is the case it names. Eight cards means eight independent NewStates
	// that the screen cannot address — no way to open the chapter a reader just
	// came back from, and no way to shut them all. Holding the map here makes
	// the expansion a fact about the session rather than about eight widgets,
	// which is also what lets `open` set it.
	//
	// # Why it is seeded open rather than empty
	//
	// A contents screen whose every chapter is shut is eight headers and
	// nothing else, and a reader arriving for the first time has no reason to
	// guess which one to press. Chapter 1 is the answer for somebody who has
	// opened nothing, and it stops being the answer the moment they open a
	// lesson — `open` expands whatever chapter they landed in, so backing out
	// lands on a card already showing the row they came from.
	expanded core.State[map[int]bool]
}

// App is the root view: a Navigator whose initial route is the table of
// contents; lessons are pushed on top of it.
//
// The progress state lives in a named scope on the context *above* the
// Navigator, not in the home route's frame. A frame's hooks die with the
// frame, and although the root frame never leaves the stack today, progress
// is conceptually session state, not screen state — putting it above the
// Navigator means a future Reset (or a second entry point straight into a
// lesson) cannot orphan it. This is the ctx.Scope("session") shape the
// navigation docs recommend for state that must outlive frames.
func App(ctx *core.Context) core.View {
	sctx := ctx.Scope("tutorial-session")
	t := &tutorial{
		visited: core.NewState(sctx, map[string]bool{}),
		current: core.NewState(sctx, ""),
		// Chapter 1 open, the rest shut. See the field's doc for why this is a
		// seed rather than an empty map.
		expanded: core.NewState(sctx, map[int]bool{0: true}),
	}
	// Same scope, for the same reason: the route handler moves frames, so
	// it must outlive them.
	t.useDeepLinks(sctx)
	return core.Navigator(t.Home)
}

// markVisited records that a lesson has been opened. It copies the map before
// writing: earlier render passes hold the previous value, and the reconciler
// diffs old tree against new, so state must be replaced, never mutated in
// place — the same immutability rule every example follows for slices.
//
// Only ever called from event handlers (a row tap, the Next button), never
// during a render pass: Set marks the tree dirty, and a Set inside render
// would schedule renders forever.
//
// # It also opens the lesson's chapter, and that is deliberate
//
// The contents screen's chapter cards are collapsed by default, so backing out
// of a lesson has to land on a card that is showing the row the reader came
// from. The obvious place for that is the door they went through — and there
// are two doors, `open` for the app's own controls and `goTo` for an inbound
// deep link, which is exactly the drift this file already worries about one
// function down ("so the address bar cannot drift from the screen because one
// path forgot to report"). Putting it here instead makes it true by
// construction: every path that records a lesson as opened opens its chapter,
// because recording it *is* opening it.
//
// A lesson ID that names no lesson leaves the expansion alone rather than
// guessing a chapter. Nothing produces one today — both callers hold a
// lessonEntry — and resolveRoute is the function that would have to start
// lying for it to happen.
func (t *tutorial) markVisited(id string) {
	old := t.visited.Get()
	next := make(map[string]bool, len(old)+1)
	maps.Copy(next, old)
	next[id] = true
	t.visited.Set(next)

	if e, ok := resolveRoute(id); ok {
		t.setExpanded(e.ChapterNum-1, true)
	}
}

// setExpanded opens or shuts one chapter's card on the contents screen.
//
// Replaced rather than mutated, for markVisited's reason one function up:
// earlier render passes hold the previous map and the reconciler diffs the old
// tree against the new, so a map edited in place would make the two agree about
// a change that has not been rendered yet.
//
// Shut is a deletion rather than a false, so the map holds only what is open.
// Nothing reads the difference today; it is what keeps the map from growing a
// permanent entry per chapter the reader ever closed.
func (t *tutorial) setExpanded(chapter int, open bool) {
	old := t.expanded.Get()
	next := make(map[int]bool, len(old)+1)
	maps.Copy(next, old)
	if open {
		next[chapter] = true
	} else {
		delete(next, chapter)
	}
	t.expanded.Set(next)
}

// progress reports how many lessons have been opened, out of how many exist.
func (t *tutorial) progress() (opened, total int) {
	seen := t.visited.Get()
	for _, e := range flatLessons {
		if seen[e.ID] {
			opened++
		}
	}
	return opened, len(flatLessons)
}
