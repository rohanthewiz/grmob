# Package core — Accessibility

```go
import "github.com/rohanthewiz/grmob/core"
```

Roles, selected and expanded states, value ranges and the accessibility audit.

One of 11 topic pages of [package core](core.md), which has the package overview and an index of every topic. This page documents the declarations in `core/role.go`, `core/popup.go`, `core/selected.go`, `core/expanded.go`, `core/value.go`, `core/a11y_audit.go`.

## Index

- [Constants](#constants) — `ConcernDanglingReference`, `ConcernDuplicateAccessibilityID`, `ConcernInertDisclosure`, `ConcernInertFollowsFocus`, `ConcernInvalidAccessibilityID`, `ConcernNestedComposite`, `ConcernUnusableValueRange`
- [`func AuditTree`](#func-audittree)
- [`func CompositeWalkStopsAt`](#func-compositewalkstopsat)
- [`type CompositeWalk`](#type-compositewalk)
    - [`func CompositeWalkAt`](#func-compositewalkat)
    - [`func (CompositeWalk) String`](#func-compositewalk-string)
- [`type ExpandedState`](#type-expandedstate)
    - [`func ExpandedStates`](#func-expandedstates)
    - [`func ExpandedWhen`](#func-expandedwhen)
- [`type PopupKind`](#type-popupkind)
    - [`func PopupKinds`](#func-popupkinds)
- [`type Progress`](#type-progress)
- [`type ProgressReading`](#type-progressreading)
- [`type Role`](#type-role)
    - [`func CompositeMemberRole`](#func-compositememberrole)
    - [`func KeyboardComposites`](#func-keyboardcomposites)
    - [`func Roles`](#func-roles)
    - [`func TappableContainerRoles`](#func-tappablecontainerroles)
- [`type SelectedState`](#type-selectedstate)
    - [`func SelectedStates`](#func-selectedstates)
    - [`func SelectedWhen`](#func-selectedwhen)
- [`type ValueRange`](#type-valuerange)
    - [`func ValueOf`](#func-valueof)
    - [`func (ValueRange) Progress`](#func-valuerange-progress)
    - [`func (ValueRange) Stated`](#func-valuerange-stated)
    - [`func (ValueRange) Unparsed`](#func-valuerange-unparsed)
    - [`func (ValueRange) WithText`](#func-valuerange-withtext)

## Constants

Concern kinds for the accessibility audit. Declared here rather than beside the others in debug.go so the seven arrive with the walk that produces them. The walk's eighth finding, ConcernInertPlacement, is declared in placement\_audit.go for the same reason: beside the argument for it.

```go
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
```

<small>[core/a11y_audit.go:73](https://github.com/rohanthewiz/grmob/blob/master/core/a11y_audit.go#L73)</small>

## Functions

### func AuditTree

```go
func AuditTree(root *Node)
```

AuditTree runs the whole-tree checks over a finished render tree: the seven accessibility findings above and the placement finding beside them.

Called by the render driver after the pass that produced the tree, beside EndRenderPass and for the same reason: both describe a \*completed\* pass, and running either over a half-built tree reports nothing except that something went wrong earlier. A no-op (one atomic load) when debug mode is off.

Hosts that drive passes through render.Manager get this for free. A hand-rolled pass loop should call it with the tree it just rendered.

nil is not an error: a pass that produced no tree has nothing to audit, which is the state a host is in before its first render.

<small>[core/a11y_audit.go:140](https://github.com/rohanthewiz/grmob/blob/master/core/a11y_audit.go#L140)</small>

### func CompositeWalkStopsAt

```go
func CompositeWalkStopsAt(outer, inner Role) bool
```

CompositeWalkStopsAt is CompositeWalkAt for a caller that only wants the bool, and it is kept because that is what the two runtimes' walks and the audit's sentence actually ask: "does my rotation reach inside this node".

It answers true for both non-descending values, which is exactly the conflation CompositeWalkAt exists to undo — so this is safe only for a caller that has already established both roles are composites, and every caller in this repository has (AuditTree tests hasKeyboard on both ends before it asks, and wasm/verify's pins iterate KeyboardComposites()). A caller that has not should ask CompositeWalkAt and handle the third value, because for a role with no walk this returns a confident \`true\` about a rotation that does not exist.

<small>[core/role.go:912](https://github.com/rohanthewiz/grmob/blob/master/core/role.go#L912)</small>

## Types

### type CompositeWalk

```go
type CompositeWalk int
```

CompositeWalk is what an outer composite's member walk does at a nested composite: one of two answers, or the value that says the question does not apply.

#### Why the third value exists

This started as CompositeWalkStopsAt alone, returning a bool. Two of its three arms returned true — a container with no keyboard, and a toolbar whose walk genuinely does stop — and the two are different facts: one is "the arrows step over this node", the other is "there are no arrows". The distinction was real in the code, with a paragraph on each arm saying so, and invisible from outside: a caller holding the \`true\` could not tell which it had, and the safe reading of a non-composite ("stops") is a confident statement about a walk that does not exist.

Making it a value rather than a doc note is the same move CompositeMemberRole made one function up when its two empty answers became (member, composite): the fact is put where the compiler and the caller can both see it, instead of in a sentence asking the caller to have already checked something.

<small>[core/role.go:935](https://github.com/rohanthewiz/grmob/blob/master/core/role.go#L935)</small>

```go
const (
	// CompositeWalkNotApplicable: the outer role has no keyboard, so it has no
	// member walk and neither of the other two values is true of it.
	//
	// Zero so that a CompositeWalk nobody assigned reads as "no claim" rather
	// than as one of the answers — the same trade every unset field in
	// core.Style makes, and it is safe here for the reason it is not safe for
	// flex-shrink (see ShrinkNone): the useful default really is the empty one.
	CompositeWalkNotApplicable CompositeWalk = iota

	// CompositeWalkStops: the outer widget's arrows step over the inner one
	// whole. Its members are still its own; what it loses is any element of
	// its member role buried inside the nested widget.
	CompositeWalkStops

	// CompositeWalkDescends: the outer widget's walk carries on through the
	// inner one, so an element of the outer's member role inside the nested
	// widget's subtree is pooled into the outer's rotation and the arrows can
	// land inside a widget they are not steering.
	CompositeWalkDescends
)
```

#### func CompositeWalkAt

```go
func CompositeWalkAt(outer, inner Role) CompositeWalk
```

CompositeWalkAt reports what an outer composite's member walk does when it meets a nested composite: stop there, descend through it, or — the case that is not an answer — nothing, because the outer role has no walk.

##### Why core states a rule about a walk in another language

AuditTree is the reader. ConcernNestedComposite is a finding about a pair of containers, and \*what goes wrong\* is not the same for every pair — so a report that did not know this rule could only describe one of the two outcomes, and would describe the other one wrongly. That is the same argument KeyboardComposites makes one level up: the audit is the pass whose job is telling a Go author what their finished tree amounts to, and it cannot get the answer from a runtime written in JavaScript.

##### The rule

	outer has no keyboard          not applicable. There is no member walk, so
	(a heading, a Box)             neither answer is true of it — see
	                               CompositeWalkNotApplicable for why that is a
	                               value rather than a `true` chosen for safety.

	outer names no member role     stop. Nothing says whose a plain button is,
	(toolbar)                      so a control inside a nested composite
	                               belongs to the nested one.

	inner has the same members     stop. Two listboxes pooling their options
	                               would let one widget's arrows walk out into
	                               the other's rows.

	otherwise                      descend. An option below a tablist is still
	                               the listbox's option — the roles say whose
	                               it is, and stopping would lose a member the
	                               vocabulary has already assigned.

Member roles are unique per container, so the middle case is the same set of pairs as \`inner == outer\`; it is written against the member role because that is the question the runtime's walk actually asks (\`compositeMemberRole(child) === memberRole\`), and a fourth pattern sharing a member role with a third would land here rather than in a surprise.

##### What descending costs, and why it is still right

A descending pair is still two tab stops — both containers keep a roving tabindex either way, which is the part of the finding that never varies. What differs is the reach: where a stopping pair's outer arrows step over the inner widget whole, a descending pair's outer arrows can land \*inside\* it, on any element of the outer's member role buried in the inner's subtree. Neither is what ARIA describes for nested composites, and the framework's refusal to guess is documented at ConcernNestedComposite.

<small>[core/role.go:876](https://github.com/rohanthewiz/grmob/blob/master/core/role.go#L876)</small>

#### func (CompositeWalk) String

```go
func (w CompositeWalk) String() string
```

String names the value for a message. The three spellings are the words the audit's finding and this file's docs already use, so a report built from a %v and a report written by hand read the same.

<small>[core/role.go:962](https://github.com/rohanthewiz/grmob/blob/master/core/role.go#L962)</small>

### type ExpandedState

```go
type ExpandedState string
```

ExpandedState is whether a disclosure is \*open\* — the accordion section showing its body, the twisty that has been turned.

It is the third of the state types, after SelectedState, and it exists because a control can be on and open at the same time. comps.Accordion is the widget that asked for it: it is the one stateful widget in the components package, its header is the only thing on screen that knows whether the section is showing, and until this type existed the only thing that said so was a chevron glyph — which a screen reader announces as "black right-pointing small triangle" or, more often, not at all.

#### Why this is a second type and not SelectedState reused

The two carry the same three values for the same reason (see below), and reusing the type would have compiled, exported and rendered. It is turned down on two grounds.

\*They are independent facts about one node.\* A control can be selected and expanded at once — a menu button that is both the current tab and showing its submenu — so they are two fields, and two fields typed the same are two fields a caller can transpose. core.AccessibilitySelected(ExpandedOpen) is nonsense that would type-check.

\*They are scoped differently.\* ARIA defines aria-selected for four of this framework's roles and aria-pressed for a fifth; aria-expanded is defined for a sixth set that overlaps but does not match. A shared type would suggest a shared guard, and the guards are what make each state legal where it is written.

#### Why three values and not a bool

The same argument SelectedState makes, arriving from the opposite end. There "off" and "not selectable" were two facts a bool has one spelling for; here it is "closed" and "not a disclosure".

	ExpandedUnset    a Box, a heading, a row of text. Makes no claim, which is
	                 what every node in every tree was before this field, so
	                 the zero value is a no-op.
	ExpandedOpen     the section that is showing.
	ExpandedClosed   a disclosure that could be open and is not.

The third is again the one that would be lost, and losing it is worse here than it is for a selection. A closed accordion that says nothing is announced as an ordinary button: a reader is told they can press it and not that there is anything behind it. "Collapsed" is the whole of what invites the press.

#### Why the values are ARIA's own spellings

Because both web targets write them into the attribute verbatim and need no mapping table, which is the trade core.Role and SelectedState both made. The two natives map what they can, which here is one of them — see Style.AccessibilityExpanded.

<small>[core/expanded.go:56](https://github.com/rohanthewiz/grmob/blob/master/core/expanded.go#L56)</small>

```go
const (
	// ExpandedUnset is the zero value: this node is not a disclosure and says
	// nothing about being open. Every renderer writes no attribute and offers
	// no action.
	ExpandedUnset ExpandedState = ""

	// ExpandedOpen — the disclosure is showing its content.
	ExpandedOpen ExpandedState = "true"

	// ExpandedClosed — the disclosure is a disclosure and is shut. See the
	// type doc for why this is not the same as ExpandedUnset.
	ExpandedClosed ExpandedState = "false"
)
```

#### func ExpandedStates

```go
func ExpandedStates() []ExpandedState
```

ExpandedStates returns both stated values, in declaration order.

ExpandedUnset is excluded for the reason SelectedStates() excludes its own zero value: it is the absence of a claim rather than one of the states, and a coverage check that demanded a renderer arm for it would be asking each renderer to implement "unstated".

Pinned to the const block above by expanded\_enum\_test.go, and consumed by the exporters' round-trip checks the way SelectedStates() is.

<small>[core/expanded.go:95](https://github.com/rohanthewiz/grmob/blob/master/core/expanded.go#L95)</small>

#### func ExpandedWhen

```go
func ExpandedWhen(open bool) ExpandedState
```

ExpandedWhen turns the bool a disclosure already holds into the stated pair.

The twin of SelectedWhen, and it earns its place the same way: the widget owns a \`expanded bool\` (comps.Accordion holds one in NewState), so the conversion would otherwise be written by hand at each call site, and the tempting hand-rolled version — set ExpandedOpen when open, leave it alone otherwise — is exactly the silence the third value exists to prevent.

<small>[core/expanded.go:79](https://github.com/rohanthewiz/grmob/blob/master/core/expanded.go#L79)</small>

### type PopupKind

```go
type PopupKind string
```

PopupKind is what a control opens: the kind of element that appears when it is activated, stated so a reader can say so before the press.

It is the vocabulary of aria-haspopup, and it exists for the near miss Style.AccessibilityExpanded names and turns down. A trigger that opens a modal looks exactly like a disclosure — comps.DatePicker's even flips a glyph — and it is not one: aria-expanded says the content is here, in the page, and can be shown or hidden, where a popup is a new surface the reader is moved into. Before this type the honest thing was to say nothing, and a reader pressing comps.Menu's "⋯" was told "button" and then found itself in a dialog with no warning.

#### Why one value, when ARIA has seven

ARIA's attribute takes false, true, menu, listbox, tree, grid and dialog. This carries the one a widget here can honestly say:

	dialog    comps.Menu and comps.DatePicker both present through core.Modal,
	          which both web targets write as role="dialog". The value names
	          what the reader actually lands in.
	menu      also `true`, its synonym. comps.Menu is *called* a menu and is
	          not one to a reader: its sheet is a dialog of buttons, and
	          core.Role has no `menu`/`menuitem` pair
	          (aria/verify/refusals_test.go records why). Saying menu would
	          announce a menu keyboard — arrows between items, Escape back to
	          the trigger — that nothing supplies.
	listbox   a combobox's popup is a listbox implicitly, so the one widget
	          with a listbox popup (comps.SearchableSelect, through
	          RoleComboBox) needs no attribute, and no other trigger opens one.
	tree      no widget has either, and both are refused patterns.
	grid
	false     the absence of a claim, which is PopupNone.

A value is added when a widget opens that kind of surface, not before: a constant naming a popup nothing presents would be a claim the framework cannot keep.

#### Why the values are ARIA's own spellings

The same trade core.Role, SelectedState and ExpandedState made: both web targets write the value verbatim and need no table.

#### What each target does with it

The two web targets write aria-haspopup, scoped to the roles ARIA defines it on (see Style.AccessibilityHasPopup). Neither native has anything to put it in — Compose's semantics and SwiftUI's traits have no popup property — and both present a Modal through a platform dialog that announces itself when it opens, which is the half of the warning that matters most there. The key crosses the bridge and is deliberately unparsed on both.

<small>[core/popup.go:53](https://github.com/rohanthewiz/grmob/blob/master/core/popup.go#L53)</small>

```go
const (
	// PopupNone is the zero value: this control opens nothing, or says
	// nothing about what it opens. Both web targets write no attribute.
	PopupNone PopupKind = ""

	// PopupDialog — activating this control opens a dialog. See the type doc
	// for why this is the only kind a widget here can say.
	PopupDialog PopupKind = "dialog"
)
```

#### func PopupKinds

```go
func PopupKinds() []PopupKind
```

PopupKinds returns every stated value, in declaration order. PopupNone is excluded for the reason ExpandedStates() excludes its own zero value: it is the absence of a claim rather than one of the kinds.

<small>[core/popup.go:68](https://github.com/rohanthewiz/grmob/blob/master/core/popup.go#L68)</small>

### type Progress

```go
type Progress struct {
	Reading       ProgressReading
	Now, Min, Max float64
}
```

Progress is a ValueRange's numbers, resolved.

Now, Min and Max are meaningful when Reading is ProgressDeterminate. For ProgressEmptyRange they are the numbers as stated, unclamped, so a caller reporting the problem can name them; for the other two readings they are zero, which is not a position.

<small>[core/value.go:192](https://github.com/rohanthewiz/grmob/blob/master/core/value.go#L192)</small>

### type ProgressReading

```go
type ProgressReading string
```

ProgressReading is what a ValueRange's three numbers amount to once somebody has to act on them.

#### Why core owns this and the web exporters do not use it

The two DOM targets hand aria-valuenow/-min/-max to a browser verbatim, and a browser applies ARIA's rules itself: the implicit 0..100, the reading of a missing position as an indeterminate bar. That pass-through is right and it is also why this reading had nowhere to live — the only code in the repository applying those rules was Kotlin, in a three-way branch inside a Compose semantics lambda, checked by looking for substrings in the file.

The rules are ARIA's and this type's, though, not Compose's: both of them are already stated in prose on the fields below, and the Kotlin is a transliteration of that prose. Naming them here makes the transliteration comparable — android/verify runs GrMobProgress.kt against this function over internal/valuefixture's table — which is the same relationship core.SelectMenuSections has with the four picker menus.

Text has no part in it. The words are a separate claim on a separate property (aria-valuetext, stateDescription, accessibilityValue) and they are announced on nodes that carry no range at all, so a reading about the numbers must not depend on them.

<small>[core/value.go:160](https://github.com/rohanthewiz/grmob/blob/master/core/value.go#L160)</small>

```go
const (
	// Nothing numeric was stated. A `text` on an ordinary node must not turn
	// it into a progress bar, so this is the reading that leaves a platform's
	// range property untouched.
	ProgressUnstated ProgressReading = "unstated"

	// Bounds and no position: a bar that is running with no idea how far.
	// ARIA spells it by omitting aria-valuenow; Compose has a name for it.
	ProgressIndeterminate ProgressReading = "indeterminate"

	// A position inside a real range. Min and Max carry ARIA's own defaults
	// of 0 and 100 when unstated, which is what makes a bare Now announce as
	// a percentage, and Now is clamped into the range.
	ProgressDeterminate ProgressReading = "determinate"

	// A position inside a range that is not one — Max at or below Min. It is
	// separated from Unstated because the two are different mistakes and a
	// platform may want to treat them differently: Compose cannot express it
	// at all (ProgressBarRangeInfo requires a non-empty range and throws), so
	// it drops the property rather than crashing a render over an
	// accessibility annotation.
	ProgressEmptyRange ProgressReading = "empty-range"
)
```

### type Role

```go
type Role string
```

Role is what a node \*is\* to assistive technology, as distinct from what it is called (AccessibilityLabel) or what tapping it does (AccessibilityHint).

A screen reader announces "Sermons, heading" or "March, column header" because something told it the element's kind. Nothing in this framework could say that until now: every container is a Box or a Row, every one of them exports as a \<div>, and a screen built entirely out of them is a flat run of text to VoiceOver and TalkBack no matter how carefully it is labelled. Three widgets hit the wall independently — DataTable wanting to be a table, the screen-furniture bundle wanting a banner, a heading and a search landmark, and Calendar's forty-two tappable day cells wanting to be buttons — which is what turned "an ARIA role prop, some day" into this file.

#### Why the values are spelled in ARIA

The set has to be \*some\* vocabulary, and the four renderers do not share one. ARIA is the only candidate that is a published standard with a name for every case here; SwiftUI's AccessibilityTraits and Compose's SemanticsProperties are small, partly overlapping sets that would each need a mapping table whichever vocabulary core picked. Choosing ARIA means the two DOM targets need no table at all — the value is the attribute — and the two natives map what they can, which is the same work they already do for ContentMode and the alignments.

#### What each target does with a role

	role          | DOM (both)      | SwiftUI trait  | Compose semantics
	--------------+-----------------+----------------+------------------------
	heading       | role="heading"  | .isHeader      | heading()
	columnheader  | role=…          | .isHeader      | heading()
	button        | role="button"   | .isButton      | role = Role.Button
	link          | role="link"     | .isLink        | —
	search        | role="search"   | .isSearchField | —
	img           | role="img"      | .isImage       | role = Role.Image
	tab           | role="tab"      | —              | role = Role.Tab
	tablist       | role="tablist"  | .isTabBar      | —
	radiogroup    | role=…          | —              | selectableGroup()
	radio         | role="radio"    | —              | role = Role.RadioButton
	status        | role="status"   | —              | liveRegion = Polite
	alert         | role="alert"    | —              | liveRegion = Assertive
	log           | role="log"      | —              | liveRegion = Polite
	progressbar   | role=…          | —              | — (but see below)
	the other 14  | role=…          | —              | —

The other fourteen are table, rowgroup, row, cell, list, listitem, listbox, option, tabpanel, banner, navigation, toolbar, combobox and group — the tabular set, both collection pairs, the region a tab shows, the landmarks, the field that owns a popup list, and the naming role.

progressbar has a row of its own because its dashes mean less than the others'. The \*role\* maps to nothing on either phone — neither has a word for what a progress bar is — while the value beside it maps to Compose's progressBarRangeInfo, which is one of the better mappings in this framework: TalkBack turns the numbers into a percentage it localizes itself. So the thing a reader most wants to hear does arrive on one native; it arrives through Style.AccessibilityValue rather than through this field. See core.ValueRange.

The tab pair is the one row of that table where the two natives disagree about \*which half\* they can say, and it is a useful illustration of why the vocabulary is ARIA's rather than either platform's. Compose has a Role.Tab for the control and nothing for the strip around it; SwiftUI has .isTabBar for the strip and nothing for the control. Neither could have supplied the pair, and a caller marking up a tab strip sets both and gets whichever half each platform knows.

Fifteen of the twenty-eight do nothing on either native, and that is the honest state of those platforms rather than a gap to be filled later: neither has a tabular semantics vocabulary a role can be mapped onto (Compose has collectionInfo, which describes counts and indices this prop does not carry), neither has a listbox in its semantics vocabulary (both spell a chosen item as a \*state\* instead, which is why the selectable pair costs them nothing to leave out — see RoleListBox), and neither has landmarks at all — VoiceOver's rotor navigates by heading, not by banner.

RoleGroup is the one empty pair in that fifteen that is empty for the opposite reason, and it is worth telling apart. The other fourteen are silent because the platform has no way to say the thing; \`group\` is silent because neither platform \*needs\* it — both honour an accessibility label on any node at all, and making that label legal is the whole of what the role does. See its own block below.

A role that maps to nothing is still worth setting. The web is a first-class target here, the mapping can improve later without the call sites changing, and a role that is right on one platform and inert on two is strictly better than a div.

#### A structural role owns what is inside it

The tabular five and the three collection pairs are not labels on a container — they are claims about what the container holds. role="list" says its children are listitems; role="listbox" says its children are options; role="table" says its children are rows, or rowgroups holding rows. A reader acts on the claim rather than re-deriving it: it announces the count ("list, five items"), it offers item-by-item navigation, and it reads the structure instead of the text.

A container can fail that claim in two directions, and both produce something worse than no role at all.

\*A gap in the chain.\* An unroled element between the container and its items breaks the ownership: role="table" wrapping a plain div wrapping the rows reports a table with no rows. This is what RoleRowGroup exists to close, and its comment below has the detail.

\*A foreign child.\* A container that holds something which is not an item — a footer, a heading, a spinner — is claiming a structure it does not have. ARIA specifies the children a role requires and not what to do with any others, so what a reader makes of the odd child is an implementation's choice rather than a promise: it may be counted, skipped, or announced as the item it is not.

So: \*\*a container that mixes items with chrome cannot take a structural role.\*\* Either the chrome moves outside the container, or the container stays roleless — in which case its contents are announced as the text they are, which is what everything was before this type existed and is not a regression. Reaching for the role anyway is the one move that makes the screen worse.

The rule is easy to meet by accident, because the shapes that hit it are the ordinary ones. DataTable meets it three times in one widget: it puts a rowgroup on its body list to close a gap, \*withholds\* that rowgroup when the list holds a placeholder instead of rows, and documents a grouped table's band headings — foreign children it cannot move — as a limit it cannot close from where it sits. A paged list in the first app to adopt these roles met it a fourth time without having read any of that: its "Load more" footer sits inside the core.List, so the list cannot be a list.

The landmarks, the live regions and the content roles carry no such promise and are not subject to this. A banner, a navigation region or a log owns whatever it likes; RoleHeading, RoleButton, RoleLink, RoleImg and RoleTab describe the node itself. Five of the const blocks below hold a role that makes a claim about its children — the tabular set, the three collection pairs, and the tablist half of the tab pair — which is where to look rather than here if a role is ever added to any of them. (RoleTab itself does not: it describes one control, the way RoleButton does, and only the strip around it claims what it contains.)

#### Roles a node type carries for itself

Some semantics are not the author's to state, because the node type already knows them. core.Button exports as a \<button> and builds a real control on both natives, so nobody has to say \`button\` — RoleButton exists for the \*other\* case, a Box or a Row with an OnTap, which every renderer draws as inert scenery.

core.Modal is the same shape and has no vocabulary entry at all. The two natives already present it through a platform dialog (a SwiftUI sheet, a Compose Dialog), each of which announces itself; the two DOM renderers drew a plain div, so the overlay was the one target where a dialog was not a dialog. Both now write role="dialog" and aria-modal="true" as part of the Modal chassis, next to the fixed-overlay rules — semantics the node type owns, not a value a caller passes.

So there is deliberately no RoleDialog. Adding one would put the burden back on the author for something three of the four targets already do unasked, and would cost two more native arms that could only be empty — which in this vocabulary means "this platform cannot say it", the opposite of the truth here. An author who overrides a Modal's role with a core.AccessibilityRole still wins, on the same principle the chassis follows for style: the framework's default goes first.

RoleTabPanel is \*not\* a third case of that shape, and for five sessions it was recorded as one. The argument that kept it out was that a tab panel is not really a role but one end of a \*relationship\* — the announcement a reader gives ("tab 2 of 3, Sermons, tab panel") comes from aria-controls and aria-labelledby pointing between two elements, and both are IDREFs, which Style does not carry. That was true when it was written and stopped being true the moment AccessibilityControls landed: the pointing half exists now, and the constant was the only piece still missing from a hand-built strip.

So the division is not "does the node type know it" but "can an author say it". core.TabView still owns its own wiring end to end — it mints the ids, writes this role, and keeps aria-selected in step — exactly as core.Modal owns its dialog role; the difference from RoleDialog is that a hand-built tab strip is a shape people actually build, and a hand-built modal is not. examples/social's bottom bar is one, and until this constant its regions were \`group\`s that three tabs claimed to control.

#### What used to block it, and what replaced the block

The WASM runtime has to tell a panel it wired itself from a role an author wrote, or it unwires and rewires the same element on alternate syncs. It did that by the value — "tabpanel" could only have come from wireTabPanel, because no core.Role spelled it — which made the \*absence of this constant\* load-bearing, and which is why the entry sat.

The discriminator is now a data-grmob-panel marker that both web targets write, in the channel data-grmob-chrome already uses for the same kind of fact: this element is something the framework put here. That is a better answer than the old one even setting the constant aside, because it says what it means — the old test asked "is this value one no author could have written", which is a fact about the vocabulary standing in for a fact about the element.

RoleTab and RoleTabList were never in the same position and were always present: they say what a control and a strip \*are\*, which is a claim about one element.

#### Every renderer names every role

Both natives dispatch on the string, arm by arm, so a role with no arm falls into a catch-all and is silently inert — the same failure ContentMode has, where a mode nobody taught the natives about draws as \`fit\` on device and as the browser default on the web with no error anywhere. So each native spells out the roles it does \*not\* implement alongside the ones it does, and mobile/verify/role\_test.go holds both dispatches against Roles(). Adding a constant below without adding it there fails \`go test ./...\`.

<small>[core/role.go:212](https://github.com/rohanthewiz/grmob/blob/master/core/role.go#L212)</small>

Tabular structure. The five together describe a table to a screen reader — separately they describe nothing, since a cell outside a row outside a table is not a thing ARIA recognizes. DataTable sets all five.

RoleRowGroup is the one that looks like padding and is not. A table's rows have to be \*owned\* by the table, and an unroled container between the two breaks the ownership: role="table" wrapping a plain div wrapping the rows reports a table with no rows. DataTable has exactly that shape — its body is a core.List, which is a div — so without a rowgroup on it the other four would describe an empty table, which is worse than describing nothing.

```go
const (
	RoleTable        Role = "table"
	RoleRowGroup     Role = "rowgroup"
	RoleRow          Role = "row"
	RoleColumnHeader Role = "columnheader"
	RoleCell         Role = "cell"
)
```

Collections. The looser cousin of the table pair, for a run of items that is a list rather than a grid — GroupedList's bands, a strip of cards.

RoleList makes the same claim about its children that RoleTable does, and loses it to chrome just as easily — the rule is structural, not tabular: a container holding items \*and\* a "Load more" footer, a section heading or a spinner is not a list, whatever it looks like. See "A structural role owns what is inside it" above — that is the section to read before putting one of these on a container that was not built to hold items alone.

```go
const (
	RoleList     Role = "list"
	RoleListItem Role = "listitem"
)
```

The selectable collection: a run of choices, and one choice in it. The fourth structural block, and the pair \`list\`/\`listitem\` above cannot stand in for.

##### Why a second collection pair rather than a state on the first

Because ARIA will not carry it. \`aria-selected\` is defined for gridcell, option, row, tab and columnheader — not for \`listitem\` — so a list item that says it is chosen says it into a void on both web targets: the attribute is written, the DOM inspector shows it, and no reader announces anything. A list is \*content\* and a listbox is a \*control\*, and the state only exists on the control side.

comps.ListRow is what asked. Its selected row spelled the state into its own accessible name (", selected") because both other doors were shut: \`listitem\` cannot carry the state, and \`button\` — which carries the neighbouring \`aria-pressed\` — would make the row a foreign child of any role="list" around it, costing the whole list its shape for one row's announcement. This pair is the door that was left.

##### What a listbox promises, and who keeps the promise

A listbox is a real control in ARIA's model, and the pattern that goes with it is larger than two attributes: the container takes keyboard focus, the arrow keys move an active option, and the reader is told which option is active through a roving tabindex or aria-activedescendant.

None of that is \*here\*, and it does not need to be. This type is a vocabulary — it says what a node is, and nothing in core stamps a tabindex or reads an arrow key (core/focus.go is about putting the cursor in a named field, which is a different question and one that costs a render pass per keystroke). What closed the gap instead was noticing that the vocabulary already says everything the pattern needs:

	what is a member of what   this pair, and the structural rule above that
	                           makes a listbox's contents its own
	which one is chosen        Style.AccessibilitySelected
	which way the arrows go    the container's own layout axis
	what activation means      the OnTap the author already wrote

So the WASM runtime supplies the whole keyboard half from what crosses the wire, with no new prop and no source change in any screen that had already said the above — see "Composite widgets are operable" in docs/platforms/wasm.md. htmlout deliberately writes none of it: a roving tabindex without the handler that moves it takes every option but one out of the tab order and reaches none of them, so the attribute is behaviour rather than semantics and a static export must not carry it.

On the two phones there was never a gap — VoiceOver and TalkBack navigate a collection by swipe, not by arrow key — which is also why neither native has a listbox in its semantics vocabulary at all: both spell a chosen item as the \`selected\` state this pair exists to make \*legal\*, and they honour that state on any node without being told what contains it.

##### The depth question, answered the other way

\`listitem\` carries aria-level and \`option\` does not, so a row cannot be both a choice and a depth: the two roles are exclusive and only one of them takes a level. ARIA does have a role for an item that is both — \`treeitem\` inside a \`tree\`, which supports aria-level and aria-selected together — and it is deliberately not here, because a tree is a third pattern with its own expansion state and its own keyboard contract, and nothing in this repository has one. See comps.ListRow.Selectable, which is where the two fields meet and where the precedence is written down.

```go
const (
	RoleListBox Role = "listbox"
	RoleOption  Role = "option"
)
```

The radio pair: a set of mutually exclusive choices that are all on screen, and one choice in it. The third collection pair, and the third structural block that makes a claim about its children — role="radiogroup" says the things inside it are radios. See "A structural role owns what is inside it" above.

##### Why a pair of its own when listbox and option already carry a choice

Because the two patterns promise different things, and a widget that wears the wrong one is announced as the wrong control. An option is \*selected\*; a radio is \*checked\*. A listbox is one tab stop whose arrows move a highlight and leave the choice alone unless the widget asks otherwise (AccessibilitySelectionFollowsFocus); a radio group is one tab stop whose arrows move the check itself, always. comps.RadioGroup shipped as a listbox for want of this pair and said "option, selected" for "radio button, checked" — true, and the weaker of the two words.

##### One state field, a third attribute

The state is Style.AccessibilitySelected, the field every other choice uses, and not a new AccessibilityChecked. The two web exporters write it as aria-checked for a radio, beside aria-selected for an option and a tab and aria-pressed for a button; the natives have one spelling of "this one is on" each and use it for all of them. A second field would let a radio carry both a selection and a check, which is a state no control has.

##### What each target does with it

	web       role="radiogroup" / role="radio" and aria-checked; the WASM
	          runtime adds the keyboard (one tab stop on the checked radio,
	          the arrows move and check), htmlout writes no tabindex, as for
	          every composite
	Compose   radiogroup is selectableGroup(), radio is Role.RadioButton, and
	          the state is the `selected` property grMobSelected sets
	SwiftUI   no trait for either; the state arrives as .isSelected, the same
	          loss the listbox pair has on this platform

```go
const (
	RoleRadioGroup Role = "radiogroup"
	RoleRadio      Role = "radio"
)
```

The tab pair: a strip of controls that switches what the screen is showing, and one control in it. The third structural block, and it makes the same claim about its children the other two do — role="tablist" says the things inside it are tabs, so a strip that also holds a "+" button or a count is not a tablist. See "A structural role owns what is inside it" above.

Both are for a \*hand-built\* strip. core.TabView needs neither: it writes these two, plus the tabpanel half and the aria-controls/aria-labelledby wiring between them, from the node type — see "Roles a node type carries for itself" above for why there is no RoleTabPanel to complete the set.

A tab is the vocabulary's second selectable control, after RoleButton, and the state it wants is Style.AccessibilitySelected. The pair is what makes the strip legible: a reader announces "tab, selected" only when the tab says so, and a tablist in which \*no\* tab says so announces every one of them as unselected. So a strip sets the state on every tab, not just the live one — SelectedOff is a value with a job here, not a way of saying nothing.

It is also what the arrow keys move between. The WASM runtime reads this pair the same way it reads the listbox one and supplies ARIA's keyboard half — one tab stop for the strip, Left/Right within it (Up/Down for a strip laid out as a column), Home and End to the ends — from the roles and the selection alone. A hand-built strip gets it by saying what it is; see the listbox pair above for the argument, and docs/platforms/wasm.md for what each target does.

```go
const (
	RoleTab     Role = "tab"
	RoleTabList Role = "tablist"
)
```

Landmarks: the regions of a screen a reader jumps between rather than reads through. AppBar is a banner, a tab strip is navigation, SearchField is a search, ChipStrip is a toolbar.

```go
const (
	RoleBanner     Role = "banner"
	RoleNavigation Role = "navigation"
	RoleSearch     Role = "search"
	RoleToolbar    Role = "toolbar"
)
```

Live regions: content that changes on its own and should be announced when it does, without the reader having to be looking at it.

The first two differ in how rudely they interrupt. Status waits for a pause — "saved", "3 new items". Alert cuts in — a failure, an expiry, anything the reader must hear before continuing. Banner picks between them by variant, which is the distinction its Variant already draws visually.

RoleLog is the third, and it is not a politeness level: log and status are both polite, and on Android they are the same call. What differs is the \*shape of the content\*, which is a promise about the element rather than about the interruption.

	status   one advisory that is replaced. "Saved", "3 new items". A reader
	         announces the region's new state, and the old text is gone.
	log      a record that is appended to and whose order is meaningful — a
	         chat transcript, a console, a running import. A reader announces
	         what arrived, and what came before it is still there to be read
	         back.

The distinction is the difference between "the region now says this" and "this was added at the end", which is why a transcript marked \`status\` announces correctly and reads back wrong: the whole conversation is one region that has, as far as the reader is concerned, just changed entirely.

It maps to Compose's polite live region, the same call \`status\` makes, because that is the whole of what Compose can say — an honest collapse rather than a second spelling of one fact. On the web the two are different roles with different reading behaviour, which is where the field earns its place.

```go
const (
	RoleStatus Role = "status"
	RoleAlert  Role = "alert"
	RoleLog    Role = "log"
)
```

Content roles: what a node is when it is not a region.

RoleButton is for a tappable container — a Box or a Row with an OnTap, which every renderer draws as inert scenery and every screen reader announces as text. A core.Button needs none of this; it is already a \<button> on the web and a real control on both natives.

RoleLink is the other half of that pair, and the distinction is not cosmetic: a button does something \*here\* and a link goes somewhere else. A reader deciding whether to follow a control needs to know which, and the framework has no node type that carries the difference — core.OpenURL is a callback like any other, so a row that dials a phone number and a row that files a form are the same tappable Box until one of them says otherwise. RoleHeading is the one value here with a second question attached: how deep the heading sits. That is Style.AccessibilityHeadingLevel, a separate int rather than a lettered set of constants — see its doc for why six more spellings of "heading" would cost both DOM renderers the mapping table this vocabulary exists to avoid.

RoleImg is for a node that is a \*picture\* — something whose meaning is carried by its arrangement rather than by any text inside it, and which therefore needs one text alternative standing in for the whole thing. comps.Compass is the case that asked for it: a rose read in tree order is "N W E S" whatever direction it is pointing, so the widget hides its parts and speaks once.

It was also, for a while, the only role that made such a label \*work at all\* on the web: ARIA forbids an accessible name on a generic element, so an AccessibilityLabel on a plain container was dropped by screen readers rather than announced. RoleGroup below is now the general answer to that, and the division between the two is what the node is rather than what it needs — an img stands in for its parts and should hide them, a group names them and leaves them readable. Reach for this one only when the picture reading is true.

A node with this role should hide its children, or the reader gets the alternative \*and\* the parts it was standing in for.

```go
const (
	RoleHeading Role = "heading"
	RoleButton  Role = "button"
	RoleLink    Role = "link"
	RoleImg     Role = "img"
)
```

The field that owns a popup list: a text input whose typing filters the options under it, with the keyboard moving through them while focus stays in the field.

##### Why a role of its own when the listbox pair already carries a choice

Because the listbox pattern answers the wrong question for a field. A listbox is its own tab stop: Tab moves focus \*into\* it and the arrows move among its options. comps.SearchableSelect shipped that way, and it cost the field its focus — the list sits under a field the user is typing in, so reaching an option meant leaving the caret, and a keyboard pick removed the focused option with the list and dropped focus onto the page.

ARIA's combobox pattern is the shape that keeps both: the field keeps DOM focus the whole time, and aria-activedescendant names which option the arrows have reached. A reader announces that option as if it were focused, the caret stays where it was, and typing carries on. The listbox is still there and still RoleListBox — it is the \*popup\*, named by the field's aria-controls, and the runtime takes it out of the tab order (see "The combobox pattern" in wasm/grmob-runtime.js).

##### What a combobox states, and where each part lives

	aria-expanded           Style.AccessibilityExpanded — whether the list is
	                        showing. ARIA *requires* it on this role, so a
	                        combobox with the field unset is incomplete.
	aria-controls           Style.AccessibilityControls — the listbox's
	                        AccessibilityID. Also required while the popup
	                        shows.
	aria-activedescendant   written by the WASM runtime alone, per keystroke.
	                        It is behaviour rather than a fact about the tree
	                        (which option the arrows reached changes with no
	                        render in between), the same argument that keeps
	                        a roving tabindex out of core and out of htmlout.
	aria-haspopup           implicit: a combobox's popup is a listbox unless
	                        it says otherwise, so nothing is written.

##### It goes on the field, not the container

ARIA 1.2 moved the role onto the input itself (1.1 put it on a wrapper that owned both the input and the list), and the reason is the one above: the element with focus has to be the one carrying the state. So the node that takes this role is a core.Input, exported as \<input role="combobox">, which the HTML-ARIA mapping allows for a text input.

##### What each target does with it

	web       role="combobox"; the runtime adds the keyboard (arrows move the
	          active option, Enter picks, focus never leaves the field)
	Compose   nothing. Role.DropdownList is the near miss and is turned down:
	          TalkBack would announce the text field as a drop-down list,
	          which is a control you open rather than one you type into
	SwiftUI   nothing; the field is a TextField, which VoiceOver already
	          announces as editable, and no trait names a popup

On both phones the list is reached by swiping, as every collection is, so the pattern's keyboard half has nothing to replace there.

```go
const RoleComboBox Role = "combobox"
```

The naming role: the least a container can be, and the only thing that makes an accessible name on one legal at all.

##### The silence it closes

Every layout node in this framework exports as a \<div> or a \<span>, and both tags carry the implicit ARIA role \`generic\`. ARIA prohibits an accessible name on \`generic\` — aria-label and aria-labelledby are listed under "roles which cannot be named" — and browsers enforce it by pruning the name from the accessibility tree. So:

	core.Box(core.AccessibilityLabel("Unread messages"), …)

wrote a correct-looking attribute that no screen reader on either web target announced, while VoiceOver and TalkBack read it out perfectly, because a SwiftUI accessibilityLabel and a Compose contentDescription are honoured on any node without asking what it is. Two targets silent, two fine — which is what let it ship: the two that work are the two a developer is most likely to be testing on.

RoleImg was the first door out and it is the wrong shape for most rows. It says the node is a \*picture\* whose parts should be hidden behind one alternative, which is true of comps.Compass and false of a list row, a disclosure header or a stat tile — all of which want their contents read as well as their name.

##### Why \`group\` and not one of the louder candidates

	region     also nameable, and a landmark. A reader adds every region to
	           the list it jumps between, so naming six rows would put six
	           entries in a screen's table of contents.
	button     claims a control, makes its children presentational (a heading
	           inside one stops being a heading), and is a foreign child of any
	           list around it — see comps.ListRow, which turned it down
	           for exactly that.
	group      "a set of user interface objects", nameable, not a landmark,
	           and with no required children and no presentational-children
	           rule. It says these things belong together and this is what they
	           are called, and nothing else.

That "nothing else" is the whole recommendation. A role is a claim, and the structural rule above says a claim a container cannot keep is worse than no role at all; \`group\` is the one value in this vocabulary that promises nothing about what it holds, so it can be given to a container nobody has looked inside.

##### It is also a fallback, not only a constant

Because the silence is a framework bug rather than an author's mistake, the two web exporters supply this role themselves: a node with an accessible name, no role of its own, and a generic tag is written role="group" so the name is heard. An author who says anything more specific wins — the fallback only ever fills an empty slot. See accessibilityAttrs in htmlout/export.go and applyAccessibility in wasm/grmob-runtime.js, which restate one rule.

The fallback cannot make anything worse, which is the argument for doing it silently. Before it, the name was invalid ARIA that was dropped; after it, the name is valid ARIA that is announced. The one thing it could disturb is a structural container's claim about its children — but a generic div inside a role="list" was never a \`listitem\` either, so a \`group\` there is the same foreign child it already was, one attribute louder.

##### Both natives leave it empty for the opposite of the usual reason

Nine of the roles here are inert on SwiftUI and Compose because those platforms have no way to say the thing. This one is inert because they have no need to: both already announce a label on any node, so the role that makes the label legal buys them nothing. Setting it therefore costs nothing anywhere and closes a two-target silence.

```go
const RoleGroup Role = "group"
```

The zero value. A node that never sets a role has none, which is what every node had before this type existed: the renderers emit no attribute, add no trait and set no semantics.

```go
const RoleNone Role = ""
```

The one valued role: a control that is somewhere between two ends.

It is the only value in this vocabulary that reads Style.AccessibilityValue, and that pairing is ARIA's own scoping rather than a shortlist. aria-valuenow and its two bounds are defined for meter, progressbar, scrollbar, slider, spinbutton and a focusable separator; of those, this is the only one core has a role for, and the absences are all the same absence — no widget here is a meter, a scrollbar or a spinbutton, and core.Slider is a node type that exports as \<input type="range">, which carries the whole range natively and would have a second, contradicting claim written onto it by an ARIA one.

##### What it closes

comps.ProgressBar had no way to say it was a progress bar or how far along it was, so it said both into its accessible \*name\*: "Upload, 45 percent". That is the move Chip's ", selected" suffix was deleted for — a name is meant to be stable, so a bar ticking from 44 to 45 re-announced the whole thing, and nothing could act on a number buried in a string. With the role and the range, a reader announces the name once and the value as it moves.

##### A determinate bar and an indeterminate one are the same role

ARIA spells the difference by \*omitting\* aria-valuenow: a progressbar with a range is a bar with a known position, and one without is a spinner that is running. So an indeterminate bar is this role and a zero ValueRange, which falls out of the vocabulary rather than needing a value of its own.

##### Both natives

Compose has progressBarRangeInfo, which takes the numbers and announces a percentage TalkBack localizes — one of the better-mapped values here, and the reason grMobValue exists beside grMobRole. SwiftUI has no numeric equivalent at all; it takes the ValueRange's Text through accessibilityValue and nothing else, which is the honest half. See core.ValueRange.

```go
const RoleProgressBar Role = "progressbar"
```

The region a tab shows: the third member of the tab family, and the one that only makes sense with a reference beside it.

A tabpanel on its own says almost nothing — it is a section of a page — and what makes it announce as "tab panel, Sermons" is being pointed at. So this is the one role in the vocabulary that is not much use without Style.AccessibilityID and a tab's Style.AccessibilityControls, and the two arrived in the opposite order: the pointing existed for a session before the thing it points at could say what it was.

	// the strip
	core.Row(core.AccessibilityRole(core.RoleTabList),
	    Chip{Label: "Home", Style: []core.StyleProp{
	        core.AccessibilityRole(core.RoleTab),
	        core.AccessibilitySelected(core.SelectedWhen(tab == "home")),
	        core.AccessibilityControls("app-panel"),
	    }},
	)
	// the region it switches
	core.Box(core.AccessibilityRole(core.RoleTabPanel),
	    core.AccessibilityID("app-panel"), core.AccessibilityLabel("Home"), …)

It makes no claim about its children, unlike the tablist half — a panel holds whatever a screen holds — so it is not subject to the structural rule above and can go on any container.

core.TabView writes it from the node type and needs no author to. See "Roles a node type carries for itself" for why that is not a reason to leave the constant out, and for the marker that lets the runtime tell its own writes from an author's.

```go
const RoleTabPanel Role = "tabpanel"
```

#### func CompositeMemberRole

```go
func CompositeMemberRole(container Role) (member Role, composite bool)
```

CompositeMemberRole returns the role ARIA gives the members of a composite container, or "" for a composite whose members ARIA does not name.

	listbox      option
	radiogroup   radio
	tablist      tab
	toolbar      ""      ARIA defines no `toolbaritem`

It is the second half of what KeyboardComposites states — that list says which containers have a keyboard, this says how each one recognises the things the arrows move between — and it is here for the same reason the list is: a caller (and AuditTree) asks core what a role means, and the alternative is the fact living only in the WASM runtime's COMPOSITE\_MEMBERS, where no Go reader can consult it.

##### Two different empty answers, and why the second return exists

The empty answer is a real answer and not a "not found". A toolbar has a keyboard; what it does not have is a role that says "this is one of my members", which is exactly why the runtime has to supply a membership rule of its own (TappableContainerRoles is the Go half of that rule).

A RoleHeading has neither — no keyboard, and so no members to name — and it used to get the same "" back. The doc said callers separated the two by asking KeyboardComposites first, which is a contract a doc comment cannot enforce and which the one caller inside core got right by accident of never being handed a non-composite. CompositeWalkStopsAt reads this answer and returns "the walk stops here" for an empty one; handed a RoleHeading it produced a confident statement about a walk that does not exist.

So the two cases are separated where they are made:

	CompositeMemberRole(RoleListBox)  -> RoleOption, true
	CompositeMemberRole(RoleToolbar)  -> "",         true   a keyboard, no
	                                                        member role
	CompositeMemberRole(RoleHeading)  -> "",         false  no keyboard at all

\`composite\` is exactly membership of KeyboardComposites, and role\_control\_test.go holds the two to each other — a container added to that list and not here would report false for a role that has a keyboard, which is the same class of quiet wrong answer one table over.

<small>[core/role.go:813](https://github.com/rohanthewiz/grmob/blob/master/core/role.go#L813)</small>

#### func KeyboardComposites

```go
func KeyboardComposites() []Role
```

KeyboardComposites returns the container roles that get an ARIA keyboard pattern — a roving tabindex, arrow movement, Home and End — from the WASM runtime.

##### Why core states a fact about one target's runtime

It is not a list of what the runtime happens to implement; it is the list of roles for which \*setting a composite-keyboard style prop means anything at all\*, and that is a question a caller asks of core. AuditTree is the first reader: core.AccessibilitySelectionFollowsFocus is a statement about what a widget's keyboard does, and on a role with no keyboard it is a claim about nothing — which no exporter can notice, because knowing these four roles where an attribute is written would put the list in two places.

The runtime keeps the same four in two tables split by a different question (whether ARIA names the members), and wasm/verify holds their union to this function. So a fourth pattern is one edit here and a failing check there, rather than a role that quietly gains a keyboard the audit still calls inert.

##### Why these four and not the rest of ARIA's patterns

\`listbox\`, \`radiogroup\` and \`tablist\` are the three ARIA structures that both name their members and own their children, so the runtime can find a container's members by role. A radio group differs from a listbox in one behaviour rather than in its walk: its arrows move the check itself, so the runtime follows focus inside one without being asked, and AccessibilitySelectionFollowsFocus on it states what it already does. \`toolbar\` names no member role — ARIA defines no \`toolbaritem\` — and is here anyway because the pattern is real and the runtime supplies the membership rule itself: a toolbar's controls are the natively focusable tags plus the containers that say they are controls.

\`menu\`, \`menubar\`, \`tree\`, \`treegrid\` and \`grid\` are the patterns ARIA describes that this framework refuses, each for a stated reason — aria/verify/refusals\_test.go holds every refusal to what the pattern actually requires. \`list\` is deliberately absent and is the near miss worth naming: it is content rather than a control, and ARIA gives it no keyboard at all.

\`combobox\` is absent for the opposite reason: it has a keyboard, and the keyboard is not a composite's. Its focus never moves — the field keeps it and aria-activedescendant names the option the arrows reached — so there is no tab stop to rove and nothing for the member walk or the audit's nested-composite rule to say. The listbox it controls is in this list, and the runtime stands it down while it is a combobox's popup; see RoleComboBox.

Container order matches Roles(); the members are not here, because being a member is a fact about a role's parent rather than about the role.

<small>[core/role.go:768](https://github.com/rohanthewiz/grmob/blob/master/core/role.go#L768)</small>

#### func Roles

```go
func Roles() []Role
```

Roles returns every declared Role except RoleNone, in declaration order.

RoleNone is excluded because it is the absence of a role rather than one of them: it is the field's zero value, no renderer has an arm for it, and a coverage check that demanded one would be asking each renderer to implement "unset". Everything downstream that iterates roles — the native dispatch pins, the DOM export test — wants the twenty-seven that do something.

A fresh slice per call rather than a package-level var, which any importer could write to. Twenty-seven elements are cheaper to build than to defend.

Pinned to the const blocks above by role\_enum\_test.go, which reads this file's syntax tree: adding a constant without adding it here should fail \`go test ./...\` rather than silently shrink the set every renderer's coverage check rests on.

<small>[core/role.go:704](https://github.com/rohanthewiz/grmob/blob/master/core/role.go#L704)</small>

#### func TappableContainerRoles

```go
func TappableContainerRoles() []Role
```

TappableContainerRoles returns the roles whose whole purpose is to make an ordinary container announce itself as a control.

These are the two the "Content roles" block above argues for in as many words: a Box or a Row with an OnTap, which every renderer draws as inert scenery and every screen reader announces as text until one of these says otherwise. A core.Button needs neither — it is already a \<button> on the web and a real control on both natives.

##### Why it is a list and who reads it

The WASM runtime's toolbar keyboard needs it. A toolbar's members are named by no role (ARIA defines no \`toolbaritem\`), so the runtime has to be told what a control is, and its answer is two rules: a natively focusable tag, or a container carrying one of \*these\* roles together with an OnTap. That second rule was a pair of bare strings in the runtime pinned against a pair of constants hand-written in a test — three copies of one fact, none of which was the fact itself.

The fact is here now, and role\_control\_test.go is what makes it a property rather than a fourth copy: every role core declares is either in this list or in a table saying why it is not one, so a new role cannot be added without somebody deciding. That was the actual hole — a future RoleCheckbox would be a tappable container by exactly the argument above, and would silently not be a toolbar member.

##### Why not "every role a screen reader calls a widget"

Because the question is narrower than it looks: not "is this thing interactive" but "does putting this role on a plain container make it a control the browser should give a tab stop to". RoleOption and RoleTab are interactive and are \*not\* here — they are members of a composite, whose tab stop belongs to their container and not to them, and taking one as a toolbar's control would put a second keyboard on a widget that has one.

<small>[core/role.go:1011](https://github.com/rohanthewiz/grmob/blob/master/core/role.go#L1011)</small>

### type SelectedState

```go
type SelectedState string
```

SelectedState is whether a control is \*on\* — the tab that is showing, the filter chip that is applied, the calendar day that is chosen.

It is the state half of core.Role. A role says what a control is and the label says what it is called; neither can say that this one of five chips is the one in effect, and until this type existed nothing in the framework could. What a widget did instead was write the state into the name — comps.Chip appended ", selected" to its AccessibilityLabel — which announces once, in the wrong place (a name is meant to be stable, and a reader that re-announces the control after a tap says the whole altered name rather than the changed state), and which no platform can act on.

#### Why three values and not a bool

Because "off" and "not a thing that can be on" are different facts and a bool has one spelling for both.

	SelectedUnset   a Box, a heading, a row of text. The node makes no claim,
	                which is the state every node in every tree was in before
	                this field existed, so the zero value is a no-op.
	SelectedOn      the chip that is applied, the tab that is showing.
	SelectedOff     a control that *could* be on and is not.

The third is the one a bool loses, and losing it is not cosmetic. A tablist in which only the live tab carries a state is malformed: a reader counting "tab 2 of 5" needs all five to say something, and the four that stay quiet are announced as plain tabs while the fifth is announced as selected — a strip that reads as though it has one tab and four pieces of furniture. So a strip sets the state on every control it draws, and SelectedOff is a value with a job rather than a way of saying nothing.

#### Why the values are ARIA's own spellings

For the reason core.Role's are: the two DOM targets write the value into the attribute verbatim and need no mapping table, and the two natives map what they can — which is the same trade the vocabulary made and the same place it pays off. \`mixed\`, ARIA's third value for aria-pressed and aria-checked, is deliberately absent: it describes a control governing a partially-selected set, no widget here has one, and a value nobody can produce is a fourth arm on every renderer for nothing.

<small>[core/selected.go:43](https://github.com/rohanthewiz/grmob/blob/master/core/selected.go#L43)</small>

```go
const (
	// SelectedUnset is the zero value: this node says nothing about being on
	// or off. Every renderer writes no attribute, adds no trait and sets no
	// semantics property, which is what every node did before this type.
	SelectedUnset SelectedState = ""

	// SelectedOn — the control is on.
	SelectedOn SelectedState = "true"

	// SelectedOff — the control can be on and is not. See the type doc for
	// why this is not the same as SelectedUnset.
	SelectedOff SelectedState = "false"
)
```

#### func SelectedStates

```go
func SelectedStates() []SelectedState
```

SelectedStates returns both stated values, in declaration order.

SelectedUnset is excluded for the reason RoleNone is excluded from Roles(): it is the field's zero value rather than one of the states, no renderer has an arm for it, and a coverage check that demanded one would be asking each renderer to implement "unstated".

Pinned to the const block above by selected\_enum\_test.go, and consumed by the exporters' round-trip checks the way Roles() is.

<small>[core/selected.go:85](https://github.com/rohanthewiz/grmob/blob/master/core/selected.go#L85)</small>

#### func SelectedWhen

```go
func SelectedWhen(on bool) SelectedState
```

SelectedWhen turns a widget's plain bool into the stated pair.

Every widget that has one of these holds a \`Selected bool\` — the caller owns the selection and the widget renders it — so the conversion would otherwise be three lines at each of them, and the tempting two-line version (\`if on { … SelectedOn }\` and nothing else) is exactly the bug the type doc warns about: it leaves the unselected controls silent.

It is a function rather than a method so that the bool reads as the subject: core.AccessibilitySelected(core.SelectedWhen(c.Selected)).

<small>[core/selected.go:69](https://github.com/rohanthewiz/grmob/blob/master/core/selected.go#L69)</small>

### type ValueRange

```go
type ValueRange struct {
	// Now is the current position, as ARIA's aria-valuenow. Empty means
	// unstated, which for a progressbar is ARIA's own spelling of
	// "indeterminate" — a bar that is running with no idea how far.
	Now string `json:",omitzero"`

	// Min and Max are the ends of the range, aria-valuemin and aria-valuemax.
	// Empty on both means ARIA's defaults, which are 0 and 100 — so a bare
	// Now reads as a percentage, which is what a progress fraction wants and
	// is why ValueOf's two-argument sibling would have been a trap: "3" with
	// no range announces as 3%, not as step 3.
	Min string `json:",omitzero"`
	Max string `json:",omitzero"`

	// Text replaces the number in the announcement when the digits are not
	// what a listener wants to hear — "3 of 5", "medium", "£12.50". ARIA says
	// a reader announces this *instead of* Now, so a Text that disagrees with
	// the number is the version the user gets.
	Text string `json:",omitzero"`
}
```

ValueRange is where a valued control sits inside its range — the fraction an upload has finished, the step a wizard is on.

It is the fourth of core's accessibility state vocabularies, after SelectedState (is this control on), ExpandedState (is this disclosure open) and the two level ints (how deep does this sit). Those three answer yes/no or a single integer; this one answers "how far along, out of what", which is three numbers and cannot be one field.

#### What asked for it

comps.ProgressBar, which had no way to say any of it. Its accessible name was built as "Upload, 45 percent" — the value spelled into the \*name\* channel, which is the exact move comps.Chip's ", selected" suffix was deleted for. A name is meant to be stable: a reader that re-announces a control says the whole altered name rather than the changed part, so a bar ticking from 44 to 45 re-announced "Upload, 45 percent" instead of "45 percent", and no platform could act on the number because no platform could find it.

#### Why the numbers are strings

For the reason Role's values and SelectedState's are ARIA's own spellings: the two DOM targets write them into the attribute verbatim and need no mapping table. The Style struct already carries numbers this way wherever zero is a legal value — Width, Height and MaxWidth are all strings — and that is the deciding reason here rather than a stylistic one:

	a float64 Now of 0 is a bar at the start of an upload, and it is also the
	zero value of the field. Style merges on "non-zero wins", so a stated 0
	would be indistinguishable from an unstated one and would be dropped by
	every merge in the chain.

SelectedState solved the same problem the same way — SelectedOff is the stated string "false" and SelectedUnset is "" — and ValueOf is the constructor that keeps a caller from having to think about it.

#### Why it is one field and not four

Style's two level ints merge independently \*on purpose\*: they answer disjoint questions, and dropping one because the other was set would make the result depend on which Style in the chain happened to name the role. This is the opposite case. Now, Min and Max are one fact in three parts — "45" means 45% under ARIA's implicit 0..100 and means nothing at all without knowing whether the range is 0..100 or 1..5 — so two Styles each merging half a range would produce a claim neither of them made. Merging as a unit is what makes that unwritable: a stated range replaces a stated range whole.

#### Text is the value channel, and it is not only for a range

Text becomes aria-valuetext on the web, Compose's stateDescription and SwiftUI's accessibilityValue. Those last two are honoured on any node, which makes this the value channel GrMobStyle.swift's AccessibilityExpanded note says the framework does not have. The difference that makes it safe now is whose words they are: a renderer emitting the literal "expanded" would be inventing English for every app in every locale, where this string is the app's own — the same line AccessibilityLabel and AccessibilityHint sit on.

The web is stricter than the natives here, exactly as it is for a selection: ARIA scopes aria-valuetext to the range roles, so a Text on an unroled container is dropped by a browser and announced by both phones. Each platform says the truest thing it can.

#### Why every field is \`omitzero\`

The same rule Style and EdgeInsets carry, and here it is the rule rather than the bytes: an empty field on this struct already means "unstated" by construction — that is the whole reason the three numbers are strings, per the argument above — and all four readers turn a missing key back into the empty string (optString, \`as? String ?? ""\`, \`v.Now || ""\`, and htmlout, which reads the Go value and never sees JSON). So presence carries nothing and the tags cost nothing.

The saving on the tutorial's contents screen is ten bytes, because one node on it states a range. That is not the point. The point is that "every struct that crosses this bridge omits its zeros" is now true without exception, which is what TestEveryWireFieldOmitsZero pins — an untagged field added to a wire struct is the failure mode the Style tags were worth 370KB catching late.

<small>[core/value.go:87](https://github.com/rohanthewiz/grmob/blob/master/core/value.go#L87)</small>

#### func ValueOf

```go
func ValueOf(now, min, max float64) ValueRange
```

ValueOf states a range from the three numbers a caller is holding.

The formatting is 'f' with the shortest round-tripping precision rather than %g, and that is not cosmetic: %g switches to scientific notation past six digits, so a byte counter would emit aria-valuenow="1.048576e+06" — which is not a number ARIA accepts and which no reader announces. -1 precision is what keeps a whole value short ("45", not "45.000000").

<small>[core/value.go:115](https://github.com/rohanthewiz/grmob/blob/master/core/value.go#L115)</small>

#### func (ValueRange) Progress

```go
func (v ValueRange) Progress() Progress
```

Progress resolves the three numbers into the one claim they make.

The parse is deliberately strict about what counts as a number: an empty string is unstated, and so is anything that does not parse — including the non-finite spellings ("NaN", "Inf") that Go's and Kotlin's parsers both accept and that no range property on any platform can hold. A bar whose position failed to parse is a bar with no position, which is a state ARIA already has a meaning for.

<small>[core/value.go:205](https://github.com/rohanthewiz/grmob/blob/master/core/value.go#L205)</small>

#### func (ValueRange) Stated

```go
func (v ValueRange) Stated() bool
```

Stated reports whether this range says anything at all. The zero value says nothing, which is what every node in every tree carried before this type existed, and is the condition Style.Merge and both web exporters test.

<small>[core/value.go:298](https://github.com/rohanthewiz/grmob/blob/master/core/value.go#L298)</small>

#### func (ValueRange) Unparsed

```go
func (v ValueRange) Unparsed() []string
```

Unparsed names the stated numeric fields that are not numbers this vocabulary can use, in the order the type declares them.

##### Why the reading alone is not enough to report one

Progress answers what the three strings amount to, and it answers it in ARIA's own terms: a position that does not parse is a bar with no position, which is a state ARIA already has a meaning for, and a bound that does not parse falls back to ARIA's default. Both are the right resolution and neither is what the author wrote — and from the outside the two cases are indistinguishable from the ones where nothing was stated at all.

	ValueRange{Now: "half", Min: "0", Max: "100"}   reads as indeterminate
	ValueRange{Min: "0", Max: "100"}                reads as indeterminate

The first is a mistake and the second is a bar that is genuinely running with no idea how far. So the fields are named separately from the reading, and core.AuditTree is what reports the first without reporting the second.

##### And it is not a harmless mistake

The three targets that parse these strings do not agree about a string that is not a number. Compose and this package read it as absent; a browser reads aria-valuenow="half" as 0 and pins the bar at the start of its range, and reads aria-valuemax="lots" as 0 and then clamps the position down to it. So an unparseable number is not "no number" — it is a different wrong answer on each platform, with nothing anywhere saying so. wasm/verify's browser pass holds Chrome to that divergence case by case.

Text has no part in it: it is words by design, and any string is a legal one.

<small>[core/value.go:266](https://github.com/rohanthewiz/grmob/blob/master/core/value.go#L266)</small>

#### func (ValueRange) WithText

```go
func (v ValueRange) WithText(text string) ValueRange
```

WithText adds the spoken form to a range, for a control whose number is not what a listener wants to hear.

	core.ValueOf(3, 1, 5).WithText("step 3 of 5")

A method rather than a fourth argument to ValueOf, because most ranges do not want one: a percentage announces perfectly well as a percentage, in whatever language the reader is set to, and supplying an English string would take that localization away.

<small>[core/value.go:132](https://github.com/rohanthewiz/grmob/blob/master/core/value.go#L132)</small>

