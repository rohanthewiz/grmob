// Package htmlout exports a rendered core.Node tree as a standalone HTML
// document. It is the demo/inspection path (the example apps print its output);
// the WASM runtime does not consume it, so readability is favored over
// compactness.
package htmlout

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/rohanthewiz/element"
	"github.com/rohanthewiz/grmob/core"
)

// ExportHTML renders the node tree into a complete HTML document.
//
// Output is built on the element library rather than hand-assembled strings so
// that escaping is handled once, in one place: element quote-escapes every
// attribute value (a raw double quote is the attribute-breakout character), and
// text content goes through TE(), which entity-escapes it. User-originated
// strings — Text content, input values, labels, image srcs — therefore cannot
// re-enter the document as live markup.
func ExportHTML(node *core.Node) string {
	b := element.NewBuilder()
	// b.Html writes the <!DOCTYPE html> declaration itself.
	b.Html("lang", "en").R(
		b.Body().R(
			// "root" is the node path of the tree's root, the same name Go's
			// reconciler gives it (reconcile.Patch's TargetIDs are "root/1/0")
			// and the same one the WASM runtime mounts with. Nothing in the
			// exported document carries the path itself — this is a static
			// snapshot with no patches to address — but the TabView chrome
			// derives its element ids from it, and deriving them from the same
			// name on both web targets is what makes those ids identical
			// rather than merely well-formed. See tabScope in tabview.go.
			renderNode(b, node, imposed{}, "root"),
		),
	)
	// Pretty re-indents the compact single-pass output for human readers.
	// Escaped content is inert entities by this point, so re-parsing is safe.
	return b.Pretty()
}

// imposed is what a *parent* puts on the element standing in for one of its
// children — the two channels through which a decision that belongs to the
// parent reaches markup that is assembled in the child.
//
// Two callers fill it in, and both for the same underlying reason — the fact
// being expressed is the parent's and the markup is the child's:
//
//	renderTabView     hides the pages that are not selected (decl) and names
//	                  each page as the panel its tab controls (attrs). A page
//	                  does not know it is a page; only the TabView does.
//	renderContainer   places every child of a core.ZStack in the overlay's one
//	                  grid cell (decl). A layer does not know it is a layer.
//
//	decl   a CSS declaration list, appended after everything the node itself
//	       declares so it wins the browser's last-one-wins parse — which is
//	       what lets display:none outrank the display:flex a stack container
//	       is given unconditionally
//	attrs  element-style key/value pairs, appended to the node's own attribute
//	       list. One of the three the panel wiring writes — role — is an
//	       attribute the node can also write for itself, so renderNode asks
//	       whether the parent has claimed the slot (imposesRole) and skips its
//	       own if so. HTML gives an element one value per attribute name and a
//	       browser keeps the *first*, so writing both would silently hand the
//	       page the weaker of the two roles. The other two do not collide: the
//	       wiring's id is only ever written to a page that has no
//	       AccessibilityID of its own (tabPanelBox), and no core.Style field
//	       maps onto aria-labelledby at all.
type imposed struct {
	decl  string
	attrs []string
}

