package core

import (
	"fmt"
	"strconv"
)

// The input builders all take the same mixed argument list the containers do
// — style props and behavior props in any order — rather than the
// `...StyleProp` they took originally. The widening is source-compatible (a
// StyleProp is a PropsAndChildren), and it is what lets a field carry
// OnFocus/OnBlur:
//
//	core.Input(v, "you@example.com", onChange,
//	    core.Padding(8),
//	    core.OnBlur(func() { form.MarkBlurred("email") }),
//	)
//
// See leafNode for the argument contract, for the one call shape the widening
// does break, and for why a View passed to one of these is a debug-mode
// concern rather than a silent no-op.

func Input(value string, placeholder string, onChange func(string), props ...PropsAndChildren) View {
	return ComponentFunc(func(ctx *Context) *Node {
		return leafNode(ctx, "Input", ctx.Theme().Components.Input, map[string]any{
			"value":       value,
			"placeholder": placeholder,
			"onChange":    ctx.registerTextCallback(onChange),
		}, props)
	})
}

// InputWithSubmit is Input plus a submit action: pressing the keyboard's
// return key (iOS) or IME done action (Android) dispatches onSubmit. The
// submit rides the existing void-callback channel — the renderers read the
// "onSubmit" prop and dispatch it exactly like a Button's onClick — so the
// bridge surface is unchanged. A separate builder rather than a variadic
// change to Input keeps every existing call site compiling untouched.
func InputWithSubmit(value string, placeholder string, onChange func(string), onSubmit func(), props ...PropsAndChildren) View {
	return ComponentFunc(func(ctx *Context) *Node {
		return leafNode(ctx, "Input", ctx.Theme().Components.Input, map[string]any{
			"value":       value,
			"placeholder": placeholder,
			"onChange":    ctx.registerTextCallback(onChange),
			"onSubmit":    ctx.registerCallback(onSubmit),
		}, props)
	})
}

func Checkbox(checked bool, onToggle func(bool), props ...PropsAndChildren) View {
	return ComponentFunc(func(ctx *Context) *Node {
		return leafNode(ctx, "Checkbox", ctx.Theme().Components.CheckBox, map[string]any{
			"checked":  checked,
			"onToggle": ctx.registerBoolCallback(onToggle),
		}, props)
	})
}

func InputPassword(value string, placeholder string, onChange func(string), props ...PropsAndChildren) View {
	return ComponentFunc(func(ctx *Context) *Node {
		return leafNode(ctx, "InputPassword", ctx.Theme().Components.Input, map[string]any{
			"value":       value,
			"placeholder": placeholder,
			"onChange":    ctx.registerTextCallback(onChange),
		}, props)
	})
}

func NumericInput(value int, onChange func(int), props ...PropsAndChildren) View {
	return ComponentFunc(func(ctx *Context) *Node {
		id := ctx.registerTextCallback(func(val string) {
			if n, err := strconv.Atoi(val); err == nil {
				onChange(n)
			}
		})

		return leafNode(ctx, "NumericInput", ctx.Theme().Components.Input, map[string]any{
			"value":    fmt.Sprintf("%d", value),
			"onChange": id,
		}, props)
	})
}

