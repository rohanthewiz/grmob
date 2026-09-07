package core

import (
	"fmt"
	"sort"
	"strings"
)

// The accessibility audit: the debug-mode pass over a finished tree, for the
// failure modes that are invisible on every target rather than merely quiet on
// two.
//
// # Why a tree walk and not a guard in the exporters
//
// Three of the four findings below are about *relationships between elements*,
// and no renderer can see one. htmlout writes an id as it walks past the node
// carrying it and has no index of the document it is building; the WASM runtime
// applies a patch to one element and has no index at all. Both say so in their
// own comments — "a dangling IDREF is inert" — and both are right that they
// cannot do better from where they sit.
//
// A finished tree can be walked, though, and core.SetDebugMode already walks
// one: the cursor audit and the duplicate-key check are the same shape, run at
// the same point in the pass, reported through the same collector. This is the
// third such check and the first about semantics rather than about the
// framework's own bookkeeping.
//
// # Why these four and not a general ARIA validator
//
// Each of them is a failure with no symptom. A duplicate id resolves to
// whichever element the browser saw first, so a tab strip switches the wrong
// region and nothing errors. A dangling aria-controls announces a tab that
// governs nothing, which sounds exactly like a tab. An id with a space in it is
// invalid HTML that browsers accept and getElementById never finds. A
// disclosure with no handler offers TalkBack an action nothing performs.
//
// Everything else in ARIA that this vocabulary can express is already caught
// where it is written: a state on a role that cannot carry it is dropped by
// both exporters *by design* and documented at each guard, and a structural
// role over foreign children is a judgement about content that no walk can
// make. The line is "would a reader be told something false, with nothing
// anywhere saying so".
//
// # The fifth finding, which is not about accessibility
//
// AuditTree's walk also carries ConcernInertPlacement — a core.StackAlign on a
// node no overlay will place. It is a layout fact rather than a semantic one,
// and it is in this walk because it is the same *shape*: a relationship between
// a node and its container, silent on all four targets, invisible to every
// exporter for structural reasons. The argument is in placement_audit.go, which
// is where it lives; only the call in walk() is here.
//
// So the walk's subject is a little wider than this file's name: it is the
// failures a finished tree can be asked about and a renderer cannot. The four
// below were the first family, not the only one.
//
// # Cost
//
// One walk of the finished tree per pass, in debug mode only — the same
// condition the other two checks run under, and the same order of work the
// reconciler is about to do anyway. Off, it is a single atomic load at the call
// site in render.Manager. The placement check adds one map lookup per node.

// Concern kinds for the accessibility audit. Declared here rather than beside
// the others in debug.go so the four arrive with the walk that produces them.
// The walk's fifth finding, ConcernInertPlacement, is declared in
// placement_audit.go for the same reason: beside the argument for it.
const (
	// ConcernDuplicateAccessibilityID: two elements in one tree carry the
	// same core.Style.AccessibilityID. Ids are document-global and nothing
	// rewrites the string, so both are written verbatim — which is invalid
	// HTML, and which makes every aria-controls pointing at that id resolve
	// to whichever element the browser parsed first.
	ConcernDuplicateAccessibilityID = "duplicate-accessibility-id"

	// ConcernDanglingReference: a core.Style.AccessibilityControls names an
	// id no element in the tree claims. The attribute is written anyway (an
	// exporter has no index to check against) and a reader following it finds
	// nothing, so the control announces as governing a region that is not
	// there.
	ConcernDanglingReference = "dangling-aria-reference"

	// ConcernInvalidAccessibilityID: an AccessibilityID that is not a usable
	// HTML id — one containing whitespace, or one starting with the "grmob-"
	// prefix core.TabView's own minted ids live in.
	ConcernInvalidAccessibilityID = "invalid-accessibility-id"

	// ConcernInertDisclosure: a node states core.Style.AccessibilityExpanded
	// and carries no handler to act on it. The web writes aria-expanded
	// regardless, and Compose does not: its expand()/collapse() are semantics
	// *actions*, and Renderer.kt wires them to the node's own click callback,
	// so a node with a state and nothing to perform gets neither. That is the
	// right behaviour — an action nothing can perform is worse than none —
	// and it is a rule stated only in a comment, where the equivalent web rule
	// (a state on an unroled node is dropped) has a test on both targets.
	ConcernInertDisclosure = "inert-disclosure"
)

