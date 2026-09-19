package core

import "fmt"

func FlexGrow(value float64) StyleProp {
	return styleFunc(func(s *Style) {
		s.FlexGrow = value
	})
}

// FlexShrink sets a flex item's shrink factor. Zero means "do not shrink", and
// it is stored as core.ShrinkNone — see that constant for why this one number
// cannot use the zero-means-unset convention every other number in Style does.
//
// The mapping lives here rather than in Style.Merge because this is the only
// door into the field: a caller writes core.FlexShrink(0) and a renderer reads
// Style.ShrinkFactor(), and nothing in between has to know about the sentinel.
func FlexShrink(value float64) StyleProp {
	return styleFunc(func(s *Style) {
		if value == 0 {
			s.FlexShrink = ShrinkNone
			return
		}
		s.FlexShrink = value
	})
}
func FlexBasis(value string) StyleProp {
	return styleFunc(func(s *Style) {
		s.FlexBasis = value
	})
}
func AlignSelf(value AlignItems) StyleProp {
	return styleFunc(func(s *Style) {
		s.AlignSelf = value
	})
}
func FlexWrap(enabled bool) StyleProp {
	return styleFunc(func(s *Style) {
		if enabled {
			s.FlexWrap = "wrap"
		} else {
			s.FlexWrap = "nowrap"
		}
	})
}
func RowGap(px float64) StyleProp {
	return styleFunc(func(s *Style) {
		s.RowGap = px
	})
}

func ColumnGap(px float64) StyleProp {
	return styleFunc(func(s *Style) {
		s.ColumnGap = px
	})
}
func MinWidth(value string) StyleProp {
	return styleFunc(func(s *Style) {
		s.MinWidth = value
	})
}

func MinHeight(value string) StyleProp {
	return styleFunc(func(s *Style) {
		s.MinHeight = value
	})
}
func Overflow(value string) StyleProp {
	return styleFunc(func(s *Style) {
		s.Overflow = value // "hidden", "scroll", "visible"
	})
}

// WhiteSpace controls how runs of spaces and line breaks in a Text are
// treated: "nowrap" keeps a line whole and lets the container scroll or clip
// it, "pre" additionally preserves the literal spacing, "normal" is the
// default wrapping behavior.
//
// Style.WhiteSpace has been carried to both web targets since it was added,
// but nothing could set it — this is the missing constructor, not a new
// capability. It matters most for text whose columns mean something (code
// listings, tabular output), where wrapping restarts the continuation at
// column zero and makes the indentation actively misleading.
//
// It is a no-op on the natives, which have no equivalent knob: SwiftUI and
// Compose wrap by line-break policy rather than by a CSS-style property.
func WhiteSpace(value string) StyleProp {
	return styleFunc(func(s *Style) {
		s.WhiteSpace = value // "normal", "nowrap", "pre", "pre-line"
	})
}

// MaxLines caps a Text at n lines and ends the last one with an ellipsis ("…")
// where the text is cut; n ≤ 0 removes the cap. It is the one portable way to
// keep a label to its box: WhiteSpace("nowrap") is web-only, and without a cap
// a label wider than a flex slot widens the slot.
//
//	CSS (both web targets)
//	  n = 1   white-space:nowrap; overflow:hidden; text-overflow:ellipsis
//	  n > 1   display:-webkit-box; -webkit-box-orient:vertical;
//	          -webkit-line-clamp:n; overflow:hidden
//	Compose   Text(maxLines = n, overflow = TextOverflow.Ellipsis)
//	SwiftUI   .lineLimit(n).truncationMode(.tail)
//
// One line uses the nowrap spelling rather than line-clamp:1 because a clamp
// only cuts between lines: a single word wider than the box would overflow it
// uncut, where nowrap + ellipsis cuts inside the word as the natives do.
//
// # The minimum it leaves a flex item
//
// A capped Text can be squeezed to nothing on every target, which is what
// lets it sit in a weighted slot without widening it. On the web that falls
// out of overflow:hidden (CSS's automatic minimum of a box that clips is 0);
// Compose's weighted children take the width they are given and truncate in
// it; and iOS's min-content floor (GrMobMinContent) floors a capped Text at 0
// to match the web rather than at its widest word.
//
// Only Text reads it. The web targets write the declarations on any node, as
// they do every CSS-shaped field, but the natives apply it to Text alone.
func MaxLines(n int) StyleProp {
	return styleFunc(func(s *Style) {
		s.MaxLines = max(n, 0)
	})
}

