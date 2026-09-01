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
// building blocks (prose, code blocks, demo panels); chapterN.go files hold
// the content. Later chapters append to Chapters in lesson.go.
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
	}
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
func (t *tutorial) markVisited(id string) {
	old := t.visited.Get()
	next := make(map[string]bool, len(old)+1)
	maps.Copy(next, old)
	next[id] = true
	t.visited.Set(next)
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