// renderNode writes one node (and, for containers, its subtree) into the
// builder. The return value exists only so calls can sit inline as R()
// arguments, which is how element establishes evaluation order; it is ignored.
//
// from is what the parent imposes on this node; see imposed. It is forwarded
// whole through the transparent branch below, because a Fragment used as a tab
// page has no box of its own, so what the parent meant for the page has to
// reach the children that are standing in for it.
//
// path is this node's address in the node tree ("root", "root/1", "root/1/0"),
// walked by child index exactly as reconcile.Patch builds its TargetIDs and as
// the WASM runtime writes its data-node-path attributes. It is not emitted:
// a static document has no patches to address. It exists so that a TabView can
// derive document-unique element ids for the tab/panel wiring, and derive them
// the same way the runtime does — see tabScope in tabview.go. Transparency
// does not disturb it: a Fragment has no element, but it is still a node, and
// its children are still its children by index on every target.
func renderNode(b *element.Builder, node *core.Node, from imposed, path string) (x any) {
	if node == nil {
		return
	}

	// Grouping nodes are emitted as their children, with no box of their own.
	// Which types those are, why a wrapper for them is a layout bug rather
	// than a redundant element, and why the WASM runtime is the one DOM
	// renderer that cannot do this, are all in transparentTypes (tag.go).
	if IsTransparent(node.Type) {
		for i, child := range node.Children {
			renderNode(b, child, from, childPath(path, i))
		}
		return
	}

	// Shared attributes: inline style first, then the callback-ID data
	// attributes the WASM runtime dispatches on (their names are a contract
	// with runtime/main.go's event bridge).
	attrs := make([]string, 0, 10)
	// The declaration list is assembled before the attribute so the two
	// prop-driven declarations below (object-fit, pointer-events) can join it
	// rather than needing a second, illegal, style attribute.
	sv := styleValue(node.Style, node.Type)
	if node.Type == "Image" {
		sv = addDecl(sv, objectFitDecl(getStr(node.Props["contentMode"])))
	}
	// The Modal overlay chassis, ahead of any style the node carries so an
	// author's style still wins. core.ModalNode carries no Style of its own —
	// its whole look is these fixed rules plus the backdrop prop — which is
	// why the declarations are authored here rather than in core.
	if node.Type == "Modal" {
		sv = addDecl(modalChassis(node.Props), sv)
	}
	// The grid chassis, ahead of the author's style for the same reason as
	// Modal's. A <pre> already has a monospace font and no wrapping; the
	// declarations here pin the two things a browser default leaves open
	// (the margin a <pre> carries, and a line height the rows can be sized
	// against) and let a wide grid scroll sideways rather than overflow.
	if node.Type == "TextGrid" {
		sv = addDecl(textGridChassis, sv)
	}
	if node.Type == "GridRow" {
		sv = addDecl(gridRowChassis, sv)
	}
	// The editor chassis, and the gutter inset that goes with it. Ahead of the
	// author's style for the same reason the three above are, and in two
	// pieces because only one of them depends on the node's props: the fixed
	// rules are a constant, the padding is a function of whether there is a
	// gutter and how many digits it needs. See codeeditor.go.
	if node.Type == "CodeEditor" {
		sv = addDecl(codeEditorChassis, sv)
	}
	// And the prose editor's, which is the opposite chassis: a code surface
	// does not wrap and a document does. See richtext.go.
	if node.Type == "RichTextEditor" {
		sv = addDecl(richTextChassis, sv)
	}
	// The Spacer's chassis, ahead of the author's style for the same reason
	// as the three above.
	if node.Type == "Spacer" {
		sv = addDecl(spacerChassis(node.Props), sv)
	}
	if node.Style != nil && node.Style.Disabled && !isFormControl(node.Type) {
		// HTML's disabled attribute is only valid on form controls, so a
		// disabled container gets the ARIA state plus the one declaration
		// that actually makes a browser stop routing pointer events to it.
		// Together they are the closest the export gets to what Compose's
		// `enabled = false` and SwiftUI's `.disabled(true)` do natively.
		sv = addDecl(sv, "pointer-events:none")
		attrs = append(attrs, "aria-disabled", "true")
	}
	// The gutter's inset, after the author's own declarations rather than
	// before them like every other chassis above. The exception is deliberate:
	// the other chassis rules are a *look* an author may disagree with, and
	// this one is layout the line numbers depend on — a padding-left the author
	// set would slide the numbers over the code. The WASM runtime writes it
	// after every style patch for the same reason (syncCodeGutter), so the two
	// DOM targets draw one picture.
	if node.Type == "CodeEditor" {
		sv = addDecl(sv, codeEditorPadding(node))
	}
	// Last, so it outranks the node's own declarations — including the
	// display:flex a stack container is given unconditionally, which is
	// exactly what a hidden tab page has to be talked out of.
	sv = addDecl(sv, from.decl)
	if sv != "" {
		attrs = append(attrs, "style", sv)
	}
	attrs = append(attrs, accessibilityAttrs(node.Style, node.Type, imposesRole(from.attrs))...)
	// The parent's attributes, after the node's own. Order is presentational
	// only — an attribute list is a set, not a cascade — but keeping the
	// node's own first means a reader of the markup sees what the node said
	// about itself before what its container said about it.
	attrs = append(attrs, from.attrs...)
	// A slice, not a map: map iteration order would make attribute order (and
	// therefore the exported document) nondeterministic across runs.
	for _, cb := range [...]struct{ prop, attr string }{
		{"onClick", "data-onclick"},
		{"onChange", "data-onchange"},
		{"onToggle", "data-ontoggle"},
		// Focus and blur are attributes here for the same reason the others
		// are: the export is a static document, so all it can do is record
		// which callback ID the edge belongs to and leave the wiring to
		// whatever loads it. Both are exported for every node type that
		// carries them — a browser gives focus to more than the natives do
		// (a link, anything with tabindex), and the export has no business
		// narrowing that.
		{"onFocus", "data-onfocus"},
		{"onBlur", "data-onblur"},
		// The return key's handler, which is also the traversal action: a
		// field in a core.UseFocusOrder carries an onSubmit that Go wired to
		// focus the next one. Exported like the rest — the ID, not the
		// behavior — so a loader that wires the other callbacks gets working
		// keyboard traversal from the same table.
		{"onSubmit", "data-onsubmit"},
		// core.OnEndReached. Recorded, not wired, like every other ID here:
		// the edge is a scroll position and a static document has no
		// observer to report one. What the attribute buys is that a loader
		// which does have one — grmob-runtime.js, or anything reading this
		// table — knows which callback the bottom of this list belongs to
		// without re-deriving it from the tree.
		{"onEndReached", "data-onendreached"},
		// core.MapView's three. Recorded like the rest — the ID, not the
		// behavior — which is what makes an exported map upgradeable: a page
		// that loads Leaflet, reads the region off data-lat/lng/zoom and wires
		// these three IDs is the live node, built out of the static document.
		// The editing surfaces' selection report (core.OnSelectionChange).
		// Recorded like the rest — the ID, not the behavior — because a static
		// document has no caret whose movement could be reported.
		{"onSelectionChange", "data-onselectionchange"},
		{"onRegionChange", "data-onregionchange"},
		{"onMarkerTap", "data-onmarkertap"},
		{"onMapTap", "data-onmaptap"},
	} {
		if id, ok := node.Props[cb.prop].(string); ok {
			attrs = append(attrs, cb.attr, id)
		}
	}
	// element emits key="value" pairs only, so the bare boolean attribute is
	// written in its spec-blessed long form — the same shape the checked
	// attribute uses below. A disabled control also stops firing the events
	// whose callback IDs were just attached, which is the point: the Go
	// handler stays registered (see Style.Disabled) and the platform, not the
	// app, refuses to dispatch.
	if node.Style != nil && node.Style.Disabled && isFormControl(node.Type) {
		attrs = append(attrs, "disabled", "disabled")
	}
	// core.Focus, rendered the one way a static document can render it.
	//
	// The focus command's two props are a runtime coordination pair — the
	// epoch says *when*, which is a question a snapshot cannot ask — so
	// neither is exported verbatim. What survives the export is the standing
	// instruction the last command left behind, and HTML already has a
	// spelling for it: autofocus puts the cursor in this field when the page
	// loads, which is exactly what "focus" means to a document with no events
	// yet. "blur" and "" export as nothing, because a freshly loaded page has
	// no focus to release.
	//
	// Long form to match the disabled and checked attributes above; element
	// emits key="value" pairs only.
	if getStr(node.Props["focusAction"]) == "focus" && isFormControl(node.Type) {
		attrs = append(attrs, "autofocus", "autofocus")
	}
	// core.UseFocusOrder's keyboard action, in the spelling HTML has for it.
	//
	// Unlike the focus command, this one survives the export intact: the
	// keyboard hint is a standing property of the field ("this key means
	// next"), not a moment in time, which is exactly the kind of thing a
	// static document can carry. A soft keyboard relabels its return key from
	// it; a hardware keyboard ignores it, so the traversal *behavior* travels
	// as data-onsubmit above and this attribute only says what the key reads.
	//
	// "done" for a field that acts on return and does not advance, mirroring
	// Android's ImeAction.Done and SwiftUI's .done. Nothing at all for a field
	// with neither, because enterkeyhint has no empty value — the attribute's
	// absence is how HTML spells "the browser's default return key".
	if isFormControl(node.Type) {
		switch {
		case getStr(node.Props["imeAction"]) == "next":
			attrs = append(attrs, "enterkeyhint", "next")
		case getStr(node.Props["onSubmit"]) != "":
			attrs = append(attrs, "enterkeyhint", "done")
		}
	}

	switch node.Type {
	case "Input":
		b.Input(withLead(attrs, "type", InputTypeFor(node.Type),
			"value", getStr(node.Props["value"]),
			"placeholder", getStr(node.Props["placeholder"]))...).R()
	case "InputPassword":
		b.Input(withLead(attrs, "type", InputTypeFor(node.Type),
			"value", getStr(node.Props["value"]),
			"placeholder", getStr(node.Props["placeholder"]))...).R()
	case "NumericInput":
		b.Input(withLead(attrs, "type", InputTypeFor(node.Type),
			"value", getStr(node.Props["value"]))...).R()
	case "TextArea":
		rows := 3
		if r, ok := node.Props["rows"].(int); ok {
			rows = r
		}
		// TE keeps a value containing "</textarea>" from closing the element.
		b.TextArea(withLead(attrs, "rows", strconv.Itoa(rows))...).TE(getStr(node.Props["value"]))
	case "Checkbox":
		lead := []string{"type", InputTypeFor(node.Type)}
		if v, ok := node.Props["checked"].(bool); ok && v {
			// element emits key="value" pairs only; checked="checked" is the
			// spec-blessed spelling of the bare boolean attribute.
			lead = append(lead, "checked", "checked")
		}
		b.Input(withLead(attrs, lead...)...).R()
	case "Switch":
		// The same element as a Checkbox — type="checkbox" comes from the
		// shared table — plus the one attribute that makes it a switch.
		//
		// `switch` is HTML's own (WHATWG): on a checkbox it asks the browser
		// to draw a track and a thumb instead of a box and a tick. Safari
		// does; most engines do not yet, and those draw the checkbox, which is
		// the same bool in the same state rather than a broken control. The
		// role attribute is what closes the remaining gap — a reader announces
		// a switch everywhere, drawn or not — and it arrives through
		// switchSemantics rather than here, because a role has one slot per
		// element and an author's own Style may have filled it.
		//
		// switch="switch" rather than a bare `switch`, for checked's reason
		// one arm up: element emits key="value" pairs, and repeating the name
		// is the spec-blessed spelling of a bare boolean attribute.
		lead := []string{"type", InputTypeFor(node.Type), "switch", "switch"}
		if v, ok := node.Props["checked"].(bool); ok && v {
			lead = append(lead, "checked", "checked")
		}
		b.Input(withLead(attrs, lead...)...).R()
	case "Slider":
		// A range input carries its bounds as attributes. The numbers are
		// formatted with the shortest round-trip form ('g', -1) so 0.5 stays
		// "0.5" and 30 stays "30" rather than "30.000000"; step is omitted
		// when unset, which leaves the browser's own default (1) — the same
		// as the runtime's continuous default only for integer ranges, but a
		// static export has no drag to be continuous about.
		lead := []string{"type", InputTypeFor(node.Type),
			"min", formatNumber(node.Props["min"]),
			"max", formatNumber(node.Props["max"]),
			"value", formatNumber(node.Props["value"])}
		if step := formatNumber(node.Props["step"]); step != "" && step != "0" {
			lead = append(lead, "step", step)
		}
		b.Input(withLead(attrs, lead...)...).R()
	case "Select":
		renderSelect(b, node, attrs)
	case "MapView":
		// A placeholder box carrying the region it was looking at, as data.
		//
		// An export cannot draw a map: there is no engine in a static
		// document and no tiles to fetch. What it can do is not lose the
		// information — a grey box that does not say where it was pointing is
		// a worse snapshot than one that does — so the region rides out as
		// data attributes, in the same wire form every host reports a region
		// in (core.FormatRegion).
		//
		// That also makes the export upgradeable. A page that loads Leaflet
		// and reads these attributes draws the real map, which is precisely
		// what the WASM runtime does with the same div; the difference between
		// the two targets is a script tag rather than a different document.
		renderContainer(b, node, withLead(attrs, mapDataAttrs(node)...), path)
	case "Marker":
		// Data, not a box: the coordinates and the id, with nothing inside.
		// It gets an element at all because patches are addressed positionally
		// — see the Marker row in tags.
		b.Div(withLead(attrs, markerDataAttrs(node)...)...).R()
	case "Image":
		if src, ok := node.Props["src"].(string); ok {
			b.Img(withLead(attrs, "src", src)...).R()
			return
		}
		// No src: fall through to the default container rendering, matching
		// how unknown/underspecified nodes degrade to a plain div.
		renderContainer(b, node, attrs, path)
	case "Text":
		b.Span(attrs...).TE(getStr(node.Props["content"]))
	case "GridRow":
		renderGridRow(b, node, attrs)
	case "CodeEditor":
		// A box like any other container, plus the line-number gutter ahead of
		// the rows; see codeeditor.go.
		renderCodeEditor(b, node, attrs, path)
	case "RichTextEditor":
		// The document as markup, read-only; see richtext.go.
		renderRichTextEditor(b, node, attrs)
	case "Button":
		b.Button(attrs...).TE(getStr(node.Props["label"]))
	case "CameraView":
		b.Div(attrs...).T("[Camera View]") // placeholder text authored here, not user data
	case "TabView":
		// A box like any other container, plus the bar and the page
		// selection the wire contract asks for; see tabview.go.
		renderTabView(b, node, attrs, path)
	default:
		renderContainer(b, node, attrs, path)
	}
	return
}

// renderContainer renders a generic container tag with the node's children.
// element writes the opening tag when Ele() is called and the closing tag when
// R() runs, so the children rendered in between land inside the element.
func renderContainer(b *element.Builder, node *core.Node, attrs []string, path string) {
	// The tag comes from the shared table rather than a switch here, so the
	// WASM runtime's copy has something to be checked against. The typed
	// element calls in renderNode above (b.Span, b.Button, b.Img, ...) still
	// spell their tags themselves, for readability; TestExportedTagsMatchTable
	// is what holds them to the same table.
	//
	// Fragment and Theme never reach here — renderNode emits their children
	// directly rather than a box.
	e := b.Ele(TagFor(node.Type), attrs...)
	// What this container imposes on each of its children. An overlay is the
	// second caller of the imposed channel after the TabView pages, and it is
	// there for the same reason they are: a child has no idea it is a layer,
	// and only the container above it knows that every child belongs in the
	// same grid cell. See OverlayChildDecl.
	//
	// A Fragment child forwards the declaration to its own children rather
	// than absorbing it (renderNode's transparent branch passes `from`
	// through), which is exactly right — a Fragment has no box, so the layers
	// are its children, and a core.For inside a ZStack overlays what it
	// generated instead of stacking it.
	overlay := IsOverlay(node.Type)
	for i, c := range node.Children {
		child := imposed{}
		if overlay {
			child.decl = OverlayChildDecl
			// core.Style.StackAlign, imposed rather than written by the layer
			// itself — which is what makes the prop inert outside a stack.
			//
			// The value is the *child's* and the decision to honour it is the
			// parent's, so this is the one imposed declaration whose content
			// varies per child; everything else the channel has carried has
			// been one string for the whole sibling set. Appended after
			// grid-area for readability only: the two never name the same
			// property.
			//
			// It has to travel this way rather than through the child's own
			// declaration list because align-self means something else to a
			// flex item. A layer prop that reached every container would
			// re-place a Row's children the moment somebody wrote it on the
			// wrong node, silently and only on the web.
			child.decl += "; " + StackPlacementFor(stackAlignOf(c))
		}
		renderNode(b, c, child, childPath(path, i))
	}
	e.R()
}

