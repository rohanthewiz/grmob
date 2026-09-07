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
//
// The transcript carries one thing besides those: the picker-menu case table
// (internal/menufixture), which has nothing to do with the replay and rides
// along because the harness reads one file. It is the same table ios/verify
// and android/verify run their transliterations against — this runtime carries
// a fourth one, selectMenuSections in grmob-runtime.js, and select_test.mjs
// rebuilds the sections back out of the DOM to compare with Go's answer.
package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"

	"github.com/rohanthewiz/grmob/components"
	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/examples/mobileapp"
	"github.com/rohanthewiz/grmob/examples/signup"
	"github.com/rohanthewiz/grmob/internal/menufixture"
	"github.com/rohanthewiz/grmob/internal/palette"
	"github.com/rohanthewiz/grmob/jsonout"
	"github.com/rohanthewiz/grmob/render"
)

type scenario struct {
	Name    string   `json:"name"`
	Initial string   `json:"initial"`
	Steps   []string `json:"steps"`
	Final   string   `json:"final"`
}

// transcript is what the harness reads: the replay scenarios plus the picker
// menu cases.
//
// An object rather than the bare array of scenarios this used to emit,
// because a second, unrelated table had to travel in the same file — the same
// arrangement ios/verify's transcript already had, and for the same reason:
// run.sh generates one file and every .mjs suite reads it.
type transcript struct {
	Scenarios []scenario         `json:"scenarios"`
	MenuCases []menufixture.Case `json:"menuCases"`
	// Widgets are real components rendered through real themes, for the
	// browser pass. See widgetCase: it is the third unrelated table riding in
	// this file, and for the same reason as the second — run.sh generates one
	// file and every consumer reads it.
	Widgets []widgetCase `json:"widgets"`
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

	// And move the picker off its initial value, for the same reason: a
	// transcript whose picker never leaves "free" cannot tell a runtime that
	// applies the value from one that renders the first option and stops.
	//
	// What this step covers is that the value lands through a patch, and that
	// the <option> elements stay invisible to the tree comparison — they are
	// chrome, and an unmarked one would show up here as three extra nodes Go
	// never rendered. That the same-list case is answered *without* rebuilding
	// the options is a property no final tree can see (the rebuilt options
	// would be identical), so it is pinned in select_test.mjs instead.
	record(&steps, mgr.DispatchTextCallback(prop(mgr.RenderInitial(), "Select", 0, "onChange"), "pro"))

	return scenario{Name: "signup", Initial: initial, Steps: steps, Final: mgr.RenderInitial()}
}

func main() {
	out, err := json.Marshal(transcript{
		Scenarios: []scenario{demoScenario(), signupScenario()},
		MenuCases: menufixture.Cases(),
		Widgets:   widgetCases(),
	})
	if err != nil {
		fatal("marshal transcript: %v", err)
	}
	os.Stdout.Write(out)
}

// --- The widget swatches ----------------------------------------------------

// widgetCase is one real widget, rendered by Go, with the colours it actually
// puts on screen.
//
// # Why the browser pass needed this
//
// palette.mjs's rows are (tone, backdrop) pairs painted by browser.mjs itself:
// a 120x52 box in the backdrop with an 80x24 box inside it carrying a 1px
// border in the tone. That geometry is a *model* of a control boundary, and it
// proved the thing it was built to prove — that Chrome puts the census's hexes
// on the screen, through the runtime's style mapping, with no alpha or colour
// management in between.
//
// What it could not prove is that any widget draws them. Everything between
// core.ColorPalette.ControlBorder and a pixel goes through `components`, and
// `components` is Go: the browser pass mounts JSON and cannot call it. So a
// chip whose ring had stopped being the boundary tone — a Style override, a
// dropped BorderWidth, a fallback taken — would leave every swatch in
// palette.mjs painting perfectly and every Go test passing on hex strings.
//
// This closes that: the tree below is a real components.Chip rendered through
// a real theme, and the three colours are read off the *rendered node* rather
// than off the palette. What the browser then checks is that the widget's own
// declarations reach the screen.
type widgetCase struct {
	Theme string `json:"theme"`
	What  string `json:"what"`
	// Tree is the rendered widget as JSON, ready for GrMob.mount.
	Tree string `json:"tree"`
	// Page is the fill behind the widget, Fill is the widget's own, and Ring
	// is the boundary tone between them — each read off the node that carries
	// it, so a widget that stopped declaring one is a case with an empty
	// column rather than a case that still checks the palette's answer.
	Page string `json:"page"`
	Fill string `json:"fill"`
	Ring string `json:"ring"`
	// The two ratios the census computes for this ring, carried across so a
	// failure can say what was believed about the pair. Nothing in the browser
	// recomputes them, for the reason palette.mjs gives.
	RatioOnPage float64 `json:"ratioOnPage"`
	RatioOnFill float64 `json:"ratioOnFill"`

	// RingFrom names the Go authority this widget's boundary is supposed to
	// come from, so widget_test.go can hold each case to its own.
	//
	// It exists because the two widgets here read the tone from two different
	// places, deliberately. components.chipRing reads
	// Colors.ControlBorderColor — the role — while core.Input reads
	// Components.Input.BorderColor, the field base, which is a *literal* that
	// core/theme_test.go pins to the role separately. The two hold the same hex
	// in all three bundled themes, so a case checked against the wrong one
	// would pass, and the day somebody restyles their fields is the day both
	// checks would have been wrong at once.
	//
	// A string rather than the expected hex: carrying the hex would make this
	// a copy of the answer, and the point is to name the *source* so the Go
	// side goes and reads it. A spelling widget_test.go does not know is a
	// failure, which is what stops a new case borrowing its neighbour's
	// authority by leaving the field blank.
	RingFrom string `json:"ringFrom"`
}

