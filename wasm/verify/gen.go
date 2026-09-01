// Command gen emits patch transcripts for the JavaScript runtime conformance
// harness (see run.sh).
//
// It drives real example apps through render.Manager the way the browser
// drives them — RenderInitial once, then one Dispatch per user event — and
// records the initial tree, every patch batch in order, and the final tree.
// The JS harness mounts the initial tree through the real
// wasm/grmob-runtime.js, applies the batches, walks the resulting DOM back
// into a tree, and must arrive at that same final render.
//
// This is the WASM analog of ios/verify: it proves the JavaScript runtime
// agrees with the Go reconciler without needing a browser. What it cannot
// prove is anything about actual rendering — layout, whether enterkeyhint
// relabels a soft keyboard, whether focus() opens one. Those need a browser
// and stay out of scope here, exactly as the iOS UI layer needs a simulator.
//
// Two scenarios, because they exercise disjoint halves of the runtime:
//
//	demo    examples/mobileapp — tab switches that replace whole subtrees,
//	        keyed list rows, and the four callback kinds. This is the
//	        structural half: add/remove/replace/add-child and update-style.
//	signup  examples/signup — the focus half: focus commands (focusEpoch /
//	        focusAction), keyboard traversal (imeAction / onSubmit) and the
//	        prop churn a validating form produces.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/examples/mobileapp"
	"github.com/rohanthewiz/grmob/examples/signup"
	"github.com/rohanthewiz/grmob/render"
)

type scenario struct {
	Name    string   `json:"name"`
	Initial string   `json:"initial"`
	Steps   []string `json:"steps"`
	Final   string   `json:"final"`
}

