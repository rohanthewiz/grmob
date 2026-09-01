package core

import "fmt"

type PropsAndChildren any

// containerNode builds a container node of the given type from a mixed
// argument list of style props, behavior props, and child views. All the
// flex-style containers (Row, Column, Card, Box, List) share it so behavior
// props work uniformly: BehaviorProps apply to the node that is actually
// returned — an earlier version of Column applied them to a throwaway props
// map, so OnClick on a container silently never reached the renderer.
//
// Ordering contract: behavior props register their callbacks in argument
// order, before any child renders (children are collected during the loop
// but rendered after it) — so a container's own callback IDs always precede
// its children's within a render pass, regardless of argument interleaving.
//
// Nil contract: a nil item is skipped. That is what makes MaybeProp work —
// its false path returns an untyped nil rather than a placeholder node, so a
// conditional prop that does not apply leaves no trace in the tree at all.
func containerNode(ctx *Context, typ string, base Style, items []PropsAndChildren) *Node {
	style := &base
	n := &Node{Type: typ, Style: style}
	var children []View
	for _, item := range items {
		switch v := item.(type) {
		case StyleProp:
			v.Apply(style)
		case View:
			children = append(children, v)
		case BehaviorProp:
			v.Apply(ctx, n)
		case nil:
			// A nil item is a contract, not an accident: MaybeProp(false, ...)
			// returns an untyped nil so a dropped prop costs the tree nothing.
			// Cased explicitly ahead of the default below so that contract is
			// visible here, at the one place it is honored, and so the
			// unknown-item concern cannot fire on it.
		default:
			// Anything else silently vanished. That is a real footgun — a bare
			// core.Style instead of core.UseStyle(style), a WhenClause that
			// never reached MatchBool, a *Node instead of a View — and it
			// produces no compile error and no visible failure, just a missing
			// property. Debug mode names the type that was dropped.
			if IsDebugMode() {
				upsertConcern(ConcernUnknownItem,
					fmt.Sprintf("%s: argument of type %T is not a StyleProp, BehaviorProp or View and was ignored", typ, v))
			}
		}
	}
	n.Children = renderAll(ctx, typ, children)
	return n
}

// leafNode builds a childless node — a text field, a checkbox — from the same
// mixed argument list the containers take, minus the children.
//
// It exists because focus is a leaf event. core.OnFocus/OnBlur are ordinary
// BehaviorProps, and BehaviorProps only ever reached nodes built by
// containerNode; the input builders took `...StyleProp`, so there was no way
// to spell "this field, on blur". Widening them to PropsAndChildren is
// source-compatible for the same reason Scroll's widening was — a StyleProp
// is a PropsAndChildren — and it means every future event prop lands on the
// inputs for free instead of spawning another WithX builder.
//
// The one call shape the widening does break is forwarding: a []StyleProp
// cannot be spread into a ...PropsAndChildren, so a wrapper that collected
// style props into a slice has to widen its own slice too. forms' bound
// builders are the only ones in the tree that did, and they now take
// []core.PropsAndChildren.
//
// Ordering contract: props is the builder's own prop map (value, placeholder,
// its onChange ID), already populated, so a builder's intrinsic callbacks
// always hold the lower IDs of the pass and no argument a caller writes can
// move them. The items then register in argument order, as they do in a
// container.
//
// Nil contract: as containerNode's — a nil item is MaybeProp's false path and
// leaves no trace.
//
// A View passed here is the one shape containerNode accepts and this cannot:
// a leaf has nowhere to put a child. It is reported rather than silently
// dropped, which is the whole point of ConcernUnknownItem.
//
// Focus contract: a leaf whose type the platform gives keyboard focus to
// (focusableLeafTypes) carries the current focus command, so a
// core.DismissKeyboard can reach the field the *user* tapped into — a field
// no app code ever named with a FocusTarget. The stamp goes on before the
// items run, so a FocusTarget in the argument list overwrites the default
// action with "focus"; it writes the same epoch either way, so the overwrite
// is a no-op on every node that is not the command's target. Until something
// issues a focus command the stamp writes nothing at all, which is what keeps
// an app that never touches focus rendering the exact trees it did before.
func leafNode(ctx *Context, typ string, base Style, props map[string]any, items []PropsAndChildren) *Node {
	style := &base
	if props == nil {
		props = map[string]any{}
	}
	if focusableLeafTypes[typ] {
		stampFocus(ctx, props, nil)
	}
	// The style pointer goes into the node before the props are applied, so a
	// StyleProp reached later in the loop still lands on the node that is
	// returned — the same aliasing containerNode relies on.
	n := &Node{Type: typ, Props: props, Style: style}

	for _, item := range items {
		switch v := item.(type) {
		case StyleProp:
			v.Apply(style)
		case BehaviorProp:
			v.Apply(ctx, n)
		case nil:
			// MaybeProp's false path; see containerNode's nil contract.
		default:
			if IsDebugMode() {
				upsertConcern(ConcernUnknownItem,
					fmt.Sprintf("%s: argument of type %T is not a StyleProp or BehaviorProp and was ignored (a leaf node takes no children)", typ, v))
			}
		}
	}
	return n
}

