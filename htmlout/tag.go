package htmlout

import "sort"

// tags is the one authoritative statement of the node type -> HTML tag table.
//
// Every renderer that targets the DOM has to answer the same question — which
// element does a Row become, which does a Checkbox become — and two of them
// are not written in Go:
//
//	htmlout (this package)      queries it through TagFor
//	wasm/grmob-runtime.js       restates it in JavaScript (tagForType)
//
// Only the first can literally share this map; the runtime calls
// document.createElement in the browser and has no way to ask Go. Its copy is
// pinned to this one by TestRuntimeTagsMatchGo in wasm/verify, which reads the
// table out of the runtime source and compares it here under a plain
// `go test ./...`, so the surviving duplication is checked rather than
// remembered. This is the same treatment inputTypes gets, and for the same
// reason: a copy that cannot drift silently is a restatement, not a second
// source.
//
// The table is a census, not a list of exceptions: the node types that become
// a plain <div> are spelled out alongside the ones that do not. (It said
// "fourteen" of them for a while and was wrong by two, which is the argument
// against counting them here at all — the list is the census, and a number
// beside it is a second copy that goes stale on its own.)
// The default below still exists for a node type nobody has taught either
// renderer about, but a type that is merely *ordinary* should appear here, so
// that adding a node type to core and forgetting the renderers shows up as a
// gap in a list rather than as silence.
//
// The tag alone does not always finish the job. Six types share <input>, and
// which control the browser draws is decided by the type attribute — see
// inputTypes, whose keys are exactly the <input> rows here. One pair it cannot
// separate, and says so: a Switch and a Checkbox are the same type attribute
// and differ by one more.
var tags = map[string]string{
	"Text":     "span",
	"Button":   "button",
	"Image":    "img",
	"TextArea": "textarea",

	// The picker (core.Select). Its <option> elements are built from the
	// options prop rather than from child nodes, so they are not in this
	// table: no patch is ever addressed to one and none of them carries a
	// style. See core.Select on why the list travels as a prop.
	"Select": "select",

	// A monospace grid and its rows (core.TextGrid). <pre> is the one element
	// whose default styling already says "fixed pitch, no wrapping"; each row
	// is a block inside it and each run a <span>, so a row's runs can carry
	// their own colours without the grid being anything but text.
	"TextGrid": "pre",
	"GridRow":  "div",

	// The prose editor (core.RichTextEditor). A <div>, because what it holds is
	// a *document* — headings, paragraphs, lists — and there is no element that
	// means "a document". On the web target the same div is made
	// contenteditable; here it holds richtext.Doc.HTML() and nothing else.
	"RichTextEditor": "div",

	// The programmer's editor (core.CodeEditor). The same <pre> a TextGrid is,
	// and for the same reason — its rows *are* a grid's rows, built by the same
	// gridRowNode — with the gutter and, in the live runtime, a transparent
	// <textarea> overlaid on it as chrome inside the same box. See
	// codeeditor.go for why the rows stay the <pre>'s direct children.
	"CodeEditor": "pre",

	// The six that share one tag and are told apart by inputTypes — except
	// for the last pair, which inputTypes cannot tell apart at all: a Switch
	// and a Checkbox are both type="checkbox", and what separates them is the
	// `switch` attribute the exporter writes for one of them. See inputTypes.
	"Input":         "input",
	"InputPassword": "input",
	"NumericInput":  "input",
	"Checkbox":      "input",
	"Switch":        "input",
	"Slider":        "input",

	// Containers and boxes. A <div> is the honest answer for all of them:
	// what distinguishes a Row from a Column in HTML is the flex declarations
	// styleValue emits, not the element, which is also why the runtime keeps
	// the Go type in data-node-type rather than trying to read it back off
	// the tag.
	"Box":      "div",
	"Card":     "div",
	"Column":   "div",
	"Row":      "div",
	"Scroll":   "div",
	"SafeArea": "div",
	"List":     "div",
	"Modal":    "div",
	"TabView":  "div",
	"Spacer":   "div",

	// The z-stack. A <div> like the rest — what makes it an overlay is the
	// single-cell grid styleValue gives it and the grid-area it imposes on
	// its children, not the element. See overlayTypes in stack.go.
	"ZStack": "div",

	// A placeholder box in both DOM renderers; neither opens a camera.
	"CameraView": "div",

	// The live map (core.MapView) and its pins. A <div> in both DOM
	// renderers, and the two differ in what happens to it afterwards: the
	// WASM runtime hands the div to Leaflet, while a static export leaves it
	// a placeholder, as CameraView's is — an exported document has no engine
	// to run and no tiles to fetch.
	//
	// A Marker is data rather than a box, and it gets an element anyway
	// because a patch path is positional: a node with no element would send
	// every later patch to the wrong place. It is exported with the
	// coordinates on it and nothing inside it; see renderNode's arm.
	"MapView": "div",
	"Marker":  "div",
}

