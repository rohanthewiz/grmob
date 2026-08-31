package components

import "github.com/rohanthewiz/grmob/core"

// Screen is the root scaffold every app in this repo was hand-spelling: the
// safe-area inset, an optional scroll region, and the vertical column that
// holds the screen's content.
//
//	SafeArea
//	  └─ Scroll            (only when Scroll is true)
//	       └─ Column       ← Gap / Fill / Style land here
//	            ├─ Children[0]
//	            └─ …
//
// Five call sites spelled some subset of that by hand, and each picked its own
// subset — one wrapped in Scroll, two set a Gap, one set FlexGrow(1), and no
// two agreed on the order the props were written in. The shape is not hard to
// type; the value of naming it is that there is now one place to hang the
// things a screen root will eventually need (a keyboard-aware scroll area, a
// pull-to-refresh region, per-platform inset behavior) instead of five.
//
// # The zero value is exactly SafeArea(Column(children...))
//
// Every field defaults to contributing nothing, so the zero value renders the
// bare scaffold and no style props at all — the theme's Column base carries
// through untouched. That is deliberate and it is what let all five migrations
// below stay byte-identical: a field only speaks when the caller sets it. It
// applies specifically to Gap, which is applied only when non-zero. An
// explicit core.Gap(0) would *overwrite* a gap the theme's Column base had set
// (style props mutate the style directly rather than merging into it), so
// "unset" and "zero" have to mean the same thing here — the absence of a gap,
// not the imposition of one.
//
// # Nil children are skipped
//
// Children flows into core.Column's argument list, which skips nil items (the
// same contract that makes core.MaybeProp work). So the conditional-region
// idiom this codebase already uses for slots —
//
//	var banner core.View
//	if offline {
//	    banner = OfflineBanner()
//	}
//	return components.Screen{Children: []core.View{banner, body}}
//
// — costs the tree no node at all when the condition is false, rather than the
// empty Fragment a core.If would leave behind for the reconciler to walk on
// every pass. (That Fragment draws nothing; the cost is the node, not a gap.)
type Screen struct {
	// Children are the screen's content, laid out top to bottom in the
	// column. A nil entry is skipped (see above).
	Children []core.View

	// Scroll wraps the column in core.Scroll, making the whole screen
	// scrollable. Leave it false when the screen has its own scrolling region
	// inside it — a core.List, or a Scroll around one section — since a
	// scroll view nested in a scroll view fights for the same drag on both
	// natives.
	Scroll bool

	// Gap is the uniform vertical spacing between children, in points. Zero
	// means "don't set one", not "zero spacing" — the theme's Column base
	// keeps whatever it had. Use Gap for uniform runs and core.Spacer between
	// specific children when the spacing differs (that rule predates this
	// widget; see examples/fintechapp).
	Gap float64

	// Fill makes the column claim the full height of the safe area
	// (FlexGrow(1)) rather than shrinking to its content. Set it when a child
	// needs to expand into the leftover space — a list that should fill the
	// screen and push a footer to the bottom — because a FlexGrow child can
	// only grow inside a parent that has height to give.
	//
	// Fill with Scroll is legal but unusual: it makes the scrolled content at
	// least as tall as the viewport, which is how you bottom-anchor a footer
	// on a short page. It does not make a scroll view fill anything.
	Fill bool

	// Style is applied to the column, after Gap and Fill, so a caller can
	// override either — or add padding and a background the scaffold itself
	// has no opinion about.
	Style []core.StyleProp
}

func (s Screen) Render(ctx *core.Context) *core.Node {
	// Build core.Column's mixed prop/child argument list. Capacity is exact:
	// the two optional style props, the caller's overrides, and the children.
	items := make([]core.PropsAndChildren, 0, len(s.Style)+len(s.Children)+2)

	if s.Fill {
		items = append(items, core.FlexGrow(1))
	}
	if s.Gap != 0 {
		items = append(items, core.Gap(s.Gap))
	}
	// Caller Style last: containerNode applies style props in argument order,
	// so anything here wins over the two props above.
	for _, sp := range s.Style {
		items = append(items, sp)
	}
	for _, child := range s.Children {
		items = append(items, child)
	}

	inner := core.Column(items...)
	if s.Scroll {
		inner = core.Scroll(inner)
	}
	// SafeArea takes a single child and renders it immediately, so this
	// returns the fully rendered node rather than another View.
	return core.SafeArea(inner).Render(ctx)
}
