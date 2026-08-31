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
