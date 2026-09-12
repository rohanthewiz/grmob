# Package components

```go
import "github.com/rohanthewiz/grmob/components"
```

Package components is grmob's widget library: higher-level UI pieces built entirely on the public core API, in the idiom of element's components package (Workstream 3 of the element-lessons plan).

## The struct-widget idiom

Every widget here is a struct implementing core.View, configured through named fields:

	components.Card{
	    Title: "Account",
	    Body:  balanceSummary,
	    Footer: components.Badge{Text: "verified"},
	}

Structs, not more constructor funcs in core, for two reasons. Named fields scale to many optional knobs where positional arguments do not — a widget can grow a field without breaking a single call site. And a core.View-typed field is a natural composition slot: Card's Header/Body/Footer accept any view, the way element's Card distinguishes Body (a string) from BodyComponent (a component). Where a widget offers both a simple path and a slot (Card.Title vs Card.Header), the slot wins when both are set.

## Discipline

The package deliberately lives outside core and touches nothing internal: if a widget can't be built out here, that is a gap in core's primitives, not a reason to reach inside. Widgets take their look from ctx.Theme() — colors come from the palette, sizes from the spacing/typography scales, never hard-coded — and accept Style overrides for per-use adjustment. (core keeps the widgets it already had — modal, toast, tabview; new widgets land here.)

## Hooks inside widgets

A widget's Render receives the caller's Context, so the hook rules apply exactly as they do to any component: a widget that calls NewState consumes a positional slot on the caller's context and must therefore be rendered unconditionally, every pass, like any other hook user. core.SetDebugMode flags violations as cursor-drift concerns.

Two widgets do: Accordion (expanded or collapsed) and DatePicker (is the sheet open, which month is being browsed). Both own state that is purely about the widget's own presentation, which is the bar — anything an application might want to read, drive or persist stays with the caller. Calendar is the counter-example worth keeping in view: the month on screen looks like private view state and is not, because a screen opening on the month of its next event has to be able to say so, so Calendar takes no hooks and DatePicker is where that state gets packaged for the form case.

## Index

