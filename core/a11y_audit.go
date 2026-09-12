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
// Four of the seven findings below are about *relationships between elements*,
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
// # Why these seven and not a general ARIA validator
//
// Each of them is a failure with no symptom. A duplicate id resolves to
// whichever element the browser saw first, so a tab strip switches the wrong
// region and nothing errors. A dangling aria-controls announces a tab that
// governs nothing, which sounds exactly like a tab. An id with a space in it is
// invalid HTML that browsers accept and getElementById never finds. A
// disclosure with no handler offers TalkBack an action nothing performs. A
// selection-follows-focus flag on a role with no arrow keys is a contract
// about a keyboard nobody has. A composite inside a composite is two tab stops
// where ARIA describes one, and both widgets work. A range whose numbers are
// not numbers is resolved by every target and by no two of them the same way,
// while the bar on screen goes on drawing the caller's own float.
//
// Everything else in ARIA that this vocabulary can express is already caught
// where it is written: a state on a role that cannot carry it is dropped by
// both exporters *by design* and documented at each guard, and a structural
// role over foreign children is a judgement about content that no walk can
// make. The line is "would a reader be told something false, with nothing
// anywhere saying so".
//
// # The eighth finding, which is not about accessibility
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
// this file opened with were the first family, not the only one.
//
// # Cost
//
// One walk of the finished tree per pass, in debug mode only — the same
// condition the other two checks run under, and the same order of work the
// reconciler is about to do anyway. Off, it is a single atomic load at the call
// site in render.Manager. The placement check adds one map lookup per node.

// Concern kinds for the accessibility audit. Declared here rather than beside
// the others in debug.go so the seven arrive with the walk that produces them.
// The walk's eighth finding, ConcernInertPlacement, is declared in
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

	// ConcernInertFollowsFocus: a node states
	// core.Style.AccessibilitySelectionFollowsFocus and carries no role that
	// has a keyboard for a selection to follow. The WASM runtime writes the
	// data attribute on any node that asks — deliberately, since consulting
	// the composite tables where an attribute is written would put those
	// tables in two places — so the flag lands in the DOM and is read by
	// nothing, and no other target writes anything for it at all.
	ConcernInertFollowsFocus = "inert-follows-focus"

	// ConcernUnusableValueRange: a node states a core.Style.AccessibilityValue
	// whose numbers no platform can use as written — a position or a bound
	// that is not a number, or a range whose Max is at or below its Min. Every
	// target resolves it and no two of them resolve it the same way, so the
	// bar announces a different wrong number on each. See checkValueRange.
	ConcernUnusableValueRange = "unusable-value-range"

	// ConcernNestedComposite: a container with an ARIA keyboard pattern sits
	// inside another one. Both keep their own roving tabindex, so the pair is
	// two tab stops where ARIA describes one — the outer widget's arrows step
	// over the inner widget whole. It is deliberate and it is a divergence;
	// see checkNestedComposite.
	ConcernNestedComposite = "nested-composite"
)

// AuditTree runs the whole-tree checks over a finished render tree: the seven
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
	// "" is the placing container above the root: nothing. See placerFor. The
	// zero compositeAncestor is the same statement about keyboards: nothing
	// above the root has one.
	a.walk(root, "root", "", compositeAncestor{})
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
func (a *a11yAudit) walk(n *Node, path, placer string, composite compositeAncestor) {
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
		a.checkFollowsFocus(n, path)
		a.checkValueRange(n, path)
		reportInertPlacement(n, path, placer)
		composite = a.checkNestedComposite(n, path, composite)
	}
	childPlacer := placerFor(n.Type, placer)
	for i, child := range n.Children {
		a.walk(child, fmt.Sprintf("%s/%d", path, i), childPlacer, composite)
	}
}

