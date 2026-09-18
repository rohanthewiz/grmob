package core

import "sync"

// Scroll commands: bring a named node into view.
//
// # What it is for
//
// A scroll position belongs to the host. Go never learns it and never sets
// it, which is right for everything a user scrolls. The exception is an app
// that wants to *take* the reader somewhere: a "jump to the demo" link, a
// form that scrolls to its first error, a table of contents. That needs one
// imperative command, and like core.Focus it can only travel the one channel
// Go has to the host, the render tree.
//
//	core.Column(core.ScrollTarget("errors"), errorList)
//	...
//	core.Button("Show the errors", func() { core.ScrollIntoView(ctx, "errors") })
//
// # The encoding
//
// The same one core.Focus uses, for the same reasons (see focus.go): a
// command is a counter, not a bool, so a second command to the same node is
// still a change a host can see. The node whose ScrollTarget name the latest
// command named carries
//
//	scrollEpoch   the command's number, counting from 1 per app
//
// and every other node carries nothing. A host scrolls the node into view when
// it meets an epoch higher than any it has applied, whether it meets it on a
// node already on screen or on one it is only now building (a command issued
// one pass before its target exists, as a command issued while navigating
// is).
//
// "Higher than any it has applied" is per app, not per node, and that is the
// difference from focus. Nothing consumes the stamp, so the node keeps its
// epoch; a host that went by "a node's first sight" would scroll to it again
// every time it was rebuilt, and a layout change rebuilds whole subtrees. The
// app-wide high-water mark makes a command act once.
//
// # What "into view" means
//
// Each platform's own bring-into-view: scrolled the least distance that shows
// the whole node, or its leading edge if it cannot all fit, in every scroll
// container between it and the screen. That is Element.scrollIntoView with
// block "nearest" on the web, BringIntoViewRequester.bringIntoView on Compose,
// and ScrollViewProxy.scrollTo with no anchor on SwiftUI. All three agree on
// it without being asked to, which is why it is the definition rather than an
// alignment one of them would have to imitate.
//
// # Names, not refs
//
// core.Focus names its node with a FocusRef from a hook. A scroll target is
// named with a string instead, because the common target is not a component's
// own node but one somewhere else on the screen, often in a list whose length
// varies, where a hook per target would break the rules of hooks. A name is
// scoped to the app, like a key: two targets with the same name both carry the
// stamp, and each host scrolls to whichever it meets first.
//
// Not reached by htmlout, whose exports are static documents with nothing to
// run a command, or by core.List's windowed rows on the natives, whose
// off-screen rows do not exist to be scrolled to.

// scrollToState is the app's latest scroll command.
type scrollToState struct {
	mu     sync.Mutex
	epoch  int
	target string
}

// ScrollTarget names the node it is applied to, so ScrollIntoView can bring
// it into view. An empty name is no target.
func ScrollTarget(name string) BehaviorProp {
	if name == "" {
		return nil
	}
	return behaviorFunc(func(ctx *Context, n *Node) {
		if ctx == nil || ctx.scrollTo == nil {
			return
		}
		ctx.scrollTo.mu.Lock()
		epoch, target := ctx.scrollTo.epoch, ctx.scrollTo.target
		ctx.scrollTo.mu.Unlock()
		// Only the latest command's target is stamped. An app that never
		// scrolls renders byte-identical trees to before.
		if epoch == 0 || target != name {
			return
		}
		if n.Props == nil {
			n.Props = map[string]any{}
		}
		n.Props["scrollEpoch"] = epoch
	})
}

// ScrollIntoView brings the node named name into view, as of the next render
// pass. Called from an event handler, or from anywhere a State.Set may be:
// it requests a pass itself.
//
// A name no node carries scrolls nothing, and the command still takes an
// epoch: it happened, and had no target on screen, as a Focus on an absent
// ref does.
func ScrollIntoView(ctx *Context, name string) {
	if ctx == nil || ctx.scrollTo == nil || name == "" {
		return
	}
	ctx.scrollTo.mu.Lock()
	ctx.scrollTo.epoch++
	ctx.scrollTo.target = name
	ctx.scrollTo.mu.Unlock()
	ctx.RequestRender()
}