// Responsive registers a style variant under a named key in PseudoStates
// (":hover", ":focus", or a breakpoint name).
//
// The entry is written into a fresh map rather than into whatever map the
// target already holds. A Style is copied by assignment throughout the
// framework — containerNode starts each node from a shallow copy of the
// theme's component Style — so the target's map may well be the theme's own.
// Writing into it in place would edit the theme for every render afterwards.
func Responsive(breakpoint string, style Style) StyleProp {
	return styleFunc(func(s *Style) {
		next := make(map[string]Style, len(s.PseudoStates)+1)
		for k, v := range s.PseudoStates {
			next[k] = v
		}
		next[breakpoint] = style
		s.PseudoStates = next
	})
}
func FontSize(size float64) StyleProp {
	return styleFunc(func(s *Style) {
		s.FontSize = size
	})
}

func TextColor(hex string) StyleProp {
	return styleFunc(func(s *Style) {
		s.TextColor = hex
	})
}

// AccentColor sets the tint of a platform-drawn control: a Switch's on
// track, a Slider's filled track, a Checkbox's box. It overrides the theme's
// Primary, which those three use by default. See Style.AccentColor.
func AccentColor(hex string) StyleProp {
	return styleFunc(func(s *Style) {
		s.AccentColor = hex
	})
}
func Gap(px float64) StyleProp {
	return styleFunc(func(s *Style) {
		s.Gap = px
	})
}

func BackgroundColor(hex string) StyleProp {
	return styleFunc(func(s *Style) {
		s.Background = hex
	})
}
func Align(a Alignment) StyleProp {
	return styleFunc(func(s *Style) {
		s.Align = a
	})
}

func Display(mode DisplayMode) StyleProp {
	return styleFunc(func(s *Style) {
		s.Display = mode
	})
}

func Padding(all int) StyleProp {
	return styleFunc(func(s *Style) {
		s.Padding = EdgeInsets{
			Top: all, Right: all, Bottom: all, Left: all,
		}
	})
}
func BorderRadius(px float64) StyleProp {
	return styleFunc(func(s *Style) {
		s.BorderRadius = px
		// One radius for all four corners, so an earlier CornerRadii is
		// replaced: the later prop is the shape (see CornerRadii).
		s.Corners = Corners{}
	})
}

func Shadow(elevation float64) StyleProp {
	return styleFunc(func(s *Style) {
		s.Shadow = elevation
	})
}

// Rotate turns the node clockwise by deg degrees about its own centre,
// without disturbing the layout around it. See Style.Rotate for what each
// renderer maps it onto and why the angle is not normalised.
//
// Unlike UseStyle, this setter can force zero: Rotate(0) writes the field,
// which is how a caller clears an angle a theme or role style supplied.
func Rotate(deg float64) StyleProp {
	return styleFunc(func(s *Style) {
		s.Rotate = deg
	})
}

