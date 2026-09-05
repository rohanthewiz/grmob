package core

import "sync"

// List is the virtualized sibling of Column: a vertically scrolling container
// whose children are laid out lazily by the native renderer (Compose
// LazyColumn, SwiftUI LazyVStack), so a thousand-row feed composes only the
// rows on screen. Column + Scroll remains the right choice for short content;
// List is for long, data-driven collections.
//
// Give every child a stable identity with Keyed(id, ...) — the native lazy
// containers use the key to keep row state (and recycled views) attached to
// the same data across insertions, removals, and reorders. Unkeyed children
// fall back to positional identity, which behaves like Column but loses row
// state on reorder.
//
// It shares Column's theme base and the standard container argument contract:
// style props, behavior props (e.g. OnClick on the list surface), and child
// views in any order.
func List(stylePropsAndChildren ...PropsAndChildren) View {
	return ComponentFunc(func(ctx *Context) *Node {
		return containerNode(ctx, "List", ctx.Theme().Components.Column, stylePropsAndChildren)
	})
}

// StickyHeader pins a List child to the top of the viewport while the rows
// it introduces scroll past underneath it, releasing it when the next sticky
// child arrives to take its place. It is the group band of an archive feed —
// the month over a run of sermons, the day over a run of transactions.
//
//	core.List(
//	    core.Keyed("group:2026-01", core.Row(core.StickyHeader(), monthLabel)),
//	    core.Keyed("s1", row), core.Keyed("s2", row),
//	)
//
// components.GroupedList{StickyHeaders: true} is the widget spelling; this
// is the primitive underneath it.
//
// # Why a StyleProp, and why no new field
//
// Sticky positioning is not a new idea to this tree: Style.Position already
// carries PositionSticky, and both DOM targets already emit `position`,
// `top` and `z-index` verbatim, so the web half of this feature has always
// worked and needed no code. What was missing was the two natives, which
// declined Position outright ("no Compose analog at this layer") — true of
// `fixed` and `absolute`, and not true of `sticky`, which is precisely what
// a Compose stickyHeader and a SwiftUI pinned Section header are.
//
// So the marker is the CSS one, and the renderers converge on it rather than
// on a private flag:
//
//	web       position:sticky; top:0; z-index:1
//	Compose   LazyColumn { stickyHeader { … } } for the marked rows
//	SwiftUI   LazyVStack(pinnedViews: .sectionHeaders) + Section(header:)
//
// Top and ZIndex are *supplied* rather than assigned — a caller's own values
// survive in either argument order — because both are load-bearing on the
// web and neither has a sensible zero: a sticky box with no offset never
// sticks, and one at the default stacking level is painted over by the rows
// that scroll under it.
//
// # Where it does something
//
// A List child. Both natives implement pinning inside their lazy container
// and have nowhere to put it otherwise, so a Column child carrying this is
// sticky in a browser and inert on a phone. That asymmetry is why the name
// says List's word for the thing ("header") rather than CSS's word for the
// mechanism.
func StickyHeader() StyleProp {
	return styleFunc(func(s *Style) {
		s.Position = PositionSticky
		if s.Top == "" {
			s.Top = "0"
		}
		if s.ZIndex == 0 {
			s.ZIndex = 1
		}
	})
}

// OnEndReached fires when the user scrolls within a few rows of the bottom of
// a List: the "fetch the next page" edge that turns a manual
// components.LoadMore button into an infinite feed.
//
//	core.List(
//	    core.OnEndReached(pager.LoadNext),
//	    rows...,
//	)
//
// # The debounce, and why it is here rather than in four renderers
//
// The edge is a *scroll position*, so every renderer reports it more than
// once for the same bottom: Compose's snapshot flow emits on each new
// last-visible index, SwiftUI's .onAppear re-fires when a row is recycled
// back into view, and an IntersectionObserver fires on entry and on every
// resize that keeps the sentinel visible. A slow fetch therefore sees two or
// three calls before its first page lands, and an offset pager answers that
// by loading page 2 twice.
//
// The fix is one line of state and it belongs on this side of the bridge:
// remember how many rows the list held when the handler last ran, and refuse
// to run again until that number changes. A fetch that appends rows unlocks
// the next fire; a fetch that returns nothing (the feed is exhausted, or it
// failed) leaves the guard closed, which is exactly right — scrolling at the
// bottom of a list that just came back empty should not re-ask forever. A
// caller that wants the retry offers a button; that is what
// components.LoadMore's error arm has always been for.
//
// Doing it in Go also means the four renderers each get to be as naive as
// their platform makes convenient, and none of them has to agree with the
// others about what "once" means.
//
// The row count is read at *dispatch* time off the node this prop was
// applied to, which by then holds the children the pass rendered. (Behaviour
// props run before children in containerNode, so there is nothing to count
// yet when this closure is built — only when it is called.)
//
// # The guard's one sharp edge
//
// State is keyed by callback ID, and callback IDs are positional: the Nth
// void handler registered in a pass is always "cb_N" (see callbackRegistry).
// Two different Lists in two different screens can therefore inherit the
// same ID across a navigation, and the second one's first end-reached is
// swallowed if it happens to hold exactly as many rows as the first did when
// the first last fired. That is the same stale-ID window the registry itself
// documents, and it closes when identity-keyed IDs land; nothing here can
// close it earlier, because the two lists are indistinguishable from this
// side.
func OnEndReached(handler func()) BehaviorProp {
	return behaviorFunc(func(ctx *Context, n *Node) {
		if n.Props == nil {
			n.Props = map[string]any{}
		}
		// id is captured by reference and assigned before the closure can
		// possibly run: registration happens during render, dispatch only
		// after the tree has been handed to a renderer.
		var id string
		id = ctx.registerCallback(func() {
			if !ctx.endReached.shouldFire(id, len(n.Children)) {
				return
			}
			handler()
		})
		n.Props["onEndReached"] = id
	})
}

// endReachedState is OnEndReached's per-app-tree debounce ledger: for each
// live end-reached callback ID, the row count the list held the last time
// its handler was allowed to run.
//
// It is a pointer field on Context, shared by every derived context, for the
// same reason the callback registry and the navigation stack are: two apps
// in one process (or two Managers in one test binary) must not see each
// other's guards.
//
// Bounded without any special effort: IDs are assigned by per-pass sequence,
// so the key space is the largest number of void callbacks any single pass
// has ever registered — not the number of passes. purge() trims it further
// on every diff, dropping the IDs a pass stopped registering.
type endReachedState struct {
	mu      sync.Mutex
	firedAt map[string]int
}

func newEndReachedState() *endReachedState {
	return &endReachedState{firedAt: map[string]int{}}
}

// shouldFire reports whether the handler behind id may run for a list
// currently holding rows children, and records the decision when it may.
//
// The recorded value is the count *at the moment of firing*, not the count
// when the fetch returns: an in-flight fetch has not changed the list yet,
// so a second scroll event arriving mid-flight sees the same number and is
// refused, which is the whole point.
func (s *endReachedState) shouldFire(id string, rows int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if last, ok := s.firedAt[id]; ok && last == rows {
		return false
	}
	s.firedAt[id] = rows
	return true
}

// purge drops the guards for callback IDs that are no longer registered,
// which is how a List that left the tree stops occupying a row in the
// ledger. Called from the callback registry's own purge, so the two stay in
// step by construction.
func (s *endReachedState) purge(live func(string) bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id := range s.firedAt {
		if !live(id) {
			delete(s.firedAt, id)
		}
	}
}
