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

// Modal draws its Content over the screen, above a scrim, while Visible is
// true. It is controlled: OnDismiss reports a scrim tap or the platform's
// dismiss gesture, and the modal closes only when Visible goes false.
//
// Each host presents it in its own idiom:
//
//	target    presentation                      content that fits
//	───────   ───────────────────────────────   ───────────────────────
//	web ×2    fixed overlay, flex column        centred in the window
//	Compose   Dialog window                     centred window
//	SwiftUI   sheet, medium and large detents   top of the sheet
//
// Content taller than the space it is given scrolls on every host, from its
// top: the web overlay is overflow-y:auto with justify-content: safe center,
// the Compose dialog and the SwiftUI sheet wrap their content in a vertical
// scroll. Content that fits is laid out exactly as it would be without one. A
// FlexGrow child still fills the Compose dialog's height when nothing
// overflows (comps.ActionSheet's filler relies on it).
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

// ModalContent sets the views drawn inside the overlay. It appends rather than
// replaces, so content may be assembled across several props if a caller finds
// that clearer; order is render order, top to bottom.
//
// The content renders every pass regardless of Visible — a Modal hides, it
// does not unmount. Visible is an ordinary prop the host maps to visibility
// (display on the web), so toggling it is a cheap prop patch, not a subtree
// add/remove, and any state hooks inside the content survive a close. That
// makes the trade-off against navigation explicit: a dismissed modal reopens
// exactly as it was left, where a popped Navigator frame starts fresh. A
// dialog whose state must NOT survive dismissal should reset it in OnDismiss,
// where the intent is recorded.
func ModalContent(children ...View) ModalProp {
	return modalFunc(func(m *ModalNode) {
		m.Content = append(m.Content, children...)
	})
}