// SelectOption is one choice in a Select: the value the app works in, and the
// label a person reads.
//
// Two fields rather than a bare string because the two are different things
// often enough to be worth the type — a country code and a country name, a
// status enum and a sentence — and a widget that took only strings would push
// every caller into keeping a parallel slice. An empty Label means "the value
// is readable enough", which is the common small case (a list of sizes, a list
// of years) and keeps that case a one-word literal.
type SelectOption struct {
	Value string
	Label string

	// Group is the heading this option is filed under: a country's continent,
	// a font's family, "Recently used" above the rest. Empty means the option
	// stands on its own at the top level of the list, which is what every
	// option did before this field existed.
	//
	// # Consecutive options with the same Group form one section
	//
	// Runs, not a gather. Two options naming "Europe" with an American one
	// between them make *two* Europe sections, in the order they were written.
	//
	// That is the honest reading and the only one this widget can offer. The
	// list's order is the caller's — it is what a person sees and what the
	// keyboard walks — and a gather would silently reorder it to suit the
	// headings, which is a bigger change than the one being asked for and one
	// no renderer could undo. Sorting a list into its sections is a line of Go
	// at the call site; un-sorting one is not.
	//
	// Each target draws a run as its own construct: an <optgroup> on the web,
	// a Section in the iOS menu, a heading item in the Android dropdown. All
	// three are labels rather than options — none of them is selectable, and
	// none of them carries a Value.
	//
	// Which options form which run is decided once, by SelectMenuSections
	// (select_menu.go), and not by each renderer — see that file for what four
	// copies of this rule cost.
	//
	// # A heading with nothing under it cannot be written
	//
	// This field is a property of an *option*, so a section with no options
	// has nothing to declare it: a run exists because some option named it.
	// SelectMenuSections therefore never produces an empty section, which
	// SelectMenuSection.First relies on and TestAnEmptySectionIsUnreachable
	// pins.
	//
	// That is a limit rather than an oversight. Declaring a heading
	// independently means a second list beside the options, and then a rule
	// for matching the two — which headings are in use, what a heading with no
	// matching option does, what an option naming a heading that is not in the
	// list does. The run-based reading was chosen precisely to have no
	// matching problem in it, and an empty section is the one thing that
	// reading cannot express. Nothing has asked for it: every real request has
	// been "this category is empty, say so", which is not an empty section at
	// all.
	//
	// What to write instead is a placeholder option, disabled:
	//
	//	{Group: "Archive", Label: "Nothing archived yet", Disabled: true}
	//
	// It is better than an empty section on every target rather than merely
	// possible: an <optgroup> with no <option> in it, a SwiftUI Section with
	// no Button and a Compose heading with no rows are each a label a screen
	// reader announces and a pointer cannot reach, and none of them says why
	// the category is empty. A disabled row says it in the caller's own words,
	// in the place a person is already looking. internal/menufixture carries
	// the shape, so all four picker menus are checked against it.
	Group string

	// Disabled greys this option out: visible, announced, and not choosable.
	// The plan a caller has outgrown, the size that is out of stock, the
	// timezone their region does not offer.
	//
	// Distinct from leaving the option out, which is the alternative and is
	// usually worse: an option that vanishes takes its explanation with it,
	// and a list that changes length between renders is one a person has to
	// re-read. A disabled option says *this exists and you cannot have it*.
	//
	// It does not stop Go from being handed the value. Every target refuses
	// the tap or the click, so nothing reaches onChange through the control —
	// but a Select is controlled, and an app that sets its own state to a
	// disabled option's value will find the widget showing it, because the
	// value shown is always the one Go passed. That is the same contract an
	// out-of-list value lands under; see Select.
	Disabled bool

	// GroupDisabled marks this option's whole *run* unavailable: the paid
	// plans on a free account, a shipping tier this address cannot use, a
	// "Coming soon" family that is worth showing and not worth offering.
	//
	// # Any option in the run is enough
	//
	// The declaration is read off every option, not off the first one. A
	// caller writing it on the second entry of a run and getting nothing would
	// have no way to find that out — a menu is drawn behind a tap, there is no
	// error channel here, and the option would look exactly like an option
	// that had been read. Making any one of them decide is the reading with no
	// silent failure in it.
	//
	// The cost is that a run's state is not known until the run is closed,
	// which is real bookkeeping: SelectMenuSections walks back over the run's
	// items when it flushes one. That is a cost paid once, in the authority,
	// which is the reason the authority exists.
	//
	// # It is not the same as disabling every option by hand
	//
	// Marking each option Disabled refuses each tap and says nothing about the
	// heading, which stays as legible as the ones above it. GroupDisabled
	// carries to the section — SelectMenuSection.Disabled — so the *heading*
	// can be greyed too, and so the web can write <optgroup disabled> once
	// rather than an attribute per option.
	//
	// Every item of a disabled run is still marked Disabled on its way out, so
	// the refusal reaches the two targets that have no section-level control
	// at all. See SelectMenuSection.Disabled.
	//
	// On an ungrouped run (Group empty) there is no heading to grey and, on
	// the web, no <optgroup> to carry the attribute — so it degrades to
	// exactly "every option in the run is disabled", which is the honest
	// answer rather than a special case.
	GroupDisabled bool
}

// Option builds a SelectOption, mirroring Tab's constructor next door.
func Option(value, label string) SelectOption {
	return SelectOption{Value: value, Label: label}
}

