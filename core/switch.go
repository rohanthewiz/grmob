package core

// Switch is the instant-effect boolean: a track and a thumb, flipped to turn
// one thing on or off right now. Airplane mode, notifications, dark theme.
//
//	core.Switch(settings.Notify, func(on bool) { settings.SetNotify(on) })
//
// Each platform draws its own — Material 3's Switch on Android, SwiftUI's
// Toggle on iOS, and an <input type="checkbox" switch role="switch"> in the
// browser and in htmlout.
//
// # Why this is a node type and not a flag on Checkbox
//
// The two controls look similar in a props list and are not interchangeable
// on screen, and the difference is *when the choice takes effect*. A checkbox
// collects a value that something else will act on — a form's "remember me",
// a row's selection, a terms box above a Submit button — so a tick that sits
// there unacted-on is the expected state. A switch acts on the tap: there is
// no Submit, and a switch that needed one would be read as broken.
//
// That is a platform-idiom difference rather than a styling one, which is
// what makes it a type. Both natives draw *both* controls, and they are
// different controls there (Compose's Checkbox and Switch; on iOS the
// platform has no checkbox at all and Toggle stands in for one — see
// GrMobCheckbox in the SwiftUI renderer). A bool prop on Checkbox would have
// the reconciler swapping one platform control for another inside one node's
// update-props patch, which is exactly the work a node type does properly:
// a changed type is a replace, and a replace is how a control is exchanged.
//
// # The state crosses the wire as "checked"
//
// Go says `on` because a switch is on, and the wire says `checked` because
// that is what the DOM calls it — the same split core.SelectedState makes
// when it spells a bool "true" in a prop map.
//
// It is not only tidiness. Both web renderers already carry a `checked` prop:
// htmlout writes the boolean attribute from it and the WASM runtime assigns
// el.checked from it on create *and* on update-props. Naming the prop `on`
// would have meant a second spelling of both halves in both renderers — four
// new branches whose only job is to mean what an existing branch already
// means — and the update half is the one that would have been forgotten,
// because a switch drawn correctly on the first render and frozen thereafter
// looks like a working widget until somebody changes its value from Go.
//
// # What announces it, and why the role is not a core.Role
//
// HTML has no switch control. It has a *switch attribute* on a checkbox
// (WHATWG HTML; Safari draws it, most browsers do not yet), and it has
// role="switch", which tells a screen reader what this is in every browser
// regardless. Both are written, so the control announces itself correctly
// everywhere and is drawn correctly where the browser can — and where it
// cannot, it degrades to a checkbox, which is the same bool in the same
// state.
//
// That role is written from the *node type*, with no Style involved, which
// makes this the second such node after Modal's dialog — see
// htmlout.CarriesOwnRole. It is deliberately not a value in the Role
// vocabulary: every Role a caller can spell obliges all four renderers to
// grow an arm for it (core.Roles() is held against each renderer's dispatch
// in mobile/verify), and there is nothing for the natives to do here. A
// Material Switch and a SwiftUI Toggle announce themselves as switches
// already. `switch` therefore joins aria/spec.NearMisses for the reason
// `dialog` is there: a role this framework emits and does not name.
//
// # It reads the theme's CheckBox base
//
// The same base the other boolean control reads, and not a field of its own.
// What that style actually contributes is geometry and display — both natives
// read only margin and size off a control's style (marginAndSize in the
// Compose renderer, marginAndSizeOnly in SwiftUI), and on the web a control
// drawn by the user agent ignores a fill. A Components.Switch field would
// therefore be a palette entry no palette could spend, measured by the
// contrast census as though some surface were drawn from it.
//
// Like Checkbox it carries no label: a control's label is the caller's, and
// components.FormField and components.InputRow already own that slot.
//
// # Keyboard focus
//
// Not in focusableLeafTypes, for Checkbox's reason one file over: neither
// native renderer gives one keyboard focus, so the stamp would emit a patch
// per focus command that nothing on the far side reads. A browser focuses the
// <input> for free, as it does a checkbox's.
func Switch(on bool, onToggle func(bool), props ...PropsAndChildren) View {
	return ComponentFunc(func(ctx *Context) *Node {
		return leafNode(ctx, "Switch", ctx.Theme().Components.CheckBox, map[string]any{
			"checked":  on,
			"onToggle": ctx.registerBoolCallback(onToggle),
		}, props)
	})
}