// Translate shifts the node's painted box by x along the inline axis and y
// down the block axis, without disturbing the layout around it. Its purpose is
// motion: paired with core.Transition, changing it slides a node, which is how
// comps.Drawer brings its panel in from the edge.
//
//	core.Box(core.Transition(250, core.EaseOut), core.Translate("-100%", ""))
//
// Each axis takes "Npx", a bare number (the same px), or "N%" of the node's
// own box on that axis, so "-100%" moves a panel exactly its own width
// whatever that width is. "" or any zero leaves the axis where it is; any
// other form is ignored on all four targets alike.
//
// # Paint, not layout
//
// Like Rotate, the box keeps the size and place it laid out with and its
// siblings never reflow; only the pixels, and the touch target with them,
// move. A node translated out of its parent overflows it and is clipped by
// the parent's Overflow("hidden"), which both natives read for exactly this.
//
//	CSS       translate: calc(var(--grmob-inline, 1) * x) y
//	Compose   a layout modifier placing the box at placeRelative(x, y)
//	SwiftUI   a GeometryEffect whose translation is resolved against the
//	          view's own size
//
// All three hit-test the moved box where it is drawn. The CSS property is
// the individual `translate`, not `transform`, so it composes with Rotate's
// transform and Spin's `rotate` in CSS's fixed order: translate outermost.
// The natives apply it outside both rotations for the same result.
//
// # Leading, not left
//
// A positive x moves toward the trailing edge: right in a left-to-right
// layout, left in a right-to-left one. That is the natives' own rule
// (Compose's placeRelative mirrors, and SwiftUI mirrors a GeometryEffect's
// translation under RTL, measured with ImageRenderer), and it is what a
// widget wants: a drawer on the leading edge hides at "-100%" in both
// directions. CSS translate is physical, so both web targets multiply x by
// --grmob-inline, a custom property TranslateDirectionCSS sets to -1 under
// dir="rtl". Without the rule on the page the multiplier falls back to 1,
// which is right for every left-to-right document. y has no such question.
//
// Zero on both axes writes no declaration on the web, so an untranslated node
// never becomes a containing block for its fixed-position descendants, and a
// transition to zero still animates, because CSS interpolates to none.
func Translate(x, y string) StyleProp {
	return styleFunc(func(s *Style) {
		s.TranslateX = x
		s.TranslateY = y
	})
}

// TranslateDirectionCSS is the stylesheet rule both web targets pair with a
// non-zero Style.TranslateX: it sets --grmob-inline to -1 under dir="rtl" and
// back to 1 under dir="ltr", so the web's physical translate becomes
// Translate's leading-relative one. Custom properties inherit, so matching the
// dir attribute where it is written covers every element beneath it, and a
// nested dir="ltr" inside an RTL page switches back. The attribute rather than
// :dir(), which also reads only the attribute, because the plain selector is
// supported everywhere and matches far fewer elements.
const TranslateDirectionCSS = "[dir=rtl]{--grmob-inline:-1}[dir=ltr]{--grmob-inline:1}"

func FontWeight(weight Weight) StyleProp {
	return styleFunc(func(s *Style) {
		s.FontWeight = weight
	})
}

func Width(w string) StyleProp {
	return styleFunc(func(s *Style) {
		s.Width = w
	})
}

// MaxWidth caps a node's width: CSS `max-width`, honoured on all four targets.
//
// The value is a dimension string: "320px" (or a bare number, in points on the
// natives), a percentage of the width the parent offers, or "" / "none" for no
// cap. The cap limits the painted box, padding included, and never the margin
// around it. It never grows a box — a label narrower than its cap keeps its own
// width — and it wins over a wider Width: Width("600px") with MaxWidth("520px")
// draws 520.
//
// A stretched child of a Column (the default for most children) fills the
// column up to the cap and sits at the start of the line, as in a browser;
// centre it with the parent's AlignItems. The web targets pass the string
// through verbatim, so units the natives do not read ("vw", "em") cap only
// there. One case differs on the natives: a FlexGrow child of a Row whose cap
// binds keeps its share of the row and leaves the rest empty, where CSS hands
// the remainder to the other growers.
func MaxWidth(w string) StyleProp {
	return styleFunc(func(s *Style) {
		s.MaxWidth = w
	})
}
func Height(w string) StyleProp {
	return styleFunc(func(s *Style) {
		s.Height = w
	})
}
func MaxHeight(w string) StyleProp {
	return styleFunc(func(s *Style) {
		s.MaxHeight = w
	})
}
func Background(w string) StyleProp {
	return styleFunc(func(s *Style) {
		s.Background = w
	})
}

func LinearGradient(x, y, z string) string {
	return fmt.Sprintf(`linear-gradient(%s, #%s, #%s)`, x, y, z)
}

func Margin(all int) StyleProp {
	return styleFunc(func(s *Style) {
		s.Margin = EdgeInsets{
			Top: all, Right: all, Bottom: all, Left: all,
		}
	})
}