// AuditTree runs the whole-tree checks over a finished render tree: the four
// accessibility findings above and the placement finding beside them.
//
// Called by the render driver after the pass that produced the tree, beside
// EndRenderPass and for the same reason: both describe a *completed* pass, and
// running either over a half-built tree reports nothing except that something
// went wrong earlier. A no-op (one atomic load) when debug mode is off.
//
// Hosts that drive passes through render.Manager get this for free. A
// hand-rolled pass loop should call it with the tree it just rendered.
//
// nil is not an error: a pass that produced no tree has nothing to audit, which
// is the state a host is in before its first render.
func AuditTree(root *Node) {
	if !IsDebugMode() || root == nil {
		return
	}
	a := &a11yAudit{ids: make(map[string][]string)}
	// "" is the placing container above the root: nothing. See placerFor.
	a.walk(root, "root", "")
	a.report()
}

// a11yAudit is the state one walk accumulates.
//
// The ids map is id → the paths that claim it, rather than id → count, because
// the count alone answers "something is wrong" and the paths answer "where" —
// and a duplicate id is precisely the bug where knowing one of the two
// locations is no help.
//
// controls is a slice rather than a set because a reference that dangles twice
// is two call sites to fix, and both want naming. It is resolved after the walk
// rather than during it: a tab may point at a panel that appears later in
// document order, which is the ordinary shape rather than an exotic one — a
// bottom tab bar sits under the region it switches.
type a11yAudit struct {
	ids      map[string][]string
	controls []reference
}

type reference struct {
	target string
	path   string
}

// placer is the node type of the container that would place this node — the
// nearest ancestor that is not a grouping container. It is threaded rather
// than looked up because a walk already knows it and a node does not carry a
// parent pointer; see placerFor and placement_audit.go.
func (a *a11yAudit) walk(n *Node, path, placer string) {
	if n == nil {
		return
	}
	if s := n.Style; s != nil {
		if s.AccessibilityID != "" {
			a.ids[s.AccessibilityID] = append(a.ids[s.AccessibilityID], path)
			a.checkID(s.AccessibilityID, path)
		}
		if s.AccessibilityControls != "" {
			a.controls = append(a.controls, reference{s.AccessibilityControls, path})
		}
		a.checkDisclosure(n, path)
		reportInertPlacement(n, path, placer)
	}
	childPlacer := placerFor(n.Type, placer)
	for i, child := range n.Children {
		a.walk(child, fmt.Sprintf("%s/%d", path, i), childPlacer)
	}
}

// checkID applies the two rules an AccessibilityID has to keep, both of which
// are stated in that field's doc and neither of which anything enforced.
//
// Whitespace is the HTML one: an id is a single token, so "app panel" is not an
// id at all — the attribute is written, the document is invalid, and
// getElementById and every `#id` selector fail to find it while the element
// sits there looking correct.
//
// The prefix is the framework's. "grmob-" is where core.TabView's minted ids
// live (tabScope in htmlout/tabview.go and its twin in grmob-runtime.js), and
// an author id that lands in that space breaks a wiring they never wrote rather
// than one of their own — which is the harder bug of the two to find, because
// nothing in their code mentions the id that broke.
//
// The empty id needs no check: "" is the field's zero value and means "no id",
// which is what every node in every tree carries.
func (a *a11yAudit) checkID(id, path string) {
	if strings.ContainsFunc(id, isHTMLSpace) {
		upsertConcern(ConcernInvalidAccessibilityID, fmt.Sprintf(
			"%s has AccessibilityID %q, which contains whitespace: an HTML id is a "+
				"single token, so this is written verbatim into an invalid document and "+
				"no aria-controls, getElementById or #id selector will ever resolve it",
			path, id))
	}
	if strings.HasPrefix(id, reservedIDPrefix) {
		upsertConcern(ConcernInvalidAccessibilityID, fmt.Sprintf(
			"%s has AccessibilityID %q, which starts with the reserved %q prefix: that "+
				"is where core.TabView's own minted tab and panel ids live, so a "+
				"collision breaks a wiring the app never wrote",
			path, id, reservedIDPrefix))
	}
}

