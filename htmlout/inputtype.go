package htmlout

// inputTypes is the one authoritative statement of the node type -> HTML
// <input> type table.
//
// Five Go node types share the <input> tag, and an <input> with no type
// attribute is a text box, so this attribute is the only thing that makes a
// checkbox a checkbox rather than a text field. Every renderer that targets
// the DOM therefore needs the same table:
//
//	htmlout (this package)      queries it through InputTypeFor
//	wasm/grmob-runtime.js       restates it in JavaScript (inputTypeFor)
//	wasm/verify/replay_test.mjs restates it again as a conformance check
//
// Only the first can literally share this map — the other two are JavaScript.
// The runtime's copy is pinned to this one by TestRuntimeInputTypesMatchGo in
// wasm/verify, which reads the table out of the runtime source and compares it
// here, so the duplication is checked rather than remembered. The replay
// test's copy is left deliberately independent: a conformance test that read
// the implementation's table would only prove the implementation agrees with
// itself.
//
// Every node type absent from this map has a tag that already says what it is
// (a <span>, a <textarea>, a <button>, a <div>) and gets no type attribute at
// all — which is why the zero value of the lookup, "", is the right answer for
// them rather than an error.
var inputTypes = map[string]string{
	"Input":         "text",
	"InputPassword": "password",
	"NumericInput":  "number",
	"Checkbox":      "checkbox",
	"Slider":        "range",
}

// InputTypeFor returns the HTML <input> type attribute a node type renders as,
// or "" for a node that is not an <input>.
func InputTypeFor(nodeType string) string {
	return inputTypes[nodeType]
}

// InputTypes returns a copy of the whole table, for the callers that must
// enumerate it rather than query it — currently only the WASM runtime
// conformance test, which has to compare table against table and so cannot go
// through InputTypeFor one key at a time.
//
// A copy, not the map itself: a package-level map is reachable and writable by
// any importer, and a table this small is cheaper to copy than to defend.
func InputTypes() map[string]string {
	out := make(map[string]string, len(inputTypes))
	for k, v := range inputTypes {
		out[k] = v
	}
	return out
}
