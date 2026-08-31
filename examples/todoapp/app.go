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

	"github.com/rohanthewiz/grmob/components"
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
	// No hairline constant: the rule's tint now comes from the theme's Border
	// role, which is the same #E5E5EA this app used to spell out by hand.
	//
	// No danger constant either. The delete and bulk-clear buttons used to
	// pin #B3261E; they now ask for components.VariantError and take the
	// theme's Error role. That *is* a visible change — white on #B3261E
	// became the contrast-picked ink on #FF3B30 — and it is the point: the
	// app's destructive red is now the same red its error Badge would use,
	// and it retints with a theme swap. Both pairings clear WCAG AA.
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

	// Bulk-clear exists only when something is completed. Declared as a nil
	// core.View rather than wrapped in core.If so the footer row can simply
	// leave its trailing slot empty (an interface nil, not a typed nil — the
	// variable is only ever assigned inside the branch).
	var clearButton core.View
	if doneCount > 0 {
		// Same destructive treatment as the per-row delete, and now literally
		// the same declaration: VariantError names the intent, and the widget
		// picks both the fill and a legible ink from the palette. What this
		// replaced spelled the pair out by hand, in a different shape from the
		// delete button that meant the same thing.
		clearButton = components.Button{
			Label:   "Clear completed",
			OnTap:   clearDone,
			Variant: components.VariantError,
			Style:   []core.StyleProp{core.FontSize(13)},
		}
	}

	// --- View ------------------------------------------------------------

	// Fill (FlexGrow(1) on the column) is load-bearing here, not decoration:
	// the List below asks to grow into the leftover space, and a flex child
	// can only grow inside a parent that has height to give. Without it the
	// column would shrink to its content and the list would never expand.
	//
	// Scroll stays false for the same reason chat's does — the List is
	// virtualized and scrolls itself.
	return components.Screen{
		Fill: true,
		Gap:  12,
		Children: []core.View{
			core.Text("Todos", core.UseStyle(core.Style{
				FontSize:   28,
				FontWeight: core.Bold,
			})),

			// Entry row. The Input is fully controlled (value in, onChange
			// out); the keyboard's return/done key and the Add button are
			// two paths to the same commit — and now literally one handler,
			// since the button inherits OnSubmit rather than repeating it.
			// Gap is unset: the widget defaults to the theme's SM step,
			// which is the 8 this row used to spell out.
			components.InputRow{
				Value:       draft.Get(),
				Placeholder: "What needs doing?",
				OnChange:    func(v string) { draft.Set(v) },
				OnSubmit:    addTodo,
				Button: components.Button{
					Label:             "Add",
					AccessibilityHint: "Adds the task typed in the field",
				},
			},

			filterBar(filter.Get(), func(i int) { filter.Set(i) }),

			// Hairline separator. The widget owns the thickness, the tint and
			// the decorative-only accessibility treatment — the tint by
			// reading the theme's Border role, so this app's rule retints with
			// a theme swap instead of staying pinned to a local constant.
			components.Separator{},

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
			// has something to act on. A ListRow whose Trailing is simply nil
			// when there is nothing to clear — that leaves the tree with no
			// node for the absent button at all, where the core.If this
			// replaced emitted a real (empty) child for the reconciler to
			// diff on every pass.
			components.ListRow{
				Content: core.Text(fmt.Sprintf("%d %s left", remaining, itemWord), core.UseStyle(core.Style{
					FontSize:  13,
					TextColor: colorDim,
				})),
				Trailing: clearButton,
			},
		},
	}
}

// filterBar is a components.SegmentedControl. The widget owns the row, the
// 8pt gap, the per-segment key, and the index comparison that decides which
// chip is selected; what stays here is what is actually this app's — the
// palette override and the sentence a screen reader reads.
//
// The filter constants are indices into filterLabels (see their declaration),
// so the app's own enum is the control's Selected value with no mapping in
// between.
func filterBar(active int, onSelect func(int)) core.View {
	return components.SegmentedControl{
		Labels:    filterLabels,
		Selected:  active,
		OnSelect:  onSelect,
		KeyPrefix: "filter-",
		// The template: everything every chip shares. Label, Selected and
		// OnTap are the control's to fill in.
		Segment: components.Chip{
			Style: []core.StyleProp{
				core.FontSize(13),
				core.Transition(200, core.EaseInOut),
			},
			// The theme's Button base paints a white label; on the pale
			// accent background that is illegible, so the selected chip
			// overrides both colors together (Chip's theme default is
			// overridden to keep this app's palette).
			SelectedStyle: []core.StyleProp{
				core.BackgroundColor(colorAccent),
				core.TextColor(colorAccentInk),
			},
			AccessibilityHint: "Filters the task list",
		},
		// The one thing that varies per segment and is not the caption:
		// "Active" is announced as "Show active tasks". Chip appends
		// ", selected" to whichever name it is given.
		SegmentLabel: func(label string, _ int) string {
			return "Show " + strings.ToLower(label) + " tasks"
		},
	}
}

// todoRow is a pure function of its Todo — no hooks, per the positional-slot
// constraint explained on App. Completion is conveyed by dimming the title;
// the Style struct has no strikethrough field yet.
//
// It is a components.ListRow: checkbox leading, title in the growing middle,
// delete button trailing. The widget owns what used to be hand-rolled here —
// the FlexGrow(1) that pins the ✕ to the trailing edge, and the vertical
// centring of a checkbox against a text line.
//
// The title goes in Content rather than Title because it is *conditionally
// styled*: Title takes the theme's Body role verbatim, and this one has to
// dim when the todo is done. Content is the escape hatch for exactly that —
// an arbitrary view in the growing slot.
func todoRow(t Todo, setDone func(int, bool), remove func(int)) core.View {
	titleColor := "#000000"
	if t.Done {
		titleColor = colorDim
	}

	// Capture the ID, not the loop variable's slice position: the closures
	// outlive this pass and must still address the right todo after the
	// visible slice is rebuilt under a different filter.
	id := t.ID

	return core.Keyed(fmt.Sprintf("todo-%d", t.ID), components.ListRow{
		Leading: core.Checkbox(t.Done, func(v bool) { setDone(id, v) }),
		Content: core.Text(t.Title, core.UseStyle(core.Style{
			FontSize:  16,
			TextColor: titleColor,
		})),
		// Destructive affordance: the theme's Button base is a medium blue,
		// which both hides the red glyph and reads as a primary action. The
		// variant says which role to take instead; the widget resolves the
		// fill and a legible ink for it.
		Trailing: components.Button{
			Label:              "✕",
			OnTap:              func() { remove(id) },
			Variant:            components.VariantError,
			Style:              []core.StyleProp{core.FontSize(13)},
			AccessibilityLabel: "Delete " + t.Title,
		},
		Style: []core.StyleProp{
			core.Padding(8),
			core.Gap(10),
			core.Transition(200, core.EaseInOut),
		},
		// Not ListRow.Selected: "done" is not "selected". The widget's
		// Selected flag appends ", selected", whereas this app announces
		// completion — so the whole label stays the app's to compose.
		AccessibilityLabel: rowAccessibilityLabel(t),
	})
}

func rowAccessibilityLabel(t Todo) string {
	if t.Done {
		return t.Title + ", completed"
	}
	return t.Title + ", not completed"
}
