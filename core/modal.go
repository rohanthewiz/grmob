package core

type ModalProp interface {
	Apply(*ModalNode)
}

type ModalNode struct {
	Visible   bool
	OnDismiss func()
	Backdrop  string
	Content   []View
}

func Modal(props ...ModalProp) View {
	return ComponentFunc(func(ctx *Context) *Node {
		node := &ModalNode{
			Visible:  false,
			Backdrop: "#00000088", // default
		}

		for _, p := range props {
			p.Apply(node)
		}

		children := renderAll(ctx, "Modal", node.Content)

		propMap := map[string]any{
			"visible":  node.Visible,
			"backdrop": node.Backdrop,
		}

		if node.OnDismiss != nil {
			propMap["onDismiss"] = ctx.registerCallback(node.OnDismiss)
		}

		return &Node{
			Type:     "Modal",
			Props:    propMap,
			Children: children,
		}
	})
}

type modalFunc func(*ModalNode)

func (f modalFunc) Apply(m *ModalNode) { f(m) }

func Visible(v bool) ModalProp {
	return modalFunc(func(m *ModalNode) {
		m.Visible = v
	})
}

func OnDismiss(fn func()) ModalProp {
	return modalFunc(func(m *ModalNode) {
		m.OnDismiss = fn
	})
}

func Backdrop(color string) ModalProp {
	return modalFunc(func(m *ModalNode) {
		m.Backdrop = color
	})
}
