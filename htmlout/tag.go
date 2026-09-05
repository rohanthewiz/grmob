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
// The table is a census, not a list of exceptions: the fourteen node types
// that become a plain <div> are spelled out alongside the ones that do not.
// The default below still exists for a node type nobody has taught either
// renderer about, but a type that is merely *ordinary* should appear here, so
// that adding a node type to core and forgetting the renderers shows up as a
// gap in a list rather than as silence.
//
// The tag alone does not always finish the job. Four types share <input>, and
// which control the browser draws is decided by the type attribute — see
// inputTypes, whose four keys are exactly the four <input> rows here.
var tags = map[string]string{
	"Text":     "span",
	"Button":   "button",
	"Image":    "img",
	"TextArea": "textarea",

	// A monospace grid and its rows (core.TextGrid). <pre> is the one element
	// whose default styling already says "fixed pitch, no wrapping"; each row
	// is a block inside it and each run a <span>, so a row's runs can carry
	// their own colours without the grid being anything but text.
	"TextGrid": "pre",
	"GridRow":  "div",

	// The five that share one tag and are told apart by inputTypes.
	"Input":         "input",
	"InputPassword": "input",
	"NumericInput":  "input",
	"Checkbox":      "input",
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

	// A placeholder box in both DOM renderers; neither opens a camera.
	"CameraView": "div",
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

// CarriesOwnRole reports whether a node type states its own ARIA role, with no
// core.Style involved.
//
// Modal is the only one: core.ModalNode has no Style field at all, and the
// overlay is a dialog by virtue of being an overlay — see modalSemantics in
// export.go, and core/role.go's "Roles a node type carries for itself" for why
// this is not a value in the Role vocabulary.
//
// Exported because the TabView wiring has to know: the role attribute has one
// slot per element, and a page whose type already filled it must not be given
// role="tabpanel" on top.
func CarriesOwnRole(nodeType string) bool {
	return nodeType == "Modal"
}