// defaultTag is what an unrecognized node type renders as. A div is the
// neutral choice — it draws nothing of its own and accepts any style — and it
// is what both DOM renderers already fell back to before this table existed.
const defaultTag = "div"

// TagFor returns the HTML tag a node type renders as.
//
// Fragment and Theme are not in the table and must not be asked: see
// transparentTypes. The lookup answers *which element*, never *whether an
// element*, and a caller that has not made the transparency decision first
// gets defaultTag — a box those two node types are not supposed to have.
func TagFor(nodeType string) string {
	if tag, ok := tags[nodeType]; ok {
		return tag
	}
	return defaultTag
}

// Tags returns a copy of the whole table, for the callers that must enumerate
// it rather than query it — the WASM runtime conformance test, which has to
// compare table against table and so cannot go through TagFor one key at a
// time, and this package's own test that the exporter agrees with it.
//
// A copy, not the map itself, for the reason InputTypes returns one: a
// package-level map is reachable and writable by any importer.
func Tags() map[string]string {
	out := make(map[string]string, len(tags))
	for k, v := range tags {
		out[k] = v
	}
	return out
}

// genericTags are the tags in the table above whose implicit ARIA role is
// `generic` — that is, the ones that name nothing to a screen reader on their
// own, so writing a role= attribute onto them adds meaning instead of
// destroying it.
//
// The distinction has one caller today, the TabView panel wiring: a tab's
// aria-controls has to name an element carrying role="tabpanel", and the page
// it points at is whatever node type the app put there. Stamping the role onto
// a <button>, an <img> or an <input> page would replace the role the browser
// already gives it, which is a worse outcome than leaving that page unwired —
// the whole point of the wiring is accessibility, so it must not cost any.
//
// A whitelist rather than a blacklist of the roled tags, because the safe
// direction for a tag nobody has considered yet is "not eligible": a new row
// in the table above joins this set deliberately or not at all.
//
// The WASM runtime restates it as GENERIC_TAGS in grmob-runtime.js, and
// TestRuntimeGenericTagsMatchGo in wasm/verify compares the two under a plain
// `go test ./...` — the same treatment tags and inputTypes get, and for the
// same reason: a copy that cannot drift silently is a restatement rather than
// a second source.
var genericTags = map[string]bool{
	"div":  true,
	"span": true,
	"pre":  true,
}

// IsGenericTag reports whether a tag's implicit ARIA role is `generic`, and so
// whether a role= attribute may be written onto it. See genericTags.
func IsGenericTag(tag string) bool {
	return genericTags[tag]
}