func FlexDir(dir FlexDirection) StyleProp {
	return styleFunc(func(s *Style) {
		s.FlexDirection = dir
	})
}

func Justify(j JustifyContent) StyleProp {
	return styleFunc(func(s *Style) {
		s.JustifyContent = j
	})
}

func AlignItemsProp(a AlignItems) StyleProp {
	return styleFunc(func(s *Style) {
		s.AlignItems = a
	})
}

// The three named flex-container types are StyleProps in their own right, so
// the spelling that mirrors core.Align(...) and core.Justify(...) works too:
//
//	core.Column(core.AlignItems(core.AlignItemsCenter), ...)
//
// Without these methods that expression is a type conversion producing a bare
// string value, which containerNode's PropsAndChildren dispatch cannot
// recognize and drops — silently outside debug mode. It was the most natural
// thing to write and it compiled, so an app shipped with every one of its
// AlignItems lost and its columns left-packed on both natives. Making the
// value itself apply removes the trap rather than documenting it.
func (a AlignItems) Apply(s *Style)     { s.AlignItems = a }
func (j JustifyContent) Apply(s *Style) { s.JustifyContent = j }
func (d FlexDirection) Apply(s *Style)  { s.FlexDirection = d }

func Bottom(v string) StyleProp {
	return styleFunc(func(s *Style) {
		s.Bottom = v
	})
}

func Left(v string) StyleProp {
	return styleFunc(func(s *Style) {
		s.Left = v
	})
}

func Right(v string) StyleProp {
	return styleFunc(func(s *Style) {
		s.Right = v
	})
}

func ZIndex(v int) StyleProp {
	return styleFunc(func(s *Style) {
		s.ZIndex = v
	})
}

// PaddingVertical sets the top and bottom insets. Writes the explicit sides
// as well as the shorthand, for the reason given on PaddingHorizontal.
func PaddingVertical(px int) StyleProp {
	return styleFunc(func(s *Style) {
		s.Padding.Vertical = px
		s.Padding.Top = px
		s.Padding.Bottom = px
	})
}

// AccessibilityLabel gives screen readers a name for the element (TalkBack
// contentDescription, VoiceOver label). Set it on anything non-textual a user
// can perceive or activate — images, icon buttons, tappable rows.
func AccessibilityLabel(label string) StyleProp {
	return styleFunc(func(s *Style) {
		s.AccessibilityLabel = label
	})
}

// AccessibilityHint describes the *result* of activating the element
// ("Opens the article"). VoiceOver reads it natively; TalkBack has no hint
// slot, so the Android renderer appends it to the content description.
func AccessibilityHint(hint string) StyleProp {
	return styleFunc(func(s *Style) {
		s.AccessibilityHint = hint
	})
}

// AccessibilityRole says what the element is: a heading, a table cell, a
// search landmark, a tappable Box that is really a button.
//
//	core.Box(core.AccessibilityRole(core.RoleHeading), core.Text("Sermons"))
//
// It is the third question a screen reader asks, after the name
// (AccessibilityLabel) and the effect (AccessibilityHint), and the one nothing
// here could answer until it existed — every container exports as a <div> and
// announces as text. See core.Role for the vocabulary and for which of the
// four renderers honors which value.
//
// Roles are not synthesized from node type or from props: a Box with an OnTap
// is a button only if it says so. Guessing would mean a widget that wraps a
// tappable row in a tappable card announcing two nested buttons, and the
// widget is the only layer that knows which one is the control.
func AccessibilityRole(role Role) StyleProp {
	return styleFunc(func(s *Style) {
		s.AccessibilityRole = role
	})
}