// Select is the picker: one value chosen from a fixed list.
//
//	core.Select(form.Country, []core.SelectOption{
//	    {Value: "us", Label: "United States"},
//	    {Value: "pt", Label: "Portugal"},
//	}, func(v string) { form.Country = v })
//
// Controlled, like every other input here: the value shown is always the one
// Go passed, and a change goes up as an event. onChange carries the option's
// *Value*, never its label or its index — the index is the one identity that
// changes when the list is reordered, and a label is written to be read.
//
// # What each target draws
//
//	web       a <select>, whose options are a prop rather than child nodes
//	iOS       a Menu whose label is the chosen option's text
//	Android   a Box anchored to a DropdownMenu, same shape
//
// The natives are deliberately *not* built from a platform picker control
// (SwiftUI's .pickerStyle(.menu), Material's ExposedDropdownMenuBox). Both of
// those draw a frame of their own, and the whole rule this widget lands under
// is that the Go style owns the frame — see borderResetTypes in htmlout/tag.go,
// which the web half joins for the same reason. A control whose edge came from
// the platform on two targets and from the theme on two others is the
// divergence that rule exists to prevent.
//
// # It reads the theme's Input base
//
// A picker is a field: it sits in a form beside text inputs, and a picker that
// did not match the fields around it would look like a mistake. Reading
// Components.Input is also how it inherits the frame those fields grew — the
// same move components.DatePicker makes for the same reason, and the reason
// this widget needs no palette role of its own.
//
// # Options are a prop, not children
//
// A <select>'s options are elements, but they are not *nodes*: no patch is
// ever addressed to one, they carry no style, and they cannot hold a subtree.
// Sending them as children would put four renderers in the business of
// deciding which child is chrome, which is the complication core.TabView's
// tabs prop already avoids one node type over. The web renderer builds the
// <option> elements from the prop; both natives read the same list.
//
// # Grouped and disabled options
//
// SelectOption carries a Group, a Disabled and a GroupDisabled beside its two
// required fields; see the type. All three are drawn by every target — an
// <optgroup>, a disabled <option> and <optgroup disabled> on the web; a
// Section and a disabled Button in the iOS menu; a heading item and a disabled
// item in the Android dropdown.
//
// What a heading still cannot carry is an icon, and that is a decision rather
// than a gap: an <optgroup>'s label is an attribute, so the web can hold text
// and nothing else. A heading with an icon on two targets and without one on
// the other two is the divergence this widget refuses everywhere else — the
// same argument that keeps it off the platform picker controls, one property
// down.
//
// Neither reaches this function as anything but a map key, which is the point:
// the flattening below is the one place that knows what a SelectOption is, and
// the four renderers each read a list of flat string maps. A fifth field would
// land here and nowhere else.
//
// What the renderers do *not* each decide is which options form which run.
// SelectMenuSections (select_menu.go) takes the flattened list and answers
// that once; htmlout calls it, and the two natives carry transliterations that
// ios/verify checks against it.
func Select(value string, options []SelectOption, onChange func(string), props ...PropsAndChildren) View {
	return ComponentFunc(func(ctx *Context) *Node {
		// Flattened to []map[string]string here rather than in each renderer,
		// which is what TabView's tabs prop does and for the same reason: the
		// wire is JSON, and a Go struct crossing it as itself would oblige
		// every renderer to know the field names Go happened to capitalise.
		//
		// The label defaults to the value at this seam, once, so no renderer
		// has to carry the fallback and none of them can disagree about it.
		opts := make([]map[string]string, 0, len(options))
		for _, o := range options {
			label := o.Label
			if label == "" {
				label = o.Value
			}
			opt := map[string]string{"value": o.Value, "label": label}
			// The two optional keys are written only when they say something,
			// so an ordinary option crosses the wire in exactly the shape it
			// always did — which matters beyond tidiness: the WASM runtime
			// decides whether to rebuild a picker's <option> elements by
			// comparing the list's JSON, and rebuilding closes an open
			// drop-down mid-choice. A key that appeared on every option with
			// an empty value would change every signature the first time this
			// shipped, for nothing.
			//
			// Disabled is a string rather than a bool because this map is
			// []map[string]string — one flat shape all four renderers already
			// read. "true" is the spelling core.SelectedState uses for the
			// same reason one property over: it is what the DOM writes, so the
			// web half needs no translation.
			if o.Group != "" {
				opt["group"] = o.Group
			}
			if o.Disabled {
				opt["disabled"] = "true"
			}
			// Written per option even though it describes the run, because
			// the run is not a thing on the wire: the flat list is, and a
			// renderer rebuilds the runs from it. See GroupDisabled for why
			// any one option carrying it is enough, and select_menu.go for
			// where the four renderers agree about that.
			if o.GroupDisabled {
				opt["groupDisabled"] = "true"
			}
			opts = append(opts, opt)
		}
		return leafNode(ctx, "Select", ctx.Theme().Components.Input, map[string]any{
			"value":    value,
			"options":  opts,
			"onChange": ctx.registerTextCallback(onChange),
		}, props)
	})
}

func TextArea(value string, onChange func(string), rows int, props ...PropsAndChildren) View {
	return ComponentFunc(func(ctx *Context) *Node {
		return leafNode(ctx, "TextArea", ctx.Theme().Components.TextArea, map[string]any{
			"value":    value,
			"rows":     rows,
			"onChange": ctx.registerTextCallback(onChange),
		}, props)
	})
}