// GenericTags returns the role-free tags, sorted so that a test looping over
// them reports in a stable order. Exported for the reason Tags and
// TransparentTypes are: the WASM conformance test has to compare set against
// set, and a hand-written list there would be exactly the untracked second
// copy this file exists to remove.
func GenericTags() []string {
	out := make([]string, 0, len(genericTags))
	for t := range genericTags {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// transparentTypes are the grouping nodes that have no visual box of their
// own: core.For wraps its generated children in a Fragment, and core.WithTheme
// wraps a subtree in a Theme. Neither ever carries a Style (core/conditionals.go,
// core/layout.go and core/theme.go all construct the node with Children and
// nothing else), and both natives already render them transparently — SwiftUI
// as a Group, Compose as a bare RenderChildren into the parent's scope.
//
// They are held apart from tags rather than given a "div" entry because a
// wrapper for them is not a cosmetic difference. Inside a flex container the
// wrapper becomes the single flex item, and it swallows the parent's gap,
// flex-direction and alignment before any of it can reach the children that
// were supposed to receive them.
//
// # The one place the two DOM renderers genuinely disagree
//
// htmlout honors the transparency; the WASM runtime cannot, and boxes both in
// a <div>. That is structural rather than an oversight. Patches are addressed
// by positional path — reconcile.Patch.TargetID is "root/1/0", built by walking
// node.Children (reconcile/patch.go), and the runtime resolves it against
// data-node-path attributes it wrote by walking the same indices. Dropping an
// element for a Fragment would put the DOM out of step with the node tree and
// send every patch beneath it to the wrong element. htmlout is a static
// snapshot with no patch stream to keep addressable, so flattening costs it
// nothing.
//
// The divergence is named here, and pinned by TestRuntimeTagsMatchGo, so that
// it reads as a decision with a reason rather than as drift nobody caught.
var transparentTypes = map[string]bool{
	"Fragment": true,
	"Theme":    true,
}

// IsTransparent reports whether a node type renders its children directly into
// the parent, with no element of its own.
func IsTransparent(nodeType string) bool {
	return transparentTypes[nodeType]
}

// TransparentTypes returns the transparent node types, sorted so that a test
// looping over them reports in a stable order. Exported for the same reason
// Tags is: the WASM conformance test has to know which types are excluded from
// the tag comparison, and a hand-written list there would be exactly the
// untracked second copy this file exists to remove.
func TransparentTypes() []string {
	out := make([]string, 0, len(transparentTypes))
	for t := range transparentTypes {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// ownRoles are the node types that state their own ARIA role, and the role
// each one states. Neither value is in the core.Role vocabulary, and that is
// the property the map is here to carry: these are roles this framework emits
// and does not name — see core/role.go's "Roles a node type carries for
// itself", and aria/spec.NearMisses, where both appear so the guards have
// something to argue with.
//
//	Modal    a dialog by virtue of being an overlay. core.ModalNode has no
//	         Style field at all, so the role has nowhere else to come from.
//	         modalSemantics (export.go) writes it, plus the aria-modal no Role
//	         could express.
//	Switch   a switch by virtue of being one. HTML has no switch element, so
//	         the control is an <input type="checkbox"> and this attribute is
//	         what makes a reader announce it as the thing it is. See
//	         core.Switch, and switchSemantics in export.go.
//
// It was a single `nodeType == "Modal"` comparison while Modal was alone, and
// the second entry is what turns the question into a table: the cost of
// getting this wrong is a *duplicate* role attribute on one element, which is
// invalid and which no amount of reading the two call sites would reveal.
var ownRoles = map[string]string{
	"Modal":  "dialog",
	"Switch": "switch",
}

// CarriesOwnRole reports whether a node type states its own ARIA role, with no
// core.Style involved. See ownRoles.
//
// Exported because the TabView wiring has to know: the role attribute has one
// slot per element, and a page whose type already filled it must not be given
// role="tabpanel" on top.
func CarriesOwnRole(nodeType string) bool {
	_, ok := ownRoles[nodeType]
	return ok
}

// OwnRoleFor returns the ARIA role a node type states for itself, or "" for a
// type that states none. See ownRoles.
//
// Exported for the reason Tags is: the WASM runtime has the same two node
// types to answer for and cannot ask Go at runtime, so wasm/verify holds its
// copy against this one rather than against a list written twice.
func OwnRoleFor(nodeType string) string {
	return ownRoles[nodeType]
}

// borderResetTypes are the node types whose *user-agent* stylesheet draws a
// border of its own, and which therefore have to be told not to when the Go
// style asks for no border.
//
// # The divergence this closes
//
// Every other target draws a border only when asked. Compose applies a
// Modifier.border and SwiftUI a .grMobBorder overlay, both guarded on
// `BorderWidth > 0 && BorderColor != ""`, and the two DOM renderers emit their
// `border` declaration under the same guard — so "no border in the style" means
// "no border on screen" on three targets and, on the web, meant "whatever the
// browser draws". A <button>, an <input> and a <textarea> are the tags in the
// table above where the browser draws something, and no style could turn it
// off, since core.BorderWidth(0) emits nothing and nothing is exactly what
// leaves the user agent in charge.
//
// The visible cost was components.Button's EmphasisGhost, documented as
// "EmphasisOutlined without the rule" and drawing a rule on both web targets
// and none on both phones. There was no call-site workaround.
//
// # Why this is keyed by node type and not by tag
//
// It was a set of tags while <button> was the only member, because one node
// type becomes a <button> and the two questions were the same question. Text
// fields ended that: five node types share <input> and only three of them want
// the reset.
//
//	Input, InputPassword, NumericInput   a frame the style should own
//	TextArea                             the same, one tag over
//	Select                               the same, a third tag over
//	Checkbox, Switch, Slider             the user agent draws the *control*
//
// A checkbox's border is not chrome around the control, it is the box; a switch
// joins it on that argument, since it is the same element with one attribute
// more; a range track has no border to reset in the first place. Both draw through
// `appearance: auto`, where a browser ignores the property anyway — so keying
// by tag would have been harmless today and wrong on the day someone reaches
// for appearance:none. The question the set answers is "does this element draw
// a frame the Go style is meant to own", which is per-control, so the map is
// per-node-type and the tag lookup drops out of the call.
//
// # The <select> row, and how it was decided
//
// A picker was the open question this set was left holding: the answer was
// written down ("join if the style is meant to own the frame, stay out if the
// browser draws the control") a good while before core.Select existed to be
// asked about. It joins, and the deciding fact is what the *other three*
// targets do rather than anything about the tag.
//
// core.Select is drawn on both natives as an ordinary styled box with a menu
// hung off it — a SwiftUI Menu, a Compose DropdownMenu — deliberately not as a
// platform picker control, because SwiftUI's .pickerStyle(.menu) and Material's
// ExposedDropdownMenuBox each draw a frame of their own that no Go style could
// remove. So on three targets out of four the frame is the theme's, from the
// Components.Input base the widget reads, and the web is again the one place a
// user agent was drawing a second one on top.
//
// Only the frame is reset. A <select>'s drop-down indicator is not chrome
// around the control, it is the thing that says the control is a picker — the
// checkbox's own argument, one row down — and `border` does not touch it. It
// stays, on every browser, and it is the one part of the control the theme
// does not own.
//
// # Why the text fields could not join until the themes moved
//
// Resetting a border the theme does not replace is levelling down, and until
// both bundled themes grew a Components.Input / Components.TextArea frame the
// reset would have left every web text field an unmarked rectangle — the
// browser's border was the only thing drawing the control at all. The natives
// already had that problem (Compose renders a bare BasicTextField, SwiftUI a
// .plain textFieldStyle, and both draw only what the Go style asks for), which
// is what made the missing frame a theme bug rather than an argument for
// keeping the web's. Both themes now state one, so all four targets draw the
// same edge from the same field.
//
// A theme that predates those defaults and sets no Input border of its own now
// renders a borderless field on the web, as it always did on both phones. That
// is the point of the reset rather than a casualty of it: one style, one
// answer, everywhere.
//
// The WASM runtime restates this as BORDER_RESET_TYPES in grmob-runtime.js, and
// TestRuntimeBorderResetTypesMatchGo in wasm/verify compares the two under a
// plain `go test ./...` — the same treatment tags, inputTypes and genericTags
// get, and for the same reason.
var borderResetTypes = map[string]bool{
	"Button":        true,
	"Input":         true,
	"InputPassword": true,
	"NumericInput":  true,
	"TextArea":      true,
	"Select":        true,
}

// ResetsUABorder reports whether a node type needs an explicit "no border"
// written for it when the style declares none. See borderResetTypes.
func ResetsUABorder(nodeType string) bool {
	return borderResetTypes[nodeType]
}

// BorderResetTypes returns those node types, sorted so that a test looping over
// them reports in a stable order. Exported for the reason GenericTags is: the
// WASM conformance test has to compare set against set.
func BorderResetTypes() []string {
	out := make([]string, 0, len(borderResetTypes))
	for t := range borderResetTypes {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}
