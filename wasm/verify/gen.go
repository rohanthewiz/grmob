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
// The transcript carries three things besides those, none of which has anything
// to do with the replay; they ride along because the harness reads one file:
//
//	menuCases  internal/menufixture, the picker-menu case table ios/verify and
//	           android/verify run their transliterations against. This runtime
//	           carries a fourth one, selectMenuSections in grmob-runtime.js, and
//	           select_test.mjs rebuilds the sections back out of the DOM to
//	           compare with Go's answer.
//	widgets    real components.Chip and core.Input trees rendered through every
//	           bundled theme, for the browser pass to paint and sample. See
//	           widgetCase.
//	bands      internal/bandfixture, components.GroupHeader's two inset
//	           arrangements, for the browser pass to lay out and measure against
//	           the answers ios/verify's flex solver gives.
//	bandRenders real components.GroupHeaders, one per bundled theme per shape,
//	           for the two band questions that are measurements of a rendered
//	           widget rather than arithmetic over numbers. See bandRender.
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
	"github.com/rohanthewiz/grmob/internal/bandfixture"
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
	// Bands are components.GroupHeader's two inset arrangements, for the
	// browser pass to lay out and measure. The fourth table, same reason.
	//
	// ios/verify already solves these through GrMobFlexSolver and records that
	// the two arrangements divide an overflow deficit differently there,
	// because that solver shrinks in proportion to a base that includes the
	// child's own padding. CSS distributes shrink over the *inner* flex base
	// size, which is a different rule — and until this the repository had
	// stated that in a comment and never asked a browser. browser.mjs mounts
	// them and measures the rects.
	Bands []bandfixture.Case `json:"bands"`
	// BandRenders are real components.GroupHeaders, rendered through every
	// bundled theme, for the browser to lay out with real glyphs in them. The
	// fifth table, same reason as the fourth.
	//
	// Not a duplicate of Bands. That table is the band as arithmetic — two
	// arrangements of the same chrome over synthetic content — and it is the
	// right shape for the question it answers and cannot reach two others: a
	// cross-axis one (does the disclosure's button fill the growing wrapper it
	// sits in, which is whether a press lands on the whole band) and a
	// question about text (is the padded control really the band's tallest
	// child once real glyphs are in it). See bandRender.
	BandRenders []bandRender `json:"bandRenders"`
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
		Scenarios:   []scenario{demoScenario(), signupScenario()},
		MenuCases:   menufixture.Cases(),
		Widgets:     widgetCases(),
		Bands:       bandfixture.Cases(),
		BandRenders: bandRenders(),
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

	out := make([]widgetCase, 0, len(names)*len(widgetBuilders))
	for _, name := range names {
		for _, w := range widgetBuilders {
			c, err := renderWidgetCase(name, byName[name], w)
			if err != nil {
				fatal("%v", err)
			}
			out = append(out, c)
		}
	}
	return out
}

// widgetBuilder is one widget the census paints, in every theme.
type widgetBuilder struct {
	what     string
	ringFrom string
	// build takes the theme so a widget that needs one can read it. The view
	// is built inside a theme's render context, which is why this is a
	// function rather than a value.
	build func(*core.Theme) core.View
}

