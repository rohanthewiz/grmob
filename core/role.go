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
//	img           | role="img"      | .isImage       | role = Role.Image
//	tab           | role="tab"      | —              | role = Role.Tab
//	tablist       | role="tablist"  | .isTabBar      | —
//	status        | role="status"   | —              | liveRegion = Polite
//	alert         | role="alert"    | —              | liveRegion = Assertive
//	log           | role="log"      | —              | liveRegion = Polite
//	the other 12  | role=…          | —              | —
//
// The other twelve are table, rowgroup, row, cell, list, listitem, listbox,
// option, banner, navigation, toolbar and group — the tabular set, both
// collection pairs, the landmarks, and the naming role.
//
// The tab pair is the one row of that table where the two natives disagree
// about *which half* they can say, and it is a useful illustration of why the
// vocabulary is ARIA's rather than either platform's. Compose has a Role.Tab
// for the control and nothing for the strip around it; SwiftUI has
// .isTabBar for the strip and nothing for the control. Neither could have
// supplied the pair, and a caller marking up a tab strip sets both and gets
// whichever half each platform knows.
//
// Twelve of the twenty-three do nothing on either native, and that is the
// honest state of those platforms rather than a gap to be filled later:
// neither has a tabular semantics vocabulary a role can be mapped onto (Compose
// has collectionInfo, which describes counts and indices this prop does not
// carry), neither has a listbox in its semantics vocabulary (both spell a
// chosen item as a *state* instead, which is why the selectable pair costs
// them nothing to leave out — see RoleListBox), and neither has landmarks at
// all — VoiceOver's rotor navigates by heading, not by banner.
//
// RoleGroup is the one empty pair in that twelve that is empty for the
// opposite reason, and it is worth telling apart. The other eleven are silent
// because the platform has no way to say the thing; `group` is silent because
// neither platform *needs* it — both honour an accessibility label on any node
// at all, and making that label legal is the whole of what the role does. See
// its own block below.
//
// A role that maps to nothing is still worth setting. The web is a first-class
// target here, the mapping can improve later without the call sites changing,
// and a role that is right on one platform and inert on two is strictly better
// than a div.
//
// # A structural role owns what is inside it
//
// The tabular five and the two collection pairs are not labels on a container
// — they are claims about what the container holds. role="list" says its
// children are listitems; role="listbox" says its children are options;
// role="table" says its children are rows, or rowgroups holding rows. A reader acts on the claim rather than re-deriving
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
// and are not subject to this. A banner, a navigation region or a log owns
// whatever it likes; RoleHeading, RoleButton, RoleLink, RoleImg and RoleTab
// describe the node itself.
// Four of the const blocks below hold a role that makes a claim about its
// children — the tabular set, both collection pairs, and the tablist half of
// the tab pair — which is where to look rather than here if a role is ever
// added to any of them. (RoleTab itself does not: it describes one control, the way
// RoleButton does, and only the strip around it claims what it contains.)
//
// # Roles a node type carries for itself
//
// Some semantics are not the author's to state, because the node type already
// knows them. core.Button exports as a <button> and builds a real control on
// both natives, so nobody has to say `button` — RoleButton exists for the
// *other* case, a Box or a Row with an OnTap, which every renderer draws as
// inert scenery.
//
// core.Modal is the same shape and has no vocabulary entry at all. The two
// natives already present it through a platform dialog (a SwiftUI sheet, a
// Compose Dialog), each of which announces itself; the two DOM renderers drew
// a plain div, so the overlay was the one target where a dialog was not a
// dialog. Both now write role="dialog" and aria-modal="true" as part of the
// Modal chassis, next to the fixed-overlay rules — semantics the node type
// owns, not a value a caller passes.
//
// So there is deliberately no RoleDialog. Adding one would put the burden back
// on the author for something three of the four targets already do
// unasked, and would cost two more native arms that could only be empty —
// which in this vocabulary means "this platform cannot say it", the opposite
// of the truth here. An author who overrides a Modal's role with a
// core.AccessibilityRole still wins, on the same principle the chassis follows
// for style: the framework's default goes first.
//
// There is deliberately no RoleTabPanel either, and it is the third case of
// the same shape rather than an oversight beside RoleTab and RoleTabList.
//
// A tab panel is not really a role: it is one end of a *relationship*. The
// announcement a reader gives ("tab 2 of 3, Sermons, tab panel") comes from
// aria-controls and aria-labelledby pointing between the two elements, and
// both of those are IDREFs. Style carries values, not references — the same
// reason accessibilityAttrs spells a hint as aria-description rather than
// aria-describedby — so a RoleTabPanel would hand an author the half of the
// wiring that says the least and no way at all to write the half that says
// the most.
//
// core.TabView already owns that relationship end to end. Both DOM renderers
// mint the ids, write role="tabpanel" on the page, wire aria-controls and
// aria-labelledby across, and keep aria-selected in step with the selection;
// both natives hand the whole strip to the platform's own tab container,
// which announces itself. It is exactly Modal's shape: semantics the node
// type owns because it is the only thing that can see both halves.
//
// The absence is load-bearing in one more place, which is worth knowing
// before anyone adds the constant "for symmetry". The WASM runtime tells the
// wiring's own role apart from an author's by the value — an element carrying
// "tabpanel" can only have got it from wireTabPanel, because no core.Role
// spells it — and uses that to avoid unwiring and rewiring a panel on
// alternate syncs. htmlout's TestNoRoleCollidesWithTheTabPanelWiring holds
// the vocabulary to it.
//
// RoleTab and RoleTabList are not in the same position and are therefore
// present: they say what a control and a strip *are*, which is a claim about
// one element, and a hand-built strip of chips that switches a screen's
// content has no node type to say it for them.
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

