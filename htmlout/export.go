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
// Exactly one caller fills it in: renderTabView, which hides the pages that
// are not selected (decl) and names each page as the panel its tab controls
// (attrs). Both have to arrive this way because a page does not know it is a
// page; only the TabView above it does.
//
//	decl   a CSS declaration list, appended after everything the node itself
//	       declares so it wins the browser's last-one-wins parse — which is
//	       what lets display:none outrank the display:flex a stack container
//	       is given unconditionally
//	attrs  element-style key/value pairs, appended to the node's own attribute
//	       list. Nothing here ever collides with an attribute the node writes
//	       for itself: the panel wiring is id/role/aria-labelledby, and no
//	       core.Style field maps onto any of the three.
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

	// Spacer is pure layout: a fixed square void, no children, no other props.
	//
	// Both axes, not just height. core.Spacer(n) is size x size on both
	// natives (Compose Spacer(Modifier.size(n.dp)), SwiftUI Color.clear
	// .frame(width:height:)), so a Spacer inside a Row separated its siblings
	// by n points on device and by nothing at all in the browser, where a
	// zero-width box between two flex items is invisible.
	//
	// flex-shrink:0 is the other half: a flex item's default is to shrink
	// under pressure, and a gap whose whole job is to hold a fixed distance
	// must not be the thing that gives way. The natives have fixed frames and
	// no equivalent to shrink, so this is what reproduces their behavior
	// rather than adding to it.
	if node.Type == "Spacer" {
		if size, ok := node.Props["size"].(int); ok {
			// What the parent imposed still applies: this branch returns
			// before the shared attribute assembly below, so a Spacer used as
			// a tab page would otherwise be the one node type that could
			// neither be hidden nor named as a panel.
			decls := fmt.Sprintf("width:%dpx; height:%dpx; flex-shrink:0", size, size)
			b.Div(append([]string{"style", addDecl(decls, from.decl)}, from.attrs...)...).R()
			return
		}
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
	if node.Style != nil && node.Style.Disabled && !isFormControl(node.Type) {
		// HTML's disabled attribute is only valid on form controls, so a
		// disabled container gets the ARIA state plus the one declaration
		// that actually makes a browser stop routing pointer events to it.
		// Together they are the closest the export gets to what Compose's
		// `enabled = false` and SwiftUI's `.disabled(true)` do natively.
		sv = addDecl(sv, "pointer-events:none")
		attrs = append(attrs, "aria-disabled", "true")
	}
	// Last, so it outranks the node's own declarations — including the
	// display:flex a stack container is given unconditionally, which is
	// exactly what a hidden tab page has to be talked out of.
	sv = addDecl(sv, from.decl)
	if sv != "" {
		attrs = append(attrs, "style", sv)
	}
	attrs = append(attrs, accessibilityAttrs(node.Style)...)
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
	for i, child := range node.Children {
		renderNode(b, child, imposed{}, childPath(path, i))
	}
	e.R()
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
func modalChassis(props map[string]any) string {
	display := "none"
	if v, ok := props["visible"].(bool); ok && v {
		// flex, not block: the overlay centers its content.
		display = "flex"
	}
	decls := "position:fixed; top:0; left:0; right:0; bottom:0; " +
		"display:" + display + "; flex-direction:column; " +
		"align-items:center; justify-content:center; z-index:1000"
	// The scrim. Absent rather than transparent when the prop is missing: a
	// hand-built ModalNode may omit it, and core.Modal always supplies one.
	if backdrop := getStr(props["backdrop"]); backdrop != "" {
		decls = addDecl(decls, "background:"+backdrop)
	}
	return decls
}

// accessibilityAttrs maps core.Style's three semantics fields onto the ARIA
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
// The hint maps to aria-description rather than aria-describedby: the latter
// takes an ID reference, and a static export has no stable IDs to point at
// (nor a place to hang the referenced text). aria-description is the
// attribute form of the same idea. Its support is thinner than the rest of
// ARIA — it is the newest of the three — so it is the one place here where
// the export states the intent ahead of universal support, on the same
// reasoning as enterkeyhint above: the alternative is dropping the author's
// hint entirely.
func accessibilityAttrs(s *core.Style) []string {
	if s == nil {
		return nil
	}
	if s.AccessibilityHidden {
		return []string{"aria-hidden", "true"}
	}
	attrs := make([]string, 0, 4)
	if s.AccessibilityLabel != "" {
		attrs = append(attrs, "aria-label", s.AccessibilityLabel)
	}
	if s.AccessibilityHint != "" {
		attrs = append(attrs, "aria-description", s.AccessibilityHint)
	}
	return attrs
}

// isFormControl reports whether the node exports as an HTML element that
// accepts the disabled attribute. Everything else is a div or a span, where
// disabled is not a valid attribute and would simply be ignored.
func isFormControl(nodeType string) bool {
	switch nodeType {
	case "Button", "Input", "InputPassword", "NumericInput", "TextArea", "Checkbox", "Slider":
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
	isFlex := stackAxis != "" ||
		s.Gap != 0 || s.RowGap != 0 || s.ColumnGap != 0 ||
		s.JustifyContent != "" || alignItems != "" || s.FlexDirection != ""
	if isFlex {
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
	// A node that is not a flex container keeps the verbatim emission this
	// exporter has always produced.
	//
	// "visible" and "hidden" have been split off entirely (see below): they
	// are not CSS display keywords, so emitting them here produced a
	// declaration the browser discarded — the mode was stated in Go, written
	// into the document, and had no effect anywhere.
	if isCSSDisplay(s.Display) && (!isFlex || s.Display == core.DisplayNone) {
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
	if s.BorderWidth != 0 && s.BorderColor != "" {
		styles = append(styles, fmt.Sprintf("border:%gpx solid %s", s.BorderWidth, s.BorderColor))
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
	if s.FlexShrink != 0 {
		styles = append(styles, fmt.Sprintf("flex-shrink:%g", s.FlexShrink))
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