// AccessibilityHeadingLevel says how deep a heading sits — 1 for the screen's
// name, 2 for a section inside it, down to 6.
//
//	core.Box(
//		core.AccessibilityRole(core.RoleHeading),
//		core.AccessibilityHeadingLevel(2),
//		core.Text("March"),
//	)
//
// Paired with RoleHeading, never alone: the role says the node is a heading
// and this says which tier, so a level with no role describes the depth of
// something that is not a heading and every renderer drops it. The two are
// separate props rather than one because the tier is genuinely optional —
// every heading written before this existed is still a correct heading, it
// just does not say where it sits.
//
// What a level buys is the outline. Without one, a screen with a bar title
// over a run of section bands announces a flat list of peers, and a reader
// navigating by heading cannot tell the screen's name from the band inside
// it. comps.AppBar and comps.GroupedList set 1 and 2 for exactly
// that pair, so the common case needs no call site at all.
//
// See Style.AccessibilityHeadingLevel for the range rule (out-of-range is
// dropped, not clamped) and for which of the four renderers can express a
// level — Compose cannot, and that is stated rather than faked.
func AccessibilityHeadingLevel(level int) StyleProp {
	return styleFunc(func(s *Style) {
		s.AccessibilityHeadingLevel = level
	})
}

// AccessibilityNestingLevel says how deep an item sits inside a nested
// collection — 1 for a top-level item, 2 for one inside it, upward with no
// ceiling.
//
//	core.Box(
//		core.AccessibilityRole(core.RoleListItem),
//		core.AccessibilityNestingLevel(2),
//		core.Text("Compline"),
//	)
//
// Paired with RoleListItem or RoleRow, never alone, and never with
// RoleHeading — the depth of a heading is AccessibilityHeadingLevel, which is
// a different question with a different range. Both become aria-level on the
// web, and which one is read is decided entirely by the role, so the two can
// never contend for the attribute.
//
// What a depth buys is the shape of the tree. A flattened nested list — every
// item a sibling of every other — is what a reader gets from a run of divs,
// and it is also what it gets from a correctly roled list whose items do not
// say how deep they are: "list, twelve items" for something the eye reads as
// three groups of four.
//
// Nothing in the framework sets one. Neither DataTable's rows (a flat table)
// nor any bundled widget nests a collection inside itself, so unlike the
// heading pair — which comps.AppBar and comps.GroupedList set for
// every app without a call site — this is a prop an application reaches for
// when it builds the nesting itself.
//
// See Style.AccessibilityNestingLevel for why this is a second field rather
// than a widened first one, and for the two natives that cannot express a
// depth at all.
func AccessibilityNestingLevel(level int) StyleProp {
	return styleFunc(func(s *Style) {
		s.AccessibilityNestingLevel = level
	})
}

// AccessibilitySelected says whether this control is on — the applied filter
// chip, the tab that is showing, the chosen day in a calendar.
//
//	core.Box(
//		core.AccessibilityRole(core.RoleTab),
//		core.AccessibilitySelected(core.SelectedWhen(i == current)),
//		core.Text(label),
//	)
//
// Use core.SelectedWhen to convert the bool a widget already holds. Passing
// core.SelectedOn alone and leaving the other controls unset is the mistake
// the three-valued type exists to prevent — see SelectedState.
//
// Paired with a role that can carry a state, exactly as a level is: tab, row
// and columnheader take aria-selected, a button takes aria-pressed, and a
// state on anything else is dropped by both web targets because ARIA does not
// define either attribute there. A core.Button needs no role of its own; the
// node type is one. See Style.AccessibilitySelected for the two-attribute
// mapping and for why the natives do not scope it the same way.
func AccessibilitySelected(state SelectedState) StyleProp {
	return styleFunc(func(s *Style) {
		s.AccessibilitySelected = state
	})
}

// AccessibilityExpanded says whether this disclosure is open — the accordion
// section showing its body, the twisty that has been turned.
//
//	core.Button(title, toggle,
//		core.AccessibilityExpanded(core.ExpandedWhen(open.Get())),
//	)
//
// Use core.ExpandedWhen to convert the bool the widget already holds. Setting
// only the open case and leaving the shut one unset is the mistake the
// three-valued type exists to prevent: a closed disclosure that says nothing
// is announced as an ordinary button, and "collapsed" is the whole of what
// invites the press.
//
// Paired with a role that can carry it, as a level and a selection both are —
// and *not* the same list a selection takes. aria-expanded is defined for
// button, link, listbox, row, columnheader and combobox among the roles this
// framework carries, which drops option and adds link and listbox. A core.Button needs
// no role of its own, the node type being one; anything else is dropped by
// both web targets. See Style.AccessibilityExpanded for the full table, for
// the dialog-shaped near miss it deliberately does not cover, and for why one
// native maps this and the other cannot.
func AccessibilityExpanded(state ExpandedState) StyleProp {
	return styleFunc(func(s *Style) {
		s.AccessibilityExpanded = state
	})
}

