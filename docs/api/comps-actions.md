# Package comps — Buttons & choices

```go
import "github.com/rohanthewiz/grmob/comps"
```

Buttons and their variants, copy buttons, links, chips, segmented controls, steppers, ratings and badges.

One of 7 topic pages of [package comps](comps.md), which has the package overview and an index of every topic. This page documents the declarations in `comps/button.go`, `comps/variant.go`, `comps/copy_button.go`, `comps/link.go`, `comps/chip.go`, `comps/chip_strip.go`, `comps/segmented_control.go`, `comps/stepper.go`, `comps/rating.go`, `comps/badge.go`.

## Index

- [Constants](#constants) — `ColorTransparent`, `ConcernLinkInert`
- [`type Badge`](#type-badge)
    - [`func (Badge) Render`](#func-badge-render)
- [`type Button`](#type-button)
    - [`func (Button) Render`](#func-button-render)
- [`type Chip`](#type-chip)
    - [`func (Chip) Render`](#func-chip-render)
- [`type ChipStrip`](#type-chipstrip)
    - [`func (ChipStrip) Render`](#func-chipstrip-render)
- [`type CopyButton`](#type-copybutton)
    - [`func (CopyButton) Render`](#func-copybutton-render)
- [`type Emphasis`](#type-emphasis)
- [`type Link`](#type-link)
    - [`func (Link) Render`](#func-link-render)
    - [`func (Link) Span`](#func-link-span)
- [`type Prominence`](#type-prominence)
- [`type Rating`](#type-rating)
    - [`func (Rating) Render`](#func-rating-render)
- [`type SegmentedControl`](#type-segmentedcontrol)
    - [`func (SegmentedControl) Render`](#func-segmentedcontrol-render)
- [`type Stepper`](#type-stepper)
    - [`func (Stepper) Render`](#func-stepper-render)
- [`type Variant`](#type-variant)
    - [`func (Variant) Color`](#func-variant-color)
    - [`func (Variant) Ink`](#func-variant-ink)
    - [`func (Variant) OnLight`](#func-variant-onlight)

## Constants

ColorTransparent is a fully transparent fill, written in the CSS byte order (#RRGGBBAA) that every target parses: htmlout emits it verbatim as a CSS Color 4 hex, and both native parseColor implementations handle the 8-digit form with alpha last.

It exists because core.Style has no "clear" or "unset" for a color, and an \*empty\* Background is not transparent — it means "inherit the theme's Button base", which is a solid Primary fill. Outlined and Ghost need an actual hole, not an omission.

```go
const ColorTransparent = "#00000000"
```

<small>[comps/button.go:14](https://github.com/rohanthewiz/grmob/blob/master/comps/button.go#L14)</small>

ConcernLinkInert is raised, in debug builds only, when a Link has neither a URL nor an OnTap. It is drawn in the link colour and announced as a link, and a tap does nothing — a promise the screen makes and does not keep, which on screen looks exactly like a working link.

```go
const ConcernLinkInert = "link-inert"
```

<small>[comps/link.go:9](https://github.com/rohanthewiz/grmob/blob/master/comps/link.go#L9)</small>

## Types

### type Badge

```go
type Badge struct {
	Text string

	// Variant selects the semantic color role: Success, Warning, Error, or
	// the zero value for the theme's Primary.
	Variant Variant

	// Color is the pill background. Empty takes the Variant's role color; a
	// literal here overrides the variant, since an explicit color is the more
	// specific instruction.
	Color string

	// TextColor is the label ink. Empty is resolved by Variant.Ink: the
	// theme's Background for VariantDefault, and for a status variant
	// whichever of the theme's two ink roles reads better on the fill.
	TextColor string

	// Style is applied after the badge's own pill styling.
	Style []core.StyleProp
}
```

Badge is a small non-interactive status pill — a count on a tab, a "verified" mark, a state label. For a \*selectable\* pill, use Chip.

#### Variants

Variant names the badge's meaning and takes its colors from the palette's status roles, so a status pill stops carrying literal hex:

	comps.Badge{Text: "Paid", Variant: comps.VariantSuccess}
	comps.Badge{Text: "Expiring", Variant: comps.VariantWarning}
	comps.Badge{Text: "Failed", Variant: comps.VariantError}

The zero value is VariantDefault — the theme's Primary — which is the look every badge written before this field had, so adding it restyles nothing.

The label ink is chosen per variant by contrast against the resolved background rather than fixed, because the palette pairs no ink with a status role and the right answer flips between themes. See Variant.Ink; the short version is that a fixed white ink would render DefaultTheme's Success and Warning badges at ~2.2:1, which is unreadable.

#### Color is not the message

A variant is \*reinforcement\* for Text, never a substitute for it. Nothing here announces "warning" to a screen reader, and a reader who cannot distinguish the tints sees only the label — so the label has to say it ("Overdue", not "!"). That is WCAG 1.4.1, and it is why Variant deliberately does not synthesize an accessibility label the way Avatar does: Avatar has one obvious thing to say, a badge's meaning is already in its Text.

<small>[comps/badge.go:34](https://github.com/rohanthewiz/grmob/blob/master/comps/badge.go#L34)</small>

#### func (Badge) Render

```go
func (b Badge) Render(ctx *core.Context) *core.Node
```

<small>[comps/badge.go:55](https://github.com/rohanthewiz/grmob/blob/master/comps/badge.go#L55)</small>

### type Button

```go
type Button struct {
	Label string
	OnTap func()

	// Variant selects the semantic color role: Success, Warning, Error, or
	// the zero value for the theme's Primary.
	Variant Variant

	// Emphasis selects how that color is spent: filled (zero), outlined, or
	// ghost.
	Emphasis Emphasis

	// FullWidth stretches the button across its parent instead of hugging its
	// label. It sets both Width and a block Display: the bundled themes give
	// Button an inline display, and width has no effect on an inline box in
	// CSS. The natives read Display only for "hidden", so the block half is
	// inert there and the width alone does the work.
	FullWidth bool

	// Disabled renders the muted treatment and marks the control inert.
	//
	// Three independent things have to be true, and the widget does not get to
	// pick two:
	//
	//   - The platform must refuse to dispatch. That is core.Disabled, which
	//     every renderer now maps onto the native state (Compose's
	//     `enabled = false`, SwiftUI's `.disabled(true)`, the HTML disabled
	//     attribute). It is also what makes the control announce itself as
	//     disabled to a screen reader — VoiceOver says "dimmed", TalkBack
	//     reads the Disabled property — which is why the label no longer
	//     carries a hand-written ", disabled" suffix. Doing both would
	//     announce the state twice.
	//   - The handler must still be registered, replaced with a no-op rather
	//     than dropped: core.Button registers whatever it is given, and a nil
	//     func() in the registry panics if a native tap arrives in the window
	//     between the user pressing and the disabling patch landing.
	//   - It must look inert, which is the colorProps treatment below.
	Disabled bool

	// Style is applied after the variant treatment, so any single property
	// here overrides it.
	Style []core.StyleProp

	// AccessibilityLabel replaces the visible label for screen readers — use
	// it when Label is a glyph ("✕"). AccessibilityHint describes the effect
	// of tapping.
	AccessibilityLabel string
	AccessibilityHint  string

	// FocusRef names the button for core.Focus, so a handler elsewhere can
	// move focus onto it: Drawer's close button, which the control that
	// opened the drawer focuses so a keyboard or screen-reader user lands
	// inside the panel rather than on a control now hidden behind it. Nil
	// names nothing.
	FocusRef *core.FocusRef
}
```

Button is a themed action button with two orthogonal color axes and no per-call hex.

	comps.Button{Label: "Save", OnTap: save}                                // theme Button base
	comps.Button{Label: "Delete", Variant: comps.VariantError, OnTap: rm}
	comps.Button{Label: "Cancel", Emphasis: comps.EmphasisOutlined, OnTap: back}
	comps.Button{Label: "Skip", Emphasis: comps.EmphasisGhost, OnTap: skip}

#### The zero value applies nothing

Button{Label: l, OnTap: f} renders exactly core.Button(l, f) — not "core.Button re-derived from the palette". With both axes at their zero values the widget contributes no color props at all, so the theme's own Components.Button carries the look through untouched. That matters because a theme's Button base is allowed to differ from Colors.Primary; re-deriving would silently overwrite that choice, and the difference is invisible in the bundled themes where the two happen to agree.

#### Secondary is deliberately not a Variant

Variant carries \*meaning\* — success, warning, error. Secondary is a brand slot a theme may set to anything (MaterialTheme makes it teal), so a "Variant: Secondary" would put a brand color where a reader expects a status. A button that wants the brand's second color says so through Style, which is the escape hatch for exactly the case the semantic roles do not cover:

	comps.Button{Label: "Recharge", Emphasis: comps.EmphasisOutlined,
	    Style: []core.StyleProp{core.TextColor(t.Colors.Secondary), core.BorderColor(t.Colors.Secondary)}}

#### Contrast, and the limit of what this widget can promise

EmphasisFilled owns both the fill and the label, so it picks the label by contrast against the fill (Variant.Ink) and is tested to clear WCAG AA under both bundled themes.

Outlined and Ghost own neither: the fill is transparent, so the label's real backdrop is whatever the button was placed on, which the widget cannot see. Their label is therefore the role's \*on-light\* tone — the palette's second value per role, dark enough to be read as ink on a light surface — rather than the fill colour. Measured against each theme's own Background (both are #FFFFFF), with the value each replaced in brackets:

	            Default          Material
	default      7.56:1 [same]    7.63:1 [same]
	success      5.40:1 [2.22]    5.13:1 [same]
	warning      5.28:1 [2.20]    5.60:1 [3.08]
	error        5.38:1 [3.55]    7.33:1 [same]

All eight clear WCAG AA (4.5:1); four of the eight did not before. A "same" means the role needed no second tone and the theme declares the role itself, rather than the field being blank — see the palette's own doc for why a measurement is stated rather than left to the fallback.

DefaultTheme's default row reads "same" and did not always: its Primary was iOS systemBlue at 4.02:1 here, and the tone was the second value that fixed this treatment while the \*filled\* one stayed illegible. The role has since moved to Apple's accessible blue, which is the same hex the tone already carried — so the two collapsed into one, and the number in this row is unchanged by it.

The promise is still narrower than EmphasisFilled's. These numbers hold against a theme's Background, and a button placed on some other surface — a tinted card, a photo — is measured against that instead, which nothing here can know. What changed is that the default case is now legible rather than documented as illegible.

A theme that declares no on-light tones falls back to the role colour, i.e. to the bracketed numbers, and to exactly the pixels this widget painted before the palette had a second value. Darkening a role colour \*here\* was considered and rejected for the reason it always was: it would repaint a hex the theme author chose. Declaring the second value is the theme's call; spending it is this widget's — and when DefaultTheme later did darken its Primary, it was to fix the \*filled\* treatment, whose declared white on systemBlue this widget could only document.

<small>[comps/button.go:145](https://github.com/rohanthewiz/grmob/blob/master/comps/button.go#L145)</small>

#### func (Button) Render

```go
func (b Button) Render(ctx *core.Context) *core.Node
```

<small>[comps/button.go:202](https://github.com/rohanthewiz/grmob/blob/master/comps/button.go#L202)</small>

### type Chip

```go
type Chip struct {
	Label    string
	Selected bool
	OnTap    func()

	// Prominence tunes the unselected state alone: quiet (zero) or loud. It
	// says nothing about the selected chip, which is the theme's Button base
	// in both — there is nothing louder to give it, and the row must keep
	// reading as "this is the one you picked" either way.
	//
	// UnselectedStyle still wins where it is set, being the more specific of
	// the two: Prominence picks between the widget's own treatments, that
	// replaces them.
	Prominence Prominence

	// Style is applied to every chip, selected or not, before the state's
	// own styling. The state wins on any field both set — otherwise one
	// Style shared across a strip would flatten the very distinction the
	// strip exists to draw. Use it for the properties that are the same in
	// both states (radius, padding, font) and the two state fields below for
	// the ones that are not.
	Style []core.StyleProp

	// SelectedStyle replaces the selected default described above; nil takes
	// the default. UnselectedStyle is its counterpart for the other state.
	//
	// Both distinguish nil from empty: a nil slice means "use the theme
	// default", an allocated but empty one means "apply nothing", which is
	// how a caller drops a default rather than overriding it.
	SelectedStyle   []core.StyleProp
	UnselectedStyle []core.StyleProp

	// AccessibilityLabel names the chip for screen readers.
	// AccessibilityHint describes the effect of tapping.
	//
	// The name no longer carries the state. Until core.Style had a slot for
	// one, this widget appended ", selected" to whatever named the chip,
	// because a name was the only channel it had; the state now goes out as
	// core.AccessibilitySelected and reaches every target as the platform's
	// own idea of a control being on (aria-pressed on both web targets, a
	// `selected` semantics property in Compose, the .isSelected trait in
	// SwiftUI).
	//
	// Both would announce it twice, which is the reasoning Button's Disabled
	// field already gives for dropping its own ", disabled" suffix. It is
	// also the better half to keep: a name is meant to be stable, so a reader
	// re-announcing the control after a tap read out the whole altered name,
	// where a state change is announced as a state change.
	AccessibilityLabel string
	AccessibilityHint  string
}
```

Chip is a selectable pill — a filter toggle, a tag picker entry. It renders as a themed Button whose look shifts with Selected, so switching selection patches two style fields instead of restructuring the row (the pattern the todoapp filter bar established; this widget is that bar's extraction).

Selection is controlled by the caller: Chip holds no state, it just renders Selected and reports taps through OnTap. A group of chips is therefore one piece of parent state plus a loop.

#### Which state is the loud one

Selected is the prominent state: the theme's Button base — a solid fill with the base's own label colour. Unselected is the quiet one: a Surface fill, TextPrimary ink and a ring in the theme's control-boundary tone.

That is the reverse of what this widget shipped with, and the reversal is the whole of the change. The original default painted the \*selected\* chip Surface-on-Primary and left the unselected chips on the solid Button base, so a filter row read inverted — the four options the reader had not chosen shouted, and the one they had chosen receded. It was what the doc comment said it was, so it was a design call rather than a bug, but three separate consumers reported the same surprise on first sight of the rendered output, which is the point at which a design call is wrong.

A caller who wants the old look back has it in one field:

	comps.Chip{Label: l, Selected: sel, OnTap: f,
	    SelectedStyle: []core.StyleProp{
	        core.BackgroundColor(t.Colors.Surface), core.TextColor(t.Colors.Primary),
	    },
	    UnselectedStyle: []core.StyleProp{}, // empty, not nil: "apply nothing"
	}

#### How loud the quiet state is, is a second question

Which state is louder is settled above and is not negotiable — that was the bug. \*How much\* quieter the other one is has two right answers, and Prominence is the field that picks: quiet (the default, a Surface fill) for a filter row, loud (an outline in the chip's own accent) for a row of suggestions the reader is meant to reach into. See Prominence.

Neither answer is "invisible". Both treatments draw a 1px ring at control-boundary weight — the loud one in the chip's own accent, the quiet one in Colors.ControlBorder — because WCAG 1.4.11 puts a 3:1 floor under the edge that identifies a control, and a chip's fill clears it in neither state. See stateStyle for the numbers.

#### The selected default restates the theme's own Button colors

It sets the fill and the ink to the values Components.Button already carries, so the look is the theme's, not a re-derivation of it. That is the rule Button's zero value follows, for the reason it gives: a theme's Button base is allowed to differ from Colors.Primary, and a widget rebuilding the fill out of the palette would silently overwrite that choice in a way the bundled themes — where the two agree — cannot show.

Restating rather than simply letting the base show through is what makes "the state wins over Style" true on this side as well. A color the selected default never set could not beat one in Style, so a strip handed a single shared Style{BackgroundColor(x)} would paint x on the selected chip and the quiet fill on all the others — the inversion again, by a different route.

The border is there for geometry, not for decoration. Only the unselected chip wants a visible rule, but a rule on one state alone makes that state 2px wider and taller: a chip declares no size, so its border adds to its content size under either box model, and a pill that grows when you tap it is a worse artifact than the one this fixes. So both states carry a 1px border and the selected one paints it in its own fill, where it cannot be seen.

<small>[comps/chip.go:132](https://github.com/rohanthewiz/grmob/blob/master/comps/chip.go#L132)</small>

#### func (Chip) Render

```go
func (c Chip) Render(ctx *core.Context) *core.Node
```

<small>[comps/chip.go:184](https://github.com/rohanthewiz/grmob/blob/master/comps/chip.go#L184)</small>

### type ChipStrip

```go
type ChipStrip struct {
	// Chips are the strip's contents, in order.
	Chips []Chip

	// Children is the escape hatch: arbitrary views in place of Chips, taking
	// precedence when set. For a strip mixing chips with something else — a
	// trailing "+ Add", a Badge among the tags — or for chips already built
	// by a helper of the caller's own. Nil entries are skipped.
	Children []core.View

	// Gap is the spacing between chips, both across and between lines. Zero
	// takes the theme's SM step.
	Gap float64

	// Scrollable makes the strip one line that pans sideways instead of a
	// block that wraps; see the type comment for when each is right.
	//
	// It is a core.Scroll carrying core.Horizontal, not a Row with an
	// overflow: the natives have no CSS to fall back on and implement
	// sideways panning in their scroll composites alone (Compose
	// horizontalScroll, SwiftUI ScrollView(.horizontal)), so the node type
	// has to change, not just a style. Style still lands on the strip
	// itself either way — the Scroll *is* the strip when this is set, so
	// there is no extra box to configure.
	Scrollable bool

	// Style is applied after the widget's own defaults, so the wrap, the gap
	// and the (removed) row padding are all overridable.
	Style []core.StyleProp
}
```

ChipStrip lays out a run of Chips that wraps onto as many lines as it needs — a filter bar, the tags on an article, the scripture references on a sermon, the quick amounts on a giving form.

	comps.ChipStrip{Chips: []comps.Chip{
	    {Label: "All",      Selected: f == "",      OnTap: func() { filter.Set("") }},
	    {Label: "Sermons",  Selected: f == "sermon", OnTap: func() { filter.Set("sermon") }},
	    {Label: "Articles", Selected: f == "article", OnTap: func() { filter.Set("article") }},
	}}

#### Why the field is \[]Chip and not a parallel vocabulary

The tempting API is Labels \[]string plus Selected func(string) bool plus OnTap func(int) — and it is a second way to describe a chip, which then has to grow its own Style, its own AccessibilityLabel, its own everything as Chip does. Taking \[]Chip means the strip adds layout and nothing else: a chip configured here is configured exactly as a chip configured anywhere, and a caller who needs one of Chip's knobs already has it.

That rule is also why there is no strip-level Prominence, tempting though a loud strip is to ask for: it would be a second place to configure a chip, and every field added there is one Chip and ChipStrip then have to keep in step. A strip whose chips are all loud sets the field where the chips are built, which is the same loop that already sets Label and Selected.

#### ChipStrip is not SegmentedControl

SegmentedControl is one-of-N: a fixed, exhaustive set where exactly one option is live, drawn as a single joined control. ChipStrip is the loose case — any number selected including none, a set that comes from data and changes length, entries that may not be selectable at all. Reach for the segmented control when the options are a closed choice, this when they are a collection.

#### Wrapping or scrolling

The default is to wrap: a strip with more chips than fit takes a second line. That is the right shape for a set the reader should see all of — the tags on an article, the scripture references on a sermon — and it costs the layout nothing.

Scrollable is the other shape. A long filter bar reads better as one line that pans sideways, because a bar that grows to three lines pushes the content it filters off the screen, and the chips past the fold are a hint that there is more rather than a queue demanding to be read.

	comps.ChipStrip{Scrollable: true, Chips: years}

The two are exclusive by construction — a scrolling strip is one line, so there is nothing to wrap — and Scrollable wins when both are asked for.

<small>[comps/chip_strip.go:55](https://github.com/rohanthewiz/grmob/blob/master/comps/chip_strip.go#L55)</small>

#### func (ChipStrip) Render

```go
func (c ChipStrip) Render(ctx *core.Context) *core.Node
```

<small>[comps/chip_strip.go:86](https://github.com/rohanthewiz/grmob/blob/master/comps/chip_strip.go#L86)</small>

### type CopyButton

```go
type CopyButton struct {
	// Text is what lands on the clipboard. Empty disables the button; see
	// "Nothing to copy means inert".
	Text string

	// Label is the visible caption; empty gives "Copy".
	Label string

	// CopiedMessage is the toast shown after a copy; empty gives "Copied".
	CopiedMessage string

	// Variant and Emphasis are passed to the Button unchanged. The zero value
	// is a filled Primary button; a copy action sitting beside the thing it
	// copies usually wants EmphasisOutlined or EmphasisGhost.
	Variant  Variant
	Emphasis Emphasis

	// Disabled makes the button inert even with Text set.
	Disabled bool

	// AccessibilityLabel names the button for screen readers; empty uses the
	// visible Label. AccessibilityHint describes the effect; empty gives
	// "Copies to the clipboard".
	AccessibilityLabel string
	AccessibilityHint  string

	// Style is applied to the Button after its variant treatment.
	Style []core.StyleProp

	// FocusRef names the button for core.Focus.
	FocusRef *core.FocusRef
}
```

CopyButton is a button that puts a fixed string on the system clipboard and confirms it: a code snippet, an invite link, a QR code's payload, an order number.

	comps.CopyButton{Text: inviteURL, Label: "Copy link"}

	tap ──► core.WriteClipboard(Text)
	    ──► core.Haptic(HapticLight)
	    ──► core.ShowToast(CopiedMessage)          "Copied"

#### The toast confirms, not the caption

The familiar web idiom flips the button's own caption to "Copied ✓" for a second or two. That needs a timer to flip it back, and a timer here is a hook (hooks.UseTimeoutWhile), which would make CopyButton unsafe to render inside a conditional or a loop. Its first consumer is the tutorial's codeBlock, which is built in exactly those places and says in its own doc that a widget with hook obligations could not be. Banner's doc had already assigned the job: "Use the toast for 'Copied'". So the platform's transient overlay confirms, and the widget stays stateless, like Stepper and TimePicker.

The haptic is the light tick, the one a successful small action gets; a device without a motor, and every web target, ignores it.

#### Nothing to copy means inert

An empty Text disables the button rather than reporting a concern. It is a legitimate state — an invite link still being fetched, an order number not yet assigned — and a disabled Copy says truthfully that there is nothing to copy yet. The one thing an enabled Copy must not do with an empty string is write it: core.WriteClipboard("") clears the clipboard, which is a real request when made on purpose and never what a button labelled "Copy" means.

#### Accessibility

Several copy buttons on one screen — one per code block — would all be announced "Copy", so AccessibilityLabel is where a caller says what is copied ("Copy code", "Copy invite link"). The copied text itself is not read out: it is usually on screen beside the button, and a URL or a snippet spoken in full is noise. The toast is announced by each platform's own toast machinery.

#### Compact on Android

The button is marked with a touchTarget of "compact", which only this widget writes. On Android it drops material3's minimums, a 40dp content height inside a 48dp touch target, and draws the button at its padding and label, as SwiftUI and the web already do. A code block's Copy sat in a strip 48dp tall on Android, against about 22px elsewhere, because the strip is the button's height.

That is a trade, taken on purpose and for this widget only: Material's 48dp is an accessibility floor for touch. A copy action beside the text it copies is a secondary control, and a code block with a 48dp band above it reads as broken. Every other Button keeps the floor; see GrMobButton in the Android renderer.

#### Theme roles read

None of its own: the button is a comps.Button, and reads what Button reads for the Variant and Emphasis given.

<small>[comps/copy_button.go:68](https://github.com/rohanthewiz/grmob/blob/master/comps/copy_button.go#L68)</small>

#### func (CopyButton) Render

```go
func (c CopyButton) Render(ctx *core.Context) *core.Node
```

Render draws the button. It takes no hook slot, so it may be rendered conditionally.

<small>[comps/copy_button.go:103](https://github.com/rohanthewiz/grmob/blob/master/comps/copy_button.go#L103)</small>

### type Emphasis

```go
type Emphasis string
```

Emphasis is how strongly a button asserts itself: how much of the variant's color it spends. It is the second of Button's two color axes.

#### Why two axes and not one enum

The obvious API is a single enum — Primary | Secondary | Danger | Ghost — and it is what the component gap analysis first sketched. It was dropped because it conflates two independent questions: \*which\* color (the meaning) and \*how much\* of it (the visual weight). A flat enum cannot express an outlined destructive button, the ordinary shape of a "Delete" confirmation, without a fifth value, and then a ghost destructive needs a sixth.

Splitting them also lets Button reuse Variant verbatim, so a danger Button and an error Badge are the same red by construction rather than by two palettes agreeing. See Variant, which Badge already uses.

<small>[comps/button.go:31](https://github.com/rohanthewiz/grmob/blob/master/comps/button.go#L31)</small>

```go
const (
	// EmphasisFilled is the zero value: a solid fill in the variant's color
	// with a contrast-picked label. This is the look core.Button has always
	// had, which is what makes the field's zero value a no-op.
	EmphasisFilled Emphasis = ""

	// EmphasisOutlined is a transparent fill, a 1px rule and a label both in
	// the variant's color — a secondary action that still names its meaning.
	EmphasisOutlined Emphasis = "outlined"

	// EmphasisGhost is EmphasisOutlined without the rule: label only. For a
	// tertiary action, a toolbar glyph, or a tab that must not look like a
	// pill.
	EmphasisGhost Emphasis = "ghost"
)
```

### type Link

```go
type Link struct {
	// Text is the visible link text and its accessible name.
	Text string

	// URL is opened with core.OpenURL when OnTap is nil.
	URL string

	// OnTap handles the tap instead of opening URL.
	OnTap func()

	// AccessibilityHint describes where the link goes when Text alone does not
	// ("Opens in your browser").
	AccessibilityHint string

	// Style is applied to the link text after its defaults.
	Style []core.StyleProp
}
```

Link is a line of text that goes somewhere: a terms page, a help article, a "Forgot password?" under a sign-in form.

	comps.Link{Text: "Privacy policy", URL: "https://example.com/privacy"}
	comps.Link{Text: "Forgot password?", OnTap: showReset}

	┌ Box  role=link  name=Text  onClick ┐
	│  Text  (Primary's on-light tone)   │
	└────────────────────────────────────┘

#### A link and not a ghost Button

The two look alike and the difference is the one core.RoleLink's doc draws: a button does something here, a link goes somewhere else. A reader deciding whether to follow a control needs to know which, so the node carries RoleLink, which the web maps to role="link" and both natives to their link trait. StaticMap's tappable form made the same call.

#### OnTap or URL

OnTap wins when it is set: an in-app destination (a Navigator push, a sheet) is a link too, and the caller knows how to get there. Otherwise a tap calls core.OpenURL(URL), which hands the address to the platform — the browser, or the app registered for the scheme (mailto:, tel:).

#### On its own line, and inside a sentence

Rendered, a Link is a line of its own, not underlined: the link colour and the role carry the distinction, which is enough for a line that is nothing but the link. Inside running text use Link.Span, a run of a core.Paragraph in the same colour and underlined, because there the colour is the only other thing that says which words are the link.

#### Theme roles read

	Ink          Colors.Primary's on-light tone (Variant.OnLight)
	Type         Typography.Body

<small>[comps/link.go:48](https://github.com/rohanthewiz/grmob/blob/master/comps/link.go#L48)</small>

#### func (Link) Render

```go
func (l Link) Render(ctx *core.Context) *core.Node
```

<small>[comps/link.go:100](https://github.com/rohanthewiz/grmob/blob/master/comps/link.go#L100)</small>

#### func (Link) Span

```go
func (l Link) Span(ctx *core.Context) core.Span
```

Render draws the link. It takes no hook slot. Span is this link as a run of a core.Paragraph: the same colour, underlined, and the same tap (OnTap, else opening URL), inside a sentence rather than on a line of its own.

	core.Paragraph([]core.Span{
	    {Text: "By continuing you accept the "},
	    comps.Link{Text: "terms", URL: termsURL}.Span(ctx),
	    {Text: "."},
	})

Underlined where the standalone Link is not: on its own line a link is told apart by being a line of its own in the link colour, and inside a sentence the colour is the only thing left, which a reader who cannot see it would miss (WCAG 1.4.1).

<small>[comps/link.go:81](https://github.com/rohanthewiz/grmob/blob/master/comps/link.go#L81)</small>

### type Prominence

```go
type Prominence string
```

Prominence is how loudly a Chip's \*unselected\* state asserts itself. It is the answer to a question the widget shipped with one answer to and which turns out to have two, both right.

	the sermons year filter    a set of options, most of them not chosen. The
	                           row is chrome above the list it filters, and a
	                           loud row of years competes with the archive.
	                           Quiet.

	a giving form's suggested  four ways to answer the screen's only question,
	amounts                    and the fast path most gifts take. A row of grey
	                           pills over an empty amount field does not read
	                           as "tap one of these". Loud.

Material draws exactly this distinction — a \*filter\* chip against a \*suggestion\* chip — with different default prominence for each. This field is that distinction, arriving here because the first two consumers of the widget wanted one each and the second had to spell its treatment by hand.

#### Why this is a new type rather than Button's Emphasis

Emphasis is the nearest thing in the package and is deliberately not reused, for two reasons that both bite.

Its zero value is EmphasisFilled — the loud one — because a Button with no opinion is a solid button. Chip's zero has to stay quiet, because that is the look every chip in an existing tree already has and the field must be a no-op. Sharing the type would mean the same zero value meaning opposite things in two widgets a page apart.

And Emphasis describes a whole control, where this describes \*one of a chip's two states\*. EmphasisGhost has no meaning here: a chip with no box is indistinguishable from a run of text, and the selected/unselected pair is exactly what a chip exists to draw.

<small>[comps/chip.go:39](https://github.com/rohanthewiz/grmob/blob/master/comps/chip.go#L39)</small>

```go
const (
	// ProminenceQuiet is the zero value: a Surface fill, TextPrimary ink and
	// a ring in the theme's control-boundary tone. Right for a filter row,
	// which is chrome above the content it filters.
	//
	// "Quiet" is about the fill and the ink. The ring is not part of what
	// recedes — it is the only thing that says the pill is a control, and it
	// used to be drawn in the divider role, which made a chip that receded
	// out of sight rather than into the background. See stateStyle.
	ProminenceQuiet Prominence = ""

	// ProminenceLoud draws the unselected chip as an outline in the chip's
	// own accent — the accent as ink and as a 1px rule over a transparent
	// fill, which is what EmphasisOutlined spends on a secondary button.
	//
	// It is not a return to the pre-inversion look, and the difference is the
	// whole point: the old default painted every unselected chip a *solid*
	// fill and left the chosen one pale. Here the selected chip keeps its
	// solid fill while its neighbours are outlines, so the row says both
	// "pick one of these" and "this is the one you picked".
	ProminenceLoud Prominence = "loud"
)
```

### type Rating

```go
type Rating struct {
	// Value is the score, from 0 to Max. It is rounded to whole glyphs, or
	// to halves with Halves.
	Value float64

	// Halves rounds Value to the nearest half and draws a half star for it.
	// See "Halves are drawn, not typed".
	Halves bool

	// Max is the number of glyphs. Zero means 5.
	Max int

	// OnChange receives the tapped glyph's 1-based position. Nil, like
	// ReadOnly, draws a display-only rating.
	OnChange func(int)

	// ReadOnly drops the handlers and the per-glyph buttons.
	ReadOnly bool

	// Glyph and EmptyGlyph draw filled and unfilled positions. Empty uses
	// "★" and "☆".
	Glyph, EmptyGlyph string

	// Label is the group's accessible name. Empty uses "Rating".
	Label string

	// Style is applied to the row after the widget's own props.
	Style []core.StyleProp
}
```

Rating is a row of stars (or any glyph), read-only or tappable: a review score on a product row, or the "how was it?" prompt after an order.

	comps.Rating{Value: 3, OnChange: stars.Set}              // interactive
	comps.Rating{Value: 4.5, ReadOnly: true, Label: "Score"} // display only

	┌ Row  role=group  label="Rating"  value="3 of 5" ─┐
	│   ★     ★     ★     ☆     ☆                      │
	│  btn   btn   btn   btn   btn   (interactive only) │
	└───────────────────────────────────────────────────┘

#### Value is a float so half-stars need no signature change

By default it rounds to the nearest whole glyph (4.5 draws five). Halves draws the half-glyph this field was typed for: the value is rounded to the nearest half instead, and a caller storing an average never changed type. OnChange reports whole numbers either way, because a tap lands on a whole glyph.

#### Halves are drawn, not typed

Half a "★" would be a text glyph clipped by a half-width box, and that is the one thing a text node cannot be trusted to do: SwiftUI truncates a Text proposed less than its width to "…" rather than letting it overflow to be clipped, and Compose would wrap or squeeze it. So with Halves set every position is a small core.Canvas star — the same shape for full, empty and half, so a half sits beside its neighbours as one drawing — and the half is exact geometry rather than a clip: a regular five-point star is symmetric about its vertical axis, which runs through the top point and the bottom inner vertex, so its left half is the polygon of the vertices on that side.

	        0  (top point, on the axis)
	       ╱╲
	8 ────9  1──── 2        left half = 0 → 9 → 8 → 7 → 6 → 5 → 0
	   ╲        ╱
	    7      3            k: angle −90° + 36°·k, radius R (even k)
	   ╱   5    ╲               or R·0.382 (odd k)
	  6 ╱    ╲   4
	         (5 is the bottom inner vertex, on the axis)

Glyph and EmptyGlyph do not apply to a Halves rating, since its glyphs are drawings. Without Halves the tree is exactly what it was before the field existed.

#### Interactive glyphs are buttons; read-only glyphs are decoration

An interactive glyph is a Box with RoleButton, named "3 of 5", so each star is a separate, labelled tab stop and activation target. Tapping the star that is already the value does nothing, the same "report only changes" rule Stepper follows. A read-only rating registers no callbacks and hides the glyphs from assistive technology: the group's value ("4 of 5") is the one announcement, instead of five "black star" readings.

#### Accessibility

The row is RoleGroup with Label (default "Rating") as its name and the rounded score as an AccessibilityValue. As with Stepper, the natives read the value's text on any node and the web scopes aria-value\* to progressbar, so on the web the read-only group is announced by name and the per-star labels carry the score for the interactive one.

#### Theme roles read

	Filled glyph   Colors.WarningOnLightColor() — amber that holds contrast on
	               a light surface, where Colors.Warning is about 2:1
	Empty glyph    Colors.TextSecondary
	Glyph size     Typography.Subtitle
	Gap            Spacing.XS

<small>[comps/rating.go:79](https://github.com/rohanthewiz/grmob/blob/master/comps/rating.go#L79)</small>

#### func (Rating) Render

```go
func (r Rating) Render(ctx *core.Context) *core.Node
```

Render builds the glyph row described in the type doc.

<small>[comps/rating.go:110](https://github.com/rohanthewiz/grmob/blob/master/comps/rating.go#L110)</small>

### type SegmentedControl

```go
type SegmentedControl struct {
	// Labels are the segment captions, left to right. Selected indexes this
	// slice.
	Labels []string

	// Selected is the index of the active segment. Out of range selects
	// nothing.
	Selected int

	// OnSelect fires with the tapped segment's index.
	//
	// A nil OnSelect renders an inert control rather than one that panics:
	// core.Button registers whatever handler it is given, and a nil func in
	// the registry crashes when a native tap dispatches to it.
	OnSelect func(int)

	// Segment is the template every segment is rendered from. Its Label,
	// Selected and OnTap are overwritten per segment; everything else —
	// Style, SelectedStyle, AccessibilityHint — applies to all of them.
	Segment Chip

	// SegmentLabel derives a segment's accessibility label from its caption
	// and index. Nil leaves Chip to announce the caption itself. Which
	// segment is live is announced separately, as a control state, so this
	// returns the name only and the name does not change when the selection
	// moves.
	SegmentLabel func(label string, index int) string

	// KeyPrefix is prepended to each segment's reconciler key, which is
	// otherwise the caption. Set it when two controls on one screen could
	// otherwise draw from the same captions — keys only have to be unique
	// among siblings, but a prefix also makes a debug-mode duplicate-key
	// concern name the control it came from.
	//
	// Captions are assumed distinct. Two segments with the same caption
	// collide, which debug mode reports rather than silently mismatching
	// rows — a segmented control with two identical captions is a bug in the
	// caller either way.
	KeyPrefix string

	// Gap is the horizontal spacing between segments, in points. Zero means
	// the theme's SM step, not zero spacing — the segments are the control's
	// own internal layout, the same reasoning InputRow's Gap carries. Ask for
	// no gap through Style: []core.StyleProp{core.Gap(0)}.
	Gap float64

	// Style is applied to the row after Gap, so it overrides it.
	Style []core.StyleProp
}
```

SegmentedControl is a controlled single-select rendered as a row of chips — a filter bar, a mode switcher, a scope picker.

	Row (Gap)
	  ├─ Chip "All"     ← Selected == 0
	  ├─ Chip "Active"
	  └─ Chip "Done"

It is the extraction of todoapp's filter bar, which was the loop the Chip widget itself came out of. Chip solved one segment; what stayed hand-written was everything around it — the row, the gap, the keying, the index comparison, and the per-segment accessibility label. Those are the parts with the quiet failure modes: forget the key and the reconciler matches segments by position, forget the accessibility label and a screen reader reads three unnamed buttons.

	comps.SegmentedControl{
	    Labels:   []string{"All", "Active", "Done"},
	    Selected: filter.Get(),
	    OnSelect: func(i int) { filter.Set(i) },
	}

#### Selection is an index, and the caller owns it

The control holds no state: it renders Selected and reports taps. That is the same contract Chip has, one level up, and it is what lets the selected index be the app's own filter enum — todoapp's filterAll/filterActive/ filterDone are literally indices into Labels.

A Selected outside the range of Labels selects nothing. That is a legal state, not a defensive check: a scope picker that starts with no scope chosen says so with -1 rather than by adding a fourth "none" segment.

#### Segment is a template, not a set of pass-through fields

Everything a Chip can do — Style, SelectedStyle, the accessibility hint — is set once on Segment and applies to every segment; Label, Selected and OnTap are filled in per segment and any value set for them on the template is ignored, since those three are exactly what the control is computing. The alternative was re-exporting Chip's surface as SegmentStyle, SelectedSegmentStyle, SegmentHint and so on, which grows a field every time Chip does. This is the move InputRow already makes with Button.

#### Why the accessibility label is a function

It is the one thing that genuinely varies per segment and is not derivable from the caption: todoapp announces "Show active tasks" for a chip captioned "Active". A parallel \[]string would have to be kept in step with Labels by hand, so it is a function of the caption instead, and nil means "let Chip use the caption itself".

#### What a screen reader makes of the row

A group of toggle buttons, which is what this is: every segment states core.AccessibilitySelected (Chip does it, per segment), so the live one announces as pressed and the others as not. Nothing here claims the row is anything in particular — no landmark, no structural role — because a segmented control is not one thing on every screen it appears on.

It becomes a \*tab strip\* the moment its segments switch what the screen below is showing, and that is a claim only the caller can make. It takes two props and no new field:

	comps.SegmentedControl{
	    Labels:   []string{"Sermons", "Articles"},
	    Selected: tab.Get(),
	    OnSelect: func(i int) { tab.Set(i) },
	    Style:    []core.StyleProp{core.AccessibilityRole(core.RoleTabList)},
	    Segment:  comps.Chip{Style: []core.StyleProp{core.AccessibilityRole(core.RoleTab)}},
	}

The state each Chip already sets then goes out as aria-selected instead of aria-pressed, because the two web exporters pick the attribute from the role — nothing in this widget or in Chip has to know which arrangement it is in. See core.Style.AccessibilitySelected.

Two things that arrangement does not buy, both of which are ARIA's rules rather than this widget's limits. A tablist claims its children are tabs, so a row that also holds a count or an add button is not one (see core/role.go). And the panel the tabs control cannot be pointed at from here: aria-controls is an IDREF, and core.Style carries values rather than references — a real wired tab strip is core.TabView, which owns both ends of that relationship.

<small>[comps/segmented_control.go:88](https://github.com/rohanthewiz/grmob/blob/master/comps/segmented_control.go#L88)</small>

#### func (SegmentedControl) Render

```go
func (s SegmentedControl) Render(ctx *core.Context) *core.Node
```

<small>[comps/segmented_control.go:138](https://github.com/rohanthewiz/grmob/blob/master/comps/segmented_control.go#L138)</small>

### type Stepper

```go
type Stepper struct {
	// Value is the caller's current number.
	Value int

	// Min and Max bound the value when Max > Min; otherwise it is unbounded.
	Min, Max int

	// Step is the amount one tap adds or removes. Zero or negative means 1.
	Step int

	// OnChange receives the new, clamped value. It is not called when a tap
	// would leave the value unchanged.
	OnChange func(int)

	// Label is the group's accessible name ("Quantity"). It is not drawn:
	// place the stepper in a ListRow's Trailing or a FormField for a visible
	// label, which keeps the stepper usable inside either.
	Label string

	// Format renders the value between the buttons and in the accessibility
	// value. Nil uses strconv.Itoa.
	Format func(int) string

	// Disabled disables both buttons regardless of the bounds.
	Disabled bool

	// DecreaseLabel and IncreaseLabel name the two buttons for screen readers.
	// Empty uses "Decrease" and "Increase".
	DecreaseLabel, IncreaseLabel string

	// Style is applied to the row after the widget's own props.
	Style []core.StyleProp
}
```

Stepper is a number with a − and a + beside it: a quantity in a cart, the guests on a booking, the font size in a settings screen.

	comps.Stepper{Value: qty.Get(), Min: 1, Max: 20, OnChange: qty.Set, Label: "Quantity"}

core.NumericInput is the typing form of the same value. A Stepper is the tapping form, for small ranges where two taps beat opening a keyboard. Chapter 1 of the tutorial once hand-rolled it three times (FontSize, Gap and BorderRadius); those lessons now build theirs from this type.

	┌ Row  role=group  label=Label  value="3" ─┐
	│  [ − ]      3      [ + ]                 │
	└──────────────────────────────────────────┘
	   disabled          disabled
	   at Min            at Max

#### Controlled, and clamped in the widget

The widget holds no state. A tap computes Value∓Step, clamps it into the bounds and calls OnChange only when the result differs from Value, so a caller's handler is a plain setter and never sees a no-op or an out-of-range number. At a bound the button that would leave the range is rendered Disabled, which is what both natives and the browser announce as unavailable; the handler still stays registered, per core.Style.Disabled.

#### Bounds are opt-in: they apply when Max > Min

Go has no "unset" int, and Min: 0, Max: 0 is not a range anyone means, so a zero-value pair leaves the stepper unbounded. Any range where Max > Min is enforced at both ends; a caller wanting only a floor sets a large Max.

#### Buttons are outlined, not ghost

The plan sketched ghost buttons to keep the pair quiet. A ghost "−" is a bare glyph with no visible edge, which on a phone reads as text rather than as a target, so both buttons are EmphasisOutlined — the treatment the tutorial's hand-rolled stepper had already settled on.

#### Accessibility

There is no spinbutton role in core.Role and none is added for this: the row is RoleGroup with Label as its name and the current value as an AccessibilityValue. Both natives read the value's Text on any node (Compose stateDescription, SwiftUI accessibilityValue). The web scopes aria-value\* to progressbar and drops it here, and on the web the visible number between the buttons is read in document order instead. The buttons are named "Decrease"/"Increase" by default, because "−" is announced as "minus" or not at all; DecreaseLabel and IncreaseLabel localise them.

#### Theme roles read

	Row gap         Spacing.SM
	Value text      Typography.Body, bold
	Buttons         comps.Button outlined: Colors.Primary's on-light tone

<small>[comps/stepper.go:63](https://github.com/rohanthewiz/grmob/blob/master/comps/stepper.go#L63)</small>

#### func (Stepper) Render

```go
func (s Stepper) Render(ctx *core.Context) *core.Node
```

Render builds the group row described in the type doc.

<small>[comps/stepper.go:125](https://github.com/rohanthewiz/grmob/blob/master/comps/stepper.go#L125)</small>

### type Variant

```go
type Variant string
```

Variant selects a widget's semantic color role — what a piece of UI \*means\* rather than what it looks like. It is shared across the package rather than owned by Badge so Badge, Button, Banner and any later status surface resolve the same four roles the same way, and so a caller can pass one value around. (This once said "a future Alert, Banner": Banner is that Alert — the inline status strip — so no separate Alert is planned.)

It is a string enum with an empty zero value, matching core's Alignment and DisplayMode. That is load-bearing here: the zero value must be the existing look, or adding the field would restyle every Badge already in a tree.

<small>[comps/variant.go:20](https://github.com/rohanthewiz/grmob/blob/master/comps/variant.go#L20)</small>

```go
const (
	// VariantDefault is the zero value: the theme's Primary, the badge look
	// that predates variants.
	VariantDefault Variant = ""
	VariantSuccess Variant = "success"
	VariantWarning Variant = "warning"
	VariantError   Variant = "error"
)
```

#### func (Variant) Color

```go
func (v Variant) Color(t *core.Theme) string
```

Color resolves the variant to a background from the theme's palette.

Success and Warning go through their resolver methods so a theme predating those roles falls back to a visible default rather than to no color; Error is one of the palette's original seven and is read directly, since no theme can be missing it.

<small>[comps/variant.go:37](https://github.com/rohanthewiz/grmob/blob/master/comps/variant.go#L37)</small>

#### func (Variant) Ink

```go
func (v Variant) Ink(t *core.Theme, bg string) string
```

Ink returns the label color to lay over bg.

##### Why this is computed rather than a fixed pairing

The palette names one color per role and no ink to go with it, so a status fill arrives without a partner. Picking one badly is not a cosmetic problem: under DefaultTheme, white on Success (#34C759) is 2.22:1 and white on Warning (#FF9500) is 2.20:1 — below even the 3:1 large-text floor, i.e. a badge nobody can read. Black on those two is ~9.5:1.

A fixed per-variant pairing would not survive a theme swap either, because the correct ink \*flips direction\* between the two bundled themes: Success is a light green under DefaultTheme (wants dark ink) and a dark green under MaterialTheme (wants light ink). So the choice is made per color, against the theme's own two ink roles, at render time.

##### The variant is not consulted, and used to be

VariantDefault had an arm of its own here that returned the theme's Background whatever bg was, to keep the Primary/Background pairing both bundled themes chose and Button paints. That is still the answer it gets — but it is now reached by asking the theme rather than by exempting a constant, which is inkOn's whole subject. Two things fall out of the swap:

  - Badge{Color: "#FFF9C4"} with no variant used to get white ink on pale yellow, because the exemption ignored bg entirely. Badge's own doc already promised the opposite ("resolved against bg, so an explicit Color still gets a legible ink picked for it"); it is true now.
  - A theme that states no Components.Button base at all — examples exist, see the Components note in examples/fintechapp — has declared no pairing, so its default variant is measured like any other. That is the one case whose pixels move, and towards the more legible ink.

<small>[comps/variant.go:117](https://github.com/rohanthewiz/grmob/blob/master/comps/variant.go#L117)</small>

#### func (Variant) OnLight

```go
func (v Variant) OnLight(t *core.Theme) string
```

OnLight resolves the variant to the ink-weight tone of its role — the value to spend when the color \*is\* the ink, rather than the fill something else is laid over.

Color and this are the two halves of one role, and which one a widget wants is decided by what it does with it:

	Color     a fill. The ink over it is chosen by contrast (Ink, below), so
	          a mid-tone works and the pair clears AA on every bundled theme.
	OnLight   ink itself — an outlined button's label and rule, a loud chip's
	          outline. The backdrop is whatever the widget was placed on,
	          which the widget cannot see, so the value has to be dark enough
	          to be read against a light surface on its own.

VariantDefault resolves through the palette's Primary tone here, with no special arm, and the reason is that there is nothing for one to preserve. Ink's answer for the default is a \*pairing\* the theme itself declares (Background over Primary, which is what Button already paints, and which Ink now reads back rather than assuming); no theme declares anything about a role spent as ink on an unknown backdrop, because before these tones existed every caller spent the role colour raw — which is exactly what the unset fallback still returns.

<small>[comps/variant.go:72](https://github.com/rohanthewiz/grmob/blob/master/comps/variant.go#L72)</small>

