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
//	pins       internal/pinfixture, one overflowing Row in four arrangements,
//	           for the browser to lay out so that the CSS half of the pin census
//	           is a measurement of a browser rather than of one solver.
package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/rohanthewiz/grmob/components"
	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/examples/mobileapp"
	"github.com/rohanthewiz/grmob/examples/signup"
	"github.com/rohanthewiz/grmob/internal/bandfixture"
	"github.com/rohanthewiz/grmob/internal/menufixture"
	"github.com/rohanthewiz/grmob/internal/palette"
	"github.com/rohanthewiz/grmob/internal/pinfixture"
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
	// InkProbes are the antialiasing probes the band grid's ink assertions
	// rest on — one per distinct text declaration those assertions read, each
	// naming the boxes it answers for. See inkProbe: the rendering mode used to
	// be asked once for the whole page, and what one probe licensed was a claim
	// about thirty boxes it did not paint.
	InkProbes []inkProbe `json:"inkProbes"`
	// InkLigatures are the character pairs inkGlyphPerCharacter refuses,
	// mounted in one of the declarations the ink scan reads so the browser can
	// ask the face it actually resolved which of them it draws as one glyph.
	// See inkLigature: the refusal is a superset by design, and this is the
	// only thing that measures it.
	InkLigatures []inkLigature `json:"inkLigatures"`
	// Pins are internal/pinfixture's four arrangements of one overflowing Row.
	// The sixth table, same reason as the rest.
	//
	// ios/verify already solves these through GrMobFlexSolver and compares them
	// with the Compose column the fixture carries. That solver is this
	// repository's CSS arithmetic rather than a browser's, and the band census
	// two fields up records one place where it and a real Chrome part company
	// — so "these children have no padding, therefore the two agree" was a
	// sentence with nothing behind it. browser.mjs mounts the same Rows.
	Pins []pinfixture.Case `json:"pins"`
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
	// The pin fixture states what would make it vacuous, and it is asked on the
	// side that owns the numbers: a Row that does not overflow squeezes nobody,
	// so a pinned child in it is indistinguishable from an unpinned one and
	// every comparison downstream would pass by never reaching the arithmetic.
	// ios/verify's gen.go asks the same question for the same reason — this is
	// a second writer of the same fixture, not a second copy of the rule.
	if err := pinfixture.Validate(); err != nil {
		fatal("the pinned-Row fixture is vacuous: %v", err)
	}
	renders, probes, ligatures := bandRenders()
	out, err := json.Marshal(transcript{
		Scenarios:    []scenario{demoScenario(), signupScenario()},
		MenuCases:    menufixture.Cases(),
		Widgets:      widgetCases(),
		Bands:        bandfixture.Cases(),
		BandRenders:  renders,
		InkProbes:    probes,
		InkLigatures: ligatures,
		Pins:         pinfixture.Cases(),
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

	// Leading is the control's FIRST child — the words on the plain branch,
	// the chevron on the disclosure — and ControlPadLeft the control's own
	// leading padding, which is what separates the two.
	//
	// # What they close
	//
	// The band with a caller's own indent (see bandRenderBuilders) exists so
	// the ink scan meets a label rect the other shapes never produce, and
	// every assertion over it was one the other four also make. Nothing said
	// the label had actually MOVED: a ControlStyle that stopped reaching the
	// control would render the theme's own inset, the browser would measure
	// that inset, and the shape would pass while exercising nothing.
	//
	// So the indent is stated by the fixture rather than read back off the
	// node it is supposed to have set (renderBandCase checks the rendered
	// padding against the constant the builder declared), and the browser
	// holds the content's leading edge to it in pixels. The tap target's own
	// leading edge is asserted separately and must NOT move — that is the
	// difference between padding on the control and padding on the Row, and
	// the reason ControlStyle is a safe thing to hand a caller.
	//
	// The first child rather than the label, because the disclosure branch
	// puts the chevron in front of the words: `label.x - control.x` is the
	// padding on one branch and the padding plus a glyph and a gap on the
	// other, and only one of those is a declaration.
	Leading        string  `json:"leading"`
	ControlPadLeft float64 `json:"controlPadLeft"`
	// ControlIndent is the indent this shape declared through ControlStyle, or
	// 0 for a shape that declares none and takes the theme's own band inset.
	// Carried so a failure can name which of the two the number is: the same
	// pixel measurement means "the caller's indent did not arrive" on one shape
	// and "the band's chrome did not arrive" on the other four.
	ControlIndent float64 `json:"controlIndent"`

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

	// Pair names the parity comparison this shape is one half of, empty for a
	// shape in none. See bandRenderBuilders: the browser pairs the two halves
	// by this name rather than working out from Collapsible and Badge which two
	// shapes are supposed to be the same band, which stopped being a question
	// two booleans could answer the moment a second count-hidden shape arrived.
	Pair string `json:"pair"`

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

	// LabelInk is the colour the band's own words are declared in, and
	// BadgeFill the count pill's background.
	//
	// # Why one pixel was not enough
	//
	// The grid read exactly one colour per band: the Row's own fill, six
	// device-independent pixels in and vertically centred, which lands in the
	// control's leading padding where there is no ink. That answers "did the
	// band paint" and nothing else — a band that painted its fill over an
	// invisible label passes it, and so does one whose count pill has stopped
	// drawing a pill.
	//
	// The widget grid does not have that shape: it reads a fill AND scans both
	// boundary edges, on the argument that either single edge is consistent
	// with a correct frame. The same argument applies here. A band is a fill, a
	// run of words and a count, and two of the three were unread.
	//
	// Read off the rendered nodes rather than off the theme, for the reason
	// Fill is: what a browser is asked is whether the widget's OWN declaration
	// reaches the screen, and a colour taken from theme.Colors would still be
	// there after the widget stopped declaring it.
	LabelInk  string `json:"labelInk"`
	BadgeFill string `json:"badgeFill"`

	// BadgePadLeft is the count pill's own leading padding, in
	// device-independent pixels.
	//
	// It is here so that the pixel the browser samples inside the pill is
	// DERIVED from the widget rather than chosen against it. The sample used to
	// be a literal 5 — inside the pill's radius and inside its 8px leading
	// padding, on the themes bundled today — which is a number measured against
	// a widget's current insets, exactly the shape the census's byte windows
	// were one level up. A pill whose padding shrank to 4 would put that sample
	// on a digit and the check would fail describing a colour, not a cause.
	//
	// Half the padding is what the browser samples: far enough from the pill's
	// leading edge, which at mid-height is the apex of a 999-radius curve and
	// therefore antialiased, and short of the first digit. Both ends of that
	// are why renderBandCase refuses a padding too small to sample inside.
	BadgePadLeft float64 `json:"badgePadLeft"`

	// BadgeInk is the colour the count's DIGITS are declared in, and
	// BadgePadRight the pill's trailing padding.
	//
	// # Why a pill's fill was not the whole of a count
	//
	// "A band is a fill, a run of words and a count" is the argument the ink
	// scan was added for, and only two thirds of it were ever read. The label
	// got a scan; the count got the same single point sample the band's fill
	// gets, taken inside the pill's own leading padding — which is a run of
	// fill with no digit in it by construction. A pill that painted itself
	// perfectly and rendered no number at all passed, which is precisely the
	// state the LABEL was in before any of this existed.
	//
	// So the digits are scanned the way the words are, and the two paddings say
	// where: the pill's radius is 999, so its leading and trailing edges are
	// the apexes of a curve and every pixel there is a blend with the band
	// behind it. Between the two paddings is the digits' own box — the one
	// region of the pill that is fill and digit and nothing else.
	//
	// Read off the rendered node rather than off the theme, for the reason Fill
	// and LabelInk are: what a browser is asked is whether the widget's OWN
	// declaration reaches the screen.
	BadgeInk      string  `json:"badgeInk"`
	BadgePadRight float64 `json:"badgePadRight"`

	// LabelProbe and BadgeProbe name the antialiasing probe that answers for
	// each of the two scanned boxes. See inkProbe.
	LabelProbe string `json:"labelProbe"`
	BadgeProbe string `json:"badgeProbe"`
}

// inkProbe is one text declaration, reproduced in black on white so the
// rendering mode it resolves to can be read as a colour.
//
// # The one measurement that stood in for thirty
//
// The whole band ink scan rests on antialiasing being LINEAR: a glyph pixel at
// coverage c is `fill + c*(want - fill)`, so every legitimate pixel in a text
// box is on the segment between the two colours (browser.mjs's offSegment) and
// a stem's interior is the declared colour exactly (its INK_EPSILON). That is
// grayscale antialiasing's arithmetic. LCD subpixel antialiasing gives each
// channel its own coverage, and under it ordinary correct text is off that line
// by tens of channels.
//
// The mode was asked once — black words on a white page, one size, one weight,
// one box — and what that licensed was a claim about every label and every
// count in the grid, in colours the probe did not paint, at weights it did not
// set, at whatever alpha a theme's ink carries. Chrome picks a text rendering
// path PER ELEMENT: a translucent ink, a transform, a compositing ancestor can
// each move one box off the path its neighbour took. So one measurement stood
// in for thirty, and the thirty are the ones the tolerance is spent on.
//
// # What a probe is, and what it can and cannot reproduce
//
// Each probe carries the scanned node's OWN Style — every field of it, so a
// declaration this file has never thought about (a LineHeight, a Rotate, a
// Transition) travels with it — over a white box, with the text colour
// replaced by black at the same alpha. Black at any alpha over white is a grey,
// so the verdict stays the one a screenshot can state without begging the
// question: under grayscale antialiasing every blend of two greys is a grey, so
// a pixel whose channels differ is a subpixel-rendered one. Asking the probe in
// the band's own colours instead would be asking offSegment, which is the thing
// this exists to license.
//
// What it reproduces is the declaration and its immediate backdrop. What it
// cannot reproduce is an ancestor — a band that acquired a transform or a
// compositing layer would move its own text's path without moving its probe's.
// That is a smaller gap than one probe for the whole page, and it is a gap:
// the probes answer for declarations, and the day something above a band starts
// deciding how its glyphs are drawn, this stops being the right question.
//
// # Why they are deduplicated
//
// Two bands that declare the same words in the same face at the same alpha are
// one question, and the grid is a fold budget — the bands have to stay above
// the bottom of the viewport for any of them to be scanned at all. Key is the
// declaration itself, so a theme that stopped differing from another silently
// shares its answer instead of paying for a second one.
type inkProbe struct {
	// Key is the probe's identity: the declaration it reproduces, rendered.
	// Two boxes with the same key are the same question.
	Key string `json:"key"`
	// What names the declarations this probe answers for, so a failure can say
	// which parts of the grid it invalidates rather than only which box was
	// coloured.
	What string `json:"what"`
	// Tree is the probe box, as JSON, ready to be mounted alongside the bands.
	Tree string `json:"tree"`
	// Width is the box's declared width in device-independent pixels. The
	// browser holds the rendered rect to it: probes are laid out in one Row to
	// keep the grid's height, and a Row that ran out of width would squeeze
	// them into slivers and the scan would report a grey page because it read
	// almost none of one.
	Width float64 `json:"width"`

	// subject is which of a band's two scanned boxes this probe was built for.
	// Unexported: it is how bandRenders knows which field of the case to put
	// the key in, and the browser reads the key rather than the reason.
	subject inkProbeSubject

	// style is the declaration this probe's own Text node carries, kept so the
	// ligature row can be drawn in one of the declarations the scan actually
	// reads rather than in whatever the page inherits. Unexported for the same
	// reason as subject: it never crosses to the browser, which reads the
	// rendered tree. See inkLigatureRow.
	style core.Style
}

// inkProbeSubject is which scanned box a probe answers for.
type inkProbeSubject int

const (
	inkProbeLabel inkProbeSubject = iota
	inkProbeBadge
)

// The run of glyphs every probe paints.
//
// Nothing but vertical strokes with gaps between them, which is the shape that
// produces the most antialiased edge per pixel scanned — and a fringe, when
// there is one, lives on an edge.
const inkProbeText = "illlim"

// How wide a probe box is, in device-independent pixels.
//
// Enough for inkProbeText at any caption size a theme sets, and small enough
// that the probes fit on one line of bandRenderWidth beside each other. The
// browser checks the rendered width against this rather than trusting it.
const inkProbeWidth = 56

// The character pairs a Latin text face is entitled to draw as one glyph.
//
// The f-ligatures are the ones every serif face in common use carries, the UA
// default these bands render in among them; "st" is the discretionary one that
// some carry as well. A superset is the right shape: what this has to rule out
// is a fixture string whose glyph count is a question about which optional
// ligatures a particular build shipped with, and being wrong in the direction
// of refusing an innocent pair costs a fixture author one word.
var inkLigatureSeeds = []string{
	"ff", "fi", "fl", "ft", "fb", "fh", "fj", "fk", "st",
}

// inkGlyphPerCharacter says why a browser's glyph count for this string cannot
// be compared against its characters, or "" when it can.
//
// # A property of the fixture, presented as a property of the face
//
// browser.mjs's inkFaceFault holds every scanned run to one glyph per
// character: `measureText` reads the DOM's characters and
// CSS.getPlatformFontsForNode counts the compositor's glyphs, so the pair is a
// third answer to "was the string on the page the string that was measured".
//
// That claim holds on this grid because of what this grid SAYS — "January
// 2026", one longer title, a single digit, "illlim" — and not because a face
// draws a glyph per character in general. A face is entitled to draw one glyph
// for "fi" and the common ones do. The first fixture string with an
// f-ligature in it therefore makes that check fire for a reason that is
// correct rendering, and it fires as a font substitution in the middle of an
// antialiasing check rather than as the fixture choice it actually is.
//
// So the choice is made where the strings are and it fails here, with the
// string in front of whoever wrote it. Non-ASCII is refused for the same
// reason from the other direction: a composed form, a combining mark or an
// emoji sequence is any number of glyphs for any number of code points, and
// the two counts stop being comparable before any face has an opinion.
func inkGlyphPerCharacter(s string) string {
	for _, r := range s {
		if r < 0x20 || r > 0x7e {
			return fmt.Sprintf("%q is outside printable ASCII, where a code point "+
				"and a glyph stop being the same unit", r)
		}
	}
	low := strings.ToLower(s)
	for _, seed := range inkLigatureSeeds {
		if strings.Contains(low, seed) {
			return fmt.Sprintf("it contains %q, which a text face may draw as one "+
				"glyph", seed)
		}
	}
	return ""
}

// inkGlyphFault is inkGlyphPerCharacter with the whole argument attached, for
// the three places a scanned run's text is decided.
func inkGlyphFault(where, what, s string) error {
	why := inkGlyphPerCharacter(s)
	if why == "" {
		return nil
	}
	return fmt.Errorf("%s: %s is %q, and %s.\n\n"+
		"browser.mjs holds every run this grid scans to being drawn with one glyph "+
		"per character — the DOM's characters are what its width is measured over "+
		"and the compositor's glyphs are what it counts, and the two agreeing is the "+
		"only evidence there that the string on the page is the string that was "+
		"measured. That agreement is a property of the strings named here, not of "+
		"the face. This one breaks it, and the check it breaks presents itself as a "+
		"font substitution in the middle of an antialiasing scan. Choose another "+
		"word, or take the glyph count out of inkFaceFault and say what replaces it.",
		where, what, s, why)
}

// inkProbeFor builds the probe that answers for one scanned text node.
//
// Returns the probe and its key. The key is the rendered declaration, so
// two nodes that declare the same thing produce the same probe.
func inkProbeFor(text *core.Node, subject inkProbeSubject, what string) (inkProbe, error) {
	if text == nil || text.Style == nil {
		return inkProbe{}, fmt.Errorf(
			"%s: no styled text node to build an antialiasing probe from", what)
	}
	// The probe's own run is scanned like any other, so the string it paints is
	// held to the same property the bands' are. See inkGlyphPerCharacter.
	if err := inkGlyphFault(what, "the probe's own run (inkProbeText)",
		inkProbeText); err != nil {
		return inkProbe{}, err
	}
	// The node's own Style, copied whole. A field this file has never heard of
	// is a field that might change how Chrome draws the glyphs, and the probe
	// is only worth anything while it is drawing them the same way.
	style := *text.Style
	ink, err := inkProbeColor(style.TextColor)
	if err != nil {
		return inkProbe{}, fmt.Errorf("%s: %w", what, err)
	}
	style.TextColor = ink
	// The box's own paint, not the text's: a probe reads its whole rect, and a
	// background on the run would be a second colour in it.
	style.Background = ""
	style.Width = ""
	style.Height = ""

	box := &core.Node{
		Type: "Box",
		Style: &core.Style{
			Background: "#FFFFFF",
			Width:      fmt.Sprintf("%dpx", inkProbeWidth),
			// A flex item in the probe Row below. Without this a Row narrower
			// than its probes shrinks them all, and a probe read across two
			// device pixels is a page that looks grey because almost none of
			// it was looked at.
			FlexShrink: core.ShrinkNone,
		},
		Children: []*core.Node{{
			Type:  "Text",
			Props: map[string]any{"content": inkProbeText},
			Style: &style,
		}},
	}
	tree := jsonout.Export(box)
	return inkProbe{
		Key: tree, What: what, Tree: tree, Width: inkProbeWidth, subject: subject,
		style: style,
	}, nil
}

// inkLigature is one of the pairs inkGlyphPerCharacter refuses, mounted so the
// question can be put to the face this build actually resolved.
//
// # A superset that was never measured
//
// inkLigatureSeeds is nine pairs chosen from what serif faces commonly carry,
// and the refusal is deliberately a superset: what it has to rule out is a
// fixture string whose glyph count is a question about which optional
// ligatures a particular build shipped with. That argument is about faces in
// general, and it was the whole of what stood behind the list — while the one
// party that can answer for THIS face was already on the line. browser.mjs
// reads CSS.getPlatformFontsForNode for every scanned run in the grid, and a
// glyph count is exactly the answer to "does this face draw 'st' as one
// glyph".
//
// So the pairs are mounted and asked. Two characters drawn as one glyph is a
// refusal this build earns; two drawn as two is a refusal carried for a build
// that is not this one, which is a fair thing to carry and a different thing
// to say. The census in browser.mjs says which, and neither answer is a
// failure: the list stays a superset, and it stops being an unmeasured one.
type inkLigature struct {
	// Pair is the two characters, which is also what the mounted node says.
	Pair string `json:"pair"`
	// Tree is the Text node, as JSON, ready to be mounted alongside the probes.
	Tree string `json:"tree"`
}

// inkLigatureRow builds the mounted question, one Text per refused pair.
//
// Drawn in a scanned run's own declaration rather than in whatever the page
// inherits, because ligation is a property of the resolved FACE and the face
// is what a declaration resolves to. The probe's copy of that declaration is
// the one used: it is the same Style the scanned node carries with the ink
// replaced by black, and browser.mjs already holds every scanned box and its
// probe to being drawn by one family — so a row drawn in it is a row drawn in
// the grid's face, and browser.mjs checks that rather than assuming it.
func inkLigatureRow(style core.Style) []inkLigature {
	out := make([]inkLigature, 0, len(inkLigatureSeeds))
	for _, pair := range inkLigatureSeeds {
		// The declaration whole, minus the box-shaped fields: these are bare
		// Text nodes in a Row and a declared width or ground would be laying
		// out a box rather than asking about a face.
		s := style
		s.Background = ""
		s.Width = ""
		s.Height = ""
		out = append(out, inkLigature{
			Pair: pair,
			Tree: jsonout.Export(&core.Node{
				Type:  "Text",
				Props: map[string]any{"content": pair},
				Style: &s,
			}),
		})
	}
	return out
}

// inkProbeColor is black at the alpha the declared ink carries.
//
// The alpha travels because it is one of the things that can move an element
// onto another rendering path, and because it costs nothing to keep: black at
// any alpha over white composites to a grey, which is the property the whole
// probe rests on.
func inkProbeColor(ink string) (string, error) {
	switch len(ink) {
	case 7:
		return "#000000", nil
	case 9:
		return "#000000" + ink[7:], nil
	}
	return "", fmt.Errorf(
		"the text is declared in %q, which is neither #RRGGBB nor #RRGGBBAA. The "+
			"antialiasing probe repaints it as black at the same alpha so the "+
			"screenshot can be read as greys, and a colour it cannot take apart is a "+
			"declaration it cannot stand in for", ink)
}

// bandRenderBuilders is one entry per band shape, so a shape added here is
// added for every bundled theme.
//
// # The two branches, twice, and one shape that is neither
//
// The plain band with a badge is the baseline the disclosure is compared
// against — the two branches are supposed to be the same band geometrically,
// which is what components.GroupHeader.ControlStyle promises a caller who adds
// a handler. The disclosure with a badge is the tap-target case. Hiding the
// count is the asymmetric side components.bandInsets' `trailing` parameter
// exists for: the control's trailing edge becomes the band's own inset rather
// than the gap before a badge.
//
// The count-hidden band used to exist on the disclosure branch only, which made
// the parity comparison a claim about badged bands and left the OTHER shape's
// two branches uncompared. `pair` groups the two halves of each comparison, so
// the browser pairs what this file says is one band rather than inferring the
// pairing from two booleans — and both pairs are now checked.
//
// # And a shape that exists to move the label
//
// Every shape above puts the label's rect in the same place: hard against the
// band's own leading inset, at a width the same string always takes. The ink
// scan therefore read one rect nine times, so a scan that worked only for a
// label sitting exactly there would have passed nine times over.
//
// The last entry is the one that is somewhere else, and it is three departures
// at once because the point is the rect and not any one of them: a caller's own
// indent through ControlStyle (which is what that field is for — a nested
// band's label — and which must NOT move the tap target), no badge, and a title
// long enough that the run of words is most of the band. If it ever grew long
// enough to wrap, the scan's own row check says so by name rather than
// reporting a label with no ink in it.
var bandRenderBuilders = []struct {
	what        string
	collapsible bool
	// pair names the parity comparison this shape is one half of, or is empty
	// for a shape that is in none. Each named pair must have exactly one plain
	// and one disclosure band — checked in bandRenders, because a pair with
	// one half is a comparison that silently does not run.
	pair string
	// indent is the leading padding this shape's ControlStyle declares, or 0
	// for a shape that declares none and takes the theme's own band inset.
	//
	// Stated here rather than read back off the rendered node, which is the
	// whole point of it: renderBandCase holds the control's rendered padding
	// to this number, so a ControlStyle that stopped reaching the control is a
	// failure here rather than a shape that quietly exercises nothing. See
	// bandRender.Leading for the pixel half of the same claim.
	indent int
	build  func() components.GroupHeader
}{
	{"a plain band", false, "badged", 0, func() components.GroupHeader {
		return components.GroupHeader{Group: bandRenderGroup}
	}},
	{"a disclosure band", true, "badged", 0, func() components.GroupHeader {
		return components.GroupHeader{
			Group: bandRenderGroup, Expanded: true, OnToggle: func() {},
		}
	}},
	{"a plain band, count hidden", false, "unbadged", 0, func() components.GroupHeader {
		return components.GroupHeader{Group: bandRenderGroup, HideCount: true}
	}},
	{"a disclosure band, count hidden", true, "unbadged", 0, func() components.GroupHeader {
		return components.GroupHeader{
			Group: bandRenderGroup, Expanded: true, OnToggle: func() {},
			HideCount: true,
		}
	}},
	{"a plain band, indented, long title, no count", false, "", bandRenderIndent,
		func() components.GroupHeader {
			return components.GroupHeader{
				Group: bandRenderLongGroup, HideCount: true,
				ControlStyle: []core.StyleProp{core.PaddingLeft(bandRenderIndent)},
			}
		}},
}

// The group every rendered band titles. The same one internal/bandfixture
// reads its numbers off, so the two tables are describing one band.
var bandRenderGroup = components.Group{Key: "2026-01", Label: "January 2026", Count: 3}

// The group the odd shape titles: the same band, named at a length a real feed
// produces, so the label's rect is most of the band rather than a short run at
// its leading edge.
//
// Short enough to stay on one line inside bandRenderWidth at a caption size —
// the scan reads three rows of ONE line box — and long enough that its rect
// bears no resemblance to the other four.
var bandRenderLongGroup = components.Group{
	Key: "2026-01-long", Label: "January 2026, week by week", Count: 3,
}

// How far the odd shape indents its control, in device-independent pixels.
//
// Larger than the band's own horizontal inset in every bundled theme, so the
// label's rect is unmistakably somewhere the other shapes never put it, and
// small enough that the words still have most of the band to occupy.
const bandRenderIndent = 40

// bandRenderWidth is the page the bands are laid out in.
//
// Wide enough that the label has slack to grow into — the grow weight is the
// whole subject, and a band squeezed to its content would answer the tap-target
// question with "yes, because there was nowhere else to go".
const bandRenderWidth = 320

// bandRenderThemes is the set of palettes the grid mounts every shape through:
// the bundled ones, and one derived from a bundled one that differs in the only
// way none of them do.
//
// # What three palettes were not varying
//
// The five shapes differ from each other in geometry, and every theme paints
// them the same way: what a bundled theme changes is the PALETTE. Every one of
// them sets Typography.Caption to a normal weight at twelve or thirteen points
// with no line height of its own, so every band in the grid has a label line
// box of about the same height, and every fraction of it lands in about the
// same place.
//
// That matters because the ink scan is three fractions of a line box measured
// against a font's x-height band, and the whole of what makes those fractions
// safe is the relationship between the two. A theme with a taller caption tier
// — a larger size, an explicit LineHeight — moves the band inside the box, and
// nothing in the grid was ever laid out in one. Three palettes at one type
// scale is a scan that has only ever been asked one question about metrics.
//
// # The derived theme
//
// A copy of a bundled theme with its caption tier changed and nothing else, so
// the difference between it and the theme it came from is exactly the thing
// being varied. Both departures at once, because they move the band by
// different mechanisms and a check has no reason to meet them separately:
// FontSize scales the glyphs (and with them the x-height the band is), and
// LineHeight adds leading around them without touching either — which is the
// one that pushes the x-height band DOWN inside the box while leaving the
// fractions where they were.
//
// It is a real theme by construction rather than a hand-written palette: the
// contrast the ink scan needs, the pill's own ink, every colour the other
// checks read, all come from the theme it copies.
func bandRenderThemes() map[string]*core.Theme {
	out := map[string]*core.Theme{}
	for name, t := range core.BundledThemes() {
		out[name] = t
	}
	base, ok := out[bandRenderTallBase]
	if !ok {
		fatal("bandRenderThemes derives its tall-caption theme from %q and "+
			"core.BundledThemes has no such theme", bandRenderTallBase)
	}
	tall := *base
	tall.Typography.Caption.FontSize = base.Typography.Caption.FontSize + 6
	tall.Typography.Caption.LineHeight = int(base.Typography.Caption.FontSize) + 16
	// The derived theme has to actually differ, or it is a fourth copy of the
	// third and the grid pays for twenty bands to ask fifteen questions. Both
	// halves, because either alone would leave the other unvaried.
	if tall.Typography.Caption.FontSize == base.Typography.Caption.FontSize ||
		tall.Typography.Caption.LineHeight == base.Typography.Caption.LineHeight {
		fatal("the tall-caption theme's Caption is %gpx over a line height of %d and "+
			"%s's is %gpx over %d — they have to differ in BOTH, which is the only "+
			"reason this theme is in the grid: a size scales the glyphs and with them "+
			"the x-height band, and leading moves that band inside the line box "+
			"without touching it",
			tall.Typography.Caption.FontSize, tall.Typography.Caption.LineHeight,
			bandRenderTallBase, base.Typography.Caption.FontSize,
			base.Typography.Caption.LineHeight)
	}
	// And that it differs in NOTHING ELSE, which is the other half of the same
	// sentence and the half nothing was holding.
	//
	// "A copy of a bundled theme with its caption tier changed and nothing else"
	// is what makes this a fourth palette in name and a second type scale in
	// fact — and it is what the grid's account of itself rests on. The bands
	// read captions, so a theme that also moved a colour would be varying two
	// things at once through checks that cannot tell them apart: the ink scan
	// would be looking at a new palette and the metric question would be
	// answered by whichever of the two the failure happened to name.
	//
	// Reflective, and over the whole Theme, so a tier or a role added to
	// core.Theme is covered without anybody remembering. A field this file has
	// never heard of is exactly the kind that would be copied silently.
	drifted := themeDifferences(reflect.ValueOf(*base), reflect.ValueOf(tall), "")
	allowed := map[string]bool{}
	for _, path := range bandRenderTallVaries {
		allowed[path] = false
	}
	varies, off := []string{}, []string{}
	for _, path := range drifted {
		if _, ok := allowed[path]; ok {
			allowed[path] = true
			varies = append(varies, path)
		} else {
			off = append(off, path)
		}
	}
	if len(off) > 0 {
		fatal("the tall-caption theme is derived from %s and differs from it at %v, "+
			"which is not one of %v.\n\n"+
			"It is in this grid to be one palette at two type scales — the bands read "+
			"captions, so that is the one departure their checks can attribute. A "+
			"second difference makes it a fourth palette, and every failure it "+
			"produces names whichever of the two changes the check happened to be "+
			"looking at.", bandRenderTallBase, off, bandRenderTallVaries)
	}
	// And that the walk found the departure it is supposed to permit. Two lines
	// above have just insisted the caption tier differs in both size and
	// leading; a walk that reports no difference at all is not agreeing with
	// them, it is not looking — and "it differs in nothing else" would then be
	// a sentence produced by a function that never returns anything.
	if len(varies) == 0 {
		fatal("themeDifferences finds the tall-caption theme identical to %s at every "+
			"one of %v, and the two checks above have just held it to differing there "+
			"in both size and leading. The walk that says this theme varies nothing but "+
			"its caption tier is reporting that it varies nothing, which is not the "+
			"same claim", bandRenderTallBase, bandRenderTallVaries)
	}
	// And that every path the list permits is one the theme actually takes.
	//
	// The two directions are different faults. Above: a departure nobody
	// permitted, which is the theme varying more than the grid can attribute.
	// Here: a permission nothing uses, which is the list claiming a departure
	// that is not happening — and a list with slack in it is one that would go
	// on passing after the derivation above stopped making the change, because
	// "differs only where permitted" is satisfied by differing nowhere.
	//
	// And an unused permission has two quite different causes, which used to
	// arrive as one message pointing at the derivation. A path can go unmoved
	// because the assignments above stopped making that change, or because the
	// path does not name a leaf of core.Theme at all — a field renamed under
	// it, or a typo in this list — in which case the derivation is fine and the
	// reader has been sent to look at it. They separate on the struct: the
	// first names a real leaf, the second names nothing. Same shape as
	// pinfixture's deleted-versus-reworded pair, and the same reason for
	// bothering — the diagnosis is what the message is for.
	// The leaves, with the near-miss threshold measured over them. See
	// themeLeafSet: the number is a fact about these names and travels with
	// them, rather than being a constant about whichever struct was in front of
	// whoever wrote it.
	set := themeLeaves(reflect.ValueOf(*base))
	leaves := set.leaves
	unused, unknown := []string{}, []string{}
	for _, path := range bandRenderTallVaries {
		if allowed[path] {
			continue
		}
		if leaves[path] {
			unused = append(unused, path)
		} else {
			unknown = append(unknown, path)
		}
	}
	if len(unknown) > 0 {
		fatal("bandRenderTallVaries permits the tall-caption theme to differ at %v, "+
			"and core.Theme has no leaf by that name.\n\n"+
			"themeDifferences walks the struct and spells a leaf by its dotted path "+
			"from the Theme down, so a path nothing in that walk can ever produce is "+
			"a permission that can never be spent — and the failure it produces on "+
			"its own is \"the theme does not differ there\", which sends the reader "+
			"to the two assignments above when what happened was a rename in "+
			"core.Theme or a typo here. The derivation is not the thing to look "+
			"at.\n\n%s.", unknown, themeNearMiss(set, unknown[0]))
	}
	if len(unused) > 0 {
		fatal("bandRenderTallVaries permits the tall-caption theme to differ from %s at "+
			"%v — leaves core.Theme does have — and it does not differ there.\n\n"+
			"The list is the exact set of leaves this theme is a copy-with-one-change "+
			"at, so a path in it that nothing moves is a permission standing open for "+
			"a change nobody makes. The path is real, so this is the derivation above "+
			"having stopped making it — in which case this is a fourth copy of %s and "+
			"the grid pays for twenty bands to ask fifteen questions.",
			bandRenderTallBase, unused, bandRenderTallBase)
	}

	out[bandRenderTallName] = &tall
	return out
}

// Where the tall-caption theme is allowed to differ from the theme it copies:
// the exact leaves, in the spelling themeDifferences produces.
//
// # Why a set of leaves and not the tier's prefix
//
// It was "Typography.Caption.", matched with strings.HasPrefix, and the two
// agreed by construction: the prefix was compared against paths this same file
// produces, so a typo in it made every caption difference "off" and failed
// loudly. What it could not say is WHICH caption fields are allowed to move.
//
// core.Theme's Caption is a core.Style — a struct with a great many leaves —
// and this theme moves two of them. A third changed alongside them satisfied a
// prefix perfectly, and the comment beside the derivation went on naming two:
// the grid would be varying a colour, a weight or a letter-spacing through
// checks whose account of themselves says it varies a type scale. The
// permission has to be as narrow as the claim, so it is the claim.
//
// Both directions are held below — nothing outside this list may differ, and
// everything in it must. And the second of those has two causes that used to
// arrive as one message: a path that names a real leaf and does not move is the
// derivation having stopped making the change, and a path that names no leaf at
// all is a rename in core.Theme or a typo here. The first sends a reader to the
// two assignments; the second sends them to the wrong place, which is why they
// are told apart.
var bandRenderTallVaries = []string{
	"Typography.Caption.FontSize",
	"Typography.Caption.LineHeight",
}

// themeDifferences is every leaf path at which two themes disagree.
//
// Structs are walked; anything else is compared whole with reflect.DeepEqual,
// which is what makes this cover a field nobody here has read: a new tier is a
// struct and gets walked, a new role is a string and gets compared.
//
// The path is dotted from the Theme down — "Typography.Caption.FontSize" — so a
// caller can ask about a subtree with a prefix.
func themeDifferences(a, b reflect.Value, path string) []string {
	if a.Kind() == reflect.Struct {
		var out []string
		for i := 0; i < a.NumField(); i++ {
			name := a.Type().Field(i).Name
			out = append(out,
				themeDifferences(a.Field(i), b.Field(i), path+name+".")...)
		}
		return out
	}
	if reflect.DeepEqual(a.Interface(), b.Interface()) {
		return nil
	}
	// The trailing dot goes: this is a leaf, and the prefix a caller compares
	// against carries its own.
	return []string{strings.TrimSuffix(path, ".")}
}

// How far from a real leaf a path may be and still be reported as a probable
// typo of it.
//
// # The number, and where it comes from
//
// The near-miss answer used to be case-insensitive equality and nothing else,
// which reaches exactly one kind of slip. A transposition, a doubled letter and
// a singular written for a plural are all typos and all further out than that,
// and every one of them was reported as "a name that was never right" — a
// confident diagnosis of a deliberate choice, made about a slip.
//
// A distance covers all three and needs a threshold, and the honest way to pick
// one is to measure the struct it is asked about rather than to pick a number
// that looks reasonable. Two measurements, both in
// TestThemeNearMissThresholdIsDerivedFromCoreTheme:
//
//	the smallest distance between      1 — Spacing.XS and Spacing.XL
//	two sibling leaves
//	the most siblings within 1 of      1 — so an answer at this distance is
//	any leaf                               at most two names
//	the most siblings within 2         4 — so the next threshold out turns
//	                                       the answer into a list of five
//
// The first of those is the interesting one and it settles the shape of the
// message rather than the size of the number: core.Theme HAS a pair of leaves
// one edit apart, so at any threshold at all a path can be one edit from two
// real names, and "this is what was meant" is a claim this function is not in a
// position to make. It names every candidate instead.
//
// The second and third make 1 the threshold: at one edit the answer stays a
// pair, at two it becomes five, and a wall of names is the thing this whole
// function exists not to print.
//
// # What this constant is now, and what it is not
//
// It is core.Theme's answer, pinned. The derivation itself lives in
// themeLeaves and runs over whatever leaves it is handed — see themeLeafSet
// for why the number had to stop being a constant the function read.
const themeNearMissEdits = 1

// How far a threshold derived from a struct is allowed to reach.
//
// themeLeaves searches upward for the largest distance that keeps an answer to
// a pair of names, and a sparse struct has no such distance: two leaves under
// one parent are never crowded, however far the search goes. That is not a
// licence to report a name five edits away as "probably what was meant" — past
// a few characters the claim stops being about typing and becomes a guess
// about intent, which is the claim this function was rewritten not to make.
//
// This is a judgement and not a measurement, and it is the only one left here.
// Three is where a slip stops being one slip: a transposition plus a case
// change is two, a doubled letter in a word already misspelt is two, and
// nothing that reads as a typing accident to a person is further out than
// three without being a different word.
const themeNearMissReach = 3

// themeLeafSet is a struct's leaf paths together with the near-miss threshold
// measured over THEM.
//
// # The threshold was derived from one struct and the function took any
//
// themeNearMissEdits is 1 because of three readings of core.Theme's own field
// names, and themeNearMiss used to take a bare map[string]bool and read the
// constant. The only caller passes core.Theme's leaves, so the derivation held
// for the one use — and nothing in the signature or in the constant said which
// struct the number was about. A second caller with a different struct would
// have got a threshold measured against somebody else's field names, silently,
// and the message would have gone on citing Spacing.XS and Spacing.XL as its
// reason for hedging about a struct that has neither.
//
// So the measurement travels with the leaves. themeLeaves does it once per
// struct — the cost is a per-parent quadratic and the callers build one set
// and reuse it — themeNearMiss reads `edits` off the set it is handed, and the
// ambiguity sentence names the closest pair THIS set has.
type themeLeafSet struct {
	// leaves is every dotted path themeLeafPaths produced, as a set.
	leaves map[string]bool

	// edits is the largest distance at which an answer over these names is
	// still at most a pair, floored at 1 and capped at themeNearMissReach.
	edits int

	// closest is the closest pair of SIBLING leaves and closestD how far apart
	// they are. This is what decides the shape of the message rather than the
	// size of the threshold: a set whose closest siblings are three apart can
	// report one candidate and call it the answer, and one with a pair at
	// distance 1 cannot, at any threshold. -1 when no parent has two leaves.
	closest  string
	closestD int

	// crowdAt is the most siblings any leaf has within `edits`, and crowdBeyond
	// the most within one more — the two readings that bound the threshold from
	// each side. atWorst and beyondWorst name the leaves they were measured at,
	// so a test that disagrees with the derivation can say where.
	crowdAt, crowdBeyond int
	atWorst, beyondWorst string

	// cappedByReach is true when the search stopped at themeNearMissReach
	// rather than at this struct's own crowding — that is, when a judgement
	// and not these names decided the threshold.
	//
	// The two are different facts with one number in front of them, and
	// nothing recorded which. core.Theme measures 1 against a ceiling of 3, so
	// the ceiling is slack over everything this repository asks about and the
	// distinction costs nothing here; a sparser struct lands ON the ceiling and
	// gets a threshold that is a judgement about typing accidents wearing a
	// measurement's clothes. The message says which, because "nothing is within
	// 3 edits of it" invites a reader to try 4, and whether that is available
	// is exactly what this flag knows.
	cappedByReach bool

	// afforded is how wide a threshold these names would carry with the ceiling
	// taken off, and affordedOpen says the search for it ran out of distances
	// to try rather than finding a crowd.
	//
	// # The flag says which stopped the search and not what the other one was
	//
	// cappedByReach turns "nothing is within 3 edits of it" into "and 3 is a
	// judgement, not a measurement" — which is the half a reader needs to know
	// they are looking at a constant. The half it does not carry is what the
	// constant is costing them, and the message was stating it anyway: "these
	// names are far enough apart to carry a wider one" is a claim about
	// crowding at 4, and the search stops at 3 without ever asking. A capped
	// set whose names ARE crowded at 4 got that sentence too, and it was
	// wrong — the ceiling and the crowding stop in the same place there, and a
	// reader was being sent to raise a constant that would find nothing.
	//
	// So the same upward search continues past the ceiling, bounded by the
	// longest leaf name: an edit distance is never more than the longer of the
	// two strings, so past that every sibling pair is inside the threshold and
	// the answer stops moving. A search that gets there without crowding is one
	// where no width crowds these names at all — a set with no parent holding
	// three leaves — and that is reported as the shape it is rather than as a
	// number that would read like a measurement.
	//
	// Equal to edits whenever the ceiling was not what stopped the search,
	// because then the crowding already answered the question.
	afforded     int
	affordedOpen bool
}

// themeLeaves reads a struct's leaves and measures the near-miss threshold over
// them. See themeLeafSet.
func themeLeaves(v reflect.Value) themeLeafSet {
	return themeLeafSetOf(themeLeafPaths(v, ""))
}

// themeLeafSetOf is the measurement, over dotted paths rather than over a
// struct.
//
// Split from themeLeaves so the derivation can be put in front of names that
// are not core.Theme's. That is the one thing a test over core.Theme alone
// cannot show: one struct gives one answer, and a constant with a walk in
// front of it would give the same one.
func themeLeafSetOf(paths []string) themeLeafSet {
	set := themeLeafSet{leaves: map[string]bool{}, closestD: -1}

	// Grouped by parent, because that is the only comparison themeNearMiss
	// makes: a slip under Typography.Caption is answered by Typography.Caption's
	// own names, and a leaf three parents away is not a candidate however close
	// it spells.
	byParent := map[string][]string{}
	for _, path := range paths {
		set.leaves[path] = true
		cut := strings.LastIndex(path, ".") + 1
		byParent[path[:cut]] = append(byParent[path[:cut]], path[cut:])
	}

	// Every sibling pair's distance, once. The two crowd readings below and the
	// closest-pair reading are three questions about the same numbers, and
	// recomputing them per question is the whole cost of this function.
	type group struct {
		parent string
		names  []string
		d      [][]int
	}
	groups := make([]group, 0, len(byParent))
	parents := make([]string, 0, len(byParent))
	for parent := range byParent {
		parents = append(parents, parent)
	}
	sort.Strings(parents)
	// The longest leaf name, which bounds every edit distance these names can
	// produce. See afforded.
	span := 0
	for _, parent := range parents {
		names := byParent[parent]
		sort.Strings(names)
		for _, n := range names {
			if len(n) > span {
				span = len(n)
			}
		}
		low := make([]string, len(names))
		for i, n := range names {
			low[i] = strings.ToLower(n)
		}
		d := make([][]int, len(names))
		for i := range names {
			d[i] = make([]int, len(names))
		}
		for i := range names {
			for j := 0; j < i; j++ {
				d[i][j] = themeEditDistance(low[i], low[j])
				d[j][i] = d[i][j]
				if set.closestD < 0 || d[i][j] < set.closestD {
					set.closestD = d[i][j]
					set.closest = parent + names[j] + " and " + parent + names[i]
				}
			}
		}
		groups = append(groups, group{parent: parent, names: names, d: d})
	}

	// The most OTHER siblings any leaf has within a distance, and the leaf that
	// has them.
	crowd := func(within int) (int, string) {
		most, worst := 0, ""
		for _, g := range groups {
			for i := range g.names {
				n := 0
				for j := range g.names {
					if i != j && g.d[i][j] <= within {
						n++
					}
				}
				if n > most {
					most, worst = n, g.parent+g.names[i]
				}
			}
		}
		return most, worst
	}

	// The largest threshold that keeps an answer to a pair of names, searched
	// upward from the floor.
	//
	// One is the floor rather than a candidate: below it there is no threshold
	// at all, only exact match, and a struct crowded enough that even one edit
	// reaches several siblings gets 1 with a longer list — which the message
	// prints, because "several names are one edit from this" is a true and
	// useful thing to say and a silently narrowed threshold is not.
	set.edits = 1
	// Which of the two stopped the search, recorded where it is known rather
	// than inferred afterwards from the number. Inferring it would need
	// crowdBeyond, and crowdBeyond is measured at edits+1 for BOTH exits — a
	// struct whose crowding stopped the search exactly at the ceiling would be
	// indistinguishable from one the ceiling stopped. See cappedByReach.
	set.cappedByReach = true
	for set.edits < themeNearMissReach {
		n, worst := crowd(set.edits + 1)
		if n > 1 {
			set.crowdBeyond, set.beyondWorst = n, worst
			set.cappedByReach = false
			break
		}
		set.edits++
	}
	set.crowdAt, set.atWorst = crowd(set.edits)
	if set.beyondWorst == "" {
		set.crowdBeyond, set.beyondWorst = crowd(set.edits + 1)
	}
	// And, when the ceiling is what stopped the search, how far these names
	// would have gone without it. See afforded: the flag says which of the two
	// bound the threshold, and this is what the other one was.
	set.afforded = set.edits
	if set.cappedByReach {
		for set.afforded <= span {
			if n, _ := crowd(set.afforded + 1); n > 1 {
				break
			}
			set.afforded++
		}
		// Past the longest name every pair is inside the threshold, so a search
		// that got there found no crowd at any width — which is a fact about the
		// SHAPE of the set (no parent with three leaves) and not a number.
		set.affordedOpen = set.afforded > span
	}
	return set
}

// themeEditDistance is the optimal string alignment distance — an insert, a
// delete, a substitution or a swap of two ADJACENT characters, each costing
// one.
//
// The transposition is why this is not plain Levenshtein, and it is the reason
// the threshold can stay at 1: swapping two letters is the most common typo
// there is and costs two edits without it, which would have needed a threshold
// of 2 and the five-name answer the note above measures.
//
// Case is folded by the caller, so a leaf differing only in case is at distance
// zero and is reported as the separate, stronger thing it is.
func themeEditDistance(a, b string) int {
	d := make([][]int, len(a)+1)
	for i := range d {
		d[i] = make([]int, len(b)+1)
		d[i][0] = i
	}
	for j := 0; j <= len(b); j++ {
		d[0][j] = j
	}
	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			best := d[i-1][j-1] + cost
			if v := d[i-1][j] + 1; v < best {
				best = v
			}
			if v := d[i][j-1] + 1; v < best {
				best = v
			}
			// The adjacent swap, which the two single-character edits above
			// would otherwise charge twice for.
			if i > 1 && j > 1 && a[i-1] == b[j-2] && a[i-2] == b[j-1] {
				if v := d[i-2][j-2] + 1; v < best {
					best = v
				}
			}
			d[i][j] = best
		}
	}
	return d[len(a)][len(b)]
}

