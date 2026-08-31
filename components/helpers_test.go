package components

import (
	"github.com/rohanthewiz/grmob/core"
)

// findFirst walks the rendered node tree depth-first and returns the first
// node satisfying pred. Tests locate structure with predicates rather than
// hard-coded child indices so they don't break when a widget gains a
// decorative wrapper.
func findFirst(n *core.Node, pred func(*core.Node) bool) *core.Node {
	if n == nil {
		return nil
	}
	if pred(n) {
		return n
	}
	for _, c := range n.Children {
		if found := findFirst(c, pred); found != nil {
			return found
		}
	}
	return nil
}

// findText finds a Text node whose content equals s.
func findText(n *core.Node, s string) *core.Node {
	return findFirst(n, func(n *core.Node) bool {
		return n.Type == "Text" && n.Props["content"] == s
	})
}
