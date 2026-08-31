package core

func If(condition bool, view View) View {
	if condition {
		return view
	}
	return Fragment()
}

func For[T any](items []T, render func(item T, index int) View) View {
	return ComponentFunc(func(ctx *Context) *Node {
		var children []View
		for i, item := range items {
			children = append(children, render(item, i))
		}
		return &Node{
			Type:     "Fragment",
			Children: renderAll(ctx, "For", children),
		}
	})
}

func IfElse(condition bool, thenView View, elseView View) View {
	if condition {
		return thenView
	}
	return elseView
}

type WhenClause struct {
	Condition bool
	View      View
}

func When(cond bool, view View) WhenClause {
	return WhenClause{Condition: cond, View: view}
}

func Otherwise(view View) WhenClause {
	return WhenClause{Condition: true, View: view}
}

func MatchBool(clauses ...WhenClause) View {
	for _, clause := range clauses {
		if clause.Condition {
			return clause.View
		}
	}
	return Fragment()
}

// MatchCase Generic Match for comparable values
type MatchCase[T comparable] struct {
	Value   T
	View    View
	Default bool
}

func Case[T comparable](val T, view View) MatchCase[T] {
	return MatchCase[T]{Value: val, View: view}
}

func Default[T comparable](view View) MatchCase[T] {
	return MatchCase[T]{Default: true, View: view}
}

func Match[T comparable](input T, cases ...MatchCase[T]) View {
	for _, c := range cases {
		if c.Default || c.Value == input {
			return c.View
		}
	}
	return Fragment()
}

// MaybeProp conditionally contributes one item to a container's argument list:
// Row, Column, Card, Box and List, the variadic ...PropsAndChildren builders.
// It returns prop when cond holds and an untyped nil otherwise, and
// containerNode skips a nil item, so a false condition costs the tree nothing
// at all — no node, no slot, no style.
//
// It exists because core.If cannot do this job, in two separate ways:
//
//  1. If(false, view) returns Fragment(), and an empty Fragment is still a
//     real child node: the reconciler walks and diffs it on every pass, and
//     it occupies a child index, so anything addressing children by position
//     counts it. If earns its place where the alternative is a whole branch
//     of the tree; it is the wrong tool for one optional item in a row of
//     three.
//
//     It does not, however, draw anything. A grouping node with no children
//     renders no box on any of the three targets — that is what both native
//     renderers have always done, and what the HTML exporter now does too.
//     An earlier version of this note claimed the empty Fragment took a flex
//     slot and opened a stray Gap; that was true only of the exporter, which
//     wrapped every grouping node in a div, and it is fixed. The cost is a
//     node, not a gap.
//
//  2. If is typed View -> View. There is no If for a StyleProp or a
//     BehaviorProp, so "apply this padding only when selected" or "attach
//     OnClick only when a handler was supplied" had no expression form at all.
//     MaybeProp takes PropsAndChildren, so it covers all three item kinds
//     with one helper.
//
// Together those replace the accumulate-into-a-slice idiom this codebase kept
// reaching for:
//
//	items := make([]core.PropsAndChildren, 0, 3)
//	items = append(items, core.UseStyle(bubble))
//	if !mine {
//	    items = append(items, core.Text(from))
//	}
//	items = append(items, core.Text(body))
//	return core.Column(items...)
//
//	// becomes
//	return core.Column(
//	    core.UseStyle(bubble),
//	    core.MaybeProp(!mine, core.Text(from)),
//	    core.Text(body),
//	)
//
// Two limits, both deliberate:
//
// prop is evaluated eagerly, like any Go argument — the condition does not
// guard it. That is safe for the prop constructors, which only build values
// (core.Text returns a closure; nothing renders until the container renders
// it), but MaybeProp is not a substitute for an if statement around an
// expression that would panic or do real work on the false path.
//
// The return type is PropsAndChildren (i.e. any), so this is only valid in the
// container builders' argument lists. Text and Button take typed variadics
// (...StyleProp), which will not accept it — and must not, since their loops
// call Apply on every element and would panic on a nil.
func MaybeProp(cond bool, prop PropsAndChildren) PropsAndChildren {
	if cond {
		return prop
	}
	return nil
}
