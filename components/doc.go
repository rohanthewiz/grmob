// Package components is grmob's widget library: higher-level UI pieces built
// entirely on the public core API, in the idiom of element's components
// package (Workstream 3 of the element-lessons plan).
//
// # The struct-widget idiom
//
// Every widget here is a struct implementing core.View, configured through
// named fields:
//
//	components.Card{
//	    Title: "Account",
//	    Body:  balanceSummary,
//	    Footer: components.Badge{Text: "verified"},
//	}
//
// Structs, not more constructor funcs in core, for two reasons. Named fields
// scale to many optional knobs where positional arguments do not — a widget
// can grow a field without breaking a single call site. And a core.View-typed
// field is a natural composition slot: Card's Header/Body/Footer accept any
// view, the way element's Card distinguishes Body (a string) from
// BodyComponent (a component). Where a widget offers both a simple path and a
// slot (Card.Title vs Card.Header), the slot wins when both are set.
//
// # Discipline
//
// The package deliberately lives outside core and touches nothing internal:
// if a widget can't be built out here, that is a gap in core's primitives,
// not a reason to reach inside. Widgets take their look from ctx.Theme() —
// colors come from the palette, sizes from the spacing/typography scales,
// never hard-coded — and accept Style overrides for per-use adjustment.
// (core keeps the widgets it already had — modal, toast, tabview; new
// widgets land here.)
//
// # Hooks inside widgets
//
// A widget's Render receives the caller's Context, so the hook rules apply
// exactly as they do to any component: a widget that calls NewState (only
// Accordion, currently) consumes a positional slot on the caller's context
// and must therefore be rendered unconditionally, every pass, like any other
// hook user. core.SetDebugMode flags violations as cursor-drift concerns.
package components