// themeNearMiss is what a path that names no leaf was probably trying to say.
//
// A core.Style has seventy leaves under it and a core.Theme has 872; printing
// them all is the wall this repository keeps deciding not to produce. So the
// answer is narrowed to the ones a typo actually reaches: a leaf under the same
// parent that differs only in case, or one within the set's own threshold.
//
// The two are reported separately because they are different strengths of
// claim. A name that differs only in case is almost certainly the one that was
// meant; a name an edit away is a candidate, and this set may have sibling
// leaves that close to each other, in which case it is not the only one — see
// themeLeafSet, where the measurements are, and themeNearMissEdits for what
// they come to for core.Theme.
//
// With nothing within the threshold, what is said is how many leaves the
// parent has and what an edit covers, rather than a verdict on whether the
// name was ever right. That verdict was the previous version's and it was
// stated with more confidence than case-insensitive equality could support.
func themeNearMiss(set themeLeafSet, path string) string {
	cut := strings.LastIndex(path, ".") + 1
	parent, want := path[:cut], strings.ToLower(path[cut:])
	same, near, under := []string{}, []string{}, 0
	for leaf := range set.leaves {
		if !strings.HasPrefix(leaf, parent) {
			continue
		}
		under++
		switch d := themeEditDistance(strings.ToLower(leaf[cut:]), want); {
		case d == 0:
			same = append(same, leaf)
		case d <= set.edits:
			near = append(near, leaf)
		}
	}
	sort.Strings(same)
	sort.Strings(near)
	// "one edit" reads better than "1 edits" and the threshold is 1 for
	// core.Theme, so the singular is the sentence this prints today; the plural
	// is what a sparser struct would get.
	edits := "one edit"
	if set.edits != 1 {
		edits = fmt.Sprintf("%d edits", set.edits)
	}
	if len(same) > 0 {
		return fmt.Sprintf("%s differs from it only in case, and is probably what "+
			"was meant", strings.Join(same, " and "))
	}
	if len(near) == 1 {
		return fmt.Sprintf("%s is within %s of it — a letter added, dropped or "+
			"changed, or two adjacent letters swapped — and is probably what was meant",
			near[0], edits)
	}
	if len(near) > 1 {
		return fmt.Sprintf("%s are each within %s of it, and which was meant is not "+
			"something this can say: this struct has sibling leaves %d apart (%s), so "+
			"a name at this distance can belong to more than one of them",
			strings.Join(near, " and "), edits, set.closestD, set.closest)
	}
	// Where the threshold came from, on the one arm that invites the reader to
	// wonder whether a wider one would have found something — and, when the
	// answer is yes, how much wider. See afforded: "these names could carry
	// more" was being said without the width, and on a set crowded one step
	// past the ceiling it was being said wrongly.
	why := ""
	if set.cappedByReach {
		switch {
		case set.affordedOpen:
			why = fmt.Sprintf(" That threshold is themeNearMissReach and not this "+
				"struct's own crowding: no parent here holds three leaves, so no "+
				"width crowds these names at all and the ceiling is the only thing "+
				"bounding the answer. %d edits is where a slip stops reading as one "+
				"slip.", themeNearMissReach)
		case set.afforded > set.edits:
			why = fmt.Sprintf(" That threshold is themeNearMissReach and not this "+
				"struct's own crowding: these names would carry %d before a name at "+
				"that distance could belong to more than one of them, and %d edits is "+
				"where a slip stops reading as one slip.", set.afforded,
				themeNearMissReach)
		default:
			why = fmt.Sprintf(" That threshold is themeNearMissReach, and this "+
				"struct's own crowding stops in the same place: %d is the widest "+
				"these names carry either way, so raising the ceiling would find "+
				"nothing here.", set.afforded)
		}
	}
	return fmt.Sprintf("nothing under %s is within %s of it, case ignored, and "+
		"that prefix has %d leaves. One edit covers a letter added, dropped or "+
		"changed and two adjacent letters swapped — a doubled letter, a singular for "+
		"a plural, a transposition — so a slip of that kind is ruled out. Two "+
		"independent slips are not, and neither is a name that was never right.%s",
		strings.TrimSuffix(parent, "."), edits, under, why)
}

