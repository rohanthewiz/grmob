// Package htmlout exports a rendered core.Node tree as a standalone HTML
// document. It is the demo/inspection path (the example apps print its output);
// the WASM runtime does not consume it, so readability is favored over
// compactness.
package htmlout

import (
	"fmt"
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
			renderNode(b, node),
		),
	)
	// Pretty re-indents the compact single-pass output for human readers.
	// Escaped content is inert entities by this point, so re-parsing is safe.
	return b.Pretty()
}

// renderNode writes one node (and, for containers, its subtree) into the
// builder. The return value exists only so calls can sit inline as R()
// arguments, which is how element establishes evaluation order; it is ignored.
func renderNode(b *element.Builder, node *core.Node) (x any) {
	if node == nil {
		return
	}

	// Spacer is pure layout: a fixed-height gap, no children, no other props.
	if node.Type == "Spacer" {
		if size, ok := node.Props["size"].(int); ok {
			b.Div("style", fmt.Sprintf("height:%dpx", size)).R()
			return
		}
	}

	// Grouping nodes are emitted as their children, with no box of their own.
	// Which types those are, why a wrapper for them is a layout bug rather
	// than a redundant element, and why the WASM runtime is the one DOM
	// renderer that cannot do this, are all in transparentTypes (tag.go).
	if IsTransparent(node.Type) {
		for _, child := range node.Children {
			renderNode(b, child)
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
	if node.Style != nil && node.Style.Disabled && !isFormControl(node.Type) {
		// HTML's disabled attribute is only valid on form controls, so a
		// disabled container gets the ARIA state plus the one declaration
		// that actually makes a browser stop routing pointer events to it.
		// Together they are the closest the export gets to what Compose's
		// `enabled = false` and SwiftUI's `.disabled(true)` do natively.
		sv = addDecl(sv, "pointer-events:none")
		attrs = append(attrs, "aria-disabled", "true")
	}
	if sv != "" {
		attrs = append(attrs, "style", sv)
	}
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
	case "Image":
		if src, ok := node.Props["src"].(string); ok {
			b.Img(withLead(attrs, "src", src)...).R()
			return
		}
		// No src: fall through to the default container rendering, matching
		// how unknown/underspecified nodes degrade to a plain div.
		renderContainer(b, node, attrs)
	case "Text":
		b.Span(attrs...).TE(getStr(node.Props["content"]))
	case "Button":
		b.Button(attrs...).TE(getStr(node.Props["label"]))
	case "CameraView":
		b.Div(attrs...).T("[Camera View]") // placeholder text authored here, not user data
	default:
		renderContainer(b, node, attrs)
	}
	return
}

// renderContainer renders a generic container tag with the node's children.
// element writes the opening tag when Ele() is called and the closing tag when
// R() runs, so the children rendered in between land inside the element.
func renderContainer(b *element.Builder, node *core.Node, attrs []string) {
	// The tag comes from the shared table rather than a switch here, so the
	// WASM runtime's copy has something to be checked against. The typed
	// element calls in renderNode above (b.Span, b.Button, b.Img, ...) still
	// spell their tags themselves, for readability; TestExportedTagsMatchTable
	// is what holds them to the same table.
	//
	// Fragment and Theme never reach here — renderNode emits their children
	// directly rather than a box.
	e := b.Ele(TagFor(node.Type), attrs...)
	for _, child := range node.Children {
		renderNode(b, child)
	}
	e.R()
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

// isFormControl reports whether the node exports as an HTML element that
// accepts the disabled attribute. Everything else is a div or a span, where
// disabled is not a valid attribute and would simply be ignored.
func isFormControl(nodeType string) bool {
	switch nodeType {
	case "Button", "Input", "InputPassword", "NumericInput", "TextArea", "Checkbox":
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
// nodeType is needed for Gap: CSS gap only has meaning on a flex/grid
// container, and the main axis it spaces along is the node's own stacking
// direction (Row lays out horizontally, every other container vertically) —
// information that lives in the node type, not in Style.
func styleValue(s *core.Style, nodeType string) string {
	if s == nil {
		return ""
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
	// Only nodes that actually set one of these become flex containers;
	// everything else keeps the block-flow output this exporter has always
	// produced. The main axis defaults to the node's own stacking direction
	// (Row horizontal, every other container vertical) and an explicit
	// FlexDirection overrides it.
	dir := "column"
	if nodeType == "Row" {
		dir = "row"
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
	isFlex := s.Gap != 0 || s.JustifyContent != "" || alignItems != "" || s.FlexDirection != ""
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
	if s.Display != "" && (!isFlex || s.Display == core.DisplayNone) {
		styles = append(styles, fmt.Sprintf("display:%s", s.Display))
	}
	if s.Padding != (core.EdgeInsets{}) {
		styles = append(styles, fmt.Sprintf("padding:%dpx %dpx %dpx %dpx", s.Padding.Top, s.Padding.Right, s.Padding.Bottom, s.Padding.Left))
	}
	if s.Margin != (core.EdgeInsets{}) {
		styles = append(styles, fmt.Sprintf("margin:%dpx %dpx %dpx %dpx", s.Margin.Top, s.Margin.Right, s.Margin.Bottom, s.Margin.Left))
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
	return strings.Join(styles, "; ")
}