// compositeAncestor is the nearest enclosing container that has a keyboard
// pattern, or the zero value above the outermost one.
//
// Threaded down the walk rather than looked up, for the reason placer is: a
// node carries no parent pointer, and the walk already knows the answer. Both
// halves are kept because the finding needs to name where the outer widget is
// as well as what it is — "a tablist inside a toolbar" is the fact, and
// "root/2" is what makes it findable.
type compositeAncestor struct {
	role Role
	path string
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

// checkFollowsFocus flags a keyboard contract stated on a widget that has no
// keyboard.
//
// core.AccessibilitySelectionFollowsFocus says what a composite's arrow keys
// do — choose the member they land on, not merely focus it — and there are
// exactly three container roles in this framework whose arrows do anything
// (core.KeyboardComposites). On anything else the flag is inert in the
// strongest sense available: the WASM runtime writes
// data-grmob-selection-follows-focus for any node that asks, so the claim is
// in the document and no code path ever reads it back, while htmlout and both
// natives write nothing for it on purpose.
//
// # Why here rather than at the writer
//
// The runtime's applyAccessibility says why in as many words: it does not
// consult the composite tables, because knowing them where attributes are
// written would make the tables a fact in two places. That is the right call
// and it leaves the flag unchecked everywhere — which is exactly the shape
// this walk exists for, a claim that is silently true of nothing.
//
// # The empty role is not exempt
//
// A Box with the flag and no role at all is the commonest way to get here (a
// tab strip whose core.AccessibilityRole was forgotten, or moved to a child),
// and it is the case worth reporting most: the widget looks like a tab strip,
// reads as a plain group, and has no arrow keys — so the missing role is the
// bug and the flag is the only evidence anyone intended one.
//
// # A toolbar counts, and this is not an endorsement
//
// The runtime's arrow handler funnels every composite through one function, so
// a toolbar carrying the flag does invoke its controls on arrow. Whether that
// is a good idea for a toolbar is a separate judgement — ARIA recommends
// selection-follows-focus for tabs, allows it for a single-select listbox and
// says nothing about toolbars — and this check's subject is only whether
// anything reads the flag. Reporting a role that does read it would be a
// second, weaker claim wearing this one's name.
func (a *a11yAudit) checkFollowsFocus(n *Node, path string) {
	if !n.Style.AccessibilitySelectionFollowsFocus {
		return
	}
	role := n.Style.AccessibilityRole
	if hasKeyboard(role) {
		return
	}
	stated := "carries no AccessibilityRole"
	if role != "" {
		stated = fmt.Sprintf("has AccessibilityRole %q", role)
	}
	upsertConcern(ConcernInertFollowsFocus, fmt.Sprintf(
		"%s states AccessibilitySelectionFollowsFocus and %s: the flag is a "+
			"statement about what a composite's arrow keys do, and only %s have "+
			"one — so this is written into the DOM as "+
			"data-grmob-selection-follows-focus, read by nothing, and written at "+
			"all by no other target",
		path, stated, joinRoles(KeyboardComposites())))
}

// checkValueRange flags a stated range that no platform can use as written.
//
// # Why this is a finding and the guards are not
//
// Everything else about core.Style.AccessibilityValue is already answered
// where it is written. A range on a role that cannot carry one is dropped by
// both web exporters *by design*, documented at each guard and tested on both
// targets; a role with no range at all is ARIA's own spelling of an
// indeterminate bar and is not a mistake. Neither belongs here, for the reason
// this file's header gives.
//
// What is left is the case where the author stated numbers and the numbers are
// not usable — and it is the one shape in this vocabulary that fails
// *differently on every target*:
//
//	                     Now: "half"            Min: "9", Max: "1"
//	web (Chrome)         a bar pinned at 0      a bar pinned at 9
//	Compose              indeterminate          the property is dropped
//	SwiftUI              nothing either way     nothing either way
//	core.Progress        indeterminate          empty-range
//
// Not one of those is what was written, none of them errors, and the bar looks
// correct on screen in every case — the fill is drawn from the caller's own
// float, which never went through this vocabulary at all. So the number a
// sighted user sees and the number a reader announces disagree, silently, and
// the only place that can notice is a walk of the finished tree.
//
// # Why it asks ValueRange rather than reading the strings
//
// The two questions are core.ValueRange's own: Unparsed names the fields that
// are not numbers, and Progress says what the three of them amount to once
// ARIA's defaults and clamping are applied. Restating either here would be a
// second copy of the rule the renderers are held to — and the empty-range
// arm in particular is not a string test at all, since "45" with no bounds is
// fine and "5" between 9 and 1 is not.
//
// This is also the audit's first use of Progress, which until now had no Go
// consumer: comps.ProgressBar states all three numbers itself and is
// determinate by construction, so the only readers of the reading were a test
// and a Kotlin transliteration.
func (a *a11yAudit) checkValueRange(n *Node, path string) {
	v := n.Style.AccessibilityValue
	if !v.Stated() {
		return
	}

	// The fields first, because an unparseable bound is *also* what produces
	// a nonsense range downstream, and naming the field is the more useful
	// half of the report.
	if bad := v.Unparsed(); len(bad) > 0 {
		upsertConcern(ConcernUnusableValueRange, fmt.Sprintf(
			"%s states an AccessibilityValue whose %s not a number: %s. A browser "+
				"reads an unparseable aria-value* as 0 and pins the bar there, Compose "+
				"reads it as absent and announces an indeterminate bar, and core.Progress "+
				"agrees with Compose — three answers, none of them the one written, and "+
				"the bar on screen goes on drawing the caller's own float",
			path, fieldsAre(bad), quoteFields(v, bad)))
		return
	}

	if p := v.Progress(); p.Reading == ProgressEmptyRange {
		upsertConcern(ConcernUnusableValueRange, fmt.Sprintf(
			"%s states an AccessibilityValue whose range is empty: Max %s is at or "+
				"below Min %s, so there is no position for Now %s to be at. Compose "+
				"cannot express it — ProgressBarRangeInfo throws on an empty range, so "+
				"Renderer.kt drops the property rather than crashing a render over an "+
				"annotation — and a browser keeps the numbers and clamps the position "+
				"to whichever end it lands outside",
			path, quote(v.Max), quote(v.Min), quote(v.Now)))
	}
}

// fieldsAre reads "field Now is" or "fields Now and Max are", so the report is
// a sentence in both the one-field case and the several-field one.
func fieldsAre(names []string) string {
	if len(names) == 1 {
		return "field " + names[0] + " is"
	}
	return "fields " + strings.Join(names[:len(names)-1], ", ") +
		" and " + names[len(names)-1] + " are"
}

// quoteFields prints the offending values, so the report names the string
// rather than only the field it was in — which is the half a reader needs, as
// the value is usually a formatting bug ("45%", "1.048576e+06") rather than a
// typo.
func quoteFields(v ValueRange, names []string) string {
	parts := make([]string, len(names))
	for i, name := range names {
		var value string
		switch name {
		case "Now":
			value = v.Now
		case "Min":
			value = v.Min
		case "Max":
			value = v.Max
		}
		parts[i] = name + " = " + quote(value)
	}
	return strings.Join(parts, ", ")
}

// quote renders a stated value, or the word for an unstated one — an empty
// string in this report means ARIA's default was used, which is a different
// statement from a value of "".
func quote(s string) string {
	if s == "" {
		return "(unstated, so ARIA's default)"
	}
	return "\"" + s + "\""
}

// checkNestedComposite flags one keyboard pattern inside another, and returns
// the ancestor its own children should be told about.
//
// # The shape, and why it is two tab stops
//
// The WASM runtime finds a composite's members by walking its subtree. The
// inner container keeps its own roving tabindex whatever the pair, so the
// pair is always two stops in the page's tab order:
//
//	Tab  -> [ All ] ( Sermons | Articles ) [ More ]     the toolbar's stop
//	Tab  ->         (   ^ the strip's own stop   )      the tablist's
//
// ARIA describes one: the pattern makes the nested widget's *current* member
// the outer widget's member, so a single Tab reaches the pair and the outer
// arrows cross into the strip.
//
// # What the outer arrows then do, which is not one answer
//
// This used to be reported as one outcome — "the outer widget's arrows step
// over the inner one whole" — on the strength of both member walks stopping at
// a nested composite. Only one of them stops unconditionally. The rule is
// core.CompositeWalkAt, and for a pair that are both composites — which is the
// only pair this reaches — it has three cases:
//
//	toolbar around anything      stops. Its members are named by no role at
//	                             all (ARIA defines no `toolbaritem`), so
//	                             nothing says whose a button inside a nested
//	                             widget is.
//	listbox in listbox,          stops. The two would otherwise pool their
//	tablist in tablist           options, and one widget's arrows would walk
//	                             out into the other's rows.
//	listbox around tablist,      DESCENDS. An option below a tablist is still
//	tablist around listbox,      the listbox's option — the roles say whose it
//	either around a toolbar      is — so the walk carries on through.
//
// The finding says which, because the two are different bugs to go looking
// for. A stopped pair loses the inner widget from the outer's rotation; a
// descending pair gains members the author never offered the outer widget,
// and the arrows land inside a widget they are not steering.
//
// Stated in core rather than derived here so that one rule answers for the
// audit's sentence and for the runtime's two walks at once; wasm/verify's
// keynav_test.go is what holds the runtime to it, and keynav_test.mjs
// exercises a descending pair in a real DOM.
//
// Read through CompositeWalkAt and not CompositeWalkStopsAt. The bool has a
// fourth case the three above do not name — an outer role with no keyboard,
// for which it answers `true` because that is the safe reading rather than a
// true one — and this sentence is the one place in the repository where
// writing the stop wording for that case would produce a finding that
// describes a rotation the tree does not have.
//
// # Why the framework diverges rather than fixing it
//
// Doing what ARIA describes means two composites writing tabindex onto one
// element — the strip's selected tab would be both the tablist's stop and the
// toolbar's member — and that needs a rule about which of them owns the write
// when they disagree. There is no such rule, and inventing one silently is
// worse than the divergence: what the framework does instead leaves every
// control reachable, which every guess here can lose.
//
// # Why it is reported here and nowhere else
//
// It is a relationship between two elements, which is this walk's whole
// subject. The runtime could see it — the walk is standing on both nodes when
// it stops — but a runtime that reported it would be reporting it to a browser
// console at the moment a keystroke arrives, on a target the author may not be
// running. This is the pass whose job is telling an author, in Go, what their
// finished tree amounts to.
//
// A concern rather than a refusal: the shape works, every control in it is
// reachable, and nothing in this repository builds one. What the author is
// owed is knowing they built the two-stop version on purpose.
func (a *a11yAudit) checkNestedComposite(
	n *Node, path string, outer compositeAncestor,
) compositeAncestor {
	role := n.Style.AccessibilityRole
	if !hasKeyboard(role) {
		// Not a composite: whatever the ancestor was, it still is. The walk
		// does not stop at ordinary containers, which is the point — a strip
		// buried three Boxes deep inside a toolbar is the same finding.
		return outer
	}
	if outer.role != "" {
		// The half that never varies, and the half that does. Both containers
		// keep a roving tabindex whatever the pair, so it is always two tab
		// stops; what the outer widget's arrows then do depends on whether its
		// member walk stops at this node or descends through it, and that rule
		// is core.CompositeWalkStopsAt — stated once, and pinned to the
		// runtime's two walks by wasm/verify's keynav_test.go.
		//
		// Reporting the wrong one of these is worse than reporting neither: an
		// author told the strip is "stepped over" will not go looking for the
		// option of theirs that the outer widget's arrows can now land on.
		// The second return is discarded here and only here: this branch runs
		// for a pair AuditTree has already established are both composites, so
		// the answer is known to be true. Spelling it out is what keeps the
		// discard a statement rather than a habit.
		outerMembers, _ := CompositeMemberRole(outer.role)
		var reach string
		// Three arms rather than an if/else on CompositeWalkStopsAt, because
		// core.CompositeWalkAt now answers with three values and the third one
		// is the case this sentence must not be written for. The bool spelling
		// folds "there is no walk" into "the walk stops", which would put the
		// step-over sentence on a pair where nothing steps over anything.
		switch CompositeWalkAt(outer.role, role) {
		case CompositeWalkStops:
			reach = "the outer widget's arrows step over the inner one whole"
		case CompositeWalkDescends:
			reach = fmt.Sprintf(
				"the outer widget's walk descends through this one, so any %q of "+
					"its own buried inside the %q is pooled into the outer widget's "+
					"rotation and its arrows land inside a widget they are not "+
					"steering",
				outerMembers, role)
		case CompositeWalkNotApplicable:
			// Not reachable from here: both ends of the pair have passed
			// hasKeyboard, which is exactly the membership CompositeMemberRole
			// reports as its second return. It is written out anyway, and it
			// says nothing rather than guessing — a role with no walk has no
			// nested-composite finding to make, and the ancestor is still
			// updated below so the next composite down is compared against
			// this node.
			return compositeAncestor{role: role, path: path}
		}
		upsertConcern(ConcernNestedComposite, fmt.Sprintf(
			"%s is a %q inside the %q at %s: both keep their own roving tabindex, "+
				"so the pair is two tab stops and %s. ARIA describes one stop — it "+
				"makes the inner widget's current member the outer widget's member — "+
				"and doing that means two widgets writing tabindex onto one element, "+
				"which needs an owner rule this framework does not have. Every "+
				"control stays reachable either way",
			path, role, outer.role, outer.path, reach))
	}
	return compositeAncestor{role: role, path: path}
}

// hasKeyboard reports whether a role is one of the containers with an ARIA
// keyboard pattern. A loop over three values rather than a package-level set,
// which any importer could write to — the same trade Roles() makes.
func hasKeyboard(role Role) bool {
	for _, composite := range KeyboardComposites() {
		if role == composite {
			return true
		}
	}
	return false
}

// joinRoles renders a role list for a message. Small enough to inline and
// separate because the list comes from core.KeyboardComposites(): spelling the
// three roles into the sentence would be the fourth hand-written copy of them.
func joinRoles(roles []Role) string {
	out := make([]string, len(roles))
	for i, r := range roles {
		out[i] = string(r)
	}
	return strings.Join(out, ", ")
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
