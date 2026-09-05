package core

// Role is what a node *is* to assistive technology, as distinct from what it
// is called (AccessibilityLabel) or what tapping it does (AccessibilityHint).
//
// A screen reader announces "Sermons, heading" or "March, column header"
// because something told it the element's kind. Nothing in this framework
// could say that until now: every container is a Box or a Row, every one of
// them exports as a <div>, and a screen built entirely out of them is a flat
// run of text to VoiceOver and TalkBack no matter how carefully it is
// labelled. Three widgets hit the wall independently — DataTable wanting to
// be a table, the screen-furniture bundle wanting a banner, a heading and a
// search landmark, and Calendar's forty-two tappable day cells wanting to be
// buttons — which is what turned "an ARIA role prop, some day" into this file.
//
// # Why the values are spelled in ARIA
//
// The set has to be *some* vocabulary, and the four renderers do not share
// one. ARIA is the only candidate that is a published standard with a name
// for every case here; SwiftUI's AccessibilityTraits and Compose's
// SemanticsProperties are small, partly overlapping sets that would each need
// a mapping table whichever vocabulary core picked. Choosing ARIA means the
// two DOM targets need no table at all — the value is the attribute — and the
// two natives map what they can, which is the same work they already do for
// ContentMode and the alignments.
//
// # What each target does with a role
//
//	role          | DOM (both)      | SwiftUI trait  | Compose semantics
//	--------------+-----------------+----------------+------------------------
//	heading       | role="heading"  | .isHeader      | heading()
//	columnheader  | role=…          | .isHeader      | heading()
//	button        | role="button"   | .isButton      | role = Role.Button
//	link          | role="link"     | .isLink        | —
//	search        | role="search"   | .isSearchField | —
//	status        | role="status"   | —              | liveRegion = Polite
//	alert         | role="alert"    | —              | liveRegion = Assertive
//	the other nine| role=…          | —              | —
//
// The other nine are table, rowgroup, row, cell, list, listitem, banner,
// navigation and toolbar — the tabular set, the collection pair, and the
// landmarks.
//
// Nine of the sixteen do nothing on either native, and that is the
// honest state of those platforms rather than a gap to be filled later:
// neither has a tabular semantics vocabulary a role can be mapped onto (Compose
// has collectionInfo, which describes counts and indices this prop does not
// carry), and neither has landmarks at all — VoiceOver's rotor navigates by
// heading, not by banner.
//
// A role that maps to nothing is still worth setting. The web is a first-class
// target here, the mapping can improve later without the call sites changing,
// and a role that is right on one platform and inert on two is strictly better
// than a div.
//
// # A structural role owns what is inside it
//
// The tabular five and the collection pair are not labels on a container —
// they are claims about what the container holds. role="list" says its
// children are listitems; role="table" says its children are rows, or
// rowgroups holding rows. A reader acts on the claim rather than re-deriving
// it: it announces the count ("list, five items"), it offers item-by-item
// navigation, and it reads the structure instead of the text.
//
// A container can fail that claim in two directions, and both produce
// something worse than no role at all.
//
// *A gap in the chain.* An unroled element between the container and its items
// breaks the ownership: role="table" wrapping a plain div wrapping the rows
// reports a table with no rows. This is what RoleRowGroup exists to close, and
// its comment below has the detail.
//
// *A foreign child.* A container that holds something which is not an item — a
// footer, a heading, a spinner — is claiming a structure it does not have.
// ARIA specifies the children a role requires and not what to do with any
// others, so what a reader makes of the odd child is an implementation's
// choice rather than a promise: it may be counted, skipped, or announced as
// the item it is not.
//
// So: **a container that mixes items with chrome cannot take a structural
// role.** Either the chrome moves outside the container, or the container
// stays roleless — in which case its contents are announced as the text they
// are, which is what everything was before this type existed and is not a
// regression. Reaching for the role anyway is the one move that makes the
// screen worse.
//
// The rule is easy to meet by accident, because the shapes that hit it are the
// ordinary ones. DataTable meets it three times in one widget: it puts a
// rowgroup on its body list to close a gap, *withholds* that rowgroup when the
// list holds a placeholder instead of rows, and documents a grouped table's
// band headings — foreign children it cannot move — as a limit it cannot close
// from where it sits. A paged list in the first app to adopt these roles met it
// a fourth time without having read any of that: its "Load more" footer sits
// inside the core.List, so the list cannot be a list.
//
// The landmarks, the live regions and the content roles carry no such promise
// and are not subject to this. A banner or a navigation region owns whatever
// it likes; RoleHeading, RoleButton and RoleLink describe the node itself.
// Only the two structural const blocks below — the tabular set and the
// collection pair — make a claim about their children, which is where to look
// rather than here if a role is ever added to either.
//
// # Every renderer names every role
//
// Both natives dispatch on the string, arm by arm, so a role with no arm falls
// into a catch-all and is silently inert — the same failure ContentMode has,
// where a mode nobody taught the natives about draws as `fit` on device and as
// the browser default on the web with no error anywhere. So each native spells
// out the roles it does *not* implement alongside the ones it does, and
// mobile/verify/role_test.go holds both dispatches against Roles(). Adding a
// constant below without adding it there fails `go test ./...`.
type Role string