// themeLeafPaths is every leaf path a core.Theme has, in the spelling
// themeDifferences produces.
//
// The same walk with the comparison taken out, so the two cannot disagree about
// what a path looks like. It exists to tell two failures apart that arrived as
// one: see where it is called.
func themeLeafPaths(v reflect.Value, path string) []string {
	if v.Kind() == reflect.Struct {
		var out []string
		for i := 0; i < v.NumField(); i++ {
			out = append(out,
				themeLeafPaths(v.Field(i), path+v.Type().Field(i).Name+".")...)
		}
		return out
	}
	return []string{strings.TrimSuffix(path, ".")}
}

const (
	// The theme the tall-caption one is a copy of, and the name it goes under.
	// Named so a failure in the grid says which of the two it is, and so the
	// pair reads as "this palette, at another type scale" rather than as a
	// fourth palette.
	bandRenderTallBase = "DefaultTheme"
	bandRenderTallName = "DefaultTheme+tallCaption"
)

func bandRenders() ([]bandRender, []inkProbe, []inkLigature) {
	byName := bandRenderThemes()
	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)

	// Each parity pair must have exactly one plain band and one disclosure
	// band. The comparison in check 10 runs only where both halves are present,
	// so a pair with one half is a comparison that does not happen and reports
	// nothing — the same silence an opt-in list produces one level up.
	branches := map[string][]string{}
	for _, b := range bandRenderBuilders {
		if b.pair == "" {
			continue
		}
		branch := "plain"
		if b.collapsible {
			branch = "disclosure"
		}
		branches[b.pair] = append(branches[b.pair], branch)
	}
	for pair, got := range branches {
		sort.Strings(got)
		if len(got) != 2 || got[0] != "disclosure" || got[1] != "plain" {
			fatal("bandRenderBuilders' pair %q holds %v — a parity pair is one plain "+
				"band and one disclosure band, and check 10 compares only the pairs it "+
				"finds both halves of", pair, got)
		}
	}

	out := make([]bandRender, 0, len(names)*len(bandRenderBuilders))
	// The probes, deduplicated by the declaration they reproduce and kept in
	// the order they were first met so the transcript is stable. See inkProbe.
	probes := []inkProbe{}
	seen := map[string]int{}
	// Every shape's rendered control padding, per theme, so the indented shape
	// can be held to having actually moved something.
	padByTheme := map[string]map[string]float64{}
	for _, name := range names {
		padByTheme[name] = map[string]float64{}
		for _, b := range bandRenderBuilders {
			c, built, err := renderBandCase(
				name, byName[name], b.what, b.collapsible, b.indent, b.build())
			if err != nil {
				fatal("%v", err)
			}
			c.Pair = b.pair
			for _, p := range built {
				at, ok := seen[p.Key]
				if !ok {
					at = len(probes)
					seen[p.Key] = at
					probes = append(probes, p)
				} else {
					// A second declaration answered by the same probe. Both
					// names ride along, because a coloured probe invalidates
					// every box that shares it and the failure has to say so.
					probes[at].What += "; " + p.What
				}
				if p.subject == inkProbeLabel {
					c.LabelProbe = p.Key
				} else {
					c.BadgeProbe = p.Key
				}
			}
			padByTheme[name][b.what] = c.ControlPadLeft
			out = append(out, c)
		}
	}

	// The indented shape's control padding must differ from the shapes that
	// declare none, in every theme.
	//
	// renderBandCase already holds it to the number the builder declared, which
	// says the ControlStyle reached the control. This says the number is worth
	// declaring: an indent that happened to equal the theme's own band inset
	// would put the label exactly where the other four shapes put it, and the
	// shape that exists to give the ink scan a rect it has not met would be
	// handing it the same rect again.
	for _, name := range names {
		for _, b := range bandRenderBuilders {
			if b.indent == 0 {
				continue
			}
			for _, other := range bandRenderBuilders {
				if other.indent != 0 {
					continue
				}
				if padByTheme[name][b.what] == padByTheme[name][other.what] {
					fatal("%s: %q indents its control to %g and %q renders the same "+
						"%g without asking. The indented shape is in this grid to put "+
						"the label's rect somewhere the others never do, and at this "+
						"inset it is the same rect — raise bandRenderIndent above every "+
						"bundled theme's band inset", name, b.what,
						padByTheme[name][b.what], other.what,
						padByTheme[name][other.what])
				}
			}
		}
	}

	// And the question the refusal in inkGlyphPerCharacter has been making
	// without asking anybody. See inkLigatureRow.
	//
	// Drawn in the first probe's declaration. Any of them would do — every
	// scanned box and the probe that answers for it are held to one resolved
	// family in browser.mjs, so the grid has one face until that check says
	// otherwise — and the first is the one that does not need a rule for
	// choosing. An empty probe table means the grid reads no text at all,
	// which is a failure browser.mjs already reports by name.
	var ligatures []inkLigature
	if len(probes) > 0 {
		ligatures = inkLigatureRow(probes[0].style)
	}
	return out, probes, ligatures
}

