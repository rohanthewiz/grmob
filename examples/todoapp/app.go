// Package todoapp is a self-contained todo application built on the grmob
// core widgets. It follows the same integration contract as examples/mobileapp:
// the init below registers the root view with the mobile bridge, so swapping
// this package into ios/build.sh or android/build.sh ships it natively, and
// the _test.go beside it drives the exact call sequence the native shells make
// for instant feedback without a simulator.
//
// Data persists across launches via an embedded bytdb database (store.go).
// The mutation helpers below are the single choke point for writes, and each
// one write-throughs to the store; the store's snapshot at open seeds the
// initial state. With no data directory registered (mobile.SetDataDir — web
// preview, bare tests) the store is nil and the app runs in-memory unchanged.
package todoapp

import (
	"fmt"
	"strings"

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
// native library, leaving the bridge with a nil manager.
func AppName() string { return "GrMob Todo" }

// Todo is a plain value, not a pointer: mutation helpers copy the slice and
// replace items wholesale, so the reconciler always diffs against an
// unaliased previous tree. Sharing pointers across renders would let a
// handler mutate state the previous render pass already captured.
type Todo struct {
	ID    int
	Title string
	Done  bool
}

// Filter values double as indices into the filter-bar labels, which keeps the
// bar renderable with a single For loop instead of three near-identical rows.
const (
	filterAll = iota
	filterActive
	filterDone
)

var filterLabels = []string{"All", "Active", "Done"}

// Palette. Kept as named constants rather than a custom Theme because the
// app otherwise leans on the default theme's component styles; only accents
// and de-emphasis colors are ours.
const (
	colorDim       = "#3C3C4399" // secondary text (iOS systemGray-ish)
	colorAccent    = "#E8F0FE"   // selected filter chip background
	colorAccentInk = "#0B57D0"   // selected filter chip label — readable on colorAccent
	colorDanger    = "#B3261E"   // destructive-action background (delete, clear)
	colorHair      = "#E5E5EA"   // hairline divider
)

// App is the root view. All state lives here, in the root component, and is
// passed down as values + closures. This is deliberate: grmob hooks are
// positional slots on the Context (React rules apply), so per-row NewState
// calls inside a list that grows, shrinks, or reorders would silently read
// another row's slot. Rows below are therefore pure functions of their Todo.
func App(ctx *core.Context) core.View {
	// Lazy-open the persistent store (no-op nil when no data dir is set) and
	// seed state from its snapshot, synchronously: the persisted rows are in
	// the initial tree, where a hooks.UseEffect load (async by design) would
	// mount empty and patch the rows in a frame later. NewState only consumes
	// the initial value on the context's first pass, so the snapshot call is
	// wasted work on re-renders — but it's a copy of an in-memory slice, not
	// a query, and the store methods are cheap after open.
	st := openStore()
	storedTodos, storedNextID := st.snapshot()

	todos := core.NewState(ctx, storedTodos)
	draft := core.NewState(ctx, "")
	filter := core.NewState(ctx, filterAll)
	// Monotonic ID source, resumed from the store so IDs stay unique across
	// launches. Indices can't serve as identities because deletion shifts
	// them, which would break both row keys and toggle targets.
	nextID := core.NewState(ctx, storedNextID)

	// --- Mutations -------------------------------------------------------
	// Every write goes through one of these helpers, and each builds a fresh
	// slice before Set. Set marks the context dirty and requests a render, so
	// the UI below never needs manual refresh plumbing. Each helper also
	// write-throughs to the store with the matching row operation — memory
	// first (the UI must never wait on or be blocked by a disk error), disk
	// second, no branching because the store methods are nil-safe.

	addTodo := func() {
		title := strings.TrimSpace(draft.Get())
		if title == "" {
			return // ignore blank submissions rather than surfacing an error state
		}
		created := Todo{ID: nextID.Get(), Title: title}
		todos.Set(append(append([]Todo{}, todos.Get()...), created))
		nextID.Set(nextID.Get() + 1)
		draft.Set("") // clear the input so consecutive adds need no manual erase
		st.add(created)
	}

	setDone := func(id int, done bool) {
		next := make([]Todo, len(todos.Get()))
		copy(next, todos.Get())
		for i := range next {
			if next[i].ID == id {
				next[i].Done = done
			}
		}
		todos.Set(next)
		st.setDone(id, done)
	}

	removeTodo := func(id int) {
		next := make([]Todo, 0, len(todos.Get()))
		for _, t := range todos.Get() {
			if t.ID != id {
				next = append(next, t)
			}
		}
		todos.Set(next)
		st.remove(id)
	}

	clearDone := func() {
		next := make([]Todo, 0, len(todos.Get()))
		for _, t := range todos.Get() {
			if !t.Done {
				next = append(next, t)
			}
		}
		todos.Set(next)
		st.clearDone()
	}

	// --- Derived values --------------------------------------------------
	// Computed on every pass from the single source of truth instead of being
	// stored, so they can never drift out of sync with the todo list.

	remaining, doneCount := 0, 0
	for _, t := range todos.Get() {
		if t.Done {
			doneCount++
		} else {
			remaining++
		}
	}

	visible := make([]Todo, 0, len(todos.Get()))
	for _, t := range todos.Get() {
		switch filter.Get() {
		case filterActive:
			if !t.Done {
				visible = append(visible, t)
			}
		case filterDone:
			if t.Done {
				visible = append(visible, t)
			}
		default:
			visible = append(visible, t)
		}
	}

	emptyMessage := "No tasks yet — add one above."
	switch filter.Get() {
	case filterActive:
		emptyMessage = "No active tasks."
	case filterDone:
		emptyMessage = "No completed tasks."
	}

	itemWord := "items"
	if remaining == 1 {
		itemWord = "item"
	}

	// --- View ------------------------------------------------------------

	return core.SafeArea(
		core.Column(
			core.FlexGrow(1),
			core.Gap(12),

			core.Text("Todos", core.UseStyle(core.Style{
				FontSize:   28,
				FontWeight: core.Bold,
			})),

			// Entry row. The Input is fully controlled (value in, onChange
			// out); the keyboard's return/done key and the Add button are
			// two paths to the same commit.
			core.Row(
				core.Gap(8),
				core.InputWithSubmit(draft.Get(), "What needs doing?",
					func(v string) { draft.Set(v) },
					addTodo,
					core.FlexGrow(1),
				),
				core.Button("Add", addTodo,
					core.AccessibilityHint("Adds the task typed in the field"),
				),
			),

			filterBar(filter.Get(), func(i int) { filter.Set(i) }),

			// Hairline separator, decorative only — hidden from screen readers.
			core.Box(
				core.Height("1px"),
				core.BackgroundColor(colorHair),
				core.AccessibilityHidden(),
			),

			// The list is virtualized (LazyColumn / LazyVStack natively), so
			// it stays cheap even with many rows. Rows are keyed by todo ID:
			// the reconciler matches children by index and replaces a slot
			// when keys differ, which keeps rows visually correct across
			// insert/delete at the cost of transient native state — fine for
			// rows that hold none.
			core.IfElse(len(visible) == 0,
				core.Text(emptyMessage, core.UseStyle(core.Style{
					FontSize:  14,
					TextColor: colorDim,
				})),
				core.List(
					core.FlexGrow(1),
					core.For(visible, func(t Todo, _ int) core.View {
						return todoRow(t, setDone, removeTodo)
					}),
				),
			),

			// Footer: live count plus bulk-clear, which only appears once it
			// has something to act on.
			core.Row(
				core.Gap(8),
				core.Text(fmt.Sprintf("%d %s left", remaining, itemWord), core.UseStyle(core.Style{
					FontSize:  13,
					TextColor: colorDim,
				}), core.FlexGrow(1)),
				core.If(doneCount > 0,
					// Same destructive treatment as the per-row delete: the
					// red glyph color was invisible against the Button base's
					// blue background.
					core.Button("Clear completed", clearDone, core.UseStyle(core.Style{
						FontSize:   13,
						TextColor:  "#FFFFFF",
						Background: colorDanger,
					})),
				),
			),
		),
	)
}

// filterBar renders the three filter chips from one loop. The active chip is
// distinguished by style only, so switching filters patches two backgrounds
// instead of restructuring the row.
func filterBar(active int, onSelect func(int)) core.View {
	return core.Row(
		core.Gap(8),
		core.For(filterLabels, func(label string, i int) core.View {
			styles := []core.StyleProp{
				core.FontSize(13),
				core.Transition(200, core.EaseInOut),
				core.AccessibilityHint("Filters the task list"),
			}
			accLabel := "Show " + strings.ToLower(label) + " tasks"
			if i == active {
				// The theme's Button base paints a white label; on the pale
				// accent background that is illegible, so the selected chip
				// overrides both colors together.
				styles = append(styles,
					core.BackgroundColor(colorAccent),
					core.TextColor(colorAccentInk),
				)
				accLabel += ", selected"
			}
			styles = append(styles, core.AccessibilityLabel(accLabel))
			return core.Keyed("filter-"+label,
				core.Button(label, func() { onSelect(i) }, styles...),
			)
		}),
	)
}

// todoRow is a pure function of its Todo — no hooks, per the positional-slot
// constraint explained on App. Completion is conveyed by dimming the title;
// the Style struct has no strikethrough field yet.
func todoRow(t Todo, setDone func(int, bool), remove func(int)) core.View {
	titleColor := "#000000"
	if t.Done {
		titleColor = colorDim
	}

	// Capture the ID, not the loop variable's slice position: the closures
	// outlive this pass and must still address the right todo after the
	// visible slice is rebuilt under a different filter.
	id := t.ID

	return core.Keyed(fmt.Sprintf("todo-%d", t.ID), core.Row(
		core.Padding(8),
		core.Gap(10),
		core.Transition(200, core.EaseInOut),
		core.AccessibilityLabel(rowAccessibilityLabel(t)),

		core.Checkbox(t.Done, func(v bool) { setDone(id, v) }),
		core.Text(t.Title, core.UseStyle(core.Style{
			FontSize:  16,
			TextColor: titleColor,
		}), core.FlexGrow(1)),
		// Destructive affordance: the theme's Button base is a medium blue,
		// which both hides the red glyph and reads as a primary action — so
		// the delete button overrides the pair, white glyph on danger red.
		core.Button("✕", func() { remove(id) },
			core.FontSize(13),
			core.TextColor("#FFFFFF"),
			core.BackgroundColor(colorDanger),
			core.AccessibilityLabel("Delete "+t.Title),
		),
	))
}

func rowAccessibilityLabel(t Todo) string {
	if t.Done {
		return t.Title + ", completed"
	}
	return t.Title + ", not completed"
}