// AccessibilityHasPopup says what activating this control opens.
//
//	comps.Button{Label: "⋯", AccessibilityLabel: "Note actions",
//		Style: []core.StyleProp{core.AccessibilityHasPopup(core.PopupDialog)}}
//
// It is the relationship AccessibilityExpanded deliberately does not cover: a
// trigger that presents a core.Modal is not a disclosure, and says so with this
// instead. Paired with a role ARIA 1.2 defines the attribute on — button,
// link, tab, columnheader and combobox among core's — or with a core.Button,
// whose node type is one; anything else is dropped by both web targets. Neither native reads it. See PopupKind and
// Style.AccessibilityHasPopup.
func AccessibilityHasPopup(kind PopupKind) StyleProp {
	return styleFunc(func(s *Style) {
		s.AccessibilityHasPopup = kind
	})
}

// AccessibilityCurrent says this item is the current one of its set.
//
//	core.Column(core.AccessibilityRole(core.RoleButton),
//		core.AccessibilityCurrent(core.CurrentPage), …)
//
// aria-current on both web targets, on any role. Both natives announce it as
// selected unless AccessibilitySelected is stated. See CurrentKind and
// Style.AccessibilityCurrent.
func AccessibilityCurrent(kind CurrentKind) StyleProp {
	return styleFunc(func(s *Style) {
		s.AccessibilityCurrent = kind
	})
}

// AccessibilityKeyShortcuts declares the keys that activate this control from
// elsewhere on the screen, in aria-keyshortcuts spelling.
//
//	comps.Button{Label: "›", Style: []core.StyleProp{
//		core.AccessibilityKeyShortcuts("PageDown")}}
//
// A chord holding Control, Alt or Meta ("Control+S") is answered from anywhere
// on the screen, on the web, Compose and SwiftUI; a bare key ("PageDown") only
// by a widget that owns it, which today is a grid's PageUp and PageDown on the
// web. See Style.AccessibilityKeyShortcuts for the table and the reasons.
func AccessibilityKeyShortcuts(keys string) StyleProp {
	return styleFunc(func(s *Style) {
		s.AccessibilityKeyShortcuts = keys
	})
}

// AccessibilityValue says where a valued control sits inside its range — how
// far an upload has got, which step a wizard is on.
//
//	core.Row(
//		core.AccessibilityRole(core.RoleProgressBar),
//		core.AccessibilityLabel("Upload"),
//		core.AccessibilityValue(core.ValueOf(45, 0, 100)),
//		…
//	)
//
// Use core.ValueOf to convert the numbers the widget is already holding, and
// .WithText when the digits are not what a listener wants to hear ("step 3 of
// 5"). Leaving it unset beside RoleProgressBar is not an omission — it is
// ARIA's own spelling of an *indeterminate* bar, one that is running with no
// idea how far.
//
// Paired with a role that can carry it, as the level, the selection and the
// disclosure all are, and with the narrowest list of the four: aria-valuenow
// and its bounds are defined for six roles and core carries one of them,
// progressbar. core.Slider is the near miss and is deliberately outside —
// it exports as <input type="range">, which states its own range natively.
//
// ValueRange.Text is the half that is not web-only: it reaches Compose's
// stateDescription and SwiftUI's accessibilityValue, neither of which asks
// what the node is. See Style.AccessibilityValue for the guard table and
// core.ValueRange for why the numbers are strings.
func AccessibilityValue(v ValueRange) StyleProp {
	return styleFunc(func(s *Style) {
		s.AccessibilityValue = v
	})
}