// renderBandCase renders one band through one theme and locates the nodes the
// browser measures.
//
// An error rather than fatal, for the same reason renderWidgetCase returns one:
// a band whose rendered shape has changed must stop a `go run` and fail a
// `go test`, and os.Exit does the first and takes the whole test binary down
// doing the second.
func renderBandCase(name string, theme *core.Theme, what string, collapsible bool,
	indent int, band components.GroupHeader) (bandRender, []inkProbe, error) {

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
		return bandRender{}, nil, fmt.Errorf(
			"%s/%s: the page box rendered %d children, want the band alone",
			name, what, len(page.Children))
	}
	row := page.Children[0]
	if row.Style == nil {
		return bandRender{}, nil, fmt.Errorf("%s/%s: the band Row rendered with no Style",
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
		return bandRender{}, nil, fmt.Errorf(
			"%s/%s: the band Row declares no background. Its fill is why a band may span "+
				"its container edge to edge, and with none there is nothing for the "+
				"browser to read back", name, what)
	}
	if c.Fill == c.Page {
		return bandRender{}, nil, fmt.Errorf(
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
				return bandRender{}, nil, fmt.Errorf(
					"%s/%s: the band has more than one growing child (%d and %d) — "+
						"the badge is pinned by a single grow weight, and two of them "+
						"share the slack", name, what, grow, i)
			}
			grow = i
		}
	}
	if grow < 0 {
		return bandRender{}, nil, fmt.Errorf(
			"%s/%s: no child of the band grows — the whole tap-target question is "+
				"about padding on a *stretched* child", name, what)
	}
	if grow != 0 {
		return bandRender{}, nil, fmt.Errorf(
			"%s/%s: the growing child is at index %d and the leading edge is index 0 — "+
				"the label is supposed to come first", name, what, grow)
	}

	if len(row.Children) > 2 {
		return bandRender{}, nil, fmt.Errorf(
			"%s/%s: the band rendered %d children and this fixture reads at most two",
			name, what, len(row.Children))
	}
	if len(row.Children) == 2 {
		c.Badge = "root/0/1"
		badge := row.Children[1]
		if badge.Style == nil || badge.Style.Background == "" {
			return bandRender{}, nil, fmt.Errorf(
				"%s/%s: the count badge declares no background. It is a pill, and a "+
					"pill with no fill is a number sitting on the band — which is a "+
					"different widget and one no pixel here could tell apart from a "+
					"badge that failed to paint", name, what)
		}
		c.BadgeFill = badge.Style.Background
		c.BadgePadLeft = float64(badge.Style.Padding.Left)
		c.BadgePadRight = float64(badge.Style.Padding.Right)
		c.BadgeInk = badge.Style.TextColor
		// The sample point is half of this, so a pill with less than two
		// device-independent pixels of leading padding has no interior for the
		// browser to read: the sample lands on the corner's antialiasing or on
		// the first digit, and either one fails while describing a colour
		// rather than a cause.
		if c.BadgePadLeft < 2 {
			return bandRender{}, nil, fmt.Errorf(
				"%s/%s: the count pill's leading padding is %gpx. The browser samples "+
					"the pill's fill half a padding in — the leading edge itself is the "+
					"apex of a 999-radius curve and every pixel there is a blend — so a "+
					"padding this small leaves nowhere inside the pill that is fill and "+
					"not digit", name, what, c.BadgePadLeft)
		}
		if c.BadgeFill == c.Fill {
			return bandRender{}, nil, fmt.Errorf(
				"%s/%s: the badge and the band behind it are both %s, so a pixel taken "+
					"inside the pill agrees with either answer and the paint check "+
					"cannot fail", name, what, c.BadgeFill)
		}
		// The digits, which are the other half of a count and were unread until
		// the scan below existed. Everything asserted about the label's ink is
		// asserted about this one, and for the same reasons.
		if c.BadgeInk == "" {
			return bandRender{}, nil, fmt.Errorf(
				"%s/%s: the count pill declares no text colour, so what a browser draws "+
					"the number in is whatever it inherits and there is nothing to read "+
					"back. components.Badge resolves an ink against its own fill "+
					"(Variant.Ink) precisely so the digits are legible on it — a pill "+
					"that stopped declaring one would still lay out identically",
				name, what)
		}
		if c.BadgeInk == c.BadgeFill {
			return bandRender{}, nil, fmt.Errorf(
				"%s/%s: the count's digits and the pill behind them are both %s, so the "+
					"number is invisible and a scan that found the digits would be "+
					"finding the pill", name, what, c.BadgeInk)
		}
		// The scan's window is what lies between the two paddings, and a pill
		// whose paddings meet has no such window: the digits' own box would be
		// empty and "no digit found" would be a fact about the arithmetic
		// rather than about the paint.
		if c.BadgePadRight < 2 {
			return bandRender{}, nil, fmt.Errorf(
				"%s/%s: the count pill's trailing padding is %gpx. The digit scan reads "+
					"the box between the two paddings — the pill's own edges are the "+
					"apexes of a 999-radius curve and every pixel there is a blend with "+
					"the band behind it — so a padding this small leaves the scan reading "+
					"the corner rather than the number", name, what, c.BadgePadRight)
		}
	}

	// Where the insets are, and therefore what a finger lands on. On the plain
	// branch the growing child carries them itself; as a disclosure it is a
	// wrapper with no chrome at all and the button inside it is the target.
	//
	// controlNode and labelNode are the two the branches disagree about, kept
	// so the code after them can ask one question of both: the control's own
	// leading inset, and the antialiasing probe the label's declaration needs.
	growing := row.Children[grow]
	var controlNode, labelNode *core.Node
	if collapsible {
		if len(growing.Children) != 1 {
			return bandRender{}, nil, fmt.Errorf(
				"%s/%s: the heading wrapper holds %d children, want the button alone",
				name, what, len(growing.Children))
		}
		control := growing.Children[0]
		if control.Style == nil {
			return bandRender{}, nil, fmt.Errorf(
				"%s/%s: the band's button rendered with no Style", name, what)
		}
		if control.Props["onClick"] == nil {
			return bandRender{}, nil, fmt.Errorf(
				"%s/%s: the node inside the heading wrapper has no handler, so it is "+
					"not the thing a press lands on", name, what)
		}
		// The wrapper must be geometrically invisible, or "the button fills the
		// wrapper" would be a claim about a box with chrome of its own and the
		// tap target would stop short of the band by however much it carries.
		if p := growing.Style.Padding; p != (core.EdgeInsets{}) {
			return bandRender{}, nil, fmt.Errorf(
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
			return bandRender{}, nil, fmt.Errorf(
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
			return bandRender{}, nil, fmt.Errorf(
				"%s/%s: the button's children are not a chevron and the group's label — "+
					"the height comparison in check 10 is an equation over exactly those two",
				name, what)
		}
		for _, child := range control.Children {
			if child.Props["content"] == band.Group.Label && child.Style != nil {
				c.LabelInk = child.Style.TextColor
				labelNode = child
			}
		}
		controlNode = control
		// The control's first child, whatever it is. On this branch it is the
		// chevron, and the browser holds ITS leading edge to the control's own
		// padding — `label.x - control.x` here is the padding plus a glyph and
		// a gap, which is not a declaration anything states.
		c.Leading = fmt.Sprintf("%s/0", c.Control)
	} else {
		c.Control = "root/0/0"
		if len(growing.Children) != 1 {
			return bandRender{}, nil, fmt.Errorf(
				"%s/%s: the plain band's control holds %d children, want the words alone",
				name, what, len(growing.Children))
		}
		if growing.Children[0].Props["content"] != band.Group.Label {
			return bandRender{}, nil, fmt.Errorf(
				"%s/%s: the plain band's control does not hold the group's label",
				name, what)
		}
		c.Label = "root/0/0/0"
		if growing.Children[0].Style != nil {
			c.LabelInk = growing.Children[0].Style.TextColor
		}
		labelNode = growing.Children[0]
		controlNode = growing
		// On this branch the control's first child IS the words, so the two
		// paths are the same node — which is what makes the indent's effect
		// directly visible on this shape.
		c.Leading = c.Label
		// The plain branch's growing child IS the control, so of course it
		// grows; the field means "the control has a weight of its own beyond
		// being the growing child", which on this branch it cannot.
		c.ControlGrows = false
	}

	// The control's own leading padding, and whether the shape's declared
	// indent reached it.
	//
	// Read off the rendered node for the reason every other number here is —
	// what the browser is asked is whether the widget's own declaration reached
	// the screen — and then held to the number bandRenderBuilders DECLARED,
	// which is the half that reading it back cannot do. A ControlStyle that
	// stopped being applied would render the theme's own inset, this would
	// carry that inset, and the browser would confirm it: two readings of one
	// side of the comparison, agreeing.
	c.ControlPadLeft = float64(controlNode.Style.Padding.Left)
	c.ControlIndent = float64(indent)
	if indent > 0 && c.ControlPadLeft != float64(indent) {
		return bandRender{}, nil, fmt.Errorf(
			"%s/%s: the shape declares ControlStyle PaddingLeft(%d) and the control "+
				"rendered with %g. That style is the caller's own indent — the one thing "+
				"this shape is in the grid to exercise — and a band whose ControlStyle "+
				"stopped reaching the control lays out exactly like the four shapes that "+
				"declare none", name, what, indent, c.ControlPadLeft)
	}
	if c.ControlPadLeft <= 0 {
		return bandRender{}, nil, fmt.Errorf(
			"%s/%s: the control carries no leading padding. The band's chrome is on the "+
				"control rather than on the Row precisely so a press lands on the whole "+
				"band, and with none there is no inset between the tap target's edge and "+
				"its content for the browser to measure", name, what)
	}

	// The ink, on whichever branch found it. Both branches give the label the
	// same three declarations (GroupHeader spells them once), so this is one
	// check rather than two.
	if c.LabelInk == "" {
		return bandRender{}, nil, fmt.Errorf(
			"%s/%s: the band's label declares no text colour, so what a browser paints "+
				"the words in is whatever it inherits and there is nothing to read back. "+
				"GroupHeader gives the label core.TextColor(TextSecondary) — a band that "+
				"stopped would still lay out identically", name, what)
	}
	if c.LabelInk == c.Fill {
		return bandRender{}, nil, fmt.Errorf(
			"%s/%s: the label's ink and the band behind it are both %s, so the words are "+
				"invisible and a check that found the ink would be finding the fill",
			name, what, c.LabelInk)
	}

	// And that both scanned runs are strings the browser's glyph count can be
	// held to. See inkGlyphPerCharacter: that check is one of the three answers
	// to "was the string on the page the string that was measured", and it is
	// true of this grid because of the words chosen here. Asked at the rendered
	// node rather than at bandRenderGroup, because what the browser counts
	// glyphs for is what the component put in the DOM — a label the widget
	// truncated or decorated is a different string from the one the fixture
	// declared, and it is the rendered one the claim has to hold of.
	var badgeNode *core.Node
	if c.Badge != "" {
		badgeNode = row.Children[1]
	}
	for _, run := range []struct {
		what string
		node *core.Node
	}{
		{"the band's label", labelNode},
		{"the count badge's digits", badgeNode},
	} {
		if run.node == nil {
			continue
		}
		content, _ := run.node.Props["content"].(string)
		if err := inkGlyphFault(
			fmt.Sprintf("%s/%s", name, what), run.what, content); err != nil {
			return bandRender{}, nil, err
		}
	}

	// The antialiasing probes: one per box this case's ink assertions read.
	//
	// Built from the scanned nodes themselves rather than from the theme, so a
	// declaration that moved between the widget and the palette travels with
	// the box that carries it. bandRenders deduplicates them — two bands that
	// declare the same words in the same face at the same alpha are one
	// question. See inkProbe.
	var probes []inkProbe
	labelProbe, err := inkProbeFor(labelNode, inkProbeLabel,
		fmt.Sprintf("%s/%s's label", name, what))
	if err != nil {
		return bandRender{}, nil, err
	}
	probes = append(probes, labelProbe)
	if c.Badge != "" {
		badgeProbe, err := inkProbeFor(row.Children[1], inkProbeBadge,
			fmt.Sprintf("%s/%s's count", name, what))
		if err != nil {
			return bandRender{}, nil, err
		}
		probes = append(probes, badgeProbe)
	}
	return c, probes, nil
}