// The selectable collection: a run of choices, and one choice in it. The
// fourth structural block, and the pair `list`/`listitem` above cannot stand
// in for.
//
// # Why a second collection pair rather than a state on the first
//
// Because ARIA will not carry it. `aria-selected` is defined for gridcell,
// option, row, tab and columnheader — not for `listitem` — so a list item
// that says it is chosen says it into a void on both web targets: the
// attribute is written, the DOM inspector shows it, and no reader announces
// anything. A list is *content* and a listbox is a *control*, and the state
// only exists on the control side.
//
// components.ListRow is what asked. Its selected row spelled the state into
// its own accessible name (", selected") because both other doors were shut:
// `listitem` cannot carry the state, and `button` — which carries the
// neighbouring `aria-pressed` — would make the row a foreign child of any
// role="list" around it, costing the whole list its shape for one row's
// announcement. This pair is the door that was left.
//
// # What a listbox promises, and who keeps the promise
//
// A listbox is a real control in ARIA's model, and the pattern that goes with
// it is larger than two attributes: the container takes keyboard focus, the
// arrow keys move an active option, and the reader is told which option is
// active through a roving tabindex or aria-activedescendant.
//
// None of that is *here*, and it does not need to be. This type is a
// vocabulary — it says what a node is, and nothing in core stamps a tabindex
// or reads an arrow key (core/focus.go is about putting the cursor in a named
// field, which is a different question and one that costs a render pass per
// keystroke). What closed the gap instead was noticing that the vocabulary
// already says everything the pattern needs:
//
//	what is a member of what   this pair, and the structural rule above that
//	                           makes a listbox's contents its own
//	which one is chosen        Style.AccessibilitySelected
//	which way the arrows go    the container's own layout axis
//	what activation means      the OnTap the author already wrote
//
// So the WASM runtime supplies the whole keyboard half from what crosses the
// wire, with no new prop and no source change in any screen that had already
// said the above — see "Composite widgets are operable" in
// docs/platforms/wasm.md. htmlout deliberately writes none of it: a roving
// tabindex without the handler that moves it takes every option but one out of
// the tab order and reaches none of them, so the attribute is behaviour rather
// than semantics and a static export must not carry it.
//
// On the two phones there was never a gap — VoiceOver and TalkBack navigate a
// collection by swipe, not by arrow key — which is also why neither native has
// a listbox in its semantics vocabulary at all: both spell a chosen item as
// the `selected` state this pair exists to make *legal*, and they honour that
// state on any node without being told what contains it.
//
// # The depth question, answered the other way
//
// `listitem` carries aria-level and `option` does not, so a row cannot be
// both a choice and a depth: the two roles are exclusive and only one of them
// takes a level. ARIA does have a role for an item that is both — `treeitem`
// inside a `tree`, which supports aria-level and aria-selected together — and
// it is deliberately not here, because a tree is a third pattern with its own
// expansion state and its own keyboard contract, and nothing in this
// repository has one. See components.ListRow.Selectable, which is where the
// two fields meet and where the precedence is written down.
const (
	RoleListBox Role = "listbox"
	RoleOption  Role = "option"
)