// AccessibilityID gives this element a document-global name that another
// element can point at with AccessibilityControls. It becomes the `id`
// attribute on both web targets and is deliberately unread on both natives.
//
//	core.Box(core.AccessibilityID("app-panel"), core.AccessibilityLabel("Home"), …)
//
// Uniqueness is the caller's, as it is in hand-written HTML, and the "grmob-"
// prefix is reserved for core.TabView's own wiring. See
// Style.AccessibilityID for the whole argument — including why this and
// AccessibilityControls are the only two IDREF props in the vocabulary.
func AccessibilityID(id string) StyleProp {
	return styleFunc(func(s *Style) {
		s.AccessibilityID = id
	})
}

// AccessibilityControls says which element this control switches, by the
// AccessibilityID that element was given. It becomes aria-controls on both web
// targets and is deliberately unread on both natives.
//
//	core.Box(
//		core.AccessibilityRole(core.RoleTab),
//		core.AccessibilitySelected(core.SelectedWhen(tab == "home")),
//		core.AccessibilityID("home-tab"),
//		core.AccessibilityControls("app-panel"),
//		core.Text("Home"),
//	)
//
// It is written verbatim and nothing checks that the target exists: an export
// is one document at a time and a runtime patch is one element at a time, so
// neither target can see the whole page at the moment the attribute is
// written. A reference to an id nothing answers to is inert rather than
// harmful, which is the same trade aria-description makes. See
// Style.AccessibilityID for why this is the one relationship the vocabulary
// carries.
func AccessibilityControls(id string) StyleProp {
	return styleFunc(func(s *Style) {
		s.AccessibilityControls = id
	})
}

// AccessibilityHidden removes the element (and its subtree) from the
// accessibility tree — for decorative content a screen reader should skip.
func AccessibilityHidden() StyleProp {
	return styleFunc(func(s *Style) {
		s.AccessibilityHidden = true
	})
}

// AccessibilitySelectionFollowsFocus makes a composite widget choose the member
// the arrow keys land on, by invoking that member's own OnTap.
//
// Set on the container — the listbox or the tablist — not on the members. See
// Style.AccessibilitySelectionFollowsFocus for when it is right and when it is
// the wrong thing to ask for.
//
// A no-arg flag rather than a bool, like AccessibilityHidden and unlike
// Disabled: a caller does not have this in a variable, and there is no case for
// forcing it back off — a widget that does not want it writes no prop.
func AccessibilitySelectionFollowsFocus() StyleProp {
	return styleFunc(func(s *Style) {
		s.AccessibilitySelectionFollowsFocus = true
	})
}

// Disabled hands the node to the platform's own disabled state: it stops
// accepting taps, keystrokes and focus, and screen readers announce it as
// disabled (Compose `enabled = false`, SwiftUI `.disabled(true)`, the HTML
// `disabled` attribute). See Style.Disabled for the full contract.
//
// It takes the value rather than being a no-arg flag (unlike
// AccessibilityHidden) because the caller almost always has a bool in hand —
// `core.Disabled(sending.Get())` — and because passing false is the only way
// to force a node back to enabled: UseStyle's "a zero value means unset" rule
// means a Style{Disabled: false} cannot clear a flag already on the target.
func Disabled(disabled bool) StyleProp {
	return styleFunc(func(s *Style) {
		s.Disabled = disabled
	})
}

// Inert takes the node and everything inside it out of reach on the web: out
// of the tab order, out of pointer events and out of the accessibility tree
// (the HTML `inert` attribute). On Compose it takes the subtree out of the
// keyboard's focus traversal; SwiftUI does not read it. See Style.Inert for
// why it is a flag of its own and for what each native does.
//
//	core.Box(core.Inert(drawerOpen), screen)
//
// A bool for Disabled's reason: passing false is the only way to clear a flag
// UseStyle has already put on the target.
func Inert(inert bool) StyleProp {
	return styleFunc(func(s *Style) {
		s.Inert = inert
	})
}

func (s Style) With(other Style) Style {
	merged := s
	UseStyle(other).Apply(&merged)
	return merged
}
