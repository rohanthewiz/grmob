package core

type CameraProp interface {
	Apply(*CameraNode)
}

type CameraNode struct {
	OnCapture func(string)
	OnError   func(string)
	Active    bool
	Flash     bool
	Facing    string
	Overlay   View
	Style     Style
}

func CameraView(props ...CameraProp) View {
	return ComponentFunc(func(ctx *Context) *Node {
		node := &CameraNode{
			Active: true, // default
			Facing: "rear",
		}

		for _, p := range props {
			p.Apply(node)
		}

		propMap := map[string]any{
			"active": node.Active,
			"flash":  node.Flash,
			"facing": node.Facing,
		}

		if node.OnCapture != nil {
			propMap["onCapture"] = ctx.registerTextCallback(node.OnCapture)
		}

		if node.OnError != nil {
			propMap["onError"] = ctx.registerTextCallback(node.OnError)
		}

		children := []View{}
		if node.Overlay != nil {
			children = append(children, node.Overlay)
		}

		return &Node{
			Type:     "CameraView",
			Props:    propMap,
			Style:    &node.Style,
			Children: renderAll(ctx, "CameraView", children),
		}
	})
}

type cameraFunc func(*CameraNode)

func (f cameraFunc) Apply(c *CameraNode) { f(c) }

func WithFlash(enabled bool) CameraProp {
	return cameraFunc(func(c *CameraNode) {
		c.Flash = enabled
	})
}

func SetFacing(facing string) CameraProp {
	return cameraFunc(func(c *CameraNode) {
		c.Facing = facing
	})
}

func OnError(fn func(string)) CameraProp {
	return cameraFunc(func(c *CameraNode) {
		c.OnError = fn
	})
}

func OnCapture(fn func(string)) CameraProp {
	return cameraFunc(func(c *CameraNode) {
		c.OnCapture = fn
	})
}

func WithOverlay(view View) CameraProp {
	return cameraFunc(func(c *CameraNode) {
		c.Overlay = view
	})
}

func WithStyle(style Style) CameraProp {
	return cameraFunc(func(c *CameraNode) {
		c.Style = style
	})
}