// The tab pair: a strip of controls that switches what the screen is showing,
// and one control in it. The third structural block, and it makes the same
// claim about its children the other two do — role="tablist" says the things
// inside it are tabs, so a strip that also holds a "+" button or a count is
// not a tablist. See "A structural role owns what is inside it" above.
//
// Both are for a *hand-built* strip. core.TabView needs neither: it writes
// these two, plus the tabpanel half and the aria-controls/aria-labelledby
// wiring between them, from the node type — see "Roles a node type carries
// for itself" above for why there is no RoleTabPanel to complete the set.
//
// A tab is the vocabulary's second selectable control, after RoleButton, and
// the state it wants is Style.AccessibilitySelected. The pair is what makes
// the strip legible: a reader announces "tab, selected" only when the tab
// says so, and a tablist in which *no* tab says so announces every one of
// them as unselected. So a strip sets the state on every tab, not just the
// live one — SelectedOff is a value with a job here, not a way of saying
// nothing.
//
// It is also what the arrow keys move between. The WASM runtime reads this
// pair the same way it reads the listbox one and supplies ARIA's keyboard
// half — one tab stop for the strip, Left/Right within it (Up/Down for a strip
// laid out as a column), Home and End to the ends — from the roles and the
// selection alone. A hand-built strip gets it by saying what it is; see the
// listbox pair above for the argument, and docs/platforms/wasm.md for what
// each target does.
const (
	RoleTab     Role = "tab"
	RoleTabList Role = "tablist"
)