// The zero value. A node that never sets a role has none, which is what every
// node had before this type existed: the renderers emit no attribute, add no
// trait and set no semantics.
const RoleNone Role = ""

// Tabular structure. The five together describe a table to a screen reader —
// separately they describe nothing, since a cell outside a row outside a table
// is not a thing ARIA recognizes. DataTable sets all five.
//
// RoleRowGroup is the one that looks like padding and is not. A table's rows
// have to be *owned* by the table, and an unroled container between the two
// breaks the ownership: role="table" wrapping a plain div wrapping the rows
// reports a table with no rows. DataTable has exactly that shape — its body is
// a core.List, which is a div — so without a rowgroup on it the other four
// would describe an empty table, which is worse than describing nothing.
const (
	RoleTable        Role = "table"
	RoleRowGroup     Role = "rowgroup"
	RoleRow          Role = "row"
	RoleColumnHeader Role = "columnheader"
	RoleCell         Role = "cell"
)

// Collections. The looser cousin of the table pair, for a run of items that is
// a list rather than a grid — GroupedList's bands, a strip of cards.
//
// RoleList makes the same claim about its children that RoleTable does, and
// loses it to chrome just as easily — the rule is structural, not tabular: a container holding items *and* a "Load
// more" footer, a section heading or a spinner is not a list, whatever it
// looks like. See "A structural role owns what is inside it" above — that is
// the section to read before putting one of these on a container that was not
// built to hold items alone.
const (
	RoleList     Role = "list"
	RoleListItem Role = "listitem"
)

// Landmarks: the regions of a screen a reader jumps between rather than reads
// through. AppBar is a banner, a tab strip is navigation, SearchField is a
// search, ChipStrip is a toolbar.
const (
	RoleBanner     Role = "banner"
	RoleNavigation Role = "navigation"
	RoleSearch     Role = "search"
	RoleToolbar    Role = "toolbar"
)

// Live regions: content that changes on its own and should be announced when
// it does, without the reader having to be looking at it.
//
// The two differ in how rudely they interrupt. Status waits for a pause —
// "saved", "3 new items". Alert cuts in — a failure, an expiry, anything the
// reader must hear before continuing. Banner picks between them by variant,
// which is the distinction its Variant already draws visually.
const (
	RoleStatus Role = "status"
	RoleAlert  Role = "alert"
)

// Content roles: what a node is when it is not a region.
//
// RoleButton is for a tappable container — a Box or a Row with an OnTap, which
// every renderer draws as inert scenery and every screen reader announces as
// text. A core.Button needs none of this; it is already a <button> on the web
// and a real control on both natives.
//
// RoleLink is the other half of that pair, and the distinction is not
// cosmetic: a button does something *here* and a link goes somewhere else. A
// reader deciding whether to follow a control needs to know which, and the
// framework has no node type that carries the difference — core.OpenURL is a
// callback like any other, so a row that dials a phone number and a row that
// files a form are the same tappable Box until one of them says otherwise.
const (
	RoleHeading Role = "heading"
	RoleButton  Role = "button"
	RoleLink    Role = "link"
)

// Roles returns every declared Role except RoleNone, in declaration order.
//
// RoleNone is excluded because it is the absence of a role rather than one of
// them: it is the field's zero value, no renderer has an arm for it, and a
// coverage check that demanded one would be asking each renderer to implement
// "unset". Everything downstream that iterates roles — the native dispatch
// pins, the DOM export test — wants the sixteen that do something.
//
// A fresh slice per call rather than a package-level var, which any importer
// could write to. Sixteen elements are cheaper to build than to defend.
//
// Pinned to the const blocks above by role_enum_test.go, which reads this
// file's syntax tree: adding a constant without adding it here should fail
// `go test ./...` rather than silently shrink the set every renderer's
// coverage check rests on.
func Roles() []Role {
	return []Role{
		RoleTable, RoleRowGroup, RoleRow, RoleColumnHeader, RoleCell,
		RoleList, RoleListItem,
		RoleBanner, RoleNavigation, RoleSearch, RoleToolbar,
		RoleStatus, RoleAlert,
		RoleHeading, RoleButton, RoleLink,
	}
}