// reservedIDPrefix is the id namespace core.TabView mints into. Stated here as
// well as in the two exporters because this is the only place that can check
// it: an exporter writing one element's id has no way to know it is about to
// mint the same string for a panel three nodes later.
const reservedIDPrefix = "grmob-"

// isHTMLSpace is the "ASCII whitespace" set the HTML specification defines for
// tokenizing attribute values. Spelled out rather than reached for through
// unicode.IsSpace, which is a wider set — an id containing U+00A0 is legal HTML
// and would be reported as broken.
func isHTMLSpace(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\f', '\r':
		return true
	}
	return false
}

// checkDisclosure flags a stated ExpandedState that no platform can act on.
//
// The guard is "neither handler", not "no onClick", because that is what
// Renderer.kt's gestureModifier actually tests: it returns early when a node
// carries neither, so a node with a long-press alone still reaches the
// disclosure wiring. Checking only for a click would report a node that works.
//
// core.Button is deliberately not exempt. A Button with no handler is as inert
// as a Box with none — the node type makes it a control, and a control with
// nothing behind it is the same dead end.
func (a *a11yAudit) checkDisclosure(n *Node, path string) {
	if n.Style.AccessibilityExpanded == ExpandedUnset {
		return
	}
	if n.Props != nil {
		if _, ok := n.Props["onClick"]; ok {
			return
		}
		if _, ok := n.Props["onLongPress"]; ok {
			return
		}
	}
	upsertConcern(ConcernInertDisclosure, fmt.Sprintf(
		"%s states AccessibilityExpanded %q and carries no OnClick or OnLongPress: "+
			"Compose says a disclosure with expand()/collapse() actions, which need a "+
			"handler to perform, so this is announced on the two web targets and is "+
			"silently nothing on Android",
		path, n.Style.AccessibilityExpanded))
}

// report resolves what the walk collected. Both findings need the whole tree,
// which is the reason this pass exists at all.
//
// Sorted before reporting, because a map iterates in a random order and the
// collector deduplicates on the detail string: an unsorted list of paths would
// make the same duplicate id read as a new finding on the passes where the
// order came out differently, and the count that is supposed to say "this has
// been wrong for 200 frames" would sit at 1 forever.
func (a *a11yAudit) report() {
	for id, paths := range a.ids {
		if len(paths) < 2 {
			continue
		}
		sort.Strings(paths)
		upsertConcern(ConcernDuplicateAccessibilityID, fmt.Sprintf(
			"AccessibilityID %q is claimed by %d elements (%s): ids are document-global "+
				"and nothing rewrites them, so the document is invalid and every "+
				"aria-controls pointing here resolves to whichever the browser saw first",
			id, len(paths), strings.Join(paths, ", ")))
	}
	for _, ref := range a.controls {
		if _, ok := a.ids[ref.target]; ok {
			continue
		}
		upsertConcern(ConcernDanglingReference, fmt.Sprintf(
			"%s has AccessibilityControls %q and no element in the tree has that "+
				"AccessibilityID: the attribute is written anyway, and a reader "+
				"following it announces a control that governs nothing",
			ref.path, ref.target))
	}
}