// widgetBuilders is one entry per widget, so a case added here is added for
// every theme and the two loops cannot drift apart.
//
// A package-level var rather than a local, because widget_test.go renders these
// same two builders through a theme of its own — one whose ControlBorder role
// and Input base are deliberately different hexes, which is the one arrangement
// that can tell each widget's *provenance* from its pixels. See
// TestEachWidgetReadsTheAuthorityItNames there. A second list in the test would
// be a second set of widgets, and the provenance it proved would be theirs.
var widgetBuilders = []widgetBuilder{
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

// renderWidgetCase renders one widget through one theme and reads the three
// colours off the resulting nodes.
//
// It returns an error rather than calling fatal so a test can drive it: a
// widget whose rendered shape has changed must stop a `go run` and fail a
// `go test`, and os.Exit does the first and takes the whole test binary down
// doing the second.
//
// `name` labels the theme in messages and is not read from it — the caller is
// the one with the map key, and widget_test.go builds a theme that has no name
// in core at all.
func renderWidgetCase(name string, theme *core.Theme, w widgetBuilder) (widgetCase, error) {
	ctx := core.NewContext().WithTheme(theme)
	ctx.BeginRenderPass()
	page := core.Box(
		core.BackgroundColor(theme.Colors.Background),
		core.Padding(24),
		core.Width("240px"),
		w.build(theme),
	).Render(ctx)

	if len(page.Children) != 1 {
		return widgetCase{}, fmt.Errorf(
			"%s/%s: the page box rendered %d children, want the widget alone",
			name, w.what, len(page.Children))
	}
	widget := page.Children[0]
	if page.Style == nil || widget.Style == nil {
		return widgetCase{}, fmt.Errorf(
			"%s/%s: a widget swatch rendered a node with no Style", name, w.what)
	}

	c := widgetCase{
		Theme: name, What: w.what,
		Tree:     jsonout.Export(page),
		Page:     page.Style.Background,
		Fill:     widget.Style.Background,
		Ring:     widget.Style.BorderColor,
		RingFrom: w.ringFrom,
	}
	var err error
	if c.RatioOnPage, err = ratioBetween(name, c.Ring, c.Page); err != nil {
		return widgetCase{}, err
	}
	if c.RatioOnFill, err = ratioBetween(name, c.Ring, c.Fill); err != nil {
		return widgetCase{}, err
	}
	return c, nil
}

// ratioBetween is the census's own arithmetic, through internal/palette, so
// the number travelling to the browser is the number components/variant_test.go
// measures. A colour that does not parse is an error rather than zero: a ratio
// of 0 would print in a failure as a claim somebody made.
func ratioBetween(theme, a, b string) (float64, error) {
	la, oka := palette.Luminance(a)
	lb, okb := palette.Luminance(b)
	if !oka || !okb {
		return 0, fmt.Errorf("%s: a widget swatch declares an unparseable colour (%q, %q)",
			theme, a, b)
	}
	return math.Round(palette.Ratio(la, lb)*100) / 100, nil
}

// --- The rendered bands -----------------------------------------------------

// bandRender is one real components.GroupHeader, rendered through one bundled
// theme, with the paths of the nodes a browser has to measure.
//
// # The two questions the arithmetic fixture cannot reach
//
// internal/bandfixture carries the band as *numbers* — insets, a gap, a grow
// weight and synthetic content sizes — because the thing it is checking is
// arithmetic, and arithmetic over made-up sizes is still the same arithmetic.
// That is the right shape for the question it answers (do two arrangements of
// the same chrome resolve to the same geometry) and it is the wrong shape for
// two others:
//
//	the tap target   the disclosure branch puts the band's insets on a *button*
//	                 one level inside the Row's growing heading wrapper, so
//	                 whether a press lands on the whole band depends on whether
//	                 a non-growing child fills a growing parent. That is a
//	                 CROSS-axis question, and GrMobFlexSolver is a main-axis
//	                 distributor: it has no answer, and bandfixture's own
//	                 arrangementOf renders the plain band precisely to stay out
//	                 of its way. The picture in components.bandInsets assumes
//	                 the answer.
//
//	the taller child bandfixture's third case turns on the padded control being
//	                 taller than the badge, and the reason given is that the
//	                 control's vertical insets are larger AND that both wrap the
//	                 same caption type — the first half is checked in Go, and
//	                 the second is a claim about *text*, which is a measurement
//	                 no Go test can take. A bold caption shorter than a plain
//	                 one by more than the 4pt the insets differ by would make
//	                 the badge the tallest child in a real band, and every
//	                 SameHeight case downstream would be describing a layout
//	                 nothing builds.
//
// Both are measurements of a rendered band with real glyphs in it, which is
// exactly what this pass already has and nothing else in the repository does.
// So these are real GroupHeaders — the widget, not a model of it — exported the
// way the widget swatches are, and browser.mjs mounts and measures them.
//
// # Why the paths travel with the tree
//
// The browser addresses nodes by the data-node-path the runtime writes while
// walking Children, so "the control" is an index chain. Computed here by
// *finding* the node — the growing child of the band Row, and the button
// inside it where there is one — rather than spelled as a literal in the
// harness: a band whose shape changed would then move the paths with it, or
// fail here where the shape is known, instead of silently measuring whatever
// node had inherited the index.
type bandRender struct {
	Theme string `json:"theme"`
	What  string `json:"what"`
	// Tree is the page holding the band, as JSON, ready for GrMob.mount.
	Tree string `json:"tree"`

	// Band is the band Row's path, Control the node a finger lands on, and
	// Wrapper the growing heading wrapper between them — empty on the plain
	// branch, where the growing child *is* the control. Badge is empty on a
	// band with the count hidden.
	Band    string `json:"band"`
	Control string `json:"control"`
	Wrapper string `json:"wrapper"`
	Badge   string `json:"badge"`

	// Label is the run of words inside the control, and Chevron the glyph the
	// disclosure branch puts before it — empty on the plain branch, which has
	// none.
	//
	// They are here because the two branches turn out NOT to be the same
	// height, and the difference is content rather than chrome: a control's
	// height is its tallest child plus its own insets, the insets are the same
	// in both branches, and the disclosure has one child the plain band does
	// not. Carrying both paths is what lets the browser state that as an
	// equation instead of a tolerance. See check 10.
	Label   string `json:"label"`
	Chevron string `json:"chevron"`

	// RowLeft and RowRight are the band Row's own horizontal insets. The
	// leading one is 0 — that is the move, and the browser measures it rather
	// than assuming it — and the trailing one is the badge's breathing room,
	// the one inset that never went onto the control, so the browser can say
	// where the control's trailing edge is supposed to stop.
	RowLeft  float64 `json:"rowLeft"`
	RowRight float64 `json:"rowRight"`
	// Gap is the band Row's own gap, which is 0 today and is carried rather
	// than assumed for the same reason.
	Gap float64 `json:"gap"`

	// ControlGrows reports whether the control declares a grow weight of its
	// own. It is false on both branches and that is the point: on the plain
	// branch the growing child IS the control, and on the disclosure branch
	// the control is a non-growing child of a growing wrapper. A control that
	// started growing would make the browser's answer true for a reason that
	// has nothing to do with the question.
	ControlGrows bool `json:"controlGrows"`

	// Collapsible separates the two branches, which are asked different
	// things: only the disclosure has a wrapper to fill.
	Collapsible bool `json:"collapsible"`

	// Fill is the band Row's own background and Page the fill of the box it
	// sits on, both read off the rendered nodes the way the widget grid reads
	// its three.
	//
	// They are here because this grid mounted nine real widgets and read no
	// pixels at all. Every other browser check that mounts something real
	// samples a colour; this one measured rects, so a band that laid out
	// perfectly and painted nothing would have passed every assertion in it —
	// and "the band has a fill of its own" is the whole reason it may span its
	// container edge to edge rather than being inset like a row.
	//
	// Page is carried alongside so the sample can distinguish. A theme whose
	// Surface equalled its Background would make the pixel read agree with
	// both answers at once, which is a check that cannot fail rather than one
	// that passes.
	Fill string `json:"fill"`
	Page string `json:"page"`
}

// bandRenderBuilders is one entry per band shape, so a shape added here is
// added for every bundled theme.
//
// Three shapes, and each earns its place. The plain band with a badge is the
// baseline the disclosure is compared against — the two branches are supposed
// to be the same band geometrically, which is what
// components.GroupHeader.ControlStyle promises a caller who adds a handler. The
// disclosure with a badge is the tap-target case. The disclosure with the count
// hidden is the one where the control's trailing edge is the band's own
// trailing inset rather than the gap before a badge, which is the asymmetric
// side components.bandInsets' `trailing` parameter exists for.
var bandRenderBuilders = []struct {
	what        string
	collapsible bool
	build       func() components.GroupHeader
}{
	{"a plain band", false, func() components.GroupHeader {
		return components.GroupHeader{Group: bandRenderGroup}
	}},
	{"a disclosure band", true, func() components.GroupHeader {
		return components.GroupHeader{
			Group: bandRenderGroup, Expanded: true, OnToggle: func() {},
		}
	}},
	{"a disclosure band, count hidden", true, func() components.GroupHeader {
		return components.GroupHeader{
			Group: bandRenderGroup, Expanded: true, OnToggle: func() {},
			HideCount: true,
		}
	}},
}

// The group every rendered band titles. The same one internal/bandfixture
// reads its numbers off, so the two tables are describing one band.
var bandRenderGroup = components.Group{Key: "2026-01", Label: "January 2026", Count: 3}

// bandRenderWidth is the page the bands are laid out in.
//
// Wide enough that the label has slack to grow into — the grow weight is the
// whole subject, and a band squeezed to its content would answer the tap-target
// question with "yes, because there was nowhere else to go".
const bandRenderWidth = 320

func bandRenders() []bandRender {
	byName := core.BundledThemes()
	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]bandRender, 0, len(names)*len(bandRenderBuilders))
	for _, name := range names {
		for _, b := range bandRenderBuilders {
			c, err := renderBandCase(name, byName[name], b.what, b.collapsible, b.build())
			if err != nil {
				fatal("%v", err)
			}
			out = append(out, c)
		}
	}
	return out
}

