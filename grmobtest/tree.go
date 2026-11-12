package grmobtest

import (
	"strconv"
	"strings"

	"github.com/rohanthewiz/grmob/core"
)

// Find returns the first node under root, root included, that pred accepts,
// in depth-first order (a parent before its children, children in order), or
// nil.
//
// Depth-first and parent-first means a predicate an ancestor also satisfies
// ("shows the word Inbox") finds the outermost match. A test that wants the
// innermost writes the predicate to say so, or uses FindAll and takes the
// last.
func Find(root *core.Node, pred func(*core.Node) bool) *core.Node {
	if root == nil {
		return nil
	}
	if pred(root) {
		return root
	}
	for _, c := range root.Children {
		if f := Find(c, pred); f != nil {
			return f
		}
	}
	return nil
}

// FindAll returns every node under root, root included, that pred accepts,
// in the order Find visits them.
func FindAll(root *core.Node, pred func(*core.Node) bool) []*core.Node {
	var out []*core.Node
	var walk func(*core.Node)
	walk = func(n *core.Node) {
		if n == nil {
			return
		}
		if pred(n) {
			out = append(out, n)
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)
	return out
}

// HasText reports whether a Text node under root says exactly s.
//
// Exact, and Text only, because it is the check that cannot pass by accident:
// HasText(tree, "3") is not satisfied by "13 items" or by a button labelled
// "3". Shows is the looser question.
func HasText(root *core.Node, s string) bool {
	return Find(root, func(n *core.Node) bool {
		return n.Type == "Text" && n.Props["content"] == s
	}) != nil
}

// Shows reports whether s appears, as a substring, in any word root puts on
// screen: a Text's content, a Button's label, a row's title, a field's
// placeholder or value, one line of a code block.
//
// Every string prop is searched, because "on screen" is not one node type:
// a test asking whether a screen shows "Paid" should not need to know whether
// the word is a Text or a chip's label. Two things are left out:
//
//   - callback IDs ("cb_3", "txt_cb_r2/e0/1"), so Shows("3") cannot be
//     satisfied by the third button's handler;
//   - Style, which is a separate field on the wire, so a colour or a length
//     is never mistaken for a word.
//
// A code block (core.TextGrid) draws a line as several coloured runs, so a
// GridRow is matched on its runs joined; a match spanning two lines is not a
// match, as it would not be for a Text.
func Shows(root *core.Node, s string) bool {
	return Find(root, func(n *core.Node) bool { return nodeShows(n, s) }) != nil
}

func nodeShows(n *core.Node, s string) bool {
	if n.Type == "GridRow" && strings.Contains(gridRowText(n), s) {
		return true
	}
	for _, v := range n.Props {
		if propShows(v, s) {
			return true
		}
	}
	return false
}

// propShows searches one decoded prop value, recursing into lists and maps
// (a menu's items, a chart's labels), and skips callback IDs wherever they
// sit.
func propShows(v any, s string) bool {
	switch v := v.(type) {
	case string:
		return !isCallbackID(v) && strings.Contains(v, s)
	case []any:
		for _, e := range v {
			if propShows(e, s) {
				return true
			}
		}
	case map[string]any:
		for _, e := range v {
			if propShows(e, s) {
				return true
			}
		}
	}
	return false
}

// callbackPrefixes are the kinds core numbers callback IDs under (core's
// event.go: void, text, bool, int, and the back claim). An ID is the prefix
// and a number or a key path, so no visible word a test would look for starts
// this way.
var callbackPrefixes = []string{"cb_", "txt_cb_", "bool_cb_", "int_cb_", "back_cb_"}

func isCallbackID(v string) bool {
	for _, p := range callbackPrefixes {
		if strings.HasPrefix(v, p) {
			return true
		}
	}
	return false
}

// gridRowText joins one TextGrid row's runs. A run crosses the wire as
// core.GridRun's JSON, with its glyphs under "t".
func gridRowText(n *core.Node) string {
	runs, ok := n.Props["runs"].([]any)
	if !ok {
		return ""
	}
	var b strings.Builder
	for _, raw := range runs {
		if run, ok := raw.(map[string]any); ok {
			if t, ok := run["t"].(string); ok {
				b.WriteString(t)
			}
		}
	}
	return b.String()
}

// Button accepts the Button labelled label.
func Button(label string) func(*core.Node) bool {
	return func(n *core.Node) bool {
		return n.Type == "Button" && n.Props["label"] == label
	}
}

// OfType accepts every node of the type ("Input", "Checkbox", "List").
func OfType(nodeType string) func(*core.Node) bool {
	return func(n *core.Node) bool { return n.Type == nodeType }
}

// Named accepts every node whose accessible name (core.AccessibilityLabel) is
// name: the words a screen reader says for it, and the most stable address a
// control has.
func Named(name string) func(*core.Node) bool {
	return func(n *core.Node) bool { return accessibleName(n) == name }
}

func accessibleName(n *core.Node) string {
	if n.Style == nil {
		return ""
	}
	return n.Style.AccessibilityLabel
}

func quote(s string) string { return strconv.Quote(s) }