// node mirrors just enough of core.Node's JSON to hunt down callback IDs.
type node struct {
	Type     string
	Props    map[string]any
	Children []*node
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func parse(tree string) *node {
	var n node
	if err := json.Unmarshal([]byte(tree), &n); err != nil {
		fatal("tree is not valid JSON: %v\n%s", err, tree)
	}
	return &n
}

// findAll collects every node satisfying pred, in document order — the same
// order the JS harness walks the DOM in, so "the second password field" means
// the same thing on both sides.
func findAll(n *node, pred func(*node) bool, out []*node) []*node {
	if n == nil {
		return out
	}
	if pred(n) {
		out = append(out, n)
	}
	for _, c := range n.Children {
		out = findAll(c, pred, out)
	}
	return out
}

// prop reads a string prop off the nth node of a given type that carries it,
// failing loudly rather than emitting a transcript that silently skipped an
// event.
//
// "that carries it" rather than "the nth of that type" because a Row is a
// layout primitive: a Feed screen is full of them and only the ones that are
// list rows have an onClick. Counting only the nodes that answer the question
// keeps a call site describing the event ("the first tappable row") instead
// of the markup that happens to surround it.
//
// Callback IDs are per-pass sequence numbers, so every call site re-reads the
// current tree — the discipline the browser follows, since update-props hands
// it the new ID.
func prop(tree string, nodeType string, nth int, name string) string {
	hits := findAll(parse(tree), func(n *node) bool {
		if n.Type != nodeType {
			return false
		}
		_, ok := n.Props[name].(string)
		return ok
	}, nil)
	if len(hits) <= nth {
		fatal("wanted %s #%d carrying %q, found %d in tree:\n%s", nodeType, nth, name, len(hits), tree)
	}
	return hits[nth].Props[name].(string)
}

// record appends a patch batch, dropping the empty ones. A no-change render
// serializes as "[]" and carries nothing for the harness to apply; keeping it
// would only make the step count a worse description of what happened.
func record(steps *[]string, patches string) {
	if patches != "" && patches != "[]" {
		*steps = append(*steps, patches)
	}
}

// demoScenario drives examples/mobileapp: the structural half.
func demoScenario() scenario {
	mgr := render.New(core.NewContext(), mobileapp.App)
	defer mgr.Close()

	initial := mgr.RenderInitial()
	var steps []string

	// Void event: tap Increment. A counter bump restyles nothing and rewrites
	// one Text node — the narrowest patch the runtime handles.
	record(&steps, mgr.DispatchCallback(prop(initial, "Button", 0, "onClick")))

	// Int event: switch to the Form tab. A tab switch replaces a whole
	// subtree, which is the patch type most likely to strand a node path.
	record(&steps, mgr.DispatchIntCallback(prop(mgr.RenderInitial(), "TabView", 0, "onTabChange"), 1))

	// Text events: type a name, then revise it. The second is the one that
	// matters — it proves an update-props patch reaches an element the first
	// patch created rather than the mount did.
	record(&steps, mgr.DispatchTextCallback(prop(mgr.RenderInitial(), "Input", 0, "onChange"), "Ada"))
	record(&steps, mgr.DispatchTextCallback(prop(mgr.RenderInitial(), "Input", 0, "onChange"), "Grace"))

	// Bool event: tick the subscription checkbox.
	record(&steps, mgr.DispatchBoolCallback(prop(mgr.RenderInitial(), "Checkbox", 0, "onToggle"), true))

	// The Feed tab, then a tap and a long-press on its first row: both
	// restyle rows and rewrite a status line, exercising keyed-children diffs
	// inside a List node.
	record(&steps, mgr.DispatchIntCallback(prop(mgr.RenderInitial(), "TabView", 0, "onTabChange"), 2))
	record(&steps, mgr.DispatchCallback(prop(mgr.RenderInitial(), "Row", 0, "onClick")))
	record(&steps, mgr.DispatchCallback(prop(mgr.RenderInitial(), "Row", 0, "onLongPress")))

	// Back to the Counter tab, replacing the subtree a second time.
	record(&steps, mgr.DispatchIntCallback(prop(mgr.RenderInitial(), "TabView", 0, "onTabChange"), 0))

	// The interval hook's first tick is a full second after registration and
	// this replay runs in milliseconds, so the snapshot races nothing — and
	// any tick that did sneak in was recorded as a step and is part of the
	// final tree anyway.
	return scenario{Name: "demo", Initial: initial, Steps: steps, Final: mgr.RenderInitial()}
}

// signupScenario drives examples/signup: the focus half.
func signupScenario() scenario {
	mgr := render.New(core.NewContext().WithTheme(core.DefaultTheme), signup.App)
	defer mgr.Close()

	initial := mgr.RenderInitial()
	var steps []string

	// Keyboard traversal, twice: the email field's Next key, then the
	// password field's. Each is an ordinary onSubmit dispatch that turns into
	// a focus command, so these two steps carry the imeAction props and the
	// focusEpoch/focusAction stamps that only exist once a command is issued.
	record(&steps, mgr.DispatchCallback(prop(mgr.RenderInitial(), "Input", 0, "onSubmit")))
	record(&steps, mgr.DispatchCallback(prop(mgr.RenderInitial(), "InputPassword", 0, "onSubmit")))

	// Fill the form. The confirmation field is the second InputPassword —
	// both share a placeholder, so position is the only way to name it, on
	// this side and in the JS harness alike.
	record(&steps, mgr.DispatchTextCallback(prop(mgr.RenderInitial(), "Input", 0, "onChange"), "taken@example.com"))
	record(&steps, mgr.DispatchTextCallback(prop(mgr.RenderInitial(), "InputPassword", 0, "onChange"), "hunter2222"))
	record(&steps, mgr.DispatchTextCallback(prop(mgr.RenderInitial(), "InputPassword", 1, "onChange"), "hunter2222"))
	record(&steps, mgr.DispatchBoolCallback(prop(mgr.RenderInitial(), "Checkbox", 0, "onToggle"), true))

	// Blur the email field: under RevealOnBlur this reveals or clears one
	// field's error, which is the prop churn a validating form actually
	// produces between keystrokes.
	record(&steps, mgr.DispatchCallback(prop(mgr.RenderInitial(), "Input", 0, "onBlur")))

	// Submit an address the fake server rejects. The error path issues a
	// core.Focus at the email field, so this batch re-stamps every focusable
	// leaf — the many-patches-per-command shape core/focus.go documents.
	submit := findAll(parse(mgr.RenderInitial()), func(n *node) bool {
		return n.Type == "Button" && n.Props["label"] == "Create account"
	}, nil)
	if len(submit) == 0 {
		fatal("no Create account button in the signup tree")
	}
	record(&steps, mgr.DispatchCallback(submit[0].Props["onClick"].(string)))

	// Now succeed. The form is replaced wholesale by the confirmation screen,
	// which is the only thing in either scenario that changes the tree's
	// *shape* — add, remove and replace patches, and the node paths they
	// strand if the runtime gets them wrong. Without a step like this the
	// replay only ever proves that update-props lands on the right element.
	record(&steps, mgr.DispatchTextCallback(prop(mgr.RenderInitial(), "Input", 0, "onChange"), "fresh@example.com"))
	submit = findAll(parse(mgr.RenderInitial()), func(n *node) bool {
		return n.Type == "Button" && n.Props["label"] == "Create account"
	}, nil)
	if len(submit) == 0 {
		fatal("no Create account button after correcting the address")
	}
	record(&steps, mgr.DispatchCallback(submit[0].Props["onClick"].(string)))

	// And back, so the replay covers the shape change in both directions.
	again := findAll(parse(mgr.RenderInitial()), func(n *node) bool {
		return n.Type == "Button" && n.Props["label"] != "Create account"
	}, nil)
	if len(again) == 0 {
		fatal("no button on the confirmation screen")
	}
	record(&steps, mgr.DispatchCallback(again[0].Props["onClick"].(string)))

	// Leave the terms box ticked, so the final tree the harness compares
	// against carries a checked Checkbox. Every earlier tick is undone by the
	// successful submit that resets the form, and a transcript whose only
	// checkbox is unticked cannot tell a renderer that reads the state from
	// one that hardcodes false.
	record(&steps, mgr.DispatchBoolCallback(prop(mgr.RenderInitial(), "Checkbox", 0, "onToggle"), true))

	return scenario{Name: "signup", Initial: initial, Steps: steps, Final: mgr.RenderInitial()}
}

func main() {
	out, err := json.Marshal([]scenario{demoScenario(), signupScenario()})
	if err != nil {
		fatal("marshal transcript: %v", err)
	}
	os.Stdout.Write(out)
}