// renderBandCase renders one band through one theme and locates the nodes the
// browser measures.
//
// An error rather than fatal, for the same reason renderWidgetCase returns one:
// a band whose rendered shape has changed must stop a `go run` and fail a
// `go test`, and os.Exit does the first and takes the whole test binary down
// doing the second.
func renderBandCase(name string, theme *core.Theme, what string, collapsible bool,
	band components.GroupHeader) (bandRender, error) {

	ctx := core.NewContext().WithTheme(theme)
	ctx.BeginRenderPass()
	// A page with no padding of its own: the band is supposed to span its
	// container edge to edge (that is what its own fill is for), and an inset
	// page would make every leading-edge measurement below relative to a
	// margin the widget knows nothing about.
	page := core.Box(
		core.BackgroundColor(theme.Colors.Background),
		core.Width(fmt.Sprintf("%dpx", bandRenderWidth)),
		band,
	).Render(ctx)

	if len(page.Children) != 1 {
		return bandRender{}, fmt.Errorf(
			"%s/%s: the page box rendered %d children, want the band alone",
			name, what, len(page.Children))
	}
	row := page.Children[0]
	if row.Style == nil {
		return bandRender{}, fmt.Errorf("%s/%s: the band Row rendered with no Style",
			name, what)
	}

	c := bandRender{
		Theme: name, What: what, Tree: jsonout.Export(page),
		Band: "root/0",
		// Read off the rendered nodes rather than off the theme: what the
		// browser is asked is whether the band's OWN declaration reaches the
		// screen, and a colour taken from theme.Colors would still be there
		// after the widget stopped declaring it.
		Fill:     row.Style.Background,
		Page:     page.Style.Background,
		RowLeft:  float64(row.Style.Padding.Left),
		RowRight: float64(row.Style.Padding.Right),
		Gap:      row.Style.Gap,
		// The count is the last child when there is one; hidden, the growing
		// child is all there is.
		Collapsible: collapsible,
	}

	if c.Fill == "" {
		return bandRender{}, fmt.Errorf(
			"%s/%s: the band Row declares no background. Its fill is why a band may span "+
				"its container edge to edge, and with none there is nothing for the "+
				"browser to read back", name, what)
	}
	if c.Fill == c.Page {
		return bandRender{}, fmt.Errorf(
			"%s/%s: the band's fill and the page behind it are both %s, so a pixel taken "+
				"inside the band agrees with either answer and the paint check cannot "+
				"fail. A theme whose Surface equals its Background needs a different "+
				"backdrop here, not a check that passes on it", name, what, c.Fill)
	}

	// The growing child, found rather than indexed. There must be exactly one:
	// the band pins its badge to the trailing edge with a grow weight on the
	// heading rather than with JustifyContent (see GroupHeader.Render), and a
	// second growing child would divide the slack and move the badge inward.
	grow := -1
	for i, child := range row.Children {
		if child.Style != nil && child.Style.FlexGrow > 0 {
			if grow >= 0 {
				return bandRender{}, fmt.Errorf(
					"%s/%s: the band has more than one growing child (%d and %d) — "+
						"the badge is pinned by a single grow weight, and two of them "+
						"share the slack", name, what, grow, i)
			}
			grow = i
		}
	}
	if grow < 0 {
		return bandRender{}, fmt.Errorf(
			"%s/%s: no child of the band grows — the whole tap-target question is "+
				"about padding on a *stretched* child", name, what)
	}
	if grow != 0 {
		return bandRender{}, fmt.Errorf(
			"%s/%s: the growing child is at index %d and the leading edge is index 0 — "+
				"the label is supposed to come first", name, what, grow)
	}

	if len(row.Children) > 2 {
		return bandRender{}, fmt.Errorf(
			"%s/%s: the band rendered %d children and this fixture reads at most two",
			name, what, len(row.Children))
	}
	if len(row.Children) == 2 {
		c.Badge = "root/0/1"
	}

	// Where the insets are, and therefore what a finger lands on. On the plain
	// branch the growing child carries them itself; as a disclosure it is a
	// wrapper with no chrome at all and the button inside it is the target.
	growing := row.Children[grow]
	if collapsible {
		if len(growing.Children) != 1 {
			return bandRender{}, fmt.Errorf(
				"%s/%s: the heading wrapper holds %d children, want the button alone",
				name, what, len(growing.Children))
		}
		control := growing.Children[0]
		if control.Style == nil {
			return bandRender{}, fmt.Errorf(
				"%s/%s: the band's button rendered with no Style", name, what)
		}
		if control.Props["onClick"] == nil {
			return bandRender{}, fmt.Errorf(
				"%s/%s: the node inside the heading wrapper has no handler, so it is "+
					"not the thing a press lands on", name, what)
		}
		// The wrapper must be geometrically invisible, or "the button fills the
		// wrapper" would be a claim about a box with chrome of its own and the
		// tap target would stop short of the band by however much it carries.
		if p := growing.Style.Padding; p != (core.EdgeInsets{}) {
			return bandRender{}, fmt.Errorf(
				"%s/%s: the heading wrapper carries padding %+v — it is supposed to be "+
					"geometrically invisible, and chrome on it is chrome the tap target "+
					"does not reach", name, what, p)
		}
		c.Wrapper, c.Control = "root/0/0", "root/0/0/0"
		c.ControlGrows = control.Style.FlexGrow > 0
		// Two children, in order: the chevron then the words. Located by
		// content rather than by index — an arm that swapped them would
		// otherwise have the browser comparing the label's line box with
		// itself.
		if len(control.Children) != 2 {
			return bandRender{}, fmt.Errorf(
				"%s/%s: the button holds %d children, want the chevron and the words",
				name, what, len(control.Children))
		}
		for i, child := range control.Children {
			path := fmt.Sprintf("%s/%d", c.Control, i)
			if child.Props["content"] == band.Group.Label {
				c.Label = path
			} else {
				c.Chevron = path
			}
		}
		if c.Label == "" || c.Chevron == "" {
			return bandRender{}, fmt.Errorf(
				"%s/%s: the button's children are not a chevron and the group's label — "+
					"the height comparison in check 10 is an equation over exactly those two",
				name, what)
		}
	} else {
		c.Control = "root/0/0"
		if len(growing.Children) != 1 {
			return bandRender{}, fmt.Errorf(
				"%s/%s: the plain band's control holds %d children, want the words alone",
				name, what, len(growing.Children))
		}
		if growing.Children[0].Props["content"] != band.Group.Label {
			return bandRender{}, fmt.Errorf(
				"%s/%s: the plain band's control does not hold the group's label",
				name, what)
		}
		c.Label = "root/0/0/0"
		// The plain branch's growing child IS the control, so of course it
		// grows; the field means "the control has a weight of its own beyond
		// being the growing child", which on this branch it cannot.
		c.ControlGrows = false
	}
	return c, nil
}