func Row(stylePropsAndChildren ...PropsAndChildren) View {
	return ComponentFunc(func(ctx *Context) *Node {
		return containerNode(ctx, "Row", ctx.Theme().Components.Row, stylePropsAndChildren)
	})
}

func Card(stylePropsAndChildren ...PropsAndChildren) View {
	return ComponentFunc(func(ctx *Context) *Node {
		return containerNode(ctx, "Card", ctx.Theme().Components.Card, stylePropsAndChildren)
	})
}

func Spacer(size int) View {
	return ComponentFunc(func(ctx *Context) *Node {
		return &Node{
			Type: "Spacer",
			Props: map[string]any{
				"size": size,
			},
		}
	})
}

// Scroll is a vertically scrolling region: its children are laid out at their
// natural height and the viewport pans over them.
//
// It takes the standard container argument list — style props, behavior props
// and child views in any order — rather than the bare ...View it used to,
// because both native renderers have always applied a Scroll node's style
// (Compose boxModifier, SwiftUI grMobBox) and Go had no way to set one. The
// widening is source-compatible: a View is a PropsAndChildren, so every
// existing core.Scroll(child) call still compiles and, with no props supplied,
// still renders the same box.
//
// Unlike Column and Row it has no theme base — like Box, it is the
// unopinionated container, and a scroll region that arrived with the theme
// Column's screen padding would inset every screen that wraps itself in one.
//
// See KeyboardAware for the software-keyboard behavior.
func Scroll(stylePropsAndChildren ...PropsAndChildren) View {
	return ComponentFunc(func(ctx *Context) *Node {
		return containerNode(ctx, "Scroll", Style{}, stylePropsAndChildren)
	})
}

func SafeArea(child View) View {
	return ComponentFunc(func(ctx *Context) *Node {
		return &Node{
			Type:     "SafeArea",
			Props:    map[string]any{},
			Children: []*Node{child.Render(ctx)},
		}
	})
}

func Fragment(children ...View) View {
	return ComponentFunc(func(ctx *Context) *Node {
		if len(children) == 1 {
			return children[0].Render(ctx)
		}
		return &Node{
			Type:     "Fragment",
			Children: renderAll(ctx, "Fragment", children),
		}
	})
}

func Column(stylePropsAndChildren ...PropsAndChildren) View {
	return ComponentFunc(func(ctx *Context) *Node {
		return containerNode(ctx, "Column", ctx.Theme().Components.Column, stylePropsAndChildren)
	})
}

func Box(stylePropsAndChildren ...PropsAndChildren) View {
	return ComponentFunc(func(ctx *Context) *Node {
		// Box has no theme base by design: it is the unopinionated container.
		return containerNode(ctx, "Box", Style{}, stylePropsAndChildren)
	})
}
func Divider(height int, color string) View {
	return Box(
		Height(fmt.Sprintf("%dpx", height)),
		BackgroundColor(color),
		Margin(8),
	)
}
func BorderColor(hex string) StyleProp {
	return styleFunc(func(s *Style) {
		s.BorderColor = hex
	})
}
func BorderWidth(px float64) StyleProp {
	return styleFunc(func(s *Style) {
		s.BorderWidth = px
	})
}

// renderAll renders a sibling list. It is the one choke point every
// container's children pass through (containerNode, Fragment, For, Modal,
// TabView, camera overlays), which makes it the natural home for the debug
// duplicate-key check: parentType names the container in the concern so a
// finding points at the right sibling list.
func renderAll(ctx *Context, parentType string, views []View) []*Node {
	var out []*Node
	for _, v := range views {
		out = append(out, v.Render(ctx))
	}
	if IsDebugMode() {
		checkDuplicateKeys(parentType, out)
	}
	return out
}