// stackAlignOf reads a node's placement, tolerating the nil Style a
// hand-assembled node may have. Every node core builds carries one.
func stackAlignOf(node *core.Node) core.StackAlignment {
	if node == nil || node.Style == nil {
		return core.StackAlignCenter
	}
	return node.Style.StackAlign
}

// childPath is the node path of child i of the node at path — the one place
// the path shape is spelled, so that this exporter, reconcile.Patch's
// TargetIDs and the runtime's data-node-path attributes stay one convention
// rather than three that happen to agree.
func childPath(path string, i int) string {
	return path + "/" + strconv.Itoa(i)
}

// textGridChassis and gridRowChassis are the fixed rules of a core.TextGrid
// and its rows; see renderNode. The line height is explicit so an empty row
// (a GridRow with no runs, which a <div> would collapse to nothing) still
// takes one line, keeping every row on the cell grid it belongs to.
//
// # Where the white space is significant, and where it must not be
//
// The three levels each say something different, and the split is what makes
// a grid survive being pretty-printed:
//
//	grid  white-space:normal   the <pre>'s own default is `pre`, and this
//	                           overrides it, so the newlines and indentation
//	                           the exporter puts *between* the row elements
//	                           are formatting rather than content
//	row   white-space:nowrap   a code line or a terminal row is one line; the
//	                           row must not break between two runs. nowrap
//	                           still collapses, so the line break the
//	                           exporter leaves before each </div> disappears
//	run   white-space:pre      (gridRunStyle) the run's own spaces are the
//	                           only white space in a grid that means anything
//
// Written this way rather than left as one `white-space: pre` on the <pre>
// because ExportHTML re-indents its output for human readers, and inside a
// `white-space: pre` element that indentation is text: every row gained a
// trailing line break and the grid gained a blank line between each pair of
// rows. Confining the significance to the runs makes the grid indifferent to
// how the markup around it is laid out — a property worth having whatever the
// formatter does.
//
// The WASM runtime states the same three rules (its grid chassis in
// styleFromGrMob, and applyGridRuns for the run), so the two DOM targets draw
// a grid identically. It has no formatter of its own, so it does not need
// them; it carries them so that it does not *differ*.
const (
	textGridChassis = "margin:0; line-height:1.2; white-space:normal; overflow-x:auto"
	gridRowChassis  = "min-height:1.2em; white-space:nowrap"
)

// renderGridRow writes one row of a core.TextGrid: a <div> of <span> runs,
// each span carrying the run's white-space rule plus only the declarations
// its run actually set, so a run in the grid's own colours is a span with one
// declaration. The runs are the typed slice core.TextGrid built; a
// hand-assembled node with some other shape exports as an empty row rather
// than a guess.
// renderSelect writes a picker and its options.
//
// The options come from the props rather than from node.Children, which is
// core.Select's contract and the reason this needs a case of its own: every
// other leaf in the switch above is childless in the markup too.
//
// The chosen option is marked with `selected` rather than the element being
// given a value attribute, because <select> has no value attribute — the
// selection lives on the options. This is also why an unmatched value degrades
// the way a browser would anyway: nothing is marked, and the browser shows the
// first option, which is what a live <select> does with an out-of-list value.
//
// An <optgroup>'s label goes through element's attribute path, for the same
// reason the options' own two halves do (renderSelectOptions): a heading is as
// user-originated as the list it stands over.
//
// # Groups are runs, and the decomposition is core's
//
// core.SelectOption.Group makes consecutive options with the same heading one
// <optgroup>, in the order they were written — see the field for why a gather
// would be the wrong shape. Which options form which run is core's
// SelectMenuSections, not this function's: the same split has to be made by
// four renderers, and the one piece of bookkeeping it needs (a run is closed
// by the *next* option naming a different heading, so the last run has to be
// flushed explicitly) is exactly the piece that went untested here for a
// release. What is left below is the markup — an <optgroup> per section that
// names one, the options written straight into the <select> for a section
// that does not.
func renderSelect(b *element.Builder, node *core.Node, attrs []string) {
	value := getStr(node.Props["value"])
	e := b.Ele("select", attrs...)
	// The wire shape core.Select flattens to; see its doc. A hand-built node
	// carrying something else renders as an empty picker rather than panicking,
	// which is the same degradation an Image with no src gets.
	if opts, ok := node.Props["options"].([]map[string]string); ok {
		for _, section := range core.SelectMenuSections(opts) {
			// The ungrouped run has no wrapper: its options are children of
			// the <select> itself, which is where every option lived before
			// the field existed. A disabled ungrouped run therefore has
			// nowhere to put the attribute, which is why the refusal rides on
			// the options rather than here — see SelectMenuSection.Disabled.
			if section.Heading != "" {
				lead := []string{"label", section.Heading}
				if section.Disabled {
					// The bare boolean attribute, spelled the way every other
					// one here is. <optgroup disabled> greys the heading and
					// refuses the whole run in one attribute; the per-option
					// disabled below is still written, because the two natives
					// have no section-level control and core propagates it for
					// them.
					lead = append(lead, "disabled", "disabled")
				}
				group := b.Ele("optgroup", lead...)
				renderSelectOptions(b, section.Items, value)
				group.R()
				continue
			}
			renderSelectOptions(b, section.Items, value)
		}
	}
	e.R()
}

// renderSelectOptions writes one section's <option> elements.
//
// element escapes both halves: the value goes through the attribute path
// (quote-escaped) and the label through TE (entity-escaped), so an option
// carrying markup cannot re-enter the document as markup. Options are as
// user-originated as any other content here — a country list read from a
// server is the normal case.
func renderSelectOptions(b *element.Builder, items []core.SelectMenuItem, value string) {
	for _, item := range items {
		lead := []string{"value", item.Value}
		if item.Value == value {
			// element emits key="value" pairs only; selected="selected" is
			// the spec-blessed spelling of the bare boolean attribute, as
			// checked="checked" is on a Checkbox.
			lead = append(lead, "selected", "selected")
		}
		// The same spelling, and the same reason. A disabled option is
		// still rendered and still announced — that is the whole point of
		// disabling one rather than omitting it — it simply cannot be
		// chosen.
		if item.Disabled {
			lead = append(lead, "disabled", "disabled")
		}
		b.Ele("option", lead...).TE(item.Label)
	}
}

func renderGridRow(b *element.Builder, node *core.Node, attrs []string) {
	runs, _ := node.Props["runs"].(core.GridRow)
	e := b.Div(attrs...)
	for _, run := range runs {
		span := b.Span("style", gridRunStyle(run))
		// A run made only of white space — an indent, the gap between two
		// coloured tokens, a terminal's blank cells — has to be written as
		// character references. The pretty-printer discards any text node
		// that is nothing but white space (it cannot tell a run's spaces from
		// its own indentation), and a `&#32;` is not white space to it while
		// being exactly a space to the browser.
		//
		// Safe to write unescaped, and only here: the branch is entered only
		// when every rune is white space, and spaceRefs emits nothing but
		// digits inside `&#...;`, so no character that could open a tag or an
		// entity can reach the output through it. Every run with a glyph in
		// it still goes through TE.
		if strings.TrimSpace(run.Text) == "" {
			span.T(spaceRefs(run.Text))
			continue
		}
		span.TE(run.Text)
	}
	e.R()
}

// spaceRefs rewrites a white-space-only string as numeric character
// references, one per rune. Called only from renderGridRow, on text it has
// already established is entirely white space.
func spaceRefs(text string) string {
	var b strings.Builder
	for _, r := range text {
		fmt.Fprintf(&b, "&#%d;", r)
	}
	return b.String()
}

// gridRunStyle is a run's declarations: the white-space rule every run
// carries, then its own colours and attributes. The Grid* attributes map onto
// CSS where CSS has a spelling and onto opacity for dim, which it does not;
// underline and strike share text-decoration and are emitted together.
//
// white-space:pre is unconditional because the run is the only level of a
// grid whose spaces are content — see textGridChassis for the other two, and
// for why the significance is pushed down this far.
func gridRunStyle(run core.GridRun) string {
	decl := "white-space:pre"
	if run.Fg != "" {
		decl = addDecl(decl, "color:"+run.Fg)
	}
	if run.Bg != "" {
		decl = addDecl(decl, "background:"+run.Bg)
	}
	if run.Attr&core.GridBold != 0 {
		decl = addDecl(decl, "font-weight:700")
	}
	if run.Attr&core.GridDim != 0 {
		decl = addDecl(decl, "opacity:0.6")
	}
	if run.Attr&core.GridItalic != 0 {
		decl = addDecl(decl, "font-style:italic")
	}
	var lines []string
	if run.Attr&core.GridUnderline != 0 {
		lines = append(lines, "underline")
	}
	if run.Attr&core.GridStrike != 0 {
		lines = append(lines, "line-through")
	}
	if len(lines) > 0 {
		decl = addDecl(decl, "text-decoration:"+strings.Join(lines, " "))
	}
	return decl
}

// withLead prepends type-specific attribute pairs (type, value, src, ...) ahead
// of the shared style/data attributes, preserving the attribute order the
// previous string-based exporter emitted.
func withLead(attrs []string, lead ...string) []string {
	return append(lead, attrs...)
}

// mapDataAttrs is a MapView's region as attributes a loader can read back.
//
// Three separate numbers rather than one formatted region string, because the
// reader is JavaScript: `Number(el.dataset.lat)` is the whole parse, where a
// combined "lat,lng,zoom" would put a split and three conversions in every
// consumer. The *event* direction is combined (core.FormatRegion) because there
// the channel is one text callback and the parse happens once, in Go.
//
// Shortest round-trip formatting, the same rule formatNumber follows for a
// slider's bounds: 38.7223 stays "38.7223" and a whole degree stays "38".
func mapDataAttrs(node *core.Node) []string {
	attrs := []string{
		"data-lat", formatNumber(node.Props["lat"]),
		"data-lng", formatNumber(node.Props["lng"]),
		"data-zoom", formatNumber(node.Props["zoom"]),
	}
	// Written only when asked, so an ordinary map exports the attributes it
	// always did. The blue dot is the host map's own feature and a static
	// document has no user position to draw, so this is a record of the
	// request rather than a rendering of it.
	if on, ok := node.Props["showUser"].(bool); ok && on {
		attrs = append(attrs, "data-show-user", "true")
	}
	return attrs
}