// The two spellings RingFrom takes. Declared here, beside the field, and read
// by widget_test.go, which maps each to the value it names.
const (
	ringFromRole      = "Colors.ControlBorder"
	ringFromInputBase = "Components.Input.BorderColor"
)

// widgetCases renders both halves of ControlBorder's job, per bundled theme:
// a quiet chip's ring and a text field's frame.
//
// # Why a chip and why quiet
//
// components.Chip is the widget core.ColorPalette.ControlBorder was introduced
// for (see chipRing), and ProminenceQuiet is the arm that draws the ring: a
// filter row is chrome that is meant to recede, and the ring is the whole of
// what says the control is there. It is also the widget with *two* backdrops —
// the page outside the ring and its own Surface fill inside it — which is the
// pair shape the census wants at least one real widget to have.
//
// Unselected, because a selected chip paints its accent over both.
//
// # Why a field as well
//
// The chip is the role's second spender; the field frame is the first, and it
// spends the tone through a different Go value. components.chipRing reads
// Colors.ControlBorderColor — the role — while core.Input reads
// Components.Input.BorderColor, a literal each theme states and
// core/theme_test.go pins to the role separately, because a core.Style is a
// value and a component default cannot call a resolver. The two hold the same
// hex in all three bundled themes, so a swatch for one says nothing about the
// other: that is what RingFrom is for.
//
// It is also a second tag. Both widgets draw over a user-agent border — a chip
// is a tappable control and exports as a `<button>`, which is the first member
// htmlout's borderResetTypes ever had — but the rules differ (`<button>` is
// given `outset`, `<input>` `inset`, and an `<input>` arrives with a fill and
// padding of its own). Whether the theme's frame *replaces* the user agent's
// rather than tinting it has no answer in Go: a renderer emitting
// `border-color` and `border-width` without `border-style` leaves the UA style
// in force, which paints one hex as two tones while every tree comparison,
// every exporter test and every contrast calculation passes.
//
// # The page
//
// A Box carrying the theme's own Colors.Background rather than the document's
// default white, so the ring's outer backdrop is the one the census measured
// rather than whatever the browser's body happens to be. Both cases are built
// the same way, and the browser reads both at root and root/0 for that reason.
func widgetCases() []widgetCase {
	byName := core.BundledThemes()
	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)

	// One entry per widget, so a case added here is added for every theme and
	// the two loops cannot drift apart. The view is built inside a theme's
	// context, which is why it is a function rather than a value.
	widgets := []struct {
		what     string
		ringFrom string
		build    func(*core.Theme) core.View
	}{
		{"quiet Chip", ringFromRole, func(*core.Theme) core.View {
			return components.Chip{Label: "Sermons"}
		}},
		// Empty value and no handler: what is under test is the frame, and a
		// field with text in it puts ink near the fill sample. The placeholder
		// is empty for the same reason — placeholder ink is drawn inside the
		// padding, which is exactly where the fill is read.
		{"Input frame", ringFromInputBase, func(*core.Theme) core.View {
			return core.Input("", "", nil)
		}},
	}

	out := make([]widgetCase, 0, len(names)*len(widgets))
	for _, name := range names {
		theme := byName[name]
		for _, w := range widgets {
			ctx := core.NewContext().WithTheme(theme)
			ctx.BeginRenderPass()
			page := core.Box(
				core.BackgroundColor(theme.Colors.Background),
				core.Padding(24),
				core.Width("240px"),
				w.build(theme),
			).Render(ctx)

			if len(page.Children) != 1 {
				fatal("%s/%s: the page box rendered %d children, want the widget alone",
					name, w.what, len(page.Children))
			}
			widget := page.Children[0]
			if page.Style == nil || widget.Style == nil {
				fatal("%s/%s: a widget swatch rendered a node with no Style", name, w.what)
			}

			c := widgetCase{
				Theme: name, What: w.what,
				Tree:     jsonout.Export(page),
				Page:     page.Style.Background,
				Fill:     widget.Style.Background,
				Ring:     widget.Style.BorderColor,
				RingFrom: w.ringFrom,
			}
			c.RatioOnPage = ratioBetween(name, c.Ring, c.Page)
			c.RatioOnFill = ratioBetween(name, c.Ring, c.Fill)
			out = append(out, c)
		}
	}
	return out
}

// ratioBetween is the census's own arithmetic, through internal/palette, so
// the number travelling to the browser is the number components/variant_test.go
// measures. A colour that does not parse is fatal rather than zero: a ratio of
// 0 would print in a failure as a claim somebody made.
func ratioBetween(theme, a, b string) float64 {
	la, oka := palette.Luminance(a)
	lb, okb := palette.Luminance(b)
	if !oka || !okb {
		fatal("%s: a widget swatch declares an unparseable colour (%q, %q)", theme, a, b)
	}
	return math.Round(palette.Ratio(la, lb)*100) / 100
}