- [Constants](#constants) — `ColorTransparent`, `ConcernNoMapProvider`, `ConcernPartialSort`, `DefaultMapHeight`, `DefaultMapPanelHeight`, `DefaultMapScale`, `DefaultMapWidth`, `DefaultMapZoom`, `FitPadding`, `MaxFitZoom`, `MaxGoogleMapScale`, `MaxMapDimension`, and 5 more
- [Variables](#variables) — `RichToolbarDefault`
- [`func FitRegion`](#func-fitregion)
- [`func GoogleMapsHandoff`](#func-googlemapshandoff)
- [`func OSMStaticMap`](#func-osmstaticmap)
- [`func OpenStreetMapHandoff`](#func-openstreetmaphandoff)
- [`func PlaceCount`](#func-placecount)
- [`type Accordion`](#type-accordion)
    - [`func (Accordion) Render`](#func-accordion-render)
- [`type AppBar`](#type-appbar)
    - [`func (AppBar) Render`](#func-appbar-render)
- [`type Avatar`](#type-avatar)
    - [`func (Avatar) Render`](#func-avatar-render)
- [`type Badge`](#type-badge)
    - [`func (Badge) Render`](#func-badge-render)
- [`type Banner`](#type-banner)
    - [`func (Banner) Render`](#func-banner-render)
- [`type Button`](#type-button)
    - [`func (Button) Render`](#func-button-render)
- [`type Calendar`](#type-calendar)
    - [`func (Calendar) Render`](#func-calendar-render)
- [`type Card`](#type-card)
    - [`func (Card) Render`](#func-card-render)
- [`type Chip`](#type-chip)
    - [`func (Chip) Render`](#func-chip-render)
- [`type ChipStrip`](#type-chipstrip)
    - [`func (ChipStrip) Render`](#func-chipstrip-render)
- [`type CodeEditor`](#type-codeeditor)
    - [`func (CodeEditor) Render`](#func-codeeditor-render)
- [`type Collapse`](#type-collapse)
- [`type CollapseBand`](#type-collapseband)
    - [`func (CollapseBand) Render`](#func-collapseband-render)
- [`type Column`](#type-column)
- [`type Compass`](#type-compass)
    - [`func (Compass) Render`](#func-compass-render)
- [`type DataTable`](#type-datatable)
    - [`func (DataTable) Render`](#func-datatable-render)
- [`type DatePicker`](#type-datepicker)
    - [`func (DatePicker) Render`](#func-datepicker-render)
- [`type Emphasis`](#type-emphasis)
- [`type EmptyState`](#type-emptystate)
    - [`func (EmptyState) Render`](#func-emptystate-render)
- [`type FormField`](#type-formfield)
    - [`func (FormField) Render`](#func-formfield-render)
- [`type Group`](#type-group)
- [`type GroupHeader`](#type-groupheader)
    - [`func (GroupHeader) Render`](#func-groupheader-render)
- [`type GroupedList`](#type-groupedlist)
    - [`func (GroupedList) AutoLoadWithheld`](#func-groupedlist-autoloadwithheld)
    - [`func (GroupedList) Render`](#func-groupedlist-render)
- [`type InputRow`](#type-inputrow)
    - [`func (InputRow) Render`](#func-inputrow-render)
- [`type ListRow`](#type-listrow)
    - [`func (ListRow) Render`](#func-listrow-render)
- [`type LoadMore`](#type-loadmore)
    - [`func (LoadMore) Render`](#func-loadmore-render)
- [`type MapHandoff`](#type-maphandoff)
- [`type MapPanel`](#type-mappanel)
    - [`func (MapPanel) Render`](#func-mappanel-render)
- [`type MapPin`](#type-mappin)
- [`type Pagination`](#type-pagination)
    - [`func (Pagination) Render`](#func-pagination-render)
- [`type ProgressBar`](#type-progressbar)
    - [`func (ProgressBar) Render`](#func-progressbar-render)
- [`type Prominence`](#type-prominence)
- [`type RichTextEditor`](#type-richtexteditor)
    - [`func (RichTextEditor) Render`](#func-richtexteditor-render)
- [`type RichToolItem`](#type-richtoolitem)
- [`type RichToolbar`](#type-richtoolbar)
    - [`func UseRichToolbar`](#func-userichtoolbar)
    - [`func (*RichToolbar) Selection`](#func-richtoolbar-selection)
- [`type Screen`](#type-screen)
    - [`func (Screen) Render`](#func-screen-render)
- [`type SearchField`](#type-searchfield)
    - [`func (SearchField) Render`](#func-searchfield-render)
- [`type SegmentedControl`](#type-segmentedcontrol)
    - [`func (SegmentedControl) Render`](#func-segmentedcontrol-render)
- [`type Separator`](#type-separator)
    - [`func (Separator) Render`](#func-separator-render)
- [`type Skeleton`](#type-skeleton)
    - [`func (Skeleton) Render`](#func-skeleton-render)
- [`type Sort`](#type-sort)
- [`type StatTile`](#type-stattile)
    - [`func (StatTile) Render`](#func-stattile-render)
- [`type StaticMap`](#type-staticmap)
    - [`func (StaticMap) Area`](#func-staticmap-area)
    - [`func (StaticMap) Render`](#func-staticmap-render)
- [`type StaticMapArea`](#type-staticmaparea)
- [`type StaticMapProvider`](#type-staticmapprovider)
    - [`func GoogleStaticMap`](#func-googlestaticmap)
- [`type Tabs`](#type-tabs)
    - [`func (Tabs) Render`](#func-tabs-render)
- [`type Variant`](#type-variant)
    - [`func (Variant) Color`](#func-variant-color)
    - [`func (Variant) Ink`](#func-variant-ink)
    - [`func (Variant) OnLight`](#func-variant-onlight)

## Constants

```go
const (
	// DefaultMapPanelHeight is a map tall enough to have a shape and short
	// enough to leave a caption and a row or two of context on a phone screen.
	DefaultMapPanelHeight = "280px"

	// MinFitSpread is the tightest view FitRegion will open at, in degrees of
	// latitude — about 550 metres, which is a few blocks.
	//
	// It exists because a single pin has a spread of zero and the logarithm of
	// a division by zero is not a zoom level. A floor rather than a branch:
	// one rule, no arm that runs only for one point, and the floor is the view
	// a single pin wants anyway.
	MinFitSpread = 0.005

	// FitPadding widens the fitted span so the outermost pins are not against
	// the edge of the frame. A fifth, which is about one pin's height at the
	// sizes a panel is drawn at.
	FitPadding = 1.2

	// MinFitZoom and MaxFitZoom bound the answer. 19 is as deep as the
	// standard tile pyramid goes; 1 is a view half the globe wide, which is
	// the widest this returns.
	//
	// Not 0, even though 0 is the world and would fit anything: core.Region
	// reads a zero Zoom as "unstated" and substitutes core.DefaultMapZoom, so
	// returning 0 here would hand back a neighbourhood view of a set spanning
	// continents — the exact opposite of what was asked for. See
	// core.Region.Zoom, which explains why zero cannot mean the world.
	MinFitZoom = 1.0
	MaxFitZoom = 19.0
)
```

<small>[components/map_panel.go:116](https://github.com/rohanthewiz/grmob/blob/master/components/map_panel.go#L116)</small>

The defaults and the one clamp, named so a caller can reason about them without reading the render function.

```go
const (
	// DefaultMapZoom is street level: a block or two across the image, which
	// is the scale at which "where is this" is legible. 15 on the slippy
	// scale, the same number both providers mean by it.
	//
	// One level deeper than core.MapView's default, and the gap is the
	// difference between the two widgets: a static map answers "where is this
	// one place", a live map is usually showing a set, and one level out is
	// about four times the area.
	DefaultMapZoom = 15

	// DefaultMapWidth and DefaultMapHeight are a 16:9 panel the width of a
	// phone card. Landscape because a map of one point has nothing to say
	// vertically that it does not say horizontally, and a tall map wastes the
	// screen a caption wants.
	DefaultMapWidth  = 320
	DefaultMapHeight = 180

	// MaxMapDimension is the smaller of the two bundled providers' limits.
	// Google's free static API refuses above 640 per side; the OSM service's
	// ceiling is the same order. See StaticMap.Width for why this clamps
	// rather than passing the request through to be refused.
	MaxMapDimension = 640

	// DefaultMapScale is one image pixel per logical pixel, which is what a
	// caller who says nothing about the device gets — and what every caller
	// got before Scale existed. See "Device pixel ratio" on StaticMap for why
	// it is not read off the screen.
	DefaultMapScale = 1

	// MaxMapScale is 3, the deepest ratio shipping phones use. It clamps the
	// *request*, and is not a promise about the answer: Google's API tops out
	// at MaxGoogleMapScale and a provider with no scale of its own serves 1x
	// whatever it is handed.
	MaxMapScale = 3

	// MaxGoogleMapScale is what the Maps Static API accepts — 1 or 2, and
	// nothing else. A 3x device therefore gets the 2x image, which is the
	// sharpest thing that API will serve and still four times the pixels of
	// the 1x it used to get.
	//
	// Google applies its size limit *before* scaling, which is why Width and
	// Height are not also divided down to make room: a scale=2 request for a
	// 640px box is accepted and served as 1280px.
	MaxGoogleMapScale = 2

	// MercatorLatLimit is where Web Mercator stops. Every tile provider here
	// projects with it, so this is the edge of the addressable world rather
	// than a choice — see StaticMap.Lat.
	MercatorLatLimit = 85.0511
)
```

<small>[components/static_map.go:235](https://github.com/rohanthewiz/grmob/blob/master/components/static_map.go#L235)</small>

ColorTransparent is a fully transparent fill, written in the CSS byte order (#RRGGBBAA) that every target parses: htmlout emits it verbatim as a CSS Color 4 hex, and both native parseColor implementations handle the 8-digit form with alpha last.

It exists because core.Style has no "clear" or "unset" for a color, and an \*empty\* Background is not transparent — it means "inherit the theme's Button base", which is a solid Primary fill. Outlined and Ghost need an actual hole, not an omission.

```go
const ColorTransparent = "#00000000"
```

<small>[components/button.go:14](https://github.com/rohanthewiz/grmob/blob/master/components/button.go#L14)</small>

ConcernNoMapProvider: a StaticMap rendered with no Provider, which draws an empty frame. It is a development-time finding rather than a panic because the failure is survivable — a screen missing its map is still a screen — and because the fix is configuration, which is exactly the class of mistake that is invisible in a running app and obvious in a concern list.

```go
const ConcernNoMapProvider = "no-map-provider"
```

<small>[components/static_map.go:318](https://github.com/rohanthewiz/grmob/blob/master/components/static_map.go#L318)</small>

ConcernPartialSort: a DataTable sorted client-side (the active Sort names a column with a Less) while its Pagination declares a PageCount — which is the caller saying the server chooses which rows arrive. The table can only order the window it was handed, so the header claims an ordering over the whole table and delivers one over one page of it. The fix is to drop the column's Less and keep Sortable, letting OnSort go into the query.

Reported through core.ReportConcern rather than detected in core: this is a widget-level contract, and core has no business knowing what a DataTable is. Debug mode only, like every other concern.

```go
const ConcernPartialSort = "partial-sort"
```

<small>[components/data_table.go:64](https://github.com/rohanthewiz/grmob/blob/master/components/data_table.go#L64)</small>

RichToolLink is the sentinel Command that opens the link prompt. Not a core command: core.EditLink needs a URL, and the prompt is where one comes from.

```go
const RichToolLink = "components:link"
```

<small>[components/rich_text_editor.go:105](https://github.com/rohanthewiz/grmob/blob/master/components/rich_text_editor.go#L105)</small>

## Variables

RichToolbarDefault is the toolbar most notes want: the five marks, three block kinds, the two lists, a quote, and the link prompt.

A var rather than a func so a caller can take a slice of it, append to it, or reorder it — which is the whole reason the toolbar is a list of items rather than a bool. It is package state, so treat it as read-only; UseRichToolbar copies it.

```go
var RichToolbarDefault = []RichToolItem{
	{Label: "B", Command: core.EditBold, AccessibilityLabel: "Bold"},
	{Label: "I", Command: core.EditItalic, AccessibilityLabel: "Italic"},
	{Label: "U", Command: core.EditUnderline, AccessibilityLabel: "Underline"},
	{Label: "S", Command: core.EditStrike, AccessibilityLabel: "Strikethrough"},
	{Label: "</>", Command: core.EditCode, AccessibilityLabel: "Inline code"},
	{Label: "H1", Command: core.EditBlock(richtext.Heading1), AccessibilityLabel: "Heading 1"},
	{Label: "H2", Command: core.EditBlock(richtext.Heading2), AccessibilityLabel: "Heading 2"},
	{Label: "¶", Command: core.EditBlock(richtext.Paragraph), AccessibilityLabel: "Paragraph"},
	{Label: "•", Command: core.EditBlock(richtext.Bullet), AccessibilityLabel: "Bulleted list"},
	{Label: "1.", Command: core.EditBlock(richtext.Numbered), AccessibilityLabel: "Numbered list"},
	{Label: "❝", Command: core.EditBlock(richtext.Quote), AccessibilityLabel: "Quote"},
	{Label: "🔗", Command: RichToolLink, AccessibilityLabel: "Add link"},
}
```

<small>[components/rich_text_editor.go:114](https://github.com/rohanthewiz/grmob/blob/master/components/rich_text_editor.go#L114)</small>

## Functions

### func FitRegion

```go
func FitRegion(pins []MapPin) (core.Region, bool)
```

FitRegion is the view that contains every pin: a centre and the zoom level at which the whole set is on screen.

Exported separately from the widget because it is the reusable half. A caller driving core.MapView directly — holding the region in state, echoing OnRegionChange — still needs this answer for its opening view, and the alternative is every such caller deriving the same logarithm.

	zoom ≈ log2(360 / spread in degrees)

#### The projection correction

The longitude spread is scaled by the cosine of the centre latitude before the two spans are compared. A degree of longitude narrows towards the poles and a degree of latitude does not, so a fit computed from raw degrees is too tight in Reykjavík and about right in Quito. Whichever span needs the wider view decides.

Very near a pole the cosine approaches zero and the correction stops being meaningful, so it is floored — a map that far north is showing one place anyway.

#### Two sets it does not fit, and says so here rather than pretending

A set spanning more than half the globe is clamped to MinFitZoom rather than contained: that is a view 180 degrees wide, and the alternative would be a zoom core.Region cannot express (see MinFitZoom). Pins outside it are off screen, and a map of the whole planet would not have shown them usefully anyway.

A set straddling the antimeridian used to be measured the long way round: Tokyo and Honolulu are 62 degrees apart going east and were read as 298 going the other way, so the fit opened on the Atlantic with both pins off the edges. That is fixed, and the fix is longitudeSpan — the smallest arc of longitude containing every point, found as the complement of the widest gap between adjacent points.

The thing that makes it a fix rather than a trade is that the answer does not change for any set that does not straddle. For those, the widest gap IS the one that wraps from the easternmost point back round to the westernmost, so its complement runs from min to max and the centre is the same place the average used to give, arrived at by the general rule.

The same \*place\*, not the same float. The old path added two longitudes and halved; this one adds half a width to an endpoint, which is one subtraction fewer and cancels less. A real set of pins in Austin moved from -97.75800000000001 to -97.758 — fourteen significant figures of agreement, a few nanometres of ground, and a re-recorded snapshot. Worth saying plainly because "nothing moves" is what this paragraph wanted to claim and is not quite what is true.

The centre it returns is the middle of that arc, which is the contract a \*fit\* wants. A circular mean — the direction of the summed unit vectors — is the other candidate and is the wrong one here: it is pulled by clusters, so nineteen pins in Tokyo and one in Honolulu would centre on Tokyo and leave the twentieth off screen, which is precisely what a fit must not do.

#### An empty set has no answer

FitRegion of nothing returns the zero Region and false. There is no sensible centre for no points, and the tempting answer — 0,0 — is a real place in the Gulf of Guinea that a map will happily draw. Same rule as components.StaticMap's "Zero is a place": a caller with nothing to show must not render the map at all.

<small>[components/map_panel.go:212](https://github.com/rohanthewiz/grmob/blob/master/components/map_panel.go#L212)</small>

### func GoogleMapsHandoff

```go
func GoogleMapsHandoff(lat, lng float64, label string) string
```

GoogleMapsHandoff is the default tap target: Google's documented cross-platform maps URL.

It is the one URL all three platforms resolve, and on both phones it reaches the installed maps app rather than a browser tab — which is the whole point of handing off at all.

The label is deliberately not in it, and the reason is worth stating because the URL has a slot that looks like it wants one. \`query\` is a \*search\*: given a name it runs a geocode, which for "St Mary's" lands on whichever St Mary's the search liked, possibly in another country. Given a coordinate pair it lands on the coordinate. The app knows exactly where this place is, so it says so, and the pin is unnamed rather than wrong.

A caller who wants the name \*and\* the point has to pick a platform to say it to — \`geo:lat,lng?q=lat,lng(Label)\` on Android, \`[https://maps.apple.com/?ll=lat,lng&q=Label](https://maps.apple.com/?ll=lat,lng&q=Label)\` on iOS — which is what the label parameter on this signature is for.

<small>[components/static_map.go:422](https://github.com/rohanthewiz/grmob/blob/master/components/static_map.go#L422)</small>

### func OSMStaticMap

```go
func OSMStaticMap(a StaticMapArea) string
```

OSMStaticMap renders through staticmap.openstreetmap.de, the OpenStreetMap community's static-image service.

Deprecated: that service no longer exists. The OpenStreetMap wiki's StaticMapLite page says "This service has been discontinued", and the host stopped resolving — NXDOMAIN, checked 2026-09-11. Every URL this function builds is now a fetch that fails and a frame that stays empty.

It is kept rather than deleted for two reasons. A caller that names it still compiles, which is the difference between a deprecation and a breakage; and a provider that returns a URL to a dead host is a far easier thing to diagnose than a symbol that has gone away, because the URL is printable and the failure is one nslookup from being understood.

It was this package's default, on the strength of being keyless. See "The image is a network fetch, and the provider is required" on StaticMap for what replaced it, which is nothing, and why.

Scale is ignored. The service had no scale parameter while it was up.

The marker style name ("ol-marker") is the service's own vocabulary rather than anything this package defines, which is the general shape of a provider function — it translates a StaticMapArea into one service's dialect and nothing more.

<small>[components/static_map.go:357](https://github.com/rohanthewiz/grmob/blob/master/components/static_map.go#L357)</small>

### func OpenStreetMapHandoff

```go
func OpenStreetMapHandoff(lat, lng float64, label string) string
```

OpenStreetMapHandoff opens the point on openstreetmap.org, for an app that would rather not send its users to Google.

It opens a browser on every platform, including the two with a maps app installed — OSM has no app with a URL scheme to claim the link. That is the trade, and it is the reason this is not the default: directions are what a person taps a map for, and a browser is a worse place to get them.

<small>[components/static_map.go:436](https://github.com/rohanthewiz/grmob/blob/master/components/static_map.go#L436)</small>

### func PlaceCount

```go
func PlaceCount(n int) string
```

PlaceCount is the caption a MapPanel usually wants: how many places are on the map, which is not the same number as how many things the caller has.

A recurring event is many entries and one pin; an item with no coordinates is not on the map at all. Saying "3 places" rather than "3 events" is the difference between a caption and a wrong count, and it is a mistake worth one exported function to not make twice.

<small>[components/map_panel.go:383](https://github.com/rohanthewiz/grmob/blob/master/components/map_panel.go#L383)</small>

## Types

### type Accordion

```go
type Accordion struct {
	Title string
	// Header replaces the default chevron+Title header content when set.
	// The tap target and toggle behavior stay with the Accordion either way.
	Header  core.View
	Content core.View

	// HeadingLevel is where the Title sits in the screen's outline. Zero is
	// level 3 — a disclosure sits inside a section, one tier below a Card
	// title or a GroupedList band — and a screen built entirely of accordions
	// under an AppBar should say 2.
	//
	// It applies to the default header only. A Header replaces that content
	// and is the caller's to describe, on the same division Card.Title and
	// Card.Header draw.
	//
	// # The heading wraps the control, which is ARIA's own accordion shape
	//
	// This is one of the two widgets in the package whose heading is not on
	// the words. The rest put the role on the Text node, because a row also
	// holds a badge or a chevron and a heading spanning the row would be named
	// "▸ What is a hook" rather than "What is a hook". Here the tier rides a
	// Box *around* the header row, and the row is a button:
	//
	//	Box    role=heading, aria-level=3, aria-label="What is a hook"
	//	  Row  role=button,  aria-expanded="false", aria-label="What is a hook"
	//	    Text "▸"        presentational, inside the button
	//	    Text "What is a hook"
	//
	// It took three tries to land there, both of the rejected ones looked
	// right, and the whole argument now lives on components.disclosure — the
	// shared shape this widget and the collapsible GroupedList band are both
	// built out of. It moved there when the band arrived, because two copies
	// of a four-paragraph argument do not stay in step: the failure is not
	// that the second copy is wrong on the day it lands, it is that a later
	// fix to one leaves the other announcing something else.
	//
	// The wrapper is added only for the default header. A Header replaces the
	// content and is the caller's to describe, on the same division Card.Title
	// and Card.Header draw — so a custom header gets the button and its state
	// and no heading at all.
	//
	// See headingLevel in heading.go for the package's outline and for how to
	// ask for a heading with no tier at all.
	HeadingLevel int

	// InitiallyExpanded seeds the state on the first pass only; after that
	// the accordion follows the user's taps.
	InitiallyExpanded bool
	// Style is applied to the outer column.
	Style []core.StyleProp
}
```

Accordion is a collapsible section: a tappable header that shows or hides its Content.

It owns its expanded/collapsed state via NewState, which makes it the one widget in this package with hook obligations: render an Accordion unconditionally, in a stable position, every pass — exactly the rules for calling NewState directly (core.SetDebugMode reports violations as cursor-drift concerns). Content, on the other hand, is only rendered while expanded, so it must not contain hooks of its own: they would come and go with the toggle, which is the conditional-hook bug. Interactive, hook-free content (buttons, inputs bound to parent state) is fine — its callbacks re-register on every pass the content is visible.

<small>[components/accordion.go:17](https://github.com/rohanthewiz/grmob/blob/master/components/accordion.go#L17)</small>

#### func (Accordion) Render

```go
func (a Accordion) Render(ctx *core.Context) *core.Node
```

<small>[components/accordion.go:70](https://github.com/rohanthewiz/grmob/blob/master/components/accordion.go#L70)</small>

### type AppBar

```go
type AppBar struct {
	// Title is the screen's name. Subtitle is a quieter second line under it
	// — a count, a date, a connection state.
	Title    string
	Subtitle string

	// Content is the escape hatch for the growing middle: an arbitrary view
	// in place of the Title/Subtitle stack. Takes precedence when set.
	//
	// The middle is rendered either way, empty or not: it is the flexible
	// slot that pins Actions to the trailing edge, so making it conditional
	// would make the pinning conditional. Same rule, and the same reason, as
	// ListRow's middle column.
	Content core.View

	// Leading replaces the automatic back control entirely. A nil Leading
	// with HideBack unset draws the back button when core.CanPop is true.
	Leading core.View

	// HideBack suppresses the automatic back control on a screen that can pop
	// but should not offer it — a wizard step that must be completed or
	// abandoned through its own buttons.
	HideBack bool

	// OnBack replaces core.Pop as what the automatic back control does. Set
	// it to confirm before leaving, or to pop more than one frame; call
	// core.Pop yourself from inside it when the answer is yes.
	OnBack func()

	// BackGlyph is the back control's label. Empty is "‹". It is the one
	// piece of the automatic control worth a field of its own — swapping "‹"
	// for "←" or "Back" otherwise costs the caller the whole Leading slot,
	// CanPop test and Pop wiring included.
	BackGlyph string

	// Actions are the trailing controls, in leading-to-trailing order. Nil
	// entries are skipped, so a conditional action can be a nil variable
	// rather than a filtered slice.
	Actions []core.View

	// HideSeparator drops the hairline under the bar.
	//
	// The rule is on by default because the zero value has to work on the
	// zero-value screen: an unstyled bar sits on the same Background as the
	// content below it, and with nothing between them the title reads as the
	// first line of the page. A bar given its own fill through Style
	// separates itself and will usually want this set.
	HideSeparator bool

	// Style is applied to the bar row, after the widget's own defaults and
	// before the children — so padding, background and alignment are all
	// overridable. It does not reach the separator; use a Leading/Content
	// slot or your own Separator for that.
	Style []core.StyleProp
}
```

AppBar is the title strip at the top of a screen: an optional back affordance, the screen's name, and trailing actions.

	components.AppBar{Title: "Sermons"}
	components.AppBar{Title: "Sermon", Actions: []core.View{shareButton}}

	┌ Box ─────────────────────────────────────────────────────────┐
	│ ┌ Row ───────────────────────────────────────────────────────┤
	│ │ [‹]  ┌ Box FlexGrow(1) ──────────┐  [Action] [Action]      │
	│ │      │ Title                     │                         │
	│ │      │ Subtitle                  │                         │
	│ │      └───────────────────────────┘                         │
	│ ├────────────────────────────────────────────────────────────┤
	│ │ Separator                                                  │
	└─┴────────────────────────────────────────────────────────────┘
	       └──── takes all the slack, pinning Actions right ────┘

#### It is a Row, not a platform navigation bar

grmob has no AppBar node, so there is nothing here that floats over the content, collapses on scroll, claims the status bar, or animates a title between screens. This is an ordinary row of ordinary widgets that happens to sit first in a screen's column — which is what makes it identical on all four targets, and what makes components.Screen's SafeArea, not the bar, responsible for keeping it clear of the notch.

#### The back control appears only when there is somewhere to go

With no Leading and no HideBack, the bar draws a back button exactly when core.CanPop says the navigation stack has a screen underneath. The root of a tab shell therefore gets no arrow without the caller having to say so, and a pushed detail screen gets one without the caller having to wire it — which is the behavior every navigation framework has, reached here through the one question core exposes. See core.CanPop for why asking beats rendering a control that would no-op.

#### Slots beat fields, as everywhere in this package

Content overrides Title/Subtitle and Leading overrides the back control, the same simple-path-plus-slot idiom as Card.Title vs Card.Header. Setting Leading is also how you get a back control the widget would not have drawn — a close button on a modally presented screen, where CanPop is false.

<small>[components/app_bar.go:47](https://github.com/rohanthewiz/grmob/blob/master/components/app_bar.go#L47)</small>

#### func (AppBar) Render

```go
func (a AppBar) Render(ctx *core.Context) *core.Node
```

<small>[components/app_bar.go:103](https://github.com/rohanthewiz/grmob/blob/master/components/app_bar.go#L103)</small>

### type Avatar

```go
type Avatar struct {
	// Src is the image URL. Empty falls back to the initials disc.
	Src string

	// Name is the person's full name: the source of both the derived initials
	// and the accessibility label.
	Name string

	// Initials overrides what the fallback disc shows. Set it when the derived
	// pair is wrong — a mononym, a handle, a name whose ordering the
	// first-word/last-word rule gets backwards.
	Initials string

	// Size is the diameter in px; 0 means 40.
	Size float64

	// Background is the fallback disc's fill; empty uses the theme's Primary.
	// TextColor is the initials' ink; empty uses the theme's Background, which
	// is the palette's designated on-Primary color.
	Background string
	TextColor  string

	// Style is applied last, over both branches, so it can restyle the image
	// and the disc identically (a ring, a shadow, a square crop).
	Style []core.StyleProp

	// AccessibilityLabel names the avatar, overriding Name. See the type
	// comment for what happens when both are empty.
	AccessibilityLabel string
}
```

Avatar is the circular portrait that fronts a person in a list row, a header, or a comment: a remote image when there is one, and initials on a colored disc when there is not.

	components.Avatar{Src: user.PhotoURL, Name: user.Name}   // image, labelled
	components.Avatar{Name: "Ada Lovelace"}                  // "AL" on a disc

#### The circle

Both branches are the same square with BorderRadius = Size/2. An oversized radius would be simpler (Badge uses 999 to get a stadium at any height), but a circle needs the radius to track the diameter exactly: a fixed 999 on a square still yields a circle, yet any caller Style that changes Size would silently keep the old geometry. Deriving it means Size stays the single knob.

#### A note on non-square images

The iOS renderer scales images with .scaledToFit, so a portrait that is not square letterboxes inside the circle rather than filling it (Compose's AsyncImage defaults to Fit as well). That is a renderer-level choice shared with every other Image in the framework, not something Avatar can set from Go today — the fix is a ContentMode prop on Image, which would want its own pass across both renderers.

#### Accessibility

Unlike ListRow, Avatar \*does\* synthesize a label, because it can: an avatar has exactly one meaning, the person it depicts, and Name is that meaning. The rule is:

	AccessibilityLabel set  -> used verbatim
	Name set                -> used as the label
	neither                 -> the node is hidden from assistive tech

The last case is the important one. An avatar with no name is decoration sitting next to text that already names the person, and an unlabeled image in that position is announced as an unhelpful "image" — or, worse, as its URL. Hiding it is the correct default rather than a fallback.

<small>[components/avatar.go:50](https://github.com/rohanthewiz/grmob/blob/master/components/avatar.go#L50)</small>

#### func (Avatar) Render

```go
func (a Avatar) Render(ctx *core.Context) *core.Node
```

<small>[components/avatar.go:81](https://github.com/rohanthewiz/grmob/blob/master/components/avatar.go#L81)</small>

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

	components.Badge{Text: "Paid", Variant: components.VariantSuccess}
	components.Badge{Text: "Expiring", Variant: components.VariantWarning}
	components.Badge{Text: "Failed", Variant: components.VariantError}

The zero value is VariantDefault — the theme's Primary — which is the look every badge written before this field had, so adding it restyles nothing.

The label ink is chosen per variant by contrast against the resolved background rather than fixed, because the palette pairs no ink with a status role and the right answer flips between themes. See Variant.Ink; the short version is that a fixed white ink would render DefaultTheme's Success and Warning badges at ~2.2:1, which is unreadable.

#### Color is not the message

A variant is \*reinforcement\* for Text, never a substitute for it. Nothing here announces "warning" to a screen reader, and a reader who cannot distinguish the tints sees only the label — so the label has to say it ("Overdue", not "!"). That is WCAG 1.4.1, and it is why Variant deliberately does not synthesize an accessibility label the way Avatar does: Avatar has one obvious thing to say, a badge's meaning is already in its Text.

<small>[components/badge.go:34](https://github.com/rohanthewiz/grmob/blob/master/components/badge.go#L34)</small>

#### func (Badge) Render

```go
func (b Badge) Render(ctx *core.Context) *core.Node
```

<small>[components/badge.go:55](https://github.com/rohanthewiz/grmob/blob/master/components/badge.go#L55)</small>

### type Banner

```go
type Banner struct {
	// Text is the message. Content is the escape hatch for the growing middle
	// — an arbitrary view in its place, taking precedence when set.
	Text    string
	Content core.View

	// Variant selects the semantic role: Success, Warning, Error, or the zero
	// value for the theme's Primary — the neutral "here is some information"
	// strip.
	Variant Variant

	// Glyph is the leading mark. Empty takes the variant's default (ⓘ ✓ ⚠ ⊗);
	// NoGlyph drops it entirely, for a strip that should read as quietly as
	// possible.
	Glyph   string
	NoGlyph bool

	// ActionLabel and OnAction render a single trailing action — "Retry",
	// "Reconnect", "Undo". Action is the slot form and takes precedence.
	//
	// The built button is a *default* ghost, not one tinted with the banner's
	// variant. The strip already says what kind of thing it is twice, in the
	// border and the glyph; a third telling would put the least legible
	// combination this package has (see Button's contrast table — outlined
	// and ghost own neither their fill nor their backdrop) on the one control
	// the user is meant to hit.
	ActionLabel string
	OnAction    func()
	Action      core.View

	// OnDismiss adds a trailing ✕ that closes the banner. It is the caller's
	// job to stop rendering the widget; nothing here holds state.
	OnDismiss func()

	// Style is applied after the widget's own treatment, so any of it can be
	// overridden. A caller who wants an edge-to-edge strip with no frame —
	// the shape a banner pinned directly under an AppBar usually wants —
	// spells that out:
	//
	//	Style: []core.StyleProp{core.BorderWidth(0), core.BorderRadius(0)}
	Style []core.StyleProp
}
```

Banner is the inline strip that tells the user something about the screen they are on: a failed refresh over content that is still good, a "Reconnecting…", an offline notice, a "Your changes were saved".

	components.Banner{Text: "Could not refresh. Showing saved copy.",
	    Variant: components.VariantWarning, ActionLabel: "Retry", OnAction: reload}

	┌ Row ─────────────────────────────────────────────────────────┐
	│ ⚠  ┌ Box FlexGrow(1) ───────────────┐  [Retry]  [✕]          │
	│    │ Text                           │                        │
	│    └────────────────────────────────┘                        │
	└──────────────────────────────────────────────────────────────┘

#### It is not a toast

core.ShowToast reaches the platform's own transient overlay and disappears on a timer. A Banner is part of the tree: it stays until the state that produced it changes, which is what a condition the user may need to act on requires. Use the toast for "Copied", the banner for "You are offline".

#### The variant is a tint, not a fill

Badge and a filled Button spend the whole variant color as a background. A strip that runs the width of the screen cannot: a saturated Error red across a screen reads as a failure of the app rather than of one fetch, and the palette carries no muted \*container\* tone to fill with instead. (It carries an on-light tone now, which is the opposite end of the range — ink for a light surface, not a wash to sit behind one — so it does not answer this. A container tone would still be a palette decision, not a Banner one.)

So the variant is spent on the edges: a hairline border and the leading glyph take the role's on-light tone, the fill stays the theme's Surface, and the text keeps the primary ink so it is legible whatever the role. That also means a Banner's contrast does not depend on which variant it is, which the alternatives could not promise.

#### Color is not the message, again

Nothing here announces "error" to a screen reader: the glyph is marked decorative (a reader saying "circled times" is worse than silence) and a border has no voice. Text must therefore carry the meaning on its own — "Could not refresh", not "Something went wrong" next to a red edge. Same rule Badge documents, and the same WCAG 1.4.1 behind it.

#### It announces itself when it appears

A banner is a live region: it turns up because something changed, usually while the reader is somewhere else on the screen, and a message nobody is looking at is a message nobody gets. So the strip carries core.RoleAlert when its variant is Error and core.RoleStatus otherwise — the same split the variant already draws visually, in the one vocabulary that has a word for "interrupt" and a word for "mention at the next pause".

Error interrupts because a failed action is the case where continuing is the wrong thing to do; everything else waits, because "Saved" arriving mid-sentence is how a live region becomes a thing users switch off.

The role is on the strip rather than on the message so that an appearing banner is announced whole — the text, and the label of any action beside it, which is the part the reader needs in order to know what to do about it.

It is a default, not a fixture. Style is applied after the widget's own props, so a caller can name a different role, or core.RoleNone for a strip that is really static content and should not interrupt anything.

<small>[components/banner.go:70](https://github.com/rohanthewiz/grmob/blob/master/components/banner.go#L70)</small>

#### func (Banner) Render

```go
func (b Banner) Render(ctx *core.Context) *core.Node
```

<small>[components/banner.go:113](https://github.com/rohanthewiz/grmob/blob/master/components/banner.go#L113)</small>

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
}
```

Button is a themed action button with two orthogonal color axes and no per-call hex.

	components.Button{Label: "Save", OnTap: save}                                // theme Button base
	components.Button{Label: "Delete", Variant: components.VariantError, OnTap: rm}
	components.Button{Label: "Cancel", Emphasis: components.EmphasisOutlined, OnTap: back}
	components.Button{Label: "Skip", Emphasis: components.EmphasisGhost, OnTap: skip}

#### The zero value applies nothing

Button{Label: l, OnTap: f} renders exactly core.Button(l, f) — not "core.Button re-derived from the palette". With both axes at their zero values the widget contributes no color props at all, so the theme's own Components.Button carries the look through untouched. That matters because a theme's Button base is allowed to differ from Colors.Primary; re-deriving would silently overwrite that choice, and the difference is invisible in the bundled themes where the two happen to agree.

#### Secondary is deliberately not a Variant

Variant carries \*meaning\* — success, warning, error. Secondary is a brand slot a theme may set to anything (MaterialTheme makes it teal), so a "Variant: Secondary" would put a brand color where a reader expects a status. A button that wants the brand's second color says so through Style, which is the escape hatch for exactly the case the semantic roles do not cover:

	components.Button{Label: "Recharge", Emphasis: components.EmphasisOutlined,
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

<small>[components/button.go:145](https://github.com/rohanthewiz/grmob/blob/master/components/button.go#L145)</small>

#### func (Button) Render

```go
func (b Button) Render(ctx *core.Context) *core.Node
```

<small>[components/button.go:195](https://github.com/rohanthewiz/grmob/blob/master/components/button.go#L195)</small>

### type Calendar

```go
type Calendar struct {
	// Month is any instant within the month to draw; only its year and month
	// (and its location) are read. Zero falls back to Selected, then to
	// Today; with all three zero the widget renders nothing.
	Month time.Time

	// OnMonthChange is asked to move a month back or forward, and receives
	// midday on the first of the new month. Nil draws no arrows.
	OnMonthChange func(time.Time)

	// Selected is the highlighted day; zero highlights none. Compared by
	// calendar day in the calendar's location, so any instant during the day
	// selects it.
	Selected time.Time

	// OnSelect fires with the tapped day at midday in the calendar's location
	// (see the type comment). Nil renders an inert grid — a month display
	// rather than a picker.
	OnSelect func(time.Time)

	// Deselectable makes a tap on the already-selected day report the zero
	// time through OnSelect instead of the day, so a calendar standing in for
	// a filter can be un-set without a "Show all" button beside it. Off by
	// default; see "The zero time goes both ways" for why the default is that
	// way round.
	//
	// It changes what a tap *reports* and nothing about how the cell is drawn.
	// What it does change is how well the announcement fits: every cell states
	// core.AccessibilitySelected, which reaches the web as aria-pressed, and a
	// pressed toggle button that un-presses when you activate it is exactly
	// what a deselectable day is. Without this field the cell is a toggle that
	// only turns on, which is the honest report of a grid where the selection
	// can move but not clear.
	//
	// That state used to be a ", selected" suffix on the spoken name, because
	// core.Style had no slot for a state. It has one now, and the suffix is
	// gone — see dayLabel.
	Deselectable bool

	// Today rings the current day without selecting it, so "today" and "the
	// day I picked" can be two different cells and both be visible. Zero
	// draws no ring; the widget does not consult the clock.
	Today time.Time

	// Min and Max bound the selectable range, inclusive and compared by
	// calendar day — a Max of "today at 15:04" still includes today. Days
	// outside are drawn like adjacent days and are inert, and an arrow whose
	// whole target month lies outside the range is disabled.
	//
	// Zero on either side is unbounded.
	Min time.Time
	Max time.Time

	// Marked counts what a day has on it — events, deadlines, services — and
	// the cell draws that many dots under its number, capped at
	// calendarMaxDots. Zero draws none, and so does a negative; nil is the
	// same as a function that always answers zero.
	//
	// A count rather than a bool because two services on one Sunday and one
	// service on one Sunday are different facts about the day, and a reader
	// scanning a month for its busy weeks is asking exactly that question. A
	// caller holding only a yes/no writes it as a count and loses nothing:
	//
	//	Marked: func(d time.Time) int { if hasEvent(d) { return 1 }; return 0 }
	//
	// It is called once per visible cell — 42 times per render, adjacent
	// months included — so it should be a lookup, not a query.
	//
	// The dots are decoration and hidden from assistive technology. The widget
	// knows how many things a day holds and nothing about what any of them is,
	// so there is nothing it could truthfully announce; a count worth speaking
	// goes into the spoken name through DayLabel, which is the seam that
	// knows.
	Marked func(time.Time) int

	// WeekStart is the weekday the grid's leftmost column is. The zero value
	// is time.Sunday, which is also the intended default.
	WeekStart time.Weekday

	// MonthLabel names the month in the header; nil gives "January 2006".
	// WeekdayLabel captions a column; nil gives the first two letters of the
	// English name ("Su", "Mo", …). DayLabel is the *spoken* name of a cell
	// for a screen reader; nil gives "Monday, January 2, 2006", to which the
	// widget appends ", today" when it applies. The selection is not part of
	// the name — it is announced as the control state it is; see dayLabel.
	MonthLabel   func(time.Time) string
	WeekdayLabel func(time.Weekday) string
	DayLabel     func(time.Time) string

	// Header replaces the default month row, arrows and all — for a screen
	// whose own chrome already carries the month, or one that navigates by
	// year. Navigation is then entirely yours: OnMonthChange is not called
	// from anywhere else.
	//
	// To drop the header without replacing it, pass core.Fragment().
	Header core.View

	// Style is applied to the outer column after its defaults.
	Style []core.StyleProp
}
```

Calendar is a controlled month grid: seven weekday captions over six rows of day cells, with a selected day, an optional "today" ring, optional dots for the days that have something on them, and arrows that ask the caller to change month.

	┌─────────────────────────────────────────┐
	│  ‹        March 2026              ›     │  <- Header (arrows only if OnMonthChange)
	│  Su  Mo  Tu  We  Th  Fr  Sa             │  <- weekday captions
	│  ·1   2   3   4   5   6 ··7             │  <- ·n = one mark, ··n = two
	│   8   9  10  11 [12] 13  14             │  <- [n] = Selected
	│  15  16  17 ·18  19  20  21             │
	│  22  23  24  25  26  27  28             │
	│  29  30  31   1   2   3   4             │  <- adjacent days, dimmed and inert
	│   5   6   7   8   9  10  11             │
	└─────────────────────────────────────────┘

	month := core.NewState(ctx, someDate)
	components.Calendar{
	    Month:         month.Get(),
	    OnMonthChange: month.Set,
	    Selected:      picked.Get(),
	    OnSelect:      picked.Set,
	    Today:         today,                 // see "The widget never asks what time it is"
	    Marked:        func(d time.Time) int { return len(eventsOn(d)) },
	}

#### Everything is the caller's, including which month is on screen

The widget holds no state and calls no hook, so it may be rendered conditionally — the same contract every widget here but Accordion keeps. That extends to the \*visible month\*, which looks like private view state and is not: an events screen wants to open on the month of the next event, a booking form wants to jump to the month a search result lands in, and a screen with two calendars side by side wants them to move together. A widget that owned its month could serve none of those.

The cost is one piece of state at the call site, as above. DatePicker is the packaging of exactly that state for the form case; reach for this when the grid is part of the screen rather than behind a field.

A nil OnMonthChange draws no arrows. That is the static case — a month with its events dotted, printed into a page — not a broken one.

#### The zero time goes both ways

Selected already spells "nothing is chosen" as the zero time. With Deselectable set, OnSelect reports that same zero when the reader taps the day that is already selected — so a grid used as a \*filter\* is cleared from the grid itself, the value making a round trip through the caller's state with no second callback to wire and nothing beside the calendar to build.

It is off by default, and the default is the interesting half. A picker asking which day the appointment is has no "no day" to offer, and there a stray second tap that quietly emptied the field would lose an answer the reader never asked to lose. So the widget does not guess which of the two it is in: a filter opts in, a picker leaves it alone, and DatePicker forces it off and puts the way out on its own Clear button.

There is deliberately no OnDeselect. Two callbacks setting the same piece of state is two things for every consumer to keep in step, and the day a screen wants to tell them apart it can test for the zero it was handed.

#### The widget never asks what time it is

There is no time.Now() in here, and Today is a field rather than something the widget works out. Three reasons, in ascending order of how much they bite:

  - A render that reads the clock is not a pure function of its inputs, so a snapshot test of "the March 2026 grid" would drift into a different picture every midnight.
  - "Today" is a question about a time zone, and the widget cannot know whether it should answer in the device's zone, the congregation's, or the one the data is stamped in. The caller can.
  - A zero Today draws no ring, which is the honest rendering of a calendar that has not been told what day it is.

#### The month is always six rows

A month spans four to six weeks depending on its length and the weekday it starts on, and a grid that grew and shrank with it would change height as the reader pages through the year — pushing whatever sits below the calendar up and down on every arrow tap. So the grid is always 6×7 = 42 cells, padded at both ends with the adjacent months' days.

The fixed shape pays a second time on the reconciler: changing month patches 42 text contents and their styles and touches no structure at all, where a variable grid would add and remove whole rows.

Those adjacent days are drawn dimmed and are \*inert\*. They are there so the grid reads as a grid — six full rows, no ragged hole — not as targets. A controlled calendar cannot move its own month, so a tap on the trailing "2" under March would either select a date the visible grid no longer highlights or fire two callbacks the caller has to sequence; the arrows are the one deliberate way the month changes.

#### Dates in, dates out: midday, not midnight

Every cell is built at \*\*12:00 in the calendar's location\*\*, and that is the value handed to Marked and OnSelect. It looks like it should be midnight and must not be.

Midnight does not exist on every calendar day. Chile springs forward at 24:00, so 2026-09-06 in America/Santiago begins at 01:00 and Go resolves time.Date(2026, 9, 6, 0, 0, 0, 0, santiago) to \*2026-09-05 23:00\* — the previous day. A grid built at midnight would therefore emit two cells that both read as September 5, and September 6 would be unselectable, in exactly the zones nobody testing in UTC ever looks at. Midday is skipped by no transition in the tz database.

So the value is an instant \*inside\* the intended day rather than at its edge. A caller who wants the day itself takes the triple that a date actually is:

	OnSelect: func(d time.Time) { y, m, day := d.Date(); … }

#### Which location

The calendar works in the location of its anchor — Month if set, else Selected, else Today — and converts Selected, Min, Max and Today into it before reducing them to a calendar day. An event stamped in UTC therefore lands on the day it happened \*locally\*, which is what a reader comparing the grid to their own week expects; a caller who means otherwise passes a Month already in the location they mean.

With all three of Month, Selected and Today zero there is no anchor and no clock to fall back on, and the widget renders nothing.

#### Localization

Go's time package formats in English only, so MonthLabel, WeekdayLabel and DayLabel are the seams for everything a reader sees as a word. The day \*numbers\* are numerals and are not routed through anything.

<small>[components/calendar.go:143](https://github.com/rohanthewiz/grmob/blob/master/components/calendar.go#L143)</small>

#### func (Calendar) Render

```go
func (c Calendar) Render(ctx *core.Context) *core.Node
```

<small>[components/calendar.go:274](https://github.com/rohanthewiz/grmob/blob/master/components/calendar.go#L274)</small>

### type Card

```go
type Card struct {
	Title  string
	Header core.View // overrides Title when set
	Body   core.View
	Footer core.View

	// HeadingLevel is where the Title sits in the screen's outline. Zero is
	// level 2 — a card is a section of a screen, the same tier as a
	// GroupedList band — and a card nested inside one of those should say 3.
	//
	// It applies to Title alone. A Header replaces the line entirely, and the
	// view in it is the caller's to describe: this widget cannot know whether
	// it was handed a heading, a row of controls, or an avatar.
	//
	// See headingLevel in heading.go for the package's outline and for how to
	// ask for a heading with no tier at all.
	HeadingLevel int

	// Style is applied to the card container after the theme's Card base.
	Style []core.StyleProp
}
```

Card is a surface with optional header, body, and footer regions, rendered on core.Card (so it inherits the theme's Card base: background, padding, radius, shadow).

Title is the simple path — a themed title line; Header is the escape hatch — an arbitrary view in the same position, taking precedence over Title when both are set. This mirrors element's Card, whose Body (string) and BodyComponent (component) coexist the same way.

<small>[components/card.go:13](https://github.com/rohanthewiz/grmob/blob/master/components/card.go#L13)</small>

#### func (Card) Render

```go
func (c Card) Render(ctx *core.Context) *core.Node
```

<small>[components/card.go:35](https://github.com/rohanthewiz/grmob/blob/master/components/card.go#L35)</small>

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

	components.Chip{Label: l, Selected: sel, OnTap: f,
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

The border is there for geometry, not for decoration. Only the unselected chip wants a visible rule, but a rule on one state alone makes that state 2px wider and taller wherever box-sizing is content-box (the static export sets no reset, so it is), and a pill that grows when you tap it is a worse artifact than the one this fixes. So both states carry a 1px border and the selected one paints it in its own fill, where it cannot be seen.

<small>[components/chip.go:132](https://github.com/rohanthewiz/grmob/blob/master/components/chip.go#L132)</small>

#### func (Chip) Render

```go
func (c Chip) Render(ctx *core.Context) *core.Node
```

<small>[components/chip.go:184](https://github.com/rohanthewiz/grmob/blob/master/components/chip.go#L184)</small>

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

	components.ChipStrip{Chips: []components.Chip{
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

	components.ChipStrip{Scrollable: true, Chips: years}

The two are exclusive by construction — a scrolling strip is one line, so there is nothing to wrap — and Scrollable wins when both are asked for.

<small>[components/chip_strip.go:55](https://github.com/rohanthewiz/grmob/blob/master/components/chip_strip.go#L55)</small>

#### func (ChipStrip) Render

```go
func (c ChipStrip) Render(ctx *core.Context) *core.Node
```

<small>[components/chip_strip.go:86](https://github.com/rohanthewiz/grmob/blob/master/components/chip_strip.go#L86)</small>

### type CodeEditor

```go
type CodeEditor struct {
	// Value is the buffer's text. The editor is controlled: it renders what it
	// is given and reports edits through OnChange.
	Value string

	// OnChange receives every edit. A nil OnChange makes the editor read-only
	// in practice — it will render Value and drop keystrokes — so set ReadOnly
	// instead when that is what you mean, which also tells the platform.
	OnChange func(string)

	// Language picks the lexer by name: "go", "json", or "" for no colour. An
	// unknown name is uncoloured rather than an error, so a screen whose editor
	// mis-spells its language still renders. It also picks the comment marker
	// the comment command toggles; see CommentPrefix.
	Language string

	// Highlighter overrides Language with a lexer of your own — anything
	// satisfying highlight.Highlighter, including one that caches.
	Highlighter highlight.Highlighter

	// Scheme is the palette. The zero value picks highlight.Darcula or
	// highlight.Light from the theme's own background, so an editor in a light
	// app is not a dark rectangle in the middle of the screen and one in a dark
	// app is not a white page.
	Scheme highlight.Scheme

	// LineNumbers turns on the gutter. Off by default: a three-line snippet
	// with line numbers reads as a listing rather than as code.
	LineNumbers bool

	// ReadOnly shows a caret and allows selection while refusing edits. This is
	// the display half of the widget — a code block in a document, a payload in
	// a log viewer — and it is deliberately not Disabled: the reader is meant
	// to select the code and copy it.
	ReadOnly bool

	// TabSize is how many spaces one indent is worth. Zero is core's default of
	// four; use TabSize(0) on the node directly for a literal tab.
	TabSize int

	// CommentPrefix overrides the line-comment marker the toolbar's comment
	// button toggles. Empty takes it from Language.
	CommentPrefix string

	// Height fixes the editor's height, so a long buffer scrolls inside it
	// rather than growing the screen. Empty lets it size to its content.
	Height string

	// Toolbar, when set, adds a row of editing commands above the editor and is
	// the ref they are sent to. Build it with core.UseEditorRef(ctx) in the
	// calling component — see the type's doc for why the ref is yours and not
	// this widget's.
	Toolbar *core.EditorRef

	// OnSelectionChange reports the caret as byte offsets into Value. Useful
	// for a status line ("line 12, column 4") and for a toolbar that has to
	// know whether anything is selected.
	OnSelectionChange func(start, end int)

	// Style is applied to the editor after the widget's own surface, so the
	// fill, the ink, the radius and the padding are all overridable.
	Style []core.StyleProp
}
```

CodeEditor is a programmer's editor: a monospace buffer with syntax colour, an optional line-number gutter, and an optional toolbar of the editing commands a code surface needs.

	components.CodeEditor{
	    Value:       src.Get(),
	    OnChange:    src.Set,
	    Language:    "go",
	    LineNumbers: true,
	    Height:      "240px",
	}

	┌ Column ──────────────────────────────────────────────────────┐
	│ ┌ Row (the toolbar, only when Toolbar names a ref) ────────┐ │
	│ │ [ ⇥ ]  [ ⇤ ]  [ // ]                                     │ │
	│ └──────────────────────────────────────────────────────────┘ │
	│ ┌ core.CodeEditor ─────────────────────────────────────────┐ │
	│ │  1  func main() {                                        │ │
	│ │  2      println("hi")                                    │ │
	│ │  3  }                                                    │ │
	│ └──────────────────────────────────────────────────────────┘ │
	└──────────────────────────────────────────────────────────────┘

It is the widget over core.CodeEditor and is what application code should reach for: it runs the highlighter, picks a colour scheme that suits the theme, and builds the toolbar. The node underneath is the primitive, and its doc carries the three rules every host implements.

#### It holds no state and calls no hook

Value is the caller's, exactly as SearchField's is, and every keystroke arrives through OnChange.

The hook question is worth spelling out, because the obvious design fails it. A toolbar needs a core.EditorRef, a ref must be stable across passes, and the way to get one is core.UseEditorRef — a hook. Calling it in here would make every CodeEditor a hook-slot consumer, and therefore something that must be rendered unconditionally on every pass, like Accordion and DatePicker. That is a fine obligation for a date field and a bad one for a \*code block\*: a read-only editor with no toolbar is the thing a document renders inside an \`if\`, inside a loop, inside a lesson body. So the ref is the caller's, and it is the Toolbar field itself — a toolbar is exactly "a ref plus some buttons", and naming the ref is how you ask for one:

	ref := core.UseEditorRef(ctx)
	components.CodeEditor{Value: v, OnChange: set, Toolbar: ref}

An editor with no toolbar touches no hook and can be rendered anywhere.

#### The highlighter runs every pass, deliberately

No memoization. go/scanner over a thousand lines is well under a millisecond — the tutorial has re-lexed every snippet on every render pass since it had snippets — and the alternative is hooks.UseMemo, which is the hook obligation this widget has just been designed out of. If a buffer ever grows past the point where that is true, the answer is for the \*caller\* to memoize and pass a Highlighter that caches, not for this widget to start consuming slots.

<small>[components/code_editor.go:65](https://github.com/rohanthewiz/grmob/blob/master/components/code_editor.go#L65)</small>

#### func (CodeEditor) Render

```go
func (c CodeEditor) Render(ctx *core.Context) *core.Node
```

<small>[components/code_editor.go:129](https://github.com/rohanthewiz/grmob/blob/master/components/code_editor.go#L129)</small>

### type Collapse

```go
type Collapse struct {
	// IsCollapsed reports whether a group's run is hidden. Nil means no.
	IsCollapsed func(Group) bool
	// OnToggle is called with the group whose band was pressed. Nil leaves
	// the bands as plain headings — see GroupHeader.Expanded.
	OnToggle func(Group)
}
```

Collapse is caller-owned collapse state for a banded collection: which groups are shut, and what to do when one is pressed.

#### Why the caller holds it

GroupedList calls no hook, which is what lets it be rendered conditionally — inside a core.IfElse against a pager's loaded flag, say — without disturbing the caller's hook cursor. Owning collapse state would end that, and the widget is the wrong place for it anyway: which months are shut is screen state, it usually wants to survive a pager reload, and a screen that wants "collapse all" has no way to reach inside a widget's NewState.

So the screen keeps a set and answers two questions:

	shut := core.NewState(ctx, map[string]bool{})
	GroupedList[Sermon]{
	    GroupBy: byMonth,
	    Collapse: components.Collapse{
	        IsCollapsed: func(g components.Group) bool { return shut.Get()[g.Key] },
	        OnToggle: func(g components.Group) {
	            next := maps.Clone(shut.Get())
	            next[g.Key] = !next[g.Key]
	            shut.Set(next)
	        },
	    },
	}

components.Accordion is the other answer to the same question and stays the right one for a single section: it owns its state, and the hook obligations that come with it are documented on the widget. A list of twenty bands is where owning the state stops being a convenience — twenty independent NewStates that a reorder cannot move, and no way to shut them all.

#### One type, two functions

They are two halves of one fact and are useless apart, which is the same argument core.ValueRange makes for its three numbers. As two fields on GroupedList a caller could supply either alone: IsCollapsed without OnToggle is a list with rows nobody can bring back, and OnToggle without IsCollapsed is a control that announces a state it does not have. The zero value is "nothing collapses", which is what every list that has never heard of this keeps doing.

<small>[components/grouping.go:122](https://github.com/rohanthewiz/grmob/blob/master/components/grouping.go#L122)</small>

### type CollapseBand

```go
type CollapseBand struct {
	// Collapse is the caller's own collapse state — the same value given to
	// GroupedList. The zero value builds a plain heading; see above.
	Collapse Collapse

	// Group is the run this control is about. Its Label names both the heading
	// and the button unless Content replaces the words.
	Group Group

	// HeadingLevel is where the band sits in the screen's outline. Zero is
	// level 2, the same default GroupHeader takes and for the same reason —
	// see GroupHeader.HeadingLevel, which is the field this mirrors.
	HeadingLevel int

	// Content goes inside the button, after the chevron. Empty takes the
	// group's Label in the band's own type, which is the common case and the
	// reason this is a slice rather than a required field.
	//
	// Whatever goes here is presentational: a reader does not descend into a
	// button's children, and the button's name comes from Group.Label. So an
	// icon needs no aria-hidden and a count put here stops being announced —
	// which is why the default band keeps its badge outside the control.
	Content []core.View

	// Style is applied to the heading wrapper, which is the node a caller's
	// layout sees. GroupedList's own band puts core.FlexGrow(1) here so the
	// count badge sits hard against the trailing edge; a caller arranging
	// their own row decides that for themselves.
	Style []core.StyleProp

	// ControlStyle is applied to the button, which is the node a finger lands
	// on. This is where a caller's own chrome belongs.
	//
	// # The question it answers
	//
	// A band's padding on the row that holds the control is dead space: the
	// button fills the box it was given and the insets are outside it, so the
	// 16px before the chevron is a place a press does nothing. GroupHeader
	// used to be built that way and is not any more (see bandInsets), and a
	// caller writing their own row inherits the same question one level out:
	//
	//	core.Row(                                    core.Row(
	//	    core.PaddingHorizontal(16),                  CollapseBand{
	//	    CollapseBand{...},              becomes          Collapse: shut, Group: g,
	//	    Badge{Text: count},                              ControlStyle: []core.StyleProp{
	//	)                                                        core.PaddingLeft(16),
	//	                                                         core.PaddingRight(8)},
	//	                                                 },
	//	                                                 Badge{Text: count},
	//	                                                 core.PaddingRight(16),
	//	                                             )
	//
	// Same pixels; a target that reaches the leading edge rather than starting
	// 16px into it.
	//
	// On an inactive Collapse there is no button and this lands on the plain
	// heading, for the same reason GroupHeader.ControlStyle does: the two
	// branches must be the same size, or a band changes shape on the day
	// somebody gives it a handler.
	ControlStyle []core.StyleProp
}
```

CollapseBand is the disclosure control the default band builds, on its own: a heading wrapping a button that carries aria-expanded and toggles one group's run. No Surface, no padding, no count badge — the chrome is the caller's.

#### The gap it closes

GroupedList.Collapse reaches past a Header override for the row emission, so an override's run still hides, and it stops at the override for the \*control\*, because a band the widget also built would be a second control for the same run. That division is right and it left the override author holding three things at once: a button, an aria-expanded that has to be stated on every pass open or shut, and a heading wrapper whose nesting order — heading around button, named explicitly so the chevron never reaches it — is four paragraphs of argument in components.disclosure, which is unexported.

GroupHeader is still the answer when the whole default band will do; it takes Expanded and OnToggle and builds all of it. This is the answer when it will not:

	Header: func(g components.Group) core.View {
	    return core.Row(
	        core.PaddingHorizontal(16),
	        components.CollapseBand{Collapse: shut, Group: g},
	        Avatar{Name: leader[g.Key]},
	        Badge{Text: strconv.Itoa(g.Count)},
	    )
	},

The Collapse passed here is the caller's own — the same value handed to GroupedList — which is what keeps the control and the row hiding answering to one state. An override that built its control from a second Collapse would have a chevron pointing one way and a run obeying the other.

#### An inactive Collapse builds a heading and no control

The zero Collapse — and one with IsCollapsed and no OnToggle — produces the label in a plain heading, exactly as GroupHeader's own non-disclosure branch does. Not a button with a dead handler: an expansion stated with nothing to toggle it is announced on both web targets, is silently nothing on Android, and is what core.AuditTree reports as ConcernInertDisclosure. Building one here would be building the thing the audit exists to find.

<small>[components/grouping.go:194](https://github.com/rohanthewiz/grmob/blob/master/components/grouping.go#L194)</small>

#### func (CollapseBand) Render

```go
func (b CollapseBand) Render(ctx *core.Context) *core.Node
```

<small>[components/grouping.go:256](https://github.com/rohanthewiz/grmob/blob/master/components/grouping.go#L256)</small>

### type Column

```go
type Column[T any] struct {
	Title string

	// Text is the simple path: the cell shows this string in body type.
	// Cell is the slot: any view, taking precedence when both are set — the
	// same simple-path-plus-slot idiom as Card.Title vs Card.Header.
	Text func(T) string
	Cell func(T) core.View

	// Weight is the column's FlexGrow share of the row's slack. 0 hugs the
	// cell's content, which is right for a short code or a glyph column;
	// give the column that carries the row's meaning a weight so it takes
	// what the fixed ones leave.
	Weight float64

	// Align positions the cell's content on the row axis; the zero value is
	// the leading edge. Numbers want JustifyEnd.
	Align core.JustifyContent

	// Narrow marks a column the table drops in Compact mode: a phone shows
	// title and date, a tablet adds teacher and place.
	Narrow bool

	// Less orders two rows for a client-side sort on this column; setting it
	// also makes the header tappable. Sortable makes the header tappable
	// without a comparator, for a table whose caller sorts server-side and
	// only needs to hear which column was asked for.
	//
	// Which of the two to set is not a matter of convenience: it depends on
	// whether DataTable.Rows is the whole set or a window of it. See the
	// "Sorting sorts the rows you hand it" section on DataTable.
	Less     func(a, b T) bool
	Sortable bool
}
```

Column describes one column of a DataTable: its header, how a row's value is drawn, how much of the row's width it takes, and whether it sorts.

<small>[components/data_table.go:12](https://github.com/rohanthewiz/grmob/blob/master/components/data_table.go#L12)</small>

### type Compass

```go
type Compass struct {
	// Heading is the bearing to draw, in degrees clockwise from north. Any
	// value works: it is normalised for the readout and the spoken label, and
	// applied as-is to the rotation (see core.Style.Rotate on winding).
	Heading float64

	// Size is the rose's diameter in px; 0 means 160.
	Size float64

	// ShowDegrees adds the numeric readout under the rose — "312° NW". Off by
	// default: a compass beside a map is a picture, and the number is noise
	// until something asks for it.
	ShowDegrees bool

	// Background is the rose's fill and Color its lettering and border; empty
	// uses the theme's Surface and TextSecondary. NorthColor inks the N and
	// the index mark, and defaults to the theme's Primary — north is the one
	// letter a reader looks for, and the accent is what makes it findable
	// while the rose is turning.
	Background string
	Color      string
	NorthColor string

	// Style is applied last, to the outer column, so a caller can space the
	// widget or give the whole thing a margin without reaching inside it.
	Style []core.StyleProp

	// AccessibilityLabel overrides the spoken bearing. See the type comment.
	AccessibilityLabel string
}
```

Compass draws a bearing as a compass rose: a round card lettered N/E/S/W that turns so its N points the way north actually is, under a fixed index mark at the top showing where the device is pointing.

	h := hooks.UseHeading(ctx)
	components.Compass{Heading: h.Magnetic, ShowDegrees: true}

	      ┌──▼───┐        index mark — fixed, over the rose's rim
	      │   N  │        the rose turns by -Heading, so N stays
	      │ W   E│        pointing at north while the device turns
	      │   S  │
	      └──────┘
	      312° NW         optional readout

#### Which half turns

Two conventions exist and they are not interchangeable. A magnetic compass has a fixed card and a needle that swings to north. A navigation compass — what every phone ships — turns the whole card and keeps a fixed index at twelve o'clock, so the bearing is read where the index crosses the rose. This is the second one, because the second one answers the question a phone user is asking ("which way am I facing", read off the top) rather than the question a hiker with a paper map is asking ("where is north").

The rose therefore rotates by \*minus\* the heading. Turning the device clockwise increases the heading, and the rose must turn counter-clockwise by the same amount to keep pointing at the same piece of the world.

#### Where the index mark sits, and where it used to

On the rose's rim, at twelve o'clock, drawn over it — which is where a compass index belongs and where this one could not go for as long as core had no z-axis container. Box stacks its children vertically on all four targets (settled deliberately — see the "Box overlay divergence" note in core.Box), and CSS absolute positioning is a declared web-only prop neither native renderer reads, so anything drawn \*over\* the rose would have been a web-only widget wearing a portable name. The mark was parked in the row above instead, costing a glyph of height and reading as a separate thing pointing at the dial rather than as part of it.

core.ZStack is that container, and this is its first consumer. The dial is a stack of two layers — the rose, then the mark, which asks for the top with core.StackAlign because a ZStack centres every layer that says nothing.

The mark used to be wrapped in a full-height column justifying its child to the start, which is the escape core.ZStack documented while it had no per-child alignment. This widget being its only consumer is what kept the prop out; StackAlign is the second half of that argument arriving.

The rose's inset went from half a letter to a whole one to make room. The mark's glyph is three quarters of a letter tall, so a ring that deep is what keeps N clear of it at heading zero — the one bearing where the two deliberately coincide, since an index pointing at N is exactly what facing north looks like.

#### A float, not a core.Heading

The field is a bearing in degrees rather than the sensor's own struct, so the widget also draws a bearing that has nothing to do with the compass — the direction of a route leg, a wind reading, the way a photograph was taken. hooks.UseHeading hands over the sensor's; anything else can hand over its own, and neither has to know about the other.

#### Accessibility

The rose is four letters whose positions carry the meaning, which is exactly the content a screen reader cannot convey: read aloud in tree order it is "N W E S", turning or not. So the whole widget announces once, as a spoken bearing ("Heading 312 degrees, northwest"), and every part inside it is hidden. AccessibilityLabel overrides the sentence for a caller whose bearing means something more specific than a heading.

The container takes core.RoleImg, and it is still the right value now that the two web exporters supply core.RoleGroup to any named container that has none (see core.RoleGroup). Both make the label legal — ARIA forbids an accessible name on a generic element, which is what a plain core layout node exports as, so the label alone was dropped by screen readers on both web targets while both natives read it out. What only \`img\` says is the part this widget depends on: a picture standing in for one fact, whose parts a reader should not read. \`group\` names its children and leaves them readable, which for a rose is "Heading 312 degrees, northwest" followed by "N W E S".

<small>[components/compass.go:90](https://github.com/rohanthewiz/grmob/blob/master/components/compass.go#L90)</small>

#### func (Compass) Render

```go
func (c Compass) Render(ctx *core.Context) *core.Node
```

<small>[components/compass.go:121](https://github.com/rohanthewiz/grmob/blob/master/components/compass.go#L121)</small>

### type DataTable

```go
type DataTable[T any] struct {
	Columns []Column[T]
	Rows    []T

	// Key returns the reconciler key for a row; nil falls back to positional
	// keys (see GroupedList for the trade-off).
	Key func(T) string

	// GroupBy, Header and HideTrailingCount work as in GroupedList; a group
	// header spans the full row.
	//
	// HideTrailingCount is for the same case there and here: a table fed by
	// an append-style pager, whose last group can still grow. It has no work
	// to do for a table that paginates through Pagination, where every page
	// shows a complete slice of rows the table already holds.
	GroupBy           func(T) Group
	Header            func(Group) core.View
	HideTrailingCount bool

	// StickyHeaders pins each group band while its run scrolls under it, as
	// in GroupedList.
	//
	// It pins the *group* bands, not the column header. The column header is
	// a sibling of the body list rather than a row inside it — that is what
	// keeps it from scrolling away in the first place — and both natives
	// implement pinning inside their lazy container only, so there is nothing
	// there for the marker to mean.
	StickyHeaders bool

	// HeadingLevel places the default group bands in the screen's outline, as
	// in GroupedList. Zero is level 2; a table nested inside a Card should say
	// 3. It reaches GroupHeader, so it does nothing without GroupBy and
	// nothing under a Header override.
	//
	// It says nothing about the *column* header, whose cells carry
	// core.RoleColumnHeader and no level — aria-level is defined for heading,
	// listitem and row, and pointedly not for columnheader.
	HeadingLevel int

	// Sort is the active sort, nil for none. OnSort receives the requested
	// sort when a sortable header is tapped: the same column toggles
	// direction, a different column starts ascending.
	//
	// Whether the table then does the sorting depends on the column: see
	// "Sorting sorts the rows you hand it" above before giving a column a
	// Less on a table whose Rows come a page at a time.
	Sort   *Sort
	OnSort func(Sort)

	// Pagination, when non-nil, renders the numbered footer and — when it
	// carries a PageSize and no PageCount — slices Rows to the current page.
	Pagination *Pagination

	// OnRowTap makes every body row a target.
	OnRowTap func(T)
	// Selected tints a row with the theme's Surface, ListRow's convention.
	Selected func(T) bool

	// Compact drops Narrow columns.
	Compact bool
	// Dividers inserts a hairline between consecutive body rows.
	Dividers bool

	// Empty is rendered in the body when there are no rows; Loading is
	// rendered in the body instead of the rows whenever it is non-nil, so a
	// refresh keeps its header and footer in place.
	Empty   core.View
	Loading core.View

	// Footer is an arbitrary view after the body, before the Pagination
	// footer: a LoadMore for a table that appends rather than pages.
	Footer core.View

	// Style is applied to the outer Column; HeaderStyle to the header row;
	// RowStyle to every body row, after each one's defaults.
	Style       []core.StyleProp
	HeaderStyle []core.StyleProp
	RowStyle    []core.StyleProp
}
```

DataTable is a header row over a virtualized, keyed body of cell rows, with controlled sorting, optional grouping, optional pagination and a compact mode for narrow screens.

	┌ Column ────────────────────────────────────────────┐
	│ ┌ Row (header) ────────────────────────────────┐   │
	│ │ Title ▲          Teacher         Date        │   │  <- tap sorts
	│ └──────────────────────────────────────────────┘   │
	│ ┌ List (body) ─────────────────────────────────┐   │
	│ │ ▒ January 2026                          (2)  │   │  <- GroupHeader
	│ │ Grace and Truth   Pastor Ray     Jan 12      │   │  <- key Key(row)
	│ │ ─────────────────────────────────────────    │   │  <- Dividers
	│ │ The Vine          Guest          Jan 5       │   │
	│ └──────────────────────────────────────────────┘   │
	│ ‹ Prev            Page 1 of 4            Next ›    │  <- Pagination
	└────────────────────────────────────────────────────┘

#### Order of operations

Rows are sorted, then paged, then grouped: the sort decides the order, paging takes a window of it, and grouping reads runs off that window. Grouping last is what makes a client-side page's headers agree with its rows, and sorting first is what makes group runs follow the sort (a table sorted by teacher and grouped by month shows a month once per teacher run — the honest rendering; see groupRuns).

#### Sorting sorts the rows you hand it

A column with Less is sorted by the table, over Rows — all of Rows, and only Rows. That is right whenever Rows is the whole set: a slice the caller holds, or a fetch that completed, including the client-paged case where Pagination slices rows the table already has.

It is wrong when Rows is a window somebody else chose. An offset pager's "the three pages loaded so far", or a server-paged table declaring PageCount, is a subset selected under some other ordering. Sorting that yields the alphabetically-first of the rows that happen to be loaded, and puts it under a header claiming to be the alphabetically-first rows. The result carries no sign of which it is, which is exactly what makes the mistake worth naming: a partial sort looks like a working one, and the rows that disprove it are the ones not fetched.

For that table set Sortable instead of Less. The header still taps and OnSort still fires; the sort goes into the next query, and the server returns a window of the ordering the reader actually asked for.

One shape of this is detectable and is reported as a debug concern: Sort active on a Less column while Pagination declares a PageCount, since a declared PageCount is the caller saying the server does the paging. See ConcernPartialSort. The append-style case has no such signal — a slice of accumulated pages is indistinguishable from a complete one — so that half is left to this documentation.

#### Who owns what

Everything is controlled. The table holds no state and calls no hook: Sort is read from the caller and OnSort reports the tap, Pagination.Page is read and OnChange reports the step. Only the \*work\* is optionally the table's: a column with Less is sorted here, a Pagination with PageSize and no PageCount is sliced here. Both are pure functions of the inputs.

#### Compact

A phone is too narrow for a five-column table. Compact drops every Narrow column and leaves the rest; the header and cells stay in step because both walk the same filtered column list. The caller decides when the table is compact — from a breakpoint, a settings toggle, an orientation event.

<small>[components/data_table.go:133](https://github.com/rohanthewiz/grmob/blob/master/components/data_table.go#L133)</small>

#### func (DataTable) Render

```go
func (d DataTable[T]) Render(ctx *core.Context) *core.Node
```

<small>[components/data_table.go:213](https://github.com/rohanthewiz/grmob/blob/master/components/data_table.go#L213)</small>

### type DatePicker

```go
type DatePicker struct {
	// Selected is the chosen day; zero shows Placeholder instead.
	Selected time.Time

	// OnSelect fires with the tapped day and the sheet closes. The value is
	// midday in the calendar's location — see Calendar's "Dates in, dates
	// out". Nil renders a summary that opens a sheet nothing can be picked
	// from, which is Calendar's inert display behind a field; use Disabled
	// for a field that should not open at all.
	OnSelect func(time.Time)

	// OnClear puts a "Clear" button in the sheet that empties the field and
	// closes it. Nil renders no such button: whether a date is optional is
	// the form's question, not the picker's, and a widget that always offered
	// to clear a required field would be offering an invalid state.
	OnClear func()

	// Placeholder is the summary's text when nothing is selected. Empty
	// leaves the trigger blank but still tappable.
	Placeholder string

	// Format is the time layout the summary is written in; empty gives
	// "Jan 2, 2006".
	//
	// Go's time package names months and weekdays in English only, so a
	// layout carrying a name is a layout in English. Where that matters, use
	// a numeric layout — "2006-01-02" reads the same in every language and
	// has none of the day/month ambiguity of "02/01/2006".
	Format string

	// Calendar is the template the sheet's grid is rendered from, exactly as
	// SegmentedControl.Segment is the template for its chips. Today, Min,
	// Max, Marked, WeekStart, the three label functions, Header and Style all
	// apply; Month, OnMonthChange, Selected and OnSelect are overwritten,
	// since those four are what the picker is for, and Deselectable is forced
	// off — clearing a field is OnClear's job here, for the reason given
	// where the sheet sets it.
	Calendar Calendar

	// Title names the sheet. Empty leaves the heading row to the buttons
	// alone, which is right when the FormField label above the trigger has
	// already said what is being picked.
	Title string

	// ClearLabel and CloseLabel caption the sheet's two ways out; empty gives
	// "Clear" and a ✕ glyph.
	ClearLabel string
	CloseLabel string

	// Disabled marks the trigger inert: it neither opens nor announces itself
	// as actionable. The sheet's own contents are disabled with it, so a tap
	// racing the patch cannot land in an open picker.
	Disabled bool

	// AccessibilityLabel names the trigger; empty announces the summary text,
	// which is the date or the placeholder. AccessibilityHint describes what
	// tapping does.
	AccessibilityLabel string
	AccessibilityHint  string

	// Style is applied to the trigger row after its defaults. The sheet is
	// styled through Calendar.Style and the theme.
	Style []core.StyleProp
}
```

DatePicker is a date field: a tappable summary of the chosen day that opens a Calendar in a modal sheet and closes again on the tap that picks.

	┌──────────────────────────┐        ┌───────────────────────────┐
	│ Jan 2, 2026          📅  │  tap → │ Event date      Clear  ✕  │
	└──────────────────────────┘        │  ‹    January 2026     ›  │
	                                    │  Su Mo Tu We Th Fr Sa     │
	                                    │   …      [2]     …        │
	                                    └───────────────────────────┘

	components.FormField{
	    Label: "Event date",
	    Input: components.DatePicker{
	        Selected: date.Get(),
	        OnSelect: date.Set,
	        Calendar: components.Calendar{Today: today, Min: today},
	    },
	}

#### It is the input, not the field

There is no Label, Hint, Error or Required here, because FormField already owns all four and any input can sit in its slot. A picker that grew its own label would be a second way to write a form, worded and spaced slightly differently from every other field on the screen.

#### Two states, both of them the widget's

This is the second widget in the package that owns state (Accordion is the other), and it owns exactly the two pieces no application ever wants: is the sheet open, and which month is being browsed inside it. Owning them means inheriting the hook rules — render a DatePicker unconditionally, in a stable position, every pass, the same obligation calling core.NewState directly carries. Calendar itself takes no hooks, so the grid on a screen is still free to be conditional; it is this packaging that is not.

The browsed month is held as a \*zero\* time.Time until an arrow is tapped, rather than being seeded from the selection when the picker mounts. Seeding would go stale the moment the caller set a date from somewhere else — a "next Sunday" shortcut, a form loading a saved draft — and the picker would open on the month the screen first rendered in. A zero Month is exactly what Calendar's own anchor fallback reads as "follow Selected, then Today", so the two states cost one line between them and no re-derivation. Opening the sheet resets it, so the picker always opens on the month it is showing.

#### Picking closes it

A single date has nothing to confirm: the tap that chooses is the tap that finishes, so there is no Done button standing between the two. What the sheet does carry is the ways \*out\* — the backdrop, the ✕, and Clear when the field is clearable — because a reader who opened it to look at March needs to leave without having changed anything.

<small>[components/date_picker.go:61](https://github.com/rohanthewiz/grmob/blob/master/components/date_picker.go#L61)</small>

#### func (DatePicker) Render

```go
func (p DatePicker) Render(ctx *core.Context) *core.Node
```

<small>[components/date_picker.go:139](https://github.com/rohanthewiz/grmob/blob/master/components/date_picker.go#L139)</small>

### type Emphasis

```go
type Emphasis string
```

Emphasis is how strongly a button asserts itself: how much of the variant's color it spends. It is the second of Button's two color axes.

#### Why two axes and not one enum

The obvious API is a single enum — Primary | Secondary | Danger | Ghost — and it is what the component gap analysis first sketched. It was dropped because it conflates two independent questions: \*which\* color (the meaning) and \*how much\* of it (the visual weight). A flat enum cannot express an outlined destructive button, the ordinary shape of a "Delete" confirmation, without a fifth value, and then a ghost destructive needs a sixth.

Splitting them also lets Button reuse Variant verbatim, so a danger Button and an error Badge are the same red by construction rather than by two palettes agreeing. See Variant, which Badge already uses.

<small>[components/button.go:31](https://github.com/rohanthewiz/grmob/blob/master/components/button.go#L31)</small>

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

### type EmptyState

```go
type EmptyState struct {
	// Glyph is the large mark above the text. Empty draws none, which is the
	// right call for the busy case where a mark would look like a state the
	// user is meant to read.
	//
	// It is decoration and is hidden from assistive technology: a reader
	// announcing "open mailbox with lowered flag" ahead of "No messages yet"
	// is noise. Title has to carry the meaning.
	Glyph string

	// Title is the primary line — what is going on. Hint is the quieter line
	// under it — what to do about it.
	Title string
	Hint  string

	// ActionLabel and OnAction render the way out: "Retry", "Clear filters",
	// "Invite someone". Action is the slot form and takes precedence, for a
	// pair of buttons or anything else.
	//
	// The built button is outlined, not filled. An empty state is a dead end,
	// not a call to action — a solid Primary button in the middle of an empty
	// screen is the loudest thing on it, and the screen has nothing to say.
	ActionLabel string
	OnAction    func()
	Action      core.View

	// Style is applied after the widget's own defaults, so the padding, the
	// centering and the full width are all overridable.
	Style []core.StyleProp
}
```

EmptyState is the centered placeholder that stands in for content a screen does not have: a list with nothing in it, a fetch still in flight, a fetch that failed.

	components.EmptyState{Glyph: "🔍", Title: "No sermons match “grace”",
	    Hint: "Try a different word, or clear the filters."}

#### One widget for empty, busy and failed

Those three look like three states but they are one shape — a mark, a line saying what is going on, a quieter line saying what to do about it, and sometimes a way out — and screens that build them separately end up wording and spacing them differently:

	empty   EmptyState{Glyph: "📭", Title: "No messages yet"}
	busy    EmptyState{Title: "Loading sermons…"}
	failed  EmptyState{Glyph: "☁", Title: err.Error(),
	            ActionLabel: "Retry", OnAction: reload}

The busy case is a line of text rather than a spinner on purpose: core has no indeterminate progress node, and ProgressBar is determinate (it takes a 0..1 value), so there is nothing here to animate a wait with. Naming what is loading is more useful than an animation in any case — "Loading sermons…" tells the user which of the screen's three sections is slow.

#### The busy line moves when the wait is not the whole screen

The three-state example above puts the busy words in Title, which is right for a wait that owns the screen and wrong for the other place a wait renders: under the last row of a paged list, as the footer that says the next page is coming. There, body-sized primary ink reads as one more row — the reader tries to parse "Loading sermons…" as content — so the tail case puts the words in Hint and leaves Title empty:

	screen  EmptyState{Title: "Loading sermons…"}
	tail    EmptyState{Hint: "Loading more…"}

The rule the split comes out of is \*errors and empties speak in the primary line; a wait speaks there only when it is the whole screen\*. The widget cannot apply it itself, because it is handed a slot and never learns whether that slot is a screen's middle or a list's end — so this is the caller's call, written down here so the next screen does not re-derive it and land somewhere else.

A tail usually wants less air than a screen-sized placeholder too. Padding is a default like any other and Style is applied after it:

	EmptyState{Hint: "Loading more…", Style: []core.StyleProp{core.Padding(t.Spacing.SM)}}

#### Width is load-bearing

The column sets Width 100%, which looks redundant and is not. On both natives a column hugs its widest child (Compose wrap-content; grmob's SwiftUI layout does the same), so without it the whole block sits at the leading edge with its children centered inside a box only as wide as the longest line — centered text that is not centered on the screen. The two DOM targets fill the line already, as any block box does, so the bug is invisible on the target you are most likely to be looking at.

<small>[components/empty_state.go:63](https://github.com/rohanthewiz/grmob/blob/master/components/empty_state.go#L63)</small>

#### func (EmptyState) Render

```go
func (e EmptyState) Render(ctx *core.Context) *core.Node
```

<small>[components/empty_state.go:94](https://github.com/rohanthewiz/grmob/blob/master/components/empty_state.go#L94)</small>

### type FormField

```go
type FormField struct {
	Label string

	// Required draws the conventional marker after the label. It is
	// annotation only: this widget validates nothing, and setting it does not
	// make an empty field fail — forms.Required is what does that.
	//
	// Which is why it is worth feeding it from forms.Form.Required rather
	// than from a literal true. A hand-set flag is a second, independent
	// claim about the same field, and the two drift the first time a rule is
	// added or dropped: a marked field that submits empty, or an unmarked one
	// the user cannot get past. The form derives its answer from the rules
	// themselves, so the mark cannot outlive the rule that justified it.
	//
	// Ignored when Label is empty — there is nothing to mark. A control whose
	// label lives elsewhere (the checkbox row in examples/signup, whose title
	// belongs to the ListRow) has to carry its own.
	Required bool

	// Hint is the quiet guidance line under the input. Error replaces it
	// when non-empty — a field shows one line of feedback, and an error
	// outranks guidance.
	Hint  string
	Error string
	Input core.View
	// Style is applied to the wrapping column after the field's own layout.
	Style []core.StyleProp
}
```

FormField wraps any input in the label / input / hint-or-error frame — the pattern element's form\_field made its most-used component, transplanted: the wrapper owns the text furniture so every form in an app annotates its inputs the same way, and the Input slot keeps it agnostic to what is being wrapped (Input, TextArea, NumericInput, a custom picker...).

The widget renders feedback; it does not produce any. Error is a string the caller supplies, and the thing that supplies it — including the decision about \*when\* a message should be visible at all — is package forms:

	components.FormField{
	    Label: "Email",
	    Hint:  "We never share it",
	    Error: form.Error("email"),
	    Input: form.Input("email", "you@example.com"),
	}

That split is why the dependency runs one way and only in the caller: forms produces the strings and the bound controls, this widget frames them, and neither package imports the other. Required follows the same shape — the form knows which fields reject an empty value, the widget only draws the mark:

	Required: form.Required("email"),

<small>[components/form_field.go:29](https://github.com/rohanthewiz/grmob/blob/master/components/form_field.go#L29)</small>

#### func (FormField) Render

```go
func (f FormField) Render(ctx *core.Context) *core.Node
```

<small>[components/form_field.go:64](https://github.com/rohanthewiz/grmob/blob/master/components/form_field.go#L64)</small>

### type Group

```go
type Group struct {
	Key   string
	Label string
	Count int

	// Trailing reports whether this is the last run in the collection — the
	// one an append-style pager extends, and the only one whose Count is
	// still open (see GroupedList.HideTrailingCount for why that matters).
	//
	// Filled in by the widget, like Count: a value a GroupBy callback sets is
	// overwritten. The two grouping walks agree about it — groupRuns marks its
	// last run and trailingRun marks the run it went looking for — so a band
	// and the auto-load decision are answering the same question.
	//
	// It exists for the Header override. HideTrailingCount is documented as a
	// decision an override "owns itself", and until this field an override
	// could not make it: the rule is *don't publish an open run's count*, and
	// nothing handed to the override said which run was open. Deriving it
	// meant re-walking Items with the same GroupBy the widget had just walked.
	//
	//	Header: func(g components.Group) core.View {
	//	    label := g.Label
	//	    if !g.Trailing || !pager.HasMore {
	//	        label += " (" + strconv.Itoa(g.Count) + ")"
	//	    }
	//	    return ...
	//	}
	Trailing bool

	// AutoLoadWithheld reports that this run being shut is why the list is
	// rendering without its edge sensor — the state
	// GroupedList.OnEndReached's "a shut trailing group withholds it" section
	// describes, attached to the run that caused it.
	//
	// True on at most one group of a list, and always a Trailing one. False
	// throughout a list with no OnEndReached, since there is no sensor to
	// withhold; false on every band of a DataTable, which has no edge sensor
	// at all.
	//
	// # Why the widget states it rather than the band deriving it
	//
	// GroupedList.AutoLoadWithheld answers the *caller*, who can then build a
	// footer. A Header override is a different reader in a different place: it
	// is handed a Group and nothing else, so a band that wanted to say
	// "collapsed — auto-load is off here" had to close over Items, GroupBy and
	// Collapse and re-derive an answer the widget had just computed.
	//
	//	Header: func(g components.Group) core.View {
	//	    band := core.Row(components.CollapseBand{Collapse: shut, Group: g})
	//	    if g.AutoLoadWithheld {
	//	        band = core.Row(band, components.Badge{Text: "paused"})
	//	    }
	//	    return band
	//	},
	//
	// Deriving it in the override is also easy to get subtly wrong, which is
	// the stronger half of the argument. The composite is Trailing *and* the
	// run is hidden *and* a sensor was given — and "the run is hidden" is
	// Collapse.hides, which needs OnToggle as well as IsCollapsed: a caller
	// with a predicate and no handler hides nothing, so their bands would
	// announce a pause the list is not taking. One statement, from the value
	// that made the decision.
	//
	// The default GroupHeader ignores it. What a band says about a paused feed
	// is a wording decision, and the default band's vocabulary is a label and a
	// count; this is the fact an override needs to make that decision, not a
	// decision the widget makes for it.
	AutoLoadWithheld bool
}
```

Group identifies one run of rows in a grouped collection. GroupBy callbacks return Key and Label; the grouping engine fills Count. Key is what decides group identity and becomes the header's reconciler key, so it should be stable and comparable ("2026-01"), while Label is what people read ("January 2026").

<small>[components/grouping.go:10](https://github.com/rohanthewiz/grmob/blob/master/components/grouping.go#L10)</small>

### type GroupHeader

```go
type GroupHeader struct {
	Group Group

	// HideCount drops the trailing badge; the label stands alone.
	HideCount bool

	// HeadingLevel is where the band sits in the screen's outline. Zero is
	// level 2 — a band is a section of the screen whose name an AppBar's
	// title carries at level 1 — and a banded list nested inside a Card (also
	// level 2) should say 3.
	//
	// The field is also how a screen with no bar stops lying. A bandless feed
	// used to start its outline at 2 with no 1 above it, which is a soft lint
	// on the web and nothing at all to either native; the alternative was
	// announcing a screen's name and its March band as peers, so 2 stayed as
	// the lesser of two wrongs. Now the caller in that position writes 1.
	//
	// See headingLevel in heading.go for the package's outline and for how to
	// ask for a heading with no tier at all.
	HeadingLevel int

	// Expanded and OnToggle turn the band into a disclosure: the label becomes
	// a button carrying aria-expanded, with a chevron ahead of it, and the run
	// beneath is the caller's to hide.
	//
	// # Both or neither
	//
	// OnToggle nil is the ordinary band, and Expanded is then ignored. That is
	// not a silent drop of a stated fact — it is what core.ExpandedUnset
	// means, and a caller who states one without the other is reported by
	// core.SetDebugMode as ConcernInertDisclosure, because an expansion with
	// no handler is announced on both web targets and is silently nothing on
	// Android. See components.disclosure, which is where the pairing and the
	// ARIA shape are argued.
	//
	// # The badge stays outside the button
	//
	// A button's children are presentational — a reader does not descend into
	// them — so a count inside the control would stop being announced, and the
	// count is real content rather than chrome. It therefore sits beside the
	// button rather than within it, which is also what keeps the heading named
	// "January 2026" instead of "January 2026 3".
	//
	// The cost is that the badge is not part of the tap target: the button
	// runs from the band's leading edge and stops where the count begins.
	// That is the ordinary shape of a header row with a trailing badge, and
	// the alternative — folding the count into the button's accessible name —
	// would be assembling an English phrase in the renderer, which is the move
	// Chip's ", selected" suffix was deleted for.
	//
	// The band's own insets used to be excluded too, which was not the same
	// kind of cost: the count is content a press should not toggle, and the
	// padding is chrome. They are on the control now — see bandInsets — so
	// what the target excludes is exactly the thing that is not the control.
	Expanded bool
	OnToggle func()

	// Style is applied to the band Row after its defaults: its fill, its
	// margins, core.StickyHeader, a width.
	//
	// Not its insets. The band's padding lives on the control now — see
	// bandInsets for why and for the picture — so a caller who wants the
	// label flush left writes it in ControlStyle. Putting a
	// core.PaddingLeft(0) here still reaches the Row, where it is a no-op on
	// every side but the badge's own trailing inset.
	Style []core.StyleProp

	// ControlStyle is applied to the band's control after the insets: the
	// node a finger lands on, and therefore where the band's own padding is.
	//
	// A caller indenting a nested band's label, or shipping a denser feed,
	// reaches for this rather than Style — and gets a tap target that moves
	// with the chrome instead of a strip in the middle of it.
	//
	// On a band with no OnToggle there is no control, and this lands on the
	// plain heading that stands in for one. That is deliberate: the two
	// branches are the same band geometrically, and a knob that silently did
	// nothing on one of them would be a relayout the first time a caller
	// added a handler.
	//
	// "The same band geometrically" is a claim about the chrome, and it is
	// exact: browser.mjs measures both branches in every bundled theme and the
	// control's leading and trailing edges are identical. The band's HEIGHT is
	// not, by one point — the disclosure's button holds a chevron the plain
	// band does not, a control is as tall as its tallest child plus its own
	// insets, and that glyph's line box exceeds the caption's in the font stacks
	// Chrome resolves. That is content, not chrome, and it is checked as the
	// equation it is rather than waved at: the difference between the two bands
	// must equal the chevron's overhang over the words exactly, so chrome
	// drifting between the branches still fails.
	ControlStyle []core.StyleProp
}
```

GroupHeader is the default band rendered above each group in GroupedList and DataTable: the label in bold caption ink on the theme's Surface, with the row count as a badge pinned to the trailing edge. It is exported so a caller can render it with a different Count or Label from inside a Header override, or reuse it in a hand-built list.

<small>[components/grouping.go:421](https://github.com/rohanthewiz/grmob/blob/master/components/grouping.go#L421)</small>

#### func (GroupHeader) Render

```go
func (h GroupHeader) Render(ctx *core.Context) *core.Node
```

<small>[components/grouping.go:595](https://github.com/rohanthewiz/grmob/blob/master/components/grouping.go#L595)</small>

### type GroupedList

```go
type GroupedList[T any] struct {
	Items []T

	// Key returns the reconciler key for an item; see the type comment.
	Key func(T) string
	// Row draws one item. Required.
	Row func(T) core.View

	// GroupBy assigns each item to a Group by Key and Label; Count is filled
	// in. Nil renders a flat list.
	GroupBy func(T) Group
	// Header overrides the default GroupHeader for each group.
	Header func(Group) core.View

	// HideTrailingCount drops the count badge from the *last* group's
	// header. Set it from a pager's "there is more" flag.
	//
	// A group's Count counts the rows the list was handed, which under an
	// append-style pager means "the rows loaded so far". Every group but the
	// last is closed — the next group's first row ended it — so its count is
	// final. The last one is still open: the next page can extend it, and a
	// header that says "June 2026 (1)" above a run about to become four is
	// not a stale number, it is a wrong one, and it changes under the reader
	// on a tap they did not think was a question about June.
	//
	//	HideTrailingCount: pager.HasMore
	//
	// So the trailing header shows its label alone until the feed is
	// exhausted, at which point the flag goes false and the count appears.
	// The header keeps its key across that, so the reconciler patches the
	// badge in rather than replacing the band.
	//
	// This is only about the default GroupHeader. A Header override is handed
	// the Group unchanged — its Count included — and owns the decision
	// itself; there is no way for the widget to reach inside a view the
	// caller built.
	HideTrailingCount bool

	// StickyHeaders pins each group's band to the top of the viewport while
	// its run scrolls underneath, releasing it when the next band arrives.
	// The reader always knows which month they are looking at, which is the
	// whole reason a feed is banded in the first place.
	//
	//	GroupedList[Sermon]{GroupBy: byMonth, StickyHeaders: true, ...}
	//
	// It is core.StickyHeader on the default GroupHeader, so it does nothing
	// without GroupBy — there are no bands in a flat list to pin.
	//
	// A Header override is handed the Group and builds its own view, which
	// this cannot reach into; such a header pins itself by putting
	// core.StickyHeader() in its own Style. That is the same division
	// HideTrailingCount draws, and for the same reason: a view the caller
	// built is the caller's.
	StickyHeaders bool

	// HeadingLevel places the default bands in the screen's outline. Zero is
	// level 2 — a band is a section of the screen an AppBar's title names at
	// level 1 — and a banded list inside a Card should say 3, while a feed on
	// a screen with no bar at all should say 1.
	//
	// It reaches GroupHeader, so it does nothing without GroupBy and nothing
	// under a Header override, on the same division StickyHeaders draws: a
	// view the caller built is the caller's to place in the outline. See
	// GroupHeader.HeadingLevel.
	HeadingLevel int

	// Dividers inserts a theme hairline between consecutive rows of a group
	// (not after the last row, where the next header or the footer follows).
	Dividers bool

	// Collapse turns the bands into disclosures whose runs the reader can
	// shut, with the state held by the caller. The zero value is the list as
	// it has always been.
	//
	//	Collapse: components.Collapse{
	//	    IsCollapsed: func(g components.Group) bool { return shut.Get()[g.Key] },
	//	    OnToggle:    func(g components.Group) { ... },
	//	}
	//
	// It does nothing without GroupBy — there are no bands in a flat list to
	// shut — on the same division StickyHeaders draws. Unlike StickyHeaders
	// it is *not* ignored under a Header override: the override owns the
	// band, and this owns whether the rows under it are emitted, which is not
	// something a view the caller built can reach. Such a caller renders their
	// own control and calls the same OnToggle.
	//
	// See Collapse for why the state is the caller's and why the two
	// functions are one type, and GroupHeader.Expanded for the band's ARIA
	// shape.
	//
	// It interacts with OnEndReached, which is the one place two of this
	// widget's features are in tension: a shut *trailing* group withholds the
	// edge sensor, because a page fetched into a hidden run appears nowhere
	// and silences the guard behind it. See OnEndReached.
	Collapse Collapse

	// Empty is rendered in place of the rows when Items is empty. Nil renders
	// an empty list.
	Empty core.View
	// Footer is appended after the rows: a LoadMore, a Pagination, a
	// summary. It is rendered whether or not Items is empty, so a pager's
	// error state is still reachable when the first page failed.
	Footer core.View

	// OnEndReached turns the feed into an infinite one: it fires when the
	// reader scrolls within a few rows of the bottom, so the next page is
	// fetched without a tap.
	//
	//	GroupedList[Sermon]{
	//	    Items:        pager.Items,
	//	    OnEndReached: pager.LoadMore,
	//	    Footer:       LoadMore{HasMore: pager.HasMore, Loading: pager.Loading, Err: pager.Err, OnLoadMore: pager.LoadMore},
	//	}
	//
	// # Keep the Footer
	//
	// Auto-loading replaces the *tap*, not the tail. The footer is still
	// where "Loading…" and a failed page's Retry live, and it is still the
	// manual fallback on a target with no way to report the edge (a static
	// export, a browser without IntersectionObserver). A screen that drops
	// its LoadMore for this gains a feed that silently stops at whatever
	// page failed.
	//
	// Passing the pager's load function to both is correct and is the
	// intended shape: core.OnEndReached will not re-ask until the row count
	// changes, so a button tap and a scroll cannot double-load, and a page
	// that came back empty leaves the edge quiet until something else moves.
	//
	// Nil leaves the list exactly as it was — a manual pager, driven by its
	// footer.
	//
	// # A shut trailing group withholds it
	//
	// The two features are in tension and neither one is wrong. An append
	// pager can only ever extend the *last* run (see the type comment), and a
	// shut run emits no rows at all — so when the reader has collapsed the
	// group at the bottom of the feed, a page that arrives is a page that
	// appears nowhere.
	//
	// Left alone, that is worse than it sounds. core.OnEndReached will not
	// re-ask until the List's child count changes, and a page absorbed into a
	// hidden run changes nothing, so the *first* auto-load spends a page and
	// every one after it is refused. The feed reads as exhausted, the pager's
	// offset has moved, and nothing anywhere says so.
	//
	//	shut trailing group, auto-load left on
	//	  scroll to bottom -> fetch page 3 -> 20 rows into a hidden run
	//	  -> child count unchanged -> the guard closes -> the feed is over
	//
	// So the prop is withheld while that group is shut, and comes back the
	// moment it is opened. Suppressing rather than starving is the honest
	// half: the reader has said they do not want to see this run, and
	// fetching more of it in the background is work nobody asked for and
	// nobody can look at.
	//
	// What stays reachable is the Footer, which is why "keep the Footer"
	// above is not only about static targets. components.LoadMore is a button
	// that calls the same function directly, so a reader who wants the next
	// page while the last group is shut has one — and pressing it is a
	// deliberate act, where a scroll is not.
	//
	// This is a fact about the *last* run only. A shut group anywhere above
	// it hides its rows and changes nothing about the edge, because the pager
	// was never going to extend it.
	//
	// AutoLoadWithheld is how a caller finds out it has happened. Without it
	// the withholding is invisible from outside — a feed that has stopped
	// fetching and a feed that has run out look identical — and a screen that
	// hides its footer on the strength of auto-loading has no way out at all.
	OnEndReached func()

	// Style is applied to the List after its defaults. The defaults shed the
	// theme Column's padding and gap: rows and headers are flush and spacing
	// is theirs to add, so a hairline divider is really one pixel tall.
	Style []core.StyleProp
}
```

GroupedList is a virtualized, keyed list of typed items with optional run-length group headers and a footer slot for a pager: the shape of an archive feed — sermons by month, transactions by day, messages by sender.

#### What it settles

Every paged screen in an app builds the same core.List by hand: a keyed row per item, an empty note when there are none, and a "Load more" tail. The widget owns that assembly so the screen owns only the three things that differ — how to key an item, how to draw one, and where its pages come from.

	┌ List ──────────────────────────────────┐
	│ ▒ January 2026                     (3) │  <- GroupHeader, key "group:2026-01"
	│   Row(item)                            │  <- key Key(item)
	│   Row(item)                            │
	│   Row(item)                            │
	│ ▒ December 2025                    (1) │
	│   Row(item)                            │
	│           [ Load more ]                │  <- Footer
	└────────────────────────────────────────┘

#### Grouping is by run, not by bucket

GroupBy is evaluated in Items order and a header is emitted wherever the key changes (see groupRuns). Items must therefore arrive already ordered by the grouping — which a feed sorted by date, grouped by month, always is — and an offset pager that appends pages can only ever grow the last group, so nothing above the fold moves on "Load more".

#### Identity

core.List keeps row state attached to keys across insertions and reorders, so Key must be unique across the whole list and stable across renders. A nil Key falls back to positional keys, which is correct for a static list and loses row state on reorder for a live one, exactly as core.List documents. Group headers take "group:"+Group.Key, so a row key can never collide with a header even when a caller keys rows by the same string.

#### No hooks

The widget holds no state and calls no hook, so it may be rendered conditionally — inside core.IfElse against a pager's loaded flag, say — without disturbing the caller's hook cursor.

<small>[components/grouped_list.go:49](https://github.com/rohanthewiz/grmob/blob/master/components/grouped_list.go#L49)</small>

#### func (GroupedList) AutoLoadWithheld

```go
func (g GroupedList[T]) AutoLoadWithheld() bool
```

AutoLoadWithheld reports whether this list is about to render \*without\* the edge sensor it was given — the state OnEndReached's "a shut trailing group withholds it" section describes.

##### Why a caller needs to be able to ask

Withholding the sensor is the right call and it is invisible. The reader collapses the last group, scrolling quietly stops fetching, and the only cue anywhere is the absence of new rows — which is exactly what a feed that has genuinely run out looks like. Nothing in the tree says which of the two it is, and until this method nothing outside Render could work it out either: the answer is composed from Items, GroupBy and Collapse, three fields the caller holds separately and none of which means anything alone.

The footer is where that matters, because the footer is the way out. A screen whose Footer is a bare LoadMore is already fine — the button is visible and calls the same function — and the shape that is not fine is the common one:

	Footer: hasMore ? LoadMore{...} : nil     // "the feed is over"

With the last group shut, hasMore is true, so that one is fine too. The trap is the other direction — a footer hidden while auto-load is believed to be doing the work:

	list := components.GroupedList[Sermon]{
	    Items: pager.Items, GroupBy: byMonth, Collapse: shut,
	    OnEndReached: pager.LoadMore,
	}
	// Shown when there is more to fetch *and* nothing is fetching it.
	if pager.HasMore && list.AutoLoadWithheld() {
	    list.Footer = components.LoadMore{HasMore: true, OnLoadMore: pager.LoadMore, ...}
	}

A method rather than a second field, because it is derived: a field would be a copy of a fact the widget already computes, free to disagree with it the moment either the items or the collapse state moved. The value has to be built before it can be asked, which is why the example assigns Footer after the literal — the same ordering any derived-from-itself decision takes.

It answers false when OnEndReached is nil. There is no sensor to withhold on a manual pager, and a caller asking this question is asking whether the automatic path is off \*right now\*, not whether the last group happens to be shut. Collapse.hides is the field to ask for that.

<small>[components/grouped_list.go:282](https://github.com/rohanthewiz/grmob/blob/master/components/grouped_list.go#L282)</small>

#### func (GroupedList) Render

```go
func (g GroupedList[T]) Render(ctx *core.Context) *core.Node
```

<small>[components/grouped_list.go:286](https://github.com/rohanthewiz/grmob/blob/master/components/grouped_list.go#L286)</small>

### type InputRow

```go
type InputRow struct {
	// Value is the field's text. The field is fully controlled: it displays
	// exactly this, and OnChange is the only way it changes.
	Value string

	// Placeholder is the empty-state text inside the field. It is the field's
	// only label — InputRow has no caption slot; wrap it in a FormField when
	// one is needed.
	Placeholder string

	// OnChange fires on every keystroke and is what keeps Value in step with
	// what the user typed. Without it the field is read-only in practice: the
	// next render paints Value back over the keystrokes.
	OnChange func(string)

	// OnSubmit is the commit action: the keyboard's return key (iOS) or IME
	// done action (Android), and the trailing button's tap.
	//
	// When it is nil the field is built without a submit path at all rather
	// than with one wired to a no-op, so neither platform advertises a submit
	// affordance the row would ignore.
	OnSubmit func()

	// Button is the trailing commit button. Its zero value renders nothing;
	// a Button with a Label renders with OnTap defaulted to OnSubmit. Every
	// other Button field — Variant, Emphasis, Disabled, Style, the
	// accessibility pair — works as it does anywhere else.
	Button Button

	// Gap is the horizontal spacing between the field and the button, in
	// points. Zero means the theme's SM step, not zero spacing; see above.
	Gap float64

	// Style is applied to the row after Gap, so a caller can override it —
	// or add the background and padding that make the row read as a docked
	// bar, which the widget itself has no opinion about.
	Style []core.StyleProp
}
```

InputRow is the composer: a single-line text field that fills the row, and an optional trailing button that commits it.

	Row (Gap)
	  ├─ Input   ← FlexGrow(1), value / placeholder / onChange / onSubmit
	  └─ Button  ← only when Button.Label is set; OnTap defaults to OnSubmit

Two call sites spelled this out by hand — chat's message composer and todoapp's entry row — and they were near-identical down to the Gap(8) and the FlexGrow(1). What they also shared was the wiring, which is the part that is easy to get subtly wrong: one commit action reached by three paths (the keyboard's return/done key, the trailing button, and — through onChange — the Go state that both of them read).

#### The three paths, and why OnSubmit drives two of them

A composer is not "a field and a button that happen to sit together": the button \*is\* the field's submit, rendered as a tap target for the case where the keyboard's return key is not obvious or not reachable. Both hand-written sites therefore named the same helper twice, once as core.InputWithSubmit's onSubmit and once as the button's OnTap. Here OnSubmit is the commit action and the button inherits it, so the two cannot drift apart:

	components.InputRow{
	    Value:       draft.Get(),
	    Placeholder: "What needs doing?",
	    OnChange:    func(v string) { draft.Set(v) },
	    OnSubmit:    addTodo,
	    Button:      components.Button{Label: "Add"},
	}

Setting Button.OnTap explicitly still wins — a "Send anyway" that skips a validation the keyboard path performs is a real shape — but it has to be said out loud.

#### Gap defaults to the theme's step, unlike Screen's

Screen's Gap treats zero as "don't set one", because the spacing between a screen's sections is the app's decision and a theme's Column base may already carry one. The gap here is the opposite kind of thing: it is the widget's own internal layout — the field and the button must not touch — so InputRow owns it the way FormField owns the spacing between its label and its input. Zero therefore means "the theme's SM step" (8pt in both bundled themes, which is exactly what both hand-written sites had picked).

A caller that genuinely wants no gap says so through Style, which is applied after: Style: \[]core.StyleProp{core.Gap(0)}.

#### The trailing button is optional

A zero Button renders nothing at all — no node, not an empty one — so a search field or a filter box that commits on the return key alone is the same widget with one less field set. Absence is keyed on Label because a button with no visible label is not a button; a glyph button spells its glyph ("✕", "→") in Label and its meaning in AccessibilityLabel.

#### The input is owned, not slotted

Unlike FormField, which takes whatever input it is given, InputRow builds its own — the wiring above \*is\* the widget, and a slot would hand it back to the caller. The consequence is that the field itself takes no per-call styling; a composer that needs to restyle its input has outgrown this and should go back to core.Row + core.InputWithSubmit.

<small>[components/input_row.go:68](https://github.com/rohanthewiz/grmob/blob/master/components/input_row.go#L68)</small>

#### func (InputRow) Render

```go
func (r InputRow) Render(ctx *core.Context) *core.Node
```

<small>[components/input_row.go:107](https://github.com/rohanthewiz/grmob/blob/master/components/input_row.go#L107)</small>

### type ListRow

```go
type ListRow struct {
	// Leading whose width is text is worth core.FlexShrink(0), and this is
	// the one trap in the slot.
	//
	// The centre column claims the row's slack, so when the row's content
	// overflows — a long title on a phone — the deficit is shared out among
	// the children that can shrink, and a bare core.Text is the most
	// compressible thing in the row. Every host floors that share at the
	// text's min-content width, so a word is never ground down to a glyph:
	// the web and Compose always did, and iOS does since GrMobMinContent,
	// which a simulator run of examples/tutorial forced after it rendered the
	// lesson numbers as 4 / . / 1 / 2.
	//
	// What the floor does NOT promise is that the text stays on one line. A
	// row number has no break opportunity in it, so its min-content is the
	// whole of it; a two-word label's is its longer word, and a row tight
	// enough will wrap it. FlexShrink(0) is the declaration that says the
	// slot's width is not negotiable at all, and a leading slot whose width
	// is meant to be read at a glance wants it on every host.
	//
	// A fixed-size control — a Checkbox, an icon with a Width — is unaffected,
	// which is why this is a note on the field rather than a wrapper around it:
	// ListRow cannot add a style prop to a View a caller handed it, and
	// wrapping every slot in a pinned Box would be two extra nodes per row of
	// every list to fix the case where the caller passes text.
	//
	// Leading is the control at the start of the row: a checkbox, an icon,
	// an avatar. Nil renders nothing and costs no node.
	Leading core.View

	// Title is the row's primary line, Subtitle the quieter second line.
	Title    string
	Subtitle string
	// Content is the escape hatch for the middle: an arbitrary view in the
	// growing slot, taking precedence over Title/Subtitle when set. Same
	// simple-path-plus-slot idiom as Card.Title vs Card.Header.
	Content core.View

	// Trailing is the action or value pinned to the end of the row: a badge,
	// an amount, a delete button, a chevron.
	Trailing core.View

	// OnTap and OnLongPress make the whole row a target. They are wired only
	// when non-nil, so a purely presentational row carries no callback and
	// no gesture recognizer on any platform. Both may be set at once: the
	// renderers wire them as a single recognizer, so a long press never also
	// fires the tap.
	OnTap       func()
	OnLongPress func()

	// Selected drives the row's selected look, and how the state is announced
	// — as a real core.AccessibilitySelected when Selectable is set, and
	// otherwise as a ", selected" suffix on AccessibilityLabel. See "How the
	// state is announced" in the type comment for why there are two answers.
	Selected bool

	// Selectable says this row is one choice in a listbox: it takes
	// core.RoleOption and states core.AccessibilitySelected for *both* values
	// of Selected, so a reader announces "selected" and "not selected" rather
	// than announcing the chosen row and passing silently over the rest. That
	// is core.SelectedOff doing the job it exists for, one widget over from
	// the tab strip whose argument it was written for.
	//
	// # The container is the caller's to role, and must be
	//
	// An `option` is owned by a `listbox` (see "A structural role owns what is
	// inside it" in core/role.go). This widget renders one row and cannot see
	// what it was put in, so set core.RoleListBox on the container yourself,
	// or leave this field false. An orphan `option` is the "table with no
	// rows" failure one row down.
	//
	//	core.List(
	//	    core.AccessibilityRole(core.RoleListBox),
	//	    ListRow{Title: "Weekly",  Selectable: true, Selected: plan == weekly,
	//	            AccessibilityLabel: "Weekly", OnTap: choose(weekly)},
	//	    ListRow{Title: "Monthly", Selectable: true, Selected: plan == monthly,
	//	            AccessibilityLabel: "Monthly", OnTap: choose(monthly)},
	//	)
	//
	// The same foreign-child rule applies as ever: a listbox holding a
	// "Load more" footer or a section heading is not a listbox.
	//
	// # It wins over NestingLevel, and the depth is lost
	//
	// The two fields ask for different roles and a node has one. `option`
	// takes aria-selected and not aria-level; `listitem` takes aria-level and
	// not aria-selected. So a row setting both describes a container that is a
	// list and a listbox at once, which does not exist, and this field is the
	// half that wins: the state is what the row is being tapped to change,
	// and a depth inside a container that has not claimed to be a list is
	// decoration.
	//
	// ARIA does have a role carrying both — `treeitem` inside a `tree` — and
	// core.Role deliberately does not, because a tree is a third pattern with
	// its own expansion state and keyboard contract and nothing here has one.
	// See the RoleListBox block in core/role.go.
	//
	// # What each target does with it
	//
	// The two web targets write role="option" and aria-selected. Neither
	// native names a listbox or an option, but both announce the *state* on
	// any node — a SwiftUI .isSelected trait, a Compose `selected` property —
	// so the row still reads as chosen on device and it is only the
	// container's word that is missing. Setting this therefore adds on every
	// target and costs nothing on any.
	//
	// # What it buys on the keyboard, and what the caller has to do for it
	//
	// A listbox in ARIA's full pattern takes focus, moves an active option
	// with the arrow keys and reports which one through a roving tabindex.
	// The WASM runtime supplies all of it — one tab stop per list, Up/Down
	// within it, Home and End, and Enter or Space running this row's OnTap —
	// but only for rows inside a container that says it is a listbox. That is
	// the same core.RoleListBox the field above already asks the caller for,
	// so the keyboard arrives with the container's role and not with this
	// flag: a Selectable row in an unroled Box is an `option` with nothing to
	// be an option of, and gets no more keyboard than it did before.
	//
	// A static htmlout export writes no tab stops at all, deliberately — see
	// core.RoleListBox. On both phones none of this was ever missing:
	// VoiceOver and TalkBack navigate a collection by swipe.
	Selectable bool

	// NestingLevel is how deep this row sits in a nested collection — 1 for a
	// top-level item, 2 for one inside it, and on down with no ceiling. It
	// makes the row a `listitem` at that depth; zero leaves it the unroled Box
	// it has always been. Ignored when Selectable is set, which takes the
	// row's one role for `option` — see that field.
	//
	// # What it is for
	//
	// A tree flattened into one list. That is the case ARIA defines aria-level
	// on `listitem` for: the rows are siblings in the markup because a list is
	// a flat run of children — which is also what core.List's virtualization
	// requires — so the depth a reader needs has nowhere else to live. Without
	// it an outline is announced as a flat run of items and every indent is
	// pixels only.
	//
	//	core.List(
	//	    core.AccessibilityRole(core.RoleList),
	//	    ListRow{Title: "Gospels",  NestingLevel: 1},
	//	    ListRow{Title: "Matthew",  NestingLevel: 2, Style: indent(1)},
	//	    ListRow{Title: "Sermon on the Mount", NestingLevel: 3, Style: indent(2)},
	//	)
	//
	// # The container is the caller's to role, and must be
	//
	// A `listitem` is owned by a `list` (see "A structural role owns what is
	// inside it" in core/role.go). This widget renders one row and cannot see
	// what it was put in, so it cannot supply the other half — set RoleList on
	// the container yourself, or leave this field at zero. An orphan
	// `listitem` is the "table with no rows" failure one row down.
	//
	// The corollary is worth stating because it costs something: a row inside
	// a role="list" must not also be a role="button", so a tappable row in an
	// outline announces as an item at a depth and not as a control. That is
	// the same foreign-child rule that kept the selected state off this widget,
	// read from the other side — and it is not a regression, because a ListRow
	// has never carried RoleButton.
	//
	// # What each target does with it
	//
	// The web writes aria-level. Neither native has a nesting-depth property
	// at all, so the role goes out and the depth does not, which is the honest
	// gap nine of core's twenty roles already have.
	NestingLevel int

	// Style is applied to the row container after ListRow's own defaults
	// (which sit on top of the theme's Row base), so every default here —
	// gap, vertical centering, the theme's row padding — is overridable.
	Style []core.StyleProp
	// SelectedStyle is applied on top when Selected, after Style, so
	// selection wins over the base look. When nil, a theme default is used:
	// a Surface background tint.
	SelectedStyle []core.StyleProp

	// AccessibilityLabel names the whole row for screen readers.
	// AccessibilityHint describes what tapping does.
	//
	// A Selected row with no Selectable gets ", selected" appended, because
	// the name is then the only place the state can be said; a Selectable row
	// leaves the name alone and states the selection properly. See "How the
	// state is announced" in the type comment.
	//
	// No label is synthesized from Title: a row is a compound control whose
	// slots (a badge's amount, a trailing control's own name) carry meaning
	// the widget cannot see, and labelling the container overrides how those
	// children are announced. Naming the row is therefore the caller's call,
	// exactly as it is for Chip.
	AccessibilityLabel string
	AccessibilityHint  string
}
```

ListRow is the leading-control / flexible-title / trailing-action shape that every list in the examples hand-rolls: a checkbox and a task with a delete button, an avatar and a name with a chevron, a label and an amount.

#### Why the widget exists

The shape was written five times across the examples and the instances did not agree on how the trailing slot gets pinned to the trailing edge. Some used Justify(JustifyBetween) on the row; others used FlexGrow(1) on the middle Text. The two are not equivalent:

	JustifyBetween  distributes slack *between every pair* of children, so a
	                row with no trailing slot pushes leading and title apart.
	FlexGrow(1)     gives all the slack to one child, so leading and trailing
	                stay hard against the row's edges in every configuration.

ListRow settles it on FlexGrow: the middle column is the row's spine and it always grows. That is also why the middle column is rendered even when Title, Subtitle and Content are all empty — unlike Card, which omits empty regions. Here the middle is structure, not content: making it conditional would make the pinning conditional too, which is precisely the inconsistency this widget exists to remove.

	┌ Row ─────────────────────────────────────────────────────────┐
	│ [Leading]  ┌ Column FlexGrow(1) ────────────┐    [Trailing]  │
	│            │ Title                          │                │
	│            │ Subtitle                       │                │
	│            └────────────────────────────────┘                │
	└──────────────────────────────────────────────────────────────┘
	            └──────── takes all the slack ────┘

#### Selection

Selection is controlled by the caller, as with Chip: ListRow holds no state, it renders Selected and reports taps. Selected rows take the theme's Surface as a background tint — the palette's only muted \*fill\*, and still the right one now that Border exists, since Border is a stroke role; there is no dedicated Selected entry. How the state reaches a screen reader is the next section's subject and depends on Selectable.

#### How the state is announced, and why it took a role to do it

The state is scoped by role on both web targets, because ARIA scopes the attributes it becomes: a state on an unroled element is dropped by screen readers exactly as an accessible name on one is. The \*name\* half of that has since been closed for every widget at once — the two web exporters supply core.RoleGroup to a named container that has no role, so an unroled row's AccessibilityLabel is now announced on all four targets rather than two. The state half could not be closed the same way, because there is no role that carries a selection and fits any container (see core.Style.AccessibilitySelected). A ListRow is a Box, so announcing a selection properly means giving the row a role, and for three versions of this widget every candidate was wrong:

	RoleButton    true only for a tappable row, and a role="button" child
	              makes the row a *foreign child* of any role="list" it sits
	              in — the structural rule in core/role.go, which says such a
	              container may then take no list role at all. One widget's
	              announcement would cost the enclosing list its shape.
	RoleListItem  the honest description of a row, and ARIA defines neither
	              state attribute for it. `aria-selected` is scoped to
	              gridcell, option, row, tab, columnheader and rowheader, and
	              a list item is none of them.

So the row wrote ", selected" into its own accessible name instead — the true thing said in the weaker of the two places, announced once, inside a string that is meant to be stable.

core.RoleOption and core.RoleListBox are the door that was left, and Selectable is how a caller walks through it. A row that takes the option role carries a real core.AccessibilitySelected, every renderer announces it as a state rather than as part of a name, and the suffix does not appear.

A row that is \*not\* Selectable is unchanged: it still appends the suffix when it has both a label and a selection, because it still has no role that could carry the state, and saying the true thing weakly beats not saying it. The suffix does at least reach a reader now — the group role the exporters supply is what makes the name it rides on audible on the web at all, which for the three sessions before that role existed it was not.

#### Two roles, one row, and the caller picks

NestingLevel and Selectable both give the row a role and the two roles are exclusive — see Selectable for the precedence, which is a fact about the roles rather than about this widget's willingness.

Both are opt-in, and that is the ownership rule rather than caution: a \`listitem\` with no \`list\` around it, or an \`option\` with no \`listbox\`, is a role naming a structure that is not there, which core/role.go calls worse than no role at all. A row cannot see its container, so it cannot make that true by itself — the caller roles the enclosing collection and marks each row to match, and a row that is asked for neither is exactly the unroled Box it has always been.

<small>[components/list_row.go:97](https://github.com/rohanthewiz/grmob/blob/master/components/list_row.go#L97)</small>

#### func (ListRow) Render

```go
func (r ListRow) Render(ctx *core.Context) *core.Node
```

<small>[components/list_row.go:290](https://github.com/rohanthewiz/grmob/blob/master/components/list_row.go#L290)</small>

### type LoadMore

```go
type LoadMore struct {
	HasMore bool
	Loading bool
	Err     error

	// OnLoadMore fetches the next page. OnRetry re-runs a failed fetch; when
	// nil, Retry falls back to OnLoadMore, which is the right answer for a
	// pager whose load call is idempotent about the offset.
	OnLoadMore func()
	OnRetry    func()

	// Label, LoadingLabel and RetryLabel override the copy. ErrorText
	// replaces err.Error() — for an app that maps transport errors to
	// something a person should read.
	Label        string
	LoadingLabel string
	RetryLabel   string
	ErrorText    string

	// Style is applied to the tail's container after its defaults.
	Style []core.StyleProp
}
```

LoadMore is the append-style tail of an offset-paged list: the "Load more" button, the "Loading…" note while a page is in flight, and the retry strip when a page failed. It exists because every paged screen in an app hand-rolls exactly this state machine at the bottom of its list.

	HasMore  Loading  Err   renders
	false    false    nil   nothing (the list is complete)
	true     false    nil   [Load more]
	*        true     *     Loading…
	*        false    set   message  [Retry]

Err wins over HasMore because a failed fetch says nothing about whether more rows exist; Loading wins over Err because a retry in flight has superseded the failure it is retrying.

<small>[components/paging.go:108](https://github.com/rohanthewiz/grmob/blob/master/components/paging.go#L108)</small>

#### func (LoadMore) Render

```go
func (l LoadMore) Render(ctx *core.Context) *core.Node
```

<small>[components/paging.go:131](https://github.com/rohanthewiz/grmob/blob/master/components/paging.go#L131)</small>

### type MapHandoff

```go
type MapHandoff func(lat, lng float64, label string) string
```

MapHandoff turns a point and its name into a URL for core.OpenURL. See StaticMap.Handoff.

<small>[components/static_map.go:331](https://github.com/rohanthewiz/grmob/blob/master/components/static_map.go#L331)</small>

### type MapPanel

```go
type MapPanel struct {
	// Pins are the points to show. An empty set renders Empty rather than a
	// map, because the alternative is a map of 0,0 — see FitRegion, which
	// refuses to answer for a set with nothing in it.
	Pins []MapPin

	// OnPinTap is called with the id of the pin the user tapped. Pins with an
	// empty id are drawn and never reported, which is core.Marker's own rule.
	OnPinTap func(id string)

	// OnMapTap is called with the coordinates of a tap that did not hit a pin.
	// The suppression is each host's; see core.OnMapTap.
	OnMapTap func(lat, lng float64)

	// Height sizes the map. "" means DefaultMapPanelHeight. Width is always
	// 100%: a map narrower than its column is a map with a gutter beside it,
	// and a caller who wants one wraps this in a box.
	Height string

	// Caption is a line under the map — a count, a legend, a note about what
	// is not on it. Empty draws nothing, including no padding.
	Caption string

	// Empty is what to render for a set with no pins in it. The zero value is
	// a generic EmptyState; a caller who knows what the reader was looking for
	// should say so, because "nothing to show" is the least useful sentence a
	// screen can end on.
	Empty EmptyState

	// Zoom overrides the fitted zoom level. 0 means "fit the pins", which is
	// what this widget is for; a value is for a caller who wants a fixed scale
	// and only the centring.
	Zoom float64

	// Style is applied last, to the outer column, so a caller can give the
	// panel a margin or a border without reaching inside it.
	Style []core.StyleProp
}
```

MapPanel is "where are these": a live map over a set of points, opened at a view that contains all of them.

	components.MapPanel{
	    Pins: []components.MapPin{
	        {ID: "hall", Lat: 38.7223, Lng: -9.1393, Title: "The hall"},
	        {ID: "annex", Lat: 38.7251, Lng: -9.1402, Title: "The annex"},
	    },
	    OnPinTap: func(id string) { open(id) },
	    Height:   "320px",
	}

#### What this adds to core.MapView, which is the whole of its case

One thing, and it is the thing every consumer of a set of points needs and would otherwise write itself: the \*opening region\*. core.MapView takes a Region and applies it only when it changes, which is correct and leaves the caller holding a question — what region shows all of my points? — whose answer is a bounding box, a projection correction and a logarithm. See FitRegion, which is that arithmetic and is exported for a caller who wants it without this widget.

Everything else here is arrangement a caller could write in ten lines and which is worth having in one place anyway: the pins as keyed children, the empty state for a set with nothing in it, and an optional caption.

#### What it deliberately does not add

No region state, no recentre control, no echo of OnRegionChange. The region is computed once from the pins and handed over, and where the reader takes the map from there is the reader's. An app that wants to follow the map reaches for core.MapView directly and holds the Region itself — which is the arrangement core.MapView's "The map is controlled, with the echo guard a drag needs" describes, and it is a different widget's job.

The pins changing DOES move the map: a new set is a new fitted region, which core.MapView applies because it changed. That is the right behaviour for the case this widget is for — the set is the subject — and it is the reason a caller whose pins update every few seconds wants core.MapView instead.

#### Tiles are somebody else's bandwidth

Inherited from core.MapView, and unchanged by this widget: two of the three hosts draw OpenStreetMap tiles from the project's own servers, under a usage policy. See that node's doc. Unlike StaticMap this needs no key from anyone, because each host's own engine draws the map — which is the odd asymmetry that the richer widget is the free one.

<small>[components/map_panel.go:58](https://github.com/rohanthewiz/grmob/blob/master/components/map_panel.go#L58)</small>

#### func (MapPanel) Render

```go
func (m MapPanel) Render(ctx *core.Context) *core.Node
```

<small>[components/map_panel.go:313](https://github.com/rohanthewiz/grmob/blob/master/components/map_panel.go#L313)</small>

### type MapPin

```go
type MapPin struct {
	// ID is what OnPinTap reports, and what the reconciler keys the marker by.
	// Empty is allowed — the single unnamed "you are here" pin — and is never
	// reported. See core.Marker.
	ID string
	// Lat and Lng are the point, in degrees.
	Lat, Lng float64
	// Title is the callout the platform shows when the pin is tapped. It is
	// not an accessibility label; a map's annotations are announced by each
	// platform's own map accessibility.
	Title string
}
```

MapPin is one point in a MapPanel: the data a core.Marker needs, as a value a caller can hold in a slice and sort.

A struct rather than four arguments for the reason StaticMapArea is one: a caller builds these in a loop from their own data, and a field added here is a field existing code ignores.

<small>[components/map_panel.go:103](https://github.com/rohanthewiz/grmob/blob/master/components/map_panel.go#L103)</small>

### type Pagination

```go
type Pagination struct {
	// Page is the 0-based current page.
	Page int
	// PageSize is rows per page; only read by a collection doing its own
	// slicing (see the type comment).
	PageSize int
	// PageCount is the total number of pages, 0 when unknown.
	PageCount int
	OnChange  func(page int)

	// PrevLabel and NextLabel override the button text. Empty takes the
	// defaults below.
	PrevLabel string
	NextLabel string

	// Style is applied to the footer row after its defaults.
	Style []core.StyleProp
}
```

Pagination is the numbered-page footer: "‹ Prev  Page 2 of 7  Next ›".

It serves two paging models with one struct, told apart by PageCount:

  - PageCount > 0: the caller owns the pages. Whatever rows it hands the collection are the current page, and OnChange asks it to fetch another. This is the server-side shape.
  - PageCount == 0 and PageSize > 0: the collection owns the pages. It slices the full row set itself and derives PageCount from len(rows). This is the client-side shape; DataTable implements it, GroupedList does not (a grouped feed pages by "Load more", not by number).
  - PageCount == 0 and PageSize == 0: open-ended. The label shows only the current page and Next is never disabled, for a caller that learns there is no next page only by asking.

Selection is controlled, as everywhere in this package: Page is read from the caller's state and OnChange writes it back.

<small>[components/paging.go:22](https://github.com/rohanthewiz/grmob/blob/master/components/paging.go#L22)</small>

#### func (Pagination) Render

```go
func (p Pagination) Render(ctx *core.Context) *core.Node
```

<small>[components/paging.go:41](https://github.com/rohanthewiz/grmob/blob/master/components/paging.go#L41)</small>

### type ProgressBar

```go
type ProgressBar struct {
	// Value is the completed fraction, 0 to 1. Values outside that range are
	// clamped rather than rejected: a bar fed a ratio from live counters
	// should pin at full and keep rendering, not draw outside its track.
	// NaN is treated as 0.
	Value float64

	// Thickness is the track height in px; 0 means 6.
	Thickness float64

	// Color is the fill; empty uses the theme's Primary. TrackColor is the
	// groove behind it; empty uses the theme's Surface.
	Color      string
	TrackColor string

	// Style is applied to the track after the defaults, so the bar's width,
	// margins and corner are all overridable.
	Style []core.StyleProp

	// AccessibilityLabel names what is progressing ("Upload"), and nothing
	// else — the value is announced separately, through the role and the
	// range below. When empty the bar is hidden from assistive tech: an
	// unlabeled bar announces a bare number with nothing to attach it to, and
	// a bar beside its own "Uploading, 45%" caption should stay silent.
	//
	// It used to carry the percentage as well ("Upload, 45 percent"), because
	// no renderer had a progress semantic to put the number in. Three of the
	// four do now, and a name was the wrong channel for it in the way
	// components.Chip's old ", selected" suffix was: a name is meant to be
	// stable, so a bar ticking from 44 to 45 re-announced the whole string
	// rather than the part that changed, and nothing could act on a number
	// buried in it.
	AccessibilityLabel string

	// ValueText is the spoken form of the value, for a caller who wants
	// particular words ("almost done", "3 of 5 uploaded"). Empty is the
	// normal case and means "let each platform say the number in its own
	// words", which is the better answer on three of the four targets:
	//
	//	web ×2    aria-valuenow over an implicit 0..100, which a browser
	//	          announces as a localized percentage
	//	Compose   ProgressBarRangeInfo, which TalkBack localizes the same way
	//	SwiftUI   nothing. There is no numeric accessibility value on this
	//	          platform — accessibilityValue takes a string — so an
	//	          unaccompanied bar announces its name alone.
	//
	// That last row is why the field exists rather than being left out: iOS is
	// the one target where the value genuinely has nowhere to go, and an app
	// that would rather have English than silence there can say so. It costs
	// the localization on the other three, which is why it is not the default
	// — ARIA and Compose both announce this text *instead of* the number.
	//
	// The renderers deliberately do not supply it themselves. A framework
	// emitting "45 percent" would be inventing English for every app in every
	// locale, which is the same move GrMobStyle.swift turns down for
	// AccessibilityExpanded; these are the caller's own words.
	ValueText string
}
```

ProgressBar is the determinate track-and-fill bar: an upload, a download, a quota, a multi-step form's position.

	components.ProgressBar{Value: 0.45, AccessibilityLabel: "Upload"}

#### Why the fill is a percentage width and not a pair of flex weights

The obvious construction is two boxes weighted FlexGrow(v) and FlexGrow(1-v), letting the flex algorithm split the track. That was exact on Android, where FlexGrow maps onto Compose's Modifier.weight — and silently wrong on iOS, where it mapped onto frame(maxWidth: .infinity). SwiftUI stacks have no weight, so two growers split the free space \*equally regardless of their values\*: every bar would have rendered at 50% on iOS, at every value, with nothing in the tree to suggest a bug.

A percentage width is proportional on all three targets instead:

	Android   fillMaxWidth(fraction)  — exact
	HTML      width:<pct>%            — exact
	iOS       containerRelativeFrame  — proportional, see the caveat below

The iOS caveat is that containerRelativeFrame measures against the nearest \*container\* (the scroll view or root), not the immediate parent. A bar that spans its container — the common case, a full-width bar in a screen column — is therefore exact; one inset inside a narrow card reads wider than it should. That is an over-long fill in an uncommon layout, against a permanently-half-full bar everywhere.

That custom SwiftUI Layout has since landed: GrMobFlexStack, with the arithmetic in GrMobFlexSolver, resolves FlexGrow by value on all three targets. This widget has not been migrated, so the caveat above is still what it does today — but the blocker is gone, and moving to flex would remove the caveat, since a flex child is measured against its immediate parent rather than the nearest container.

#### The fill is always rendered

Even at Value 0, where it is zero pixels wide. Keeping the child count fixed means advancing progress is a style patch on one node rather than an insert or a remove, so the reconciler emits an update-style op per frame instead of restructuring the tree — which is also what lets a Transition on the fill animate the bar smoothly.

<small>[components/progress_bar.go:52](https://github.com/rohanthewiz/grmob/blob/master/components/progress_bar.go#L52)</small>

#### func (ProgressBar) Render

```go
func (p ProgressBar) Render(ctx *core.Context) *core.Node
```

<small>[components/progress_bar.go:111](https://github.com/rohanthewiz/grmob/blob/master/components/progress_bar.go#L111)</small>

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

<small>[components/chip.go:39](https://github.com/rohanthewiz/grmob/blob/master/components/chip.go#L39)</small>

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

### type RichTextEditor

```go
type RichTextEditor struct {
	// Doc is the document. The editor is controlled: it renders what it is
	// given and reports edits through OnChange.
	Doc richtext.Doc

	// OnChange receives every edit as a whole document. A nil OnChange makes
	// the editor read-only in practice; set ReadOnly instead when that is what
	// you mean, which also tells the platform.
	OnChange func(richtext.Doc)

	// Placeholder is the prompt an empty editor shows.
	Placeholder string

	// ReadOnly shows the document with a caret and a selection and refuses
	// edits. Deliberately not Disabled: a note being displayed is content the
	// reader is meant to select and copy.
	ReadOnly bool

	// Toolbar, when set, adds the command row above the editor. Build it with
	// components.UseRichToolbar(ctx) in the calling component.
	Toolbar *RichToolbar

	// MinHeight gives an empty editor something to be. Without it a document
	// with one line in it is one line tall, which reads as a text field rather
	// than as a place to write.
	MinHeight string

	// Height fixes the editor's height instead, so a long document scrolls
	// inside it rather than growing the screen.
	Height string

	// Style is applied to the editor after the widget's own frame.
	Style []core.StyleProp
}
```

RichTextEditor is a formatted-text editor: bold, italics, headings, lists, quotes, links — with a toolbar whose buttons show what is active under the caret.

	bar := components.UseRichToolbar(ctx)

	components.RichTextEditor{
	    Doc:         note.Get(),
	    OnChange:    note.Set,
	    Placeholder: "Write something…",
	    Toolbar:     bar,
	    MinHeight:   "160px",
	}

	┌ Column ──────────────────────────────────────────────────────┐
	│ ┌ Row, wrapping (the toolbar) ─────────────────────────────┐ │
	│ │ [B] [I] [U] [S] [</>] [H1] [H2] [•] [1.] [""] [🔗]       │ │
	│ └──────────────────────────────────────────────────────────┘ │
	│ ┌ core.RichTextEditor ─────────────────────────────────────┐ │
	│ │  The document, edited in place by the platform's own     │ │
	│ │  text engine.                                            │ │
	│ └──────────────────────────────────────────────────────────┘ │
	└──────────────────────────────────────────────────────────────┘

#### A read-only editor with no toolbar is the display half

A comment, a note, a description — anything that shows formatted text the reader did not write. There is no second "RichTextView" node, because there does not need to be one: the renderer's own text engine draws the document either way, and \`ReadOnly\` is the difference between reading it and writing it. That is also why this widget takes no hook of its own; see below.

#### The toolbar is the caller's, and that is what keeps the display half free

A toolbar needs three things that must survive a render pass: a core.EditorRef to send commands to, the last reported selection (so the bold button can look pressed), and whether the link prompt is open. All three are hooks, and a widget that called them would make \*every\* RichTextEditor a hook-slot consumer — something that must be rendered unconditionally on every pass, like Accordion and DatePicker.

That is a fine obligation for an editor with a toolbar and a bad one for a note being \*displayed\*, which is the thing rendered inside an \`if\`, inside a loop, inside a list of comments. So UseRichToolbar is the hook, the caller makes it, and an editor with no toolbar touches nothing. components.CodeEditor makes the same split for the same reason.

<small>[components/rich_text_editor.go:54](https://github.com/rohanthewiz/grmob/blob/master/components/rich_text_editor.go#L54)</small>

#### func (RichTextEditor) Render

```go
func (r RichTextEditor) Render(ctx *core.Context) *core.Node
```

<small>[components/rich_text_editor.go:180](https://github.com/rohanthewiz/grmob/blob/master/components/rich_text_editor.go#L180)</small>

### type RichToolItem

```go
type RichToolItem struct {
	Label   string
	Command string
	// AccessibilityLabel names the button for a screen reader. A toolbar is
	// glyphs — "B", "H1", "🔗" — and a glyph is not a name.
	AccessibilityLabel string
}
```

RichToolItem is one button on the toolbar: what it says, and what it sends.

Command is a core Edit\* constant or one of the two builders (core.EditBlock, core.EditLink) — except for RichToolLink, which is this package's own sentinel for "open the link prompt", because a URL has to be typed before there is a command to send.

<small>[components/rich_text_editor.go:95](https://github.com/rohanthewiz/grmob/blob/master/components/rich_text_editor.go#L95)</small>

### type RichToolbar

```go
type RichToolbar struct {
	// Items is what the toolbar offers, in order. Defaults to a copy of
	// RichToolbarDefault; assign to it to offer something else.
	Items []RichToolItem
	// contains filtered or unexported fields
}
```

RichToolbar is everything a toolbar needs that has to survive a render pass.

Built by UseRichToolbar, which is a hook: the ref must be the same pointer every pass or the buttons would command an editor nobody is listening to, and the selection has to be remembered between the report arriving and the next render drawing the buttons from it.

<small>[components/rich_text_editor.go:135](https://github.com/rohanthewiz/grmob/blob/master/components/rich_text_editor.go#L135)</small>

#### func UseRichToolbar

```go
func UseRichToolbar(ctx *core.Context) *RichToolbar
```

UseRichToolbar builds the state a RichTextEditor's toolbar needs.

Four hook slots, in a fixed order, so — like any hook user — it must be called unconditionally on every pass of the component that owns it.

	func NoteScreen(ctx *core.Context) core.View {
	    note := core.NewState(ctx, richtext.Doc{})
	    bar  := components.UseRichToolbar(ctx)
	    return components.RichTextEditor{Doc: note.Get(), OnChange: note.Set, Toolbar: bar}
	}

<small>[components/rich_text_editor.go:156](https://github.com/rohanthewiz/grmob/blob/master/components/rich_text_editor.go#L156)</small>

#### func (*RichToolbar) Selection

```go
func (b *RichToolbar) Selection() core.RichSelection
```

Selection is the last selection the editor reported, which is what the toolbar draws its pressed state from — and is worth reading directly for a status line, or to decide whether a "Link" action makes sense.

<small>[components/rich_text_editor.go:178](https://github.com/rohanthewiz/grmob/blob/master/components/rich_text_editor.go#L178)</small>

### type Screen

```go
type Screen struct {
	// Children are the screen's content, laid out top to bottom in the
	// column. A nil entry is skipped (see above).
	Children []core.View

	// Scroll wraps the column in core.Scroll, making the whole screen
	// scrollable. Leave it false when the screen has its own scrolling region
	// inside it — a core.List, or a Scroll around one section — since a
	// scroll view nested in a scroll view fights for the same drag on both
	// natives.
	//
	// Following that advice with a List costs nothing in layout: the scaffold
	// drops its own inset when the List is the whole content, so the page sits
	// where the scrolled Column drew it. See "A scrolling child is the page".
	Scroll bool

	// KeyboardAware makes that scroll region shrink to sit above the software
	// keyboard, so a focused field near the bottom of a form is scrolled
	// somewhere visible rather than under the keys. See core.KeyboardAware
	// for what each platform does with it and what it deliberately does not
	// cover.
	//
	// Without Scroll it lifts the content column whole instead, which is the
	// behavior a screen with something docked at its bottom wants — chat's
	// composer, a checkout bar — since that bar sits outside any scrolling
	// region and is the one thing the keyboard covers.
	KeyboardAware bool

	// Gap is the uniform vertical spacing between children, in points. Zero
	// means "don't set one", not "zero spacing" — the theme's Column base
	// keeps whatever it had. Use Gap for uniform runs and core.Spacer between
	// specific children when the spacing differs (that rule predates this
	// widget; see examples/fintechapp).
	Gap float64

	// Fill makes the column claim the full height of the safe area
	// (FlexGrow(1)) rather than shrinking to its content. Set it when a child
	// needs to expand into the leftover space — a list that should fill the
	// screen and push a footer to the bottom — because a FlexGrow child can
	// only grow inside a parent that has height to give.
	//
	// Fill with Scroll is legal but unusual: it makes the scrolled content at
	// least as tall as the viewport, which is how you bottom-anchor a footer
	// on a short page. It does not make a scroll view fill anything.
	Fill bool

	// Style is applied to the column, after Gap and Fill, so a caller can
	// override either — or add padding and a background the scaffold itself
	// has no opinion about.
	Style []core.StyleProp
}
```

Screen is the root scaffold every app in this repo was hand-spelling: the safe-area inset, an optional scroll region, and the vertical column that holds the screen's content.

	SafeArea
	  └─ Scroll            (only when Scroll is true; KeyboardAware rides here)
	       └─ Column       ← Gap / Fill / Style land here
	            ├─ Children[0]
	            └─ …

Five call sites spelled some subset of that by hand, and each picked its own subset — one wrapped in Scroll, two set a Gap, one set FlexGrow(1), and no two agreed on the order the props were written in. The shape is not hard to type; the value of naming it is that there is now one place to hang the things a screen root will eventually need (a pull-to-refresh region, per-platform inset behavior) instead of five — KeyboardAware below is the first of them to arrive.

#### The zero value is exactly SafeArea(Column(children...))

Every field defaults to contributing nothing, so the zero value renders the bare scaffold and no style props at all — the theme's Column base carries through untouched. That is deliberate and it is what let all five migrations below stay byte-identical: a field only speaks when the caller sets it. It applies specifically to Gap, which is applied only when non-zero. An explicit core.Gap(0) would \*overwrite\* a gap the theme's Column base had set (style props mutate the style directly rather than merging into it), so "unset" and "zero" have to mean the same thing here — the absence of a gap, not the imposition of one.

#### Nil children are skipped

Children flows into core.Column's argument list, which skips nil items (the same contract that makes core.MaybeProp work). So the conditional-region idiom this codebase already uses for slots —

	var banner core.View
	if offline {
	    banner = OfflineBanner()
	}
	return components.Screen{Children: []core.View{banner, body}}

— costs the tree no node at all when the condition is false, rather than the empty Fragment a core.If would leave behind for the reconciler to walk on every pass. (That Fragment draws nothing; the cost is the node, not a gap.)

#### A scrolling child is the page, and is inset once

Screen's column carries the theme's Components.Column base, whose only entry in every bundled theme is the standard 12/16 inset. So does core.List — it is the one other container in the tree built on that same base. A screen whose whole content is a List therefore used to be inset twice, and the doubling is invisible in code because neither inset is written anywhere:

	SafeArea
	  └─ Column   padding 12/16   ← the theme's, via Screen
	       └─ List padding 12/16   ← the theme's, again

Every child of that list drew 16 points further in than the same content in a Scroll'd Column, which is the shape Screen.Scroll's own documentation recommends migrating \*away\* from. So the scaffold now drops its column's padding when its content is a single scrolling page:

	Screen{Children: []core.View{core.List(rows...)}}   inset once, by the List

Three things about the rule are deliberate.

"Only child" is counted after nil entries are skipped, so the conditional-slot idiom above keeps working: a screen holding a nil banner and a List is a single-child screen, exactly as the tree the reconciler walks is. That is why the decision is made on the \*rendered\* child rather than on the Go value — the count that matters is the one core.Column would arrive at, and a wrapper widget (components.GroupedList) is a List only after it renders.

The set is node types that scroll and arrive pre-inset, which today is core.List alone. core.Scroll is deliberately outside it: a Scroll carries no theme base, so its content is inset once — by this column — and dropping that would move the page rather than unstack it.

Style still wins. The cleared padding is applied ahead of the caller's Style props, so a screen that asks for core.Padding(24) around its list gets 24, and one that wants the old doubled behavior can still spell it. Nothing else about the column changes: a Gap, a Fill and a background all survive, because it is only the inset that was ever duplicated.

<small>[components/screen.go:90](https://github.com/rohanthewiz/grmob/blob/master/components/screen.go#L90)</small>

#### func (Screen) Render

```go
func (s Screen) Render(ctx *core.Context) *core.Node
```

<small>[components/screen.go:192](https://github.com/rohanthewiz/grmob/blob/master/components/screen.go#L192)</small>

### type SearchField

```go
type SearchField struct {
	// Value is the current text. The field is controlled: it renders what it
	// is given and reports edits through OnChange.
	Value string

	// Placeholder is the empty-field prompt. Empty is "Search".
	Placeholder string

	// OnChange receives every edit, including the clear. A nil OnChange makes
	// the field read-only in practice — it will render Value and drop
	// keystrokes — so it is worth setting even on a field you expect not to
	// change.
	OnChange func(string)

	// OnSubmit fires on the keyboard's return / IME done action. When nil the
	// field carries no submit callback at all, which keeps a
	// search-as-you-type box off the InputWithSubmit path entirely.
	OnSubmit func()

	// OnClear replaces what the clear button does. The default is
	// OnChange(""), which is what a clear means for a controlled field; set
	// this when clearing also has to dismiss results, cancel a pending
	// request, or restore a previous view.
	OnClear func()

	// Glyph is the leading mark. Empty is 🔍; NoGlyph drops it, for a field
	// whose surroundings already say what it searches.
	Glyph   string
	NoGlyph bool

	// AccessibilityLabel names the field for screen readers. Empty falls back
	// to the placeholder, which is the resolved one — so the default field
	// announces as "Search" rather than as nothing.
	//
	// The fallback exists because a placeholder is not a label on any
	// platform: it is a value hint that vanishes on the first keystroke, so a
	// field relying on it alone is unnamed for exactly the users who most
	// need the name.
	AccessibilityLabel string

	// Style is applied to the row after the widget's own frame, so the fill,
	// the radius and the padding are all overridable.
	Style []core.StyleProp
}
```

SearchField is a text field dressed as a search box: a leading magnifier, a flexible input, and a clear button that appears once there is something to clear.

	components.SearchField{
	    Value:    query.Get(),
	    OnChange: query.Set,
	    OnSubmit: run,
	}

	┌ Row (Surface, rounded) ──────────────────────────────────────┐
	│ 🔍   ┌ Input FlexGrow(1) ─────────────┐   [✕]                │
	│      └────────────────────────────────┘                      │
	└──────────────────────────────────────────────────────────────┘

#### It holds no state and calls no hook

Value is the caller's, exactly as with Chip's Selected and DataTable's Sort, and every keystroke arrives through OnChange for the caller to store. That is worth stating because the obvious alternative — a field that owns its own text — would make this the second widget in the package with hook obligations, and a search box is a thing screens render conditionally (in a header that appears when a "Search" action is tapped), which is precisely what a hook-slot consumer must not be.

#### Debouncing is deliberately not in here

A controlled field cannot debounce its own OnChange: the value has to reach state on the keystroke or the characters do not appear. What wants delaying is the \*reaction\* — the query, the filter, the fetch — and that lives in the caller. hooks.UseDebounce is the piece for it:

	d := hooks.UseDebounce(ctx, 250*time.Millisecond)
	components.SearchField{
	    Value: query.Get(),
	    OnChange: func(s string) {
	        query.Set(s)                       // now, so typing looks like typing
	        d.Call(func() { search(s) })       // in 250ms, if the typing stopped
	    },
	    OnSubmit: func() { d.Cancel(); search(query.Get()) },
	}

Splitting it that way is also what makes the two paths honest: Enter should search immediately, and Cancel is how the pending call gets out of the way.

#### The frame

The row paints the theme's Surface at the theme's own field radius and the input inside it is flattened — transparent, no radius, no padding and no border of its own. Without that the theme's Input base (its own background, corners and rule) would draw a second box inside the first, which is what a hand-rolled search row looks like before someone notices.

The border half of that is newer than the rest and the reason is worth keeping: the theme's Input entry used to carry no rule at all, so on the web a search field was quietly wearing the \*browser's\* — which the row's own fill mostly hid, and which no core.BorderWidth(0) could have removed anyway (see borderResetTypes in htmlout/tag.go). Now that both bundled themes state a field frame, the second box would be drawn deliberately and on all four targets, so the flattening has to say so.

<small>[components/search_field.go:65](https://github.com/rohanthewiz/grmob/blob/master/components/search_field.go#L65)</small>

#### func (SearchField) Render

```go
func (s SearchField) Render(ctx *core.Context) *core.Node
```

<small>[components/search_field.go:110](https://github.com/rohanthewiz/grmob/blob/master/components/search_field.go#L110)</small>

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

	components.SegmentedControl{
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

	components.SegmentedControl{
	    Labels:   []string{"Sermons", "Articles"},
	    Selected: tab.Get(),
	    OnSelect: func(i int) { tab.Set(i) },
	    Style:    []core.StyleProp{core.AccessibilityRole(core.RoleTabList)},
	    Segment:  components.Chip{Style: []core.StyleProp{core.AccessibilityRole(core.RoleTab)}},
	}

The state each Chip already sets then goes out as aria-selected instead of aria-pressed, because the two web exporters pick the attribute from the role — nothing in this widget or in Chip has to know which arrangement it is in. See core.Style.AccessibilitySelected.

Two things that arrangement does not buy, both of which are ARIA's rules rather than this widget's limits. A tablist claims its children are tabs, so a row that also holds a count or an add button is not one (see core/role.go). And the panel the tabs control cannot be pointed at from here: aria-controls is an IDREF, and core.Style carries values rather than references — a real wired tab strip is core.TabView, which owns both ends of that relationship.

<small>[components/segmented_control.go:88](https://github.com/rohanthewiz/grmob/blob/master/components/segmented_control.go#L88)</small>

#### func (SegmentedControl) Render

```go
func (s SegmentedControl) Render(ctx *core.Context) *core.Node
```

<small>[components/segmented_control.go:138](https://github.com/rohanthewiz/grmob/blob/master/components/segmented_control.go#L138)</small>

### type Separator

```go
type Separator struct {
	// Color overrides the hairline tint. Empty takes the theme's Border role.
	Color string

	// Thickness is the rule's height in px; 0 means 1. Fractional values are
	// carried through to the platforms as-is (a "0.5px" hairline is a real
	// request on a 2x/3x display, and both renderers parse a float).
	Thickness float64

	// Inset indents the rule from both ends, in px. This is the list idiom
	// where the rule starts under the text rather than under the leading
	// avatar or checkbox, so the leading column reads as one continuous
	// stripe.
	//
	// It is applied as core.MarginHorizontal, which writes the two explicit
	// sides as well as the axis shorthand. The field has now been spelled
	// three ways for one reason each: a Left/Right pair, because the two web
	// targets once read the per-side fields only and dropped the shorthand; a
	// bare EdgeInsets.Horizontal, once every renderer resolved the shorthand
	// into the unset sides; and the prop, because a whole EdgeInsets through
	// UseStyle replaces Margin outright and so clears any vertical gap the
	// caller's Style below asked for. The prop touches one axis and leaves
	// the other alone, which is what this field always meant.
	Inset int

	// Style is applied last, so every default above is overridable.
	Style []core.StyleProp
}
```

Separator is the hairline rule between list rows and between sections.

core.Divider already draws a line, but it takes a literal color and force-applies Margin(8) — an unconditional gap that makes it wrong inside a list, which is presumably why neither example that wanted a rule used it. Separator leaves spacing to the caller and takes its tint from the theme's Border role, so the common case is the zero value:

	components.Separator{}

The tint reads through ColorPalette.BorderColor rather than off the Border field, so a theme written before that role existed draws a visible default hairline instead of an invisible one. Surface would have been the nearest pre-existing neutral and is the wrong answer: it is a \*fill\* color, so on a Surface-colored panel a Surface hairline disappears.

It is always hidden from assistive technology. A rule is decoration: it carries no information a screen reader can use, and announcing one between every pair of rows in a list turns a 20-row feed into 39 utterances.

#### Horizontal only

There is no Vertical field yet. A vertical rule has to stretch to its row's height, which means cross-axis stretch, and that used to be the blocker: neither renderer mapped AlignItems "stretch", so the field would have advertised something that collapsed to zero height on both platforms.

Both renderers map it today — Compose pins a stretched Row to IntrinsicSize.Max and gives each child fillMaxHeight(), and SwiftUI's GrMobFlexStack proposes the full cross extent to a stretched child — so the field is now a widget change rather than a renderer one, waiting on a caller that wants it. One asymmetry to know when it lands: a Row reads alignItems alone and never the simpler Align fallback (Align is a text-alignment concept and has never applied to a row's vertical axis), so the containing row has to spell out AlignItems "stretch".

<small>[components/separator.go:44](https://github.com/rohanthewiz/grmob/blob/master/components/separator.go#L44)</small>

#### func (Separator) Render

```go
func (s Separator) Render(ctx *core.Context) *core.Node
```

<small>[components/separator.go:73](https://github.com/rohanthewiz/grmob/blob/master/components/separator.go#L73)</small>

### type Skeleton

```go
type Skeleton struct {
	// Lines is how many bars to stack. Zero means one.
	Lines int

	// Height is a bar's height in points. Zero takes the theme's body font
	// size, so a text placeholder is about as tall as the text it stands in
	// for and scales with the theme.
	Height float64

	// Width is every bar's width, as a CSS-ish length ("100%", "180px").
	// Empty is "100%".
	Width string

	// LastLineWidth shortens the final bar, which is what makes a stack read
	// as a paragraph rather than as a table. Empty is "60%". It applies only
	// when Lines is 2 or more: on a single bar the last line is the only
	// line, and silently rendering it at 60% would make the simplest call
	// surprising.
	LastLineWidth string

	// Gap is the space between bars. Zero takes the theme's SM step.
	Gap float64

	// Radius is the bar's corner radius. Zero is 4 — enough to read as a
	// placeholder rather than as a rule. Set it to half the height (or just
	// to 999, which clamps) for a pill, or with a square Width/Height for the
	// circle an avatar placeholder wants.
	Radius float64

	// Color overrides the bar fill. Empty takes the theme's Border role; see
	// the type comment for why that and not Surface.
	Color string

	// AccessibilityLabel names the whole block for assistive technology.
	// Empty is "Loading".
	//
	// The individual bars are always hidden — a reader walking six unlabeled
	// boxes is worse than silence — and the label goes on the container
	// instead, which takes core.RoleStatus.
	//
	// # Why `status` and not the `group` it used to get
	//
	// Both web exporters supply RoleGroup to a named container that says
	// nothing about what it is, which made the name legal and stopped there:
	// `group` says "these things belong together and this is what they are
	// called", so a reader announced "Loading" only if the user happened to
	// walk onto the block. A skeleton is not a group of things — the bars
	// stand in for content that is not here yet — and what it is is ARIA's
	// definition of `status`: one advisory that is *replaced*. The wait
	// announces itself when it starts, and the content replaces it when it
	// arrives, which is exactly the region's contract.
	//
	// It is a live region, so it is announced without the reader looking at
	// it, which is what the old note here said a screen could not rely on and
	// had to put in text instead. That caveat is closed on the two web targets
	// and on Android (Compose's polite live region); SwiftUI has no live
	// region property, so on iOS this is still a labelled container, which
	// VoiceOver announces on arrival but does not interrupt for.
	//
	// For no announcement at all, pass core.AccessibilityHidden() in Style.
	AccessibilityLabel string

	// Style is applied to the container after the widget's own defaults.
	// Per-bar styling is not exposed: a skeleton whose bars differ is a
	// composition of Skeletons, not one Skeleton with more knobs.
	Style []core.StyleProp
}
```

Skeleton is the grey placeholder that holds a screen's shape while its content loads: one bar, or a stack of them standing in for a paragraph.

	components.Skeleton{}                            // one line
	components.Skeleton{Lines: 3}                    // a paragraph, last line short
	components.Skeleton{Width: "44px", Height: 44, Radius: 999}   // an avatar

#### Skeleton or EmptyState

They answer different questions. A skeleton says "content is coming and it will look roughly like this", which is worth saying when the layout is known and stable — a feed of rows, a profile header. EmptyState says "there is nothing here yet, and here is why", which is what a screen with no predictable shape, or a wait long enough to need explaining, wants instead. A list of three placeholder rows reads better than "Loading…"; a whole screen of grey bars reads worse.

#### No shimmer, and why that is not a shortcut

The moving highlight every design system puts on a skeleton is a repeating keyframe animation. core.Transition is not one — it animates a property from one declared value to another, driven natively, and there is no state change here to drive. The alternative is looping it from Go with hooks.UseInterval, which would push a render pass and a patch across the bridge for every frame of a decoration, on every placeholder on screen. That is the one thing the framework's "declare in Go, animate natively" model exists to avoid, so the bars are static until a repeating animation is a core primitive.

#### The color is the Border role, not Surface

Surface is the palette's obvious "muted fill", and it is the wrong answer for the same reason Separator gives: it is the fill a \*panel\* uses, so a Surface bar inside a card disappears. Border is the neutral that is visible against both Background and Surface, which is where placeholders sit. It is nominally a stroke role and this is a fill — the palette carries no third neutral, and being visible beats being nominally correct.

<small>[components/skeleton.go:46](https://github.com/rohanthewiz/grmob/blob/master/components/skeleton.go#L46)</small>

#### func (Skeleton) Render

```go
func (s Skeleton) Render(ctx *core.Context) *core.Node
```

<small>[components/skeleton.go:114](https://github.com/rohanthewiz/grmob/blob/master/components/skeleton.go#L114)</small>

### type Sort

```go
type Sort struct {
	Column int
	Desc   bool
}
```

Sort names the active sort column and direction. DataTable.Sort is a pointer so that "no sort" is nil rather than an ambiguous column 0.

<small>[components/data_table.go:49](https://github.com/rohanthewiz/grmob/blob/master/components/data_table.go#L49)</small>

### type StatTile

```go
type StatTile struct {
	// Label is what the figure measures. Value is the figure itself,
	// pre-formatted — the widget does no number formatting, because currency,
	// grouping and locale are the caller's, not a layout widget's.
	Label string
	Value string

	// Delta is the movement line under the value: "+18 vs last week", "−3%",
	// "unchanged". Empty renders nothing.
	Delta string

	// DeltaVariant colors the delta with a semantic role. The zero value is
	// the theme's secondary ink — neutral, saying nothing about whether the
	// movement is good. See the type comment for why this one field departs
	// from the package's usual reading of VariantDefault.
	//
	// The same contrast caveat applies here as to an outlined or ghost
	// Button, and for the same reason: the role color is laid on a backdrop
	// the widget cannot see, so the theme's own numbers are what you get.
	// Against each theme's Background, Success is 2.22:1 under DefaultTheme
	// and Warning 2.20:1 — well under the 4.5:1 body-text floor, i.e. a delta
	// that is decoration rather than text. Under MaterialTheme the same two
	// are 5.13:1 and 3.08:1. So on the default palette a colored delta needs
	// to be reinforcement for wording that already says it ("+18, best week
	// this term"), never the only place the direction appears; a caller who
	// needs the color itself to be legible wants a Badge, which owns its fill
	// and picks its ink by contrast.
	DeltaVariant Variant

	// Fill makes the tile take an equal share of a row rather than hugging
	// its content, which is what a row of two or three tiles wants.
	//
	// It sets FlexGrow *and* a zero FlexBasis, which looks like belt and
	// braces and is actually what makes the four targets agree. Compose's
	// Modifier.weight and SwiftUI's equivalent divide the whole axis by
	// weight, so equal grows are already equal widths there; CSS flex-grow
	// divides only the *leftover* space, so on the web a tile with a longer
	// value would come out wider. Zeroing the basis is what makes CSS
	// distribute the whole axis too. The natives ignore FlexBasis, so the one
	// prop that is inert on two targets is precisely the one that converges
	// the other two.
	Fill bool

	// OnTap makes the whole tile a target — a KPI that opens the report
	// behind it. Wired only when non-nil, so a presentational tile registers
	// no callback.
	OnTap func()

	// AccessibilityLabel names the tile as one thing, which is usually what
	// you want: a reader moving through three tiles should hear "Attendance,
	// 412, up 18 from last week", not six fragments. Nothing is synthesized
	// when it is empty — the three lines are then announced separately, which
	// is correct but wordier.
	AccessibilityLabel string
	AccessibilityHint  string

	// Style is applied after the widget's own defaults.
	Style []core.StyleProp
}
```

StatTile is one figure with its name and, optionally, its movement — the unit a dashboard, a profile header, or an account summary is made of.

	core.Row(core.Gap(12),
	    components.StatTile{Label: "Attendance", Value: "412", Fill: true,
	        Delta: "+18 vs last week", DeltaVariant: components.VariantSuccess},
	    components.StatTile{Label: "Giving", Value: "MZN 42,750", Fill: true},
	)

#### It has no frame, deliberately

"Tile" names the content, not a card. The widget renders a text stack and paints nothing: no background, no border, no padding of its own. That is what lets the two common arrangements both be composition rather than configuration — three tiles inside one core.Card, or three tiles each in their own — instead of a Framed bool that is wrong half the time. The fintech example's balance block was already exactly this shape inside a card; it is the card that was doing the framing, and it still is.

#### Order: label, then figure

The label sits above the value. In a row of tiles that keeps the labels on one line at the top and the figures on another below them, which survives labels of different lengths; the other order ("412" over "Attendance") makes the figures ragged the moment one label wraps to two lines. For a centered arrangement, pass core.AlignItemsProp(core.AlignItemsCenter) in Style — each line then shrinks to its content and centers.

#### The delta's zero value is neutral, not Primary

Everywhere else in this package VariantDefault resolves to the theme's Primary — that is what Badge and Button do, and what makes their zero value a no-op. Here it resolves to the secondary text ink instead, and the reason is that a delta is a measurement rather than a status. Whether a number going up is good is entirely the caller's domain: attendance up is success, expenses up is not, and latency up is an incident. A widget that guessed — by coloring on the sign, or by defaulting to the brand color as if any movement were noteworthy — would be confidently wrong on half the tiles a real screen carries. So the default says nothing, and a caller who knows what the movement means says it with DeltaVariant.

<small>[components/stat_tile.go:45](https://github.com/rohanthewiz/grmob/blob/master/components/stat_tile.go#L45)</small>

#### func (StatTile) Render

```go
func (s StatTile) Render(ctx *core.Context) *core.Node
```

<small>[components/stat_tile.go:105](https://github.com/rohanthewiz/grmob/blob/master/components/stat_tile.go#L105)</small>

### type StaticMap

```go
type StaticMap struct {
	// Lat and Lng are the point to show, in degrees.
	//
	// Latitude is clamped to the Web Mercator limit (±85.0511°) rather than to
	// ±90: every tile provider here projects with Mercator, where the poles
	// are at infinity, and a request past the limit comes back as an error
	// image rather than as a view of the Arctic. Longitude wraps instead of
	// clamping, because longitude is a circle — 190° is 170°W, and clamping it
	// to 180 would move the point rather than name it.
	Lat, Lng float64

	// Zoom is the slippy-tile zoom level both providers share: 0 is the whole
	// world, 19 is a building. 0 means DefaultMapZoom (street level), which is
	// the scale "where is this" is asking at.
	//
	// Zero cannot mean "the whole world" here, and that is the one place this
	// widget spends a legitimate value on a default. A world map centred on a
	// church is a picture of the Atlantic; a caller who genuinely wants zoom 0
	// has a provider of their own to ask.
	Zoom int

	// Width and Height are the image's size in px. 0 means
	// DefaultMapWidth/DefaultMapHeight, a card-width landscape panel.
	//
	// Both are clamped to MaxMapDimension, which is the smaller of what the
	// two bundled providers will serve. A request past it is refused by the
	// server, and a refusal arrives as a broken image with no error anywhere —
	// so the clamp is what turns "nothing rendered and nobody knows why" into
	// "a slightly smaller map than you asked for".
	Width, Height int

	// Scale is the device pixel ratio to ask the provider for: 1, 2 or 3.
	// 0 means DefaultMapScale, which is 1x.
	//
	// It changes the image, never the box: a Scale of 2 leaves the widget
	// exactly Width by Height logical pixels on screen and asks for four
	// times as many actual pixels to fill them. Clamped to MaxMapScale and
	// then spent, or ignored, by the provider — see "Device pixel ratio" on
	// the type for both halves of that.
	Scale int

	// Marker draws a pin at the point. Off by default: a map whose centre *is*
	// the subject does not always need one, and a single pin in the middle of
	// a small image can hide the very corner a reader is looking at.
	Marker bool

	// Label is what the place is called — "Lisbon Baptist Church", "The
	// rehearsal hall". It is the spoken name of the image, and it is handed to
	// the Handoff func, which may or may not have anywhere to put it: neither
	// built-in hand-off does (see GoogleMapsHandoff), and a platform-specific
	// one a caller writes can.
	//
	// Empty falls back to the coordinates, which is honest and reads badly —
	// "Map at 38.7223, -9.1393". Worth setting for that reason alone.
	Label string

	// Provider builds the image URL. Required: nil renders no image at all
	// and reports ConcernNoMapProvider in debug mode. See "The image is a
	// network fetch, and the provider is required" on the type for why this
	// stopped having a default.
	Provider StaticMapProvider

	// Handoff builds the URL a tap opens. nil means GoogleMapsHandoff.
	//
	// Setting it to a func that returns "" is how a caller says *no hand-off*:
	// core.OpenURL drops an empty URL, and this widget reads the same answer
	// one step earlier and renders a picture with no link role and no
	// callback, rather than a control that does nothing when tapped.
	Handoff MapHandoff

	// OnTap replaces the hand-off entirely — a caller who wants to push their
	// own map screen, open a sheet of directions, or log the tap first.
	//
	// It takes precedence over Handoff, and it does not compose with it: a
	// widget that both called back and opened an external app would leave the
	// caller no way to express either one alone. An OnTap that wants the
	// hand-off too can ask for it: core.OpenURL(components.GoogleMapsHandoff(
	// lat, lng, label)).
	OnTap func()

	// Style is applied last, to the outer box, so a caller can give the map a
	// margin, a radius or a border without reaching inside it.
	Style []core.StyleProp

	// AccessibilityLabel overrides the spoken name. AccessibilityHint
	// overrides the "opens in your maps app" description a tappable map
	// carries; it is ignored when there is nothing to tap.
	AccessibilityLabel string
	AccessibilityHint  string
}
```

StaticMap is "where is this": a map image of one point, which hands off to the platform's own maps app when it is tapped.

	components.StaticMap{
	    Lat: 38.7223, Lng: -9.1393,
	    Label:  "Lisbon Baptist Church",
	    Marker: true,
	}

	┌─────────────────────────┐
	│                         │   a core.Image of a rendered map,
	│           📍            │   fetched from a tile provider
	│                         │
	└─────────────────────────┘
	        tap ──▶ core.OpenURL ──▶ Google Maps / Apple Maps / a browser

#### Why an image and a hand-off rather than a map view

Because this answers the question an app actually asks. "Where is the church", "where is this event" wants a picture that says \*there\*, and then the user wants directions — which is a thing the platform's maps app does better than any embedded view could, with the user's own saved home address, their transport preferences and their downloaded tiles.

It is also the version that exists today on all four targets with no renderer work at all: a core.Image the reconciler already knows how to patch and a core.OpenURL the three hosts already hand to the system. A live panning, zooming map is a node type (MapKit, osmdroid, Leaflet — three host implementations and a marker protocol), which is a much larger feature and is the thing to build when an app needs to \*interact\* with a map rather than point at one.

#### The image is a network fetch, and the provider is required

Nothing here draws a map. The widget builds a URL and hands it to core.Image, so what arrives is whatever the provider serves — which makes the provider a decision an app has to own:

	GoogleStaticMap(key)  a paid API with an availability promise, and the
	                      only bundled provider whose host resolves.
	a func of your own    StaticMapProvider is one function; a provider this
	                      package has never heard of is five lines.
	OSMStaticMap          deprecated, and dead. See that function.

There is no default, and there used to be. OSMStaticMap was it, on the argument that a widget nobody can render without first buying something is a widget nobody evaluates — which was the right argument for as long as there was a keyless service to point at. There is not one now: every static-map service the OpenStreetMap wiki still lists takes a key, and the two keyless entries on that page are a web form and an HTML-embed generator, neither of which is a URL an image node can fetch.

So a StaticMap with no Provider renders its frame and no image, exactly as GoogleStaticMap("") does, and records ConcernNoMapProvider in debug mode. A build that has not chosen should look unfinished and say so, rather than draw a map of a host it cannot reach.

#### Why this cannot just draw the tiles itself

The obvious escape is to stop asking for a rendered image and compose one out of the raster tiles this repository already uses — tile.openstreetmap.org is keyless, alive, and is what core.MapView draws through on all three hosts. It is not available here. A tile grid centred on an arbitrary point needs its tiles placed at pixel offsets inside a clipped box, which is absolute positioning, and absolute positioning is the one Style.Position value with no Compose analog (see GrMobStyle.kt, which says so). A widget that laid out correctly on web and iOS and drifted on Android is worse than one that asks for a key.

#### Device pixel ratio

Width and Height are \*logical\* pixels: they set the size of the box on the screen. Scale is how many device pixels the provider should put inside each of them. Two numbers because they answer two questions — how big is this, and how sharp is it — and until Scale existed one number answered both: a 320px request stretched across a 2x phone's 640 real pixels, which arrives visibly soft and could not be said otherwise from the call site.

	Width: 320, Scale: 2    a 320-logical-px box holding a 640px image

Nothing reads the ratio off the device, because nothing in this framework knows it: there is no host event that reports screen metrics, on any of the three renderers. A caller who has the number states it; a caller who says nothing is choosing 1x, which is what every caller got before the field existed.

Whether the number can be spent is the provider's business. Google's API has a scale parameter that doubles the raster without touching the cartography; a provider with no such parameter ignores the field, which is why this is a StaticMapArea value rather than a multiply applied to Width before the provider sees it. The multiply would be wrong twice over: asking a map service for twice the pixels at the same zoom returns twice as much \*map\*, and asking at zoom+1 returns the same ground drawn for a deeper zoom, whose labels then land at half the physical size they were drawn for.

#### The hand-off is one URL for three platforms

Each platform has a scheme of its own — \`geo:\` on Android, \`maps://\` on iOS — and nothing in this framework knows which platform it is running on (deliberately: see core.OpenURL, which promises only the part that is portable). So the default Handoff is an https URL that all three resolve, and that on both phones reaches the installed maps app rather than a browser tab.

An app that does know its platform can say so in one line — \`Handoff: func(lat, lng float64, label string) string { return "geo:..." }\` — and an app that would rather not leave for Google has OpenStreetMapHandoff, which opens a browser everywhere.

#### Accessibility

A map image read by a screen reader is a rectangle with nothing in it: the meaning is in the arrangement, which is exactly the case core.RoleImg exists for (see its doc, and components.Compass, which made the same argument about a compass rose). So the widget announces once and hides the image inside it.

When it is tappable the role is core.RoleLink rather than RoleButton, and the distinction is the one core.RoleLink's doc draws: a button does something here and a link goes somewhere else. Tapping this leaves the app entirely, which is as far as "somewhere else" goes, and a reader deciding whether to follow it deserves to know that before they do.

#### Zero is a place

Lat 0, Lng 0 is the Gulf of Guinea, and this widget draws it. There is no "unset" coordinate to detect — a float64 pair has no third state — so a caller whose location has not loaded yet must not render the widget at all, exactly as they would not render an EmptyState's action with no handler. components.Skeleton is the placeholder for that gap.

<small>[components/static_map.go:142](https://github.com/rohanthewiz/grmob/blob/master/components/static_map.go#L142)</small>

#### func (StaticMap) Area

```go
func (m StaticMap) Area() StaticMapArea
```

Area resolves the caller's fields into the view a provider is handed: the defaults applied, the size clamped, the latitude clamped and the longitude wrapped. Everything downstream reads this rather than the struct, so there is one statement of what a zero means.

Exported because the answer is worth asking for from outside. A caller who wants to know what URL this widget will request — to log it, to pre-warm a cache, to show it in a tutorial — can ask the provider about this rather than re-deriving the defaults, which is the one way to get a second answer that disagrees.

<small>[components/static_map.go:478](https://github.com/rohanthewiz/grmob/blob/master/components/static_map.go#L478)</small>

#### func (StaticMap) Render

```go
func (m StaticMap) Render(ctx *core.Context) *core.Node
```

<small>[components/static_map.go:583](https://github.com/rohanthewiz/grmob/blob/master/components/static_map.go#L583)</small>

### type StaticMapArea

```go
type StaticMapArea struct {
	Lat, Lng      float64
	Zoom          int
	Width, Height int

	// Scale is the device pixel ratio asked for, already defaulted and
	// clamped. Width and Height stay logical: a provider that honours Scale
	// returns Width*Scale actual pixels for the same map, and a provider that
	// cannot honour it returns Width by Height and is not wrong to. See
	// "Device pixel ratio" on StaticMap.
	Scale int

	Marker bool
}
```

StaticMapArea is the view a provider is asked to render: the point, the scale, the pixel size, and whether to pin it.

A struct rather than five arguments because a provider is a function a caller writes, and a signature that grows is a signature that breaks every one of them. A field added here is a field an existing provider ignores.

The values arrive already defaulted and already clamped — a provider never sees a zero Zoom, a 4000px Width or a latitude off the end of Mercator — so every provider is spared the same four lines and none of them can disagree about what a zero means.

<small>[components/static_map.go:298](https://github.com/rohanthewiz/grmob/blob/master/components/static_map.go#L298)</small>

### type StaticMapProvider

```go
type StaticMapProvider func(StaticMapArea) string
```

StaticMapProvider turns a view into an image URL. See StaticMap.Provider.

<small>[components/static_map.go:321](https://github.com/rohanthewiz/grmob/blob/master/components/static_map.go#L321)</small>

#### func GoogleStaticMap

```go
func GoogleStaticMap(key string) StaticMapProvider
```

GoogleStaticMap renders through the Google Static Maps API with the given key, which the caller has obtained and is billed for.

A constructor rather than a bare provider because the key is the caller's: it is configuration, it differs per build, and a provider that read it out of a package variable would be a second place for a deployment to be wrong.

An empty key yields a provider that returns "", which renders the widget as a box with no image in it rather than as a map of Google's "this request is not authorized" error tile. A misconfigured build should look unfinished, not broken.

<small>[components/static_map.go:380](https://github.com/rohanthewiz/grmob/blob/master/components/static_map.go#L380)</small>

### type Tabs

```go
type Tabs struct {
	Items []core.TabItem // build with core.Tab(label, icon)
	// Selected is the controlled selection index; pair it with OnChange
	// writing to the state it is read from.
	Selected int
	OnChange func(int)
	// Content holds the tab pages. By the core.TabView contract all pages
	// are children of the node; the native side shows the selected one.
	Content []core.View
}
```

Tabs is the named-field facade over core.TabView.

Wrap, not supersede (the open question in the element-lessons plan): core.TabView defines a wire contract — the "TabView" node type with its tabs/selectedIndex/onTabChange props that the native renderers consume — and node-type contracts belong in core, next to the registry of types the renderers know. What TabView lacks is only ergonomics: four positional option props with no record of which is which at a call site. This struct supplies the field names and delegates everything else, so there is exactly one tab implementation to keep in sync with the renderers.

<small>[components/tabs.go:15](https://github.com/rohanthewiz/grmob/blob/master/components/tabs.go#L15)</small>

#### func (Tabs) Render

```go
func (t Tabs) Render(ctx *core.Context) *core.Node
```

<small>[components/tabs.go:26](https://github.com/rohanthewiz/grmob/blob/master/components/tabs.go#L26)</small>

### type Variant

```go
type Variant string
```

Variant selects a widget's semantic color role — what a piece of UI \*means\* rather than what it looks like. It is shared across the package rather than owned by Badge so a future Alert, Banner or status Chip resolves the same four roles the same way, and so a caller can pass one value around.

It is a string enum with an empty zero value, matching core's Alignment and DisplayMode. That is load-bearing here: the zero value must be the existing look, or adding the field would restyle every Badge already in a tree.

<small>[components/variant.go:18](https://github.com/rohanthewiz/grmob/blob/master/components/variant.go#L18)</small>

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

<small>[components/variant.go:35](https://github.com/rohanthewiz/grmob/blob/master/components/variant.go#L35)</small>

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

<small>[components/variant.go:115](https://github.com/rohanthewiz/grmob/blob/master/components/variant.go#L115)</small>

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

<small>[components/variant.go:70](https://github.com/rohanthewiz/grmob/blob/master/components/variant.go#L70)</small>