// markerDataAttrs is one pin's identity and position. The title is omitted
// rather than written empty — a marker with no callout is the common case, and
// an attribute whose value is "" is one a reader has to test for rather than
// look up.
func markerDataAttrs(node *core.Node) []string {
	attrs := make([]string, 0, 8)
	// Omitted when empty, like the title: an unnamed marker is a supported
	// thing to write (see core.Marker) and an attribute whose value is "" is
	// one a reader has to test for rather than look up.
	if id := getStr(node.Props["id"]); id != "" {
		attrs = append(attrs, "data-marker-id", id)
	}
	attrs = append(attrs,
		"data-lat", formatNumber(node.Props["lat"]),
		"data-lng", formatNumber(node.Props["lng"]),
	)
	if title := getStr(node.Props["title"]); title != "" {
		attrs = append(attrs, "data-title", title)
	}
	return attrs
}

func getStr(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// modalChassis is the fixed-overlay look and the visible/backdrop state of a
// Modal node, as one CSS declaration list.
//
// Until this existed the exporter rendered a Modal as a plain div and ignored
// both props, so a *closed* dialog's content was laid out inline in the
// document — the export showed a screen no user would ever see, with the
// modal's body spliced into the middle of it. Every other target honors
// Visible (Renderer.swift and Renderer.kt gate on it, the WASM runtime toggles
// display), which made this the one renderer out of four that disagreed about
// what was on screen.
//
// The declarations are the WASM runtime's chassis restated (createElement's
// Modal branch in grmob-runtime.js): the same fixed inset-0 box, the same
// centered flex column, and the same z-index of 1000, chosen to sit under the
// toast layer's 2000 so a toast confirming a dialog's action is not drawn
// behind the dialog.
//
// display, not visibility: a closed modal must take no space and swallow no
// clicks, which is display:none's meaning and not visibility:hidden's. That is
// the same split styleValue makes for DisplayNone against DisplayHidden.
// spacerChassis is the fixed look of a Spacer: a square void that does not
// give way.
//
// Both axes, not just height. core.Spacer(n) is size x size on both natives
// (Compose Spacer(Modifier.size(n.dp)), SwiftUI Color.clear
// .frame(width:height:)), so a Spacer inside a Row separated its siblings by n
// points on device and by nothing at all in the browser, where a zero-width
// box between two flex items is invisible.
//
// flex-shrink:0 is the other half: a flex item's default is to shrink under
// pressure, and a gap whose whole job is to hold a fixed distance must not be
// the thing that gives way. The natives have fixed frames and no equivalent to
// shrink, so this reproduces their behavior rather than adding to it.
//
// # Why it is a chassis and no longer an early return
//
// These three declarations used to be written by a branch that returned before
// the shared attribute assembly, which made a Spacer the one node type whose
// own Style was dropped entirely — and it was dropped *by* the size, which is
// backwards: everywhere else in this file a type's fixed look goes ahead of
// the author's style so the author still wins (modalChassis says so in as many
// words, and core.ModalNode is the same shape — a node core builds with no
// Style of its own, reachable with one only by hand).
//
// Four other things came back with the move, each of which the WASM runtime
// had been doing all along for the same node: a Spacer's accessibility
// attributes, its callback IDs, its children, and its own style declarations.
// core.Spacer(n) builds none of them, so on every tree core produces the
// output is byte-for-byte what the early return emitted; a hand-assembled node
// is the case that changed, and it changed toward what the other DOM renderer
// already did.
//
// An empty string when there is no int size, which is the case that already
// fell through to the shared path before the branch moved into it.
func spacerChassis(props map[string]any) string {
	size, ok := props["size"].(int)
	if !ok {
		return ""
	}
	return fmt.Sprintf("width:%dpx; height:%dpx; flex-shrink:0", size, size)
}

func modalChassis(props map[string]any) string {
	display := "none"
	if v, ok := props["visible"].(bool); ok && v {
		// flex, not block: the overlay centers its content.
		display = "flex"
	}
	decls := ""
	for _, d := range modalChassisDecls {
		decls = addDecl(decls, d[0]+":"+d[1])
	}
	// display sits between the box and the flex rules in the written order,
	// but CSS is not positional and the two DOM targets differ on where it
	// comes from at all — see ModalChassis — so it is appended rather than
	// woven into the table.
	decls = addDecl(decls, "display:"+display)
	// The scrim. Absent rather than transparent when the prop is missing: a
	// hand-built ModalNode may omit it, and core.Modal always supplies one.
	if backdrop := getStr(props["backdrop"]); backdrop != "" {
		decls = addDecl(decls, "background:"+backdrop)
	}
	return decls
}

// The property/value pairs of the Modal chassis that are the same on every
// render — everything but display (the open/closed state, which comes from the
// visible prop) and background (the scrim, which comes from backdrop).
//
// A table rather than a string literal because the WASM runtime states the
// same nine declarations, and two copies of one rule drift silently. Ordered
// pairs rather than a map so the declaration list this builds is stable, which
// is what keeps the exporter's golden output from depending on map iteration.
var modalChassisDecls = [][2]string{
	{"position", "fixed"},
	{"top", "0"},
	{"left", "0"},
	{"right", "0"},
	{"bottom", "0"},
	{"flex-direction", "column"},
	{"align-items", "center"},
	{"justify-content", "center"},
	// Under the toast layer's 2000, so a toast confirming a dialog's action is
	// not drawn behind the dialog.
	{"z-index", "1000"},
}

// ModalChassis returns the fixed declarations of a Modal's overlay look, for
// the WASM runtime conformance test.
//
// The runtime states the same set in styleFromGrMob, spelled as CSSOM property
// names with a `||` per line so an author's Style wins — which is what this
// exporter gets from the cascade by writing the chassis first. The two are
// compared by TestRuntimeModalChassisMatchesGo.
//
// display and background are deliberately not in it. Both are prop-driven, and
// the two targets get them by different routes: this exporter writes the whole
// declaration list at once from props it can see, while the runtime's style
// pass never sees a prop and abstains from display entirely, leaving it to the
// visible prop's own path.
//
// A copy, not the slice itself, for the reason StackAxes returns one: a
// package-level slice is reachable and writable by any importer.
func ModalChassis() [][2]string {
	out := make([][2]string, len(modalChassisDecls))
	copy(out, modalChassisDecls)
	return out
}

// accessibilityAttrs maps core.Style's semantics fields onto the ARIA
// attributes that mean the same thing, as name/value pairs ready to append to
// an attribute slice.
//
// Both natives have read these since they existed (Compose contentDescription
// / clearAndSetSemantics, SwiftUI accessibilityLabel / accessibilityHint /
// accessibilityHidden), and the two web targets read none of them — so a
// components.Separator marked AccessibilityHidden was correctly skipped by
// TalkBack and VoiceOver and announced as a stray element by every screen
// reader on the web.
//
// aria-hidden wins alone. It prunes the element and its subtree from the
// accessibility tree, which makes a name or a description on the same node
// contradictory rather than additive; Compose's clearAndSetSemantics branch
// and SwiftUI's accessibilityHidden branch make the same exclusive choice.
//
// The role maps to the `role` attribute, which is the whole of the mapping:
// core.Role's values are ARIA's own spellings, chosen so that the two DOM
// targets need no table (see core/role.go). It is emitted verbatim even where
// the element already implies it — a Button carrying core.RoleButton exports
// as <button role="button"> — because suppressing the redundant case would
// mean this function knowing the tag table, and a redundant role is inert
// while a missing one is not. A node that names itself and says nothing about
// what it is gets one supplied; see ariaRole.
//
// roleImposed says the node's container has already written a role onto this
// element (the TabView panel wiring is the only one that does), in which case
// this writes none of its own. An attribute has one slot and a browser keeps
// the first value it parses, so a second role= would not be additive — it
// would decide the element's role by document order. See imposed.
//
// An id and an aria-controls come from core.Style.AccessibilityID and
// core.Style.AccessibilityControls, verbatim in both directions. They are the
// vocabulary's only two references, and nothing here checks that the target of
// one exists: an export is a snapshot of one tree and has no index of the
// document it will become part of. A dangling IDREF is inert, which is the
// same trade the hint makes below.
//
// A heading's level maps to aria-level, which is the attribute form of the
// same question and is scoped to the roles ARIA defines it for — see
// headingLevel below.
//
// A selected state maps to aria-selected or aria-pressed, and which of the
// two is again the role's decision — see ariaSelected below, which is the
// same switch running in the other direction.
//
// An expanded state maps to aria-expanded, guarded by a *third* role list that
// is neither of the other two — see ariaExpanded, which is where the three
// lists are set against each other.
//
// A Modal gets role="dialog" and aria-modal="true" from its node type rather
// than from a Style, which is modalSemantics' subject; a Switch gets
// role="switch" the same way. selfRoleSemantics is the seam both arrive
// through, and htmlout.ownRoles is the table of which types do this.
//
// The hint maps to aria-description rather than aria-describedby: the latter
// takes an ID reference, and a static export has no stable IDs to point at
// (nor a place to hang the referenced text). aria-description is the
// attribute form of the same idea. Its support is thinner than the rest of
// ARIA — it is the newest of the three — so it is the one place here where
// the export states the intent ahead of universal support, on the same
// reasoning as enterkeyhint above: the alternative is dropping the author's
// hint entirely.
func accessibilityAttrs(s *core.Style, nodeType string, roleImposed bool) []string {
	// A Modal is the node type whose semantics do not come from a Style at
	// all: core.ModalNode has no Style field, so `s` is nil for every dialog
	// core.Modal builds, and the role would have nowhere to come from if this
	// function only read styles.
	//
	// A Switch is the other self-roling type and answers the nil case the same
	// way, which is the point of asking CarriesOwnRole here rather than
	// comparing against "Modal" twice: a hand-built node with no Style is
	// still the control its type says it is, and a role that appeared only
	// when a caller happened to style the node would be missing from exactly
	// the trees nobody wrote a Style for.
	if s == nil {
		if CarriesOwnRole(nodeType) {
			return selfRoleSemantics(nodeType, core.RoleNone)
		}
		return nil
	}
	if s.AccessibilityHidden {
		// Hidden wins over the Modal semantics too. An overlay pruned from the
		// accessibility tree has no element for role="dialog" to describe, and
		// aria-modal on a hidden node would claim the rest of the document is
		// inert behind something a reader cannot reach.
		return []string{"aria-hidden", "true"}
	}
	attrs := make([]string, 0, 16)
	switch {
	case CarriesOwnRole(nodeType):
		attrs = append(attrs, selfRoleSemantics(nodeType, s.AccessibilityRole)...)
	case roleImposed:
		// The container owns the slot; see the doc above.
	default:
		if role := ariaRole(s, nodeType); role != "" {
			attrs = append(attrs, "role", role)
		}
	}
	if level := ariaLevel(s); level != "" {
		attrs = append(attrs, "aria-level", level)
	}
	// Which way a composite runs, for the three roles ARIA defines the
	// attribute on. Read off the author's own role rather than the effective
	// one above: the two values this function can supply where the author
	// stated nothing — `group` from ariaRole and `dialog` from modalSemantics
	// — are not oriented roles, so there is nothing the fallback could add
	// here. See AriaOrientationFor for why the axis is the answer.
	if o := AriaOrientationFor(s.AccessibilityRole, nodeType, s.FlexDirection); o != "" {
		attrs = append(attrs, "aria-orientation", o)
	}
	if name, value := ariaSelected(s, nodeType); name != "" {
		attrs = append(attrs, name, value)
	}
	if expanded := ariaExpanded(s, nodeType); expanded != "" {
		attrs = append(attrs, "aria-expanded", expanded)
	}
	attrs = append(attrs, ariaValue(s)...)
	if s.AccessibilityID != "" {
		attrs = append(attrs, "id", s.AccessibilityID)
	}
	if s.AccessibilityControls != "" {
		attrs = append(attrs, "aria-controls", s.AccessibilityControls)
	}
	if s.AccessibilityLabel != "" {
		attrs = append(attrs, "aria-label", s.AccessibilityLabel)
	}
	if s.AccessibilityHint != "" {
		attrs = append(attrs, "aria-description", s.AccessibilityHint)
	}
	return attrs
}

// imposesRole reports whether a parent's imposed attribute list already carries
// a role for this element. The list is element's flat name/value form, so the
// names sit at the even indices.
//
// Derived from the list rather than declared beside it, so it cannot fall out
// of step with what tabPanelAttrs actually writes.
func imposesRole(attrs []string) bool {
	for i := 0; i+1 < len(attrs); i += 2 {
		if attrs[i] == "role" {
			return true
		}
	}
	return false
}

// ariaRole is the value of the role attribute for one node: what the author
// said, or — when they said nothing and the element would otherwise be unable
// to carry the name they gave it — core.RoleGroup.
//
// # The silence the fallback closes
//
// Every layout node here exports as a <div> or a <span>, and both tags have
// the implicit ARIA role `generic`. ARIA prohibits an accessible name on
// `generic`, and browsers enforce that by pruning the name out of the
// accessibility tree — so a core.Box carrying an AccessibilityLabel wrote a
// correct-looking aria-label that no screen reader on either web target
// announced, while VoiceOver and TalkBack read it out perfectly (a SwiftUI
// accessibilityLabel and a Compose contentDescription are honoured on any
// node). Two targets silent and two fine is what let it ship.
//
// `group` is the smallest role that makes the name legal: it is nameable, it
// is not a landmark (so a named row does not join the list of regions a reader
// jumps between), it requires no particular children, and it does not make the
// ones it has presentational. It says these things belong together and this is
// what they are called, and nothing more — which is what lets it be supplied
// to a container nothing has looked inside. See core.RoleGroup for the
// candidates that were turned down.
//
// # Three guards, and what each one is protecting
//
//	no name          the fallback exists to rescue a name. A roleless,
//	                 nameless container is a div, which is what it should be.
//	a roled tag      only genericTags may be given a role — writing one onto a
//	                 <button>, an <img> or an <input> would *replace* the role
//	                 the browser already gives it. Those tags can carry a name
//	                 without help, which is why they need no rescue.
//	a self-roling
//	type             a Modal is a dialog by virtue of being a Modal
//	                 (modalSemantics), and dialog is nameable too.
//
// The fallback is silent because it cannot make anything worse: before it the
// name was invalid ARIA that was dropped, and after it the name is valid ARIA
// that is announced. The one claim it could disturb is a structural
// container's — a role="list" says its children are listitems — but a generic
// div inside one was never a listitem either, so a group there is the same
// foreign child it already was, one attribute louder. See core/role.go's
// "A structural role owns what is inside it".
//
// grmob-runtime.js restates this as ariaRole; TestRuntimeSuppliesTheGroupRole
// in wasm/verify holds the two together.
func ariaRole(s *core.Style, nodeType string) string {
	if s.AccessibilityRole != core.RoleNone {
		return string(s.AccessibilityRole)
	}
	if s.AccessibilityLabel == "" || CarriesOwnRole(nodeType) || !IsGenericTag(TagFor(nodeType)) {
		return ""
	}
	return string(core.RoleGroup)
}

// selfRoleSemantics is the one seam the self-roling node types arrive through:
// the types in htmlout.ownRoles, whose role comes from what they *are* rather
// than from a core.Style.
//
// It exists because there are two of them now. While Modal was alone the
// question and the answer were one comparison in one place; the second entry
// (core.Switch) is what makes the shape worth naming, because the failure mode
// is a duplicate role= on one element and an attribute has one slot — a
// browser keeps the first value it parses, so two writers would settle an
// element's role by source order.
//
// An author's own role wins, which is Modal's rule generalised rather than a
// new one: a caller who wrote a Style saying what this node is means it, and
// nothing here outranks them. The aria-modal that rides along for a dialog is
// Modal's alone — it is the one fact in this file that no Role could express,
// since a Modal is the only node in the framework that knows the rest of the
// screen is inert behind it.
func selfRoleSemantics(nodeType string, authored core.Role) []string {
	if nodeType == "Modal" {
		return modalSemantics(authored)
	}
	role := string(authored)
	if role == "" {
		role = OwnRoleFor(nodeType)
	}
	return []string{"role", role}
}

// modalSemantics is the accessibility half of the Modal chassis: the pair of
// attributes that turn a fixed-position div into a dialog.
//
// It is emitted by node type rather than asked for, the way modalChassis's CSS
// is, and for the same reason — core.ModalNode has no Style to carry either
// one. The natives need no equivalent: a SwiftUI sheet and a Compose Dialog
// are dialogs to VoiceOver and TalkBack already, which left the two DOM
// targets as the only ones where an overlay announced as an unnamed group.
// See core/role.go's "Roles a node type carries for itself" for why this is
// not a RoleDialog constant.
//
// aria-modal is unconditional because it is not expressible through core.Role
// at all — it is a second attribute, and a Modal is the only node in the
// framework that knows the rest of the screen is inert behind it. A *closed*
// modal never reaches a reader to be wrong about: modalChassis gives it
// display:none, which takes it and its subtree out of the accessibility tree.
//
// An author's own role wins, on the same precedent the chassis sets for style
// (its declarations go first so the node's own style outranks them): a
// hand-built Modal node that says role="alertdialog" — a value core.Role does
// not carry but a caller's Style could still be given via some future
// vocabulary — means it, and this has no business overruling it.
func modalSemantics(authored core.Role) []string {
	role := string(authored)
	if role == "" {
		role = "dialog"
	}
	return []string{"role", role, "aria-modal", "true"}
}

// ariaLevel renders whichever of core.Style's two level fields the node's role
// calls for, or "" when there is nothing valid to write.
//
// # One attribute, two fields, and why the switch is the point
//
// ARIA defines aria-level for exactly three roles — heading, listitem and row
// — and core carries the tier of a heading and the depth of a collection item
// as separate ints, because the two are validated differently (see
// Style.AccessibilityNestingLevel for the argument). They meet again here, at
// the single attribute both become.
//
// Writing that as two functions, each guarding on its own roles, would leave
// the caller holding an attribute slot two writers could reach. A switch on
// the role makes the exclusion structural instead: a node has one role, the
// arms are disjoint, and there is no arrangement of the two fields that
// produces two values for one attribute. Setting both fields is not an error
// and needs no rule of its own — whichever the role does not name is simply
// not read.
//
// The role guard itself is ARIA's own scoping. A level on any other role
// describes the depth of something that has no depth, which is why a
// DataTable's column headers take the role and no level: columnheader is
// pointedly not among the three.
//
// # Both ranges drop rather than clamp
//
// A heading stops at 6 because that is as far as HTML's h1-h6 and SwiftUI's
// .h1-.h6 go; rewriting a 7 into a 6 would export a structure the caller never
// described. A nesting depth has no upper bound in ARIA ("an integer greater
// than or equal to 1") and none here, so only the zero value and negatives are
// dropped — capping it would flatten a legitimately deep tree, which is the
// same lie in the other direction.
func ariaLevel(s *core.Style) string {
	switch s.AccessibilityRole {
	case core.RoleHeading:
		if s.AccessibilityHeadingLevel < 1 || s.AccessibilityHeadingLevel > 6 {
			return ""
		}
		return strconv.Itoa(s.AccessibilityHeadingLevel)
	case core.RoleListItem, core.RoleRow:
		if s.AccessibilityNestingLevel < 1 {
			return ""
		}
		return strconv.Itoa(s.AccessibilityNestingLevel)
	}
	return ""
}

// ariaSelected renders core.Style.AccessibilitySelected as the attribute the
// node's role calls for, as a name/value pair, or two empty strings when
// there is nothing valid to write.
//
// # One field, two attributes — ariaLevel in a mirror
//
// ariaLevel above resolves two Go fields onto one attribute by switching on
// the role. This is the same switch answering the opposite question: one Go
// field onto two attributes, because ARIA has two words for "on" and they are
// not synonyms.
//
//	aria-selected   one of a set — a tab among tabs, a row among rows.
//	                Choosing one unchooses the others.
//	aria-pressed    a toggle that answers only for itself.
//
// A filter chip is pressed; a tab is selected. Saying the wrong one announces
// the control as a member of a set that does not exist, which is worse than
// saying nothing — the same standard the structural roles are held to.
//
// # The role list is ARIA's own scoping, not a shortlist
//
// aria-selected is defined for gridcell, option, row, tab, columnheader and
// rowheader; of those, core.Role carries option, tab, row and columnheader.
// The one near miss is worth naming because it looks like it belongs:
//
//	cell       is not gridcell. A table cell is not selectable; a grid cell
//	           in an interactive grid is, and core.Role has no grid.
//
// listitem was a second near miss until core.RoleOption existed, and the two
// are still not interchangeable: a list item is *content* and an option is a
// *control in a listbox*, so a row that wants to announce a selection has to
// take the option role and give up the listitem one — along with aria-level,
// which ARIA defines for listitem and not for option. components.ListRow is
// where that trade is made and its Selectable field is where it is written
// down.
//
// aria-pressed is defined for button alone. A core.Button gets it without a
// role because the node type already is one — the same rule that gives a
// core.Modal its dialog role — which is why this takes the node type beside
// the style. That case is not an optimisation: components.Chip renders as a
// core.Button with no role set, so without it the widget that most wants this
// attribute would be the one node that could not have it.
//
// Everything else writes nothing. ARIA does not define either attribute for a
// generic element, so a state on an unroled Box is dropped by the reader
// rather than announced, and writing it anyway would put invalid ARIA in the
// document and change nothing a user hears.
//
// A *name* on a generic element is the same failure and is no longer left to
// fail: ariaRole supplies core.RoleGroup so the name has something legal to
// sit on. A state gets no equivalent rescue, and the asymmetry is deliberate.
// `group` fits any container, so supplying it invents nothing; there is no
// role that carries a selection and fits any container — the four that do are
// option, tab, row and columnheader, and choosing between them would be this
// function deciding what a node is.
func ariaSelected(s *core.Style, nodeType string) (string, string) {
	if s.AccessibilitySelected == core.SelectedUnset {
		return "", ""
	}
	value := string(s.AccessibilitySelected)
	switch s.AccessibilityRole {
	case core.RoleOption, core.RoleTab, core.RoleRow, core.RoleColumnHeader:
		return "aria-selected", value
	case core.RoleButton:
		return "aria-pressed", value
	case core.RoleNone:
		// No role of its own: the node type is the only thing left that can
		// say what this is, and <button> is the one that carries a state.
		if nodeType == "Button" {
			return "aria-pressed", value
		}
	}
	return "", ""
}

// ariaExpanded renders core.Style.AccessibilityExpanded as the aria-expanded
// value, or "" when there is nothing valid to write.
//
// # The third state field, and the simplest of the three
//
// ariaLevel resolves two Go fields onto one attribute and ariaSelected
// resolves one field onto two. This is one onto one: the value is ARIA's own
// spelling and goes out verbatim. All the work is in the guard.
//
// # The role list is ARIA's, and it is not ariaSelected's
//
// aria-expanded is defined for application, button, checkbox, combobox,
// gridcell, link, listbox, menuitem, row, rowheader, tab and treeitem, and
// inherits into columnheader, menuitemcheckbox, menuitemradio and switch. Of
// those, core.Role carries button, link, listbox, row, tab and columnheader.
//
// The overlap with ariaSelected's list is partial in both directions, which is
// the fact worth stating because the two guards look like they should be one:
//
//	option     takes aria-selected and *not* aria-expanded. An option is a
//	           leaf choice; the thing that expands is the listbox around it.
//	link       and listbox take aria-expanded and neither selection
//	           attribute — a link that discloses a section, and the popup half
//	           of a combobox.
//	cell       is not gridcell, the same near miss ariaSelected names.
//
// So a shared guard would be wrong at four roles, which is one of the two
// reasons core.ExpandedState is a type of its own rather than SelectedState
// reused.
//
// A core.Button gets it with no role at all, on the rule that gives a Modal
// its dialog role: the node type already is a button. That is load-bearing
// rather than convenient — ARIA's disclosure pattern *is* a button, so the
// element that most wants this attribute is exactly the one that carries no
// core.Role.
//
// Everything else writes nothing, and there is no RoleGroup-shaped rescue.
// ariaRole supplies `group` to a named container so its name has something
// legal to sit on; `group` is not among the roles above, so there is no value
// that both fits any container and carries a disclosure. A widget that wants
// this attribute has to *be* a control, which is what components.Accordion's
// header row became when it adopted it.
func ariaExpanded(s *core.Style, nodeType string) string {
	if s.AccessibilityExpanded == core.ExpandedUnset {
		return ""
	}
	value := string(s.AccessibilityExpanded)
	switch s.AccessibilityRole {
	case core.RoleButton, core.RoleLink, core.RoleListBox, core.RoleRow, core.RoleColumnHeader, core.RoleTab:
		return value
	case core.RoleNone:
		// No role of its own: the node type is the only thing left that can
		// say what this is, and <button> is the one that discloses.
		if nodeType == "Button" {
			return value
		}
	}
	return ""
}

// isFormControl reports whether the node exports as an HTML element that
// accepts the disabled attribute. Everything else is a div or a span, where
// disabled is not a valid attribute and would simply be ignored.
func isFormControl(nodeType string) bool {
	switch nodeType {
	case "Button", "Input", "InputPassword", "NumericInput", "TextArea", "Checkbox", "Switch", "Slider", "Select":
		return true
	}
	return false
}

// objectFitDecl wraps the shared table's value in the CSS declaration
// styleValue's list is built from. The mapping itself lives in objectFits
// (objectfit.go), which the WASM runtime's copy is checked against; only the
// "object-fit:" prefix is this function's own, because the runtime assigns the
// value to a property and never spells the declaration.
//
// An unset (or unrecognized) mode yields "", which addDecl drops — the
// browser's own object-fit default is `fill`, but an <img> with no explicit
// size is laid out at its intrinsic ratio either way, which is what a
// mode-less Image has always exported as.
func objectFitDecl(mode string) string {
	fit := ObjectFitFor(mode)
	if fit == "" {
		return ""
	}
	return "object-fit:" + fit
}

// addDecl appends one "prop:value" declaration to a declaration list, joining
// with the same "; " separator styleValue uses and tolerating either side
// being empty.
func addDecl(list, decl string) string {
	switch {
	case decl == "":
		return list
	case list == "":
		return decl
	}
	return list + "; " + decl
}

// styleValue serializes the subset of Style the HTML exporter understands into
// a CSS declaration list ("" when nothing is set). The caller places it in a
// style attribute; element handles the attribute-value escaping.
//
// nodeType is needed for two things Style alone cannot answer: which axis Gap
// spaces along (CSS gap only has meaning on a flex/grid container, and the
// main axis is the node's own stacking direction), and whether the node is a
// stack container at all — see stackAxes.
//
// A nil Style is treated as an empty one rather than short-circuited, because
// a stack container has a declaration list even with no Style: the whole
// point of stackAxes is that the stacking is not something the author has to
// ask for. Every other branch below reads the zero value and emits nothing,
// so a non-container with no Style still returns "".
func styleValue(s *core.Style, nodeType string) string {
	if s == nil {
		s = &core.Style{}
	}
	styles := []string{}
	if s.TextColor != "" {
		styles = append(styles, fmt.Sprintf("color:%s", s.TextColor))
	}
	if s.Background != "" {
		styles = append(styles, fmt.Sprintf("background:%s", s.Background))
	}
	if s.FontSize != 0 {
		styles = append(styles, fmt.Sprintf("font-size:%gpx", s.FontSize))
	}
	// core.Weight's values (Light 200, Normal 400, Bold 700) are literal CSS
	// font-weight numbers, so the int crosses unconverted. Both natives have
	// always honored the field; until this line the DOM targets dropped it,
	// so core.Bold rendered as regular text on the web. The WASM runtime
	// gained the same emission in styleFromGrMob at the same time.
	if s.FontWeight != 0 {
		styles = append(styles, fmt.Sprintf("font-weight:%d", s.FontWeight))
	}
	// LineHeight is an absolute line box height in px, not a CSS unitless
	// multiplier — that is what the field means on the natives (Compose takes
	// `lineHeight = n.sp`, SwiftUI derives a lineSpacing of n minus the font
	// size), so the unit has to be written or the same number would mean
	// "n times the font size" here and "n points" there.
	if s.LineHeight != 0 {
		styles = append(styles, fmt.Sprintf("line-height:%dpx", s.LineHeight))
	}
	// Emitted verbatim: core's dimension strings ("40px", "45%", "auto") are
	// already CSS lengths, which is where the format came from. The native
	// renderers parse the same strings back into Compose modifiers and
	// SwiftUI frames.
	//
	// Without these, every widget that sizes itself — a 1px Separator, an
	// Avatar's disc, a ProgressBar's fill — exported as a zero-height or
	// full-width box, so the HTML target silently disagreed with both natives
	// about the layout.
	if s.Width != "" {
		styles = append(styles, fmt.Sprintf("width:%s", s.Width))
	}
	if s.Height != "" {
		styles = append(styles, fmt.Sprintf("height:%s", s.Height))
	}
	// Was an inline switch with three arms; it is a table lookup now because
	// the WASM runtime needs the same mapping and had none at all. See
	// htmlout/textalign.go for what the two used to disagree about.
	if decl := textAlignDecl(string(s.Align)); decl != "" {
		styles = append(styles, decl)
	}
	// Flex container properties. How these interact with Style.Display is
	// resolved where Display is emitted, below; the short version is that a
	// node these props turn into a flex container stays one.
	//
	// The native renderers implement these directly — a Compose Row/Column or
	// a SwiftUI HStack/VStack is inherently a stack, so Gap becomes
	// Arrangement.spacedBy / stack spacing and JustifyContent becomes the
	// arrangement. HTML has no such default: a plain <div> is block flow and
	// ignores gap, justify-content and align-items entirely, so the container
	// must be made flex for any of them to do anything.
	//
	// A stack container is flex whether or not this Style asks for it, and
	// every other node type becomes one only by setting one of these props.
	// stackAxes is the table that draws that line, along with the axis each
	// stack uses; the "" it returns for a non-container is what leaves a Text
	// or a Button carrying a stray container prop in block flow unless it
	// really asked otherwise.
	//
	// The axis: the node's own stacking direction, overridden by an explicit
	// FlexDirection. "column" is the fallback for a non-container, which is
	// reachable — a node outside the table that sets Gap still needs an axis
	// to space along, and vertical is what this exporter has always used.
	stackAxis := StackAxisFor(nodeType)
	dir := stackAxis
	if dir == "" {
		dir = "column"
	}
	if s.FlexDirection != "" {
		dir = string(s.FlexDirection)
	}
	// The effective cross-axis value: AlignItems, else the Align fallback the
	// natives have always read (crossAxisValue in Renderer.swift). The gate is
	// twofold — the node type must be one of the vertical-stacking containers
	// (see alignFallbackAxes for why a type table and not a "not Row" test),
	// and the direction must not have been flipped to a row by an explicit
	// FlexDirection, because the fallback applies to a horizontal cross axis
	// only, on every target. The prefix test rather than equality is for
	// "column-reverse", whose cross axis is horizontal all the same.
	alignItems := string(s.AlignItems)
	if alignItems == "" && strings.HasPrefix(dir, "column") {
		if AlignFallbackAxisFor(nodeType) != "" {
			alignItems = CrossAxisAlignFor(string(s.Align))
		}
	}
	// The two gap longhands promote a box exactly as Gap does: `gap` IS
	// `row-gap` plus `column-gap`, so a node that sets one of them has asked
	// for the same spacing by another name and needs the same flex container
	// to get it. They were left out while both were web-only decorations;
	// once the natives learned to read them as their stacks' spacing,
	// omitting them here meant core.RowGap(8) on a Column spaced the children
	// on a phone and emitted an inert `row-gap` into a block-flow div.
	// The z-stack is a container too, and not a flex one. It short-circuits
	// the whole block below rather than sitting beside it: a ZStack that
	// carried a Gap or an AlignItems would otherwise be turned into a flex
	// container by the test underneath and stop overlaying its children
	// entirely, which is a silent and total loss of the thing the node type
	// exists for. The props are simply inert on an overlay, as they are on a
	// Text — there is one cell and nothing to space along.
	isOverlay := IsOverlay(nodeType)
	isFlex := !isOverlay &&
		(stackAxis != "" ||
			s.Gap != 0 || s.RowGap != 0 || s.ColumnGap != 0 ||
			s.JustifyContent != "" || alignItems != "" || s.FlexDirection != "")
	if isOverlay {
		// inline-grid for the same reason the flex branch writes inline-flex:
		// an inline-level node that is also a grid needs both halves, and
		// "display" has one slot.
		if s.Display == core.DisplayInline {
			styles = append(styles, strings.Replace(OverlayChassis, "display:grid", "display:inline-grid", 1))
		} else {
			styles = append(styles, OverlayChassis)
		}
	} else if isFlex {
		// inline-flex is the one CSS spelling that keeps both halves when a
		// Display: inline node is also a flex container: the inline level the
		// author asked for and the flex layout its container props require.
		display := "flex"
		if s.Display == core.DisplayInline {
			display = "inline-flex"
		}
		styles = append(styles, "display:"+display, fmt.Sprintf("flex-direction:%s", dir))
		if s.Gap != 0 {
			styles = append(styles, fmt.Sprintf("gap:%gpx", s.Gap))
		}
		if s.JustifyContent != "" {
			styles = append(styles, fmt.Sprintf("justify-content:%s", s.JustifyContent))
		}
		if alignItems != "" {
			styles = append(styles, fmt.Sprintf("align-items:%s", alignItems))
		}
	}
	// Style.Display, resolved against the flex container above rather than
	// simply emitted last. It used to be emitted last precisely so it would
	// win the browser's last-declaration-wins parse, on the theory that an
	// explicit Display is the author's word — but the merge in containerNode
	// erases who set what, and DefaultTheme's Card style carries Display:
	// block, so every themed Card's own theme was killing the align-items the
	// author asked for (explicitly or through the Align fallback). One target
	// out of four: the natives read Display only to honor "none" (both
	// Renderer.swift and Renderer.kt bail out before any layout), the WASM
	// runtime deliberately emits no Display at all (styleFromGrMob explains
	// why), and this exporter alone let "block" beat the container.
	//
	// So on a flex container:
	//
	//   - "none" still lands after display:flex and wins: hiding beats layout
	//     on every target that reads Display at all.
	//   - "block" is not emitted: a block-level flex container is exactly
	//     display:flex, so the mode's whole meaning is already stated.
	//   - "inline" was folded into the container above as inline-flex.
	//   - "visible"/"hidden" are not CSS display keywords; the browser was
	//     already dropping them as invalid after the flex declaration, so the
	//     dead declaration is simply no longer written.
	//
	// A z-stack is gated identically and for identical reasons — "none" still
	// wins, "block" is already said by display:grid, "inline" was folded into
	// inline-grid — which is why the two containers share one condition rather
	// than growing a second copy of this paragraph.
	//
	// A node that is not a flex container keeps the verbatim emission this
	// exporter has always produced.
	//
	// "visible" and "hidden" have been split off entirely (see below): they
	// are not CSS display keywords, so emitting them here produced a
	// declaration the browser discarded — the mode was stated in Go, written
	// into the document, and had no effect anywhere.
	if isCSSDisplay(s.Display) && ((!isFlex && !isOverlay) || s.Display == core.DisplayNone) {
		styles = append(styles, fmt.Sprintf("display:%s", s.Display))
	}
	// DisplayHidden / DisplayVisible, in the CSS property that actually means
	// what they say. Both natives read the mode this way — Renderer.swift
	// applies .opacity(0) and Renderer.kt an alpha of 0, keeping the node's
	// space and dropping its pixels — and `visibility` is that behavior's CSS
	// spelling. `display:none`, the mode above, is the other one: no pixels
	// AND no space, which is why the two cannot share a property.
	//
	// "visible" is emitted rather than dropped as a no-op: it is the CSS
	// default, but a node nested inside a hidden ancestor inherits hidden, and
	// an explicit DisplayVisible is the only way an author can say "not that
	// one". The natives get this for free (opacity does not inherit).
	switch s.Display {
	case core.DisplayHidden:
		styles = append(styles, "visibility:hidden")
	case core.DisplayVisible:
		styles = append(styles, "visibility:visible")
	}
	// EdgeCSS, not four field reads: core.EdgeInsets also carries the
	// Horizontal/Vertical shorthand pair, which both natives resolve into the
	// unset sides and which this exporter used to drop on the floor. See
	// htmlout/edges.go for the rule and for what it silently cost.
	if s.Padding != (core.EdgeInsets{}) {
		styles = append(styles, "padding:"+EdgeCSS(s.Padding))
	}
	if s.Margin != (core.EdgeInsets{}) {
		styles = append(styles, "margin:"+EdgeCSS(s.Margin))
	}
	// Flex *item* properties, as opposed to the container properties above:
	// they describe how this node behaves inside its parent's flex layout, so
	// they need no display:flex of their own.
	if s.FlexGrow != 0 {
		styles = append(styles, fmt.Sprintf("flex-grow:%g", s.FlexGrow))
	}
	if s.BorderRadius != 0 {
		styles = append(styles, fmt.Sprintf("border-radius:%gpx", s.BorderRadius))
	}
	// Rotation is a paint transform, so it goes out whatever the display mode
	// and needs no companion declaration: transform-origin defaults to the
	// box's centre, which is the one origin core.Rotate offers.
	//
	// %g rather than a rounded form: unlike the shadow arithmetic below, the
	// angle is the caller's own number and is not derived, so there is nothing
	// to round away. A compass fed 123.4 degrees should export 123.4deg — and
	// an unwrapped bearing past 360 stays past 360 here, because the winding
	// is meaningful under a transition (see core.Style.Rotate).
	if s.Rotate != 0 {
		styles = append(styles, fmt.Sprintf("transform:rotate(%gdeg)", s.Rotate))
	}
	// Shadow is a single elevation number on every target — Compose's
	// Modifier.shadow(elevation) and SwiftUI's .shadow(radius:y:) both take
	// one — and CSS box-shadow wants offsets, a blur and a color. The
	// arithmetic here is the SwiftUI mapping restated (grMobShadow in
	// GrMobStyle.swift: blur = elevation/2, y offset = elevation/3), so the
	// three targets that draw a shadow at all draw comparable ones from the
	// same core.Shadow(4).
	//
	// The color is SwiftUI's default shadow black at a third alpha; CSS has no
	// default, so it has to be spelled. Elevation is in px like every other
	// dimension core emits.
	//
	// Rounded to two decimals rather than printed at full float precision: an
	// elevation of 4 divides into 1.3333333333333333, which is noise in a
	// declaration measured in device pixels. The WASM runtime rounds the same
	// way, so the two targets emit the same string for the same elevation.
	if s.Shadow != 0 {
		styles = append(styles, fmt.Sprintf("box-shadow:0 %gpx %gpx rgba(0,0,0,0.33)",
			round2(s.Shadow/3), round2(s.Shadow/2)))
	}
	// Both natives already honor BorderColor/BorderWidth — Compose applies a
	// Modifier.border, SwiftUI a .grMobBorder overlay — so a widget that draws
	// a rule (components.Button's outlined emphasis) had an edge on device and
	// none in the HTML export. Same class of silent disagreement the Width and
	// Height emission above fixed.
	//
	// Both halves are required, matching the natives: Compose skips the border
	// unless borderWidth > 0 && borderColor != null, so a color with no width
	// or a width with no color draws nothing there and must draw nothing here.
	//
	// The else arm is the other half of the same agreement, and it is the one
	// the guard alone got wrong: on three targets "no border in the style"
	// means no border on screen, and on the web it meant "whatever the user
	// agent draws" — which for a <button> is a 2px outset rule that no
	// core.BorderWidth(0) could turn off, because emitting nothing is exactly
	// what leaves the browser in charge. See borderResetTypes in tag.go for
	// the node types this applies to, and for why the two <input> types whose
	// user agent draws the whole control are not among them.
	if s.BorderWidth != 0 && s.BorderColor != "" {
		styles = append(styles, fmt.Sprintf("border:%gpx solid %s", s.BorderWidth, s.BorderColor))
	} else if ResetsUABorder(nodeType) {
		styles = append(styles, "border:none")
	}
	if s.Transition != "" {
		// core.Transition's canonical "<ms>ms <easing>" is valid CSS as-is;
		// "all" scopes it to every animatable property, matching the native
		// renderers' behavior.
		styles = append(styles, fmt.Sprintf("transition:all %s", s.Transition))
	}
	// Style.Animation is a CSS animation shorthand ("bounce 2s infinite"). It
	// is emitted verbatim, and it is the one property here that needs
	// something this exporter does not produce: a matching @keyframes rule.
	// The export writes no stylesheet at all, so the declaration is inert
	// until the document is embedded in a page that defines the keyframes by
	// name. That is still strictly better than dropping it — the name is the
	// author's, and a target that can honor it now receives it. Neither native
	// reads the field.
	if s.Animation != "" {
		styles = append(styles, "animation:"+s.Animation)
	}
	// The remaining CSS-shaped fields of core.Style. Every one of them existed
	// on the struct with a StyleProp constructor and no reader on any of the
	// four targets — declared in Go, dropped everywhere. They are cheap here
	// (a direct property each) and expensive on the natives (Compose and
	// SwiftUI have no direct equivalent for most), so the web pair honors them
	// and the native gap is documented rather than faked.
	//
	// Emitted verbatim for the same reason Width and Height are: core's
	// dimension strings ("40px", "45%", "auto") are already CSS lengths, and
	// the enums (Position, AlignItems, FlexWrap, Overflow, WhiteSpace) hold
	// the CSS keywords themselves.
	if s.MinWidth != "" {
		styles = append(styles, "min-width:"+s.MinWidth)
	}
	if s.MinHeight != "" {
		styles = append(styles, "min-height:"+s.MinHeight)
	}
	if s.MaxWidth != "" {
		styles = append(styles, "max-width:"+s.MaxWidth)
	}
	if s.MaxHeight != "" {
		styles = append(styles, "max-height:"+s.MaxHeight)
	}
	if s.Overflow != "" {
		styles = append(styles, "overflow:"+s.Overflow)
	}
	if s.WhiteSpace != "" {
		styles = append(styles, "white-space:"+s.WhiteSpace)
	}
	// Out-of-flow placement. The offsets are emitted whether or not Position
	// is set, matching how CSS itself treats them: they are inert on a static
	// box rather than an error, and a node can inherit a positioned ancestor's
	// containing block without restating its own Position.
	if s.Position != "" {
		styles = append(styles, "position:"+string(s.Position))
	}
	if s.Top != "" {
		styles = append(styles, "top:"+s.Top)
	}
	if s.Right != "" {
		styles = append(styles, "right:"+s.Right)
	}
	if s.Bottom != "" {
		styles = append(styles, "bottom:"+s.Bottom)
	}
	if s.Left != "" {
		styles = append(styles, "left:"+s.Left)
	}
	if s.ZIndex != 0 {
		styles = append(styles, "z-index:"+strconv.Itoa(s.ZIndex))
	}
	// FlexWrap is not part of the isFlex decision above, and is deliberately
	// not in it: unlike Gap and its two longhands, it asks for nothing on its
	// own — flex-wrap only has an effect once the box is already a flex
	// container, so promoting a box for it alone would change that box's
	// layout to no purpose.
	if s.FlexWrap != "" {
		styles = append(styles, "flex-wrap:"+s.FlexWrap)
	}
	// The axis gaps, emitted after the `gap` shorthand the isFlex block
	// writes so the cascade lets an axis value win over the isotropic one.
	// (grmob-runtime.js reaches the same result the other way round: the
	// CSSOM has no cascade within one assignment pass, so it resolves the two
	// axes in JS and writes only the longhands.)
	if s.RowGap != 0 {
		styles = append(styles, fmt.Sprintf("row-gap:%gpx", s.RowGap))
	}
	if s.ColumnGap != 0 {
		styles = append(styles, fmt.Sprintf("column-gap:%gpx", s.ColumnGap))
	}
	// Flex *item* properties, joining FlexGrow above: they describe how this
	// node behaves inside its parent's layout, so they need no display:flex of
	// their own.
	if s.AlignSelf != "" {
		styles = append(styles, "align-self:"+string(s.AlignSelf))
	}
	if s.FlexBasis != "" {
		styles = append(styles, "flex-basis:"+s.FlexBasis)
	}
	// Through the resolver, not off the field: core.FlexShrink(0) stores
	// core.ShrinkNone because zero already means "unset" for every other
	// number in a Style, and flex-shrink is the one whose CSS initial value is
	// not zero. A guard spelled `!= 0` here — which is what this was — made
	// "do not shrink" write nothing at all.
	if shrink, declared := s.ShrinkFactor(); declared {
		styles = append(styles, fmt.Sprintf("flex-shrink:%g", shrink))
	}
	return strings.Join(styles, "; ")
}

// round2 rounds to two decimal places, the precision a CSS length measured in
// device pixels is meaningful at.
func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// isCSSDisplay reports whether a DisplayMode names an actual CSS display
// keyword. Three of the five do; "visible" and "hidden" name a visibility,
// which styleValue emits through that property instead.
//
// A census rather than a "not those two" test, for the same reason tagForType
// spells out its plain-div rows: a mode added to core.DisplayMode and not
// taught to this exporter shows up as a missing case here rather than as an
// invalid declaration in the output.
func isCSSDisplay(m core.DisplayMode) bool {
	switch m {
	case core.DisplayNone, core.DisplayBlock, core.DisplayInline:
		return true
	}
	return false
}

// formatNumber renders a numeric prop for an attribute value: "" when the
// prop is absent or not a number. Both float64 (what core writes) and int
// (what a hand-built node might carry) are accepted.
func formatNumber(v any) string {
	switch n := v.(type) {
	case float64:
		return strconv.FormatFloat(n, 'g', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(n), 'g', -1, 32)
	case int:
		return strconv.Itoa(n)
	}
	return ""
}

// ariaValue renders core.Style.AccessibilityValue as the aria-value* family,
// as name/value pairs, or nothing when there is nothing valid to write.
//
// # The fourth state field, and the narrowest guard of the four
//
// ariaLevel resolves two Go fields onto one attribute, ariaSelected one field
// onto two, ariaExpanded one onto one. This is one field onto *four*
// attributes, and the reason they travel together rather than as four fields is
// in core.ValueRange: Now, Min and Max are one fact in three parts, and "45" is
// 45% out of ARIA's implicit 0..100 and is step 45 out of 1..50 — the same
// digits describing two different bars.
//
// The role list is ARIA's own scoping and is the shortest one here.
// aria-valuenow, -valuemin and -valuemax are defined for meter, progressbar,
// scrollbar, slider, spinbutton and a focusable separator; core.Role carries
// progressbar and nothing else on that list, so there is one arm. The near miss
// worth naming, because it looks like it belongs:
//
//	Slider     is a node type, not a role. It exports as <input type="range">,
//	           which states value/min/max as real attributes the browser reads
//	           — so an ARIA range on top would be a second claim about one
//	           fact, and the two would disagree the moment either moved.
//
// A stated role with an unstated range is not an omission and is not corrected:
// ARIA spells an *indeterminate* progress bar by leaving aria-valuenow off, so
// a bar that is running with no idea how far is exactly this role and this zero
// value. Which is also why each of the four is written only when it is stated,
// rather than defaulted — supplying a 0 would turn every indeterminate bar into
// one pinned at the start.
//
// # aria-valuetext is guarded with them and reaches further than they do
//
// It is scoped to the same roles, so it is written under the same guard. But
// unlike the numbers it has a mapping on both phones — Compose's
// stateDescription, SwiftUI's accessibilityValue — neither of which asks what
// the node is. That asymmetry is the one a selection already has, and it is
// ARIA's strictness rather than the framework's: each platform says the truest
// thing it can.
//
// grmob-runtime.js restates this as ariaValue and the two must agree;
// TestRuntimeGuardsTheValueTheSameWay holds them together.
func ariaValue(s *core.Style) []string {
	if !s.AccessibilityValue.Stated() {
		return nil
	}
	// A switch with one arm rather than an equality test, for the shape the
	// three functions above have: the guard is a role list, it happens to have
	// one member today, and a second range role arrives as an arm rather than
	// as a rewrite.
	switch s.AccessibilityRole {
	case core.RoleProgressBar:
	default:
		return nil
	}
	v := s.AccessibilityValue
	attrs := make([]string, 0, 8)
	// Each written only when stated. An unstated bound is ARIA's own default
	// (0 and 100), which is what makes a bare Now announce as a percentage.
	for _, pair := range []struct{ name, value string }{
		{"aria-valuenow", v.Now},
		{"aria-valuemin", v.Min},
		{"aria-valuemax", v.Max},
		{"aria-valuetext", v.Text},
	} {
		if pair.value != "" {
			attrs = append(attrs, pair.name, pair.value)
		}
	}
	return attrs
}