// RoleListItem and RoleRow are the two values with a depth question attached:
// how far inside a nested collection the item sits. That is
// Style.AccessibilityNestingLevel, which is a *second* int rather than a
// widening of the heading one — ARIA's aria-level serves all three roles, but
// a heading's tier stops at 6 (that is all HTML and SwiftUI can spell) and a
// nesting depth has no ceiling, so one field would have to pick a rule that is
// wrong for one of them. See that field's doc for the rest of the argument.

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
// The first two differ in how rudely they interrupt. Status waits for a pause
// — "saved", "3 new items". Alert cuts in — a failure, an expiry, anything the
// reader must hear before continuing. Banner picks between them by variant,
// which is the distinction its Variant already draws visually.
//
// RoleLog is the third, and it is not a politeness level: log and status are
// both polite, and on Android they are the same call. What differs is the
// *shape of the content*, which is a promise about the element rather than
// about the interruption.
//
//	status   one advisory that is replaced. "Saved", "3 new items". A reader
//	         announces the region's new state, and the old text is gone.
//	log      a record that is appended to and whose order is meaningful — a
//	         chat transcript, a console, a running import. A reader announces
//	         what arrived, and what came before it is still there to be read
//	         back.
//
// The distinction is the difference between "the region now says this" and
// "this was added at the end", which is why a transcript marked `status`
// announces correctly and reads back wrong: the whole conversation is one
// region that has, as far as the reader is concerned, just changed entirely.
//
// It maps to Compose's polite live region, the same call `status` makes,
// because that is the whole of what Compose can say — an honest collapse
// rather than a second spelling of one fact. On the web the two are different
// roles with different reading behaviour, which is where the field earns its
// place.
const (
	RoleStatus Role = "status"
	RoleAlert  Role = "alert"
	RoleLog    Role = "log"
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
// RoleHeading is the one value here with a second question attached: how deep
// the heading sits. That is Style.AccessibilityHeadingLevel, a separate int
// rather than a lettered set of constants — see its doc for why six more
// spellings of "heading" would cost both DOM renderers the mapping table this
// vocabulary exists to avoid.
//
// RoleImg is for a node that is a *picture* — something whose meaning is
// carried by its arrangement rather than by any text inside it, and which
// therefore needs one text alternative standing in for the whole thing.
// components.Compass is the case that asked for it: a rose read in tree order
// is "N W E S" whatever direction it is pointing, so the widget hides its
// parts and speaks once.
//
// It was also, for a while, the only role that made such a label *work at all*
// on the web: ARIA forbids an accessible name on a generic element, so an
// AccessibilityLabel on a plain container was dropped by screen readers rather
// than announced. RoleGroup below is now the general answer to that, and the
// division between the two is what the node is rather than what it needs — an
// img stands in for its parts and should hide them, a group names them and
// leaves them readable. Reach for this one only when the picture reading is
// true.
//
// A node with this role should hide its children, or the reader gets the
// alternative *and* the parts it was standing in for.
const (
	RoleHeading Role = "heading"
	RoleButton  Role = "button"
	RoleLink    Role = "link"
	RoleImg     Role = "img"
)

// The naming role: the least a container can be, and the only thing that makes
// an accessible name on one legal at all.
//
// # The silence it closes
//
// Every layout node in this framework exports as a <div> or a <span>, and both
// tags carry the implicit ARIA role `generic`. ARIA prohibits an accessible
// name on `generic` — aria-label and aria-labelledby are listed under "roles
// which cannot be named" — and browsers enforce it by pruning the name from
// the accessibility tree. So:
//
//	core.Box(core.AccessibilityLabel("Unread messages"), …)
//
// wrote a correct-looking attribute that no screen reader on either web target
// announced, while VoiceOver and TalkBack read it out perfectly, because a
// SwiftUI accessibilityLabel and a Compose contentDescription are honoured on
// any node without asking what it is. Two targets silent, two fine — which is
// what let it ship: the two that work are the two a developer is most likely
// to be testing on.
//
// RoleImg was the first door out and it is the wrong shape for most rows. It
// says the node is a *picture* whose parts should be hidden behind one
// alternative, which is true of components.Compass and false of a list row, a
// disclosure header or a stat tile — all of which want their contents read as
// well as their name.
//
// # Why `group` and not one of the louder candidates
//
//	region     also nameable, and a landmark. A reader adds every region to
//	           the list it jumps between, so naming six rows would put six
//	           entries in a screen's table of contents.
//	button     claims a control, makes its children presentational (a heading
//	           inside one stops being a heading), and is a foreign child of any
//	           list around it — see components.ListRow, which turned it down
//	           for exactly that.
//	group      "a set of user interface objects", nameable, not a landmark,
//	           and with no required children and no presentational-children
//	           rule. It says these things belong together and this is what they
//	           are called, and nothing else.
//
// That "nothing else" is the whole recommendation. A role is a claim, and the
// structural rule above says a claim a container cannot keep is worse than no
// role at all; `group` is the one value in this vocabulary that promises
// nothing about what it holds, so it can be given to a container nobody has
// looked inside.
//
// # It is also a fallback, not only a constant
//
// Because the silence is a framework bug rather than an author's mistake, the
// two web exporters supply this role themselves: a node with an accessible
// name, no role of its own, and a generic tag is written role="group" so the
// name is heard. An author who says anything more specific wins — the fallback
// only ever fills an empty slot. See accessibilityAttrs in htmlout/export.go
// and applyAccessibility in wasm/grmob-runtime.js, which restate one rule.
//
// The fallback cannot make anything worse, which is the argument for doing it
// silently. Before it, the name was invalid ARIA that was dropped; after it,
// the name is valid ARIA that is announced. The one thing it could disturb is
// a structural container's claim about its children — but a generic div inside
// a role="list" was never a `listitem` either, so a `group` there is the same
// foreign child it already was, one attribute louder.
//
// # Both natives leave it empty for the opposite of the usual reason
//
// Nine of the roles here are inert on SwiftUI and Compose because those
// platforms have no way to say the thing. This one is inert because they have
// no need to: both already announce a label on any node, so the role that
// makes the label legal buys them nothing. Setting it therefore costs nothing
// anywhere and closes a two-target silence.
const RoleGroup Role = "group"

// Roles returns every declared Role except RoleNone, in declaration order.
//
// RoleNone is excluded because it is the absence of a role rather than one of
// them: it is the field's zero value, no renderer has an arm for it, and a
// coverage check that demanded one would be asking each renderer to implement
// "unset". Everything downstream that iterates roles — the native dispatch
// pins, the DOM export test — wants the twenty-three that do something.
//
// A fresh slice per call rather than a package-level var, which any importer
// could write to. Twenty-three elements are cheaper to build than to defend.
//
// Pinned to the const blocks above by role_enum_test.go, which reads this
// file's syntax tree: adding a constant without adding it here should fail
// `go test ./...` rather than silently shrink the set every renderer's
// coverage check rests on.
func Roles() []Role {
	return []Role{
		RoleTable, RoleRowGroup, RoleRow, RoleColumnHeader, RoleCell,
		RoleList, RoleListItem,
		RoleListBox, RoleOption,
		RoleTab, RoleTabList,
		RoleBanner, RoleNavigation, RoleSearch, RoleToolbar,
		RoleStatus, RoleAlert, RoleLog,
		RoleHeading, RoleButton, RoleLink, RoleImg,
		RoleGroup,
	}
}
